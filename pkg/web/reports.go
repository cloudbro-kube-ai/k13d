package web

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/db"
	corev1 "k8s.io/api/core/v1"
)

// ComprehensiveReport contains all cluster information for export
type ComprehensiveReport struct {
	GeneratedAt      time.Time           `json:"generated_at"`
	GeneratedBy      string              `json:"generated_by"`
	ClusterInfo      ClusterInfo         `json:"cluster_info"`
	NodeSummary      NodeSummary         `json:"node_summary"`
	Nodes            []NodeInfo          `json:"nodes"`
	NamespaceSummary NamespaceSummary    `json:"namespace_summary"`
	Namespaces       []NamespaceInfo     `json:"namespaces"`
	Workloads        WorkloadSummary     `json:"workloads"`
	Pods             []PodInfo           `json:"pods"`
	Deployments      []DeploymentInfo    `json:"deployments"`
	Services         []ServiceInfo       `json:"services"`
	SecurityInfo     SecurityInfo        `json:"security_info"`
	SecurityScan     *SecurityScanReport `json:"security_scan,omitempty"`
	FinOpsAnalysis   FinOpsAnalysis      `json:"finops_analysis"`
	Images           []ImageInfo         `json:"images"`
	Events           []EventInfo         `json:"events"`
	MetricsHistory   *MetricsHistory     `json:"metrics_history,omitempty"`
	AIAnalysis       string              `json:"ai_analysis,omitempty"`
	HealthScore      float64             `json:"health_score"`
}

// SecurityScanReport contains results from security scanning tools
type SecurityScanReport struct {
	ScanTime          time.Time                      `json:"scan_time"`
	Duration          string                         `json:"duration"`
	OverallScore      float64                        `json:"overall_score"`
	RiskLevel         string                         `json:"risk_level"`
	ToolsUsed         []string                       `json:"tools_used"`
	ImageVulnSummary  *ImageVulnerabilitySummary     `json:"image_vulnerabilities,omitempty"`
	PodSecurityIssues []PodSecurityIssueReport       `json:"pod_security_issues,omitempty"`
	RBACIssues        []RBACIssueReport              `json:"rbac_issues,omitempty"`
	NetworkIssues     []NetworkIssueReport           `json:"network_issues,omitempty"`
	CISBenchmark      *CISBenchmarkReport            `json:"cis_benchmark,omitempty"`
	Recommendations   []SecurityRecommendationReport `json:"recommendations,omitempty"`
}

// ImageVulnerabilitySummary summarizes container image vulnerabilities
type ImageVulnerabilitySummary struct {
	TotalImages      int `json:"total_images"`
	ScannedImages    int `json:"scanned_images"`
	VulnerableImages int `json:"vulnerable_images"`
	CriticalCount    int `json:"critical_count"`
	HighCount        int `json:"high_count"`
	MediumCount      int `json:"medium_count"`
	LowCount         int `json:"low_count"`
}

// PodSecurityIssueReport represents a pod security issue for reports
type PodSecurityIssueReport struct {
	Namespace   string `json:"namespace"`
	Pod         string `json:"pod"`
	Container   string `json:"container,omitempty"`
	Issue       string `json:"issue"`
	Severity    string `json:"severity"`
	Remediation string `json:"remediation"`
}

// RBACIssueReport represents an RBAC issue for reports
type RBACIssueReport struct {
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Namespace   string `json:"namespace,omitempty"`
	Issue       string `json:"issue"`
	Severity    string `json:"severity"`
	Remediation string `json:"remediation"`
}

// NetworkIssueReport represents a network issue for reports
type NetworkIssueReport struct {
	Namespace   string `json:"namespace"`
	Resource    string `json:"resource"`
	Issue       string `json:"issue"`
	Severity    string `json:"severity"`
	Remediation string `json:"remediation"`
}

// CISBenchmarkReport contains CIS benchmark results for reports
type CISBenchmarkReport struct {
	Version     string  `json:"version"`
	TotalChecks int     `json:"total_checks"`
	PassCount   int     `json:"pass_count"`
	FailCount   int     `json:"fail_count"`
	WarnCount   int     `json:"warn_count"`
	Score       float64 `json:"score"`
}

// SecurityRecommendationReport represents a security recommendation
type SecurityRecommendationReport struct {
	Priority    int    `json:"priority"`
	Category    string `json:"category"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
	Remediation string `json:"remediation"`
}

// MetricsHistory contains time-series data for the report
type MetricsHistory struct {
	Period         string                `json:"period"` // e.g., "24h", "7d"
	DataPoints     int                   `json:"data_points"`
	ClusterMetrics []ClusterMetricPoint  `json:"cluster_metrics"`
	Summary        MetricsHistorySummary `json:"summary"`
}

// ClusterMetricPoint is a single point in time-series
type ClusterMetricPoint struct {
	Timestamp   string `json:"timestamp"`
	CPUUsage    int64  `json:"cpu_usage_millis"`
	MemoryUsage int64  `json:"memory_usage_mb"`
	RunningPods int    `json:"running_pods"`
	ReadyNodes  int    `json:"ready_nodes"`
}

// MetricsHistorySummary provides statistical summary of metrics
type MetricsHistorySummary struct {
	AvgCPUUsage    int64   `json:"avg_cpu_usage_millis"`
	MaxCPUUsage    int64   `json:"max_cpu_usage_millis"`
	AvgMemoryUsage int64   `json:"avg_memory_usage_mb"`
	MaxMemoryUsage int64   `json:"max_memory_usage_mb"`
	AvgRunningPods float64 `json:"avg_running_pods"`
	MaxRunningPods int     `json:"max_running_pods"`
}

// FinOpsAnalysis contains cost optimization insights
type FinOpsAnalysis struct {
	TotalEstimatedMonthlyCost float64                   `json:"total_estimated_monthly_cost"`
	CostByNamespace           []NamespaceCost           `json:"cost_by_namespace"`
	ResourceEfficiency        ResourceEfficiencyInfo    `json:"resource_efficiency"`
	CostOptimizations         []CostOptimization        `json:"cost_optimizations"`
	UnderutilizedResources    []UnderutilizedResource   `json:"underutilized_resources"`
	OverprovisionedWorkloads  []OverprovisionedWorkload `json:"overprovisioned_workloads"`
}

// NamespaceCost represents estimated cost per namespace
type NamespaceCost struct {
	Namespace      string  `json:"namespace"`
	PodCount       int     `json:"pod_count"`
	CPURequests    string  `json:"cpu_requests"`
	MemoryRequests string  `json:"memory_requests"`
	EstimatedCost  float64 `json:"estimated_cost"`
	CostPercentage float64 `json:"cost_percentage"`
}

// ResourceEfficiencyInfo contains resource utilization metrics
type ResourceEfficiencyInfo struct {
	TotalCPURequests         string  `json:"total_cpu_requests"`
	TotalCPULimits           string  `json:"total_cpu_limits"`
	TotalMemoryRequests      string  `json:"total_memory_requests"`
	TotalMemoryLimits        string  `json:"total_memory_limits"`
	CPURequestsVsCapacity    float64 `json:"cpu_requests_vs_capacity"`
	MemoryRequestsVsCapacity float64 `json:"memory_requests_vs_capacity"`
	PodsWithoutRequests      int     `json:"pods_without_requests"`
	PodsWithoutLimits        int     `json:"pods_without_limits"`
}

// CostOptimization represents a cost saving recommendation
type CostOptimization struct {
	Category        string  `json:"category"`
	Description     string  `json:"description"`
	Impact          string  `json:"impact"`
	EstimatedSaving float64 `json:"estimated_saving"`
	Priority        string  `json:"priority"` // high, medium, low
}

// UnderutilizedResource represents a resource with low utilization
type UnderutilizedResource struct {
	Name         string  `json:"name"`
	Namespace    string  `json:"namespace"`
	ResourceType string  `json:"resource_type"`
	CPUUsage     float64 `json:"cpu_usage_percent"`
	MemoryUsage  float64 `json:"memory_usage_percent"`
	Suggestion   string  `json:"suggestion"`
}

// OverprovisionedWorkload represents a workload with excessive resources
type OverprovisionedWorkload struct {
	Name              string `json:"name"`
	Namespace         string `json:"namespace"`
	WorkloadType      string `json:"workload_type"`
	CurrentReplicas   int    `json:"current_replicas"`
	SuggestedReplicas int    `json:"suggested_replicas"`
	Reason            string `json:"reason"`
}

type ClusterInfo struct {
	ServerVersion string `json:"server_version"`
	Platform      string `json:"platform"`
	TotalNodes    int    `json:"total_nodes"`
	TotalPods     int    `json:"total_pods"`
}

type NodeSummary struct {
	Total    int `json:"total"`
	Ready    int `json:"ready"`
	NotReady int `json:"not_ready"`
}

type NodeInfo struct {
	Name             string   `json:"name"`
	Status           string   `json:"status"`
	Roles            []string `json:"roles"`
	KubeletVersion   string   `json:"kubelet_version"`
	OS               string   `json:"os"`
	Architecture     string   `json:"architecture"`
	CPUCapacity      string   `json:"cpu_capacity"`
	MemoryCapacity   string   `json:"memory_capacity"`
	PodCapacity      string   `json:"pod_capacity"`
	ContainerRuntime string   `json:"container_runtime"`
	InternalIP       string   `json:"internal_ip"`
	CreationTime     string   `json:"creation_time"`
}

type NamespaceSummary struct {
	Total  int `json:"total"`
	Active int `json:"active"`
}

type NamespaceInfo struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	PodCount     int    `json:"pod_count"`
	DeployCount  int    `json:"deploy_count"`
	ServiceCount int    `json:"service_count"`
	CreationTime string `json:"creation_time"`
}

type WorkloadSummary struct {
	TotalPods        int `json:"total_pods"`
	RunningPods      int `json:"running_pods"`
	PendingPods      int `json:"pending_pods"`
	FailedPods       int `json:"failed_pods"`
	TotalDeployments int `json:"total_deployments"`
	HealthyDeploys   int `json:"healthy_deployments"`
	TotalServices    int `json:"total_services"`
	TotalConfigMaps  int `json:"total_configmaps"`
	TotalSecrets     int `json:"total_secrets"`
}

type PodInfo struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Status    string   `json:"status"`
	Ready     string   `json:"ready"`
	Restarts  int      `json:"restarts"`
	Node      string   `json:"node"`
	IP        string   `json:"ip"`
	Images    []string `json:"images"`
	Age       string   `json:"age"`
}

type DeploymentInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Ready     string `json:"ready"`
	UpToDate  int    `json:"up_to_date"`
	Available int    `json:"available"`
	Strategy  string `json:"strategy"`
	Age       string `json:"age"`
}

type ServiceInfo struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Type       string `json:"type"`
	ClusterIP  string `json:"cluster_ip"`
	ExternalIP string `json:"external_ip"`
	Ports      string `json:"ports"`
	Age        string `json:"age"`
}

type SecurityInfo struct {
	ServiceAccounts     int `json:"service_accounts"`
	Roles               int `json:"roles"`
	RoleBindings        int `json:"role_bindings"`
	ClusterRoles        int `json:"cluster_roles"`
	ClusterRoleBindings int `json:"cluster_role_bindings"`
	Secrets             int `json:"secrets"`
	PrivilegedPods      int `json:"privileged_pods"`
	HostNetworkPods     int `json:"host_network_pods"`
	RootContainers      int `json:"root_containers"`
}

type ImageInfo struct {
	Image      string `json:"image"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	PodCount   int    `json:"pod_count"`
}

