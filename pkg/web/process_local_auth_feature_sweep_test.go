package web

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestProcessE2E_LocalAuthFeatureSweep(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping process E2E test in short mode")
	}
	if os.Getenv("K13D_RUN_PROCESS_E2E") != "1" {
		t.Skip("set K13D_RUN_PROCESS_E2E=1 to run the process-based local auth E2E tests")
	}

	requireProcessE2EClusterAccess(t)

	server := startProcessLocalAuthE2EServer(t)
	defer server.Close(t)

	server.Login(t)
	csrfToken := server.CSRFToken(t)

	type requestCase struct {
		name         string
		method       string
		path         string
		payload      interface{}
		headers      map[string]string
		wantStatuses []int
		wantContains []string
	}

	cases := []requestCase{
		{name: "version", method: http.MethodGet, path: "/api/version", wantStatuses: []int{http.StatusOK}, wantContains: []string{`"version"`}},
		{name: "current-user", method: http.MethodGet, path: "/api/auth/me", wantStatuses: []int{http.StatusOK}, wantContains: []string{server.adminUser, `"role"`}},
		{name: "permissions", method: http.MethodGet, path: "/api/auth/permissions", wantStatuses: []int{http.StatusOK}, wantContains: []string{`"features"`}},
		{name: "settings", method: http.MethodGet, path: "/api/settings", wantStatuses: []int{http.StatusOK}, wantContains: []string{`"language"`}},
		{name: "llm-status", method: http.MethodGet, path: "/api/llm/status", wantStatuses: []int{http.StatusOK}, wantContains: []string{`"provider"`}},
		{name: "ai-ping", method: http.MethodGet, path: "/api/ai/ping", wantStatuses: []int{http.StatusOK, http.StatusServiceUnavailable}},
		{name: "models", method: http.MethodGet, path: "/api/models", wantStatuses: []int{http.StatusOK}},
		{name: "active-model", method: http.MethodGet, path: "/api/models/active", wantStatuses: []int{http.StatusOK, http.StatusNotFound}},
		{name: "mcp-servers", method: http.MethodGet, path: "/api/mcp/servers", wantStatuses: []int{http.StatusOK}},
		{name: "mcp-tools", method: http.MethodGet, path: "/api/mcp/tools", wantStatuses: []int{http.StatusOK}},
		{name: "k8s-pods", method: http.MethodGet, path: "/api/k8s/pods", wantStatuses: []int{http.StatusOK}, wantContains: []string{`"items"`}},
		{name: "k8s-namespaces", method: http.MethodGet, path: "/api/k8s/namespaces", wantStatuses: []int{http.StatusOK}, wantContains: []string{`"items"`}},
		{name: "safety-analyze", method: http.MethodPost, path: "/api/safety/analyze", payload: map[string]string{"command": "kubectl get pods -n default"}, headers: map[string]string{"X-CSRF-Token": csrfToken}, wantStatuses: []int{http.StatusOK}, wantContains: []string{`"category":"read-only"`}},
		{name: "contexts", method: http.MethodGet, path: "/api/contexts", wantStatuses: []int{http.StatusOK}, wantContains: []string{`"contexts"`}},
		{name: "notification-history", method: http.MethodGet, path: "/api/notifications/history", wantStatuses: []int{http.StatusOK}},
		{name: "notification-status", method: http.MethodGet, path: "/api/notifications/status", wantStatuses: []int{http.StatusOK}, wantContains: []string{`"running"`}},
		{name: "portforwards", method: http.MethodGet, path: "/api/portforward/list", wantStatuses: []int{http.StatusOK}},
		{name: "metrics-summary", method: http.MethodGet, path: "/api/metrics/history/summary", wantStatuses: []int{http.StatusOK}},
		{name: "prometheus-settings", method: http.MethodGet, path: "/api/prometheus/settings", wantStatuses: []int{http.StatusOK}},
		{name: "trivy-status", method: http.MethodGet, path: "/api/security/trivy/status", wantStatuses: []int{http.StatusOK}, wantContains: []string{`"installed"`}},
		{name: "trivy-instructions", method: http.MethodGet, path: "/api/security/trivy/instructions", wantStatuses: []int{http.StatusOK}, wantContains: []string{`"instructions"`}},
		{name: "audit", method: http.MethodGet, path: "/api/audit", wantStatuses: []int{http.StatusOK}},
		{name: "roles", method: http.MethodGet, path: "/api/roles", wantStatuses: []int{http.StatusOK}, wantContains: []string{`"roles"`}},
		{name: "admin-users", method: http.MethodGet, path: "/api/admin/users", wantStatuses: []int{http.StatusOK}, wantContains: []string{server.adminUser}},
		{name: "admin-status", method: http.MethodGet, path: "/api/admin/status", wantStatuses: []int{http.StatusOK}, wantContains: []string{`"auth_mode":"local"`}},
		{name: "access-requests-list", method: http.MethodGet, path: "/api/access/requests", wantStatuses: []int{http.StatusOK}, wantContains: []string{`"requests"`}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			headers := copyProcessE2EHeaders(tc.headers)
			if tc.payload != nil {
				headers["Content-Type"] = "application/json"
				if headers["X-CSRF-Token"] == "" {
					headers["X-CSRF-Token"] = csrfToken
				}
			}

			resp, body := doProcessE2ERequest(t, server.client, tc.method, server.baseURL+tc.path, tc.payload, headers)
			resp.Body.Close()

			if !processE2EStatusAllowed(resp.StatusCode, tc.wantStatuses) {
				t.Fatalf("%s %s returned %d, want one of %v\n%s", tc.method, tc.path, resp.StatusCode, tc.wantStatuses, body)
			}

			bodyText := string(body)
			for _, want := range tc.wantContains {
				if !strings.Contains(bodyText, want) {
					t.Fatalf("%s %s response missing %q\n%s", tc.method, tc.path, want, bodyText)
				}
			}
		})
	}

	t.Run("access-request-create-list-approve", func(t *testing.T) {
		reviewerUser := "e2e-reviewer"
		reviewerPass := "e2e-reviewer-pass"

		createUserResp, createUserBody := doProcessE2ERequest(t, server.client, http.MethodPost, server.baseURL+"/api/admin/users", map[string]string{
			"username": reviewerUser,
			"password": reviewerPass,
			"role":     "viewer",
			"email":    "e2e-reviewer@example.com",
		}, map[string]string{
			"Content-Type": "application/json",
			"X-CSRF-Token": csrfToken,
		})
		createUserResp.Body.Close()
		if createUserResp.StatusCode != http.StatusCreated {
			t.Fatalf("create reviewer user failed with status %d:\n%s", createUserResp.StatusCode, createUserBody)
		}

		t.Cleanup(func() {
			deleteResp, deleteBody := doProcessE2ERequest(t, server.client, http.MethodDelete, server.baseURL+"/api/admin/users/"+reviewerUser, nil, map[string]string{
				"X-CSRF-Token": csrfToken,
			})
			deleteResp.Body.Close()
			if deleteResp.StatusCode != http.StatusOK && deleteResp.StatusCode != http.StatusNotFound {
				t.Fatalf("delete reviewer user failed with status %d:\n%s", deleteResp.StatusCode, deleteBody)
			}
		})

		reviewerClient, reviewerLogin := server.NewAuthenticatedClient(t, reviewerUser, reviewerPass)
		if reviewerLogin.Role != "viewer" {
			t.Fatalf("expected reviewer login role viewer, got %+v", reviewerLogin)
		}
		reviewerCSRF := server.CSRFTokenForClient(t, reviewerClient)

		createResp, createBody := doProcessE2ERequest(t, reviewerClient, http.MethodPost, server.baseURL+"/api/access/request", map[string]string{
			"action":    string(ActionScale),
			"resource":  "deployments",
			"namespace": "default",
			"reason":    "Need temporary access for E2E verification",
		}, map[string]string{
			"Content-Type": "application/json",
			"X-CSRF-Token": reviewerCSRF,
		})
		createResp.Body.Close()
		if createResp.StatusCode != http.StatusCreated {
			t.Fatalf("create access request failed with status %d:\n%s", createResp.StatusCode, createBody)
		}

		var created struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}
		if err := json.Unmarshal(createBody, &created); err != nil {
			t.Fatalf("failed to decode created access request: %v\n%s", err, createBody)
		}
		if created.ID == "" || created.Status != "pending" {
			t.Fatalf("unexpected create access request response: %+v", created)
		}

		listResp, listBody := doProcessE2ERequest(t, server.client, http.MethodGet, server.baseURL+"/api/access/requests", nil, nil)
		listResp.Body.Close()
		if listResp.StatusCode != http.StatusOK || !strings.Contains(string(listBody), created.ID) {
			t.Fatalf("expected access request list to include %q, got status=%d body=%s", created.ID, listResp.StatusCode, listBody)
		}

		approveResp, approveBody := doProcessE2ERequest(t, server.client, http.MethodPost, server.baseURL+"/api/access/approve/"+created.ID, map[string]string{
			"note": "approved by process E2E",
		}, map[string]string{
			"Content-Type": "application/json",
			"X-CSRF-Token": csrfToken,
		})
		approveResp.Body.Close()
		if approveResp.StatusCode != http.StatusOK {
			t.Fatalf("approve access request failed with status %d:\n%s", approveResp.StatusCode, approveBody)
		}
		if !strings.Contains(string(approveBody), `"status":"approved"`) {
			t.Fatalf("expected approved status in response, got:\n%s", approveBody)
		}

		finalListResp, finalListBody := doProcessE2ERequest(t, server.client, http.MethodGet, server.baseURL+"/api/access/requests", nil, nil)
		finalListResp.Body.Close()
		if finalListResp.StatusCode != http.StatusOK {
			t.Fatalf("final access request list failed with status %d:\n%s", finalListResp.StatusCode, finalListBody)
		}
		if strings.Contains(string(finalListBody), created.ID) {
			t.Fatalf("approved access request %q should not remain in pending list:\n%s", created.ID, finalListBody)
		}
	})

	t.Run("ollama-model-registration-warning", func(t *testing.T) {
		modelName := "e2e-ollama-warning"

		createResp, createBody := doProcessE2ERequest(t, server.client, http.MethodPost, server.baseURL+"/api/models", map[string]string{
			"name":     modelName,
			"provider": "ollama",
			"model":    "llama3.2",
			"endpoint": "http://localhost:11434",
		}, map[string]string{
			"Content-Type": "application/json",
			"X-CSRF-Token": csrfToken,
		})
		createResp.Body.Close()
		if createResp.StatusCode != http.StatusOK {
			t.Fatalf("create model failed with status %d:\n%s", createResp.StatusCode, createBody)
		}
		if !strings.Contains(string(createBody), `"warning"`) || !strings.Contains(string(createBody), `tools/function calling`) {
			t.Fatalf("expected Ollama tool warning in create response:\n%s", createBody)
		}

		t.Cleanup(func() {
			deleteResp, deleteBody := doProcessE2ERequest(t, server.client, http.MethodDelete, server.baseURL+"/api/models?name="+modelName, nil, map[string]string{
				"X-CSRF-Token": csrfToken,
			})
			deleteResp.Body.Close()
			if deleteResp.StatusCode != http.StatusOK && deleteResp.StatusCode != http.StatusNotFound {
				t.Fatalf("delete model failed with status %d:\n%s", deleteResp.StatusCode, deleteBody)
			}
		})

		listResp, listBody := doProcessE2ERequest(t, server.client, http.MethodGet, server.baseURL+"/api/models", nil, nil)
		listResp.Body.Close()
		if listResp.StatusCode != http.StatusOK {
			t.Fatalf("list models failed with status %d:\n%s", listResp.StatusCode, listBody)
		}
		if !strings.Contains(string(listBody), modelName) || !strings.Contains(string(listBody), `tools/function calling`) {
			t.Fatalf("expected Ollama warning in models list:\n%s", listBody)
		}
	})
}

func processE2EStatusAllowed(got int, want []int) bool {
	for _, candidate := range want {
		if got == candidate {
			return true
		}
	}
	return false
}

func copyProcessE2EHeaders(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}
