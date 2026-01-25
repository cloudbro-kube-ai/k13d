package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"
)

func GetConfigMapsView(ctx context.Context, client *k8s.Client, namespace, filter string) (ResourceView, error) {
	headers := []string{"NAMESPACE", "NAME", "DATA", "AGE"}
	cms, err := client.ListConfigMaps(ctx, namespace)
	if err != nil {
		return ResourceView{}, err
	}

	var rows [][]TableCell
	for _, cm := range cms {
		if filter != "" && !strings.Contains(cm.Name, filter) {
			continue
		}
		rows = append(rows, []TableCell{
			{Text: cm.Namespace},
			{Text: cm.Name},
			{Text: fmt.Sprintf("%d", len(cm.Data))},
			{Text: formatAge(time.Since(cm.CreationTimestamp.Time))},
		})
	}

	return ResourceView{Headers: headers, Rows: rows}, nil
}
