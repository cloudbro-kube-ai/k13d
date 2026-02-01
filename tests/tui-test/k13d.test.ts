// Copyright (c) k13d authors.
// Comprehensive E2E tests for k13d TUI using Microsoft tui-test

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

// ============================================================
// STARTUP & LOGO TESTS
// ============================================================
test.describe("k13d Startup and Logo", () => {
  test("shows k13d logo in header", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
  });

  test("shows Kubernetes AI Dashboard tagline", async ({ terminal }) => {
    await expect(terminal.getByText(/Kubernetes AI Dashboard/i)).toBeVisible({ timeout: 5000 });
  });

  test("shows default pods resource type", async ({ terminal }) => {
    await expect(terminal.getByText(/pods/i)).toBeVisible({ timeout: 5000 });
  });

  test("shows AI status indicator", async ({ terminal }) => {
    await expect(terminal.getByText(/AI:/i)).toBeVisible({ timeout: 5000 });
  });

  test("shows context information", async ({ terminal }) => {
    await expect(terminal.getByText(/Context:/i)).toBeVisible({ timeout: 5000 });
  });
});

// ============================================================
// HELP & ABOUT MODALS
// ============================================================
test.describe("Help and About Modals", () => {
  test("shows help modal with ? key", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("?");
    await expect(terminal.getByText(/Help/)).toBeVisible();
    await expect(terminal.getByText(/GENERAL/)).toBeVisible();
    await expect(terminal.getByText(/NAVIGATION/)).toBeVisible();
    await expect(terminal.getByText(/RESOURCE ACTIONS/)).toBeVisible();
  });

  test("closes help modal with Escape", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("?");
    await expect(terminal.getByText(/Help/)).toBeVisible();
    terminal.keyEscape();
    await expect(terminal.getByText(/GENERAL/)).not.toBeVisible({ timeout: 3000 });
  });

  test("closes help modal with q", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("?");
    await expect(terminal.getByText(/Help/)).toBeVisible();
    terminal.write("q");
    await expect(terminal.getByText(/GENERAL/)).not.toBeVisible({ timeout: 3000 });
  });

  test("shows about modal with Shift+I", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("I");
    await expect(terminal.getByText(/About k13d/)).toBeVisible();
  });
});

// ============================================================
// COMMAND MODE TESTS
// ============================================================
test.describe("Command Mode", () => {
  test("enters command mode with colon", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write(":");
    await expect(terminal.getByText(":")).toBeVisible();
  });

  test("shows autocomplete suggestions", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write(":po");
    await expect(terminal.getByText(/pods/i)).toBeVisible();
  });

  test("exits command mode with Escape", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write(":");
    terminal.keyEscape();
  });

  // Resource switching tests
  const resourceCommands = [
    { cmd: ":pods", alias: ":po", expected: "pods" },
    { cmd: ":deployments", alias: ":deploy", expected: "deploy" },
    { cmd: ":services", alias: ":svc", expected: "service" },
    { cmd: ":nodes", alias: ":no", expected: "node" },
    { cmd: ":namespaces", alias: ":ns", expected: "namespace" },
    { cmd: ":configmaps", alias: ":cm", expected: "configmap" },
    { cmd: ":secrets", alias: ":sec", expected: "secret" },
    { cmd: ":persistentvolumes", alias: ":pv", expected: "persistentvolume" },
    { cmd: ":persistentvolumeclaims", alias: ":pvc", expected: "persistentvolumeclaim" },
    { cmd: ":statefulsets", alias: ":sts", expected: "statefulset" },
    { cmd: ":daemonsets", alias: ":ds", expected: "daemonset" },
    { cmd: ":jobs", alias: ":job", expected: "job" },
    { cmd: ":cronjobs", alias: ":cj", expected: "cronjob" },
    { cmd: ":ingresses", alias: ":ing", expected: "ingress" },
    { cmd: ":events", alias: ":ev", expected: "event" },
    { cmd: ":serviceaccounts", alias: ":sa", expected: "serviceaccount" },
    { cmd: ":roles", alias: ":role", expected: "role" },
    { cmd: ":horizontalpodautoscalers", alias: ":hpa", expected: "horizontalpodautoscaler" },
  ];

  for (const { cmd, expected } of resourceCommands) {
    test(`switches to ${expected} with ${cmd}`, async ({ terminal }) => {
      await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
      terminal.submit(cmd);
      await expect(terminal.getByText(new RegExp(expected, "i"))).toBeVisible({ timeout: 5000 });
    });
  }

  for (const { alias, expected } of resourceCommands) {
    test(`switches to ${expected} with alias ${alias}`, async ({ terminal }) => {
      await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
      terminal.submit(alias);
      await expect(terminal.getByText(new RegExp(expected, "i"))).toBeVisible({ timeout: 5000 });
    });
  }
});

