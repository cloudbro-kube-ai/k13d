package ui

import (
	"context"
	"fmt"
	"time"
)

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
