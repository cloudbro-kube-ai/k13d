package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai"
	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

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
		a.closeModal("context-switcher")
		a.SetFocus(a.table)

		if selectedCtx == currentCtx {
			return
		}

		a.safeGo("switchContext", func() {
			a.flashMsg(fmt.Sprintf("Switching to context: %s...", selectedCtx), false)

			// Stop watcher before switching (it holds old cluster connection)
			a.stopWatch()

			err := a.k8s.SwitchContext(selectedCtx)
			if err != nil {
				a.flashMsg(fmt.Sprintf("Failed to switch context: %v", err), true)
				return
			}

			// Reset namespace to new context's default and clear cached namespace list
			newNs := a.k8s.GetCurrentNamespace()
			a.mx.Lock()
			a.currentNamespace = newNs
			a.namespaces = nil
			a.mx.Unlock()

			a.flashMsg(fmt.Sprintf("Switched to context: %s", selectedCtx), false)
			a.updateHeader()
			a.refresh()

			// Restart watcher for new cluster
			a.startWatch()
		})
	})

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			a.closeModal("context-switcher")
			a.SetFocus(a.table)
			return nil
		}
		return event
	})

	a.showModal("context-switcher", centered(list, 60, min(len(contexts)+4, 20)), true)
}

func (a *App) toggleAIPanel() {
	a.mx.Lock()
	a.showAIPanel = !a.showAIPanel
	show := a.showAIPanel
	if !show {
		if a.aiPanelRestoreWidth != 0 {
			a.aiPanelWidth = clampAIPanelWidth(a.aiPanelRestoreWidth)
		}
		a.aiPanelFullscreen = false
	}
	a.mx.Unlock()

	a.QueueUpdateDraw(func() {
		a.rebuildContentLayout(show)
	})

	if show {
		a.flashMsg("AI panel opened. Alt+F expands it to full size.", false)
		return
	}
	a.flashMsg("AI panel hidden. Press Ctrl+E to reopen.", false)
}

// useNamespace switches to the selected namespace (k9s u key)
func (a *App) useNamespace() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	if resource != "namespaces" && resource != "ns" {
		a.flashMsg("The 'u' key (use namespace) only works in namespaces view. Navigate to namespaces first using :namespaces", true)
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	nsName := a.getTableCellText(row, 0)
	if nsName == "" {
		return
	}

	a.flashMsg(fmt.Sprintf("Switched to namespace: %s", nsName), false)

	// navigateTo() handles stop-watch, state mutation, refresh, and start-watch safely
	a.navigateTo("pods", nsName, "")
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
			sb.WriteString(" [green]✓[white] Kubernetes: Connected\n")
			sb.WriteString(fmt.Sprintf("   Context: %s\n", ctxName))
			sb.WriteString(fmt.Sprintf("   Cluster: %s\n", cluster))
		}
	} else {
		sb.WriteString(" [red]✗[white] Kubernetes: Client not initialized\n")
	}

	sb.WriteString("\n")

	// AI status
	a.aiMx.RLock()
	aiClient := a.aiClient
	a.aiMx.RUnlock()
	if aiClient != nil && aiClient.IsReady() {
		sb.WriteString(fmt.Sprintf(" [green]✓[white] AI: Online (%s)\n", aiClient.GetModel()))
	} else {
		sb.WriteString(" [red]✗[white] AI: Offline\n")
		sb.WriteString("   Configure in ~/.config/k13d/config.yaml\n")
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
			a.closeModal("health")
			a.SetFocus(a.table)
			return nil
		}
		return event
	})

	a.showModal("health", centered(health, 60, 18), true)
}

// showAbout displays about modal with logo
func (a *App) showAbout() {
	about := AboutModal()
	a.showModal("about", centered(about, 60, 35), true)
	a.SetFocus(about)

	about.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc || event.Rune() == 'q' {
			a.closeModal("about")
			a.SetFocus(a.table)
			return nil
		}
		return event
	})
}

