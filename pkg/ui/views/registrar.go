package views

import (
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ui/actions"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ui/render"
)

// GVR represents a Group/Version/Resource identifier.
type GVR string

// Common GVRs
const (
	GVRPods                   GVR = "v1/pods"
	GVRDeployments            GVR = "apps/v1/deployments"
	GVRServices               GVR = "v1/services"
	GVRNamespaces             GVR = "v1/namespaces"
	GVRNodes                  GVR = "v1/nodes"
	GVRConfigMaps             GVR = "v1/configmaps"
	GVRSecrets                GVR = "v1/secrets"
	GVRIngresses              GVR = "networking.k8s.io/v1/ingresses"
	GVRStatefulSets           GVR = "apps/v1/statefulsets"
	GVRDaemonSets             GVR = "apps/v1/daemonsets"
	GVRReplicaSets            GVR = "apps/v1/replicasets"
	GVRJobs                   GVR = "batch/v1/jobs"
	GVRCronJobs               GVR = "batch/v1/cronjobs"
	GVRPersistentVolumes      GVR = "v1/persistentvolumes"
	GVRPersistentVolumeClaims GVR = "v1/persistentvolumeclaims"
	GVRStorageClasses         GVR = "storage.k8s.io/v1/storageclasses"
	GVREvents                 GVR = "v1/events"
	GVRServiceAccounts        GVR = "v1/serviceaccounts"
	GVRRoles                  GVR = "rbac.authorization.k8s.io/v1/roles"
	GVRRoleBindings           GVR = "rbac.authorization.k8s.io/v1/rolebindings"
	GVRClusterRoles           GVR = "rbac.authorization.k8s.io/v1/clusterroles"
	GVRClusterRoleBindings    GVR = "rbac.authorization.k8s.io/v1/clusterrolebindings"
	GVRContexts               GVR = "contexts" // Not a real GVR, but used for context switching
)

// ViewerFunc creates a new view for a resource type.
type ViewerFunc func(namespace string) View

// MetaViewer holds the factory function and metadata for a resource view.
type MetaViewer struct {
	// ViewerFn creates the view.
	ViewerFn ViewerFunc
	// ActionsFn returns resource-specific actions.
	ActionsFn func() *actions.KeyActions
	// RendererFn creates the renderer.
	RendererFn func() render.Renderer
	// Aliases for the resource (e.g., "po" for pods).
	Aliases []string
	// Namespaced indicates if the resource is namespaced.
	Namespaced bool
}

// MetaViewers is a registry of viewers by GVR.
type MetaViewers map[GVR]MetaViewer

// Registrar manages view registration and lookup.
type Registrar struct {
	viewers MetaViewers
	aliases map[string]GVR
}

// NewRegistrar creates a new Registrar with default viewers.
func NewRegistrar() *Registrar {
	r := &Registrar{
		viewers: make(MetaViewers),
		aliases: make(map[string]GVR),
	}
	r.loadDefaultViewers()
	return r
}

