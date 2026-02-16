package web

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/db"
	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ==========================================
// K8s Resource Handlers
// ==========================================

func (s *Server) handleK8sResource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/k8s/")
	parts := strings.Split(path, "/")
	resource := parts[0]

	namespace := r.URL.Query().Get("namespace")
	name := r.URL.Query().Get("name")
	format := r.URL.Query().Get("format")               // "yaml" for YAML output
	labelSelector := r.URL.Query().Get("labelSelector") // for filtering by labels
	// Empty namespace means "all namespaces" - don't default to "default"

	username := r.Header.Get("X-Username")
	if username == "" {
		username = "anonymous"
	}

	// Record audit log (view actions are skipped by default)
	db.RecordAudit(db.AuditEntry{
		User:       username,
		Action:     "view",
		ActionType: db.ActionTypeView, // Will be skipped unless config.IncludeViews is true
		Resource:   resource,
		Details:    fmt.Sprintf("namespace=%s, name=%s", namespace, name),
	})

	// If name is specified and format is yaml, return YAML for single resource
	if name != "" && format == "yaml" {
		s.handleResourceYAML(w, r, resource, namespace, name)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var items []map[string]interface{}
	var err error

	switch resource {
	case "pods":
		var pods []corev1.Pod
		if labelSelector != "" {
			// Use label selector if provided
			podList, e := s.k8sClient.Clientset.CoreV1().Pods(namespace).List(r.Context(), metav1.ListOptions{
				LabelSelector: labelSelector,
			})
			if e != nil {
				err = e
			} else {
				pods = podList.Items
			}
		} else {
			pods, err = s.k8sClient.ListPods(r.Context(), namespace)
		}
		if err == nil {
			items = make([]map[string]interface{}, len(pods))
			for i, pod := range pods {
				// Extract container names
				containers := make([]string, len(pod.Spec.Containers))
				for j, c := range pod.Spec.Containers {
					containers[j] = c.Name
				}
				items[i] = map[string]interface{}{
					"name":       pod.Name,
					"namespace":  pod.Namespace,
					"status":     string(pod.Status.Phase),
					"ready":      getPodReadyCount(&pod),
					"restarts":   getPodRestarts(&pod),
					"age":        formatAge(pod.CreationTimestamp.Time),
					"node":       pod.Spec.NodeName,
					"ip":         pod.Status.PodIP,
					"containers": containers,
				}
			}
		}

	case "deployments":
		deps, e := s.k8sClient.ListDeployments(r.Context(), namespace)
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(deps))
			for i, dep := range deps {
				replicas := int32(1)
				if dep.Spec.Replicas != nil {
					replicas = *dep.Spec.Replicas
				}
				items[i] = map[string]interface{}{
					"name":      dep.Name,
					"namespace": dep.Namespace,
					"ready":     fmt.Sprintf("%d/%d", dep.Status.ReadyReplicas, replicas),
					"upToDate":  dep.Status.UpdatedReplicas,
					"available": dep.Status.AvailableReplicas,
					"age":       formatAge(dep.CreationTimestamp.Time),
					"selector":  formatLabelSelector(dep.Spec.Selector),
				}
			}
		}

	case "services":
		svcs, e := s.k8sClient.ListServices(r.Context(), namespace)
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(svcs))
			for i, svc := range svcs {
				ports := make([]string, len(svc.Spec.Ports))
				for j, p := range svc.Spec.Ports {
					ports[j] = fmt.Sprintf("%d/%s", p.Port, p.Protocol)
				}
				items[i] = map[string]interface{}{
					"name":       svc.Name,
					"namespace":  svc.Namespace,
					"type":       string(svc.Spec.Type),
					"clusterIP":  svc.Spec.ClusterIP,
					"externalIP": getExternalIPs(&svc),
					"ports":      strings.Join(ports, ", "),
					"age":        formatAge(svc.CreationTimestamp.Time),
				}
			}
		}

	case "namespaces":
		nss, e := s.k8sClient.ListNamespaces(r.Context())
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(nss))
			for i, ns := range nss {
				items[i] = map[string]interface{}{
					"name":   ns.Name,
					"status": string(ns.Status.Phase),
					"age":    formatAge(ns.CreationTimestamp.Time),
				}
			}
		}

	case "nodes":
		nodes, e := s.k8sClient.ListNodes(r.Context())
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(nodes))
			for i, node := range nodes {
				items[i] = map[string]interface{}{
					"name":    node.Name,
					"status":  getNodeStatus(&node),
					"roles":   getNodeRoles(&node),
					"version": node.Status.NodeInfo.KubeletVersion,
					"age":     formatAge(node.CreationTimestamp.Time),
				}
			}
		}

	case "events":
		events, e := s.k8sClient.ListEvents(r.Context(), namespace)
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(events))
			for i, ev := range events {
				items[i] = map[string]interface{}{
					"name":      ev.Name,
					"namespace": ev.Namespace,
					"type":      ev.Type,
					"reason":    ev.Reason,
					"message":   ev.Message,
					"count":     ev.Count,
					"lastSeen":  ev.LastTimestamp.Time.Format(time.RFC3339),
					"involvedObject": map[string]interface{}{
						"kind":      ev.InvolvedObject.Kind,
						"name":      ev.InvolvedObject.Name,
						"namespace": ev.InvolvedObject.Namespace,
					},
				}
			}
		}

	case "statefulsets":
		sts, e := s.k8sClient.ListStatefulSets(r.Context(), namespace)
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(sts))
			for i, st := range sts {
				replicas := int32(1)
				if st.Spec.Replicas != nil {
					replicas = *st.Spec.Replicas
				}
				items[i] = map[string]interface{}{
					"name":      st.Name,
					"namespace": st.Namespace,
					"ready":     fmt.Sprintf("%d/%d", st.Status.ReadyReplicas, replicas),
					"age":       formatAge(st.CreationTimestamp.Time),
					"selector":  formatLabelSelector(st.Spec.Selector),
				}
			}
		}

	case "daemonsets":
		ds, e := s.k8sClient.ListDaemonSets(r.Context(), namespace)
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(ds))
			for i, d := range ds {
				items[i] = map[string]interface{}{
					"name":         d.Name,
					"namespace":    d.Namespace,
					"desired":      d.Status.DesiredNumberScheduled,
					"current":      d.Status.CurrentNumberScheduled,
					"ready":        d.Status.NumberReady,
					"upToDate":     d.Status.UpdatedNumberScheduled,
					"available":    d.Status.NumberAvailable,
					"nodeSelector": formatNodeSelector(d.Spec.Template.Spec.NodeSelector),
					"age":          formatAge(d.CreationTimestamp.Time),
					"selector":     formatLabelSelector(d.Spec.Selector),
				}
			}
		}

	case "configmaps":
		cms, e := s.k8sClient.ListConfigMaps(r.Context(), namespace)
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(cms))
			for i, cm := range cms {
				items[i] = map[string]interface{}{
					"name":      cm.Name,
					"namespace": cm.Namespace,
					"data":      len(cm.Data),
					"age":       formatAge(cm.CreationTimestamp.Time),
				}
			}
		}

	case "secrets":
		secrets, e := s.k8sClient.ListSecrets(r.Context(), namespace)
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(secrets))
			for i, sec := range secrets {
				items[i] = map[string]interface{}{
					"name":      sec.Name,
					"namespace": sec.Namespace,
					"type":      string(sec.Type),
					"data":      len(sec.Data),
					"age":       formatAge(sec.CreationTimestamp.Time),
				}
			}
		}

	case "ingresses":
		ings, e := s.k8sClient.ListIngresses(r.Context(), namespace)
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(ings))
			for i, ing := range ings {
				hosts := []string{}
				for _, rule := range ing.Spec.Rules {
					hosts = append(hosts, rule.Host)
				}
				items[i] = map[string]interface{}{
					"name":      ing.Name,
					"namespace": ing.Namespace,
					"class":     getIngressClass(&ing),
					"hosts":     strings.Join(hosts, ", "),
					"address":   getIngressAddress(&ing),
					"age":       formatAge(ing.CreationTimestamp.Time),
				}
			}
		}

	case "clusterroles":
		roles, e := s.k8sClient.ListClusterRoles(r.Context())
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(roles))
			for i, role := range roles {
				items[i] = map[string]interface{}{
					"name":  role.Name,
					"rules": len(role.Rules),
					"age":   formatAge(role.CreationTimestamp.Time),
				}
			}
		}

	case "clusterrolebindings":
		bindings, e := s.k8sClient.ListClusterRoleBindings(r.Context())
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(bindings))
			for i, b := range bindings {
				subjects := []string{}
				for _, subj := range b.Subjects {
					subjects = append(subjects, fmt.Sprintf("%s/%s", subj.Kind, subj.Name))
				}
				items[i] = map[string]interface{}{
					"name":     b.Name,
					"role":     b.RoleRef.Name,
					"subjects": strings.Join(subjects, ", "),
					"age":      formatAge(b.CreationTimestamp.Time),
				}
			}
		}

	case "roles":
		roles, e := s.k8sClient.ListRoles(r.Context(), namespace)
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(roles))
			for i, role := range roles {
				items[i] = map[string]interface{}{
					"name":      role.Name,
					"namespace": role.Namespace,
					"rules":     len(role.Rules),
					"age":       formatAge(role.CreationTimestamp.Time),
				}
			}
		}

	case "rolebindings":
		bindings, e := s.k8sClient.ListRoleBindings(r.Context(), namespace)
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(bindings))
			for i, b := range bindings {
				subjects := []string{}
				for _, subj := range b.Subjects {
					subjects = append(subjects, fmt.Sprintf("%s/%s", subj.Kind, subj.Name))
				}
				items[i] = map[string]interface{}{
					"name":      b.Name,
					"namespace": b.Namespace,
					"role":      b.RoleRef.Name,
					"subjects":  strings.Join(subjects, ", "),
					"age":       formatAge(b.CreationTimestamp.Time),
				}
			}
		}

	case "serviceaccounts":
		sas, e := s.k8sClient.ListServiceAccounts(r.Context(), namespace)
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(sas))
			for i, sa := range sas {
				items[i] = map[string]interface{}{
					"name":      sa.Name,
					"namespace": sa.Namespace,
					"secrets":   len(sa.Secrets),
					"age":       formatAge(sa.CreationTimestamp.Time),
				}
			}
		}

	case "persistentvolumes", "pv":
		pvs, e := s.k8sClient.ListPersistentVolumes(r.Context())
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(pvs))
			for i, pv := range pvs {
				claim := "<none>"
				if pv.Spec.ClaimRef != nil {
					claim = fmt.Sprintf("%s/%s", pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name)
				}
				items[i] = map[string]interface{}{
					"name":          pv.Name,
					"capacity":      pv.Spec.Capacity.Storage().String(),
					"accessModes":   getAccessModes(pv.Spec.AccessModes),
					"reclaimPolicy": string(pv.Spec.PersistentVolumeReclaimPolicy),
					"status":        string(pv.Status.Phase),
					"claim":         claim,
					"storageClass":  pv.Spec.StorageClassName,
					"age":           formatAge(pv.CreationTimestamp.Time),
				}
			}
		}

	case "persistentvolumeclaims", "pvc":
		pvcs, e := s.k8sClient.ListPersistentVolumeClaims(r.Context(), namespace)
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(pvcs))
			for i, pvc := range pvcs {
				capacity := "<pending>"
				if pvc.Status.Capacity != nil {
					if storage, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
						capacity = storage.String()
					}
				}
				items[i] = map[string]interface{}{
					"name":         pvc.Name,
					"namespace":    pvc.Namespace,
					"status":       string(pvc.Status.Phase),
					"volume":       pvc.Spec.VolumeName,
					"capacity":     capacity,
					"accessModes":  getAccessModes(pvc.Spec.AccessModes),
					"storageClass": getStorageClass(pvc.Spec.StorageClassName),
					"age":          formatAge(pvc.CreationTimestamp.Time),
				}
			}
		}

	case "storageclasses", "sc":
		scs, e := s.k8sClient.ListStorageClasses(r.Context())
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(scs))
			for i, sc := range scs {
				items[i] = map[string]interface{}{
					"name":          sc.Name,
					"provisioner":   sc.Provisioner,
					"reclaimPolicy": getReclaimPolicy(sc.ReclaimPolicy),
					"volumeBinding": string(*sc.VolumeBindingMode),
					"allowExpand":   sc.AllowVolumeExpansion != nil && *sc.AllowVolumeExpansion,
					"age":           formatAge(sc.CreationTimestamp.Time),
				}
			}
		}

	case "jobs":
		jobs, e := s.k8sClient.ListJobs(r.Context(), namespace)
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(jobs))
			for i, job := range jobs {
				items[i] = map[string]interface{}{
					"name":        job.Name,
					"namespace":   job.Namespace,
					"completions": getJobCompletions(&job),
					"duration":    getJobDuration(&job),
					"age":         formatAge(job.CreationTimestamp.Time),
				}
			}
		}

	case "cronjobs":
		cjs, e := s.k8sClient.ListCronJobs(r.Context(), namespace)
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(cjs))
			for i, cj := range cjs {
				lastSchedule := "<never>"
				if cj.Status.LastScheduleTime != nil {
					lastSchedule = formatAge(cj.Status.LastScheduleTime.Time) + " ago"
				}
				items[i] = map[string]interface{}{
					"name":         cj.Name,
					"namespace":    cj.Namespace,
					"schedule":     cj.Spec.Schedule,
					"suspend":      cj.Spec.Suspend != nil && *cj.Spec.Suspend,
					"active":       len(cj.Status.Active),
					"lastSchedule": lastSchedule,
					"age":          formatAge(cj.CreationTimestamp.Time),
				}
			}
		}

	case "replicasets", "rs":
		rs, e := s.k8sClient.ListReplicaSets(r.Context(), namespace)
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(rs))
			for i, rset := range rs {
				replicas := int32(0)
				if rset.Spec.Replicas != nil {
					replicas = *rset.Spec.Replicas
				}
				items[i] = map[string]interface{}{
					"name":      rset.Name,
					"namespace": rset.Namespace,
					"desired":   replicas,
					"current":   rset.Status.Replicas,
					"ready":     fmt.Sprintf("%d/%d", rset.Status.ReadyReplicas, replicas),
					"age":       formatAge(rset.CreationTimestamp.Time),
					"selector":  formatLabelSelector(rset.Spec.Selector),
				}
			}
		}

	case "hpa", "horizontalpodautoscalers":
		hpas, e := s.k8sClient.ListHPAs(r.Context(), namespace)
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(hpas))
			for i, h := range hpas {
				items[i] = map[string]interface{}{
					"name":        h.Name,
					"namespace":   h.Namespace,
					"reference":   fmt.Sprintf("%s/%s", h.Spec.ScaleTargetRef.Kind, h.Spec.ScaleTargetRef.Name),
					"minReplicas": getMinReplicas(h.Spec.MinReplicas),
					"maxReplicas": h.Spec.MaxReplicas,
					"replicas":    h.Status.CurrentReplicas,
					"age":         formatAge(h.CreationTimestamp.Time),
				}
			}
		}

	case "networkpolicies", "netpol":
		policies, e := s.k8sClient.ListNetworkPolicies(r.Context(), namespace)
		err = e
		if err == nil {
			items = make([]map[string]interface{}, len(policies))
			for i, np := range policies {
				items[i] = map[string]interface{}{
					"name":        np.Name,
					"namespace":   np.Namespace,
					"podSelector": formatLabelSelector(&np.Spec.PodSelector),
					"age":         formatAge(np.CreationTimestamp.Time),
				}
			}
		}

	default:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(K8sResourceResponse{
			Kind:      resource,
			Items:     []map[string]interface{}{},
			Error:     fmt.Sprintf("Unknown resource type: %s", resource),
			Timestamp: time.Now(),
		})
		return
	}

	if err != nil {
		json.NewEncoder(w).Encode(K8sResourceResponse{
			Kind:      resource,
			Error:     err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	json.NewEncoder(w).Encode(K8sResourceResponse{
		Kind:      resource,
		Items:     items,
		Timestamp: time.Now(),
	})
}

// handleResourceYAML returns YAML for a single resource
func (s *Server) handleResourceYAML(w http.ResponseWriter, r *http.Request, resource, namespace, name string) {
	// Map resource names to GVR
	gvr, ok := s.k8sClient.GetGVR(resource)
	if !ok {
		http.Error(w, fmt.Sprintf("Unknown resource type: %s", resource), http.StatusBadRequest)
		return
	}

	yaml, err := s.k8sClient.GetResourceYAML(r.Context(), namespace, name, gvr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get YAML: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(yaml))
}

// handleCustomResources handles Custom Resource API endpoints
// GET /api/crd/ - List all CRDs with details
// GET /api/crd/{crdName} - Get CRD info
// GET /api/crd/{crdName}/instances?namespace=xxx - List CR instances
// GET /api/crd/{crdName}/instances/{name}?namespace=xxx&format=yaml - Get CR instance (optionally as YAML)
func (s *Server) handleCustomResources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.k8sClient == nil {
		http.Error(w, "Kubernetes client not available", http.StatusServiceUnavailable)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/crd/")
	parts := strings.Split(path, "/")

	namespace := r.URL.Query().Get("namespace")
	format := r.URL.Query().Get("format")

	w.Header().Set("Content-Type", "application/json")

	// GET /api/crd/ - List all CRDs
	if path == "" || path == "/" {
		crds, err := s.k8sClient.ListCRDsDetailed(r.Context())
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": err.Error(),
				"items": []interface{}{},
			})
			return
		}

		// Convert to response format
		items := make([]map[string]interface{}, len(crds))
		for i, crd := range crds {
			items[i] = map[string]interface{}{
				"name":           crd.Name,
				"group":          crd.Group,
				"version":        crd.Version,
				"kind":           crd.Kind,
				"plural":         crd.Plural,
				"namespaced":     crd.Namespaced,
				"shortNames":     crd.ShortNames,
				"printerColumns": crd.PrinterColumns,
			}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"kind":  "CustomResourceDefinitionList",
			"items": items,
		})
		return
	}

	crdName := parts[0]

	// GET /api/crd/{crdName} - Get CRD info
	if len(parts) == 1 {
		crdInfo, err := s.k8sClient.GetCRDInfo(r.Context(), crdName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get CRD: %v", err), http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"name":           crdInfo.Name,
			"group":          crdInfo.Group,
			"version":        crdInfo.Version,
			"kind":           crdInfo.Kind,
			"plural":         crdInfo.Plural,
			"namespaced":     crdInfo.Namespaced,
			"shortNames":     crdInfo.ShortNames,
			"printerColumns": crdInfo.PrinterColumns,
		})
		return
	}

	// GET /api/crd/{crdName}/instances - List CR instances
	if parts[1] == "instances" {
		crdInfo, err := s.k8sClient.GetCRDInfo(r.Context(), crdName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get CRD: %v", err), http.StatusNotFound)
			return
		}

		// GET /api/crd/{crdName}/instances/{name} - Get single CR instance
		if len(parts) >= 3 && parts[2] != "" {
			instanceName := parts[2]

			if format == "yaml" {
				yamlStr, err := s.k8sClient.GetCustomResourceYAML(r.Context(), crdInfo, namespace, instanceName)
				if err != nil {
					http.Error(w, fmt.Sprintf("Failed to get CR YAML: %v", err), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.Write([]byte(yamlStr))
				return
			}

			cr, err := s.k8sClient.GetCustomResource(r.Context(), crdInfo, namespace, instanceName)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to get CR: %v", err), http.StatusNotFound)
				return
			}

			json.NewEncoder(w).Encode(cr.Object)
			return
		}

		// List all instances
		instances, err := s.k8sClient.ListCustomResources(r.Context(), crdInfo, namespace)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": err.Error(),
				"items": []interface{}{},
			})
			return
		}

		// Convert to response format
		items := make([]map[string]interface{}, len(instances))
		for i, inst := range instances {
			ns := inst.GetNamespace()
			age := formatAge(inst.GetCreationTimestamp().Time)

			items[i] = map[string]interface{}{
				"name":      inst.GetName(),
				"namespace": ns,
				"age":       age,
				"kind":      inst.GetKind(),
			}

			// Extract status.phase or status.state if available
			if status, ok := inst.Object["status"].(map[string]interface{}); ok {
				if phase, ok := status["phase"].(string); ok {
					items[i]["status"] = phase
				} else if state, ok := status["state"].(string); ok {
					items[i]["status"] = state
				} else if ready, ok := status["ready"].(bool); ok {
					if ready {
						items[i]["status"] = "Ready"
					} else {
						items[i]["status"] = "NotReady"
					}
				}
			}

			// Extract printer column values
			extraFields := make(map[string]string)
			for _, col := range crdInfo.PrinterColumns {
				// Skip Age (already provided) and Name (already provided)
				colKey := strings.ToLower(strings.ReplaceAll(col.Name, " ", "_"))
				if colKey == "age" || colKey == "name" || colKey == "namespace" {
					continue
				}
				val := k8s.ResolveJSONPath(inst.Object, col.JSONPath)
				if val != "" {
					extraFields[colKey] = val
					// Also set status from printer column if not already set
					if colKey == "status" || colKey == "phase" || colKey == "state" || colKey == "ready" {
						if _, exists := items[i]["status"]; !exists {
							items[i]["status"] = val
						}
					}
				}
			}
			if len(extraFields) > 0 {
				items[i]["extra"] = extraFields
			}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"kind":           crdInfo.Kind + "List",
			"crd":            crdInfo.Name,
			"namespaced":     crdInfo.Namespaced,
			"printerColumns": crdInfo.PrinterColumns,
			"items":          items,
		})
		return
	}

	http.Error(w, "Invalid path", http.StatusBadRequest)
}

