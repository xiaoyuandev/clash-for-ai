# WSL / Linux 使用文档

## 文档目的

这份文档用于说明 Clash for AI 在以下场景下的实际使用方式：

1. `WSL`
2. `Linux server`
3. `macOS` 作为近似 Linux 的本地验证环境

这份文档只讨论 `Go core + Web / PWA` 的补充入口使用方式。

必须再次强调：

1. `Electron` 仍是本地主入口
2. `Web / PWA` 是 WSL 和 Linux server 的补充入口
3. 本文档不是桌面端替代方案说明

## 适用场景

这份文档适合以下用户：

1. 主要在 `WSL` 中运行 `Codex CLI`、`Claude Code` 或其他 CLI 工具
2. 在 `Linux server`、云主机、NAS、家庭服务器中运行 Clash for AI
3. 想先在 `macOS` 上用近似 Linux 的方式验证 `Go core + Web` 工作流

## 总体原则

在 WSL / Linux 场景下，推荐的使用模式是：

1. 在目标环境中直接运行 `clash-for-ai-core`
2. 通过浏览器访问该实例暴露的 Web 管理页面
3. 在这个 Web 页面里管理 `Providers`、`Models`、`Logs`、`Tools`

这个模式的核心好处是：

1. 配置文件写入发生在 `core` 实际运行的那台机器上
2. `Tools` 页的一键配置会作用于目标环境本机
3. 不需要桌面端跨环境替 WSL 或 Linux server 写配置

## 与桌面端的关系

你需要把两个入口分清楚：

### Electron 桌面端负责

1. 本地 core 生命周期
2. 托盘
3. 自启动
4. 桌面窗口与本地桌面体验
5. 桌面应用更新

### Web / PWA 补充入口负责

1. 浏览器中的管理页面
2. WSL 环境下的配置与查看
3. Linux server 环境下的配置与查看
4. 无头环境中的补充管理体验

## 一、运行前准备

## 1. 基础要求

你至少需要：

1. 一个可运行 `clash-for-ai-core` 的环境
2. 浏览器
3. 上游 Provider 的 `Base URL` 和 `API Key`

如果你准备验证 `Tools` 一键配置，还需要至少安装其中一种：

1. `Codex CLI`
2. `Claude Code`

## 2. 当前默认配置

当前 core 的主要配置来源是环境变量，默认值如下：

1. `HTTP_PORT=3456`
2. `CORE_DATA_DIR=./data`
3. `LOCAL_GATEWAY_RUNTIME_HOST=127.0.0.1`
4. `LOCAL_GATEWAY_RUNTIME_PORT=3457`
5. `LOCAL_GATEWAY_RUNTIME_KIND=ai-mini-gateway`
6. `LOCAL_GATEWAY_RUNTIME_EXECUTABLE=` 空

这些值来自当前实现的 [core/internal/config/config.go](/Users/yuanjunliang/Documents/workspace/profile/clash-for-ai/core/internal/config/config.go:1)。

这里最关键的一点是：

1. `LOCAL_GATEWAY_RUNTIME_EXECUTABLE` 默认没有值
2. 如果它没有被配置，`Go core` 不会自动拉起 local gateway runtime
3. 这就是启动日志里 `runtime executable is not configured` 的原因

## 3. 启动 local gateway 还需要配置什么

如果你希望 `Go core + Web / PWA` 这条路径在启动 core 时自动拉起 local gateway，至少要额外配置：

1. `LOCAL_GATEWAY_RUNTIME_EXECUTABLE`

建议一起显式配置：

1. `LOCAL_GATEWAY_RUNTIME_KIND=ai-mini-gateway`
2. `LOCAL_GATEWAY_RUNTIME_HOST=127.0.0.1`
3. `LOCAL_GATEWAY_RUNTIME_PORT=3457`
4. `LOCAL_GATEWAY_RUNTIME_DATA_DIR`

推荐完整配置示例：

```text
HTTP_PORT=3456
CORE_DATA_DIR=/opt/clash-for-ai/data
LOCAL_GATEWAY_RUNTIME_KIND=ai-mini-gateway
LOCAL_GATEWAY_RUNTIME_EXECUTABLE=/opt/clash-for-ai/bin/ai-mini-gateway
LOCAL_GATEWAY_RUNTIME_HOST=127.0.0.1
LOCAL_GATEWAY_RUNTIME_PORT=3457
LOCAL_GATEWAY_RUNTIME_DATA_DIR=/opt/clash-for-ai/data/local-gateway
```

