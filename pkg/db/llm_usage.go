package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// LLMUsageRecord represents a single LLM API call with token usage
type LLMUsageRecord struct {
	ID                int64     `json:"id"`
	Timestamp         time.Time `json:"timestamp"`
	RequestID         string    `json:"request_id"`
	User              string    `json:"user"`
	Provider          string    `json:"provider"`
	Model             string    `json:"model"`
	RequestType       string    `json:"request_type"` // chat, completion, embedding
	PromptTokens      int       `json:"prompt_tokens"`
	CompletionTokens  int       `json:"completion_tokens"`
	TotalTokens       int       `json:"total_tokens"`
	ToolsCalled       int       `json:"tools_called"`
	RequestDurationMs int64     `json:"request_duration_ms"`
	Success           bool      `json:"success"`
	ErrorMessage      string    `json:"error_message,omitempty"`
}

// LLMUsageFilter contains filter criteria for querying LLM usage records
type LLMUsageFilter struct {
	User      string
	Model     string
	Provider  string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
}

// LLMUsageStats contains aggregated statistics for LLM usage
type LLMUsageStats struct {
	TotalRequests      int64                      `json:"total_requests"`
	TotalPromptTokens  int64                      `json:"total_prompt_tokens"`
	TotalCompTokens    int64                      `json:"total_completion_tokens"`
	TotalTokens        int64                      `json:"total_tokens"`
	TotalToolsCalled   int64                      `json:"total_tools_called"`
	AvgRequestDuration float64                    `json:"avg_request_duration_ms"`
	SuccessRate        float64                    `json:"success_rate"`
	ByModel            map[string]ModelUsageStats `json:"by_model"`
	ByUser             map[string]int64           `json:"by_user"`
	TimeSeriesHourly   []TimeSeriesPoint          `json:"time_series_hourly,omitempty"`
}

// ModelUsageStats contains usage statistics for a specific model
type ModelUsageStats struct {
	Requests     int64 `json:"requests"`
	TotalTokens  int64 `json:"total_tokens"`
	PromptTokens int64 `json:"prompt_tokens"`
	CompTokens   int64 `json:"completion_tokens"`
}

// TimeSeriesPoint represents a single point in time series data
type TimeSeriesPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	Requests     int64     `json:"requests"`
	TotalTokens  int64     `json:"total_tokens"`
	PromptTokens int64     `json:"prompt_tokens"`
	CompTokens   int64     `json:"completion_tokens"`
}

// InitLLMUsageTable creates the llm_usage table if it doesn't exist
func InitLLMUsageTable() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	query := `
	CREATE TABLE IF NOT EXISTS llm_usage (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		request_id TEXT NOT NULL,
		user TEXT NOT NULL DEFAULT 'anonymous',
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		request_type TEXT DEFAULT 'chat',
		prompt_tokens INTEGER DEFAULT 0,
		completion_tokens INTEGER DEFAULT 0,
		total_tokens INTEGER DEFAULT 0,
		tools_called INTEGER DEFAULT 0,
		request_duration_ms INTEGER DEFAULT 0,
		success INTEGER DEFAULT 1,
		error_message TEXT DEFAULT ''
	);

	CREATE INDEX IF NOT EXISTS idx_llm_usage_timestamp ON llm_usage(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_llm_usage_user ON llm_usage(user);
	CREATE INDEX IF NOT EXISTS idx_llm_usage_model ON llm_usage(model);
	CREATE INDEX IF NOT EXISTS idx_llm_usage_provider ON llm_usage(provider);
	`

	_, err := DB.Exec(query)
	return err
}

// RecordLLMUsage inserts a new LLM usage record
func RecordLLMUsage(record LLMUsageRecord) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	query := `
	INSERT INTO llm_usage (
		request_id, user, provider, model, request_type,
		prompt_tokens, completion_tokens, total_tokens,
		tools_called, request_duration_ms, success, error_message
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	success := 1
	if !record.Success {
		success = 0
	}

	_, err := DB.Exec(query,
		record.RequestID,
		record.User,
		record.Provider,
		record.Model,
		record.RequestType,
		record.PromptTokens,
		record.CompletionTokens,
		record.TotalTokens,
		record.ToolsCalled,
		record.RequestDurationMs,
		success,
		record.ErrorMessage,
	)

	return err
}

// GetLLMUsage retrieves LLM usage records with optional filtering
func GetLLMUsage(ctx context.Context, filter LLMUsageFilter) ([]LLMUsageRecord, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
	SELECT id, timestamp, request_id, user, provider, model, request_type,
	       prompt_tokens, completion_tokens, total_tokens, tools_called,
	       request_duration_ms, success, error_message
	FROM llm_usage
	WHERE 1=1
	`
	args := []interface{}{}

	if filter.User != "" {
		query += " AND user = ?"
		args = append(args, filter.User)
	}
	if filter.Model != "" {
		query += " AND model = ?"
		args = append(args, filter.Model)
	}
	if filter.Provider != "" {
		query += " AND provider = ?"
		args = append(args, filter.Provider)
	}
	if !filter.StartTime.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, filter.EndTime)
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	} else {
		query += " LIMIT 100"
	}

	rows, err := DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []LLMUsageRecord
	for rows.Next() {
		var r LLMUsageRecord
		var success int
		var timestamp string

		err := rows.Scan(
			&r.ID, &timestamp, &r.RequestID, &r.User, &r.Provider, &r.Model, &r.RequestType,
			&r.PromptTokens, &r.CompletionTokens, &r.TotalTokens, &r.ToolsCalled,
			&r.RequestDurationMs, &success, &r.ErrorMessage,
		)
		if err != nil {
			continue
		}

		r.Timestamp, _ = time.Parse("2006-01-02 15:04:05", timestamp)
		r.Success = success == 1
		records = append(records, r)
	}

	return records, nil
}

