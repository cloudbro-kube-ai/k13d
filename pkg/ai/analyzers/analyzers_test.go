package analyzers

import (
	"context"
	"testing"
)

// --- Registry Tests ---

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	findings := r.AnalyzeAll(context.Background(), &ResourceInfo{Kind: "Pod"})
	if len(findings) != 0 {
		t.Errorf("empty registry should produce no findings, got %d", len(findings))
	}
}

func TestRegistryNilResource(t *testing.T) {
	r := DefaultRegistry()
	findings := r.AnalyzeAll(context.Background(), nil)
	if findings != nil {
		t.Errorf("nil resource should return nil findings, got %v", findings)
	}
}

func TestDefaultRegistryContainsAllAnalyzers(t *testing.T) {
	r := DefaultRegistry()
	if len(r.analyzers) != 4 {
		t.Errorf("DefaultRegistry should have 4 analyzers, got %d", len(r.analyzers))
	}
	names := map[string]bool{}
	for _, a := range r.analyzers {
		names[a.Name()] = true
	}
	for _, want := range []string{"pod", "service", "deployment", "node"} {
		if !names[want] {
			t.Errorf("DefaultRegistry missing analyzer %q", want)
		}
	}
}

// --- Pod Analyzer Tests ---

func TestPodAnalyzer_HealthyPod(t *testing.T) {
	a := &PodAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind:      "Pod",
		Name:      "healthy-pod",
		Namespace: "default",
		Status:    "Running",
		Containers: []ContainerInfo{
			{Name: "app", Ready: true, State: "running", RestartCount: 0},
		},
	})
	if len(findings) != 0 {
		t.Errorf("healthy pod should have no findings, got %d: %+v", len(findings), findings)
	}
}

func TestPodAnalyzer_WrongKind(t *testing.T) {
	a := &PodAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{Kind: "Service"})
	if len(findings) != 0 {
		t.Errorf("wrong kind should have no findings, got %d", len(findings))
	}
}

func TestPodAnalyzer_NilResource(t *testing.T) {
	a := &PodAnalyzer{}
	findings := a.Analyze(context.Background(), nil)
	if len(findings) != 0 {
		t.Errorf("nil resource should have no findings, got %d", len(findings))
	}
}

func TestPodAnalyzer_CrashLoopBackOff(t *testing.T) {
	a := &PodAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind:      "Pod",
		Name:      "crash-pod",
		Namespace: "default",
		Status:    "Running",
		Containers: []ContainerInfo{
			{Name: "app", Reason: "CrashLoopBackOff", RestartCount: 10, State: "waiting"},
		},
	})
	assertFindingExists(t, findings, SeverityCritical, "CrashLoopBackOff")
}

func TestPodAnalyzer_OOMKilled(t *testing.T) {
	a := &PodAnalyzer{}

	// Test by reason
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind:      "Pod",
		Name:      "oom-pod",
		Namespace: "default",
		Containers: []ContainerInfo{
			{Name: "app", Reason: "OOMKilled", ExitCode: 137},
		},
	})
	assertFindingExists(t, findings, SeverityCritical, "OOMKilled")

	// Test by exit code 137 only
	findings = a.Analyze(context.Background(), &ResourceInfo{
		Kind:      "Pod",
		Name:      "oom-pod2",
		Namespace: "default",
		Containers: []ContainerInfo{
			{Name: "app", ExitCode: 137},
		},
	})
	assertFindingExists(t, findings, SeverityCritical, "OOMKilled")
}

func TestPodAnalyzer_ImagePullBackOff(t *testing.T) {
	a := &PodAnalyzer{}
	for _, reason := range []string{"ImagePullBackOff", "ErrImagePull"} {
		findings := a.Analyze(context.Background(), &ResourceInfo{
			Kind:      "Pod",
			Name:      "img-pod",
			Namespace: "default",
			Containers: []ContainerInfo{
				{Name: "app", Reason: reason, Image: "nonexistent:latest", State: "waiting"},
			},
		})
		assertFindingExists(t, findings, SeverityCritical, "image pull error")
	}
}

