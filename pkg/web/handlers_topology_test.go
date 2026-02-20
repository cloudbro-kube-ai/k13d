package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/cloudbro-kube-ai/k13d/pkg/db"
	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

// TestHandleTopology tests the /api/topology HTTP handler.
func TestHandleTopology(t *testing.T) {
	dbPath := "test_topology.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	replicas := int32(2)
	deployUID := types.UID("deploy-uid-123")
	pathType := networkingv1.PathTypePrefix
	minReplicas := int32(1)

	fakeClientset := fake.NewSimpleClientset( //nolint:staticcheck
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "web-deploy", Namespace: "default",
				UID:    deployUID,
				Labels: map[string]string{"app.kubernetes.io/name": "web-app"},
			},
			Spec:   appsv1.DeploymentSpec{Replicas: &replicas},
			Status: appsv1.DeploymentStatus{ReadyReplicas: 2},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "web-pod-1", Namespace: "default",
				Labels: map[string]string{"app": "web", "app.kubernetes.io/name": "web-app"},
				OwnerReferences: []metav1.OwnerReference{
					{UID: deployUID, Kind: "ReplicaSet", Name: "web-deploy-abc"},
				},
			},
			Spec:   corev1.PodSpec{NodeName: "node-1"},
			Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{Ready: true}}},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "web-svc", Namespace: "default",
				Labels: map[string]string{"app.kubernetes.io/name": "web-app"},
			},
			Spec: corev1.ServiceSpec{
				Type:      corev1.ServiceTypeClusterIP,
				ClusterIP: "10.96.0.10",
				Selector:  map[string]string{"app": "web"},
				Ports:     []corev1.ServicePort{{Port: 80, Protocol: corev1.ProtocolTCP}},
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: "web-ingress", Namespace: "default"},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{
						Host: "example.com",
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{Path: "/", PathType: &pathType, Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "web-svc", Port: networkingv1.ServiceBackendPort{Number: 80}}}},
								},
							},
						},
					},
				},
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "app-config", Namespace: "default"},
			Data:       map[string]string{"key": "value"},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "app-secret", Namespace: "default"},
			Type:       corev1.SecretTypeOpaque,
			Data:       map[string][]byte{"pass": []byte("s3cret")},
		},
		&autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{Name: "web-hpa", Namespace: "default"},
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{Kind: "Deployment", Name: "web-deploy"},
				MinReplicas:    &minReplicas,
				MaxReplicas:    5,
			},
			Status: autoscalingv2.HorizontalPodAutoscalerStatus{CurrentReplicas: 2},
		},
	)

	server := &Server{
		cfg:              &config.Config{Language: "en"},
		k8sClient:        &k8s.Client{Clientset: fakeClientset},
		authManager:      NewAuthManager(&AuthConfig{Enabled: false, SessionDuration: time.Hour, AuthMode: "local", DefaultAdmin: "admin", DefaultPassword: "admin123", Quiet: true}),
		port:             8080,
		pendingApprovals: make(map[string]*PendingToolApproval),
	}

	t.Run("GET returns topology JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/topology?namespace=default", nil)
		w := httptest.NewRecorder()

		server.handleTopology(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp TopologyResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify we have nodes
		if len(resp.Nodes) == 0 {
			t.Fatal("Expected at least one topology node")
		}

		// Build a node map for assertions
		nodeMap := make(map[string]TopologyNode)
		for _, n := range resp.Nodes {
			nodeMap[n.ID] = n
		}

		// Check Deployment node exists
		deployID := "Deployment/default/web-deploy"
		if n, ok := nodeMap[deployID]; !ok {
			t.Errorf("Expected Deployment node %s", deployID)
		} else {
			if n.Status != "running" {
				t.Errorf("Expected Deployment status 'running', got %q", n.Status)
			}
			if n.Group != "web-app" {
				t.Errorf("Expected Group 'web-app', got %q", n.Group)
			}
		}

		// Check Pod node exists
		podID := "Pod/default/web-pod-1"
		if _, ok := nodeMap[podID]; !ok {
			t.Errorf("Expected Pod node %s", podID)
		}

		// Check Service node exists
		svcID := "Service/default/web-svc"
		if n, ok := nodeMap[svcID]; !ok {
			t.Errorf("Expected Service node %s", svcID)
		} else if n.Group != "web-app" {
			t.Errorf("Expected Service Group 'web-app', got %q", n.Group)
		}

		// Check Ingress node exists
		ingID := "Ingress/default/web-ingress"
		if _, ok := nodeMap[ingID]; !ok {
			t.Errorf("Expected Ingress node %s", ingID)
		}

		// Check ConfigMap and Secret nodes
		cmID := "ConfigMap/default/app-config"
		if _, ok := nodeMap[cmID]; !ok {
			t.Errorf("Expected ConfigMap node %s", cmID)
		}
		secID := "Secret/default/app-secret"
		if _, ok := nodeMap[secID]; !ok {
			t.Errorf("Expected Secret node %s", secID)
		}

		// Check HPA node
		hpaID := "HPA/default/web-hpa"
		if _, ok := nodeMap[hpaID]; !ok {
			t.Errorf("Expected HPA node %s", hpaID)
		}

		// Verify edges exist
		if len(resp.Edges) == 0 {
			t.Error("Expected at least one topology edge")
		}

		// Build edge set for lookup
		edgeSet := make(map[string]string)
		for _, e := range resp.Edges {
			edgeSet[e.Source+"->"+e.Target] = e.Type
		}

		// Service should select the Pod
		svcToPod := svcID + "->" + podID
		if typ, ok := edgeSet[svcToPod]; !ok {
			t.Errorf("Expected edge %s", svcToPod)
		} else if typ != "selects" {
			t.Errorf("Expected edge type 'selects', got %q", typ)
		}

		// Ingress should route to Service
		ingToSvc := ingID + "->" + svcID
		if typ, ok := edgeSet[ingToSvc]; !ok {
			t.Errorf("Expected edge %s", ingToSvc)
		} else if typ != "routes" {
			t.Errorf("Expected edge type 'routes', got %q", typ)
		}

		// HPA should scale Deployment
		hpaToDepl := hpaID + "->" + deployID
		if typ, ok := edgeSet[hpaToDepl]; !ok {
			t.Errorf("Expected edge %s", hpaToDepl)
		} else if typ != "scales" {
			t.Errorf("Expected edge type 'scales', got %q", typ)
		}
	})

	t.Run("POST method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/topology?namespace=default", nil)
		w := httptest.NewRecorder()

		server.handleTopology(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})

	t.Run("empty namespace returns all namespaces", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/topology", nil)
		w := httptest.NewRecorder()

		server.handleTopology(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp TopologyResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// With empty namespace, buildTopology uses "" which lists all namespaces
		if len(resp.Nodes) == 0 {
			t.Error("Expected nodes when querying all namespaces")
		}
	})
}

