---
title: Providers
description: 关于 Base URL、/v1 和模型列表兼容性的说明。
slug: zh-cn/providers
---

## Base URL 的推荐写法

对于 OpenAI 兼容中转服务，最稳妥的写法通常是 **带 `/v1` 的 Base URL**。

例如：

```text
https://example.com/v1
https://api.example.com/v1
```

虽然 Clash for AI 已经对模型列表请求补了一层兼容处理，但对终端用户来说，直接使用文档中要求的 `/v1` Base URL 依然最清晰。

## 模型列表获取行为

不同中转服务提供模型列表的方式并不一致。

常见情况包括：

1. 服务商根本不提供模型列表接口
2. 只支持 `/v1/models`，不支持 `/models`
3. 返回的 JSON 结构不是标准 OpenAI 风格

因此，模型列表获取应该被理解为兼容性能力，而不是每个中转服务都一定支持的基础能力。

## 鉴权说明

Clash for AI 当前主要面向：

1. OpenAI 兼容上游
2. Anthropic 兼容上游

也就是说：

1. 支持 OpenAI 风格 `Authorization: Bearer ...`
2. 支持 Anthropic 风格 `x-api-key`
3. 当前还不支持 Gemini 原生协议作为一等上游协议
