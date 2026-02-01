// Package testutil provides TUI testing utilities inspired by Microsoft's tui-test.
// This implements auto-wait, assertions, and terminal helpers in Go.
package testutil

import (
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

// Terminal wraps tcell.SimulationScreen with tui-test style API.
type Terminal struct {
	t      *testing.T
	screen tcell.SimulationScreen
	mu     sync.Mutex
}

// NewTerminal creates a new test terminal.
func NewTerminal(t *testing.T) *Terminal {
	t.Helper()
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("Failed to init simulation screen: %v", err)
	}
	screen.SetSize(120, 40)
	return &Terminal{t: t, screen: screen}
}

// Screen returns the underlying tcell.SimulationScreen.
func (term *Terminal) Screen() tcell.SimulationScreen {
	return term.screen
}

// Write sends text to the terminal.
func (term *Terminal) Write(text string) {
	term.mu.Lock()
	defer term.mu.Unlock()
	for _, r := range text {
		term.screen.InjectKey(tcell.KeyRune, r, tcell.ModNone)
		time.Sleep(5 * time.Millisecond)
	}
}

// Submit sends text followed by Enter.
func (term *Terminal) Submit(text string) {
	term.Write(text)
	term.KeyEnter()
}

// KeyEnter sends Enter key.
func (term *Terminal) KeyEnter() {
	term.mu.Lock()
	defer term.mu.Unlock()
	term.screen.InjectKey(tcell.KeyEnter, 0, tcell.ModNone)
	time.Sleep(10 * time.Millisecond)
}

// KeyEscape sends Escape key.
func (term *Terminal) KeyEscape() {
	term.mu.Lock()
	defer term.mu.Unlock()
	term.screen.InjectKey(tcell.KeyEscape, 0, tcell.ModNone)
	time.Sleep(10 * time.Millisecond)
}

// KeyTab sends Tab key.
func (term *Terminal) KeyTab() {
	term.mu.Lock()
	defer term.mu.Unlock()
	term.screen.InjectKey(tcell.KeyTab, 0, tcell.ModNone)
	time.Sleep(10 * time.Millisecond)
}

// KeyBackspace sends Backspace key.
func (term *Terminal) KeyBackspace(count int) {
	term.mu.Lock()
	defer term.mu.Unlock()
	for i := 0; i < count; i++ {
		term.screen.InjectKey(tcell.KeyBackspace2, 0, tcell.ModNone)
		time.Sleep(5 * time.Millisecond)
	}
}

// KeyCtrl sends Ctrl+key.
func (term *Terminal) KeyCtrl(key rune) {
	term.mu.Lock()
	defer term.mu.Unlock()
	switch key {
	case 'c', 'C':
		term.screen.InjectKey(tcell.KeyCtrlC, 0, tcell.ModCtrl)
	case 'd', 'D':
		term.screen.InjectKey(tcell.KeyCtrlD, 0, tcell.ModCtrl)
	case 'h', 'H':
		term.screen.InjectKey(tcell.KeyCtrlH, 0, tcell.ModCtrl)
	case 'l', 'L':
		term.screen.InjectKey(tcell.KeyCtrlL, 0, tcell.ModCtrl)
	default:
		term.screen.InjectKey(tcell.Key(key-'a'+1), 0, tcell.ModCtrl)
	}
	time.Sleep(10 * time.Millisecond)
}

// Key sends a specific key.
func (term *Terminal) Key(key tcell.Key) {
	term.mu.Lock()
	defer term.mu.Unlock()
	term.screen.InjectKey(key, 0, tcell.ModNone)
	time.Sleep(10 * time.Millisecond)
}

// GetContent returns the current screen content as a string.
func (term *Terminal) GetContent() string {
	term.mu.Lock()
	defer term.mu.Unlock()

	w, h := term.screen.Size()
	var content strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, _, _, _ := term.screen.GetContent(x, y)
			if r == 0 {
				content.WriteRune(' ')
			} else {
				content.WriteRune(r)
			}
		}
		content.WriteRune('\n')
	}
	return content.String()
}

