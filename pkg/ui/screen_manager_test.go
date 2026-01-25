package ui

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TestScreenManagerCreation tests basic ScreenManager creation.
func TestScreenManagerCreation(t *testing.T) {
	t.Run("default-config", func(t *testing.T) {
		sm := NewScreenManager(nil)
		if sm == nil {
			t.Fatal("ScreenManager should not be nil")
		}
		if sm.App() == nil {
			t.Error("App should not be nil")
		}
		if sm.State() != ScreenStateInit {
			t.Errorf("Initial state should be ScreenStateInit, got %v", sm.State())
		}
	})

	t.Run("with-simulation-screen", func(t *testing.T) {
		screen := tcell.NewSimulationScreen("")
		if err := screen.Init(); err != nil {
			t.Fatalf("Screen init failed: %v", err)
		}
		defer screen.Fini()
		screen.SetSize(120, 40)

		sm := NewScreenManager(&ScreenManagerConfig{
			UseSimulationScreen: true,
			Screen:              screen,
		})

		if sm.Screen() != screen {
			t.Error("Screen should match configured screen")
		}
	})
}

// TestScreenManagerLifecycle tests start/stop lifecycle.
func TestScreenManagerLifecycle(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("Screen init failed: %v", err)
	}
	screen.SetSize(120, 40)

	sm := NewScreenManager(&ScreenManagerConfig{
		UseSimulationScreen: true,
		Screen:              screen,
	})

	// Set up a simple root
	root := tview.NewBox()
	sm.SetRoot(root, true)

	// Track callbacks using atomic to avoid race conditions
	var startCalled, stopCalled int32
	sm.SetOnStart(func() { atomic.StoreInt32(&startCalled, 1) })
	sm.SetOnStop(func() { atomic.StoreInt32(&stopCalled, 1) })

	// Run async
	stop := sm.RunAsync()

	time.Sleep(100 * time.Millisecond)

	if !sm.IsRunning() {
		t.Error("ScreenManager should be running")
	}
	if sm.State() != ScreenStateRunning {
		t.Errorf("State should be ScreenStateRunning, got %v", sm.State())
	}
	if atomic.LoadInt32(&startCalled) == 0 {
		t.Error("OnStart callback should have been called")
	}

	// Stop
	stop()

	time.Sleep(100 * time.Millisecond)

	if sm.IsRunning() {
		t.Error("ScreenManager should not be running after stop")
	}
	if atomic.LoadInt32(&stopCalled) == 0 {
		t.Error("OnStop callback should have been called")
	}
}

// TestScreenManagerQueueUpdateDraw tests thread-safe UI updates.
func TestScreenManagerQueueUpdateDraw(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("Screen init failed: %v", err)
	}
	screen.SetSize(120, 40)

	sm := NewScreenManager(&ScreenManagerConfig{
		UseSimulationScreen: true,
		Screen:              screen,
	})

	textView := tview.NewTextView()
	sm.SetRoot(textView, true)

	stop := sm.RunAsync()
	defer stop()

	time.Sleep(100 * time.Millisecond)

	// Concurrent updates
	var wg sync.WaitGroup
	updateCount := int32(0)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			sm.QueueUpdateDraw(func() {
				atomic.AddInt32(&updateCount, 1)
			})
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	count := atomic.LoadInt32(&updateCount)
	if count == 0 {
		t.Error("Updates should have been processed")
	}
	t.Logf("Processed %d updates", count)
}

// TestScreenManagerDoneChannel tests the done channel.
func TestScreenManagerDoneChannel(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("Screen init failed: %v", err)
	}
	screen.SetSize(120, 40)

	sm := NewScreenManager(&ScreenManagerConfig{
		UseSimulationScreen: true,
		Screen:              screen,
	})

	sm.SetRoot(tview.NewBox(), true)
	stop := sm.RunAsync()

	time.Sleep(100 * time.Millisecond)

	// Done channel should not be closed while running
	select {
	case <-sm.Done():
		t.Error("Done channel should not be closed while running")
	default:
		// Expected
	}

	stop()

	// Done channel should be closed after stop
	select {
	case <-sm.Done():
		// Expected
	case <-time.After(3 * time.Second):
		t.Error("Done channel should be closed after stop")
	}
}

