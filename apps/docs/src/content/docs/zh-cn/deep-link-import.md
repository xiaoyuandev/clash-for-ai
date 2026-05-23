---
title: 面向中转站的 Deep Link 导入
description: 说明第三方中转站网页如何唤起 Relay Switch，并在用户确认后导入 Provider 或 Models 配置。
slug: zh-cn/deep-link-import
---

## 这是什么

Relay Switch 支持桌面端 deep link 导入流程。

第三方网页可以通过如下链接唤起桌面应用：

```text
relay-switch://v1/import?resource=provider&payload=BASE64URL_JSON
```

桌面应用会：

1. 打开应用或把已有窗口拉到前台
2. 解析导入请求
3. 弹出导入确认弹窗
4. 只有在用户确认后才真正导入

当前最小支持范围只有：

1. `provider`
2. `model`

如果你想直接使用现成的生成器页面，可以打开：

<a href="../../deeplink.html" target="_blank" rel="noreferrer">/deeplink.html</a>

## 快速接入函数

如果你的中转站网站只想增加一个 Provider 导入按钮，可以直接使用这个函数。

必填输入：

1. `name`
2. `baseUrl`
3. `apiKey`

```js
function createRelaySwitchProviderImportUrl({ name, baseUrl, apiKey }) {
  const payload = { name, baseUrl, apiKey };
  const json = JSON.stringify(payload);
  const bytes = new TextEncoder().encode(json);
  let binary = "";

  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }

  const encodedPayload = btoa(binary)
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/g, "");

  return `relay-switch://v1/import?resource=provider&payload=${encodedPayload}`;
}

// 在按钮点击事件中使用：
const url = createRelaySwitchProviderImportUrl({
  name: "OpenRouter",
  baseUrl: "https://openrouter.ai/api/v1",
  apiKey: "sk-or-example"
});

window.location.href = url;
```

对第三方中转站来说，一个常见接入方式是：

```html
<button id="import-to-relay-switch">导入到 Relay Switch</button>

<script>
  document.getElementById("import-to-relay-switch").addEventListener("click", () => {
    window.location.href = createRelaySwitchProviderImportUrl({
      name: "Your Relay Name",
      baseUrl: "https://relay.example.com/v1",
      apiKey: "USER_VISIBLE_OR_USER_GENERATED_API_KEY"
    });
  });
</script>
```

## URL 格式

请使用下面的结构：

```text
relay-switch://v1/import?resource=<provider|model>&payload=<base64url-json>
```

规则：

1. `resource` 只能是 `provider` 或 `model`
2. `payload` 必须是 base64url 编码后的 JSON 对象
3. 如果路由不支持、payload 非法或缺少必填字段，应用会拒绝导入

## 本地验证方式

如果你想在自己的机器上验证这条链路，建议使用打包后的桌面应用，而不是只依赖 `pnpm dev`。

原因：

1. 浏览器点击 `relay-switch://...` 时，实际是让操作系统去打开这个协议
2. 操作系统必须先知道哪个应用负责处理这个 scheme
3. 打包后的桌面应用会更稳定地声明这个协议处理器
4. 仅靠开发态进程，浏览器通常无法稳定识别这个协议

推荐验证步骤：

1. 先构建一个本地打包版桌面应用
2. 至少启动一次这个打包后的应用
3. 打开 <a href="../../deeplink.html" target="_blank" rel="noreferrer">/deeplink.html</a>
4. 点击 `Open Deep Link`
5. 确认 Relay Switch 被唤起，并出现导入确认弹窗

如果浏览器提示这个 scheme 没有注册 handler，最常见的原因就是打包后的应用还没有安装或至少启动过一次。

## Provider payload

支持字段：

```json
{
  "name": "OpenRouter",
  "baseUrl": "https://openrouter.ai/api/v1",
  "apiKey": "sk-or-example"
}
```

说明：

1. `name`、`baseUrl`、`apiKey` 为必填
2. 这里的公开 payload 设计刻意与当前桌面端手动添加 Provider 的表单字段保持一致
3. 对第三方接入方来说，最小导入流程不需要额外提供 `authMode`
4. Relay Switch 当前会按现有添加表单的默认 bearer 行为处理这类导入

示例 deep link：

```text
relay-switch://v1/import?resource=provider&payload=eyJuYW1lIjoiT3BlblJvdXRlciIsImJhc2VVcmwiOiJodHRwczovL29wZW5yb3V0ZXIuYWkvYXBpL3YxIiwiYXBpS2V5Ijoic2stb3ItZXhhbXBsZSJ9
```

## Model payload

支持字段：

```json
{
  "name": "Relay Models",
  "baseUrl": "https://relay.example.com/v1",
  "apiKey": "sk-model-example",
  "providerType": "openai-compatible",
  "modelIds": ["gpt-4o-mini", "claude-3-7-sonnet"]
}
```

说明：

1. `name`、`baseUrl`、`apiKey` 以及至少一个模型 ID 为必填
2. `providerType` 为可选
3. 支持的 `providerType` 值：
   `openai-compatible`
   `anthropic-compatible`
4. 如果不传 `providerType`，Relay Switch 默认按 `openai-compatible` 处理
5. `modelIds` 中第一个模型会作为默认模型 ID

示例 deep link：

```text
relay-switch://v1/import?resource=model&payload=eyJuYW1lIjoiUmVsYXkgTW9kZWxzIiwiYmFzZVVybCI6Imh0dHBzOi8vcmVsYXkuZXhhbXBsZS5jb20vdjEiLCJhcGlLZXkiOiJzay1tb2RlbC1leGFtcGxlIiwicHJvdmlkZXJUeXBlIjoib3BlbmFpLWNvbXBhdGlibGUiLCJtb2RlbElkcyI6WyJncHQtNG8tbWluaSIsImNsYXVkZS0zLTctc29ubmV0Il19
```

## 如何生成 payload

payload 的规则是：

1. 先准备 JSON 对象
2. 按 UTF-8 编码
3. 再做 base64url 编码

base64url 的规则：

1. 把 `+` 替换成 `-`
2. 把 `/` 替换成 `_`
3. 去掉末尾的 `=`

JavaScript 示例：

```js
function toBase64Url(value) {
  return btoa(JSON.stringify(value))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/g, "");
}

const payload = toBase64Url({
  name: "OpenRouter",
  baseUrl: "https://openrouter.ai/api/v1",
  apiKey: "sk-or-example"
});

const url = `relay-switch://v1/import?resource=provider&payload=${payload}`;
```

## 用户体验流程

当用户点击 deep link 后：

1. 系统询问是否打开 Relay Switch
2. Relay Switch 被唤起
3. 应用展示导入确认弹窗
4. 用户选择确认或取消
5. 只有确认后才真正写入配置

## 安全建议

不要把这套流程当作静默导入。

建议：

1. 默认保留用户确认步骤
2. 不要公开传播包含真实 API Key 的链接
3. 尽量使用短期链接或用户主动生成的链接
4. 发送方也应对 `baseUrl` 和 payload 字段做基础校验

## 当前限制

当前最小实现还不支持：

1. `provider + models` 组合导入
2. 签名 payload
3. 通过 token 拉取远程配置
4. 自动覆盖或合并现有配置
