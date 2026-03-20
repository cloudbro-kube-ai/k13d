package ui

import (
	"context"
	"testing"
	"time"
)

func TestShowPodContainersOpensModal(t *testing.T) {
	app := NewTestApp(TestAppConfig{
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})

	app.refresh()
	app.table.Select(1, 0)

	row, _ := app.table.GetSelection()
	if row != 1 {
		t.Fatalf("expected first data row to be selected, got %d", row)
	}

	entries, err := app.listPodContainers(context.Background(), app.getTableCellText(1, 0), app.getTableCellText(1, 1))
	if err != nil {
		t.Fatalf("expected pod container listing to succeed, got %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one container entry for selected pod")
	}

	app.showPodContainers()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if hasTestPage(app, "pod-containers") {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	flashText := ""
	done := make(chan struct{})
	app.QueueUpdateDraw(func() {
		flashText = app.flash.GetText(false)
		close(done)
	})
	<-done

	t.Fatalf("expected pod container modal to open; flash=%q", flashText)
}

func hasTestPage(app *App, name string) bool {
	var has bool
	done := make(chan struct{})
	app.QueueUpdateDraw(func() {
		has = app.pages.HasPage(name)
		close(done)
	})
	<-done
	return has
}
