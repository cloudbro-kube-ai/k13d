package models

import (
	"context"
	"sync"
	"time"
)

// ResourceModel represents a generic Kubernetes resource model.
type ResourceModel struct {
	*FilteredTable
	gvr         string // Group/Version/Resource identifier (e.g., "v1/pods")
	namespace   string
	lastRefresh time.Time
	refreshMx   sync.RWMutex
	autoRefresh bool
	refreshCh   chan struct{}
	stopCh      chan struct{}
	stopOnce    sync.Once
}

// NewResourceModel creates a new ResourceModel.
func NewResourceModel(gvr, namespace string) *ResourceModel {
	return &ResourceModel{
		FilteredTable: NewFilteredTable(),
		gvr:           gvr,
		namespace:     namespace,
		refreshCh:     make(chan struct{}, 1),
		stopCh:        make(chan struct{}),
	}
}

// GVR returns the Group/Version/Resource identifier.
func (rm *ResourceModel) GVR() string {
	return rm.gvr
}

// Namespace returns the current namespace.
func (rm *ResourceModel) Namespace() string {
	rm.refreshMx.RLock()
	defer rm.refreshMx.RUnlock()
	return rm.namespace
}

// SetNamespace updates the namespace.
func (rm *ResourceModel) SetNamespace(ns string) {
	rm.refreshMx.Lock()
	rm.namespace = ns
	rm.refreshMx.Unlock()
}

// LastRefresh returns the time of the last data refresh.
func (rm *ResourceModel) LastRefresh() time.Time {
	rm.refreshMx.RLock()
	defer rm.refreshMx.RUnlock()
	return rm.lastRefresh
}

// SetLastRefresh updates the last refresh time.
func (rm *ResourceModel) SetLastRefresh(t time.Time) {
	rm.refreshMx.Lock()
	rm.lastRefresh = t
	rm.refreshMx.Unlock()
}

// RequestRefresh requests a data refresh (non-blocking).
func (rm *ResourceModel) RequestRefresh() {
	select {
	case rm.refreshCh <- struct{}{}:
	default:
		// Refresh already pending
	}
}

// RefreshChan returns the refresh request channel.
func (rm *ResourceModel) RefreshChan() <-chan struct{} {
	return rm.refreshCh
}

// SetAutoRefresh enables or disables auto-refresh.
func (rm *ResourceModel) SetAutoRefresh(enabled bool) {
	rm.refreshMx.Lock()
	rm.autoRefresh = enabled
	rm.refreshMx.Unlock()
}

// IsAutoRefresh returns true if auto-refresh is enabled.
func (rm *ResourceModel) IsAutoRefresh() bool {
	rm.refreshMx.RLock()
	defer rm.refreshMx.RUnlock()
	return rm.autoRefresh
}

// Stop stops the resource model (for cleanup).
// Safe to call multiple times.
func (rm *ResourceModel) Stop() {
	rm.stopOnce.Do(func() {
		close(rm.stopCh)
	})
}

// StopChan returns the stop channel.
func (rm *ResourceModel) StopChan() <-chan struct{} {
	return rm.stopCh
}

// ResourceFetcher is the interface for fetching resource data.
type ResourceFetcher interface {
	// Fetch retrieves the resource data.
	Fetch(ctx context.Context, namespace string) (headers []string, rows [][]string, err error)
}

// ClusterModel represents the cluster-level state.
type ClusterModel struct {
	currentContext   string
	currentNamespace string
	namespaces       []string
	contexts         []string
	recentNs         []string
	mx               sync.RWMutex
	listeners        []ClusterListener
}

// ClusterListener is notified when cluster state changes.
type ClusterListener interface {
	ContextChanged(context string)
	NamespaceChanged(namespace string)
	NamespacesUpdated(namespaces []string)
}

// NewClusterModel creates a new ClusterModel.
func NewClusterModel() *ClusterModel {
	return &ClusterModel{
		namespaces: make([]string, 0),
		contexts:   make([]string, 0),
		recentNs:   make([]string, 0),
		listeners:  make([]ClusterListener, 0),
	}
}

