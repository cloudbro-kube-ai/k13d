package ui

import (
	"context"
	"fmt"
	"strings"
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
	headers := []string{"NAME", "STATUS", "ROLE", "CPU", "MEM", "GPU", "AGE"}
	nodes, err := a.k8s.ListNodes(ctx)
	if err != nil {
		return headers, nil, err
	}

	usage := loadNodeUsageSnapshots(ctx, a.k8s, nodes)
	var rows [][]string
	for _, n := range nodes {
		snapshot := usage[n.Name]

		rows = append(rows, []string{
			n.Name,
			nodeStatusSummary(n),
			nodeRoleSummary(n),
			formatNodeCPUUsage(snapshot),
			formatNodeMemoryUsage(snapshot),
			formatNodeGPUUsage(snapshot),
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

	namespaceList := make([]string, 0, len(nss)+1)
	namespaceList = append(namespaceList, "")

	var rows [][]string
	for _, n := range nss {
		namespaceList = append(namespaceList, n.Name)
		rows = append(rows, []string{
			n.Name,
			string(n.Status.Phase),
			formatAge(n.CreationTimestamp.Time),
		})
	}

	a.mx.Lock()
	a.namespaces = namespaceList
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

func (a *App) fetchGenericResource(ctx context.Context, resource, ns string) ([]string, [][]string, error) {
	headers := []string{"NAMESPACE", "NAME", "AGE"}
	return headers, nil, fmt.Errorf("resource type '%s' not yet implemented", resource)
}
