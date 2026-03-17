const { test, expect } = require('@playwright/test');

const username = process.env.K13D_E2E_USERNAME;
const password = process.env.K13D_E2E_PASSWORD;
const expectedProvider = process.env.K13D_E2E_EXPECT_PROVIDER || 'openai';
const expectedModel = process.env.K13D_E2E_EXPECT_MODEL || 'test-model';

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
  await expect(page.locator('#panel-title')).toHaveText(/Pods/i);
}

async function openCustomView(page, resource, containerSelector, functionName) {
  const nav = page.locator(`.nav-item[data-resource="${resource}"]`);
  const container = page.locator(containerSelector);

  await expect(nav).toBeVisible();
  await nav.click({ force: true });

  try {
    await expect(container).toBeVisible({ timeout: 3000 });
    return;
  } catch (error) {
    await page.evaluate((name) => {
      const fn = window[name];
      if (typeof fn === 'function') {
        fn();
      }
    }, functionName);
    await expect(container).toBeVisible();
  }
}

test('local auth browser journey covers main web workflows', async ({ page }) => {
  await login(page);

  await expect(page.locator('#namespace-select option')).not.toHaveCount(0);
  await page.fill('#filter-input', 'zzz-no-match');
  await expect(page.locator('#filter-input')).toHaveValue('zzz-no-match');
  await page.fill('#filter-input', '');

  await openCustomView(page, 'overview', '#overview-container', 'showOverviewPanel');

  await openCustomView(page, 'applications', '#applications-container', 'showApplicationsView');

  await openCustomView(page, 'topology', '#topology-container', 'showTopology');

  await page.getByText('Reports', { exact: true }).click();
  await expect(page.locator('#reports-modal')).toBeVisible();
  await page.getByRole('button', { name: 'Preview Report' }).click();
  await expect(page.locator('#report-status')).toContainText(/Generating report preview/i);
  await page.getByRole('button', { name: 'Close reports' }).click();
  await expect(page.locator('#reports-modal')).toBeHidden();

  await page.getByRole('button', { name: 'Settings' }).click();
  await expect(page.locator('#settings-modal')).toBeVisible();

  await page.getByRole('tab', { name: 'AI' }).click();
  await expect(page.locator('#settings-ai')).toBeVisible();
  await expect(page.locator('#setting-llm-provider')).toHaveValue(expectedProvider);
  await expect.poll(async () => page.locator('#setting-llm-model').inputValue()).not.toBe('');
  await page.selectOption('#setting-llm-provider', 'ollama');
  await page.fill('#setting-llm-model', 'gpt-oss:20b');
  await page.evaluate(() => window.updateLLMToolSupportWarning && window.updateLLMToolSupportWarning());
  await expect(page.locator('#llm-tool-support-warning')).toContainText(/tools\/function calling/i);
  await page.getByRole('button', { name: /Add Model Profile/i }).click();
  await page.selectOption('#new-model-provider', 'ollama');
  await page.fill('#new-model-model', 'llama3.2');
  await page.evaluate(() => window.updateNewModelToolSupportWarning && window.updateNewModelToolSupportWarning());
  await expect(page.locator('#new-model-tool-support-warning')).toContainText(/gpt-oss:20b/i);
  await page.locator('#add-model-form').getByRole('button', { name: 'Cancel' }).click();

  await expect(page.locator('#admin-tab')).toBeVisible();
  await page.locator('#admin-tab').click();
  await expect(page.locator('#settings-admin')).toBeVisible();

  await page.getByRole('tab', { name: 'About' }).click();
  await expect(page.locator('#settings-about')).toBeVisible();
  await page.getByRole('button', { name: 'Close settings' }).click();
  await expect(page.locator('#settings-modal')).toBeHidden();

  const aiPanel = page.locator('#ai-panel');
  if (!(await aiPanel.isVisible())) {
    await page.locator('#ai-toggle-btn').click();
  }
  await expect(aiPanel).toBeVisible();

  const firstPrompt = `history first ${Date.now()}`;
  const secondPrompt = `history second ${Date.now()}`;

  await page.evaluate(({ firstPrompt, secondPrompt }) => {
    localStorage.setItem('k13d_query_history', JSON.stringify([firstPrompt, secondPrompt]));
  }, { firstPrompt, secondPrompt });
  await page.reload();
  await expect(page.locator('.top-bar')).toBeVisible();
  if (!(await aiPanel.isVisible())) {
    await page.locator('#ai-toggle-btn').click();
  }
  await expect(aiPanel).toBeVisible();

  const storedHistory = await page.evaluate(() => JSON.parse(localStorage.getItem('k13d_query_history') || '[]').slice(-2));
  expect(storedHistory).toEqual([firstPrompt, secondPrompt]);

  await page.fill('#ai-input', 'draft prompt');
  await page.keyboard.press('ArrowUp');
  await expect(page.locator('#ai-input')).toHaveValue(secondPrompt);
  await page.keyboard.press('ArrowUp');
  await expect(page.locator('#ai-input')).toHaveValue(firstPrompt);
  await page.keyboard.press('ArrowDown');
  await expect(page.locator('#ai-input')).toHaveValue(secondPrompt);
  await page.keyboard.press('ArrowDown');
  await expect(page.locator('#ai-input')).toHaveValue('draft prompt');

  await page.getByRole('button', { name: /keyboard shortcuts/i }).click();
  await expect(page.locator('#shortcuts-modal')).toBeVisible();
  await page.getByRole('button', { name: 'Close shortcuts' }).click();
  await expect(page.locator('#shortcuts-modal')).toBeHidden();

  await page.locator('#logout-btn').click();
  await expect(page.locator('#login-page')).toBeVisible();
});
