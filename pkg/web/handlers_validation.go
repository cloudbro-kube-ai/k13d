package web

import (
	"encoding/json"
	"net/http"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai/analyzers"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// handleValidate runs cross-resource validation for a namespace.
func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		MethodNotAllowed(w, http.MethodGet)
		return
	}

	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		BadRequest(w, "namespace query parameter is required")
		return
	}

	ctx := r.Context()

	// Fetch resources from the namespace
	var resources []*analyzers.ResourceInfo
	resourceCounts := map[string]int{}

	// Pods
	pods, err := s.k8sClient.ListPods(ctx, namespace)
	if err == nil {
		resourceCounts["pods"] = len(pods)
		for i := range pods {
			resources = append(resources, &analyzers.ResourceInfo{
				Kind:      "Pod",
				Name:      pods[i].Name,
				Namespace: pods[i].Namespace,
				Labels:    pods[i].Labels,
				Raw:       extractPodRefs(&pods[i]),
			})
		}
	}

	// Services
	services, err := s.k8sClient.ListServices(ctx, namespace)
	if err == nil {
		resourceCounts["services"] = len(services)
		for i := range services {
			raw := make(map[string]interface{})
			if len(services[i].Spec.Selector) > 0 {
				raw["selectorLabels"] = services[i].Spec.Selector
			}
			resources = append(resources, &analyzers.ResourceInfo{
				Kind:      "Service",
				Name:      services[i].Name,
				Namespace: services[i].Namespace,
				Labels:    services[i].Labels,
				Raw:       raw,
			})
		}
	}

	// ConfigMaps
	configMaps, err := s.k8sClient.ListConfigMaps(ctx, namespace)
	if err == nil {
		resourceCounts["configmaps"] = len(configMaps)
		for i := range configMaps {
			resources = append(resources, &analyzers.ResourceInfo{
				Kind:      "ConfigMap",
				Name:      configMaps[i].Name,
				Namespace: configMaps[i].Namespace,
				Labels:    configMaps[i].Labels,
			})
		}
	}

	// Secrets
	secrets, err := s.k8sClient.ListSecrets(ctx, namespace)
	if err == nil {
		resourceCounts["secrets"] = len(secrets)
		for i := range secrets {
			resources = append(resources, &analyzers.ResourceInfo{
				Kind:      "Secret",
				Name:      secrets[i].Name,
				Namespace: secrets[i].Namespace,
				Labels:    secrets[i].Labels,
			})
		}
	}

	// Deployments
	deployments, err := s.k8sClient.ListDeployments(ctx, namespace)
	if err == nil {
		resourceCounts["deployments"] = len(deployments)
		for i := range deployments {
			resources = append(resources, &analyzers.ResourceInfo{
				Kind:      "Deployment",
				Name:      deployments[i].Name,
				Namespace: deployments[i].Namespace,
				Labels:    deployments[i].Labels,
				Raw:       extractDeploymentRefs(&deployments[i]),
			})
		}
	}

	// StatefulSets
	statefulSets, err := s.k8sClient.ListStatefulSets(ctx, namespace)
	if err == nil {
		resourceCounts["statefulsets"] = len(statefulSets)
		for i := range statefulSets {
			resources = append(resources, &analyzers.ResourceInfo{
				Kind:      "StatefulSet",
				Name:      statefulSets[i].Name,
				Namespace: statefulSets[i].Namespace,
				Labels:    statefulSets[i].Labels,
			})
		}
	}

	// Ingresses
	ingresses, err := s.k8sClient.ListIngresses(ctx, namespace)
	if err == nil {
		resourceCounts["ingresses"] = len(ingresses)
		for i := range ingresses {
			raw := make(map[string]interface{})
			var svcRefs []string
			for _, rule := range ingresses[i].Spec.Rules {
				if rule.HTTP == nil {
					continue
				}
				for _, path := range rule.HTTP.Paths {
					if path.Backend.Service != nil {
						svcRefs = append(svcRefs, path.Backend.Service.Name)
					}
				}
			}
			if ingresses[i].Spec.DefaultBackend != nil && ingresses[i].Spec.DefaultBackend.Service != nil {
				svcRefs = append(svcRefs, ingresses[i].Spec.DefaultBackend.Service.Name)
			}
			if len(svcRefs) > 0 {
				raw["serviceRefs"] = svcRefs
			}
			resources = append(resources, &analyzers.ResourceInfo{
				Kind:      "Ingress",
				Name:      ingresses[i].Name,
				Namespace: ingresses[i].Namespace,
				Labels:    ingresses[i].Labels,
				Raw:       raw,
			})
		}
	}

	// HPAs
	hpas, err := s.k8sClient.ListHorizontalPodAutoscalers(ctx, namespace)
	if err == nil {
		resourceCounts["hpas"] = len(hpas)
		for i := range hpas {
			raw := make(map[string]interface{})
			targetRef := hpas[i].Spec.ScaleTargetRef.Kind + "/" + hpas[i].Spec.ScaleTargetRef.Name
			raw["targetRef"] = targetRef
			resources = append(resources, &analyzers.ResourceInfo{
				Kind:      "HorizontalPodAutoscaler",
				Name:      hpas[i].Name,
				Namespace: hpas[i].Namespace,
				Labels:    hpas[i].Labels,
				Raw:       raw,
			})
		}
	}

	// Run cross-validation
	registry := analyzers.DefaultRegistry()
	findings := registry.ValidateCross(ctx, resources)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"namespace":       namespace,
		"findings":        findings,
		"total":           len(findings),
		"resources_scanned": len(resources),
		"resource_counts": resourceCounts,
	})
}

