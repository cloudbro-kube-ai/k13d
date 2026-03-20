package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
)

const nodeUsageBarWidth = 5

type nodeUsageSnapshot struct {
	CPUUsedMilli     int64
	CPUCapacityMilli int64
	CPUAvailable     bool
	MemUsedMB        int64
	MemCapacityMB    int64
	MemAvailable     bool
	GPUUsed          int64
	GPUCapacity      int64
	GPUAvailable     bool
	Estimated        bool
}

func loadNodeUsageSnapshots(ctx context.Context, client *k8s.Client, nodes []corev1.Node) map[string]nodeUsageSnapshot {
	snapshots := make(map[string]nodeUsageSnapshot, len(nodes))
	for _, node := range nodes {
		snapshots[node.Name] = nodeUsageSnapshot{
			CPUCapacityMilli: nodeCPUCapacityMilli(node),
			MemCapacityMB:    nodeMemoryCapacityMB(node),
			GPUCapacity:      nodeGPUCapacity(node),
		}
	}

	metrics, err := client.GetNodeMetrics(ctx)
	estimated := false
	usageAvailable := err == nil && len(metrics) > 0
	if !usageAvailable {
		metrics, err = client.GetNodeMetricsFromPodRequests(ctx)
		usageAvailable = err == nil
		estimated = usageAvailable
	}

	if usageAvailable {
		for _, node := range nodes {
			snapshot := snapshots[node.Name]
			snapshot.CPUAvailable = true
			snapshot.MemAvailable = true
			snapshot.Estimated = estimated
			if metric, ok := metrics[node.Name]; ok {
				snapshot.CPUUsedMilli = metric[0]
				snapshot.MemUsedMB = metric[1]
			}
			snapshots[node.Name] = snapshot
		}
	}

	requests, err := client.GetNodeResourceRequests(ctx)
	if err == nil {
		for _, node := range nodes {
			snapshot := snapshots[node.Name]
			snapshot.GPUAvailable = true
			if req, ok := requests[node.Name]; ok {
				snapshot.GPUUsed = req.GPU
			}
			snapshots[node.Name] = snapshot
		}
	}

	return snapshots
}

func nodeRoleSummary(node corev1.Node) string {
	roles := make([]string, 0, len(node.Labels))
	controlPlane := false
	for label := range node.Labels {
		if !strings.HasPrefix(label, "node-role.kubernetes.io/") {
			continue
		}
		role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
		switch role {
		case "control-plane", "master":
			controlPlane = true
		case "":
		default:
			roles = append(roles, role)
		}
	}

	sort.Strings(roles)
	if controlPlane {
		return "control-plane"
	}
	if len(roles) == 0 {
		return "worker"
	}
	return strings.Join(roles, ",")
}

func nodeStatusSummary(node corev1.Node) string {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

func nodeCPUCapacityMilli(node corev1.Node) int64 {
	if cpu, ok := node.Status.Allocatable[corev1.ResourceCPU]; ok {
		return cpu.MilliValue()
	}
	return 0
}

func nodeMemoryCapacityMB(node corev1.Node) int64 {
	if mem, ok := node.Status.Allocatable[corev1.ResourceMemory]; ok {
		return mem.Value() / 1024 / 1024
	}
	return 0
}

func nodeGPUCapacity(node corev1.Node) int64 {
	var total int64
	for name, qty := range node.Status.Allocatable {
		if strings.Contains(strings.ToLower(string(name)), "gpu") {
			total += qty.Value()
		}
	}
	return total
}

func formatNodeCPUUsage(snapshot nodeUsageSnapshot) string {
	return formatNodeUsageCell(snapshot.CPUUsedMilli, snapshot.CPUCapacityMilli, snapshot.CPUAvailable, snapshot.Estimated, formatCPUCoreValue)
}

func formatNodeMemoryUsage(snapshot nodeUsageSnapshot) string {
	return formatNodeUsageCell(snapshot.MemUsedMB, snapshot.MemCapacityMB, snapshot.MemAvailable, snapshot.Estimated, formatMemoryValueMB)
}

func formatNodeGPUUsage(snapshot nodeUsageSnapshot) string {
	return formatNodeUsageCell(snapshot.GPUUsed, snapshot.GPUCapacity, snapshot.GPUAvailable, false, formatCountValue)
}

func formatNodeUsageCell(used, capacity int64, available bool, estimated bool, formatter func(int64) string) string {
	if !available {
		return "-"
	}
	if capacity <= 0 {
		if used == 0 {
			return "-"
		}
		return fmt.Sprintf("%s/? %s", formatter(used), renderNodeUsageBar(used, capacity))
	}

	prefix := ""
	if estimated {
		prefix = "~"
	}
	return fmt.Sprintf("%s%s/%s %s", prefix, formatter(used), formatter(capacity), renderNodeUsageBar(used, capacity))
}

func renderNodeUsageBar(used, capacity int64) string {
	if capacity <= 0 {
		if used > 0 {
			return strings.Repeat("#", nodeUsageBarWidth)
		}
		return strings.Repeat("-", nodeUsageBarWidth)
	}

	filled := int(float64(used) / float64(capacity) * float64(nodeUsageBarWidth))
	if used > 0 && filled == 0 {
		filled = 1
	}
	if filled > nodeUsageBarWidth {
		filled = nodeUsageBarWidth
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("#", filled) + strings.Repeat("-", nodeUsageBarWidth-filled)
}

func formatCPUCoreValue(milli int64) string {
	cores := float64(milli) / 1000
	if milli%1000 == 0 {
		return fmt.Sprintf("%.0fc", cores)
	}
	return fmt.Sprintf("%.1fc", cores)
}

func formatMemoryValueMB(memMB int64) string {
	if memMB >= 1024 {
		gib := float64(memMB) / 1024
		if memMB%1024 == 0 {
			return fmt.Sprintf("%.0fGi", gib)
		}
		return fmt.Sprintf("%.1fGi", gib)
	}
	return fmt.Sprintf("%dMi", memMB)
}

func formatCountValue(v int64) string {
	return fmt.Sprintf("%d", v)
}
