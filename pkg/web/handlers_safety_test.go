package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAnalyzeK8sSafety(t *testing.T) {
	tests := []struct {
		name              string
		command           string
		namespace         string
		expectedRiskLevel string
		expectedSafe      bool
		expectedApproval  bool
	}{
		{
			name:              "Read-only get pods",
			command:           "kubectl get pods",
			expectedRiskLevel: "safe",
			expectedSafe:      true,
			expectedApproval:  false,
		},
		{
			name:              "Describe deployment",
			command:           "kubectl describe deployment nginx",
			expectedRiskLevel: "safe",
			expectedSafe:      true,
			expectedApproval:  false,
		},
		{
			name:              "Get logs",
			command:           "kubectl logs pod-name",
			expectedRiskLevel: "safe",
			expectedSafe:      true,
			expectedApproval:  false,
		},
		{
			name:              "Delete pod - warning",
			command:           "kubectl delete pod nginx-pod",
			expectedRiskLevel: "warning",
			expectedSafe:      true,
			expectedApproval:  true,
		},
		{
			name:              "Delete deployment - dangerous",
			command:           "kubectl delete deployment nginx",
			expectedRiskLevel: "dangerous",
			expectedSafe:      false,
			expectedApproval:  true,
		},
		{
			name:              "Delete namespace - critical",
			command:           "kubectl delete namespace test-ns",
			expectedRiskLevel: "critical",
			expectedSafe:      false,
			expectedApproval:  true,
		},
		{
			name:              "Delete with --all - warning (pod level)",
			command:           "kubectl delete pods --all",
			expectedRiskLevel: "warning",
			expectedSafe:      true,
			expectedApproval:  true,
		},
		{
			name:              "Drain node - critical",
			command:           "kubectl drain node-1",
			expectedRiskLevel: "critical",
			expectedSafe:      false,
			expectedApproval:  true,
		},
		{
			name:              "Scale deployment - warning",
			command:           "kubectl scale deployment nginx --replicas=3",
			expectedRiskLevel: "warning",
			expectedSafe:      true,
			expectedApproval:  true,
		},
		{
			name:              "Scale to zero - warning",
			command:           "kubectl scale deployment nginx --replicas=0",
			expectedRiskLevel: "warning",
			expectedSafe:      true,
			expectedApproval:  true,
		},
		{
			name:              "Apply manifest - warning",
			command:           "kubectl apply -f deployment.yaml",
			expectedRiskLevel: "warning",
			expectedSafe:      true,
			expectedApproval:  true,
		},
		{
			name:              "Rollout restart - warning",
			command:           "kubectl rollout restart deployment nginx",
			expectedRiskLevel: "warning",
			expectedSafe:      true,
			expectedApproval:  true,
		},
		{
			name:              "Force delete - critical",
			command:           "kubectl delete pod nginx --force --grace-period=0",
			expectedRiskLevel: "critical",
			expectedSafe:      false,
			expectedApproval:  true,
		},
		{
			name:              "Exec command - critical",
			command:           "kubectl exec -it pod-name -- /bin/bash",
			expectedRiskLevel: "critical",
			expectedSafe:      false,
			expectedApproval:  true,
		},
		{
			name:              "Delete PVC - critical",
			command:           "kubectl delete pvc data-volume",
			expectedRiskLevel: "critical",
			expectedSafe:      false,
			expectedApproval:  true,
		},
		{
			name:              "Delete secret - dangerous",
			command:           "kubectl delete secret app-secret",
			expectedRiskLevel: "dangerous",
			expectedSafe:      false,
			expectedApproval:  true,
		},
		{
			name:              "Patch resource - dangerous",
			command:           "kubectl patch deployment nginx -p '{\"spec\":{\"replicas\":5}}'",
			expectedRiskLevel: "dangerous",
			expectedSafe:      false,
			expectedApproval:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := SafetyAnalysisRequest{
				Command:   tt.command,
				Namespace: tt.namespace,
			}

			result := analyzeK8sSafety(req)

			if result.RiskLevel != tt.expectedRiskLevel {
				t.Errorf("RiskLevel = %v, want %v", result.RiskLevel, tt.expectedRiskLevel)
			}

			if result.Safe != tt.expectedSafe {
				t.Errorf("Safe = %v, want %v", result.Safe, tt.expectedSafe)
			}

			if result.RequiresApproval != tt.expectedApproval {
				t.Errorf("RequiresApproval = %v, want %v", result.RequiresApproval, tt.expectedApproval)
			}
		})
	}
}

