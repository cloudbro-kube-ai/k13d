package ui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai/agent"
)

// AIPanel manages the AI assistant UI component.
// It implements agent.AgentListener to receive events from the agent (k9s pattern).
// It also supports channel-based communication as an alternative.
type AIPanel struct {
	*tview.Flex

	// UI Components
	outputView *tview.TextView
	inputField *tview.InputField
	statusBar  *tview.TextView

	// Agent
	agent *agent.Agent

	// State
	isShowingApproval bool
	currentApproval   *agent.ChoiceRequest
	approvalCallback  func(bool) // For synchronous approval handling
	mu                sync.Mutex

	// Callbacks
	onSubmit func(string) // Called when user submits a question
	onFocus  func()       // Called when panel gains focus
	app      *tview.Application
}

// Ensure AIPanel implements AgentListener
var _ agent.AgentListener = (*AIPanel)(nil)

// NewAIPanel creates a new AI panel component
func NewAIPanel(app *tview.Application) *AIPanel {
	p := &AIPanel{
		app: app,
	}

	// Output view for AI responses
	p.outputView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})
	p.outputView.SetBorder(false)

	// Input field for questions
	p.inputField = tview.NewInputField().
		SetLabel("[cyan]> [white]").
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetPlaceholder("Ask a question...")

	p.inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			text := p.inputField.GetText()
			if text != "" && p.onSubmit != nil {
				p.inputField.SetText("")
				p.onSubmit(text)
			}
		}
	})

	// Status bar
	p.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	p.statusBar.SetText("[gray]Ready[white]")

	// Layout: output on top, status in middle, input at bottom
	p.Flex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(p.outputView, 0, 1, false).
		AddItem(p.statusBar, 1, 0, false).
		AddItem(p.inputField, 1, 0, true)

	p.SetBorder(true).
		SetTitle(" AI Assistant ").
		SetBorderColor(tcell.ColorDarkMagenta)

	// Set up key handling
	p.setupKeyHandlers()

	return p
}

// SetAgent connects the panel to an agent using the Listener pattern (k9s style)
func (p *AIPanel) SetAgent(a *agent.Agent) {
	p.agent = a
	// Register as listener (k9s pattern)
	a.SetListener(p)
	// Also start channel listener for backward compatibility
	go p.listenToAgent()
}

// SetAgentWithChannels connects using only channel-based communication
// Use SetAgent for the preferred listener pattern
func (p *AIPanel) SetAgentWithChannels(a *agent.Agent) {
	p.agent = a
	go p.listenToAgent()
}

// SetOnSubmit sets the callback for when user submits a question
func (p *AIPanel) SetOnSubmit(fn func(string)) {
	p.onSubmit = fn
}

// SetOnFocus sets the callback for when panel gains focus
func (p *AIPanel) SetOnFocus(fn func()) {
	p.onFocus = fn
}

// listenToAgent processes messages from the agent
func (p *AIPanel) listenToAgent() {
	if p.agent == nil {
		return
	}

	for msg := range p.agent.Output {
		switch msg.Type {
		case agent.MsgText:
			p.app.QueueUpdateDraw(func() {
				p.appendText("\n" + msg.Content)
			})

		case agent.MsgStreamChunk:
			p.app.QueueUpdateDraw(func() {
				p.appendText(msg.Content)
			})

		case agent.MsgStreamEnd:
			p.app.QueueUpdateDraw(func() {
				p.appendText("\n")
				p.setStatus("Ready")
			})

		case agent.MsgError:
			p.app.QueueUpdateDraw(func() {
				p.appendText(fmt.Sprintf("\n[red]Error: %s[white]\n", msg.Content))
				p.setStatus("Error")
			})

		case agent.MsgUserChoiceRequest:
			p.app.QueueUpdateDraw(func() {
				p.showApprovalUI(msg.Choice)
			})

		case agent.MsgToolCallRequest:
			p.app.QueueUpdateDraw(func() {
				p.showToolCallUI(msg.ToolCall)
			})

		case agent.MsgToolCallResponse:
			p.app.QueueUpdateDraw(func() {
				p.showToolResultUI(msg.ToolCall)
			})

		case agent.MsgStateChange:
			p.app.QueueUpdateDraw(func() {
				p.updateStatusFromState(msg.Content)
			})
		}
	}
}