func TestPodAnalyzer_HighRestartCount(t *testing.T) {
	a := &PodAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind:      "Pod",
		Name:      "restart-pod",
		Namespace: "default",
		Containers: []ContainerInfo{
			{Name: "app", RestartCount: 10, Ready: true, State: "running"},
		},
	})
	assertFindingExists(t, findings, SeverityWarning, "high restart count")
}

func TestPodAnalyzer_HighRestartNotDuplicated(t *testing.T) {
	a := &PodAnalyzer{}
	// When CrashLoopBackOff is the reason, high restart should not also fire
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind:      "Pod",
		Name:      "crash-pod",
		Namespace: "default",
		Containers: []ContainerInfo{
			{Name: "app", Reason: "CrashLoopBackOff", RestartCount: 10, State: "waiting"},
		},
	})
	count := 0
	for _, f := range findings {
		if containsSubstring(f.Title, "restart") {
			count++
		}
	}
	// CrashLoopBackOff finding mentions restarts, but high restart finding should be suppressed
	if count > 0 {
		// The high restart check explicitly skips CrashLoopBackOff
		for _, f := range findings {
			if containsSubstring(f.Title, "high restart") {
				t.Error("high restart finding should not fire when CrashLoopBackOff is already reported")
			}
		}
	}
}

func TestPodAnalyzer_ContainerNotReady(t *testing.T) {
	a := &PodAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind:      "Pod",
		Name:      "notready-pod",
		Namespace: "default",
		Containers: []ContainerInfo{
			{Name: "app", Ready: false, State: "running"},
		},
	})
	assertFindingExists(t, findings, SeverityWarning, "not ready")
}

func TestPodAnalyzer_PendingNoEvents(t *testing.T) {
	a := &PodAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind:      "Pod",
		Name:      "pending-pod",
		Namespace: "default",
		Status:    "Pending",
	})
	assertFindingExists(t, findings, SeverityWarning, "Pending")
}

func TestPodAnalyzer_PendingWithSchedulingEvents(t *testing.T) {
	a := &PodAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind:      "Pod",
		Name:      "pending-pod",
		Namespace: "default",
		Status:    "Pending",
		Events: []Event{
			{Type: "Normal", Reason: "Scheduled", Message: "Successfully assigned"},
		},
	})
	// Should not report pending issue if there are scheduling events
	for _, f := range findings {
		if containsSubstring(f.Title, "Pending") {
			t.Error("should not report pending when scheduling events exist")
		}
	}
}

func TestPodAnalyzer_MultipleIssues(t *testing.T) {
	a := &PodAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind:      "Pod",
		Name:      "multi-issue-pod",
		Namespace: "default",
		Containers: []ContainerInfo{
			{Name: "app", Reason: "CrashLoopBackOff", RestartCount: 10},
			{Name: "sidecar", Reason: "ImagePullBackOff", Image: "bad:latest"},
		},
	})
	if len(findings) < 2 {
		t.Errorf("expected at least 2 findings for multiple issues, got %d", len(findings))
	}
}

func TestPodAnalyzer_EmptyContainers(t *testing.T) {
	a := &PodAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind:      "Pod",
		Name:      "empty-pod",
		Namespace: "default",
		Status:    "Running",
	})
	if len(findings) != 0 {
		t.Errorf("pod with no containers and Running status should have no findings, got %d", len(findings))
	}
}

// --- Service Analyzer Tests ---

func TestServiceAnalyzer_HealthyService(t *testing.T) {
	a := &ServiceAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind:      "Service",
		Name:      "healthy-svc",
		Namespace: "default",
	})
	if len(findings) != 0 {
		t.Errorf("healthy service should have no findings, got %d", len(findings))
	}
}

func TestServiceAnalyzer_WrongKind(t *testing.T) {
	a := &ServiceAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{Kind: "Pod"})
	if len(findings) != 0 {
		t.Errorf("wrong kind should have no findings, got %d", len(findings))
	}
}

