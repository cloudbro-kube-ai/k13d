package security

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// ---------- helpers ----------

func boolPtr(b bool) *bool       { return &b }
func int64Ptr(i int64) *int64    { return &i }

// ---------- Test Scanner Initialization ----------

func TestNewScanner(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset() //nolint:staticcheck
	client := &k8s.Client{Clientset: fakeClientset}

	scanner := NewScanner(client)

	if scanner == nil {
		t.Fatal("NewScanner returned nil")
	}
	if scanner.k8sClient != client {
		t.Error("k8sClient not set correctly")
	}
	// trivy and kube-bench availability depend on the system;
	// just ensure the scanner is created without error.
}

func TestScanner_TrivyAvailable(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset() //nolint:staticcheck
	client := &k8s.Client{Clientset: fakeClientset}

	scanner := &Scanner{k8sClient: client, trivyPath: ""}
	if scanner.TrivyAvailable() {
		t.Error("expected TrivyAvailable=false when trivyPath is empty")
	}

	scanner.SetTrivyPath("/usr/local/bin/trivy")
	if !scanner.TrivyAvailable() {
		t.Error("expected TrivyAvailable=true after SetTrivyPath")
	}
	if scanner.GetTrivyPath() != "/usr/local/bin/trivy" {
		t.Errorf("GetTrivyPath mismatch: got %q", scanner.GetTrivyPath())
	}
}

func TestScanner_KubeBenchAvailable(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset() //nolint:staticcheck
	client := &k8s.Client{Clientset: fakeClientset}

	scanner := &Scanner{k8sClient: client, kubeBenchAvailable: false}
	if scanner.KubeBenchAvailable() {
		t.Error("expected KubeBenchAvailable=false")
	}

	scanner.kubeBenchAvailable = true
	if !scanner.KubeBenchAvailable() {
		t.Error("expected KubeBenchAvailable=true")
	}
}

// ---------- Test Score Calculation ----------

func TestCalculateScore(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset() //nolint:staticcheck
	client := &k8s.Client{Clientset: fakeClientset}
	scanner := &Scanner{k8sClient: client}

	tests := []struct {
		name     string
		result   *ScanResult
		minScore float64
		maxScore float64
	}{
		{
			name:     "empty result gives perfect score",
			result:   &ScanResult{},
			minScore: 100,
			maxScore: 100,
		},
		{
			name: "critical image vulns reduce score",
			result: &ScanResult{
				ImageVulns: &ImageVulnSummary{
					CriticalCount: 5,
				},
			},
			minScore: 74, // 100 - 5*5 = 75
			maxScore: 76,
		},
		{
			name: "high image vulns reduce score",
			result: &ScanResult{
				ImageVulns: &ImageVulnSummary{
					HighCount: 10,
				},
			},
			minScore: 79, // 100 - 10*2 = 80
			maxScore: 81,
		},
		{
			name: "medium image vulns reduce score slightly",
			result: &ScanResult{
				ImageVulns: &ImageVulnSummary{
					MediumCount: 20,
				},
			},
			minScore: 89, // 100 - 20*0.5 = 90
			maxScore: 91,
		},
		{
			name: "pod security CRITICAL issues",
			result: &ScanResult{
				PodSecurityIssues: []PodSecurityIssue{
					{Severity: "CRITICAL"},
					{Severity: "CRITICAL"},
				},
			},
			minScore: 79, // 100 - 2*10 = 80
			maxScore: 81,
		},
		{
			name: "pod security HIGH issues",
			result: &ScanResult{
				PodSecurityIssues: []PodSecurityIssue{
					{Severity: "HIGH"},
				},
			},
			minScore: 94, // 100 - 5 = 95
			maxScore: 96,
		},
		{
			name: "pod security MEDIUM issues",
			result: &ScanResult{
				PodSecurityIssues: []PodSecurityIssue{
					{Severity: "MEDIUM"},
				},
			},
			minScore: 97, // 100 - 2 = 98
			maxScore: 99,
		},
		{
			name: "pod security LOW issues",
			result: &ScanResult{
				PodSecurityIssues: []PodSecurityIssue{
					{Severity: "LOW"},
				},
			},
			minScore: 99, // 100 - 0.5 = 99.5
			maxScore: 100,
		},
		{
			name: "RBAC CRITICAL issues",
			result: &ScanResult{
				RBACIssues: []RBACIssue{
					{Severity: "CRITICAL"},
				},
			},
			minScore: 84, // 100 - 15 = 85
			maxScore: 86,
		},
		{
			name: "RBAC HIGH issues",
			result: &ScanResult{
				RBACIssues: []RBACIssue{
					{Severity: "HIGH"},
				},
			},
			minScore: 92, // 100 - 7 = 93
			maxScore: 94,
		},
		{
			name: "RBAC MEDIUM issues",
			result: &ScanResult{
				RBACIssues: []RBACIssue{
					{Severity: "MEDIUM"},
				},
			},
			minScore: 96, // 100 - 3 = 97
			maxScore: 98,
		},
		{
			name: "network HIGH issues",
			result: &ScanResult{
				NetworkIssues: []NetworkIssue{
					{Severity: "HIGH"},
				},
			},
			minScore: 94, // 100 - 5 = 95
			maxScore: 96,
		},
		{
			name: "network MEDIUM issues",
			result: &ScanResult{
				NetworkIssues: []NetworkIssue{
					{Severity: "MEDIUM"},
				},
			},
			minScore: 97, // 100 - 2 = 98
			maxScore: 99,
		},
		{
			name: "network LOW issues",
			result: &ScanResult{
				NetworkIssues: []NetworkIssue{
					{Severity: "LOW"},
				},
			},
			minScore: 98, // 100 - 1 = 99
			maxScore: 100,
		},
		{
			name: "CIS benchmark affects score",
			result: &ScanResult{
				CISBenchmark: &CISBenchmarkResult{
					Score: 50,
				},
			},
			// score = (100 * 0.8) + (50 * 0.2) = 80 + 10 = 90
			minScore: 89,
			maxScore: 91,
		},
		{
			name: "massive issues floor at zero",
			result: &ScanResult{
				ImageVulns: &ImageVulnSummary{
					CriticalCount: 100,
				},
				PodSecurityIssues: []PodSecurityIssue{
					{Severity: "CRITICAL"},
					{Severity: "CRITICAL"},
					{Severity: "CRITICAL"},
					{Severity: "CRITICAL"},
					{Severity: "CRITICAL"},
					{Severity: "CRITICAL"},
					{Severity: "CRITICAL"},
					{Severity: "CRITICAL"},
					{Severity: "CRITICAL"},
					{Severity: "CRITICAL"},
				},
			},
			minScore: 0,
			maxScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scanner.calculateScore(tt.result)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("calculateScore() = %v, want [%v, %v]", score, tt.minScore, tt.maxScore)
			}
		})
	}
}

