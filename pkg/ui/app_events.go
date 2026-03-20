package ui

import (
	"strings"
	"sync/atomic"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/gdamore/tcell/v2"
)

// setupKeybindings configures keyboard shortcuts (k9s compatible)
func (a *App) setupKeybindings() {
	isShiftTab := func(event *tcell.EventKey) bool {
		if event == nil {
			return false
		}
		return event.Key() == tcell.KeyBacktab || (event.Key() == tcell.KeyTab && event.Modifiers()&tcell.ModShift != 0)
	}

	a.table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if a.handleAIPanelResizeKey(event) {
			return nil
		}
		if isShiftTab(event) {
			if a.showAIPanel {
				a.focusAITranscript()
				return nil
			}
			return event
		}

		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				a.Stop()
				return nil
			case ':':
				a.SetFocus(a.cmdInput)
				return nil
			case '/':
				a.startFilter()
				return nil
			case '?':
				a.showHelp()
				return nil
			case 'r':
				a.stopWatch()
				a.safeGo("refresh-and-watch", func() {
					a.refresh()
					a.startWatch()
				})
				return nil
			case 'n':
				a.cycleNamespace()
				return nil
			// k9s style: 0 = all namespaces, 1-9 = select namespace by number
			case '0':
				a.safeGo("switchToAllNamespaces", a.switchToAllNamespaces)
				return nil
			case '1', '2', '3', '4', '5', '6', '7', '8', '9':
				num := int(event.Rune() - '0')
				a.safeGo("selectNamespaceByNumber", func() {
					a.selectNamespaceByNumber(num)
				})
				return nil
			case 'l':
				a.showLogs()
				return nil
			case 'p':
				a.showLogsPrevious()
				return nil
			case 'd':
				a.showDescribe() // k9s: d = describe
				return nil
			case 'y':
				a.showYAML() // k9s: y = yaml
				return nil
			case 'e':
				a.editResource() // k9s: e = edit
				return nil
			case 's':
				a.execShell() // k9s: s = shell
				return nil
			case 'a':
				a.attachContainer() // k9s: a = attach
				return nil
			case 'c':
				a.showContextSwitcher() // context switcher
				return nil
			case 'g':
				a.table.Select(1, 0) // go to top
				return nil
			case 'G':
				a.table.Select(a.table.GetRowCount()-1, 0) // go to bottom
				return nil
			case 'u':
				a.useNamespace() // k9s: u = use namespace
				return nil
			case 'o':
				a.showNode() // k9s: o = show node (for pods)
				return nil
			case 'O':
				a.showSettings() // Shift+O = settings/options
				return nil
			case 'k':
				a.killPod() // k9s: k or Ctrl+K = kill pod
				return nil
			case 'b':
				a.showBenchmark() // k9s: b = benchmark (services)
				return nil
			case 'f':
				a.showPortForwards() // f = show active port-forwards
				return nil
			case 't':
				a.triggerCronJob() // k9s: t = trigger (cronjobs)
				return nil
			case 'z':
				a.showRelatedResource() // k9s: z = zoom (show related)
				return nil
			case 'F':
				a.portForward() // k9s: Shift+F = port-forward
				return nil
			case 'S':
				a.scaleResource() // k9s: Shift+S = scale
				return nil
			case 'R':
				a.restartResource() // k9s: Shift+R = restart
				return nil
			case 'B':
				a.toggleBriefing() // Shift+B = toggle briefing panel
				return nil
			case 'I':
				a.showAbout() // Shift+I = about/info
			// k9s-style column sorting (Shift + column key)
			case 'N':
				a.sortByColumnName("NAME") // Shift+N = sort by Name
				return nil
			case 'A':
				a.sortByColumnName("AGE") // Shift+A = sort by Age
				return nil
			case 'T':
				a.sortByColumnName("STATUS") // Shift+T = sort by sTatus
				return nil
			case 'P':
				a.sortByColumnName("NAMESPACE") // Shift+P = sort by namespace (ns)
				return nil
			case 'C':
				a.sortByColumnName("RESTARTS") // Shift+C = sort by restart Count
				return nil
			case 'D':
				a.sortByColumnName("READY") // Shift+D = sort by reaDy
				return nil
			case '!':
				a.sortByColumn(0) // Shift+1 = sort by column 1
				return nil
			case '@':
				a.sortByColumn(1) // Shift+2 = sort by column 2
				return nil
			case '#':
				a.sortByColumn(2) // Shift+3 = sort by column 3
				return nil
			case '$':
				a.sortByColumn(3) // Shift+4 = sort by column 4
				return nil
			case '%':
				a.sortByColumn(4) // Shift+5 = sort by column 5
				return nil
			case '^':
				a.sortByColumn(5) // Shift+6 = sort by column 6
				return nil
			case ' ':
				a.toggleSelection() // k9s: Space = toggle selection (multi-select)
				return nil
			}
		case tcell.KeyTab:
			if a.showAIPanel {
				a.focusAIInput()
			}
			return nil
		case tcell.KeyEnter:
			a.drillDown() // k9s: Enter = drill down to related resource
			return nil
		case tcell.KeyEsc:
			a.goBack() // k9s: Esc = go back
			return nil
		case tcell.KeyCtrlD:
			a.confirmDelete() // k9s: Ctrl+D = delete
			return nil
		case tcell.KeyCtrlK:
			a.killPod() // k9s: Ctrl+K = kill pod
			return nil
		case tcell.KeyCtrlU:
			a.pageUp() // k9s: Ctrl+U = page up
			return nil
		case tcell.KeyCtrlF:
			a.pageDown() // k9s: Ctrl+F = page down (vim style)
			return nil
		case tcell.KeyCtrlB:
			a.pageUp() // k9s: Ctrl+B = page up (vim style)
			return nil
		case tcell.KeyCtrlC:
			a.Stop()
			return nil
		case tcell.KeyCtrlE:
			a.toggleAIPanel()
			return nil
		case tcell.KeyCtrlI:
			a.aiBriefing() // Ctrl+I = AI-generated briefing
			return nil
		}

		// Check plugin shortcuts for current resource (k9s plugin pattern)
		if a.plugins != nil {
			a.mx.RLock()
			resource := a.currentResource
			a.mx.RUnlock()

			for name, plugin := range a.plugins.GetPluginsForScope(resource) {
				if matchPluginShortcut(event, plugin.ShortCut) {
					a.safeGo("executePlugin-"+name, func() { a.executePlugin(name, plugin) })
					return nil
				}
			}
		}

		return event
	})

	a.aiPanel.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if a.handleAIPanelResizeKey(event) {
			return nil
		}
		if isShiftTab(event) {
			a.SetFocus(a.table)
			return nil
		}

		switch event.Key() {
		case tcell.KeyTab:
			a.focusAIInput()
			return nil
		case tcell.KeyUp:
			row, col := a.aiPanel.GetScrollOffset()
			if row > 0 {
				a.aiPanel.ScrollTo(row-1, col)
			}
			return nil
		case tcell.KeyDown:
			row, col := a.aiPanel.GetScrollOffset()
			a.aiPanel.ScrollTo(row+1, col)
			return nil
		case tcell.KeyPgUp:
			row, col := a.aiPanel.GetScrollOffset()
			_, _, _, height := a.aiPanel.GetInnerRect()
			newRow := row - height
			if newRow < 0 {
				newRow = 0
			}
			a.aiPanel.ScrollTo(newRow, col)
			return nil
		case tcell.KeyPgDn:
			row, col := a.aiPanel.GetScrollOffset()
			_, _, _, height := a.aiPanel.GetInnerRect()
			a.aiPanel.ScrollTo(row+height, col)
			return nil
		case tcell.KeyEnter:
			// Approve pending MCP tool call (use atomic for lock-free check)
			if atomic.LoadInt32(&a.hasToolCall) == 1 {
				a.approveToolCall(true)
				return nil
			}
		case tcell.KeyEsc:
			// Cancel pending MCP tool call (use atomic for lock-free check)
			if atomic.LoadInt32(&a.hasToolCall) == 1 {
				a.approveToolCall(false)
				a.SetFocus(a.table)
				return nil
			}
			// Clear pending decisions when escaping
			a.clearPendingDecisions()
			a.SetFocus(a.table)
			return nil
		case tcell.KeyCtrlE:
			a.toggleAIPanel()
			return nil
		case tcell.KeyRune:
			// Handle Y/N for MCP tool approval (kubectl-ai style)
			if atomic.LoadInt32(&a.hasToolCall) == 1 {
				switch event.Rune() {
				case 'y', 'Y':
					a.approveToolCall(true)
					return nil
				case 'n', 'N':
					a.approveToolCall(false)
					a.SetFocus(a.table)
					return nil
				}
			}
			// Handle decision input (1-9 to execute command, A to execute all)
			a.aiMx.RLock()
			numDecisions := len(a.pendingDecisions)
			a.aiMx.RUnlock()
			if numDecisions > 0 {
				switch event.Rune() {
				case '1', '2', '3', '4', '5', '6', '7', '8', '9':
					idx := int(event.Rune() - '1')
					if idx < numDecisions {
						a.safeGo("executeDecision", func() { a.executeDecision(idx) })
					}
					return nil
				case 'a', 'A':
					a.safeGo("executeAllDecisions", func() { a.executeAllDecisions() })
					return nil
				}
			}
			// Vim-style scroll keys (only when no pending approval/decisions)
			switch event.Rune() {
			case 'j':
				row, col := a.aiPanel.GetScrollOffset()
				a.aiPanel.ScrollTo(row+1, col)
				return nil
			case 'k':
				row, col := a.aiPanel.GetScrollOffset()
				if row > 0 {
					a.aiPanel.ScrollTo(row-1, col)
				}
				return nil
			case 'G':
				a.aiPanel.ScrollToEnd()
				return nil
			case 'g':
				a.aiPanel.ScrollTo(0, 0)
				return nil
			}
		}
		return event
	})
}

