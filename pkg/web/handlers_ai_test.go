package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai/session"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/config"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/db"
)

// setupAITestServer creates a test server for AI handler tests.
// If withSessions is true, a temp session store is initialized.
func setupAITestServer(t *testing.T, withSessions bool) *Server {
	t.Helper()

	cfg := &config.Config{
		Language: "en",
		LLM: config.LLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
			Endpoint: "https://api.openai.com/v1",
			APIKey:   "test-key",
		},
	}

	authConfig := &AuthConfig{
		Enabled:         false,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "admin123",
		Quiet:           true,
	}

	s := &Server{
		cfg:              cfg,
		aiClient:         nil, // nil by default; tests can set this
		authManager:      NewAuthManager(authConfig),
		port:             8080,
		pendingApprovals: make(map[string]*PendingToolApproval),
	}

	if withSessions {
		dir := t.TempDir()
		store, err := session.NewStoreWithDir(dir)
		if err != nil {
			t.Fatalf("Failed to create session store: %v", err)
		}
		s.sessionStore = store
	}

	return s
}

// ==================== handleLLMStatus Tests ====================

func TestLLMStatus_ReturnsConfigWhenNoClient(t *testing.T) {
	dbPath := "test_llm_status.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	s := setupAITestServer(t, false)

	req := httptest.NewRequest(http.MethodGet, "/api/llm/status", nil)
	w := httptest.NewRecorder()

	s.handleLLMStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// When aiClient is nil, configured should be false
	if body["configured"] != false {
		t.Errorf("Expected configured=false, got %v", body["configured"])
	}
	if body["provider"] != "openai" {
		t.Errorf("Expected provider=openai, got %v", body["provider"])
	}
	if body["model"] != "gpt-4" {
		t.Errorf("Expected model=gpt-4, got %v", body["model"])
	}
	if body["has_api_key"] != true {
		t.Errorf("Expected has_api_key=true, got %v", body["has_api_key"])
	}
	if body["embedded_llm"] != false {
		t.Errorf("Expected embedded_llm=false, got %v", body["embedded_llm"])
	}
}

func TestLLMStatus_DefaultEndpointHints(t *testing.T) {
	dbPath := "test_llm_status_hints.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	providers := []struct {
		name             string
		expectedEndpoint string
	}{
		{"openai", "https://api.openai.com/v1"},
		{"ollama", "http://localhost:11434"},
		{"anthropic", "https://api.anthropic.com"},
		{"gemini", "https://generativelanguage.googleapis.com/v1beta"},
		{"azure", "(Azure OpenAI endpoint required)"},
	}

	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			s := setupAITestServer(t, false)
			s.cfg.LLM.Provider = p.name
			s.cfg.LLM.Endpoint = "" // empty to trigger hint

			req := httptest.NewRequest(http.MethodGet, "/api/llm/status", nil)
			w := httptest.NewRecorder()

			s.handleLLMStatus(w, req)

			var body map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &body)

			if body["default_endpoint"] != p.expectedEndpoint {
				t.Errorf("Expected default_endpoint=%q, got %v", p.expectedEndpoint, body["default_endpoint"])
			}
		})
	}
}

func TestLLMStatus_MethodNotAllowed(t *testing.T) {
	s := setupAITestServer(t, false)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/llm/status", nil)
			w := httptest.NewRecorder()

			s.handleLLMStatus(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected 405, got %d", w.Code)
			}
		})
	}
}

// ==================== handleAIPing Tests ====================

func TestAIPing_NilClient(t *testing.T) {
	s := setupAITestServer(t, false)
	s.aiClient = nil

	req := httptest.NewRequest(http.MethodGet, "/api/ai/ping", nil)
	w := httptest.NewRecorder()

	s.handleAIPing(w, req)

	// Should return an error when no AI client is configured
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// WriteError produces APIError JSON with "code" and "message" fields
	if body["code"] == nil {
		t.Error("Expected code field in response when aiClient is nil")
	}
	if code, ok := body["code"].(string); !ok || code != ErrCodeLLMNotConfigured {
		t.Errorf("Expected code=%s, got %v", ErrCodeLLMNotConfigured, body["code"])
	}
}

// ==================== handleSessions Tests ====================

