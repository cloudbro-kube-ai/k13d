package web

import (
	"context"
	"time"
)

func (rg *ReportGenerator) generateMetricsHistory(ctx context.Context) *MetricsHistory {
	if rg.server.metricsCollector == nil {
		return nil
	}

	store := rg.server.metricsCollector.GetStore()
	contextName, _ := rg.server.k8sClient.GetCurrentContext()

	end := time.Now()
	start := end.Add(-24 * time.Hour)

	// Get cluster metrics for the last 24 hours
	metrics, err := store.GetClusterMetrics(ctx, contextName, start, end, 100)
	if err != nil || len(metrics) == 0 {
		return nil
	}

	history := &MetricsHistory{
		Period:     "24h",
		DataPoints: len(metrics),
	}

	var totalCPU, totalMem int64
	var totalPods float64
	var maxCPU, maxMem int64
	var maxPods int

	// Convert to data points (reverse order to chronological)
	for i := len(metrics) - 1; i >= 0; i-- {
		m := metrics[i]
		point := ClusterMetricPoint{
			Timestamp:   m.Timestamp.Format("2006-01-02 15:04"),
			CPUUsage:    m.UsedCPUMillis,
			MemoryUsage: m.UsedMemoryMB,
			RunningPods: m.RunningPods,
			ReadyNodes:  m.ReadyNodes,
		}
		history.ClusterMetrics = append(history.ClusterMetrics, point)

		// Calculate summary stats
		totalCPU += m.UsedCPUMillis
		totalMem += m.UsedMemoryMB
		totalPods += float64(m.RunningPods)

		if m.UsedCPUMillis > maxCPU {
			maxCPU = m.UsedCPUMillis
		}
		if m.UsedMemoryMB > maxMem {
			maxMem = m.UsedMemoryMB
		}
		if m.RunningPods > maxPods {
			maxPods = m.RunningPods
		}
	}

	if len(metrics) > 0 {
		history.Summary = MetricsHistorySummary{
			AvgCPUUsage:    totalCPU / int64(len(metrics)),
			MaxCPUUsage:    maxCPU,
			AvgMemoryUsage: totalMem / int64(len(metrics)),
			MaxMemoryUsage: maxMem,
			AvgRunningPods: totalPods / float64(len(metrics)),
			MaxRunningPods: maxPods,
		}
	}

	return history
}

// generateSecurityScan runs security scanning and returns results for reports
