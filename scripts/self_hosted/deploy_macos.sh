#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "This deploy script currently supports macOS only." >&2
  exit 1
fi

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 /path/to/k13d-binary" >&2
  exit 1
fi

SOURCE_BIN="$1"
DEPLOY_ROOT="${K13D_DEPLOY_ROOT:-$HOME/Library/Application Support/k13d-deploy}"
BIN_DIR="$DEPLOY_ROOT/bin"
LOG_DIR="$DEPLOY_ROOT/log"
TARGET_BIN="$BIN_DIR/k13d"
LAUNCH_LABEL="${K13D_LAUNCH_LABEL:-net.fingerscore.k13d.web}"
UID_VALUE="$(id -u)"

mkdir -p "$BIN_DIR" "$LOG_DIR"

if [[ ! -f "$SOURCE_BIN" ]]; then
  echo "Binary not found: $SOURCE_BIN" >&2
  exit 1
fi

TMP_BIN="$TARGET_BIN.new"
cp "$SOURCE_BIN" "$TMP_BIN"
chmod 0755 "$TMP_BIN"
mv "$TMP_BIN" "$TARGET_BIN"

PLIST_PATH="$HOME/Library/LaunchAgents/$LAUNCH_LABEL.plist"
if [[ -f "$PLIST_PATH" ]]; then
  launchctl kickstart -k "gui/$UID_VALUE/$LAUNCH_LABEL" >/dev/null 2>&1 || true
fi

for _ in {1..30}; do
  if curl -fsS "http://127.0.0.1:${K13D_INTERNAL_PORT:-18080}/api/health" >/dev/null 2>&1; then
    echo "k13d deploy successful."
    exit 0
  fi
  sleep 1
done

echo "k13d health check did not become ready after deploy." >&2
exit 1
