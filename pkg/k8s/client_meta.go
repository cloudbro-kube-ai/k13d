package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
)

func (c *Client) GetContextInfo() (ctxName, cluster, user string, err error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	rawConfig, err := kubeConfig.RawConfig()
	if err != nil {
		return "", "", "", err
	}

	ctxName = rawConfig.CurrentContext
	if ctx, ok := rawConfig.Contexts[ctxName]; ok {
		cluster = ctx.Cluster
		user = ctx.AuthInfo
	}

	return ctxName, cluster, user, nil
}

func (c *Client) GetCurrentContext() (string, error) {
	ctx, _, _, err := c.GetContextInfo()
	return ctx, err
}

func (c *Client) GetCurrentNamespace() string {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	ns, _, _ := kubeConfig.Namespace()
	return ns
}

func (c *Client) GetServerVersion() (string, error) {
	version, err := c.Clientset.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	return version.GitVersion, nil
}

func (c *Client) ListNamespaces(ctx context.Context) ([]corev1.Namespace, error) {
	log.Debugf("ListNamespaces: ENTER")

	// Create a context with timeout if not already set
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ns, err := c.Clientset.CoreV1().Namespaces().List(ctxWithTimeout, metav1.ListOptions{})
	if err != nil {
		log.Errorf("ListNamespaces: ERROR: %v", err)
		return nil, err
	}

	log.Infof("ListNamespaces: SUCCESS (found %d)", len(ns.Items))
	return ns.Items, nil
}

func (c *Client) ListNodes(ctx context.Context) ([]corev1.Node, error) {
	nodes, err := c.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return nodes.Items, nil
}

func (c *Client) ListContexts() ([]string, string, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	config, err := loadingRules.Load()
	if err != nil {
		return nil, "", err
	}
	var contexts []string
	for name := range config.Contexts {
		contexts = append(contexts, name)
	}
	return contexts, config.CurrentContext, nil
}

// DrainNode cordons and drains a node
func (c *Client) DrainNode(ctx context.Context, nodeName string, gracePeriod int64) error {
	// First, cordon the node
	if err := c.CordonNode(ctx, nodeName); err != nil {
		return fmt.Errorf("failed to cordon node: %w", err)
	}

	// Get all pods on the node
	pods, err := c.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})
	if err != nil {
		return fmt.Errorf("failed to list pods on node: %w", err)
	}

	// Evict each pod (skip DaemonSet pods and mirror pods)
	for _, pod := range pods.Items {
		// Skip DaemonSet pods
		if pod.OwnerReferences != nil {
			isDaemonSet := false
			for _, ref := range pod.OwnerReferences {
				if ref.Kind == "DaemonSet" {
					isDaemonSet = true
					break
				}
			}
			if isDaemonSet {
				continue
			}
		}

		// Skip mirror pods
		if _, ok := pod.Annotations["kubernetes.io/config.mirror"]; ok {
			continue
		}

		// Delete pod with grace period
		deleteOptions := metav1.DeleteOptions{}
		if gracePeriod > 0 {
			deleteOptions.GracePeriodSeconds = &gracePeriod
		}
		err := c.Clientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, deleteOptions)
		if err != nil {
			log.Errorf("Failed to delete pod %s/%s: %v", pod.Namespace, pod.Name, err)
		}
	}

	return nil
}

// CordonNode marks a node as unschedulable
func (c *Client) CordonNode(ctx context.Context, nodeName string) error {
	payload := []byte(`{"spec":{"unschedulable":true}}`)
	_, err := c.Clientset.CoreV1().Nodes().Patch(ctx, nodeName, types.MergePatchType, payload, metav1.PatchOptions{})
	return err
}

// UncordonNode marks a node as schedulable
func (c *Client) UncordonNode(ctx context.Context, nodeName string) error {
	payload := []byte(`{"spec":{"unschedulable":false}}`)
	_, err := c.Clientset.CoreV1().Nodes().Patch(ctx, nodeName, types.MergePatchType, payload, metav1.PatchOptions{})
	return err
}

