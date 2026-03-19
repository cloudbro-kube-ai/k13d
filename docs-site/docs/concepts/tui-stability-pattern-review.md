# TUI Stability Pattern Review

This note compares `k13d` against several CLI/TUI projects with the goal of improving:

- redraw stability and flicker behavior
- deadlock and race resistance
- configuration ergonomics and safety

The intent is not to copy foreign UX wholesale. We want to keep k13d's existing feature set and interaction model, while borrowing implementation patterns that improve stability.

## Scope

Repositories reviewed:

- `google-gemini/gemini-cli`
- `openclaw/openclaw`
- `anthropics/claude-code`
- `openai/codex`
- `derailed/k9s`

Focus areas:

- TUI redraw scheduling
- watcher and refresh serialization
- terminal mode and alternate-screen behavior
- config layering, validation, and operator diagnostics

## Current k13d Baseline

Relevant local files:

- `pkg/ui/app.go`
- `pkg/ui/app_layout.go`
- `pkg/ui/app_navigation.go`
- `pkg/ui/app_resources.go`
- `pkg/ui/app_ai.go`
- `pkg/ui/screen_manager.go`
- `pkg/ui/stability_regression_test.go`
- `pkg/config/config.go`
- `pkg/config/views.go`
- `pkg/config/hotkeys.go`
- `pkg/config/aliases.go`
- `pkg/config/plugins.go`
- `pkg/config/styles.go`

Strengths already present in k13d:

- `navigateTo()` centralizes resource and namespace transitions, which already removed several ABBA-style deadlocks.
- `requestSync()` plus `SetBeforeDrawFunc()` gives us a manual recovery path for screen ghosting.
- `flashSeq` prevents stale timers from clearing newer flash messages.
- `QueueUpdateDraw()` avoids calling into `tview` after stop and avoids the old deadlocking atomic batching layer.
- `stability_regression_test.go` already captures several historical deadlocks and race regressions.
- config is already split into focused files such as `views.yaml`, `hotkeys.yaml`, `aliases.yaml`, `plugins.yaml`, and styles.

Current pain points visible in the code:

- redraw requests are still produced from many hot paths independently
- watcher refreshes can overlap with other refresh triggers
- AI streaming, spinner updates, and watch-driven refreshes all compete for the same draw path
- screen sync is reactive, but redraw scheduling is not yet centrally coalesced
- config is split, but layering and validation are still much simpler than the best references

## Repository Findings

### 1. Codex

Most useful for low-level TUI stability.

Key files:

- `codex-rs/tui/src/tui/frame_requester.rs`
- `codex-rs/tui/src/tui/frame_rate_limiter.rs`
- `codex-rs/tui/src/tui.rs`
- `codex-rs/tui/src/custom_terminal.rs`
- `docs/tui-alternate-screen.md`

Patterns worth borrowing:

- A dedicated frame requester actor coalesces many draw requests into one scheduled redraw.
- Frame emission is rate-limited, which prevents animation and background events from flooding the renderer.
- Terminal mode setup and restoration are explicit and symmetrical.
- Startup flushes buffered terminal input before the interactive loop begins.
- Alternate screen behavior is user-configurable with `auto`, `always`, and `never`.
- `auto` mode makes terminal-multiplexer-specific decisions instead of assuming fullscreen mode is always best.

Why this matters for k13d:

- k13d currently lets many goroutines call `QueueUpdateDraw()` directly.
- A draw scheduler would reduce burst redraws from loading spinners, AI streaming, flash messages, and watcher callbacks.
- Alternate-screen policy would give users a supported escape hatch for scrollback or terminal compatibility complaints instead of forcing one behavior.

Best adoption candidates:

- `FrameRequester`-style redraw broker in front of `QueueUpdateDraw()`
- a small FPS clamp for non-critical redraws
- `tui.alternate_screen: auto|always|never`
- optional `--no-alt-screen`
- startup input flush before first render

### 2. Gemini CLI

Most useful for rendering policy and config layering.

Key files:

- `packages/cli/src/interactiveCli.tsx`
- `packages/cli/src/config/settings.ts`
- `packages/cli/src/ui/contexts/ScrollProvider.tsx`
- `packages/cli/src/ui/components/shared/VirtualizedList.tsx`
- `docs/cli/settings.md`

Patterns worth borrowing:

