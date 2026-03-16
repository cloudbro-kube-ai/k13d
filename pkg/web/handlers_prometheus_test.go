package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

func TestPrometheusSettings_GET_IncludesRetentionDays(t *testing.T) {
	s := setupSettingsTestServer(t)
	s.cfg.Prometheus.ExposeMetrics = true
	s.cfg.Prometheus.CollectionInterval = 120
	s.cfg.Storage.MetricsRetentionDays = 14

	req := httptest.NewRequest(http.MethodGet, "/api/prometheus/settings", nil)
	w := httptest.NewRecorder()

	s.handlePrometheusSettings(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/prometheus/settings: status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["metrics_retention_days"] != float64(14) {
		t.Fatalf("metrics_retention_days = %v, want 14", resp["metrics_retention_days"])
	}
}

func TestPrometheusSettings_PUT_PersistsRetentionDays(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("K13D_CONFIG", configPath)

	s := setupSettingsTestServer(t)
	s.cfg.Storage.MetricsRetentionDays = 30

	req := httptest.NewRequest(http.MethodPut, "/api/prometheus/settings", strings.NewReader(`{
		"expose_metrics": true,
		"external_url": "https://prom.example.com",
		"collect_k8s_metrics": true,
		"collection_interval": 300,
		"metrics_retention_days": 90
	}`))
	w := httptest.NewRecorder()

	s.handlePrometheusSettings(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT /api/prometheus/settings: status = %d, want %d body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	if s.cfg.Storage.MetricsRetentionDays != 90 {
		t.Fatalf("Storage.MetricsRetentionDays = %d, want 90", s.cfg.Storage.MetricsRetentionDays)
	}

	loaded, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loaded.Storage.MetricsRetentionDays != 90 {
		t.Fatalf("saved Storage.MetricsRetentionDays = %d, want 90", loaded.Storage.MetricsRetentionDays)
	}
}
