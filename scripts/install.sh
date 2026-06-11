#!/usr/bin/env bash

set -euo pipefail

REPO="${RELAY_SWITCH_RELEASE_REPO:-xiaoyuandev/relay-switch}"
INSTALL_ROOT="${RELAY_SWITCH_INSTALL_ROOT:-$HOME/.local/share/relay-switch}"
BIN_DIR="${RELAY_SWITCH_BIN_DIR:-$HOME/.local/bin}"
SERVICE_NAME="${RELAY_SWITCH_SERVICE_NAME:-relay-switch}"
HTTP_HOST="${RELAY_SWITCH_HTTP_HOST:-127.0.0.1}"
HTTP_PORT="${RELAY_SWITCH_HTTP_PORT:-3456}"
LOCAL_GATEWAY_PORT="${RELAY_SWITCH_LOCAL_GATEWAY_PORT:-3457}"
DATA_DIR="${RELAY_SWITCH_DATA_DIR:-$HOME/.local/share/relay-switch/data}"
RUNTIME_DATA_DIR="${RELAY_SWITCH_RUNTIME_DATA_DIR:-$DATA_DIR/local-gateway}"
SYSTEMD_USER_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
REQUESTED_VERSION="${RELAY_SWITCH_VERSION:-}"

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

has_systemd_user() {
  command -v systemctl >/dev/null 2>&1 || return 1
  systemctl --user --version >/dev/null 2>&1 || return 1
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

resolve_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf 'amd64\n' ;;
    aarch64|arm64) printf 'arm64\n' ;;
    *) fail "unsupported architecture: $(uname -m)" ;;
  esac
}

resolve_version() {
  if [ -n "$REQUESTED_VERSION" ]; then
    printf '%s\n' "$REQUESTED_VERSION"
    return
  fi

  curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1
}

verify_archive_checksum() {
  local archive_path="$1"
  local checksum_path="$2"

  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$(dirname "$archive_path")" && sha256sum -c "$(basename "$checksum_path")" --ignore-missing)
    return
  fi

  if command -v shasum >/dev/null 2>&1; then
    local expected
    expected="$(grep " $(basename "$archive_path")\$" "$checksum_path" | awk '{print $1}')"
    [ -n "$expected" ] || fail "checksum entry not found for $(basename "$archive_path")"
    local actual
    actual="$(shasum -a 256 "$archive_path" | awk '{print $1}')"
    [ "$actual" = "$expected" ] || fail "checksum mismatch for $(basename "$archive_path")"
    return
  fi

  info "checksum tool not found; skipped verification"
}

setup_wsl_hint() {
  if grep -qi microsoft /proc/version 2>/dev/null; then
    info "WSL detected. Access the Web UI from Windows via http://localhost:$HTTP_PORT"
  fi
}

need_cmd curl
need_cmd tar

[ "$(uname -s)" = "Linux" ] || fail "this installer currently supports Linux/WSL only"

ARCH="$(resolve_arch)"
VERSION="$(resolve_version)"
[ -n "$VERSION" ] || fail "failed to resolve release version from GitHub"

case "$VERSION" in
  v*) ;;
  *) fail "release version must start with 'v', got: $VERSION" ;;
esac

ASSET_NAME="relay-switch-server_${VERSION}_linux_${ARCH}.tar.gz"
CHECKSUM_NAME="relay-switch-server_${VERSION}_SHA256SUMS.txt"
DOWNLOAD_BASE="https://github.com/$REPO/releases/download/$VERSION"

TMP_DIR="$(mktemp -d)"
ARCHIVE_PATH="$TMP_DIR/$ASSET_NAME"
CHECKSUM_PATH="$TMP_DIR/$CHECKSUM_NAME"

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

info "downloading $ASSET_NAME from release $VERSION"
curl -fL "$DOWNLOAD_BASE/$ASSET_NAME" -o "$ARCHIVE_PATH"

if [ -n "$REQUESTED_VERSION" ]; then
  info "using pinned release version $VERSION"
else
  info "using latest release version $VERSION"
fi

if curl -fsSL "$DOWNLOAD_BASE/$CHECKSUM_NAME" -o "$CHECKSUM_PATH"; then
  verify_archive_checksum "$ARCHIVE_PATH" "$CHECKSUM_PATH"
