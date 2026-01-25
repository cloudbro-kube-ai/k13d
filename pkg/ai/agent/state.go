// Package agent provides a state-machine based AI agent following kubectl-ai patterns.
package agent

// State represents the current state of the AI agent
type State int

const (
	// StateIdle - Agent is waiting for user input
	StateIdle State = iota
	// StateRunning - Agent is processing/calling LLM
	StateRunning
	// StateToolAnalysis - Agent is analyzing tool calls
	StateToolAnalysis
	// StateWaitingForApproval - Agent is waiting for user approval
	StateWaitingForApproval
	// StateDone - Agent has completed the conversation turn
	StateDone
	// StateError - Agent encountered an error
	StateError
)

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateRunning:
		return "running"
	case StateToolAnalysis:
		return "analyzing"
	case StateWaitingForApproval:
		return "waiting"
	case StateDone:
		return "done"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// IsTerminal returns true if the state is a terminal state
func (s State) IsTerminal() bool {
	return s == StateDone || s == StateError
}

// CanTransitionTo returns true if the state can transition to the target state
func (s State) CanTransitionTo(target State) bool {
	validTransitions := map[State][]State{
		StateIdle:               {StateRunning},
		StateRunning:            {StateToolAnalysis, StateDone, StateError},
		StateToolAnalysis:       {StateWaitingForApproval, StateRunning, StateDone},
		StateWaitingForApproval: {StateRunning, StateDone, StateError},
		StateDone:               {StateIdle},
		StateError:              {StateIdle},
	}

	targets, ok := validTransitions[s]
	if !ok {
		return false
	}

	for _, t := range targets {
		if t == target {
			return true
		}
	}
	return false
}
