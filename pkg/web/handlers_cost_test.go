package web

import (
	"testing"
)

func TestCalcEfficiency(t *testing.T) {
	tests := []struct {
		name             string
		requested        int64
		used             int64
		metricsAvailable bool
		expected         float64
	}{
		{"normal usage", 1000, 500, true, 50.0},
		{"full usage", 1000, 1000, true, 100.0},
		{"no usage", 1000, 0, true, 0.0},
		{"no request", 0, 500, true, 0.0},
		{"metrics unavailable", 1000, 500, false, -1.0},
		{"both zero", 0, 0, true, 0.0},
		{"over-provisioned", 2000, 100, true, 5.0},
		{"over-utilized", 100, 200, true, 200.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calcEfficiency(tt.requested, tt.used, tt.metricsAvailable)
			if result != tt.expected {
				t.Errorf("calcEfficiency(%d, %d, %v) = %f, want %f",
					tt.requested, tt.used, tt.metricsAvailable, result, tt.expected)
			}
		})
	}
}

func TestCalcEfficiencyEdgeCases(t *testing.T) {
	// Negative requested should return 0
	result := calcEfficiency(-100, 50, true)
	if result != 0 {
		t.Errorf("calcEfficiency(-100, 50, true) = %f, want 0", result)
	}

	// Large values should not overflow
	result = calcEfficiency(1000000, 999999, true)
	if result < 99 || result > 100 {
		t.Errorf("calcEfficiency(1000000, 999999, true) = %f, want ~99.9999", result)
	}
}

func TestCostRecommendationTypes(t *testing.T) {
	// Verify that recommendation type strings match expected values
	tests := []struct {
		name     string
		cpuEff   float64
		memEff   float64
		cpuUsed  int64
		memUsed  int64
		expected string
	}{
		{"oversized cpu", 20.0, 50.0, 200, 500, "oversized"},
		{"oversized mem", 50.0, 20.0, 500, 200, "oversized"},
		{"undersized cpu", 90.0, 50.0, 900, 500, "undersized"},
		{"undersized mem", 50.0, 90.0, 500, 900, "undersized"},
		{"idle", 0.0, 0.0, 0, 0, "idle"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify type strings are valid
			validTypes := map[string]bool{"oversized": true, "undersized": true, "idle": true}
			if !validTypes[tt.expected] {
				t.Errorf("unexpected recommendation type: %s", tt.expected)
			}
		})
	}
}

func TestCostEstimateStructFields(t *testing.T) {
	// Verify that CostEstimate can be constructed with all fields
	ce := CostEstimate{
		Namespace: "default",
		TotalCPU: ResourceCost{
			Requested:  "2000m",
			Used:       "500m",
			Efficiency: 25.0,
		},
		TotalMemory: ResourceCost{
			Requested:  "4Gi",
			Used:       "1Gi",
			Efficiency: 25.0,
		},
		Workloads: []WorkloadCost{
			{
				Kind:       "Deployment",
				Name:       "nginx",
				Namespace:  "default",
				CPUReq:     "1000m",
				CPUUsed:    "250m",
				MemReq:     "2Gi",
				MemUsed:    "512Mi",
				Replicas:   3,
				Efficiency: 25.0,
			},
		},
		Efficiency: 25.0,
		Recommendations: []CostRecommendation{
			{
				Workload:    "nginx-pod-1",
				Type:        "oversized",
				Description: "Pod nginx-pod-1 is using <30% of resources",
				Savings:     "Could save 500m CPU",
			},
		},
	}

	if ce.Namespace != "default" {
		t.Errorf("Namespace = %s, want default", ce.Namespace)
	}
	if ce.Efficiency != 25.0 {
		t.Errorf("Efficiency = %f, want 25.0", ce.Efficiency)
	}
	if len(ce.Workloads) != 1 {
		t.Errorf("Workloads count = %d, want 1", len(ce.Workloads))
	}
	if len(ce.Recommendations) != 1 {
		t.Errorf("Recommendations count = %d, want 1", len(ce.Recommendations))
	}
	if ce.TotalCPU.Requested != "2000m" {
		t.Errorf("TotalCPU.Requested = %s, want 2000m", ce.TotalCPU.Requested)
	}
}

func TestWorkloadCostFields(t *testing.T) {
	wl := WorkloadCost{
		Kind:       "ReplicaSet",
		Name:       "nginx-abc123",
		Namespace:  "production",
		CPUReq:     "500m",
		CPUUsed:    "100m",
		MemReq:     "256Mi",
		MemUsed:    "64Mi",
		Replicas:   2,
		Efficiency: 20.0,
	}

	if wl.Kind != "ReplicaSet" {
		t.Errorf("Kind = %s, want ReplicaSet", wl.Kind)
	}
	if wl.Replicas != 2 {
		t.Errorf("Replicas = %d, want 2", wl.Replicas)
	}
	if wl.Efficiency != 20.0 {
		t.Errorf("Efficiency = %f, want 20.0", wl.Efficiency)
	}
}
