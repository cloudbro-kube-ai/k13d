package ui

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// updateGolden is a flag to update golden files instead of comparing.
// Usage: go test -run TestGolden ./pkg/ui/... -update
var updateGolden = flag.Bool("update", false, "update golden files")

// TUITester provides headless TUI testing capabilities using tcell's SimulationScreen.
// This allows automated testing of TUI interactions without a real terminal.
//
// Features:
// - Safe lifecycle management with proper cleanup
// - Thread-safe key injection and content reading
// - Screen capture and comparison helpers
// - Timeout-based assertions for async operations
// - Freeze/deadlock detection
type TUITester struct {
	t      testing.TB
	App    *tview.Application
	Screen tcell.SimulationScreen
	UIApp  *App // The k13d App instance under test

	mu      sync.Mutex
	running int32 // atomic: 1 if running
	closed  int32 // atomic: 1 if closed
	done    chan struct{}

	// Screen capture history for debugging
	captureHistory []ScreenCapture
	captureMx      sync.Mutex
	maxCaptures    int

	// Lifecycle hooks
	onClose func()
}

// ScreenCapture represents a captured screen state.
type ScreenCapture struct {
	Timestamp time.Time
	Content   string
	Width     int
	Height    int
	Label     string
}

// TUITesterConfig holds configuration for TUITester.
type TUITesterConfig struct {
	// Width sets the initial screen width (default: 120).
	Width int
	// Height sets the initial screen height (default: 40).
	Height int
	// MaxCaptures sets the maximum number of screen captures to keep (default: 10).
	MaxCaptures int
	// OnClose is called when the tester is closed.
	OnClose func()
}

// NewTUITester creates a new TUI test harness with SimulationScreen.
func NewTUITester(t testing.TB) *TUITester {
	return NewTUITesterWithConfig(t, nil)
}

// NewTUITesterWithConfig creates a new TUI test harness with custom configuration.
func NewTUITesterWithConfig(t testing.TB, cfg *TUITesterConfig) *TUITester {
	t.Helper()

	if cfg == nil {
		cfg = &TUITesterConfig{}
	}
	if cfg.Width == 0 {
		cfg.Width = 120
	}
	if cfg.Height == 0 {
		cfg.Height = 40
	}
	if cfg.MaxCaptures == 0 {
		cfg.MaxCaptures = 10
	}

	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("SimulationScreen init failed: %v", err)
	}
	screen.SetSize(cfg.Width, cfg.Height)

	tvApp := tview.NewApplication().SetScreen(screen)

	tt := &TUITester{
		t:              t,
		App:            tvApp,
		Screen:         screen,
		done:           make(chan struct{}),
		captureHistory: make([]ScreenCapture, 0, cfg.MaxCaptures),
		maxCaptures:    cfg.MaxCaptures,
		onClose:        cfg.OnClose,
	}

	// Register cleanup on test completion
	t.Cleanup(func() {
		tt.Close()
	})

	return tt
}

// SetSize changes the simulated terminal size.
func (tt *TUITester) SetSize(width, height int) {
	tt.Screen.SetSize(width, height)
}

// Size returns the current screen size.
func (tt *TUITester) Size() (width, height int) {
	return tt.Screen.Size()
}

// IsRunning returns true if the tester is currently running.
func (tt *TUITester) IsRunning() bool {
	return atomic.LoadInt32(&tt.running) == 1
}

// IsClosed returns true if the tester has been closed.
func (tt *TUITester) IsClosed() bool {
	return atomic.LoadInt32(&tt.closed) == 1
}