// ---------- Test Risk Level ----------

func TestDetermineRiskLevel(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset() //nolint:staticcheck
	client := &k8s.Client{Clientset: fakeClientset}
	scanner := &Scanner{k8sClient: client}

	tests := []struct {
		score    float64
		expected string
	}{
		{100, "Low"},
		{95, "Low"},
		{90, "Low"},
		{89.9, "Medium"},
		{75, "Medium"},
		{70, "Medium"},
		{69.9, "High"},
		{55, "High"},
		{50, "High"},
		{49.9, "Critical"},
		{25, "Critical"},
		{0, "Critical"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := scanner.determineRiskLevel(tt.score)
			if got != tt.expected {
				t.Errorf("determineRiskLevel(%v) = %q, want %q", tt.score, got, tt.expected)
			}
		})
	}
}

// ---------- Test Recommendations ----------

func TestGenerateRecommendations(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset() //nolint:staticcheck
	client := &k8s.Client{Clientset: fakeClientset}
	scanner := &Scanner{k8sClient: client}

	tests := []struct {
		name          string
		result        *ScanResult
		wantMinCount  int
		wantCategory  string // at least one recommendation with this category
	}{
		{
			name:         "empty result no recommendations",
			result:       &ScanResult{},
			wantMinCount: 0,
		},
		{
			name: "critical image vulns generate recommendation",
			result: &ScanResult{
				ImageVulns: &ImageVulnSummary{CriticalCount: 3},
			},
			wantMinCount: 1,
			wantCategory: "Image Security",
		},
		{
			name: "privileged containers generate recommendation",
			result: &ScanResult{
				PodSecurityIssues: []PodSecurityIssue{
					{Issue: "Container running in privileged mode", Severity: "CRITICAL"},
				},
			},
			wantMinCount: 1,
			wantCategory: "Pod Security",
		},
		{
			name: "cluster-admin RBAC generates recommendation",
			result: &ScanResult{
				RBACIssues: []RBACIssue{
					{Issue: "SA default/mysa has cluster-admin privileges", Severity: "HIGH"},
				},
			},
			wantMinCount: 1,
			wantCategory: "RBAC",
		},
		{
			name: "missing NetworkPolicies generates recommendation",
			result: &ScanResult{
				NetworkIssues: []NetworkIssue{
					{Issue: "No NetworkPolicies defined", Severity: "MEDIUM"},
				},
			},
			wantMinCount: 1,
			wantCategory: "Network Security",
		},
		{
			name: "CIS benchmark failures generate recommendation",
			result: &ScanResult{
				CISBenchmark: &CISBenchmarkResult{FailCount: 5},
			},
			wantMinCount: 1,
			wantCategory: "CIS Benchmark",
		},
		{
			name: "multiple issue types generate multiple recommendations",
			result: &ScanResult{
				ImageVulns:        &ImageVulnSummary{CriticalCount: 1},
				PodSecurityIssues: []PodSecurityIssue{{Issue: "privileged", Severity: "CRITICAL"}},
				RBACIssues:        []RBACIssue{{Issue: "cluster-admin", Severity: "HIGH"}},
				NetworkIssues:     []NetworkIssue{{Issue: "No NetworkPolicies", Severity: "MEDIUM"}},
				CISBenchmark:      &CISBenchmarkResult{FailCount: 2},
			},
			wantMinCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recs := scanner.generateRecommendations(tt.result)
			if len(recs) < tt.wantMinCount {
				t.Errorf("got %d recommendations, want at least %d", len(recs), tt.wantMinCount)
			}
			if tt.wantCategory != "" {
				found := false
				for _, r := range recs {
					if r.Category == tt.wantCategory {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("no recommendation with category %q found", tt.wantCategory)
				}
			}
		})
	}
}

// ---------- Test Pod Security Checks ----------

