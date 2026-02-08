package resources

import (
	"context"
	"strings"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"
)

func GetIngressesView(ctx context.Context, client *k8s.Client, namespace, filter string) (ResourceView, error) {
	headers := []string{"NAMESPACE", "NAME", "CLASS", "ADDRESS", "PORTS", "AGE"}
	ings, err := client.ListIngresses(ctx, namespace)
	if err != nil {
		return ResourceView{}, err
	}

	var rows [][]TableCell
	for _, ing := range ings {
		if filter != "" && !strings.Contains(ing.Name, filter) {
			continue
		}
		class := "<none>"
		if ing.Spec.IngressClassName != nil {
			class = *ing.Spec.IngressClassName
		}
		address := ""
		for _, lb := range ing.Status.LoadBalancer.Ingress {
			if lb.IP != "" {
				address = lb.IP
			} else {
				address = lb.Hostname
			}
		}
		rows = append(rows, []TableCell{
			{Text: ing.Namespace},
			{Text: ing.Name},
			{Text: class},
			{Text: address},
			{Text: "80, 443"},
			{Text: FormatAge(time.Since(ing.CreationTimestamp.Time))},
		})
	}

	return ResourceView{Headers: headers, Rows: rows}, nil
}
