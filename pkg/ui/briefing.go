package ui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	corev1 "k8s.io/api/core/v1"
)

// BriefingData holds the computed briefing information
type BriefingData struct {
	HealthScore      int    // 0-100
	HealthStatus     string // "healthy", "warning", "critical"
	TotalPods        int
	RunningPods      int
	PendingPods      int
	FailedPods       int
	TotalNodes       int
	ReadyNodes       int
	TotalDeployments int
	ReadyDeployments int
	CPUPercent       float64
	MemoryPercent    float64
	Namespace        string   // "" for all namespaces
	Alerts           []string // Warning/error messages
	ContextName      string
	ClusterName      string
}

// BriefingPanel displays a natural language cluster health summary
type BriefingPanel struct {
	*tview.TextView
	app         *App
	visible     bool
	pulseActive bool
	pulseIdx    int
	pulseChars  []rune
	data        *BriefingData
	mu          sync.RWMutex
	stopPulse   chan struct{}
}

// NewBriefingPanel creates a new briefing panel
func NewBriefingPanel(app *App) *BriefingPanel {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false).
		SetWrap(true)
	tv.SetBorder(true).
		SetTitle(" Cluster Briefing ").
		SetBorderColor(tcell.ColorDarkGreen)
	tv.SetBackgroundColor(tcell.ColorDefault)

	b := &BriefingPanel{
		TextView:   tv,
		app:        app,
		visible:    true, // Default ON per user preference
		pulseChars: []rune{'●', '◐', '○', '◑'},
		stopPulse:  make(chan struct{}),
	}

	// Initial placeholder text
	b.SetText(" [gray]Loading cluster status...[white]")

	return b
}

// Toggle toggles the visibility of the briefing panel
func (b *BriefingPanel) Toggle() {
	b.mu.Lock()
	b.visible = !b.visible
	visible := b.visible
	b.mu.Unlock()

	if visible {
		b.startPulse()
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// Silently recover - briefing update failure is non-critical
				}
			}()
			ctx, cancel := context.WithTimeout(b.app.getAppContext(), 10*time.Second)
			defer cancel()
			b.Update(ctx)
		}()
	} else {
		b.stopPulseAnimation()
	}
}

// IsVisible returns true if the panel is visible
func (b *BriefingPanel) IsVisible() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.visible
}

// Update fetches cluster data and updates the briefing text
func (b *BriefingPanel) Update(ctx context.Context) error {
	data, err := b.fetchData(ctx)
	if err != nil {
		b.app.QueueUpdateDraw(func() {
			b.SetText(fmt.Sprintf(" [red]Error loading cluster data: %v[white]", err))
		})
		return err
	}

	b.mu.Lock()
	b.data = data
	b.mu.Unlock()

	b.updateDisplay()
	return nil
}

