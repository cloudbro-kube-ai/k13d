package web

import (
	"testing"
)

func TestCalculateHealthScore(t *testing.T) {
	tests := []struct {
		name         string
		healthyNodes int
		totalNodes   int
		runningPods  int
		totalPods    int
		want         float64
	}{
		{"all healthy", 3, 3, 10, 10, 100.0},
		{"half healthy", 1, 2, 5, 10, 50.0},
		{"no nodes", 0, 0, 5, 10, 75.0}, // nodeScore=50 (default), podScore=25
		{"no pods", 2, 4, 0, 0, 75.0},   // nodeScore=25, podScore=50 (default)
		{"mixed health", 2, 4, 8, 10, 65.0},
		{"all unhealthy", 0, 2, 0, 10, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateHealthScore(tt.healthyNodes, tt.totalNodes, tt.runningPods, tt.totalPods)
			if got != tt.want {
				t.Errorf("calculateHealthScore(%d, %d, %d, %d) = %v, want %v",
					tt.healthyNodes, tt.totalNodes, tt.runningPods, tt.totalPods, got, tt.want)
			}
		})
	}
}

func TestComprehensiveReport_Struct(t *testing.T) {
	report := &ComprehensiveReport{
		GeneratedBy: "test-user",
		HealthScore: 95.5,
		ClusterInfo: ClusterInfo{
			ServerVersion: "v1.28.0",
			Platform:      "kubernetes",
			TotalNodes:    3,
			TotalPods:     10,
		},
	}

	if report.GeneratedBy != "test-user" {
		t.Errorf("expected GeneratedBy 'test-user', got %s", report.GeneratedBy)
	}

	if report.HealthScore != 95.5 {
		t.Errorf("expected HealthScore 95.5, got %f", report.HealthScore)
	}

	if report.ClusterInfo.TotalNodes != 3 {
		t.Errorf("expected TotalNodes 3, got %d", report.ClusterInfo.TotalNodes)
	}
}

func TestNewReportGenerator(t *testing.T) {
	// Test with nil server
	rg := NewReportGenerator(nil)
	if rg == nil {
		t.Fatal("expected ReportGenerator to be created even with nil server")
	}

	if rg.server != nil {
		t.Error("expected server to be nil")
	}
}

