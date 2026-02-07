package ui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/db"
	"github.com/rivo/tview"
)

// portForwardInfo tracks a running port-forward process
type portForwardInfo struct {
	Cmd        *exec.Cmd
	Namespace  string
	Name       string
	LocalPort  string
	RemotePort string
}

// checkTUIPermission checks RBAC authorization for TUI actions
// Returns true if allowed, false if denied (with flash error shown)
func (a *App) checkTUIPermission(resource, action string) bool {
	if a.authorizer == nil {
		return true // No authorizer configured, allow all (backward compatible)
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
		db.RecordAudit(db.AuditEntry{
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

	db.RecordAudit(entry)
}

// showLogs shows logs for selected pod with Vim-style navigation
func (a *App) showLogs() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	if resource != "pods" && resource != "po" {
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	ns := a.table.GetCell(row, 0).Text
	name := a.table.GetCell(row, 1).Text

	// Use VimViewer for Vim-style navigation and search
	logView := NewVimViewer(a, "logs",
		fmt.Sprintf(" Logs: %s/%s [gray](Esc:close /search n/N:next/prev Ctrl+D/U:scroll)[white] ", ns, name))

	logView.SetContent("[yellow]Loading...[white]")

	a.pages.AddPage("logs", logView, true, true)
	a.SetFocus(logView)

	// Fetch logs
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		logs, err := a.k8s.GetPodLogs(ctx, ns, name, "", 100)
		a.QueueUpdateDraw(func() {
			if err != nil {
				logView.SetContent(fmt.Sprintf("[red]Error: %v", err))
			} else if logs == "" {
				logView.SetContent("[gray]No logs available")
			} else {
				logView.SetContent(logs)
				logView.ScrollToEnd()
			}
		})
	}()
}

// describeResource shows YAML for selected resource
func (a *App) describeResource() {
	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	var ns, name string
	switch resource {
	case "nodes", "no", "namespaces", "ns":
		name = a.table.GetCell(row, 0).Text
	default:
		ns = a.table.GetCell(row, 0).Text
		name = a.table.GetCell(row, 1).Text
	}

	descView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	descView.SetBorder(true).
		SetTitle(fmt.Sprintf(" %s: %s (Press Esc to close) ", resource, name))

	a.pages.AddPage("describe", descView, true, true)
	a.SetFocus(descView)

	// Fetch YAML
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		gvr, ok := a.k8s.GetGVR(resource)
		if !ok {
			a.QueueUpdateDraw(func() {
				descView.SetText(fmt.Sprintf("[red]Unknown resource type: %s", resource))
			})
			return
		}

		yaml, err := a.k8s.GetResourceYAML(ctx, ns, name, gvr)
		a.QueueUpdateDraw(func() {
			if err != nil {
				descView.SetText(fmt.Sprintf("[red]Error: %v", err))
			} else {
				descView.SetText(yaml)
			}
		})
	}()

	descView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			a.pages.RemovePage("describe")
			a.SetFocus(a.table)
			return nil
		}
		return event
	})
}

// confirmDelete shows a delete confirmation dialog
func (a *App) confirmDelete() {
	a.mx.RLock()
	resource := a.currentResource
	selectedCount := len(a.selectedRows)
	a.mx.RUnlock()

	// RBAC check
	if !a.checkTUIPermission(resource, "delete") {
		return
	}

	// Check for multi-select deletion
	if selectedCount > 0 {
		a.confirmDeleteMultiple()
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	// Get resource info
	var ns, name string
	switch resource {
	case "nodes", "no", "namespaces", "ns":
		name = a.table.GetCell(row, 0).Text
	default:
		ns = a.table.GetCell(row, 0).Text
		name = a.table.GetCell(row, 1).Text
	}

	// Create confirmation modal
	modal := tview.NewModal().
		SetText(fmt.Sprintf("[red]Delete %s?[white]\n\n%s/%s\n\nThis action cannot be undone.", resource, ns, name)).
		AddButtons([]string{"Cancel", "Delete"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.pages.RemovePage("delete-confirm")
			a.SetFocus(a.table)

			if buttonLabel == "Delete" {
				go a.deleteResource(ns, name, resource)
			}
		})

	modal.SetBackgroundColor(tcell.ColorDarkRed)

	a.pages.AddPage("delete-confirm", modal, true, true)
}

// confirmDeleteMultiple confirms deletion of multiple selected resources (k9s style)
func (a *App) confirmDeleteMultiple() {
	a.mx.RLock()
	resource := a.currentResource
	selectedCount := len(a.selectedRows)
	selectedRowsCopy := make(map[int]bool)
	for k, v := range a.selectedRows {
		selectedRowsCopy[k] = v
	}
	a.mx.RUnlock()

	if selectedCount == 0 {
		return
	}

	// Build list of resources to delete
	var items []struct{ ns, name string }
	for row := range selectedRowsCopy {
		var ns, name string
		switch resource {
		case "nodes", "no", "namespaces", "ns":
			name = strings.TrimSpace(tview.TranslateANSI(a.table.GetCell(row, 0).Text))
		default:
			ns = strings.TrimSpace(tview.TranslateANSI(a.table.GetCell(row, 0).Text))
			name = strings.TrimSpace(tview.TranslateANSI(a.table.GetCell(row, 1).Text))
		}
		if name != "" {
			items = append(items, struct{ ns, name string }{ns, name})
		}
	}

	// Create confirmation modal
	modal := tview.NewModal().
		SetText(fmt.Sprintf("[red]Delete %d %s?[white]\n\nThis action cannot be undone.", len(items), resource)).
		AddButtons([]string{"Cancel", "Delete All"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.pages.RemovePage("delete-confirm")
			a.SetFocus(a.table)

			if buttonLabel == "Delete All" {
				go func() {
					for _, item := range items {
						a.deleteResource(item.ns, item.name, resource)
					}
					a.clearSelections()
					a.refresh()
				}()
			}
		})

	modal.SetBackgroundColor(tcell.ColorDarkRed)

	a.pages.AddPage("delete-confirm", modal, true, true)
}

// deleteResource deletes the specified resource
func (a *App) deleteResource(ns, name, resource string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	gvr, ok := a.k8s.GetGVR(resource)
	if !ok {
		a.flashMsg(fmt.Sprintf("Unknown resource type: %s", resource), true)
		a.recordTUIAudit("delete", fmt.Sprintf("%s/%s", resource, name), "Unknown resource type", false, "Unknown resource type")
		return
	}

	a.flashMsg(fmt.Sprintf("Deleting %s/%s...", resource, name), false)

	resourcePath := fmt.Sprintf("%s/%s", resource, name)
	if ns != "" {
		resourcePath = fmt.Sprintf("%s/%s/%s", ns, resource, name)
	}

	err := a.k8s.DeleteResource(ctx, gvr, ns, name)
	if err != nil {
		a.flashMsg(fmt.Sprintf("Delete failed: %v", err), true)
		a.recordTUIAudit("delete", resourcePath, fmt.Sprintf("Failed to delete %s", name), false, err.Error())
		return
	}

	a.flashMsg(fmt.Sprintf("Deleted %s/%s", resource, name), false)
	a.recordTUIAudit("delete", resourcePath, fmt.Sprintf("Deleted %s %s", resource, name), true, "")
	go a.refresh()
}

// execShell opens an interactive shell in the selected pod
func (a *App) execShell() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	if resource != "pods" && resource != "po" {
		a.flashMsg("Shell only available for pods", true)
		return
	}

	// RBAC check
	if !a.checkTUIPermission("pods", "exec") {
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	ns := a.table.GetCell(row, 0).Text
	name := a.table.GetCell(row, 1).Text

	// Suspend the TUI and run kubectl exec
	a.Suspend(func() {
		// Try bash first, fall back to sh
		cmd := exec.Command("kubectl", "exec", "-it", "-n", ns, name, "--", "/bin/bash")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			// Try sh if bash fails
			cmd2 := exec.Command("kubectl", "exec", "-it", "-n", ns, name, "--", "/bin/sh")
			cmd2.Stdin = os.Stdin
			cmd2.Stdout = os.Stdout
			cmd2.Stderr = os.Stderr
			if err2 := cmd2.Run(); err2 != nil {
				fmt.Fprintf(os.Stderr, "\nShell failed: %v\nPress Enter to return...\n", err2)
				bufio.NewReader(os.Stdin).ReadString('\n')
			}
		}
	})
}

