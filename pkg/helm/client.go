package helm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"
)

// Client wraps Helm SDK for release management
type Client struct {
	settings   *cli.EnvSettings
	namespace  string
	kubeconfig string
}

// Release represents a Helm release
type Release struct {
	Name       string                 `json:"name"`
	Namespace  string                 `json:"namespace"`
	Revision   int                    `json:"revision"`
	Status     string                 `json:"status"`
	Chart      string                 `json:"chart"`
	AppVersion string                 `json:"appVersion"`
	Updated    time.Time              `json:"updated"`
	Values     map[string]interface{} `json:"values,omitempty"`
}

// ReleaseHistory represents a release revision
type ReleaseHistory struct {
	Revision    int       `json:"revision"`
	Status      string    `json:"status"`
	Chart       string    `json:"chart"`
	AppVersion  string    `json:"appVersion"`
	Description string    `json:"description"`
	Updated     time.Time `json:"updated"`
}

// Repository represents a Helm repository
type Repository struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// NewClient creates a new Helm client
func NewClient(namespace string, kubeconfig string) *Client {
	settings := cli.New()
	if kubeconfig != "" {
		settings.KubeConfig = kubeconfig
	}
	if namespace == "" {
		namespace = "default"
	}

	return &Client{
		settings:   settings,
		namespace:  namespace,
		kubeconfig: kubeconfig,
	}
}

// getActionConfig creates a new action configuration
func (c *Client) getActionConfig(namespace string) (*action.Configuration, error) {
	if namespace == "" {
		namespace = c.namespace
	}

	actionConfig := new(action.Configuration)
	err := actionConfig.Init(c.settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) {
		// Log function - silently ignore for now
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize action config: %w", err)
	}
	return actionConfig, nil
}

// ListReleases lists all Helm releases
func (c *Client) ListReleases(ctx context.Context, namespace string, allNamespaces bool) ([]Release, error) {
	ns := namespace
	if allNamespaces {
		ns = ""
	}

	actionConfig, err := c.getActionConfig(ns)
	if err != nil {
		return nil, err
	}

	listAction := action.NewList(actionConfig)
	listAction.AllNamespaces = allNamespaces
	if !allNamespaces && namespace != "" {
		listAction.AllNamespaces = false
	}

	results, err := listAction.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to list releases: %w", err)
	}

	releases := make([]Release, 0, len(results))
	for _, r := range results {
		releases = append(releases, Release{
			Name:       r.Name,
			Namespace:  r.Namespace,
			Revision:   r.Version,
			Status:     string(r.Info.Status),
			Chart:      r.Chart.Metadata.Name + "-" + r.Chart.Metadata.Version,
			AppVersion: r.Chart.Metadata.AppVersion,
			Updated:    r.Info.LastDeployed.Time,
		})
	}

	return releases, nil
}

// GetRelease gets details of a specific release
func (c *Client) GetRelease(ctx context.Context, name string, namespace string) (*Release, error) {
	actionConfig, err := c.getActionConfig(namespace)
	if err != nil {
		return nil, err
	}

	getAction := action.NewGet(actionConfig)
	r, err := getAction.Run(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get release %s: %w", name, err)
	}

	return &Release{
		Name:       r.Name,
		Namespace:  r.Namespace,
		Revision:   r.Version,
		Status:     string(r.Info.Status),
		Chart:      r.Chart.Metadata.Name + "-" + r.Chart.Metadata.Version,
		AppVersion: r.Chart.Metadata.AppVersion,
		Updated:    r.Info.LastDeployed.Time,
		Values:     r.Config,
	}, nil
}

// GetReleaseHistory gets the revision history of a release
func (c *Client) GetReleaseHistory(ctx context.Context, name string, namespace string) ([]ReleaseHistory, error) {
	actionConfig, err := c.getActionConfig(namespace)
	if err != nil {
		return nil, err
	}

	historyAction := action.NewHistory(actionConfig)
	historyAction.Max = 10 // Limit to last 10 revisions

	results, err := historyAction.Run(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get release history: %w", err)
	}

	history := make([]ReleaseHistory, 0, len(results))
	for _, r := range results {
		history = append(history, ReleaseHistory{
			Revision:    r.Version,
			Status:      string(r.Info.Status),
			Chart:       r.Chart.Metadata.Name + "-" + r.Chart.Metadata.Version,
			AppVersion:  r.Chart.Metadata.AppVersion,
			Description: r.Info.Description,
			Updated:     r.Info.LastDeployed.Time,
		})
	}

	return history, nil
}

