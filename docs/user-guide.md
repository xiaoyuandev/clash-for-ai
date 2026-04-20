# Clash for AI 使用教程

## 1. 这个工具解决什么问题

Clash for AI 是一个本地 AI API 代理工具。

它的作用是：

1. 你只需要在自己喜欢的编程工具里配置一次本地地址。
2. 后续切换不同 Provider，不需要再去每个工具里重复改配置。
3. 你可以在 Clash for AI 里统一管理 Provider、查看健康状态、排查请求问题。

---

## 2. 使用前你需要知道什么

这个工具分成两部分：

1. 桌面端管理界面
2. 本地 API Gateway

你的编程工具不会直接连接远程 Provider，而是先连接本地 Gateway，再由 Clash for AI 转发到你在桌面端配置好的 Provider。

---

## 3. 正常使用流程

### 第一步：启动 Clash for AI

启动桌面应用后，Clash for AI 会在本地启动一个 API 服务。

默认本地地址通常是：

```text
http://127.0.0.1:3456/v1
```

如果 `3456` 端口被占用，程序会自动尝试下一个可用端口。

你可以在两个位置确认当前实际地址：

1. 桌面界面顶部的 `connected api base`
2. 启动终端中的 `[core] selected port ...`

例如：

```text
[core] selected port 3457, api base http://127.0.0.1:3457
```

那么你的工具里应该填写：

```text
http://127.0.0.1:3457/v1
```

---

### 第二步：在 Clash for AI 中添加 Provider

在桌面端 `Providers` 页面中填写：

1. `Name`
2. `Base URL`
3. `API Key`

例如：

```text
Name: kocodex
Base URL: https://kocodex.link/v1
API Key: sk-xxxx
```

说明：

1. `Name` 只是方便你自己识别。
2. `Base URL` 填远程 Provider 的真实地址。
3. `API Key` 填这个 Provider 提供给你的密钥。
4. 不需要手动配置 `Auth mode`，系统会自动处理认证头。

添加成功后，Provider 会出现在列表中。

如果有多个 Provider，你可以点击 `Activate` 切换当前生效的 Provider。

---

### 第三步：在你的编程工具里接入本地代理

无论你使用什么工具，核心原则都一样：

1. `Base URL` 填 Clash for AI 的本地地址
2. `API Key` 可以填任意非空字符串

原因：

1. 实际转发时，Clash for AI 会使用你在桌面端保存的远程 Provider 密钥。
2. 客户端发过来的认证头会被代理层自动处理。
3. 所以工具侧通常不需要再填写真实的远程密钥。

推荐写法：

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

如果实际端口不是 `3456`，请替换成当前运行中的端口。

---

## 4. 在常见工具中如何配置

不同工具的字段名称可能不一样，但本质上都在填写两项：

1. API Base URL
2. API Key

### 4.1 OpenAI-compatible 工具

如果某个工具支持自定义 OpenAI API 地址，通常这样配：

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

适用范围通常包括：

1. AI 编程助手
2. 聊天客户端
3. VS Code / JetBrains 插件
4. 自己写的脚本

---

### 4.2 Claude Code / 类 Claude 工具

如果工具支持自定义 OpenAI-compatible endpoint，也可以直接接到 Clash for AI。

仍然建议填写：

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

注意：

1. 并不是所有 Claude 类工具都允许手动改 endpoint。
2. 如果某个工具只能连接官方接口，Clash for AI 就无法接管它。

---

### 4.3 Cursor / Cherry Studio / Chatbox / 其他桌面客户端

只要该工具支持：

1. 自定义 API 地址
2. 自定义 API Key

就按下面方式填写：

```text
API Base: http://127.0.0.1:3456/v1
API Key: dummy
```

如果工具还要求选择模型，直接选择当前活跃 Provider 支持的模型名即可。

---

### 4.4 自己写代码时如何使用

#### JavaScript / TypeScript

```ts
const client = new OpenAI({
  apiKey: "dummy",
  baseURL: "http://127.0.0.1:3456/v1"
});
```

#### Python

```python
from openai import OpenAI

client = OpenAI(
    api_key="dummy",
    base_url="http://127.0.0.1:3456/v1",
)
```

说明：

1. `api_key` / `apiKey` 只要非空即可。
2. 真实密钥由 Clash for AI 在本地转发时处理。

---

## 5. 推荐配置方式

建议你把所有工具都统一接到同一个本地地址。

例如：

```text
http://127.0.0.1:3456/v1
```

这样做的好处是：

1. 只改一次工具配置
2. 后续换 Provider 不用再动工具
3. 哪家 Provider 当前生效，由 Clash for AI 控制

---

## 6. 切换 Provider 的方式

当你已经在多个工具中都接入本地地址后，切换 Provider 的步骤非常简单：

1. 打开 Clash for AI
2. 进入 `Providers`
3. 找到目标 Provider
4. 点击 `Activate`

切换后：

1. 你的工具配置不需要改
2. 新发出的请求会自动走新的 Provider

---

## 7. 如何检查是否配置成功

你可以用下面几个方法确认：

### 方法一：看 Provider 页面

1. 页面顶部 `core: ok`
2. `connected api base` 显示正常
3. Provider 已成功创建

### 方法二：做一次健康检查

在 Provider 卡片点击 `Check`。

如果返回：

```text
OK 200 in xxx ms
```

说明这个 Provider 基本连通。

### 方法三：看请求日志

进入 `Logs` 页面，查看是否已经有请求记录。

如果你的工具发起过请求，但日志为空，通常说明：

1. 工具没有真正连到 Clash for AI
2. `Base URL` 写错了
3. 本地端口写错了

---

## 8. 常见问题

### 8.1 为什么工具里填了 `dummy` 也能工作

因为工具填写的 key 只是为了满足客户端本身的非空校验。

真正用于访问远程 Provider 的密钥，来自你在 Clash for AI 里保存的 Provider 配置。

---

### 8.2 为什么我填了 `http://127.0.0.1:3456/v1` 还是失败

先检查：

1. Clash for AI 是否正在运行
2. 实际端口是不是已经自动切到 `3457`、`3458` 等
3. 页面顶部的 `connected api base` 是多少
4. 启动终端里 `[core] selected port ...` 显示多少

如果当前实际端口不是 `3456`，你必须把工具里的地址改成真实端口。

---

### 8.3 为什么创建 Provider 成功了，但请求还是失败

这通常是下面几种原因：

1. 当前激活的不是你刚添加的 Provider
2. `Base URL` 写错
3. API Key 无效
4. 远程 Provider 本身不可用
5. 你选择的模型该 Provider 不支持

建议操作：

1. 先点击 `Activate`
2. 再点击 `Check`
3. 再查看 `Logs`

---

### 8.4 这个工具会替我自动处理认证头吗

会。

当前逻辑是：

1. 优先透传客户端原始请求
2. 自动覆盖常见认证头
3. 自动处理常见 OpenAI-compatible 场景

所以大多数用户不需要理解 `Authorization`、`x-api-key`、`api-key` 的区别。

---

## 9. 最推荐的用户操作方式

如果你只想最快开始使用，直接按下面做：

1. 启动 Clash for AI
2. 在 `Providers` 页面添加远程 Provider
3. 点击 `Activate`
4. 在你的工具里填写：

```text
Base URL: http://127.0.0.1:3456/v1
API Key: dummy
```

5. 如果 `3456` 不可用，就改成当前实际端口

这就是最简单、最稳定的接入方式。
