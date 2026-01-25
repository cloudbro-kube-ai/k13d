package views

import (
	"context"
	"strings"
	"testing"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ui/actions"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ui/render"
	"github.com/rivo/tview"
)

// testView is a minimal View implementation for testing
type testView struct {
	name      string
	started   bool
	stopped   bool
	primitive tview.Primitive
}

func (v *testView) Name() string                 { return v.name }
func (v *testView) Init(_ context.Context) error { return nil }
func (v *testView) Start()                       { v.started = true }
func (v *testView) Stop()                        { v.stopped = true }
func (v *testView) Actions() *actions.KeyActions { return actions.NewKeyActions() }
func (v *testView) Primitive() tview.Primitive {
	if v.primitive == nil {
		v.primitive = tview.NewBox()
	}
	return v.primitive
}
func (v *testView) SetFocus(_ *tview.Application) {}

func TestNewPageStack(t *testing.T) {
	stack := NewPageStack(nil)
	if stack == nil {
		t.Fatal("NewPageStack returned nil")
	}
	if stack.Len() != 0 {
		t.Errorf("Len() = %d, want 0", stack.Len())
	}
}

func TestPageStackPushPop(t *testing.T) {
	stack := NewPageStack(nil)

	view := &testView{name: "test-view"}
	stack.Push(view)

	if stack.Len() != 1 {
		t.Errorf("Len() after push = %d, want 1", stack.Len())
	}

	current := stack.Top()
	if current == nil {
		t.Fatal("Top() returned nil")
	}
	if current.Name() != "test-view" {
		t.Errorf("Top().Name() = %s, want test-view", current.Name())
	}

	// Verify Start() was called
	if !view.started {
		t.Error("View.Start() was not called on push")
	}

	popped := stack.Pop()
	if popped == nil {
		t.Fatal("Pop() returned nil")
	}
	if stack.Len() != 0 {
		t.Errorf("Len() after pop = %d, want 0", stack.Len())
	}

	// Verify Stop() was called
	if !view.stopped {
		t.Error("View.Stop() was not called on pop")
	}
}

func TestPageStackGoBack(t *testing.T) {
	stack := NewPageStack(nil)

	view1 := &testView{name: "view1"}
	view2 := &testView{name: "view2"}
	stack.Push(view1)
	stack.Push(view2)

	// view1 should have been stopped when view2 was pushed
	if !view1.stopped {
		t.Error("View1 should be stopped when View2 is pushed")
	}

	ok := stack.GoBack()
	if !ok {
		t.Error("GoBack() should return true")
	}

	if stack.Len() != 1 {
		t.Errorf("Len() after GoBack = %d, want 1", stack.Len())
	}

	current := stack.Top()
	if current.Name() != "view1" {
		t.Errorf("Top().Name() = %s, want view1", current.Name())
	}
}

func TestPageStackBreadcrumbs(t *testing.T) {
	stack := NewPageStack(nil)

	stack.Push(&testView{name: "pods"})
	stack.Push(&testView{name: "pod-details"})
	stack.Push(&testView{name: "logs"})

	breadcrumbs := stack.Breadcrumbs()
	if len(breadcrumbs) != 3 {
		t.Errorf("Breadcrumbs() len = %d, want 3", len(breadcrumbs))
	}

	expected := "pods > pod-details > logs"
	actual := strings.Join(breadcrumbs, " > ")
	if actual != expected {
		t.Errorf("Breadcrumbs() = %s, want %s", actual, expected)
	}
}

func TestPageStackClear(t *testing.T) {
	stack := NewPageStack(nil)

	v1 := &testView{name: "view1"}
	v2 := &testView{name: "view2"}
	stack.Push(v1)
	stack.Push(v2)
	stack.Clear()

	if stack.Len() != 0 {
		t.Errorf("Len() after Clear = %d, want 0", stack.Len())
	}

	// Verify Stop() was called on both
	if !v1.stopped || !v2.stopped {
		t.Error("Stop() should be called on all views when clearing")
	}
}

func TestPageStackEmptyOperations(t *testing.T) {
	stack := NewPageStack(nil)

	// Pop on empty should return nil
	if stack.Pop() != nil {
		t.Error("Pop() on empty stack should return nil")
	}

	// GoBack on empty should return false
	if stack.GoBack() {
		t.Error("GoBack() on empty stack should return false")
	}

	// Top on empty should return nil
	if stack.Top() != nil {
		t.Error("Top() on empty stack should return nil")
	}

	// CanGoBack on empty should return false
	if stack.CanGoBack() {
		t.Error("CanGoBack() on empty stack should return false")
	}
}