// portForward shows port forwarding dialog
func (a *App) portForward() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	if resource != "pods" && resource != "po" && resource != "services" && resource != "svc" {
		a.flashMsg("Port forward only available for pods and services", true)
		return
	}

	// RBAC check
	if !a.checkTUIPermission(resource, "port-forward") {
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	ns := a.table.GetCell(row, 0).Text
	name := a.table.GetCell(row, 1).Text

	// Create port forward dialog
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(fmt.Sprintf(" Port Forward: %s/%s ", ns, name))

	var localPort, remotePort string
	form.AddInputField("Local Port:", "8080", 10, nil, func(text string) {
		localPort = text
	})
	form.AddInputField("Remote Port:", "80", 10, nil, func(text string) {
		remotePort = text
	})
	form.AddButton("Forward", func() {
		a.pages.RemovePage("port-forward")
		a.SetFocus(a.table)

		if localPort == "" || remotePort == "" {
			a.flashMsg("Both ports are required", true)
			return
		}

		go a.startPortForward(ns, name, resource, localPort, remotePort)
	})
	form.AddButton("Cancel", func() {
		a.pages.RemovePage("port-forward")
		a.SetFocus(a.table)
	})

	a.pages.AddPage("port-forward", centered(form, 50, 12), true, true)
}

// startPortForward starts port forwarding in background
func (a *App) startPortForward(ns, name, resource, localPort, remotePort string) {
	resourceType := "pod"
	if resource == "services" || resource == "svc" {
		resourceType = "svc"
	}

	target := fmt.Sprintf("%s/%s", resourceType, name)
	portMap := fmt.Sprintf("%s:%s", localPort, remotePort)

	a.flashMsg(fmt.Sprintf("Starting port forward %s -> %s:%s", localPort, name, remotePort), false)

	cmd := exec.Command("kubectl", "port-forward", "-n", ns, target, portMap)
	err := cmd.Start()
	if err != nil {
		a.flashMsg(fmt.Sprintf("Port forward failed: %v", err), true)
		return
	}

	// Track the port-forward process
	pf := &portForwardInfo{
		Cmd:        cmd,
		Namespace:  ns,
		Name:       name,
		LocalPort:  localPort,
		RemotePort: remotePort,
	}
	a.pfMx.Lock()
	a.portForwards = append(a.portForwards, pf)
	a.pfMx.Unlock()

	// Wait for process to exit in background and clean up
	go func() {
		cmd.Wait()
		a.pfMx.Lock()
		for i, p := range a.portForwards {
			if p == pf {
				a.portForwards = append(a.portForwards[:i], a.portForwards[i+1:]...)
				break
			}
		}
		a.pfMx.Unlock()
	}()

	a.flashMsg(fmt.Sprintf("Port forward active: localhost:%s -> %s:%s (PID: %d)", localPort, name, remotePort, cmd.Process.Pid), false)
}

// showPortForwards displays active port-forward processes
func (a *App) showPortForwards() {
	a.pfMx.Lock()
	forwards := make([]*portForwardInfo, len(a.portForwards))
	copy(forwards, a.portForwards)
	a.pfMx.Unlock()

	if len(forwards) == 0 {
		a.flashMsg("No active port-forwards", false)
		return
	}

	list := tview.NewList()
	list.SetBorder(true).SetTitle(fmt.Sprintf(" Active Port Forwards (%d) - Enter to stop, Esc to close ", len(forwards)))

	for i, pf := range forwards {
		pid := 0
		if pf.Cmd.Process != nil {
			pid = pf.Cmd.Process.Pid
		}
		list.AddItem(
			fmt.Sprintf("localhost:%s -> %s/%s:%s", pf.LocalPort, pf.Namespace, pf.Name, pf.RemotePort),
			fmt.Sprintf("PID: %d", pid),
			rune('1'+i),
			nil,
		)
	}

	list.SetSelectedFunc(func(idx int, _, _ string, _ rune) {
		if idx < len(forwards) {
			pf := forwards[idx]
			if pf.Cmd.Process != nil {
				pf.Cmd.Process.Kill()
				a.flashMsg(fmt.Sprintf("Stopped port-forward localhost:%s", pf.LocalPort), false)
			}
		}
		a.pages.RemovePage("portforwards")
		a.SetFocus(a.table)
	})

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			a.pages.RemovePage("portforwards")
			a.SetFocus(a.table)
			return nil
		}
		return event
	})

	a.pages.AddPage("portforwards", centered(list, 65, 20), true, true)
	a.SetFocus(list)
}

