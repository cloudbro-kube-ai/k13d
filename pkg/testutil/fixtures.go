package testutil

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

// Fixtures provides reusable test data for Kubernetes resources.
// Use these fixtures instead of creating ad-hoc test data in each test.
type Fixtures struct{}

// NewFixtures creates a new Fixtures instance.
func NewFixtures() *Fixtures {
	return &Fixtures{}
}

// Pod creates a test pod with sensible defaults.
func (f *Fixtures) Pod(name, namespace string, opts ...PodOption) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"app": name},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "main", Image: "nginx:latest"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}
	for _, opt := range opts {
		opt(pod)
	}
	return pod
}

// PodOption modifies a Pod fixture.
type PodOption func(*corev1.Pod)

// WithPodPhase sets the pod phase.
func WithPodPhase(phase corev1.PodPhase) PodOption {
	return func(p *corev1.Pod) { p.Status.Phase = phase }
}

// WithPodLabels sets pod labels.
func WithPodLabels(labels map[string]string) PodOption {
	return func(p *corev1.Pod) { p.Labels = labels }
}

// WithPodIP sets the pod IP.
func WithPodIP(ip string) PodOption {
	return func(p *corev1.Pod) { p.Status.PodIP = ip }
}

// WithContainerStatus adds container status.
func WithContainerStatus(name string, ready bool, restarts int32) PodOption {
	return func(p *corev1.Pod) {
		p.Status.ContainerStatuses = append(p.Status.ContainerStatuses,
			corev1.ContainerStatus{Name: name, Ready: ready, RestartCount: restarts})
	}
}

// Deployment creates a test deployment.
func (f *Fixtures) Deployment(name, namespace string, replicas int32, opts ...DeploymentOption) *appsv1.Deployment {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "main", Image: "nginx:latest"},
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			Replicas:          replicas,
			ReadyReplicas:     replicas,
			AvailableReplicas: replicas,
		},
	}
	for _, opt := range opts {
		opt(dep)
	}
	return dep
}

// DeploymentOption modifies a Deployment fixture.
type DeploymentOption func(*appsv1.Deployment)

// WithDeploymentUnavailable sets unavailable replicas.
func WithDeploymentUnavailable(unavailable int32) DeploymentOption {
	return func(d *appsv1.Deployment) {
		d.Status.UnavailableReplicas = unavailable
		d.Status.AvailableReplicas = *d.Spec.Replicas - unavailable
	}
}

// Service creates a test service.
func (f *Fixtures) Service(name, namespace string, svcType corev1.ServiceType, opts ...ServiceOption) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Type:      svcType,
			ClusterIP: "10.96.0.1",
			Ports: []corev1.ServicePort{
				{Port: 80, Protocol: corev1.ProtocolTCP},
			},
			Selector: map[string]string{"app": name},
		},
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// ServiceOption modifies a Service fixture.
type ServiceOption func(*corev1.Service)

// WithExternalIP sets external IP for LoadBalancer.
func WithExternalIP(ip string) ServiceOption {
	return func(s *corev1.Service) {
		s.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{{IP: ip}}
	}
}

// Node creates a test node.
func (f *Fixtures) Node(name string, ready bool, opts ...NodeOption) *corev1.Node {
	status := corev1.ConditionTrue
	if !ready {
		status = corev1.ConditionFalse
	}
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: status},
			},
			NodeInfo: corev1.NodeSystemInfo{
				KubeletVersion: "v1.28.0",
			},
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("3800m"),
				corev1.ResourceMemory: resource.MustParse("7Gi"),
			},
		},
	}
	for _, opt := range opts {
		opt(node)
	}
	return node
}

// NodeOption modifies a Node fixture.
type NodeOption func(*corev1.Node)

// WithNodeRole adds a role label.
func WithNodeRole(role string) NodeOption {
	return func(n *corev1.Node) {
		if n.Labels == nil {
			n.Labels = make(map[string]string)
		}
		n.Labels["node-role.kubernetes.io/"+role] = ""
	}
}

// Namespace creates a test namespace.
func (f *Fixtures) Namespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status:     corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
	}
}

// ConfigMap creates a test configmap.
func (f *Fixtures) ConfigMap(name, namespace string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
}

// Secret creates a test secret.
func (f *Fixtures) Secret(name, namespace string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: data,
	}
}

// StatefulSet creates a test statefulset.
func (f *Fixtures) StatefulSet(name, namespace string, replicas int32) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: name,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
		},
		Status: appsv1.StatefulSetStatus{
			Replicas:      replicas,
			ReadyReplicas: replicas,
		},
	}
}

