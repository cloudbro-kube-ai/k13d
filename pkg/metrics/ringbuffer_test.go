package metrics

import (
	"sync"
	"testing"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/db"
)

func TestRingBuffer_Basic(t *testing.T) {
	rb := NewRingBuffer[int](3)

	if rb.Len() != 0 {
		t.Fatalf("Len() = %d, want 0", rb.Len())
	}
	if snap := rb.Snapshot(); snap != nil {
		t.Fatalf("Snapshot() = %v, want nil", snap)
	}

	rb.Push(1)
	rb.Push(2)
	rb.Push(3)

	if rb.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", rb.Len())
	}

	snap := rb.Snapshot()
	if len(snap) != 3 || snap[0] != 1 || snap[1] != 2 || snap[2] != 3 {
		t.Fatalf("Snapshot = %v, want [1 2 3]", snap)
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	rb := NewRingBuffer[int](3)
	rb.Push(1)
	rb.Push(2)
	rb.Push(3)
	rb.Push(4) // overwrites 1

	snap := rb.Snapshot()
	if len(snap) != 3 || snap[0] != 2 || snap[1] != 3 || snap[2] != 4 {
		t.Fatalf("Snapshot after overflow = %v, want [2 3 4]", snap)
	}

	rb.Push(5) // overwrites 2
	rb.Push(6) // overwrites 3
	snap = rb.Snapshot()
	if snap[0] != 4 || snap[1] != 5 || snap[2] != 6 {
		t.Fatalf("Snapshot after double overflow = %v, want [4 5 6]", snap)
	}
}

func TestRingBuffer_PushBatch(t *testing.T) {
	rb := NewRingBuffer[int](5)
	rb.PushBatch([]int{1, 2, 3, 4, 5, 6, 7})

	snap := rb.Snapshot()
	if len(snap) != 5 {
		t.Fatalf("Snapshot len = %d, want 5", len(snap))
	}
	if snap[0] != 3 || snap[4] != 7 {
		t.Fatalf("Snapshot = %v, want [3 4 5 6 7]", snap)
	}
}

func TestRingBuffer_PushBatchEmpty(t *testing.T) {
	rb := NewRingBuffer[int](5)
	rb.PushBatch(nil)
	rb.PushBatch([]int{})
	if rb.Len() != 0 {
		t.Fatalf("Len after empty batch = %d", rb.Len())
	}
}

func TestRingBuffer_Clear(t *testing.T) {
	rb := NewRingBuffer[int](3)
	rb.Push(1)
	rb.Push(2)
	rb.Clear()

	if rb.Len() != 0 {
		t.Fatalf("Len after Clear = %d, want 0", rb.Len())
	}
	if snap := rb.Snapshot(); snap != nil {
		t.Fatalf("Snapshot after Clear = %v, want nil", snap)
	}
}

func TestRingBuffer_Concurrent(t *testing.T) {
	rb := NewRingBuffer[int](100)
	var wg sync.WaitGroup

	// Writer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10000; i++ {
			rb.Push(i)
		}
	}()

	// Readers
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				_ = rb.Snapshot()
				_ = rb.Len()
			}
		}()
	}

	wg.Wait()

	if rb.Len() != 100 {
		t.Fatalf("Len after concurrent ops = %d, want 100", rb.Len())
	}
}

func TestMetricsCache_ClusterMetrics(t *testing.T) {
	cache := NewMetricsCache(100)
	now := time.Now()
	ctx := "test-context"

	for i := 0; i < 10; i++ {
		cache.PushCluster(db.ClusterMetrics{
			Timestamp: now.Add(time.Duration(i) * time.Minute),
			Context:   ctx,
			TotalPods: i * 10,
		})
	}

	// Query last 5 minutes (entries 5-9)
	start := now.Add(5 * time.Minute)
	end := now.Add(10 * time.Minute)
	result := cache.GetClusterMetrics(ctx, start, end, 100)

	if len(result) != 5 {
		t.Fatalf("GetClusterMetrics returned %d items, want 5", len(result))
	}
	// Newest first
	if result[0].TotalPods != 90 {
		t.Errorf("First result TotalPods = %d, want 90", result[0].TotalPods)
	}
}

func TestMetricsCache_ClusterMetrics_NilForOlderRange(t *testing.T) {
	cache := NewMetricsCache(100)
	now := time.Now()

	cache.PushCluster(db.ClusterMetrics{
		Timestamp: now,
		Context:   "ctx",
	})

	// Request a range before the cached data
	result := cache.GetClusterMetrics("ctx", now.Add(-2*time.Hour), now.Add(-1*time.Hour), 100)
	if result != nil {
		t.Error("Expected nil for range older than cache")
	}
}

