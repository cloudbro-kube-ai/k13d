package ui

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai/safety"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestBuildAIPromptIncludesDetailedSelectionContext(t *testing.T) {
	prompt := buildAIPrompt("Why is this pod failing?", aiPromptContext{
		Resource:          "pods",
		Namespace:         "default",
		SelectedResource:  "pods",
		SelectedName:      "api-7d9d8",
		SelectedNamespace: "default",
		SelectedSummary:   "NAME=api-7d9d8 | STATUS=CrashLoopBackOff | RESTARTS=5",
		DetailedContext:   "### Resource Manifest (YAML)\napiVersion: v1",
	})

	for _, want := range []string{
		"Current resource view: pods.",
		"Namespace scope: default.",
		"Selected row: NAME=api-7d9d8 | STATUS=CrashLoopBackOff | RESTARTS=5.",
		"Selected object: pods/api-7d9d8.",
		"Selected resource context:",
		"Why is this pod failing?",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q\n%s", want, prompt)
		}
	}
}

func TestBuildAIPromptHandlesNoSelection(t *testing.T) {
	prompt := buildAIPrompt("List risky workloads", aiPromptContext{
		Resource:  "deployments",
		Namespace: "",
	})

	if !strings.Contains(prompt, "Current resource view: deployments.") {
		t.Fatalf("prompt should include current view: %s", prompt)
	}
	if !strings.Contains(prompt, "Namespace scope: all namespaces.") {
		t.Fatalf("prompt should describe all-namespace scope: %s", prompt)
	}
	if strings.Contains(prompt, "Selected object:") {
		t.Fatalf("prompt should not mention a selected object when none exists: %s", prompt)
	}
}

func TestTrimAIBlockAddsTruncationNotice(t *testing.T) {
	got := trimAIBlock(strings.Repeat("abcdef", 10), 12)
	if !strings.Contains(got, "...[truncated]") {
		t.Fatalf("expected truncation notice, got %q", got)
	}
}

func TestSummarizeAIToolResultHandlesEmptyOutput(t *testing.T) {
	if got := summarizeAIToolResult("   "); got != "(no output)" {
		t.Fatalf("expected empty tool output marker, got %q", got)
	}
}

func TestAIInputHistoryRecallNavigatesAndDeduplicates(t *testing.T) {
	app := CreateMinimalTestApp()

	app.addAIInputHistory("first question")
	app.addAIInputHistory("second question")
	app.addAIInputHistory("second question")

	if len(app.aiInputHistory) != 2 {
		t.Fatalf("expected deduplicated history length 2, got %d", len(app.aiInputHistory))
	}

	if got := app.recallAIInputHistory(-1); got != "second question" {
		t.Fatalf("expected most recent history item, got %q", got)
	}
	if got := app.recallAIInputHistory(-1); got != "first question" {
		t.Fatalf("expected older history item, got %q", got)
	}
	if got := app.recallAIInputHistory(1); got != "second question" {
		t.Fatalf("expected forward history navigation, got %q", got)
	}
	if got := app.recallAIInputHistory(1); got != "" {
		t.Fatalf("expected history navigation to reset at newest item, got %q", got)
	}
}

func TestHandleAICommandHelpClearAndUnknown(t *testing.T) {
	app := NewTestApp(TestAppConfig{
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})

	if app.handleAICommand("why is this failing?") {
		t.Fatal("plain text input should not be treated as an AI slash command")
	}

	if !app.handleAICommand("/help") {
		t.Fatal("/help should be handled as an AI command")
	}
	if text := app.aiPanel.GetText(false); !strings.Contains(text, "AI Help") || !strings.Contains(text, "/context") {
		t.Fatalf("expected help output in AI panel, got:\n%s", text)
	}

	if !app.handleAICommand("/unknown") {
		t.Fatal("/unknown should still be consumed as an AI slash command")
	}
	if text := app.aiPanel.GetText(false); !strings.Contains(text, "Unknown Command") {
		t.Fatalf("expected unknown command notice, got:\n%s", text)
	}

	app.startAITurn("Will this be cleared?", aiPromptContext{Resource: "pods"}, "chat")
	if !app.handleAICommand("/clear") {
		t.Fatal("/clear should be handled as an AI command")
	}

	text := app.aiPanel.GetText(false)
	if !strings.Contains(text, "AI Assistant") || !strings.Contains(text, "Try:") {
		t.Fatalf("expected reset AI conversation text, got:\n%s", text)
	}
	if strings.Contains(text, "Will this be cleared?") {
		t.Fatalf("expected /clear to reset prior transcript, got:\n%s", text)
	}
}

func TestShowAIContextPreviewUsesSelectedRow(t *testing.T) {
	app := NewTestApp(TestAppConfig{
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})

	app.refresh()
	candidate := app.currentAISelectionCandidate()
	if candidate.IsZero() {
		t.Fatal("expected refresh to select a table row for AI context")
	}

	app.showAIContextPreview()
	text := app.aiPanel.GetText(false)

	for _, want := range []string{
		"Context Preview",
		"View: pods",
		"Namespace: default",
		"Selected row available: pods default/" + candidate.Name,
		"not attached yet",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("context preview missing %q\n%s", want, text)
		}
	}
}