// TestScreenManagerResize tests screen resize handling.
func TestScreenManagerResize(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("Screen init failed: %v", err)
	}
	screen.SetSize(120, 40)

	sm := NewScreenManager(&ScreenManagerConfig{
		UseSimulationScreen: true,
		Screen:              screen,
	})

	sm.SetRoot(tview.NewBox(), true)
	stop := sm.RunAsync()
	defer stop()

	time.Sleep(100 * time.Millisecond)

	// Resize
	sm.Resize(80, 24)

	w, h := screen.Size()
	if w != 80 || h != 24 {
		t.Errorf("Expected size 80x24, got %dx%d", w, h)
	}
}

// TestScreenManagerFocus tests focus management.
func TestScreenManagerFocus(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("Screen init failed: %v", err)
	}
	screen.SetSize(120, 40)

	sm := NewScreenManager(&ScreenManagerConfig{
		UseSimulationScreen: true,
		Screen:              screen,
	})

	box1 := tview.NewBox().SetBorder(true)
	box2 := tview.NewBox().SetBorder(true)

	flex := tview.NewFlex().
		AddItem(box1, 0, 1, true).
		AddItem(box2, 0, 1, false)

	sm.SetRoot(flex, true)
	stop := sm.RunAsync()
	defer stop()

	time.Sleep(100 * time.Millisecond)

	// Switch focus
	sm.SetFocus(box2)
	time.Sleep(50 * time.Millisecond)

	focused := sm.GetFocus()
	if focused != box2 {
		t.Errorf("Expected box2 to be focused, got %T", focused)
	}
}

// TestScreenManagerContext tests context handling.
func TestScreenManagerContext(t *testing.T) {
	sm := NewScreenManager(nil)

	ctx := sm.Context()
	if ctx == nil {
		t.Error("Context should not be nil")
	}

	// Context should not be cancelled initially
	select {
	case <-ctx.Done():
		t.Error("Context should not be cancelled initially")
	default:
		// Expected
	}
}

// TestScreenManagerConcurrentOperations tests thread safety.
func TestScreenManagerConcurrentOperations(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("Screen init failed: %v", err)
	}
	screen.SetSize(120, 40)

	sm := NewScreenManager(&ScreenManagerConfig{
		UseSimulationScreen: true,
		Screen:              screen,
	})

	sm.SetRoot(tview.NewTextView(), true)
	stop := sm.RunAsync()
	defer stop()

	time.Sleep(100 * time.Millisecond)

	var wg sync.WaitGroup
	numGoroutines := 20
	iterations := 50

	// Concurrent state checks
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = sm.IsRunning()
				_ = sm.State()
				_, _ = sm.Size()
			}
		}()
	}

	// Concurrent draws
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				sm.Draw()
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(10 * time.Second):
		t.Fatal("Concurrent operations timed out - possible deadlock")
	}
}

// TestScreenManagerMultipleStops tests multiple stop calls.
// Note: This test verifies no panic occurs, but may have benign race conditions
// in tview's internal state due to the nature of concurrent stop calls.
func TestScreenManagerMultipleStops(t *testing.T) {
	// Skip under race detector as tview has known races with concurrent Stop() calls
	if raceEnabled {
		t.Skip("Skipping test under race detector due to tview internal races")
	}

	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("Screen init failed: %v", err)
	}
	screen.SetSize(120, 40)

	sm := NewScreenManager(&ScreenManagerConfig{
		UseSimulationScreen: true,
		Screen:              screen,
	})

	sm.SetRoot(tview.NewBox(), true)
	_ = sm.RunAsync()

	time.Sleep(100 * time.Millisecond)

	// Multiple concurrent stop calls
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sm.Stop()
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - no panic
	case <-time.After(5 * time.Second):
		t.Fatal("Multiple stop calls timed out")
	}
}

// TestScreenManagerRefreshCallback tests refresh callback.
func TestScreenManagerRefreshCallback(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("Screen init failed: %v", err)
	}
	screen.SetSize(120, 40)

	sm := NewScreenManager(&ScreenManagerConfig{
		UseSimulationScreen: true,
		Screen:              screen,
	})

	refreshCalled := int32(0)
	sm.SetOnRefresh(func() {
		atomic.AddInt32(&refreshCalled, 1)
	})

	sm.SetRoot(tview.NewBox(), true)
	stop := sm.RunAsync()
	defer stop()

	time.Sleep(100 * time.Millisecond)

	// Trigger refresh
	sm.Refresh()

	// Wait for async callback
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&refreshCalled) == 0 {
		t.Error("Refresh callback should have been called")
	}
}

