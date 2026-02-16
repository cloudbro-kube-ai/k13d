package server

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// DefaultTools returns the default k13d tools for the MCP server
func DefaultTools() []*Tool {
	return []*Tool{
		KubectlTool(),
		BashTool(),
		KubectlGetTool(),
		KubectlDescribeTool(),
		KubectlLogsTool(),
		KubectlApplyTool(),
	}
}

// KubectlTool returns the generic kubectl tool
func KubectlTool() *Tool {
	return &Tool{
		Name:        "kubectl",
		Description: "Execute any kubectl command to manage Kubernetes resources. Use this for get, describe, create, apply, delete, scale, logs, exec, and other Kubernetes operations.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "The kubectl command to execute (without 'kubectl' prefix). Examples: 'get pods -n default', 'describe deployment nginx', 'logs pod/nginx'",
				},
				"namespace": map[string]interface{}{
					"type":        "string",
					"description": "Optional namespace. If not specified, uses the namespace from the command or current context.",
				},
			},
			"required": []string{"command"},
		},
		Handler: kubectlHandler,
	}
}

// KubectlGetTool returns a specialized kubectl get tool
func KubectlGetTool() *Tool {
	return &Tool{
		Name:        "kubectl_get",
		Description: "Get Kubernetes resources. Returns a list of resources with their status.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"resource": map[string]interface{}{
					"type":        "string",
					"description": "Resource type to get (pods, deployments, services, nodes, etc.)",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Optional specific resource name",
				},
				"namespace": map[string]interface{}{
					"type":        "string",
					"description": "Namespace (use 'all' for all namespaces)",
				},
				"output": map[string]interface{}{
					"type":        "string",
					"description": "Output format: wide, yaml, json, name",
					"enum":        []string{"wide", "yaml", "json", "name", ""},
				},
				"selector": map[string]interface{}{
					"type":        "string",
					"description": "Label selector (e.g., 'app=nginx')",
				},
			},
			"required": []string{"resource"},
		},
		Handler: kubectlGetHandler,
	}
}

// KubectlDescribeTool returns a kubectl describe tool
func KubectlDescribeTool() *Tool {
	return &Tool{
		Name:        "kubectl_describe",
		Description: "Describe a Kubernetes resource in detail, including events and status.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"resource": map[string]interface{}{
					"type":        "string",
					"description": "Resource type (pod, deployment, service, node, etc.)",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Resource name",
				},
				"namespace": map[string]interface{}{
					"type":        "string",
					"description": "Namespace",
				},
			},
			"required": []string{"resource", "name"},
		},
		Handler: kubectlDescribeHandler,
	}
}

// KubectlLogsTool returns a kubectl logs tool
func KubectlLogsTool() *Tool {
	return &Tool{
		Name:        "kubectl_logs",
		Description: "Get logs from a pod or container.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pod": map[string]interface{}{
					"type":        "string",
					"description": "Pod name",
				},
				"namespace": map[string]interface{}{
					"type":        "string",
					"description": "Namespace",
				},
				"container": map[string]interface{}{
					"type":        "string",
					"description": "Container name (if pod has multiple containers)",
				},
				"tail": map[string]interface{}{
					"type":        "integer",
					"description": "Number of lines to show from the end (default: 100)",
				},
				"previous": map[string]interface{}{
					"type":        "boolean",
					"description": "Show logs from previous terminated container",
				},
			},
			"required": []string{"pod"},
		},
		Handler: kubectlLogsHandler,
	}
}

// KubectlApplyTool returns a kubectl apply tool
func KubectlApplyTool() *Tool {
	return &Tool{
		Name:        "kubectl_apply",
		Description: "Apply a Kubernetes manifest from YAML content.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"manifest": map[string]interface{}{
					"type":        "string",
					"description": "YAML manifest content to apply",
				},
				"namespace": map[string]interface{}{
					"type":        "string",
					"description": "Namespace to apply in",
				},
				"dry_run": map[string]interface{}{
					"type":        "boolean",
					"description": "If true, only validate without applying",
				},
			},
			"required": []string{"manifest"},
		},
		Handler: kubectlApplyHandler,
	}
}

// BashTool returns the bash tool
func BashTool() *Tool {
	return &Tool{
		Name:        "bash",
		Description: "Execute bash shell commands. Use for non-kubectl operations like file operations, curl, jq, helm, etc. Be cautious with destructive commands.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "The bash command to execute",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Timeout in seconds (default: 30, max: 300)",
				},
			},
			"required": []string{"command"},
		},
		Handler: bashHandler,
	}
}

// Handler implementations

func kubectlHandler(ctx context.Context, args map[string]interface{}) (string, error) {
	command, _ := args["command"].(string)
	namespace, _ := args["namespace"].(string)

	if command == "" {
		return "", fmt.Errorf("command is required")
	}

	// Strip "kubectl" prefix if present and parse into individual arguments
	cmdStr := command
	if strings.HasPrefix(cmdStr, "kubectl ") {
		cmdStr = strings.TrimPrefix(cmdStr, "kubectl ")
	} else if cmdStr == "kubectl" {
		cmdStr = ""
	}
	cmdStr = strings.TrimSpace(cmdStr)
	cmdArgs := strings.Fields(cmdStr)

	// Add namespace if specified and not already in command
	if namespace != "" {
		hasNamespace := false
		for _, arg := range cmdArgs {
			if arg == "-n" || arg == "--namespace" || strings.HasPrefix(arg, "-n=") || strings.HasPrefix(arg, "--namespace=") {
				hasNamespace = true
				break
			}
		}
		if !hasNamespace {
			cmdArgs = append([]string{"-n", namespace}, cmdArgs...)
		}
	}

	return runKubectlArgs(ctx, cmdArgs, 60*time.Second)
}

