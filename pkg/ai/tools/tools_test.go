package tools

import (
	"strings"
	"testing"
)

func TestRegistryListReturnsSortedTools(t *testing.T) {
	registry := NewRegistry()
	registry.RegisterMCPTool("zeta_tool", "zeta", "server-a", map[string]interface{}{"type": "object"})
	registry.RegisterMCPTool("alpha_tool", "alpha", "server-a", map[string]interface{}{"type": "object"})

	got := registry.List()
	want := []string{"alpha_tool", "bash", "kubectl", "zeta_tool"}
	if len(got) != len(want) {
		t.Fatalf("List() returned %d tools, want %d", len(got), len(want))
	}
	for i, tool := range got {
		if tool.Name != want[i] {
			t.Fatalf("List()[%d] = %q, want %q", i, tool.Name, want[i])
		}
	}
}

func TestRegistryGetMCPToolsReturnsSortedTools(t *testing.T) {
	registry := NewRegistry()
	registry.RegisterMCPTool("pods_list", "pods", "server-a", map[string]interface{}{"type": "object"})
	registry.RegisterMCPTool("cluster_overview", "overview", "server-a", map[string]interface{}{"type": "object"})

	got := registry.GetMCPTools()
	want := []string{"cluster_overview", "pods_list"}
	if len(got) != len(want) {
		t.Fatalf("GetMCPTools() returned %d tools, want %d", len(got), len(want))
	}
	for i, tool := range got {
		if tool.Name != want[i] {
			t.Fatalf("GetMCPTools()[%d] = %q, want %q", i, tool.Name, want[i])
		}
	}
}

func TestValidateKubectlToolCommandBlocksUnsupportedInteractiveCommands(t *testing.T) {
	tests := []string{
		"kubectl edit deployment api",
		"kubectl port-forward pod/api 8080:80",
		"kubectl attach pod/api",
		"kubectl exec -it pod/api -- sh",
	}

	for _, command := range tests {
		if err := ValidateKubectlToolCommand(command); err == nil {
			t.Fatalf("ValidateKubectlToolCommand(%q) expected error", command)
		} else if !strings.Contains(err.Error(), "cannot be approved") {
			t.Fatalf("ValidateKubectlToolCommand(%q) error = %q, want cannot be approved guidance", command, err.Error())
		}
	}
}

func TestValidateBashToolCommandBlocksKubernetesBypass(t *testing.T) {
	tests := []string{
		"kubectl get pods -A",
		"helm list -A",
	}

	for _, command := range tests {
		if err := ValidateBashToolCommand(command); err == nil {
			t.Fatalf("ValidateBashToolCommand(%q) expected error", command)
		} else if !strings.Contains(err.Error(), "cannot be approved") {
			t.Fatalf("ValidateBashToolCommand(%q) error = %q, want cannot be approved guidance", command, err.Error())
		}
	}
}
