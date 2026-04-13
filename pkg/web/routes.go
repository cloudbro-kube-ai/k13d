package web

import "net/http"

// registerPublicRoutes sets up unauthenticated endpoints (health, version, auth flow).
func (s *Server) registerPublicRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", withRecovery(s.handleHealth))
	mux.HandleFunc("/api/version", s.handleVersion)

	// Authentication (public for login/logout flow)
	mux.HandleFunc("/api/auth/login", s.authManager.HandleLogin)
	mux.HandleFunc("/api/auth/logout", s.authManager.HandleLogout)
	mux.HandleFunc("/api/auth/kubeconfig", s.authManager.HandleKubeconfigLogin)
	mux.HandleFunc("/api/auth/status", s.authManager.HandleAuthStatus)
	mux.HandleFunc("/api/auth/csrf-token", s.authManager.HandleCSRFToken)

	// OIDC/SSO (public for OAuth flow)
	mux.HandleFunc("/api/auth/oidc/login", s.authManager.HandleOIDCLogin)
	mux.HandleFunc("/api/auth/oidc/callback", s.authManager.HandleOIDCCallback)
	mux.HandleFunc("/api/auth/oidc/status", s.authManager.HandleOIDCStatus)
	mux.HandleFunc("/api/auth/ldap/status", s.authManager.AuthMiddleware(s.authManager.AdminMiddleware(s.authManager.HandleLDAPStatus)))
	mux.HandleFunc("/api/auth/ldap/test", s.authManager.AuthMiddleware(s.authManager.AdminMiddleware(s.authManager.HandleLDAPTest)))

	// Prometheus scrape endpoint (no auth for scraping)
	if s.cfg.Prometheus.ExposeMetrics {
		mux.HandleFunc("/metrics", s.handlePrometheusMetrics)
	}
}

// registerAuthRoutes sets up authenticated user routes (current user, permissions, roles).
func (s *Server) registerAuthRoutes(mux *http.ServeMux) {
	auth := s.authManager.AuthMiddleware

	mux.HandleFunc("/api/auth/me", auth(s.authManager.HandleCurrentUser))
	mux.HandleFunc("/api/auth/permissions", auth(s.handleUserPermissions))

	// Role management
	mux.HandleFunc("/api/roles", auth(s.handleRoles))
	mux.HandleFunc("/api/roles/", auth(s.authManager.AdminMiddleware(s.handleRoleByName)))
}

// registerAIRoutes sets up AI assistant, LLM, MCP, and session routes.
func (s *Server) registerAIRoutes(mux *http.ServeMux) {
	auth := s.authManager.AuthMiddleware
	aiFeature := s.authorizer.FeatureMiddleware(FeatureAIAssistant)

	// AI chat and tool approval (feature-gated)
	mux.HandleFunc("/api/chat/agentic", auth(aiFeature(s.handleAgenticChat)))
	mux.HandleFunc("/api/tool/approve", auth(aiFeature(s.handleToolApprove)))

	// AI session management
	mux.HandleFunc("/api/sessions", auth(s.handleSessions))
	mux.HandleFunc("/api/sessions/", auth(s.handleSession))

	// AI / LLM configuration and status
	mux.HandleFunc("/api/settings", auth(s.handleSettings))
	mux.HandleFunc("/api/settings/llm", auth(s.handleLLMSettings))
	mux.HandleFunc("/api/settings/agent", auth(s.handleAgentSettings))
	mux.HandleFunc("/api/settings/tool-approval", auth(s.handleToolApprovalSettings))
	mux.HandleFunc("/api/llm/test", auth(s.handleLLMTest))
	mux.HandleFunc("/api/llm/status", auth(s.handleLLMStatus))
	mux.HandleFunc("/api/ai/ping", auth(s.handleAIPing))
	mux.HandleFunc("/api/llm/ollama/status", auth(s.handleOllamaStatus))
	mux.HandleFunc("/api/llm/ollama/pull", auth(s.handleOllamaPull))
	mux.HandleFunc("/api/llm/usage", auth(s.handleLLMUsage))
	mux.HandleFunc("/api/llm/usage/stats", auth(s.handleLLMUsageStats))
	mux.HandleFunc("/api/llm/available-models", auth(s.handleAvailableModels))
	mux.HandleFunc("/api/models", auth(s.handleModels))
	mux.HandleFunc("/api/models/active", auth(s.handleActiveModel))

	// MCP server management
	mux.HandleFunc("/api/mcp/servers", auth(s.handleMCPServers))
	mux.HandleFunc("/api/mcp/tools", auth(s.handleMCPTools))

	// AI troubleshooting
	mux.HandleFunc("/api/troubleshoot", auth(s.handleTroubleshoot))
}

