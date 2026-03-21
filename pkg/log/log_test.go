package log

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

// setupTestLogger replaces the package-level logger with one that writes to buf.
func setupTestLogger(t *testing.T, buf *bytes.Buffer) {
	t.Helper()
	logger = log.New(buf, "", 0) // no flags for predictable output
	currentLevel = LevelDebug    // capture everything
}

func TestSetLevel(t *testing.T) {
	tests := []struct {
		input string
		want  Level
	}{
		{"debug", LevelDebug},
		{"info", LevelInfo},
		{"warn", LevelWarn},
		{"warning", LevelWarn},
		{"error", LevelError},
		{"DEBUG", LevelDebug},
		{"INFO", LevelInfo},
		{"WARN", LevelWarn},
		{"ERROR", LevelError},
		{"unknown", LevelInfo},
		{"", LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			SetLevel(tt.input)
			if currentLevel != tt.want {
				t.Errorf("SetLevel(%q) = %d, want %d", tt.input, currentLevel, tt.want)
			}
		})
	}
}

func TestInfof(t *testing.T) {
	var buf bytes.Buffer
	setupTestLogger(t, &buf)

	Infof("hello %s", "world")

	if !strings.Contains(buf.String(), "[INFO] hello world") {
		t.Errorf("Infof output = %q, want to contain %q", buf.String(), "[INFO] hello world")
	}
}

func TestErrorf(t *testing.T) {
	var buf bytes.Buffer
	setupTestLogger(t, &buf)

	Errorf("something failed: %v", "timeout")

	if !strings.Contains(buf.String(), "[ERROR] something failed: timeout") {
		t.Errorf("Errorf output = %q, want to contain %q", buf.String(), "[ERROR] something failed: timeout")
	}
}

func TestDebugf(t *testing.T) {
	var buf bytes.Buffer
	setupTestLogger(t, &buf)

	Debugf("debug value: %d", 42)

	if !strings.Contains(buf.String(), "[DEBUG] debug value: 42") {
		t.Errorf("Debugf output = %q, want to contain %q", buf.String(), "[DEBUG] debug value: 42")
	}
}

func TestWarnf(t *testing.T) {
	var buf bytes.Buffer
	setupTestLogger(t, &buf)

	Warnf("low disk: %d%%", 95)

	if !strings.Contains(buf.String(), "[WARN] low disk: 95%") {
		t.Errorf("Warnf output = %q, want to contain %q", buf.String(), "[WARN] low disk: 95%")
	}
}

func TestLevelFiltering(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		logFunc  func()
		wantLog  bool
	}{
		{"info suppresses debug", LevelInfo, func() { Debugf("debug msg") }, false},
		{"info allows info", LevelInfo, func() { Infof("info msg") }, true},
		{"info allows error", LevelInfo, func() { Errorf("error msg") }, true},
		{"error suppresses warn", LevelError, func() { Warnf("warn msg") }, false},
		{"error suppresses info", LevelError, func() { Infof("info msg") }, false},
		{"error allows error", LevelError, func() { Errorf("error msg") }, true},
		{"debug allows all", LevelDebug, func() { Debugf("debug msg") }, true},
		{"warn suppresses debug", LevelWarn, func() { Debugf("debug msg") }, false},
		{"warn allows warn", LevelWarn, func() { Warnf("warn msg") }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger = log.New(&buf, "", 0)
			currentLevel = tt.level

			tt.logFunc()

			hasOutput := buf.Len() > 0
			if hasOutput != tt.wantLog {
				t.Errorf("level=%d: hasOutput=%v, want=%v, buf=%q", tt.level, hasOutput, tt.wantLog, buf.String())
			}
		})
	}
}

func TestNilLoggerSafety(t *testing.T) {
	// Ensure no panic when logger is nil
	savedLogger := logger
	logger = nil
	defer func() { logger = savedLogger }()

	// These should not panic
	Infof("test")
	Errorf("test")
	Debugf("test")
	Warnf("test")
}

func TestLevelConstants(t *testing.T) {
	// Verify ordering: Debug < Info < Warn < Error
	if LevelDebug >= LevelInfo {
		t.Error("LevelDebug should be less than LevelInfo")
	}
	if LevelInfo >= LevelWarn {
		t.Error("LevelInfo should be less than LevelWarn")
	}
	if LevelWarn >= LevelError {
		t.Error("LevelWarn should be less than LevelError")
	}
}
