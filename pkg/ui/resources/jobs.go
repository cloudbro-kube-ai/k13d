package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
)

func GetJobsView(ctx context.Context, client *k8s.Client, namespace, filter string) (ResourceView, error) {
	headers := []string{"NAMESPACE", "NAME", "COMPLETIONS", "DURATION", "AGE"}
	jobs, err := client.ListJobs(ctx, namespace)
	if err != nil {
		return ResourceView{}, err
	}

	var rows [][]TableCell
	for _, job := range jobs {
		if filter != "" && !strings.Contains(job.Name, filter) && !strings.Contains(job.Namespace, filter) {
			continue
		}

		// Completions
		succeeded := job.Status.Succeeded
		completions := int32(1)
		if job.Spec.Completions != nil {
			completions = *job.Spec.Completions
		}
		completionsStr := fmt.Sprintf("%d/%d", succeeded, completions)

		// Duration
		duration := "-"
		if job.Status.StartTime != nil {
			start := job.Status.StartTime.Time
			end := time.Now()
			if job.Status.CompletionTime != nil {
				end = job.Status.CompletionTime.Time
			}
			duration = formatDuration(end.Sub(start))
		}

		// Status color
		color := tcell.ColorWhite
		if succeeded >= completions {
			color = tcell.ColorGreen
		} else if job.Status.Failed > 0 {
			color = tcell.ColorRed
		} else if job.Status.Active > 0 {
			color = tcell.ColorYellow
		}

		rows = append(rows, []TableCell{
			{Text: job.Namespace},
			{Text: job.Name},
			{Text: completionsStr, Color: color},
			{Text: duration},
			{Text: FormatAge(time.Since(job.CreationTimestamp.Time))},
		})
	}

	return ResourceView{Headers: headers, Rows: rows}, nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
	return fmt.Sprintf("%dd%dh", int(d.Hours()/24), int(d.Hours())%24)
}