## 3.1 `ai-mini-gateway` 二进制从哪里来

当前文档已经说明了要配置：

1. `LOCAL_GATEWAY_RUNTIME_EXECUTABLE`

但这条配置成立的前提是：你本机已经有可用的 `ai-mini-gateway` 二进制。

目前推荐的准备方式有两种：

### 方式 A：你已经有本地源码仓库

如果你已经单独拉取了 `ai-mini-gateway` 源码仓库，那么最直接的方式就是：

1. 在该仓库中构建出二进制
2. 将构建产物放到固定目录
3. 把 `LOCAL_GATEWAY_RUNTIME_EXECUTABLE` 指向那个绝对路径

例如你当前本机的测试路径就是：

```text
/Users/yuanjunliang/Documents/workspace/profile/ai-mini-gateway/bin/ai-mini-gateway
```

### 方式 B：你使用已经准备好的 release binary

如果不是源码联调，而是部署到 WSL / Linux server，更推荐：

1. 使用已经打好的 `ai-mini-gateway` release binary
2. 放到固定目录，例如：

```text
/opt/clash-for-ai/bin/ai-mini-gateway
```

然后把：

```text
LOCAL_GATEWAY_RUNTIME_EXECUTABLE=/opt/clash-for-ai/bin/ai-mini-gateway
```

写进环境变量配置。

## 3.2 推荐放置位置

为了让文档、systemd 和环境变量模板保持一致，推荐路径如下：

### WSL / Linux server

```text
/opt/clash-for-ai/bin/ai-mini-gateway
```

### 本地开发 / macOS 模拟 Linux server

如果你本地已有联调仓库，可以直接使用仓库内构建产物，例如：

```text
/Users/yuanjunliang/Documents/workspace/profile/ai-mini-gateway/bin/ai-mini-gateway
```

或者你也可以自己统一放到：

```text
$HOME/clash-for-ai/bin/ai-mini-gateway
```

关键不是必须放哪，而是：

1. 路径要稳定
2. 路径要是绝对路径
3. `LOCAL_GATEWAY_RUNTIME_EXECUTABLE` 要和它保持一致

## 3.3 推荐检查方式

在真正启动 `clash-for-ai-core` 前，先确认目标二进制存在并可执行。

例如：

```bash
ls -l /opt/clash-for-ai/bin/ai-mini-gateway
```

或本地开发时：

```bash
ls -l /Users/yuanjunliang/Documents/workspace/profile/ai-mini-gateway/bin/ai-mini-gateway
```

如果需要，补执行权限：

```bash
chmod +x /path/to/ai-mini-gateway
```

## 3.4 和 `LOCAL_GATEWAY_RUNTIME_EXECUTABLE` 的关系

这两者是一一对应的：

1. `ai-mini-gateway` 二进制放在哪里
2. `LOCAL_GATEWAY_RUNTIME_EXECUTABLE` 就必须写成那个绝对路径

如果路径写错，会出现：

1. core 能启动
2. local gateway 拉起失败
3. 启动日志里出现 bootstrap 相关错误

所以这条配置不只是“填一个值”，而是“把 runtime 二进制路径明确告诉 core”。

## 4. 这些配置应该写在哪里

按你的运行方式不同，建议写在不同位置。

### 方式 A：直接写在当前 shell 命令里

适合：

1. WSL 临时验证
2. Linux 临时验证
3. macOS 本地模拟 Linux server

### 方式 B：写进 `systemd` 的环境文件

适合：

1. Linux server 长期运行
2. 希望开机自启

推荐位置：

```text
/etc/clash-for-ai/clash-for-ai.env
```

可直接参考：

1. [.docs/systemd-environment-template.env](/Users/yuanjunliang/Documents/workspace/profile/clash-for-ai/.docs/systemd-environment-template.env:1)
2. [.docs/systemd-deployment-template.md](/Users/yuanjunliang/Documents/workspace/profile/clash-for-ai/.docs/systemd-deployment-template.md:1)

### 方式 C：写进 `~/.zshrc` 或 `~/.bashrc`

适合：

1. 你长期只在同一个用户下手工运行 core
2. 不打算用 systemd

