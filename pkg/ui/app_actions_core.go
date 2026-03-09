package ui

import (
	"fmt"

	"github.com/cloudbro-kube-ai/k13d/pkg/db"
	"github.com/rivo/tview"
)

// checkTUIPermission checks RBAC authorization for TUI actions
// Returns true if allowed, false if denied (with flash error shown)
func (a *App) checkTUIPermission(resource, action string) bool {
	if a.authorizer == nil {
		// Warn once that RBAC is not configured (fail-open for backward compatibility)
		if !a.warnedNoRBAC {
			a.warnedNoRBAC = true
			a.logger.Warn("TUI RBAC not configured - all actions allowed. Configure authorizer for production use.")
		}
		return true
	}

	role := a.tuiRole
	if role == "" {
		role = "admin" // Default: admin for backward compatibility
	}

	a.mx.RLock()
	ns := a.currentNamespace
	a.mx.RUnlock()

	allowed, reason := a.authorizer.IsAllowed(role, resource, action, ns)
	if !allowed {
		a.flashMsg(fmt.Sprintf("Permission denied: %s", reason), true)

		// Record denial in audit log
		_ = db.RecordAudit(db.AuditEntry{
			User:            a.getTUIUser(),
			Action:          "authz_denied",
			Resource:        resource,
			Details:         reason,
			ActionType:      db.ActionTypeAuthzDenied,
			Source:          "tui",
			Success:         false,
			ErrorMsg:        reason,
			RequestedAction: action,
			TargetResource:  resource,
			TargetNamespace: ns,
			AuthzDecision:   "denied",
		})
		return false
	}
	return true
}

// getTUIUser returns the current TUI username
func (a *App) getTUIUser() string {
	if a.tuiRole != "" {
		return fmt.Sprintf("tui-user(%s)", a.tuiRole)
	}
	return "tui-user"
}

// recordTUIAudit records an audit entry for TUI actions with k8s context
func (a *App) recordTUIAudit(action, resource, details string, success bool, errMsg string) {
	entry := db.AuditEntry{
		User:       a.getTUIUser(),
		Action:     action,
		Resource:   resource,
		Details:    details,
		ActionType: db.ActionTypeMutation,
		Source:     "tui",
		Success:    success,
		ErrorMsg:   errMsg,
	}

	// Get k8s context info
	if a.k8s != nil {
		ctxName, cluster, user, err := a.k8s.GetContextInfo()
		if err == nil {
			entry.K8sContext = ctxName
			entry.K8sCluster = cluster
			entry.K8sUser = user
		}
	}

	// Get current namespace
	a.mx.RLock()
	entry.Namespace = a.currentNamespace
	a.mx.RUnlock()

	_ = db.RecordAudit(entry)
}

// safeSuspend wraps tview.Application.Suspend with a guard for SimulationScreen.
// SimulationScreen does not support Suspend (calling Fini on it panics with
// "close of closed channel"), so in test mode we just run fn directly.
func (a *App) safeSuspend(fn func()) {
	if a.useSimScreen {
		fn()
		return
	}
	a.Application.Suspend(fn)
}

// showModal adds a modal page with a full terminal sync to prevent ghosting
func (a *App) showModal(name string, p tview.Primitive, resize bool) {
	a.pages.AddPage(name, p, resize, true)
	a.requestSync()
}

// closeModal removes a modal page with a full terminal sync to clear artifacts
func (a *App) closeModal(name string) {
	a.pages.RemovePage(name)
	a.requestSync()
}

// getTableCellText safely retrieves cell text, returning "" if cell is nil.
func (a *App) getTableCellText(row, col int) string {
	cell := a.table.GetCell(row, col)
	if cell == nil {
		return ""
	}
	return cell.Text
}

// getCurrentContext returns the current k8s context name
func (a *App) getCurrentContext() string {
	if a.k8s == nil {
		return ""
	}
	ctxName, _, _, err := a.k8s.GetContextInfo()
	if err != nil {
		return ""
	}
	return ctxName
}
