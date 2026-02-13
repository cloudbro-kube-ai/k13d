package ui

import (
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildDeploymentTree(t *testing.T) {
	deps := []appsv1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "nginx"},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(3),
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 3,
			},
		},
	}

	rsList := []appsv1.ReplicaSet{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "nginx-abc123",
				OwnerReferences: []metav1.OwnerReference{
					{Name: "nginx", Kind: "Deployment"},
				},
			},
			Spec: appsv1.ReplicaSetSpec{
				Replicas: int32Ptr(3),
			},
			Status: appsv1.ReplicaSetStatus{
				ReadyReplicas: 3,
			},
		},
	}

	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "nginx-abc123-pod1",
				OwnerReferences: []metav1.OwnerReference{
					{Name: "nginx-abc123", Kind: "ReplicaSet"},
				},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "nginx-abc123-pod2",
				OwnerReferences: []metav1.OwnerReference{
					{Name: "nginx-abc123", Kind: "ReplicaSet"},
				},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
	}

	tree := BuildDeploymentTree(deps, rsList, pods)

	if len(tree) != 1 {
		t.Fatalf("expected 1 deployment node, got %d", len(tree))
	}

	depNode := tree[0]
	if !strings.Contains(depNode.Text, "Deployment/nginx") {
		t.Errorf("expected 'Deployment/nginx' in text, got %q", depNode.Text)
	}
	if !strings.Contains(depNode.Text, "3/3 ready") {
		t.Errorf("expected '3/3 ready' in text, got %q", depNode.Text)
	}

	if len(depNode.Children) != 1 {
		t.Fatalf("expected 1 replicaset child, got %d", len(depNode.Children))
	}

	rsNode := depNode.Children[0]
	if !strings.Contains(rsNode.Text, "ReplicaSet/nginx-abc123") {
		t.Errorf("expected 'ReplicaSet/nginx-abc123' in text, got %q", rsNode.Text)
	}

	if len(rsNode.Children) != 2 {
		t.Fatalf("expected 2 pod children, got %d", len(rsNode.Children))
	}

	for _, pod := range rsNode.Children {
		if !strings.Contains(pod.Text, "Pod/nginx-abc123-pod") {
			t.Errorf("expected 'Pod/nginx-abc123-pod*' in text, got %q", pod.Text)
		}
		if !strings.Contains(pod.Text, "Running") {
			t.Errorf("expected 'Running' in pod text, got %q", pod.Text)
		}
	}
}

func TestBuildDeploymentTree_SkipsZeroReplicaRS(t *testing.T) {
	deps := []appsv1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "app"},
			Spec:       appsv1.DeploymentSpec{Replicas: int32Ptr(2)},
			Status:     appsv1.DeploymentStatus{ReadyReplicas: 2},
		},
	}

	rsList := []appsv1.ReplicaSet{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "app-old",
				OwnerReferences: []metav1.OwnerReference{{Name: "app", Kind: "Deployment"}},
			},
			Spec:   appsv1.ReplicaSetSpec{Replicas: int32Ptr(0)},
			Status: appsv1.ReplicaSetStatus{Replicas: 0},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "app-current",
				OwnerReferences: []metav1.OwnerReference{{Name: "app", Kind: "Deployment"}},
			},
			Spec:   appsv1.ReplicaSetSpec{Replicas: int32Ptr(2)},
			Status: appsv1.ReplicaSetStatus{ReadyReplicas: 2},
		},
	}

	tree := BuildDeploymentTree(deps, rsList, nil)
	if len(tree) != 1 {
		t.Fatal("expected 1 deployment")
	}
	if len(tree[0].Children) != 1 {
		t.Errorf("expected 1 RS (old one skipped), got %d", len(tree[0].Children))
	}
	if !strings.Contains(tree[0].Children[0].Text, "app-current") {
		t.Errorf("expected current RS, got %q", tree[0].Children[0].Text)
	}
}