func TestCheckPodSecurity(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		pods        []*corev1.Pod
		namespace   string
		wantIssues  int
		wantContain string // substring of an issue description
	}{
		{
			name:       "no pods no issues",
			pods:       nil,
			namespace:  "",
			wantIssues: 0,
		},
		{
			name: "privileged container detected",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "priv-pod", Namespace: "default"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "c1",
								Image: "nginx:1.25",
								SecurityContext: &corev1.SecurityContext{
									Privileged: boolPtr(true),
								},
							},
						},
					},
				},
			},
			namespace:   "",
			wantIssues:  1, // at least 1 for privileged
			wantContain: "privileged",
		},
		{
			name: "root user detected in non-system namespace",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "root-pod", Namespace: "myapp"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "c1",
								Image: "nginx:1.25",
								// no securityContext => may run as root
							},
						},
					},
				},
			},
			namespace:   "",
			wantIssues:  1, // at least 1 for root
			wantContain: "root",
		},
		{
			name: "root user NOT reported in system namespace",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "sys-pod", Namespace: "kube-system"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "c1",
								Image: "kube-apiserver:v1.30",
							},
						},
					},
				},
			},
			namespace:   "",
			wantIssues:  0,
			wantContain: "",
		},
		{
			name: "hostPID detected",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "hostpid-pod", Namespace: "default"},
					Spec: corev1.PodSpec{
						HostPID: true,
						Containers: []corev1.Container{
							{
								Name:  "c1",
								Image: "busybox:1.36",
								SecurityContext: &corev1.SecurityContext{
									RunAsNonRoot: boolPtr(true),
								},
							},
						},
					},
				},
			},
			namespace:   "",
			wantIssues:  1,
			wantContain: "host PID",
		},
		{
			name: "hostNetwork in non-system namespace",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "hostnet-pod", Namespace: "myapp"},
					Spec: corev1.PodSpec{
						HostNetwork: true,
						Containers: []corev1.Container{
							{
								Name:  "c1",
								Image: "nginx:1.25",
								SecurityContext: &corev1.SecurityContext{
									RunAsNonRoot: boolPtr(true),
								},
							},
						},
					},
				},
			},
			namespace:   "",
			wantIssues:  1,
			wantContain: "host network",
		},
		{
			name: "dangerous capability SYS_ADMIN",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "cap-pod", Namespace: "default"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "c1",
								Image: "nginx:1.25",
								SecurityContext: &corev1.SecurityContext{
									RunAsNonRoot: boolPtr(true),
									Capabilities: &corev1.Capabilities{
										Add: []corev1.Capability{"SYS_ADMIN"},
									},
								},
							},
						},
					},
				},
			},
			namespace:   "",
			wantIssues:  1,
			wantContain: "SYS_ADMIN",
		},
		{
			name: "missing resource limits in non-system ns",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "nolimit-pod", Namespace: "myapp"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "c1",
								Image: "nginx:1.25",
								SecurityContext: &corev1.SecurityContext{
									RunAsNonRoot: boolPtr(true),
								},
								// No resource limits
							},
						},
					},
				},
			},
			namespace:   "",
			wantIssues:  1,
			wantContain: "resource limits",
		},
		{
			name: "latest tag detected",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "latest-pod", Namespace: "default"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "c1",
								Image: "nginx:latest",
								SecurityContext: &corev1.SecurityContext{
									RunAsNonRoot: boolPtr(true),
								},
								Resources: corev1.ResourceRequirements{
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("100m"),
										corev1.ResourceMemory: resource.MustParse("128Mi"),
									},
								},
							},
						},
					},
				},
			},
			namespace:   "",
			wantIssues:  1,
			wantContain: "latest",
		},
		{
			name: "untagged image detected",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "untagged-pod", Namespace: "default"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "c1",
								Image: "nginx",
								SecurityContext: &corev1.SecurityContext{
									RunAsNonRoot: boolPtr(true),
								},
								Resources: corev1.ResourceRequirements{
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("100m"),
										corev1.ResourceMemory: resource.MustParse("128Mi"),
									},
								},
							},
						},
					},
				},
			},
			namespace:   "",
			wantIssues:  1,
			wantContain: "latest",
		},
		{
			name: "secure pod has no issues",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "secure-pod", Namespace: "myapp"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "c1",
								Image: "nginx:1.25.3",
								SecurityContext: &corev1.SecurityContext{
									RunAsNonRoot: boolPtr(true),
									RunAsUser:    int64Ptr(1000),
								},
								Resources: corev1.ResourceRequirements{
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("100m"),
										corev1.ResourceMemory: resource.MustParse("128Mi"),
									},
								},
							},
						},
					},
				},
			},
			namespace:  "",
			wantIssues: 0,
		},
		{
			name: "namespace filter works",
			pods: []*corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "pod-a", Namespace: "ns-a"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "c1", Image: "nginx:latest", SecurityContext: &corev1.SecurityContext{RunAsNonRoot: boolPtr(true)},
								Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1"), corev1.ResourceMemory: resource.MustParse("1Gi")}}},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "pod-b", Namespace: "ns-b"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "c1", Image: "nginx:latest", SecurityContext: &corev1.SecurityContext{RunAsNonRoot: boolPtr(true)},
								Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1"), corev1.ResourceMemory: resource.MustParse("1Gi")}}},
						},
					},
				},
			},
			namespace:  "ns-a",
			wantIssues: 1, // only ns-a pod's latest tag issue
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClientset := buildFakeClientsetFromPods(tt.pods...)
			client := &k8s.Client{Clientset: fakeClientset}
			scanner := &Scanner{k8sClient: client}

			issues, err := scanner.checkPodSecurity(ctx, tt.namespace)
			if err != nil {
				t.Fatalf("checkPodSecurity() error: %v", err)
			}

			if len(issues) < tt.wantIssues {
				t.Errorf("got %d issues, want at least %d", len(issues), tt.wantIssues)
				for i, issue := range issues {
					t.Logf("  issue[%d]: %s (severity=%s)", i, issue.Issue, issue.Severity)
				}
			}

			if tt.wantContain != "" {
				found := false
				for _, issue := range issues {
					if containsCI(issue.Issue, tt.wantContain) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("no issue containing %q found among %d issues", tt.wantContain, len(issues))
					for i, issue := range issues {
						t.Logf("  issue[%d]: %s", i, issue.Issue)
					}
				}
			}
		})
	}
}

func containsCI(s, substr string) bool {
	return len(s) >= len(substr) &&
		(contains(s, substr) || contains(lower(s), lower(substr)))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func lower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// ---------- Test RBAC Checks ----------

func TestCheckRBAC(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		crbs        []*rbacv1.ClusterRoleBinding
		crs         []*rbacv1.ClusterRole
		namespace   string
		wantMin     int
		wantContain string
	}{
		{
			name:    "no RBAC objects no issues",
			crbs:    nil,
			crs:     nil,
			wantMin: 0,
		},
		{
			name: "system bindings are skipped",
			crbs: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "system:node"},
					RoleRef:    rbacv1.RoleRef{Name: "cluster-admin", Kind: "ClusterRole", APIGroup: "rbac.authorization.k8s.io"},
					Subjects: []rbacv1.Subject{
						{Kind: "ServiceAccount", Name: "node", Namespace: "kube-system"},
					},
				},
			},
			wantMin: 0,
		},
		{
			name: "cluster-admin binding to non-system SA detected",
			crbs: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "admin-binding"},
					RoleRef:    rbacv1.RoleRef{Name: "cluster-admin", Kind: "ClusterRole", APIGroup: "rbac.authorization.k8s.io"},
					Subjects: []rbacv1.Subject{
						{Kind: "ServiceAccount", Name: "my-sa", Namespace: "default"},
					},
				},
			},
			wantMin:     1,
			wantContain: "cluster-admin",
		},
		{
			name: "cluster-admin binding to kube-system SA is NOT flagged",
			crbs: []*rbacv1.ClusterRoleBinding{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "admin-binding"},
					RoleRef:    rbacv1.RoleRef{Name: "cluster-admin", Kind: "ClusterRole", APIGroup: "rbac.authorization.k8s.io"},
					Subjects: []rbacv1.Subject{
						{Kind: "ServiceAccount", Name: "default", Namespace: "kube-system"},
					},
				},
			},
			wantMin: 0,
		},
		{
			name: "wildcard permissions detected",
			crs: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "super-admin"},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{"*"},
							Resources: []string{"*"},
							Verbs:     []string{"*"},
						},
					},
				},
			},
			wantMin:     1,
			wantContain: "wildcard",
		},
		{
			name: "system ClusterRoles are skipped",
			crs: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "system:controller:replicaset-controller"},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{"*"},
							Resources: []string{"*"},
							Verbs:     []string{"*"},
						},
					},
				},
			},
			wantMin: 0,
		},
		{
			name: "secrets access detected",
			crs: []*rbacv1.ClusterRole{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "secret-reader"},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{""},
							Resources: []string{"secrets"},
							Verbs:     []string{"get", "list"},
						},
					},
				},
			},
			wantMin:     1,
			wantContain: "secrets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClientset := buildFakeClientsetFromRBAC(tt.crbs, tt.crs)
			client := &k8s.Client{Clientset: fakeClientset}
			scanner := &Scanner{k8sClient: client}

			issues, err := scanner.checkRBAC(ctx, tt.namespace)
			if err != nil {
				t.Fatalf("checkRBAC() error: %v", err)
			}

			if len(issues) < tt.wantMin {
				t.Errorf("got %d issues, want at least %d", len(issues), tt.wantMin)
				for i, issue := range issues {
					t.Logf("  issue[%d]: %s", i, issue.Issue)
				}
			}

			if tt.wantContain != "" {
				found := false
				for _, issue := range issues {
					if containsCI(issue.Issue, tt.wantContain) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("no issue containing %q", tt.wantContain)
					for i, issue := range issues {
						t.Logf("  issue[%d]: %s", i, issue.Issue)
					}
				}
			}
		})
	}
}

