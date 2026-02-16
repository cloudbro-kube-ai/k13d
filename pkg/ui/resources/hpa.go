package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
)

func GetHPAView(ctx context.Context, client *k8s.Client, namespace, filter string) (ResourceView, error) {
	headers := []string{"NAMESPACE", "NAME", "REFERENCE", "TARGETS", "MINPODS", "MAXPODS", "REPLICAS", "AGE"}
	hpas, err := client.ListHorizontalPodAutoscalers(ctx, namespace)
	if err != nil {
		return ResourceView{}, err
	}

	var rows [][]TableCell
	for _, hpa := range hpas {
		if filter != "" && !strings.Contains(hpa.Name, filter) && !strings.Contains(hpa.Namespace, filter) {
			continue
		}

		// Reference
		reference := fmt.Sprintf("%s/%s", hpa.Spec.ScaleTargetRef.Kind, hpa.Spec.ScaleTargetRef.Name)

		// Targets - show current/target for CPU if available
		targets := "<unknown>"
		if len(hpa.Spec.Metrics) > 0 {
			var targetStrs []string
			for _, metric := range hpa.Spec.Metrics {
				if metric.Resource != nil && metric.Resource.Target.AverageUtilization != nil {
					targetStrs = append(targetStrs, fmt.Sprintf("%s:%d%%", metric.Resource.Name, *metric.Resource.Target.AverageUtilization))
				}
			}
			if len(targetStrs) > 0 {
				targets = strings.Join(targetStrs, ", ")
			}
		}

		// Add current metrics if available
		if len(hpa.Status.CurrentMetrics) > 0 {
			for i, current := range hpa.Status.CurrentMetrics {
				if current.Resource != nil && current.Resource.Current.AverageUtilization != nil {
					if i < len(hpa.Spec.Metrics) && hpa.Spec.Metrics[i].Resource != nil {
						target := int32(0)
						if hpa.Spec.Metrics[i].Resource.Target.AverageUtilization != nil {
							target = *hpa.Spec.Metrics[i].Resource.Target.AverageUtilization
						}
						targets = fmt.Sprintf("%d%%/%d%%", *current.Resource.Current.AverageUtilization, target)
						break
					}
				}
			}
		}

		minPods := "1"
		if hpa.Spec.MinReplicas != nil {
			minPods = fmt.Sprintf("%d", *hpa.Spec.MinReplicas)
		}
		maxPods := fmt.Sprintf("%d", hpa.Spec.MaxReplicas)

		replicas := fmt.Sprintf("%d", hpa.Status.CurrentReplicas)
		replicasColor := tcell.ColorWhite
		if hpa.Status.CurrentReplicas >= hpa.Spec.MaxReplicas {
			replicasColor = tcell.ColorYellow
		} else if hpa.Status.CurrentReplicas == 0 {
			replicasColor = tcell.ColorRed
		} else {
			replicasColor = tcell.ColorGreen
		}

		rows = append(rows, []TableCell{
			{Text: hpa.Namespace},
			{Text: hpa.Name},
			{Text: reference},
			{Text: targets},
			{Text: minPods},
			{Text: maxPods},
			{Text: replicas, Color: replicasColor},
			{Text: FormatAge(time.Since(hpa.CreationTimestamp.Time))},
		})
	}

	return ResourceView{Headers: headers, Rows: rows}, nil
}
