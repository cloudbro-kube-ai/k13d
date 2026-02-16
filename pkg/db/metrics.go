package db

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// MetricsStore handles time-series metrics storage
type MetricsStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// ClusterMetrics represents a snapshot of cluster metrics at a point in time
type ClusterMetrics struct {
	ID               int64     `json:"id"`
	Timestamp        time.Time `json:"timestamp"`
	Context          string    `json:"context"`           // Kubernetes context name
	TotalNodes       int       `json:"total_nodes"`       // Total nodes
	ReadyNodes       int       `json:"ready_nodes"`       // Ready nodes
	TotalPods        int       `json:"total_pods"`        // Total pods
	RunningPods      int       `json:"running_pods"`      // Running pods
	PendingPods      int       `json:"pending_pods"`      // Pending pods
	FailedPods       int       `json:"failed_pods"`       // Failed pods
	TotalCPUMillis   int64     `json:"total_cpu_millis"`  // Total CPU in millicores
	UsedCPUMillis    int64     `json:"used_cpu_millis"`   // Used CPU in millicores
	TotalMemoryMB    int64     `json:"total_memory_mb"`   // Total memory in MB
	UsedMemoryMB     int64     `json:"used_memory_mb"`    // Used memory in MB
	TotalDeployments int       `json:"total_deployments"` // Total deployments
	ReadyDeployments int       `json:"ready_deployments"` // Ready deployments
	Namespace        string    `json:"namespace"`         // Namespace filter (empty for all)
}

// NodeMetrics represents a snapshot of node metrics
type NodeMetrics struct {
	ID            int64     `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	Context       string    `json:"context"`
	NodeName      string    `json:"node_name"`
	CPUMillis     int64     `json:"cpu_millis"`
	MemoryMB      int64     `json:"memory_mb"`
	CPUCapacity   int64     `json:"cpu_capacity"` // CPU capacity in millicores
	MemCapacity   int64     `json:"mem_capacity"` // Memory capacity in MB
	PodCount      int       `json:"pod_count"`    // Number of pods on this node
	IsReady       bool      `json:"is_ready"`
	IsSchedulable bool      `json:"is_schedulable"`
}

// PodMetrics represents a snapshot of pod metrics
type PodMetrics struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Context   string    `json:"context"`
	Namespace string    `json:"namespace"`
	PodName   string    `json:"pod_name"`
	CPUMillis int64     `json:"cpu_millis"`
	MemoryMB  int64     `json:"memory_mb"`
	Status    string    `json:"status"`
	Restarts  int       `json:"restarts"`
}

// NewMetricsStore creates a new metrics store
func NewMetricsStore() (*MetricsStore, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	store := &MetricsStore{db: DB}
	if err := store.createMetricsTables(); err != nil {
		return nil, err
	}
	return store, nil
}

// createMetricsTables creates the metrics tables if they don't exist
func (s *MetricsStore) createMetricsTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS cluster_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			context TEXT NOT NULL,
			total_nodes INTEGER DEFAULT 0,
			ready_nodes INTEGER DEFAULT 0,
			total_pods INTEGER DEFAULT 0,
			running_pods INTEGER DEFAULT 0,
			pending_pods INTEGER DEFAULT 0,
			failed_pods INTEGER DEFAULT 0,
			total_cpu_millis INTEGER DEFAULT 0,
			used_cpu_millis INTEGER DEFAULT 0,
			total_memory_mb INTEGER DEFAULT 0,
			used_memory_mb INTEGER DEFAULT 0,
			total_deployments INTEGER DEFAULT 0,
			ready_deployments INTEGER DEFAULT 0,
			namespace TEXT DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS node_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			context TEXT NOT NULL,
			node_name TEXT NOT NULL,
			cpu_millis INTEGER DEFAULT 0,
			memory_mb INTEGER DEFAULT 0,
			cpu_capacity INTEGER DEFAULT 0,
			mem_capacity INTEGER DEFAULT 0,
			pod_count INTEGER DEFAULT 0,
			is_ready INTEGER DEFAULT 1,
			is_schedulable INTEGER DEFAULT 1
		);`,
		`CREATE TABLE IF NOT EXISTS pod_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			context TEXT NOT NULL,
			namespace TEXT NOT NULL,
			pod_name TEXT NOT NULL,
			cpu_millis INTEGER DEFAULT 0,
			memory_mb INTEGER DEFAULT 0,
			status TEXT DEFAULT '',
			restarts INTEGER DEFAULT 0
		);`,
		// Indexes for efficient queries
		`CREATE INDEX IF NOT EXISTS idx_cluster_metrics_timestamp ON cluster_metrics(timestamp DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_cluster_metrics_context ON cluster_metrics(context);`,
		`CREATE INDEX IF NOT EXISTS idx_node_metrics_timestamp ON node_metrics(timestamp DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_node_metrics_node ON node_metrics(node_name);`,
		`CREATE INDEX IF NOT EXISTS idx_pod_metrics_timestamp ON pod_metrics(timestamp DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_pod_metrics_namespace ON pod_metrics(namespace);`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("failed to create metrics table: %w", err)
		}
	}
	return nil
}

