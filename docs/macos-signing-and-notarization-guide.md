# Relay Switch macOS 签名与 GitHub Actions 配置教程

本文档面向当前仓库的维护者，说明如何把 Relay Switch 的 macOS 桌面应用从“免费 ad-hoc 签名”升级到：

1. `Developer ID Application` 正式签名
2. `Developer ID Installer` 安装包签名
3. Apple notarization 公证
4. GitHub Actions 自动构建与发布

目标是让用户下载 `.dmg` / `.pkg` / `.zip` 后，不再频繁看到 “无法验证开发者” 或强烈的安全拦截提示。

---

## 1. 当前仓库现状

当前仓库里的 macOS 打包配置在：

- `apps/desktop/electron-builder.yml`
- `.github/workflows/release-desktop.yml`
- `.github/release-template.md`

其中当前配置里最关键的一点是：

- `apps/desktop/electron-builder.yml` 里 `mac.notarize: false`

这意味着：

1. 即使打包成功，构建产物默认也不会走 Apple notarization
2. 没有有效的 Developer ID 证书时，`electron-builder` 会退回到 ad-hoc 签名
3. macOS Gatekeeper 会把这种构建视为“未验证来源”，用户体验很差

---

## 2. 推荐方案

对于当前项目，推荐使用下面这套组合：

1. `Developer ID Application` 证书
   用来签名 `.app`
2. `Developer ID Installer` 证书
   用来签名 `.pkg`
3. `App Store Connect Team API Key`
   用来做 notarization

这是当前 Apple + electron-builder 组合下最稳的一种 CI 方案。

为什么这样选：

1. `Developer ID Application` 是 Mac 非 App Store 分发的标准签名方式  
   Apple 官方：Developer ID certificates  
   https://developer.apple.com/help/account/certificates/create-developer-id-certificates/
2. `Developer ID Installer` 专门用于 `.pkg`  
   electron-builder 也明确区分 app signing 和 installer signing  
   https://www.electron.build/code-signing.html
3. notarization 推荐使用 API key，而不是 Apple ID + app-specific password  
   electron-builder 官方也推荐优先使用 API key 这套环境变量  
   https://www.electron.build/electron-builder.interface.macconfiguration

---

## 3. 你需要准备的东西

### Apple 侧

1. Apple Developer Program 账号
2. 可创建证书的权限
   通常是 `Account Holder`
3. App Store Connect 中可创建 Team API Key 的权限
   通常是 `Account Holder` 或 `Admin`

### 本地 Mac 侧

1. 一台装有 Keychain Access 的 macOS 机器
2. Xcode Command Line Tools
3. 能登录 Apple Developer / App Store Connect 的浏览器会话

### GitHub 侧

1. 仓库管理员权限
2. 能写入 `Repository Secrets` 的权限

GitHub Secrets 官方说明：

https://docs.github.com/en/actions/concepts/security/about-secrets

---

## 3.1 个人开发者账号能不能用

可以，但要区分“能做正式签名”与“能不能顺利接到 GitHub Actions 自动公证”这两件事。

### 可以做的部分

如果你是付费的 Apple Developer Program 个人账号，通常可以：

1. 创建 `Developer ID Application`
2. 创建 `Developer ID Installer`
3. 本地完成正式签名
4. 提交 macOS app 做 notarization

也就是说：

- **个人开发者账号并不等于不能做 macOS 正式分发**

### 需要重点确认的部分

对于当前仓库推荐的 CI 方案，真正关键的是你能不能创建：

1. `Developer ID Application`
2. `Developer ID Installer`
3. **App Store Connect Team API Key**

如果这三项都能拿到，就可以按本文档里的 GitHub Actions 方案走。

### 为什么文档里强调 Team API Key

因为 Apple 官方明确说明：

- `Individual API keys` 不能用于 `notaryTool`

而我们当前推荐的 CI notarization 方案，就是通过 API key 驱动 `notaryTool` / electron-builder 的公证流程。

所以：

1. `个人开发者账号` 可以用于整个发布流程
2. 但 **不建议依赖 Individual API Key**
3. 应优先确认你在 App Store Connect 中是否能创建 **Team Key**

### 你应该怎么判断自己当前账号是否够用

最直接的检查方式不是先研究账号类型，而是直接去后台看这三件事：

1. Apple Developer 后台里能否创建 `Developer ID Application`
2. Apple Developer 后台里能否创建 `Developer ID Installer`
3. App Store Connect 里是否能进入：
   `Users and Access -> Integrations -> App Store Connect API -> Team Keys`

