# Clash for AI

Clash for AI 是一个面向多 AI Gateway / 中转 API 使用场景的本地桌面网关工具。

它的定位是：

1. 在本地提供一个统一 API 入口
2. 在这个统一入口后面切换不同上游 Gateway
3. 用桌面界面管理 Provider、健康检查和请求日志

它并不是某一个特定 AI 工具的专用管理器，而是更偏向“本地网关 + 多上游切换控制台”。

[English README](./README.md)

它当前提供：

1. 一个稳定的本地统一接入地址
2. 一个可视化的 Provider 切换控制台
3. 本地请求日志和健康检查能力，方便排障

## 核心思路

Clash for AI 的思路很简单：

1. 你的工具统一连接本地网关
2. 上游 Gateway 的切换在桌面应用中完成
3. 真实 Provider 凭证、健康状态和请求日志都在本地集中管理

## Screenshot

<p align="center">
  <img src="./docs/images/readme/quick-start-provider-form.png" style="width: 100%; height: auto;">
</p>

<p align="center">
  <img src="./docs/images/readme/connectatool.png" style="width: 100%; height: auto;">
</p>

## 这个项目解决了什么问题

Clash for AI 主要面向经常切换不同 AI Gateway / 中转 API 的用户。

它主要解决两个问题：

1. 中转 API 服务不稳定，用户需要在不同中转 API 服务商之间频繁切换
2. 当你同时使用多个编程工具、聊天客户端或脚本时，每次切换服务商都要重复修改配置

Clash for AI 的做法是在你的工具前面放一个本地 Gateway。

你的工具只需要统一接入本地地址一次，之后切换上游 Gateway 时，不再需要逐个修改工具配置，只需要在桌面应用里切换即可。

## 它是怎么工作的

Clash for AI 会在你的机器上运行一个本地 API Gateway。

大多数编辑器、聊天客户端、CLI 工具或自定义脚本会先连接本地地址：

```text
http://127.0.0.1:3456/v1
```

然后 Clash for AI 再把请求转发到当前在桌面应用中激活的 Provider。

当前版本的本地接入能力主要围绕 OpenAI-compatible 本地入口展开；对于 Anthropic-compatible 上游和部分 Claude 风格工具，项目已经有适配和接入流程，但这部分能力仍在持续完善。

这意味着：

1. 切换 Provider 时，不需要重新配置每个工具
2. Provider 凭证统一在本地管理
3. 可以直接在桌面界面查看健康状态和请求日志



## 快速接入

如果你暂时不想先看完整使用手册，可以先按下面方式快速接入。

### 1. 在 Clash for AI 里添加 Provider

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

在 Clash for AI 中，你也可以直接打开 `Tools` 页面，使用对已支持 CLI 的一键接入流程。

需要说明的是：当前版本最稳定的本地入口仍然是 OpenAI-compatible 这条链路；Anthropic 风格本地接入和上游兼容能力仍在持续完善。如果你的工具同时支持自定义 OpenAI-compatible endpoint，优先使用 `http://127.0.0.1:3456/v1` 会更稳妥。

### IDE / 插件 / 桌面客户端

对于 IDE、编辑器插件和桌面聊天客户端，打开它们的 Provider 配置页面并填写：

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

在 Clash for AI 里，你也可以进入 `Tools` 页面查看这些已整理好的接入参数。

<p align="center">
  <img src="./docs/images/readme/settings.png" style="width: 100%; height: auto;">
</p>

<p align="center">
  <img src="./docs/images/readme/connectatool.png" style="width: 100%; height: auto;">
</p>

如果工具像 Cursor 或 Cherry Studio 一样还要求选择 Provider Type / Protocol，请优先选择 OpenAI-compatible 自定义 Provider 模式，再填写上面的参数。

对于 Cursor，可以进入它的自定义 Provider 配置界面，选择 OpenAI-compatible 模式，然后填写本地 Base URL 和 `dummy` API Key。

<p align="center">
  <img src="./docs/images/readme/corsor-config.png" style="width: 100%; height: auto;">
</p>

### SDK 脚本 / 本地应用

如果你希望在自己的脚本里，通过 Clash for AI 和当前激活的 Provider 交互，只需要把 SDK 或 HTTP 请求指向本地网关，而不是直接请求上游中转服务。

使用 OpenAI SDK 的示例：

```ts
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: "dummy",
  baseURL: "http://127.0.0.1:3456/v1"
});

const response = await client.responses.create({
  model: "gpt-4.1",
  input: "Say hello from Clash for AI."
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
      { "role": "user", "content": "Say hello from Clash for AI." }
    ]
  }'
```

最终由哪个模型实际响应，仍然取决于你的脚本发送的模型名，以及桌面应用里当前激活的是哪个 Provider。

## 文档入口

如果你想看更完整的分步说明、工具接入示例和排障说明，请继续阅读：

- [使用教程](./docs/user-guide.md)
- [English README](./README.md)

## 当前更适合怎样理解协议支持

通常情况下，上游 Gateway 会同时支持 OpenAI 和 Anthropic 两种协议标准。

Clash for AI 当前也围绕这两类兼容接口设计，但从实际实现成熟度看：

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

1. 项目名称 `Clash for AI`
2. Logo
3. Icon
4. 其他品牌资产

## 状态

项目仍在持续开发中，接口、打包流程和更新行为后续仍可能调整。
