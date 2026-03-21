package ui

import "testing"

func TestParseApprovedKubectlCommandAcceptsPlainKubectlArgs(t *testing.T) {
	args, err := parseApprovedKubectlCommand(`kubectl get pods -n kube-system -o jsonpath='{.items[*].metadata.name}'`)
	if err != nil {
		t.Fatalf("expected valid kubectl command, got error: %v", err)
	}
	if len(args) == 0 {
		t.Fatal("expected parsed kubectl args")
	}
	if args[0] != "get" {
		t.Fatalf("expected first kubectl arg to be get, got %q", args[0])
	}
}

func TestParseApprovedKubectlCommandRejectsShellFeatures(t *testing.T) {
	tests := []string{
		"kubectl get pods | head",
		"kubectl get pods && kubectl get svc",
		"kubectl get pods > pods.txt",
	}

	for _, command := range tests {
		if _, err := parseApprovedKubectlCommand(command); err == nil {
			t.Fatalf("expected shell-feature command %q to be rejected", command)
		}
	}
}

func TestParseApprovedKubectlCommandRejectsNonKubectlPrograms(t *testing.T) {
	if _, err := parseApprovedKubectlCommand("bash -c 'kubectl get pods'"); err == nil {
		t.Fatal("expected non-kubectl program to be rejected")
	}
}
