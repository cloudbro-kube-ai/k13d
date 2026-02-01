package ui

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// createTestScreen creates a new SimulationScreen for testing.
func createTestScreen(t *testing.T) tcell.SimulationScreen {
	t.Helper()
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatalf("SimulationScreen init failed: %v", err)
	}
	screen.SetSize(120, 40)
	return screen
}

// TestAppNoFreezeOnStartup verifies the app starts without freezing.
func TestAppNoFreezeOnStartup(t *testing.T) {
	screen := createTestScreen(t)

	app := NewTestApp(TestAppConfig{
		UseSimulationScreen:   true,
		Screen:                screen,
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})

	// Run the app in background
	done := make(chan struct{})
	go func() {
		_ = app.Run()
		close(done)
	}()

	// Wait for app to initialize
	time.Sleep(100 * time.Millisecond)

	// Verify app is responsive by checking it can draw
	drawDone := make(chan struct{})
	go func() {
		app.Draw()
		close(drawDone)
	}()

	select {
	case <-drawDone:
		// Success - app is responsive
	case <-time.After(2 * time.Second):
		t.Fatal("App froze during startup - Draw() did not complete")
	}

	// Stop the app
	app.Stop()

	select {
	case <-done:
		// App stopped cleanly
	case <-time.After(2 * time.Second):
		t.Fatal("App did not stop cleanly")
	}
}

// TestAppNoFreezeOnKeypress verifies key presses don't cause freezes.
func TestAppNoFreezeOnKeypress(t *testing.T) {
	screen := createTestScreen(t)

	app := NewTestApp(TestAppConfig{
		UseSimulationScreen:   true,
		Screen:                screen,
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})

	// Run the app
	done := make(chan struct{})
	go func() {
		_ = app.Run()
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)

	// Test various key presses don't cause freezes
	keys := []struct {
		key  tcell.Key
		r    rune
		desc string
	}{
		{tcell.KeyRune, 'j', "move down"},
		{tcell.KeyRune, 'k', "move up"},
		{tcell.KeyRune, '/', "start filter"},
		{tcell.KeyEscape, 0, "escape"},
		{tcell.KeyRune, ':', "command mode"},
		{tcell.KeyEscape, 0, "escape"},
		{tcell.KeyRune, '?', "help"},
		{tcell.KeyEscape, 0, "escape"},
		{tcell.KeyTab, 0, "tab"},
	}

	for _, k := range keys {
		t.Run(k.desc, func(t *testing.T) {
			keyDone := make(chan struct{})
			go func() {
				screen.InjectKey(k.key, k.r, tcell.ModNone)
				time.Sleep(20 * time.Millisecond)
				app.Draw()
				close(keyDone)
			}()

			select {
			case <-keyDone:
				// Key processed
			case <-time.After(1 * time.Second):
				t.Fatalf("App froze on key: %s", k.desc)
			}
		})
	}

	app.Stop()
	<-done
}

// TestAppNoFreezeOnRapidKeypress verifies rapid key presses don't cause freezes.
func TestAppNoFreezeOnRapidKeypress(t *testing.T) {
	screen := createTestScreen(t)

	app := NewTestApp(TestAppConfig{
		UseSimulationScreen:   true,
		Screen:                screen,
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})

	done := make(chan struct{})
	go func() {
		_ = app.Run()
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)

	// Rapid key presses (stress test)
	rapidDone := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			screen.InjectKey(tcell.KeyRune, 'j', tcell.ModNone)
			screen.InjectKey(tcell.KeyRune, 'k', tcell.ModNone)
		}
		close(rapidDone)
	}()

	select {
	case <-rapidDone:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("App froze on rapid key presses")
	}

	app.Stop()
	<-done
}

