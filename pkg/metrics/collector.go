// Package metrics provides periodic metrics collection for time-series data
package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/db"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/log"
	corev1 "k8s.io/api/core/v1"
)

// CollectorConfig holds configuration for the metrics collector
type CollectorConfig struct {
	// Interval between collections (default: 1 minute)
	Interval time.Duration
	// Retention period for old metrics (default: 7 days)
	Retention time.Duration
	// CleanupInterval for removing old metrics (default: 1 hour)
	CleanupInterval time.Duration
	// Namespace to filter (empty for all namespaces)
	Namespace string
}

// DefaultConfig returns default collector configuration
func DefaultConfig() *CollectorConfig {
	return &CollectorConfig{
		Interval:        1 * time.Minute,
		Retention:       7 * 24 * time.Hour, // 7 days
		CleanupInterval: 1 * time.Hour,
		Namespace:       "",
	}
}

// Collector collects and stores Kubernetes metrics periodically
type Collector struct {
	k8sClient *k8s.Client
	store     *db.MetricsStore
	config    *CollectorConfig
	stopCh    chan struct{}
	running   bool
	mu        sync.RWMutex
}

// NewCollector creates a new metrics collector
func NewCollector(k8sClient *k8s.Client, config *CollectorConfig) (*Collector, error) {
	if config == nil {
		config = DefaultConfig()
	}

	store, err := db.NewMetricsStore()
	if err != nil {
		return nil, err
	}

	return &Collector{
		k8sClient: k8sClient,
		store:     store,
		config:    config,
		stopCh:    make(chan struct{}),
	}, nil
}

// Start begins periodic metrics collection
func (c *Collector) Start() {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.stopCh = make(chan struct{})
	c.mu.Unlock()

	log.Infof("Starting metrics collector (interval: %s, retention: %s)", c.config.Interval, c.config.Retention)

	// Initial collection
	go c.collectOnce()

	// Start periodic collection
	go c.runCollector()

	// Start cleanup routine
	go c.runCleanup()
}

// Stop stops the metrics collector
func (c *Collector) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return
	}

	close(c.stopCh)
	c.running = false
	log.Infof("Metrics collector stopped")
}

// IsRunning returns whether the collector is running
func (c *Collector) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// GetStore returns the metrics store for querying
func (c *Collector) GetStore() *db.MetricsStore {
	return c.store
}

// CollectNow triggers an immediate metrics collection
func (c *Collector) CollectNow() error {
	return c.collectOnce()
}

func (c *Collector) runCollector() {
	ticker := time.NewTicker(c.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.collectOnce(); err != nil {
				log.Errorf("Failed to collect metrics: %v", err)
			}
		case <-c.stopCh:
			return
		}
	}
}

func (c *Collector) runCleanup() {
	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := c.store.CleanupOldMetrics(ctx, c.config.Retention); err != nil {
				log.Errorf("Failed to cleanup old metrics: %v", err)
			}
			cancel()
		case <-c.stopCh:
			return
		}
	}
}

func (c *Collector) collectOnce() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	contextName, err := c.k8sClient.GetCurrentContext()
	if err != nil {
		return err
	}

	timestamp := time.Now()

	// Collect cluster-level metrics
	clusterMetrics, err := c.collectClusterMetrics(ctx, contextName, timestamp)
	if err != nil {
		log.Warnf("Failed to collect cluster metrics: %v", err)
	} else if err := c.store.SaveClusterMetrics(ctx, clusterMetrics); err != nil {
		log.Warnf("Failed to save cluster metrics: %v", err)
	}

	// Collect node metrics
	if err := c.collectNodeMetrics(ctx, contextName, timestamp); err != nil {
		log.Warnf("Failed to collect node metrics: %v", err)
	}

	// Collect pod metrics (sampled to avoid too much data)
	if err := c.collectPodMetrics(ctx, contextName, timestamp); err != nil {
		log.Warnf("Failed to collect pod metrics: %v", err)
	}

	log.Debugf("Metrics collection completed for context %s", contextName)
	return nil
}

