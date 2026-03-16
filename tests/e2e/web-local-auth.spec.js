const { test, expect } = require('@playwright/test');

const username = process.env.K13D_E2E_USERNAME;
const password = process.env.K13D_E2E_PASSWORD;
const expectedProvider = process.env.K13D_E2E_EXPECT_PROVIDER || 'openai';
const expectedModel = process.env.K13D_E2E_EXPECT_MODEL || 'test-model';

async function login(page) {
  await page.goto('/');
  await expect(page.locator('#login-page')).toBeVisible();
  await expect.poll(async () => page.evaluate(() => window.__AUTH_MODE__)).toBe('local');

  await page.fill('#login-username', username);
  await page.fill('#login-password', password);
  await page.press('#login-password', 'Enter');

  await expect(page.locator('#app')).toHaveClass(/active/);
  await expect(page.locator('#user-badge')).toHaveText(username);
  await expect(page.locator('#panel-title')).toHaveText(/Pods/i);
}

async function focusWorkspace(page) {
  await page.evaluate(() => {
    const active = document.activeElement;
    if (active && typeof active.blur === 'function') {
      active.blur();
    }
  });
}

test('local auth browser journey covers main web workflows', async ({ page }) => {
  await login(page);

  await expect(page.locator('#namespace-select option')).not.toHaveCount(0);
  await page.fill('#filter-input', 'zzz-no-match');
  await expect(page.locator('#filter-input')).toHaveValue('zzz-no-match');
  await page.fill('#filter-input', '');

  await focusWorkspace(page);
  await page.keyboard.type(':');
  await expect(page.locator('#command-bar-overlay')).toHaveClass(/active/);
  await page.fill('#command-input', 'services');
  await page.keyboard.press('Enter');
  await expect(page.locator('#panel-title')).toHaveText(/Services/);

  await focusWorkspace(page);
  await page.keyboard.press('2');
  await expect(page.locator('#panel-title')).toHaveText(/Deployments/);

  await page.locator('.nav-item[data-resource="overview"]').click();
  await expect(page.locator('#overview-container')).toBeVisible();

  await page.locator('.nav-item[data-resource="applications"]').click();
  await expect(page.locator('#applications-container')).toBeVisible();

  await page.locator('.nav-item[data-resource="topology"]').click();
  await expect(page.locator('#topology-container')).toBeVisible();

  await page.getByText('Reports', { exact: true }).click();
  await expect(page.locator('#reports-modal')).toHaveClass(/active/);
  await page.getByRole('button', { name: 'Preview Report' }).click();
  await expect(page.locator('#report-status')).toContainText(/Generating report preview/i);
  await page.getByRole('button', { name: 'Close reports' }).click();
  await expect(page.locator('#reports-modal')).not.toHaveClass(/active/);

  await page.getByRole('button', { name: 'Settings' }).click();
  await expect(page.locator('#settings-modal')).toHaveClass(/active/);

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
  await expect(page.locator('#settings-modal')).not.toHaveClass(/active/);

  const aiPanel = page.locator('#ai-panel');
  if (!(await aiPanel.isVisible())) {
    await page.locator('#ai-toggle-btn').click();
  }
  await expect(aiPanel).toBeVisible();

  await page.fill('#ai-input', 'hello from browser e2e');
  await page.keyboard.press('Enter');
  await expect(page.locator('#ai-messages')).toContainText(/AI Assistant Not Configured/i);

  await focusWorkspace(page);
  await page.keyboard.type('?');
  await expect(page.locator('#shortcuts-modal')).toHaveClass(/active/);
  await page.keyboard.press('Escape');
  await expect(page.locator('#shortcuts-modal')).not.toHaveClass(/active/);

  await page.locator('#logout-btn').click();
  await expect(page.locator('#login-page')).toBeVisible();
  await expect(page.locator('#app')).not.toHaveClass(/active/);
});
