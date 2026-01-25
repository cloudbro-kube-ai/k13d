package agent

import "time"

// MessageType identifies the kind of message
type MessageType int

const (
	// MsgText - Regular text content from LLM
	MsgText MessageType = iota
	// MsgError - Error message
	MsgError
	// MsgToolCallRequest - Agent wants to execute a tool
	MsgToolCallRequest
	// MsgToolCallResponse - Tool execution result
	MsgToolCallResponse
	// MsgUserInputRequest - Agent needs text input from user
	MsgUserInputRequest
	// MsgUserChoiceRequest - Agent needs approval (Y/N)
	MsgUserChoiceRequest
	// MsgUserChoiceResponse - User's approval decision
	MsgUserChoiceResponse
	// MsgStateChange - Agent state changed
	MsgStateChange
	// MsgStreamChunk - Streaming text chunk
	MsgStreamChunk
	// MsgStreamEnd - End of streaming
	MsgStreamEnd
)

// String returns the string representation of the message type
func (t MessageType) String() string {
	switch t {
	case MsgText:
		return "text"
	case MsgError:
		return "error"
	case MsgToolCallRequest:
		return "tool_call_request"
	case MsgToolCallResponse:
		return "tool_call_response"
	case MsgUserInputRequest:
		return "user_input_request"
	case MsgUserChoiceRequest:
		return "user_choice_request"
	case MsgUserChoiceResponse:
		return "user_choice_response"
	case MsgStateChange:
		return "state_change"
	case MsgStreamChunk:
		return "stream_chunk"
	case MsgStreamEnd:
		return "stream_end"
	default:
		return "unknown"
	}
}

// Message is the communication unit between UI and Agent
type Message struct {
	Type        MessageType
	Content     string
	ToolCall    *ToolCallInfo
	Choice      *ChoiceRequest
	Error       error
	IsStreaming bool
	Timestamp   time.Time
}

// NewTextMessage creates a new text message
func NewTextMessage(content string) *Message {
	return &Message{
		Type:      MsgText,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// NewStreamChunk creates a new streaming chunk message
func NewStreamChunk(content string) *Message {
	return &Message{
		Type:        MsgStreamChunk,
		Content:     content,
		IsStreaming: true,
		Timestamp:   time.Now(),
	}
}

// NewStreamEnd creates a stream end message
func NewStreamEnd() *Message {
	return &Message{
		Type:      MsgStreamEnd,
		Timestamp: time.Now(),
	}
}

// NewErrorMessage creates a new error message
func NewErrorMessage(err error) *Message {
	return &Message{
		Type:      MsgError,
		Content:   err.Error(),
		Error:     err,
		Timestamp: time.Now(),
	}
}

// NewStateChangeMessage creates a new state change message
func NewStateChangeMessage(state State) *Message {
	return &Message{
		Type:      MsgStateChange,
		Content:   state.String(),
		Timestamp: time.Now(),
	}
}

// NewToolCallRequestMessage creates a tool call request message
func NewToolCallRequestMessage(toolCall *ToolCallInfo) *Message {
	return &Message{
		Type:      MsgToolCallRequest,
		ToolCall:  toolCall,
		Content:   toolCall.Command,
		Timestamp: time.Now(),
	}
}

// NewChoiceRequestMessage creates a choice request message
func NewChoiceRequestMessage(choice *ChoiceRequest) *Message {
	return &Message{
		Type:      MsgUserChoiceRequest,
		Choice:    choice,
		Content:   choice.Description,
		Timestamp: time.Now(),
	}
}

// NewChoiceResponseMessage creates a choice response message
func NewChoiceResponseMessage(approved bool) *Message {
	content := "rejected"
	if approved {
		content = "approved"
	}
	return &Message{
		Type:      MsgUserChoiceResponse,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// ToolCallInfo contains tool execution details
type ToolCallInfo struct {
	ID          string
	Name        string
	Command     string
	Args        map[string]interface{}
	IsDangerous bool
	IsReadOnly  bool
	Warnings    []string
	Result      string
	Error       error
	Approved    bool
	ExecutedAt  time.Time
}

// ChoiceRequest represents a decision needed from user
type ChoiceRequest struct {
	ID          string
	Title       string
	Description string
	Command     string
	Options     []ChoiceOption
	Timeout     time.Duration
	CreatedAt   time.Time
}

// NewApprovalRequest creates a standard approval request for tool execution
func NewApprovalRequest(id, command string, isDangerous bool) *ChoiceRequest {
	title := "Command Approval Required"
	if isDangerous {
		title = "DANGEROUS Command - Approval Required"
	}

	return &ChoiceRequest{
		ID:          id,
		Title:       title,
		Description: command,
		Command:     command,
		Options: []ChoiceOption{
			{ID: "approve", Label: "Execute", Key: 'Y'},
			{ID: "reject", Label: "Cancel", Key: 'N'},
		},
		Timeout:   30 * time.Second,
		CreatedAt: time.Now(),
	}
}

// ChoiceOption represents one option in a choice
type ChoiceOption struct {
	ID    string
	Label string
	Key   rune // Keyboard shortcut (e.g., 'Y', 'N')
}
