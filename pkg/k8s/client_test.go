package k8s

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

// ============================================================================
// Test Fixtures - Centralized test data creation
// ============================================================================

// testFixtures provides reusable test objects
type testFixtures struct{}

func newFixtures() *testFixtures { return &testFixtures{} }

func (f *testFixtures) pod(name, ns string) *corev1.Pod {
	return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}
}

func (f *testFixtures) node(name string) *corev1.Node {
	return &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: name}}
}

func (f *testFixtures) namespace(name string) *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
}

func (f *testFixtures) deployment(name, ns string) *appsv1.Deployment {
	replicas := int32(1)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
	}
}

func (f *testFixtures) service(name, ns string) *corev1.Service {
	return &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}
}

func (f *testFixtures) configMap(name, ns string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Data:       map[string]string{"key": "value"},
	}
}

func (f *testFixtures) secret(name, ns string) *corev1.Secret {
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}
}

func (f *testFixtures) event(name, ns string) *corev1.Event {
	return &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Type:       "Warning", Reason: "TestReason", Message: "Test message",
	}
}

// ============================================================================
// Interface Contract Tests - Validates Reader interface behavior
// ============================================================================

// TestReaderContract validates that Client implements Reader interface correctly.
// This is the primary test - if interface contract is satisfied, implementation
// details can change without breaking consumers.
func TestReaderContract(t *testing.T) {
	fix := newFixtures()

	// Create client with comprehensive test data
	objects := []runtime.Object{
		fix.pod("pod-1", "default"),
		fix.pod("pod-2", "default"),
		fix.node("node-1"),
		fix.namespace("default"),
		fix.namespace("kube-system"),
		fix.deployment("deploy-1", "default"),
		fix.service("svc-1", "default"),
		fix.configMap("cm-1", "default"),
		fix.secret("secret-1", "default"),
		fix.event("event-1", "default"),
		&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "sts-1", Namespace: "default"}},
		&appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "ds-1", Namespace: "default"}},
		&appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "rs-1", Namespace: "default"}},
		&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "job-1", Namespace: "default"}},
		&batchv1.CronJob{ObjectMeta: metav1.ObjectMeta{Name: "cj-1", Namespace: "default"}},
		&networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: "ing-1", Namespace: "default"}},
		&networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: "np-1", Namespace: "default"}},
		&rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: "role-1", Namespace: "default"}},
		&rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "rb-1", Namespace: "default"}},
		&rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: "cr-1"}},
		&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "crb-1"}},
		&corev1.PersistentVolume{ObjectMeta: metav1.ObjectMeta{Name: "pv-1"}},
		&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "pvc-1", Namespace: "default"}},
		&storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{Name: "sc-1"}},
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "sa-1", Namespace: "default"}},
	}

	client := &Client{Clientset: fake.NewSimpleClientset(objects...)}
	ctx := context.Background()

	// Use Reader interface - this is what consumers should depend on
	var reader Reader = client

	// Table-driven tests for all list operations
	tests := []struct {
		name     string
		listFunc func() (int, error)
		want     int
	}{
		{"ListPods", func() (int, error) { p, e := reader.ListPods(ctx, "default"); return len(p), e }, 2},
		{"ListNodes", func() (int, error) { n, e := reader.ListNodes(ctx); return len(n), e }, 1},
		{"ListNamespaces", func() (int, error) { n, e := reader.ListNamespaces(ctx); return len(n), e }, 2},
		{"ListDeployments", func() (int, error) { d, e := reader.ListDeployments(ctx, "default"); return len(d), e }, 1},
		{"ListServices", func() (int, error) { s, e := reader.ListServices(ctx, "default"); return len(s), e }, 1},
		{"ListConfigMaps", func() (int, error) { c, e := reader.ListConfigMaps(ctx, "default"); return len(c), e }, 1},
		{"ListSecrets", func() (int, error) { s, e := reader.ListSecrets(ctx, "default"); return len(s), e }, 1},
		{"ListEvents", func() (int, error) { e, err := reader.ListEvents(ctx, "default"); return len(e), err }, 1},
		{"ListStatefulSets", func() (int, error) { s, e := reader.ListStatefulSets(ctx, "default"); return len(s), e }, 1},
		{"ListDaemonSets", func() (int, error) { d, e := reader.ListDaemonSets(ctx, "default"); return len(d), e }, 1},
		{"ListReplicaSets", func() (int, error) { r, e := reader.ListReplicaSets(ctx, "default"); return len(r), e }, 1},
		{"ListJobs", func() (int, error) { j, e := reader.ListJobs(ctx, "default"); return len(j), e }, 1},
		{"ListCronJobs", func() (int, error) { c, e := reader.ListCronJobs(ctx, "default"); return len(c), e }, 1},
		{"ListIngresses", func() (int, error) { i, e := reader.ListIngresses(ctx, "default"); return len(i), e }, 1},
		{"ListNetworkPolicies", func() (int, error) { n, e := reader.ListNetworkPolicies(ctx, "default"); return len(n), e }, 1},
		{"ListRoles", func() (int, error) { r, e := reader.ListRoles(ctx, "default"); return len(r), e }, 1},
		{"ListRoleBindings", func() (int, error) { r, e := reader.ListRoleBindings(ctx, "default"); return len(r), e }, 1},
		{"ListClusterRoles", func() (int, error) { c, e := reader.ListClusterRoles(ctx); return len(c), e }, 1},
		{"ListClusterRoleBindings", func() (int, error) { c, e := reader.ListClusterRoleBindings(ctx); return len(c), e }, 1},
		{"ListPersistentVolumes", func() (int, error) { p, e := reader.ListPersistentVolumes(ctx); return len(p), e }, 1},
		{"ListPersistentVolumeClaims", func() (int, error) { p, e := reader.ListPersistentVolumeClaims(ctx, "default"); return len(p), e }, 1},
		{"ListStorageClasses", func() (int, error) { s, e := reader.ListStorageClasses(ctx); return len(s), e }, 1},
		{"ListServiceAccounts", func() (int, error) { s, e := reader.ListServiceAccounts(ctx, "default"); return len(s), e }, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.listFunc()
			if err != nil {
				t.Fatalf("%s failed: %v", tt.name, err)
			}
			if got != tt.want {
				t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
			}
		})
	}
}

