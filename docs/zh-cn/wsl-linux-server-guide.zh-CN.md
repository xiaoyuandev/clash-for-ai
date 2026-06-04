# Relay Switch WSL / Linux Server 部署与使用说明

本文档说明如何在 `WSL` 或普通 `Linux server` 上部署 Relay Switch，并通过浏览器使用 web 管理界面。

## 1. 当前部署形态

当前 WSL / Linux server 方案由三部分组成：

1. `relay-switch-core`
2. 内嵌的 `ai-mini-gateway` runtime
3. `apps/web` 构建出的浏览器管理界面

安装完成后，默认会提供：

1. OpenAI-compatible 本地入口：`http://127.0.0.1:3456/v1`
2. Web 管理界面：`http://127.0.0.1:3456`
3. 本地 models gateway runtime：`http://127.0.0.1:3457/v1`

说明：

1. `3456` 是 Relay Switch 主入口
2. `3457` 是内嵌 local gateway runtime 端口
3. Web UI 与 API 共用 `3456`，`/api/*`、`/v1/*` 仍然由 core 处理

## 2. 前置要求

生产安装脚本默认从 GitHub Release 下载稳定版本。

目标机器至少需要：

1. `curl`
2. `tar`

如果你希望校验 SHA256，建议额外具备以下其一：

1. `sha256sum`
2. `shasum`

## 3. 一键安装

当前推荐命令：

```bash
curl -fsSL https://raw.githubusercontent.com/xiaoyuandev/relay-switch/main/scripts/install.sh | bash
```

默认行为：

1. 下载最新 release 的 Linux server 安装包
2. 解压其中的 `relay-switch-core`、`ai-mini-gateway` 和 web UI
3. 生成 `systemd --user` 服务
4. 自动启动 Relay Switch

说明：

1. `scripts/install.sh` 是生产默认安装脚本
2. `scripts/install-from-source.sh` 是开发安装脚本，只适合源码联调和未发布分支验证

安装完成后，访问：

```text
http://127.0.0.1:3456
```

如果你是在本机 Linux 桌面环境中使用，直接本地浏览器打开即可。

如果你是在远程 Linux server 上使用，请自行通过 SSH 端口转发或反向代理暴露访问入口。

## 4. 常用环境变量

安装脚本支持以下变量：

```bash
RELAY_SWITCH_VERSION=vX.Y.Z
RELAY_SWITCH_HTTP_PORT=3456
RELAY_SWITCH_LOCAL_GATEWAY_PORT=3457
RELAY_SWITCH_INSTALL_ROOT="$HOME/.local/share/relay-switch"
RELAY_SWITCH_DATA_DIR="$HOME/.local/share/relay-switch/data"
```

例如把主入口改到 `8080`：

```bash
curl -fsSL https://raw.githubusercontent.com/xiaoyuandev/relay-switch/main/scripts/install.sh | RELAY_SWITCH_HTTP_PORT=8080 bash
```

安装完成后的主入口会变成：

```text
http://127.0.0.1:8080/v1
```

如果你需要固定到某个已发布版本或执行回滚，可以显式指定：

```bash
curl -fsSL https://raw.githubusercontent.com/xiaoyuandev/relay-switch/main/scripts/install.sh | RELAY_SWITCH_VERSION=vX.Y.Z bash
```

回滚时同样使用这个方式，把 `RELAY_SWITCH_VERSION` 改成目标 release tag 即可。

## 5. 服务管理

默认会安装一个 `systemd --user` 服务：

```bash
systemctl --user status relay-switch
systemctl --user restart relay-switch
journalctl --user -u relay-switch -n 200 -f
```

同时也会生成一个辅助命令：

```bash
relay-switch start
relay-switch stop
relay-switch restart
relay-switch status
relay-switch logs
relay-switch run
```

说明：

1. `start / stop / restart / status / logs` 优先走 `systemd --user`
2. `run` 会以前台方式直接启动 core，适合调试

## 6. WSL 特别说明

### 6.1 WSL 里如何访问 UI

如果你在 Windows 上使用 WSL，通常可以直接从 Windows 浏览器访问：

```text
http://localhost:3456
```

如果端口不是默认值，请替换成你实际安装时使用的端口。

### 6.2 WSL 里的 systemd

较新的 WSL 已经支持 `systemd`，但不是所有环境都默认开启。

如果 `systemctl --user` 不可用，安装脚本不会中断，但会退回到手动启动模式。此时用：

```bash
relay-switch run
```

如果你希望 WSL 内也能使用 user service，请先开启该发行版的 `systemd`。

## 7. 首次使用流程

启动后，按下面顺序配置：

1. 打开 `http://127.0.0.1:3456`
2. 进入 `Providers`
3. 添加你的上游 provider
4. 在工具里把 Base URL 改成 `http://127.0.0.1:3456/v1`
5. API Key 填任意非空值，例如 `dummy`

例如：

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

## 8. 远程服务器访问方式

如果你是在远程 Linux server 部署，通常有两种方式：

### 方式一：SSH 端口转发

本地执行：

```bash
ssh -L 3456:127.0.0.1:3456 your-server
```

然后本地浏览器访问：

```text
http://127.0.0.1:3456
```

### 方式二：反向代理

把你的反向代理转发到：

```text
http://127.0.0.1:3456
```

注意：

1. 当前 core 默认绑定 `127.0.0.1`
2. 更安全的做法是继续保持本地绑定，再由 Nginx / Caddy 负责公网入口

## 9. 目录结构

默认安装后主要目录：

```text
~/.local/share/relay-switch/
  bin/
  data/
  web/
  release/
  relay-switch.env
```

说明：

1. `bin/` 放 `relay-switch-core` 和 `ai-mini-gateway`
2. `data/` 放 sqlite 和凭证文件
3. `web/` 是浏览器 UI 构建产物
4. `release/` 保留最近一次解压的 release 包内容

## 10. 升级方式

当前升级方式就是重新执行安装命令：

```bash
curl -fsSL https://raw.githubusercontent.com/xiaoyuandev/relay-switch/main/scripts/install.sh | bash
```

脚本会：

1. 下载最新 release 包
2. 覆盖安装二进制和 web 资源
3. 重启 `systemd --user` 服务

## 11. 排障

### 11.1 看服务日志

```bash
journalctl --user -u relay-switch -n 200 -f
```

### 11.2 健康检查

```bash
curl http://127.0.0.1:3456/health
```

预期返回：

```json
{"status":"ok","version":"1.1.0"}
```

### 11.3 检查 web UI 文件是否存在

```bash
ls ~/.local/share/relay-switch/web
```

### 11.4 检查 runtime binary 是否存在

```bash
ls ~/.local/share/relay-switch/bin
```

你应该能看到：

1. `relay-switch-core`
2. `ai-mini-gateway`

### 11.5 工具调用失败，但 UI 能打开

优先检查：

1. 你的工具是否填成了 `http://127.0.0.1:3456/v1`
2. `API Key` 是否至少是非空字符串
3. `Providers` 中是否已经有激活的 provider
4. provider 的 `Base URL` 是否写对

## 12. 开发版安装脚本

如果你是本地开发、联调或验证未发布分支，可以继续使用源码构建脚本：

```bash
curl -fsSL https://raw.githubusercontent.com/xiaoyuandev/relay-switch/main/scripts/install-from-source.sh | bash
```

这个脚本会直接拉源码并本机构建，因此更适合开发环境，不适合作为生产默认入口。
