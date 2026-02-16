package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// NetPolNode represents a node in the network policy graph
type NetPolNode struct {
	ID        string            `json:"id"`
	Kind      string            `json:"kind"` // "Pod", "Namespace", "External"
	Name      string            `json:"name"`
	Namespace string            `json:"namespace,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// NetPolEdge represents an edge in the network policy graph
type NetPolEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`   // "allow-ingress", "allow-egress"
	Policy string `json:"policy"` // NetworkPolicy name
	Ports  string `json:"ports,omitempty"`
}

// NetPolPolicySummary represents a network policy summary for card-based UI
type NetPolPolicySummary struct {
	Name         string   `json:"name"`
	Namespace    string   `json:"namespace"`
	PodSelector  string   `json:"pod_selector"`
	IngressRules []string `json:"ingress_rules"`
	EgressRules  []string `json:"egress_rules"`
}

// NetPolVisualizationResponse is the response for the network policy visualization endpoint
type NetPolVisualizationResponse struct {
	Nodes       []NetPolNode          `json:"nodes"`
	Edges       []NetPolEdge          `json:"edges"`
	Policies    []NetPolPolicySummary `json:"policies"`
	PolicyCount int                   `json:"policyCount"`
}

func (s *Server) handleNetworkPolicyVisualization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	namespace := r.URL.Query().Get("namespace")
	ctx := r.Context()

	var (
		wg      sync.WaitGroup
		pods    []corev1.Pod
		netpols []networkingv1.NetworkPolicy
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		var err error
		pods, err = s.k8sClient.ListPods(ctx, namespace)
		if err != nil {
			pods = nil
		}
	}()
	go func() {
		defer wg.Done()
		var err error
		netpols, err = s.k8sClient.ListNetworkPolicies(ctx, namespace)
		if err != nil {
			netpols = nil
		}
	}()
	wg.Wait()

	var nodes []NetPolNode
	var edges []NetPolEdge
	nodeSet := make(map[string]bool)

	addNode := func(n NetPolNode) {
		if !nodeSet[n.ID] {
			nodes = append(nodes, n)
			nodeSet[n.ID] = true
		}
	}

	// Add pod nodes
	for _, pod := range pods {
		id := fmt.Sprintf("Pod/%s/%s", pod.Namespace, pod.Name)
		addNode(NetPolNode{
			ID:        id,
			Kind:      "Pod",
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Labels:    pod.Labels,
		})
	}

	// Process network policies
	for _, np := range netpols {
		// Find pods matched by the policy's podSelector
		matchedPods := matchPods(pods, np.Spec.PodSelector.MatchLabels, np.Namespace)

		// Process ingress rules
		for _, ingress := range np.Spec.Ingress {
			ports := formatPolicyPorts(ingress.Ports)
			for _, from := range ingress.From {
				if from.PodSelector != nil {
					// Pods matching the selector can send traffic to matched pods
					sourcePods := matchPods(pods, from.PodSelector.MatchLabels, np.Namespace)
					for _, src := range sourcePods {
						srcID := fmt.Sprintf("Pod/%s/%s", src.Namespace, src.Name)
						for _, dst := range matchedPods {
							dstID := fmt.Sprintf("Pod/%s/%s", dst.Namespace, dst.Name)
							edges = append(edges, NetPolEdge{
								Source: srcID,
								Target: dstID,
								Type:   "allow-ingress",
								Policy: np.Name,
								Ports:  ports,
							})
						}
					}
				}
				if from.NamespaceSelector != nil {
					nsID := fmt.Sprintf("Namespace/%s", formatSelector(from.NamespaceSelector.MatchLabels))
					addNode(NetPolNode{
						ID:   nsID,
						Kind: "Namespace",
						Name: formatSelector(from.NamespaceSelector.MatchLabels),
					})
					for _, dst := range matchedPods {
						dstID := fmt.Sprintf("Pod/%s/%s", dst.Namespace, dst.Name)
						edges = append(edges, NetPolEdge{
							Source: nsID,
							Target: dstID,
							Type:   "allow-ingress",
							Policy: np.Name,
							Ports:  ports,
						})
					}
				}
				if from.IPBlock != nil {
					extID := fmt.Sprintf("External/%s", from.IPBlock.CIDR)
					addNode(NetPolNode{
						ID:   extID,
						Kind: "External",
						Name: from.IPBlock.CIDR,
					})
					for _, dst := range matchedPods {
						dstID := fmt.Sprintf("Pod/%s/%s", dst.Namespace, dst.Name)
						edges = append(edges, NetPolEdge{
							Source: extID,
							Target: dstID,
							Type:   "allow-ingress",
							Policy: np.Name,
							Ports:  ports,
						})
					}
				}
			}
		}

		// Process egress rules
		for _, egress := range np.Spec.Egress {
			ports := formatPolicyPorts(egress.Ports)
			for _, to := range egress.To {
				if to.PodSelector != nil {
					destPods := matchPods(pods, to.PodSelector.MatchLabels, np.Namespace)
					for _, src := range matchedPods {
						srcID := fmt.Sprintf("Pod/%s/%s", src.Namespace, src.Name)
						for _, dst := range destPods {
							dstID := fmt.Sprintf("Pod/%s/%s", dst.Namespace, dst.Name)
							edges = append(edges, NetPolEdge{
								Source: srcID,
								Target: dstID,
								Type:   "allow-egress",
								Policy: np.Name,
								Ports:  ports,
							})
						}
					}
				}
				if to.IPBlock != nil {
					extID := fmt.Sprintf("External/%s", to.IPBlock.CIDR)
					addNode(NetPolNode{
						ID:   extID,
						Kind: "External",
						Name: to.IPBlock.CIDR,
					})
					for _, src := range matchedPods {
						srcID := fmt.Sprintf("Pod/%s/%s", src.Namespace, src.Name)
						edges = append(edges, NetPolEdge{
							Source: srcID,
							Target: extID,
							Type:   "allow-egress",
							Policy: np.Name,
							Ports:  ports,
						})
					}
				}
			}
		}
	}

	// Build policy summaries for card-based UI
	var policySummaries []NetPolPolicySummary
	for _, np := range netpols {
		summary := NetPolPolicySummary{
			Name:        np.Name,
			Namespace:   np.Namespace,
			PodSelector: formatSelector(np.Spec.PodSelector.MatchLabels),
		}
		for _, ingress := range np.Spec.Ingress {
			ports := formatPolicyPorts(ingress.Ports)
			for _, from := range ingress.From {
				rule := ""
				if from.PodSelector != nil {
					rule = "From pods: " + formatSelector(from.PodSelector.MatchLabels)
				} else if from.NamespaceSelector != nil {
					rule = "From ns: " + formatSelector(from.NamespaceSelector.MatchLabels)
				} else if from.IPBlock != nil {
					rule = "From IP: " + from.IPBlock.CIDR
				}
				if rule != "" && ports != "all" {
					rule += " (" + ports + ")"
				}
				if rule != "" {
					summary.IngressRules = append(summary.IngressRules, rule)
				}
			}
			if len(ingress.From) == 0 {
				summary.IngressRules = append(summary.IngressRules, "Allow all ("+ports+")")
			}
		}
		for _, egress := range np.Spec.Egress {
			ports := formatPolicyPorts(egress.Ports)
			for _, to := range egress.To {
				rule := ""
				if to.PodSelector != nil {
					rule = "To pods: " + formatSelector(to.PodSelector.MatchLabels)
				} else if to.NamespaceSelector != nil {
					rule = "To ns: " + formatSelector(to.NamespaceSelector.MatchLabels)
				} else if to.IPBlock != nil {
					rule = "To IP: " + to.IPBlock.CIDR
				}
				if rule != "" && ports != "all" {
					rule += " (" + ports + ")"
				}
				if rule != "" {
					summary.EgressRules = append(summary.EgressRules, rule)
				}
			}
			if len(egress.To) == 0 {
				summary.EgressRules = append(summary.EgressRules, "Allow all ("+ports+")")
			}
		}
		policySummaries = append(policySummaries, summary)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NetPolVisualizationResponse{
		Nodes:       nodes,
		Edges:       edges,
		Policies:    policySummaries,
		PolicyCount: len(netpols),
	})
}

