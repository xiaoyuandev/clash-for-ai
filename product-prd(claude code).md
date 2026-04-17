# Clash for AI 产品方案 v2 （Claude Code Max产出）

## 1. 产品概述

### 1.1 一句话定义

Clash for AI 是一个面向 AI 重度用户的本地工具，用于管理多个中转 API 服务商，提供一键切换、测速、余额查询和统一本地接入能力。

### 1.2 产品本质

一个中转 API 的管理面板 + 一个本地反向代理。

用户在各类 AI 工具中只需配置一次 `http://localhost:3456/v1`，之后所有中转商的切换、状态查看、故障排查都在 Clash for AI 中完成，无需反复修改工具配置。

### 1.3 关键前提

本产品对接的上游是**中转 API 服务商**，不是官方模型厂商（OpenAI、Anthropic、Google 等）。

这一前提决定了产品形态：

1. 所有上游已经同时支持 OpenAI-compatible 和 Anthropic-compatible 接口，不需要做协议适配。
2. 本地 Gateway 本质是反向代理 + Auth 注入，不是协议转换器。
3. 代理透传所有 `/v1/*` 路径，不解析也不转换请求体，天然兼容不同协议格式。
4. 产品核心价值在管理和可观测，不在底层协议兼容。

```
实际请求链路（以两种典型工具为例）：

Claude Code:
  POST http://localhost:3456/v1/messages (Anthropic 格式)
  ↓ 反向代理：URL 重写 + Auth 注入，透传请求体
  POST https://api.provider-a.com/v1/messages
  ↓ 中转商内部转发 → Anthropic

Cursor / Chatbox:
  POST http://localhost:3456/v1/chat/completions (OpenAI 格式)
  ↓ 反向代理：URL 重写 + Auth 注入，透传请求体
  POST https://api.provider-a.com/v1/chat/completions
  ↓ 中转商内部转发 → OpenAI / Anthropic / Google ...
```

关键设计决策：代理不关心请求体的格式是 OpenAI 还是 Anthropic，只负责转发。协议兼容性由中转商保证，不是本产品的职责。

### 1.4 要解决的核心问题

| # | 问题 | 当前状况 |
|---|------|----------|
| 1 | 切换中转商成本高 | 每次切换要改多个工具的 base_url 和 api_key |
| 2 | 故障归因困难 | 请求失败后不知道是额度用完、鉴权失败还是上游挂了 |
| 3 | 状态不透明 | 不知道当前中转商的延迟、余额、可用性 |
| 4 | 配置管理混乱 | 多个中转商的 key 散落在各个工具的配置文件中 |

### 1.5 产品目标

1. 工具只配一次，后续切换在 UI 中完成。
2. 请求失败时，用户能立刻知道原因。
3. 用户能一目了然地看到所有中转商的状态。

## 2. 目标用户

### 2.1 核心用户

同时使用两家及以上中转 API 服务商的开发者和 AI 重度用户。

典型场景：

- 主力用中转商 A，但 A 时不时抽风，需要临时切到 B。
- A 的额度快用完了，先切到 B 顶一下。
- 新发现了中转商 C，想测一下速度和稳定性再决定是否迁移。
- 在 Claude Code、Cursor、Chatbox 等多个工具间切换，每次都要改配置。

### 2.2 不服务的用户

1. 只用单一官方 API（如直连 OpenAI）且无切换需求的用户。
2. 需要企业级权限、审计、云端协作的团队。
3. 期望系统级 VPN / 网络代理的用户。

## 3. 竞品分析

### 3.1 vs cc-switch

cc-switch 通过修改环境变量或工具配置文件来切换服务商。

| 维度 | cc-switch | Clash for AI |
|------|-----------|--------------|
| 切换方式 | 改配置文件 / 环境变量 | UI 一键切换，工具无感 |
| 对工具的侵入 | 每次切换都改工具配置 | 工具只配一次 localhost |
| 可观测性 | 无 | 测速、余额、错误分类、请求日志 |
| 多工具支持 | 每个工具单独切换 | 所有工具统一切换 |
| 使用门槛 | CLI，需理解配置结构 | Web UI，直观操作 |

