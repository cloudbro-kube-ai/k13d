package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Logo ASCII art for k13d
const (
	// K13D ASCII Logo - Kubernetes AI Dashboard
	Logo = `
 ██╗  ██╗ ██╗██████╗ ██████╗
 ██║ ██╔╝███║╚════██╗██╔══██╗
 █████╔╝ ╚██║ █████╔╝██║  ██║
 ██╔═██╗  ██║ ╚═══██╗██║  ██║
 ██║  ██╗ ██║██████╔╝██████╔╝
 ╚═╝  ╚═╝ ╚═╝╚═════╝ ╚═════╝ `

	// Tagline
	Tagline = "Kubernetes AI Dashboard"
)

// Version is set at build time via main.go; defaults to dev.
var Version = "dev"

// LogoColors returns the logo with gradient colors
func LogoColors() string {
	lines := strings.Split(Logo, "\n")
	var result strings.Builder

	// Gradient from cyan to blue
	colors := []string{"[#00FFFF]", "[#00DDFF]", "[#00BBFF]", "[#0099FF]", "[#0077FF]", "[#0055FF]"}

	for i, line := range lines {
		if len(line) == 0 {
			continue
		}
		colorIdx := i % len(colors)
		result.WriteString(colors[colorIdx])
		result.WriteString(line)
		result.WriteString("[-]\n")
	}

	return result.String()
}

// SplashScreen creates a splash screen with the logo
type SplashScreen struct {
	*tview.Flex
	logo     *tview.TextView
	info     *tview.TextView
	progress *tview.TextView
}

// NewSplashScreen creates a new splash screen
func NewSplashScreen() *SplashScreen {
	// Logo view
	logo := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	logo.SetText(LogoColors())

	// Info view
	info := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	info.SetText(fmt.Sprintf("[yellow]%s[-]\n[gray]%s[-]", Tagline, Version))

	// Progress view
	progress := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	progress.SetText("[gray]Initializing...[-]")

	// Layout
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(logo, 8, 0, false).
		AddItem(info, 3, 0, false).
		AddItem(progress, 2, 0, false).
		AddItem(nil, 0, 1, false)

	return &SplashScreen{
		Flex:     flex,
		logo:     logo,
		info:     info,
		progress: progress,
	}
}

// SetProgress updates the progress message
func (s *SplashScreen) SetProgress(msg string) {
	s.progress.SetText(fmt.Sprintf("[gray]%s[-]", msg))
}

// SetError shows an error message
func (s *SplashScreen) SetError(msg string) {
	s.progress.SetText(fmt.Sprintf("[red]Error: %s[-]", msg))
}

// SetReady shows the ready state
func (s *SplashScreen) SetReady() {
	s.progress.SetText("[green]Ready! Press any key to continue...[-]")
}

// AnimatedLogo creates an animated logo effect.
// Pass a done channel (or nil) to allow early cancellation.
func AnimatedLogo(app *tview.Application, view *tview.TextView, duration time.Duration, done ...chan struct{}) {
	lines := strings.Split(Logo, "\n")
	totalLines := len(lines)
	if totalLines == 0 {
		return
	}

	var doneCh chan struct{}
	if len(done) > 0 {
		doneCh = done[0]
	}

	// Calculate delay per line
	lineDelay := duration / time.Duration(totalLines)

	// Animate line by line
	for i := 0; i <= totalLines; i++ {
		// Check for cancellation
		if doneCh != nil {
			select {
			case <-doneCh:
				return
			default:
			}
		}
		currentLines := i
		app.QueueUpdateDraw(func() {
			var result strings.Builder
			colors := []string{"[#00FFFF]", "[#00DDFF]", "[#00BBFF]", "[#0099FF]", "[#0077FF]", "[#0055FF]"}

			for j := 0; j < currentLines && j < len(lines); j++ {
				if len(lines[j]) == 0 {
					continue
				}
				colorIdx := j % len(colors)
				result.WriteString(colors[colorIdx])
				result.WriteString(lines[j])
				result.WriteString("[-]\n")
			}
			view.SetText(result.String())
		})
		if doneCh != nil {
			select {
			case <-doneCh:
				return
			case <-time.After(lineDelay):
			}
		} else {
			time.Sleep(lineDelay)
		}
	}
}

// HeaderLogo returns a compact logo for the header
func HeaderLogo() string {
	return "[#00FFFF::b]k[#00BBFF]1[#0077FF]3[#0055FF]d[-::-]"
}

