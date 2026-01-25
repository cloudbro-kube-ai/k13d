// Package security provides security scanning and compliance checking for Kubernetes clusters
package security

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SeverityLevel represents vulnerability severity
type SeverityLevel string

const (
	SeverityCritical SeverityLevel = "CRITICAL"
	SeverityHigh     SeverityLevel = "HIGH"
	SeverityMedium   SeverityLevel = "MEDIUM"
	SeverityLow      SeverityLevel = "LOW"
	SeverityUnknown  SeverityLevel = "UNKNOWN"
)

// Scanner provides security scanning capabilities
type Scanner struct {
	k8sClient          *k8s.Client
	trivyPath          string
	kubeBenchAvailable bool
	mu                 sync.RWMutex
}

// ScanResult contains overall security scan results
type ScanResult struct {
	ScanTime          time.Time                `json:"scan_time"`
	Duration          string                   `json:"duration"`
	ClusterName       string                   `json:"cluster_name"`
	OverallScore      float64                  `json:"overall_score"` // 0-100
	RiskLevel         string                   `json:"risk_level"`    // Critical, High, Medium, Low
	ImageVulns        *ImageVulnSummary        `json:"image_vulnerabilities,omitempty"`
	PodSecurityIssues []PodSecurityIssue       `json:"pod_security_issues,omitempty"`
	RBACIssues        []RBACIssue              `json:"rbac_issues,omitempty"`
	NetworkIssues     []NetworkIssue           `json:"network_issues,omitempty"`
	CISBenchmark      *CISBenchmarkResult      `json:"cis_benchmark,omitempty"`
	Recommendations   []SecurityRecommendation `json:"recommendations,omitempty"`
}

// ImageVulnSummary summarizes image vulnerabilities
type ImageVulnSummary struct {
	TotalImages      int               `json:"total_images"`
	ScannedImages    int               `json:"scanned_images"`
	VulnerableImages int               `json:"vulnerable_images"`
	CriticalCount    int               `json:"critical_count"`
	HighCount        int               `json:"high_count"`
	MediumCount      int               `json:"medium_count"`
	LowCount         int               `json:"low_count"`
	TopVulnerable    []ImageVulnDetail `json:"top_vulnerable,omitempty"`
}

// ImageVulnDetail contains details about a vulnerable image
type ImageVulnDetail struct {
	Image         string          `json:"image"`
	Namespace     string          `json:"namespace"`
	PodCount      int             `json:"pod_count"`
	CriticalCount int             `json:"critical_count"`
	HighCount     int             `json:"high_count"`
	MediumCount   int             `json:"medium_count"`
	LowCount      int             `json:"low_count"`
	TopVulns      []Vulnerability `json:"top_vulns,omitempty"`
}

// Vulnerability represents a single vulnerability
type Vulnerability struct {
	ID          string        `json:"id"`
	Severity    SeverityLevel `json:"severity"`
	Package     string        `json:"package"`
	Version     string        `json:"version"`
	FixedIn     string        `json:"fixed_in,omitempty"`
	Description string        `json:"description,omitempty"`
}

// PodSecurityIssue represents a pod security concern
type PodSecurityIssue struct {
	Namespace   string `json:"namespace"`
	Pod         string `json:"pod"`
	Container   string `json:"container,omitempty"`
	Issue       string `json:"issue"`
	Severity    string `json:"severity"`
	Remediation string `json:"remediation"`
}

// RBACIssue represents an RBAC security concern
type RBACIssue struct {
	Kind        string `json:"kind"` // Role, ClusterRole, RoleBinding, ClusterRoleBinding
	Name        string `json:"name"`
	Namespace   string `json:"namespace,omitempty"`
	Issue       string `json:"issue"`
	Severity    string `json:"severity"`
	Remediation string `json:"remediation"`
}

// NetworkIssue represents a network security concern
type NetworkIssue struct {
	Namespace   string `json:"namespace"`
	Resource    string `json:"resource"`
	Issue       string `json:"issue"`
	Severity    string `json:"severity"`
	Remediation string `json:"remediation"`
}

// CISBenchmarkResult contains CIS Kubernetes Benchmark results
type CISBenchmarkResult struct {
	Version     string                `json:"version"`
	TotalChecks int                   `json:"total_checks"`
	PassCount   int                   `json:"pass_count"`
	FailCount   int                   `json:"fail_count"`
	WarnCount   int                   `json:"warn_count"`
	InfoCount   int                   `json:"info_count"`
	Score       float64               `json:"score"` // Percentage
	Sections    []CISBenchmarkSection `json:"sections,omitempty"`
}

