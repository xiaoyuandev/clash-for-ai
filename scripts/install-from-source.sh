#!/usr/bin/env bash

set -euo pipefail

REPO_URL="${RELAY_SWITCH_REPO_URL:-https://github.com/xiaoyuandev/relay-switch.git}"
BRANCH="${RELAY_SWITCH_BRANCH:-main}"
INSTALL_ROOT="${RELAY_SWITCH_INSTALL_ROOT:-$HOME/.local/share/relay-switch}"
BIN_DIR="${RELAY_SWITCH_BIN_DIR:-$HOME/.local/bin}"
SERVICE_NAME="${RELAY_SWITCH_SERVICE_NAME:-relay-switch}"
HTTP_HOST="${RELAY_SWITCH_HTTP_HOST:-127.0.0.1}"
HTTP_PORT="${RELAY_SWITCH_HTTP_PORT:-3456}"
LOCAL_GATEWAY_PORT="${RELAY_SWITCH_LOCAL_GATEWAY_PORT:-3457}"
DATA_DIR="${RELAY_SWITCH_DATA_DIR:-$HOME/.local/share/relay-switch/data}"
RUNTIME_DATA_DIR="${RELAY_SWITCH_RUNTIME_DATA_DIR:-$DATA_DIR/local-gateway}"
RUNTIME_KIND="${RELAY_SWITCH_RUNTIME_KIND:-ai-mini-gateway}"
SYSTEMD_USER_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"

info() {
  printf '[relay-switch] %s\n' "$*"
}

fail() {
  printf '[relay-switch] error: %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

detect_shell_profile() {
  if [ -n "${ZDOTDIR:-}" ] && [ -f "${ZDOTDIR}/.zshrc" ]; then
    printf '%s\n' "${ZDOTDIR}/.zshrc"
    return
  fi
  if [ -f "$HOME/.zshrc" ]; then
    printf '%s\n' "$HOME/.zshrc"
    return
  fi
  if [ -f "$HOME/.bashrc" ]; then
    printf '%s\n' "$HOME/.bashrc"
    return
  fi
  printf '%s\n' "$HOME/.profile"
}

append_path_hint() {
  case ":$PATH:" in
    *":$BIN_DIR:"*) return 0 ;;
  esac

  local profile
  profile="$(detect_shell_profile)"
  mkdir -p "$(dirname "$profile")"
  touch "$profile"

  if ! grep -Fq "$BIN_DIR" "$profile"; then
    {
      printf '\n# Added by Relay Switch installer\n'
      printf 'export PATH="%s:$PATH"\n' "$BIN_DIR"
    } >>"$profile"
    info "added $BIN_DIR to PATH in $profile"
  fi
}

has_systemd_user() {
  command -v systemctl >/dev/null 2>&1 || return 1
  systemctl --user --version >/dev/null 2>&1 || return 1
}

setup_wsl_lingering_hint() {
  if grep -qi microsoft /proc/version 2>/dev/null; then
    info "WSL detected. If systemd user services are disabled, start manually with:"
    info "  $BIN_DIR/relay-switch run"
  fi
}

need_cmd git
need_cmd go
need_cmd pnpm

mkdir -p "$INSTALL_ROOT" "$BIN_DIR" "$DATA_DIR" "$RUNTIME_DATA_DIR"

SRC_DIR="$INSTALL_ROOT/src"
WEB_DIST_DIR="$INSTALL_ROOT/web"
CORE_BIN="$INSTALL_ROOT/bin/relay-switch-core"
GATEWAY_BIN="$INSTALL_ROOT/bin/ai-mini-gateway"
ENV_FILE="$INSTALL_ROOT/relay-switch.env"
SERVICE_FILE="$SYSTEMD_USER_DIR/${SERVICE_NAME}.service"
LAUNCHER="$BIN_DIR/relay-switch"

if [ -d "$SRC_DIR/.git" ]; then
  info "updating source tree in $SRC_DIR"
  git -C "$SRC_DIR" fetch --depth=1 origin "$BRANCH"
  git -C "$SRC_DIR" checkout "$BRANCH"
  git -C "$SRC_DIR" reset --hard "origin/$BRANCH"
else
  info "cloning source tree into $SRC_DIR"
  rm -rf "$SRC_DIR"
  git clone --depth=1 --branch "$BRANCH" "$REPO_URL" "$SRC_DIR"