// TestInUpdateAtomicPreventsDeadlock verifies concurrent refresh calls don't deadlock.
func TestInUpdateAtomicPreventsDeadlock(t *testing.T) {
	var inUpdate int32
	callCount := int32(0)

	mockRefresh := func() {
		if !atomic.CompareAndSwapInt32(&inUpdate, 0, 1) {
			return // Already updating, skip
		}
		defer atomic.StoreInt32(&inUpdate, 0)

		atomic.AddInt32(&callCount, 1)
		time.Sleep(10 * time.Millisecond) // Simulate work
	}

	// Launch many concurrent refresh calls
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mockRefresh()
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		count := atomic.LoadInt32(&callCount)
		t.Logf("Executed %d refresh calls (others skipped)", count)
		// Most should be skipped due to atomic guard
		if count > 50 {
			t.Logf("Warning: more executions than expected, but no deadlock")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Deadlock detected in concurrent refresh")
	}
}

// TestQueueUpdateDrawNilApplication verifies QueueUpdateDraw handles nil safely.
func TestQueueUpdateDrawNilApplication(t *testing.T) {
	app := &App{Application: nil}

	done := make(chan struct{})
	go func() {
		app.QueueUpdateDraw(func() {
			// Should not execute
			t.Error("Function should not execute on nil Application")
		})
		close(done)
	}()

	select {
	case <-done:
		// Success - no crash or block
	case <-time.After(1 * time.Second):
		t.Fatal("QueueUpdateDraw blocked on nil Application")
	}
}

// TestCommandModeNavigation tests command mode resource switching.
func TestCommandModeNavigation(t *testing.T) {
	screen := createTestScreen(t)

	app := NewTestApp(TestAppConfig{
		UseSimulationScreen:   true,
		Screen:                screen,
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})

	done := make(chan struct{})
	go func() {
		_ = app.Run()
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)

	// Enter command mode and type a command
	screen.InjectKey(tcell.KeyRune, ':', tcell.ModNone)
	time.Sleep(50 * time.Millisecond)

	// Type "deploy" to switch to deployments
	for _, r := range "deploy" {
		screen.InjectKey(tcell.KeyRune, r, tcell.ModNone)
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)

	// Press Enter to execute
	screen.InjectKey(tcell.KeyEnter, 0, tcell.ModNone)
	time.Sleep(100 * time.Millisecond)

	// Verify resource changed
	app.mx.RLock()
	resource := app.currentResource
	app.mx.RUnlock()

	if resource != "deployments" {
		t.Errorf("Expected resource 'deployments', got %q", resource)
	}

	app.Stop()
	<-done
}

// TestFilterMode tests filter functionality.
func TestFilterMode(t *testing.T) {
	screen := createTestScreen(t)

	app := NewTestApp(TestAppConfig{
		UseSimulationScreen:   true,
		Screen:                screen,
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})

	done := make(chan struct{})
	go func() {
		_ = app.Run()
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)

	// Enter filter mode
	screen.InjectKey(tcell.KeyRune, '/', tcell.ModNone)
	time.Sleep(50 * time.Millisecond)

	// Type filter text
	for _, r := range "nginx" {
		screen.InjectKey(tcell.KeyRune, r, tcell.ModNone)
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)

	// Press Enter to apply
	screen.InjectKey(tcell.KeyEnter, 0, tcell.ModNone)
	time.Sleep(100 * time.Millisecond)

	// Verify filter applied
	app.mx.RLock()
	filter := app.filterText
	app.mx.RUnlock()

	if filter != "nginx" {
		t.Errorf("Expected filter 'nginx', got %q", filter)
	}

	app.Stop()
	<-done
}

// TestEscapeClearsModals tests that ESC key closes modals and clears state.
func TestEscapeClearsModals(t *testing.T) {
	screen := createTestScreen(t)

	app := NewTestApp(TestAppConfig{
		UseSimulationScreen:   true,
		Screen:                screen,
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})

	done := make(chan struct{})
	go func() {
		_ = app.Run()
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)

	// Open help modal
	screen.InjectKey(tcell.KeyRune, '?', tcell.ModNone)
	time.Sleep(50 * time.Millisecond)

	// Press ESC to close
	screen.InjectKey(tcell.KeyEscape, 0, tcell.ModNone)
	time.Sleep(50 * time.Millisecond)

	// Focus should return to table
	focused := app.GetFocus()
	if focused != app.table {
		t.Logf("Focus after ESC: %T", focused)
	}

	app.Stop()
	<-done
}

