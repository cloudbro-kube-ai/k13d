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

// AccessRequestState represents the state of an access request
type AccessRequestState string

const (
	AccessRequestPending  AccessRequestState = "pending"
	AccessRequestApproved AccessRequestState = "approved"
	AccessRequestDenied   AccessRequestState = "denied"
	AccessRequestExpired  AccessRequestState = "expired"
)

// AccessRequest represents a request for elevated access (Teleport-inspired)
type AccessRequest struct {
	ID          string             `json:"id"`
	RequestedBy string             `json:"requested_by"`
	Action      Action             `json:"action"`
	Resource    string             `json:"resource"`
	Namespace   string             `json:"namespace"`
	Reason      string             `json:"reason"`
	State       AccessRequestState `json:"state"`
	ReviewedBy  string             `json:"reviewed_by,omitempty"`
	ReviewNote  string             `json:"review_note,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	ReviewedAt  time.Time          `json:"reviewed_at,omitempty"`
	ExpiresAt   time.Time          `json:"expires_at"`
}

// AccessRequestManager manages access requests (Teleport-inspired workflow)
type AccessRequestManager struct {
	mu       sync.RWMutex
	requests map[string]*AccessRequest
	ttl      time.Duration // Default TTL for approved requests
}

// NewAccessRequestManager creates a new access request manager
func NewAccessRequestManager(ttl time.Duration) *AccessRequestManager {
	if ttl == 0 {
		ttl = 30 * time.Minute
	}
	return &AccessRequestManager{
		requests: make(map[string]*AccessRequest),
		ttl:      ttl,
	}
}

// CreateRequest creates a new access request
func (m *AccessRequestManager) CreateRequest(requestedBy string, action Action, resource, namespace, reason string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := generateAccessRequestID()

	req := &AccessRequest{
		ID:          id,
		RequestedBy: requestedBy,
		Action:      action,
		Resource:    resource,
		Namespace:   namespace,
		Reason:      reason,
		State:       AccessRequestPending,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(m.ttl),
	}

	m.requests[id] = req

	// Persist to database
	if db.DB != nil {
		db.DB.Exec(
			`INSERT INTO access_requests (id, requested_by, action, resource, namespace, reason, state, created_at, expires_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, requestedBy, string(action), resource, namespace, reason,
			string(AccessRequestPending), req.CreatedAt, req.ExpiresAt)
	}

	// Record audit event
	db.RecordAudit(db.AuditEntry{
		User:            requestedBy,
		Action:          "access_request_created",
		Resource:        resource,
		Details:         fmt.Sprintf("Requested %s on %s/%s: %s", action, namespace, resource, reason),
		ActionType:      db.ActionTypeAccessRequest,
		Source:          "web",
		Success:         true,
		RequestedAction: string(action),
		TargetResource:  resource,
		TargetNamespace: namespace,
		AccessRequestID: id,
	})

	return id, nil
}

// ApproveRequest approves an access request (reviewer must be different from requester)
func (m *AccessRequestManager) ApproveRequest(id, reviewer, note string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	req, exists := m.requests[id]
	if !exists {
		return fmt.Errorf("access request not found: %s", id)
	}

	if req.State != AccessRequestPending {
		return fmt.Errorf("access request is not pending: %s (current state: %s)", id, req.State)
	}

	// Prevent self-approval (Teleport pattern)
	if req.RequestedBy == reviewer {
		return fmt.Errorf("cannot approve your own access request")
	}

	// Check if expired
	if time.Now().After(req.ExpiresAt) {
		req.State = AccessRequestExpired
		return fmt.Errorf("access request has expired")
	}

	req.State = AccessRequestApproved
	req.ReviewedBy = reviewer
	req.ReviewNote = note
	req.ReviewedAt = time.Now()

	// Update database
	if db.DB != nil {
		db.DB.Exec(
			`UPDATE access_requests SET state = ?, reviewed_by = ?, review_note = ?, reviewed_at = ? WHERE id = ?`,
			string(AccessRequestApproved), reviewer, note, req.ReviewedAt, id)
	}

	// Record audit event
	db.RecordAudit(db.AuditEntry{
		User:            reviewer,
		Action:          "access_request_approved",
		Resource:        req.Resource,
		Details:         fmt.Sprintf("Approved %s on %s/%s for %s", req.Action, req.Namespace, req.Resource, req.RequestedBy),
		ActionType:      db.ActionTypeAccessApproved,
		Source:          "web",
		Success:         true,
		RequestedAction: string(req.Action),
		TargetResource:  req.Resource,
		TargetNamespace: req.Namespace,
		AccessRequestID: id,
		ReviewerUser:    reviewer,
		AuthzDecision:   "approved",
	})

	return nil
}

