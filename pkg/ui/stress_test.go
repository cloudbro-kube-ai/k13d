package ui

import (
	"sync"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

// ============================================================================
// Stress Tests for Rapid TUI Navigation
// Tests concurrent rapid operations that could expose race conditions
// ============================================================================

// TestStress_RapidResourceSwitch tests rapid sequential resource switching
func TestStress_RapidResourceSwitch(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	resources := []string{"pods", "deploy", "svc", "no", "ns", "pods", "deploy"}

	// Rapidly switch between resources
	for i := 0; i < 3; i++ {
		for _, res := range resources {
			ctx.Command(res)
			time.Sleep(5 * time.Millisecond) // Very rapid switching
		}
	}

	ctx.ExpectNoFreeze()
}

// TestStress_ConcurrentResourceSwitch tests concurrent resource switches
func TestStress_ConcurrentResourceSwitch(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.ExpectNoDeadlock(10*time.Second,
		func(c *TUITestContext) {
			for i := 0; i < 20; i++ {
				c.Command("pods")
				time.Sleep(10 * time.Millisecond)
			}
		},
		func(c *TUITestContext) {
			for i := 0; i < 20; i++ {
				c.Command("deploy")
				time.Sleep(10 * time.Millisecond)
			}
		},
		func(c *TUITestContext) {
			for i := 0; i < 20; i++ {
				c.Command("svc")
				time.Sleep(10 * time.Millisecond)
			}
		},
	)
}

// TestStress_RapidNamespaceSwitch tests rapid namespace switching
func TestStress_RapidNamespaceSwitch(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	// Simulate rapid namespace switches via number keys (0-5)
	for i := 0; i < 50; i++ {
		ctx.PressRune(rune('0' + (i % 3))) // Toggle between 0, 1, 2
		time.Sleep(5 * time.Millisecond)
	}

	ctx.ExpectNoFreeze()
}

// TestStress_ConcurrentNamespaceSwitch tests concurrent namespace switching
func TestStress_ConcurrentNamespaceSwitch(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.ExpectNoDeadlock(10*time.Second,
		func(c *TUITestContext) {
			for i := 0; i < 30; i++ {
				c.PressRune('0') // All namespaces
				time.Sleep(10 * time.Millisecond)
			}
		},
		func(c *TUITestContext) {
			for i := 0; i < 30; i++ {
				c.PressRune('1') // Default namespace
				time.Sleep(10 * time.Millisecond)
			}
		},
	)
}

// TestStress_RapidFilterToggle tests rapid filter on/off
func TestStress_RapidFilterToggle(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	// Rapidly toggle filter mode
	for i := 0; i < 30; i++ {
		ctx.PressRune('/').Type("test").Press(tcell.KeyEnter)
		time.Sleep(10 * time.Millisecond)
		ctx.PressRune('/').Type("").Press(tcell.KeyEnter) // Clear filter
		time.Sleep(10 * time.Millisecond)
	}

	ctx.ExpectNoFreeze()
}

// TestStress_ConcurrentFilteringAndNavigation tests filtering while navigating
func TestStress_ConcurrentFilteringAndNavigation(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.ExpectNoDeadlock(15*time.Second,
		func(c *TUITestContext) {
			for i := 0; i < 10; i++ {
				c.PressRune('/').Type("filter").Press(tcell.KeyEnter)
				time.Sleep(50 * time.Millisecond)
			}
		},
		func(c *TUITestContext) {
			for i := 0; i < 20; i++ {
				c.PressRune('j').PressRune('k')
				time.Sleep(20 * time.Millisecond)
			}
		},
	)
}

// TestStress_RapidNavigateToAndBack tests rapid drilldown and back operations
func TestStress_RapidNavigateToAndBack(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.Command("pods").Wait(100 * time.Millisecond)

	// Rapidly navigate forward and back
	for i := 0; i < 20; i++ {
		ctx.Press(tcell.KeyEnter) // Drill down
		time.Sleep(10 * time.Millisecond)
		ctx.Escape() // Go back
		time.Sleep(10 * time.Millisecond)
	}

	ctx.ExpectNoFreeze()
}

// TestStress_ConcurrentNavigateToAndGoBack tests concurrent drilldown and back
func TestStress_ConcurrentNavigateToAndGoBack(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.Command("pods").Wait(100 * time.Millisecond)

	ctx.ExpectNoDeadlock(10*time.Second,
		func(c *TUITestContext) {
			for i := 0; i < 15; i++ {
				c.Press(tcell.KeyEnter)
				time.Sleep(20 * time.Millisecond)
			}
		},
		func(c *TUITestContext) {
			for i := 0; i < 15; i++ {
				c.Escape()
				time.Sleep(20 * time.Millisecond)
			}
		},
	)
}

// TestStress_RapidShowNodeCalls tests rapid showNode operations
func TestStress_RapidShowNodeCalls(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.Command("no").Wait(100 * time.Millisecond)

	// Rapidly trigger node view
	for i := 0; i < 30; i++ {
		ctx.Press(tcell.KeyEnter)
		time.Sleep(10 * time.Millisecond)
		ctx.Escape()
		time.Sleep(10 * time.Millisecond)
	}

	ctx.ExpectNoFreeze()
}

// TestStress_ConcurrentShowNodeAndGoBack tests concurrent showNode + goBack
func TestStress_ConcurrentShowNodeAndGoBack(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.Command("no").Wait(100 * time.Millisecond)

	// This tests Fix #1 - deadlock in showNode/showRelatedResource
	ctx.ExpectNoDeadlock(10*time.Second,
		func(c *TUITestContext) {
			for i := 0; i < 15; i++ {
				c.Press(tcell.KeyEnter) // showNode
				time.Sleep(20 * time.Millisecond)
			}
		},
		func(c *TUITestContext) {
			for i := 0; i < 15; i++ {
				c.Escape() // goBack
				time.Sleep(20 * time.Millisecond)
			}
		},
	)
}

// TestStress_MixedConcurrentOperations tests a realistic mix of concurrent operations
func TestStress_MixedConcurrentOperations(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.ExpectNoDeadlock(15*time.Second,
		// Resource switching goroutine
		func(c *TUITestContext) {
			resources := []string{"pods", "deploy", "svc"}
			for i := 0; i < 10; i++ {
				c.Command(resources[i%len(resources)])
				time.Sleep(30 * time.Millisecond)
			}
		},
		// Namespace switching goroutine
		func(c *TUITestContext) {
			for i := 0; i < 15; i++ {
				c.PressRune(rune('0' + (i % 3)))
				time.Sleep(20 * time.Millisecond)
			}
		},
		// Navigation goroutine
		func(c *TUITestContext) {
			for i := 0; i < 20; i++ {
				c.PressRune('j')
				time.Sleep(10 * time.Millisecond)
			}
		},
		// Filter goroutine
		func(c *TUITestContext) {
			for i := 0; i < 10; i++ {
				c.PressRune('/').Type("test").Press(tcell.KeyEnter)
				time.Sleep(30 * time.Millisecond)
			}
		},
	)
}

// TestStress_RapidDrawCalls tests rapid concurrent QueueUpdateDraw calls
func TestStress_RapidDrawCalls(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				ctx.app.QueueUpdateDraw(func() {
					// Force redraw cycle
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
		// Success
	case <-time.After(10 * time.Second):
		t.Fatal("Rapid QueueUpdateDraw() calls deadlocked")
	}
}

// TestStress_ConcurrentQueueUpdate tests concurrent QueueUpdate calls
func TestStress_ConcurrentQueueUpdate(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	var wg sync.WaitGroup
	callCount := 0
	var mu sync.Mutex

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				ctx.app.QueueUpdate(func() {
					mu.Lock()
					callCount++
					mu.Unlock()
				})
				time.Sleep(1 * time.Millisecond)
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
	case <-time.After(10 * time.Second):
		t.Fatal("Concurrent QueueUpdate calls deadlocked")
	}

	time.Sleep(200 * time.Millisecond) // Let queued updates complete

	mu.Lock()
	count := callCount
	mu.Unlock()

	if count == 0 {
		t.Error("Expected QueueUpdate callbacks to be called")
	}
}

// TestStress_HighFrequencyKeyPresses tests extremely rapid key presses
func TestStress_HighFrequencyKeyPresses(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	// Simulate 200 rapid key presses (j/k navigation)
	for i := 0; i < 200; i++ {
		if i%2 == 0 {
			ctx.PressRune('j')
		} else {
			ctx.PressRune('k')
		}
		time.Sleep(2 * time.Millisecond)
	}

	ctx.ExpectNoFreeze()
}

// TestStress_RapidCommandMode tests rapid command mode entry/exit
func TestStress_RapidCommandMode(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	for i := 0; i < 50; i++ {
		ctx.PressRune(':')
		time.Sleep(5 * time.Millisecond)
		ctx.Escape()
		time.Sleep(5 * time.Millisecond)
	}

	ctx.ExpectNoFreeze()
}

// TestStress_RapidHelpModal tests rapid help modal open/close
func TestStress_RapidHelpModal(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	for i := 0; i < 30; i++ {
		ctx.PressRune('?')
		time.Sleep(10 * time.Millisecond)
		ctx.Escape()
		time.Sleep(10 * time.Millisecond)
	}

	ctx.ExpectNoFreeze()
}

// TestStress_ConcurrentStateReads tests concurrent reads of app state via QueueUpdate
func TestStress_ConcurrentStateReads(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				ctx.app.QueueUpdate(func() {
					_ = ctx.app.currentResource
					_ = ctx.app.currentNamespace
					_ = ctx.app.filterText
				})
				time.Sleep(1 * time.Millisecond)
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
	case <-time.After(10 * time.Second):
		t.Fatal("Concurrent state reads deadlocked")
	}
}

// TestStress_TabNavigationRapid tests rapid tab key presses
func TestStress_TabNavigationRapid(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	for i := 0; i < 100; i++ {
		ctx.Tab()
		time.Sleep(5 * time.Millisecond)
	}

	ctx.ExpectNoFreeze()
}

// TestStress_PageUpDownRapid tests rapid page up/down operations
func TestStress_PageUpDownRapid(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.Command("pods").Wait(100 * time.Millisecond)

	for i := 0; i < 50; i++ {
		ctx.Press(tcell.KeyPgUp)
		time.Sleep(5 * time.Millisecond)
		ctx.Press(tcell.KeyPgDn)
		time.Sleep(5 * time.Millisecond)
	}

	ctx.ExpectNoFreeze()
}
