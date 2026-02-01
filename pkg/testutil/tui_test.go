package testutil

import (
	"regexp"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

func TestTerminalWrite(t *testing.T) {
	term := NewTerminal(t)
	defer term.Screen().Fini()

	// Inject keys directly without sleep (for unit test)
	term.Screen().InjectKey(tcell.KeyRune, 't', tcell.ModNone)
	term.Screen().InjectKey(tcell.KeyRune, 'e', tcell.ModNone)
	term.Screen().InjectKey(tcell.KeyRune, 's', tcell.ModNone)
	term.Screen().InjectKey(tcell.KeyRune, 't', tcell.ModNone)
}

func TestTerminalSubmit(t *testing.T) {
	term := NewTerminal(t)
	defer term.Screen().Fini()

	term.Screen().InjectKey(tcell.KeyRune, 'h', tcell.ModNone)
	term.Screen().InjectKey(tcell.KeyEnter, 0, tcell.ModNone)
}

func TestLocatorIsVisible(t *testing.T) {
	term := NewTerminal(t)
	defer term.Screen().Fini()

	// Empty screen test
	loc := term.GetByText("nonexistent")
	if loc.IsVisible() {
		t.Error("Expected text to not be visible on empty screen")
	}
}

func TestLocatorWithRegex(t *testing.T) {
	term := NewTerminal(t)
	defer term.Screen().Fini()

	pattern := regexp.MustCompile(`test\d+`)
	loc := term.GetByRegex(pattern)

	// Empty screen should not match
	if loc.IsVisible() {
		t.Error("Expected pattern to not match on empty screen")
	}
}

func TestExpectWithTimeout(t *testing.T) {
	term := NewTerminal(t)
	defer term.Screen().Fini()

	expect := NewExpect(t, ExpectOptions{Timeout: 100 * time.Millisecond})

	// This should fail quickly since text won't appear
	// We just test it doesn't hang
	_ = expect
}

func TestTUITestContext(t *testing.T) {
	tt := NewTUITest(t)
	defer tt.Cleanup()

	// Basic context creation
	if tt.Terminal == nil {
		t.Error("Terminal should not be nil")
	}
	if tt.Expect == nil {
		t.Error("Expect should not be nil")
	}
}

func TestTerminalKeySequence(t *testing.T) {
	term := NewTerminal(t)
	defer term.Screen().Fini()

	// Test key sequence doesn't panic - inject directly
	term.Screen().InjectKey(tcell.KeyRune, ':', tcell.ModNone)
	term.Screen().InjectKey(tcell.KeyEnter, 0, tcell.ModNone)
	term.Screen().InjectKey(tcell.KeyEscape, 0, tcell.ModNone)
	term.Screen().InjectKey(tcell.KeyTab, 0, tcell.ModNone)
	term.Screen().InjectKey(tcell.KeyBackspace2, 0, tcell.ModNone)
	term.Screen().InjectKey(tcell.KeyCtrlC, 0, tcell.ModCtrl)
}

func TestGetContent(t *testing.T) {
	term := NewTerminal(t)
	defer term.Screen().Fini()

	content := term.GetContent()
	if content == "" {
		t.Error("Content should not be empty (should have space characters)")
	}
}
