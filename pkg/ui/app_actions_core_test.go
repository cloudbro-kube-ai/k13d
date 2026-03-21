package ui

import (
	"sync/atomic"
	"testing"
)

func TestSafeSuspendRequestsSyncAfterReturn(t *testing.T) {
	screen := createTestScreen(t)
	app := NewTestApp(TestAppConfig{
		UseSimulationScreen:   true,
		Screen:                screen,
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})

	var ran atomic.Bool
	atomic.StoreInt32(&app.needsSync, 0)

	app.safeSuspend(func() {
		ran.Store(true)
	})

	if !ran.Load() {
		t.Fatal("expected suspended function to run")
	}
	if got := atomic.LoadInt32(&app.needsSync); got != 1 {
		t.Fatalf("expected safeSuspend to request a screen sync, got %d", got)
	}
}
