package analyzers

import (
	"context"
	"fmt"
	"strings"
)

// CrossValidator performs cross-resource validation.
// Unlike single-resource analyzers, it checks relationships between multiple resources.
type CrossValidator struct{}

func (cv *CrossValidator) Name() string { return "cross-validator" }

// ValidateCross checks for mismatches between related resources.
// It takes a list of ResourceInfo from the same namespace and checks:
// 1. Service selector vs Pod labels mismatch
// 2. ConfigMap/Secret references that don't exist
// 3. Ingress referencing non-existent Services
// 4. HPA targeting non-existent Deployments/StatefulSets
func (cv *CrossValidator) ValidateCross(ctx context.Context, resources []*ResourceInfo) []Finding {
	if len(resources) == 0 {
		return nil
	}

	var findings []Finding
	findings = append(findings, cv.checkServicePodSelector(resources)...)
	findings = append(findings, cv.checkMissingConfigMapSecretRefs(resources)...)
	findings = append(findings, cv.checkIngressServiceRefs(resources)...)
	findings = append(findings, cv.checkHPATargetRefs(resources)...)
	return findings
}

// checkServicePodSelector checks if Service selectors match any Pod labels.
func (cv *CrossValidator) checkServicePodSelector(resources []*ResourceInfo) []Finding {
	// Collect all pods
	var pods []*ResourceInfo
	for _, r := range resources {
		if r.Kind == "Pod" {
			pods = append(pods, r)
		}
	}

	var findings []Finding
	for _, r := range resources {
		if r.Kind != "Service" {
			continue
		}
		selectorLabels := getStringMap(r.Raw, "selectorLabels")
		if len(selectorLabels) == 0 {
			continue
		}

		matched := false
		for _, pod := range pods {
			if labelsMatch(selectorLabels, pod.Labels) {
				matched = true
				break
			}
		}
		if !matched {
			ref := fmt.Sprintf("Service/%s/%s", r.Namespace, r.Name)
			findings = append(findings, Finding{
				Analyzer: "cross-validator",
				Resource: ref,
				Severity: SeverityWarning,
				Title:    fmt.Sprintf("Service '%s' selector does not match any pods", r.Name),
				Details:  fmt.Sprintf("Service selector labels %v do not match any pod labels in namespace '%s'.", selectorLabels, r.Namespace),
				Suggestions: []string{
					"Verify the service selector labels match pod labels",
					"Check that target pods are running in the same namespace",
					fmt.Sprintf("Run: kubectl get pods -n %s -l %s", r.Namespace, formatLabelSelector(selectorLabels)),
				},
			})
		}
	}
	return findings
}

// checkMissingConfigMapSecretRefs checks if referenced ConfigMaps/Secrets exist.
func (cv *CrossValidator) checkMissingConfigMapSecretRefs(resources []*ResourceInfo) []Finding {
	// Build sets of existing ConfigMaps and Secrets
	configMaps := make(map[string]bool)
	secrets := make(map[string]bool)
	for _, r := range resources {
		switch r.Kind {
		case "ConfigMap":
			configMaps[r.Name] = true
		case "Secret":
			secrets[r.Name] = true
		}
	}

	var findings []Finding
	for _, r := range resources {
		if r.Kind != "Pod" && r.Kind != "Deployment" {
			continue
		}
		ref := fmt.Sprintf("%s/%s/%s", r.Kind, r.Namespace, r.Name)

		// Check ConfigMap references
		cmRefs := getStringSlice(r.Raw, "configMapRefs")
		for _, cmName := range cmRefs {
			if !configMaps[cmName] {
				findings = append(findings, Finding{
					Analyzer: "cross-validator",
					Resource: ref,
					Severity: SeverityWarning,
					Title:    fmt.Sprintf("%s '%s' references ConfigMap '%s' which does not exist", r.Kind, r.Name, cmName),
					Details:  fmt.Sprintf("The %s references ConfigMap '%s', but no ConfigMap with that name was found in namespace '%s'.", r.Kind, cmName, r.Namespace),
					Suggestions: []string{
						fmt.Sprintf("Create the missing ConfigMap: kubectl create configmap %s -n %s", cmName, r.Namespace),
						"Check for typos in the ConfigMap name",
						fmt.Sprintf("Run: kubectl get configmaps -n %s", r.Namespace),
					},
				})
			}
		}

		// Check Secret references
		secretRefs := getStringSlice(r.Raw, "secretRefs")
		for _, secretName := range secretRefs {
			if !secrets[secretName] {
				findings = append(findings, Finding{
					Analyzer: "cross-validator",
					Resource: ref,
					Severity: SeverityWarning,
					Title:    fmt.Sprintf("%s '%s' references Secret '%s' which does not exist", r.Kind, r.Name, secretName),
					Details:  fmt.Sprintf("The %s references Secret '%s', but no Secret with that name was found in namespace '%s'.", r.Kind, secretName, r.Namespace),
					Suggestions: []string{
						fmt.Sprintf("Create the missing Secret: kubectl create secret generic %s -n %s", secretName, r.Namespace),
						"Check for typos in the Secret name",
						fmt.Sprintf("Run: kubectl get secrets -n %s", r.Namespace),
					},
				})
			}
		}
	}
	return findings
}

