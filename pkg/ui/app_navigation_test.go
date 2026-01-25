package ui

import (
	"sync"
	"testing"
)

func TestNavHistoryStack(t *testing.T) {
	// Test that navigation stack works correctly with App struct
	app := &App{
		navigationStack: nil,
		navMx:           sync.Mutex{},
	}

	// Push some items (simulating drillDown behavior)
	app.navMx.Lock()
	app.navigationStack = append(app.navigationStack, navHistory{"pods", "default", ""})
	app.navigationStack = append(app.navigationStack, navHistory{"deployments", "kube-system", "nginx"})
	app.navMx.Unlock()

	app.navMx.Lock()
	if len(app.navigationStack) != 2 {
		t.Errorf("Expected 2 items in stack, got %d", len(app.navigationStack))
	}
	app.navMx.Unlock()

	// Pop last item (simulating goBack behavior)
	app.navMx.Lock()
	prev := app.navigationStack[len(app.navigationStack)-1]
	app.navigationStack = app.navigationStack[:len(app.navigationStack)-1]
	app.navMx.Unlock()

	if prev.resource != "deployments" {
		t.Errorf("Expected resource 'deployments', got %q", prev.resource)
	}
	if prev.namespace != "kube-system" {
		t.Errorf("Expected namespace 'kube-system', got %q", prev.namespace)
	}
	if prev.filter != "nginx" {
		t.Errorf("Expected filter 'nginx', got %q", prev.filter)
	}

	app.navMx.Lock()
	if len(app.navigationStack) != 1 {
		t.Errorf("Expected 1 item in stack after pop, got %d", len(app.navigationStack))
	}
	app.navMx.Unlock()
}

func TestNavHistoryStruct(t *testing.T) {
	nav := navHistory{
		resource:  "pods",
		namespace: "default",
		filter:    "nginx",
	}

	if nav.resource != "pods" {
		t.Errorf("Expected resource 'pods', got %q", nav.resource)
	}
	if nav.namespace != "default" {
		t.Errorf("Expected namespace 'default', got %q", nav.namespace)
	}
	if nav.filter != "nginx" {
		t.Errorf("Expected filter 'nginx', got %q", nav.filter)
	}
}

func TestNavStackConcurrency(t *testing.T) {
	// Test that navigation stack is thread-safe
	app := &App{
		navigationStack: nil,
		navMx:           sync.Mutex{},
	}

	var wg sync.WaitGroup
	numGoroutines := 100

	// Simulate concurrent push operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			app.navMx.Lock()
			app.navigationStack = append(app.navigationStack, navHistory{
				resource:  "pods",
				namespace: "default",
				filter:    "",
			})
			app.navMx.Unlock()
		}(i)
	}

	wg.Wait()

	app.navMx.Lock()
	finalLen := len(app.navigationStack)
	app.navMx.Unlock()

	if finalLen != numGoroutines {
		t.Errorf("Expected %d items in stack, got %d", numGoroutines, finalLen)
	}
}
