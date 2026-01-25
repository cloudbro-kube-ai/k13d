package ui

import (
	"context"
	"testing"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestTUIK8sClientInterface verifies all K8s methods used by TUI exist and work
// This ensures TUI and k8s.Client interfaces stay in sync
func TestTUIK8sClientInterface(t *testing.T) {
	// Create fake clientset with test data
	fakeClientset := fake.NewSimpleClientset(
		// Namespaces
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},

		// Pods
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "default"},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		},

		// Nodes
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "test-node"}},

		// Deployments
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "test-deploy", Namespace: "default"},
		},

		// Services
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "test-svc", Namespace: "default"},
		},

		// Events
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{Name: "test-event", Namespace: "default"},
		},

		// ConfigMaps
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "test-cm", Namespace: "default"},
		},

		// Secrets
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "test-secret", Namespace: "default"},
		},

		// PersistentVolumes
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{Name: "test-pv"},
		},

		// PersistentVolumeClaims
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "test-pvc", Namespace: "default"},
		},

		// StorageClasses
		&storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{Name: "test-sc"},
		},

		// ReplicaSets
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{Name: "test-rs", Namespace: "default"},
		},

		// DaemonSets
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: "test-ds", Namespace: "default"},
		},

		// StatefulSets
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "test-sts", Namespace: "default"},
		},

		// Jobs
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Name: "test-job", Namespace: "default"},
		},

		// CronJobs
		&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{Name: "test-cj", Namespace: "default"},
		},

		// Ingresses
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: "test-ing", Namespace: "default"},
		},

		// Endpoints
		&corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{Name: "test-ep", Namespace: "default"},
		},

		// NetworkPolicies
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: "test-netpol", Namespace: "default"},
		},

		// ServiceAccounts
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "test-sa", Namespace: "default"},
		},

		// Roles
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: "test-role", Namespace: "default"},
		},

		// RoleBindings
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "test-rb", Namespace: "default"},
		},

		// ClusterRoles
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: "test-cr"},
		},

		// ClusterRoleBindings
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "test-crb"},
		},
	)

	k8sClient := &k8s.Client{
		Clientset: fakeClientset,
	}

	ctx := context.Background()

	// Test all methods used by TUI (from app_fetch.go)
	t.Run("ListNamespaces", func(t *testing.T) {
		nss, err := k8sClient.ListNamespaces(ctx)
		if err != nil {
			t.Fatalf("ListNamespaces failed: %v", err)
		}
		if len(nss) != 2 {
			t.Errorf("Expected 2 namespaces, got %d", len(nss))
		}
	})

	t.Run("ListPods", func(t *testing.T) {
		pods, err := k8sClient.ListPods(ctx, "default")
		if err != nil {
			t.Fatalf("ListPods failed: %v", err)
		}
		if len(pods) != 1 {
			t.Errorf("Expected 1 pod, got %d", len(pods))
		}
	})

	t.Run("ListNodes", func(t *testing.T) {
		nodes, err := k8sClient.ListNodes(ctx)
		if err != nil {
			t.Fatalf("ListNodes failed: %v", err)
		}
		if len(nodes) != 1 {
			t.Errorf("Expected 1 node, got %d", len(nodes))
		}
	})

	t.Run("ListDeployments", func(t *testing.T) {
		deps, err := k8sClient.ListDeployments(ctx, "default")
		if err != nil {
			t.Fatalf("ListDeployments failed: %v", err)
		}
		if len(deps) != 1 {
			t.Errorf("Expected 1 deployment, got %d", len(deps))
		}
	})

	t.Run("ListServices", func(t *testing.T) {
		svcs, err := k8sClient.ListServices(ctx, "default")
		if err != nil {
			t.Fatalf("ListServices failed: %v", err)
		}
		if len(svcs) != 1 {
			t.Errorf("Expected 1 service, got %d", len(svcs))
		}
	})

	t.Run("ListEvents", func(t *testing.T) {
		events, err := k8sClient.ListEvents(ctx, "default")
		if err != nil {
			t.Fatalf("ListEvents failed: %v", err)
		}
		if len(events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(events))
		}
	})

	t.Run("ListConfigMaps", func(t *testing.T) {
		cms, err := k8sClient.ListConfigMaps(ctx, "default")
		if err != nil {
			t.Fatalf("ListConfigMaps failed: %v", err)
		}
		if len(cms) != 1 {
			t.Errorf("Expected 1 configmap, got %d", len(cms))
		}
	})

	t.Run("ListSecrets", func(t *testing.T) {
		secrets, err := k8sClient.ListSecrets(ctx, "default")
		if err != nil {
			t.Fatalf("ListSecrets failed: %v", err)
		}
		if len(secrets) != 1 {
			t.Errorf("Expected 1 secret, got %d", len(secrets))
		}
	})

	t.Run("ListPersistentVolumes", func(t *testing.T) {
		pvs, err := k8sClient.ListPersistentVolumes(ctx)
		if err != nil {
			t.Fatalf("ListPersistentVolumes failed: %v", err)
		}
		if len(pvs) != 1 {
			t.Errorf("Expected 1 PV, got %d", len(pvs))
		}
	})

	t.Run("ListPersistentVolumeClaims", func(t *testing.T) {
		pvcs, err := k8sClient.ListPersistentVolumeClaims(ctx, "default")
		if err != nil {
			t.Fatalf("ListPersistentVolumeClaims failed: %v", err)
		}
		if len(pvcs) != 1 {
			t.Errorf("Expected 1 PVC, got %d", len(pvcs))
		}
	})

	t.Run("ListStorageClasses", func(t *testing.T) {
		scs, err := k8sClient.ListStorageClasses(ctx)
		if err != nil {
			t.Fatalf("ListStorageClasses failed: %v", err)
		}
		if len(scs) != 1 {
			t.Errorf("Expected 1 storage class, got %d", len(scs))
		}
	})

	t.Run("ListReplicaSets", func(t *testing.T) {
		rss, err := k8sClient.ListReplicaSets(ctx, "default")
		if err != nil {
			t.Fatalf("ListReplicaSets failed: %v", err)
		}
		if len(rss) != 1 {
			t.Errorf("Expected 1 replicaset, got %d", len(rss))
		}
	})

	t.Run("ListDaemonSets", func(t *testing.T) {
		dss, err := k8sClient.ListDaemonSets(ctx, "default")
		if err != nil {
			t.Fatalf("ListDaemonSets failed: %v", err)
		}
		if len(dss) != 1 {
			t.Errorf("Expected 1 daemonset, got %d", len(dss))
		}
	})

	t.Run("ListStatefulSets", func(t *testing.T) {
		stss, err := k8sClient.ListStatefulSets(ctx, "default")
		if err != nil {
			t.Fatalf("ListStatefulSets failed: %v", err)
		}
		if len(stss) != 1 {
			t.Errorf("Expected 1 statefulset, got %d", len(stss))
		}
	})

	t.Run("ListJobs", func(t *testing.T) {
		jobs, err := k8sClient.ListJobs(ctx, "default")
		if err != nil {
			t.Fatalf("ListJobs failed: %v", err)
		}
		if len(jobs) != 1 {
			t.Errorf("Expected 1 job, got %d", len(jobs))
		}
	})

	t.Run("ListCronJobs", func(t *testing.T) {
		cjs, err := k8sClient.ListCronJobs(ctx, "default")
		if err != nil {
			t.Fatalf("ListCronJobs failed: %v", err)
		}
		if len(cjs) != 1 {
			t.Errorf("Expected 1 cronjob, got %d", len(cjs))
		}
	})

	t.Run("ListIngresses", func(t *testing.T) {
		ings, err := k8sClient.ListIngresses(ctx, "default")
		if err != nil {
			t.Fatalf("ListIngresses failed: %v", err)
		}
		if len(ings) != 1 {
			t.Errorf("Expected 1 ingress, got %d", len(ings))
		}
	})

	t.Run("ListEndpoints", func(t *testing.T) {
		eps, err := k8sClient.ListEndpoints(ctx, "default")
		if err != nil {
			t.Fatalf("ListEndpoints failed: %v", err)
		}
		if len(eps) != 1 {
			t.Errorf("Expected 1 endpoint, got %d", len(eps))
		}
	})

	t.Run("ListNetworkPolicies", func(t *testing.T) {
		netpols, err := k8sClient.ListNetworkPolicies(ctx, "default")
		if err != nil {
			t.Fatalf("ListNetworkPolicies failed: %v", err)
		}
		if len(netpols) != 1 {
			t.Errorf("Expected 1 network policy, got %d", len(netpols))
		}
	})

	t.Run("ListServiceAccounts", func(t *testing.T) {
		sas, err := k8sClient.ListServiceAccounts(ctx, "default")
		if err != nil {
			t.Fatalf("ListServiceAccounts failed: %v", err)
		}
		if len(sas) != 1 {
			t.Errorf("Expected 1 service account, got %d", len(sas))
		}
	})

	t.Run("ListRoles", func(t *testing.T) {
		roles, err := k8sClient.ListRoles(ctx, "default")
		if err != nil {
			t.Fatalf("ListRoles failed: %v", err)
		}
		if len(roles) != 1 {
			t.Errorf("Expected 1 role, got %d", len(roles))
		}
	})

	t.Run("ListRoleBindings", func(t *testing.T) {
		rbs, err := k8sClient.ListRoleBindings(ctx, "default")
		if err != nil {
			t.Fatalf("ListRoleBindings failed: %v", err)
		}
		if len(rbs) != 1 {
			t.Errorf("Expected 1 role binding, got %d", len(rbs))
		}
	})

	t.Run("ListClusterRoles", func(t *testing.T) {
		crs, err := k8sClient.ListClusterRoles(ctx)
		if err != nil {
			t.Fatalf("ListClusterRoles failed: %v", err)
		}
		if len(crs) != 1 {
			t.Errorf("Expected 1 cluster role, got %d", len(crs))
		}
	})

	t.Run("ListClusterRoleBindings", func(t *testing.T) {
		crbs, err := k8sClient.ListClusterRoleBindings(ctx)
		if err != nil {
			t.Fatalf("ListClusterRoleBindings failed: %v", err)
		}
		if len(crbs) != 1 {
			t.Errorf("Expected 1 cluster role binding, got %d", len(crbs))
		}
	})

	// Test utility methods used by TUI
	t.Run("GetGVR", func(t *testing.T) {
		// Test that GetGVR returns correct GVR for common resources
		// Note: events use ListEvents directly, not GetGVR
		resources := []string{
			"pods", "deployments", "services", "nodes", "namespaces",
			"configmaps", "secrets", "ingresses",
		}
		for _, r := range resources {
			gvr, ok := k8sClient.GetGVR(r)
			if !ok {
				t.Errorf("GetGVR(%s) returned false", r)
			}
			if gvr.Resource == "" {
				t.Errorf("GetGVR(%s) returned empty resource", r)
			}
		}
	})

	t.Run("GetCommonResources", func(t *testing.T) {
		resources := k8sClient.GetCommonResources()
		if len(resources) == 0 {
			t.Error("GetCommonResources returned empty list")
		}
	})
}

