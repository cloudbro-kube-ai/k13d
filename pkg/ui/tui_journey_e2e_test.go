package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai/safety"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func aiPanelWidthForTest(t *testing.T, app *App) int {
	t.Helper()

	done := make(chan struct{})
	width := 0
	app.QueueUpdate(func() {
		_, _, width, _ = app.aiContainer.GetRect()
		close(done)
	})
	<-done

	return width
}

func focusTUITable(t *testing.T, ctx *TUITestContext) {
	t.Helper()

	done := make(chan struct{})
	ctx.app.QueueUpdate(func() {
		ctx.app.SetFocus(ctx.app.table)
		close(done)
	})
	<-done
}

func TestTUIJourney_OperatorWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TUI journey test in short mode")
	}

	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()
	focusTUITable(t, ctx)

	ctx.Command("deploy").
		ExpectResource("deployments").
		ExpectNoFreeze()

	ctx.PressRune('/').
		Type("nginx").
		Press(tcell.KeyEnter).
		Wait(100 * time.Millisecond).
		ExpectFilter("nginx").
		ExpectNoFreeze()

	ctx.PressRune('?').
		ExpectPage("help")
	ctx.PressRune('?').
		Wait(100 * time.Millisecond).
		ExpectNoFreeze()
	focusTUITable(t, ctx)

	ctx.PressRune('N').
		Wait(100 * time.Millisecond).
		ExpectNoFreeze()
	focusTUITable(t, ctx)

	ctx.PressRune('I').
		ExpectPage("about").
		Escape().
		Wait(100 * time.Millisecond).
		ExpectNoFreeze()
	focusTUITable(t, ctx)

	ctx.Press(tcell.KeyCtrlE).Wait(100 * time.Millisecond)
	ctx.app.mx.RLock()
	showingAI := ctx.app.showAIPanel
	ctx.app.mx.RUnlock()
	if !showingAI {
		t.Fatal("expected Ctrl+E to open the AI panel")
	}

	focusTUITable(t, ctx)
	ctx.Tab().ExpectFocus(func(primitive tview.Primitive) bool {
		return primitive == ctx.app.aiInput
	})

	ctx.Submit("/context").Wait(150 * time.Millisecond)
	aiText := ctx.textViewText(ctx.app.aiPanel)
	for _, want := range []string{"Context Preview", "View: deployments", "Namespace: default"} {
		if !strings.Contains(aiText, want) {
			t.Fatalf("AI panel context preview missing %q\n%s", want, aiText)
		}
	}

	ctx.Press(tcell.KeyEsc).Wait(100 * time.Millisecond).ExpectNoFreeze()
	focusTUITable(t, ctx)

	ctx.PressRune('0').
		Wait(150 * time.Millisecond).
		ExpectNamespace("").
		ExpectNoFreeze()
}

func TestTUIJourney_AITranscriptFocusAndPromptReturn(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	focusTUITable(t, ctx)

	ctx.Press(tcell.KeyCtrlE).Wait(100 * time.Millisecond)
	ctx.app.mx.RLock()
	showingAI := ctx.app.showAIPanel
	ctx.app.mx.RUnlock()
	if !showingAI {
		t.Fatal("expected Ctrl+E to open the AI panel")
	}

	ctx.ShiftTab().
		Wait(100 * time.Millisecond).
		ExpectFocus(func(primitive tview.Primitive) bool {
			return primitive == ctx.app.aiPanel
		})

	status := ctx.textViewText(ctx.app.aiStatusBar)
	if !strings.Contains(status, "History") || !strings.Contains(status, "Tab prompt") {
		t.Fatalf("expected transcript focus status, got %q", status)
	}

	ctx.Tab().
		Wait(100 * time.Millisecond).
		ExpectFocus(func(primitive tview.Primitive) bool {
			return primitive == ctx.app.aiInput
		})

	status = ctx.textViewText(ctx.app.aiStatusBar)
	if !strings.Contains(status, "Enter send") || !strings.Contains(status, "Shift+Tab history") {
		t.Fatalf("expected prompt focus status, got %q", status)
	}

	frameTitle := ""
	itemCount := 0
	done := make(chan struct{})
	ctx.app.QueueUpdate(func() {
		frameTitle = ctx.app.aiInputFrame.GetTitle()
		itemCount = ctx.app.aiInputFrame.GetItemCount()
		close(done)
	})
	<-done
	if frameTitle != " Prompt " || itemCount != 1 {
		t.Fatalf("expected prompt frame boundary, title=%q items=%d", frameTitle, itemCount)
	}
}

func TestTUIJourney_CommandHistoryNamespaceHintAndSelection(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	ctx.Command("pods").
		Wait(50 * time.Millisecond).
		Command("services").
		Wait(50 * time.Millisecond).
		Command("namespaces").
		Wait(100 * time.Millisecond).
		ExpectResource("namespaces")

	ctx.PressRune(':').Wait(50 * time.Millisecond)
	ctx.PressRune('1').Wait(100 * time.Millisecond)
	ctx.ExpectContentContains("default")
	ctx.Press(tcell.KeyEsc).Wait(50 * time.Millisecond)

	rowBefore, _ := safeGetTableSelection(ctx.app)
	ctx.PressRune(' ')
	ctx.Wait(80 * time.Millisecond).ExpectNoFreeze()

	ctx.app.mx.RLock()
	selected := len(ctx.app.selectedRows)
	ctx.app.mx.RUnlock()
	if selected == 0 {
		t.Fatal("expected space to toggle a table row selection")
	}

	ctx.PressRune('j').Wait(50 * time.Millisecond)
	rowAfter, _ := safeGetTableSelection(ctx.app)
	if rowAfter <= rowBefore {
		t.Fatalf("expected j to move selection down, before=%d after=%d", rowBefore, rowAfter)
	}

	ctx.PressRune(':').Wait(50 * time.Millisecond)
	ctx.Press(tcell.KeyUp).Wait(50 * time.Millisecond)
	ctx.ExpectContentContains("namespaces")
	ctx.Press(tcell.KeyEsc).Wait(50 * time.Millisecond)
}

