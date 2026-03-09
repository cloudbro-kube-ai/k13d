package ui

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// VimViewer is a text viewer with Vim-style navigation and search
type VimViewer struct {
	*tview.TextView
	app           *App
	pageName      string
	mu            sync.RWMutex // Protects mutable state below
	searchPattern string
	searchRegex   *regexp.Regexp
	searchMatches []int // Line numbers with matches
	currentMatch  int   // Current match index
	searchMode    bool  // True when in search input mode
	searchInput   string
	content       string   // Original content for searching
	lines         []string // Split lines for navigation
	totalLines    int      // Total line count

	// Secret decode toggle
	isSecretView  bool   // True when viewing a Secret resource
	secretDecoded bool   // True when base64 values are decoded
	rawYAML       string // Original YAML content for secret toggle

	// Log viewer enhancements
	isLogView  bool // True when viewing logs
	autoScroll bool // Toggle with 's'
	textWrap   bool // Toggle with 'w'
}

// NewVimViewer creates a new viewer with Vim-style keybindings
func NewVimViewer(app *App, pageName, title string) *VimViewer {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	tv.SetBorder(true).
		SetTitle(title).
		SetTitleAlign(tview.AlignLeft)

	v := &VimViewer{
		TextView:     tv,
		app:          app,
		pageName:     pageName,
		searchMode:   false,
		currentMatch: -1,
	}

	v.setupInputCapture()
	return v
}

// SetContent sets the viewer content and prepares search indexes
func (v *VimViewer) SetContent(content string) {
	v.mu.Lock()
	v.content = content
	v.lines = strings.Split(content, "\n")
	v.totalLines = len(v.lines)
	v.mu.Unlock()
	v.TextView.Clear()
	v.TextView.SetText(content)
}

// setupInputCapture configures Vim-style keybindings
func (v *VimViewer) setupInputCapture() {
	v.TextView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle search input mode
		if v.searchMode {
			return v.handleSearchInput(event)
		}

		// Normal mode keybindings
		switch event.Key() {
		case tcell.KeyEsc:
			// Clear search or close viewer
			if v.searchPattern != "" {
				v.clearSearch()
				return nil
			}
			v.close()
			return nil

		case tcell.KeyCtrlD:
			// Half page down (Vim Ctrl+D)
			v.scrollHalfPageDown()
			return nil

		case tcell.KeyCtrlU:
			// Half page up (Vim Ctrl+U)
			v.scrollHalfPageUp()
			return nil

		case tcell.KeyCtrlF:
			// Full page down (Vim Ctrl+F)
			v.scrollPageDown()
			return nil

		case tcell.KeyCtrlB:
			// Full page up (Vim Ctrl+B)
			v.scrollPageUp()
			return nil

		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				// Quit viewer
				v.close()
				return nil

			case '/':
				// Enter search mode
				v.enterSearchMode()
				return nil

			case 'n':
				// Next search match
				v.nextMatch()
				return nil

			case 'N':
				// Previous search match
				v.prevMatch()
				return nil

			case 'g':
				// Go to top (gg in Vim, single g here)
				v.ScrollToBeginning()
				return nil

			case 'G':
				// Go to bottom
				v.ScrollToEnd()
				return nil

			case 'j':
				// Scroll down one line
				row, col := v.GetScrollOffset()
				v.ScrollTo(row+1, col)
				return nil

			case 'k':
				// Scroll up one line
				row, col := v.GetScrollOffset()
				if row > 0 {
					v.ScrollTo(row-1, col)
				}
				return nil

			case 'x':
				// Toggle secret base64 decode
				if v.isSecretView {
					v.toggleSecretDecode()
					return nil
				}

			case 's':
				// Toggle auto-scroll for log view
				if v.isLogView {
					v.autoScroll = !v.autoScroll
					v.updateTitle()
					if v.autoScroll {
						v.ScrollToEnd()
					}
					return nil
				}

			case 't':
				// Timestamp filter hint for log view
				if v.isLogView {
					if v.app != nil {
						v.app.flashMsg("Use /pattern to search timestamps in logs", false)
					}
					return nil
				}

			case 'w':
				// Toggle text wrap for log view
				if v.isLogView {
					v.textWrap = !v.textWrap
					v.SetWrap(v.textWrap)
					v.updateTitle()
					return nil
				}

			case 'm':
				// Insert visual separator mark in log view
				if v.isLogView {
					current := v.TextView.GetText(false)
					separator := "\n────────── mark ──────────\n"
					v.SetContent(current + separator)
					v.ScrollToEnd()
					return nil
				}
			}
		}

		return event
	})
}

