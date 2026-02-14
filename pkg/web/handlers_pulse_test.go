package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/db"
)

func TestHandlePulse(t *testing.T) {
	dbPath := "test_pulse.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	server, _ := setupK8sTestServer(t)

	handler := http.HandlerFunc(server.handlePulse)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		checkResponse  func(t *testing.T, data WebPulseData)
	}{
		{
			name:           "GET pulse returns cluster data",
			path:           "/api/pulse",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, data WebPulseData) {
				// 2 pods from test fixtures
				if data.PodsTotal != 2 {
					t.Errorf("Expected 2 pods total, got %d", data.PodsTotal)
				}
				if data.PodsRunning != 1 {
					t.Errorf("Expected 1 running pod, got %d", data.PodsRunning)
				}
				if data.PodsPending != 1 {
					t.Errorf("Expected 1 pending pod, got %d", data.PodsPending)
				}

				// 1 deployment
				if data.DeploysTotal != 1 {
					t.Errorf("Expected 1 deployment, got %d", data.DeploysTotal)
				}

				// 1 statefulset
				if data.STSTotal != 1 {
					t.Errorf("Expected 1 statefulset, got %d", data.STSTotal)
				}

				// 1 daemonset
				if data.DSTotal != 1 {
					t.Errorf("Expected 1 daemonset, got %d", data.DSTotal)
				}

				// 1 job (completed)
				if data.JobsTotal != 1 {
					t.Errorf("Expected 1 job, got %d", data.JobsTotal)
				}
				if data.JobsComplete != 1 {
					t.Errorf("Expected 1 complete job, got %d", data.JobsComplete)
				}

				// 2 nodes
				if data.NodesTotal != 2 {
					t.Errorf("Expected 2 nodes, got %d", data.NodesTotal)
				}
				if data.NodesReady != 1 {
					t.Errorf("Expected 1 ready node, got %d", data.NodesReady)
				}
				if data.NodesNotReady != 1 {
					t.Errorf("Expected 1 not-ready node, got %d", data.NodesNotReady)
				}

				// 1 event from test fixtures
				if len(data.Events) != 1 {
					t.Errorf("Expected 1 event, got %d", len(data.Events))
				}

				if data.Timestamp.IsZero() {
					t.Error("Expected non-zero timestamp")
				}
			},
		},
		{
			name:           "GET pulse with namespace filter",
			path:           "/api/pulse?namespace=default",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, data WebPulseData) {
				if data.PodsTotal != 2 {
					t.Errorf("Expected 2 pods in default ns, got %d", data.PodsTotal)
				}
			},
		},
		{
			name:           "GET pulse with nonexistent namespace",
			path:           "/api/pulse?namespace=nonexistent",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, data WebPulseData) {
				if data.PodsTotal != 0 {
					t.Errorf("Expected 0 pods in nonexistent ns, got %d", data.PodsTotal)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
				return
			}

			if tt.checkResponse != nil {
				var data WebPulseData
				if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}
				tt.checkResponse(t, data)
			}
		})
	}
}

func TestHandlePulseNilK8sClient(t *testing.T) {
	server := &Server{k8sClient: nil}
	handler := http.HandlerFunc(server.handlePulse)

	req := httptest.NewRequest(http.MethodGet, "/api/pulse", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}
	if body["code"] != ErrCodeK8sError {
		t.Errorf("Expected K8S_ERROR code, got %v", body["code"])
	}
}

func TestFormatAge_Pulse(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		t        time.Time
		contains string
	}{
		{"seconds", now.Add(-30 * time.Second), "s"},
		{"minutes", now.Add(-5 * time.Minute), "m"},
		{"hours", now.Add(-3 * time.Hour), "h"},
		{"days", now.Add(-48 * time.Hour), "d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAge(tt.t)
			if result == "" {
				t.Error("Expected non-empty result")
			}
			if len(result) < 2 {
				t.Errorf("Expected at least 2 chars, got %q", result)
			}
		})
	}
}
