package server

import (
	"testing"
)

func TestDefaultTools(t *testing.T) {
	tools := DefaultTools()

	// Should have 6 default tools
	expectedTools := []string{
		"kubectl",
		"bash",
		"kubectl_get",
		"kubectl_describe",
		"kubectl_logs",
		"kubectl_apply",
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("DefaultTools() returned %d tools, want %d", len(tools), len(expectedTools))
	}

	toolMap := make(map[string]*Tool)
	for _, tool := range tools {
		toolMap[tool.Name] = tool
	}

	for _, name := range expectedTools {
		tool, ok := toolMap[name]
		if !ok {
			t.Errorf("missing tool: %s", name)
			continue
		}

		// Verify tool has required fields
		if tool.Description == "" {
			t.Errorf("tool %s has no description", name)
		}
		if tool.InputSchema == nil {
			t.Errorf("tool %s has no input schema", name)
		}
		if tool.Handler == nil {
			t.Errorf("tool %s has no handler", name)
		}
	}
}

func TestKubectlTool(t *testing.T) {
	tool := KubectlTool()

	if tool.Name != "kubectl" {
		t.Errorf("Name = %q, want kubectl", tool.Name)
	}

	// Check input schema
	props, ok := tool.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("InputSchema properties not found")
	}

	if _, ok := props["command"]; !ok {
		t.Error("command property not found in InputSchema")
	}

	// Check required field
	required, ok := tool.InputSchema["required"].([]string)
	if !ok || len(required) == 0 {
		t.Error("required field not properly set")
	}
}

func TestBashTool(t *testing.T) {
	tool := BashTool()

	if tool.Name != "bash" {
		t.Errorf("Name = %q, want bash", tool.Name)
	}

	props, ok := tool.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("InputSchema properties not found")
	}

	if _, ok := props["command"]; !ok {
		t.Error("command property not found")
	}
	if _, ok := props["timeout"]; !ok {
		t.Error("timeout property not found")
	}
}

func TestKubectlGetTool(t *testing.T) {
	tool := KubectlGetTool()

	if tool.Name != "kubectl_get" {
		t.Errorf("Name = %q, want kubectl_get", tool.Name)
	}

	props, ok := tool.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("InputSchema properties not found")
	}

	requiredFields := []string{"resource", "name", "namespace", "output", "selector"}
	for _, field := range requiredFields {
		if _, ok := props[field]; !ok {
			t.Errorf("%s property not found", field)
		}
	}
}

func TestKubectlDescribeTool(t *testing.T) {
	tool := KubectlDescribeTool()

	if tool.Name != "kubectl_describe" {
		t.Errorf("Name = %q, want kubectl_describe", tool.Name)
	}

	required, ok := tool.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("required field not found")
	}

	// resource and name are required
	hasResource, hasName := false, false
	for _, r := range required {
		if r == "resource" {
			hasResource = true
		}
		if r == "name" {
			hasName = true
		}
	}

	if !hasResource {
		t.Error("resource should be required")
	}
	if !hasName {
		t.Error("name should be required")
	}
}

func TestKubectlLogsTool(t *testing.T) {
	tool := KubectlLogsTool()

	if tool.Name != "kubectl_logs" {
		t.Errorf("Name = %q, want kubectl_logs", tool.Name)
	}

	props, ok := tool.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("InputSchema properties not found")
	}

	optionalFields := []string{"pod", "namespace", "container", "tail", "previous"}
	for _, field := range optionalFields {
		if _, ok := props[field]; !ok {
			t.Errorf("%s property not found", field)
		}
	}
}

func TestKubectlApplyTool(t *testing.T) {
	tool := KubectlApplyTool()

	if tool.Name != "kubectl_apply" {
		t.Errorf("Name = %q, want kubectl_apply", tool.Name)
	}

	props, ok := tool.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("InputSchema properties not found")
	}

	// manifest property should exist
	if _, ok := props["manifest"]; !ok {
		t.Error("manifest property not found")
	}

	// dry_run property should exist
	if _, ok := props["dry_run"]; !ok {
		t.Error("dry_run property not found")
	}
}

func TestToolSchemaType(t *testing.T) {
	tools := DefaultTools()

	for _, tool := range tools {
		schemaType, ok := tool.InputSchema["type"].(string)
		if !ok {
			t.Errorf("tool %s: schema type is not string", tool.Name)
			continue
		}
		if schemaType != "object" {
			t.Errorf("tool %s: schema type = %q, want object", tool.Name, schemaType)
		}
	}
}