type EventInfo struct {
	Type      string `json:"type"`
	Reason    string `json:"reason"`
	Object    string `json:"object"`
	Message   string `json:"message"`
	Count     int    `json:"count"`
	FirstSeen string `json:"first_seen"`
	LastSeen  string `json:"last_seen"`
}

// ReportGenerator handles report generation
type ReportGenerator struct {
	server *Server
}

// NewReportGenerator creates a new report generator
func NewReportGenerator(server *Server) *ReportGenerator {
	return &ReportGenerator{server: server}
}

// ReportSections defines which sections to include in the report.
// If nil or empty, all sections are included (backward compatible).
type ReportSections struct {
	Nodes         bool
	Namespaces    bool
	Workloads     bool // pods, deployments, services, images
	Events        bool
	SecurityBasic bool // pod security, RBAC, network (no Trivy)
	SecurityFull  bool // full scan with Trivy image vulnerability scanning
	FinOps        bool
	Metrics       bool
}

// AllSections returns ReportSections with everything enabled.
func AllSections() *ReportSections {
	return &ReportSections{
		Nodes: true, Namespaces: true, Workloads: true, Events: true,
		SecurityBasic: true, FinOps: true, Metrics: true,
	}
}

// ParseSections parses a comma-separated sections string into ReportSections.
// Returns nil (meaning all sections) if the input is empty.
func ParseSections(s string) *ReportSections {
	if s == "" {
		return nil
	}
	sec := &ReportSections{}
	for _, part := range strings.Split(s, ",") {
		switch strings.TrimSpace(part) {
		case "nodes":
			sec.Nodes = true
		case "namespaces":
			sec.Namespaces = true
		case "workloads":
			sec.Workloads = true
		case "events":
			sec.Events = true
		case "security":
			sec.SecurityBasic = true
		case "security_full":
			sec.SecurityBasic = true
			sec.SecurityFull = true
		case "finops":
			sec.FinOps = true
		case "metrics":
			sec.Metrics = true
		}
	}
	return sec
}

// GenerateComprehensiveReport gathers all cluster data
func (rg *ReportGenerator) GenerateComprehensiveReport(ctx context.Context, username string) (*ComprehensiveReport, error) {
	return rg.GenerateReport(ctx, username, nil)
}

// GenerateReport gathers cluster data for the specified sections.
// If sections is nil, all sections are included.
func (rg *ReportGenerator) GenerateReport(ctx context.Context, username string, sections *ReportSections) (*ComprehensiveReport, error) {
	// nil means all sections
	all := sections == nil
	if all {
		sections = AllSections()
	}
	report := &ComprehensiveReport{
		GeneratedAt: time.Now(),
		GeneratedBy: username,
	}

	// Always get nodes (needed for health score and cluster info)
	nodes, err := rg.server.k8sClient.ListNodes(ctx)
	if err == nil {
		report.NodeSummary.Total = len(nodes)
		for _, node := range nodes {
			info := NodeInfo{
				Name:             node.Name,
				KubeletVersion:   node.Status.NodeInfo.KubeletVersion,
				OS:               node.Status.NodeInfo.OSImage,
				Architecture:     node.Status.NodeInfo.Architecture,
				ContainerRuntime: node.Status.NodeInfo.ContainerRuntimeVersion,
				CPUCapacity:      node.Status.Capacity.Cpu().String(),
				MemoryCapacity:   node.Status.Capacity.Memory().String(),
				PodCapacity:      node.Status.Capacity.Pods().String(),
				CreationTime:     node.CreationTimestamp.Format(time.RFC3339),
			}

			// Get roles
			for label := range node.Labels {
				if strings.HasPrefix(label, "node-role.kubernetes.io/") {
					role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
					info.Roles = append(info.Roles, role)
				}
			}
			if len(info.Roles) == 0 {
				info.Roles = []string{"worker"}
			}

			// Get status
			for _, cond := range node.Status.Conditions {
				if cond.Type == corev1.NodeReady {
					if cond.Status == corev1.ConditionTrue {
						info.Status = "Ready"
						report.NodeSummary.Ready++
					} else {
						info.Status = "NotReady"
						report.NodeSummary.NotReady++
					}
					break
				}
			}

			// Get IP
			for _, addr := range node.Status.Addresses {
				if addr.Type == corev1.NodeInternalIP {
					info.InternalIP = addr.Address
					break
				}
			}

			report.Nodes = append(report.Nodes, info)
		}
	}

	// Get namespaces
	namespaces, err := rg.server.k8sClient.ListNamespaces(ctx)
	if err == nil {
		report.NamespaceSummary.Total = len(namespaces)
		for _, ns := range namespaces {
			info := NamespaceInfo{
				Name:         ns.Name,
				Status:       string(ns.Status.Phase),
				CreationTime: ns.CreationTimestamp.Format(time.RFC3339),
			}

			if ns.Status.Phase == corev1.NamespaceActive {
				report.NamespaceSummary.Active++
			}

			// Count resources in namespace
			pods, _ := rg.server.k8sClient.ListPods(ctx, ns.Name)
			info.PodCount = len(pods)

			deps, _ := rg.server.k8sClient.ListDeployments(ctx, ns.Name)
			info.DeployCount = len(deps)

			svcs, _ := rg.server.k8sClient.ListServices(ctx, ns.Name)
			info.ServiceCount = len(svcs)

			report.Namespaces = append(report.Namespaces, info)
		}
	}

	// Gather workload data
	imageCount := make(map[string]int)

	for _, ns := range namespaces {
		// Pods
		pods, _ := rg.server.k8sClient.ListPods(ctx, ns.Name)
		for _, pod := range pods {
			report.Workloads.TotalPods++

			switch pod.Status.Phase {
			case corev1.PodRunning:
				report.Workloads.RunningPods++
			case corev1.PodPending:
				report.Workloads.PendingPods++
			case corev1.PodFailed:
				report.Workloads.FailedPods++
			}

			// Count restarts
			restarts := 0
			for _, cs := range pod.Status.ContainerStatuses {
				restarts += int(cs.RestartCount)
			}

			// Get images
			var images []string
			for _, c := range pod.Spec.Containers {
				images = append(images, c.Image)
				imageCount[c.Image]++
			}

			// Security checks
			for _, c := range pod.Spec.Containers {
				if c.SecurityContext != nil {
					if c.SecurityContext.Privileged != nil && *c.SecurityContext.Privileged {
						report.SecurityInfo.PrivilegedPods++
					}
					if c.SecurityContext.RunAsUser != nil && *c.SecurityContext.RunAsUser == 0 {
						report.SecurityInfo.RootContainers++
					}
				}
			}
			if pod.Spec.HostNetwork {
				report.SecurityInfo.HostNetworkPods++
			}

			ready := 0
			total := len(pod.Status.ContainerStatuses)
			for _, cs := range pod.Status.ContainerStatuses {
				if cs.Ready {
					ready++
				}
			}

			podInfo := PodInfo{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Status:    string(pod.Status.Phase),
				Ready:     fmt.Sprintf("%d/%d", ready, total),
				Restarts:  restarts,
				Node:      pod.Spec.NodeName,
				IP:        pod.Status.PodIP,
				Images:    images,
				Age:       time.Since(pod.CreationTimestamp.Time).Round(time.Second).String(),
			}
			report.Pods = append(report.Pods, podInfo)
		}

		// Deployments
		deps, _ := rg.server.k8sClient.ListDeployments(ctx, ns.Name)
		for _, dep := range deps {
			report.Workloads.TotalDeployments++

			replicas := int32(1)
			if dep.Spec.Replicas != nil {
				replicas = *dep.Spec.Replicas
			}

			if dep.Status.ReadyReplicas == replicas {
				report.Workloads.HealthyDeploys++
			}

			strategy := "RollingUpdate"
			if dep.Spec.Strategy.Type != "" {
				strategy = string(dep.Spec.Strategy.Type)
			}

			depInfo := DeploymentInfo{
				Name:      dep.Name,
				Namespace: dep.Namespace,
				Ready:     fmt.Sprintf("%d/%d", dep.Status.ReadyReplicas, replicas),
				UpToDate:  int(dep.Status.UpdatedReplicas),
				Available: int(dep.Status.AvailableReplicas),
				Strategy:  strategy,
				Age:       time.Since(dep.CreationTimestamp.Time).Round(time.Second).String(),
			}
			report.Deployments = append(report.Deployments, depInfo)
		}

		// Services
		svcs, _ := rg.server.k8sClient.ListServices(ctx, ns.Name)
		for _, svc := range svcs {
			report.Workloads.TotalServices++

			ports := make([]string, len(svc.Spec.Ports))
			for i, p := range svc.Spec.Ports {
				ports[i] = fmt.Sprintf("%d/%s", p.Port, p.Protocol)
			}

			externalIP := "<none>"
			if len(svc.Status.LoadBalancer.Ingress) > 0 {
				ips := []string{}
				for _, ing := range svc.Status.LoadBalancer.Ingress {
					if ing.IP != "" {
						ips = append(ips, ing.IP)
					} else if ing.Hostname != "" {
						ips = append(ips, ing.Hostname)
					}
				}
				if len(ips) > 0 {
					externalIP = strings.Join(ips, ", ")
				}
			}

			svcInfo := ServiceInfo{
				Name:       svc.Name,
				Namespace:  svc.Namespace,
				Type:       string(svc.Spec.Type),
				ClusterIP:  svc.Spec.ClusterIP,
				ExternalIP: externalIP,
				Ports:      strings.Join(ports, ", "),
				Age:        time.Since(svc.CreationTimestamp.Time).Round(time.Second).String(),
			}
			report.Services = append(report.Services, svcInfo)
		}

		// ConfigMaps & Secrets count
		configmaps, _ := rg.server.k8sClient.ListConfigMaps(ctx, ns.Name)
		report.Workloads.TotalConfigMaps += len(configmaps)

		secrets, _ := rg.server.k8sClient.ListSecrets(ctx, ns.Name)
		report.SecurityInfo.Secrets += len(secrets)
	}

	// Build image list
	for image, count := range imageCount {
		parts := strings.Split(image, ":")
		repo := parts[0]
		tag := "latest"
		if len(parts) > 1 {
			tag = parts[1]
		}

		report.Images = append(report.Images, ImageInfo{
			Image:      image,
			Repository: repo,
			Tag:        tag,
			PodCount:   count,
		})
	}

	// Sort images by pod count
	sort.Slice(report.Images, func(i, j int) bool {
		return report.Images[i].PodCount > report.Images[j].PodCount
	})

	// Get events (warnings only, last 50)
	events, _ := rg.server.k8sClient.ListEvents(ctx, "")
	warningEvents := []EventInfo{}
	for _, event := range events {
		if event.Type == "Warning" {
			warningEvents = append(warningEvents, EventInfo{
				Type:      event.Type,
				Reason:    event.Reason,
				Object:    fmt.Sprintf("%s/%s", event.InvolvedObject.Kind, event.InvolvedObject.Name),
				Message:   event.Message,
				Count:     int(event.Count),
				FirstSeen: event.FirstTimestamp.Format(time.RFC3339),
				LastSeen:  event.LastTimestamp.Format(time.RFC3339),
			})
		}
	}
	// Keep only last 50 warning events
	if len(warningEvents) > 50 {
		warningEvents = warningEvents[:50]
	}
	report.Events = warningEvents

	// Calculate health score
	report.HealthScore = calculateHealthScore(
		report.NodeSummary.Ready, report.NodeSummary.Total,
		report.Workloads.RunningPods, report.Workloads.TotalPods,
	)

	// Set cluster info
	report.ClusterInfo = ClusterInfo{
		TotalNodes: report.NodeSummary.Total,
		TotalPods:  report.Workloads.TotalPods,
	}

	// Generate FinOps analysis
	if sections.FinOps {
		report.FinOpsAnalysis = rg.generateFinOpsAnalysis(ctx, namespaces, report)
	}

	// Add metrics history if collector is available
	if sections.Metrics && rg.server.metricsCollector != nil {
		report.MetricsHistory = rg.generateMetricsHistory(ctx)
	}

	// Run security scan if scanner is available
	if sections.SecurityBasic && rg.server.securityScanner != nil {
		if sections.SecurityFull {
			report.SecurityScan = rg.generateFullSecurityScan(ctx)
		} else {
			report.SecurityScan = rg.generateSecurityScan(ctx)
		}
	}

	return report, nil
}

