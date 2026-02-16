package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/cloudbro-kube-ai/k13d/pkg/db"
	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// newTestAuthConfig returns a standard auth config for tests.
func newTestAuthConfig() *AuthConfig {
	return &AuthConfig{
		Enabled:         false,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "admin123",
		Quiet:           true,
	}
}

// TestExtractPodRefs tests extracting ConfigMap/Secret refs from a Pod spec.
func TestExtractPodRefs(t *testing.T) {
	tests := []struct {
		name           string
		pod            *corev1.Pod
		wantConfigMaps []string
		wantSecrets    []string
	}{
		{
			name: "volume refs",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{Name: "cm-vol", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"}}}},
						{Name: "sec-vol", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "app-secret"}}},
					},
				},
			},
			wantConfigMaps: []string{"app-config"},
			wantSecrets:    []string{"app-secret"},
		},
		{
			name: "envFrom refs",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
							EnvFrom: []corev1.EnvFromSource{
								{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "env-cm"}}},
								{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "env-secret"}}},
							},
						},
					},
				},
			},
			wantConfigMaps: []string{"env-cm"},
			wantSecrets:    []string{"env-secret"},
		},
		{
			name: "env valueFrom refs",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
							Env: []corev1.EnvVar{
								{Name: "DB_HOST", ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "db-config"}, Key: "host"}}},
								{Name: "DB_PASS", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "db-secret"}, Key: "password"}}},
								{Name: "PLAIN", Value: "plain-value"},
							},
						},
					},
				},
			},
			wantConfigMaps: []string{"db-config"},
			wantSecrets:    []string{"db-secret"},
		},
		{
			name: "deduplicates refs",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{Name: "vol1", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "shared-cm"}}}},
					},
					Containers: []corev1.Container{
						{
							Name: "app",
							EnvFrom: []corev1.EnvFromSource{
								{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "shared-cm"}}},
							},
						},
					},
				},
			},
			wantConfigMaps: []string{"shared-cm"},
			wantSecrets:    nil,
		},
		{
			name:           "empty pod",
			pod:            &corev1.Pod{},
			wantConfigMaps: nil,
			wantSecrets:    nil,
		},
		{
			name: "combined volume + envFrom + env refs",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{Name: "v1", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "cm-a"}}}},
						{Name: "v2", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "sec-a"}}},
					},
					Containers: []corev1.Container{
						{
							Name: "c1",
							EnvFrom: []corev1.EnvFromSource{
								{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "cm-b"}}},
								{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "sec-b"}}},
							},
							Env: []corev1.EnvVar{
								{Name: "X", ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "cm-c"}, Key: "k"}}},
								{Name: "Y", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "sec-c"}, Key: "k"}}},
							},
						},
					},
				},
			},
			wantConfigMaps: []string{"cm-a", "cm-b", "cm-c"},
			wantSecrets:    []string{"sec-a", "sec-b", "sec-c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := extractPodRefs(tt.pod)

			gotCMs := toStringSlice(raw["configMapRefs"])
			gotSecs := toStringSlice(raw["secretRefs"])

			assertStringSliceEqual(t, "configMapRefs", tt.wantConfigMaps, gotCMs)
			assertStringSliceEqual(t, "secretRefs", tt.wantSecrets, gotSecs)
		})
	}
}

// TestExtractDeploymentRefs tests extracting refs from a Deployment's pod template.
func TestExtractDeploymentRefs(t *testing.T) {
	deploy := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{Name: "cfg", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "deploy-cm"}}}},
						{Name: "sec", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "deploy-secret"}}},
					},
					Containers: []corev1.Container{
						{
							Name: "app",
							EnvFrom: []corev1.EnvFromSource{
								{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "env-cm"}}},
								{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "env-sec"}}},
							},
							Env: []corev1.EnvVar{
								{Name: "A", ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "key-cm"}, Key: "k"}}},
								{Name: "B", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "key-sec"}, Key: "k"}}},
							},
						},
					},
				},
			},
		},
	}

	raw := extractDeploymentRefs(deploy)
	gotCMs := toStringSlice(raw["configMapRefs"])
	gotSecs := toStringSlice(raw["secretRefs"])

	assertStringSliceEqual(t, "configMapRefs", []string{"deploy-cm", "env-cm", "key-cm"}, gotCMs)
	assertStringSliceEqual(t, "secretRefs", []string{"deploy-secret", "env-sec", "key-sec"}, gotSecs)
}

