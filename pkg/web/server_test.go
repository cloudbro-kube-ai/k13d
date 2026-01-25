package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Note: Helper function tests have been moved to helpers_test.go

func TestSSEWriter(t *testing.T) {
	w := httptest.NewRecorder()

	sse := &SSEWriter{
		w:       w,
		flusher: w,
	}

	err := sse.Write("test message")
	if err != nil {
		t.Errorf("SSEWriter.Write() error = %v", err)
	}

	expected := "data: test message\n\n"
	if w.Body.String() != expected {
		t.Errorf("SSEWriter output = %q, want %q", w.Body.String(), expected)
	}
}

func TestCorsMiddleware(t *testing.T) {
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name           string
		method         string
		origin         string
		expectedStatus int
		handlerCalled  bool
		expectCORS     bool
	}{
		{"OPTIONS preflight with allowed origin", http.MethodOptions, "http://localhost:8080", http.StatusOK, false, true},
		{"GET request with allowed origin", http.MethodGet, "http://localhost:8080", http.StatusOK, true, true},
		{"POST request with allowed origin", http.MethodPost, "http://localhost:3000", http.StatusOK, true, true},
		{"GET request without origin", http.MethodGet, "", http.StatusOK, true, false},
		{"GET request with disallowed origin", http.MethodGet, "http://evil.com", http.StatusOK, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled = false
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			w := httptest.NewRecorder()

			corsMiddleware(testHandler).ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.expectedStatus)
			}

			if handlerCalled != tt.handlerCalled {
				t.Errorf("handler called = %v, want %v", handlerCalled, tt.handlerCalled)
			}

			// Check CORS headers
			corsHeader := w.Header().Get("Access-Control-Allow-Origin")
			if tt.expectCORS {
				if corsHeader != tt.origin {
					t.Errorf("Access-Control-Allow-Origin = %q, want %q", corsHeader, tt.origin)
				}
			} else {
				if corsHeader != "" {
					t.Errorf("Access-Control-Allow-Origin should be empty for disallowed origin, got %q", corsHeader)
				}
			}
		})
	}
}

