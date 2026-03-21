# 2026-02-26 - Android：生成签名 keystore + GitHub Secrets（本地脚本）

## 背景 / 目标

在配置 `MyFlowHub-Android` 的 GitHub Release 自动发布时，需要在仓库的 Actions Secrets 中提供 Android 签名所需的 keystore 与密码信息。

用户当前遇到的问题：
- Windows PowerShell 中无法直接执行 `keytool`：本机未安装/未配置 JDK，导致 `keytool` 不在 PATH。
- 同时 PowerShell 不支持使用 `\` 作为换行续行符，导致多行命令的参数被当作独立命令执行。

本次变更目标：
- 在 workspace 根目录提供一个 **不进入任何 repo git** 的脚本，帮助在 Windows 上一键生成 keystore 与 base64，并输出 GitHub Secrets 的配置指引。

## 具体变更内容

### 新增

- `scripts/gen-android-release-secrets.ps1`
  - 自动定位 `keytool`（PATH/JAVA_HOME/常见 Android Studio/JDK 路径）
  - 交互式生成 `myflowhub-release.jks`（默认输出到 `.tmp\\android-signing\\`）
  - 生成单行 base64 文件 `myflowhub-release.jks.b64`
  - 输出需要配置的 4 个 GitHub Secrets 名称与填写方式
- `scripts/plan_android-release-secrets.md`
  - 本次脚本新增的可交接计划文档

### 修改
- 无（不修改任何 `repo/*` 仓库的运行逻辑）

### 删除
- 无

## 任务映射（plan）

对应：`scripts/plan_android-release-secrets.md`
- S1 - 需求与接口确认
- S2 - 实现 `gen-android-release-secrets.ps1`
- S3 - 归档变更（本文）

## 关键设计决策与权衡

1) **不走 worktree / 不进 repo git**
- 脚本属于 workspace 控制面辅助工具（类似已有 `scripts/run-dev.ps1`），用户明确要求“不走 worktree”。

2) **避免把密码写入文件**
- 脚本仅生成 keystore 与 base64 文件；keystore 密码与 key 密码由 `keytool` 交互输入，用户自行保管并手动填入 GitHub Secrets。

3) **keytool 自动定位优先**
- 优先使用 PATH/JAVA_HOME，其次检查 Android Studio/JDK 常见安装路径，降低新机器首次配置成本。
- 若仍找不到 keytool，脚本会输出可操作的安装/定位建议并退出。

## 测试与验证方式 / 结果

### 本机（用户侧）验证步骤
1) 运行脚本：
   - `.\scripts\gen-android-release-secrets.ps1`
2) 根据脚本输出，把以下 4 个值添加到 GitHub 仓库的 **Repository secrets**：
   - `ANDROID_KEYSTORE_BASE64`
   - `ANDROID_KEYSTORE_PASSWORD`
   - `ANDROID_KEY_ALIAS`
   - `ANDROID_KEY_PASSWORD`
3) 推送 tag 触发 Release workflow（示例）：
   - `git tag v0.1.0`
   - `git push origin v0.1.0`
4) 在 GitHub 查看：
   - Actions：`release` workflow 成功
   - Releases：Assets 含 `app-release.apk` / `myflowhub.aar` / `build-info.txt`

说明：当前容器环境不保证存在 JDK/Android SDK，脚本主要用于用户本机执行。

## 潜在影响与回滚方案

- 影响：
  - 新增脚本仅用于本地生成签名材料与配置指引，不影响任何仓库运行逻辑。
  - 若 keystore 丢失，将无法对同一包名进行后续覆盖升级（风险已在脚本输出提示）。
- 回滚：
  - 删除 `scripts/gen-android-release-secrets.ps1` 与 `scripts/plan_android-release-secrets.md` 即可。