func TestSessions_ListEmpty(t *testing.T) {
	s := setupAITestServer(t, true)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	w := httptest.NewRecorder()

	s.handleSessions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var sessions []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &sessions); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("Expected empty session list, got %d", len(sessions))
	}
}

func TestSessions_CreateAndList(t *testing.T) {
	s := setupAITestServer(t, true)

	// Create a session via POST
	req := httptest.NewRequest(http.MethodPost, "/api/sessions", nil)
	w := httptest.NewRecorder()

	s.handleSessions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var created map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if created["id"] == nil || created["id"] == "" {
		t.Error("Expected session ID in response")
	}
	if created["provider"] != "openai" {
		t.Errorf("Expected provider=openai, got %v", created["provider"])
	}

	// Now list sessions - should have 1
	req = httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	w = httptest.NewRecorder()

	s.handleSessions(w, req)

	var sessions []interface{}
	json.Unmarshal(w.Body.Bytes(), &sessions)

	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}
}

func TestSessions_ClearAll(t *testing.T) {
	s := setupAITestServer(t, true)

	// Create two sessions
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/sessions", nil)
		w := httptest.NewRecorder()
		s.handleSessions(w, req)
	}

	// Clear all sessions
	req := httptest.NewRequest(http.MethodDelete, "/api/sessions", nil)
	w := httptest.NewRecorder()

	s.handleSessions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)

	if body["status"] != "cleared" {
		t.Errorf("Expected status=cleared, got %v", body["status"])
	}

	// Verify list is empty
	req = httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	w = httptest.NewRecorder()
	s.handleSessions(w, req)

	var sessions []interface{}
	json.Unmarshal(w.Body.Bytes(), &sessions)

	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions after clear, got %d", len(sessions))
	}
}

func TestSessions_NilStore(t *testing.T) {
	s := setupAITestServer(t, false) // no session store

	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	w := httptest.NewRecorder()

	s.handleSessions(w, req)

	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)

	if body["code"] != ErrCodeInternalError {
		t.Errorf("Expected error code %s, got %v", ErrCodeInternalError, body["code"])
	}
}

func TestSessions_MethodNotAllowed(t *testing.T) {
	s := setupAITestServer(t, true)

	req := httptest.NewRequest(http.MethodPatch, "/api/sessions", nil)
	w := httptest.NewRecorder()

	s.handleSessions(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", w.Code)
	}
}

// ==================== handleSession (single) Tests ====================

func TestSession_GetByID(t *testing.T) {
	s := setupAITestServer(t, true)

	// Create a session first
	created, _ := s.sessionStore.Create("openai", "gpt-4")

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+created.ID, nil)
	w := httptest.NewRecorder()

	s.handleSession(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)

	if body["id"] != created.ID {
		t.Errorf("Expected id=%s, got %v", created.ID, body["id"])
	}
}

func TestSession_GetNotFound(t *testing.T) {
	s := setupAITestServer(t, true)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/nonexistent-id-12345", nil)
	w := httptest.NewRecorder()

	s.handleSession(w, req)

	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)

	if body["code"] != ErrCodeNotFound {
		t.Errorf("Expected error code %s, got %v", ErrCodeNotFound, body["code"])
	}
}

func TestSession_Delete(t *testing.T) {
	s := setupAITestServer(t, true)

	created, _ := s.sessionStore.Create("openai", "gpt-4")

	req := httptest.NewRequest(http.MethodDelete, "/api/sessions/"+created.ID, nil)
	w := httptest.NewRecorder()

	s.handleSession(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)

	if body["status"] != "deleted" {
		t.Errorf("Expected status=deleted, got %v", body["status"])
	}

	// Verify session is gone
	req = httptest.NewRequest(http.MethodGet, "/api/sessions/"+created.ID, nil)
	w = httptest.NewRecorder()
	s.handleSession(w, req)

	json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != ErrCodeNotFound {
		t.Errorf("Expected NOT_FOUND after delete, got %v", body["code"])
	}
}

