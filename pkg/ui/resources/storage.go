package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
)

func GetPersistentVolumesView(ctx context.Context, client *k8s.Client, filter string) (ResourceView, error) {
	headers := []string{"NAME", "CAPACITY", "ACCESS MODES", "RECLAIM POLICY", "STATUS", "CLAIM", "STORAGECLASS", "AGE"}
	pv, err := client.ListPersistentVolumes(ctx)
	if err != nil {
		return ResourceView{}, err
	}
	var rows [][]TableCell
	for _, p := range pv {
		if filter != "" && !strings.Contains(p.Name, filter) {
			continue
		}
		claim := ""
		if p.Spec.ClaimRef != nil {
			claim = fmt.Sprintf("%s/%s", p.Spec.ClaimRef.Namespace, p.Spec.ClaimRef.Name)
		}
		rows = append(rows, []TableCell{
			{Text: p.Name},
			{Text: p.Spec.Capacity.Storage().String()},
			{Text: fmt.Sprintf("%v", p.Spec.AccessModes)},
			{Text: string(p.Spec.PersistentVolumeReclaimPolicy)},
			{Text: string(p.Status.Phase)},
			{Text: claim},
			{Text: p.Spec.StorageClassName},
			{Text: FormatAge(time.Since(p.CreationTimestamp.Time))},
		})
	}
	return ResourceView{Headers: headers, Rows: rows}, nil
}

func GetPersistentVolumeClaimsView(ctx context.Context, client *k8s.Client, namespace, filter string) (ResourceView, error) {
	headers := []string{"NAMESPACE", "NAME", "STATUS", "VOLUME", "CAPACITY", "ACCESS MODES", "STORAGECLASS", "AGE"}
	pvc, err := client.ListPersistentVolumeClaims(ctx, namespace)
	if err != nil {
		return ResourceView{}, err
	}
	var rows [][]TableCell
	for _, p := range pvc {
		if filter != "" && !strings.Contains(p.Name, filter) {
			continue
		}
		storageClass := "<none>"
		if p.Spec.StorageClassName != nil {
			storageClass = *p.Spec.StorageClassName
		}
		rows = append(rows, []TableCell{
			{Text: p.Namespace},
			{Text: p.Name},
			{Text: string(p.Status.Phase)},
			{Text: p.Spec.VolumeName},
			{Text: p.Spec.Resources.Requests.Storage().String()},
			{Text: fmt.Sprintf("%v", p.Spec.AccessModes)},
			{Text: storageClass},
			{Text: FormatAge(time.Since(p.CreationTimestamp.Time))},
		})
	}
	return ResourceView{Headers: headers, Rows: rows}, nil
}

func GetStorageClassesView(ctx context.Context, client *k8s.Client, filter string) (ResourceView, error) {
	headers := []string{"NAME", "PROVISIONER", "RECLAIMPOLICY", "VOLUMEBINDINGMODE", "ALLOWVOLUMEEXPANSION", "AGE"}
	sc, err := client.ListStorageClasses(ctx)
	if err != nil {
		return ResourceView{}, err
	}
	var rows [][]TableCell
	for _, s := range sc {
		if filter != "" && !strings.Contains(s.Name, filter) {
			continue
		}
		reclaim := ""
		if s.ReclaimPolicy != nil {
			reclaim = string(*s.ReclaimPolicy)
		}
		binding := ""
		if s.VolumeBindingMode != nil {
			binding = string(*s.VolumeBindingMode)
		}
		allow := "false"
		if s.AllowVolumeExpansion != nil && *s.AllowVolumeExpansion {
			allow = "true"
		}
		rows = append(rows, []TableCell{
			{Text: s.Name},
			{Text: s.Provisioner},
			{Text: reclaim},
			{Text: binding},
			{Text: allow},
			{Text: FormatAge(time.Since(s.CreationTimestamp.Time))},
		})
	}
	return ResourceView{Headers: headers, Rows: rows}, nil
}
