package ui

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// refresh reloads the current resource list with atomic guard (k9s pattern)
func (a *App) refresh() {
	// Atomic guard to prevent concurrent updates (k9s pattern)
	for i := 0; i < 10; i++ {
		if atomic.CompareAndSwapInt32(&a.inUpdate, 0, 1) {
			break
		}
		if i == 9 {
			a.logger.Debug("Dropping refresh - update still in progress after retries")
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	defer atomic.StoreInt32(&a.inUpdate, 0)

	a.startLoading()
	defer a.stopLoading()

	ctx := a.prepareContext()

	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	// Show loading state
	a.queueUpdateDrawDirect(func() {
		a.table.Clear()
		a.table.SetTitle(fmt.Sprintf(" %s - Loading... ", resource))
		a.table.SetCell(0, 0, tview.NewTableCell("Loading...").SetTextColor(tcell.ColorYellow))
	})

	// Fetch with exponential backoff
	var headers []string
	var rows [][]string
	var fetchErr error

	bf := backoff.NewExponentialBackOff()
	bf.InitialInterval = 300 * time.Millisecond
	bf.MaxElapsedTime = 10 * time.Second

	err := backoff.Retry(func() error {
		select {
		case <-ctx.Done():
			return backoff.Permanent(ctx.Err())
		default:
		}

		headers, rows, fetchErr = a.fetchResources(ctx)
		if fetchErr != nil {
			a.logger.Warn("Fetch failed, retrying", "error", fetchErr, "resource", resource)
			return fetchErr
		}
		return nil
	}, backoff.WithContext(bf, ctx))

	if err != nil {
		if ctx.Err() != nil {
			a.logger.Debug("Refresh cancelled", "resource", resource)
			return
		}
		a.logger.Error("Fetch failed after retries", "error", err, "resource", resource)
		a.flashMsg(fmt.Sprintf("Failed to load %s: %v", resource, err), true)
		a.queueUpdateDrawDirect(func() {
			a.table.Clear()
			a.table.SetTitle(fmt.Sprintf(" %s - Error ", resource))
			a.table.SetCell(0, 0, tview.NewTableCell(fmt.Sprintf("Error: %v", err)).SetTextColor(tcell.ColorRed))
		})
		return
	}

	a.mx.Lock()
	a.tableHeaders = headers
	a.tableRows = rows
	currentFilter := a.filterText
	a.mx.Unlock()

	if currentFilter != "" {
		a.applyFilterText(currentFilter)
	} else {
		a.queueUpdateDrawDirect(func() {
			a.table.Clear()
			for i, h := range headers {
				cell := tview.NewTableCell(h).
					SetTextColor(tcell.ColorYellow).
					SetAttributes(tcell.AttrBold).
					SetSelectable(false).
					SetExpansion(1)
				a.table.SetCell(0, i, cell)
			}
			for r, row := range rows {
				for c, text := range row {
					color := tcell.ColorWhite
					if a.isStatusColumn(c) {
						color = a.statusColor(text)
					}
					cell := tview.NewTableCell(text).
						SetTextColor(color).
						SetExpansion(1)
					a.table.SetCell(r+1, c, cell)
				}
			}
			count := len(rows)
			a.table.SetTitle(fmt.Sprintf(" %s (%d) ", resource, count))
			if count > 0 {
				a.table.Select(1, 0)
			}
			a.refreshTableDecorations()
			a.applyAIChrome()
		})
	}

	a.queueUpdateDrawDirect(func() {
		a.updateStatusBar()
		a.applyAIChrome()
	})

	if a.briefing != nil && a.briefing.IsVisible() {
		a.safeGo("briefing-update", func() { _ = a.briefing.Update(ctx) })
	}

	a.logger.Info("Refresh completed", "resource", resource, "count", len(rows))
}

// startFilter activates filter mode
func (a *App) startFilter() {
	a.cmdInput.SetLabel(" / ")
	a.cmdHint.SetText("[gray]Filter: text | /regex/ | -f fuzzy | -l label=value | Esc to clear")
	a.cmdInput.SetText(a.filterText)
	a.SetFocus(a.cmdInput)

	var filterTimer *time.Timer
	var filterMu sync.Mutex

	a.cmdInput.SetChangedFunc(func(text string) {
		filterMu.Lock()
		if filterTimer != nil {
			filterTimer.Stop()
		}
		filterTimer = time.AfterFunc(100*time.Millisecond, func() {
			a.applyFilterText(text)
		})
		filterMu.Unlock()
	})

	a.cmdInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			text := a.cmdInput.GetText()
			a.mx.Lock()
			if strings.HasPrefix(text, "/") && strings.HasSuffix(text, "/") && len(text) > 2 {
				a.filterText = text[1 : len(text)-1]
				a.filterRegex = true
			} else {
				a.filterText = text
				a.filterRegex = false
			}
			a.mx.Unlock()
			a.cmdInput.SetLabel(" : ")
			a.cmdHint.SetText("")
			a.restoreAutocompleteHandler()
			a.SetFocus(a.table)
			return nil
		case tcell.KeyEsc:
			a.mx.Lock()
			a.filterText = ""
			a.filterRegex = false
			a.mx.Unlock()
			a.cmdInput.SetText("")
			a.cmdInput.SetLabel(" : ")
			a.cmdHint.SetText("")
			a.applyFilterText("")
			a.restoreAutocompleteHandler()
			a.SetFocus(a.table)
			return nil
		}
		return event
	})
}