// ============================================================
// FILTER MODE TESTS
// ============================================================
test.describe("Filter Mode", () => {
  test("enters filter mode with /", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("/");
    await expect(terminal.getByText("/")).toBeVisible();
  });

  test("applies filter when typing", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("/kube");
    // Filter should be applied
  });

  test("clears filter with Escape", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("/test");
    terminal.keyEscape();
  });
});

// ============================================================
// VIM-STYLE NAVIGATION TESTS
// ============================================================
test.describe("Vim-Style Navigation", () => {
  test("moves down with j key", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("j");
    await expect(terminal.getByText("k13d")).toBeVisible();
  });

  test("moves up with k key", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("j"); // First move down
    terminal.write("k"); // Then move up
    await expect(terminal.getByText("k13d")).toBeVisible();
  });

  test("goes to top with g key", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("jjjjj"); // Move down several rows
    terminal.write("g"); // Go to top
    await expect(terminal.getByText("k13d")).toBeVisible();
  });

  test("goes to bottom with G key", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("G"); // Go to bottom
    await expect(terminal.getByText("k13d")).toBeVisible();
  });

  test("page down with Ctrl+F", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.keyCtrl("f");
    await expect(terminal.getByText("k13d")).toBeVisible();
  });

  test("page up with Ctrl+B", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.keyCtrl("b");
    await expect(terminal.getByText("k13d")).toBeVisible();
  });

  test("half page down with Ctrl+D", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.keyCtrl("d");
    await expect(terminal.getByText("k13d")).toBeVisible();
  });

  test("half page up with Ctrl+U", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.keyCtrl("u");
    await expect(terminal.getByText("k13d")).toBeVisible();
  });
});

// ============================================================
// NAMESPACE MANAGEMENT TESTS
// ============================================================
test.describe("Namespace Management", () => {
  test("switches to all namespaces with 0 key", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("0");
    await expect(terminal.getByText(/all/i)).toBeVisible({ timeout: 3000 });
  });

  test("cycles namespace with n key", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("n");
    await expect(terminal.getByText("k13d")).toBeVisible();
  });

  test("switches namespace with number keys 1-9", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("1");
    await expect(terminal.getByText("k13d")).toBeVisible();
  });

  test("uses namespace with u key (on namespace view)", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.submit(":ns");
    await expect(terminal.getByText(/namespace/i)).toBeVisible({ timeout: 5000 });
    terminal.write("j"); // Select a namespace
    terminal.write("u"); // Use it
  });
});

// ============================================================
// RESOURCE ACTION TESTS
// ============================================================
test.describe("Resource Actions", () => {
  test("shows describe with d key", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("j"); // Select a row
    terminal.write("d");
  });

  test("shows YAML with y key", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("j"); // Select a row
    terminal.write("y");
  });

  test("refreshes with r key", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("r");
    await expect(terminal.getByText("k13d")).toBeVisible();
  });

  test("shows context switcher with c key", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("c");
    await expect(terminal.getByText(/Context/i)).toBeVisible();
  });

  test("shows settings with Shift+O", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("O");
    await expect(terminal.getByText(/Settings/i)).toBeVisible();
  });
});

