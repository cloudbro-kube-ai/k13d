package tools

import "testing"

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
