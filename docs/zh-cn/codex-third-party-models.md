# 在 Codex 里使用第三方大模型

本文档说明如何通过 Relay Switch 在 Codex CLI 里使用 DeepSeek、Qwen、GLM、MiniMax 等第三方大模型，并解释这些模型如何出现在 Codex 的 `/model` 菜单里。

先说结论：

1. Relay Switch 一开始其实就可以支持 Codex 使用第三方大模型。
2. 只要上游兼容 OpenAI 接口，或者能通过 Local Gateway 适配成 OpenAI-compatible 接口，Codex 就可以把请求发给 Relay Switch，再由 Relay Switch 转发到上游。
3. 之前的限制不是“第三方模型不能用”，而是这些模型不会自然出现在 Codex 的 `/model` 菜单里。
4. 现在 Relay Switch 补齐了 Codex 模型目录配置，可以把你配置的第三方模型显示到 `/model` 菜单中。

也就是说，这里要分清两件事：

1. **能不能使用第三方模型**：由 Codex 是否连到 Relay Switch，以及 Relay Switch 后面的 Provider / Local Gateway 是否能转发决定。
2. **能不能在 `/model` 菜单里看到第三方模型**：由 Relay Switch 的 Codex 模型列表和模型目录开关决定。

## 1. 工作原理

Relay Switch 是一个运行在本地的 AI API 网关。Codex 不需要直接连接每一个模型服务商，而是统一连接本地地址：

```text
http://127.0.0.1:3456/v1
```

之后请求流向是：

```text
Codex CLI -> Relay Switch 本地网关 -> 当前启用的 Provider 或 Local Gateway -> 实际大模型上游
```

这个结构带来两个好处：

1. Codex 侧可以一直使用同一个 OpenAI-compatible endpoint。
2. 切换中转站、官方模型服务或本地 Local Gateway 时，只需要在 Relay Switch 里切换，不需要反复修改 Codex 配置。

如果 `/model` 菜单里没有 DeepSeek、Qwen 或 MiniMax，不代表它们不能用。它只说明 Codex 当前的模型菜单目录不知道 Relay Switch 里有哪些自定义模型。

## 2. 配置模型来源

先在 Relay Switch 里准备一个能响应请求的模型来源。你可以选择 Provider 或 Local Gateway。

### 方式一：使用中转 API Provider

如果你已经有 OpenAI-compatible 中转 API，例如基于 `new-api`、`one-api`、`sub2api` 或其他 relay 服务，可以直接在 `Providers` 页面添加。

需要填写：

```text
Name
Base URL
API Key
```

建议：

1. `Base URL` 通常填写到 `/v1`，例如 `https://example.com/v1`。
2. 保存后先执行模型检测或健康检查，确认 Relay Switch 能访问上游。
3. 在 `Providers` 页面启用这个 Provider。

这种方式适合：

1. 你的上游已经聚合了多个模型。
2. 上游本身兼容 OpenAI API。
3. 你只是希望 Codex 固定连接 Relay Switch，由 Relay Switch 负责切换上游。

### 方式二：使用 Local Gateway 接入原生模型

如果你想直接接入 DeepSeek、MiniMax、Qwen、GLM 等模型厂商官方 API，建议走 `Models` 页面和 Local Gateway。

操作路径：

1. 打开 `Models` 页面。
2. 新增模型来源。
3. 选择 `Provider Type`，例如 `openai-compatible`。
4. 填写 `Base URL`、`API Key` 和模型 ID。
5. 启用该模型来源。
6. 同步到 Local Gateway。
7. 回到 `Providers` 页面，启用 `Local Gateway`。

这种方式适合：

1. 你想直接接入模型厂商官方 API。
2. 不同模型来源的协议、模型列表或能力不完全一致。
3. 你希望 Relay Switch 在本地把这些模型统一整理成一个 OpenAI-compatible 入口。

## 3. 配置 Codex 连接 Relay Switch

模型来源准备好之后，需要让 Codex CLI 指向 Relay Switch 的本地入口。

### 推荐方式：在 Tools 页面一键配置

