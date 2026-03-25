# Plan - win-embed-dist-placeholder

## Workflow Information
- Repo: `MyFlowHub-Win`
- Branch: `fix/win-embed-dist-placeholder`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-embed-dist-placeholder`
- Current Stage: `4`

## Stage Records

### Initialization
- guide.md:
  - 已读取 `D:\project\MyFlowHub3\guide.md`。
- base/worktree confirmation:
  - 控制面仓库：`D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - 当前实现 worktree：`D:\project\MyFlowHub3\worktrees\fix-win-embed-dist-placeholder`
  - 本轮仅涉及 `MyFlowHub-Win` 单仓。

### Stage 1 - Requirements Analysis
#### Goal
- 修复 `MyFlowHub-Win` 在执行 Wails bindings / `go mod tidy` 时因 `//go:embed all:frontend/dist` 找不到可嵌入文件而失败的问题。

#### Scope
- 必须:
  - 确认 `frontend/dist` 在未构建前端时始终至少存在一个可嵌入文件。
  - 让 `go mod tidy`、`GOWORK=off go test ./...` 和 Wails 构建链不再依赖手工恢复占位文件。
  - 记录本次修复与验证结果。
- 可选:
  - 补充可复用的排查线索。
- 不做:
  - 不修改 Win 前端业务逻辑、界面行为和 bindings API。
  - 不引入新的前端构建工具或额外构建阶段。

#### Use Cases
- 在新 worktree 或 CI 中，尚未执行前端产物构建时，Wails 仍可完成 bindings 生成与 Go 侧依赖分析。
- 本地执行 `wails build -nopackage` 或 Wails CLI 生成 bindings 时，不再因为 `frontend/dist` 为空而失败。

#### Functional Requirements
- `frontend/dist` 必须在仓库干净状态下保留至少一个可被 `go:embed` 识别的文件。
- 前端构建脚本在清理 / 重建 `dist` 后，必须自动补回该占位文件。
- 现有 `main.go` 的 `//go:embed all:frontend/dist` 路径必须保持可用。

#### Non-functional Requirements
- 改动面最小，优先修复占位文件策略，不扩大到运行时代码。
- 继续保持 `frontend/dist` 作为构建产物目录，不把真实前端产物纳入版本库。

#### Inputs / Outputs
- Inputs:
  - 当前 `main.go` 的 `//go:embed all:frontend/dist`
  - 当前 `.gitignore` 仅保留 `frontend/dist/.keep`
  - 当前主线存在 `frontend/dist/.keep` 被删导致的 Wails 构建错误
- Outputs:
  - 更稳健的 `frontend/dist` 占位文件策略
  - 对应构建脚本和忽略规则更新
  - repo-level change archive

#### Edge Cases
- 前端构建工具会先清空 `dist`，占位文件若不在构建后重建，问题会再次出现。
- 若只保留隐藏文件占位，后续手工清理或跨平台工具行为可能继续导致目录被误判为空。

#### Acceptance Criteria
- `go mod tidy` 通过。
- `GOWORK=off go test ./... -count=1` 通过。
- `GOWORK=off wails build -nopackage` 通过，且不再报 `cannot embed directory frontend/dist: contains no embeddable files`。

#### Risks
- 若占位文件策略与现有 `.gitignore` / 前端 build 脚本不一致，后续构建仍可能把 `dist` 清成空目录。

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 保留 `main.go` 的 `go:embed` 路径不变，改为使用普通占位文件而不是只依赖隐藏 `.keep`。
- 更新 `.gitignore` 允许该占位文件纳入版本库。
- 更新 `frontend/package.json` build 脚本，在 `vite build` 后自动重建占位文件，避免 `dist` 被清空后再次触发同类问题。

#### Alternatives Considered
- 方案 A：只在本地恢复 `.keep`
  - 放弃原因：不能修复仓库策略，后续清理或新环境仍会复现。
