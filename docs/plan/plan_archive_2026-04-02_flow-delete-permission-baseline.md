# Plan - subproto-delete-baseline

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-SubProto`
- Branch: `fix/delete-baseline-mainline`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-delete-baseline`
- Current Stage: `4 Change Archive`

## Stage Records

### Initialization
- `guide.md`: 未发现 repo-local `guide.md`，沿用 workspace 根 `guide.md` 约束
- base/worktree confirmation:
  - base=`main`
  - 本轮仅涉及 `MyFlowHub-SubProto/flow`
  - 已创建独立 worktree：`D:\project\MyFlowHub3\worktrees\subproto-delete-baseline`
  - 本轮不在主仓路径直接改业务代码

### Stage 1 - Requirements Analysis
#### Goal
- 处理 `flow` 模块当前 delete 基线失败，使测试期望与现行 `flow.delete` 权限模型保持一致。

#### Scope
- 必须：
  - 复核 delete 稳定 requirements/specs 与当前实现是否一致
  - 修复 `TestFlowDeleteSuccess`
  - 修复 `TestFlowDeleteNotFound`
  - 修复 `TestFlowDeleteInterruptsActiveRun`
  - 修复 `TestFlowDeleteFileFailureKeepsState`
  - 跑通 delete 相关目标测试，并尽量恢复整包 `flow` 回归
- 可选：
  - 若 helper 抽象不清晰，可做最小测试辅助收敛
- 不做：
  - 不改 `flow.delete` 权限语义
  - 不放宽默认 `node` 角色权限
  - 不扩展新的 flow action 或持久化语义

#### Use Cases
- 拥有 `flow.delete` 的调用方可以成功删除 flow、删除不存在 flow 时得到 `404`、删除运行中 flow 时能中断 run。
- 未拥有 `flow.delete` 的调用方继续得到 `403 permission denied`。

#### Functional Requirements
- delete 成功路径测试必须显式处于“具备 `flow.delete` 权限”的前置条件下
- delete 拒绝路径测试必须继续覆盖无 `flow.delete` 权限场景
- delete 文件删除失败场景必须在通过权限判定后命中 `500 delete file failed`
- 删除运行中 flow 时，run 仍需转为 `cancelled`

#### Non-functional Requirements
- 采用最小安全改动
- 不引入与当前 requirements/specs 冲突的新默认权限
- 测试前置应显式、可审计，避免继续依赖隐含默认值

#### Inputs / Outputs
- 输入：
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
  - 已有归档：
    - `docs/change/2026-03-21_subproto-flow-delete.md`
    - `docs/change/2026-03-26_auth-default-role-hierarchy.md`
    - `docs/change/2026-04-02_flow-local-vars-detail-mainline.md`
- 输出：
  - `flow/delete_test.go`
  - 需要时的其他 delete 相关测试文件
  - 本轮 `docs/change` 归档

#### Edge Cases
- 默认 `node` 角色不具备 `flow.delete`
- `auth.default_perms="*"` 在存在 `role_perms[node]` 时不会覆盖 node 角色权限集合
- 成功删除与 `permission denied` 场景必须继续可区分

#### Acceptance Criteria
1. 四个当前 delete 基线失败测试恢复通过
2. `TestFlowDeletePermissionDenied` 继续通过
3. `go test github.com/yttydcs/myflowhub-subproto/flow/... -count=1 -p 1` 通过
4. requirements/specs 无需改动或已明确记录为何无需改动

#### Risks
- 若误把问题当成运行时缺陷，可能错误放宽 `flow.delete` 权限
- 若测试辅助配置不集中，后续再引入默认权限调整时会继续漂移

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 保持 `flow.delete` 运行时权限判定不变，最小修正 delete 测试前置：
  - 需要 delete 成功的用例显式授予 `flow.delete`
  - 需要 delete 拒绝的用例保留当前 `node` / `flow.set` 受限前置

#### Alternatives Considered
- 备选：修改 `permission.Config`，让 `auth.default_perms` 覆盖 `role_perms[node]`
- 不采用原因：
  - 会改变全局 auth 默认语义
  - 与 `2026-03-26_auth-default-role-hierarchy` 收口后的稳定行为冲突
- 备选：在 `flow` handler 中对 delete 继续沿用 `flow.set`
- 不采用原因：
  - 直接违背稳定 spec 中 `flow.delete` 独立权限要求

#### Module Responsibilities
- `flow/delete_test.go`
  - 持有 delete 成功 / 失败 / 中断运行的权限前置
- `flow/runtime_fix_test.go`
  - 若涉及 delete 文件失败基线，保持权限前置与主用例一致
- `flow` 运行时代码
  - 本轮默认不改，除非验证后发现真实实现缺口

#### Data / Call Flow
- 测试构建 config
- `NewHandlerWithConfig(...)` 经 `runtimedeps.Resolve(...)` 获取 `permission.SharedConfig(cfg)`
- `handleDelete -> hasPermission(origin, permFlowDelete)`
- 权限通过后才进入 `applyDeleteLocal`

#### Interface Drafts
- 不新增外部接口
- 继续使用稳定权限：
  - `flow.set`
  - `flow.delete`

#### Error Handling and Safety
- 保留 `403 permission denied`
- 保留 `404 not found`
- 保留 `500 delete file failed`
- 不把权限失败伪装成删除失败