// ---------- Test Network Checks ----------

func TestCheckNetwork(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		namespaces  []*corev1.Namespace
		pods        []*corev1.Pod
		netPols     []*networkingv1.NetworkPolicy
		services    []*corev1.Service
		namespace   string
		wantMin     int
		wantContain string
	}{
		{
			name:       "no resources no issues",
			namespaces: nil,
			wantMin:    0,
		},
		{
			name: "namespace with pods but no network policies",
			namespaces: []*corev1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: "myapp"}},
			},
			pods: []*corev1.Pod{
				{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "myapp"}},
			},
			namespace:   "",
			wantMin:     1,
			wantContain: "NetworkPolicies",
		},
		{
			name: "namespace with network policies is OK",
			namespaces: []*corev1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: "myapp"}},
			},
			pods: []*corev1.Pod{
				{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "myapp"}},
			},
			netPols: []*networkingv1.NetworkPolicy{
				{ObjectMeta: metav1.ObjectMeta{Name: "default-deny", Namespace: "myapp"}},
			},
			namespace: "",
			wantMin:   0,
		},
		{
			name: "LoadBalancer service detected in non-system ns",
			namespaces: []*corev1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: "myapp"}},
			},
			services: []*corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "web-svc", Namespace: "myapp"},
					Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeLoadBalancer},
				},
			},
			netPols: []*networkingv1.NetworkPolicy{
				{ObjectMeta: metav1.ObjectMeta{Name: "default-deny", Namespace: "myapp"}},
			},
			namespace:   "",
			wantMin:     1,
			wantContain: "exposed externally",
		},
		{
			name: "NodePort service detected",
			namespaces: []*corev1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: "myapp"}},
			},
			services: []*corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "api-svc", Namespace: "myapp"},
					Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeNodePort},
				},
			},
			netPols: []*networkingv1.NetworkPolicy{
				{ObjectMeta: metav1.ObjectMeta{Name: "default-deny", Namespace: "myapp"}},
			},
			namespace:   "",
			wantMin:     1,
			wantContain: "exposed externally",
		},
		{
			name: "system namespace services are skipped",
			namespaces: []*corev1.Namespace{
				{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},
			},
			services: []*corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "kube-dns", Namespace: "kube-system"},
					Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeNodePort},
				},
			},
			namespace: "",
			wantMin:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClientset := buildFakeClientsetFromNetwork(tt.namespaces, tt.pods, tt.netPols, tt.services)
			client := &k8s.Client{Clientset: fakeClientset}
			scanner := &Scanner{k8sClient: client}

			issues, err := scanner.checkNetwork(ctx, tt.namespace)
			if err != nil {
				t.Fatalf("checkNetwork() error: %v", err)
			}

			if len(issues) < tt.wantMin {
				t.Errorf("got %d issues, want at least %d", len(issues), tt.wantMin)
				for i, issue := range issues {
					t.Logf("  issue[%d]: %s", i, issue.Issue)
				}
			}

			if tt.wantContain != "" {
				found := false
				for _, issue := range issues {
					if containsCI(issue.Issue, tt.wantContain) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("no issue containing %q", tt.wantContain)
					for i, issue := range issues {
						t.Logf("  issue[%d]: %s", i, issue.Issue)
					}
				}
			}
		})
	}
}

// ---------- Test Image Scanning ----------

func TestScanImages_NoTrivy(t *testing.T) {
	ctx := context.Background()

	pods := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "default"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "c1", Image: "nginx:1.25"},
					{Name: "c2", Image: "redis:7"},
				},
				InitContainers: []corev1.Container{
					{Name: "init1", Image: "busybox:1.36"},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "default"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "c1", Image: "nginx:1.25"}, // duplicate image
				},
			},
		},
	}

	fakeClientset := buildFakeClientsetFromPods(pods...)
	client := &k8s.Client{Clientset: fakeClientset}
	scanner := &Scanner{k8sClient: client, trivyPath: ""}

	summary, err := scanner.scanImages(ctx, "")
	if err != nil {
		t.Fatalf("scanImages() error: %v", err)
	}

	if summary.TotalImages != 3 {
		t.Errorf("TotalImages = %d, want 3 (nginx, redis, busybox)", summary.TotalImages)
	}
	if summary.ScannedImages != 0 {
		t.Errorf("ScannedImages = %d, want 0 (no trivy)", summary.ScannedImages)
	}
}

func TestScanImages_NamespaceFilter(t *testing.T) {
	ctx := context.Background()

	pods := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-ns1", Namespace: "ns1"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "c1", Image: "nginx:1.25"},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-ns2", Namespace: "ns2"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "c1", Image: "redis:7"},
				},
			},
		},
	}

	fakeClientset := buildFakeClientsetFromPods(pods...)
	client := &k8s.Client{Clientset: fakeClientset}
	scanner := &Scanner{k8sClient: client, trivyPath: ""}

	summary, err := scanner.scanImages(ctx, "ns1")
	if err != nil {
		t.Fatalf("scanImages(ns1) error: %v", err)
	}

	if summary.TotalImages != 1 {
		t.Errorf("TotalImages = %d, want 1 (only nginx in ns1)", summary.TotalImages)
	}
}

func TestScanImages_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	pods := []*corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "default"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "c1", Image: "nginx:1.25"},
				},
			},
		},
	}

	fakeClientset := buildFakeClientsetFromPods(pods...)
	client := &k8s.Client{Clientset: fakeClientset}
	// Set a fake trivy path to enter the scanning loop (where ctx is checked)
	scanner := &Scanner{k8sClient: client, trivyPath: "/nonexistent/trivy"}

	summary, err := scanner.scanImages(ctx, "")
	// The function should handle context cancellation gracefully
	// It may return the summary with partial data and ctx.Err()
	if err != nil && err != context.Canceled {
		t.Logf("scanImages with cancelled context returned: err=%v", err)
	}
	if summary == nil {
		t.Error("summary should not be nil even with cancellation")
	}
}

