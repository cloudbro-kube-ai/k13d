package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// XRayView shows hierarchical resource relationships (k9s :xray equivalent)
type XRayView struct {
	*tview.TreeView
	app  *App
	root *tview.TreeNode
}

// XRayNode represents a single node in the XRay tree for testability
type XRayNode struct {
	Text     string
	Children []*XRayNode
}

// NewXRayView creates a new XRay view showing resource ownership hierarchy
func NewXRayView(app *App, resourceType, namespace string) *XRayView {
	root := tview.NewTreeNode(fmt.Sprintf("XRay: %s", resourceType)).
		SetColor(tcell.NewRGBColor(122, 162, 247)) // #7aa2f7 blue

	tree := tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root)
	tree.SetBorder(true).
		SetTitle(fmt.Sprintf(" XRay: %s (Esc:close  j/k:navigate) ", resourceType)).
		SetTitleAlign(tview.AlignLeft)

	xray := &XRayView{
		TreeView: tree,
		app:      app,
		root:     root,
	}

	return xray
}

// Refresh fetches cluster data and populates the tree
func (x *XRayView) Refresh(ctx context.Context, resourceType, namespace string) {
	if x.app.k8s == nil {
		x.root.ClearChildren()
		x.root.AddChild(tview.NewTreeNode("[red]Kubernetes client not available[white]"))
		return
	}

	x.root.SetText(fmt.Sprintf("XRay: %s", resourceType))
	x.root.ClearChildren()

	k := x.app.k8s

	// Build tree based on resource type
	switch normalizeResourceType(resourceType) {
	case "deployments":
		x.buildDeploymentTree(ctx, k, namespace)
	case "statefulsets":
		x.buildStatefulSetTree(ctx, k, namespace)
	case "jobs":
		x.buildJobTree(ctx, k, namespace)
	case "cronjobs":
		x.buildCronJobTree(ctx, k, namespace)
	case "daemonsets":
		x.buildDaemonSetTree(ctx, k, namespace)
	default:
		// Default to deployments
		x.buildDeploymentTree(ctx, k, namespace)
	}

	if len(x.root.GetChildren()) == 0 {
		x.root.AddChild(tview.NewTreeNode("[gray]No resources found[white]"))
	}
}

// normalizeResourceType maps aliases to canonical resource names
func normalizeResourceType(rt string) string {
	switch strings.ToLower(rt) {
	case "deploy", "deployments", "deployment":
		return "deployments"
	case "sts", "statefulsets", "statefulset":
		return "statefulsets"
	case "job", "jobs":
		return "jobs"
	case "cj", "cronjobs", "cronjob":
		return "cronjobs"
	case "ds", "daemonsets", "daemonset":
		return "daemonsets"
	default:
		return "deployments"
	}
}

// buildDeploymentTree creates: Deployment → ReplicaSet → Pod hierarchy
func (x *XRayView) buildDeploymentTree(ctx context.Context, k interface{}, namespace string) {
	client := x.app.k8s

	deps, err := client.ListDeployments(ctx, namespace)
	if err != nil {
		x.root.AddChild(tview.NewTreeNode(fmt.Sprintf("[red]Error: %v[white]", err)))
		return
	}

	rsList, err := client.ListReplicaSets(ctx, namespace)
	if err != nil {
		rsList = nil
	}

	pods, err := client.ListPods(ctx, namespace)
	if err != nil {
		pods = nil
	}

	tree := BuildDeploymentTree(deps, rsList, pods)
	addXRayNodesToTree(x.root, tree)
}

// buildStatefulSetTree creates: StatefulSet → Pod hierarchy
func (x *XRayView) buildStatefulSetTree(ctx context.Context, k interface{}, namespace string) {
	client := x.app.k8s

	stses, err := client.ListStatefulSets(ctx, namespace)
	if err != nil {
		x.root.AddChild(tview.NewTreeNode(fmt.Sprintf("[red]Error: %v[white]", err)))
		return
	}

	pods, err := client.ListPods(ctx, namespace)
	if err != nil {
		pods = nil
	}

	tree := BuildStatefulSetTree(stses, pods)
	addXRayNodesToTree(x.root, tree)
}

// buildJobTree creates: Job → Pod hierarchy
func (x *XRayView) buildJobTree(ctx context.Context, k interface{}, namespace string) {
	client := x.app.k8s

	jobs, err := client.ListJobs(ctx, namespace)
	if err != nil {
		x.root.AddChild(tview.NewTreeNode(fmt.Sprintf("[red]Error: %v[white]", err)))
		return
	}

	pods, err := client.ListPods(ctx, namespace)
	if err != nil {
		pods = nil
	}

	tree := BuildJobTree(jobs, pods)
	addXRayNodesToTree(x.root, tree)
}

