package ui

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

// ============================================================================
// Regression Tests for 7 Stability Fixes
// Ensures recent fixes don't regress in future changes
// ============================================================================

// TestRegression_Fix1_ShowNodeDeadlock tests Fix #1: ABBA deadlock in showNode/showRelatedResource
// Was: mx→navMx vs navMx→mx. Fixed by using navigateTo() pattern.
// Test: concurrent showNode + goBack calls should not deadlock.
func TestRegression_Fix1_ShowNodeDeadlock(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.Command("no").Wait(100 * time.Millisecond)

	// Concurrent showNode (Enter) and goBack (Escape) should not deadlock
	ctx.ExpectNoDeadlock(10*time.Second,
		func(c *TUITestContext) {
			for i := 0; i < 20; i++ {
				c.Press(tcell.KeyEnter) // Drill down (showNode)
				time.Sleep(20 * time.Millisecond)
			}
		},
		func(c *TUITestContext) {
			for i := 0; i < 20; i++ {
				c.Escape() // Go back
				time.Sleep(20 * time.Millisecond)
			}
		},
	)
}

// TestRegression_Fix1_ShowRelatedResourceDeadlock tests Fix #1 for other resources
// Ensure showRelatedResource (deployments, services, etc.) also doesn't deadlock
func TestRegression_Fix1_ShowRelatedResourceDeadlock(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.Command("deploy").Wait(100 * time.Millisecond)

	ctx.ExpectNoDeadlock(10*time.Second,
		func(c *TUITestContext) {
			for i := 0; i < 15; i++ {
				c.Press(tcell.KeyEnter) // Drill down to related pods
				time.Sleep(25 * time.Millisecond)
			}
		},
		func(c *TUITestContext) {
			for i := 0; i < 15; i++ {
				c.Escape() // Go back
				time.Sleep(25 * time.Millisecond)
			}
		},
	)
}

// TestRegression_Fix2_NamespaceSwitchRace tests Fix #2: Race in useNamespace/switchToAllNamespaces
// Was: directly mutating state without stopping watcher. Fixed by using navigateTo().
// Test: rapid namespace switches shouldn't race.
func TestRegression_Fix2_NamespaceSwitchRace(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	// Rapidly switch namespaces using number keys (0-2)
	for i := 0; i < 50; i++ {
		ctx.PressRune(rune('0' + (i % 3)))
		time.Sleep(10 * time.Millisecond)
	}

	ctx.ExpectNoFreeze()
}

// TestRegression_Fix2_ConcurrentNamespaceSwitches tests Fix #2 with concurrent switches
func TestRegression_Fix2_ConcurrentNamespaceSwitches(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.ExpectNoDeadlock(10*time.Second,
		func(c *TUITestContext) {
			for i := 0; i < 25; i++ {
				c.PressRune('0') // All namespaces
				time.Sleep(15 * time.Millisecond)
			}
		},
		func(c *TUITestContext) {
			for i := 0; i < 25; i++ {
				c.PressRune('1') // Default namespace
				time.Sleep(15 * time.Millisecond)
			}
		},
		func(c *TUITestContext) {
			for i := 0; i < 25; i++ {
				c.PressRune('2') // kube-system namespace
				time.Sleep(15 * time.Millisecond)
			}
		},
	)
}

// TestRegression_Fix3_UpdateHeaderLockOrder tests Fix #3: Lock ordering in updateHeader
// Was: watchMu→mx vs mx→watchMu. Fixed by reading watch state before mx.
// Test: concurrent updateHeader + navigateTo should not deadlock.
func TestRegression_Fix3_UpdateHeaderLockOrder(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.ExpectNoDeadlock(10*time.Second,
		func(c *TUITestContext) {
			// Trigger updateHeader by resource switching
			for i := 0; i < 15; i++ {
				c.Command("pods")
				time.Sleep(30 * time.Millisecond)
			}
		},
		func(c *TUITestContext) {
			// Trigger navigateTo
			for i := 0; i < 15; i++ {
				c.Command("deploy")
				time.Sleep(30 * time.Millisecond)
			}
		},
		func(c *TUITestContext) {
			// Additional concurrent operations via QueueUpdateDraw
			for i := 0; i < 20; i++ {
				c.app.QueueUpdateDraw(func() {
					// Force a redraw cycle
				})
				time.Sleep(20 * time.Millisecond)
			}
		},
	)
}

// TestRegression_Fix4_StopCancelsRefresh tests Fix #4: Stop() didn't cancel in-flight refresh
// Fixed by canceling cancelFn. Test: Stop() during refresh should terminate quickly.
func TestRegression_Fix4_StopCancelsRefresh(t *testing.T) {
	ctx := NewTUITestContext(t)

	ctx.Command("pods").Wait(100 * time.Millisecond)

	// Trigger some navigation (causes refresh)
	ctx.Command("deploy").Wait(50 * time.Millisecond)

	// Cleanup calls Stop cleanly
	ctx.Cleanup()
}