// registerK8sRoutes sets up Kubernetes resource read routes, cluster overview, and search.
func (s *Server) registerK8sRoutes(mux *http.ServeMux) {
	auth := s.authManager.AuthMiddleware
	view := s.authorizer.AuthzMiddleware("*", ActionView)
	apply := s.authorizer.AuthzMiddleware("*", ActionApply)

	mux.HandleFunc("/api/k8s/apply", auth(apply(s.handleYamlApply)))
	mux.HandleFunc("/api/k8s/", auth(view(s.handleK8sResource)))
	mux.HandleFunc("/api/crd/", auth(view(s.handleCustomResources)))
	mux.HandleFunc("/api/overview", auth(s.handleClusterOverview))
	mux.HandleFunc("/api/applications", auth(s.handleApplications))
	mux.HandleFunc("/api/cost", auth(s.handleCostEstimate))
	mux.HandleFunc("/api/search", auth(s.handleGlobalSearch))
	mux.HandleFunc("/api/safety/analyze", auth(s.handleSafetyAnalysis))
	mux.HandleFunc("/api/pulse", auth(s.handlePulse))
	mux.HandleFunc("/api/xray", auth(s.handleXRay))
	mux.HandleFunc("/api/diff", auth(s.handleResourceDiff))
	mux.HandleFunc("/api/healing/rules", auth(s.handleHealingRules))
	mux.HandleFunc("/api/healing/events", auth(s.handleHealingEvents))
	mux.HandleFunc("/api/resource/references", auth(s.handleResourceReferences))

	// Multi-cluster context management
	mux.HandleFunc("/api/contexts", auth(s.handleContexts))
	mux.HandleFunc("/api/contexts/switch", auth(s.authorizer.AuthzMiddleware("*", ActionEdit)(s.handleContextSwitch)))

	// Pod operations
	mux.HandleFunc("/api/pods/", auth(s.handlePodLogs))
	mux.HandleFunc("/api/workload/pods", auth(s.handleWorkloadPods))

	// Event timeline (feature-gated)
	mux.HandleFunc("/api/events/timeline", auth(s.authorizer.FeatureMiddleware(FeatureEventTimeline)(s.handleEventTimeline)))

	// Topology (feature-gated)
	mux.HandleFunc("/api/topology/", auth(s.authorizer.FeatureMiddleware(FeatureTopology)(s.handleTopology)))

	// WebSocket terminal (feature-gated)
	terminalHandler := NewTerminalHandler(s.k8sClient)
	mux.HandleFunc("/api/terminal/", auth(s.authorizer.FeatureMiddleware(FeatureTerminal)(terminalHandler.HandleTerminal)))
	mux.HandleFunc("/api/tui/shell", auth(s.authorizer.FeatureMiddleware(FeatureHostTerminal)(s.HandleTUIShell)))

	// GitOps status (ArgoCD / Flux)
	mux.HandleFunc("/api/gitops/status", auth(s.handleGitOpsStatus))

	// Velero backup status
	mux.HandleFunc("/api/velero/backups", auth(s.handleVeleroBackups))
	mux.HandleFunc("/api/velero/schedules", auth(s.handleVeleroSchedules))

	// Notification webhook configuration
	mux.HandleFunc("/api/notifications/config", auth(s.authorizer.AuthzMiddleware("*", ActionEdit)(s.handleNotificationConfig)))
	mux.HandleFunc("/api/notifications/test", auth(s.authorizer.AuthzMiddleware("*", ActionEdit)(s.handleNotificationTest)))
	mux.HandleFunc("/api/notifications/history", auth(s.handleNotificationHistory))
	mux.HandleFunc("/api/notifications/status", auth(s.handleNotificationStatus))
}

