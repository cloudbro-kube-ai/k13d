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
	"k8s.io/client-go/kubernetes/fake"
)

func TestListPods(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	})
	client := &Client{Clientset: clientset}

	pods, err := client.ListPods(ctx, "default")
	if err != nil {
		t.Fatalf("ListPods failed: %v", err)
	}

	if len(pods) != 1 {
		t.Errorf("Expected 1 pod, got %d", len(pods))
	}
	if pods[0].Name != "test-pod" {
		t.Errorf("Expected pod name test-pod, got %s", pods[0].Name)
	}
}

func TestListNodes(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset(&corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
	})
	client := &Client{Clientset: clientset}

	nodes, err := client.ListNodes(ctx)
	if err != nil {
		t.Fatalf("ListNodes failed: %v", err)
	}

	if len(nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(nodes))
	}
}

func TestListDeployments(t *testing.T) {
	ctx := context.Background()
	replicas := int32(3)
	clientset := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deploy",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	})
	client := &Client{Clientset: clientset}

	deps, err := client.ListDeployments(ctx, "default")
	if err != nil {
		t.Fatalf("ListDeployments failed: %v", err)
	}

	if len(deps) != 1 {
		t.Errorf("Expected 1 deployment, got %d", len(deps))
	}
	if deps[0].Name != "test-deploy" {
		t.Errorf("Expected deployment name test-deploy, got %s", deps[0].Name)
	}
}

func TestListStatefulSets(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset(&appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sts",
			Namespace: "default",
		},
	})
	client := &Client{Clientset: clientset}

	sts, err := client.ListStatefulSets(ctx, "default")
	if err != nil {
		t.Fatalf("ListStatefulSets failed: %v", err)
	}

	if len(sts) != 1 {
		t.Errorf("Expected 1 statefulset, got %d", len(sts))
	}
}

func TestListServices(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-svc",
			Namespace: "default",
		},
	})
	client := &Client{Clientset: clientset}

	svcs, err := client.ListServices(ctx, "default")
	if err != nil {
		t.Fatalf("ListServices failed: %v", err)
	}

	if len(svcs) != 1 {
		t.Errorf("Expected 1 service, got %d", len(svcs))
	}
}

func TestListConfigMaps(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "default",
		},
		Data: map[string]string{"key": "value"},
	})
	client := &Client{Clientset: clientset}

	cms, err := client.ListConfigMaps(ctx, "default")
	if err != nil {
		t.Fatalf("ListConfigMaps failed: %v", err)
	}

	if len(cms) != 1 {
		t.Errorf("Expected 1 configmap, got %d", len(cms))
	}
}

func TestListSecrets(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
	})
	client := &Client{Clientset: clientset}

	secrets, err := client.ListSecrets(ctx, "default")
	if err != nil {
		t.Fatalf("ListSecrets failed: %v", err)
	}

	if len(secrets) != 1 {
		t.Errorf("Expected 1 secret, got %d", len(secrets))
	}
}

func TestListIngresses(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset(&networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ing",
			Namespace: "default",
		},
	})
	client := &Client{Clientset: clientset}

	ings, err := client.ListIngresses(ctx, "default")
	if err != nil {
		t.Fatalf("ListIngresses failed: %v", err)
	}

	if len(ings) != 1 {
		t.Errorf("Expected 1 ingress, got %d", len(ings))
	}
}

func TestListNamespaces(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},
	)
	client := &Client{Clientset: clientset}

	nss, err := client.ListNamespaces(ctx)
	if err != nil {
		t.Fatalf("ListNamespaces failed: %v", err)
	}

	if len(nss) != 2 {
		t.Errorf("Expected 2 namespaces, got %d", len(nss))
	}
}

func TestListEvents(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset(&corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-event",
			Namespace: "default",
		},
		Type:    "Warning",
		Reason:  "TestReason",
		Message: "Test message",
	})
	client := &Client{Clientset: clientset}

	events, err := client.ListEvents(ctx, "default")
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}
}

func TestListRoles(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset(&rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-role",
			Namespace: "default",
		},
	})
	client := &Client{Clientset: clientset}

	roles, err := client.ListRoles(ctx, "default")
	if err != nil {
		t.Fatalf("ListRoles failed: %v", err)
	}

	if len(roles) != 1 {
		t.Errorf("Expected 1 role, got %d", len(roles))
	}
}

func TestListRoleBindings(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset(&rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rb",
			Namespace: "default",
		},
	})
	client := &Client{Clientset: clientset}

	rbs, err := client.ListRoleBindings(ctx, "default")
	if err != nil {
		t.Fatalf("ListRoleBindings failed: %v", err)
	}

	if len(rbs) != 1 {
		t.Errorf("Expected 1 rolebinding, got %d", len(rbs))
	}
}

