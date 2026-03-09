package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func setupApplicationsTestServer(t *testing.T) *Server {
	t.Helper()

	one := int32(1)

	fakeClientset := fake.NewClientset( //nolint:staticcheck
		// Labeled deployment
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx",
				Namespace: "default",
				Labels:    map[string]string{"app.kubernetes.io/name": "nginx", "app.kubernetes.io/version": "1.21"},
			},
			Spec: appsv1.DeploymentSpec{Replicas: &one},
		},
		// Labeled service
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-svc",
				Namespace: "default",
				Labels:    map[string]string{"app.kubernetes.io/name": "nginx"},
			},
		},
		// Labeled pod (running and ready)
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-pod-1",
				Namespace: "default",
				Labels:    map[string]string{"app.kubernetes.io/name": "nginx"},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{Ready: true},
				},
			},
		},
		// Unlabeled configmap (should be ungrouped)
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-root-ca.crt",
				Namespace: "default",
			},
		},
		// Labeled configmap for nginx
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-config",
				Namespace: "default",
				Labels:    map[string]string{"app.kubernetes.io/name": "nginx"},
			},
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

func TestHandleApplications_ReturnsJSON(t *testing.T) {
	server := setupApplicationsTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/applications?namespace=default", nil)
	w := httptest.NewRecorder()

	server.handleApplications(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var groups []ApplicationGroup
	if err := json.NewDecoder(w.Body).Decode(&groups); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should have nginx group + ungrouped
	if len(groups) < 1 {
		t.Fatalf("expected at least 1 group, got %d", len(groups))
	}

	// Find nginx group
	var nginx *ApplicationGroup
	for i := range groups {
		if groups[i].Name == "nginx" {
			nginx = &groups[i]
			break
		}
	}
	if nginx == nil {
		t.Fatal("expected to find 'nginx' group")
	}

	if nginx.Version != "1.21" {
		t.Errorf("expected version '1.21', got %q", nginx.Version)
	}
	if nginx.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %q", nginx.Status)
	}
	if nginx.PodCount != 1 {
		t.Errorf("expected 1 pod, got %d", nginx.PodCount)
	}
	if nginx.ReadyPods != 1 {
		t.Errorf("expected 1 ready pod, got %d", nginx.ReadyPods)
	}
	if len(nginx.Resources["Deployment"]) != 1 {
		t.Errorf("expected 1 deployment resource, got %d", len(nginx.Resources["Deployment"]))
	}
	if len(nginx.Resources["Service"]) != 1 {
		t.Errorf("expected 1 service resource, got %d", len(nginx.Resources["Service"]))
	}
	if len(nginx.Resources["ConfigMap"]) != 1 {
		t.Errorf("expected 1 configmap resource, got %d", len(nginx.Resources["ConfigMap"]))
	}
}

