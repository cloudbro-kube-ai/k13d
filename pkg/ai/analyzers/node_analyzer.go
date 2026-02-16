package analyzers

import (
	"context"
	"fmt"
)

// NodeAnalyzer checks for common Node issues.
type NodeAnalyzer struct{}

func (a *NodeAnalyzer) Name() string { return "node" }

func (a *NodeAnalyzer) Analyze(ctx context.Context, resource *ResourceInfo) []Finding {
	if resource == nil || resource.Kind != "Node" {
		return nil
	}

	var findings []Finding
	ref := fmt.Sprintf("Node/%s", resource.Name)

	for _, c := range resource.Conditions {
		findings = append(findings, a.analyzeCondition(ref, resource, &c)...)
	}

	return findings
}

func (a *NodeAnalyzer) analyzeCondition(ref string, resource *ResourceInfo, c *Condition) []Finding {
	switch c.Type {
	case "Ready":
		if c.Status == "False" || c.Status == "Unknown" {
			return []Finding{{
				Analyzer: "node",
				Resource: ref,
				Severity: SeverityCritical,
				Title:    fmt.Sprintf("Node %s is NotReady", resource.Name),
				Details:  fmt.Sprintf("Node is in %s state. Reason: %s. %s", c.Status, c.Reason, c.Message),
				Suggestions: []string{
					"Check kubelet status: systemctl status kubelet",
					"Check node resources (CPU, memory, disk)",
					"Review kubelet logs: journalctl -u kubelet",
					"Verify network connectivity to the API server",
				},
			}}
		}
	case "DiskPressure":
		if c.Status == "True" {
			return []Finding{{
				Analyzer: "node",
				Resource: ref,
				Severity: SeverityCritical,
				Title:    fmt.Sprintf("Node %s has DiskPressure", resource.Name),
				Details:  fmt.Sprintf("Node is experiencing disk pressure. %s", c.Message),
				Suggestions: []string{
					"Clean up unused images: docker system prune or crictl rmi --prune",
					"Remove unused containers and volumes",
					"Check for large log files",
					"Consider expanding disk capacity",
				},
			}}
		}
	case "MemoryPressure":
		if c.Status == "True" {
			return []Finding{{
				Analyzer: "node",
				Resource: ref,
				Severity: SeverityCritical,
				Title:    fmt.Sprintf("Node %s has MemoryPressure", resource.Name),
				Details:  fmt.Sprintf("Node is experiencing memory pressure. %s", c.Message),
				Suggestions: []string{
					"Identify memory-heavy pods: kubectl top pods --sort-by=memory",
					"Consider evicting or rescheduling non-critical workloads",
					"Review pod memory limits and requests",
					"Consider adding more memory or nodes to the cluster",
				},
			}}
		}
	case "PIDPressure":
		if c.Status == "True" {
			return []Finding{{
				Analyzer: "node",
				Resource: ref,
				Severity: SeverityWarning,
				Title:    fmt.Sprintf("Node %s has PIDPressure", resource.Name),
				Details:  fmt.Sprintf("Node is running low on available process IDs. %s", c.Message),
				Suggestions: []string{
					"Identify pods with many processes",
					"Check for fork bombs or runaway processes",
					"Consider increasing the PID limit on the node",
				},
			}}
		}
	}

	// Check unschedulable (cordon) via status field
	if resource.Status == "SchedulingDisabled" || resource.Status == "cordoned" {
		// Only emit once by checking the condition type
		if c.Type == "Ready" {
			return []Finding{{
				Analyzer: "node",
				Resource: ref,
				Severity: SeverityInfo,
				Title:    fmt.Sprintf("Node %s is cordoned (unschedulable)", resource.Name),
				Details:  "Node is marked as unschedulable. No new pods will be scheduled on this node.",
				Suggestions: []string{
					"If maintenance is complete, uncordon the node: kubectl uncordon " + resource.Name,
					"Check if workloads need to be drained: kubectl drain " + resource.Name + " --ignore-daemonsets",
				},
			}}
		}
	}

	return nil
}