// showSortPicker displays a modal to choose sort column
func (a *App) showSortPicker() {
	a.mx.RLock()
	headers := a.tableHeaders
	sortCol := a.sortColumn
	sortAsc := a.sortAscending
	a.mx.RUnlock()

	if len(headers) == 0 {
		a.flashMsg("No columns available to sort", true)
		return
	}

	list := tview.NewList()
	list.SetBorder(true).SetTitle(" Sort By ")
	list.SetBackgroundColor(tcell.NewRGBColor(26, 27, 38))       // #1a1b26
	list.SetMainTextColor(tcell.NewRGBColor(192, 202, 245))      // #c0caf5
	list.SetSecondaryTextColor(tcell.NewRGBColor(169, 177, 214)) // #a9b1d6
	list.SetSelectedBackgroundColor(tcell.NewRGBColor(41, 46, 66))
	list.SetSelectedTextColor(tcell.NewRGBColor(122, 162, 247)) // #7aa2f7

	for i, h := range headers {
		label := h
		desc := ""
		if i == sortCol {
			dir := "▲ ascending"
			if !sortAsc {
				dir = "▼ descending"
			}
			label = fmt.Sprintf("%s  %s", h, dir)
			desc = "  (current — select again to toggle direction)"
		}
		list.AddItem(label, desc, 0, nil)
	}

	list.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		a.closeModal("sort-picker")
		a.SetFocus(a.table)
		a.sortByColumn(index)
	})

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			a.closeModal("sort-picker")
			a.SetFocus(a.table)
			return nil
		}
		return event
	})

	height := len(headers)*2 + 4
	if height > 20 {
		height = 20
	}
	a.showModal("sort-picker", centered(list, 55, height), true)
	a.SetFocus(list)
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

func (a *App) defaultTableCellColor(row, col int) tcell.Color {
	if a.isStatusColumn(col) {
		return a.statusColor(a.getTableCellText(row, col))
	}
	return tcell.ColorWhite
}

func (a *App) isStatusColumn(col int) bool {
	a.mx.RLock()
	defer a.mx.RUnlock()
	if col < 0 || col >= len(a.tableHeaders) {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(a.tableHeaders[col]), "STATUS")
}

func (a *App) rowMatchesAttachedAIContext(row int) bool {
	attached := a.getAttachedAIContext()
	if attached.IsZero() {
		return false
	}
	candidate := a.aiSelectionCandidateForRow(row)
	if candidate.IsZero() {
		return false
	}
	return attached.Matches(candidate)
}

func (a *App) aiSelectionCandidateForRow(row int) aiAttachedSelection {
	if row <= 0 || a.table == nil {
		return aiAttachedSelection{}
	}

	a.mx.RLock()
	resource := a.currentResource
	ns := a.currentNamespace
	headers := append([]string(nil), a.tableHeaders...)
	a.mx.RUnlock()

	nameIdx := nameColumnIndex(resource)
	name := strings.TrimSpace(a.getTableCellText(row, nameIdx))
	if name == "" {
		return aiAttachedSelection{}
	}

	selectedNS := ns
	if nameIdx != 0 {
		if candidateNS := strings.TrimSpace(a.getTableCellText(row, 0)); candidateNS != "" {
			selectedNS = candidateNS
		}
	}

	var parts []string
	for i, header := range headers {
		value := strings.TrimSpace(a.getTableCellText(row, i))
		if value == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", header, value))
	}

	return aiAttachedSelection{
		Resource:  resource,
		Namespace: selectedNS,
		Name:      name,
		Summary:   strings.Join(parts, " | "),
	}
}

func (a *App) refreshTableDecorations() {
	if a.table == nil {
		return
	}
	for row := 1; row < a.table.GetRowCount(); row++ {
		a.updateRowSelection(row)
	}
}

