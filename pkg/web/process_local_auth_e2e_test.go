package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

func TestProcessE2E_LocalAuthLoginFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping process E2E test in short mode")
	}
	if os.Getenv("K13D_RUN_PROCESS_E2E") != "1" {
		t.Skip("set K13D_RUN_PROCESS_E2E=1 to run the process-based local auth E2E test")
	}
	requireProcessE2EClusterAccess(t)

	server := startProcessLocalAuthE2EServer(t)
	defer server.Close(t)

	assertLocalAuthLoginFlowChecks(t, server, "en")
	assertLocalAuthLogout(t, server)
}

func TestProcessE2E_LocalAuthLoginFlow_NoConfigFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping process E2E test in short mode")
	}
	if os.Getenv("K13D_RUN_PROCESS_E2E") != "1" {
		t.Skip("set K13D_RUN_PROCESS_E2E=1 to run the process-based local auth E2E test")
	}

	requireProcessE2EClusterAccess(t)

	server := startProcessLocalAuthE2EServerWithoutConfig(t)
	defer server.Close(t)

	if _, err := os.Stat(server.configPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected config file to be absent before mutation, got err=%v", err)
	}

	csrfToken := assertLocalAuthLoginFlowChecks(t, server, "ko")

	if _, err := os.Stat(server.configPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read-only flow should not create config.yaml, got err=%v", err)
	}

	switchResp, switchBody := doProcessE2ERequest(t, server.client, http.MethodPut, server.baseURL+"/api/models/active", map[string]string{
		"name": "gpt-oss-local",
	}, map[string]string{
		"Content-Type": "application/json",
		"X-CSRF-Token": csrfToken,
	})
	switchResp.Body.Close()
	if switchResp.StatusCode != http.StatusOK {
		t.Fatalf("switch active model failed with status %d:\n%s", switchResp.StatusCode, switchBody)
	}

	var switchResult struct {
		Status      string `json:"status"`
		ActiveModel string `json:"active_model"`
	}
	if err := json.Unmarshal(switchBody, &switchResult); err != nil {
		t.Fatalf("failed to decode active model switch response: %v\n%s", err, switchBody)
	}
	if switchResult.Status != "switched" || switchResult.ActiveModel != "gpt-oss-local" {
		t.Fatalf("unexpected active model switch response: %+v", switchResult)
	}

	if info, err := os.Stat(server.configPath); err != nil {
		t.Fatalf("expected config file to be created after saving settings: %v", err)
	} else if info.Mode().Perm() != 0600 {
		t.Fatalf("config file mode = %o, want 600", info.Mode().Perm())
	}

	t.Setenv("K13D_CONFIG", server.configPath)
	loadedCfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() after no-config E2E save error = %v", err)
	}
	if loadedCfg.ActiveModel != "gpt-oss-local" {
		t.Fatalf("ActiveModel = %s, want gpt-oss-local", loadedCfg.ActiveModel)
	}
	if loadedCfg.LLM.Provider != "ollama" {
		t.Fatalf("LLM.Provider = %s, want ollama", loadedCfg.LLM.Provider)
	}
	if loadedCfg.LLM.Model != config.DefaultOllamaModel {
		t.Fatalf("LLM.Model = %s, want %s", loadedCfg.LLM.Model, config.DefaultOllamaModel)
	}

	assertLocalAuthLogout(t, server)
}

