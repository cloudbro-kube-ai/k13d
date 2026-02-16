package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
	"github.com/cloudbro-kube-ai/k13d/pkg/log"
)

func GetPodsView(ctx context.Context, client *k8s.Client, namespace, filter string, getStatusColor func(string) any) (ResourceView, error) {
	log.Infof("GetPodsView called for namespace: %s, filter: %s", namespace, filter)
	headers := []string{"NAMESPACE", "NAME", "READY", "STATUS", "RESTARTS", "CPU(m)", "MEM(MB)", "AGE"}
	podsList, err := client.ListPods(ctx, namespace)
	if err != nil {
		return ResourceView{}, err
	}

	log.Infof("GetPodsView: fetching pod metrics for namespace: %s", namespace)
	metrics, err := client.GetPodMetrics(ctx, namespace)
	if err != nil {
		log.Warnf("Failed to fetch pod metrics: %v", err)
	} else {
		log.Infof("Successfully fetched metrics for %d pods", len(metrics))
	}
	var rows [][]TableCell

	for _, p := range podsList {
		if filter != "" && !strings.Contains(p.Name, filter) {
			continue
		}
		readyParts := "0/0"
		if len(p.Status.ContainerStatuses) > 0 {
			ready := 0
			for _, cs := range p.Status.ContainerStatuses {
				if cs.Ready {
					ready++
				}
			}
			readyParts = fmt.Sprintf("%d/%d", ready, len(p.Status.ContainerStatuses))
		}
		restarts := 0
		if len(p.Status.ContainerStatuses) > 0 {
			restarts = int(p.Status.ContainerStatuses[0].RestartCount)
		}

		cpu, mem := "-", "-"
		if m, ok := metrics[p.Name]; ok {
			cpu = fmt.Sprintf("%dm", m[0])
			mem = fmt.Sprintf("%dMB", m[1])
		}

		rows = append(rows, []TableCell{
			{Text: p.Namespace},
			{Text: p.Name}, // Color handled by dashboard for now or passed in
			{Text: readyParts},
			{Text: string(p.Status.Phase)},
			{Text: fmt.Sprintf("%d", restarts)},
			{Text: cpu},
			{Text: mem},
			{Text: FormatAge(time.Since(p.CreationTimestamp.Time))},
		})
	}

	return ResourceView{Headers: headers, Rows: rows}, nil
}
