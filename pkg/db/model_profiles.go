package db

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// ErrDBNotInitialized is returned when database is not initialized
var ErrDBNotInitialized = errors.New("database not initialized")

// ModelProfile represents a saved LLM model configuration in the database
type ModelProfile struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Provider        string    `json:"provider"`
	Model           string    `json:"model"`
	Endpoint        string    `json:"endpoint,omitempty"`
	APIKeyHash      string    `json:"-"`                // Hashed API key for security
	HasAPIKey       bool      `json:"has_api_key"`      // Indicates if API key is set
	Region          string    `json:"region,omitempty"` // For AWS Bedrock
	AzureDeployment string    `json:"azure_deployment,omitempty"`
	Description     string    `json:"description,omitempty"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	CreatedBy       string    `json:"created_by,omitempty"`

	// Usage statistics (populated from llm_usage table)
	TotalRequests int        `json:"total_requests,omitempty"`
	TotalTokens   int        `json:"total_tokens,omitempty"`
	LastUsedAt    *time.Time `json:"last_used_at,omitempty"`
}

// InitModelProfilesTable creates the model_profiles table if it doesn't exist
func InitModelProfilesTable() error {
	if DB == nil {
		return ErrDBNotInitialized
	}

	query := `
	CREATE TABLE IF NOT EXISTS model_profiles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		endpoint TEXT,
		api_key_hash TEXT,
		region TEXT,
		azure_deployment TEXT,
		description TEXT,
		is_active INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_by TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_model_profiles_name ON model_profiles(name);
	CREATE INDEX IF NOT EXISTS idx_model_profiles_provider ON model_profiles(provider);
	CREATE INDEX IF NOT EXISTS idx_model_profiles_active ON model_profiles(is_active);
	`

	_, err := DB.Exec(query)
	return err
}

// SaveModelProfile creates or updates a model profile
func SaveModelProfile(profile *ModelProfile) error {
	if DB == nil {
		return ErrDBNotInitialized
	}

	now := time.Now()

	// Check if profile exists
	var existingID int64
	err := DB.QueryRow("SELECT id FROM model_profiles WHERE name = ?", profile.Name).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Insert new profile
		query := `
		INSERT INTO model_profiles (
			name, provider, model, endpoint, api_key_hash,
			region, azure_deployment, description, is_active,
			created_at, updated_at, created_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

		result, err := DB.Exec(query,
			profile.Name, profile.Provider, profile.Model, profile.Endpoint, profile.APIKeyHash,
			profile.Region, profile.AzureDeployment, profile.Description, boolToInt(profile.IsActive),
			now, now, profile.CreatedBy,
		)
		if err != nil {
			return err
		}

		profile.ID, _ = result.LastInsertId()
		profile.CreatedAt = now
		profile.UpdatedAt = now
	} else if err == nil {
		// Update existing profile
		query := `
		UPDATE model_profiles SET
			provider = ?, model = ?, endpoint = ?,
			region = ?, azure_deployment = ?, description = ?,
			is_active = ?, updated_at = ?
		WHERE id = ?`

		// Only update api_key_hash if provided
		if profile.APIKeyHash != "" {
			query = `
			UPDATE model_profiles SET
				provider = ?, model = ?, endpoint = ?, api_key_hash = ?,
				region = ?, azure_deployment = ?, description = ?,
				is_active = ?, updated_at = ?
			WHERE id = ?`
			_, err = DB.Exec(query,
				profile.Provider, profile.Model, profile.Endpoint, profile.APIKeyHash,
				profile.Region, profile.AzureDeployment, profile.Description,
				boolToInt(profile.IsActive), now, existingID,
			)
		} else {
			_, err = DB.Exec(query,
				profile.Provider, profile.Model, profile.Endpoint,
				profile.Region, profile.AzureDeployment, profile.Description,
				boolToInt(profile.IsActive), now, existingID,
			)
		}

		if err != nil {
			return err
		}

		profile.ID = existingID
		profile.UpdatedAt = now
	} else {
		return err
	}

	return nil
}