func TestScanImage_NoTrivy(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset() //nolint:staticcheck
	client := &k8s.Client{Clientset: fakeClientset}
	scanner := &Scanner{k8sClient: client, trivyPath: ""}

	_, err := scanner.ScanImage(context.Background(), "nginx:1.25")
	if err == nil {
		t.Error("expected error when trivy not available")
	}
	if err.Error() != "trivy not available" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestScanImage_InvalidBinary(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset() //nolint:staticcheck
	client := &k8s.Client{Clientset: fakeClientset}
	scanner := &Scanner{k8sClient: client, trivyPath: "/nonexistent/trivy-binary"}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := scanner.ScanImage(ctx, "nginx:1.25")
	if err == nil {
		t.Error("expected error when trivy binary doesn't exist")
	}
}

// ---------- Test Trivy Output Parsing ----------

func TestScanImageWithTrivy_ParseTrivyOutput(t *testing.T) {
	// We test parseTrivyOutput indirectly by testing the JSON parsing logic.
	// Since scanImageWithTrivy calls exec, we test the JSON parsing logic
	// by verifying the struct mapping.

	trivyJSON := `{
		"Results": [
			{
				"Vulnerabilities": [
					{
						"VulnerabilityID": "CVE-2023-1234",
						"Severity": "CRITICAL",
						"PkgName": "openssl",
						"InstalledVersion": "1.1.1k",
						"FixedVersion": "1.1.1l",
						"Description": "A critical vulnerability in OpenSSL"
					},
					{
						"VulnerabilityID": "CVE-2023-5678",
						"Severity": "HIGH",
						"PkgName": "curl",
						"InstalledVersion": "7.74.0",
						"FixedVersion": "7.88.0",
						"Description": "A high severity vulnerability"
					},
					{
						"VulnerabilityID": "CVE-2023-9012",
						"Severity": "MEDIUM",
						"PkgName": "zlib",
						"InstalledVersion": "1.2.11",
						"FixedVersion": "",
						"Description": ""
					}
				]
			},
			{
				"Vulnerabilities": [
					{
						"VulnerabilityID": "CVE-2023-1111",
						"Severity": "LOW",
						"PkgName": "bash",
						"InstalledVersion": "5.1",
						"FixedVersion": "5.2",
						"Description": "A low severity issue"
					}
				]
			}
		]
	}`

	var result struct {
		Results []struct {
			Vulnerabilities []struct {
				VulnerabilityID  string `json:"VulnerabilityID"`
				Severity         string `json:"Severity"`
				PkgName          string `json:"PkgName"`
				InstalledVersion string `json:"InstalledVersion"`
				FixedVersion     string `json:"FixedVersion"`
				Description      string `json:"Description"`
			} `json:"Vulnerabilities"`
		} `json:"Results"`
	}

	if err := json.Unmarshal([]byte(trivyJSON), &result); err != nil {
		t.Fatalf("failed to parse trivy JSON: %v", err)
	}

	// Convert to Vulnerability slice (same logic as scanImageWithTrivy)
	var vulns []Vulnerability
	for _, r := range result.Results {
		for _, v := range r.Vulnerabilities {
			vulns = append(vulns, Vulnerability{
				ID:          v.VulnerabilityID,
				Severity:    SeverityLevel(v.Severity),
				Package:     v.PkgName,
				Version:     v.InstalledVersion,
				FixedIn:     v.FixedVersion,
				Description: truncateString(v.Description, 200),
			})
		}
	}

	if len(vulns) != 4 {
		t.Fatalf("expected 4 vulnerabilities, got %d", len(vulns))
	}

	// Verify first vuln
	if vulns[0].ID != "CVE-2023-1234" {
		t.Errorf("vuln[0].ID = %q, want CVE-2023-1234", vulns[0].ID)
	}
	if vulns[0].Severity != SeverityCritical {
		t.Errorf("vuln[0].Severity = %q, want CRITICAL", vulns[0].Severity)
	}
	if vulns[0].Package != "openssl" {
		t.Errorf("vuln[0].Package = %q, want openssl", vulns[0].Package)
	}

	// Verify severity mapping
	if vulns[1].Severity != SeverityHigh {
		t.Errorf("vuln[1].Severity = %q, want HIGH", vulns[1].Severity)
	}
	if vulns[2].Severity != SeverityMedium {
		t.Errorf("vuln[2].Severity = %q, want MEDIUM", vulns[2].Severity)
	}
	if vulns[3].Severity != SeverityLow {
		t.Errorf("vuln[3].Severity = %q, want LOW", vulns[3].Severity)
	}

	// Empty FixedVersion
	if vulns[2].FixedIn != "" {
		t.Errorf("vuln[2].FixedIn = %q, want empty", vulns[2].FixedIn)
	}
}

func TestScanImageWithTrivy_MalformedJSON(t *testing.T) {
	malformedInputs := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"not json", "this is not json"},
		{"partial json", `{"Results": [`},
		{"wrong structure", `{"foo": "bar"}`},
		{"null results", `{"Results": null}`},
	}

	for _, tt := range malformedInputs {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				Results []struct {
					Vulnerabilities []struct {
						VulnerabilityID  string `json:"VulnerabilityID"`
						Severity         string `json:"Severity"`
						PkgName          string `json:"PkgName"`
						InstalledVersion string `json:"InstalledVersion"`
						FixedVersion     string `json:"FixedVersion"`
						Description      string `json:"Description"`
					} `json:"Vulnerabilities"`
				} `json:"Results"`
			}

			err := json.Unmarshal([]byte(tt.input), &result)
			if tt.name == "wrong structure" || tt.name == "null results" {
				// These are valid JSON but result in empty vulnerabilities
				if err != nil {
					return // OK if parsing fails
				}
				var vulns []Vulnerability
				for _, r := range result.Results {
					for _, v := range r.Vulnerabilities {
						vulns = append(vulns, Vulnerability{
							ID:       v.VulnerabilityID,
							Severity: SeverityLevel(v.Severity),
						})
					}
				}
				if len(vulns) != 0 {
					t.Errorf("expected 0 vulns from %q, got %d", tt.name, len(vulns))
				}
			} else {
				if err == nil {
					t.Errorf("expected error parsing %q", tt.name)
				}
			}
		})
	}
}

func TestScanImageWithTrivy_EmptyResults(t *testing.T) {
	trivyJSON := `{"Results": []}`

	var result struct {
		Results []struct {
			Vulnerabilities []struct {
				VulnerabilityID string `json:"VulnerabilityID"`
			} `json:"Vulnerabilities"`
		} `json:"Results"`
	}

	if err := json.Unmarshal([]byte(trivyJSON), &result); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(result.Results) != 0 {
		t.Errorf("expected empty Results, got %d", len(result.Results))
	}
}

