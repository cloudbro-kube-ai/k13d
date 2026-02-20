package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestHealingStoreAddRule(t *testing.T) {
	store := NewHealingStore()

	rule := HealingRule{
		Name:    "restart-crashloop",
		Enabled: true,
		Condition: HealingCondition{
			Type:      "crashloop",
			Threshold: 5,
			Duration:  "5m",
		},
		Action: HealingAction{
			Type: "restart",
		},
		Cooldown:   "10m",
		MaxRetries: 3,
	}

	created, err := store.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule() error = %v", err)
	}
	if created.ID == "" {
		t.Error("AddRule() did not assign an ID")
	}
	if created.Name != "restart-crashloop" {
		t.Errorf("AddRule() name = %s, want restart-crashloop", created.Name)
	}

	rules := store.GetRules()
	if len(rules) != 1 {
		t.Fatalf("GetRules() count = %d, want 1", len(rules))
	}
	if rules[0].ID != created.ID {
		t.Errorf("GetRules()[0].ID = %s, want %s", rules[0].ID, created.ID)
	}
}

func TestHealingStoreAddRuleEmptyName(t *testing.T) {
	store := NewHealingStore()

	_, err := store.AddRule(HealingRule{})
	if err == nil {
		t.Error("AddRule() with empty name should return error")
	}
}

func TestHealingStoreUpdateRule(t *testing.T) {
	store := NewHealingStore()

	created, _ := store.AddRule(HealingRule{
		Name:    "test-rule",
		Enabled: true,
		Condition: HealingCondition{
			Type: "crashloop",
		},
		Action: HealingAction{
			Type: "restart",
		},
	})

	updated := HealingRule{
		Name:    "test-rule-updated",
		Enabled: false,
		Condition: HealingCondition{
			Type:      "oom",
			Threshold: 3,
		},
		Action: HealingAction{
			Type: "delete_pod",
		},
		Cooldown: "15m",
	}

	err := store.UpdateRule(created.ID, updated)
	if err != nil {
		t.Fatalf("UpdateRule() error = %v", err)
	}

	rules := store.GetRules()
	if rules[0].Name != "test-rule-updated" {
		t.Errorf("UpdateRule() name = %s, want test-rule-updated", rules[0].Name)
	}
	if rules[0].Enabled {
		t.Error("UpdateRule() enabled should be false")
	}
	if rules[0].Condition.Type != "oom" {
		t.Errorf("UpdateRule() condition type = %s, want oom", rules[0].Condition.Type)
	}
}

func TestHealingStoreUpdateRuleNotFound(t *testing.T) {
	store := NewHealingStore()

	err := store.UpdateRule("nonexistent", HealingRule{Name: "x"})
	if err == nil {
		t.Error("UpdateRule() with nonexistent ID should return error")
	}
}

func TestHealingStoreDeleteRule(t *testing.T) {
	store := NewHealingStore()

	r1, _ := store.AddRule(HealingRule{Name: "rule-1", Action: HealingAction{Type: "restart"}})
	if _, err := store.AddRule(HealingRule{Name: "rule-2", Action: HealingAction{Type: "notify"}}); err != nil {
		t.Fatalf("AddRule() error = %v", err)
	}

	err := store.DeleteRule(r1.ID)
	if err != nil {
		t.Fatalf("DeleteRule() error = %v", err)
	}

	rules := store.GetRules()
	if len(rules) != 1 {
		t.Fatalf("GetRules() count after delete = %d, want 1", len(rules))
	}
	if rules[0].Name != "rule-2" {
		t.Errorf("remaining rule name = %s, want rule-2", rules[0].Name)
	}
}

func TestHealingStoreDeleteRuleNotFound(t *testing.T) {
	store := NewHealingStore()

	err := store.DeleteRule("nonexistent")
	if err == nil {
		t.Error("DeleteRule() with nonexistent ID should return error")
	}
}

