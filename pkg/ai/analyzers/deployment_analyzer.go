package analyzers

import (
	"context"
	"fmt"
)

// DeploymentAnalyzer checks for common Deployment issues.
type DeploymentAnalyzer struct{}

func (a *DeploymentAnalyzer) Name() string { return "deployment" }

func (a *DeploymentAnalyzer) Analyze(ctx context.Context, resource *ResourceInfo) []Finding {
	if resource == nil || resource.Kind != "Deployment" {
		return nil
	}

	var findings []Finding
	ref := fmt.Sprintf("Deployment/%s/%s", resource.Namespace, resource.Name)

	findings = append(findings, a.checkUnavailableReplicas(ref, resource)...)
	findings = append(findings, a.checkRolloutStuck(ref, resource)...)
	findings = append(findings, a.checkZeroReplicas(ref, resource)...)

	return findings
}

func (a *DeploymentAnalyzer) checkUnavailableReplicas(ref string, resource *ResourceInfo) []Finding {
	for _, c := range resource.Conditions {
		if c.Type == "Available" && c.Status == "False" {
			return []Finding{{
				Analyzer: "deployment",
				Resource: ref,
				Severity: SeverityCritical,
				Title:    "Deployment has unavailable replicas",
				Details:  fmt.Sprintf("Deployment does not have minimum availability. Reason: %s. %s", c.Reason, c.Message),
				Suggestions: []string{
					"Check pod status: kubectl get pods -l app=" + resource.Name + " -n " + resource.Namespace,
					"Review pod events for scheduling or resource issues",
					"Verify resource requests don't exceed node capacity",
				},
			}}
		}
	}
	return nil
}

func (a *DeploymentAnalyzer) checkRolloutStuck(ref string, resource *ResourceInfo) []Finding {
	for _, c := range resource.Conditions {
		if c.Type == "Progressing" && c.Status == "False" {
			return []Finding{{
				Analyzer: "deployment",
				Resource: ref,
				Severity: SeverityCritical,
				Title:    "Deployment rollout is stuck",
				Details:  fmt.Sprintf("Deployment is not progressing. Reason: %s. %s", c.Reason, c.Message),
				Suggestions: []string{
					"Check if new pods are failing to start",
					"Review the deployment rollout history: kubectl rollout history deployment/" + resource.Name + " -n " + resource.Namespace,
					"Consider rolling back: kubectl rollout undo deployment/" + resource.Name + " -n " + resource.Namespace,
				},
			}}
		}
	}
	return nil
}

func (a *DeploymentAnalyzer) checkZeroReplicas(ref string, resource *ResourceInfo) []Finding {
	if resource.Status == "0/0" || resource.Status == "scaled-to-zero" {
		return []Finding{{
			Analyzer: "deployment",
			Resource: ref,
			Severity: SeverityInfo,
			Title:    "Deployment is scaled to zero replicas",
			Details:  "Deployment has zero desired replicas. No pods are running.",
			Suggestions: []string{
				"If intentional, no action needed",
				"To scale up: kubectl scale deployment/" + resource.Name + " --replicas=1 -n " + resource.Namespace,
			},
		}}
	}
	return nil
}