func TestToggleSelectedAIContextAttachesSelectionAndHighlightsRow(t *testing.T) {
	app := NewTestApp(TestAppConfig{
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})

	app.refresh()
	candidate := app.currentAISelectionCandidate()
	if candidate.IsZero() {
		t.Fatal("expected a selected row to be available for AI attachment")
	}

	app.toggleSelectedAIContext()

	attached := app.getAttachedAIContext()
	if !attached.Matches(candidate) {
		t.Fatalf("expected AI attachment to match current row, got %+v", attached)
	}

	ctx := app.getAIPromptContext()
	if ctx.SelectedResource != candidate.Resource || ctx.SelectedName != candidate.Name {
		t.Fatalf("expected prompt context to use attached row, got %+v", ctx)
	}

	meta := app.aiMetaBar.GetText(false)
	if !strings.Contains(meta, "(attached)") || !strings.Contains(meta, candidate.Name) {
		t.Fatalf("expected AI meta bar to show attached context, got %q", meta)
	}

	cell := app.table.GetCell(1, 0)
	if cell == nil {
		t.Fatal("expected first data row cell to exist")
	}
	_, background, _ := cell.Style.Decompose()
	if background == tcell.ColorDefault {
		t.Fatalf("expected attached row to use subtle highlight, got default background")
	}

	app.toggleSelectedAIContext()
	if attached := app.getAttachedAIContext(); !attached.IsZero() {
		t.Fatalf("expected AI attachment to clear on second toggle, got %+v", attached)
	}
}

func TestApproveToolCallClearsStateWhenDelivered(t *testing.T) {
	app := CreateMinimalTestApp()
	app.setToolCallState("kubectl", `{"command":"get pods"}`, "kubectl get pods")

	if got := atomic.LoadInt32(&app.hasToolCall); got != 1 {
		t.Fatalf("expected pending tool call flag to be set, got %d", got)
	}

	app.approveToolCall(true)

	select {
	case approved := <-app.pendingToolApproval:
		if !approved {
			t.Fatal("expected tool approval to send true")
		}
	default:
		t.Fatal("expected approval signal to be delivered")
	}

	if got := atomic.LoadInt32(&app.hasToolCall); got != 0 {
		t.Fatalf("expected pending tool call flag to clear, got %d", got)
	}

	app.aiMx.RLock()
	info := app.currentToolCallInfo
	app.aiMx.RUnlock()
	if info.Name != "" || info.Args != "" || info.Command != "" {
		t.Fatalf("expected tool call info to be cleared, got %+v", info)
	}
}

func TestApproveToolCallKeepsStateWhenChannelIsFull(t *testing.T) {
	app := CreateMinimalTestApp()
	app.pendingToolApproval <- true
	app.setToolCallState("kubectl", `{"command":"delete pod nginx"}`, "kubectl delete pod nginx")

	app.approveToolCall(false)

	if got := atomic.LoadInt32(&app.hasToolCall); got != 1 {
		t.Fatalf("expected pending tool call flag to remain set when channel is full, got %d", got)
	}

	app.aiMx.RLock()
	info := app.currentToolCallInfo
	app.aiMx.RUnlock()
	if info.Command != "kubectl delete pod nginx" {
		t.Fatalf("expected tool call state to remain untouched, got %+v", info)
	}
}

func TestShowToolApprovalModalOpensAndStoresFocus(t *testing.T) {
	app := NewTestApp(TestAppConfig{
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})
	app.showAIPanel = true
	app.SetFocus(app.aiInput)

	app.showToolApprovalModal("kubectl", "kubectl scale deployment api --replicas=3", &safety.Decision{
		Category: "write",
		Warnings: []string{"This changes live cluster state."},
	})

	if !app.pages.HasPage(toolApprovalModalName) {
		t.Fatal("expected tool approval modal page to be present")
	}

	app.aiMx.RLock()
	focus := app.toolApprovalFocus
	app.aiMx.RUnlock()
	if focus != app.aiInput {
		t.Fatalf("expected tool approval modal to remember prior focus, got %T", focus)
	}
}

func TestApproveToolCallClosesModalAndRestoresFocus(t *testing.T) {
	app := NewTestApp(TestAppConfig{
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})
	app.showAIPanel = true
	app.SetFocus(app.aiInput)
	app.setToolCallState("kubectl", `{"command":"get pods"}`, "kubectl get pods")
	app.showToolApprovalModal("kubectl", "kubectl get pods", &safety.Decision{Category: "read-only"})

	app.approveToolCall(true)

	if app.pages.HasPage(toolApprovalModalName) {
		t.Fatal("expected tool approval modal to close after approval is delivered")
	}
	if got := app.GetFocus(); got != app.aiInput {
		t.Fatalf("expected focus to return to AI input, got %T", got)
	}
}