else
  info "checksum file not found; skipped verification"
fi

mkdir -p "$INSTALL_ROOT" "$BIN_DIR" "$DATA_DIR" "$RUNTIME_DATA_DIR"
rm -rf "$INSTALL_ROOT/release"
mkdir -p "$INSTALL_ROOT/release"
tar -xzf "$ARCHIVE_PATH" -C "$INSTALL_ROOT/release"

PACKAGE_DIR="$INSTALL_ROOT/release/relay-switch-server_${VERSION}_linux_${ARCH}"
[ -d "$PACKAGE_DIR" ] || fail "unexpected archive layout: $PACKAGE_DIR not found"

mkdir -p "$INSTALL_ROOT/bin" "$INSTALL_ROOT/web"
install -m 0755 "$PACKAGE_DIR/bin/relay-switch-core" "$INSTALL_ROOT/bin/relay-switch-core"
install -m 0755 "$PACKAGE_DIR/bin/ai-mini-gateway" "$INSTALL_ROOT/bin/ai-mini-gateway"
rm -rf "$INSTALL_ROOT/web"
mkdir -p "$INSTALL_ROOT/web"
cp -R "$PACKAGE_DIR/web/." "$INSTALL_ROOT/web/"
cp "$PACKAGE_DIR/release.json" "$INSTALL_ROOT/release.json"
cp "$PACKAGE_DIR/ai-mini-gateway-manifest.json" "$INSTALL_ROOT/ai-mini-gateway-manifest.json"

ENV_FILE="$INSTALL_ROOT/relay-switch.env"
SERVICE_FILE="$SYSTEMD_USER_DIR/${SERVICE_NAME}.service"
LAUNCHER="$BIN_DIR/relay-switch"

cat >"$ENV_FILE" <<EOF
HTTP_PORT=$HTTP_PORT
HTTP_HOST=$HTTP_HOST
CORE_DATA_DIR=$DATA_DIR
WEB_ASSETS_DIR=$INSTALL_ROOT/web
LOCAL_GATEWAY_RUNTIME_KIND=ai-mini-gateway
LOCAL_GATEWAY_RUNTIME_EXECUTABLE=$INSTALL_ROOT/bin/ai-mini-gateway
LOCAL_GATEWAY_RUNTIME_HOST=127.0.0.1
LOCAL_GATEWAY_RUNTIME_PORT=$LOCAL_GATEWAY_PORT
LOCAL_GATEWAY_RUNTIME_DATA_DIR=$RUNTIME_DATA_DIR
EOF

cat >"$LAUNCHER" <<EOF
#!/usr/bin/env bash
set -euo pipefail

SERVICE_NAME="$SERVICE_NAME"
ENV_FILE="$ENV_FILE"
CORE_BIN="$INSTALL_ROOT/bin/relay-switch-core"
EOF

cat >>"$LAUNCHER" <<'EOF'
info() {
  printf '[relay-switch] %s\n' "$*"
}

fail() {
  printf '[relay-switch] error: %s\n' "$*" >&2
  exit 1
}

has_systemd_user() {
  command -v systemctl >/dev/null 2>&1 || return 1
  systemctl --user --version >/dev/null 2>&1 || return 1
}

require_systemd_user() {
  has_systemd_user || fail "systemd --user is unavailable; use 'relay-switch run' to start in the foreground"
}

validate_port() {
  local port="$1"
  case "$port" in
    ''|*[!0-9]*) fail "HTTP port must be an integer from 1 to 65535, got: $port" ;;
  esac
  if [ "$port" -lt 1 ] || [ "$port" -gt 65535 ]; then
    fail "HTTP port must be an integer from 1 to 65535, got: $port"
  fi
}

set_env_value() {
  local key="$1"
  local value="$2"
  local tmp_file

  [ -f "$ENV_FILE" ] || fail "runtime env file not found: $ENV_FILE"
  tmp_file="$(mktemp "${ENV_FILE}.tmp.XXXXXX")"
  awk -v key="$key" -v value="$value" '
    BEGIN { replaced = 0 }
    $0 ~ "^" key "=" {
      print key "=" value
      replaced = 1
      next
    }
    { print }
    END {
      if (!replaced) {
        print key "=" value
      }
    }
  ' "$ENV_FILE" >"$tmp_file"
  mv "$tmp_file" "$ENV_FILE"
}

