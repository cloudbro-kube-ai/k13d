package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// PulseView shows cluster health overview (k9s :pulses equivalent)
type PulseView struct {
	*tview.TextView
	app *App
}

// NewPulseView creates a new pulse view showing cluster health overview
func NewPulseView(app *App) *PulseView {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	tv.SetBorder(true).
		SetTitle(" Cluster Pulse (Esc:close  r:refresh) ").
		SetTitleAlign(tview.AlignLeft)

	return &PulseView{
		TextView: tv,
		app:      app,
	}
}

// PulseData holds pre-fetched cluster data for rendering the pulse view.
// This struct enables testing the rendering logic independently of K8s API calls.
type PulseData struct {
	// Pod counts by phase
	PodsRunning int
	PodsPending int
	PodsFailed  int
	PodsOther   int
	PodsTotal   int

	// Deployment counts
	DeploysReady    int
	DeploysUpdating int
	DeploysTotal    int

	// StatefulSet counts
	STSReady int
	STSTotal int

	// DaemonSet counts
	DSReady int
	DSTotal int

	// Job counts
	JobsComplete int
	JobsActive   int
	JobsFailed   int
	JobsTotal    int

	// Node counts
	NodesReady    int
	NodesNotReady int
	NodesTotal    int

	// CPU metrics (millicores)
	CPUUsed     int64 // millicores used
	CPUCapacity int64 // millicores capacity
	CPUAvail    bool  // true if metrics are available

	// Memory metrics (MiB)
	MemUsed     int64 // MiB used
	MemCapacity int64 // MiB capacity
	MemAvail    bool  // true if metrics are available

	// Recent events (last 5 warning/normal events)
	Events []PulseEvent
}

// PulseEvent is a simplified event for the pulse view
type PulseEvent struct {
	Type    string // "Normal" or "Warning"
	Reason  string
	Message string
	Age     string
}

// Refresh fetches cluster data and renders the pulse dashboard
func (p *PulseView) Refresh(ctx context.Context) {
	if p.app.k8s == nil {
		p.SetText("[red]Kubernetes client not available[white]")
		return
	}

	p.SetText("[yellow]Loading cluster data...[white]")

	data := p.fetchData(ctx)
	content := RenderPulse(data)
	p.SetText(content)
}

// fetchData gathers all cluster data for the pulse view
func (p *PulseView) fetchData(ctx context.Context) PulseData {
	var data PulseData
	k := p.app.k8s

	p.app.mx.RLock()
	ns := p.app.currentNamespace
	p.app.mx.RUnlock()

	// Pods
	if pods, err := k.ListPods(ctx, ns); err == nil {
		data.PodsTotal = len(pods)
		for _, pod := range pods {
			switch pod.Status.Phase {
			case "Running":
				data.PodsRunning++
			case "Pending":
				data.PodsPending++
			case "Failed":
				data.PodsFailed++
			default:
				data.PodsOther++
			}
		}
	}

	// Deployments
	if deps, err := k.ListDeployments(ctx, ns); err == nil {
		data.DeploysTotal = len(deps)
		for _, dep := range deps {
			desired := int32(1)
			if dep.Spec.Replicas != nil {
				desired = *dep.Spec.Replicas
			}
			if dep.Status.ReadyReplicas >= desired && dep.Status.UnavailableReplicas == 0 {
				data.DeploysReady++
			} else {
				data.DeploysUpdating++
			}
		}
	}

	// StatefulSets
	if stses, err := k.ListStatefulSets(ctx, ns); err == nil {
		data.STSTotal = len(stses)
		for _, sts := range stses {
			desired := int32(1)
			if sts.Spec.Replicas != nil {
				desired = *sts.Spec.Replicas
			}
			if sts.Status.ReadyReplicas >= desired {
				data.STSReady++
			}
		}
	}

	// DaemonSets
	if dss, err := k.ListDaemonSets(ctx, ns); err == nil {
		data.DSTotal = len(dss)
		for _, ds := range dss {
			if ds.Status.NumberReady >= ds.Status.DesiredNumberScheduled {
				data.DSReady++
			}
		}
	}

	// Jobs
	if jobs, err := k.ListJobs(ctx, ns); err == nil {
		data.JobsTotal = len(jobs)
		for _, job := range jobs {
			if job.Status.Succeeded > 0 && job.Status.Active == 0 {
				data.JobsComplete++
			} else if job.Status.Failed > 0 && job.Status.Active == 0 {
				data.JobsFailed++
			} else if job.Status.Active > 0 {
				data.JobsActive++
			}
		}
	}

	// Nodes
	if nodes, err := k.ListNodes(ctx); err == nil {
		data.NodesTotal = len(nodes)
		for _, node := range nodes {
			ready := false
			for _, cond := range node.Status.Conditions {
				if cond.Type == "Ready" && cond.Status == "True" {
					ready = true
					break
				}
			}
			if ready {
				data.NodesReady++
			} else {
				data.NodesNotReady++
			}
		}

		// CPU/Memory capacity from nodes
		for _, node := range nodes {
			if cpu, ok := node.Status.Allocatable["cpu"]; ok {
				data.CPUCapacity += cpu.MilliValue()
			}
			if mem, ok := node.Status.Allocatable["memory"]; ok {
				data.MemCapacity += mem.Value() / 1024 / 1024 // MiB
			}
		}
	}

	// Node metrics (usage)
	if nodeMetrics, err := k.GetNodeMetrics(ctx); err == nil && len(nodeMetrics) > 0 {
		data.CPUAvail = true
		data.MemAvail = true
		for _, m := range nodeMetrics {
			data.CPUUsed += m[0] // millicores
			data.MemUsed += m[1] // MiB
		}
	}

	// Events (last 5 most recent)
	if events, err := k.ListEvents(ctx, ns); err == nil {
		// Sort by last timestamp descending
		sort.Slice(events, func(i, j int) bool {
			return events[i].LastTimestamp.Time.After(events[j].LastTimestamp.Time)
		})
		count := 0
		for _, ev := range events {
			if count >= 5 {
				break
			}
			data.Events = append(data.Events, PulseEvent{
				Type:    ev.Type,
				Reason:  ev.Reason,
				Message: truncatePulseString(ev.Message, 50),
				Age:     formatPulseAge(ev.LastTimestamp.Time),
			})
			count++
		}
	}

	return data
}

