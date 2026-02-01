package ui

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ============================================================================
// Modern TUI Test Framework
// Inspired by Microsoft's tui-test patterns
// ============================================================================

// TUITestContext provides a complete testing environment for TUI tests.
// This encapsulates common setup/teardown and provides fluent assertions.
type TUITestContext struct {
	t       *testing.T
	app     *App
	screen  tcell.SimulationScreen
	done    chan struct{}
	timeout time.Duration
}

// NewTUITestContext creates a new test context with sensible defaults.
func NewTUITestContext(t *testing.T) *TUITestContext {
	t.Helper()
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("Failed to init screen: %v", err)
	}
	screen.SetSize(120, 40)

	app := NewTestApp(TestAppConfig{
		UseSimulationScreen:   true,
		Screen:                screen,
		SkipBackgroundLoading: true,
	})

	ctx := &TUITestContext{
		t:       t,
		app:     app,
		screen:  screen,
		done:    make(chan struct{}),
		timeout: 5 * time.Second,
	}

	// Start app in background
	go func() {
		_ = app.Run()
		close(ctx.done)
	}()

	// Wait for initial render
	time.Sleep(100 * time.Millisecond)

	return ctx
}

// Cleanup stops the app and cleans up resources. Call with defer.
func (ctx *TUITestContext) Cleanup() {
	ctx.app.Stop()
	select {
	case <-ctx.done:
	case <-time.After(2 * time.Second):
		ctx.t.Log("Warning: App did not stop cleanly within timeout")
	}
}

// WithTimeout sets custom timeout for assertions.
func (ctx *TUITestContext) WithTimeout(d time.Duration) *TUITestContext {
	ctx.timeout = d
	return ctx
}

// ============================================================================
// Key Input Methods
// ============================================================================

// Type sends text as key presses.
func (ctx *TUITestContext) Type(text string) *TUITestContext {
	for _, r := range text {
		ctx.screen.InjectKey(tcell.KeyRune, r, tcell.ModNone)
		time.Sleep(10 * time.Millisecond)
	}
	return ctx
}

// Press sends a key press.
func (ctx *TUITestContext) Press(key tcell.Key) *TUITestContext {
	ctx.screen.InjectKey(key, 0, tcell.ModNone)
	time.Sleep(20 * time.Millisecond)
	return ctx
}

// PressRune sends a rune key press.
func (ctx *TUITestContext) PressRune(r rune) *TUITestContext {
	ctx.screen.InjectKey(tcell.KeyRune, r, tcell.ModNone)
	time.Sleep(20 * time.Millisecond)
	return ctx
}

// Submit sends text followed by Enter.
func (ctx *TUITestContext) Submit(text string) *TUITestContext {
	return ctx.Type(text).Press(tcell.KeyEnter)
}

// Command enters command mode and submits a command.
func (ctx *TUITestContext) Command(cmd string) *TUITestContext {
	return ctx.PressRune(':').Submit(cmd)
}

// Escape sends Escape key.
func (ctx *TUITestContext) Escape() *TUITestContext {
	return ctx.Press(tcell.KeyEscape)
}

// Tab sends Tab key.
func (ctx *TUITestContext) Tab() *TUITestContext {
	return ctx.Press(tcell.KeyTab)
}

// Wait waits for a duration.
func (ctx *TUITestContext) Wait(d time.Duration) *TUITestContext {
	time.Sleep(d)
	return ctx
}

// ============================================================================
// Assertions
// ============================================================================

// ExpectResource asserts the current resource type.
func (ctx *TUITestContext) ExpectResource(expected string) *TUITestContext {
	ctx.t.Helper()
	deadline := time.Now().Add(ctx.timeout)

	for time.Now().Before(deadline) {
		ctx.app.mx.RLock()
		current := ctx.app.currentResource
		ctx.app.mx.RUnlock()

		if current == expected {
			return ctx
		}
		time.Sleep(50 * time.Millisecond)
	}

	ctx.app.mx.RLock()
	actual := ctx.app.currentResource
	ctx.app.mx.RUnlock()
	ctx.t.Errorf("Expected resource %q, got %q", expected, actual)
	return ctx
}

