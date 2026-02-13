package analyzers

import (
	"context"
	"testing"
)

func TestCrossValidator_Name(t *testing.T) {
	cv := &CrossValidator{}
	if cv.Name() != "cross-validator" {
		t.Errorf("expected name 'cross-validator', got %q", cv.Name())
	}
}

func TestCrossValidator_EmptyResources(t *testing.T) {
	cv := &CrossValidator{}
	findings := cv.ValidateCross(context.Background(), nil)
	if len(findings) != 0 {
		t.Errorf("expected no findings for nil resources, got %d", len(findings))
	}

	findings = cv.ValidateCross(context.Background(), []*ResourceInfo{})
	if len(findings) != 0 {
		t.Errorf("expected no findings for empty resources, got %d", len(findings))
	}
}

func TestCrossValidator_ServicePodMatch(t *testing.T) {
	cv := &CrossValidator{}
	resources := []*ResourceInfo{
		{
			Kind: "Service", Name: "my-svc", Namespace: "default",
			Raw: map[string]interface{}{
				"selectorLabels": map[string]string{"app": "web"},
			},
		},
		{
			Kind: "Pod", Name: "web-pod", Namespace: "default",
			Labels: map[string]string{"app": "web", "version": "v1"},
		},
	}

	findings := cv.ValidateCross(context.Background(), resources)
	if len(findings) != 0 {
		t.Errorf("expected no findings when service selector matches pod, got %d: %+v", len(findings), findings)
	}
}

func TestCrossValidator_ServicePodMismatch(t *testing.T) {
	cv := &CrossValidator{}
	resources := []*ResourceInfo{
		{
			Kind: "Service", Name: "my-svc", Namespace: "default",
			Raw: map[string]interface{}{
				"selectorLabels": map[string]string{"app": "web"},
			},
		},
		{
			Kind: "Pod", Name: "other-pod", Namespace: "default",
			Labels: map[string]string{"app": "api"},
		},
	}

	findings := cv.ValidateCross(context.Background(), resources)
	assertFindingExists(t, findings, SeverityWarning, "selector does not match any pods")
}

func TestCrossValidator_ServiceNoSelector(t *testing.T) {
	cv := &CrossValidator{}
	resources := []*ResourceInfo{
		{
			Kind: "Service", Name: "headless-svc", Namespace: "default",
			Raw: map[string]interface{}{},
		},
	}

	findings := cv.ValidateCross(context.Background(), resources)
	if len(findings) != 0 {
		t.Errorf("service with no selector should produce no findings, got %d", len(findings))
	}
}

func TestCrossValidator_ServiceNoPods(t *testing.T) {
	cv := &CrossValidator{}
	resources := []*ResourceInfo{
		{
			Kind: "Service", Name: "lonely-svc", Namespace: "default",
			Raw: map[string]interface{}{
				"selectorLabels": map[string]string{"app": "ghost"},
			},
		},
	}

	findings := cv.ValidateCross(context.Background(), resources)
	assertFindingExists(t, findings, SeverityWarning, "selector does not match any pods")
}

func TestCrossValidator_MissingConfigMap(t *testing.T) {
	cv := &CrossValidator{}
	resources := []*ResourceInfo{
		{
			Kind: "Pod", Name: "my-pod", Namespace: "default",
			Raw: map[string]interface{}{
				"configMapRefs": []string{"app-config"},
			},
		},
	}

	findings := cv.ValidateCross(context.Background(), resources)
	assertFindingExists(t, findings, SeverityWarning, "references ConfigMap 'app-config' which does not exist")
}

func TestCrossValidator_ExistingConfigMap(t *testing.T) {
	cv := &CrossValidator{}
	resources := []*ResourceInfo{
		{
			Kind: "Pod", Name: "my-pod", Namespace: "default",
			Raw: map[string]interface{}{
				"configMapRefs": []string{"app-config"},
			},
		},
		{Kind: "ConfigMap", Name: "app-config", Namespace: "default"},
	}

	findings := cv.ValidateCross(context.Background(), resources)
	for _, f := range findings {
		if containsSubstring(f.Title, "ConfigMap") {
			t.Errorf("should not report missing ConfigMap when it exists: %+v", f)
		}
	}
}

