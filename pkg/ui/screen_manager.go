// Package ui provides the terminal user interface components.
package ui

import (
	"context"
	"sync"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/ui/views"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ScreenState represents the current state of the TUI screen.
type ScreenState string

const (
	ScreenStateInit    ScreenState = "init"
	ScreenStateRunning ScreenState = "running"
	ScreenStateStopped ScreenState = "stopped"
)

// ScreenManager coordinates screen management between App and views.
// It provides a unified interface for managing the TUI lifecycle,
// integrating both the legacy App-based UI and the new k9s-style views.
type ScreenManager struct {
	app       *tview.Application
	screen    tcell.Screen
	pageStack *views.PageStack

	// State management
	state     ScreenState
	stateMx   sync.RWMutex
	initOnce  sync.Once
	closeOnce sync.Once

	// Callbacks
	onStart   func()
	onStop    func()
	onError   func(error)
	onRefresh func()

	// Lifecycle control
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// ScreenManagerConfig holds configuration for ScreenManager.
type ScreenManagerConfig struct {
	// UseSimulationScreen enables headless testing mode.
	UseSimulationScreen bool
	// Screen is the optional screen to use (for testing).
	Screen tcell.Screen
	// InitialWidth sets the initial screen width.
	InitialWidth int
	// InitialHeight sets the initial screen height.
	InitialHeight int
}

// NewScreenManager creates a new ScreenManager with optional config.
func NewScreenManager(cfg *ScreenManagerConfig) *ScreenManager {
	if cfg == nil {
		cfg = &ScreenManagerConfig{}
	}

	ctx, cancel := context.WithCancel(context.Background())

	sm := &ScreenManager{
		app:    tview.NewApplication(),
		state:  ScreenStateInit,
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	// Configure screen
	if cfg.UseSimulationScreen && cfg.Screen != nil {
		sm.screen = cfg.Screen
		sm.app.SetScreen(cfg.Screen)
	}

	// Create page stack
	sm.pageStack = views.NewPageStack(sm.app)

	return sm
}

// App returns the underlying tview.Application.
func (sm *ScreenManager) App() *tview.Application {
	return sm.app
}

// Screen returns the underlying tcell.Screen.
func (sm *ScreenManager) Screen() tcell.Screen {
	return sm.screen
}

// PageStack returns the view page stack.
func (sm *ScreenManager) PageStack() *views.PageStack {
	return sm.pageStack
}

// State returns the current screen state.
func (sm *ScreenManager) State() ScreenState {
	sm.stateMx.RLock()
	defer sm.stateMx.RUnlock()
	return sm.state
}

// setState updates the screen state.
func (sm *ScreenManager) setState(s ScreenState) {
	sm.stateMx.Lock()
	defer sm.stateMx.Unlock()
	sm.state = s
}

// IsRunning returns true if the screen manager is running.
func (sm *ScreenManager) IsRunning() bool {
	return sm.State() == ScreenStateRunning
}

// SetRoot sets the root primitive for the application.
func (sm *ScreenManager) SetRoot(root tview.Primitive, fullscreen bool) *ScreenManager {
	sm.app.SetRoot(root, fullscreen)
	return sm
}

// SetOnStart sets the callback to be called when the screen starts.
func (sm *ScreenManager) SetOnStart(fn func()) *ScreenManager {
	sm.onStart = fn
	return sm
}

// SetOnStop sets the callback to be called when the screen stops.
func (sm *ScreenManager) SetOnStop(fn func()) *ScreenManager {
	sm.onStop = fn
	return sm
}

// SetOnError sets the callback to be called on errors.
func (sm *ScreenManager) SetOnError(fn func(error)) *ScreenManager {
	sm.onError = fn
	return sm
}

// SetOnRefresh sets the callback for refresh requests.
func (sm *ScreenManager) SetOnRefresh(fn func()) *ScreenManager {
	sm.onRefresh = fn
	return sm
}

// Run starts the TUI application (blocking).
func (sm *ScreenManager) Run() error {
	sm.initOnce.Do(func() {
		sm.setState(ScreenStateRunning)
		if sm.onStart != nil {
			sm.onStart()
		}
	})

	err := sm.app.Run()

	sm.closeOnce.Do(func() {
		sm.setState(ScreenStateStopped)
		sm.cancel()
		close(sm.done)
		if sm.onStop != nil {
			sm.onStop()
		}
	})

	return err
}

// RunAsync starts the TUI application in a goroutine.
// Returns a cleanup function that stops the application.
func (sm *ScreenManager) RunAsync() func() {
	ready := make(chan struct{})
	origOnStart := sm.onStart
	sm.onStart = func() {
		if origOnStart != nil {
			origOnStart()
		}
		close(ready)
	}

	go func() {
		_ = sm.Run()
	}()

	// Wait for initialization with timeout fallback
	select {
	case <-ready:
	case <-time.After(2 * time.Second):
	}

	return func() {
		sm.Stop()
	}
}

// Stop stops the TUI application.
func (sm *ScreenManager) Stop() {
	if !sm.IsRunning() {
		return
	}
	sm.app.Stop()

	// Wait for clean shutdown
	select {
	case <-sm.done:
	case <-time.After(2 * time.Second):
		// Timeout, force close
	}
}

// QueueUpdateDraw queues a UI update safely.
// tview internally serializes QueueUpdateDraw calls, so no additional
// atomic guard is needed. The previous atomic-based batching caused
// deadlocks when Draw() was called inside a QueueUpdate handler.
func (sm *ScreenManager) QueueUpdateDraw(f func()) {
	if sm.app == nil || !sm.IsRunning() {
		return
	}
	sm.app.QueueUpdateDraw(f)
}

// Draw forces a screen redraw.
func (sm *ScreenManager) Draw() {
	if sm.app != nil && sm.IsRunning() {
		sm.app.Draw()
	}
}

// SetFocus sets focus to the specified primitive.
func (sm *ScreenManager) SetFocus(p tview.Primitive) {
	if sm.app != nil && sm.IsRunning() {
		sm.app.SetFocus(p)
	}
}

// GetFocus returns the currently focused primitive.
func (sm *ScreenManager) GetFocus() tview.Primitive {
	if sm.app == nil {
		return nil
	}
	return sm.app.GetFocus()
}

// PushView pushes a new view onto the page stack.
func (sm *ScreenManager) PushView(view views.View) {
	if sm.pageStack != nil {
		sm.pageStack.Push(view)
	}
}

// PopView pops the top view from the page stack.
func (sm *ScreenManager) PopView() views.View {
	if sm.pageStack == nil {
		return nil
	}
	return sm.pageStack.Pop()
}

// GoBack navigates back in the view stack.
func (sm *ScreenManager) GoBack() bool {
	if sm.pageStack == nil {
		return false
	}
	return sm.pageStack.GoBack()
}

// CurrentView returns the current top view.
func (sm *ScreenManager) CurrentView() views.View {
	if sm.pageStack == nil {
		return nil
	}
	return sm.pageStack.Top()
}

// Context returns the screen manager's context.
func (sm *ScreenManager) Context() context.Context {
	return sm.ctx
}

// Done returns a channel that is closed when the screen manager stops.
func (sm *ScreenManager) Done() <-chan struct{} {
	return sm.done
}

// Refresh triggers a refresh of the current view.
func (sm *ScreenManager) Refresh() {
	if sm.onRefresh != nil {
		go sm.onRefresh()
	}
}

// Size returns the current screen size.
func (sm *ScreenManager) Size() (width, height int) {
	if sm.screen != nil {
		return sm.screen.Size()
	}
	// Default size for non-simulation screens
	return 80, 24
}

// Resize changes the simulated screen size (for testing).
func (sm *ScreenManager) Resize(width, height int) {
	if simScreen, ok := sm.screen.(tcell.SimulationScreen); ok {
		simScreen.SetSize(width, height)
	}
}
