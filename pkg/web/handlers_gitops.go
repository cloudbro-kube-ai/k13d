package web

import (
	"encoding/json"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GitOpsApplication represents an ArgoCD or Flux application
type GitOpsApplication struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Status     string `json:"status"`
	SyncStatus string `json:"syncStatus,omitempty"`
	Source     string `json:"source,omitempty"`
	Revision   string `json:"revision,omitempty"`
	Message    string `json:"message,omitempty"`
}

// GitOpsStatusResponse is the response for the GitOps status endpoint
type GitOpsStatusResponse struct {
	ArgoCD  []GitOpsApplication `json:"argocd"`
	Flux    []GitOpsApplication `json:"flux"`
	Message string              `json:"message,omitempty"`
}

func (s *Server) handleGitOpsStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	namespace := r.URL.Query().Get("namespace")
	resp := GitOpsStatusResponse{}

	// Try to query ArgoCD Application CRDs
	argoGVR := schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "applications",
	}

	var argoList interface{}
	var argoErr error
	if namespace != "" {
		argoList, argoErr = s.k8sClient.Dynamic.Resource(argoGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	} else {
		argoList, argoErr = s.k8sClient.Dynamic.Resource(argoGVR).List(ctx, metav1.ListOptions{})
	}

	if argoErr == nil {
		items := extractUnstructuredItems(argoList)
		for _, item := range items {
			app := GitOpsApplication{
				Name:      getNestedString(item, "metadata", "name"),
				Namespace: getNestedString(item, "metadata", "namespace"),
			}
			// Extract status
			app.Status = getNestedString(item, "status", "health", "status")
			app.SyncStatus = getNestedString(item, "status", "sync", "status")
			app.Source = getNestedString(item, "spec", "source", "repoURL")
			app.Revision = getNestedString(item, "status", "sync", "revision")
			if app.Status == "" {
				app.Status = "Unknown"
			}
			resp.ArgoCD = append(resp.ArgoCD, app)
		}
	}

	// Try to query Flux Kustomization CRDs
	fluxGVR := schema.GroupVersionResource{
		Group:    "kustomize.toolkit.fluxcd.io",
		Version:  "v1",
		Resource: "kustomizations",
	}

	var fluxList interface{}
	var fluxErr error
	if namespace != "" {
		fluxList, fluxErr = s.k8sClient.Dynamic.Resource(fluxGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	} else {
		fluxList, fluxErr = s.k8sClient.Dynamic.Resource(fluxGVR).List(ctx, metav1.ListOptions{})
	}

	if fluxErr == nil {
		items := extractUnstructuredItems(fluxList)
		for _, item := range items {
			app := GitOpsApplication{
				Name:      getNestedString(item, "metadata", "name"),
				Namespace: getNestedString(item, "metadata", "namespace"),
			}
			app.Source = getNestedString(item, "spec", "sourceRef", "name")
			// Extract status from conditions
			conditions := getNestedSlice(item, "status", "conditions")
			for _, cond := range conditions {
				if cm, ok := cond.(map[string]interface{}); ok {
					if cm["type"] == "Ready" {
						if cm["status"] == "True" {
							app.Status = "Ready"
						} else {
							app.Status = "NotReady"
						}
						if msg, ok := cm["message"].(string); ok {
							app.Message = msg
						}
					}
				}
			}
			if app.Status == "" {
				app.Status = "Unknown"
			}
			resp.Flux = append(resp.Flux, app)
		}
	}

	if argoErr != nil && fluxErr != nil {
		resp.Message = "Neither ArgoCD nor Flux CRDs found in cluster"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// extractUnstructuredItems extracts items from an unstructured list
func extractUnstructuredItems(list interface{}) []map[string]interface{} {
	// Access via reflection-free approach
	if m, ok := list.(interface{ UnstructuredContent() map[string]interface{} }); ok {
		content := m.UnstructuredContent()
		if items, ok := content["items"].([]interface{}); ok {
			var result []map[string]interface{}
			for _, item := range items {
				if obj, ok := item.(map[string]interface{}); ok {
					result = append(result, obj)
				}
			}
			return result
		}
	}
	return nil
}

// getNestedString safely gets a nested string value from a map
func getNestedString(obj map[string]interface{}, fields ...string) string {
	current := obj
	for i, field := range fields {
		if i == len(fields)-1 {
			if val, ok := current[field].(string); ok {
				return val
			}
			return ""
		}
		if next, ok := current[field].(map[string]interface{}); ok {
			current = next
		} else {
			return ""
		}
	}
	return ""
}

// getNestedSlice safely gets a nested slice value from a map
func getNestedSlice(obj map[string]interface{}, fields ...string) []interface{} {
	current := obj
	for i, field := range fields {
		if i == len(fields)-1 {
			if val, ok := current[field].([]interface{}); ok {
				return val
			}
			return nil
		}
		if next, ok := current[field].(map[string]interface{}); ok {
			current = next
		} else {
			return nil
		}
	}
	return nil
}