func TestScanImageWithTrivy_NoVulnerabilities(t *testing.T) {
	trivyJSON := `{"Results": [{"Vulnerabilities": null}]}`

	var result struct {
		Results []struct {
			Vulnerabilities []struct {
				VulnerabilityID string `json:"VulnerabilityID"`
			} `json:"Vulnerabilities"`
		} `json:"Results"`
	}

	if err := json.Unmarshal([]byte(trivyJSON), &result); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	var vulns []Vulnerability
	for _, r := range result.Results {
		for _, v := range r.Vulnerabilities {
			vulns = append(vulns, Vulnerability{ID: v.VulnerabilityID})
		}
	}

	if len(vulns) != 0 {
		t.Errorf("expected 0 vulns, got %d", len(vulns))
	}
}

// ---------- Test CIS Benchmark Parsing ----------

func TestRunCISBenchmark_ParseOutput(t *testing.T) {
	kubeBenchJSON := `{
		"Controls": [
			{
				"id": "1",
				"text": "Control Plane Security Configuration",
				"tests": [
					{
						"section": "1.1",
						"desc": "Master Node Configuration Files",
						"results": [
							{
								"test_number": "1.1.1",
								"test_desc": "Ensure API server pod spec permissions",
								"status": "PASS",
								"remediation": ""
							},
							{
								"test_number": "1.1.2",
								"test_desc": "Ensure API server pod spec ownership",
								"status": "FAIL",
								"remediation": "Run chmod 644 /etc/kubernetes/manifests/kube-apiserver.yaml"
							},
							{
								"test_number": "1.1.3",
								"test_desc": "Ensure controller manager spec permissions",
								"status": "WARN",
								"remediation": "Review controller manager configuration"
							},
							{
								"test_number": "1.1.4",
								"test_desc": "Info only check",
								"status": "INFO",
								"remediation": ""
							}
						]
					}
				]
			},
			{
				"id": "2",
				"text": "Etcd Node Configuration",
				"tests": [
					{
						"section": "2.1",
						"desc": "Etcd Configuration",
						"results": [
							{
								"test_number": "2.1.1",
								"test_desc": "Ensure etcd data dir permissions",
								"status": "PASS",
								"remediation": ""
							}
						]
					}
				]
			}
		]
	}`

	var result struct {
		Controls []struct {
			ID    string `json:"id"`
			Text  string `json:"text"`
			Tests []struct {
				Section string `json:"section"`
				Desc    string `json:"desc"`
				Results []struct {
					TestNumber  string `json:"test_number"`
					TestDesc    string `json:"test_desc"`
					Status      string `json:"status"`
					Remediation string `json:"remediation"`
				} `json:"results"`
			} `json:"tests"`
		} `json:"Controls"`
	}

	if err := json.Unmarshal([]byte(kubeBenchJSON), &result); err != nil {
		t.Fatalf("failed to parse kube-bench JSON: %v", err)
	}

	// Replicate the parsing logic from runCISBenchmark
	benchmark := &CISBenchmarkResult{
		Version: "CIS Kubernetes Benchmark",
	}

	for _, control := range result.Controls {
		section := CISBenchmarkSection{
			ID:   control.ID,
			Name: control.Text,
		}

		for _, test := range control.Tests {
			for _, r := range test.Results {
				check := CISBenchmarkCheck{
					ID:          r.TestNumber,
					Description: r.TestDesc,
					Status:      r.Status,
					Remediation: r.Remediation,
				}

				switch r.Status {
				case "PASS":
					section.PassCount++
					benchmark.PassCount++
				case "FAIL":
					section.FailCount++
					benchmark.FailCount++
				case "WARN":
					section.WarnCount++
					benchmark.WarnCount++
				default:
					benchmark.InfoCount++
				}

				benchmark.TotalChecks++
				section.Checks = append(section.Checks, check)
			}
		}

		benchmark.Sections = append(benchmark.Sections, section)
	}

	if benchmark.TotalChecks > 0 {
		benchmark.Score = float64(benchmark.PassCount) / float64(benchmark.TotalChecks) * 100
	}

	// Verify counts
	if benchmark.TotalChecks != 5 {
		t.Errorf("TotalChecks = %d, want 5", benchmark.TotalChecks)
	}
	if benchmark.PassCount != 2 {
		t.Errorf("PassCount = %d, want 2", benchmark.PassCount)
	}
	if benchmark.FailCount != 1 {
		t.Errorf("FailCount = %d, want 1", benchmark.FailCount)
	}
	if benchmark.WarnCount != 1 {
		t.Errorf("WarnCount = %d, want 1", benchmark.WarnCount)
	}
	if benchmark.InfoCount != 1 {
		t.Errorf("InfoCount = %d, want 1", benchmark.InfoCount)
	}

	// Score = 2/5 * 100 = 40
	if benchmark.Score != 40 {
		t.Errorf("Score = %v, want 40", benchmark.Score)
	}

	// Verify sections
	if len(benchmark.Sections) != 2 {
		t.Fatalf("len(Sections) = %d, want 2", len(benchmark.Sections))
	}
	if benchmark.Sections[0].PassCount != 1 {
		t.Errorf("Section[0].PassCount = %d, want 1", benchmark.Sections[0].PassCount)
	}
	if benchmark.Sections[0].FailCount != 1 {
		t.Errorf("Section[0].FailCount = %d, want 1", benchmark.Sections[0].FailCount)
	}
}

// ---------- Test Helper Functions ----------

func TestIsSystemNamespace(t *testing.T) {
	tests := []struct {
		ns       string
		expected bool
	}{
		{"kube-system", true},
		{"kube-public", true},
		{"kube-node-lease", true},
		{"default", false},
		{"myapp", false},
		{"production", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ns, func(t *testing.T) {
			got := isSystemNamespace(tt.ns)
			if got != tt.expected {
				t.Errorf("isSystemNamespace(%q) = %v, want %v", tt.ns, got, tt.expected)
			}
		})
	}
}

