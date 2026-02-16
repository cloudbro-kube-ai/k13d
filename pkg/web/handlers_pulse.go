package web

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"time"
)

// WebPulseData is the JSON response for /api/pulse.
// Mirrors ui.PulseData but decoupled from the TUI package.
type WebPulseData struct {
	// Pod counts
	PodsRunning int `json:"pods_running"`
	PodsPending int `json:"pods_pending"`
	PodsFailed  int `json:"pods_failed"`
	PodsOther   int `json:"pods_other"`
	PodsTotal   int `json:"pods_total"`

	// Deployment counts
	DeploysReady    int `json:"deploys_ready"`
	DeploysUpdating int `json:"deploys_updating"`
	DeploysTotal    int `json:"deploys_total"`

	// StatefulSet counts
	STSReady int `json:"sts_ready"`
	STSTotal int `json:"sts_total"`

	// DaemonSet counts
	DSReady int `json:"ds_ready"`
	DSTotal int `json:"ds_total"`

	// Job counts
	JobsComplete int `json:"jobs_complete"`
	JobsActive   int `json:"jobs_active"`
	JobsFailed   int `json:"jobs_failed"`
	JobsTotal    int `json:"jobs_total"`

	// Node counts
	NodesReady    int `json:"nodes_ready"`
	NodesNotReady int `json:"nodes_not_ready"`
	NodesTotal    int `json:"nodes_total"`

	// CPU metrics (millicores)
	CPUUsed     int64 `json:"cpu_used_milli"`
	CPUCapacity int64 `json:"cpu_capacity_milli"`
	CPUAvail    bool  `json:"cpu_avail"`

	// Memory metrics (MiB)
	MemUsed     int64 `json:"mem_used_mib"`
	MemCapacity int64 `json:"mem_capacity_mib"`
	MemAvail    bool  `json:"mem_avail"`

	// Recent events
	Events []WebPulseEvent `json:"events"`

	Timestamp time.Time `json:"timestamp"`
}

// WebPulseEvent is a simplified event for the pulse response.
type WebPulseEvent struct {
	Type    string `json:"type"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
	Age     string `json:"age"`
}

// handlePulse returns a JSON snapshot of cluster health (like the TUI :pulse view).
func (s *Server) handlePulse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		MethodNotAllowed(w, http.MethodGet)
		return
	}

	if s.k8sClient == nil {
		WriteError(w, NewAPIError(ErrCodeK8sError, "Kubernetes client not available"))
		return
	}

	ns := r.URL.Query().Get("namespace")
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	data := s.fetchPulseData(ctx, ns)
	data.Timestamp = time.Now()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// fetchPulseData gathers cluster data, mirroring ui.PulseView.fetchData.
func (s *Server) fetchPulseData(ctx context.Context, namespace string) WebPulseData {
	var data WebPulseData
	k := s.k8sClient

	// Pods
	if pods, err := k.ListPods(ctx, namespace); err == nil {
		data.PodsTotal = len(pods)
		for _, pod := range pods {
			switch pod.Status.Phase {
			case "Running":
				data.PodsRunning++
			case "Pending":
				data.PodsPending++
			case "Failed":
				data.PodsFailed++
			default:
				data.PodsOther++
			}
		}
	}

	// Deployments
	if deps, err := k.ListDeployments(ctx, namespace); err == nil {
		data.DeploysTotal = len(deps)
		for _, dep := range deps {
			desired := int32(1)
			if dep.Spec.Replicas != nil {
				desired = *dep.Spec.Replicas
			}
			if dep.Status.ReadyReplicas >= desired && dep.Status.UnavailableReplicas == 0 {
				data.DeploysReady++
			} else {
				data.DeploysUpdating++
			}
		}
	}

	// StatefulSets
	if stses, err := k.ListStatefulSets(ctx, namespace); err == nil {
		data.STSTotal = len(stses)
		for _, sts := range stses {
			desired := int32(1)
			if sts.Spec.Replicas != nil {
				desired = *sts.Spec.Replicas
			}
			if sts.Status.ReadyReplicas >= desired {
				data.STSReady++
			}
		}
	}

	// DaemonSets
	if dss, err := k.ListDaemonSets(ctx, namespace); err == nil {
		data.DSTotal = len(dss)
		for _, ds := range dss {
			if ds.Status.NumberReady >= ds.Status.DesiredNumberScheduled {
				data.DSReady++
			}
		}
	}

	// Jobs
	if jobs, err := k.ListJobs(ctx, namespace); err == nil {
		data.JobsTotal = len(jobs)
		for _, job := range jobs {
			if job.Status.Succeeded > 0 && job.Status.Active == 0 {
				data.JobsComplete++
			} else if job.Status.Failed > 0 && job.Status.Active == 0 {
				data.JobsFailed++
			} else if job.Status.Active > 0 {
				data.JobsActive++
			}
		}
	}

	// Nodes
	if nodes, err := k.ListNodes(ctx); err == nil {
		data.NodesTotal = len(nodes)
		for _, node := range nodes {
			ready := false
			for _, cond := range node.Status.Conditions {
				if cond.Type == "Ready" && cond.Status == "True" {
					ready = true
					break
				}
			}
			if ready {
				data.NodesReady++
			} else {
				data.NodesNotReady++
			}
		}

		// CPU/Memory capacity from nodes
		for _, node := range nodes {
			if cpu, ok := node.Status.Allocatable["cpu"]; ok {
				data.CPUCapacity += cpu.MilliValue()
			}
			if mem, ok := node.Status.Allocatable["memory"]; ok {
				data.MemCapacity += mem.Value() / 1024 / 1024
			}
		}
	}

	// Node metrics (usage)
	if nodeMetrics, err := k.GetNodeMetrics(ctx); err == nil && len(nodeMetrics) > 0 {
		data.CPUAvail = true
		data.MemAvail = true
		for _, m := range nodeMetrics {
			data.CPUUsed += m[0]
			data.MemUsed += m[1]
		}
	}

	// Events (last 5 most recent)
	if events, err := k.ListEvents(ctx, namespace); err == nil {
		sort.Slice(events, func(i, j int) bool {
			return events[i].LastTimestamp.Time.After(events[j].LastTimestamp.Time)
		})
		count := 0
		for _, ev := range events {
			if count >= 5 {
				break
			}
			age := ""
			if !ev.LastTimestamp.Time.IsZero() {
				age = formatAge(ev.LastTimestamp.Time)
			}
			msg := ev.Message
			if len(msg) > 80 {
				msg = msg[:77] + "..."
			}
			data.Events = append(data.Events, WebPulseEvent{
				Type:    ev.Type,
				Reason:  ev.Reason,
				Message: msg,
				Age:     age,
			})
			count++
		}
	}

	return data
}

// formatAge is defined in operations.go — do not redeclare here.