// TestTUIK8sClientAllNamespaces verifies listing resources across all namespaces
func TestTUIK8sClientAllNamespaces(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "default"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "kube-system"}},
	)

	k8sClient := &k8s.Client{
		Clientset: fakeClientset,
	}

	ctx := context.Background()

	// Empty namespace should list all namespaces
	pods, err := k8sClient.ListPods(ctx, "")
	if err != nil {
		t.Fatalf("ListPods with empty namespace failed: %v", err)
	}
	if len(pods) != 2 {
		t.Errorf("Expected 2 pods across all namespaces, got %d", len(pods))
	}
}

// TestTUIAppResourceCommands verifies command definitions cover all TUI resources
func TestTUIAppResourceCommands(t *testing.T) {
	// Resources that TUI fetches (from app_fetch.go)
	tuiResources := []string{
		"pods", "deployments", "services", "nodes", "namespaces", "events",
		"configmaps", "secrets", "persistentvolumes", "persistentvolumeclaims",
		"storageclasses", "replicasets", "daemonsets", "statefulsets",
		"jobs", "cronjobs", "replicationcontrollers", "ingresses",
		"endpoints", "networkpolicies", "serviceaccounts", "roles",
		"rolebindings", "clusterroles", "clusterrolebindings",
	}

	// Check that all TUI resources have a command definition
	for _, resource := range tuiResources {
		found := false
		for _, cmd := range commands {
			if cmd.name == resource {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("TUI resource %q has no command definition", resource)
		}
	}
}

// TestTUIAppResourceAliases verifies all resource aliases work
func TestTUIAppResourceAliases(t *testing.T) {
	app := &App{
		namespaces: []string{"", "default"},
	}

	// Common aliases used in TUI
	aliases := map[string]string{
		"po":     "pods",
		"deploy": "deployments",
		"svc":    "services",
		"no":     "nodes",
		"ns":     "namespaces",
		"ev":     "events",
		"cm":     "configmaps",
		"sec":    "secrets",
		"pv":     "persistentvolumes",
		"pvc":    "persistentvolumeclaims",
		"sc":     "storageclasses",
		"rs":     "replicasets",
		"ds":     "daemonsets",
		"sts":    "statefulsets",
		"job":    "jobs",
		"cj":     "cronjobs",
		"ing":    "ingresses",
		"ep":     "endpoints",
		"netpol": "networkpolicies",
		"sa":     "serviceaccounts",
	}

	for alias, expectedResource := range aliases {
		completions := app.getCompletions(alias)
		if len(completions) == 0 {
			t.Errorf("Alias %q returned no completions", alias)
			continue
		}
		if completions[0] != expectedResource {
			t.Errorf("Alias %q: expected %q, got %q", alias, expectedResource, completions[0])
		}
	}
}

// TestTUIResourceViewMapping verifies each resource has a fetch handler
func TestTUIResourceViewMapping(t *testing.T) {
	// This test verifies that all commands map to actual fetch functions
	// by checking the command definitions exist

	resourceCommands := []string{}
	actionCommands := []string{}

	for _, cmd := range commands {
		switch cmd.category {
		case "resource":
			resourceCommands = append(resourceCommands, cmd.name)
		case "action":
			actionCommands = append(actionCommands, cmd.name)
		}
	}

	// Verify we have both resource and action commands
	if len(resourceCommands) == 0 {
		t.Error("No resource commands defined")
	}
	if len(actionCommands) == 0 {
		t.Error("No action commands defined")
	}

	// Verify expected minimum counts
	if len(resourceCommands) < 30 {
		t.Errorf("Expected at least 30 resource commands, got %d", len(resourceCommands))
	}
	if len(actionCommands) < 3 {
		t.Errorf("Expected at least 3 action commands, got %d", len(actionCommands))
	}
}
