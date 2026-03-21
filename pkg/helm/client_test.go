package helm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
)

// TestNewClient verifies constructor defaults and overrides
func TestNewClient(t *testing.T) {
	t.Run("default namespace", func(t *testing.T) {
		c := NewClient("", "")
		if c.namespace != "default" {
			t.Errorf("namespace = %q, want %q", c.namespace, "default")
		}
		if c.settings == nil {
			t.Fatal("settings is nil")
		}
	})

	t.Run("custom namespace", func(t *testing.T) {
		c := NewClient("kube-system", "")
		if c.namespace != "kube-system" {
			t.Errorf("namespace = %q, want %q", c.namespace, "kube-system")
		}
	})

	t.Run("custom kubeconfig", func(t *testing.T) {
		c := NewClient("", "/tmp/kubeconfig")
		if c.kubeconfig != "/tmp/kubeconfig" {
			t.Errorf("kubeconfig = %q, want %q", c.kubeconfig, "/tmp/kubeconfig")
		}
		if c.settings.KubeConfig != "/tmp/kubeconfig" {
			t.Errorf("settings.KubeConfig = %q, want %q", c.settings.KubeConfig, "/tmp/kubeconfig")
		}
	})
}

// TestValuesToYAML tests the values-to-YAML conversion
func TestValuesToYAML(t *testing.T) {
	tests := []struct {
		name    string
		values  map[string]interface{}
		wantErr bool
		check   func(string) bool
	}{
		{
			name:   "simple values",
			values: map[string]interface{}{"replicas": 3, "image": "nginx"},
			check: func(s string) bool {
				return len(s) > 0
			},
		},
		{
			name:   "nested values",
			values: map[string]interface{}{"service": map[string]interface{}{"type": "ClusterIP", "port": 80}},
			check: func(s string) bool {
				return len(s) > 0
			},
		},
		{
			name:   "empty values",
			values: map[string]interface{}{},
			check: func(s string) bool {
				return s == "{}\n"
			},
		},
		{
			name:   "nil values",
			values: nil,
			check: func(s string) bool {
				return s == "null\n"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValuesToYAML(tt.values)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValuesToYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.check != nil && !tt.check(result) {
				t.Errorf("ValuesToYAML() = %q, check failed", result)
			}
		})
	}
}

// TestYAMLToValues tests the YAML-to-values conversion
func TestYAMLToValues(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
		check   func(map[string]interface{}) bool
	}{
		{
			name: "simple values",
			yaml: "replicas: 3\nimage: nginx\n",
			check: func(v map[string]interface{}) bool {
				return v["image"] == "nginx"
			},
		},
		{
			name: "nested values",
			yaml: "service:\n  type: ClusterIP\n  port: 80\n",
			check: func(v map[string]interface{}) bool {
				svc, ok := v["service"].(map[string]interface{})
				return ok && svc["type"] == "ClusterIP"
			},
		},
		{
			name: "empty yaml",
			yaml: "",
			check: func(v map[string]interface{}) bool {
				return len(v) == 0
			},
		},
		{
			name:    "invalid yaml",
			yaml:    ":\n  :\n  bad: [unclosed",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := YAMLToValues(tt.yaml)
			if (err != nil) != tt.wantErr {
				t.Errorf("YAMLToValues() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.check != nil && !tt.check(result) {
				t.Errorf("YAMLToValues() = %v, check failed", result)
			}
		})
	}
}

// TestValuesRoundTrip verifies YAML <-> map conversion roundtrip
func TestValuesRoundTrip(t *testing.T) {
	original := map[string]interface{}{
		"replicas": 3,
		"image":    "nginx:latest",
		"service": map[string]interface{}{
			"type": "ClusterIP",
			"port": 80,
		},
	}

	yamlStr, err := ValuesToYAML(original)
	if err != nil {
		t.Fatalf("ValuesToYAML() error = %v", err)
	}

	restored, err := YAMLToValues(yamlStr)
	if err != nil {
		t.Fatalf("YAMLToValues() error = %v", err)
	}

	if restored["image"] != original["image"] {
		t.Errorf("roundtrip image = %v, want %v", restored["image"], original["image"])
	}
	// sigs.k8s.io/yaml delegates to JSON unmarshaling, which uses float64 for numbers
	restoredReplicas, ok := restored["replicas"].(float64)
	if !ok {
		t.Errorf("roundtrip replicas type = %T, want float64", restored["replicas"])
	} else if restoredReplicas != 3.0 {
		t.Errorf("roundtrip replicas = %v, want 3", restoredReplicas)
	}
}

