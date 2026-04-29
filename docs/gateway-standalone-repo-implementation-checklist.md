# Gateway 独立仓库实施 Checklist

本文是 [`docs/gateway-standalone-repo-contract-and-integration-plan.md`](./gateway-standalone-repo-contract-and-integration-plan.md) 的落地执行版。

目标不是重复 contract，而是回答 4 个问题：

1. 当前代码已经做到哪个阶段
2. 还缺哪些关键能力
3. 具体应该先改哪些代码
4. 后续如何对照本文逐步迁出 gateway 独立仓库

如果本文与 contract 文档冲突，以 contract 文档为准；如果本文缺少执行细节，则以本文补充为准。

## 1. 当前阶段判断

当前实现处于：

1. 阶段 1 基本完成
2. 阶段 2 进行中，完成度约 60%-70%
3. 阶段 3 尚未开始
4. 阶段 4 尚未开始

一句话判断：

当前仓库已经具备“把 gateway 当独立 runtime 启动和探测”的骨架，但还没有完全做到“当前项目只通过 adapter + HTTP contract 管理和使用 gateway”。

## 2. 已完成能力

### 2.1 Embedded / External 接入骨架已存在

当前项目已经具备两种 runtime 接入模式：

1. external 模式通过 `LOCAL_GATEWAY_RUNTIME_BASE_URL` 接入外部 runtime
2. embedded 模式通过 adapter 启动本地 runtime 进程

对应代码：

1. `core/internal/app/app.go`
2. `core/internal/gatewayadapter/external_runtime.go`
3. `core/internal/gatewayadapter/embedded_local_runtime.go`

### 2.2 Required capability 检测已存在

当前已经定义并校验 required capability：

1. `supports_openai_compatible`
2. `supports_anthropic_compatible`
3. `supports_models_api`
4. `supports_stream`

对应代码：

1. `core/internal/gatewayadapter/adapter.go`
2. `core/internal/gatewayadapter/external_runtime.go`
3. `core/internal/app/app.go`

### 2.3 Runtime 独立数据目录已落地

当前已经为 runtime 准备独立 data dir：

1. `local-gateway-runtime/clash-for-ai.db`
2. `local-gateway-runtime/credentials.json`

同时已有一次性迁移逻辑，把旧的 core 内部状态迁入 runtime data dir。

对应代码：

1. `core/internal/app/app.go`

### 2.4 Local gateway provider 已作为 runtime provider 初始化

当前项目会根据 runtime discover 结果更新 `system-local-gateway.base_url`，然后把它当成系统 provider 使用。

对应代码：

1. `core/internal/app/app.go`
2. `core/internal/provider/service.go`

### 2.5 Runtime HTTP contract 的服务端骨架已存在

当前 runtime 侧已经具备下面这些 HTTP 路由：

1. `GET /health`
2. `GET /v1/models`
3. `POST /v1/chat/completions`
4. `POST /v1/responses`
5. `POST /v1/messages`
6. `POST /v1/messages/count_tokens`
7. `GET /admin/model-sources`
8. `POST /admin/model-sources`
9. `PUT /admin/model-sources/:id`
10. `DELETE /admin/model-sources/:id`
11. `PUT /admin/model-sources/order`
12. `GET /admin/selected-models`
13. `PUT /admin/selected-models`

对应代码：

1. `core/internal/gateway/local_runtime_handler.go`

## 3. 未完成的关键问题

这些问题不解决，阶段 2 就不能算完成。

### 3.1 Adapter 还没有真正消费 optional admin contract

虽然 runtime handler 已经实现了 admin API，但 adapter 侧还没有真正通过 HTTP 调用这些 API。

当前表现：

1. `EmbeddedLocalRuntimeAdapter` 的 admin 方法全部返回 `ErrRuntimeAdminUnsupported`
2. `ExternalRuntimeAdapter` 的 admin 方法全部返回 `ErrRuntimeAdminUnsupported`

这说明：

1. 当前项目还不能统一通过 adapter 管理 runtime 内部 model sources
2. 当前项目还不能统一通过 adapter 管理 runtime 内部 selected-models
3. external / embedded 两种模式在管理面上还没有实现真正一致

对应代码：

1. `core/internal/gatewayadapter/embedded_local_runtime.go`
2. `core/internal/gatewayadapter/external_runtime.go`

### 3.2 Embedded 模式仍然启动当前仓库自己的 core binary

当前 embedded 模式不是启动一个独立 gateway binary，而是启动当前程序自身，然后通过：

1. `CLASH_FOR_AI_MODE=local-gateway-runtime`

