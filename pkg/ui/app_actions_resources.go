package ui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// confirmDelete shows a delete confirmation dialog
func (a *App) confirmDelete() {
	a.mx.RLock()
	resource := a.currentResource
	selectedCount := len(a.selectedRows)
	a.mx.RUnlock()

	// RBAC check
	if !a.checkTUIPermission(resource, "delete") {
		return
	}

	// Check for multi-select deletion
	if selectedCount > 0 {
		a.confirmDeleteMultiple()
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	// Get resource info
	var ns, name string
	switch resource {
	case "nodes", "no", "namespaces", "ns":
		name = a.getTableCellText(row, 0)
	default:
		ns = a.getTableCellText(row, 0)
		name = a.getTableCellText(row, 1)
	}

	// Create confirmation modal
	modal := tview.NewModal().
		SetText(fmt.Sprintf("[red]Delete %s?[white]\n\n%s/%s\n\nThis action cannot be undone.", resource, ns, name)).
		AddButtons([]string{"Cancel", "Delete"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.closeModal("delete-confirm")
			a.SetFocus(a.table)

			if buttonLabel == "Delete" {
				a.safeGo("deleteResource", func() { a.deleteResource(ns, name, resource) })
			}
		})

	modal.SetBackgroundColor(tcell.ColorDarkRed)

	a.showModal("delete-confirm", modal, true)
}

// confirmDeleteMultiple confirms deletion of multiple selected resources (k9s style)
func (a *App) confirmDeleteMultiple() {
	a.mx.RLock()
	resource := a.currentResource
	selectedCount := len(a.selectedRows)
	selectedRowsCopy := make(map[int]bool)
	for k, v := range a.selectedRows {
		selectedRowsCopy[k] = v
	}
	a.mx.RUnlock()

	if selectedCount == 0 {
		return
	}

	// Build list of resources to delete
	var items []struct{ ns, name string }
	for row := range selectedRowsCopy {
		var ns, name string
		switch resource {
		case "nodes", "no", "namespaces", "ns":
			name = strings.TrimSpace(tview.TranslateANSI(a.getTableCellText(row, 0)))
		default:
			ns = strings.TrimSpace(tview.TranslateANSI(a.getTableCellText(row, 0)))
			name = strings.TrimSpace(tview.TranslateANSI(a.getTableCellText(row, 1)))
		}
		if name != "" {
			items = append(items, struct{ ns, name string }{ns, name})
		}
	}

	// Create confirmation modal
	modal := tview.NewModal().
		SetText(fmt.Sprintf("[red]Delete %d %s?[white]\n\nThis action cannot be undone.", len(items), resource)).
		AddButtons([]string{"Cancel", "Delete All"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.closeModal("delete-confirm")
			a.SetFocus(a.table)

			if buttonLabel == "Delete All" {
				a.safeGo("deleteResource-batch", func() {
					for _, item := range items {
						a.deleteResource(item.ns, item.name, resource)
					}
					a.clearSelections()
					a.refresh()
				})
			}
		})

	modal.SetBackgroundColor(tcell.ColorDarkRed)

	a.showModal("delete-confirm", modal, true)
}

// deleteResource deletes the specified resource
func (a *App) deleteResource(ns, name, resource string) {
	ctx, cancel := context.WithTimeout(a.getAppContext(), 30*time.Second)
	defer cancel()

	gvr, ok := a.k8s.GetGVR(resource)
	if !ok {
		a.flashMsg(fmt.Sprintf("Unknown resource type: %s", resource), true)
		a.recordTUIAudit("delete", fmt.Sprintf("%s/%s", resource, name), "Unknown resource type", false, "Unknown resource type")
		return
	}

	a.flashMsg(fmt.Sprintf("Deleting %s/%s...", resource, name), false)

	resourcePath := fmt.Sprintf("%s/%s", resource, name)
	if ns != "" {
		resourcePath = fmt.Sprintf("%s/%s/%s", ns, resource, name)
	}

	err := a.k8s.DeleteResource(ctx, gvr, ns, name)
	if err != nil {
		a.flashMsg(fmt.Sprintf("Delete failed: %v", err), true)
		a.recordTUIAudit("delete", resourcePath, fmt.Sprintf("Failed to delete %s", name), false, err.Error())
		return
	}

	a.flashMsg(fmt.Sprintf("Deleted %s/%s", resource, name), false)
	a.recordTUIAudit("delete", resourcePath, fmt.Sprintf("Deleted %s %s", resource, name), true, "")
	a.safeGo("deleteResource-refresh", func() { a.refresh() })
}

// editResource opens the resource in $EDITOR (k9s e key)
func (a *App) editResource() {
	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	var ns, name string
	switch resource {
	case "nodes", "no", "namespaces", "ns", "persistentvolumes", "storageclasses",
		"clusterroles", "clusterrolebindings", "customresourcedefinitions":
		name = a.getTableCellText(row, 0)
	default:
		ns = a.getTableCellText(row, 0)
		name = a.getTableCellText(row, 1)
	}

	resourcePath := fmt.Sprintf("%s/%s", resource, name)
	if ns != "" {
		resourcePath = fmt.Sprintf("%s/%s/%s", ns, resource, name)
	}

	// Suspend TUI and run kubectl edit
	a.Suspend(func() {
		var cmd *exec.Cmd
		if ns != "" {
			cmd = exec.Command("kubectl", "edit", resource, name, "-n", ns)
		} else {
			cmd = exec.Command("kubectl", "edit", resource, name)
		}
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "\nEdit failed: %v\nPress Enter to return...\n", err)
			_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		}
	})

	// Record audit (we can't know if edit was actually saved, so just record the attempt)
	a.recordTUIAudit("edit", resourcePath, fmt.Sprintf("Edited %s %s via $EDITOR", resource, name), true, "")

	// Refresh after edit
	a.safeGo("editResource-refresh", func() { a.refresh() })
}