func (c *Collector) collectClusterMetrics(ctx context.Context, contextName string, timestamp time.Time) (*db.ClusterMetrics, error) {
	metrics := &db.ClusterMetrics{
		Timestamp: timestamp,
		Context:   contextName,
		Namespace: c.config.Namespace,
	}

	// Get nodes
	nodes, err := c.k8sClient.ListNodes(ctx)
	if err == nil {
		metrics.TotalNodes = len(nodes)
		for _, node := range nodes {
			for _, cond := range node.Status.Conditions {
				if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
					metrics.ReadyNodes++
					break
				}
			}
			// Sum capacity
			if cpu := node.Status.Capacity.Cpu(); cpu != nil {
				metrics.TotalCPUMillis += cpu.MilliValue()
			}
			if mem := node.Status.Capacity.Memory(); mem != nil {
				metrics.TotalMemoryMB += mem.Value() / 1024 / 1024
			}
		}
	}

	// Get pods
	pods, err := c.k8sClient.ListPods(ctx, c.config.Namespace)
	if err == nil {
		metrics.TotalPods = len(pods)
		for _, pod := range pods {
			switch pod.Status.Phase {
			case corev1.PodRunning:
				metrics.RunningPods++
			case corev1.PodPending:
				metrics.PendingPods++
			case corev1.PodFailed:
				metrics.FailedPods++
			}
		}
	}

	// Get resource usage from metrics-server
	nodeMetrics, err := c.k8sClient.GetNodeMetrics(ctx)
	if err == nil {
		for _, usage := range nodeMetrics {
			metrics.UsedCPUMillis += usage[0]
			metrics.UsedMemoryMB += usage[1]
		}
	}

	// Get deployments
	deployments, err := c.k8sClient.ListDeployments(ctx, c.config.Namespace)
	if err == nil {
		metrics.TotalDeployments = len(deployments)
		for _, dep := range deployments {
			if dep.Status.ReadyReplicas == dep.Status.Replicas && dep.Status.Replicas > 0 {
				metrics.ReadyDeployments++
			}
		}
	}

	return metrics, nil
}

func (c *Collector) collectNodeMetrics(ctx context.Context, contextName string, timestamp time.Time) error {
	nodes, err := c.k8sClient.ListNodes(ctx)
	if err != nil {
		return err
	}

	// Get metrics from metrics-server
	nodeMetricsMap, _ := c.k8sClient.GetNodeMetrics(ctx)

	// Count pods per node
	pods, _ := c.k8sClient.ListPods(ctx, "")
	podCountByNode := make(map[string]int)
	for _, pod := range pods {
		if pod.Spec.NodeName != "" {
			podCountByNode[pod.Spec.NodeName]++
		}
	}

	for _, node := range nodes {
		nm := &db.NodeMetrics{
			Timestamp: timestamp,
			Context:   contextName,
			NodeName:  node.Name,
			PodCount:  podCountByNode[node.Name],
		}

		// Check readiness
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady {
				nm.IsReady = cond.Status == corev1.ConditionTrue
				break
			}
		}
		nm.IsSchedulable = !node.Spec.Unschedulable

		// Get capacity
		if cpu := node.Status.Capacity.Cpu(); cpu != nil {
			nm.CPUCapacity = cpu.MilliValue()
		}
		if mem := node.Status.Capacity.Memory(); mem != nil {
			nm.MemCapacity = mem.Value() / 1024 / 1024
		}

		// Get usage from metrics-server
		if usage, ok := nodeMetricsMap[node.Name]; ok {
			nm.CPUMillis = usage[0]
			nm.MemoryMB = usage[1]
		}

		if err := c.store.SaveNodeMetrics(ctx, nm); err != nil {
			log.Warnf("Failed to save node metrics for %s: %v", node.Name, err)
		}
	}

	return nil
}

func (c *Collector) collectPodMetrics(ctx context.Context, contextName string, timestamp time.Time) error {
	pods, err := c.k8sClient.ListPods(ctx, c.config.Namespace)
	if err != nil {
		return err
	}

	// Get metrics from metrics-server
	podMetricsMap, _ := c.k8sClient.GetPodMetrics(ctx, c.config.Namespace)

	// Only store metrics for pods with non-zero usage (to reduce storage)
	for _, pod := range pods {
		// Skip terminated pods
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			continue
		}

		pm := &db.PodMetrics{
			Timestamp: timestamp,
			Context:   contextName,
			Namespace: pod.Namespace,
			PodName:   pod.Name,
			Status:    string(pod.Status.Phase),
		}

		// Count restarts
		for _, cs := range pod.Status.ContainerStatuses {
			pm.Restarts += int(cs.RestartCount)
		}

		// Get usage from metrics-server
		if usage, ok := podMetricsMap[pod.Name]; ok {
			pm.CPUMillis = usage[0]
			pm.MemoryMB = usage[1]
		}

		// Only save if there's actual usage or it's a running pod
		if pm.CPUMillis > 0 || pm.MemoryMB > 0 || pod.Status.Phase == corev1.PodRunning {
			if err := c.store.SavePodMetrics(ctx, pm); err != nil {
				log.Warnf("Failed to save pod metrics for %s/%s: %v", pod.Namespace, pod.Name, err)
			}
		}
	}

	return nil
}