走到 runtime 分支。

这意味着：

1. 现在只是进程拆开了
2. 但仓库边界和发布边界还没有拆开

对应代码：

1. `core/cmd/clash-for-ai-core/main.go`
2. `core/internal/gatewayadapter/embedded_local_runtime.go`

### 3.3 Runtime 核心实现仍然留在当前仓库

目前 runtime 的核心实现仍然在主仓库中，包括：

1. runtime 入口
2. runtime HTTP handler
3. inbound parser
4. upstream executor
5. runtime state
6. runtime model source 管理

这意味着：

1. 当前仓库仍然直接持有 runtime 实现
2. 还没有进入真正的拆仓阶段

### 3.4 Runtime 状态迁移仍然由 core 直接操作 runtime schema

当前 `prepareLocalRuntimeState` 会直接打开 runtime sqlite 和 runtime credentials，并写入 runtime 内部数据结构。

这在阶段 2 过渡期可接受，但到阶段 3 时应尽量收敛为：

1. 一次性迁移脚本
2. 或 runtime 自己提供初始化 / import 能力

而不是长期由 core 直接耦合 runtime 内部 schema。

## 4. 结论：当前最优先任务

当前最优先的不是“立刻拆仓”，而是先把当前项目训练成只会通过 adapter + contract 使用 runtime。

具体来说，先完成这件事：

1. 让 core 的管理面也只通过 adapter 调用 runtime admin API

如果这一步没完成，后面拆仓只会把内部耦合原样搬去另一个仓库。

## 5. 迁仓 Checklist

## 5.1 应保留在当前仓库的代码

以下内容属于控制面，继续保留在当前仓库：

1. provider 管理与持久化
2. active provider 管理
3. provider healthcheck
4. provider-scoped Claude Code metadata
5. 桌面端 UI
6. 工具接入逻辑
7. gateway adapter
8. embedded / external runtime 选择逻辑
9. `system-local-gateway` 这个系统 provider 的定义

主要涉及：

1. `core/internal/provider/**`
2. `core/internal/health/**`
3. `core/internal/gatewayadapter/**`
4. `core/internal/api/**`
5. `apps/desktop/**`

## 5.2 应迁出到独立 gateway 仓库的代码

以下内容属于 runtime，应迁出：

1. runtime binary 入口
2. runtime HTTP handler
3. runtime inbound request parser
4. runtime upstream executor
5. runtime `/health`
6. runtime `/v1/models`
7. runtime `/v1/chat/completions`
8. runtime `/v1/responses`
9. runtime `/v1/messages`
10. runtime `/v1/messages/count_tokens`
11. runtime `/admin/model-sources*`
12. runtime `/admin/selected-models`
13. runtime 内部 sqlite / credentials / selected-models state
14. runtime 内部 model source 管理

当前文件映射：

1. `core/cmd/clash-for-ai-core/main.go` 中的 runtime 分支
2. `core/internal/app/app.go` 中的 `RunLocalGatewayRuntime`
3. `core/internal/gateway/local_runtime_handler.go`
4. `core/internal/localgateway/**`
5. `core/internal/localgatewaystate/**`
6. `core/internal/modelsource/**` 中属于 local gateway runtime 的部分

## 5.3 阶段 2 完成前禁止删除的内容

在 adapter 和 contract 闭环完成前，不要急着删除：

1. `prepareLocalRuntimeState`
2. runtime 迁移兼容逻辑
3. `CLASH_FOR_AI_MODE=local-gateway-runtime` 分支

原因：

1. 它们仍是当前运行链路的一部分
2. 过早删除会让 embedded 模式退化

## 6. 代码级改造方案

以下方案按实际开发顺序组织。

## 6.1 第一步：补齐 adapter 的 admin HTTP 客户端

目标：

1. 无论 embedded 还是 external，core 都只通过 adapter 管 runtime admin

### 要修改的文件

1. `core/internal/gatewayadapter/embedded_local_runtime.go`
2. `core/internal/gatewayadapter/external_runtime.go`
3. `core/internal/gatewayadapter/adapter.go`

### 具体改造内容

为两类 adapter 真正实现以下方法：

1. `ListModelSources`
2. `CreateModelSource`
3. `UpdateModelSource`
4. `DeleteModelSource`
5. `ReplaceModelSourceOrder`
6. `ListSelectedModels`
7. `ReplaceSelectedModels`

实现要求：

1. 全部通过 HTTP 调 runtime
2. embedded 与 external 逻辑保持一致
3. capability 不支持时返回 `ErrRuntimeAdminUnsupported`
4. HTTP 404 / 405 / 501 应映射为 admin unsupported
5. 其它 4xx / 5xx 应保留为真实错误

