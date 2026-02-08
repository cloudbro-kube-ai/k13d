package resources

import (
	"context"
	"strings"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"
)

func GetNamespacesView(ctx context.Context, client *k8s.Client, filter string) (ResourceView, error) {
	headers := []string{"NAME", "STATUS", "AGE"}
	namespaces, err := client.ListNamespaces(ctx)
	if err != nil {
		return ResourceView{}, err
	}

	var rows [][]TableCell
	for _, ns := range namespaces {
		if filter != "" && !strings.Contains(ns.Name, filter) {
			continue
		}
		status := string(ns.Status.Phase)
		rows = append(rows, []TableCell{
			{Text: ns.Name},
			{Text: status},
			{Text: FormatAge(time.Since(ns.CreationTimestamp.Time))},
		})
	}

	return ResourceView{Headers: headers, Rows: rows}, nil
}
