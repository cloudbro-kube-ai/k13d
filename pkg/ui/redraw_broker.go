package ui

import (
	"sync"
	"sync/atomic"
	"time"
)

const defaultRedrawBatchWindow = 16 * time.Millisecond

// redrawBroker coalesces many redraw requests into a single QueueUpdateDraw call.
// This reduces visible flicker and avoids flooding tview with redraw requests from
// multiple asynchronous producers such as spinners, AI streaming, and flash messages.
type redrawBroker struct {
	batchWindow time.Duration
	flushFn     func(func())

	mu      sync.Mutex
	pending []func()

	signal   chan struct{}
	stopCh   chan struct{}
	stopOnce sync.Once
	stopped  int32
}

func newRedrawBroker(batchWindow time.Duration, flushFn func(func())) *redrawBroker {
	if batchWindow <= 0 {
		batchWindow = defaultRedrawBatchWindow
	}

	b := &redrawBroker{
		batchWindow: batchWindow,
		flushFn:     flushFn,
		signal:      make(chan struct{}, 1),
		stopCh:      make(chan struct{}),
	}

	go b.run()

	return b
}

func (b *redrawBroker) Schedule(f func()) {
	if f == nil || atomic.LoadInt32(&b.stopped) == 1 {
		return
	}

	b.mu.Lock()
	if atomic.LoadInt32(&b.stopped) == 1 {
		b.mu.Unlock()
		return
	}
	b.pending = append(b.pending, f)
	b.mu.Unlock()

	select {
	case b.signal <- struct{}{}:
	default:
	}
}

func (b *redrawBroker) Stop() {
	b.stopOnce.Do(func() {
		atomic.StoreInt32(&b.stopped, 1)
		close(b.stopCh)
	})
}

func (b *redrawBroker) run() {
	for {
		select {
		case <-b.stopCh:
			return
		case <-b.signal:
		}

		timer := time.NewTimer(b.batchWindow)
	collect:
		for {
			select {
			case <-b.stopCh:
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				return
			case <-b.signal:
				// Keep collecting until the batch window expires.
			case <-timer.C:
				break collect
			}
		}

		callbacks := b.takePending()
		if len(callbacks) == 0 || b.flushFn == nil || atomic.LoadInt32(&b.stopped) == 1 {
			continue
		}

		b.flushFn(func() {
			for _, cb := range callbacks {
				if cb != nil {
					cb()
				}
			}
		})
	}
}

func (b *redrawBroker) takePending() []func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.pending) == 0 {
		return nil
	}

	callbacks := b.pending
	b.pending = nil
	return callbacks
}
