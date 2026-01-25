package web

import (
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFormatAge(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "seconds ago",
			time:     now.Add(-30 * time.Second),
			expected: "30s",
		},
		{
			name:     "minutes ago",
			time:     now.Add(-5 * time.Minute),
			expected: "5m",
		},
		{
			name:     "minutes and seconds",
			time:     now.Add(-5*time.Minute - 30*time.Second),
			expected: "5m30s",
		},
		{
			name:     "hours ago",
			time:     now.Add(-2 * time.Hour),
			expected: "2h",
		},
		{
			name:     "hours and minutes",
			time:     now.Add(-2*time.Hour - 30*time.Minute),
			expected: "2h30m",
		},
		{
			name:     "days ago",
			time:     now.Add(-3 * 24 * time.Hour),
			expected: "3d",
		},
		{
			name:     "days and hours",
			time:     now.Add(-3*24*time.Hour - 12*time.Hour),
			expected: "3d12h",
		},
		{
			name:     "months ago",
			time:     now.Add(-45 * 24 * time.Hour),
			expected: "1M15d",
		},
		{
			name:     "years ago",
			time:     now.Add(-400 * 24 * time.Hour),
			expected: "1y1M",
		},
		{
			name:     "just now",
			time:     now,
			expected: "0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAge(tt.time)
			if result != tt.expected {
				t.Errorf("formatAge() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestGetRevision(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		expected    int64
	}{
		{
			name:        "nil annotations",
			annotations: nil,
			expected:    0,
		},
		{
			name:        "empty annotations",
			annotations: map[string]string{},
			expected:    0,
		},
		{
			name: "no revision annotation",
			annotations: map[string]string{
				"other-annotation": "value",
			},
			expected: 0,
		},
		{
			name: "valid revision",
			annotations: map[string]string{
				"deployment.kubernetes.io/revision": "5",
			},
			expected: 5,
		},
		{
			name: "revision 1",
			annotations: map[string]string{
				"deployment.kubernetes.io/revision": "1",
			},
			expected: 1,
		},
		{
			name: "high revision number",
			annotations: map[string]string{
				"deployment.kubernetes.io/revision": "100",
			},
			expected: 100,
		},
		{
			name: "invalid revision format",
			annotations: map[string]string{
				"deployment.kubernetes.io/revision": "abc",
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: tt.annotations,
				},
			}
			result := getRevision(rs)
			if result != tt.expected {
				t.Errorf("getRevision() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestDeploymentScaleRequest(t *testing.T) {
	req := DeploymentScaleRequest{
		Namespace: "default",
		Name:      "nginx",
		Replicas:  3,
	}

	if req.Namespace != "default" {
		t.Errorf("Namespace = %s, want default", req.Namespace)
	}
	if req.Name != "nginx" {
		t.Errorf("Name = %s, want nginx", req.Name)
	}
	if req.Replicas != 3 {
		t.Errorf("Replicas = %d, want 3", req.Replicas)
	}
}

func TestDeploymentRollbackRequest(t *testing.T) {
	req := DeploymentRollbackRequest{
		Namespace: "production",
		Name:      "api",
		Revision:  5,
	}

	if req.Namespace != "production" {
		t.Errorf("Namespace = %s, want production", req.Namespace)
	}
	if req.Name != "api" {
		t.Errorf("Name = %s, want api", req.Name)
	}
	if req.Revision != 5 {
		t.Errorf("Revision = %d, want 5", req.Revision)
	}

	// Test zero revision (rollback to previous)
	reqPrev := DeploymentRollbackRequest{
		Namespace: "default",
		Name:      "web",
		Revision:  0,
	}
	if reqPrev.Revision != 0 {
		t.Errorf("Revision = %d, want 0 for previous", reqPrev.Revision)
	}
}
