const { test, expect } = require('@playwright/test');

const username = process.env.K13D_E2E_USERNAME;
const password = process.env.K13D_E2E_PASSWORD;

test.setTimeout(240000);

async function login(page) {
  await page.goto('/');
  await expect(page.locator('#login-page')).toBeVisible();
  await expect.poll(async () => page.evaluate(() => window.__AUTH_MODE__)).toBe('local');

  const loginResp = await page.request.post('/api/auth/login', {
    data: { username, password }
  });
  expect(loginResp.ok()).toBeTruthy();
  const loginData = await loginResp.json();
  await page.evaluate((token) => {
    localStorage.setItem('k13d_token', token);
  }, loginData.token);
  await page.reload();

  await expect(page.locator('.top-bar')).toBeVisible();
  await expect(page.locator('#user-badge')).toHaveText(/admin/i);
}

async function openSettings(page) {
  await page.getByRole('button', { name: 'Settings' }).click();
  await expect(page.locator('#settings-modal')).toBeVisible();
}

async function switchTab(page, name, selector) {
  await page.getByRole('tab', { name }).click();
  await expect(page.locator(selector)).toBeVisible();
}

async function fetchJSON(page, url, options = {}) {
  return await page.evaluate(async ({ url, options }) => {
    const response = await fetchWithAuth(url, options);
    const text = await response.text();
    let data = null;
    try {
      data = text ? JSON.parse(text) : null;
    } catch {
      data = text;
    }
    return { ok: response.ok, status: response.status, data, text };
  }, { url, options });
}

