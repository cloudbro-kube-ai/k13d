package resources

import (
	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
	"github.com/gdamore/tcell/v2"
)

func GetContextsView(client *k8s.Client) (ResourceView, string, error) {
	headers := []string{"NAME", "CURRENT"}
	contexts, current, err := client.ListContexts()
	if err != nil {
		return ResourceView{}, "", err
	}
	var rows [][]TableCell
	for _, name := range contexts {
		color := tcell.ColorWhite
		indicator := ""
		if name == current {
			color = tcell.ColorGreen
			indicator = "*"
		}
		rows = append(rows, []TableCell{
			{Text: name, Color: color},
			{Text: indicator, Color: color},
		})
	}
	return ResourceView{Headers: headers, Rows: rows}, current, nil
}
