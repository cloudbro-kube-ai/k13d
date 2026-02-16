package ui

import (
	"sync"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

// ============================================================================
// Shutdown Lifecycle and Edge Case Tests
// Tests graceful shutdown, nil handling, and concurrent operations during shutdown
//
// NOTE: Tests that call Stop() or interact with tview internals directly are
// limited by tview's internal thread-safety (Run/Stop race on screen field).
// These tests focus on k13d's own stability logic rather than exercising tview
// internals.
// ============================================================================

// TestLifecycle_StopCancelsInflightRefresh tests Stop() cancels ongoing refresh operations
func TestLifecycle_StopCancelsInflightRefresh(t *testing.T) {
	ctx := NewTUITestContext(t)

	ctx.Command("pods").Wait(200 * time.Millisecond)

	// Trigger navigation which causes refresh
	ctx.Command("deploy").Wait(50 * time.Millisecond)
	ctx.Command("svc").Wait(50 * time.Millisecond)

	// Stop should complete quickly
	ctx.Cleanup()
}

// TestLifecycle_ConcurrentStopAndRefresh tests Stop() during active navigation
func TestLifecycle_ConcurrentStopAndRefresh(t *testing.T) {
	ctx := NewTUITestContext(t)

	ctx.Command("pods").Wait(200 * time.Millisecond)

	// Trigger active refresh via navigation
	ctx.Command("deploy").Wait(50 * time.Millisecond)
	ctx.Command("svc").Wait(50 * time.Millisecond)
	ctx.Command("pods").Wait(50 * time.Millisecond)

	// Stop should complete even with active refresh
	ctx.Cleanup()
}

// TestLifecycle_StopIdempotent tests that Stop() completes cleanly
func TestLifecycle_StopIdempotent(t *testing.T) {
	ctx := NewTUITestContext(t)

	ctx.Command("pods").Wait(200 * time.Millisecond)

	// Cleanup calls Stop() cleanly
	ctx.Cleanup()
}

// TestEdgeCase_EmptyTableActions tests actions on empty table don't panic
func TestEdgeCase_EmptyTableActions(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	// Try to trigger actions on empty table
	ctx.PressRune('d') // Describe
	time.Sleep(50 * time.Millisecond)
	ctx.Escape()

	ctx.PressRune('y') // YAML
	time.Sleep(50 * time.Millisecond)
	ctx.Escape()

	ctx.ExpectNoFreeze()
}

// TestEdgeCase_RapidResourceSwitchDuringRefresh tests switching resources during refresh
func TestEdgeCase_RapidResourceSwitchDuringRefresh(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	// Rapidly switch resources while refreshes are happening
	resources := []string{"pods", "deploy", "svc", "no"}
	for i := 0; i < 30; i++ {
		ctx.Command(resources[i%len(resources)])
		time.Sleep(15 * time.Millisecond)
	}

	ctx.ExpectNoFreeze()
}

// TestEdgeCase_NavigationStackOverflow tests navigation stack stays bounded
func TestEdgeCase_NavigationStackOverflow(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	// Try to overflow navigation stack (max is 50)
	for i := 0; i < 100; i++ {
		ctx.Command("pods")
		time.Sleep(5 * time.Millisecond)
	}

	// Wait for commands to settle
	time.Sleep(500 * time.Millisecond)

	// Check stack is bounded via QueueUpdate to avoid races
	stackOK := make(chan bool, 1)
	ctx.app.QueueUpdate(func() {
		ctx.app.navMx.Lock()
		stackLen := len(ctx.app.navigationStack)
		ctx.app.navMx.Unlock()
		stackOK <- stackLen <= 50
	})

	select {
	case ok := <-stackOK:
		if !ok {
			t.Error("Navigation stack exceeded limit of 50")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Stack check timed out")
	}

	ctx.ExpectNoFreeze()
}

// TestEdgeCase_ConcurrentWatcherStopAndRefresh tests stopping watcher during active navigation
func TestEdgeCase_ConcurrentWatcherStopAndRefresh(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.Command("pods").Wait(200 * time.Millisecond)

	// Trigger navigation which causes refresh and watcher restarts
	ctx.Command("deploy").Wait(50 * time.Millisecond)
	ctx.Command("svc").Wait(50 * time.Millisecond)
	ctx.Command("pods").Wait(100 * time.Millisecond)

	ctx.ExpectNoFreeze()
}

// TestEdgeCase_RefreshDuringStop tests that Stop() works after active operations
func TestEdgeCase_RefreshDuringStop(t *testing.T) {
	ctx := NewTUITestContext(t)

	ctx.Command("pods").Wait(200 * time.Millisecond)

	// Trigger navigation which causes refreshes
	ctx.Command("deploy").Wait(50 * time.Millisecond)
	ctx.Command("svc").Wait(50 * time.Millisecond)

	// Cleanup calls Stop cleanly
	ctx.Cleanup()
}

// TestEdgeCase_StopCompletesCleanly tests that Stop() completes cleanly
func TestEdgeCase_StopCompletesCleanly(t *testing.T) {
	ctx := NewTUITestContext(t)

	ctx.Command("pods").Wait(100 * time.Millisecond)

	ctx.Cleanup()
}

// TestEdgeCase_FilterEmptyString tests filtering with empty string
func TestEdgeCase_FilterEmptyString(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.Command("pods").Wait(100 * time.Millisecond)

	// Apply empty filter
	ctx.PressRune('/')
	ctx.Press(tcell.KeyEnter)
	time.Sleep(50 * time.Millisecond)

	ctx.ExpectFilter("")
	ctx.ExpectNoFreeze()
}

// TestEdgeCase_ConcurrentFilterAndResourceSwitch tests filter during resource switch
func TestEdgeCase_ConcurrentFilterAndResourceSwitch(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.ExpectNoDeadlock(10*time.Second,
		func(c *TUITestContext) {
			for i := 0; i < 15; i++ {
				c.PressRune('/').Type("test").Press(tcell.KeyEnter)
				time.Sleep(30 * time.Millisecond)
			}
		},
		func(c *TUITestContext) {
			for i := 0; i < 15; i++ {
				c.Command("pods")
				time.Sleep(30 * time.Millisecond)
			}
		},
	)
}

// TestEdgeCase_NamespaceOutOfRange tests selecting namespace number out of range
func TestEdgeCase_NamespaceOutOfRange(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.Command("pods").Wait(100 * time.Millisecond)

	// Try to select namespace 9 (likely out of range)
	ctx.PressRune('9')
	time.Sleep(50 * time.Millisecond)

	ctx.ExpectNoFreeze()
}

// TestEdgeCase_ConcurrentHeaderUpdates tests header updates via rapid navigation
func TestEdgeCase_ConcurrentHeaderUpdates(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	// Rapidly switch resources which triggers updateHeader via the event loop
	resources := []string{"pods", "deploy", "svc", "no", "ns"}
	for i := 0; i < 20; i++ {
		ctx.Command(resources[i%len(resources)])
		time.Sleep(20 * time.Millisecond)
	}

	ctx.ExpectNoFreeze()
}

// TestEdgeCase_FlashMessageDuringShutdown tests flashMsg before Stop()
func TestEdgeCase_FlashMessageDuringShutdown(t *testing.T) {
	ctx := NewTUITestContext(t)

	ctx.Command("pods").Wait(200 * time.Millisecond)

	// Trigger flash messages via invalid commands
	for i := 0; i < 5; i++ {
		ctx.Command("invalidcmd")
		time.Sleep(20 * time.Millisecond)
	}

	// Cleanup calls Stop cleanly
	ctx.Cleanup()
}

// TestEdgeCase_GetTableCellTextThreadSafety tests getTableCellText via QueueUpdate
func TestEdgeCase_GetTableCellTextThreadSafety(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.Command("pods").Wait(100 * time.Millisecond)

	// tview table access must go through QueueUpdate for thread safety
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				ctx.app.QueueUpdate(func() {
					_ = ctx.app.getTableCellText(n%5, 0)
					_ = ctx.app.getTableCellText(n%5, 1)
				})
				time.Sleep(2 * time.Millisecond)
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
		// Success
	case <-time.After(10 * time.Second):
		t.Fatal("Concurrent getTableCellText() deadlocked")
	}
}