// GetLLMUsageStats retrieves aggregated statistics for LLM usage
func GetLLMUsageStats(ctx context.Context, filter LLMUsageFilter) (*LLMUsageStats, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	stats := &LLMUsageStats{
		ByModel: make(map[string]ModelUsageStats),
		ByUser:  make(map[string]int64),
	}

	// Build base where clause
	whereClause := "1=1"
	args := []interface{}{}

	if filter.User != "" {
		whereClause += " AND user = ?"
		args = append(args, filter.User)
	}
	if filter.Provider != "" {
		whereClause += " AND provider = ?"
		args = append(args, filter.Provider)
	}
	if !filter.StartTime.IsZero() {
		whereClause += " AND timestamp >= ?"
		args = append(args, filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		whereClause += " AND timestamp <= ?"
		args = append(args, filter.EndTime)
	}

	// Get overall stats
	query := fmt.Sprintf(`
	SELECT
		COUNT(*) as total_requests,
		COALESCE(SUM(prompt_tokens), 0) as total_prompt_tokens,
		COALESCE(SUM(completion_tokens), 0) as total_comp_tokens,
		COALESCE(SUM(total_tokens), 0) as total_tokens,
		COALESCE(SUM(tools_called), 0) as total_tools_called,
		COALESCE(AVG(request_duration_ms), 0) as avg_duration,
		COALESCE(AVG(success), 1) as success_rate
	FROM llm_usage
	WHERE %s
	`, whereClause)

	row := DB.QueryRowContext(ctx, query, args...)
	err := row.Scan(
		&stats.TotalRequests,
		&stats.TotalPromptTokens,
		&stats.TotalCompTokens,
		&stats.TotalTokens,
		&stats.TotalToolsCalled,
		&stats.AvgRequestDuration,
		&stats.SuccessRate,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get stats by model
	modelQuery := fmt.Sprintf(`
	SELECT model,
	       COUNT(*) as requests,
	       COALESCE(SUM(total_tokens), 0) as total_tokens,
	       COALESCE(SUM(prompt_tokens), 0) as prompt_tokens,
	       COALESCE(SUM(completion_tokens), 0) as comp_tokens
	FROM llm_usage
	WHERE %s
	GROUP BY model
	`, whereClause)

	modelRows, err := DB.QueryContext(ctx, modelQuery, args...)
	if err != nil {
		return nil, err
	}
	defer modelRows.Close()

	for modelRows.Next() {
		var model string
		var ms ModelUsageStats
		if err := modelRows.Scan(&model, &ms.Requests, &ms.TotalTokens, &ms.PromptTokens, &ms.CompTokens); err != nil {
			continue
		}
		stats.ByModel[model] = ms
	}

	// Get stats by user
	userQuery := fmt.Sprintf(`
	SELECT user, COUNT(*) as requests
	FROM llm_usage
	WHERE %s
	GROUP BY user
	ORDER BY requests DESC
	LIMIT 20
	`, whereClause)

	userRows, err := DB.QueryContext(ctx, userQuery, args...)
	if err != nil {
		return nil, err
	}
	defer userRows.Close()

	for userRows.Next() {
		var user string
		var count int64
		if err := userRows.Scan(&user, &count); err != nil {
			continue
		}
		stats.ByUser[user] = count
	}

	// Get hourly time series (last 24 hours)
	timeSeriesQuery := fmt.Sprintf(`
	SELECT
		strftime('%%Y-%%m-%%d %%H:00:00', timestamp) as hour,
		COUNT(*) as requests,
		COALESCE(SUM(total_tokens), 0) as total_tokens,
		COALESCE(SUM(prompt_tokens), 0) as prompt_tokens,
		COALESCE(SUM(completion_tokens), 0) as comp_tokens
	FROM llm_usage
	WHERE %s AND timestamp >= datetime('now', '-24 hours')
	GROUP BY hour
	ORDER BY hour ASC
	`, whereClause)

	tsRows, err := DB.QueryContext(ctx, timeSeriesQuery, args...)
	if err != nil {
		return nil, err
	}
	defer tsRows.Close()

	for tsRows.Next() {
		var ts TimeSeriesPoint
		var hourStr string
		if err := tsRows.Scan(&hourStr, &ts.Requests, &ts.TotalTokens, &ts.PromptTokens, &ts.CompTokens); err != nil {
			continue
		}
		ts.Timestamp, _ = time.Parse("2006-01-02 15:04:05", hourStr)
		stats.TimeSeriesHourly = append(stats.TimeSeriesHourly, ts)
	}

	return stats, nil
}

// CleanupOldLLMUsage removes LLM usage records older than the specified days
func CleanupOldLLMUsage(days int) (int64, error) {
	if DB == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	query := `DELETE FROM llm_usage WHERE timestamp < datetime('now', ?)`
	result, err := DB.Exec(query, fmt.Sprintf("-%d days", days))
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}
