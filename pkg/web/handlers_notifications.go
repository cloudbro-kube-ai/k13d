package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// NotificationConfig represents webhook notification settings
type NotificationConfig struct {
	Enabled    bool     `json:"enabled"`
	WebhookURL string   `json:"webhook_url"`
	Channel    string   `json:"channel,omitempty"`
	Events     []string `json:"events"`   // e.g., ["pod_crash", "deploy_fail", "security_alert"]
	Provider   string   `json:"provider"` // "slack", "discord", "teams"
}

// notificationStore stores notification config in memory
// (could be extended to use DB)
var (
	notifConfig   *NotificationConfig
	notifConfigMu sync.RWMutex
)

func init() {
	notifConfig = &NotificationConfig{
		Enabled: false,
		Events:  []string{},
	}
}

func (s *Server) handleNotificationConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		notifConfigMu.RLock()
		defer notifConfigMu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		// Don't expose the full webhook URL in GET responses
		safeConfig := *notifConfig
		if safeConfig.WebhookURL != "" {
			safeConfig.WebhookURL = maskWebhookURL(safeConfig.WebhookURL)
		}
		json.NewEncoder(w).Encode(safeConfig)

	case http.MethodPost:
		var cfg NotificationConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if cfg.Enabled && cfg.WebhookURL == "" {
			http.Error(w, "Webhook URL is required when notifications are enabled", http.StatusBadRequest)
			return
		}

		if cfg.WebhookURL != "" {
			if err := validateWebhookURL(cfg.WebhookURL); err != nil {
				http.Error(w, "Invalid webhook URL: "+err.Error(), http.StatusBadRequest)
				return
			}
		}

		notifConfigMu.Lock()
		notifConfig = &cfg
		notifConfigMu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Notification config saved",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleNotificationTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	notifConfigMu.RLock()
	cfg := *notifConfig
	notifConfigMu.RUnlock()

	if !cfg.Enabled || cfg.WebhookURL == "" {
		http.Error(w, "Notifications not configured", http.StatusBadRequest)
		return
	}

	// Build test message based on provider
	payload, err := buildTestPayload(cfg.Provider, cfg.Channel)
	if err != nil {
		http.Error(w, "Failed to build test payload: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Send test notification
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(cfg.WebhookURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		http.Error(w, "Failed to send test notification: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		http.Error(w, fmt.Sprintf("Webhook returned status %d", resp.StatusCode), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Test notification sent successfully",
	})
}

func buildTestPayload(provider, channel string) ([]byte, error) {
	timestamp := time.Now().Format(time.RFC3339)

	switch provider {
	case "slack":
		msg := map[string]interface{}{
			"text": fmt.Sprintf("[k13d Test] This is a test notification from k13d at %s", timestamp),
		}
		if channel != "" {
			msg["channel"] = channel
		}
		return json.Marshal(msg)

	case "discord":
		return json.Marshal(map[string]interface{}{
			"content": fmt.Sprintf("[k13d Test] This is a test notification from k13d at %s", timestamp),
		})

	case "teams":
		return json.Marshal(map[string]interface{}{
			"@type":      "MessageCard",
			"@context":   "http://schema.org/extensions",
			"themeColor": "0076D7",
			"summary":    "k13d Test Notification",
			"title":      "k13d Test Notification",
			"text":       fmt.Sprintf("This is a test notification from k13d at %s", timestamp),
		})

	default:
		// Generic JSON payload
		return json.Marshal(map[string]interface{}{
			"text":      fmt.Sprintf("[k13d Test] Test notification at %s", timestamp),
			"timestamp": timestamp,
		})
	}
}

func maskWebhookURL(u string) string {
	if len(u) <= 20 {
		return "****"
	}
	return u[:15] + "..." + u[len(u)-5:]
}

// validateWebhookURL ensures the webhook URL is safe (HTTPS, no private IPs)
func validateWebhookURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("malformed URL")
	}

	if u.Scheme != "https" {
		return fmt.Errorf("only HTTPS URLs are allowed")
	}

	host := u.Hostname()

	// Resolve the host to check for private/loopback IPs
	ips, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("cannot resolve host")
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("private or loopback addresses are not allowed")
		}
	}

	return nil
}
