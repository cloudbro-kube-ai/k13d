package web

import (
	"encoding/json"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// VeleroBackup represents a Velero backup resource
type VeleroBackup struct {
	Name           string `json:"name"`
	Namespace      string `json:"namespace"`
	Status         string `json:"status"`
	Created        string `json:"created,omitempty"`
	Expiration     string `json:"expiration,omitempty"`
	IncludedNS     string `json:"includedNamespaces,omitempty"`
	StorageLocation string `json:"storageLocation,omitempty"`
}

// VeleroSchedule represents a Velero schedule resource
type VeleroSchedule struct {
	Name           string `json:"name"`
	Namespace      string `json:"namespace"`
	Schedule       string `json:"schedule"`
	LastBackup     string `json:"lastBackup,omitempty"`
	Status         string `json:"status"`
}

// VeleroResponse is the response for Velero endpoints
type VeleroResponse struct {
	Installed bool        `json:"installed"`
	Items     interface{} `json:"items,omitempty"`
	Message   string      `json:"message,omitempty"`
}

func (s *Server) handleVeleroBackups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	backupGVR := schema.GroupVersionResource{
		Group:    "velero.io",
		Version:  "v1",
		Resource: "backups",
	}

	list, err := s.k8sClient.Dynamic.Resource(backupGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(VeleroResponse{
			Installed: false,
			Message:   "Velero not found in cluster",
		})
		return
	}

	var backups []VeleroBackup
	for _, item := range list.Items {
		backup := VeleroBackup{
			Name:      getNestedString(item.Object, "metadata", "name"),
			Namespace: getNestedString(item.Object, "metadata", "namespace"),
			Status:    getNestedString(item.Object, "status", "phase"),
			Created:   getNestedString(item.Object, "metadata", "creationTimestamp"),
		}
		backup.Expiration = getNestedString(item.Object, "status", "expiration")
		backup.StorageLocation = getNestedString(item.Object, "spec", "storageLocation")

		// Extract included namespaces
		if ns := getNestedSlice(item.Object, "spec", "includedNamespaces"); len(ns) > 0 {
			first, _ := ns[0].(string)
			backup.IncludedNS = first
		}

		if backup.Status == "" {
			backup.Status = "Unknown"
		}
		backups = append(backups, backup)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(VeleroResponse{
		Installed: true,
		Items:     backups,
	})
}

func (s *Server) handleVeleroSchedules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	scheduleGVR := schema.GroupVersionResource{
		Group:    "velero.io",
		Version:  "v1",
		Resource: "schedules",
	}

	list, err := s.k8sClient.Dynamic.Resource(scheduleGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(VeleroResponse{
			Installed: false,
			Message:   "Velero not found in cluster",
		})
		return
	}

	var schedules []VeleroSchedule
	for _, item := range list.Items {
		schedule := VeleroSchedule{
			Name:      getNestedString(item.Object, "metadata", "name"),
			Namespace: getNestedString(item.Object, "metadata", "namespace"),
			Schedule:  getNestedString(item.Object, "spec", "schedule"),
			Status:    getNestedString(item.Object, "status", "phase"),
		}
		schedule.LastBackup = getNestedString(item.Object, "status", "lastBackup")

		if schedule.Status == "" {
			schedule.Status = "Active"
		}
		schedules = append(schedules, schedule)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(VeleroResponse{
		Installed: true,
		Items:     schedules,
	})
}
