---
title: FAQ
description: 关于配置、模型列表和 fallback 的常见问题。
slug: zh-cn/faq
---

## 为什么工具里可以把 API Key 写成 `dummy`？

因为真正的上游密钥保存在 Clash for AI 里，并在转发时注入。很多 OpenAI 兼容工具只要求本地填写一个非空值即可。

## 为什么有的 Provider 能拉到模型列表，有的不能？

因为不同中转服务暴露模型信息的方式不一样。有些支持 `/v1/models`，有些返回自定义格式，有些则根本不提供模型发现接口。

## Clash for AI 会替用户决定模型吗？

不会。真正决定请求哪个模型的，仍然是客户端工具本身。

Clash for AI 的已选模型排序只会在这些条件下生效：

1. 进入网关的请求已经带了一个命中已选列表的模型名
2. 上游请求失败，并且错误属于可重试条件，比如 `429` 或 `5xx`

## 当前项目支持 Gemini 原生 API 吗？

当前不支持。只有当上游把 Gemini 模型封装成 OpenAI 兼容或 Anthropic 兼容接口时，Clash for AI 才能接入。
