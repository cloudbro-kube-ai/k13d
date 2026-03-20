package ui

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	defaultAIPanelWidth = 52
	minAIPanelWidth     = 36
	maxAIPanelWidth     = 90
	aiPanelResizeStep   = 4
)

// setupUI initializes all UI components
func (a *App) setupUI() {
	// Color scheme: use per-context skin if loaded, otherwise Tokyo Night defaults
	s := a.styles
	headerBg := s.K13s.Body.BgColor.ToTcellColor()
	tableBorder := s.K13s.Frame.FocusBorderColor.ToTcellColor()
	tableSelect := s.K13s.Views.Table.RowSelected.BgColor.ToTcellColor()
	aiBorder := tcell.NewRGBColor(187, 154, 247) // #bb9af7 (accent purple, no skin equivalent)
	statusBg := s.K13s.StatusBar.BgColor.ToTcellColor()

	// Header with gradient-like appearance
	a.header = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	a.header.SetBackgroundColor(headerBg)

	// Main table with enhanced styling
	a.table = tview.NewTable().
		SetSelectable(true, false).
		SetFixed(1, 0).
		SetSeparator('│')
	a.table.SetBorder(true).
		SetBorderColor(tableBorder).
		SetTitle(" Resources ").
		SetTitleColor(tableBorder)
	a.table.SetSelectedStyle(tcell.StyleDefault.
		Background(tableSelect).
		Foreground(tcell.ColorWhite).
		Bold(true))
	a.table.SetSelectionChangedFunc(func(row, column int) {
		a.applyAIChrome()
	})

	// AI Panel with enhanced styling
	a.aiPanel = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	a.aiPanel.SetBorder(false)

	a.aiMetaBar = tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true)
	a.aiMetaBar.SetBackgroundColor(tcell.ColorDefault)

	a.aiStatusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false)
	a.aiStatusBar.SetBackgroundColor(tcell.ColorDefault)
	a.aiStatusBar.SetTextAlign(tview.AlignCenter)

	// AI Input field with better styling
	a.aiInput = tview.NewInputField().
		SetLabel("[#bb9af7] ⟩ [-]").
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetPlaceholder("Ask AI... (/help for commands)")
	a.aiInput.SetPlaceholderStyle(tcell.StyleDefault.Foreground(tcell.ColorDarkGray))
	a.aiInput.SetLabelColor(aiBorder)
	a.setupAIInput()

	a.aiInputFrame = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.aiInput, 1, 0, true)
	a.aiInputFrame.SetBorder(true).
		SetTitle(" Prompt ").
		SetBorderColor(aiBorder).
		SetTitleColor(aiBorder)

	// Flash message area (k9s pattern)
	a.flash = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	// Briefing panel (natural language cluster summary)
	// Skip in test mode to prevent pulse animation blocking
	if !a.skipBriefing {
		a.briefing = NewBriefingPanel(a)
	}

	// Status bar with enhanced color
	a.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	a.statusBar.SetBackgroundColor(statusBg)
	a.statusBar.SetTextColor(tcell.ColorBlack)

	// Command input with enhanced styling
	a.cmdInput = tview.NewInputField().
		SetLabel("[#7aa2f7] :[white] ").
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorDefault)
	a.cmdInput.SetLabelColor(tableBorder)

	// Autocomplete hint (dimmed text showing suggestion)
	a.cmdHint = tview.NewTextView().
		SetDynamicColors(true)

	// Autocomplete dropdown with enhanced styling
	a.cmdDropdown = tview.NewList().
		ShowSecondaryText(true).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(tableSelect).
		SetSelectedTextColor(tcell.ColorWhite)
	a.cmdDropdown.SetBorder(true).
		SetTitle(" Commands ").
		SetBorderColor(tableBorder).
		SetTitleColor(tableBorder)

	// Setup autocomplete behavior
	a.setupAutocomplete()

	// AI Panel container with enhanced border
	a.aiContainer = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.aiMetaBar, 2, 0, false).
		AddItem(a.aiPanel, 0, 1, false).
		AddItem(a.aiStatusBar, 1, 0, false).
		AddItem(a.aiInputFrame, 3, 0, true)
	a.aiContainer.SetBorder(true).
		SetTitle(" 🤖 AI Assistant ").
		SetBorderColor(aiBorder).
		SetTitleColor(aiBorder)

	// Content area (table + AI panel)
	a.contentFlex = tview.NewFlex()
	a.contentFlex.AddItem(a.table, 0, 3, true)
	if a.showAIPanel {
		a.contentFlex.AddItem(a.aiContainer, a.currentAIPanelWidth(), 0, false)
	}

	// Command bar with hint overlay
	cmdFlex := tview.NewFlex().
		AddItem(a.cmdInput, 0, 1, true).
		AddItem(a.cmdHint, 0, 2, false)

	// Main layout with optional briefing panel
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.header, 4, 0, false). // 4 lines: title, context info, namespace preview
		AddItem(a.flash, 1, 0, false)

	// Add briefing panel only if it's enabled
	if a.briefing != nil {
		mainFlex.AddItem(a.briefing, 5, 0, false) // 3 lines content + 2 border
	}

	mainFlex.
		AddItem(a.contentFlex, 0, 1, true).
		AddItem(a.statusBar, 1, 0, false).
		AddItem(cmdFlex, 1, 0, false)

	// Pages
	a.pages = tview.NewPages().
		AddPage("main", mainFlex, true, true)

	a.SetRoot(a.pages, true)

	// Initial UI state
	a.updateHeader()
	a.resetAIConversation()
	a.applyAIChrome()
	a.updateStatusBar()
}

