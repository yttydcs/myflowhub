# 2026-03-01 MyFlowHub-Win：修复 Release workflow 的 tag 格式校验（vX.Y.Z）

## 变更背景 / 目标

仓库已新增 GitHub Actions 的 CI build 与 tag release，但在首次发布 `v0.0.1` 时，Release workflow 在 `Validate tag format` 步骤直接失败，导致后续构建与创建 GitHub Release 被跳过。

本次目标：
- 允许严格 SemVer tag（`v1.2.3`）通过校验并继续执行发布流程；
- 非法 tag 继续直接失败（安全默认）；
- 不影响 `pull_request -> main` 与 `push -> main` 的 CI build。

## 具体变更内容（新增 / 修改 / 删除）

### 修改
- `.github/workflows/release.yml`
  - 修复 PowerShell 正则：将 `^v\\d+\\.\\d+\\.\\d+$` 改为 `^v\d+\.\d+\.\d+$`。
  - 增加注释，避免再次误用“双反斜杠”。

### 新增
- `docs/change/2026-03-01_win-release-tag-validate.md`（本文档）

### 删除
- 无

## 对应 plan.md 任务映射
- R1：Workspace 准备（独占 worktree + 分支）
- R2：修复 `release.yml` 的 tag 校验正则
- R3：GitHub 上验证（删除远端 tag 后重新 push 触发 Release）

## 关键设计决策与权衡

1) **继续使用宽松 tag glob 触发 + workflow 内严格校验**
- GitHub tags filter 是 glob，难以表达严格 SemVer；
- 因此维持 `v*.*.*` 触发，再用正则 `^v\d+\.\d+\.\d+$` 严格校验并失败（符合“非法 tag 直接失败”）。

2) **PowerShell 正则转义规则明确化**
- PowerShell 单引号字符串不会处理反斜杠转义，`\\d` 会变成匹配字面量 `\d`；
- 因此正则应使用单反斜杠（`\d`、`\.`），并通过注释固化该约束。

## 测试与验证方式 / 结果

### 本地验证（PowerShell）
- `('v0.0.1' -match '^v\d+\.\d+\.\d+$') -eq $true`
- `('v0.0.1-rc1' -match '^v\d+\.\d+\.\d+$') -eq $false`

### GitHub 验证（需要在合并到 main 后执行）
由于 `v0.0.1` 已存在，需先删除远端 tag 再重新推送触发：
1) 删除远端 tag：`git push --delete origin v0.0.1`
2) 重新创建/指向 main HEAD 的 tag 并推送：`git tag -fa v0.0.1 -m "发布 v0.0.1" && git push origin v0.0.1`
3) 预期：
   - `Release (Windows amd64)` workflow 通过；
   - 生成 GitHub Release；
   - assets 包含：
     - `myflowhub-win.exe`
     - `myflowhub-win.exe.sha256`

## 潜在影响与回滚方案

### 潜在影响
- 仅影响 Release workflow 的 tag 校验逻辑；对应用构建产物无行为影响。

### 回滚方案
- revert `.github/workflows/release.yml` 的此次修改即可恢复旧行为（不推荐，因为会导致合法 tag 仍失败）。
- 如误操作 tag：
  - 删除错误 tag（本地与远端）；
  - 删除对应 GitHub Release（如已生成）。

