package ai

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Action struct {
	Timestamp time.Time
	Resource  string
	Namespace string
	Name      string
	Operation string // Create, Patch, Delete
	Details   string
}

type Reporter struct {
	OutputPath string
	Actions    []Action
}

func NewReporter(outputPath string) *Reporter {
	if outputPath == "" {
		home, _ := os.UserHomeDir()
		outputPath = filepath.Join(home, ".local", "state", "k13d", "reports")
	}
	_ = os.MkdirAll(outputPath, 0755)
	return &Reporter{
		OutputPath: outputPath,
	}
}

func (r *Reporter) Record(action Action) {
	r.Actions = append(r.Actions, action)
	_ = r.SaveHTML() // Auto-save for now
}

func (r *Reporter) SaveHTML() error {
	fileName := fmt.Sprintf("report-%s.html", time.Now().Format("2006-01-02"))
	path := filepath.Join(r.OutputPath, fileName)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprint(f, `<!DOCTYPE html>
<html>
<head>
    <title>k13d Agentic Report</title>
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background-color: #f4f7f6; color: #333; margin: 40px; }
        h1 { color: #2c3e50; border-bottom: 2px solid #3498db; padding-bottom: 10px; }
        table { width: 100%; border-collapse: collapse; margin-top: 20px; background: white; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
        th, td { padding: 15px; text-align: left; border-bottom: 1px solid #eee; }
        th { background-color: #3498db; color: white; }
        tr:hover { background-color: #f1f1f1; }
        .operation { font-weight: bold; text-transform: uppercase; }
        .create { color: #27ae60; }
        .patch { color: #f39c12; }
        .delete { color: #e74c3c; }
        details { font-size: 0.9em; color: #666; cursor: pointer; }
    </style>
</head>
<body>
    <h1>k13d Agentic Resource Change Report</h1>
    <p>Generated on: `+time.Now().Format(time.RFC1123)+`</p>
    <table>
        <thead>
            <tr>
                <th>Time</th>
                <th>Operation</th>
                <th>Resource</th>
                <th>Namespace</th>
                <th>Name</th>
                <th>Details</th>
            </tr>
        </thead>
        <tbody>`)

	for _, a := range r.Actions {
		opClass := strings.ToLower(a.Operation)
		fmt.Fprintf(f, `
            <tr>
                <td>%s</td>
                <td><span class="operation %s">%s</span></td>
                <td>%s</td>
                <td>%s</td>
                <td>%s</td>
                <td><details><summary>View Details</summary><pre>%s</pre></details></td>
            </tr>`,
			a.Timestamp.Format("15:04:05"),
			html.EscapeString(opClass), html.EscapeString(a.Operation),
			html.EscapeString(a.Resource),
			html.EscapeString(a.Namespace),
			html.EscapeString(a.Name),
			html.EscapeString(a.Details),
		)
	}

	fmt.Fprint(f, `
        </tbody>
    </table>
</body>
</html>`)

	return nil
}
