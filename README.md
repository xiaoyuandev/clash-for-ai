# Relay Switch

[English](./README.en.md) | [中文](./README.md)

<a href="https://www.relayswitch.dev/" target="_blank" rel="noopener noreferrer">公开文档</a> | <a href="https://www.relayswitch.dev/deep-link-import/" target="_blank" rel="noopener noreferrer">Deep Link 导入说明</a>

Relay Switch 把多个 AI 中转 API、原生大模型来源和本地工具接入统一到一个本地入口。

如果你经常在 Cursor、Kiro、Cherry Studio、Codex、Claude Code、OpenClaw、Hermes Agent 或自己的脚本里切换模型和服务商配置，Relay Switch 的目标是让你只配置一次本地地址，之后在一个界面里切换上游。

## 目录

- [为什么需要 Relay Switch，解决了哪些痛点？](#为什么需要-relay-switch解决了哪些痛点)
- [它能帮你做什么](#它能帮你做什么)
- [Screenshot](#screenshot)
- [安装 & 快速开始](#安装--快速开始)
- [桌面端模块说明](#桌面端模块说明)
- [Provider 和 Model 到底有什么区别？](#provider-和-model-到底有什么区别)
- [使用指南](#使用指南)
- [文档入口](#文档入口)
- [常见问题](#常见问题)
- [本地开发](#本地开发)
- [项目结构](#项目结构)

## 为什么需要 Relay Switch，解决了哪些痛点？

很多 AI 用户真正头疼的不是“没有模型可用”，而是：

1. 中转 API 服务商不稳定，其中一家额度用完或者掉线时，需要快速切换到另一家服务商
2. 同时使用多个编程工具、桌面客户端或 SDK 脚本时，每次切换服务商都要重复修改配置
3. 部分用户不会、也不想频繁改配置文件，但仍然希望能稳定使用多个模型来源

Relay Switch 的做法是在你的工具前面放一个本地网关。

你的工具只需要统一接入一次：

```text
http://127.0.0.1:3456/v1
```

之后切换模型、中转 API 服务商或本地模型来源时，只需要在 Relay Switch 里操作，不需要逐个修改 Cursor、Kiro、Cherry Studio、Codex、Claude Code 或脚本配置。

## 它能帮你做什么

1. 一键切换管理中转 API 服务，而无需重启编程工具。例如基于 `new-api`、`sub2api` 等框架搭建的中转站
2. 在本地维护一个 Local Gateway，用来管理希望直接接入各类兼容 OpenAI 或者 Anthropic 接口的原生大模型的使用场景。例如 MiniMax、DeepSeek 等。在模型页面添加，在供应商页面启用 LocalGateway 即可使用
3. 支持各种编程工具，例如 Codex CLI、Claude Code、OpenCode、Codex app、Cursor、VSCode、Cherry Studio
4. 为本地 AI 工具提供统一接口 `http://127.0.0.1:3456`。切换模型或者切换中转服务不需要重启编程工具，不会每次切换都修改编程工具的配置文件
5. 跨平台，可以安装在 Windows、macOS、Linux 桌面环境，也可以安装在 Linux server 和 WSL 里使用
6. 可以一键监测当前 API Key 的可用模型列表，支持只显示可用的模型列表筛选
7. 支持从网页一键导入 Provider 或 Model 配置，可以直接体验 [Deep Link 导入](https://www.relayswitch.dev/deeplink.html)

## Screenshot

<p align="center">
  <img src="./docs/images/readme/quick-start-provider-form.png" style="width: 49%; height: auto;">
  <img src="./docs/images/readme/connectatool.png" style="width: 49%; height: auto;">
</p>

<p align="center">
  <img src="./docs/images/readme/models-config.png" style="width: 49%; height: auto;">
  <img src="./docs/images/readme/tools-config.png" style="width: 49%; height: auto;">
</p>

## 安装 & 快速开始

### 1. 选择适合你的运行方式

**1.1 macOS、Windows、Ubuntu Desktop 桌面端用户**

推荐使用桌面版客户端，可到 [Release页面](https://github.com/xiaoyuandev/relay-switch/releases) 下载对应平台的最新版本

对于 macOS 用户首次安装启动时可能会看到类似提示：

```text
“Relay Switch” 无法打开，因为无法验证开发者。
```

或者：

```text
“Relay Switch” 无法打开，因为 Apple 无法检查其是否包含恶意软件。
```

这是因为当前项目的公开分发流程里，macOS 这边采用了免费 ad-hoc 风格的签名路径，而不是完整的付费 Apple开发者账户的 可信分发链路。

如果遇到这种情况，用户可以这样操作：

1. 如果应用还在下载目录或临时目录中，先移动到 `/Applications`
2. 在 Finder 中右键 `Relay Switch.app`
3. 选择 `打开`
4. 在系统确认框里再次选择 `打开`

如果右键打开后仍然没有成功，可以继续：

1. 打开 `系统设置`
2. 进入 `隐私与安全性`
3. 找到 Relay Switch 的安全拦截提示
4. 点击 `仍要打开`

如果你熟悉命令行，也可以在确认应用来自官方发布页面后，使用 `xattr` 移除 macOS 给应用添加的隔离标记：

```bash
sudo xattr -rd com.apple.quarantine "/Applications/Relay Switch.app"
```

执行完成后，再从 Finder 或 Launchpad 启动 `Relay Switch.app`。

通常第一次成功打开之后，后续再启动就不会反复出现相同的拦截提示。

如果某个 release 同时提供了 `.pkg` 安装包，优先使用 `.pkg` 安装，通常体验会比手动拖拽裸 `.app` 更稳定。


**1.2 WSL 或 Linux Server 用户** 推荐使用命令行安装：

```bash
curl -fsSL https://raw.githubusercontent.com/xiaoyuandev/relay-switch/main/scripts/install.sh | bash
```

安装完成后默认提供：

1. Web 管理界面：`http://127.0.0.1:3456`
2. OpenAI-compatible 本地入口：`http://127.0.0.1:3456/v1`

如果需要在安装时指定监听地址和端口，可以通过环境变量覆盖默认值：

```bash
RELAY_SWITCH_HTTP_HOST=0.0.0.0 RELAY_SWITCH_HTTP_PORT=8080 \
  curl -fsSL https://raw.githubusercontent.com/xiaoyuandev/relay-switch/main/scripts/install.sh | bash
```

`0.0.0.0` 表示监听所有网卡，远程访问时请使用服务器实际 IP 或域名，例如 `http://<server-ip>:8080`。

**服务管理**

默认会安装一个 `systemd --user` 服务：

```bash
systemctl --user status relay-switch
systemctl --user restart relay-switch
journalctl --user -u relay-switch -n 200 -f
```

同时也会生成一个辅助命令：

```bash
relay-switch start
relay-switch stop
relay-switch restart
relay-switch status
relay-switch logs
relay-switch run
```

说明：

1. `start / stop / restart / status / logs` 优先走 `systemd --user`
2. `run` 会以前台方式直接启动 core，适合调试


**远程服务器访问方式**

如果你是在远程 Linux server 部署，通常有两种方式：

- 方式一：通过 SSH 隧道访问 Web UI 管理页面

本地执行：

```bash
ssh -N -L 3456:127.0.0.1:3456 user@host
```

然后本地浏览器访问：

```text
http://127.0.0.1:3456
```

- 方式二：反向代理

把你的反向代理转发到：

```text
http://127.0.0.1:3456
```

注意：

1. 当前 core 默认绑定 `127.0.0.1`
2. 更安全的做法是继续保持本地绑定，再由 Nginx / Caddy 负责公网入口


完整说明见：

- [WSL / Linux Server 部署与使用说明](./docs/zh-cn/wsl-linux-server-guide.zh-CN.md)
- [WSL / Linux Server Deployment Guide (English)](./docs/en/wsl-linux-server-guide.md)

### 2. 添加 Provider 或 Model 来源

在 `Providers` 页面添加你的中转 API 服务商，填写 `Name`、`Base URL` 和 `API Key`。

如果你希望直接接入原生大模型，可以在 `Models` 页面添加本地模型来源，并同步到本地 Models Gateway。

### 3. 把工具统一接到本地地址

大多数支持 OpenAI-compatible 接口的工具都可以这样配置：

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

之后无论你切换上游 Provider，还是切换本地 Models Gateway，工具侧都可以继续使用同一个本地地址，无需修改工具侧配置。

### 4. 从网页一键导入配置

Relay Switch 支持通过 Deep Link 从网页导入 Provider 或 Model 配置。

> 也欢迎各大中转API服务商接入，接入方式可查看: https://www.relayswitch.dev/deep-link-import/

可以先体验：

```text
https://www.relayswitch.dev/deeplink.html
```

## 桌面端模块说明

当前桌面应用主要分成五个模块。

### Provider 和 Model 到底有什么区别？

为什么产品里同时有 `Provider` 页面和 `Model` 页面？

一句话解释：`Provider` 用来管理中转站，`Model` 页面背后则是一个运行在本地的 中转站 LocalGateway，用来统一管理原生模型来源。

架构上可以把它理解成这样：

```mermaid
flowchart LR
    P[Provider]

    P --> R1[中转站 A\nnew-api]
    P --> R2[中转站 B\nsub2api]
    P --> R3[LocalGateway\nModel 页面]

    R1 --> M11[gpt-4.1]
    R1 --> M12[deepseek-chat]
    R1 --> M13[qwen-plus]

    R2 --> M21[claude-sonnet]
    R2 --> M22[gpt-4o]
    R2 --> M23[gemini-2.5-pro]

    R3 --> M31[glm-4.5]
    R3 --> M32[minimax-text]
    R3 --> M33[qwen-max]
```

这两个页面看起来都和“大模型”有关，但它们解决的问题并不一样：

1. `Provider` 解决的是“我的 Codex、Claude Code、Cursor 应该通过哪个 API 服务出去”
2. `Model` 解决的是“我本地想管理哪些原生模型来源，并通过 LocalGateway 统一暴露出去”

所以也可以把它理解成：

1. `Provider` 页面面向工具接入和上游切换
2. `Model` 页面面向原生模型来源管理
3. LocalGateway 会作为一个可选的 Provider 出现在 `Providers` 页面里

这个拆分的目的不是增加概念，而是为了避免把“切换中转服务”和“管理原生模型来源”混成一类问题。前者更关注 `Base URL`、`API Key`、当前启用哪个上游；后者更关注模型列表、能力差异、协议兼容和本地统一暴露。

更详细的介绍，可以查看：[Provider 和 Model 详细介绍](./docs/zh-cn/provider-vs-model.md)。

### 1. Providers

`Providers` 页面用于管理主本地网关后面可以切换的上游服务。

最容易理解的方式是：

1. `Providers` 主要用于管理各种中转 API 服务
2. 这些服务通常是远程聚合或代理平台
3. 例如类似 `new-api`、`one-api`、`sub2api` 这类服务

你可以在这里：

1. 添加和编辑 Provider
2. 切换当前激活的上游 Provider
3. 运行 Provider health check
4. 查看某个 Provider 实际暴露的模型
5. 为 Claude Code 配置当前 Provider 的模型槽位映射

所以如果用户主要目的是在多个远程中转服务之间切换，`Providers` 就是最核心的页面。

### 2. Models

`Models` 页面要解决的是另一类问题。

它用于管理一个运行在本地的 Models Gateway。这个本地 gateway 会把原生模型上游来源统一整理出来，每一条 source 都可以指向：

1. OpenAI-compatible 上游
2. Anthropic-compatible 上游

你可以在这里：

1. 添加本地模型来源
2. 自动探测或手动填写模型 ID
3. 启用或禁用某个模型来源
4. 把这些来源同步到内嵌的本地 gateway runtime

这个模块存在的原因是：

1. 很多原生大模型上游并不适合直接按“单个激活 Provider”去管理
2. 不同原生上游会暴露不同的模型 ID 和模型列表
3. 用户有时需要的是一个运行在本地的兼容网关，效果更像在本机跑一个小型的 `new-api` 或 `sub2api`
4. 这样就可以把很多兼容 OpenAI-compatible 和 Anthropic-compatible 的原生大模型统一挂到这一层本地 gateway 下面

所以 `Models` 页面引入的是一层独立的本地 Models Gateway。它不是用来切换远程聚合中转服务，而是让 Relay Switch 可以在本地维护一组原生模型来源，并通过一个本地兼容网关向外暴露。

实际关系可以这样理解：

1. `Providers` 负责管理远程中转服务，也负责展示这个本地运行的 Models Gateway
2. `Models` 负责管理这个本地 Models Gateway 背后的 source 列表
3. 这个本地 Models Gateway 会默认出现在 `Providers` 管理列表里，作为一个可选择的 Provider

也就是说：

1. `Models` 配置的是“本地 gateway 里面有哪些原生模型来源”
2. 这个本地 gateway 会作为一个 Provider 出现在 `Providers` 页面
3. `Providers` 页面最终负责让用户在“远程中转服务”和“本地 Models Gateway”之间切换

#### 模型预设配置

`Models` 表单可以从远程 JSON 加载可选的模型预设。预设只是辅助填写表单的快捷方式，最终保存的 `Name`、`Provider Type`、`Base URL`、`API Key` 和 `Model IDs` 仍然以用户在表单里的最终输入为准。

默认配置地址是：

```text
https://www.relayswitch.dev/model-presets.json
```

前端构建时可以通过环境变量覆盖：

```bash
VITE_MODEL_PRESETS_URL=https://example.com/model-presets.json pnpm --filter web build
```

前端会在打开 `Models` 页面时拉取这个 JSON，校验表单需要的字段，并把 JSON catalog 存到浏览器 local storage 作为本地缓存。格式不合法的 JSON 不会写入缓存。如果拉取或校验失败，且本地已经有缓存，则继续使用缓存。如果没有缓存，预设列表为空，用户仍然可以手动填写模型表单。

源文件是 `config/model-presets.json`。docs 站点构建时会把它发布到默认 URL 对应的静态 JSON endpoint，并在 schema 不合法时让构建失败。你可以用 `pnpm validate:model-presets` 在本地校验。仓库里的默认内容初始化为空列表：

```json
{
  "schema_version": 1,
  "updated_at": "",
  "presets": []
}
```

示例配置：

```json
{
  "schema_version": 1,
  "updated_at": "2026-05-29",
  "presets": [
    {
      "id": "openai-gpt-4o",
      "label": "OpenAI GPT-4o",
      "description": "OpenAI GPT-4o through the official OpenAI-compatible endpoint.",
      "aliases": ["gpt-4o", "4o"],
      "providers": [
        {
          "id": "openai",
          "label": "OpenAI",
          "provider_type": "openai-compatible",
          "base_url": "https://api.openai.com/v1",
          "models_api": "supported",
          "model_ids": []
        }
      ],
      "tags": ["openai", "popular"]
    },
    {
      "id": "deepseek",
      "label": "DeepSeek",
      "description": "DeepSeek OpenAI-compatible endpoint with preset model IDs.",
      "aliases": ["deepseek-chat", "deepseek-reasoner"],
      "providers": [
        {
          "id": "deepseek",
          "label": "DeepSeek",
          "provider_type": "openai-compatible",
          "base_url": "https://api.deepseek.com/v1",
          "models_api": "unsupported",
          "model_ids": ["deepseek-chat", "deepseek-reasoner"]
        }
      ],
      "tags": ["deepseek", "popular", "reasoning"]
    }
  ]
}
```

字段说明：

1. `schema_version` 必须是 `1`。
2. `presets[].id` 必须唯一。
3. `presets[].label` 会显示在 `Name` 输入框的下拉列表里。
4. `aliases` 和 `tags` 会参与下拉列表搜索过滤。
5. `providers[].provider_type` 必须是 `openai-compatible` 或 `anthropic-compatible`。
6. `providers[].base_url` 会在选择对应 provider 后自动填入表单。
7. `providers[].models_api` 可选值：
   - `supported`：用户填写 API Key 后，Relay Switch 会尝试自动拉取模型 ID。
   - `unsupported`：Relay Switch 会直接使用 JSON 里的 `model_ids`。
   - `auto`：预留给暂时不确定行为的 provider。
8. 当 `models_api` 是 `supported` 时，`providers[].model_ids` 可以为空数组。当 `models_api` 是 `unsupported` 时，必须至少配置一个模型 ID。
9. `disabled: true` 可以隐藏某个 preset，但不需要从 JSON 里删除。
10. `deprecated: true` 可以保留某个 preset，同时标记它后续会清理。

#### Provider 预设配置

`Providers` 表单可以从远程 JSON 加载可选的供应商预设。预设只负责辅助填写 `Name` 和 `Base URL`，最终保存的 `API Key` 仍然由用户在表单里填写。

默认配置地址是：

```text
https://www.relayswitch.dev/provider-presets.json
```

前端构建时可以通过环境变量覆盖：

```bash
VITE_PROVIDER_PRESETS_URL=https://example.com/provider-presets.json pnpm --filter web build
```

源文件是 `config/provider-presets.json`。以后新增 Provider 预设只需要更新这个文件。docs 站点构建时会把它发布到默认 URL 对应的静态 JSON endpoint，并在 schema 不合法时让构建失败。你可以用 `pnpm validate:provider-presets` 在本地校验。

示例配置：

```json
{
  "schema_version": 1,
  "updated_at": "2026-06-08",
  "presets": [
    {
      "name": "kocodex",
      "base_url": "https://kocodex.link"
    }
  ]
}
```

字段说明：

1. `schema_version` 必须是 `1`。
2. `presets[].name` 必须唯一。
3. `presets[].base_url` 必须是合法的绝对 URL。
4. `presets[]` 条目只支持 `name` 和 `base_url`。
5. 预设不会保存或分发 `API Key`。

### 3. Tools

`Tools` 页面用于帮助客户端工具正确接入 Relay Switch。

你可以在这里：

1. 复制已经整理好的本地接入参数
2. 对 Codex CLI、Claude Code 执行一键接入
3. 查看 Cursor、Cherry Studio、SDK 脚本等工具的接入说明
4. 通过拖拽的方式将支持的模型匹配到 Claude Code 对应模型上，在 Claude Code 对话里通过 `/model` 命令切换对应的模型

### 4. Logs

`Logs` 页面用于查看经过本地网关的请求历史。

你可以在这里：

1. 查看最近请求
2. 查看 Provider、Model、路径、延迟等信息
3. 当上游异常时，直接读取失败记录

### 5. Settings

`Settings` 页面用于管理桌面应用本身的系统行为。

你可以在这里：

1. 查看运行时状态
2. 修改本地端口
3. 检查桌面应用更新
4. 控制启动与托盘行为
   - 开机自启
   - 静默启动
   - 关闭时最小化到托盘

## 使用指南

下面是把工具接入 Relay Switch 的基本流程。

### 1. 在 Relay Switch 里添加 Provider

打开桌面应用中的 `Providers` 页面，填写：

1. `Name`
2. `Base URL`
3. `API Key`

对于 OpenAI-compatible 中转服务，通常推荐填写带 `/v1` 的 Base URL。

对于其他兼容接口，是否填写 `/v1` 取决于上游文档和实际实现。当前项目对 OpenAI-compatible 场景支持最成熟。

<p align="center">
  <img src="./docs/images/readme/quick-start-provider-form.png" style="width: 100%; height: auto;">
</p>

### 2. 把工具接到本地地址

大多数支持 OpenAI-compatible 接口的工具都可以这样配置：

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

如果运行时使用的不是 `3456`，请以桌面应用里显示的 `connected api base` 为准。

### 3. 需要时在 `Tools` 页面完成专用接入

`Tools` 页面会提供：

1. 可直接复制的接入参数
2. 对 Codex CLI、Claude Code 的一键配置能力
3. 对 Cursor、Cherry Studio、SDK 脚本等场景的接入引导
4. 通过拖拽的方式将支持的模型匹配到 Claude Code 对应模型上，在 Claude Code 对话里通过 `/model` 命令切换对应的模型

### CLI 工具

对于 Codex CLI 这类 OpenAI 兼容 CLI，先在当前 shell 中设置环境变量，再启动工具：

```bash
export OPENAI_BASE_URL="http://127.0.0.1:3456/v1"
export OPENAI_API_KEY="dummy"
```

然后在同一个终端会话里启动 CLI。

对于 Claude Code 这类 Anthropic 风格工具，当前项目提供了对应的环境变量接入方式：

```bash
export ANTHROPIC_BASE_URL="http://127.0.0.1:3456"
export ANTHROPIC_AUTH_TOKEN="dummy"
```

在 Relay Switch 中，你也可以直接打开 `Tools` 页面，使用对已支持 CLI 的一键接入流程。

需要说明的是：当前版本最稳定的本地入口仍然是 OpenAI-compatible 这条链路；Anthropic 风格本地接入和上游兼容能力仍在持续完善。如果你的工具同时支持自定义 OpenAI-compatible endpoint，优先使用 `http://127.0.0.1:3456/v1` 会更稳妥。

### IDE / 插件 / 桌面客户端

对于 IDE、编辑器插件和桌面聊天客户端，打开它们的 Provider 配置页面并填写：

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

在 Relay Switch 里，你也可以进入 `Tools` 页面查看这些已整理好的接入参数。

<p align="center">
  <img src="./docs/images/readme/settings.png" style="width: 49%; height: auto;">
  <img src="./docs/images/readme/connectatool.png" style="width: 49%; height: auto;">
</p>

如果工具像 Cursor 或 Cherry Studio 一样还要求选择 Provider Type / Protocol，请优先选择 OpenAI-compatible 自定义 Provider 模式，再填写上面的参数。

对于 Cursor，可以进入它的自定义 Provider 配置界面，选择 OpenAI-compatible 模式，然后填写本地 Base URL 和 `dummy` API Key。

<p align="center">
  <img src="./docs/images/readme/corsor-config.png" style="width: 72%; height: auto;">
</p>

### SDK 脚本 / 本地应用

如果你希望在自己的脚本里，通过 Relay Switch 和当前激活的 Provider 交互，只需要把 SDK 或 HTTP 请求指向本地网关，而不是直接请求上游中转服务。

使用 OpenAI SDK 的示例：

```ts
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: "dummy",
  baseURL: "http://127.0.0.1:3456/v1"
});

const response = await client.responses.create({
  model: "gpt-4.1",
  input: "Say hello from Relay Switch."
});

console.log(response.output_text);
```

也可以直接使用 HTTP 请求：

```bash
curl http://127.0.0.1:3456/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dummy" \
  -d '{
    "model": "gpt-4.1",
    "messages": [
      { "role": "user", "content": "Say hello from Relay Switch." }
    ]
  }'
```

最终由哪个模型实际响应，仍然取决于你的脚本发送的模型名，以及桌面应用里当前激活的是哪个 Provider。

## 文档入口

如果你想看更完整的分步说明、工具接入示例和排障说明，请继续阅读：

- [使用教程](./docs/zh-cn/user-guide.md)
- [Provider 和 Model 详细介绍](./docs/zh-cn/provider-vs-model.md)
- [在 Codex 里使用第三方大模型](./docs/zh-cn/codex-third-party-models.md)
- [WSL / Linux Server 部署与使用说明](./docs/zh-cn/wsl-linux-server-guide.zh-CN.md)
- [WSL / Linux Server Deployment Guide (English)](./docs/en/wsl-linux-server-guide.md)
- [English README](./README.en.md)

如果你是 `WSL` 或 `Linux server` 用户，建议优先阅读 server 部署文档，其中包含 `RELAY_SWITCH_VERSION` 固定版本安装与回滚说明。

## 当前更适合怎样理解协议支持

通常情况下，上游 Gateway 会同时支持 OpenAI 和 Anthropic 两种协议标准。

Relay Switch 当前也围绕这两类兼容接口设计，但从实际实现成熟度看：

1. OpenAI-compatible 本地接入是当前最清晰、最稳定的主路径
2. Anthropic-compatible 上游认证和部分工具接入已经覆盖
3. Anthropic 风格的完整本地协议支持仍在持续完善

因此，README 和接入建议里凡是能选择 OpenAI-compatible 自定义 endpoint 的工具，当前都建议优先走这条路径。

## 关于模型列表

Provider 模型列表获取已经实现，但它更适合被理解为“兼容能力”而不是“所有上游都保证提供的标准能力”。

原因通常包括：

1. 不同 Gateway 的模型列表 endpoint 可能不完全一致
2. 有些上游不提供标准模型列表接口
3. 返回 JSON 结构可能有差异

所以如果某个 Provider 能正常转发请求，但模型列表显示不完整，并不一定代表这个 Provider 本身不可用。

## 常见问题

<details>
<summary>用户的请求日志记录会上传到远端吗？</summary>

不会。用户的请求日志记录仅保存在本地，我们不会上传任何日志记录到远端。

</details>

<details>
<summary>为什么 macOS 初次安装会提示“无法验证开发者”？</summary>

当前公开发布的 macOS 构建，在用户第一次安装或第一次启动时，仍然可能出现 Gatekeeper 安全拦截。

原因是：当前项目的公开分发流程里，macOS 这边仍然采用了免费 ad-hoc 风格的签名路径，而不是对所有公开产物都走完整的付费 Apple 可信分发链路。

这也是为什么用户可能会看到类似提示：

```text
“Relay Switch” 无法打开，因为无法验证开发者。
```

或者：

```text
“Relay Switch” 无法打开，因为 Apple 无法检查其是否包含恶意软件。
```

如果遇到这种情况，用户可以这样操作：

1. 如果应用还在下载目录或临时目录中，先移动到 `/Applications`
2. 在 Finder 中右键 `Relay Switch.app`
3. 选择 `打开`
4. 在系统确认框里再次选择 `打开`

如果右键打开后仍然没有成功，可以继续：

1. 打开 `系统设置`
2. 进入 `隐私与安全性`
3. 找到 Relay Switch 的安全拦截提示
4. 点击 `仍要打开`

如果你熟悉命令行，也可以在确认应用来自官方发布页面后，使用 `xattr` 移除 macOS 给应用添加的隔离标记：

```bash
sudo xattr -rd com.apple.quarantine "/Applications/Relay Switch.app"
```

执行完成后，再从 Finder 或 Launchpad 启动 `Relay Switch.app`。

通常第一次成功打开之后，后续再启动就不会反复出现相同的拦截提示。

如果某个 release 同时提供了 `.pkg` 安装包，优先使用 `.pkg` 安装，通常体验会比手动拖拽裸 `.app` 更稳定。

</details>

## 本地开发

要求：

1. Node.js
2. pnpm
3. 如果要本地构建核心服务，还需要 Go toolchain

安装依赖：

```bash
pnpm install
```

启动桌面应用开发模式：

```bash
pnpm dev
```

启动 Web 端本地联调模式：

```bash
pnpm dev:web
```

`pnpm dev:web` 会同时启动 core 服务和 Web dev server，适合只调试 Web 管理界面时使用。默认端口为：

1. core API：`3456`
2. local gateway runtime：`3457`

如果本机端口被其他程序占用，或需要接入本地开发版 `ai-mini-gateway`，可以在仓库根目录的 `.env.local` 中覆盖：

```bash
HTTP_PORT=3456
LOCAL_GATEWAY_RUNTIME_PORT=3457
LOCAL_GATEWAY_RUNTIME_EXECUTABLE=/path/to/ai-mini-gateway/bin/ai-mini-gateway
```

这些配置都是本地联调辅助配置；不配置时会使用默认端口。`pnpm dev:web` 启动前会释放上述端口上的旧监听进程，确保 core 和 local gateway 使用最新代码重新启动。

构建桌面应用：

```bash
pnpm build
```

构建各平台安装包：

```bash
pnpm --filter desktop build:mac
pnpm --filter desktop build:win
pnpm --filter desktop build:linux
```

如果 `ai-mini-gateway` 上游发布了新版本，打包前可以先同步内嵌运行时版本：

```bash
pnpm --filter desktop update:ai-mini-gateway-runtime
pnpm --filter desktop update:ai-mini-gateway-runtime v0.1.1
pnpm --filter desktop update:ai-mini-gateway-runtime v0.1.1 --prepare
```

默认命令会跟到 latest release，也可以传具体版本号固定到指定 tag。加上 `--prepare` 会顺手刷新本地内嵌的运行时二进制和版本元数据。

## 项目结构

```text
apps/desktop   Electron 桌面应用
core/          Go 本地网关与 Provider 管理后端
docs/          面向用户的公开文档
```

## License

本项目使用 GNU Affero General Public License v3.0 only。

详见：

- [LICENSE](./LICENSE)

## Brand Notice

本仓库源码采用 AGPL-3.0-only 授权，但以下内容并不默认随源码授权一起开放使用：

1. 项目名称 `Relay Switch`
2. Logo
3. Icon
4. 其他品牌资产

## 状态

项目仍在持续开发中，接口、打包流程和更新行为后续仍可能调整。