// extractPodRefs extracts ConfigMap and Secret references from a Pod.
func extractPodRefs(pod *corev1.Pod) map[string]interface{} {
	raw := make(map[string]interface{})
	var cmRefs, secretRefs []string

	for _, vol := range pod.Spec.Volumes {
		if vol.ConfigMap != nil {
			cmRefs = append(cmRefs, vol.ConfigMap.Name)
		}
		if vol.Secret != nil {
			secretRefs = append(secretRefs, vol.Secret.SecretName)
		}
	}

	for _, container := range pod.Spec.Containers {
		for _, envFrom := range container.EnvFrom {
			if envFrom.ConfigMapRef != nil {
				cmRefs = append(cmRefs, envFrom.ConfigMapRef.Name)
			}
			if envFrom.SecretRef != nil {
				secretRefs = append(secretRefs, envFrom.SecretRef.Name)
			}
		}
		for _, env := range container.Env {
			if env.ValueFrom == nil {
				continue
			}
			if env.ValueFrom.ConfigMapKeyRef != nil {
				cmRefs = append(cmRefs, env.ValueFrom.ConfigMapKeyRef.Name)
			}
			if env.ValueFrom.SecretKeyRef != nil {
				secretRefs = append(secretRefs, env.ValueFrom.SecretKeyRef.Name)
			}
		}
	}

	if len(cmRefs) > 0 {
		raw["configMapRefs"] = dedupStrings(cmRefs)
	}
	if len(secretRefs) > 0 {
		raw["secretRefs"] = dedupStrings(secretRefs)
	}
	return raw
}

// extractDeploymentRefs extracts ConfigMap and Secret references from a Deployment's pod template.
func extractDeploymentRefs(deploy *appsv1.Deployment) map[string]interface{} {
	raw := make(map[string]interface{})
	var cmRefs, secretRefs []string

	for _, vol := range deploy.Spec.Template.Spec.Volumes {
		if vol.ConfigMap != nil {
			cmRefs = append(cmRefs, vol.ConfigMap.Name)
		}
		if vol.Secret != nil {
			secretRefs = append(secretRefs, vol.Secret.SecretName)
		}
	}

	for _, container := range deploy.Spec.Template.Spec.Containers {
		for _, envFrom := range container.EnvFrom {
			if envFrom.ConfigMapRef != nil {
				cmRefs = append(cmRefs, envFrom.ConfigMapRef.Name)
			}
			if envFrom.SecretRef != nil {
				secretRefs = append(secretRefs, envFrom.SecretRef.Name)
			}
		}
		for _, env := range container.Env {
			if env.ValueFrom == nil {
				continue
			}
			if env.ValueFrom.ConfigMapKeyRef != nil {
				cmRefs = append(cmRefs, env.ValueFrom.ConfigMapKeyRef.Name)
			}
			if env.ValueFrom.SecretKeyRef != nil {
				secretRefs = append(secretRefs, env.ValueFrom.SecretKeyRef.Name)
			}
		}
	}

	if len(cmRefs) > 0 {
		raw["configMapRefs"] = dedupStrings(cmRefs)
	}
	if len(secretRefs) > 0 {
		raw["secretRefs"] = dedupStrings(secretRefs)
	}
	return raw
}

// dedupStrings removes duplicate strings from a slice.
func dedupStrings(s []string) []string {
	seen := make(map[string]bool, len(s))
	result := make([]string, 0, len(s))
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
