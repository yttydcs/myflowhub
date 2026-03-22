# Plan - MyFlowHub-Win：GitHub Actions 自动构建（CI）+ Tag Release（Windows amd64）

## Workflow 信息
- 范围：单仓库（`MyFlowHub-Win`）
- 分支：`chore/win-actions-release`
- Worktree：`d:\project\MyFlowHub3\worktrees\win-actions-release\MyFlowHub-Win`
- Base：`main`（当前 worktree 基于：`ba1aae0`）
- 规范：
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文，前缀可英文）

## 背景与需求（已确认）

### 目标
为仓库增加 GitHub Actions，实现：
1) `pull_request -> main`：自动构建 Windows `amd64`（不发布）。
2) `push -> main`：自动构建 Windows `amd64`（不发布）。
3) `push tag (vX.Y.Z)`：自动构建并创建 GitHub Release，上传构建产物（发布）。

### 约束
- Release 仅允许 tag 格式：`v1.2.3`（严格 SemVer，不允许 `-rc` 等后缀）。
- “只在 main 上跑 release”的判定：tag 指向的提交必须在 `origin/main` 历史中（祖先即可，不要求是 main 当前 HEAD）。
- 若 tag 不满足格式或不在 main 历史：workflow **直接失败**。

### 构建命令（当前仓库 README）
- Windows build：`wails build -platform windows/amd64`
- 产物预期：`build/bin/myflowhub-win.exe`

### 运行环境选型（CI）
- OS：GitHub `windows-latest`
- Node：`22`（LTS，兼容 Vite 5 / Vue 3）
- Go：遵循 `go.mod` 的 `toolchain go1.24.5`（CI 固定 `1.24.5`）
- Wails CLI：与 `go.mod` 中依赖对齐，固定 `v2.11.0`

### Artifact（已确认）
- CI 构建产物上传为 Actions artifact，保留 `30` 天（用于下载验证）。

## 总体方案（简述）
采用 **两条 workflow**（权限更小、更清晰）：
1) `ci-build.yml`：PR / main push 触发，执行 build 并上传 artifact（30 天）。
2) `release.yml`：tag push 触发；校验 tag 格式与 main 历史关系；build；用 `GITHUB_TOKEN` 创建 Release 并上传资产（exe + sha256）。

## 3.1) 计划拆分（Checklist）

### CI-1：Workspace 准备
- 目标：确保独占 worktree + 专业分支，避免在 `repo/` 直接改动。
- 当前状态：已创建 worktree 与分支（本计划文件即该 worktree 内）。
- 验收：
  - `git status -sb`：在 `chore/win-actions-release`，工作区干净。
- 回滚点：
  - `git worktree remove` + `git worktree prune`，删除分支（若未推送）。

### CI-2：新增 CI 构建 workflow（PR + push main）
- 目标：
  - `pull_request -> main` / `push -> main` 时：构建 Windows `amd64`，并上传 artifact（保留 30 天）。
- 涉及文件（预期）：
  - `.github/workflows/ci-build.yml`
- 验收：
  - 在 GitHub 上 PR / push main 能看到 workflow 运行；
  - 构建成功后可下载 artifact，包含 `myflowhub-win.exe`（或明确的产物路径）。
- 测试点：
  - 依赖安装可复现（Node/Go/Wails 版本固定）；
  - 缓存启用后，重复运行时间下降（非强制）。
- 回滚点：
  - revert `.github/workflows/ci-build.yml`。

### CI-3：新增 Release workflow（tag vX.Y.Z）
- 目标：
  - `push tag v1.2.3` 时：校验 tag 格式与 main 历史关系；构建 Windows `amd64`；创建 GitHub Release 并上传资产。
- 涉及文件（预期）：
  - `.github/workflows/release.yml`
- 验收：
  - tag `v1.2.3` push 后 workflow 成功；
  - Release 自动生成，资产包含：
    - `myflowhub-win.exe`
    - `sha256` 校验文件（文件名可审计、可追溯）。
- 测试点：
  - tag 非 `v\\d+\\.\\d+\\.\\d+`：workflow 失败；
  - tag commit 不在 main 历史：workflow 失败；
  - Release 权限最小化（仅发布 job 需要 `contents: write`）。
- 回滚点：
  - revert `.github/workflows/release.yml`；
  - 如误发 tag：删除 tag / 删除 Release（按需手工操作）。

### CI-4：本地静态校验（不依赖 GitHub）
- 目标：在提交前尽量发现低级错误（路径、命令、YAML 拼写）。
- 验收：
  - `.github/workflows/*.yml` 存在且路径正确；
  - workflow 中引用的产物路径与 README 一致（`build/bin/myflowhub-win.exe`）。
- 回滚点：revert 对 workflow 的修改。

### CI-5：Code Review（阶段 3.3）
- 按 AGENTS 3.3 清单逐项审查（需求覆盖/架构/性能/一致性/安全/测试）。

### CI-6：归档变更（阶段 4）
- 新增文档：`docs/change/2026-02-28_win-actions-release.md`
- 必须包含：
  - 背景/目标、变更清单、与 CI-1~CI-4 任务映射、关键设计决策与权衡（tag 校验 / main 限制 / 权限与缓存策略）、验证方式/结果、回滚方案。

## 依赖、风险与注意事项
- 依赖：
  - GitHub Actions runner 可下载到 `Go 1.24.5`（如失败需调整为 `go-version-file: go.mod` 或显式版本）。
- 风险：
  - Wails CLI 安装或构建在 `windows-latest` 上偶发失败（通常可通过固定版本 + cache 缓解）。
  - 产物路径若未来调整，需要同步更新 workflow 的上传路径。
