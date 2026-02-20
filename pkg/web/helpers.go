package web

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai/safety"
	"github.com/cloudbro-kube-ai/k13d/pkg/db"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ==========================================
// String Helpers
// ==========================================

// truncateString truncates a string to maxLen and adds "..."
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ==========================================
// Pod Helpers
// ==========================================

// getPodReadyCount returns the ready container count as "ready/total"
func getPodReadyCount(pod *corev1.Pod) string {
	ready := 0
	total := len(pod.Status.ContainerStatuses)
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Ready {
			ready++
		}
	}
	return fmt.Sprintf("%d/%d", ready, total)
}

// getPodRestarts returns the total restart count for all containers
func getPodRestarts(pod *corev1.Pod) int32 {
	var restarts int32
	for _, cs := range pod.Status.ContainerStatuses {
		restarts += cs.RestartCount
	}
	return restarts
}

// ==========================================
// Service Helpers
// ==========================================

// getExternalIPs returns external IPs for a service
func getExternalIPs(svc *corev1.Service) string {
	if len(svc.Status.LoadBalancer.Ingress) > 0 {
		ips := make([]string, len(svc.Status.LoadBalancer.Ingress))
		for i, ing := range svc.Status.LoadBalancer.Ingress {
			if ing.IP != "" {
				ips[i] = ing.IP
			} else {
				ips[i] = ing.Hostname
			}
		}
		return strings.Join(ips, ", ")
	}
	if len(svc.Spec.ExternalIPs) > 0 {
		return strings.Join(svc.Spec.ExternalIPs, ", ")
	}
	return "<none>"
}

// ==========================================
// Node Helpers
// ==========================================

// getNodeStatus returns the node status (Ready/NotReady)
func getNodeStatus(node *corev1.Node) string {
	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			if cond.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

// getNodeRoles returns the node roles from labels
func getNodeRoles(node *corev1.Node) string {
	roles := []string{}
	for label := range node.Labels {
		if strings.HasPrefix(label, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
			roles = append(roles, role)
		}
	}
	if len(roles) == 0 {
		return "<none>"
	}
	return strings.Join(roles, ", ")
}

// formatNodeSelector formats a node selector map to string
func formatNodeSelector(selector map[string]string) string {
	if len(selector) == 0 {
		return "<none>"
	}
	pairs := []string{}
	for k, v := range selector {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(pairs, ", ")
}

// ==========================================
// Ingress Helpers
// ==========================================

// getIngressClass returns the ingress class name
func getIngressClass(ing *networkingv1.Ingress) string {
	if ing.Spec.IngressClassName != nil {
		return *ing.Spec.IngressClassName
	}
	if class, ok := ing.Annotations["kubernetes.io/ingress.class"]; ok {
		return class
	}
	return "<none>"
}

// getIngressAddress returns the ingress load balancer address
func getIngressAddress(ing *networkingv1.Ingress) string {
	if len(ing.Status.LoadBalancer.Ingress) > 0 {
		addrs := []string{}
		for _, addr := range ing.Status.LoadBalancer.Ingress {
			if addr.IP != "" {
				addrs = append(addrs, addr.IP)
			} else if addr.Hostname != "" {
				addrs = append(addrs, addr.Hostname)
			}
		}
		return strings.Join(addrs, ", ")
	}
	return "<pending>"
}

// ==========================================
// Storage Helpers
// ==========================================

// getAccessModes converts access modes to abbreviated strings
func getAccessModes(modes []corev1.PersistentVolumeAccessMode) string {
	modeStrs := []string{}
	for _, m := range modes {
		switch m {
		case corev1.ReadWriteOnce:
			modeStrs = append(modeStrs, "RWO")
		case corev1.ReadOnlyMany:
			modeStrs = append(modeStrs, "ROX")
		case corev1.ReadWriteMany:
			modeStrs = append(modeStrs, "RWX")
		case corev1.ReadWriteOncePod:
			modeStrs = append(modeStrs, "RWOP")
		default:
			modeStrs = append(modeStrs, string(m))
		}
	}
	return strings.Join(modeStrs, ", ")
}

// getStorageClass returns the storage class name
func getStorageClass(sc *string) string {
	if sc != nil {
		return *sc
	}
	return "<default>"
}

// getReclaimPolicy returns the reclaim policy string
func getReclaimPolicy(policy *corev1.PersistentVolumeReclaimPolicy) string {
	if policy != nil {
		return string(*policy)
	}
	return "Delete"
}

// ==========================================
// Job Helpers
// ==========================================

// getJobCompletions returns completions as "succeeded/total"
func getJobCompletions(job *batchv1.Job) string {
	completions := int32(1)
	if job.Spec.Completions != nil {
		completions = *job.Spec.Completions
	}
	return fmt.Sprintf("%d/%d", job.Status.Succeeded, completions)
}

// getJobDuration returns the job duration
func getJobDuration(job *batchv1.Job) string {
	if job.Status.StartTime == nil {
		return "<pending>"
	}
	if job.Status.CompletionTime != nil {
		return formatDuration(job.Status.CompletionTime.Sub(job.Status.StartTime.Time))
	}
	return formatDuration(time.Since(job.Status.StartTime.Time))
}

// ==========================================
// Time Helpers
// ==========================================

// formatDuration formats a duration into a human-readable string.
// For durations >= 24h, shows days (e.g., "21d9h").
// For smaller durations, shows hours/minutes (e.g., "5h30m", "45m20s").
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		if hours > 0 {
			return fmt.Sprintf("%dd%dh", days, hours)
		}
		return fmt.Sprintf("%dd", days)
	}
	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%dh%dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}
	if minutes > 0 {
		if seconds > 0 {
			return fmt.Sprintf("%dm%ds", minutes, seconds)
		}
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%ds", seconds)
}

