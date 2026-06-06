#!/usr/bin/env bash
set -euo pipefail

CODEX_BIN="${K13D_CODEX_BIN:-codex}"
BASE_BRANCH="${K13D_GHA_BASE_BRANCH:-main}"
ISSUE_NUMBER="${K13D_GHA_ISSUE_NUMBER:-unknown}"
ISSUE_TITLE="${K13D_GHA_ISSUE_TITLE:-}"
ISSUE_URL="${K13D_GHA_ISSUE_URL:-}"
REPOSITORY="${K13D_GHA_REPOSITORY:-}"
REVIEW_LANGUAGE="${K13D_GHA_REVIEW_LANGUAGE:-ko}"
MODEL="${K13D_CODEX_REVIEW_MODEL:-}"

if ! command -v "$CODEX_BIN" >/dev/null 2>&1; then
  echo "Codex CLI is required for automated code review. Set K13D_CODEX_BIN or install codex." >&2
  exit 127
fi

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "run-agent-review.sh must run inside a git worktree." >&2
  exit 1
fi

REMOTE="${K13D_GHA_REMOTE:-origin}"
git fetch "$REMOTE" "$BASE_BRANCH" --quiet >/dev/null 2>&1 || true

BASE_REF="$BASE_BRANCH"
if git rev-parse --verify "$REMOTE/$BASE_BRANCH" >/dev/null 2>&1; then
  BASE_REF="$REMOTE/$BASE_BRANCH"
elif ! git rev-parse --verify "$BASE_REF" >/dev/null 2>&1; then
  BASE_REF="HEAD~1"
fi

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/k13d-codex-review.XXXXXX")"
trap 'rm -rf "$TMP_DIR"' EXIT

LAST_MESSAGE="$TMP_DIR/review.md"
LOG_FILE="$TMP_DIR/codex.log"

PROMPT="$(cat <<PROMPT
Review the changes for k13d GitHub issue #${ISSUE_NUMBER}.

Issue title: ${ISSUE_TITLE}
Issue URL: ${ISSUE_URL}
Repository: ${REPOSITORY}
Preferred review language: ${REVIEW_LANGUAGE}

Use a strict code-review mindset:
- Put findings first, ordered by severity.
- Focus on bugs, regressions, security risks, concurrency hazards, broken tests, and missing verification.
- Include file and line references when possible.
- If there are no findings, say that explicitly and mention residual risks or test gaps.
- Do not print secrets, tokens, or raw environment values.
- Write the final review in Korean when the preferred review language is Korean.
PROMPT
)"

ARGS=(exec review --base "$BASE_REF" --ephemeral --output-last-message "$LAST_MESSAGE")
if [[ -n "$(git status --porcelain)" ]]; then
  ARGS+=(--uncommitted)
fi
if [[ -n "$MODEL" ]]; then
  ARGS+=(-m "$MODEL")
fi

if ! printf '%s\n' "$PROMPT" | "$CODEX_BIN" "${ARGS[@]}" >"$LOG_FILE" 2>&1; then
  cat "$LOG_FILE" >&2
  exit 1
fi

if [[ -s "$LAST_MESSAGE" ]]; then
  cat "$LAST_MESSAGE"
else
  cat "$LOG_FILE"
fi
