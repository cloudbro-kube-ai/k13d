package web

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// ApplicationGroup is the API response type for application grouping
type ApplicationGroup struct {
	Name      string                   `json:"name"`
	Version   string                   `json:"version,omitempty"`
	Component string                   `json:"component,omitempty"`
	Status    string                   `json:"status"` // "healthy", "degraded", "failing"
	PodCount  int                      `json:"podCount"`
	ReadyPods int                      `json:"readyPods"`
	Resources map[string][]ResourceRef `json:"resources"`
}

// ResourceRef identifies a single Kubernetes resource
type ResourceRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Status    string `json:"status,omitempty"`
}

func (s *Server) handleApplications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	namespace := r.URL.Query().Get("namespace")
	ctx := r.Context()

	groups, err := s.buildApplicationGroups(ctx, namespace)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(groups)
}

func (s *Server) buildApplicationGroups(ctx context.Context, namespace string) ([]ApplicationGroup, error) {
	var (
		mu           sync.Mutex
		wg           sync.WaitGroup
		pods         []corev1.Pod
		deployments  []appsv1.Deployment
		statefulSets []appsv1.StatefulSet
		daemonSets   []appsv1.DaemonSet
		services     []corev1.Service
		configMaps   []corev1.ConfigMap
		secrets      []corev1.Secret
		ingresses    []networkingv1.Ingress
	)

	type fetchResult struct {
		name string
		err  error
	}
	errCh := make(chan fetchResult, 8)

	fetch := func(name string, fn func() error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(); err != nil {
				errCh <- fetchResult{name, err}
			}
		}()
	}

	fetch("pods", func() error {
		var err error
		pods, err = s.k8sClient.ListPods(ctx, namespace)
		return err
	})
	fetch("deployments", func() error {
		var err error
		deployments, err = s.k8sClient.ListDeployments(ctx, namespace)
		return err
	})
	fetch("statefulsets", func() error {
		var err error
		statefulSets, err = s.k8sClient.ListStatefulSets(ctx, namespace)
		return err
	})
	fetch("daemonsets", func() error {
		var err error
		daemonSets, err = s.k8sClient.ListDaemonSets(ctx, namespace)
		return err
	})
	fetch("services", func() error {
		var err error
		services, err = s.k8sClient.ListServices(ctx, namespace)
		return err
	})
	fetch("configmaps", func() error {
		var err error
		configMaps, err = s.k8sClient.ListConfigMaps(ctx, namespace)
		return err
	})
	fetch("secrets", func() error {
		var err error
		secrets, err = s.k8sClient.ListSecrets(ctx, namespace)
		return err
	})
	fetch("ingresses", func() error {
		var err error
		ingresses, err = s.k8sClient.ListIngresses(ctx, namespace)
		return err
	})

	wg.Wait()
	close(errCh)

	// Log errors but continue with partial data
	for range errCh {
		// non-critical: partial results are still useful
	}

	// Build groups keyed by app.kubernetes.io/name
	groupMap := make(map[string]*ApplicationGroup)
	var ungroupedResources map[string][]ResourceRef

	getOrCreate := func(name string) *ApplicationGroup {
		mu.Lock()
		defer mu.Unlock()
		if g, ok := groupMap[name]; ok {
			return g
		}
		g := &ApplicationGroup{
			Name:      name,
			Resources: make(map[string][]ResourceRef),
		}
		groupMap[name] = g
		return g
	}

	addUngrouped := func(kind, name, ns string) {
		mu.Lock()
		defer mu.Unlock()
		if ungroupedResources == nil {
			ungroupedResources = make(map[string][]ResourceRef)
		}
		ungroupedResources[kind] = append(ungroupedResources[kind], ResourceRef{Name: name, Namespace: ns})
	}

	// Group Deployments
	for _, d := range deployments {
		appName := d.Labels["app.kubernetes.io/name"]
		if appName == "" {
			addUngrouped("Deployment", d.Name, d.Namespace)
			continue
		}
		g := getOrCreate(appName)
		if v := d.Labels["app.kubernetes.io/version"]; v != "" && g.Version == "" {
			g.Version = v
		}
		if c := d.Labels["app.kubernetes.io/component"]; c != "" && g.Component == "" {
			g.Component = c
		}
		status := "running"
		replicas := int32(1)
		if d.Spec.Replicas != nil {
			replicas = *d.Spec.Replicas
		}
		if d.Status.ReadyReplicas < replicas {
			status = "pending"
		}
		g.Resources["Deployment"] = append(g.Resources["Deployment"], ResourceRef{
			Name: d.Name, Namespace: d.Namespace, Status: status,
		})
	}

	// Group StatefulSets
	for _, s := range statefulSets {
		appName := s.Labels["app.kubernetes.io/name"]
		if appName == "" {
			addUngrouped("StatefulSet", s.Name, s.Namespace)
			continue
		}
		g := getOrCreate(appName)
		if v := s.Labels["app.kubernetes.io/version"]; v != "" && g.Version == "" {
			g.Version = v
		}
		status := "running"
		replicas := int32(1)
		if s.Spec.Replicas != nil {
			replicas = *s.Spec.Replicas
		}
		if s.Status.ReadyReplicas < replicas {
			status = "pending"
		}
		g.Resources["StatefulSet"] = append(g.Resources["StatefulSet"], ResourceRef{
			Name: s.Name, Namespace: s.Namespace, Status: status,
		})
	}

	// Group DaemonSets
	for _, ds := range daemonSets {
		appName := ds.Labels["app.kubernetes.io/name"]
		if appName == "" {
			addUngrouped("DaemonSet", ds.Name, ds.Namespace)
			continue
		}
		g := getOrCreate(appName)
		g.Resources["DaemonSet"] = append(g.Resources["DaemonSet"], ResourceRef{
			Name: ds.Name, Namespace: ds.Namespace,
		})
	}

	// Group Services
	for _, svc := range services {
		appName := svc.Labels["app.kubernetes.io/name"]
		if appName == "" {
			addUngrouped("Service", svc.Name, svc.Namespace)
			continue
		}
		g := getOrCreate(appName)
		g.Resources["Service"] = append(g.Resources["Service"], ResourceRef{
			Name: svc.Name, Namespace: svc.Namespace,
		})
	}

	// Group ConfigMaps
	for _, cm := range configMaps {
		appName := cm.Labels["app.kubernetes.io/name"]
		if appName == "" {
			addUngrouped("ConfigMap", cm.Name, cm.Namespace)
			continue
		}
		g := getOrCreate(appName)
		g.Resources["ConfigMap"] = append(g.Resources["ConfigMap"], ResourceRef{
			Name: cm.Name, Namespace: cm.Namespace,
		})
	}

	// Group Secrets
	for _, sec := range secrets {
		appName := sec.Labels["app.kubernetes.io/name"]
		if appName == "" {
			addUngrouped("Secret", sec.Name, sec.Namespace)
			continue
		}
		g := getOrCreate(appName)
		g.Resources["Secret"] = append(g.Resources["Secret"], ResourceRef{
			Name: sec.Name, Namespace: sec.Namespace,
		})
	}

	// Group Ingresses
	for _, ing := range ingresses {
		appName := ing.Labels["app.kubernetes.io/name"]
		if appName == "" {
			addUngrouped("Ingress", ing.Name, ing.Namespace)
			continue
		}
		g := getOrCreate(appName)
		g.Resources["Ingress"] = append(g.Resources["Ingress"], ResourceRef{
			Name: ing.Name, Namespace: ing.Namespace,
		})
	}

	// Count pods per group and calculate health
	for _, pod := range pods {
		appName := pod.Labels["app.kubernetes.io/name"]
		if appName == "" {
			continue
		}
		if g, ok := groupMap[appName]; ok {
			g.PodCount++
			if pod.Status.Phase == corev1.PodRunning {
				allReady := true
				for _, cs := range pod.Status.ContainerStatuses {
					if !cs.Ready {
						allReady = false
						break
					}
				}
				if allReady {
					g.ReadyPods++
				}
			}
		}
	}

	// Calculate status
	for _, g := range groupMap {
		if g.PodCount == 0 {
			g.Status = "healthy"
		} else if g.ReadyPods == g.PodCount {
			g.Status = "healthy"
		} else if g.ReadyPods == 0 {
			g.Status = "failing"
		} else {
			g.Status = "degraded"
		}
	}

	// Build sorted result
	var result []ApplicationGroup
	for _, g := range groupMap {
		result = append(result, *g)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	// Add ungrouped at the end
	if len(ungroupedResources) > 0 {
		result = append(result, ApplicationGroup{
			Name:      "Ungrouped",
			Status:    "healthy",
			Resources: ungroupedResources,
		})
	}

	return result, nil
}
