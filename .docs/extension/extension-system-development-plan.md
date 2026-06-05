# Relay Switch Extension System Development Plan

状态：Milestone 1 已实现
日期：2026-06-05
分支：`feature/extension-system`
目录：`.docs/extension/`
设计来源：`.docs/extension/relay-switch-extension-system-design.md`

## 1. 目标

本文档用于把插件系统总设计拆解为可执行开发计划。目标是在不打乱 Relay Switch 现有核心能力的前提下，逐步实现插件系统基础设施，并为后续三个样板插件做好承载能力：

1. Markdown Archive Plugin。
2. RTK Integration Plugin。
3. Protocol Bridge Runtime Plugin。

插件系统第一阶段不追求完整插件市场，也不开放所有高风险能力。开发重点是先建立稳定边界：

```text
Manifest Loader
Plugin Registry
Plugin State API
Settings / Commands
Audit Logs
Tool Integration Contribution
Transcript / Exporter Contribution
Runtime / Protocol Bridge Contribution 预留
```

## 2. 当前项目切入点

### 2.1 Core

当前 core 主要模块：

```text
core/internal/api
core/internal/app
core/internal/config
core/internal/credential
core/internal/gateway
core/internal/localgateway
core/internal/logging
core/internal/provider
core/internal/tooling
core/internal/storage
```

插件系统建议新增：

```text
core/internal/extension
core/internal/extension/manifest
core/internal/extension/registry
core/internal/extension/runtime
core/internal/extension/audit
core/internal/extension/broker
core/internal/extension/contrib
```

第一版可以先合并为较少包，避免过早拆分：

```text
core/internal/extension
core/internal/extension/manifest
core/internal/extension/audit
```

### 2.2 UI

当前前端有 Web 和 Desktop renderer 两套：

```text
apps/web/src
apps/desktop/src/renderer/src
```

新增 UI 时需要同步两边，沿用现有页面模式。

建议新增：

```text
PluginsPage
PluginDetailPage
PluginSettingsForm
PluginAuditPanel
```

第一版可以只做 `PluginsPage` 和基础详情，不做复杂插件页面渲染。

### 2.3 配置和运行时

当前 runtime 有 core 和 local gateway。插件系统需要新增插件扫描路径和启用状态。

默认 user scope：

```text
~/.relay-switch/extensions/{plugin-id}/relay-switch-plugin.json
```

bundled scope 后续可从桌面资源目录读取。

## 3. 开发原则

1. 不影响现有 Provider、Gateway、Models、Tools 使用。
2. 默认没有插件时系统行为不变。
3. 高风险能力先只定义，不开放执行。
4. 插件禁用必须立即生效。
5. Provider API Key 不进入插件 runtime。
6. 所有插件敏感操作走 audit log。
7. 测试覆盖 manifest validation、registry state、API contract。

## 4. Phase 1：Manifest Loader 和 Registry

### 4.1 目标

实现插件系统最小基础设施：

1. 扫描插件目录。
2. 解析 manifest。
3. 校验 manifest。
4. 维护插件 enable / disable 状态。
5. 对 UI 暴露插件列表 API。
6. 暂不启动插件进程。

### 4.2 Core 模块

新增：

```text
core/internal/extension/model.go
core/internal/extension/manifest.go
core/internal/extension/service.go
core/internal/extension/repository.go
core/internal/extension/sqlite_repository.go
```

模型草案：

```go
type PluginScope string

const (
    PluginScopeUser    PluginScope = "user"
    PluginScopeBundled PluginScope = "bundled"
    PluginScopeProject PluginScope = "project"
    PluginScopeManaged PluginScope = "managed"
)

type PluginStatus string

const (
    PluginStatusInstalled    PluginStatus = "installed"
    PluginStatusEnabled      PluginStatus = "enabled"
    PluginStatusDisabled     PluginStatus = "disabled"
    PluginStatusIncompatible PluginStatus = "incompatible"
    PluginStatusInvalid      PluginStatus = "invalid"
)

type Plugin struct {
    ID           string
    Name         string
    Version      string
    Description  string
    Publisher    string
    Scope        PluginScope
    ManifestPath string
    Enabled      bool
    Status       PluginStatus
    LastError    string
    Manifest     Manifest
    CreatedAt    string
    UpdatedAt    string
}
```