// TestReleaseJSON verifies Release struct serialization
func TestReleaseJSON(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	r := Release{
		Name:       "my-release",
		Namespace:  "default",
		Revision:   3,
		Status:     "deployed",
		Chart:      "nginx-1.2.3",
		AppVersion: "1.25.0",
		Updated:    now,
		Values:     map[string]interface{}{"replicas": 2},
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded Release
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Name != r.Name {
		t.Errorf("name = %q, want %q", decoded.Name, r.Name)
	}
	if decoded.Namespace != r.Namespace {
		t.Errorf("namespace = %q, want %q", decoded.Namespace, r.Namespace)
	}
	if decoded.Revision != r.Revision {
		t.Errorf("revision = %d, want %d", decoded.Revision, r.Revision)
	}
	if decoded.Status != r.Status {
		t.Errorf("status = %q, want %q", decoded.Status, r.Status)
	}
	if decoded.Chart != r.Chart {
		t.Errorf("chart = %q, want %q", decoded.Chart, r.Chart)
	}
}

// TestReleaseHistoryJSON verifies ReleaseHistory struct serialization
func TestReleaseHistoryJSON(t *testing.T) {
	h := ReleaseHistory{
		Revision:    2,
		Status:      "superseded",
		Chart:       "nginx-1.2.2",
		AppVersion:  "1.24.0",
		Description: "Upgrade complete",
		Updated:     time.Now(),
	}

	data, err := json.Marshal(h)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded ReleaseHistory
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Revision != h.Revision {
		t.Errorf("revision = %d, want %d", decoded.Revision, h.Revision)
	}
	if decoded.Description != h.Description {
		t.Errorf("description = %q, want %q", decoded.Description, h.Description)
	}
}

// TestRepositoryJSON verifies Repository struct serialization
func TestRepositoryJSON(t *testing.T) {
	r := Repository{
		Name: "bitnami",
		URL:  "https://charts.bitnami.com/bitnami",
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded Repository
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Name != r.Name {
		t.Errorf("name = %q, want %q", decoded.Name, r.Name)
	}
	if decoded.URL != r.URL {
		t.Errorf("url = %q, want %q", decoded.URL, r.URL)
	}
}

// TestChartResultJSON verifies ChartResult struct serialization
func TestChartResultJSON(t *testing.T) {
	cr := ChartResult{
		Name:        "bitnami/nginx",
		Version:     "15.0.0",
		AppVersion:  "1.25.0",
		Description: "NGINX web server",
	}

	data, err := json.Marshal(cr)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded ChartResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Name != cr.Name {
		t.Errorf("name = %q, want %q", decoded.Name, cr.Name)
	}
	if decoded.Description != cr.Description {
		t.Errorf("description = %q, want %q", decoded.Description, cr.Description)
	}
}

// TestListRepositories_EmptyFile tests listing repos when no file exists
func TestListRepositories_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()

	c := &Client{
		namespace: "default",
	}
	// Create a settings-like object pointing to temp dir
	c.settings = createTestSettings(tmpDir)

	repos, err := c.ListRepositories()
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}

	if len(repos) != 0 {
		t.Errorf("len(repos) = %d, want 0", len(repos))
	}
}