// TestReaderEmptyResults validates behavior when no resources exist.
func TestReaderEmptyResults(t *testing.T) {
	client := &Client{Clientset: fake.NewSimpleClientset()}
	ctx := context.Background()
	var reader Reader = client

	// All list operations should succeed with no error when namespace is empty
	pods, err := reader.ListPods(ctx, "default")
	if err != nil {
		t.Fatalf("ListPods failed: %v", err)
	}
	// Note: K8s fake client returns nil slice when empty, which is acceptable Go behavior
	// The important contract is: no error and len() == 0
	if len(pods) != 0 {
		t.Errorf("Expected 0 pods, got %d", len(pods))
	}
}

// TestReaderNamespaceIsolation validates namespace filtering works correctly.
func TestReaderNamespaceIsolation(t *testing.T) {
	fix := newFixtures()
	objects := []runtime.Object{
		fix.pod("pod-default", "default"),
		fix.pod("pod-kube", "kube-system"),
	}
	client := &Client{Clientset: fake.NewSimpleClientset(objects...)}
	ctx := context.Background()
	var reader Reader = client

	defaultPods, _ := reader.ListPods(ctx, "default")
	kubePods, _ := reader.ListPods(ctx, "kube-system")

	if len(defaultPods) != 1 || defaultPods[0].Name != "pod-default" {
		t.Errorf("Expected 1 pod in default, got %d", len(defaultPods))
	}
	if len(kubePods) != 1 || kubePods[0].Name != "pod-kube" {
		t.Errorf("Expected 1 pod in kube-system, got %d", len(kubePods))
	}
}

// ============================================================================
// GVR Mapping Tests - These test implementation-specific behavior
// ============================================================================

func TestGetGVR(t *testing.T) {
	client := &Client{}

	tests := []struct {
		input   string
		wantRes string
		wantOK  bool
	}{
		// Standard resources
		{"pods", "pods", true},
		{"po", "pods", true},
		{"services", "services", true},
		{"svc", "services", true},
		{"deployments", "deployments", true},
		{"deploy", "deployments", true},
		{"statefulsets", "statefulsets", true},
		{"sts", "statefulsets", true},
		{"daemonsets", "daemonsets", true},
		{"ds", "daemonsets", true},
		{"jobs", "jobs", true},
		{"cronjobs", "cronjobs", true},
		{"cj", "cronjobs", true},
		{"configmaps", "configmaps", true},
		{"cm", "configmaps", true},
		{"secrets", "secrets", true},
		{"ingresses", "ingresses", true},
		{"ing", "ingresses", true},
		// RBAC
		{"roles", "roles", true},
		{"rolebindings", "rolebindings", true},
		{"rb", "rolebindings", true},
		{"clusterroles", "clusterroles", true},
		{"clusterrolebindings", "clusterrolebindings", true},
		{"crb", "clusterrolebindings", true},
		// Storage
		{"persistentvolumes", "persistentvolumes", true},
		{"pv", "persistentvolumes", true},
		{"persistentvolumeclaims", "persistentvolumeclaims", true},
		{"pvc", "persistentvolumeclaims", true},
		{"storageclasses", "storageclasses", true},
		{"sc", "storageclasses", true},
		// Other
		{"serviceaccounts", "serviceaccounts", true},
		{"sa", "serviceaccounts", true},
		{"hpa", "horizontalpodautoscalers", true},
		{"networkpolicies", "networkpolicies", true},
		{"netpol", "networkpolicies", true},
		// Invalid
		{"invalid", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gvr, ok := client.GetGVR(tt.input)
			if ok != tt.wantOK {
				t.Errorf("GetGVR(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if tt.wantOK && gvr.Resource != tt.wantRes {
				t.Errorf("GetGVR(%q) = %v, want %v", tt.input, gvr.Resource, tt.wantRes)
			}
		})
	}
}

