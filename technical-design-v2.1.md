# Clash for AI 技术模块设计 v2.1

本文档定义首版的技术模块边界、数据结构、关键流程和建议目录结构。

## 1. 目标

首版技术方案只追求三件事：

1. 稳定接收本地 `/v1/*` 请求并转发。
2. 稳定管理多 Provider 和活跃切换。
3. 稳定记录最小可用的观测信息。

## 2. 建议仓库结构

建议采用单仓结构：

```text
clash-for-ai/
  apps/
    desktop/
      package.json
      electron/
        main.ts
        tray.ts
        go-process.ts
      src/
        app/
        pages/
        components/
        services/
        types/
  core/
    cmd/
      clash-for-ai-core/
        main.go
    internal/
      app/
      api/
      config/
      provider/
      credential/
      gateway/
      health/
      logging/
      provideradapter/
      storage/
      settings/
    migrations/
    go.mod
  docs/
```

如果当前阶段想更轻量，也可以先用双目录：

```text
desktop/
core/
docs/
```

但模块边界应保持一致。

## 3. 核心模块设计

### 3.1 `config`

职责：

1. 管理应用级配置
2. 解析端口、数据目录、日志级别等配置
3. 提供默认值

建议结构：

```go
type AppConfig struct {
    HTTPPort int
    DataDir string
    LogLevel string
    GatewayBind string
}
```

### 3.2 `provider`

职责：

1. 管理 Provider 元数据
2. 维护活跃 Provider
3. 对外提供 Provider CRUD 和查询服务

建议结构：

