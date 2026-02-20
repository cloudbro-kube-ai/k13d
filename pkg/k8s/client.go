package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/log"
	"gopkg.in/yaml.v2" // TODO: Migrate to yaml.v3 for consistency with pkg/config (uses v3)
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
)

type Client struct {
	mu        sync.RWMutex
	Clientset kubernetes.Interface
	Dynamic   dynamic.Interface
	Config    *rest.Config
	Metrics   *metricsv1beta1.MetricsV1beta1Client
}

func NewClient() (*Client, error) {
	// Try in-cluster config first (for running inside Kubernetes)
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig (for running outside Kubernetes)
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

		config, err = kubeConfig.ClientConfig()
		if err != nil {
			return nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	metricsClient, err := metricsv1beta1.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		Clientset: clientset,
		Dynamic:   dynamicClient,
		Config:    config,
		Metrics:   metricsClient,
	}, nil
}

func (c *Client) SwitchContext(contextName string) error {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{CurrentContext: contextName}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	metricsClient, err := metricsv1beta1.NewForConfig(config)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.Clientset = clientset
	c.Dynamic = dynamicClient
	c.Config = config
	c.Metrics = metricsClient
	c.mu.Unlock()
	return nil
}

func (c *Client) ListPods(ctx context.Context, namespace string) ([]corev1.Pod, error) {
	log.Debugf("ListPods: ENTER (namespace: %s)", namespace)

	// Create a context with timeout if not already set
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.Infof("ListPods: calling c.Clientset.CoreV1().Pods(%s).List", namespace)
	pods, err := c.Clientset.CoreV1().Pods(namespace).List(ctxWithTimeout, metav1.ListOptions{})
	if err != nil {
		log.Errorf("ListPods: ERROR: %v", err)
		return nil, err
	}

	log.Infof("ListPods: SUCCESS (found %d)", len(pods.Items))
	return pods.Items, nil
}

func (c *Client) ListNodes(ctx context.Context) ([]corev1.Node, error) {
	nodes, err := c.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return nodes.Items, nil
}

func (c *Client) ListDeployments(ctx context.Context, namespace string) ([]appsv1.Deployment, error) {
	deps, err := c.Clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return deps.Items, nil
}

func (c *Client) ListStatefulSets(ctx context.Context, namespace string) ([]appsv1.StatefulSet, error) {
	stses, err := c.Clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return stses.Items, nil
}

func (c *Client) ListServices(ctx context.Context, namespace string) ([]corev1.Service, error) {
	svcs, err := c.Clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return svcs.Items, nil
}

func (c *Client) ListConfigMaps(ctx context.Context, namespace string) ([]corev1.ConfigMap, error) {
	cms, err := c.Clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return cms.Items, nil
}

func (c *Client) ListSecrets(ctx context.Context, namespace string) ([]corev1.Secret, error) {
	secrets, err := c.Clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return secrets.Items, nil
}