func TestBuildStatefulSetTree(t *testing.T) {
	stses := []appsv1.StatefulSet{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "redis"},
			Spec:       appsv1.StatefulSetSpec{Replicas: int32Ptr(3)},
			Status:     appsv1.StatefulSetStatus{ReadyReplicas: 3},
		},
	}

	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "redis-0",
				OwnerReferences: []metav1.OwnerReference{{Name: "redis", Kind: "StatefulSet"}},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "redis-1",
				OwnerReferences: []metav1.OwnerReference{{Name: "redis", Kind: "StatefulSet"}},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
	}

	tree := BuildStatefulSetTree(stses, pods)
	if len(tree) != 1 {
		t.Fatalf("expected 1 STS node, got %d", len(tree))
	}
	if !strings.Contains(tree[0].Text, "StatefulSet/redis") {
		t.Errorf("expected 'StatefulSet/redis' in text, got %q", tree[0].Text)
	}
	if len(tree[0].Children) != 2 {
		t.Errorf("expected 2 pod children, got %d", len(tree[0].Children))
	}
}

func TestBuildJobTree(t *testing.T) {
	jobs := []batchv1.Job{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "backup"},
			Status: batchv1.JobStatus{
				Succeeded: 1,
				Active:    0,
			},
		},
	}

	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "backup-pod1",
				OwnerReferences: []metav1.OwnerReference{{Name: "backup", Kind: "Job"}},
			},
			Status: corev1.PodStatus{Phase: corev1.PodSucceeded},
		},
	}

	tree := BuildJobTree(jobs, pods)
	if len(tree) != 1 {
		t.Fatalf("expected 1 job node, got %d", len(tree))
	}
	if !strings.Contains(tree[0].Text, "Job/backup") {
		t.Errorf("expected 'Job/backup' in text, got %q", tree[0].Text)
	}
	if !strings.Contains(tree[0].Text, "Complete") {
		t.Errorf("expected 'Complete' in job text, got %q", tree[0].Text)
	}
	if len(tree[0].Children) != 1 {
		t.Errorf("expected 1 pod child, got %d", len(tree[0].Children))
	}
}

func TestBuildCronJobTree(t *testing.T) {
	cjs := []batchv1.CronJob{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "nightly-backup"},
			Spec: batchv1.CronJobSpec{
				Schedule: "0 2 * * *",
			},
		},
	}

	jobs := []batchv1.Job{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "nightly-backup-123",
				OwnerReferences: []metav1.OwnerReference{{Name: "nightly-backup", Kind: "CronJob"}},
			},
			Status: batchv1.JobStatus{Succeeded: 1},
		},
	}

	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "nightly-backup-123-pod",
				OwnerReferences: []metav1.OwnerReference{{Name: "nightly-backup-123", Kind: "Job"}},
			},
			Status: corev1.PodStatus{Phase: corev1.PodSucceeded},
		},
	}

	tree := BuildCronJobTree(cjs, jobs, pods)
	if len(tree) != 1 {
		t.Fatalf("expected 1 cronjob node, got %d", len(tree))
	}
	if !strings.Contains(tree[0].Text, "CronJob/nightly-backup") {
		t.Errorf("expected 'CronJob/nightly-backup' in text, got %q", tree[0].Text)
	}
	if !strings.Contains(tree[0].Text, "0 2 * * *") {
		t.Errorf("expected schedule in text, got %q", tree[0].Text)
	}
	if len(tree[0].Children) != 1 {
		t.Fatalf("expected 1 job child, got %d", len(tree[0].Children))
	}
	if len(tree[0].Children[0].Children) != 1 {
		t.Errorf("expected 1 pod grandchild, got %d", len(tree[0].Children[0].Children))
	}
}

func TestBuildDaemonSetTree(t *testing.T) {
	dss := []appsv1.DaemonSet{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "fluentd"},
			Status: appsv1.DaemonSetStatus{
				NumberReady:            3,
				DesiredNumberScheduled: 3,
			},
		},
	}

	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "fluentd-node1",
				OwnerReferences: []metav1.OwnerReference{{Name: "fluentd", Kind: "DaemonSet"}},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
	}

	tree := BuildDaemonSetTree(dss, pods)
	if len(tree) != 1 {
		t.Fatalf("expected 1 daemonset node, got %d", len(tree))
	}
	if !strings.Contains(tree[0].Text, "DaemonSet/fluentd") {
		t.Errorf("expected 'DaemonSet/fluentd' in text, got %q", tree[0].Text)
	}
	if !strings.Contains(tree[0].Text, "3/3 ready") {
		t.Errorf("expected '3/3 ready' in text, got %q", tree[0].Text)
	}
}