func kubectlGetHandler(ctx context.Context, args map[string]interface{}) (string, error) {
	resource, _ := args["resource"].(string)
	name, _ := args["name"].(string)
	namespace, _ := args["namespace"].(string)
	output, _ := args["output"].(string)
	selector, _ := args["selector"].(string)

	if resource == "" {
		return "", fmt.Errorf("resource is required")
	}

	cmdArgs := []string{"get", resource}
	if name != "" {
		cmdArgs = append(cmdArgs, name)
	}
	if namespace == "all" {
		cmdArgs = append(cmdArgs, "--all-namespaces")
	} else if namespace != "" {
		cmdArgs = append(cmdArgs, "-n", namespace)
	}
	if output != "" {
		cmdArgs = append(cmdArgs, "-o", output)
	}
	if selector != "" {
		cmdArgs = append(cmdArgs, "-l", selector)
	}

	return runKubectlArgs(ctx, cmdArgs, 30*time.Second)
}

func kubectlDescribeHandler(ctx context.Context, args map[string]interface{}) (string, error) {
	resource, _ := args["resource"].(string)
	name, _ := args["name"].(string)
	namespace, _ := args["namespace"].(string)

	if resource == "" || name == "" {
		return "", fmt.Errorf("resource and name are required")
	}

	cmdArgs := []string{"describe", resource, name}
	if namespace != "" {
		cmdArgs = append(cmdArgs, "-n", namespace)
	}

	return runKubectlArgs(ctx, cmdArgs, 30*time.Second)
}

func kubectlLogsHandler(ctx context.Context, args map[string]interface{}) (string, error) {
	pod, _ := args["pod"].(string)
	namespace, _ := args["namespace"].(string)
	container, _ := args["container"].(string)
	tail, _ := args["tail"].(float64) // JSON numbers are float64
	previous, _ := args["previous"].(bool)

	if pod == "" {
		return "", fmt.Errorf("pod is required")
	}

	cmdArgs := []string{"logs", pod}
	if namespace != "" {
		cmdArgs = append(cmdArgs, "-n", namespace)
	}
	if container != "" {
		cmdArgs = append(cmdArgs, "-c", container)
	}
	if tail > 0 {
		cmdArgs = append(cmdArgs, fmt.Sprintf("--tail=%d", int(tail)))
	} else {
		cmdArgs = append(cmdArgs, "--tail=100")
	}
	if previous {
		cmdArgs = append(cmdArgs, "--previous")
	}

	return runKubectlArgs(ctx, cmdArgs, 30*time.Second)
}

func kubectlApplyHandler(ctx context.Context, args map[string]interface{}) (string, error) {
	manifest, _ := args["manifest"].(string)
	namespace, _ := args["namespace"].(string)
	dryRun, _ := args["dry_run"].(bool)

	if manifest == "" {
		return "", fmt.Errorf("manifest is required")
	}

	cmdArgs := []string{"apply", "-f", "-"}
	if namespace != "" {
		cmdArgs = append(cmdArgs, "-n", namespace)
	}
	if dryRun {
		cmdArgs = append(cmdArgs, "--dry-run=client")
	}

	return runKubectlArgsWithInput(ctx, cmdArgs, manifest, 30*time.Second)
}

func bashHandler(ctx context.Context, args map[string]interface{}) (string, error) {
	command, _ := args["command"].(string)
	timeoutSec, _ := args["timeout"].(float64)

	if command == "" {
		return "", fmt.Errorf("command is required")
	}

	timeout := 30 * time.Second
	if timeoutSec > 0 {
		if timeoutSec > 300 {
			timeoutSec = 300
		}
		timeout = time.Duration(timeoutSec) * time.Second
	}

	return runCommand(ctx, command, timeout)
}

// runKubectlArgs executes kubectl with explicit arguments (no shell interpolation)
func runKubectlArgs(ctx context.Context, args []string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kubectl", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("command timed out after %v", timeout)
	}

	if err != nil && output == "" {
		return "", err
	}

	return output, nil
}

// runKubectlArgsWithInput executes kubectl with explicit arguments and stdin input
func runKubectlArgsWithInput(ctx context.Context, args []string, input string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Stdin = strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("command timed out after %v", timeout)
	}

	if err != nil && output == "" {
		return "", err
	}

	return output, nil
}

// runCommand executes a shell command
func runCommand(ctx context.Context, cmdStr string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", cmdStr)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("command timed out after %v", timeout)
	}

	if err != nil && output == "" {
		return "", err
	}

	return output, nil
}

// runCommandWithInput executes a command with stdin input
func runCommandWithInput(ctx context.Context, cmdStr, input string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", cmdStr)
	cmd.Stdin = strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("command timed out after %v", timeout)
	}

	if err != nil && output == "" {
		return "", err
	}

	return output, nil
}
