# Plan - root run-dev Win Wails GOWORK

## Workflow Information
- Repo: `D:\project\MyFlowHub3`
- Branch: `fix/root-run-dev-wails-gowork`
- Base: `master`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-root-run-dev-wails-gowork`
- Current Stage: `4`
- Participating modules:
  - `scripts/run-dev.ps1`
  - `go.work`
  - `repo/MyFlowHub-Win`
- External dependencies:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
    - purpose: reproduce Wails bindings failure under workspace mode
  - `D:\project\MyFlowHub3\worktrees\proto-stream-subproto`
    - purpose: provide Win Stream development proto via repo-local replace

## Stage Records

### Initialization
- `guide.md`
  - 已读取 workspace 根 `D:\project\MyFlowHub3\guide.md`
  - 已读取 `$m-autoflow` 的 `references/initialization.md`、`references/stages.md`、`references/m-docs-integration.md`
  - 已读取 `$m-docs` 的 `SKILL.md`、`references/requirement-impact.md`、`references/lessons-rules.md`
- repo / branch / worktree confirmation
  - implementation repo: `D:\project\MyFlowHub3`
  - dedicated branch: `fix/root-run-dev-wails-gowork`
  - dedicated worktree: `D:\project\MyFlowHub3\worktrees\fix-root-run-dev-wails-gowork`
  - implementation will stay inside the worktree only
- baseline facts
  - `repo/MyFlowHub-Win` 当前 `go.mod` 已包含 `require github.com/yttydcs/myflowhub-proto v0.1.5`
  - `repo/MyFlowHub-Win` 当前 `replace github.com/yttydcs/myflowhub-proto => ../../worktrees/proto-stream-subproto`
  - `repo/MyFlowHub-Win` 在 `$env:GOWORK='off'` 下执行 `go mod tidy` 和 `wails generate module` 均通过
  - 同一目录在默认 `GOWORK=D:\project\MyFlowHub3\go.work` 下执行 `wails generate module` 失败，报：
    - `module github.com/yttydcs/myflowhub-proto provides package github.com/yttydcs/myflowhub-proto/protocol/stream and is replaced but not required`

### Stage 1 - Requirements Analysis
#### Goal
- 修复 workspace 根 `scripts/run-dev.ps1` 启动 Win 时的 Wails bindings 失败问题，使默认启动链路绕开根 `go.work` 对 Win 模块的错误 workspace 解析。

#### Scope
- 必须
  - 让 `scripts/run-dev.ps1` 默认启动 Win 时使用 `GOWORK=off`
  - 保留脚本对全局 `-GoWorkOff` 的已有支持
  - 为需要旧 workspace 行为的场景保留显式 opt-out
  - 在 root docs 中归档本次 root cause、验证方式与排查规则
- 可选
  - 若验证显示 MetricsNode 也受同类问题影响，再考虑复用相同策略
- 不做
  - 不修改 `repo/MyFlowHub-Win` 业务代码
  - 不修改根 `go.work` 的模块清单
  - 不把 `proto-stream-subproto` worktree 写进根 `go.work`

#### Use Cases
- 开发者直接执行 `.\scripts\run-dev.ps1`，Win 可以正常进入 `wails dev`
- 开发者仍可通过显式脚本参数选择保留 Win 的 workspace 模式

#### Functional Requirements
- Win 进程默认不得继承根 `go.work` 进入 Wails bindings 生成
- 脚本必须显式体现 Win 的 `GOWORK` 决策，而不是依赖调用者手工记忆
- 文档必须解释为什么 `repo/MyFlowHub-Win` 在 `GOWORK=on` 下失败、在 `GOWORK=off` 下成功

#### Non-functional Requirements
- 保持脚本变更面最小
- 不影响 Server 与 MetricsNode 的现有默认启动方式
- 命名清晰，避免用户误解全局 `-GoWorkOff` 与 Win 默认行为的优先级

#### Inputs / Outputs
- 输入
  - 根脚本 `scripts/run-dev.ps1`
  - 根 `go.work`
  - `repo/MyFlowHub-Win` 的 Wails 启动命令
- 输出
  - 更新后的 `scripts/run-dev.ps1`
  - 根 `docs/change` 与 `docs/lessons` 归档

#### Edge Cases
- 用户显式要求保留 Win 的 workspace 模式
- 全局 `-GoWorkOff` 与 Win 的单独行为同时存在时，优先级必须明确
- 根 `go.work` 未来再加入其它与 Win repo-local replace 冲突的模块时，仍应遵循同一排查思路

#### Acceptance Criteria
- 在 `repo/MyFlowHub-Win` 中：
  - `wails generate module` 在默认 `GOWORK=on` 下稳定复现当前错误
  - `$env:GOWORK='off'; wails generate module` 通过
- 脚本修复后，Win 默认启动命令等价于带 `GOWORK=off` 的 Wails 启动
- 脚本文档和 root lessons 能让后续开发者快速判断“该不该关 workspace 模式”

#### Risks
- 将 Win 默认切到 `GOWORK=off` 会让依赖根 `go.work` 做联调的 Win 场景变成显式 opt-in
- 如果后续还有其它 Wails app 也受同类问题影响，本轮只修 Win，可能还需 follow-up

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 方案 A（采用）
  - 在 `scripts/run-dev.ps1` 中为 Win 进程单独增加“默认 `GOWORK=off`，可显式 opt-out”的环境拼装
  - 继续保留现有全局 `-GoWorkOff`，作为对所有进程统一关 workspace 的更强开关
- 不采用方案
  - 修改根 `go.work`
    - 理由：会改变整个 workspace 的模块装配，副作用明显大于脚本级修复
  - 修改 `repo/MyFlowHub-Win/go.mod` 让它在 `GOWORK=on` 下强行兼容当前根 workspace
    - 理由：根因在 root workspace 与启动脚本组合，不是 Win 业务模块本身

#### Module Responsibilities
- `scripts/run-dev.ps1`
  - 负责决定 Win/Server/MetricsNode 各自的启动环境
- `docs/change/*`
  - 归档本次脚本行为调整与验证证据
- `docs/lessons/wails-binding-proto-drift.md`
  - 记录“Wails CLI 在 workspace 模式下被错误模块图污染”的排查规则

#### Data / Call Flow
1. 用户在 workspace 根执行 `.\scripts\run-dev.ps1`
2. 脚本计算 common env
3. 启动 Win 前，脚本额外决定 Win 是否强制 `GOWORK=off`
4. Win 进程执行 `wails dev`
5. Wails 先执行 `go mod tidy` / `wails generate module`
6. 由于 Win 不再继承根 `go.work`，repo-local `go.mod` / `replace` 按单模块图解析并通过

#### Interface Draft
- `scripts/run-dev.ps1`
  - 新增显式 opt-out 参数，例如：
    - `-WinUseWorkspace`
  - 默认行为：
    - Win：`GOWORK=off`
    - Server / MetricsNode：保持现状
  - 优先级：
    - `-GoWorkOff` 高于一切，仍对所有进程生效

#### Error Handling and Safety
- 参数名必须直接表达“Win 是否沿用 workspace”
- 脚本帮助文本和示例必须覆盖新行为，避免隐式变化难以发现

#### Performance and Testing Strategy
- 复现对照：
  - `wails generate module`
  - `$env:GOWORK='off'; wails generate module`
- 脚本验证：
  - 通过与脚本默认 Win 环境等价的命令验证
- 回归：
  - 保持 Server / MetricsNode 启动命令字符串未被无关改动

#### Extensibility Design Points
- 若后续 MetricsNode 也需要默认关闭 workspace，可复用同一进程级环境拼装模式
- 若未来要统一更复杂的 per-process env 策略，可继续抽取局部 helper，但本轮不做额外重构

#### Issue List
- none

### Stage 3.1 - Planning
#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口
- docs tree 无需 bootstrap 或 repair
- stable truth 不变，维持在现有 `requirements/specs`
- workflow result 进入：
  - `docs/change/2026-03-29_root-run-dev-wails-gowork.md`
- reusable troubleshooting knowledge 进入：
  - `docs/lessons/wails-binding-proto-drift.md`
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements
  - none
- Related specs
  - none
- Related lessons
  - `D:\project\MyFlowHub3\worktrees\fix-root-run-dev-wails-gowork\docs\lessons\wails-binding-proto-drift.md`

#### Executable Task List
- [x] `ROOTRUN-1` 修改 `scripts/run-dev.ps1`，让 Win 默认以 `GOWORK=off` 启动并保留 opt-out
- [x] `ROOTRUN-2` 执行 workspace/Wails 验证，确认默认 Win 路径等价于修复后的环境
- [x] `ROOTRUN-3` 完成 3.3 checklist
- [x] `ROOTRUN-4` 归档 `docs/change` 与 `docs/lessons`

#### Task Details
##### `ROOTRUN-1` - Update run-dev Win env
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-root-run-dev-wails-gowork`
- Goal
  - 让根脚本默认启动 Win 时关闭 workspace 模式
- Files
  - `scripts/run-dev.ps1`
- Acceptance
  - Win 启动命令在默认场景下显式包含 `GOWORK=off`
  - 用户可通过显式参数保留旧行为
- Tests
  - script text review
  - command-level validation
- Rollback
  - 回退 `scripts/run-dev.ps1`

##### `ROOTRUN-2` - Validate Wails behavior
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-root-run-dev-wails-gowork`
- Goal
  - 用实测证明问题只出现在 workspace 模式，脚本默认值切换即可覆盖用户报错链路
- Files
  - no additional implementation files required
- Acceptance
  - `repo/MyFlowHub-Win` 下 `wails generate module` 在默认 workspace 模式失败
  - `repo/MyFlowHub-Win` 下 `$env:GOWORK='off'; wails generate module` 通过
- Tests
  - `wails generate module`
  - `$env:GOWORK='off'; wails generate module`
- Rollback
  - no code rollback needed

##### `ROOTRUN-3` - Review
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-root-run-dev-wails-gowork`
- Goal
  - 对照需求、脚本边界和验证完整性完成 3.3 review
- Files
  - `plan.md`
- Acceptance
  - 3.3 checklist 逐项判定
- Tests
  - review checklist
- Rollback
  - 更新 `plan.md` review 记录

##### `ROOTRUN-4` - Archive
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-root-run-dev-wails-gowork`
- Goal
  - 在 root docs 中沉淀本次 run-dev / Wails workspace 排查结果
- Files
  - `docs/change/2026-03-29_root-run-dev-wails-gowork.md`
  - `docs/change/README.md`
  - `docs/lessons/wails-binding-proto-drift.md`
  - `docs/lessons/README.md`
- Acceptance
  - 变更和 lesson 均可脱离当前对话独立复用
- Tests
  - manual doc review
- Rollback
  - 回退新增/更新文档

#### Dependencies
- `repo/MyFlowHub-Win`
  - 作为真实复现与验证目标

#### Risks and Notes
- 当前 worktree 的 `plan.md` 仅用作本轮控制文档；workflow 收口时需要把正文归档到 `docs/plan/`，并将根主线 `plan.md` 恢复为全局索引形态
- 这次修复的是 root 启动脚本默认行为，不是 `repo/MyFlowHub-Win` 的 semver 依赖收口

#### Parallelism Assessment
- 不派发子Agent
- 原因
  - 当前会话未获得显式子Agent授权
  - 变更面集中在一个脚本和少量 docs，串行更快

阻塞：否
进入 3.2

### Stage 3.2 - Implementation
#### File-level Change Summary
- `ROOTRUN-1`
  - `scripts/run-dev.ps1`
    - 新增 `-WinUseWorkspace` 参数，作为 Win 保留根 `go.work` 的显式 opt-out
    - Win 启动分支在未指定 `-WinUseWorkspace` 且未使用全局 `-GoWorkOff` 时，追加 `GOWORK=off`
    - 增加 `Win GOWORK：...` 日志，明确当前 Win 的模块解析模式
- `ROOTRUN-2`
  - `repo/MyFlowHub-Win`
    - 对照验证 `wails generate module` 在默认 workspace 模式失败
    - 对照验证 `$env:GOWORK='off'; wails generate module` 通过
- `ROOTRUN-4`
  - `docs/change/2026-03-29_root-run-dev-wails-gowork.md`
  - `docs/change/README.md`
  - `docs/lessons/wails-binding-proto-drift.md`
  - `docs/lessons/README.md`
    - 归档本次 root cause、验证路径与长期排查线索

#### Design Notes
- 保持 `-GoWorkOff` 为全局最高优先级，避免新增参数改变既有全局行为。
- Win 的默认 `GOWORK=off` 只在 Win 进程级注入，不污染 Server / MetricsNode 的默认启动环境。
- 采用显式 `-WinUseWorkspace`，把“需要 workspace 联调”的路径收敛为 opt-in，默认路径优先稳定可复现。

#### Validation / Evidence
- root worktree
  - `pwsh -NoProfile -File .\scripts\run-dev.ps1 -SkipServer -SkipWin -SkipMetricsNode`
  - 结果：通过
  - 目的：验证脚本语法、参数解析和 help block 不被新参数破坏
- `repo/MyFlowHub-Win`
  - `wails generate module`
  - 结果：失败
  - 关键报错：`module github.com/yttydcs/myflowhub-proto provides package github.com/yttydcs/myflowhub-proto/protocol/stream and is replaced but not required`
- `repo/MyFlowHub-Win`
  - `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 说明：仍会输出 `Not found: time.Time`，但退出码为 `0`，属于现有非阻塞行为

#### Task Status
- `ROOTRUN-1`：完成
- `ROOTRUN-2`：完成
- `ROOTRUN-4`：完成

#### Issue List
- none

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - Win 默认启动路径已切到 `GOWORK=off`，并保留显式 opt-out
- 架构合理性：通过
  - 修复集中在 root 启动脚本，不改根 `go.work` 模块拓扑
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 仅增加一次进程级环境拼装和日志输出，无新增热路径开销
- 可读性与一致性：通过
  - 参数命名与日志语义直接对应 Win 的 workspace 行为
- 可扩展性与配置化：通过
  - 进程级 env 拼装模式可复用到其它 Wails app，但本轮未提前泛化
- 稳定性与安全：通过
  - 默认值从“依赖调用者记忆”改为“脚本内显式安全默认值”
- 测试覆盖情况：通过
  - 已完成脚本解析验证与 `GOWORK=on/off` 实测对照
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 未使用子Agent；任务映射、验证证据和归档均已记录

Review 结论：通过
进入 4

### Stage 4 - Change Archive
#### Archive Outputs
- 已新增 `docs/change/2026-03-29_root-run-dev-wails-gowork.md`
- 已更新 `docs/change/README.md`
- 已更新 `docs/lessons/wails-binding-proto-drift.md`
- 已更新 `docs/lessons/README.md`

#### Docs Routing Check
- Requirements impact：`none`
- Specs impact：`none`
- Lessons impact：`updated`
- 可复用排查知识已从 `docs/change` 提升到 `docs/lessons`，未只留在单次归档里

#### Workflow Status
- 本轮归档已完成
- 当前工作树仍保留 workflow 控制版 `plan.md`
- 待用户确认结束 workflow 后，再执行：
  - 归档本工作树 `plan.md` 到根 `docs/plan/`
  - 将根 `plan.md` 恢复为全局索引形态
  - merge / worktree cleanup

#### Issue List
- none