func TestSession_UpdateTitle(t *testing.T) {
	s := setupAITestServer(t, true)

	created, _ := s.sessionStore.Create("openai", "gpt-4")

	body := `{"title":"My Custom Title"}`
	req := httptest.NewRequest(http.MethodPut, "/api/sessions/"+created.ID, strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleSession(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	// Verify title was updated
	sess, err := s.sessionStore.Get(created.ID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}
	if sess.Title != "My Custom Title" {
		t.Errorf("Expected title 'My Custom Title', got %q", sess.Title)
	}
}

func TestSession_EmptyID(t *testing.T) {
	s := setupAITestServer(t, true)

	// Path is /api/sessions/ with empty ID after trim
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/", nil)
	w := httptest.NewRecorder()

	s.handleSession(w, req)

	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)

	if body["code"] != ErrCodeBadRequest {
		t.Errorf("Expected error code %s, got %v", ErrCodeBadRequest, body["code"])
	}
}

func TestSession_NilStore(t *testing.T) {
	s := setupAITestServer(t, false) // no session store

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/some-id", nil)
	w := httptest.NewRecorder()

	s.handleSession(w, req)

	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)

	if body["code"] != ErrCodeInternalError {
		t.Errorf("Expected error code %s, got %v", ErrCodeInternalError, body["code"])
	}
}

func TestSession_MethodNotAllowed(t *testing.T) {
	s := setupAITestServer(t, true)

	req := httptest.NewRequest(http.MethodPatch, "/api/sessions/some-id", nil)
	w := httptest.NewRecorder()

	s.handleSession(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", w.Code)
	}
}

// ==================== handleToolApprove Tests ====================

func TestToolApprove_ValidApproval(t *testing.T) {
	s := setupAITestServer(t, false)

	// Create a pending approval with buffered channel
	approvalID := "test-approval-1"
	s.pendingApprovals[approvalID] = &PendingToolApproval{
		ID:        approvalID,
		ToolName:  "kubectl",
		Command:   "kubectl delete pod test",
		Category:  "dangerous",
		Timestamp: time.Now(),
		Response:  make(chan bool, 1),
	}

	body := `{"id":"test-approval-1","approved":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/tool/approve", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleToolApprove(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "ok" {
		t.Errorf("Expected status=ok, got %v", resp["status"])
	}

	// Verify the approval was sent to the channel
	select {
	case approved := <-s.pendingApprovals[approvalID].Response:
		if !approved {
			t.Error("Expected approval to be true")
		}
	default:
		t.Error("Expected approval response in channel")
	}
}

func TestToolApprove_Rejection(t *testing.T) {
	s := setupAITestServer(t, false)

	approvalID := "test-approval-reject"
	s.pendingApprovals[approvalID] = &PendingToolApproval{
		ID:       approvalID,
		Response: make(chan bool, 1),
	}

	body := `{"id":"test-approval-reject","approved":false}`
	req := httptest.NewRequest(http.MethodPost, "/api/tool/approve", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleToolApprove(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	select {
	case approved := <-s.pendingApprovals[approvalID].Response:
		if approved {
			t.Error("Expected approval to be false (rejected)")
		}
	default:
		t.Error("Expected rejection response in channel")
	}
}

func TestToolApprove_InvalidBody(t *testing.T) {
	s := setupAITestServer(t, false)

	req := httptest.NewRequest(http.MethodPost, "/api/tool/approve", strings.NewReader("not json"))
	w := httptest.NewRecorder()

	s.handleToolApprove(w, req)

	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)

	if body["code"] != ErrCodeBadRequest {
		t.Errorf("Expected error code %s, got %v", ErrCodeBadRequest, body["code"])
	}
}

func TestToolApprove_NonExistentID(t *testing.T) {
	s := setupAITestServer(t, false)

	body := `{"id":"does-not-exist","approved":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/tool/approve", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleToolApprove(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["code"] != ErrCodeNotFound {
		t.Errorf("Expected error code %s, got %v", ErrCodeNotFound, resp["code"])
	}
}