// CISBenchmarkSection represents a section of CIS benchmark
type CISBenchmarkSection struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	PassCount int                 `json:"pass_count"`
	FailCount int                 `json:"fail_count"`
	WarnCount int                 `json:"warn_count"`
	Checks    []CISBenchmarkCheck `json:"checks,omitempty"`
}

// CISBenchmarkCheck represents a single CIS benchmark check
type CISBenchmarkCheck struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Status      string `json:"status"` // PASS, FAIL, WARN, INFO
	Remediation string `json:"remediation,omitempty"`
}

// SecurityRecommendation provides actionable security advice
type SecurityRecommendation struct {
	Priority    int    `json:"priority"` // 1-5, 1 being highest
	Category    string `json:"category"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
	Remediation string `json:"remediation"`
}

// NewScanner creates a new security scanner
func NewScanner(k8sClient *k8s.Client) *Scanner {
	s := &Scanner{
		k8sClient: k8sClient,
	}

	// Check if trivy is available
	if path, err := exec.LookPath("trivy"); err == nil {
		s.trivyPath = path
	}

	// Check if kube-bench is available
	if _, err := exec.LookPath("kube-bench"); err == nil {
		s.kubeBenchAvailable = true
	}

	return s
}

// Scan performs a comprehensive security scan
func (s *Scanner) Scan(ctx context.Context, namespace string) (*ScanResult, error) {
	startTime := time.Now()

	result := &ScanResult{
		ScanTime: startTime,
	}

	// Get cluster context
	if contextName, err := s.k8sClient.GetCurrentContext(); err == nil {
		result.ClusterName = contextName
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Scan images for vulnerabilities
	wg.Add(1)
	go func() {
		defer wg.Done()
		if vulns, err := s.scanImages(ctx, namespace); err == nil {
			mu.Lock()
			result.ImageVulns = vulns
			mu.Unlock()
		}
	}()

	// Check pod security
	wg.Add(1)
	go func() {
		defer wg.Done()
		if issues, err := s.checkPodSecurity(ctx, namespace); err == nil {
			mu.Lock()
			result.PodSecurityIssues = issues
			mu.Unlock()
		}
	}()

	// Check RBAC
	wg.Add(1)
	go func() {
		defer wg.Done()
		if issues, err := s.checkRBAC(ctx, namespace); err == nil {
			mu.Lock()
			result.RBACIssues = issues
			mu.Unlock()
		}
	}()

	// Check network policies
	wg.Add(1)
	go func() {
		defer wg.Done()
		if issues, err := s.checkNetwork(ctx, namespace); err == nil {
			mu.Lock()
			result.NetworkIssues = issues
			mu.Unlock()
		}
	}()

	// Run CIS benchmark (if kube-bench available)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if s.kubeBenchAvailable {
			if bench, err := s.runCISBenchmark(ctx); err == nil {
				mu.Lock()
				result.CISBenchmark = bench
				mu.Unlock()
			}
		} else {
			// Perform basic CIS-style checks
			if bench, err := s.performBasicCISChecks(ctx, namespace); err == nil {
				mu.Lock()
				result.CISBenchmark = bench
				mu.Unlock()
			}
		}
	}()

	wg.Wait()

	// Calculate overall score and generate recommendations
	result.OverallScore = s.calculateScore(result)
	result.RiskLevel = s.determineRiskLevel(result.OverallScore)
	result.Recommendations = s.generateRecommendations(result)
	result.Duration = time.Since(startTime).String()

	return result, nil
}

// scanImages scans container images for vulnerabilities
func (s *Scanner) scanImages(ctx context.Context, namespace string) (*ImageVulnSummary, error) {
	summary := &ImageVulnSummary{}

	// Get all pods
	var pods *corev1.PodList
	var err error

	if namespace == "" {
		pods, err = s.k8sClient.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	} else {
		pods, err = s.k8sClient.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	// Collect unique images
	imageMap := make(map[string][]struct{ namespace, pod string })
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			imageMap[container.Image] = append(imageMap[container.Image], struct{ namespace, pod string }{pod.Namespace, pod.Name})
		}
		for _, container := range pod.Spec.InitContainers {
			imageMap[container.Image] = append(imageMap[container.Image], struct{ namespace, pod string }{pod.Namespace, pod.Name})
		}
	}

	summary.TotalImages = len(imageMap)

	// If trivy is available, scan images
	if s.trivyPath != "" {
		for image, locations := range imageMap {
			select {
			case <-ctx.Done():
				return summary, ctx.Err()
			default:
			}

			vulns, err := s.scanImageWithTrivy(ctx, image)
			if err != nil {
				continue // Skip images that can't be scanned
			}

			summary.ScannedImages++

			detail := ImageVulnDetail{
				Image:     image,
				Namespace: locations[0].namespace,
				PodCount:  len(locations),
			}

			for _, v := range vulns {
				switch v.Severity {
				case SeverityCritical:
					detail.CriticalCount++
					summary.CriticalCount++
				case SeverityHigh:
					detail.HighCount++
					summary.HighCount++
				case SeverityMedium:
					detail.MediumCount++
					summary.MediumCount++
				case SeverityLow:
					detail.LowCount++
					summary.LowCount++
				}
			}

			if detail.CriticalCount > 0 || detail.HighCount > 0 {
				summary.VulnerableImages++
				// Keep top 5 vulnerabilities
				if len(vulns) > 5 {
					detail.TopVulns = vulns[:5]
				} else {
					detail.TopVulns = vulns
				}
				summary.TopVulnerable = append(summary.TopVulnerable, detail)
			}

			// Limit to top 10 vulnerable images
			if len(summary.TopVulnerable) >= 10 {
				break
			}
		}
	} else {
		// Without trivy, just report image count and basic info
		summary.ScannedImages = 0
	}

	return summary, nil
}

// scanImageWithTrivy scans a single image using trivy
func (s *Scanner) scanImageWithTrivy(ctx context.Context, image string) ([]Vulnerability, error) {
	cmd := exec.CommandContext(ctx, s.trivyPath, "image", "--format", "json", "--severity", "CRITICAL,HIGH,MEDIUM,LOW", "--quiet", image)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var result struct {
		Results []struct {
			Vulnerabilities []struct {
				VulnerabilityID  string `json:"VulnerabilityID"`
				Severity         string `json:"Severity"`
				PkgName          string `json:"PkgName"`
				InstalledVersion string `json:"InstalledVersion"`
				FixedVersion     string `json:"FixedVersion"`
				Description      string `json:"Description"`
			} `json:"Vulnerabilities"`
		} `json:"Results"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	var vulns []Vulnerability
	for _, r := range result.Results {
		for _, v := range r.Vulnerabilities {
			vulns = append(vulns, Vulnerability{
				ID:          v.VulnerabilityID,
				Severity:    SeverityLevel(v.Severity),
				Package:     v.PkgName,
				Version:     v.InstalledVersion,
				FixedIn:     v.FixedVersion,
				Description: truncateString(v.Description, 200),
			})
		}
	}

	return vulns, nil
}