// applyFilterText filters the table
func (a *App) applyFilterText(filter string) {
	a.mx.RLock()
	headers := a.tableHeaders
	rows := a.tableRows
	resource := a.currentResource
	sortCol := a.sortColumn
	sortAsc := a.sortAscending
	a.mx.RUnlock()

	if len(headers) == 0 || len(rows) == 0 {
		return
	}

	mode, pattern := detectFilterMode(filter)
	isRegex := false
	filterPattern := pattern
	if mode == filterModeText {
		if strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") && len(pattern) > 2 {
			filterPattern = pattern[1 : len(pattern)-1]
			isRegex = true
		}
	}

	var re *regexp.Regexp
	var err error
	if isRegex && filterPattern != "" {
		re, err = regexp.Compile("(?i)" + filterPattern)
		if err != nil {
			isRegex = false
			filterPattern = pattern
		}
	}

	var filteredRows [][]string
	switch mode {
	case filterModeFuzzy:
		if pattern != "" {
			nameCol := nameColumnIndex(resource)
			filteredRows = FuzzyFilter(rows, pattern, nameCol)
		} else {
			filteredRows = rows
		}
	case filterModeLabel:
		if pattern != "" {
			filteredRows = LabelFilter(rows, pattern)
		} else {
			filteredRows = rows
		}
	default:
		filteredRows = nil
	}

	filterLower := strings.ToLower(filterPattern)

	a.QueueUpdateDraw(func() {
		a.table.Clear()
		for i, h := range headers {
			displayHeader := h
			headerColor := tcell.NewRGBColor(224, 175, 104)
			if sortCol == i {
				if sortAsc {
					displayHeader = h + " ▲"
				} else {
					displayHeader = h + " ▼"
				}
				headerColor = tcell.NewRGBColor(125, 207, 255)
			}
			cell := tview.NewTableCell(displayHeader).
				SetTextColor(headerColor).
				SetAttributes(tcell.AttrBold).
				SetSelectable(false).
				SetExpansion(1).
				SetBackgroundColor(tcell.NewRGBColor(36, 40, 59))
			a.table.SetCell(0, i, cell)
		}

		renderRows := filteredRows
		if mode == filterModeText {
			renderRows = rows
		}

		rowIdx := 1
		for _, row := range renderRows {
			if mode == filterModeText && filterPattern != "" {
				match := false
				for _, cell := range row {
					if isRegex && re != nil {
						if re.MatchString(cell) {
							match = true
							break
						}
					} else {
						if strings.Contains(strings.ToLower(cell), filterLower) {
							match = true
							break
						}
					}
				}
				if !match {
					continue
				}
			}

			nameCol := nameColumnIndex(resource)
			for c, text := range row {
				color := tcell.ColorWhite
				if a.isStatusColumn(c) {
					color = a.statusColor(text)
				}
				displayText := text
				switch mode {
				case filterModeFuzzy:
					if pattern != "" && c == nameCol {
						displayText = highlightFuzzyMatch(text, pattern)
					}
				case filterModeLabel:
				default:
					if filterPattern != "" {
						if isRegex && re != nil {
							displayText = a.highlightRegexMatch(text, re)
						} else if strings.Contains(strings.ToLower(text), filterLower) {
							displayText = a.highlightMatch(text, filterLower)
						}
					}
				}
				cell := tview.NewTableCell(displayText).
					SetTextColor(color).
					SetExpansion(1)
				a.table.SetCell(rowIdx, c, cell)
			}
			rowIdx++
		}

		filterInfo := ""
		if filter != "" {
			switch mode {
			case filterModeFuzzy:
				filterInfo = fmt.Sprintf(" [fuzzy: %s]", pattern)
			case filterModeLabel:
				filterInfo = fmt.Sprintf(" [label: %s]", pattern)
			default:
				if isRegex {
					filterInfo = fmt.Sprintf(" [regex: %s]", filterPattern)
				} else {
					filterInfo = fmt.Sprintf(" [filter: %s]", filter)
				}
			}
		}
		a.table.SetTitle(fmt.Sprintf(" %s (%d/%d)%s ", resource, rowIdx-1, len(rows), filterInfo))
		if rowIdx > 1 {
			a.table.Select(1, 0)
		}
		a.refreshTableDecorations()
	})
	a.updateStatusBar()
}