func TestHealingStoreRecordEvent(t *testing.T) {
	store := NewHealingStore()

	store.RecordEvent(HealingEvent{
		RuleName:  "restart-crashloop",
		Resource:  "pod/nginx-123",
		Namespace: "default",
		Action:    "restart",
		Result:    "success",
	})

	events := store.GetEvents(10)
	if len(events) != 1 {
		t.Fatalf("GetEvents() count = %d, want 1", len(events))
	}
	if events[0].RuleName != "restart-crashloop" {
		t.Errorf("event RuleName = %s, want restart-crashloop", events[0].RuleName)
	}
	if events[0].Timestamp == "" {
		t.Error("event Timestamp should be auto-populated")
	}
	if events[0].Result != "success" {
		t.Errorf("event Result = %s, want success", events[0].Result)
	}
}

func TestHealingStoreGetEventsLimit(t *testing.T) {
	store := NewHealingStore()

	for i := 0; i < 10; i++ {
		store.RecordEvent(HealingEvent{
			RuleName:  "rule",
			Resource:  "pod/test",
			Namespace: "default",
			Action:    "restart",
			Result:    "success",
		})
	}

	events := store.GetEvents(5)
	if len(events) != 5 {
		t.Errorf("GetEvents(5) count = %d, want 5", len(events))
	}

	// Limit larger than available should return all
	events = store.GetEvents(20)
	if len(events) != 10 {
		t.Errorf("GetEvents(20) count = %d, want 10", len(events))
	}

	// Zero limit should return all
	events = store.GetEvents(0)
	if len(events) != 10 {
		t.Errorf("GetEvents(0) count = %d, want 10", len(events))
	}
}

func TestHealingStoreConcurrentAccess(t *testing.T) {
	store := NewHealingStore()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = store.AddRule(HealingRule{
				Name:   "concurrent-rule",
				Action: HealingAction{Type: "restart"},
			})
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.GetRules()
		}()
	}

	// Concurrent events
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.RecordEvent(HealingEvent{
				RuleName: "test",
				Resource: "pod/x",
				Action:   "restart",
				Result:   "success",
			})
		}()
	}

	wg.Wait()

	rules := store.GetRules()
	if len(rules) != 50 {
		t.Errorf("after concurrent writes, rules count = %d, want 50", len(rules))
	}

	events := store.GetEvents(0)
	if len(events) != 50 {
		t.Errorf("after concurrent writes, events count = %d, want 50", len(events))
	}
}

func TestHealingStoreNamespaceFiltering(t *testing.T) {
	store := NewHealingStore()

	if _, err := store.AddRule(HealingRule{
		Name:       "ns-rule",
		Enabled:    true,
		Namespaces: []string{"production", "staging"},
		Condition:  HealingCondition{Type: "crashloop"},
		Action:     HealingAction{Type: "restart"},
	}); err != nil {
		t.Fatalf("AddRule() error = %v", err)
	}

	rules := store.GetRules()
	if len(rules[0].Namespaces) != 2 {
		t.Errorf("Namespaces count = %d, want 2", len(rules[0].Namespaces))
	}
	if rules[0].Namespaces[0] != "production" {
		t.Errorf("Namespaces[0] = %s, want production", rules[0].Namespaces[0])
	}
}

func TestHandleHealingRulesGET(t *testing.T) {
	store := NewHealingStore()
	if _, err := store.AddRule(HealingRule{
		Name:      "test-rule",
		Enabled:   true,
		Condition: HealingCondition{Type: "crashloop"},
		Action:    HealingAction{Type: "restart"},
	}); err != nil {
		t.Fatalf("AddRule() error = %v", err)
	}

	s := &Server{healingStore: store}

	req := httptest.NewRequest(http.MethodGet, "/api/healing/rules", nil)
	w := httptest.NewRecorder()

	s.handleHealingRules(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/healing/rules status = %d, want 200", w.Code)
	}

	var resp map[string][]HealingRule
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp["rules"]) != 1 {
		t.Errorf("rules count = %d, want 1", len(resp["rules"]))
	}
}