// setupKeyHandlers configures key event handling
func (p *AIPanel) setupKeyHandlers() {
	p.outputView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle approval keys
		if p.handleApprovalKey(event) {
			return nil
		}

		// Handle arrow key scrolling
		row, col := p.outputView.GetScrollOffset()
		switch event.Key() {
		case tcell.KeyUp:
			if row > 0 {
				p.outputView.ScrollTo(row-1, col)
			}
			return nil
		case tcell.KeyDown:
			p.outputView.ScrollTo(row+1, col)
			return nil
		case tcell.KeyPgUp:
			_, _, _, height := p.outputView.GetInnerRect()
			if row > height {
				p.outputView.ScrollTo(row-height, col)
			} else {
				p.outputView.ScrollTo(0, col)
			}
			return nil
		case tcell.KeyPgDn:
			_, _, _, height := p.outputView.GetInnerRect()
			p.outputView.ScrollTo(row+height, col)
			return nil
		case tcell.KeyHome:
			p.outputView.ScrollTo(0, 0)
			return nil
		case tcell.KeyEnd:
			p.outputView.ScrollToEnd()
			return nil
		case tcell.KeyTab:
			// Tab to switch to input
			p.app.SetFocus(p.inputField)
			return nil
		}

		// j/k vim-style scrolling
		switch event.Rune() {
		case 'j':
			p.outputView.ScrollTo(row+1, col)
			return nil
		case 'k':
			if row > 0 {
				p.outputView.ScrollTo(row-1, col)
			}
			return nil
		case 'g':
			p.outputView.ScrollTo(0, 0)
			return nil
		case 'G':
			p.outputView.ScrollToEnd()
			return nil
		}

		return event
	})
}

// handleApprovalKey processes key events for approval
func (p *AIPanel) handleApprovalKey(event *tcell.EventKey) bool {
	p.mu.Lock()
	isApproval := p.isShowingApproval
	p.mu.Unlock()

	if !isApproval || p.agent == nil {
		return false
	}

	switch event.Key() {
	case tcell.KeyEnter:
		p.sendApproval(true)
		return true
	case tcell.KeyEscape:
		p.sendApproval(false)
		return true
	}

	switch event.Rune() {
	case 'Y', 'y':
		p.sendApproval(true)
		return true
	case 'N', 'n':
		p.sendApproval(false)
		return true
	}

	return false
}

// sendApproval sends approval response to agent
func (p *AIPanel) sendApproval(approved bool) {
	p.mu.Lock()
	p.isShowingApproval = false
	p.currentApproval = nil
	callback := p.approvalCallback
	p.approvalCallback = nil
	p.mu.Unlock()

	// Use callback if set (synchronous approval handler pattern)
	if callback != nil {
		callback(approved)
	} else if p.agent != nil {
		// Fall back to channel-based approval
		p.agent.SendApproval(approved)
	}

	status := "[red]Cancelled[white]"
	if approved {
		status = "[green]Approved[white]"
	}
	p.appendText(fmt.Sprintf("\n%s\n", status))
	if approved {
		p.setStatus("Executing...")
	} else {
		p.setStatus("Ready")
	}
}

// showApprovalUI displays the approval dialog
func (p *AIPanel) showApprovalUI(choice *agent.ChoiceRequest) {
	p.mu.Lock()
	p.isShowingApproval = true
	p.currentApproval = choice
	p.mu.Unlock()

	var sb strings.Builder
	sb.WriteString("\n\n")
	sb.WriteString("[yellow::b]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[white::-]\n")
	sb.WriteString(fmt.Sprintf("[yellow::b]  %s  [white::-]\n", choice.Title))
	sb.WriteString("[yellow::b]━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━[white::-]\n\n")
	sb.WriteString(fmt.Sprintf("[cyan]%s[white]\n\n", choice.Command))
	sb.WriteString("[gray]Press [green]Y[gray] or [green]Enter[gray] to approve, ")
	sb.WriteString("[red]N[gray] or [red]Esc[gray] to cancel[white]\n")

	p.appendText(sb.String())
	p.setStatus("Waiting for approval...")

	// Focus on output view to capture keys
	p.app.SetFocus(p.outputView)
}

