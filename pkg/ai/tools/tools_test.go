package tools

import (
	"fmt"
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

func TestResolveKubectlPathWithUsesOverride(t *testing.T) {
	t.Parallel()

	got, err := resolveKubectlPathWith("/custom/kubectl", func(candidate string) (string, error) {
		if candidate == "/custom/kubectl" {
			return candidate, nil
		}
		return "", fmt.Errorf("unexpected lookup: %s", candidate)
	})
	if err != nil {
		t.Fatalf("resolveKubectlPathWith() error = %v", err)
	}
	if got != "/custom/kubectl" {
		t.Fatalf("resolveKubectlPathWith() = %q, want /custom/kubectl", got)
	}
}

func TestResolveKubectlPathWithFallsBackToCommonLocations(t *testing.T) {
	t.Parallel()

	got, err := resolveKubectlPathWith("", func(candidate string) (string, error) {
		switch candidate {
		case "kubectl":
			return "", fmt.Errorf("not in PATH")
		case "/usr/bin/kubectl":
			return candidate, nil
		default:
			return "", fmt.Errorf("not found: %s", candidate)
		}
	})
	if err != nil {
		t.Fatalf("resolveKubectlPathWith() error = %v", err)
	}
	if got != "/usr/bin/kubectl" {
		t.Fatalf("resolveKubectlPathWith() = %q, want /usr/bin/kubectl", got)
	}
}

func TestResolveKubectlPathWithReturnsHelpfulError(t *testing.T) {
	t.Parallel()

	_, err := resolveKubectlPathWith("", func(candidate string) (string, error) {
		return "", fmt.Errorf("not found: %s", candidate)
	})
	if err == nil {
		t.Fatal("resolveKubectlPathWith() expected error")
	}
	if !strings.Contains(err.Error(), kubectlPathEnvVar) {
		t.Fatalf("error = %q, want env var hint", err)
	}
	if !strings.Contains(err.Error(), "common locations") {
		t.Fatalf("error = %q, want common locations guidance", err)
	}
}