// GetReleaseValues gets the values of a release
func (c *Client) GetReleaseValues(ctx context.Context, name string, namespace string, allValues bool) (map[string]interface{}, error) {
	actionConfig, err := c.getActionConfig(namespace)
	if err != nil {
		return nil, err
	}

	getValues := action.NewGetValues(actionConfig)
	getValues.AllValues = allValues

	values, err := getValues.Run(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get release values: %w", err)
	}

	return values, nil
}

// InstallRelease installs a Helm chart
func (c *Client) InstallRelease(ctx context.Context, name string, chartRef string, namespace string, values map[string]interface{}, opts *InstallOptions) (*Release, error) {
	actionConfig, err := c.getActionConfig(namespace)
	if err != nil {
		return nil, err
	}

	installAction := action.NewInstall(actionConfig)
	installAction.ReleaseName = name
	installAction.Namespace = namespace
	installAction.CreateNamespace = opts != nil && opts.CreateNamespace
	installAction.Wait = opts != nil && opts.Wait
	installAction.Timeout = 5 * time.Minute
	if opts != nil && opts.Timeout > 0 {
		installAction.Timeout = opts.Timeout
	}

	// Locate chart
	chartPath, err := installAction.ChartPathOptions.LocateChart(chartRef, c.settings)
	if err != nil {
		return nil, fmt.Errorf("failed to locate chart: %w", err)
	}

	// Load chart
	ch, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart: %w", err)
	}

	// Install
	r, err := installAction.Run(ch, values)
	if err != nil {
		return nil, fmt.Errorf("failed to install release: %w", err)
	}

	return &Release{
		Name:       r.Name,
		Namespace:  r.Namespace,
		Revision:   r.Version,
		Status:     string(r.Info.Status),
		Chart:      r.Chart.Metadata.Name + "-" + r.Chart.Metadata.Version,
		AppVersion: r.Chart.Metadata.AppVersion,
		Updated:    r.Info.LastDeployed.Time,
	}, nil
}

// UpgradeRelease upgrades an existing release
func (c *Client) UpgradeRelease(ctx context.Context, name string, chartRef string, namespace string, values map[string]interface{}, opts *UpgradeOptions) (*Release, error) {
	actionConfig, err := c.getActionConfig(namespace)
	if err != nil {
		return nil, err
	}

	upgradeAction := action.NewUpgrade(actionConfig)
	upgradeAction.Namespace = namespace
	upgradeAction.Wait = opts != nil && opts.Wait
	upgradeAction.ReuseValues = opts != nil && opts.ReuseValues
	upgradeAction.ResetValues = opts != nil && opts.ResetValues
	upgradeAction.Timeout = 5 * time.Minute
	if opts != nil && opts.Timeout > 0 {
		upgradeAction.Timeout = opts.Timeout
	}

	// Locate chart
	chartPath, err := upgradeAction.ChartPathOptions.LocateChart(chartRef, c.settings)
	if err != nil {
		return nil, fmt.Errorf("failed to locate chart: %w", err)
	}

	// Load chart
	ch, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart: %w", err)
	}

	// Upgrade
	r, err := upgradeAction.Run(name, ch, values)
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade release: %w", err)
	}

	return &Release{
		Name:       r.Name,
		Namespace:  r.Namespace,
		Revision:   r.Version,
		Status:     string(r.Info.Status),
		Chart:      r.Chart.Metadata.Name + "-" + r.Chart.Metadata.Version,
		AppVersion: r.Chart.Metadata.AppVersion,
		Updated:    r.Info.LastDeployed.Time,
	}, nil
}

