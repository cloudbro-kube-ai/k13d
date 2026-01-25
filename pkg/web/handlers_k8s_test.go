package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/config"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/db"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// setupK8sTestServer creates a test server with comprehensive fake K8s objects
func setupK8sTestServer(t *testing.T) (*Server, http.Handler) {
	t.Helper()

	cfg := &config.Config{
		Language:    "en",
		EnableAudit: true,
		LLM: config.LLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
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
	authManager := NewAuthManager(authConfig)

	// Create comprehensive fake K8s objects
	replicas := int32(3)
	completions := int32(1)
	suspend := false
	ingressClassName := "nginx"
	storageClassName := "standard"
	volumeBindingMode := storagev1.VolumeBindingImmediate
	allowExpand := true
	reclaimPolicy := corev1.PersistentVolumeReclaimDelete

	fakeClientset := fake.NewSimpleClientset(
		// Namespaces
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "default"},
			Status:     corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "kube-system"},
			Status:     corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
		},

		// Pods
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-1",
				Namespace: "default",
				Labels:    map[string]string{"app": "test"},
			},
			Spec: corev1.PodSpec{
				NodeName: "node-1",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				PodIP: "10.0.0.1",
				ContainerStatuses: []corev1.ContainerStatus{
					{Name: "app", Ready: true, RestartCount: 2},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-2",
				Namespace: "default",
				Labels:    map[string]string{"app": "test"},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodPending,
			},
		},

		// Deployments
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "test"},
				},
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:     2,
				UpdatedReplicas:   3,
				AvailableReplicas: 2,
			},
		},

		// Services
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-service",
				Namespace: "default",
			},
			Spec: corev1.ServiceSpec{
				Type:      corev1.ServiceTypeClusterIP,
				ClusterIP: "10.96.0.1",
				Ports: []corev1.ServicePort{
					{Port: 80, Protocol: corev1.ProtocolTCP},
					{Port: 443, Protocol: corev1.ProtocolTCP},
				},
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lb-service",
				Namespace: "default",
			},
			Spec: corev1.ServiceSpec{
				Type:      corev1.ServiceTypeLoadBalancer,
				ClusterIP: "10.96.0.2",
				Ports:     []corev1.ServicePort{{Port: 80, Protocol: corev1.ProtocolTCP}},
			},
			Status: corev1.ServiceStatus{
				LoadBalancer: corev1.LoadBalancerStatus{
					Ingress: []corev1.LoadBalancerIngress{
						{IP: "203.0.113.1"},
					},
				},
			},
		},

		// Nodes
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-1",
				Labels: map[string]string{
					"node-role.kubernetes.io/control-plane": "",
					"node-role.kubernetes.io/master":        "",
				},
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
				NodeInfo: corev1.NodeSystemInfo{
					KubeletVersion: "v1.28.0",
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-2",
				Labels: map[string]string{
					"node-role.kubernetes.io/worker": "",
				},
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
				},
				NodeInfo: corev1.NodeSystemInfo{
					KubeletVersion: "v1.28.0",
				},
			},
		},

		// Events
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-event",
				Namespace: "default",
			},
			Type:          corev1.EventTypeWarning,
			Reason:        "FailedScheduling",
			Message:       "No nodes available",
			Count:         5,
			LastTimestamp: metav1.Now(),
		},

		// StatefulSets
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset",
				Namespace: "default",
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "statefulset-test"},
				},
			},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas: 2,
			},
		},

		// DaemonSets
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-daemonset",
				Namespace: "default",
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "daemonset-test"},
				},
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						NodeSelector: map[string]string{"role": "worker"},
					},
				},
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 3,
				CurrentNumberScheduled: 2,
				NumberReady:            2,
				UpdatedNumberScheduled: 2,
				NumberAvailable:        2,
			},
		},

		// ConfigMaps
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-configmap",
				Namespace: "default",
			},
			Data: map[string]string{
				"config.yaml": "key: value",
				"env":         "production",
			},
		},

		// Secrets
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "default",
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"password": []byte("secret123"),
			},
		},

		// Ingresses
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-ingress",
				Namespace: "default",
			},
			Spec: networkingv1.IngressSpec{
				IngressClassName: &ingressClassName,
				Rules: []networkingv1.IngressRule{
					{Host: "example.com"},
					{Host: "api.example.com"},
				},
			},
			Status: networkingv1.IngressStatus{
				LoadBalancer: networkingv1.IngressLoadBalancerStatus{
					Ingress: []networkingv1.IngressLoadBalancerIngress{
						{IP: "203.0.113.10"},
					},
				},
			},
		},

		// ClusterRoles
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-clusterrole",
			},
			Rules: []rbacv1.PolicyRule{
				{APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"get", "list"}},
			},
		},

		// ClusterRoleBindings
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-clusterrolebinding",
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "test-clusterrole",
			},
			Subjects: []rbacv1.Subject{
				{Kind: "ServiceAccount", Name: "default", Namespace: "default"},
			},
		},

		// Roles
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-role",
				Namespace: "default",
			},
			Rules: []rbacv1.PolicyRule{
				{APIGroups: []string{""}, Resources: []string{"secrets"}, Verbs: []string{"get"}},
			},
		},

		// RoleBindings
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-rolebinding",
				Namespace: "default",
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     "test-role",
			},
			Subjects: []rbacv1.Subject{
				{Kind: "User", Name: "test-user"},
			},
		},

		// ServiceAccounts
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-serviceaccount",
				Namespace: "default",
			},
			Secrets: []corev1.ObjectReference{
				{Name: "test-sa-token"},
			},
		},

		// PersistentVolumes
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pv",
			},
			Spec: corev1.PersistentVolumeSpec{
				Capacity: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("10Gi"),
				},
				AccessModes:                   []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
				StorageClassName:              "standard",
				ClaimRef: &corev1.ObjectReference{
					Namespace: "default",
					Name:      "test-pvc",
				},
			},
			Status: corev1.PersistentVolumeStatus{
				Phase: corev1.VolumeBound,
			},
		},

		// PersistentVolumeClaims
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pvc",
				Namespace: "default",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				StorageClassName: &storageClassName,
				VolumeName:       "test-pv",
			},
			Status: corev1.PersistentVolumeClaimStatus{
				Phase: corev1.ClaimBound,
				Capacity: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("10Gi"),
				},
			},
		},

		// StorageClasses
		&storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "standard",
			},
			Provisioner:          "kubernetes.io/aws-ebs",
			ReclaimPolicy:        &reclaimPolicy,
			VolumeBindingMode:    &volumeBindingMode,
			AllowVolumeExpansion: &allowExpand,
		},

		// Jobs
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-job",
				Namespace: "default",
			},
			Spec: batchv1.JobSpec{
				Completions: &completions,
			},
			Status: batchv1.JobStatus{
				Succeeded:      1,
				StartTime:      &metav1.Time{Time: time.Now().Add(-10 * time.Minute)},
				CompletionTime: &metav1.Time{Time: time.Now().Add(-5 * time.Minute)},
			},
		},

		// CronJobs
		&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cronjob",
				Namespace: "default",
			},
			Spec: batchv1.CronJobSpec{
				Schedule: "*/5 * * * *",
				Suspend:  &suspend,
			},
			Status: batchv1.CronJobStatus{
				LastScheduleTime: &metav1.Time{Time: time.Now().Add(-3 * time.Minute)},
				Active:           []corev1.ObjectReference{{Name: "test-job"}},
			},
		},

		// ReplicaSets
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-replicaset",
				Namespace: "default",
			},
			Spec: appsv1.ReplicaSetSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "replicaset-test"},
				},
			},
			Status: appsv1.ReplicaSetStatus{
				Replicas:      3,
				ReadyReplicas: 2,
			},
		},

		// HPAs
		&autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-hpa",
				Namespace: "default",
			},
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
					Kind: "Deployment",
					Name: "test-deployment",
				},
				MinReplicas: &replicas,
				MaxReplicas: 10,
			},
			Status: autoscalingv2.HorizontalPodAutoscalerStatus{
				CurrentReplicas: 3,
			},
		},

		// NetworkPolicies
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-networkpolicy",
				Namespace: "default",
			},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "test"},
				},
			},
		},
	)

	k8sClient := &k8s.Client{
		Clientset: fakeClientset,
	}

	server := &Server{
		cfg:              cfg,
		aiClient:         nil,
		k8sClient:        k8sClient,
		authManager:      authManager,
		port:             8080,
		pendingApprovals: make(map[string]*PendingToolApproval),
	}
	server.reportGenerator = NewReportGenerator(server)

	mux := getServerMux(server)
	return server, mux
}