// TestBuildTopology tests the topology building logic in detail.
func TestBuildTopology(t *testing.T) {
	dbPath := "test_build_topology.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	replicas := int32(2)
	deployUID := types.UID("deploy-uid-1")
	rsUID := types.UID("rs-uid-1")

	fakeClientset := fake.NewSimpleClientset( //nolint:staticcheck
		// Deployment -> ReplicaSet -> Pod chain
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "api-deploy", Namespace: "prod",
				UID:    deployUID,
				Labels: map[string]string{"app.kubernetes.io/name": "api"},
			},
			Spec:   appsv1.DeploymentSpec{Replicas: &replicas},
			Status: appsv1.DeploymentStatus{ReadyReplicas: 2},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "api-deploy-rs1", Namespace: "prod",
				UID: rsUID,
				OwnerReferences: []metav1.OwnerReference{
					{UID: deployUID, Kind: "Deployment", Name: "api-deploy"},
				},
			},
			Status: appsv1.ReplicaSetStatus{Replicas: 2, ReadyReplicas: 2},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "api-pod-1", Namespace: "prod",
				Labels: map[string]string{"app": "api"},
				OwnerReferences: []metav1.OwnerReference{
					{UID: rsUID, Kind: "ReplicaSet", Name: "api-deploy-rs1"},
				},
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{Name: "cfg", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "api-config"}}}},
					{Name: "sec", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "api-secret"}}},
				},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{Ready: true}}},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "api-config", Namespace: "prod"},
			Data:       map[string]string{"k": "v"},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "api-secret", Namespace: "prod"},
			Type:       corev1.SecretTypeOpaque,
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "api-svc", Namespace: "prod"},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{"app": "api"},
				Ports:    []corev1.ServicePort{{Port: 8080, Protocol: corev1.ProtocolTCP}},
			},
		},
		// StatefulSet
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "db-sts", Namespace: "prod",
				Labels: map[string]string{"app.kubernetes.io/name": "database"},
			},
			Spec:   appsv1.StatefulSetSpec{Replicas: &replicas},
			Status: appsv1.StatefulSetStatus{ReadyReplicas: 1},
		},
		// DaemonSet
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "log-ds", Namespace: "prod",
				Labels: map[string]string{"app.kubernetes.io/name": "logging"},
			},
			Status: appsv1.DaemonSetStatus{DesiredNumberScheduled: 3, NumberReady: 3},
		},
	)

	server := &Server{
		cfg:              &config.Config{Language: "en"},
		k8sClient:        &k8s.Client{Clientset: fakeClientset},
		pendingApprovals: make(map[string]*PendingToolApproval),
	}

	resp, err := server.buildTopology(context.Background(), "prod")
	if err != nil {
		t.Fatalf("buildTopology failed: %v", err)
	}

	// Build node/edge maps
	nodeMap := make(map[string]TopologyNode)
	for _, n := range resp.Nodes {
		nodeMap[n.ID] = n
	}
	edgeSet := make(map[string]string)
	for _, e := range resp.Edges {
		edgeSet[e.Source+"->"+e.Target] = e.Type
	}

	t.Run("Deployment node", func(t *testing.T) {
		n, ok := nodeMap["Deployment/prod/api-deploy"]
		if !ok {
			t.Fatal("Expected Deployment node")
		}
		if n.Group != "api" {
			t.Errorf("Expected Group 'api', got %q", n.Group)
		}
		if n.Info["replicas"] != "2/2" {
			t.Errorf("Expected replicas '2/2', got %q", n.Info["replicas"])
		}
	})

	t.Run("ReplicaSet node and owns edge", func(t *testing.T) {
		rsID := "ReplicaSet/prod/api-deploy-rs1"
		if _, ok := nodeMap[rsID]; !ok {
			t.Fatal("Expected ReplicaSet node")
		}
		// Deployment owns ReplicaSet
		ownsEdge := "Deployment/prod/api-deploy->" + rsID
		if typ, ok := edgeSet[ownsEdge]; !ok {
			t.Errorf("Expected owns edge from Deployment to ReplicaSet")
		} else if typ != "owns" {
			t.Errorf("Expected edge type 'owns', got %q", typ)
		}
	})

	t.Run("Pod owns edge from ReplicaSet", func(t *testing.T) {
		podID := "Pod/prod/api-pod-1"
		rsID := "ReplicaSet/prod/api-deploy-rs1"
		ownsEdge := rsID + "->" + podID
		if typ, ok := edgeSet[ownsEdge]; !ok {
			t.Errorf("Expected owns edge from ReplicaSet to Pod")
		} else if typ != "owns" {
			t.Errorf("Expected type 'owns', got %q", typ)
		}
	})

	t.Run("Pod mounts ConfigMap and Secret", func(t *testing.T) {
		podID := "Pod/prod/api-pod-1"
		cmID := "ConfigMap/prod/api-config"
		secID := "Secret/prod/api-secret"

		if typ, ok := edgeSet[podID+"->"+cmID]; !ok {
			t.Error("Expected mounts edge to ConfigMap")
		} else if typ != "mounts" {
			t.Errorf("Expected type 'mounts', got %q", typ)
		}

		if typ, ok := edgeSet[podID+"->"+secID]; !ok {
			t.Error("Expected mounts edge to Secret")
		} else if typ != "mounts" {
			t.Errorf("Expected type 'mounts', got %q", typ)
		}
	})

	t.Run("Service selects Pod", func(t *testing.T) {
		svcID := "Service/prod/api-svc"
		podID := "Pod/prod/api-pod-1"
		if typ, ok := edgeSet[svcID+"->"+podID]; !ok {
			t.Error("Expected selects edge from Service to Pod")
		} else if typ != "selects" {
			t.Errorf("Expected type 'selects', got %q", typ)
		}
	})

	t.Run("StatefulSet node with Group", func(t *testing.T) {
		n, ok := nodeMap["StatefulSet/prod/db-sts"]
		if !ok {
			t.Fatal("Expected StatefulSet node")
		}
		if n.Group != "database" {
			t.Errorf("Expected Group 'database', got %q", n.Group)
		}
		if n.Status != "pending" {
			t.Errorf("Expected status 'pending' (1/2 ready), got %q", n.Status)
		}
	})

	t.Run("DaemonSet node with Group", func(t *testing.T) {
		n, ok := nodeMap["DaemonSet/prod/log-ds"]
		if !ok {
			t.Fatal("Expected DaemonSet node")
		}
		if n.Group != "logging" {
			t.Errorf("Expected Group 'logging', got %q", n.Group)
		}
		if n.Status != "running" {
			t.Errorf("Expected status 'running', got %q", n.Status)
		}
	})
}