// TestListRepositories_WithEntries tests listing repos from a valid file
func TestListRepositories_WithEntries(t *testing.T) {
	tmpDir := t.TempDir()
	repoFile := filepath.Join(tmpDir, "repositories.yaml")

	// Write a valid helm repo file
	f := repo.NewFile()
	f.Update(&repo.Entry{Name: "stable", URL: "https://charts.helm.sh/stable"})
	f.Update(&repo.Entry{Name: "bitnami", URL: "https://charts.bitnami.com/bitnami"})
	if err := f.WriteFile(repoFile, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	c := &Client{namespace: "default"}
	c.settings = createTestSettings(tmpDir)

	repos, err := c.ListRepositories()
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}

	if len(repos) != 2 {
		t.Fatalf("len(repos) = %d, want 2", len(repos))
	}

	// Check names (order may vary, so collect into map)
	names := map[string]string{}
	for _, r := range repos {
		names[r.Name] = r.URL
	}
	if names["stable"] != "https://charts.helm.sh/stable" {
		t.Errorf("stable URL = %q, want %q", names["stable"], "https://charts.helm.sh/stable")
	}
	if names["bitnami"] != "https://charts.bitnami.com/bitnami" {
		t.Errorf("bitnami URL = %q, want %q", names["bitnami"], "https://charts.bitnami.com/bitnami")
	}
}

// TestRemoveRepository tests removing a repo from the file
func TestRemoveRepository(t *testing.T) {
	tmpDir := t.TempDir()
	repoFile := filepath.Join(tmpDir, "repositories.yaml")

	f := repo.NewFile()
	f.Update(&repo.Entry{Name: "to-remove", URL: "https://example.com"})
	f.Update(&repo.Entry{Name: "to-keep", URL: "https://keep.example.com"})
	if err := f.WriteFile(repoFile, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	c := &Client{namespace: "default"}
	c.settings = createTestSettings(tmpDir)

	if err := c.RemoveRepository("to-remove"); err != nil {
		t.Fatalf("RemoveRepository() error = %v", err)
	}

	// Verify removal
	repos, err := c.ListRepositories()
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("len(repos) = %d, want 1", len(repos))
	}
	if repos[0].Name != "to-keep" {
		t.Errorf("remaining repo = %q, want %q", repos[0].Name, "to-keep")
	}
}

// TestRemoveRepository_NotFound tests removing a non-existent repo
func TestRemoveRepository_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	repoFile := filepath.Join(tmpDir, "repositories.yaml")

	f := repo.NewFile()
	f.Update(&repo.Entry{Name: "existing", URL: "https://example.com"})
	if err := f.WriteFile(repoFile, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	c := &Client{namespace: "default"}
	c.settings = createTestSettings(tmpDir)

	err := c.RemoveRepository("nonexistent")
	if err == nil {
		t.Error("RemoveRepository() expected error for nonexistent repo, got nil")
	}
}

// TestInstallOptions verifies option structs
func TestInstallOptions(t *testing.T) {
	opts := &InstallOptions{
		CreateNamespace: true,
		Wait:            true,
		Timeout:         10 * time.Minute,
	}

	if !opts.CreateNamespace {
		t.Error("CreateNamespace = false, want true")
	}
	if opts.Timeout != 10*time.Minute {
		t.Errorf("Timeout = %v, want %v", opts.Timeout, 10*time.Minute)
	}
}

// TestUpgradeOptions verifies option structs
func TestUpgradeOptions(t *testing.T) {
	opts := &UpgradeOptions{
		Wait:        true,
		ReuseValues: true,
		ResetValues: false,
		Timeout:     3 * time.Minute,
	}

	if !opts.ReuseValues {
		t.Error("ReuseValues = false, want true")
	}
	if opts.ResetValues {
		t.Error("ResetValues = true, want false")
	}
}

// TestInterfaceCompliance verifies Client implements HelmClient
func TestInterfaceCompliance(t *testing.T) {
	var _ HelmClient = (*Client)(nil)
}

// createTestSettings creates a cli.EnvSettings-like configuration pointing to tmpDir
func createTestSettings(tmpDir string) *cli.EnvSettings {
	settings := cli.New()
	settings.RepositoryConfig = filepath.Join(tmpDir, "repositories.yaml")
	settings.RepositoryCache = filepath.Join(tmpDir, "cache")
	_ = os.MkdirAll(filepath.Join(tmpDir, "cache"), 0755)
	return settings
}
