package ui

import (
	"strings"
	"testing"
)

func TestRenderPulse_AllHealthy(t *testing.T) {
	data := PulseData{
		PodsRunning:  10,
		PodsTotal:    10,
		DeploysReady: 3,
		DeploysTotal: 3,
		STSReady:     2,
		STSTotal:     2,
		DSReady:      1,
		DSTotal:      1,
		JobsComplete: 5,
		JobsTotal:    5,
		NodesReady:   3,
		NodesTotal:   3,
		CPUUsed:      6500,
		CPUCapacity:  10000,
		CPUAvail:     true,
		MemUsed:      4096,
		MemCapacity:  8192,
		MemAvail:     true,
	}

	result := RenderPulse(data)

	// Should contain healthy indicators
	if !strings.Contains(result, "✓ 10 Running") {
		t.Errorf("expected '✓ 10 Running' in output, got:\n%s", result)
	}
	if !strings.Contains(result, "✓ 3 Ready") {
		t.Errorf("expected '✓ 3 Ready' for deployments in output")
	}
	if !strings.Contains(result, "✓ 5 Complete") {
		t.Errorf("expected '✓ 5 Complete' for jobs in output")
	}

	// Should NOT contain warning/error indicators
	if strings.Contains(result, "⚠") {
		t.Errorf("unexpected warning indicator in healthy cluster output")
	}
	if strings.Contains(result, "✗") {
		t.Errorf("unexpected error indicator in healthy cluster output")
	}

	// Should contain CPU bar
	if !strings.Contains(result, "65%") {
		t.Errorf("expected CPU percentage in output")
	}
	if !strings.Contains(result, "6.5/10.0 cores") {
		t.Errorf("expected CPU cores in output")
	}

	// Should contain Memory bar
	if !strings.Contains(result, "50%") {
		t.Errorf("expected memory percentage in output")
	}
}

func TestRenderPulse_WithIssues(t *testing.T) {
	data := PulseData{
		PodsRunning:     8,
		PodsPending:     2,
		PodsFailed:      1,
		PodsTotal:       11,
		DeploysReady:    2,
		DeploysUpdating: 1,
		DeploysTotal:    3,
		NodesReady:      2,
		NodesNotReady:   1,
		NodesTotal:      3,
	}

	result := RenderPulse(data)

	if !strings.Contains(result, "⚠ 2 Pending") {
		t.Errorf("expected '⚠ 2 Pending' in output")
	}
	if !strings.Contains(result, "✗ 1 Failed") {
		t.Errorf("expected '✗ 1 Failed' in output")
	}
	if !strings.Contains(result, "⚠ 1 Updating") {
		t.Errorf("expected '⚠ 1 Updating' in output")
	}
	if !strings.Contains(result, "✗ 1 NotReady") {
		t.Errorf("expected '✗ 1 NotReady' for nodes in output")
	}
}

func TestRenderPulse_NoMetrics(t *testing.T) {
	data := PulseData{
		CPUAvail: false,
		MemAvail: false,
	}

	result := RenderPulse(data)

	if !strings.Contains(result, "N/A") {
		t.Errorf("expected 'N/A' for unavailable metrics")
	}
}

func TestRenderPulse_EmptyCluster(t *testing.T) {
	data := PulseData{}

	result := RenderPulse(data)

	// Should render without panic, with zero counts
	if !strings.Contains(result, "✓ 0 Running") {
		t.Errorf("expected '✓ 0 Running' in empty cluster output")
	}
	if !strings.Contains(result, "No recent events") {
		t.Errorf("expected 'No recent events' in empty cluster output")
	}
}

func TestRenderPulse_WithEvents(t *testing.T) {
	data := PulseData{
		Events: []PulseEvent{
			{Type: "Warning", Reason: "OOMKilled", Message: "Pod my-pod OOMKilled", Age: "5m"},
			{Type: "Normal", Reason: "Scaled", Message: "Deployment nginx scaled to 3", Age: "10m"},
		},
	}

	result := RenderPulse(data)

	if !strings.Contains(result, "OOMKilled") {
		t.Errorf("expected OOMKilled event in output")
	}
	if !strings.Contains(result, "Scaled") {
		t.Errorf("expected Scaled event in output")
	}
}

func TestRenderBar(t *testing.T) {
	tests := []struct {
		name     string
		pct      float64
		width    int
		wantFill int
	}{
		{"zero", 0, 10, 0},
		{"half", 50, 10, 5},
		{"full", 100, 10, 10},
		{"over", 150, 10, 10},
		{"negative", -10, 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderBar(tt.pct, tt.width)
			// Count filled blocks (█) - they appear after color tags
			filled := strings.Count(result, "█")
			empty := strings.Count(result, "░")
			if filled != tt.wantFill {
				t.Errorf("renderBar(%v, %d): got %d filled, want %d", tt.pct, tt.width, filled, tt.wantFill)
			}
			if filled+empty != tt.width {
				t.Errorf("renderBar(%v, %d): total width %d, want %d", tt.pct, tt.width, filled+empty, tt.width)
			}
		})
	}
}

func TestRenderBar_ColorThresholds(t *testing.T) {
	// Low usage - green
	low := renderBar(30, 10)
	if !strings.Contains(low, "[green]") {
		t.Errorf("expected green color for 30%% usage")
	}

	// Medium usage - yellow
	med := renderBar(70, 10)
	if !strings.Contains(med, "[yellow]") {
		t.Errorf("expected yellow color for 70%% usage")
	}

	// High usage - red
	high := renderBar(90, 10)
	if !strings.Contains(high, "[red]") {
		t.Errorf("expected red color for 90%% usage")
	}
}

func TestTruncateString(t *testing.T) {
	if got := truncatePulseString("hello", 10); got != "hello" {
		t.Errorf("truncatePulseString('hello', 10) = %q, want 'hello'", got)
	}
	if got := truncatePulseString("hello world this is long", 10); got != "hello w..." {
		t.Errorf("truncatePulseString('hello world this is long', 10) = %q, want 'hello w...'", got)
	}
}

func TestNewPulseView(t *testing.T) {
	app := &App{}
	pv := NewPulseView(app)

	if pv == nil {
		t.Fatal("NewPulseView returned nil")
	}
	if pv.app != app {
		t.Error("PulseView.app not set correctly")
	}
	title := pv.GetTitle()
	if !strings.Contains(title, "Pulse") {
		t.Errorf("expected 'Pulse' in title, got %q", title)
	}
}