// cleanupPortForwards kills all active port-forward processes
func (a *App) cleanupPortForwards() {
	a.pfMx.Lock()
	defer a.pfMx.Unlock()
	for _, pf := range a.portForwards {
		if pf.Cmd.Process != nil {
			pf.Cmd.Process.Kill()
		}
	}
	a.portForwards = nil
}

// showContextSwitcher displays context selection dialog
func (a *App) showContextSwitcher() {
	if a.k8s == nil {
		a.flashMsg("K8s client not available", true)
		return
	}

	contexts, currentCtx, err := a.k8s.ListContexts()
	if err != nil {
		a.flashMsg(fmt.Sprintf("Failed to list contexts: %v", err), true)
		return
	}

	list := tview.NewList()
	list.SetBorder(true).SetTitle(" Switch Context (Enter to select, Esc to cancel) ")

	for i, ctx := range contexts {
		prefix := "  "
		if ctx == currentCtx {
			prefix = "* "
		}
		list.AddItem(prefix+ctx, "", rune('1'+i), nil)
	}

	list.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		selectedCtx := contexts[index]
		a.pages.RemovePage("context-switcher")
		a.SetFocus(a.table)

		if selectedCtx == currentCtx {
			return
		}

		go func() {
			a.flashMsg(fmt.Sprintf("Switching to context: %s...", selectedCtx), false)
			err := a.k8s.SwitchContext(selectedCtx)
			if err != nil {
				a.flashMsg(fmt.Sprintf("Failed to switch context: %v", err), true)
				return
			}

			a.flashMsg(fmt.Sprintf("Switched to context: %s", selectedCtx), false)
			a.updateHeader()
			a.refresh()
		}()
	})

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			a.pages.RemovePage("context-switcher")
			a.SetFocus(a.table)
			return nil
		}
		return event
	})

	a.pages.AddPage("context-switcher", centered(list, 60, min(len(contexts)+4, 20)), true, true)
}

// showHealth displays system health status
func (a *App) showHealth() {
	health := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	health.SetBorder(true).SetTitle(" System Health (Press Esc to close) ")

	var sb strings.Builder
	sb.WriteString(" [yellow::b]k13d Health Status[white::-]\n\n")

	// K8s connectivity
	if a.k8s != nil {
		ctxName, cluster, _, err := a.k8s.GetContextInfo()
		if err != nil {
			sb.WriteString(" [red]✗[white] Kubernetes: Not connected\n")
		} else {
			sb.WriteString(fmt.Sprintf(" [green]✓[white] Kubernetes: Connected\n"))
			sb.WriteString(fmt.Sprintf("   Context: %s\n", ctxName))
			sb.WriteString(fmt.Sprintf("   Cluster: %s\n", cluster))
		}
	} else {
		sb.WriteString(" [red]✗[white] Kubernetes: Client not initialized\n")
	}

	sb.WriteString("\n")

	// AI status
	if a.aiClient != nil && a.aiClient.IsReady() {
		sb.WriteString(fmt.Sprintf(" [green]✓[white] AI: Online (%s)\n", a.aiClient.GetModel()))
	} else {
		sb.WriteString(" [red]✗[white] AI: Offline\n")
		sb.WriteString("   Configure in ~/.kube-ai-dashboard/config.yaml\n")
	}

	sb.WriteString("\n")

	// Config
	if a.config != nil {
		sb.WriteString(fmt.Sprintf(" [gray]Language:[white] %s\n", a.config.Language))
	}

	sb.WriteString("\n [gray]Press Esc to close[white]")

	health.SetText(sb.String())

	health.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			a.pages.RemovePage("health")
			a.SetFocus(a.table)
			return nil
		}
		return event
	})

	a.pages.AddPage("health", centered(health, 60, 18), true, true)
}

// showAbout displays about modal with logo
func (a *App) showAbout() {
	about := AboutModal()
	a.pages.AddPage("about", centered(about, 60, 35), true, true)
	a.SetFocus(about)

	about.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc || event.Rune() == 'q' {
			a.pages.RemovePage("about")
			a.SetFocus(a.table)
			return nil
		}
		return event
	})
}

// showHelp displays help modal
func (a *App) showHelp() {
	helpText := fmt.Sprintf(`
%s
[gray]k9s compatible keybindings with AI assistance[white]

[cyan::b]GENERAL[white::-]
  [yellow]:[white]        Command mode        [yellow]?[white]        Help
  [yellow]/[white]        Filter mode         [yellow]Esc[white]      Back/Clear/Cancel
  [yellow]Tab[white]      AI Panel focus      [yellow]Enter[white]    Select/Drill-down
  [yellow]Ctrl+E[white]   Toggle AI panel     [yellow]Shift+O[white]  Settings/LLM Config
  [yellow]q/Ctrl+C[white] Quit application

[cyan::b]NAVIGATION[white::-]
  [yellow]j/Down[white]   Down                [yellow]k/Up[white]     Up
  [yellow]g[white]        Top                 [yellow]G[white]        Bottom
  [yellow]Ctrl+F[white]   Page down           [yellow]Ctrl+B[white]   Page up
  [yellow]Ctrl+D[white]   Half page down      [yellow]Ctrl+U[white]   Half page up

[cyan::b]RESOURCE ACTIONS[white::-]
  [yellow]d[white]        Describe            [yellow]y[white]        YAML view
  [yellow]e[white]        Edit ($EDITOR)      [yellow]Ctrl+D[white]   Delete
  [yellow]r[white]        Refresh             [yellow]c[white]        Switch context
  [yellow]n[white]        Cycle namespace     [yellow]Space[white]    Multi-select

[cyan::b]NAMESPACE SHORTCUTS[white::-] (k9s style)
  [yellow]0[white] All namespaces      [yellow]n[white]   Cycle through namespaces
  [yellow]u[white] Use namespace (on namespace view)
  [yellow]:ns <name>[white]           Switch to specific namespace

[cyan::b]POD ACTIONS[white::-]
  [yellow]l[white]        Logs                [yellow]p[white]        Previous logs
  [yellow]s[white]        Shell               [yellow]a[white]        Attach
  [yellow]o[white]        Show node           [yellow]k/Ctrl+K[white] Kill (force delete)
  [yellow]Shift+F[white]  Port forward        [yellow]f[white]        Show port-forward

[cyan::b]WORKLOAD ACTIONS[white::-] (Deploy/StatefulSet/DaemonSet/ReplicaSet)
  [yellow]S[white]        Scale               [yellow]R[white]        Restart/Rollout
  [yellow]z[white]        Show ReplicaSets    [yellow]Enter[white]    Show Pods

[cyan::b]VIEWER (Logs/Describe/YAML)[white::-] - Vim-style navigation
  [yellow]j/k[white]      Scroll down/up      [yellow]g/G[white]      Top/Bottom
  [yellow]Ctrl+D[white]   Half page down      [yellow]Ctrl+U[white]   Half page up
  [yellow]Ctrl+F[white]   Full page down      [yellow]Ctrl+B[white]   Full page up
  [yellow]/[white]        Search mode         [yellow]n/N[white]      Next/Prev match
  [yellow]q/Esc[white]    Close viewer

[cyan::b]COMMAND EXAMPLES[white::-] (press : to enter command mode)
  [yellow]:pods[white] [yellow]:po[white]              List pods
  [yellow]:pods -n kube-system[white]  List pods in specific namespace
  [yellow]:pods -A[white]              List pods in all namespaces
  [yellow]:deploy[white] [yellow]:dp[white]            List deployments
  [yellow]:svc[white] [yellow]:services[white]         List services
  [yellow]:ns kube-system[white]       Switch to namespace
  [yellow]:ctx[white] [yellow]:context[white]          Switch context

[cyan::b]AI ASSISTANT[white::-] (Tab to focus, type and press Enter)
  Ask natural language questions or request kubectl commands:
  - "Show me all pods in kube-system namespace"
  - "Why is my pod crashing?"
  - "Scale deployment nginx to 3 replicas"
  - "Show recent events for this deployment"

  [gray]AI will suggest commands. Press Y to execute, N to cancel.[white]

[gray]Press Esc, q, or ? to close this help[white]
`, LogoColors())

	help := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetText(helpText)
	help.SetBorder(true).SetTitle(" Help ")

	a.pages.AddPage("help", centered(help, 75, 55), true, true)
	a.SetFocus(help)

	help.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc || event.Rune() == 'q' || event.Rune() == '?' {
			a.pages.RemovePage("help")
			a.SetFocus(a.table)
			return nil
		}
		return event
	})
}

