# 在 Codex 里使用第三方大模型

本文档说明如何通过 Relay Switch 在 Codex CLI 里使用第三方大模型，并让这些模型显示在 Codex 的 `/model` 菜单里。

适用场景：

1. 你希望在 Codex 里使用 DeepSeek、Qwen、GLM、MiniMax 等第三方模型。
2. 你希望 Codex 请求统一走 Relay Switch 本地网关。
3. 你希望自定义模型可以出现在 Codex 的 `/model` 菜单中，而不是每次手动输入模型名。

## 1. 基本概念

Relay Switch 是一个本地 AI API 网关。

Codex 连接 Relay Switch 后，请求会先进入本地地址：

```text
http://127.0.0.1:3456/v1
```

然后 Relay Switch 再根据当前启用的 Provider 或 Local Gateway，把请求转发到实际上游。

所以在 Codex 中使用第三方模型分为两件事：

1. 让请求能正常转发到第三方模型。
2. 让第三方模型显示在 Codex 的 `/model` 菜单里。

第一件事由 Relay Switch 的 Provider / Local Gateway 负责。

第二件事由 Relay Switch 生成 Codex profile 和 model catalog 负责。

## 2. 选择接入方式

你可以按自己的模型来源选择一种方式。

### 方式一：使用中转 API Provider

如果你已经有一个 OpenAI-compatible 中转 API，例如基于 new-api、one-api、sub2api 或其他 relay 服务，可以直接在 Provider 页面添加它。

需要填写：

```text
Name
Base URL
API Key
```

然后在 Provider 页面启用这个 Provider。

这种方式适合：

1. 你的上游已经聚合了多个模型。
2. 上游本身已经兼容 OpenAI API。
3. 你只是希望 Codex 统一走 Relay Switch。

### 方式二：使用 Local Gateway 接入原生模型

如果你想直接接入 DeepSeek、MiniMax、Qwen、GLM 等原生模型服务，建议使用 Models 页面。

操作路径：

1. 打开 Models 页面。
2. 新增模型来源。
3. 选择 Provider Type，例如 `openai-compatible`。
4. 填写 Base URL、API Key、模型 ID。
5. 启用该模型来源。
6. 同步到 Local Gateway。
7. 回到 Provider 页面，启用 Local Gateway。

这种方式适合：

1. 你想直接接入模型厂商官方 API。
2. 不同模型来源的协议或能力不完全一致。
3. 你希望 Relay Switch 在本地统一管理这些模型来源。

## 3. 配置 Codex 模型列表

接下来需要告诉 Relay Switch：

```text
哪些模型应该显示在 Codex 的 /model 菜单里？
```

打开 Provider 页面，选择当前要用于 Codex 的 Provider。

在该 Provider 的 Codex 模型列表中添加模型名。

例如：

```text
deepseek-chat
deepseek-reasoner
qwen3-coder
MiniMax-M2.7
```

这里填写的模型名应该和上游实际支持的模型 ID 一致。

如果你启用的是 Local Gateway，也要确保这些模型已经在 Models 页面对应 source 中配置并同步。

## 4. 同步 Codex 配置

打开 Tools 页面，找到 Codex CLI。

执行 Codex 配置或同步操作。

Relay Switch 会生成以下内容：

```text
~/.codex/relay-switch.config.toml
~/.codex/relay-switch-model-catalog.json
```

其中：

1. `relay-switch.config.toml` 是 Relay Switch 专用的 Codex profile。
2. `relay-switch-model-catalog.json` 是 Codex 的模型目录文件。

这份模型目录会让 Codex 的 `/model` 菜单显示你在 Relay Switch 中配置的第三方模型。

## 5. 启动 Codex

配置完成后，使用 Relay Switch profile 启动 Codex：

```bash
codex -p relay-switch
```

如果你在 Tools 页面安装了快捷命令，也可以使用：

```bash
codex-rs
```

进入 Codex 后，输入：

```text
/model
```

你应该可以在模型菜单中看到 Relay Switch 注入的第三方模型。

选择模型后，Codex 的请求会进入 Relay Switch 本地网关，再转发到当前启用的 Provider 或 Local Gateway。

## 6. DeepSeek 使用建议

如果你要使用 DeepSeek，推荐走 Local Gateway。

原因是 Codex 当前主要使用 OpenAI Responses API 风格请求，而 DeepSeek 官方 OpenAI-compatible 接口主要是 Chat Completions 风格。

Local Gateway 会负责做必要的协议适配。

推荐配置：

```text
Models 页面:
  Provider Type: openai-compatible
  Base URL: https://api.deepseek.com
  Model IDs: deepseek-chat, deepseek-reasoner

Provider 页面:
  启用 Local Gateway
  Codex 模型列表添加 deepseek-chat / deepseek-reasoner
```

配置后同步 Codex，再使用：

```bash
codex -p relay-switch
```

然后通过 `/model` 选择 DeepSeek 模型。

## 7. 常见问题

### 模型没有出现在 /model 菜单里

先检查：

1. Provider 页面是否已经添加 Codex 模型列表。
2. 是否执行过 Codex 配置或同步。
3. 是否用 `codex -p relay-switch` 或 `codex-rs` 启动。
4. 修改模型列表后是否重启了 Codex。

Codex 通常不会在运行中自动刷新模型目录。新增或删除模型后，建议重启 Codex。

### 模型出现在菜单里，但请求失败

先检查：

1. 当前启用的 Provider 是否支持该模型。
2. API Key 是否正确。
3. Base URL 是否正确。
4. 如果走 Local Gateway，Models 页面是否已经同步。
5. Logs 页面里上游返回的错误是什么。

模型出现在 `/model` 菜单里，只代表 Codex 能选择它，不代表上游一定能成功响应。

### 使用 DeepSeek 时没有响应或返回协议错误

优先确认：

1. Relay Switch 是否已集成支持 DeepSeek 的 ai-mini-gateway runtime 版本。
2. Local Gateway 是否运行正常。
3. DeepSeek source 是否已启用并同步。
4. Provider 页面当前是否启用了 Local Gateway。
5. Codex 模型列表里的模型名是否和 DeepSeek source 暴露的模型 ID 一致。

### 不想影响原来的 Codex 配置怎么办

Relay Switch 使用独立 profile。

默认 Codex 配置不会被直接替换。

你可以继续使用：

```bash
codex
```

也可以在需要 Relay Switch 第三方模型环境时使用：

```bash
codex -p relay-switch
```

或者：

```bash
codex-rs
```

## 8. 推荐使用流程

最常见的完整流程是：

1. 在 Provider 页面添加中转 API，或在 Models 页面添加原生模型来源。
2. 启用对应 Provider，或启用 Local Gateway。
3. 在 Provider 页面添加 Codex 模型列表。
4. 在 Tools 页面同步 Codex 配置。
5. 使用 `codex -p relay-switch` 或 `codex-rs` 启动 Codex。
6. 在 Codex 中输入 `/model` 选择第三方模型。
7. 在 Relay Switch Logs 页面查看请求和错误。

这样你可以把模型和 Provider 管理留在 Relay Switch 里，把 Codex 当成稳定的编码入口使用。