// handlePodLogs handles GET /api/pods/{namespace}/{name}/logs
func (s *Server) handlePodLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /api/pods/{namespace}/{name}/logs
	path := strings.TrimPrefix(r.URL.Path, "/api/pods/")
	parts := strings.Split(path, "/")

	if len(parts) < 3 || parts[2] != "logs" {
		http.Error(w, "Invalid path. Expected /api/pods/{namespace}/{name}/logs", http.StatusBadRequest)
		return
	}

	namespace := parts[0]
	podName := parts[1]
	container := r.URL.Query().Get("container")
	tailLines := r.URL.Query().Get("tailLines")
	follow := r.URL.Query().Get("follow") == "true"
	previous := r.URL.Query().Get("previous") == "true"

	// Build log options
	opts := &corev1.PodLogOptions{
		Previous: previous,
	}

	if container != "" {
		opts.Container = container
	}

	if tailLines != "" {
		if lines, err := strconv.ParseInt(tailLines, 10, 64); err == nil {
			opts.TailLines = &lines
		}
	}

	if follow {
		opts.Follow = true
	}

	// Get logs from k8s client
	clientset := s.k8sClient.Clientset
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, opts)

	if follow {
		// Streaming logs
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		stream, err := req.Stream(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to stream logs: %v", err), http.StatusInternalServerError)
			return
		}
		defer stream.Close()

		scanner := bufio.NewScanner(stream)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintf(w, "data: %s\n\n", line)
			flusher.Flush()
		}
	} else {
		// Non-streaming logs
		stream, err := req.Stream(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get logs: %v", err), http.StatusInternalServerError)
			return
		}
		defer stream.Close()

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		scanner := bufio.NewScanner(stream)
		for scanner.Scan() {
			w.Write(scanner.Bytes())
			w.Write([]byte("\n"))
		}
	}
}

