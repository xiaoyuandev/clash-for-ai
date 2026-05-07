# Web 管理端与工具接入开发计划

## 文档目的

这份文档用于把以下三份架构文档收敛为一份可执行开发计划：

1. [.docs/web-management-architecture.md](/Users/yuanjunliang/Documents/workspace/profile/clash-for-ai/.docs/web-management-architecture.md:1)
2. [.docs/apps-web-architecture.md](/Users/yuanjunliang/Documents/workspace/profile/clash-for-ai/.docs/apps-web-architecture.md:1)
3. [.docs/go-core-tool-integrations-architecture.md](/Users/yuanjunliang/Documents/workspace/profile/clash-for-ai/.docs/go-core-tool-integrations-architecture.md:1)

后续开发以本计划文档作为执行基线。

如果实现过程中需要改变顺序、范围或职责边界，必须先更新本计划文档和对应架构文档，再调整代码。

## 最高优先级原则

整个开发过程必须始终坚守以下原则：

1. `Electron` 仍是本地主入口
2. `Web / PWA` 是 WSL 和 Linux server 的补充入口
3. 不允许把“页面复用”演进成“桌面端被 Web 端替代”
4. `Go core` 是唯一业务中心
5. 工具一键配置能力迁移到 `Go core` 是为了统一能力，不是为了削弱 Electron 的定位

这五条原则优先级高于具体实现便利性。

## 范围定义

本计划覆盖以下工作：

1. 新增 `apps/web`
2. 为 WSL 和 Linux server 提供浏览器管理入口
3. 将工具一键配置能力从 Electron 主进程迁移到 Go core
4. 让 Electron 和 Web 共享部分页面能力
5. 为 Web 端增加可选的 PWA 安装形态

本计划不覆盖以下工作：

1. 用 Web/PWA 替代 Electron
2. 重写 Go core 为 TS/Node
3. 增加新的业务后端
4. 多节点统一控制台
5. 多租户、鉴权、用户体系
6. 非必要的大规模状态管理重构

## 当前现状

当前仓库现状如下：

1. `core/` 已经提供核心 API 和本地网关能力
2. `apps/desktop/` 已经提供 Electron 桌面壳和 React renderer
3. 工具一键配置能力当前主要位于 `apps/desktop/src/main/tool-integrations.ts`
4. 还没有 `apps/web`
5. WSL 和 Linux server 还没有专用管理入口

## `apps/web` 推荐技术栈

`apps/web` 推荐采用以下技术栈组合：

1. `React`
2. `TypeScript`
3. `Vite`
4. `React Router`
5. `Tailwind CSS`
6. 原生 `fetch` + 轻量 `api.ts` 封装
7. 延续现有国际化方案思路
8. `vite-plugin-pwa`

推荐原则：

1. 优先复用当前 `apps/desktop` renderer 的技术基础
2. 不引入新的重型状态管理库
3. 不引入新的 UI 组件库
4. 样式系统统一采用 `Tailwind CSS`
5. PWA 只作为安装形态增强，不改变产品主入口关系

明确不推荐在当前阶段引入：

1. `Next.js`
2. `Remix`
3. `Redux`
4. `Zustand`
5. `TanStack Query`
6. `Ant Design`
7. `MUI`
8. `Chakra UI`

原因：

1. 当前目标是快速建立补充入口，不是重建整套前端架构
2. 技术栈越重，迁移成本和维护噪音越大
3. 当前页面规模和交互复杂度不需要这些额外抽象

## `apps/web` 推荐依赖清单

建议将依赖分为三类。

### 1. 核心运行时依赖

1. `react`
2. `react-dom`
3. `react-router-dom`

### 2. 开发与构建依赖

1. `typescript`
2. `vite`
3. `@vitejs/plugin-react`
4. `vite-plugin-pwa`

### 3. 样式与构建辅助依赖

1. `tailwindcss`
2. `postcss`
3. `autoprefixer`

如果后续接入现有 lint / format 体系，可再按仓库统一规范补充，但第一阶段不把工具链扩张作为重点任务。

## `apps/web` 推荐目录结构

建议目录结构如下：