func TestChatRequest_JSON(t *testing.T) {
	req := ChatRequest{Message: "test message"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded ChatRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Message != req.Message {
		t.Errorf("decoded message = %q, want %q", decoded.Message, req.Message)
	}
}

func TestChatResponse_JSON(t *testing.T) {
	resp := ChatResponse{
		Response: "AI response",
		Command:  "kubectl get pods",
		Error:    "",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded ChatResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Response != resp.Response {
		t.Errorf("decoded response = %q, want %q", decoded.Response, resp.Response)
	}
}

func TestK8sResourceResponse_JSON(t *testing.T) {
	resp := K8sResourceResponse{
		Kind: "pods",
		Items: []map[string]interface{}{
			{"name": "test-pod", "namespace": "default"},
		},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded K8sResourceResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Kind != resp.Kind {
		t.Errorf("decoded kind = %q, want %q", decoded.Kind, resp.Kind)
	}

	if len(decoded.Items) != len(resp.Items) {
		t.Errorf("decoded items count = %d, want %d", len(decoded.Items), len(resp.Items))
	}
}

// TestClassifyCommand has been moved to helpers_test.go

func TestSSEWriter_WriteEvent(t *testing.T) {
	w := httptest.NewRecorder()

	sse := &SSEWriter{
		w:       w,
		flusher: w,
	}

	sse.WriteEvent("approval", `{"id":"test123"}`)

	expected := "event: approval\ndata: {\"id\":\"test123\"}\n\n"
	if w.Body.String() != expected {
		t.Errorf("SSEWriter output = %q, want %q", w.Body.String(), expected)
	}
}

func TestHandleAgenticChat_MethodNotAllowed(t *testing.T) {
	// Create a minimal server for testing
	server := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/api/chat/agentic", nil)
	w := httptest.NewRecorder()

	server.handleAgenticChat(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleAgenticChat_InvalidJSON(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(http.MethodPost, "/api/chat/agentic", strings.NewReader("invalid json"))
	w := httptest.NewRecorder()

	server.handleAgenticChat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleAgenticChat_NoAIClient(t *testing.T) {
	server := &Server{
		aiClient: nil,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/chat/agentic", strings.NewReader(`{"message":"hello"}`))
	w := httptest.NewRecorder()

	server.handleAgenticChat(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleToolApprove_MethodNotAllowed(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/api/tool/approve", nil)
	w := httptest.NewRecorder()

	server.handleToolApprove(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleToolApprove_InvalidJSON(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(http.MethodPost, "/api/tool/approve", strings.NewReader("invalid json"))
	w := httptest.NewRecorder()

	server.handleToolApprove(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleToolApprove_ApprovalNotFound(t *testing.T) {
	server := &Server{
		pendingApprovals: make(map[string]*PendingToolApproval),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/tool/approve", strings.NewReader(`{"id":"nonexistent","approved":true}`))
	w := httptest.NewRecorder()

	server.handleToolApprove(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPendingToolApproval_ApproveFlow(t *testing.T) {
	server := &Server{
		pendingApprovals: make(map[string]*PendingToolApproval),
	}

	// Create a pending approval
	approval := &PendingToolApproval{
		ID:        "test_approval_123",
		ToolName:  "kubectl",
		Command:   "kubectl scale deployment nginx --replicas=3",
		Category:  "modifying",
		Timestamp: time.Now(),
		Response:  make(chan bool, 1),
	}

	server.pendingApprovalMutex.Lock()
	server.pendingApprovals["test_approval_123"] = approval
	server.pendingApprovalMutex.Unlock()

	// Approve in goroutine
	go func() {
		req := httptest.NewRequest(http.MethodPost, "/api/tool/approve", strings.NewReader(`{"id":"test_approval_123","approved":true}`))
		w := httptest.NewRecorder()
		server.handleToolApprove(w, req)
	}()

	// Wait for response
	select {
	case approved := <-approval.Response:
		if !approved {
			t.Error("expected approval to be true")
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for approval response")
	}
}

func TestPendingToolApproval_RejectFlow(t *testing.T) {
	server := &Server{
		pendingApprovals: make(map[string]*PendingToolApproval),
	}

	// Create a pending approval
	approval := &PendingToolApproval{
		ID:        "test_approval_456",
		ToolName:  "kubectl",
		Command:   "kubectl delete pod nginx",
		Category:  "dangerous",
		Timestamp: time.Now(),
		Response:  make(chan bool, 1),
	}

	server.pendingApprovalMutex.Lock()
	server.pendingApprovals["test_approval_456"] = approval
	server.pendingApprovalMutex.Unlock()

	// Reject in goroutine
	go func() {
		req := httptest.NewRequest(http.MethodPost, "/api/tool/approve", strings.NewReader(`{"id":"test_approval_456","approved":false}`))
		w := httptest.NewRecorder()
		server.handleToolApprove(w, req)
	}()

	// Wait for response
	select {
	case approved := <-approval.Response:
		if approved {
			t.Error("expected approval to be false")
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for approval response")
	}
}

func TestHandleCustomResources_MethodNotAllowed(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(http.MethodPost, "/api/crd/", nil)
	w := httptest.NewRecorder()

	server.handleCustomResources(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleCustomResources_NoK8sClient(t *testing.T) {
	server := &Server{
		k8sClient: nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/crd/", nil)
	w := httptest.NewRecorder()

	server.handleCustomResources(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// Test ChatRequest with Language field
func TestChatRequest_WithLanguage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMsg  string
		wantLang string
	}{
		{
			name:     "with language",
			input:    `{"message":"hello","language":"ko"}`,
			wantMsg:  "hello",
			wantLang: "ko",
		},
		{
			name:     "without language",
			input:    `{"message":"hello"}`,
			wantMsg:  "hello",
			wantLang: "",
		},
		{
			name:     "with english language",
			input:    `{"message":"test","language":"en"}`,
			wantMsg:  "test",
			wantLang: "en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req ChatRequest
			if err := json.Unmarshal([]byte(tt.input), &req); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if req.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", req.Message, tt.wantMsg)
			}

			if req.Language != tt.wantLang {
				t.Errorf("Language = %q, want %q", req.Language, tt.wantLang)
			}
		})
	}
}

// Test getLanguageInstruction function
func TestGetLanguageInstruction(t *testing.T) {
	tests := []struct {
		name      string
		lang      string
		wantEmpty bool
		contains  string
	}{
		{
			name:      "korean",
			lang:      "ko",
			wantEmpty: false,
			contains:  "Korean",
		},
		{
			name:      "chinese",
			lang:      "zh",
			wantEmpty: false,
			contains:  "Chinese",
		},
		{
			name:      "japanese",
			lang:      "ja",
			wantEmpty: false,
			contains:  "Japanese",
		},
		{
			name:      "english returns empty",
			lang:      "en",
			wantEmpty: true,
			contains:  "",
		},
		{
			name:      "unknown language returns empty",
			lang:      "fr",
			wantEmpty: true,
			contains:  "",
		},
		{
			name:      "empty string returns empty",
			lang:      "",
			wantEmpty: true,
			contains:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getLanguageInstruction(tt.lang)

			if tt.wantEmpty && result != "" {
				t.Errorf("getLanguageInstruction(%q) = %q, want empty", tt.lang, result)
			}

			if !tt.wantEmpty && result == "" {
				t.Errorf("getLanguageInstruction(%q) returned empty, want non-empty", tt.lang)
			}

			if tt.contains != "" && !strings.Contains(result, tt.contains) {
				t.Errorf("getLanguageInstruction(%q) = %q, want to contain %q", tt.lang, result, tt.contains)
			}
		})
	}
}

// Test handleAIPing endpoint
func TestHandleAIPing_NoAIClient(t *testing.T) {
	server := &Server{
		aiClient: nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/ai/ping", nil)
	w := httptest.NewRecorder()

	server.handleAIPing(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

// Test VersionInfo struct
func TestVersionInfo_JSON(t *testing.T) {
	info := VersionInfo{
		Version:   "v0.6.0",
		BuildTime: "2025-01-23T12:00:00Z",
		GitCommit: "abc1234def5678",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded VersionInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Version != info.Version {
		t.Errorf("decoded version = %q, want %q", decoded.Version, info.Version)
	}
	if decoded.BuildTime != info.BuildTime {
		t.Errorf("decoded build_time = %q, want %q", decoded.BuildTime, info.BuildTime)
	}
	if decoded.GitCommit != info.GitCommit {
		t.Errorf("decoded git_commit = %q, want %q", decoded.GitCommit, info.GitCommit)
	}
}

// Test handleVersion endpoint with version info
func TestHandleVersion_WithVersionInfo(t *testing.T) {
	server := &Server{
		versionInfo: &VersionInfo{
			Version:   "v0.6.0",
			BuildTime: "2025-01-23T12:00:00Z",
			GitCommit: "abc1234def5678",
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
	w := httptest.NewRecorder()

	server.handleVersion(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if response["version"] != "v0.6.0" {
		t.Errorf("version = %v, want v0.6.0", response["version"])
	}
	if response["build_time"] != "2025-01-23T12:00:00Z" {
		t.Errorf("build_time = %v, want 2025-01-23T12:00:00Z", response["build_time"])
	}
	if response["git_commit"] != "abc1234def5678" {
		t.Errorf("git_commit = %v, want abc1234def5678", response["git_commit"])
	}
}

// Test handleVersion endpoint without version info (dev build)
func TestHandleVersion_DevBuild(t *testing.T) {
	server := &Server{
		versionInfo: nil,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
	w := httptest.NewRecorder()

	server.handleVersion(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if response["version"] != "dev" {
		t.Errorf("version = %v, want dev", response["version"])
	}
	if response["build_time"] != "unknown" {
		t.Errorf("build_time = %v, want unknown", response["build_time"])
	}
	if response["git_commit"] != "unknown" {
		t.Errorf("git_commit = %v, want unknown", response["git_commit"])
	}
}

// Test handleHealth includes version
func TestHandleHealth_IncludesVersion(t *testing.T) {
	authConfig := &AuthConfig{
		Enabled:  false,
		AuthMode: "local",
		Quiet:    true,
	}

	server := &Server{
		versionInfo: &VersionInfo{
			Version:   "v0.6.0",
			BuildTime: "2025-01-23T12:00:00Z",
			GitCommit: "abc1234",
		},
		authManager: NewAuthManager(authConfig),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if response["version"] != "v0.6.0" {
		t.Errorf("version = %v, want v0.6.0", response["version"])
	}
	if response["status"] != "ok" {
		t.Errorf("status = %v, want ok", response["status"])
	}
}

// Test handleHealth with dev version
func TestHandleHealth_DevVersion(t *testing.T) {
	authConfig := &AuthConfig{
		Enabled:  false,
		AuthMode: "local",
		Quiet:    true,
	}

	server := &Server{
		versionInfo: nil,
		authManager: NewAuthManager(authConfig),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if response["version"] != "dev" {
		t.Errorf("version = %v, want dev", response["version"])
	}
}
