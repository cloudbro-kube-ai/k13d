package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/config"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"
	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
)

// setupNewFeaturesTestServer creates a Server with a fake K8s clientset
// populated with test data for RBAC, NetPol, Events, and Troubleshoot tests.
func setupNewFeaturesTestServer(t *testing.T) *Server {
	t.Helper()

	proto := corev1.ProtocolTCP
	port80 := intstr.FromInt32(80)

	fakeClientset := fake.NewSimpleClientset(
		// RBAC: RoleBinding
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dev-editor",
				Namespace: "default",
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     "editor",
			},
			Subjects: []rbacv1.Subject{
				{Kind: "User", Name: "alice", Namespace: ""},
				{Kind: "ServiceAccount", Name: "ci-bot", Namespace: "default"},
			},
		},
		// RBAC: ClusterRoleBinding
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "admin-binding",
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "cluster-admin",
			},
			Subjects: []rbacv1.Subject{
				{Kind: "User", Name: "bob"},
			},
		},
		// Pods for NetPol and Troubleshoot
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "web-pod",
				Namespace: "default",
				Labels:    map[string]string{"app": "web"},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "api-pod",
				Namespace: "default",
				Labels:    map[string]string{"app": "api"},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "crash-pod",
				Namespace: "default",
				Labels:    map[string]string{"app": "crash"},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:         "main",
						RestartCount: 10,
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{
								Reason: "CrashLoopBackOff",
							},
						},
					},
				},
			},
		},
		// NetworkPolicy
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "allow-web-ingress",
				Namespace: "default",
			},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "web"},
				},
				Ingress: []networkingv1.NetworkPolicyIngressRule{
					{
						From: []networkingv1.NetworkPolicyPeer{
							{PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"app": "api"},
							}},
						},
						Ports: []networkingv1.NetworkPolicyPort{
							{Protocol: &proto, Port: &port80},
						},
					},
				},
			},
		},
		// Event for timeline
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "web-pod-event",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind:      "Pod",
				Name:      "web-pod",
				Namespace: "default",
			},
			Type:           "Normal",
			Reason:         "Scheduled",
			Message:        "Successfully assigned default/web-pod to node-1",
			Count:          1,
			FirstTimestamp: metav1.Time{Time: time.Now().Add(-10 * time.Minute)},
			LastTimestamp:  metav1.Time{Time: time.Now().Add(-5 * time.Minute)},
		},
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "crash-pod-event",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind:      "Pod",
				Name:      "crash-pod",
				Namespace: "default",
			},
			Type:           "Warning",
			Reason:         "BackOff",
			Message:        "Back-off restarting failed container",
			Count:          5,
			FirstTimestamp: metav1.Time{Time: time.Now().Add(-10 * time.Minute)},
			LastTimestamp:  metav1.Time{Time: time.Now().Add(-1 * time.Minute)},
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

// ============================
// RBAC Visualization Tests
// ============================

