package web

import (
	"context"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
)

func (rg *ReportGenerator) generateFinOpsAnalysis(ctx context.Context, namespaces []corev1.Namespace, report *ComprehensiveReport) FinOpsAnalysis {
	analysis := FinOpsAnalysis{
		EstimationModel:          "heuristic compute estimate from running pod requests with live metrics preferred",
		CostByNamespace:          []NamespaceCost{},
		CostOptimizations:        []CostOptimization{},
		UnderutilizedResources:   []UnderutilizedResource{},
		OverprovisionedWorkloads: []OverprovisionedWorkload{},
	}

	// Approximate compute-only reference pricing.
	// This is intentionally conservative and should be described as a heuristic,
	// not a cloud billing replacement.
	const cpuHourlyCost = 0.04
	const memoryHourlyCost = 0.004
	const monthlyHours = 730.0
	const mib = int64(1024 * 1024)
	const gib = float64(1024 * 1024 * 1024)

	type podUsage struct {
		cpuMilli  int64
		memBytes  int64
		available bool
	}

	podUsageByKey := make(map[string]podUsage)
	metricsSource := "live_metrics"

	for _, ns := range namespaces {
		metrics, err := rg.server.k8sClient.GetPodMetrics(ctx, ns.Name)
		if err != nil {
			metricsSource = "request_fallback"
			podUsageByKey = make(map[string]podUsage)
			break
		}
		for podName, values := range metrics {
			if len(values) < 2 {
				continue
			}
			podUsageByKey[ns.Name+"/"+podName] = podUsage{
				cpuMilli:  values[0],
				memBytes:  values[1] * mib,
				available: true,
			}
		}
	}

	if metricsSource == "request_fallback" {
		for _, ns := range namespaces {
			metrics, err := rg.server.k8sClient.GetPodMetricsFromRequests(ctx, ns.Name)
			if err != nil {
				metricsSource = "unavailable"
				podUsageByKey = make(map[string]podUsage)
				break
			}
			for podName, values := range metrics {
				if len(values) < 2 {
					continue
				}
				podUsageByKey[ns.Name+"/"+podName] = podUsage{
					cpuMilli:  values[0],
					memBytes:  values[1] * mib,
					available: true,
				}
			}
		}
	}

	switch metricsSource {
	case "live_metrics":
		analysis.EstimationNotes = append(analysis.EstimationNotes,
			"Live pod metrics were available and used for usage and efficiency fields.",
			"Estimated monthly cost uses running pod requests, but bumps to live usage when usage exceeds requests.",
			"This report estimates compute only and should not be treated as an exact cloud invoice.",
		)
	case "request_fallback":
		analysis.EstimationNotes = append(analysis.EstimationNotes,
			"metrics-server was unavailable, so usage fields were estimated from pod resource requests.",
			"Estimated monthly cost is conservative but still heuristic and excludes provider-specific charges such as control-plane, storage, and egress.",
		)
	default:
		analysis.EstimationNotes = append(analysis.EstimationNotes,
			"Neither live metrics nor request-derived usage were available for all namespaces.",
			"Usage-based efficiency percentages should be treated as unavailable in this report.",
		)
	}

	var totalCPURequests, totalCPULimits int64
	var totalMemRequests, totalMemLimits int64
	var totalCPUUsage, totalMemUsage int64
	var totalNodeCPUAllocatable, totalNodeMemAllocatable int64
	var podsWithoutRequests, podsWithoutLimits int

	nodes, _ := rg.server.k8sClient.ListNodes(ctx)
	for _, node := range nodes {
		totalNodeCPUAllocatable += node.Status.Allocatable.Cpu().MilliValue()
		totalNodeMemAllocatable += node.Status.Allocatable.Memory().Value()
	}

	nsCosts := make(map[string]*NamespaceCost)

	for _, ns := range namespaces {
		pods, _ := rg.server.k8sClient.ListPods(ctx, ns.Name)

		nsCost := &NamespaceCost{
			Namespace: ns.Name,
			PodCount:  len(pods),
		}

		var nsCPURequests, nsMemRequests int64
		var nsCPUUsage, nsMemUsage int64
		var nsBillableCPU, nsBillableMem int64

		for _, pod := range pods {
			if pod.Status.Phase != corev1.PodRunning {
				continue
			}
			nsCost.RunningPodCount++

			key := pod.Namespace + "/" + pod.Name
			usage := podUsageByKey[key]

			podHasRequests := false
			podHasLimits := false

			var podCPURequests, podMemRequests int64
			for _, container := range pod.Spec.Containers {
				if cpuReq, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
					podCPURequests += cpuReq.MilliValue()
					totalCPURequests += cpuReq.MilliValue()
					nsCPURequests += cpuReq.MilliValue()
					podHasRequests = true
				}
				if memReq, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
					podMemRequests += memReq.Value()
					totalMemRequests += memReq.Value()
					nsMemRequests += memReq.Value()
					podHasRequests = true
				}

				if cpuLim, ok := container.Resources.Limits[corev1.ResourceCPU]; ok {
					totalCPULimits += cpuLim.MilliValue()
					podHasLimits = true
				}
				if memLim, ok := container.Resources.Limits[corev1.ResourceMemory]; ok {
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

			if usage.available {
				totalCPUUsage += usage.cpuMilli
				totalMemUsage += usage.memBytes
				nsCPUUsage += usage.cpuMilli
				nsMemUsage += usage.memBytes
			}

			billableCPU := podCPURequests
			billableMem := podMemRequests
			if metricsSource == "live_metrics" && usage.available {
				if usage.cpuMilli > billableCPU {
					billableCPU = usage.cpuMilli
				}
				if usage.memBytes > billableMem {
					billableMem = usage.memBytes
				}
			}
			nsBillableCPU += billableCPU
			nsBillableMem += billableMem

			if metricsSource == "live_metrics" && usage.available && podCPURequests > 0 && podMemRequests > 0 {
				cpuPct := percentInt64(usage.cpuMilli, podCPURequests)
				memPct := percentInt64(usage.memBytes, podMemRequests)
				if cpuPct < 25 && memPct < 25 {
					analysis.UnderutilizedResources = append(analysis.UnderutilizedResources, UnderutilizedResource{
						Name:         pod.Name,
						Namespace:    pod.Namespace,
						ResourceType: "Pod",
						CPUUsage:     cpuPct,
						MemoryUsage:  memPct,
						Suggestion:   "Review requests/limits or consolidate this workload if low usage persists.",
					})
				}
			}
		}

		nsCost.CPURequests = formatCoresFromMilli(nsCPURequests)
		nsCost.MemoryRequests = formatGBFromBytes(nsMemRequests)
		nsCost.CPUUsage = formatCoresFromMilli(nsCPUUsage)
		nsCost.MemoryUsage = formatGBFromBytes(nsMemUsage)
		nsCost.EstimatedCost = ((float64(nsBillableCPU) / 1000.0) * cpuHourlyCost) +
			((float64(nsBillableMem) / gib) * memoryHourlyCost)
		nsCost.EstimatedCost *= monthlyHours

		nsCosts[ns.Name] = nsCost
	}

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

	sort.Slice(analysis.CostByNamespace, func(i, j int) bool {
		return analysis.CostByNamespace[i].EstimatedCost > analysis.CostByNamespace[j].EstimatedCost
	})
	sort.Slice(analysis.UnderutilizedResources, func(i, j int) bool {
		return analysis.UnderutilizedResources[i].CPUUsage < analysis.UnderutilizedResources[j].CPUUsage
	})
	if len(analysis.UnderutilizedResources) > 20 {
		analysis.UnderutilizedResources = analysis.UnderutilizedResources[:20]
	}

	analysis.TotalEstimatedMonthlyCost = totalCost
	analysis.ResourceEfficiency = ResourceEfficiencyInfo{
		TotalCPURequests:         formatCoresFromMilli(totalCPURequests),
		TotalCPULimits:           formatCoresFromMilli(totalCPULimits),
		TotalMemoryRequests:      formatGBFromBytes(totalMemRequests),
		TotalMemoryLimits:        formatGBFromBytes(totalMemLimits),
		TotalCPUUsage:            formatCoresFromMilli(totalCPUUsage),
		TotalMemoryUsage:         formatGBFromBytes(totalMemUsage),
		CPURequestsVsCapacity:    percentInt64(totalCPURequests, totalNodeCPUAllocatable),
		MemoryRequestsVsCapacity: percentInt64(totalMemRequests, totalNodeMemAllocatable),
		PodsWithoutRequests:      podsWithoutRequests,
		PodsWithoutLimits:        podsWithoutLimits,
		MetricsSource:            metricsSource,
	}

	if metricsSource == "live_metrics" {
		analysis.ResourceEfficiency.CPUUsageVsRequests = percentInt64(totalCPUUsage, totalCPURequests)
		analysis.ResourceEfficiency.MemoryUsageVsRequests = percentInt64(totalMemUsage, totalMemRequests)
		analysis.ResourceEfficiency.CPUUsageVsCapacity = percentInt64(totalCPUUsage, totalNodeCPUAllocatable)
		analysis.ResourceEfficiency.MemoryUsageVsCapacity = percentInt64(totalMemUsage, totalNodeMemAllocatable)
	}

	analysis.CostOptimizations = rg.generateCostOptimizations(report, &analysis)
	return analysis
}