func TestHandleApplications_MethodNotAllowed(t *testing.T) {
	server := setupApplicationsTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/applications", nil)
	w := httptest.NewRecorder()

	server.handleApplications(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestHandleApplications_EmptyNamespace(t *testing.T) {
	one := int32(1)
	fakeClientset := fake.NewClientset( //nolint:staticcheck
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "app1",
				Namespace: "ns1",
				Labels:    map[string]string{"app.kubernetes.io/name": "app1"},
			},
			Spec: appsv1.DeploymentSpec{Replicas: &one},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "app2",
				Namespace: "ns2",
				Labels:    map[string]string{"app.kubernetes.io/name": "app2"},
			},
			Spec: appsv1.DeploymentSpec{Replicas: &one},
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

	server := &Server{
		cfg:         cfg,
		k8sClient:   &k8s.Client{Clientset: fakeClientset},
		authManager: authManager,
	}

	// Request without namespace filter (all namespaces)
	req := httptest.NewRequest(http.MethodGet, "/api/applications", nil)
	w := httptest.NewRecorder()

	server.handleApplications(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var groups []ApplicationGroup
	if err := json.NewDecoder(w.Body).Decode(&groups); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should see both apps across namespaces
	if len(groups) < 2 {
		t.Errorf("expected at least 2 groups for all namespaces, got %d", len(groups))
	}
}

func TestHandleApplications_UngroupedResources(t *testing.T) {
	fakeClientset := fake.NewClientset( //nolint:staticcheck
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "orphan-cm",
				Namespace: "default",
			},
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

	server := &Server{
		cfg:         cfg,
		k8sClient:   &k8s.Client{Clientset: fakeClientset},
		authManager: authManager,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/applications?namespace=default", nil)
	w := httptest.NewRecorder()

	server.handleApplications(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var groups []ApplicationGroup
	if err := json.NewDecoder(w.Body).Decode(&groups); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should have ungrouped
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Name != "Ungrouped" {
		t.Errorf("expected 'Ungrouped', got %q", groups[0].Name)
	}
	if len(groups[0].Resources["ConfigMap"]) != 1 {
		t.Errorf("expected 1 ungrouped configmap")
	}
}

func TestHandleApplications_MultiResourceTypes(t *testing.T) {
	one := int32(1)
	appLabel := map[string]string{"app.kubernetes.io/name": "myapp"}

	fakeClientset := fake.NewClientset( //nolint:staticcheck
		// StatefulSet
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "myapp-db",
				Namespace: "default",
				Labels:    appLabel,
			},
			Spec: appsv1.StatefulSetSpec{Replicas: &one},
		},
		// DaemonSet
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "myapp-agent",
				Namespace: "default",
				Labels:    appLabel,
			},
		},
		// Ingress
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "myapp-ingress",
				Namespace: "default",
				Labels:    appLabel,
			},
		},
		// Service
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "myapp-svc",
				Namespace: "default",
				Labels:    appLabel,
			},
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

	server := &Server{
		cfg:         cfg,
		k8sClient:   &k8s.Client{Clientset: fakeClientset},
		authManager: authManager,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/applications?namespace=default", nil)
	w := httptest.NewRecorder()

	server.handleApplications(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var groups []ApplicationGroup
	if err := json.NewDecoder(w.Body).Decode(&groups); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Find myapp group
	var myapp *ApplicationGroup
	for i := range groups {
		if groups[i].Name == "myapp" {
			myapp = &groups[i]
			break
		}
	}
	if myapp == nil {
		t.Fatal("expected to find 'myapp' group")
	}

	// Verify all four resource types are grouped under myapp
	if len(myapp.Resources["StatefulSet"]) != 1 {
		t.Errorf("expected 1 StatefulSet, got %d", len(myapp.Resources["StatefulSet"]))
	}
	if len(myapp.Resources["DaemonSet"]) != 1 {
		t.Errorf("expected 1 DaemonSet, got %d", len(myapp.Resources["DaemonSet"]))
	}
	if len(myapp.Resources["Ingress"]) != 1 {
		t.Errorf("expected 1 Ingress, got %d", len(myapp.Resources["Ingress"]))
	}
	if len(myapp.Resources["Service"]) != 1 {
		t.Errorf("expected 1 Service, got %d", len(myapp.Resources["Service"]))
	}

	// Verify specific resource names
	if myapp.Resources["StatefulSet"][0].Name != "myapp-db" {
		t.Errorf("expected StatefulSet name 'myapp-db', got %q", myapp.Resources["StatefulSet"][0].Name)
	}
	if myapp.Resources["DaemonSet"][0].Name != "myapp-agent" {
		t.Errorf("expected DaemonSet name 'myapp-agent', got %q", myapp.Resources["DaemonSet"][0].Name)
	}
	if myapp.Resources["Ingress"][0].Name != "myapp-ingress" {
		t.Errorf("expected Ingress name 'myapp-ingress', got %q", myapp.Resources["Ingress"][0].Name)
	}
	if myapp.Resources["Service"][0].Name != "myapp-svc" {
		t.Errorf("expected Service name 'myapp-svc', got %q", myapp.Resources["Service"][0].Name)
	}
}

func TestHandleApplications_HealthStatus(t *testing.T) {
	one := int32(1)

	fakeClientset := fake.NewClientset( //nolint:staticcheck
		// --- "healthy-app": all pods running and ready ---
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "healthy-deploy",
				Namespace: "default",
				Labels:    map[string]string{"app.kubernetes.io/name": "healthy-app"},
			},
			Spec: appsv1.DeploymentSpec{Replicas: &one},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "healthy-pod-1",
				Namespace: "default",
				Labels:    map[string]string{"app.kubernetes.io/name": "healthy-app"},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{Ready: true},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "healthy-pod-2",
				Namespace: "default",
				Labels:    map[string]string{"app.kubernetes.io/name": "healthy-app"},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{Ready: true},
				},
			},
		},

		// --- "degraded-app": some pods ready, some not ---
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "degraded-deploy",
				Namespace: "default",
				Labels:    map[string]string{"app.kubernetes.io/name": "degraded-app"},
			},
			Spec: appsv1.DeploymentSpec{Replicas: &one},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "degraded-pod-ready",
				Namespace: "default",
				Labels:    map[string]string{"app.kubernetes.io/name": "degraded-app"},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{Ready: true},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "degraded-pod-notready",
				Namespace: "default",
				Labels:    map[string]string{"app.kubernetes.io/name": "degraded-app"},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{Ready: false},
				},
			},
		},

		// --- "failing-app": no pods ready (CrashLoopBackOff) ---
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failing-deploy",
				Namespace: "default",
				Labels:    map[string]string{"app.kubernetes.io/name": "failing-app"},
			},
			Spec: appsv1.DeploymentSpec{Replicas: &one},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failing-pod-1",
				Namespace: "default",
				Labels:    map[string]string{"app.kubernetes.io/name": "failing-app"},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Ready: false,
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{
								Reason: "CrashLoopBackOff",
							},
						},
					},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failing-pod-2",
				Namespace: "default",
				Labels:    map[string]string{"app.kubernetes.io/name": "failing-app"},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodPending,
				ContainerStatuses: []corev1.ContainerStatus{
					{Ready: false},
				},
			},
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

	server := &Server{
		cfg:         cfg,
		k8sClient:   &k8s.Client{Clientset: fakeClientset},
		authManager: authManager,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/applications?namespace=default", nil)
	w := httptest.NewRecorder()

	server.handleApplications(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var groups []ApplicationGroup
	if err := json.NewDecoder(w.Body).Decode(&groups); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Build lookup map
	groupByName := make(map[string]*ApplicationGroup)
	for i := range groups {
		groupByName[groups[i].Name] = &groups[i]
	}

	// Verify healthy-app: all pods ready → "healthy"
	healthy := groupByName["healthy-app"]
	if healthy == nil {
		t.Fatal("expected to find 'healthy-app' group")
	}
	if healthy.Status != "healthy" {
		t.Errorf("healthy-app: expected status 'healthy', got %q", healthy.Status)
	}
	if healthy.PodCount != 2 {
		t.Errorf("healthy-app: expected 2 pods, got %d", healthy.PodCount)
	}
	if healthy.ReadyPods != 2 {
		t.Errorf("healthy-app: expected 2 ready pods, got %d", healthy.ReadyPods)
	}

	// Verify degraded-app: some pods ready → "degraded"
	degraded := groupByName["degraded-app"]
	if degraded == nil {
		t.Fatal("expected to find 'degraded-app' group")
	}
	if degraded.Status != "degraded" {
		t.Errorf("degraded-app: expected status 'degraded', got %q", degraded.Status)
	}
	if degraded.PodCount != 2 {
		t.Errorf("degraded-app: expected 2 pods, got %d", degraded.PodCount)
	}
	if degraded.ReadyPods != 1 {
		t.Errorf("degraded-app: expected 1 ready pod, got %d", degraded.ReadyPods)
	}

	// Verify failing-app: no pods ready → "failing"
	failing := groupByName["failing-app"]
	if failing == nil {
		t.Fatal("expected to find 'failing-app' group")
	}
	if failing.Status != "failing" {
		t.Errorf("failing-app: expected status 'failing', got %q", failing.Status)
	}
	if failing.PodCount != 2 {
		t.Errorf("failing-app: expected 2 pods, got %d", failing.PodCount)
	}
	if failing.ReadyPods != 0 {
		t.Errorf("failing-app: expected 0 ready pods, got %d", failing.ReadyPods)
	}
}

func TestHandleApplications_NamespaceFilter(t *testing.T) {
	one := int32(1)

	fakeClientset := fake.NewClientset( //nolint:staticcheck
		// App in "production" namespace
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "frontend",
				Namespace: "production",
				Labels:    map[string]string{"app.kubernetes.io/name": "frontend"},
			},
			Spec: appsv1.DeploymentSpec{Replicas: &one},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "frontend-svc",
				Namespace: "production",
				Labels:    map[string]string{"app.kubernetes.io/name": "frontend"},
			},
		},
		// App in "staging" namespace
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "backend",
				Namespace: "staging",
				Labels:    map[string]string{"app.kubernetes.io/name": "backend"},
			},
			Spec: appsv1.DeploymentSpec{Replicas: &one},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "backend-svc",
				Namespace: "staging",
				Labels:    map[string]string{"app.kubernetes.io/name": "backend"},
			},
		},
		// Another app in "production" namespace
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "api-gateway",
				Namespace: "production",
				Labels:    map[string]string{"app.kubernetes.io/name": "api-gateway"},
			},
			Spec: appsv1.DeploymentSpec{Replicas: &one},
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

	server := &Server{
		cfg:         cfg,
		k8sClient:   &k8s.Client{Clientset: fakeClientset},
		authManager: authManager,
	}

	// --- Filter to "production" namespace ---
	req := httptest.NewRequest(http.MethodGet, "/api/applications?namespace=production", nil)
	w := httptest.NewRecorder()

	server.handleApplications(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var prodGroups []ApplicationGroup
	if err := json.NewDecoder(w.Body).Decode(&prodGroups); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Build name set for production groups
	prodNames := make(map[string]bool)
	for _, g := range prodGroups {
		prodNames[g.Name] = true
	}

	if !prodNames["frontend"] {
		t.Error("expected 'frontend' in production namespace results")
	}
	if !prodNames["api-gateway"] {
		t.Error("expected 'api-gateway' in production namespace results")
	}
	if prodNames["backend"] {
		t.Error("'backend' should NOT appear in production namespace results")
	}

	// --- Filter to "staging" namespace ---
	req2 := httptest.NewRequest(http.MethodGet, "/api/applications?namespace=staging", nil)
	w2 := httptest.NewRecorder()

	server.handleApplications(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w2.Code)
	}

	var stagingGroups []ApplicationGroup
	if err := json.NewDecoder(w2.Body).Decode(&stagingGroups); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	stagingNames := make(map[string]bool)
	for _, g := range stagingGroups {
		stagingNames[g.Name] = true
	}

	if !stagingNames["backend"] {
		t.Error("expected 'backend' in staging namespace results")
	}
	if stagingNames["frontend"] {
		t.Error("'frontend' should NOT appear in staging namespace results")
	}
	if stagingNames["api-gateway"] {
		t.Error("'api-gateway' should NOT appear in staging namespace results")
	}

	// Verify resource namespaces are correct
	for _, g := range stagingGroups {
		if g.Name == "Ungrouped" {
			continue
		}
		for kind, refs := range g.Resources {
			for _, ref := range refs {
				if ref.Namespace != "staging" {
					t.Errorf("resource %s/%s has namespace %q, expected 'staging'", kind, ref.Name, ref.Namespace)
				}
			}
		}
	}
}
