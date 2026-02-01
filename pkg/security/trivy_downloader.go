package security

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// TrivyRelease represents GitHub release info
type TrivyRelease struct {
	TagName string       `json:"tag_name"`
	Assets  []TrivyAsset `json:"assets"`
}

// TrivyAsset represents a release asset
type TrivyAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// TrivyDownloader handles Trivy binary download and management
type TrivyDownloader struct {
	installDir string
	httpClient *http.Client
}

// TrivyStatus represents the current Trivy installation status
type TrivyStatus struct {
	Installed       bool   `json:"installed"`
	Version         string `json:"version,omitempty"`
	Path            string `json:"path,omitempty"`
	LatestVersion   string `json:"latest_version,omitempty"`
	UpdateAvailable bool   `json:"update_available,omitempty"`
	Error           string `json:"error,omitempty"`
}

// NewTrivyDownloader creates a new Trivy downloader
func NewTrivyDownloader() *TrivyDownloader {
	// Use k13d config directory for Trivy binary
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.TempDir()
	}
	installDir := filepath.Join(configDir, "k13d", "bin")

	return &TrivyDownloader{
		installDir: installDir,
		httpClient: &http.Client{Timeout: 5 * time.Minute},
	}
}

// GetStatus returns the current Trivy installation status
func (td *TrivyDownloader) GetStatus(ctx context.Context) *TrivyStatus {
	status := &TrivyStatus{}

	// Check system trivy first
	if path, err := exec.LookPath("trivy"); err == nil {
		status.Installed = true
		status.Path = path
		if version, err := td.getVersion(ctx, path); err == nil {
			status.Version = version
		}
	}

	// Check local installation
	localPath := td.getLocalTrivyPath()
	if !status.Installed {
		if _, err := os.Stat(localPath); err == nil {
			status.Installed = true
			status.Path = localPath
			if version, err := td.getVersion(ctx, localPath); err == nil {
				status.Version = version
			}
		}
	}

	// Try to get latest version (non-blocking for air-gapped)
	if latest, err := td.getLatestVersion(ctx); err == nil {
		status.LatestVersion = latest
		if status.Version != "" && status.Version != latest {
			status.UpdateAvailable = true
		}
	}

	return status
}

// getVersion gets the installed Trivy version
func (td *TrivyDownloader) getVersion(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, path, "version", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		// Try without --format json
		cmd = exec.CommandContext(ctx, path, "version")
		output, err = cmd.Output()
		if err != nil {
			return "", err
		}
		// Parse simple version output
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Version:") {
				return strings.TrimSpace(strings.TrimPrefix(line, "Version:")), nil
			}
		}
		return "", fmt.Errorf("could not parse version")
	}

	var versionInfo struct {
		Version string `json:"Version"`
	}
	if err := json.Unmarshal(output, &versionInfo); err != nil {
		return "", err
	}
	return versionInfo.Version, nil
}

// getLatestVersion fetches the latest Trivy version from GitHub
func (td *TrivyDownloader) getLatestVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/repos/aquasecurity/trivy/releases/latest", nil)
	if err != nil {
		return "", err
	}

	resp, err := td.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	var release TrivyRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return strings.TrimPrefix(release.TagName, "v"), nil
}

