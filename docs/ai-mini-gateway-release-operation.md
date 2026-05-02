# ai-mini-gateway 发布接入操作手册

## 1. 文档目的

本文档用于指导 `clash-for-ai` 在发布阶段如何接入和升级内置 `ai-mini-gateway` runtime binary。

它是 [docs/ai-mini-gateway-binary-distribution.md](/Users/yuanjunliang/Documents/workspace/profile/clash-for-ai/docs/ai-mini-gateway-binary-distribution.md) 的执行手册，重点回答：

1. 升级 manifest 时具体改什么
2. 如何准备 release runtime binary
3. 如何打包并验证安装包内置 runtime

## 2. 当前接入结构

当前仓库中的发布期接入点如下：

- manifest:
  [apps/desktop/resources/ai-mini-gateway/manifest.json](/Users/yuanjunliang/Documents/workspace/profile/clash-for-ai/apps/desktop/resources/ai-mini-gateway/manifest.json)
- runtime 准备脚本:
  [apps/desktop/scripts/prepare-ai-mini-gateway.mjs](/Users/yuanjunliang/Documents/workspace/profile/clash-for-ai/apps/desktop/scripts/prepare-ai-mini-gateway.mjs)
- 打包资源配置:
  [apps/desktop/electron-builder.yml](/Users/yuanjunliang/Documents/workspace/profile/clash-for-ai/apps/desktop/electron-builder.yml)
- 桌面端注入 bundled runtime 路径:
  [apps/desktop/src/main/core-process.ts](/Users/yuanjunliang/Documents/workspace/profile/clash-for-ai/apps/desktop/src/main/core-process.ts)

约定：

1. `manifest.json` 是发布输入
2. `bin/` 和 `version.json` 是脚本生成产物，不手工维护
3. 桌面端启动 `core` 时会按优先级注入 `LOCAL_GATEWAY_RUNTIME_EXECUTABLE`

## 3. 版本对齐规则

`clash-for-ai` 和 `ai-mini-gateway` 不使用同一套版本号。

采用以下规则：

1. `clash-for-ai` 保持自己的产品版本
2. `ai-mini-gateway` 保持自己的 release tag
3. `clash-for-ai` 通过 manifest pin 一个明确的 runtime 版本

也就是说：

- 产品版本看 `apps/desktop/package.json`
- runtime 版本看 `apps/desktop/resources/ai-mini-gateway/manifest.json`

manifest 中至少维护：

1. `version`
2. `commit`
3. `contract_version`
4. `source_commit_time`

推荐约束：

1. `version` 必须直接对齐 `ai-mini-gateway` GitHub Release tag，例如 `v0.1.0`
2. `commit` 必须对齐该 release 实际对应的 commit short sha
3. `contract_version` 用于判断 HTTP contract 兼容性，当前固定为 `v1`

不要做的事：

1. 不要把 `clash-for-ai` 版本号强行改成和 runtime 一样
2. 不要只更新 `version` 不更新 `commit`
3. 不要只靠 release 页面标题判断版本，不核对 tag 和 commit

## 4. ai-mini-gateway release 资产规则

`ai-mini-gateway` 当前 release workflow 产物命名规则为：

```text
ai-mini-gateway_<version>_<goos>_<goarch>.tar.gz
ai-mini-gateway_<version>_<goos>_<goarch>.zip
```

示例：

```text
ai-mini-gateway_v0.1.0_darwin_arm64.tar.gz
ai-mini-gateway_v0.1.0_linux_amd64.tar.gz
ai-mini-gateway_v0.1.0_windows_amd64.zip
```

当前脚本行为：

1. macOS / Linux 使用 `.tar.gz`
2. Windows 使用 `.zip`
3. 脚本会按当前平台自动拼接 asset 文件名

## 5. 升级 manifest

升级 `ai-mini-gateway` 时，第一步只改 manifest，不直接改二进制文件。

编辑：

[apps/desktop/resources/ai-mini-gateway/manifest.json](/Users/yuanjunliang/Documents/workspace/profile/clash-for-ai/apps/desktop/resources/ai-mini-gateway/manifest.json)

需要更新的字段：

1. `version`
2. `commit`
3. `contract_version`
4. `source_commit_time`

示例：

```json
{
  "runtime_kind": "ai-mini-gateway",
  "release_repo": "xiaoyuandev/ai-mini-gateway",
  "version": "v0.1.1",
  "commit": "abcdef1",
  "contract_version": "v1",
  "source_commit_time": "2026-05-10T10:00:00Z"
}
```

更新前检查：

1. 确认目标 tag 已经在 `ai-mini-gateway` GitHub Releases 发布
2. 确认该 tag 的 commit 与 manifest 中 `commit` 一致
3. 确认 `contract_version` 没有意外变化

## 6. 准备 runtime

### 6.1 在线模式

默认方式是让脚本从 GitHub Release 下载当前平台资产：

```bash
pnpm --filter desktop prepare:ai-mini-gateway-runtime
```

脚本会：