```text
apps/web/
  package.json
  tsconfig.json
  vite.config.ts
  index.html
  public/
    manifest.webmanifest
    icons/
  src/
    main.tsx
    App.tsx
    app/
      routes.tsx
      providers.tsx
    pages/
      providers-page.tsx
      models-page.tsx
      logs-page.tsx
      settings-page.tsx
      tools-page.tsx
    components/
    services/
      api.ts
    bridge/
      platform-bridge.ts
    hooks/
    i18n/
    theme/
    types/
    assets/
    styles/
      globals.css
```

目录约束：

1. `pages/` 放页面级实现
2. `components/` 放通用组件
3. `services/api.ts` 作为统一 API 访问入口
4. `bridge/` 只放 Web 与 Electron 的薄适配层
5. `styles/globals.css` 保持尽量薄，主样式用 `Tailwind CSS`
6. `public/manifest.webmanifest` 和图标资源用于后续 PWA 安装能力

## Tailwind CSS 使用约束

样式系统明确采用 `Tailwind CSS`，并遵守以下约束：

1. 页面样式以 `Tailwind` utility class 为主
2. 设计 token 通过少量 CSS variables 管理
3. 尽量避免重新引入一套复杂 class helper 样式系统
4. 仅保留必要的全局样式文件
5. 不使用大型第三方 UI 样式框架

推荐做法：

1. 用 `globals.css` 接入 Tailwind base/components/utilities
2. 在 `:root` 中定义颜色、圆角、阴影等 token
3. 用 Tailwind 完成绝大多数布局、间距、排版和状态样式

## 总体实施顺序

整体按四个阶段推进：

1. Phase 0：基线整理与开发准备
2. Phase 1：创建 `apps/web` 最小可运行版本
3. Phase 2：将工具一键配置能力迁移到 Go core
4. Phase 3：让桌面端与 Web 端收敛到统一 API 和部分共享页面能力
5. Phase 4：补充 PWA 能力与部署文档

执行上必须遵守这个顺序：

1. 先建 Web 入口
2. 再迁移工具能力
3. 再让桌面端切到新 API
4. 最后补 PWA 和部署体验

不要反过来先做 PWA 或先大规模抽共享包。

## Phase 0：基线整理与开发准备

状态：`已完成`

## 目标

为后续开发建立稳定基线，避免边开发边反复改方向。

## 任务

1. 确认三份架构文档已冻结为当前版本
2. 确认根目录 `pnpm workspace` 结构可支持新增 `apps/web`
3. 确认 Electron 与 Go core 当前联调方式
4. 确认现有 renderer 页面和 API service 的依赖边界
5. 确认工具页当前依赖的 Electron bridge 能力列表

## 产出

1. 本开发计划文档
2. 一份实际开发时使用的任务 checklist

## 验收标准

1. 团队对主原则没有歧义
2. 不再讨论“Web 是否替代 Electron”
3. 后续代码开发均以本计划为准

完成说明：

1. 已完成当前仓库基线盘点
2. 已确认 `apps/web` 可通过现有 workspace 结构接入
3. 已确认桌面端 renderer 的页面、i18n、theme、API 和样式可作为 Phase 1 的复用基础
4. 已确认工具页当前仍依赖 Electron bridge，这部分保留到 Phase 2 / Phase 3 迁移

## Phase 1：创建 `apps/web` 最小可运行版本

状态：`已完成`

## 目标

先建立一个可运行的 Web 管理端入口，服务 WSL 和 Linux server 用户。

第一阶段目标不是追求最优架构，而是尽快跑通最小闭环。

## 任务拆分

### 1. 创建前端工程骨架

任务：

1. 新建 `apps/web`
2. 配置 `package.json`
3. 配置 `tsconfig.json`
4. 配置 `vite.config.ts`
5. 配置 `index.html`
6. 配置基础 `main.tsx`

验收标准：

1. `pnpm --filter web dev` 可以启动
2. 浏览器可访问基础页面

### 2. 建立基础应用结构

任务：

1. 创建 `src/App.tsx` 或 `src/app/app.tsx`
2. 创建基础布局和导航
3. 接入现有主题能力
4. 接入现有国际化能力
5. 为后续页面迁移预留 `pages/`、`services/`、`types/`、`theme/`、`i18n/`

验收标准：

