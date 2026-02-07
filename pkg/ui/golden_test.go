package ui

import (
	"testing"
	"time"
)

// ============================================================================
// Golden File Tests
// Capture screen state and compare against testdata/*.golden
// Run with -update flag to regenerate: go test -run TestGolden -update ./pkg/ui/...
// ============================================================================

func TestGolden_StartupScreen(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.Wait(200 * time.Millisecond).
		ExpectGolden("startup")
}

func TestGolden_HelpModal(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.PressRune('?').
		Wait(200 * time.Millisecond).
		ExpectGolden("help-modal")
}

func TestGolden_CommandMode(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.PressRune(':').
		Wait(100 * time.Millisecond).
		ExpectGolden("command-mode")
}

func TestGolden_FilterMode(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.PressRune('/').
		Wait(100 * time.Millisecond).
		ExpectGolden("filter-mode")
}