// CurrentContext returns the current context.
func (cm *ClusterModel) CurrentContext() string {
	cm.mx.RLock()
	defer cm.mx.RUnlock()
	return cm.currentContext
}

// SetCurrentContext updates the current context.
func (cm *ClusterModel) SetCurrentContext(ctx string) {
	cm.mx.Lock()
	cm.currentContext = ctx
	listeners := make([]ClusterListener, len(cm.listeners))
	copy(listeners, cm.listeners)
	cm.mx.Unlock()

	for _, l := range listeners {
		l.ContextChanged(ctx)
	}
}

// CurrentNamespace returns the current namespace.
func (cm *ClusterModel) CurrentNamespace() string {
	cm.mx.RLock()
	defer cm.mx.RUnlock()
	return cm.currentNamespace
}

// SetCurrentNamespace updates the current namespace.
func (cm *ClusterModel) SetCurrentNamespace(ns string) {
	cm.mx.Lock()
	cm.currentNamespace = ns
	// Add to recent namespaces
	cm.addRecentNs(ns)
	listeners := make([]ClusterListener, len(cm.listeners))
	copy(listeners, cm.listeners)
	cm.mx.Unlock()

	for _, l := range listeners {
		l.NamespaceChanged(ns)
	}
}

// addRecentNs adds a namespace to the recent list (internal, must hold lock).
func (cm *ClusterModel) addRecentNs(ns string) {
	if ns == "" || ns == "all" {
		return
	}
	// Remove if exists
	for i, n := range cm.recentNs {
		if n == ns {
			cm.recentNs = append(cm.recentNs[:i], cm.recentNs[i+1:]...)
			break
		}
	}
	// Add to front
	cm.recentNs = append([]string{ns}, cm.recentNs...)
	// Limit to 10
	if len(cm.recentNs) > 10 {
		cm.recentNs = cm.recentNs[:10]
	}
}

// Namespaces returns the list of namespaces.
func (cm *ClusterModel) Namespaces() []string {
	cm.mx.RLock()
	defer cm.mx.RUnlock()
	result := make([]string, len(cm.namespaces))
	copy(result, cm.namespaces)
	return result
}

// SetNamespaces updates the list of namespaces.
func (cm *ClusterModel) SetNamespaces(ns []string) {
	cm.mx.Lock()
	cm.namespaces = ns
	listeners := make([]ClusterListener, len(cm.listeners))
	copy(listeners, cm.listeners)
	cm.mx.Unlock()

	for _, l := range listeners {
		l.NamespacesUpdated(ns)
	}
}

// RecentNamespaces returns the recent namespaces.
func (cm *ClusterModel) RecentNamespaces() []string {
	cm.mx.RLock()
	defer cm.mx.RUnlock()
	result := make([]string, len(cm.recentNs))
	copy(result, cm.recentNs)
	return result
}

// Contexts returns the list of contexts.
func (cm *ClusterModel) Contexts() []string {
	cm.mx.RLock()
	defer cm.mx.RUnlock()
	result := make([]string, len(cm.contexts))
	copy(result, cm.contexts)
	return result
}

// SetContexts updates the list of contexts.
func (cm *ClusterModel) SetContexts(ctx []string) {
	cm.mx.Lock()
	defer cm.mx.Unlock()
	cm.contexts = ctx
}

// Subscribe adds a listener for cluster state changes.
func (cm *ClusterModel) Subscribe(listener ClusterListener) {
	cm.mx.Lock()
	defer cm.mx.Unlock()
	cm.listeners = append(cm.listeners, listener)
}

// Unsubscribe removes a listener.
func (cm *ClusterModel) Unsubscribe(listener ClusterListener) {
	cm.mx.Lock()
	defer cm.mx.Unlock()
	for i, l := range cm.listeners {
		if l == listener {
			cm.listeners = append(cm.listeners[:i], cm.listeners[i+1:]...)
			return
		}
	}
}
