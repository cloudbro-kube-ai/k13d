package web

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"sort"
	"strings"
)

func (rg *ReportGenerator) generateFinOpsAnalysis(ctx context.Context, namespaces []corev1.Namespace, report *ComprehensiveReport) FinOpsAnalysis {
	analysis := FinOpsAnalysis{
		CostByNamespace:          []NamespaceCost{},
		CostOptimizations:        []CostOptimization{},
		UnderutilizedResources:   []UnderutilizedResource{},
		OverprovisionedWorkloads: []OverprovisionedWorkload{},
	}

	// Reference costs (approximate AWS EKS pricing)
	// vCPU: ~$0.04/hour, Memory: ~$0.004/GB/hour
	const cpuHourlyCost = 0.04     // per vCPU
	const memoryHourlyCost = 0.004 // per GB

	var totalCPURequests, totalCPULimits int64 // millicores
	var totalMemRequests, totalMemLimits int64 // bytes
	var totalNodeCPUCapacity int64             // millicores
	var totalNodeMemCapacity int64             // bytes
	var podsWithoutRequests, podsWithoutLimits int

	// Calculate node capacity
	nodes, _ := rg.server.k8sClient.ListNodes(ctx)
	for _, node := range nodes {
		cpu := node.Status.Capacity.Cpu()
		mem := node.Status.Capacity.Memory()
		totalNodeCPUCapacity += cpu.MilliValue()
		totalNodeMemCapacity += mem.Value()
	}

	// Analyze each namespace
	nsCosts := make(map[string]*NamespaceCost)

	for _, ns := range namespaces {
		pods, _ := rg.server.k8sClient.ListPods(ctx, ns.Name)

		nsCost := &NamespaceCost{
			Namespace: ns.Name,
			PodCount:  len(pods),
		}

		var nsCPU, nsMem int64

		for _, pod := range pods {
			podHasRequests := false
			podHasLimits := false

			for _, container := range pod.Spec.Containers {
				// Requests
				if cpuReq := container.Resources.Requests.Cpu(); cpuReq != nil {
					nsCPU += cpuReq.MilliValue()
					totalCPURequests += cpuReq.MilliValue()
					podHasRequests = true
				}
				if memReq := container.Resources.Requests.Memory(); memReq != nil {
					nsMem += memReq.Value()
					totalMemRequests += memReq.Value()
					podHasRequests = true
				}

				// Limits
				if cpuLim := container.Resources.Limits.Cpu(); cpuLim != nil {
					totalCPULimits += cpuLim.MilliValue()
					podHasLimits = true
				}
				if memLim := container.Resources.Limits.Memory(); memLim != nil {
					totalMemLimits += memLim.Value()
					podHasLimits = true
				}
			}

			if !podHasRequests {
				podsWithoutRequests++
			}
			if !podHasLimits {
				podsWithoutLimits++
			}
		}

		// Calculate namespace cost (monthly estimate)
		cpuCores := float64(nsCPU) / 1000.0
		memGB := float64(nsMem) / (1024 * 1024 * 1024)
		monthlyHours := 730.0 // average hours per month

		nsCost.CPURequests = fmt.Sprintf("%.2f cores", cpuCores)
		nsCost.MemoryRequests = fmt.Sprintf("%.2f GB", memGB)
		nsCost.EstimatedCost = (cpuCores*cpuHourlyCost + memGB*memoryHourlyCost) * monthlyHours

		nsCosts[ns.Name] = nsCost
	}

	// Calculate total and percentages
	var totalCost float64
	for _, nsCost := range nsCosts {
		totalCost += nsCost.EstimatedCost
	}

	for _, nsCost := range nsCosts {
		if totalCost > 0 {
			nsCost.CostPercentage = (nsCost.EstimatedCost / totalCost) * 100
		}
		analysis.CostByNamespace = append(analysis.CostByNamespace, *nsCost)
	}

	// Sort by cost descending
	sort.Slice(analysis.CostByNamespace, func(i, j int) bool {
		return analysis.CostByNamespace[i].EstimatedCost > analysis.CostByNamespace[j].EstimatedCost
	})

	analysis.TotalEstimatedMonthlyCost = totalCost

	// Resource efficiency
	analysis.ResourceEfficiency = ResourceEfficiencyInfo{
		TotalCPURequests:    fmt.Sprintf("%.2f cores", float64(totalCPURequests)/1000.0),
		TotalCPULimits:      fmt.Sprintf("%.2f cores", float64(totalCPULimits)/1000.0),
		TotalMemoryRequests: fmt.Sprintf("%.2f GB", float64(totalMemRequests)/(1024*1024*1024)),
		TotalMemoryLimits:   fmt.Sprintf("%.2f GB", float64(totalMemLimits)/(1024*1024*1024)),
		PodsWithoutRequests: podsWithoutRequests,
		PodsWithoutLimits:   podsWithoutLimits,
	}

	if totalNodeCPUCapacity > 0 {
		analysis.ResourceEfficiency.CPURequestsVsCapacity = float64(totalCPURequests) / float64(totalNodeCPUCapacity) * 100
	}
	if totalNodeMemCapacity > 0 {
		analysis.ResourceEfficiency.MemoryRequestsVsCapacity = float64(totalMemRequests) / float64(totalNodeMemCapacity) * 100
	}

	// Generate cost optimization recommendations
	analysis.CostOptimizations = rg.generateCostOptimizations(report, &analysis)

	// Analyze underutilized deployments
	for _, dep := range report.Deployments {
		// Check for deployments with many unavailable replicas
		parts := strings.Split(dep.Ready, "/")
		if len(parts) == 2 {
			ready := 0
			total := 0
			_, _ = fmt.Sscanf(parts[0], "%d", &ready)
			_, _ = fmt.Sscanf(parts[1], "%d", &total)

			if total > 1 && ready < total {
				analysis.OverprovisionedWorkloads = append(analysis.OverprovisionedWorkloads, OverprovisionedWorkload{
					Name:              dep.Name,
					Namespace:         dep.Namespace,
					WorkloadType:      "Deployment",
					CurrentReplicas:   total,
					SuggestedReplicas: ready,
					Reason:            fmt.Sprintf("Only %d/%d replicas are ready - consider reducing replicas or investigating issues", ready, total),
				})
			}
		}
	}

	return analysis
}

