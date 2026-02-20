// Package eval provides AI agent benchmark evaluation framework
// dryrun_test.go contains tests for dry-run evaluation
package eval

import (
	"context"
	"testing"
	"time"
)

func TestDryRunValidator_Validate(t *testing.T) {
	config := DryRunConfig{
		Mode:    DryRunToolValidation,
		Verbose: false,
	}
	validator := NewDryRunValidator(config)

	// Set expectations for test task
	validator.SetExpectation("test-create-pod", DryRunTaskExpectation{
		RequiredCommands: []CommandExpectation{
			{
				Pattern:     `kubectl (run|create|apply).*nginx-pod`,
				Description: "Create nginx pod",
				Required:    true,
			},
		},
		ForbiddenPatterns: []string{
			`--force.*--grace-period=0`,
		},
		MinToolCalls: 1,
		MaxToolCalls: 5,
	})

	tests := []struct {
		name       string
		taskID     string
		toolCalls  []ToolCallRecord
		wantPass   bool
		wantScore  float64
		wantMissed int
	}{
		{
			name:   "valid kubectl run command",
			taskID: "test-create-pod",
			toolCalls: []ToolCallRecord{
				{
					ID:        "call_1",
					ToolName:  "kubectl",
					Command:   "kubectl run nginx-pod --image=nginx:1.25",
					Timestamp: time.Now(),
				},
			},
			wantPass:   true,
			wantScore:  1.0,
			wantMissed: 0,
		},
		{
			name:   "valid kubectl create command",
			taskID: "test-create-pod",
			toolCalls: []ToolCallRecord{
				{
					ID:        "call_1",
					ToolName:  "kubectl",
					Command:   "kubectl create pod nginx-pod --image=nginx",
					Timestamp: time.Now(),
				},
			},
			wantPass:   true,
			wantScore:  1.0,
			wantMissed: 0,
		},
		{
			name:   "missing required command",
			taskID: "test-create-pod",
			toolCalls: []ToolCallRecord{
				{
					ID:        "call_1",
					ToolName:  "kubectl",
					Command:   "kubectl get pods",
					Timestamp: time.Now(),
				},
			},
			wantPass:   false,
			wantScore:  0.0,
			wantMissed: 1,
		},
		{
			name:       "too few tool calls",
			taskID:     "test-create-pod",
			toolCalls:  []ToolCallRecord{},
			wantPass:   false,
			wantScore:  0.0,
			wantMissed: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(tt.taskID, tt.toolCalls)

			if result.Success != tt.wantPass {
				t.Errorf("Validate() success = %v, want %v", result.Success, tt.wantPass)
			}

			if result.Score != tt.wantScore {
				t.Errorf("Validate() score = %v, want %v", result.Score, tt.wantScore)
			}

			if len(result.MissedPatterns) != tt.wantMissed {
				t.Errorf("Validate() missed patterns = %d, want %d", len(result.MissedPatterns), tt.wantMissed)
			}
		})
	}
}

func TestDryRunValidator_ForbiddenPatterns(t *testing.T) {
	config := DryRunConfig{
		Mode:    DryRunToolValidation,
		Verbose: false,
	}
	validator := NewDryRunValidator(config)

	validator.SetExpectation("test-forbidden", DryRunTaskExpectation{
		RequiredCommands: []CommandExpectation{
			{
				Pattern:     `kubectl delete`,
				Description: "Delete resource",
				Required:    true,
			},
		},
		ForbiddenPatterns: []string{
			`--force.*--grace-period=0`,
			`delete.*--all`,
		},
	})

	tests := []struct {
		name          string
		toolCalls     []ToolCallRecord
		wantForbidden int
	}{
		{
			name: "safe delete command",
			toolCalls: []ToolCallRecord{
				{
					ID:      "call_1",
					Command: "kubectl delete pod nginx-pod",
				},
			},
			wantForbidden: 0,
		},
		{
			name: "forbidden force delete",
			toolCalls: []ToolCallRecord{
				{
					ID:      "call_1",
					Command: "kubectl delete pod nginx-pod --force --grace-period=0",
				},
			},
			wantForbidden: 1,
		},
		{
			name: "forbidden delete all",
			toolCalls: []ToolCallRecord{
				{
					ID:      "call_1",
					Command: "kubectl delete pods --all",
				},
			},
			wantForbidden: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate("test-forbidden", tt.toolCalls)

			if len(result.ForbiddenHits) != tt.wantForbidden {
				t.Errorf("Validate() forbidden hits = %d, want %d", len(result.ForbiddenHits), tt.wantForbidden)
			}
		})
	}
}

