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

  const deploymentsNav = page.locator('.nav-item[data-resource="deployments"]');
  await expect(deploymentsNav).toBeVisible();
  await deploymentsNav.click({ force: true });
  await expect(page.locator('#panel-title')).toHaveText(/Deployments/i);
  await expect(page.locator('#table-body tr[data-index]')).not.toHaveCount(0);
  await expect(page.locator('#table-body tr[data-spacer]')).toHaveCount(0);

  const initialRowTexts = await page.locator('#table-body tr').evaluateAll((rows) =>
    rows.map((row) => row.textContent.trim()).filter(Boolean)
  );
  expect(initialRowTexts.some((text) => text === '+')).toBeFalsy();

  await page.evaluate((value) => {
    document.getElementById('filter-input').value = value;
    currentFilter = value.toLowerCase();
    applyFilterAndSort();
  }, 'zzz-no-match-deployments');
  await expect(page.locator('#table-body')).toContainText(/No deployments found/i);
  await expect(page.locator('#table-body .add-context-btn')).toHaveCount(0);
  await page.evaluate(() => {
    document.getElementById('filter-input').value = '';
    currentFilter = '';
    applyFilterAndSort();
  });

  const deploymentWithPods = await page.evaluate(async () => {
    const depResp = await fetchWithAuth('/api/k8s/deployments');
    const depData = await depResp.json();
    for (const item of depData.items || []) {
      if (!item.selector || item.selector === '*') continue;
      const podsResp = await fetchWithAuth(`/api/k8s/pods?namespace=${encodeURIComponent(item.namespace || '')}&labelSelector=${encodeURIComponent(item.selector)}`);
      const podsData = await podsResp.json();
      if ((podsData.items || []).length > 0) {
        return {
          name: item.name,
          namespace: item.namespace || '',
          selector: item.selector,
          podCount: podsData.items.length
        };
      }
    }
    return null;
  });
  expect(deploymentWithPods).not.toBeNull();

  await page.evaluate((value) => {
    document.getElementById('filter-input').value = value;
    currentFilter = value.toLowerCase();
    applyFilterAndSort();
  }, deploymentWithPods.name);

  await page.evaluate((deployment) => {
    currentResource = 'deployments';
    showResourceDetail(deployment);
  }, deploymentWithPods);
  await expect(page.locator('#detail-modal')).toBeVisible();

  await page.locator('#detail-pods-tab').click();
  await expect(page.locator('#detail-pods')).toContainText(`Selector: ${deploymentWithPods.selector}`);
  await expect(page.locator('#detail-pods tbody tr')).toHaveCount(deploymentWithPods.podCount);

  await page.locator('#detail-modal .detail-tab').getByText('Events', { exact: true }).click();
  await expect(page.locator('#detail-events')).not.toContainText(/Error loading events/i);
  await page.locator('#detail-modal .modal-close').click();
  await expect(page.locator('#detail-modal')).toBeHidden();
  await page.evaluate(() => {
    document.getElementById('filter-input').value = '';
    currentFilter = '';
    applyFilterAndSort();
  });

  await openCustomView(page, 'overview', '#overview-container', 'showOverviewPanel');

  await openCustomView(page, 'applications', '#applications-container', 'showApplicationsView');

  await openCustomView(page, 'topology', '#topology-container', 'showTopology');

  await page.evaluate(() => {
    showTimelineView();
  });
  await expect(page.locator('#timeline-container')).toBeVisible();
  await expect(page.locator('#timeline-body')).not.toContainText(/Failed:/i);

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
