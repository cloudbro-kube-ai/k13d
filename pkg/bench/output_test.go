package bench

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAnalyzer_Analyze(t *testing.T) {
	// Create test results
	now := time.Now()
	results := []*EvalResult{
		{
			TaskID:     "task-1",
			Difficulty: DifficultyEasy,
			LLMConfig:  LLMConfig{ID: "gpt-4", Model: "gpt-4"},
			Result:     ResultSuccess,
			StartTime:  now,
			EndTime:    now.Add(10 * time.Second),
			Duration:   10 * time.Second,
			RunID:      "test-run",
		},
		{
			TaskID:     "task-2",
			Difficulty: DifficultyMedium,
			LLMConfig:  LLMConfig{ID: "gpt-4", Model: "gpt-4"},
			Result:     ResultFail,
			StartTime:  now,
			EndTime:    now.Add(15 * time.Second),
			Duration:   15 * time.Second,
			RunID:      "test-run",
		},
		{
			TaskID:     "task-3",
			Difficulty: DifficultyHard,
			LLMConfig:  LLMConfig{ID: "claude", Model: "claude-3"},
			Result:     ResultSuccess,
			StartTime:  now,
			EndTime:    now.Add(20 * time.Second),
			Duration:   20 * time.Second,
			RunID:      "test-run",
		},
	}

	analyzer := NewAnalyzer("", OutputMarkdown)
	summary := analyzer.Analyze(results)

	// Check overall counts
	if summary.TotalTasks != 3 {
		t.Errorf("TotalTasks = %d, want 3", summary.TotalTasks)
	}
	if summary.SuccessCount != 2 {
		t.Errorf("SuccessCount = %d, want 2", summary.SuccessCount)
	}
	if summary.FailCount != 1 {
		t.Errorf("FailCount = %d, want 1", summary.FailCount)
	}

	// Check difficulty breakdown
	if summary.EasyTotal != 1 || summary.EasySuccess != 1 {
		t.Errorf("Easy: total=%d, success=%d, want total=1, success=1", summary.EasyTotal, summary.EasySuccess)
	}
	if summary.MediumTotal != 1 || summary.MediumSuccess != 0 {
		t.Errorf("Medium: total=%d, success=%d, want total=1, success=0", summary.MediumTotal, summary.MediumSuccess)
	}
	if summary.HardTotal != 1 || summary.HardSuccess != 1 {
		t.Errorf("Hard: total=%d, success=%d, want total=1, success=1", summary.HardTotal, summary.HardSuccess)
	}

	// Check per-LLM breakdown
	if len(summary.LLMResults) != 2 {
		t.Errorf("LLMResults count = %d, want 2", len(summary.LLMResults))
	}

	gpt4Summary, ok := summary.LLMResults["gpt-4"]
	if !ok {
		t.Error("gpt-4 summary not found")
	} else {
		if gpt4Summary.TotalTasks != 2 {
			t.Errorf("GPT-4 TotalTasks = %d, want 2", gpt4Summary.TotalTasks)
		}
		if gpt4Summary.SuccessCount != 1 {
			t.Errorf("GPT-4 SuccessCount = %d, want 1", gpt4Summary.SuccessCount)
		}
	}

	// Check pass rate
	expectedPassRate := float64(2) / float64(3) * 100
	if summary.PassAt1 != expectedPassRate {
		t.Errorf("PassAt1 = %.2f, want %.2f", summary.PassAt1, expectedPassRate)
	}
}