func TestGetGVR_CaseInsensitive(t *testing.T) {
	client := &Client{}

	tests := []string{"PODS", "Pods", "DeployMents", "CONFIGMAPS"}
	expected := []string{"pods", "pods", "deployments", "configmaps"}

	for i, input := range tests {
		gvr, ok := client.GetGVR(input)
		if !ok || gvr.Resource != expected[i] {
			t.Errorf("GetGVR(%q) = %v, want %v", input, gvr.Resource, expected[i])
		}
	}
}

// ============================================================================
// Options Tests
// ============================================================================

func TestDefaultOptions(t *testing.T) {
	t.Run("GetOptions", func(t *testing.T) {
		opts := DefaultGetOptions()
		if opts.ResourceVersion != "" {
			t.Error("expected empty resource version")
		}
	})

	t.Run("ListOptions", func(t *testing.T) {
		opts := DefaultListOptions()
		if opts.LabelSelector != "" || opts.FieldSelector != "" {
			t.Error("expected empty selectors")
		}
	})
}

// ============================================================================
// Metrics Client Tests
// ============================================================================

func TestMetricsClientRequired(t *testing.T) {
	client := &Client{Metrics: nil}
	ctx := context.Background()

	t.Run("GetPodMetrics", func(t *testing.T) {
		if _, err := client.GetPodMetrics(ctx, "default"); err == nil {
			t.Error("expected error when metrics client is nil")
		}
	})

	t.Run("GetNodeMetrics", func(t *testing.T) {
		if _, err := client.GetNodeMetrics(ctx); err == nil {
			t.Error("expected error when metrics client is nil")
		}
	})
}

// ============================================================================
// CRDInfo Tests
// ============================================================================

func TestCRDInfo(t *testing.T) {
	tests := []struct {
		name       string
		crd        CRDInfo
		wantNs     bool
		wantShorts int
	}{
		{
			name: "namespaced with short names",
			crd: CRDInfo{
				Name: "certificates.cert-manager.io", Group: "cert-manager.io",
				Version: "v1", Kind: "Certificate", Plural: "certificates",
				Namespaced: true, ShortNames: []string{"cert", "certs"},
			},
			wantNs: true, wantShorts: 2,
		},
		{
			name: "cluster-scoped no short names",
			crd: CRDInfo{
				Name: "clusterwidgets.example.com", Group: "example.com",
				Version: "v1", Kind: "ClusterWidget", Plural: "clusterwidgets",
				Namespaced: false, ShortNames: nil,
			},
			wantNs: false, wantShorts: 0,
		},
		{
			name: "empty short names slice",
			crd: CRDInfo{
				Name: "widgets.example.com", Group: "example.com",
				Version: "v1alpha1", Kind: "Widget", Plural: "widgets",
				Namespaced: true, ShortNames: []string{},
			},
			wantNs: true, wantShorts: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.crd.Namespaced != tt.wantNs {
				t.Errorf("Namespaced = %v, want %v", tt.crd.Namespaced, tt.wantNs)
			}
			if len(tt.crd.ShortNames) != tt.wantShorts {
				t.Errorf("ShortNames count = %d, want %d", len(tt.crd.ShortNames), tt.wantShorts)
			}
		})
	}
}

// ============================================================================
// Compile-time interface verification
// ============================================================================

// Ensure Client implements all interfaces
var (
	_ Reader          = (*Client)(nil)
	_ Writer          = (*Client)(nil)
	_ ContextManager  = (*Client)(nil)
	_ ClientInterface = (*Client)(nil)
)