// close closes the viewer and returns focus to the table
func (v *VimViewer) close() {
	v.app.closeModal(v.pageName)
	v.app.SetFocus(v.app.table)
}

// toggleSecretDecode switches between encoded/decoded Secret values
func (v *VimViewer) toggleSecretDecode() {
	v.secretDecoded = !v.secretDecoded
	if v.secretDecoded {
		v.SetContent(decodeSecretYAML(v.rawYAML))
	} else {
		v.SetContent(v.rawYAML)
	}
	v.updateTitle()
}

// decodeSecretYAML finds base64-encoded values in the data: section and decodes them
func decodeSecretYAML(yamlContent string) string {
	lines := strings.Split(yamlContent, "\n")
	var result []string
	inDataSection := false
	dataIndent := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect entering the "data:" section
		if trimmed == "data:" {
			inDataSection = true
			dataIndent = len(line) - len(strings.TrimLeft(line, " "))
			result = append(result, line)
			continue
		}

		// Detect leaving data section (new top-level key at same or lesser indent)
		if inDataSection && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			lineIndent := len(line) - len(strings.TrimLeft(line, " "))
			if lineIndent <= dataIndent && strings.Contains(trimmed, ":") {
				inDataSection = false
			}
		}

		// Skip stringData section - don't decode those
		if trimmed == "stringData:" {
			inDataSection = false
			result = append(result, line)
			continue
		}

		if inDataSection && strings.Contains(line, ":") {
			// Parse key: value pairs inside data section
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				value := strings.TrimSpace(parts[1])
				if value != "" && value != "|" && value != ">" {
					decoded, err := base64.StdEncoding.DecodeString(value)
					if err == nil {
						// Replace with decoded value, mark it
						result = append(result, fmt.Sprintf("%s: %s", parts[0], string(decoded)))
						continue
					}
				}
			}
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// scrollHalfPageDown scrolls down by half the visible lines
func (v *VimViewer) scrollHalfPageDown() {
	_, _, _, height := v.GetInnerRect()
	halfPage := height / 2
	if halfPage < 1 {
		halfPage = 10 // Default fallback
	}

	row, col := v.GetScrollOffset()
	v.ScrollTo(row+halfPage, col)
}

// scrollHalfPageUp scrolls up by half the visible lines
func (v *VimViewer) scrollHalfPageUp() {
	_, _, _, height := v.GetInnerRect()
	halfPage := height / 2
	if halfPage < 1 {
		halfPage = 10
	}

	row, col := v.GetScrollOffset()
	newRow := row - halfPage
	if newRow < 0 {
		newRow = 0
	}
	v.ScrollTo(newRow, col)
}

// scrollPageDown scrolls down by full page
func (v *VimViewer) scrollPageDown() {
	_, _, _, height := v.GetInnerRect()
	if height < 1 {
		height = 20
	}

	row, col := v.GetScrollOffset()
	v.ScrollTo(row+height, col)
}

// scrollPageUp scrolls up by full page
func (v *VimViewer) scrollPageUp() {
	_, _, _, height := v.GetInnerRect()
	if height < 1 {
		height = 20
	}

	row, col := v.GetScrollOffset()
	newRow := row - height
	if newRow < 0 {
		newRow = 0
	}
	v.ScrollTo(newRow, col)
}

// enterSearchMode enters search input mode
func (v *VimViewer) enterSearchMode() {
	v.searchMode = true
	v.searchInput = ""
	v.updateTitle()
}

// handleSearchInput handles keyboard input during search mode
func (v *VimViewer) handleSearchInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEsc:
		// Cancel search
		v.searchMode = false
		v.searchInput = ""
		v.updateTitle()
		return nil

	case tcell.KeyEnter:
		// Execute search
		v.searchMode = false
		v.executeSearch(v.searchInput)
		return nil

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		// Delete last character
		if len(v.searchInput) > 0 {
			v.searchInput = v.searchInput[:len(v.searchInput)-1]
			v.updateTitle()
		}
		return nil

	case tcell.KeyRune:
		// Add character to search
		v.searchInput += string(event.Rune())
		v.updateTitle()
		return nil
	}

	return event
}

