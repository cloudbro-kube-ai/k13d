package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// TopologyNode represents a Kubernetes resource as a graph node.
type TopologyNode struct {
	ID        string            `json:"id"`
	Kind      string            `json:"kind"`
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Status    string            `json:"status"`
	Info      map[string]string `json:"info,omitempty"`
}

// TopologyEdge represents a relationship between two resources.
type TopologyEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"` // "owns", "selects", "mounts", "routes", "scales"
}

// TopologyResponse is the API response for the topology endpoint.
type TopologyResponse struct {
	Nodes []TopologyNode `json:"nodes"`
	Edges []TopologyEdge `json:"edges"`
}

func nodeID(kind, namespace, name string) string {
	return fmt.Sprintf("%s/%s/%s", kind, namespace, name)
}

func (s *Server) handleTopology(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	namespace := r.URL.Query().Get("namespace")
	ctx := r.Context()

	resp, err := s.buildTopology(ctx, namespace)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) buildTopology(ctx context.Context, namespace string) (*TopologyResponse, error) {
	var (
		mu    sync.Mutex
		wg    sync.WaitGroup
		nodes []TopologyNode
		edges []TopologyEdge
	)

	// nodeSet tracks added nodes to avoid duplicates
	nodeSet := make(map[string]bool)

	addNode := func(n TopologyNode) {
		mu.Lock()
		defer mu.Unlock()
		if !nodeSet[n.ID] {
			nodes = append(nodes, n)
			nodeSet[n.ID] = true
		}
	}
	addEdge := func(e TopologyEdge) {
		mu.Lock()
		defer mu.Unlock()
		edges = append(edges, e)
	}

	// Fetch all resources in parallel
	var (
		pods         []corev1.Pod
		deployments  []appsv1.Deployment
		replicaSets  []appsv1.ReplicaSet
		statefulSets []appsv1.StatefulSet
		daemonSets   []appsv1.DaemonSet
		services     []corev1.Service
		ingresses    []networkingv1.Ingress
		jobs         []batchv1.Job
		cronJobs     []batchv1.CronJob
		configMaps   []corev1.ConfigMap
		secrets      []corev1.Secret
		pvcs         []corev1.PersistentVolumeClaim
		hpas         []autoscalingv2.HorizontalPodAutoscaler
	)

	type fetchResult struct {
		name string
		err  error
	}
	errCh := make(chan fetchResult, 13)

	fetch := func(name string, fn func() error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(); err != nil {
				errCh <- fetchResult{name, err}
			}
		}()
	}

	fetch("pods", func() error {
		var err error
		pods, err = s.k8sClient.ListPods(ctx, namespace)
		return err
	})
	fetch("deployments", func() error {
		var err error
		deployments, err = s.k8sClient.ListDeployments(ctx, namespace)
		return err
	})
	fetch("replicasets", func() error {
		var err error
		replicaSets, err = s.k8sClient.ListReplicaSets(ctx, namespace)
		return err
	})
	fetch("statefulsets", func() error {
		var err error
		statefulSets, err = s.k8sClient.ListStatefulSets(ctx, namespace)
		return err
	})
	fetch("daemonsets", func() error {
		var err error
		daemonSets, err = s.k8sClient.ListDaemonSets(ctx, namespace)
		return err
	})
	fetch("services", func() error {
		var err error
		services, err = s.k8sClient.ListServices(ctx, namespace)
		return err
	})
	fetch("ingresses", func() error {
		var err error
		ingresses, err = s.k8sClient.ListIngresses(ctx, namespace)
		return err
	})
	fetch("jobs", func() error {
		var err error
		jobs, err = s.k8sClient.ListJobs(ctx, namespace)
		return err
	})
	fetch("cronjobs", func() error {
		var err error
		cronJobs, err = s.k8sClient.ListCronJobs(ctx, namespace)
		return err
	})
	fetch("configmaps", func() error {
		var err error
		configMaps, err = s.k8sClient.ListConfigMaps(ctx, namespace)
		return err
	})
	fetch("secrets", func() error {
		var err error
		secrets, err = s.k8sClient.ListSecrets(ctx, namespace)
		return err
	})
	fetch("pvcs", func() error {
		var err error
		pvcs, err = s.k8sClient.ListPersistentVolumeClaims(ctx, namespace)
		return err
	})
	fetch("hpas", func() error {
		var err error
		hpas, err = s.k8sClient.ListHorizontalPodAutoscalers(ctx, namespace)
		return err
	})

	wg.Wait()
	close(errCh)

	// Log fetch errors but continue with partial data
	for res := range errCh {
		_ = res // non-critical: partial topology is still useful
	}

	// Build node index for owner reference lookups
	ownerIndex := make(map[string]string) // UID -> nodeID

	// --- Add Deployment nodes ---
	for _, d := range deployments {
		id := nodeID("Deployment", d.Namespace, d.Name)
		replicas := int32(1)
		if d.Spec.Replicas != nil {
			replicas = *d.Spec.Replicas
		}
		status := "running"
		if d.Status.ReadyReplicas < replicas {
			status = "pending"
		}
		if d.Status.ReadyReplicas == 0 && replicas > 0 {
			status = "failed"
		}
		addNode(TopologyNode{
			ID: id, Kind: "Deployment", Name: d.Name, Namespace: d.Namespace,
			Status: status,
			Info: map[string]string{
				"replicas": fmt.Sprintf("%d/%d", d.Status.ReadyReplicas, replicas),
			},
		})
		ownerIndex[string(d.UID)] = id
	}

	// --- Add ReplicaSet nodes ---
	for _, rs := range replicaSets {
		// Skip ReplicaSets with 0 replicas (old revisions)
		if rs.Status.Replicas == 0 {
			continue
		}
		id := nodeID("ReplicaSet", rs.Namespace, rs.Name)
		status := "running"
		if rs.Status.ReadyReplicas < rs.Status.Replicas {
			status = "pending"
		}
		addNode(TopologyNode{
			ID: id, Kind: "ReplicaSet", Name: rs.Name, Namespace: rs.Namespace,
			Status: status,
			Info: map[string]string{
				"replicas": fmt.Sprintf("%d/%d", rs.Status.ReadyReplicas, rs.Status.Replicas),
			},
		})
		ownerIndex[string(rs.UID)] = id
		// ownerReferences → Deployment
		for _, ref := range rs.OwnerReferences {
			if parentID, ok := ownerIndex[string(ref.UID)]; ok {
				addEdge(TopologyEdge{Source: parentID, Target: id, Type: "owns"})
			}
		}
	}

	// --- Add StatefulSet nodes ---
	for _, ss := range statefulSets {
		id := nodeID("StatefulSet", ss.Namespace, ss.Name)
		replicas := int32(1)
		if ss.Spec.Replicas != nil {
			replicas = *ss.Spec.Replicas
		}
		status := "running"
		if ss.Status.ReadyReplicas < replicas {
			status = "pending"
		}
		addNode(TopologyNode{
			ID: id, Kind: "StatefulSet", Name: ss.Name, Namespace: ss.Namespace,
			Status: status,
			Info: map[string]string{
				"replicas": fmt.Sprintf("%d/%d", ss.Status.ReadyReplicas, replicas),
			},
		})
		ownerIndex[string(ss.UID)] = id
	}

	// --- Add DaemonSet nodes ---
	for _, ds := range daemonSets {
		id := nodeID("DaemonSet", ds.Namespace, ds.Name)
		status := "running"
		if ds.Status.NumberReady < ds.Status.DesiredNumberScheduled {
			status = "pending"
		}
		addNode(TopologyNode{
			ID: id, Kind: "DaemonSet", Name: ds.Name, Namespace: ds.Namespace,
			Status: status,
			Info: map[string]string{
				"ready": fmt.Sprintf("%d/%d", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled),
			},
		})
		ownerIndex[string(ds.UID)] = id
	}

	// --- Add CronJob nodes ---
	for _, cj := range cronJobs {
		id := nodeID("CronJob", cj.Namespace, cj.Name)
		status := "running"
		addNode(TopologyNode{
			ID: id, Kind: "CronJob", Name: cj.Name, Namespace: cj.Namespace,
			Status: status,
			Info: map[string]string{
				"schedule": cj.Spec.Schedule,
				"active":   fmt.Sprintf("%d", len(cj.Status.Active)),
			},
		})
		ownerIndex[string(cj.UID)] = id
	}

	// --- Add Job nodes ---
	for _, j := range jobs {
		id := nodeID("Job", j.Namespace, j.Name)
		status := "running"
		if j.Status.Succeeded > 0 {
			status = "succeeded"
		}
		if j.Status.Failed > 0 && j.Status.Active == 0 {
			status = "failed"
		}
		addNode(TopologyNode{
			ID: id, Kind: "Job", Name: j.Name, Namespace: j.Namespace,
			Status: status,
			Info: map[string]string{
				"completions": fmt.Sprintf("%d/%d", j.Status.Succeeded, func() int32 {
					if j.Spec.Completions != nil {
						return *j.Spec.Completions
					}
					return 1
				}()),
			},
		})
		ownerIndex[string(j.UID)] = id
		// ownerReferences → CronJob
		for _, ref := range j.OwnerReferences {
			if parentID, ok := ownerIndex[string(ref.UID)]; ok {
				addEdge(TopologyEdge{Source: parentID, Target: id, Type: "owns"})
			}
		}
	}

	// Build ConfigMap/Secret/PVC index for mount lookups
	cmIndex := make(map[string]string)  // "namespace/name" -> nodeID
	secIndex := make(map[string]string) // "namespace/name" -> nodeID
	pvcIndex := make(map[string]string) // "namespace/name" -> nodeID

	for _, cm := range configMaps {
		id := nodeID("ConfigMap", cm.Namespace, cm.Name)
		addNode(TopologyNode{
			ID: id, Kind: "ConfigMap", Name: cm.Name, Namespace: cm.Namespace,
			Status: "running",
			Info:   map[string]string{"keys": fmt.Sprintf("%d", len(cm.Data))},
		})
		cmIndex[cm.Namespace+"/"+cm.Name] = id
	}
	for _, sec := range secrets {
		id := nodeID("Secret", sec.Namespace, sec.Name)
		addNode(TopologyNode{
			ID: id, Kind: "Secret", Name: sec.Name, Namespace: sec.Namespace,
			Status: "running",
			Info:   map[string]string{"type": string(sec.Type)},
		})
		secIndex[sec.Namespace+"/"+sec.Name] = id
	}
	for _, pvc := range pvcs {
		id := nodeID("PVC", pvc.Namespace, pvc.Name)
		status := "pending"
		if pvc.Status.Phase == corev1.ClaimBound {
			status = "running"
		}
		addNode(TopologyNode{
			ID: id, Kind: "PVC", Name: pvc.Name, Namespace: pvc.Namespace,
			Status: status,
			Info: map[string]string{
				"phase":   string(pvc.Status.Phase),
				"storage": pvc.Spec.Resources.Requests.Storage().String(),
			},
		})
		pvcIndex[pvc.Namespace+"/"+pvc.Name] = id
	}

	// --- Add Pod nodes + owner edges + mount edges ---
	for _, p := range pods {
		id := nodeID("Pod", p.Namespace, p.Name)
		status := podStatus(p)
		info := map[string]string{
			"ip":     p.Status.PodIP,
			"node":   p.Spec.NodeName,
			"status": string(p.Status.Phase),
		}
		// Count restarts
		var restarts int32
		for _, cs := range p.Status.ContainerStatuses {
			restarts += cs.RestartCount
		}
		info["restarts"] = fmt.Sprintf("%d", restarts)

		addNode(TopologyNode{
			ID: id, Kind: "Pod", Name: p.Name, Namespace: p.Namespace,
			Status: status, Info: info,
		})

		// ownerReferences edges
		for _, ref := range p.OwnerReferences {
			if parentID, ok := ownerIndex[string(ref.UID)]; ok {
				addEdge(TopologyEdge{Source: parentID, Target: id, Type: "owns"})
			}
		}

		// Volume mount edges (ConfigMap, Secret, PVC)
		for _, vol := range p.Spec.Volumes {
			if vol.ConfigMap != nil {
				if cmID, ok := cmIndex[p.Namespace+"/"+vol.ConfigMap.Name]; ok {
					addEdge(TopologyEdge{Source: id, Target: cmID, Type: "mounts"})
				}
			}
			if vol.Secret != nil {
				if secID, ok := secIndex[p.Namespace+"/"+vol.Secret.SecretName]; ok {
					addEdge(TopologyEdge{Source: id, Target: secID, Type: "mounts"})
				}
			}
			if vol.PersistentVolumeClaim != nil {
				if pvcID, ok := pvcIndex[p.Namespace+"/"+vol.PersistentVolumeClaim.ClaimName]; ok {
					addEdge(TopologyEdge{Source: id, Target: pvcID, Type: "mounts"})
				}
			}
		}

		// envFrom references
		for _, c := range p.Spec.Containers {
			for _, envFrom := range c.EnvFrom {
				if envFrom.ConfigMapRef != nil {
					if cmID, ok := cmIndex[p.Namespace+"/"+envFrom.ConfigMapRef.Name]; ok {
						addEdge(TopologyEdge{Source: id, Target: cmID, Type: "mounts"})
					}
				}
				if envFrom.SecretRef != nil {
					if secID, ok := secIndex[p.Namespace+"/"+envFrom.SecretRef.Name]; ok {
						addEdge(TopologyEdge{Source: id, Target: secID, Type: "mounts"})
					}
				}
			}
		}
	}

	// --- Add Service nodes + selector edges ---
	for _, svc := range services {
		id := nodeID("Service", svc.Namespace, svc.Name)
		ports := make([]string, 0, len(svc.Spec.Ports))
		for _, p := range svc.Spec.Ports {
			ports = append(ports, fmt.Sprintf("%d/%s", p.Port, p.Protocol))
		}
		addNode(TopologyNode{
			ID: id, Kind: "Service", Name: svc.Name, Namespace: svc.Namespace,
			Status: "running",
			Info: map[string]string{
				"type":      string(svc.Spec.Type),
				"clusterIP": svc.Spec.ClusterIP,
				"ports":     strings.Join(ports, ", "),
			},
		})

		// Service → Pods via label selector
		if len(svc.Spec.Selector) > 0 {
			for _, p := range pods {
				if p.Namespace != svc.Namespace {
					continue
				}
				if labelsMatch(svc.Spec.Selector, p.Labels) {
					podID := nodeID("Pod", p.Namespace, p.Name)
					addEdge(TopologyEdge{Source: id, Target: podID, Type: "selects"})
				}
			}
		}
	}

	// --- Add Ingress nodes + routing edges ---
	for _, ing := range ingresses {
		id := nodeID("Ingress", ing.Namespace, ing.Name)
		var hosts []string
		for _, rule := range ing.Spec.Rules {
			if rule.Host != "" {
				hosts = append(hosts, rule.Host)
			}
		}
		addNode(TopologyNode{
			ID: id, Kind: "Ingress", Name: ing.Name, Namespace: ing.Namespace,
			Status: "running",
			Info:   map[string]string{"hosts": strings.Join(hosts, ", ")},
		})

		// Ingress → Services
		for _, rule := range ing.Spec.Rules {
			if rule.HTTP == nil {
				continue
			}
			for _, path := range rule.HTTP.Paths {
				if path.Backend.Service != nil {
					svcID := nodeID("Service", ing.Namespace, path.Backend.Service.Name)
					if nodeSet[svcID] {
						addEdge(TopologyEdge{Source: id, Target: svcID, Type: "routes"})
					}
				}
			}
		}
	}

	// --- Add HPA edges ---
	for _, hpa := range hpas {
		id := nodeID("HPA", hpa.Namespace, hpa.Name)
		addNode(TopologyNode{
			ID: id, Kind: "HPA", Name: hpa.Name, Namespace: hpa.Namespace,
			Status: "running",
			Info: map[string]string{
				"minReplicas": fmt.Sprintf("%d", func() int32 {
					if hpa.Spec.MinReplicas != nil {
						return *hpa.Spec.MinReplicas
					}
					return 1
				}()),
				"maxReplicas":     fmt.Sprintf("%d", hpa.Spec.MaxReplicas),
				"currentReplicas": fmt.Sprintf("%d", hpa.Status.CurrentReplicas),
			},
		})

		// HPA → target resource
		targetKind := hpa.Spec.ScaleTargetRef.Kind
		targetName := hpa.Spec.ScaleTargetRef.Name
		targetID := nodeID(targetKind, hpa.Namespace, targetName)
		if nodeSet[targetID] {
			addEdge(TopologyEdge{Source: id, Target: targetID, Type: "scales"})
		}
	}

	return &TopologyResponse{Nodes: nodes, Edges: edges}, nil
}

// labelsMatch returns true if all selector labels are present in the resource labels.
func labelsMatch(selector, labels map[string]string) bool {
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
}

// podStatus derives a topology status string from pod phase and container statuses.
func podStatus(p corev1.Pod) string {
	switch p.Status.Phase {
	case corev1.PodRunning:
		// Check for CrashLoopBackOff
		for _, cs := range p.Status.ContainerStatuses {
			if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
				return "failed"
			}
			if !cs.Ready {
				return "pending"
			}
		}
		return "running"
	case corev1.PodPending:
		return "pending"
	case corev1.PodSucceeded:
		return "succeeded"
	case corev1.PodFailed:
		return "failed"
	default:
		return "unknown"
	}
}
