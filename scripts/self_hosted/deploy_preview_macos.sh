#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "This preview deploy script currently supports macOS only." >&2
  exit 1
fi

usage() {
  cat >&2 <<'EOF'
Usage:
  deploy_preview_macos.sh /path/to/k13d-binary
  deploy_preview_macos.sh --cleanup

Environment:
  K13D_PREVIEW_SLUG              Required preview slug, for example pr-109
  K13D_PREVIEW_URL_BASE          Public origin, default https://fingerscore.net
  K13D_PREVIEW_PATH_PREFIX       Public path prefix, default /previews
  K13D_PREVIEW_PORT              Optional explicit local port
  K13D_PREVIEW_PORT_BASE         Default 18100
  K13D_PREVIEW_PORT_RANGE        Default 2000
  K13D_DEPLOY_ENV_FILE           Defaults to ~/.config/k13d/deploy.env
  K13D_CADDYFILE                 Defaults to k13d deploy Caddyfile
EOF
}

CLEANUP=0
if [[ "${1:-}" == "--cleanup" ]]; then
  CLEANUP=1
  shift
fi

if [[ "$CLEANUP" -ne 1 && $# -lt 1 ]]; then
  usage
  exit 1
fi

SOURCE_BIN="${1:-}"
DEPLOY_ROOT="${K13D_DEPLOY_ROOT:-$HOME/Library/Application Support/k13d-deploy}"
PREVIEW_ROOT="${K13D_PREVIEW_ROOT:-$DEPLOY_ROOT/previews}"
CONFIG_DIR="${K13D_CONFIG_DIR:-$HOME/.config/k13d}"
ENV_FILE="${K13D_DEPLOY_ENV_FILE:-$CONFIG_DIR/deploy.env}"
CADDYFILE="${K13D_CADDYFILE:-$DEPLOY_ROOT/Caddyfile}"
CADDY_LABEL="${K13D_CADDY_LAUNCH_LABEL:-net.fingerscore.k13d.caddy}"
CADDY_SNIPPETS_DIR="${K13D_PREVIEW_CADDY_SNIPPETS_DIR:-$DEPLOY_ROOT/caddy/previews}"
DOMAIN="${K13D_DOMAIN:-fingerscore.net}"
PUBLIC_BASE="${K13D_PREVIEW_URL_BASE:-https://$DOMAIN}"
PATH_PREFIX="${K13D_PREVIEW_PATH_PREFIX:-/previews}"
PORT_BASE="${K13D_PREVIEW_PORT_BASE:-18100}"
PORT_RANGE="${K13D_PREVIEW_PORT_RANGE:-2000}"
INTERNAL_PORT="${K13D_INTERNAL_PORT:-18080}"
UID_VALUE="$(id -u)"

sanitize_slug() {
  printf '%s' "$1" |
    tr '[:upper:]' '[:lower:]' |
    sed -E 's|[^a-z0-9._-]+|-|g; s|^-+||; s|-+$||'
}

raw_slug="${K13D_PREVIEW_SLUG:-}"
if [[ -z "$raw_slug" ]]; then
  echo "K13D_PREVIEW_SLUG is required." >&2
  exit 1
fi

SLUG="$(sanitize_slug "$raw_slug")"
if [[ -z "$SLUG" ]]; then
  echo "K13D_PREVIEW_SLUG produced an empty slug after sanitization." >&2
  exit 1
fi

PATH_PREFIX="/${PATH_PREFIX#/}"
PATH_PREFIX="${PATH_PREFIX%/}"
PREVIEW_PATH="$PATH_PREFIX/$SLUG/"
PUBLIC_BASE="${PUBLIC_BASE%/}"
PREVIEW_URL="$PUBLIC_BASE$PREVIEW_PATH"
PREVIEW_DIR="$PREVIEW_ROOT/$SLUG"
BIN_DIR="$PREVIEW_DIR/bin"
LOG_DIR="$PREVIEW_DIR/log"
RUNTIME_DIR="$PREVIEW_DIR/runtime"
SNIPPET_PATH="$CADDY_SNIPPETS_DIR/$SLUG.caddy"
LABEL_TOKEN="$(printf '%s' "$SLUG" | sed -E 's|[^a-z0-9.-]+|-|g')"
LAUNCH_LABEL="${K13D_PREVIEW_LAUNCH_LABEL:-net.fingerscore.k13d.preview.$LABEL_TOKEN}"
PLIST_PATH="$HOME/Library/LaunchAgents/$LAUNCH_LABEL.plist"
METADATA_PATH="$PREVIEW_DIR/metadata.env"

port_in_use() {
  lsof -nP -iTCP:"$1" -sTCP:LISTEN >/dev/null 2>&1
}

resolve_port() {
  if [[ -n "${K13D_PREVIEW_PORT:-}" ]]; then
    printf '%s\n' "$K13D_PREVIEW_PORT"
    return
  fi
  if [[ -f "$METADATA_PATH" ]]; then
    # shellcheck disable=SC1090
    source "$METADATA_PATH"
    if [[ -n "${PORT:-}" ]]; then
      printf '%s\n' "$PORT"
      return
    fi
  fi

  local checksum start candidate
  checksum="$(printf '%s' "$SLUG" | cksum | awk '{print $1}')"
  start=$((PORT_BASE + checksum % PORT_RANGE))
  for offset in $(seq 0 199); do
    candidate=$((PORT_BASE + (checksum + offset) % PORT_RANGE))
    if ! port_in_use "$candidate"; then
      printf '%s\n' "$candidate"
      return
    fi
  done
  echo "No free preview port found near $start." >&2
  exit 1
}

ensure_caddyfile_import() {
  mkdir -p "$CADDY_SNIPPETS_DIR" "$(dirname "$CADDYFILE")"
  : >"$CADDY_SNIPPETS_DIR/00-empty.caddy"

  if [[ ! -f "$CADDYFILE" ]]; then
    cat >"$CADDYFILE" <<EOF
$DOMAIN {
    encode zstd gzip
    import "$CADDY_SNIPPETS_DIR/*.caddy"
    reverse_proxy 127.0.0.1:$INTERNAL_PORT
}
EOF
    return
  fi

  CADDYFILE="$CADDYFILE" CADDY_SNIPPETS_DIR="$CADDY_SNIPPETS_DIR" python3 <<'PY'
from pathlib import Path
import os

path = Path(os.environ["CADDYFILE"])
snippets = os.environ["CADDY_SNIPPETS_DIR"]
needle = f'import "{snippets}/*.caddy"'
text = path.read_text()
if needle in text:
    raise SystemExit(0)

lines = text.splitlines()
insert_at = None
for idx, line in enumerate(lines):
    stripped = line.strip()
    if stripped.startswith("encode "):
        insert_at = idx + 1
        break
if insert_at is None:
    for idx, line in enumerate(lines):
        if "{" in line:
            insert_at = idx + 1
            break
if insert_at is None:
    lines.extend(["", needle])
else:
    indent = "    "
    if insert_at > 0:
        previous = lines[insert_at - 1]
        indent = previous[: len(previous) - len(previous.lstrip())] or "    "
    lines.insert(insert_at, f"{indent}{needle}")
path.write_text("\n".join(lines) + "\n")
PY
}

reload_caddy() {
  if command -v caddy >/dev/null 2>&1; then
    caddy validate --config "$CADDYFILE" --adapter caddyfile >/dev/null
    if caddy reload --config "$CADDYFILE" --adapter caddyfile >/dev/null 2>&1; then
      return
    fi
  fi
  launchctl kickstart -k "gui/$UID_VALUE/$CADDY_LABEL" >/dev/null 2>&1 || true
}

cleanup_preview() {
  launchctl bootout "gui/$UID_VALUE" "$PLIST_PATH" >/dev/null 2>&1 || true
  rm -f "$PLIST_PATH" "$SNIPPET_PATH"
  rm -rf "$PREVIEW_DIR"
  ensure_caddyfile_import
  reload_caddy
  echo "Preview removed: $PREVIEW_URL"
}

if [[ "$CLEANUP" -eq 1 ]]; then
  cleanup_preview
  exit 0
fi

if [[ ! -f "$SOURCE_BIN" ]]; then
  echo "Binary not found: $SOURCE_BIN" >&2
  exit 1
fi

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Deploy env file not found: $ENV_FILE" >&2
  echo "Run scripts/self_hosted/install_macos.sh first or set K13D_DEPLOY_ENV_FILE." >&2
  exit 1
fi

# shellcheck disable=SC1090
source "$ENV_FILE"
: "${K13D_USERNAME:?K13D_USERNAME must be set in $ENV_FILE}"
: "${K13D_PASSWORD:?K13D_PASSWORD must be set in $ENV_FILE}"

PORT="$(resolve_port)"

mkdir -p "$BIN_DIR" "$LOG_DIR" "$RUNTIME_DIR" "$CADDY_SNIPPETS_DIR"
cp "$SOURCE_BIN" "$BIN_DIR/k13d"
chmod 0755 "$BIN_DIR/k13d"

cat >"$RUNTIME_DIR/config.yaml" <<'EOF'
language: ko
github_automation:
  enabled: false
EOF

cat >"$PREVIEW_DIR/run-k13d-preview.sh" <<EOF
#!/bin/zsh
set -euo pipefail
export PATH="/opt/homebrew/bin:/opt/homebrew/sbin:/usr/local/bin:/usr/local/sbin:/usr/bin:/bin:/usr/sbin:/sbin"
set -a
source "$ENV_FILE"
set +a
export XDG_CONFIG_HOME="$RUNTIME_DIR/xdg-config"
export XDG_DATA_HOME="$RUNTIME_DIR/xdg-data"
export XDG_CACHE_HOME="$RUNTIME_DIR/xdg-cache"
export K13D_DB_PATH="$RUNTIME_DIR/audit.db"
mkdir -p "\$XDG_CONFIG_HOME" "\$XDG_DATA_HOME" "\$XDG_CACHE_HOME"
exec "$BIN_DIR/k13d" --web --port "$PORT" --auth-mode local --config "$RUNTIME_DIR/config.yaml"
EOF
chmod 0755 "$PREVIEW_DIR/run-k13d-preview.sh"

cat >"$PLIST_PATH" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "https://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>$LAUNCH_LABEL</string>
  <key>ProgramArguments</key>
  <array>
    <string>$PREVIEW_DIR/run-k13d-preview.sh</string>
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

launchctl bootout "gui/$UID_VALUE" "$PLIST_PATH" >/dev/null 2>&1 || true
launchctl bootstrap "gui/$UID_VALUE" "$PLIST_PATH"
launchctl kickstart -k "gui/$UID_VALUE/$LAUNCH_LABEL"

for _ in {1..30}; do
  if curl -fsS "http://127.0.0.1:$PORT/api/health" >/dev/null 2>&1; then
    healthy=1
    break
  fi
  healthy=0
  sleep 1
done

if [[ "$healthy" -ne 1 ]]; then
  echo "Preview did not become ready on http://127.0.0.1:$PORT/api/health" >&2
  exit 1
fi

cat >"$SNIPPET_PATH" <<EOF
redir ${PREVIEW_PATH%/} $PREVIEW_PATH 308

handle_path ${PREVIEW_PATH}* {
    reverse_proxy 127.0.0.1:$PORT
}
EOF

cat >"$METADATA_PATH" <<EOF
SLUG=$SLUG
PORT=$PORT
PREVIEW_URL=$PREVIEW_URL
LAUNCH_LABEL=$LAUNCH_LABEL
EOF

ensure_caddyfile_import
reload_caddy

echo "K13D_PREVIEW_TARGET=http://127.0.0.1:$PORT"
echo "K13D_PREVIEW_URL=$PREVIEW_URL"
echo "Preview deployed: $PREVIEW_URL"