// GetModelProfiles retrieves all model profiles with optional usage stats
func GetModelProfiles(ctx context.Context, includeStats bool) ([]ModelProfile, error) {
	if DB == nil {
		return nil, ErrDBNotInitialized
	}

	query := `
	SELECT id, name, provider, model, endpoint,
		   CASE WHEN api_key_hash IS NOT NULL AND api_key_hash != '' THEN 1 ELSE 0 END as has_api_key,
		   region, azure_deployment, description, is_active,
		   created_at, updated_at, created_by
	FROM model_profiles
	ORDER BY is_active DESC, name ASC`

	rows, err := DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []ModelProfile
	for rows.Next() {
		var p ModelProfile
		var hasAPIKey int
		var isActive int
		var region, azureDeployment, description, createdBy sql.NullString

		err := rows.Scan(
			&p.ID, &p.Name, &p.Provider, &p.Model, &p.Endpoint,
			&hasAPIKey, &region, &azureDeployment, &description, &isActive,
			&p.CreatedAt, &p.UpdatedAt, &createdBy,
		)
		if err != nil {
			continue
		}

		p.HasAPIKey = hasAPIKey == 1
		p.IsActive = isActive == 1
		p.Region = region.String
		p.AzureDeployment = azureDeployment.String
		p.Description = description.String
		p.CreatedBy = createdBy.String

		profiles = append(profiles, p)
	}

	// Add usage statistics if requested
	if includeStats && len(profiles) > 0 {
		for i := range profiles {
			stats, err := getModelUsageStats(ctx, profiles[i].Provider, profiles[i].Model)
			if err == nil {
				profiles[i].TotalRequests = stats.TotalRequests
				profiles[i].TotalTokens = stats.TotalTokens
				profiles[i].LastUsedAt = stats.LastUsedAt
			}
		}
	}

	return profiles, nil
}

// GetModelProfileByName retrieves a model profile by name
func GetModelProfileByName(ctx context.Context, name string) (*ModelProfile, error) {
	if DB == nil {
		return nil, ErrDBNotInitialized
	}

	query := `
	SELECT id, name, provider, model, endpoint, api_key_hash,
		   CASE WHEN api_key_hash IS NOT NULL AND api_key_hash != '' THEN 1 ELSE 0 END as has_api_key,
		   region, azure_deployment, description, is_active,
		   created_at, updated_at, created_by
	FROM model_profiles
	WHERE name = ?`

	var p ModelProfile
	var hasAPIKey int
	var isActive int
	var apiKeyHash sql.NullString
	var region, azureDeployment, description, createdBy sql.NullString

	err := DB.QueryRowContext(ctx, query, name).Scan(
		&p.ID, &p.Name, &p.Provider, &p.Model, &p.Endpoint, &apiKeyHash,
		&hasAPIKey, &region, &azureDeployment, &description, &isActive,
		&p.CreatedAt, &p.UpdatedAt, &createdBy,
	)
	if err != nil {
		return nil, err
	}

	p.APIKeyHash = apiKeyHash.String
	p.HasAPIKey = hasAPIKey == 1
	p.IsActive = isActive == 1
	p.Region = region.String
	p.AzureDeployment = azureDeployment.String
	p.Description = description.String
	p.CreatedBy = createdBy.String

	return &p, nil
}

// DeleteModelProfile deletes a model profile by name
func DeleteModelProfile(name string) error {
	if DB == nil {
		return ErrDBNotInitialized
	}

	_, err := DB.Exec("DELETE FROM model_profiles WHERE name = ?", name)
	return err
}