func TestServiceAnalyzer_NilResource(t *testing.T) {
	a := &ServiceAnalyzer{}
	findings := a.Analyze(context.Background(), nil)
	if len(findings) != 0 {
		t.Errorf("nil resource should have no findings, got %d", len(findings))
	}
}

func TestServiceAnalyzer_NoEndpointsEvent(t *testing.T) {
	a := &ServiceAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind:      "Service",
		Name:      "noep-svc",
		Namespace: "default",
		Events: []Event{
			{Reason: "NoEndpoints", Message: "no matching pods found"},
		},
	})
	assertFindingExists(t, findings, SeverityWarning, "no endpoints")
}

func TestServiceAnalyzer_NoEndpointsCondition(t *testing.T) {
	a := &ServiceAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind:      "Service",
		Name:      "noep-svc",
		Namespace: "default",
		Conditions: []Condition{
			{Type: "NoEndpoints", Status: "True", Message: "selector matches no pods"},
		},
	})
	assertFindingExists(t, findings, SeverityWarning, "no endpoints")
}

func TestServiceAnalyzer_MismatchedPorts(t *testing.T) {
	a := &ServiceAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind:      "Service",
		Name:      "port-svc",
		Namespace: "default",
		Events: []Event{
			{Reason: "PortMismatch", Message: "service port 80 doesn't match container port 8080"},
		},
	})
	assertFindingExists(t, findings, SeverityWarning, "port mismatch")
}

// --- Deployment Analyzer Tests ---

func TestDeploymentAnalyzer_HealthyDeployment(t *testing.T) {
	a := &DeploymentAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind:      "Deployment",
		Name:      "healthy-deploy",
		Namespace: "default",
		Status:    "3/3",
		Conditions: []Condition{
			{Type: "Available", Status: "True"},
			{Type: "Progressing", Status: "True"},
		},
	})
	if len(findings) != 0 {
		t.Errorf("healthy deployment should have no findings, got %d: %+v", len(findings), findings)
	}
}

func TestDeploymentAnalyzer_WrongKind(t *testing.T) {
	a := &DeploymentAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{Kind: "Pod"})
	if len(findings) != 0 {
		t.Errorf("wrong kind should have no findings, got %d", len(findings))
	}
}

func TestDeploymentAnalyzer_NilResource(t *testing.T) {
	a := &DeploymentAnalyzer{}
	findings := a.Analyze(context.Background(), nil)
	if len(findings) != 0 {
		t.Errorf("nil resource should have no findings, got %d", len(findings))
	}
}

func TestDeploymentAnalyzer_UnavailableReplicas(t *testing.T) {
	a := &DeploymentAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind:      "Deployment",
		Name:      "unavail-deploy",
		Namespace: "default",
		Status:    "1/3",
		Conditions: []Condition{
			{Type: "Available", Status: "False", Reason: "MinimumReplicasUnavailable", Message: "Deployment does not have minimum availability"},
		},
	})
	assertFindingExists(t, findings, SeverityCritical, "unavailable replicas")
}

func TestDeploymentAnalyzer_RolloutStuck(t *testing.T) {
	a := &DeploymentAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind:      "Deployment",
		Name:      "stuck-deploy",
		Namespace: "default",
		Conditions: []Condition{
			{Type: "Progressing", Status: "False", Reason: "ProgressDeadlineExceeded", Message: "deadline exceeded"},
		},
	})
	assertFindingExists(t, findings, SeverityCritical, "rollout is stuck")
}

func TestDeploymentAnalyzer_ZeroReplicas(t *testing.T) {
	a := &DeploymentAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind:      "Deployment",
		Name:      "zero-deploy",
		Namespace: "default",
		Status:    "0/0",
	})
	assertFindingExists(t, findings, SeverityInfo, "scaled to zero")
}