核心差异：cc-switch 是配置改写工具，Clash for AI 是本地代理 + 管理面板。前者每次切换都要动工具配置，后者让工具配置保持不变。

### 3.2 vs One API / New API

One API 是一个 API 管理和分发平台，主要面向自建中转服务。

| 维度 | One API | Clash for AI |
|------|---------|--------------|
| 定位 | 服务端 API 管理平台 | 本地桌面工具 |
| 上游 | 对接官方模型厂商 | 对接中转 API 服务商 |
| 部署 | 需要服务器 | 本地运行，下载即用 |
| 目标用户 | 中转站运营者 | 中转站的消费者 |
| 核心能力 | 渠道管理、令牌分发、配额控制 | Provider 切换、状态观测 |

核心差异：One API 是给中转站运营者用的后台，Clash for AI 是给中转站消费者用的管理工具。两者处于产业链的不同位置。

### 3.3 vs 手动管理

大多数用户当前的做法：在 Notion 或备忘录里记着各家中转商的 base_url 和 key，需要切换时手动复制粘贴到工具配置文件中。

Clash for AI 的价值就是把这个过程自动化，并加上状态可视化。

## 4. 产品边界

### 4.1 做什么

1. 管理多个中转 API 服务商（增删改查）。
2. 提供本地统一 API 入口（反向代理）。
3. 一键切换当前生效的中转商。
4. 测速和连通性检测。
5. 余额查询（支持的中转商）。
6. 请求日志和错误分类。

### 4.2 不做什么

1. 不做协议适配（上游已经是 OpenAI-compatible）。
2. 不做系统级网络代理。
3. 不做云端 API 中转平台。
4. 不做自动路由和智能 fallback（首版）。
5. 不做订阅协议和订阅生态（首版）。
6. 不做工具配置接入助手（首版）。

## 5. 产品架构

### 5.1 整体架构

```
┌─────────────────────────────────────────────┐
│                 Web UI                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────────┐ │
│  │ Provider │ │ 状态面板  │ │  请求日志    │ │
│  │   管理   │ │ 测速/余额 │ │  错误分类    │ │
│  └──────────┘ └──────────┘ └──────────────┘ │
└──────────────────┬──────────────────────────┘
                   │ HTTP API
┌──────────────────┴──────────────────────────┐
│              Go 后端服务                      │
│  ┌──────────┐ ┌──────────┐ ┌──────────────┐ │
│  │ Provider │ │ 反向代理  │ │  观测与日志  │ │
│  │ 数据管理 │ │   引擎   │ │    采集      │ │
│  └──────────┘ └──────────┘ └──────────────┘ │
│  ┌──────────┐ ┌──────────┐                   │
│  │ 测速服务 │ │ 余额查询 │                   │
│  └──────────┘ └──────────┘                   │
└──────────────────┬──────────────────────────┘
                   │ SQLite
              ┌────┴────┐
              │ 本地存储 │
              └─────────┘
```

### 5.2 反向代理工作原理

```
外部 AI 工具发起请求（任意 /v1/* 路径）：
POST http://localhost:3456/v1/chat/completions   ← OpenAI 格式
POST http://localhost:3456/v1/messages            ← Anthropic 格式
Headers:
  Authorization: Bearer any-token-or-empty

本地反向代理处理：
1. 接收请求（不解析请求体）
2. 查找当前选中的 Provider
3. 剥掉请求路径中的 /v1 前缀，拼接到 Provider 的 base_url
   例：base_url=https://api.example.com/v1，请求路径 /v1/messages
   → 转发目标：https://api.example.com/v1/messages
4. 替换 Auth Header 为 Provider 的真实 API Key
5. 转发请求，透传响应（包括 SSE stream）
6. 记录请求日志（耗时、状态码、错误信息）
```

