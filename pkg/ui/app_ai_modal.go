package ui

import (
	"fmt"
	"strings"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai/safety"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const toolApprovalModalName = "tool-approval"

func (a *App) showToolApprovalModal(toolName, command string, decision *safety.Decision) {
	if a == nil || a.pages == nil {
		return
	}

	title, icon, borderColor, categoryLabel := toolApprovalPresentation(decision)

	modal := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(true)
	modal.SetBorder(true).
		SetTitle(" " + title + " ").
		SetBorderColor(borderColor).
		SetTitleColor(borderColor)

	var body strings.Builder
	body.WriteString(fmt.Sprintf("%s [::b]%s[::-]\n\n", icon, title))
	body.WriteString("[gray]The AI wants to execute:[-]\n")
	body.WriteString(fmt.Sprintf("[white::b]%s[-::-]\n\n", tview.Escape(trimAIBlock(command, 420))))
	body.WriteString(fmt.Sprintf("[gray]Tool:[-] [white]%s[-]\n", tview.Escape(toolName)))
	body.WriteString(fmt.Sprintf("[gray]Category:[-] %s\n", categoryLabel))

	if decision != nil && len(decision.Warnings) > 0 {
		body.WriteString("\n[yellow::b]Warnings[-::-]\n")
		for _, warning := range decision.Warnings {
			body.WriteString("[yellow]-[-] ")
			body.WriteString(tview.Escape(warning))
			body.WriteString("\n")
		}
	}

	body.WriteString("\n")
	body.WriteString("[black:#f7768e] Reject (N / Esc) [-]  ")
	body.WriteString("[black:#9ece6a] Approve (Y / Enter) [-]")
	modal.SetText(body.String())

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			a.approveToolCall(true)
			return nil
		case tcell.KeyEscape:
			a.approveToolCall(false)
			return nil
		}

		switch event.Rune() {
		case 'y', 'Y':
			a.approveToolCall(true)
			return nil
		case 'n', 'N':
			a.approveToolCall(false)
			return nil
		}

		return event
	})

	height := 13
	if decision != nil {
		height += min(len(decision.Warnings), 6)
	}

	a.aiMx.Lock()
	a.toolApprovalFocus = a.GetFocus()
	a.aiMx.Unlock()

	if a.pages.HasPage(toolApprovalModalName) {
		a.closeModal(toolApprovalModalName)
	}

	a.showModal(toolApprovalModalName, centered(modal, 82, height), true)
	a.SetFocus(modal)
}

func (a *App) closeToolApprovalModal() {
	if a == nil || a.pages == nil || !a.pages.HasPage(toolApprovalModalName) {
		return
	}

	a.closeModal(toolApprovalModalName)
	a.restoreToolApprovalFocus()
}

func (a *App) restoreToolApprovalFocus() {
	var focus tview.Primitive

	a.aiMx.Lock()
	focus = a.toolApprovalFocus
	a.toolApprovalFocus = nil
	a.aiMx.Unlock()

	if focus != nil {
		a.SetFocus(focus)
		return
	}

	a.mx.RLock()
	showAI := a.showAIPanel
	a.mx.RUnlock()

	if showAI && a.aiInput != nil {
		a.SetFocus(a.aiInput)
		return
	}
	if a.table != nil {
		a.SetFocus(a.table)
	}
}

func toolApprovalPresentation(decision *safety.Decision) (title, icon string, borderColor tcell.Color, categoryLabel string) {
	title = "Decision Required"
	icon = "[#7aa2f7]◆[-]"
	borderColor = tcell.ColorYellow
	categoryLabel = "[yellow]unknown[-]"

	if decision == nil {
		return title, icon, borderColor, categoryLabel
	}

	switch decision.Category {
	case "dangerous":
		title = "Dangerous Operation"
		icon = "[#f7768e]◆[-]"
		borderColor = tcell.NewRGBColor(247, 118, 142)
		categoryLabel = "[red::b]dangerous[-::-]"
	case "write":
		title = "Decision Required"
		icon = "[#e0af68]◆[-]"
		borderColor = tcell.NewRGBColor(224, 175, 104)
		categoryLabel = "[#e0af68]write[-]"
	case "interactive":
		title = "Interactive Command"
		icon = "[#7dcfff]◆[-]"
		borderColor = tcell.NewRGBColor(125, 207, 255)
		categoryLabel = "[#7dcfff]interactive[-]"
	case "read-only":
		title = "Approval Required"
		icon = "[#9ece6a]◆[-]"
		borderColor = tcell.NewRGBColor(158, 206, 106)
		categoryLabel = "[#9ece6a]read-only[-]"
	default:
		categoryLabel = fmt.Sprintf("[yellow]%s[-]", tview.Escape(decision.Category))
	}

	if decision.Classification != nil && decision.Classification.IsDangerous {
		title = "Dangerous Operation"
		icon = "[#f7768e]◆[-]"
		borderColor = tcell.NewRGBColor(247, 118, 142)
		categoryLabel = "[red::b]dangerous[-::-]"
	}

	return title, icon, borderColor, categoryLabel
}