// generateCostOptimizations creates cost saving recommendations
func (rg *ReportGenerator) generateCostOptimizations(report *ComprehensiveReport, analysis *FinOpsAnalysis) []CostOptimization {
	var optimizations []CostOptimization

	// Check for pods without resource requests
	if analysis.ResourceEfficiency.PodsWithoutRequests > 0 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Resource Management",
			Description:     fmt.Sprintf("%d pods are running without resource requests defined", analysis.ResourceEfficiency.PodsWithoutRequests),
			Impact:          "Without resource requests, pods may be scheduled inefficiently leading to resource contention or waste",
			EstimatedSaving: float64(analysis.ResourceEfficiency.PodsWithoutRequests) * 5.0, // $5 per pod monthly estimate
			Priority:        "high",
		})
	}

	// Check for pods without resource limits
	if analysis.ResourceEfficiency.PodsWithoutLimits > 0 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Resource Management",
			Description:     fmt.Sprintf("%d pods are running without resource limits defined", analysis.ResourceEfficiency.PodsWithoutLimits),
			Impact:          "Without limits, pods can consume unbounded resources affecting cluster stability",
			EstimatedSaving: float64(analysis.ResourceEfficiency.PodsWithoutLimits) * 3.0,
			Priority:        "medium",
		})
	}

	// Check for low cluster utilization
	if analysis.ResourceEfficiency.CPURequestsVsCapacity < 30 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Cluster Sizing",
			Description:     fmt.Sprintf("CPU utilization is only %.1f%% of cluster capacity", analysis.ResourceEfficiency.CPURequestsVsCapacity),
			Impact:          "Consider reducing node count or using smaller instance types",
			EstimatedSaving: analysis.TotalEstimatedMonthlyCost * 0.3, // 30% potential savings
			Priority:        "high",
		})
	}

	if analysis.ResourceEfficiency.MemoryRequestsVsCapacity < 30 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Cluster Sizing",
			Description:     fmt.Sprintf("Memory utilization is only %.1f%% of cluster capacity", analysis.ResourceEfficiency.MemoryRequestsVsCapacity),
			Impact:          "Consider using memory-optimized instances or reducing node count",
			EstimatedSaving: analysis.TotalEstimatedMonthlyCost * 0.2,
			Priority:        "medium",
		})
	}

	// Check for failed pods wasting resources
	if report.Workloads.FailedPods > 0 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Workload Health",
			Description:     fmt.Sprintf("%d pods are in failed state", report.Workloads.FailedPods),
			Impact:          "Failed pods may still consume resources and indicate configuration issues",
			EstimatedSaving: float64(report.Workloads.FailedPods) * 10.0,
			Priority:        "high",
		})
	}

	// Check for pending pods
	if report.Workloads.PendingPods > 0 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Scheduling",
			Description:     fmt.Sprintf("%d pods are pending and cannot be scheduled", report.Workloads.PendingPods),
			Impact:          "Pending pods indicate resource constraints or scheduling issues",
			EstimatedSaving: 0,
			Priority:        "high",
		})
	}

	// Check for many restarts indicating instability
	totalRestarts := 0
	for _, pod := range report.Pods {
		totalRestarts += pod.Restarts
	}
	if totalRestarts > 10 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Stability",
			Description:     fmt.Sprintf("Total of %d container restarts detected across pods", totalRestarts),
			Impact:          "Frequent restarts waste compute resources and may indicate memory/OOM issues",
			EstimatedSaving: float64(totalRestarts) * 0.5,
			Priority:        "medium",
		})
	}

	// LoadBalancer service costs
	lbCount := 0
	for _, svc := range report.Services {
		if svc.Type == "LoadBalancer" {
			lbCount++
		}
	}
	if lbCount > 3 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Networking",
			Description:     fmt.Sprintf("%d LoadBalancer services detected", lbCount),
			Impact:          "Each LoadBalancer incurs cloud provider costs (~$18/month each). Consider using Ingress controller",
			EstimatedSaving: float64(lbCount-1) * 18.0, // Keep 1, consolidate others
			Priority:        "medium",
		})
	}

	return optimizations
}

// GenerateAIAnalysis uses LLM to analyze the cluster state with FinOps focus
