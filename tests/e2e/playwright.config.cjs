module.exports = {
  testDir: './',
  timeout: 120000,
  workers: 1,
  retries: 0,
  use: {
    baseURL: process.env.K13D_E2E_BASE_URL,
    browserName: 'chromium',
    headless: true,
    trace: 'off',
    screenshot: 'only-on-failure',
    video: 'off',
    viewport: { width: 1440, height: 1000 },
  },
};
