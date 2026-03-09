package k8s

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

func (c *Client) ListServices(ctx context.Context, namespace string) ([]corev1.Service, error) {
	svcs, err := c.clientset().CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return svcs.Items, nil
}

func (c *Client) ListConfigMaps(ctx context.Context, namespace string) ([]corev1.ConfigMap, error) {
	cms, err := c.clientset().CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return cms.Items, nil
}

func (c *Client) ListSecrets(ctx context.Context, namespace string) ([]corev1.Secret, error) {
	secrets, err := c.clientset().CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return secrets.Items, nil
}

func (c *Client) ListIngresses(ctx context.Context, namespace string) ([]networkingv1.Ingress, error) {
	ings, err := c.clientset().NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return ings.Items, nil
}

func (c *Client) ListEvents(ctx context.Context, namespace string) ([]corev1.Event, error) {
	events, err := c.clientset().CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return events.Items, nil
}

func (c *Client) ListRoles(ctx context.Context, namespace string) ([]rbacv1.Role, error) {
	roles, err := c.clientset().RbacV1().Roles(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return roles.Items, nil
}

func (c *Client) ListRoleBindings(ctx context.Context, namespace string) ([]rbacv1.RoleBinding, error) {
	rb, err := c.clientset().RbacV1().RoleBindings(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return rb.Items, nil
}

func (c *Client) ListClusterRoles(ctx context.Context) ([]rbacv1.ClusterRole, error) {
	roles, err := c.clientset().RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return roles.Items, nil
}

func (c *Client) ListClusterRoleBindings(ctx context.Context) ([]rbacv1.ClusterRoleBinding, error) {
	crb, err := c.clientset().RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return crb.Items, nil
}

func (c *Client) ListPersistentVolumes(ctx context.Context) ([]corev1.PersistentVolume, error) {
	pv, err := c.clientset().CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return pv.Items, nil
}

func (c *Client) ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]corev1.PersistentVolumeClaim, error) {
	pvc, err := c.clientset().CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return pvc.Items, nil
}

func (c *Client) ListStorageClasses(ctx context.Context) ([]storagev1.StorageClass, error) {
	sc, err := c.clientset().StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return sc.Items, nil
}

