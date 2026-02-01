package testutil

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

// K8sReaderContractTest validates that an implementation satisfies the K8sReader contract.
// Use this in your implementation tests to ensure interface compatibility.
//
// Example:
//
//	func TestMyClient_K8sReaderContract(t *testing.T) {
//	    client := setupMyClient(t)
//	    testutil.K8sReaderContractTest(t, client)
//	}
func K8sReaderContractTest(t *testing.T, reader K8sReader) {
	t.Helper()
	ctx := context.Background()

	t.Run("ListPods", func(t *testing.T) {
		pods, err := reader.ListPods(ctx, "default")
		if err != nil {
			t.Errorf("ListPods should not error on valid namespace: %v", err)
		}
		// Contract: should return slice, not nil
		if pods == nil {
			t.Error("ListPods should return empty slice, not nil")
		}
	})

	t.Run("ListNodes", func(t *testing.T) {
		nodes, err := reader.ListNodes(ctx)
		if err != nil {
			t.Errorf("ListNodes should not error: %v", err)
		}
		if nodes == nil {
			t.Error("ListNodes should return empty slice, not nil")
		}
	})

	t.Run("ListNamespaces", func(t *testing.T) {
		ns, err := reader.ListNamespaces(ctx)
		if err != nil {
			t.Errorf("ListNamespaces should not error: %v", err)
		}
		if ns == nil {
			t.Error("ListNamespaces should return empty slice, not nil")
		}
	})

	t.Run("ListDeployments", func(t *testing.T) {
		deps, err := reader.ListDeployments(ctx, "default")
		if err != nil {
			t.Errorf("ListDeployments should not error: %v", err)
		}
		if deps == nil {
			t.Error("ListDeployments should return empty slice, not nil")
		}
	})

	t.Run("ListServices", func(t *testing.T) {
		svcs, err := reader.ListServices(ctx, "default")
		if err != nil {
			t.Errorf("ListServices should not error: %v", err)
		}
		if svcs == nil {
			t.Error("ListServices should return empty slice, not nil")
		}
	})
}

// K8sReaderWithDataTest validates K8sReader with expected data.
// Provides more thorough testing when you can seed data.
type K8sReaderWithDataTest struct {
	Reader            K8sReader
	ExpectedPodCount  int
	ExpectedNodeCount int
}

// Run executes the contract test with data validation.
func (tc *K8sReaderWithDataTest) Run(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	if tc.ExpectedPodCount > 0 {
		t.Run("ListPods_Returns_Expected_Count", func(t *testing.T) {
			pods, err := tc.Reader.ListPods(ctx, "default")
			if err != nil {
				t.Fatalf("ListPods failed: %v", err)
			}
			if len(pods) != tc.ExpectedPodCount {
				t.Errorf("Expected %d pods, got %d", tc.ExpectedPodCount, len(pods))
			}
		})
	}

	if tc.ExpectedNodeCount > 0 {
		t.Run("ListNodes_Returns_Expected_Count", func(t *testing.T) {
			nodes, err := tc.Reader.ListNodes(ctx)
			if err != nil {
				t.Fatalf("ListNodes failed: %v", err)
			}
			if len(nodes) != tc.ExpectedNodeCount {
				t.Errorf("Expected %d nodes, got %d", tc.ExpectedNodeCount, len(nodes))
			}
		})
	}
}

// LLMProviderContractTest validates that an implementation satisfies the LLMProvider contract.
func LLMProviderContractTest(t *testing.T, provider LLMProvider) {
	t.Helper()

	t.Run("Name_Not_Empty", func(t *testing.T) {
		name := provider.Name()
		if name == "" {
			t.Error("Provider Name() should not be empty")
		}
	})

	t.Run("GetModel_Returns_Value", func(t *testing.T) {
		model := provider.GetModel()
		// Model can be empty if not configured, but should not panic
		_ = model
	})

	t.Run("IsReady_Returns_Bool", func(t *testing.T) {
		// Should not panic
		_ = provider.IsReady()
	})
}

// LLMProviderStreamTest tests streaming behavior.
type LLMProviderStreamTest struct {
	Provider       LLMProvider
	TestPrompt     string
	ExpectResponse bool
}

// Run executes the streaming test.
func (tc *LLMProviderStreamTest) Run(t *testing.T) {
	t.Helper()

	if !tc.Provider.IsReady() {
		t.Skip("Provider not ready, skipping streaming test")
	}

	ctx := context.Background()
	var chunks []string
	callback := func(chunk string) {
		chunks = append(chunks, chunk)
	}

	err := tc.Provider.Ask(ctx, tc.TestPrompt, callback)
	if err != nil {
		t.Errorf("Ask failed: %v", err)
		return
	}

	if tc.ExpectResponse && len(chunks) == 0 {
		t.Error("Expected streaming response but got no chunks")
	}
}

// TableTestCase represents a generic table-driven test case.
type TableTestCase[I, O any] struct {
	Name    string
	Input   I
	Want    O
	WantErr bool
}

// RunTableTests runs table-driven tests with a test function.
func RunTableTests[I, O any](t *testing.T, cases []TableTestCase[I, O], fn func(I) (O, error), compare func(O, O) bool) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			got, err := fn(tc.Input)
			if tc.WantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if !compare(got, tc.Want) {
				t.Errorf("got %v, want %v", got, tc.Want)
			}
		})
	}
}

// AssertPodPhase checks if pod has expected phase.
func AssertPodPhase(t *testing.T, pod corev1.Pod, expected corev1.PodPhase) {
	t.Helper()
	if pod.Status.Phase != expected {
		t.Errorf("Pod %s phase = %v, want %v", pod.Name, pod.Status.Phase, expected)
	}
}

// AssertPodReady checks if pod is ready.
func AssertPodReady(t *testing.T, pod corev1.Pod) {
	t.Helper()
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			return
		}
	}
	t.Errorf("Pod %s is not ready", pod.Name)
}

// AssertNoError fails if err is not nil.
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

// AssertError fails if err is nil.
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Error("Expected error but got none")
	}
}

// AssertEqual fails if got != want.
func AssertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// AssertNotNil fails if v is nil.
func AssertNotNil(t *testing.T, v interface{}) {
	t.Helper()
	if v == nil {
		t.Error("Expected non-nil value")
	}
}