- 方案 B：修改 `main.go`，取消 `frontend/dist` 的 `go:embed`
  - 放弃原因：会改变 Wails 现有资源加载路径，改动面过大。
- 方案 C：把占位文件改成普通受管文件，并让构建脚本自动补回
  - 采用原因：最小、稳定，且能直接覆盖当前删除 `.keep` 的触发条件。

#### Module Responsibilities
- `main.go`
  - 继续通过 `//go:embed all:frontend/dist` 嵌入前端静态资源目录。
- `.gitignore`
  - 控制 `frontend/dist` 的忽略策略，只保留必要占位文件。
- `frontend/package.json`
  - 在前端 build 结束后补回占位文件，保持目录长期满足 `go:embed` 约束。

#### Data / Call Flow
- 仓库检出 -> `frontend/dist/placeholder.txt` 存在 -> `go mod tidy` / Wails bindings 通过加载 Go 包
- `npm run build` / `wails build` -> `vite build` 清理并重建 `dist` -> post-build 脚本补回 `placeholder.txt`

#### Interface Drafts
- 不新增业务接口。
- 变更面仅限：
  - `.gitignore`
  - `frontend/package.json`
  - `frontend/dist` 占位文件

#### Error Handling and Safety
- 占位文件采用普通文本文件，避免再次依赖隐藏文件语义。
- 构建脚本显式 `mkdirSync('dist', { recursive: true })` 后写入占位文件，保证构建后目录状态确定。

#### Performance and Testing Strategy
- 验证重点:
  - `go mod tidy`
  - `GOWORK=off go test ./... -count=1`
  - `GOWORK=off wails build -nopackage`

#### Extensibility Design Points
- 本次修复沉淀为 Win 构建链路的一条稳定规则：`frontend/dist` 的占位文件策略必须与 `.gitignore` 和 build 脚本一致维护。

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - 让 `MyFlowHub-Win` 的 Wails / Go 构建链在未预先产出前端文件时仍可稳定通过。
- Current state:
  - 当前主线依赖 `frontend/dist/.keep` 作为 `go:embed` 占位。
  - 控制面仓库当前已出现 `.keep` 删除，用户在 Wails 绑定阶段报 `cannot embed directory frontend/dist: contains no embeddable files`。
  - 旧归档已明确 `dist` 占位文件是既有构建约束。

#### Docs Governance Routing Decision
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements:
  - `none`
- Related specs:
  - `none`
- Related lessons:
  - `none`
- Canonical destinations:
  - workflow control -> 当前 worktree 根 `plan.md`
  - completed result -> `docs/change/`
  - reusable troubleshooting knowledge -> 如本轮值得复用，再补 `docs/lessons/`

#### Related Requirements / Specs / Lessons
- 已确认本次不改变长期产品需求或技术契约，只修复既有 Win 构建链占位策略。
- 相关历史归档：
  - `docs/change/2026-02-09_remove-fyne.md`
  - `docs/change/2026-03-21_win-frontend-build-chain.md`

#### Executable Task List
- [x] `WINEMBED1` 收敛 `frontend/dist` 占位文件策略
- [x] `WINEMBED2` 更新前端 build 脚本和忽略规则
- [x] `WINEMBED3` 完成 `go mod tidy` / `go test` / `wails build` 验证
- [x] `DOC1` 归档本次修复并更新索引

#### Task Details
##### WINEMBED1 - Stabilize Dist Placeholder
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-embed-dist-placeholder`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-embed-dist-placeholder\plan.md`
- Goal: 让 `frontend/dist` 在仓库干净状态下始终存在可嵌入文件。
- Files / Modules:
  - `frontend/dist/*`
  - `.gitignore`
- Write Set:
  - 占位文件与忽略规则
- Acceptance:
  - `go:embed all:frontend/dist` 不再因空目录失败。
- Test Points:
  - `go mod tidy`
- Rollback:
  - 回退占位文件策略改动