如果以上三项都可用，那你的账号对当前仓库已经够用。

如果前两项可以，但第三项不行，那么：

1. 你仍然可以本地完成正式签名
2. 但当前这套 GitHub Actions 自动 notarization 方案需要调整
3. 那种情况下就不建议直接照搬本文档的 CI 部分

### 对当前项目的建议

对于 Relay Switch 这个仓库：

1. **个人开发者账号可以用**
2. 但正式接 GitHub Actions 前，先确认 `Team Keys` 是否可用
3. 如果你能创建 Team Key，就继续按本文档走
4. 如果你不能创建 Team Key，先不要急着改 workflow，先确认 Apple 后台权限或团队配置

---

## 4. 第一步：创建 Developer ID 证书

Apple 官方文档：

https://developer.apple.com/help/account/certificates/create-developer-id-certificates/

需要创建两个证书：

1. `Developer ID Application`
2. `Developer ID Installer`

### 4.1 生成 CSR

在 macOS 上打开 `Keychain Access`：

1. 打开 `Keychain Access`
2. 菜单栏选择  
   `Keychain Access` → `Certificate Assistant` → `Request a Certificate From a Certificate Authority`
3. 输入你的邮箱和常用名称
4. 选择 `Saved to disk`
5. 保存得到一个 `.certSigningRequest`

### 4.2 在 Apple Developer 后台创建证书

进入：

`Certificates, Identifiers & Profiles` → `Certificates` → `+`

先创建：

1. `Developer ID Application`
2. `Developer ID Installer`

流程都是：

1. 选择证书类型
2. 上传刚才生成的 `.certSigningRequest`
3. 下载 `.cer`

### 4.3 导入到本机钥匙串

双击下载的 `.cer` 文件，把它安装到钥匙串里。

正常情况下，在 `Keychain Access` 的：

- `login`
- `My Certificates`

下面可以看到完整证书链。

---

## 5. 第二步：导出 `.p12` 证书文件

electron-builder 在 CI 中最常见的做法，是把证书导出成 `.p12` 文件，再通过 Secrets 传给 Actions。

参考：

https://www.electron.build/code-signing-mac.html

### 5.1 导出 Developer ID Application

在 `Keychain Access` 中：

1. 打开 `login` → `My Certificates`
2. 找到 `Developer ID Application: <Your Name or Company>`
3. 右键导出
4. 导出为 `.p12`
5. 设置导出密码

建议文件命名：

- `developer-id-application.p12`

### 5.2 导出 Developer ID Installer

同样操作：

1. 找到 `Developer ID Installer: <Your Name or Company>`
2. 导出为 `.p12`
3. 设置导出密码

建议文件命名：

- `developer-id-installer.p12`

### 5.3 密码建议

建议：

1. `Application p12` 使用单独密码
2. `Installer p12` 使用单独密码
3. 不要和 Apple 账号密码复用

---

## 6. 第三步：创建 App Store Connect Team API Key

Apple 官方说明：

- App Store Connect API get started  
  https://developer.apple.com/help/app-store-connect/get-started/app-store-connect-api
- Creating API Keys for App Store Connect API  
  https://developer.apple.com/documentation/appstoreconnectapi/creating-api-keys-for-app-store-connect-api

推荐使用 **Team Key**，不要优先使用个人 key。

原因：

1. Team Key 更适合 CI
2. Apple 文档明确说明：个人 key 不能用于 `notaryTool`

### 6.1 申请 / 打开 API 访问

进入：

`App Store Connect` → `Users and Access` → `Integrations`

如果你的团队还没启用 API，需要先请求开启。

### 6.2 创建 Team API Key

进入：

`Users and Access` → `Integrations` → `App Store Connect API` → `Team Keys`

然后：

1. 点击 `Generate API Key`
2. 输入名称，例如：
   `github-actions-notarization`
3. 选择合适权限
   一般用 `Admin` 最省事
4. 生成后下载 `.p8`

你会拿到：

1. `AuthKey_XXXXXX.p8`
2. `Key ID`
3. `Issuer ID`

注意：

1. `.p8` 只允许下载一次
2. 丢了就只能 revoke 后重新生成

---

## 7. 第四步：把证书和 API Key 转成适合 GitHub Secrets 的内容

### 7.1 把 `.p12` 转成 base64

在本地执行：

