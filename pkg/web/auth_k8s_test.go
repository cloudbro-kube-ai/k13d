package web

import (
	"context"
	"errors"
	"testing"

	authzv1 "k8s.io/api/authorization/v1"
)

type fakeTokenAccessReviewer struct {
	allowed map[string]bool
	err     error
}

func (f *fakeTokenAccessReviewer) CanI(_ context.Context, attrs authzv1.ResourceAttributes) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.allowed[accessReviewKey(attrs)], nil
}

func accessReviewKey(attrs authzv1.ResourceAttributes) string {
	return attrs.Verb + "|" + attrs.Group + "|" + attrs.Resource + "|" + attrs.Subresource
}

func TestDetermineRoleFromGroups(t *testing.T) {
	tests := []struct {
		name     string
		username string
		groups   []string
		want     string
	}{
		{
			name:     "service account defaults to viewer",
			username: "system:serviceaccount:default:k13d",
			groups:   []string{"system:serviceaccounts", "system:serviceaccounts:default", "system:authenticated"},
			want:     "viewer",
		},
		{
			name:     "edit group becomes user",
			username: "alice",
			groups:   []string{"dev-editors", "system:authenticated"},
			want:     "user",
		},
		{
			name:     "admin group becomes admin",
			username: "bob",
			groups:   []string{"platform-admins"},
			want:     "admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := determineRoleFromGroups(tt.username, tt.groups); got != tt.want {
				t.Fatalf("determineRoleFromGroups() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetermineRoleFromTokenAccess(t *testing.T) {
	tests := []struct {
		name    string
		allowed []authzv1.ResourceAttributes
		want    string
	}{
		{
			name: "viewer permissions",
			allowed: []authzv1.ResourceAttributes{
				{Resource: "pods", Verb: "list"},
			},
			want: "viewer",
		},
		{
			name: "user permissions",
			allowed: []authzv1.ResourceAttributes{
				{Group: "apps", Resource: "deployments", Verb: "patch"},
			},
			want: "user",
		},
		{
			name: "admin permissions",
			allowed: []authzv1.ResourceAttributes{
				{Group: "rbac.authorization.k8s.io", Resource: "clusterrolebindings", Verb: "create"},
			},
			want: "admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := make(map[string]bool, len(tt.allowed))
			for _, attrs := range tt.allowed {
				allowed[accessReviewKey(attrs)] = true
			}

			got, err := determineRoleFromTokenAccess(context.Background(), &fakeTokenAccessReviewer{
				allowed: allowed,
			})
			if err != nil {
				t.Fatalf("determineRoleFromTokenAccess() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("determineRoleFromTokenAccess() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetermineRoleFromTokenAccess_Error(t *testing.T) {
	_, err := determineRoleFromTokenAccess(context.Background(), &fakeTokenAccessReviewer{
		err: errors.New("review failed"),
	})
	if err == nil {
		t.Fatal("expected determineRoleFromTokenAccess to return an error")
	}
}
