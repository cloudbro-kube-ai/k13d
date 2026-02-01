// Package eval provides AI agent benchmark evaluation framework
// dryrun.go implements cluster-free benchmark evaluation using tool call analysis
package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// DryRunMode represents the type of dry-run evaluation
type DryRunMode string

const (
	// DryRunToolValidation validates tool calls without execution
	DryRunToolValidation DryRunMode = "tool-validation"
	// DryRunMockResponses uses mock responses for tool calls
	DryRunMockResponses DryRunMode = "mock-responses"
	// DryRunCommandAnalysis only analyzes the generated commands
	DryRunCommandAnalysis DryRunMode = "command-analysis"
)

// DryRunConfig holds configuration for dry-run benchmark
type DryRunConfig struct {
	Mode          DryRunMode
	MockResponses map[string]MockResponse // Mock responses keyed by command pattern
	Verbose       bool
}

// MockResponse defines a mock response for a command pattern
type MockResponse struct {
	Pattern      string `yaml:"pattern" json:"pattern"`                                 // Regex pattern to match command
	Response     string `yaml:"response" json:"response"`                               // Mock response to return
	ExitCode     int    `yaml:"exit_code" json:"exit_code"`                             // Mock exit code (0 = success)
	Delay        string `yaml:"delay" json:"delay"`                                     // Optional simulated delay
	IsError      bool   `yaml:"is_error" json:"is_error"`                               // Whether this is an error response
	ResourceYAML string `yaml:"resource_yaml,omitempty" json:"resource_yaml,omitempty"` // Mock YAML output
}

// ToolCallRecord records a tool call made by the AI
type ToolCallRecord struct {
	ID        string                 `json:"id"`
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
	Command   string                 `json:"command,omitempty"` // Extracted command if kubectl/bash
	Timestamp time.Time              `json:"timestamp"`
}

// DryRunTaskExpectation defines what tool calls are expected for a task
type DryRunTaskExpectation struct {
	// Required tool calls (must be present)
	RequiredCommands []CommandExpectation `yaml:"required_commands" json:"required_commands"`

	// Forbidden patterns (must NOT be present)
	ForbiddenPatterns []string `yaml:"forbidden_patterns" json:"forbidden_patterns"`

	// Expected resource creation
	ExpectedResources []ResourceExpectation `yaml:"expected_resources" json:"expected_resources"`

	// Minimum/maximum tool calls
	MinToolCalls int `yaml:"min_tool_calls" json:"min_tool_calls"`
	MaxToolCalls int `yaml:"max_tool_calls" json:"max_tool_calls"`
}

// CommandExpectation defines an expected command pattern
type CommandExpectation struct {
	Pattern     string   `yaml:"pattern" json:"pattern"`         // Regex pattern
	Description string   `yaml:"description" json:"description"` // Human-readable description
	Required    bool     `yaml:"required" json:"required"`       // Is this command required?
	Order       int      `yaml:"order" json:"order"`             // Expected order (0 = any)
	Args        []string `yaml:"args" json:"args"`               // Required arguments
}

// ResourceExpectation defines an expected K8s resource
type ResourceExpectation struct {
	Kind       string            `yaml:"kind" json:"kind"`             // e.g., "Pod", "Deployment"
	Name       string            `yaml:"name" json:"name"`             // Resource name pattern
	Namespace  string            `yaml:"namespace" json:"namespace"`   // Namespace pattern
	Labels     map[string]string `yaml:"labels" json:"labels"`         // Required labels
	Containers []string          `yaml:"containers" json:"containers"` // Required container names
	Image      string            `yaml:"image" json:"image"`           // Required image pattern
}

// DryRunResult contains the result of a dry-run evaluation
type DryRunResult struct {
	TaskID          string           `json:"task_id"`
	Success         bool             `json:"success"`
	ToolCalls       []ToolCallRecord `json:"tool_calls"`
	MatchedPatterns []string         `json:"matched_patterns"`
	MissedPatterns  []string         `json:"missed_patterns"`
	ForbiddenHits   []string         `json:"forbidden_hits"`
	Errors          []string         `json:"errors"`
	Score           float64          `json:"score"` // 0.0 - 1.0
	Duration        time.Duration    `json:"duration"`
	LLMResponse     string           `json:"llm_response,omitempty"`
}