**路径拼接规则：** 代理剥掉请求中的 `/v1` 前缀，将剩余路径拼到 `base_url`。`base_url` 里要不要包含 `/v1` 完全由用户决定，代理不做假设。

```
用户填 base_url = https://api.provider-a.com/v1
  请求 /v1/messages → 转发到 https://api.provider-a.com/v1/messages ✓

用户填 base_url = https://api.provider-b.com（不含 /v1）
  请求 /v1/messages → 转发到 https://api.provider-b.com/messages ✓
```

代理不关心也不解析请求体的格式，只做路径转发和鉴权注入。协议兼容性由上游中转商保证。

### 5.3 关键流程

#### 流程 A：首次接入

```
1. 用户打开 Clash for AI
2. 添加中转商：填写 name、base_url、api_key
3. 点击测试连通性
4. 选为当前生效的 Provider
5. 在 AI 工具中配置 base_url = http://localhost:3456/v1
6. 完成。以后不需要再动工具配置
```

#### 流程 B：切换中转商

```
1. 用户打开 Clash for AI
2. 在 Provider 列表中选择另一家
3. 点击「切换」
4. 完成。所有工具立即走新的中转商
```

#### 流程 C：故障排查

```
1. 用户发现 AI 工具报错
2. 打开 Clash for AI 查看请求日志
3. 看到错误分类：鉴权失败 / 额度不足 / 上游 5xx / 超时
4. 根据原因决定是换 Key、充值还是切换中转商
```

## 6. 核心功能设计

产品只做三件事，做到极致。

### 6.1 Provider 管理

#### 功能

1. 添加中转商（name、base_url、api_key）。
2. 编辑和删除中转商。
3. 一键切换当前生效的中转商。
4. 查看每个中转商下的可用模型列表（调用 `/v1/models` 获取）。

#### Provider 数据结构

```json
{
  "id": "uuid",
  "name": "中转商 A",
  "base_url": "https://api.provider-a.com/v1",
  "api_key": "sk-xxx",
  "is_active": false,
  "models": [],
  "created_at": "2026-04-17T12:00:00Z",
  "updated_at": "2026-04-17T12:00:00Z"
}
```

说明：

- `api_key` 在本地数据库中加密存储，UI 展示时脱敏为 `sk-****1234`。
- `is_active` 全局只有一个 Provider 为 true（首版）。
- `models` 通过调用上游 `/v1/models` 接口自动获取，也允许手动编辑。

#### 交互要点

- 添加 Provider 后自动触发连通性测试。
- 切换 Provider 立即生效，无需重启。
- 删除当前生效的 Provider 时，要求先切换到其他 Provider。
- `base_url` 输入框提示：`通常以 /v1 结尾，例如 https://api.example.com/v1`。若不以 `/v1` 结尾，展示 warning 提示用户确认，但不阻止保存。

### 6.2 本地反向代理

#### 功能

1. 监听本地端口（默认 `3456`），暴露 `http://localhost:3456/v1`。
2. 将请求转发到当前生效的中转商。
3. 替换 Auth Header。
4. 透传 SSE stream。
5. 记录每条请求的日志。

#### 转发策略（首版）

代理转发所有 `/v1/*` 路径到上游，不限定具体接口。这意味着：

- `/v1/chat/completions`（OpenAI 格式）—— Cursor、Chatbox 等工具使用
- `/v1/messages`（Anthropic 格式）—— Claude Code 等工具使用
- `/v1/models`（模型列表）—— 通用
- 其他上游支持的路径 —— 自动兼容

代理不解析请求体，不判断协议格式，只做三件事：**URL 重写、Auth 注入、透传**。能支持什么接口，取决于上游中转商，不取决于本产品。

#### 端口配置

- 默认端口 `3456`，用户可自定义。
- 启动时检测端口占用，冲突时提示用户。

#### Auth 处理策略

外部工具发来的请求可能带任意 token 或不带 token，代理统一替换为当前 Provider 的真实 API Key。这意味着用户在工具中配置的 api_key 可以是任意值（如 `sk-placeholder`），实际鉴权由 Clash for AI 管理。