// handleWorkloadPods returns pods for a specific workload (deployment, daemonset, statefulset, replicaset)
func (s *Server) handleWorkloadPods(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	namespace := r.URL.Query().Get("namespace")
	kind := r.URL.Query().Get("kind")
	name := r.URL.Query().Get("name")

	if namespace == "" || kind == "" || name == "" {
		http.Error(w, "namespace, kind, and name parameters are required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	clientset := s.k8sClient.Clientset

	// Get the workload's selector
	var selector *metav1.LabelSelector

	switch strings.ToLower(kind) {
	case "deployment":
		dep, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get deployment: %v", err), http.StatusNotFound)
			return
		}
		selector = dep.Spec.Selector

	case "daemonset":
		ds, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get daemonset: %v", err), http.StatusNotFound)
			return
		}
		selector = ds.Spec.Selector

	case "statefulset":
		sts, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get statefulset: %v", err), http.StatusNotFound)
			return
		}
		selector = sts.Spec.Selector

	case "replicaset":
		rs, err := clientset.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get replicaset: %v", err), http.StatusNotFound)
			return
		}
		selector = rs.Spec.Selector

	default:
		http.Error(w, fmt.Sprintf("Unknown workload kind: %s", kind), http.StatusBadRequest)
		return
	}

	// List pods with the selector
	labelSelector := metav1.FormatLabelSelector(selector)
	podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list pods: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	pods := make([]map[string]interface{}, len(podList.Items))
	for i, pod := range podList.Items {
		pods[i] = map[string]interface{}{
			"name":      pod.Name,
			"namespace": pod.Namespace,
			"status":    string(pod.Status.Phase),
			"ready":     getPodReadyCount(&pod),
			"restarts":  getPodRestarts(&pod),
			"age":       formatAge(pod.CreationTimestamp.Time),
			"node":      pod.Spec.NodeName,
			"ip":        pod.Status.PodIP,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workload":  name,
		"kind":      kind,
		"namespace": namespace,
		"pods":      pods,
		"count":     len(pods),
	})
}