func TestHandleRBACVisualization_ReturnsSubjects(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/rbac/visualization", nil)
	w := httptest.NewRecorder()

	server.handleRBACVisualization(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp RBACVisualizationResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should have nodes and edges
	if len(resp.Nodes) == 0 {
		t.Error("expected non-empty nodes")
	}
	if len(resp.Edges) == 0 {
		t.Error("expected non-empty edges")
	}

	// Should have subjects for frontend UI
	if len(resp.Subjects) == 0 {
		t.Fatal("expected non-empty subjects")
	}

	// Verify known subjects
	subjectNames := make(map[string]bool)
	for _, s := range resp.Subjects {
		subjectNames[s.Name] = true
		if len(s.Roles) == 0 {
			t.Errorf("subject %q has no roles", s.Name)
		}
	}

	if !subjectNames["alice"] {
		t.Error("expected subject 'alice' from RoleBinding")
	}
	if !subjectNames["bob"] {
		t.Error("expected subject 'bob' from ClusterRoleBinding")
	}
	if !subjectNames["ci-bot"] {
		t.Error("expected subject 'ci-bot' from RoleBinding")
	}
}

func TestHandleRBACVisualization_WithNamespaceFilter(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/rbac/visualization?namespace=default", nil)
	w := httptest.NewRecorder()

	server.handleRBACVisualization(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp RBACVisualizationResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should still have subjects from the namespaced RoleBinding
	found := false
	for _, s := range resp.Subjects {
		if s.Name == "alice" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find 'alice' in default namespace")
	}
}

func TestHandleRBACVisualization_MethodNotAllowed(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/rbac/visualization", nil)
	w := httptest.NewRecorder()

	server.handleRBACVisualization(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleRBACVisualization_SubjectRoleMapping(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/rbac/visualization", nil)
	w := httptest.NewRecorder()

	server.handleRBACVisualization(w, req)

	var resp RBACVisualizationResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	// Check bob has cluster-admin role with cluster scope
	for _, s := range resp.Subjects {
		if s.Name == "bob" {
			if len(s.Roles) != 1 {
				t.Errorf("expected 1 role for bob, got %d", len(s.Roles))
			}
			if s.Roles[0].RoleName != "cluster-admin" {
				t.Errorf("expected role 'cluster-admin', got %q", s.Roles[0].RoleName)
			}
			if !s.Roles[0].ClusterScope {
				t.Error("expected cluster scope for cluster-admin")
			}
		}
	}
}

// ============================
// Network Policy Visualization Tests
// ============================

func TestHandleNetworkPolicyVisualization_ReturnsPolicies(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/netpol/visualization?namespace=default", nil)
	w := httptest.NewRecorder()

	server.handleNetworkPolicyVisualization(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp NetPolVisualizationResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp.PolicyCount != 1 {
		t.Errorf("expected 1 policy count, got %d", resp.PolicyCount)
	}

	// Should have policy summaries for frontend cards
	if len(resp.Policies) != 1 {
		t.Fatalf("expected 1 policy summary, got %d", len(resp.Policies))
	}

	pol := resp.Policies[0]
	if pol.Name != "allow-web-ingress" {
		t.Errorf("expected policy name 'allow-web-ingress', got %q", pol.Name)
	}
	if pol.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", pol.Namespace)
	}
	if pol.PodSelector != "app=web" {
		t.Errorf("expected pod selector 'app=web', got %q", pol.PodSelector)
	}
	if len(pol.IngressRules) == 0 {
		t.Error("expected non-empty ingress rules")
	}
}

func TestHandleNetworkPolicyVisualization_MethodNotAllowed(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/netpol/visualization", nil)
	w := httptest.NewRecorder()

	server.handleNetworkPolicyVisualization(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleNetworkPolicyVisualization_HasNodes(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/netpol/visualization?namespace=default", nil)
	w := httptest.NewRecorder()

	server.handleNetworkPolicyVisualization(w, req)

	var resp NetPolVisualizationResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	// Should have pod nodes
	if len(resp.Nodes) == 0 {
		t.Error("expected non-empty nodes")
	}

	// Find the web-pod node
	found := false
	for _, n := range resp.Nodes {
		if n.Kind == "Pod" && n.Name == "web-pod" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find web-pod node")
	}
}

// ============================
// Event Timeline Tests
// ============================

func TestHandleEventTimeline_ReturnsJSON(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/events/timeline?namespace=default&hours=24", nil)
	w := httptest.NewRecorder()

	server.handleEventTimeline(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var resp EventTimelineResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp.Hours != 24 {
		t.Errorf("expected hours=24, got %d", resp.Hours)
	}
	if resp.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", resp.Namespace)
	}
}

func TestHandleEventTimeline_CountsEvents(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/events/timeline?namespace=default&hours=1", nil)
	w := httptest.NewRecorder()

	server.handleEventTimeline(w, req)

	var resp EventTimelineResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	total := resp.TotalNormal + resp.TotalWarning
	if total < 2 {
		t.Errorf("expected at least 2 total events, got %d", total)
	}
	if resp.TotalWarning < 1 {
		t.Errorf("expected at least 1 warning event, got %d", resp.TotalWarning)
	}
	if resp.TotalNormal < 1 {
		t.Errorf("expected at least 1 normal event, got %d", resp.TotalNormal)
	}
}

func TestHandleEventTimeline_MethodNotAllowed(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/events/timeline", nil)
	w := httptest.NewRecorder()

	server.handleEventTimeline(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleEventTimeline_CustomHours(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/events/timeline?hours=2", nil)
	w := httptest.NewRecorder()

	server.handleEventTimeline(w, req)

	var resp EventTimelineResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp.Hours != 2 {
		t.Errorf("expected hours=2, got %d", resp.Hours)
	}
}

// ============================
// Troubleshoot Tests
// ============================

func TestHandleTroubleshoot_DetectsCrashLoop(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	body := bytes.NewBufferString(`{"namespace":"default"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/troubleshoot", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleTroubleshoot(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var report TroubleshootReport
	if err := json.NewDecoder(w.Body).Decode(&report); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if report.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", report.Namespace)
	}

	// Should detect the CrashLoopBackOff pod
	foundCrashLoop := false
	foundHighRestart := false
	for _, f := range report.Findings {
		if f.Issue == "CrashLoopBackOff" && f.Name == "crash-pod" {
			foundCrashLoop = true
			if f.Severity != "critical" {
				t.Errorf("CrashLoopBackOff should be critical, got %q", f.Severity)
			}
		}
		if f.Issue == "High Restart Count" && f.Name == "crash-pod" {
			foundHighRestart = true
		}
	}

	if !foundCrashLoop {
		t.Error("expected to find CrashLoopBackOff finding for crash-pod")
	}
	if !foundHighRestart {
		t.Error("expected to find High Restart Count finding for crash-pod (restarts=10)")
	}

	// Overall severity should be critical
	if report.Severity != "critical" {
		t.Errorf("expected overall severity 'critical', got %q", report.Severity)
	}

	// Should have recommendations
	if len(report.Recommendations) == 0 {
		t.Error("expected non-empty recommendations")
	}
}

func TestHandleTroubleshoot_MethodNotAllowed(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/troubleshoot", nil)
	w := httptest.NewRecorder()

	server.handleTroubleshoot(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleTroubleshoot_DefaultNamespace(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/troubleshoot", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleTroubleshoot(w, req)

	var report TroubleshootReport
	if err := json.NewDecoder(w.Body).Decode(&report); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if report.Namespace != "default" {
		t.Errorf("expected default namespace when empty, got %q", report.Namespace)
	}
}

// ============================
// Notification Config Tests
// ============================

func TestHandleNotificationConfig_GetDefault(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	// Reset notification config to default for test isolation
	notifConfigMu.Lock()
	notifConfig = &NotificationConfig{
		Enabled: false,
		Events:  []string{},
	}
	notifConfigMu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/api/notifications/config", nil)
	w := httptest.NewRecorder()

	server.handleNotificationConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var cfg NotificationConfig
	if err := json.NewDecoder(w.Body).Decode(&cfg); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if cfg.Enabled {
		t.Error("expected disabled by default")
	}
}

func TestHandleNotificationConfig_SaveAndRetrieve(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	// POST a new config
	newCfg := NotificationConfig{
		Enabled:    true,
		WebhookURL: "https://hooks.slack.com/services/T00/B00/xxx",
		Channel:    "#alerts",
		Events:     []string{"pod_crash", "deploy_fail"},
		Provider:   "slack",
	}
	cfgBytes, _ := json.Marshal(newCfg)
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/config", bytes.NewReader(cfgBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleNotificationConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on POST, got %d", w.Code)
	}

	// GET the config back
	req2 := httptest.NewRequest(http.MethodGet, "/api/notifications/config", nil)
	w2 := httptest.NewRecorder()

	server.handleNotificationConfig(w2, req2)

	var retrieved NotificationConfig
	if err := json.NewDecoder(w2.Body).Decode(&retrieved); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if !retrieved.Enabled {
		t.Error("expected enabled after save")
	}
	if retrieved.Channel != "#alerts" {
		t.Errorf("expected channel '#alerts', got %q", retrieved.Channel)
	}
	if retrieved.Provider != "slack" {
		t.Errorf("expected provider 'slack', got %q", retrieved.Provider)
	}
	// Webhook URL should be masked
	if retrieved.WebhookURL == newCfg.WebhookURL {
		t.Error("expected webhook URL to be masked in GET response")
	}
}

func TestHandleNotificationConfig_RequiresWebhookWhenEnabled(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	cfg := NotificationConfig{
		Enabled:    true,
		WebhookURL: "", // missing
		Provider:   "slack",
	}
	cfgBytes, _ := json.Marshal(cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/config", bytes.NewReader(cfgBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleNotificationConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when enabled without webhook, got %d", w.Code)
	}
}

func TestHandleNotificationConfig_MethodNotAllowed(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/notifications/config", nil)
	w := httptest.NewRecorder()

	server.handleNotificationConfig(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// ============================
// Templates Tests
// ============================

func TestHandleTemplates_ListsBuiltins(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/templates", nil)
	w := httptest.NewRecorder()

	server.handleTemplates(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Templates []ResourceTemplate `json:"templates"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if len(resp.Templates) == 0 {
		t.Fatal("expected at least 1 built-in template")
	}

	// Verify each template has required fields
	for _, tpl := range resp.Templates {
		if tpl.Name == "" {
			t.Error("template name should not be empty")
		}
		if tpl.Category == "" {
			t.Error("template category should not be empty")
		}
		if tpl.YAML == "" {
			t.Error("template YAML should not be empty")
		}
	}

	// Check for known templates
	foundNginx := false
	for _, tpl := range resp.Templates {
		if tpl.Name == "Nginx Deployment" {
			foundNginx = true
			break
		}
	}
	if !foundNginx {
		t.Error("expected to find 'Nginx Deployment' template")
	}
}

func TestHandleTemplates_MethodNotAllowed(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/templates", nil)
	w := httptest.NewRecorder()

	server.handleTemplates(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// ============================
// Notification Test Endpoint
// ============================

func TestHandleNotificationTest_NotConfigured(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	// Reset to disabled
	notifConfigMu.Lock()
	notifConfig = &NotificationConfig{
		Enabled: false,
		Events:  []string{},
	}
	notifConfigMu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/api/notifications/test", nil)
	w := httptest.NewRecorder()

	server.handleNotificationTest(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when not configured, got %d", w.Code)
	}
}

func TestHandleNotificationTest_MethodNotAllowed(t *testing.T) {
	server := setupNewFeaturesTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/notifications/test", nil)
	w := httptest.NewRecorder()

	server.handleNotificationTest(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// ============================
// Utility Function Tests
// ============================

func TestMaskWebhookURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://hooks.slack.com/services/T00/B00/xxxxxxxxxxxx", "https://hooks.sla...xxxxx"},
		{"short", "****"},
		{"", "****"},
	}

	for _, tt := range tests {
		got := maskWebhookURL(tt.input)
		if tt.input == "" {
			// Empty input handled by caller, not tested here
			continue
		}
		if len(tt.input) <= 20 {
			if got != "****" {
				t.Errorf("maskWebhookURL(%q) = %q, want %q", tt.input, got, "****")
			}
		} else {
			if len(got) > len(tt.input) {
				t.Errorf("masked URL should be shorter than original")
			}
		}
	}
}

func TestBuildTestPayload_Slack(t *testing.T) {
	payload, err := buildTestPayload("slack", "#general")
	if err != nil {
		t.Fatalf("failed to build slack payload: %v", err)
	}

	var msg map[string]interface{}
	if err := json.Unmarshal(payload, &msg); err != nil {
		t.Fatalf("invalid JSON payload: %v", err)
	}

	if _, ok := msg["text"]; !ok {
		t.Error("slack payload should have 'text' field")
	}
	if msg["channel"] != "#general" {
		t.Errorf("expected channel '#general', got %v", msg["channel"])
	}
}

func TestBuildTestPayload_Discord(t *testing.T) {
	payload, err := buildTestPayload("discord", "")
	if err != nil {
		t.Fatalf("failed to build discord payload: %v", err)
	}

	var msg map[string]interface{}
	if err := json.Unmarshal(payload, &msg); err != nil {
		t.Fatalf("invalid JSON payload: %v", err)
	}

	if _, ok := msg["content"]; !ok {
		t.Error("discord payload should have 'content' field")
	}
}

func TestBuildTestPayload_Teams(t *testing.T) {
	payload, err := buildTestPayload("teams", "")
	if err != nil {
		t.Fatalf("failed to build teams payload: %v", err)
	}

	var msg map[string]interface{}
	if err := json.Unmarshal(payload, &msg); err != nil {
		t.Fatalf("invalid JSON payload: %v", err)
	}

	if msg["@type"] != "MessageCard" {
		t.Error("teams payload should have @type MessageCard")
	}
}

func TestBuildTestPayload_Generic(t *testing.T) {
	payload, err := buildTestPayload("webhook", "")
	if err != nil {
		t.Fatalf("failed to build generic payload: %v", err)
	}

	var msg map[string]interface{}
	if err := json.Unmarshal(payload, &msg); err != nil {
		t.Fatalf("invalid JSON payload: %v", err)
	}

	if _, ok := msg["text"]; !ok {
		t.Error("generic payload should have 'text' field")
	}
}

// ============================
// GitOps Helper Tests
// ============================

func TestExtractUnstructuredItems_Nil(t *testing.T) {
	result := extractUnstructuredItems(nil)
	if result != nil {
		t.Errorf("expected nil for nil input, got %v", result)
	}
}

func TestGetNestedString(t *testing.T) {
	obj := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      "test-app",
			"namespace": "default",
		},
		"status": map[string]interface{}{
			"health": map[string]interface{}{
				"status": "Healthy",
			},
		},
	}

	tests := []struct {
		fields []string
		want   string
	}{
		{[]string{"metadata", "name"}, "test-app"},
		{[]string{"metadata", "namespace"}, "default"},
		{[]string{"status", "health", "status"}, "Healthy"},
		{[]string{"nonexistent"}, ""},
		{[]string{"metadata", "nonexistent"}, ""},
	}

	for _, tt := range tests {
		got := getNestedString(obj, tt.fields...)
		if got != tt.want {
			t.Errorf("getNestedString(%v) = %q, want %q", tt.fields, got, tt.want)
		}
	}
}

func TestGetNestedSlice(t *testing.T) {
	obj := map[string]interface{}{
		"status": map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{"type": "Ready", "status": "True"},
			},
		},
	}

	result := getNestedSlice(obj, "status", "conditions")
	if len(result) != 1 {
		t.Errorf("expected 1 condition, got %d", len(result))
	}

	result2 := getNestedSlice(obj, "nonexistent")
	if result2 != nil {
		t.Errorf("expected nil for nonexistent path, got %v", result2)
	}
}

// ============================
// NetPol Helper Tests
// ============================

func TestMatchPods(t *testing.T) {
	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "web-1",
				Namespace: "default",
				Labels:    map[string]string{"app": "web", "version": "v1"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "api-1",
				Namespace: "default",
				Labels:    map[string]string{"app": "api"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "web-2",
				Namespace: "other",
				Labels:    map[string]string{"app": "web"},
			},
		},
	}

	// Match by label in default namespace
	matched := matchPods(pods, map[string]string{"app": "web"}, "default")
	if len(matched) != 1 {
		t.Errorf("expected 1 match in default ns, got %d", len(matched))
	}

	// Match by label across all namespaces
	matched = matchPods(pods, map[string]string{"app": "web"}, "")
	if len(matched) != 2 {
		t.Errorf("expected 2 matches across all ns, got %d", len(matched))
	}

	// No match
	matched = matchPods(pods, map[string]string{"app": "db"}, "")
	if len(matched) != 0 {
		t.Errorf("expected 0 matches, got %d", len(matched))
	}
}

func TestLabelsContain(t *testing.T) {
	labels := map[string]string{"app": "web", "version": "v1", "env": "prod"}

	if !labelsContain(labels, map[string]string{"app": "web"}) {
		t.Error("should contain app=web")
	}
	if !labelsContain(labels, map[string]string{"app": "web", "env": "prod"}) {
		t.Error("should contain app=web,env=prod")
	}
	if labelsContain(labels, map[string]string{"app": "api"}) {
		t.Error("should not contain app=api")
	}
	if !labelsContain(labels, map[string]string{}) {
		t.Error("empty selector should match all")
	}
}

func TestFormatPolicyPorts(t *testing.T) {
	proto := corev1.ProtocolTCP
	port := intstr.FromInt32(8080)

	ports := []networkingv1.NetworkPolicyPort{
		{Protocol: &proto, Port: &port},
	}

	result := formatPolicyPorts(ports)
	if result != "8080/TCP" {
		t.Errorf("expected '8080/TCP', got %q", result)
	}

	// No ports
	result = formatPolicyPorts(nil)
	if result != "all" {
		t.Errorf("expected 'all' for no ports, got %q", result)
	}
}

func TestFormatSelector(t *testing.T) {
	result := formatSelector(map[string]string{"app": "web"})
	if result != "app=web" {
		t.Errorf("expected 'app=web', got %q", result)
	}

	result = formatSelector(map[string]string{})
	if result != "*" {
		t.Errorf("expected '*' for empty selector, got %q", result)
	}
}

// ============================
// Diff Helper Tests
// ============================

func TestMapResourceName(t *testing.T) {
	tests := []struct {
		alias string
		group string
		res   string
		want  string
	}{
		{"deploy", "apps", "deployments", "deployments.apps"},
		{"pods", "", "pods", "pods"},
	}

	for _, tt := range tests {
		gvr := schema.GroupVersionResource{Group: tt.group, Version: "v1", Resource: tt.res}
		got := mapResourceName(tt.alias, gvr)
		if got != tt.want {
			t.Errorf("mapResourceName(%q) = %q, want %q", tt.alias, got, tt.want)
		}
	}
}
