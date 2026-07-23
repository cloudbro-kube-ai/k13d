package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	// welcomeLogo is the large ASCII art for the welcome screen
	welcomeLogo = `██╗  ██╗ ██╗██████╗ ██████╗
██║ ██╔╝███║╚════██╗██╔══██╗
██████╔╝ ╚██║ █████╔╝██║  ██║
██╔═██╗  ██║ ╚═══██╗██║  ██║
██║  ██╗ ██║██████╔╝██████╔╝
╚═╝  ╚═╝ ╚═╝╚═════╝ ╚═════╝ `

	welcomeTagline = "Kubernetes AI Dashboard CLI"
)

// WelcomeScreen is the initial screen shown when k13d starts
type WelcomeScreen struct {
	*tview.Flex
	app        *App
	logo       *tview.TextView
	subtitle   *tview.TextView
	menu       *tview.List
	inputLabel *tview.TextView
	input      *tview.InputField
	hintBar    *tview.TextView
	onComplete func(cmd string)
}

// NewWelcomeScreen creates a new welcome screen
func NewWelcomeScreen(app *App, onComplete func(cmd string)) *WelcomeScreen {
	w := &WelcomeScreen{
		app:        app,
		onComplete: onComplete,
	}

	// Logo view - centered ASCII art with gradient
	w.logo = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	w.renderLogo(1.0)

	// Subtitle with version
	w.subtitle = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	w.subtitle.SetText(fmt.Sprintf(
		"[#565f89]%s[-]  [#3b4261]v%s[-]",
		welcomeTagline, Version,
	))

	// Menu - quick command suggestions
	w.menu = tview.NewList().
		ShowSecondaryText(true).
		SetSelectedBackgroundColor(tcell.NewRGBColor(122, 162, 247)). // #7aa2f7
		SetSelectedTextColor(tcell.NewRGBColor(26, 27, 38)).          // #1a1b26
		SetMainTextColor(tcell.NewRGBColor(192, 202, 245)).           // #c0caf5
		SetSecondaryTextColor(tcell.NewRGBColor(86, 95, 137))         // #565f89

	// Add quick-start menu items
	w.menu.AddItem(":pods", "List pods in current namespace", 'p', nil)
	w.menu.AddItem(":svc", "List services", 's', nil)
	w.menu.AddItem(":deploy", "List deployments", 'd', nil)
	w.menu.AddItem(":ctx", "Switch Kubernetes context", 'c', nil)
	w.menu.AddItem(":ns", "Switch namespace", 'n', nil)
	w.menu.AddItem("Start Dashboard", "Go to main dashboard with defaults", 'e', nil)

	w.menu.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		switch index {
		case 0:
			w.onComplete(":pods")
		case 1:
			w.onComplete(":svc")
		case 2:
			w.onComplete(":deploy")
		case 3:
			w.onComplete(":ctx")
		case 4:
			w.onComplete(":ns")
		case 5:
			w.onComplete("")
		}
	})

	// Input label
	w.inputLabel = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	w.inputLabel.SetText("[#7aa2f7]> [white]")

	// Command input
	w.input = tview.NewInputField().
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetFieldTextColor(tcell.NewRGBColor(192, 202, 245)). // #c0caf5
		SetPlaceholder("Type a command or press Enter to start...").
		SetPlaceholderStyle(tcell.StyleDefault.Foreground(tcell.NewRGBColor(86, 95, 137))) // #565f89
	w.input.SetLabel("[#7aa2f7]⟩ [white]")

	// Hint bar at the bottom
	w.hintBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	w.hintBar.SetText("[#3b4261]↑↓ navigate  Enter select  Tab switch panel  Esc quit[-]")

	// Input container with border
	inputContainer := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(w.input, 1, 0, true)
	inputContainer.SetBorder(true).
		SetTitle(" Command ").
		SetBorderColor(tcell.NewRGBColor(122, 162, 247)). // #7aa2f7
		SetTitleColor(tcell.NewRGBColor(122, 162, 247))

	// Menu container with border
	menuContainer := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(w.menu, 0, 1, false)
	menuContainer.SetBorder(true).
		SetTitle(" Quick Start ").
		SetBorderColor(tcell.NewRGBColor(187, 154, 247)). // #bb9af7
		SetTitleColor(tcell.NewRGBColor(187, 154, 247))

	// Left panel: menu
	leftPanel := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 1, 0, false). // Top spacer
		AddItem(menuContainer, 0, 1, false).
		AddItem(nil, 1, 0, false) // Bottom spacer

	// Right panel: info
	rightContent := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetWrap(true)
	rightContent.SetText(w.buildInfoText())

	rightPanel := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 1, 0, false).
		AddItem(rightContent, 0, 1, false).
		AddItem(nil, 1, 0, false)

	// Main content area: left menu + right info
	contentArea := tview.NewFlex().
		AddItem(leftPanel, 0, 3, false).
		AddItem(rightPanel, 0, 2, false)

	// Assemble the full layout
	w.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 2, false). // Top spacer (pushes content down)
		AddItem(w.logo, 6, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(w.subtitle, 1, 0, false).
		AddItem(nil, 2, 0, false).
		AddItem(contentArea, 0, 3, false).
		AddItem(nil, 1, 0, false).
		AddItem(inputContainer, 3, 0, true).
		AddItem(nil, 1, 0, false).
		AddItem(w.hintBar, 1, 0, false)

	return w
}