// setupAutocomplete configures the command input with autocomplete
func (a *App) setupAutocomplete() {
	// Track current suggestions
	var suggestions []string
	var selectedIdx int

	// Update hint and dropdown as user types
	a.cmdInput.SetChangedFunc(func(text string) {
		suggestions = a.getCompletions(text)
		selectedIdx = 0

		if len(suggestions) > 0 && text != "" {
			// Show dimmed hint for first suggestion
			hint := suggestions[0]
			if strings.HasPrefix(hint, text) {
				remaining := hint[len(text):]
				a.cmdHint.SetText("[gray]" + remaining)
			} else {
				a.cmdHint.SetText("[gray] → " + hint)
			}
			// Show dropdown when 2+ matches for k9s-style autocomplete popup
			if len(suggestions) >= 2 {
				a.showAutocompleteDropdown(suggestions, selectedIdx)
			} else {
				a.hideAutocompleteDropdown()
			}
		} else {
			a.cmdHint.SetText("")
			a.hideAutocompleteDropdown()
		}
	})

	// Handle special keys
	a.cmdInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if a.handleAIPanelResizeKey(event) {
			return nil
		}

		text := a.cmdInput.GetText()

		switch event.Key() {
		case tcell.KeyTab:
			// Accept current suggestion
			if len(suggestions) > 0 {
				selected := suggestions[selectedIdx]
				// If it's a namespace command, add space
				if selected == "ns" || strings.HasPrefix(selected, "ns ") {
					a.cmdInput.SetText(selected + " ")
				} else {
					a.cmdInput.SetText(selected)
				}
				a.cmdHint.SetText("")
				a.hideAutocompleteDropdown()
			}
			return nil

		case tcell.KeyDown:
			// Cycle through suggestions
			if len(suggestions) > 1 {
				selectedIdx = (selectedIdx + 1) % len(suggestions)
				hint := suggestions[selectedIdx]
				if strings.HasPrefix(hint, text) {
					remaining := hint[len(text):]
					a.cmdHint.SetText("[gray]" + remaining)
				} else {
					a.cmdHint.SetText("[gray] → " + hint)
				}
				// Update dropdown selection
				if len(suggestions) >= 2 {
					a.showAutocompleteDropdown(suggestions, selectedIdx)
				}
			}
			return nil

		case tcell.KeyUp:
			// If suggestions available, cycle backwards; otherwise browse history
			if len(suggestions) > 1 {
				selectedIdx--
				if selectedIdx < 0 {
					selectedIdx = len(suggestions) - 1
				}
				hint := suggestions[selectedIdx]
				if strings.HasPrefix(hint, text) {
					remaining := hint[len(text):]
					a.cmdHint.SetText("[gray]" + remaining)
				} else {
					a.cmdHint.SetText("[gray] → " + hint)
				}
				// Update dropdown selection
				if len(suggestions) >= 2 {
					a.showAutocompleteDropdown(suggestions, selectedIdx)
				}
			} else if len(a.cmdHistory) > 0 {
				// Browse command history
				if a.cmdHistoryIdx < 0 {
					a.cmdHistoryIdx = len(a.cmdHistory) - 1
				} else if a.cmdHistoryIdx > 0 {
					a.cmdHistoryIdx--
				}
				a.cmdInput.SetText(a.cmdHistory[a.cmdHistoryIdx])
				a.cmdHint.SetText("")
			}
			return nil

		case tcell.KeyEnter:
			cmd := text
			// If hint is showing and user didn't type full command, use suggestion
			if len(suggestions) > 0 && cmd != suggestions[selectedIdx] {
				// Check if input matches number for namespace selection
				if num, ok := a.parseNamespaceNumber(cmd); ok {
					a.selectNamespaceByNumber(num)
					a.cmdInput.SetText("")
					a.cmdHint.SetText("")
					a.hideAutocompleteDropdown()
					a.cmdInput.SetLabel(" : ")
					a.SetFocus(a.table)
					return nil
				}
			}
			// Record command in history
			if cmd != "" {
				a.addCmdHistory(cmd)
			}
			a.cmdHistoryIdx = -1
			a.cmdInput.SetText("")
			a.cmdHint.SetText("")
			a.hideAutocompleteDropdown()
			a.cmdInput.SetLabel(" : ")
			a.handleCommand(cmd)
			a.SetFocus(a.table)
			return nil

		case tcell.KeyEsc:
			a.cmdInput.SetText("")
			a.cmdHint.SetText("")
			a.hideAutocompleteDropdown()
			a.cmdInput.SetLabel(" : ")
			a.SetFocus(a.table)
			return nil

		case tcell.KeyRune:
			// Check for number input (1-9) to select namespace
			if event.Rune() >= '0' && event.Rune() <= '9' && text == "" {
				// Show namespace hint
				a.showNamespaceHint()
			}
		}

		return event
	})

	a.cmdInput.SetDoneFunc(func(key tcell.Key) {
		// Already handled in InputCapture
	})
}

