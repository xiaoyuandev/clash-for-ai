---
title: 工具接入
description: 编程工具、聊天客户端和 SDK 的常见接入方式。
---

## OpenAI 兼容工具

如果某个工具支持自定义 OpenAI 兼容 Base URL，可以这样配置：

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

这种方式适用于很多 AI 编程助手、桌面聊天客户端以及自定义脚本。

## Claude Code

Claude Code 这类工具通常更偏 Anthropic 风格配置。在 Clash for AI 里，Anthropic 兼容流量一般使用不带 `/v1` 的本地根地址。

如果你的上游只支持 OpenAI 兼容接口，则应优先使用允许自定义 OpenAI endpoint 的工具模式。

## Cursor 和 Cherry Studio

Cursor、Cherry Studio 这类工具通常需要你在应用内的 Provider 配置界面粘贴参数。

推荐填写：

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

然后在工具里选择当前激活 Provider 支持的模型名。

## SDK 脚本

如果你使用 OpenAI SDK，可以把 SDK 指向本地网关，并使用任意非空 key：

```ts
const client = new OpenAI({
  apiKey: "dummy",
  baseURL: "http://127.0.0.1:3456/v1"
});
```

## 一个重要限制

Clash for AI 是本地网关和 Provider 切换工具，不会替用户主控模型选择。

Clash for AI 中配置的模型排序，只会在“请求模型已经命中已选列表，并且上游失败到可重试条件”时，作为 fallback 链条生效。
