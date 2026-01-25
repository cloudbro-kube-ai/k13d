package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"
)

func GetStatefulSetsView(ctx context.Context, client *k8s.Client, namespace, filter string) (ResourceView, error) {
	headers := []string{"NAMESPACE", "NAME", "READY", "AGE"}
	stses, err := client.ListStatefulSets(ctx, namespace)
	if err != nil {
		return ResourceView{}, err
	}

	var rows [][]TableCell
	for _, sts := range stses {
		if filter != "" && !strings.Contains(sts.Name, filter) {
			continue
		}

		ready := fmt.Sprintf("%d/%d", sts.Status.ReadyReplicas, sts.Status.Replicas)

		rows = append(rows, []TableCell{
			{Text: sts.Namespace},
			{Text: sts.Name},
			{Text: ready},
			{Text: formatAge(time.Since(sts.CreationTimestamp.Time))},
		})
	}

	return ResourceView{Headers: headers, Rows: rows}, nil
}
