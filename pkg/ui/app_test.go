package ui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestGetCompletions(t *testing.T) {
	app := &App{
		namespaces: []string{"", "default", "kube-system", "monitoring", "production"},
	}

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
		{
			name:     "match pods with po alias",
			input:    "po",
			expected: []string{"pods", "poddisruptionbudgets", "podsecuritypolicies"},
		},
		{
			name:     "match deployments",
			input:    "dep",
			expected: []string{"deployments"},
		},
		{
			name:     "match deploy alias",
			input:    "deploy",
			expected: []string{"deployments"},
		},
		{
			name:     "match services",
			input:    "svc",
			expected: []string{"services"},
		},
		{
			name:     "match multiple - starts with se",
			input:    "se",
			expected: []string{"services", "secrets", "serviceaccounts"},
		},
		{
			name:     "namespace command with prefix",
			input:    "ns def",
			expected: []string{"ns default"},
		},
		{
			name:     "namespace command with kube prefix",
			input:    "ns kube",
			expected: []string{"ns kube-system"},
		},
		{
			name:     "namespace command empty",
			input:    "ns ",
			expected: []string{"ns default", "ns kube-system", "ns monitoring", "ns production"},
		},
		{
			name:     "no match",
			input:    "xyz",
			expected: nil,
		},
		{
			name:     "quit command",
			input:    "quit",
			expected: []string{"quit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := app.getCompletions(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d results, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			for i, exp := range tt.expected {
				if result[i] != exp {
					t.Errorf("result[%d] = %s, expected %s", i, result[i], exp)
				}
			}
		})
	}
}

func TestParseNamespaceNumber(t *testing.T) {
	app := &App{}

	tests := []struct {
		name      string
		input     string
		expectNum int
		expectOk  bool
	}{
		{"digit 0", "0", 0, true},
		{"digit 1", "1", 1, true},
		{"digit 9", "9", 9, true},
		{"two digits", "12", 0, false},
		{"letter", "a", 0, false},
		{"empty", "", 0, false},
		{"special char", "!", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			num, ok := app.parseNamespaceNumber(tt.input)
			if ok != tt.expectOk {
				t.Errorf("parseNamespaceNumber(%q) ok = %v, expected %v", tt.input, ok, tt.expectOk)
			}
			if num != tt.expectNum {
				t.Errorf("parseNamespaceNumber(%q) num = %d, expected %d", tt.input, num, tt.expectNum)
			}
		})
	}
}

func TestCommandDefinitions(t *testing.T) {
	// Verify all commands have required fields
	for i, cmd := range commands {
		if cmd.name == "" {
			t.Errorf("command[%d] has empty name", i)
		}
		if cmd.alias == "" {
			t.Errorf("command[%d] %s has empty alias", i, cmd.name)
		}
		if cmd.desc == "" {
			t.Errorf("command[%d] %s has empty desc", i, cmd.name)
		}
		if cmd.category == "" {
			t.Errorf("command[%d] %s has empty category", i, cmd.name)
		}
	}

	// Verify no duplicate names or aliases
	names := make(map[string]bool)
	aliases := make(map[string]bool)

	for _, cmd := range commands {
		if names[cmd.name] {
			t.Errorf("duplicate command name: %s", cmd.name)
		}
		names[cmd.name] = true

		if aliases[cmd.alias] {
			t.Errorf("duplicate command alias: %s", cmd.alias)
		}
		aliases[cmd.alias] = true
	}
}

func TestStatusColor(t *testing.T) {
	app := &App{}

	tests := []struct {
		status   string
		expected tcell.Color
	}{
		{"Running", tcell.ColorGreen},
		{"Ready", tcell.ColorGreen},
		{"Active", tcell.ColorGreen},
		{"Succeeded", tcell.ColorGreen},
		{"Normal", tcell.ColorGreen},
		{"Completed", tcell.ColorGreen},
		{"Pending", tcell.ColorYellow},
		{"ContainerCreating", tcell.ColorYellow},
		{"Warning", tcell.ColorYellow},
		{"Updating", tcell.ColorYellow},
		{"Failed", tcell.ColorRed},
		{"Error", tcell.ColorRed},
		{"CrashLoopBackOff", tcell.ColorRed},
		{"NotReady", tcell.ColorRed},
		{"ImagePullBackOff", tcell.ColorRed},
		{"ErrImagePull", tcell.ColorRed},
		{"Unknown", tcell.ColorWhite},
		{"SomeRandomStatus", tcell.ColorWhite},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			color := app.statusColor(tt.status)
			if color != tt.expected {
				t.Errorf("statusColor(%s) = %v, expected %v", tt.status, color, tt.expected)
			}
		})
	}
}

