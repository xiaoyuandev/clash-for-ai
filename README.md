# Clash for AI

Clash for AI is a local desktop gateway for people who switch between multiple AI gateways or API relay providers.

[中文 README](./README.zh-CN.md)

It gives you:

1. One stable local endpoint for your tools
2. A desktop control plane for switching providers
3. Local request logs and health checks for debugging provider issues

## Screenshot

<p align="center">
  <img src="./docs/images/readme/quick-start-provider-form.png" style="width: 100%; height: auto;">
</p>

<p align="center">
  <img src="./docs/images/readme/connectatool.png" style="width: 100%; height: auto;">
</p>


## What Problem It Solves

Clash for AI is designed for people who depend on multiple AI gateways in daily use.

It mainly addresses two problems:

1. API relay providers can be unstable, so you may need to switch between different gateways frequently
2. If you use multiple coding tools, chat clients, or SDK scripts, changing providers often means repeatedly updating configuration in each tool

Clash for AI puts one local gateway in front of those tools.

You configure a single local endpoint once, then switch the upstream relay provider from the desktop app.

## How It Differs From cc-switch

The difference is mainly in product focus and integration model.

| Aspect | cc-switch | Clash for AI |
|---|---|---|
| Public positioning | An all-in-one desktop manager for Claude Code, Codex, Gemini CLI, OpenCode, and OpenClaw | A local desktop gateway for managing multiple AI gateways / relay providers |
| Primary target | Tool-side provider management across specific coding CLIs | Upstream gateway switching behind one shared local endpoint |
| How switching works | Its public docs focus on managing tool configs and provider presets for supported apps | Tools point to one localhost endpoint, and the desktop app switches the upstream gateway |
| Effect on tool configuration | Built around per-tool management flows for its supported apps | Avoids repeated per-tool rewrites after the initial Base URL setup |
| Supported integration style | Publicly documented around five coding CLIs, plus related MCP / Skills / prompt workflows | General local endpoint that can be reused by coding tools, chat clients such as Cherry Studio, and SDK or script-based integrations |
| Current built-in capabilities | Broad app management features such as MCP, Skills, prompts, sessions, proxy/failover, and cloud sync | Focused local gateway features such as provider switching, health checks, request logs, and stable local access |

In short: cc-switch is centered on managing specific AI coding tools, while Clash for AI is centered on managing upstream relay gateways behind a stable local API entry point.

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

### Step 1: Add a relay provider in Clash for AI

Open the `Providers` page in the desktop app and fill in:

1. `Name`
2. `Base URL`
3. `API Key`

For OpenAI-compatible relay providers, the Base URL usually ends with `/v1`.

<p align="center">
  <img src="./docs/images/readme/quick-start-provider-form.png" style="width: 100%; height: auto;">
</p>

### Step 2: Configure your tool

In most supported tools, you configure:

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

If the local app selects another port at runtime, use the actual `connected api base` shown in the desktop UI.

## Quick Connection

If you do not want to read the full guide yet, use one of these quick setup patterns.

### CLI Tools

For OpenAI-compatible CLI tools such as Codex CLI, set environment variables in the current shell before launching the tool:

```bash
export OPENAI_BASE_URL="http://127.0.0.1:3456/v1"
export OPENAI_API_KEY="dummy"
```

Then start the CLI from the same terminal session.

For Claude Code style tools, use Anthropic-style variables and the local root URL without `/v1`:

```bash
export ANTHROPIC_BASE_URL="http://127.0.0.1:3456"
export ANTHROPIC_AUTH_TOKEN="dummy"
```

### IDEs And Plugins

For IDEs, editor plugins, and desktop chat clients, open the provider settings and fill in:

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

Inside Clash for AI, you can also open `Settings -> Connect a Tool` to find the recommended connection values for supported tools.

<p align="center">
  <img src="./docs/images/readme/settings.png" style="width: 100%; height: auto;">
</p>

<p align="center">
  <img src="./docs/images/readme/connectatool.png" style="width: 100%; height: auto;">
</p>

For tools like Cursor or Cherry Studio, if there is a provider type or protocol field, choose an OpenAI-compatible custom provider mode first, then paste the values above.

In Cursor specifically, open its custom provider settings, choose an OpenAI-compatible provider mode, then fill in the local Base URL and `dummy` API key.

<p align="center">
  <img src="./docs/images/readme/corsor-config.png" style="width: 100%; height: auto;">
</p>

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