// showYAML shows YAML for selected resource (k9s y key) with Vim-style navigation
func (a *App) showYAML() {
	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	var ns, name string
	switch resource {
	case "nodes", "no", "namespaces", "ns", "persistentvolumes", "storageclasses",
		"clusterroles", "clusterrolebindings", "customresourcedefinitions":
		name = a.table.GetCell(row, 0).Text
	default:
		ns = a.table.GetCell(row, 0).Text
		name = a.table.GetCell(row, 1).Text
	}

	// Use VimViewer for Vim-style navigation and search
	yamlView := NewVimViewer(a, "yaml",
		fmt.Sprintf(" YAML: %s/%s [gray](Esc:close /search n/N:next/prev Ctrl+D/U:scroll)[white] ", resource, name))

	yamlView.SetContent("[yellow]Loading...[white]")

	a.pages.AddPage("yaml", yamlView, true, true)
	a.SetFocus(yamlView)

	// Fetch YAML
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		gvr, ok := a.k8s.GetGVR(resource)
		if !ok {
			a.QueueUpdateDraw(func() {
				yamlView.SetContent(fmt.Sprintf("[red]Unknown resource type: %s", resource))
			})
			return
		}

		yaml, err := a.k8s.GetResourceYAML(ctx, ns, name, gvr)
		a.QueueUpdateDraw(func() {
			if err != nil {
				yamlView.SetContent(fmt.Sprintf("[red]Error: %v", err))
			} else {
				yamlView.SetContent(yaml)
			}
		})
	}()
}

// showLogsPrevious shows logs for previous container (k9s p key) with Vim-style navigation
func (a *App) showLogsPrevious() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	if resource != "pods" && resource != "po" {
		a.flashMsg("Logs only available for pods", true)
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	ns := a.table.GetCell(row, 0).Text
	name := a.table.GetCell(row, 1).Text

	// Use VimViewer for Vim-style navigation and search
	logView := NewVimViewer(a, "logs",
		fmt.Sprintf(" Previous Logs: %s/%s [gray](Esc:close /search n/N:next/prev Ctrl+D/U:scroll)[white] ", ns, name))

	logView.SetContent("[yellow]Loading...[white]")

	a.pages.AddPage("logs", logView, true, true)
	a.SetFocus(logView)

	// Fetch previous logs
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		logs, err := a.k8s.GetPodLogsPrevious(ctx, ns, name, "", 100)
		a.QueueUpdateDraw(func() {
			if err != nil {
				logView.SetContent(fmt.Sprintf("[red]Error: %v", err))
			} else if logs == "" {
				logView.SetContent("[gray]No previous logs available")
			} else {
				logView.SetContent(logs)
				logView.ScrollToEnd()
			}
		})
	}()
}

// editResource opens the resource in $EDITOR (k9s e key)
func (a *App) editResource() {
	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	var ns, name string
	switch resource {
	case "nodes", "no", "namespaces", "ns", "persistentvolumes", "storageclasses",
		"clusterroles", "clusterrolebindings", "customresourcedefinitions":
		name = a.table.GetCell(row, 0).Text
	default:
		ns = a.table.GetCell(row, 0).Text
		name = a.table.GetCell(row, 1).Text
	}

	resourcePath := fmt.Sprintf("%s/%s", resource, name)
	if ns != "" {
		resourcePath = fmt.Sprintf("%s/%s/%s", ns, resource, name)
	}

	// Suspend TUI and run kubectl edit
	a.Suspend(func() {
		var cmd *exec.Cmd
		if ns != "" {
			cmd = exec.Command("kubectl", "edit", resource, name, "-n", ns)
		} else {
			cmd = exec.Command("kubectl", "edit", resource, name)
		}
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "\nEdit failed: %v\nPress Enter to return...\n", err)
			bufio.NewReader(os.Stdin).ReadString('\n')
		}
	})

	// Record audit (we can't know if edit was actually saved, so just record the attempt)
	a.recordTUIAudit("edit", resourcePath, fmt.Sprintf("Edited %s %s via $EDITOR", resource, name), true, "")

	// Refresh after edit
	go a.refresh()
}

