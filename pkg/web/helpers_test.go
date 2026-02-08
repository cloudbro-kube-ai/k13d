package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc..."},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestGetPodReadyCount(t *testing.T) {
	tests := []struct {
		name     string
		pod      *corev1.Pod
		expected string
	}{
		{
			name: "all ready",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{Ready: true},
						{Ready: true},
					},
				},
			},
			expected: "2/2",
		},
		{
			name: "partial ready",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{Ready: true},
						{Ready: false},
						{Ready: true},
					},
				},
			},
			expected: "2/3",
		},
		{
			name: "none ready",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{Ready: false},
						{Ready: false},
					},
				},
			},
			expected: "0/2",
		},
		{
			name: "no containers",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{},
				},
			},
			expected: "0/0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPodReadyCount(tt.pod)
			if result != tt.expected {
				t.Errorf("getPodReadyCount() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestGetPodRestarts(t *testing.T) {
	tests := []struct {
		name     string
		pod      *corev1.Pod
		expected int32
	}{
		{
			name: "no restarts",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{RestartCount: 0},
						{RestartCount: 0},
					},
				},
			},
			expected: 0,
		},
		{
			name: "some restarts",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{RestartCount: 3},
						{RestartCount: 5},
					},
				},
			},
			expected: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPodRestarts(tt.pod)
			if result != tt.expected {
				t.Errorf("getPodRestarts() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestGetExternalIPs(t *testing.T) {
	tests := []struct {
		name     string
		svc      *corev1.Service
		expected string
	}{
		{
			name: "load balancer IP",
			svc: &corev1.Service{
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{IP: "10.0.0.1"},
						},
					},
				},
			},
			expected: "10.0.0.1",
		},
		{
			name: "load balancer hostname",
			svc: &corev1.Service{
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{Hostname: "my-lb.example.com"},
						},
					},
				},
			},
			expected: "my-lb.example.com",
		},
		{
			name: "external IPs",
			svc: &corev1.Service{
				Spec: corev1.ServiceSpec{
					ExternalIPs: []string{"192.168.1.1", "192.168.1.2"},
				},
			},
			expected: "192.168.1.1, 192.168.1.2",
		},
		{
			name: "no external access",
			svc: &corev1.Service{
				Spec:   corev1.ServiceSpec{},
				Status: corev1.ServiceStatus{},
			},
			expected: "<none>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getExternalIPs(tt.svc)
			if result != tt.expected {
				t.Errorf("getExternalIPs() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestGetNodeStatus(t *testing.T) {
	tests := []struct {
		name     string
		node     *corev1.Node
		expected string
	}{
		{
			name: "ready",
			node: &corev1.Node{
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
					},
				},
			},
			expected: "Ready",
		},
		{
			name: "not ready",
			node: &corev1.Node{
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
					},
				},
			},
			expected: "NotReady",
		},
		{
			name: "no ready condition",
			node: &corev1.Node{
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionFalse},
					},
				},
			},
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNodeStatus(tt.node)
			if result != tt.expected {
				t.Errorf("getNodeStatus() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestGetNodeRoles(t *testing.T) {
	tests := []struct {
		name     string
		node     *corev1.Node
		expected string
	}{
		{
			name: "control-plane",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"node-role.kubernetes.io/control-plane": "",
					},
				},
			},
			expected: "control-plane",
		},
		{
			name: "multiple roles",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"node-role.kubernetes.io/control-plane": "",
						"node-role.kubernetes.io/master":        "",
					},
				},
			},
			// Order may vary
			expected: "control-plane, master",
		},
		{
			name: "no roles",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kubernetes.io/hostname": "node-1",
					},
				},
			},
			expected: "<none>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNodeRoles(tt.node)
			// For multiple roles, check contains since order may vary
			if tt.name == "multiple roles" {
				if result != "control-plane, master" && result != "master, control-plane" {
					t.Errorf("getNodeRoles() = %s, want roles containing control-plane and master", result)
				}
			} else if result != tt.expected {
				t.Errorf("getNodeRoles() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestFormatNodeSelector(t *testing.T) {
	tests := []struct {
		name     string
		selector map[string]string
		expected string
	}{
		{
			name:     "empty",
			selector: map[string]string{},
			expected: "<none>",
		},
		{
			name:     "single",
			selector: map[string]string{"disk": "ssd"},
			expected: "disk=ssd",
		},
		{
			name:     "nil",
			selector: nil,
			expected: "<none>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatNodeSelector(tt.selector)
			if result != tt.expected {
				t.Errorf("formatNodeSelector() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestGetIngressClass(t *testing.T) {
	className := "nginx"
	tests := []struct {
		name     string
		ing      *networkingv1.Ingress
		expected string
	}{
		{
			name: "spec class name",
			ing: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					IngressClassName: &className,
				},
			},
			expected: "nginx",
		},
		{
			name: "annotation class",
			ing: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "traefik",
					},
				},
			},
			expected: "traefik",
		},
		{
			name: "no class",
			ing: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			expected: "<none>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIngressClass(tt.ing)
			if result != tt.expected {
				t.Errorf("getIngressClass() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestGetIngressAddress(t *testing.T) {
	tests := []struct {
		name     string
		ing      *networkingv1.Ingress
		expected string
	}{
		{
			name: "IP address",
			ing: &networkingv1.Ingress{
				Status: networkingv1.IngressStatus{
					LoadBalancer: networkingv1.IngressLoadBalancerStatus{
						Ingress: []networkingv1.IngressLoadBalancerIngress{
							{IP: "10.0.0.100"},
						},
					},
				},
			},
			expected: "10.0.0.100",
		},
		{
			name: "hostname",
			ing: &networkingv1.Ingress{
				Status: networkingv1.IngressStatus{
					LoadBalancer: networkingv1.IngressLoadBalancerStatus{
						Ingress: []networkingv1.IngressLoadBalancerIngress{
							{Hostname: "ingress.example.com"},
						},
					},
				},
			},
			expected: "ingress.example.com",
		},
		{
			name: "pending",
			ing: &networkingv1.Ingress{
				Status: networkingv1.IngressStatus{},
			},
			expected: "<pending>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIngressAddress(tt.ing)
			if result != tt.expected {
				t.Errorf("getIngressAddress() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestGetAccessModes(t *testing.T) {
	tests := []struct {
		name     string
		modes    []corev1.PersistentVolumeAccessMode
		expected string
	}{
		{
			name:     "RWO",
			modes:    []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			expected: "RWO",
		},
		{
			name:     "multiple",
			modes:    []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce, corev1.ReadOnlyMany},
			expected: "RWO, ROX",
		},
		{
			name:     "all modes",
			modes:    []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce, corev1.ReadOnlyMany, corev1.ReadWriteMany, corev1.ReadWriteOncePod},
			expected: "RWO, ROX, RWX, RWOP",
		},
		{
			name:     "empty",
			modes:    []corev1.PersistentVolumeAccessMode{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAccessModes(tt.modes)
			if result != tt.expected {
				t.Errorf("getAccessModes() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestGetStorageClass(t *testing.T) {
	sc := "fast-ssd"
	tests := []struct {
		name     string
		sc       *string
		expected string
	}{
		{
			name:     "specified",
			sc:       &sc,
			expected: "fast-ssd",
		},
		{
			name:     "nil",
			sc:       nil,
			expected: "<default>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStorageClass(tt.sc)
			if result != tt.expected {
				t.Errorf("getStorageClass() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestGetReclaimPolicy(t *testing.T) {
	retain := corev1.PersistentVolumeReclaimRetain
	tests := []struct {
		name     string
		policy   *corev1.PersistentVolumeReclaimPolicy
		expected string
	}{
		{
			name:     "retain",
			policy:   &retain,
			expected: "Retain",
		},
		{
			name:     "nil",
			policy:   nil,
			expected: "Delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getReclaimPolicy(tt.policy)
			if result != tt.expected {
				t.Errorf("getReclaimPolicy() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestGetJobCompletions(t *testing.T) {
	two := int32(2)
	tests := []struct {
		name     string
		job      *batchv1.Job
		expected string
	}{
		{
			name: "completed",
			job: &batchv1.Job{
				Spec:   batchv1.JobSpec{Completions: &two},
				Status: batchv1.JobStatus{Succeeded: 2},
			},
			expected: "2/2",
		},
		{
			name: "partial",
			job: &batchv1.Job{
				Spec:   batchv1.JobSpec{Completions: &two},
				Status: batchv1.JobStatus{Succeeded: 1},
			},
			expected: "1/2",
		},
		{
			name: "default completions",
			job: &batchv1.Job{
				Spec:   batchv1.JobSpec{},
				Status: batchv1.JobStatus{Succeeded: 1},
			},
			expected: "1/1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getJobCompletions(tt.job)
			if result != tt.expected {
				t.Errorf("getJobCompletions() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestGetJobDuration(t *testing.T) {
	now := metav1.Now()
	past := metav1.NewTime(now.Add(-5 * time.Minute))

	tests := []struct {
		name     string
		job      *batchv1.Job
		expected string
	}{
		{
			name: "pending",
			job: &batchv1.Job{
				Status: batchv1.JobStatus{},
			},
			expected: "<pending>",
		},
		{
			name: "completed",
			job: &batchv1.Job{
				Status: batchv1.JobStatus{
					StartTime:      &past,
					CompletionTime: &now,
				},
			},
			expected: "5m", // formatDuration returns "5m" for exactly 5 minutes (no seconds)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getJobDuration(tt.job)
			if result != tt.expected {
				// For running jobs, duration changes
				if tt.name != "running" {
					t.Errorf("getJobDuration() = %s, want %s", result, tt.expected)
				}
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"seconds", 45 * time.Second, "45s"},
		{"minutes", 5*time.Minute + 30*time.Second, "5m30s"},
		{"hours", 2*time.Hour + 15*time.Minute, "2h15m"},
		{"days", 25*time.Hour + 30*time.Minute, "1d1h"},
		{"just days", 48 * time.Hour, "2d"},
		{"zero", 0, "0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %s, want %s", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestGetMinReplicas(t *testing.T) {
	five := int32(5)
	tests := []struct {
		name     string
		min      *int32
		expected int32
	}{
		{"specified", &five, 5},
		{"nil", nil, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMinReplicas(tt.min)
			if result != tt.expected {
				t.Errorf("getMinReplicas() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestFormatLabelSelector(t *testing.T) {
	tests := []struct {
		name     string
		selector *metav1.LabelSelector
		expected string
		contains []string // for cases where order may vary
	}{
		{
			name:     "nil",
			selector: nil,
			expected: "",
		},
		{
			name:     "empty",
			selector: &metav1.LabelSelector{},
			expected: "",
		},
		{
			name: "match labels",
			selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "nginx"},
			},
			expected: "app=nginx",
		},
		{
			name: "multiple match labels",
			selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "nginx", "version": "v1"},
			},
			contains: []string{"app=nginx", "version=v1"},
		},
		{
			name: "match expression In",
			selector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "env", Operator: metav1.LabelSelectorOpIn, Values: []string{"prod", "staging"}},
				},
			},
			expected: "env in (prod,staging)",
		},
		{
			name: "match expression NotIn",
			selector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "tier", Operator: metav1.LabelSelectorOpNotIn, Values: []string{"frontend"}},
				},
			},
			expected: "tier notin (frontend)",
		},
		{
			name: "match expression Exists",
			selector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "app", Operator: metav1.LabelSelectorOpExists},
				},
			},
			expected: "app",
		},
		{
			name: "match expression DoesNotExist",
			selector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "deprecated", Operator: metav1.LabelSelectorOpDoesNotExist},
				},
			},
			expected: "!deprecated",
		},
		{
			name: "combined labels and expressions",
			selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "web"},
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "env", Operator: metav1.LabelSelectorOpIn, Values: []string{"prod"}},
				},
			},
			contains: []string{"app=web", "env in (prod)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatLabelSelector(tt.selector)
			if len(tt.contains) > 0 {
				// For cases where order may vary, check all expected parts are present
				for _, part := range tt.contains {
					if !strings.Contains(result, part) {
						t.Errorf("formatLabelSelector() = %s, want contains %s", result, part)
					}
				}
			} else if result != tt.expected {
				t.Errorf("formatLabelSelector() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestClassifyCommand(t *testing.T) {
	tests := []struct {
		command  string
		expected string
	}{
		// Read-only commands
		{"kubectl get pods", "read-only"},
		{"kubectl describe deployment nginx", "read-only"},
		{"kubectl logs nginx-pod", "read-only"},
		{"kubectl top nodes", "read-only"},
		{"kubectl api-resources", "read-only"},
		{"kubectl auth can-i get pods", "read-only"},
		{"kubectl explain deployment", "read-only"},
		{"kubectl diff -f manifest.yaml", "read-only"},

		// Dangerous commands (with dangerous flags or verbs)
		// Note: The unified classifier uses AST parsing and only marks commands
		// as "dangerous" when they have dangerous FLAGS (--all, --force, etc)
		// or dangerous VERBS (drain, cordon, taint)
		{"kubectl delete --all pods", "dangerous"},
		{"kubectl drain node-1", "dangerous"},
		{"kubectl cordon node-1", "dangerous"},
		{"kubectl taint node node-1 key=value:NoSchedule", "dangerous"},
		{"kubectl delete pod nginx --force", "dangerous"},
		{"kubectl replace --force -f deployment.yaml", "dangerous"},

		// Write commands (including delete without dangerous flags)
		// The unified classifier correctly identifies these as "write"
		// because they modify state but don't have dangerous patterns
		{"kubectl delete pod nginx", "write"},
		{"kubectl delete namespace production", "write"},
		{"kubectl delete pod nginx --grace-period=0", "write"}, // grace-period=0 alone doesn't trigger dangerous (needs --force too)
		{"kubectl rollout undo deployment nginx", "write"},
		{"kubectl apply -f deployment.yaml", "write"},
		{"kubectl create deployment nginx --image=nginx", "write"},
		{"kubectl patch deployment nginx -p '{\"spec\":{\"replicas\":3}}'", "write"},
		{"kubectl edit deployment nginx", "write"},
		{"kubectl scale deployment nginx --replicas=5", "write"},
		{"kubectl rollout restart deployment nginx", "write"},
		{"kubectl set image deployment/nginx nginx=nginx:1.19", "write"},
		{"kubectl label pod nginx env=prod", "write"},
		{"kubectl annotate deployment nginx description='My app'", "write"},
		{"kubectl expose deployment nginx --port=80", "write"},
		{"kubectl run nginx --image=nginx", "write"},
		{"kubectl cp /tmp/file nginx-pod:/tmp/", "write"},

		// Interactive commands (exec is interactive, not just write)
		{"kubectl exec nginx-pod -- ls", "interactive"},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := classifyCommand(tt.command)
			if result != tt.expected {
				t.Errorf("classifyCommand(%q) = %s, want %s", tt.command, result, tt.expected)
			}
		})
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-Forwarded-For",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1, 192.168.1.1"},
			remoteAddr: "127.0.0.1:8080",
			expected:   "10.0.0.1",
		},
		{
			name:       "X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "10.0.0.2"},
			remoteAddr: "127.0.0.1:8080",
			expected:   "10.0.0.2",
		},
		{
			name:       "RemoteAddr",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.100:12345",
			expected:   "192.168.1.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			req.RemoteAddr = tt.remoteAddr

			result := getClientIP(req)
			if result != tt.expected {
				t.Errorf("getClientIP() = %s, want %s", result, tt.expected)
			}
		})
	}
}