// HeaderLogoWithContext returns header logo with context info
func HeaderLogoWithContext(cluster, namespace, resource string) string {
	var sb strings.Builder

	// Logo
	sb.WriteString(HeaderLogo())
	sb.WriteString(" ")

	// Separator
	sb.WriteString("[gray]│[-] ")

	// Cluster
	if cluster != "" {
		sb.WriteString("[yellow]")
		sb.WriteString(cluster)
		sb.WriteString("[-] ")
	}

	// Namespace
	if namespace != "" {
		sb.WriteString("[gray]»[-] [cyan]")
		sb.WriteString(namespace)
		sb.WriteString("[-] ")
	} else {
		sb.WriteString("[gray]»[-] [cyan]all[-] ")
	}

	// Resource
	if resource != "" {
		sb.WriteString("[gray]»[-] [green]")
		sb.WriteString(resource)
		sb.WriteString("[-]")
	}

	return sb.String()
}

// AboutModal creates an about modal with logo and info
func AboutModal() *tview.Flex {
	// Logo
	logo := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	logo.SetText(LogoColors())

	// Info
	info := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	info.SetText(fmt.Sprintf(`[yellow::b]%s[-::-]

[white]Version: [cyan]%s[-]

[gray]Kubernetes AI Dashboard CLI
Inspired by k9s with integrated AI assistance

[yellow]Features:[-]
[gray]• k9s-compatible keybindings
• AI-powered cluster analysis
• Natural language queries
• Tool-use with safety controls

[blue]https://github.com/cloudbro-kube-ai/k13d[-]

[darkgray]Press [yellow]Esc[-] or [yellow]q[-] to close[-]`, Tagline, Version))

	// Layout
	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(logo, 8, 0, false).
		AddItem(info, 18, 0, false).
		AddItem(nil, 0, 1, false)

	modal.SetBorder(true).
		SetTitle(" About k13d ").
		SetBorderColor(tcell.ColorDarkCyan)

	return modal
}

// ResourceIcon returns an icon for a resource type
func ResourceIcon(resource string) string {
	icons := map[string]string{
		"pods":         "🔲",
		"po":           "🔲",
		"deployments":  "📦",
		"deploy":       "📦",
		"services":     "🔌",
		"svc":          "🔌",
		"nodes":        "🖥️",
		"no":           "🖥️",
		"namespaces":   "📂",
		"ns":           "📂",
		"configmaps":   "📋",
		"cm":           "📋",
		"secrets":      "🔐",
		"sec":          "🔐",
		"ingresses":    "🌐",
		"ing":          "🌐",
		"events":       "📢",
		"ev":           "📢",
		"jobs":         "⚙️",
		"cronjobs":     "⏰",
		"cj":           "⏰",
		"statefulsets": "📊",
		"sts":          "📊",
		"daemonsets":   "👹",
		"ds":           "👹",
	}

	if icon, ok := icons[resource]; ok {
		return icon
	}
	return "📄"
}

// StatusColor returns a color for a status string
func StatusColor(status string) string {
	status = strings.ToLower(status)

	// Error states - check first for more specific matches (e.g., "notready" before "ready")
	if strings.Contains(status, "failed") ||
		strings.Contains(status, "error") ||
		strings.Contains(status, "crash") ||
		strings.Contains(status, "notready") ||
		strings.Contains(status, "backoff") ||
		strings.Contains(status, "evicted") ||
		strings.Contains(status, "oomkilled") {
		return "[red]"
	}

	// Warning states
	if strings.Contains(status, "pending") ||
		strings.Contains(status, "creating") ||
		strings.Contains(status, "warning") ||
		strings.Contains(status, "updating") ||
		strings.Contains(status, "terminating") ||
		strings.Contains(status, "unknown") {
		return "[yellow]"
	}

	// Success states
	if strings.Contains(status, "running") ||
		strings.Contains(status, "ready") ||
		strings.Contains(status, "active") ||
		strings.Contains(status, "succeeded") ||
		strings.Contains(status, "completed") ||
		strings.Contains(status, "normal") ||
		strings.Contains(status, "bound") {
		return "[green]"
	}

	return "[white]"
}

// FormatStatus returns a colored status string
func FormatStatus(status string) string {
	color := StatusColor(status)
	return fmt.Sprintf("%s%s[-]", color, status)
}

// ProgressBar returns a simple progress bar string
func ProgressBar(current, total int, width int) string {
	if total == 0 {
		return "[gray]" + strings.Repeat("░", width) + "[-]"
	}

	filled := (current * width) / total
	if filled > width {
		filled = width
	}

	var sb strings.Builder
	sb.WriteString("[green]")
	sb.WriteString(strings.Repeat("█", filled))
	sb.WriteString("[gray]")
	sb.WriteString(strings.Repeat("░", width-filled))
	sb.WriteString("[-]")

	return sb.String()
}

// SpinnerFrames for loading animation
var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner returns the current spinner frame
func Spinner(frame int) string {
	return SpinnerFrames[frame%len(SpinnerFrames)]
}

// ColoredSpinner returns a colored spinner
func ColoredSpinner(frame int, color string) string {
	return fmt.Sprintf("[%s]%s[-]", color, Spinner(frame))
}