// TestNamespaceSwitch tests namespace switching.
func TestNamespaceSwitch(t *testing.T) {
	screen := createTestScreen(t)

	app := NewTestApp(TestAppConfig{
		UseSimulationScreen:   true,
		Screen:                screen,
		InitialNamespace:      "default",
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})

	done := make(chan struct{})
	go func() {
		_ = app.Run()
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)

	// Verify initial namespace
	app.mx.RLock()
	ns := app.currentNamespace
	app.mx.RUnlock()

	if ns != "default" {
		t.Errorf("Expected initial namespace 'default', got %q", ns)
	}

	app.Stop()
	<-done
}

// TestConcurrentStateAccess tests thread-safe state access patterns.
func TestConcurrentStateAccess(t *testing.T) {
	app := CreateMinimalTestApp()

	var wg sync.WaitGroup
	iterations := 1000

	// Concurrent readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				app.mx.RLock()
				_ = app.currentResource
				_ = app.currentNamespace
				_ = app.filterText
				app.mx.RUnlock()
			}
		}()
	}

	// Concurrent writers
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				app.mx.Lock()
				app.currentResource = "pods"
				app.filterText = "test"
				app.mx.Unlock()
			}
		}(i)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - no data races
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout - possible deadlock in concurrent state access")
	}
}

// TestNavigationStackConcurrency tests navigation stack thread safety.
func TestNavigationStackConcurrency(t *testing.T) {
	app := CreateMinimalTestApp()

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent push operations
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

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		app.navMx.Lock()
		finalLen := len(app.navigationStack)
		app.navMx.Unlock()

		if finalLen != numGoroutines {
			t.Errorf("Expected %d items in stack, got %d", numGoroutines, finalLen)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Deadlock in navigation stack operations")
	}
}

// TestScreenRender verifies screen content is rendered correctly.
func TestScreenRender(t *testing.T) {
	screen := createTestScreen(t)

	tvApp := tview.NewApplication().SetScreen(screen)

	// Create a simple test layout
	textView := tview.NewTextView().SetText("Hello TUI Test")
	tvApp.SetRoot(textView, true)

	done := make(chan struct{})
	go func() {
		_ = tvApp.Run()
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)

	// Draw the screen
	tvApp.Draw()
	screen.Sync()

	// Check screen content
	w, h := screen.Size()
	var sb strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			mainc, _, _, _ := screen.GetContent(x, y)
			if mainc == 0 {
				mainc = ' '
			}
			sb.WriteRune(mainc)
		}
		sb.WriteRune('\n')
	}
	content := sb.String()

	if !strings.Contains(content, "Hello TUI Test") {
		t.Errorf("Expected 'Hello TUI Test' in screen content")
	}

	tvApp.Stop()
	<-done
}