func (c *Client) ListIngresses(ctx context.Context, namespace string) ([]networkingv1.Ingress, error) {
	ings, err := c.Clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return ings.Items, nil
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

func (c *Client) GetPodLogs(ctx context.Context, namespace, name, container string, tailLines int64) (string, error) {
	opts := &corev1.PodLogOptions{
		Container: container,
	}
	if tailLines > 0 {
		opts.TailLines = &tailLines
	}
	req := c.Clientset.CoreV1().Pods(namespace).GetLogs(name, opts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer podLogs.Close()

	buf := new(strings.Builder)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
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

func (c *Client) ListEvents(ctx context.Context, namespace string) ([]corev1.Event, error) {
	events, err := c.Clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return events.Items, nil
}

func (c *Client) GetPodLogsStream(ctx context.Context, namespace, name string) (io.ReadCloser, error) {
	req := c.Clientset.CoreV1().Pods(namespace).GetLogs(name, &corev1.PodLogOptions{Follow: true})
	return req.Stream(ctx)
}

func (c *Client) ListTable(ctx context.Context, gvr schema.GroupVersionResource, ns string) (*metav1.Table, error) {
	// For now, return error until REST client implementation is ready
	return nil, fmt.Errorf("dynamic table listing not yet implemented")
}

func (c *Client) ListRoles(ctx context.Context, namespace string) ([]rbacv1.Role, error) {
	roles, err := c.Clientset.RbacV1().Roles(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return roles.Items, nil
}

func (c *Client) ListRoleBindings(ctx context.Context, namespace string) ([]rbacv1.RoleBinding, error) {
	rb, err := c.Clientset.RbacV1().RoleBindings(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return rb.Items, nil
}

func (c *Client) ListClusterRoles(ctx context.Context) ([]rbacv1.ClusterRole, error) {
	roles, err := c.Clientset.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return roles.Items, nil
}

func (c *Client) ListClusterRoleBindings(ctx context.Context) ([]rbacv1.ClusterRoleBinding, error) {
	crb, err := c.Clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return crb.Items, nil
}

func (c *Client) ListPersistentVolumes(ctx context.Context) ([]corev1.PersistentVolume, error) {
	pv, err := c.Clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return pv.Items, nil
}

func (c *Client) ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]corev1.PersistentVolumeClaim, error) {
	pvc, err := c.Clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return pvc.Items, nil
}

func (c *Client) ListStorageClasses(ctx context.Context) ([]storagev1.StorageClass, error) {
	sc, err := c.Clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return sc.Items, nil
}

func (c *Client) ListServiceAccounts(ctx context.Context, namespace string) ([]corev1.ServiceAccount, error) {
	sa, err := c.Clientset.CoreV1().ServiceAccounts(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return sa.Items, nil
}

func (c *Client) ListDaemonSets(ctx context.Context, namespace string) ([]appsv1.DaemonSet, error) {
	dss, err := c.Clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return dss.Items, nil
}

func (c *Client) ListJobs(ctx context.Context, namespace string) ([]batchv1.Job, error) {
	jobs, err := c.Clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return jobs.Items, nil
}

func (c *Client) ListCronJobs(ctx context.Context, namespace string) ([]batchv1.CronJob, error) {
	cjs, err := c.Clientset.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return cjs.Items, nil
}

func (c *Client) ListHorizontalPodAutoscalers(ctx context.Context, namespace string) ([]autoscalingv2.HorizontalPodAutoscaler, error) {
	hpas, err := c.Clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return hpas.Items, nil
}

func (c *Client) ListNetworkPolicies(ctx context.Context, namespace string) ([]networkingv1.NetworkPolicy, error) {
	netpols, err := c.Clientset.NetworkingV1().NetworkPolicies(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return netpols.Items, nil
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

func (c *Client) ScaleResource(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string, replicas int32) error {
	// For deployments, statefulsets, etc.
	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"replicas": replicas,
		},
	}
	payload, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("failed to marshal scale patch: %w", err)
	}
	_, err = c.Dynamic.Resource(gvr).Namespace(namespace).Patch(ctx, name, types.MergePatchType, payload, metav1.PatchOptions{})
	return err
}

func (c *Client) RolloutRestart(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) error {
	// Trigger restart by updating annotation
	timestamp := time.Now().Format(time.RFC3339)
	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"kubectl.kubernetes.io/restartedAt": timestamp,
					},
				},
			},
		},
	}
	payload, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("failed to marshal restart patch: %w", err)
	}
	_, err = c.Dynamic.Resource(gvr).Namespace(namespace).Patch(ctx, name, types.MergePatchType, payload, metav1.PatchOptions{})
	return err
}

func (c *Client) PortForward(ctx context.Context, namespace, podName string, localPort, podPort int, stopCh, readyCh chan struct{}) error {
	roundTripper, upgrader, err := spdy.RoundTripperFor(c.Config)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, podName)
	// Use url.Parse to correctly extract the host, rather than TrimLeft which strips a character set
	parsedURL, parseErr := url.Parse(c.Config.Host)
	var hostIP string
	if parseErr == nil && parsedURL.Host != "" {
		hostIP = parsedURL.Host
	} else {
		// Fallback: strip common prefixes
		hostIP = strings.TrimPrefix(strings.TrimPrefix(c.Config.Host, "https://"), "http://")
	}
	serverURL := url.URL{Scheme: "https", Path: path, Host: hostIP}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, &serverURL)

	ports := []string{fmt.Sprintf("%d:%d", localPort, podPort)}
	pf, err := portforward.New(dialer, ports, stopCh, readyCh, nil, nil)
	if err != nil {
		return err
	}

	return pf.ForwardPorts()
}

