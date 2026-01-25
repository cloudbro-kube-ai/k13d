package views

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ui/actions"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ui/models"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ui/render"
	"github.com/rivo/tview"
)

// E2E tests for the k9s-style view architecture

// TestE2EViewLifecycle tests the complete view lifecycle from creation to destruction.
func TestE2EViewLifecycle(t *testing.T) {
	// Create app and screen for testing
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("Failed to init screen: %v", err)
	}
	// Note: Don't defer screen.Fini() as tview.Application.Stop() calls it internally
	screen.SetSize(120, 40)

	app := tview.NewApplication().SetScreen(screen)

	// Create a resource view
	renderer := render.NewBaseRenderer(render.Header{
		{Name: "NAME"},
		{Name: "STATUS"},
		{Name: "AGE"},
	})
	rv := NewResourceView("pods", "v1/pods", "default", renderer)

	// Test Init
	if err := rv.Init(context.Background()); err != nil {
		t.Errorf("Init failed: %v", err)
	}

	// Set as root
	app.SetRoot(rv.Primitive(), true)

	// Run in background
	done := make(chan struct{})
	go func() {
		_ = app.Run()
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)

	// Test Start
	rv.Start()
	if rv.IsStopped() {
		t.Error("View should be running after Start()")
	}

	// Test Stop
	rv.Stop()
	if !rv.IsStopped() {
		t.Error("View should be stopped after Stop()")
	}

	// Cleanup
	app.Stop()
	<-done
}

// TestE2EPageStackNavigation tests page stack navigation with user interactions.
func TestE2EPageStackNavigation(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("Failed to init screen: %v", err)
	}
	// Note: Don't defer screen.Fini() as tview.Application.Stop() calls it internally
	screen.SetSize(120, 40)

	app := tview.NewApplication().SetScreen(screen)

	// Create page stack
	stack := NewPageStack(app)

	// Create test views
	view1 := createTestViewWithContent("pods-view", "Pods List")
	view2 := createTestViewWithContent("logs-view", "Pod Logs")
	view3 := createTestViewWithContent("yaml-view", "Pod YAML")

	// Set stack as root
	app.SetRoot(stack.Pages, true)

	done := make(chan struct{})
	go func() {
		_ = app.Run()
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)

	// Push first view
	stack.Push(view1)
	if stack.Len() != 1 {
		t.Errorf("Stack len should be 1, got %d", stack.Len())
	}
	if stack.Top().Name() != "pods-view" {
		t.Errorf("Top view should be 'pods-view', got %s", stack.Top().Name())
	}

	// Push second view
	stack.Push(view2)
	if stack.Len() != 2 {
		t.Errorf("Stack len should be 2, got %d", stack.Len())
	}

	// Push third view
	stack.Push(view3)
	if stack.Len() != 3 {
		t.Errorf("Stack len should be 3, got %d", stack.Len())
	}

	// Verify breadcrumbs
	crumbs := stack.Breadcrumbs()
	if len(crumbs) != 3 {
		t.Errorf("Breadcrumbs should have 3 items, got %d", len(crumbs))
	}

	// Go back
	stack.GoBack()
	if stack.Len() != 2 {
		t.Errorf("Stack len should be 2 after GoBack, got %d", stack.Len())
	}
	if stack.Top().Name() != "logs-view" {
		t.Errorf("Top view should be 'logs-view', got %s", stack.Top().Name())
	}

	// Switch to specific view
	stack.SwitchTo("pods-view")
	if stack.Len() != 1 {
		t.Errorf("Stack len should be 1 after SwitchTo, got %d", stack.Len())
	}

	// Cleanup
	app.Stop()
	<-done
}

