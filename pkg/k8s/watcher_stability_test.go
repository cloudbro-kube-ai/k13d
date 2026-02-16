package k8s

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes/fake"
)

// ============================================================================
// Watcher Stability Tests
// Tests for watcher lifecycle, callback safety, and state transitions
// ============================================================================

// TestWatcherStability_StopPreventsOnChange tests that onChange is not called after Stop()
func TestWatcherStability_StopPreventsOnChange(t *testing.T) {
	t.Parallel()
	fakeClient := fake.NewSimpleClientset()
	client := &Client{Clientset: fakeClient}

	var callCount int32
	onChange := func() { atomic.AddInt32(&callCount, 1) }

	cfg := WatcherConfig{
		RelistInterval:   30 * time.Second,
		DebounceInterval: 50 * time.Millisecond,
		FallbackInterval: 5 * time.Second,
	}

	w := NewResourceWatcher(client, "pods", "default", onChange, testLogger(), cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.Start(ctx)

	// Wait for watcher to become active
	deadline := time.After(2 * time.Second)
	for w.State() != WatchStateActive {
		select {
		case <-deadline:
			t.Fatalf("watcher did not become active")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Record call count before stop
	beforeStop := atomic.LoadInt32(&callCount)

	// Stop the watcher
	w.Stop()

	// Wait longer than debounce interval
	time.Sleep(200 * time.Millisecond)

	// Call count should not have increased after Stop
	afterStop := atomic.LoadInt32(&callCount)
	if afterStop > beforeStop {
		t.Errorf("onChange called after Stop(): before=%d, after=%d", beforeStop, afterStop)
	}

	// Verify state is inactive
	if w.State() != WatchStateInactive {
		t.Errorf("Expected WatchStateInactive after Stop(), got %d", w.State())
	}
}

// TestWatcherStability_RapidStartStop tests rapid Start/Stop cycles
func TestWatcherStability_RapidStartStop(t *testing.T) {
	t.Parallel()
	fakeClient := fake.NewSimpleClientset()
	client := &Client{Clientset: fakeClient}

	onChange := func() {}

	cfg := WatcherConfig{
		RelistInterval:   30 * time.Second,
		DebounceInterval: 50 * time.Millisecond,
		FallbackInterval: 5 * time.Second,
	}

	w := NewResourceWatcher(client, "pods", "default", onChange, testLogger(), cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Rapidly start and stop
	for i := 0; i < 10; i++ {
		w = NewResourceWatcher(client, "pods", "default", onChange, testLogger(), cfg)
		w.Start(ctx)
		time.Sleep(20 * time.Millisecond)
		w.Stop()
		time.Sleep(10 * time.Millisecond)
	}

	// Final watcher should be stopped
	if w.State() != WatchStateInactive {
		t.Errorf("Expected WatchStateInactive after rapid cycles, got %d", w.State())
	}
}

// TestWatcherStability_ConcurrentStopCalls tests concurrent Stop() calls are safe
func TestWatcherStability_ConcurrentStopCalls(t *testing.T) {
	t.Parallel()
	fakeClient := fake.NewSimpleClientset()
	client := &Client{Clientset: fakeClient}

	onChange := func() {}

	cfg := WatcherConfig{
		RelistInterval:   30 * time.Second,
		DebounceInterval: 50 * time.Millisecond,
		FallbackInterval: 5 * time.Second,
	}

	w := NewResourceWatcher(client, "pods", "default", onChange, testLogger(), cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.Start(ctx)

	// Wait for active state
	deadline := time.After(2 * time.Second)
	for w.State() != WatchStateActive {
		select {
		case <-deadline:
			t.Fatalf("watcher did not become active")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Call Stop() concurrently
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.Stop()
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
		t.Fatal("Concurrent Stop() calls deadlocked")
	}

	// Verify final state
	if w.State() != WatchStateInactive {
		t.Errorf("Expected WatchStateInactive after concurrent stops, got %d", w.State())
	}
}

// TestWatcherStability_ContextCancelStopsWatcher tests context cancellation stops watcher cleanly
func TestWatcherStability_ContextCancelStopsWatcher(t *testing.T) {
	t.Parallel()
	fakeClient := fake.NewSimpleClientset()
	client := &Client{Clientset: fakeClient}

	onChange := func() {}

	cfg := WatcherConfig{
		RelistInterval:   30 * time.Second,
		DebounceInterval: 50 * time.Millisecond,
		FallbackInterval: 5 * time.Second,
	}

	w := NewResourceWatcher(client, "pods", "default", onChange, testLogger(), cfg)

	ctx, cancel := context.WithCancel(context.Background())

	w.Start(ctx)

	// Wait for active state
	deadline := time.After(2 * time.Second)
	for w.State() != WatchStateActive {
		select {
		case <-deadline:
			t.Fatalf("watcher did not become active")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Cancel context
	cancel()

	// Wait for goroutine to exit
	time.Sleep(200 * time.Millisecond)

	// Stop should still work after context cancel
	w.Stop()

	if w.State() != WatchStateInactive {
		t.Errorf("Expected WatchStateInactive after context cancel + Stop(), got %d", w.State())
	}
}

// TestWatcherStability_WatchToFallbackTransition tests transition from Watch to Fallback mode
func TestWatcherStability_WatchToFallbackTransition(t *testing.T) {
	t.Parallel()
	fakeClient := fake.NewSimpleClientset()
	client := &Client{Clientset: fakeClient}

	onChange := func() {}

	cfg := WatcherConfig{
		RelistInterval:   5 * time.Second,
		DebounceInterval: 50 * time.Millisecond,
		FallbackInterval: 100 * time.Millisecond,
	}

	// Use unsupported resource to force fallback
	w := NewResourceWatcher(client, "unsupported-resource", "default", onChange, testLogger(), cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.Start(ctx)

	// Wait for fallback state
	deadline := time.After(3 * time.Second)
	for w.State() != WatchStateFallback {
		select {
		case <-deadline:
			t.Fatalf("watcher did not enter fallback state, state=%d", w.State())
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	if w.State() != WatchStateFallback {
		t.Errorf("Expected WatchStateFallback, got %d", w.State())
	}

	w.Stop()
}

// TestWatcherStability_FallbackModePolling tests that fallback mode continues polling
func TestWatcherStability_FallbackModePolling(t *testing.T) {
	t.Parallel()
	fakeClient := fake.NewSimpleClientset()
	client := &Client{Clientset: fakeClient}

	var callCount int32
	onChange := func() { atomic.AddInt32(&callCount, 1) }

	cfg := WatcherConfig{
		RelistInterval:   5 * time.Second,
		DebounceInterval: 50 * time.Millisecond,
		FallbackInterval: 100 * time.Millisecond, // Fast polling for test
	}

	// Use unsupported resource to force fallback
	w := NewResourceWatcher(client, "unsupported-resource", "default", onChange, testLogger(), cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.Start(ctx)

	// Wait for fallback state
	deadline := time.After(2 * time.Second)
	for w.State() != WatchStateFallback {
		select {
		case <-deadline:
			t.Fatalf("watcher did not enter fallback state")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Wait for multiple polling cycles
	time.Sleep(400 * time.Millisecond)

	count := atomic.LoadInt32(&callCount)
	if count < 2 {
		t.Errorf("Expected at least 2 polling calls in fallback mode, got %d", count)
	}

	w.Stop()
}

// TestWatcherStability_StopDuringDebounce tests stopping during debounce window
func TestWatcherStability_StopDuringDebounce(t *testing.T) {
	t.Parallel()
	fakeClient := fake.NewSimpleClientset()
	client := &Client{Clientset: fakeClient}

	var callCount int32
	onChange := func() { atomic.AddInt32(&callCount, 1) }

	cfg := WatcherConfig{
		RelistInterval:   30 * time.Second,
		DebounceInterval: 500 * time.Millisecond, // Longer debounce
		FallbackInterval: 5 * time.Second,
	}

	w := NewResourceWatcher(client, "pods", "default", onChange, testLogger(), cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.Start(ctx)

	// Wait for active state
	deadline := time.After(2 * time.Second)
	for w.State() != WatchStateActive {
		select {
		case <-deadline:
			t.Fatalf("watcher did not become active")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Record count before stop (during debounce window)
	time.Sleep(100 * time.Millisecond) // Start debounce but don't let it fire
	beforeStop := atomic.LoadInt32(&callCount)

	// Stop during debounce
	w.Stop()

	// Wait for debounce interval to expire
	time.Sleep(600 * time.Millisecond)

	// onChange should not have been called after Stop (debounce was canceled)
	afterStop := atomic.LoadInt32(&callCount)
	if afterStop > beforeStop {
		t.Errorf("onChange called after Stop() during debounce: before=%d, after=%d", beforeStop, afterStop)
	}
}

// TestWatcherStability_RelistCycle tests periodic relist functionality
func TestWatcherStability_RelistCycle(t *testing.T) {
	t.Parallel()
	fakeClient := fake.NewSimpleClientset()
	client := &Client{Clientset: fakeClient}

	var callCount int32
	onChange := func() { atomic.AddInt32(&callCount, 1) }

	cfg := WatcherConfig{
		RelistInterval:   200 * time.Millisecond, // Fast relist for test
		DebounceInterval: 50 * time.Millisecond,
		FallbackInterval: 5 * time.Second,
	}

	w := NewResourceWatcher(client, "pods", "default", onChange, testLogger(), cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.Start(ctx)

	// Wait for active state
	deadline := time.After(2 * time.Second)
	for w.State() != WatchStateActive {
		select {
		case <-deadline:
			t.Fatalf("watcher did not become active")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Wait for multiple relist cycles
	time.Sleep(700 * time.Millisecond)

	count := atomic.LoadInt32(&callCount)
	if count < 2 {
		t.Errorf("Expected at least 2 relist-triggered onChange calls, got %d", count)
	}

	w.Stop()
}

// TestWatcherStability_StateThreadSafety tests concurrent State() reads are safe
func TestWatcherStability_StateThreadSafety(t *testing.T) {
	t.Parallel()
	fakeClient := fake.NewSimpleClientset()
	client := &Client{Clientset: fakeClient}

	onChange := func() {}

	cfg := DefaultWatcherConfig()

	w := NewResourceWatcher(client, "pods", "default", onChange, testLogger(), cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.Start(ctx)

	// Concurrently read state
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = w.State()
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
		t.Fatal("Concurrent State() reads deadlocked")
	}

	w.Stop()
}

// TestWatcherStability_ConcurrentStateChanges tests concurrent state changes
func TestWatcherStability_ConcurrentStateChanges(t *testing.T) {
	t.Parallel()
	fakeClient := fake.NewSimpleClientset()
	client := &Client{Clientset: fakeClient}

	onChange := func() {}

	cfg := DefaultWatcherConfig()

	w := NewResourceWatcher(client, "pods", "default", onChange, testLogger(), cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Start goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		w.Start(ctx)
	}()

	// State read goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_ = w.State()
			time.Sleep(5 * time.Millisecond)
		}
	}()

	// Wait for active
	time.Sleep(500 * time.Millisecond)

	// Stop goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		w.Stop()
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Concurrent state changes deadlocked")
	}
}

// TestWatcherStability_OnChangeNotCalledAfterContextCancel tests onChange is not called after context cancel
func TestWatcherStability_OnChangeNotCalledAfterContextCancel(t *testing.T) {
	t.Parallel()
	fakeClient := fake.NewSimpleClientset()
	client := &Client{Clientset: fakeClient}

	var callCount int32
	onChange := func() { atomic.AddInt32(&callCount, 1) }

	cfg := WatcherConfig{
		RelistInterval:   30 * time.Second,
		DebounceInterval: 50 * time.Millisecond,
		FallbackInterval: 5 * time.Second,
	}

	w := NewResourceWatcher(client, "pods", "default", onChange, testLogger(), cfg)

	ctx, cancel := context.WithCancel(context.Background())

	w.Start(ctx)

	// Wait for active state
	deadline := time.After(2 * time.Second)
	for w.State() != WatchStateActive {
		select {
		case <-deadline:
			t.Fatalf("watcher did not become active")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Record count before cancel
	beforeCancel := atomic.LoadInt32(&callCount)

	// Cancel context
	cancel()

	// Wait for debounce + margin
	time.Sleep(200 * time.Millisecond)

	// onChange should not have been called after context cancel
	afterCancel := atomic.LoadInt32(&callCount)
	if afterCancel > beforeCancel {
		t.Errorf("onChange called after context cancel: before=%d, after=%d", beforeCancel, afterCancel)
	}

	w.Stop()
}
