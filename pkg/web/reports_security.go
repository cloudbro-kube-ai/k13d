package web

import (
	"context"
)

func (rg *ReportGenerator) generateSecurityScan(ctx context.Context) *SecurityScanReport {
	if rg.server.securityScanner == nil {
		return nil
	}

	// Run a quick scan (without image scanning for speed)
	scanResult, err := rg.server.securityScanner.QuickScan(ctx, "")
	if err != nil {
		return nil
	}

	report := &SecurityScanReport{
		ScanTime:     scanResult.ScanTime,
		Duration:     scanResult.Duration,
		OverallScore: scanResult.OverallScore,
		RiskLevel:    scanResult.RiskLevel,
		ToolsUsed:    []string{"k13d-security-scanner"},
	}

	// Add tools info
	if rg.server.securityScanner.TrivyAvailable() {
		report.ToolsUsed = append(report.ToolsUsed, "trivy")
	}
	if rg.server.securityScanner.KubeBenchAvailable() {
		report.ToolsUsed = append(report.ToolsUsed, "kube-bench")
	}

	// Convert image vulnerabilities
	if scanResult.ImageVulns != nil {
		report.ImageVulnSummary = &ImageVulnerabilitySummary{
			TotalImages:      scanResult.ImageVulns.TotalImages,
			ScannedImages:    scanResult.ImageVulns.ScannedImages,
			VulnerableImages: scanResult.ImageVulns.VulnerableImages,
			CriticalCount:    scanResult.ImageVulns.CriticalCount,
			HighCount:        scanResult.ImageVulns.HighCount,
			MediumCount:      scanResult.ImageVulns.MediumCount,
			LowCount:         scanResult.ImageVulns.LowCount,
		}
	}

	// Convert pod security issues (limit to 20)
	for i, issue := range scanResult.PodSecurityIssues {
		if i >= 20 {
			break
		}
		report.PodSecurityIssues = append(report.PodSecurityIssues, PodSecurityIssueReport{
			Namespace:   issue.Namespace,
			Pod:         issue.Pod,
			Container:   issue.Container,
			Issue:       issue.Issue,
			Severity:    issue.Severity,
			Remediation: issue.Remediation,
		})
	}

	// Convert RBAC issues (limit to 20)
	for i, issue := range scanResult.RBACIssues {
		if i >= 20 {
			break
		}
		report.RBACIssues = append(report.RBACIssues, RBACIssueReport{
			Kind:        issue.Kind,
			Name:        issue.Name,
			Namespace:   issue.Namespace,
			Issue:       issue.Issue,
			Severity:    issue.Severity,
			Remediation: issue.Remediation,
		})
	}

	// Convert network issues (limit to 20)
	for i, issue := range scanResult.NetworkIssues {
		if i >= 20 {
			break
		}
		report.NetworkIssues = append(report.NetworkIssues, NetworkIssueReport{
			Namespace:   issue.Namespace,
			Resource:    issue.Resource,
			Issue:       issue.Issue,
			Severity:    issue.Severity,
			Remediation: issue.Remediation,
		})
	}

	// Convert CIS benchmark
	if scanResult.CISBenchmark != nil {
		report.CISBenchmark = &CISBenchmarkReport{
			Version:     scanResult.CISBenchmark.Version,
			TotalChecks: scanResult.CISBenchmark.TotalChecks,
			PassCount:   scanResult.CISBenchmark.PassCount,
			FailCount:   scanResult.CISBenchmark.FailCount,
			WarnCount:   scanResult.CISBenchmark.WarnCount,
			Score:       scanResult.CISBenchmark.Score,
		}
	}

	// Convert recommendations
	for _, rec := range scanResult.Recommendations {
		report.Recommendations = append(report.Recommendations, SecurityRecommendationReport{
			Priority:    rec.Priority,
			Category:    rec.Category,
			Title:       rec.Title,
			Description: rec.Description,
			Impact:      rec.Impact,
			Remediation: rec.Remediation,
		})
	}

	return report
}

// generateFullSecurityScan runs a full security scan including Trivy image scanning
func (rg *ReportGenerator) generateFullSecurityScan(ctx context.Context) *SecurityScanReport {
	if rg.server.securityScanner == nil {
		return nil
	}

	// Run full scan (includes Trivy image vulnerability scanning)
	scanResult, err := rg.server.securityScanner.Scan(ctx, "")
	if err != nil {
		// Fall back to quick scan
		return rg.generateSecurityScan(ctx)
	}

	report := &SecurityScanReport{
		ScanTime:     scanResult.ScanTime,
		Duration:     scanResult.Duration,
		OverallScore: scanResult.OverallScore,
		RiskLevel:    scanResult.RiskLevel,
		ToolsUsed:    []string{"k13d-security-scanner"},
	}

	if rg.server.securityScanner.TrivyAvailable() {
		report.ToolsUsed = append(report.ToolsUsed, "trivy")
	}
	if rg.server.securityScanner.KubeBenchAvailable() {
		report.ToolsUsed = append(report.ToolsUsed, "kube-bench")
	}

	if scanResult.ImageVulns != nil {
		report.ImageVulnSummary = &ImageVulnerabilitySummary{
			TotalImages:      scanResult.ImageVulns.TotalImages,
			ScannedImages:    scanResult.ImageVulns.ScannedImages,
			VulnerableImages: scanResult.ImageVulns.VulnerableImages,
			CriticalCount:    scanResult.ImageVulns.CriticalCount,
			HighCount:        scanResult.ImageVulns.HighCount,
			MediumCount:      scanResult.ImageVulns.MediumCount,
			LowCount:         scanResult.ImageVulns.LowCount,
		}
	}

	for i, issue := range scanResult.PodSecurityIssues {
		if i >= 20 {
			break
		}
		report.PodSecurityIssues = append(report.PodSecurityIssues, PodSecurityIssueReport{
			Namespace: issue.Namespace, Pod: issue.Pod, Container: issue.Container,
			Issue: issue.Issue, Severity: issue.Severity, Remediation: issue.Remediation,
		})
	}

	for i, issue := range scanResult.RBACIssues {
		if i >= 20 {
			break
		}
		report.RBACIssues = append(report.RBACIssues, RBACIssueReport{
			Kind: issue.Kind, Name: issue.Name, Namespace: issue.Namespace,
			Issue: issue.Issue, Severity: issue.Severity, Remediation: issue.Remediation,
		})
	}

	for i, issue := range scanResult.NetworkIssues {
		if i >= 20 {
			break
		}
		report.NetworkIssues = append(report.NetworkIssues, NetworkIssueReport{
			Namespace: issue.Namespace, Resource: issue.Resource,
			Issue: issue.Issue, Severity: issue.Severity, Remediation: issue.Remediation,
		})
	}

	if scanResult.CISBenchmark != nil {
		report.CISBenchmark = &CISBenchmarkReport{
			Version: scanResult.CISBenchmark.Version, TotalChecks: scanResult.CISBenchmark.TotalChecks,
			PassCount: scanResult.CISBenchmark.PassCount, FailCount: scanResult.CISBenchmark.FailCount,
			WarnCount: scanResult.CISBenchmark.WarnCount, Score: scanResult.CISBenchmark.Score,
		}
	}

	for _, rec := range scanResult.Recommendations {
		report.Recommendations = append(report.Recommendations, SecurityRecommendationReport{
			Priority: rec.Priority, Category: rec.Category, Title: rec.Title,
			Description: rec.Description, Impact: rec.Impact, Remediation: rec.Remediation,
		})
	}

	return report
}

// generateFinOpsAnalysis analyzes cost and resource efficiency
