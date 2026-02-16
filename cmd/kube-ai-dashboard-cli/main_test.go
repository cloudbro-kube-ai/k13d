package main

import "testing"

// TestVersionDefaults verifies the build-time variables have sane defaults.
// This serves as a basic smoke test that the cmd package compiles.
func TestVersionDefaults(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
	if BuildTime == "" {
		t.Error("BuildTime should not be empty")
	}
	if GitCommit == "" {
		t.Error("GitCommit should not be empty")
	}
}
