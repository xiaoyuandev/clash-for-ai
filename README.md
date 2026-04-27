# Clash for AI

Clash for AI is a local desktop gateway for people who switch between multiple AI gateways or API relay providers.

Its role is:

1. Provide one local API entry point
2. Switch different upstream gateways behind that local entry point
3. Manage providers, health checks, and request logs from a desktop UI

It is not primarily a manager for one specific AI tool. It is better understood as a local gateway plus a multi-provider control plane.

[中文 README](./README.zh-CN.md)

It currently gives you:

1. One stable local endpoint for your tools
2. A desktop control plane for switching providers
3. Local request logs and health checks for debugging provider issues

## Core Idea

The model is simple:

1. Your tools all point to one local gateway
2. Upstream gateway switching happens in the desktop app
3. Real provider credentials, health status, and request logs stay managed locally

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

## What It Does

Clash for AI runs a local API gateway on your machine.

Most editors, chat clients, CLI tools, or custom scripts connect to the local endpoint:

```text
http://127.0.0.1:3456/v1
```

Then Clash for AI forwards requests to the currently active provider you configured in the desktop app.

In the current version, the local access path is most mature around an OpenAI-compatible local entry point. Anthropic-compatible upstream handling and some Claude-style tool integrations are present, but that part of the stack is still being refined.

This means:

1. You do not need to reconfigure every tool when switching providers
2. Provider credentials stay managed locally in one place
3. You can inspect health status and request logs from the desktop UI



## Quick Start

If you do not want to read the full guide yet, use one of these quick setup patterns.

### 1. Add a provider in Clash for AI

Open the `Providers` page in the desktop app and fill in:

1. `Name`
2. `Base URL`
3. `API Key`

For OpenAI-compatible relay providers, the Base URL usually ends with `/v1`.

For other compatible APIs, whether `/v1` should be included depends on the upstream implementation. At the moment, OpenAI-compatible upstreams are the clearest and most mature path in Clash for AI.

<p align="center">
  <img src="./docs/images/readme/quick-start-provider-form.png" style="width: 100%; height: auto;">
</p>

### 2. Point your tool to the local endpoint

In most supported tools, configure:

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

If the local app selects another port at runtime, use the actual `connected api base` shown in the desktop UI.

### 3. Use the `Tools` page when you need tool-specific setup

The `Tools` page provides:

1. Copy-ready connection values
2. One-click setup for Codex CLI and Claude Code
3. Setup guidance for tools such as Cursor, Cherry Studio, and SDK scripts

### CLI Tools

For OpenAI-compatible CLI tools such as Codex CLI, set environment variables in the current shell before launching the tool:

```bash
export OPENAI_BASE_URL="http://127.0.0.1:3456/v1"
export OPENAI_API_KEY="dummy"
```

Then start the CLI from the same terminal session.

For Claude Code style tools, Clash for AI currently provides an Anthropic-style environment variable setup flow:

```bash
export ANTHROPIC_BASE_URL="http://127.0.0.1:3456"
export ANTHROPIC_AUTH_TOKEN="dummy"
```

Inside Clash for AI, you can also open the `Tools` page and use the built-in one-click setup flow for supported CLIs.

One clarification: the most stable local access path in the current release is still the OpenAI-compatible one. Anthropic-style local access and upstream compatibility are still being improved. If your tool also supports a custom OpenAI-compatible endpoint, prefer `http://127.0.0.1:3456/v1`.

### IDEs And Plugins

For IDEs, editor plugins, and desktop chat clients, open the provider settings and fill in:

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

Inside Clash for AI, open the `Tools` page to find the recommended connection values for supported tools.

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

### SDK Scripts And Local Apps

If you want to interact with the currently active model provider from your own scripts, point your SDK or HTTP client to the local Clash for AI gateway instead of the upstream relay directly.

Example with the OpenAI SDK:

```ts
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: "dummy",
  baseURL: "http://127.0.0.1:3456/v1"
});

const response = await client.responses.create({
  model: "gpt-4.1",
  input: "Say hello from Clash for AI."
});

console.log(response.output_text);
```

You can do the same thing with plain HTTP requests:

```bash
curl http://127.0.0.1:3456/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dummy" \
  -d '{
    "model": "gpt-4.1",
    "messages": [
      { "role": "user", "content": "Say hello from Clash for AI." }
    ]
  }'
```

The actual model that responds still depends on the model name your script sends and on which provider is currently active in the desktop app.

## Documentation

If you want fuller step-by-step guidance, tool-specific examples, and troubleshooting notes, continue with:

- [User Guide](./docs/user-guide.md)
- [中文 README](./README.zh-CN.md)

## How To Read Protocol Support Today

In practice, many upstream gateways expose both OpenAI-compatible and Anthropic-compatible APIs.

Clash for AI is designed around those two compatibility families, but the current implementation is not equally mature in both directions:

1. OpenAI-compatible local access is the clearest and most stable primary path
2. Anthropic-compatible upstream auth handling and some tool integrations are already covered
3. Full Anthropic-style local protocol coverage is still being improved

Because of that, for tools that let you choose a custom OpenAI-compatible endpoint, that path is currently the safest default.

## About Model Lists

Provider model list fetching exists, but it should be understood as a compatibility feature rather than a guaranteed capability of every upstream.

Common reasons include:

1. Different gateways expose model list endpoints differently
2. Some upstreams do not expose a standard model list endpoint at all
3. Returned JSON payloads may vary

So a provider can still be usable for request forwarding even if its model list is incomplete or unavailable.

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