func TestDeploymentAnalyzer_MultipleIssues(t *testing.T) {
	a := &DeploymentAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind:      "Deployment",
		Name:      "bad-deploy",
		Namespace: "default",
		Conditions: []Condition{
			{Type: "Available", Status: "False", Reason: "MinimumReplicasUnavailable"},
			{Type: "Progressing", Status: "False", Reason: "ProgressDeadlineExceeded"},
		},
	})
	if len(findings) < 2 {
		t.Errorf("expected at least 2 findings, got %d", len(findings))
	}
}

// --- Node Analyzer Tests ---

func TestNodeAnalyzer_HealthyNode(t *testing.T) {
	a := &NodeAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind:   "Node",
		Name:   "healthy-node",
		Status: "Ready",
		Conditions: []Condition{
			{Type: "Ready", Status: "True"},
			{Type: "DiskPressure", Status: "False"},
			{Type: "MemoryPressure", Status: "False"},
			{Type: "PIDPressure", Status: "False"},
		},
	})
	if len(findings) != 0 {
		t.Errorf("healthy node should have no findings, got %d: %+v", len(findings), findings)
	}
}

func TestNodeAnalyzer_WrongKind(t *testing.T) {
	a := &NodeAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{Kind: "Pod"})
	if len(findings) != 0 {
		t.Errorf("wrong kind should have no findings, got %d", len(findings))
	}
}

func TestNodeAnalyzer_NilResource(t *testing.T) {
	a := &NodeAnalyzer{}
	findings := a.Analyze(context.Background(), nil)
	if len(findings) != 0 {
		t.Errorf("nil resource should have no findings, got %d", len(findings))
	}
}

func TestNodeAnalyzer_NotReady(t *testing.T) {
	a := &NodeAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind: "Node",
		Name: "bad-node",
		Conditions: []Condition{
			{Type: "Ready", Status: "False", Reason: "KubeletNotReady", Message: "container runtime is down"},
		},
	})
	assertFindingExists(t, findings, SeverityCritical, "NotReady")
}

func TestNodeAnalyzer_ReadyUnknown(t *testing.T) {
	a := &NodeAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind: "Node",
		Name: "unknown-node",
		Conditions: []Condition{
			{Type: "Ready", Status: "Unknown", Reason: "NodeStatusUnknown"},
		},
	})
	assertFindingExists(t, findings, SeverityCritical, "NotReady")
}

func TestNodeAnalyzer_DiskPressure(t *testing.T) {
	a := &NodeAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind: "Node",
		Name: "disk-node",
		Conditions: []Condition{
			{Type: "Ready", Status: "True"},
			{Type: "DiskPressure", Status: "True", Message: "disk usage exceeds threshold"},
		},
	})
	assertFindingExists(t, findings, SeverityCritical, "DiskPressure")
}

func TestNodeAnalyzer_MemoryPressure(t *testing.T) {
	a := &NodeAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind: "Node",
		Name: "mem-node",
		Conditions: []Condition{
			{Type: "Ready", Status: "True"},
			{Type: "MemoryPressure", Status: "True", Message: "memory usage high"},
		},
	})
	assertFindingExists(t, findings, SeverityCritical, "MemoryPressure")
}

func TestNodeAnalyzer_PIDPressure(t *testing.T) {
	a := &NodeAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind: "Node",
		Name: "pid-node",
		Conditions: []Condition{
			{Type: "Ready", Status: "True"},
			{Type: "PIDPressure", Status: "True", Message: "too many processes"},
		},
	})
	assertFindingExists(t, findings, SeverityWarning, "PIDPressure")
}

func TestNodeAnalyzer_Unschedulable(t *testing.T) {
	a := &NodeAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind:   "Node",
		Name:   "cordoned-node",
		Status: "SchedulingDisabled",
		Conditions: []Condition{
			{Type: "Ready", Status: "True"},
		},
	})
	assertFindingExists(t, findings, SeverityInfo, "cordoned")
}

