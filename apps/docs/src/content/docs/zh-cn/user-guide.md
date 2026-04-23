---
title: 使用手册
description: 一份更完整的 Clash for AI 实际使用说明。
slug: zh-cn/user-guide
---

## 这份手册适合什么时候看

这页是 Clash for AI 的正式使用说明。

当你希望系统地了解下面这些问题时，优先看这一页：

1. 本地网关到底是怎么工作的
2. Provider 应该怎么添加和切换
3. 工具到底应该怎么接入
4. Models 页面里的排序到底什么时候生效
5. 请求失败时应该按什么顺序排查

## 整体流量路径

Clash for AI 位于客户端工具和上游中转 Provider 之间。

一次正常请求的路径通常是：

1. 你的工具先把请求发给本地 Clash for AI 地址
2. Clash for AI 读取当前激活的 Provider
3. Clash for AI 注入上游密钥
4. Clash for AI 把请求转发给上游 Provider
5. 上游响应再回到你的工具
6. 请求过程被记录到本地日志里

这也是为什么你的工具可以始终只配置一个本地入口，而上游切换由桌面应用统一控制。

## Provider 配置检查清单

添加 Provider 时，重点确认这三个字段：

1. `Name`
2. `Base URL`
3. `API Key`

对于 OpenAI 兼容中转服务，最稳妥的 Base URL 通常是带 `/v1` 的地址。

例如：

```text
https://example.com/v1
https://api.example.com/v1
```

如果服务商文档只给了根域名，而模型列表获取失败，建议再尝试一次带 `/v1` 的写法。

![Provider 概览](/img/quick-start-provider-form.png)

## 工具接入检查清单

对于大多数 OpenAI 兼容客户端，最简接入方式是：

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

如果应用里显示的实际端口不是 `3456`，请以桌面应用中的 `connected api base` 为准。

![Connect a Tool 界面](/img/connectatool.png)

## Models 页面到底什么时候生效

`Models` 页面**不会**替用户主控选择模型。

真正决定请求哪个模型的，仍然是客户端工具本身。

Clash for AI 里的已选模型排序，只会在这些条件下生效：

1. 当前请求是 JSON 格式的 `POST`
2. 请求体里本身已经有 `model` 字段
3. 这个模型已经命中 Clash for AI 的已选模型列表
4. 上游请求失败，且错误属于可重试条件，比如 `429`、`5xx` 或网络错误

如果请求模型不在已选列表中，Clash for AI 不会自动切换到其他备用模型。

## 关于模型列表获取的说明

模型列表获取是一个“兼容性增强能力”，不是所有中转服务都保证支持的能力。

常见失败原因包括：

1. 服务商根本不暴露模型发现接口
2. 服务商只支持 `/v1/models`
3. 服务商返回的 JSON 不是标准 OpenAI 风格
4. 服务商使用的是当前项目还不支持的原生协议

所以如果某个 Provider 请求能正常转发，但模型列表获取失败，这更像是“模型发现兼容性问题”，不一定代表这个 Provider 本身不可用。

## 推荐排查顺序

当请求异常时，建议按这个顺序查：

1. 确认本地 core 是否正常运行
2. 查看桌面应用里的 `connected api base`
3. 先跑 provider healthcheck
4. 重点确认 Base URL，尤其是否需要 `/v1`
5. 打开日志页，看上游返回的错误正文
6. 用一个已知兼容的 OpenAI 客户端复现同样的 Provider 配置

## 当前协议范围

Clash for AI 当前主要面向：

1. OpenAI 兼容上游
2. Anthropic 兼容上游

Gemini 原生协议当前还不是一等上游协议。

## 继续阅读

- 查看 [Providers](/zh-cn/providers/) 了解兼容性说明
- 查看 [工具接入](/zh-cn/tool-integration/) 了解客户端接入方式
- 查看 [FAQ](/zh-cn/faq/) 了解模型和 fallback 相关问题
