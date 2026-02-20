package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
)

// CostEstimate represents resource cost data for a namespace or cluster.
type CostEstimate struct {
	Namespace       string               `json:"namespace"`
	TotalCPU        ResourceCost         `json:"totalCPU"`
	TotalMemory     ResourceCost         `json:"totalMemory"`
	Workloads       []WorkloadCost       `json:"workloads"`
	Efficiency      float64              `json:"efficiency"`
	Recommendations []CostRecommendation `json:"recommendations,omitempty"`
}

// ResourceCost represents requested vs used for a single resource type.
type ResourceCost struct {
	Requested  string  `json:"requested"`
	Used       string  `json:"used"`
	Efficiency float64 `json:"efficiency"` // used/requested * 100
}

// WorkloadCost represents cost data for a single workload (pod).
type WorkloadCost struct {
	Kind       string  `json:"kind"`
	Name       string  `json:"name"`
	Namespace  string  `json:"namespace"`
	CPUReq     string  `json:"cpuRequested"`
	CPUUsed    string  `json:"cpuUsed"`
	MemReq     string  `json:"memRequested"`
	MemUsed    string  `json:"memUsed"`
	Replicas   int32   `json:"replicas"`
	Efficiency float64 `json:"efficiency"`
}

// CostRecommendation provides optimization guidance for a workload.
type CostRecommendation struct {
	Workload    string `json:"workload"`
	Type        string `json:"type"` // "oversized", "undersized", "idle"
	Description string `json:"description"`
	Savings     string `json:"savings,omitempty"`
}

func (s *Server) handleCostEstimate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	namespace := r.URL.Query().Get("namespace")
	ctx := r.Context()

	pods, err := s.k8sClient.ListPods(ctx, namespace)
	if err != nil {
		http.Error(w, "Failed to list pods: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch pod metrics (actual usage); nil map if unavailable
	metricsMap, metricsErr := s.k8sClient.GetPodMetrics(ctx, namespace)
	metricsAvailable := metricsErr == nil && metricsMap != nil

	var workloads []WorkloadCost
	var recommendations []CostRecommendation
	var totalCPUReq, totalCPUUsed, totalMemReq, totalMemUsed int64

	for i := range pods {
		p := &pods[i]

		// Sum container resource requests
		var cpuReqMilli, memReqMB int64
		for _, c := range p.Spec.Containers {
			if req, ok := c.Resources.Requests[corev1.ResourceCPU]; ok {
				cpuReqMilli += req.MilliValue()
			}
			if req, ok := c.Resources.Requests[corev1.ResourceMemory]; ok {
				memReqMB += req.Value() / 1024 / 1024
			}
		}

		var cpuUsedMilli, memUsedMB int64
		if metricsAvailable {
			if vals, ok := metricsMap[p.Name]; ok && len(vals) >= 2 {
				cpuUsedMilli = vals[0]
				memUsedMB = vals[1]
			}
		}

		totalCPUReq += cpuReqMilli
		totalCPUUsed += cpuUsedMilli
		totalMemReq += memReqMB
		totalMemUsed += memUsedMB

		// Determine owner kind/name
		kind := "Pod"
		ownerName := p.Name
		if len(p.OwnerReferences) > 0 {
			kind = p.OwnerReferences[0].Kind
			ownerName = p.OwnerReferences[0].Name
		}

		eff := calcEfficiency(cpuReqMilli+memReqMB, cpuUsedMilli+memUsedMB, metricsAvailable)

		wl := WorkloadCost{
			Kind:       kind,
			Name:       ownerName,
			Namespace:  p.Namespace,
			CPUReq:     fmt.Sprintf("%dm", cpuReqMilli),
			CPUUsed:    fmt.Sprintf("%dm", cpuUsedMilli),
			MemReq:     fmt.Sprintf("%dMi", memReqMB),
			MemUsed:    fmt.Sprintf("%dMi", memUsedMB),
			Replicas:   1,
			Efficiency: eff,
		}
		workloads = append(workloads, wl)

		// Generate recommendations
		if metricsAvailable && cpuReqMilli > 0 {
			cpuEff := calcEfficiency(cpuReqMilli, cpuUsedMilli, true)
			memEff := calcEfficiency(memReqMB, memUsedMB, true)

			if cpuUsedMilli == 0 && memUsedMB == 0 {
				recommendations = append(recommendations, CostRecommendation{
					Workload:    p.Name,
					Type:        "idle",
					Description: fmt.Sprintf("Pod %s has no CPU or memory usage detected", p.Name),
				})
			} else if cpuEff < 30 || memEff < 30 {
				savingsCPU := cpuReqMilli - cpuUsedMilli*2 // leave 2x headroom
				if savingsCPU < 0 {
					savingsCPU = 0
				}
				recommendations = append(recommendations, CostRecommendation{
					Workload:    p.Name,
					Type:        "oversized",
					Description: fmt.Sprintf("Pod %s is using <30%% of requested resources (CPU: %.0f%%, Mem: %.0f%%)", p.Name, cpuEff, memEff),
					Savings:     fmt.Sprintf("Could save %dm CPU", savingsCPU),
				})
			} else if cpuEff > 80 || memEff > 80 {
				recommendations = append(recommendations, CostRecommendation{
					Workload:    p.Name,
					Type:        "undersized",
					Description: fmt.Sprintf("Pod %s is using >80%% of requested resources (CPU: %.0f%%, Mem: %.0f%%)", p.Name, cpuEff, memEff),
				})
			}
		}
	}

	overallEff := calcEfficiency(totalCPUReq+totalMemReq, totalCPUUsed+totalMemUsed, metricsAvailable)

	ns := namespace
	if ns == "" {
		ns = "all"
	}

	resp := CostEstimate{
		Namespace: ns,
		TotalCPU: ResourceCost{
			Requested:  fmt.Sprintf("%dm", totalCPUReq),
			Used:       fmt.Sprintf("%dm", totalCPUUsed),
			Efficiency: calcEfficiency(totalCPUReq, totalCPUUsed, metricsAvailable),
		},
		TotalMemory: ResourceCost{
			Requested:  fmt.Sprintf("%dMi", totalMemReq),
			Used:       fmt.Sprintf("%dMi", totalMemUsed),
			Efficiency: calcEfficiency(totalMemReq, totalMemUsed, metricsAvailable),
		},
		Workloads:       workloads,
		Efficiency:      overallEff,
		Recommendations: recommendations,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// calcEfficiency computes used/requested * 100. Returns -1 if metrics unavailable.
func calcEfficiency(requested, used int64, metricsAvailable bool) float64 {
	if !metricsAvailable {
		return -1
	}
	if requested <= 0 {
		return 0
	}
	return float64(used) / float64(requested) * 100
}
