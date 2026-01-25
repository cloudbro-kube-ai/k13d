// Package views provides high-level view components following k9s patterns.
// Views are composed of UI primitives and handle user interactions.
package views

import (
	"context"

	"github.com/gdamore/tcell/v2"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ui/actions"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ui/models"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ui/render"
	"github.com/rivo/tview"
)

// View is the interface for all views.
type View interface {
	// Name returns the view name.
	Name() string
	// Init initializes the view.
	Init(ctx context.Context) error
	// Start starts the view (begin watching/refreshing).
	Start()
	// Stop stops the view (stop watching/refreshing).
	Stop()
	// Actions returns the view's key actions.
	Actions() *actions.KeyActions
	// Primitive returns the tview primitive for this view.
	Primitive() tview.Primitive
	// SetFocus sets focus to this view.
	SetFocus(app *tview.Application)
}

// Hintable is the interface for views that provide hints.
type Hintable interface {
	// Hints returns the current hints for the view.
	Hints() []actions.Hint
}

// Filterable is the interface for views that support filtering.
type Filterable interface {
	// SetFilter sets the filter text.
	SetFilter(filter string)
	// GetFilter returns the current filter.
	GetFilter() string
	// ClearFilter clears the filter.
	ClearFilter()
}

// Sortable is the interface for views that support sorting.
type Sortable interface {
	// SetSortColumn sets the sort column.
	SetSortColumn(col int, ascending bool)
	// GetSortColumn returns the current sort column and direction.
	GetSortColumn() (int, bool)
}

// BaseView provides common view functionality.
type BaseView struct {
	name    string
	actions *actions.KeyActions
	stopped bool
}

// NewBaseView creates a new BaseView.
func NewBaseView(name string) *BaseView {
	return &BaseView{
		name:    name,
		actions: actions.NewKeyActions(),
	}
}

// Name returns the view name.
func (v *BaseView) Name() string {
	return v.name
}

// Actions returns the view's key actions.
func (v *BaseView) Actions() *actions.KeyActions {
	return v.actions
}

// IsStopped returns true if the view is stopped.
func (v *BaseView) IsStopped() bool {
	return v.stopped
}

// ResourceView is a view for displaying Kubernetes resources.
type ResourceView struct {
	*BaseView
	*tview.Flex

	table     *tview.Table
	model     *models.ResourceModel
	renderer  render.Renderer
	app       *tview.Application
	namespace string
	gvr       string

	sortCol    int
	sortAsc    bool
	selectedFn func(namespace, name string)
}

// NewResourceView creates a new ResourceView.
func NewResourceView(name, gvr, namespace string, renderer render.Renderer) *ResourceView {
	v := &ResourceView{
		BaseView:  NewBaseView(name),
		Flex:      tview.NewFlex(),
		table:     tview.NewTable(),
		model:     models.NewResourceModel(gvr, namespace),
		renderer:  renderer,
		namespace: namespace,
		gvr:       gvr,
		sortAsc:   true,
	}

	// Setup table
	v.table.SetSelectable(true, false)
	v.table.SetBorder(false)
	v.table.SetBorderPadding(0, 0, 1, 1)
	v.table.SetSelectedStyle(tcell.StyleDefault.
		Foreground(tcell.ColorBlack).
		Background(tcell.ColorAqua))

	// Add table to flex
	v.Flex.AddItem(v.table, 0, 1, true)

	// Bind default actions
	v.bindDefaultActions()

	return v
}

// bindDefaultActions binds the default resource view actions.
func (v *ResourceView) bindDefaultActions() {
	// Navigation
	v.actions.AddRune('j', actions.NewKeyAction("Down", func(ctx context.Context) error {
		row, _ := v.table.GetSelection()
		if row < v.table.GetRowCount()-1 {
			v.table.Select(row+1, 0)
		}
		return nil
	}))
	v.actions.AddRune('k', actions.NewKeyAction("Up", func(ctx context.Context) error {
		row, _ := v.table.GetSelection()
		if row > 1 {
			v.table.Select(row-1, 0)
		}
		return nil
	}))
	v.actions.AddRune('g', actions.NewKeyAction("Top", func(ctx context.Context) error {
		if v.table.GetRowCount() > 1 {
			v.table.Select(1, 0)
		}
		return nil
	}))
	v.actions.AddRune('G', actions.NewKeyAction("Bottom", func(ctx context.Context) error {
		if v.table.GetRowCount() > 1 {
			v.table.Select(v.table.GetRowCount()-1, 0)
		}
		return nil
	}))
}