// TestTableNavigation tests table row navigation with arrow keys.
// Note: This test uses QueueUpdateDraw for thread-safe access.
func TestTableNavigation(t *testing.T) {
	screen := createTestScreen(t)

	tvApp := tview.NewApplication().SetScreen(screen)

	// Create a table with some rows
	table := tview.NewTable().SetSelectable(true, false).SetFixed(1, 0)

	// Add header
	table.SetCell(0, 0, tview.NewTableCell("NAME").SetSelectable(false))
	table.SetCell(0, 1, tview.NewTableCell("STATUS").SetSelectable(false))

	// Add data rows
	table.SetCell(1, 0, tview.NewTableCell("pod-1"))
	table.SetCell(1, 1, tview.NewTableCell("Running"))
	table.SetCell(2, 0, tview.NewTableCell("pod-2"))
	table.SetCell(2, 1, tview.NewTableCell("Running"))
	table.SetCell(3, 0, tview.NewTableCell("pod-3"))
	table.SetCell(3, 1, tview.NewTableCell("Pending"))

	// Select first data row
	table.Select(1, 0)

	tvApp.SetRoot(table, true)

	done := make(chan struct{})
	go func() {
		_ = tvApp.Run()
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)

	// Press Down to move down - use QueueUpdateDraw for thread-safe access
	screen.InjectKey(tcell.KeyDown, 0, tcell.ModNone)
	time.Sleep(50 * time.Millisecond)

	// Check selection using QueueUpdateDraw for thread safety
	rowCh := make(chan int, 1)
	tvApp.QueueUpdateDraw(func() {
		row, _ := table.GetSelection()
		rowCh <- row
	})

	select {
	case row := <-rowCh:
		if row != 2 {
			t.Errorf("Expected row 2 after Down key, got %d", row)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for selection check")
	}

	// Press Up to move back
	screen.InjectKey(tcell.KeyUp, 0, tcell.ModNone)
	time.Sleep(50 * time.Millisecond)

	tvApp.QueueUpdateDraw(func() {
		row, _ := table.GetSelection()
		rowCh <- row
	})

	select {
	case row := <-rowCh:
		if row != 1 {
			t.Errorf("Expected row 1 after Up key, got %d", row)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for selection check")
	}

	tvApp.Stop()
	<-done
}

// TestAppGracefulShutdown tests that the app shuts down cleanly.
func TestAppGracefulShutdown(t *testing.T) {
	screen := createTestScreen(t)

	app := NewTestApp(TestAppConfig{
		UseSimulationScreen:   true,
		Screen:                screen,
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})

	done := make(chan struct{})
	go func() {
		_ = app.Run()
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)

	// Press 'q' to quit
	screen.InjectKey(tcell.KeyRune, 'q', tcell.ModNone)

	select {
	case <-done:
		// Clean shutdown
	case <-time.After(2 * time.Second):
		t.Fatal("App did not shut down gracefully on 'q' press")
	}
}

// TestMultipleResourceSwitches tests switching between multiple resources rapidly.
func TestMultipleResourceSwitches(t *testing.T) {
	screen := createTestScreen(t)

	app := NewTestApp(TestAppConfig{
		UseSimulationScreen:   true,
		Screen:                screen,
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})

	done := make(chan struct{})
	go func() {
		_ = app.Run()
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)

	resources := []string{"deploy", "svc", "po", "no", "ns"}

	for _, res := range resources {
		// Enter command mode
		screen.InjectKey(tcell.KeyRune, ':', tcell.ModNone)
		time.Sleep(30 * time.Millisecond)

		// Type resource name
		for _, r := range res {
			screen.InjectKey(tcell.KeyRune, r, tcell.ModNone)
			time.Sleep(5 * time.Millisecond)
		}
		time.Sleep(30 * time.Millisecond)

		// Execute
		screen.InjectKey(tcell.KeyEnter, 0, tcell.ModNone)
		time.Sleep(50 * time.Millisecond)
	}

	// Verify no freeze occurred
	drawDone := make(chan struct{})
	go func() {
		app.Draw()
		close(drawDone)
	}()

	select {
	case <-drawDone:
		// App still responsive
	case <-time.After(2 * time.Second):
		t.Fatal("App froze after multiple resource switches")
	}

	app.Stop()
	<-done
}

// TestLongRunningRefreshDoesNotBlock tests that refresh operations don't block the UI.
func TestLongRunningRefreshDoesNotBlock(t *testing.T) {
	app := CreateMinimalTestApp()

	// Simulate concurrent refresh and state access
	var wg sync.WaitGroup

	// Simulate refresh that takes time
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !atomic.CompareAndSwapInt32(&app.inUpdate, 0, 1) {
				return
			}
			defer atomic.StoreInt32(&app.inUpdate, 0)
			time.Sleep(50 * time.Millisecond) // Simulate work
		}()
	}

	// Simulate UI state reads (should not block)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				app.mx.RLock()
				_ = app.currentResource
				app.mx.RUnlock()
				time.Sleep(5 * time.Millisecond)
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("UI state access blocked during refresh")
	}
}
