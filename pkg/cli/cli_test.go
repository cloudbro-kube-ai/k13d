package cli

import (
	"testing"
)

func TestHistory(t *testing.T) {
	h := NewCommandHistory()

	// Test empty history
	if _, ok := h.Previous(); ok {
		t.Error("Expected false from empty history Previous")
	}
	if _, ok := h.Next(); ok {
		t.Error("Expected false from empty history Next")
	}

	// Add commands
	h.Add("kubectl get pods")
	h.Add("kubectl get svc")
	h.Add("kubectl get nodes")

	// Test navigation with PreviousWithIdx
	idx := h.Len()
	entry, ok := h.PreviousWithIdx(&idx)
	if !ok || entry != "kubectl get nodes" {
		t.Errorf("Expected 'kubectl get nodes', got %q (ok=%v)", entry, ok)
	}
	entry, ok = h.PreviousWithIdx(&idx)
	if !ok || entry != "kubectl get svc" {
		t.Errorf("Expected 'kubectl get svc', got %q (ok=%v)", entry, ok)
	}
	entry, ok = h.PreviousWithIdx(&idx)
	if !ok || entry != "kubectl get pods" {
		t.Errorf("Expected 'kubectl get pods', got %q (ok=%v)", entry, ok)
	}
	// Stay at first
	entry, ok = h.PreviousWithIdx(&idx)
	if ok {
		t.Errorf("Expected false at start, got %q", entry)
	}

	// Test Next navigation
	entry, ok = h.NextWithIdx(&idx)
	if !ok || entry != "kubectl get svc" {
		t.Errorf("Expected 'kubectl get svc', got %q (ok=%v)", entry, ok)
	}
	entry, ok = h.NextWithIdx(&idx)
	if !ok || entry != "kubectl get nodes" {
		t.Errorf("Expected 'kubectl get nodes', got %q (ok=%v)", entry, ok)
	}
	// Past end returns empty
	entry, ok = h.NextWithIdx(&idx)
	if ok {
		t.Errorf("Expected false past end, got %q", entry)
	}

	// Test duplicate prevention
	h.Add("kubectl get nodes") // Should not add duplicate
	if h.Len() != 3 {
		t.Errorf("Expected 3 commands after duplicate add, got %d", h.Len())
	}

	// Test empty command not added
	h.Add("")
	if h.Len() != 3 {
		t.Errorf("Expected 3 commands after empty add, got %d", h.Len())
	}
}

func TestHistoryNavigationSequence(t *testing.T) {
	h := NewCommandHistory()
	h.Add("first")
	h.Add("second")
	h.Add("third")

	idx := h.Len()
	var results []string
	var entry string
	var ok bool

	entry, ok = h.PreviousWithIdx(&idx)
	if ok {
		results = append(results, entry)
	}
	entry, ok = h.PreviousWithIdx(&idx)
	if ok {
		results = append(results, entry)
	}
	entry, ok = h.PreviousWithIdx(&idx)
	if ok {
		results = append(results, entry)
	}
	entry, ok = h.NextWithIdx(&idx)
	if ok {
		results = append(results, entry)
	}
	entry, ok = h.NextWithIdx(&idx)
	if ok {
		results = append(results, entry)
	}

	expected := []string{"third", "second", "first", "second", "third"}

	if len(results) != len(expected) {
		t.Fatalf("Expected %d results, got %d", len(expected), len(results))
	}

	for i, r := range results {
		if r != expected[i] {
			t.Errorf("Step %d: expected %q, got %q", i, expected[i], r)
		}
	}
}