// showYAML shows YAML for selected resource (k9s y key) with Vim-style navigation
func (a *App) showYAML() {
	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	var ns, name string
	switch resource {
	case "nodes", "no", "namespaces", "ns", "persistentvolumes", "storageclasses",
		"clusterroles", "clusterrolebindings", "customresourcedefinitions":
		name = a.getTableCellText(row, 0)
	default:
		ns = a.getTableCellText(row, 0)
		name = a.getTableCellText(row, 1)
	}

	// Use VimViewer for Vim-style navigation and search
	isSecret := resource == "secrets" || resource == "sec"
	title := fmt.Sprintf(" YAML: %s/%s [gray](Esc:close /search n/N:next/prev Ctrl+D/U:scroll)[white] ", resource, name)
	if isSecret {
		title = fmt.Sprintf(" YAML: %s/%s [gray](Esc:close /search x:decode)[white] ", resource, name)
	}
	yamlView := NewVimViewer(a, "yaml", title)
	if isSecret {
		yamlView.isSecretView = true
		yamlView.updateTitle()
	}

	yamlView.SetContent("[yellow]Loading...[white]")

	a.showModal("yaml", yamlView, true)
	a.SetFocus(yamlView)

	// Fetch YAML
	a.safeGo("editResource-fetch", func() {
		ctx, cancel := context.WithTimeout(a.getAppContext(), 10*time.Second)
		defer cancel()

		gvr, ok := a.k8s.GetGVR(resource)
		if !ok {
			a.QueueUpdateDraw(func() {
				yamlView.SetContent(fmt.Sprintf("[red]Unknown resource type: %s", resource))
			})
			return
		}

		yaml, err := a.k8s.GetResourceYAML(ctx, ns, name, gvr)
		a.QueueUpdateDraw(func() {
			if err != nil {
				yamlView.SetContent(fmt.Sprintf("[red]Error: %v", err))
			} else {
				yamlView.SetContent(yaml)
				if isSecret {
					yamlView.rawYAML = yaml
				}
			}
		})
	})
}

