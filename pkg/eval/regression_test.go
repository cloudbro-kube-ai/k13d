package eval

import (
	"strings"
	"testing"
)

// TestFormatMessageInterpolates guards the fix where formatMessage returned the
// raw format string (literal %v) instead of interpolating args.
func TestFormatMessageInterpolates(t *testing.T) {
	got := formatMessage("task timed out after %ds (code %d)", 30, 7)
	want := "task timed out after 30s (code 7)"
	if got != want {
		t.Errorf("formatMessage() = %q, want %q", got, want)
	}
	if got := formatMessage("plain message"); got != "plain message" {
		t.Errorf("formatMessage(no args) = %q, want %q", got, "plain message")
	}
}

// TestAddFailureFormats ensures failures recorded via AddFailure are formatted,
// not left as literal format verbs.
func TestAddFailureFormats(t *testing.T) {
	r := &TaskResult{}
	r.AddFailure("invalid value %q at index %d", "x", 4)
	if len(r.Failures) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(r.Failures))
	}
	msg := r.Failures[0].Message
	if strings.Contains(msg, "%q") || strings.Contains(msg, "%d") {
		t.Errorf("failure still contains raw verbs: %q", msg)
	}
	if msg != `invalid value "x" at index 4` {
		t.Errorf("failure message = %q, want interpolated", msg)
	}
}
