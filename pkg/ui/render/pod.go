package render

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	corev1 "k8s.io/api/core/v1"
)

// PodHeader defines the columns for Pod resources.
var PodHeader = Header{
	{Name: "NAMESPACE", MinWidth: 10, MaxWidth: 30},
	{Name: "NAME", MinWidth: 20, MaxWidth: 60, Highlight: true},
	{Name: "READY", MinWidth: 5, Align: AlignRight},
	{Name: "STATUS", MinWidth: 10, MaxWidth: 20},
	{Name: "RESTARTS", MinWidth: 8, Align: AlignRight},
	{Name: "CPU", MinWidth: 6, Align: AlignRight, MX: true},
	{Name: "MEM", MinWidth: 6, Align: AlignRight, MX: true},
	{Name: "IP", MinWidth: 15, MaxWidth: 20, Wide: true},
	{Name: "NODE", MinWidth: 15, MaxWidth: 30, Wide: true},
	{Name: "AGE", MinWidth: 5, Align: AlignRight, Time: true},
}

// Pod is the renderer for Pod resources.
type Pod struct {
	*BaseRenderer
}

// NewPod creates a new Pod renderer.
func NewPod() *Pod {
	return &Pod{
		BaseRenderer: NewBaseRenderer(PodHeader),
	}
}

// RenderPod renders a Pod to a Row.
func (p *Pod) RenderPod(pod *corev1.Pod, metrics *PodMetrics) Row {
	ns := pod.Namespace
	name := pod.Name
	id := ns + "/" + name

	ready := p.readyCount(pod)
	status := p.phase(pod)
	restarts := p.restartCount(pod)
	cpu := "-"
	mem := "-"
	if metrics != nil {
		cpu = metrics.CPU
		mem = metrics.Memory
	}
	ip := pod.Status.PodIP
	if ip == "" {
		ip = "<none>"
	}
	node := pod.Spec.NodeName
	if node == "" {
		node = "<none>"
	}
	age := FormatAge(pod.CreationTimestamp.Time)

	return Row{
		ID: id,
		Fields: []string{
			ns,
			name,
			ready,
			status,
			restarts,
			cpu,
			mem,
			ip,
			node,
			age,
		},
	}
}

// ColorerFunc returns a colorer for Pod rows.
func (p *Pod) ColorerFunc() ColorerFunc {
	return func(ns string, row Row) tcell.Color {
		if len(row.Fields) < 4 {
			return tcell.ColorDefault
		}
		status := row.Fields[3] // STATUS column
		return StatusColor(status)
	}
}

// readyCount returns the ready container count as "ready/total".
func (p *Pod) readyCount(pod *corev1.Pod) string {
	total := len(pod.Spec.Containers)
	ready := 0
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Ready {
			ready++
		}
	}
	return fmt.Sprintf("%d/%d", ready, total)
}

// phase returns the pod phase/status.
func (p *Pod) phase(pod *corev1.Pod) string {
	status := string(pod.Status.Phase)

	// Check for terminating
	if pod.DeletionTimestamp != nil {
		return "Terminating"
	}

	// Check init container statuses
	for _, cs := range pod.Status.InitContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			return "Init:" + cs.State.Waiting.Reason
		}
		if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 {
			return "Init:Error"
		}
	}

	// Check container statuses for more specific status
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			return cs.State.Waiting.Reason
		}
		if cs.State.Terminated != nil {
			if cs.State.Terminated.Reason != "" {
				return cs.State.Terminated.Reason
			}
			if cs.State.Terminated.ExitCode != 0 {
				return fmt.Sprintf("Exit:%d", cs.State.Terminated.ExitCode)
			}
		}
	}

	// Check conditions
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodScheduled && c.Status == corev1.ConditionFalse {
			return "Unschedulable"
		}
	}

	return status
}

// restartCount returns the total restart count.
func (p *Pod) restartCount(pod *corev1.Pod) string {
	var restarts int32
	for _, cs := range pod.Status.ContainerStatuses {
		restarts += cs.RestartCount
	}
	return fmt.Sprintf("%d", restarts)
}

// PodMetrics holds CPU and memory metrics for a pod.
type PodMetrics struct {
	CPU    string
	Memory string
}

// ContainerRenderer renders container information.
type ContainerRenderer struct{}

// RenderContainers renders pod containers to rows.
func (c *ContainerRenderer) RenderContainers(pod *corev1.Pod) []Row {
	var rows []Row

	for _, container := range pod.Spec.Containers {
		// Find status
		var cs *corev1.ContainerStatus
		for i := range pod.Status.ContainerStatuses {
			if pod.Status.ContainerStatuses[i].Name == container.Name {
				cs = &pod.Status.ContainerStatuses[i]
				break
			}
		}

		status := "Waiting"
		ready := "false"
		restarts := "0"

		if cs != nil {
			restarts = fmt.Sprintf("%d", cs.RestartCount)
			if cs.Ready {
				ready = "true"
			}
			if cs.State.Running != nil {
				status = "Running"
			} else if cs.State.Terminated != nil {
				status = cs.State.Terminated.Reason
				if status == "" {
					status = "Terminated"
				}
			} else if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
				status = cs.State.Waiting.Reason
			}
		}

		rows = append(rows, Row{
			ID: container.Name,
			Fields: []string{
				container.Name,
				container.Image,
				status,
				ready,
				restarts,
				strings.Join(containerPorts(container), ","),
			},
		})
	}

	return rows
}

// containerPorts extracts port strings from a container.
func containerPorts(c corev1.Container) []string {
	var ports []string
	for _, p := range c.Ports {
		port := fmt.Sprintf("%d", p.ContainerPort)
		if p.Name != "" {
			port = p.Name + ":" + port
		}
		if p.Protocol != corev1.ProtocolTCP {
			port += "/" + string(p.Protocol)
		}
		ports = append(ports, port)
	}
	return ports
}