Manifest 草案：

```go
type Manifest struct {
    ManifestVersion int                  `json:"manifestVersion"`
    ID              string               `json:"id"`
    Name            string               `json:"name"`
    Version         string               `json:"version"`
    Description     string               `json:"description"`
    Publisher       string               `json:"publisher"`
    Engines         ManifestEngines      `json:"engines"`
    Entry           ManifestEntry        `json:"entry"`
    Contributes     ManifestContributes  `json:"contributes"`
    Permissions     []string             `json:"permissions"`
}
```

### 4.3 Manifest 校验

校验规则：

1. `manifestVersion == 1`。
2. `id` 非空，格式为 reverse domain 或 slug。
3. `name` 非空。
4. `version` 符合 semver。
5. `entry.type` 只能为 `process`、`none`。
6. `entry.command` 不能是绝对危险路径。
7. `permissions` 必须是已知权限。
8. unknown contribution point 默认保留但不执行，或标记 warning。

第一版建议 unknown contribution point 不阻止加载，但在 UI 中显示 unsupported。

### 4.4 数据库

新增 migration：

```sql
CREATE TABLE IF NOT EXISTS plugins (
  id TEXT PRIMARY KEY,
  version TEXT NOT NULL,
  manifest_path TEXT NOT NULL,
  scope TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'installed',
  last_error TEXT,
  manifest_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS plugin_settings (
  plugin_id TEXT NOT NULL,
  key TEXT NOT NULL,
  value_json TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (plugin_id, key)
);

CREATE TABLE IF NOT EXISTS plugin_grants (
  plugin_id TEXT NOT NULL,
  capability TEXT NOT NULL,
  approval_mode TEXT NOT NULL,
  scope_json TEXT NOT NULL DEFAULT '{}',
  updated_at TEXT NOT NULL,
  PRIMARY KEY (plugin_id, capability)
);
```

### 4.5 API

新增 endpoints：

```text
GET    /api/extensions
GET    /api/extensions/{id}
POST   /api/extensions/{id}/enable
POST   /api/extensions/{id}/disable
POST   /api/extensions/rescan
```

响应草案：

```json
{
  "id": "relay-switch.markdown-archive",
  "name": "Markdown Archive",
  "version": "0.1.0",
  "publisher": "relay-switch",
  "scope": "user",
  "enabled": false,
  "status": "installed",
  "last_error": "",
  "permissions": [],
  "contributes": {
    "commands": 2,
    "settings": 1,
    "pages": 1
  }
}
```

### 4.6 UI

新增 Plugins 页面：

1. 插件列表。
2. 状态 chip。
3. Enable / Disable。
4. Rescan。
5. 基础 manifest 信息。
6. 权限列表。
7. Last error。

### 4.7 测试

Core tests：

1. valid manifest loads。
2. invalid manifest rejected。
3. duplicate plugin id handling。
4. enable / disable persists。
5. unknown permission rejected。
6. unsupported contribution warning。

API tests：

1. list extensions。
2. get extension。
3. enable extension。
4. disable extension。
5. rescan。

### 4.8 验收

1. 启动 core 时扫描 user plugin 目录。
2. UI 能看到插件列表。
3. enable / disable 可保存。
4. 没有插件时现有功能不受影响。

## 5. Phase 2：Settings、Commands、Audit

### 5.1 目标

让插件具备可配置项和可执行命令，但先不启动复杂长期 runtime。

### 5.2 Commands

贡献点：

```json
{
  "id": "markdownArchive.syncNow",
  "title": "Sync Conversations Now",
  "category": "Archive"
}
```

Core 需要 registry：

```go
type CommandContribution struct {
    ID       string
    Title    string
    Category string
    PluginID string
}
```

API：

```text
GET  /api/extensions/commands
POST /api/extensions/commands/{commandId}/execute
```

第一版 command execution 可以仅支持 first-party/bundled commands 或 mock execution，等 Extension Host 完成后再真正调用插件进程。

### 5.3 Settings

插件 settings 由 JSON Schema 声明。

API：

```text
GET  /api/extensions/{id}/settings
PUT  /api/extensions/{id}/settings
```

