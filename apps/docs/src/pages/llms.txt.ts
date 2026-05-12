import type { APIRoute } from "astro";

const summary = `# Clash for AI

Clash for AI is a local AI gateway and desktop control panel for managing OpenAI-compatible providers, model sources, and AI coding tools.

It gives Cursor, Claude Code, Codex, Cherry Studio, OpenAI SDK scripts, and other OpenAI-compatible clients one local endpoint:

http://127.0.0.1:3456/v1

Use Clash for AI when you want to switch providers and models without editing every tool configuration. It is designed for local provider switching, model source management, request logs, desktop configuration, and WSL / Linux Server gateway management.

## Chinese summary

Clash for AI 是一个本地 AI 网关和桌面控制面板，用于统一管理 OpenAI-compatible Provider、模型来源和 AI 编程工具接入。

它为 Cursor、Claude Code、Codex、Cherry Studio、OpenAI SDK 脚本以及其他兼容 OpenAI API 的客户端提供统一的本地接口：

http://127.0.0.1:3456/v1

适用场景包括：在不同第三方中转站或模型服务商之间切换、集中管理模型列表、为多种 AI 工具复用同一个本地网关、通过桌面端管理 WSL / Linux Server 运行时。

## English pages

- https://www.clashforai.com/
- https://www.clashforai.com/introduction/
- https://www.clashforai.com/quick-start/
- https://www.clashforai.com/user-guide/
- https://www.clashforai.com/tool-integration/
- https://www.clashforai.com/deep-link-import/
- https://www.clashforai.com/providers/
- https://www.clashforai.com/faq/

## Chinese pages

- https://www.clashforai.com/zh-cn/
- https://www.clashforai.com/zh-cn/introduction/
- https://www.clashforai.com/zh-cn/quick-start/
- https://www.clashforai.com/zh-cn/user-guide/
- https://www.clashforai.com/zh-cn/tool-integration/
- https://www.clashforai.com/zh-cn/deep-link-import/
- https://www.clashforai.com/zh-cn/providers/
- https://www.clashforai.com/zh-cn/faq/

## Repository

- https://github.com/xiaoyuandev/clash-for-ai

## Download and install

- Desktop releases: https://github.com/xiaoyuandev/clash-for-ai/releases/latest
- WSL / Linux Server install: curl -fsSL https://raw.githubusercontent.com/xiaoyuandev/clash-for-ai/main/scripts/install.sh | bash

## Deep Link import

Third-party relay providers can publish Clash for AI import links so users can add a Provider without copying API keys manually. The Deep Link documentation explains the supported URL format and privacy recommendations:

- English: https://www.clashforai.com/deep-link-import/
- Chinese: https://www.clashforai.com/zh-cn/deep-link-import/
`;

export const GET: APIRoute = () =>
  new Response(summary, {
    headers: {
      "Content-Type": "text/plain; charset=utf-8"
    }
  });