// checkPodSecurity checks for pod security issues
func (s *Scanner) checkPodSecurity(ctx context.Context, namespace string) ([]PodSecurityIssue, error) {
	var issues []PodSecurityIssue

	var pods *corev1.PodList
	var err error

	if namespace == "" {
		pods, err = s.k8sClient.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	} else {
		pods, err = s.k8sClient.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	for _, pod := range pods.Items {
		// Skip system namespaces for some checks
		isSystemNS := isSystemNamespace(pod.Namespace)

		for _, container := range pod.Spec.Containers {
			sc := container.SecurityContext

			// Check for privileged containers
			if sc != nil && sc.Privileged != nil && *sc.Privileged {
				issues = append(issues, PodSecurityIssue{
					Namespace:   pod.Namespace,
					Pod:         pod.Name,
					Container:   container.Name,
					Issue:       "Container running in privileged mode",
					Severity:    "CRITICAL",
					Remediation: "Set securityContext.privileged to false",
				})
			}

			// Check for root user
			if sc == nil || sc.RunAsNonRoot == nil || !*sc.RunAsNonRoot {
				if sc == nil || sc.RunAsUser == nil || *sc.RunAsUser == 0 {
					if !isSystemNS {
						issues = append(issues, PodSecurityIssue{
							Namespace:   pod.Namespace,
							Pod:         pod.Name,
							Container:   container.Name,
							Issue:       "Container may run as root",
							Severity:    "HIGH",
							Remediation: "Set securityContext.runAsNonRoot to true or specify a non-root runAsUser",
						})
					}
				}
			}

			// Check for host namespaces
			if pod.Spec.HostPID {
				issues = append(issues, PodSecurityIssue{
					Namespace:   pod.Namespace,
					Pod:         pod.Name,
					Issue:       "Pod uses host PID namespace",
					Severity:    "HIGH",
					Remediation: "Set hostPID to false unless absolutely required",
				})
			}

			if pod.Spec.HostNetwork {
				if !isSystemNS {
					issues = append(issues, PodSecurityIssue{
						Namespace:   pod.Namespace,
						Pod:         pod.Name,
						Issue:       "Pod uses host network",
						Severity:    "MEDIUM",
						Remediation: "Set hostNetwork to false unless absolutely required",
					})
				}
			}

			// Check for capability additions
			if sc != nil && sc.Capabilities != nil {
				for _, cap := range sc.Capabilities.Add {
					if isDangerousCapability(string(cap)) {
						issues = append(issues, PodSecurityIssue{
							Namespace:   pod.Namespace,
							Pod:         pod.Name,
							Container:   container.Name,
							Issue:       fmt.Sprintf("Container has dangerous capability: %s", cap),
							Severity:    "HIGH",
							Remediation: fmt.Sprintf("Remove %s capability unless absolutely required", cap),
						})
					}
				}
			}

			// Check for missing resource limits
			if container.Resources.Limits.Cpu().IsZero() || container.Resources.Limits.Memory().IsZero() {
				if !isSystemNS {
					issues = append(issues, PodSecurityIssue{
						Namespace:   pod.Namespace,
						Pod:         pod.Name,
						Container:   container.Name,
						Issue:       "Container missing resource limits",
						Severity:    "LOW",
						Remediation: "Set CPU and memory limits to prevent resource exhaustion",
					})
				}
			}

			// Check for latest tag
			if strings.HasSuffix(container.Image, ":latest") || !strings.Contains(container.Image, ":") {
				issues = append(issues, PodSecurityIssue{
					Namespace:   pod.Namespace,
					Pod:         pod.Name,
					Container:   container.Name,
					Issue:       "Container using 'latest' or untagged image",
					Severity:    "MEDIUM",
					Remediation: "Use specific image tags for reproducibility and security",
				})
			}
		}
	}

	return issues, nil
}

// checkRBAC checks for RBAC security issues
func (s *Scanner) checkRBAC(ctx context.Context, namespace string) ([]RBACIssue, error) {
	var issues []RBACIssue

	// Check ClusterRoleBindings for overly permissive bindings
	crbs, err := s.k8sClient.Clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, crb := range crbs.Items {
		// Skip system bindings
		if strings.HasPrefix(crb.Name, "system:") || strings.HasPrefix(crb.Name, "kubeadm:") {
			continue
		}

		// Check for cluster-admin bindings
		if crb.RoleRef.Name == "cluster-admin" {
			for _, subject := range crb.Subjects {
				if subject.Kind == "ServiceAccount" && subject.Namespace != "kube-system" {
					issues = append(issues, RBACIssue{
						Kind:        "ClusterRoleBinding",
						Name:        crb.Name,
						Issue:       fmt.Sprintf("ServiceAccount %s/%s has cluster-admin privileges", subject.Namespace, subject.Name),
						Severity:    "HIGH",
						Remediation: "Use least-privilege principle; create a custom role with only required permissions",
					})
				}
			}
		}
	}

	// Check ClusterRoles for dangerous permissions
	crs, err := s.k8sClient.Clientset.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, cr := range crs.Items {
		if strings.HasPrefix(cr.Name, "system:") {
			continue
		}

		for _, rule := range cr.Rules {
			// Check for wildcard permissions
			for _, apiGroup := range rule.APIGroups {
				for _, resource := range rule.Resources {
					for _, verb := range rule.Verbs {
						if apiGroup == "*" && resource == "*" && verb == "*" {
							issues = append(issues, RBACIssue{
								Kind:        "ClusterRole",
								Name:        cr.Name,
								Issue:       "ClusterRole has full wildcard permissions (*/*/*)",
								Severity:    "CRITICAL",
								Remediation: "Define specific API groups, resources, and verbs instead of wildcards",
							})
						}
					}
				}
			}

			// Check for secrets access
			for _, resource := range rule.Resources {
				if resource == "secrets" || resource == "*" {
					for _, verb := range rule.Verbs {
						if verb == "get" || verb == "list" || verb == "watch" || verb == "*" {
							issues = append(issues, RBACIssue{
								Kind:        "ClusterRole",
								Name:        cr.Name,
								Issue:       "ClusterRole can access secrets cluster-wide",
								Severity:    "MEDIUM",
								Remediation: "Limit secrets access to specific namespaces using Role instead of ClusterRole",
							})
							break
						}
					}
				}
			}
		}
	}

	return issues, nil
}

// checkNetwork checks for network security issues
func (s *Scanner) checkNetwork(ctx context.Context, namespace string) ([]NetworkIssue, error) {
	var issues []NetworkIssue

	// Get all namespaces or specific namespace
	var namespaces []string
	if namespace == "" {
		nsList, err := s.k8sClient.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, ns := range nsList.Items {
			if !isSystemNamespace(ns.Name) {
				namespaces = append(namespaces, ns.Name)
			}
		}
	} else {
		namespaces = []string{namespace}
	}

	// Check for missing NetworkPolicies
	for _, ns := range namespaces {
		policies, err := s.k8sClient.Clientset.NetworkingV1().NetworkPolicies(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}

		if len(policies.Items) == 0 {
			// Check if there are any pods in the namespace
			pods, err := s.k8sClient.Clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{Limit: 1})
			if err == nil && len(pods.Items) > 0 {
				issues = append(issues, NetworkIssue{
					Namespace:   ns,
					Resource:    "NetworkPolicy",
					Issue:       "No NetworkPolicies defined - all pod-to-pod traffic is allowed",
					Severity:    "MEDIUM",
					Remediation: "Create NetworkPolicies to restrict traffic between pods",
				})
			}
		}
	}

	// Check for services with external IPs
	var services *corev1.ServiceList
	var err error
	if namespace == "" {
		services, err = s.k8sClient.Clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	} else {
		services, err = s.k8sClient.Clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return issues, nil
	}

	for _, svc := range services.Items {
		if isSystemNamespace(svc.Namespace) {
			continue
		}

		if svc.Spec.Type == corev1.ServiceTypeLoadBalancer || svc.Spec.Type == corev1.ServiceTypeNodePort {
			issues = append(issues, NetworkIssue{
				Namespace:   svc.Namespace,
				Resource:    fmt.Sprintf("Service/%s", svc.Name),
				Issue:       fmt.Sprintf("Service exposed externally via %s", svc.Spec.Type),
				Severity:    "LOW",
				Remediation: "Ensure external exposure is intentional and properly secured",
			})
		}
	}

	return issues, nil
}