func (c *Client) ListServiceAccounts(ctx context.Context, namespace string) ([]corev1.ServiceAccount, error) {
	sa, err := c.clientset().CoreV1().ServiceAccounts(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return sa.Items, nil
}

func (c *Client) ListHorizontalPodAutoscalers(ctx context.Context, namespace string) ([]autoscalingv2.HorizontalPodAutoscaler, error) {
	hpas, err := c.clientset().AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return hpas.Items, nil
}

func (c *Client) ListNetworkPolicies(ctx context.Context, namespace string) ([]networkingv1.NetworkPolicy, error) {
	netpols, err := c.clientset().NetworkingV1().NetworkPolicies(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return netpols.Items, nil
}

func (c *Client) ListTable(ctx context.Context, gvr schema.GroupVersionResource, ns string) (*metav1.Table, error) {
	// For now, return error until REST client implementation is ready
	return nil, fmt.Errorf("dynamic table listing not yet implemented")
}

func (c *Client) DeleteResource(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) error {
	if c.Dynamic == nil {
		return fmt.Errorf("dynamic client not initialized")
	}
	return c.dynamicClient().Resource(gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (c *Client) GetResourceYAML(ctx context.Context, namespace, name string, gvr schema.GroupVersionResource) (string, error) {
	if c.Dynamic == nil {
		return "", fmt.Errorf("dynamic client not initialized")
	}
	obj, err := c.dynamicClient().Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	obj.SetManagedFields(nil)
	obj.SetResourceVersion("")

	data, err := yaml.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *Client) GetResourceContext(ctx context.Context, ns, name, resource string) (string, error) {
	gvr, ok := c.GetGVR(resource)
	if !ok {
		return "", fmt.Errorf("unknown resource: %s", resource)
	}

	// 1. Get YAML
	yaml, err := c.GetResourceYAML(ctx, ns, name, gvr)
	if err != nil {
		return "", err
	}

	contextBuilder := strings.Builder{}
	contextBuilder.WriteString("### Resource Manifest (YAML)\n")
	contextBuilder.WriteString("```yaml\n")
	contextBuilder.WriteString(yaml)
	contextBuilder.WriteString("\n```\n\n")

	// 2. Get Related Events
	events, _ := c.ListEvents(ctx, ns)
	contextBuilder.WriteString("### Related Events\n")
	foundEvents := false
	for _, ev := range events {
		if ev.InvolvedObject.Name == name || strings.Contains(ev.Message, name) {
			contextBuilder.WriteString(fmt.Sprintf("- [%s] %s: %s\n", ev.LastTimestamp.Format(time.RFC3339), ev.Reason, ev.Message))
			foundEvents = true
		}
	}
	if !foundEvents {
		contextBuilder.WriteString("No related events found.\n")
	}
	contextBuilder.WriteString("\n")

	// 3. If Pod, get Logs (last 20 lines)
	if resource == "pods" || resource == "po" {
		contextBuilder.WriteString("### Recent Logs (Last 20 lines)\n")
		logs, err := c.GetPodLogs(ctx, ns, name, "", 20)
		if err == nil {
			contextBuilder.WriteString("```\n")
			contextBuilder.WriteString(logs)
			contextBuilder.WriteString("\n```\n")
		} else {
			contextBuilder.WriteString(fmt.Sprintf("Error fetching logs: %v\n", err))
		}
	}

	return contextBuilder.String(), nil
}

// WatchResource starts a Watch on the given resource type.
// Uses typed clientset methods for fake clientset test compatibility.
func (c *Client) WatchResource(ctx context.Context, resource, namespace string) (watch.Interface, error) {
	opts := metav1.ListOptions{}
	switch strings.ToLower(resource) {
	// Core resources
	case "pods":
		return c.clientset().CoreV1().Pods(namespace).Watch(ctx, opts)
	case "services":
		return c.clientset().CoreV1().Services(namespace).Watch(ctx, opts)
	case "nodes":
		return c.clientset().CoreV1().Nodes().Watch(ctx, opts)
	case "namespaces":
		return c.clientset().CoreV1().Namespaces().Watch(ctx, opts)
	case "events":
		return c.clientset().CoreV1().Events(namespace).Watch(ctx, opts)
	case "configmaps":
		return c.clientset().CoreV1().ConfigMaps(namespace).Watch(ctx, opts)
	case "secrets":
		return c.clientset().CoreV1().Secrets(namespace).Watch(ctx, opts)
	case "persistentvolumes":
		return c.clientset().CoreV1().PersistentVolumes().Watch(ctx, opts)
	case "persistentvolumeclaims":
		return c.clientset().CoreV1().PersistentVolumeClaims(namespace).Watch(ctx, opts)
	case "serviceaccounts":
		return c.clientset().CoreV1().ServiceAccounts(namespace).Watch(ctx, opts)
	case "endpoints":
		return c.clientset().CoreV1().Endpoints(namespace).Watch(ctx, opts)
	case "limitranges":
		return c.clientset().CoreV1().LimitRanges(namespace).Watch(ctx, opts)
	case "resourcequotas":
		return c.clientset().CoreV1().ResourceQuotas(namespace).Watch(ctx, opts)
	case "replicationcontrollers":
		return c.clientset().CoreV1().ReplicationControllers(namespace).Watch(ctx, opts)

	// Apps resources
	case "deployments":
		return c.clientset().AppsV1().Deployments(namespace).Watch(ctx, opts)
	case "statefulsets":
		return c.clientset().AppsV1().StatefulSets(namespace).Watch(ctx, opts)
	case "daemonsets":
		return c.clientset().AppsV1().DaemonSets(namespace).Watch(ctx, opts)
	case "replicasets":
		return c.clientset().AppsV1().ReplicaSets(namespace).Watch(ctx, opts)

	// Batch resources
	case "jobs":
		return c.clientset().BatchV1().Jobs(namespace).Watch(ctx, opts)
	case "cronjobs":
		return c.clientset().BatchV1().CronJobs(namespace).Watch(ctx, opts)

	// Networking resources
	case "ingresses":
		return c.clientset().NetworkingV1().Ingresses(namespace).Watch(ctx, opts)
	case "networkpolicies":
		return c.clientset().NetworkingV1().NetworkPolicies(namespace).Watch(ctx, opts)

	// RBAC resources
	case "roles":
		return c.clientset().RbacV1().Roles(namespace).Watch(ctx, opts)
	case "rolebindings":
		return c.clientset().RbacV1().RoleBindings(namespace).Watch(ctx, opts)
	case "clusterroles":
		return c.clientset().RbacV1().ClusterRoles().Watch(ctx, opts)
	case "clusterrolebindings":
		return c.clientset().RbacV1().ClusterRoleBindings().Watch(ctx, opts)

	// Storage resources
	case "storageclasses":
		return c.clientset().StorageV1().StorageClasses().Watch(ctx, opts)

	// Policy resources
	case "poddisruptionbudgets":
		return c.clientset().PolicyV1().PodDisruptionBudgets(namespace).Watch(ctx, opts)

	// Autoscaling resources
	case "horizontalpodautoscalers":
		return c.clientset().AutoscalingV2().HorizontalPodAutoscalers(namespace).Watch(ctx, opts)

	default:
		return nil, fmt.Errorf("watch not supported for resource: %s", resource)
	}
}

// ApplyYAML applies a YAML manifest to the cluster using kubectl-style apply
// It supports dry-run mode for validation without actually applying changes
func (c *Client) ApplyYAML(ctx context.Context, yamlContent string, defaultNamespace string, dryRun bool) (string, error) {
	if c.Dynamic == nil {
		return "", fmt.Errorf("dynamic client not initialized")
	}
	if defaultNamespace == "" {
		defaultNamespace = "default"
	}

	// Parse the YAML to extract basic info
	var obj map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &obj); err != nil {
		return "", fmt.Errorf("invalid YAML: %w", err)
	}

	// Extract required fields
	apiVersion, _ := obj["apiVersion"].(string)
	kind, _ := obj["kind"].(string)
	metadata, _ := obj["metadata"].(map[interface{}]interface{})

	if apiVersion == "" || kind == "" {
		return "", fmt.Errorf("YAML must contain apiVersion and kind")
	}

	if metadata == nil {
		return "", fmt.Errorf("YAML must contain metadata")
	}

	name, _ := metadata["name"].(string)
	if name == "" {
		return "", fmt.Errorf("YAML metadata must contain name")
	}

	namespace, _ := metadata["namespace"].(string)
	if namespace == "" {
		namespace = defaultNamespace
	}

	// Convert to unstructured object for dynamic client
	unstructuredObj := &unstructured.Unstructured{
		Object: convertToStringKeyMap(obj),
	}

	// Determine the GVR (GroupVersionResource) from apiVersion and kind
	gvr, err := c.getGVRForKind(apiVersion, kind)
	if err != nil {
		return "", fmt.Errorf("failed to determine resource type: %w", err)
	}

	// Determine if resource is namespaced
	namespaced := isNamespacedResource(kind)

	// Set dry-run options
	createOpts := metav1.CreateOptions{}
	updateOpts := metav1.UpdateOptions{}
	if dryRun {
		createOpts.DryRun = []string{metav1.DryRunAll}
		updateOpts.DryRun = []string{metav1.DryRunAll}
	}

	// Try to get existing resource first (for apply semantics)
	var resourceClient dynamic.ResourceInterface
	if namespaced {
		resourceClient = c.dynamicClient().Resource(gvr).Namespace(namespace)
	} else {
		resourceClient = c.dynamicClient().Resource(gvr)
	}

	existing, err := resourceClient.Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		// Resource exists - update it
		unstructuredObj.SetResourceVersion(existing.GetResourceVersion())
		_, err = resourceClient.Update(ctx, unstructuredObj, updateOpts)
		if err != nil {
			return "", fmt.Errorf("failed to update %s/%s: %w", kind, name, err)
		}
		action := "updated"
		if dryRun {
			action = "validated (dry-run)"
		}
		return fmt.Sprintf("%s/%s %s", strings.ToLower(kind), name, action), nil
	}

	// Resource doesn't exist - create it
	_, err = resourceClient.Create(ctx, unstructuredObj, createOpts)
	if err != nil {
		return "", fmt.Errorf("failed to create %s/%s: %w", kind, name, err)
	}

	action := "created"
	if dryRun {
		action = "validated (dry-run)"
	}
	return fmt.Sprintf("%s/%s %s", strings.ToLower(kind), name, action), nil
}

// ListDynamicResource lists resources using the dynamic client for any resource type
func (c *Client) ListDynamicResource(ctx context.Context, gvr schema.GroupVersionResource, namespace string) ([]map[string]interface{}, error) {
	if c.Dynamic == nil {
		return nil, fmt.Errorf("dynamic client not initialized")
	}
	var uList *unstructured.UnstructuredList
	var err error

	if namespace == "" {
		uList, err = c.dynamicClient().Resource(gvr).List(ctx, metav1.ListOptions{})
	} else {
		uList, err = c.dynamicClient().Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for _, item := range uList.Items {
		results = append(results, item.Object)
	}
	return results, nil
}

// ListCustomResources lists instances of a custom resource
func (c *Client) ListCustomResources(ctx context.Context, crdInfo *CRDInfo, namespace string) ([]unstructured.Unstructured, error) {
	if c.Dynamic == nil {
		return nil, fmt.Errorf("dynamic client not initialized")
	}
	gvr := schema.GroupVersionResource{
		Group:    crdInfo.Group,
		Version:  crdInfo.Version,
		Resource: crdInfo.Plural,
	}

	var list *unstructured.UnstructuredList
	var err error

	if crdInfo.Namespaced && namespace != "" {
		list, err = c.dynamicClient().Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	} else {
		list, err = c.dynamicClient().Resource(gvr).List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		return nil, err
	}

	return list.Items, nil
}

// GetCustomResource gets a single custom resource instance
func (c *Client) GetCustomResource(ctx context.Context, crdInfo *CRDInfo, namespace, name string) (*unstructured.Unstructured, error) {
	if c.Dynamic == nil {
		return nil, fmt.Errorf("dynamic client not initialized")
	}
	gvr := schema.GroupVersionResource{
		Group:    crdInfo.Group,
		Version:  crdInfo.Version,
		Resource: crdInfo.Plural,
	}

	if crdInfo.Namespaced && namespace != "" {
		return c.dynamicClient().Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	}
	return c.dynamicClient().Resource(gvr).Get(ctx, name, metav1.GetOptions{})
}

// GetCustomResourceYAML returns the YAML representation of a custom resource
func (c *Client) GetCustomResourceYAML(ctx context.Context, crdInfo *CRDInfo, namespace, name string) (string, error) {
	obj, err := c.GetCustomResource(ctx, crdInfo, namespace, name)
	if err != nil {
		return "", err
	}

	yamlBytes, err := yaml.Marshal(obj.Object)
	if err != nil {
		return "", err
	}

	return string(yamlBytes), nil
}

func (c *Client) ListCRDs(ctx context.Context) ([]apiextv1.CustomResourceDefinition, error) {
	if c.Dynamic == nil {
		return nil, fmt.Errorf("dynamic client not initialized")
	}
	gvr := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}
	list, err := c.dynamicClient().Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var crds []apiextv1.CustomResourceDefinition
	for _, item := range list.Items {
		var crd apiextv1.CustomResourceDefinition
		crd.Name = item.GetName()
		crd.CreationTimestamp = item.GetCreationTimestamp()
		crds = append(crds, crd)
	}
	return crds, nil
}

// PrinterColumn represents an additionalPrinterColumn from a CRD spec.
type PrinterColumn struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	JSONPath string `json:"jsonPath"`
	Priority int    `json:"priority"`
}

// CRDInfo contains information about a Custom Resource Definition
type CRDInfo struct {
	Name           string          `json:"name"`
	Group          string          `json:"group"`
	Version        string          `json:"version"`
	Kind           string          `json:"kind"`
	Plural         string          `json:"plural"`
	Namespaced     bool            `json:"namespaced"`
	ShortNames     []string        `json:"shortNames"`
	PrinterColumns []PrinterColumn `json:"printerColumns,omitempty"`
}

// GetCRDInfo returns detailed information about a CRD by name
func (c *Client) GetCRDInfo(ctx context.Context, crdName string) (*CRDInfo, error) {
	if c.Dynamic == nil {
		return nil, fmt.Errorf("dynamic client not initialized")
	}
	gvr := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}

	obj, err := c.dynamicClient().Resource(gvr).Get(ctx, crdName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// Extract spec.group
	group, _, _ := unstructured.NestedString(obj.Object, "spec", "group")

	// Extract spec.names
	names, _, _ := unstructured.NestedMap(obj.Object, "spec", "names")
	kind, _ := names["kind"].(string)
	plural, _ := names["plural"].(string)
	shortNamesRaw, _ := names["shortNames"].([]interface{})
	var shortNames []string
	for _, sn := range shortNamesRaw {
		if s, ok := sn.(string); ok {
			shortNames = append(shortNames, s)
		}
	}

	// Extract scope
	scope, _, _ := unstructured.NestedString(obj.Object, "spec", "scope")
	namespaced := scope == "Namespaced"

	// Extract versions and find the served/storage version
	versions, _, _ := unstructured.NestedSlice(obj.Object, "spec", "versions")
	var version string
	var printerColumns []PrinterColumn
	for _, v := range versions {
		if vMap, ok := v.(map[string]interface{}); ok {
			served, _ := vMap["served"].(bool)
			storage, _ := vMap["storage"].(bool)
			if served && storage {
				version, _ = vMap["name"].(string)
				printerColumns = extractPrinterColumns(vMap)
				break
			}
		}
	}
	// Fallback to first version if no storage version found
	if version == "" && len(versions) > 0 {
		if vMap, ok := versions[0].(map[string]interface{}); ok {
			version, _ = vMap["name"].(string)
			if len(printerColumns) == 0 {
				printerColumns = extractPrinterColumns(vMap)
			}
		}
	}

	return &CRDInfo{
		Name:           crdName,
		Group:          group,
		Version:        version,
		Kind:           kind,
		Plural:         plural,
		Namespaced:     namespaced,
		ShortNames:     shortNames,
		PrinterColumns: printerColumns,
	}, nil
}

// ListCRDsDetailed returns detailed information about all CRDs
func (c *Client) ListCRDsDetailed(ctx context.Context) ([]CRDInfo, error) {
	if c.Dynamic == nil {
		return nil, fmt.Errorf("dynamic client not initialized")
	}
	gvr := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}

	list, err := c.dynamicClient().Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var crds []CRDInfo
	for _, item := range list.Items {
		info, err := c.parseCRDItem(&item)
		if err != nil {
			continue
		}
		crds = append(crds, *info)
	}
	return crds, nil
}