// SetActiveModelProfile sets a profile as active and deactivates others
func SetActiveModelProfile(name string) error {
	if DB == nil {
		return ErrDBNotInitialized
	}

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Deactivate all profiles
	_, err = tx.Exec("UPDATE model_profiles SET is_active = 0")
	if err != nil {
		return err
	}

	// Activate the selected profile
	_, err = tx.Exec("UPDATE model_profiles SET is_active = 1, updated_at = ? WHERE name = ?",
		time.Now(), name)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetActiveModelProfile returns the currently active model profile
func GetActiveModelProfile(ctx context.Context) (*ModelProfile, error) {
	if DB == nil {
		return nil, ErrDBNotInitialized
	}

	query := `
	SELECT id, name, provider, model, endpoint, api_key_hash,
		   CASE WHEN api_key_hash IS NOT NULL AND api_key_hash != '' THEN 1 ELSE 0 END as has_api_key,
		   region, azure_deployment, description, is_active,
		   created_at, updated_at, created_by
	FROM model_profiles
	WHERE is_active = 1
	LIMIT 1`

	var p ModelProfile
	var hasAPIKey int
	var isActive int
	var apiKeyHash sql.NullString
	var region, azureDeployment, description, createdBy sql.NullString

	err := DB.QueryRowContext(ctx, query).Scan(
		&p.ID, &p.Name, &p.Provider, &p.Model, &p.Endpoint, &apiKeyHash,
		&hasAPIKey, &region, &azureDeployment, &description, &isActive,
		&p.CreatedAt, &p.UpdatedAt, &createdBy,
	)
	if err != nil {
		return nil, err
	}

	p.APIKeyHash = apiKeyHash.String
	p.HasAPIKey = hasAPIKey == 1
	p.IsActive = isActive == 1
	p.Region = region.String
	p.AzureDeployment = azureDeployment.String
	p.Description = description.String
	p.CreatedBy = createdBy.String

	return &p, nil
}

// modelUsageStats holds usage statistics for a model
type modelUsageStats struct {
	TotalRequests int
	TotalTokens   int
	LastUsedAt    *time.Time
}

// getModelUsageStats retrieves usage statistics for a specific model
func getModelUsageStats(ctx context.Context, provider, model string) (*modelUsageStats, error) {
	if DB == nil {
		return nil, ErrDBNotInitialized
	}

	query := `
	SELECT
		COUNT(*) as total_requests,
		COALESCE(SUM(total_tokens), 0) as total_tokens,
		MAX(timestamp) as last_used
	FROM llm_usage
	WHERE provider = ? AND model = ?`

	var stats modelUsageStats
	var lastUsed sql.NullString

	err := DB.QueryRowContext(ctx, query, provider, model).Scan(
		&stats.TotalRequests,
		&stats.TotalTokens,
		&lastUsed,
	)
	if err != nil {
		return nil, err
	}

	if lastUsed.Valid && lastUsed.String != "" {
		// Try multiple datetime formats
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05Z",
			time.RFC3339,
		}
		for _, format := range formats {
			if t, err := time.Parse(format, lastUsed.String); err == nil {
				stats.LastUsedAt = &t
				break
			}
		}
	}

	return &stats, nil
}

// SyncModelProfilesFromConfig syncs model profiles from config to database
func SyncModelProfilesFromConfig(profiles []struct {
	Name            string
	Provider        string
	Model           string
	Endpoint        string
	APIKey          string
	Region          string
	AzureDeployment string
	Description     string
}, activeModel string) error {
	if DB == nil {
		return ErrDBNotInitialized
	}

	for _, p := range profiles {
		profile := &ModelProfile{
			Name:            p.Name,
			Provider:        p.Provider,
			Model:           p.Model,
			Endpoint:        p.Endpoint,
			Region:          p.Region,
			AzureDeployment: p.AzureDeployment,
			Description:     p.Description,
			IsActive:        p.Name == activeModel,
		}

		// Hash API key if provided
		if p.APIKey != "" {
			profile.APIKeyHash = hashAPIKey(p.APIKey)
		}

		if err := SaveModelProfile(profile); err != nil {
			return err
		}
	}

	return nil
}

// hashAPIKey creates a simple hash of the API key for comparison
// Note: This is not for security, just to detect if key has changed
func hashAPIKey(apiKey string) string {
	if apiKey == "" {
		return ""
	}
	// Simple hash - first 4 chars + length + last 4 chars
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