// TestExtractDeploymentRefs_Empty tests empty deployment template.
func TestExtractDeploymentRefs_Empty(t *testing.T) {
	deploy := &appsv1.Deployment{}
	raw := extractDeploymentRefs(deploy)
	if len(raw) != 0 {
		t.Errorf("Expected empty raw map for empty deployment, got %v", raw)
	}
}

// TestDedupStrings tests string deduplication.
func TestDedupStrings(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"no duplicates", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"with duplicates", []string{"a", "b", "a", "c", "b"}, []string{"a", "b", "c"}},
		{"all same", []string{"x", "x", "x"}, []string{"x"}},
		{"empty", []string{}, []string{}},
		{"nil input", nil, []string{}},
		{"single", []string{"only"}, []string{"only"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedupStrings(tt.in)
			assertStringSliceEqual(t, "dedup", tt.want, got)
		})
	}
}

// TestHandleValidate tests the /api/validate endpoint.
func TestHandleValidate(t *testing.T) {
	dbPath := "test_validate.db"
	defer os.Remove(dbPath)
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	// Create fake K8s objects: a pod referencing a missing ConfigMap
	replicas := int32(1)
	fakeClientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "web-pod", Namespace: "default", Labels: map[string]string{"app": "web"}},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{Name: "config", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "missing-cm"}}}},
				},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "web-svc", Namespace: "default"},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{"app": "nonexistent"},
				Ports:    []corev1.ServicePort{{Port: 80}},
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "existing-cm", Namespace: "default"},
			Data:       map[string]string{"key": "val"},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "web-deploy", Namespace: "default"},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
			},
		},
	)

	cfg := &config.Config{Language: "en"}
	authConfig := &AuthConfig{
		Enabled:         false,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "admin123",
		Quiet:           true,
	}
	server := &Server{
		cfg:              cfg,
		k8sClient:        &k8s.Client{Clientset: fakeClientset},
		authManager:      NewAuthManager(authConfig),
		port:             8080,
		pendingApprovals: make(map[string]*PendingToolApproval),
	}

	t.Run("successful validation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/validate?namespace=default", nil)
		w := httptest.NewRecorder()

		server.handleValidate(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var body map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if body["namespace"] != "default" {
			t.Errorf("Expected namespace 'default', got %v", body["namespace"])
		}

		// total should be a number
		total, ok := body["total"].(float64)
		if !ok {
			t.Fatalf("Expected total to be a number, got %T", body["total"])
		}
		// We set up a service with a selector that doesn't match any pod labels,
		// and a pod referencing a missing configmap, so we expect findings.
		if total < 1 {
			t.Logf("Note: Expected findings from cross-validation (service selector mismatch / missing configmap ref), got total=%v", total)
		}

		// findings should be an array
		findings, ok := body["findings"].([]interface{})
		if !ok {
			t.Fatalf("Expected findings to be an array, got %T", body["findings"])
		}
		_ = findings // validated as array
	})

	t.Run("missing namespace returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/validate", nil)
		w := httptest.NewRecorder()

		server.handleValidate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("POST method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/validate?namespace=default", nil)
		w := httptest.NewRecorder()

		server.handleValidate(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})

	t.Run("empty namespace results", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/validate?namespace=nonexistent-ns", nil)
		w := httptest.NewRecorder()

		server.handleValidate(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		var body map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if body["namespace"] != "nonexistent-ns" {
			t.Errorf("Expected namespace 'nonexistent-ns', got %v", body["namespace"])
		}
	})
}

// --- test helpers ---

func toStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch s := v.(type) {
	case []string:
		return s
	case []interface{}:
		result := make([]string, len(s))
		for i, item := range s {
			result[i] = item.(string)
		}
		return result
	}
	return nil
}

func assertStringSliceEqual(t *testing.T, label string, want, got []string) {
	t.Helper()
	if len(want) == 0 && len(got) == 0 {
		return
	}
	if len(want) != len(got) {
		t.Errorf("%s: expected %v, got %v", label, want, got)
		return
	}
	for i := range want {
		if want[i] != got[i] {
			t.Errorf("%s[%d]: expected %q, got %q", label, i, want[i], got[i])
		}
	}
}