func (c *Client) DeleteResource(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) error {
	return c.Dynamic.Resource(gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// GetPodLogsPrevious gets logs from the previous container instance
func (c *Client) GetPodLogsPrevious(ctx context.Context, namespace, name, container string, tailLines int64) (string, error) {
	previous := true
	opts := &corev1.PodLogOptions{
		Container: container,
		Previous:  previous,
	}
	if tailLines > 0 {
		opts.TailLines = &tailLines
	}
	req := c.Clientset.CoreV1().Pods(namespace).GetLogs(name, opts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer podLogs.Close()

	buf := new(strings.Builder)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// DeletePodForce force deletes a pod with grace period 0
func (c *Client) DeletePodForce(ctx context.Context, namespace, name string) error {
	gracePeriod := int64(0)
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}
	return c.Clientset.CoreV1().Pods(namespace).Delete(ctx, name, deleteOptions)
}

func (c *Client) GetPodMetrics(ctx context.Context, namespace string) (map[string][]int64, error) {
	if c.Metrics == nil {
		return nil, fmt.Errorf("metrics client not initialized")
	}
	podMetrics, err := c.Metrics.PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	res := make(map[string][]int64)
	for _, pm := range podMetrics.Items {
		var cpu, mem int64
		for _, container := range pm.Containers {
			cpu += container.Usage.Cpu().MilliValue()
			mem += container.Usage.Memory().Value() / 1024 / 1024 // MB
		}
		res[pm.Name] = []int64{cpu, mem}
	}
	return res, nil
}

func (c *Client) GetNodeMetrics(ctx context.Context) (map[string][]int64, error) {
	if c.Metrics == nil {
		return nil, fmt.Errorf("metrics client not initialized")
	}
	nodeMetrics, err := c.Metrics.NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	res := make(map[string][]int64)
	for _, nm := range nodeMetrics.Items {
		cpu := nm.Usage.Cpu().MilliValue()
		mem := nm.Usage.Memory().Value() / 1024 / 1024 // MB
		res[nm.Name] = []int64{cpu, mem}
	}
	return res, nil
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

func (c *Client) GetResourceYAML(ctx context.Context, namespace, name string, gvr schema.GroupVersionResource) (string, error) {
	obj, err := c.Dynamic.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
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

// WatchResource starts a Watch on the given resource type.
// Uses typed clientset methods for fake clientset test compatibility.
func (c *Client) WatchResource(ctx context.Context, resource, namespace string) (watch.Interface, error) {
	opts := metav1.ListOptions{}
	switch strings.ToLower(resource) {
	// Core resources
	case "pods":
		return c.Clientset.CoreV1().Pods(namespace).Watch(ctx, opts)
	case "services":
		return c.Clientset.CoreV1().Services(namespace).Watch(ctx, opts)
	case "nodes":
		return c.Clientset.CoreV1().Nodes().Watch(ctx, opts)
	case "namespaces":
		return c.Clientset.CoreV1().Namespaces().Watch(ctx, opts)
	case "events":
		return c.Clientset.CoreV1().Events(namespace).Watch(ctx, opts)
	case "configmaps":
		return c.Clientset.CoreV1().ConfigMaps(namespace).Watch(ctx, opts)
	case "secrets":
		return c.Clientset.CoreV1().Secrets(namespace).Watch(ctx, opts)
	case "persistentvolumes":
		return c.Clientset.CoreV1().PersistentVolumes().Watch(ctx, opts)
	case "persistentvolumeclaims":
		return c.Clientset.CoreV1().PersistentVolumeClaims(namespace).Watch(ctx, opts)
	case "serviceaccounts":
		return c.Clientset.CoreV1().ServiceAccounts(namespace).Watch(ctx, opts)
	case "endpoints":
		return c.Clientset.CoreV1().Endpoints(namespace).Watch(ctx, opts)
	case "limitranges":
		return c.Clientset.CoreV1().LimitRanges(namespace).Watch(ctx, opts)
	case "resourcequotas":
		return c.Clientset.CoreV1().ResourceQuotas(namespace).Watch(ctx, opts)
	case "replicationcontrollers":
		return c.Clientset.CoreV1().ReplicationControllers(namespace).Watch(ctx, opts)

	// Apps resources
	case "deployments":
		return c.Clientset.AppsV1().Deployments(namespace).Watch(ctx, opts)
	case "statefulsets":
		return c.Clientset.AppsV1().StatefulSets(namespace).Watch(ctx, opts)
	case "daemonsets":
		return c.Clientset.AppsV1().DaemonSets(namespace).Watch(ctx, opts)
	case "replicasets":
		return c.Clientset.AppsV1().ReplicaSets(namespace).Watch(ctx, opts)

	// Batch resources
	case "jobs":
		return c.Clientset.BatchV1().Jobs(namespace).Watch(ctx, opts)
	case "cronjobs":
		return c.Clientset.BatchV1().CronJobs(namespace).Watch(ctx, opts)

	// Networking resources
	case "ingresses":
		return c.Clientset.NetworkingV1().Ingresses(namespace).Watch(ctx, opts)
	case "networkpolicies":
		return c.Clientset.NetworkingV1().NetworkPolicies(namespace).Watch(ctx, opts)

	// RBAC resources
	case "roles":
		return c.Clientset.RbacV1().Roles(namespace).Watch(ctx, opts)
	case "rolebindings":
		return c.Clientset.RbacV1().RoleBindings(namespace).Watch(ctx, opts)
	case "clusterroles":
		return c.Clientset.RbacV1().ClusterRoles().Watch(ctx, opts)
	case "clusterrolebindings":
		return c.Clientset.RbacV1().ClusterRoleBindings().Watch(ctx, opts)

	// Storage resources
	case "storageclasses":
		return c.Clientset.StorageV1().StorageClasses().Watch(ctx, opts)

	// Policy resources
	case "poddisruptionbudgets":
		return c.Clientset.PolicyV1().PodDisruptionBudgets(namespace).Watch(ctx, opts)

	// Autoscaling resources
	case "horizontalpodautoscalers":
		return c.Clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).Watch(ctx, opts)

	default:
		return nil, fmt.Errorf("watch not supported for resource: %s", resource)
	}
}

// GetRestConfig returns the REST config for the client
func (c *Client) GetRestConfig() (*rest.Config, error) {
	if c.Config != nil {
		return c.Config, nil
	}
	// Fallback: load from default config
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	return kubeConfig.ClientConfig()
}

// DefaultGetOptions returns default options for Get operations
func DefaultGetOptions() metav1.GetOptions {
	return metav1.GetOptions{}
}

// DefaultListOptions returns default options for List operations
func DefaultListOptions() metav1.ListOptions {
	return metav1.ListOptions{}
}

// RollbackDeployment rolls back a deployment to a previous revision
func (c *Client) RollbackDeployment(ctx context.Context, namespace, name string, revision int64) error {
	// Get deployment
	deployment, err := c.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Get ReplicaSets for this deployment
	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return fmt.Errorf("failed to parse selector: %w", err)
	}

	rsList, err := c.Clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to list replicasets: %w", err)
	}

	// Find the ReplicaSet with the target revision
	// NOTE: Revision parsing uses Sscanf which silently ignores malformed values (rev stays 0).
	// This is acceptable since Kubernetes guarantees the annotation format is a decimal integer.
	var targetRS *appsv1.ReplicaSet
	for i := range rsList.Items {
		rs := &rsList.Items[i]
		if rs.Annotations != nil {
			if revStr, ok := rs.Annotations["deployment.kubernetes.io/revision"]; ok {
				var rev int64
				if _, err := fmt.Sscanf(revStr, "%d", &rev); err != nil {
					continue
				}
				if rev == revision {
					targetRS = rs
					break
				}
			}
		}
	}

	if targetRS == nil {
		return fmt.Errorf("revision %d not found", revision)
	}

	// Copy the pod template from the target ReplicaSet
	deployment.Spec.Template = targetRS.Spec.Template

	// Update the deployment
	_, err = c.Clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	return nil
}

// PauseDeployment pauses a deployment's rollout
func (c *Client) PauseDeployment(ctx context.Context, namespace, name string) error {
	payload := []byte(`{"spec":{"paused":true}}`)
	_, err := c.Clientset.AppsV1().Deployments(namespace).Patch(ctx, name, types.MergePatchType, payload, metav1.PatchOptions{})
	return err
}

// ResumeDeployment resumes a paused deployment
func (c *Client) ResumeDeployment(ctx context.Context, namespace, name string) error {
	payload := []byte(`{"spec":{"paused":false}}`)
	_, err := c.Clientset.AppsV1().Deployments(namespace).Patch(ctx, name, types.MergePatchType, payload, metav1.PatchOptions{})
	return err
}

// GetDeploymentReplicaSets returns all ReplicaSets for a deployment with revision info
func (c *Client) GetDeploymentReplicaSets(ctx context.Context, namespace, name string) ([]map[string]interface{}, error) {
	deployment, err := c.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return nil, err
	}

	rsList, err := c.Clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, rs := range rsList.Items {
		revision := "0"
		if rs.Annotations != nil {
			if rev, ok := rs.Annotations["deployment.kubernetes.io/revision"]; ok {
				revision = rev
			}
		}
		result = append(result, map[string]interface{}{
			"name":      rs.Name,
			"revision":  revision,
			"replicas":  rs.Status.Replicas,
			"ready":     rs.Status.ReadyReplicas,
			"available": rs.Status.AvailableReplicas,
			"age":       time.Since(rs.CreationTimestamp.Time).Round(time.Second).String(),
		})
	}
	return result, nil
}

// TriggerCronJob creates a Job from a CronJob (manual trigger)
func (c *Client) TriggerCronJob(ctx context.Context, namespace, name string) (*batchv1.Job, error) {
	cronJob, err := c.Clientset.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cronjob: %w", err)
	}

	// Create a Job from the CronJob spec
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-manual-%d", name, time.Now().Unix()),
			Namespace: namespace,
			Labels:    cronJob.Spec.JobTemplate.Labels,
			Annotations: map[string]string{
				"cronjob.kubernetes.io/instantiate": "manual",
			},
		},
		Spec: cronJob.Spec.JobTemplate.Spec,
	}

	return c.Clientset.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
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