// Close cleans up the test harness.
// Safe to call multiple times.
func (tt *TUITester) Close() {
	if !atomic.CompareAndSwapInt32(&tt.closed, 0, 1) {
		return // Already closed
	}

	if atomic.LoadInt32(&tt.running) == 1 {
		tt.App.Stop()
		select {
		case <-tt.done:
		case <-time.After(2 * time.Second):
			// Timeout, but continue cleanup
		}
		atomic.StoreInt32(&tt.running, 0)
	}

	if tt.onClose != nil {
		tt.onClose()
	}
	// Note: tview.Application.Stop() calls screen.Fini() internally,
	// so we don't call tt.Screen.Fini() here to avoid double-close panic.
}

// RunAsync starts the tview application in a goroutine.
// Returns a stop function that should be called to terminate the app.
func (tt *TUITester) RunAsync() func() {
	if !atomic.CompareAndSwapInt32(&tt.running, 0, 1) {
		return func() {} // Already running
	}

	tt.mu.Lock()
	tt.done = make(chan struct{}) // Reset done channel
	tt.mu.Unlock()

	go func() {
		_ = tt.App.Run()
		tt.mu.Lock()
		if atomic.LoadInt32(&tt.closed) == 0 {
			select {
			case <-tt.done:
				// Already closed
			default:
				close(tt.done)
			}
		}
		tt.mu.Unlock()
		atomic.StoreInt32(&tt.running, 0)
	}()

	// Give the app time to initialize
	time.Sleep(50 * time.Millisecond)

	return func() {
		if atomic.LoadInt32(&tt.closed) == 1 {
			return
		}

		tt.App.Stop()
		select {
		case <-tt.done:
		case <-time.After(2 * time.Second):
			tt.t.Error("App did not stop within timeout")
		}
	}
}

// RunAsyncWithContext starts the app with context cancellation support.
func (tt *TUITester) RunAsyncWithContext(ctx context.Context) func() {
	stop := tt.RunAsync()

	go func() {
		<-ctx.Done()
		stop()
	}()

	return stop
}

// InjectKey sends a key event to the TUI.
func (tt *TUITester) InjectKey(key tcell.Key, r rune, mod tcell.ModMask) {
	tt.Screen.InjectKey(key, r, mod)
	// Allow event processing
	time.Sleep(20 * time.Millisecond)
	tt.App.QueueUpdateDraw(func() {})
}

// InjectKeyFast sends a key event without waiting (for rapid input).
func (tt *TUITester) InjectKeyFast(key tcell.Key, r rune, mod tcell.ModMask) {
	tt.Screen.InjectKey(key, r, mod)
}

// InjectKeyWithDelay sends a key event with a custom delay.
func (tt *TUITester) InjectKeyWithDelay(key tcell.Key, r rune, mod tcell.ModMask, delay time.Duration) {
	tt.Screen.InjectKey(key, r, mod)
	time.Sleep(delay)
	tt.App.QueueUpdateDraw(func() {})
}

// TypeString types a string character by character.
func (tt *TUITester) TypeString(s string) {
	for _, r := range s {
		tt.InjectKey(tcell.KeyRune, r, tcell.ModNone)
	}
}

// TypeStringFast types a string rapidly without per-key delays.
func (tt *TUITester) TypeStringFast(s string) {
	for _, r := range s {
		tt.InjectKeyFast(tcell.KeyRune, r, tcell.ModNone)
	}
	time.Sleep(20 * time.Millisecond)
	tt.App.QueueUpdateDraw(func() {})
}

// TypeStringWithDelay types a string with a custom per-key delay.
func (tt *TUITester) TypeStringWithDelay(s string, delay time.Duration) {
	for _, r := range s {
		tt.Screen.InjectKey(tcell.KeyRune, r, tcell.ModNone)
		time.Sleep(delay)
	}
	tt.App.QueueUpdateDraw(func() {})
}

// PressEnter sends Enter key.
func (tt *TUITester) PressEnter() {
	tt.InjectKey(tcell.KeyEnter, 0, tcell.ModNone)
}

// PressEsc sends Escape key.
func (tt *TUITester) PressEsc() {
	tt.InjectKey(tcell.KeyEscape, 0, tcell.ModNone)
}