```go
type Provider struct {
    ID string
    Name string
    BaseURL string
    APIKeyRef string
    AuthMode string
    ExtraHeaders map[string]string
    Capabilities ProviderCapabilities
    Status ProviderStatus
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

```go
type ProviderCapabilities struct {
    SupportsOpenAICompatible bool
    SupportsAnthropicCompatible bool
    SupportsModelsAPI bool
    SupportsBalanceAPI bool
    SupportsStream bool
}
```

```go
type ProviderStatus struct {
    IsActive bool
    LastHealthcheckAt *time.Time
    LastHealthStatus string
}
```

### 3.3 `credential`

职责：

1. 保存 API Key
2. 读取 API Key
3. 删除 API Key
4. 屏蔽平台差异

接口建议：

```go
type Store interface {
    Save(ctx context.Context, key string, value string) (string, error)
    Get(ctx context.Context, ref string) (string, error)
    Delete(ctx context.Context, ref string) error
}
```

说明：

1. `key` 是内部标识，如 `provider/<id>/api-key`
2. 返回值 `ref` 是安全存储引用
3. SQLite 只保存 `ref`

### 3.4 `storage`

职责：

1. 初始化 SQLite
2. 管理 migrations
3. 提供 repository 依赖

建议表：

1. `providers`
2. `request_logs`
3. `settings`
4. `healthcheck_history`

首版不建议过度拆表，避免增加迁移复杂度。

### 3.5 `gateway`

职责：

1. 接收 `/v1/*` 请求
2. 查找活跃 Provider
3. 生成上游请求
4. 注入 Header
5. 透传响应
6. 将请求结果写给 logging 模块

建议拆分：

1. `handler.go`
2. `director.go`
3. `transport.go`
4. `errors.go`

关键接口：

```go
type ActiveProviderResolver interface {
    GetActiveProvider(ctx context.Context) (*provider.Provider, error)
}
```

```go
type CredentialGetter interface {
    Get(ctx context.Context, ref string) (string, error)
}
```

### 3.6 `health`

职责：

1. 运行健康检查
2. 执行探测策略
3. 回写 Provider 健康状态

建议结构：

```go
type CheckResult struct {
    Status string
    StatusCode int
    LatencyMs int64
    Summary string
    CheckedAt time.Time
}
```

首版策略：

1. 默认 `/models`
2. 允许 Provider 自定义探测路径
3. 不默认执行生成类探测

### 3.7 `logging`

职责：

1. 标准化请求日志
2. 标准化错误分类
3. 异步落库
4. 清理旧日志

建议结构：

```go
type RequestLog struct {
    ID string
    Timestamp time.Time
    ProviderID string
    ProviderName string
    Method string
    Path string
    Model *string
    StatusCode *int
    IsStream bool
    UpstreamHost string
    LatencyMs int64
    FirstByteMs *int64
    FirstTokenMs *int64
    ErrorType *string
    ErrorMessage *string
    ErrorSnippet *string
}
```

### 3.8 `provideradapter`

职责：

1. 余额查询适配
2. Provider 特定错误识别
3. Provider 特定探测配置

接口建议：

```go
type Adapter interface {
    Name() string
    Match(baseURL string) bool
    Balance(ctx context.Context, p provider.Provider, apiKey string) (*BalanceResult, error)
    ClassifyError(statusCode int, bodySnippet string) *string
}
```

首版不要让 adapter 参与主代理链路，只参与增强能力。

### 3.9 `api`

职责：

1. 暴露管理端 API
2. 将前端需要的能力显式化
3. 保持与 Gateway 路由分离

建议分组：

1. `/api/providers/*`
2. `/api/logs/*`
3. `/api/settings/*`
4. `/health`

## 4. 关键流程设计

### 4.1 Provider 创建流程

```text
UI 提交 Provider 表单
  -> API 校验字段
  -> credential.Store.Save(api_key)
  -> provider.Repository.Create(metadata + api_key_ref)
  -> health.Service.Check(provider)
  -> 返回 Provider + 初始状态
```

关键要求：

1. API Key 保存成功后才能落 Provider 元数据
2. 健康检查失败不应导致 Provider 创建失败
3. 健康状态要与保存结果一起返回给 UI

### 4.2 活跃 Provider 切换流程

```text
UI 点击切换
  -> POST /api/providers/:id/activate
  -> provider.Service.Activate(id)
  -> 持久化 is_active
  -> 更新内存态 active provider
  -> 返回最新 active provider
```

关键要求：

1. 切换动作应具备原子性
2. 新请求必须读取最新活跃 Provider
3. 切换不影响已在进行中的旧请求

### 4.3 Gateway 请求流程

```text
客户端请求 /v1/*
  -> gateway.Handler 校验是否有活跃 Provider
  -> credential.Store.Get(api_key_ref)
  -> 构造 upstream URL
  -> 覆盖 auth headers
  -> 注入 extra headers
  -> 发送请求
  -> 透传响应
  -> 异步写日志
```

关键要求：

1. 不能依赖解析 body 才能转发
2. stream 响应不能被缓冲到内存
3. 日志失败不能影响主请求成功

### 4.4 错误分类流程

```text
响应结束或请求失败
  -> logging.Normalize(error/status/snippet)
  -> 先走通用分类
  -> 若命中 adapter，再做 provider-specific 分类
  -> 落日志
```

关键要求：

1. 分类优先保守
2. 不把不确定的 429 误标成余额不足
3. UI 应能区分通用分类与适配分类

## 5. 本地 API 设计建议

### 5.1 Provider DTO

返回给前端的 Provider 不应包含明文 API Key。

建议返回：

```json
{
  "id": "uuid",
  "name": "Provider A",
  "base_url": "https://api.example.com/v1",
  "auth_mode": "bearer",
  "extra_headers": {
    "anthropic-version": "2023-06-01"
  },
  "capabilities": {
    "supports_openai_compatible": true,
    "supports_anthropic_compatible": false,
    "supports_models_api": true,
    "supports_balance_api": false,
    "supports_stream": true
  },
  "status": {
    "is_active": true,
    "last_health_status": "ok"
  },
  "api_key_masked": "sk-****1234"
}
```

### 5.2 日志查询参数

建议支持：

1. `provider_id`
2. `status_code`
3. `error_type`
4. `page`
5. `page_size`

## 6. 数据库设计建议

### 6.1 `providers`

建议字段：

1. `id`
2. `name`
3. `base_url`
4. `api_key_ref`
5. `auth_mode`
6. `extra_headers_json`
7. `capabilities_json`
8. `is_active`
9. `last_healthcheck_at`
10. `last_health_status`
11. `created_at`
12. `updated_at`

### 6.2 `request_logs`

建议字段：

1. `id`
2. `timestamp`
3. `provider_id`
4. `provider_name`
5. `method`
6. `path`
7. `model`
8. `status_code`
9. `is_stream`
10. `upstream_host`
11. `latency_ms`
12. `first_byte_ms`
13. `first_token_ms`
14. `error_type`
15. `error_message`
16. `error_snippet`

建议索引：

1. `timestamp`
2. `provider_id`
3. `error_type`
4. `status_code`

### 6.3 `settings`

建议字段：

1. `key`
2. `value_json`
3. `updated_at`

首版设置项：

1. `http_port`
2. `log_retention_limit`
3. `window_behavior`

## 7. 并发与状态要求

### 7.1 活跃 Provider

活跃 Provider 应同时存在两层：

1. 持久化状态
2. 内存缓存

读取路径优先内存缓存，变更时写库并刷新缓存。

### 7.2 请求日志

日志写入应使用异步队列或 channel。

要求：

1. 不阻塞代理主请求
2. 队列满时允许降级丢弃低优先日志，但要有内部告警

### 7.3 超时控制

建议区分：

1. 上游连接超时
2. 首字节超时
3. 空闲连接超时

不要只配置一个总超时。

## 8. UI 模块建议

### 8.1 页面

1. Provider 管理页
2. 请求日志页
3. 设置页

### 8.2 组件

1. Provider 列表卡片
2. Provider 编辑表单
3. 健康状态徽标
4. 错误类型标签
5. 日志筛选栏

### 8.3 前端状态

建议按领域拆分：

1. `providers`
2. `logs`
3. `settings`
4. `app-runtime`

## 9. 首版不建议现在做的技术项

1. 不要在 Gateway 主链路里引入复杂 adapter
2. 不要先做插件系统
3. 不要先做自动 fallback
4. 不要先做模型映射和别名转换
5. 不要先做远程同步

这些能力都应该建立在主链路稳定之后。

## 10. 建议的第一批代码文件

如果下一步直接开工，建议先创建这些文件：

### Go Core

1. `core/cmd/clash-for-ai-core/main.go`
2. `core/internal/app/app.go`
3. `core/internal/config/config.go`
4. `core/internal/provider/model.go`
5. `core/internal/provider/repository.go`
6. `core/internal/provider/service.go`
7. `core/internal/credential/store.go`
8. `core/internal/gateway/handler.go`
9. `core/internal/api/router.go`
10. `core/internal/storage/sqlite.go`

### Desktop

1. `apps/desktop/electron/main.ts`
2. `apps/desktop/electron/go-process.ts`
3. `apps/desktop/electron/tray.ts`
4. `apps/desktop/src/app/App.tsx`
5. `apps/desktop/src/pages/providers-page.tsx`
6. `apps/desktop/src/services/api.ts`

## 11. 实现顺序结论

最合理的实现路径是：

1. 先搭工程和进程管理
2. 再做 Provider CRUD 和安全存储
3. 再做 Gateway 主链路
4. 再做日志和健康检查
5. 最后补托盘、模型、余额、测速

这样可以最早把核心风险暴露出来，也最符合这个产品首版的真实价值。