// attachContainer attaches to a container (k9s a key)
func (a *App) attachContainer() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	if resource != "pods" && resource != "po" {
		a.flashMsg("Attach only available for pods", true)
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	ns := a.table.GetCell(row, 0).Text
	name := a.table.GetCell(row, 1).Text

	// Suspend TUI and run kubectl attach
	a.Suspend(func() {
		cmd := exec.Command("kubectl", "attach", "-it", "-n", ns, name)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "\nAttach failed: %v\nPress Enter to return...\n", err)
			bufio.NewReader(os.Stdin).ReadString('\n')
		}
	})
}

// useNamespace switches to the selected namespace (k9s u key)
func (a *App) useNamespace() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	if resource != "namespaces" && resource != "ns" {
		a.flashMsg("Use 'u' only on namespaces view", true)
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	nsName := a.table.GetCell(row, 0).Text

	a.mx.Lock()
	a.currentNamespace = nsName
	a.currentResource = "pods"
	a.mx.Unlock()

	// Force full terminal sync to prevent ghosting during namespace switch
	a.requestSync()

	// Clear table immediately to prevent visual artifacts
	a.QueueUpdateDraw(func() {
		a.table.Clear()
		a.table.SetTitle(" pods - Loading... ")
		a.table.SetCell(0, 0, tview.NewTableCell("Loading...").SetTextColor(tcell.ColorYellow))
	})

	a.flashMsg(fmt.Sprintf("Switched to namespace: %s", nsName), false)

	go func() {
		a.updateHeader()
		a.refresh()
	}()
}

// showNode shows the node where the selected pod is running (k9s o key)
func (a *App) showNode() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	if resource != "pods" && resource != "po" {
		a.flashMsg("Show node only available for pods", true)
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	ns := a.table.GetCell(row, 0).Text
	name := a.table.GetCell(row, 1).Text

	// Get pod to find node
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pods, err := a.k8s.ListPods(ctx, ns)
	if err != nil {
		a.flashMsg(fmt.Sprintf("Error: %v", err), true)
		return
	}

	var nodeName string
	for _, pod := range pods {
		if pod.Name == name {
			nodeName = pod.Spec.NodeName
			break
		}
	}

	if nodeName == "" {
		a.flashMsg("Pod not scheduled to a node yet", true)
		return
	}

	// Save current state and navigate to nodes with filter
	a.mx.Lock()
	a.navMx.Lock()
	a.navigationStack = append(a.navigationStack, navHistory{resource, a.currentNamespace, a.filterText})
	a.navMx.Unlock()
	a.currentResource = "nodes"
	a.currentNamespace = ""
	a.filterText = nodeName
	a.mx.Unlock()

	a.requestSync()

	go func() {
		a.updateHeader()
		a.refresh()
	}()
}

// killPod force deletes a pod (k9s k or Ctrl+K key)
func (a *App) killPod() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	if resource != "pods" && resource != "po" {
		a.flashMsg("Kill only available for pods", true)
		return
	}

	// RBAC check
	if !a.checkTUIPermission("pods", "delete") {
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	ns := a.table.GetCell(row, 0).Text
	name := a.table.GetCell(row, 1).Text

	modal := tview.NewModal().
		SetText(fmt.Sprintf("[red]Kill pod?[white]\n\n%s/%s\n\nThis will force delete the pod.", ns, name)).
		AddButtons([]string{"Cancel", "Kill"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.pages.RemovePage("kill-confirm")
			a.SetFocus(a.table)

			if buttonLabel == "Kill" {
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()

					a.flashMsg(fmt.Sprintf("Killing pod %s/%s...", ns, name), false)

					resourcePath := fmt.Sprintf("%s/pod/%s", ns, name)
					err := a.k8s.DeletePodForce(ctx, ns, name)
					if err != nil {
						a.flashMsg(fmt.Sprintf("Kill failed: %v", err), true)
						a.recordTUIAudit("kill", resourcePath, fmt.Sprintf("Failed to force delete pod %s", name), false, err.Error())
						return
					}

					a.flashMsg(fmt.Sprintf("Killed pod %s/%s", ns, name), false)
					a.recordTUIAudit("kill", resourcePath, fmt.Sprintf("Force deleted pod %s", name), true, "")
					a.refresh()
				}()
			}
		})

	modal.SetBackgroundColor(tcell.ColorDarkRed)
	a.pages.AddPage("kill-confirm", modal, true, true)
}

// showBenchmark runs benchmark on service (k9s b key) - placeholder
func (a *App) showBenchmark() {
	a.flashMsg("Benchmark feature not yet implemented", true)
}

// triggerCronJob manually triggers a cronjob (k9s t key)
func (a *App) triggerCronJob() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	if resource != "cronjobs" && resource != "cj" {
		a.flashMsg("Trigger only available for cronjobs", true)
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	ns := a.table.GetCell(row, 0).Text
	name := a.table.GetCell(row, 1).Text

	modal := tview.NewModal().
		SetText(fmt.Sprintf("Trigger CronJob?\n\n%s/%s\n\nThis will create a new job from this cronjob.", ns, name)).
		AddButtons([]string{"Cancel", "Trigger"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.pages.RemovePage("trigger-confirm")
			a.SetFocus(a.table)

			if buttonLabel == "Trigger" {
				go func() {
					a.flashMsg(fmt.Sprintf("Triggering cronjob %s/%s...", ns, name), false)

					// Use kubectl to create job from cronjob
					jobName := fmt.Sprintf("%s-manual-%d", name, time.Now().Unix())
					resourcePath := fmt.Sprintf("%s/cronjob/%s", ns, name)
					cmd := exec.Command("kubectl", "create", "job", jobName, "--from=cronjob/"+name, "-n", ns)
					output, err := cmd.CombinedOutput()
					if err != nil {
						a.flashMsg(fmt.Sprintf("Trigger failed: %s", string(output)), true)
						a.recordTUIAudit("trigger", resourcePath, fmt.Sprintf("Failed to trigger cronjob %s", name), false, string(output))
						return
					}

					a.flashMsg(fmt.Sprintf("Created job %s from cronjob %s", jobName, name), false)
					a.recordTUIAudit("trigger", resourcePath, fmt.Sprintf("Triggered cronjob %s, created job %s", name, jobName), true, "")
					a.refresh()
				}()
			}
		})

	a.pages.AddPage("trigger-confirm", modal, true, true)
}