fi

info "installing JavaScript dependencies"
pnpm install --dir "$SRC_DIR" --frozen-lockfile

info "building web management UI"
pnpm --dir "$SRC_DIR" --filter web build

info "building core binary"
mkdir -p "$(dirname "$CORE_BIN")"
(
  cd "$SRC_DIR/core"
  go build -o "$CORE_BIN" ./cmd/relay-switch-core
)

info "preparing bundled ai-mini-gateway runtime"
pnpm --dir "$SRC_DIR" --filter desktop prepare:ai-mini-gateway-runtime

RUNTIME_SOURCE="$SRC_DIR/apps/desktop/resources/ai-mini-gateway/bin/ai-mini-gateway"
[ -f "$RUNTIME_SOURCE" ] || fail "runtime binary not found at $RUNTIME_SOURCE"
install -m 0755 "$RUNTIME_SOURCE" "$GATEWAY_BIN"

rm -rf "$WEB_DIST_DIR"
mkdir -p "$WEB_DIST_DIR"
cp -R "$SRC_DIR/apps/web/dist/." "$WEB_DIST_DIR/"

cat >"$ENV_FILE" <<EOF
HTTP_PORT=$HTTP_PORT
HTTP_HOST=$HTTP_HOST
CORE_DATA_DIR=$DATA_DIR
WEB_ASSETS_DIR=$WEB_DIST_DIR
LOCAL_GATEWAY_RUNTIME_KIND=$RUNTIME_KIND
LOCAL_GATEWAY_RUNTIME_EXECUTABLE=$GATEWAY_BIN
LOCAL_GATEWAY_RUNTIME_HOST=127.0.0.1
LOCAL_GATEWAY_RUNTIME_PORT=$LOCAL_GATEWAY_PORT
LOCAL_GATEWAY_RUNTIME_DATA_DIR=$RUNTIME_DATA_DIR
EOF

cat >"$LAUNCHER" <<EOF
#!/usr/bin/env bash
set -euo pipefail

SERVICE_NAME="$SERVICE_NAME"
ENV_FILE="$ENV_FILE"
CORE_BIN="$CORE_BIN"

run_foreground() {
  set -a
  # shellcheck disable=SC1090
  source "\$ENV_FILE"
  set +a
  exec "\$CORE_BIN"
}

case "\${1:-start}" in
  start)
    if command -v systemctl >/dev/null 2>&1 && systemctl --user --version >/dev/null 2>&1; then
      systemctl --user start "\$SERVICE_NAME"
      systemctl --user --no-pager --full status "\$SERVICE_NAME" || true
    else
      run_foreground
    fi
    ;;
  stop)
    systemctl --user stop "\$SERVICE_NAME"
    ;;
  restart)
    systemctl --user restart "\$SERVICE_NAME"
    systemctl --user --no-pager --full status "\$SERVICE_NAME" || true
    ;;
  status)
    systemctl --user --no-pager --full status "\$SERVICE_NAME"
    ;;
  logs)
    journalctl --user -u "\$SERVICE_NAME" -n 200 -f
    ;;
  run)
    run_foreground
    ;;
  *)
    echo "usage: relay-switch {start|stop|restart|status|logs|run}" >&2
    exit 1
    ;;
esac
EOF
chmod +x "$LAUNCHER"

if has_systemd_user; then
  mkdir -p "$SYSTEMD_USER_DIR"
  cat >"$SERVICE_FILE" <<EOF
[Unit]
Description=Relay Switch core service
After=network.target

[Service]
Type=simple
EnvironmentFile=$ENV_FILE
ExecStart=$CORE_BIN
Restart=on-failure
RestartSec=3

[Install]
WantedBy=default.target
EOF

  systemctl --user daemon-reload
  systemctl --user enable --now "$SERVICE_NAME"
else
  info "systemd --user is unavailable; falling back to manual launcher only"
fi

append_path_hint
setup_wsl_lingering_hint

info "installation completed"
info "API endpoint: http://$HTTP_HOST:$HTTP_PORT/v1"
info "Web UI: http://$HTTP_HOST:$HTTP_PORT"
info "Launcher: $LAUNCHER"
if has_systemd_user; then
  info "Service: systemctl --user status $SERVICE_NAME"
fi
