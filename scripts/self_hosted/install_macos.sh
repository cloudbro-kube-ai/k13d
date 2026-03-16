#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "This install script currently supports macOS only." >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DEPLOY_ROOT="${K13D_DEPLOY_ROOT:-$HOME/Library/Application Support/k13d-deploy}"
BIN_DIR="$DEPLOY_ROOT/bin"
LOG_DIR="$DEPLOY_ROOT/log"
CONFIG_DIR="${K13D_CONFIG_DIR:-$HOME/.config/k13d}"
ENV_FILE="${K13D_DEPLOY_ENV_FILE:-$CONFIG_DIR/deploy.env}"
CADDYFILE="${K13D_CADDYFILE:-$DEPLOY_ROOT/Caddyfile}"
RUN_WRAPPER="$DEPLOY_ROOT/run-k13d.sh"
DOMAIN="${K13D_DOMAIN:-fingerscore.net}"
INTERNAL_PORT="${K13D_INTERNAL_PORT:-18080}"
K13D_LABEL="${K13D_LAUNCH_LABEL:-net.fingerscore.k13d.web}"
CADDY_LABEL="${K13D_CADDY_LAUNCH_LABEL:-net.fingerscore.k13d.caddy}"
UID_VALUE="$(id -u)"

mkdir -p "$BIN_DIR" "$LOG_DIR" "$CONFIG_DIR"

if ! command -v brew >/dev/null 2>&1; then
  echo "Homebrew is required." >&2
  exit 1
fi

if ! command -v gh >/dev/null 2>&1; then
  echo "GitHub CLI (gh) is required." >&2
  exit 1
fi

if ! brew list caddy >/dev/null 2>&1; then
  brew install caddy
fi

if [[ ! -f "$ENV_FILE" ]]; then
  EXISTING_USER="$(ps -Ao command | awk '/k13d .* -web/ && /-admin-user/ {for (i=1; i<=NF; i++) if ($i == "-admin-user") print $(i+1)}' | tail -n1)"
  EXISTING_PASS="$(ps -Ao command | awk '/k13d .* -web/ && /-admin-password/ {for (i=1; i<=NF; i++) if ($i == "-admin-password") print $(i+1)}' | tail -n1)"
  ADMIN_USER="${EXISTING_USER:-admin}"
  ADMIN_PASS="${EXISTING_PASS:-$(openssl rand -base64 24 | tr -d '\n' | cut -c1-24)}"
  cat >"$ENV_FILE" <<EOF
K13D_USERNAME=$ADMIN_USER
K13D_PASSWORD=$ADMIN_PASS
EOF
  chmod 0600 "$ENV_FILE"
  echo "Created $ENV_FILE"
fi

pushd "$REPO_ROOT" >/dev/null
go run scripts/build-frontend.go
mkdir -p build
go build -o build/k13d ./cmd/kube-ai-dashboard-cli
popd >/dev/null

cp "$REPO_ROOT/build/k13d" "$BIN_DIR/k13d"
chmod 0755 "$BIN_DIR/k13d"

cat >"$RUN_WRAPPER" <<EOF
#!/bin/zsh
set -euo pipefail
export PATH="/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin"
if [[ -f "$ENV_FILE" ]]; then
  set -a
  source "$ENV_FILE"
  set +a
fi
exec "$BIN_DIR/k13d" --web --port "$INTERNAL_PORT" --auth-mode local --config "\${K13D_CONFIG:-$HOME/.config/k13d/config.yaml}"
EOF
chmod 0755 "$RUN_WRAPPER"

cat >"$CADDYFILE" <<EOF
$DOMAIN {
    encode zstd gzip
    reverse_proxy 127.0.0.1:$INTERNAL_PORT
}
EOF

K13D_PLIST="$HOME/Library/LaunchAgents/$K13D_LABEL.plist"
CADDY_PLIST="$HOME/Library/LaunchAgents/$CADDY_LABEL.plist"

cat >"$K13D_PLIST" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "https://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>$K13D_LABEL</string>
  <key>ProgramArguments</key>
  <array>
    <string>$RUN_WRAPPER</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>$LOG_DIR/k13d.stdout.log</string>
  <key>StandardErrorPath</key>
  <string>$LOG_DIR/k13d.stderr.log</string>
</dict>
</plist>
EOF

cat >"$CADDY_PLIST" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "https://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>$CADDY_LABEL</string>
  <key>ProgramArguments</key>
  <array>
    <string>/opt/homebrew/bin/caddy</string>
    <string>run</string>
    <string>--config</string>
    <string>$CADDYFILE</string>
    <string>--adapter</string>
    <string>caddyfile</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>$LOG_DIR/caddy.stdout.log</string>
  <key>StandardErrorPath</key>
  <string>$LOG_DIR/caddy.stderr.log</string>
</dict>
</plist>
EOF

launchctl bootout "gui/$UID_VALUE" "$K13D_PLIST" >/dev/null 2>&1 || true
launchctl bootstrap "gui/$UID_VALUE" "$K13D_PLIST"
launchctl kickstart -k "gui/$UID_VALUE/$K13D_LABEL"

for _ in {1..30}; do
  if curl -fsS "http://127.0.0.1:$INTERNAL_PORT/api/health" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

OLD_PORT80_PID="$(lsof -nP -iTCP:80 -sTCP:LISTEN -t 2>/dev/null || true)"
if [[ -n "$OLD_PORT80_PID" ]]; then
  OLD_CMD="$(ps -p "$OLD_PORT80_PID" -o command= || true)"
  if [[ "$OLD_CMD" == *"k13d"* && "$OLD_CMD" == *"-port 80"* ]]; then
    kill "$OLD_PORT80_PID" || true
    sleep 1
  fi
fi

launchctl bootout "gui/$UID_VALUE" "$CADDY_PLIST" >/dev/null 2>&1 || true
launchctl bootstrap "gui/$UID_VALUE" "$CADDY_PLIST"
launchctl kickstart -k "gui/$UID_VALUE/$CADDY_LABEL"

echo "Deployment services installed."
echo "  Domain:  https://$DOMAIN"
echo "  k13d:    http://127.0.0.1:$INTERNAL_PORT"
echo "  Env file: $ENV_FILE"
echo "  Logs:    $LOG_DIR"
echo "  Next:    $SCRIPT_DIR/install_runner_macos.sh"