// getCompletions returns matching commands for the input
func (a *App) getCompletions(input string) []string {
	if input == "" {
		return nil
	}

	inputLower := strings.ToLower(input)
	var matches []string

	// Check for namespace command (ns <namespace>)
	if strings.HasPrefix(inputLower, "ns ") || strings.HasPrefix(inputLower, "namespace ") {
		prefix := strings.TrimPrefix(inputLower, "ns ")
		prefix = strings.TrimPrefix(prefix, "namespace ")

		a.mx.RLock()
		namespaces := a.namespaces
		a.mx.RUnlock()

		for _, ns := range namespaces {
			if ns == "" {
				continue
			}
			if strings.HasPrefix(ns, prefix) {
				matches = append(matches, "ns "+ns)
			}
		}
		return matches
	}

	// Check for resource command with -n flag (e.g., "pods -n kube")
	if strings.Contains(inputLower, " -n ") {
		parts := strings.Split(input, " -n ")
		if len(parts) == 2 {
			resourcePart := strings.TrimSpace(parts[0])
			nsPrefix := strings.TrimSpace(parts[1])

			a.mx.RLock()
			namespaces := a.namespaces
			a.mx.RUnlock()

			for _, ns := range namespaces {
				if ns == "" {
					continue
				}
				if strings.HasPrefix(ns, nsPrefix) {
					matches = append(matches, resourcePart+" -n "+ns)
				}
			}
			return matches
		}
	}

	// Check if input ends with "-n " - suggest namespaces
	if strings.HasSuffix(inputLower, "-n ") || strings.HasSuffix(inputLower, "-n") {
		basePart := strings.TrimSuffix(input, " ")
		if !strings.HasSuffix(basePart, " ") {
			basePart = strings.TrimSuffix(basePart, "-n") + "-n "
		}

		a.mx.RLock()
		namespaces := a.namespaces
		a.mx.RUnlock()

		for _, ns := range namespaces {
			if ns == "" {
				matches = append(matches, basePart+"all")
			} else {
				matches = append(matches, basePart+ns)
			}
		}
		// Limit suggestions
		if len(matches) > 10 {
			matches = matches[:10]
		}
		return matches
	}

	// Match built-in commands first
	for _, cmd := range commands {
		if strings.HasPrefix(cmd.name, inputLower) || strings.HasPrefix(cmd.alias, inputLower) {
			matches = append(matches, cmd.name)
		}
	}

	// Match custom aliases (k9s aliases.yaml pattern)
	if a.customAliases != nil {
		for alias, target := range a.customAliases.GetAll() {
			if strings.HasPrefix(alias, inputLower) || strings.HasPrefix(target, inputLower) {
				matches = append(matches, alias)
			}
		}
	}

	// Also match API resources from cluster (including CRDs)
	a.mx.RLock()
	apiResources := a.apiResources
	a.mx.RUnlock()

	seen := make(map[string]bool)
	for _, m := range matches {
		seen[m] = true
	}

	for _, res := range apiResources {
		if seen[res.Name] {
			continue
		}
		if strings.HasPrefix(res.Name, inputLower) {
			matches = append(matches, res.Name)
			seen[res.Name] = true
		}
		// Check short names
		for _, short := range res.ShortNames {
			if strings.HasPrefix(short, inputLower) && !seen[res.Name] {
				matches = append(matches, res.Name)
				seen[res.Name] = true
				break
			}
		}
	}

	return matches
}

