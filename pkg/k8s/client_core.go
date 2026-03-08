package k8s

import (
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
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
func DefaultGetOptions() v1.GetOptions {
	return v1.GetOptions{}
}

// DefaultListOptions returns default options for List operations
func DefaultListOptions() v1.ListOptions {
	return v1.ListOptions{}
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
