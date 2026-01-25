package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"
)

// ColorCyan is a helper constant for cyan color
var colorCyan = tcell.NewRGBColor(0, 255, 255)

func GetCronJobsView(ctx context.Context, client *k8s.Client, namespace, filter string) (ResourceView, error) {
	headers := []string{"NAMESPACE", "NAME", "SCHEDULE", "SUSPEND", "ACTIVE", "LAST SCHEDULE", "AGE"}
	cjs, err := client.ListCronJobs(ctx, namespace)
	if err != nil {
		return ResourceView{}, err
	}

	var rows [][]TableCell
	for _, cj := range cjs {
		if filter != "" && !strings.Contains(cj.Name, filter) && !strings.Contains(cj.Namespace, filter) {
			continue
		}

		schedule := cj.Spec.Schedule

		suspend := "False"
		suspendColor := tcell.ColorGreen
		if cj.Spec.Suspend != nil && *cj.Spec.Suspend {
			suspend = "True"
			suspendColor = tcell.ColorYellow
		}

		active := fmt.Sprintf("%d", len(cj.Status.Active))
		activeColor := tcell.ColorWhite
		if len(cj.Status.Active) > 0 {
			activeColor = colorCyan
		}

		lastSchedule := "<none>"
		if cj.Status.LastScheduleTime != nil {
			lastSchedule = formatAge(time.Since(cj.Status.LastScheduleTime.Time))
		}

		rows = append(rows, []TableCell{
			{Text: cj.Namespace},
			{Text: cj.Name},
			{Text: schedule},
			{Text: suspend, Color: suspendColor},
			{Text: active, Color: activeColor},
			{Text: lastSchedule},
			{Text: formatAge(time.Since(cj.CreationTimestamp.Time))},
		})
	}

	return ResourceView{Headers: headers, Rows: rows}, nil
}
