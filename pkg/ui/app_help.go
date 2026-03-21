package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// showHelp displays help modal
func (a *App) showHelp() {
	helpText := fmt.Sprintf(`
%s
[gray]k9s compatible keybindings with AI assistance[white]

[cyan::b]GENERAL[white::-]
  [yellow]:[white]        Command mode        [yellow]?[white]        Help
  [yellow]/[white]        Filter mode         [yellow]Esc[white]      Back/Clear/Cancel
  [yellow]Tab[white]      AI prompt focus     [yellow]Shift+Tab[white] AI history focus
  [yellow]Ctrl+E[white]   Toggle AI panel     [yellow]Shift+O[white]  Settings/LLM Config
  [yellow]Alt+H/L[white] Resize AI panel     [yellow]Alt+F[white]    Full-size AI
  [yellow]Alt+0[white]   Reset AI width
  [yellow]q/Ctrl+C[white] Quit application

[cyan::b]AI ASSISTANT[white::-]
  [yellow]Enter[white]    Send prompt         [yellow]Up/Down[white]  Prompt history
  [yellow]j/k[white]      Scroll transcript   [yellow]PgUp/PgDn[white] Page transcript
  [yellow]g/G[white]      Transcript top/btm  [yellow]Tab[white]      Return to prompt

[cyan::b]NAVIGATION[white::-]
  [yellow]j/Down[white]   Down                [yellow]k/Up[white]     Up
  [yellow]g[white]        Top                 [yellow]G[white]        Bottom
  [yellow]Ctrl+F[white]   Page down           [yellow]Ctrl+B[white]   Page up
  [yellow]Ctrl+D[white]   Half page down      [yellow]Ctrl+U[white]   Half page up
  [yellow]Right[white]    Open / drill down   [yellow]Left/Esc[white] Back

[cyan::b]RESOURCE ACTIONS[white::-]
  [yellow]d[white]        Describe            [yellow]y[white]        YAML view
  [yellow]e[white]        Edit ($EDITOR)      [yellow]Ctrl+D[white]   Delete
  [yellow]r[white]        Refresh             [yellow]c[white]        Switch context
  [yellow]n[white]        Cycle namespace     [yellow]Space[white]    Multi-select

[cyan::b]SORTING[white::-]
  [yellow]Shift+N[white]  Sort by NAME        [yellow]Shift+A[white]  Sort by AGE
  [yellow]Shift+T[white]  Sort by STATUS      [yellow]Shift+P[white]  Sort by NAMESPACE
  [yellow]Shift+C[white]  Sort by RESTARTS    [yellow]Shift+D[white]  Sort by READY
  [yellow]:sort[white]    Sort column picker  [gray](toggle direction by sorting same column twice)[white]

[cyan::b]NAMESPACE SHORTCUTS[white::-] (k9s style)
  [yellow]0[white] All namespaces      [yellow]n[white]   Cycle through namespaces
  [yellow]1-9[white] Recent namespaces first
  [yellow]u[white] Use namespace (on namespace view)
  [yellow]:ns <name>[white]           Switch to specific namespace

[cyan::b]POD ACTIONS[white::-]
  [yellow]l[white]        Logs                [yellow]p[white]        Previous logs
  [yellow]s[white]        Shell               [yellow]a[white]        Attach
  [yellow]Enter[white]    Show containers     [yellow]o[white]        Show node
  [yellow]k/Ctrl+K[white] Kill (force delete) [yellow]Right[white]    Open containers
  [yellow]Shift+F[white]  Port forward        [yellow]f[white]        Show port-forward

[cyan::b]WORKLOAD ACTIONS[white::-] (Deploy/StatefulSet/DaemonSet/ReplicaSet)
  [yellow]S[white]        Scale               [yellow]R[white]        Restart/Rollout
  [yellow]z[white]        Show ReplicaSets    [yellow]Enter/Right[white] Open related

[cyan::b]VIEWER (Logs/Describe/YAML)[white::-] - Vim-style navigation
  [yellow]j/k[white]      Scroll down/up      [yellow]g/G[white]      Top/Bottom
  [yellow]Ctrl+D[white]   Half page down      [yellow]Ctrl+U[white]   Half page up
  [yellow]Ctrl+F[white]   Full page down      [yellow]Ctrl+B[white]   Full page up
  [yellow]/[white]        Search mode         [yellow]n/N[white]      Next/Prev match
  [yellow]q/Esc[white]    Close viewer

[cyan::b]COMMAND EXAMPLES[white::-] (press : to enter command mode)
  [yellow]:pods[white] [yellow]:po[white]              List pods
  [yellow]:pods -n kube-system[white]  List pods in specific namespace
  [yellow]:pods -A[white]              List pods in all namespaces
  [yellow]:deploy[white] [yellow]:dp[white]            List deployments
  [yellow]:svc[white] [yellow]:services[white]         List services
  [yellow]:ns kube-system[white]       Switch to namespace
  [yellow]:ctx[white] [yellow]:context[white]          Switch context

[cyan::b]AI ASSISTANT[white::-] (Tab to focus, type and press Enter)
  Ask natural language questions or request kubectl commands:
  - "Show me all pods in kube-system namespace"
  - "Why is my pod crashing?"
  - "Scale deployment nginx to 3 replicas"
  - "Show recent events for this deployment"
  - Press Enter on a selected table row while the AI panel is open to attach or detach that row as AI context

  [gray]Tool approvals open in a centered modal. Press Y/Enter to approve, N/Esc to cancel.[white]

[gray]Press Esc, q, or ? to close this help[white]
`, LogoColors())

	help := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetText(helpText)
	help.SetBorder(true).SetTitle(" Help ")

	a.showModal("help", centered(help, 75, 55), true)
	a.SetFocus(help)

	help.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc || event.Rune() == 'q' || event.Rune() == '?' {
			a.closeModal("help")
			a.SetFocus(a.table)
			return nil
		}
		return event
	})
}
