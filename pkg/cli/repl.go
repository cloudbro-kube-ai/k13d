package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai"
	"github.com/cloudbro-kube-ai/k13d/pkg/ai/safety"
	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
	"github.com/cloudbro-kube-ai/k13d/pkg/log"
)

type CLI struct {
	cfg       *config.Config
	client    *k8s.Client
	aiClient  *ai.Client
	namespace string
	history   *CommandHistory
	version   VersionInfo
	running   bool
}

func New(cfg *config.Config, ver VersionInfo) *CLI {
	return &CLI{
		cfg:       cfg,
		namespace: "default",
		history:   NewCommandHistory(),
		version:   ver,
	}
}

func (c *CLI) Start() error {
	var err error
	c.client, err = k8s.NewClient()
	if err != nil {
		log.Warnf("Failed to create Kubernetes client: %v", err)
	}

	if c.namespace == "" {
		c.namespace = "default"
	}

	if c.cfg.LLM.Provider != "" {
		ac, err := ai.NewClient(&c.cfg.LLM)
		if err == nil {
			c.aiClient = ac
		}
	}
	PrintSplash(c.version.Version)
	fmt.Println()
	fmt.Print("\033[38;5;240m:  키를 입력하고 엔터를 누르면 도움말을 볼 수 있습니다.\033[0m\n\n")
	c.running = true
	for c.running {
		input, err := c.readLine()
		if err != nil {
			if err.Error() == "interrupt" || err == io.EOF {
				fmt.Println()
				break
			}
			log.Errorf("Read error: %v", err)
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		c.history.Add(input)
		c.dispatch(input)
	}
	return nil
}

func (c *CLI) Stop() {
	c.running = false
}

func (c *CLI) dispatch(input string) {
	if strings.HasPrefix(input, ":") {
		c.handleBuiltin(input[1:])
		return
	}
	c.executeKubectl(input)
}

func (c *CLI) handleBuiltin(cmdLine string) {
	cmdLine = strings.TrimSpace(cmdLine)
	parts := strings.Fields(cmdLine)
	if len(parts) == 0 {
		PrintHelp()
		return
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "help":
		PrintHelp()
	case "quit", "exit":
		fmt.Println("Exiting k13d CLI.")
		c.Stop()
	case "clear":
		PrintSplash(c.version.Version)
	case "version":
		fmt.Printf("k13d version %s\n", c.version.Version)
		fmt.Printf("  Build time: %s\n", c.version.BuildTime)
		fmt.Printf("  Git commit: %s\n", c.version.GitCommit)
	case "namespace":
		c.handleNamespace(args)
	case "context":
		c.handleContext(args)
	case "history":
		c.printHistory()
	case "ai":
		if len(args) == 0 {
			fmt.Println("Usage: :ai <question>")
			break
		}
		c.handleAI(args)
	default:
		fmt.Printf("Unknown command: :%s\n", cmd)
		fmt.Println("Type :help for available commands.")
	}
}

func (c *CLI) handleNamespace(args []string) {
	if len(args) == 0 {
		fmt.Printf("Current namespace: %s\n", c.namespace)
		return
	}
	c.namespace = args[0]
	fmt.Printf("Namespace set to: %s\n", c.namespace)
}

func (c *CLI) handleContext(args []string) {
	if c.client == nil {
		fmt.Println("Kubernetes client not available")
		return
	}
	if len(args) == 0 {
		cur, err := c.client.GetCurrentContext()
		if err != nil {
			fmt.Printf("Error getting context: %v\n", err)
			return
		}
		fmt.Printf("Current context: %s\n", cur)
		return
	}
	err := c.client.SwitchContext(args[0])
	if err != nil {
		fmt.Printf("Error switching context: %v\n", err)
		return
	}
	fmt.Printf("Context switched to: %s\n", args[0])
}

func (c *CLI) printHistory() {
	entries := c.history.Entries()
	if len(entries) == 0 {
		fmt.Println("No command history.")
		return
	}
	fmt.Println("Command history:")
	for i, entry := range entries {
		fmt.Printf("  %3d  %s\n", i+1, entry)
	}
}

func (c *CLI) gatherClusterContext() string {
	if c.client == nil {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var b strings.Builder
	b.WriteString("Current cluster state:\n")

	// Pods in namespace
	pods, err := c.client.ListPods(ctx, c.namespace)
	if err == nil {
		running, pending, failedCount := 0, 0, 0
		for _, p := range pods {
			switch string(p.Status.Phase) {
			case "Running":
				running++
			case "Pending":
				pending++
			case "Failed":
				failedCount++
			}
		}
		b.WriteString(fmt.Sprintf("- Pods in '%s': %d total (%d running, %d pending, %d failed)\n",
			c.namespace, len(pods), running, pending, failedCount))
	}

	// Services
	svcs, err := c.client.ListServices(ctx, c.namespace)
	if err == nil {
		b.WriteString(fmt.Sprintf("- Services in '%s': %d\n", c.namespace, len(svcs)))
	}

	// Deployments
	deps, err := c.client.ListDeployments(ctx, c.namespace)
	if err == nil {
		b.WriteString(fmt.Sprintf("- Deployments in '%s': %d\n", c.namespace, len(deps)))
	}

	// Recent warning events (last 3)
	events, err := c.client.ListEvents(ctx, c.namespace)
	if err == nil {
		count := 0
		for _, ev := range events {
			if ev.Type == "Warning" {
				count++
			}
		}
		if count > 0 {
			b.WriteString(fmt.Sprintf("- Recent warnings in '%s': %d total\n", c.namespace, count))
			shown := 0
			for i := len(events) - 1; i >= 0 && shown < 3; i-- {
				if events[i].Type == "Warning" {
					b.WriteString(fmt.Sprintf("  - %s: %s\n", events[i].Reason, events[i].Message))
					shown++
				}
			}
		}
	}

	result := b.String()
	if result == "Current cluster state:\n" {
		return ""
	}
	return result
}

// parseAIArgs extracts resource/name pattern from the first arg
// e.g. "pod/nginx" -> resource="pods", name="nginx"
func parseAIArgs(args []string) (resource, name, question string) {
	if len(args) == 0 {
		return "", "", ""
	}
	first := args[0]
	if parts := strings.SplitN(first, "/", 2); len(parts) == 2 {
		res := strings.ToLower(strings.TrimSpace(parts[0]))
		nm := strings.TrimSpace(parts[1])
		if res != "" && nm != "" {
			return res, nm, strings.Join(args[1:], " ")
		}
	}
	return "", "", strings.Join(args, " ")
}

// getResourceDetailedContext fetches YAML + events + logs for a specific resource
func (c *CLI) getResourceDetailedContext(resource, name string) string {
	if c.client == nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	detailedCtx, err := c.client.GetResourceContext(ctx, c.namespace, name, resource)
	if err != nil {
		log.Debugf("Failed to get resource context: %v", err)
		return ""
	}
	return detailedCtx
}

func (c *CLI) handleAI(args []string) {
	if c.aiClient == nil {
		fmt.Println("AI provider not configured. Set LLM settings in config.yaml.")
		return
	}

	// Parse resource/name pattern (e.g. "pod/nginx why failing?")
	resource, name, question := parseAIArgs(args)
	if question == "" {
		fmt.Println("Usage: :ai [resource/name] <question>")
		return
	}

	fmt.Println()

	// Step 1: Resource-specific detailed context (if resource/name specified)
	detailedCtx := ""
	if resource != "" && name != "" {
		fmt.Printf("[cyan]Gathering context for %s/%s...[-]\n", resource, name)
		detailedCtx = c.getResourceDetailedContext(resource, name)
	}

	// Step 2: Cluster-level context
	fmt.Print("[cyan]Gathering cluster context...[-]\n")
	clusterCtx := c.gatherClusterContext()

	// Build the context block for the system prompt
	var ctxBlock strings.Builder
	ctxBlock.WriteString(fmt.Sprintf("Current namespace: %s.\n", c.namespace))
	if clusterCtx != "" {
		ctxBlock.WriteString("\n### Cluster State\n")
		ctxBlock.WriteString(clusterCtx)
	}
	if detailedCtx != "" {
		ctxBlock.WriteString("\n### Selected Resource Details\n")
		ctxBlock.WriteString(detailedCtx)
	}

	supportsTools := c.aiClient.SupportsTools()
	toolNote := ""
	if supportsTools {
		toolNote = " You have access to kubectl tool - use it to query additional information if needed."
	}

	systemPrompt := fmt.Sprintf(
		"You are a Kubernetes assistant running in CLI mode. "+
			"%s"+
			"Provide concise, evidence-based answers. "+
			"Keep responses to 2-3 paragraphs unless asked for details."+"%s",
		ctxBlock.String(), toolNote)

	fmt.Print("[cyan]Sending to AI...[-]\n\n")
	fmt.Println("--- AI Response ---")

	var respBuilder strings.Builder
	ctx := context.Background()
	fullPrompt := systemPrompt + "\n\nUser question: " + question

	if supportsTools {
		log.Debugf("Using tool-supported AI call")
		err := c.aiClient.AskWithToolsAndExecution(ctx, fullPrompt,
			func(chunk string) {
				fmt.Print(chunk)
				respBuilder.WriteString(chunk)
			},
			c.toolApprovalCallback,
			c.toolExecutionCallback,
		)
		if err != nil {
			log.Debugf("Streaming tool AI call failed: %v, falling back to non-streaming", err)
			resp, e := c.aiClient.AskNonStreaming(ctx, fullPrompt)
			if e != nil {
				fmt.Printf("Error getting AI response: %v\n", e)
				fmt.Println("------------------")
				fmt.Println()
				return
			}
			fmt.Print(resp)
		}
	} else {
		err := c.aiClient.Ask(ctx, fullPrompt, func(chunk string) {
			fmt.Print(chunk)
			respBuilder.WriteString(chunk)
		})
		if err != nil {
			log.Debugf("Streaming AI call failed: %v, falling back to non-streaming", err)
			resp, e := c.aiClient.AskNonStreaming(ctx, fullPrompt)
			if e != nil {
				fmt.Printf("Error getting AI response: %v\n", e)
				fmt.Println("------------------")
				fmt.Println()
				return
			}
			fmt.Print(resp)
		}
	}

	fmt.Println()
	fmt.Println("------------------")
	fmt.Println()
}

// toolApprovalCallback handles AI tool execution approval via stdin
func (c *CLI) toolApprovalCallback(toolName string, argsJSON string) bool {
	var args struct {
		Command   string `json:"command"`
		Namespace string `json:"namespace,omitempty"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return false
	}

	command := args.Command
	if toolName == "kubectl" && !strings.HasPrefix(command, "kubectl ") {
		command = "kubectl " + command
	}

	enforcer := safety.NewDefaultPolicyEnforcer()
	decision := enforcer.Evaluate(command)

	// Auto-approve if policy says no approval needed
	if decision.Allowed && !decision.RequiresApproval {
		return true
	}

	// Blocked by policy
	if !decision.Allowed {
		fmt.Printf("\n[red]Command blocked: %s[-]\n", decision.BlockReason)
		return false
	}

	// Requires approval - prompt user
	fmt.Printf("\n[yellow]AI wants to execute:[-] [%s] %s\n", toolName, command)
	for _, w := range decision.Warnings {
		fmt.Printf("   Warning: %s\n", w)
	}
	fmt.Printf("   [Category: %s]\n", decision.Category)
	fmt.Print("[green]Approve?[-] (Y/n/q): ")

	var response string
	fmt.Scanln(&response)
	response = strings.TrimSpace(strings.ToLower(response))

	switch response {
	case "", "y", "yes":
		return true
	case "q", "quit":
		return false
	default:
		return false
	}
}

// toolExecutionCallback reports tool execution results
func (c *CLI) toolExecutionCallback(toolName string, command string, result string, isError bool, toolType string, toolServerName string) {
	if isError {
		log.Debugf("Tool execution error - %s/%s: %s", toolName, command, result)
	} else {
		log.Debugf("Tool executed - %s/%s: %d bytes", toolName, command, len(result))
	}
}

func (c *CLI) executeKubectl(input string) {
	kubectlInput := input
	hasNamespace := strings.Contains(input, "--namespace") ||
		strings.Contains(input, "-n ")
	if !hasNamespace && c.namespace != "" && c.namespace != "default" {
		kubectlInput = input + " --namespace " + c.namespace
	}

	output, err := runKubectlCommand(kubectlInput)
	if err != nil {
		PrintError(err.Error())
		return
	}
	PrintOutput(output)
}
