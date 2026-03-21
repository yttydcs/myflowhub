# 2026-02-28 MyFlowHub-Win：GitHub Actions 自动构建 + Tag Release（Windows amd64）

## 变更背景 / 目标

仓库此前缺少 CI/CD，无法在合并或发布前自动验证 “能否构建”，也无法在打版本 tag 时自动产出可下载的 Release 资产。

本次目标：
- `pull_request -> main`：自动构建 Windows `amd64`（不发布）。
- `push -> main`：自动构建 Windows `amd64`（不发布）。
- `push tag vX.Y.Z`：自动构建并创建 GitHub Release，上传构建产物（发布）。
- 所有构建上传 Actions artifact，保留 30 天，便于下载验证。

## 具体变更内容（新增 / 修改 / 删除）

### 新增
- `.github/workflows/ci-build.yml`
  - 触发：`pull_request`（目标分支 main）、`push`（分支 main）
  - runner：`windows-latest`
  - 环境固定：
    - Go：`1.24.5`
    - Node：`22`
    - Wails CLI：`v2.11.0`（`go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0`）
  - 构建：`wails build -platform windows/amd64`
  - 产物校验：验证 `build/bin/myflowhub-win.exe` 存在，不存在则列出 `build/bin` 目录帮助定位
  - Artifact：上传 `build/bin/myflowhub-win.exe`，保留 30 天
  - 缓存：启用 Go modules 与 npm cache（基于 `frontend/package-lock.json`）
  - 并发：同一 ref 仅保留最新一次运行（`cancel-in-progress: true`）

- `.github/workflows/release.yml`
  - 触发：`push` tags（`v*.*.*`，并在 workflow 内严格校验）
  - tag 规则：
    - 仅允许 `v1.2.3`（严格 SemVer，不允许 `-rc` 等后缀）
    - tag 指向的提交必须在 `origin/main` 历史中（否则直接失败）
  - 构建：同 `ci-build.yml`
  - 校验和：生成 `build/bin/myflowhub-win.exe.sha256`
  - Artifact：上传 exe + sha256（保留 30 天）
  - Release：使用 `GITHUB_TOKEN` 创建/更新 Release 并上传资产（exe + sha256）

### 修改
- `plan.md`：切换为本 workflow 的计划与约束说明（便于交接与审计）。

### 删除
- 无

## 对应 plan.md 任务映射
- CI-1：Workspace 准备（独占 worktree + 分支）
- CI-2：新增 `ci-build.yml`
- CI-3：新增 `release.yml`
- CI-4：本地静态校验（路径/命令/产物路径）

## 关键设计决策与权衡

1) **Release 安全门禁：tag 必须在 main 历史**
- 采用 `git merge-base --is-ancestor <tagSha> origin/main` 校验。
- 好处：避免从未合入 main 的提交打 tag 误发版。
- 权衡：需要在 release workflow 中 `fetch origin main`，但成本可接受。

2) **tag 触发采用宽匹配 + 内部严格校验**
- GitHub tags filter 为 glob，不适合做严格 SemVer 校验；因此用 `v*.*.*` 触发，再用正则 `^v\\d+\\.\\d+\\.\\d+$` 严格验证并失败。

3) **性能：缓存与并发控制**
- Go 使用 `actions/setup-go` 的 cache（基于 `go.sum`）。
- Node 使用 `actions/setup-node` 的 npm cache（基于 `frontend/package-lock.json`）。
- 对 `pull_request` / `push main` 使用 concurrency 取消旧任务，避免同一分支重复占用 runner。

4) **权限最小化**
- `release.yml` 仅在发布 job 里授予 `contents: write`（用于创建 Release 和上传资产）。
- `ci-build.yml` 不需要写入仓库内容。

## 测试与验证方式 / 结果

本地无法完整模拟 GitHub Actions 环境，验证以 GitHub 侧为准：

1) PR 验证（不发布）
- 新建分支并提 PR 到 main。
- 预期：触发 `CI Build (Windows amd64)`，成功后可下载 artifact（30 天有效）。

2) main push 验证（不发布）
- 合并 PR 或直接 push 到 main（按团队策略）。
- 预期：同上，产生 artifact。

3) Release 验证（发布）
- 在 main 历史上的提交打 tag：`v1.2.3` 并 push tag。
- 预期：
  - `Release (Windows amd64)` workflow 成功；
  - 生成 GitHub Release；
  - Release assets 包含 `myflowhub-win.exe` 与 `myflowhub-win.exe.sha256`；
  - 同时产生 Actions artifact（30 天有效）。

## 潜在影响与回滚方案

### 潜在影响
- CI 会在 PR/main push 触发 Windows 构建，增加 Actions 使用量与排队时间。
- 若上游 action 或 runner 环境变化，可能导致构建偶发失败（已通过固定 Go/Node/Wails 版本与 cache 尽量降低波动）。

### 回滚方案
- 回滚构建与发布：revert 以下文件即可：
  - `.github/workflows/ci-build.yml`
  - `.github/workflows/release.yml`
- 如已误发版：
  - 删除错误 tag（本地与远端）；
  - 删除对应 GitHub Release（含 assets）。