// matchPods returns pods that match the given label selector in the specified namespace
func matchPods(pods []corev1.Pod, selector map[string]string, namespace string) []corev1.Pod {
	var matched []corev1.Pod
	for _, pod := range pods {
		if namespace != "" && pod.Namespace != namespace {
			continue
		}
		if labelsContain(pod.Labels, selector) {
			matched = append(matched, pod)
		}
	}
	return matched
}

// labelsContain checks if all selector labels are present in the target labels
func labelsContain(labels, selector map[string]string) bool {
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
}

// formatPolicyPorts formats NetworkPolicyPort slice into a readable string
func formatPolicyPorts(ports []networkingv1.NetworkPolicyPort) string {
	if len(ports) == 0 {
		return "all"
	}
	var parts []string
	for _, p := range ports {
		proto := "TCP"
		if p.Protocol != nil {
			proto = string(*p.Protocol)
		}
		if p.Port != nil {
			parts = append(parts, fmt.Sprintf("%s/%s", p.Port.String(), proto))
		} else {
			parts = append(parts, proto)
		}
	}
	return strings.Join(parts, ", ")
}

// formatSelector formats a label selector map into a readable string
func formatSelector(labels map[string]string) string {
	if len(labels) == 0 {
		return "*"
	}
	var parts []string
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, ",")
}