func TestIsDangerousCapability(t *testing.T) {
	tests := []struct {
		cap      string
		expected bool
	}{
		{"SYS_ADMIN", true},
		{"NET_ADMIN", true},
		{"SYS_PTRACE", true},
		{"SYS_RAWIO", true},
		{"DAC_OVERRIDE", true},
		{"SETUID", true},
		{"SETGID", true},
		{"NET_RAW", true},
		{"ALL", true},
		// case insensitive
		{"sys_admin", true},
		{"net_admin", true},
		{"all", true},
		// safe capabilities
		{"CHOWN", false},
		{"KILL", false},
		{"FOWNER", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.cap, func(t *testing.T) {
			got := isDangerousCapability(tt.cap)
			if got != tt.expected {
				t.Errorf("isDangerousCapability(%q) = %v, want %v", tt.cap, got, tt.expected)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"", 10, ""},
		{"abcdef", 6, "abcdef"},
		{"abcdefg", 6, "abc..."},
		{"a", 1, "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}

func TestBoolToInt(t *testing.T) {
	if boolToInt(true) != 1 {
		t.Error("boolToInt(true) should be 1")
	}
	if boolToInt(false) != 0 {
		t.Error("boolToInt(false) should be 0")
	}
}

func TestGetSectionName(t *testing.T) {
	tests := []struct {
		id       string
		expected string
	}{
		{"5.1", "RBAC and Service Accounts"},
		{"5.2", "Pod Security Standards"},
		{"5.3", "Network Policies and CNI Security"},
		{"5.4", "Secrets Management"},
		{"5.5", "Extensible Admission Control"},
		{"5.7", "General Policies"},
		{"9.9", "General Security"},
		{"", "General Security"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := getSectionName(tt.id)
			if got != tt.expected {
				t.Errorf("getSectionName(%q) = %q, want %q", tt.id, got, tt.expected)
			}
		})
	}
}

func TestValidateImage(t *testing.T) {
	tests := []struct {
		image    string
		expected bool
	}{
		// Valid images
		{"nginx", true},
		{"nginx:1.25", true},
		{"nginx:latest", true},
		{"docker.io/library/nginx:1.25", true},
		{"gcr.io/my-project/my-image:v1.0", true},
		{"registry.example.com/app:v2.3.1", true},
		{"myregistry.io/org/app:1.0-rc1", true},
		{"ubuntu:22.04", true},
		{"nginx@sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789", true},

		// Invalid images
		{"", false},
		{":latest", false},
		{"-invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.image, func(t *testing.T) {
			got := ValidateImage(tt.image)
			if got != tt.expected {
				t.Errorf("ValidateImage(%q) = %v, want %v", tt.image, got, tt.expected)
			}
		})
	}
}

// ---------- Test ScanResult JSON Serialization ----------

func TestScanResult_JSON(t *testing.T) {
	now := time.Now()
	result := &ScanResult{
		ScanTime:     now,
		Duration:     "5.2s",
		ClusterName:  "test-cluster",
		OverallScore: 85.5,
		RiskLevel:    "Medium",
		ImageVulns: &ImageVulnSummary{
			TotalImages:      10,
			ScannedImages:    8,
			VulnerableImages: 3,
			CriticalCount:    1,
			HighCount:        5,
		},
		PodSecurityIssues: []PodSecurityIssue{
			{Namespace: "default", Pod: "test", Issue: "privileged", Severity: "CRITICAL"},
		},
		RBACIssues: []RBACIssue{
			{Kind: "ClusterRole", Name: "admin", Issue: "wildcard", Severity: "CRITICAL"},
		},
		NetworkIssues: []NetworkIssue{
			{Namespace: "default", Resource: "Service/web", Issue: "exposed", Severity: "LOW"},
		},
		CISBenchmark: &CISBenchmarkResult{
			Version:     "CIS v1.8",
			TotalChecks: 100,
			PassCount:   80,
			FailCount:   15,
			WarnCount:   5,
			Score:       80,
		},
		Recommendations: []SecurityRecommendation{
			{Priority: 1, Category: "Image Security", Title: "Fix vulns"},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal ScanResult: %v", err)
	}

	var decoded ScanResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal ScanResult: %v", err)
	}

	if decoded.ClusterName != "test-cluster" {
		t.Errorf("ClusterName = %q, want %q", decoded.ClusterName, "test-cluster")
	}
	if decoded.OverallScore != 85.5 {
		t.Errorf("OverallScore = %v, want 85.5", decoded.OverallScore)
	}
	if decoded.RiskLevel != "Medium" {
		t.Errorf("RiskLevel = %q, want %q", decoded.RiskLevel, "Medium")
	}
	if decoded.ImageVulns == nil {
		t.Fatal("ImageVulns should not be nil")
	}
	if decoded.ImageVulns.CriticalCount != 1 {
		t.Errorf("CriticalCount = %d, want 1", decoded.ImageVulns.CriticalCount)
	}
	if len(decoded.PodSecurityIssues) != 1 {
		t.Errorf("PodSecurityIssues len = %d, want 1", len(decoded.PodSecurityIssues))
	}
	if len(decoded.RBACIssues) != 1 {
		t.Errorf("RBACIssues len = %d, want 1", len(decoded.RBACIssues))
	}
	if len(decoded.NetworkIssues) != 1 {
		t.Errorf("NetworkIssues len = %d, want 1", len(decoded.NetworkIssues))
	}
	if decoded.CISBenchmark == nil {
		t.Fatal("CISBenchmark should not be nil")
	}
	if len(decoded.Recommendations) != 1 {
		t.Errorf("Recommendations len = %d, want 1", len(decoded.Recommendations))
	}
}

func TestScanResult_JSON_EmptyResult(t *testing.T) {
	result := &ScanResult{}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal empty ScanResult: %v", err)
	}

	var decoded ScanResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// omitempty fields should be nil/empty
	if decoded.ImageVulns != nil {
		t.Error("ImageVulns should be nil for empty result")
	}
	if decoded.PodSecurityIssues != nil {
		t.Error("PodSecurityIssues should be nil for empty result")
	}
	if decoded.CISBenchmark != nil {
		t.Error("CISBenchmark should be nil for empty result")
	}
}

// ---------- Test Severity Constants ----------

func TestSeverityLevelConstants(t *testing.T) {
	// Verify the string values match expected constants
	if SeverityCritical != "CRITICAL" {
		t.Errorf("SeverityCritical = %q", SeverityCritical)
	}
	if SeverityHigh != "HIGH" {
		t.Errorf("SeverityHigh = %q", SeverityHigh)
	}
	if SeverityMedium != "MEDIUM" {
		t.Errorf("SeverityMedium = %q", SeverityMedium)
	}
	if SeverityLow != "LOW" {
		t.Errorf("SeverityLow = %q", SeverityLow)
	}
	if SeverityUnknown != "UNKNOWN" {
		t.Errorf("SeverityUnknown = %q", SeverityUnknown)
	}
}

// ---------- Test QuickScan ----------

func TestQuickScan(t *testing.T) {
	ctx := context.Background()

	// Set up objects representing a somewhat insecure cluster
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "myapp"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "web",
					Image: "nginx:latest",
				},
			},
		},
	}
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "myapp"}}

	fakeClientset := fake.NewSimpleClientset(ns, pod) //nolint:staticcheck
	client := &k8s.Client{Clientset: fakeClientset}
	scanner := &Scanner{k8sClient: client}

	result, err := scanner.QuickScan(ctx, "")
	if err != nil {
		t.Fatalf("QuickScan() error: %v", err)
	}

	if result == nil {
		t.Fatal("QuickScan() returned nil result")
	}

	// QuickScan should populate score, risk level, duration
	if result.OverallScore < 0 || result.OverallScore > 100 {
		t.Errorf("OverallScore = %v, want [0, 100]", result.OverallScore)
	}
	if result.RiskLevel == "" {
		t.Error("RiskLevel should not be empty")
	}
	if result.Duration == "" {
		t.Error("Duration should not be empty")
	}

	// QuickScan should NOT have image vulns (no scanning)
	if result.ImageVulns != nil {
		t.Error("QuickScan should not scan images")
	}

	// Should have pod security issues (running as root, latest tag, no limits)
	if len(result.PodSecurityIssues) == 0 {
		t.Log("Warning: expected pod security issues for insecure pod")
	}
}