func TestMetricsCache_ClusterMetrics_ContextFilter(t *testing.T) {
	cache := NewMetricsCache(100)
	now := time.Now()

	cache.PushCluster(db.ClusterMetrics{Timestamp: now, Context: "ctx-a", TotalPods: 10})
	cache.PushCluster(db.ClusterMetrics{Timestamp: now.Add(time.Minute), Context: "ctx-b", TotalPods: 20})

	result := cache.GetClusterMetrics("ctx-a", now.Add(-time.Hour), now.Add(time.Hour), 100)
	if len(result) != 1 || result[0].TotalPods != 10 {
		t.Errorf("Context filter failed: got %v", result)
	}
}

func TestMetricsCache_NodeMetrics(t *testing.T) {
	cache := NewMetricsCache(100)
	now := time.Now()

	nodes := []db.NodeMetrics{
		{Timestamp: now, Context: "ctx", NodeName: "node-1", CPUMillis: 100},
		{Timestamp: now, Context: "ctx", NodeName: "node-2", CPUMillis: 200},
	}
	cache.PushNodes(nodes)

	result := cache.GetNodeMetrics("ctx", "node-1", now.Add(-time.Hour), now.Add(time.Hour), 100)
	if len(result) != 1 || result[0].CPUMillis != 100 {
		t.Errorf("Node metrics = %v, want 1 entry with CPU 100", result)
	}

	// Non-existent node
	result = cache.GetNodeMetrics("ctx", "node-99", now.Add(-time.Hour), now.Add(time.Hour), 100)
	if result != nil {
		t.Error("Expected nil for non-existent node")
	}
}

func TestMetricsCache_PodMetrics(t *testing.T) {
	cache := NewMetricsCache(100)
	now := time.Now()

	pods := []db.PodMetrics{
		{Timestamp: now, Context: "ctx", Namespace: "default", PodName: "app-1", CPUMillis: 50},
		{Timestamp: now, Context: "ctx", Namespace: "kube-system", PodName: "coredns", CPUMillis: 10},
	}
	cache.PushPods(pods)

	result := cache.GetPodMetrics("ctx", "default", "app-1", now.Add(-time.Hour), now.Add(time.Hour), 100)
	if len(result) != 1 || result[0].CPUMillis != 50 {
		t.Errorf("Pod metrics = %v, want 1 entry with CPU 50", result)
	}
}

func TestMetricsCache_GetLatestClusterMetrics(t *testing.T) {
	cache := NewMetricsCache(100)
	now := time.Now()

	cache.PushCluster(db.ClusterMetrics{Timestamp: now, Context: "ctx", TotalPods: 10})
	cache.PushCluster(db.ClusterMetrics{Timestamp: now.Add(time.Minute), Context: "ctx", TotalPods: 20})

	latest := cache.GetLatestClusterMetrics("ctx")
	if latest == nil || latest.TotalPods != 20 {
		t.Errorf("GetLatestClusterMetrics = %v, want TotalPods=20", latest)
	}

	// Non-existent context
	if cache.GetLatestClusterMetrics("other") != nil {
		t.Error("Expected nil for non-existent context")
	}
}

func TestMetricsCache_Limit(t *testing.T) {
	cache := NewMetricsCache(100)
	now := time.Now()

	for i := 0; i < 20; i++ {
		cache.PushCluster(db.ClusterMetrics{
			Timestamp: now.Add(time.Duration(i) * time.Minute),
			Context:   "ctx",
		})
	}

	result := cache.GetClusterMetrics("ctx", now.Add(-time.Hour), now.Add(time.Hour), 5)
	if len(result) != 5 {
		t.Fatalf("Expected 5 results with limit, got %d", len(result))
	}
}

func TestMetricsCache_EmptyCache(t *testing.T) {
	cache := NewMetricsCache(100)
	now := time.Now()

	if cache.GetClusterMetrics("ctx", now, now, 100) != nil {
		t.Error("Expected nil from empty cache")
	}
	if cache.GetNodeMetrics("ctx", "n", now, now, 100) != nil {
		t.Error("Expected nil from empty node cache")
	}
	if cache.GetPodMetrics("ctx", "ns", "p", now, now, 100) != nil {
		t.Error("Expected nil from empty pod cache")
	}
	if cache.GetLatestClusterMetrics("ctx") != nil {
		t.Error("Expected nil from empty latest cache")
	}
}