func (c *Client) GetGVR(resource string) (schema.GroupVersionResource, bool) {
	m := map[string]schema.GroupVersionResource{
		"pods":                     {Group: "", Version: "v1", Resource: "pods"},
		"po":                       {Group: "", Version: "v1", Resource: "pods"},
		"services":                 {Group: "", Version: "v1", Resource: "services"},
		"svc":                      {Group: "", Version: "v1", Resource: "services"},
		"nodes":                    {Group: "", Version: "v1", Resource: "nodes"},
		"no":                       {Group: "", Version: "v1", Resource: "nodes"},
		"namespaces":               {Group: "", Version: "v1", Resource: "namespaces"},
		"ns":                       {Group: "", Version: "v1", Resource: "namespaces"},
		"deploy":                   {Group: "apps", Version: "v1", Resource: "deployments"},
		"deployments":              {Group: "apps", Version: "v1", Resource: "deployments"},
		"statefulsets":             {Group: "apps", Version: "v1", Resource: "statefulsets"},
		"sts":                      {Group: "apps", Version: "v1", Resource: "statefulsets"},
		"daemonsets":               {Group: "apps", Version: "v1", Resource: "daemonsets"},
		"ds":                       {Group: "apps", Version: "v1", Resource: "daemonsets"},
		"replicasets":              {Group: "apps", Version: "v1", Resource: "replicasets"},
		"rs":                       {Group: "apps", Version: "v1", Resource: "replicasets"},
		"jobs":                     {Group: "batch", Version: "v1", Resource: "jobs"},
		"cronjobs":                 {Group: "batch", Version: "v1", Resource: "cronjobs"},
		"cj":                       {Group: "batch", Version: "v1", Resource: "cronjobs"},
		"configmaps":               {Group: "", Version: "v1", Resource: "configmaps"},
		"cm":                       {Group: "", Version: "v1", Resource: "configmaps"},
		"secrets":                  {Group: "", Version: "v1", Resource: "secrets"},
		"ingresses":                {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
		"ing":                      {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
		"roles":                    {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles"},
		"rolebindings":             {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"},
		"rb":                       {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"},
		"clusterroles":             {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"},
		"clusterrolebindings":      {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"},
		"crb":                      {Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"},
		"persistentvolumes":        {Group: "", Version: "v1", Resource: "persistentvolumes"},
		"pv":                       {Group: "", Version: "v1", Resource: "persistentvolumes"},
		"persistentvolumeclaims":   {Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
		"pvc":                      {Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
		"storageclasses":           {Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"},
		"sc":                       {Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"},
		"serviceaccounts":          {Group: "", Version: "v1", Resource: "serviceaccounts"},
		"sa":                       {Group: "", Version: "v1", Resource: "serviceaccounts"},
		"horizontalpodautoscalers": {Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"},
		"hpa":                      {Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"},
		"networkpolicies":          {Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"},
		"netpol":                   {Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"},
	}
	gvr, ok := m[strings.ToLower(resource)]
	return gvr, ok
}

// getGVRForResource maps resource names to GroupVersionResource.
// Delegates to GetGVR to avoid duplicate GVR lookup tables.
func (c *Client) getGVRForResource(resource string) (schema.GroupVersionResource, error) {
	gvr, ok := c.GetGVR(resource)
	if ok {
		return gvr, nil
	}

	// Additional resources not in the main GetGVR map
	extra := map[string]schema.GroupVersionResource{
		"poddisruptionbudgets":      {Group: "policy", Version: "v1", Resource: "poddisruptionbudgets"},
		"customresourcedefinitions": {Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"},
	}
	if gvr, ok := extra[resource]; ok {
		return gvr, nil
	}
	return schema.GroupVersionResource{}, fmt.Errorf("unknown resource: %s", resource)
}

// getGVRForKind returns the GroupVersionResource for a given apiVersion and kind
func (c *Client) getGVRForKind(apiVersion, kind string) (schema.GroupVersionResource, error) {
	// Parse apiVersion into group and version
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}

	// Map common kinds to resources
	resourceMap := map[string]string{
		"Pod":                     "pods",
		"Service":                 "services",
		"Deployment":              "deployments",
		"StatefulSet":             "statefulsets",
		"DaemonSet":               "daemonsets",
		"ReplicaSet":              "replicasets",
		"Job":                     "jobs",
		"CronJob":                 "cronjobs",
		"ConfigMap":               "configmaps",
		"Secret":                  "secrets",
		"Ingress":                 "ingresses",
		"ServiceAccount":          "serviceaccounts",
		"Role":                    "roles",
		"RoleBinding":             "rolebindings",
		"ClusterRole":             "clusterroles",
		"ClusterRoleBinding":      "clusterrolebindings",
		"PersistentVolume":        "persistentvolumes",
		"PersistentVolumeClaim":   "persistentvolumeclaims",
		"Namespace":               "namespaces",
		"Node":                    "nodes",
		"Event":                   "events",
		"Endpoints":               "endpoints",
		"LimitRange":              "limitranges",
		"ResourceQuota":           "resourcequotas",
		"HorizontalPodAutoscaler": "horizontalpodautoscalers",
		"PodDisruptionBudget":     "poddisruptionbudgets",
		"NetworkPolicy":           "networkpolicies",
		"StorageClass":            "storageclasses",
	}

	resource, ok := resourceMap[kind]
	if !ok {
		// Handle common irregular plurals before falling back to simple "s" suffix
		lowerKind := strings.ToLower(kind)
		irregularPlurals := map[string]string{
			"ingress":       "ingresses",
			"networkpolicy": "networkpolicies",
			"endpoints":     "endpoints",
		}
		if plural, found := irregularPlurals[lowerKind]; found {
			resource = plural
		} else {
			resource = lowerKind + "s"
		}
	}

	return schema.GroupVersionResource{
		Group:    gv.Group,
		Version:  gv.Version,
		Resource: resource,
	}, nil
}

// isNamespacedResource returns true if the resource kind is namespaced
func isNamespacedResource(kind string) bool {
	clusterScopedKinds := map[string]bool{
		"Namespace":                true,
		"Node":                     true,
		"PersistentVolume":         true,
		"ClusterRole":              true,
		"ClusterRoleBinding":       true,
		"StorageClass":             true,
		"PriorityClass":            true,
		"VolumeAttachment":         true,
		"CSIDriver":                true,
		"CSINode":                  true,
		"CustomResourceDefinition": true,
	}
	return !clusterScopedKinds[kind]
}

// APIResource represents a Kubernetes API resource with metadata
type APIResource struct {
	Name       string   // Resource name (e.g., "pods", "deployments")
	ShortNames []string // Short names (e.g., "po", "deploy")
	Kind       string   // Kind (e.g., "Pod", "Deployment")
	Group      string   // API group (e.g., "", "apps")
	Version    string   // API version (e.g., "v1")
	Namespaced bool     // Whether resource is namespaced
	Verbs      []string // Supported verbs
}

// GetAPIResources returns all available API resources from the cluster
func (c *Client) GetAPIResources(ctx context.Context) ([]APIResource, error) {
	// Use discovery client to get server resources
	_, resourceLists, err := c.Clientset.Discovery().ServerGroupsAndResources()
	if err != nil {
		// Partial failure is OK - some resources might not be accessible
		log.Warnf("Partial error getting API resources: %v", err)
	}

	var resources []APIResource
	seen := make(map[string]bool) // Deduplicate by name

	for _, resourceList := range resourceLists {
		// Parse group/version from GroupVersion string (e.g., "apps/v1", "v1")
		gv := resourceList.GroupVersion
		group := ""
		version := gv
		if idx := strings.Index(gv, "/"); idx > 0 {
			group = gv[:idx]
			version = gv[idx+1:]
		}

		for _, r := range resourceList.APIResources {
			// Skip subresources (e.g., "pods/log", "deployments/scale")
			if strings.Contains(r.Name, "/") {
				continue
			}

			// Skip if already seen (prefer core/apps versions)
			key := r.Name
			if seen[key] {
				continue
			}
			seen[key] = true

			resources = append(resources, APIResource{
				Name:       r.Name,
				ShortNames: r.ShortNames,
				Kind:       r.Kind,
				Group:      group,
				Version:    version,
				Namespaced: r.Namespaced,
				Verbs:      r.Verbs,
			})
		}
	}

	return resources, nil
}

// GetCommonResources returns commonly used resources for quick access (k9s style)
func (c *Client) GetCommonResources() []APIResource {
	return []APIResource{
		{Name: "pods", ShortNames: []string{"po"}, Kind: "Pod", Group: "", Version: "v1", Namespaced: true},
		{Name: "deployments", ShortNames: []string{"deploy"}, Kind: "Deployment", Group: "apps", Version: "v1", Namespaced: true},
		{Name: "services", ShortNames: []string{"svc"}, Kind: "Service", Group: "", Version: "v1", Namespaced: true},
		{Name: "nodes", ShortNames: []string{"no"}, Kind: "Node", Group: "", Version: "v1", Namespaced: false},
		{Name: "namespaces", ShortNames: []string{"ns"}, Kind: "Namespace", Group: "", Version: "v1", Namespaced: false},
		{Name: "events", ShortNames: []string{"ev"}, Kind: "Event", Group: "", Version: "v1", Namespaced: true},
		{Name: "configmaps", ShortNames: []string{"cm"}, Kind: "ConfigMap", Group: "", Version: "v1", Namespaced: true},
		{Name: "secrets", ShortNames: []string{}, Kind: "Secret", Group: "", Version: "v1", Namespaced: true},
		{Name: "ingresses", ShortNames: []string{"ing"}, Kind: "Ingress", Group: "networking.k8s.io", Version: "v1", Namespaced: true},
		{Name: "persistentvolumeclaims", ShortNames: []string{"pvc"}, Kind: "PersistentVolumeClaim", Group: "", Version: "v1", Namespaced: true},
		{Name: "statefulsets", ShortNames: []string{"sts"}, Kind: "StatefulSet", Group: "apps", Version: "v1", Namespaced: true},
		{Name: "daemonsets", ShortNames: []string{"ds"}, Kind: "DaemonSet", Group: "apps", Version: "v1", Namespaced: true},
		{Name: "replicasets", ShortNames: []string{"rs"}, Kind: "ReplicaSet", Group: "apps", Version: "v1", Namespaced: true},
		{Name: "jobs", ShortNames: []string{}, Kind: "Job", Group: "batch", Version: "v1", Namespaced: true},
		{Name: "cronjobs", ShortNames: []string{"cj"}, Kind: "CronJob", Group: "batch", Version: "v1", Namespaced: true},
		{Name: "serviceaccounts", ShortNames: []string{"sa"}, Kind: "ServiceAccount", Group: "", Version: "v1", Namespaced: true},
		{Name: "roles", ShortNames: []string{}, Kind: "Role", Group: "rbac.authorization.k8s.io", Version: "v1", Namespaced: true},
		{Name: "rolebindings", ShortNames: []string{"rb"}, Kind: "RoleBinding", Group: "rbac.authorization.k8s.io", Version: "v1", Namespaced: true},
		{Name: "clusterroles", ShortNames: []string{}, Kind: "ClusterRole", Group: "rbac.authorization.k8s.io", Version: "v1", Namespaced: false},
		{Name: "clusterrolebindings", ShortNames: []string{"crb"}, Kind: "ClusterRoleBinding", Group: "rbac.authorization.k8s.io", Version: "v1", Namespaced: false},
		{Name: "persistentvolumes", ShortNames: []string{"pv"}, Kind: "PersistentVolume", Group: "", Version: "v1", Namespaced: false},
		{Name: "storageclasses", ShortNames: []string{"sc"}, Kind: "StorageClass", Group: "storage.k8s.io", Version: "v1", Namespaced: false},
		{Name: "networkpolicies", ShortNames: []string{"netpol"}, Kind: "NetworkPolicy", Group: "networking.k8s.io", Version: "v1", Namespaced: true},
		{Name: "horizontalpodautoscalers", ShortNames: []string{"hpa"}, Kind: "HorizontalPodAutoscaler", Group: "autoscaling", Version: "v2", Namespaced: true},
		{Name: "poddisruptionbudgets", ShortNames: []string{"pdb"}, Kind: "PodDisruptionBudget", Group: "policy", Version: "v1", Namespaced: true},
		{Name: "customresourcedefinitions", ShortNames: []string{"crd", "crds"}, Kind: "CustomResourceDefinition", Group: "apiextensions.k8s.io", Version: "v1", Namespaced: false},
	}
}