// flash displays a temporary message (k9s pattern)
func (a *App) flashMsg(msg string, isError bool) {
	color := "[green]"
	if isError {
		color = "[red]"
	}
	seq := atomic.AddInt64(&a.flashSeq, 1)
	a.QueueUpdateDraw(func() {
		// Clear before setting to prevent ghosting
		a.flash.Clear()
		a.flash.SetText(color + msg + "[white]")
	})

	// Clear after 3 seconds, but only if no newer flash message was shown
	a.safeGo("flashMsg-clear", func() {
		time.Sleep(3 * time.Second)
		if atomic.LoadInt64(&a.flashSeq) == seq {
			a.QueueUpdateDraw(func() {
				a.flash.Clear()
			})
		}
	})
}

// updateHeader updates the header text (thread-safe)
func (a *App) updateHeader() {
	ctxName := "N/A"
	cluster := "N/A"
	if a.k8s != nil {
		var err error
		ctxName, cluster, _, err = a.k8s.GetContextInfo()
		if err != nil {
			ctxName = "N/A"
			cluster = "N/A"
		}
	}

	// Read watch state FIRST (before mx) to prevent lock ordering deadlock
	// with startWatch() which acquires watchMu → mx
	watchStatus := ""
	a.watchMu.RLock()
	if a.watcher != nil {
		switch a.watcher.State() {
		case k8s.WatchStateActive:
			watchStatus = " [#9ece6a]◉ Live[-]"
		case k8s.WatchStateFallback:
			watchStatus = " [#e0af68]○ Poll[-]"
		}
	}
	a.watchMu.RUnlock()

	a.mx.RLock()
	ns := a.currentNamespace
	resource := a.currentResource
	namespaces := a.namespaces
	a.mx.RUnlock()

	currentNsDisplay := "[#9ece6a]all[-]"
	if ns != "" {
		currentNsDisplay = "[#9ece6a]" + ns + "[-]"
	}

	a.aiMx.RLock()
	headerAIClient := a.aiClient
	a.aiMx.RUnlock()
	aiStatus := "[#f7768e]● Offline[-]"
	if headerAIClient != nil && headerAIClient.IsReady() {
		aiStatus = "[#9ece6a]● Online[-]"
	}

	// Build namespace quick-select preview (show first 9 namespaces with numbers)
	nsPreview := ""
	if len(namespaces) > 1 {
		var nsParts []string
		maxShow := 9
		if len(namespaces) < maxShow {
			maxShow = len(namespaces)
		}
		for i := 0; i < maxShow; i++ {
			nsName := namespaces[i]
			if nsName == "" {
				nsName = "all"
			}
			// Highlight current namespace
			if (ns == "" && nsName == "all") || ns == nsName {
				nsParts = append(nsParts, fmt.Sprintf("[#e0af68]%d[-]:[#9ece6a::b]%s[-::-]", i, truncateNsName(nsName, 12)))
			} else {
				nsParts = append(nsParts, fmt.Sprintf("[#e0af68]%d[-]:[#565f89]%s[-]", i, truncateNsName(nsName, 12)))
			}
		}
		if len(namespaces) > maxShow {
			nsParts = append(nsParts, fmt.Sprintf("[#565f89]+%d more[-]", len(namespaces)-maxShow))
		}
		nsPreview = " " + strings.Join(nsParts, " ")
	}

	header := fmt.Sprintf(
		" %s [#565f89]%s %s[-]                                        [#bb9af7]AI[-] %s%s\n"+
			" [#565f89]⎈ Context:[-] [#7aa2f7]%s[-]  [#565f89]Cluster:[-] [#7aa2f7]%s[-]  [#565f89]NS:[-] %s  [#565f89]Resource:[-] [#7dcfff]%s[-]\n"+
			" [#565f89]Namespaces:[-]%s",
		HeaderLogo(), Tagline, Version, aiStatus, watchStatus, ctxName, cluster, currentNsDisplay, resource, nsPreview,
	)

	// Use QueueUpdateDraw only after Application.Run() has started (k9s pattern)
	if atomic.LoadInt32(&a.running) == 1 {
		a.QueueUpdateDraw(func() {
			// Clear before setting new text to prevent ghosting artifacts
			a.header.Clear()
			a.header.SetText(header)
		})
	} else {
		// Direct update during initialization (before Run())
		a.header.Clear()
		a.header.SetText(header)
	}
}

