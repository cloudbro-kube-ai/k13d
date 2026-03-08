package web

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	corev1 "k8s.io/api/core/v1"
)


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
