package safety

import (
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name        string
		cmd         string
		wantProgram string
		wantVerb    string
		wantPiped   bool
		wantChained bool
	}{
		{
			name:        "simple kubectl get",
			cmd:         "kubectl get pods",
			wantProgram: "kubectl",
			wantVerb:    "get",
		},
		{
			name:        "kubectl with namespace",
			cmd:         "kubectl get pods -n kube-system",
			wantProgram: "kubectl",
			wantVerb:    "get",
		},
		{
			name:        "kubectl delete",
			cmd:         "kubectl delete pod nginx",
			wantProgram: "kubectl",
			wantVerb:    "delete",
		},
		{
			name:        "piped command detects pipe",
			cmd:         "kubectl get pods | grep nginx",
			wantProgram: "grep", // Parser returns the last command in pipe
			wantVerb:    "",     // grep is not kubectl, no verb
			wantPiped:   true,
		},
		{
			name:        "chained command",
			cmd:         "kubectl get pods && kubectl get services",
			wantProgram: "kubectl",
			wantVerb:    "get",
			wantChained: true,
		},
		{
			name:        "helm command",
			cmd:         "helm list -A",
			wantProgram: "helm",
			wantVerb:    "list", // Verb extraction works for helm too
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := ParseCommand(tt.cmd)

			if parsed.Program != tt.wantProgram {
				t.Errorf("Program = %s, want %s", parsed.Program, tt.wantProgram)
			}
			if parsed.Verb != tt.wantVerb {
				t.Errorf("Verb = %s, want %s", parsed.Verb, tt.wantVerb)
			}
			if parsed.IsPiped != tt.wantPiped {
				t.Errorf("IsPiped = %v, want %v", parsed.IsPiped, tt.wantPiped)
			}
			if parsed.IsChained != tt.wantChained {
				t.Errorf("IsChained = %v, want %v", parsed.IsChained, tt.wantChained)
			}
		})
	}
}

func TestParseCommandNamespace(t *testing.T) {
	tests := []struct {
		name          string
		cmd           string
		wantNamespace string
	}{
		{
			name:          "short namespace flag",
			cmd:           "kubectl get pods -n kube-system",
			wantNamespace: "kube-system",
		},
		{
			name:          "long namespace flag",
			cmd:           "kubectl get pods --namespace=default",
			wantNamespace: "default",
		},
		{
			name:          "no namespace",
			cmd:           "kubectl get pods",
			wantNamespace: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := ParseCommand(tt.cmd)
			if parsed.Namespace != tt.wantNamespace {
				t.Errorf("Namespace = %s, want %s", parsed.Namespace, tt.wantNamespace)
			}
		})
	}
}

func TestAnalyzerReadOnly(t *testing.T) {
	analyzer := NewAnalyzer()

	readOnlyCommands := []string{
		"kubectl get pods",
		"kubectl describe deployment nginx",
		"kubectl logs nginx",
		"kubectl explain pods",
		"kubectl top nodes",
		"kubectl version",
		"kubectl api-resources",
		"kubectl cluster-info",
	}

	for _, cmd := range readOnlyCommands {
		t.Run(cmd, func(t *testing.T) {
			report := analyzer.Analyze(cmd)
			if !report.IsReadOnly {
				t.Errorf("Expected %s to be read-only", cmd)
			}
			if report.RequiresApproval {
				t.Errorf("Read-only command %s should not require approval", cmd)
			}
		})
	}
}

func TestAnalyzerWriteCommands(t *testing.T) {
	analyzer := NewAnalyzer()

	writeCommands := []string{
		"kubectl create deployment nginx --image=nginx",
		"kubectl apply -f deployment.yaml",
		"kubectl delete pod nginx",
		"kubectl scale deployment nginx --replicas=3",
		"kubectl patch deployment nginx -p '{}'",
		"kubectl label pod nginx env=prod",
	}

	for _, cmd := range writeCommands {
		t.Run(cmd, func(t *testing.T) {
			report := analyzer.Analyze(cmd)
			if report.IsReadOnly {
				t.Errorf("Expected %s to NOT be read-only", cmd)
			}
			if !report.RequiresApproval {
				t.Errorf("Write command %s should require approval", cmd)
			}
		})
	}
}

func TestAnalyzerDangerousCommands(t *testing.T) {
	analyzer := NewAnalyzer()

	dangerousCommands := []string{
		"kubectl delete pods --all",
		"kubectl delete namespace production",
		"kubectl drain node01 --force",
		"kubectl delete deployment nginx --force --grace-period=0",
		"kubectl cordon node01",
	}

	for _, cmd := range dangerousCommands {
		t.Run(cmd, func(t *testing.T) {
			report := analyzer.Analyze(cmd)
			if !report.IsDangerous {
				t.Errorf("Expected %s to be dangerous", cmd)
			}
			if !report.RequiresApproval {
				t.Errorf("Dangerous command %s should require approval", cmd)
			}
			if len(report.Warnings) == 0 {
				t.Errorf("Dangerous command %s should have warnings", cmd)
			}
		})
	}
}