// truncateNsName truncates namespace name for display
func truncateNsName(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}
	return name[:maxLen-1] + "…"
}

// updateStatusBar updates the status bar (k9s style)
func (a *App) updateStatusBar() {
	a.mx.RLock()
	resource := a.currentResource
	sortCol := a.sortColumn
	sortAsc := a.sortAscending
	headers := a.tableHeaders
	filter := a.filterText
	showAI := a.showAIPanel
	aiWidth := clampAIPanelWidth(a.aiPanelWidth)
	a.mx.RUnlock()

	// Enhanced status bar with Tokyo Night colors (dark text on green background)
	shortcuts := "[black]n[-][#1a1b26]NS[-] [black]0[-][#1a1b26]All[-] [black]/[-][#1a1b26]Filter[-] [black]:[-][#1a1b26]Cmd[-] [black]Ctrl+E[-][#1a1b26]AI[-] [black]?[-][#1a1b26]Help[-] [black]q[-][#1a1b26]Quit[-]"

	// Add resource-specific shortcuts
	switch resource {
	case "pods", "po":
		shortcuts = "[black]l[-][#1a1b26]Logs[-] [black]s[-][#1a1b26]Shell[-] [black]d[-][#1a1b26]Describe[-] " + shortcuts
	case "deployments", "deploy", "statefulsets", "sts", "daemonsets", "ds":
		shortcuts = "[black]S[-][#1a1b26]Scale[-] [black]R[-][#1a1b26]Restart[-] [black]d[-][#1a1b26]Describe[-] " + shortcuts
	case "namespaces", "ns":
		shortcuts = "[black]u[-][#1a1b26]Use[-] " + shortcuts
	default:
		shortcuts = "[black]d[-][#1a1b26]Describe[-] [black]y[-][#1a1b26]YAML[-] " + shortcuts
	}

	// Append sort/filter status indicators
	var indicators []string
	if sortCol >= 0 && sortCol < len(headers) {
		dir := "↑"
		if !sortAsc {
			dir = "↓"
		}
		indicators = append(indicators, fmt.Sprintf("[#1a1b26]Sort:%s%s[-]", headers[sortCol], dir))
	}
	if filter != "" {
		mode, pattern := detectFilterMode(filter)
		switch mode {
		case filterModeFuzzy:
			indicators = append(indicators, fmt.Sprintf("[#1a1b26]Fuzzy:%s[-]", pattern))
		case filterModeLabel:
			indicators = append(indicators, fmt.Sprintf("[#1a1b26]Label:%s[-]", pattern))
		default:
			indicators = append(indicators, fmt.Sprintf("[#1a1b26]Filter:%s[-]", filter))
		}
	}
	if showAI {
		indicators = append(indicators, fmt.Sprintf("[#1a1b26]AI:%dcol[-]", aiWidth))
	}
	if len(indicators) > 0 {
		shortcuts += " │ " + strings.Join(indicators, " ")
	}

	// Prepend spinner if loading
	if atomic.LoadInt32(&a.loadingCount) > 0 {
		shortcuts = ColoredSpinner(int(atomic.LoadUint32(&a.spinnerIdx)), "cyan") + " [white]Loading...[-] " + shortcuts
	}

	// Clear before setting to prevent ghosting
	a.statusBar.Clear()
	a.statusBar.SetText(shortcuts)
}