func assertLocalAuthLoginFlowChecks(t *testing.T, server *processLocalAuthE2EServer, wantLanguage string) string {
	t.Helper()

	rootResp, rootBody := doProcessE2ERequest(t, server.client, http.MethodGet, server.baseURL+"/", nil, nil)
	rootResp.Body.Close()
	rootHTML := string(rootBody)
	if !strings.Contains(rootHTML, `window.__AUTH_MODE__="local"`) {
		t.Fatalf("expected local auth mode injection in login page, got:\n%s", rootHTML)
	}
	if !strings.Contains(rootHTML, `id="password-login-form" class="login-form" style="display:block"`) {
		t.Fatalf("expected password login form to be visible for local auth, got:\n%s", rootHTML)
	}
	if strings.Contains(rootHTML, `id="token-login-form" class="login-form" style="display:block"`) {
		t.Fatalf("token form should not be force-shown for local auth, got:\n%s", rootHTML)
	}

	statusResp, statusBody := doProcessE2ERequest(t, server.client, http.MethodGet, server.baseURL+"/api/auth/status", nil, nil)
	statusResp.Body.Close()
	var authStatus struct {
		AuthEnabled bool   `json:"auth_enabled"`
		AuthMode    string `json:"auth_mode"`
	}
	if err := json.Unmarshal(statusBody, &authStatus); err != nil {
		t.Fatalf("failed to decode auth status: %v\n%s", err, statusBody)
	}
	if !authStatus.AuthEnabled || authStatus.AuthMode != "local" {
		t.Fatalf("unexpected auth status: %+v", authStatus)
	}

	server.Login(t)

	baseURLParsed, err := http.NewRequest(http.MethodGet, server.baseURL, nil)
	if err != nil {
		t.Fatalf("failed to parse base URL: %v", err)
	}
	if server.client.Jar == nil || len(server.client.Jar.Cookies(baseURLParsed.URL)) == 0 {
		t.Fatal("expected browser-style session cookie after login")
	}

	meResp, meBody := doProcessE2ERequest(t, server.client, http.MethodGet, server.baseURL+"/api/auth/me", nil, nil)
	meResp.Body.Close()
	if meResp.StatusCode != http.StatusOK {
		t.Fatalf("current-user endpoint failed with status %d:\n%s", meResp.StatusCode, meBody)
	}

	var me struct {
		Username string `json:"username"`
		Role     string `json:"role"`
		AuthMode string `json:"auth_mode"`
	}
	if err := json.Unmarshal(meBody, &me); err != nil {
		t.Fatalf("failed to decode current-user response: %v\n%s", err, meBody)
	}
	if me.Username != server.adminUser || me.Role != "admin" || me.AuthMode != "local" {
		t.Fatalf("unexpected current-user response: %+v", me)
	}

	permsResp, permsBody := doProcessE2ERequest(t, server.client, http.MethodGet, server.baseURL+"/api/auth/permissions", nil, nil)
	permsResp.Body.Close()
	if permsResp.StatusCode != http.StatusOK {
		t.Fatalf("permissions endpoint failed with status %d:\n%s", permsResp.StatusCode, permsBody)
	}

	var perms struct {
		Role     string          `json:"role"`
		Features map[string]bool `json:"features"`
	}
	if err := json.Unmarshal(permsBody, &perms); err != nil {
		t.Fatalf("failed to decode permissions response: %v\n%s", err, permsBody)
	}
	if perms.Role != "admin" || !perms.Features[string(FeatureAIAssistant)] || !perms.Features[string(FeatureSettingsAdmin)] {
		t.Fatalf("unexpected permissions payload: %+v", perms)
	}

	adminResp, adminBody := doProcessE2ERequest(t, server.client, http.MethodGet, server.baseURL+"/api/admin/users", nil, nil)
	adminResp.Body.Close()
	if adminResp.StatusCode != http.StatusOK {
		t.Fatalf("admin users endpoint failed with status %d:\n%s", adminResp.StatusCode, adminBody)
	}
	if !strings.Contains(string(adminBody), server.adminUser) {
		t.Fatalf("expected admin users response to include %q, got:\n%s", server.adminUser, adminBody)
	}

	settingsResp, settingsBody := doProcessE2ERequest(t, server.client, http.MethodGet, server.baseURL+"/api/settings", nil, nil)
	settingsResp.Body.Close()
	if settingsResp.StatusCode != http.StatusOK {
		t.Fatalf("settings endpoint failed with status %d:\n%s", settingsResp.StatusCode, settingsBody)
	}

	var settings struct {
		Language string `json:"language"`
		LLM      struct {
			Provider string `json:"provider"`
			Model    string `json:"model"`
		} `json:"llm"`
	}
	if err := json.Unmarshal(settingsBody, &settings); err != nil {
		t.Fatalf("failed to decode settings response: %v\n%s", err, settingsBody)
	}
	if settings.Language != wantLanguage {
		t.Fatalf("settings.language = %s, want %s", settings.Language, wantLanguage)
	}
	if settings.LLM.Provider != server.expectLLMProvider {
		t.Fatalf("settings.llm.provider = %s, want %s", settings.LLM.Provider, server.expectLLMProvider)
	}
	if settings.LLM.Model != server.expectLLMModel {
		t.Fatalf("settings.llm.model = %s, want %s", settings.LLM.Model, server.expectLLMModel)
	}

	llmResp, llmBody := doProcessE2ERequest(t, server.client, http.MethodGet, server.baseURL+"/api/llm/status", nil, nil)
	llmResp.Body.Close()
	if llmResp.StatusCode != http.StatusOK {
		t.Fatalf("LLM status endpoint failed with status %d:\n%s", llmResp.StatusCode, llmBody)
	}

	var llmStatus struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
	}
	if err := json.Unmarshal(llmBody, &llmStatus); err != nil {
		t.Fatalf("failed to decode LLM status response: %v\n%s", err, llmBody)
	}
	if llmStatus.Provider != server.expectLLMProvider {
		t.Fatalf("LLM status provider = %s, want %s", llmStatus.Provider, server.expectLLMProvider)
	}
	if llmStatus.Model != server.expectLLMModel {
		t.Fatalf("LLM status model = %s, want %s", llmStatus.Model, server.expectLLMModel)
	}

	csrfToken := server.CSRFToken(t)

	chatResp, chatBody := doProcessE2ERequest(t, server.client, http.MethodPost, server.baseURL+"/api/chat/agentic", map[string]string{
		"message": "hello from process e2e",
	}, map[string]string{
		"Content-Type": "application/json",
		"X-CSRF-Token": csrfToken,
	})
	chatResp.Body.Close()
	if chatResp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected agentic chat to be reachable but unconfigured (503), got %d:\n%s", chatResp.StatusCode, chatBody)
	}

	var apiErr APIError
	if err := json.Unmarshal(chatBody, &apiErr); err != nil {
		t.Fatalf("failed to decode agentic chat error: %v\n%s", err, chatBody)
	}
	if apiErr.Code != ErrCodeLLMNotConfigured {
		t.Fatalf("expected LLM_NOT_CONFIGURED error, got %+v", apiErr)
	}

	return csrfToken
}