// Locator represents a text locator for assertions.
type Locator struct {
	term    *Terminal
	text    string
	pattern *regexp.Regexp
	full    bool
}

// GetByText returns a locator that matches text.
func (term *Terminal) GetByText(text string) *Locator {
	return &Locator{term: term, text: text}
}

// GetByRegex returns a locator that matches a regex pattern.
func (term *Terminal) GetByRegex(pattern *regexp.Regexp) *Locator {
	return &Locator{term: term, pattern: pattern}
}

// Full makes the locator match full words only.
func (l *Locator) Full() *Locator {
	l.full = true
	return l
}

// IsVisible checks if the locator matches visible content.
func (l *Locator) IsVisible() bool {
	content := l.term.GetContent()
	if l.pattern != nil {
		return l.pattern.MatchString(content)
	}
	if l.full {
		// Match as full word
		pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(l.text) + `\b`)
		return pattern.MatchString(content)
	}
	return strings.Contains(content, l.text)
}

// ExpectOptions configures expect behavior.
type ExpectOptions struct {
	Timeout time.Duration
}

// DefaultExpectOptions returns default expect options.
func DefaultExpectOptions() ExpectOptions {
	return ExpectOptions{Timeout: 5 * time.Second}
}

// Expect provides tui-test style assertions.
type Expect struct {
	t    *testing.T
	opts ExpectOptions
}

// NewExpect creates a new expect instance.
func NewExpect(t *testing.T, opts ...ExpectOptions) *Expect {
	opt := DefaultExpectOptions()
	if len(opts) > 0 {
		opt = opts[0]
	}
	return &Expect{t: t, opts: opt}
}

// ToBeVisible waits for the locator to be visible.
func (e *Expect) ToBeVisible(loc *Locator) {
	e.t.Helper()
	deadline := time.Now().Add(e.opts.Timeout)
	for time.Now().Before(deadline) {
		if loc.IsVisible() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	e.t.Errorf("Expected text to be visible: %q", loc.text)
}

// NotToBeVisible waits for the locator to not be visible.
func (e *Expect) NotToBeVisible(loc *Locator) {
	e.t.Helper()
	deadline := time.Now().Add(e.opts.Timeout)
	for time.Now().Before(deadline) {
		if !loc.IsVisible() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	e.t.Errorf("Expected text to not be visible: %q", loc.text)
}

// ToContain checks if content contains text.
func (e *Expect) ToContain(term *Terminal, text string) {
	e.t.Helper()
	deadline := time.Now().Add(e.opts.Timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(term.GetContent(), text) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	e.t.Errorf("Expected content to contain: %q\nActual content:\n%s", text, term.GetContent())
}

// ToMatch checks if content matches regex.
func (e *Expect) ToMatch(term *Terminal, pattern *regexp.Regexp) {
	e.t.Helper()
	deadline := time.Now().Add(e.opts.Timeout)
	for time.Now().Before(deadline) {
		if pattern.MatchString(term.GetContent()) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	e.t.Errorf("Expected content to match: %s\nActual content:\n%s", pattern.String(), term.GetContent())
}

// TUITest provides a complete TUI testing context.
type TUITest struct {
	T        *testing.T
	Terminal *Terminal
	Expect   *Expect
}

// NewTUITest creates a new TUI test context.
func NewTUITest(t *testing.T) *TUITest {
	terminal := NewTerminal(t)
	return &TUITest{
		T:        t,
		Terminal: terminal,
		Expect:   NewExpect(t),
	}
}

// Cleanup should be called at the end of each test.
func (tt *TUITest) Cleanup() {
	tt.Terminal.screen.Fini()
}

// Wait waits for a duration.
func (tt *TUITest) Wait(d time.Duration) {
	time.Sleep(d)
}

// WaitForReady waits for the application to be ready.
func (tt *TUITest) WaitForReady(readyText string) {
	tt.Expect.ToBeVisible(tt.Terminal.GetByText(readyText))
}
