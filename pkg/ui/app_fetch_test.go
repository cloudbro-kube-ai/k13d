package ui

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestAccessModesToStrings(t *testing.T) {
	tests := []struct {
		name     string
		modes    []corev1.PersistentVolumeAccessMode
		expected []string
	}{
		{
			name:     "ReadWriteOnce",
			modes:    []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			expected: []string{"RWO"},
		},
		{
			name:     "ReadOnlyMany",
			modes:    []corev1.PersistentVolumeAccessMode{corev1.ReadOnlyMany},
			expected: []string{"ROX"},
		},
		{
			name:     "ReadWriteMany",
			modes:    []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			expected: []string{"RWX"},
		},
		{
			name:     "ReadWriteOncePod",
			modes:    []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOncePod},
			expected: []string{"RWOP"},
		},
		{
			name:     "Multiple modes",
			modes:    []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce, corev1.ReadOnlyMany},
			expected: []string{"RWO", "ROX"},
		},
		{
			name:     "Empty",
			modes:    []corev1.PersistentVolumeAccessMode{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := accessModesToStrings(tt.modes)
			if len(result) != len(tt.expected) {
				t.Errorf("accessModesToStrings() returned %d items, want %d", len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("accessModesToStrings()[%d] = %q, want %q", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestFetchNodesShowsRoleAndUsage(t *testing.T) {
	app := NewTestApp(TestAppConfig{
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
		InitialResource:       "nodes",
	})

	headers, rows, err := app.fetchNodes(context.Background())
	if err != nil {
		t.Fatalf("fetchNodes failed: %v", err)
	}

	expectedHeaders := []string{"NAME", "STATUS", "ROLE", "CPU", "MEM", "GPU", "AGE"}
	if len(headers) != len(expectedHeaders) {
		t.Fatalf("headers length = %d, want %d", len(headers), len(expectedHeaders))
	}
	for i, header := range headers {
		if header != expectedHeaders[i] {
			t.Fatalf("headers[%d] = %q, want %q", i, header, expectedHeaders[i])
		}
	}

	if len(rows) < 2 {
		t.Fatalf("expected at least 2 node rows, got %d", len(rows))
	}

	rowsByName := make(map[string][]string, len(rows))
	for _, row := range rows {
		if len(row) != len(expectedHeaders) {
			t.Fatalf("row length = %d, want %d", len(row), len(expectedHeaders))
		}
		rowsByName[row[0]] = row
	}

	controlPlane, ok := rowsByName["node-1"]
	if !ok {
		t.Fatal("expected node-1 row")
	}
	if controlPlane[2] != "control-plane" {
		t.Fatalf("node-1 role = %q, want control-plane", controlPlane[2])
	}
	if !strings.HasPrefix(controlPlane[3], "~1c/4c ") {
		t.Fatalf("node-1 CPU = %q, want estimated CPU usage cell", controlPlane[3])
	}
	if !strings.HasPrefix(controlPlane[4], "~1.8Gi/8Gi ") {
		t.Fatalf("node-1 memory = %q, want estimated memory usage cell", controlPlane[4])
	}
	if controlPlane[5] != "-" {
		t.Fatalf("node-1 GPU = %q, want -", controlPlane[5])
	}

	worker, ok := rowsByName["node-2"]
	if !ok {
		t.Fatal("expected node-2 row")
	}
	if worker[2] != "worker" {
		t.Fatalf("node-2 role = %q, want worker", worker[2])
	}
	if !strings.HasPrefix(worker[3], "~1.5c/8c ") {
		t.Fatalf("node-2 CPU = %q, want estimated CPU usage cell", worker[3])
	}
	if !strings.HasPrefix(worker[4], "~4Gi/16Gi ") {
		t.Fatalf("node-2 memory = %q, want estimated memory usage cell", worker[4])
	}
	if !strings.HasPrefix(worker[5], "1/2 ") {
		t.Fatalf("node-2 GPU = %q, want GPU usage cell", worker[5])
	}
}