// executeSearch performs the search and highlights matches
func (v *VimViewer) executeSearch(pattern string) {
	if pattern == "" {
		v.clearSearch()
		return
	}

	v.searchPattern = pattern
	v.searchMatches = nil
	v.currentMatch = -1

	// Try to compile as regex, fall back to literal search
	regex, err := regexp.Compile("(?i)" + regexp.QuoteMeta(pattern))
	if err != nil {
		regex = regexp.MustCompile(regexp.QuoteMeta(pattern))
	}
	v.searchRegex = regex

	// Find all matching lines
	for i, line := range v.lines {
		if regex.MatchString(line) {
			v.searchMatches = append(v.searchMatches, i)
		}
	}

	// Update display with highlighted matches
	v.highlightMatches()

	// Jump to first match
	if len(v.searchMatches) > 0 {
		v.currentMatch = 0
		v.jumpToMatch(v.searchMatches[0])
	}

	v.updateTitle()
}

// highlightMatches highlights search matches in the content
func (v *VimViewer) highlightMatches() {
	if v.searchRegex == nil || v.searchPattern == "" {
		v.TextView.SetText(v.content)
		return
	}

	// Highlight matches with yellow background
	highlighted := v.searchRegex.ReplaceAllStringFunc(v.content, func(match string) string {
		return "[black:yellow]" + match + "[white:-]"
	})

	v.TextView.SetText(highlighted)
}

// clearSearch clears the current search
func (v *VimViewer) clearSearch() {
	v.searchPattern = ""
	v.searchRegex = nil
	v.searchMatches = nil
	v.currentMatch = -1
	v.TextView.SetText(v.content)
	v.updateTitle()
}

// nextMatch jumps to the next search match
func (v *VimViewer) nextMatch() {
	if len(v.searchMatches) == 0 {
		return
	}

	v.currentMatch++
	if v.currentMatch >= len(v.searchMatches) {
		v.currentMatch = 0 // Wrap around
	}

	v.jumpToMatch(v.searchMatches[v.currentMatch])
	v.updateTitle()
}

// prevMatch jumps to the previous search match
func (v *VimViewer) prevMatch() {
	if len(v.searchMatches) == 0 {
		return
	}

	v.currentMatch--
	if v.currentMatch < 0 {
		v.currentMatch = len(v.searchMatches) - 1 // Wrap around
	}

	v.jumpToMatch(v.searchMatches[v.currentMatch])
	v.updateTitle()
}

// jumpToMatch scrolls to show the specified line
func (v *VimViewer) jumpToMatch(lineNum int) {
	// Center the match in the view
	_, _, _, height := v.GetInnerRect()
	if height < 1 {
		height = 20
	}

	targetRow := lineNum - height/2
	if targetRow < 0 {
		targetRow = 0
	}

	v.ScrollTo(targetRow, 0)
}

// updateTitle updates the viewer title to show search state and mode flags
func (v *VimViewer) updateTitle() {
	baseTitle := v.getBaseTitle()

	suffix := ""

	// Secret decode indicator
	if v.isSecretView {
		if v.secretDecoded {
			suffix += " [green][decoded][white]"
		}
		suffix += " [gray](x:toggle decode)[white]"
	}

	// Log viewer flags
	if v.isLogView {
		var flags []string
		if v.autoScroll {
			flags = append(flags, "auto")
		}
		if v.textWrap {
			flags = append(flags, "wrap")
		}
		if len(flags) > 0 {
			suffix += " [yellow][" + strings.Join(flags, ",") + "][white]"
		}
	}

	if v.searchMode {
		v.SetTitle(baseTitle + suffix + " [yellow]/" + v.searchInput + "_[white]")
	} else if v.searchPattern != "" {
		matchInfo := ""
		if len(v.searchMatches) > 0 {
			matchInfo = fmt.Sprintf(" [green]%s[white] (%d/%d)",
				v.searchPattern, v.currentMatch+1, len(v.searchMatches))
		} else {
			matchInfo = " [red]" + v.searchPattern + "[white] (no matches)"
		}
		v.SetTitle(baseTitle + suffix + matchInfo)
	} else {
		v.SetTitle(baseTitle + suffix)
	}
}

// getBaseTitle returns the base title without search info or mode flags
func (v *VimViewer) getBaseTitle() string {
	title := v.TextView.GetTitle()
	// Remove mode flags and search info (find earliest marker)
	markers := []string{" [/", " [green]", " [red]", " [gray](x:", " [yellow]["}
	minIdx := len(title)
	for _, m := range markers {
		if idx := strings.Index(title, m); idx > 0 && idx < minIdx {
			minIdx = idx
		}
	}
	if minIdx < len(title) {
		return title[:minIdx]
	}
	return title
}

// GetTitle returns the base title without search info (kept for compatibility)
func (v *VimViewer) GetTitle() string {
	return v.getBaseTitle()
}