// buildInfoText builds the right-side info panel content
func (w *WelcomeScreen) buildInfoText() string {
	ctxName := "N/A"
	cluster := "N/A"
	if w.app.k8s != nil {
		var err error
		ctxName, cluster, _, err = w.app.k8s.GetContextInfo()
		if err != nil {
			ctxName = "disconnected"
			cluster = "N/A"
		}
	}

	aiStatus := "[#f7768e]Offline[-]"
	if w.app.aiClient != nil && w.app.aiClient.IsReady() {
		aiStatus = "[#9ece6a]Online[-]"
	}

	nsDisplay := "[#9ece6a]all[-]"
	if w.app.currentNamespace != "" {
		nsDisplay = "[#9ece6a]" + w.app.currentNamespace + "[-]"
	}

	return fmt.Sprintf(`[#c0caf5::b]Cluster Info[-::-]

  [#565f89]Context:[-]    [#7aa2f7]%s[-]
  [#565f89]Cluster:[-]    [#7aa2f7]%s[-]
  [#565f89]Namespace:[-]  %s
  [#565f89]AI:[-]         %s

[#565f89]─────────────────────────[-]

[#c0caf5::b]Keyboard Shortcuts[-::-]

  [#e0af68]/[-]         Filter resources
  [#e0af68]:[-]         Command mode
  [#e0af68]j/k[-]       Navigate up/down
  [#e0af68]q[-]         Quit
  [#e0af68]?[-]         Help

[#565f89]─────────────────────────[-]

[#c0caf5::b]Resources[-::-]

  [#565f89]Pods:[-]      #9ece6a●[-]
  [#565f89]Services:[-]  #9ece6a●[-]
  [#565f89]Deploy:[-]    #9ece6a●[-]`,
		ctxName, cluster, nsDisplay, aiStatus,
	)
}

// renderLogo renders the ASCII logo with gradient effect
func (w *WelcomeScreen) renderLogo(progress float64) {
	lines := strings.Split(welcomeLogo, "\n")
	var result strings.Builder

	// Gradient colors from cyan to purple (Tokyo Night palette)
	colors := []string{
		"[#7dcfff]", // light cyan
		"[#7aa2f7]", // blue
		"[#bb9af7]", // purple
		"[#9d7cd8]", // dark purple
		"[#73daca]", // teal
		"[#7dcfff]", // back to cyan
	}

	visibleLines := int(float64(len(lines)) * progress)
	for i, line := range lines {
		if i >= visibleLines {
			break
		}
		if len(line) == 0 {
			continue
		}
		colorIdx := i % len(colors)
		result.WriteString(colors[colorIdx])
		result.WriteString(line)
		result.WriteString("[-]\n")
	}

	w.logo.SetText(result.String())
}

// setupInputHandlers configures keyboard handling for the welcome screen
func (w *WelcomeScreen) setupInputHandlers() {
	// Handle Enter key on input field
	w.input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			text := w.input.GetText()
			w.input.SetText("")
			if text != "" {
				w.onComplete(text)
			} else {
				// Empty input = start dashboard
				w.onComplete("")
			}
		case tcell.KeyEsc:
			w.onComplete("quit")
		case tcell.KeyTab:
			// Switch focus to menu
			w.app.SetFocus(w.menu)
		}
	})

	// Handle key events on the menu
	w.menu.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			// Let menu handle selection
			return event
		case tcell.KeyEsc:
			w.onComplete("quit")
			return nil
		case tcell.KeyTab:
			// Switch focus to input
			w.app.SetFocus(w.input)
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case ':', '/':
				// Switch to input and add the prefix
				w.app.SetFocus(w.input)
				w.input.SetText(string(event.Rune()))
				return nil
			case 'q':
				w.onComplete("quit")
				return nil
			case 'e', '\n':
				// Start dashboard
				w.onComplete("")
				return nil
			}
		}
		return event
	})

	// Handle key events on the entire welcome screen
	w.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			w.onComplete("quit")
			return nil
		case tcell.KeyTab:
			// Toggle between menu and input
			if w.input.HasFocus() {
				w.app.SetFocus(w.menu)
			} else {
				w.app.SetFocus(w.input)
			}
			return nil
		}
		return event
	})
}

// Show displays the welcome screen with animation
func (w *WelcomeScreen) Show(app *tview.Application) {
	w.setupInputHandlers()

	// Set as root and focus input
	app.SetRoot(w.Flex, true)
	app.SetFocus(w.input)

	// Animate logo reveal
	go func() {
		steps := 12
		for i := 1; i <= steps; i++ {
			progress := float64(i) / float64(steps)
			app.QueueUpdateDraw(func() {
				w.renderLogo(progress)
			})
			time.Sleep(30 * time.Millisecond)
		}
		// Final state - full logo
		app.QueueUpdateDraw(func() {
			w.renderLogo(1.0)
		})
	}()
}

// WelcomeModal wraps the welcome screen as a modal overlay
func WelcomeModal(app *App, onComplete func(cmd string)) *tview.Flex {
	welcome := NewWelcomeScreen(app, onComplete)

	// Full-screen overlay
	modal := tview.NewFlex().
		AddItem(welcome, 0, 1, true)
	modal.SetBackgroundColor(tcell.NewRGBColor(26, 27, 38)) // #1a1b26

	return modal
}