apply_runtime_overrides() {
  RUNTIME_CONFIG_CHANGED=0
  HTTP_HOST_CHANGED=0
  HTTP_PORT_CHANGED=0

  if [ -n "${RELAY_SWITCH_HTTP_HOST:-}" ]; then
    HTTP_HOST="$RELAY_SWITCH_HTTP_HOST"
    HTTP_HOST_CHANGED=1
    RUNTIME_CONFIG_CHANGED=1
  fi

  if [ -n "${RELAY_SWITCH_HTTP_PORT:-}" ]; then
    validate_port "$RELAY_SWITCH_HTTP_PORT"
    HTTP_PORT="$RELAY_SWITCH_HTTP_PORT"
    HTTP_PORT_CHANGED=1
    RUNTIME_CONFIG_CHANGED=1
  fi

  validate_port "$HTTP_PORT"
}

persist_runtime_overrides() {
  if [ "${HTTP_HOST_CHANGED:-0}" -eq 1 ]; then
    set_env_value "HTTP_HOST" "$HTTP_HOST"
  fi

  if [ "${HTTP_PORT_CHANGED:-0}" -eq 1 ]; then
    set_env_value "HTTP_PORT" "$HTTP_PORT"
  fi

  if [ "${RUNTIME_CONFIG_CHANGED:-0}" -eq 1 ]; then
    info "updated runtime config in $ENV_FILE"
  fi
}

prepare_runtime_config() {
  load_runtime_config
  apply_runtime_overrides
}

load_runtime_config() {
  [ -f "$ENV_FILE" ] || fail "runtime env file not found: $ENV_FILE"
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
  HTTP_HOST="${HTTP_HOST:-127.0.0.1}"
  HTTP_PORT="${HTTP_PORT:-3456}"
  validate_port "$HTTP_PORT"
}

port_is_listening() {
  local port="$1"

  if command -v ss >/dev/null 2>&1; then
    ss -H -ltn "sport = :$port" 2>/dev/null | grep . >/dev/null
    return
  fi

  if command -v lsof >/dev/null 2>&1; then
    lsof -n -P -iTCP:"$port" -sTCP:LISTEN 2>/dev/null | grep . >/dev/null
    return
  fi

  fail "cannot check port $port because neither 'ss' nor 'lsof' is available"
}

listener_pids_for_port() {
  local port="$1"
  local pids=""

  if command -v ss >/dev/null 2>&1; then
    pids="$(ss -H -ltnp "sport = :$port" 2>/dev/null | grep -Eo 'pid=[0-9]+' | cut -d= -f2 | sort -u || true)"
    if [ -n "$pids" ]; then
      printf '%s\n' "$pids"
      return
    fi
  fi

  if command -v lsof >/dev/null 2>&1; then
    lsof -t -n -P -iTCP:"$port" -sTCP:LISTEN 2>/dev/null | sort -u || true
  fi
}

service_is_active() {
  has_systemd_user && systemctl --user is-active --quiet "$SERVICE_NAME"
}

service_main_pid() {
  has_systemd_user || return 0
  systemctl --user show "$SERVICE_NAME" -p MainPID --value 2>/dev/null || true
}

pid_matches_core() {
  local pid="$1"
  local main_pid="${2:-}"
  local core_real
  local exe_real
  local cmdline

  if [ -n "$main_pid" ] && [ "$main_pid" != "0" ] && [ "$pid" = "$main_pid" ]; then
    return 0
  fi

  core_real="$(readlink -f "$CORE_BIN" 2>/dev/null || printf '%s' "$CORE_BIN")"
  exe_real="$(readlink -f "/proc/$pid/exe" 2>/dev/null || true)"
  if [ -n "$exe_real" ] && [ "$exe_real" = "$core_real" ]; then
    return 0
  fi

  if [ -r "/proc/$pid/cmdline" ]; then
    cmdline="$(tr '\0' ' ' <"/proc/$pid/cmdline" 2>/dev/null || true)"
    case "$cmdline" in
      *"$CORE_BIN"*) return 0 ;;
    esac
  fi

  return 1
}

pids_match_core_binary() {
  local pids="$1"
  local pid
  local matched=0

  while IFS= read -r pid; do
    [ -n "$pid" ] || continue
    matched=1
    pid_matches_core "$pid" "" || return 1
  done <<EOF_PIDS
$pids
EOF_PIDS

  [ "$matched" -eq 1 ]
}