// loadDefaultViewers loads the default resource viewers.
func (r *Registrar) loadDefaultViewers() {
	// Core resources
	r.Register(GVRPods, MetaViewer{
		ActionsFn:  actions.PodActions,
		RendererFn: func() render.Renderer { return render.NewPod() },
		Aliases:    []string{"po", "pod"},
		Namespaced: true,
	})

	r.Register(GVRDeployments, MetaViewer{
		ActionsFn:  actions.DeploymentActions,
		RendererFn: func() render.Renderer { return render.NewDeployment() },
		Aliases:    []string{"deploy", "deployment", "deployments"},
		Namespaced: true,
	})

	r.Register(GVRServices, MetaViewer{
		ActionsFn:  actions.ServiceActions,
		Aliases:    []string{"svc", "service", "services"},
		Namespaced: true,
	})

	r.Register(GVRNamespaces, MetaViewer{
		Aliases:    []string{"ns", "namespace", "namespaces"},
		Namespaced: false,
	})

	r.Register(GVRNodes, MetaViewer{
		ActionsFn:  actions.NodeActions,
		Aliases:    []string{"no", "node", "nodes"},
		Namespaced: false,
	})

	r.Register(GVRConfigMaps, MetaViewer{
		ActionsFn:  actions.ConfigMapActions,
		Aliases:    []string{"cm", "configmap", "configmaps"},
		Namespaced: true,
	})

	r.Register(GVRSecrets, MetaViewer{
		ActionsFn:  actions.SecretActions,
		Aliases:    []string{"secret", "secrets"},
		Namespaced: true,
	})

	r.Register(GVRStatefulSets, MetaViewer{
		ActionsFn:  actions.StatefulSetActions,
		Aliases:    []string{"sts", "statefulset", "statefulsets"},
		Namespaced: true,
	})

	r.Register(GVRDaemonSets, MetaViewer{
		ActionsFn:  actions.DaemonSetActions,
		Aliases:    []string{"ds", "daemonset", "daemonsets"},
		Namespaced: true,
	})

	r.Register(GVRJobs, MetaViewer{
		ActionsFn:  actions.JobActions,
		Aliases:    []string{"job", "jobs"},
		Namespaced: true,
	})

	r.Register(GVRCronJobs, MetaViewer{
		ActionsFn:  actions.CronJobActions,
		Aliases:    []string{"cj", "cronjob", "cronjobs"},
		Namespaced: true,
	})

	r.Register(GVRIngresses, MetaViewer{
		Aliases:    []string{"ing", "ingress", "ingresses"},
		Namespaced: true,
	})

	r.Register(GVRPersistentVolumes, MetaViewer{
		Aliases:    []string{"pv", "persistentvolume", "persistentvolumes"},
		Namespaced: false,
	})

	r.Register(GVRPersistentVolumeClaims, MetaViewer{
		Aliases:    []string{"pvc", "persistentvolumeclaim", "persistentvolumeclaims"},
		Namespaced: true,
	})

	r.Register(GVRStorageClasses, MetaViewer{
		Aliases:    []string{"sc", "storageclass", "storageclasses"},
		Namespaced: false,
	})

	r.Register(GVREvents, MetaViewer{
		Aliases:    []string{"ev", "event", "events"},
		Namespaced: true,
	})

	r.Register(GVRServiceAccounts, MetaViewer{
		Aliases:    []string{"sa", "serviceaccount", "serviceaccounts"},
		Namespaced: true,
	})

	r.Register(GVRRoles, MetaViewer{
		Aliases:    []string{"role", "roles"},
		Namespaced: true,
	})

	r.Register(GVRRoleBindings, MetaViewer{
		Aliases:    []string{"rb", "rolebinding", "rolebindings"},
		Namespaced: true,
	})

	r.Register(GVRClusterRoles, MetaViewer{
		Aliases:    []string{"cr", "clusterrole", "clusterroles"},
		Namespaced: false,
	})

	r.Register(GVRClusterRoleBindings, MetaViewer{
		Aliases:    []string{"crb", "clusterrolebinding", "clusterrolebindings"},
		Namespaced: false,
	})

	r.Register(GVRContexts, MetaViewer{
		Aliases:    []string{"ctx", "context", "contexts"},
		Namespaced: false,
	})
}

// Register registers a viewer for a GVR.
func (r *Registrar) Register(gvr GVR, viewer MetaViewer) {
	r.viewers[gvr] = viewer
	for _, alias := range viewer.Aliases {
		r.aliases[alias] = gvr
	}
}

// Get returns the viewer for a GVR.
func (r *Registrar) Get(gvr GVR) (MetaViewer, bool) {
	v, ok := r.viewers[gvr]
	return v, ok
}

// Lookup looks up a GVR by alias or name.
func (r *Registrar) Lookup(name string) (GVR, bool) {
	if gvr, ok := r.aliases[name]; ok {
		return gvr, true
	}
	// Check if it's a full GVR
	if _, ok := r.viewers[GVR(name)]; ok {
		return GVR(name), true
	}
	return "", false
}

// IsNamespaced returns true if the resource is namespaced.
func (r *Registrar) IsNamespaced(gvr GVR) bool {
	v, ok := r.viewers[gvr]
	if !ok {
		return true // Default to namespaced
	}
	return v.Namespaced
}

// AllAliases returns all registered aliases.
func (r *Registrar) AllAliases() []string {
	aliases := make([]string, 0, len(r.aliases))
	for alias := range r.aliases {
		aliases = append(aliases, alias)
	}
	return aliases
}

// AllGVRs returns all registered GVRs.
func (r *Registrar) AllGVRs() []GVR {
	gvrs := make([]GVR, 0, len(r.viewers))
	for gvr := range r.viewers {
		gvrs = append(gvrs, gvr)
	}
	return gvrs
}

// GetRenderer returns the renderer for a GVR.
func (r *Registrar) GetRenderer(gvr GVR) render.Renderer {
	v, ok := r.viewers[gvr]
	if !ok || v.RendererFn == nil {
		return nil
	}
	return v.RendererFn()
}

// GetActions returns the actions for a GVR.
func (r *Registrar) GetActions(gvr GVR) *actions.KeyActions {
	v, ok := r.viewers[gvr]
	if !ok || v.ActionsFn == nil {
		return nil
	}
	return v.ActionsFn()
}
