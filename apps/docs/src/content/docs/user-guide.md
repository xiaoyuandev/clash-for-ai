---
title: User Guide
description: A fuller guide to the real-world setup flow for Clash for AI.
slug: user-guide
---

## What this guide is for

This page is the practical usage guide for Clash for AI.

Use it when you want one place to understand:

1. how the local gateway works,
2. how to add and activate providers,
3. how to connect client tools,
4. what the model list means,
5. and what to check when something fails.

## How the traffic flow works

Clash for AI sits between your client tool and the upstream relay provider.

The normal request path is:

1. your tool sends a request to the local Clash for AI endpoint,
2. Clash for AI loads the active provider,
3. Clash for AI injects the upstream credential,
4. Clash for AI forwards the request to the provider,
5. the response is sent back to your tool,
6. the request is recorded in the local log view.

This is why your tools can keep one stable local Base URL while the desktop app switches the upstream provider.

## Desktop vs Web

Clash for AI now has two kinds of management entry:

1. the `Electron` desktop app,
2. the `Web / PWA` supplementary entry.

They are not equal in product role.

The rule is:

1. `Electron` remains the primary local entry,
2. `Web / PWA` mainly serves `WSL` and `Linux server` users,
3. `Web / PWA` does not replace the desktop app.

The desktop app still owns:

1. local core lifecycle,
2. tray and window integration,
3. local desktop integration,
4. desktop update flow.

The Web / PWA entry mainly provides:

1. browser-based management for a running core,
2. a usable UI for headless environments,
3. a better path for WSL and Linux server setups.

## How to use it with WSL

If you mainly run `Codex CLI`, `Claude Code`, or other CLI tools inside `WSL`, the recommended path is:

1. start `clash-for-ai-core` inside WSL,
2. open the Web UI exposed by that WSL instance in your browser,
3. manage `Providers`, `Models`, `Logs`, and `Tools` from that Web page.

The important effect is:

1. tool configuration files are written into the WSL Linux home directory,
2. you do not need the Windows desktop app to cross-configure WSL files,
3. one-click configuration matches the real runtime environment.

## How to use it on a Linux server

If Clash for AI runs on a Linux server, cloud VM, home server, or NAS, the recommended path is:

1. start `clash-for-ai-core` on that machine,
2. open the exposed Web UI from your browser,
3. manage `Providers`, `Models`, `Logs`, and `Tools` there.

This mode is intended for:

1. machines without a desktop environment,
2. long-running development servers,
3. remote browser-based management.

## PWA positioning

If you install the Web app as a PWA in Chrome or another compatible browser, keep its role clear:

1. PWA is only an installation form of the Web UI,
2. PWA gives you a more app-like browser entry,
3. PWA does not replace the Electron desktop app.

PWA can provide:

1. a standalone window,
2. a shortcut entry,
3. static asset caching.

PWA does not provide:

1. local Go core startup,
2. tray, auto-launch, or native desktop integration,
3. desktop update flow.

## One-click tool setup in supplementary mode

When you open the `Tools` page from the WSL or Linux server Web UI, one-click configuration applies to the environment where that core instance is actually running.

That means:

1. a Web page served from WSL writes into WSL paths like `~/.codex` and `~/.claude`,
2. a Web page served from a Linux server writes into that server's own user directory.

This is exactly why the Web / PWA entry is a better supplementary path for WSL and Linux server users.

## Provider setup checklist

When you add a provider, check these fields carefully:

1. `Name`
2. `Base URL`
3. `API Key`

For OpenAI-compatible relay providers, the most reliable Base URL is usually the provider endpoint with `/v1`.

Examples:

```text
https://example.com/v1
https://api.example.com/v1
```

If the provider documentation shows only a root domain, test both the documented value and the `/v1` form if model discovery fails.

<img src="../img/quick-start-provider-form.png" alt="Provider overview" />

## Tool setup checklist

For most OpenAI-compatible clients, the simplest setup is:

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

Use the actual local port shown in the desktop app if it is not `3456`.

<img src="../img/connectatool.png" alt="Connect a tool settings" />

## What the Models page actually does

The `Models` page does **not** choose the model on behalf of the client tool.

The client tool still decides the requested model name.

The ordered selected models in Clash for AI are used only as a fallback chain when:

1. the incoming request is a JSON `POST`,
2. the request already includes a model field,
3. that model is already in the selected model list,
4. and the upstream request fails with a retryable condition such as `429`, `5xx`, or a network error.

If the requested model is not in the selected list, Clash for AI will not switch to a different fallback model automatically.

## Model list compatibility notes

Model discovery is a convenience feature, not a guaranteed feature of every relay provider.

Common reasons a model list may fail:

1. the provider does not expose model discovery,
2. the provider only supports `/v1/models`,
3. the provider returns a non-standard response format,
4. the provider uses a protocol Clash for AI does not support natively.

If the provider can serve requests normally but the model list fails, treat that as a compatibility issue with discovery, not necessarily as a provider failure.

## Troubleshooting order

When something does not work, use this order:

1. Check whether the local core is running.
2. Confirm the `connected api base` in the desktop app.
3. Run the provider health check.
4. Verify the Base URL, especially whether `/v1` is required.
5. Open the request logs and read the upstream error body.
6. Re-test the same provider in a known OpenAI-compatible client.

## Current protocol scope

Clash for AI currently focuses on:

1. OpenAI-compatible upstreams
2. Anthropic-compatible upstreams

Gemini native protocol is not currently supported as a first-class upstream protocol.

## Recommended next docs

- Read [Providers](/providers/) for compatibility notes.
- Read [Tool Integration](/tool-integration/) for client setup patterns.
- Read [Deep Link Import](/deep-link-import/) if you want websites to open the desktop app and prefill import data.
- Read [FAQ](/faq/) for model and fallback behavior clarifications.
