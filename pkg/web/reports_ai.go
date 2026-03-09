package web

import (
	"context"
	"fmt"
	"strings"
)

func (rg *ReportGenerator) GenerateAIAnalysis(ctx context.Context, report *ComprehensiveReport) (string, error) {
	if rg.server.aiClient == nil || !rg.server.aiClient.IsReady() {
		return "", fmt.Errorf("AI client not available")
	}

	// Build cost optimization summary
	var costOptSummary strings.Builder
	for i, opt := range report.FinOpsAnalysis.CostOptimizations {
		if i >= 5 {
			break
		}
		costOptSummary.WriteString(fmt.Sprintf("- [%s] %s (Est. saving: $%.2f/mo)\n", opt.Priority, opt.Description, opt.EstimatedSaving))
	}

	// Build summary for AI with FinOps focus
	prompt := fmt.Sprintf(`You are a Kubernetes and FinOps expert. Analyze this cluster state and provide a comprehensive professional report (max 600 words) with special focus on cost optimization.

Cluster Summary:
- Nodes: %d total, %d ready, %d not ready
- Pods: %d total, %d running, %d pending, %d failed
- Deployments: %d total, %d healthy
- Services: %d
- Health Score: %.1f%%

FinOps / Cost Analysis:
- Estimated Monthly Cost: $%.2f
- CPU Utilization vs Capacity: %.1f%%
- Memory Utilization vs Capacity: %.1f%%
- Pods without Resource Requests: %d
- Pods without Resource Limits: %d

Top Cost Optimization Opportunities:
%s

Security Concerns:
- Privileged Pods: %d
- Host Network Pods: %d
- Root Containers: %d

Warning Events: %d

Top Images Used:
%s

Please provide:
1. Overall cluster health assessment
2. **FinOps Cost Analysis** (prioritize this section):
   - Current spending efficiency
   - Top cost drivers
   - Immediate cost reduction opportunities
   - Long-term optimization recommendations
3. Resource optimization recommendations
4. Security observations
5. Action items with priority levels

Be concise, actionable, and focus on ROI for each recommendation.`,
		report.NodeSummary.Total, report.NodeSummary.Ready, report.NodeSummary.NotReady,
		report.Workloads.TotalPods, report.Workloads.RunningPods, report.Workloads.PendingPods, report.Workloads.FailedPods,
		report.Workloads.TotalDeployments, report.Workloads.HealthyDeploys,
		report.Workloads.TotalServices,
		report.HealthScore,
		report.FinOpsAnalysis.TotalEstimatedMonthlyCost,
		report.FinOpsAnalysis.ResourceEfficiency.CPURequestsVsCapacity,
		report.FinOpsAnalysis.ResourceEfficiency.MemoryRequestsVsCapacity,
		report.FinOpsAnalysis.ResourceEfficiency.PodsWithoutRequests,
		report.FinOpsAnalysis.ResourceEfficiency.PodsWithoutLimits,
		costOptSummary.String(),
		report.SecurityInfo.PrivilegedPods, report.SecurityInfo.HostNetworkPods, report.SecurityInfo.RootContainers,
		len(report.Events),
		formatTopImages(report.Images, 5),
	)

	analysis, err := rg.server.aiClient.AskNonStreaming(ctx, prompt)
	if err != nil {
		return "", err
	}

	return analysis, nil
}

func formatTopImages(images []ImageInfo, limit int) string {
	var sb strings.Builder
	for i, img := range images {
		if i >= limit {
			break
		}
		sb.WriteString(fmt.Sprintf("- %s (used by %d pods)\n", img.Image, img.PodCount))
	}
	return sb.String()
}

func calculateHealthScore(healthyNodes, totalNodes, runningPods, totalPods int) float64 {
	if totalNodes == 0 && totalPods == 0 {
		return 100.0
	}

	nodeScore := 50.0
	if totalNodes > 0 {
		nodeScore = float64(healthyNodes) / float64(totalNodes) * 50
	}

	podScore := 50.0
	if totalPods > 0 {
		podScore = float64(runningPods) / float64(totalPods) * 50
	}

	return nodeScore + podScore
}

// ExportToCSV generates CSV format report