建议在 `gatewayadapter` 内新增一个复用层，例如：

1. `admin_client.go`

候选职责：

1. 拼接 URL
2. 发起请求
3. 统一处理 JSON encode / decode
4. 统一把 unsupported 状态码映射为 `ErrRuntimeAdminUnsupported`

### 完成标准

满足以下条件才算完成：

1. adapter 的 admin 方法不再是 stub
2. embedded / external 都可读写 runtime admin API
3. optional capability 缺失时会优雅降级，而不是崩溃

## 6.2 第二步：在 core 增加 runtime admin 控制面入口

目标：

1. UI 和前端不直接理解 runtime 内部实现
2. 所有 local gateway 管理动作都先进入 core 控制面，再由 core 转给 adapter

### 要修改的文件

1. `core/internal/api/router.go`
2. `core/internal/provider/service.go` 或新增专用 service
3. `apps/desktop/**` 中调用 local gateway 管理接口的部分

### 建议新增 API

建议新增一组明确的 local gateway control-plane API：

1. `GET /api/local-gateway/model-sources`
2. `POST /api/local-gateway/model-sources`
3. `PUT /api/local-gateway/model-sources/:id`
4. `DELETE /api/local-gateway/model-sources/:id`
5. `PUT /api/local-gateway/model-sources/order`
6. `GET /api/local-gateway/selected-models`
7. `PUT /api/local-gateway/selected-models`
8. `GET /api/local-gateway/runtime`

其中 `GET /api/local-gateway/runtime` 建议返回：

1. runtime base URL
2. runtime mode
3. runtime health
4. runtime capabilities
5. optional admin 是否可用

### 不建议的做法

不建议把这些能力继续塞回：

1. `/api/providers/:id/...`

原因：

1. `system-local-gateway` 是系统 runtime provider，不是普通 provider
2. 文档已经要求 runtime 内部 selected-models 是 runtime 内部状态
3. 用专用 local gateway 控制面 API 更不容易再次混淆边界

### 完成标准

满足以下条件才算完成：

1. 前端只通过 core control-plane API 管理 local gateway
2. core control-plane API 只通过 adapter 管理 runtime
3. local gateway 的 selected-models 不再回退到 provider repository 语义

## 6.3 第三步：把 runtime 相关 service 从 core 控制面依赖中剥离

目标：

1. core 不再长期直接依赖 runtime 内部 `modelsource` / `localgatewaystate` service

### 要修改的文件

1. `core/internal/app/app.go`
2. `core/internal/provider/service.go`
3. `core/internal/api/router.go`

### 具体改造内容

逐步做到：

1. 运行期管理路径只走 adapter
2. `prepareLocalRuntimeState` 仅保留一次性迁移职责
3. 迁移完成后，core 不再直接对 runtime 内部表做增删改查

建议把 `prepareLocalRuntimeState` 的职责限定为：

1. 首次启动时把旧数据迁入 runtime data dir
2. 清理 legacy 字段
3. 不参与日常 runtime 管理

### 完成标准

满足以下条件才算完成：

1. 日常运行中，core 不会直接调用 runtime 的内部 state service
2. runtime 的 model source / selected-models 修改都走 HTTP admin API

## 6.4 第四步：拆分独立 gateway 仓库

目标：

1. runtime 代码从当前仓库物理迁出

### 建议的新仓库结构

建议至少包含：

1. `cmd/gateway`
2. `internal/runtime/handler`
3. `internal/runtime/inbound`
4. `internal/runtime/executor`
5. `internal/runtime/providers/openai`
6. `internal/runtime/providers/anthropic`
7. `internal/runtime/state`
8. `internal/runtime/modelsource`
9. `internal/runtime/admin`
10. `internal/runtime/health`

### 从当前仓库迁出的来源

建议迁出：

1. `core/internal/gateway/local_runtime_handler.go`
2. `core/internal/localgateway/**`
3. `core/internal/localgatewaystate/**`
4. runtime 专用的 `modelsource` 实现
5. `RunLocalGatewayRuntime` 所需启动逻辑

### 注意事项

拆仓时不要把以下内容带走：

1. `provider` 控制面
2. `system-local-gateway` provider 定义
3. Claude Code model map
4. 工具接入和桌面端 UI

## 6.5 第五步：切换 embedded 模式到新仓库产物

目标：

1. embedded 模式不再启动当前 core binary
2. embedded 模式改为启动独立 gateway 仓库的发布产物

### 要修改的文件