func assertLocalAuthLogout(t *testing.T, server *processLocalAuthE2EServer) {
	t.Helper()

	logoutResp, logoutBody := doProcessE2ERequest(t, server.client, http.MethodPost, server.baseURL+"/api/auth/logout", nil, nil)
	logoutResp.Body.Close()
	if logoutResp.StatusCode != http.StatusOK {
		t.Fatalf("logout failed with status %d:\n%s", logoutResp.StatusCode, logoutBody)
	}

	meAfterLogoutResp, meAfterLogoutBody := doProcessE2ERequest(t, server.client, http.MethodGet, server.baseURL+"/api/auth/me", nil, nil)
	meAfterLogoutResp.Body.Close()
	if meAfterLogoutResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected auth/me to reject the invalidated session, got %d:\n%s", meAfterLogoutResp.StatusCode, meAfterLogoutBody)
	}
}

const processE2EConfigYAML = `llm:
  provider: openai
  model: test-model
  endpoint: ""
language: en
beginner_mode: false
notifications:
  enabled: false
  events: []
  poll_interval: 30
mcp:
  servers: []
prometheus:
  expose_metrics: false
  collect_k8s_metrics: false
  collection_interval: 60
`

func processE2ERepoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve process E2E test path")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func buildProcessE2EBinary(t *testing.T, repoRoot string) string {
	t.Helper()

	binaryPath := filepath.Join(t.TempDir(), "k13d-process-e2e")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/kube-ai-dashboard-cli")
	buildCmd.Dir = repoRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build k13d binary for process E2E: %v\n%s", err, output)
	}

	return binaryPath
}

func reserveProcessE2EPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve TCP port: %v", err)
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port
}

func processE2EEnv(configPath string) []string {
	env := make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "K13D_") {
			continue
		}
		env = append(env, entry)
	}
	env = append(env, "K13D_CONFIG="+configPath)
	return env
}

func waitForProcessE2EReady(t *testing.T, client *http.Client, done <-chan error, url string, stdout, stderr *bytes.Buffer) {
	t.Helper()

	deadline := time.Now().Add(25 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}

		select {
		case err := <-done:
			t.Fatalf("web server exited before becoming ready: %v\n%s", err, processE2ELogs(stdout, stderr))
		default:
		}

		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for web server readiness\n%s", processE2ELogs(stdout, stderr))
}

func stopProcessE2EServer(t *testing.T, cmd *exec.Cmd, done <-chan error, stdout, stderr *bytes.Buffer) {
	t.Helper()

	if cmd.Process == nil {
		return
	}

	if err := cmd.Process.Signal(os.Interrupt); err != nil && !errors.Is(err, os.ErrProcessDone) {
		t.Logf("failed to interrupt process cleanly: %v", err)
	}

	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Logf("process exited with error during cleanup: %v\n%s", err, processE2ELogs(stdout, stderr))
		}
	case <-time.After(5 * time.Second):
		if killErr := cmd.Process.Kill(); killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
			t.Logf("failed to kill process after interrupt timeout: %v", killErr)
		}
		select {
		case err := <-done:
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Logf("process exited with error after forced kill: %v\n%s", err, processE2ELogs(stdout, stderr))
			}
		case <-time.After(2 * time.Second):
			t.Logf("process did not exit after forced kill\n%s", processE2ELogs(stdout, stderr))
		}
	}
}

func doProcessE2ERequest(t *testing.T, client *http.Client, method, url string, payload interface{}, headers map[string]string) (*http.Response, []byte) {
	t.Helper()

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("failed to marshal request payload for %s %s: %v", method, url, err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("failed to build request %s %s: %v", method, url, err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request %s %s failed: %v", method, url, err)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		t.Fatalf("failed to read response body for %s %s: %v", method, url, err)
	}

	return resp, respBody
}

func processE2ELogs(stdout, stderr *bytes.Buffer) string {
	return fmt.Sprintf("stdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
}