注意：不同协议的 Auth Header 格式可能不同（OpenAI 用 `Authorization: Bearer`，Anthropic 用 `x-api-key`）。首版统一使用 `Authorization: Bearer` 注入，大多数中转商对两种格式都接受。如果遇到特殊中转商，后续可在 Provider 配置中增加 Auth 格式选项。

### 6.3 可观测性

#### 6.3.1 测速

功能：

- 对指定 Provider 发起测试请求，测量延迟。
- 支持批量测速（所有 Provider 同时测）。

测量指标：

| 指标 | 说明 |
|------|------|
| 连通性 | 是否能连上 |
| 首响应延迟 | 从发送请求到收到第一个字节 |
| 首 Token 延迟 | 从发送请求到收到第一个 SSE 事件 |
| 完整响应耗时 | 从发送请求到响应结束 |

测试方法：向 `/v1/chat/completions` 发送固定的短请求（如 `"Reply with OK"`，`max_tokens: 8`）。

#### 6.3.2 余额查询

现实约束：中转商的余额接口不统一。

支持分层：

| 级别 | 说明 | 处理方式 |
|------|------|----------|
| 一级 | 中转商提供标准余额接口 | 自动查询，UI 展示 |
| 二级 | 中转商有余额接口但格式特殊 | 通过适配器支持 |
| 三级 | 无余额接口 | UI 标记为「不支持」，不误导用户 |

首版适配 2-3 家主流中转商的余额接口即可。

#### 6.3.3 请求日志

记录通过本地代理的每条请求：

```json
{
  "id": "uuid",
  "timestamp": "2026-04-17T12:00:00Z",
  "provider_id": "uuid",
  "provider_name": "中转商 A",
  "method": "POST",
  "path": "/v1/chat/completions",
  "model": "gpt-4.1",
  "status_code": 200,
  "latency_ms": 1230,
  "error_type": null,
  "error_message": null,
  "stream": true
}
```

- 只记录元数据，不记录请求/响应 body（隐私和存储考虑）。
- 本地保留最近 1000 条，超出自动清理。
- 支持按 Provider、状态码、错误类型筛选。

#### 6.3.4 错误分类

请求失败时，根据响应进行分类：

| 错误类型 | 判断依据 | 用户可读描述 |
|----------|----------|-------------|
| `auth_error` | 401 / 403 | API Key 无效或已过期 |
| `quota_exhausted` | 429 + 余额相关信息 | 额度已用完 |
| `model_not_found` | 404 + model 相关信息 | 该中转商不支持此模型 |
| `rate_limited` | 429 | 请求频率超限，稍后重试 |
| `upstream_error` | 5xx | 中转商或上游服务异常 |
| `timeout` | 请求超时 | 请求超时，可能是网络问题 |
| `network_error` | 连接失败 | 无法连接到中转商 |

UI 目标：用户看到错误后，能立刻知道该怎么做（换 Key？充值？切换中转商？等一会儿重试？）。

## 7. 技术选型

### 7.1 推荐方案：Go 核心服务 + Electron 外壳

```
Electron 外壳进程
├── 原生窗口（应用主界面）
├── 系统托盘（macOS menu bar / Windows system tray）
├── 开机自启管理
├── Go 子进程生命周期管理（启动、健康检查、重启）
└── Web UI（React/Vue，运行在 Electron renderer）
      ↕ HTTP API（本地 localhost）
Go 核心服务（子进程）
├── 反向代理引擎（net/http/httputil）
├── Provider 数据管理 API
├── 测速 / 余额查询服务
├── 请求日志采集
└── SQLite（本地数据存储）
```

### 7.2 分层职责

| 层 | 技术 | 职责 |
|----|------|------|
| 外壳层 | Electron | 原生窗口、系统托盘、开机自启、进程管理 |
| UI 层 | React + Electron renderer | 所有界面和交互 |
| 核心服务层 | Go | 反向代理、数据管理、网络请求 |
| 存储层 | SQLite | 本地持久化 |