port_owned_by_current_service() {
  local port="$1"
  local main_pid
  local pids
  local pid
  local matched=0

  service_is_active || return 1
  main_pid="$(service_main_pid)"
  pids="$(listener_pids_for_port "$port")"
  [ -n "$pids" ] || return 1

  while IFS= read -r pid; do
    [ -n "$pid" ] || continue
    matched=1
    pid_matches_core "$pid" "$main_pid" || return 1
  done <<EOF_PIDS
$pids
EOF_PIDS

  [ "$matched" -eq 1 ]
}

format_pids() {
  tr '\n' ',' | sed 's/,$//'
}

assert_port_available() {
  local port="$1"
  local allow_current_service="${2:-no}"
  local pids
  local pid_list

  if ! command -v ss >/dev/null 2>&1 && ! command -v lsof >/dev/null 2>&1; then
    info "warning: cannot pre-check port $port because neither 'ss' nor 'lsof' is available"
    return 0
  fi

  if ! port_is_listening "$port"; then
    return 0
  fi

  if [ "$allow_current_service" = "yes" ] && port_owned_by_current_service "$port"; then
    return 0
  fi

  pids="$(listener_pids_for_port "$port")"
  if [ -n "$pids" ]; then
    pid_list="$(printf '%s\n' "$pids" | format_pids)"
    if pids_match_core_binary "$pids"; then
      fail "port $port is already in use by relay-switch-core process PID(s): $pid_list. Stop that process or choose another port with RELAY_SWITCH_HTTP_PORT."
    fi
    fail "port $port is already in use by PID(s): $pid_list. Stop that process or choose another port with RELAY_SWITCH_HTTP_PORT."
  fi

  fail "port $port is already in use, but the owning process could not be identified. Stop the process, check with 'sudo ss -ltnp \"sport = :$port\"', or choose another port with RELAY_SWITCH_HTTP_PORT."
}

exec_core() {
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
  exec "$CORE_BIN"
}

run_foreground() {
  prepare_runtime_config
  assert_port_available "$HTTP_PORT" "no"
  persist_runtime_overrides
  exec_core
}

start_service() {
  prepare_runtime_config

  if has_systemd_user; then
    if service_is_active; then
      assert_port_available "$HTTP_PORT" "yes"
      persist_runtime_overrides
      info "$SERVICE_NAME is already running; restarting to apply current config"
      systemctl --user restart "$SERVICE_NAME"
    else
      assert_port_available "$HTTP_PORT" "no"
      persist_runtime_overrides
      systemctl --user start "$SERVICE_NAME"
    fi
    systemctl --user --no-pager --full status "$SERVICE_NAME" || true
  else
    assert_port_available "$HTTP_PORT" "no"
    persist_runtime_overrides
    exec_core
  fi
}

restart_service() {
  require_systemd_user
  prepare_runtime_config
  assert_port_available "$HTTP_PORT" "yes"
  persist_runtime_overrides
  systemctl --user restart "$SERVICE_NAME"
  systemctl --user --no-pager --full status "$SERVICE_NAME" || true
}

case "${1:-start}" in
  start)
    start_service
    ;;
  stop)
    require_systemd_user
    systemctl --user stop "$SERVICE_NAME"
    ;;
  restart)
    restart_service
    ;;
  status)
    require_systemd_user
    systemctl --user --no-pager --full status "$SERVICE_NAME"
    ;;
  logs)
    require_systemd_user
    journalctl --user -u "$SERVICE_NAME" -n 200 -f
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
ExecStart=$INSTALL_ROOT/bin/relay-switch-core
Restart=on-failure
RestartSec=3

[Install]
WantedBy=default.target
EOF

  systemctl --user daemon-reload
  systemctl --user enable "$SERVICE_NAME"
  "$LAUNCHER" start
else
  info "systemd --user is unavailable; use '$LAUNCHER run' to start manually"
fi

append_path_hint
setup_wsl_hint

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

info "installation completed"
info "Release: $VERSION"
info "API endpoint: http://$HTTP_HOST:$HTTP_PORT/v1"
info "Web UI: http://$HTTP_HOST:$HTTP_PORT"
