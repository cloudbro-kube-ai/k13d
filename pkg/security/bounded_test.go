package security

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// TestRunScanCommandBounded verifies normal stdout capture and that a failing
// command surfaces its stderr in the error (guards the bounded-output helper
// added to cap trivy/kube-bench memory use).
func TestRunScanCommandBounded(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh")
	}

	// Normal capture.
	out, err := runScanCommandBounded(exec.Command("sh", "-c", "printf 'hello world'"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "hello world" {
		t.Errorf("stdout = %q, want %q", string(out), "hello world")
	}

	// Failing command: stderr should be folded into the error.
	_, err = runScanCommandBounded(exec.Command("sh", "-c", "echo boom >&2; exit 3"))
	if err == nil {
		t.Fatal("expected error from failing command")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error should contain stderr 'boom', got: %v", err)
	}
}