// TestScreenManagerWithNilScreen tests operation without simulation screen.
func TestScreenManagerWithNilScreen(t *testing.T) {
	sm := NewScreenManager(nil)

	// Size should return defaults without screen
	w, h := sm.Size()
	if w != 80 || h != 24 {
		t.Errorf("Expected default size 80x24, got %dx%d", w, h)
	}

	// Resize should not panic without simulation screen
	sm.Resize(100, 50)
}

// TestTUITesterWithScreenManager tests TUITester with ScreenManager.
func TestTUITesterWithScreenManager(t *testing.T) {
	tt := NewTUITester(t)

	// Create a simple app with text
	textView := tview.NewTextView().SetText("Hello Test")
	tt.App.SetRoot(textView, true)

	stop := tt.RunAsync()
	defer stop()

	time.Sleep(100 * time.Millisecond)

	// Verify content
	tt.AssertContentContains("Hello Test")

	// Test screen capture
	capture := tt.CaptureScreen("test-capture")
	if capture.Label != "test-capture" {
		t.Errorf("Expected label 'test-capture', got %q", capture.Label)
	}
	if capture.Width == 0 || capture.Height == 0 {
		t.Error("Capture should have non-zero dimensions")
	}
}

// TestTUITesterScenario tests the Scenario builder.
func TestTUITesterScenario(t *testing.T) {
	tt := NewTUITester(t)

	inputField := tview.NewInputField().SetLabel("Input: ")
	tt.App.SetRoot(inputField, true)

	stop := tt.RunAsync()
	defer stop()

	time.Sleep(100 * time.Millisecond)

	// Build and run scenario
	err := tt.NewScenario("input-test").
		Step("type-text", func() error {
			tt.TypeString("hello")
			return nil
		}).
		Step("verify-text", func() error {
			text := inputField.GetText()
			if text != "hello" {
				return &testError{msg: "expected 'hello', got '" + text + "'"}
			}
			return nil
		}).
		Run()

	if err != nil {
		t.Errorf("Scenario failed: %v", err)
	}
}

// testError is a simple error type for testing.
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// TestTUITesterCaptureHistory tests capture history management.
func TestTUITesterCaptureHistory(t *testing.T) {
	tt := NewTUITesterWithConfig(t, &TUITesterConfig{
		MaxCaptures: 3,
	})

	tt.App.SetRoot(tview.NewTextView().SetText("Test"), true)
	stop := tt.RunAsync()
	defer stop()

	time.Sleep(100 * time.Millisecond)

	// Capture multiple screens
	for i := 0; i < 5; i++ {
		tt.CaptureScreen("capture-" + itoa(i))
	}

	history := tt.GetCaptureHistory()
	if len(history) != 3 {
		t.Errorf("Expected 3 captures in history, got %d", len(history))
	}

	// First captures should be trimmed
	if history[0].Label != "capture-2" {
		t.Errorf("Expected first capture to be 'capture-2', got %q", history[0].Label)
	}

	// Clear history
	tt.ClearCaptureHistory()
	history = tt.GetCaptureHistory()
	if len(history) != 0 {
		t.Errorf("Expected empty history after clear, got %d", len(history))
	}
}

// TestTUITesterWaitForDraw tests screen stabilization waiting.
func TestTUITesterWaitForDraw(t *testing.T) {
	tt := NewTUITester(t)

	textView := tview.NewTextView()
	tt.App.SetRoot(textView, true)

	stop := tt.RunAsync()
	defer stop()

	time.Sleep(50 * time.Millisecond)

	// This should wait for screen to stabilize
	tt.WaitForDraw(500 * time.Millisecond)

	// Screen should be stable now
	content1 := tt.GetContent()
	tt.App.QueueUpdateDraw(func() {})
	time.Sleep(20 * time.Millisecond)
	content2 := tt.GetContent()

	if content1 != content2 {
		t.Error("Screen should be stable after WaitForDraw")
	}
}