// TestK8sResourceHandlerPods tests the pods endpoint
func TestK8sResourceHandlerPods(t *testing.T) {
	dbPath := "test_k8s_pods.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "List all pods",
			path:           "/api/k8s/pods",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				items := body["items"].([]interface{})
				if len(items) != 2 {
					t.Errorf("Expected 2 pods, got %d", len(items))
				}
				if body["kind"] != "pods" {
					t.Errorf("Expected kind 'pods', got %v", body["kind"])
				}
			},
		},
		{
			name:           "List pods in default namespace",
			path:           "/api/k8s/pods?namespace=default",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				items := body["items"].([]interface{})
				if len(items) != 2 {
					t.Errorf("Expected 2 pods in default namespace, got %d", len(items))
				}
			},
		},
		{
			name:           "List pods with label selector",
			path:           "/api/k8s/pods?namespace=default&labelSelector=app=test",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				items := body["items"].([]interface{})
				if len(items) < 1 {
					t.Errorf("Expected at least 1 pod with app=test label")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var body map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, body)
			}
		})
	}
}

// TestK8sResourceHandlerDeployments tests the deployments endpoint
func TestK8sResourceHandlerDeployments(t *testing.T) {
	dbPath := "test_k8s_deployments.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/k8s/deployments", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	items := body["items"].([]interface{})
	if len(items) != 1 {
		t.Errorf("Expected 1 deployment, got %d", len(items))
	}

	dep := items[0].(map[string]interface{})
	if dep["name"] != "test-deployment" {
		t.Errorf("Expected deployment name 'test-deployment', got %v", dep["name"])
	}
	if dep["ready"] != "2/3" {
		t.Errorf("Expected ready '2/3', got %v", dep["ready"])
	}
}