1. Web 端有基础导航壳
2. 样式风格与当前桌面端保持一致或接近
3. 不依赖 Electron 也能正常渲染

### 3. 建立 API 访问基线

任务：

1. 在 `apps/web` 中建立统一 `api.ts`
2. 支持同源 API 访问
3. 支持开发环境通过配置指定 API 地址
4. 保留后续对 Electron 注入 `apiBase` 的兼容能力

验收标准：

1. Web 端可以直接请求 Go core API
2. 开发时可以切到本地运行的 core 地址

### 4. 迁移基础页面

优先顺序：

1. `Providers`
2. `Models`
3. `Logs`
4. `Settings`

任务：

1. 迁移页面结构
2. 迁移依赖的类型定义
3. 迁移依赖的样式和 UI 工具函数
4. 去除页面对 `window.desktopBridge` 的直接依赖

验收标准：

1. 这四个页面可在 Web 端独立访问
2. 能通过 HTTP API 完成核心读写
3. 页面不因缺少 Electron 能力而崩溃

### 5. 明确 Web 端定位文案

任务：

1. 页面和文档中明确 Web 端用于 WSL 和 Linux server
2. 不出现“Web 是主入口”或“桌面端将被替代”的表述

验收标准：

1. 页面文案不会误导产品定位
2. 评审时可清晰区分桌面主入口和 Web 补充入口

## Phase 1 验收标准

1. `apps/web` 存在并可独立运行
2. 浏览器中可访问 Providers / Models / Logs / Settings
3. Web 端通过 HTTP API 与 Go core 通信
4. Electron 不参与时，WSL 和 Linux server 也有可用管理入口
5. 产品定位没有偏移

完成说明：

1. 已创建 `apps/web`
2. 已确定并落地 `React + TypeScript + Vite + React Router + Tailwind CSS + vite-plugin-pwa`
3. 已建立 Web 端基础布局、路由、主题、国际化和 API 基线
4. 已迁移 `Providers / Models / Logs / Settings / Tools` 的最小可运行版本
5. `Tools` 页当前在 Web 端先以引导模式工作，一键配置将在 Phase 2 由 Go core API 接管
6. 已通过 `pnpm --filter web build`

## Phase 2：将工具一键配置能力迁移到 Go core

状态：`已完成`

## 目标

让工具一键配置能力脱离 Electron 宿主机限制，天然适配：

1. 本地桌面环境
2. WSL 内运行 core 的环境
3. Linux server 运行 core 的环境

## 任务拆分

### 1. 新建 `internal/tooling` 模块

建议文件：

```text
core/internal/tooling/
  model.go
  service.go
  codex.go
  claude.go
  backup.go
  fs.go
  detect.go
```

任务：

1. 定义工具 ID 和状态模型
2. 定义统一 service 入口
3. 实现基础文件读写封装
4. 实现基础可执行文件探测
5. 实现运行环境识别

验收标准：

1. `internal/tooling` 可独立编译
2. 模块边界清晰，不依赖 Electron

### 2. 迁移 Codex CLI 配置逻辑

任务：

1. 迁移 `~/.codex/config.toml` 写入逻辑
2. 迁移 `~/.codex/auth.json` 写入逻辑
3. 迁移配置检测逻辑
4. 迁移备份与恢复逻辑

验收标准：

1. Go core 可独立完成 Codex CLI 检测
2. Go core 可独立完成 Codex CLI 配置
3. Go core 可独立完成 Codex CLI 恢复

### 3. 迁移 Claude Code 配置逻辑

任务：

1. 迁移 `~/.claude/settings.json` 写入逻辑
2. 迁移环境变量写入逻辑
3. 迁移 active provider `claude_code_model_map` 同步逻辑
4. 迁移备份与恢复逻辑

验收标准：

1. Go core 可独立完成 Claude Code 检测
2. Go core 可独立完成 Claude Code 配置
3. Go core 可独立完成 Claude Code 恢复

### 4. 增加工具相关 API

任务：

1. 在 `core/internal/api/router.go` 增加 `GET /api/tools`
2. 增加 `POST /api/tools/{id}/configure`
3. 增加 `POST /api/tools/{id}/restore`
4. 增加 `GET /api/runtime`

验收标准：