// fetchData collects all data needed for the briefing (parallelized)
func (b *BriefingPanel) fetchData(ctx context.Context) (*BriefingData, error) {
	if b.app.k8s == nil {
		return nil, fmt.Errorf("K8s client not available")
	}

	b.app.mx.RLock()
	ns := b.app.currentNamespace
	b.app.mx.RUnlock()

	data := &BriefingData{
		Namespace: ns,
	}

	// Get context info (fast, non-API call)
	ctxName, cluster, _, err := b.app.k8s.GetContextInfo()
	if err == nil {
		data.ContextName = ctxName
		data.ClusterName = cluster
	}

	// Fetch pods, nodes, deployments, and metrics in parallel
	type podsResult struct {
		pods []corev1.Pod
		err  error
	}
	type nodesResult struct {
		nodes []corev1.Node
		err   error
	}
	podsCh := make(chan podsResult, 1)
	nodesCh := make(chan nodesResult, 1)
	deployReadyCh := make(chan [2]int, 1) // [total, ready]

	// Goroutine 1: Fetch pods
	go func() {
		defer func() {
			if r := recover(); r != nil {
				podsCh <- podsResult{err: fmt.Errorf("panic: %v", r)}
			}
		}()
		pods, err := b.app.k8s.ListPods(ctx, ns)
		podsCh <- podsResult{pods: pods, err: err}
	}()

	// Goroutine 2: Fetch nodes (always cluster-wide)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				nodesCh <- nodesResult{err: fmt.Errorf("panic: %v", r)}
			}
		}()
		nodes, err := b.app.k8s.ListNodes(ctx)
		nodesCh <- nodesResult{nodes: nodes, err: err}
	}()

	// Goroutine 3: Fetch deployments
	go func() {
		defer func() {
			if r := recover(); r != nil {
				deployReadyCh <- [2]int{0, 0}
			}
		}()
		deployments, err := b.app.k8s.ListDeployments(ctx, ns)
		if err != nil {
			deployReadyCh <- [2]int{0, 0}
			return
		}
		total := len(deployments)
		ready := 0
		for _, d := range deployments {
			replicas := int32(1)
			if d.Spec.Replicas != nil {
				replicas = *d.Spec.Replicas
			}
			if d.Status.ReadyReplicas >= replicas {
				ready++
			}
		}
		deployReadyCh <- [2]int{total, ready}
	}()

	// Goroutine 4: Fetch metrics (optional, may not be available)
	type metricsResult struct {
		podMetrics  map[string][]int64
		nodeMetrics map[string][]int64
	}
	metricsResultCh := make(chan metricsResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				metricsResultCh <- metricsResult{}
			}
		}()
		podMetrics, _ := b.app.k8s.GetPodMetrics(ctx, ns)
		nodeMetrics, _ := b.app.k8s.GetNodeMetrics(ctx)
		metricsResultCh <- metricsResult{podMetrics: podMetrics, nodeMetrics: nodeMetrics}
	}()

	// Collect results
	pr := <-podsCh
	if pr.err == nil {
		pods := pr.pods
		data.TotalPods = len(pods)
		for _, p := range pods {
			switch p.Status.Phase {
			case corev1.PodRunning:
				allReady := true
				for _, cs := range p.Status.ContainerStatuses {
					if !cs.Ready {
						allReady = false
						break
					}
				}
				if allReady {
					data.RunningPods++
				} else {
					data.PendingPods++
				}
			case corev1.PodPending:
				data.PendingPods++
			case corev1.PodFailed:
				data.FailedPods++
			}

			// Check for CrashLoopBackOff
			for _, cs := range p.Status.ContainerStatuses {
				if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
					data.Alerts = append(data.Alerts, fmt.Sprintf("Pod %s/%s in CrashLoopBackOff", p.Namespace, p.Name))
				}
			}
		}
	}

	nr := <-nodesCh
	if nr.err == nil {
		nodes := nr.nodes
		data.TotalNodes = len(nodes)
		for _, n := range nodes {
			for _, c := range n.Status.Conditions {
				if c.Type == corev1.NodeReady && c.Status == corev1.ConditionTrue {
					data.ReadyNodes++
					break
				}
			}
		}
		if data.ReadyNodes < data.TotalNodes {
			notReady := data.TotalNodes - data.ReadyNodes
			data.Alerts = append(data.Alerts, fmt.Sprintf("%d node(s) not ready", notReady))
		}
	}

	dr := <-deployReadyCh
	data.TotalDeployments = dr[0]
	data.ReadyDeployments = dr[1]

	mr := <-metricsResultCh
	podMetrics := mr.podMetrics
	nodeMetrics := mr.nodeMetrics

	// Calculate resource usage from node metrics
	if len(nodeMetrics) > 0 {
		var totalCPU, totalMem int64
		for _, m := range nodeMetrics {
			totalCPU += m[0]
			totalMem += m[1]
		}
		// Rough estimation: assume 4000m CPU and 16Gi memory per node as baseline
		if data.TotalNodes > 0 {
			maxCPU := int64(data.TotalNodes * 4000)
			maxMem := int64(data.TotalNodes * 16384) // 16Gi in Mi
			data.CPUPercent = float64(totalCPU) / float64(maxCPU) * 100
			data.MemoryPercent = float64(totalMem) / float64(maxMem) * 100

			if data.CPUPercent > 90 {
				data.Alerts = append(data.Alerts, fmt.Sprintf("High CPU usage: %.0f%%", data.CPUPercent))
			}
			if data.MemoryPercent > 90 {
				data.Alerts = append(data.Alerts, fmt.Sprintf("High memory usage: %.0f%%", data.MemoryPercent))
			}
		}
	} else if len(podMetrics) > 0 {
		// Fallback: estimate from pod metrics
		var totalCPU, totalMem int64
		for _, m := range podMetrics {
			totalCPU += m[0]
			totalMem += m[1]
		}
		// Very rough estimate
		data.CPUPercent = float64(totalCPU) / float64(data.TotalPods*500) * 100
		data.MemoryPercent = float64(totalMem) / float64(data.TotalPods*512) * 100
	}

	// Calculate health score
	data.HealthScore = calculateHealthScore(data)
	data.HealthStatus = healthStatusFromScore(data.HealthScore)

	// Limit alerts to top 3
	if len(data.Alerts) > 3 {
		data.Alerts = data.Alerts[:3]
	}

	return data, nil
}