// SaveClusterMetrics saves cluster-level metrics
func (s *MetricsStore) SaveClusterMetrics(ctx context.Context, m *ClusterMetrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `INSERT INTO cluster_metrics
		(timestamp, context, total_nodes, ready_nodes, total_pods, running_pods, pending_pods, failed_pods,
		 total_cpu_millis, used_cpu_millis, total_memory_mb, used_memory_mb, total_deployments, ready_deployments, namespace)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	ts := m.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}

	_, err := s.db.ExecContext(ctx, query,
		ts, m.Context, m.TotalNodes, m.ReadyNodes, m.TotalPods, m.RunningPods, m.PendingPods, m.FailedPods,
		m.TotalCPUMillis, m.UsedCPUMillis, m.TotalMemoryMB, m.UsedMemoryMB, m.TotalDeployments, m.ReadyDeployments, m.Namespace)
	return err
}

// SaveNodeMetrics saves node-level metrics
func (s *MetricsStore) SaveNodeMetrics(ctx context.Context, m *NodeMetrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `INSERT INTO node_metrics
		(timestamp, context, node_name, cpu_millis, memory_mb, cpu_capacity, mem_capacity, pod_count, is_ready, is_schedulable)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	ts := m.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}

	isReady := 0
	if m.IsReady {
		isReady = 1
	}
	isSchedulable := 0
	if m.IsSchedulable {
		isSchedulable = 1
	}

	_, err := s.db.ExecContext(ctx, query,
		ts, m.Context, m.NodeName, m.CPUMillis, m.MemoryMB, m.CPUCapacity, m.MemCapacity, m.PodCount, isReady, isSchedulable)
	return err
}