第一版 settings form 可以用现有 UI 组件渲染常见类型：

```text
string
boolean
integer
number
enum
array string
```

### 5.4 Audit Logs

新增表：

```sql
CREATE TABLE IF NOT EXISTS plugin_audit_logs (
  id TEXT PRIMARY KEY,
  timestamp TEXT NOT NULL,
  plugin_id TEXT NOT NULL,
  plugin_version TEXT NOT NULL,
  capability TEXT NOT NULL,
  action TEXT NOT NULL,
  resource_type TEXT,
  resource_id TEXT,
  status TEXT NOT NULL,
  latency_ms INTEGER,
  approval_source TEXT,
  error_message TEXT,
  metadata_json TEXT
);
```

API：

```text
GET /api/extensions/audit-logs
GET /api/extensions/{id}/audit-logs
```

### 5.5 测试

1. command registry。
2. command execution audit。
3. settings save/load。
4. JSON Schema basic validation。
5. audit log list。

### 5.6 验收

1. 插件可以展示 settings。
2. settings 可保存。
3. commands 能显示。
4. command 执行记录 audit log。

## 6. Phase 3：Extension Host 和 Process Runtime

### 6.1 目标

启动插件进程，并通过 JSON-RPC 或 localhost HTTP 与插件通信。

### 6.2 推荐协议

第一版建议使用 JSON-RPC over stdio。

原因：

1. 不需要额外端口。
2. 更容易限制外部访问。
3. 适合 commands/settings/background task。

Protocol bridge runtime 可以单独使用 HTTP，因为它需要 streaming。

### 6.3 生命周期

状态：

```text
starting
running
degraded
crashed
stopped
```

策略：

1. enabled 才启动。
2. disable 后停止。
3. crash 后带退避重启。
4. 连续失败后标记 degraded。

### 6.4 Host API

最小 RPC：

```text
initialize
shutdown
executeCommand
getStatus
settingsChanged
```

请求示例：

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "executeCommand",
  "params": {
    "commandId": "markdownArchive.syncNow",
    "args": {}
  }
}
```

### 6.5 安全

1. 插件进程环境变量只包含必要信息。
2. 不把 Provider API Key 放进 env。
3. stdout/stderr 做大小限制。
4. RPC timeout。
5. 禁用插件后取消 in-flight command。

### 6.6 测试

1. fake plugin initialize。
2. execute command。
3. timeout。
4. crash detection。
5. disable stops process。

## 7. Phase 4：Tool Integration Extensions

### 7.1 目标

把现有工具集成能力接入 contribution registry，并为 RTK 插件做准备。

### 7.2 贡献点

```json
{
  "toolIntegrations": [
    {
      "id": "rtk",
      "title": "RTK",
      "supportsDetect": true,
      "supportsConfigure": true,
      "supportsRestore": true
    }
  ]
}
```

### 7.3 Broker

新增 Process Broker：

```text
process.exec.declared
process.readVersion
```

新增 Tool Config Broker：

```text
tool.config.read
tool.config.write
tool.config.backup
```

第一版只开放 declared commands，不开放任意 shell。

### 7.4 与现有 tooling 的关系

当前 `core/internal/tooling` 里的 Codex/Claude Code/Cursor 集成可以先保持不动。

实现方式：

1. 新 registry 可以同时登记内置 tool integrations。
2. UI 从 registry 读取 tool cards。
3. 旧 API 可逐步迁移到 extension contribution 后面。

### 7.5 验收

1. RTK 插件可以贡献 Tool card。
2. Tool card 可以执行 detect/configure/restore。
3. 执行前备份配置。
4. 操作写 audit log。

## 8. Phase 5：Transcript 和 Markdown 归档能力

### 8.1 目标

为 Markdown Archive Plugin 提供 host capability。

### 8.2 Transcript Broker

能力：

```text
tool.transcripts.read
filesystem.toolTranscriptStore
```

API：

```go
type TranscriptSource string