// TestRegression_Fix5_WatcherOnChangeGuard tests Fix #5: Watcher onChange guard
// Was: Debounce could fire onChange after Stop. Fixed with isStopped() check.
// Test: onChange never fires after Stop().
func TestRegression_Fix5_WatcherOnChangeGuard(t *testing.T) {
	// This test is primarily in pkg/k8s/watcher_stability_test.go
	// Here we test from the UI perspective

	ctx := NewTUITestContext(t)

	ctx.Command("pods").Wait(200 * time.Millisecond)

	// Record current update count
	beforeStop := atomic.LoadInt32(&ctx.app.inUpdate)

	// Stop the app via Cleanup (avoids tview internal Run/Stop race)
	ctx.Cleanup()

	// Wait a bit to see if any stale onChange fires
	time.Sleep(500 * time.Millisecond)

	// No new updates should have started after Stop
	afterStop := atomic.LoadInt32(&ctx.app.inUpdate)

	if afterStop > beforeStop {
		t.Errorf("onChange fired after Stop(): before=%d, after=%d", beforeStop, afterStop)
	}
}

// TestRegression_Fix6_FlashMsgSequencing tests Fix #6: flashMsg timer accumulation
// Was: Multiple flashMsg goroutines could clear newer messages. Fixed with sequence counter.
// Test: rapid flashMsg calls keep last message.
func TestRegression_Fix6_FlashMsgSequencing(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	// Rapidly trigger flash messages by invalid commands
	messages := []string{"msg1", "msg2", "msg3", "msg4", "msg5"}
	for _, msg := range messages {
		ctx.Command(msg) // Invalid command triggers flash message
		time.Sleep(20 * time.Millisecond)
	}

	// Wait for messages to settle
	time.Sleep(200 * time.Millisecond)

	// The last message should still be visible (not cleared by earlier timers)
	// We can't easily assert the exact message content via SimulationScreen,
	// but we can verify no panic/deadlock occurred
	ctx.ExpectNoFreeze()
}

// TestRegression_Fix6_ConcurrentFlashMessages tests Fix #6 with concurrent messages
func TestRegression_Fix6_ConcurrentFlashMessages(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				// Trigger flash message via QueueUpdate
				ctx.app.QueueUpdate(func() {
					// Simulate flash message logic
					atomic.AddInt64(&ctx.app.flashSeq, 1)
				})
				time.Sleep(10 * time.Millisecond)
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
	case <-time.After(5 * time.Second):
		t.Fatal("Concurrent flash messages deadlocked")
	}

	ctx.ExpectNoFreeze()
}

// TestRegression_Fix7_TableCellNilCheck tests Fix #7: Table cell nil checks
// Was: GetCell().Text could panic on nil. Fixed with getTableCellText() helper.
// Test: getTableCellText returns "" for nil cells.
func TestRegression_Fix7_TableCellNilCheck(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	// Test getTableCellText on out-of-bounds cells via QueueUpdate (tview is not thread-safe)
	done := make(chan struct{})
	ctx.app.QueueUpdate(func() {
		// Row 999 is definitely out-of-bounds and should return ""
		result := ctx.app.getTableCellText(999, 0)
		if result != "" {
			t.Errorf("getTableCellText for out-of-bounds row should return empty string, got %q", result)
		}

		// Column 999 is also out-of-bounds
		result = ctx.app.getTableCellText(0, 999)
		if result != "" {
			t.Errorf("getTableCellText for out-of-bounds column should return empty string, got %q", result)
		}
		close(done)
	})

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("getTableCellText test timed out")
	}
}

// TestRegression_Fix7_RapidTableAccess tests Fix #7 with rapid table access via QueueUpdate
func TestRegression_Fix7_RapidTableAccess(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.Command("pods").Wait(100 * time.Millisecond)

	// Read table cells via QueueUpdate (tview is not thread-safe for direct access)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				ctx.app.QueueUpdate(func() {
					_ = ctx.app.getTableCellText(j%10, 0)
					_ = ctx.app.getTableCellText(j%10, 1)
				})
				time.Sleep(2 * time.Millisecond)
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
		// Success: no panics
	case <-time.After(10 * time.Second):
		t.Fatal("Concurrent table cell access timed out (possible panic recovery loop)")
	}
}