// calculateHealthScore computes a health score from 0-100
func calculateHealthScore(data *BriefingData) int {
	score := 100.0

	// Pod health (40% weight)
	if data.TotalPods > 0 {
		podHealth := float64(data.RunningPods) / float64(data.TotalPods)
		score -= (1 - podHealth) * 40
	}

	// Node health (30% weight)
	if data.TotalNodes > 0 {
		nodeHealth := float64(data.ReadyNodes) / float64(data.TotalNodes)
		score -= (1 - nodeHealth) * 30
	}

	// Deployment health (20% weight)
	if data.TotalDeployments > 0 {
		deployHealth := float64(data.ReadyDeployments) / float64(data.TotalDeployments)
		score -= (1 - deployHealth) * 20
	}

	// Resource pressure (10% weight)
	if data.CPUPercent > 90 || data.MemoryPercent > 90 {
		score -= 10
	} else if data.CPUPercent > 80 || data.MemoryPercent > 80 {
		score -= 5
	}

	// Ensure score is within bounds
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return int(score)
}

// healthStatusFromScore converts a health score to a status string
func healthStatusFromScore(score int) string {
	switch {
	case score >= 90:
		return "healthy"
	case score >= 70:
		return "warning"
	default:
		return "critical"
	}
}

// updateDisplay updates the briefing text
func (b *BriefingPanel) updateDisplay() {
	// Check if app is still running to avoid blocking on QueueUpdateDraw
	if b.app == nil || !b.app.IsRunning() {
		return
	}

	b.mu.RLock()
	data := b.data
	pulseIdx := b.pulseIdx
	pulseChars := b.pulseChars
	b.mu.RUnlock()

	if data == nil {
		return
	}

	var sb strings.Builder

	// Line 1: Health overview with pulse indicator
	pulseChar := pulseChars[pulseIdx%len(pulseChars)]
	healthColor := getHealthColor(data.HealthStatus)
	sb.WriteString(fmt.Sprintf(" %s%c %s (%d%%)[white]", healthColor, pulseChar, capitalizeFirst(data.HealthStatus), data.HealthScore))

	// Pod summary
	sb.WriteString(fmt.Sprintf(" • %d pods", data.TotalPods))
	if data.RunningPods < data.TotalPods {
		sb.WriteString(fmt.Sprintf(" (%d running)", data.RunningPods))
	}

	// Node summary
	if data.TotalNodes > 0 {
		if data.ReadyNodes == data.TotalNodes {
			sb.WriteString(fmt.Sprintf(" • %d nodes ready", data.TotalNodes))
		} else {
			sb.WriteString(fmt.Sprintf(" • %d/%d nodes ready", data.ReadyNodes, data.TotalNodes))
		}
	}

	// Line 2: Resources and deployments
	sb.WriteString("\n ")
	if data.CPUPercent > 0 || data.MemoryPercent > 0 {
		cpuColor := "[white]"
		if data.CPUPercent > 90 {
			cpuColor = "[red]"
		} else if data.CPUPercent > 80 {
			cpuColor = "[yellow]"
		}
		memColor := "[white]"
		if data.MemoryPercent > 90 {
			memColor = "[red]"
		} else if data.MemoryPercent > 80 {
			memColor = "[yellow]"
		}
		sb.WriteString(fmt.Sprintf("Resources: %s%.0f%% CPU[white], %s%.0f%% memory[white]", cpuColor, data.CPUPercent, memColor, data.MemoryPercent))
	} else {
		sb.WriteString("[gray]Resources: metrics unavailable[white]")
	}

	// Deployment summary
	if data.TotalDeployments > 0 {
		if data.ReadyDeployments == data.TotalDeployments {
			sb.WriteString(fmt.Sprintf(" • All %d deployments ready", data.TotalDeployments))
		} else {
			sb.WriteString(fmt.Sprintf(" • %d/%d deployments ready", data.ReadyDeployments, data.TotalDeployments))
		}
	}

	// Line 3: Alerts or positive message
	sb.WriteString("\n ")
	if len(data.Alerts) > 0 {
		sb.WriteString("[yellow]")
		for i, alert := range data.Alerts {
			if i > 0 {
				sb.WriteString(" • ")
			}
			sb.WriteString(alert)
		}
		sb.WriteString("[white]")
	} else {
		nsDisplay := "all namespaces"
		if data.Namespace != "" {
			nsDisplay = data.Namespace + " namespace"
		}
		sb.WriteString(fmt.Sprintf("[green]✓ No issues detected in %s[white]", nsDisplay))
	}

	b.app.QueueUpdateDraw(func() {
		b.SetText(sb.String())
	})
}