// generateMetricsHistory retrieves historical metrics for the report
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
func (rg *ReportGenerator) generateFinOpsAnalysis(ctx context.Context, namespaces []corev1.Namespace, report *ComprehensiveReport) FinOpsAnalysis {
	analysis := FinOpsAnalysis{
		CostByNamespace:          []NamespaceCost{},
		CostOptimizations:        []CostOptimization{},
		UnderutilizedResources:   []UnderutilizedResource{},
		OverprovisionedWorkloads: []OverprovisionedWorkload{},
	}

	// Reference costs (approximate AWS EKS pricing)
	// vCPU: ~$0.04/hour, Memory: ~$0.004/GB/hour
	const cpuHourlyCost = 0.04     // per vCPU
	const memoryHourlyCost = 0.004 // per GB

	var totalCPURequests, totalCPULimits int64 // millicores
	var totalMemRequests, totalMemLimits int64 // bytes
	var totalNodeCPUCapacity int64             // millicores
	var totalNodeMemCapacity int64             // bytes
	var podsWithoutRequests, podsWithoutLimits int

	// Calculate node capacity
	nodes, _ := rg.server.k8sClient.ListNodes(ctx)
	for _, node := range nodes {
		cpu := node.Status.Capacity.Cpu()
		mem := node.Status.Capacity.Memory()
		totalNodeCPUCapacity += cpu.MilliValue()
		totalNodeMemCapacity += mem.Value()
	}

	// Analyze each namespace
	nsCosts := make(map[string]*NamespaceCost)

	for _, ns := range namespaces {
		pods, _ := rg.server.k8sClient.ListPods(ctx, ns.Name)

		nsCost := &NamespaceCost{
			Namespace: ns.Name,
			PodCount:  len(pods),
		}

		var nsCPU, nsMem int64

		for _, pod := range pods {
			podHasRequests := false
			podHasLimits := false

			for _, container := range pod.Spec.Containers {
				// Requests
				if cpuReq := container.Resources.Requests.Cpu(); cpuReq != nil {
					nsCPU += cpuReq.MilliValue()
					totalCPURequests += cpuReq.MilliValue()
					podHasRequests = true
				}
				if memReq := container.Resources.Requests.Memory(); memReq != nil {
					nsMem += memReq.Value()
					totalMemRequests += memReq.Value()
					podHasRequests = true
				}

				// Limits
				if cpuLim := container.Resources.Limits.Cpu(); cpuLim != nil {
					totalCPULimits += cpuLim.MilliValue()
					podHasLimits = true
				}
				if memLim := container.Resources.Limits.Memory(); memLim != nil {
					totalMemLimits += memLim.Value()
					podHasLimits = true
				}
			}

			if !podHasRequests {
				podsWithoutRequests++
			}
			if !podHasLimits {
				podsWithoutLimits++
			}
		}

		// Calculate namespace cost (monthly estimate)
		cpuCores := float64(nsCPU) / 1000.0
		memGB := float64(nsMem) / (1024 * 1024 * 1024)
		monthlyHours := 730.0 // average hours per month

		nsCost.CPURequests = fmt.Sprintf("%.2f cores", cpuCores)
		nsCost.MemoryRequests = fmt.Sprintf("%.2f GB", memGB)
		nsCost.EstimatedCost = (cpuCores*cpuHourlyCost + memGB*memoryHourlyCost) * monthlyHours

		nsCosts[ns.Name] = nsCost
	}

	// Calculate total and percentages
	var totalCost float64
	for _, nsCost := range nsCosts {
		totalCost += nsCost.EstimatedCost
	}

	for _, nsCost := range nsCosts {
		if totalCost > 0 {
			nsCost.CostPercentage = (nsCost.EstimatedCost / totalCost) * 100
		}
		analysis.CostByNamespace = append(analysis.CostByNamespace, *nsCost)
	}

	// Sort by cost descending
	sort.Slice(analysis.CostByNamespace, func(i, j int) bool {
		return analysis.CostByNamespace[i].EstimatedCost > analysis.CostByNamespace[j].EstimatedCost
	})

	analysis.TotalEstimatedMonthlyCost = totalCost

	// Resource efficiency
	analysis.ResourceEfficiency = ResourceEfficiencyInfo{
		TotalCPURequests:    fmt.Sprintf("%.2f cores", float64(totalCPURequests)/1000.0),
		TotalCPULimits:      fmt.Sprintf("%.2f cores", float64(totalCPULimits)/1000.0),
		TotalMemoryRequests: fmt.Sprintf("%.2f GB", float64(totalMemRequests)/(1024*1024*1024)),
		TotalMemoryLimits:   fmt.Sprintf("%.2f GB", float64(totalMemLimits)/(1024*1024*1024)),
		PodsWithoutRequests: podsWithoutRequests,
		PodsWithoutLimits:   podsWithoutLimits,
	}

	if totalNodeCPUCapacity > 0 {
		analysis.ResourceEfficiency.CPURequestsVsCapacity = float64(totalCPURequests) / float64(totalNodeCPUCapacity) * 100
	}
	if totalNodeMemCapacity > 0 {
		analysis.ResourceEfficiency.MemoryRequestsVsCapacity = float64(totalMemRequests) / float64(totalNodeMemCapacity) * 100
	}

	// Generate cost optimization recommendations
	analysis.CostOptimizations = rg.generateCostOptimizations(report, &analysis)

	// Analyze underutilized deployments
	for _, dep := range report.Deployments {
		// Check for deployments with many unavailable replicas
		parts := strings.Split(dep.Ready, "/")
		if len(parts) == 2 {
			ready := 0
			total := 0
			fmt.Sscanf(parts[0], "%d", &ready)
			fmt.Sscanf(parts[1], "%d", &total)

			if total > 1 && ready < total {
				analysis.OverprovisionedWorkloads = append(analysis.OverprovisionedWorkloads, OverprovisionedWorkload{
					Name:              dep.Name,
					Namespace:         dep.Namespace,
					WorkloadType:      "Deployment",
					CurrentReplicas:   total,
					SuggestedReplicas: ready,
					Reason:            fmt.Sprintf("Only %d/%d replicas are ready - consider reducing replicas or investigating issues", ready, total),
				})
			}
		}
	}

	return analysis
}

// generateCostOptimizations creates cost saving recommendations
func (rg *ReportGenerator) generateCostOptimizations(report *ComprehensiveReport, analysis *FinOpsAnalysis) []CostOptimization {
	var optimizations []CostOptimization

	// Check for pods without resource requests
	if analysis.ResourceEfficiency.PodsWithoutRequests > 0 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Resource Management",
			Description:     fmt.Sprintf("%d pods are running without resource requests defined", analysis.ResourceEfficiency.PodsWithoutRequests),
			Impact:          "Without resource requests, pods may be scheduled inefficiently leading to resource contention or waste",
			EstimatedSaving: float64(analysis.ResourceEfficiency.PodsWithoutRequests) * 5.0, // $5 per pod monthly estimate
			Priority:        "high",
		})
	}

	// Check for pods without resource limits
	if analysis.ResourceEfficiency.PodsWithoutLimits > 0 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Resource Management",
			Description:     fmt.Sprintf("%d pods are running without resource limits defined", analysis.ResourceEfficiency.PodsWithoutLimits),
			Impact:          "Without limits, pods can consume unbounded resources affecting cluster stability",
			EstimatedSaving: float64(analysis.ResourceEfficiency.PodsWithoutLimits) * 3.0,
			Priority:        "medium",
		})
	}

	// Check for low cluster utilization
	if analysis.ResourceEfficiency.CPURequestsVsCapacity < 30 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Cluster Sizing",
			Description:     fmt.Sprintf("CPU utilization is only %.1f%% of cluster capacity", analysis.ResourceEfficiency.CPURequestsVsCapacity),
			Impact:          "Consider reducing node count or using smaller instance types",
			EstimatedSaving: analysis.TotalEstimatedMonthlyCost * 0.3, // 30% potential savings
			Priority:        "high",
		})
	}

	if analysis.ResourceEfficiency.MemoryRequestsVsCapacity < 30 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Cluster Sizing",
			Description:     fmt.Sprintf("Memory utilization is only %.1f%% of cluster capacity", analysis.ResourceEfficiency.MemoryRequestsVsCapacity),
			Impact:          "Consider using memory-optimized instances or reducing node count",
			EstimatedSaving: analysis.TotalEstimatedMonthlyCost * 0.2,
			Priority:        "medium",
		})
	}

	// Check for failed pods wasting resources
	if report.Workloads.FailedPods > 0 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Workload Health",
			Description:     fmt.Sprintf("%d pods are in failed state", report.Workloads.FailedPods),
			Impact:          "Failed pods may still consume resources and indicate configuration issues",
			EstimatedSaving: float64(report.Workloads.FailedPods) * 10.0,
			Priority:        "high",
		})
	}

	// Check for pending pods
	if report.Workloads.PendingPods > 0 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Scheduling",
			Description:     fmt.Sprintf("%d pods are pending and cannot be scheduled", report.Workloads.PendingPods),
			Impact:          "Pending pods indicate resource constraints or scheduling issues",
			EstimatedSaving: 0,
			Priority:        "high",
		})
	}

	// Check for many restarts indicating instability
	totalRestarts := 0
	for _, pod := range report.Pods {
		totalRestarts += pod.Restarts
	}
	if totalRestarts > 10 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Stability",
			Description:     fmt.Sprintf("Total of %d container restarts detected across pods", totalRestarts),
			Impact:          "Frequent restarts waste compute resources and may indicate memory/OOM issues",
			EstimatedSaving: float64(totalRestarts) * 0.5,
			Priority:        "medium",
		})
	}

	// LoadBalancer service costs
	lbCount := 0
	for _, svc := range report.Services {
		if svc.Type == "LoadBalancer" {
			lbCount++
		}
	}
	if lbCount > 3 {
		optimizations = append(optimizations, CostOptimization{
			Category:        "Networking",
			Description:     fmt.Sprintf("%d LoadBalancer services detected", lbCount),
			Impact:          "Each LoadBalancer incurs cloud provider costs (~$18/month each). Consider using Ingress controller",
			EstimatedSaving: float64(lbCount-1) * 18.0, // Keep 1, consolidate others
			Priority:        "medium",
		})
	}

	return optimizations
}

