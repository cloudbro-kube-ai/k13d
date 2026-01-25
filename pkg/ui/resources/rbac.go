package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"
)

func GetRolesView(ctx context.Context, client *k8s.Client, namespace, filter string) (ResourceView, error) {
	headers := []string{"NAMESPACE", "NAME", "AGE"}
	roles, err := client.ListRoles(ctx, namespace)
	if err != nil {
		return ResourceView{}, err
	}
	var rows [][]TableCell
	for _, r := range roles {
		if filter != "" && !strings.Contains(r.Name, filter) {
			continue
		}
		rows = append(rows, []TableCell{
			{Text: r.Namespace},
			{Text: r.Name},
			{Text: formatAge(time.Since(r.CreationTimestamp.Time))},
		})
	}
	return ResourceView{Headers: headers, Rows: rows}, nil
}

func GetRoleBindingsView(ctx context.Context, client *k8s.Client, namespace, filter string) (ResourceView, error) {
	headers := []string{"NAMESPACE", "NAME", "ROLE", "AGE"}
	rb, err := client.ListRoleBindings(ctx, namespace)
	if err != nil {
		return ResourceView{}, err
	}
	var rows [][]TableCell
	for _, b := range rb {
		if filter != "" && !strings.Contains(b.Name, filter) {
			continue
		}
		rows = append(rows, []TableCell{
			{Text: b.Namespace},
			{Text: b.Name},
			{Text: fmt.Sprintf("%s/%s", b.RoleRef.Kind, b.RoleRef.Name)},
			{Text: formatAge(time.Since(b.CreationTimestamp.Time))},
		})
	}
	return ResourceView{Headers: headers, Rows: rows}, nil
}

func GetClusterRolesView(ctx context.Context, client *k8s.Client, filter string) (ResourceView, error) {
	headers := []string{"NAME", "AGE"}
	roles, err := client.ListClusterRoles(ctx)
	if err != nil {
		return ResourceView{}, err
	}
	var rows [][]TableCell
	for _, r := range roles {
		if filter != "" && !strings.Contains(r.Name, filter) {
			continue
		}
		rows = append(rows, []TableCell{
			{Text: r.Name},
			{Text: formatAge(time.Since(r.CreationTimestamp.Time))},
		})
	}
	return ResourceView{Headers: headers, Rows: rows}, nil
}

func GetClusterRoleBindingsView(ctx context.Context, client *k8s.Client, filter string) (ResourceView, error) {
	headers := []string{"NAME", "ROLE", "AGE"}
	crb, err := client.ListClusterRoleBindings(ctx)
	if err != nil {
		return ResourceView{}, err
	}
	var rows [][]TableCell
	for _, b := range crb {
		if filter != "" && !strings.Contains(b.Name, filter) {
			continue
		}
		rows = append(rows, []TableCell{
			{Text: b.Name},
			{Text: fmt.Sprintf("%s/%s", b.RoleRef.Kind, b.RoleRef.Name)},
			{Text: formatAge(time.Since(b.CreationTimestamp.Time))},
		})
	}
	return ResourceView{Headers: headers, Rows: rows}, nil
}
