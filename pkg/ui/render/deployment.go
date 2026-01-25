package render

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeploymentHeader defines the columns for Deployment resources.
var DeploymentHeader = Header{
	{Name: "NAMESPACE", MinWidth: 10, MaxWidth: 30},
	{Name: "NAME", MinWidth: 20, MaxWidth: 60, Highlight: true},
	{Name: "READY", MinWidth: 7, Align: AlignRight},
	{Name: "UP-TO-DATE", MinWidth: 10, Align: AlignRight},
	{Name: "AVAILABLE", MinWidth: 9, Align: AlignRight},
	{Name: "IMAGES", MinWidth: 30, MaxWidth: 80, Wide: true},
	{Name: "SELECTOR", MinWidth: 20, MaxWidth: 60, Wide: true},
	{Name: "AGE", MinWidth: 5, Align: AlignRight, Time: true},
}

// Deployment is the renderer for Deployment resources.
type Deployment struct {
	*BaseRenderer
}

// NewDeployment creates a new Deployment renderer.
func NewDeployment() *Deployment {
	return &Deployment{
		BaseRenderer: NewBaseRenderer(DeploymentHeader),
	}
}

// RenderDeployment renders a Deployment to a Row.
func (d *Deployment) RenderDeployment(deploy *appsv1.Deployment) Row {
	ns := deploy.Namespace
	name := deploy.Name
	id := ns + "/" + name

	desired := int32(0)
	if deploy.Spec.Replicas != nil {
		desired = *deploy.Spec.Replicas
	}
	ready := deploy.Status.ReadyReplicas
	upToDate := deploy.Status.UpdatedReplicas
	available := deploy.Status.AvailableReplicas

	// Get container images
	var images []string
	for _, c := range deploy.Spec.Template.Spec.Containers {
		images = append(images, c.Image)
	}

	// Get selector
	selector := formatLabelSelector(deploy.Spec.Selector)

	age := FormatAge(deploy.CreationTimestamp.Time)

	return Row{
		ID: id,
		Fields: []string{
			ns,
			name,
			fmt.Sprintf("%d/%d", ready, desired),
			fmt.Sprintf("%d", upToDate),
			fmt.Sprintf("%d", available),
			truncateImages(images),
			selector,
			age,
		},
	}
}

// ColorerFunc returns a colorer for Deployment rows.
func (d *Deployment) ColorerFunc() ColorerFunc {
	return func(ns string, row Row) tcell.Color {
		if len(row.Fields) < 3 {
			return tcell.ColorDefault
		}
		ready := row.Fields[2] // READY column
		return ReadyColor(ready)
	}
}

// formatLabelSelector formats a label selector to string.
func formatLabelSelector(selector *metav1.LabelSelector) string {
	if selector == nil {
		return "<none>"
	}
	var parts []string
	for k, v := range selector.MatchLabels {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	if len(parts) == 0 {
		return "<none>"
	}
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ","
		}
		result += p
	}
	return result
}

// truncateImages formats container images.
func truncateImages(images []string) string {
	if len(images) == 0 {
		return "<none>"
	}
	result := images[0]
	if len(images) > 1 {
		result += fmt.Sprintf(" (+%d)", len(images)-1)
	}
	return Truncate(result, 80)
}
