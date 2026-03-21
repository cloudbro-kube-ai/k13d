package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
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
		writeMethodNotAllowed(w)
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

// RBACPolicyRule represents a single RBAC policy rule.
type RBACPolicyRule struct {
	Verbs     []string `json:"verbs"`
	Resources []string `json:"resources"`
	APIGroups []string `json:"api_groups"`
}

// RBACBindingDetail represents a binding with its resolved role rules.
type RBACBindingDetail struct {
	BindingName string           `json:"binding_name"`
	BindingKind string           `json:"binding_kind"`
	RoleName    string           `json:"role_name"`
	RoleKind    string           `json:"role_kind"`
	Namespace   string           `json:"namespace,omitempty"`
	Rules       []RBACPolicyRule `json:"rules"`
}

// RBACSubjectDetailResponse is the response for the subject detail endpoint.
type RBACSubjectDetailResponse struct {
	Name      string              `json:"name"`
	Kind      string              `json:"kind"`
	Namespace string              `json:"namespace,omitempty"`
	Bindings  []RBACBindingDetail `json:"bindings"`
}

func (s *Server) handleRBACSubjectDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	name := r.URL.Query().Get("name")
	kind := r.URL.Query().Get("kind")
	namespace := r.URL.Query().Get("namespace")
	if name == "" || kind == "" {
		WriteError(w, NewAPIError(ErrCodeValidation, "name and kind are required"))
		return
	}

	ctx := r.Context()

	// Fetch roles and bindings in parallel
	var wg sync.WaitGroup
	roleRulesMap := make(map[string][]RBACPolicyRule) // "Role/ns/name" or "ClusterRole/name" -> rules
	var rolesMu sync.Mutex

	wg.Add(2)
	go func() {
		defer wg.Done()
		rs, err := s.k8sClient.ListRoles(ctx, namespace)
		if err != nil {
			return
		}
		rolesMu.Lock()
		for _, role := range rs {
			key := fmt.Sprintf("Role/%s/%s", role.Namespace, role.Name)
			var rules []RBACPolicyRule
			for _, r := range role.Rules {
				rules = append(rules, RBACPolicyRule{
					Verbs:     r.Verbs,
					Resources: r.Resources,
					APIGroups: r.APIGroups,
				})
			}
			roleRulesMap[key] = rules
		}
		rolesMu.Unlock()
	}()
	go func() {
		defer wg.Done()
		crs, err := s.k8sClient.ListClusterRoles(ctx)
		if err != nil {
			return
		}
		rolesMu.Lock()
		for _, cr := range crs {
			key := fmt.Sprintf("ClusterRole/%s", cr.Name)
			var rules []RBACPolicyRule
			for _, r := range cr.Rules {
				rules = append(rules, RBACPolicyRule{
					Verbs:     r.Verbs,
					Resources: r.Resources,
					APIGroups: r.APIGroups,
				})
			}
			roleRulesMap[key] = rules
		}
		rolesMu.Unlock()
	}()
	wg.Wait()

	// Find matching bindings for the target subject
	var bindings []RBACBindingDetail

	rbs, _ := s.k8sClient.ListRoleBindings(ctx, namespace)
	for _, rb := range rbs {
		for _, subject := range rb.Subjects {
			if subject.Kind != kind || subject.Name != name {
				continue
			}
			if kind == "ServiceAccount" && namespace != "" && subject.Namespace != namespace {
				continue
			}
			roleKey := fmt.Sprintf("Role/%s/%s", rb.Namespace, rb.RoleRef.Name)
			roleKind := rb.RoleRef.Kind
			if roleKind == "ClusterRole" {
				roleKey = fmt.Sprintf("ClusterRole/%s", rb.RoleRef.Name)
			}
			bindings = append(bindings, RBACBindingDetail{
				BindingName: rb.Name,
				BindingKind: "RoleBinding",
				RoleName:    rb.RoleRef.Name,
				RoleKind:    roleKind,
				Namespace:   rb.Namespace,
				Rules:       roleRulesMap[roleKey],
			})
			break
		}
	}

	crbs, _ := s.k8sClient.ListClusterRoleBindings(ctx)
	for _, crb := range crbs {
		for _, subject := range crb.Subjects {
			if subject.Kind != kind || subject.Name != name {
				continue
			}
			if kind == "ServiceAccount" && namespace != "" && subject.Namespace != namespace {
				continue
			}
			roleKey := fmt.Sprintf("ClusterRole/%s", crb.RoleRef.Name)
			bindings = append(bindings, RBACBindingDetail{
				BindingName: crb.Name,
				BindingKind: "ClusterRoleBinding",
				RoleName:    crb.RoleRef.Name,
				RoleKind:    "ClusterRole",
				Rules:       roleRulesMap[roleKey],
			})
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(RBACSubjectDetailResponse{
		Name:      name,
		Kind:      kind,
		Namespace: namespace,
		Bindings:  bindings,
	})
}
