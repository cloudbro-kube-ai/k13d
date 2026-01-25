package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"
)

func GetSecretsView(ctx context.Context, client *k8s.Client, namespace, filter string) (ResourceView, error) {
	headers := []string{"NAMESPACE", "NAME", "TYPE", "DATA", "AGE"}
	secrets, err := client.ListSecrets(ctx, namespace)
	if err != nil {
		return ResourceView{}, err
	}

	var rows [][]TableCell
	for _, s := range secrets {
		if filter != "" && !strings.Contains(s.Name, filter) {
			continue
		}
		rows = append(rows, []TableCell{
			{Text: s.Namespace},
			{Text: s.Name},
			{Text: string(s.Type)},
			{Text: fmt.Sprintf("%d", len(s.Data))},
			{Text: formatAge(time.Since(s.CreationTimestamp.Time))},
		})
	}

	return ResourceView{Headers: headers, Rows: rows}, nil
}
