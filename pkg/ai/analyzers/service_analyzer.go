package analyzers

import (
	"context"
	"fmt"
)

// ServiceAnalyzer checks for common Service issues.
type ServiceAnalyzer struct{}

func (a *ServiceAnalyzer) Name() string { return "service" }

func (a *ServiceAnalyzer) Analyze(ctx context.Context, resource *ResourceInfo) []Finding {
	if resource == nil || resource.Kind != "Service" {
		return nil
	}

	var findings []Finding
	ref := fmt.Sprintf("Service/%s/%s", resource.Namespace, resource.Name)

	findings = append(findings, a.checkNoEndpoints(ref, resource)...)
	findings = append(findings, a.checkMismatchedPorts(ref, resource)...)

	return findings
}

func (a *ServiceAnalyzer) checkNoEndpoints(ref string, resource *ResourceInfo) []Finding {
	// Look for events or conditions indicating no endpoints
	for _, e := range resource.Events {
		if e.Reason == "NoEndpoints" || e.Reason == "FailedToSelectEndpoints" {
			return []Finding{{
				Analyzer: "service",
				Resource: ref,
				Severity: SeverityWarning,
				Title:    "Service has no endpoints",
				Details:  fmt.Sprintf("Service selector does not match any running pods. %s", e.Message),
				Suggestions: []string{
					"Verify the service selector labels match pod labels",
					"Check that target pods are running and ready",
					"Run: kubectl get endpoints " + resource.Name + " -n " + resource.Namespace,
				},
			}}
		}
	}

	// Check for no-endpoints condition
	for _, c := range resource.Conditions {
		if c.Type == "NoEndpoints" && c.Status == "True" {
			return []Finding{{
				Analyzer: "service",
				Resource: ref,
				Severity: SeverityWarning,
				Title:    "Service has no endpoints",
				Details:  fmt.Sprintf("Service has no matching endpoints. %s", c.Message),
				Suggestions: []string{
					"Verify the service selector labels match pod labels",
					"Check that target pods are running and ready",
					"Run: kubectl get endpoints " + resource.Name + " -n " + resource.Namespace,
				},
			}}
		}
	}

	return nil
}

func (a *ServiceAnalyzer) checkMismatchedPorts(ref string, resource *ResourceInfo) []Finding {
	// Detect port mismatch via events
	for _, e := range resource.Events {
		if e.Reason == "PortMismatch" || e.Reason == "MismatchedPorts" {
			return []Finding{{
				Analyzer: "service",
				Resource: ref,
				Severity: SeverityWarning,
				Title:    "Service port mismatch detected",
				Details:  fmt.Sprintf("Service port may not match container port. %s", e.Message),
				Suggestions: []string{
					"Verify the service targetPort matches the container port",
					"Check the pod spec for the correct container port",
					"Run: kubectl describe svc " + resource.Name + " -n " + resource.Namespace,
				},
			}}
		}
	}

	return nil
}