func TestHandleHealingRulesPOST(t *testing.T) {
	store := NewHealingStore()
	s := &Server{healingStore: store}

	body := `{"name":"new-rule","enabled":true,"condition":{"type":"oom"},"action":{"type":"delete_pod"},"cooldown":"5m","maxRetries":2}`
	req := httptest.NewRequest(http.MethodPost, "/api/healing/rules", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleHealingRules(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST /api/healing/rules status = %d, want 201", w.Code)
	}

	var created HealingRule
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.ID == "" {
		t.Error("created rule should have an ID")
	}
	if created.Name != "new-rule" {
		t.Errorf("created rule name = %s, want new-rule", created.Name)
	}
}

func TestHandleHealingRulesPUT(t *testing.T) {
	store := NewHealingStore()
	created, _ := store.AddRule(HealingRule{
		Name:      "old-rule",
		Condition: HealingCondition{Type: "crashloop"},
		Action:    HealingAction{Type: "restart"},
	})

	s := &Server{healingStore: store}

	body := `{"name":"updated-rule","enabled":false,"condition":{"type":"pending"},"action":{"type":"notify"}}`
	req := httptest.NewRequest(http.MethodPut, "/api/healing/rules?id="+created.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleHealingRules(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT /api/healing/rules status = %d, want 200", w.Code)
	}

	rules := store.GetRules()
	if rules[0].Name != "updated-rule" {
		t.Errorf("updated rule name = %s, want updated-rule", rules[0].Name)
	}
}

func TestHandleHealingRulesDELETE(t *testing.T) {
	store := NewHealingStore()
	created, _ := store.AddRule(HealingRule{
		Name:      "to-delete",
		Condition: HealingCondition{Type: "crashloop"},
		Action:    HealingAction{Type: "restart"},
	})

	s := &Server{healingStore: store}

	req := httptest.NewRequest(http.MethodDelete, "/api/healing/rules?id="+created.ID, nil)
	w := httptest.NewRecorder()

	s.handleHealingRules(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE /api/healing/rules status = %d, want 204", w.Code)
	}

	rules := store.GetRules()
	if len(rules) != 0 {
		t.Errorf("rules count after delete = %d, want 0", len(rules))
	}
}

func TestHandleHealingRulesMethodNotAllowed(t *testing.T) {
	s := &Server{healingStore: NewHealingStore()}

	req := httptest.NewRequest(http.MethodPatch, "/api/healing/rules", nil)
	w := httptest.NewRecorder()

	s.handleHealingRules(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("PATCH /api/healing/rules status = %d, want 405", w.Code)
	}
}

func TestHandleHealingEventsGET(t *testing.T) {
	store := NewHealingStore()
	store.RecordEvent(HealingEvent{
		RuleName:  "test-rule",
		Resource:  "pod/nginx",
		Namespace: "default",
		Action:    "restart",
		Result:    "success",
	})

	s := &Server{healingStore: store}

	req := httptest.NewRequest(http.MethodGet, "/api/healing/events", nil)
	w := httptest.NewRecorder()

	s.handleHealingEvents(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/healing/events status = %d, want 200", w.Code)
	}

	var resp map[string][]HealingEvent
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp["events"]) != 1 {
		t.Errorf("events count = %d, want 1", len(resp["events"]))
	}
}

func TestHandleHealingEventsWithLimit(t *testing.T) {
	store := NewHealingStore()
	for i := 0; i < 10; i++ {
		store.RecordEvent(HealingEvent{
			RuleName: "rule",
			Resource: "pod/test",
			Action:   "restart",
			Result:   "success",
		})
	}

	s := &Server{healingStore: store}

	req := httptest.NewRequest(http.MethodGet, "/api/healing/events?limit=3", nil)
	w := httptest.NewRecorder()

	s.handleHealingEvents(w, req)

	var resp map[string][]HealingEvent
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp["events"]) != 3 {
		t.Errorf("events count with limit=3: %d, want 3", len(resp["events"]))
	}
}

func TestHandleHealingEventsMethodNotAllowed(t *testing.T) {
	s := &Server{healingStore: NewHealingStore()}

	req := httptest.NewRequest(http.MethodPost, "/api/healing/events", nil)
	w := httptest.NewRecorder()

	s.handleHealingEvents(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/healing/events status = %d, want 405", w.Code)
	}
}

func TestHealingRuleIDsAreUnique(t *testing.T) {
	store := NewHealingStore()
	ids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		created, _ := store.AddRule(HealingRule{
			Name:   "rule",
			Action: HealingAction{Type: "restart"},
		})
		if ids[created.ID] {
			t.Fatalf("duplicate ID generated: %s", created.ID)
		}
		ids[created.ID] = true
	}
}