// TestK8sResourceHandlerServices tests the services endpoint
func TestK8sResourceHandlerServices(t *testing.T) {
	dbPath := "test_k8s_services.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/k8s/services", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	items := body["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("Expected 2 services, got %d", len(items))
	}

	// Check LoadBalancer service has external IP
	for _, item := range items {
		svc := item.(map[string]interface{})
		if svc["name"] == "lb-service" {
			if svc["externalIP"] != "203.0.113.1" {
				t.Errorf("Expected externalIP '203.0.113.1', got %v", svc["externalIP"])
			}
		}
	}
}

// TestK8sResourceHandlerNodes tests the nodes endpoint
func TestK8sResourceHandlerNodes(t *testing.T) {
	dbPath := "test_k8s_nodes.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/k8s/nodes", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	items := body["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(items))
	}

	// Check node statuses
	for _, item := range items {
		node := item.(map[string]interface{})
		if node["name"] == "node-1" {
			if node["status"] != "Ready" {
				t.Errorf("Expected node-1 status 'Ready', got %v", node["status"])
			}
		}
		if node["name"] == "node-2" {
			if node["status"] != "NotReady" {
				t.Errorf("Expected node-2 status 'NotReady', got %v", node["status"])
			}
		}
	}
}

// TestK8sResourceHandlerNamespaces tests the namespaces endpoint
func TestK8sResourceHandlerNamespaces(t *testing.T) {
	dbPath := "test_k8s_namespaces.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/k8s/namespaces", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	items := body["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("Expected 2 namespaces, got %d", len(items))
	}
}