func TestListClusterRoles(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset(&rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-clusterrole",
		},
	})
	client := &Client{Clientset: clientset}

	crs, err := client.ListClusterRoles(ctx)
	if err != nil {
		t.Fatalf("ListClusterRoles failed: %v", err)
	}

	if len(crs) != 1 {
		t.Errorf("Expected 1 clusterrole, got %d", len(crs))
	}
}

func TestListClusterRoleBindings(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset(&rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-crb",
		},
	})
	client := &Client{Clientset: clientset}

	crbs, err := client.ListClusterRoleBindings(ctx)
	if err != nil {
		t.Fatalf("ListClusterRoleBindings failed: %v", err)
	}

	if len(crbs) != 1 {
		t.Errorf("Expected 1 clusterrolebinding, got %d", len(crbs))
	}
}

func TestListPersistentVolumes(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset(&corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pv",
		},
	})
	client := &Client{Clientset: clientset}

	pvs, err := client.ListPersistentVolumes(ctx)
	if err != nil {
		t.Fatalf("ListPersistentVolumes failed: %v", err)
	}

	if len(pvs) != 1 {
		t.Errorf("Expected 1 pv, got %d", len(pvs))
	}
}

func TestListPersistentVolumeClaims(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset(&corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
		},
	})
	client := &Client{Clientset: clientset}

	pvcs, err := client.ListPersistentVolumeClaims(ctx, "default")
	if err != nil {
		t.Fatalf("ListPersistentVolumeClaims failed: %v", err)
	}

	if len(pvcs) != 1 {
		t.Errorf("Expected 1 pvc, got %d", len(pvcs))
	}
}

func TestListStorageClasses(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset(&storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-sc",
		},
	})
	client := &Client{Clientset: clientset}

	scs, err := client.ListStorageClasses(ctx)
	if err != nil {
		t.Fatalf("ListStorageClasses failed: %v", err)
	}

	if len(scs) != 1 {
		t.Errorf("Expected 1 storageclass, got %d", len(scs))
	}
}

func TestListServiceAccounts(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset(&corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sa",
			Namespace: "default",
		},
	})
	client := &Client{Clientset: clientset}

	sas, err := client.ListServiceAccounts(ctx, "default")
	if err != nil {
		t.Fatalf("ListServiceAccounts failed: %v", err)
	}

	if len(sas) != 1 {
		t.Errorf("Expected 1 serviceaccount, got %d", len(sas))
	}
}

func TestListDaemonSets(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset(&appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ds",
			Namespace: "default",
		},
	})
	client := &Client{Clientset: clientset}

	dss, err := client.ListDaemonSets(ctx, "default")
	if err != nil {
		t.Fatalf("ListDaemonSets failed: %v", err)
	}

	if len(dss) != 1 {
		t.Errorf("Expected 1 daemonset, got %d", len(dss))
	}
}

func TestListJobs(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset(&batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job",
			Namespace: "default",
		},
	})
	client := &Client{Clientset: clientset}

	jobs, err := client.ListJobs(ctx, "default")
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}

	if len(jobs) != 1 {
		t.Errorf("Expected 1 job, got %d", len(jobs))
	}
}

func TestListCronJobs(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset(&batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cj",
			Namespace: "default",
		},
	})
	client := &Client{Clientset: clientset}

	cjs, err := client.ListCronJobs(ctx, "default")
	if err != nil {
		t.Fatalf("ListCronJobs failed: %v", err)
	}

	if len(cjs) != 1 {
		t.Errorf("Expected 1 cronjob, got %d", len(cjs))
	}
}

func TestListNetworkPolicies(t *testing.T) {
	ctx := context.Background()
	clientset := fake.NewSimpleClientset(&networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-netpol",
			Namespace: "default",
		},
	})
	client := &Client{Clientset: clientset}

	netpols, err := client.ListNetworkPolicies(ctx, "default")
	if err != nil {
		t.Fatalf("ListNetworkPolicies failed: %v", err)
	}

	if len(netpols) != 1 {
		t.Errorf("Expected 1 networkpolicy, got %d", len(netpols))
	}
}

func TestGetGVR(t *testing.T) {
	client := &Client{}

	cases := []struct {
		res      string
		expected string
	}{
		{"pods", "pods"},
		{"po", "pods"},
		{"services", "services"},
		{"svc", "services"},
		{"deployments", "deployments"},
		{"deploy", "deployments"},
		{"statefulsets", "statefulsets"},
		{"sts", "statefulsets"},
		{"daemonsets", "daemonsets"},
		{"ds", "daemonsets"},
		{"jobs", "jobs"},
		{"cronjobs", "cronjobs"},
		{"cj", "cronjobs"},
		{"configmaps", "configmaps"},
		{"cm", "configmaps"},
		{"secrets", "secrets"},
		{"ingresses", "ingresses"},
		{"ing", "ingresses"},
		{"roles", "roles"},
		{"rolebindings", "rolebindings"},
		{"rb", "rolebindings"},
		{"clusterroles", "clusterroles"},
		{"clusterrolebindings", "clusterrolebindings"},
		{"crb", "clusterrolebindings"},
		{"persistentvolumes", "persistentvolumes"},
		{"pv", "persistentvolumes"},
		{"persistentvolumeclaims", "persistentvolumeclaims"},
		{"pvc", "persistentvolumeclaims"},
		{"storageclasses", "storageclasses"},
		{"sc", "storageclasses"},
		{"serviceaccounts", "serviceaccounts"},
		{"sa", "serviceaccounts"},
		{"hpa", "horizontalpodautoscalers"},
		{"networkpolicies", "networkpolicies"},
		{"netpol", "networkpolicies"},
		{"invalid", ""},
	}

	for _, c := range cases {
		gvr, ok := client.GetGVR(c.res)
		if c.expected == "" {
			if ok {
				t.Errorf("expected fail for %s, but got %v", c.res, gvr)
			}
		} else {
			if !ok || gvr.Resource != c.expected {
				t.Errorf("expected %s for %s, got %v", c.expected, c.res, gvr)
			}
		}
	}
}

