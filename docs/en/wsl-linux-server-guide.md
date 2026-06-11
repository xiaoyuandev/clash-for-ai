# Relay Switch WSL / Linux Server Deployment Guide

This guide explains how to deploy Relay Switch on `WSL` or a plain `Linux server` and manage it from the browser.

## 1. Deployment Shape

The current WSL / Linux server flow includes:

1. `relay-switch-core`
2. bundled `ai-mini-gateway` runtime
3. browser-based management UI built from `apps/web`

The default endpoints after installation are:

1. OpenAI-compatible local endpoint: `http://127.0.0.1:3456/v1`
2. Web management UI: `http://127.0.0.1:3456`
3. local models gateway runtime: `http://127.0.0.1:3457/v1`

## 2. Prerequisites

The production installer downloads stable GitHub Release assets by default.

Required commands:

1. `curl`
2. `tar`

Recommended for checksum validation:

1. `sha256sum`
2. or `shasum`

## 3. One-Line Install

Latest release:

```bash
curl -fsSL https://raw.githubusercontent.com/xiaoyuandev/relay-switch/main/scripts/install.sh | bash
```

Pinned release:

```bash
curl -fsSL https://raw.githubusercontent.com/xiaoyuandev/relay-switch/main/scripts/install.sh | \
  env RELAY_SWITCH_VERSION=vX.Y.Z bash
```

Development-only source install:

```bash
curl -fsSL https://raw.githubusercontent.com/xiaoyuandev/relay-switch/main/scripts/install-from-source.sh | bash
```

Notes:

1. `scripts/install.sh` is the production installer.
2. `scripts/install-from-source.sh` is only for development, local validation, or unreleased branches.

## 4. Useful Variables

The production installer supports:

```bash
RELAY_SWITCH_VERSION=vX.Y.Z
RELAY_SWITCH_HTTP_HOST=127.0.0.1
RELAY_SWITCH_HTTP_PORT=3456
RELAY_SWITCH_LOCAL_GATEWAY_PORT=3457
RELAY_SWITCH_INSTALL_ROOT="$HOME/.local/share/relay-switch"
RELAY_SWITCH_DATA_DIR="$HOME/.local/share/relay-switch/data"
```

Example, binding the main entrypoint to all interfaces on port `8080`:

```bash
curl -fsSL https://raw.githubusercontent.com/xiaoyuandev/relay-switch/main/scripts/install.sh | \
  env RELAY_SWITCH_HTTP_HOST=0.0.0.0 RELAY_SWITCH_HTTP_PORT=8080 bash
```

After installation, the helper command can update the same runtime config:

```bash
env RELAY_SWITCH_HTTP_HOST=0.0.0.0 RELAY_SWITCH_HTTP_PORT=8080 relay-switch start
```

This writes the values to the installed `relay-switch.env`, so future starts keep using them. If `relay-switch start` finds the service already running, it restarts it to apply the current config. If another service owns the target port, it prints a clear error and refuses to start.

## 5. Service Management

The installer creates a `systemd --user` service when available:

```bash
systemctl --user status relay-switch
systemctl --user restart relay-switch
journalctl --user -u relay-switch -n 200 -f
```

It also installs a helper command:

```bash
relay-switch start
relay-switch stop
relay-switch restart
relay-switch status
relay-switch logs
relay-switch run
```

## 6. WSL Notes

From Windows, you can usually open:

```text
http://localhost:3456
```

If `systemd --user` is unavailable inside WSL, use:

```bash
relay-switch run
```

## 7. First-Time Setup

After startup:

1. open `http://127.0.0.1:3456`
2. go to `Providers`
3. add an upstream provider
4. point your tool to `http://127.0.0.1:3456/v1`
5. use any non-empty API key such as `dummy`

## 8. Rollback

To roll back to an older stable release, reinstall with a pinned tag:

```bash
curl -fsSL https://raw.githubusercontent.com/xiaoyuandev/relay-switch/main/scripts/install.sh | \
  env RELAY_SWITCH_VERSION=vX.Y.Z bash
```

## 9. Troubleshooting

Health:

```bash
curl http://127.0.0.1:3456/health
```

Release metadata:

```bash
curl http://127.0.0.1:3456/api/release
```

Logs:

```bash
journalctl --user -u relay-switch -n 200 -f
```
