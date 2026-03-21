package ui

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

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
