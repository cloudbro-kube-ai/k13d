package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

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
	aiText := ctx.app.aiPanel.GetText(false)
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