func TestAnalyzeK8sSafetyNamespaceWarnings(t *testing.T) {
	tests := []struct {
		name              string
		command           string
		namespace         string
		expectExtraWarn   bool
		expectedRiskLevel string
	}{
		{
			name:              "Delete in kube-system - escalates risk",
			command:           "kubectl delete pod coredns",
			namespace:         "kube-system",
			expectExtraWarn:   true,
			expectedRiskLevel: "dangerous", // Escalated from warning
		},
		{
			name:              "Delete in default - escalates risk",
			command:           "kubectl delete pod nginx",
			namespace:         "default",
			expectExtraWarn:   true,
			expectedRiskLevel: "dangerous", // Escalated from warning
		},
		{
			name:              "Delete in user namespace - normal risk",
			command:           "kubectl delete pod nginx",
			namespace:         "user-namespace",
			expectExtraWarn:   false,
			expectedRiskLevel: "warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := SafetyAnalysisRequest{
				Command:   tt.command,
				Namespace: tt.namespace,
			}

			result := analyzeK8sSafety(req)

			hasNamespaceWarning := false
			for _, w := range result.Warnings {
				if containsAny(w, []string{"sensitive namespace", "kube-system", "default"}) {
					hasNamespaceWarning = true
					break
				}
			}

			if tt.expectExtraWarn && !hasNamespaceWarning {
				t.Errorf("Expected namespace warning for %s, but didn't get one. Warnings: %v",
					tt.namespace, result.Warnings)
			}

			if result.RiskLevel != tt.expectedRiskLevel {
				t.Errorf("RiskLevel = %v, want %v for namespace %s",
					result.RiskLevel, tt.expectedRiskLevel, tt.namespace)
			}
		})
	}
}

func TestAnalyzeK8sSafetyProductionIndicators(t *testing.T) {
	tests := []struct {
		name              string
		command           string
		namespace         string
		expectProdWarning bool
	}{
		{
			// Production indicator detection only works on non-read-only commands
			// or when namespace contains production keywords
			name:              "Delete in prod namespace",
			command:           "kubectl delete pod nginx -n prod",
			namespace:         "",
			expectProdWarning: true,
		},
		{
			name:              "Scale in production namespace",
			command:           "kubectl scale deployment nginx --replicas=3 -n production",
			namespace:         "",
			expectProdWarning: true,
		},
		{
			name:              "Normal development command",
			command:           "kubectl scale deployment nginx -n development",
			namespace:         "",
			expectProdWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := SafetyAnalysisRequest{
				Command:   tt.command,
				Namespace: tt.namespace,
			}

			result := analyzeK8sSafety(req)

			hasProdWarning := false
			for _, w := range result.Warnings {
				if containsAny(w, []string{"production", "Production", "prod"}) {
					hasProdWarning = true
					break
				}
			}

			if tt.expectProdWarning && !hasProdWarning {
				t.Errorf("Expected production warning, but didn't get one. Warnings: %v", result.Warnings)
			}

			if tt.expectProdWarning && !result.RequiresApproval {
				t.Error("Expected RequiresApproval=true for production environment")
			}
		})
	}
}

func TestSafetyAnalysisHandler(t *testing.T) {
	// Create a minimal test server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req SafetyAnalysisRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		response := analyzeK8sSafety(req)
		json.NewEncoder(w).Encode(response)
	})

	tests := []struct {
		name           string
		method         string
		body           SafetyAnalysisRequest
		expectedStatus int
	}{
		{
			name:   "Valid POST request",
			method: http.MethodPost,
			body: SafetyAnalysisRequest{
				Command: "kubectl get pods",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid method",
			method:         http.MethodGet,
			body:           SafetyAnalysisRequest{},
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(tt.method, "/api/safety/analyze", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", rr.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var response SafetyAnalysisResponse
				if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}
			}
		})
	}
}

// Helper function
func containsAny(s string, substrings []string) bool {
	for _, sub := range substrings {
		if len(sub) > 0 && len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
