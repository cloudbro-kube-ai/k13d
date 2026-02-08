package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/watch"
)

// WatchState represents the current state of the watcher.
type WatchState int

const (
	WatchStateInactive WatchState = iota
	WatchStateActive
	WatchStateFallback // Polling mode after watch failure
)

// WatcherConfig holds watcher configuration.
type WatcherConfig struct {
	RelistInterval   time.Duration // How often to do a full re-list (default: 30s)
	DebounceInterval time.Duration // Debounce window for watch events (default: 250ms)
	FallbackInterval time.Duration // Polling interval when watch fails (default: 5s)
}

// DefaultWatcherConfig returns sensible defaults.
func DefaultWatcherConfig() WatcherConfig {
	return WatcherConfig{
		RelistInterval:   30 * time.Second,
		DebounceInterval: 250 * time.Millisecond,
		FallbackInterval: 5 * time.Second,
	}
}

// ResourceWatcher watches a Kubernetes resource and notifies on changes.
// It implements the hybrid pattern: Watch API for delta events + periodic
// full re-list for consistency.
type ResourceWatcher struct {
	client    *Client
	resource  string // e.g., "pods", "deployments"
	namespace string // "" for all namespaces

	state   WatchState
	stateMu sync.RWMutex

	onChange func() // Callback when data changes (triggers refresh)

	stopCh  chan struct{}
	stopped bool
	stopMu  sync.Mutex

	logger *slog.Logger

	cfg WatcherConfig
}

// NewResourceWatcher creates a new ResourceWatcher.
func NewResourceWatcher(client *Client, resource, namespace string,
	onChange func(), logger *slog.Logger, cfg WatcherConfig) *ResourceWatcher {

	if cfg.RelistInterval == 0 {
		cfg.RelistInterval = 30 * time.Second
	}
	if cfg.DebounceInterval == 0 {
		cfg.DebounceInterval = 250 * time.Millisecond
	}
	if cfg.FallbackInterval == 0 {
		cfg.FallbackInterval = 5 * time.Second
	}

	return &ResourceWatcher{
		client:    client,
		resource:  resource,
		namespace: namespace,
		onChange:  onChange,
		stopCh:    make(chan struct{}),
		logger:    logger,
		cfg:       cfg,
	}
}

// Start begins the watch loop in a goroutine.
func (w *ResourceWatcher) Start(ctx context.Context) {
	go w.run(ctx)
}

// Stop stops the watcher. Safe to call multiple times.
func (w *ResourceWatcher) Stop() {
	w.stopMu.Lock()
	defer w.stopMu.Unlock()
	if !w.stopped {
		w.stopped = true
		close(w.stopCh)
		w.setState(WatchStateInactive)
	}
}

// State returns the current watch state.
func (w *ResourceWatcher) State() WatchState {
	w.stateMu.RLock()
	defer w.stateMu.RUnlock()
	return w.state
}

func (w *ResourceWatcher) setState(s WatchState) {
	w.stateMu.Lock()
	w.state = s
	w.stateMu.Unlock()
}

func (w *ResourceWatcher) isStopped() bool {
	select {
	case <-w.stopCh:
		return true
	default:
		return false
	}
}

// run is the main loop that alternates between watch and fallback polling.
func (w *ResourceWatcher) run(ctx context.Context) {
	for {
		if w.isStopped() || ctx.Err() != nil {
			return
		}

		err := w.watchLoop(ctx)
		if err != nil {
			if w.isStopped() || ctx.Err() != nil {
				return
			}
			w.logger.Warn("Watch failed, falling back to polling",
				"resource", w.resource, "error", err)
			w.setState(WatchStateFallback)

			// Fallback: poll until we can retry watch
			w.pollLoop(ctx)
		}
	}
}

// watchLoop runs the Watch API loop with debouncing and periodic re-list.
func (w *ResourceWatcher) watchLoop(ctx context.Context) error {
	watcher, err := w.client.WatchResource(ctx, w.resource, w.namespace)
	if err != nil {
		return fmt.Errorf("failed to start watch: %w", err)
	}
	defer watcher.Stop()

	w.setState(WatchStateActive)

	// Debounce timer to coalesce rapid events
	var debounceTimer *time.Timer
	debounceCh := make(chan struct{}, 1)

	defer func() {
		if debounceTimer != nil {
			debounceTimer.Stop()
		}
	}()

	// Periodic re-list timer
	relistTicker := time.NewTicker(w.cfg.RelistInterval)
	defer relistTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.stopCh:
			return nil

		case event, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("watch channel closed")
			}
			if event.Type == watch.Error {
				return fmt.Errorf("watch error event received")
			}

			// Debounce: reset timer on each event
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(w.cfg.DebounceInterval, func() {
				select {
				case debounceCh <- struct{}{}:
				default: // Already pending
				}
			})

		case <-debounceCh:
			if w.onChange != nil {
				w.onChange()
			}

		case <-relistTicker.C:
			if w.onChange != nil {
				w.onChange()
			}
		}
	}
}

// pollLoop provides fallback polling when Watch is unavailable.
// Returns periodically to allow the outer loop to retry Watch.
func (w *ResourceWatcher) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.FallbackInterval)
	defer ticker.Stop()

	// Track iterations to know when to retry watch
	iterations := 0
	maxIterations := int(w.cfg.RelistInterval / w.cfg.FallbackInterval)
	if maxIterations < 1 {
		maxIterations = 1
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			if w.onChange != nil {
				w.onChange()
			}
			iterations++
			// After enough polls, return to retry watch
			if iterations >= maxIterations {
				return
			}
		}
	}
}
