package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Navigation history for back navigation
type navHistory struct {
	resource  string
	namespace string
	filter    string
}

// drillDown navigates to related resources (k9s Enter key behavior)
func (a *App) drillDown() {
	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	a.mx.RLock()
	resource := a.currentResource
	ns := a.currentNamespace
	filter := a.filterText
	a.mx.RUnlock()

	// Save current state to navigation stack (thread-safe)
	a.navMx.Lock()
	a.navigationStack = append(a.navigationStack, navHistory{resource, ns, filter})
	a.navMx.Unlock()

	// Get selected item info
	var selectedNs, selectedName string
	switch resource {
	case "nodes", "namespaces", "persistentvolumes", "storageclasses",
		"clusterroles", "clusterrolebindings", "customresourcedefinitions":
		selectedName = a.table.GetCell(row, 0).Text
	default:
		selectedNs = a.table.GetCell(row, 0).Text
		selectedName = a.table.GetCell(row, 1).Text
	}

	// Determine drill-down behavior based on resource type
	// Use navigateTo() for deadlock-safe state transitions
	switch resource {
	case "pods", "po":
		// Pod -> Show logs (container view)
		a.showLogs()
		return

	case "deployments", "deploy":
		// Deployment -> Pods with label selector
		a.navigateTo("pods", selectedNs, selectedName)

	case "services", "svc":
		// Service -> Pods (show related pods)
		a.navigateTo("pods", selectedNs, selectedName)

	case "replicasets", "rs":
		// ReplicaSet -> Pods
		a.navigateTo("pods", selectedNs, selectedName)

	case "statefulsets", "sts":
		// StatefulSet -> Pods
		a.navigateTo("pods", selectedNs, selectedName)

	case "daemonsets", "ds":
		// DaemonSet -> Pods
		a.navigateTo("pods", selectedNs, selectedName)

	case "jobs", "job":
		// Job -> Pods
		a.navigateTo("pods", selectedNs, selectedName)

	case "cronjobs", "cj":
		// CronJob -> Jobs
		a.navigateTo("jobs", selectedNs, selectedName)

	case "nodes", "no":
		// Node -> Pods on that node (all namespaces)
		a.navigateTo("pods", "", selectedName)

	case "namespaces", "ns":
		// Namespace -> Switch to that namespace and show pods
		a.navigateTo("pods", selectedName, "")

	default:
		// Default: show describe
		a.showDescribe()
		return
	}
}

// goBack returns to previous view (k9s Esc key behavior)
func (a *App) goBack() {
	a.navMx.Lock()
	if len(a.navigationStack) == 0 {
		a.navMx.Unlock()
		return
	}

	// Pop from stack (thread-safe)
	prev := a.navigationStack[len(a.navigationStack)-1]
	a.navigationStack = a.navigationStack[:len(a.navigationStack)-1]
	a.navMx.Unlock()

	// Use navigateTo for deadlock-safe state transition
	a.navigateTo(prev.resource, prev.namespace, prev.filter)
}

// pageUp scrolls up by half page
func (a *App) pageUp() {
	row, col := a.table.GetSelection()
	newRow := row - 10
	if newRow < 1 {
		newRow = 1
	}
	a.table.Select(newRow, col)
}

// pageDown scrolls down by half page
func (a *App) pageDown() {
	row, col := a.table.GetSelection()
	maxRow := a.table.GetRowCount() - 1
	newRow := row + 10
	if newRow > maxRow {
		newRow = maxRow
	}
	a.table.Select(newRow, col)
}

// setResource changes the current resource type (thread-safe, deadlock-safe)
func (a *App) setResource(resource string) {
	// Read current namespace to preserve it
	a.mx.RLock()
	ns := a.currentNamespace
	a.mx.RUnlock()

	// Use navigateTo for consistent state transition (clears filter)
	a.navigateTo(resource, ns, "")
}

// cycleNamespace cycles through namespaces (thread-safe, deadlock-safe)
func (a *App) cycleNamespace() {
	// Read all needed state under one lock
	a.mx.RLock()
	if len(a.namespaces) == 0 {
		a.mx.RUnlock()
		return
	}

	current := 0
	for i, n := range a.namespaces {
		if n == a.currentNamespace {
			current = i
			break
		}
	}

	next := (current + 1) % len(a.namespaces)
	nextNs := a.namespaces[next]
	resource := a.currentResource
	a.mx.RUnlock()

	// Track recent namespace usage
	if nextNs != "" {
		a.mx.Lock()
		a.addRecentNamespace(nextNs)
		a.mx.Unlock()
	}

	// Clear filter when switching namespace to avoid stale highlighting
	a.navigateTo(resource, nextNs, "")
}