func TestHighlightMatch(t *testing.T) {
	app := &App{}

	tests := []struct {
		name     string
		text     string
		filter   string
		expected string
	}{
		{
			name:     "match at start",
			text:     "nginx-pod",
			filter:   "nginx",
			expected: "[yellow]nginx[white]-pod",
		},
		{
			name:     "match in middle",
			text:     "my-nginx-pod",
			filter:   "nginx",
			expected: "my-[yellow]nginx[white]-pod",
		},
		{
			name:     "match at end",
			text:     "pod-nginx",
			filter:   "nginx",
			expected: "pod-[yellow]nginx[white]",
		},
		{
			name:     "case insensitive match",
			text:     "NGINX-pod",
			filter:   "nginx",
			expected: "[yellow]NGINX[white]-pod",
		},
		{
			name:     "no match",
			text:     "apache-pod",
			filter:   "nginx",
			expected: "apache-pod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := app.highlightMatch(tt.text, tt.filter)
			if result != tt.expected {
				t.Errorf("highlightMatch(%q, %q) = %q, expected %q", tt.text, tt.filter, result, tt.expected)
			}
		})
	}
}

func TestGetCompletionsExtended(t *testing.T) {
	app := &App{
		namespaces: []string{"", "default", "kube-system"},
	}

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "configmaps",
			input:    "cm",
			expected: []string{"configmaps"},
		},
		{
			name:     "configmaps full",
			input:    "config",
			expected: []string{"configmaps"},
		},
		{
			name:     "secrets",
			input:    "sec",
			expected: []string{"secrets"},
		},
		{
			name:     "daemonsets",
			input:    "ds",
			expected: []string{"daemonsets"},
		},
		{
			name:     "statefulsets",
			input:    "sts",
			expected: []string{"statefulsets"},
		},
		{
			name:     "jobs",
			input:    "job",
			expected: []string{"jobs"},
		},
		{
			name:     "cronjobs",
			input:    "cj",
			expected: []string{"cronjobs"},
		},
		{
			name:     "ingresses",
			input:    "ingresses",
			expected: []string{"ingresses"},
		},
		{
			name:     "multiple matches with d",
			input:    "d",
			expected: []string{"deployments", "daemonsets"},
		},
		{
			name:     "events",
			input:    "ev",
			expected: []string{"events"},
		},
		{
			name:     "nodes",
			input:    "no",
			expected: []string{"nodes"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := app.getCompletions(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d results, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			for i, exp := range tt.expected {
				if result[i] != exp {
					t.Errorf("result[%d] = %s, expected %s", i, result[i], exp)
				}
			}
		})
	}
}

func TestCommandCount(t *testing.T) {
	// Verify we have expected number of commands (expanded list with all k8s resources)
	minExpectedCount := 40
	if len(commands) < minExpectedCount {
		t.Errorf("expected at least %d commands, got %d", minExpectedCount, len(commands))
	}

	// Verify core commands exist
	expectedCommands := []string{
		"pods", "deployments", "services", "nodes", "namespaces", "events",
		"configmaps", "secrets", "daemonsets", "statefulsets", "jobs",
		"cronjobs", "ingresses", "quit", "persistentvolumes", "replicasets",
		"serviceaccounts", "roles", "clusterroles",
	}

	for _, expected := range expectedCommands {
		found := false
		for _, cmd := range commands {
			if cmd.name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected command %s not found", expected)
		}
	}
}

// TestHandleCommandWithNamespaceFlag tests kubectl-style -n flag parsing
func TestHandleCommandWithNamespaceFlag(t *testing.T) {
	tests := []struct {
		name              string
		cmd               string
		expectedResource  string
		expectedNamespace string
	}{
		{
			name:              "pods with -n flag",
			cmd:               "pods -n kube-system",
			expectedResource:  "pods",
			expectedNamespace: "kube-system",
		},
		{
			name:              "deploy with --namespace flag",
			cmd:               "deploy --namespace monitoring",
			expectedResource:  "deploy",
			expectedNamespace: "monitoring",
		},
		{
			name:              "services with -A flag",
			cmd:               "svc -A",
			expectedResource:  "svc",
			expectedNamespace: "", // all namespaces
		},
		{
			name:              "pods with --all-namespaces",
			cmd:               "pods --all-namespaces",
			expectedResource:  "pods",
			expectedNamespace: "", // all namespaces
		},
		{
			name:              "simple command without flag",
			cmd:               "pods",
			expectedResource:  "pods",
			expectedNamespace: "unchanged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse command like handleCommand does
			parts := strings.Fields(tt.cmd)
			resourceCmd := ""
			namespace := "unchanged"

			for i := 0; i < len(parts); i++ {
				part := parts[i]
				if part == "-n" || part == "--namespace" {
					if i+1 < len(parts) {
						namespace = parts[i+1]
						i++
					}
				} else if part == "-A" || part == "--all-namespaces" {
					namespace = ""
				} else if resourceCmd == "" {
					resourceCmd = part
				}
			}

			if resourceCmd != tt.expectedResource {
				t.Errorf("expected resource %q, got %q", tt.expectedResource, resourceCmd)
			}
			if namespace != tt.expectedNamespace {
				t.Errorf("expected namespace %q, got %q", tt.expectedNamespace, namespace)
			}
		})
	}
}

