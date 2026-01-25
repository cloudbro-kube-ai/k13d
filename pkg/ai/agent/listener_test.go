package agent

import (
	"errors"
	"testing"
)

func TestNullListener(t *testing.T) {
	// NullListener should not panic - all methods should be no-ops
	nl := NullListener{}

	// These should all be no-ops and not panic
	t.Run("AgentTextReceived", func(t *testing.T) {
		nl.AgentTextReceived("test")
	})

	t.Run("AgentStreamChunk", func(t *testing.T) {
		nl.AgentStreamChunk("chunk")
	})

	t.Run("AgentStreamEnd", func(t *testing.T) {
		nl.AgentStreamEnd()
	})

	t.Run("AgentError", func(t *testing.T) {
		nl.AgentError(errors.New("test error"))
	})

	t.Run("AgentStateChanged", func(t *testing.T) {
		nl.AgentStateChanged(StateRunning)
	})

	t.Run("AgentToolCallRequested", func(t *testing.T) {
		nl.AgentToolCallRequested(&ToolCallInfo{})
	})

	t.Run("AgentToolCallCompleted", func(t *testing.T) {
		nl.AgentToolCallCompleted(&ToolCallInfo{})
	})

	t.Run("AgentApprovalRequested", func(t *testing.T) {
		nl.AgentApprovalRequested(&ChoiceRequest{})
	})

	t.Run("AgentApprovalTimeout", func(t *testing.T) {
		nl.AgentApprovalTimeout("test-id")
	})
}

func TestMultiListener(t *testing.T) {
	ml := NewMultiListener()

	// Track calls
	var calls []string
	listener1 := &testListener{
		onText:  func(s string) { calls = append(calls, "l1:text:"+s) },
		onChunk: func(s string) { calls = append(calls, "l1:chunk:"+s) },
		onEnd:   func() { calls = append(calls, "l1:end") },
		onError: func(e error) { calls = append(calls, "l1:error:"+e.Error()) },
		onState: func(s State) { calls = append(calls, "l1:state:"+s.String()) },
	}
	listener2 := &testListener{
		onText: func(s string) { calls = append(calls, "l2:text:"+s) },
	}

	ml.Add(listener1)
	ml.Add(listener2)

	// Test broadcasting
	ml.AgentTextReceived("hello")
	if len(calls) != 2 {
		t.Errorf("Expected 2 calls, got %d", len(calls))
	}
	if calls[0] != "l1:text:hello" {
		t.Errorf("Expected l1:text:hello, got %s", calls[0])
	}
	if calls[1] != "l2:text:hello" {
		t.Errorf("Expected l2:text:hello, got %s", calls[1])
	}

	calls = nil
	ml.AgentStreamChunk("chunk1")
	if len(calls) != 1 {
		t.Errorf("Expected 1 call, got %d", len(calls))
	}

	calls = nil
	ml.AgentStreamEnd()
	if len(calls) != 1 {
		t.Errorf("Expected 1 call, got %d", len(calls))
	}

	calls = nil
	ml.AgentError(errors.New("test"))
	if len(calls) != 1 {
		t.Errorf("Expected 1 call, got %d", len(calls))
	}

	calls = nil
	ml.AgentStateChanged(StateRunning)
	if len(calls) != 1 {
		t.Errorf("Expected 1 call, got %d", len(calls))
	}

	// Test remove
	ml.Remove(listener1)
	calls = nil
	ml.AgentTextReceived("after remove")
	if len(calls) != 1 {
		t.Errorf("Expected 1 call after remove, got %d", len(calls))
	}
	if calls[0] != "l2:text:after remove" {
		t.Errorf("Expected l2:text:after remove, got %s", calls[0])
	}
}

func TestMultiListenerToolEvents(t *testing.T) {
	ml := NewMultiListener()

	var toolCalls []*ToolCallInfo
	var approvals []*ChoiceRequest
	var timeouts []string

	listener := &testListener{
		onToolRequest:  func(tc *ToolCallInfo) { toolCalls = append(toolCalls, tc) },
		onToolComplete: func(tc *ToolCallInfo) { toolCalls = append(toolCalls, tc) },
		onApproval:     func(c *ChoiceRequest) { approvals = append(approvals, c) },
		onTimeout:      func(id string) { timeouts = append(timeouts, id) },
	}

	ml.Add(listener)

	tc := &ToolCallInfo{ID: "tc-1", Name: "kubectl"}
	ml.AgentToolCallRequested(tc)
	ml.AgentToolCallCompleted(tc)

	if len(toolCalls) != 2 {
		t.Errorf("Expected 2 tool calls, got %d", len(toolCalls))
	}

	choice := &ChoiceRequest{ID: "choice-1"}
	ml.AgentApprovalRequested(choice)
	if len(approvals) != 1 {
		t.Errorf("Expected 1 approval, got %d", len(approvals))
	}

	ml.AgentApprovalTimeout("timeout-1")
	if len(timeouts) != 1 {
		t.Errorf("Expected 1 timeout, got %d", len(timeouts))
	}
}

// testListener is a configurable test listener
type testListener struct {
	onText         func(string)
	onChunk        func(string)
	onEnd          func()
	onError        func(error)
	onState        func(State)
	onToolRequest  func(*ToolCallInfo)
	onToolComplete func(*ToolCallInfo)
	onApproval     func(*ChoiceRequest)
	onTimeout      func(string)
}

func (l *testListener) AgentTextReceived(text string) {
	if l.onText != nil {
		l.onText(text)
	}
}

func (l *testListener) AgentStreamChunk(chunk string) {
	if l.onChunk != nil {
		l.onChunk(chunk)
	}
}

func (l *testListener) AgentStreamEnd() {
	if l.onEnd != nil {
		l.onEnd()
	}
}

func (l *testListener) AgentError(err error) {
	if l.onError != nil {
		l.onError(err)
	}
}

func (l *testListener) AgentStateChanged(state State) {
	if l.onState != nil {
		l.onState(state)
	}
}

func (l *testListener) AgentToolCallRequested(tc *ToolCallInfo) {
	if l.onToolRequest != nil {
		l.onToolRequest(tc)
	}
}

func (l *testListener) AgentToolCallCompleted(tc *ToolCallInfo) {
	if l.onToolComplete != nil {
		l.onToolComplete(tc)
	}
}

func (l *testListener) AgentApprovalRequested(choice *ChoiceRequest) {
	if l.onApproval != nil {
		l.onApproval(choice)
	}
}

func (l *testListener) AgentApprovalTimeout(choiceID string) {
	if l.onTimeout != nil {
		l.onTimeout(choiceID)
	}
}
