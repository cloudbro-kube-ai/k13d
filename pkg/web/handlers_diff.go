package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	goyaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DiffResponse is the response for the resource diff endpoint
type DiffResponse struct {
	Resource    string `json:"resource"`
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	CurrentYAML string `json:"currentYaml"`
	LastApplied string `json:"lastApplied"`
	HasDiff     bool   `json:"hasDiff"`
}

func (s *Server) handleResourceDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Resource  string `json:"resource"`
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Resource == "" || req.Name == "" {
		http.Error(w, "Resource type and name are required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Get the GVR for the resource type
	gvr, ok := s.k8sClient.GetGVR(req.Resource)
	if !ok {
		http.Error(w, "Unknown resource type: "+req.Resource, http.StatusBadRequest)
		return
	}

	// Get the current resource
	obj, err := s.k8sClient.Dynamic.Resource(gvr).Namespace(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		http.Error(w, "Failed to get resource: "+err.Error(), http.StatusNotFound)
		return
	}

	// Clean up managed fields for readability
	obj.SetManagedFields(nil)
	obj.SetResourceVersion("")

	currentYAML, err := goyaml.Marshal(obj.Object)
	if err != nil {
		http.Error(w, "Failed to marshal current YAML", http.StatusInternalServerError)
		return
	}

	// Extract last-applied-configuration annotation
	annotations := obj.GetAnnotations()
	lastApplied := ""
	hasDiff := false

	if annotations != nil {
		if la, ok := annotations["kubectl.kubernetes.io/last-applied-configuration"]; ok {
			// Pretty-print the last-applied JSON as YAML
			var lastAppliedObj map[string]interface{}
			if err := json.Unmarshal([]byte(la), &lastAppliedObj); err == nil {
				yamlBytes, err := goyaml.Marshal(lastAppliedObj)
				if err == nil {
					lastApplied = string(yamlBytes)
				} else {
					lastApplied = la
				}
			} else {
				lastApplied = la
			}
			hasDiff = strings.TrimSpace(string(currentYAML)) != strings.TrimSpace(lastApplied)
		}
	}

	if lastApplied == "" {
		lastApplied = "No last-applied-configuration annotation found"
	}

	// Map resource alias to full name for display
	resourceName := mapResourceName(req.Resource, gvr)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DiffResponse{
		Resource:    resourceName,
		Name:        req.Name,
		Namespace:   req.Namespace,
		CurrentYAML: string(currentYAML),
		LastApplied: lastApplied,
		HasDiff:     hasDiff,
	})
}

// mapResourceName returns a human-readable resource name
func mapResourceName(alias string, gvr schema.GroupVersionResource) string {
	if gvr.Group != "" {
		return fmt.Sprintf("%s.%s", gvr.Resource, gvr.Group)
	}
	return gvr.Resource
}
