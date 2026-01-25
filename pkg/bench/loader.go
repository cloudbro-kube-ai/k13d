package bench

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Loader handles loading benchmark tasks from the filesystem
type Loader struct {
	baseDir string
}

// NewLoader creates a new task loader
func NewLoader(baseDir string) *Loader {
	// Convert to absolute path to ensure scripts can be found
	absDir, err := filepath.Abs(baseDir)
	if err != nil {
		absDir = baseDir // Fallback to relative if Abs fails
	}
	return &Loader{baseDir: absDir}
}

// LoadTasks loads all tasks from the base directory
// Tasks are expected to be in subdirectories with task.yaml files
func (l *Loader) LoadTasks() ([]*Task, error) {
	entries, err := os.ReadDir(l.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read task directory %s: %w", l.baseDir, err)
	}

	var tasks []*Task
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		taskDir := filepath.Join(l.baseDir, entry.Name())
		taskFile := filepath.Join(taskDir, "task.yaml")

		// Check if task.yaml exists
		if _, err := os.Stat(taskFile); os.IsNotExist(err) {
			continue
		}

		task, err := l.loadTask(taskDir)
		if err != nil {
			return nil, fmt.Errorf("failed to load task %s: %w", entry.Name(), err)
		}

		// Set task ID to directory name if not specified
		if task.ID == "" {
			task.ID = entry.Name()
		}
		task.Dir = taskDir

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// LoadTask loads a single task from a directory
func (l *Loader) loadTask(taskDir string) (*Task, error) {
	taskFile := filepath.Join(taskDir, "task.yaml")
	data, err := os.ReadFile(taskFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read task file: %w", err)
	}

	var task Task
	if err := yaml.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to parse task YAML: %w", err)
	}

	// Load prompt files if specified
	for i, prompt := range task.Script {
		if prompt.File != "" {
			promptPath := filepath.Join(taskDir, prompt.File)
			content, err := os.ReadFile(promptPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read prompt file %s: %w", prompt.File, err)
			}
			task.Script[i].Text = string(content)
		}
	}

	// Validate required fields
	if len(task.Script) == 0 {
		return nil, fmt.Errorf("task must have at least one prompt in script")
	}

	// Set defaults
	if task.Timeout == "" {
		task.Timeout = "10m"
	}
	if task.Difficulty == "" {
		task.Difficulty = DifficultyMedium
	}

	return &task, nil
}

// FilterTasks filters tasks based on the given criteria
func (l *Loader) FilterTasks(tasks []*Task, opts FilterOptions) ([]*Task, error) {
	var filtered []*Task

	// Compile pattern regex if provided
	var patternRe *regexp.Regexp
	if opts.Pattern != "" {
		var err error
		patternRe, err = regexp.Compile(opts.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid task pattern regex: %w", err)
		}
	}

	for _, task := range tasks {
		// Skip disabled tasks
		if task.Disabled && !opts.IncludeDisabled {
			continue
		}

		// Filter by pattern
		if patternRe != nil && !patternRe.MatchString(task.ID) {
			continue
		}

		// Filter by difficulty
		if opts.Difficulty != "" && string(task.Difficulty) != opts.Difficulty {
			continue
		}

		// Filter by category
		if len(opts.Categories) > 0 && !containsString(opts.Categories, task.Category) {
			continue
		}

		// Filter by tags
		if len(opts.Tags) > 0 && !hasAnyTag(task.Tags, opts.Tags) {
			continue
		}

		filtered = append(filtered, task)
	}

	return filtered, nil
}

// FilterOptions specifies criteria for filtering tasks
type FilterOptions struct {
	Pattern         string   // Regex pattern to match task IDs
	Difficulty      string   // Filter by difficulty level
	Categories      []string // Filter by categories
	Tags            []string // Filter by tags
	IncludeDisabled bool     // Include disabled tasks
}

// GetTaskScript returns all prompts concatenated for a task
func (t *Task) GetTaskScript() string {
	var prompts []string
	for _, p := range t.Script {
		prompts = append(prompts, p.Text)
	}
	return strings.Join(prompts, "\n")
}

// GetSetupPath returns the full path to the setup script
func (t *Task) GetSetupPath() string {
	if t.Setup == "" {
		return ""
	}
	return filepath.Join(t.Dir, t.Setup)
}

// GetVerifierPath returns the full path to the verifier script
func (t *Task) GetVerifierPath() string {
	if t.Verifier == "" {
		return ""
	}
	return filepath.Join(t.Dir, t.Verifier)
}

// GetCleanupPath returns the full path to the cleanup script
func (t *Task) GetCleanupPath() string {
	if t.Cleanup == "" {
		return ""
	}
	return filepath.Join(t.Dir, t.Cleanup)
}

// HasArtifacts checks if the task has an artifacts directory
func (t *Task) HasArtifacts() bool {
	artifactsDir := filepath.Join(t.Dir, "artifacts")
	info, err := os.Stat(artifactsDir)
	return err == nil && info.IsDir()
}

// GetArtifactsDir returns the path to the artifacts directory
func (t *Task) GetArtifactsDir() string {
	return filepath.Join(t.Dir, "artifacts")
}

// helper functions

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func hasAnyTag(taskTags, filterTags []string) bool {
	for _, ft := range filterTags {
		for _, tt := range taskTags {
			if tt == ft {
				return true
			}
		}
	}
	return false
}
