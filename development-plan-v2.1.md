# Clash for AI 开发任务清单 v2.1

本文档基于 [product-prd-v2.1.md](/Users/yuanjunliang/Documents/workspace/profile/clash-for-ai/product-prd-v2.1.md)，用于把 PRD 直接拆成可执行开发任务。

## 1. 开发原则

1. 先打通主链路，再补可观测和体验。
2. 先做稳定透传，再做 Provider 适配。
3. 先保证 Claude Code 和一个 OpenAI-compatible 客户端可用，再扩展兼容范围。
4. 所有“看起来聪明”的逻辑都后置，首版优先保守正确。

## 2. 开发阶段

### Phase 0：工程初始化

目标：把工程跑起来，确定技术骨架和本地开发方式。

任务：

1. 初始化 monorepo 或双目录结构。
2. 初始化 Electron 应用。
3. 初始化 Go core 服务。
4. 打通 Electron 启动 Go 子进程。
5. 打通 UI 访问本地 Go API。
6. 建立基础日志输出和 `/health` 检查。

完成标准：

1. 启动桌面应用后，Go core 自动启动。
2. UI 能读取 `/health` 并展示服务在线状态。

### Phase 1：Provider 管理闭环

目标：完成 Provider 的增删改查与活跃切换。

任务：

1. 设计 Provider SQLite 表结构。
2. 抽象 Credential Store 接口。
3. 接入系统安全存储。
4. 实现 Provider CRUD API。
5. 实现活跃 Provider 切换 API。
6. 实现 Provider 列表 UI。
7. 实现添加 / 编辑 Provider 表单。
8. 实现删除 Provider 和切换交互。

完成标准：

1. 用户能创建两个 Provider。
2. API Key 不落明文数据库。
3. 用户能在 UI 中切换当前活跃 Provider。

### Phase 2：Gateway 主链路

目标：打通本地统一入口到上游 Provider 的请求转发。

任务：

1. 实现本地监听端口配置。
2. 实现 `/v1/*` 路径捕获与转发。
3. 实现 `auth_mode` 注入逻辑。
4. 实现 `extra_headers` 注入逻辑。
5. 实现 `base_url` 路径拼接逻辑。
6. 实现 stream 透传。
7. 实现无活跃 Provider 的本地错误返回。
8. 处理超时、连接失败、上游 4xx/5xx 的基础响应透传。

完成标准：

1. Claude Code 可通过 `/v1/messages` 正常请求。
2. 一个 OpenAI-compatible 客户端可通过 `/v1/chat/completions` 正常请求。
3. 切换活跃 Provider 后，新请求立即生效。

### Phase 3：健康检查与日志

目标：用户可以看到 Provider 状态并排查问题。

任务：

1. 实现单 Provider 健康检查服务。
2. 为 Provider 增加最近一次健康检查状态字段。
3. 实现请求日志模型和 SQLite 表。
4. 在 Gateway 请求链路中异步写日志。
5. 实现错误摘要截断逻辑。
6. 实现通用错误分类。
7. 实现请求日志列表 API。
8. 实现日志筛选 UI。

完成标准：

1. 用户能看到每个 Provider 最近一次健康检查结果。
2. 用户能看到最近请求记录。
3. 用户能区分 `network_error`、`timeout`、`auth_error`、`rate_limited`、`upstream_error`。

### Phase 4：桌面体验补全

目标：让应用具备持续常驻和快速切换能力。

任务：

1. 实现系统托盘。
2. 托盘展示当前活跃 Provider。
3. 托盘支持快速切换 Provider。
4. 增加窗口打开 / 隐藏逻辑。
5. 增加端口占用检测和错误提示。
6. 增加应用配置页。

完成标准：

1. 用户可不打开主窗口，仅通过托盘切换 Provider。
2. 端口冲突时用户能看到明确错误信息。

### Phase 5：增强能力

目标：补齐非核心但有价值的观测和适配能力。

任务：

1. 实现可选测速。
2. 实现 `/v1/models` 拉取与缓存。
3. 实现模型列表 UI。
4. 为 2-3 家 Provider 实现余额 adapter。
5. 实现 Provider 导入导出。
6. 建立兼容性矩阵文档和验证记录。