// RenderPulse renders pulse data to a tview-compatible string with color tags.
// Exported for testing.
func RenderPulse(data PulseData) string {
	var sb strings.Builder

	sb.WriteString(" [::b]Cluster Pulse[::-]\n\n")

	// Resource status lines
	sb.WriteString(fmt.Sprintf("  Pods:    [green]✓ %d Running[white]", data.PodsRunning))
	if data.PodsPending > 0 {
		sb.WriteString(fmt.Sprintf("  [yellow]⚠ %d Pending[white]", data.PodsPending))
	}
	if data.PodsFailed > 0 {
		sb.WriteString(fmt.Sprintf("  [red]✗ %d Failed[white]", data.PodsFailed))
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("  Deploy:  [green]✓ %d Ready[white]", data.DeploysReady))
	if data.DeploysUpdating > 0 {
		sb.WriteString(fmt.Sprintf("  [yellow]⚠ %d Updating[white]", data.DeploysUpdating))
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("  STS:     [green]✓ %d Ready[white]", data.STSReady))
	if data.STSTotal-data.STSReady > 0 {
		sb.WriteString(fmt.Sprintf("  [yellow]⚠ %d NotReady[white]", data.STSTotal-data.STSReady))
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("  DS:      [green]✓ %d Ready[white]", data.DSReady))
	if data.DSTotal-data.DSReady > 0 {
		sb.WriteString(fmt.Sprintf("  [yellow]⚠ %d NotReady[white]", data.DSTotal-data.DSReady))
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("  Jobs:    [green]✓ %d Complete[white]", data.JobsComplete))
	if data.JobsActive > 0 {
		sb.WriteString(fmt.Sprintf("  [yellow]⚠ %d Active[white]", data.JobsActive))
	}
	if data.JobsFailed > 0 {
		sb.WriteString(fmt.Sprintf("  [red]✗ %d Failed[white]", data.JobsFailed))
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("  Nodes:   [green]✓ %d Ready[white]", data.NodesReady))
	if data.NodesNotReady > 0 {
		sb.WriteString(fmt.Sprintf("  [red]✗ %d NotReady[white]", data.NodesNotReady))
	}
	sb.WriteString("\n")

	// Metrics bars
	sb.WriteString("\n")
	if data.CPUAvail && data.CPUCapacity > 0 {
		pct := float64(data.CPUUsed) / float64(data.CPUCapacity) * 100
		bar := renderBar(pct, 20)
		sb.WriteString(fmt.Sprintf("  CPU:    %s %.0f%% (%.1f/%.1f cores)\n",
			bar, pct, float64(data.CPUUsed)/1000, float64(data.CPUCapacity)/1000))
	} else {
		sb.WriteString("  CPU:    [gray]N/A (metrics-server not available)[white]\n")
	}

	if data.MemAvail && data.MemCapacity > 0 {
		pct := float64(data.MemUsed) / float64(data.MemCapacity) * 100
		bar := renderBar(pct, 20)
		sb.WriteString(fmt.Sprintf("  Memory: %s %.0f%% (%.1f/%.1f GiB)\n",
			bar, pct, float64(data.MemUsed)/1024, float64(data.MemCapacity)/1024))
	} else {
		sb.WriteString("  Memory: [gray]N/A (metrics-server not available)[white]\n")
	}

	// Recent events
	sb.WriteString("\n  [::b]Recent Events[::-]\n")
	if len(data.Events) == 0 {
		sb.WriteString("  [gray]No recent events[white]\n")
	} else {
		for _, ev := range data.Events {
			icon := "[green]✓[white]"
			if ev.Type == "Warning" {
				icon = "[yellow]⚠[white]"
			}
			sb.WriteString(fmt.Sprintf("  %s %s: %s", icon, ev.Reason, ev.Message))
			if ev.Age != "" {
				sb.WriteString(fmt.Sprintf(" [gray](%s)[white]", ev.Age))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// renderBar creates an ASCII progress bar with color
func renderBar(pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int(pct / 100 * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled

	// Color based on usage level
	color := "[green]"
	if pct >= 80 {
		color = "[red]"
	} else if pct >= 60 {
		color = "[yellow]"
	}

	return color + strings.Repeat("█", filled) + "[gray]" + strings.Repeat("░", empty) + "[white]"
}

// truncatePulseString truncates a string to maxLen, adding "..." if needed
func truncatePulseString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// formatPulseAge formats a time as a human-readable age
func formatPulseAge(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// showPulse displays the Cluster Pulse modal
func (a *App) showPulse() {
	pulse := NewPulseView(a)

	pulse.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			a.closeModal("pulse")
			a.SetFocus(a.table)
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'r' {
				a.safeGo("pulse-refresh", func() {
					pulse.Refresh(a.getAppContext())
				})
				return nil
			}
			if event.Rune() == 'q' {
				a.closeModal("pulse")
				a.SetFocus(a.table)
				return nil
			}
		}
		return event
	})

	a.showModal("pulse", centered(pulse, 65, 28), true)
	a.SetFocus(pulse)

	// Fetch data asynchronously
	a.safeGo("pulse-initial", func() {
		pulse.Refresh(a.getAppContext())
	})
}