// showNamespaceHint shows numbered namespace list in hint
func (a *App) showNamespaceHint() {
	a.mx.RLock()
	namespaces := a.namespaces
	a.mx.RUnlock()

	if len(namespaces) <= 1 {
		return
	}

	var hints []string
	for i, ns := range namespaces {
		if i == 0 {
			hints = append(hints, "[gray]0[darkgray]:all")
		} else if i <= 9 {
			hints = append(hints, fmt.Sprintf("[gray]%d[darkgray]:%s", i, ns))
		}
	}

	a.cmdHint.SetText(strings.Join(hints, " "))
}

// showAutocompleteDropdown shows the autocomplete dropdown as an overlay
func (a *App) showAutocompleteDropdown(suggestions []string, selectedIdx int) {
	a.cmdDropdown.Clear()

	for _, s := range suggestions {
		desc := ""
		// Find description from commands list
		for _, c := range commands {
			if c.name == s || c.alias == s {
				desc = fmt.Sprintf("[gray]%s", c.desc)
				break
			}
		}
		// Check custom aliases
		if desc == "" && a.customAliases != nil {
			if target, ok := a.customAliases.GetAll()[s]; ok {
				desc = fmt.Sprintf("[gray]→ %s", target)
			}
		}
		a.cmdDropdown.AddItem(s, desc, 0, nil)
	}

	if selectedIdx >= 0 && selectedIdx < len(suggestions) {
		a.cmdDropdown.SetCurrentItem(selectedIdx)
	}

	// Calculate dropdown height (max 10 items + 2 border)
	height := len(suggestions)*2 + 2 // 2 lines per item (main + secondary) + border
	if height > 22 {
		height = 22
	}

	// Position dropdown at bottom of screen, above command bar
	dropdownContainer := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).                // Spacer pushes to bottom
		AddItem(a.cmdDropdown, height, 0, false). // Dropdown
		AddItem(nil, 2, 0, false)                 // Space for status bar + cmd bar

	a.pages.AddPage("autocomplete", dropdownContainer, true, true)
	// Keep focus on command input
	a.SetFocus(a.cmdInput)
}

// hideAutocompleteDropdown removes the autocomplete dropdown overlay
func (a *App) hideAutocompleteDropdown() {
	if !a.pages.HasPage("autocomplete") {
		return
	}
	// Check focus BEFORE RemovePage: tview's Pages.RemovePage triggers Focus()
	// delegation which moves focus to the table, stealing it from cmdInput.
	restoreFocus := a.cmdInput.HasFocus()
	a.pages.RemovePage("autocomplete")
	if restoreFocus {
		a.SetFocus(a.cmdInput)
	}
}