// TestK8sResourceHandlerEvents tests the events endpoint
func TestK8sResourceHandlerEvents(t *testing.T) {
	dbPath := "test_k8s_events.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/k8s/events", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	items := body["items"].([]interface{})
	if len(items) != 1 {
		t.Errorf("Expected 1 event, got %d", len(items))
	}

	event := items[0].(map[string]interface{})
	if event["type"] != "Warning" {
		t.Errorf("Expected event type 'Warning', got %v", event["type"])
	}
	if event["reason"] != "FailedScheduling" {
		t.Errorf("Expected reason 'FailedScheduling', got %v", event["reason"])
	}
}

// TestK8sResourceHandlerStatefulSets tests the statefulsets endpoint
func TestK8sResourceHandlerStatefulSets(t *testing.T) {
	dbPath := "test_k8s_statefulsets.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/k8s/statefulsets", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	items := body["items"].([]interface{})
	if len(items) != 1 {
		t.Errorf("Expected 1 statefulset, got %d", len(items))
	}

	sts := items[0].(map[string]interface{})
	if sts["ready"] != "2/3" {
		t.Errorf("Expected ready '2/3', got %v", sts["ready"])
	}
	// Verify selector field is present for log viewing
	if selector, ok := sts["selector"].(string); ok && selector != "" {
		if selector != "app=statefulset-test" {
			t.Errorf("Expected selector 'app=statefulset-test', got %v", selector)
		}
	}
}

// TestK8sResourceHandlerDaemonSets tests the daemonsets endpoint
func TestK8sResourceHandlerDaemonSets(t *testing.T) {
	dbPath := "test_k8s_daemonsets.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/k8s/daemonsets", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	items := body["items"].([]interface{})
	if len(items) != 1 {
		t.Errorf("Expected 1 daemonset, got %d", len(items))
	}

	ds := items[0].(map[string]interface{})
	if ds["nodeSelector"] == "" {
		t.Error("Expected nodeSelector to be set")
	}
	// Verify selector field is present for log viewing
	if selector, ok := ds["selector"].(string); ok && selector != "" {
		if selector != "app=daemonset-test" {
			t.Errorf("Expected selector 'app=daemonset-test', got %v", selector)
		}
	}
}

// TestK8sResourceHandlerConfigMaps tests the configmaps endpoint
func TestK8sResourceHandlerConfigMaps(t *testing.T) {
	dbPath := "test_k8s_configmaps.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/k8s/configmaps", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	items := body["items"].([]interface{})
	if len(items) != 1 {
		t.Errorf("Expected 1 configmap, got %d", len(items))
	}

	cm := items[0].(map[string]interface{})
	if cm["data"].(float64) != 2 {
		t.Errorf("Expected 2 data keys, got %v", cm["data"])
	}
}

// TestK8sResourceHandlerSecrets tests the secrets endpoint
func TestK8sResourceHandlerSecrets(t *testing.T) {
	dbPath := "test_k8s_secrets.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/k8s/secrets", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	items := body["items"].([]interface{})
	if len(items) != 1 {
		t.Errorf("Expected 1 secret, got %d", len(items))
	}

	secret := items[0].(map[string]interface{})
	if secret["type"] != "Opaque" {
		t.Errorf("Expected type 'Opaque', got %v", secret["type"])
	}
}