// buildCronJobTree creates: CronJob → Job → Pod hierarchy
func (x *XRayView) buildCronJobTree(ctx context.Context, k interface{}, namespace string) {
	client := x.app.k8s

	cjs, err := client.ListCronJobs(ctx, namespace)
	if err != nil {
		x.root.AddChild(tview.NewTreeNode(fmt.Sprintf("[red]Error: %v[white]", err)))
		return
	}

	jobs, err := client.ListJobs(ctx, namespace)
	if err != nil {
		jobs = nil
	}

	pods, err := client.ListPods(ctx, namespace)
	if err != nil {
		pods = nil
	}

	tree := BuildCronJobTree(cjs, jobs, pods)
	addXRayNodesToTree(x.root, tree)
}

// buildDaemonSetTree creates: DaemonSet → Pod hierarchy
func (x *XRayView) buildDaemonSetTree(ctx context.Context, k interface{}, namespace string) {
	client := x.app.k8s

	dss, err := client.ListDaemonSets(ctx, namespace)
	if err != nil {
		x.root.AddChild(tview.NewTreeNode(fmt.Sprintf("[red]Error: %v[white]", err)))
		return
	}

	pods, err := client.ListPods(ctx, namespace)
	if err != nil {
		pods = nil
	}

	tree := BuildDaemonSetTree(dss, pods)
	addXRayNodesToTree(x.root, tree)
}

// --- Tree building functions (exported for testing) ---

// BuildDeploymentTree builds the XRay tree for deployments: Deployment → ReplicaSet → Pod
func BuildDeploymentTree(deps []appsv1.Deployment, rsList []appsv1.ReplicaSet, pods []corev1.Pod) []*XRayNode {
	var nodes []*XRayNode

	for _, dep := range deps {
		desired := int32(1)
		if dep.Spec.Replicas != nil {
			desired = *dep.Spec.Replicas
		}
		depNode := &XRayNode{
			Text: fmt.Sprintf("Deployment/%s (%d/%d ready)", dep.Name, dep.Status.ReadyReplicas, desired),
		}

		// Find ReplicaSets owned by this deployment
		for _, rs := range rsList {
			if !isOwnedBy(rs.OwnerReferences, dep.Name, "Deployment") {
				continue
			}
			rsDesired := int32(0)
			if rs.Spec.Replicas != nil {
				rsDesired = *rs.Spec.Replicas
			}
			// Skip RS with 0 replicas (old revisions)
			if rsDesired == 0 && rs.Status.Replicas == 0 {
				continue
			}
			rsNode := &XRayNode{
				Text: fmt.Sprintf("ReplicaSet/%s (%d/%d)", rs.Name, rs.Status.ReadyReplicas, rsDesired),
			}

			// Find Pods owned by this ReplicaSet
			for _, pod := range pods {
				if isOwnedBy(pod.OwnerReferences, rs.Name, "ReplicaSet") {
					rsNode.Children = append(rsNode.Children, &XRayNode{
						Text: fmt.Sprintf("Pod/%s (%s)", pod.Name, podStatusText(pod)),
					})
				}
			}

			depNode.Children = append(depNode.Children, rsNode)
		}

		nodes = append(nodes, depNode)
	}

	return nodes
}

// BuildStatefulSetTree builds the XRay tree for statefulsets: StatefulSet → Pod
func BuildStatefulSetTree(stses []appsv1.StatefulSet, pods []corev1.Pod) []*XRayNode {
	var nodes []*XRayNode

	for _, sts := range stses {
		desired := int32(1)
		if sts.Spec.Replicas != nil {
			desired = *sts.Spec.Replicas
		}
		stsNode := &XRayNode{
			Text: fmt.Sprintf("StatefulSet/%s (%d/%d ready)", sts.Name, sts.Status.ReadyReplicas, desired),
		}

		for _, pod := range pods {
			if isOwnedBy(pod.OwnerReferences, sts.Name, "StatefulSet") {
				stsNode.Children = append(stsNode.Children, &XRayNode{
					Text: fmt.Sprintf("Pod/%s (%s)", pod.Name, podStatusText(pod)),
				})
			}
		}

		nodes = append(nodes, stsNode)
	}

	return nodes
}

// BuildJobTree builds the XRay tree for jobs: Job → Pod
func BuildJobTree(jobs []batchv1.Job, pods []corev1.Pod) []*XRayNode {
	var nodes []*XRayNode

	for _, job := range jobs {
		status := jobStatusText(job)
		jobNode := &XRayNode{
			Text: fmt.Sprintf("Job/%s (%s)", job.Name, status),
		}

		for _, pod := range pods {
			if isOwnedBy(pod.OwnerReferences, job.Name, "Job") {
				jobNode.Children = append(jobNode.Children, &XRayNode{
					Text: fmt.Sprintf("Pod/%s (%s)", pod.Name, podStatusText(pod)),
				})
			}
		}

		nodes = append(nodes, jobNode)
	}

	return nodes
}

