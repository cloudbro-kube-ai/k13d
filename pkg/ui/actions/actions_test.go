package actions

import (
	"context"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestNewKeyActions(t *testing.T) {
	ka := NewKeyActions()
	if ka == nil {
		t.Fatal("NewKeyActions returned nil")
	}
	if ka.Len() != 0 {
		t.Errorf("Len() = %d, want 0", ka.Len())
	}
}

func TestAddAndGet(t *testing.T) {
	ka := NewKeyActions()

	called := false
	action := NewKeyAction("Test", func(ctx context.Context) error {
		called = true
		return nil
	})

	ka.Add(tcell.KeyEnter, action)

	// Simulate key event
	event := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	got, ok := ka.Get(event)
	if !ok {
		t.Fatal("Get returned false")
	}
	if got.Description != "Test" {
		t.Errorf("Description = %s, want Test", got.Description)
	}

	// Execute action
	if got.Action != nil {
		_ = got.Action(context.Background())
	}
	if !called {
		t.Error("Action was not called")
	}
}

func TestAddRune(t *testing.T) {
	ka := NewKeyActions()

	action := NewKeyAction("Quit", nil)
	ka.AddRune('q', action)

	event := tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)
	got, ok := ka.Get(event)
	if !ok {
		t.Fatal("Get returned false for rune")
	}
	if got.Description != "Quit" {
		t.Errorf("Description = %s, want Quit", got.Description)
	}
	if got.Rune != 'q' {
		t.Errorf("Rune = %c, want q", got.Rune)
	}
}

func TestAddWithMod(t *testing.T) {
	ka := NewKeyActions()

	// For Ctrl+letter combinations, terminals typically send a control key code
	// tcell maps Ctrl+R to KeyCtrlR (82)
	// Use Add with the KeyCtrl[Letter] constant
	ka.Add(tcell.KeyCtrlR, KeyAction{
		Description: "Refresh",
	})

	// Test with KeyCtrlR event (this is what terminals actually send)
	event := tcell.NewEventKey(tcell.KeyCtrlR, 0, tcell.ModNone)
	got, ok := ka.Get(event)
	if !ok {
		t.Fatal("Get returned false for Ctrl+R")
	}
	if got.Description != "Refresh" {
		t.Errorf("Description = %s, want Refresh", got.Description)
	}
}

func TestAddWithModNonRune(t *testing.T) {
	ka := NewKeyActions()

	// For non-rune keys with modifiers, use AddWithMod
	ka.AddWithMod(tcell.KeyF1, tcell.ModCtrl, KeyAction{
		Description: "Help",
	})

	event := tcell.NewEventKey(tcell.KeyF1, 0, tcell.ModCtrl)
	got, ok := ka.Get(event)
	if !ok {
		t.Fatal("Get returned false for Ctrl+F1")
	}
	if got.Description != "Help" {
		t.Errorf("Description = %s, want Help", got.Description)
	}
}

func TestDelete(t *testing.T) {
	ka := NewKeyActions()

	ka.AddRune('q', NewKeyAction("Quit", nil))
	if ka.Len() != 1 {
		t.Errorf("Len() = %d, want 1", ka.Len())
	}

	ka.Delete(tcell.KeyRune, 'q', tcell.ModNone)
	if ka.Len() != 0 {
		t.Errorf("Len() = %d after delete, want 0", ka.Len())
	}
}

func TestMerge(t *testing.T) {
	ka1 := NewKeyActions()
	ka1.AddRune('a', NewKeyAction("Action A", nil))

	ka2 := NewKeyActions()
	ka2.AddRune('b', NewKeyAction("Action B", nil))
	ka2.AddRune('a', NewKeyAction("Action A Override", nil))

	ka1.Merge(ka2)

	if ka1.Len() != 2 {
		t.Errorf("Len() = %d, want 2", ka1.Len())
	}

	// Check override
	event := tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone)
	got, _ := ka1.Get(event)
	if got.Description != "Action A Override" {
		t.Errorf("Description = %s, want Action A Override", got.Description)
	}
}