// UninstallRelease uninstalls a release
func (c *Client) UninstallRelease(ctx context.Context, name string, namespace string, keepHistory bool) error {
	actionConfig, err := c.getActionConfig(namespace)
	if err != nil {
		return err
	}

	uninstallAction := action.NewUninstall(actionConfig)
	uninstallAction.KeepHistory = keepHistory

	_, err = uninstallAction.Run(name)
	if err != nil {
		return fmt.Errorf("failed to uninstall release: %w", err)
	}

	return nil
}

// RollbackRelease rolls back a release to a previous revision
func (c *Client) RollbackRelease(ctx context.Context, name string, namespace string, revision int) error {
	actionConfig, err := c.getActionConfig(namespace)
	if err != nil {
		return err
	}

	rollbackAction := action.NewRollback(actionConfig)
	rollbackAction.Version = revision
	rollbackAction.Wait = true
	rollbackAction.Timeout = 5 * time.Minute

	err = rollbackAction.Run(name)
	if err != nil {
		return fmt.Errorf("failed to rollback release: %w", err)
	}

	return nil
}

// ==========================================
// Repository Management
// ==========================================

// ListRepositories lists all configured Helm repositories
func (c *Client) ListRepositories() ([]Repository, error) {
	repoFile := c.settings.RepositoryConfig
	f, err := repo.LoadFile(repoFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Repository{}, nil
		}
		return nil, fmt.Errorf("failed to load repository file: %w", err)
	}

	repos := make([]Repository, 0, len(f.Repositories))
	for _, r := range f.Repositories {
		repos = append(repos, Repository{
			Name: r.Name,
			URL:  r.URL,
		})
	}

	return repos, nil
}

// AddRepository adds a new Helm repository
func (c *Client) AddRepository(name string, url string) error {
	repoFile := c.settings.RepositoryConfig

	// Ensure directory exists
	err := os.MkdirAll(filepath.Dir(repoFile), 0755)
	if err != nil {
		return fmt.Errorf("failed to create repository config directory: %w", err)
	}

	// Load or create repo file
	f, err := repo.LoadFile(repoFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load repository file: %w", err)
	}
	if f == nil {
		f = repo.NewFile()
	}

	// Check if repo already exists
	if f.Has(name) {
		return fmt.Errorf("repository %s already exists", name)
	}

	// Create entry
	entry := &repo.Entry{
		Name: name,
		URL:  url,
	}

	// Download and cache index
	r, err := repo.NewChartRepository(entry, nil)
	if err != nil {
		return fmt.Errorf("failed to create chart repository: %w", err)
	}

	cacheDir := c.settings.RepositoryCache
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "helm-cache")
	}
	r.CachePath = cacheDir

	_, err = r.DownloadIndexFile()
	if err != nil {
		return fmt.Errorf("failed to download index file: %w", err)
	}

	// Add to file and save
	f.Update(entry)
	if err := f.WriteFile(repoFile, 0644); err != nil {
		return fmt.Errorf("failed to write repository file: %w", err)
	}

	return nil
}

// RemoveRepository removes a Helm repository
func (c *Client) RemoveRepository(name string) error {
	repoFile := c.settings.RepositoryConfig

	f, err := repo.LoadFile(repoFile)
	if err != nil {
		return fmt.Errorf("failed to load repository file: %w", err)
	}

	if !f.Remove(name) {
		return fmt.Errorf("repository %s not found", name)
	}

	if err := f.WriteFile(repoFile, 0644); err != nil {
		return fmt.Errorf("failed to write repository file: %w", err)
	}

	return nil
}

// UpdateRepositories updates all repository indices
func (c *Client) UpdateRepositories() error {
	repoFile := c.settings.RepositoryConfig
	f, err := repo.LoadFile(repoFile)
	if err != nil {
		return fmt.Errorf("failed to load repository file: %w", err)
	}

	cacheDir := c.settings.RepositoryCache
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "helm-cache")
	}

	for _, entry := range f.Repositories {
		r, err := repo.NewChartRepository(entry, nil)
		if err != nil {
			continue // Skip failed repos
		}
		r.CachePath = cacheDir
		r.DownloadIndexFile()
	}

	return nil
}

// ==========================================
// Options and helpers
// ==========================================

// InstallOptions contains options for install
type InstallOptions struct {
	CreateNamespace bool
	Wait            bool
	Timeout         time.Duration
}