// DaemonSet creates a test daemonset.
func (f *Fixtures) DaemonSet(name, namespace string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: appsv1.DaemonSetStatus{
			DesiredNumberScheduled: 3,
			CurrentNumberScheduled: 3,
			NumberReady:            3,
		},
	}
}

// Ingress creates a test ingress.
func (f *Fixtures) Ingress(name, namespace, host string) *networkingv1.Ingress {
	pathType := networkingv1.PathTypePrefix
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: name,
											Port: networkingv1.ServiceBackendPort{Number: 80},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Job creates a test job.
func (f *Fixtures) Job(name, namespace string, completed bool) *batchv1.Job {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if completed {
		job.Status.Succeeded = 1
		job.Status.CompletionTime = &metav1.Time{}
	}
	return job
}

// CronJob creates a test cronjob.
func (f *Fixtures) CronJob(name, namespace, schedule string) *batchv1.CronJob {
	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: schedule,
		},
	}
}

// Role creates a test role.
func (f *Fixtures) Role(name, namespace string) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list"},
			},
		},
	}
}

// ClusterRole creates a test clusterrole.
func (f *Fixtures) ClusterRole(name string) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		},
	}
}

// StorageClass creates a test storage class.
func (f *Fixtures) StorageClass(name, provisioner string) *storagev1.StorageClass {
	reclaimPolicy := corev1.PersistentVolumeReclaimDelete
	bindingMode := storagev1.VolumeBindingImmediate
	return &storagev1.StorageClass{
		ObjectMeta:           metav1.ObjectMeta{Name: name},
		Provisioner:          provisioner,
		ReclaimPolicy:        &reclaimPolicy,
		VolumeBindingMode:    &bindingMode,
		AllowVolumeExpansion: boolPtr(true),
	}
}

// Event creates a test event.
func (f *Fixtures) Event(name, namespace, eventType, reason, message string) *corev1.Event {
	return &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type:          eventType,
		Reason:        reason,
		Message:       message,
		Count:         1,
		LastTimestamp: metav1.Now(),
	}
}

// ServiceAccount creates a test service account.
func (f *Fixtures) ServiceAccount(name, namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func boolPtr(b bool) *bool { return &b }

// FakeClientBuilder helps build fake.Clientset with fixtures.
type FakeClientBuilder struct {
	fixtures *Fixtures
	objects  []runtime.Object
}

// NewFakeClientBuilder creates a builder for fake K8s clients.
func NewFakeClientBuilder() *FakeClientBuilder {
	return &FakeClientBuilder{
		fixtures: NewFixtures(),
		objects:  make([]runtime.Object, 0),
	}
}

// WithPod adds a pod to the fake client.
func (b *FakeClientBuilder) WithPod(name, namespace string, opts ...PodOption) *FakeClientBuilder {
	b.objects = append(b.objects, b.fixtures.Pod(name, namespace, opts...))
	return b
}

// WithDeployment adds a deployment to the fake client.
func (b *FakeClientBuilder) WithDeployment(name, namespace string, replicas int32, opts ...DeploymentOption) *FakeClientBuilder {
	b.objects = append(b.objects, b.fixtures.Deployment(name, namespace, replicas, opts...))
	return b
}

// WithService adds a service to the fake client.
func (b *FakeClientBuilder) WithService(name, namespace string, svcType corev1.ServiceType, opts ...ServiceOption) *FakeClientBuilder {
	b.objects = append(b.objects, b.fixtures.Service(name, namespace, svcType, opts...))
	return b
}

// WithNode adds a node to the fake client.
func (b *FakeClientBuilder) WithNode(name string, ready bool, opts ...NodeOption) *FakeClientBuilder {
	b.objects = append(b.objects, b.fixtures.Node(name, ready, opts...))
	return b
}

// WithNamespace adds a namespace to the fake client.
func (b *FakeClientBuilder) WithNamespace(name string) *FakeClientBuilder {
	b.objects = append(b.objects, b.fixtures.Namespace(name))
	return b
}

// WithObject adds a custom runtime.Object to the fake client.
func (b *FakeClientBuilder) WithObject(obj runtime.Object) *FakeClientBuilder {
	b.objects = append(b.objects, obj)
	return b
}

// Build creates the fake clientset.
func (b *FakeClientBuilder) Build() *fake.Clientset {
	return fake.NewSimpleClientset(b.objects...) //nolint:staticcheck // SA1019: migrating to NewClientset requires generated apply configurations
}