// DryRunValidator validates tool calls against expectations
type DryRunValidator struct {
	config       DryRunConfig
	expectations map[string]DryRunTaskExpectation
}

// NewDryRunValidator creates a new dry-run validator
func NewDryRunValidator(config DryRunConfig) *DryRunValidator {
	return &DryRunValidator{
		config:       config,
		expectations: make(map[string]DryRunTaskExpectation),
	}
}

// SetExpectation sets the expectation for a task
func (v *DryRunValidator) SetExpectation(taskID string, exp DryRunTaskExpectation) {
	v.expectations[taskID] = exp
}

// Validate validates tool calls against task expectations
func (v *DryRunValidator) Validate(taskID string, toolCalls []ToolCallRecord) *DryRunResult {
	result := &DryRunResult{
		TaskID:    taskID,
		ToolCalls: toolCalls,
		Success:   true,
	}

	exp, ok := v.expectations[taskID]
	if !ok {
		// No specific expectations, use generic validation
		result.Score = v.genericValidation(toolCalls)
		return result
	}

	// Check required commands
	for _, cmdExp := range exp.RequiredCommands {
		if cmdExp.Required {
			found := false
			for _, tc := range toolCalls {
				if v.matchCommand(tc.Command, cmdExp) {
					found = true
					result.MatchedPatterns = append(result.MatchedPatterns, cmdExp.Description)
					break
				}
			}
			if !found {
				result.Success = false
				result.MissedPatterns = append(result.MissedPatterns, cmdExp.Description)
			}
		}
	}

	// Check forbidden patterns
	for _, pattern := range exp.ForbiddenPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("invalid forbidden pattern: %s", pattern))
			continue
		}
		for _, tc := range toolCalls {
			if re.MatchString(tc.Command) {
				result.Success = false
				result.ForbiddenHits = append(result.ForbiddenHits,
					fmt.Sprintf("command '%s' matches forbidden pattern '%s'", tc.Command, pattern))
			}
		}
	}

	// Check tool call count limits
	if exp.MinToolCalls > 0 && len(toolCalls) < exp.MinToolCalls {
		result.Success = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("too few tool calls: got %d, expected at least %d", len(toolCalls), exp.MinToolCalls))
	}
	if exp.MaxToolCalls > 0 && len(toolCalls) > exp.MaxToolCalls {
		result.Success = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("too many tool calls: got %d, expected at most %d", len(toolCalls), exp.MaxToolCalls))
	}

	// Calculate score
	result.Score = v.calculateScore(result, exp)

	return result
}

// matchCommand checks if a command matches an expectation
func (v *DryRunValidator) matchCommand(cmd string, exp CommandExpectation) bool {
	re, err := regexp.Compile(exp.Pattern)
	if err != nil {
		return false
	}

	if !re.MatchString(cmd) {
		return false
	}

	// Check required arguments
	for _, arg := range exp.Args {
		if !strings.Contains(cmd, arg) {
			return false
		}
	}

	return true
}

// genericValidation performs basic validation without specific expectations
func (v *DryRunValidator) genericValidation(toolCalls []ToolCallRecord) float64 {
	if len(toolCalls) == 0 {
		return 0.0
	}

	score := 0.5 // Base score for making any tool calls

	// Check for kubectl usage
	hasKubectl := false
	for _, tc := range toolCalls {
		if strings.HasPrefix(tc.Command, "kubectl") {
			hasKubectl = true
			break
		}
	}
	if hasKubectl {
		score += 0.3
	}

	// Penalize if there are errors
	hasErrors := false
	for _, tc := range toolCalls {
		if strings.Contains(strings.ToLower(tc.Command), "error") {
			hasErrors = true
			break
		}
	}
	if !hasErrors {
		score += 0.2
	}

	return score
}