// handleCommand processes command input
func (a *App) handleCommand(cmd string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return
	}

	// Handle namespace filtering (deadlock-safe)
	if strings.HasPrefix(cmd, "ns ") || strings.HasPrefix(cmd, "namespace ") {
		parts := strings.Fields(cmd)
		if len(parts) >= 2 {
			newNs := parts[1]
			if newNs == "all" || newNs == "*" {
				newNs = ""
			}
			// Track recent namespace usage
			if newNs != "" {
				a.mx.Lock()
				a.addRecentNamespace(newNs)
				a.mx.Unlock()
			}
			// Read current state
			a.mx.RLock()
			resource := a.currentResource
			a.mx.RUnlock()
			// Clear filter when switching namespace to avoid stale highlighting
			a.navigateTo(resource, newNs, "")
		}
		return
	}

	// Parse command with -n/--namespace flag (kubectl style: pods -n kube-system)
	parts := strings.Fields(cmd)
	resourceCmd := ""
	namespace := ""
	hasNamespaceFlag := false

	for i := 0; i < len(parts); i++ {
		part := parts[i]
		if part == "-n" || part == "--namespace" {
			hasNamespaceFlag = true
			if i+1 < len(parts) {
				namespace = parts[i+1]
				i++ // skip namespace value
			}
		} else if part == "-A" || part == "--all-namespaces" {
			hasNamespaceFlag = true
			namespace = "" // all namespaces
		} else if resourceCmd == "" {
			resourceCmd = part
		}
	}

	// Determine target namespace
	targetNs := ""
	if hasNamespaceFlag {
		if namespace == "all" || namespace == "*" || strings.Contains(cmd, "-A") {
			targetNs = ""
		} else {
			targetNs = namespace
		}
	} else {
		// Preserve current namespace if no flag specified
		a.mx.RLock()
		targetNs = a.currentNamespace
		a.mx.RUnlock()
	}

	// Use the resource command if found
	if resourceCmd != "" {
		// Use the commands list to handle all resource types dynamically
		for _, c := range commands {
			if resourceCmd == c.name || resourceCmd == c.alias {
				if c.category == "resource" {
					// Use navigateTo with target namespace (deadlock-safe)
					a.navigateTo(c.name, targetNs, "")
					return
				}
			}
		}
	}

	// Fallback: Use the commands list for simple commands without flags
	for _, c := range commands {
		if cmd == c.name || cmd == c.alias {
			if c.category == "resource" {
				// Use navigateTo with target namespace (deadlock-safe)
				a.navigateTo(c.name, targetNs, "")
				return
			}
		}
	}

	// Handle actions
	switch cmd {
	case "health", "status":
		a.showHealth()
	case "context", "ctx":
		a.showContextSwitcher()
	case "help", "?":
		a.showHelp()
	case "q", "quit", "exit":
		a.Stop()
	}
}

// statusColor returns color based on status
func (a *App) statusColor(status string) tcell.Color {
	switch status {
	case "Running", "Ready", "Active", "Succeeded", "Normal", "Completed":
		return tcell.ColorGreen
	case "Pending", "ContainerCreating", "Warning", "Updating":
		return tcell.ColorYellow
	case "Failed", "Error", "CrashLoopBackOff", "NotReady", "ImagePullBackOff", "ErrImagePull":
		return tcell.ColorRed
	default:
		return tcell.ColorWhite
	}
}

// Helper functions

func centered(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 0, true).
			AddItem(nil, 0, 1, false), width, 0, true).
		AddItem(nil, 0, 1, false)
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// navigateTo changes resource, namespace, and filter atomically (deadlock-safe)
// This is a centralized helper to avoid scattered Lock patterns
func (a *App) navigateTo(resource, namespace, filter string) {
	a.mx.Lock()
	a.currentResource = resource
	a.currentNamespace = namespace
	a.filterText = filter
	// Reset sort when changing resources
	a.sortColumn = -1
	a.sortAscending = true
	a.mx.Unlock()

	// Always run UI updates in goroutine after releasing lock
	go func() {
		a.updateHeader()
		a.updateStatusBar()
		a.refresh()
	}()
}

// sortByColumnName finds and sorts by column name (k9s Shift+N/A/S/R style)
func (a *App) sortByColumnName(colName string) {
	a.mx.RLock()
	headers := a.tableHeaders
	a.mx.RUnlock()

	// Find column index by name (case-insensitive)
	colUpper := strings.ToUpper(colName)
	for i, h := range headers {
		if strings.ToUpper(h) == colUpper {
			a.sortByColumn(i)
			return
		}
	}

	// Column not found, show message
	a.flashMsg(fmt.Sprintf("Column '%s' not found", colName), true)
}

