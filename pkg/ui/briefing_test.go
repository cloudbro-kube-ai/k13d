package ui

import (
	"testing"
)

func TestCalculateHealthScore(t *testing.T) {
	tests := []struct {
		name     string
		data     *BriefingData
		expected int
		minScore int
		maxScore int
	}{
		{
			name: "perfect health",
			data: &BriefingData{
				TotalPods:        10,
				RunningPods:      10,
				TotalNodes:       3,
				ReadyNodes:       3,
				TotalDeployments: 5,
				ReadyDeployments: 5,
				CPUPercent:       50,
				MemoryPercent:    60,
			},
			minScore: 95,
			maxScore: 100,
		},
		{
			name: "all pods failed",
			data: &BriefingData{
				TotalPods:        10,
				RunningPods:      0,
				TotalNodes:       3,
				ReadyNodes:       3,
				TotalDeployments: 5,
				ReadyDeployments: 5,
				CPUPercent:       50,
				MemoryPercent:    60,
			},
			minScore: 50,
			maxScore: 70,
		},
		{
			name: "all nodes down",
			data: &BriefingData{
				TotalPods:        10,
				RunningPods:      10,
				TotalNodes:       3,
				ReadyNodes:       0,
				TotalDeployments: 5,
				ReadyDeployments: 5,
				CPUPercent:       50,
				MemoryPercent:    60,
			},
			minScore: 60,
			maxScore: 80,
		},
		{
			name: "high resource usage",
			data: &BriefingData{
				TotalPods:        10,
				RunningPods:      10,
				TotalNodes:       3,
				ReadyNodes:       3,
				TotalDeployments: 5,
				ReadyDeployments: 5,
				CPUPercent:       95,
				MemoryPercent:    92,
			},
			minScore: 85,
			maxScore: 95,
		},
		{
			name: "critical - everything failing",
			data: &BriefingData{
				TotalPods:        10,
				RunningPods:      2,
				TotalNodes:       3,
				ReadyNodes:       1,
				TotalDeployments: 5,
				ReadyDeployments: 1,
				CPUPercent:       95,
				MemoryPercent:    95,
			},
			minScore: 0,
			maxScore: 40,
		},
		{
			name: "empty cluster",
			data: &BriefingData{
				TotalPods:        0,
				RunningPods:      0,
				TotalNodes:       0,
				ReadyNodes:       0,
				TotalDeployments: 0,
				ReadyDeployments: 0,
			},
			minScore: 95,
			maxScore: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateHealthScore(tt.data)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("calculateHealthScore() = %d, want between %d and %d", score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestHealthStatusFromScore(t *testing.T) {
	tests := []struct {
		score    int
		expected string
	}{
		{100, "healthy"},
		{95, "healthy"},
		{90, "healthy"},
		{89, "warning"},
		{70, "warning"},
		{69, "critical"},
		{50, "critical"},
		{0, "critical"},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.score)), func(t *testing.T) {
			result := healthStatusFromScore(tt.score)
			if result != tt.expected {
				t.Errorf("healthStatusFromScore(%d) = %q, want %q", tt.score, result, tt.expected)
			}
		})
	}
}

func TestGetHealthColor(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"healthy", "[green]"},
		{"warning", "[yellow]"},
		{"critical", "[red]"},
		{"unknown", "[white]"},
		{"", "[white]"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := getHealthColor(tt.status)
			if result != tt.expected {
				t.Errorf("getHealthColor(%q) = %q, want %q", tt.status, result, tt.expected)
			}
		})
	}
}

func TestBriefingData(t *testing.T) {
	// Test that BriefingData can be constructed with all fields
	data := &BriefingData{
		HealthScore:      85,
		HealthStatus:     "warning",
		TotalPods:        10,
		RunningPods:      8,
		PendingPods:      1,
		FailedPods:       1,
		TotalNodes:       3,
		ReadyNodes:       3,
		TotalDeployments: 5,
		ReadyDeployments: 4,
		CPUPercent:       75.5,
		MemoryPercent:    80.2,
		Namespace:        "default",
		Alerts:           []string{"1 pod failing", "High memory"},
		ContextName:      "test-context",
		ClusterName:      "test-cluster",
	}

	if data.HealthScore != 85 {
		t.Errorf("HealthScore = %d, want 85", data.HealthScore)
	}

	if len(data.Alerts) != 2 {
		t.Errorf("Alerts length = %d, want 2", len(data.Alerts))
	}
}
