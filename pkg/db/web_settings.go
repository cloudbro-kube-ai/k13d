package db

import (
	"database/sql"
	"fmt"
	"time"
)

// WebSetting represents a key-value setting stored in SQLite
type WebSetting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// InitWebSettingsTable creates the web_settings table if it doesn't exist.
// TODO: DDL uses SQLite-only syntax (TEXT PRIMARY KEY, ON CONFLICT).
// Add multi-DB DDL variants when supporting Postgres/MySQL.
func InitWebSettingsTable() error {
	if DB == nil {
		return ErrDBNotInitialized
	}

	query := `
	CREATE TABLE IF NOT EXISTS web_settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := DB.Exec(query)
	return err
}

// SaveWebSetting saves or updates a web setting
func SaveWebSetting(key, value string) error {
	if DB == nil {
		return ErrDBNotInitialized
	}

	query := `
	INSERT INTO web_settings (key, value, updated_at)
	VALUES (?, ?, ?)
	ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`

	_, err := DB.Exec(query, key, value, time.Now())
	return err
}

// GetWebSetting retrieves a web setting by key
func GetWebSetting(key string) (string, error) {
	if DB == nil {
		return "", ErrDBNotInitialized
	}

	var value string
	err := DB.QueryRow("SELECT value FROM web_settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// GetWebSettingWithDefault retrieves a web setting or returns the default value
func GetWebSettingWithDefault(key, defaultValue string) string {
	value, err := GetWebSetting(key)
	if err != nil || value == "" {
		return defaultValue
	}
	return value
}

// GetAllWebSettings retrieves all web settings
func GetAllWebSettings() (map[string]string, error) {
	if DB == nil {
		return nil, ErrDBNotInitialized
	}

	rows, err := DB.Query("SELECT key, value FROM web_settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		settings[key] = value
	}
	return settings, nil
}

// GetWebSettingsWithPrefix retrieves all settings with a given prefix
func GetWebSettingsWithPrefix(prefix string) (map[string]string, error) {
	if DB == nil {
		return nil, ErrDBNotInitialized
	}

	rows, err := DB.Query("SELECT key, value FROM web_settings WHERE key LIKE ?", prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		settings[key] = value
	}
	return settings, nil
}

// DeleteWebSetting deletes a web setting by key
func DeleteWebSetting(key string) error {
	if DB == nil {
		return ErrDBNotInitialized
	}

	_, err := DB.Exec("DELETE FROM web_settings WHERE key = ?", key)
	return err
}

// SaveWebSettings saves multiple settings at once
func SaveWebSettings(settings map[string]string) error {
	if DB == nil {
		return ErrDBNotInitialized
	}

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
		INSERT INTO web_settings (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for key, value := range settings {
		if _, err := stmt.Exec(key, value, now); err != nil {
			return fmt.Errorf("failed to save setting %s: %w", key, err)
		}
	}

	return tx.Commit()
}