func TestCrossValidator_MissingSecret(t *testing.T) {
	cv := &CrossValidator{}
	resources := []*ResourceInfo{
		{
			Kind: "Pod", Name: "my-pod", Namespace: "default",
			Raw: map[string]interface{}{
				"secretRefs": []string{"db-password"},
			},
		},
	}

	findings := cv.ValidateCross(context.Background(), resources)
	assertFindingExists(t, findings, SeverityWarning, "references Secret 'db-password' which does not exist")
}

func TestCrossValidator_ExistingSecret(t *testing.T) {
	cv := &CrossValidator{}
	resources := []*ResourceInfo{
		{
			Kind: "Pod", Name: "my-pod", Namespace: "default",
			Raw: map[string]interface{}{
				"secretRefs": []string{"db-password"},
			},
		},
		{Kind: "Secret", Name: "db-password", Namespace: "default"},
	}

	findings := cv.ValidateCross(context.Background(), resources)
	for _, f := range findings {
		if containsSubstring(f.Title, "Secret") {
			t.Errorf("should not report missing Secret when it exists: %+v", f)
		}
	}
}

func TestCrossValidator_DeploymentMissingConfigMap(t *testing.T) {
	cv := &CrossValidator{}
	resources := []*ResourceInfo{
		{
			Kind: "Deployment", Name: "my-deploy", Namespace: "default",
			Raw: map[string]interface{}{
				"configMapRefs": []string{"missing-cm"},
			},
		},
	}

	findings := cv.ValidateCross(context.Background(), resources)
	assertFindingExists(t, findings, SeverityWarning, "Deployment 'my-deploy' references ConfigMap 'missing-cm'")
}

func TestCrossValidator_IngressMissingService(t *testing.T) {
	cv := &CrossValidator{}
	resources := []*ResourceInfo{
		{
			Kind: "Ingress", Name: "my-ingress", Namespace: "default",
			Raw: map[string]interface{}{
				"serviceRefs": []string{"frontend-svc"},
			},
		},
	}

	findings := cv.ValidateCross(context.Background(), resources)
	assertFindingExists(t, findings, SeverityWarning, "Ingress 'my-ingress' references Service 'frontend-svc' which does not exist")
}

func TestCrossValidator_IngressExistingService(t *testing.T) {
	cv := &CrossValidator{}
	resources := []*ResourceInfo{
		{
			Kind: "Ingress", Name: "my-ingress", Namespace: "default",
			Raw: map[string]interface{}{
				"serviceRefs": []string{"frontend-svc"},
			},
		},
		{Kind: "Service", Name: "frontend-svc", Namespace: "default"},
	}

	findings := cv.ValidateCross(context.Background(), resources)
	for _, f := range findings {
		if containsSubstring(f.Title, "Ingress") {
			t.Errorf("should not report missing Service when it exists: %+v", f)
		}
	}
}

func TestCrossValidator_HPAMissingTarget(t *testing.T) {
	cv := &CrossValidator{}
	resources := []*ResourceInfo{
		{
			Kind: "HorizontalPodAutoscaler", Name: "my-hpa", Namespace: "default",
			Raw: map[string]interface{}{
				"targetRef": "Deployment/my-deploy",
			},
		},
	}

	findings := cv.ValidateCross(context.Background(), resources)
	assertFindingExists(t, findings, SeverityWarning, "HPA 'my-hpa' targets Deployment 'my-deploy' which does not exist")
}

func TestCrossValidator_HPAExistingTarget(t *testing.T) {
	cv := &CrossValidator{}
	resources := []*ResourceInfo{
		{
			Kind: "HorizontalPodAutoscaler", Name: "my-hpa", Namespace: "default",
			Raw: map[string]interface{}{
				"targetRef": "Deployment/my-deploy",
			},
		},
		{Kind: "Deployment", Name: "my-deploy", Namespace: "default"},
	}

	findings := cv.ValidateCross(context.Background(), resources)
	for _, f := range findings {
		if containsSubstring(f.Title, "HPA") {
			t.Errorf("should not report missing target when it exists: %+v", f)
		}
	}
}

