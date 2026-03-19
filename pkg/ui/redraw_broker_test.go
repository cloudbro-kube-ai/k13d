package ui

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestRedrawBrokerCoalescesBurst(t *testing.T) {
	var flushCount int32
	var callbackCount int32
	done := make(chan struct{})

	broker := newRedrawBroker(5*time.Millisecond, func(f func()) {
		atomic.AddInt32(&flushCount, 1)
		f()
	})
	defer broker.Stop()

	for i := 0; i < 10; i++ {
		broker.Schedule(func() {
			if atomic.AddInt32(&callbackCount, 1) == 10 {
				close(done)
			}
		})
	}

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for coalesced redraw callbacks")
	}

	time.Sleep(20 * time.Millisecond)

	if got := atomic.LoadInt32(&flushCount); got != 1 {
		t.Fatalf("expected 1 redraw flush for burst, got %d", got)
	}

	if got := atomic.LoadInt32(&callbackCount); got != 10 {
		t.Fatalf("expected 10 callbacks to execute, got %d", got)
	}
}

func TestRedrawBrokerSeparatesDistinctBatches(t *testing.T) {
	var flushCount int32
	var callbackCount int32

	broker := newRedrawBroker(5*time.Millisecond, func(f func()) {
		atomic.AddInt32(&flushCount, 1)
		f()
	})
	defer broker.Stop()

	broker.Schedule(func() {
		atomic.AddInt32(&callbackCount, 1)
	})

	time.Sleep(25 * time.Millisecond)

	broker.Schedule(func() {
		atomic.AddInt32(&callbackCount, 1)
	})

	time.Sleep(25 * time.Millisecond)

	if got := atomic.LoadInt32(&flushCount); got != 2 {
		t.Fatalf("expected 2 redraw flushes for separated batches, got %d", got)
	}

	if got := atomic.LoadInt32(&callbackCount); got != 2 {
		t.Fatalf("expected 2 callbacks to execute, got %d", got)
	}
}
