package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// AppGroup represents an application with all its related resources
type AppGroup struct {
	Name      string              // app.kubernetes.io/name value
	Version   string              // app.kubernetes.io/version (if present)
	Component string              // app.kubernetes.io/component (if present)
	Resources map[string][]string // kind -> list of resource names
	Status    string              // overall: "healthy", "degraded", "failing"
	PodCount  int                 // total pods
	ReadyPods int                 // ready pods
}

// AppView shows applications grouped by app.kubernetes.io/name
type AppView struct {
	*tview.TreeView
	app  *App
	root *tview.TreeNode
}

// NewAppView creates a new application-centric view
func NewAppView(app *App, namespace string) *AppView {
	root := tview.NewTreeNode("Applications").
		SetColor(tcell.NewRGBColor(122, 162, 247)) // #7aa2f7 blue

	tree := tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root)
	tree.SetBorder(true).
		SetTitle(" Applications (Esc:close  j/k:navigate) ").
		SetTitleAlign(tview.AlignLeft)

	return &AppView{
		TreeView: tree,
		app:      app,
		root:     root,
	}
}

// Refresh fetches cluster data and populates the application tree
func (v *AppView) Refresh(ctx context.Context, namespace string) {
	if v.app.k8s == nil {
		v.root.ClearChildren()
		v.root.AddChild(tview.NewTreeNode("[red]Kubernetes client not available[white]"))
		return
	}

	v.root.ClearChildren()
	k := v.app.k8s

	// Fetch resources
	pods, _ := k.ListPods(ctx, namespace)
	deployments, _ := k.ListDeployments(ctx, namespace)
	statefulSets, _ := k.ListStatefulSets(ctx, namespace)
	daemonSets, _ := k.ListDaemonSets(ctx, namespace)
	services, _ := k.ListServices(ctx, namespace)
	configMaps, _ := k.ListConfigMaps(ctx, namespace)
	secrets, _ := k.ListSecrets(ctx, namespace)
	ingresses, _ := k.ListIngresses(ctx, namespace)

	groups := BuildAppGroups(pods, deployments, statefulSets, daemonSets, services, configMaps, secrets, ingresses)
	addAppGroupsToTree(v.root, groups)

	if len(v.root.GetChildren()) == 0 {
		v.root.AddChild(tview.NewTreeNode("[gray]No resources found[white]"))
	}
}