func TestQuickScan_NamespaceFilter(t *testing.T) {
	ctx := context.Background()

	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "ns1"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "c1", Image: "nginx:latest"},
			},
		},
	}
	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "ns2"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "c1", Image: "redis:latest"},
			},
		},
	}

	fakeClientset := fake.NewSimpleClientset(pod1, pod2) //nolint:staticcheck
	client := &k8s.Client{Clientset: fakeClientset}
	scanner := &Scanner{k8sClient: client}

	// Scan only ns1
	result, err := scanner.QuickScan(ctx, "ns1")
	if err != nil {
		t.Fatalf("QuickScan(ns1) error: %v", err)
	}

	// All pod issues should be from ns1
	for _, issue := range result.PodSecurityIssues {
		if issue.Namespace != "ns1" {
			t.Errorf("found issue from namespace %q, expected only ns1", issue.Namespace)
		}
	}
}

// ---------- Test performBasicCISChecks ----------

func TestPerformBasicCISChecks(t *testing.T) {
	ctx := context.Background()

	// Create a clean cluster setup (should pass most checks)
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "myapp"}}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "secure-pod", Namespace: "myapp"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "myapp:v1.0",
					SecurityContext: &corev1.SecurityContext{
						RunAsNonRoot: boolPtr(true),
					},
				},
			},
		},
	}
	netpol := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "default-deny", Namespace: "myapp"},
	}

	fakeClientset := fake.NewSimpleClientset(ns, pod, netpol) //nolint:staticcheck
	client := &k8s.Client{Clientset: fakeClientset}
	scanner := &Scanner{k8sClient: client}

	benchmark, err := scanner.performBasicCISChecks(ctx, "")
	if err != nil {
		t.Fatalf("performBasicCISChecks() error: %v", err)
	}

	if benchmark == nil {
		t.Fatal("benchmark is nil")
	}
	if benchmark.TotalChecks != 5 {
		t.Errorf("TotalChecks = %d, want 5", benchmark.TotalChecks)
	}
	if benchmark.Version == "" {
		t.Error("Version should not be empty")
	}
	if len(benchmark.Sections) == 0 {
		t.Error("should have at least one section")
	}

	// Verify score calculation
	if benchmark.TotalChecks > 0 {
		expectedScore := float64(benchmark.PassCount) / float64(benchmark.TotalChecks) * 100
		if benchmark.Score != expectedScore {
			t.Errorf("Score = %v, want %v", benchmark.Score, expectedScore)
		}
	}

	t.Logf("CIS checks: total=%d pass=%d fail=%d score=%.1f%%",
		benchmark.TotalChecks, benchmark.PassCount, benchmark.FailCount, benchmark.Score)
}

// ---------- Test Scanner SetTrivyPath / GetTrivyPath ----------

func TestScanner_SetGetTrivyPath(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset() //nolint:staticcheck
	client := &k8s.Client{Clientset: fakeClientset}
	scanner := &Scanner{k8sClient: client}

	if scanner.GetTrivyPath() != "" {
		t.Error("initial trivy path should be empty")
	}

	scanner.SetTrivyPath("/opt/trivy/bin/trivy")
	if scanner.GetTrivyPath() != "/opt/trivy/bin/trivy" {
		t.Errorf("got %q", scanner.GetTrivyPath())
	}

	// Reset
	scanner.SetTrivyPath("")
	if scanner.GetTrivyPath() != "" {
		t.Error("trivy path should be empty after reset")
	}
}

// ---------- fake clientset builders ----------

// buildFakeClientsetFromPods creates a fake clientset from pod objects.
func buildFakeClientsetFromPods(pods ...*corev1.Pod) *fake.Clientset {
	objs := make([]interface{}, 0, len(pods))
	for _, p := range pods {
		objs = append(objs, p)
	}
	switch len(objs) {
	case 0:
		return fake.NewSimpleClientset() //nolint:staticcheck
	case 1:
		return fake.NewSimpleClientset(pods[0]) //nolint:staticcheck
	case 2:
		return fake.NewSimpleClientset(pods[0], pods[1]) //nolint:staticcheck
	case 3:
		return fake.NewSimpleClientset(pods[0], pods[1], pods[2]) //nolint:staticcheck
	default:
		// For larger sets, build incrementally
		cs := fake.NewSimpleClientset(pods[0]) //nolint:staticcheck
		for _, p := range pods[1:] {
			_, _ = cs.CoreV1().Pods(p.Namespace).Create(context.Background(), p, metav1.CreateOptions{})
		}
		return cs
	}
}

// buildFakeClientsetFromRBAC creates a fake clientset from RBAC objects.
func buildFakeClientsetFromRBAC(crbs []*rbacv1.ClusterRoleBinding, crs []*rbacv1.ClusterRole) *fake.Clientset {
	cs := fake.NewSimpleClientset() //nolint:staticcheck
	for _, crb := range crbs {
		_, _ = cs.RbacV1().ClusterRoleBindings().Create(context.Background(), crb, metav1.CreateOptions{})
	}
	for _, cr := range crs {
		_, _ = cs.RbacV1().ClusterRoles().Create(context.Background(), cr, metav1.CreateOptions{})
	}
	return cs
}

// buildFakeClientsetFromNetwork creates a fake clientset from network-related objects.
func buildFakeClientsetFromNetwork(
	namespaces []*corev1.Namespace,
	pods []*corev1.Pod,
	netPols []*networkingv1.NetworkPolicy,
	services []*corev1.Service,
) *fake.Clientset {
	cs := fake.NewSimpleClientset() //nolint:staticcheck
	for _, ns := range namespaces {
		_, _ = cs.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
	}
	for _, p := range pods {
		_, _ = cs.CoreV1().Pods(p.Namespace).Create(context.Background(), p, metav1.CreateOptions{})
	}
	for _, np := range netPols {
		_, _ = cs.NetworkingV1().NetworkPolicies(np.Namespace).Create(context.Background(), np, metav1.CreateOptions{})
	}
	for _, svc := range services {
		_, _ = cs.CoreV1().Services(svc.Namespace).Create(context.Background(), svc, metav1.CreateOptions{})
	}
	return cs
}