// BuildCronJobTree builds the XRay tree for cronjobs: CronJob → Job → Pod
func BuildCronJobTree(cjs []batchv1.CronJob, jobs []batchv1.Job, pods []corev1.Pod) []*XRayNode {
	var nodes []*XRayNode

	for _, cj := range cjs {
		cjNode := &XRayNode{
			Text: fmt.Sprintf("CronJob/%s (schedule: %s)", cj.Name, cj.Spec.Schedule),
		}

		for _, job := range jobs {
			if !isOwnedBy(job.OwnerReferences, cj.Name, "CronJob") {
				continue
			}
			status := jobStatusText(job)
			jobNode := &XRayNode{
				Text: fmt.Sprintf("Job/%s (%s)", job.Name, status),
			}

			for _, pod := range pods {
				if isOwnedBy(pod.OwnerReferences, job.Name, "Job") {
					jobNode.Children = append(jobNode.Children, &XRayNode{
						Text: fmt.Sprintf("Pod/%s (%s)", pod.Name, podStatusText(pod)),
					})
				}
			}

			cjNode.Children = append(cjNode.Children, jobNode)
		}

		nodes = append(nodes, cjNode)
	}

	return nodes
}

// BuildDaemonSetTree builds the XRay tree for daemonsets: DaemonSet → Pod
func BuildDaemonSetTree(dss []appsv1.DaemonSet, pods []corev1.Pod) []*XRayNode {
	var nodes []*XRayNode

	for _, ds := range dss {
		dsNode := &XRayNode{
			Text: fmt.Sprintf("DaemonSet/%s (%d/%d ready)", ds.Name, ds.Status.NumberReady, ds.Status.DesiredNumberScheduled),
		}

		for _, pod := range pods {
			if isOwnedBy(pod.OwnerReferences, ds.Name, "DaemonSet") {
				dsNode.Children = append(dsNode.Children, &XRayNode{
					Text: fmt.Sprintf("Pod/%s (%s)", pod.Name, podStatusText(pod)),
				})
			}
		}

		nodes = append(nodes, dsNode)
	}

	return nodes
}

// --- Helper functions ---

// isOwnedBy checks if the ownerReferences contain an owner with the given name and kind
func isOwnedBy(refs []metav1.OwnerReference, name, kind string) bool {
	for _, ref := range refs {
		if ref.Name == name && ref.Kind == kind {
			return true
		}
	}
	return false
}

// podStatusText returns a colored status string for a pod
func podStatusText(pod corev1.Pod) string {
	status := string(pod.Status.Phase)

	// Check container statuses for more specific state
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			status = cs.State.Waiting.Reason
			break
		}
	}

	switch status {
	case "Running":
		return "[green]Running[white]"
	case "Succeeded":
		return "[green]Succeeded[white]"
	case "Pending":
		return "[yellow]Pending[white]"
	case "ContainerCreating":
		return "[yellow]ContainerCreating[white]"
	case "CrashLoopBackOff":
		return "[red]CrashLoopBackOff[white]"
	case "ImagePullBackOff", "ErrImagePull":
		return "[red]" + status + "[white]"
	case "Failed":
		return "[red]Failed[white]"
	default:
		return status
	}
}

// jobStatusText returns a status string for a job
func jobStatusText(job batchv1.Job) string {
	if job.Status.Succeeded > 0 && job.Status.Active == 0 {
		return "[green]Complete[white]"
	}
	if job.Status.Failed > 0 && job.Status.Active == 0 {
		return "[red]Failed[white]"
	}
	if job.Status.Active > 0 {
		return "[yellow]Active[white]"
	}
	return "[gray]Pending[white]"
}

// addXRayNodesToTree recursively adds XRayNode children to a tview.TreeNode
func addXRayNodesToTree(parent *tview.TreeNode, nodes []*XRayNode) {
	for _, n := range nodes {
		child := tview.NewTreeNode(n.Text).
			SetSelectable(true).
			SetExpanded(true)

		// Color the tree node based on content
		if strings.Contains(n.Text, "[red]") {
			child.SetColor(tcell.NewRGBColor(247, 118, 142)) // red
		} else if strings.Contains(n.Text, "[yellow]") {
			child.SetColor(tcell.NewRGBColor(224, 175, 104)) // yellow
		} else if strings.Contains(n.Text, "[green]") {
			child.SetColor(tcell.NewRGBColor(158, 206, 106)) // green
		} else {
			child.SetColor(tcell.NewRGBColor(192, 202, 245)) // primary text
		}

		addXRayNodesToTree(child, n.Children)
		parent.AddChild(child)
	}
}

// showXRay displays the XRay modal for the given resource type
func (a *App) showXRay(resourceType string) {
	if resourceType == "" {
		resourceType = "deploy"
	}

	a.mx.RLock()
	ns := a.currentNamespace
	a.mx.RUnlock()

	xray := NewXRayView(a, resourceType, ns)

	xray.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			a.closeModal("xray")
			a.SetFocus(a.table)
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'q' {
				a.closeModal("xray")
				a.SetFocus(a.table)
				return nil
			}
		}
		return event
	})

	a.showModal("xray", centered(xray, 80, 30), true)
	a.SetFocus(xray)

	// Fetch data asynchronously
	a.safeGo("xray-initial", func() {
		xray.Refresh(a.getAppContext(), resourceType, ns)
	})
}