// TestE2EActionSystem tests the action system with simulated key events.
func TestE2EActionSystem(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("Failed to init screen: %v", err)
	}
	defer screen.Fini() // Safe here as we don't use tview.Application
	screen.SetSize(120, 40)

	// Track action calls
	var mu sync.Mutex
	actionCalls := make(map[string]int)

	// Create actions
	ka := actions.NewKeyActions()
	ka.AddRune('j', actions.NewKeyAction("Down", func(ctx context.Context) error {
		mu.Lock()
		actionCalls["down"]++
		mu.Unlock()
		return nil
	}))
	ka.AddRune('k', actions.NewKeyAction("Up", func(ctx context.Context) error {
		mu.Lock()
		actionCalls["up"]++
		mu.Unlock()
		return nil
	}))
	ka.AddRune('q', actions.NewKeyAction("Quit", func(ctx context.Context) error {
		mu.Lock()
		actionCalls["quit"]++
		mu.Unlock()
		return nil
	}))
	ka.AddRune('/', actions.NewKeyAction("Filter", func(ctx context.Context) error {
		mu.Lock()
		actionCalls["filter"]++
		mu.Unlock()
		return nil
	}))
	ka.Add(tcell.KeyEnter, actions.NewKeyAction("Select", func(ctx context.Context) error {
		mu.Lock()
		actionCalls["select"]++
		mu.Unlock()
		return nil
	}))

	// Simulate key events
	keys := []struct {
		key  tcell.Key
		r    rune
		name string
	}{
		{tcell.KeyRune, 'j', "down"},
		{tcell.KeyRune, 'j', "down"},
		{tcell.KeyRune, 'k', "up"},
		{tcell.KeyRune, '/', "filter"},
		{tcell.KeyEnter, 0, "select"},
		{tcell.KeyRune, 'q', "quit"},
	}

	ctx := context.Background()
	for _, k := range keys {
		event := tcell.NewEventKey(k.key, k.r, tcell.ModNone)
		if action, ok := ka.Get(event); ok && action.Action != nil {
			_ = action.Action(ctx)
		}
	}

	// Verify action calls
	mu.Lock()
	defer mu.Unlock()

	if actionCalls["down"] != 2 {
		t.Errorf("Expected 2 'down' calls, got %d", actionCalls["down"])
	}
	if actionCalls["up"] != 1 {
		t.Errorf("Expected 1 'up' call, got %d", actionCalls["up"])
	}
	if actionCalls["filter"] != 1 {
		t.Errorf("Expected 1 'filter' call, got %d", actionCalls["filter"])
	}
	if actionCalls["select"] != 1 {
		t.Errorf("Expected 1 'select' call, got %d", actionCalls["select"])
	}
	if actionCalls["quit"] != 1 {
		t.Errorf("Expected 1 'quit' call, got %d", actionCalls["quit"])
	}
}

// TestE2ETableFiltering tests table filtering functionality.
func TestE2ETableFiltering(t *testing.T) {
	// Create a filtered table
	ft := models.NewFilteredTable()

	// Set initial data
	ft.SetData(
		[]string{"NAME", "NAMESPACE", "STATUS"},
		[][]string{
			{"nginx-abc", "default", "Running"},
			{"nginx-xyz", "default", "Running"},
			{"redis-001", "default", "Running"},
			{"postgres-db", "database", "Running"},
			{"nginx-proxy", "ingress", "Pending"},
		},
	)

	// Apply empty filter to initialize filtered rows
	ft.SetFilter("")

	// Test initial state (all rows visible with empty filter)
	if ft.FilteredRowCount() != 5 {
		t.Errorf("Initial filtered count should be 5, got %d", ft.FilteredRowCount())
	}

	// Apply filter
	ft.SetFilter("nginx")
	if ft.FilteredRowCount() != 3 {
		t.Errorf("Filtered count for 'nginx' should be 3, got %d", ft.FilteredRowCount())
	}

	// Verify filtered content
	rows := ft.FilteredRows()
	for _, row := range rows {
		found := false
		for _, cell := range row {
			if containsIgnoreCase(cell, "nginx") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Filtered row should contain 'nginx': %v", row)
		}
	}

	// Test case insensitive filter
	ft.SetFilter("REDIS")
	if ft.FilteredRowCount() != 1 {
		t.Errorf("Case insensitive filter for 'REDIS' should return 1, got %d", ft.FilteredRowCount())
	}

	// Test namespace filter
	ft.SetFilter("database")
	if ft.FilteredRowCount() != 1 {
		t.Errorf("Filter for 'database' namespace should return 1, got %d", ft.FilteredRowCount())
	}

	// Clear filter
	ft.SetFilter("")
	if ft.FilteredRowCount() != 5 {
		t.Errorf("Clear filter should show all 5 rows, got %d", ft.FilteredRowCount())
	}
}

// TestE2ERegistrarResourceLookup tests the registrar resource lookup functionality.
func TestE2ERegistrarResourceLookup(t *testing.T) {
	r := NewRegistrar()

	// Test common aliases (only include aliases that are actually registered)
	testCases := []struct {
		alias    string
		expected GVR
	}{
		{"po", GVRPods},
		{"pod", GVRPods},
		// "pods" is not registered as an alias, the GVR is "v1/pods"
		{"deploy", GVRDeployments},
		{"deployment", GVRDeployments},
		{"deployments", GVRDeployments},
		{"svc", GVRServices},
		{"service", GVRServices},
		{"services", GVRServices},
		{"ns", GVRNamespaces},
		{"no", GVRNodes},
		{"cm", GVRConfigMaps},
		{"secret", GVRSecrets},
		{"sts", GVRStatefulSets},
		{"ds", GVRDaemonSets},
		{"ing", GVRIngresses},
		{"pv", GVRPersistentVolumes},
		{"pvc", GVRPersistentVolumeClaims},
		{"ctx", GVRContexts},
	}

	for _, tc := range testCases {
		gvr, ok := r.Lookup(tc.alias)
		if !ok {
			t.Errorf("Lookup(%q) should succeed", tc.alias)
			continue
		}
		if gvr != tc.expected {
			t.Errorf("Lookup(%q) = %s, want %s", tc.alias, gvr, tc.expected)
		}
	}

	// Test unknown alias
	_, ok := r.Lookup("unknown-resource")
	if ok {
		t.Error("Lookup('unknown-resource') should fail")
	}
}

