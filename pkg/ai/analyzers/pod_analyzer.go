package analyzers

import (
	"context"
	"fmt"
)

// PodAnalyzer checks for common Pod failure conditions.
type PodAnalyzer struct{}

func (a *PodAnalyzer) Name() string { return "pod" }

func (a *PodAnalyzer) Analyze(ctx context.Context, resource *ResourceInfo) []Finding {
	if resource == nil || resource.Kind != "Pod" {
		return nil
	}

	var findings []Finding
	ref := fmt.Sprintf("Pod/%s/%s", resource.Namespace, resource.Name)

	for _, c := range resource.Containers {
		findings = append(findings, a.analyzeContainer(ref, &c)...)
	}

	findings = append(findings, a.analyzePendingPod(ref, resource)...)

	return findings
}

func (a *PodAnalyzer) analyzeContainer(ref string, c *ContainerInfo) []Finding {
	var findings []Finding

	// CrashLoopBackOff detection
	if c.Reason == "CrashLoopBackOff" {
		findings = append(findings, Finding{
			Analyzer: "pod",
			Resource: ref,
			Severity: SeverityCritical,
			Title:    fmt.Sprintf("Container %q is in CrashLoopBackOff", c.Name),
			Details:  fmt.Sprintf("Container %q is repeatedly crashing and being restarted. Restart count: %d.", c.Name, c.RestartCount),
			Suggestions: []string{
				"Check container logs: kubectl logs <pod> -c " + c.Name,
				"Check for application errors or misconfigurations",
				"Verify the container entrypoint and command are correct",
				"Check resource limits - the container may be OOMKilled",
			},
		})
	}

	// OOMKilled detection
	if c.Reason == "OOMKilled" || c.ExitCode == 137 {
		findings = append(findings, Finding{
			Analyzer: "pod",
			Resource: ref,
			Severity: SeverityCritical,
			Title:    fmt.Sprintf("Container %q was OOMKilled", c.Name),
			Details:  fmt.Sprintf("Container %q was terminated due to exceeding its memory limit (exit code %d).", c.Name, c.ExitCode),
			Suggestions: []string{
				"Increase the container memory limit in the pod spec",
				"Profile the application to identify memory leaks",
				"Check if the application has a memory-bound workload that needs tuning",
			},
		})
	}

	// ImagePullBackOff / ErrImagePull detection
	if c.Reason == "ImagePullBackOff" || c.Reason == "ErrImagePull" {
		findings = append(findings, Finding{
			Analyzer: "pod",
			Resource: ref,
			Severity: SeverityCritical,
			Title:    fmt.Sprintf("Container %q has image pull error: %s", c.Name, c.Reason),
			Details:  fmt.Sprintf("Container %q cannot pull image %q. Reason: %s. %s", c.Name, c.Image, c.Reason, c.Message),
			Suggestions: []string{
				"Verify the image name and tag are correct",
				"Check if the image exists in the registry",
				"Ensure image pull secrets are configured if using a private registry",
				"Check network connectivity to the container registry",
			},
		})
	}

	// High restart count
	if c.RestartCount > 5 && c.Reason != "CrashLoopBackOff" {
		findings = append(findings, Finding{
			Analyzer: "pod",
			Resource: ref,
			Severity: SeverityWarning,
			Title:    fmt.Sprintf("Container %q has high restart count (%d)", c.Name, c.RestartCount),
			Details:  fmt.Sprintf("Container %q has been restarted %d times, indicating instability.", c.Name, c.RestartCount),
			Suggestions: []string{
				"Check container logs for recurring errors",
				"Review liveness and readiness probe configurations",
				"Check resource limits and requests",
			},
		})
	}

	// Container not ready but running
	if c.State == "running" && !c.Ready {
		findings = append(findings, Finding{
			Analyzer: "pod",
			Resource: ref,
			Severity: SeverityWarning,
			Title:    fmt.Sprintf("Container %q is running but not ready", c.Name),
			Details:  fmt.Sprintf("Container %q is in running state but has not passed readiness checks.", c.Name),
			Suggestions: []string{
				"Check readiness probe configuration and endpoint",
				"Verify the application is listening on the expected port",
				"Check application startup time and adjust initialDelaySeconds if needed",
			},
		})
	}

	return findings
}

func (a *PodAnalyzer) analyzePendingPod(ref string, resource *ResourceInfo) []Finding {
	if resource.Status != "Pending" {
		return nil
	}

	// Check for scheduling-related events
	hasSchedulingEvents := false
	for _, e := range resource.Events {
		if e.Reason == "FailedScheduling" || e.Reason == "Scheduled" {
			hasSchedulingEvents = true
			break
		}
	}

	if !hasSchedulingEvents {
		return []Finding{{
			Analyzer: "pod",
			Resource: ref,
			Severity: SeverityWarning,
			Title:    "Pod is stuck in Pending state",
			Details:  "Pod is in Pending state with no scheduling events, which may indicate resource constraints or node selector issues.",
			Suggestions: []string{
				"Check node resources: kubectl describe nodes",
				"Verify node selectors and tolerations match available nodes",
				"Check if PersistentVolumeClaims are bound",
				"Review resource requests against cluster capacity",
			},
		}}
	}

	return nil
}