func TestFinOpsAnalysis_Struct(t *testing.T) {
	analysis := FinOpsAnalysis{
		TotalEstimatedMonthlyCost: 150.50,
		CostByNamespace: []NamespaceCost{
			{
				Namespace:      "default",
				PodCount:       5,
				CPURequests:    "1.5 cores",
				MemoryRequests: "2.5 GB",
				EstimatedCost:  75.25,
				CostPercentage: 50.0,
			},
			{
				Namespace:      "kube-system",
				PodCount:       3,
				CPURequests:    "1.0 cores",
				MemoryRequests: "1.5 GB",
				EstimatedCost:  50.00,
				CostPercentage: 33.2,
			},
		},
		ResourceEfficiency: ResourceEfficiencyInfo{
			TotalCPURequests:         "2.5 cores",
			TotalCPULimits:           "4.0 cores",
			TotalMemoryRequests:      "4.0 GB",
			TotalMemoryLimits:        "8.0 GB",
			CPURequestsVsCapacity:    25.0,
			MemoryRequestsVsCapacity: 40.0,
			PodsWithoutRequests:      2,
			PodsWithoutLimits:        3,
		},
		CostOptimizations: []CostOptimization{
			{
				Category:        "Resource Management",
				Description:     "2 pods without resource requests",
				Impact:          "May cause scheduling issues",
				EstimatedSaving: 10.0,
				Priority:        "high",
			},
		},
		UnderutilizedResources: []UnderutilizedResource{
			{
				Name:         "idle-pod",
				Namespace:    "default",
				ResourceType: "Pod",
				CPUUsage:     5.0,
				MemoryUsage:  10.0,
				Suggestion:   "Consider scaling down",
			},
		},
		OverprovisionedWorkloads: []OverprovisionedWorkload{
			{
				Name:              "over-deploy",
				Namespace:         "default",
				WorkloadType:      "Deployment",
				CurrentReplicas:   5,
				SuggestedReplicas: 2,
				Reason:            "Low utilization",
			},
		},
	}

	// Test total cost
	if analysis.TotalEstimatedMonthlyCost != 150.50 {
		t.Errorf("expected TotalEstimatedMonthlyCost 150.50, got %f", analysis.TotalEstimatedMonthlyCost)
	}

	// Test namespace costs
	if len(analysis.CostByNamespace) != 2 {
		t.Errorf("expected 2 namespace costs, got %d", len(analysis.CostByNamespace))
	}
	if analysis.CostByNamespace[0].Namespace != "default" {
		t.Errorf("expected first namespace 'default', got %s", analysis.CostByNamespace[0].Namespace)
	}

	// Test resource efficiency
	if analysis.ResourceEfficiency.CPURequestsVsCapacity != 25.0 {
		t.Errorf("expected CPURequestsVsCapacity 25.0, got %f", analysis.ResourceEfficiency.CPURequestsVsCapacity)
	}
	if analysis.ResourceEfficiency.PodsWithoutRequests != 2 {
		t.Errorf("expected PodsWithoutRequests 2, got %d", analysis.ResourceEfficiency.PodsWithoutRequests)
	}

	// Test cost optimizations
	if len(analysis.CostOptimizations) != 1 {
		t.Errorf("expected 1 cost optimization, got %d", len(analysis.CostOptimizations))
	}
	if analysis.CostOptimizations[0].Priority != "high" {
		t.Errorf("expected priority 'high', got %s", analysis.CostOptimizations[0].Priority)
	}

	// Test underutilized resources
	if len(analysis.UnderutilizedResources) != 1 {
		t.Errorf("expected 1 underutilized resource, got %d", len(analysis.UnderutilizedResources))
	}

	// Test overprovisioned workloads
	if len(analysis.OverprovisionedWorkloads) != 1 {
		t.Errorf("expected 1 overprovisioned workload, got %d", len(analysis.OverprovisionedWorkloads))
	}
	if analysis.OverprovisionedWorkloads[0].CurrentReplicas != 5 {
		t.Errorf("expected CurrentReplicas 5, got %d", analysis.OverprovisionedWorkloads[0].CurrentReplicas)
	}
}

func TestComprehensiveReport_WithFinOps(t *testing.T) {
	report := &ComprehensiveReport{
		GeneratedBy: "test-user",
		HealthScore: 85.0,
		ClusterInfo: ClusterInfo{
			TotalNodes: 5,
			TotalPods:  20,
		},
		FinOpsAnalysis: FinOpsAnalysis{
			TotalEstimatedMonthlyCost: 500.00,
			ResourceEfficiency: ResourceEfficiencyInfo{
				CPURequestsVsCapacity:    35.0,
				MemoryRequestsVsCapacity: 45.0,
			},
		},
	}

	// Verify FinOps analysis is included
	if report.FinOpsAnalysis.TotalEstimatedMonthlyCost != 500.00 {
		t.Errorf("expected TotalEstimatedMonthlyCost 500.00, got %f", report.FinOpsAnalysis.TotalEstimatedMonthlyCost)
	}

	if report.FinOpsAnalysis.ResourceEfficiency.CPURequestsVsCapacity != 35.0 {
		t.Errorf("expected CPURequestsVsCapacity 35.0, got %f", report.FinOpsAnalysis.ResourceEfficiency.CPURequestsVsCapacity)
	}
}

func TestCostOptimization_Priorities(t *testing.T) {
	tests := []struct {
		name     string
		priority string
		valid    bool
	}{
		{"high priority", "high", true},
		{"medium priority", "medium", true},
		{"low priority", "low", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := CostOptimization{
				Category:        "Test",
				Description:     "Test optimization",
				Priority:        tt.priority,
				EstimatedSaving: 10.0,
			}

			if opt.Priority != tt.priority {
				t.Errorf("expected priority %s, got %s", tt.priority, opt.Priority)
			}
		})
	}
}

