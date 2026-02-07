package web

import (
	"testing"
	"time"
)

func TestAccessRequest_CreateAndApprove(t *testing.T) {
	m := NewAccessRequestManager(30 * time.Minute)

	id, err := m.CreateRequest("alice", ActionScale, "deployments", "production", "Need to scale nginx")
	if err != nil {
		t.Fatalf("CreateRequest failed: %v", err)
	}

	if id == "" {
		t.Fatal("Expected non-empty request ID")
	}

	// Verify it's pending
	req := m.GetRequest(id)
	if req == nil {
		t.Fatal("Expected to find request")
	}
	if req.State != AccessRequestPending {
		t.Errorf("Expected pending state, got %s", req.State)
	}

	// Approve by a different user
	err = m.ApproveRequest(id, "bob-admin", "Approved for scaling")
	if err != nil {
		t.Fatalf("ApproveRequest failed: %v", err)
	}

	// Verify it's approved
	req = m.GetRequest(id)
	if req.State != AccessRequestApproved {
		t.Errorf("Expected approved state, got %s", req.State)
	}
	if req.ReviewedBy != "bob-admin" {
		t.Errorf("Expected reviewer bob-admin, got %s", req.ReviewedBy)
	}
}

func TestAccessRequest_SelfApprovalDenied(t *testing.T) {
	m := NewAccessRequestManager(30 * time.Minute)

	id, _ := m.CreateRequest("alice", ActionScale, "deployments", "production", "Need to scale")

	// Alice cannot approve her own request
	err := m.ApproveRequest(id, "alice", "Self-approval")
	if err == nil {
		t.Error("Expected error for self-approval")
	}

	// Verify still pending
	req := m.GetRequest(id)
	if req.State != AccessRequestPending {
		t.Errorf("Expected pending state after self-approval attempt, got %s", req.State)
	}
}

func TestAccessRequest_Expiration(t *testing.T) {
	// Create with very short TTL
	m := NewAccessRequestManager(1 * time.Millisecond)

	id, _ := m.CreateRequest("alice", ActionDelete, "pods", "default", "Test")

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Should not be approved after expiry
	if m.IsApproved("alice", "pods", ActionDelete, "default") {
		t.Error("Expected expired request to not be approved")
	}

	// Try to approve expired request
	err := m.ApproveRequest(id, "bob", "Late approval")
	if err == nil {
		t.Error("Expected error when approving expired request")
	}
}

func TestAccessRequest_DenyRequest(t *testing.T) {
	m := NewAccessRequestManager(30 * time.Minute)

	id, _ := m.CreateRequest("alice", ActionDelete, "pods", "production", "Need to delete old pods")

	err := m.DenyRequest(id, "bob-admin", "Not authorized for production")
	if err != nil {
		t.Fatalf("DenyRequest failed: %v", err)
	}

	req := m.GetRequest(id)
	if req.State != AccessRequestDenied {
		t.Errorf("Expected denied state, got %s", req.State)
	}
	if req.ReviewedBy != "bob-admin" {
		t.Errorf("Expected reviewer bob-admin, got %s", req.ReviewedBy)
	}
}

func TestAccessRequest_IsApproved(t *testing.T) {
	m := NewAccessRequestManager(30 * time.Minute)

	// Not approved before request
	if m.IsApproved("alice", "deployments", ActionScale, "production") {
		t.Error("Expected not approved without request")
	}

	id, _ := m.CreateRequest("alice", ActionScale, "deployments", "production", "Scale nginx")
	m.ApproveRequest(id, "bob", "OK")

	// Should be approved now
	if !m.IsApproved("alice", "deployments", ActionScale, "production") {
		t.Error("Expected approved after approval")
	}

	// Different action should not be approved
	if m.IsApproved("alice", "deployments", ActionDelete, "production") {
		t.Error("Expected not approved for different action")
	}

	// Different user should not be approved
	if m.IsApproved("charlie", "deployments", ActionScale, "production") {
		t.Error("Expected not approved for different user")
	}
}

func TestAccessRequest_PendingList(t *testing.T) {
	m := NewAccessRequestManager(30 * time.Minute)

	m.CreateRequest("alice", ActionScale, "deployments", "default", "Scale 1")
	m.CreateRequest("bob", ActionDelete, "pods", "default", "Delete old")
	id3, _ := m.CreateRequest("charlie", ActionRestart, "deployments", "default", "Restart")

	// All should be pending
	pending := m.GetPendingRequests()
	if len(pending) != 3 {
		t.Errorf("Expected 3 pending requests, got %d", len(pending))
	}

	// Approve one
	m.ApproveRequest(id3, "admin", "OK")

	pending = m.GetPendingRequests()
	if len(pending) != 2 {
		t.Errorf("Expected 2 pending requests after approval, got %d", len(pending))
	}
}

func TestAccessRequest_DoubleApproval(t *testing.T) {
	m := NewAccessRequestManager(30 * time.Minute)

	id, _ := m.CreateRequest("alice", ActionScale, "deployments", "default", "Scale")
	m.ApproveRequest(id, "bob", "OK")

	// Second approval should fail
	err := m.ApproveRequest(id, "charlie", "Also OK")
	if err == nil {
		t.Error("Expected error for double approval")
	}
}

func TestAccessRequest_NotFound(t *testing.T) {
	m := NewAccessRequestManager(30 * time.Minute)

	err := m.ApproveRequest("nonexistent", "bob", "OK")
	if err == nil {
		t.Error("Expected error for nonexistent request")
	}

	err = m.DenyRequest("nonexistent", "bob", "No")
	if err == nil {
		t.Error("Expected error for nonexistent request")
	}
}