// GenerateAIAnalysis uses LLM to analyze the cluster state with FinOps focus
func (rg *ReportGenerator) GenerateAIAnalysis(ctx context.Context, report *ComprehensiveReport) (string, error) {
	if rg.server.aiClient == nil || !rg.server.aiClient.IsReady() {
		return "", fmt.Errorf("AI client not available")
	}

	// Build cost optimization summary
	var costOptSummary strings.Builder
	for i, opt := range report.FinOpsAnalysis.CostOptimizations {
		if i >= 5 {
			break
		}
		costOptSummary.WriteString(fmt.Sprintf("- [%s] %s (Est. saving: $%.2f/mo)\n", opt.Priority, opt.Description, opt.EstimatedSaving))
	}

	// Build summary for AI with FinOps focus
	prompt := fmt.Sprintf(`You are a Kubernetes and FinOps expert. Analyze this cluster state and provide a comprehensive professional report (max 600 words) with special focus on cost optimization.

Cluster Summary:
- Nodes: %d total, %d ready, %d not ready
- Pods: %d total, %d running, %d pending, %d failed
- Deployments: %d total, %d healthy
- Services: %d
- Health Score: %.1f%%

FinOps / Cost Analysis:
- Estimated Monthly Cost: $%.2f
- CPU Utilization vs Capacity: %.1f%%
- Memory Utilization vs Capacity: %.1f%%
- Pods without Resource Requests: %d
- Pods without Resource Limits: %d

Top Cost Optimization Opportunities:
%s

Security Concerns:
- Privileged Pods: %d
- Host Network Pods: %d
- Root Containers: %d

Warning Events: %d

Top Images Used:
%s

Please provide:
1. Overall cluster health assessment
2. **FinOps Cost Analysis** (prioritize this section):
   - Current spending efficiency
   - Top cost drivers
   - Immediate cost reduction opportunities
   - Long-term optimization recommendations
3. Resource optimization recommendations
4. Security observations
5. Action items with priority levels

Be concise, actionable, and focus on ROI for each recommendation.`,
		report.NodeSummary.Total, report.NodeSummary.Ready, report.NodeSummary.NotReady,
		report.Workloads.TotalPods, report.Workloads.RunningPods, report.Workloads.PendingPods, report.Workloads.FailedPods,
		report.Workloads.TotalDeployments, report.Workloads.HealthyDeploys,
		report.Workloads.TotalServices,
		report.HealthScore,
		report.FinOpsAnalysis.TotalEstimatedMonthlyCost,
		report.FinOpsAnalysis.ResourceEfficiency.CPURequestsVsCapacity,
		report.FinOpsAnalysis.ResourceEfficiency.MemoryRequestsVsCapacity,
		report.FinOpsAnalysis.ResourceEfficiency.PodsWithoutRequests,
		report.FinOpsAnalysis.ResourceEfficiency.PodsWithoutLimits,
		costOptSummary.String(),
		report.SecurityInfo.PrivilegedPods, report.SecurityInfo.HostNetworkPods, report.SecurityInfo.RootContainers,
		len(report.Events),
		formatTopImages(report.Images, 5),
	)

	analysis, err := rg.server.aiClient.AskNonStreaming(ctx, prompt)
	if err != nil {
		return "", err
	}

	return analysis, nil
}

func formatTopImages(images []ImageInfo, limit int) string {
	var sb strings.Builder
	for i, img := range images {
		if i >= limit {
			break
		}
		sb.WriteString(fmt.Sprintf("- %s (used by %d pods)\n", img.Image, img.PodCount))
	}
	return sb.String()
}

func calculateHealthScore(healthyNodes, totalNodes, runningPods, totalPods int) float64 {
	if totalNodes == 0 && totalPods == 0 {
		return 100.0
	}

	nodeScore := 50.0
	if totalNodes > 0 {
		nodeScore = float64(healthyNodes) / float64(totalNodes) * 50
	}

	podScore := 50.0
	if totalPods > 0 {
		podScore = float64(runningPods) / float64(totalPods) * 50
	}

	return nodeScore + podScore
}

// ExportToCSV generates CSV format report
func (rg *ReportGenerator) ExportToCSV(report *ComprehensiveReport) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header section
	writer.Write([]string{"K13d Cluster Report"})
	writer.Write([]string{"Generated At:", report.GeneratedAt.Format(time.RFC3339)})
	writer.Write([]string{"Generated By:", report.GeneratedBy})
	writer.Write([]string{"Health Score:", fmt.Sprintf("%.1f%%", report.HealthScore)})
	writer.Write([]string{""})

	// Cluster Summary
	writer.Write([]string{"=== CLUSTER SUMMARY ==="})
	writer.Write([]string{"Metric", "Value"})
	writer.Write([]string{"Total Nodes", fmt.Sprintf("%d", report.NodeSummary.Total)})
	writer.Write([]string{"Ready Nodes", fmt.Sprintf("%d", report.NodeSummary.Ready)})
	writer.Write([]string{"Total Pods", fmt.Sprintf("%d", report.Workloads.TotalPods)})
	writer.Write([]string{"Running Pods", fmt.Sprintf("%d", report.Workloads.RunningPods)})
	writer.Write([]string{"Pending Pods", fmt.Sprintf("%d", report.Workloads.PendingPods)})
	writer.Write([]string{"Failed Pods", fmt.Sprintf("%d", report.Workloads.FailedPods)})
	writer.Write([]string{"Total Deployments", fmt.Sprintf("%d", report.Workloads.TotalDeployments)})
	writer.Write([]string{"Healthy Deployments", fmt.Sprintf("%d", report.Workloads.HealthyDeploys)})
	writer.Write([]string{"Total Services", fmt.Sprintf("%d", report.Workloads.TotalServices)})
	writer.Write([]string{""})

	// Nodes
	writer.Write([]string{"=== NODES ==="})
	writer.Write([]string{"Name", "Status", "Roles", "Version", "CPU", "Memory", "IP"})
	for _, node := range report.Nodes {
		writer.Write([]string{
			node.Name,
			node.Status,
			strings.Join(node.Roles, ","),
			node.KubeletVersion,
			node.CPUCapacity,
			node.MemoryCapacity,
			node.InternalIP,
		})
	}
	writer.Write([]string{""})

	// Namespaces
	writer.Write([]string{"=== NAMESPACES ==="})
	writer.Write([]string{"Name", "Status", "Pods", "Deployments", "Services"})
	for _, ns := range report.Namespaces {
		writer.Write([]string{
			ns.Name,
			ns.Status,
			fmt.Sprintf("%d", ns.PodCount),
			fmt.Sprintf("%d", ns.DeployCount),
			fmt.Sprintf("%d", ns.ServiceCount),
		})
	}
	writer.Write([]string{""})

	// Pods
	writer.Write([]string{"=== PODS ==="})
	writer.Write([]string{"Name", "Namespace", "Status", "Ready", "Restarts", "Node", "IP", "Age"})
	for _, pod := range report.Pods {
		writer.Write([]string{
			pod.Name,
			pod.Namespace,
			pod.Status,
			pod.Ready,
			fmt.Sprintf("%d", pod.Restarts),
			pod.Node,
			pod.IP,
			pod.Age,
		})
	}
	writer.Write([]string{""})

	// Deployments
	writer.Write([]string{"=== DEPLOYMENTS ==="})
	writer.Write([]string{"Name", "Namespace", "Ready", "Up-to-date", "Available", "Strategy", "Age"})
	for _, dep := range report.Deployments {
		writer.Write([]string{
			dep.Name,
			dep.Namespace,
			dep.Ready,
			fmt.Sprintf("%d", dep.UpToDate),
			fmt.Sprintf("%d", dep.Available),
			dep.Strategy,
			dep.Age,
		})
	}
	writer.Write([]string{""})

	// Services
	writer.Write([]string{"=== SERVICES ==="})
	writer.Write([]string{"Name", "Namespace", "Type", "ClusterIP", "ExternalIP", "Ports", "Age"})
	for _, svc := range report.Services {
		writer.Write([]string{
			svc.Name,
			svc.Namespace,
			svc.Type,
			svc.ClusterIP,
			svc.ExternalIP,
			svc.Ports,
			svc.Age,
		})
	}
	writer.Write([]string{""})

	// Images
	writer.Write([]string{"=== CONTAINER IMAGES ==="})
	writer.Write([]string{"Image", "Repository", "Tag", "Pod Count"})
	for _, img := range report.Images {
		writer.Write([]string{
			img.Image,
			img.Repository,
			img.Tag,
			fmt.Sprintf("%d", img.PodCount),
		})
	}
	writer.Write([]string{""})

	// Security
	writer.Write([]string{"=== SECURITY SUMMARY ==="})
	writer.Write([]string{"Metric", "Value"})
	writer.Write([]string{"Secrets Count", fmt.Sprintf("%d", report.SecurityInfo.Secrets)})
	writer.Write([]string{"Privileged Pods", fmt.Sprintf("%d", report.SecurityInfo.PrivilegedPods)})
	writer.Write([]string{"Host Network Pods", fmt.Sprintf("%d", report.SecurityInfo.HostNetworkPods)})
	writer.Write([]string{"Root Containers", fmt.Sprintf("%d", report.SecurityInfo.RootContainers)})
	writer.Write([]string{""})

	// FinOps Analysis
	writer.Write([]string{"=== FINOPS COST ANALYSIS ==="})
	writer.Write([]string{"Metric", "Value"})
	writer.Write([]string{"Estimated Monthly Cost", fmt.Sprintf("$%.2f", report.FinOpsAnalysis.TotalEstimatedMonthlyCost)})
	writer.Write([]string{"Total CPU Requests", report.FinOpsAnalysis.ResourceEfficiency.TotalCPURequests})
	writer.Write([]string{"Total CPU Limits", report.FinOpsAnalysis.ResourceEfficiency.TotalCPULimits})
	writer.Write([]string{"Total Memory Requests", report.FinOpsAnalysis.ResourceEfficiency.TotalMemoryRequests})
	writer.Write([]string{"Total Memory Limits", report.FinOpsAnalysis.ResourceEfficiency.TotalMemoryLimits})
	writer.Write([]string{"CPU Utilization vs Capacity", fmt.Sprintf("%.1f%%", report.FinOpsAnalysis.ResourceEfficiency.CPURequestsVsCapacity)})
	writer.Write([]string{"Memory Utilization vs Capacity", fmt.Sprintf("%.1f%%", report.FinOpsAnalysis.ResourceEfficiency.MemoryRequestsVsCapacity)})
	writer.Write([]string{"Pods Without Requests", fmt.Sprintf("%d", report.FinOpsAnalysis.ResourceEfficiency.PodsWithoutRequests)})
	writer.Write([]string{"Pods Without Limits", fmt.Sprintf("%d", report.FinOpsAnalysis.ResourceEfficiency.PodsWithoutLimits)})
	writer.Write([]string{""})

	// Cost by Namespace
	if len(report.FinOpsAnalysis.CostByNamespace) > 0 {
		writer.Write([]string{"=== COST BY NAMESPACE ==="})
		writer.Write([]string{"Namespace", "Pod Count", "CPU Requests", "Memory Requests", "Est. Cost/Month", "% of Total"})
		for _, ns := range report.FinOpsAnalysis.CostByNamespace {
			writer.Write([]string{
				ns.Namespace,
				fmt.Sprintf("%d", ns.PodCount),
				ns.CPURequests,
				ns.MemoryRequests,
				fmt.Sprintf("$%.2f", ns.EstimatedCost),
				fmt.Sprintf("%.1f%%", ns.CostPercentage),
			})
		}
		writer.Write([]string{""})
	}

	// Cost Optimization Recommendations
	if len(report.FinOpsAnalysis.CostOptimizations) > 0 {
		writer.Write([]string{"=== COST OPTIMIZATION RECOMMENDATIONS ==="})
		writer.Write([]string{"Priority", "Category", "Description", "Impact", "Est. Saving/Month"})
		for _, opt := range report.FinOpsAnalysis.CostOptimizations {
			writer.Write([]string{
				opt.Priority,
				opt.Category,
				opt.Description,
				opt.Impact,
				fmt.Sprintf("$%.2f", opt.EstimatedSaving),
			})
		}
		writer.Write([]string{""})
	}

	// Security Scan Results
	if report.SecurityScan != nil {
		writer.Write([]string{"=== SECURITY SCAN RESULTS ==="})
		writer.Write([]string{"Overall Score", fmt.Sprintf("%.1f", report.SecurityScan.OverallScore)})
		writer.Write([]string{"Risk Level", report.SecurityScan.RiskLevel})
		writer.Write([]string{"Scan Duration", report.SecurityScan.Duration})
		writer.Write([]string{"Tools Used", strings.Join(report.SecurityScan.ToolsUsed, ", ")})
		writer.Write([]string{""})

		if report.SecurityScan.ImageVulnSummary != nil {
			writer.Write([]string{"--- Image Vulnerabilities ---"})
			writer.Write([]string{"Total Images", fmt.Sprintf("%d", report.SecurityScan.ImageVulnSummary.TotalImages)})
			writer.Write([]string{"Scanned Images", fmt.Sprintf("%d", report.SecurityScan.ImageVulnSummary.ScannedImages)})
			writer.Write([]string{"Vulnerable Images", fmt.Sprintf("%d", report.SecurityScan.ImageVulnSummary.VulnerableImages)})
			writer.Write([]string{"Critical", fmt.Sprintf("%d", report.SecurityScan.ImageVulnSummary.CriticalCount)})
			writer.Write([]string{"High", fmt.Sprintf("%d", report.SecurityScan.ImageVulnSummary.HighCount)})
			writer.Write([]string{"Medium", fmt.Sprintf("%d", report.SecurityScan.ImageVulnSummary.MediumCount)})
			writer.Write([]string{"Low", fmt.Sprintf("%d", report.SecurityScan.ImageVulnSummary.LowCount)})
			writer.Write([]string{""})
		}

		if len(report.SecurityScan.PodSecurityIssues) > 0 {
			writer.Write([]string{"--- Pod Security Issues ---"})
			writer.Write([]string{"Namespace", "Pod", "Issue", "Severity"})
			for _, issue := range report.SecurityScan.PodSecurityIssues {
				writer.Write([]string{issue.Namespace, issue.Pod, issue.Issue, issue.Severity})
			}
			writer.Write([]string{""})
		}

		if len(report.SecurityScan.RBACIssues) > 0 {
			writer.Write([]string{"--- RBAC Issues ---"})
			writer.Write([]string{"Kind", "Name", "Issue", "Severity"})
			for _, issue := range report.SecurityScan.RBACIssues {
				writer.Write([]string{issue.Kind, issue.Name, issue.Issue, issue.Severity})
			}
			writer.Write([]string{""})
		}

		if report.SecurityScan.CISBenchmark != nil {
			writer.Write([]string{"--- CIS Benchmark ---"})
			writer.Write([]string{"Version", report.SecurityScan.CISBenchmark.Version})
			writer.Write([]string{"Score", fmt.Sprintf("%.1f%%", report.SecurityScan.CISBenchmark.Score)})
			writer.Write([]string{"Pass", fmt.Sprintf("%d", report.SecurityScan.CISBenchmark.PassCount)})
			writer.Write([]string{"Fail", fmt.Sprintf("%d", report.SecurityScan.CISBenchmark.FailCount)})
			writer.Write([]string{"Warn", fmt.Sprintf("%d", report.SecurityScan.CISBenchmark.WarnCount)})
			writer.Write([]string{""})
		}

		if len(report.SecurityScan.Recommendations) > 0 {
			writer.Write([]string{"--- Security Recommendations ---"})
			writer.Write([]string{"Priority", "Category", "Title", "Description"})
			for _, rec := range report.SecurityScan.Recommendations {
				writer.Write([]string{
					fmt.Sprintf("%d", rec.Priority),
					rec.Category,
					rec.Title,
					rec.Description,
				})
			}
			writer.Write([]string{""})
		}
	}

	// Warning Events
	if len(report.Events) > 0 {
		writer.Write([]string{"=== WARNING EVENTS ==="})
		writer.Write([]string{"Type", "Reason", "Object", "Message", "Count", "Last Seen"})
		for _, event := range report.Events {
			msg := event.Message
			if len(msg) > 100 {
				msg = msg[:100] + "..."
			}
			writer.Write([]string{
				event.Type,
				event.Reason,
				event.Object,
				msg,
				fmt.Sprintf("%d", event.Count),
				event.LastSeen,
			})
		}
		writer.Write([]string{""})
	}

	// AI Analysis
	if report.AIAnalysis != "" {
		writer.Write([]string{"=== AI ANALYSIS ==="})
		// Split analysis into lines for CSV
		lines := strings.Split(report.AIAnalysis, "\n")
		for _, line := range lines {
			writer.Write([]string{line})
		}
	}

	writer.Flush()
	return buf.Bytes(), writer.Error()
}

