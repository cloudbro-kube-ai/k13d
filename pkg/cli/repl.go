package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai"
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

func (c *CLI) handleAI(args []string) {
	if c.aiClient == nil {
		fmt.Println("AI provider not configured. Set LLM settings in config.yaml.")
		return
	}

	question := strings.Join(args, " ")
	systemPrompt := fmt.Sprintf(
		"You are a Kubernetes assistant running in CLI mode. "+
			"The current namespace is '%s'. "+
			"Provide concise, helpful responses for Kubernetes tasks. "+
			"Keep responses to 2-3 paragraphs maximum unless asked for details.",
		c.namespace)

	fmt.Println()
	fmt.Println("--- AI Response ---")

	ctx := context.Background()
	fullPrompt := systemPrompt + "\n\nUser question: " + question

	var respBuilder strings.Builder
	err := c.aiClient.Ask(ctx, fullPrompt, func(chunk string) {
		fmt.Print(chunk)
		respBuilder.WriteString(chunk)
	})
	if err != nil {
		log.Debugf("Streaming AI call failed: %v, falling back to non-streaming", err)
		resp, err := c.aiClient.AskNonStreaming(ctx, fullPrompt)
		if err != nil {
			fmt.Printf("Error getting AI response: %v\n", err)
			fmt.Println("------------------")
			fmt.Println()
			return
		}
		fmt.Print(resp)
	}
	fmt.Println()
	fmt.Println("------------------")
	fmt.Println()
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