// showRelatedResource shows related resources (k9s z key)
func (a *App) showRelatedResource() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	var ns, name string
	switch resource {
	case "nodes", "namespaces", "persistentvolumes", "storageclasses",
		"clusterroles", "clusterrolebindings", "customresourcedefinitions":
		name = a.table.GetCell(row, 0).Text
	default:
		ns = a.table.GetCell(row, 0).Text
		name = a.table.GetCell(row, 1).Text
	}

	// Different behavior based on resource type
	switch resource {
	case "deployments", "deploy":
		// Show ReplicaSets
		a.mx.Lock()
		a.navMx.Lock()
		a.navigationStack = append(a.navigationStack, navHistory{resource, a.currentNamespace, a.filterText})
		a.navMx.Unlock()
		a.currentResource = "replicasets"
		a.currentNamespace = ns
		a.filterText = name
		a.mx.Unlock()
		go func() {
			a.updateHeader()
			a.refresh()
		}()

	default:
		a.flashMsg(fmt.Sprintf("No related resources for %s", resource), true)
	}
}

// scaleResource scales a deployment/statefulset (k9s Shift+S key)
func (a *App) scaleResource() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	// Only scalable resources
	scalable := map[string]bool{
		"deployments": true, "deploy": true,
		"statefulsets": true, "sts": true,
		"replicasets": true, "rs": true,
	}

	if !scalable[resource] {
		a.flashMsg("Scale only available for deployments, statefulsets, replicasets", true)
		return
	}

	// RBAC check
	if !a.checkTUIPermission(resource, "scale") {
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	ns := a.table.GetCell(row, 0).Text
	name := a.table.GetCell(row, 1).Text

	// Create scale dialog
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(fmt.Sprintf(" Scale: %s/%s ", ns, name))

	var replicas string
	form.AddInputField("Replicas:", "1", 10, tview.InputFieldInteger, func(text string) {
		replicas = text
	})
	form.AddButton("Scale", func() {
		// Validate replica count
		n, err := fmt.Sscanf(replicas, "%d", new(int))
		if n != 1 || err != nil {
			a.flashMsg("Invalid replica count: must be a number", true)
			return
		}
		var replicaCount int
		fmt.Sscanf(replicas, "%d", &replicaCount)
		if replicaCount < 0 || replicaCount > 999 {
			a.flashMsg("Replica count must be between 0 and 999", true)
			return
		}

		a.pages.RemovePage("scale-dialog")
		a.SetFocus(a.table)

		go func() {
			a.flashMsg(fmt.Sprintf("Scaling %s/%s to %s replicas...", ns, name, replicas), false)

			resourceType := resource
			if resourceType == "deploy" {
				resourceType = "deployment"
			} else if resourceType == "sts" {
				resourceType = "statefulset"
			} else if resourceType == "rs" {
				resourceType = "replicaset"
			}

			resourcePath := fmt.Sprintf("%s/%s/%s", ns, resourceType, name)
			cmd := exec.Command("kubectl", "scale", resourceType, name, "-n", ns, "--replicas="+replicas)
			output, err := cmd.CombinedOutput()
			if err != nil {
				a.flashMsg(fmt.Sprintf("Scale failed: %s", string(output)), true)
				a.recordTUIAudit("scale", resourcePath, fmt.Sprintf("Failed to scale to %s replicas", replicas), false, string(output))
				return
			}

			a.flashMsg(fmt.Sprintf("Scaled %s/%s to %s replicas", ns, name, replicas), false)
			a.recordTUIAudit("scale", resourcePath, fmt.Sprintf("Scaled to %s replicas", replicas), true, "")
			a.refresh()
		}()
	})
	form.AddButton("Cancel", func() {
		a.pages.RemovePage("scale-dialog")
		a.SetFocus(a.table)
	})

	a.pages.AddPage("scale-dialog", centered(form, 50, 10), true, true)
}

// restartResource restarts a deployment/statefulset (k9s Shift+R key)
func (a *App) restartResource() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	restartable := map[string]bool{
		"deployments": true, "deploy": true,
		"statefulsets": true, "sts": true,
		"daemonsets": true, "ds": true,
	}

	if !restartable[resource] {
		a.flashMsg("Restart only available for deployments, statefulsets, daemonsets", true)
		return
	}

	// RBAC check
	if !a.checkTUIPermission(resource, "restart") {
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	ns := a.table.GetCell(row, 0).Text
	name := a.table.GetCell(row, 1).Text

	modal := tview.NewModal().
		SetText(fmt.Sprintf("Restart %s?\n\n%s/%s\n\nThis will trigger a rolling restart.", resource, ns, name)).
		AddButtons([]string{"Cancel", "Restart"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.pages.RemovePage("restart-confirm")
			a.SetFocus(a.table)

			if buttonLabel == "Restart" {
				go func() {
					a.flashMsg(fmt.Sprintf("Restarting %s/%s...", ns, name), false)

					resourceType := resource
					if resourceType == "deploy" {
						resourceType = "deployment"
					} else if resourceType == "sts" {
						resourceType = "statefulset"
					} else if resourceType == "ds" {
						resourceType = "daemonset"
					}

					resourcePath := fmt.Sprintf("%s/%s/%s", ns, resourceType, name)
					cmd := exec.Command("kubectl", "rollout", "restart", resourceType, name, "-n", ns)
					output, err := cmd.CombinedOutput()
					if err != nil {
						a.flashMsg(fmt.Sprintf("Restart failed: %s", string(output)), true)
						a.recordTUIAudit("restart", resourcePath, fmt.Sprintf("Failed to rollout restart %s", name), false, string(output))
						return
					}

					a.flashMsg(fmt.Sprintf("Restarted %s/%s", ns, name), false)
					a.recordTUIAudit("restart", resourcePath, fmt.Sprintf("Rollout restart %s", name), true, "")
					a.refresh()
				}()
			}
		})

	a.pages.AddPage("restart-confirm", modal, true, true)
}

