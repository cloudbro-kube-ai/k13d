package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
	"github.com/gdamore/tcell/v2"
)

func GetDaemonSetsView(ctx context.Context, client *k8s.Client, namespace, filter string) (ResourceView, error) {
	headers := []string{"NAMESPACE", "NAME", "DESIRED", "CURRENT", "READY", "UP-TO-DATE", "AVAILABLE", "NODE SELECTOR", "AGE"}
	dss, err := client.ListDaemonSets(ctx, namespace)
	if err != nil {
		return ResourceView{}, err
	}

	var rows [][]TableCell
	for _, ds := range dss {
		if filter != "" && !strings.Contains(ds.Name, filter) && !strings.Contains(ds.Namespace, filter) {
			continue
		}

		desired := fmt.Sprintf("%d", ds.Status.DesiredNumberScheduled)
		current := fmt.Sprintf("%d", ds.Status.CurrentNumberScheduled)
		ready := fmt.Sprintf("%d", ds.Status.NumberReady)
		upToDate := fmt.Sprintf("%d", ds.Status.UpdatedNumberScheduled)
		available := fmt.Sprintf("%d", ds.Status.NumberAvailable)

		// Node selector
		nodeSelector := "<none>"
		if len(ds.Spec.Template.Spec.NodeSelector) > 0 {
			var selectors []string
			for k, v := range ds.Spec.Template.Spec.NodeSelector {
				selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
			}
			nodeSelector = strings.Join(selectors, ",")
		}

		// Status color
		color := tcell.ColorWhite
		if ds.Status.NumberReady == ds.Status.DesiredNumberScheduled && ds.Status.DesiredNumberScheduled > 0 {
			color = tcell.ColorGreen
		} else if ds.Status.NumberReady == 0 && ds.Status.DesiredNumberScheduled > 0 {
			color = tcell.ColorRed
		} else if ds.Status.NumberReady < ds.Status.DesiredNumberScheduled {
			color = tcell.ColorYellow
		}

		rows = append(rows, []TableCell{
			{Text: ds.Namespace},
			{Text: ds.Name},
			{Text: desired},
			{Text: current},
			{Text: ready, Color: color},
			{Text: upToDate},
			{Text: available},
			{Text: nodeSelector},
			{Text: FormatAge(time.Since(ds.CreationTimestamp.Time))},
		})
	}

	return ResourceView{Headers: headers, Rows: rows}, nil
}