func TestCrossValidator_HPAStatefulSetTarget(t *testing.T) {
	cv := &CrossValidator{}
	resources := []*ResourceInfo{
		{
			Kind: "HorizontalPodAutoscaler", Name: "ss-hpa", Namespace: "default",
			Raw: map[string]interface{}{
				"targetRef": "StatefulSet/my-ss",
			},
		},
		{Kind: "StatefulSet", Name: "my-ss", Namespace: "default"},
	}

	findings := cv.ValidateCross(context.Background(), resources)
	for _, f := range findings {
		if containsSubstring(f.Title, "HPA") {
			t.Errorf("should not report missing StatefulSet target when it exists: %+v", f)
		}
	}
}

func TestCrossValidator_HPANoTargetRef(t *testing.T) {
	cv := &CrossValidator{}
	resources := []*ResourceInfo{
		{
			Kind: "HorizontalPodAutoscaler", Name: "empty-hpa", Namespace: "default",
			Raw: map[string]interface{}{},
		},
	}

	findings := cv.ValidateCross(context.Background(), resources)
	for _, f := range findings {
		if containsSubstring(f.Title, "HPA") {
			t.Errorf("HPA with no targetRef should produce no findings: %+v", f)
		}
	}
}

func TestCrossValidator_AllValid(t *testing.T) {
	cv := &CrossValidator{}
	resources := []*ResourceInfo{
		{
			Kind: "Service", Name: "web-svc", Namespace: "default",
			Raw: map[string]interface{}{
				"selectorLabels": map[string]string{"app": "web"},
			},
		},
		{
			Kind: "Pod", Name: "web-pod-1", Namespace: "default",
			Labels: map[string]string{"app": "web"},
			Raw: map[string]interface{}{
				"configMapRefs": []string{"web-config"},
				"secretRefs":    []string{"web-secret"},
			},
		},
		{Kind: "ConfigMap", Name: "web-config", Namespace: "default"},
		{Kind: "Secret", Name: "web-secret", Namespace: "default"},
		{
			Kind: "Ingress", Name: "web-ingress", Namespace: "default",
			Raw: map[string]interface{}{
				"serviceRefs": []string{"web-svc"},
			},
		},
		{
			Kind: "HorizontalPodAutoscaler", Name: "web-hpa", Namespace: "default",
			Raw: map[string]interface{}{
				"targetRef": "Deployment/web-deploy",
			},
		},
		{Kind: "Deployment", Name: "web-deploy", Namespace: "default"},
	}

	findings := cv.ValidateCross(context.Background(), resources)
	if len(findings) != 0 {
		t.Errorf("expected no findings when all references are valid, got %d: %+v", len(findings), findings)
	}
}

func TestCrossValidator_MixedValidAndInvalid(t *testing.T) {
	cv := &CrossValidator{}
	resources := []*ResourceInfo{
		// Valid service-pod match
		{
			Kind: "Service", Name: "good-svc", Namespace: "default",
			Raw: map[string]interface{}{
				"selectorLabels": map[string]string{"app": "good"},
			},
		},
		{
			Kind: "Pod", Name: "good-pod", Namespace: "default",
			Labels: map[string]string{"app": "good"},
		},
		// Invalid service (no matching pods)
		{
			Kind: "Service", Name: "bad-svc", Namespace: "default",
			Raw: map[string]interface{}{
				"selectorLabels": map[string]string{"app": "missing"},
			},
		},
		// Pod with missing ConfigMap
		{
			Kind: "Pod", Name: "ref-pod", Namespace: "default",
			Raw: map[string]interface{}{
				"configMapRefs": []string{"existing-cm", "missing-cm"},
			},
		},
		{Kind: "ConfigMap", Name: "existing-cm", Namespace: "default"},
		// Ingress with missing service
		{
			Kind: "Ingress", Name: "bad-ing", Namespace: "default",
			Raw: map[string]interface{}{
				"serviceRefs": []string{"ghost-svc"},
			},
		},
	}

	findings := cv.ValidateCross(context.Background(), resources)
	if len(findings) != 3 {
		t.Errorf("expected 3 findings for mixed valid/invalid, got %d: %+v", len(findings), findings)
	}
}

