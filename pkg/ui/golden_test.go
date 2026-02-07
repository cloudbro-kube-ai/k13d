package ui

import (
	"os"
	"testing"
	"time"
)

// ============================================================================
// Golden File Tests
// Capture screen state and compare against testdata/*.golden
// Run with -update flag to regenerate: go test -run TestGolden -update ./pkg/ui/...
//
// Note: These tests are skipped in CI due to timing-dependent screen rendering.
// The screen may show "Loading..." instead of actual content in headless CI.
// ============================================================================

func isCI() bool {
	return os.Getenv("CI") == "true"
}

func TestGolden_StartupScreen(t *testing.T) {
	if isCI() {
		t.Skip("Skipping golden test in CI - screen timing differs in headless mode")
	}
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.Wait(200 * time.Millisecond).
		ExpectGolden("startup")
}

func TestGolden_HelpModal(t *testing.T) {
	if isCI() {
		t.Skip("Skipping golden test in CI - screen timing differs in headless mode")
	}
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.PressRune('?').
		Wait(200 * time.Millisecond).
		ExpectGolden("help-modal")
}

func TestGolden_CommandMode(t *testing.T) {
	if isCI() {
		t.Skip("Skipping golden test in CI - screen timing differs in headless mode")
	}
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.PressRune(':').
		Wait(100 * time.Millisecond).
		ExpectGolden("command-mode")
}

func TestGolden_FilterMode(t *testing.T) {
	if isCI() {
		t.Skip("Skipping golden test in CI - screen timing differs in headless mode")
	}
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.PressRune('/').
		Wait(100 * time.Millisecond).
		ExpectGolden("filter-mode")
}