// parseCRDItem extracts CRDInfo from an unstructured CRD object
func (c *Client) parseCRDItem(obj *unstructured.Unstructured) (*CRDInfo, error) {
	group, _, _ := unstructured.NestedString(obj.Object, "spec", "group")
	names, _, _ := unstructured.NestedMap(obj.Object, "spec", "names")
	kind, _ := names["kind"].(string)
	plural, _ := names["plural"].(string)
	shortNamesRaw, _ := names["shortNames"].([]interface{})
	var shortNames []string
	for _, sn := range shortNamesRaw {
		if s, ok := sn.(string); ok {
			shortNames = append(shortNames, s)
		}
	}

	scope, _, _ := unstructured.NestedString(obj.Object, "spec", "scope")
	namespaced := scope == "Namespaced"

	versions, _, _ := unstructured.NestedSlice(obj.Object, "spec", "versions")
	var version string
	var printerColumns []PrinterColumn
	for _, v := range versions {
		if vMap, ok := v.(map[string]interface{}); ok {
			served, _ := vMap["served"].(bool)
			storage, _ := vMap["storage"].(bool)
			if served && storage {
				version, _ = vMap["name"].(string)
				printerColumns = extractPrinterColumns(vMap)
				break
			}
		}
	}
	if version == "" && len(versions) > 0 {
		if vMap, ok := versions[0].(map[string]interface{}); ok {
			version, _ = vMap["name"].(string)
			if len(printerColumns) == 0 {
				printerColumns = extractPrinterColumns(vMap)
			}
		}
	}

	return &CRDInfo{
		Name:           obj.GetName(),
		Group:          group,
		Version:        version,
		Kind:           kind,
		Plural:         plural,
		Namespaced:     namespaced,
		ShortNames:     shortNames,
		PrinterColumns: printerColumns,
	}, nil
}

