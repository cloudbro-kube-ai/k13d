package k8s

import (
	"context"
	"io"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Reader defines read-only operations on Kubernetes resources.
// Tests should validate behavior through this interface for better
// backward compatibility when implementations change.
type Reader interface {
	// Core resources
	ListPods(ctx context.Context, namespace string) ([]corev1.Pod, error)
	ListNodes(ctx context.Context) ([]corev1.Node, error)
	ListNamespaces(ctx context.Context) ([]corev1.Namespace, error)
	ListServices(ctx context.Context, namespace string) ([]corev1.Service, error)
	ListConfigMaps(ctx context.Context, namespace string) ([]corev1.ConfigMap, error)
	ListSecrets(ctx context.Context, namespace string) ([]corev1.Secret, error)
	ListEvents(ctx context.Context, namespace string) ([]corev1.Event, error)
	ListServiceAccounts(ctx context.Context, namespace string) ([]corev1.ServiceAccount, error)
	ListPersistentVolumes(ctx context.Context) ([]corev1.PersistentVolume, error)
	ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]corev1.PersistentVolumeClaim, error)

	// Apps resources
	ListDeployments(ctx context.Context, namespace string) ([]appsv1.Deployment, error)
	ListStatefulSets(ctx context.Context, namespace string) ([]appsv1.StatefulSet, error)
	ListDaemonSets(ctx context.Context, namespace string) ([]appsv1.DaemonSet, error)
	ListReplicaSets(ctx context.Context, namespace string) ([]appsv1.ReplicaSet, error)

	// Batch resources
	ListJobs(ctx context.Context, namespace string) ([]batchv1.Job, error)
	ListCronJobs(ctx context.Context, namespace string) ([]batchv1.CronJob, error)

	// Networking resources
	ListIngresses(ctx context.Context, namespace string) ([]networkingv1.Ingress, error)
	ListNetworkPolicies(ctx context.Context, namespace string) ([]networkingv1.NetworkPolicy, error)

	// RBAC resources
	ListRoles(ctx context.Context, namespace string) ([]rbacv1.Role, error)
	ListClusterRoles(ctx context.Context) ([]rbacv1.ClusterRole, error)
	ListRoleBindings(ctx context.Context, namespace string) ([]rbacv1.RoleBinding, error)
	ListClusterRoleBindings(ctx context.Context) ([]rbacv1.ClusterRoleBinding, error)

	// Storage resources
	ListStorageClasses(ctx context.Context) ([]storagev1.StorageClass, error)

	// Autoscaling resources
	ListHorizontalPodAutoscalers(ctx context.Context, namespace string) ([]autoscalingv2.HorizontalPodAutoscaler, error)

	// Resource details
	GetPodLogs(ctx context.Context, namespace, name, container string, tailLines int64) (string, error)
	GetPodLogsStream(ctx context.Context, namespace, name string) (io.ReadCloser, error)
	GetResourceYAML(ctx context.Context, namespace, name string, gvr schema.GroupVersionResource) (string, error)
	DescribeResource(ctx context.Context, kind, namespace, name string) (string, error)
	ListTable(ctx context.Context, gvr schema.GroupVersionResource, ns string) (*metav1.Table, error)
}

// Writer defines write operations on Kubernetes resources.
type Writer interface {
	// Scale operations
	ScaleResource(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string, replicas int32) error

	// Rollout operations
	RolloutRestart(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) error

	// Delete operations
	DeleteResource(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) error

	// Apply operations
	ApplyYAML(ctx context.Context, yamlContent string, defaultNamespace string, dryRun bool) (string, error)
}

// ContextManager defines context and namespace operations.
type ContextManager interface {
	GetContextInfo() (ctxName, cluster, user string, err error)
	GetCurrentContext() (string, error)
	GetCurrentNamespace() string
	ListContexts() ([]string, string, error)
	SwitchContext(contextName string) error
	GetServerVersion() (string, error)
}

// ClientInterface combines all K8s client operations.
// This is the main interface that consuming code should depend on.
type ClientInterface interface {
	Reader
	Writer
	ContextManager
}

// Ensure Client implements ClientInterface at compile time.
var _ ClientInterface = (*Client)(nil)