// highlightMatch wraps matching text with color tags
func (a *App) highlightMatch(text, filter string) string {
	lower := strings.ToLower(text)
	idx := strings.Index(lower, filter)
	if idx < 0 {
		return text
	}
	before := text[:idx]
	match := text[idx : idx+len(filter)]
	after := text[idx+len(filter):]
	return before + "[yellow]" + match + "[white]" + after
}

// highlightRegexMatch wraps regex-matching text with color tags
func (a *App) highlightRegexMatch(text string, re *regexp.Regexp) string {
	if re == nil {
		return text
	}
	matches := re.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return text
	}
	var result strings.Builder
	lastEnd := 0
	for _, match := range matches {
		start, end := match[0], match[1]
		result.WriteString(text[lastEnd:start])
		result.WriteString("[yellow]")
		result.WriteString(text[start:end])
		result.WriteString("[white]")
		lastEnd = end
	}
	result.WriteString(text[lastEnd:])
	return result.String()
}

// startWatch starts a resource watcher
func (a *App) startWatch() {
	if a.k8s == nil || atomic.LoadInt32(&a.stopping) == 1 {
		return
	}
	a.watchMu.Lock()
	a.stopWatchLocked()
	a.mx.RLock()
	resource := a.currentResource
	namespace := a.currentNamespace
	a.mx.RUnlock()
	parentCtx := a.appCtx
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx, cancel := context.WithCancel(parentCtx)
	a.watchCancel = cancel
	onChange := func() {
		a.safeGo("watch-refresh", func() { a.refresh() })
	}
	cfg := k8s.DefaultWatcherConfig()
	a.watcher = k8s.NewResourceWatcher(a.k8s, resource, namespace, onChange, a.logger, cfg)
	a.watcher.Start(ctx)
	a.logger.Info("Started watch", "resource", resource, "namespace", namespace)
	a.watchMu.Unlock()
	a.updateHeader()
}

// stopWatch stops the current resource watcher
func (a *App) stopWatch() {
	a.watchMu.Lock()
	defer a.watchMu.Unlock()
	a.stopWatchLocked()
}

// stopWatchLocked stops the watcher
func (a *App) stopWatchLocked() {
	if a.watcher != nil {
		a.watcher.Stop()
		a.watcher = nil
	}
	if a.watchCancel != nil {
		a.watchCancel()
		a.watchCancel = nil
	}
}

// prepareContext cancels previous operations and creates new context
func (a *App) prepareContext() context.Context {
	a.cancelLock.Lock()
	defer a.cancelLock.Unlock()
	if a.cancelFn != nil {
		a.cancelFn()
	}
	parentCtx := a.appCtx
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx, cancel := context.WithCancel(parentCtx)
	a.cancelFn = cancel
	return ctx
}

// loadAPIResources fetches available API resources
func (a *App) loadAPIResources() {
	if a.k8s == nil {
		return
	}
	a.startLoading()
	defer a.stopLoading()
	ctx, cancel := context.WithTimeout(a.getAppContext(), 10*time.Second)
	defer cancel()
	resources, err := a.k8s.GetAPIResources(ctx)
	if err != nil {
		a.logger.Warn("Failed to load API resources", "error", err)
		resources = a.k8s.GetCommonResources()
	}
	a.mx.Lock()
	a.apiResources = resources
	a.mx.Unlock()
	a.logger.Info("Loaded API resources", "count", len(resources))
}

