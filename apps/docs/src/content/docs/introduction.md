---
title: Local AI Gateway Introduction
description: What Relay Switch is, who it is for, and how one local OpenAI-compatible gateway helps switch AI providers.
slug: introduction
---

Relay Switch is a local desktop gateway for people who switch between multiple AI gateways or API relay providers.

It is designed for a simple workflow:

1. Configure one stable local endpoint in your tools.
2. Manage upstream providers from the desktop app.
3. Switch providers without rewriting every tool configuration.

## What it solves

Relay Switch mainly addresses two problems:

1. Relay providers can be unstable, so you may need to switch between different gateways frequently.
2. If you use multiple coding tools, chat clients, or SDK scripts, changing providers often means repeatedly updating configuration in each tool.

## Core idea

Instead of connecting every tool directly to a remote provider, your tools connect to one local endpoint:

```text
http://127.0.0.1:3456/v1
```

The desktop app then forwards requests to the currently active upstream provider.