// SavePodMetrics saves pod-level metrics
func (s *MetricsStore) SavePodMetrics(ctx context.Context, m *PodMetrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `INSERT INTO pod_metrics
		(timestamp, context, namespace, pod_name, cpu_millis, memory_mb, status, restarts)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	ts := m.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}

	_, err := s.db.ExecContext(ctx, query,
		ts, m.Context, m.Namespace, m.PodName, m.CPUMillis, m.MemoryMB, m.Status, m.Restarts)
	return err
}

// SaveNodeMetricsBatch saves multiple node metrics in a single transaction.
func (s *MetricsStore) SaveNodeMetricsBatch(ctx context.Context, metrics []NodeMetrics) error {
	if len(metrics) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO node_metrics
		(timestamp, context, node_name, cpu_millis, memory_mb, cpu_capacity, mem_capacity, pod_count, is_ready, is_schedulable)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	for _, m := range metrics {
		ts := m.Timestamp
		if ts.IsZero() {
			ts = time.Now()
		}
		isReady := 0
		if m.IsReady {
			isReady = 1
		}
		isSchedulable := 0
		if m.IsSchedulable {
			isSchedulable = 1
		}
		if _, err := stmt.ExecContext(ctx, ts, m.Context, m.NodeName, m.CPUMillis, m.MemoryMB,
			m.CPUCapacity, m.MemCapacity, m.PodCount, isReady, isSchedulable); err != nil {
			return fmt.Errorf("insert node %s: %w", m.NodeName, err)
		}
	}

	return tx.Commit()
}

// SavePodMetricsBatch saves multiple pod metrics in a single transaction.
func (s *MetricsStore) SavePodMetricsBatch(ctx context.Context, metrics []PodMetrics) error {
	if len(metrics) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO pod_metrics
		(timestamp, context, namespace, pod_name, cpu_millis, memory_mb, status, restarts)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	for _, m := range metrics {
		ts := m.Timestamp
		if ts.IsZero() {
			ts = time.Now()
		}
		if _, err := stmt.ExecContext(ctx, ts, m.Context, m.Namespace, m.PodName,
			m.CPUMillis, m.MemoryMB, m.Status, m.Restarts); err != nil {
			return fmt.Errorf("insert pod %s/%s: %w", m.Namespace, m.PodName, err)
		}
	}

	return tx.Commit()
}

// GetClusterMetrics retrieves cluster metrics for a time range
func (s *MetricsStore) GetClusterMetrics(ctx context.Context, contextName string, start, end time.Time, limit int) ([]ClusterMetrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, timestamp, context, total_nodes, ready_nodes, total_pods, running_pods, pending_pods, failed_pods,
		total_cpu_millis, used_cpu_millis, total_memory_mb, used_memory_mb, total_deployments, ready_deployments, namespace
		FROM cluster_metrics WHERE context = ? AND timestamp BETWEEN ? AND ? ORDER BY timestamp DESC LIMIT ?`

	rows, err := s.db.QueryContext(ctx, query, contextName, start, end, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []ClusterMetrics
	for rows.Next() {
		var m ClusterMetrics
		err := rows.Scan(&m.ID, &m.Timestamp, &m.Context, &m.TotalNodes, &m.ReadyNodes, &m.TotalPods, &m.RunningPods,
			&m.PendingPods, &m.FailedPods, &m.TotalCPUMillis, &m.UsedCPUMillis, &m.TotalMemoryMB, &m.UsedMemoryMB,
			&m.TotalDeployments, &m.ReadyDeployments, &m.Namespace)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return metrics, nil
}

// GetNodeMetricsHistory retrieves node metrics for a time range
func (s *MetricsStore) GetNodeMetricsHistory(ctx context.Context, contextName, nodeName string, start, end time.Time, limit int) ([]NodeMetrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, timestamp, context, node_name, cpu_millis, memory_mb, cpu_capacity, mem_capacity, pod_count, is_ready, is_schedulable
		FROM node_metrics WHERE context = ? AND node_name = ? AND timestamp BETWEEN ? AND ? ORDER BY timestamp DESC LIMIT ?`

	rows, err := s.db.QueryContext(ctx, query, contextName, nodeName, start, end, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []NodeMetrics
	for rows.Next() {
		var m NodeMetrics
		var isReady, isSchedulable int
		err := rows.Scan(&m.ID, &m.Timestamp, &m.Context, &m.NodeName, &m.CPUMillis, &m.MemoryMB,
			&m.CPUCapacity, &m.MemCapacity, &m.PodCount, &isReady, &isSchedulable)
		if err != nil {
			return nil, err
		}
		m.IsReady = isReady == 1
		m.IsSchedulable = isSchedulable == 1
		metrics = append(metrics, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return metrics, nil
}

// GetPodMetricsHistory retrieves pod metrics for a time range
func (s *MetricsStore) GetPodMetricsHistory(ctx context.Context, contextName, namespace, podName string, start, end time.Time, limit int) ([]PodMetrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, timestamp, context, namespace, pod_name, cpu_millis, memory_mb, status, restarts
		FROM pod_metrics WHERE context = ? AND namespace = ? AND pod_name = ? AND timestamp BETWEEN ? AND ? ORDER BY timestamp DESC LIMIT ?`

	rows, err := s.db.QueryContext(ctx, query, contextName, namespace, podName, start, end, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []PodMetrics
	for rows.Next() {
		var m PodMetrics
		err := rows.Scan(&m.ID, &m.Timestamp, &m.Context, &m.Namespace, &m.PodName, &m.CPUMillis, &m.MemoryMB, &m.Status, &m.Restarts)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return metrics, nil
}

// GetLatestClusterMetrics retrieves the most recent cluster metrics
func (s *MetricsStore) GetLatestClusterMetrics(ctx context.Context, contextName string) (*ClusterMetrics, error) {
	metrics, err := s.GetClusterMetrics(ctx, contextName, time.Now().Add(-24*time.Hour), time.Now(), 1)
	if err != nil {
		return nil, err
	}
	if len(metrics) == 0 {
		return nil, nil
	}
	return &metrics[0], nil
}

// validMetricsTables is the set of known metrics table names, used to prevent SQL injection
// when constructing cleanup queries with fmt.Sprintf.
var validMetricsTables = map[string]bool{
	"cluster_metrics": true,
	"node_metrics":    true,
	"pod_metrics":     true,
}

// CleanupOldMetrics removes metrics older than the specified duration
func (s *MetricsStore) CleanupOldMetrics(ctx context.Context, retention time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-retention)

	for _, table := range []string{"cluster_metrics", "node_metrics", "pod_metrics"} {
		if !validMetricsTables[table] {
			return fmt.Errorf("unknown metrics table: %s", table)
		}
		query := fmt.Sprintf("DELETE FROM %s WHERE timestamp < ?", table)
		if _, err := s.db.ExecContext(ctx, query, cutoff); err != nil {
			return fmt.Errorf("failed to cleanup %s: %w", table, err)
		}
	}
	return nil
}

// GetMetricsSummary returns a summary of stored metrics
func (s *MetricsStore) GetMetricsSummary(ctx context.Context, contextName string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := make(map[string]interface{})

	// Count cluster metrics
	var clusterCount int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cluster_metrics WHERE context = ?", contextName).Scan(&clusterCount)
	if err != nil {
		return nil, err
	}
	summary["cluster_metrics_count"] = clusterCount

	// Get time range
	var minTime, maxTime sql.NullTime
	err = s.db.QueryRowContext(ctx, "SELECT MIN(timestamp), MAX(timestamp) FROM cluster_metrics WHERE context = ?", contextName).Scan(&minTime, &maxTime)
	if err != nil {
		return nil, err
	}
	if minTime.Valid {
		summary["oldest_metric"] = minTime.Time
		summary["newest_metric"] = maxTime.Time
	}

	// Count node metrics
	var nodeCount int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT node_name) FROM node_metrics WHERE context = ?", contextName).Scan(&nodeCount)
	if err != nil {
		return nil, err
	}
	summary["unique_nodes_tracked"] = nodeCount

	// Count pod metrics
	var podCount int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT pod_name) FROM pod_metrics WHERE context = ?", contextName).Scan(&podCount)
	if err != nil {
		return nil, err
	}
	summary["unique_pods_tracked"] = podCount

	return summary, nil
}

// validIntervalFormats maps allowed interval values to SQLite strftime formats.
// Only these values are accepted to prevent SQL injection via the interval parameter.
var validIntervalFormats = map[string]string{
	"hour": "%Y-%m-%d %H:00:00",
	"day":  "%Y-%m-%d 00:00:00",
}

// GetAggregatedClusterMetrics returns aggregated metrics (hourly/daily averages).
// TODO: Uses SQLite-specific strftime(); add DB-specific aggregation for Postgres/MySQL.
func (s *MetricsStore) GetAggregatedClusterMetrics(ctx context.Context, contextName string, start, end time.Time, interval string) ([]map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Validate interval against whitelist
	groupFormat, ok := validIntervalFormats[interval]
	if !ok {
		groupFormat = validIntervalFormats["hour"]
	}

	query := fmt.Sprintf(`SELECT
		strftime('%s', timestamp) as period,
		AVG(total_nodes) as avg_total_nodes,
		AVG(ready_nodes) as avg_ready_nodes,
		AVG(total_pods) as avg_total_pods,
		AVG(running_pods) as avg_running_pods,
		AVG(used_cpu_millis) as avg_cpu_millis,
		AVG(used_memory_mb) as avg_memory_mb,
		MAX(used_cpu_millis) as max_cpu_millis,
		MAX(used_memory_mb) as max_memory_mb
		FROM cluster_metrics
		WHERE context = ? AND timestamp BETWEEN ? AND ?
		GROUP BY period
		ORDER BY period DESC`, groupFormat)

	rows, err := s.db.QueryContext(ctx, query, contextName, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var period string
		var avgNodes, avgReadyNodes, avgPods, avgRunningPods, avgCPU, avgMem, maxCPU, maxMem float64
		err := rows.Scan(&period, &avgNodes, &avgReadyNodes, &avgPods, &avgRunningPods, &avgCPU, &avgMem, &maxCPU, &maxMem)
		if err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"period":           period,
			"avg_total_nodes":  avgNodes,
			"avg_ready_nodes":  avgReadyNodes,
			"avg_total_pods":   avgPods,
			"avg_running_pods": avgRunningPods,
			"avg_cpu_millis":   avgCPU,
			"avg_memory_mb":    avgMem,
			"max_cpu_millis":   maxCPU,
			"max_memory_mb":    maxMem,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