// ExportToHTML generates HTML format for PDF conversion
func (rg *ReportGenerator) ExportToHTML(report *ComprehensiveReport) string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>K13d Cluster Assessment Report</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 40px; color: #333; line-height: 1.6; }
h1 { color: #1a1b26; border-bottom: 3px solid #7aa2f7; padding-bottom: 10px; margin-bottom: 20px; }
h2 { color: #24283b; margin-top: 40px; border-bottom: 2px solid #7aa2f7; padding-bottom: 8px; }
h2 a { color: inherit; text-decoration: none; }
h2 a:hover { color: #7aa2f7; }
h3 { color: #414868; margin-top: 25px; }
table { width: 100%; border-collapse: collapse; margin: 15px 0; font-size: 12px; }
th, td { padding: 10px 12px; text-align: left; border: 1px solid #ddd; }
th { background: #24283b; color: white; font-weight: 600; }
tr:nth-child(even) { background: #f8f9fa; }
tr:hover { background: #e9ecef; }
.metric-card { display: inline-block; background: #f8f9fa; padding: 15px 25px; margin: 10px; border-radius: 8px; text-align: center; border: 1px solid #e0e0e0; }
.metric-value { font-size: 28px; font-weight: bold; color: #7aa2f7; }
.metric-label { font-size: 12px; color: #666; margin-top: 5px; }
.health-score { font-size: 48px; font-weight: bold; color: #28a745; }
.health-score.warning { color: #ffc107; }
.health-score.critical { color: #dc3545; }
.status-pass { color: #28a745; font-weight: bold; }
.status-warn { color: #ffc107; font-weight: bold; }
.status-fail { color: #dc3545; font-weight: bold; }
.status-running { color: #28a745; font-weight: bold; }
.status-pending { color: #ffc107; font-weight: bold; }
.status-failed { color: #dc3545; font-weight: bold; }
.ai-analysis { background: #f8f9fa; border-left: 4px solid #7aa2f7; padding: 20px; margin: 20px 0; white-space: pre-wrap; }
.warning-box { background: #fff3cd; border: 1px solid #ffc107; border-left: 4px solid #ffc107; padding: 12px 15px; margin: 15px 0; border-radius: 4px; }
.info-box { background: #e7f3ff; border: 1px solid #7aa2f7; border-left: 4px solid #7aa2f7; padding: 12px 15px; margin: 15px 0; border-radius: 4px; }
.cost-card { display: inline-block; background: #e8f5e9; padding: 15px 25px; margin: 10px; border-radius: 8px; text-align: center; border: 1px solid #4caf50; }
.cost-value { font-size: 24px; font-weight: bold; color: #2e7d32; }
.cost-label { font-size: 11px; color: #666; margin-top: 5px; }
.priority-high { color: #dc3545; font-weight: bold; }
.priority-medium { color: #ffc107; font-weight: bold; }
.priority-low { color: #28a745; font-weight: bold; }
.savings-badge { background: #28a745; color: white; padding: 3px 10px; border-radius: 4px; font-size: 12px; font-weight: 600; }
.toc { background: #f8f9fa; border: 1px solid #e0e0e0; border-radius: 8px; padding: 20px 30px; margin: 30px 0; }
.toc h3 { margin-top: 0; color: #24283b; border-bottom: 1px solid #ddd; padding-bottom: 10px; }
.toc ul { list-style: none; padding: 0; margin: 0; }
.toc li { padding: 6px 0; }
.toc a { color: #7aa2f7; text-decoration: none; font-weight: 500; }
.toc a:hover { text-decoration: underline; }
.toc .toc-subsection { margin-left: 20px; font-size: 13px; }
.section-number { color: #7aa2f7; font-weight: bold; margin-right: 8px; }
.back-to-top { font-size: 11px; color: #7aa2f7; text-decoration: none; float: right; }
.back-to-top:hover { text-decoration: underline; }
.footer { margin-top: 50px; text-align: center; color: #999; font-size: 11px; padding-top: 20px; border-top: 1px solid #e0e0e0; }
.report-meta { background: #f8f9fa; padding: 15px 20px; border-radius: 8px; margin-bottom: 30px; }
.report-meta p { margin: 5px 0; }
@media print { body { margin: 20px; } .back-to-top { display: none; } }
</style>
</head>
<body>
`)

	// Header
	sb.WriteString(`<h1 id="top">K13d Cluster Assessment Report</h1>`)
	sb.WriteString(`<div class="report-meta">`)
	sb.WriteString(fmt.Sprintf(`<p><strong>Report Generated:</strong> %s</p>`, report.GeneratedAt.Format("2006-01-02 15:04:05 MST")))
	sb.WriteString(fmt.Sprintf(`<p><strong>Generated By:</strong> %s</p>`, report.GeneratedBy))
	sb.WriteString(fmt.Sprintf(`<p><strong>Cluster Version:</strong> %s</p>`, report.ClusterInfo.ServerVersion))
	sb.WriteString(`</div>`)

	// Table of Contents
	sb.WriteString(`<div class="toc">`)
	sb.WriteString(`<h3>Table of Contents</h3>`)
	sb.WriteString(`<ul>`)
	sb.WriteString(`<li><a href="#section-1"><span class="section-number">1.</span> Executive Summary</a></li>`)
	if report.MetricsHistory != nil && len(report.MetricsHistory.ClusterMetrics) > 0 {
		sb.WriteString(`<li><a href="#section-2"><span class="section-number">2.</span> Resource Usage History</a></li>`)
	}
	if report.SecurityScan != nil {
		sb.WriteString(`<li><a href="#section-3"><span class="section-number">3.</span> Security Assessment</a>`)
		sb.WriteString(`<ul class="toc-subsection">`)
		if report.SecurityScan.ImageVulnSummary != nil {
			sb.WriteString(`<li><a href="#section-3-1">3.1 Image Vulnerabilities</a></li>`)
		}
		if len(report.SecurityScan.PodSecurityIssues) > 0 {
			sb.WriteString(`<li><a href="#section-3-2">3.2 Pod Security Issues</a></li>`)
		}
		if len(report.SecurityScan.RBACIssues) > 0 {
			sb.WriteString(`<li><a href="#section-3-3">3.3 RBAC Issues</a></li>`)
		}
		if report.SecurityScan.CISBenchmark != nil {
			sb.WriteString(`<li><a href="#section-3-4">3.4 CIS Benchmark Results</a></li>`)
		}
		if len(report.SecurityScan.Recommendations) > 0 {
			sb.WriteString(`<li><a href="#section-3-5">3.5 Security Recommendations</a></li>`)
		}
		sb.WriteString(`</ul></li>`)
	}
	if report.AIAnalysis != "" {
		sb.WriteString(`<li><a href="#section-4"><span class="section-number">4.</span> AI Analysis</a></li>`)
	}
	sb.WriteString(`<li><a href="#section-5"><span class="section-number">5.</span> Cluster Infrastructure</a>`)
	sb.WriteString(`<ul class="toc-subsection">`)
	sb.WriteString(`<li><a href="#section-5-1">5.1 Nodes</a></li>`)
	sb.WriteString(`<li><a href="#section-5-2">5.2 Namespaces</a></li>`)
	sb.WriteString(`</ul></li>`)
	sb.WriteString(`<li><a href="#section-6"><span class="section-number">6.</span> Workloads</a>`)
	sb.WriteString(`<ul class="toc-subsection">`)
	sb.WriteString(`<li><a href="#section-6-1">6.1 Pods</a></li>`)
	sb.WriteString(`<li><a href="#section-6-2">6.2 Deployments</a></li>`)
	sb.WriteString(`<li><a href="#section-6-3">6.3 Services</a></li>`)
	sb.WriteString(`<li><a href="#section-6-4">6.4 Container Images</a></li>`)
	sb.WriteString(`</ul></li>`)
	sb.WriteString(`<li><a href="#section-7"><span class="section-number">7.</span> FinOps Cost Analysis</a>`)
	sb.WriteString(`<ul class="toc-subsection">`)
	sb.WriteString(`<li><a href="#section-7-1">7.1 Resource Efficiency</a></li>`)
	sb.WriteString(`<li><a href="#section-7-2">7.2 Cost by Namespace</a></li>`)
	sb.WriteString(`<li><a href="#section-7-3">7.3 Optimization Recommendations</a></li>`)
	sb.WriteString(`</ul></li>`)
	sb.WriteString(`<li><a href="#section-8"><span class="section-number">8.</span> Security Summary</a></li>`)
	if len(report.Events) > 0 {
		sb.WriteString(`<li><a href="#section-9"><span class="section-number">9.</span> Warning Events</a></li>`)
	}
	sb.WriteString(`</ul>`)
	sb.WriteString(`</div>`)

	// Section 1: Executive Summary
	sb.WriteString(`<h2 id="section-1"><a href="#section-1"><span class="section-number">1.</span> Executive Summary</a><a href="#top" class="back-to-top">[Back to Top]</a></h2>`)

	// Health Score
	healthClass := ""
	healthStatus := "PASS"
	if report.HealthScore < 70 {
		healthClass = "critical"
		healthStatus = "FAIL"
	} else if report.HealthScore < 90 {
		healthClass = "warning"
		healthStatus = "WARN"
	}
	sb.WriteString(fmt.Sprintf(`<div style="text-align: center; margin: 30px 0;">
<div class="health-score %s">%.0f%%</div>
<div style="color: #666; margin-top: 10px;">Overall Cluster Health Score - <strong class="status-%s">%s</strong></div>
</div>`, healthClass, report.HealthScore, strings.ToLower(healthStatus), healthStatus))

	// Summary Cards
	sb.WriteString(`<div style="text-align: center;">`)
	sb.WriteString(fmt.Sprintf(`<div class="metric-card"><div class="metric-value">%d</div><div class="metric-label">Total Nodes (%d Ready)</div></div>`,
		report.NodeSummary.Total, report.NodeSummary.Ready))
	sb.WriteString(fmt.Sprintf(`<div class="metric-card"><div class="metric-value">%d</div><div class="metric-label">Total Pods (%d Running)</div></div>`,
		report.Workloads.TotalPods, report.Workloads.RunningPods))
	sb.WriteString(fmt.Sprintf(`<div class="metric-card"><div class="metric-value">%d</div><div class="metric-label">Deployments</div></div>`,
		report.Workloads.TotalDeployments))
	sb.WriteString(fmt.Sprintf(`<div class="metric-card"><div class="metric-value">%d</div><div class="metric-label">Services</div></div>`,
		report.Workloads.TotalServices))
	sb.WriteString(fmt.Sprintf(`<div class="metric-card"><div class="metric-value">%d</div><div class="metric-label">Namespaces</div></div>`,
		report.NamespaceSummary.Total))
	sb.WriteString(`</div>`)

	// Section 2: Metrics History (if available)
	if report.MetricsHistory != nil && len(report.MetricsHistory.ClusterMetrics) > 0 {
		sb.WriteString(`<h2 id="section-2"><a href="#section-2"><span class="section-number">2.</span> Resource Usage History</a><a href="#top" class="back-to-top">[Back to Top]</a></h2>`)
		sb.WriteString(`<p>Resource usage trends over the last 24 hours:</p>`)
		sb.WriteString(`<div style="text-align: center; margin: 20px 0;">`)
		sb.WriteString(fmt.Sprintf(`<div class="metric-card"><div class="metric-value">%d</div><div class="metric-label">Data Points</div></div>`,
			report.MetricsHistory.DataPoints))
		sb.WriteString(fmt.Sprintf(`<div class="metric-card"><div class="metric-value">%dm</div><div class="metric-label">Avg CPU</div></div>`,
			report.MetricsHistory.Summary.AvgCPUUsage))
		sb.WriteString(fmt.Sprintf(`<div class="metric-card"><div class="metric-value">%dm</div><div class="metric-label">Max CPU</div></div>`,
			report.MetricsHistory.Summary.MaxCPUUsage))
		sb.WriteString(fmt.Sprintf(`<div class="metric-card"><div class="metric-value">%d MB</div><div class="metric-label">Avg Memory</div></div>`,
			report.MetricsHistory.Summary.AvgMemoryUsage))
		sb.WriteString(fmt.Sprintf(`<div class="metric-card"><div class="metric-value">%d MB</div><div class="metric-label">Max Memory</div></div>`,
			report.MetricsHistory.Summary.MaxMemoryUsage))
		sb.WriteString(`</div>`)

		// Resource usage table (sample every 6th point for readability)
		sb.WriteString(`<table><tr><th>Timestamp</th><th>CPU (millicores)</th><th>Memory (MB)</th><th>Running Pods</th><th>Ready Nodes</th></tr>`)
		for i, point := range report.MetricsHistory.ClusterMetrics {
			if i%6 != 0 && i != len(report.MetricsHistory.ClusterMetrics)-1 {
				continue
			}
			sb.WriteString(fmt.Sprintf(`<tr><td>%s</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td></tr>`,
				point.Timestamp, point.CPUUsage, point.MemoryUsage, point.RunningPods, point.ReadyNodes))
		}
		sb.WriteString(`</table>`)
	}

	// Section 3: Security Assessment (if available)
	if report.SecurityScan != nil {
		riskClass := "status-pass"
		switch report.SecurityScan.RiskLevel {
		case "Critical":
			riskClass = "status-fail"
		case "High":
			riskClass = "status-fail"
		case "Medium":
			riskClass = "status-warn"
		}

		sb.WriteString(`<h2 id="section-3"><a href="#section-3"><span class="section-number">3.</span> Security Assessment</a><a href="#top" class="back-to-top">[Back to Top]</a></h2>`)
		sb.WriteString(`<p>Comprehensive security analysis of the Kubernetes cluster:</p>`)
		sb.WriteString(`<div style="text-align: center; margin: 20px 0;">`)
		sb.WriteString(fmt.Sprintf(`<div class="metric-card"><div class="metric-value">%.0f</div><div class="metric-label">Security Score</div></div>`,
			report.SecurityScan.OverallScore))
		sb.WriteString(fmt.Sprintf(`<div class="metric-card"><div class="metric-value %s">%s</div><div class="metric-label">Risk Level</div></div>`,
			riskClass, report.SecurityScan.RiskLevel))
		sb.WriteString(fmt.Sprintf(`<div class="metric-card"><div class="metric-value">%s</div><div class="metric-label">Scan Duration</div></div>`,
			report.SecurityScan.Duration))
		sb.WriteString(`</div>`)
		sb.WriteString(fmt.Sprintf(`<p><strong>Assessment Tools:</strong> %s</p>`, strings.Join(report.SecurityScan.ToolsUsed, ", ")))

		// 3.1 Image Vulnerabilities
		if report.SecurityScan.ImageVulnSummary != nil && report.SecurityScan.ImageVulnSummary.ScannedImages > 0 {
			sb.WriteString(`<h3 id="section-3-1"><span class="section-number">3.1</span> Image Vulnerabilities</h3>`)
			sb.WriteString(`<table><tr><th>Metric</th><th>Count</th><th>Status</th></tr>`)
			sb.WriteString(fmt.Sprintf(`<tr><td>Total Images</td><td>%d</td><td>-</td></tr>`, report.SecurityScan.ImageVulnSummary.TotalImages))
			sb.WriteString(fmt.Sprintf(`<tr><td>Scanned Images</td><td>%d</td><td>-</td></tr>`, report.SecurityScan.ImageVulnSummary.ScannedImages))
			vulnStatus := "PASS"
			vulnClass := "status-pass"
			if report.SecurityScan.ImageVulnSummary.VulnerableImages > 0 {
				vulnStatus = "WARN"
				vulnClass = "status-warn"
			}
			sb.WriteString(fmt.Sprintf(`<tr><td>Vulnerable Images</td><td>%d</td><td class="%s">%s</td></tr>`, report.SecurityScan.ImageVulnSummary.VulnerableImages, vulnClass, vulnStatus))
			critStatus := "PASS"
			critClass := "status-pass"
			if report.SecurityScan.ImageVulnSummary.CriticalCount > 0 {
				critStatus = "FAIL"
				critClass = "status-fail"
			}
			sb.WriteString(fmt.Sprintf(`<tr><td>Critical Vulnerabilities</td><td>%d</td><td class="%s">%s</td></tr>`, report.SecurityScan.ImageVulnSummary.CriticalCount, critClass, critStatus))
			highStatus := "PASS"
			highClass := "status-pass"
			if report.SecurityScan.ImageVulnSummary.HighCount > 0 {
				highStatus = "FAIL"
				highClass = "status-fail"
			}
			sb.WriteString(fmt.Sprintf(`<tr><td>High Vulnerabilities</td><td>%d</td><td class="%s">%s</td></tr>`, report.SecurityScan.ImageVulnSummary.HighCount, highClass, highStatus))
			sb.WriteString(fmt.Sprintf(`<tr><td>Medium Vulnerabilities</td><td>%d</td><td class="status-warn">INFO</td></tr>`, report.SecurityScan.ImageVulnSummary.MediumCount))
			sb.WriteString(fmt.Sprintf(`<tr><td>Low Vulnerabilities</td><td>%d</td><td>INFO</td></tr>`, report.SecurityScan.ImageVulnSummary.LowCount))
			sb.WriteString(`</table>`)
		}

		// 3.2 Pod Security Issues
		if len(report.SecurityScan.PodSecurityIssues) > 0 {
			sb.WriteString(`<h3 id="section-3-2"><span class="section-number">3.2</span> Pod Security Issues</h3>`)
			sb.WriteString(fmt.Sprintf(`<p>Found <strong>%d</strong> pod security issues:</p>`, len(report.SecurityScan.PodSecurityIssues)))
			sb.WriteString(`<table><tr><th>Namespace</th><th>Pod</th><th>Issue Description</th><th>Severity</th><th>Status</th></tr>`)
			for i, issue := range report.SecurityScan.PodSecurityIssues {
				if i >= 15 {
					sb.WriteString(fmt.Sprintf(`<tr><td colspan="5"><em>... and %d more issues (see full scan for details)</em></td></tr>`, len(report.SecurityScan.PodSecurityIssues)-15))
					break
				}
				sevClass := "status-pass"
				status := "INFO"
				if issue.Severity == "CRITICAL" || issue.Severity == "HIGH" {
					sevClass = "status-fail"
					status = "FAIL"
				} else if issue.Severity == "MEDIUM" {
					sevClass = "status-warn"
					status = "WARN"
				}
				sb.WriteString(fmt.Sprintf(`<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td class="%s">%s</td></tr>`,
					issue.Namespace, issue.Pod, issue.Issue, issue.Severity, sevClass, status))
			}
			sb.WriteString(`</table>`)
		}

		// 3.3 RBAC Issues
		if len(report.SecurityScan.RBACIssues) > 0 {
			sb.WriteString(`<h3 id="section-3-3"><span class="section-number">3.3</span> RBAC Issues</h3>`)
			sb.WriteString(fmt.Sprintf(`<p>Found <strong>%d</strong> RBAC configuration issues:</p>`, len(report.SecurityScan.RBACIssues)))
			sb.WriteString(`<table><tr><th>Kind</th><th>Name</th><th>Issue Description</th><th>Severity</th><th>Status</th></tr>`)
			for i, issue := range report.SecurityScan.RBACIssues {
				if i >= 15 {
					sb.WriteString(fmt.Sprintf(`<tr><td colspan="5"><em>... and %d more issues (see full scan for details)</em></td></tr>`, len(report.SecurityScan.RBACIssues)-15))
					break
				}
				sevClass := "status-pass"
				status := "INFO"
				if issue.Severity == "CRITICAL" || issue.Severity == "HIGH" {
					sevClass = "status-fail"
					status = "FAIL"
				} else if issue.Severity == "MEDIUM" {
					sevClass = "status-warn"
					status = "WARN"
				}
				sb.WriteString(fmt.Sprintf(`<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td class="%s">%s</td></tr>`,
					issue.Kind, issue.Name, issue.Issue, issue.Severity, sevClass, status))
			}
			sb.WriteString(`</table>`)
		}

		// 3.4 CIS Benchmark
		if report.SecurityScan.CISBenchmark != nil {
			sb.WriteString(`<h3 id="section-3-4"><span class="section-number">3.4</span> CIS Benchmark Results</h3>`)
			sb.WriteString(`<p>CIS Kubernetes Benchmark compliance assessment:</p>`)
			sb.WriteString(`<table><tr><th>Metric</th><th>Value</th><th>Status</th></tr>`)
			sb.WriteString(fmt.Sprintf(`<tr><td>Benchmark Version</td><td>%s</td><td>-</td></tr>`, report.SecurityScan.CISBenchmark.Version))
			sb.WriteString(fmt.Sprintf(`<tr><td>Total Checks</td><td>%d</td><td>-</td></tr>`, report.SecurityScan.CISBenchmark.TotalChecks))
			scoreClass := "status-pass"
			scoreStatus := "PASS"
			if report.SecurityScan.CISBenchmark.Score < 70 {
				scoreClass = "status-fail"
				scoreStatus = "FAIL"
			} else if report.SecurityScan.CISBenchmark.Score < 90 {
				scoreClass = "status-warn"
				scoreStatus = "WARN"
			}
			sb.WriteString(fmt.Sprintf(`<tr><td>Compliance Score</td><td>%.1f%%</td><td class="%s">%s</td></tr>`, report.SecurityScan.CISBenchmark.Score, scoreClass, scoreStatus))
			sb.WriteString(fmt.Sprintf(`<tr><td>Passed Checks</td><td>%d</td><td class="status-pass">PASS</td></tr>`, report.SecurityScan.CISBenchmark.PassCount))
			sb.WriteString(fmt.Sprintf(`<tr><td>Failed Checks</td><td>%d</td><td class="status-fail">FAIL</td></tr>`, report.SecurityScan.CISBenchmark.FailCount))
			sb.WriteString(fmt.Sprintf(`<tr><td>Warning Checks</td><td>%d</td><td class="status-warn">WARN</td></tr>`, report.SecurityScan.CISBenchmark.WarnCount))
			sb.WriteString(`</table>`)
		}

		// 3.5 Security Recommendations
		if len(report.SecurityScan.Recommendations) > 0 {
			sb.WriteString(`<h3 id="section-3-5"><span class="section-number">3.5</span> Security Recommendations</h3>`)
			sb.WriteString(`<p>Prioritized security improvement recommendations:</p>`)
			sb.WriteString(`<table><tr><th>Priority</th><th>Category</th><th>Recommendation</th><th>Impact</th></tr>`)
			for _, rec := range report.SecurityScan.Recommendations {
				prioClass := "status-pass"
				if rec.Priority <= 2 {
					prioClass = "status-fail"
				} else if rec.Priority <= 3 {
					prioClass = "status-warn"
				}
				sb.WriteString(fmt.Sprintf(`<tr><td class="%s">P%d</td><td>%s</td><td>%s</td><td>%s</td></tr>`,
					prioClass, rec.Priority, rec.Category, rec.Title, rec.Impact))
			}
			sb.WriteString(`</table>`)
		}
	}

	// Section 4: AI Analysis (if available)
	if report.AIAnalysis != "" {
		sb.WriteString(`<h2 id="section-4"><a href="#section-4"><span class="section-number">4.</span> AI Analysis</a><a href="#top" class="back-to-top">[Back to Top]</a></h2>`)
		sb.WriteString(`<p>AI-powered cluster analysis and recommendations:</p>`)
		sb.WriteString(fmt.Sprintf(`<div class="ai-analysis">%s</div>`, report.AIAnalysis))
	}

	// Section 5: Cluster Infrastructure
	sb.WriteString(`<h2 id="section-5"><a href="#section-5"><span class="section-number">5.</span> Cluster Infrastructure</a><a href="#top" class="back-to-top">[Back to Top]</a></h2>`)

	// 5.1 Nodes
	sb.WriteString(`<h3 id="section-5-1"><span class="section-number">5.1</span> Nodes</h3>`)
	sb.WriteString(fmt.Sprintf(`<p>Total: <strong>%d</strong> nodes (%d Ready, %d Not Ready)</p>`, report.NodeSummary.Total, report.NodeSummary.Ready, report.NodeSummary.NotReady))
	sb.WriteString(`<table><tr><th>Name</th><th>Status</th><th>Roles</th><th>Version</th><th>CPU Capacity</th><th>Memory Capacity</th><th>Internal IP</th></tr>`)
	for _, node := range report.Nodes {
		statusClass := "status-pass"
		status := "Ready"
		if node.Status != "Ready" {
			statusClass = "status-fail"
			status = node.Status
		}
		sb.WriteString(fmt.Sprintf(`<tr><td>%s</td><td class="%s">%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>`,
			node.Name, statusClass, status, strings.Join(node.Roles, ", "), node.KubeletVersion, node.CPUCapacity, node.MemoryCapacity, node.InternalIP))
	}
	sb.WriteString(`</table>`)

	// 5.2 Namespaces
	sb.WriteString(`<h3 id="section-5-2"><span class="section-number">5.2</span> Namespaces</h3>`)
	sb.WriteString(fmt.Sprintf(`<p>Total: <strong>%d</strong> namespaces (%d Active)</p>`, report.NamespaceSummary.Total, report.NamespaceSummary.Active))
	sb.WriteString(`<table><tr><th>Name</th><th>Status</th><th>Pods</th><th>Deployments</th><th>Services</th></tr>`)
	for _, ns := range report.Namespaces {
		sb.WriteString(fmt.Sprintf(`<tr><td>%s</td><td>%s</td><td>%d</td><td>%d</td><td>%d</td></tr>`,
			ns.Name, ns.Status, ns.PodCount, ns.DeployCount, ns.ServiceCount))
	}
	sb.WriteString(`</table>`)

	// Section 6: Workloads
	sb.WriteString(`<h2 id="section-6"><a href="#section-6"><span class="section-number">6.</span> Workloads</a><a href="#top" class="back-to-top">[Back to Top]</a></h2>`)

	// 6.1 Pods (limit to first 50 for readability)
	sb.WriteString(`<h3 id="section-6-1"><span class="section-number">6.1</span> Pods</h3>`)
	sb.WriteString(fmt.Sprintf(`<p>Total: <strong>%d</strong> pods (%d Running, %d Pending, %d Failed)</p>`,
		report.Workloads.TotalPods, report.Workloads.RunningPods, report.Workloads.PendingPods, report.Workloads.FailedPods))
	if len(report.Pods) > 50 {
		sb.WriteString(fmt.Sprintf(`<p><em>Showing first 50 of %d pods</em></p>`, len(report.Pods)))
	}
	sb.WriteString(`<table><tr><th>Name</th><th>Namespace</th><th>Status</th><th>Ready</th><th>Restarts</th><th>Node</th><th>Age</th></tr>`)
	for i, pod := range report.Pods {
		if i >= 50 {
			break
		}
		statusClass := "status-pass"
		switch pod.Status {
		case "Pending":
			statusClass = "status-warn"
		case "Failed", "CrashLoopBackOff", "Error":
			statusClass = "status-fail"
		}
		sb.WriteString(fmt.Sprintf(`<tr><td>%s</td><td>%s</td><td class="%s">%s</td><td>%s</td><td>%d</td><td>%s</td><td>%s</td></tr>`,
			pod.Name, pod.Namespace, statusClass, pod.Status, pod.Ready, pod.Restarts, pod.Node, pod.Age))
	}
	sb.WriteString(`</table>`)

	// 6.2 Deployments
	sb.WriteString(`<h3 id="section-6-2"><span class="section-number">6.2</span> Deployments</h3>`)
	sb.WriteString(fmt.Sprintf(`<p>Total: <strong>%d</strong> deployments (%d Healthy)</p>`, report.Workloads.TotalDeployments, report.Workloads.HealthyDeploys))
	sb.WriteString(`<table><tr><th>Name</th><th>Namespace</th><th>Ready</th><th>Up-to-date</th><th>Available</th><th>Strategy</th><th>Age</th></tr>`)
	for _, dep := range report.Deployments {
		sb.WriteString(fmt.Sprintf(`<tr><td>%s</td><td>%s</td><td>%s</td><td>%d</td><td>%d</td><td>%s</td><td>%s</td></tr>`,
			dep.Name, dep.Namespace, dep.Ready, dep.UpToDate, dep.Available, dep.Strategy, dep.Age))
	}
	sb.WriteString(`</table>`)

	// 6.3 Services
	sb.WriteString(`<h3 id="section-6-3"><span class="section-number">6.3</span> Services</h3>`)
	sb.WriteString(fmt.Sprintf(`<p>Total: <strong>%d</strong> services</p>`, report.Workloads.TotalServices))
	sb.WriteString(`<table><tr><th>Name</th><th>Namespace</th><th>Type</th><th>Cluster IP</th><th>External IP</th><th>Ports</th></tr>`)
	for _, svc := range report.Services {
		sb.WriteString(fmt.Sprintf(`<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>`,
			svc.Name, svc.Namespace, svc.Type, svc.ClusterIP, svc.ExternalIP, svc.Ports))
	}
	sb.WriteString(`</table>`)

	// 6.4 Container Images
	sb.WriteString(`<h3 id="section-6-4"><span class="section-number">6.4</span> Container Images</h3>`)
	sb.WriteString(fmt.Sprintf(`<p>Total: <strong>%d</strong> unique images in use</p>`, len(report.Images)))
	sb.WriteString(`<table><tr><th>Repository</th><th>Tag</th><th>Pod Count</th></tr>`)
	for i, img := range report.Images {
		if i >= 25 {
			sb.WriteString(fmt.Sprintf(`<tr><td colspan="3"><em>... and %d more images</em></td></tr>`, len(report.Images)-25))
			break
		}
		sb.WriteString(fmt.Sprintf(`<tr><td>%s</td><td>%s</td><td>%d</td></tr>`,
			img.Repository, img.Tag, img.PodCount))
	}
	sb.WriteString(`</table>`)

	// Section 7: FinOps Cost Analysis
	sb.WriteString(`<h2 id="section-7"><a href="#section-7"><span class="section-number">7.</span> FinOps Cost Analysis</a><a href="#top" class="back-to-top">[Back to Top]</a></h2>`)
	sb.WriteString(`<p>Resource cost analysis and optimization opportunities:</p>`)
	sb.WriteString(`<div style="text-align: center; margin: 20px 0;">`)
	sb.WriteString(fmt.Sprintf(`<div class="cost-card"><div class="cost-value">$%.2f</div><div class="cost-label">Est. Monthly Cost</div></div>`,
		report.FinOpsAnalysis.TotalEstimatedMonthlyCost))
	sb.WriteString(fmt.Sprintf(`<div class="cost-card"><div class="cost-value">%.1f%%</div><div class="cost-label">CPU Utilization</div></div>`,
		report.FinOpsAnalysis.ResourceEfficiency.CPURequestsVsCapacity))
	sb.WriteString(fmt.Sprintf(`<div class="cost-card"><div class="cost-value">%.1f%%</div><div class="cost-label">Memory Utilization</div></div>`,
		report.FinOpsAnalysis.ResourceEfficiency.MemoryRequestsVsCapacity))
	sb.WriteString(`</div>`)

	// 7.1 Resource Efficiency
	sb.WriteString(`<h3 id="section-7-1"><span class="section-number">7.1</span> Resource Efficiency</h3>`)
	sb.WriteString(`<table><tr><th>Metric</th><th>Value</th><th>Status</th></tr>`)
	sb.WriteString(fmt.Sprintf(`<tr><td>Total CPU Requests</td><td>%s</td><td>-</td></tr>`, report.FinOpsAnalysis.ResourceEfficiency.TotalCPURequests))
	sb.WriteString(fmt.Sprintf(`<tr><td>Total CPU Limits</td><td>%s</td><td>-</td></tr>`, report.FinOpsAnalysis.ResourceEfficiency.TotalCPULimits))
	sb.WriteString(fmt.Sprintf(`<tr><td>Total Memory Requests</td><td>%s</td><td>-</td></tr>`, report.FinOpsAnalysis.ResourceEfficiency.TotalMemoryRequests))
	sb.WriteString(fmt.Sprintf(`<tr><td>Total Memory Limits</td><td>%s</td><td>-</td></tr>`, report.FinOpsAnalysis.ResourceEfficiency.TotalMemoryLimits))
	reqStatus := "PASS"
	reqClass := "status-pass"
	if report.FinOpsAnalysis.ResourceEfficiency.PodsWithoutRequests > 0 {
		reqStatus = "WARN"
		reqClass = "status-warn"
	}
	sb.WriteString(fmt.Sprintf(`<tr><td>Pods Without Requests</td><td>%d</td><td class="%s">%s</td></tr>`, report.FinOpsAnalysis.ResourceEfficiency.PodsWithoutRequests, reqClass, reqStatus))
	limStatus := "PASS"
	limClass := "status-pass"
	if report.FinOpsAnalysis.ResourceEfficiency.PodsWithoutLimits > 0 {
		limStatus = "WARN"
		limClass = "status-warn"
	}
	sb.WriteString(fmt.Sprintf(`<tr><td>Pods Without Limits</td><td>%d</td><td class="%s">%s</td></tr>`, report.FinOpsAnalysis.ResourceEfficiency.PodsWithoutLimits, limClass, limStatus))
	sb.WriteString(`</table>`)

	// 7.2 Cost by Namespace
	if len(report.FinOpsAnalysis.CostByNamespace) > 0 {
		sb.WriteString(`<h3 id="section-7-2"><span class="section-number">7.2</span> Cost by Namespace</h3>`)
		sb.WriteString(`<table><tr><th>Namespace</th><th>Pods</th><th>CPU Requests</th><th>Memory Requests</th><th>Est. Cost/Month</th><th>% of Total</th></tr>`)
		for i, ns := range report.FinOpsAnalysis.CostByNamespace {
			if i >= 15 {
				sb.WriteString(fmt.Sprintf(`<tr><td colspan="6"><em>... and %d more namespaces</em></td></tr>`, len(report.FinOpsAnalysis.CostByNamespace)-15))
				break
			}
			sb.WriteString(fmt.Sprintf(`<tr><td>%s</td><td>%d</td><td>%s</td><td>%s</td><td>$%.2f</td><td>%.1f%%</td></tr>`,
				ns.Namespace, ns.PodCount, ns.CPURequests, ns.MemoryRequests, ns.EstimatedCost, ns.CostPercentage))
		}
		sb.WriteString(`</table>`)
	}

	// 7.3 Cost Optimization Recommendations
	if len(report.FinOpsAnalysis.CostOptimizations) > 0 {
		totalSavings := 0.0
		for _, opt := range report.FinOpsAnalysis.CostOptimizations {
			totalSavings += opt.EstimatedSaving
		}
		sb.WriteString(`<h3 id="section-7-3"><span class="section-number">7.3</span> Optimization Recommendations</h3>`)
		sb.WriteString(fmt.Sprintf(`<p><strong>Total Potential Savings:</strong> <span class="savings-badge">$%.2f/month</span></p>`, totalSavings))
		sb.WriteString(`<table><tr><th>Priority</th><th>Category</th><th>Recommendation</th><th>Impact</th><th>Est. Savings</th></tr>`)
		for _, opt := range report.FinOpsAnalysis.CostOptimizations {
			priorityClass := "priority-" + opt.Priority
			sb.WriteString(fmt.Sprintf(`<tr><td class="%s">%s</td><td>%s</td><td>%s</td><td>%s</td><td>$%.2f</td></tr>`,
				priorityClass, strings.ToUpper(opt.Priority), opt.Category, opt.Description, opt.Impact, opt.EstimatedSaving))
		}
		sb.WriteString(`</table>`)
	}

	// Section 8: Security Summary
	sb.WriteString(`<h2 id="section-8"><a href="#section-8"><span class="section-number">8.</span> Security Summary</a><a href="#top" class="back-to-top">[Back to Top]</a></h2>`)
	if report.SecurityInfo.PrivilegedPods > 0 || report.SecurityInfo.HostNetworkPods > 0 || report.SecurityInfo.RootContainers > 0 {
		sb.WriteString(`<div class="warning-box"><strong>Warning:</strong> Security concerns detected - review privileged pods and root containers</div>`)
	}
	sb.WriteString(`<table><tr><th>Security Metric</th><th>Count</th><th>Status</th></tr>`)
	sb.WriteString(fmt.Sprintf(`<tr><td>Total Secrets</td><td>%d</td><td>INFO</td></tr>`, report.SecurityInfo.Secrets))
	sb.WriteString(fmt.Sprintf(`<tr><td>Service Accounts</td><td>%d</td><td>INFO</td></tr>`, report.SecurityInfo.ServiceAccounts))
	sb.WriteString(fmt.Sprintf(`<tr><td>Roles</td><td>%d</td><td>INFO</td></tr>`, report.SecurityInfo.Roles))
	sb.WriteString(fmt.Sprintf(`<tr><td>RoleBindings</td><td>%d</td><td>INFO</td></tr>`, report.SecurityInfo.RoleBindings))
	sb.WriteString(fmt.Sprintf(`<tr><td>ClusterRoles</td><td>%d</td><td>INFO</td></tr>`, report.SecurityInfo.ClusterRoles))
	sb.WriteString(fmt.Sprintf(`<tr><td>ClusterRoleBindings</td><td>%d</td><td>INFO</td></tr>`, report.SecurityInfo.ClusterRoleBindings))
	privStatus := "PASS"
	privClass := "status-pass"
	if report.SecurityInfo.PrivilegedPods > 0 {
		privStatus = "FAIL"
		privClass = "status-fail"
	}
	sb.WriteString(fmt.Sprintf(`<tr><td>Privileged Pods</td><td>%d</td><td class="%s">%s</td></tr>`, report.SecurityInfo.PrivilegedPods, privClass, privStatus))
	hostStatus := "PASS"
	hostClass := "status-pass"
	if report.SecurityInfo.HostNetworkPods > 0 {
		hostStatus = "WARN"
		hostClass = "status-warn"
	}
	sb.WriteString(fmt.Sprintf(`<tr><td>Host Network Pods</td><td>%d</td><td class="%s">%s</td></tr>`, report.SecurityInfo.HostNetworkPods, hostClass, hostStatus))
	rootStatus := "PASS"
	rootClass := "status-pass"
	if report.SecurityInfo.RootContainers > 0 {
		rootStatus = "WARN"
		rootClass = "status-warn"
	}
	sb.WriteString(fmt.Sprintf(`<tr><td>Root Containers</td><td>%d</td><td class="%s">%s</td></tr>`, report.SecurityInfo.RootContainers, rootClass, rootStatus))
	sb.WriteString(`</table>`)

	// Section 9: Warning Events
	if len(report.Events) > 0 {
		sb.WriteString(`<h2 id="section-9"><a href="#section-9"><span class="section-number">9.</span> Warning Events</a><a href="#top" class="back-to-top">[Back to Top]</a></h2>`)
		sb.WriteString(fmt.Sprintf(`<p>Recent warning events in the cluster (%d total):</p>`, len(report.Events)))
		sb.WriteString(`<table><tr><th>Reason</th><th>Object</th><th>Message</th><th>Count</th></tr>`)
		for i, event := range report.Events {
			if i >= 25 {
				sb.WriteString(fmt.Sprintf(`<tr><td colspan="4"><em>... and %d more events</em></td></tr>`, len(report.Events)-25))
				break
			}
			msg := event.Message
			if len(msg) > 100 {
				msg = msg[:100] + "..."
			}
			sb.WriteString(fmt.Sprintf(`<tr><td>%s</td><td>%s</td><td>%s</td><td>%d</td></tr>`,
				event.Reason, event.Object, msg, event.Count))
		}
		sb.WriteString(`</table>`)
	}

	// Footer
	sb.WriteString(`<div class="footer">`)
	sb.WriteString(fmt.Sprintf(`<p>Generated by K13d - AI-Powered Kubernetes Dashboard</p>`))
	sb.WriteString(fmt.Sprintf(`<p>Report generated on %s</p>`, report.GeneratedAt.Format("2006-01-02 15:04:05 MST")))
	sb.WriteString(`</div>`)
	sb.WriteString(`</body></html>`)

	return sb.String()
}

// HandleReports handles report-related API requests
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
		db.RecordAudit(db.AuditEntry{
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
			w.Write(csvData)

		case "html":
			htmlData := rg.ExportToHTML(report)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if download {
				w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=k13d-report-%s.html", time.Now().Format("20060102-150405")))
			}
			w.Write([]byte(htmlData))

		default: // json
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(report)
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
	db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   "preview_report",
		Resource: "cluster",
		Details:  fmt.Sprintf("AI: %v, Sections: %s", includeAI, r.URL.Query().Get("sections")),
	})

	// Return HTML for preview (no Content-Disposition header)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	htmlData := rg.ExportToHTML(report)
	w.Write([]byte(htmlData))
}
