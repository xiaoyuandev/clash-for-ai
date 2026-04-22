# Clash for AI

Clash for AI is a local desktop gateway for people who use multiple AI API providers across different coding tools.

[中文 README](./README.zh-CN.md)

It gives you:

1. One stable local endpoint for your tools
2. A desktop control plane for switching providers
3. Local request logs and health checks for debugging provider issues

## What Problem It Solves

If you use more than one AI provider, switching between them usually means touching every tool again:

1. Updating environment variables or provider config files
2. Re-entering endpoints and API keys in editors, CLIs, and desktop apps
3. Repeating the same setup when you want to test another provider
4. Losing time debugging whether a failure came from auth, quota, network, or the upstream service

Clash for AI reduces that operational cost by putting a local gateway in front of your tools.

You configure your tools once against a local endpoint, then switch the upstream provider from the desktop app without rewriting each tool's configuration.

## How It Differs From cc-switch

`cc-switch` is primarily a configuration rewriting tool. It switches providers by changing environment variables or tool configuration files.

Clash for AI takes a different approach:

1. Your tools point to one stable localhost endpoint instead of being rewritten for each switch
2. Provider switching happens from a desktop UI instead of repeated config edits
3. Logs and health checks are built in, so you can inspect failures locally
4. Multiple tools can switch together because they all depend on the same local gateway

In short: `cc-switch` changes tool configuration; Clash for AI keeps tool configuration stable and changes the upstream behind a local gateway.

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
- [中文 README](./README.zh-CN.md)

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

Build packaged desktop releases:

```bash
pnpm --filter desktop build:mac
pnpm --filter desktop build:win
pnpm --filter desktop build:linux
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