func TestGetGVR_CaseInsensitive(t *testing.T) {
	client := &Client{}

	cases := []struct {
		res      string
		expected string
	}{
		{"PODS", "pods"},
		{"Pods", "pods"},
		{"DeployMents", "deployments"},
		{"CONFIGMAPS", "configmaps"},
	}

	for _, c := range cases {
		gvr, ok := client.GetGVR(c.res)
		if !ok || gvr.Resource != c.expected {
			t.Errorf("expected %s for %s, got %v (ok=%v)", c.expected, c.res, gvr, ok)
		}
	}
}

func TestDefaultGetOptions(t *testing.T) {
	opts := DefaultGetOptions()
	if opts.ResourceVersion != "" {
		t.Error("expected empty resource version")
	}
}

func TestDefaultListOptions(t *testing.T) {
	opts := DefaultListOptions()
	if opts.LabelSelector != "" {
		t.Error("expected empty label selector")
	}
	if opts.FieldSelector != "" {
		t.Error("expected empty field selector")
	}
}

func TestClient_GetPodMetrics_NoMetricsClient(t *testing.T) {
	ctx := context.Background()
	client := &Client{Metrics: nil}

	_, err := client.GetPodMetrics(ctx, "default")
	if err == nil {
		t.Error("expected error when metrics client is nil")
	}
}

func TestClient_GetNodeMetrics_NoMetricsClient(t *testing.T) {
	ctx := context.Background()
	client := &Client{Metrics: nil}

	_, err := client.GetNodeMetrics(ctx)
	if err == nil {
		t.Error("expected error when metrics client is nil")
	}
}

func TestCRDInfo_Struct(t *testing.T) {
	crdInfo := &CRDInfo{
		Name:       "certificates.cert-manager.io",
		Group:      "cert-manager.io",
		Version:    "v1",
		Kind:       "Certificate",
		Plural:     "certificates",
		Namespaced: true,
		ShortNames: []string{"cert", "certs"},
	}

	if crdInfo.Name != "certificates.cert-manager.io" {
		t.Errorf("expected name certificates.cert-manager.io, got %s", crdInfo.Name)
	}
	if crdInfo.Group != "cert-manager.io" {
		t.Errorf("expected group cert-manager.io, got %s", crdInfo.Group)
	}
	if crdInfo.Version != "v1" {
		t.Errorf("expected version v1, got %s", crdInfo.Version)
	}
	if crdInfo.Kind != "Certificate" {
		t.Errorf("expected kind Certificate, got %s", crdInfo.Kind)
	}
	if crdInfo.Plural != "certificates" {
		t.Errorf("expected plural certificates, got %s", crdInfo.Plural)
	}
	if !crdInfo.Namespaced {
		t.Error("expected namespaced to be true")
	}
	if len(crdInfo.ShortNames) != 2 {
		t.Errorf("expected 2 short names, got %d", len(crdInfo.ShortNames))
	}
}

func TestCRDInfo_ClusterScoped(t *testing.T) {
	crdInfo := &CRDInfo{
		Name:       "clusterwidgets.example.com",
		Group:      "example.com",
		Version:    "v1",
		Kind:       "ClusterWidget",
		Plural:     "clusterwidgets",
		Namespaced: false,
		ShortNames: nil,
	}

	if crdInfo.Namespaced {
		t.Error("expected namespaced to be false for cluster-scoped resource")
	}
	if crdInfo.ShortNames != nil {
		t.Error("expected nil short names")
	}
}

func TestCRDInfo_EmptyShortNames(t *testing.T) {
	crdInfo := &CRDInfo{
		Name:       "widgets.example.com",
		Group:      "example.com",
		Version:    "v1alpha1",
		Kind:       "Widget",
		Plural:     "widgets",
		Namespaced: true,
		ShortNames: []string{},
	}

	if len(crdInfo.ShortNames) != 0 {
		t.Errorf("expected 0 short names, got %d", len(crdInfo.ShortNames))
	}
}