打开 Relay Switch 的 `Tools` 页面，找到 `Codex CLI`，执行一键配置。

Relay Switch 会更新 Codex 的本地配置：

```text
~/.codex/config.toml
~/.codex/auth.json
```

核心效果是让 Codex 使用本地 OpenAI-compatible endpoint：

```toml
model_provider = "OpenAI"
experimental_bearer_token = "dummy"

[model_providers.OpenAI]
name = "OpenAI"
base_url = "http://127.0.0.1:3456/v1"
wire_api = "responses"
requires_openai_auth = true
```

这里的 `dummy` 只是 Codex 到 Relay Switch 本地网关之间的占位 token。真正的上游 API Key 仍然保存在 Relay Switch 的 Provider 或 Models 配置里。

### 临时方式：使用当前 shell 环境变量

如果你只想临时测试，也可以在当前终端里设置 OpenAI-compatible 环境变量，然后从同一个终端启动 Codex：

```bash
export OPENAI_BASE_URL="http://127.0.0.1:3456/v1"
export OPENAI_API_KEY="dummy"
codex
```

这种方式只影响当前 shell 会话。关闭终端后，环境变量不会自动保留。

## 4. 不进入 `/model` 菜单也可以使用

这是最容易误解的地方。

Relay Switch 早期已经能支持第三方模型，因为 Codex 可以直接把某个模型名作为请求里的 `model` 发给 Relay Switch。只要 Relay Switch 当前启用的 Provider 或 Local Gateway 能识别这个模型 ID，请求就可以正常转发。

例如你已经在 Relay Switch 后面配置好了 `deepseek-chat`，可以直接启动：

```bash
codex -m deepseek-chat
```

也可以在 `~/.codex/config.toml` 顶层设置默认模型：

```toml
model = "deepseek-chat"
```

然后直接启动：

```bash
codex
```

这种方式的特点是：

1. 可以使用第三方模型。
2. 不依赖 Codex 的 `/model` 菜单。
3. 需要你自己记住准确的模型 ID。
4. 如果要切换模型，需要重新指定 `-m` 或修改配置里的 `model`。

所以，之前第三方模型并不是不能用，只是体验不完整：用户需要手动输入模型名，而不是在 `/model` 菜单里直接选择。

## 5. 让第三方模型出现在 `/model` 菜单里

如果你希望 Codex 的 `/model` 菜单显示第三方模型，需要在 Relay Switch 里启用 Codex 模型目录功能。

操作流程：

1. 打开 `Providers` 页面。
2. 选择当前要给 Codex 使用的 Provider。
3. 进入 `模型映射` 区域。
4. 切换到 `Codex 模型列表`。
5. 打开 `启用第三方模型`。
6. 按需打开 `隐藏官方大模型`。
7. 添加你希望显示到 `/model` 菜单里的模型名。

示例模型名：

```text
deepseek-chat
deepseek-reasoner
qwen3-coder
glm-4.5
MiniMax-M2.7
```

这里填写的模型名应该和上游实际支持的模型 ID 一致。

如果你启用的是普通中转 API Provider，这些模型名应该存在于该中转 API 的模型列表里。

如果你启用的是 `Local Gateway`，这些模型名应该已经在 `Models` 页面对应 source 中配置并同步。

启用后，Relay Switch 会维护这些文件：

```text
~/.codex/relay-switch-models.json
~/.codex/relay-switch-model-catalog.json
```

同时会在 `~/.codex/config.toml` 中写入模型目录配置：

```toml
model_catalog_json = "/path/to/.codex/relay-switch-model-catalog.json"
```

如果当前 Provider 的 Codex 模型列表里有启用的模型，Relay Switch 还会把第一个启用模型同步为 Codex 的默认 `model`。

## 6. 启动并切换模型

完成配置后，启动 Codex：

```bash
codex
```

进入 Codex 后输入：

```text
/model
```

你应该可以在菜单里看到 Relay Switch 注入的第三方模型。选择模型后，Codex 会继续请求 Relay Switch 本地网关，Relay Switch 再转发到当前启用的 Provider 或 Local Gateway。

