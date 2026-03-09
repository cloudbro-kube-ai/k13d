package k8s

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

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

func (c *Client) GetPodLogsStream(ctx context.Context, namespace, name string) (io.ReadCloser, error) {
	req := c.Clientset.CoreV1().Pods(namespace).GetLogs(name, &corev1.PodLogOptions{Follow: true})
	return req.Stream(ctx)
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

// GetPodMetricsFromRequests estimates pod resource usage from container resource requests.
// Used as a fallback when metrics-server is unavailable.
func (c *Client) GetPodMetricsFromRequests(ctx context.Context, namespace string) (map[string][]int64, error) {
	pods, err := c.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	res := make(map[string][]int64)
	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}
		var cpu, mem int64
		for _, container := range pod.Spec.Containers {
			if req, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
				cpu += req.MilliValue()
			}
			if req, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
				mem += req.Value() / 1024 / 1024
			}
		}
		res[pod.Name] = []int64{cpu, mem}
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

// GetNodeMetricsFromPodRequests estimates node resource usage by summing
// resource requests of all running pods scheduled on each node.
// Used as a fallback when metrics-server is unavailable.
func (c *Client) GetNodeMetricsFromPodRequests(ctx context.Context) (map[string][]int64, error) {
	pods, err := c.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	res := make(map[string][]int64)
	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning || pod.Spec.NodeName == "" {
			continue
		}
		var cpu, mem int64
		for _, container := range pod.Spec.Containers {
			if req, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
				cpu += req.MilliValue()
			}
			if req, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
				mem += req.Value() / 1024 / 1024
			}
		}
		if existing, ok := res[pod.Spec.NodeName]; ok {
			res[pod.Spec.NodeName] = []int64{existing[0] + cpu, existing[1] + mem}
		} else {
			res[pod.Spec.NodeName] = []int64{cpu, mem}
		}
	}
	return res, nil
}
