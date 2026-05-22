import type { APIRoute } from "astro";

const summary = `# Relay Switch

Relay Switch is a local AI gateway and desktop control panel for managing OpenAI-compatible providers, model sources, and AI coding tools.

It gives Cursor, Claude Code, Codex, Cherry Studio, OpenAI SDK scripts, and other OpenAI-compatible clients one local endpoint:

http://127.0.0.1:3456/v1

Use Relay Switch when you want to switch providers and models without editing every tool configuration. It is designed for local provider switching, model source management, request logs, desktop configuration, and WSL / Linux Server gateway management.

## Chinese summary

Relay Switch 是一个本地 AI 网关和桌面控制面板，用于统一管理 OpenAI-compatible Provider、模型来源和 AI 编程工具接入。

它为 Cursor、Claude Code、Codex、Cherry Studio、OpenAI SDK 脚本以及其他兼容 OpenAI API 的客户端提供统一的本地接口：

http://127.0.0.1:3456/v1

适用场景包括：在不同第三方中转站或模型服务商之间切换、集中管理模型列表、为多种 AI 工具复用同一个本地网关、通过桌面端管理 WSL / Linux Server 运行时。

## English pages

- https://www.relayswitch.dev/
- https://www.relayswitch.dev/introduction/
- https://www.relayswitch.dev/quick-start/
- https://www.relayswitch.dev/user-guide/
- https://www.relayswitch.dev/tool-integration/
- https://www.relayswitch.dev/deep-link-import/
- https://www.relayswitch.dev/providers/
- https://www.relayswitch.dev/faq/

## Chinese pages

- https://www.relayswitch.dev/zh-cn/
- https://www.relayswitch.dev/zh-cn/introduction/
- https://www.relayswitch.dev/zh-cn/quick-start/
- https://www.relayswitch.dev/zh-cn/user-guide/
- https://www.relayswitch.dev/zh-cn/tool-integration/
- https://www.relayswitch.dev/zh-cn/deep-link-import/
- https://www.relayswitch.dev/zh-cn/providers/
- https://www.relayswitch.dev/zh-cn/faq/

## Repository

- https://github.com/xiaoyuandev/relay-switch

## Download and install

- Desktop releases: https://github.com/xiaoyuandev/relay-switch/releases/latest
- WSL / Linux Server install: curl -fsSL https://raw.githubusercontent.com/xiaoyuandev/relay-switch/main/scripts/install.sh | bash

## Deep Link import

Third-party relay providers can publish Relay Switch import links so users can add a Provider without copying API keys manually. The Deep Link documentation explains the supported URL format and privacy recommendations:

- English: https://www.relayswitch.dev/deep-link-import/
- Chinese: https://www.relayswitch.dev/zh-cn/deep-link-import/
`;

export const GET: APIRoute = () =>
  new Response(summary, {
    headers: {
      "Content-Type": "text/plain; charset=utf-8"
    }
  });