1. API 路由可用
2. 错误码和错误信息稳定
3. Web 端可以直接调用

### 5. 增加测试

任务：

1. 为工具状态判断增加测试
2. 为配置写入增加测试
3. 为备份恢复增加测试
4. 为 router 增加基础接口测试

验收标准：

1. 核心工具写入逻辑具备自动化测试
2. 新增 API 至少有基础路由覆盖

## Phase 2 验收标准

1. Linux 环境直接运行 core 时可以一键配置 Codex CLI
2. Linux 环境直接运行 core 时可以一键配置 Claude Code
3. WSL 内运行 core 时，配置文件落在 WSL 用户目录
4. Electron 和 Web 都可以通过统一 API 使用工具能力
5. Electron 仍然是本地桌面用户进入工具能力的主入口

完成说明：

1. 已新增 `core/internal/tooling`
2. 已将 Codex CLI 与 Claude Code 的检测、配置、备份、恢复逻辑迁移到 Go core
3. 已新增 `/api/tools`
4. 已新增 `/api/tools/{id}/configure`
5. 已新增 `/api/tools/{id}/restore`
6. 已新增 `/api/runtime`
7. 已通过 `go test ./...`

## Phase 3：让桌面端与 Web 端收敛到统一 API 和部分共享页面能力

状态：`已完成`

## 目标

让桌面端不再依赖 Electron 主进程私有工具实现，而改为复用 Go core API。

## 任务拆分

### 1. 改造 `apps/web` 的 Tools 页面

任务：

1. 基于 `/api/tools` 渲染工具状态
2. 基于 `/api/tools/{id}/configure` 执行一键配置
3. 基于 `/api/tools/{id}/restore` 执行恢复
4. 基于 `/api/runtime` 展示运行环境提示

验收标准：

1. Tools 页在 Web 端可完整工作
2. 不依赖 Electron bridge 进行工具配置

### 2. 改造 `apps/desktop` 的 Tools 页面

任务：

1. 将桌面端 Tools 页面改为调用 Go core API
2. 保留命令复制、外链打开等少量桌面能力
3. 减少或移除对主进程工具配置逻辑的依赖

验收标准：

1. 桌面端 Tools 页与 Web 端共用同一套工具能力语义
2. 桌面端用户体验不回退

### 3. 收敛共享类型和共享 UI

任务：

1. 抽公共 `types`
2. 抽公共 `ui` 或样式工具
3. 清理桌面端和 Web 端重复实现

注意：

1. 这一阶段做“够用的抽取”
2. 不追求一次性大规模重构

验收标准：

1. 共享代码比复制代码更多
2. 桌面和 Web 两套页面不会继续发散

### 4. 逐步废弃 Electron 主进程中的旧工具实现

任务：

1. 标记 `apps/desktop/src/main/tool-integrations.ts` 为过渡实现
2. 在桌面端新链路稳定后移除旧入口调用
3. 最终仅保留桌面壳相关 bridge 能力

验收标准：

1. 工具配置核心逻辑不再依赖 Electron 主进程
2. Electron 主进程职责明显收缩到桌面壳

## Phase 3 验收标准

1. Web 端和桌面端都通过 Go core API 管理工具接入
2. 桌面端工具能力不回退
3. Electron 主进程不再是工具配置能力中心
4. 页面复用增加，但 Electron 主入口定位不变

完成说明：

1. Web 端 `Tools` 页已切到 Go core `/api/tools` 和 `/api/runtime`
2. 桌面端 `Tools` 页已切到 Go core `/api/tools` 和 `/api/runtime`
3. 桌面端与 Web 端的工具能力语义已统一
4. Electron 主进程旧工具实现已经从主调用链路移除，并完成过渡代码清理
5. 已通过 `pnpm --filter web build`
6. 已通过 `pnpm --filter desktop typecheck`

## Phase 4：补充 PWA 能力与部署文档

状态：`已完成`

## 目标

为 WSL 和 Linux server 用户提供更接近应用体验的 Web 入口，但不改变产品主次关系。

## 任务拆分

### 1. 增加 PWA 基础能力

任务：

1. 增加 Web App Manifest
2. 增加基础图标资源
3. 配置可安装能力
4. 配置静态资源缓存策略

