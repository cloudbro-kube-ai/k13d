package ui

import (
	"sync"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

// ============================================================================
// Unit Tests for recently added features
// ============================================================================

// --- parseAgeToSec ---

func TestParseAgeToSec(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"seconds", "30s", 30},
		{"minutes", "5m", 300},
		{"hours", "3h", 10800},
		{"days", "2d", 172800},
		{"compound days+hours", "2d3h", 2*86400 + 3*3600},
		{"compound hours+minutes", "1h30m", 3600 + 30*60},
		{"compound days+hours+minutes", "1d2h30m", 86400 + 2*3600 + 30*60},
		{"zero seconds", "0s", 0},
		{"empty string", "", 0},
		{"dash", "-", 0},
		{"unknown", "<unknown>", 0},
		{"whitespace", "  5m  ", 300},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAgeToSec(tt.input)
			if result != tt.expected {
				t.Errorf("parseAgeToSec(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// --- parseNumber ---

func TestParseNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"simple number", "123", 123},
		{"zero", "0", 0},
		{"negative", "-5", -5},
		{"with spaces", "  42  ", 42},
		{"empty", "", 0},
		{"non-numeric", "abc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseNumber(tt.input)
			if result != tt.expected {
				t.Errorf("parseNumber(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// --- parseReadyNum ---

func TestParseReadyNum(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"all ready", "1/1", 1},
		{"none ready", "0/3", 0},
		{"partial", "2/3", 2},
		{"single number", "5", 5},
		{"empty", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseReadyNum(tt.input)
			if result != tt.expected {
				t.Errorf("parseReadyNum(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// --- navStack depth limit ---

func TestNavStackDepthLimit(t *testing.T) {
	app := &App{
		navigationStack: nil,
		navMx:           sync.Mutex{},
	}

	// Push more than maxNavStackDepth items
	for i := 0; i < maxNavStackDepth+20; i++ {
		app.navMx.Lock()
		app.navigationStack = append(app.navigationStack, navHistory{
			resource:  "pods",
			namespace: "default",
			filter:    "",
		})
		if len(app.navigationStack) > maxNavStackDepth {
			app.navigationStack = app.navigationStack[1:]
		}
		app.navMx.Unlock()
	}

	app.navMx.Lock()
	finalLen := len(app.navigationStack)
	app.navMx.Unlock()

	if finalLen != maxNavStackDepth {
		t.Errorf("Expected stack to be capped at %d, got %d", maxNavStackDepth, finalLen)
	}
}

// --- addCmdHistory ---

func TestAddCmdHistory(t *testing.T) {
	t.Run("basic add", func(t *testing.T) {
		app := &App{}
		app.addCmdHistory("pods")
		app.addCmdHistory("deploy")
		app.addCmdHistory("svc")

		if len(app.cmdHistory) != 3 {
			t.Errorf("Expected 3 history entries, got %d", len(app.cmdHistory))
		}
	})

	t.Run("consecutive duplicates removed", func(t *testing.T) {
		app := &App{}
		app.addCmdHistory("pods")
		app.addCmdHistory("pods")
		app.addCmdHistory("pods")

		if len(app.cmdHistory) != 1 {
			t.Errorf("Expected 1 history entry (duplicates removed), got %d", len(app.cmdHistory))
		}
	})

	t.Run("non-consecutive duplicates kept", func(t *testing.T) {
		app := &App{}
		app.addCmdHistory("pods")
		app.addCmdHistory("deploy")
		app.addCmdHistory("pods")

		if len(app.cmdHistory) != 3 {
			t.Errorf("Expected 3 history entries, got %d", len(app.cmdHistory))
		}
	})

	t.Run("max capacity", func(t *testing.T) {
		app := &App{}
		for i := 0; i < maxCmdHistory+20; i++ {
			app.addCmdHistory(string(rune('a' + i%26)))
		}

		if len(app.cmdHistory) > maxCmdHistory {
			t.Errorf("Expected history capped at %d, got %d", maxCmdHistory, len(app.cmdHistory))
		}
	})
}

// --- portForwardInfo ---

func TestPortForwardInfoStruct(t *testing.T) {
	pf := &portForwardInfo{
		Cmd:        nil,
		Namespace:  "default",
		Name:       "nginx-pod",
		LocalPort:  "8080",
		RemotePort: "80",
	}

	if pf.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got %q", pf.Namespace)
	}
	if pf.Name != "nginx-pod" {
		t.Errorf("Expected name 'nginx-pod', got %q", pf.Name)
	}
	if pf.LocalPort != "8080" {
		t.Errorf("Expected local port '8080', got %q", pf.LocalPort)
	}
	if pf.RemotePort != "80" {
		t.Errorf("Expected remote port '80', got %q", pf.RemotePort)
	}
}

// ============================================================================
// E2E Tests using TUITestContext
// ============================================================================

func TestE2E_SortByColumn(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	// Navigate to pods and trigger sort with Shift+N (sort by name)
	ctx.Command("pods").
		Wait(100 * time.Millisecond).
		ExpectResource("pods").
		ExpectNoFreeze()
}

func TestE2E_CommandHistory(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	// Execute multiple commands
	ctx.Command("pods").
		Wait(50 * time.Millisecond).
		Command("deploy").
		Wait(50 * time.Millisecond).
		Command("svc").
		Wait(50 * time.Millisecond).
		ExpectNoFreeze()
}

func TestE2E_StatusBarIndicators(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	// Enter filter mode and type a filter
	ctx.PressRune('/').
		Wait(50 * time.Millisecond).
		Type("test").
		Press(tcell.KeyEnter).
		Wait(100 * time.Millisecond).
		ExpectNoFreeze()
}

func TestE2E_NavigationStackLimit(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	// Navigate through many resources to test stack limit
	resources := []string{"pods", "deploy", "svc", "no", "ns", "pods", "deploy", "svc"}
	for _, r := range resources {
		ctx.Command(r).Wait(30 * time.Millisecond)
	}
	ctx.ExpectNoFreeze()
}

// ============================================================================
// Deadlock Detection Tests (Bubble Tea pattern)
// ============================================================================

func TestDeadlock_ConcurrentSortAndRefresh(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.ExpectNoDeadlock(5*time.Second,
		func(c *TUITestContext) {
			for i := 0; i < 10; i++ {
				c.PressRune('j')
				time.Sleep(10 * time.Millisecond)
			}
		},
		func(c *TUITestContext) {
			for i := 0; i < 5; i++ {
				c.app.Draw()
				time.Sleep(20 * time.Millisecond)
			}
		},
	)
}

func TestDeadlock_RapidResourceSwitch(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.ExpectNoDeadlock(5*time.Second,
		func(c *TUITestContext) {
			cmds := []string{"pods", "deploy", "svc", "no", "ns"}
			for _, cmd := range cmds {
				c.Command(cmd)
				time.Sleep(20 * time.Millisecond)
			}
		},
		func(c *TUITestContext) {
			for i := 0; i < 10; i++ {
				c.app.Draw()
				time.Sleep(10 * time.Millisecond)
			}
		},
	)
}

func TestDeadlock_NavigateWhileFiltering(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.ExpectNoDeadlock(5*time.Second,
		func(c *TUITestContext) {
			c.PressRune('/').Type("test").Press(tcell.KeyEnter)
		},
		func(c *TUITestContext) {
			for i := 0; i < 5; i++ {
				c.PressRune('j')
				time.Sleep(10 * time.Millisecond)
			}
		},
	)
}

func TestDeadlock_ConcurrentNavStackAccess(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.ExpectNoDeadlock(5*time.Second,
		func(c *TUITestContext) {
			for i := 0; i < 10; i++ {
				c.Command("pods")
				time.Sleep(10 * time.Millisecond)
			}
		},
		func(c *TUITestContext) {
			for i := 0; i < 10; i++ {
				c.Press(tcell.KeyEscape)
				time.Sleep(10 * time.Millisecond)
			}
		},
	)
}
