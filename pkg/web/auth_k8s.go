package web

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"encoding/json"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// RuntimeEnvironment indicates where k13d is running
type RuntimeEnvironment string

const (
	// RuntimeInCluster - running inside Kubernetes cluster (needs token auth)
	RuntimeInCluster RuntimeEnvironment = "in-cluster"
	// RuntimeLocal - running locally with kubeconfig (can use kubeconfig auth)
	RuntimeLocal RuntimeEnvironment = "local"
)

// K8sTokenValidator validates Kubernetes service account tokens
type K8sTokenValidator struct {
	clientset   *kubernetes.Clientset
	restConfig  *rest.Config
	environment RuntimeEnvironment
	// For local mode: cached user info from kubeconfig
	kubeconfigUser string
}

// NewK8sTokenValidator creates a new K8s token validator
func NewK8sTokenValidator() (*K8sTokenValidator, error) {
	var config *rest.Config
	var environment RuntimeEnvironment
	var kubeconfigUser string

	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		environment = RuntimeInCluster
	} else {
		// Fallback to kubeconfig for local development
		environment = RuntimeLocal
		config, kubeconfigUser, err = loadKubeconfigWithUser()
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &K8sTokenValidator{
		clientset:      clientset,
		restConfig:     config,
		environment:    environment,
		kubeconfigUser: kubeconfigUser,
	}, nil
}

// loadKubeconfigWithUser loads kubeconfig and extracts current user/context info
func loadKubeconfigWithUser() (*rest.Config, string, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	// Get the raw config to extract user info
	rawConfig, err := kubeConfig.RawConfig()
	if err != nil {
		return nil, "", err
	}

	// Get current context and user
	currentContext := rawConfig.CurrentContext
	var username string
	if ctx, ok := rawConfig.Contexts[currentContext]; ok {
		username = ctx.AuthInfo
		// If username is a path or complex, simplify it
		if username == "" {
			username = currentContext
		}
	} else {
		username = currentContext
	}

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, "", err
	}

	return config, username, nil
}

// GetEnvironment returns the runtime environment
func (v *K8sTokenValidator) GetEnvironment() RuntimeEnvironment {
	return v.environment
}

// GetKubeconfigUser returns the current kubeconfig user (for local mode)
func (v *K8sTokenValidator) GetKubeconfigUser() string {
	return v.kubeconfigUser
}

// ValidateToken validates a Kubernetes token and returns user info
func (v *K8sTokenValidator) ValidateToken(ctx context.Context, token string) (*TokenReview, error) {
	review := &authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{
			Token: token,
		},
	}

	result, err := v.clientset.AuthenticationV1().TokenReviews().Create(ctx, review, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("token review failed: %w", err)
	}

	if !result.Status.Authenticated {
		return nil, fmt.Errorf("token not authenticated")
	}

	return &TokenReview{
		Authenticated: result.Status.Authenticated,
		Username:      result.Status.User.Username,
		UID:           result.Status.User.UID,
		Groups:        result.Status.User.Groups,
	}, nil
}

// TokenReview represents the result of a token validation
type TokenReview struct {
	Authenticated bool     `json:"authenticated"`
	Username      string   `json:"username"`
	UID           string   `json:"uid"`
	Groups        []string `json:"groups"`
}

// ValidateK8sToken validates a Kubernetes service account token
func (am *AuthManager) ValidateK8sToken(ctx context.Context, token string) (*Session, error) {
	// Check cache first
	am.mu.RLock()
	if session, exists := am.tokenSessions[token]; exists {
		if time.Now().Before(session.ExpiresAt) {
			am.mu.RUnlock()
			return session, nil
		}
	}
	am.mu.RUnlock()

	// Validate with K8s API
	if am.tokenValidator == nil {
		return nil, fmt.Errorf("K8s token validator not available")
	}

	review, err := am.tokenValidator.ValidateToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Determine role from groups
	role := "viewer"
	for _, group := range review.Groups {
		if strings.Contains(group, "admin") || strings.Contains(group, "cluster-admin") {
			role = "admin"
			break
		}
		if strings.Contains(group, "edit") || strings.Contains(group, "developer") {
			role = "user"
		}
	}

	// Create cached session
	session := &Session{
		ID:        generateSessionID(),
		UserID:    review.UID,
		Username:  review.Username,
		Role:      role,
		Source:    "k8s-token",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(am.config.SessionDuration),
	}

	// Cache the session
	am.mu.Lock()
	am.tokenSessions[token] = session
	am.mu.Unlock()

	return session, nil
}

// HandleKubeconfigLogin handles auto-login using current kubeconfig credentials
// This is only available when running locally (not in-cluster)
func (am *AuthManager) HandleKubeconfigLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if we're running locally with kubeconfig
	if am.tokenValidator == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "Kubernetes client not available",
		})
		return
	}

	if am.tokenValidator.GetEnvironment() != RuntimeLocal {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "Kubeconfig login is only available when running locally",
		})
		return
	}

	// Get the kubeconfig user
	kubeconfigUser := am.tokenValidator.GetKubeconfigUser()
	if kubeconfigUser == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "Could not determine kubeconfig user",
		})
		return
	}

	// Create a session for the kubeconfig user
	session := &Session{
		ID:        generateSessionID(),
		UserID:    "kubeconfig-" + kubeconfigUser,
		Username:  kubeconfigUser,
		Role:      "admin", // Kubeconfig user gets admin role (they have full kubectl access anyway)
		Source:    "kubeconfig",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(am.config.SessionDuration),
	}

	am.mu.Lock()
	am.sessions[session.ID] = session
	am.mu.Unlock()

	// Set session cookie with security flags
	http.SetCookie(w, &http.Cookie{
		Name:     "k13d_session",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(LoginResponse{
		Token:     session.ID,
		Username:  session.Username,
		Role:      session.Role,
		ExpiresAt: session.ExpiresAt,
		AuthMode:  "kubeconfig",
	})
}
