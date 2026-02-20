package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// TroubleshootFinding represents a single issue found during troubleshooting
type TroubleshootFinding struct {
	Resource  string `json:"resource"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Issue     string `json:"issue"`
	Severity  string `json:"severity"` // "critical", "warning", "info"
	Details   string `json:"details,omitempty"`
}

// TroubleshootReport is the response for the troubleshoot endpoint
type TroubleshootReport struct {
	Namespace       string                `json:"namespace"`
	Findings        []TroubleshootFinding `json:"findings"`
	Recommendations []string              `json:"recommendations"`
	Severity        string                `json:"severity"` // overall: "critical", "warning", "healthy"
	Timestamp       time.Time             `json:"timestamp"`
}

func (s *Server) handleTroubleshoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Namespace string `json:"namespace"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	namespace := req.Namespace
	if namespace == "" {
		namespace = "default"
	}

	ctx := r.Context()

	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		pods   []corev1.Pod
		events []corev1.Event
		quotas []corev1.ResourceQuota
	)

	wg.Add(3)
	go func() {
		defer wg.Done()
		var err error
		pods, err = s.k8sClient.ListPods(ctx, namespace)
		if err != nil {
			pods = nil
		}
	}()
	go func() {
		defer wg.Done()
		var err error
		events, err = s.k8sClient.ListEvents(ctx, namespace)
		if err != nil {
			events = nil
		}
	}()
	go func() {
		defer wg.Done()
		var err error
		quotas, err = s.k8sClient.ListResourceQuotas(ctx, namespace)
		if err != nil {
			quotas = nil
		}
	}()
	wg.Wait()

	var findings []TroubleshootFinding
	var recommendations []string
	overallSeverity := "healthy"

	addFinding := func(f TroubleshootFinding) {
		mu.Lock()
		defer mu.Unlock()
		findings = append(findings, f)
		if f.Severity == "critical" {
			overallSeverity = "critical"
		} else if f.Severity == "warning" && overallSeverity != "critical" {
			overallSeverity = "warning"
		}
	}

	// Check pods for issues
	for _, pod := range pods {
		// CrashLoopBackOff
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
				addFinding(TroubleshootFinding{
					Resource:  "Pod",
					Namespace: pod.Namespace,
					Name:      pod.Name,
					Issue:     "CrashLoopBackOff",
					Severity:  "critical",
					Details:   fmt.Sprintf("Container %s is crash-looping (restarts: %d)", cs.Name, cs.RestartCount),
				})
			}

			// OOMKilled
			if cs.LastTerminationState.Terminated != nil && cs.LastTerminationState.Terminated.Reason == "OOMKilled" {
				addFinding(TroubleshootFinding{
					Resource:  "Pod",
					Namespace: pod.Namespace,
					Name:      pod.Name,
					Issue:     "OOMKilled",
					Severity:  "critical",
					Details:   fmt.Sprintf("Container %s was killed due to out-of-memory", cs.Name),
				})
			}

			// High restart count
			if cs.RestartCount > 5 {
				addFinding(TroubleshootFinding{
					Resource:  "Pod",
					Namespace: pod.Namespace,
					Name:      pod.Name,
					Issue:     "High Restart Count",
					Severity:  "warning",
					Details:   fmt.Sprintf("Container %s has %d restarts", cs.Name, cs.RestartCount),
				})
			}
		}

		// Pending pods
		if pod.Status.Phase == corev1.PodPending {
			reason := "Unknown"
			for _, cond := range pod.Status.Conditions {
				if cond.Status == corev1.ConditionFalse {
					reason = string(cond.Reason)
					break
				}
			}
			addFinding(TroubleshootFinding{
				Resource:  "Pod",
				Namespace: pod.Namespace,
				Name:      pod.Name,
				Issue:     "Pending",
				Severity:  "warning",
				Details:   fmt.Sprintf("Pod is stuck in Pending state (reason: %s)", reason),
			})
		}

		// Failed pods
		if pod.Status.Phase == corev1.PodFailed {
			addFinding(TroubleshootFinding{
				Resource:  "Pod",
				Namespace: pod.Namespace,
				Name:      pod.Name,
				Issue:     "Failed",
				Severity:  "critical",
				Details:   fmt.Sprintf("Pod has failed (reason: %s)", pod.Status.Reason),
			})
		}

		// ImagePullBackOff
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil && (cs.State.Waiting.Reason == "ImagePullBackOff" || cs.State.Waiting.Reason == "ErrImagePull") {
				addFinding(TroubleshootFinding{
					Resource:  "Pod",
					Namespace: pod.Namespace,
					Name:      pod.Name,
					Issue:     cs.State.Waiting.Reason,
					Severity:  "critical",
					Details:   fmt.Sprintf("Container %s cannot pull image: %s", cs.Name, cs.Image),
				})
			}
		}
	}

	// Check warning events
	recentCutoff := time.Now().Add(-1 * time.Hour)
	for _, event := range events {
		if event.Type == "Warning" && event.LastTimestamp.Time.After(recentCutoff) {
			addFinding(TroubleshootFinding{
				Resource:  event.InvolvedObject.Kind,
				Namespace: event.Namespace,
				Name:      event.InvolvedObject.Name,
				Issue:     event.Reason,
				Severity:  "warning",
				Details:   event.Message,
			})
		}
	}

	// Check resource quotas
	for _, quota := range quotas {
		for resource, used := range quota.Status.Used {
			if hard, ok := quota.Status.Hard[resource]; ok {
				usedVal := used.Value()
				hardVal := hard.Value()
				if hardVal > 0 && float64(usedVal)/float64(hardVal) > 0.9 {
					addFinding(TroubleshootFinding{
						Resource:  "ResourceQuota",
						Namespace: quota.Namespace,
						Name:      quota.Name,
						Issue:     "Quota Near Limit",
						Severity:  "warning",
						Details:   fmt.Sprintf("Resource %s is at %d/%d (>90%%)", resource, usedVal, hardVal),
					})
				}
			}
		}
	}

	// Generate recommendations based on findings
	hasCrashLoop := false
	hasOOM := false
	hasPending := false
	hasImagePull := false

	for _, f := range findings {
		switch f.Issue {
		case "CrashLoopBackOff":
			hasCrashLoop = true
		case "OOMKilled":
			hasOOM = true
		case "Pending":
			hasPending = true
		case "ImagePullBackOff", "ErrImagePull":
			hasImagePull = true
		}
	}

	if hasCrashLoop {
		recommendations = append(recommendations, "Check container logs for crash-looping pods: kubectl logs <pod> --previous")
	}
	if hasOOM {
		recommendations = append(recommendations, "Increase memory limits for OOM-killed containers or optimize memory usage")
	}
	if hasPending {
		recommendations = append(recommendations, "Check node resources and scheduling constraints for pending pods")
	}
	if hasImagePull {
		recommendations = append(recommendations, "Verify image names, tags, and registry credentials for image pull failures")
	}
	if len(findings) == 0 {
		recommendations = append(recommendations, "No issues found. Cluster namespace appears healthy.")
	}

	report := TroubleshootReport{
		Namespace:       namespace,
		Findings:        findings,
		Recommendations: recommendations,
		Severity:        overallSeverity,
		Timestamp:       time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}