**原则：业务逻辑全部在 Go 层，Electron 只负责壳和 UI 渲染，不承担任何业务逻辑。**

### 7.3 选型理由

| 维度 | Go + Electron | 纯 Electron | Tauri | Go 单二进制 |
|------|--------------|-------------|-------|------------|
| 桌面体验 | 好 | 好 | 好 | 无原生窗口 |
| 反向代理能力 | Go 标准库，成熟稳定 | Node.js 可做但不是强项 | Rust 可做 | Go 标准库 |
| 开发速度 | 中 | 快 | 慢（Rust） | 快 |
| 包体大小 | ~100MB | ~150MB | ~10MB | ~15MB |
| 核心逻辑可维护性 | 高（Go 类型安全） | 中（JS） | 高（Rust） | 高（Go） |
| 开机自启 / 托盘 | Electron 原生支持 | 原生支持 | 原生支持 | 需第三方库 |

选择 Go + Electron 的核心理由：Go 是写网络代理最合适的语言（标准库 `net/http/httputil` 直接支持，SSE 透传简洁），Electron 提供最成熟的桌面壳体验，两者各司其职。

### 7.4 技术架构细节

#### 反向代理核心（Go）

```go
// 伪代码，展示路径剥离和转发逻辑
func proxyHandler(w http.ResponseWriter, r *http.Request) {
    provider := getCurrentActiveProvider()

    // 剥掉 /v1 前缀，拼接到 provider.BaseURL
    // 例：/v1/messages → provider.BaseURL + /messages
    path := strings.TrimPrefix(r.URL.Path, "/v1")
    target, _ := url.Parse(provider.BaseURL + path)

    // 替换 Auth Header
    r.Header.Set("Authorization", "Bearer " + provider.APIKey)

    // 转发，透传响应（含 SSE stream）
    proxy := httputil.NewSingleHostReverseProxy(target)
    proxy.ServeHTTP(w, r)

    // 异步记录日志
    go logRequest(provider, r, statusCode, latency)
}
```

#### 进程通信

Electron 主进程通过 `child_process.spawn` 启动 Go 服务，通过 HTTP 与其通信：

```
Electron main process
  → spawn go-core（传入端口、数据目录等启动参数）
  → 轮询 /health 确认启动完成
  → renderer 通过 localhost HTTP API 操作数据

Go 服务退出时，Electron 自动重启
应用退出时，Electron 负责终止 Go 子进程
```

#### 本地存储（Go 侧）

- SQLite 单文件数据库，位于系统标准应用数据目录。
- 存储：Provider 配置、请求日志、测速历史、余额缓存。
- API Key 在数据库中使用 AES-256 加密存储，密钥由本机硬件信息派生。

#### 系统托盘（Electron 侧）

托盘菜单功能：
- 显示当前活跃 Provider 名称
- 快速切换 Provider（子菜单列出所有 Provider）
- 打开主界面
- 退出

## 8. MVP 范围

### 8.1 MVP 只做 6 件事