如果走这种方式，重新打开 shell 后再启动 core。

## 5. 推荐目录习惯

建议在 WSL / Linux 环境中给 core 一个固定工作目录，例如：

```text
~/clash-for-ai/
```

然后把运行产生的数据放在：

```text
~/clash-for-ai/data/
```

这样更方便持久化和排查问题。

## 二、WSL 使用方式

## 1. 推荐模型

在 WSL 场景中，推荐模型是：

1. `clash-for-ai-core` 运行在 WSL
2. 你的 CLI 工具也运行在同一个 WSL
3. 浏览器通过宿主机访问 WSL 暴露的 Web 页面

这意味着：

1. `~/.codex`
2. `~/.claude`
3. `~/.clash-for-ai`

这些路径都指向 WSL 的 Linux 用户目录，而不是 Windows 用户目录。

## 2. 启动 core

在 WSL 终端中进入你的项目目录，然后运行：

```bash
cd /path/to/clash-for-ai/core
go run ./cmd/clash-for-ai-core
```

如果你已经构建了二进制，也可以直接运行：

```bash
./clash-for-ai-core
```

如果要显式指定端口和数据目录，可以这样：

```bash
HTTP_PORT=3456 \
CORE_DATA_DIR="$HOME/clash-for-ai/data" \
LOCAL_GATEWAY_RUNTIME_KIND=ai-mini-gateway \
LOCAL_GATEWAY_RUNTIME_EXECUTABLE="$HOME/clash-for-ai/bin/ai-mini-gateway" \
LOCAL_GATEWAY_RUNTIME_PORT=3457 \
LOCAL_GATEWAY_RUNTIME_DATA_DIR="$HOME/clash-for-ai/data/local-gateway" \
go run ./cmd/clash-for-ai-core
```

如果你当前没有配置 `LOCAL_GATEWAY_RUNTIME_EXECUTABLE`，那么：

1. core 本身仍然会启动
2. Web 页面仍然可以访问
3. 但 local gateway 不会自动拉起
4. 启动日志会打印 `runtime executable is not configured`

## 3. 访问 Web 页面

默认情况下，core 监听：

```text
http://127.0.0.1:3456
```

如果浏览器和 WSL 在同一台机器上，可以直接访问：

```text
http://127.0.0.1:3456
```

如果你后续改了端口，就用实际端口替换。

## 4. 在 Web 页面中配置 Provider

进入 Web 页面后，推荐按这个顺序配置：

1. `Providers`
2. `Models`
3. `Tools`

其中：

1. `Providers` 用于添加和切换上游
2. `Models` 用于本地网关模型来源管理
3. `Tools` 用于生成命令预设，以及执行 `Codex CLI` / `Claude Code` 的一键配置

## 5. 验证 Tools 一键配置是否写到 WSL

在 WSL 的 Web 页面中点击 `Codex CLI` 或 `Claude Code` 的一键配置后，可以在 WSL 终端中检查：

```bash
ls -la ~/.codex
ls -la ~/.claude
ls -la ~/.clash-for-ai/tool-backups
```

如果是 `Codex CLI`，重点看：

```bash
cat ~/.codex/config.toml
cat ~/.codex/auth.json
```

如果是 `Claude Code`，重点看：

```bash
cat ~/.claude/settings.json
```

这些文件应该位于 WSL 的 Linux 用户目录中。

## 三、Linux server 使用方式

## 1. 推荐模型

在 Linux server 场景中，推荐模型是：

1. `clash-for-ai-core` 直接运行在目标 Linux 机器上
2. 浏览器远程访问这台机器上的 Web 页面
3. 所有工具配置都作用于这台 Linux 机器自身

## 2. 启动 core

最简单的方式：

```bash
cd /path/to/clash-for-ai/core
go run ./cmd/clash-for-ai-core
```

如果使用固定数据目录：

```bash
HTTP_PORT=3456 \
CORE_DATA_DIR="/opt/clash-for-ai/data" \
LOCAL_GATEWAY_RUNTIME_KIND=ai-mini-gateway \
LOCAL_GATEWAY_RUNTIME_EXECUTABLE="/opt/clash-for-ai/bin/ai-mini-gateway" \
LOCAL_GATEWAY_RUNTIME_PORT=3457 \
LOCAL_GATEWAY_RUNTIME_DATA_DIR="/opt/clash-for-ai/data/local-gateway" \
go run ./cmd/clash-for-ai-core
```