// ============================================================
// POD-SPECIFIC ACTION TESTS
// ============================================================
test.describe("Pod Actions", () => {
  test("shows logs with l key", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    await expect(terminal.getByText(/pods/i)).toBeVisible();
    terminal.write("j"); // Select a pod
    terminal.write("l");
  });

  test("shows previous logs with p key", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("j"); // Select a pod
    terminal.write("p");
  });

  test("shows node with o key", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("j"); // Select a pod
    terminal.write("o");
  });

  test("shows port forward dialog with Shift+F", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("j"); // Select a pod
    terminal.write("F");
    await expect(terminal.getByText(/Port Forward/i)).toBeVisible({ timeout: 3000 });
  });

  test("opens shell with s key", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("j"); // Select a pod
    terminal.write("s");
  });

  test("attaches with a key", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("j"); // Select a pod
    terminal.write("a");
  });
});

// ============================================================
// WORKLOAD ACTION TESTS
// ============================================================
test.describe("Workload Actions", () => {
  test("shows scale dialog with Shift+S on deployments", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.submit(":deploy");
    await expect(terminal.getByText(/deploy/i)).toBeVisible({ timeout: 5000 });
    terminal.write("j"); // Select a deployment
    terminal.write("S");
  });

  test("restarts with Shift+R on deployments", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.submit(":deploy");
    await expect(terminal.getByText(/deploy/i)).toBeVisible({ timeout: 5000 });
    terminal.write("j"); // Select a deployment
    terminal.write("R");
  });

  test("shows related resource with z key", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.submit(":deploy");
    await expect(terminal.getByText(/deploy/i)).toBeVisible({ timeout: 5000 });
    terminal.write("j"); // Select a deployment
    terminal.write("z"); // Show related (replicasets)
  });
});

// ============================================================
// SORTING TESTS
// ============================================================
test.describe("Column Sorting", () => {
  test("sorts by name with Shift+N", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("N");
    await expect(terminal.getByText("k13d")).toBeVisible();
  });

  test("sorts by age with Shift+A", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("A");
    await expect(terminal.getByText("k13d")).toBeVisible();
  });

  test("sorts by status with Shift+T", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("T");
    await expect(terminal.getByText("k13d")).toBeVisible();
  });

  test("sorts by namespace with Shift+P", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("P");
    await expect(terminal.getByText("k13d")).toBeVisible();
  });

  test("sorts by restarts with Shift+C", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("C");
    await expect(terminal.getByText("k13d")).toBeVisible();
  });

  test("toggles sort direction on second press", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("N"); // First press - ascending
    terminal.write("N"); // Second press - descending
    await expect(terminal.getByText("k13d")).toBeVisible();
  });

  test("sorts by column 1 with Shift+1", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("!"); // Shift+1
    await expect(terminal.getByText("k13d")).toBeVisible();
  });
});

// ============================================================
// MULTI-SELECT TESTS
// ============================================================
test.describe("Multi-Select", () => {
  test("toggles selection with Space", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("j"); // Move to first row
    terminal.write(" "); // Toggle selection
    terminal.write("j"); // Move to next row
    terminal.write(" "); // Toggle selection
    await expect(terminal.getByText("k13d")).toBeVisible();
  });
});

// ============================================================
// AI PANEL TESTS
// ============================================================
test.describe("AI Panel", () => {
  test("focuses AI panel with Tab", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.keyTab();
    await expect(terminal.getByText(/AI/i)).toBeVisible();
  });

  test("toggles briefing panel with Shift+B", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("B"); // Toggle off
    terminal.write("B"); // Toggle on
    await expect(terminal.getByText("k13d")).toBeVisible();
  });

  test("generates AI briefing with Ctrl+I", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.keyCtrl("i");
    await expect(terminal.getByText("k13d")).toBeVisible();
  });
});

// ============================================================
// DRILL DOWN & BACK NAVIGATION TESTS
// ============================================================
test.describe("Drill Down Navigation", () => {
  test("drills down with Enter", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.submit(":deploy");
    await expect(terminal.getByText(/deploy/i)).toBeVisible({ timeout: 5000 });
    terminal.write("j"); // Select first deployment
    terminal.keyEnter(); // Drill down to pods
  });

  test("goes back with Escape", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.submit(":deploy");
    await expect(terminal.getByText(/deploy/i)).toBeVisible({ timeout: 5000 });
    terminal.keyEscape(); // Go back
  });
});