// PressTab sends Tab key.
func (tt *TUITester) PressTab() {
	tt.InjectKey(tcell.KeyTab, 0, tcell.ModNone)
}

// PressBackspace sends Backspace key.
func (tt *TUITester) PressBackspace() {
	tt.InjectKey(tcell.KeyBackspace2, 0, tcell.ModNone)
}

// PressDelete sends Delete key.
func (tt *TUITester) PressDelete() {
	tt.InjectKey(tcell.KeyDelete, 0, tcell.ModNone)
}

// PressUp sends Up arrow key.
func (tt *TUITester) PressUp() {
	tt.InjectKey(tcell.KeyUp, 0, tcell.ModNone)
}

// PressDown sends Down arrow key.
func (tt *TUITester) PressDown() {
	tt.InjectKey(tcell.KeyDown, 0, tcell.ModNone)
}

// PressLeft sends Left arrow key.
func (tt *TUITester) PressLeft() {
	tt.InjectKey(tcell.KeyLeft, 0, tcell.ModNone)
}

// PressRight sends Right arrow key.
func (tt *TUITester) PressRight() {
	tt.InjectKey(tcell.KeyRight, 0, tcell.ModNone)
}

// PressKey sends a rune key (like 'j', 'k', 'q', etc.).
func (tt *TUITester) PressKey(r rune) {
	tt.InjectKey(tcell.KeyRune, r, tcell.ModNone)
}

// PressCtrl sends a Ctrl+key combination.
func (tt *TUITester) PressCtrl(r rune) {
	// Map common Ctrl combinations
	switch r {
	case 'a':
		tt.InjectKey(tcell.KeyCtrlA, 0, tcell.ModCtrl)
	case 'b':
		tt.InjectKey(tcell.KeyCtrlB, 0, tcell.ModCtrl)
	case 'c':
		tt.InjectKey(tcell.KeyCtrlC, 0, tcell.ModCtrl)
	case 'd':
		tt.InjectKey(tcell.KeyCtrlD, 0, tcell.ModCtrl)
	case 'e':
		tt.InjectKey(tcell.KeyCtrlE, 0, tcell.ModCtrl)
	case 'f':
		tt.InjectKey(tcell.KeyCtrlF, 0, tcell.ModCtrl)
	case 'h':
		tt.InjectKey(tcell.KeyCtrlH, 0, tcell.ModCtrl)
	case 'i':
		tt.InjectKey(tcell.KeyCtrlI, 0, tcell.ModCtrl)
	case 'k':
		tt.InjectKey(tcell.KeyCtrlK, 0, tcell.ModCtrl)
	case 'l':
		tt.InjectKey(tcell.KeyCtrlL, 0, tcell.ModCtrl)
	case 'u':
		tt.InjectKey(tcell.KeyCtrlU, 0, tcell.ModCtrl)
	default:
		tt.InjectKey(tcell.KeyRune, r, tcell.ModCtrl)
	}
}

// PressShift sends a Shift+key combination.
func (tt *TUITester) PressShift(r rune) {
	tt.InjectKey(tcell.KeyRune, r, tcell.ModShift)
}

// PressAlt sends an Alt+key combination.
func (tt *TUITester) PressAlt(r rune) {
	tt.InjectKey(tcell.KeyRune, r, tcell.ModAlt)
}

// GetContent extracts the current screen content as text.
func (tt *TUITester) GetContent() string {
	tt.Screen.Sync()
	w, h := tt.Screen.Size()
	var sb strings.Builder

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			str, _, _ := tt.Screen.Get(x, y)
			if len(str) == 0 {
				sb.WriteRune(' ')
			} else {
				sb.WriteString(str)
			}
		}
		sb.WriteRune('\n')
	}
	return sb.String()
}