func TestCrossValidator_InterfaceSliceHandling(t *testing.T) {
	cv := &CrossValidator{}
	// Test with []interface{} instead of []string (common when unmarshaling JSON)
	resources := []*ResourceInfo{
		{
			Kind: "Pod", Name: "json-pod", Namespace: "default",
			Raw: map[string]interface{}{
				"configMapRefs": []interface{}{"cm-from-json"},
			},
		},
	}

	findings := cv.ValidateCross(context.Background(), resources)
	assertFindingExists(t, findings, SeverityWarning, "references ConfigMap 'cm-from-json' which does not exist")
}

func TestCrossValidator_InterfaceMapHandling(t *testing.T) {
	cv := &CrossValidator{}
	// Test with map[string]interface{} instead of map[string]string for selector labels
	resources := []*ResourceInfo{
		{
			Kind: "Service", Name: "json-svc", Namespace: "default",
			Raw: map[string]interface{}{
				"selectorLabels": map[string]interface{}{"app": "web"},
			},
		},
	}

	findings := cv.ValidateCross(context.Background(), resources)
	// No pods → should find mismatch
	assertFindingExists(t, findings, SeverityWarning, "selector does not match any pods")
}

func TestCrossValidator_FindingsHaveRequiredFields(t *testing.T) {
	cv := &CrossValidator{}
	resources := []*ResourceInfo{
		{
			Kind: "Service", Name: "no-match-svc", Namespace: "test-ns",
			Raw: map[string]interface{}{
				"selectorLabels": map[string]string{"app": "ghost"},
			},
		},
		{
			Kind: "Pod", Name: "ref-pod", Namespace: "test-ns",
			Raw: map[string]interface{}{
				"configMapRefs": []string{"missing-cm"},
			},
		},
		{
			Kind: "Ingress", Name: "bad-ing", Namespace: "test-ns",
			Raw: map[string]interface{}{
				"serviceRefs": []string{"missing-svc"},
			},
		},
		{
			Kind: "HorizontalPodAutoscaler", Name: "bad-hpa", Namespace: "test-ns",
			Raw: map[string]interface{}{
				"targetRef": "Deployment/missing-deploy",
			},
		},
	}

	findings := cv.ValidateCross(context.Background(), resources)
	if len(findings) < 4 {
		t.Fatalf("expected at least 4 findings, got %d", len(findings))
	}

	for i, f := range findings {
		if f.Analyzer == "" {
			t.Errorf("finding %d: missing Analyzer field", i)
		}
		if f.Analyzer != "cross-validator" {
			t.Errorf("finding %d: expected analyzer 'cross-validator', got %q", i, f.Analyzer)
		}
		if f.Resource == "" {
			t.Errorf("finding %d: missing Resource field", i)
		}
		if f.Severity == "" {
			t.Errorf("finding %d: missing Severity field", i)
		}
		if f.Title == "" {
			t.Errorf("finding %d: missing Title field", i)
		}
		if f.Details == "" {
			t.Errorf("finding %d: missing Details field", i)
		}
		if len(f.Suggestions) == 0 {
			t.Errorf("finding %d: missing Suggestions", i)
		}
	}
}

func TestCrossValidator_RegistryValidateCross(t *testing.T) {
	r := DefaultRegistry()
	resources := []*ResourceInfo{
		{
			Kind: "Service", Name: "orphan-svc", Namespace: "default",
			Raw: map[string]interface{}{
				"selectorLabels": map[string]string{"app": "nothing"},
			},
		},
	}

	findings := r.ValidateCross(context.Background(), resources)
	if len(findings) != 1 {
		t.Errorf("expected 1 finding from registry ValidateCross, got %d", len(findings))
	}
}