const (
    TranscriptSourceClaudeCode TranscriptSource = "claude-code"
    TranscriptSourceCodexCLI   TranscriptSource = "codex-cli"
)
```

Broker 职责：

1. 只读取允许的 transcript 路径。
2. 不读取 auth token 文件。
3. 支持 discover。
4. 支持 load raw session。
5. 支持 file stat。

### 8.3 Filesystem Broker

能力：

```text
filesystem.userSelectedDirectory.write
filesystem.pluginData
```

职责：

1. 用户选择输出目录。
2. 插件只能写入授权目录。
3. 原子写入。
4. path sanitizer。

### 8.4 Export Index

新增表：

```sql
CREATE TABLE IF NOT EXISTS transcript_exports (
  source TEXT NOT NULL,
  session_id TEXT NOT NULL,
  raw_path TEXT NOT NULL,
  raw_mtime INTEGER NOT NULL,
  raw_size INTEGER NOT NULL,
  output_path TEXT NOT NULL,
  exported_at TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  PRIMARY KEY (source, session_id)
);
```

### 8.5 验收

1. Claude Code session 可导出 Markdown。
2. Codex session 可导出 Markdown。
3. 输出目录可配置。
4. 增量同步跳过未变化文件。
5. 自动同步可开关。

## 9. Phase 6：Provider Actions 和 Credential Broker

### 9.1 目标

让插件可以做 Provider 余额、额度、账号状态查询，但不读取 API Key。

### 9.2 Request Templates

Manifest：

```json
{
  "providerRequestTemplates": [
    {
      "id": "openai-credit-grants",
      "purpose": "balance.check",
      "providerMatch": {
        "baseUrlHosts": ["api.openai.com"]
      },
      "method": "GET",
      "path": "/dashboard/billing/credit_grants",
      "auth": "provider"
    }
  ]
}
```

### 9.3 Credential Broker

职责：

1. 校验 plugin grant。
2. 校验 template。
3. 校验 provider host。
4. 注入凭证。
5. 过滤响应。
6. 审计。

### 9.4 验收

1. 插件无法读取 API Key。
2. 插件无法向任意 URL 注入凭证。
3. 所有 Provider request 写 audit log。

## 10. Phase 7：Protocol Bridge Runtime

### 10.1 目标

为协议桥接 runtime 提供 first-party 集成。

### 10.2 贡献点

```text
gatewayProtocolBridges
runtimeAdapters
```

### 10.3 Core 改动

1. Provider 增加 bridge config。
2. Gateway handler 增加 bridge 分流。
3. Bridge runtime manager。
4. Request log 增加 bridge 字段。

### 10.4 Provider Proxy

第三方 bridge 需要 streaming Provider Proxy。第一版可以不开放，只支持 first-party trusted runtime。

### 10.5 验收

1. Provider 开启 bridge 后，Claude Code `/v1/messages` 可转 OpenAI Responses。
2. Provider 关闭 bridge 后，行为回到透明代理。
3. Bridge runtime 不可用时 fail closed。
4. 日志显示 original model 和 mapped model。

## 11. API 路由清单

建议新增：

```text
GET    /api/extensions
GET    /api/extensions/{id}
POST   /api/extensions/{id}/enable
POST   /api/extensions/{id}/disable
POST   /api/extensions/rescan

GET    /api/extensions/commands
POST   /api/extensions/commands/{commandId}/execute

GET    /api/extensions/{id}/settings
PUT    /api/extensions/{id}/settings

GET    /api/extensions/audit-logs
GET    /api/extensions/{id}/audit-logs
```

后续新增：

```text
GET    /api/extensions/tool-integrations
POST   /api/extensions/tool-integrations/{id}/detect
POST   /api/extensions/tool-integrations/{id}/configure
POST   /api/extensions/tool-integrations/{id}/restore

GET    /api/extensions/transcripts/sources
POST   /api/extensions/transcripts/sync