// requestSync requests a full terminal sync on the next draw cycle.
func (a *App) requestSync() {
	atomic.StoreInt32(&a.needsSync, 1)
}

// QueueUpdateDraw queues up a UI action and redraws.
func (a *App) QueueUpdateDraw(f func()) {
	if a.Application == nil || atomic.LoadInt32(&a.stopping) == 1 {
		return
	}
	if atomic.LoadInt32(&a.running) == 0 {
		f()
		return
	}
	a.ensureRedrawBroker().Schedule(f)
}

// queueUpdateDrawDirect queues an update without spawning a goroutine.
func (a *App) queueUpdateDrawDirect(f func()) {
	if a.Application == nil || atomic.LoadInt32(&a.stopping) == 1 {
		return
	}
	if atomic.LoadInt32(&a.running) == 0 {
		f()
		return
	}
	a.Application.QueueUpdateDraw(f)
}

func clampAIPanelWidth(width int) int {
	if width == 0 {
		width = defaultAIPanelWidth
	}
	if width < minAIPanelWidth {
		return minAIPanelWidth
	}
	if width > maxAIPanelWidth {
		return maxAIPanelWidth
	}
	return width
}

func (a *App) currentAIPanelWidth() int {
	a.mx.RLock()
	width := a.aiPanelWidth
	a.mx.RUnlock()
	return clampAIPanelWidth(width)
}

func (a *App) isAIFocused(primitive tview.Primitive) bool {
	switch primitive {
	case a.aiInput, a.aiInputFrame, a.aiPanel, a.aiContainer, a.aiMetaBar, a.aiStatusBar:
		return true
	default:
		return false
	}
}

func (a *App) resizeAIPanelTo(width int) {
	currentFocus := a.GetFocus()
	focusAI := a.isAIFocused(currentFocus)

	a.mx.Lock()
	currentWidth := clampAIPanelWidth(a.aiPanelWidth)
	nextWidth := clampAIPanelWidth(width)
	showAI := a.showAIPanel
	a.aiPanelWidth = nextWidth
	a.mx.Unlock()

	if showAI {
		a.QueueUpdateDraw(func() {
			a.rebuildContentLayout(focusAI)
			if focusAI && currentFocus != nil {
				a.SetFocus(currentFocus)
			}
		})
	}

	if nextWidth != currentWidth {
		a.flashMsg(fmt.Sprintf("AI panel width: %d columns", nextWidth), false)
	}
}

func (a *App) adjustAIPanelWidth(delta int) {
	if delta == 0 {
		return
	}
	a.resizeAIPanelTo(a.currentAIPanelWidth() + delta)
}

func (a *App) resetAIPanelWidth() {
	a.resizeAIPanelTo(defaultAIPanelWidth)
}

func (a *App) rebuildContentLayout(focusAI bool) {
	if a.contentFlex == nil {
		return
	}
	a.contentFlex.Clear()
	a.contentFlex.AddItem(a.table, 0, 3, !focusAI)
	if a.showAIPanel && a.aiContainer != nil {
		a.contentFlex.AddItem(a.aiContainer, a.currentAIPanelWidth(), 0, focusAI)
	}
	if focusAI && a.showAIPanel {
		a.focusAIInput()
	} else {
		a.SetFocus(a.table)
	}
	a.requestSync()
	a.updateStatusBar()
	a.applyAIChrome()
}

// startLoading increments the background task counter
func (a *App) startLoading() {
	atomic.AddInt32(&a.loadingCount, 1)
	a.QueueUpdateDraw(func() { a.updateStatusBar() })
}

// stopLoading decrements the background task counter
func (a *App) stopLoading() {
	if atomic.AddInt32(&a.loadingCount, -1) <= 0 {
		atomic.StoreInt32(&a.loadingCount, 0)
		a.QueueUpdateDraw(func() { a.updateStatusBar() })
	}
}
