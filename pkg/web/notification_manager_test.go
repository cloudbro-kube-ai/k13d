package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/cloudbro-kube-ai/k13d/pkg/mcp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setupNotifTestServer(t *testing.T) *Server {
	t.Helper()
	cfg := config.NewDefaultConfig()
	authConfig := &AuthConfig{
		Enabled:         false,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "admin123",
		Quiet:           true,
	}
	authManager := NewAuthManager(authConfig)
	nm := NewNotificationManager(nil, cfg)
	return &Server{
		cfg:              cfg,
		mcpClient:        mcp.NewClient(),
		authManager:      authManager,
		port:             8080,
		pendingApprovals: make(map[string]*PendingToolApproval),
		notifManager:     nm,
	}
}

func TestClassifyEvent(t *testing.T) {
	tests := []struct {
		name   string
		event  corev1.Event
		expect string
	}{
		{
			name: "CrashLoopBackOff",
			event: corev1.Event{
				Type:   "Warning",
				Reason: "CrashLoopBackOff",
			},
			expect: "pod_crash",
		},
		{
			name: "BackOff",
			event: corev1.Event{
				Type:   "Warning",
				Reason: "BackOff",
			},
			expect: "pod_crash",
		},
		{
			name: "OOMKilled",
			event: corev1.Event{
				Type:   "Warning",
				Reason: "OOMKilled",
			},
			expect: "oom_killed",
		},
		{
			name: "OOMKilling",
			event: corev1.Event{
				Type:   "Warning",
				Reason: "OOMKilling",
			},
			expect: "oom_killed",
		},
		{
			name: "OOM in message",
			event: corev1.Event{
				Type:    "Warning",
				Reason:  "Killing",
				Message: "Container was OOMKilled",
			},
			expect: "oom_killed",
		},
		{
			name: "NodeNotReady",
			event: corev1.Event{
				Type:   "Warning",
				Reason: "NodeNotReady",
			},
			expect: "node_not_ready",
		},
		{
			name: "NodeHasDiskPressure",
			event: corev1.Event{
				Type:   "Warning",
				Reason: "NodeHasDiskPressure",
			},
			expect: "node_not_ready",
		},
		{
			name: "FailedCreate",
			event: corev1.Event{
				Type:   "Warning",
				Reason: "FailedCreate",
			},
			expect: "deploy_fail",
		},
		{
			name: "ImagePullBackOff",
			event: corev1.Event{
				Type:   "Warning",
				Reason: "ImagePullBackOff",
			},
			expect: "image_pull_fail",
		},
		{
			name: "ErrImagePull",
			event: corev1.Event{
				Type:   "Warning",
				Reason: "ErrImagePull",
			},
			expect: "image_pull_fail",
		},
		{
			name: "Normal event ignored",
			event: corev1.Event{
				Type:   "Normal",
				Reason: "Scheduled",
			},
			expect: "",
		},
		{
			name: "Unknown warning",
			event: corev1.Event{
				Type:   "Warning",
				Reason: "SomeOtherReason",
			},
			expect: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyEvent(&tt.event)
			if got != tt.expect {
				t.Errorf("classifyEvent() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestEventHash(t *testing.T) {
	e1 := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
		InvolvedObject: corev1.ObjectReference{
			Name: "my-pod",
			Kind: "Pod",
		},
		Reason: "BackOff",
		Count:  5,
	}
	e2 := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
		InvolvedObject: corev1.ObjectReference{
			Name: "my-pod",
			Kind: "Pod",
		},
		Reason: "BackOff",
		Count:  5,
	}
	e3 := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Namespace: "kube-system"},
		InvolvedObject: corev1.ObjectReference{
			Name: "other-pod",
			Kind: "Pod",
		},
		Reason: "OOMKilled",
		Count:  1,
	}

	h1 := eventHash(e1)
	h2 := eventHash(e2)
	h3 := eventHash(e3)

	if h1 != h2 {
		t.Error("Same events should produce same hash")
	}
	if h1 == h3 {
		t.Error("Different events should produce different hash")
	}
	if len(h1) == 0 {
		t.Error("Hash should not be empty")
	}
}

