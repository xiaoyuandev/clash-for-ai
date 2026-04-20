# Clash for AI 发版与自动更新说明

## 1. 这份文档解决什么问题

这份文档说明两件事：

1. 你如何发布 Clash for AI 的新版本
2. 用户安装后的客户端如何检测并下载更新

当前项目已经接入了 Electron 应用内更新逻辑，但它是否真正生效，取决于你是否按规范发布版本产物到 GitHub Releases。

---

## 2. 自动更新的工作原理

Clash for AI 当前使用的是：

1. `electron-builder`
2. `electron-updater`
3. GitHub Releases 作为发布源

工作流程如下：

1. 你在项目里提升应用版本号
2. 你构建打包产物
3. 你把产物发布到 GitHub Releases
4. 已安装客户端启动后检查更新
5. 如果发现新版本，客户端显示可更新状态
6. 用户点击下载更新
7. 下载完成后，用户点击重启并安装

---

## 3. 当前项目里的更新配置

当前已经完成的基础配置包括：

1. `apps/desktop/package.json` 中已引入 `electron-updater`
2. `apps/desktop/electron-builder.yml` 中已配置 GitHub 发布源
3. 主进程已接入更新状态管理
4. 设置页已支持：
   - 检查更新
   - 下载更新
   - 重启安装

注意：

1. 开发模式下不会真正检查更新
2. 只有打包后的应用才会启用自动更新

---

## 4. 你每次发版要做什么

### 第一步：修改版本号

你需要先修改：

```text
apps/desktop/package.json
```

例如从：

```json
"version": "0.1.0"
```

改成：

```json
"version": "0.1.1"
```

建议遵循语义化版本：

1. 修复问题：`patch`
2. 新增功能但兼容旧行为：`minor`
3. 有破坏性变更：`major`

当前仓库还提供了自动化发版工作流：

```text
.github/workflows/release-desktop.yml
```

当你推送形如 `v0.1.1` 的 tag 时，GitHub Actions 会自动：

1. 安装依赖
2. 构建桌面应用
3. 调用 `electron-builder`
4. 把产物发布到 GitHub Releases

---

### 第二步：构建应用

先执行：

```bash
pnpm --filter desktop build
```

如果需要产出 macOS 安装包，再执行：

```bash
pnpm --filter desktop build:mac
```

产物会由 `electron-builder` 输出。

---

### 第三步：创建 GitHub Release

自动更新依赖 GitHub Releases 上的发布文件。

你需要：

1. 推送代码到 GitHub
2. 创建一个新的 Git tag
3. 基于该 tag 创建 GitHub Release
4. 上传对应的安装包与更新元数据文件

常见做法：

```bash
git tag v0.1.1
git push origin v0.1.1
```

如果你使用仓库里的 GitHub Actions 自动化工作流，通常不需要再手动创建 Release。
工作流会在 tag 推送后自动创建或更新对应的 GitHub Release。

---

## 5. 自动更新需要哪些发布文件

对于 Electron 自动更新来说，单纯上传一个 `.dmg` 不够。

你还需要上传 `electron-builder` 生成的更新元数据文件。

通常至少包括：

1. 安装包
2. 对应的最新版本描述文件

在 macOS 场景下，常见会出现：

1. `.dmg`
2. `.zip`
3. `latest-mac.yml`

说明：

1. `latest-mac.yml` 用于告诉客户端“最新版本是谁、下载地址是什么”
2. `.zip` 常常是自动更新实际使用的包
3. `.dmg` 更多是给用户手动下载安装使用

所以你发布时不要只传 `.dmg`。

---

## 6. 用户侧会发生什么

当用户安装的是打包后的正式版本时：

1. 打开应用
2. 进入 `Settings`
3. 点击 `Check for Updates`

如果有新版本：

1. 状态会从 `idle` 变成 `available`
2. 页面会显示可用版本号
3. 用户点击 `Download Update`
4. 下载完成后，状态变成 `downloaded`
5. 用户点击 `Restart to Install`

这时应用会退出并安装新版本。

---

## 7. 为什么开发模式看不到更新

这是正常行为。

原因：

1. `electron-updater` 只应该在正式打包的应用里工作
2. 开发模式下没有稳定的安装包上下文
3. 也不应该在本地调试时误触发更新流程

所以当前实现里，开发模式会明确显示：

```text
Update checks are only available in packaged builds.
```

---

## 8. 推荐的发布流程

建议你以后每次发版都按这个顺序来：

1. 修改版本号
2. 提交代码
3. 推送到 GitHub
4. 打 tag
5. 构建安装包
6. 创建 GitHub Release
7. 上传安装包和更新元数据文件
8. 在一台已安装旧版本的机器上测试更新

---

## 9. 发布时最容易踩的坑

### 9.1 只上传了 `.dmg`

这通常会导致：

1. 用户可以手动下载
2. 但应用内自动更新无法工作

原因是缺少更新元数据文件和增量更新所需文件。

---

### 9.2 版本号没变

如果你重新发布了同一个版本号，客户端通常不会认为这是新版本。

所以每次正式发版必须递增版本号。

---

### 9.3 Release 没有公开

如果 GitHub Release 没有按客户端可访问的方式发布，更新检查可能会失败。

---

### 9.4 本地调试时误以为更新功能坏了

开发模式不参与自动更新，这是设计行为，不是故障。

---

## 10. 当前阶段的建议

现阶段建议先采用：

1. GitHub Releases 作为唯一发布源
2. 手动创建 Release
3. 应用内提供“检查更新 + 下载更新 + 重启安装”

先把整条链路跑通，再考虑以后扩展：

1. CI 自动打包
2. CI 自动创建 Release
3. 渠道区分（stable / beta）
4. 灰度发布

---

## 11. 一句话总结

自动更新不是“代码写完就结束”，而是：

1. 客户端更新逻辑
2. 打包产物
3. GitHub Release 发布规范

这三部分必须同时成立，用户端的自动更新才会真正可用。
