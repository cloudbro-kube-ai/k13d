---
name: go-best-practices
description: Go language best practices and idiomatic patterns. Use this skill when writing or reviewing Go code, ensuring code quality, or applying Go conventions. Focused on Kubernetes client-go patterns.
version: 1.0.0
---

# Go Best Practices Skill

Idiomatic Go patterns and best practices, with emphasis on Kubernetes development.

## When to Use

- Writing new Go code
- Reviewing Go code for best practices
- Refactoring existing Go code
- Learning Go idioms

## Core Go Principles

### 1. Error Handling

```go
// GOOD: Always handle errors explicitly
result, err := doSomething()
if err != nil {
    return fmt.Errorf("doSomething failed: %w", err)
}

// GOOD: Use error wrapping for context
if err := validateInput(input); err != nil {
    return fmt.Errorf("invalid input %q: %w", input, err)
}

// BAD: Ignoring errors
result, _ := doSomething() // Never do this

// BAD: Checking error without handling
if err != nil {
    // empty block
}
```

### 2. Context Handling

```go
// GOOD: Pass context as first parameter
func (c *Client) GetPods(ctx context.Context, namespace string) ([]Pod, error) {
    // Respect context cancellation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }

    // Use context for API calls
    return c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
}

// GOOD: Set reasonable timeouts
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()

// BAD: Using context.Background() everywhere
pods, _ := client.GetPods(context.Background(), "default")
```

### 3. Goroutines and Concurrency

```go
// GOOD: Use WaitGroup for goroutine synchronization
func processItems(items []Item) error {
    var wg sync.WaitGroup
    errCh := make(chan error, len(items))

    for _, item := range items {
        wg.Add(1)
        go func(item Item) {
            defer wg.Done()
            if err := process(item); err != nil {
                errCh <- err
            }
        }(item) // Pass item to avoid closure capture
    }

    wg.Wait()
    close(errCh)

    for err := range errCh {
        if err != nil {
            return err
        }
    }
    return nil
}

// BAD: Goroutine leak - no way to stop
go func() {
    for {
        doWork() // Runs forever
    }
}()

// GOOD: Stoppable goroutine
go func() {
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            doWork()
        }
    }
}()
```

### 4. Defer Patterns

```go
// GOOD: Defer for cleanup
func readFile(path string) ([]byte, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    return io.ReadAll(f)
}

// BAD: Defer in loop (resource accumulation)
for _, path := range paths {
    f, _ := os.Open(path)
    defer f.Close() // All closes happen at function end!
}

// GOOD: Extract to function or close explicitly
for _, path := range paths {
    if err := processFile(path); err != nil {
        return err
    }
}

func processFile(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    defer f.Close()
    // process file
    return nil
}
```

### 5. Slice and Map Initialization

```go
// GOOD: Pre-allocate when size is known
result := make([]string, 0, len(input))
for _, item := range input {
    result = append(result, item.Name)
}

// GOOD: Initialize maps before use
m := make(map[string]int)
m["key"] = 1

// BAD: Nil map assignment panics
var m map[string]int
m["key"] = 1 // panic!

// GOOD: Check map existence
if val, ok := m["key"]; ok {
    // use val
}
```

### 6. Interface Design

```go
// GOOD: Small, focused interfaces
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

// Compose interfaces
type ReadWriter interface {
    Reader
    Writer
}

// GOOD: Accept interfaces, return structs
func ProcessData(r io.Reader) (*Result, error) {
    // Implementation
}

// BAD: Large interfaces
type DoEverything interface {
    Read()
    Write()
    Delete()
    Update()
    Validate()
    // ... 20 more methods
}
```

## Kubernetes Client-Go Patterns

### 1. Client Creation

```go
// GOOD: Support both in-cluster and out-of-cluster
func NewK8sClient() (*kubernetes.Clientset, error) {
    config, err := rest.InClusterConfig()
    if err != nil {
        // Fallback to kubeconfig
        kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
        config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
        if err != nil {
            return nil, fmt.Errorf("failed to build config: %w", err)
        }
    }

    // Set reasonable defaults
    config.QPS = 50
    config.Burst = 100

    return kubernetes.NewForConfig(config)
}
```

### 2. Informer Usage

