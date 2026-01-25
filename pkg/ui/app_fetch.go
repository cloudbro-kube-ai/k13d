package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// fetchResources dispatches to the appropriate fetch function based on resource type
func (a *App) fetchResources(ctx context.Context) ([]string, [][]string, error) {
	if a.k8s == nil {
		return nil, nil, fmt.Errorf("K8s client not available")
	}

	a.mx.RLock()
	resource := a.currentResource
	ns := a.currentNamespace
	a.mx.RUnlock()

	switch resource {
	case "pods":
		return a.fetchPods(ctx, ns)
	case "deployments":
		return a.fetchDeployments(ctx, ns)
	case "services":
		return a.fetchServices(ctx, ns)
	case "nodes":
		return a.fetchNodes(ctx)
	case "namespaces":
		return a.fetchNamespaces(ctx)
	case "events":
		return a.fetchEvents(ctx, ns)
	case "configmaps":
		return a.fetchConfigMaps(ctx, ns)
	case "secrets":
		return a.fetchSecrets(ctx, ns)
	case "persistentvolumes":
		return a.fetchPersistentVolumes(ctx)
	case "persistentvolumeclaims":
		return a.fetchPersistentVolumeClaims(ctx, ns)
	case "storageclasses":
		return a.fetchStorageClasses(ctx)
	case "replicasets":
		return a.fetchReplicaSets(ctx, ns)
	case "daemonsets":
		return a.fetchDaemonSets(ctx, ns)
	case "statefulsets":
		return a.fetchStatefulSets(ctx, ns)
	case "jobs":
		return a.fetchJobs(ctx, ns)
	case "cronjobs":
		return a.fetchCronJobs(ctx, ns)
	case "replicationcontrollers":
		return a.fetchReplicationControllers(ctx, ns)
	case "ingresses":
		return a.fetchIngresses(ctx, ns)
	case "endpoints":
		return a.fetchEndpoints(ctx, ns)
	case "networkpolicies":
		return a.fetchNetworkPolicies(ctx, ns)
	case "serviceaccounts":
		return a.fetchServiceAccounts(ctx, ns)
	case "roles":
		return a.fetchRoles(ctx, ns)
	case "rolebindings":
		return a.fetchRoleBindings(ctx, ns)
	case "clusterroles":
		return a.fetchClusterRoles(ctx)
	case "clusterrolebindings":
		return a.fetchClusterRoleBindings(ctx)
	case "poddisruptionbudgets":
		return a.fetchPodDisruptionBudgets(ctx, ns)
	case "limitranges":
		return a.fetchLimitRanges(ctx, ns)
	case "resourcequotas":
		return a.fetchResourceQuotas(ctx, ns)
	case "horizontalpodautoscalers":
		return a.fetchHPAs(ctx, ns)
	case "customresourcedefinitions":
		return a.fetchCRDs(ctx)
	default:
		// Try generic fetch for unknown resources
		return a.fetchGenericResource(ctx, resource, ns)
	}
}

