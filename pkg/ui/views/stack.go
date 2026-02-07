package views

import (
	"sync"

	"github.com/rivo/tview"
)

// StackListener is notified of stack changes.
type StackListener interface {
	StackPushed(view View)
	StackPopped(view View)
	StackTop(view View)
}

// PageStack manages a stack of views with lifecycle coordination.
type PageStack struct {
	*tview.Pages
	app       *tview.Application
	stack     []View
	listeners []StackListener
	mx        sync.RWMutex
}

// NewPageStack creates a new PageStack.
func NewPageStack(app *tview.Application) *PageStack {
	return &PageStack{
		Pages:     tview.NewPages(),
		app:       app,
		stack:     make([]View, 0),
		listeners: make([]StackListener, 0),
	}
}

// Push pushes a view onto the stack.
func (ps *PageStack) Push(view View) {
	ps.mx.Lock()
	defer ps.mx.Unlock()

	// Stop current top view
	if len(ps.stack) > 0 {
		current := ps.stack[len(ps.stack)-1]
		current.Stop()
	}

	// Add new view
	ps.stack = append(ps.stack, view)
	ps.Pages.AddPage(view.Name(), view.Primitive(), true, true)

	// Start new view
	view.Start()
	view.SetFocus(ps.app)

	// Notify listeners
	for _, l := range ps.listeners {
		l.StackPushed(view)
		l.StackTop(view)
	}
}

// Pop pops the top view from the stack.
func (ps *PageStack) Pop() View {
	ps.mx.Lock()
	defer ps.mx.Unlock()

	if len(ps.stack) == 0 {
		return nil
	}

	// Stop and remove top view
	top := ps.stack[len(ps.stack)-1]
	top.Stop()
	ps.stack = ps.stack[:len(ps.stack)-1]
	ps.Pages.RemovePage(top.Name())

	// Notify listeners
	for _, l := range ps.listeners {
		l.StackPopped(top)
	}

	// Resume previous view
	if len(ps.stack) > 0 {
		current := ps.stack[len(ps.stack)-1]
		ps.Pages.SwitchToPage(current.Name())
		current.Start()
		current.SetFocus(ps.app)

		for _, l := range ps.listeners {
			l.StackTop(current)
		}
	}

	return top
}

// Top returns the top view without removing it.
func (ps *PageStack) Top() View {
	ps.mx.RLock()
	defer ps.mx.RUnlock()

	if len(ps.stack) == 0 {
		return nil
	}
	return ps.stack[len(ps.stack)-1]
}

// Len returns the stack depth.
func (ps *PageStack) Len() int {
	ps.mx.RLock()
	defer ps.mx.RUnlock()
	return len(ps.stack)
}

// Clear removes all views from the stack.
func (ps *PageStack) Clear() {
	ps.mx.Lock()
	defer ps.mx.Unlock()

	for _, v := range ps.stack {
		v.Stop()
		ps.Pages.RemovePage(v.Name())
	}
	ps.stack = make([]View, 0)
}

// Breadcrumbs returns the view names as breadcrumbs.
func (ps *PageStack) Breadcrumbs() []string {
	ps.mx.RLock()
	defer ps.mx.RUnlock()

	crumbs := make([]string, len(ps.stack))
	for i, v := range ps.stack {
		crumbs[i] = v.Name()
	}
	return crumbs
}

// Subscribe adds a stack listener.
func (ps *PageStack) Subscribe(listener StackListener) {
	ps.mx.Lock()
	defer ps.mx.Unlock()
	ps.listeners = append(ps.listeners, listener)
}

// Unsubscribe removes a stack listener.
func (ps *PageStack) Unsubscribe(listener StackListener) {
	ps.mx.Lock()
	defer ps.mx.Unlock()
	for i, l := range ps.listeners {
		if l == listener {
			ps.listeners = append(ps.listeners[:i], ps.listeners[i+1:]...)
			return
		}
	}
}

// CanGoBack returns true if there's a view to go back to.
func (ps *PageStack) CanGoBack() bool {
	ps.mx.RLock()
	defer ps.mx.RUnlock()
	return len(ps.stack) > 1
}

// GoBack pops the top view and returns to the previous one.
func (ps *PageStack) GoBack() bool {
	if !ps.CanGoBack() {
		return false
	}
	ps.Pop()
	return true
}

// Replace replaces the top view with a new view.
func (ps *PageStack) Replace(view View) {
	ps.mx.Lock()
	defer ps.mx.Unlock()

	// Stop and remove current top
	if len(ps.stack) > 0 {
		top := ps.stack[len(ps.stack)-1]
		top.Stop()
		ps.Pages.RemovePage(top.Name())
		ps.stack = ps.stack[:len(ps.stack)-1]
	}

	// Inline push logic to avoid releasing and re-acquiring the lock
	ps.stack = append(ps.stack, view)
	ps.Pages.AddPage(view.Name(), view.Primitive(), true, true)

	view.Start()
	view.SetFocus(ps.app)

	for _, l := range ps.listeners {
		l.StackPushed(view)
		l.StackTop(view)
	}
}

// SwitchTo switches to a specific view by name.
// If the view exists in the stack, it becomes the top.
// If not found, returns false.
func (ps *PageStack) SwitchTo(name string) bool {
	ps.mx.Lock()
	defer ps.mx.Unlock()

	for i, v := range ps.stack {
		if v.Name() == name {
			// Stop all views above this one
			for j := len(ps.stack) - 1; j > i; j-- {
				ps.stack[j].Stop()
				ps.Pages.RemovePage(ps.stack[j].Name())
			}
			ps.stack = ps.stack[:i+1]

			// Resume this view
			ps.Pages.SwitchToPage(name)
			v.Start()
			v.SetFocus(ps.app)

			for _, l := range ps.listeners {
				l.StackTop(v)
			}
			return true
		}
	}
	return false
}