## 3. 后台运行建议

如果你不想一直占用前台终端，可以使用：

1. `tmux`
2. `screen`
3. `systemd`

当前文档先给出最小可行的 `tmux` 方式：

```bash
tmux new -s clash-for-ai
cd /path/to/clash-for-ai/core
HTTP_PORT=3456 \
CORE_DATA_DIR="/opt/clash-for-ai/data" \
LOCAL_GATEWAY_RUNTIME_KIND=ai-mini-gateway \
LOCAL_GATEWAY_RUNTIME_EXECUTABLE="/opt/clash-for-ai/bin/ai-mini-gateway" \
LOCAL_GATEWAY_RUNTIME_PORT=3457 \
LOCAL_GATEWAY_RUNTIME_DATA_DIR="/opt/clash-for-ai/data/local-gateway" \
go run ./cmd/clash-for-ai-core
```

之后：

```bash
Ctrl+b d
```

将会话放到后台。

## 4. 访问 Web 页面

当前实现默认绑定：

```text
127.0.0.1
```

这意味着如果你要从其他机器访问，需要自行处理网络转发，例如：

1. SSH 本地端口转发
2. 反向代理

最简单的 SSH 方式：

```bash
ssh -L 3456:127.0.0.1:3456 user@your-server
```

然后在本地浏览器访问：

```text
http://127.0.0.1:3456
```

这个方式最稳妥，也符合当前默认只监听本机回环地址的设计。

## 5. 验证 Tools 一键配置是否写到 server

在 Linux server 的 Web 页面中点击一键配置后，在 server 上检查：

```bash
ls -la ~/.codex
ls -la ~/.claude
ls -la ~/.clash-for-ai/tool-backups
```

确认：

1. 工具配置文件写到了 Linux server 用户目录
2. 备份目录也写到了 Linux server 用户目录

## 四、PWA 使用方式

## 1. PWA 是什么

当前 `apps/web` 已经带有：

1. `manifest.webmanifest`
2. 图标
3. `vite-plugin-pwa`

因此在兼容浏览器中，你可以把 Web 页面安装成 PWA。

## 2. PWA 的正确理解

必须明确：

1. PWA 只是 Web 页面的一种安装形态
2. PWA 不是 Electron 桌面端替代品
3. PWA 不负责本地 Go core 生命周期

PWA 可以提供：

1. 独立窗口
2. 快捷入口
3. 更像应用的浏览器体验

PWA 不提供：

1. 托盘
2. 自启动
3. 桌面应用更新
4. 本地进程拉起

## 3. 如何安装 PWA

以 Chrome 为例：

1. 打开 `http://127.0.0.1:3456`
2. 等页面加载完成
3. 使用地址栏或菜单中的安装选项
4. 安装后从独立窗口打开

安装后本质上仍然是同一个 Web 页面，只是入口更像应用。

## 五、macOS 作为近似 Linux 的验证环境

你当前开发环境是 macOS。

虽然 macOS 不等于 Linux，但它很适合作为“近似 Linux 本地验证环境”来测试以下内容：

1. `Go core + Web` 启动链路
2. 浏览器访问 Web 页面
3. PWA 安装形态
4. `Tools` 页 API 行为
5. `Codex CLI` / `Claude Code` 本地配置写入逻辑

原因是：

1. 当前 `Go core` 是跨平台的
2. Web 管理页不依赖 Electron 才能工作
3. `~/.codex`、`~/.claude`、`~/.clash-for-ai` 这类路径语义在 macOS 上同样成立

## 1. macOS 下的推荐验证步骤

在 macOS 上你可以按这个顺序验证：

1. 在项目根目录安装前端依赖
2. 运行 `clash-for-ai-core`
3. 打开 `apps/web`
4. 添加一个测试 Provider
5. 在 `Tools` 页点击一键配置
6. 检查本机用户目录是否生成配置和备份

建议命令：

```bash
cd /path/to/clash-for-ai
pnpm install
```

终端 1：