如果你刚刚修改过 Codex 模型列表，建议重启 Codex。Codex 通常不会在运行中自动刷新模型目录。

## 7. DeepSeek 配置建议

如果你要使用 DeepSeek，推荐优先走 Local Gateway。

原因是 Codex 当前更适合连接 OpenAI Responses API 风格的入口，而 DeepSeek 官方 OpenAI-compatible 接口主要是 Chat Completions 风格。Local Gateway 可以在本地负责必要的协议适配。

推荐配置：

```text
Models 页面:
  Provider Type: openai-compatible
  Base URL: https://api.deepseek.com
  Model IDs: deepseek-chat, deepseek-reasoner

Providers 页面:
  启用 Local Gateway
  Codex 模型列表添加 deepseek-chat / deepseek-reasoner
  打开启用第三方模型
```

配置后重启 Codex，在 `/model` 菜单里选择 DeepSeek 模型即可。

如果只是临时测试，也可以跳过菜单配置，直接运行：

```bash
codex -m deepseek-chat
```

## 8. 常见问题

### `/model` 菜单里没有第三方模型

先检查：

1. `Providers` 页面当前启用的 Provider 是否正确。
2. 该 Provider 的 `Codex 模型列表` 是否添加了模型名。
3. `启用第三方模型` 是否已经打开。
4. 模型名是否处于启用状态，且不是空值。
5. 修改模型列表后是否重启了 Codex。

模型列表更新后，需要重启 Codex 才能刷新菜单。

### 菜单里有模型，但请求失败

先检查：

1. 当前启用的 Provider 或 Local Gateway 是否真的支持该模型 ID。
2. API Key 是否正确。
3. Base URL 是否正确。
4. 如果走 Local Gateway，`Models` 页面是否已经同步。
5. Relay Switch 的 `Logs` 页面里上游返回了什么错误。

模型出现在 `/model` 菜单里，只代表 Codex 能选择它，不代表上游一定能成功响应。

### 直接 `codex -m xxx` 能用，但 `/model` 菜单看不到

这说明请求转发链路是通的，问题只在模型目录配置。

重点检查：

1. `启用第三方模型` 是否打开。
2. `Codex 模型列表` 是否添加了同一个模型名。
3. `~/.codex/config.toml` 里是否存在 `model_catalog_json`。
4. 修改后是否重启 Codex。

### `/model` 菜单里只想显示第三方模型

在 `Providers` 页面进入当前 Provider 的 `Codex 模型列表`，打开 `隐藏官方大模型`。

这样 Relay Switch 会保留模型目录文件，但把官方模型条目标记为隐藏，只把你配置的第三方模型展示出来。

### 不想影响原来的 Codex 配置怎么办

Relay Switch 一键配置会修改 `~/.codex/config.toml` 和 `~/.codex/auth.json`，并在操作前创建备份。

如果你只是临时测试，可以优先使用环境变量方式：

```bash
export OPENAI_BASE_URL="http://127.0.0.1:3456/v1"
export OPENAI_API_KEY="dummy"
codex -m deepseek-chat
```

如果已经使用了一键配置，也可以在 `Tools` 页面恢复最近一次备份。

## 9. 推荐流程

最常见的完整流程是：

1. 在 `Providers` 页面添加中转 API，或在 `Models` 页面添加原生模型来源。
2. 启用对应 Provider，或启用 `Local Gateway`。
3. 在 `Tools` 页面把 Codex CLI 配置到 Relay Switch 本地入口。
4. 先用 `codex -m 模型名` 验证第三方模型能否正常响应。
5. 在当前 Provider 的 `Codex 模型列表` 添加这些模型名。
6. 打开 `启用第三方模型`，按需打开 `隐藏官方大模型`。
7. 重启 Codex，输入 `/model`，从菜单中选择第三方模型。
8. 如果请求失败，到 Relay Switch 的 `Logs` 页面查看上游错误。

这样可以把模型来源、Provider 切换和错误排查都留在 Relay Switch 里，让 Codex 只负责作为稳定的编码入口。