// Extended List functions for additional resources

func (c *Client) ListReplicaSets(ctx context.Context, namespace string) ([]appsv1.ReplicaSet, error) {
	opts := metav1.ListOptions{}
	if namespace == "" {
		list, err := c.Clientset.AppsV1().ReplicaSets("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return list.Items, nil
	}
	list, err := c.Clientset.AppsV1().ReplicaSets(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) ListReplicationControllers(ctx context.Context, namespace string) ([]corev1.ReplicationController, error) {
	opts := metav1.ListOptions{}
	if namespace == "" {
		list, err := c.Clientset.CoreV1().ReplicationControllers("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return list.Items, nil
	}
	list, err := c.Clientset.CoreV1().ReplicationControllers(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) ListEndpoints(ctx context.Context, namespace string) ([]corev1.Endpoints, error) { //nolint:staticcheck
	opts := metav1.ListOptions{}
	if namespace == "" {
		list, err := c.Clientset.CoreV1().Endpoints("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return list.Items, nil
	}
	list, err := c.Clientset.CoreV1().Endpoints(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) ListPodDisruptionBudgets(ctx context.Context, namespace string) ([]policyv1.PodDisruptionBudget, error) {
	opts := metav1.ListOptions{}
	if namespace == "" {
		list, err := c.Clientset.PolicyV1().PodDisruptionBudgets("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return list.Items, nil
	}
	list, err := c.Clientset.PolicyV1().PodDisruptionBudgets(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) ListLimitRanges(ctx context.Context, namespace string) ([]corev1.LimitRange, error) {
	opts := metav1.ListOptions{}
	if namespace == "" {
		list, err := c.Clientset.CoreV1().LimitRanges("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return list.Items, nil
	}
	list, err := c.Clientset.CoreV1().LimitRanges(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) ListResourceQuotas(ctx context.Context, namespace string) ([]corev1.ResourceQuota, error) {
	opts := metav1.ListOptions{}
	if namespace == "" {
		list, err := c.Clientset.CoreV1().ResourceQuotas("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return list.Items, nil
	}
	list, err := c.Clientset.CoreV1().ResourceQuotas(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) ListHPAs(ctx context.Context, namespace string) ([]autoscalingv2.HorizontalPodAutoscaler, error) {
	opts := metav1.ListOptions{}
	if namespace == "" {
		list, err := c.Clientset.AutoscalingV2().HorizontalPodAutoscalers("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return list.Items, nil
	}
	list, err := c.Clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) ListCRDs(ctx context.Context) ([]apiextv1.CustomResourceDefinition, error) {
	gvr := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}
	list, err := c.Dynamic.Resource(gvr).List(ctx, metav1.ListOptions{})
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
	gvr := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}

	obj, err := c.Dynamic.Resource(gvr).Get(ctx, crdName, metav1.GetOptions{})
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
	gvr := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}

	list, err := c.Dynamic.Resource(gvr).List(ctx, metav1.ListOptions{})
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

// ListCustomResources lists instances of a custom resource
func (c *Client) ListCustomResources(ctx context.Context, crdInfo *CRDInfo, namespace string) ([]unstructured.Unstructured, error) {
	gvr := schema.GroupVersionResource{
		Group:    crdInfo.Group,
		Version:  crdInfo.Version,
		Resource: crdInfo.Plural,
	}

	var list *unstructured.UnstructuredList
	var err error

	if crdInfo.Namespaced && namespace != "" {
		list, err = c.Dynamic.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	} else {
		list, err = c.Dynamic.Resource(gvr).List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		return nil, err
	}

	return list.Items, nil
}

// GetCustomResource gets a single custom resource instance
func (c *Client) GetCustomResource(ctx context.Context, crdInfo *CRDInfo, namespace, name string) (*unstructured.Unstructured, error) {
	gvr := schema.GroupVersionResource{
		Group:    crdInfo.Group,
		Version:  crdInfo.Version,
		Resource: crdInfo.Plural,
	}

	if crdInfo.Namespaced && namespace != "" {
		return c.Dynamic.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	}
	return c.Dynamic.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
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

// DescribeResource returns kubectl describe-like output for a resource
func (c *Client) DescribeResource(ctx context.Context, resource, namespace, name string) (string, error) {
	var result strings.Builder

	switch resource {
	case "pods":
		pod, err := c.Clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
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
		events, _ := c.Clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
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
		dep, err := c.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
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
		svc, err := c.Clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
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
		node, err := c.Clientset.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
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
		gvr, err := c.getGVRForResource(resource)
		if err != nil {
			return "", err
		}
		var obj interface{}
		if namespace != "" {
			unstructured, err := c.Dynamic.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return "", err
			}
			obj = unstructured.Object
		} else {
			unstructured, err := c.Dynamic.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
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

// ListDynamicResource lists resources using the dynamic client for any resource type
func (c *Client) ListDynamicResource(ctx context.Context, gvr schema.GroupVersionResource, namespace string) ([]map[string]interface{}, error) {
	var uList *unstructured.UnstructuredList
	var err error

	if namespace == "" {
		uList, err = c.Dynamic.Resource(gvr).List(ctx, metav1.ListOptions{})
	} else {
		uList, err = c.Dynamic.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
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

// ==========================================
// Port Forwarding
// ==========================================

// StartPortForward starts port forwarding to a pod
func (c *Client) StartPortForward(namespace, podName string, localPort, remotePort int, stopChan chan struct{}) error {
	// Get the pod to verify it exists
	_, err := c.Clientset.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pod: %w", err)
	}

	// Create the URL for the pod exec endpoint
	req := c.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(c.Config)
	if err != nil {
		return fmt.Errorf("failed to create round tripper: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, req.URL())

	ports := []string{fmt.Sprintf("%d:%d", localPort, remotePort)}
	readyChan := make(chan struct{})
	errChan := make(chan error)

	fw, err := portforward.New(dialer, ports, stopChan, readyChan, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to create port forwarder: %w", err)
	}

	go func() {
		errChan <- fw.ForwardPorts()
	}()

	select {
	case <-readyChan:
		fmt.Printf("Port forwarding is ready: localhost:%d -> %s/%s:%d\n", localPort, namespace, podName, remotePort)
	case err := <-errChan:
		return fmt.Errorf("port forwarding failed: %w", err)
	case <-stopChan:
		return nil
	}

	// Wait for stop signal or error
	select {
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("port forwarding error: %w", err)
		}
	case <-stopChan:
	}

	return nil
}

// ApplyYAML applies a YAML manifest to the cluster using kubectl-style apply
// It supports dry-run mode for validation without actually applying changes
func (c *Client) ApplyYAML(ctx context.Context, yamlContent string, defaultNamespace string, dryRun bool) (string, error) {
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
		resourceClient = c.Dynamic.Resource(gvr).Namespace(namespace)
	} else {
		resourceClient = c.Dynamic.Resource(gvr)
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

// convertToStringKeyMap converts a map[interface{}]interface{} to map[string]interface{}
// This is needed because YAML unmarshals to interface{} keys, but unstructured needs string keys
func convertToStringKeyMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = convertValue(v)
	}
	return result
}

func convertValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[interface{}]interface{}:
		result := make(map[string]interface{})
		for k, v := range val {
			result[fmt.Sprintf("%v", k)] = convertValue(v)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, v := range val {
			result[k] = convertValue(v)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, v := range val {
			result[i] = convertValue(v)
		}
		return result
	default:
		return v
	}
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
