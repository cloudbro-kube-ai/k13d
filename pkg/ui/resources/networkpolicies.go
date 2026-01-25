package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/k8s"
)

func GetNetworkPoliciesView(ctx context.Context, client *k8s.Client, namespace, filter string) (ResourceView, error) {
	headers := []string{"NAMESPACE", "NAME", "POD-SELECTOR", "POLICY-TYPES", "AGE"}
	netpols, err := client.ListNetworkPolicies(ctx, namespace)
	if err != nil {
		return ResourceView{}, err
	}

	var rows [][]TableCell
	for _, np := range netpols {
		if filter != "" && !strings.Contains(np.Name, filter) && !strings.Contains(np.Namespace, filter) {
			continue
		}

		// Pod Selector
		podSelector := "<all>"
		if len(np.Spec.PodSelector.MatchLabels) > 0 {
			var selectors []string
			for k, v := range np.Spec.PodSelector.MatchLabels {
				selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
			}
			podSelector = strings.Join(selectors, ",")
		}

		// Policy Types
		policyTypes := "<none>"
		if len(np.Spec.PolicyTypes) > 0 {
			var types []string
			for _, pt := range np.Spec.PolicyTypes {
				types = append(types, string(pt))
			}
			policyTypes = strings.Join(types, ",")
		}

		rows = append(rows, []TableCell{
			{Text: np.Namespace},
			{Text: np.Name},
			{Text: podSelector},
			{Text: policyTypes},
			{Text: formatAge(time.Since(np.CreationTimestamp.Time))},
		})
	}

	return ResourceView{Headers: headers, Rows: rows}, nil
}
