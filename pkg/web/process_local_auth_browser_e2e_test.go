package web

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestProcessE2E_LocalAuthBrowserFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping browser E2E test in short mode")
	}
	if os.Getenv("K13D_RUN_BROWSER_E2E") != "1" {
		t.Skip("set K13D_RUN_BROWSER_E2E=1 to run the browser-based local auth E2E test")
	}

	requireProcessE2EClusterAccess(t)

	server := startProcessLocalAuthE2EServer(t)
	defer server.Close(t)

	runLocalAuthBrowserFlow(t, server)
}

func TestProcessE2E_LocalAuthBrowserFlow_NoConfigFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping browser E2E test in short mode")
	}
	if os.Getenv("K13D_RUN_BROWSER_E2E") != "1" {
		t.Skip("set K13D_RUN_BROWSER_E2E=1 to run the browser-based local auth E2E test")
	}

	requireProcessE2EClusterAccess(t)

	server := startProcessLocalAuthE2EServerWithoutConfig(t)
	defer server.Close(t)

	runLocalAuthBrowserFlow(t, server)
}

func runLocalAuthBrowserFlow(t *testing.T, server *processLocalAuthE2EServer) {
	t.Helper()

	playwrightDir := setupPlaywrightWorkspace(t, server.repoRoot)
	ensurePlaywrightChromium(t, playwrightDir)

	cmd := exec.Command("npx", "playwright", "test", "web-local-auth.spec.js", "--config=playwright.config.cjs", "--workers=1", "--reporter=line")
	cmd.Dir = playwrightDir
	cmd.Env = append(os.Environ(),
		"K13D_E2E_BASE_URL="+server.baseURL,
		"K13D_E2E_USERNAME="+server.adminUser,
		"K13D_E2E_PASSWORD="+server.adminPass,
		"K13D_E2E_EXPECT_PROVIDER="+server.expectLLMProvider,
		"K13D_E2E_EXPECT_MODEL="+server.expectLLMModel,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("browser E2E failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
}

func ensurePlaywrightChromium(t *testing.T, workdir string) {
	t.Helper()

	args := []string{"playwright", "install", "chromium"}
	if runtime.GOOS == "linux" && os.Getenv("CI") == "true" {
		args = []string{"playwright", "install", "--with-deps", "chromium"}
	}

	cmd := exec.Command("npx", args...)
	cmd.Dir = workdir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to install Playwright Chromium: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
}

func setupPlaywrightWorkspace(t *testing.T, repoRoot string) string {
	t.Helper()

	workdir := t.TempDir()
	packageJSON := `{
  "name": "k13d-browser-e2e",
  "private": true,
  "devDependencies": {
    "@playwright/test": "1.58.2"
  }
}
`

	if err := os.WriteFile(filepath.Join(workdir, "package.json"), []byte(packageJSON), 0600); err != nil {
		t.Fatalf("failed to write Playwright package.json: %v", err)
	}

	for _, name := range []string{"playwright.config.cjs", "web-local-auth.spec.js"} {
		sourcePath := filepath.Join(repoRoot, "tests", "e2e", name)
		content, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatalf("failed to read %s: %v", sourcePath, err)
		}
		if err := os.WriteFile(filepath.Join(workdir, name), content, 0600); err != nil {
			t.Fatalf("failed to write Playwright file %s: %v", name, err)
		}
	}

	cmd := exec.Command("npm", "install", "--silent", "--no-fund", "--no-audit")
	cmd.Dir = workdir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to install Playwright test runner: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}

	return workdir
}