// DenyRequest denies an access request
func (m *AccessRequestManager) DenyRequest(id, reviewer, note string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	req, exists := m.requests[id]
	if !exists {
		return fmt.Errorf("access request not found: %s", id)
	}

	if req.State != AccessRequestPending {
		return fmt.Errorf("access request is not pending: %s", id)
	}

	req.State = AccessRequestDenied
	req.ReviewedBy = reviewer
	req.ReviewNote = note
	req.ReviewedAt = time.Now()

	// Update database
	if db.DB != nil {
		db.DB.Exec(
			`UPDATE access_requests SET state = ?, reviewed_by = ?, review_note = ?, reviewed_at = ? WHERE id = ?`,
			string(AccessRequestDenied), reviewer, note, req.ReviewedAt, id)
	}

	// Record audit event
	db.RecordAudit(db.AuditEntry{
		User:            reviewer,
		Action:          "access_request_denied",
		Resource:        req.Resource,
		Details:         fmt.Sprintf("Denied %s on %s/%s for %s: %s", req.Action, req.Namespace, req.Resource, req.RequestedBy, note),
		ActionType:      db.ActionTypeAccessDenied,
		Source:          "web",
		Success:         true,
		RequestedAction: string(req.Action),
		TargetResource:  req.Resource,
		TargetNamespace: req.Namespace,
		AccessRequestID: id,
		ReviewerUser:    reviewer,
		AuthzDecision:   "denied",
	})

	return nil
}

// IsApproved checks if a user has an approved access request for a specific action
func (m *AccessRequestManager) IsApproved(username, resource string, action Action, namespace string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, req := range m.requests {
		if req.RequestedBy != username {
			continue
		}
		if req.State != AccessRequestApproved {
			continue
		}
		if time.Now().After(req.ExpiresAt) {
			continue
		}
		if req.Action != action {
			continue
		}
		if req.Resource != resource && req.Resource != "*" {
			continue
		}
		if namespace != "" && req.Namespace != namespace && req.Namespace != "*" {
			continue
		}
		return true
	}
	return false
}

// GetPendingRequests returns all pending access requests
func (m *AccessRequestManager) GetPendingRequests() []*AccessRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var pending []*AccessRequest
	now := time.Now()
	for _, req := range m.requests {
		if req.State == AccessRequestPending && now.Before(req.ExpiresAt) {
			pending = append(pending, req)
		}
	}
	return pending
}

// GetRequest returns a specific access request
func (m *AccessRequestManager) GetRequest(id string) *AccessRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.requests[id]
}

// CleanupExpired removes expired requests
func (m *AccessRequestManager) CleanupExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for id, req := range m.requests {
		if req.State == AccessRequestPending && now.After(req.ExpiresAt) {
			req.State = AccessRequestExpired
			// Update database
			if db.DB != nil {
				db.DB.Exec("UPDATE access_requests SET state = ? WHERE id = ?",
					string(AccessRequestExpired), id)
			}
		}
	}
}

// generateAccessRequestID creates a unique access request ID
func generateAccessRequestID() string {
	return "ar-" + generateSessionID()[:20]
}

// ---- HTTP Handlers ----

// HandleCreateAccessRequest handles POST /api/access/request
func (m *AccessRequestManager) HandleCreateAccessRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Action    string `json:"action"`
		Resource  string `json:"resource"`
		Namespace string `json:"namespace"`
		Reason    string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Action == "" || req.Resource == "" {
		http.Error(w, "Action and resource are required", http.StatusBadRequest)
		return
	}

	username := r.Header.Get("X-Username")
	id, err := m.CreateRequest(username, Action(req.Action), req.Resource, req.Namespace, req.Reason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"id":     id,
		"status": "pending",
	})
}

// HandleListAccessRequests handles GET /api/access/requests
func (m *AccessRequestManager) HandleListAccessRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Cleanup expired before listing
	m.CleanupExpired()

	pending := m.GetPendingRequests()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"requests": pending,
		"total":    len(pending),
	})
}

// HandleApproveAccessRequest handles POST /api/access/approve/{id}
func (m *AccessRequestManager) HandleApproveAccessRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	id := strings.TrimPrefix(r.URL.Path, "/api/access/approve/")
	if id == "" || id == r.URL.Path {
		http.Error(w, "Request ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Note string `json:"note"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	reviewer := r.Header.Get("X-Username")
	if err := m.ApproveRequest(id, reviewer, req.Note); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "approved",
		"id":     id,
	})
}

// HandleDenyAccessRequest handles POST /api/access/deny/{id}
func (m *AccessRequestManager) HandleDenyAccessRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL
	id := strings.TrimPrefix(r.URL.Path, "/api/access/deny/")
	if id == "" || id == r.URL.Path {
		http.Error(w, "Request ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Note string `json:"note"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	reviewer := r.Header.Get("X-Username")
	if err := m.DenyRequest(id, reviewer, req.Note); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "denied",
		"id":     id,
	})
}