func TestEventDeduplication(t *testing.T) {
	nm := &NotificationManager{
		sentEvents: make(map[string]time.Time),
	}

	key := "test-hash-123"

	if nm.wasRecentlySent(key) {
		t.Error("Should not be recently sent initially")
	}

	nm.markSent(key)

	if !nm.wasRecentlySent(key) {
		t.Error("Should be recently sent after marking")
	}

	if nm.DedupCount() != 1 {
		t.Errorf("DedupCount() = %d, want 1", nm.DedupCount())
	}
}

func TestEventDeduplicationExpiry(t *testing.T) {
	nm := &NotificationManager{
		sentEvents: make(map[string]time.Time),
	}

	// Manually set an old entry
	nm.sentEventsMu.Lock()
	nm.sentEvents["old-key"] = time.Now().Add(-15 * time.Minute)
	nm.sentEvents["fresh-key"] = time.Now()
	nm.sentEventsMu.Unlock()

	nm.cleanupDedup()

	if nm.DedupCount() != 1 {
		t.Errorf("After cleanup, DedupCount() = %d, want 1 (only fresh-key)", nm.DedupCount())
	}
}

func TestNotificationHistory(t *testing.T) {
	nm := &NotificationManager{
		sentEvents: make(map[string]time.Time),
	}

	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
		InvolvedObject: corev1.ObjectReference{
			Name: "test-pod",
			Kind: "Pod",
		},
		Message: "Container crashed",
	}

	nm.recordHistory(event, "pod_crash", "slack", nil)

	history := nm.GetHistory()
	if len(history) != 1 {
		t.Fatalf("GetHistory() len = %d, want 1", len(history))
	}
	if history[0].EventType != "pod_crash" {
		t.Errorf("EventType = %q, want %q", history[0].EventType, "pod_crash")
	}
	if !history[0].Success {
		t.Error("Success should be true for nil error")
	}
	if history[0].Resource != "Pod/test-pod" {
		t.Errorf("Resource = %q, want %q", history[0].Resource, "Pod/test-pod")
	}
}

func TestNotificationHistoryCap(t *testing.T) {
	nm := &NotificationManager{
		sentEvents: make(map[string]time.Time),
	}

	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
		InvolvedObject: corev1.ObjectReference{
			Name: "test-pod",
			Kind: "Pod",
		},
		Message: "test",
	}

	for i := 0; i < 120; i++ {
		nm.recordHistory(event, "pod_crash", "slack", nil)
	}

	history := nm.GetHistory()
	if len(history) != 100 {
		t.Errorf("GetHistory() len = %d, want 100 (capped)", len(history))
	}
}

func TestNotificationManagerLifecycle(t *testing.T) {
	cfg := config.NewDefaultConfig()
	nm := NewNotificationManager(nil, cfg)

	if nm.IsRunning() {
		t.Error("Should not be running initially")
	}

	nm.Start()
	if !nm.IsRunning() {
		t.Error("Should be running after Start()")
	}

	// Calling Start again should be a no-op
	nm.Start()
	if !nm.IsRunning() {
		t.Error("Should still be running after second Start()")
	}

	nm.Stop()
	if nm.IsRunning() {
		t.Error("Should not be running after Stop()")
	}

	// Calling Stop again should be a no-op
	nm.Stop()
}

func TestDispatchSlack(t *testing.T) {
	var receivedBody map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	nm := &NotificationManager{
		sentEvents: make(map[string]time.Time),
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	ncfg := &config.NotificationsConfig{
		Provider:   "slack",
		WebhookURL: ts.URL,
		Channel:    "#alerts",
	}

	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
		InvolvedObject: corev1.ObjectReference{
			Name: "crash-pod",
			Kind: "Pod",
		},
		Reason:  "CrashLoopBackOff",
		Message: "Container exited with code 1",
	}

	err := nm.sendSlack(ncfg, event, "pod_crash")
	if err != nil {
		t.Fatalf("sendSlack() error: %v", err)
	}
	if receivedBody["channel"] != "#alerts" {
		t.Errorf("channel = %v, want #alerts", receivedBody["channel"])
	}
	text, _ := receivedBody["text"].(string)
	if text == "" {
		t.Error("text should not be empty")
	}
}

