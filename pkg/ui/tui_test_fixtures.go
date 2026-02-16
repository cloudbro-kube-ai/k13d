package ui

import (
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestAppConfig holds configuration for creating test App instances.
type TestAppConfig struct {
	// UseSimulationScreen enables headless testing with tcell.SimulationScreen
	UseSimulationScreen bool
	// Screen is the simulation screen to use (if UseSimulationScreen is true)
	Screen tcell.SimulationScreen
	// InitialResource sets the starting resource view (default: "pods")
	InitialResource string
	// InitialNamespace sets the starting namespace (default: "default")
	InitialNamespace string
	// SkipBackgroundLoading prevents loadAPIResources and loadNamespaces goroutines
	SkipBackgroundLoading bool
	// SkipBriefing disables the briefing panel to prevent pulse animation blocking in tests
	SkipBriefing bool
}

// NewTestApp creates a minimal App instance suitable for testing.
// It uses a fake K8s clientset and can optionally use a simulation screen.
func NewTestApp(cfg TestAppConfig) *App {
	fakeClientset := CreateFakeClientset()

	k8sClient := &k8s.Client{
		Clientset: fakeClientset,
	}

	// Default values
	if cfg.InitialResource == "" {
		cfg.InitialResource = "pods"
	}
	if cfg.InitialNamespace == "" {
		cfg.InitialNamespace = "default"
	}

	// Create tview application
	tvApp := tview.NewApplication()
	if cfg.UseSimulationScreen && cfg.Screen != nil {
		tvApp.SetScreen(cfg.Screen)
	}

	// Silent logger for tests
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError, // Only show errors in tests
	}))

	app := &App{
		Application:         tvApp,
		config:              config.NewDefaultConfig(),
		k8s:                 k8sClient,
		aiClient:            nil, // No AI in tests
		currentResource:     cfg.InitialResource,
		currentNamespace:    cfg.InitialNamespace,
		namespaces:          []string{"", "default", "kube-system", "kube-public"},
		recentNamespaces:    make([]string, 0),
		maxRecentNamespaces: 9,
		showAIPanel:         false, // Disabled for simpler testing
		selectedRows:        make(map[int]bool),
		sortColumn:          -1,
		sortAscending:       true,
		pendingToolApproval: make(chan bool, 1),
		logger:              logger,
		mx:                  sync.RWMutex{},
		navMx:               sync.Mutex{},
		aiMx:                sync.RWMutex{},
		cancelLock:          sync.Mutex{},
		watchMu:             sync.Mutex{},
		skipBriefing:        cfg.SkipBriefing, // Disable briefing to prevent pulse animation blocking
		styles:              config.DefaultStyles(),
	}

	app.setupUI()
	app.setupKeybindings()

	// Optionally skip background loading for faster tests
	if !cfg.SkipBackgroundLoading {
		go app.loadAPIResources()
		go app.loadNamespaces()
	}

	return app
}

