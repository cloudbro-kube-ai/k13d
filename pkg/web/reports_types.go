package web

import (
	"time"
)

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