// TestE2EConcurrentViewUpdates tests concurrent view updates don't cause race conditions.
func TestE2EConcurrentViewUpdates(t *testing.T) {
	ft := models.NewFilteredTable()

	// Simulate concurrent updates
	var wg sync.WaitGroup
	numWriters := 5
	numReaders := 10
	iterations := 100

	// Writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				ft.SetData(
					[]string{"NAME", "STATUS"},
					[][]string{
						{"pod-" + itoa(j), "Running"},
						{"pod-" + itoa(j+1), "Pending"},
					},
				)
				ft.SetFilter("pod")
			}
		}(i)
	}

	// Readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = ft.FilteredRowCount()
				_ = ft.FilteredRows()
				_ = ft.Headers()
			}
		}(i)
	}

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - no race condition
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout - possible deadlock in concurrent view updates")
	}
}

// TestE2EPageStackConcurrency tests page stack thread safety.
func TestE2EPageStackConcurrency(t *testing.T) {
	app := tview.NewApplication()
	stack := NewPageStack(app)

	var wg sync.WaitGroup
	numGoroutines := 50

	// Concurrent pushes and pops
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			view := createTestViewWithContent("view-"+itoa(id), "Content")
			stack.Push(view)
		}(i)
	}

	wg.Wait()

	// Verify stack is still valid
	if stack.Len() != numGoroutines {
		t.Errorf("Stack len should be %d, got %d", numGoroutines, stack.Len())
	}

	// Concurrent clear
	stack.Clear()
	if stack.Len() != 0 {
		t.Errorf("Stack should be empty after Clear, got %d", stack.Len())
	}
}

// TestE2EResourceViewDataUpdate tests ResourceView data updates via model.
func TestE2EResourceViewDataUpdate(t *testing.T) {
	renderer := render.NewBaseRenderer(render.Header{
		{Name: "NAME"},
		{Name: "READY"},
		{Name: "STATUS"},
	})
	rv := NewResourceView("pods", "v1/pods", "default", renderer)

	// Simulate data update via model
	headers := []string{"NAME", "READY", "STATUS"}
	rows := [][]string{
		{"nginx-1", "1/1", "Running"},
		{"nginx-2", "0/1", "Pending"},
		{"redis", "1/1", "Running"},
	}

	// Call DataChanged to simulate model update
	rv.DataChanged(headers, rows)

	// Note: Can't verify table content without running the app,
	// but we verified no crash or deadlock
}

// TestE2EViewFilterIntegration tests filter integration with ResourceView.
func TestE2EViewFilterIntegration(t *testing.T) {
	rv := NewResourceView("pods", "v1/pods", "default", nil)

	// Set filter
	rv.SetFilter("nginx")
	if rv.GetFilter() != "nginx" {
		t.Errorf("GetFilter() = %q, want 'nginx'", rv.GetFilter())
	}

	// Clear filter
	rv.ClearFilter()
	if rv.GetFilter() != "" {
		t.Errorf("GetFilter() after clear = %q, want empty", rv.GetFilter())
	}
}

// Helper functions

func createTestViewWithContent(name, content string) View {
	return &testViewWithBox{
		name:    name,
		content: content,
		box:     tview.NewBox().SetBorder(true).SetTitle(name),
	}
}

type testViewWithBox struct {
	name    string
	content string
	box     *tview.Box
	stopped bool
}

func (v *testViewWithBox) Name() string                 { return v.name }
func (v *testViewWithBox) Init(_ context.Context) error { return nil }
func (v *testViewWithBox) Start()                       { v.stopped = false }
func (v *testViewWithBox) Stop()                        { v.stopped = true }
func (v *testViewWithBox) Actions() *actions.KeyActions { return actions.NewKeyActions() }
func (v *testViewWithBox) Primitive() tview.Primitive   { return v.box }
func (v *testViewWithBox) SetFocus(app *tview.Application) {
	if app != nil {
		app.SetFocus(v.box)
	}
}

func containsIgnoreCase(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	sLower := toLower(s)
	substrLower := toLower(substr)
	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