// getHealthColor returns the color tag for a health status
func getHealthColor(status string) string {
	switch status {
	case "healthy":
		return "[green]"
	case "warning":
		return "[yellow]"
	case "critical":
		return "[red]"
	default:
		return "[white]"
	}
}

// startPulse starts the pulse animation
func (b *BriefingPanel) startPulse() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.pulseActive {
		return
	}
	b.pulseActive = true
	b.stopPulse = make(chan struct{})

	// Capture the channel to avoid race when stopPulse is recreated
	stopCh := b.stopPulse

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Silently recover - pulse animation failure is non-critical
			}
		}()
		ticker := time.NewTicker(400 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				b.mu.Lock()
				if !b.pulseActive {
					b.mu.Unlock()
					return
				}
				b.pulseIdx = (b.pulseIdx + 1) % len(b.pulseChars)
				b.mu.Unlock()

				b.updateDisplay()
			case <-stopCh:
				return
			}
		}
	}()
}

// stopPulseAnimation stops the pulse animation
func (b *BriefingPanel) stopPulseAnimation() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.pulseActive {
		b.pulseActive = false
		close(b.stopPulse)
	}
}

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// UpdateWithAI requests an AI-generated briefing (Shift+B)
func (b *BriefingPanel) UpdateWithAI() {
	if b.app.aiClient == nil || !b.app.aiClient.IsReady() {
		b.app.flashMsg("AI not available for briefing", true)
		return
	}

	b.app.QueueUpdateDraw(func() {
		b.SetText(" [cyan]Generating AI briefing...[white]")
	})

	ctx := b.app.getAppContext()
	data, err := b.fetchData(ctx)
	if err != nil {
		b.app.QueueUpdateDraw(func() {
			b.SetText(fmt.Sprintf(" [red]Error: %v[white]", err))
		})
		return
	}

	prompt := fmt.Sprintf(`You are a Kubernetes cluster health assistant. Generate a 3-line briefing based on this data:
- Health Score: %d/100
- Pods: %d total, %d running, %d pending, %d failed
- Nodes: %d total, %d ready
- Deployments: %d total, %d ready
- CPU: %.0f%%, Memory: %.0f%%
- Alerts: %v
- Namespace: %s (empty means all)

Write a concise, friendly 3-line summary. Use relevant emoji. Be informative but brief.
Example format:
● Cluster healthy (95%%) with 45 pods across 3 nodes
  Resources look good: 45%% CPU, 62%% memory used
  ✓ All systems operational in production`,
		data.HealthScore,
		data.TotalPods, data.RunningPods, data.PendingPods, data.FailedPods,
		data.TotalNodes, data.ReadyNodes,
		data.TotalDeployments, data.ReadyDeployments,
		data.CPUPercent, data.MemoryPercent,
		data.Alerts,
		data.Namespace,
	)

	var response strings.Builder
	err = b.app.aiClient.Ask(ctx, prompt, func(chunk string) {
		response.WriteString(chunk)
		b.app.QueueUpdateDraw(func() {
			b.SetText(" " + response.String())
		})
	})

	if err != nil {
		b.app.QueueUpdateDraw(func() {
			b.SetText(fmt.Sprintf(" [red]AI Error: %v[white]", err))
		})
	}
}