async function putJSON(page, url, payload) {
  return fetchJSON(page, url, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
}

async function postJSON(page, url, payload) {
  return fetchJSON(page, url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
}

async function setClassToggle(page, selector, enabled) {
  const locator = page.locator(selector);
  const active = await locator.evaluate(el => el.classList.contains('active'));
  if (active !== enabled) {
    await locator.click();
  }
}

async function setRangeValue(page, selector, value) {
  await page.locator(selector).evaluate((el, nextValue) => {
    el.value = String(nextValue);
    el.dispatchEvent(new Event('input', { bubbles: true }));
    el.dispatchEvent(new Event('change', { bubbles: true }));
  }, value);
}

async function setCheckbox(page, selector, checked) {
  await page.locator(selector).evaluate((el, nextChecked) => {
    el.checked = !!nextChecked;
    el.dispatchEvent(new Event('input', { bubbles: true }));
    el.dispatchEvent(new Event('change', { bubbles: true }));
  }, checked);
}

async function installDialogStubs(page) {
  await page.evaluate(() => {
    window.__dialogMessages = [];
    window.__promptResponse = 'TempPass!567';
    window.confirm = () => true;
    window.prompt = () => window.__promptResponse;
    window.alert = (message) => {
      window.__dialogMessages.push(String(message));
    };
  });
}

async function latestDialogMessage(page) {
  return await page.evaluate(() => {
    const messages = window.__dialogMessages || [];
    return messages.length > 0 ? messages[messages.length - 1] : '';
  });
}

async function clearDialogMessages(page) {
  await page.evaluate(() => {
    window.__dialogMessages = [];
  });
}

async function loginStatus(apiRequest, nextUsername, nextPassword) {
  const resp = await apiRequest.post('/api/auth/login', {
    data: { username: nextUsername, password: nextPassword }
  });
  return resp.status();
}

function normalizeNotificationRestore(original) {
  const payload = {
    enabled: original.enabled === true,
    provider: original.provider || 'slack',
    channel: original.channel || '',
    events: Array.isArray(original.events) ? original.events : []
  };

  if (original.provider === 'email') {
    payload.webhook_url = '';
    payload.smtp = {
      host: original.smtp?.host || '',
      port: original.smtp?.port || 587,
      username: original.smtp?.username || '',
      password: '',
      from: original.smtp?.from || '',
      to: Array.isArray(original.smtp?.to) ? original.smtp.to : [],
      use_tls: original.smtp?.use_tls !== false
    };
  } else {
    payload.webhook_url = '';
    payload.preserve_webhook_url = !!original.webhook_url;
  }

  return payload;
}

test('settings modal exercises each tab and persists configurable state', async ({ page, request }) => {
  const unique = Date.now();
  const tempModelName = `pw-e2e-model-${unique}`;
  const tempMCPName = `pw-e2e-mcp-${unique}`;
  const tempUser = `pw_e2e_user_${unique}`;
  const tempRole = `pw_e2e_role_${unique}`;

  await login(page);
  await installDialogStubs(page);
  await openSettings(page);

  const originalSettings = (await fetchJSON(page, '/api/settings')).data;
  const originalToolApproval = (await fetchJSON(page, '/api/settings/tool-approval')).data;
  const originalAgent = (await fetchJSON(page, '/api/settings/agent')).data;
  const originalPrometheus = (await fetchJSON(page, '/api/prometheus/settings')).data;
  const originalNotifications = (await fetchJSON(page, '/api/notifications/config')).data;
  const originalLocalState = await page.evaluate(() => ({
    theme: localStorage.getItem('k13d_theme'),
    streaming: localStorage.getItem('k13d_use_streaming'),
    autoRefresh: localStorage.getItem('k13d_auto_refresh'),
    refreshInterval: localStorage.getItem('k13d_refresh_interval'),
    guardrails: localStorage.getItem('k13d-guardrails'),
    securityPreferences: localStorage.getItem('k13d-security-preferences')
  }));

  try {
    console.log('[settings] general');
    await switchTab(page, 'General', '#settings-general');

    const desiredLanguage = originalSettings.language === 'en' ? 'ko' : 'en';
    const desiredLogLevel = originalSettings.log_level === 'info' ? 'debug' : 'info';
    const desiredTimezone = originalSettings.timezone === 'UTC' ? 'Asia/Seoul' : 'UTC';

    await page.selectOption('#setting-language', desiredLanguage);
    await page.selectOption('#setting-log-level', desiredLogLevel);
    await page.selectOption('#setting-timezone', desiredTimezone);
    await setClassToggle(page, '#setting-streaming', false);
    await setClassToggle(page, '#setting-auto-refresh', false);
    await page.selectOption('#setting-refresh-interval', '120');
    await page.selectOption('#setting-theme', 'tokyo-night');

    await page.locator('#settings-modal .modal-footer').getByRole('button', { name: 'Save' }).click();
    await expect(page.locator('#settings-modal')).toBeHidden();

    const savedSettings = (await fetchJSON(page, '/api/settings')).data;
    expect(savedSettings.language).toBe(desiredLanguage);
    expect(savedSettings.log_level).toBe(desiredLogLevel);
    expect(savedSettings.timezone).toBe(desiredTimezone);

    const currentLocalState = await page.evaluate(() => ({
      theme: localStorage.getItem('k13d_theme'),
      streaming: localStorage.getItem('k13d_use_streaming'),
      autoRefresh: localStorage.getItem('k13d_auto_refresh'),
      refreshInterval: localStorage.getItem('k13d_refresh_interval')
    }));
    expect(currentLocalState.theme).toBe('tokyo-night');
    expect(currentLocalState.streaming).toBe('false');
    expect(currentLocalState.autoRefresh).toBe('false');
    expect(currentLocalState.refreshInterval).toBe('120');

    console.log('[settings] ai');
    await openSettings(page);
    await switchTab(page, 'AI', '#settings-ai');
    await expect(page.locator('#setting-llm-provider')).not.toHaveValue('');
    await expect.poll(async () => page.locator('#setting-llm-model').inputValue()).not.toBe('');

    await page.getByRole('button', { name: /Add Model Profile/i }).click();
    await page.fill('#new-model-name', tempModelName);
    await page.selectOption('#new-model-provider', 'ollama');
    await page.fill('#new-model-model', 'llama3.2');
    await page.fill('#new-model-endpoint', 'http://localhost:11434');
    await page.fill('#new-model-description', 'Playwright temporary model');
    await page.evaluate(() => window.updateNewModelToolSupportWarning && window.updateNewModelToolSupportWarning());
    await expect(page.locator('#new-model-tool-support-warning')).toContainText(/tools\/function calling/i);
    await page.locator('#add-model-form').getByRole('button', { name: 'Add Profile' }).click();
    await expect(page.locator('#models-list')).toContainText(tempModelName);

    const modelRow = page.locator('#models-list .settings-row').filter({ hasText: tempModelName });
    await modelRow.getByRole('button', { name: 'Delete' }).click();
    await expect(page.locator('#models-list')).not.toContainText(tempModelName);

    await setClassToggle(page, '#ta-auto-approve-ro', true);
    await setClassToggle(page, '#ta-require-write', false);
    await setClassToggle(page, '#ta-block-dangerous', true);
    await setClassToggle(page, '#ta-require-unknown', false);
    await page.fill('#ta-timeout', '120');
    await page.fill('#ta-blocked-patterns', 'kubectl delete ns.*\nrm -rf.*');
    await page.getByRole('button', { name: 'Save Tool Approval Settings' }).click();

    const toolApproval = (await fetchJSON(page, '/api/settings/tool-approval')).data;
    expect(toolApproval.auto_approve_read_only).toBe(true);
    expect(toolApproval.require_approval_for_write).toBe(false);
    expect(toolApproval.block_dangerous).toBe(true);
    expect(toolApproval.require_approval_for_unknown).toBe(false);
    expect(toolApproval.approval_timeout_seconds).toBe(120);
    expect(toolApproval.blocked_patterns).toEqual(['kubectl delete ns.*', 'rm -rf.*']);

    await setRangeValue(page, '#agent-max-iterations', 7);
    await page.selectOption('#agent-reasoning-effort', 'high');
    await setRangeValue(page, '#agent-temperature', 120);
    await page.fill('#agent-max-tokens', '8192');
    await page.getByRole('button', { name: 'Save Agent Settings' }).click();

    const agentSettings = (await fetchJSON(page, '/api/settings/agent')).data;
    expect(agentSettings.max_iterations).toBe(7);
    expect(agentSettings.reasoning_effort).toBe('high');
    expect(agentSettings.temperature).toBe(1.2);
    expect(agentSettings.max_tokens).toBe(8192);

    await setClassToggle(page, '#guardrails-toggle', false);
    await setClassToggle(page, '#guardrails-strict-toggle', true);
    await setClassToggle(page, '#guardrails-auto-analyze', false);
    const guardrails = await page.evaluate(() => JSON.parse(localStorage.getItem('k13d-guardrails') || '{}'));
    expect(guardrails.enabled).toBe(false);
    expect(guardrails.strictMode).toBe(true);
    expect(guardrails.autoAnalyze).toBe(false);

    console.log('[settings] mcp');
    await switchTab(page, 'MCP', '#settings-mcp');
    await page.getByRole('button', { name: /Add MCP Server/i }).click();
    await page.fill('#new-mcp-name', tempMCPName);
    await page.fill('#new-mcp-command', 'echo');
    await page.fill('#new-mcp-args', 'hello');
    await page.fill('#new-mcp-description', 'Playwright temporary MCP');
    await page.uncheck('#new-mcp-enabled');
    await page.locator('#add-mcp-form').getByRole('button', { name: 'Add Server' }).click();
    await expect(page.locator('#mcp-servers-list')).toContainText(tempMCPName);
    await expect(page.locator('#mcp-servers-list')).toContainText(/DISABLED/i);

    const mcpRow = page.locator('#mcp-servers-list .settings-row').filter({ hasText: tempMCPName });
    await mcpRow.getByRole('button', { name: 'Enable' }).click();
    await expect(mcpRow).toContainText(/DISCONNECTED|Disable/i);
    if (await mcpRow.getByRole('button', { name: 'Reconnect' }).isVisible()) {
      await mcpRow.getByRole('button', { name: 'Reconnect' }).click();
      await expect.poll(() => latestDialogMessage(page)).toContain('Failed to reconnect');
    }
    await mcpRow.getByRole('button', { name: 'Disable' }).click();
    await expect(mcpRow).toContainText(/DISABLED/i);
    await mcpRow.getByRole('button', { name: 'Delete' }).click();
    await expect(page.locator('#mcp-servers-list')).not.toContainText(tempMCPName);

    console.log('[settings] security');
    await switchTab(page, 'Security', '#settings-security');
    await expect(page.locator('#trivy-status-text')).not.toHaveText('');
    await setCheckbox(page, '#security-scan-images', false);
    await page.selectOption('#security-min-severity', 'CRITICAL');
    const securityPreferences = await page.evaluate(() => JSON.parse(localStorage.getItem('k13d-security-preferences') || '{}'));
    expect(securityPreferences.scan_images).toBe(false);
    expect(securityPreferences.min_severity).toBe('CRITICAL');
    await page.locator('#settings-security').getByRole('button', { name: 'Refresh' }).click();

    console.log('[settings] metrics');
    await switchTab(page, 'Metrics', '#settings-metrics');
    await expect.poll(async () => (await fetchJSON(page, '/api/prometheus/settings')).data.collection_interval).toBe(originalPrometheus.collection_interval || 60);
    await setCheckbox(page, '#prometheus-expose-metrics', true);
    await page.fill('#prometheus-external-url', '');
    await setCheckbox(page, '#prometheus-collect-k8s', false);
    await page.selectOption('#prometheus-collection-interval', '300');
    await page.selectOption('#prometheus-retention-days', '14');
    await page.locator('#settings-metrics').getByRole('button', { name: 'Test Connection' }).click();
    await expect(page.locator('#prometheus-test-result')).toContainText(/Please enter a Prometheus URL/i);
    await page.locator('#settings-metrics').getByRole('button', { name: 'Save Prometheus Settings' }).click();

    await expect.poll(async () => (await fetchJSON(page, '/api/prometheus/settings')).data.expose_metrics).toBe(true);
    const prometheusSettings = (await fetchJSON(page, '/api/prometheus/settings')).data;
    expect(prometheusSettings.collect_k8s_metrics).toBe(false);
    expect(prometheusSettings.collection_interval).toBe(300);
    expect(prometheusSettings.metrics_retention_days).toBe(14);
    await expect(page.locator('#metrics-storage-info')).toContainText(/14 days/i);

    console.log('[settings] notifications');
    await switchTab(page, 'Notifications', '#settings-notifications');
    await expect.poll(async () => page.locator('#notif-platform').inputValue()).toBe(originalNotifications.provider || 'slack');
    await page.locator('#notif-platform').evaluate((el) => {
      el.value = 'email';
      el.dispatchEvent(new Event('change', { bubbles: true }));
    });
    await expect(page.locator('#notif-smtp-section')).toBeVisible();
    await page.getByRole('button', { name: 'Send Test' }).click();
    await expect(page.locator('#notif-test-result')).toContainText(/Failed:/i);

    console.log('[settings] admin');
    await switchTab(page, 'Admin', '#settings-admin');
    await expect(page.locator('#auth-save-btn')).toBeDisabled();
    await expect(page.locator('#auth-runtime-note')).toContainText(/read-only/i);

    await page.getByRole('button', { name: /Add User/i }).click();
    await page.fill('#new-user-username', tempUser);
    await page.fill('#new-user-password', 'TempPass!234');
    await page.fill('#new-user-email', 'temp@example.com');
    await page.selectOption('#new-user-role', 'viewer');
    await clearDialogMessages(page);
    await page.locator('#add-user-form').getByRole('button', { name: 'Add User' }).click();
    await expect
      .poll(async () => await page.locator('#admin-users-list .settings-row').filter({ hasText: tempUser }).count())
      .toBeGreaterThan(0);

    const userRow = page.locator('#admin-users-list .settings-row').filter({ hasText: tempUser });
    await clearDialogMessages(page);
    await userRow.getByRole('button', { name: 'Reset Password' }).click();
    await expect(page.locator('#reset-password-modal')).toBeVisible();
    await page.fill('#reset-password-input', 'TempPass!567');
    await page.locator('#reset-password-modal').getByRole('button', { name: 'Reset Password' }).click();
    await expect(page.locator('#reset-password-modal')).toBeHidden();
    await expect.poll(async () => await loginStatus(request, tempUser, 'TempPass!567')).toBe(200);

    await clearDialogMessages(page);
    await userRow.getByRole('button', { name: 'Delete' }).click();
    await expect
      .poll(async () => await page.locator('#admin-users-list .settings-row').filter({ hasText: tempUser }).count(), { timeout: 10000 })
      .toBe(0);
    await expect.poll(async () => await loginStatus(request, tempUser, 'TempPass!567')).toBe(401);

    await page.getByRole('button', { name: /Create Custom Role/i }).click();
    await expect(page.locator('#role-editor-modal')).toBeVisible();
    await page.fill('#new-role-name', tempRole);
    await page.fill('#new-role-desc', 'Playwright role');
    await page.locator('#new-role-features input[value="reports"]').uncheck();
    await page.locator('#role-editor-modal').getByRole('button', { name: 'Create' }).click();
    await expect.poll(async () => (await fetchJSON(page, `/api/roles/${encodeURIComponent(tempRole)}`)).status).toBe(200);
    const createdRole = (await fetchJSON(page, `/api/roles/${encodeURIComponent(tempRole)}`)).data;
    expect(createdRole.description).toBe('Playwright role');
    expect(createdRole.allowed_features).not.toContain('reports');

    await putJSON(page, `/api/roles/${encodeURIComponent(tempRole)}`, {
      description: 'Updated Playwright role',
      allowed_features: [...createdRole.allowed_features, 'reports'],
      is_custom: true
    });
    const updatedRole = (await fetchJSON(page, `/api/roles/${tempRole}`)).data;
    expect(updatedRole.description).toBe('Updated Playwright role');
    expect(updatedRole.allowed_features).toContain('reports');
    await fetchJSON(page, `/api/roles/${encodeURIComponent(tempRole)}`, { method: 'DELETE' });
    await expect.poll(async () => (await fetchJSON(page, `/api/roles/${encodeURIComponent(tempRole)}`)).status).toBe(404);

    console.log('[settings] about');
    await switchTab(page, 'About', '#settings-about');
    await expect(page.locator('#about-version')).not.toContainText('Loading');
    await expect(page.locator('#about-build-time')).not.toContainText('Loading');

    await page.getByRole('button', { name: 'Close settings' }).click();
    await expect(page.locator('#settings-modal')).toBeHidden();
  } finally {
    await fetchJSON(page, `/api/models?name=${encodeURIComponent(tempModelName)}`, { method: 'DELETE' });
    await fetchJSON(page, `/api/mcp/servers?name=${encodeURIComponent(tempMCPName)}`, { method: 'DELETE' });
    await fetchJSON(page, `/api/admin/users/${encodeURIComponent(tempUser)}`, { method: 'DELETE' });
    await fetchJSON(page, `/api/roles/${encodeURIComponent(tempRole)}`, { method: 'DELETE' });

    await putJSON(page, '/api/settings/tool-approval', originalToolApproval);
    await putJSON(page, '/api/settings/agent', originalAgent);
    await putJSON(page, '/api/prometheus/settings', {
      expose_metrics: originalPrometheus.expose_metrics,
      external_url: originalPrometheus.external_url || '',
      collect_k8s_metrics: originalPrometheus.collect_k8s_metrics !== false,
      collection_interval: originalPrometheus.collection_interval || 60,
      metrics_retention_days: originalPrometheus.metrics_retention_days || 30
    });
    await postJSON(page, '/api/notifications/config', normalizeNotificationRestore(originalNotifications));
    await putJSON(page, '/api/settings', {
      language: originalSettings.language,
      beginner_mode: originalSettings.beginner_mode,
      enable_audit: originalSettings.enable_audit,
      log_level: originalSettings.log_level,
      timezone: originalSettings.timezone || 'auto'
    });

    await page.evaluate((state) => {
      const setOrRemove = (key, value) => {
        if (value === null || value === undefined) {
          localStorage.removeItem(key);
        } else {
          localStorage.setItem(key, value);
        }
      };
      setOrRemove('k13d_theme', state.theme);
      setOrRemove('k13d_use_streaming', state.streaming);
      setOrRemove('k13d_auto_refresh', state.autoRefresh);
      setOrRemove('k13d_refresh_interval', state.refreshInterval);
      setOrRemove('k13d-guardrails', state.guardrails);
      setOrRemove('k13d-security-preferences', state.securityPreferences);
    }, originalLocalState);
  }
});