// showToolCallUI displays tool call information
func (p *AIPanel) showToolCallUI(tc *agent.ToolCallInfo) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n[blue]Tool: %s[white]\n", tc.Name))
	sb.WriteString(fmt.Sprintf("[gray]Command: [cyan]%s[white]\n", tc.Command))

	if len(tc.Warnings) > 0 {
		for _, w := range tc.Warnings {
			sb.WriteString(fmt.Sprintf("[yellow]Warning: %s[white]\n", w))
		}
	}

	p.appendText(sb.String())
}

// showToolResultUI displays tool execution result
func (p *AIPanel) showToolResultUI(tc *agent.ToolCallInfo) {
	var sb strings.Builder
	sb.WriteString("[gray]Result:[white]\n")

	// Truncate long results
	result := tc.Result
	if len(result) > 500 {
		result = result[:500] + "\n... (truncated)"
	}
	sb.WriteString(fmt.Sprintf("[white]%s[white]\n", result))

	p.appendText(sb.String())
}

// appendText appends text to the output view
func (p *AIPanel) appendText(text string) {
	fmt.Fprint(p.outputView, text)
	p.outputView.ScrollToEnd()
}

// setStatus updates the status bar
func (p *AIPanel) setStatus(status string) {
	p.statusBar.SetText(fmt.Sprintf("[gray]%s[white]", status))
}

// updateStatusFromState updates status based on agent state
func (p *AIPanel) updateStatusFromState(state string) {
	switch state {
	case "idle":
		p.setStatus("Ready")
	case "running":
		p.setStatus("Thinking...")
	case "analyzing":
		p.setStatus("Analyzing...")
	case "waiting":
		p.setStatus("Waiting for approval...")
	case "done":
		p.setStatus("Ready")
	case "error":
		p.setStatus("Error")
	default:
		p.setStatus(state)
	}
}

// Clear clears the output view
func (p *AIPanel) Clear() {
	p.outputView.Clear()
	p.setStatus("Ready")
}

// SetText sets the output view text
func (p *AIPanel) SetText(text string) {
	p.outputView.SetText(text)
}

// GetOutputView returns the output text view for direct manipulation
func (p *AIPanel) GetOutputView() *tview.TextView {
	return p.outputView
}

// GetInputField returns the input field for direct manipulation
func (p *AIPanel) GetInputField() *tview.InputField {
	return p.inputField
}

// Focus sets focus to the input field
func (p *AIPanel) Focus(delegate func(p tview.Primitive)) {
	delegate(p.inputField)
	if p.onFocus != nil {
		p.onFocus()
	}
}

// ShowThinking displays a thinking indicator
func (p *AIPanel) ShowThinking() {
	p.appendText("\n[gray]Thinking...[white]")
	p.setStatus("Thinking...")
}

// ShowError displays an error message
func (p *AIPanel) ShowError(err error) {
	p.appendText(fmt.Sprintf("\n[red]Error: %v[white]\n", err))
	p.setStatus("Error")
}

// ShowMessage displays a message
func (p *AIPanel) ShowMessage(format string, args ...interface{}) {
	p.appendText(fmt.Sprintf("\n"+format+"\n", args...))
}

// StreamChunk appends a streaming chunk (no newline)
func (p *AIPanel) StreamChunk(chunk string) {
	p.appendText(chunk)
}

// EndStream marks the end of streaming
func (p *AIPanel) EndStream() {
	p.appendText("\n")
	p.setStatus("Ready")
}

// ShowDecisionRequired displays a decision prompt (for backward compatibility)
func (p *AIPanel) ShowDecisionRequired(title, command string, isDangerous bool, warnings []string) {
	choice := &agent.ChoiceRequest{
		ID:          fmt.Sprintf("decision-%d", time.Now().UnixNano()),
		Title:       title,
		Description: command,
		Command:     command,
	}

	p.mu.Lock()
	p.isShowingApproval = true
	p.currentApproval = choice
	p.mu.Unlock()

	var sb strings.Builder
	sb.WriteString("\n\n")

	if isDangerous {
		sb.WriteString("[red::b]━━━ DANGEROUS COMMAND ━━━[white::-]\n\n")
	} else {
		sb.WriteString("[yellow::b]━━━ DECISION REQUIRED ━━━[white::-]\n\n")
	}

	sb.WriteString(fmt.Sprintf("[cyan]%s[white]\n\n", command))

	for _, w := range warnings {
		sb.WriteString(fmt.Sprintf("[yellow]⚠ %s[white]\n", w))
	}

	sb.WriteString("\n[gray]Press [green]Y[gray]/[green]Enter[gray] to approve, ")
	sb.WriteString("[red]N[gray]/[red]Esc[gray] to cancel[white]")

	p.appendText(sb.String())
	p.setStatus("Waiting for approval...")
}