// registerWorkloadOperationRoutes sets up RBAC-protected mutation routes
// for deployments, statefulsets, daemonsets, cronjobs, nodes, and port forwarding.
func (s *Server) registerWorkloadOperationRoutes(mux *http.ServeMux) {
	auth := s.authManager.AuthMiddleware

	// Deployment operations
	mux.HandleFunc("/api/deployment/scale", auth(s.authorizer.AuthzMiddleware("deployments", ActionScale)(s.handleDeploymentScale)))
	mux.HandleFunc("/api/deployment/restart", auth(s.authorizer.AuthzMiddleware("deployments", ActionRestart)(s.handleDeploymentRestart)))
	mux.HandleFunc("/api/deployment/pause", auth(s.authorizer.AuthzMiddleware("deployments", ActionEdit)(s.handleDeploymentPause)))
	mux.HandleFunc("/api/deployment/resume", auth(s.authorizer.AuthzMiddleware("deployments", ActionEdit)(s.handleDeploymentResume)))
	mux.HandleFunc("/api/deployment/rollback", auth(s.authorizer.AuthzMiddleware("deployments", ActionEdit)(s.handleDeploymentRollback)))
	mux.HandleFunc("/api/deployment/history", auth(s.handleDeploymentHistory))

	// StatefulSet operations
	mux.HandleFunc("/api/statefulset/scale", auth(s.authorizer.AuthzMiddleware("statefulsets", ActionScale)(s.handleStatefulSetScale)))
	mux.HandleFunc("/api/statefulset/restart", auth(s.authorizer.AuthzMiddleware("statefulsets", ActionRestart)(s.handleStatefulSetRestart)))

	// DaemonSet operations
	mux.HandleFunc("/api/daemonset/restart", auth(s.authorizer.AuthzMiddleware("daemonsets", ActionRestart)(s.handleDaemonSetRestart)))

	// CronJob operations
	mux.HandleFunc("/api/cronjob/trigger", auth(s.authorizer.AuthzMiddleware("cronjobs", ActionCreate)(s.handleCronJobTrigger)))
	mux.HandleFunc("/api/cronjob/suspend", auth(s.authorizer.AuthzMiddleware("cronjobs", ActionEdit)(s.handleCronJobSuspend)))

	// Node operations
	mux.HandleFunc("/api/node/cordon", auth(s.authorizer.AuthzMiddleware("nodes", ActionEdit)(s.handleNodeCordon)))
	mux.HandleFunc("/api/node/drain", auth(s.authorizer.AuthzMiddleware("nodes", ActionEdit)(s.handleNodeDrain)))
	mux.HandleFunc("/api/node/pods", auth(s.handleNodePods))

	// Port forwarding
	mux.HandleFunc("/api/portforward/start", auth(s.authorizer.AuthzMiddleware("pods", ActionPortForward)(s.handlePortForwardStart)))
	mux.HandleFunc("/api/portforward/list", auth(s.handlePortForwardList))
	mux.HandleFunc("/api/portforward/", auth(s.handlePortForwardStop))
}

// registerHelmRoutes sets up Helm release management routes (feature-gated).
func (s *Server) registerHelmRoutes(mux *http.ServeMux) {
	auth := s.authManager.AuthMiddleware
	helm := s.authorizer.FeatureMiddleware(FeatureHelmManagement)

	mux.HandleFunc("/api/helm/releases", auth(helm(s.handleHelmReleases)))
	mux.HandleFunc("/api/helm/release/", auth(helm(s.handleHelmRelease)))
	mux.HandleFunc("/api/helm/install", auth(helm(s.authorizer.AuthzMiddleware("helm", ActionCreate)(s.handleHelmInstall))))
	mux.HandleFunc("/api/helm/upgrade", auth(helm(s.authorizer.AuthzMiddleware("helm", ActionEdit)(s.handleHelmUpgrade))))
	mux.HandleFunc("/api/helm/uninstall", auth(helm(s.authorizer.AuthzMiddleware("helm", ActionDelete)(s.handleHelmUninstall))))
	mux.HandleFunc("/api/helm/rollback", auth(helm(s.authorizer.AuthzMiddleware("helm", ActionEdit)(s.handleHelmRollback))))
	mux.HandleFunc("/api/helm/repos", auth(helm(s.handleHelmRepos)))
	mux.HandleFunc("/api/helm/search", auth(helm(s.handleHelmSearch)))
}

