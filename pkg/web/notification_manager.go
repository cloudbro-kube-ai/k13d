package web

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/config"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
)

// NotificationManager watches K8s events and dispatches notifications.
type NotificationManager struct {
	k8sClient *k8s.Client
	cfg       *config.Config
	stopCh    chan struct{}
	running   bool
	mu        sync.RWMutex

	// Deduplication: hash → last-sent time
	sentEvents   map[string]time.Time
	sentEventsMu sync.Mutex

	// Recent notification history for UI
	history   []NotificationHistoryEntry
	historyMu sync.RWMutex

	httpClient *http.Client
}

// NotificationHistoryEntry records a sent notification.
type NotificationHistoryEntry struct {
	Timestamp time.Time `json:"timestamp"`
	EventType string    `json:"event_type"`
	Resource  string    `json:"resource"`
	Namespace string    `json:"namespace"`
	Message   string    `json:"message"`
	Provider  string    `json:"provider"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}

// NewNotificationManager creates a new notification manager.
func NewNotificationManager(k8sClient *k8s.Client, cfg *config.Config) *NotificationManager {
	return &NotificationManager{
		k8sClient:  k8sClient,
		cfg:        cfg,
		sentEvents: make(map[string]time.Time),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Start begins the event watching goroutines.
func (nm *NotificationManager) Start() {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	if nm.running {
		return
	}
	nm.running = true
	nm.stopCh = make(chan struct{})
	go nm.runWatcher()
	go nm.runCleanup()
}

// Stop halts the event watching.
func (nm *NotificationManager) Stop() {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	if !nm.running {
		return
	}
	nm.running = false
	close(nm.stopCh)
}

// IsRunning returns whether the manager is actively watching.
func (nm *NotificationManager) IsRunning() bool {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return nm.running
}

// Restart stops and starts the manager (used after config change).
func (nm *NotificationManager) Restart() {
	nm.Stop()
	nm.Start()
}

// GetHistory returns recent notification history.
func (nm *NotificationManager) GetHistory() []NotificationHistoryEntry {
	nm.historyMu.RLock()
	defer nm.historyMu.RUnlock()
	result := make([]NotificationHistoryEntry, len(nm.history))
	copy(result, nm.history)
	return result
}

// DedupCount returns the number of entries in the dedup map.
func (nm *NotificationManager) DedupCount() int {
	nm.sentEventsMu.Lock()
	defer nm.sentEventsMu.Unlock()
	return len(nm.sentEvents)
}

func (nm *NotificationManager) runWatcher() {
	interval := time.Duration(nm.cfg.Notifications.PollInterval) * time.Second
	if interval < 10*time.Second {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			nm.pollAndDispatch()
		case <-nm.stopCh:
			return
		}
	}
}

func (nm *NotificationManager) runCleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			nm.cleanupDedup()
		case <-nm.stopCh:
			return
		}
	}
}

func (nm *NotificationManager) pollAndDispatch() {
	ncfg := nm.cfg.Notifications
	if !ncfg.Enabled || len(ncfg.Events) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	events, err := nm.k8sClient.ListEvents(ctx, "")
	if err != nil {
		fmt.Printf("[notifications] Failed to list events: %v\n", err)
		return
	}

	cutoff := time.Now().Add(-2 * time.Duration(nm.cfg.Notifications.PollInterval) * time.Second)

	for i := range events {
		event := &events[i]
		eventTime := event.LastTimestamp.Time
		if eventTime.IsZero() {
			eventTime = event.CreationTimestamp.Time
		}
		if eventTime.Before(cutoff) {
			continue
		}

		notifType := classifyEvent(event)
		if notifType == "" {
			continue
		}
		if !containsString(ncfg.Events, notifType) {
			continue
		}

		eventKey := eventHash(event)
		if nm.wasRecentlySent(eventKey) {
			continue
		}

		err := nm.dispatch(&ncfg, event, notifType)
		nm.markSent(eventKey)
		nm.recordHistory(event, notifType, ncfg.Provider, err)
	}
}

// classifyEvent maps K8s event reasons to notification types.
func classifyEvent(event *corev1.Event) string {
	if event.Type != "Warning" {
		return ""
	}
	reason := event.Reason
	switch {
	case reason == "BackOff" || reason == "CrashLoopBackOff":
		return "pod_crash"
	case reason == "OOMKilling" || reason == "OOMKilled" ||
		strings.Contains(event.Message, "OOMKilled"):
		return "oom_killed"
	case reason == "NodeNotReady" || reason == "NodeHasDiskPressure" ||
		reason == "NodeHasMemoryPressure" ||
		(event.InvolvedObject.Kind == "Node" && reason == "NotReady"):
		return "node_not_ready"
	case reason == "FailedCreate" || reason == "FailedRollout" ||
		reason == "ReplicaSetCreateError":
		return "deploy_fail"
	case reason == "Failed" && strings.Contains(event.Message, "ImagePullBackOff"):
		return "image_pull_fail"
	case reason == "ErrImagePull" || reason == "ImagePullBackOff":
		return "image_pull_fail"
	default:
		return ""
	}
}

func eventHash(event *corev1.Event) string {
	data := fmt.Sprintf("%s/%s/%s/%s/%d",
		event.Namespace, event.InvolvedObject.Name,
		event.Reason, event.InvolvedObject.Kind, event.Count)
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:8])
}

func (nm *NotificationManager) wasRecentlySent(key string) bool {
	nm.sentEventsMu.Lock()
	defer nm.sentEventsMu.Unlock()
	t, ok := nm.sentEvents[key]
	if !ok {
		return false
	}
	return time.Since(t) < 5*time.Minute
}

func (nm *NotificationManager) markSent(key string) {
	nm.sentEventsMu.Lock()
	defer nm.sentEventsMu.Unlock()
	nm.sentEvents[key] = time.Now()
}

func (nm *NotificationManager) cleanupDedup() {
	nm.sentEventsMu.Lock()
	defer nm.sentEventsMu.Unlock()
	for k, t := range nm.sentEvents {
		if time.Since(t) > 10*time.Minute {
			delete(nm.sentEvents, k)
		}
	}
}

func (nm *NotificationManager) recordHistory(event *corev1.Event, notifType, provider string, err error) {
	entry := NotificationHistoryEntry{
		Timestamp: time.Now(),
		EventType: notifType,
		Resource:  event.InvolvedObject.Kind + "/" + event.InvolvedObject.Name,
		Namespace: event.Namespace,
		Message:   truncate(event.Message, 200),
		Provider:  provider,
		Success:   err == nil,
	}
	if err != nil {
		entry.Error = err.Error()
	}

	nm.historyMu.Lock()
	defer nm.historyMu.Unlock()
	nm.history = append([]NotificationHistoryEntry{entry}, nm.history...)
	if len(nm.history) > 100 {
		nm.history = nm.history[:100]
	}
}

// dispatch routes the notification to the correct provider.
func (nm *NotificationManager) dispatch(ncfg *config.NotificationsConfig, event *corev1.Event, notifType string) error {
	switch ncfg.Provider {
	case "slack":
		return nm.sendSlack(ncfg, event, notifType)
	case "discord":
		return nm.sendDiscord(ncfg, event, notifType)
	case "teams":
		return nm.sendTeams(ncfg, event, notifType)
	case "email":
		return nm.sendEmail(event, notifType)
	default:
		return nm.sendCustomWebhook(ncfg, event, notifType)
	}
}

func (nm *NotificationManager) sendSlack(ncfg *config.NotificationsConfig, event *corev1.Event, notifType string) error {
	payload := map[string]interface{}{
		"text": fmt.Sprintf(":warning: *[k13d] %s*\n*%s/%s* in `%s`\n> %s",
			notifType, event.InvolvedObject.Kind, event.InvolvedObject.Name,
			event.Namespace, truncate(event.Message, 500)),
	}
	if ncfg.Channel != "" {
		payload["channel"] = ncfg.Channel
	}
	return nm.postJSON(ncfg.WebhookURL, payload)
}

func (nm *NotificationManager) sendDiscord(ncfg *config.NotificationsConfig, event *corev1.Event, notifType string) error {
	content := fmt.Sprintf("⚠️ **[k13d] %s**\n**%s/%s** in `%s`\n> %s",
		notifType, event.InvolvedObject.Kind, event.InvolvedObject.Name,
		event.Namespace, truncate(event.Message, 1500))
	return nm.postJSON(ncfg.WebhookURL, map[string]interface{}{
		"content": content,
	})
}

func (nm *NotificationManager) sendTeams(ncfg *config.NotificationsConfig, event *corev1.Event, notifType string) error {
	return nm.postJSON(ncfg.WebhookURL, map[string]interface{}{
		"@type":      "MessageCard",
		"@context":   "http://schema.org/extensions",
		"themeColor": "FF0000",
		"summary":    fmt.Sprintf("k13d: %s", notifType),
		"title":      fmt.Sprintf("k13d Alert: %s", notifType),
		"sections": []map[string]interface{}{
			{
				"facts": []map[string]string{
					{"name": "Resource", "value": event.InvolvedObject.Kind + "/" + event.InvolvedObject.Name},
					{"name": "Namespace", "value": event.Namespace},
					{"name": "Reason", "value": event.Reason},
					{"name": "Message", "value": truncate(event.Message, 500)},
				},
			},
		},
	})
}

func (nm *NotificationManager) sendEmail(event *corev1.Event, notifType string) error {
	smtpCfg := nm.cfg.Notifications.SMTP
	if smtpCfg.Host == "" || len(smtpCfg.To) == 0 {
		return fmt.Errorf("SMTP not configured")
	}

	subject := fmt.Sprintf("[k13d] %s: %s/%s in %s",
		notifType, event.InvolvedObject.Kind, event.InvolvedObject.Name, event.Namespace)
	body := fmt.Sprintf("Event Type: %s\nResource: %s/%s\nNamespace: %s\nReason: %s\nMessage: %s\nTime: %s",
		notifType, event.InvolvedObject.Kind, event.InvolvedObject.Name,
		event.Namespace, event.Reason, event.Message, event.LastTimestamp.Format(time.RFC3339))

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		smtpCfg.From, strings.Join(smtpCfg.To, ","), subject, body)

	addr := fmt.Sprintf("%s:%d", smtpCfg.Host, smtpCfg.Port)

	var auth smtp.Auth
	if smtpCfg.Username != "" {
		auth = smtp.PlainAuth("", smtpCfg.Username, smtpCfg.Password, smtpCfg.Host)
	}

	if smtpCfg.UseTLS {
		return nm.sendEmailTLS(addr, auth, smtpCfg, msg)
	}
	return smtp.SendMail(addr, auth, smtpCfg.From, smtpCfg.To, []byte(msg))
}

func (nm *NotificationManager) sendEmailTLS(addr string, auth smtp.Auth, smtpCfg config.SMTPConfig, msg string) error {
	tlsConfig := &tls.Config{ServerName: smtpCfg.Host}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial failed: %w", err)
	}

	client, err := smtp.NewClient(conn, smtpCfg.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("SMTP client failed: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}
	if err := client.Mail(smtpCfg.From); err != nil {
		return err
	}
	for _, to := range smtpCfg.To {
		if err := client.Rcpt(to); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return err
	}
	return w.Close()
}

func (nm *NotificationManager) sendCustomWebhook(ncfg *config.NotificationsConfig, event *corev1.Event, notifType string) error {
	return nm.postJSON(ncfg.WebhookURL, map[string]interface{}{
		"source":    "k13d",
		"type":      notifType,
		"kind":      event.InvolvedObject.Kind,
		"name":      event.InvolvedObject.Name,
		"namespace": event.Namespace,
		"reason":    event.Reason,
		"message":   event.Message,
		"count":     event.Count,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// SendTestEmail sends a test email to verify SMTP configuration.
func (nm *NotificationManager) SendTestEmail() error {
	smtpCfg := nm.cfg.Notifications.SMTP
	if smtpCfg.Host == "" || len(smtpCfg.To) == 0 {
		return fmt.Errorf("SMTP not configured")
	}
	subject := "[k13d] Test Notification"
	body := fmt.Sprintf("This is a test notification from k13d at %s", time.Now().Format(time.RFC3339))
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		smtpCfg.From, strings.Join(smtpCfg.To, ","), subject, body)

	addr := fmt.Sprintf("%s:%d", smtpCfg.Host, smtpCfg.Port)
	var auth smtp.Auth
	if smtpCfg.Username != "" {
		auth = smtp.PlainAuth("", smtpCfg.Username, smtpCfg.Password, smtpCfg.Host)
	}
	if smtpCfg.UseTLS {
		return nm.sendEmailTLS(addr, auth, smtpCfg, msg)
	}
	return smtp.SendMail(addr, auth, smtpCfg.From, smtpCfg.To, []byte(msg))
}

func (nm *NotificationManager) postJSON(url string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := nm.httpClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		// Single retry after 2 seconds
		time.Sleep(2 * time.Second)
		resp, err = nm.httpClient.Post(url, "application/json", bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("webhook failed after retry: %w", err)
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