// SetApprovalCallback sets a callback for when approval is given (for backward compatibility)
func (p *AIPanel) SetApprovalCallback(callback func(bool)) {
	p.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		p.mu.Lock()
		isApproval := p.isShowingApproval
		p.mu.Unlock()

		if !isApproval {
			return event
		}

		switch event.Key() {
		case tcell.KeyEnter:
			p.mu.Lock()
			p.isShowingApproval = false
			p.mu.Unlock()
			callback(true)
			return nil
		case tcell.KeyEscape:
			p.mu.Lock()
			p.isShowingApproval = false
			p.mu.Unlock()
			callback(false)
			return nil
		}

		switch event.Rune() {
		case 'Y', 'y':
			p.mu.Lock()
			p.isShowingApproval = false
			p.mu.Unlock()
			callback(true)
			return nil
		case 'N', 'n':
			p.mu.Lock()
			p.isShowingApproval = false
			p.mu.Unlock()
			callback(false)
			return nil
		}

		return event
	})
}

// ============================================================================
// AgentListener Interface Implementation (k9s pattern)
// ============================================================================

// AgentTextReceived handles text events from the agent
func (p *AIPanel) AgentTextReceived(text string) {
	p.app.QueueUpdateDraw(func() {
		p.appendText("\n" + text)
	})
}

// AgentStreamChunk handles streaming chunks from the agent
func (p *AIPanel) AgentStreamChunk(chunk string) {
	p.app.QueueUpdateDraw(func() {
		p.appendText(chunk)
	})
}

// AgentStreamEnd handles stream end events
func (p *AIPanel) AgentStreamEnd() {
	p.app.QueueUpdateDraw(func() {
		p.appendText("\n")
		p.setStatus("Ready")
	})
}

// AgentError handles error events from the agent
func (p *AIPanel) AgentError(err error) {
	p.app.QueueUpdateDraw(func() {
		p.appendText(fmt.Sprintf("\n[red]Error: %v[white]\n", err))
		p.setStatus("Error")
	})
}

// AgentStateChanged handles state change events
func (p *AIPanel) AgentStateChanged(state agent.State) {
	p.app.QueueUpdateDraw(func() {
		p.updateStatusFromState(state.String())
	})
}

// AgentToolCallRequested handles tool call request events
func (p *AIPanel) AgentToolCallRequested(tc *agent.ToolCallInfo) {
	p.app.QueueUpdateDraw(func() {
		p.showToolCallUI(tc)
	})
}

// AgentToolCallCompleted handles tool call completion events
func (p *AIPanel) AgentToolCallCompleted(tc *agent.ToolCallInfo) {
	p.app.QueueUpdateDraw(func() {
		p.showToolResultUI(tc)
	})
}

// AgentApprovalRequested handles approval request events
func (p *AIPanel) AgentApprovalRequested(choice *agent.ChoiceRequest) {
	p.app.QueueUpdateDraw(func() {
		p.showApprovalUI(choice)
	})
}

// AgentApprovalTimeout handles approval timeout events
func (p *AIPanel) AgentApprovalTimeout(choiceID string) {
	p.app.QueueUpdateDraw(func() {
		p.mu.Lock()
		p.isShowingApproval = false
		p.currentApproval = nil
		p.mu.Unlock()
		p.appendText("\n[yellow]Approval timeout - command cancelled[white]\n")
		p.setStatus("Ready")
	})
}

// ============================================================================
// AgentApprovalHandler Interface Implementation
// ============================================================================

// RequestApproval handles synchronous approval requests (k9s pattern)
// This is called when the agent uses the approval handler instead of channels
func (p *AIPanel) RequestApproval(choice *agent.ChoiceRequest, callback func(bool)) {
	p.mu.Lock()
	p.approvalCallback = callback
	p.mu.Unlock()

	p.app.QueueUpdateDraw(func() {
		p.showApprovalUI(choice)
	})
}

// Ensure AIPanel implements AgentApprovalHandler
var _ agent.AgentApprovalHandler = (*AIPanel)(nil)