func TestToolApprove_AlreadyProcessed(t *testing.T) {
	s := setupAITestServer(t, false)

	approvalID := "test-approval-dup"
	ch := make(chan bool, 1)
	ch <- true // fill the channel so next send would block
	s.pendingApprovals[approvalID] = &PendingToolApproval{
		ID:       approvalID,
		Response: ch,
	}

	body := `{"id":"test-approval-dup","approved":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/tool/approve", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleToolApprove(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["code"] != ErrCodeConflict {
		t.Errorf("Expected error code %s, got %v", ErrCodeConflict, resp["code"])
	}
}

func TestToolApprove_MethodNotAllowed(t *testing.T) {
	s := setupAITestServer(t, false)

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/tool/approve", nil)
			w := httptest.NewRecorder()

			s.handleToolApprove(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

func TestToolApprove_ConcurrentAccess(t *testing.T) {
	s := setupAITestServer(t, false)

	// Create multiple pending approvals
	for i := 0; i < 10; i++ {
		id := "concurrent-" + string(rune('a'+i))
		s.pendingApprovals[id] = &PendingToolApproval{
			ID:       id,
			Response: make(chan bool, 1),
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			id := "concurrent-" + string(rune('a'+idx))
			body := `{"id":"` + id + `","approved":true}`
			req := httptest.NewRequest(http.MethodPost, "/api/tool/approve", strings.NewReader(body))
			w := httptest.NewRecorder()
			s.handleToolApprove(w, req)
		}(i)
	}
	wg.Wait()
}

// ==================== handleSafetyAnalysis Tests ====================

func TestSafetyAnalysis_ReadOnlyCommand(t *testing.T) {
	s := setupAITestServer(t, false)

	body := `{"command":"kubectl get pods -n default"}`
	req := httptest.NewRequest(http.MethodPost, "/api/safety/analyze", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleSafetyAnalysis(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp SafetyAnalysisResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !resp.Safe {
		t.Error("Expected 'kubectl get pods' to be safe")
	}
	if resp.RiskLevel != "safe" {
		t.Errorf("Expected risk_level=safe, got %s", resp.RiskLevel)
	}
	if resp.Category != "read-only" {
		t.Errorf("Expected category=read-only, got %s", resp.Category)
	}
	if resp.RequiresApproval {
		t.Error("Expected read-only command to not require approval")
	}
}

func TestSafetyAnalysis_DangerousDelete(t *testing.T) {
	s := setupAITestServer(t, false)

	body := `{"command":"kubectl delete deployment nginx -n production"}`
	req := httptest.NewRequest(http.MethodPost, "/api/safety/analyze", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleSafetyAnalysis(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp SafetyAnalysisResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Safe {
		t.Error("Expected 'kubectl delete deployment' to be unsafe")
	}
	if resp.RiskLevel != "dangerous" {
		t.Errorf("Expected risk_level=dangerous, got %s", resp.RiskLevel)
	}
	if !resp.RequiresApproval {
		t.Error("Expected dangerous command to require approval")
	}
	if len(resp.Warnings) == 0 {
		t.Error("Expected warnings for dangerous command")
	}
	if len(resp.Recommendations) == 0 {
		t.Error("Expected recommendations for dangerous command")
	}
}

func TestSafetyAnalysis_CriticalNamespaceDelete(t *testing.T) {
	s := setupAITestServer(t, false)

	body := `{"command":"kubectl delete namespace kube-system"}`
	req := httptest.NewRequest(http.MethodPost, "/api/safety/analyze", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleSafetyAnalysis(w, req)

	var resp SafetyAnalysisResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Safe {
		t.Error("Expected namespace deletion to be unsafe")
	}
	if resp.RiskLevel != "critical" {
		t.Errorf("Expected risk_level=critical, got %s", resp.RiskLevel)
	}
	if resp.AffectedScope != "namespace" {
		t.Errorf("Expected affected_scope=namespace, got %s", resp.AffectedScope)
	}
}

func TestSafetyAnalysis_WarningCommand(t *testing.T) {
	s := setupAITestServer(t, false)

	body := `{"command":"kubectl scale deployment nginx --replicas=5"}`
	req := httptest.NewRequest(http.MethodPost, "/api/safety/analyze", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleSafetyAnalysis(w, req)

	var resp SafetyAnalysisResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.RiskLevel != "warning" {
		t.Errorf("Expected risk_level=warning, got %s", resp.RiskLevel)
	}
	if !resp.RequiresApproval {
		t.Error("Expected warning command to require approval")
	}
}

func TestSafetyAnalysis_SensitiveNamespace(t *testing.T) {
	s := setupAITestServer(t, false)

	body := `{"command":"kubectl delete pod test","namespace":"kube-system"}`
	req := httptest.NewRequest(http.MethodPost, "/api/safety/analyze", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleSafetyAnalysis(w, req)

	var resp SafetyAnalysisResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Operating on kube-system should escalate risk
	if resp.RiskLevel == "safe" {
		t.Error("Expected elevated risk for kube-system namespace")
	}

	found := false
	for _, w := range resp.Warnings {
		if strings.Contains(w, "kube-system") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected warning mentioning kube-system")
	}
}

func TestSafetyAnalysis_ProductionIndicator(t *testing.T) {
	s := setupAITestServer(t, false)

	body := `{"command":"kubectl apply -f deploy.yaml","namespace":"production"}`
	req := httptest.NewRequest(http.MethodPost, "/api/safety/analyze", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleSafetyAnalysis(w, req)

	var resp SafetyAnalysisResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if !resp.RequiresApproval {
		t.Error("Expected approval required for production namespace")
	}

	found := false
	for _, w := range resp.Warnings {
		if strings.Contains(w, "production") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected warning about production environment")
	}
}

func TestSafetyAnalysis_EmptyCommand(t *testing.T) {
	s := setupAITestServer(t, false)

	body := `{"command":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/safety/analyze", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleSafetyAnalysis(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["code"] != ErrCodeBadRequest {
		t.Errorf("Expected error code %s, got %v", ErrCodeBadRequest, resp["code"])
	}
}