#### Performance and Testing Strategy
- 先跑 4 个失败 delete 测试
- 再跑 `TestFlowDeletePermissionDenied`
- 最后跑整包 `flow` 回归：`go test github.com/yttydcs/myflowhub-subproto/flow/... -count=1 -p 1`

#### Extensibility Design Points
- 把“成功删除必须显式拥有 `flow.delete`”固定在测试辅助里，后续增加 delete 相关测试时可直接复用

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- 目标：恢复 `flow` delete 基线测试，并确认问题属于测试授权前置漂移，而不是 delete 运行时回退
- 当前状态：
  - `TestFlowDeleteSuccess` / `NotFound` / `InterruptsActiveRun` / `FileFailureKeepsState` 统一返回 `403`
  - 当前稳定 spec 要求 delete 使用独立权限 `flow.delete`
  - `auth.default_role=node` 的默认 `role_perms` 不包含 `flow.delete`

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Requirements impact: `none`
- Specs impact: `none`
- Canonical destination:
  - 稳定行为边界 -> `repo/MyFlowHub-Server/docs/requirements/flow_data_dag.md`
  - 稳定技术契约 -> `repo/MyFlowHub-Server/docs/specs/flow.md`
  - 本轮结论与验证 -> stage 4 `docs/change`
- 相关 lessons：
  - 复用现有 `auth-default-role-hierarchy` 归档，不新增 lesson

#### Related Requirements / Specs / Lessons
- Related requirements:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
- Related specs:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- Related lessons:
  - 无

#### Executable Task List
- `DEL-BL-1`：收敛 delete 测试授权前置
- `DEL-BL-2`：回归 delete 失败集与整包 `flow`
- `DEL-BL-3`：stage 3.3 review 与归档

#### Task Details
##### DEL-BL-1 - align delete test permission baseline
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-delete-baseline`
- Plan Path: `D:\project\MyFlowHub3\worktrees\subproto-delete-baseline\todo.md`
- Goal: 让 delete 成功路径测试显式拥有 `flow.delete`
- Files / Modules:
  - `flow/delete_test.go`
  - 需要时的 `flow/runtime_fix_test.go`
- Write Set:
  - `flow/delete_test.go`
  - 可选 `flow/runtime_fix_test.go`
- Acceptance:
  - 4 个失败测试不再统一停在 `403`
  - `permission denied` 用例继续保留
- Test Points:
  - `TestFlowDeleteSuccess`
  - `TestFlowDeleteNotFound`
  - `TestFlowDeleteInterruptsActiveRun`
  - `TestFlowDeleteFileFailureKeepsState`
  - `TestFlowDeletePermissionDenied`
- Rollback:
  - 回退测试前置配置改动

##### DEL-BL-2 - regression verification
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-delete-baseline`
- Plan Path: `D:\project\MyFlowHub3\worktrees\subproto-delete-baseline\todo.md`
- Goal: 验证 delete 基线恢复且未引入其他 flow 回退
- Files / Modules:
  - 无新增计划内写集，以测试为主
- Write Set:
  - 无计划内新增写集
- Acceptance:
  - delete 目标测试通过
  - 整包 `flow` 回归通过
- Test Points:
  - `go test github.com/yttydcs/myflowhub-subproto/flow/... -run 'TestFlowDeleteSuccess|TestFlowDeleteNotFound|TestFlowDeleteInterruptsActiveRun|TestFlowDeleteFileFailureKeepsState|TestFlowDeletePermissionDenied' -count=1 -p 1`
  - `go test github.com/yttydcs/myflowhub-subproto/flow/... -count=1 -p 1`
- Rollback:
  - 不单独回滚，依赖 `DEL-BL-1`

##### DEL-BL-3 - review and archive
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-delete-baseline`
- Plan Path: `D:\project\MyFlowHub3\worktrees\subproto-delete-baseline\todo.md`
- Goal: 完成 stage 3.3 review 与本轮 `docs/change` 归档
- Files / Modules:
  - `docs/change/*`
- Write Set:
  - `docs/change/*`
- Acceptance:
  - review 记录完整
  - 本轮结论与验证可追溯
- Test Points:
  - 文档自检
- Rollback:
  - 删除本轮新增 archive 并恢复索引

#### Dependencies
- 单仓 workflow，无跨仓实现依赖
- requirements/specs 以 `MyFlowHub-Server/docs` 为准

#### Risks and Notes
- 根因已定位到测试授权前置，不要顺手改动 `permission.Config`
- `GOWORK=off` 直接在模块目录跑测试会受当前依赖/`go.sum` 口径影响，优先沿用 workspace 模式命令

#### Parallelism Assessment
- 不使用子Agent
- 原因：
  - 写集小且集中在 delete 测试
  - 当前关键路径是复现与单点修复，不值得拆分

#### Issue List
- none

阻塞：否
进入 3.2

## Stage 3.3 - Code Review
- 需求覆盖：通过
- 架构合理性：通过
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：通过
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过

## Current State
- `DEL-BL-1` 已完成
- `DEL-BL-2` 已完成
- `DEL-BL-3` 已完成
- 已归档到 `docs/change/2026-04-02_flow-delete-permission-baseline.md`