1. 读取 `manifest.json`
2. 根据当前平台推导目标 asset 文件名
3. 从 GitHub Release 下载 asset
4. 解压出 binary
5. 复制到：
   `apps/desktop/resources/ai-mini-gateway/bin/`
6. 生成：
   `apps/desktop/resources/ai-mini-gateway/version.json`

### 6.2 离线模式

如果当前环境不方便直接从 GitHub 下载，可显式传入本地 asset 路径：

```bash
AI_MINI_GATEWAY_RELEASE_ASSET=/absolute/path/to/ai-mini-gateway_v0.1.0_darwin_arm64.tar.gz \
pnpm --filter desktop prepare:ai-mini-gateway-runtime
```

### 6.3 显式覆盖下载地址

如需走镜像或临时下载地址：

```bash
AI_MINI_GATEWAY_RELEASE_ASSET_URL=https://example.com/ai-mini-gateway_v0.1.0_darwin_arm64.tar.gz \
pnpm --filter desktop prepare:ai-mini-gateway-runtime
```

### 6.4 生成产物检查

执行后应检查：

1. `apps/desktop/resources/ai-mini-gateway/bin/` 下存在当前平台 binary
2. `apps/desktop/resources/ai-mini-gateway/version.json` 已生成
3. `version.json` 中的 `version`、`commit`、`source_url` 与预期一致

注意：

1. `bin/` 和 `version.json` 不提交 git
2. 这些都是打包前临时产物

## 7. 打包步骤

### 7.1 最小打包前检查

在正式打包前，至少执行：

```bash
pnpm --filter desktop typecheck
```

如果涉及 core 行为改动，建议再执行：

```bash
cd core && go test ./internal/localgateway/... ./internal/api/...
```

### 7.2 本地打包

按平台执行：

```bash
pnpm --filter desktop build:mac
```

```bash
pnpm --filter desktop build:win
```

```bash
pnpm --filter desktop build:linux
```

这些命令已经串联：

1. `pnpm build`
2. `pnpm build:core`
3. `pnpm prepare:ai-mini-gateway-runtime`
4. `electron-builder`

### 7.3 仅验证资源打包

如果只想先验证资源是否进入安装包，可执行：

```bash
pnpm --filter desktop build:unpack
```

然后检查输出目录中是否包含：

```text
resources/ai-mini-gateway/bin/ai-mini-gateway
resources/ai-mini-gateway/version.json
resources/ai-mini-gateway/manifest.json
```

## 8. 发布步骤

标准发布顺序如下：

1. 在 `ai-mini-gateway` 仓库确认目标 release 已存在
2. 在 `clash-for-ai` 更新 `manifest.json`
3. 运行 `pnpm --filter desktop prepare:ai-mini-gateway-runtime`
4. 执行必要测试和 typecheck
5. 执行目标平台打包命令
6. 本地验证安装包内置 runtime 正常启动
7. 提交 manifest 和相关代码变更
8. 发布新的 `clash-for-ai` 安装包

## 9. 验证清单

发布前至少确认以下项目：

- `manifest.json` 中的 `version` 和 `commit` 已更新
- `prepare-ai-mini-gateway-runtime` 成功执行
- `version.json` 已生成且内容正确
- 安装包内包含 `resources/ai-mini-gateway/bin/*`
- 桌面端启动后 `Local Gateway Runtime` 页面能读到正确 `version/commit/runtime_kind`
- 未设置 `LOCAL_GATEWAY_RUNTIME_EXECUTABLE` 时，仍可从安装包内置 binary 正常启动 runtime

## 10. 失败排查

### 10.1 下载失败

优先检查：

1. `manifest.json` 的 `version` 是否真实存在于 GitHub Release
2. 目标平台 asset 名称是否符合 release workflow 规则
3. 网络是否可访问 GitHub Release 地址

### 10.2 解压失败

优先检查：

1. 下载到的文件是否真的是对应平台 archive
2. macOS / Linux 是否存在 `tar`
3. Windows 是否可用 `powershell Expand-Archive`
4. 非 Windows zip 解压是否有 `unzip`

### 10.3 打包后 runtime 缺失

优先检查：

1. `pnpm prepare:ai-mini-gateway-runtime` 是否在打包前执行
2. `apps/desktop/electron-builder.yml` 的 `extraResources` 是否包含 `resources/ai-mini-gateway`
3. 产物目录中的 `bin/` 是否在打包前已生成

### 10.4 安装后 runtime 未启动

优先检查：

1. 安装包内 `resources/ai-mini-gateway/bin/` 是否真实存在 binary
2. 桌面端是否已把 bundled binary 路径注入到 `LOCAL_GATEWAY_RUNTIME_EXECUTABLE`
3. `Local Gateway Runtime` 页面里 `last_error` 的具体报错内容

## 11. 推荐发布口径

在变更说明中建议明确写出：

1. 本次 `clash-for-ai` 内置的 `ai-mini-gateway` 版本
2. 对应 commit
3. `contract_version`

示例：

```text
Bundled ai-mini-gateway runtime:
- version: v0.1.0
- commit: 2c5df19
- contract_version: v1
```