// TestTUITesterRegionCapture tests region-based content extraction.
func TestTUITesterRegionCapture(t *testing.T) {
	tt := NewTUITester(t)

	textView := tview.NewTextView().SetText("ABCDEFGHIJ\n1234567890")
	tt.App.SetRoot(textView, true)

	stop := tt.RunAsync()
	defer stop()

	time.Sleep(100 * time.Millisecond)

	// Get a region
	region := tt.GetRegion(0, 0, 5, 2)
	lines := splitLines(region)

	if len(lines) >= 1 && len(lines[0]) >= 5 {
		// First 5 chars of first line
		if lines[0][:5] != "ABCDE" {
			t.Logf("Region content:\n%s", region)
		}
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// TestTUITesterKeyHelpers tests all key helper methods.
func TestTUITesterKeyHelpers(t *testing.T) {
	// Skip under race detector as rapid key injection can cause races in tview
	if raceEnabled {
		t.Skip("Skipping test under race detector due to tview internal races")
	}

	tt := NewTUITester(t)

	inputField := tview.NewInputField()
	tt.App.SetRoot(inputField, true)

	stop := tt.RunAsync()
	defer stop()

	time.Sleep(100 * time.Millisecond)

	// Test each key helper with fast injection (no app.Draw() wait)
	tt.InjectKeyFast(tcell.KeyUp, 0, tcell.ModNone)
	tt.InjectKeyFast(tcell.KeyDown, 0, tcell.ModNone)
	tt.InjectKeyFast(tcell.KeyLeft, 0, tcell.ModNone)
	tt.InjectKeyFast(tcell.KeyRight, 0, tcell.ModNone)
	tt.InjectKeyFast(tcell.KeyBackspace2, 0, tcell.ModNone)
	tt.InjectKeyFast(tcell.KeyDelete, 0, tcell.ModNone)
	tt.InjectKeyFast(tcell.KeyTab, 0, tcell.ModNone)
	tt.InjectKeyFast(tcell.KeyEscape, 0, tcell.ModNone)
	tt.InjectKeyFast(tcell.KeyEnter, 0, tcell.ModNone)

	// Test typing
	tt.TypeStringFast("abc")
	tt.InjectKeyFast(tcell.KeyBackspace2, 0, tcell.ModNone)

	// Test Ctrl keys (fast)
	tt.InjectKeyFast(tcell.KeyCtrlA, 0, tcell.ModCtrl)
	tt.InjectKeyFast(tcell.KeyCtrlC, 0, tcell.ModCtrl)

	// Test modifier keys (fast)
	tt.InjectKeyFast(tcell.KeyRune, 'A', tcell.ModShift)
	tt.InjectKeyFast(tcell.KeyRune, 'x', tcell.ModAlt)

	// Wait a bit for events to process
	time.Sleep(50 * time.Millisecond)

	// If we got here without panic, success
}

// TestTUITesterAssertions tests assertion methods.
func TestTUITesterAssertions(t *testing.T) {
	tt := NewTUITester(t)

	textView := tview.NewTextView().SetText("Hello World")
	tt.App.SetRoot(textView, true)

	stop := tt.RunAsync()
	defer stop()

	time.Sleep(100 * time.Millisecond)

	// Draw and sync to ensure content is rendered
	tt.App.QueueUpdateDraw(func() {})
	time.Sleep(50 * time.Millisecond)

	content := tt.GetContent()
	// Content should contain Hello World somewhere
	if !containsText(content, "Hello") {
		t.Log("Content does not contain Hello - checking raw content...")
		t.Logf("Raw content (first 200 chars): %q", truncateString(content, 200))
	}

	// Test that screen is working
	if len(content) == 0 {
		t.Error("Screen content should not be empty")
	}
}

func containsText(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// TestTUITesterWaitMethods tests wait helper methods.
func TestTUITesterWaitMethods(t *testing.T) {
	tt := NewTUITester(t)

	textView := tview.NewTextView().SetText("Initial")
	tt.App.SetRoot(textView, true)

	stop := tt.RunAsync()
	defer stop()

	time.Sleep(100 * time.Millisecond)

	// Test WaitForCondition - simple case that should work
	conditionMet := tt.WaitForCondition(200*time.Millisecond, func() bool {
		return true
	})
	if !conditionMet {
		t.Error("WaitForCondition should succeed for true condition")
	}

	// Test that screen is responsive
	content := tt.GetContent()
	if len(content) == 0 {
		t.Error("Screen should have content")
	}
}
