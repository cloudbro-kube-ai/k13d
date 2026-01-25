package actions

import (
	"github.com/gdamore/tcell/v2"
)

// CommonActions returns the shared actions available across all views.
// These follow k9s conventions for navigation and basic operations.
func CommonActions() *KeyActions {
	ka := NewKeyActions()

	// Navigation (will be bound to actual handlers by App)
	ka.AddRune('q', NewSharedAction("Quit/Back", nil))
	ka.Add(tcell.KeyEsc, NewSharedAction("Back/Cancel", nil))
	ka.Add(tcell.KeyEnter, NewSharedAction("Select/Enter", nil))

	// Vim-style navigation
	ka.AddRune('j', NewSharedAction("Down", nil))
	ka.AddRune('k', NewSharedAction("Up", nil))
	ka.AddRune('h', NewSharedAction("Left", nil))
	ka.AddRune('l', NewSharedAction("Right", nil))
	ka.AddRune('g', NewSharedAction("Top", nil))
	ka.AddRune('G', NewSharedAction("Bottom", nil))

	// Page navigation
	ka.AddWithMod(tcell.KeyRune, tcell.ModCtrl, KeyAction{
		Rune:        'u',
		Description: "Page Up",
		Opts:        ActionOpts{Visible: true, Shared: true},
	})
	ka.AddWithMod(tcell.KeyRune, tcell.ModCtrl, KeyAction{
		Rune:        'd',
		Description: "Page Down",
		Opts:        ActionOpts{Visible: true, Shared: true},
	})

	// Command mode
	ka.AddRune(':', NewSharedAction("Command Mode", nil))
	ka.AddRune('/', NewSharedAction("Filter", nil))

	// Help
	ka.AddRune('?', NewSharedAction("Help", nil))

	// Refresh
	ka.AddWithMod(tcell.KeyRune, tcell.ModCtrl, KeyAction{
		Rune:        'r',
		Description: "Refresh",
		Opts:        ActionOpts{Visible: true, Shared: true},
	})

	return ka
}

// ResourceActions returns common actions for resource views.
func ResourceActions() *KeyActions {
	ka := NewKeyActions()

	// View operations
	ka.AddRune('y', NewKeyAction("YAML", nil))
	ka.AddRune('d', NewKeyAction("Describe", nil))
	ka.AddRune('e', NewKeyAction("Edit", nil))

	// Logs (for pods and related)
	ka.AddRune('l', NewKeyAction("Logs", nil))

	// AI operations
	ka.AddRune('L', NewKeyAction("AI Analyze", nil))

	// Delete (dangerous)
	ka.AddWithMod(tcell.KeyRune, tcell.ModCtrl, NewDangerousAction("Delete", nil))

	// Multi-select
	ka.Add(tcell.KeyRune, KeyAction{
		Rune:        ' ',
		Description: "Toggle Select",
		Opts:        ActionOpts{Visible: true},
	})

	return ka
}

// PodActions returns Pod-specific actions.
func PodActions() *KeyActions {
	ka := NewKeyActions()

	ka.AddRune('s', NewKeyAction("Shell", nil))
	ka.AddRune('a', NewKeyAction("Attach", nil))
	ka.Add(tcell.KeyRune, KeyAction{
		Rune:        'F',
		Modifiers:   tcell.ModShift,
		Description: "Port Forward",
		Opts:        ActionOpts{Visible: true},
	})
	ka.AddRune('p', NewKeyAction("Logs Previous", nil))

	return ka
}

// DeploymentActions returns Deployment-specific actions.
func DeploymentActions() *KeyActions {
	ka := NewKeyActions()

	ka.AddRune('s', NewKeyAction("Scale", nil))
	ka.AddRune('r', NewKeyAction("Restart", nil))
	ka.AddRune('R', NewKeyAction("Rollback", nil))
	ka.AddRune('i', NewKeyAction("Image", nil))

	return ka
}

// NodeActions returns Node-specific actions.
func NodeActions() *KeyActions {
	ka := NewKeyActions()

	ka.AddRune('c', NewKeyAction("Cordon", nil))
	ka.AddRune('u', NewKeyAction("Uncordon", nil))
	ka.AddRune('D', NewDangerousAction("Drain", nil))

	return ka
}

// ServiceActions returns Service-specific actions.
func ServiceActions() *KeyActions {
	ka := NewKeyActions()

	ka.Add(tcell.KeyRune, KeyAction{
		Rune:        'F',
		Modifiers:   tcell.ModShift,
		Description: "Port Forward",
		Opts:        ActionOpts{Visible: true},
	})

	return ka
}

// SecretActions returns Secret-specific actions.
func SecretActions() *KeyActions {
	ka := NewKeyActions()

	ka.AddRune('x', NewKeyAction("Decode", nil))

	return ka
}

// ConfigMapActions returns ConfigMap-specific actions.
func ConfigMapActions() *KeyActions {
	ka := NewKeyActions()

	// ConfigMaps use the common resource actions

	return ka
}

// CronJobActions returns CronJob-specific actions.
func CronJobActions() *KeyActions {
	ka := NewKeyActions()

	ka.AddRune('t', NewKeyAction("Trigger", nil))
	ka.AddRune('s', NewKeyAction("Suspend/Resume", nil))

	return ka
}

// JobActions returns Job-specific actions.
func JobActions() *KeyActions {
	ka := NewKeyActions()

	ka.AddRune('l', NewKeyAction("Logs", nil))

	return ka
}

// StatefulSetActions returns StatefulSet-specific actions.
func StatefulSetActions() *KeyActions {
	ka := NewKeyActions()

	ka.AddRune('s', NewKeyAction("Scale", nil))
	ka.AddRune('r', NewKeyAction("Restart", nil))

	return ka
}

// DaemonSetActions returns DaemonSet-specific actions.
func DaemonSetActions() *KeyActions {
	ka := NewKeyActions()

	ka.AddRune('r', NewKeyAction("Restart", nil))

	return ka
}
