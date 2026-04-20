# Clash for AI

Clash for AI is a local desktop gateway for people who use multiple AI API providers across different coding tools.

It gives you:

1. One stable local endpoint for your tools
2. A desktop control plane for switching providers
3. Local request logs and health checks for debugging provider issues

## What It Does

Clash for AI runs a local API gateway on your machine.

Your editor, chat client, CLI tool, or custom script connects to the local endpoint:

```text
http://127.0.0.1:3456/v1
```

Then Clash for AI forwards requests to the currently active provider you configured in the desktop app.

This means:

1. You do not need to reconfigure every tool when switching providers
2. Provider credentials stay managed locally in one place
3. You can inspect health status and request logs from the desktop UI

## Current Features

1. Local desktop app built with Electron
2. Local Go gateway for API forwarding
3. Provider management
4. Active provider switching
5. Health checks
6. Request logging
7. Automatic local port fallback when the default port is occupied
8. In-app update flow for packaged builds

## How To Use

See the end-user setup guide:

- [User Guide](./docs/user-guide.md)

In most supported tools, you configure:

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

If the local app selects another port at runtime, use the actual `connected api base` shown in the desktop UI.

## Local Development

Requirements:

1. Node.js
2. pnpm
3. Go toolchain, if you want the core service to build locally

Install dependencies:

```bash
pnpm install
```

Run the desktop app in development mode:

```bash
pnpm dev
```

Build the desktop app:

```bash
pnpm build
```

## Project Structure

```text
apps/desktop   Electron desktop application
core/          Go local gateway and provider management backend
docs/          Public user-facing documentation
```

## License

This project is licensed under the GNU Affero General Public License v3.0 only.

See:

- [LICENSE](./LICENSE)

## Brand Notice

The source code in this repository is licensed under AGPL-3.0-only.

However:

1. The project name `Clash for AI`
2. Logos
3. Icons
4. Other brand assets

are not granted for unrestricted use by this source license unless explicitly stated otherwise.

## Status

This project is under active development. Interfaces, packaging flow, and update behavior may still change.
