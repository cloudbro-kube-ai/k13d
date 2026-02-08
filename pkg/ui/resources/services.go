package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"
)

func GetServicesView(ctx context.Context, client *k8s.Client, namespace, filter string) (ResourceView, error) {
	headers := []string{"NAMESPACE", "NAME", "TYPE", "CLUSTER-IP", "EXTERNAL-IP", "PORT(S)", "AGE"}
	svcs, err := client.ListServices(ctx, namespace)
	if err != nil {
		return ResourceView{}, err
	}

	var rows [][]TableCell
	for _, svc := range svcs {
		if filter != "" && !strings.Contains(svc.Name, filter) {
			continue
		}
		externalIP := "<none>"
		if len(svc.Status.LoadBalancer.Ingress) > 0 {
			if svc.Status.LoadBalancer.Ingress[0].IP != "" {
				externalIP = svc.Status.LoadBalancer.Ingress[0].IP
			} else if svc.Status.LoadBalancer.Ingress[0].Hostname != "" {
				externalIP = svc.Status.LoadBalancer.Ingress[0].Hostname
			}
		}

		ports := []string{}
		for _, p := range svc.Spec.Ports {
			ports = append(ports, fmt.Sprintf("%d/%s", p.Port, p.Protocol))
		}

		rows = append(rows, []TableCell{
			{Text: svc.Namespace},
			{Text: svc.Name},
			{Text: string(svc.Spec.Type)},
			{Text: svc.Spec.ClusterIP},
			{Text: externalIP},
			{Text: strings.Join(ports, ",")},
			{Text: FormatAge(time.Since(svc.CreationTimestamp.Time))},
		})
	}

	return ResourceView{Headers: headers, Rows: rows}, nil
}
