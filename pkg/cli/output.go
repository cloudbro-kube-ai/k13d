package cli

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

const (
	// maxLinesBeforePager is the threshold before output is paginated.
	maxLinesBeforePager = 20

	// pagerPrompt is shown at the bottom when paginating.
	pagerPrompt = "-- more (Space=next, Q=quit) --"
)

// PrintOutput prints command output to stdout. If the output exceeds the
// terminal height, it paginates with the pager prompt.
func PrintOutput(out string) {
	if out == "" {
		return
	}

	lines := strings.Split(out, "\n")
	_, termHeight := DetectTermSize()
	// Reserve 2 lines for prompt + blank
	available := termHeight - 2
	if available < 5 {
		available = 5
	}

	if len(lines) <= maxLinesBeforePager || len(lines) <= available {
		// Print directly
		fmt.Print(out)
		if !strings.HasSuffix(out, "\n") {
			fmt.Println()
		}
		return
	}

	pageOutput(lines, available)
}

// pageOutput shows output page by page, waiting for user input.
func pageOutput(lines []string, pageSize int) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		// Non-interactive: print all
		for _, line := range lines {
			fmt.Println(line)
		}
		return
	}

	// Switch terminal to raw mode for single-key reading
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		// Fallback: print all
		for _, line := range lines {
			fmt.Println(line)
		}
		return
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	total := len(lines)
	start := 0
	buf := make([]byte, 3)

	for start < total {
		end := start + pageSize
		if end > total {
			end = total
		}

		// Print page content
		for _, line := range lines[start:end] {
			fmt.Print(line, "\r\n")
		}

		if end >= total {
			break
		}

		// Print pager prompt
		fmt.Print(pagerPrompt, "\r\n")

		// Wait for key press
	waitKey:
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return
		}
		if n == 0 {
			return
		}

		key := buf[0]
		switch key {
		case ' ':
			// Space: next page
			start = end
		case 'q', 'Q', 0x1b: // q, Q, or Escape
			// Quit paging
			start = total
		case 0x03: // Ctrl+C
			start = total
		default:
			goto waitKey
		}
	}

	// Restore to cooked mode for the prompt
	_ = term.Restore(int(os.Stdin.Fd()), oldState)
}

// PrintError prints an error message in a formatted way.
func PrintError(msg string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
}

// PrintInfo prints an informational message.
func PrintInfo(msg string) {
	fmt.Println(msg)
}

// PrintHelp prints the help screen.
func PrintHelp() {
	help := `
Built-in Commands:
  :help                   Show this help message
  :quit, :exit            Exit CLI mode
  :clear                  Clear screen and show splash
  :version                Show k13d version information
  :namespace <name>       Set default namespace
  :context <name>         Switch Kubernetes context
  :history                Show command history
  :ai <question>          Ask AI about your cluster
  :model [name]           Show current model or switch profile
  :mcp [list|tools|status] Manage MCP servers
Any other input is executed as a kubectl command.

Examples:
  get pods
  get pods -n kube-system
  get deployments
  describe pod nginx-xxx
  logs pod/nginx-xxx
  get nodes

Navigation:
  Up/Down arrows    Navigate command history
  Ctrl+C            Cancel / exit
  Tab               Auto-complete commands
`
	fmt.Println(help)
}