func TestApproveToolCallKeepsModalWhenChannelIsFull(t *testing.T) {
	app := NewTestApp(TestAppConfig{
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})
	app.showAIPanel = true
	app.pendingToolApproval <- true
	app.setToolCallState("kubectl", `{"command":"delete pod nginx"}`, "kubectl delete pod nginx")
	app.showToolApprovalModal("kubectl", "kubectl delete pod nginx", &safety.Decision{Category: "dangerous"})

	app.approveToolCall(false)

	if !app.pages.HasPage(toolApprovalModalName) {
		t.Fatal("expected tool approval modal to remain open when approval could not be delivered")
	}
}

func TestAdjustAIPanelWidthClampsAndResets(t *testing.T) {
	app := NewTestApp(TestAppConfig{
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})
	app.showAIPanel = true

	app.adjustAIPanelWidth(-10_000)
	if got := app.currentAIPanelWidth(); got != minAIPanelWidth {
		t.Fatalf("expected AI panel width to clamp to minimum %d, got %d", minAIPanelWidth, got)
	}

	app.adjustAIPanelWidth(10_000)
	if got := app.currentAIPanelWidth(); got != maxAIPanelWidth {
		t.Fatalf("expected AI panel width to clamp to maximum %d, got %d", maxAIPanelWidth, got)
	}

	app.resetAIPanelWidth()
	if got := app.currentAIPanelWidth(); got != defaultAIPanelWidth {
		t.Fatalf("expected AI panel width to reset to default %d, got %d", defaultAIPanelWidth, got)
	}
}

func TestAdjustAIPanelWidthPreservesAIPanelFocus(t *testing.T) {
	app := NewTestApp(TestAppConfig{
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})
	app.showAIPanel = true
	app.rebuildContentLayout(true)
	app.SetFocus(app.aiPanel)

	app.adjustAIPanelWidth(aiPanelResizeStep)

	if got := app.GetFocus(); got != app.aiPanel {
		t.Fatalf("expected AI panel focus to be preserved while resizing, got %T", got)
	}
}

func TestAIInputFrameProvidesPromptBoundary(t *testing.T) {
	app := NewTestApp(TestAppConfig{
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})

	if app.aiInputFrame == nil {
		t.Fatal("expected AI input frame to be initialized")
	}
	if title := app.aiInputFrame.GetTitle(); title != " Prompt " {
		t.Fatalf("expected AI input frame title %q, got %q", " Prompt ", title)
	}
	if got := app.aiInputFrame.GetItemCount(); got != 1 {
		t.Fatalf("expected AI input frame to wrap one input item, got %d", got)
	}
}

func TestAIFocusHelpersUpdateStatusHints(t *testing.T) {
	app := NewTestApp(TestAppConfig{
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})
	app.showAIPanel = true
	app.rebuildContentLayout(true)

	app.focusAITranscript()
	if got := app.GetFocus(); got != app.aiPanel {
		t.Fatalf("expected transcript focus, got %T", got)
	}
	if status := app.aiStatusBar.GetText(false); !strings.Contains(status, "History") || !strings.Contains(status, "Tab prompt") {
		t.Fatalf("expected transcript status hint, got %q", status)
	}

	app.focusAIInput()
	if got := app.GetFocus(); got != app.aiInput {
		t.Fatalf("expected input focus, got %T", got)
	}
	if status := app.aiStatusBar.GetText(false); !strings.Contains(status, "Enter send") || !strings.Contains(status, "Shift+Tab history") {
		t.Fatalf("expected input status hint, got %q", status)
	}
}

func TestTUIAIPanelToggleAndHelpCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TUI AI interaction test in short mode")
	}

	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	focusReady := make(chan struct{})
	ctx.app.QueueUpdate(func() {
		ctx.app.SetFocus(ctx.app.table)
		close(focusReady)
	})
	<-focusReady

	ctx.Press(tcell.KeyCtrlE).Wait(100 * time.Millisecond)

	ctx.app.mx.RLock()
	showingAI := ctx.app.showAIPanel
	ctx.app.mx.RUnlock()
	if !showingAI {
		t.Fatal("expected Ctrl+E to open the AI panel")
	}

	aiFocusReady := make(chan struct{})
	ctx.app.QueueUpdate(func() {
		ctx.app.SetFocus(ctx.app.aiInput)
		close(aiFocusReady)
	})
	<-aiFocusReady

	ctx.ExpectFocus(func(primitive tview.Primitive) bool {
		return primitive == ctx.app.aiInput
	})
	ctx.Submit("/help").Wait(150 * time.Millisecond)

	helpText := ctx.textViewText(ctx.app.aiPanel)
	if !strings.Contains(helpText, "AI Help") || !strings.Contains(helpText, "/context") {
		t.Fatalf("expected /help to render in AI panel, got:\n%s", helpText)
	}

	ctx.Press(tcell.KeyCtrlE).Wait(100 * time.Millisecond).ExpectNoFreeze()

	ctx.app.mx.RLock()
	showingAI = ctx.app.showAIPanel
	ctx.app.mx.RUnlock()
	if showingAI {
		t.Fatal("expected second Ctrl+E to hide the AI panel")
	}
}