- Rendering mode is explicitly configurable.
- Incremental rendering is framed as a tradeoff: less flicker, but possible artifacts.
- Alternate buffer usage is environment-aware.
- Slow renders are measured and logged.
- Workspace settings are only merged when the workspace is trusted.
- High-churn lists are virtualized instead of redrawing full history every time.

Why this matters for k13d:

- k13d already has a manual `screen.Sync()` recovery path, but it has no concept of safe vs incremental render policy.
- the AI panel is a good candidate for draw coalescing or viewport-only updates
- k13d config does not yet distinguish global defaults from workspace- or cluster-specific overrides with trust semantics

Best adoption candidates:

- `tui.render_mode: safe|incremental`
- slow redraw telemetry
- coalesced or virtualized AI transcript rendering
- environment-aware alternate-buffer policy
- layered settings merge with trust checks for workspace-local overrides

### 3. K9s

Most useful for Kubernetes-native watcher and config patterns.

Key files:

- `internal/config/files.go`
- `internal/config/config.go`
- `internal/config/alias.go`
- `internal/config/hotkey.go`
- `internal/config/plugin.go`
- `internal/config/views.go`
- `internal/model/table.go`
- `internal/watch/factory.go`
- `internal/ui/flash.go`

Patterns worth borrowing:

- Config, data, and state directories are separated cleanly with XDG-aware helpers.
- Many config files support global plus context-specific overlays.
- YAML is schema-validated.
- Plugins are loaded from multiple XDG roots and decoded with known-field checking.
- Table refresh is model-driven and guarded against redundant concurrent updates.
- Watch infrastructure is shared, bounded, and cache-aware rather than being managed ad hoc per screen.

Why this matters for k13d:

- k13d is already inspired by k9s and should keep leaning in that direction for TUI state ownership.
- config file splitting is already present, but k13d lacks the same level of context overlay and validation maturity.
- watcher lifecycle is still simpler and more direct than k9s, which is easier to reason about but also easier to overlap accidentally under load.

Best adoption candidates:

- global plus context-specific overlays for `aliases`, `hotkeys`, `plugins`, `views`, and `styles`
- schema validation for all split config files
- `k13d info` diagnostics command that prints active config, data, log, and state paths
- model-side refresh guards that drop or merge redundant updates
- a more explicit watch factory or watch session abstraction

### 4. OpenClaw

Most useful for async serialization and operational safety.

Key files:

- `src/memory/qmd-manager.ts`
- `src/channels/transport/stall-watchdog.ts`
- `src/cli/config-cli.ts`
- `src/config/io.ts`

Patterns worth borrowing:

- One in-flight update at a time, with forced refreshes queued behind the current run.
- Retry and backoff logic is explicit.
- Watchdog timers detect stalls and can trigger recovery behavior.
- Config mutations support dry-run validation before writing.
- Config path mutation defends against prototype-pollution-style invalid keys.
- Config IO includes backups, validation, and audit-oriented behavior.

Why this matters for k13d:

- watcher callbacks currently trigger `a.refresh()` directly via a goroutine
- the app can receive refresh pressure from navigation, user commands, AI tool effects, and watcher events at the same time
- config writes are functional today, but safer validation and mutation helpers would reduce operator mistakes

Best adoption candidates:

- serialized refresh manager with `pending`, `queued forced`, and `latest wins` semantics
- stall watchdog for watcher inactivity and long refreshes
- config dry-run mode for settings changes
- automatic config backup before write

### 5. Claude Code

Useful mostly for settings and plugin governance, not for TUI engine internals.

Key files:

- `examples/settings/README.md`
- `plugins/README.md`

What we can borrow:

- clearer organization of plugin packaging
- settings hierarchy conventions
- explicit policy surfaces for permissions and hooks

What we should not over-index on:

- the public repository does not expose the core TUI runtime
- it is not a strong source for blinking, rendering, or deadlock internals

## Recommended Adoption Matrix

### Flicker and Redraw Stability

Highest-value changes:

1. Add a redraw broker in front of `QueueUpdateDraw()`.
2. Coalesce repeated redraw requests within a short window.
3. Cap non-critical redraws to a fixed maximum rate.
4. Add an explicit render mode toggle for users who prefer correctness over smoothness, or vice versa.

Suggested mapping:

- Codex: frame requester and FPS limiter
- Gemini: incremental rendering policy and slow-render telemetry
- k13d local target files: `pkg/ui/app.go`, `pkg/ui/app_layout.go`, `pkg/ui/screen_manager.go`, `pkg/ui/app_ai.go`

### Deadlock and Concurrency Stability

Highest-value changes:

1. Replace direct multi-source refresh calls with a serialized refresh actor.
2. Distinguish ordinary refresh from forced refresh.
3. Add a stall watchdog for watchers and long-running refreshes.
4. Keep lock ordering rules documented and enforced by regression tests.

Suggested mapping:

- OpenClaw: pending update plus forced queue pattern
- K9s: model-side update guard and watch lifecycle discipline
- k13d local target files: `pkg/ui/app_resources.go`, `pkg/ui/app_navigation.go`, `pkg/ui/app.go`, `pkg/ui/stability_regression_test.go`

### Configuration Improvements

Highest-value changes:

1. Separate config, data, state, and log roots more clearly.
2. Add schema validation to all split YAML files.
3. Support context-specific overlays for per-cluster or per-context customizations.
4. Add a `k13d info` command for active paths and runtime source diagnostics.
5. Consider dry-run validation before writing config changes.

Suggested mapping:

- K9s: XDG layout, context overlays, schema validation, diagnostics
- Gemini: layered settings precedence and trust-aware workspace merges
- OpenClaw: dry-run validation and safer mutation helpers
- k13d local target files: `pkg/config/config.go`, `pkg/config/views.go`, `pkg/config/hotkeys.go`, `pkg/config/aliases.go`, `pkg/config/plugins.go`

## Concrete k13d Refactor Plan

### Phase 1: Stop Draw Flooding

Goal:

- reduce visible blinking without changing user-visible behavior

Changes:

- introduce a redraw scheduler abstraction
- route spinner, flash, AI streaming flushes, and watcher-driven UI updates through that scheduler
- log slow redraws and count coalesced redraws

Expected result:

- fewer back-to-back full-screen redraw bursts
- easier diagnosis of terminals that still flicker

### Phase 2: Serialize Refresh and Watch Recovery

Goal:

- make refresh behavior deterministic under concurrent triggers

Changes:

- create a single refresh pipeline with coalescing
- add forced refresh queue semantics for explicit user actions
- add watchdog metrics and fallback logging for stalled watches

Expected result:

- lower deadlock and race risk
- reduced refresh stampedes under heavy watch activity

### Phase 3: Harden Config

Goal:

- make configuration safer and more predictable for real operators

Changes:

- add schema validation for split YAML files
- add context-specific overlays
- extend runtime source diagnostics
- optionally add config backup and dry-run write validation

Expected result:

- fewer silent config failures
- better operator confidence when customizing k13d

## Recommended First Implementation Order

If we want maximum impact with minimal surface area, implement in this order:

1. Redraw broker and rate limiter
2. Serialized refresh manager
3. Watchdog for watch inactivity and long refresh
4. Alternate-screen and render-mode config
5. Config schema validation and context overlays

## Important Guardrails

- Do not import React or ratatui-style architecture into k13d just because Gemini or Codex use it.
- Do not replace k13d's existing UX with upstream UX.
- Do not overcomplicate watch infrastructure before adding serialization and observability.
- Do not introduce workspace-local config execution without a trust model.
- Do not add config layering without clear precedence rules and runtime diagnostics.

## Summary

The highest-leverage references are:

- Codex for redraw scheduling and alternate-screen policy
- Gemini CLI for render-mode policy, performance instrumentation, and config layering ideas
- K9s for watcher and configuration architecture
- OpenClaw for serialized async updates, watchdogs, and config safety

The public Claude Code repository is useful for settings and plugin packaging ideas, but it is not a strong source for low-level TUI stability internals.

## Additional TUI References

After the first pass, we also reviewed additional TUI applications that are especially strong in layout ergonomics and test discipline:

- `jesseduffield/lazygit`
- `jesseduffield/lazydocker`
- `charmbracelet/bubbletea`

These projects are less relevant for Kubernetes watch architecture than K9s, but they are very relevant for how a TUI stays understandable and testable as it grows.

### Lazygit

Most useful for:

- explicit busy/idle semantics for integration tests
- user-configurable panel geometry
- auto-generated keybinding documentation from the real binding graph
- reuse of integration flows for demo recordings

Patterns worth borrowing:

- A busy/idle task model lets tests wait for "work complete" instead of sleeping and retrying.
- Worker tasks are registered before the goroutine starts, which prevents false-idle windows in tests.
- Side-panel width is config-driven rather than being scattered as hardcoded view sizes.
- Split mode is configurable (`horizontal`, `vertical`, `flexible`) and treated as an operator preference.
- Keybinding cheat sheets are generated from actual bindings rather than manually maintained docs.
- Demo recordings piggyback on integration tests instead of relying on a totally separate automation stack.

Why this matters for k13d:

- Our TUI E2E coverage is already improving, but we still rely on sleeps in many places where an explicit "UI is idle" signal would be more deterministic.
- We just added a resizable AI side panel; Lazygit is a strong reference for eventually persisting that layout cleanly.
- Our help modal and docs are currently hand-maintained, which means keybinding drift is likely as the TUI grows.

Best adoption candidates:

- a lightweight busy/idle tracker for TUI tests and long-running refresh operations
- persisted `tui.ai_panel_width` or `tui.layout.ai_panel_width`
- future split policy options such as `fixed`, `ratio`, or `flexible`
- generated keybinding/help documentation from registered bindings
- replayable demo flows built from existing TUI journey tests

### Lazydocker

Most useful for:

- config schema ergonomics
- editor and subprocess boundary testing
- keybinding cheat-sheet generation from runtime bindings

Patterns worth borrowing:

- The config docs explicitly surface JSON schema support for editor IntelliSense.
- External editor launching is isolated behind a small OS-command boundary.
- Editor resolution is unit-tested with injected environment and command runners.
- Cheatsheet generation uses the app's actual initial keybindings rather than a separate static table.

Why this matters for k13d:

- Our `kubectl edit` style flows can become flaky in headless test environments if they inherit an interactive editor.
- We already saw a real local example of this: `go test ./pkg/ui` needed `EDITOR=true` to avoid a non-interactive `vim` path during headless execution.
- TUI keybinding docs are becoming richer, which increases the value of generation over manual duplication.

Best adoption candidates:

- a tiny editor-launch resolver with injectable env/command hooks
- explicit non-interactive editor behavior for headless tests
- schema hints and validation notes for split YAML config files
- generated TUI shortcut reference docs

### Bubble Tea

Most useful for:

- byte-level terminal golden tests
- explicit terminal capability toggles
- TTY-aware runtime setup

Patterns worth borrowing:

- Screen output tests compare the exact emitted escape sequences and terminal state changes to goldens.
- Tests force a known terminal profile and window size to avoid environmental drift.
- Alternate screen, mouse mode, cursor state, bracketed paste, and keyboard enhancements are all covered as renderer state transitions.
- TTY input/output setup is explicit and symmetrical.

Why this matters for k13d:

- We already use screen goldens for some TUI states, but we can go deeper for modal and renderer-state regressions.
- The new approval modal, redraw broker, and AI panel resizing all benefit from deterministic terminal-state verification.
- Future alternate-screen or render-mode settings will be easier to ship safely if the terminal mode transitions are golden-tested.

Best adoption candidates:

- renderer-state goldens for modal open/close, help overlays, and approval dialogs
- a dedicated render-mode regression suite
- stricter known-environment setup for simulation-screen based tests
- coverage for terminal capability toggles before we expose more user-facing render settings

## Updated Recommendation Matrix

In addition to the original priorities, the next-most-valuable TUI-specific borrowings now look like this:

1. Add a busy/idle tracker so journey tests can wait on deterministic UI quiescence instead of timing sleeps.
2. Persist the AI panel width once we decide the config shape and precedence rules.
3. Generate TUI keybinding docs and possibly help content from registered bindings.
4. Add deeper terminal-state golden tests around modals, overlays, and future alternate-screen settings.
5. Make editor-dependent paths explicitly headless-safe in tests.

## What We Already Adopted

The current k13d branch has already adopted two direct results of this review:

- a redraw broker in front of `QueueUpdateDraw()` to reduce draw flooding
- a Web-UI-style centered approval modal for TUI tool approvals
- a resizable right-side AI panel with `Alt+H`, `Alt+L`, and `Alt+0`

That means the remaining high-value follow-up work is now less about raw redraw stability and more about:

- refresh serialization and watchdogs
- deterministic busy/idle testing
- config-backed layout persistence
- generated help and shortcut documentation