1. `core/internal/gatewayadapter/embedded_local_runtime.go`
2. 打包 / 发布相关脚本
3. 桌面端资源打包配置

### 具体改造内容

把当前：

1. `exec.Command(a.executable)`

改为：

1. 启动独立 gateway binary 路径

并继续保留：

1. data dir 注入
2. host / port 注入
3. `/health` 等待逻辑

### 完成标准

满足以下条件才算完成：

1. 当前仓库内不再需要 runtime 分支启动逻辑
2. embedded / external 只是部署差异，不再是实现差异

## 6.6 第六步：删除当前仓库中的 runtime 核心实现

只有在前 5 步全部完成后，才删除：

1. `RunLocalGatewayRuntime`
2. `CLASH_FOR_AI_MODE=local-gateway-runtime` 逻辑
3. `core/internal/gateway/local_runtime_handler.go`
4. `core/internal/localgateway/**`
5. `core/internal/localgatewaystate/**`
6. runtime 专属 `modelsource` 实现

删除标准：

1. 当前仓库已只剩 control plane + adapter
2. 所有运行与管理动作都通过独立 runtime contract 完成

## 7. 推荐开发顺序

建议严格按下面顺序开发：

1. 实现 adapter admin HTTP 客户端
2. 新增 core local gateway control-plane API
3. 前端改为调用新的 control-plane API
4. 补齐 capability 驱动的降级逻辑
5. 收敛 `prepareLocalRuntimeState` 为一次性迁移逻辑
6. 拆出独立 gateway 仓库
7. embedded 模式切到新仓库 binary
8. 删除当前仓库内残留 runtime 实现

不建议跳步：

1. 不建议在 adapter admin 还没打通时先拆仓
2. 不建议在 control-plane API 还没切换时先删内部 service

## 8. 建议拆分的开发任务

为了便于逐步落地，建议拆成下面这些 PR / task。

### Task 1：Gateway Admin Adapter

范围：

1. `gatewayadapter` 实现所有 admin HTTP client
2. 增加单测覆盖 embedded / external / unsupported 场景

完成后收益：

1. 当前项目第一次真正具备“通过 contract 管 runtime admin”的能力

### Task 2：Local Gateway Control Plane API

范围：

1. 新增 `/api/local-gateway/*`
2. core 内部接 adapter

完成后收益：

1. local gateway 管理入口和普通 provider 管理入口彻底分开

### Task 3：Frontend Switch

范围：

1. 桌面端切到 `/api/local-gateway/*`
2. 根据 capability 显示只读 / 可编辑状态

完成后收益：

1. UI 语义与 contract 语义对齐

### Task 4：Runtime Migration Cleanup

范围：

1. 收敛 runtime legacy 迁移逻辑
2. 减少 core 对 runtime 内部 schema 的直接触达

完成后收益：

1. 为拆仓清障

### Task 5：Extract Standalone Gateway Repo

范围：

1. 建独立仓库
2. 迁 runtime 核心代码
3. 独立构建和发布

完成后收益：

1. 阶段 3 完成

### Task 6：Switch Embedded Runtime Artifact

范围：

1. embedded 模式改启动独立仓库 binary
2. 删除当前仓库 runtime 主体实现

完成后收益：

1. 阶段 4 完成

## 9. 验收标准

当下面全部满足时，说明可以认为迁仓目标达成：

1. 当前项目可以通过 embedded binary 启动独立 gateway runtime
2. 当前项目可以通过 external URL 接入任意满足 contract 的 gateway runtime
3. 当前项目只依赖 stable contract，不依赖 runtime 内部包结构
4. runtime 使用独立 data dir
5. runtime 内部 model source / selected-models 不再由 core 直接持有和日常管理
6. optional admin 缺失时，系统仍能只读 provider + 正常转发
7. 当前仓库中不再包含 runtime 核心实现，只保留 adapter 和接入逻辑

## 10. 开发时的判断原则

后续每个改动都可以先问 3 个问题：

1. 这个改动是不是在增强“当前项目只通过 contract 使用 runtime”？
2. 这个改动是不是在减少“core 对 runtime 内部实现的直接依赖”？
3. 这个改动是不是在提升“独立仓库迁出后的可运行性和可维护性”？

如果 3 个问题里至少有 2 个答案是否定的，这个改动优先级通常不高。

## 11. 一句话执行结论

接下来的正确方向不是先搬代码，而是：

先补齐 adapter admin 和 core control-plane 闭环，把当前项目训练成真正只会通过 contract 把 gateway 当第三方 runtime 使用；然后再拆出独立仓库并切换 embedded 产物。