```bash
base64 -i developer-id-application.p12 | pbcopy
```

和：

```bash
base64 -i developer-id-installer.p12 | pbcopy
```

分别复制它们的 base64 内容。

### 7.2 `.p8` 怎么存

推荐直接把 `.p8` 文件内容原样存到 GitHub Secret，不要自行改格式。

可以用：

```bash
cat AuthKey_XXXXXX.p8
```

复制全文，包括：

```text
-----BEGIN PRIVATE KEY-----
...
-----END PRIVATE KEY-----
```

---

## 8. 第五步：在 GitHub 仓库里配置 Secrets

进入：

`GitHub Repository` → `Settings` → `Secrets and variables` → `Actions`

建议创建这些 `Repository Secrets`：

### 证书相关

1. `CSC_LINK`
   值：`Developer ID Application .p12` 的 base64

2. `CSC_KEY_PASSWORD`
   值：`Developer ID Application .p12` 的导出密码

3. `CSC_INSTALLER_LINK`
   值：`Developer ID Installer .p12` 的 base64

4. `CSC_INSTALLER_KEY_PASSWORD`
   值：`Developer ID Installer .p12` 的导出密码

这些变量名来自 electron-builder 官方：

https://www.electron.build/code-signing.html

### notarization 相关

5. `APPLE_API_KEY`
   值：`AuthKey_XXXXXX.p8` 的完整内容

6. `APPLE_API_KEY_ID`
   值：Apple 后台显示的 `Key ID`

7. `APPLE_API_ISSUER`
   值：Apple 后台显示的 `Issuer ID`

electron-builder 对 notarization 的环境变量要求见：

https://www.electron.build/electron-builder.interface.macconfiguration

---

## 9. 第六步：修改当前仓库配置

### 9.1 修改 `apps/desktop/electron-builder.yml`

当前文件中有：

```yml
mac:
  target:
    - dmg
    - pkg
    - zip
  entitlements: build/entitlements.mac.plist
  entitlementsInherit: build/entitlements.mac.plist
  notarize: false
```

这里建议改为：

```yml
mac:
  target:
    - dmg
    - pkg
    - zip
  entitlements: build/entitlements.mac.plist
  entitlementsInherit: build/entitlements.mac.plist
```

也就是：

1. 删除 `notarize: false`

原因：

1. 当前这行会明确关闭 electron-builder 的 notarization
2. 当 `APPLE_API_KEY` / `APPLE_API_KEY_ID` / `APPLE_API_ISSUER` 提供后，应允许 electron-builder 自动执行 notarization

---

### 9.2 修改 `.github/workflows/release-desktop.yml`

当前 workflow 里真正执行发布的是：

```yml
- name: Build and publish desktop release
  env:
    GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  run: pnpm --filter desktop release:${{ matrix.target }}
```

建议改为：

```yml
- name: Build and publish desktop release
  env:
    GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    CSC_LINK: ${{ secrets.CSC_LINK }}
    CSC_KEY_PASSWORD: ${{ secrets.CSC_KEY_PASSWORD }}
    CSC_INSTALLER_LINK: ${{ secrets.CSC_INSTALLER_LINK }}
    CSC_INSTALLER_KEY_PASSWORD: ${{ secrets.CSC_INSTALLER_KEY_PASSWORD }}
    APPLE_API_KEY: ${{ secrets.APPLE_API_KEY }}
    APPLE_API_KEY_ID: ${{ secrets.APPLE_API_KEY_ID }}
    APPLE_API_ISSUER: ${{ secrets.APPLE_API_ISSUER }}
  run: pnpm --filter desktop release:${{ matrix.target }}
```

这样做的特点是：

1. 配置最简单
2. 非 macOS job 会忽略这些 Apple 变量
3. 不需要单独为 macOS 再拆一个发布 step

如果你想更严格，也可以只在 `matrix.target == 'mac'` 时注入这些变量，但当前仓库没必要复杂化。

---

### 9.3 更新 `.github/release-template.md`

当前模板里有一句：

```md
This build is currently distributed without Apple notarization.
```

当正式启用签名 + notarization 后，应该把这段提示删除或改写，否则对用户是错误信息。

建议改成类似：

```md
This macOS build is signed with Developer ID and submitted for Apple notarization.
```

---

## 10. 第七步：本地先做一次验证

在真正上 GitHub Actions 前，建议先本地验证一遍。

### 10.1 在本机环境里导入证书

确保：