// showDescribe shows describe output for selected resource (like kubectl describe) with Vim-style navigation
func (a *App) showDescribe() {
	row, _ := a.table.GetSelection()
	if row <= 0 {
		a.flashMsg("No resource selected", true)
		return
	}

	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	// Get namespace and name from table
	nsCell := a.table.GetCell(row, 0)
	nameCell := a.table.GetCell(row, 1)
	if nsCell == nil || nameCell == nil {
		a.flashMsg("Cannot get resource info", true)
		return
	}

	ns := nsCell.Text
	name := nameCell.Text

	// For cluster-scoped resources, name is in column 0
	if resource == "nodes" || resource == "namespaces" || resource == "persistentvolumes" ||
		resource == "storageclasses" || resource == "clusterroles" || resource == "clusterrolebindings" ||
		resource == "customresourcedefinitions" {
		name = ns
		ns = ""
	}

	// Use VimViewer for Vim-style navigation and search
	descView := NewVimViewer(a, "describe",
		fmt.Sprintf(" Describe: %s/%s [gray](Esc:close /search n/N:next/prev Ctrl+D/U:scroll)[white] ", resource, name))

	descView.SetContent("[yellow]Loading...[white]")

	// Add to pages
	a.pages.AddPage("describe", descView, true, true)
	a.SetFocus(descView)

	// Fetch describe output in background
	go func() {
		ctx := a.prepareContext()
		output, err := a.k8s.DescribeResource(ctx, resource, ns, name)
		if err != nil {
			a.QueueUpdateDraw(func() {
				descView.SetContent(fmt.Sprintf("[red]Error: %v[white]", err))
			})
			return
		}

		a.QueueUpdateDraw(func() {
			descView.SetContent(output)
			descView.ScrollToBeginning()
		})
	}()
}

// toggleSelection toggles selection of the current row (k9s Space key)
func (a *App) toggleSelection() {
	row, _ := a.table.GetSelection()
	if row <= 0 { // Skip header row
		return
	}

	a.mx.Lock()
	if a.selectedRows[row] {
		delete(a.selectedRows, row)
	} else {
		a.selectedRows[row] = true
	}
	selectedCount := len(a.selectedRows)
	a.mx.Unlock()

	// Update row visual
	a.updateRowSelection(row)

	// Move to next row
	rowCount := a.table.GetRowCount()
	if row < rowCount-1 {
		a.table.Select(row+1, 0)
	}

	// Update status bar with selection count
	if selectedCount > 0 {
		a.flashMsg(fmt.Sprintf("%d item(s) selected - Ctrl+D to delete selected", selectedCount), false)
	}
}

// updateRowSelection updates visual styling for a row based on selection state
func (a *App) updateRowSelection(row int) {
	a.mx.RLock()
	isSelected := a.selectedRows[row]
	a.mx.RUnlock()

	colCount := a.table.GetColumnCount()
	for col := 0; col < colCount; col++ {
		cell := a.table.GetCell(row, col)
		if cell != nil {
			if isSelected {
				// Highlight selected rows with cyan background
				cell.SetBackgroundColor(tcell.ColorDarkCyan)
				cell.SetTextColor(tcell.ColorWhite)
			} else {
				// Reset to default
				cell.SetBackgroundColor(tcell.ColorDefault)
				cell.SetTextColor(tcell.ColorWhite)
			}
		}
	}
}

// clearSelections clears all selections
func (a *App) clearSelections() {
	a.mx.Lock()
	for row := range a.selectedRows {
		delete(a.selectedRows, row)
	}
	a.mx.Unlock()

	// Reset all row visuals
	rowCount := a.table.GetRowCount()
	for row := 1; row < rowCount; row++ {
		a.updateRowSelection(row)
	}
}

// getSelectedResources returns names of selected resources (or current if none selected)
func (a *App) getSelectedResources() []string {
	a.mx.RLock()
	selectedCount := len(a.selectedRows)
	a.mx.RUnlock()

	if selectedCount == 0 {
		// No selection, return current row
		row, _ := a.table.GetSelection()
		if row > 0 {
			cell := a.table.GetCell(row, 0)
			if cell != nil {
				name := strings.TrimSpace(tview.TranslateANSI(cell.Text))
				// Handle namespace/name format
				parts := strings.Fields(name)
				if len(parts) > 0 {
					return []string{parts[len(parts)-1]}
				}
			}
		}
		return nil
	}

	// Return all selected resources
	a.mx.RLock()
	defer a.mx.RUnlock()

	var resources []string
	for row := range a.selectedRows {
		cell := a.table.GetCell(row, 0)
		if cell != nil {
			// For namespaced resources, column 0 might be namespace, column 1 is name
			name := strings.TrimSpace(tview.TranslateANSI(cell.Text))
			// Check if there's a second column with name
			if a.table.GetColumnCount() > 1 {
				nameCell := a.table.GetCell(row, 1)
				if nameCell != nil {
					possibleName := strings.TrimSpace(tview.TranslateANSI(nameCell.Text))
					// If first column looks like a namespace, use second column
					if possibleName != "" && !strings.Contains(name, "-") {
						name = possibleName
					}
				}
			}
			if name != "" {
				resources = append(resources, name)
			}
		}
	}
	return resources
}

// toggleBriefing toggles the briefing panel visibility (Shift+B)
func (a *App) toggleBriefing() {
	if a.briefing == nil {
		return
	}

	a.briefing.Toggle()

	if a.briefing.IsVisible() {
		a.flashMsg("Briefing panel enabled", false)
	} else {
		a.flashMsg("Briefing panel hidden", false)
	}
}

// aiBriefing generates an AI-enhanced briefing (Ctrl+I)
func (a *App) aiBriefing() {
	if a.briefing == nil {
		return
	}

	if !a.briefing.IsVisible() {
		a.briefing.Toggle()
	}

	go a.briefing.UpdateWithAI()
}