```bash
cd core
HTTP_PORT=3456 \
CORE_DATA_DIR="$HOME/clash-for-ai/data" \
LOCAL_GATEWAY_RUNTIME_KIND=ai-mini-gateway \
LOCAL_GATEWAY_RUNTIME_EXECUTABLE="$HOME/clash-for-ai/bin/ai-mini-gateway" \
LOCAL_GATEWAY_RUNTIME_PORT=3457 \
LOCAL_GATEWAY_RUNTIME_DATA_DIR="$HOME/clash-for-ai/data/local-gateway" \
go run ./cmd/clash-for-ai-core
```

结合你当前本机目录，也可以直接使用下面这条可运行命令：

```bash
cd /Users/yuanjunliang/Documents/workspace/profile/clash-for-ai/core && \
HTTP_PORT=3456 \
CORE_DATA_DIR="/Users/yuanjunliang/Documents/workspace/profile/clash-for-ai/core/data" \
LOCAL_GATEWAY_RUNTIME_KIND=ai-mini-gateway \
LOCAL_GATEWAY_RUNTIME_EXECUTABLE="/Users/yuanjunliang/Documents/workspace/profile/ai-mini-gateway/bin/ai-mini-gateway" \
LOCAL_GATEWAY_RUNTIME_HOST=127.0.0.1 \
LOCAL_GATEWAY_RUNTIME_PORT=3457 \
LOCAL_GATEWAY_RUNTIME_DATA_DIR="/Users/yuanjunliang/Documents/workspace/profile/clash-for-ai/core/data/local-gateway" \
go run ./cmd/clash-for-ai-core
```

如果 local gateway 正常拉起，终端里应重点看到类似日志：

```text
[core] http api listening on http://127.0.0.1:3456
[local-gateway] started successfully on http://127.0.0.1:3457
```

终端 2：

```bash
cd /path/to/clash-for-ai
pnpm --filter web dev
```

结合你当前本机目录，也可以直接使用下面这条可运行命令：

```bash
cd /Users/yuanjunliang/Documents/workspace/profile/clash-for-ai && \
pnpm --filter web dev
```

然后访问：

```text
http://127.0.0.1:4173
```

如果你想验证构建后的 PWA 形态：

```bash
pnpm --filter web build
pnpm --filter web preview
```

## 2. macOS 下验证文件写入

验证 `Codex CLI`：

```bash
cat ~/.codex/config.toml
cat ~/.codex/auth.json
```

验证 `Claude Code`：

```bash
cat ~/.claude/settings.json
```

验证备份目录：

```bash
find ~/.clash-for-ai/tool-backups -maxdepth 3 -print
```

这虽然不是 Linux，但足够验证：

1. Go core 是否正确写文件
2. Tools API 是否正常
3. 前端是否已切到新的 Go core 配置链路

## 六、推荐验证顺序

如果你要完整验证 WSL / Linux 方案，建议按这个顺序：

1. 先在 macOS 上验证本地 `Go core + Web + Tools` 流程
2. 再在 WSL 中验证配置文件是否落到 WSL 用户目录
3. 最后在 Linux server 中验证 SSH 转发 + 浏览器访问流程

这样最省时间，也最容易分离问题。

## 七、当前已验证能力

结合当前已落地代码，本轮已验证：

1. `go test ./...` 通过
2. `pnpm --filter web build` 通过
3. `pnpm --filter desktop typecheck` 通过

这说明：

1. Go core 的 `tooling` 模块已编译通过
2. Web 端 `Tools` 页已经切到 Go core API
3. Electron 主进程旧的工具过渡实现已清理

## 八、已知限制

当前你需要知道这些限制：

1. `Electron` 仍是本地主入口，Web/PWA 不是桌面端替代品
2. Linux server 默认绑定 `127.0.0.1`，远程访问建议走 SSH 转发
3. `Cherry Studio` 的 deep link 唤起仍是桌面端专属能力
4. 文档现在主要覆盖单实例、单用户、自管环境
5. 如果没有配置 `LOCAL_GATEWAY_RUNTIME_EXECUTABLE`，core 不会自动拉起 local gateway

## 九、推荐后续动作

如果你接下来要继续推进，建议顺序是：

1. 先在 macOS 上按本文档跑一遍完整验证
2. 再去 WSL 验证真实的 `~/.codex` / `~/.claude` 写入
3. 最后补一版 `systemd` 运行模板文档

这样可以最快把 WSL / Linux 使用链路从“架构成立”推进到“真实可用”。
