// Copyright (c) k13d authors.
// TUI Test configuration for k13d E2E tests

import { defineConfig } from "@microsoft/tui-test";

export default defineConfig({
  retries: 2,
  trace: process.env.CI ? true : false,
  expect: {
    timeout: 10000, // 10 seconds for assertions
  },
});
