package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func (c *Client) ListDeployments(ctx context.Context, namespace string) ([]appsv1.Deployment, error) {
	deps, err := c.clientset().AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return deps.Items, nil
}

func (c *Client) ListStatefulSets(ctx context.Context, namespace string) ([]appsv1.StatefulSet, error) {
	stses, err := c.clientset().AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return stses.Items, nil
}

func (c *Client) ListDaemonSets(ctx context.Context, namespace string) ([]appsv1.DaemonSet, error) {
	dss, err := c.clientset().AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return dss.Items, nil
}

func (c *Client) ListJobs(ctx context.Context, namespace string) ([]batchv1.Job, error) {
	jobs, err := c.clientset().BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return jobs.Items, nil
}

func (c *Client) ListCronJobs(ctx context.Context, namespace string) ([]batchv1.CronJob, error) {
	cjs, err := c.clientset().BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return cjs.Items, nil
}

func (c *Client) ListReplicaSets(ctx context.Context, namespace string) ([]appsv1.ReplicaSet, error) {
	opts := metav1.ListOptions{}
	if namespace == "" {
		list, err := c.clientset().AppsV1().ReplicaSets("").List(ctx, opts)
		if err != nil {
			return nil, err
		}
		return list.Items, nil
	}
	list, err := c.clientset().AppsV1().ReplicaSets(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (c *Client) ScaleResource(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string, replicas int32) error {
	if c.Dynamic == nil {
		return fmt.Errorf("dynamic client not initialized")
	}
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
	_, err = c.dynamicClient().Resource(gvr).Namespace(namespace).Patch(ctx, name, types.MergePatchType, payload, metav1.PatchOptions{})
	return err
}

func (c *Client) RolloutRestart(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) error {
	if c.Dynamic == nil {
		return fmt.Errorf("dynamic client not initialized")
	}
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
	_, err = c.dynamicClient().Resource(gvr).Namespace(namespace).Patch(ctx, name, types.MergePatchType, payload, metav1.PatchOptions{})
	return err
}

// RollbackDeployment rolls back a deployment to a previous revision
func (c *Client) RollbackDeployment(ctx context.Context, namespace, name string, revision int64) error {
	// Get deployment
	deployment, err := c.clientset().AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Get ReplicaSets for this deployment
	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return fmt.Errorf("failed to parse selector: %w", err)
	}

	rsList, err := c.clientset().AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to list replicasets: %w", err)
	}

	// Find the ReplicaSet with the target revision
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
	_, err = c.clientset().AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	return nil
}

// PauseDeployment pauses a deployment's rollout
func (c *Client) PauseDeployment(ctx context.Context, namespace, name string) error {
	payload := []byte(`{"spec":{"paused":true}}`)
	_, err := c.clientset().AppsV1().Deployments(namespace).Patch(ctx, name, types.MergePatchType, payload, metav1.PatchOptions{})
	return err
}

// ResumeDeployment resumes a paused deployment
func (c *Client) ResumeDeployment(ctx context.Context, namespace, name string) error {
	payload := []byte(`{"spec":{"paused":false}}`)
	_, err := c.clientset().AppsV1().Deployments(namespace).Patch(ctx, name, types.MergePatchType, payload, metav1.PatchOptions{})
	return err
}

// GetDeploymentReplicaSets returns all ReplicaSets for a deployment with revision info
func (c *Client) GetDeploymentReplicaSets(ctx context.Context, namespace, name string) ([]map[string]interface{}, error) {
	deployment, err := c.clientset().AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return nil, err
	}

	rsList, err := c.clientset().AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{
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
	cronJob, err := c.clientset().BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
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

	return c.clientset().BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
}