// GetContentTrimmed returns screen content with trailing whitespace removed from each line.
func (tt *TUITester) GetContentTrimmed() string {
	content := tt.GetContent()
	lines := strings.Split(content, "\n")
	var trimmed []string
	for _, line := range lines {
		trimmed = append(trimmed, strings.TrimRight(line, " "))
	}
	return strings.Join(trimmed, "\n")
}

// GetLines returns the screen content as individual lines.
func (tt *TUITester) GetLines() []string {
	content := tt.GetContentTrimmed()
	return strings.Split(content, "\n")
}

// GetLine returns a specific line from the screen (0-indexed).
func (tt *TUITester) GetLine(lineNum int) string {
	lines := tt.GetLines()
	if lineNum < 0 || lineNum >= len(lines) {
		return ""
	}
	return lines[lineNum]
}

// GetCellAt returns the character at a specific position.
func (tt *TUITester) GetCellAt(x, y int) rune {
	str, _, _ := tt.Screen.Get(x, y)
	if len(str) == 0 {
		return ' '
	}
	r := []rune(str)
	return r[0]
}

// GetRegion returns a rectangular region of the screen.
func (tt *TUITester) GetRegion(x, y, width, height int) string {
	var sb strings.Builder
	for row := y; row < y+height; row++ {
		for col := x; col < x+width; col++ {
			str, _, _ := tt.Screen.Get(col, row)
			if len(str) == 0 {
				sb.WriteRune(' ')
			} else {
				sb.WriteString(str)
			}
		}
		if row < y+height-1 {
			sb.WriteRune('\n')
		}
	}
	return sb.String()
}

// CaptureScreen saves the current screen state to history.
func (tt *TUITester) CaptureScreen(label string) ScreenCapture {
	w, h := tt.Screen.Size()
	capture := ScreenCapture{
		Timestamp: time.Now(),
		Content:   tt.GetContentTrimmed(),
		Width:     w,
		Height:    h,
		Label:     label,
	}

	tt.captureMx.Lock()
	tt.captureHistory = append(tt.captureHistory, capture)
	// Trim history if exceeds max
	if len(tt.captureHistory) > tt.maxCaptures {
		tt.captureHistory = tt.captureHistory[1:]
	}
	tt.captureMx.Unlock()

	return capture
}

// GetCaptureHistory returns all captured screen states.
func (tt *TUITester) GetCaptureHistory() []ScreenCapture {
	tt.captureMx.Lock()
	defer tt.captureMx.Unlock()
	result := make([]ScreenCapture, len(tt.captureHistory))
	copy(result, tt.captureHistory)
	return result
}

// ClearCaptureHistory clears the capture history.
func (tt *TUITester) ClearCaptureHistory() {
	tt.captureMx.Lock()
	defer tt.captureMx.Unlock()
	tt.captureHistory = make([]ScreenCapture, 0, tt.maxCaptures)
}

// AssertContentContains checks if the screen contains expected text.
func (tt *TUITester) AssertContentContains(expected string) bool {
	tt.t.Helper()
	content := tt.GetContent()
	if !strings.Contains(content, expected) {
		tt.t.Errorf("Screen missing %q\nActual content:\n%s", expected, tt.GetContentTrimmed())
		return false
	}
	return true
}

// AssertContentNotContains checks that the screen does NOT contain unexpected text.
func (tt *TUITester) AssertContentNotContains(unexpected string) bool {
	tt.t.Helper()
	content := tt.GetContent()
	if strings.Contains(content, unexpected) {
		tt.t.Errorf("Screen unexpectedly contains %q\nActual content:\n%s", unexpected, tt.GetContentTrimmed())
		return false
	}
	return true
}

// AssertContentMatches checks if the screen content matches a regex pattern.
func (tt *TUITester) AssertContentMatches(pattern string) bool {
	tt.t.Helper()
	content := tt.GetContent()
	re, err := regexp.Compile(pattern)
	if err != nil {
		tt.t.Errorf("Invalid regex pattern %q: %v", pattern, err)
		return false
	}
	if !re.MatchString(content) {
		tt.t.Errorf("Screen content does not match pattern %q\nActual content:\n%s", pattern, tt.GetContentTrimmed())
		return false
	}
	return true
}

