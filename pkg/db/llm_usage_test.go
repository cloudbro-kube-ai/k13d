package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestLLMUsageRecording(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_llm_usage.db")
	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer func() { _ = Close() }()

	// Test recording LLM usage
	record := LLMUsageRecord{
		Timestamp:         time.Now(),
		RequestID:         "test-req-001",
		User:              "testuser",
		Provider:          "openai",
		Model:             "gpt-4",
		PromptTokens:      100,
		CompletionTokens:  50,
		TotalTokens:       150,
		RequestDurationMs: 500,
		Success:           true,
	}

	if err := RecordLLMUsage(record); err != nil {
		t.Fatalf("Failed to record LLM usage: %v", err)
	}

	// Record more entries
	for i := 0; i < 5; i++ {
		r := LLMUsageRecord{
			Timestamp:         time.Now().Add(-time.Duration(i) * time.Hour),
			RequestID:         "test-req-" + string(rune('a'+i)),
			User:              "testuser",
			Provider:          "anthropic",
			Model:             "claude-3-sonnet",
			PromptTokens:      200 + i*10,
			CompletionTokens:  100 + i*5,
			TotalTokens:       300 + i*15,
			RequestDurationMs: int64(400 + i*50),
			Success:           true,
		}
		if err := RecordLLMUsage(r); err != nil {
			t.Fatalf("Failed to record LLM usage %d: %v", i, err)
		}
	}
}

func TestGetLLMUsage(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_llm_usage_get.db")
	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer func() { _ = Close() }()

	ctx := context.Background()

	// Record test data
	baseTime := time.Now()
	records := []LLMUsageRecord{
		{Timestamp: baseTime, RequestID: "req-1", User: "user1", Provider: "openai", Model: "gpt-4", PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150, Success: true},
		{Timestamp: baseTime.Add(-time.Hour), RequestID: "req-2", User: "user1", Provider: "openai", Model: "gpt-4", PromptTokens: 200, CompletionTokens: 100, TotalTokens: 300, Success: true},
		{Timestamp: baseTime.Add(-2 * time.Hour), RequestID: "req-3", User: "user2", Provider: "anthropic", Model: "claude-3", PromptTokens: 150, CompletionTokens: 75, TotalTokens: 225, Success: false, ErrorMessage: "rate limit"},
	}

	for _, r := range records {
		if err := RecordLLMUsage(r); err != nil {
			t.Fatalf("Failed to record: %v", err)
		}
	}

	// Test get with no filter
	filter := LLMUsageFilter{Limit: 100}
	results, err := GetLLMUsage(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to get LLM usage: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("Expected 3 records, got %d", len(results))
	}

	// Test filter by user
	filter = LLMUsageFilter{User: "user1", Limit: 100}
	results, err = GetLLMUsage(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to get LLM usage by user: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 records for user1, got %d", len(results))
	}

	// Test filter by provider
	filter = LLMUsageFilter{Provider: "anthropic", Limit: 100}
	results, err = GetLLMUsage(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to get LLM usage by provider: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 record for anthropic, got %d", len(results))
	}

	// Test filter by model
	filter = LLMUsageFilter{Model: "gpt-4", Limit: 100}
	results, err = GetLLMUsage(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to get LLM usage by model: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 records for gpt-4, got %d", len(results))
	}

	// Test limit
	filter = LLMUsageFilter{Limit: 2}
	results, err = GetLLMUsage(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to get LLM usage with limit: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 records with limit, got %d", len(results))
	}
}

func TestGetLLMUsageStats(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_llm_usage_stats.db")
	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer func() { _ = Close() }()

	ctx := context.Background()

	// Record test data
	baseTime := time.Now()
	records := []LLMUsageRecord{
		{Timestamp: baseTime, RequestID: "req-1", Provider: "openai", Model: "gpt-4", PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150, RequestDurationMs: 500, Success: true},
		{Timestamp: baseTime.Add(-30 * time.Minute), RequestID: "req-2", Provider: "openai", Model: "gpt-4", PromptTokens: 200, CompletionTokens: 100, TotalTokens: 300, RequestDurationMs: 600, Success: true},
		{Timestamp: baseTime.Add(-time.Hour), RequestID: "req-3", Provider: "anthropic", Model: "claude-3", PromptTokens: 150, CompletionTokens: 75, TotalTokens: 225, RequestDurationMs: 400, Success: true},
		{Timestamp: baseTime.Add(-2 * time.Hour), RequestID: "req-4", Provider: "openai", Model: "gpt-3.5", PromptTokens: 50, CompletionTokens: 25, TotalTokens: 75, RequestDurationMs: 200, Success: false},
	}

	for _, r := range records {
		if err := RecordLLMUsage(r); err != nil {
			t.Fatalf("Failed to record: %v", err)
		}
	}

	// Get stats for last 24 hours
	filter := LLMUsageFilter{
		StartTime: baseTime.Add(-24 * time.Hour),
	}
	stats, err := GetLLMUsageStats(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to get LLM usage stats: %v", err)
	}

	// Verify stats
	if stats.TotalRequests != 4 {
		t.Errorf("Expected 4 total requests, got %d", stats.TotalRequests)
	}
	if stats.TotalTokens != 750 { // 150+300+225+75
		t.Errorf("Expected 750 total tokens, got %d", stats.TotalTokens)
	}
	if stats.TotalPromptTokens != 500 { // 100+200+150+50
		t.Errorf("Expected 500 prompt tokens, got %d", stats.TotalPromptTokens)
	}
	if stats.TotalCompTokens != 250 { // 50+100+75+25
		t.Errorf("Expected 250 completion tokens, got %d", stats.TotalCompTokens)
	}

	// Verify model breakdown
	if len(stats.ByModel) < 2 {
		t.Errorf("Expected at least 2 models in breakdown, got %d", len(stats.ByModel))
	}
}

func TestLLMUsageWithEmptyDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_llm_usage_empty.db")
	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer func() { _ = Close() }()

	ctx := context.Background()

	// Get usage from empty DB
	filter := LLMUsageFilter{Limit: 100}
	results, err := GetLLMUsage(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to get LLM usage from empty DB: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 records from empty DB, got %d", len(results))
	}

	// Get stats from empty DB
	stats, err := GetLLMUsageStats(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to get stats from empty DB: %v", err)
	}
	if stats.TotalRequests != 0 {
		t.Errorf("Expected 0 total requests from empty DB, got %d", stats.TotalRequests)
	}
}
