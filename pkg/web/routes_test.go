package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

// TestAllRouteGroupsRegister verifies that all route group methods
// register at least one handler without panicking.
func TestAllRouteGroupsRegister(t *testing.T) {
	cfg := &config.Config{
		Language:    "en",
		EnableAudit: false,
		LLM: config.LLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
	}

	authConfig := &AuthConfig{
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "test123",
		Quiet:           true,
	}
	authManager := NewAuthManager(authConfig)
	authorizer := NewAuthorizer()
	accessReqManager := NewAccessRequestManager(30 * time.Minute)

	server := &Server{
		cfg:                  cfg,
		authManager:          authManager,
		authorizer:           authorizer,
		accessRequestManager: accessReqManager,
		portForwardSessions:  make(map[string]*PortForwardSession),
	}
	server.reportGenerator = NewReportGenerator(server)

	mux := http.NewServeMux()

	// All 10 route group methods should register without panic
	server.registerPublicRoutes(mux)
	server.registerAuthRoutes(mux)
	server.registerAIRoutes(mux)
	server.registerK8sRoutes(mux)
	server.registerWorkloadOperationRoutes(mux)
	server.registerHelmRoutes(mux)
	server.registerMetricsRoutes(mux)
	server.registerSecurityRoutes(mux)
	server.registerVisualizationRoutes(mux)
	server.registerAdminRoutes(mux)
}

// TestCriticalEndpointsRegistered verifies that key API endpoints
// return a non-404 response (i.e., they are actually registered).
func TestCriticalEndpointsRegistered(t *testing.T) {
	cfg := &config.Config{
		Language:    "en",
		EnableAudit: false,
		LLM: config.LLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
	}

	authConfig := &AuthConfig{
		Enabled:         false,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		Quiet:           true,
	}
	authManager := NewAuthManager(authConfig)
	authorizer := NewAuthorizer()
	accessReqManager := NewAccessRequestManager(30 * time.Minute)

	server := &Server{
		cfg:                  cfg,
		authManager:          authManager,
		authorizer:           authorizer,
		accessRequestManager: accessReqManager,
		portForwardSessions:  make(map[string]*PortForwardSession),
	}
	server.reportGenerator = NewReportGenerator(server)

	mux := http.NewServeMux()
	server.registerPublicRoutes(mux)
	server.registerAuthRoutes(mux)
	server.registerAIRoutes(mux)
	server.registerK8sRoutes(mux)
	server.registerWorkloadOperationRoutes(mux)
	server.registerHelmRoutes(mux)
	server.registerMetricsRoutes(mux)
	server.registerSecurityRoutes(mux)
	server.registerVisualizationRoutes(mux)
	server.registerAdminRoutes(mux)

	// Critical endpoints that must be registered
	endpoints := []string{
		"/api/health",
		"/api/version",
		"/api/auth/login",
		"/api/auth/logout",
		"/api/auth/status",
	}

	for _, ep := range endpoints {
		t.Run(ep, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, ep, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			// 404 means the route was not registered
			if w.Code == http.StatusNotFound {
				t.Errorf("endpoint %s returned 404 — not registered", ep)
			}
		})
	}
}