func TestTUIJourney_ToolApprovalModal(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()

	done := make(chan struct{})
	ctx.app.QueueUpdate(func() {
		ctx.app.mx.Lock()
		ctx.app.showAIPanel = true
		ctx.app.mx.Unlock()
		ctx.app.rebuildContentLayout(true)
		ctx.app.SetFocus(ctx.app.aiInput)
		ctx.app.setToolCallState("kubectl", `{"command":"scale deployment nginx --replicas=3"}`, "kubectl scale deployment nginx --replicas=3")
		ctx.app.showToolApprovalModal("kubectl", "kubectl scale deployment nginx --replicas=3", &safety.Decision{
			Category: "write",
			Warnings: []string{"This changes live cluster state."},
		})
		close(done)
	})
	<-done

	ctx.ExpectPage(toolApprovalModalName).ExpectNoFreeze()

	ctx.PressRune('n').Wait(100 * time.Millisecond)

	select {
	case approved := <-ctx.app.pendingToolApproval:
		if approved {
			t.Fatal("expected modal rejection to send false approval")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected rejection approval signal to be delivered")
	}

	if safeHasPage(ctx.app, toolApprovalModalName) {
		t.Fatal("expected tool approval modal to close after rejection")
	}

	ctx.ExpectFocus(func(primitive tview.Primitive) bool {
		return primitive == ctx.app.aiInput
	})

	done = make(chan struct{})
	ctx.app.QueueUpdate(func() {
		ctx.app.SetFocus(ctx.app.aiInput)
		ctx.app.setToolCallState("kubectl", `{"command":"scale deployment nginx --replicas=5"}`, "kubectl scale deployment nginx --replicas=5")
		ctx.app.showToolApprovalModal("kubectl", "kubectl scale deployment nginx --replicas=5", &safety.Decision{
			Category: "write",
			Warnings: []string{"This changes live cluster state."},
		})
		close(done)
	})
	<-done

	ctx.ExpectPage(toolApprovalModalName).ExpectNoFreeze()

	ctx.Press(tcell.KeyEnter).Wait(100 * time.Millisecond)

	select {
	case approved := <-ctx.app.pendingToolApproval:
		if !approved {
			t.Fatal("expected modal approval to send true approval")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected approval signal to be delivered")
	}

	if safeHasPage(ctx.app, toolApprovalModalName) {
		t.Fatal("expected tool approval modal to close after approval")
	}

	ctx.ExpectFocus(func(primitive tview.Primitive) bool {
		return primitive == ctx.app.aiInput
	})
}

func TestTUIJourney_AIPanelResize(t *testing.T) {
	ctx := NewTUITestContext(t)
	defer ctx.Cleanup()
	focusTUITable(t, ctx)

	ctx.Press(tcell.KeyCtrlE).Wait(100 * time.Millisecond).ExpectNoFreeze()

	initialStateWidth := ctx.app.currentAIPanelWidth()
	initialRectWidth := aiPanelWidthForTest(t, ctx.app)
	if initialRectWidth <= 0 {
		t.Fatalf("expected AI panel to have a visible width, got %d", initialRectWidth)
	}

	ctx.PressAlt('l').Wait(100 * time.Millisecond).ExpectNoFreeze()
	widerStateWidth := ctx.app.currentAIPanelWidth()
	widerRectWidth := aiPanelWidthForTest(t, ctx.app)
	if widerStateWidth <= initialStateWidth {
		t.Fatalf("expected Alt+L to increase AI panel width, before=%d after=%d", initialStateWidth, widerStateWidth)
	}
	if widerRectWidth <= initialRectWidth {
		t.Fatalf("expected AI panel rect width to increase, before=%d after=%d", initialRectWidth, widerRectWidth)
	}

	ctx.PressAlt('h').Wait(100 * time.Millisecond).ExpectNoFreeze()
	shrunkStateWidth := ctx.app.currentAIPanelWidth()
	shrunkRectWidth := aiPanelWidthForTest(t, ctx.app)
	if shrunkStateWidth >= widerStateWidth {
		t.Fatalf("expected Alt+H to decrease AI panel width, before=%d after=%d", widerStateWidth, shrunkStateWidth)
	}
	if shrunkRectWidth >= widerRectWidth {
		t.Fatalf("expected AI panel rect width to decrease, before=%d after=%d", widerRectWidth, shrunkRectWidth)
	}

	ctx.PressAlt('0').Wait(100 * time.Millisecond).ExpectNoFreeze()
	resetStateWidth := ctx.app.currentAIPanelWidth()
	resetRectWidth := aiPanelWidthForTest(t, ctx.app)
	if resetStateWidth != defaultAIPanelWidth {
		t.Fatalf("expected Alt+0 to reset AI panel width to %d, got %d", defaultAIPanelWidth, resetStateWidth)
	}
	if resetRectWidth != initialRectWidth {
		t.Fatalf("expected AI panel rect width to reset to %d, got %d", initialRectWidth, resetRectWidth)
	}
}