func (a *App) fetchPods(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "STATUS", "READY", "RESTARTS", "AGE"}
	pods, err := a.k8s.ListPods(ctx, ns)
	if err != nil {
		return headers, nil, err
	}

	var rows [][]string
	for _, p := range pods {
		ready := 0
		total := len(p.Status.ContainerStatuses)
		var restarts int32
		for _, cs := range p.Status.ContainerStatuses {
			if cs.Ready {
				ready++
			}
			restarts += cs.RestartCount
		}

		status := string(p.Status.Phase)
		for _, cs := range p.Status.ContainerStatuses {
			if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
				status = cs.State.Waiting.Reason
				break
			}
		}

		rows = append(rows, []string{
			p.Namespace,
			p.Name,
			status,
			fmt.Sprintf("%d/%d", ready, total),
			fmt.Sprintf("%d", restarts),
			formatAge(p.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchDeployments(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "STATUS", "READY", "UP-TO-DATE", "AGE"}
	deps, err := a.k8s.ListDeployments(ctx, ns)
	if err != nil {
		return headers, nil, err
	}

	var rows [][]string
	for _, d := range deps {
		replicas := int32(1)
		if d.Spec.Replicas != nil {
			replicas = *d.Spec.Replicas
		}
		status := "Ready"
		if d.Status.ReadyReplicas < replicas {
			status = "Updating"
		}
		if d.Status.ReadyReplicas == 0 && replicas > 0 {
			status = "NotReady"
		}

		rows = append(rows, []string{
			d.Namespace,
			d.Name,
			status,
			fmt.Sprintf("%d/%d", d.Status.ReadyReplicas, replicas),
			fmt.Sprintf("%d", d.Status.UpdatedReplicas),
			formatAge(d.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchServices(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "TYPE", "CLUSTER-IP", "PORTS", "AGE"}
	svcs, err := a.k8s.ListServices(ctx, ns)
	if err != nil {
		return headers, nil, err
	}

	var rows [][]string
	for _, s := range svcs {
		var ports []string
		for _, p := range s.Spec.Ports {
			ports = append(ports, fmt.Sprintf("%d/%s", p.Port, p.Protocol))
		}

		rows = append(rows, []string{
			s.Namespace,
			s.Name,
			string(s.Spec.Type),
			s.Spec.ClusterIP,
			strings.Join(ports, ","),
			formatAge(s.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchNodes(ctx context.Context) ([]string, [][]string, error) {
	headers := []string{"NAME", "STATUS", "ROLES", "VERSION", "AGE"}
	nodes, err := a.k8s.ListNodes(ctx)
	if err != nil {
		return headers, nil, err
	}

	var rows [][]string
	for _, n := range nodes {
		status := "NotReady"
		for _, c := range n.Status.Conditions {
			if c.Type == corev1.NodeReady && c.Status == corev1.ConditionTrue {
				status = "Ready"
			}
		}

		roles := []string{}
		for label := range n.Labels {
			if strings.HasPrefix(label, "node-role.kubernetes.io/") {
				role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
				roles = append(roles, role)
			}
		}
		if len(roles) == 0 {
			roles = []string{"<none>"}
		}

		rows = append(rows, []string{
			n.Name,
			status,
			strings.Join(roles, ","),
			n.Status.NodeInfo.KubeletVersion,
			formatAge(n.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchNamespaces(ctx context.Context) ([]string, [][]string, error) {
	headers := []string{"NAME", "STATUS", "AGE"}
	nss, err := a.k8s.ListNamespaces(ctx)
	if err != nil {
		return headers, nil, err
	}

	// Build namespace list and rows without holding lock
	namespaceList := make([]string, 0, len(nss)+1)
	namespaceList = append(namespaceList, "") // Empty string for "all namespaces"

	var rows [][]string
	for _, n := range nss {
		namespaceList = append(namespaceList, n.Name)
		rows = append(rows, []string{
			n.Name,
			string(n.Status.Phase),
			formatAge(n.CreationTimestamp.Time),
		})
	}

	// Cache namespaces for cycling (single lock acquisition)
	a.mx.Lock()
	a.namespaces = namespaceList
	a.mx.Unlock()

	// Reorder by recent usage
	reordered := a.reorderNamespacesByRecent()
	a.mx.Lock()
	a.namespaces = reordered
	a.mx.Unlock()

	return headers, rows, nil
}

func (a *App) fetchEvents(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "TYPE", "REASON", "OBJECT", "MESSAGE"}
	events, err := a.k8s.ListEvents(ctx, ns)
	if err != nil {
		return headers, nil, err
	}

	var rows [][]string
	for _, e := range events {
		msg := e.Message
		if len(msg) > 60 {
			msg = msg[:57] + "..."
		}
		rows = append(rows, []string{
			e.Namespace,
			e.Type,
			e.Reason,
			e.InvolvedObject.Name,
			msg,
		})
	}
	return headers, rows, nil
}

func (a *App) fetchConfigMaps(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "DATA", "AGE"}
	cms, err := a.k8s.ListConfigMaps(ctx, ns)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, cm := range cms {
		rows = append(rows, []string{
			cm.Namespace,
			cm.Name,
			fmt.Sprintf("%d", len(cm.Data)),
			formatAge(cm.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchSecrets(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "TYPE", "DATA", "AGE"}
	secrets, err := a.k8s.ListSecrets(ctx, ns)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, s := range secrets {
		rows = append(rows, []string{
			s.Namespace,
			s.Name,
			string(s.Type),
			fmt.Sprintf("%d", len(s.Data)),
			formatAge(s.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchPersistentVolumes(ctx context.Context) ([]string, [][]string, error) {
	headers := []string{"NAME", "CAPACITY", "ACCESS MODES", "STATUS", "CLAIM", "AGE"}
	pvs, err := a.k8s.ListPersistentVolumes(ctx)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, pv := range pvs {
		capacity := ""
		if storage, ok := pv.Spec.Capacity[corev1.ResourceStorage]; ok {
			capacity = storage.String()
		}
		claim := ""
		if pv.Spec.ClaimRef != nil {
			claim = fmt.Sprintf("%s/%s", pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name)
		}
		rows = append(rows, []string{
			pv.Name,
			capacity,
			strings.Join(accessModesToStrings(pv.Spec.AccessModes), ","),
			string(pv.Status.Phase),
			claim,
			formatAge(pv.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchPersistentVolumeClaims(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "STATUS", "VOLUME", "CAPACITY", "AGE"}
	pvcs, err := a.k8s.ListPersistentVolumeClaims(ctx, ns)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, pvc := range pvcs {
		capacity := ""
		if pvc.Status.Capacity != nil {
			if storage, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
				capacity = storage.String()
			}
		}
		rows = append(rows, []string{
			pvc.Namespace,
			pvc.Name,
			string(pvc.Status.Phase),
			pvc.Spec.VolumeName,
			capacity,
			formatAge(pvc.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchStorageClasses(ctx context.Context) ([]string, [][]string, error) {
	headers := []string{"NAME", "PROVISIONER", "RECLAIM POLICY", "ALLOW EXPANSION", "AGE"}
	scs, err := a.k8s.ListStorageClasses(ctx)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, sc := range scs {
		reclaim := "<default>"
		if sc.ReclaimPolicy != nil {
			reclaim = string(*sc.ReclaimPolicy)
		}
		expand := "false"
		if sc.AllowVolumeExpansion != nil && *sc.AllowVolumeExpansion {
			expand = "true"
		}
		rows = append(rows, []string{
			sc.Name,
			sc.Provisioner,
			reclaim,
			expand,
			formatAge(sc.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchReplicaSets(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "DESIRED", "CURRENT", "READY", "AGE"}
	rss, err := a.k8s.ListReplicaSets(ctx, ns)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, rs := range rss {
		desired := int32(0)
		if rs.Spec.Replicas != nil {
			desired = *rs.Spec.Replicas
		}
		rows = append(rows, []string{
			rs.Namespace,
			rs.Name,
			fmt.Sprintf("%d", desired),
			fmt.Sprintf("%d", rs.Status.Replicas),
			fmt.Sprintf("%d", rs.Status.ReadyReplicas),
			formatAge(rs.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchDaemonSets(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "DESIRED", "CURRENT", "READY", "AGE"}
	dss, err := a.k8s.ListDaemonSets(ctx, ns)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, ds := range dss {
		rows = append(rows, []string{
			ds.Namespace,
			ds.Name,
			fmt.Sprintf("%d", ds.Status.DesiredNumberScheduled),
			fmt.Sprintf("%d", ds.Status.CurrentNumberScheduled),
			fmt.Sprintf("%d", ds.Status.NumberReady),
			formatAge(ds.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchStatefulSets(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "READY", "AGE"}
	stss, err := a.k8s.ListStatefulSets(ctx, ns)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, sts := range stss {
		replicas := int32(0)
		if sts.Spec.Replicas != nil {
			replicas = *sts.Spec.Replicas
		}
		rows = append(rows, []string{
			sts.Namespace,
			sts.Name,
			fmt.Sprintf("%d/%d", sts.Status.ReadyReplicas, replicas),
			formatAge(sts.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchJobs(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "COMPLETIONS", "DURATION", "AGE"}
	jobs, err := a.k8s.ListJobs(ctx, ns)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, job := range jobs {
		completions := int32(1)
		if job.Spec.Completions != nil {
			completions = *job.Spec.Completions
		}
		duration := "<running>"
		if job.Status.CompletionTime != nil && job.Status.StartTime != nil {
			d := job.Status.CompletionTime.Sub(job.Status.StartTime.Time)
			duration = d.Round(time.Second).String()
		}
		rows = append(rows, []string{
			job.Namespace,
			job.Name,
			fmt.Sprintf("%d/%d", job.Status.Succeeded, completions),
			duration,
			formatAge(job.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchCronJobs(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "SCHEDULE", "SUSPEND", "ACTIVE", "LAST SCHEDULE", "AGE"}
	cjs, err := a.k8s.ListCronJobs(ctx, ns)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, cj := range cjs {
		suspend := "False"
		if cj.Spec.Suspend != nil && *cj.Spec.Suspend {
			suspend = "True"
		}
		lastSchedule := "<none>"
		if cj.Status.LastScheduleTime != nil {
			lastSchedule = formatAge(cj.Status.LastScheduleTime.Time)
		}
		rows = append(rows, []string{
			cj.Namespace,
			cj.Name,
			cj.Spec.Schedule,
			suspend,
			fmt.Sprintf("%d", len(cj.Status.Active)),
			lastSchedule,
			formatAge(cj.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchReplicationControllers(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "DESIRED", "CURRENT", "READY", "AGE"}
	rcs, err := a.k8s.ListReplicationControllers(ctx, ns)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, rc := range rcs {
		desired := int32(0)
		if rc.Spec.Replicas != nil {
			desired = *rc.Spec.Replicas
		}
		rows = append(rows, []string{
			rc.Namespace,
			rc.Name,
			fmt.Sprintf("%d", desired),
			fmt.Sprintf("%d", rc.Status.Replicas),
			fmt.Sprintf("%d", rc.Status.ReadyReplicas),
			formatAge(rc.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchIngresses(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "CLASS", "HOSTS", "ADDRESS", "AGE"}
	ings, err := a.k8s.ListIngresses(ctx, ns)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, ing := range ings {
		class := "<none>"
		if ing.Spec.IngressClassName != nil {
			class = *ing.Spec.IngressClassName
		}
		var hosts []string
		for _, rule := range ing.Spec.Rules {
			if rule.Host != "" {
				hosts = append(hosts, rule.Host)
			}
		}
		var addresses []string
		for _, lb := range ing.Status.LoadBalancer.Ingress {
			if lb.IP != "" {
				addresses = append(addresses, lb.IP)
			} else if lb.Hostname != "" {
				addresses = append(addresses, lb.Hostname)
			}
		}
		rows = append(rows, []string{
			ing.Namespace,
			ing.Name,
			class,
			strings.Join(hosts, ","),
			strings.Join(addresses, ","),
			formatAge(ing.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchEndpoints(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "ENDPOINTS", "AGE"}
	eps, err := a.k8s.ListEndpoints(ctx, ns)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, ep := range eps {
		var addrs []string
		for _, subset := range ep.Subsets {
			for _, addr := range subset.Addresses {
				for _, port := range subset.Ports {
					addrs = append(addrs, fmt.Sprintf("%s:%d", addr.IP, port.Port))
				}
			}
		}
		epStr := strings.Join(addrs, ",")
		if len(epStr) > 50 {
			epStr = epStr[:47] + "..."
		}
		rows = append(rows, []string{
			ep.Namespace,
			ep.Name,
			epStr,
			formatAge(ep.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchNetworkPolicies(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "POD-SELECTOR", "AGE"}
	netpols, err := a.k8s.ListNetworkPolicies(ctx, ns)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, np := range netpols {
		selector := "<all>"
		if len(np.Spec.PodSelector.MatchLabels) > 0 {
			var parts []string
			for k, v := range np.Spec.PodSelector.MatchLabels {
				parts = append(parts, fmt.Sprintf("%s=%s", k, v))
			}
			selector = strings.Join(parts, ",")
		}
		rows = append(rows, []string{
			np.Namespace,
			np.Name,
			selector,
			formatAge(np.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchServiceAccounts(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "SECRETS", "AGE"}
	sas, err := a.k8s.ListServiceAccounts(ctx, ns)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, sa := range sas {
		rows = append(rows, []string{
			sa.Namespace,
			sa.Name,
			fmt.Sprintf("%d", len(sa.Secrets)),
			formatAge(sa.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchRoles(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "AGE"}
	roles, err := a.k8s.ListRoles(ctx, ns)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, r := range roles {
		rows = append(rows, []string{
			r.Namespace,
			r.Name,
			formatAge(r.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchRoleBindings(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "ROLE", "AGE"}
	rbs, err := a.k8s.ListRoleBindings(ctx, ns)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, rb := range rbs {
		roleRef := fmt.Sprintf("%s/%s", rb.RoleRef.Kind, rb.RoleRef.Name)
		rows = append(rows, []string{
			rb.Namespace,
			rb.Name,
			roleRef,
			formatAge(rb.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchClusterRoles(ctx context.Context) ([]string, [][]string, error) {
	headers := []string{"NAME", "AGE"}
	crs, err := a.k8s.ListClusterRoles(ctx)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, cr := range crs {
		rows = append(rows, []string{
			cr.Name,
			formatAge(cr.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchClusterRoleBindings(ctx context.Context) ([]string, [][]string, error) {
	headers := []string{"NAME", "ROLE", "AGE"}
	crbs, err := a.k8s.ListClusterRoleBindings(ctx)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, crb := range crbs {
		roleRef := fmt.Sprintf("%s/%s", crb.RoleRef.Kind, crb.RoleRef.Name)
		rows = append(rows, []string{
			crb.Name,
			roleRef,
			formatAge(crb.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchPodDisruptionBudgets(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "MIN AVAILABLE", "MAX UNAVAILABLE", "ALLOWED DISRUPTIONS", "AGE"}
	pdbs, err := a.k8s.ListPodDisruptionBudgets(ctx, ns)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, pdb := range pdbs {
		minAvail := "<none>"
		if pdb.Spec.MinAvailable != nil {
			minAvail = pdb.Spec.MinAvailable.String()
		}
		maxUnavail := "<none>"
		if pdb.Spec.MaxUnavailable != nil {
			maxUnavail = pdb.Spec.MaxUnavailable.String()
		}
		rows = append(rows, []string{
			pdb.Namespace,
			pdb.Name,
			minAvail,
			maxUnavail,
			fmt.Sprintf("%d", pdb.Status.DisruptionsAllowed),
			formatAge(pdb.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchLimitRanges(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "AGE"}
	lrs, err := a.k8s.ListLimitRanges(ctx, ns)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, lr := range lrs {
		rows = append(rows, []string{
			lr.Namespace,
			lr.Name,
			formatAge(lr.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchResourceQuotas(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "AGE"}
	rqs, err := a.k8s.ListResourceQuotas(ctx, ns)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, rq := range rqs {
		rows = append(rows, []string{
			rq.Namespace,
			rq.Name,
			formatAge(rq.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchHPAs(ctx context.Context, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "REFERENCE", "TARGETS", "MINPODS", "MAXPODS", "REPLICAS", "AGE"}
	hpas, err := a.k8s.ListHPAs(ctx, ns)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, hpa := range hpas {
		ref := fmt.Sprintf("%s/%s", hpa.Spec.ScaleTargetRef.Kind, hpa.Spec.ScaleTargetRef.Name)
		minPods := int32(1)
		if hpa.Spec.MinReplicas != nil {
			minPods = *hpa.Spec.MinReplicas
		}
		rows = append(rows, []string{
			hpa.Namespace,
			hpa.Name,
			ref,
			"<complex>",
			fmt.Sprintf("%d", minPods),
			fmt.Sprintf("%d", hpa.Spec.MaxReplicas),
			fmt.Sprintf("%d", hpa.Status.CurrentReplicas),
			formatAge(hpa.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchCRDs(ctx context.Context) ([]string, [][]string, error) {
	headers := []string{"NAME", "CREATED AT"}
	crds, err := a.k8s.ListCRDs(ctx)
	if err != nil {
		return headers, nil, err
	}
	var rows [][]string
	for _, crd := range crds {
		rows = append(rows, []string{
			crd.Name,
			formatAge(crd.CreationTimestamp.Time),
		})
	}
	return headers, rows, nil
}

func (a *App) fetchGenericResource(ctx context.Context, resource, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "AGE"}
	return headers, nil, fmt.Errorf("resource type '%s' not yet implemented", resource)
}

// Helper function for PV access modes
func accessModesToStrings(modes []corev1.PersistentVolumeAccessMode) []string {
	var result []string
	for _, m := range modes {
		switch m {
		case corev1.ReadWriteOnce:
			result = append(result, "RWO")
		case corev1.ReadOnlyMany:
			result = append(result, "ROX")
		case corev1.ReadWriteMany:
			result = append(result, "RWX")
		case corev1.ReadWriteOncePod:
			result = append(result, "RWOP")
		}
	}
	return result
}