func TestNodeAnalyzer_MultipleIssues(t *testing.T) {
	a := &NodeAnalyzer{}
	findings := a.Analyze(context.Background(), &ResourceInfo{
		Kind: "Node",
		Name: "bad-node",
		Conditions: []Condition{
			{Type: "Ready", Status: "False", Reason: "KubeletNotReady"},
			{Type: "DiskPressure", Status: "True"},
			{Type: "MemoryPressure", Status: "True"},
		},
	})
	if len(findings) < 3 {
		t.Errorf("expected at least 3 findings for multiple issues, got %d: %+v", len(findings), findings)
	}
}

// --- Integration Test: Full Registry ---

func TestFullRegistryAnalysis(t *testing.T) {
	r := DefaultRegistry()

	// Test that each analyzer only fires for its resource type
	tests := []struct {
		name     string
		resource *ResourceInfo
		wantMin  int
	}{
		{
			name: "crashing pod",
			resource: &ResourceInfo{
				Kind: "Pod", Name: "crash", Namespace: "default",
				Containers: []ContainerInfo{
					{Name: "app", Reason: "CrashLoopBackOff", RestartCount: 10},
				},
			},
			wantMin: 1,
		},
		{
			name: "deployment unavailable",
			resource: &ResourceInfo{
				Kind: "Deployment", Name: "bad", Namespace: "default",
				Conditions: []Condition{{Type: "Available", Status: "False"}},
			},
			wantMin: 1,
		},
		{
			name: "node not ready",
			resource: &ResourceInfo{
				Kind: "Node", Name: "bad-node",
				Conditions: []Condition{{Type: "Ready", Status: "False"}},
			},
			wantMin: 1,
		},
		{
			name: "unknown kind",
			resource: &ResourceInfo{
				Kind: "ConfigMap", Name: "test", Namespace: "default",
			},
			wantMin: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := r.AnalyzeAll(context.Background(), tt.resource)
			if len(findings) < tt.wantMin {
				t.Errorf("expected at least %d findings, got %d: %+v", tt.wantMin, len(findings), findings)
			}
		})
	}
}

// --- Finding Quality Tests ---

func TestFindingsHaveRequiredFields(t *testing.T) {
	r := DefaultRegistry()
	resources := []*ResourceInfo{
		{
			Kind: "Pod", Name: "p", Namespace: "ns",
			Containers: []ContainerInfo{{Name: "c", Reason: "CrashLoopBackOff"}},
		},
		{
			Kind: "Deployment", Name: "d", Namespace: "ns",
			Conditions: []Condition{{Type: "Available", Status: "False"}},
		},
		{
			Kind: "Node", Name: "n",
			Conditions: []Condition{{Type: "Ready", Status: "False"}},
		},
	}

	for _, res := range resources {
		findings := r.AnalyzeAll(context.Background(), res)
		for i, f := range findings {
			if f.Analyzer == "" {
				t.Errorf("finding %d for %s: missing Analyzer field", i, res.Kind)
			}
			if f.Resource == "" {
				t.Errorf("finding %d for %s: missing Resource field", i, res.Kind)
			}
			if f.Severity == "" {
				t.Errorf("finding %d for %s: missing Severity field", i, res.Kind)
			}
			if f.Title == "" {
				t.Errorf("finding %d for %s: missing Title field", i, res.Kind)
			}
			if f.Details == "" {
				t.Errorf("finding %d for %s: missing Details field", i, res.Kind)
			}
			if len(f.Suggestions) == 0 {
				t.Errorf("finding %d for %s: missing Suggestions", i, res.Kind)
			}
		}
	}
}

// --- Test Helpers ---

func assertFindingExists(t *testing.T, findings []Finding, severity Severity, titleSubstring string) {
	t.Helper()
	for _, f := range findings {
		if f.Severity == severity && containsSubstring(f.Title, titleSubstring) {
			return
		}
	}
	t.Errorf("expected finding with severity=%s and title containing %q, got findings: %+v", severity, titleSubstring, findings)
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	// Case-insensitive substring search
	sl := toLower(s)
	subsl := toLower(substr)
	for i := 0; i <= len(sl)-len(subsl); i++ {
		if sl[i:i+len(subsl)] == subsl {
			return true
		}
	}
	return false
}

func toLower(s string) string {
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