1. `Developer ID Application` 已在钥匙串
2. `Developer ID Installer` 已在钥匙串

### 10.2 配置 notarization 环境变量

例如：

```bash
export APPLE_API_KEY_ID="YOUR_KEY_ID"
export APPLE_API_ISSUER="YOUR_ISSUER_ID"
export APPLE_API_KEY="$(cat /path/to/AuthKey_XXXXXX.p8)"
```

### 10.3 本地打包

在仓库根目录执行：

```bash
pnpm --filter desktop build:mac
```

观察日志，重点看：

1. 不应再出现 `falling back to ad-hoc signature`
2. 应能看到正式签名身份
3. 应触发 notarization

---

## 11. 第八步：GitHub Actions 验证

### 11.1 手动触发

当前 workflow 支持：

- `workflow_dispatch`
- `push tags: v*`

建议先用：

1. `Actions`
2. `Release Desktop`
3. `Run workflow`

先手动试一次。

### 11.2 看日志重点

你要确认：

1. 没有再使用 ad-hoc 签名
2. `Developer ID Application` 被用于 `.app`
3. `Developer ID Installer` 被用于 `.pkg`
4. notarization 没被跳过
5. 最终产物成功上传 GitHub Release

---

## 12. 常见问题

### 12.1 仍然显示 ad-hoc 签名

常见原因：

1. `CSC_LINK` 没传进去
2. `CSC_KEY_PASSWORD` 错了
3. `.p12` 导出时不完整
4. 导出的并不是 `Developer ID Application`

### 12.2 `.pkg` 没有正确签名

常见原因：

1. 没提供 `CSC_INSTALLER_LINK`
2. 没提供 `CSC_INSTALLER_KEY_PASSWORD`
3. 使用了 Application 证书去签 Installer

### 12.3 notarization 被跳过

常见原因：

1. `mac.notarize: false` 还在
2. `APPLE_API_KEY` / `APPLE_API_KEY_ID` / `APPLE_API_ISSUER` 缺失
3. API key 不是 Team Key
4. API key 权限不足

### 12.4 `APPLE_API_KEY` 到底存什么

这里存的是：

- `.p8` 文件的**完整文本内容**

不是：

1. 文件路径
2. base64 后的 `.p8`
3. 只存 key id

### 12.5 Individual API Key 能不能用

不建议。

Apple 文档明确指出：

- Individual API key 不能用于 `notaryTool`

因此 CI 应优先使用 Team API Key。

---

## 13. 推荐的最终状态

完成后，这个仓库在 macOS 发布链路上的目标状态应该是：

1. 本地开发时可直接用钥匙串正式签名
2. GitHub Actions 通过 Secrets 自动完成签名
3. 发布产物自动完成 notarization
4. 下载后的 `.dmg` / `.pkg` / `.zip` 不再频繁触发严重安全警告

---

## 14. 这份教程落地到当前仓库后的最小改动清单

你至少需要做这三件事：

1. 配置 GitHub Secrets：
   - `CSC_LINK`
   - `CSC_KEY_PASSWORD`
   - `CSC_INSTALLER_LINK`
   - `CSC_INSTALLER_KEY_PASSWORD`
   - `APPLE_API_KEY`
   - `APPLE_API_KEY_ID`
   - `APPLE_API_ISSUER`

2. 修改：
   - `apps/desktop/electron-builder.yml`
   删除 `mac.notarize: false`

3. 修改：
   - `.github/workflows/release-desktop.yml`
   给发布 step 注入上述 Secrets

---

## 15. 参考链接

### Apple 官方

- Developer ID certificates  
  https://developer.apple.com/help/account/certificates/create-developer-id-certificates/

- Notarizing macOS software before distribution  
  https://developer.apple.com/documentation/security/notarizing-macos-software-before-distribution

- App Store Connect API get started  
  https://developer.apple.com/help/app-store-connect/get-started/app-store-connect-api

- Creating API Keys for App Store Connect API  
  https://developer.apple.com/documentation/appstoreconnectapi/creating-api-keys-for-app-store-connect-api

### electron-builder

- Code signing env vars  
  https://www.electron.build/code-signing.html

- macOS signing / export certificate  
  https://www.electron.build/code-signing-mac.html

- mac notarization env requirements  
  https://www.electron.build/electron-builder.interface.macconfiguration

### GitHub Actions

- GitHub Actions Secrets  
  https://docs.github.com/en/actions/concepts/security/about-secrets
