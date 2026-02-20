package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
}

func TestResourceWatcher_StartStop(t *testing.T) {
	fakeClient := fake.NewSimpleClientset() //nolint:staticcheck
	client := &Client{Clientset: fakeClient}

	var called int32
	onChange := func() { atomic.AddInt32(&called, 1) }

	cfg := WatcherConfig{
		RelistInterval:   10 * time.Second,
		DebounceInterval: 50 * time.Millisecond,
		FallbackInterval: 1 * time.Second,
	}

	w := NewResourceWatcher(client, "pods", "default", onChange, testLogger(), cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.Start(ctx)

	// Wait for watch to become active
	deadline := time.After(2 * time.Second)
	for {
		if w.State() == WatchStateActive {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("watcher did not become active, state=%d", w.State())
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	if w.State() != WatchStateActive {
		t.Errorf("expected WatchStateActive, got %d", w.State())
	}

	w.Stop()

	if w.State() != WatchStateInactive {
		t.Errorf("expected WatchStateInactive after Stop, got %d", w.State())
	}

	// Stop should be idempotent
	w.Stop()
	if w.State() != WatchStateInactive {
		t.Errorf("expected WatchStateInactive after second Stop, got %d", w.State())
	}
}

func TestResourceWatcher_WatchEventCallback(t *testing.T) {
	fakeClient := fake.NewSimpleClientset() //nolint:staticcheck
	client := &Client{Clientset: fakeClient}

	var callCount int32
	onChange := func() { atomic.AddInt32(&callCount, 1) }

	cfg := WatcherConfig{
		RelistInterval:   30 * time.Second,
		DebounceInterval: 50 * time.Millisecond,
		FallbackInterval: 5 * time.Second,
	}

	w := NewResourceWatcher(client, "pods", "default", onChange, testLogger(), cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.Start(ctx)

	// Wait for active state
	deadline := time.After(2 * time.Second)
	for w.State() != WatchStateActive {
		select {
		case <-deadline:
			t.Fatalf("watcher did not become active")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Create a pod — this should trigger a watch event
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}
	_, err := fakeClient.CoreV1().Pods("default").Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create pod: %v", err)
	}

	// Wait for onChange to be called (debounce interval + margin)
	time.Sleep(200 * time.Millisecond)

	count := atomic.LoadInt32(&callCount)
	if count == 0 {
		t.Errorf("expected onChange to be called at least once, got %d calls", count)
	}

	w.Stop()
}

func TestResourceWatcher_Debounce(t *testing.T) {
	fakeClient := fake.NewSimpleClientset() //nolint:staticcheck
	client := &Client{Clientset: fakeClient}

	var callCount int32
	onChange := func() { atomic.AddInt32(&callCount, 1) }

	cfg := WatcherConfig{
		RelistInterval:   30 * time.Second,
		DebounceInterval: 200 * time.Millisecond, // Longer debounce for this test
		FallbackInterval: 5 * time.Second,
	}

	w := NewResourceWatcher(client, "pods", "default", onChange, testLogger(), cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.Start(ctx)

	// Wait for active state
	deadline := time.After(2 * time.Second)
	for w.State() != WatchStateActive {
		select {
		case <-deadline:
			t.Fatalf("watcher did not become active")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Create multiple pods rapidly — these should be debounced into fewer onChange calls
	for i := 0; i < 5; i++ {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pod-%d", i),
				Namespace: "default",
			},
		}
		_, err := fakeClient.CoreV1().Pods("default").Create(ctx, pod, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("failed to create pod %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond) // Small gap between events
	}

	// Wait for debounce to fire
	time.Sleep(400 * time.Millisecond)

	count := atomic.LoadInt32(&callCount)
	// With 200ms debounce and 5 events 10ms apart, we expect fewer calls than events
	if count >= 5 {
		t.Errorf("expected debouncing to reduce calls below 5, got %d", count)
	}
	if count == 0 {
		t.Errorf("expected at least 1 onChange call, got 0")
	}

	w.Stop()
}

func TestResourceWatcher_Relist(t *testing.T) {
	fakeClient := fake.NewSimpleClientset() //nolint:staticcheck
	client := &Client{Clientset: fakeClient}

	var callCount int32
	onChange := func() { atomic.AddInt32(&callCount, 1) }

	cfg := WatcherConfig{
		RelistInterval:   100 * time.Millisecond, // Very short for testing
		DebounceInterval: 50 * time.Millisecond,
		FallbackInterval: 5 * time.Second,
	}

	w := NewResourceWatcher(client, "pods", "default", onChange, testLogger(), cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.Start(ctx)

	// Wait for at least 3 relist intervals
	time.Sleep(400 * time.Millisecond)

	count := atomic.LoadInt32(&callCount)
	if count < 2 {
		t.Errorf("expected at least 2 relist-triggered onChange calls, got %d", count)
	}

	w.Stop()
}

func TestResourceWatcher_FallbackOnError(t *testing.T) {
	fakeClient := fake.NewSimpleClientset() //nolint:staticcheck
	client := &Client{Clientset: fakeClient}

	var callCount int32
	onChange := func() { atomic.AddInt32(&callCount, 1) }

	cfg := WatcherConfig{
		RelistInterval:   5 * time.Second,
		DebounceInterval: 50 * time.Millisecond,
		FallbackInterval: 100 * time.Millisecond, // Short for testing
	}

	// Use an unsupported resource to force watch failure
	w := NewResourceWatcher(client, "unsupported-resource", "default", onChange, testLogger(), cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.Start(ctx)

	// Wait for fallback state
	deadline := time.After(2 * time.Second)
	for w.State() != WatchStateFallback {
		select {
		case <-deadline:
			t.Fatalf("watcher did not enter fallback state, state=%d", w.State())
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	if w.State() != WatchStateFallback {
		t.Errorf("expected WatchStateFallback, got %d", w.State())
	}

	// Wait for polling to trigger onChange
	time.Sleep(300 * time.Millisecond)

	count := atomic.LoadInt32(&callCount)
	if count == 0 {
		t.Errorf("expected fallback polling to trigger onChange, got 0 calls")
	}

	w.Stop()
}

func TestResourceWatcher_ContextCancel(t *testing.T) {
	fakeClient := fake.NewSimpleClientset() //nolint:staticcheck
	client := &Client{Clientset: fakeClient}

	onChange := func() {}

	cfg := WatcherConfig{
		RelistInterval:   30 * time.Second,
		DebounceInterval: 50 * time.Millisecond,
		FallbackInterval: 5 * time.Second,
	}

	w := NewResourceWatcher(client, "pods", "default", onChange, testLogger(), cfg)

	ctx, cancel := context.WithCancel(context.Background())

	w.Start(ctx)

	// Wait for active state
	deadline := time.After(2 * time.Second)
	for w.State() != WatchStateActive {
		select {
		case <-deadline:
			t.Fatalf("watcher did not become active")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Cancel context
	cancel()

	// Give goroutine time to exit
	time.Sleep(100 * time.Millisecond)

	// Watcher should still report its last state (Active), but the goroutine has exited
	// Calling Stop explicitly should work fine
	w.Stop()

	if w.State() != WatchStateInactive {
		t.Errorf("expected WatchStateInactive after Stop, got %d", w.State())
	}
}

func TestWatchResource_SupportedTypes(t *testing.T) {
	fakeClient := fake.NewSimpleClientset() //nolint:staticcheck
	client := &Client{Clientset: fakeClient}

	ctx := context.Background()

	supportedResources := []string{
		"pods", "services", "nodes", "namespaces", "events",
		"configmaps", "secrets", "persistentvolumes", "persistentvolumeclaims",
		"serviceaccounts", "endpoints", "limitranges", "resourcequotas",
		"replicationcontrollers",
		"deployments", "statefulsets", "daemonsets", "replicasets",
		"jobs", "cronjobs",
		"ingresses", "networkpolicies",
		"roles", "rolebindings", "clusterroles", "clusterrolebindings",
		"storageclasses",
		"poddisruptionbudgets",
		"horizontalpodautoscalers",
	}

	for _, resource := range supportedResources {
		t.Run(resource, func(t *testing.T) {
			w, err := client.WatchResource(ctx, resource, "default")
			if err != nil {
				t.Errorf("WatchResource(%q) returned error: %v", resource, err)
				return
			}
			if w == nil {
				t.Errorf("WatchResource(%q) returned nil watcher", resource)
				return
			}
			w.Stop()
		})
	}
}

func TestWatchResource_UnsupportedType(t *testing.T) {
	fakeClient := fake.NewSimpleClientset() //nolint:staticcheck
	client := &Client{Clientset: fakeClient}

	ctx := context.Background()

	_, err := client.WatchResource(ctx, "nonexistent", "default")
	if err == nil {
		t.Error("expected error for unsupported resource type, got nil")
	}
}
