package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/db"
)

func TestHandleLLMUsage_MethodNotAllowed(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(http.MethodPost, "/api/llm/usage", nil)
	w := httptest.NewRecorder()

	server.handleLLMUsage(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleLLMUsage_EmptyDB(t *testing.T) {
	dbPath := "test_llm_handler.db"
	defer os.Remove(dbPath)

	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	server := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/api/llm/usage", nil)
	w := httptest.NewRecorder()

	server.handleLLMUsage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Response uses "items" key
	items, ok := resp["items"]
	if !ok {
		t.Fatal("Expected items key in response")
	}
	// Items can be nil or empty array
	if items != nil {
		records, ok := items.([]interface{})
		if ok && len(records) != 0 {
			t.Errorf("Expected 0 records, got %d", len(records))
		}
	}
}

func TestHandleLLMUsage_WithData(t *testing.T) {
	dbPath := "test_llm_handler_data.db"
	defer os.Remove(dbPath)

	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	// Insert test data
	record := db.LLMUsageRecord{
		Timestamp:        time.Now(),
		RequestID:        "test-001",
		User:             "testuser",
		Provider:         "openai",
		Model:            "gpt-4",
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		Success:          true,
	}
	if err := db.RecordLLMUsage(record); err != nil {
		t.Fatalf("Failed to record LLM usage: %v", err)
	}

	server := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/api/llm/usage?limit=10", nil)
	w := httptest.NewRecorder()

	server.handleLLMUsage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check count
	count, ok := resp["count"].(float64)
	if !ok {
		t.Fatal("Expected count in response")
	}
	if int(count) != 1 {
		t.Errorf("Expected count 1, got %d", int(count))
	}
}

func TestHandleLLMUsage_WithFilters(t *testing.T) {
	dbPath := "test_llm_handler_filter.db"
	defer os.Remove(dbPath)

	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	// Insert test data with different providers
	records := []db.LLMUsageRecord{
		{Timestamp: time.Now(), RequestID: "r1", User: "user1", Provider: "openai", Model: "gpt-4", TotalTokens: 100, Success: true},
		{Timestamp: time.Now(), RequestID: "r2", User: "user2", Provider: "anthropic", Model: "claude-3", TotalTokens: 200, Success: true},
		{Timestamp: time.Now(), RequestID: "r3", User: "user1", Provider: "openai", Model: "gpt-3.5", TotalTokens: 50, Success: true},
	}
	for _, r := range records {
		if err := db.RecordLLMUsage(r); err != nil {
			t.Fatalf("Failed to record: %v", err)
		}
	}

	server := &Server{}

	// Filter by provider
	req := httptest.NewRequest(http.MethodGet, "/api/llm/usage?provider=openai", nil)
	w := httptest.NewRecorder()
	server.handleLLMUsage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	count := int(resp["count"].(float64))
	if count != 2 {
		t.Errorf("Expected 2 openai records, got %d", count)
	}

	// Filter by user
	req = httptest.NewRequest(http.MethodGet, "/api/llm/usage?user=user1", nil)
	w = httptest.NewRecorder()
	server.handleLLMUsage(w, req)

	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	count = int(resp["count"].(float64))
	if count != 2 {
		t.Errorf("Expected 2 user1 records, got %d", count)
	}
}

func TestHandleLLMUsageStats_MethodNotAllowed(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(http.MethodPost, "/api/llm/usage/stats", nil)
	w := httptest.NewRecorder()

	server.handleLLMUsageStats(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleLLMUsageStats_EmptyDB(t *testing.T) {
	dbPath := "test_llm_stats_empty.db"
	defer os.Remove(dbPath)

	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	server := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/api/llm/usage/stats", nil)
	w := httptest.NewRecorder()

	server.handleLLMUsageStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Response wraps stats in "stats" key
	stats, ok := resp["stats"].(map[string]interface{})
	if !ok {
		// Empty stats might be nil
		if resp["stats"] == nil {
			return // OK for empty DB
		}
		t.Fatalf("Expected stats object in response, got %T", resp["stats"])
	}

	if total, ok := stats["total_requests"].(float64); ok && total != 0 {
		t.Errorf("Expected 0 total requests, got %v", total)
	}
}

func TestHandleLLMUsageStats_WithData(t *testing.T) {
	dbPath := "test_llm_stats_data.db"
	defer os.Remove(dbPath)

	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	// Insert test data
	records := []db.LLMUsageRecord{
		{Timestamp: time.Now(), RequestID: "r1", Provider: "openai", Model: "gpt-4", PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150, Success: true},
		{Timestamp: time.Now().Add(-time.Hour), RequestID: "r2", Provider: "openai", Model: "gpt-4", PromptTokens: 200, CompletionTokens: 100, TotalTokens: 300, Success: true},
		{Timestamp: time.Now().Add(-2 * time.Hour), RequestID: "r3", Provider: "anthropic", Model: "claude-3", PromptTokens: 150, CompletionTokens: 75, TotalTokens: 225, Success: false},
	}
	for _, r := range records {
		if err := db.RecordLLMUsage(r); err != nil {
			t.Fatalf("Failed to record: %v", err)
		}
	}

	server := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/api/llm/usage/stats?hours=24", nil)
	w := httptest.NewRecorder()

	server.handleLLMUsageStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Response wraps stats in "stats" key
	stats, ok := resp["stats"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected stats object in response, got %T", resp["stats"])
	}

	totalRequests := int(stats["total_requests"].(float64))
	if totalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", totalRequests)
	}

	totalTokens := int(stats["total_tokens"].(float64))
	if totalTokens != 675 { // 150+300+225
		t.Errorf("Expected 675 total tokens, got %d", totalTokens)
	}

	// Check model breakdown (it's a map, not array)
	byModel, ok := stats["by_model"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected by_model map in stats")
	}
	if len(byModel) < 2 {
		t.Errorf("Expected at least 2 models, got %d", len(byModel))
	}
}

func TestHandleLLMUsageStats_CustomHours(t *testing.T) {
	dbPath := "test_llm_stats_hours.db"
	defer os.Remove(dbPath)

	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	// Insert recent data
	records := []db.LLMUsageRecord{
		{Timestamp: time.Now(), RequestID: "r1", Provider: "openai", Model: "gpt-4", TotalTokens: 100, Success: true},
		{Timestamp: time.Now(), RequestID: "r2", Provider: "openai", Model: "gpt-4", TotalTokens: 200, Success: true},
	}
	for _, r := range records {
		if err := db.RecordLLMUsage(r); err != nil {
			t.Fatalf("Failed to record: %v", err)
		}
	}

	server := &Server{}

	// Request stats for last 168 hours (7 days) - should include all records
	req := httptest.NewRequest(http.MethodGet, "/api/llm/usage/stats?hours=168", nil)
	w := httptest.NewRecorder()
	server.handleLLMUsageStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify minutes parameter is reflected in response (168 hours = 10080 minutes)
	minutes, ok := resp["minutes"].(float64)
	if !ok || int(minutes) != 168*60 {
		t.Errorf("Expected minutes=10080 (168 hours), got %v", minutes)
	}
}