完成标准：

1. 用户可以手动触发测速。
2. 已支持 Provider 可看到余额。
3. 兼容性矩阵可持续维护。

## 3. 模块级任务拆解

### 3.1 Go Core

#### `core/app`

任务：

1. 启动配置加载
2. HTTP Server 启动
3. 路由注册
4. 生命周期管理

#### `core/provider`

任务：

1. Provider model
2. Provider repository
3. Provider service
4. Active provider state management

#### `core/credential`

任务：

1. 统一 Credential Store 接口
2. 各平台实现适配层
3. Provider API key 的 save/get/delete

#### `core/gateway`

任务：

1. 路径重写
2. Header 注入
3. 上游请求转发
4. Stream 透传
5. 请求上下文打点

#### `core/health`

任务：

1. 健康检查策略
2. 探测请求执行
3. 结果写回 Provider 状态

#### `core/logging`

任务：

1. Request log model
2. Error normalization
3. Async log sink
4. Log retention cleanup

#### `core/provideradapter`

任务：

1. 定义 balance adapter 接口
2. 定义 provider-specific error matcher
3. 注册适配器

#### `core/api`

任务：

1. Provider API
2. Health API
3. Log API
4. Settings API

### 3.2 Electron Main

任务：

1. 应用单实例控制
2. Go 子进程启动与退出管理
3. 健康轮询
4. 系统托盘
5. 窗口管理
6. 端口和启动错误提示

### 3.3 Frontend UI

任务：

1. Provider 列表页
2. Provider 编辑弹窗
3. 状态面板
4. 请求日志页
5. 设置页
6. 全局错误与空状态处理

## 4. 后端 API 清单

### 4.1 Provider API

1. `GET /api/providers`
2. `POST /api/providers`
3. `PUT /api/providers/:id`
4. `DELETE /api/providers/:id`
5. `POST /api/providers/:id/activate`
6. `POST /api/providers/:id/healthcheck`
7. `GET /api/providers/:id/models`
8. `GET /api/providers/:id/balance`

### 4.2 日志与设置 API

1. `GET /api/logs`
2. `DELETE /api/logs`
3. `GET /api/settings`
4. `PUT /api/settings`
5. `GET /health`

## 5. 测试任务

### 5.1 单元测试

1. `base_url` 路径拼接
2. `auth_mode` header 注入
3. 活跃 Provider 切换逻辑
4. 错误分类逻辑
5. 日志截断逻辑

### 5.2 集成测试

1. 本地 Gateway 转发到 mock upstream
2. stream 透传
3. Provider 切换后请求命中新上游
4. 健康检查结果写回

### 5.3 手工验收测试

1. Claude Code 走 `/v1/messages`
2. Cursor 或 Chatbox 走 `/v1/chat/completions`
3. 托盘切换 Provider
4. 上游 401 / 429 / 5xx 排障验证

## 6. 优先级顺序

按实际开发顺序，建议严格执行：

1. 工程初始化
2. Provider CRUD
3. Credential Store
4. Active Provider 状态管理
5. Gateway 主链路
6. Stream 透传
7. 请求日志
8. 健康检查
9. 通用错误分类
10. 托盘
11. 模型列表
12. 测速
13. 余额 adapter

## 7. 建议交付顺序

第一周：

1. 工程骨架
2. Provider CRUD
3. 安全存储接口
4. Active Provider 切换

第二周：

1. Gateway 主链路
2. Stream 透传
3. Claude Code / OpenAI-compatible 双路径验证

第三周：

1. 健康检查
2. 请求日志
3. 通用错误分类
4. 托盘

第四周：

1. 模型列表
2. 测速
3. Provider adapter
4. 兼容性矩阵整理

## 8. 当前开发入口建议

如果下一步直接开始编码，建议先落下面这些基础文件：

1. Go core 目录结构
2. Provider 数据表与 repository
3. Credential Store 接口
4. Gateway handler 骨架
5. Electron main 启动 Go 子进程逻辑
6. 最小 Provider 列表 UI

这 6 项完成后，产品主链路就不会再是纸面设计。