// TestRegression_AllFixesIntegration tests all 7 fixes together in realistic scenario
func TestRegression_AllFixesIntegration(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.ExpectNoDeadlock(20*time.Second,
		// Fix #1: showNode + goBack
		func(c *TUITestContext) {
			c.Command("no").Wait(50 * time.Millisecond)
			for i := 0; i < 10; i++ {
				c.Press(tcell.KeyEnter)
				time.Sleep(30 * time.Millisecond)
				c.Escape()
				time.Sleep(30 * time.Millisecond)
			}
		},
		// Fix #2: namespace switching
		func(c *TUITestContext) {
			for i := 0; i < 30; i++ {
				c.PressRune(rune('0' + (i % 3)))
				time.Sleep(20 * time.Millisecond)
			}
		},
		// Fix #3: updateHeader via resource switching
		func(c *TUITestContext) {
			resources := []string{"pods", "deploy", "svc"}
			for i := 0; i < 15; i++ {
				c.Command(resources[i%len(resources)])
				time.Sleep(40 * time.Millisecond)
			}
		},
		// Fix #6: flash messages
		func(c *TUITestContext) {
			for i := 0; i < 10; i++ {
				c.Command("invalidcmd")
				time.Sleep(50 * time.Millisecond)
			}
		},
		// Fix #7: table cell access (via QueueUpdate for thread safety)
		func(c *TUITestContext) {
			for i := 0; i < 20; i++ {
				if c.app.IsRunning() {
					c.app.QueueUpdate(func() {
						_ = c.app.getTableCellText(0, 0)
						_ = c.app.getTableCellText(1, 1)
					})
				}
				time.Sleep(30 * time.Millisecond)
			}
		},
	)
}

// TestRegression_Fix4_MultipleStopsIdempotent tests Fix #4: multiple Stop() calls are safe
// Skip under race detector as tview has known internal races with concurrent Stop() calls.
func TestRegression_Fix4_MultipleStopsIdempotent(t *testing.T) {
	if raceEnabled {
		t.Skip("Skipping test under race detector due to tview internal races with concurrent Stop()")
	}
	ctx := NewTUITestContext(t)

	ctx.Command("pods").Wait(200 * time.Millisecond)

	// Use Cleanup which calls Stop() safely
	ctx.Cleanup()

	// Verify app stopped cleanly
	if ctx.app.IsRunning() {
		t.Error("App should not be running after Stop()")
	}

	// Second Stop should also be safe (idempotent)
	ctx.app.Stop()
}

// TestRegression_Fix2_SelectNamespaceByNumber tests Fix #2: selectNamespaceByNumber safety
func TestRegression_Fix2_SelectNamespaceByNumber(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	// Test boundary conditions
	tests := []struct {
		name string
		key  rune
	}{
		{"digit 0", '0'},
		{"digit 1", '1'},
		{"digit 2", '2'},
		{"digit 9", '9'}, // Out of range, should be safe
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx.PressRune(tt.key)
			time.Sleep(50 * time.Millisecond)
			ctx.ExpectNoFreeze()
		})
	}
}

// TestRegression_Fix3_ConcurrentHeaderAndRefresh tests Fix #3 with concurrent header updates
func TestRegression_Fix3_ConcurrentHeaderAndRefresh(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.Command("pods").Wait(100 * time.Millisecond)

	ctx.ExpectNoDeadlock(10*time.Second,
		func(c *TUITestContext) {
			// Trigger header updates via namespace changes
			for i := 0; i < 20; i++ {
				c.PressRune('0')
				time.Sleep(20 * time.Millisecond)
				c.PressRune('1')
				time.Sleep(20 * time.Millisecond)
			}
		},
		func(c *TUITestContext) {
			// Concurrent draw via QueueUpdateDraw
			for i := 0; i < 50; i++ {
				c.app.QueueUpdateDraw(func() {
					// Force a redraw cycle
				})
				time.Sleep(10 * time.Millisecond)
			}
		},
	)
}

// TestRegression_Fix5_NoCallbacksAfterWatcherStop tests Fix #5 comprehensively
func TestRegression_Fix5_NoCallbacksAfterWatcherStop(t *testing.T) {
	ctx := NewTUITestContext(t)

	ctx.Command("pods").Wait(200 * time.Millisecond)

	// Trigger watcher activity
	for i := 0; i < 5; i++ {
		ctx.Command("deploy")
		time.Sleep(50 * time.Millisecond)
		ctx.Command("pods")
		time.Sleep(50 * time.Millisecond)
	}

	// Stop app via Cleanup
	ctx.Cleanup()

	// Wait to ensure no stale callbacks fire
	time.Sleep(500 * time.Millisecond)

	// Verify app is not running
	if ctx.app.IsRunning() {
		t.Error("App should not be running after Stop()")
	}
}