GET    /api/extensions/runtimes
POST   /api/extensions/runtimes/{id}/restart
```

## 12. 前端开发计划

### 12.1 新增类型

```text
apps/web/src/types/extension.ts
apps/desktop/src/renderer/src/types/extension.ts
```

### 12.2 API service

```text
apps/web/src/services/extensions.ts
apps/desktop/src/renderer/src/services/extensions.ts
```

### 12.3 页面

```text
apps/web/src/pages/plugins-page.tsx
apps/desktop/src/renderer/src/pages/plugins-page.tsx
```

### 12.4 导航

增加 nav item：

```text
Plugins
```

中文：

```text
插件
```

### 12.5 UI 原则

1. 插件管理是运维型页面，保持密度和清晰。
2. 不做营销式布局。
3. 权限和状态要可扫描。
4. 高风险权限用明确提示。
5. Enable / Disable 按钮要体现状态。

## 13. 测试矩阵

### 13.1 Go tests

```text
core/internal/extension
core/internal/api
core/internal/storage
```

测试项：

1. manifest parser。
2. manifest validator。
3. repository CRUD。
4. service scan。
5. enable / disable。
6. API routes。
7. audit log。

### 13.2 Frontend tests

如果现有项目没有 frontend test 基础，至少做手工验证和 build。

测试项：

1. Plugins page loads。
2. Plugin list empty state。
3. Plugin cards。
4. Permission list。
5. Enable / Disable。

### 13.3 End-to-end 手工验证

1. 创建临时 user plugin manifest。
2. Rescan 后 UI 可见。
3. Enable 后状态保持。
4. Disable 后状态保持。
5. 删除 manifest 后状态变更合理。

## 14. 第一轮实现建议

为了控制范围，第一轮只做：

```text
Manifest Loader
Plugin SQLite Repository
Plugin Service
Extension API
Plugins Page
Settings schema placeholder
Commands registry placeholder
Audit table placeholder
```

暂不做：

```text
实际启动插件进程
Process Broker
Credential Broker
Filesystem Broker
Gateway Hooks
Protocol Bridge
Markdown export
RTK command execution
```

这样可以先把系统骨架稳定下来。

## 15. 分支开发里程碑

### Milestone 1：插件骨架

状态：已完成（2026-06-05）

产出：

1. DB migration。已完成。
2. manifest parser。已完成。
3. service scan。已完成。
4. API list/get。已完成。
5. UI list。已完成。

### Milestone 2：启停和设置

产出：

1. enable / disable。
2. settings storage。
3. settings UI。
4. audit logs。

### Milestone 3：命令和工具集成

产出：

1. command registry。
2. tool integration registry。
3. declared process placeholder。
4. RTK 插件 demo manifest。

### Milestone 4：Markdown 插件 host 能力

产出：

1. transcript broker。
2. filesystem broker。
3. background task。
4. Markdown Archive first-party plugin。

### Milestone 5：Protocol Bridge first-party runtime

产出：

1. bridge runtime skeleton。
2. Provider bridge config。
3. gateway分流。
4. streaming conversion MVP。

## 16. 风险

### 16.1 范围膨胀

插件系统很容易一次性做太大。必须坚持阶段拆分。

### 16.2 权限过宽

`filesystem.homeConfig`、`process.exec`、`network.external` 必须谨慎。

### 16.3 主项目耦合

first-party 插件可以使用内部 API，但要避免公开 API 和内置实现完全分裂。

### 16.4 UI 双份维护

Web 和 Desktop renderer 需要同步变更。可以后续考虑共享 UI 包，目前按现有模式同步。

### 16.5 ignored docs

`.docs/` 当前被 git 忽略。若这些开发文档需要进入提交，需要调整 ignore 策略或另行复制到可跟踪目录。

## 17. 待确认问题

1. 第一轮是否只做 user scope，bundled scope 是否同步做？
2. Plugin manifest 文件名是否固定为 `relay-switch-plugin.json`？
3. 插件运行时协议第一版选 JSON-RPC over stdio 还是 HTTP？
4. UI 中 Plugins 页面放在主导航还是 Settings 子页？
5. 是否需要把 `.docs/extension/` 从 git ignore 中排除，以便分支提交文档？
6. first-party bundled plugins 是否也通过同一张 `plugins` 表管理启用状态？
7. 插件权限 grant 第一版是否做 install-time grant，还是先只显示不审批？

## 18. 推荐下一步

在当前 `feature/extension-system` 分支上，建议先实现 Milestone 1：

```text
DB migration
core/internal/extension manifest parser
Plugin service scan user directory
/api/extensions list/get/rescan
Plugins page empty/list state
```

Milestone 1 完成后，再评估 settings、commands 和 Extension Host 的具体协议选择。
