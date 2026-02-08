package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"
)

func GetNodesView(ctx context.Context, client *k8s.Client, filter string) (ResourceView, error) {
	headers := []string{"NAME", "STATUS", "ROLES", "VERSION", "CPU(m)", "MEM(MB)", "AGE"}
	nodes, err := client.ListNodes(ctx)
	if err != nil {
		return ResourceView{}, err
	}

	metrics, _ := client.GetNodeMetrics(ctx)
	var rows [][]TableCell

	for _, n := range nodes {
		if filter != "" && !strings.Contains(n.Name, filter) {
			continue
		}
		status := "Ready"
		for _, c := range n.Status.Conditions {
			if c.Type == "Ready" && c.Status != "True" {
				status = "NotReady"
			}
		}
		roles := "<none>"
		for l := range n.Labels {
			if strings.HasPrefix(l, "node-role.kubernetes.io/") {
				roles = strings.TrimPrefix(l, "node-role.kubernetes.io/")
			}
		}

		cpu, mem := "-", "-"
		if m, ok := metrics[n.Name]; ok {
			cpu = fmt.Sprintf("%dm", m[0])
			mem = fmt.Sprintf("%dMB", m[1])
		}

		rows = append(rows, []TableCell{
			{Text: n.Name},
			{Text: status},
			{Text: roles},
			{Text: n.Status.NodeInfo.KubeletVersion},
			{Text: cpu},
			{Text: mem},
			{Text: FormatAge(time.Since(n.CreationTimestamp.Time))},
		})
	}

	return ResourceView{Headers: headers, Rows: rows}, nil
}