// ==========================================
// HPA Helpers
// ==========================================

// getMinReplicas returns the minimum replicas, defaulting to 1
func getMinReplicas(min *int32) int32 {
	if min != nil {
		return *min
	}
	return 1
}

// formatLabelSelector formats a label selector to a query string for use in API calls
func formatLabelSelector(selector *metav1.LabelSelector) string {
	if selector == nil {
		return ""
	}
	if len(selector.MatchLabels) == 0 && len(selector.MatchExpressions) == 0 {
		return ""
	}
	pairs := []string{}
	// Handle matchLabels - format as key=value for label selector query
	for k, v := range selector.MatchLabels {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	// Handle matchExpressions
	for _, expr := range selector.MatchExpressions {
		switch expr.Operator {
		case metav1.LabelSelectorOpIn:
			pairs = append(pairs, fmt.Sprintf("%s in (%s)", expr.Key, strings.Join(expr.Values, ",")))
		case metav1.LabelSelectorOpNotIn:
			pairs = append(pairs, fmt.Sprintf("%s notin (%s)", expr.Key, strings.Join(expr.Values, ",")))
		case metav1.LabelSelectorOpExists:
			pairs = append(pairs, expr.Key)
		case metav1.LabelSelectorOpDoesNotExist:
			pairs = append(pairs, fmt.Sprintf("!%s", expr.Key))
		}
	}
	return strings.Join(pairs, ",")
}

// ==========================================
// Command Classification
// ==========================================

// classifyCommand categorizes a kubectl command for safety.
// This function now uses the unified safety.Classifier which provides
// consistent classification across TUI and Web UI, including detection
// of piped commands, chained commands, and file redirects.
//
// Deprecated: For new code, use safety.Classify() directly for full classification
// or safety.Evaluate() for policy-based decisions.
func classifyCommand(command string) string {
	// Use unified classifier from safety package
	classification := safety.Classify(command)
	return classification.Category
}

// ==========================================
// Audit Logging Helpers
// ==========================================

// getK8sContextInfo retrieves current k8s context, cluster, and user info
func (s *Server) getK8sContextInfo() (context, cluster, user string) {
	if s.k8sClient == nil {
		return "", "", ""
	}
	ctx, cl, usr, err := s.k8sClient.GetContextInfo()
	if err != nil {
		return "", "", ""
	}
	return ctx, cl, usr
}

// recordAuditWithK8sContext records an audit entry with k8s context info
func (s *Server) recordAuditWithK8sContext(r *http.Request, entry db.AuditEntry) {
	// Get username from header
	if entry.User == "" {
		entry.User = r.Header.Get("X-Username")
		if entry.User == "" {
			entry.User = "anonymous"
		}
	}

	// Get k8s context info
	k8sContext, k8sCluster, k8sUser := s.getK8sContextInfo()
	entry.K8sContext = k8sContext
	entry.K8sCluster = k8sCluster
	entry.K8sUser = k8sUser

	// Set source
	entry.Source = "web"

	// Get client IP
	entry.ClientIP = getClientIP(r)

	// Get session ID from cookie
	if cookie, err := r.Cookie("session_id"); err == nil {
		entry.SessionID = cookie.Value
	}

	_ = db.RecordAudit(entry)
}

// getClientIP extracts client IP from request.
// WARNING: X-Forwarded-For and X-Real-IP headers can be spoofed by clients.
// This function should only be used for logging and rate limiting, NOT for
// security decisions, unless the server is behind a trusted reverse proxy
// that strips and re-sets these headers.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colonIdx := strings.LastIndex(ip, ":"); colonIdx != -1 {
		ip = ip[:colonIdx]
	}
	return ip
}
