package db

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func TestModelProfilesCRUD(t *testing.T) {
	dbPath := "test_model_profiles.db"
	defer os.Remove(dbPath)

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}
	defer Close()

	ctx := context.Background()

	// Create a profile
	profile := &ModelProfile{
		Name:        "test-gpt4",
		Provider:    "openai",
		Model:       "gpt-4",
		Endpoint:    "https://api.openai.com/v1",
		Description: "Test GPT-4 profile",
		IsActive:    true,
		CreatedBy:   "test-user",
	}

	// Test Save
	if err := SaveModelProfile(profile); err != nil {
		t.Fatalf("Failed to save profile: %v", err)
	}

	if profile.ID == 0 {
		t.Error("Profile ID should be set after save")
	}

	// Test Get by name
	retrieved, err := GetModelProfileByName(ctx, "test-gpt4")
	if err != nil {
		t.Fatalf("Failed to get profile: %v", err)
	}

	if retrieved.Provider != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", retrieved.Provider)
	}

	if retrieved.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", retrieved.Model)
	}

	if !retrieved.IsActive {
		t.Error("Profile should be active")
	}

	// Test GetAll
	profiles, err := GetModelProfiles(ctx, false)
	if err != nil {
		t.Fatalf("Failed to get profiles: %v", err)
	}

	if len(profiles) != 1 {
		t.Errorf("Expected 1 profile, got %d", len(profiles))
	}

	// Test Update
	profile.Description = "Updated description"
	if err := SaveModelProfile(profile); err != nil {
		t.Fatalf("Failed to update profile: %v", err)
	}

	updated, _ := GetModelProfileByName(ctx, "test-gpt4")
	if updated.Description != "Updated description" {
		t.Error("Description should be updated")
	}

	// Test Delete
	if err := DeleteModelProfile("test-gpt4"); err != nil {
		t.Fatalf("Failed to delete profile: %v", err)
	}

	_, err = GetModelProfileByName(ctx, "test-gpt4")
	if err == nil {
		t.Error("Profile should be deleted")
	}
}

func TestSetActiveModelProfile(t *testing.T) {
	dbPath := "test_model_profiles_active.db"
	defer os.Remove(dbPath)

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}
	defer Close()

	ctx := context.Background()

	// Create two profiles
	profile1 := &ModelProfile{
		Name:     "profile1",
		Provider: "openai",
		Model:    "gpt-4",
		IsActive: true,
	}
	profile2 := &ModelProfile{
		Name:     "profile2",
		Provider: "anthropic",
		Model:    "claude-3",
		IsActive: false,
	}

	SaveModelProfile(profile1)
	SaveModelProfile(profile2)

	// Set profile2 as active
	if err := SetActiveModelProfile("profile2"); err != nil {
		t.Fatalf("Failed to set active profile: %v", err)
	}

	// Check that profile2 is now active
	active, err := GetActiveModelProfile(ctx)
	if err != nil {
		t.Fatalf("Failed to get active profile: %v", err)
	}

	if active.Name != "profile2" {
		t.Errorf("Expected active profile 'profile2', got '%s'", active.Name)
	}

	// Check that profile1 is no longer active
	p1, _ := GetModelProfileByName(ctx, "profile1")
	if p1.IsActive {
		t.Error("profile1 should no longer be active")
	}
}

func TestModelProfileWithStats(t *testing.T) {
	dbPath := "test_model_profiles_stats.db"
	defer os.Remove(dbPath)

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}
	defer Close()

	ctx := context.Background()

	// Create a profile
	profile := &ModelProfile{
		Name:     "stats-test",
		Provider: "openai",
		Model:    "gpt-4",
		IsActive: true,
	}
	SaveModelProfile(profile)

	// Record some usage
	for i := 0; i < 3; i++ {
		record := LLMUsageRecord{
			RequestID:        fmt.Sprintf("req-%d", i),
			User:             "test-user",
			Provider:         "openai",
			Model:            "gpt-4",
			RequestType:      "chat",
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			Success:          true,
		}
		if err := RecordLLMUsage(record); err != nil {
			t.Fatalf("Failed to record LLM usage: %v", err)
		}
	}

	// Get profiles with stats
	profiles, err := GetModelProfiles(ctx, true)
	if err != nil {
		t.Fatalf("Failed to get profiles: %v", err)
	}

	if len(profiles) != 1 {
		t.Fatalf("Expected 1 profile, got %d", len(profiles))
	}

	if profiles[0].TotalRequests != 3 {
		t.Errorf("Expected 3 requests, got %d", profiles[0].TotalRequests)
	}

	if profiles[0].TotalTokens != 450 {
		t.Errorf("Expected 450 tokens, got %d", profiles[0].TotalTokens)
	}
}

func TestHashAPIKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"short", "***"},
		{"sk-1234567890abcdefghij", "sk-1...ghij"},
	}

	for _, test := range tests {
		result := hashAPIKey(test.input)
		if result != test.expected {
			t.Errorf("hashAPIKey(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}