func TestAnalyzerInteractiveCommands(t *testing.T) {
	analyzer := NewAnalyzer()

	interactiveCommands := []string{
		"kubectl exec -it nginx -- /bin/bash",
		"kubectl port-forward svc/nginx 8080:80",
	}

	for _, cmd := range interactiveCommands {
		t.Run(cmd, func(t *testing.T) {
			report := analyzer.Analyze(cmd)
			if !report.IsInteractive {
				t.Errorf("Expected %s to be interactive", cmd)
			}
			if !report.RequiresApproval {
				t.Errorf("Interactive command %s should require approval", cmd)
			}
		})
	}
}

func TestAnalyzerPipedCommands(t *testing.T) {
	analyzer := NewAnalyzer()

	cmd := "kubectl get pods | grep nginx"
	report := analyzer.Analyze(cmd)

	if !report.RequiresApproval {
		t.Error("Piped command should require approval")
	}

	hasWarning := false
	for _, w := range report.Warnings {
		if w == "Piped command detected" {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("Piped command should have warning")
	}
}

func TestAnalyzerHelmCommands(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		cmd           string
		isReadOnly    bool
		needsApproval bool
	}{
		{"helm list", true, false},
		{"helm status nginx", true, false},
		{"helm install nginx bitnami/nginx", false, true},
		{"helm upgrade nginx bitnami/nginx", false, true},
		{"helm uninstall nginx", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			report := analyzer.Analyze(tt.cmd)
			if report.IsReadOnly != tt.isReadOnly {
				t.Errorf("IsReadOnly = %v, want %v", report.IsReadOnly, tt.isReadOnly)
			}
			if report.RequiresApproval != tt.needsApproval {
				t.Errorf("RequiresApproval = %v, want %v", report.RequiresApproval, tt.needsApproval)
			}
		})
	}
}

func TestQuickCheck(t *testing.T) {
	tests := []struct {
		cmd           string
		wantReadOnly  bool
		wantDangerous bool
	}{
		{"kubectl get pods", true, false},
		{"kubectl describe node", true, false},
		{"kubectl delete pod nginx", false, true},
		{"kubectl delete pods --all", false, true},
		{"kubectl drain node --force", false, true},
		{"rm -rf /", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			isReadOnly, isDangerous := QuickCheck(tt.cmd)
			if isReadOnly != tt.wantReadOnly {
				t.Errorf("isReadOnly = %v, want %v", isReadOnly, tt.wantReadOnly)
			}
			if isDangerous != tt.wantDangerous {
				t.Errorf("isDangerous = %v, want %v", isDangerous, tt.wantDangerous)
			}
		})
	}
}

func TestParsedCommandFlags(t *testing.T) {
	cmd := "kubectl delete pods nginx --force --grace-period=0 -n default"
	parsed := ParseCommand(cmd)

	if !parsed.HasFlag("--force") {
		t.Error("Should have --force flag")
	}

	// --grace-period=0 is stored as --grace-period with value "0"
	if !parsed.HasFlag("--grace-period") {
		t.Error("Should have --grace-period flag")
	}

	if parsed.GetFlagValue("--grace-period") != "0" {
		t.Errorf("--grace-period value = %s, want 0", parsed.GetFlagValue("--grace-period"))
	}

	if !parsed.HasFlag("-n") {
		t.Error("Should have -n flag")
	}

	if parsed.HasFlag("--nonexistent") {
		t.Error("Should not have --nonexistent flag")
	}

	if !parsed.HasAnyFlag("--force", "--all") {
		t.Error("HasAnyFlag should return true for --force")
	}
}

func TestParseCommandRedirects(t *testing.T) {
	tests := []struct {
		name        string
		cmd         string
		hasRedirect bool
	}{
		{"no redirect", "kubectl get pods", false},
		{"output redirect", "kubectl get pods > output.txt", true},
		{"append redirect", "kubectl get pods >> output.txt", true},
		{"input redirect", "kubectl apply -f < input.yaml", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := ParseCommand(tt.cmd)
			if parsed.HasRedirect != tt.hasRedirect {
				t.Errorf("HasRedirect = %v, want %v", parsed.HasRedirect, tt.hasRedirect)
			}
		})
	}
}

func TestParseCommandResource(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		resource string
	}{
		{"pods resource", "kubectl get pods nginx", "pods"},
		{"deployment resource", "kubectl delete deployment nginx", "deployment"},
		{"service resource", "kubectl describe svc nginx-service", "svc"},
		{"no resource", "kubectl get", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := ParseCommand(tt.cmd)
			if parsed.Resource != tt.resource {
				t.Errorf("Resource = %s, want %s", parsed.Resource, tt.resource)
			}
		})
	}
}

