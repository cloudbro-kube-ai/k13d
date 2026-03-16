package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/k8s"
)

type processLocalAuthE2EServer struct {
	repoRoot          string
	baseURL           string
	adminUser         string
	adminPass         string
	expectLLMProvider string
	expectLLMModel    string
	client            *http.Client
	cmd               *exec.Cmd
	done              chan error
	stdout            bytes.Buffer
	stderr            bytes.Buffer
	configPath        string
}

type processLocalAuthE2EOptions struct {
	writeConfig       bool
	configYAML        string
	expectLLMProvider string
	expectLLMModel    string
}

func requireProcessE2EClusterAccess(t *testing.T) {
	t.Helper()

	if _, err := k8s.NewClient(); err != nil {
		t.Skipf("process E2E requires kubeconfig or in-cluster credentials: %v", err)
	}
}

func startProcessLocalAuthE2EServer(t *testing.T) *processLocalAuthE2EServer {
	t.Helper()

	return startProcessLocalAuthE2EServerWithOptions(t, processLocalAuthE2EOptions{
		writeConfig:       true,
		configYAML:        processE2EConfigYAML,
		expectLLMProvider: "openai",
		expectLLMModel:    "test-model",
	})
}

func startProcessLocalAuthE2EServerWithoutConfig(t *testing.T) *processLocalAuthE2EServer {
	t.Helper()

	return startProcessLocalAuthE2EServerWithOptions(t, processLocalAuthE2EOptions{
		writeConfig:       false,
		expectLLMProvider: "upstage",
		expectLLMModel:    "solar-pro2",
	})
}

func startProcessLocalAuthE2EServerWithOptions(t *testing.T, opts processLocalAuthE2EOptions) *processLocalAuthE2EServer {
	t.Helper()

	repoRoot := processE2ERepoRoot(t)
	binaryPath := buildProcessE2EBinary(t, repoRoot)
	port := reserveProcessE2EPort(t)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	dbPath := filepath.Join(tmpDir, "audit.db")
	if opts.writeConfig {
		configYAML := opts.configYAML
		if configYAML == "" {
			configYAML = processE2EConfigYAML
		}
		if err := os.WriteFile(configPath, []byte(configYAML), 0600); err != nil {
			t.Fatalf("failed to write temporary config: %v", err)
		}
	}

	baseURL := "http://127.0.0.1:" + strconv.Itoa(port)
	adminUser := "e2e-admin"
	adminPass := "e2e-password-123"

	ctx, cancel := processE2ECommandContext(t)
	t.Cleanup(cancel)

	cmd := exec.CommandContext(ctx, binaryPath,
		"--web",
		"--auth-mode", "local",
		"--admin-user", adminUser,
		"--admin-password", adminPass,
		"--port", strconv.Itoa(port),
		"--db-path", dbPath,
		"--config", configPath,
	)
	cmd.Dir = repoRoot
	cmd.Env = processE2EEnv(configPath)

	server := &processLocalAuthE2EServer{
		repoRoot:          repoRoot,
		baseURL:           baseURL,
		adminUser:         adminUser,
		adminPass:         adminPass,
		expectLLMProvider: opts.expectLLMProvider,
		expectLLMModel:    opts.expectLLMModel,
		cmd:               cmd,
		done:              make(chan error, 1),
		configPath:        configPath,
	}
	cmd.Stdout = &server.stdout
	cmd.Stderr = &server.stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start local-auth web server: %v", err)
	}

	go func() {
		server.done <- cmd.Wait()
	}()

	readinessClient := &http.Client{Timeout: 10 * time.Second}
	waitForProcessE2EReady(t, readinessClient, server.done, baseURL+"/api/health", &server.stdout, &server.stderr)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}
	server.client = &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
	}

	return server
}

func processE2ECommandContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()

	if deadline, ok := t.Deadline(); ok {
		// Leave a small buffer so cleanup can still stop the child process cleanly
		// before the testing framework tears the test down.
		buffer := 15 * time.Second
		if boundedDeadline := deadline.Add(-buffer); time.Now().Before(boundedDeadline) {
			return context.WithDeadline(context.Background(), boundedDeadline)
		}
	}

	return context.WithCancel(context.Background())
}

func (s *processLocalAuthE2EServer) Close(t *testing.T) {
	t.Helper()
	stopProcessE2EServer(t, s.cmd, s.done, &s.stdout, &s.stderr)
}

func (s *processLocalAuthE2EServer) Login(t *testing.T) LoginResponse {
	t.Helper()

	loginResp, loginBody := doProcessE2ERequest(t, s.client, http.MethodPost, s.baseURL+"/api/auth/login", map[string]string{
		"username": s.adminUser,
		"password": s.adminPass,
	}, map[string]string{
		"Content-Type": "application/json",
	})
	loginResp.Body.Close()
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login failed with status %d:\n%s", loginResp.StatusCode, loginBody)
	}

	var loginResult LoginResponse
	if err := json.Unmarshal(loginBody, &loginResult); err != nil {
		t.Fatalf("failed to decode login response: %v\n%s", err, loginBody)
	}
	if loginResult.Username != s.adminUser || loginResult.Role != "admin" || loginResult.Token == "" || loginResult.AuthMode != "local" {
		t.Fatalf("unexpected login response: %+v", loginResult)
	}

	return loginResult
}

func (s *processLocalAuthE2EServer) CSRFToken(t *testing.T) string {
	return s.CSRFTokenForClient(t, s.client)
}

func (s *processLocalAuthE2EServer) NewAuthenticatedClient(t *testing.T, username, password string) (*http.Client, LoginResponse) {
	t.Helper()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
	}

	loginResp, loginBody := doProcessE2ERequest(t, client, http.MethodPost, s.baseURL+"/api/auth/login", map[string]string{
		"username": username,
		"password": password,
	}, map[string]string{
		"Content-Type": "application/json",
	})
	loginResp.Body.Close()
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login failed for %q with status %d:\n%s", username, loginResp.StatusCode, loginBody)
	}

	var loginResult LoginResponse
	if err := json.Unmarshal(loginBody, &loginResult); err != nil {
		t.Fatalf("failed to decode login response for %q: %v\n%s", username, err, loginBody)
	}
	if loginResult.Username != username || loginResult.Token == "" || loginResult.AuthMode != "local" {
		t.Fatalf("unexpected login response for %q: %+v", username, loginResult)
	}

	return client, loginResult
}

func (s *processLocalAuthE2EServer) CSRFTokenForClient(t *testing.T, client *http.Client) string {
	t.Helper()

	resp, body := doProcessE2ERequest(t, client, http.MethodGet, s.baseURL+"/api/auth/csrf-token", nil, nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("csrf endpoint failed with status %d:\n%s", resp.StatusCode, body)
	}

	var csrf struct {
		Token string `json:"csrf_token"`
	}
	if err := json.Unmarshal(body, &csrf); err != nil {
		t.Fatalf("failed to decode CSRF token: %v\n%s", err, body)
	}
	if csrf.Token == "" {
		t.Fatal("expected non-empty CSRF token")
	}
	return csrf.Token
}
