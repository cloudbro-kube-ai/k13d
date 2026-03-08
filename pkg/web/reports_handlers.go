package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"github.com/cloudbro-kube-ai/k13d/pkg/db"
)

func (rg *ReportGenerator) HandleReports(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("X-Username")
	if username == "" {
		username = "anonymous"
	}

	format := r.URL.Query().Get("format") // json, csv, html
	includeAI := r.URL.Query().Get("ai") == "true"
	download := r.URL.Query().Get("download") == "true" // Force download (vs preview)
	sections := ParseSections(r.URL.Query().Get("sections"))

	switch r.Method {
	case http.MethodGet:
		// Generate report with selected sections
		report, err := rg.GenerateReport(r.Context(), username, sections)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Add AI analysis if requested
		if includeAI {
			analysis, err := rg.GenerateAIAnalysis(r.Context(), report)
			if err == nil {
				report.AIAnalysis = analysis
			}
		}

		// Record audit
		_ = db.RecordAudit(db.AuditEntry{
			User:     username,
			Action:   "generate_report",
			Resource: "cluster",
			Details:  fmt.Sprintf("Format: %s, AI: %v, Download: %v", format, includeAI, download),
		})

		// Return in requested format
		switch format {
		case "csv":
			csvData, err := rg.ExportToCSV(report)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			if download {
				w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=k13d-report-%s.csv", time.Now().Format("20060102-150405")))
			}
			_, _ = w.Write(csvData)

		case "html":
			htmlData := rg.ExportToHTML(report)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if download {
				w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=k13d-report-%s.html", time.Now().Format("20060102-150405")))
			}
			_, _ = w.Write([]byte(htmlData))

		default: // json
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(report)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleReportPreview handles report preview in a new window
func (rg *ReportGenerator) HandleReportPreview(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("X-Username")
	if username == "" {
		username = "anonymous"
	}

	includeAI := r.URL.Query().Get("ai") == "true"
	sections := ParseSections(r.URL.Query().Get("sections"))

	// Generate report with selected sections
	report, err := rg.GenerateReport(r.Context(), username, sections)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add AI analysis if requested
	if includeAI {
		analysis, err := rg.GenerateAIAnalysis(r.Context(), report)
		if err == nil {
			report.AIAnalysis = analysis
		}
	}

	// Record audit
	_ = db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   "preview_report",
		Resource: "cluster",
		Details:  fmt.Sprintf("AI: %v, Sections: %s", includeAI, r.URL.Query().Get("sections")),
	})

	// Return HTML for preview (no Content-Disposition header)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	htmlData := rg.ExportToHTML(report)
	_, _ = w.Write([]byte(htmlData))
}