| # | 功能 | 优先级 | 预估工作量 |
|---|------|--------|-----------|
| 1 | Provider 增删改查 | P0 | 小 |
| 2 | 本地反向代理（转发所有 /v1/* 路径 + stream） | P0 | 中 |
| 3 | 一键切换 Provider | P0 | 小 |
| 4 | 连通性测试 + 测速 | P0 | 中 |
| 5 | 请求日志 + 错误分类 | P1 | 中 |
| 6 | 余额查询（2-3 家） | P1 | 中 |

### 8.2 MVP 明确不做

1. 订阅协议和订阅导入。
2. 工具配置接入助手。
3. 自动 fallback 和智能路由。
4. 模型级别的选择和切换（首版以 Provider 为粒度）。
5. 云同步、账号系统。
6. 系统级代理。
7. 协议适配（不同中转商之间的协议差异由中转商自身解决）。

### 8.3 里程碑

#### M1：能用（2 周）

- Provider CRUD + 本地存储
- 反向代理核心（转发所有 /v1/* 路径 + SSE stream）
- 一键切换 Provider
- 最小 Web UI

交付标准：用户能添加两个中转商，在 UI 中切换，Claude Code 通过 localhost 正常工作。

#### M2：好用（+1 周）

- 连通性测试 + 批量测速
- 请求日志查看
- 错误分类展示
- 系统托盘

交付标准：用户请求失败时，能在 UI 中看到明确的错误原因。

#### M3：完整（+1 周）

- 余额查询（适配 2-3 家主流中转商）
- Provider 导入导出（JSON 文件）
- 模型列表展示（调用 /v1/models）
- UI 优化和体验打磨

交付标准：完成首批用户内测。

## 9. 成功标准

MVP 成功的衡量标准：

1. 用户能在 3 分钟内完成首次接入（添加 Provider + 配置工具）。
2. 用户切换中转商的操作耗时 < 3 秒（点一下按钮）。
3. 请求失败时，用户能在 10 秒内找到原因。
4. 用户接入后不再需要修改 AI 工具的配置。
5. 至少有 10 个真实用户愿意持续使用。

## 10. 风险与应对

### 风险 1：用户不愿意改一次工具配置

有些用户可能连"把 base_url 改成 localhost"这一步都觉得麻烦。

应对：
- 提供清晰的接入文档，覆盖主流工具（Claude Code、Cursor、Chatbox 等）。
- 首版不做自动接入，用文档解决。验证需求后再考虑自动化。

### 风险 2：中转商余额接口不统一

应对：
- 余额功能分层支持，有接口的展示，没接口的标记"不支持"。
- 不把余额做成核心卖点，它是锦上添花。

### 风险 3：差异化不够强

用户可能觉得"我手动切一下也不麻烦"。

应对：
- 核心差异化不只是"切换方便"，而是**可观测性**——测速、错误分类、请求日志。
- 这些能力是手动管理完全不具备的。
- 产品宣传应强调"看得见"而不只是"切得快"。

### 风险 4：本地端口被占用或被安全软件拦截

应对：
- 允许自定义端口。
- 启动时检测端口冲突，给出明确提示。
- 提供常见安全软件的放行指引。

### 风险 5：SSE stream 转发的兼容性

部分中转商的 SSE 实现可能有细微差异。

应对：
- 反向代理层对 SSE 做最小处理，尽量透传原始数据。
- 首批适配时覆盖 2-3 家主流中转商，确认 stream 兼容。

## 11. 后续版本方向

以下功能只在 MVP 验证用户价值后推进。

### Phase 2：增强管理能力

- Provider 分组和标签。
- 模型级别的选择和切换。
- 收藏模型。
- Provider 导入：支持从 URL 导入配置。
- 更多中转商的余额适配。

### Phase 3：智能化

- 自动 fallback：当前 Provider 失败时自动切到备选。
- 基于延迟的自动选择。
- 测速历史趋势图。
- 更完整的请求分析面板。

### Phase 4：生态

- 订阅协议：定义标准格式，让中转商可以提供一键导入。
- 工具接入助手：自动帮用户配置常见 AI 工具。
- 社区 Provider 分享。
- 配置云端备份。

---

## 附录 A：与 v1 PRD 的主要变更

| 维度 | v1 | v2 |
|------|----|----|
| 上游定位 | 未明确，暗含官方厂商 | 明确为中转 API 服务商 |
| 架构层级 | 四层（含执行配置层、适配层） | 两层（管理面板 + 反向代理） |
| Gateway 定位 | 协议转换器 | 纯反向代理 |
| 订阅协议 | 首版必做，定义了完整 Schema | 延后到 Phase 4 |
| 接入助手 | 首版必做 | 延后，用文档替代 |
| MVP 范围 | 10 项必做 | 6 项必做 |
| 技术选型 | 未指定 | Go 核心服务 + Electron 外壳 |
| 竞品分析 | 无 | 新增 vs cc-switch / One API / 手动管理 |
