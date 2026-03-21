# Plan - MyFlowHub-Win：修复 Release 的 tag 格式校验（vX.Y.Z）

## Workflow 信息
- 范围：单仓库（`MyFlowHub-Win`）
- 分支：`fix/win-release-tag-fix`
- Worktree：`d:\project\MyFlowHub3\worktrees\win-release-tag-fix\MyFlowHub-Win`
- Base：`main`（当前 worktree 基于：`0bfff4e`）
- 规范：
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文，前缀可英文）

## 背景（问题清单与用户需求）

### 问题（已复现，可审计）
Release workflow（`.github/workflows/release.yml`）在 `Validate tag format` 步骤失败，导致 tag 发布无法继续执行构建与创建 GitHub Release。

触发实例：
- tag：`v0.0.1`
- 失败步骤：`Validate tag format`

根因（已定位）：
- `release.yml` 中 PowerShell 正则写成了 `^v\\d+\\.\\d+\\.\\d+$`。
- 在 PowerShell 单引号字符串里，反斜杠不会被转义，导致正则实际匹配的是字面量 `\\d` / `\\.`，从而 **无法匹配** `v0.0.1` 这类正常 tag。

### 需求（已确认）
1) Release 仅允许严格 SemVer tag：`v1.2.3`（不允许 `-rc` 等后缀）。
2) 当 tag 不合法时：workflow **直接失败**（避免误以为发布成功）。
3) main / PR 的 CI build 保持不变。

## 目标
1) `v0.0.1` 这类合法 tag 能通过校验并继续执行 Release workflow。
2) 非法 tag（如 `v0.0.1-rc1` / `v0.0` / `0.0.1`）依旧会被拒绝并失败。

## 范围与约束
- 必须：
  - 仅修复 `.github/workflows/release.yml` 的 tag 校验逻辑（最小变更）。
  - 不改 Wails 构建命令、不改产物路径与 Release 上传逻辑。
- 不做：
  - 不新增预发布策略（`-rc`）支持。
  - 不调整 CI build（`ci-build.yml`）行为。

## 总体方案（简述）
- 将正则从 `^v\\d+\\.\\d+\\.\\d+$` 修正为 `^v\d+\.\d+\.\d+$`（PowerShell 正则应使用单反斜杠）。
- 为避免再次误用，采用明确的注释说明：PowerShell regex 不需要双反斜杠。

## 3.1) 计划拆分（Checklist）

### R1 - Workspace 准备
- 目标：独占 worktree + 分支，避免在 `repo/` 直接改动。
- 当前状态：已完成（本计划文件即该 worktree 内）。
- 验收：
  - `git status -sb`：在 `fix/win-release-tag-fix`，工作区干净。
- 回滚点：
  - `git worktree remove` + `git worktree prune`，删除分支（若未推送）。

### R2 - 修复 release.yml 的 tag 校验正则
- 目标：`Validate tag format` 能正确识别 `v0.0.1` 为合法。
- 涉及文件：
  - `.github/workflows/release.yml`
- 验收：
  - 本地 PowerShell 断言通过：
    - `('v0.0.1' -match '^v\d+\.\d+\.\d+$') -eq $true`
    - `('v0.0.1-rc1' -match '^v\d+\.\d+\.\d+$') -eq $false`
- 回滚点：
  - revert `release.yml` 改动。

### R3 - GitHub 上验证（重新触发 Release）
- 目标：让 `v0.0.1` 重新触发 Release workflow 并成功发布。
- 说明：tag 已存在时无法“再次 push 触发”。
- 方案（需用户确认其一）：
  - A) 删除远端 tag `v0.0.1` 后重新 push（推荐，语义最清晰）
  - B) force 更新 tag 并 push（会改写 tag 对象）
  - C) 发布新版本 tag（如 `v0.0.2`）
- 验收：
  - `Release (Windows amd64)` workflow 成功；
  - GitHub Release 出现并包含：
    - `myflowhub-win.exe`
    - `myflowhub-win.exe.sha256`
- 回滚点：
  - 删除 tag / 删除 Release 资产（按需手工）。

### R4 - Code Review（阶段 3.3）
- 按 AGENTS 3.3 清单逐项审查（需求覆盖/架构/性能/一致性/安全/测试）。

### R5 - 归档变更（阶段 4）
- 新增文档：`docs/change/2026-03-01_win-release-tag-validate.md`
- 必须包含：
  - 背景/目标、变更清单、与 R1~R3 任务映射、关键决策与权衡、验证方式/结果、回滚方案。