// Download downloads and installs Trivy
func (td *TrivyDownloader) Download(ctx context.Context, progressCallback func(progress int, status string)) error {
	if progressCallback != nil {
		progressCallback(0, "Fetching latest release info...")
	}

	// Get latest release info
	release, err := td.getLatestRelease(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest release: %w", err)
	}

	// Find appropriate asset
	asset := td.findAsset(release)
	if asset == nil {
		return fmt.Errorf("no suitable binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	if progressCallback != nil {
		progressCallback(10, fmt.Sprintf("Downloading %s...", asset.Name))
	}

	// Create install directory
	if err := os.MkdirAll(td.installDir, 0755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	// Download the asset
	req, err := http.NewRequestWithContext(ctx, "GET", asset.BrowserDownloadURL, nil)
	if err != nil {
		return err
	}

	resp, err := td.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "trivy-*.tar.gz")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Download with progress
	downloaded := int64(0)
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := tmpFile.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)
			if progressCallback != nil && asset.Size > 0 {
				progress := int(10 + (float64(downloaded)/float64(asset.Size))*70)
				progressCallback(progress, fmt.Sprintf("Downloaded %.1f MB / %.1f MB",
					float64(downloaded)/(1024*1024),
					float64(asset.Size)/(1024*1024)))
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	if progressCallback != nil {
		progressCallback(80, "Extracting...")
	}

	// Extract
	if err := td.extractTarGz(tmpFile.Name()); err != nil {
		return fmt.Errorf("failed to extract: %w", err)
	}

	if progressCallback != nil {
		progressCallback(100, "Installation complete!")
	}

	return nil
}

// getLatestRelease fetches the latest release from GitHub
func (td *TrivyDownloader) getLatestRelease(ctx context.Context) (*TrivyRelease, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/repos/aquasecurity/trivy/releases/latest", nil)
	if err != nil {
		return nil, err
	}

	resp, err := td.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var release TrivyRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// findAsset finds the appropriate asset for current OS/arch
func (td *TrivyDownloader) findAsset(release *TrivyRelease) *TrivyAsset {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	// Map arch names
	archMap := map[string]string{
		"amd64": "64bit",
		"arm64": "ARM64",
		"386":   "32bit",
	}
	if mapped, ok := archMap[arch]; ok {
		arch = mapped
	}

	// Map OS names
	osMap := map[string]string{
		"darwin":  "macOS",
		"linux":   "Linux",
		"windows": "Windows",
	}
	if mapped, ok := osMap[osName]; ok {
		osName = mapped
	}

	// Find matching asset
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, strings.ToLower(osName)) &&
			strings.Contains(name, strings.ToLower(arch)) &&
			strings.HasSuffix(name, ".tar.gz") {
			return &asset
		}
	}

	return nil
}

// extractTarGz extracts the trivy binary from tar.gz
func (td *TrivyDownloader) extractTarGz(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Only extract the trivy binary
		if header.Name == "trivy" || header.Name == "trivy.exe" {
			targetPath := filepath.Join(td.installDir, header.Name)
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
			return nil
		}
	}

	return fmt.Errorf("trivy binary not found in archive")
}

// getLocalTrivyPath returns the path to locally installed Trivy
func (td *TrivyDownloader) getLocalTrivyPath() string {
	name := "trivy"
	if runtime.GOOS == "windows" {
		name = "trivy.exe"
	}
	return filepath.Join(td.installDir, name)
}

// GetTrivyPath returns the path to Trivy binary (system or local)
func (td *TrivyDownloader) GetTrivyPath() string {
	// System trivy first
	if path, err := exec.LookPath("trivy"); err == nil {
		return path
	}

	// Local installation
	localPath := td.getLocalTrivyPath()
	if _, err := os.Stat(localPath); err == nil {
		return localPath
	}

	return ""
}

// GetInstallInstructions returns instructions for manual installation
func GetInstallInstructions() string {
	switch runtime.GOOS {
	case "darwin":
		return `# macOS - Install via Homebrew:
brew install trivy

# Or download manually:
curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin`
	case "linux":
		return `# Linux - Install via package manager:
# Debian/Ubuntu:
sudo apt-get install wget apt-transport-https gnupg lsb-release
wget -qO - https://aquasecurity.github.io/trivy-repo/deb/public.key | sudo apt-key add -
echo deb https://aquasecurity.github.io/trivy-repo/deb $(lsb_release -sc) main | sudo tee -a /etc/apt/sources.list.d/trivy.list
sudo apt-get update && sudo apt-get install trivy

# Or install script:
curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin`
	case "windows":
		return `# Windows - Install via Chocolatey:
choco install trivy

# Or download from:
https://github.com/aquasecurity/trivy/releases/latest`
	default:
		return "Visit https://github.com/aquasecurity/trivy/releases to download Trivy for your platform."
	}
}