func TestMockToolExecutor_Execute(t *testing.T) {
	mockResponses := DefaultKubectlMockResponses()
	executor := NewMockToolExecutor(mockResponses, false)

	tests := []struct {
		name    string
		command string
		wantErr bool
	}{
		{
			name:    "kubectl get pods",
			command: "kubectl get pods",
			wantErr: false,
		},
		{
			name:    "kubectl get deployments",
			command: "kubectl get deployments",
			wantErr: false,
		},
		{
			name:    "kubectl create namespace",
			command: "kubectl create namespace test-ns",
			wantErr: false,
		},
		{
			name:    "kubectl apply",
			command: "kubectl apply -f manifest.yaml",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]interface{}{
				"command": tt.command,
			}
			result, err := executor.Execute(context.TODO(), "kubectl", args)

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if result == "" {
				t.Error("Execute() returned empty result")
			}
		})
	}
}

func TestGenericValidation(t *testing.T) {
	config := DryRunConfig{
		Mode:    DryRunToolValidation,
		Verbose: false,
	}
	validator := NewDryRunValidator(config)

	tests := []struct {
		name         string
		toolCalls    []ToolCallRecord
		wantMinScore float64
	}{
		{
			name:         "no tool calls",
			toolCalls:    []ToolCallRecord{},
			wantMinScore: 0.0,
		},
		{
			name: "has kubectl command",
			toolCalls: []ToolCallRecord{
				{
					ID:      "call_1",
					Command: "kubectl get pods",
				},
			},
			wantMinScore: 0.8,
		},
		{
			name: "non-kubectl command",
			toolCalls: []ToolCallRecord{
				{
					ID:      "call_1",
					Command: "echo hello",
				},
			},
			wantMinScore: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test without expectations (uses generic validation)
			result := validator.Validate("unknown-task", tt.toolCalls)

			if result.Score < tt.wantMinScore {
				t.Errorf("genericValidation() score = %v, want >= %v", result.Score, tt.wantMinScore)
			}
		})
	}
}

func TestDryRunBenchmarkReport_FormatMarkdown(t *testing.T) {
	report := &DryRunBenchmarkReport{
		SuiteName:     "Test Suite",
		RunID:         "test-run-123",
		Mode:          "tool-validation",
		StartTime:     time.Now().Add(-time.Minute),
		EndTime:       time.Now(),
		TotalDuration: "1m0s",
		TaskCount:     2,
		Results: []DryRunTaskResult{
			{
				TaskID:     "task-1",
				TaskName:   "Test Task 1",
				Difficulty: DifficultyEasy,
				Success:    true,
				Score:      1.0,
				ToolCalls: []ToolCallRecord{
					{ID: "call_1", Command: "kubectl get pods"},
				},
				MatchedPatterns: []string{"Get pods command"},
			},
			{
				TaskID:     "task-2",
				TaskName:   "Test Task 2",
				Difficulty: DifficultyMedium,
				Success:    false,
				Score:      0.5,
				ToolCalls: []ToolCallRecord{
					{ID: "call_1", Command: "kubectl describe pod"},
				},
				MissedPatterns: []string{"Delete command"},
			},
		},
		Summary: DryRunSummary{
			TotalTasks:   2,
			PassedTasks:  1,
			FailedTasks:  1,
			PassRate:     50.0,
			AverageScore: 0.75,
			ByDifficulty: map[TaskDifficulty]DiffStat{
				DifficultyEasy:   {Total: 1, Passed: 1, PassRate: 100.0, AvgScore: 1.0},
				DifficultyMedium: {Total: 1, Passed: 0, PassRate: 0.0, AvgScore: 0.5},
			},
		},
	}

	md := report.FormatMarkdown()

	// Check that markdown contains expected sections
	expectedSections := []string{
		"# k13d Dry-Run Benchmark Report",
		"## Summary",
		"## Results by Difficulty",
		"## Detailed Results",
		"task-1",
		"task-2",
	}

	for _, section := range expectedSections {
		if !containsString(md, section) {
			t.Errorf("FormatMarkdown() missing section: %s", section)
		}
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsString(s[1:], substr) || s[:len(substr)] == substr)
}