验收标准：

1. Web 端可被安装为 PWA
2. PWA 启动形态可用

### 2. 明确 PWA 边界

任务：

1. 在文档中明确 PWA 只是安装形态
2. 在界面说明中明确它不负责本地 core 生命周期
3. 不引入任何试图模拟 Electron 托盘、自启动、进程拉起的能力

验收标准：

1. PWA 定位明确
2. 不会误导用户把 PWA 当桌面端替代品

### 3. 补充部署文档

任务：

1. 补充 WSL 启动文档
2. 补充 Linux server 启动文档
3. 补充 Web 访问方式说明
4. 补充 PWA 安装说明

验收标准：

1. WSL 用户可按文档完成安装和访问
2. Linux server 用户可按文档完成运行和访问

## Phase 4 验收标准

1. WSL 和 Linux server 用户可以通过浏览器或 PWA 使用管理界面
2. PWA 体验成立，但不承担 Electron 职责
3. 文档清晰说明桌面主入口和 Web 补充入口的关系

完成说明：

1. `apps/web` 已具备 `manifest`、图标和 `vite-plugin-pwa`
2. Web 构建产物已包含 PWA 所需文件
3. 已补充公开使用文档，明确 WSL、Linux server 和 PWA 的使用方式
4. 已在文档中再次明确 `Electron` 是本地主入口，`Web / PWA` 只是补充入口

## 实际开发顺序

为了避免交叉依赖，实际开发按以下顺序执行：

1. 建立 `apps/web` 工程骨架
2. 迁移 `Providers / Models / Logs / Settings`
3. 在 Go core 中建立 `internal/tooling`
4. 实现 `/api/tools` 和 `/api/runtime`
5. 迁移 Codex CLI 配置逻辑
6. 迁移 Claude Code 配置逻辑
7. 在 `apps/web` 中完成 Tools 页
8. 在 `apps/desktop` 中切换 Tools 页到新 API
9. 收敛共享类型与共享 UI
10. 清理 Electron 主进程旧工具逻辑
11. 增加 PWA 基础能力
12. 补部署与使用文档

## 禁止事项

开发过程中禁止以下行为：

1. 以“复用更方便”为由把 Electron 降级成 Web 壳
2. 以“实现更快”为由继续把新工具能力放回 Electron 主进程
3. 在未完成 Go core API 收敛前先做复杂共享包大重构
4. 在未完成 Web 最小闭环前先做 PWA 细节打磨
5. 为了 WSL 兼容重新引入跨宿主机目标抽象

## 风险与应对

## 1. 产品定位跑偏

风险：

实现中容易把“页面复用”误写成“Web 主导”。

应对：

1. 所有阶段文档和 PR 描述都必须重申主原则
2. 涉及入口描述的 UI 文案必须评审

## 2. Web 与桌面端双份实现继续发散

风险：

`apps/web` 建起来后，`apps/desktop` 继续独立发展。

应对：

1. 从 Phase 3 开始强制桌面端切到统一 API
2. 优先抽共享类型和共享 UI

## 3. 工具迁移过程中桌面端体验回退

风险：

桌面端原有一键配置功能在迁移期间失效。

应对：

1. 迁移期间保留旧实现作为过渡
2. 新链路稳定后再删除旧逻辑

## 4. 一开始重构过大

风险：

同时做 Web 化、共享抽包、状态重构、PWA，会导致节奏失控。

应对：

1. 严格按阶段推进
2. 每阶段只完成该阶段目标

## 每阶段完成定义

一个阶段只有在满足以下条件后才算完成：

1. 代码已提交到当前工作分支
2. 对应验收标准全部满足
3. 相关文档已同步更新
4. 不引入新的产品定位歧义

## 最终交付标准

整个计划完成后，项目应达到以下状态：

1. 本地桌面用户仍以 Electron 为主入口
2. WSL 用户可通过浏览器或 PWA 管理运行在 WSL 内的 core
3. Linux server 用户可通过浏览器或 PWA 管理运行在 server 上的 core
4. 工具一键配置能力由 Go core 统一提供
5. Electron 与 Web 在不改变主次关系的前提下复用部分页面能力
6. 产品方向不会因为 Web/PWA 的加入而跑偏
