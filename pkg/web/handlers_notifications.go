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

// NotificationConfig represents webhook notification settings (API contract)
type NotificationConfig struct {
	Enabled    bool            `json:"enabled"`
	WebhookURL string          `json:"webhook_url"`
	Channel    string          `json:"channel,omitempty"`
	Events     []string        `json:"events"`
	Provider   string          `json:"provider"`
	SMTP       *SMTPConfigJSON `json:"smtp,omitempty"`
}

// SMTPConfigJSON is the JSON API contract for SMTP settings
type SMTPConfigJSON struct {
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Username string   `json:"username"`
	Password string   `json:"password,omitempty"` // accepted on POST, never returned on GET
	From     string   `json:"from"`
	To       []string `json:"to"`
	UseTLS   bool     `json:"use_tls"`
}

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
		safeConfig := *notifConfig
		if safeConfig.WebhookURL != "" {
			safeConfig.WebhookURL = maskWebhookURL(safeConfig.WebhookURL)
		}
		// Include SMTP info (without password) if email provider
		if safeConfig.Provider == "email" {
			smtpCfg := s.cfg.Notifications.SMTP
			safeConfig.SMTP = &SMTPConfigJSON{
				Host:     smtpCfg.Host,
				Port:     smtpCfg.Port,
				Username: smtpCfg.Username,
				From:     smtpCfg.From,
				To:       smtpCfg.To,
				UseTLS:   smtpCfg.UseTLS,
			}
		}
		json.NewEncoder(w).Encode(safeConfig)

	case http.MethodPost:
		var cfg NotificationConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Email provider doesn't need webhook URL
		if cfg.Enabled && cfg.Provider != "email" && cfg.WebhookURL == "" {
			http.Error(w, "Webhook URL is required when notifications are enabled", http.StatusBadRequest)
			return
		}

		if cfg.WebhookURL != "" && cfg.Provider != "email" {
			if err := validateWebhookURL(cfg.WebhookURL); err != nil {
				http.Error(w, "Invalid webhook URL: "+err.Error(), http.StatusBadRequest)
				return
			}
		}

		// Update in-memory config
		notifConfigMu.Lock()
		notifConfig = &cfg
		notifConfigMu.Unlock()

		// Persist to config.yaml
		s.cfg.Notifications.Enabled = cfg.Enabled
		s.cfg.Notifications.Provider = cfg.Provider
		s.cfg.Notifications.WebhookURL = cfg.WebhookURL
		s.cfg.Notifications.Channel = cfg.Channel
		s.cfg.Notifications.Events = cfg.Events

		// Handle SMTP config
		if cfg.Provider == "email" && cfg.SMTP != nil {
			s.cfg.Notifications.SMTP.Host = cfg.SMTP.Host
			if cfg.SMTP.Port > 0 {
				s.cfg.Notifications.SMTP.Port = cfg.SMTP.Port
			}
			s.cfg.Notifications.SMTP.Username = cfg.SMTP.Username
			if cfg.SMTP.Password != "" {
				s.cfg.Notifications.SMTP.Password = cfg.SMTP.Password
			}
			s.cfg.Notifications.SMTP.From = cfg.SMTP.From
			s.cfg.Notifications.SMTP.To = cfg.SMTP.To
			s.cfg.Notifications.SMTP.UseTLS = cfg.SMTP.UseTLS
		}

		if err := s.cfg.Save(); err != nil {
			fmt.Printf("Warning: failed to persist notification config: %v\n", err)
		}

		// Update notification manager
		if s.notifManager != nil {
			if cfg.Enabled {
				if !s.notifManager.IsRunning() {
					s.notifManager.Start()
				}
			} else {
				if s.notifManager.IsRunning() {
					s.notifManager.Stop()
				}
			}
		}

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

	if !cfg.Enabled {
		http.Error(w, "Notifications not configured", http.StatusBadRequest)
		return
	}

	// Email provider uses SMTP
	if cfg.Provider == "email" {
		if s.notifManager == nil {
			http.Error(w, "Notification manager not initialized", http.StatusInternalServerError)
			return
		}
		if err := s.notifManager.SendTestEmail(); err != nil {
			http.Error(w, "Failed to send test email: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Test email sent successfully",
		})
		return
	}

	if cfg.WebhookURL == "" {
		http.Error(w, "Webhook URL not configured", http.StatusBadRequest)
		return
	}

	payload, err := buildTestPayload(cfg.Provider, cfg.Channel)
	if err != nil {
		http.Error(w, "Failed to build test payload: "+err.Error(), http.StatusInternalServerError)
		return
	}

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

func (s *Server) handleNotificationHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if s.notifManager == nil {
		json.NewEncoder(w).Encode([]NotificationHistoryEntry{})
		return
	}
	json.NewEncoder(w).Encode(s.notifManager.GetHistory())
}

func (s *Server) handleNotificationStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	running := false
	dedupCount := 0
	if s.notifManager != nil {
		running = s.notifManager.IsRunning()
		dedupCount = s.notifManager.DedupCount()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"running":     running,
		"dedup_count": dedupCount,
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

func validateWebhookURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("malformed URL")
	}

	if u.Scheme != "https" {
		return fmt.Errorf("only HTTPS URLs are allowed")
	}

	host := u.Hostname()

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