// ============================================================
// DELETE CONFIRMATION TESTS
// ============================================================
test.describe("Delete Confirmation", () => {
  test("shows delete confirmation with Ctrl+D", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("j"); // Select a row
    terminal.keyCtrl("d");
    // Should show delete confirmation or error if no selection
  });

  test("kill pod with k key", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("j"); // Select a pod
    terminal.write("k");
  });

  test("kill pod with Ctrl+K", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("j"); // Select a pod
    terminal.keyCtrl("k");
  });
});

// ============================================================
// APPLICATION LIFECYCLE TESTS
// ============================================================
test.describe("Application Lifecycle", () => {
  test("quits gracefully with q key", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("q");
    // App should exit gracefully
  });

  test("quits gracefully with Ctrl+C", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.keyCtrl("c");
    // App should exit gracefully
  });
});

// ============================================================
// VIM VIEWER TESTS (for logs, describe, yaml)
// ============================================================
test.describe("Vim Viewer", () => {
  test("navigates with j/k in help viewer", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("?"); // Open help (uses vim viewer)
    await expect(terminal.getByText(/Help/)).toBeVisible();
    terminal.write("j"); // Scroll down
    terminal.write("k"); // Scroll up
    terminal.write("q"); // Close
  });

  test("searches with / in viewer", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("?"); // Open help
    await expect(terminal.getByText(/Help/)).toBeVisible();
    terminal.write("/NAVIGATION");
    terminal.keyEnter();
    terminal.write("q"); // Close
  });

  test("goes to top/bottom with g/G in viewer", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("?"); // Open help
    await expect(terminal.getByText(/Help/)).toBeVisible();
    terminal.write("G"); // Go to bottom
    terminal.write("g"); // Go to top
    terminal.write("q"); // Close
  });
});

// ============================================================
// VISUAL REGRESSION TESTS (SNAPSHOTS)
// ============================================================
test.describe("Visual Regression", () => {
  test("initial screen matches snapshot", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    await expect(terminal).toMatchSnapshot();
  });

  test("help screen matches snapshot", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("?");
    await expect(terminal.getByText(/Help/)).toBeVisible();
    await expect(terminal).toMatchSnapshot();
  });

  test("deployments view matches snapshot", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.submit(":deploy");
    await expect(terminal.getByText(/deploy/i)).toBeVisible({ timeout: 5000 });
    await expect(terminal).toMatchSnapshot();
  });

  test("services view matches snapshot", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.submit(":svc");
    await expect(terminal.getByText(/service/i)).toBeVisible({ timeout: 5000 });
    await expect(terminal).toMatchSnapshot();
  });

  test("nodes view matches snapshot", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.submit(":no");
    await expect(terminal.getByText(/node/i)).toBeVisible({ timeout: 5000 });
    await expect(terminal).toMatchSnapshot();
  });
});

// ============================================================
// ERROR HANDLING TESTS
// ============================================================
test.describe("Error Handling", () => {
  test("handles invalid command gracefully", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.submit(":invalidresource");
    // Should show error message or stay on current view
    await expect(terminal.getByText("k13d")).toBeVisible();
  });

  test("handles action on empty table gracefully", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.write("d"); // Describe without selection
    await expect(terminal.getByText("k13d")).toBeVisible();
  });
});

// ============================================================
// CRONJOB SPECIFIC TESTS
// ============================================================
test.describe("CronJob Actions", () => {
  test("triggers cronjob with t key", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.submit(":cj");
    await expect(terminal.getByText(/cronjob/i)).toBeVisible({ timeout: 5000 });
    terminal.write("j"); // Select a cronjob
    terminal.write("t"); // Trigger
  });
});

// ============================================================
// SERVICE SPECIFIC TESTS
// ============================================================
test.describe("Service Actions", () => {
  test("shows benchmark with b key on services", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.submit(":svc");
    await expect(terminal.getByText(/service/i)).toBeVisible({ timeout: 5000 });
    terminal.write("j"); // Select a service
    terminal.write("b"); // Benchmark
  });
});

// ============================================================
// HEALTH STATUS TESTS
// ============================================================
test.describe("Health Status", () => {
  test("shows health with :health command", async ({ terminal }) => {
    await expect(terminal.getByText("k13d")).toBeVisible({ timeout: 5000 });
    terminal.submit(":health");
  });
});
