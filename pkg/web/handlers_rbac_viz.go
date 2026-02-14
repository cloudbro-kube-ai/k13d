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

// RBACVisualizationResponse is the response for the RBAC visualization endpoint
type RBACVisualizationResponse struct {
	Nodes []RBACNode `json:"nodes"`
	Edges []RBACEdge `json:"edges"`
}

func (s *Server) handleRBACVisualization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	namespace := r.URL.Query().Get("namespace")
	ctx := r.Context()

	var (
		mu    sync.Mutex
		nodes []RBACNode
		edges []RBACEdge
	)
	nodeSet := make(map[string]bool)

	addNode := func(n RBACNode) {
		mu.Lock()
		defer mu.Unlock()
		if !nodeSet[n.ID] {
			nodes = append(nodes, n)
			nodeSet[n.ID] = true
		}
	}
	addEdge := func(e RBACEdge) {
		mu.Lock()
		defer mu.Unlock()
		edges = append(edges, e)
	}

	// Fetch RoleBindings (namespaced)
	roleBindings, err := s.k8sClient.ListRoleBindings(ctx, namespace)
	if err == nil {
		for _, rb := range roleBindings {
			// Add role node
			roleID := fmt.Sprintf("Role/%s/%s", rb.Namespace, rb.RoleRef.Name)
			if rb.RoleRef.Kind == "ClusterRole" {
				roleID = fmt.Sprintf("ClusterRole/%s", rb.RoleRef.Name)
			}
			addNode(RBACNode{
				ID:        roleID,
				Kind:      rb.RoleRef.Kind,
				Name:      rb.RoleRef.Name,
				Namespace: rb.Namespace,
			})

			// Add subject nodes and edges
			for _, subject := range rb.Subjects {
				subjectID := fmt.Sprintf("%s/%s/%s", subject.Kind, subject.Namespace, subject.Name)
				addNode(RBACNode{
					ID:        subjectID,
					Kind:      subject.Kind,
					Name:      subject.Name,
					Namespace: subject.Namespace,
				})
				addEdge(RBACEdge{
					Source:      subjectID,
					Target:      roleID,
					BindingName: rb.Name,
					BindingKind: "RoleBinding",
					Namespace:   rb.Namespace,
				})
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
				subjectID := fmt.Sprintf("%s/%s/%s", subject.Kind, subject.Namespace, subject.Name)
				addNode(RBACNode{
					ID:        subjectID,
					Kind:      subject.Kind,
					Name:      subject.Name,
					Namespace: subject.Namespace,
				})
				addEdge(RBACEdge{
					Source:      subjectID,
					Target:      roleID,
					BindingName: crb.Name,
					BindingKind: "ClusterRoleBinding",
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(RBACVisualizationResponse{
		Nodes: nodes,
		Edges: edges,
	})
}
