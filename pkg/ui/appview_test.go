package ui

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildAppGroups_WithLabels(t *testing.T) {
	pods := []corev1.Pod{
		{
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
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-pod-2",
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
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-pod-3",
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
	}

	deployments := []appsv1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx",
				Namespace: "default",
				Labels: map[string]string{
					"app.kubernetes.io/name":    "nginx",
					"app.kubernetes.io/version": "1.21",
				},
			},
			Spec: appsv1.DeploymentSpec{Replicas: int32Ptr(3)},
		},
	}

	services := []corev1.Service{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-svc",
				Namespace: "default",
				Labels:    map[string]string{"app.kubernetes.io/name": "nginx"},
			},
		},
	}

	configMaps := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nginx-config",
				Namespace: "default",
				Labels:    map[string]string{"app.kubernetes.io/name": "nginx"},
			},
		},
	}

	groups := BuildAppGroups(pods, deployments, nil, nil, services, configMaps, nil, nil)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	g := groups[0]
	if g.Name != "nginx" {
		t.Errorf("expected name 'nginx', got %q", g.Name)
	}
	if g.Version != "1.21" {
		t.Errorf("expected version '1.21', got %q", g.Version)
	}
	if g.PodCount != 3 {
		t.Errorf("expected 3 pods, got %d", g.PodCount)
	}
	if g.ReadyPods != 3 {
		t.Errorf("expected 3 ready pods, got %d", g.ReadyPods)
	}
	if g.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %q", g.Status)
	}
	if len(g.Resources["Deployment"]) != 1 {
		t.Errorf("expected 1 deployment, got %d", len(g.Resources["Deployment"]))
	}
	if len(g.Resources["Service"]) != 1 {
		t.Errorf("expected 1 service, got %d", len(g.Resources["Service"]))
	}
	if len(g.Resources["ConfigMap"]) != 1 {
		t.Errorf("expected 1 configmap, got %d", len(g.Resources["ConfigMap"]))
	}
}

func TestBuildAppGroups_Ungrouped(t *testing.T) {
	configMaps := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-root-ca.crt",
				Namespace: "default",
			},
		},
	}

	secrets := []corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default-token-abc",
				Namespace: "default",
			},
		},
	}

	groups := BuildAppGroups(nil, nil, nil, nil, nil, configMaps, secrets, nil)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group (ungrouped), got %d", len(groups))
	}

	g := groups[0]
	if g.Name != "Ungrouped Resources" {
		t.Errorf("expected name 'Ungrouped Resources', got %q", g.Name)
	}
	if len(g.Resources["ConfigMap"]) != 1 {
		t.Errorf("expected 1 ungrouped configmap, got %d", len(g.Resources["ConfigMap"]))
	}
	if len(g.Resources["Secret"]) != 1 {
		t.Errorf("expected 1 ungrouped secret, got %d", len(g.Resources["Secret"]))
	}
}

func TestBuildAppGroups_HealthStatus(t *testing.T) {
	tests := []struct {
		name           string
		pods           []corev1.Pod
		deployments    []appsv1.Deployment
		expectedStatus string
	}{
		{
			name: "healthy - all pods ready",
			pods: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "app-1",
						Labels: map[string]string{"app.kubernetes.io/name": "myapp"},
					},
					Status: corev1.PodStatus{
						Phase:             corev1.PodRunning,
						ContainerStatuses: []corev1.ContainerStatus{{Ready: true}},
					},
				},
			},
			deployments: []appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "myapp",
						Labels: map[string]string{"app.kubernetes.io/name": "myapp"},
					},
				},
			},
			expectedStatus: "healthy",
		},
		{
			name: "degraded - some pods not ready",
			pods: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "app-1",
						Labels: map[string]string{"app.kubernetes.io/name": "myapp"},
					},
					Status: corev1.PodStatus{
						Phase:             corev1.PodRunning,
						ContainerStatuses: []corev1.ContainerStatus{{Ready: true}},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "app-2",
						Labels: map[string]string{"app.kubernetes.io/name": "myapp"},
					},
					Status: corev1.PodStatus{
						Phase:             corev1.PodPending,
						ContainerStatuses: []corev1.ContainerStatus{{Ready: false}},
					},
				},
			},
			deployments: []appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "myapp",
						Labels: map[string]string{"app.kubernetes.io/name": "myapp"},
					},
				},
			},
			expectedStatus: "degraded",
		},
		{
			name: "failing - no pods ready",
			pods: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "app-1",
						Labels: map[string]string{"app.kubernetes.io/name": "myapp"},
					},
					Status: corev1.PodStatus{
						Phase:             corev1.PodFailed,
						ContainerStatuses: []corev1.ContainerStatus{{Ready: false}},
					},
				},
			},
			deployments: []appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "myapp",
						Labels: map[string]string{"app.kubernetes.io/name": "myapp"},
					},
				},
			},
			expectedStatus: "failing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := BuildAppGroups(tt.pods, tt.deployments, nil, nil, nil, nil, nil, nil)
			if len(groups) != 1 {
				t.Fatalf("expected 1 group, got %d", len(groups))
			}
			if groups[0].Status != tt.expectedStatus {
				t.Errorf("expected status %q, got %q", tt.expectedStatus, groups[0].Status)
			}
		})
	}
}