// loadNamespaces fetches namespaces
func (a *App) loadNamespaces() {
	if a.k8s == nil {
		return
	}
	a.startLoading()
	defer a.stopLoading()
	ctx, cancel := context.WithTimeout(a.getAppContext(), 10*time.Second)
	defer cancel()
	nss, err := a.k8s.ListNamespaces(ctx)
	if err != nil {
		a.logger.Warn("Failed to load namespaces", "error", err)
		return
	}
	namespaceList := make([]string, 0, len(nss)+1)
	namespaceList = append(namespaceList, "")
	for _, n := range nss {
		namespaceList = append(namespaceList, n.Name)
	}
	a.mx.Lock()
	a.namespaces = namespaceList
	a.mx.Unlock()
	reordered := a.reorderNamespacesByRecent()
	a.mx.Lock()
	a.namespaces = reordered
	a.mx.Unlock()
	a.updateHeader()
	a.logger.Info("Loaded namespaces", "count", len(nss))
}

// reorderNamespacesByRecent reorders namespaces list
func (a *App) reorderNamespacesByRecent() []string {
	a.mx.RLock()
	allNamespaces := make([]string, len(a.namespaces))
	copy(allNamespaces, a.namespaces)
	recent := make([]string, len(a.recentNamespaces))
	copy(recent, a.recentNamespaces)
	a.mx.RUnlock()
	result := make([]string, 0, len(allNamespaces))
	hasAll := false
	for _, ns := range allNamespaces {
		if ns == "" {
			hasAll = true
			break
		}
	}
	if hasAll {
		result = append(result, "")
	}
	nsSet := make(map[string]bool)
	for _, ns := range allNamespaces {
		nsSet[ns] = true
	}
	addedSet := make(map[string]bool)
	addedSet[""] = true
	for _, ns := range recent {
		if nsSet[ns] && !addedSet[ns] {
			result = append(result, ns)
			addedSet[ns] = true
		}
	}
	remaining := make([]string, 0)
	for _, ns := range allNamespaces {
		if !addedSet[ns] {
			remaining = append(remaining, ns)
		}
	}
	sort.Strings(remaining)
	result = append(result, remaining...)
	return result
}

// addRecentNamespace adds a namespace to the recent list
func (a *App) addRecentNamespace(ns string) {
	if ns == "" {
		return
	}
	newRecent := make([]string, 0, a.maxRecentNamespaces)
	for _, r := range a.recentNamespaces {
		if r != ns {
			newRecent = append(newRecent, r)
		}
	}
	newRecent = append([]string{ns}, newRecent...)
	if len(newRecent) > a.maxRecentNamespaces {
		newRecent = newRecent[:a.maxRecentNamespaces]
	}
	a.recentNamespaces = newRecent
}

// switchToAllNamespaces switches to all namespaces
func (a *App) switchToAllNamespaces() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()
	a.flashMsg("Switched to: all namespaces", false)
	a.navigateTo(resource, "", "")
}

// selectNamespaceByNumber selects namespace by number
func (a *App) selectNamespaceByNumber(num int) {
	a.mx.Lock()
	if num >= len(a.namespaces) {
		a.mx.Unlock()
		a.flashMsg(fmt.Sprintf("Namespace #%d not found", num), true)
		return
	}
	selectedNs := a.namespaces[num]
	nsName := selectedNs
	if nsName == "" {
		nsName = "all"
	}
	if selectedNs != "" {
		a.addRecentNamespace(selectedNs)
	}
	resource := a.currentResource
	a.mx.Unlock()
	a.flashMsg(fmt.Sprintf("Switched to namespace: %s", nsName), false)
	a.navigateTo(resource, selectedNs, "")
}

// filterMode indicates the active filter type.
type filterMode int

const (
	filterModeText filterMode = iota
	filterModeFuzzy
	filterModeLabel
)

// detectFilterMode parses a filter string
func detectFilterMode(filter string) (filterMode, string) {
	if strings.HasPrefix(filter, "-f ") {
		return filterModeFuzzy, strings.TrimPrefix(filter, "-f ")
	}
	if strings.HasPrefix(filter, "-l ") {
		return filterModeLabel, strings.TrimPrefix(filter, "-l ")
	}
	return filterModeText, filter
}

// nameColumnIndex returns the index of the name column
func nameColumnIndex(resource string) int {
	switch resource {
	case "nodes", "no", "namespaces", "ns", "persistentvolumes", "pv",
		"storageclasses", "sc", "clusterroles", "cr",
		"clusterrolebindings", "crb", "customresourcedefinitions", "crd":
		return 0
	default:
		return 1
	}
}