func TestAnalyzerNonKubectlCommands(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		cmd              string
		requiresApproval bool
	}{
		{"ls -la", true},
		{"cat /etc/passwd", true},
		{"rm -rf /tmp/test", true},
		{"echo hello", true},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			report := analyzer.Analyze(tt.cmd)
			if report.RequiresApproval != tt.requiresApproval {
				t.Errorf("RequiresApproval = %v, want %v", report.RequiresApproval, tt.requiresApproval)
			}
		})
	}
}

func TestAnalyzerEmptyCommand(t *testing.T) {
	analyzer := NewAnalyzer()

	report := analyzer.Analyze("")
	if report.Parsed == nil {
		t.Error("Parsed should not be nil for empty command")
	}
}

func TestAnalyzerDeleteNamespace(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		cmd         string
		isDangerous bool
		hasWarning  bool
	}{
		{"kubectl delete namespace production", true, true},
		{"kubectl delete ns staging", true, true},
		{"kubectl delete pod nginx", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			report := analyzer.Analyze(tt.cmd)
			if report.IsDangerous != tt.isDangerous {
				t.Errorf("IsDangerous = %v, want %v", report.IsDangerous, tt.isDangerous)
			}

			hasNsWarning := false
			for _, w := range report.Warnings {
				if containsSubstring(w, "namespace") {
					hasNsWarning = true
					break
				}
			}
			if tt.hasWarning && !hasNsWarning {
				t.Errorf("Expected namespace warning for %s, got warnings: %v", tt.cmd, report.Warnings)
			}
		})
	}
}

func TestAnalyzerDeleteAllNamespaces(t *testing.T) {
	analyzer := NewAnalyzer()

	cmd := "kubectl delete pods --all-namespaces"
	report := analyzer.Analyze(cmd)

	if !report.IsDangerous {
		t.Error("Delete with --all-namespaces should be dangerous")
	}

	hasAllNsWarning := false
	for _, w := range report.Warnings {
		if containsSubstring(w, "all namespaces") {
			hasAllNsWarning = true
			break
		}
	}
	if !hasAllNsWarning {
		t.Errorf("Should have all namespaces warning, got: %v", report.Warnings)
	}
}

func TestAnalyzerReportType(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		cmd          string
		expectedType CommandType
	}{
		{"kubectl get pods", TypeReadOnly},
		{"kubectl apply -f deploy.yaml", TypeWrite},
		{"kubectl drain node1", TypeDangerous},
		{"kubectl exec -it pod -- bash", TypeInteractive},
		{"ls -la", TypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			report := analyzer.Analyze(tt.cmd)
			if report.Type != tt.expectedType {
				t.Errorf("Type = %v, want %v", report.Type, tt.expectedType)
			}
		})
	}
}

func TestAnalyzerChainedCommands(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		cmd       string
		isChained bool
	}{
		{"kubectl get pods && kubectl get services", true},
		{"kubectl get pods || echo 'no pods'", true},
		{"kubectl get pods", false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			report := analyzer.Analyze(tt.cmd)
			hasChainedWarning := false
			for _, w := range report.Warnings {
				if containsSubstring(w, "Chained") {
					hasChainedWarning = true
					break
				}
			}
			if tt.isChained && !hasChainedWarning {
				t.Errorf("Expected chained warning for %s", tt.cmd)
			}
		})
	}
}

func TestAnalyzerRmCommand(t *testing.T) {
	analyzer := NewAnalyzer()

	// Note: The Analyzer doesn't detect rm as dangerous because checkDangerousPatterns
	// is only called for kubectl/helm commands. For rm detection, use QuickCheck instead.
	tests := []struct {
		cmd              string
		requiresApproval bool
		typeUnknown      bool
	}{
		{"rm --recursive dir", true, true},
		{"rm file.txt", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			report := analyzer.Analyze(tt.cmd)
			if report.RequiresApproval != tt.requiresApproval {
				t.Errorf("RequiresApproval = %v, want %v for %s", report.RequiresApproval, tt.requiresApproval, tt.cmd)
			}
			if tt.typeUnknown && report.Type != TypeUnknown {
				t.Errorf("Type = %v, want TypeUnknown for %s", report.Type, tt.cmd)
			}
		})
	}
}

func TestQuickCheckRmPatterns(t *testing.T) {
	// QuickCheck does simple string matching, so it catches rm -rf patterns
	tests := []struct {
		cmd           string
		wantDangerous bool
	}{
		{"rm -rf /tmp/test", true},
		{"rm file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			_, isDangerous := QuickCheck(tt.cmd)
			if isDangerous != tt.wantDangerous {
				t.Errorf("QuickCheck isDangerous = %v, want %v", isDangerous, tt.wantDangerous)
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && containsAt(s, substr)))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