##### WINEMBED2 - Align Build Script
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-embed-dist-placeholder`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-embed-dist-placeholder\plan.md`
- Goal: 确保前端 build 后自动重建占位文件。
- Files / Modules:
  - `frontend/package.json`
- Write Set:
  - build script only
- Acceptance:
  - `vite build` 后 `dist` 仍保留占位文件。
- Test Points:
  - `wails build -nopackage`
- Rollback:
  - 回退构建脚本改动

##### WINEMBED3 - Regression Validation
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-embed-dist-placeholder`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-embed-dist-placeholder\plan.md`
- Goal: 证明 Wails / Go 构建链恢复稳定。
- Files / Modules:
  - none
- Write Set:
  - none
- Acceptance:
  - `go mod tidy`、`GOWORK=off go test ./... -count=1`、`GOWORK=off wails build -nopackage` 全部通过。
- Test Points:
  - 同 Acceptance
- Rollback:
  - 停止发布并回到 `WINEMBED1` / `WINEMBED2`

##### DOC1 - Archive And Index
- Owner: Main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-embed-dist-placeholder`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-embed-dist-placeholder\plan.md`
- Goal: 记录 Win 构建链 `dist` 占位文件修复和验证结果。
- Files / Modules:
  - `docs/change/*`
  - `docs/change/README.md`
- Write Set:
  - archive docs only
- Acceptance:
  - 归档和索引完整。
- Test Points:
  - manual index check
- Rollback:
  - 回退本次归档文件

#### Dependencies
- `WINEMBED1` -> `WINEMBED2` -> `WINEMBED3` -> `DOC1`

#### Risks and Notes
- 控制面仓库当前已存在 `.keep` 删除，后续 workflow 结束时 merge 需要显式处理该差异。
- 本轮不并行；问题范围小，且验证依赖同一条构建链。
- `GOWORK=off wails build -nopackage` 已通过；日志中仍有 `Not found: time.Time` 的 Wails 侧提示，但不影响本次 `frontend/dist` embed 修复结论。

#### Parallelism Assessment
- Current decision: `不并行实现`
- Reason:
  - 改动集中在同一仓的同一构建链，读写面高度重叠，没有必要拆分。

### Stage 3.2 - Implementation Record
- `WINEMBED1`
  - 将 `frontend/dist` 的占位策略从隐藏 `.keep` 调整为普通文本文件 `placeholder.txt`。
- `WINEMBED2`
  - `.gitignore` 已对白名单切换到 `frontend/dist/placeholder.txt`。
  - `frontend/package.json` 的 `build` 脚本已改为在 `vite build` 后补回 `dist/placeholder.txt`。
- `WINEMBED3`
  - `go mod tidy` -> 通过
  - `GOWORK=off go test ./... -count=1` -> 通过
  - `GOWORK=off wails build -nopackage` -> 通过

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - 已覆盖 `go mod tidy` / Wails bindings 失败的根因和回归验证。
- 架构合理性：通过
  - 保持 `go:embed` 路径不变，只修复占位文件策略。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 仅调整构建时占位文件，不影响运行时路径。
- 可读性与一致性：通过
  - 规则统一收敛到 `.gitignore`、`frontend/dist`、`frontend/package.json` 三处。
- 可扩展性与配置化：通过
  - 不新增工具链分支或环境特判。
- 稳定性与安全：通过
  - 构建前和构建后都保证 `frontend/dist` 可嵌入。
- 测试覆盖情况：通过
  - `go mod tidy`、`go test`、`wails build` 三条关键链路全部验证。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未使用子 Agent。

### Stage 4 - Archive Record
- repo archive
  - `docs/change/2026-03-25_win-embed-dist-placeholder.md`
- lesson updates
  - `docs/lessons/wails-embed-dist-placeholder.md`
  - `docs/lessons/README.md`
- index updates
  - `docs/change/README.md`

#### Issue List
- none

阻塞：否
Stage 4 完成，等待用户确认是否结束 workflow
