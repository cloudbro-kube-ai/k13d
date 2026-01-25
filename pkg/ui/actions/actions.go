// Package actions provides a centralized key action system following k9s patterns.
// This replaces switch-based keybinding handling with a registry-based approach.
package actions

import (
	"context"
	"sync"

	"github.com/gdamore/tcell/v2"
)

// ActionHandler represents a function that handles a key action.
type ActionHandler func(ctx context.Context) error

// ActionOpts represents options for a key action.
type ActionOpts struct {
	Visible   bool // Show in help menu
	Shared    bool // Available across all views
	Dangerous bool // Requires confirmation
}

// KeyAction represents a single key binding action.
type KeyAction struct {
	Key         tcell.Key     // The key (e.g., tcell.KeyEnter)
	Rune        rune          // The rune for character keys (e.g., 'l')
	Modifiers   tcell.ModMask // Key modifiers (Ctrl, Alt, Shift)
	Description string        // Human-readable description
	Action      ActionHandler // The handler function
	Opts        ActionOpts    // Action options
}

// NewKeyAction creates a new key action with sensible defaults.
func NewKeyAction(description string, action ActionHandler, opts ...ActionOpts) KeyAction {
	ka := KeyAction{
		Description: description,
		Action:      action,
		Opts:        ActionOpts{Visible: true},
	}
	if len(opts) > 0 {
		ka.Opts = opts[0]
	}
	return ka
}

// NewSharedAction creates a shared action available across all views.
func NewSharedAction(description string, action ActionHandler) KeyAction {
	return KeyAction{
		Description: description,
		Action:      action,
		Opts:        ActionOpts{Visible: true, Shared: true},
	}
}

// NewDangerousAction creates an action that requires confirmation.
func NewDangerousAction(description string, action ActionHandler) KeyAction {
	return KeyAction{
		Description: description,
		Action:      action,
		Opts:        ActionOpts{Visible: true, Dangerous: true},
	}
}

// KeyActions is a thread-safe registry of key actions.
type KeyActions struct {
	actions map[string]*KeyAction
	mx      sync.RWMutex
}

// NewKeyActions creates a new KeyActions registry.
func NewKeyActions() *KeyActions {
	return &KeyActions{
		actions: make(map[string]*KeyAction),
	}
}

// keyString generates a unique string key for the action.
func keyString(key tcell.Key, r rune, mod tcell.ModMask) string {
	if key == tcell.KeyRune {
		if mod != 0 {
			return string([]rune{rune(mod), r})
		}
		return string(r)
	}
	return string([]rune{rune(key), rune(mod)})
}

// Add adds a key action to the registry.
func (ka *KeyActions) Add(key tcell.Key, action KeyAction) {
	ka.mx.Lock()
	defer ka.mx.Unlock()
	action.Key = key
	k := keyString(key, 0, action.Modifiers)
	ka.actions[k] = &action
}

// AddRune adds a rune-based key action to the registry.
func (ka *KeyActions) AddRune(r rune, action KeyAction) {
	ka.mx.Lock()
	defer ka.mx.Unlock()
	action.Key = tcell.KeyRune
	action.Rune = r
	k := keyString(tcell.KeyRune, r, action.Modifiers)
	ka.actions[k] = &action
}

// AddWithMod adds an action with modifier keys.
func (ka *KeyActions) AddWithMod(key tcell.Key, mod tcell.ModMask, action KeyAction) {
	ka.mx.Lock()
	defer ka.mx.Unlock()
	action.Key = key
	action.Modifiers = mod
	k := keyString(key, 0, mod)
	ka.actions[k] = &action
}

// AddRuneWithMod adds a rune action with modifier keys.
func (ka *KeyActions) AddRuneWithMod(r rune, mod tcell.ModMask, action KeyAction) {
	ka.mx.Lock()
	defer ka.mx.Unlock()
	action.Key = tcell.KeyRune
	action.Rune = r
	action.Modifiers = mod
	k := keyString(tcell.KeyRune, r, mod)
	ka.actions[k] = &action
}

// Get retrieves an action for the given key event.
func (ka *KeyActions) Get(event *tcell.EventKey) (*KeyAction, bool) {
	ka.mx.RLock()
	defer ka.mx.RUnlock()

	key := event.Key()
	r := event.Rune()
	mod := event.Modifiers()

	k := keyString(key, r, mod)
	action, ok := ka.actions[k]
	return action, ok
}

// Delete removes an action from the registry.
func (ka *KeyActions) Delete(key tcell.Key, r rune, mod tcell.ModMask) {
	ka.mx.Lock()
	defer ka.mx.Unlock()
	k := keyString(key, r, mod)
	delete(ka.actions, k)
}

// Merge merges another KeyActions into this one.
// Existing actions with the same key are overwritten.
func (ka *KeyActions) Merge(other *KeyActions) {
	if other == nil {
		return
	}
	ka.mx.Lock()
	defer ka.mx.Unlock()
	other.mx.RLock()
	defer other.mx.RUnlock()

	for k, v := range other.actions {
		ka.actions[k] = v
	}
}

// Clear removes all actions from the registry.
func (ka *KeyActions) Clear() {
	ka.mx.Lock()
	defer ka.mx.Unlock()
	ka.actions = make(map[string]*KeyAction)
}

// List returns all visible actions for help display.
func (ka *KeyActions) List() []*KeyAction {
	ka.mx.RLock()
	defer ka.mx.RUnlock()

	var list []*KeyAction
	for _, action := range ka.actions {
		if action.Opts.Visible {
			list = append(list, action)
		}
	}
	return list
}

// SharedActions returns only shared actions.
func (ka *KeyActions) SharedActions() []*KeyAction {
	ka.mx.RLock()
	defer ka.mx.RUnlock()

	var list []*KeyAction
	for _, action := range ka.actions {
		if action.Opts.Shared && action.Opts.Visible {
			list = append(list, action)
		}
	}
	return list
}

// Len returns the number of registered actions.
func (ka *KeyActions) Len() int {
	ka.mx.RLock()
	defer ka.mx.RUnlock()
	return len(ka.actions)
}

// KeyDescriptor returns a human-readable string for a key action.
func (ka *KeyAction) KeyDescriptor() string {
	var s string

	// Add modifiers
	if ka.Modifiers&tcell.ModCtrl != 0 {
		s += "Ctrl+"
	}
	if ka.Modifiers&tcell.ModAlt != 0 {
		s += "Alt+"
	}
	if ka.Modifiers&tcell.ModShift != 0 {
		s += "Shift+"
	}

	// Add key or rune
	if ka.Key == tcell.KeyRune {
		s += string(ka.Rune)
	} else {
		s += tcell.KeyNames[ka.Key]
	}

	return s
}

// Hint represents a menu hint for display.
type Hint struct {
	Key         string
	Description string
	Dangerous   bool
}

// Hints returns hints for all visible actions.
func (ka *KeyActions) Hints() []Hint {
	actions := ka.List()
	hints := make([]Hint, 0, len(actions))
	for _, a := range actions {
		hints = append(hints, Hint{
			Key:         a.KeyDescriptor(),
			Description: a.Description,
			Dangerous:   a.Opts.Dangerous,
		})
	}
	return hints
}