// CreateFakeClientset creates a fake Kubernetes clientset populated with test data.
// This provides a realistic set of resources for TUI testing.
func CreateFakeClientset() *fake.Clientset {
	return fake.NewSimpleClientset(
		// Namespaces
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "default"},
			Status:     corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "kube-system"},
			Status:     corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "kube-public"},
			Status:     corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
		},

		// Pods - various states
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-pod",
				Namespace: "default",
				Labels:    map[string]string{"app": "nginx"},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "nginx", Image: "nginx:1.21"},
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{Name: "nginx", Ready: true, RestartCount: 0},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "redis-pod",
				Namespace: "default",
				Labels:    map[string]string{"app": "redis"},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "redis", Image: "redis:7"},
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{Name: "redis", Ready: true, RestartCount: 2},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "coredns-pod",
				Namespace: "kube-system",
				Labels:    map[string]string{"k8s-app": "kube-dns"},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "coredns", Image: "coredns/coredns:1.9"},
				},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failing-pod",
				Namespace: "default",
				Labels:    map[string]string{"app": "broken"},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodFailed,
				ContainerStatuses: []corev1.ContainerStatus{
					{Name: "app", Ready: false, RestartCount: 5},
				},
			},
		},

		// Nodes
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
				Allocatable: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("8Gi"),
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "node-2"},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
				Allocatable: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("8"),
					corev1.ResourceMemory: resource.MustParse("16Gi"),
				},
			},
		},

		// Deployments
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-deployment",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(3),
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          3,
				ReadyReplicas:     3,
				AvailableReplicas: 3,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "redis-deployment",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(1),
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				ReadyReplicas:     1,
				AvailableReplicas: 1,
			},
		},

		// Services
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-service",
				Namespace: "default",
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					{Port: 80, Protocol: corev1.ProtocolTCP},
				},
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "redis-service",
				Namespace: "default",
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					{Port: 6379, Protocol: corev1.ProtocolTCP},
				},
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubernetes",
				Namespace: "default",
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					{Port: 443, Protocol: corev1.ProtocolTCP},
				},
			},
		},

		// Events
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-pod.event1",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind:      "Pod",
				Name:      "nginx-pod",
				Namespace: "default",
			},
			Reason:  "Scheduled",
			Message: "Successfully assigned default/nginx-pod to node-1",
			Type:    "Normal",
		},
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failing-pod.event1",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind:      "Pod",
				Name:      "failing-pod",
				Namespace: "default",
			},
			Reason:  "BackOff",
			Message: "Back-off restarting failed container",
			Type:    "Warning",
		},

		// ConfigMaps
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-config",
				Namespace: "default",
			},
			Data: map[string]string{
				"nginx.conf": "server { listen 80; }",
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "coredns-config",
				Namespace: "kube-system",
			},
			Data: map[string]string{
				"Corefile": ".:53 { forward . /etc/resolv.conf }",
			},
		},

		// Secrets
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "app-secret",
				Namespace: "default",
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"password": []byte("secret123"),
			},
		},

		// StatefulSets
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "redis-statefulset",
				Namespace: "default",
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: int32Ptr(3),
			},
			Status: appsv1.StatefulSetStatus{
				Replicas:      3,
				ReadyReplicas: 3,
			},
		},

		// DaemonSets
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "node-exporter",
				Namespace: "kube-system",
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 2,
				CurrentNumberScheduled: 2,
				NumberReady:            2,
			},
		},

		// ReplicaSets
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-deployment-abc123",
				Namespace: "default",
			},
			Spec: appsv1.ReplicaSetSpec{
				Replicas: int32Ptr(3),
			},
			Status: appsv1.ReplicaSetStatus{
				Replicas:      3,
				ReadyReplicas: 3,
			},
		},

		// Jobs
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "backup-job",
				Namespace: "default",
			},
			Status: batchv1.JobStatus{
				Succeeded: 1,
			},
		},

		// CronJobs
		&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cleanup-cronjob",
				Namespace: "default",
			},
			Spec: batchv1.CronJobSpec{
				Schedule: "0 * * * *",
			},
		},

		// Ingresses
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-ingress",
				Namespace: "default",
			},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{Host: "nginx.example.com"},
				},
			},
		},

		// RBAC Resources
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default",
				Namespace: "default",
			},
		},
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-reader",
				Namespace: "default",
			},
		},
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "read-pods",
				Namespace: "default",
			},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster-admin",
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster-admin-binding",
			},
		},
	)
}

// int32Ptr returns a pointer to an int32 value.
func int32Ptr(i int32) *int32 {
	return &i
}

// CreateMinimalTestApp creates the most minimal App instance for unit tests.
// This is useful for testing individual methods without full TUI setup.
func CreateMinimalTestApp() *App {
	return &App{
		currentResource:     "pods",
		currentNamespace:    "default",
		namespaces:          []string{"", "default", "kube-system"},
		recentNamespaces:    make([]string, 0),
		maxRecentNamespaces: 9,
		selectedRows:        make(map[int]bool),
		sortColumn:          -1,
		sortAscending:       true,
		mx:                  sync.RWMutex{},
		navMx:               sync.Mutex{},
		aiMx:                sync.RWMutex{},
		cancelLock:          sync.Mutex{},
		watchMu:             sync.Mutex{},
		pendingToolApproval: make(chan bool, 1),
		logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}
