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

	cmdStr := command
	if !strings.HasPrefix(cmdStr, "kubectl") {
		cmdStr = "kubectl " + cmdStr
	}

	if namespace != "" && !strings.Contains(cmdStr, "-n ") && !strings.Contains(cmdStr, "--namespace") {
		cmdStr = strings.Replace(cmdStr, "kubectl ", fmt.Sprintf("kubectl -n %s ", namespace), 1)
	}

	return runCommand(ctx, cmdStr, 60*time.Second)
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

	cmd := "kubectl get " + resource
	if name != "" {
		cmd += " " + name
	}
	if namespace == "all" {
		cmd += " --all-namespaces"
	} else if namespace != "" {
		cmd += " -n " + namespace
	}
	if output != "" {
		cmd += " -o " + output
	}
	if selector != "" {
		cmd += " -l " + selector
	}

	return runCommand(ctx, cmd, 30*time.Second)
}

func kubectlDescribeHandler(ctx context.Context, args map[string]interface{}) (string, error) {
	resource, _ := args["resource"].(string)
	name, _ := args["name"].(string)
	namespace, _ := args["namespace"].(string)

	if resource == "" || name == "" {
		return "", fmt.Errorf("resource and name are required")
	}

	cmd := fmt.Sprintf("kubectl describe %s %s", resource, name)
	if namespace != "" {
		cmd += " -n " + namespace
	}

	return runCommand(ctx, cmd, 30*time.Second)
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

	cmd := "kubectl logs " + pod
	if namespace != "" {
		cmd += " -n " + namespace
	}
	if container != "" {
		cmd += " -c " + container
	}
	if tail > 0 {
		cmd += fmt.Sprintf(" --tail=%d", int(tail))
	} else {
		cmd += " --tail=100"
	}
	if previous {
		cmd += " --previous"
	}

	return runCommand(ctx, cmd, 30*time.Second)
}

func kubectlApplyHandler(ctx context.Context, args map[string]interface{}) (string, error) {
	manifest, _ := args["manifest"].(string)
	namespace, _ := args["namespace"].(string)
	dryRun, _ := args["dry_run"].(bool)

	if manifest == "" {
		return "", fmt.Errorf("manifest is required")
	}

	cmd := "kubectl apply -f -"
	if namespace != "" {
		cmd += " -n " + namespace
	}
	if dryRun {
		cmd += " --dry-run=client"
	}

	return runCommandWithInput(ctx, cmd, manifest, 30*time.Second)
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