// TestK8sResourceHandlerIngresses tests the ingresses endpoint
func TestK8sResourceHandlerIngresses(t *testing.T) {
	dbPath := "test_k8s_ingresses.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/k8s/ingresses", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	items := body["items"].([]interface{})
	if len(items) != 1 {
		t.Errorf("Expected 1 ingress, got %d", len(items))
	}

	ing := items[0].(map[string]interface{})
	if ing["class"] != "nginx" {
		t.Errorf("Expected class 'nginx', got %v", ing["class"])
	}
	if ing["address"] != "203.0.113.10" {
		t.Errorf("Expected address '203.0.113.10', got %v", ing["address"])
	}
}

// TestK8sResourceHandlerRBAC tests the RBAC-related endpoints
func TestK8sResourceHandlerRBAC(t *testing.T) {
	dbPath := "test_k8s_rbac.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	tests := []struct {
		name          string
		path          string
		expectedCount int
	}{
		{"ClusterRoles", "/api/k8s/clusterroles", 1},
		{"ClusterRoleBindings", "/api/k8s/clusterrolebindings", 1},
		{"Roles", "/api/k8s/roles", 1},
		{"RoleBindings", "/api/k8s/rolebindings", 1},
		{"ServiceAccounts", "/api/k8s/serviceaccounts", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var body map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			items := body["items"].([]interface{})
			if len(items) != tt.expectedCount {
				t.Errorf("Expected %d %s, got %d", tt.expectedCount, tt.name, len(items))
			}
		})
	}
}

// TestK8sResourceHandlerStorage tests storage-related endpoints
func TestK8sResourceHandlerStorage(t *testing.T) {
	dbPath := "test_k8s_storage.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	tests := []struct {
		name          string
		path          string
		expectedCount int
		checkField    string
		expectedValue interface{}
	}{
		{"PersistentVolumes", "/api/k8s/persistentvolumes", 1, "", nil},
		{"PersistentVolumes (alias)", "/api/k8s/pv", 1, "", nil},
		{"PersistentVolumeClaims", "/api/k8s/persistentvolumeclaims", 1, "", nil},
		{"PersistentVolumeClaims (alias)", "/api/k8s/pvc", 1, "", nil},
		{"StorageClasses", "/api/k8s/storageclasses", 1, "", nil},
		{"StorageClasses (alias)", "/api/k8s/sc", 1, "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var body map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			items := body["items"].([]interface{})
			if len(items) != tt.expectedCount {
				t.Errorf("Expected %d items, got %d", tt.expectedCount, len(items))
			}
		})
	}
}

// TestK8sResourceHandlerJobs tests job-related endpoints
func TestK8sResourceHandlerJobs(t *testing.T) {
	dbPath := "test_k8s_jobs.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	t.Run("Jobs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/k8s/jobs", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var body map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		items := body["items"].([]interface{})
		if len(items) != 1 {
			t.Errorf("Expected 1 job, got %d", len(items))
		}

		job := items[0].(map[string]interface{})
		if job["completions"] != "1/1" {
			t.Errorf("Expected completions '1/1', got %v", job["completions"])
		}
	})

	t.Run("CronJobs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/k8s/cronjobs", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var body map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		items := body["items"].([]interface{})
		if len(items) != 1 {
			t.Errorf("Expected 1 cronjob, got %d", len(items))
		}

		cj := items[0].(map[string]interface{})
		if cj["schedule"] != "*/5 * * * *" {
			t.Errorf("Expected schedule '*/5 * * * *', got %v", cj["schedule"])
		}
		if cj["active"].(float64) != 1 {
			t.Errorf("Expected 1 active job, got %v", cj["active"])
		}
	})
}