// calculateScore calculates the final score based on results
func (v *DryRunValidator) calculateScore(result *DryRunResult, exp DryRunTaskExpectation) float64 {
	if len(exp.RequiredCommands) == 0 {
		return v.genericValidation(result.ToolCalls)
	}

	requiredCount := 0
	for _, cmd := range exp.RequiredCommands {
		if cmd.Required {
			requiredCount++
		}
	}

	if requiredCount == 0 {
		return 1.0
	}

	matchedCount := len(result.MatchedPatterns)
	baseScore := float64(matchedCount) / float64(requiredCount)

	// Penalize forbidden hits
	forbiddenPenalty := float64(len(result.ForbiddenHits)) * 0.1
	baseScore -= forbiddenPenalty

	if baseScore < 0 {
		baseScore = 0
	}
	if baseScore > 1 {
		baseScore = 1
	}

	return baseScore
}

// MockToolExecutor simulates tool execution with mock responses
type MockToolExecutor struct {
	responses []MockResponse
	verbose   bool
}

// NewMockToolExecutor creates a new mock tool executor
func NewMockToolExecutor(responses []MockResponse, verbose bool) *MockToolExecutor {
	return &MockToolExecutor{
		responses: responses,
		verbose:   verbose,
	}
}

// Execute returns a mock response for a command
func (m *MockToolExecutor) Execute(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
	command := ""
	if cmd, ok := args["command"].(string); ok {
		command = cmd
	}

	// Find matching mock response
	for _, resp := range m.responses {
		re, err := regexp.Compile(resp.Pattern)
		if err != nil {
			continue
		}

		if re.MatchString(command) {
			// Simulate delay if specified
			if resp.Delay != "" {
				if delay, err := time.ParseDuration(resp.Delay); err == nil {
					time.Sleep(delay)
				}
			}

			if resp.IsError {
				return resp.Response, fmt.Errorf("command failed: %s", resp.Response)
			}

			return resp.Response, nil
		}
	}

	// Default mock responses for common commands
	return m.getDefaultResponse(command), nil
}

// getDefaultResponse returns a default mock response for common kubectl commands
func (m *MockToolExecutor) getDefaultResponse(command string) string {
	cmd := strings.ToLower(command)

	// kubectl get commands
	if strings.Contains(cmd, "kubectl get") {
		if strings.Contains(cmd, "pods") || strings.Contains(cmd, "pod") {
			return `NAME                    READY   STATUS    RESTARTS   AGE
nginx-deployment-abc    1/1     Running   0          5m
nginx-deployment-def    1/1     Running   0          5m`
		}
		if strings.Contains(cmd, "deployments") || strings.Contains(cmd, "deploy") {
			return `NAME               READY   UP-TO-DATE   AVAILABLE   AGE
nginx-deployment   2/2     2            2           10m`
		}
		if strings.Contains(cmd, "services") || strings.Contains(cmd, "svc") {
			return `NAME         TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)   AGE
kubernetes   ClusterIP   10.96.0.1      <none>        443/TCP   30d
nginx-svc    ClusterIP   10.96.100.1    <none>        80/TCP    5m`
		}
		if strings.Contains(cmd, "nodes") || strings.Contains(cmd, "node") {
			return `NAME           STATUS   ROLES           AGE   VERSION
docker-desktop Ready    control-plane   30d   v1.28.0`
		}
		if strings.Contains(cmd, "namespaces") || strings.Contains(cmd, "ns") {
			return `NAME              STATUS   AGE
default           Active   30d
kube-system       Active   30d
kube-public       Active   30d`
		}
		// Generic get response
		return "NAME              STATUS   AGE\nresource-1        Active   5m"
	}

	// kubectl apply/create commands
	if strings.Contains(cmd, "kubectl apply") || strings.Contains(cmd, "kubectl create") {
		if strings.Contains(cmd, "namespace") || strings.Contains(cmd, "ns") {
			return "namespace/test-ns created"
		}
		if strings.Contains(cmd, "deployment") {
			return "deployment.apps/nginx-deployment created"
		}
		if strings.Contains(cmd, "service") || strings.Contains(cmd, "svc") {
			return "service/nginx-svc created"
		}
		if strings.Contains(cmd, "pod") {
			return "pod/nginx-pod created"
		}
		if strings.Contains(cmd, "configmap") || strings.Contains(cmd, "cm") {
			return "configmap/my-config created"
		}
		if strings.Contains(cmd, "secret") {
			return "secret/my-secret created"
		}
		return "resource created"
	}

	// kubectl delete commands
	if strings.Contains(cmd, "kubectl delete") {
		return "resource deleted"
	}

	// kubectl describe commands
	if strings.Contains(cmd, "kubectl describe") {
		return `Name:         nginx-pod
Namespace:    default
Status:       Running
IP:           10.244.0.5
Containers:
  nginx:
    Image:          nginx:1.25
    State:          Running
    Ready:          True
Events:
  Normal  Scheduled  5m   default-scheduler  Successfully assigned default/nginx-pod to docker-desktop
  Normal  Pulled     5m   kubelet            Container image "nginx:1.25" already present on machine
  Normal  Created    5m   kubelet            Created container nginx
  Normal  Started    5m   kubelet            Started container nginx`
	}

	// kubectl scale commands
	if strings.Contains(cmd, "kubectl scale") {
		return "deployment.apps/nginx-deployment scaled"
	}

	// kubectl logs commands
	if strings.Contains(cmd, "kubectl logs") {
		return `2024-01-01 10:00:00 [info] Starting nginx...
2024-01-01 10:00:01 [info] nginx started successfully`
	}

	// kubectl exec commands
	if strings.Contains(cmd, "kubectl exec") {
		return "command executed successfully"
	}

	// kubectl label/annotate commands
	if strings.Contains(cmd, "kubectl label") {
		return "pod/nginx-pod labeled"
	}
	if strings.Contains(cmd, "kubectl annotate") {
		return "pod/nginx-pod annotated"
	}

	// kubectl rollout commands
	if strings.Contains(cmd, "kubectl rollout") {
		if strings.Contains(cmd, "status") {
			return "deployment \"nginx-deployment\" successfully rolled out"
		}
		if strings.Contains(cmd, "restart") {
			return "deployment.apps/nginx-deployment restarted"
		}
		return "rollout completed"
	}

	// Default response
	return "OK"
}

