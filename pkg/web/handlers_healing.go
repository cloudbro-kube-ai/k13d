package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// HealingRule defines an auto-remediation rule.
type HealingRule struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Enabled    bool             `json:"enabled"`
	Condition  HealingCondition `json:"condition"`
	Action     HealingAction    `json:"action"`
	Cooldown   string           `json:"cooldown"`
	MaxRetries int              `json:"maxRetries"`
	Namespaces []string         `json:"namespaces,omitempty"`
}

// HealingCondition describes the trigger condition for a healing rule.
type HealingCondition struct {
	Type      string `json:"type"`                // "crashloop", "oom", "pending", "high_restart"
	Threshold int    `json:"threshold,omitempty"` // e.g., restart count threshold
	Duration  string `json:"duration,omitempty"`  // e.g., "5m" for how long condition must persist
}

// HealingAction describes the remediation action to take.
type HealingAction struct {
	Type       string            `json:"type"` // "restart", "scale_up", "notify", "delete_pod"
	Parameters map[string]string `json:"parameters,omitempty"`
}

// HealingEvent records a healing action taken.
type HealingEvent struct {
	Timestamp string `json:"timestamp"`
	RuleName  string `json:"ruleName"`
	Resource  string `json:"resource"`
	Namespace string `json:"namespace"`
	Action    string `json:"action"`
	Result    string `json:"result"` // "success", "failed", "skipped"
	Details   string `json:"details,omitempty"`
}

// HealingStore provides thread-safe in-memory storage for healing rules and events.
type HealingStore struct {
	mu     sync.RWMutex
	rules  []HealingRule
	events []HealingEvent
	nextID int
}

// NewHealingStore creates a new in-memory healing store.
func NewHealingStore() *HealingStore {
	return &HealingStore{
		rules:  []HealingRule{},
		events: []HealingEvent{},
		nextID: 1,
	}
}

// AddRule adds a new healing rule and assigns an ID.
func (hs *HealingStore) AddRule(rule HealingRule) (HealingRule, error) {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	if rule.Name == "" {
		return HealingRule{}, fmt.Errorf("rule name is required")
	}

	rule.ID = fmt.Sprintf("rule-%d", hs.nextID)
	hs.nextID++
	hs.rules = append(hs.rules, rule)
	return rule, nil
}

// GetRules returns all healing rules.
func (hs *HealingStore) GetRules() []HealingRule {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	result := make([]HealingRule, len(hs.rules))
	copy(result, hs.rules)
	return result
}

// UpdateRule updates an existing healing rule by ID.
func (hs *HealingStore) UpdateRule(id string, rule HealingRule) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	for i, r := range hs.rules {
		if r.ID == id {
			rule.ID = id
			hs.rules[i] = rule
			return nil
		}
	}
	return fmt.Errorf("rule not found: %s", id)
}

// DeleteRule removes a healing rule by ID.
func (hs *HealingStore) DeleteRule(id string) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	for i, r := range hs.rules {
		if r.ID == id {
			hs.rules = append(hs.rules[:i], hs.rules[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("rule not found: %s", id)
}

// RecordEvent records a healing event.
func (hs *HealingStore) RecordEvent(event HealingEvent) {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	if event.Timestamp == "" {
		event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	hs.events = append(hs.events, event)
}

// GetEvents returns the most recent healing events, limited by count.
func (hs *HealingStore) GetEvents(limit int) []HealingEvent {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	if limit <= 0 || limit > len(hs.events) {
		limit = len(hs.events)
	}

	// Return the most recent events (from the end)
	start := len(hs.events) - limit
	result := make([]HealingEvent, limit)
	copy(result, hs.events[start:])
	return result
}

func (s *Server) handleHealingRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		rules := s.healingStore.GetRules()
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"rules": rules,
		})

	case http.MethodPost:
		var rule HealingRule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}
		created, err := s.healingStore.AddRule(rule)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(created)

	case http.MethodPut:
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "Missing rule id query parameter", http.StatusBadRequest)
			return
		}
		var rule HealingRule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.healingStore.UpdateRule(id, rule); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		rule.ID = id
		_ = json.NewEncoder(w).Encode(rule)

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "Missing rule id query parameter", http.StatusBadRequest)
			return
		}
		if err := s.healingStore.DeleteRule(id); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleHealingEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	events := s.healingStore.GetEvents(limit)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
	})
}
