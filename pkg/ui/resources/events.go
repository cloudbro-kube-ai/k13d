package resources

import (
	"context"
	"strings"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"
)

func GetEventsView(ctx context.Context, client *k8s.Client, namespace, filter string) (ResourceView, error) {
	headers := []string{"NAMESPACE", "LAST SEEN", "TYPE", "REASON", "OBJECT", "MESSAGE"}
	events, err := client.ListEvents(ctx, namespace)
	if err != nil {
		return ResourceView{}, err
	}

	var rows [][]TableCell
	for _, ev := range events {
		if filter != "" && !strings.Contains(ev.Message, filter) && !strings.Contains(ev.InvolvedObject.Name, filter) {
			continue
		}

		lastSeen := formatAge(time.Since(ev.LastTimestamp.Time))
		if ev.LastTimestamp.IsZero() {
			lastSeen = formatAge(time.Since(ev.EventTime.Time))
		}

		rows = append(rows, []TableCell{
			{Text: ev.Namespace},
			{Text: lastSeen},
			{Text: ev.Type},
			{Text: ev.Reason},
			{Text: ev.InvolvedObject.Kind + "/" + ev.InvolvedObject.Name},
			{Text: ev.Message},
		})
	}

	return ResourceView{Headers: headers, Rows: rows}, nil
}