// handleClusterOverview returns a high-level cluster overview
func (s *Server) handleClusterOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Gather cluster stats in parallel
	var nodeCount, readyNodes int
	var podCount, runningPods int
	var deploymentCount, healthyDeployments int
	var namespaceCount int

	// Get node stats
	nodes, err := s.k8sClient.ListNodes(ctx)
	if err == nil {
		nodeCount = len(nodes)
		for _, node := range nodes {
			if getNodeStatus(&node) == "Ready" {
				readyNodes++
			}
		}
	}

	// Get pod stats
	pods, err := s.k8sClient.ListPods(ctx, "")
	if err == nil {
		podCount = len(pods)
		for _, pod := range pods {
			if pod.Status.Phase == corev1.PodRunning {
				runningPods++
			}
		}
	}

	// Get deployment stats
	deployments, err := s.k8sClient.ListDeployments(ctx, "")
	if err == nil {
		deploymentCount = len(deployments)
		for _, dep := range deployments {
			replicas := int32(1)
			if dep.Spec.Replicas != nil {
				replicas = *dep.Spec.Replicas
			}
			if dep.Status.ReadyReplicas == replicas {
				healthyDeployments++
			}
		}
	}

	// Get namespace count
	namespaces, err := s.k8sClient.ListNamespaces(ctx)
	if err == nil {
		namespaceCount = len(namespaces)
	}

	// Get current context info
	contextName, _ := s.k8sClient.GetCurrentContext()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"context": contextName,
		"nodes": map[string]interface{}{
			"total": nodeCount,
			"ready": readyNodes,
		},
		"pods": map[string]interface{}{
			"total":   podCount,
			"running": runningPods,
		},
		"deployments": map[string]interface{}{
			"total":   deploymentCount,
			"healthy": healthyDeployments,
		},
		"namespaces": namespaceCount,
		"timestamp":  time.Now(),
	})
}

