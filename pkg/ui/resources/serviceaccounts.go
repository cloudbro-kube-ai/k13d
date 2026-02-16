package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
)

func GetServiceAccountsView(ctx context.Context, client *k8s.Client, namespace, filter string) (ResourceView, error) {
	headers := []string{"NAMESPACE", "NAME", "SECRETS", "AGE"}
	sa, err := client.ListServiceAccounts(ctx, namespace)
	if err != nil {
		return ResourceView{}, err
	}
	var rows [][]TableCell
	for _, s := range sa {
		if filter != "" && !strings.Contains(s.Name, filter) {
			continue
		}
		rows = append(rows, []TableCell{
			{Text: s.Namespace},
			{Text: s.Name},
			{Text: fmt.Sprintf("%d", len(s.Secrets))},
			{Text: FormatAge(time.Since(s.CreationTimestamp.Time))},
		})
	}
	return ResourceView{Headers: headers, Rows: rows}, nil
}