func (a *App) handleAIPanelResizeKey(event *tcell.EventKey) bool {
	if event == nil || event.Modifiers()&tcell.ModAlt == 0 {
		return false
	}

	a.mx.RLock()
	showAI := a.showAIPanel
	a.mx.RUnlock()
	if !showAI {
		return false
	}

	switch event.Key() {
	case tcell.KeyLeft:
		a.adjustAIPanelWidth(-aiPanelResizeStep)
		return true
	case tcell.KeyRight:
		a.adjustAIPanelWidth(aiPanelResizeStep)
		return true
	case tcell.KeyRune:
		switch event.Rune() {
		case 'h', 'H':
			a.adjustAIPanelWidth(-aiPanelResizeStep)
			return true
		case 'l', 'L':
			a.adjustAIPanelWidth(aiPanelResizeStep)
			return true
		case '0':
			a.resetAIPanelWidth()
			return true
		}
	}

	return false
}

// setupAIInput configures the AI input field
func (a *App) setupAIInput() {
	a.aiInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			question := a.aiInput.GetText()
			if question != "" {
				a.aiInput.SetText("")
				if a.handleAICommand(question) {
					return
				}
				a.addAIInputHistory(question)
				a.safeGo("askAI", func() {
					a.askAI(question)
				})
			}
		}
	})

	a.aiInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if a.handleAIPanelResizeKey(event) {
			return nil
		}
		if event.Key() == tcell.KeyBacktab || (event.Key() == tcell.KeyTab && event.Modifiers()&tcell.ModShift != 0) {
			a.focusAITranscript()
			return nil
		}

		switch event.Key() {
		case tcell.KeyUp:
			a.aiInput.SetText(a.recallAIInputHistory(-1))
			return nil
		case tcell.KeyDown:
			a.aiInput.SetText(a.recallAIInputHistory(1))
			return nil
		case tcell.KeyCtrlE:
			a.toggleAIPanel()
			return nil
		case tcell.KeyEsc, tcell.KeyTab:
			a.SetFocus(a.table)
			return nil
		}
		return event
	})
}

