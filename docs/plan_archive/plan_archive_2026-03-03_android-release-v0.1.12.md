# Plan - Android：打 tag 发布 Release（v0.1.12）

## Workflow 信息
- Repo：`MyFlowHub-Android`
- 分支：`chore/android-release-v0.1.12`
- Worktree：`d:\project\MyFlowHub3\worktrees\chore-android-release-v0.1.12`
- Base：`repo/MyFlowHub-Android/main`
- 目标 tag：`v0.1.12`
- 参考：`d:\project\MyFlowHub3\guide.md`（commit 信息中文）

## 背景 / 目标
- 背景：Android `main` 已包含最新变更（含 `hubmobile` 使用 `management v0.1.2`）。
- 目标：创建并 push `v0.1.12` tag，触发 `.github/workflows/release.yml`：
  - 构建并签名 `app-release.apk`
  - 上传 `myflowhub.aar`
  - 生成 `build-info.txt`
  - 发布 GitHub Release（自动生成 release notes）

## 非目标
- 不修改任何业务代码/UI。
- 不调整 CI 流程（仅触发现有 release workflow）。

## 约束（边界）
- `release.yml` 仅监听 `push tags: v*.*.*`，因此 tag 必须满足 `vMAJOR.MINOR.PATCH`。
- Release workflow 依赖仓库 Secrets（如 `ANDROID_KEYSTORE_BASE64` 等）；若 Secrets 缺失会导致 release 失败。
- tag 一旦推送到远端，原则上不删除；如需修复，建议发布下一个 patch 版本。

## 验收标准
- 远端存在 tag `v0.1.12`（`git ls-remote --tags origin v0.1.12` 可见）。
- GitHub Actions 的 `release` workflow 针对 `v0.1.12` 成功完成，并在 Release 附件中包含：
  - `app-release.apk`
  - `myflowhub.aar`
  - `build-info.txt`

---

## 3.1) 计划拆分（Checklist）

### REL0 - 确认 tag 未被占用
- 目标：避免重复 tag 触发失败或覆盖历史 release。
- 操作：
  - `git tag --list v0.1.12`
  - `git ls-remote --tags origin v0.1.12`
- 验收：本地/远端均不存在 `v0.1.12`。

### REL1 - 创建并推送 tag
- 目标：触发 release workflow 发布正式 Release。
- 操作：
  - `git tag -a v0.1.12 <commit> -m "release: v0.1.12"`
  - `git push origin v0.1.12`
- 验收：远端 tag 可见。
- 回滚：
  - 若刚推送且确认未被消费，可删除 tag（高风险，不推荐）；优先改为发布 `v0.1.13`。

### REL2 - 记录与归档
- 目标：可审计记录本次 release 的 tag、commit 与验证方式。
- 输出：
  - `docs/change/2026-03-03_android-release-v0.1.12.md`（本 worktree）
  - 同步到全局 `docs/change/` 与 `plan.md` 的 Workflow Archives

