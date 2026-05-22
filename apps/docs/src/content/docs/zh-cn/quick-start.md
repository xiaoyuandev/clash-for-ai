---
title: 快速开始：安装并接入 AI 工具
description: 安装 Relay Switch，添加 OpenAI 兼容 Provider，并把 Cursor、Claude Code、Codex 或其他工具接入本地网关。
slug: zh-cn/quick-start
---

## 使用前准备

你需要：

1. 一个 Relay Switch 桌面应用安装包，或者本地开发环境
2. 至少一个可用的上游 Provider / 中转服务
3. 一个支持自定义 API endpoint 的工具

## 第一步：启动 Relay Switch

启动桌面应用后，Relay Switch 会在本地启动一个 API Gateway。

默认地址通常是：

```text
http://127.0.0.1:3456/v1
```

如果默认端口被占用，应用会改用其他本地端口。请以应用界面显示的 `connected api base` 为准。

<img src="../../img/quick-start-launch.png" alt="Relay Switch 启动后的本地网关首页" />

## 第二步：添加 Provider

进入 `Providers` 页面，填写：

1. `Name`
2. `Base URL`
3. `API Key`

对于 OpenAI 兼容中转服务，通常推荐填写带 `/v1` 的 Base URL。

<img src="../../img/quick-start-provider-form.png" alt="Relay Switch OpenAI 兼容 Provider 配置表单" />

## 第三步：激活 Provider

如果你配置了多个 Provider，请点击目标 Provider 的 `Enable`。

只有当前激活的 Provider 会接收转发流量。

## 第四步：在工具中接入本地网关

大多数 OpenAI 兼容工具都可以这样配置：

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

这里的本地 API Key 通常可以是任意非空字符串，因为真正的上游密钥由 Relay Switch 在本地转发时处理。

## 第五步：发送测试请求

工具接入完成后：

1. 发起一次正常请求
2. 打开 Relay Switch 的 `Logs` 页面
3. 确认请求已经记录下来

如果请求失败，先检查 provider healthcheck，再看请求日志中的详细报错。

<img src="../../img/quick-start-request-log.png" alt="Relay Switch 本地网关请求日志" />
