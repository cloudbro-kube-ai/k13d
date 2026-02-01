// Package testutil provides shared testing utilities and interface contracts.
// This package enables interface-based testing for better backward compatibility
// and reduced coupling to implementation details.
package testutil

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
)

// K8sReader defines the read-only interface for Kubernetes resources.
// Tests should validate behavior through this interface rather than
// implementation details. This allows internal changes without breaking tests.
type K8sReader interface {
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
}

// K8sWriter defines the write interface for Kubernetes resources.
type K8sWriter interface {
	// Deployment operations
	ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) error
	RestartDeployment(ctx context.Context, namespace, name string) error
	DeleteDeployment(ctx context.Context, namespace, name string) error

	// Pod operations
	DeletePod(ctx context.Context, namespace, name string) error

	// Generic operations
	ApplyYAML(ctx context.Context, yaml string) error
	DeleteResource(ctx context.Context, kind, namespace, name string) error
}

// K8sClient combines read and write operations.
type K8sClient interface {
	K8sReader
	K8sWriter
}

// LLMProvider defines the interface for LLM providers.
// This matches pkg/ai/providers.Provider but is defined here
// to avoid circular dependencies in tests.
type LLMProvider interface {
	Name() string
	Ask(ctx context.Context, prompt string, callback func(string)) error
	AskNonStreaming(ctx context.Context, prompt string) (string, error)
	IsReady() bool
	GetModel() string
	ListModels(ctx context.Context) ([]string, error)
}

// LLMToolProvider extends LLMProvider with tool calling support.
type LLMToolProvider interface {
	LLMProvider
	SupportsTools() bool
}

// SessionStore defines the interface for session management.
type SessionStore interface {
	Create(id string) error
	Get(id string) (interface{}, error)
	Delete(id string) error
	List() ([]string, error)
}

// AuditLogger defines the interface for audit logging.
type AuditLogger interface {
	Log(action, resource, details string) error
	Query(filter AuditFilter) ([]AuditEntry, error)
}

// AuditFilter specifies criteria for querying audit logs.
type AuditFilter struct {
	Action    string
	Resource  string
	StartTime int64
	EndTime   int64
	Limit     int
}

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	ID        int64
	Timestamp int64
	Action    string
	Resource  string
	Details   string
}