```go
// GOOD: Use informers for watching resources
func setupInformer(clientset *kubernetes.Clientset) cache.SharedIndexInformer {
    factory := informers.NewSharedInformerFactory(clientset, 30*time.Second)
    podInformer := factory.Core().V1().Pods().Informer()

    podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc: func(obj interface{}) {
            pod := obj.(*corev1.Pod)
            log.Printf("Pod added: %s/%s", pod.Namespace, pod.Name)
        },
        UpdateFunc: func(oldObj, newObj interface{}) {
            // Handle update
        },
        DeleteFunc: func(obj interface{}) {
            // Handle tombstone for deleted objects
            pod, ok := obj.(*corev1.Pod)
            if !ok {
                tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
                if !ok {
                    return
                }
                pod, ok = tombstone.Obj.(*corev1.Pod)
                if !ok {
                    return
                }
            }
            log.Printf("Pod deleted: %s/%s", pod.Namespace, pod.Name)
        },
    })

    return podInformer
}

// Start informer with context
stopCh := make(chan struct{})
go informer.Run(stopCh)

// Wait for cache sync
if !cache.WaitForCacheSync(stopCh, informer.HasSynced) {
    return fmt.Errorf("timed out waiting for cache sync")
}
```

### 3. Resource Operations

```go
// GOOD: Use retry for transient errors
err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
    deployment, err := client.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
    if err != nil {
        return err
    }

    deployment.Spec.Replicas = pointer.Int32(3)

    _, err = client.AppsV1().Deployments(ns).Update(ctx, deployment, metav1.UpdateOptions{})
    return err
})

// GOOD: Use field selectors to reduce API load
pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
    FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
})

// GOOD: Use label selectors
pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
    LabelSelector: "app=myapp,env=prod",
})
```

### 4. Error Handling for K8s

```go
// GOOD: Check specific error types
import "k8s.io/apimachinery/pkg/api/errors"

pod, err := client.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
if errors.IsNotFound(err) {
    // Handle not found - maybe create it
    return createPod(ctx, client, ns, name)
}
if err != nil {
    return fmt.Errorf("failed to get pod: %w", err)
}

// Other useful checks
if errors.IsConflict(err) { /* retry */ }
if errors.IsForbidden(err) { /* permission issue */ }
if errors.IsAlreadyExists(err) { /* resource exists */ }
```

## Testing Patterns

### 1. Table-Driven Tests

```go
func TestCalculate(t *testing.T) {
    tests := []struct {
        name     string
        input    int
        expected int
        wantErr  bool
    }{
        {"positive", 5, 25, false},
        {"zero", 0, 0, false},
        {"negative", -5, 25, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Calculate(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Calculate() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.expected {
                t.Errorf("Calculate() = %v, want %v", got, tt.expected)
            }
        })
    }
}
```

### 2. Fake Kubernetes Client

```go
import "k8s.io/client-go/kubernetes/fake"

func TestPodLister(t *testing.T) {
    // Create fake client with initial objects
    clientset := fake.NewSimpleClientset(
        &corev1.Pod{
            ObjectMeta: metav1.ObjectMeta{
                Name:      "test-pod",
                Namespace: "default",
            },
        },
    )

    // Test your code
    lister := NewPodLister(clientset)
    pods, err := lister.List("default")
    if err != nil {
        t.Fatal(err)
    }
    if len(pods) != 1 {
        t.Errorf("expected 1 pod, got %d", len(pods))
    }
}
```

## Code Organization

```
pkg/
├── api/           # API types and handlers
├── client/        # External client wrappers
├── config/        # Configuration handling
├── controller/    # Business logic controllers
├── k8s/           # Kubernetes client utilities
├── middleware/    # HTTP middleware
├── model/         # Data models
├── service/       # Service layer
└── util/          # Utility functions
```

## Common Anti-Patterns

| Anti-Pattern | Problem | Solution |
|--------------|---------|----------|
| `panic` in library code | Crashes caller | Return errors instead |
| `init()` with side effects | Hard to test | Use explicit initialization |
| Global variables | Concurrency issues | Dependency injection |
| Bare returns | Unclear intent | Explicit return values |
| Deep nesting | Hard to read | Early returns |
| Empty interface `interface{}` | No type safety | Use generics or specific types |