func TestBuildDeploymentTree_Empty(t *testing.T) {
	tree := BuildDeploymentTree(nil, nil, nil)
	if len(tree) != 0 {
		t.Errorf("expected empty tree for nil inputs, got %d nodes", len(tree))
	}
}

func TestBuildStatefulSetTree_Empty(t *testing.T) {
	tree := BuildStatefulSetTree(nil, nil)
	if len(tree) != 0 {
		t.Errorf("expected empty tree, got %d", len(tree))
	}
}

func TestBuildJobTree_Empty(t *testing.T) {
	tree := BuildJobTree(nil, nil)
	if len(tree) != 0 {
		t.Errorf("expected empty tree, got %d", len(tree))
	}
}

func TestBuildCronJobTree_Empty(t *testing.T) {
	tree := BuildCronJobTree(nil, nil, nil)
	if len(tree) != 0 {
		t.Errorf("expected empty tree, got %d", len(tree))
	}
}

func TestBuildDaemonSetTree_Empty(t *testing.T) {
	tree := BuildDaemonSetTree(nil, nil)
	if len(tree) != 0 {
		t.Errorf("expected empty tree, got %d", len(tree))
	}
}

func TestPodStatusText(t *testing.T) {
	tests := []struct {
		name   string
		pod    corev1.Pod
		expect string
	}{
		{
			name:   "running",
			pod:    corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning}},
			expect: "Running",
		},
		{
			name:   "pending",
			pod:    corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending}},
			expect: "Pending",
		},
		{
			name:   "failed",
			pod:    corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodFailed}},
			expect: "Failed",
		},
		{
			name: "crashloop",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}},
					},
				},
			},
			expect: "CrashLoopBackOff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := podStatusText(tt.pod)
			if !strings.Contains(result, tt.expect) {
				t.Errorf("podStatusText() = %q, want it to contain %q", result, tt.expect)
			}
		})
	}
}

func TestJobStatusText(t *testing.T) {
	tests := []struct {
		name   string
		job    batchv1.Job
		expect string
	}{
		{
			name:   "complete",
			job:    batchv1.Job{Status: batchv1.JobStatus{Succeeded: 1, Active: 0}},
			expect: "Complete",
		},
		{
			name:   "active",
			job:    batchv1.Job{Status: batchv1.JobStatus{Active: 1}},
			expect: "Active",
		},
		{
			name:   "failed",
			job:    batchv1.Job{Status: batchv1.JobStatus{Failed: 1, Active: 0}},
			expect: "Failed",
		},
		{
			name:   "pending",
			job:    batchv1.Job{},
			expect: "Pending",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := jobStatusText(tt.job)
			if !strings.Contains(result, tt.expect) {
				t.Errorf("jobStatusText() = %q, want it to contain %q", result, tt.expect)
			}
		})
	}
}

func TestNormalizeResourceType(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"deploy", "deployments"},
		{"deployments", "deployments"},
		{"sts", "statefulsets"},
		{"job", "jobs"},
		{"cj", "cronjobs"},
		{"ds", "daemonsets"},
		{"unknown", "deployments"}, // defaults to deployments
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalizeResourceType(tt.input); got != tt.expect {
				t.Errorf("normalizeResourceType(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

func TestIsOwnedBy(t *testing.T) {
	refs := []metav1.OwnerReference{
		{Name: "nginx", Kind: "Deployment"},
		{Name: "redis", Kind: "StatefulSet"},
	}

	if !isOwnedBy(refs, "nginx", "Deployment") {
		t.Error("expected isOwnedBy to return true for nginx/Deployment")
	}
	if isOwnedBy(refs, "nginx", "StatefulSet") {
		t.Error("expected isOwnedBy to return false for nginx/StatefulSet")
	}
	if isOwnedBy(refs, "missing", "Deployment") {
		t.Error("expected isOwnedBy to return false for missing resource")
	}
	if isOwnedBy(nil, "nginx", "Deployment") {
		t.Error("expected isOwnedBy to return false for nil refs")
	}
}

func TestNewXRayView(t *testing.T) {
	app := &App{}
	xv := NewXRayView(app, "deploy", "default")

	if xv == nil {
		t.Fatal("NewXRayView returned nil")
	}
	if xv.app != app {
		t.Error("XRayView.app not set correctly")
	}
	title := xv.GetTitle()
	if !strings.Contains(title, "XRay") {
		t.Errorf("expected 'XRay' in title, got %q", title)
	}
}