// ExpectPage asserts that a page exists.
func (ctx *TUITestContext) ExpectPage(name string) *TUITestContext {
	ctx.t.Helper()
	deadline := time.Now().Add(ctx.timeout)

	for time.Now().Before(deadline) {
		has := false
		done := make(chan struct{})
		ctx.app.QueueUpdate(func() {
			has = ctx.app.pages.HasPage(name)
			close(done)
		})
		<-done

		if has {
			return ctx
		}
		time.Sleep(50 * time.Millisecond)
	}

	ctx.t.Errorf("Expected page %q to exist", name)
	return ctx
}

// ExpectFocus asserts that a specific primitive has focus.
func (ctx *TUITestContext) ExpectFocus(check func(tview.Primitive) bool) *TUITestContext {
	ctx.t.Helper()
	deadline := time.Now().Add(ctx.timeout)

	for time.Now().Before(deadline) {
		var focused tview.Primitive
		done := make(chan struct{})
		ctx.app.QueueUpdate(func() {
			focused = ctx.app.GetFocus()
			close(done)
		})
		<-done

		if check(focused) {
			return ctx
		}
		time.Sleep(50 * time.Millisecond)
	}

	ctx.t.Error("Focus assertion failed")
	return ctx
}

// ExpectNoFreeze asserts that the app is responsive.
func (ctx *TUITestContext) ExpectNoFreeze() *TUITestContext {
	ctx.t.Helper()
	done := make(chan struct{})
	go func() {
		ctx.app.Draw()
		close(done)
	}()

	select {
	case <-done:
		return ctx
	case <-time.After(ctx.timeout):
		ctx.t.Fatal("App is frozen - Draw() did not complete")
		return ctx
	}
}

// ============================================================================
// Modern Tests Using New Framework
// ============================================================================

func TestModern_ResourceNavigation(t *testing.T) {
	resources := []struct {
		cmd      string
		expected string
	}{
		{"pods", "pods"},
		{"deploy", "deployments"},
		{"svc", "services"},
		{"no", "nodes"},
		{"ns", "namespaces"},
	}

	for _, tc := range resources {
		t.Run(tc.cmd, func(t *testing.T) {
			ctx := NewTUITestContext(t)
			defer ctx.Cleanup()

			ctx.Command(tc.cmd).
				ExpectResource(tc.expected).
				ExpectNoFreeze()
		})
	}
}

func TestModern_KeyboardShortcuts(t *testing.T) {
	tests := []struct {
		name   string
		action func(*TUITestContext)
	}{
		{"filter mode", func(ctx *TUITestContext) { ctx.PressRune('/') }},
		{"command mode", func(ctx *TUITestContext) { ctx.PressRune(':') }},
		{"help modal", func(ctx *TUITestContext) { ctx.PressRune('?') }},
		{"vim j", func(ctx *TUITestContext) { ctx.PressRune('j') }},
		{"vim k", func(ctx *TUITestContext) { ctx.PressRune('k') }},
		{"tab switch", func(ctx *TUITestContext) { ctx.Tab() }},
		{"escape", func(ctx *TUITestContext) { ctx.Escape() }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := NewTUITestContext(t)
			defer ctx.Cleanup()

			tc.action(ctx)
			ctx.ExpectNoFreeze()
		})
	}
}

func TestModern_RapidKeyPress(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	// Rapid key presses shouldn't cause freeze
	for i := 0; i < 50; i++ {
		ctx.PressRune('j').PressRune('k')
	}
	ctx.ExpectNoFreeze()
}

func TestModern_CommandSequence(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.Command("pods").
		ExpectResource("pods").
		Command("deploy").
		ExpectResource("deployments").
		Command("svc").
		ExpectResource("services").
		ExpectNoFreeze()
}

func TestModern_HelpModal(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	// Open help
	ctx.PressRune('?').
		Wait(100 * time.Millisecond).
		ExpectPage("help").
		ExpectNoFreeze()

	// Close help
	ctx.Escape().
		Wait(100 * time.Millisecond).
		ExpectNoFreeze()
}

func TestModern_FilterMode(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.PressRune('/').
		Wait(50 * time.Millisecond).
		Type("test-filter").
		Press(tcell.KeyEnter).
		ExpectNoFreeze()
}

func TestModern_TabNavigation(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	// Tab through panels
	for i := 0; i < 5; i++ {
		ctx.Tab().Wait(50 * time.Millisecond)
	}
	ctx.ExpectNoFreeze()
}