// UpgradeOptions contains options for upgrade
type UpgradeOptions struct {
	Wait        bool
	ReuseValues bool
	ResetValues bool
	Timeout     time.Duration
}

// GetReleaseManifest gets the rendered manifests of a release
func (c *Client) GetReleaseManifest(ctx context.Context, name string, namespace string) (string, error) {
	actionConfig, err := c.getActionConfig(namespace)
	if err != nil {
		return "", err
	}

	getAction := action.NewGet(actionConfig)
	r, err := getAction.Run(name)
	if err != nil {
		return "", fmt.Errorf("failed to get release: %w", err)
	}

	return r.Manifest, nil
}

// GetReleaseNotes gets the notes of a release
func (c *Client) GetReleaseNotes(ctx context.Context, name string, namespace string) (string, error) {
	actionConfig, err := c.getActionConfig(namespace)
	if err != nil {
		return "", err
	}

	getAction := action.NewGet(actionConfig)
	r, err := getAction.Run(name)
	if err != nil {
		return "", fmt.Errorf("failed to get release: %w", err)
	}

	return r.Info.Notes, nil
}

// SearchCharts searches for charts in configured repositories
func (c *Client) SearchCharts(keyword string) ([]ChartResult, error) {
	// Load repo index files
	repoFile := c.settings.RepositoryConfig
	f, err := repo.LoadFile(repoFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load repository file: %w", err)
	}

	cacheDir := c.settings.RepositoryCache
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "helm-cache")
	}

	results := make([]ChartResult, 0)
	keyword = strings.ToLower(keyword)

	for _, entry := range f.Repositories {
		indexPath := filepath.Join(cacheDir, fmt.Sprintf("%s-index.yaml", entry.Name))
		indexFile, err := repo.LoadIndexFile(indexPath)
		if err != nil {
			continue
		}

		for chartName, versions := range indexFile.Entries {
			if len(versions) == 0 {
				continue
			}
			// Search in name and description
			if strings.Contains(strings.ToLower(chartName), keyword) ||
				strings.Contains(strings.ToLower(versions[0].Description), keyword) {
				results = append(results, ChartResult{
					Name:        entry.Name + "/" + chartName,
					Version:     versions[0].Version,
					AppVersion:  versions[0].AppVersion,
					Description: versions[0].Description,
				})
			}
		}
	}

	return results, nil
}

// ChartResult represents a chart search result
type ChartResult struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	AppVersion  string `json:"appVersion"`
	Description string `json:"description"`
}

// ValuesToYAML converts values map to YAML string
func ValuesToYAML(values map[string]interface{}) (string, error) {
	data, err := yaml.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// YAMLToValues converts YAML string to values map
func YAMLToValues(yamlStr string) (map[string]interface{}, error) {
	values := make(map[string]interface{})
	err := yaml.Unmarshal([]byte(yamlStr), &values)
	if err != nil {
		return nil, err
	}
	return values, nil
}

// Ensure Client implements interface
var _ HelmClient = (*Client)(nil)

// HelmClient interface for Helm operations
type HelmClient interface {
	ListReleases(ctx context.Context, namespace string, allNamespaces bool) ([]Release, error)
	GetRelease(ctx context.Context, name string, namespace string) (*Release, error)
	GetReleaseHistory(ctx context.Context, name string, namespace string) ([]ReleaseHistory, error)
	GetReleaseValues(ctx context.Context, name string, namespace string, allValues bool) (map[string]interface{}, error)
	InstallRelease(ctx context.Context, name string, chartRef string, namespace string, values map[string]interface{}, opts *InstallOptions) (*Release, error)
	UpgradeRelease(ctx context.Context, name string, chartRef string, namespace string, values map[string]interface{}, opts *UpgradeOptions) (*Release, error)
	UninstallRelease(ctx context.Context, name string, namespace string, keepHistory bool) error
	RollbackRelease(ctx context.Context, name string, namespace string, revision int) error
	ListRepositories() ([]Repository, error)
	AddRepository(name string, url string) error
	RemoveRepository(name string) error
	UpdateRepositories() error
}