// showDescribe shows describe output for selected resource (like kubectl describe) with Vim-style navigation
func (a *App) showDescribe() {
	row, _ := a.table.GetSelection()
	if row <= 0 {
		a.flashMsg("No resource selected. Please select a resource from the list first.", true)
		return
	}

	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	// Get namespace and name from table
	nsCell := a.table.GetCell(row, 0)
	nameCell := a.table.GetCell(row, 1)
	if nsCell == nil || nameCell == nil {
		a.flashMsg("Cannot get resource info from table. The resource may be invalid or not fully loaded.", true)
		return
	}

	ns := nsCell.Text
	name := nameCell.Text

	// For cluster-scoped resources, name is in column 0
	if resource == "nodes" || resource == "namespaces" || resource == "persistentvolumes" ||
		resource == "storageclasses" || resource == "clusterroles" || resource == "clusterrolebindings" ||
		resource == "customresourcedefinitions" {
		name = ns
		ns = ""
	}

	// Use VimViewer for Vim-style navigation and search
	descView := NewVimViewer(a, "describe",
		fmt.Sprintf(" Describe: %s/%s [gray](Esc:close /search n/N:next/prev Ctrl+D/U:scroll)[white] ", resource, name))

	descView.SetContent("[yellow]Loading...[white]")

	// Add to pages
	a.showModal("describe", descView, true)
	a.SetFocus(descView)

	// Fetch describe output in background
	a.safeGo("describeResource-fetch", func() {
		ctx := a.prepareContext()
		output, err := a.k8s.DescribeResource(ctx, resource, ns, name)
		if err != nil {
			a.QueueUpdateDraw(func() {
				descView.SetContent(fmt.Sprintf("[red]Error: %v[white]", err))
			})
			return
		}

		a.QueueUpdateDraw(func() {
			descView.SetContent(output)
			descView.ScrollToBeginning()
		})
	})
}

// showRelatedResource shows related resources (k9s z key)
func (a *App) showRelatedResource() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	var ns, name string
	switch resource {
	case "nodes", "namespaces", "persistentvolumes", "storageclasses",
		"clusterroles", "clusterrolebindings", "customresourcedefinitions":
		name = a.getTableCellText(row, 0)
	default:
		ns = a.getTableCellText(row, 0)
		name = a.getTableCellText(row, 1)
	}

	// Different behavior based on resource type
	switch resource {
	case "deployments", "deploy":
		// Push nav history (navMx only, no nesting with mx)
		a.navMx.Lock()
		a.mx.RLock()
		a.navigationStack = append(a.navigationStack, navHistory{resource, a.currentNamespace, a.filterText})
		a.mx.RUnlock()
		a.navMx.Unlock()

		// navigateTo() handles mx, watcher, and refresh safely
		a.navigateTo("replicasets", ns, name)

	default:
		a.flashMsg(fmt.Sprintf("No related resources defined for %s. Try navigating manually using command mode (:)", resource), true)
	}
}

// showNode shows the node where the selected pod is running (k9s o key)
func (a *App) showNode() {
	a.mx.RLock()
	resource := a.currentResource
	a.mx.RUnlock()

	if resource != "pods" && resource != "po" {
		a.flashMsg("Show node is only available for pods. Navigate to pods view first using :pods", true)
		return
	}

	row, _ := a.table.GetSelection()
	if row <= 0 {
		return
	}

	ns := a.getTableCellText(row, 0)
	name := a.getTableCellText(row, 1)

	// Get pod to find node
	ctx, cancel := context.WithTimeout(a.getAppContext(), 5*time.Second)
	defer cancel()

	pods, err := a.k8s.ListPods(ctx, ns)
	if err != nil {
		a.flashMsg(fmt.Sprintf("Failed to list pods in namespace %s: %v. Check cluster connectivity.", ns, err), true)
		return
	}

	var nodeName string
	for _, pod := range pods {
		if pod.Name == name {
			nodeName = pod.Spec.NodeName
			break
		}
	}

	if nodeName == "" {
		a.flashMsg("Pod not scheduled to a node yet. Wait for the scheduler to assign it or check pod events.", true)
		return
	}

	// Push nav history (navMx only, no nesting with mx)
	a.navMx.Lock()
	a.mx.RLock()
	a.navigationStack = append(a.navigationStack, navHistory{resource, a.currentNamespace, a.filterText})
	a.mx.RUnlock()
	a.navMx.Unlock()

	// navigateTo() handles mx, watcher, and refresh safely
	a.navigateTo("nodes", "", nodeName)
}