// checkIngressServiceRefs checks if Ingress backend services exist.
func (cv *CrossValidator) checkIngressServiceRefs(resources []*ResourceInfo) []Finding {
	// Build set of existing Services
	services := make(map[string]bool)
	for _, r := range resources {
		if r.Kind == "Service" {
			services[r.Name] = true
		}
	}

	var findings []Finding
	for _, r := range resources {
		if r.Kind != "Ingress" {
			continue
		}
		ref := fmt.Sprintf("Ingress/%s/%s", r.Namespace, r.Name)

		svcRefs := getStringSlice(r.Raw, "serviceRefs")
		for _, svcName := range svcRefs {
			if !services[svcName] {
				findings = append(findings, Finding{
					Analyzer: "cross-validator",
					Resource: ref,
					Severity: SeverityWarning,
					Title:    fmt.Sprintf("Ingress '%s' references Service '%s' which does not exist", r.Name, svcName),
					Details:  fmt.Sprintf("Ingress backend references Service '%s', but no Service with that name was found in namespace '%s'.", svcName, r.Namespace),
					Suggestions: []string{
						"Create the missing Service or update the Ingress backend",
						"Check for typos in the Service name",
						fmt.Sprintf("Run: kubectl get services -n %s", r.Namespace),
					},
				})
			}
		}
	}
	return findings
}

// checkHPATargetRefs checks if HPA target deployments/statefulsets exist.
func (cv *CrossValidator) checkHPATargetRefs(resources []*ResourceInfo) []Finding {
	// Build set of existing Deployments and StatefulSets
	workloads := make(map[string]bool)
	for _, r := range resources {
		if r.Kind == "Deployment" || r.Kind == "StatefulSet" {
			workloads[r.Kind+"/"+r.Name] = true
		}
	}

	var findings []Finding
	for _, r := range resources {
		if r.Kind != "HorizontalPodAutoscaler" {
			continue
		}
		ref := fmt.Sprintf("HorizontalPodAutoscaler/%s/%s", r.Namespace, r.Name)

		targetRef := getStringValue(r.Raw, "targetRef")
		if targetRef == "" {
			continue
		}

		if !workloads[targetRef] {
			parts := strings.SplitN(targetRef, "/", 2)
			targetKind := targetRef
			targetName := targetRef
			if len(parts) == 2 {
				targetKind = parts[0]
				targetName = parts[1]
			}
			findings = append(findings, Finding{
				Analyzer: "cross-validator",
				Resource: ref,
				Severity: SeverityWarning,
				Title:    fmt.Sprintf("HPA '%s' targets %s '%s' which does not exist", r.Name, targetKind, targetName),
				Details:  fmt.Sprintf("HPA references %s, but no matching workload was found in namespace '%s'.", targetRef, r.Namespace),
				Suggestions: []string{
					fmt.Sprintf("Create the missing %s or update the HPA target", targetKind),
					"Check for typos in the target reference",
					fmt.Sprintf("Run: kubectl get %s -n %s", strings.ToLower(targetKind)+"s", r.Namespace),
				},
			})
		}
	}
	return findings
}

// --- Helpers ---

// getStringMap extracts a map[string]string from Raw data.
func getStringMap(raw map[string]interface{}, key string) map[string]string {
	if raw == nil {
		return nil
	}
	v, ok := raw[key]
	if !ok {
		return nil
	}
	switch m := v.(type) {
	case map[string]string:
		return m
	case map[string]interface{}:
		result := make(map[string]string, len(m))
		for k, val := range m {
			if s, ok := val.(string); ok {
				result[k] = s
			}
		}
		return result
	}
	return nil
}

// getStringSlice extracts a []string from Raw data.
func getStringSlice(raw map[string]interface{}, key string) []string {
	if raw == nil {
		return nil
	}
	v, ok := raw[key]
	if !ok {
		return nil
	}
	switch s := v.(type) {
	case []string:
		return s
	case []interface{}:
		result := make([]string, 0, len(s))
		for _, item := range s {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return nil
}

// getStringValue extracts a string value from Raw data.
func getStringValue(raw map[string]interface{}, key string) string {
	if raw == nil {
		return ""
	}
	v, ok := raw[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// labelsMatch checks if all selector labels exist in the pod labels.
func labelsMatch(selector, labels map[string]string) bool {
	if len(selector) == 0 {
		return true
	}
	if len(labels) == 0 {
		return false
	}
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
}

// formatLabelSelector formats a label map as a kubectl label selector string.
func formatLabelSelector(labels map[string]string) string {
	parts := make([]string, 0, len(labels))
	for k, v := range labels {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ",")
}
