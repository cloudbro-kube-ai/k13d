package web

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// RBACNode represents a node in the RBAC graph
type RBACNode struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"` // "User", "Group", "ServiceAccount", "Role", "ClusterRole"
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// RBACEdge represents an edge in the RBAC graph (binding -> role)
type RBACEdge struct {
	Source      string `json:"source"`
	Target      string `json:"target"`
	BindingName string `json:"bindingName"`
	BindingKind string `json:"bindingKind"` // "RoleBinding" or "ClusterRoleBinding"
	Namespace   string `json:"namespace,omitempty"`
}

// RBACSubjectInfo represents a subject with its associated roles (for card-based UI)
type RBACSubjectInfo struct {
	Name      string        `json:"name"`
	Kind      string        `json:"kind"`
	Namespace string        `json:"namespace,omitempty"`
	Roles     []RBACRoleRef `json:"roles"`
}

// RBACRoleRef represents a role reference for a subject
type RBACRoleRef struct {
	RoleName     string `json:"role_name"`
	ClusterScope bool   `json:"cluster_scope"`
}

// RBACVisualizationResponse is the response for the RBAC visualization endpoint
type RBACVisualizationResponse struct {
	Nodes    []RBACNode        `json:"nodes"`
	Edges    []RBACEdge        `json:"edges"`
	Subjects []RBACSubjectInfo `json:"subjects"`
}

func (s *Server) handleRBACVisualization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	namespace := r.URL.Query().Get("namespace")
	subjectKindFilter := r.URL.Query().Get("subject_kind") // "User", "Group", "ServiceAccount"
	ctx := r.Context()

	var nodes []RBACNode
	var edges []RBACEdge
	nodeSet := make(map[string]bool)

	// subjectRoles maps subject ID to its roles
	subjectRoles := make(map[string]*RBACSubjectInfo)

	addNode := func(n RBACNode) {
		if !nodeSet[n.ID] {
			nodes = append(nodes, n)
			nodeSet[n.ID] = true
		}
	}

	addSubjectRole := func(subjectID, subjectKind, subjectName, subjectNS, roleName string, clusterScope bool) {
		si, ok := subjectRoles[subjectID]
		if !ok {
			si = &RBACSubjectInfo{
				Name:      subjectName,
				Kind:      subjectKind,
				Namespace: subjectNS,
			}
			subjectRoles[subjectID] = si
		}
		si.Roles = append(si.Roles, RBACRoleRef{
			RoleName:     roleName,
			ClusterScope: clusterScope,
		})
	}

	// Fetch RoleBindings (namespaced)
	roleBindings, err := s.k8sClient.ListRoleBindings(ctx, namespace)
	if err == nil {
		for _, rb := range roleBindings {
			roleID := fmt.Sprintf("Role/%s/%s", rb.Namespace, rb.RoleRef.Name)
			clusterScope := false
			if rb.RoleRef.Kind == "ClusterRole" {
				roleID = fmt.Sprintf("ClusterRole/%s", rb.RoleRef.Name)
				clusterScope = true
			}
			addNode(RBACNode{
				ID:        roleID,
				Kind:      rb.RoleRef.Kind,
				Name:      rb.RoleRef.Name,
				Namespace: rb.Namespace,
			})

			for _, subject := range rb.Subjects {
				if subjectKindFilter != "" && subject.Kind != subjectKindFilter {
					continue
				}
				subjectID := fmt.Sprintf("%s/%s/%s", subject.Kind, subject.Namespace, subject.Name)
				addNode(RBACNode{
					ID:        subjectID,
					Kind:      subject.Kind,
					Name:      subject.Name,
					Namespace: subject.Namespace,
				})
				edges = append(edges, RBACEdge{
					Source:      subjectID,
					Target:      roleID,
					BindingName: rb.Name,
					BindingKind: "RoleBinding",
					Namespace:   rb.Namespace,
				})
				addSubjectRole(subjectID, subject.Kind, subject.Name, subject.Namespace, rb.RoleRef.Name, clusterScope)
			}
		}
	}

	// Fetch ClusterRoleBindings
	clusterRoleBindings, err := s.k8sClient.ListClusterRoleBindings(ctx)
	if err == nil {
		for _, crb := range clusterRoleBindings {
			roleID := fmt.Sprintf("ClusterRole/%s", crb.RoleRef.Name)
			addNode(RBACNode{
				ID:   roleID,
				Kind: "ClusterRole",
				Name: crb.RoleRef.Name,
			})

			for _, subject := range crb.Subjects {
				if subjectKindFilter != "" && subject.Kind != subjectKindFilter {
					continue
				}
				subjectID := fmt.Sprintf("%s/%s/%s", subject.Kind, subject.Namespace, subject.Name)
				addNode(RBACNode{
					ID:        subjectID,
					Kind:      subject.Kind,
					Name:      subject.Name,
					Namespace: subject.Namespace,
				})
				edges = append(edges, RBACEdge{
					Source:      subjectID,
					Target:      roleID,
					BindingName: crb.Name,
					BindingKind: "ClusterRoleBinding",
				})
				addSubjectRole(subjectID, subject.Kind, subject.Name, subject.Namespace, crb.RoleRef.Name, true)
			}
		}
	}

	// Build subjects list from map
	var subjects []RBACSubjectInfo
	for _, si := range subjectRoles {
		subjects = append(subjects, *si)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(RBACVisualizationResponse{
		Nodes:    nodes,
		Edges:    edges,
		Subjects: subjects,
	})
}
