package security

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestNewTrivyDownloader(t *testing.T) {
	td := NewTrivyDownloader()
	if td == nil {
		t.Fatal("NewTrivyDownloader returned nil")
	}
	if td.installDir == "" {
		t.Error("installDir should not be empty")
	}
	if td.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
}

func TestTrivyDownloader_GetStatus(t *testing.T) {
	td := NewTrivyDownloader()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := td.GetStatus(ctx)
	if status == nil {
		t.Fatal("GetStatus returned nil")
	}

	// Status should have some value
	t.Logf("Trivy status: installed=%v, version=%s, path=%s",
		status.Installed, status.Version, status.Path)
}

func TestTrivyDownloader_GetLocalTrivyPath(t *testing.T) {
	td := NewTrivyDownloader()
	path := td.getLocalTrivyPath()

	if path == "" {
		t.Error("getLocalTrivyPath returned empty string")
	}

	// Check for correct extension on Windows
	if runtime.GOOS == "windows" {
		if len(path) < 4 || path[len(path)-4:] != ".exe" {
			t.Error("Windows path should end with .exe")
		}
	}
}

func TestTrivyDownloader_findAsset(t *testing.T) {
	td := NewTrivyDownloader()

	// Mock release with various assets
	release := &TrivyRelease{
		TagName: "v0.50.0",
		Assets: []TrivyAsset{
			{Name: "trivy_0.50.0_Linux-64bit.tar.gz", BrowserDownloadURL: "https://example.com/linux64.tar.gz", Size: 50000000},
			{Name: "trivy_0.50.0_Linux-ARM64.tar.gz", BrowserDownloadURL: "https://example.com/linuxarm64.tar.gz", Size: 48000000},
			{Name: "trivy_0.50.0_macOS-64bit.tar.gz", BrowserDownloadURL: "https://example.com/macos64.tar.gz", Size: 52000000},
			{Name: "trivy_0.50.0_macOS-ARM64.tar.gz", BrowserDownloadURL: "https://example.com/macosarm64.tar.gz", Size: 51000000},
			{Name: "trivy_0.50.0_Windows-64bit.zip", BrowserDownloadURL: "https://example.com/win64.zip", Size: 53000000},
			{Name: "trivy_0.50.0_checksums.txt", BrowserDownloadURL: "https://example.com/checksums.txt", Size: 1000},
		},
	}

	asset := td.findAsset(release)

	// Should find an asset for current platform (unless on an unusual platform)
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		if asset == nil {
			t.Errorf("findAsset returned nil for %s/%s", runtime.GOOS, runtime.GOARCH)
		} else {
			t.Logf("Found asset: %s", asset.Name)
		}
	}
}

func TestGetInstallInstructions(t *testing.T) {
	instructions := GetInstallInstructions()
	if instructions == "" {
		t.Error("GetInstallInstructions returned empty string")
	}

	// Should contain some useful text
	if len(instructions) < 50 {
		t.Error("Instructions seem too short")
	}

	t.Logf("Instructions for %s:\n%s", runtime.GOOS, instructions)
}