// updateRowSelection updates visual styling for a row based on selection and AI context state.
func (a *App) updateRowSelection(row int) {
	a.mx.RLock()
	isSelected := a.selectedRows[row]
	a.mx.RUnlock()
	isAIContext := a.rowMatchesAttachedAIContext(row)

	colCount := a.table.GetColumnCount()
	for col := 0; col < colCount; col++ {
		cell := a.table.GetCell(row, col)
		if cell != nil {
			background := tcell.ColorDefault
			textColor := a.defaultTableCellColor(row, col)
			if isSelected {
				background = tcell.ColorDarkCyan
				textColor = tcell.ColorWhite
			} else if isAIContext {
				background = tcell.NewRGBColor(41, 46, 66)
				if textColor == tcell.ColorWhite {
					textColor = tcell.NewRGBColor(198, 208, 245)
				}
			}
			cell.SetBackgroundColor(background)
			cell.SetTextColor(textColor)
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

	a.refreshTableDecorations()
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

	a.safeGo("briefing-ai", func() { a.briefing.UpdateWithAI() })
}

// showModelSelector displays a modal for switching AI model profiles
func (a *App) showModelSelector() {
	if err := a.reloadConfigFromDisk(); err != nil {
		a.flashMsg(fmt.Sprintf("Failed to reload config: %v", err), true)
		return
	}

	if a.config == nil || len(a.config.Models) == 0 {
		a.flashMsg("No AI model profiles configured. Add model definitions to your config.yaml file under the 'models' section.", true)
		return
	}

	list := tview.NewList().
		ShowSecondaryText(true).
		SetHighlightFullLine(true)
	list.SetBorder(true).SetTitle(" Select AI Model (Enter to switch, Esc to cancel) ")

	for _, m := range a.config.Models {
		prefix := "  "
		if m.Name == a.config.ActiveModel {
			prefix = "* "
		}
		mainText := prefix + m.Name
		secondText := fmt.Sprintf("  %s / %s", m.Provider, m.Model)
		if m.Description != "" {
			secondText += " - " + m.Description
		}
		list.AddItem(mainText, secondText, 0, nil)
	}

	list.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		if index < len(a.config.Models) {
			name := a.config.Models[index].Name
			a.closeModal("model-selector")
			a.SetFocus(a.table)
			a.switchModel(name)
		}
	})

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			a.closeModal("model-selector")
			a.SetFocus(a.table)
			return nil
		}
		return event
	})

	height := len(a.config.Models)*2 + 4
	if height > 20 {
		height = 20
	}
	a.showModal("model-selector", centered(list, 65, height), true)
	a.SetFocus(list)
}

// switchModel switches to a named AI model profile
func (a *App) switchModel(name string) {
	if err := a.reloadConfigFromDisk(); err != nil {
		a.flashMsg(fmt.Sprintf("Failed to reload config: %v", err), true)
		return
	}

	if a.config == nil {
		a.flashMsg("Configuration not available. Cannot switch AI model without config.yaml.", true)
		return
	}

	if !a.config.SetActiveModel(name) {
		a.flashMsg(fmt.Sprintf("Model profile '%s' not found in config.yaml. Check available models using :model command.", name), true)
		return
	}

	// Save config
	if err := a.config.Save(); err != nil {
		a.flashMsg(fmt.Sprintf("Failed to save config: %v. Model switch may not persist.", err), true)
		return
	}

	// Reinitialize AI client with new model
	newClient, err := ai.NewClient(&a.config.LLM)
	if err != nil {
		a.flashMsg(fmt.Sprintf("Failed to initialize model '%s': %v. Check your API keys and model configuration.", name, err), true)
		return
	}
	a.aiMx.Lock()
	a.aiClient = newClient
	a.aiMx.Unlock()
	msg := fmt.Sprintf("Switched to model: %s (%s/%s)", name, a.config.LLM.Provider, a.config.LLM.Model)
	if a.config.LLM.Provider == "ollama" {
		msg += " | Ollama models must support tools/function calling."
	}
	a.flashMsg(msg, false)
	a.updateHeader()
	a.QueueUpdateDraw(func() {
		a.applyAIChrome()
	})
}

