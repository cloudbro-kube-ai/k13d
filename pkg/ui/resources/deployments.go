package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
)

func GetDeploymentsView(ctx context.Context, client *k8s.Client, namespace, filter string) (ResourceView, error) {
	headers := []string{"NAMESPACE", "NAME", "READY", "UP-TO-DATE", "AVAILABLE", "AGE"}
	deps, err := client.ListDeployments(ctx, namespace)
	if err != nil {
		return ResourceView{}, err
	}

	var rows [][]TableCell
	for _, dep := range deps {
		if filter != "" && !strings.Contains(dep.Name, filter) {
			continue
		}

		ready := fmt.Sprintf("%d/%d", dep.Status.ReadyReplicas, dep.Status.Replicas)
		upToDate := fmt.Sprintf("%d", dep.Status.UpdatedReplicas)
		available := fmt.Sprintf("%d", dep.Status.AvailableReplicas)

		rows = append(rows, []TableCell{
			{Text: dep.Namespace},
			{Text: dep.Name},
			{Text: ready},
			{Text: upToDate},
			{Text: available},
			{Text: FormatAge(time.Since(dep.CreationTimestamp.Time))},
		})
	}

	return ResourceView{Headers: headers, Rows: rows}, nil
}
