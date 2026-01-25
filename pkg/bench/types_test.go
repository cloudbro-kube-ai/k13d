package bench

import (
	"testing"
	"time"
)

func TestTaskDifficulty(t *testing.T) {
	tests := []struct {
		difficulty TaskDifficulty
		expected   string
	}{
		{DifficultyEasy, "easy"},
		{DifficultyMedium, "medium"},
		{DifficultyHard, "hard"},
	}

	for _, tt := range tests {
		if string(tt.difficulty) != tt.expected {
			t.Errorf("TaskDifficulty = %s, want %s", tt.difficulty, tt.expected)
		}
	}
}

func TestTaskResult(t *testing.T) {
	tests := []struct {
		result   TaskResult
		expected string
	}{
		{ResultSuccess, "success"},
		{ResultFail, "fail"},
		{ResultError, "error"},
		{ResultTimeout, "timeout"},
		{ResultSkipped, "skipped"},
	}

	for _, tt := range tests {
		if string(tt.result) != tt.expected {
			t.Errorf("TaskResult = %s, want %s", tt.result, tt.expected)
		}
	}
}

func TestTaskIsolation(t *testing.T) {
	tests := []struct {
		isolation TaskIsolation
		expected  string
	}{
		{IsolationNamespace, "namespace"},
		{IsolationCluster, "cluster"},
		{IsolationNone, ""},
	}

	for _, tt := range tests {
		if string(tt.isolation) != tt.expected {
			t.Errorf("TaskIsolation = %s, want %s", tt.isolation, tt.expected)
		}
	}
}

func TestEvalResult_Duration(t *testing.T) {
	start := time.Now()
	end := start.Add(5 * time.Second)

	result := &EvalResult{
		StartTime: start,
		EndTime:   end,
		Duration:  end.Sub(start),
	}

	if result.Duration != 5*time.Second {
		t.Errorf("Duration = %v, want 5s", result.Duration)
	}
}

func TestBenchmarkSummary_Init(t *testing.T) {
	summary := &BenchmarkSummary{
		LLMResults: make(map[string]*LLMSummary),
	}

	if summary.LLMResults == nil {
		t.Error("LLMResults should not be nil")
	}

	// Add an LLM result
	summary.LLMResults["test"] = &LLMSummary{
		TotalTasks:   10,
		SuccessCount: 8,
		PassRate:     80.0,
	}

	if len(summary.LLMResults) != 1 {
		t.Errorf("LLMResults count = %d, want 1", len(summary.LLMResults))
	}
}

func TestLLMConfig_ID(t *testing.T) {
	config := LLMConfig{
		ID:       "openai-gpt-4",
		Provider: "openai",
		Model:    "gpt-4",
		Endpoint: "https://api.openai.com/v1",
	}

	if config.ID != "openai-gpt-4" {
		t.Errorf("ID = %s, want openai-gpt-4", config.ID)
	}
}

func TestFailure(t *testing.T) {
	failure := Failure{
		Type:     "contains",
		Expected: "success",
		Actual:   "error occurred",
		Message:  "Output does not contain 'success'",
	}

	if failure.Type != "contains" {
		t.Errorf("Type = %s, want contains", failure.Type)
	}
	if failure.Expected != "success" {
		t.Errorf("Expected = %s, want success", failure.Expected)
	}
}

func TestExpectation(t *testing.T) {
	// Test contains
	exp1 := Expectation{Contains: "pod.*created"}
	if exp1.Contains != "pod.*created" {
		t.Errorf("Contains = %s, want pod.*created", exp1.Contains)
	}

	// Test notContains
	exp2 := Expectation{NotContains: "error"}
	if exp2.NotContains != "error" {
		t.Errorf("NotContains = %s, want error", exp2.NotContains)
	}

	// Test exitCode
	exitCode := 0
	exp3 := Expectation{ExitCode: &exitCode}
	if *exp3.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", *exp3.ExitCode)
	}
}

func TestPrompt(t *testing.T) {
	// Inline prompt
	p1 := Prompt{Text: "Create a pod"}
	if p1.Text != "Create a pod" {
		t.Errorf("Text = %s, want 'Create a pod'", p1.Text)
	}

	// File-based prompt
	p2 := Prompt{File: "prompt.txt", Timeout: "5m"}
	if p2.File != "prompt.txt" {
		t.Errorf("File = %s, want prompt.txt", p2.File)
	}
	if p2.Timeout != "5m" {
		t.Errorf("Timeout = %s, want 5m", p2.Timeout)
	}
}

func TestRunConfig_Defaults(t *testing.T) {
	config := RunConfig{
		TaskDir:     "tasks",
		Parallelism: 4,
	}

	if config.TaskDir != "tasks" {
		t.Errorf("TaskDir = %s, want tasks", config.TaskDir)
	}
	if config.Parallelism != 4 {
		t.Errorf("Parallelism = %d, want 4", config.Parallelism)
	}
}

func TestClusterPolicy(t *testing.T) {
	tests := []struct {
		policy   ClusterPolicy
		expected string
	}{
		{ClusterAlwaysCreate, "always"},
		{ClusterCreateIfNotExist, "create_if_not"},
		{ClusterDoNotCreate, "do_not_create"},
	}

	for _, tt := range tests {
		if string(tt.policy) != tt.expected {
			t.Errorf("ClusterPolicy = %s, want %s", tt.policy, tt.expected)
		}
	}
}