func TestSafetyAnalysis_InvalidBody(t *testing.T) {
	s := setupAITestServer(t, false)

	req := httptest.NewRequest(http.MethodPost, "/api/safety/analyze", strings.NewReader("not json"))
	w := httptest.NewRecorder()

	s.handleSafetyAnalysis(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["code"] != ErrCodeBadRequest {
		t.Errorf("Expected error code %s, got %v", ErrCodeBadRequest, resp["code"])
	}
}

func TestSafetyAnalysis_MethodNotAllowed(t *testing.T) {
	s := setupAITestServer(t, false)

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/safety/analyze", nil)
			w := httptest.NewRecorder()

			s.handleSafetyAnalysis(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// ==================== handleLLMSettings Tests ====================

func TestLLMSettings_MethodNotAllowed(t *testing.T) {
	s := setupAITestServer(t, false)

	// handleLLMSettings only accepts PUT; other methods should return 405
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/settings/llm", nil)
			w := httptest.NewRecorder()

			s.handleLLMSettings(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

func TestLLMSettings_InvalidBody(t *testing.T) {
	s := setupAITestServer(t, false)

	req := httptest.NewRequest(http.MethodPut, "/api/settings/llm", strings.NewReader("not json"))
	w := httptest.NewRecorder()

	s.handleLLMSettings(w, req)

	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)

	if body["code"] != ErrCodeBadRequest {
		t.Errorf("Expected error code %s, got %v", ErrCodeBadRequest, body["code"])
	}
}

// ==================== handleLLMTest Tests ====================

func TestLLMTest_NilClientWithoutBody(t *testing.T) {
	s := setupAITestServer(t, false)
	s.aiClient = nil

	req := httptest.NewRequest(http.MethodGet, "/api/llm/test", nil)
	w := httptest.NewRecorder()

	s.handleLLMTest(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)

	if body["connected"] != false {
		t.Errorf("Expected connected=false when aiClient is nil, got %v", body["connected"])
	}
	if body["error"] == nil || body["error"] == "" {
		t.Error("Expected error message when aiClient is nil")
	}
}

func TestLLMTest_MethodNotAllowed(t *testing.T) {
	s := setupAITestServer(t, false)

	for _, method := range []string{http.MethodPut, http.MethodDelete, http.MethodPatch} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/llm/test", nil)
			w := httptest.NewRecorder()

			s.handleLLMTest(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// ==================== analyzeK8sSafety Unit Tests ====================

func TestAnalyzeK8sSafety_DescribeCommand(t *testing.T) {
	resp := analyzeK8sSafety(SafetyAnalysisRequest{Command: "kubectl describe pod test-pod"})

	if !resp.Safe {
		t.Error("Expected describe to be safe")
	}
	if resp.RiskLevel != "safe" {
		t.Errorf("Expected risk_level=safe, got %s", resp.RiskLevel)
	}
}

func TestAnalyzeK8sSafety_LogsCommand(t *testing.T) {
	resp := analyzeK8sSafety(SafetyAnalysisRequest{Command: "kubectl logs my-pod -f"})

	if !resp.Safe {
		t.Error("Expected logs to be safe")
	}
}

func TestAnalyzeK8sSafety_DrainNode(t *testing.T) {
	resp := analyzeK8sSafety(SafetyAnalysisRequest{Command: "kubectl drain node worker-1"})

	if resp.Safe {
		t.Error("Expected drain node to be unsafe")
	}
	if resp.RiskLevel != "critical" {
		t.Errorf("Expected risk_level=critical, got %s", resp.RiskLevel)
	}
	if resp.AffectedScope != "cluster" {
		t.Errorf("Expected affected_scope=cluster, got %s", resp.AffectedScope)
	}
}

func TestAnalyzeK8sSafety_ForceDelete(t *testing.T) {
	resp := analyzeK8sSafety(SafetyAnalysisRequest{Command: "kubectl delete pod test --force --grace-period=0"})

	if resp.Safe {
		t.Error("Expected force delete to be unsafe")
	}
	if resp.RiskLevel != "critical" {
		t.Errorf("Expected risk_level=critical for force delete, got %s", resp.RiskLevel)
	}
}

func TestAnalyzeK8sSafety_AllNamespaces(t *testing.T) {
	resp := analyzeK8sSafety(SafetyAnalysisRequest{Command: "kubectl delete pods --all-namespaces"})

	if resp.Safe {
		t.Error("Expected all-namespaces delete to be unsafe")
	}
	if resp.AffectedScope != "cluster" {
		t.Errorf("Expected affected_scope=cluster, got %s", resp.AffectedScope)
	}
}

func TestAnalyzeK8sSafety_ScaleToZero(t *testing.T) {
	// Pattern "scale --replicas=0" matches substring, so use exact format
	resp := analyzeK8sSafety(SafetyAnalysisRequest{Command: "kubectl scale --replicas=0 deployment test"})

	if resp.Safe {
		t.Error("Expected scale to zero to be unsafe")
	}
	if resp.RiskLevel != "dangerous" {
		t.Errorf("Expected risk_level=dangerous for scale --replicas=0, got %s", resp.RiskLevel)
	}
}

func TestAnalyzeK8sSafety_ApplyCommand(t *testing.T) {
	resp := analyzeK8sSafety(SafetyAnalysisRequest{Command: "kubectl apply -f deployment.yaml"})

	if resp.RiskLevel == "safe" {
		t.Error("Expected apply to not be safe")
	}
	if !resp.RequiresApproval {
		t.Error("Expected apply to require approval")
	}
}

// ==================== getLanguageInstruction Tests ====================

func TestAIGetLanguageInstruction(t *testing.T) {
	tests := []struct {
		lang     string
		contains string
		empty    bool
	}{
		{"ko", "Korean", false},
		{"zh", "Chinese", false},
		{"ja", "Japanese", false},
		{"en", "", true},
		{"fr", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			result := getLanguageInstruction(tt.lang)
			if tt.empty && result != "" {
				t.Errorf("Expected empty for lang=%q, got %q", tt.lang, result)
			}
			if !tt.empty && !strings.Contains(result, tt.contains) {
				t.Errorf("Expected result to contain %q for lang=%q, got %q", tt.contains, tt.lang, result)
			}
		})
	}
}

// ==================== getLLMCapabilities Tests ====================

func TestGetLLMCapabilities_NilClient(t *testing.T) {
	caps := getLLMCapabilities(nil, "openai")

	if caps.ToolCalling {
		t.Error("Expected ToolCalling=false for nil client")
	}
	if !caps.Streaming {
		t.Error("Expected Streaming=true")
	}
}

func TestGetLLMCapabilities_Providers(t *testing.T) {
	// When client is nil, getLLMCapabilities returns early with defaults
	// (no JSONMode, no MaxTokens, no Recommendation)
	caps := getLLMCapabilities(nil, "openai")
	if caps.Streaming != true {
		t.Errorf("Expected Streaming=true, got %v", caps.Streaming)
	}
	if caps.ToolCalling != false {
		t.Errorf("Expected ToolCalling=false when client is nil, got %v", caps.ToolCalling)
	}
	// With nil client, JSONMode and MaxTokens are not populated (early return)
	if caps.JSONMode != false {
		t.Errorf("Expected JSONMode=false when client is nil, got %v", caps.JSONMode)
	}
}
