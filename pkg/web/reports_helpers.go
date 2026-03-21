package web

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func normalizeReportSections(sections *ReportSections) ReportSections {
	if sections == nil {
		return *AllSections()
	}

	normalized := *sections
	if normalized.SecurityFull {
		normalized.SecurityBasic = true
	}
	return normalized
}

func reportSectionsOrAll(report *ComprehensiveReport) ReportSections {
	if report == nil {
		return *AllSections()
	}
	sections := report.IncludedSections
	if !sections.Nodes && !sections.Namespaces && !sections.Workloads &&
		!sections.Events && !sections.SecurityBasic && !sections.SecurityFull &&
		!sections.FinOps && !sections.Metrics {
		return *AllSections()
	}
	if sections.SecurityFull {
		sections.SecurityBasic = true
	}
	return sections
}

func formatNodeTaints(node corev1.Node) []string {
	if len(node.Spec.Taints) == 0 {
		return nil
	}

	taints := make([]string, 0, len(node.Spec.Taints))
	for _, taint := range node.Spec.Taints {
		value := taint.Key
		if taint.Value != "" {
			value = fmt.Sprintf("%s=%s", taint.Key, taint.Value)
		}
		if taint.Effect != "" {
			value = fmt.Sprintf("%s:%s", value, taint.Effect)
		}
		taints = append(taints, value)
	}
	return taints
}

func nodeWarnings(node corev1.Node) (warnings []string, hasPressure bool) {
	if node.Spec.Unschedulable {
		warnings = append(warnings, "Cordoned")
	}

	for _, condition := range node.Status.Conditions {
		switch condition.Type {
		case corev1.NodeReady:
			if condition.Status != corev1.ConditionTrue {
				warnings = append(warnings, "NotReady")
			}
		case corev1.NodeMemoryPressure:
			if condition.Status == corev1.ConditionTrue {
				warnings = append(warnings, "MemoryPressure")
				hasPressure = true
			}
		case corev1.NodeDiskPressure:
			if condition.Status == corev1.ConditionTrue {
				warnings = append(warnings, "DiskPressure")
				hasPressure = true
			}
		case corev1.NodePIDPressure:
			if condition.Status == corev1.ConditionTrue {
				warnings = append(warnings, "PIDPressure")
				hasPressure = true
			}
		case corev1.NodeNetworkUnavailable:
			if condition.Status == corev1.ConditionTrue {
				warnings = append(warnings, "NetworkUnavailable")
			}
		}
	}

	return warnings, hasPressure
}