// Init initializes the view.
func (v *ResourceView) Init(ctx context.Context) error {
	// Subscribe to model updates
	v.model.Subscribe(v)
	return nil
}

// Start starts the view.
func (v *ResourceView) Start() {
	v.stopped = false
	v.model.RequestRefresh()
}

// Stop stops the view.
func (v *ResourceView) Stop() {
	v.stopped = true
}

// Primitive returns the tview primitive.
func (v *ResourceView) Primitive() tview.Primitive {
	return v.Flex
}

// SetFocus sets focus to the table.
func (v *ResourceView) SetFocus(app *tview.Application) {
	v.app = app
	app.SetFocus(v.table)
}

// DataChanged handles model data updates.
func (v *ResourceView) DataChanged(headers []string, rows [][]string) {
	if v.app == nil {
		return
	}
	v.app.QueueUpdateDraw(func() {
		v.renderTable(headers, rows)
	})
}

// DataFailed handles model data errors.
func (v *ResourceView) DataFailed(err error) {
	// Handle error (could show flash message)
}

// renderTable renders the table from data.
func (v *ResourceView) renderTable(headers []string, rows [][]string) {
	v.table.Clear()

	// Render headers
	for col, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignLeft)
		v.table.SetCell(0, col, cell)
	}

	// Get colorer if available
	var colorer render.ColorerFunc
	if v.renderer != nil {
		colorer = v.renderer.ColorerFunc()
	}

	// Render rows
	for rowIdx, row := range rows {
		var rowColor tcell.Color = tcell.ColorDefault
		if colorer != nil && len(row) > 0 {
			ns := ""
			if len(row) > 0 {
				ns = row[0]
			}
			rowColor = colorer(ns, render.Row{Fields: row})
		}

		for colIdx, cell := range row {
			tableCell := tview.NewTableCell(cell).
				SetTextColor(rowColor).
				SetAlign(tview.AlignLeft)
			v.table.SetCell(rowIdx+1, colIdx, tableCell)
		}
	}

	// Preserve selection
	rowCount := v.table.GetRowCount()
	if rowCount > 1 {
		currentRow, _ := v.table.GetSelection()
		if currentRow < 1 {
			v.table.Select(1, 0)
		} else if currentRow >= rowCount {
			v.table.Select(rowCount-1, 0)
		}
	}
}

// SetFilter implements Filterable.
func (v *ResourceView) SetFilter(filter string) {
	v.model.SetFilter(filter)
}

// GetFilter implements Filterable.
func (v *ResourceView) GetFilter() string {
	return v.model.GetFilter()
}

// ClearFilter implements Filterable.
func (v *ResourceView) ClearFilter() {
	v.model.SetFilter("")
}

// SetSortColumn implements Sortable.
func (v *ResourceView) SetSortColumn(col int, ascending bool) {
	v.sortCol = col
	v.sortAsc = ascending
}

// GetSortColumn implements Sortable.
func (v *ResourceView) GetSortColumn() (int, bool) {
	return v.sortCol, v.sortAsc
}

// GetSelectedResource returns the currently selected resource.
func (v *ResourceView) GetSelectedResource() (namespace, name string) {
	row, _ := v.table.GetSelection()
	if row < 1 || row >= v.table.GetRowCount() {
		return "", ""
	}

	// Assuming first column is namespace and second is name
	nsCell := v.table.GetCell(row, 0)
	nameCell := v.table.GetCell(row, 1)
	if nsCell != nil && nameCell != nil {
		return nsCell.Text, nameCell.Text
	}
	return "", ""
}

// SetSelectedCallback sets the callback for when a resource is selected.
func (v *ResourceView) SetSelectedCallback(fn func(namespace, name string)) {
	v.selectedFn = fn
	v.table.SetSelectedFunc(func(row, col int) {
		if row < 1 {
			return
		}
		ns, name := v.GetSelectedResource()
		if fn != nil && name != "" {
			fn(ns, name)
		}
	})
}

// Hints returns the view hints.
func (v *ResourceView) Hints() []actions.Hint {
	return v.actions.Hints()
}