func TestAnalyzer_FormatJSON(t *testing.T) {
	results := []*EvalResult{
		{
			TaskID:    "task-1",
			LLMConfig: LLMConfig{ID: "gpt-4"},
			Result:    ResultSuccess,
		},
	}

	analyzer := NewAnalyzer("", OutputJSON)
	summary := analyzer.Analyze(results)

	data, err := analyzer.formatJSON(summary, results)
	if err != nil {
		t.Fatalf("formatJSON failed: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("Invalid JSON output: %v", err)
	}

	// Check structure
	if _, ok := parsed["summary"]; !ok {
		t.Error("JSON output missing 'summary' field")
	}
	if _, ok := parsed["results"]; !ok {
		t.Error("JSON output missing 'results' field")
	}
}

func TestAnalyzer_FormatJSONL(t *testing.T) {
	results := []*EvalResult{
		{TaskID: "task-1", Result: ResultSuccess},
		{TaskID: "task-2", Result: ResultFail},
		{TaskID: "task-3", Result: ResultError},
	}

	analyzer := NewAnalyzer("", OutputJSONL)
	data, err := analyzer.formatJSONL(results)
	if err != nil {
		t.Fatalf("formatJSONL failed: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i, err)
		}
	}
}

func TestAnalyzer_FormatMarkdown(t *testing.T) {
	now := time.Now()
	results := []*EvalResult{
		{
			TaskID:     "task-1",
			TaskName:   "Test Task",
			Difficulty: DifficultyEasy,
			LLMConfig:  LLMConfig{ID: "gpt-4", Model: "gpt-4"},
			Result:     ResultSuccess,
			StartTime:  now,
			EndTime:    now.Add(10 * time.Second),
			Duration:   10 * time.Second,
			RunID:      "test-run",
		},
	}

	analyzer := NewAnalyzer("", OutputMarkdown)
	summary := analyzer.Analyze(results)

	data, err := analyzer.formatMarkdown(summary, results)
	if err != nil {
		t.Fatalf("formatMarkdown failed: %v", err)
	}

	content := string(data)

	// Check for key sections
	expectedSections := []string{
		"# K13D AI Benchmark Results",
		"## Overall Summary",
		"## Results by Difficulty",
		"## Results by LLM",
		"## Detailed Results",
	}

	for _, section := range expectedSections {
		if !strings.Contains(content, section) {
			t.Errorf("Markdown output missing section: %s", section)
		}
	}

	// Check for task details
	if !strings.Contains(content, "task-1") {
		t.Error("Markdown output missing task-1")
	}
}

func TestResultEmoji(t *testing.T) {
	tests := []struct {
		result   TaskResult
		expected string
	}{
		{ResultSuccess, "✅"},
		{ResultFail, "❌"},
		{ResultError, "⚠️"},
		{ResultTimeout, "⏱️"},
		{ResultSkipped, "⏭️"},
	}

	for _, tt := range tests {
		t.Run(string(tt.result), func(t *testing.T) {
			if got := resultEmoji(tt.result); got != tt.expected {
				t.Errorf("resultEmoji(%s) = %s, want %s", tt.result, got, tt.expected)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello..."},
		{"short", 100, "short"},
		{"", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := truncateString(tt.input, tt.maxLen); got != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}

func TestMarshalJSON(t *testing.T) {
	data := map[string]string{"key": "value"}

	result, err := marshalJSON(data)
	if err != nil {
		t.Fatalf("marshalJSON failed: %v", err)
	}

	expected := "{\n  \"key\": \"value\"\n}"
	if string(result) != expected {
		t.Errorf("marshalJSON() = %s, want %s", result, expected)
	}
}

func TestMarshalYAML(t *testing.T) {
	data := map[string]string{"key": "value"}

	result, err := marshalYAML(data)
	if err != nil {
		t.Fatalf("marshalYAML failed: %v", err)
	}

	// YAML output should contain the key-value pair
	if !strings.Contains(string(result), "key: value") {
		t.Errorf("marshalYAML() = %s, expected to contain 'key: value'", result)
	}
}

func TestGetFailedResults(t *testing.T) {
	results := []*EvalResult{
		{TaskID: "task-1", Result: ResultSuccess},
		{TaskID: "task-2", Result: ResultFail},
		{TaskID: "task-3", Result: ResultError},
		{TaskID: "task-4", Result: ResultSuccess},
		{TaskID: "task-5", Result: ResultTimeout},
		{TaskID: "task-6", Result: ResultSkipped},
	}

	failed := GetFailedResults(results)

	if len(failed) != 4 {
		t.Errorf("GetFailedResults() returned %d results, want 4", len(failed))
	}

	// Verify only non-success results are returned
	for _, r := range failed {
		if r.Result == ResultSuccess {
			t.Errorf("GetFailedResults() included success result: %s", r.TaskID)
		}
	}

	// Test with empty results
	empty := GetFailedResults([]*EvalResult{})
	if len(empty) != 0 {
		t.Errorf("GetFailedResults([]) returned %d results, want 0", len(empty))
	}

	// Test with all success
	allSuccess := []*EvalResult{
		{TaskID: "task-1", Result: ResultSuccess},
		{TaskID: "task-2", Result: ResultSuccess},
	}
	noFailed := GetFailedResults(allSuccess)
	if len(noFailed) != 0 {
		t.Errorf("GetFailedResults(all success) returned %d results, want 0", len(noFailed))
	}
}

func TestAnalyzer_FormatYAML(t *testing.T) {
	results := []*EvalResult{
		{
			TaskID:    "task-1",
			LLMConfig: LLMConfig{ID: "gpt-4"},
			Result:    ResultSuccess,
		},
	}

	analyzer := NewAnalyzer("", OutputYAML)
	summary := analyzer.Analyze(results)

	data, err := analyzer.formatYAML(summary, results)
	if err != nil {
		t.Fatalf("formatYAML failed: %v", err)
	}

	content := string(data)

	// Check for key YAML sections
	if !strings.Contains(content, "summary:") {
		t.Error("YAML output missing 'summary:' section")
	}
	if !strings.Contains(content, "results:") {
		t.Error("YAML output missing 'results:' section")
	}
	if !strings.Contains(content, "taskid: task-1") && !strings.Contains(content, "taskId: task-1") {
		t.Error("YAML output missing task-1")
	}
}

func TestAnalyzer_Analyze_Empty(t *testing.T) {
	analyzer := NewAnalyzer("", OutputMarkdown)
	summary := analyzer.Analyze([]*EvalResult{})

	if summary.TotalTasks != 0 {
		t.Errorf("TotalTasks = %d, want 0", summary.TotalTasks)
	}
	if summary.LLMResults == nil {
		t.Error("LLMResults should not be nil for empty input")
	}
}

func TestAnalyzer_Analyze_AllResults(t *testing.T) {
	now := time.Now()
	results := []*EvalResult{
		{
			TaskID:     "task-1",
			Difficulty: DifficultyEasy,
			LLMConfig:  LLMConfig{ID: "gpt-4", Model: "gpt-4"},
			Result:     ResultSuccess,
			StartTime:  now,
			EndTime:    now.Add(10 * time.Second),
			Duration:   10 * time.Second,
			RunID:      "test-run",
		},
		{
			TaskID:     "task-2",
			Difficulty: DifficultyMedium,
			LLMConfig:  LLMConfig{ID: "gpt-4", Model: "gpt-4"},
			Result:     ResultError,
			StartTime:  now.Add(10 * time.Second),
			EndTime:    now.Add(20 * time.Second),
			Duration:   10 * time.Second,
			RunID:      "test-run",
		},
		{
			TaskID:     "task-3",
			Difficulty: DifficultyHard,
			LLMConfig:  LLMConfig{ID: "gpt-4", Model: "gpt-4"},
			Result:     ResultTimeout,
			StartTime:  now.Add(20 * time.Second),
			EndTime:    now.Add(30 * time.Second),
			Duration:   10 * time.Second,
			RunID:      "test-run",
		},
		{
			TaskID:     "task-4",
			Difficulty: DifficultyEasy,
			LLMConfig:  LLMConfig{ID: "gpt-4", Model: "gpt-4"},
			Result:     ResultSkipped,
			StartTime:  now.Add(30 * time.Second),
			EndTime:    now.Add(31 * time.Second),
			Duration:   1 * time.Second,
			RunID:      "test-run",
		},
	}

	analyzer := NewAnalyzer("", OutputMarkdown)
	summary := analyzer.Analyze(results)

	// Check all result types are counted
	if summary.SuccessCount != 1 {
		t.Errorf("SuccessCount = %d, want 1", summary.SuccessCount)
	}
	if summary.ErrorCount != 2 { // Error + Timeout
		t.Errorf("ErrorCount = %d, want 2", summary.ErrorCount)
	}
	if summary.SkippedCount != 1 {
		t.Errorf("SkippedCount = %d, want 1", summary.SkippedCount)
	}
}

func TestResultEmoji_Unknown(t *testing.T) {
	// Test with an unknown result type
	result := resultEmoji(TaskResult("unknown"))
	if result != "❓" {
		t.Errorf("resultEmoji(unknown) = %s, want ❓", result)
	}
}

func TestAnalyzer_FormatMarkdown_WithFailures(t *testing.T) {
	now := time.Now()
	results := []*EvalResult{
		{
			TaskID:     "task-1",
			TaskName:   "Test Task",
			Difficulty: DifficultyEasy,
			LLMConfig:  LLMConfig{ID: "gpt-4", Model: "gpt-4"},
			Result:     ResultFail,
			StartTime:  now,
			EndTime:    now.Add(10 * time.Second),
			Duration:   10 * time.Second,
			RunID:      "test-run",
			Failures: []Failure{
				{Type: "contains", Message: "Expected pattern not found"},
			},
		},
		{
			TaskID:     "task-2",
			TaskName:   "Error Task",
			Difficulty: DifficultyMedium,
			LLMConfig:  LLMConfig{ID: "gpt-4", Model: "gpt-4"},
			Result:     ResultError,
			StartTime:  now,
			EndTime:    now.Add(5 * time.Second),
			Duration:   5 * time.Second,
			RunID:      "test-run",
			Error:      "Connection timeout",
		},
	}

	analyzer := NewAnalyzer("", OutputMarkdown)
	summary := analyzer.Analyze(results)

	data, err := analyzer.formatMarkdown(summary, results)
	if err != nil {
		t.Fatalf("formatMarkdown failed: %v", err)
	}

	content := string(data)

	// Check that failure message is included (truncated)
	if !strings.Contains(content, "Expected pattern not found") {
		t.Error("Markdown should contain failure message")
	}

	// Check that error message is included
	if !strings.Contains(content, "Connection timeout") {
		t.Error("Markdown should contain error message")
	}
}

func TestNewAnalyzer_DefaultFormat(t *testing.T) {
	// Test with empty format (should default to markdown)
	analyzer := NewAnalyzer("/tmp/output", "")
	if analyzer.outputFormat != OutputMarkdown {
		t.Errorf("Default format = %s, want %s", analyzer.outputFormat, OutputMarkdown)
	}
}
