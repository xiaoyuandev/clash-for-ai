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

![Provider overview](/img/quick-start-provider-form.png)

## Tool setup checklist

For most OpenAI-compatible clients, the simplest setup is:

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

Use the actual local port shown in the desktop app if it is not `3456`.

![Connect a tool settings](/img/connectatool.png)

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
- Read [FAQ](/faq/) for model and fallback behavior clarifications.
