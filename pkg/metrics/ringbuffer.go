package metrics

import (
	"sync"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/db"
)

// RingBuffer is a generic, thread-safe circular buffer for time-series data.
// It overwrites the oldest entry when full.
type RingBuffer[T any] struct {
	mu    sync.RWMutex
	items []T
	head  int // next write position
	count int // number of valid items (0..cap)
	cap   int
}

// NewRingBuffer creates a ring buffer with the given capacity.
func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	return &RingBuffer[T]{
		items: make([]T, capacity),
		cap:   capacity,
	}
}

// Push appends an item, overwriting the oldest if full.
func (rb *RingBuffer[T]) Push(item T) {
	rb.mu.Lock()
	rb.items[rb.head] = item
	rb.head = (rb.head + 1) % rb.cap
	if rb.count < rb.cap {
		rb.count++
	}
	rb.mu.Unlock()
}

// PushBatch appends multiple items with a single lock acquisition.
func (rb *RingBuffer[T]) PushBatch(items []T) {
	if len(items) == 0 {
		return
	}
	rb.mu.Lock()
	for _, item := range items {
		rb.items[rb.head] = item
		rb.head = (rb.head + 1) % rb.cap
		if rb.count < rb.cap {
			rb.count++
		}
	}
	rb.mu.Unlock()
}

// Snapshot returns a copy of all valid items in chronological order (oldest first).
func (rb *RingBuffer[T]) Snapshot() []T {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if rb.count == 0 {
		return nil
	}

	result := make([]T, rb.count)
	start := (rb.head - rb.count + rb.cap) % rb.cap
	for i := 0; i < rb.count; i++ {
		result[i] = rb.items[(start+i)%rb.cap]
	}
	return result
}

// Len returns the number of valid items.
func (rb *RingBuffer[T]) Len() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}

// Clear empties the buffer.
func (rb *RingBuffer[T]) Clear() {
	rb.mu.Lock()
	rb.head = 0
	rb.count = 0
	rb.mu.Unlock()
}

// DefaultRingCapacity is the default entries per ring buffer.
// At 1-minute collection interval, 1800 = 30 hours of data.
const DefaultRingCapacity = 1800

// MetricsCache provides in-memory caching of recent metrics using ring buffers.
type MetricsCache struct {
	mu       sync.RWMutex
	cluster  *RingBuffer[db.ClusterMetrics]
	nodes    map[string]*RingBuffer[db.NodeMetrics] // keyed by nodeName
	pods     map[string]*RingBuffer[db.PodMetrics]  // keyed by "namespace/podName"
	capacity int
}

// NewMetricsCache creates a new in-memory metrics cache.
func NewMetricsCache(capacity int) *MetricsCache {
	if capacity <= 0 {
		capacity = DefaultRingCapacity
	}
	return &MetricsCache{
		cluster:  NewRingBuffer[db.ClusterMetrics](capacity),
		nodes:    make(map[string]*RingBuffer[db.NodeMetrics]),
		pods:     make(map[string]*RingBuffer[db.PodMetrics]),
		capacity: capacity,
	}
}

// PushCluster adds a cluster metrics entry.
func (mc *MetricsCache) PushCluster(m db.ClusterMetrics) {
	mc.cluster.Push(m)
}

// PushNodes adds a batch of node metrics entries.
func (mc *MetricsCache) PushNodes(metrics []db.NodeMetrics) {
	mc.mu.Lock()
	for _, m := range metrics {
		ring, ok := mc.nodes[m.NodeName]
		if !ok {
			ring = NewRingBuffer[db.NodeMetrics](mc.capacity)
			mc.nodes[m.NodeName] = ring
		}
		ring.Push(m)
	}
	mc.mu.Unlock()
}

// PushPods adds a batch of pod metrics entries.
func (mc *MetricsCache) PushPods(metrics []db.PodMetrics) {
	mc.mu.Lock()
	for _, m := range metrics {
		key := m.Namespace + "/" + m.PodName
		ring, ok := mc.pods[key]
		if !ok {
			ring = NewRingBuffer[db.PodMetrics](mc.capacity)
			mc.pods[key] = ring
		}
		ring.Push(m)
	}
	mc.mu.Unlock()
}

// GetClusterMetrics returns cluster metrics within the time range from cache.
// Returns nil if the cache is empty (caller should fall back to SQLite).
func (mc *MetricsCache) GetClusterMetrics(contextName string, start, end time.Time, limit int) []db.ClusterMetrics {
	all := mc.cluster.Snapshot()
	if len(all) == 0 {
		return nil
	}

	var result []db.ClusterMetrics
	// Reverse for newest-first (matches SQLite ORDER BY timestamp DESC)
	for i := len(all) - 1; i >= 0; i-- {
		m := all[i]
		if m.Context != contextName {
			continue
		}
		if m.Timestamp.Before(start) || m.Timestamp.After(end) {
			continue
		}
		result = append(result, m)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

// GetNodeMetrics returns node metrics for a specific node within the time range.
// Returns nil if the node is not in cache.
func (mc *MetricsCache) GetNodeMetrics(contextName, nodeName string, start, end time.Time, limit int) []db.NodeMetrics {
	mc.mu.RLock()
	ring, ok := mc.nodes[nodeName]
	mc.mu.RUnlock()
	if !ok {
		return nil
	}

	all := ring.Snapshot()
	if len(all) == 0 {
		return nil
	}

	var result []db.NodeMetrics
	for i := len(all) - 1; i >= 0; i-- {
		m := all[i]
		if m.Context != contextName {
			continue
		}
		if m.Timestamp.Before(start) || m.Timestamp.After(end) {
			continue
		}
		result = append(result, m)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

// GetPodMetrics returns pod metrics for a specific pod within the time range.
// Returns nil if the pod is not in cache.
func (mc *MetricsCache) GetPodMetrics(contextName, namespace, podName string, start, end time.Time, limit int) []db.PodMetrics {
	key := namespace + "/" + podName
	mc.mu.RLock()
	ring, ok := mc.pods[key]
	mc.mu.RUnlock()
	if !ok {
		return nil
	}

	all := ring.Snapshot()
	if len(all) == 0 {
		return nil
	}

	var result []db.PodMetrics
	for i := len(all) - 1; i >= 0; i-- {
		m := all[i]
		if m.Context != contextName {
			continue
		}
		if m.Timestamp.Before(start) || m.Timestamp.After(end) {
			continue
		}
		result = append(result, m)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

// GetLatestClusterMetrics returns the most recent cluster metrics entry from cache.
func (mc *MetricsCache) GetLatestClusterMetrics(contextName string) *db.ClusterMetrics {
	all := mc.cluster.Snapshot()
	for i := len(all) - 1; i >= 0; i-- {
		if all[i].Context == contextName {
			return &all[i]
		}
	}
	return nil
}
