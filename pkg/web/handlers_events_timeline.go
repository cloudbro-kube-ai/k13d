package web

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"time"
)

// EventTimeWindow represents events grouped by a time window
type EventTimeWindow struct {
	Timestamp    time.Time `json:"timestamp"`
	NormalCount  int       `json:"normalCount"`
	WarningCount int       `json:"warningCount"`
	Events       []EventSummary `json:"events,omitempty"`
}

// EventSummary is a condensed event for timeline display
type EventSummary struct {
	Type      string    `json:"type"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	Object    string    `json:"object"` // Kind/Name
	Count     int32     `json:"count"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
}

// EventTimelineResponse is the response for the event timeline endpoint
type EventTimelineResponse struct {
	Windows      []EventTimeWindow `json:"windows"`
	TotalNormal  int               `json:"totalNormal"`
	TotalWarning int               `json:"totalWarning"`
	Namespace    string            `json:"namespace"`
	Hours        int               `json:"hours"`
}

func (s *Server) handleEventTimeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	namespace := r.URL.Query().Get("namespace")
	hoursStr := r.URL.Query().Get("hours")
	hours := 24
	if hoursStr != "" {
		if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 && h <= 168 { // max 7 days
			hours = h
		}
	}

	ctx := r.Context()

	events, err := s.k8sClient.ListEvents(ctx, namespace)
	if err != nil {
		http.Error(w, "Failed to list events: "+err.Error(), http.StatusInternalServerError)
		return
	}

	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	windowSize := time.Hour
	if hours <= 2 {
		windowSize = time.Minute
	} else if hours <= 6 {
		windowSize = 10 * time.Minute
	}

	// Group events by time windows
	windowMap := make(map[int64]*EventTimeWindow)
	totalNormal := 0
	totalWarning := 0

	for _, event := range events {
		eventTime := event.LastTimestamp.Time
		if eventTime.IsZero() {
			eventTime = event.CreationTimestamp.Time
		}

		// Skip events outside our time range
		if eventTime.Before(cutoff) {
			continue
		}

		// Calculate window key (truncate to window size)
		windowKey := eventTime.Truncate(windowSize).Unix()

		tw, ok := windowMap[windowKey]
		if !ok {
			tw = &EventTimeWindow{
				Timestamp: time.Unix(windowKey, 0),
			}
			windowMap[windowKey] = tw
		}

		if event.Type == "Warning" {
			tw.WarningCount++
			totalWarning++
		} else {
			tw.NormalCount++
			totalNormal++
		}

		tw.Events = append(tw.Events, EventSummary{
			Type:      event.Type,
			Reason:    event.Reason,
			Message:   event.Message,
			Object:    event.InvolvedObject.Kind + "/" + event.InvolvedObject.Name,
			Count:     event.Count,
			FirstSeen: event.FirstTimestamp.Time,
			LastSeen:  event.LastTimestamp.Time,
		})
	}

	// Convert map to sorted slice
	var windows []EventTimeWindow
	for _, tw := range windowMap {
		windows = append(windows, *tw)
	}
	sort.Slice(windows, func(i, j int) bool {
		return windows[i].Timestamp.Before(windows[j].Timestamp)
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(EventTimelineResponse{
		Windows:      windows,
		TotalNormal:  totalNormal,
		TotalWarning: totalWarning,
		Namespace:    namespace,
		Hours:        hours,
	})
}
