package ui

import (
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