// showSettings displays settings modal with LLM connection test and save functionality
func (a *App) showSettings() {
	// Create settings form
	form := tview.NewForm()

	// LLM Status indicator
	statusText := "[gray]●[white] LLM Status: Unknown"
	statusView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(statusText)
	statusView.SetBackgroundColor(tcell.ColorDefault)

	// Get current config
	provider := a.config.LLM.Provider
	model := a.config.LLM.Model
	endpoint := a.config.LLM.Endpoint
	apiKey := "" // Don't show existing API key
	hasAPIKey := a.config.LLM.APIKey != ""

	// Provider dropdown
	providers := []string{"openai", "ollama", "solar", "gemini", "anthropic", "bedrock", "azopenai"}
	providerIndex := 0
	for i, p := range providers {
		if p == provider {
			providerIndex = i
			break
		}
	}

	form.AddDropDown("Provider", providers, providerIndex, func(option string, index int) {
		provider = option
		// Auto-fill default endpoints and models for convenience
		switch option {
		case "ollama":
			if endpoint == "" || endpoint == "https://api.openai.com/v1" {
				endpoint = "http://localhost:11434"
				// Update endpoint field
				if item := form.GetFormItemByLabel("Endpoint"); item != nil {
					if input, ok := item.(*tview.InputField); ok {
						input.SetText(endpoint)
					}
				}
			}
			if model == "" || model == "gpt-4o" {
				model = "llama3.2"
				if item := form.GetFormItemByLabel("Model"); item != nil {
					if input, ok := item.(*tview.InputField); ok {
						input.SetText(model)
					}
				}
			}
		case "openai":
			if model == "" || model == "llama3.2" {
				model = "gpt-4o"
				if item := form.GetFormItemByLabel("Model"); item != nil {
					if input, ok := item.(*tview.InputField); ok {
						input.SetText(model)
					}
				}
			}
		case "anthropic":
			if model == "" {
				model = "claude-sonnet-4-20250514"
				if item := form.GetFormItemByLabel("Model"); item != nil {
					if input, ok := item.(*tview.InputField); ok {
						input.SetText(model)
					}
				}
			}
		case "solar":
			if endpoint == "" {
				endpoint = "https://api.upstage.ai/v1"
				if item := form.GetFormItemByLabel("Endpoint"); item != nil {
					if input, ok := item.(*tview.InputField); ok {
						input.SetText(endpoint)
					}
				}
			}
			if model == "" {
				model = "solar-pro2"
				if item := form.GetFormItemByLabel("Model"); item != nil {
					if input, ok := item.(*tview.InputField); ok {
						input.SetText(model)
					}
				}
			}
		}
	})
	form.AddInputField("Model", model, 30, nil, func(text string) {
		model = text
	})
	form.AddInputField("Endpoint", endpoint, 50, nil, func(text string) {
		endpoint = text
	})
	form.AddPasswordField("API Key", "", 50, '*', func(text string) {
		apiKey = text
	})

	// Helper function to update infoView
	updateInfoView := func(infoView *tview.TextView) {
		currentAPIKey := hasAPIKey
		if apiKey != "" {
			currentAPIKey = true
		}
		infoText := fmt.Sprintf(` [cyan::b]LLM Configuration[white::-]
 Provider: [yellow]%s[white]  Model: [yellow]%s[white]
 API Key: %s  Endpoint: %s
`,
			provider, model,
			map[bool]string{true: "[green]Set[white]", false: "[red]Not set[white]"}[currentAPIKey],
			map[bool]string{true: "[green]" + endpoint + "[white]", false: "[gray](default)[white]"}[endpoint != ""])
		infoView.SetText(infoText)
	}

	// Create info view first (we'll reference it in Save button)
	infoText := fmt.Sprintf(` [cyan::b]LLM Configuration[white::-]
 Provider: [yellow]%s[white]  Model: [yellow]%s[white]
 API Key: %s  Endpoint: %s
`,
		provider, model,
		map[bool]string{true: "[green]Set[white]", false: "[red]Not set[white]"}[hasAPIKey],
		map[bool]string{true: "[green]" + endpoint + "[white]", false: "[gray](default)[white]"}[endpoint != ""])

	infoView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(infoText)
	infoView.SetBackgroundColor(tcell.ColorDefault)

	// Add Save button
	form.AddButton("Save", func() {
		statusView.SetText("[yellow]◐[white] Saving configuration...")
		a.QueueUpdateDraw(func() {})

		go func() {
			// Update config
			a.config.LLM.Provider = provider
			a.config.LLM.Model = model
			a.config.LLM.Endpoint = endpoint
			if apiKey != "" {
				a.config.LLM.APIKey = apiKey
				hasAPIKey = true
			}

			// Save config to file
			if err := a.config.Save(); err != nil {
				a.QueueUpdateDraw(func() {
					statusView.SetText(fmt.Sprintf("[red]✗[white] Failed to save: %s", err))
				})
				return
			}

			// Reinitialize AI client with new config
			newClient, err := ai.NewClient(&a.config.LLM)
			if err != nil {
				a.QueueUpdateDraw(func() {
					statusView.SetText(fmt.Sprintf("[yellow]●[white] Saved, but client init failed: %s", err))
				})
				return
			}
			a.aiClient = newClient

			a.QueueUpdateDraw(func() {
				statusView.SetText("[green]●[white] Configuration saved! Press 'Test Connection' to verify")
				updateInfoView(infoView)
				a.updateHeader() // Update AI status in header
			})
		}()
	})

	// Add test connection button
	form.AddButton("Test", func() {
		statusView.SetText("[yellow]◐[white] Testing connection...")
		a.QueueUpdateDraw(func() {})

		go func() {
			var resultText string
			if a.aiClient == nil {
				resultText = "[red]✗[white] LLM Not Configured - Save settings first"
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				status := a.aiClient.TestConnection(ctx)
				if status.Connected {
					resultText = fmt.Sprintf("[green]●[white] Connected! %s/%s (%dms)",
						status.Provider, status.Model, status.ResponseTime)
				} else {
					resultText = fmt.Sprintf("[red]✗[white] Failed: %s", status.Error)
					if status.Message != "" {
						resultText += "\n    [gray]" + status.Message + "[white]"
					}
				}
			}

			a.QueueUpdateDraw(func() {
				statusView.SetText(resultText)
			})
		}()
	})

	form.AddButton("Close", func() {
		a.pages.RemovePage("settings")
		a.SetFocus(a.table)
	})

	// Combine into flex layout
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(infoView, 4, 0, false).
		AddItem(statusView, 2, 0, false).
		AddItem(form, 0, 1, true)

	flex.SetBorder(true).SetTitle(" Settings (Esc to close) ")
	flex.SetBackgroundColor(tcell.ColorDefault)

	// Wrap in centered container
	a.pages.AddPage("settings", centered(flex, 70, 28), true, true)
	a.SetFocus(form)

	// Handle escape
	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			a.pages.RemovePage("settings")
			a.SetFocus(a.table)
			return nil
		}
		return event
	})

	// Check initial status
	go func() {
		var initialStatus string
		if a.aiClient == nil {
			initialStatus = "[gray]●[white] LLM Not Configured - Enter settings and Save"
		} else if a.aiClient.IsReady() {
			initialStatus = fmt.Sprintf("[yellow]●[white] LLM: %s/%s - Press 'Test' to verify",
				a.aiClient.GetProvider(), a.aiClient.GetModel())
		} else {
			initialStatus = "[gray]●[white] LLM Configuration Incomplete - Enter settings and Save"
		}
		a.QueueUpdateDraw(func() {
			statusView.SetText(initialStatus)
		})
	}()
}