func TestDispatchDiscord(t *testing.T) {
	var receivedBody map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	nm := &NotificationManager{
		sentEvents: make(map[string]time.Time),
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	ncfg := &config.NotificationsConfig{
		Provider:   "discord",
		WebhookURL: ts.URL,
	}

	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Namespace: "kube-system"},
		InvolvedObject: corev1.ObjectReference{
			Name: "node-1",
			Kind: "Node",
		},
		Reason:  "NodeNotReady",
		Message: "Node condition Ready is False",
	}

	err := nm.sendDiscord(ncfg, event, "node_not_ready")
	if err != nil {
		t.Fatalf("sendDiscord() error: %v", err)
	}
	content, _ := receivedBody["content"].(string)
	if content == "" {
		t.Error("content should not be empty")
	}
}

func TestDispatchTeams(t *testing.T) {
	var receivedBody map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	nm := &NotificationManager{
		sentEvents: make(map[string]time.Time),
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	ncfg := &config.NotificationsConfig{
		Provider:   "teams",
		WebhookURL: ts.URL,
	}

	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
		InvolvedObject: corev1.ObjectReference{
			Name: "my-deploy",
			Kind: "Deployment",
		},
		Reason:  "FailedCreate",
		Message: "Failed to create pod",
	}

	err := nm.sendTeams(ncfg, event, "deploy_fail")
	if err != nil {
		t.Fatalf("sendTeams() error: %v", err)
	}
	if receivedBody["@type"] != "MessageCard" {
		t.Errorf("@type = %v, want MessageCard", receivedBody["@type"])
	}
}

func TestDispatchCustomWebhook(t *testing.T) {
	var receivedBody map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	nm := &NotificationManager{
		sentEvents: make(map[string]time.Time),
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	ncfg := &config.NotificationsConfig{
		Provider:   "custom",
		WebhookURL: ts.URL,
	}

	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
		InvolvedObject: corev1.ObjectReference{
			Name: "test-pod",
			Kind: "Pod",
		},
		Reason:  "ErrImagePull",
		Message: "Failed to pull image nginx:invalid",
	}

	err := nm.sendCustomWebhook(ncfg, event, "image_pull_fail")
	if err != nil {
		t.Fatalf("sendCustomWebhook() error: %v", err)
	}
	if receivedBody["source"] != "k13d" {
		t.Errorf("source = %v, want k13d", receivedBody["source"])
	}
	if receivedBody["type"] != "image_pull_fail" {
		t.Errorf("type = %v, want image_pull_fail", receivedBody["type"])
	}
}

func TestPostJSONRetry(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	nm := &NotificationManager{
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	err := nm.postJSON(ts.URL, map[string]string{"test": "data"})
	if err != nil {
		t.Fatalf("postJSON() error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (no retry needed)", callCount)
	}
}

func TestPostJSONError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	nm := &NotificationManager{
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	err := nm.postJSON(ts.URL, map[string]string{"test": "data"})
	if err == nil {
		t.Error("postJSON() should return error for 500 status")
	}
}

func TestContainsString(t *testing.T) {
	slice := []string{"pod_crash", "oom_killed", "node_not_ready"}
	if !containsString(slice, "pod_crash") {
		t.Error("Should contain pod_crash")
	}
	if containsString(slice, "deploy_fail") {
		t.Error("Should not contain deploy_fail")
	}
	if containsString(nil, "anything") {
		t.Error("nil slice should not contain anything")
	}
}

func TestTruncate(t *testing.T) {
	if truncate("short", 10) != "short" {
		t.Error("Short string should not be truncated")
	}
	result := truncate("this is a longer string", 10)
	if result != "this is a ..." {
		t.Errorf("truncate() = %q, want %q", result, "this is a ...")
	}
}

func TestHandleNotificationHistory(t *testing.T) {
	s := setupNotifTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/notifications/history", nil)
	w := httptest.NewRecorder()
	s.handleNotificationHistory(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var result []NotificationHistoryEntry
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected empty history, got %d entries", len(result))
	}
}

func TestHandleNotificationHistoryMethodNotAllowed(t *testing.T) {
	s := setupNotifTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/notifications/history", nil)
	w := httptest.NewRecorder()
	s.handleNotificationHistory(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleNotificationStatus(t *testing.T) {
	s := setupNotifTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/notifications/status", nil)
	w := httptest.NewRecorder()
	s.handleNotificationStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	if result["running"] != false {
		t.Error("Expected running=false")
	}
}

func TestHandleNotificationStatusMethodNotAllowed(t *testing.T) {
	s := setupNotifTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/notifications/status", nil)
	w := httptest.NewRecorder()
	s.handleNotificationStatus(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}