func TestParseSections(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantNil  bool
		checkFn  func(*ReportSections) bool
		checkMsg string
	}{
		{
			name:    "empty string returns nil (all sections)",
			input:   "",
			wantNil: true,
		},
		{
			name:     "single section - nodes",
			input:    "nodes",
			checkFn:  func(s *ReportSections) bool { return s.Nodes && !s.Workloads && !s.FinOps },
			checkMsg: "only Nodes should be true",
		},
		{
			name:     "multiple sections",
			input:    "nodes,workloads,finops",
			checkFn:  func(s *ReportSections) bool { return s.Nodes && s.Workloads && s.FinOps && !s.SecurityBasic },
			checkMsg: "Nodes, Workloads, FinOps should be true; SecurityBasic should be false",
		},
		{
			name:     "security enables SecurityBasic",
			input:    "security",
			checkFn:  func(s *ReportSections) bool { return s.SecurityBasic && !s.SecurityFull },
			checkMsg: "SecurityBasic should be true, SecurityFull should be false",
		},
		{
			name:     "security_full enables both SecurityBasic and SecurityFull",
			input:    "security_full",
			checkFn:  func(s *ReportSections) bool { return s.SecurityBasic && s.SecurityFull },
			checkMsg: "both SecurityBasic and SecurityFull should be true",
		},
		{
			name:     "events and metrics",
			input:    "events,metrics",
			checkFn:  func(s *ReportSections) bool { return s.Events && s.Metrics && !s.Nodes },
			checkMsg: "Events and Metrics should be true",
		},
		{
			name:     "namespaces section",
			input:    "namespaces",
			checkFn:  func(s *ReportSections) bool { return s.Namespaces && !s.Nodes },
			checkMsg: "Namespaces should be true, Nodes should be false",
		},
		{
			name:     "all sections combined",
			input:    "nodes,namespaces,workloads,events,security_full,finops,metrics",
			checkFn:  func(s *ReportSections) bool { return s.Nodes && s.Namespaces && s.Workloads && s.Events && s.SecurityBasic && s.SecurityFull && s.FinOps && s.Metrics },
			checkMsg: "all sections should be true",
		},
		{
			name:  "unknown sections are ignored",
			input: "nodes,unknown,invalid",
			checkFn: func(s *ReportSections) bool {
				return s.Nodes && !s.Workloads && !s.FinOps && !s.SecurityBasic
			},
			checkMsg: "only Nodes should be true, unknown sections ignored",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSections(tt.input)
			if tt.wantNil {
				if result != nil {
					t.Errorf("ParseSections(%q) should return nil, got %+v", tt.input, result)
				}
				return
			}
			if result == nil {
				t.Fatalf("ParseSections(%q) returned nil, expected non-nil", tt.input)
			}
			if !tt.checkFn(result) {
				t.Errorf("ParseSections(%q): %s, got %+v", tt.input, tt.checkMsg, result)
			}
		})
	}
}

func TestAllSections(t *testing.T) {
	s := AllSections()
	if s == nil {
		t.Fatal("AllSections() returned nil")
	}
	if !s.Nodes || !s.Namespaces || !s.Workloads || !s.Events || !s.SecurityBasic || !s.FinOps || !s.Metrics {
		t.Errorf("AllSections() should have all standard sections enabled, got %+v", s)
	}
	// SecurityFull should NOT be enabled by default (it's slow with Trivy)
	if s.SecurityFull {
		t.Error("AllSections() should not enable SecurityFull by default")
	}
}

func TestReportSections_SecurityFullImpliesBasic(t *testing.T) {
	s := ParseSections("security_full")
	if s == nil {
		t.Fatal("ParseSections returned nil")
	}
	if !s.SecurityBasic {
		t.Error("security_full should also enable SecurityBasic")
	}
	if !s.SecurityFull {
		t.Error("security_full should enable SecurityFull")
	}
}

func TestNamespaceCost_CostPercentage(t *testing.T) {
	costs := []NamespaceCost{
		{Namespace: "ns1", EstimatedCost: 100.0, CostPercentage: 50.0},
		{Namespace: "ns2", EstimatedCost: 60.0, CostPercentage: 30.0},
		{Namespace: "ns3", EstimatedCost: 40.0, CostPercentage: 20.0},
	}

	var totalPercentage float64
	for _, c := range costs {
		totalPercentage += c.CostPercentage
	}

	if totalPercentage != 100.0 {
		t.Errorf("expected total percentage 100.0, got %f", totalPercentage)
	}
}