// AssertLineContains checks if a specific line contains expected text.
func (tt *TUITester) AssertLineContains(lineNum int, expected string) bool {
	tt.t.Helper()
	line := tt.GetLine(lineNum)
	if !strings.Contains(line, expected) {
		tt.t.Errorf("Line %d missing %q\nActual line: %q", lineNum, expected, line)
		return false
	}
	return true
}

// AssertFocusedPrimitive checks if the focused primitive matches expected type.
func (tt *TUITester) AssertFocusedPrimitive(expected tview.Primitive) bool {
	tt.t.Helper()
	focused := tt.App.GetFocus()
	if focused != expected {
		tt.t.Errorf("Expected focus on %T, got %T", expected, focused)
		return false
	}
	return true
}

// WaitForContent waits for expected text to appear on screen.
func (tt *TUITester) WaitForContent(expected string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		tt.App.QueueUpdateDraw(func() {})
		if strings.Contains(tt.GetContent(), expected) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// WaitForContentGone waits for text to disappear from screen.
func (tt *TUITester) WaitForContentGone(unexpected string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		tt.App.QueueUpdateDraw(func() {})
		if !strings.Contains(tt.GetContent(), unexpected) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// WaitForCondition waits for a condition to become true.
func (tt *TUITester) WaitForCondition(timeout time.Duration, condition func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

// WaitForDraw waits for the screen to be drawn and stabilized.
func (tt *TUITester) WaitForDraw(timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	var lastContent string
	for time.Now().Before(deadline) {
		tt.App.QueueUpdateDraw(func() {})
		content := tt.GetContent()
		if content == lastContent {
			return // Screen stabilized
		}
		lastContent = content
		time.Sleep(30 * time.Millisecond)
	}
}

// AssertNoFreeze verifies that an action completes within timeout (freeze detection).
func (tt *TUITester) AssertNoFreeze(timeout time.Duration, action func()) bool {
	tt.t.Helper()
	done := make(chan struct{})
	go func() {
		action()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		tt.t.Fatalf("Action did not complete within %v - possible freeze/deadlock", timeout)
		return false
	}
}

// AssertNoFreezeWithCapture runs action and captures screen on timeout.
func (tt *TUITester) AssertNoFreezeWithCapture(timeout time.Duration, action func()) bool {
	tt.t.Helper()
	done := make(chan struct{})
	go func() {
		action()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		capture := tt.CaptureScreen("freeze-detected")
		tt.t.Fatalf("Action did not complete within %v - possible freeze/deadlock\nScreen at timeout:\n%s",
			timeout, capture.Content)
		return false
	}
}

// GetFocusedPrimitive returns the currently focused tview primitive.
func (tt *TUITester) GetFocusedPrimitive() tview.Primitive {
	return tt.App.GetFocus()
}

// ScreenDump returns a formatted screen dump for debugging.
func (tt *TUITester) ScreenDump() string {
	w, h := tt.Screen.Size()
	return "=== Screen Dump (" + itoa(w) + "x" + itoa(h) + ") ===\n" +
		tt.GetContentTrimmed() + "\n=== End Dump ==="
}

// ScreenDumpWithBorder returns a screen dump with visual border for debugging.
func (tt *TUITester) ScreenDumpWithBorder() string {
	w, _ := tt.Screen.Size()
	content := tt.GetContentTrimmed()
	lines := strings.Split(content, "\n")

	var sb strings.Builder
	// Top border
	sb.WriteString("┌")
	for i := 0; i < w; i++ {
		sb.WriteString("─")
	}
	sb.WriteString("┐\n")

	// Content with side borders
	for _, line := range lines {
		sb.WriteString("│")
		sb.WriteString(line)
		// Pad to width
		for i := len(line); i < w; i++ {
			sb.WriteString(" ")
		}
		sb.WriteString("│\n")
	}

	// Bottom border
	sb.WriteString("└")
	for i := 0; i < w; i++ {
		sb.WriteString("─")
	}
	sb.WriteString("┘")

	return sb.String()
}

// LogScreen logs the current screen content to the test log.
func (tt *TUITester) LogScreen(label string) {
	tt.t.Logf("%s:\n%s", label, tt.ScreenDump())
}

// LogScreenOnFailure captures the screen and logs it only if the test fails.
func (tt *TUITester) LogScreenOnFailure(label string) {
	capture := tt.CaptureScreen(label)
	tt.t.Cleanup(func() {
		if tt.t.Failed() {
			tt.t.Logf("Screen capture at '%s':\n%s", label, capture.Content)
		}
	})
}

// Scenario represents a test scenario with steps.
type Scenario struct {
	tt    *TUITester
	name  string
	steps []func() error
}

// NewScenario creates a new test scenario.
func (tt *TUITester) NewScenario(name string) *Scenario {
	return &Scenario{
		tt:    tt,
		name:  name,
		steps: make([]func() error, 0),
	}
}

// Step adds a step to the scenario.
func (s *Scenario) Step(name string, action func() error) *Scenario {
	s.steps = append(s.steps, func() error {
		if err := action(); err != nil {
			return fmt.Errorf("step %q failed: %w", name, err)
		}
		return nil
	})
	return s
}

// Run executes all steps in the scenario.
func (s *Scenario) Run() error {
	for i, step := range s.steps {
		if err := step(); err != nil {
			s.tt.CaptureScreen(fmt.Sprintf("scenario-%s-step-%d-failed", s.name, i))
			return fmt.Errorf("scenario %q: %w", s.name, err)
		}
	}
	return nil
}

// goldenDir returns the path to the testdata directory for golden files.
func goldenDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata")
}

// goldenPath returns the full path for a golden file.
func goldenPath(name string) string {
	return filepath.Join(goldenDir(), name+".golden")
}

// GetContentNormalized returns screen content with trailing whitespace removed
// from each line and empty trailing lines removed for deterministic comparison.
func (tt *TUITester) GetContentNormalized() string {
	content := tt.GetContentTrimmed()
	lines := strings.Split(content, "\n")

	// Remove trailing empty lines
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n") + "\n"
}

// CompareGolden compares current screen content against a golden file.
// If -update flag is set, it writes the current content as the new golden file.
// On mismatch, it writes actual content to a .actual file for easy diffing.
func (tt *TUITester) CompareGolden(t *testing.T, name string) bool {
	t.Helper()

	actual := tt.GetContentNormalized()
	path := goldenPath(name)

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("Failed to create golden dir: %v", err)
		}
		if err := os.WriteFile(path, []byte(actual), 0o644); err != nil {
			t.Fatalf("Failed to write golden file: %v", err)
		}
		t.Logf("Updated golden file: %s", path)
		return true
	}

	expected, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No golden file yet — write actual and fail
			actualPath := path + ".actual"
			_ = os.WriteFile(actualPath, []byte(actual), 0o644)
			t.Fatalf("Golden file not found: %s\nRun with -update to create it.\nActual written to: %s", path, actualPath)
		}
		t.Fatalf("Failed to read golden file: %v", err)
	}

	if actual != string(expected) {
		actualPath := path + ".actual"
		_ = os.WriteFile(actualPath, []byte(actual), 0o644)
		t.Errorf("Screen content does not match golden file: %s\nActual written to: %s\nRun 'diff %s %s' to see differences",
			path, actualPath, path, actualPath)
		return false
	}

	return true
}

// itoa is a simple int to string converter to avoid fmt import in hot paths.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