// TestK8sResourceHandlerReplicaSets tests the replicasets endpoint
func TestK8sResourceHandlerReplicaSets(t *testing.T) {
	dbPath := "test_k8s_replicasets.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	// Test both main path and alias
	paths := []string{"/api/k8s/replicasets", "/api/k8s/rs"}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var body map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			items := body["items"].([]interface{})
			if len(items) != 1 {
				t.Errorf("Expected 1 replicaset, got %d", len(items))
			}

			rs := items[0].(map[string]interface{})
			if rs["desired"].(float64) != 3 {
				t.Errorf("Expected desired 3, got %v", rs["desired"])
			}
			if rs["ready"].(string) != "2/3" {
				t.Errorf("Expected ready '2/3', got %v", rs["ready"])
			}
			// Verify selector field is present for log viewing
			if selector, ok := rs["selector"].(string); ok && selector != "" {
				if selector != "app=replicaset-test" {
					t.Errorf("Expected selector 'app=replicaset-test', got %v", selector)
				}
			}
		})
	}
}

// TestK8sResourceHandlerHPA tests the HPA endpoint
func TestK8sResourceHandlerHPA(t *testing.T) {
	dbPath := "test_k8s_hpa.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	// Test both main path and alias
	paths := []string{"/api/k8s/hpa", "/api/k8s/horizontalpodautoscalers"}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var body map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			items := body["items"].([]interface{})
			if len(items) != 1 {
				t.Errorf("Expected 1 HPA, got %d", len(items))
			}

			hpa := items[0].(map[string]interface{})
			if hpa["reference"] != "Deployment/test-deployment" {
				t.Errorf("Expected reference 'Deployment/test-deployment', got %v", hpa["reference"])
			}
			if hpa["maxReplicas"].(float64) != 10 {
				t.Errorf("Expected maxReplicas 10, got %v", hpa["maxReplicas"])
			}
		})
	}
}

// TestK8sResourceHandlerNetworkPolicies tests the networkpolicies endpoint
func TestK8sResourceHandlerNetworkPolicies(t *testing.T) {
	dbPath := "test_k8s_networkpolicies.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	// Test both main path and alias
	paths := []string{"/api/k8s/networkpolicies", "/api/k8s/netpol"}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var body map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			items := body["items"].([]interface{})
			if len(items) != 1 {
				t.Errorf("Expected 1 network policy, got %d", len(items))
			}

			np := items[0].(map[string]interface{})
			if np["name"] != "test-networkpolicy" {
				t.Errorf("Expected name 'test-networkpolicy', got %v", np["name"])
			}
		})
	}
}

// TestK8sResourceHandlerUnknownResource tests handling of unknown resource types
func TestK8sResourceHandlerUnknownResource(t *testing.T) {
	dbPath := "test_k8s_unknown.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/k8s/unknownresource", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if body["error"] == nil || body["error"] == "" {
		t.Error("Expected error message for unknown resource type")
	}

	items := body["items"].([]interface{})
	if len(items) != 0 {
		t.Errorf("Expected empty items for unknown resource, got %d", len(items))
	}
}