func TestClear(t *testing.T) {
	ka := NewKeyActions()
	ka.AddRune('a', NewKeyAction("A", nil))
	ka.AddRune('b', NewKeyAction("B", nil))

	ka.Clear()
	if ka.Len() != 0 {
		t.Errorf("Len() = %d after clear, want 0", ka.Len())
	}
}

func TestList(t *testing.T) {
	ka := NewKeyActions()
	ka.AddRune('a', NewKeyAction("Visible", nil))
	ka.AddRune('b', KeyAction{
		Description: "Hidden",
		Opts:        ActionOpts{Visible: false},
	})

	list := ka.List()
	if len(list) != 1 {
		t.Errorf("List() len = %d, want 1", len(list))
	}
	if list[0].Description != "Visible" {
		t.Errorf("Description = %s, want Visible", list[0].Description)
	}
}

func TestSharedActions(t *testing.T) {
	ka := NewKeyActions()
	ka.AddRune('a', NewSharedAction("Shared", nil))
	ka.AddRune('b', NewKeyAction("Not Shared", nil))

	shared := ka.SharedActions()
	if len(shared) != 1 {
		t.Errorf("SharedActions() len = %d, want 1", len(shared))
	}
}

func TestNewSharedAction(t *testing.T) {
	action := NewSharedAction("Test", nil)
	if !action.Opts.Shared {
		t.Error("Shared should be true")
	}
	if !action.Opts.Visible {
		t.Error("Visible should be true")
	}
}

func TestNewDangerousAction(t *testing.T) {
	action := NewDangerousAction("Delete", nil)
	if !action.Opts.Dangerous {
		t.Error("Dangerous should be true")
	}
	if !action.Opts.Visible {
		t.Error("Visible should be true")
	}
}

func TestKeyDescriptor(t *testing.T) {
	tests := []struct {
		name     string
		action   KeyAction
		expected string
	}{
		{
			name: "Enter key",
			action: KeyAction{
				Key: tcell.KeyEnter,
			},
			expected: "Enter",
		},
		{
			name: "Rune key",
			action: KeyAction{
				Key:  tcell.KeyRune,
				Rune: 'q',
			},
			expected: "q",
		},
		{
			name: "Ctrl+R",
			action: KeyAction{
				Key:       tcell.KeyRune,
				Rune:      'r',
				Modifiers: tcell.ModCtrl,
			},
			expected: "Ctrl+r",
		},
		{
			name: "Shift+F",
			action: KeyAction{
				Key:       tcell.KeyRune,
				Rune:      'F',
				Modifiers: tcell.ModShift,
			},
			expected: "Shift+F",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.action.KeyDescriptor()
			if got != tt.expected {
				t.Errorf("KeyDescriptor() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestHints(t *testing.T) {
	ka := NewKeyActions()
	ka.AddRune('q', NewKeyAction("Quit", nil))
	ka.AddRune('d', NewDangerousAction("Delete", nil))

	hints := ka.Hints()
	if len(hints) != 2 {
		t.Errorf("Hints() len = %d, want 2", len(hints))
	}

	// Check that dangerous flag is preserved
	hasDangerous := false
	for _, h := range hints {
		if h.Dangerous {
			hasDangerous = true
			break
		}
	}
	if !hasDangerous {
		t.Error("Expected at least one dangerous hint")
	}
}

func TestMergeNil(t *testing.T) {
	ka := NewKeyActions()
	ka.AddRune('a', NewKeyAction("A", nil))

	// Should not panic
	ka.Merge(nil)

	if ka.Len() != 1 {
		t.Errorf("Len() = %d, want 1", ka.Len())
	}
}

func TestGetNotFound(t *testing.T) {
	ka := NewKeyActions()

	event := tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone)
	_, ok := ka.Get(event)
	if ok {
		t.Error("Get should return false for non-existent key")
	}
}