// sortByColumn sorts the table by the specified column (k9s Shift+N/A/S/R style)
// columnIdx is 0-based, toggle direction if same column
func (a *App) sortByColumn(columnIdx int) {
	a.mx.Lock()
	if a.sortColumn == columnIdx {
		// Toggle direction
		a.sortAscending = !a.sortAscending
	} else {
		a.sortColumn = columnIdx
		a.sortAscending = true
	}
	sortAsc := a.sortAscending
	rows := a.tableRows
	headers := a.tableHeaders
	a.mx.Unlock()

	if len(rows) == 0 || columnIdx < 0 || columnIdx >= len(headers) {
		return
	}

	// Sort rows
	sortedRows := make([][]string, len(rows))
	copy(sortedRows, rows)

	// Determine sort type based on column header
	colName := ""
	if columnIdx < len(headers) {
		colName = strings.ToUpper(headers[columnIdx])
	}

	// Sort function
	sortFunc := func(i, j int) bool {
		valI := ""
		valJ := ""
		if columnIdx < len(sortedRows[i]) {
			valI = sortedRows[i][columnIdx]
		}
		if columnIdx < len(sortedRows[j]) {
			valJ = sortedRows[j][columnIdx]
		}

		var result bool

		// Numeric columns
		if colName == "RESTARTS" || colName == "COUNT" || colName == "DESIRED" ||
			colName == "CURRENT" || colName == "AVAILABLE" || colName == "ACTIVE" ||
			colName == "DATA" || colName == "SECRETS" {
			numI := parseNumber(valI)
			numJ := parseNumber(valJ)
			result = numI < numJ
		} else if colName == "READY" || colName == "COMPLETIONS" {
			// Ready format (e.g., "1/1")
			numI := parseReadyNum(valI)
			numJ := parseReadyNum(valJ)
			result = numI < numJ
		} else if colName == "AGE" || colName == "DURATION" || colName == "LAST SEEN" {
			// Age format (e.g., "5d", "3h", "10m")
			secI := parseAgeToSec(valI)
			secJ := parseAgeToSec(valJ)
			result = secI < secJ
		} else {
			// String comparison (case-insensitive)
			result = strings.ToLower(valI) < strings.ToLower(valJ)
		}

		if sortAsc {
			return result
		}
		return !result
	}

	// Sort
	for i := 0; i < len(sortedRows)-1; i++ {
		for j := i + 1; j < len(sortedRows); j++ {
			if !sortFunc(i, j) {
				sortedRows[i], sortedRows[j] = sortedRows[j], sortedRows[i]
			}
		}
	}

	// Update stored rows
	a.mx.Lock()
	a.tableRows = sortedRows
	a.mx.Unlock()

	// Show sort indicator in flash message
	direction := "↑"
	if !sortAsc {
		direction = "↓"
	}
	colHeader := headers[columnIdx]
	a.flashMsg(fmt.Sprintf("Sorted by %s %s", colHeader, direction), false)

	// Re-render table with filter applied
	a.mx.RLock()
	filter := a.filterText
	a.mx.RUnlock()
	a.applyFilterText(filter)
}

// parseNumber extracts a number from a string
func parseNumber(s string) int {
	s = strings.TrimSpace(s)
	num := 0
	fmt.Sscanf(s, "%d", &num)
	return num
}

// parseReadyNum extracts the first number from "X/Y" format
func parseReadyNum(s string) int {
	parts := strings.Split(s, "/")
	if len(parts) > 0 {
		return parseNumber(parts[0])
	}
	return 0
}

// parseAgeToSec converts age string to seconds for comparison
func parseAgeToSec(age string) int {
	age = strings.TrimSpace(age)
	if age == "" || age == "-" || age == "<unknown>" {
		return 0
	}

	total := 0
	// Handle compound ages like "2d3h" or "1h30m"
	for len(age) > 0 {
		var num int
		var unit byte
		n, _ := fmt.Sscanf(age, "%d%c", &num, &unit)
		if n < 2 {
			break
		}

		switch unit {
		case 's':
			total += num
		case 'm':
			total += num * 60
		case 'h':
			total += num * 3600
		case 'd':
			total += num * 86400
		}

		// Move past the parsed portion
		idx := strings.IndexAny(age, "smhd")
		if idx >= 0 && idx < len(age)-1 {
			age = age[idx+1:]
		} else {
			break
		}
	}
	return total
}
