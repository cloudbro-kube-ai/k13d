package safety

import (
	"testing"
)

func TestClassifier_Classify(t *testing.T) {
	c := NewClassifier()

	tests := []struct {
		name             string
		command          string
		expectedCategory string
		requiresApproval bool
		isDangerous      bool
	}{
		// Read-only commands
		{
			name:             "kubectl get pods",
			command:          "kubectl get pods",
			expectedCategory: "read-only",
			requiresApproval: false,
			isDangerous:      false,
		},
		{
			name:             "kubectl describe deployment",
			command:          "kubectl describe deployment nginx",
			expectedCategory: "read-only",
			requiresApproval: false,
			isDangerous:      false,
		},
		{
			name:             "kubectl logs",
			command:          "kubectl logs pod/nginx",
			expectedCategory: "read-only",
			requiresApproval: false,
			isDangerous:      false,
		},

		// Write commands
		{
			name:             "kubectl apply",
			command:          "kubectl apply -f deployment.yaml",
			expectedCategory: "write",
			requiresApproval: true,
			isDangerous:      false,
		},
		{
			name:             "kubectl create namespace",
			command:          "kubectl create namespace test",
			expectedCategory: "write",
			requiresApproval: true,
			isDangerous:      false,
		},
		{
			name:             "kubectl scale",
			command:          "kubectl scale deployment nginx --replicas=3",
			expectedCategory: "write",
			requiresApproval: true,
			isDangerous:      false,
		},

		// Dangerous commands
		{
			name:             "kubectl delete with --all",
			command:          "kubectl delete pods --all",
			expectedCategory: "dangerous",
			requiresApproval: true,
			isDangerous:      true,
		},
		{
			name:             "kubectl delete with --force",
			command:          "kubectl delete pod nginx --force --grace-period=0",
			expectedCategory: "dangerous",
			requiresApproval: true,
			isDangerous:      true,
		},
		{
			name:             "kubectl drain",
			command:          "kubectl drain node-1",
			expectedCategory: "dangerous",
			requiresApproval: true,
			isDangerous:      true,
		},
		{
			name:             "kubectl delete namespace",
			command:          "kubectl delete namespace production",
			expectedCategory: "write",
			requiresApproval: true,
			isDangerous:      true, // Namespace deletion is flagged as dangerous
		},

		// Piped commands (should require approval regardless of base command)
		// Note: Category is determined by the FIRST command in the pipe,
		// but RequiresApproval is true because piping is detected
		{
			name:             "piped kubectl get",
			command:          "kubectl get pods | grep nginx",
			expectedCategory: "unknown", // Last command (grep) determines type when piped
			requiresApproval: true,      // Piped commands always require approval
			isDangerous:      false,
		},
		{
			name:             "piped with dangerous command",
			command:          "kubectl get pods | xargs rm -rf",
			expectedCategory: "unknown", // Last command determines type
			requiresApproval: true,
			isDangerous:      false,
		},

		// Chained commands
		{
			name:             "chained commands",
			command:          "kubectl get pods && kubectl delete pod nginx",
			expectedCategory: "read-only",
			requiresApproval: true, // Chained commands always require approval
			isDangerous:      false,
		},

		// Unknown commands
		{
			name:             "custom script",
			command:          "./my-script.sh",
			expectedCategory: "unknown",
			requiresApproval: true,
			isDangerous:      false,
		},
		{
			name:             "python command",
			command:          "python3 deploy.py",
			expectedCategory: "unknown",
			requiresApproval: true,
			isDangerous:      false,
		},

		// Helm commands
		{
			name:             "helm list",
			command:          "helm list",
			expectedCategory: "read-only",
			requiresApproval: false,
			isDangerous:      false,
		},
		{
			name:             "helm install",
			command:          "helm install nginx bitnami/nginx",
			expectedCategory: "write",
			requiresApproval: true,
			isDangerous:      false,
		},

		// Interactive commands
		{
			name:             "kubectl exec",
			command:          "kubectl exec -it nginx -- bash",
			expectedCategory: "interactive",
			requiresApproval: true,
			isDangerous:      false,
		},
		{
			name:             "kubectl port-forward",
			command:          "kubectl port-forward svc/nginx 8080:80",
			expectedCategory: "interactive",
			requiresApproval: true,
			isDangerous:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Classify(tt.command)

			if result.Category != tt.expectedCategory {
				t.Errorf("Category = %q, want %q", result.Category, tt.expectedCategory)
			}

			if result.RequiresApproval != tt.requiresApproval {
				t.Errorf("RequiresApproval = %v, want %v", result.RequiresApproval, tt.requiresApproval)
			}

			if result.IsDangerous != tt.isDangerous {
				t.Errorf("IsDangerous = %v, want %v", result.IsDangerous, tt.isDangerous)
			}
		})
	}
}

func TestClassifier_ClassifyQuick(t *testing.T) {
	c := NewClassifier()

	tests := []struct {
		name        string
		command     string
		isReadOnly  bool
		isDangerous bool
	}{
		{
			name:        "kubectl get pods",
			command:     "kubectl get pods",
			isReadOnly:  true,
			isDangerous: false,
		},
		{
			name:        "kubectl delete with --all",
			command:     "kubectl delete pods --all",
			isReadOnly:  false,
			isDangerous: true,
		},
		{
			name:        "kubectl drain",
			command:     "kubectl drain node-1",
			isReadOnly:  false,
			isDangerous: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.ClassifyQuick(tt.command)

			if result.IsReadOnly != tt.isReadOnly {
				t.Errorf("IsReadOnly = %v, want %v", result.IsReadOnly, tt.isReadOnly)
			}

			if result.IsDangerous != tt.isDangerous {
				t.Errorf("IsDangerous = %v, want %v", result.IsDangerous, tt.isDangerous)
			}
		})
	}
}

func TestDefaultClassifier(t *testing.T) {
	// Test package-level convenience function
	result := Classify("kubectl get pods")

	if result.Category != "read-only" {
		t.Errorf("Expected read-only category, got %s", result.Category)
	}

	if result.RequiresApproval {
		t.Error("Expected no approval required for kubectl get pods")
	}
}
