package bench

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_LoadTasks(t *testing.T) {
	// Create temp directory with test tasks
	tempDir := t.TempDir()

	// Create test task 1
	task1Dir := filepath.Join(tempDir, "test-task-1")
	if err := os.MkdirAll(task1Dir, 0755); err != nil {
		t.Fatal(err)
	}

	task1YAML := `
name: Test Task 1
description: A simple test task
category: testing
difficulty: easy
tags:
  - test
  - simple
timeout: 5m
script:
  - prompt: "Create a test resource"
verifier: verify.sh
expect:
  - contains: "created"
`
	if err := os.WriteFile(filepath.Join(task1Dir, "task.yaml"), []byte(task1YAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create test task 2
	task2Dir := filepath.Join(tempDir, "test-task-2")
	if err := os.MkdirAll(task2Dir, 0755); err != nil {
		t.Fatal(err)
	}

	task2YAML := `
name: Test Task 2
description: A medium difficulty task
category: troubleshooting
difficulty: medium
disabled: false
script:
  - prompt: "Fix the broken resource"
verifier: verify.sh
`
	if err := os.WriteFile(filepath.Join(task2Dir, "task.yaml"), []byte(task2YAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Load tasks
	loader := NewLoader(tempDir)
	tasks, err := loader.LoadTasks()
	if err != nil {
		t.Fatalf("LoadTasks failed: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}

	// Verify task properties
	taskMap := make(map[string]*Task)
	for _, task := range tasks {
		taskMap[task.ID] = task
	}

	task1, ok := taskMap["test-task-1"]
	if !ok {
		t.Error("test-task-1 not found")
	} else {
		if task1.Name != "Test Task 1" {
			t.Errorf("Expected name 'Test Task 1', got '%s'", task1.Name)
		}
		if task1.Difficulty != DifficultyEasy {
			t.Errorf("Expected difficulty 'easy', got '%s'", task1.Difficulty)
		}
		if task1.Category != "testing" {
			t.Errorf("Expected category 'testing', got '%s'", task1.Category)
		}
		if len(task1.Script) != 1 {
			t.Errorf("Expected 1 prompt, got %d", len(task1.Script))
		}
	}

	task2, ok := taskMap["test-task-2"]
	if !ok {
		t.Error("test-task-2 not found")
	} else {
		if task2.Difficulty != DifficultyMedium {
			t.Errorf("Expected difficulty 'medium', got '%s'", task2.Difficulty)
		}
	}
}

func TestLoader_FilterTasks(t *testing.T) {
	tasks := []*Task{
		{ID: "easy-1", Difficulty: DifficultyEasy, Category: "creation", Tags: []string{"pods"}},
		{ID: "easy-2", Difficulty: DifficultyEasy, Category: "creation", Tags: []string{"services"}},
		{ID: "medium-1", Difficulty: DifficultyMedium, Category: "troubleshooting", Tags: []string{"pods", "debugging"}},
		{ID: "hard-1", Difficulty: DifficultyHard, Category: "networking", Tags: []string{"ingress"}},
		{ID: "disabled-1", Difficulty: DifficultyEasy, Disabled: true},
	}

	loader := NewLoader("")

	// Filter by difficulty
	filtered, err := loader.FilterTasks(tasks, FilterOptions{Difficulty: "easy"})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 2 {
		t.Errorf("Expected 2 easy tasks (excluding disabled), got %d", len(filtered))
	}

	// Filter by category
	filtered, err = loader.FilterTasks(tasks, FilterOptions{Categories: []string{"creation"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 2 {
		t.Errorf("Expected 2 creation tasks, got %d", len(filtered))
	}

	// Filter by tags
	filtered, err = loader.FilterTasks(tasks, FilterOptions{Tags: []string{"pods"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 2 {
		t.Errorf("Expected 2 tasks with 'pods' tag, got %d", len(filtered))
	}

	// Filter by pattern
	filtered, err = loader.FilterTasks(tasks, FilterOptions{Pattern: "easy-.*"})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 2 {
		t.Errorf("Expected 2 tasks matching 'easy-.*', got %d", len(filtered))
	}

	// Include disabled
	filtered, err = loader.FilterTasks(tasks, FilterOptions{
		Difficulty:      "easy",
		IncludeDisabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 3 {
		t.Errorf("Expected 3 easy tasks (including disabled), got %d", len(filtered))
	}
}

func TestLoader_LoadPromptFile(t *testing.T) {
	tempDir := t.TempDir()
	taskDir := filepath.Join(tempDir, "prompt-task")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create prompt file
	promptContent := "This is a prompt loaded from a file.\nIt can be multi-line."
	if err := os.WriteFile(filepath.Join(taskDir, "prompt.txt"), []byte(promptContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create task with promptFile reference
	taskYAML := `
name: Prompt File Task
script:
  - promptFile: prompt.txt
verifier: verify.sh
`
	if err := os.WriteFile(filepath.Join(taskDir, "task.yaml"), []byte(taskYAML), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tempDir)
	tasks, err := loader.LoadTasks()
	if err != nil {
		t.Fatalf("LoadTasks failed: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(tasks))
	}

	task := tasks[0]
	if len(task.Script) != 1 {
		t.Fatalf("Expected 1 prompt, got %d", len(task.Script))
	}

	if task.Script[0].Text != promptContent {
		t.Errorf("Prompt content mismatch.\nExpected: %s\nGot: %s", promptContent, task.Script[0].Text)
	}
}

func TestTask_GetPaths(t *testing.T) {
	task := &Task{
		Dir:      "/path/to/task",
		Setup:    "setup.sh",
		Verifier: "verify.sh",
		Cleanup:  "cleanup.sh",
	}

	if got := task.GetSetupPath(); got != "/path/to/task/setup.sh" {
		t.Errorf("GetSetupPath() = %s, want /path/to/task/setup.sh", got)
	}

	if got := task.GetVerifierPath(); got != "/path/to/task/verify.sh" {
		t.Errorf("GetVerifierPath() = %s, want /path/to/task/verify.sh", got)
	}

	if got := task.GetCleanupPath(); got != "/path/to/task/cleanup.sh" {
		t.Errorf("GetCleanupPath() = %s, want /path/to/task/cleanup.sh", got)
	}

	// Test empty paths
	emptyTask := &Task{Dir: "/path/to/task"}
	if got := emptyTask.GetSetupPath(); got != "" {
		t.Errorf("GetSetupPath() = %s, want empty string", got)
	}
}

func TestTask_GetTaskScript(t *testing.T) {
	task := &Task{
		Script: []Prompt{
			{Text: "First prompt"},
			{Text: "Second prompt"},
			{Text: "Third prompt"},
		},
	}

	expected := "First prompt\nSecond prompt\nThird prompt"
	if got := task.GetTaskScript(); got != expected {
		t.Errorf("GetTaskScript() = %s, want %s", got, expected)
	}
}
