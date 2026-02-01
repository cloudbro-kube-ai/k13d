// Copyright (c) k13d authors.
// E2E tests for k13d TUI using Microsoft tui-test

import { test, expect, Shell } from "@microsoft/tui-test";
import * as path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const k13dBinary = path.join(__dirname, "k13d");

// Configure to use bash shell and k13d binary
test.use({
  shell: Shell.Bash,
  program: { file: k13dBinary, args: ["--no-cluster"] },
});

test.describe("k13d startup", () => {
  test("shows application header", async ({ terminal }) => {
    // Wait for the app to render
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
  });

  test("shows default resource type", async ({ terminal }) => {
    // Should show pods view by default
    await expect(terminal.getByText(/pods/i)).toBeVisible({ timeout: 5000 });
  });
});

test.describe("keyboard navigation", () => {
  test("command mode with colon", async ({ terminal }) => {
    // Wait for app to load
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });

    // Enter command mode
    terminal.write(":");
    await expect(terminal.getByText(":")).toBeVisible();
  });

  test("filter mode with slash", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });

    // Enter filter mode
    terminal.write("/");
    await expect(terminal.getByText("/")).toBeVisible();
  });

  test("help modal with question mark", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });

    // Open help
    terminal.write("?");
    await expect(terminal.getByText(/help|shortcut|keys/i)).toBeVisible();

    // Close help with Escape
    terminal.keyEscape();
    await expect(terminal.getByText(/help|shortcut|keys/i)).not.toBeVisible();
  });
});

test.describe("resource switching", () => {
  test("switch to deployments with :deploy", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });

    terminal.submit(":deploy");
    await expect(terminal.getByText(/deploy/i)).toBeVisible();
  });

  test("switch to services with :svc", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });

    terminal.submit(":svc");
    await expect(terminal.getByText(/service/i)).toBeVisible();
  });

  test("switch to nodes with :nodes", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });

    terminal.submit(":nodes");
    await expect(terminal.getByText(/node/i)).toBeVisible();
  });

  test("switch to namespaces with :ns", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });

    terminal.submit(":ns");
    await expect(terminal.getByText(/namespace/i)).toBeVisible();
  });
});

test.describe("vim-style navigation", () => {
  test("j/k for up/down navigation", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });

    // Navigate down
    terminal.write("j");
    // Navigate up
    terminal.write("k");
    // Should not crash
    await expect(terminal.getByText("k13d")).toBeVisible();
  });

  test("g for go to top", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });

    // Go to top (gg)
    terminal.write("g");
    terminal.write("g");
    await expect(terminal.getByText("k13d")).toBeVisible();
  });

  test("G for go to bottom", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });

    // Go to bottom
    terminal.write("G");
    await expect(terminal.getByText("k13d")).toBeVisible();
  });
});

test.describe("AI assistant", () => {
  test("toggle AI panel with Tab", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });

    // Toggle AI panel
    terminal.keyTab();
    // Should not crash and panel should toggle
    await expect(terminal.getByText("k13d")).toBeVisible();
  });
});

test.describe("application lifecycle", () => {
  test("graceful quit with q", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });

    // Quit
    terminal.write("q");
    // App should exit gracefully
  });

  test("quit with Ctrl+C", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });

    // Ctrl+C
    terminal.keyCtrl("c");
    // App should exit gracefully
  });
});

test.describe("visual regression", () => {
  test("initial screen matches snapshot", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });

    // Take snapshot
    await expect(terminal).toMatchSnapshot();
  });

  test("help screen matches snapshot", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });

    terminal.write("?");
    await expect(terminal.getByText(/help|shortcut/i)).toBeVisible();

    await expect(terminal).toMatchSnapshot();
  });
});