// TestK8sResourceHandlerMethodNotAllowed tests that non-GET requests are rejected
func TestK8sResourceHandlerMethodNotAllowed(t *testing.T) {
	dbPath := "test_k8s_method.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/k8s/pods", nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

// TestClusterOverview tests the cluster overview endpoint
func TestClusterOverview(t *testing.T) {
	dbPath := "test_cluster_overview.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/overview", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check nodes
	nodes := body["nodes"].(map[string]interface{})
	if nodes["total"].(float64) != 2 {
		t.Errorf("Expected 2 total nodes, got %v", nodes["total"])
	}
	if nodes["ready"].(float64) != 1 {
		t.Errorf("Expected 1 ready node, got %v", nodes["ready"])
	}

	// Check pods
	pods := body["pods"].(map[string]interface{})
	if pods["total"].(float64) != 2 {
		t.Errorf("Expected 2 total pods, got %v", pods["total"])
	}

	// Check deployments
	deployments := body["deployments"].(map[string]interface{})
	if deployments["total"].(float64) != 1 {
		t.Errorf("Expected 1 total deployment, got %v", deployments["total"])
	}

	// Check namespaces
	if body["namespaces"].(float64) != 2 {
		t.Errorf("Expected 2 namespaces, got %v", body["namespaces"])
	}
}

// TestWorkloadPods tests the workload pods endpoint
func TestWorkloadPods(t *testing.T) {
	dbPath := "test_workload_pods.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	tests := []struct {
		name           string
		query          string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "Get deployment pods",
			query:          "namespace=default&kind=deployment&name=test-deployment",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "Get statefulset pods",
			query:          "namespace=default&kind=statefulset&name=test-statefulset",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "Get daemonset pods",
			query:          "namespace=default&kind=daemonset&name=test-daemonset",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "Get replicaset pods",
			query:          "namespace=default&kind=replicaset&name=test-replicaset",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "Missing namespace",
			query:          "kind=deployment&name=test-deployment",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "Missing kind",
			query:          "namespace=default&name=test-deployment",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "Missing name",
			query:          "namespace=default&kind=deployment",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "Unknown kind",
			query:          "namespace=default&kind=unknown&name=test",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "Non-existent deployment",
			query:          "namespace=default&kind=deployment&name=nonexistent",
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/workload/pods?"+tt.query, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestWorkloadPodsMethodNotAllowed tests that non-GET requests are rejected
func TestWorkloadPodsMethodNotAllowed(t *testing.T) {
	dbPath := "test_workload_pods_method.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/workload/pods?namespace=default&kind=deployment&name=test-deployment", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// TestClusterOverviewMethodNotAllowed tests that non-GET requests are rejected
func TestClusterOverviewMethodNotAllowed(t *testing.T) {
	dbPath := "test_overview_method.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/overview", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// TestK8sResourceWithXUsernameHeader tests that X-Username header is used for audit logging
func TestK8sResourceWithXUsernameHeader(t *testing.T) {
	dbPath := "test_k8s_username.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/k8s/pods", nil)
	req.Header.Set("X-Username", "testuser")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestK8sResourceResponseTimestamp tests that responses include timestamps
func TestK8sResourceResponseTimestamp(t *testing.T) {
	dbPath := "test_k8s_timestamp.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	_, mux := setupK8sTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/k8s/pods", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if body["timestamp"] == nil {
		t.Error("Expected timestamp in response")
	}
}

// TestHandleGlobalSearch tests the global search endpoint
func TestHandleGlobalSearch(t *testing.T) {
	server, mux := setupK8sTestServer(t)
	defer server.Stop()

	tests := []struct {
		name           string
		query          string
		namespace      string
		expectedStatus int
		checkResults   bool
	}{
		{
			name:           "Search for nginx",
			query:          "nginx",
			namespace:      "",
			expectedStatus: http.StatusOK,
			checkResults:   true,
		},
		{
			name:           "Search in specific namespace",
			query:          "test",
			namespace:      "default",
			expectedStatus: http.StatusOK,
			checkResults:   true,
		},
		{
			name:           "Empty query",
			query:          "",
			namespace:      "",
			expectedStatus: http.StatusBadRequest,
			checkResults:   false,
		},
		{
			name:           "Search with no results",
			query:          "nonexistent-xyz-12345",
			namespace:      "",
			expectedStatus: http.StatusOK,
			checkResults:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/search?q=" + tt.query
			if tt.namespace != "" {
				url += "&namespace=" + tt.namespace
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResults && w.Code == http.StatusOK {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if response["results"] == nil {
					t.Error("Expected results in response")
				}
				if response["query"] != tt.query {
					t.Errorf("Expected query %s, got %v", tt.query, response["query"])
				}
			}
		})
	}
}

// TestHandleGlobalSearch_MethodNotAllowed tests that non-GET requests are rejected
func TestHandleGlobalSearch_MethodNotAllowed(t *testing.T) {
	server, mux := setupK8sTestServer(t)
	defer server.Stop()

	req := httptest.NewRequest(http.MethodPost, "/api/search?q=test", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}