// registerMetricsRoutes sets up metrics, Prometheus, and audit/report routes.
func (s *Server) registerMetricsRoutes(mux *http.ServeMux) {
	auth := s.authManager.AuthMiddleware
	metrics := s.authorizer.FeatureMiddleware(FeatureMetrics)

	mux.HandleFunc("/api/metrics/pods", auth(metrics(s.handlePodMetrics)))
	mux.HandleFunc("/api/metrics/nodes", auth(metrics(s.handleNodeMetrics)))
	mux.HandleFunc("/api/metrics/history/cluster", auth(metrics(s.handleClusterMetricsHistory)))
	mux.HandleFunc("/api/metrics/history/nodes", auth(metrics(s.handleNodeMetricsHistory)))
	mux.HandleFunc("/api/metrics/history/pods", auth(metrics(s.handlePodMetricsHistory)))
	mux.HandleFunc("/api/metrics/history/summary", auth(metrics(s.handleMetricsSummary)))
	mux.HandleFunc("/api/metrics/history/aggregated", auth(metrics(s.handleAggregatedMetrics)))
	mux.HandleFunc("/api/metrics/collect", auth(metrics(s.handleMetricsCollectNow)))

	// Prometheus
	mux.HandleFunc("/api/prometheus/settings", auth(s.handlePrometheusSettings))
	mux.HandleFunc("/api/prometheus/test", auth(s.handlePrometheusTest))
	mux.HandleFunc("/api/prometheus/query", auth(s.handlePrometheusQuery))

	// Audit and reports (feature-gated)
	mux.HandleFunc("/api/audit", auth(s.authorizer.FeatureMiddleware(FeatureAuditLogs)(s.handleAuditLogs)))
	mux.HandleFunc("/api/reports", auth(s.authorizer.FeatureMiddleware(FeatureReports)(s.reportGenerator.HandleReports)))
	mux.HandleFunc("/api/reports/preview", auth(s.authorizer.FeatureMiddleware(FeatureReports)(s.reportGenerator.HandleReportPreview)))
}

// registerSecurityRoutes sets up security scanning routes (feature-gated).
func (s *Server) registerSecurityRoutes(mux *http.ServeMux) {
	auth := s.authManager.AuthMiddleware
	sec := s.authorizer.FeatureMiddleware(FeatureSecurityScan)

	mux.HandleFunc("/api/security/scan", auth(sec(s.handleSecurityScan)))
	mux.HandleFunc("/api/security/scan/quick", auth(sec(s.handleSecurityQuickScan)))
	mux.HandleFunc("/api/security/scans", auth(sec(s.handleSecurityScanHistory)))
	mux.HandleFunc("/api/security/scans/stats", auth(sec(s.handleSecurityScanStats)))
	mux.HandleFunc("/api/security/scan/", auth(sec(s.handleSecurityScanDetail)))
	mux.HandleFunc("/api/security/trivy/status", auth(sec(s.handleTrivyStatus)))
	mux.HandleFunc("/api/security/trivy/install", auth(sec(s.handleTrivyInstall)))
	mux.HandleFunc("/api/security/trivy/instructions", auth(sec(s.handleTrivyInstructions)))
}

// registerVisualizationRoutes sets up RBAC and network policy visualization routes.
func (s *Server) registerVisualizationRoutes(mux *http.ServeMux) {
	auth := s.authManager.AuthMiddleware

	mux.HandleFunc("/api/rbac/visualization", auth(s.handleRBACVisualization))
	mux.HandleFunc("/api/rbac/subject/detail", auth(s.handleRBACSubjectDetail))
	mux.HandleFunc("/api/netpol/visualization", auth(s.handleNetworkPolicyVisualization))
}

// registerAdminRoutes sets up admin-only user management and access request routes.
func (s *Server) registerAdminRoutes(mux *http.ServeMux) {
	auth := s.authManager.AuthMiddleware
	admin := s.authManager.AdminMiddleware

	mux.HandleFunc("/api/admin/users", auth(admin(s.handleAdminUsers)))
	mux.HandleFunc("/api/admin/users/", auth(admin(s.handleAdminUserAction)))
	mux.HandleFunc("/api/admin/reset-password", auth(admin(s.authManager.HandleResetPassword)))
	mux.HandleFunc("/api/admin/status", auth(admin(s.authManager.HandleAuthStatus)))
	mux.HandleFunc("/api/admin/lock", auth(admin(s.authManager.HandleLockUser)))
	mux.HandleFunc("/api/admin/unlock", auth(admin(s.authManager.HandleUnlockUser)))

	// Access request workflow (Teleport-inspired)
	mux.HandleFunc("/api/access/request", auth(s.accessRequestManager.HandleCreateAccessRequest))
	mux.HandleFunc("/api/access/requests", auth(s.accessRequestManager.HandleListAccessRequests))
	mux.HandleFunc("/api/access/approve/", auth(admin(s.accessRequestManager.HandleApproveAccessRequest)))
	mux.HandleFunc("/api/access/deny/", auth(admin(s.accessRequestManager.HandleDenyAccessRequest)))
}