func TestPageStackCanGoBack(t *testing.T) {
	stack := NewPageStack(nil)

	if stack.CanGoBack() {
		t.Error("CanGoBack() should be false on empty stack")
	}

	stack.Push(&testView{name: "view1"})
	if stack.CanGoBack() {
		t.Error("CanGoBack() should be false with only one view")
	}

	stack.Push(&testView{name: "view2"})
	if !stack.CanGoBack() {
		t.Error("CanGoBack() should be true with two views")
	}
}

type mockStackListener struct {
	pushCount  int
	popCount   int
	topCount   int
	lastPushed View
	lastPopped View
	lastTop    View
}

func (m *mockStackListener) StackPushed(v View) { m.pushCount++; m.lastPushed = v }
func (m *mockStackListener) StackPopped(v View) { m.popCount++; m.lastPopped = v }
func (m *mockStackListener) StackTop(v View)    { m.topCount++; m.lastTop = v }

func TestPageStackListener(t *testing.T) {
	stack := NewPageStack(nil)
	listener := &mockStackListener{}
	stack.Subscribe(listener)

	view := &testView{name: "test"}
	stack.Push(view)
	if listener.pushCount != 1 {
		t.Errorf("pushCount = %d, want 1", listener.pushCount)
	}
	if listener.topCount != 1 {
		t.Errorf("topCount = %d, want 1", listener.topCount)
	}
	if listener.lastPushed != view {
		t.Error("lastPushed should be the pushed view")
	}

	stack.Pop()
	if listener.popCount != 1 {
		t.Errorf("popCount = %d, want 1", listener.popCount)
	}
}

func TestPageStackRemoveListener(t *testing.T) {
	stack := NewPageStack(nil)
	listener := &mockStackListener{}
	stack.Subscribe(listener)
	stack.Unsubscribe(listener)

	stack.Push(&testView{name: "test"})
	if listener.pushCount != 0 {
		t.Error("Listener should not be called after unsubscribe")
	}
}

func TestPageStackReplace(t *testing.T) {
	stack := NewPageStack(nil)

	v1 := &testView{name: "view1"}
	v2 := &testView{name: "view2"}

	stack.Push(v1)
	stack.Replace(v2)

	if stack.Len() != 1 {
		t.Errorf("Len() after Replace = %d, want 1", stack.Len())
	}

	top := stack.Top()
	if top.Name() != "view2" {
		t.Errorf("Top().Name() = %s, want view2", top.Name())
	}

	if !v1.stopped {
		t.Error("Original view should be stopped after Replace")
	}
}

func TestPageStackSwitchTo(t *testing.T) {
	stack := NewPageStack(nil)

	v1 := &testView{name: "view1"}
	v2 := &testView{name: "view2"}
	v3 := &testView{name: "view3"}

	stack.Push(v1)
	stack.Push(v2)
	stack.Push(v3)

	ok := stack.SwitchTo("view1")
	if !ok {
		t.Error("SwitchTo(view1) should return true")
	}

	if stack.Len() != 1 {
		t.Errorf("Len() after SwitchTo = %d, want 1", stack.Len())
	}

	// Views 2 and 3 should be stopped
	if !v2.stopped || !v3.stopped {
		t.Error("Views above target should be stopped")
	}

	// Try switching to non-existent view
	ok = stack.SwitchTo("nonexistent")
	if ok {
		t.Error("SwitchTo(nonexistent) should return false")
	}
}

// TestRegistrar tests the Registrar implementation
func TestRegistrar(t *testing.T) {
	r := NewRegistrar()

	// Get should work for default viewers
	_, ok := r.Get(GVRPods)
	if !ok {
		t.Error("Get(GVRPods) should return true")
	}

	// Lookup should work with aliases
	gvr, ok := r.Lookup("po")
	if !ok {
		t.Error("Lookup(po) should return true")
	}
	if gvr != GVRPods {
		t.Errorf("Lookup(po) = %s, want %s", gvr, GVRPods)
	}

	// Lookup should work with full GVR
	gvr, ok = r.Lookup(string(GVRPods))
	if !ok {
		t.Error("Lookup(v1/pods) should return true")
	}
}

func TestRegistrarIsNamespaced(t *testing.T) {
	r := NewRegistrar()

	tests := []struct {
		gvr        GVR
		namespaced bool
	}{
		{GVRPods, true},
		{GVRDeployments, true},
		{GVRNodes, false},
		{GVRNamespaces, false},
		{GVRPersistentVolumes, false},
	}

	for _, tt := range tests {
		if r.IsNamespaced(tt.gvr) != tt.namespaced {
			t.Errorf("IsNamespaced(%s) = %v, want %v", tt.gvr, !tt.namespaced, tt.namespaced)
		}
	}
}

