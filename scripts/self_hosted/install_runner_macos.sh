#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "This runner install script currently supports macOS only." >&2
  exit 1
fi

if ! command -v gh >/dev/null 2>&1; then
  echo "GitHub CLI (gh) is required." >&2
  exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required." >&2
  exit 1
fi

REPO="${K13D_RUNNER_REPO:-$(gh repo view --json nameWithOwner -q .nameWithOwner)}"
RUNNER_ROOT="${K13D_RUNNER_ROOT:-$HOME/Library/Application Support/k13d-deploy/github-runner}"
RUNNER_WORKDIR="${K13D_RUNNER_WORKDIR:-$HOME/Library/Application Support/k13d-deploy/github-runner-work}"
RUNNER_LOG_DIR="${K13D_RUNNER_LOG_DIR:-$HOME/Library/Application Support/k13d-deploy/log}"
RUNNER_NAME="${K13D_RUNNER_NAME:-$(hostname -s)-k13d-prod}"
RUNNER_LABELS="${K13D_RUNNER_LABELS:-k13d-prod,fingerscore}"
LAUNCH_LABEL="${K13D_RUNNER_LAUNCH_LABEL:-net.fingerscore.k13d.runner}"
UID_VALUE="$(id -u)"

mkdir -p "$RUNNER_ROOT" "$RUNNER_WORKDIR" "$RUNNER_LOG_DIR"

DOWNLOAD_URL="$(
  gh api "repos/$REPO/actions/runners/downloads" | python3 -c '
import json, sys
items = json.load(sys.stdin)
for item in items:
    if item.get("os") == "osx" and item.get("architecture") == "arm64":
        print(item["download_url"])
        break
'
)"

if [[ -z "$DOWNLOAD_URL" ]]; then
  echo "Failed to resolve a darwin-arm64 runner download URL." >&2
  exit 1
fi

ARCHIVE_NAME="$(basename "$DOWNLOAD_URL")"
ARCHIVE_PATH="$RUNNER_ROOT/$ARCHIVE_NAME"

if [[ ! -f "$RUNNER_ROOT/config.sh" ]]; then
  rm -rf "$RUNNER_ROOT"/*
  curl -L "$DOWNLOAD_URL" -o "$ARCHIVE_PATH"
  tar -xzf "$ARCHIVE_PATH" -C "$RUNNER_ROOT"
fi

REG_TOKEN="$(gh api "repos/$REPO/actions/runners/registration-token" --method POST -q .token)"
if [[ -z "$REG_TOKEN" ]]; then
  echo "Failed to obtain a runner registration token." >&2
  exit 1
fi

pushd "$RUNNER_ROOT" >/dev/null
./config.sh \
  --url "https://github.com/$REPO" \
  --token "$REG_TOKEN" \
  --name "$RUNNER_NAME" \
  --labels "$RUNNER_LABELS" \
  --work "$RUNNER_WORKDIR" \
  --unattended \
  --replace
popd >/dev/null

RUNNER_WRAPPER="$RUNNER_ROOT/run-launchd.sh"
cat >"$RUNNER_WRAPPER" <<EOF
#!/bin/zsh
set -euo pipefail
cd "$RUNNER_ROOT"
exec "$RUNNER_ROOT/run.sh"
EOF
chmod 0755 "$RUNNER_WRAPPER"

PLIST_PATH="$HOME/Library/LaunchAgents/$LAUNCH_LABEL.plist"
cat >"$PLIST_PATH" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "https://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>$LAUNCH_LABEL</string>
  <key>ProgramArguments</key>
  <array>
    <string>$RUNNER_WRAPPER</string>
  </array>
  <key>WorkingDirectory</key>
  <string>$RUNNER_ROOT</string>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>$RUNNER_LOG_DIR/github-runner.stdout.log</string>
  <key>StandardErrorPath</key>
  <string>$RUNNER_LOG_DIR/github-runner.stderr.log</string>
</dict>
</plist>
EOF

launchctl bootout "gui/$UID_VALUE" "$PLIST_PATH" >/dev/null 2>&1 || true
launchctl bootstrap "gui/$UID_VALUE" "$PLIST_PATH"
launchctl kickstart -k "gui/$UID_VALUE/$LAUNCH_LABEL"

echo "GitHub self-hosted runner installed:"
echo "  Repo:   $REPO"
echo "  Name:   $RUNNER_NAME"
echo "  Labels: $RUNNER_LABELS"
