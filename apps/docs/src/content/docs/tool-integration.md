---
title: Tool Integration
description: Common ways to connect coding tools, chat clients, and SDKs.
---

## OpenAI-compatible tools

If a tool supports a custom OpenAI-compatible Base URL, configure:

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

This works well for many coding assistants, desktop chat clients, and custom scripts.

## Claude Code

Claude Code style tools often use Anthropic-style settings. In Clash for AI, the local root URL without `/v1` is used for Anthropic-compatible flows.

If your upstream provider is OpenAI-compatible only, use tools that allow an OpenAI-style custom endpoint instead.

## Cursor and Cherry Studio

Cursor and Cherry Studio usually expect you to paste values into in-app provider fields.

Use:

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

Then select a model supported by your active provider.

## SDK scripts

For OpenAI SDK usage, point the SDK to the local endpoint and use any non-empty API key:

```ts
const client = new OpenAI({
  apiKey: "dummy",
  baseURL: "http://127.0.0.1:3456/v1"
});
```

## Important limitation

Clash for AI is a local gateway and provider switcher. The client tool still chooses the requested model name.

The ordered models configured in Clash for AI only act as a fallback chain when the requested model is already inside that selected list and the upstream request fails with a retryable condition.
