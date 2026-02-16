package agent

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai/analyzers"
)

func TestAnonymizerIntegration_Enabled(t *testing.T) {
	cfg := &Config{
		EnableAnonymization: true,
	}
	agent := New(cfg)
	agent.provider = &promptCapturingProvider{}
	agent.StartSession("test", "model")

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Send a message containing an IP address (should be anonymized)
	err := agent.Ask(ctx, "Check server at 192.168.1.100")
	if err != nil && err != context.DeadlineExceeded {
		t.Fatalf("Ask failed: %v", err)
	}

	// The prompt sent to the provider should NOT contain the raw IP
	captured := agent.provider.(*promptCapturingProvider)
	if strings.Contains(captured.lastPrompt, "192.168.1.100") {
		t.Error("Prompt sent to LLM should have anonymized IP address")
	}
	if !strings.Contains(captured.lastPrompt, "<IP_1>") {
		t.Error("Prompt sent to LLM should contain anonymized placeholder <IP_1>")
	}
}

func TestAnonymizerIntegration_Disabled(t *testing.T) {
	cfg := &Config{
		EnableAnonymization: false,
	}
	agent := New(cfg)
	agent.provider = &promptCapturingProvider{}
	agent.StartSession("test", "model")

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := agent.Ask(ctx, "Check server at 192.168.1.100")
	if err != nil && err != context.DeadlineExceeded {
		t.Fatalf("Ask failed: %v", err)
	}

	captured := agent.provider.(*promptCapturingProvider)
	if !strings.Contains(captured.lastPrompt, "192.168.1.100") {
		t.Error("Prompt should contain raw IP when anonymization is disabled")
	}
}

func TestAnonymizerIntegration_Deanonymize(t *testing.T) {
	cfg := &Config{
		EnableAnonymization: true,
	}
	agent := New(cfg)
	// Provider that echoes back the placeholder it received
	agent.provider = &echoPlaceholderProvider{}
	agent.StartSession("test", "model")

	var mu sync.Mutex
	var chunks []string
	done := make(chan struct{})
	go func() {
		defer close(done)
		for msg := range agent.Output {
			if msg.Type == MsgStreamChunk {
				mu.Lock()
				chunks = append(chunks, msg.Content)
				mu.Unlock()
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := agent.Ask(ctx, "Check 10.0.0.1")
	if err != nil && err != context.DeadlineExceeded {
		t.Fatalf("Ask failed: %v", err)
	}

	// Wait briefly for output goroutine to process remaining messages, then close
	time.Sleep(50 * time.Millisecond)

	// Read chunks under lock
	mu.Lock()
	combined := strings.Join(chunks, "")
	mu.Unlock()

	if strings.Contains(combined, "<IP_") {
		t.Error("Output should not contain anonymized placeholder after deanonymization")
	}
}

func TestPreAnalysisIntegration(t *testing.T) {
	registry := analyzers.NewRegistry()
	registry.Register(&analyzers.PodAnalyzer{})

	cfg := &Config{
		AnalyzerRegistry: registry,
	}
	agent := New(cfg)
	agent.provider = &promptCapturingProvider{}

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	resourceContext := "Kind: Pod\nName: my-pod\nNamespace: default\nStatus: CrashLoopBackOff\nRestarts: 15"
	err := agent.AskWithContext(ctx, "Why is this pod failing?", resourceContext)
	if err != nil && err != context.DeadlineExceeded {
		t.Fatalf("AskWithContext failed: %v", err)
	}

	captured := agent.provider.(*promptCapturingProvider)
	if !strings.Contains(captured.lastPrompt, "[Pre-analysis findings]") {
		t.Error("Prompt should contain pre-analysis findings")
	}
}

func TestPreAnalysisIntegration_NilRegistry(t *testing.T) {
	cfg := &Config{}
	agent := New(cfg)

	// The default registry is set, override it to nil for this test
	agent.analyzerRegistry = nil

	result := agent.runPreAnalysis(context.Background(), "Kind: Pod\nStatus: Running")
	if result != "" {
		t.Errorf("runPreAnalysis with nil registry should return empty, got %q", result)
	}
}

func TestPreAnalysisIntegration_EmptyContext(t *testing.T) {
	agent := New(nil)

	result := agent.runPreAnalysis(context.Background(), "")
	if result != "" {
		t.Errorf("runPreAnalysis with empty context should return empty, got %q", result)
	}
}

func TestPreAnalysisIntegration_NoFindings(t *testing.T) {
	registry := analyzers.NewRegistry()
	// Empty registry - no analyzers registered
	cfg := &Config{
		AnalyzerRegistry: registry,
	}
	agent := New(cfg)

	result := agent.runPreAnalysis(context.Background(), "Kind: Pod\nStatus: Running")
	if result != "" {
		t.Errorf("runPreAnalysis with no findings should return empty, got %q", result)
	}
}

// promptCapturingProvider captures the last prompt sent to it
type promptCapturingProvider struct {
	lastPrompt string
}

func (p *promptCapturingProvider) Name() string     { return "capturing" }
func (p *promptCapturingProvider) GetModel() string { return "capturing-model" }
func (p *promptCapturingProvider) Ask(ctx context.Context, prompt string, cb func(string)) error {
	p.lastPrompt = prompt
	if cb != nil {
		cb("OK")
	}
	return nil
}
func (p *promptCapturingProvider) AskNonStreaming(ctx context.Context, prompt string) (string, error) {
	p.lastPrompt = prompt
	return "OK", nil
}
func (p *promptCapturingProvider) IsReady() bool                                    { return true }
func (p *promptCapturingProvider) ListModels(ctx context.Context) ([]string, error) { return nil, nil }

// echoPlaceholderProvider echoes back the prompt with placeholder markers
type echoPlaceholderProvider struct {
	lastPrompt string
}

func (p *echoPlaceholderProvider) Name() string     { return "echo" }
func (p *echoPlaceholderProvider) GetModel() string { return "echo-model" }
func (p *echoPlaceholderProvider) Ask(ctx context.Context, prompt string, cb func(string)) error {
	p.lastPrompt = prompt
	if cb != nil {
		// Echo back the IP placeholder if present, simulating LLM using the placeholder
		if strings.Contains(prompt, "<IP_1>") {
			cb("The server at <IP_1> is reachable")
		} else {
			cb("OK")
		}
	}
	return nil
}
func (p *echoPlaceholderProvider) AskNonStreaming(ctx context.Context, prompt string) (string, error) {
	p.lastPrompt = prompt
	return "OK", nil
}
func (p *echoPlaceholderProvider) IsReady() bool                                    { return true }
func (p *echoPlaceholderProvider) ListModels(ctx context.Context) ([]string, error) { return nil, nil }