// ==========================================
// Global Search Handler
// ==========================================

// SearchResult represents a single search result item
type SearchResult struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Status    string `json:"status,omitempty"`
	Age       string `json:"age,omitempty"`
}

// handleGlobalSearch searches across multiple resource types
func (s *Server) handleGlobalSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := strings.ToLower(r.URL.Query().Get("q"))
	namespace := r.URL.Query().Get("namespace")
	limitStr := r.URL.Query().Get("limit")

	if query == "" {
		http.Error(w, "Search query 'q' is required", http.StatusBadRequest)
		return
	}

	limit := 50 // Default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}

	var results []SearchResult
	ctx := r.Context()

	// Search Pods
	pods, err := s.k8sClient.ListPods(ctx, namespace)
	if err == nil {
		for _, pod := range pods {
			if strings.Contains(strings.ToLower(pod.Name), query) ||
				strings.Contains(strings.ToLower(pod.Namespace), query) {
				results = append(results, SearchResult{
					Kind:      "Pod",
					Name:      pod.Name,
					Namespace: pod.Namespace,
					Status:    string(pod.Status.Phase),
					Age:       formatAge(pod.CreationTimestamp.Time),
				})
			}
		}
	}

	// Search Deployments
	deployments, err := s.k8sClient.ListDeployments(ctx, namespace)
	if err == nil {
		for _, dep := range deployments {
			if strings.Contains(strings.ToLower(dep.Name), query) ||
				strings.Contains(strings.ToLower(dep.Namespace), query) {
				status := "Updating"
				if dep.Status.ReadyReplicas == dep.Status.Replicas {
					status = "Ready"
				}
				results = append(results, SearchResult{
					Kind:      "Deployment",
					Name:      dep.Name,
					Namespace: dep.Namespace,
					Status:    status,
					Age:       formatAge(dep.CreationTimestamp.Time),
				})
			}
		}
	}

	// Search Services
	services, err := s.k8sClient.ListServices(ctx, namespace)
	if err == nil {
		for _, svc := range services {
			if strings.Contains(strings.ToLower(svc.Name), query) ||
				strings.Contains(strings.ToLower(svc.Namespace), query) {
				results = append(results, SearchResult{
					Kind:      "Service",
					Name:      svc.Name,
					Namespace: svc.Namespace,
					Status:    string(svc.Spec.Type),
					Age:       formatAge(svc.CreationTimestamp.Time),
				})
			}
		}
	}

	// Search StatefulSets
	statefulsets, err := s.k8sClient.ListStatefulSets(ctx, namespace)
	if err == nil {
		for _, sts := range statefulsets {
			if strings.Contains(strings.ToLower(sts.Name), query) ||
				strings.Contains(strings.ToLower(sts.Namespace), query) {
				status := "Updating"
				if sts.Status.ReadyReplicas == sts.Status.Replicas {
					status = "Ready"
				}
				results = append(results, SearchResult{
					Kind:      "StatefulSet",
					Name:      sts.Name,
					Namespace: sts.Namespace,
					Status:    status,
					Age:       formatAge(sts.CreationTimestamp.Time),
				})
			}
		}
	}

	// Search DaemonSets
	daemonsets, err := s.k8sClient.ListDaemonSets(ctx, namespace)
	if err == nil {
		for _, ds := range daemonsets {
			if strings.Contains(strings.ToLower(ds.Name), query) ||
				strings.Contains(strings.ToLower(ds.Namespace), query) {
				status := "Updating"
				if ds.Status.NumberReady == ds.Status.DesiredNumberScheduled {
					status = "Ready"
				}
				results = append(results, SearchResult{
					Kind:      "DaemonSet",
					Name:      ds.Name,
					Namespace: ds.Namespace,
					Status:    status,
					Age:       formatAge(ds.CreationTimestamp.Time),
				})
			}
		}
	}

	// Search ConfigMaps
	configmaps, err := s.k8sClient.ListConfigMaps(ctx, namespace)
	if err == nil {
		for _, cm := range configmaps {
			if strings.Contains(strings.ToLower(cm.Name), query) ||
				strings.Contains(strings.ToLower(cm.Namespace), query) {
				results = append(results, SearchResult{
					Kind:      "ConfigMap",
					Name:      cm.Name,
					Namespace: cm.Namespace,
					Age:       formatAge(cm.CreationTimestamp.Time),
				})
			}
		}
	}

	// Search Secrets
	secrets, err := s.k8sClient.ListSecrets(ctx, namespace)
	if err == nil {
		for _, sec := range secrets {
			if strings.Contains(strings.ToLower(sec.Name), query) ||
				strings.Contains(strings.ToLower(sec.Namespace), query) {
				results = append(results, SearchResult{
					Kind:      "Secret",
					Name:      sec.Name,
					Namespace: sec.Namespace,
					Status:    string(sec.Type),
					Age:       formatAge(sec.CreationTimestamp.Time),
				})
			}
		}
	}

	// Search Ingresses
	ingresses, err := s.k8sClient.ListIngresses(ctx, namespace)
	if err == nil {
		for _, ing := range ingresses {
			if strings.Contains(strings.ToLower(ing.Name), query) ||
				strings.Contains(strings.ToLower(ing.Namespace), query) {
				results = append(results, SearchResult{
					Kind:      "Ingress",
					Name:      ing.Name,
					Namespace: ing.Namespace,
					Age:       formatAge(ing.CreationTimestamp.Time),
				})
			}
		}
	}

	// Search Nodes (no namespace)
	if namespace == "" {
		nodes, err := s.k8sClient.ListNodes(ctx)
		if err == nil {
			for _, node := range nodes {
				if strings.Contains(strings.ToLower(node.Name), query) {
					status := "NotReady"
					for _, cond := range node.Status.Conditions {
						if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
							status = "Ready"
							break
						}
					}
					results = append(results, SearchResult{
						Kind:   "Node",
						Name:   node.Name,
						Status: status,
						Age:    formatAge(node.CreationTimestamp.Time),
					})
				}
			}
		}
	}

	// Search Namespaces
	if namespace == "" {
		namespaces, err := s.k8sClient.ListNamespaces(ctx)
		if err == nil {
			for _, ns := range namespaces {
				if strings.Contains(strings.ToLower(ns.Name), query) {
					results = append(results, SearchResult{
						Kind:   "Namespace",
						Name:   ns.Name,
						Status: string(ns.Status.Phase),
						Age:    formatAge(ns.CreationTimestamp.Time),
					})
				}
			}
		}
	}

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	// Ensure results is never nil
	if results == nil {
		results = []SearchResult{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
		"total":   len(results),
		"query":   query,
	})
}

// YamlApplyRequest represents a request to apply YAML to the cluster
type YamlApplyRequest struct {
	YAML      string `json:"yaml"`
	Namespace string `json:"namespace"`
	DryRun    bool   `json:"dryRun"`
}

// handleYamlApply handles POST /api/k8s/apply for applying YAML manifests
func (s *Server) handleYamlApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req YamlApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid request body: " + err.Error(),
		})
		return
	}

	if req.YAML == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "YAML content is required",
		})
		return
	}

	username := r.Header.Get("X-Username")
	if username == "" {
		username = "anonymous"
	}

	// Apply YAML using kubectl
	result, err := s.k8sClient.ApplyYAML(r.Context(), req.YAML, req.Namespace, req.DryRun)

	// Record audit log
	actionType := db.ActionTypeMutation
	if req.DryRun {
		actionType = db.ActionTypeView
	}
	db.RecordAudit(db.AuditEntry{
		User:       username,
		Action:     "apply",
		ActionType: actionType,
		Resource:   "yaml",
		Details:    fmt.Sprintf("namespace=%s, dryRun=%v", req.Namespace, req.DryRun),
	})

	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  err.Error(),
			"dryRun": req.DryRun,
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": result,
		"dryRun":  req.DryRun,
	})
}