const maxCmdHistory = 50

// addCmdHistory adds a command to the history, deduplicating consecutive repeats.
func (a *App) addCmdHistory(cmd string) {
	if len(a.cmdHistory) > 0 && a.cmdHistory[len(a.cmdHistory)-1] == cmd {
		return // Don't add consecutive duplicates
	}
	a.cmdHistory = append(a.cmdHistory, cmd)
	if len(a.cmdHistory) > maxCmdHistory {
		a.cmdHistory = a.cmdHistory[1:]
	}
}

// parseNamespaceNumber parses input as namespace number
func (a *App) parseNamespaceNumber(input string) (int, bool) {
	if len(input) != 1 {
		return 0, false
	}
	if input[0] >= '0' && input[0] <= '9' {
		return int(input[0] - '0'), true
	}
	return 0, false
}

// restoreAutocompleteHandler restores the default autocomplete behavior
func (a *App) restoreAutocompleteHandler() {
	a.setupAutocomplete()
}

// matchPluginShortcut checks if a key event matches a plugin shortcut string
func matchPluginShortcut(event *tcell.EventKey, shortcut string) bool {
	hasCtrl, hasShift, hasAlt, key := config.ParseShortcut(shortcut)

	// Check modifiers
	mod := event.Modifiers()
	if hasCtrl != (mod&tcell.ModCtrl != 0) {
		return false
	}
	if hasAlt != (mod&tcell.ModAlt != 0) {
		return false
	}

	// For shift, check if the rune is uppercase or if Shift modifier is set
	if hasShift {
		if mod&tcell.ModShift == 0 && !(event.Rune() >= 'A' && event.Rune() <= 'Z') {
			return false
		}
	}

	// Check key
	if len(key) == 1 {
		targetRune := rune(key[0])
		if hasShift {
			// Match uppercase version
			if event.Rune() != targetRune && event.Rune() != rune(strings.ToUpper(key)[0]) {
				return false
			}
		} else if hasCtrl {
			// Ctrl+key: tcell uses KeyCtrlA = 1, etc.
			if strings.ToLower(key) == strings.ToUpper(key) {
				// Non-letter, just match rune
				return event.Rune() == targetRune
			}
			lowerRune := rune(strings.ToLower(key)[0])
			ctrlKey := tcell.Key(lowerRune - 'a' + 1)
			return event.Key() == ctrlKey
		} else {
			return event.Rune() == targetRune
		}
	}

	return true
}