// ExtractCommandFromToolCall extracts the command from a tool call
func ExtractCommandFromToolCall(toolName string, argsJSON string) (string, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", err
	}

	if cmd, ok := args["command"].(string); ok {
		return cmd, nil
	}

	return "", fmt.Errorf("no command found in arguments")
}

// DryRunBenchmarkTask represents a task configured for dry-run evaluation
type DryRunBenchmarkTask struct {
	BenchmarkTask
	DryRunExpectations DryRunTaskExpectation `yaml:"dryrun_expectations" json:"dryrun_expectations"`
}

// DefaultKubectlMockResponses returns default mock responses for common kubectl commands
func DefaultKubectlMockResponses() []MockResponse {
	return []MockResponse{
		{
			Pattern:  `kubectl get pods?`,
			Response: "NAME                    READY   STATUS    RESTARTS   AGE\nnginx-deployment-abc    1/1     Running   0          5m",
		},
		{
			Pattern:  `kubectl get deploy`,
			Response: "NAME               READY   UP-TO-DATE   AVAILABLE   AGE\nnginx-deployment   2/2     2            2           10m",
		},
		{
			Pattern:  `kubectl get svc|kubectl get services?`,
			Response: "NAME         TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)   AGE\nnginx-svc    ClusterIP   10.96.100.1    <none>        80/TCP    5m",
		},
		{
			Pattern:  `kubectl create namespace`,
			Response: "namespace/test-ns created",
		},
		{
			Pattern:  `kubectl apply`,
			Response: "resource applied",
		},
		{
			Pattern:  `kubectl delete`,
			Response: "resource deleted",
		},
		{
			Pattern:  `kubectl scale`,
			Response: "deployment scaled",
		},
		{
			Pattern:  `kubectl describe`,
			Response: "Name: resource\nNamespace: default\nStatus: Running",
		},
		{
			Pattern:  `kubectl logs`,
			Response: "Log output from container...",
		},
		{
			Pattern:  `kubectl exec`,
			Response: "Command executed in container",
		},
	}
}