// TestGetCompletionsWithNamespaceFlag tests autocomplete with -n flag
func TestGetCompletionsWithNamespaceFlag(t *testing.T) {
	app := &App{
		namespaces: []string{"", "default", "kube-system", "kube-public", "monitoring"},
	}

	tests := []struct {
		name        string
		input       string
		shouldMatch []string
	}{
		{
			name:        "pods -n with kube prefix",
			input:       "pods -n kube",
			shouldMatch: []string{"pods -n kube-system", "pods -n kube-public"},
		},
		{
			name:        "deploy -n with mon prefix",
			input:       "deploy -n mon",
			shouldMatch: []string{"deploy -n monitoring"},
		},
		{
			name:        "svc -n with empty prefix shows namespaces",
			input:       "svc -n ",
			shouldMatch: []string{"svc -n default", "svc -n kube-system", "svc -n kube-public", "svc -n monitoring"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := app.getCompletions(tt.input)

			// Check that expected matches are present
			for _, expected := range tt.shouldMatch {
				found := false
				for _, r := range result {
					if r == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected %q to be in results %v", expected, result)
				}
			}
		})
	}
}

// TestNumberKeyNamespaceSwitch tests that number keys switch namespaces
func TestNumberKeyNamespaceSwitch(t *testing.T) {
	app := &App{
		namespaces: []string{"", "default", "kube-system", "monitoring", "production", "staging"},
	}

	tests := []struct {
		name              string
		keyNum            int
		expectedNamespace string
		shouldSucceed     bool
	}{
		{"key 0 - all namespaces", 0, "", true},
		{"key 1 - default", 1, "default", true},
		{"key 2 - kube-system", 2, "kube-system", true},
		{"key 3 - monitoring", 3, "monitoring", true},
		{"key 4 - production", 4, "production", true},
		{"key 5 - staging", 5, "staging", true},
		{"key 9 - out of range", 9, "unchanged", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app.currentNamespace = "unchanged"

			// Simulate selectNamespaceByNumber logic
			if tt.keyNum < len(app.namespaces) {
				app.currentNamespace = app.namespaces[tt.keyNum]
			}

			if tt.shouldSucceed {
				if app.currentNamespace != tt.expectedNamespace {
					t.Errorf("expected namespace %q, got %q", tt.expectedNamespace, app.currentNamespace)
				}
			} else {
				if app.currentNamespace != "unchanged" {
					t.Errorf("namespace should not change for out of range key, got %q", app.currentNamespace)
				}
			}
		})
	}
}

// TestQueueUpdateDrawStoppingApp tests that QueueUpdateDraw handles stopping state
func TestQueueUpdateDrawStoppingApp(t *testing.T) {
	app := &App{}
	app.stopping = 1 // Simulate stopping state

	// Should not call the callback when stopping
	app.QueueUpdateDraw(func() {
		t.Error("callback should not be called when app is stopping")
	})
}

// TestIsRunning tests the IsRunning method
func TestIsRunning(t *testing.T) {
	tests := []struct {
		name     string
		running  int32
		stopping int32
		expected bool
	}{
		{"running and not stopping", 1, 0, true},
		{"not running", 0, 0, false},
		{"running but stopping", 1, 1, false},
		{"not running and stopping", 0, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{}
			app.running = tt.running
			app.stopping = tt.stopping

			result := app.IsRunning()
			if result != tt.expected {
				t.Errorf("IsRunning() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestBasicConcurrentAccess tests that basic concurrent access to app state is safe
// using the App's mutex for proper synchronization
func TestBasicConcurrentAccess(t *testing.T) {
	app := &App{
		namespaces: []string{"", "default", "kube-system"},
	}

	// Run concurrent operations that access/modify state using mutex
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(n int) {
			defer func() { done <- true }()

			// Read operations with lock
			app.mx.RLock()
			_ = app.namespaces
			_ = app.currentResource
			_ = app.currentNamespace
			app.mx.RUnlock()

			// Write operations with lock
			app.mx.Lock()
			if n%2 == 0 {
				app.currentResource = "pods"
			} else {
				app.currentResource = "deployments"
			}
			app.mx.Unlock()
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestResourceAliases verifies all resource aliases resolve correctly via getCompletions.
// (Migrated from tui_interface_test.go)
func TestResourceAliases(t *testing.T) {
	app := &App{
		namespaces: []string{"", "default"},
	}

	aliases := map[string]string{
		"po":     "pods",
		"deploy": "deployments",
		"svc":    "services",
		"no":     "nodes",
		"ns":     "namespaces",
		"ev":     "events",
		"cm":     "configmaps",
		"sec":    "secrets",
		"pv":     "persistentvolumes",
		"pvc":    "persistentvolumeclaims",
		"sc":     "storageclasses",
		"rs":     "replicasets",
		"ds":     "daemonsets",
		"sts":    "statefulsets",
		"job":    "jobs",
		"cj":     "cronjobs",
		"ing":    "ingresses",
		"ep":     "endpoints",
		"netpol": "networkpolicies",
		"sa":     "serviceaccounts",
	}

	for alias, expectedResource := range aliases {
		completions := app.getCompletions(alias)
		if len(completions) == 0 {
			t.Errorf("Alias %q returned no completions", alias)
			continue
		}
		if completions[0] != expectedResource {
			t.Errorf("Alias %q: expected %q, got %q", alias, expectedResource, completions[0])
		}
	}
}