func TestBuildAppGroups_MultipleApps(t *testing.T) {
	deployments := []appsv1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "nginx",
				Labels: map[string]string{"app.kubernetes.io/name": "nginx"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "api-server",
				Labels: map[string]string{"app.kubernetes.io/name": "api-server"},
			},
		},
	}

	statefulSets := []appsv1.StatefulSet{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "redis",
				Labels: map[string]string{"app.kubernetes.io/name": "redis"},
			},
		},
	}

	services := []corev1.Service{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "nginx-svc",
				Labels: map[string]string{"app.kubernetes.io/name": "nginx"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "api-svc",
				Labels: map[string]string{"app.kubernetes.io/name": "api-server"},
			},
		},
	}

	ingresses := []networkingv1.Ingress{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "nginx-ingress",
				Labels: map[string]string{"app.kubernetes.io/name": "nginx"},
			},
		},
	}

	groups := BuildAppGroups(nil, deployments, statefulSets, nil, services, nil, nil, ingresses)

	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}

	// Groups should be sorted by name
	if groups[0].Name != "api-server" {
		t.Errorf("expected first group 'api-server', got %q", groups[0].Name)
	}
	if groups[1].Name != "nginx" {
		t.Errorf("expected second group 'nginx', got %q", groups[1].Name)
	}
	if groups[2].Name != "redis" {
		t.Errorf("expected third group 'redis', got %q", groups[2].Name)
	}

	// Check nginx has deployment + service + ingress
	nginx := groups[1]
	if len(nginx.Resources["Deployment"]) != 1 {
		t.Errorf("nginx: expected 1 deployment, got %d", len(nginx.Resources["Deployment"]))
	}
	if len(nginx.Resources["Service"]) != 1 {
		t.Errorf("nginx: expected 1 service, got %d", len(nginx.Resources["Service"]))
	}
	if len(nginx.Resources["Ingress"]) != 1 {
		t.Errorf("nginx: expected 1 ingress, got %d", len(nginx.Resources["Ingress"]))
	}

	// Check redis has statefulset
	redis := groups[2]
	if len(redis.Resources["StatefulSet"]) != 1 {
		t.Errorf("redis: expected 1 statefulset, got %d", len(redis.Resources["StatefulSet"]))
	}
}

func TestBuildAppGroups_Empty(t *testing.T) {
	groups := BuildAppGroups(nil, nil, nil, nil, nil, nil, nil, nil)
	if len(groups) != 0 {
		t.Errorf("expected 0 groups for empty input, got %d", len(groups))
	}
}

func TestBuildAppGroups_MixedGroupedAndUngrouped(t *testing.T) {
	deployments := []appsv1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "nginx",
				Labels: map[string]string{"app.kubernetes.io/name": "nginx"},
			},
		},
	}

	configMaps := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "nginx-config",
				Labels: map[string]string{"app.kubernetes.io/name": "nginx"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-root-ca.crt",
				// No app label
			},
		},
	}

	groups := BuildAppGroups(nil, deployments, nil, nil, nil, configMaps, nil, nil)

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups (nginx + ungrouped), got %d", len(groups))
	}

	// First should be nginx (sorted), second ungrouped (always last)
	if groups[0].Name != "nginx" {
		t.Errorf("expected first group 'nginx', got %q", groups[0].Name)
	}
	if groups[1].Name != "Ungrouped Resources" {
		t.Errorf("expected second group 'Ungrouped Resources', got %q", groups[1].Name)
	}
}

func TestAppGroupStatusText(t *testing.T) {
	tests := []struct {
		status   string
		contains string
	}{
		{"healthy", "green"},
		{"degraded", "yellow"},
		{"failing", "red"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := appGroupStatusText(tt.status)
			if result == "" {
				t.Errorf("expected non-empty result for status %q", tt.status)
			}
		})
	}
}

func TestBuildAppGroups_DaemonSets(t *testing.T) {
	daemonSets := []appsv1.DaemonSet{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "fluentd",
				Labels: map[string]string{"app.kubernetes.io/name": "logging"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-exporter",
				// No app label
			},
		},
	}

	groups := BuildAppGroups(nil, nil, nil, daemonSets, nil, nil, nil, nil)

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	if groups[0].Name != "logging" {
		t.Errorf("expected first group 'logging', got %q", groups[0].Name)
	}
	if len(groups[0].Resources["DaemonSet"]) != 1 {
		t.Errorf("expected 1 daemonset in logging group")
	}

	if groups[1].Name != "Ungrouped Resources" {
		t.Errorf("expected ungrouped, got %q", groups[1].Name)
	}
}