// extractPrinterColumns parses additionalPrinterColumns from a CRD version map.
func extractPrinterColumns(versionMap map[string]interface{}) []PrinterColumn {
	cols, ok := versionMap["additionalPrinterColumns"].([]interface{})
	if !ok || len(cols) == 0 {
		return nil
	}
	var result []PrinterColumn
	for _, col := range cols {
		colMap, ok := col.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := colMap["name"].(string)
		colType, _ := colMap["type"].(string)
		jsonPath, _ := colMap["jsonPath"].(string)
		priority, _ := colMap["priority"].(int64)
		if name != "" && jsonPath != "" {
			result = append(result, PrinterColumn{
				Name:     name,
				Type:     colType,
				JSONPath: jsonPath,
				Priority: int(priority),
			})
		}
	}
	return result
}

// ResolveJSONPath extracts a value from an unstructured map using a simplified JSONPath.
// Supports: .field.subfield, .field[index], .field[?(@.key=="value")].resultField
func ResolveJSONPath(obj map[string]interface{}, jsonPath string) string {
	path := strings.TrimPrefix(jsonPath, ".")
	if path == "" {
		return ""
	}
	val := resolvePathRecursive(obj, path)
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case bool:
		if v {
			return "True"
		}
		return "False"
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%.2f", v)
	case int64:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func resolvePathRecursive(current interface{}, path string) interface{} {
	if path == "" || current == nil {
		return current
	}
	m, ok := current.(map[string]interface{})
	if !ok {
		return nil
	}

	// Check for array bracket in current segment
	dotIdx := strings.Index(path, ".")
	bracketIdx := strings.Index(path, "[")

	// Simple field (no dot, no bracket)
	if dotIdx < 0 && bracketIdx < 0 {
		return m[path]
	}

	// Array bracket comes before dot (or no dot)
	if bracketIdx >= 0 && (dotIdx < 0 || bracketIdx < dotIdx) {
		fieldName := path[:bracketIdx]
		rest := path[bracketIdx:]

		arr, ok := m[fieldName].([]interface{})
		if !ok {
			return nil
		}

		bracketEnd := strings.Index(rest, "]")
		if bracketEnd < 0 {
			return nil
		}

		bracketContent := rest[1:bracketEnd]
		remaining := ""
		if bracketEnd+1 < len(rest) {
			remaining = strings.TrimPrefix(rest[bracketEnd+1:], ".")
		}

		// Array filter: ?(@.key=="value")
		if strings.HasPrefix(bracketContent, "?(@.") {
			expr := strings.TrimPrefix(bracketContent, "?(@.")
			expr = strings.TrimSuffix(expr, ")")
			parts := strings.SplitN(expr, "==", 2)
			if len(parts) == 2 {
				key := parts[0]
				value := strings.Trim(parts[1], "\"'")
				for _, elem := range arr {
					elemMap, ok := elem.(map[string]interface{})
					if !ok {
						continue
					}
					if fmt.Sprintf("%v", elemMap[key]) == value {
						if remaining == "" {
							return elem
						}
						return resolvePathRecursive(elem, remaining)
					}
				}
			}
			return nil
		}

		// Numeric index
		idx, err := strconv.Atoi(bracketContent)
		if err == nil && idx >= 0 && idx < len(arr) {
			if remaining == "" {
				return arr[idx]
			}
			return resolvePathRecursive(arr[idx], remaining)
		}
		return nil
	}

	// Dot-separated path
	fieldName := path[:dotIdx]
	rest := path[dotIdx+1:]
	return resolvePathRecursive(m[fieldName], rest)
}

func (c *Client) ListReplicationControllers(ctx context.Context, namespace string) ([]corev1.ReplicationController, error) {
	opts := metav1.ListOptions{}
	if namespace == "" {
		list, err := c.clientset().CoreV1().ReplicationControllers("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return list.Items, nil
	}
	list, err := c.clientset().CoreV1().ReplicationControllers(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) ListEndpoints(ctx context.Context, namespace string) ([]corev1.Endpoints, error) { //nolint:staticcheck
	opts := metav1.ListOptions{}
	if namespace == "" {
		list, err := c.clientset().CoreV1().Endpoints("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return list.Items, nil
	}
	list, err := c.clientset().CoreV1().Endpoints(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) ListPodDisruptionBudgets(ctx context.Context, namespace string) ([]policyv1.PodDisruptionBudget, error) {
	opts := metav1.ListOptions{}
	if namespace == "" {
		list, err := c.clientset().PolicyV1().PodDisruptionBudgets("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return list.Items, nil
	}
	list, err := c.clientset().PolicyV1().PodDisruptionBudgets(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) ListLimitRanges(ctx context.Context, namespace string) ([]corev1.LimitRange, error) {
	opts := metav1.ListOptions{}
	if namespace == "" {
		list, err := c.clientset().CoreV1().LimitRanges("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return list.Items, nil
	}
	list, err := c.clientset().CoreV1().LimitRanges(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) ListResourceQuotas(ctx context.Context, namespace string) ([]corev1.ResourceQuota, error) {
	opts := metav1.ListOptions{}
	if namespace == "" {
		list, err := c.clientset().CoreV1().ResourceQuotas("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return list.Items, nil
	}
	list, err := c.clientset().CoreV1().ResourceQuotas(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) ListHPAs(ctx context.Context, namespace string) ([]autoscalingv2.HorizontalPodAutoscaler, error) {
	opts := metav1.ListOptions{}
	if namespace == "" {
		list, err := c.clientset().AutoscalingV2().HorizontalPodAutoscalers("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return list.Items, nil
	}
	list, err := c.clientset().AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// DescribeResource returns kubectl describe-like output for a resource
func (c *Client) DescribeResource(ctx context.Context, resource, namespace, name string) (string, error) {
	var result strings.Builder

	switch resource {
	case "pods":
		pod, err := c.clientset().CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		result.WriteString(fmt.Sprintf("Name:         %s\n", pod.Name))
		result.WriteString(fmt.Sprintf("Namespace:    %s\n", pod.Namespace))
		result.WriteString(fmt.Sprintf("Node:         %s\n", pod.Spec.NodeName))
		result.WriteString(fmt.Sprintf("Status:       %s\n", pod.Status.Phase))
		result.WriteString(fmt.Sprintf("IP:           %s\n", pod.Status.PodIP))
		result.WriteString(fmt.Sprintf("Created:      %s\n", pod.CreationTimestamp.Format(time.RFC3339)))
		result.WriteString("\nLabels:\n")
		for k, v := range pod.Labels {
			result.WriteString(fmt.Sprintf("  %s=%s\n", k, v))
		}
		result.WriteString("\nContainers:\n")
		for _, c := range pod.Spec.Containers {
			result.WriteString(fmt.Sprintf("  %s:\n", c.Name))
			result.WriteString(fmt.Sprintf("    Image:   %s\n", c.Image))
			result.WriteString("    Ports:   ")
			var ports []string
			for _, p := range c.Ports {
				ports = append(ports, fmt.Sprintf("%d/%s", p.ContainerPort, p.Protocol))
			}
			result.WriteString(strings.Join(ports, ", ") + "\n")
		}
		result.WriteString("\nConditions:\n")
		for _, cond := range pod.Status.Conditions {
			result.WriteString(fmt.Sprintf("  Type: %s, Status: %s\n", cond.Type, cond.Status))
		}
		result.WriteString("\nEvents:\n")
		events, _ := c.clientset().CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Pod", name),
		})
		if events != nil && len(events.Items) > 0 {
			for _, e := range events.Items {
				result.WriteString(fmt.Sprintf("  %s  %s  %s\n", e.LastTimestamp.Format("15:04:05"), e.Reason, e.Message))
			}
		} else {
			result.WriteString("  <none>\n")
		}

	case "deployments":
		dep, err := c.clientset().AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		replicas := int32(1)
		if dep.Spec.Replicas != nil {
			replicas = *dep.Spec.Replicas
		}
		result.WriteString(fmt.Sprintf("Name:         %s\n", dep.Name))
		result.WriteString(fmt.Sprintf("Namespace:    %s\n", dep.Namespace))
		result.WriteString(fmt.Sprintf("Replicas:     %d desired | %d updated | %d total | %d available | %d unavailable\n",
			replicas, dep.Status.UpdatedReplicas, dep.Status.Replicas, dep.Status.AvailableReplicas, dep.Status.UnavailableReplicas))
		result.WriteString(fmt.Sprintf("Strategy:     %s\n", dep.Spec.Strategy.Type))
		result.WriteString(fmt.Sprintf("Created:      %s\n", dep.CreationTimestamp.Format(time.RFC3339)))
		result.WriteString("\nLabels:\n")
		for k, v := range dep.Labels {
			result.WriteString(fmt.Sprintf("  %s=%s\n", k, v))
		}
		result.WriteString("\nSelector:\n")
		for k, v := range dep.Spec.Selector.MatchLabels {
			result.WriteString(fmt.Sprintf("  %s=%s\n", k, v))
		}
		result.WriteString("\nPod Template:\n")
		for _, c := range dep.Spec.Template.Spec.Containers {
			result.WriteString(fmt.Sprintf("  Container: %s\n", c.Name))
			result.WriteString(fmt.Sprintf("    Image:   %s\n", c.Image))
		}
		result.WriteString("\nConditions:\n")
		for _, cond := range dep.Status.Conditions {
			result.WriteString(fmt.Sprintf("  Type: %s, Status: %s, Reason: %s\n", cond.Type, cond.Status, cond.Reason))
		}

	case "services":
		svc, err := c.clientset().CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		result.WriteString(fmt.Sprintf("Name:         %s\n", svc.Name))
		result.WriteString(fmt.Sprintf("Namespace:    %s\n", svc.Namespace))
		result.WriteString(fmt.Sprintf("Type:         %s\n", svc.Spec.Type))
		result.WriteString(fmt.Sprintf("ClusterIP:    %s\n", svc.Spec.ClusterIP))
		result.WriteString(fmt.Sprintf("Created:      %s\n", svc.CreationTimestamp.Format(time.RFC3339)))
		result.WriteString("\nLabels:\n")
		for k, v := range svc.Labels {
			result.WriteString(fmt.Sprintf("  %s=%s\n", k, v))
		}
		result.WriteString("\nSelector:\n")
		for k, v := range svc.Spec.Selector {
			result.WriteString(fmt.Sprintf("  %s=%s\n", k, v))
		}
		result.WriteString("\nPorts:\n")
		for _, p := range svc.Spec.Ports {
			result.WriteString(fmt.Sprintf("  %s %d/%s -> %d\n", p.Name, p.Port, p.Protocol, p.TargetPort.IntVal))
		}

	case "nodes":
		node, err := c.clientset().CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		result.WriteString(fmt.Sprintf("Name:         %s\n", node.Name))
		result.WriteString(fmt.Sprintf("Created:      %s\n", node.CreationTimestamp.Format(time.RFC3339)))
		result.WriteString("\nLabels:\n")
		for k, v := range node.Labels {
			result.WriteString(fmt.Sprintf("  %s=%s\n", k, v))
		}
		result.WriteString("\nConditions:\n")
		for _, cond := range node.Status.Conditions {
			result.WriteString(fmt.Sprintf("  Type: %s, Status: %s\n", cond.Type, cond.Status))
		}
		result.WriteString("\nCapacity:\n")
		for k, v := range node.Status.Capacity {
			result.WriteString(fmt.Sprintf("  %s: %s\n", k, v.String()))
		}
		result.WriteString("\nAllocatable:\n")
		for k, v := range node.Status.Allocatable {
			result.WriteString(fmt.Sprintf("  %s: %s\n", k, v.String()))
		}
		result.WriteString("\nSystem Info:\n")
		result.WriteString(fmt.Sprintf("  OS Image:             %s\n", node.Status.NodeInfo.OSImage))
		result.WriteString(fmt.Sprintf("  Kernel Version:       %s\n", node.Status.NodeInfo.KernelVersion))
		result.WriteString(fmt.Sprintf("  Container Runtime:    %s\n", node.Status.NodeInfo.ContainerRuntimeVersion))
		result.WriteString(fmt.Sprintf("  Kubelet Version:      %s\n", node.Status.NodeInfo.KubeletVersion))

	default:
		// Generic describe using dynamic client
		if c.Dynamic == nil {
			return "", fmt.Errorf("dynamic client not initialized")
		}
		gvr, err := c.getGVRForResource(resource)
		if err != nil {
			return "", err
		}
		var obj interface{}
		if namespace != "" {
			unstructured, err := c.dynamicClient().Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return "", err
			}
			obj = unstructured.Object
		} else {
			unstructured, err := c.dynamicClient().Resource(gvr).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return "", err
			}
			obj = unstructured.Object
		}
		data, err := yaml.Marshal(obj)
		if err != nil {
			return "", err
		}
		result.WriteString(string(data))
	}

	return result.String(), nil
}
