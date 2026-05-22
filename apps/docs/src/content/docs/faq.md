---
title: FAQ for Local AI Gateway Setup
description: Common questions about Relay Switch API keys, OpenAI-compatible model lists, fallback behavior, and Gemini compatibility.
slug: faq
---

## Why does my tool use `dummy` as the API key?

Because the real upstream credential is stored inside Relay Switch and injected during forwarding. Many OpenAI-compatible tools only require a non-empty local key value.

## Why can one provider fetch the model list while another cannot?

Relay services do not always expose model metadata in the same way. Some support `/v1/models`, some return custom formats, and some do not expose model discovery at all.

## Does Relay Switch choose the model for the user?

No. The client tool still decides which model to request.

Relay Switch only uses its selected model ordering as a fallback chain under specific conditions:

1. The incoming request already names a model in the selected list.
2. The upstream request fails with a retryable status such as `429` or `5xx`.

## Does Relay Switch support Gemini native API?

Not currently. It can work with Gemini models only when the upstream provider exposes them through an OpenAI-compatible or Anthropic-compatible interface.