// TestBuildTopology_EmptyNamespace tests topology with no resources.
func TestBuildTopology_EmptyNamespace(t *testing.T) {
	dbPath := "test_topology_empty.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	fakeClientset := fake.NewSimpleClientset() //nolint:staticcheck
	server := &Server{
		cfg:              &config.Config{Language: "en"},
		k8sClient:        &k8s.Client{Clientset: fakeClientset},
		pendingApprovals: make(map[string]*PendingToolApproval),
	}

	resp, err := server.buildTopology(context.Background(), "empty-ns")
	if err != nil {
		t.Fatalf("buildTopology failed: %v", err)
	}

	if len(resp.Nodes) != 0 {
		t.Errorf("Expected 0 nodes for empty namespace, got %d", len(resp.Nodes))
	}
	if len(resp.Edges) != 0 {
		t.Errorf("Expected 0 edges for empty namespace, got %d", len(resp.Edges))
	}
}

// TestLabelsMatch tests the label matching helper.
func TestLabelsMatch(t *testing.T) {
	tests := []struct {
		name     string
		selector map[string]string
		labels   map[string]string
		want     bool
	}{
		{"exact match", map[string]string{"app": "web"}, map[string]string{"app": "web"}, true},
		{"subset match", map[string]string{"app": "web"}, map[string]string{"app": "web", "tier": "frontend"}, true},
		{"mismatch", map[string]string{"app": "web"}, map[string]string{"app": "api"}, false},
		{"missing key", map[string]string{"app": "web"}, map[string]string{"tier": "frontend"}, false},
		{"empty selector matches all", map[string]string{}, map[string]string{"app": "web"}, true},
		{"nil labels", map[string]string{"app": "web"}, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := labelsMatch(tt.selector, tt.labels)
			if got != tt.want {
				t.Errorf("labelsMatch(%v, %v) = %v, want %v", tt.selector, tt.labels, got, tt.want)
			}
		})
	}
}

// TestPodStatus tests the podStatus helper function.
func TestPodStatus(t *testing.T) {
	tests := []struct {
		name string
		pod  corev1.Pod
		want string
	}{
		{
			"running healthy",
			corev1.Pod{
				Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{Ready: true}}},
			},
			"running",
		},
		{
			"running but not ready",
			corev1.Pod{
				Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{Ready: false}}},
			},
			"pending",
		},
		{
			"crashloopbackoff",
			corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}},
					},
				},
			},
			"failed",
		},
		{
			"pending",
			corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending}},
			"pending",
		},
		{
			"succeeded",
			corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodSucceeded}},
			"succeeded",
		},
		{
			"failed",
			corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodFailed}},
			"failed",
		},
		{
			"unknown",
			corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPhase("Unknown")}},
			"unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := podStatus(tt.pod)
			if got != tt.want {
				t.Errorf("podStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestNodeID tests the nodeID helper.
func TestNodeID(t *testing.T) {
	got := nodeID("Deployment", "default", "nginx")
	want := "Deployment/default/nginx"
	if got != want {
		t.Errorf("nodeID() = %q, want %q", got, want)
	}
}
