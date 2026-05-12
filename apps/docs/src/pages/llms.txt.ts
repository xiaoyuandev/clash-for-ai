import type { APIRoute } from "astro";

const summary = `# Clash for AI

Clash for AI is a local AI gateway and desktop control panel for managing OpenAI-compatible providers, model sources, and AI coding tools.

It gives Cursor, Claude Code, Codex, Cherry Studio, OpenAI SDK scripts, and other OpenAI-compatible clients one local endpoint:

http://127.0.0.1:3456/v1

Use Clash for AI when you want to switch providers and models without editing every tool configuration.

Important pages:
- https://www.clashforai.com/
- https://www.clashforai.com/introduction/
- https://www.clashforai.com/quick-start/
- https://www.clashforai.com/tool-integration/
- https://www.clashforai.com/providers/
- https://www.clashforai.com/faq/
- https://github.com/xiaoyuandev/clash-for-ai

Download:
- Desktop releases: https://github.com/xiaoyuandev/clash-for-ai/releases/latest
- WSL / Linux Server install: curl -fsSL https://raw.githubusercontent.com/xiaoyuandev/clash-for-ai/main/scripts/install.sh | bash
`;

export const GET: APIRoute = () =>
  new Response(summary, {
    headers: {
      "Content-Type": "text/plain; charset=utf-8"
    }
  });
