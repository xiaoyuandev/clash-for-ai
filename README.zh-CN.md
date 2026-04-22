# Clash for AI

Clash for AI 是一个面向多 AI Provider 使用场景的本地桌面网关工具，适合同时在多个编程工具、聊天客户端或脚本里接入不同 AI 服务的用户。

[English README](./README.md)

它提供：

1. 一个稳定的本地统一接入地址
2. 一个可视化的 Provider 切换控制台
3. 本地请求日志和健康检查能力，方便排障

## 这个项目解决了什么问题

如果你同时使用多个 AI Provider，日常切换通常会带来这些成本：

1. 需要反复修改环境变量或工具配置文件
2. 需要在编辑器、CLI、桌面客户端里重复填写 Base URL 和 API Key
3. 想测试另一个 Provider 时，要重新改一轮配置
4. 出现失败时，很难快速判断是鉴权、额度、网络还是上游服务异常

Clash for AI 的做法是在你的工具前面放一个本地 Gateway。

你的工具只需要统一接入本地地址一次，之后切换上游 Provider 时，不再需要逐个修改工具配置，只需要在桌面应用里切换即可。

## 这个项目和 cc-switch 的区别

`cc-switch` 的核心思路是改写环境变量或工具配置，让工具切到另一个服务商。

Clash for AI 的思路不同：

1. 工具统一指向一个稳定的 localhost 地址，而不是每次切换都改配置
2. Provider 切换通过桌面 UI 完成，而不是反复编辑配置文件
3. 内置日志和健康检查，便于定位问题来源
4. 多个工具可以一起切换，因为它们都依赖同一个本地 Gateway

一句话概括：`cc-switch` 是配置改写工具，Clash for AI 是“本地代理 + 管理面板”。

## 它是怎么工作的

Clash for AI 会在你的机器上运行一个本地 API Gateway。

你的编辑器、聊天客户端、CLI 工具或自定义脚本先连接本地地址：

```text
http://127.0.0.1:3456/v1
```

然后 Clash for AI 再把请求转发到当前在桌面应用中激活的 Provider。

这意味着：

1. 切换 Provider 时，不需要重新配置每个工具
2. Provider 凭证统一在本地管理
3. 可以直接在桌面界面查看健康状态和请求日志

## 当前能力

1. 基于 Electron 的桌面应用
2. 基于 Go 的本地转发网关
3. Provider 管理
4. 当前生效 Provider 切换
5. 健康检查
6. 请求日志
7. 默认端口被占用时自动选择可用本地端口
8. 打包版本内置更新流程

## 如何使用

查看用户使用文档：

- [使用教程](./docs/user-guide.md)
- [English README](./README.md)

大多数支持 OpenAI-compatible 接口的工具都可以这样配置：

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

如果运行时使用的不是 `3456`，请以桌面应用里显示的 `connected api base` 为准。

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
