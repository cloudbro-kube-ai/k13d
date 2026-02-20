package web

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// XRayTreeNode represents a node in the resource hierarchy JSON tree.
type XRayTreeNode struct {
	Kind     string          `json:"kind"`
	Name     string          `json:"name"`
	Status   string          `json:"status"`
	Children []*XRayTreeNode `json:"children,omitempty"`
}

// XRayResponse is the JSON envelope for /api/xray.
type XRayResponse struct {
	Type      string          `json:"type"`
	Namespace string          `json:"namespace"`
	Nodes     []*XRayTreeNode `json:"nodes"`
	Timestamp time.Time       `json:"timestamp"`
}

// handleXRay returns resource hierarchy as a JSON tree.
func (s *Server) handleXRay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		MethodNotAllowed(w, http.MethodGet)
		return
	}

	if s.k8sClient == nil {
		WriteError(w, NewAPIError(ErrCodeK8sError, "Kubernetes client not available"))
		return
	}

	resType := r.URL.Query().Get("type")
	if resType == "" {
		resType = "deploy"
	}
	ns := r.URL.Query().Get("namespace")

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp := XRayResponse{
		Type:      normalizeXRayType(resType),
		Namespace: ns,
		Timestamp: time.Now(),
	}

	switch resp.Type {
	case "deploy":
		resp.Nodes = s.xrayDeployments(ctx, ns)
	case "sts":
		resp.Nodes = s.xrayStatefulSets(ctx, ns)
	case "job":
		resp.Nodes = s.xrayJobs(ctx, ns)
	case "cj":
		resp.Nodes = s.xrayCronJobs(ctx, ns)
	case "ds":
		resp.Nodes = s.xrayDaemonSets(ctx, ns)
	default:
		resp.Nodes = s.xrayDeployments(ctx, ns)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func normalizeXRayType(t string) string {
	switch strings.ToLower(t) {
	case "deploy", "deployment", "deployments":
		return "deploy"
	case "sts", "statefulset", "statefulsets":
		return "sts"
	case "job", "jobs":
		return "job"
	case "cj", "cronjob", "cronjobs":
		return "cj"
	case "ds", "daemonset", "daemonsets":
		return "ds"
	default:
		return "deploy"
	}
}

// --- tree builders ---

func (s *Server) xrayDeployments(ctx context.Context, ns string) []*XRayTreeNode {
	k := s.k8sClient
	deps, err := k.ListDeployments(ctx, ns)
	if err != nil {
		return nil
	}
	rsList, _ := k.ListReplicaSets(ctx, ns)
	pods, _ := k.ListPods(ctx, ns)

	var nodes []*XRayTreeNode
	for _, dep := range deps {
		desired := int32(1)
		if dep.Spec.Replicas != nil {
			desired = *dep.Spec.Replicas
		}
		depNode := &XRayTreeNode{
			Kind:   "Deployment",
			Name:   dep.Name,
			Status: deployStatus(dep, desired),
		}

		for _, rs := range rsList {
			if !xrayOwnedBy(rs.OwnerReferences, dep.Name, "Deployment") {
				continue
			}
			rsDesired := int32(0)
			if rs.Spec.Replicas != nil {
				rsDesired = *rs.Spec.Replicas
			}
			if rsDesired == 0 && rs.Status.Replicas == 0 {
				continue
			}
			rsNode := &XRayTreeNode{
				Kind:   "ReplicaSet",
				Name:   rs.Name,
				Status: rsStatus(rs, rsDesired),
			}
			for _, pod := range pods {
				if xrayOwnedBy(pod.OwnerReferences, rs.Name, "ReplicaSet") {
					rsNode.Children = append(rsNode.Children, podNode(pod))
				}
			}
			depNode.Children = append(depNode.Children, rsNode)
		}
		nodes = append(nodes, depNode)
	}
	return nodes
}

func (s *Server) xrayStatefulSets(ctx context.Context, ns string) []*XRayTreeNode {
	k := s.k8sClient
	stses, err := k.ListStatefulSets(ctx, ns)
	if err != nil {
		return nil
	}
	pods, _ := k.ListPods(ctx, ns)

	var nodes []*XRayTreeNode
	for _, sts := range stses {
		desired := int32(1)
		if sts.Spec.Replicas != nil {
			desired = *sts.Spec.Replicas
		}
		stsNode := &XRayTreeNode{
			Kind:   "StatefulSet",
			Name:   sts.Name,
			Status: stsStatus(sts, desired),
		}
		for _, pod := range pods {
			if xrayOwnedBy(pod.OwnerReferences, sts.Name, "StatefulSet") {
				stsNode.Children = append(stsNode.Children, podNode(pod))
			}
		}
		nodes = append(nodes, stsNode)
	}
	return nodes
}

func (s *Server) xrayJobs(ctx context.Context, ns string) []*XRayTreeNode {
	k := s.k8sClient
	jobs, err := k.ListJobs(ctx, ns)
	if err != nil {
		return nil
	}
	pods, _ := k.ListPods(ctx, ns)

	var nodes []*XRayTreeNode
	for _, job := range jobs {
		jobNode := &XRayTreeNode{
			Kind:   "Job",
			Name:   job.Name,
			Status: jobXRayStatus(job),
		}
		for _, pod := range pods {
			if xrayOwnedBy(pod.OwnerReferences, job.Name, "Job") {
				jobNode.Children = append(jobNode.Children, podNode(pod))
			}
		}
		nodes = append(nodes, jobNode)
	}
	return nodes
}

func (s *Server) xrayCronJobs(ctx context.Context, ns string) []*XRayTreeNode {
	k := s.k8sClient
	cjs, err := k.ListCronJobs(ctx, ns)
	if err != nil {
		return nil
	}
	jobs, _ := k.ListJobs(ctx, ns)
	pods, _ := k.ListPods(ctx, ns)

	var nodes []*XRayTreeNode
	for _, cj := range cjs {
		cjNode := &XRayTreeNode{
			Kind:   "CronJob",
			Name:   cj.Name,
			Status: cj.Spec.Schedule,
		}
		for _, job := range jobs {
			if !xrayOwnedBy(job.OwnerReferences, cj.Name, "CronJob") {
				continue
			}
			jobNode := &XRayTreeNode{
				Kind:   "Job",
				Name:   job.Name,
				Status: jobXRayStatus(job),
			}
			for _, pod := range pods {
				if xrayOwnedBy(pod.OwnerReferences, job.Name, "Job") {
					jobNode.Children = append(jobNode.Children, podNode(pod))
				}
			}
			cjNode.Children = append(cjNode.Children, jobNode)
		}
		nodes = append(nodes, cjNode)
	}
	return nodes
}

func (s *Server) xrayDaemonSets(ctx context.Context, ns string) []*XRayTreeNode {
	k := s.k8sClient
	dss, err := k.ListDaemonSets(ctx, ns)
	if err != nil {
		return nil
	}
	pods, _ := k.ListPods(ctx, ns)

	var nodes []*XRayTreeNode
	for _, ds := range dss {
		dsNode := &XRayTreeNode{
			Kind:   "DaemonSet",
			Name:   ds.Name,
			Status: dsStatus(ds),
		}
		for _, pod := range pods {
			if xrayOwnedBy(pod.OwnerReferences, ds.Name, "DaemonSet") {
				dsNode.Children = append(dsNode.Children, podNode(pod))
			}
		}
		nodes = append(nodes, dsNode)
	}
	return nodes
}

// --- helpers ---

func xrayOwnedBy(refs []metav1.OwnerReference, name, kind string) bool {
	for _, ref := range refs {
		if ref.Name == name && ref.Kind == kind {
			return true
		}
	}
	return false
}

func podNode(pod corev1.Pod) *XRayTreeNode {
	status := string(pod.Status.Phase)
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			status = cs.State.Waiting.Reason
			break
		}
	}
	return &XRayTreeNode{
		Kind:   "Pod",
		Name:   pod.Name,
		Status: status,
	}
}

func deployStatus(dep appsv1.Deployment, desired int32) string {
	if dep.Status.ReadyReplicas >= desired && dep.Status.UnavailableReplicas == 0 {
		return "Ready"
	}
	return "Updating"
}

func rsStatus(rs appsv1.ReplicaSet, desired int32) string {
	if rs.Status.ReadyReplicas >= desired {
		return "Ready"
	}
	return "Updating"
}

func stsStatus(sts appsv1.StatefulSet, desired int32) string {
	if sts.Status.ReadyReplicas >= desired {
		return "Ready"
	}
	return "Updating"
}

func dsStatus(ds appsv1.DaemonSet) string {
	if ds.Status.NumberReady >= ds.Status.DesiredNumberScheduled {
		return "Ready"
	}
	return "NotReady"
}

func jobXRayStatus(job batchv1.Job) string {
	if job.Status.Succeeded > 0 && job.Status.Active == 0 {
		return "Complete"
	}
	if job.Status.Failed > 0 && job.Status.Active == 0 {
		return "Failed"
	}
	if job.Status.Active > 0 {
		return "Active"
	}
	return "Pending"
}
