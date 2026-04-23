---
title: 工具接入
description: 编程工具、聊天客户端和 SDK 的常见接入方式。
slug: zh-cn/tool-integration
---

## OpenAI 兼容工具

如果某个工具支持自定义 OpenAI 兼容 Base URL，可以这样配置：

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

这种方式适用于很多 AI 编程助手、桌面聊天客户端以及自定义脚本。

## CLI 工具如何通过环境变量接入

对于 CLI 工具，最稳妥的方式通常是在当前 shell 中先设置环境变量，再从同一个终端启动工具。

### Codex CLI

使用标准 OpenAI 兼容环境变量：

```bash
export OPENAI_BASE_URL="http://127.0.0.1:3456/v1"
export OPENAI_API_KEY="dummy"
```

然后在同一个终端会话里启动 Codex CLI。

这样 Codex CLI 会始终连接本地统一入口，而真正切换上游 Provider 的动作由 Clash for AI 负责。

### 通用 OpenAI 兼容 CLI

很多支持自定义 OpenAI endpoint 的 CLI 工具，也可以直接使用同样的写法：

```bash
export OPENAI_BASE_URL="http://127.0.0.1:3456/v1"
export OPENAI_API_KEY="dummy"
```

如果这个工具本身会读取标准 OpenAI 环境变量，一般就可以直接复用 Clash for AI。

## Claude Code

Claude Code 这类工具通常更偏 Anthropic 风格配置。在 Clash for AI 里，Anthropic 兼容流量一般使用不带 `/v1` 的本地根地址。

如果你的上游只支持 OpenAI 兼容接口，则应优先使用允许自定义 OpenAI endpoint 的工具模式。

### Claude Code 如何通过环境变量接入

使用 Anthropic 风格环境变量：

```bash
export ANTHROPIC_BASE_URL="http://127.0.0.1:3456"
export ANTHROPIC_AUTH_TOKEN="dummy"
```

然后在同一个终端会话里启动 Claude Code。

对于 Anthropic 风格流量，本地地址使用不带 `/v1` 的根地址。

## IDE / 插件如何填写配置参数

对于 IDE、编辑器插件和桌面客户端，通常可以按下面这套思路配置：

1. 打开工具里的 Provider 设置
2. 如果支持，选择 OpenAI-compatible 自定义 Provider 模式
3. 填写本地 Base URL
4. 填写任意非空 API Key
5. 再选择当前激活 Provider 支持的模型

### Cursor

Cursor 通常是应用内手动填写 Provider 字段。

推荐填写：

```text
Provider Type: OpenAI Compatible
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

然后在工具里选择当前激活 Provider 支持的模型名。

### Cherry Studio

Cherry Studio 同样是应用内手动填写字段。

推荐填写：

```text
Provider Protocol: OpenAI Compatible
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

如果 Cherry Studio 能从当前中转服务自动拉到模型列表，就直接从列表里选；如果拉不到，则可能需要按上游服务支持情况手动输入模型名。

### 通用 IDE / 插件字段检查清单

如果你配置的是其他 IDE 或插件，按这个清单找字段：

1. 找 `Base URL`、`API Base`、`Endpoint` 或 `OpenAI Base URL`
2. 填 Clash for AI 的本地地址，并带上 `/v1`
3. API Key 填 `dummy` 或任意非空值
4. 确认你选择的模型确实是当前激活 Provider 支持的模型

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

![IDE / 插件接入字段](/img/connectatool.png)