// runCISBenchmark runs kube-bench for CIS benchmark
func (s *Scanner) runCISBenchmark(ctx context.Context) (*CISBenchmarkResult, error) {
	cmd := exec.CommandContext(ctx, "kube-bench", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var result struct {
		Controls []struct {
			ID    string `json:"id"`
			Text  string `json:"text"`
			Tests []struct {
				Section string `json:"section"`
				Desc    string `json:"desc"`
				Results []struct {
					TestNumber  string `json:"test_number"`
					TestDesc    string `json:"test_desc"`
					Status      string `json:"status"`
					Remediation string `json:"remediation"`
				} `json:"results"`
			} `json:"tests"`
		} `json:"Controls"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	benchmark := &CISBenchmarkResult{
		Version: "CIS Kubernetes Benchmark",
	}

	for _, control := range result.Controls {
		section := CISBenchmarkSection{
			ID:   control.ID,
			Name: control.Text,
		}

		for _, test := range control.Tests {
			for _, r := range test.Results {
				check := CISBenchmarkCheck{
					ID:          r.TestNumber,
					Description: r.TestDesc,
					Status:      r.Status,
					Remediation: r.Remediation,
				}

				switch r.Status {
				case "PASS":
					section.PassCount++
					benchmark.PassCount++
				case "FAIL":
					section.FailCount++
					benchmark.FailCount++
				case "WARN":
					section.WarnCount++
					benchmark.WarnCount++
				default:
					benchmark.InfoCount++
				}

				benchmark.TotalChecks++
				section.Checks = append(section.Checks, check)
			}
		}

		benchmark.Sections = append(benchmark.Sections, section)
	}

	if benchmark.TotalChecks > 0 {
		benchmark.Score = float64(benchmark.PassCount) / float64(benchmark.TotalChecks) * 100
	}

	return benchmark, nil
}

// performBasicCISChecks performs basic CIS-style checks without kube-bench
func (s *Scanner) performBasicCISChecks(ctx context.Context, namespace string) (*CISBenchmarkResult, error) {
	benchmark := &CISBenchmarkResult{
		Version: "Basic CIS-style Checks (kube-bench not available)",
	}

	checks := []struct {
		id          string
		description string
		check       func() bool
		remediation string
	}{
		{
			id:          "5.1.1",
			description: "Ensure that the cluster-admin role is only used where required",
			check: func() bool {
				crbs, err := s.k8sClient.Clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
				if err != nil {
					return false
				}
				count := 0
				for _, crb := range crbs.Items {
					if crb.RoleRef.Name == "cluster-admin" && !strings.HasPrefix(crb.Name, "system:") {
						count++
					}
				}
				return count <= 1
			},
			remediation: "Review cluster-admin bindings and use least-privilege roles where possible",
		},
		{
			id:          "5.2.1",
			description: "Minimize the admission of privileged containers",
			check: func() bool {
				pods, err := s.k8sClient.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
				if err != nil {
					return false
				}
				for _, pod := range pods.Items {
					if isSystemNamespace(pod.Namespace) {
						continue
					}
					for _, c := range pod.Spec.Containers {
						if c.SecurityContext != nil && c.SecurityContext.Privileged != nil && *c.SecurityContext.Privileged {
							return false
						}
					}
				}
				return true
			},
			remediation: "Review and remove privileged containers where not required",
		},
		{
			id:          "5.2.2",
			description: "Minimize the admission of containers wishing to share the host PID namespace",
			check: func() bool {
				pods, err := s.k8sClient.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
				if err != nil {
					return false
				}
				for _, pod := range pods.Items {
					if isSystemNamespace(pod.Namespace) {
						continue
					}
					if pod.Spec.HostPID {
						return false
					}
				}
				return true
			},
			remediation: "Review and remove hostPID access where not required",
		},
		{
			id:          "5.3.1",
			description: "Ensure that the CNI in use supports Network Policies",
			check: func() bool {
				// Check if any network policies exist
				policies, err := s.k8sClient.Clientset.NetworkingV1().NetworkPolicies("").List(ctx, metav1.ListOptions{Limit: 1})
				if err != nil {
					return false
				}
				// If network policies exist, CNI supports them
				return len(policies.Items) > 0
			},
			remediation: "Use a CNI plugin that supports NetworkPolicy (e.g., Calico, Cilium, Weave)",
		},
		{
			id:          "5.4.1",
			description: "Prefer using secrets as files over secrets as environment variables",
			check: func() bool {
				pods, err := s.k8sClient.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
				if err != nil {
					return false
				}
				envSecretCount := 0
				for _, pod := range pods.Items {
					if isSystemNamespace(pod.Namespace) {
						continue
					}
					for _, c := range pod.Spec.Containers {
						for _, env := range c.Env {
							if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
								envSecretCount++
							}
						}
					}
				}
				return envSecretCount < 10 // Arbitrary threshold
			},
			remediation: "Mount secrets as files instead of environment variables where possible",
		},
	}

	for _, c := range checks {
		check := CISBenchmarkCheck{
			ID:          c.id,
			Description: c.description,
			Remediation: c.remediation,
		}

		if c.check() {
			check.Status = "PASS"
			benchmark.PassCount++
		} else {
			check.Status = "FAIL"
			benchmark.FailCount++
		}

		benchmark.TotalChecks++

		// Add to appropriate section
		sectionID := c.id[:3]
		found := false
		for i := range benchmark.Sections {
			if benchmark.Sections[i].ID == sectionID {
				benchmark.Sections[i].Checks = append(benchmark.Sections[i].Checks, check)
				if check.Status == "PASS" {
					benchmark.Sections[i].PassCount++
				} else {
					benchmark.Sections[i].FailCount++
				}
				found = true
				break
			}
		}
		if !found {
			benchmark.Sections = append(benchmark.Sections, CISBenchmarkSection{
				ID:        sectionID,
				Name:      getSectionName(sectionID),
				PassCount: boolToInt(check.Status == "PASS"),
				FailCount: boolToInt(check.Status == "FAIL"),
				Checks:    []CISBenchmarkCheck{check},
			})
		}
	}

	if benchmark.TotalChecks > 0 {
		benchmark.Score = float64(benchmark.PassCount) / float64(benchmark.TotalChecks) * 100
	}

	return benchmark, nil
}

// calculateScore calculates overall security score
func (s *Scanner) calculateScore(result *ScanResult) float64 {
	score := 100.0

	// Deduct for image vulnerabilities
	if result.ImageVulns != nil {
		score -= float64(result.ImageVulns.CriticalCount) * 5
		score -= float64(result.ImageVulns.HighCount) * 2
		score -= float64(result.ImageVulns.MediumCount) * 0.5
	}

	// Deduct for pod security issues
	for _, issue := range result.PodSecurityIssues {
		switch issue.Severity {
		case "CRITICAL":
			score -= 10
		case "HIGH":
			score -= 5
		case "MEDIUM":
			score -= 2
		case "LOW":
			score -= 0.5
		}
	}

	// Deduct for RBAC issues
	for _, issue := range result.RBACIssues {
		switch issue.Severity {
		case "CRITICAL":
			score -= 15
		case "HIGH":
			score -= 7
		case "MEDIUM":
			score -= 3
		}
	}

	// Deduct for network issues
	for _, issue := range result.NetworkIssues {
		switch issue.Severity {
		case "HIGH":
			score -= 5
		case "MEDIUM":
			score -= 2
		case "LOW":
			score -= 1
		}
	}

	// Factor in CIS benchmark
	if result.CISBenchmark != nil {
		// Weight CIS score at 20% of total
		cisWeight := result.CISBenchmark.Score * 0.2
		score = (score * 0.8) + cisWeight
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// determineRiskLevel determines overall risk level
func (s *Scanner) determineRiskLevel(score float64) string {
	switch {
	case score >= 90:
		return "Low"
	case score >= 70:
		return "Medium"
	case score >= 50:
		return "High"
	default:
		return "Critical"
	}
}

// generateRecommendations generates security recommendations
func (s *Scanner) generateRecommendations(result *ScanResult) []SecurityRecommendation {
	var recommendations []SecurityRecommendation

	// Critical vulnerabilities
	if result.ImageVulns != nil && result.ImageVulns.CriticalCount > 0 {
		recommendations = append(recommendations, SecurityRecommendation{
			Priority:    1,
			Category:    "Image Security",
			Title:       "Fix Critical Vulnerabilities",
			Description: fmt.Sprintf("%d critical vulnerabilities found in container images", result.ImageVulns.CriticalCount),
			Impact:      "Critical vulnerabilities can be exploited for remote code execution or privilege escalation",
			Remediation: "Update base images and packages to patched versions. Run 'trivy image <image>' for details",
		})
	}

	// Privileged containers
	privilegedCount := 0
	for _, issue := range result.PodSecurityIssues {
		if strings.Contains(issue.Issue, "privileged") {
			privilegedCount++
		}
	}
	if privilegedCount > 0 {
		recommendations = append(recommendations, SecurityRecommendation{
			Priority:    1,
			Category:    "Pod Security",
			Title:       "Remove Privileged Containers",
			Description: fmt.Sprintf("%d containers running in privileged mode", privilegedCount),
			Impact:      "Privileged containers have root access to the host and can escape container isolation",
			Remediation: "Review each privileged container and remove the privilege unless absolutely required",
		})
	}

	// RBAC cluster-admin
	for _, issue := range result.RBACIssues {
		if strings.Contains(issue.Issue, "cluster-admin") {
			recommendations = append(recommendations, SecurityRecommendation{
				Priority:    2,
				Category:    "RBAC",
				Title:       "Minimize cluster-admin Usage",
				Description: issue.Issue,
				Impact:      "cluster-admin grants unlimited access to all cluster resources",
				Remediation: "Create custom roles with only required permissions using the principle of least privilege",
			})
			break
		}
	}

	// Network policies
	networkPolicyMissing := 0
	for _, issue := range result.NetworkIssues {
		if strings.Contains(issue.Issue, "NetworkPolicies") {
			networkPolicyMissing++
		}
	}
	if networkPolicyMissing > 0 {
		recommendations = append(recommendations, SecurityRecommendation{
			Priority:    3,
			Category:    "Network Security",
			Title:       "Implement Network Policies",
			Description: fmt.Sprintf("%d namespaces without NetworkPolicies", networkPolicyMissing),
			Impact:      "Without NetworkPolicies, all pod-to-pod traffic is allowed, increasing blast radius of compromised pods",
			Remediation: "Create default-deny NetworkPolicies and explicitly allow required traffic",
		})
	}

	// CIS benchmark failures
	if result.CISBenchmark != nil && result.CISBenchmark.FailCount > 0 {
		recommendations = append(recommendations, SecurityRecommendation{
			Priority:    3,
			Category:    "CIS Benchmark",
			Title:       "Address CIS Benchmark Failures",
			Description: fmt.Sprintf("%d CIS benchmark checks failed", result.CISBenchmark.FailCount),
			Impact:      "CIS benchmarks represent security best practices for Kubernetes",
			Remediation: "Review failed checks and implement recommended remediations",
		})
	}

	return recommendations
}

// Helper functions

func isSystemNamespace(ns string) bool {
	systemNS := map[string]bool{
		"kube-system":     true,
		"kube-public":     true,
		"kube-node-lease": true,
		"default":         false, // default is not system
	}
	return systemNS[ns]
}

func isDangerousCapability(cap string) bool {
	dangerous := map[string]bool{
		"SYS_ADMIN":    true,
		"NET_ADMIN":    true,
		"SYS_PTRACE":   true,
		"SYS_RAWIO":    true,
		"DAC_OVERRIDE": true,
		"SETUID":       true,
		"SETGID":       true,
		"NET_RAW":      true,
		"ALL":          true,
	}
	return dangerous[strings.ToUpper(cap)]
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func getSectionName(id string) string {
	names := map[string]string{
		"5.1": "RBAC and Service Accounts",
		"5.2": "Pod Security Standards",
		"5.3": "Network Policies and CNI Security",
		"5.4": "Secrets Management",
		"5.5": "Extensible Admission Control",
		"5.7": "General Policies",
	}
	if name, ok := names[id]; ok {
		return name
	}
	return "General Security"
}

// TrivyAvailable returns whether trivy is available
func (s *Scanner) TrivyAvailable() bool {
	return s.trivyPath != ""
}

// KubeBenchAvailable returns whether kube-bench is available
func (s *Scanner) KubeBenchAvailable() bool {
	return s.kubeBenchAvailable
}

// ScanImage scans a single image (public API)
func (s *Scanner) ScanImage(ctx context.Context, image string) ([]Vulnerability, error) {
	if s.trivyPath == "" {
		return nil, fmt.Errorf("trivy not available")
	}
	return s.scanImageWithTrivy(ctx, image)
}

// QuickScan performs a quick security assessment without image scanning
func (s *Scanner) QuickScan(ctx context.Context, namespace string) (*ScanResult, error) {
	startTime := time.Now()

	result := &ScanResult{
		ScanTime: startTime,
	}

	if contextName, err := s.k8sClient.GetCurrentContext(); err == nil {
		result.ClusterName = contextName
	}

	// Only run quick checks
	var wg sync.WaitGroup
	var mu sync.Mutex

	wg.Add(3)

	go func() {
		defer wg.Done()
		if issues, err := s.checkPodSecurity(ctx, namespace); err == nil {
			mu.Lock()
			result.PodSecurityIssues = issues
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		if issues, err := s.checkRBAC(ctx, namespace); err == nil {
			mu.Lock()
			result.RBACIssues = issues
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		if issues, err := s.checkNetwork(ctx, namespace); err == nil {
			mu.Lock()
			result.NetworkIssues = issues
			mu.Unlock()
		}
	}()

	wg.Wait()

	result.OverallScore = s.calculateScore(result)
	result.RiskLevel = s.determineRiskLevel(result.OverallScore)
	result.Recommendations = s.generateRecommendations(result)
	result.Duration = time.Since(startTime).String()

	return result, nil
}

// ValidateImage checks if image reference is valid
func ValidateImage(image string) bool {
	// Basic validation: image should not be empty and should have valid format
	if image == "" {
		return false
	}
	// Simple regex for image format
	pattern := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]*[a-zA-Z0-9](:[a-zA-Z0-9._-]+)?(@sha256:[a-f0-9]{64})?$`)
	return pattern.MatchString(image)
}
