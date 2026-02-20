package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func setupApplicationsTestServer(t *testing.T) *Server {
	t.Helper()

	one := int32(1)

	fakeClientset := fake.NewSimpleClientset( //nolint:staticcheck
		// Labeled deployment
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx",
				Namespace: "default",
				Labels:    map[string]string{"app.kubernetes.io/name": "nginx", "app.kubernetes.io/version": "1.21"},
			},
			Spec: appsv1.DeploymentSpec{Replicas: &one},
		},
		// Labeled service
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-svc",
				Namespace: "default",
				Labels:    map[string]string{"app.kubernetes.io/name": "nginx"},
			},
		},
		// Labeled pod (running and ready)
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-pod-1",
				Namespace: "default",
				Labels:    map[string]string{"app.kubernetes.io/name": "nginx"},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{Ready: true},
				},
			},
		},
		// Unlabeled configmap (should be ungrouped)
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-root-ca.crt",
				Namespace: "default",
			},
		},
		// Labeled configmap for nginx
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-config",
				Namespace: "default",
				Labels:    map[string]string{"app.kubernetes.io/name": "nginx"},
			},
		},
	)

	cfg := &config.Config{Language: "en"}
	authConfig := &AuthConfig{
		Enabled:         false,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		Quiet:           true,
	}
	authManager := NewAuthManager(authConfig)

	return &Server{
		cfg:         cfg,
		k8sClient:   &k8s.Client{Clientset: fakeClientset},
		authManager: authManager,
	}
}

func TestHandleApplications_ReturnsJSON(t *testing.T) {
	server := setupApplicationsTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/applications?namespace=default", nil)
	w := httptest.NewRecorder()

	server.handleApplications(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var groups []ApplicationGroup
	if err := json.NewDecoder(w.Body).Decode(&groups); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should have nginx group + ungrouped
	if len(groups) < 1 {
		t.Fatalf("expected at least 1 group, got %d", len(groups))
	}

	// Find nginx group
	var nginx *ApplicationGroup
	for i := range groups {
		if groups[i].Name == "nginx" {
			nginx = &groups[i]
			break
		}
	}
	if nginx == nil {
		t.Fatal("expected to find 'nginx' group")
	}

	if nginx.Version != "1.21" {
		t.Errorf("expected version '1.21', got %q", nginx.Version)
	}
	if nginx.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %q", nginx.Status)
	}
	if nginx.PodCount != 1 {
		t.Errorf("expected 1 pod, got %d", nginx.PodCount)
	}
	if nginx.ReadyPods != 1 {
		t.Errorf("expected 1 ready pod, got %d", nginx.ReadyPods)
	}
	if len(nginx.Resources["Deployment"]) != 1 {
		t.Errorf("expected 1 deployment resource, got %d", len(nginx.Resources["Deployment"]))
	}
	if len(nginx.Resources["Service"]) != 1 {
		t.Errorf("expected 1 service resource, got %d", len(nginx.Resources["Service"]))
	}
	if len(nginx.Resources["ConfigMap"]) != 1 {
		t.Errorf("expected 1 configmap resource, got %d", len(nginx.Resources["ConfigMap"]))
	}
}

func TestHandleApplications_MethodNotAllowed(t *testing.T) {
	server := setupApplicationsTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/applications", nil)
	w := httptest.NewRecorder()

	server.handleApplications(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestHandleApplications_EmptyNamespace(t *testing.T) {
	one := int32(1)
	fakeClientset := fake.NewSimpleClientset( //nolint:staticcheck
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "app1",
				Namespace: "ns1",
				Labels:    map[string]string{"app.kubernetes.io/name": "app1"},
			},
			Spec: appsv1.DeploymentSpec{Replicas: &one},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "app2",
				Namespace: "ns2",
				Labels:    map[string]string{"app.kubernetes.io/name": "app2"},
			},
			Spec: appsv1.DeploymentSpec{Replicas: &one},
		},
	)

	cfg := &config.Config{Language: "en"}
	authConfig := &AuthConfig{
		Enabled:         false,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		Quiet:           true,
	}
	authManager := NewAuthManager(authConfig)

	server := &Server{
		cfg:         cfg,
		k8sClient:   &k8s.Client{Clientset: fakeClientset},
		authManager: authManager,
	}

	// Request without namespace filter (all namespaces)
	req := httptest.NewRequest(http.MethodGet, "/api/applications", nil)
	w := httptest.NewRecorder()

	server.handleApplications(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var groups []ApplicationGroup
	if err := json.NewDecoder(w.Body).Decode(&groups); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should see both apps across namespaces
	if len(groups) < 2 {
		t.Errorf("expected at least 2 groups for all namespaces, got %d", len(groups))
	}
}

func TestHandleApplications_UngroupedResources(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset( //nolint:staticcheck
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "orphan-cm",
				Namespace: "default",
			},
		},
	)

	cfg := &config.Config{Language: "en"}
	authConfig := &AuthConfig{
		Enabled:         false,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		Quiet:           true,
	}
	authManager := NewAuthManager(authConfig)

	server := &Server{
		cfg:         cfg,
		k8sClient:   &k8s.Client{Clientset: fakeClientset},
		authManager: authManager,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/applications?namespace=default", nil)
	w := httptest.NewRecorder()

	server.handleApplications(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var groups []ApplicationGroup
	if err := json.NewDecoder(w.Body).Decode(&groups); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should have ungrouped
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Name != "Ungrouped" {
		t.Errorf("expected 'Ungrouped', got %q", groups[0].Name)
	}
	if len(groups[0].Resources["ConfigMap"]) != 1 {
		t.Errorf("expected 1 ungrouped configmap")
	}
}