// showPlugins displays a modal listing all configured plugins
func (a *App) showPlugins() {
	var sb strings.Builder
	sb.WriteString("[cyan::b]Configured Plugins[white::-]\n\n")

	if a.plugins == nil || len(a.plugins.Plugins) == 0 {
		sb.WriteString("[gray]No plugins configured.\n\n")
		sb.WriteString("Add plugins in: ~/.config/k13d/plugins.yaml\n\n")
		sb.WriteString("Example:\n")
		sb.WriteString("[yellow]plugins:\n")
		sb.WriteString("  dive:\n")
		sb.WriteString("    shortCut: Ctrl-I\n")
		sb.WriteString("    description: Dive into container image\n")
		sb.WriteString("    scopes: [pods]\n")
		sb.WriteString("    command: dive\n")
		sb.WriteString("    args: [$IMAGE][white]\n")
	} else {
		sb.WriteString(fmt.Sprintf("  %-15s %-12s %-20s %s\n", "NAME", "SHORTCUT", "SCOPES", "DESCRIPTION"))
		sb.WriteString(fmt.Sprintf("  %-15s %-12s %-20s %s\n", "────", "────────", "──────", "───────────"))
		for name, plugin := range a.plugins.Plugins {
			scopes := strings.Join(plugin.Scopes, ", ")
			sb.WriteString(fmt.Sprintf("  %-15s %-12s %-20s %s\n", name, plugin.ShortCut, scopes, plugin.Description))
		}
		sb.WriteString(fmt.Sprintf("\n[gray]Total: %d plugins loaded[white]\n", len(a.plugins.Plugins)))
	}

	sb.WriteString("\n[gray]Config: ~/.config/k13d/plugins.yaml[white]")
	sb.WriteString("\n[gray]Variables: $NAMESPACE, $NAME, $CONTEXT, $IMAGE, $LABELS.key, $ANNOTATIONS.key[white]")
	sb.WriteString("\n\n[gray]Press Esc to close[white]")

	pluginView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetText(sb.String())
	pluginView.SetBorder(true).SetTitle(" Plugins (Esc to close) ")

	a.showModal("plugins", centered(pluginView, 80, 30), true)
	a.SetFocus(pluginView)

	pluginView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc || event.Rune() == 'q' {
			a.closeModal("plugins")
			a.SetFocus(a.table)
			return nil
		}
		return event
	})
}

// executePlugin runs a plugin command with the current resource context
func (a *App) executePlugin(name string, plugin config.PluginConfig) {
	row, _ := a.table.GetSelection()
	if row <= 0 {
		a.flashMsg("No resource selected. Please select a resource from the list before running a plugin.", true)
		return
	}

	// Build plugin context from selected resource
	a.mx.RLock()
	ns := a.currentNamespace
	a.mx.RUnlock()

	// Get resource name from table
	resourceName := ""
	resourceNs := ns
	if a.table.GetColumnCount() > 1 {
		cell0 := a.table.GetCell(row, 0)
		cell1 := a.table.GetCell(row, 1)
		if cell0 != nil && cell1 != nil {
			resourceNs = cell0.Text
			resourceName = cell1.Text
		}
	}

	ctx := &config.PluginContext{
		Namespace: resourceNs,
		Name:      resourceName,
		Context:   a.getCurrentContext(),
	}

	if plugin.Confirm {
		expandedArgs := plugin.ExpandArgs(ctx)
		cmdStr := plugin.Command + " " + strings.Join(expandedArgs, " ")
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Run plugin '%s'?\n\n%s", name, cmdStr)).
			AddButtons([]string{"Cancel", "Execute"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				a.closeModal("plugin-confirm")
				a.SetFocus(a.table)
				if buttonLabel == "Execute" {
					a.safeGo("runPlugin-"+name, func() { a.runPlugin(name, plugin, ctx) })
				}
			})
		a.showModal("plugin-confirm", modal, true)
		return
	}

	a.safeGo("runPlugin-"+name, func() { a.runPlugin(name, plugin, ctx) })
}

// runPlugin executes a plugin command
func (a *App) runPlugin(name string, plugin config.PluginConfig, ctx *config.PluginContext) {
	if plugin.Background {
		a.flashMsg(fmt.Sprintf("Running plugin '%s' in background...", name), false)
		if err := plugin.Execute(a.getAppContext(), ctx); err != nil {
			a.flashMsg(fmt.Sprintf("Plugin '%s' error: %v", name, err), true)
		}
		return
	}

	// Foreground execution - suspend TUI
	a.flashMsg(fmt.Sprintf("Running plugin '%s'...", name), false)
	a.safeSuspend(func() {
		if err := plugin.Execute(a.getAppContext(), ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Plugin '%s' error: %v\n", name, err)
		}
	})
	a.requestSync()
	a.refresh()
}

// showBenchmark runs benchmark on service (k9s b key) - placeholder
func (a *App) showBenchmark() {
	a.flashMsg("Benchmark feature is not yet implemented. This feature will be available in a future release.", true)
}