// BuildAppGroups groups resources by app.kubernetes.io/name label.
// Exported for testing.
func BuildAppGroups(
	pods []corev1.Pod,
	deployments []appsv1.Deployment,
	statefulSets []appsv1.StatefulSet,
	daemonSets []appsv1.DaemonSet,
	services []corev1.Service,
	configMaps []corev1.ConfigMap,
	secrets []corev1.Secret,
	ingresses []networkingv1.Ingress,
) []AppGroup {
	groups := make(map[string]*AppGroup)
	var ungrouped AppGroup
	ungrouped.Name = "Ungrouped Resources"
	ungrouped.Resources = make(map[string][]string)

	getOrCreate := func(name string) *AppGroup {
		if g, ok := groups[name]; ok {
			return g
		}
		g := &AppGroup{
			Name:      name,
			Resources: make(map[string][]string),
		}
		groups[name] = g
		return g
	}

	// Group Deployments
	for _, d := range deployments {
		appName := d.Labels["app.kubernetes.io/name"]
		if appName == "" {
			ungrouped.Resources["Deployment"] = append(ungrouped.Resources["Deployment"], d.Name)
			continue
		}
		g := getOrCreate(appName)
		if v := d.Labels["app.kubernetes.io/version"]; v != "" && g.Version == "" {
			g.Version = v
		}
		if c := d.Labels["app.kubernetes.io/component"]; c != "" && g.Component == "" {
			g.Component = c
		}
		g.Resources["Deployment"] = append(g.Resources["Deployment"], d.Name)
	}

	// Group StatefulSets
	for _, s := range statefulSets {
		appName := s.Labels["app.kubernetes.io/name"]
		if appName == "" {
			ungrouped.Resources["StatefulSet"] = append(ungrouped.Resources["StatefulSet"], s.Name)
			continue
		}
		g := getOrCreate(appName)
		if v := s.Labels["app.kubernetes.io/version"]; v != "" && g.Version == "" {
			g.Version = v
		}
		g.Resources["StatefulSet"] = append(g.Resources["StatefulSet"], s.Name)
	}

	// Group DaemonSets
	for _, ds := range daemonSets {
		appName := ds.Labels["app.kubernetes.io/name"]
		if appName == "" {
			ungrouped.Resources["DaemonSet"] = append(ungrouped.Resources["DaemonSet"], ds.Name)
			continue
		}
		g := getOrCreate(appName)
		g.Resources["DaemonSet"] = append(g.Resources["DaemonSet"], ds.Name)
	}

	// Group Services
	for _, svc := range services {
		appName := svc.Labels["app.kubernetes.io/name"]
		if appName == "" {
			ungrouped.Resources["Service"] = append(ungrouped.Resources["Service"], svc.Name)
			continue
		}
		g := getOrCreate(appName)
		g.Resources["Service"] = append(g.Resources["Service"], svc.Name)
	}

	// Group ConfigMaps
	for _, cm := range configMaps {
		appName := cm.Labels["app.kubernetes.io/name"]
		if appName == "" {
			ungrouped.Resources["ConfigMap"] = append(ungrouped.Resources["ConfigMap"], cm.Name)
			continue
		}
		g := getOrCreate(appName)
		g.Resources["ConfigMap"] = append(g.Resources["ConfigMap"], cm.Name)
	}

	// Group Secrets
	for _, sec := range secrets {
		appName := sec.Labels["app.kubernetes.io/name"]
		if appName == "" {
			ungrouped.Resources["Secret"] = append(ungrouped.Resources["Secret"], sec.Name)
			continue
		}
		g := getOrCreate(appName)
		g.Resources["Secret"] = append(g.Resources["Secret"], sec.Name)
	}

	// Group Ingresses
	for _, ing := range ingresses {
		appName := ing.Labels["app.kubernetes.io/name"]
		if appName == "" {
			ungrouped.Resources["Ingress"] = append(ungrouped.Resources["Ingress"], ing.Name)
			continue
		}
		g := getOrCreate(appName)
		g.Resources["Ingress"] = append(g.Resources["Ingress"], ing.Name)
	}

	// Count pods per app group and calculate health
	for _, pod := range pods {
		appName := pod.Labels["app.kubernetes.io/name"]
		if appName == "" {
			continue
		}
		if g, ok := groups[appName]; ok {
			g.PodCount++
			if pod.Status.Phase == corev1.PodRunning {
				allReady := true
				for _, cs := range pod.Status.ContainerStatuses {
					if !cs.Ready {
						allReady = false
						break
					}
				}
				if allReady {
					g.ReadyPods++
				}
			}
		}
	}

	// Calculate status for each group
	for _, g := range groups {
		g.Status = calculateAppStatus(g)
	}

	// Sort groups by name
	var result []AppGroup
	for _, g := range groups {
		result = append(result, *g)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	// Add ungrouped at the end if any
	if len(ungrouped.Resources) > 0 {
		ungrouped.Status = "healthy"
		result = append(result, ungrouped)
	}

	return result
}

// calculateAppStatus determines overall application health
func calculateAppStatus(g *AppGroup) string {
	if g.PodCount == 0 {
		// No pods tracked; if there are workload resources, status is unknown
		hasWorkload := len(g.Resources["Deployment"]) > 0 ||
			len(g.Resources["StatefulSet"]) > 0 ||
			len(g.Resources["DaemonSet"]) > 0
		if hasWorkload {
			return "healthy" // workloads exist but no labeled pods found
		}
		return "healthy"
	}
	if g.ReadyPods == g.PodCount {
		return "healthy"
	}
	if g.ReadyPods == 0 {
		return "failing"
	}
	return "degraded"
}

// addAppGroupsToTree populates the tview tree with application groups
func addAppGroupsToTree(parent *tview.TreeNode, groups []AppGroup) {
	// Resource kinds in display order
	kindOrder := []string{"Deployment", "StatefulSet", "DaemonSet", "Service", "ConfigMap", "Secret", "Ingress"}

	for _, g := range groups {
		// Build group header
		var header string
		if g.Name == "Ungrouped Resources" {
			header = fmt.Sprintf("─── %s ───", g.Name)
		} else {
			statusIcon := "[green]✓[white]"
			if g.Status == "degraded" {
				statusIcon = "[yellow]⚠[white]"
			} else if g.Status == "failing" {
				statusIcon = "[red]✗[white]"
			}

			if g.PodCount > 0 {
				header = fmt.Sprintf("%s (%d/%d pods ready) %s", g.Name, g.ReadyPods, g.PodCount, statusIcon)
			} else {
				header = fmt.Sprintf("%s %s", g.Name, statusIcon)
			}
		}

		groupNode := tview.NewTreeNode(header).
			SetSelectable(true).
			SetExpanded(true)

		// Color based on status
		switch g.Status {
		case "healthy":
			groupNode.SetColor(tcell.NewRGBColor(158, 206, 106)) // green
		case "degraded":
			groupNode.SetColor(tcell.NewRGBColor(224, 175, 104)) // yellow
		case "failing":
			groupNode.SetColor(tcell.NewRGBColor(247, 118, 142)) // red
		default:
			groupNode.SetColor(tcell.NewRGBColor(192, 202, 245)) // primary text
		}

		// Add resources as children in consistent order
		for _, kind := range kindOrder {
			names, ok := g.Resources[kind]
			if !ok {
				continue
			}
			sort.Strings(names)
			for _, name := range names {
				text := fmt.Sprintf("%s: %s", kind, name)
				child := tview.NewTreeNode(text).
					SetSelectable(true).
					SetColor(tcell.NewRGBColor(192, 202, 245)) // primary text
				groupNode.AddChild(child)
			}
		}

		// Add any remaining kinds not in kindOrder
		var extraKinds []string
		for kind := range g.Resources {
			found := false
			for _, k := range kindOrder {
				if kind == k {
					found = true
					break
				}
			}
			if !found {
				extraKinds = append(extraKinds, kind)
			}
		}
		sort.Strings(extraKinds)
		for _, kind := range extraKinds {
			names := g.Resources[kind]
			sort.Strings(names)
			for _, name := range names {
				text := fmt.Sprintf("%s: %s", kind, name)
				child := tview.NewTreeNode(text).
					SetSelectable(true).
					SetColor(tcell.NewRGBColor(192, 202, 245))
				groupNode.AddChild(child)
			}
		}

		parent.AddChild(groupNode)
	}
}

// showApplications displays the Application-Centric View modal
func (a *App) showApplications() {
	a.mx.RLock()
	ns := a.currentNamespace
	a.mx.RUnlock()

	appView := NewAppView(a, ns)

	appView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			a.closeModal("applications")
			a.SetFocus(a.table)
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				a.closeModal("applications")
				a.SetFocus(a.table)
				return nil
			case 'r':
				a.safeGo("applications-refresh", func() {
					appView.Refresh(a.getAppContext(), ns)
				})
				return nil
			}
		}
		return event
	})

	a.showModal("applications", centered(appView, 80, 30), true)
	a.SetFocus(appView)

	// Fetch data asynchronously
	a.safeGo("applications-initial", func() {
		appView.Refresh(a.getAppContext(), ns)
	})
}

// appGroupStatusText returns a display-friendly status (used in tests)
func appGroupStatusText(status string) string {
	switch status {
	case "healthy":
		return "[green]✓[white]"
	case "degraded":
		return "[yellow]⚠[white]"
	case "failing":
		return "[red]✗[white]"
	default:
		return strings.ToUpper(status)
	}
}