// generateCostOptimizations creates cost saving recommendations.
// The recommendations are intentionally conservative because the report
// estimate is heuristic and not a full provider billing model.
func (rg *ReportGenerator) generateCostOptimizations(report *ComprehensiveReport, analysis *FinOpsAnalysis) []CostOptimization {
	var optimizations []CostOptimization

	if analysis.ResourceEfficiency.PodsWithoutRequests > 0 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Resource Governance",
			Description:     fmt.Sprintf("%d running pods do not declare resource requests", analysis.ResourceEfficiency.PodsWithoutRequests),
			Impact:          "This weakens scheduling accuracy and makes cost estimates less reliable.",
			EstimatedSaving: 0,
			Priority:        "high",
		})
	}

	if analysis.ResourceEfficiency.PodsWithoutLimits > 0 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Resource Governance",
			Description:     fmt.Sprintf("%d running pods do not declare resource limits", analysis.ResourceEfficiency.PodsWithoutLimits),
			Impact:          "Missing limits increase noisy-neighbor risk and can hide overconsumption.",
			EstimatedSaving: 0,
			Priority:        "medium",
		})
	}

	if analysis.ResourceEfficiency.MetricsSource == "live_metrics" && len(analysis.UnderutilizedResources) > 0 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Rightsizing",
			Description:     fmt.Sprintf("%d pods are using less than 25%% of both requested CPU and memory", len(analysis.UnderutilizedResources)),
			Impact:          "These pods are good candidates for request/limit review before changing replica counts.",
			EstimatedSaving: 0,
			Priority:        "medium",
		})
	}

	if analysis.ResourceEfficiency.CPURequestsVsCapacity < 30 && analysis.ResourceEfficiency.MemoryRequestsVsCapacity < 30 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Cluster Sizing",
			Description:     fmt.Sprintf("Requested CPU is %.1f%% and requested memory is %.1f%% of allocatable node capacity", analysis.ResourceEfficiency.CPURequestsVsCapacity, analysis.ResourceEfficiency.MemoryRequestsVsCapacity),
			Impact:          "Review node pool sizing or autoscaler minimums. Savings depend on your actual cloud SKUs and reserved capacity.",
			EstimatedSaving: 0,
			Priority:        "medium",
		})
	}

	if report.Workloads.FailedPods > 0 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Workload Health",
			Description:     fmt.Sprintf("%d pods are in Failed state", report.Workloads.FailedPods),
			Impact:          "Treat this as an operations issue first; direct savings depend on restart policy and workload ownership.",
			EstimatedSaving: 0,
			Priority:        "high",
		})
	}

	if report.Workloads.PendingPods > 0 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Scheduling",
			Description:     fmt.Sprintf("%d pods are Pending and cannot currently be scheduled", report.Workloads.PendingPods),
			Impact:          "Pending pods usually indicate placement or capacity issues rather than immediate savings.",
			EstimatedSaving: 0,
			Priority:        "high",
		})
	}

	lbCount := 0
	for _, svc := range report.Services {
		if svc.Type == "LoadBalancer" {
			lbCount++
		}
	}
	if lbCount > 1 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Networking",
			Description:     fmt.Sprintf("%d LoadBalancer services detected", lbCount),
			Impact:          "Each LoadBalancer often adds a direct provider charge. Consider consolidating behind Ingress where appropriate.",
			EstimatedSaving: float64(lbCount-1) * 18.0,
			Priority:        "medium",
		})
	}

	return optimizations
}

func formatCoresFromMilli(milli int64) string {
	return fmt.Sprintf("%.2f cores", float64(milli)/1000.0)
}

func formatGBFromBytes(bytes int64) string {
	return fmt.Sprintf("%.2f GB", float64(bytes)/(1024*1024*1024))
}

func percentInt64(value, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(value) / float64(total) * 100
}