func TestRegistrarAllAliases(t *testing.T) {
	r := NewRegistrar()

	aliases := r.AllAliases()
	if len(aliases) == 0 {
		t.Error("AllAliases() should not be empty")
	}

	// Check that "po" is in aliases
	found := false
	for _, a := range aliases {
		if a == "po" {
			found = true
			break
		}
	}
	if !found {
		t.Error("AllAliases() should contain 'po'")
	}
}

func TestRegistrarAllGVRs(t *testing.T) {
	r := NewRegistrar()

	gvrs := r.AllGVRs()
	if len(gvrs) == 0 {
		t.Error("AllGVRs() should not be empty")
	}

	// Check that GVRPods is in gvrs
	found := false
	for _, g := range gvrs {
		if g == GVRPods {
			found = true
			break
		}
	}
	if !found {
		t.Error("AllGVRs() should contain GVRPods")
	}
}

func TestRegistrarGetRenderer(t *testing.T) {
	r := NewRegistrar()

	// Pods should have a renderer
	renderer := r.GetRenderer(GVRPods)
	if renderer == nil {
		t.Error("GetRenderer(GVRPods) should not be nil")
	}

	// Unknown GVR should return nil
	renderer = r.GetRenderer(GVR("unknown/unknown"))
	if renderer != nil {
		t.Error("GetRenderer(unknown) should be nil")
	}
}

func TestRegistrarGetActions(t *testing.T) {
	r := NewRegistrar()

	// Pods should have actions
	actns := r.GetActions(GVRPods)
	if actns == nil {
		t.Error("GetActions(GVRPods) should not be nil")
	}

	// Unknown GVR should return nil
	actns = r.GetActions(GVR("unknown/unknown"))
	if actns != nil {
		t.Error("GetActions(unknown) should be nil")
	}
}

// TestBaseView tests the BaseView implementation
func TestBaseView(t *testing.T) {
	bv := NewBaseView("test-base")

	if bv.Name() != "test-base" {
		t.Errorf("Name() = %s, want test-base", bv.Name())
	}

	// Actions should not be nil
	if bv.Actions() == nil {
		t.Error("Actions() returned nil")
	}

	// IsStopped should initially be false
	if bv.IsStopped() {
		t.Error("IsStopped() should be false initially")
	}
}

// TestResourceView tests the ResourceView implementation
func TestResourceView(t *testing.T) {
	renderer := render.NewBaseRenderer(render.Header{
		{Name: "NAME"},
		{Name: "STATUS"},
	})

	rv := NewResourceView("pods", "v1/pods", "default", renderer)

	if rv.Name() != "pods" {
		t.Errorf("Name() = %s, want pods", rv.Name())
	}

	// Should have resource view actions
	actns := rv.Actions()
	if actns == nil {
		t.Error("Actions() returned nil")
	}

	// Primitive should not be nil
	if rv.Primitive() == nil {
		t.Error("Primitive() should not be nil")
	}

	// Init should not return error
	if err := rv.Init(context.Background()); err != nil {
		t.Errorf("Init() returned error: %v", err)
	}
}

func TestResourceViewFilter(t *testing.T) {
	rv := NewResourceView("pods", "v1/pods", "default", nil)

	rv.SetFilter("nginx")
	if rv.GetFilter() != "nginx" {
		t.Errorf("GetFilter() = %s, want nginx", rv.GetFilter())
	}

	rv.ClearFilter()
	if rv.GetFilter() != "" {
		t.Error("GetFilter() should be empty after ClearFilter")
	}
}

func TestResourceViewSort(t *testing.T) {
	rv := NewResourceView("pods", "v1/pods", "default", nil)

	rv.SetSortColumn(2, false)
	col, asc := rv.GetSortColumn()
	if col != 2 {
		t.Errorf("GetSortColumn() col = %d, want 2", col)
	}
	if asc {
		t.Error("GetSortColumn() asc should be false")
	}
}

func TestResourceViewLifecycle(t *testing.T) {
	rv := NewResourceView("pods", "v1/pods", "default", nil)

	rv.Start()
	if rv.IsStopped() {
		t.Error("IsStopped() should be false after Start")
	}

	rv.Stop()
	if !rv.IsStopped() {
		t.Error("IsStopped() should be true after Stop")
	}
}

func TestResourceViewHints(t *testing.T) {
	rv := NewResourceView("pods", "v1/pods", "default", nil)

	hints := rv.Hints()
	// Should have at least the navigation actions (j, k, g, G)
	if len(hints) < 4 {
		t.Errorf("Hints() should have at least 4 hints, got %d", len(hints))
	}
}
