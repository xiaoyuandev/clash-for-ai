---
title: Providers
description: Notes about provider setup, `/v1`, and model list compatibility.
slug: providers
---

## Base URL recommendations

For OpenAI-compatible relay services, the safest Base URL is usually the provider endpoint **with `/v1` included**.

Examples:

```text
https://example.com/v1
https://api.example.com/v1
```

Clash for AI has compatibility handling for model list requests, but using the documented `/v1` Base URL is still the clearest setup for end users.

## Model list behavior

Different relay services expose model lists differently.

Common issues:

1. The provider may not expose a model list endpoint at all.
2. The provider may only respond at `/v1/models`, not `/models`.
3. The returned JSON may differ from standard OpenAI-compatible formats.

Because of this, model list fetching should be treated as a compatibility feature, not as a guaranteed capability of every relay provider.

## Authentication notes

Clash for AI currently focuses on OpenAI-compatible and Anthropic-compatible upstream integrations.

That means:

1. OpenAI-style `Authorization: Bearer ...` flows are supported.
2. Anthropic-style `x-api-key` flows are supported.
3. Gemini native protocol is not currently supported as a first-class upstream protocol.
