package db

import (
	"os"
	"path/filepath"
	"testing"
)

func setupWebSettingsDB(t *testing.T) func() {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "k13d-websettings-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	if err := Init(dbPath); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Init() error = %v", err)
	}

	if err := InitWebSettingsTable(); err != nil {
		Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("InitWebSettingsTable() error = %v", err)
	}

	return func() {
		Close()
		os.RemoveAll(tmpDir)
	}
}

func TestInitWebSettingsTable(t *testing.T) {
	cleanup := setupWebSettingsDB(t)
	defer cleanup()

	// Table should exist — calling again should be idempotent
	if err := InitWebSettingsTable(); err != nil {
		t.Errorf("Second InitWebSettingsTable() error = %v", err)
	}
}

func TestSaveAndGetWebSetting(t *testing.T) {
	cleanup := setupWebSettingsDB(t)
	defer cleanup()

	// Save
	if err := SaveWebSetting("theme", "dark"); err != nil {
		t.Fatalf("SaveWebSetting() error = %v", err)
	}

	// Get
	val, err := GetWebSetting("theme")
	if err != nil {
		t.Fatalf("GetWebSetting() error = %v", err)
	}
	if val != "dark" {
		t.Errorf("GetWebSetting(theme) = %q, want %q", val, "dark")
	}

	// Update (upsert)
	if err := SaveWebSetting("theme", "light"); err != nil {
		t.Fatalf("SaveWebSetting update error = %v", err)
	}
	val, err = GetWebSetting("theme")
	if err != nil {
		t.Fatalf("GetWebSetting() after update error = %v", err)
	}
	if val != "light" {
		t.Errorf("GetWebSetting(theme) after update = %q, want %q", val, "light")
	}
}

func TestGetWebSetting_NotFound(t *testing.T) {
	cleanup := setupWebSettingsDB(t)
	defer cleanup()

	val, err := GetWebSetting("nonexistent")
	if err != nil {
		t.Fatalf("GetWebSetting() error = %v", err)
	}
	if val != "" {
		t.Errorf("GetWebSetting(nonexistent) = %q, want empty", val)
	}
}

func TestGetWebSettingWithDefault(t *testing.T) {
	cleanup := setupWebSettingsDB(t)
	defer cleanup()

	// Not set — should return default
	val := GetWebSettingWithDefault("language", "en")
	if val != "en" {
		t.Errorf("GetWebSettingWithDefault = %q, want %q", val, "en")
	}

	// Set — should return actual value
	SaveWebSetting("language", "ko")
	val = GetWebSettingWithDefault("language", "en")
	if val != "ko" {
		t.Errorf("GetWebSettingWithDefault after save = %q, want %q", val, "ko")
	}
}

func TestGetAllWebSettings(t *testing.T) {
	cleanup := setupWebSettingsDB(t)
	defer cleanup()

	SaveWebSetting("key1", "val1")
	SaveWebSetting("key2", "val2")
	SaveWebSetting("key3", "val3")

	all, err := GetAllWebSettings()
	if err != nil {
		t.Fatalf("GetAllWebSettings() error = %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("GetAllWebSettings() returned %d, want 3", len(all))
	}
	if all["key1"] != "val1" {
		t.Errorf("all[key1] = %q, want %q", all["key1"], "val1")
	}
}

func TestGetWebSettingsWithPrefix(t *testing.T) {
	cleanup := setupWebSettingsDB(t)
	defer cleanup()

	SaveWebSetting("llm.provider", "openai")
	SaveWebSetting("llm.model", "gpt-4")
	SaveWebSetting("theme", "dark")

	llmSettings, err := GetWebSettingsWithPrefix("llm.")
	if err != nil {
		t.Fatalf("GetWebSettingsWithPrefix() error = %v", err)
	}
	if len(llmSettings) != 2 {
		t.Fatalf("prefix filter returned %d, want 2", len(llmSettings))
	}
	if llmSettings["llm.provider"] != "openai" {
		t.Errorf("llm.provider = %q, want %q", llmSettings["llm.provider"], "openai")
	}
}

func TestDeleteWebSetting(t *testing.T) {
	cleanup := setupWebSettingsDB(t)
	defer cleanup()

	SaveWebSetting("to-delete", "value")
	if err := DeleteWebSetting("to-delete"); err != nil {
		t.Fatalf("DeleteWebSetting() error = %v", err)
	}

	val, _ := GetWebSetting("to-delete")
	if val != "" {
		t.Errorf("after delete, val = %q, want empty", val)
	}
}

func TestSaveWebSettings_Batch(t *testing.T) {
	cleanup := setupWebSettingsDB(t)
	defer cleanup()

	batch := map[string]string{
		"batch.a": "1",
		"batch.b": "2",
		"batch.c": "3",
	}

	if err := SaveWebSettings(batch); err != nil {
		t.Fatalf("SaveWebSettings() error = %v", err)
	}

	all, _ := GetAllWebSettings()
	if len(all) != 3 {
		t.Fatalf("batch save: got %d settings, want 3", len(all))
	}
	if all["batch.b"] != "2" {
		t.Errorf("batch.b = %q, want %q", all["batch.b"], "2")
	}
}

func TestWebSettings_DBNotInitialized(t *testing.T) {
	// Save current DB and set to nil
	savedDB := DB
	DB = nil
	defer func() { DB = savedDB }()

	if err := InitWebSettingsTable(); err != ErrDBNotInitialized {
		t.Errorf("InitWebSettingsTable() = %v, want ErrDBNotInitialized", err)
	}
	if err := SaveWebSetting("k", "v"); err != ErrDBNotInitialized {
		t.Errorf("SaveWebSetting() = %v, want ErrDBNotInitialized", err)
	}
	if _, err := GetWebSetting("k"); err != ErrDBNotInitialized {
		t.Errorf("GetWebSetting() = %v, want ErrDBNotInitialized", err)
	}
	if _, err := GetAllWebSettings(); err != ErrDBNotInitialized {
		t.Errorf("GetAllWebSettings() = %v, want ErrDBNotInitialized", err)
	}
	if _, err := GetWebSettingsWithPrefix("p"); err != ErrDBNotInitialized {
		t.Errorf("GetWebSettingsWithPrefix() = %v, want ErrDBNotInitialized", err)
	}
	if err := DeleteWebSetting("k"); err != ErrDBNotInitialized {
		t.Errorf("DeleteWebSetting() = %v, want ErrDBNotInitialized", err)
	}
	if err := SaveWebSettings(map[string]string{"k": "v"}); err != ErrDBNotInitialized {
		t.Errorf("SaveWebSettings() = %v, want ErrDBNotInitialized", err)
	}
}
