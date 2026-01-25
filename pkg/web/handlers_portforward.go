package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/db"
)

// ==========================================
// Port Forwarding Handlers
// ==========================================

// PortForwardSession represents an active port forward
type PortForwardSession struct {
	ID         string    `json:"id"`
	Namespace  string    `json:"namespace"`
	Pod        string    `json:"pod"`
	LocalPort  int       `json:"localPort"`
	RemotePort int       `json:"remotePort"`
	Active     bool      `json:"active"`
	StartedAt  time.Time `json:"startedAt"`
	stopChan   chan struct{}
}

var (
	portForwardSessions = make(map[string]*PortForwardSession)
	pfMutex             sync.Mutex
)

func (s *Server) handlePortForwardStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Namespace  string `json:"namespace"`
		Pod        string `json:"pod"`
		LocalPort  int    `json:"localPort"`
		RemotePort int    `json:"remotePort"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate session ID
	sessionID := fmt.Sprintf("pf-%d", time.Now().UnixNano())

	session := &PortForwardSession{
		ID:         sessionID,
		Namespace:  req.Namespace,
		Pod:        req.Pod,
		LocalPort:  req.LocalPort,
		RemotePort: req.RemotePort,
		Active:     true,
		StartedAt:  time.Now(),
		stopChan:   make(chan struct{}),
	}

	// Start port forward in goroutine
	go func() {
		err := s.k8sClient.StartPortForward(
			req.Namespace,
			req.Pod,
			req.LocalPort,
			req.RemotePort,
			session.stopChan,
		)
		if err != nil {
			fmt.Printf("Port forward error: %v\n", err)
		}
		pfMutex.Lock()
		if sess, ok := portForwardSessions[sessionID]; ok {
			sess.Active = false
		}
		pfMutex.Unlock()
	}()

	pfMutex.Lock()
	portForwardSessions[sessionID] = session
	pfMutex.Unlock()

	// Record audit
	username := r.Header.Get("X-Username")
	db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   "port_forward_start",
		Resource: "pod",
		Details:  fmt.Sprintf("%s/%s local:%d remote:%d", req.Namespace, req.Pod, req.LocalPort, req.RemotePort),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (s *Server) handlePortForwardList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pfMutex.Lock()
	sessions := make([]*PortForwardSession, 0, len(portForwardSessions))
	for _, sess := range portForwardSessions {
		sessions = append(sessions, sess)
	}
	pfMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": sessions,
	})
}

func (s *Server) handlePortForwardStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract session ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/portforward/")
	sessionID := strings.TrimSuffix(path, "/")

	pfMutex.Lock()
	session, ok := portForwardSessions[sessionID]
	if !ok {
		pfMutex.Unlock()
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Stop the port forward
	close(session.stopChan)
	delete(portForwardSessions, sessionID)
	pfMutex.Unlock()

	// Record audit
	username := r.Header.Get("X-Username")
	db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   "port_forward_stop",
		Resource: "pod",
		Details:  fmt.Sprintf("%s/%s", session.Namespace, session.Pod),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}
