package ui

import (
	"strings"
	"testing"
)

func TestLogo(t *testing.T) {
	// Logo should not be empty
	if Logo == "" {
		t.Error("Logo should not be empty")
	}

	// Logo should contain k13d characters
	if !strings.Contains(Logo, "██") {
		t.Error("Logo should contain block characters")
	}
}

func TestLogoColors(t *testing.T) {
	colored := LogoColors()

	// Should contain color codes
	if !strings.Contains(colored, "[#") {
		t.Error("LogoColors should contain color codes")
	}

	// Should contain closing tags
	if !strings.Contains(colored, "[-]") {
		t.Error("LogoColors should contain closing color tags")
	}
}

func TestHeaderLogo(t *testing.T) {
	header := HeaderLogo()

	// Should contain k13d
	if !strings.Contains(header, "k") || !strings.Contains(header, "1") ||
		!strings.Contains(header, "3") || !strings.Contains(header, "d") {
		t.Error("HeaderLogo should contain k13d characters")
	}

	// Should contain color codes
	if !strings.Contains(header, "[#") {
		t.Error("HeaderLogo should contain color codes")
	}
}

func TestHeaderLogoWithContext(t *testing.T) {
	tests := []struct {
		name      string
		cluster   string
		namespace string
		resource  string
		wantParts []string
	}{
		{
			name:      "all fields",
			cluster:   "my-cluster",
			namespace: "default",
			resource:  "pods",
			wantParts: []string{"k", "my-cluster", "default", "pods"},
		},
		{
			name:      "empty namespace",
			cluster:   "my-cluster",
			namespace: "",
			resource:  "pods",
			wantParts: []string{"k", "my-cluster", "all", "pods"},
		},
		{
			name:      "empty all",
			cluster:   "",
			namespace: "",
			resource:  "",
			wantParts: []string{"k", "all"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HeaderLogoWithContext(tt.cluster, tt.namespace, tt.resource)
			for _, part := range tt.wantParts {
				if !strings.Contains(result, part) {
					t.Errorf("HeaderLogoWithContext() = %v, should contain %v", result, part)
				}
			}
		})
	}
}

func TestTagline(t *testing.T) {
	if Tagline == "" {
		t.Error("Tagline should not be empty")
	}

	if !strings.Contains(Tagline, "Kubernetes") {
		t.Error("Tagline should contain 'Kubernetes'")
	}
}

func TestVersion(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestResourceIcon(t *testing.T) {
	tests := []struct {
		resource string
		wantIcon bool
	}{
		{"pods", true},
		{"po", true},
		{"deployments", true},
		{"deploy", true},
		{"services", true},
		{"svc", true},
		{"nodes", true},
		{"namespaces", true},
		{"configmaps", true},
		{"secrets", true},
		{"unknown", true}, // Should return default icon
	}

	for _, tt := range tests {
		t.Run(tt.resource, func(t *testing.T) {
			icon := ResourceIcon(tt.resource)
			if icon == "" {
				t.Errorf("ResourceIcon(%q) returned empty string", tt.resource)
			}
		})
	}
}

func TestLogoStatusColor(t *testing.T) {
	tests := []struct {
		status    string
		wantColor string
	}{
		{"Running", "[green]"},
		{"running", "[green]"},
		{"Ready", "[green]"},
		{"Active", "[green]"},
		{"Succeeded", "[green]"},
		{"Completed", "[green]"},
		{"Normal", "[green]"},
		{"Bound", "[green]"},
		{"Pending", "[yellow]"},
		{"ContainerCreating", "[yellow]"},
		{"Warning", "[yellow]"},
		{"Updating", "[yellow]"},
		{"Terminating", "[yellow]"},
		{"Unknown", "[yellow]"},
		{"Failed", "[red]"},
		{"Error", "[red]"},
		{"CrashLoopBackOff", "[red]"},
		{"NotReady", "[red]"},
		{"ImagePullBackOff", "[red]"},
		{"Evicted", "[red]"},
		{"OOMKilled", "[red]"},
		{"SomeOtherStatus", "[white]"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			color := StatusColor(tt.status)
			if color != tt.wantColor {
				t.Errorf("StatusColor(%q) = %q, want %q", tt.status, color, tt.wantColor)
			}
		})
	}
}

func TestFormatStatus(t *testing.T) {
	result := FormatStatus("Running")
	if !strings.Contains(result, "Running") {
		t.Error("FormatStatus should contain the status text")
	}
	if !strings.Contains(result, "[green]") {
		t.Error("FormatStatus for Running should contain green color")
	}
}

func TestProgressBar(t *testing.T) {
	tests := []struct {
		name    string
		current int
		total   int
		width   int
	}{
		{"empty", 0, 10, 10},
		{"half", 5, 10, 10},
		{"full", 10, 10, 10},
		{"zero total", 0, 0, 10},
		{"over 100%", 15, 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProgressBar(tt.current, tt.total, tt.width)
			if result == "" {
				t.Error("ProgressBar should not return empty string")
			}
			// Should contain color codes
			if !strings.Contains(result, "[") {
				t.Error("ProgressBar should contain color codes")
			}
		})
	}
}

func TestSpinner(t *testing.T) {
	// Test all frames
	for i := 0; i < len(SpinnerFrames)*2; i++ {
		frame := Spinner(i)
		if frame == "" {
			t.Errorf("Spinner(%d) returned empty string", i)
		}
	}
}

func TestColoredSpinner(t *testing.T) {
	result := ColoredSpinner(0, "cyan")
	if !strings.Contains(result, "[cyan]") {
		t.Error("ColoredSpinner should contain color code")
	}
	if !strings.Contains(result, "[-]") {
		t.Error("ColoredSpinner should contain closing tag")
	}
}

func TestNewSplashScreen(t *testing.T) {
	splash := NewSplashScreen()
	if splash == nil {
		t.Error("NewSplashScreen should not return nil")
	}
	if splash.logo == nil {
		t.Error("SplashScreen.logo should not be nil")
	}
	if splash.info == nil {
		t.Error("SplashScreen.info should not be nil")
	}
	if splash.progress == nil {
		t.Error("SplashScreen.progress should not be nil")
	}
}

func TestSplashScreenSetProgress(t *testing.T) {
	splash := NewSplashScreen()
	splash.SetProgress("Loading...")
	// Should not panic
}

func TestSplashScreenSetError(t *testing.T) {
	splash := NewSplashScreen()
	splash.SetError("Something went wrong")
	// Should not panic
}

func TestSplashScreenSetReady(t *testing.T) {
	splash := NewSplashScreen()
	splash.SetReady()
	// Should not panic
}

func TestAboutModal(t *testing.T) {
	modal := AboutModal()
	if modal == nil {
		t.Error("AboutModal should not return nil")
	}
}
