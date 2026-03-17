package web

import (
	"net/http/httptest"
	"os"
	"slices"
	"testing"
)

func TestGetWebSocketAllowedOrigins_AppendsConfiguredOriginsAndKeepsLocalDefaults(t *testing.T) {
	t.Setenv("K13D_WS_ALLOWED_ORIGINS", "https://fingerscore.net, https://admin.fingerscore.net ")
	t.Setenv("K13D_DOMAIN", "")

	origins := getWebSocketAllowedOrigins()

	if !slices.Contains(origins, "http://localhost") {
		t.Fatalf("expected localhost to remain allowed, got %v", origins)
	}
	if !slices.Contains(origins, "https://fingerscore.net") {
		t.Fatalf("expected fingerscore origin to be allowed, got %v", origins)
	}
	if !slices.Contains(origins, "https://admin.fingerscore.net") {
		t.Fatalf("expected admin origin to be trimmed and allowed, got %v", origins)
	}
}

func TestGetWebSocketAllowedOrigins_FallsBackToDomainEnv(t *testing.T) {
	t.Setenv("K13D_WS_ALLOWED_ORIGINS", "")
	t.Setenv("K13D_DOMAIN", "fingerscore.net")

	origins := getWebSocketAllowedOrigins()

	if !slices.Contains(origins, "https://fingerscore.net") {
		t.Fatalf("expected domain-derived origin to be allowed, got %v", origins)
	}
}

func TestCheckOrigin_AllowsConfiguredOriginWithPort(t *testing.T) {
	origOrigins := allowedOrigins
	defer func() { allowedOrigins = origOrigins }()

	t.Setenv("K13D_WS_ALLOWED_ORIGINS", "https://fingerscore.net")
	t.Setenv("K13D_DOMAIN", "")
	allowedOrigins = getWebSocketAllowedOrigins()

	req := httptest.NewRequest("GET", "/api/terminal/default/pod", nil)
	req.Header.Set("Origin", "https://fingerscore.net:443")

	if !checkOrigin(req) {
		t.Fatalf("expected configured origin with explicit port to be allowed")
	}
}

func TestGetWebSocketAllowedOrigins_DoesNotMutateProcessEnv(t *testing.T) {
	before := os.Getenv("K13D_WS_ALLOWED_ORIGINS")
	t.Setenv("K13D_WS_ALLOWED_ORIGINS", "https://fingerscore.net")

	_ = getWebSocketAllowedOrigins()

	if got := os.Getenv("K13D_WS_ALLOWED_ORIGINS"); got == "" || got != "https://fingerscore.net" {
		t.Fatalf("expected env to remain unchanged, before=%q got=%q", before, got)
	}
}
