# 2026-04-04 Flow 完备性审查

## 当前阶段
- Stage 1 已完成
- Stage 2 已完成
- Stage 3.1 已完成
- Stage 3.2 已完成
- Stage 3.3 已完成
- Stage 4 已完成

## 仓库与执行上下文
- Control Repo: `D:\project\MyFlowHub3`
- Control Base branch: `master`
- Control Worktree branch: `chore/flow-completeness-audit`
- Control Worktree path: `D:\project\MyFlowHub3\worktrees\chore-flow-completeness-audit`
- Active control doc: `D:\project\MyFlowHub3\worktrees\chore-flow-completeness-audit\todo.md`
- Inspection targets:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Proto`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-SubProto`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Notes:
  - 本轮是只读审查 workflow，不在 inspection targets 内做实现改动。
  - 若审查结论进入修复阶段，再为对应 repo 单独创建 worktree 与计划文档。

## 使用 $m-docs 的文档路由结论
- 文档分类: `plan` + `change`
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\requirements\flow-editor-visual-form.md`
- Related specs:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\specs\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\docs\specs\protocol_map.md`
- Related lessons:
  - `none`
- Canonical destination:
  - stable truth: `requirements` / `specs`
  - workflow control doc: 当前 control worktree 根 `todo.md`
  - completed review archive: `D:\project\MyFlowHub3\docs\change\2026-04-04_flow-completeness-review.md`
  - reusable troubleshooting knowledge: 只有在发现可复用排查规律时才进入 `docs/lessons`

## Stage 1 - 需求分析

### 目标
- 判断当前 `MyFlowHub3` 的 `flow` 能力是否在以下四层达到“完备”：
  - 稳定 requirements/specs 已定义且彼此一致
  - Proto / SubProto / Server runtime 已覆盖稳定契约
  - Win authoring / deploy / run / detail 消费路径与稳定契约一致
  - 已存在的测试与归档证据足以支撑“主能力闭环”

### 范围

#### Must
- 审查 `flow` 的稳定契约、协议映射、后端 runtime、Win surface、验证闭环。
- 给出缺口、风险、残余验证盲区和建议优先级。

#### Optional
- 指出文档、测试或产品面已经具备但仍需补强的次级项。

#### Not In Scope
- 本轮不实现功能。
- 本轮不修改 requirements、specs、runtime、UI。
- 本轮不做线上运行或环境写入。

### 使用场景
- 需要判断 `flow` 是否已从“可演示”进入“相对完备可交付”。
- 需要知道当前缺的是契约、实现、Win 入口，还是测试与验证闭环。
- 需要基于现有 repo 与归档证据，给出下一步补强顺序。

### 功能需求
- 审查必须覆盖稳定真相源：
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\requirements\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\specs\flow-editor-visual-form.md`
- 审查必须核对 Proto `protocol/flow` 与 workspace `docs/specs/protocol_map.md` 的动作和字段映射。
- 审查必须核对 SubProto `flow` runtime 是否覆盖：
  - `set/delete/run/cancel_run/status/detail/list_runs/list/get`
  - `trigger: interval / cron / event / var_changed`
  - `node kinds: call / compose / transform / set_var / branch / foreach / subflow`
  - `graph validation / active-run / dedup / retry backoff / archive / permission`
- 审查必须核对 Win 是否覆盖：
  - authoring: editor ordinary mode / deploy trigger / project metadata
  - consumption: `run / status / detail / list / get / cancel_run / list_runs`
  - Wails / service / store / page 的真实用户入口
- 审查必须给出明确结论：`complete`、`partial` 或 `not complete`，并列出依据。

### 非功能需求
- 结论必须以 repo 与 docs 证据为准，不能只依据历史对话。
- 结论必须区分：
  - 文档存在
  - 协议存在
  - runtime 存在
  - UI 可见入口存在
  - 测试或验证证据存在
- 对不确定项必须标成残余风险，不能推断为“已完备”。

### 输入输出
- 输入:
  - workspace docs
  - repo source
  - repo-local docs
  - 历史 change / plan 归档
- 输出:
  - 审查结论
  - 缺口清单
  - 风险与残余盲区
  - 下一步建议顺序

### 边界异常
- 若 requirements/specs 与代码冲突，以稳定 requirements/specs 为准，并把偏差记为缺口。
- 若 change 归档声称“已完成”，但 repo 当前 surface 不可见，只能算历史证据，不能直接算完备。
- 若 workspace 根存在未提交 flow 文档，只作为线索，不替代 canonical docs。

### 验收标准
- 能指出 `flow` 当前是 `complete`、`partial` 还是 `not complete`。
- 结论至少覆盖契约面、runtime 面、Win 面、验证面四类。
- 每个主要缺口都能落到具体 repo、文件或动作级别。

### 风险
- flow 最近迭代密集，`docs/change` 可能比主线仓源码更超前。
- workspace 根存在未提交 flow 相关文档，需避免误当 canonical truth。
- Win surface 可能具备内部方法但没有用户可见入口，需要同时看 service、store、page。

## Stage 2 - 架构设计

### 总体方案（含选型理由 / 备选对比）
- 采用“分层闭环审查”，按 `requirements/specs -> Proto/SubProto/Server -> Win -> tests/change evidence` 逐层核对。
- 不采用“只看 Proto”或“只看 Win 页面”的单层审查，因为这两种都会把局部覆盖误判成整体完备。
- 结论口径:
  - `complete`: 四层均闭环，且残余风险不影响主能力交付
  - `partial`: 主链路大体成立，但仍有用户面、契约面或验证面缺口
  - `not complete`: 主链路存在明显断点

### 模块职责
- `MyFlowHub-Proto`
  - `protocol/flow/*` 提供稳定 wire、action、payload 定义
- `MyFlowHub-SubProto`
  - `flow/*` 提供 flow runtime、graph 校验、trigger、archive、handler 与测试
- `MyFlowHub-Server`
  - `docs/requirements` / `docs/specs` 提供跨仓稳定真相
  - `modules/defaultset` / runtime 装配作为服务端集成边界
- `MyFlowHub-Win`
  - `frontend/src/stores/flow*.ts`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/pages/Flow.vue`
  - 相关 services 与 tests
  - 提供 authoring、deploy、run、detail、status、list 等用户可见入口
- `MyFlowHub3/docs/change`
  - 只作为历史实施与验证证据

### 数据 / 调用流
- 稳定需求和规范由 Server / Win docs 定义。
- Proto 将动作和 payload 固化为 wire contract。
- SubProto 实现 handler、graph validation、runtime execution、trigger、archive。
- Server 负责将 flow handler 装配进默认模块集并提供集成测试边界。
- Win 通过 service -> store -> page / window 暴露 authoring 与 runtime 消费路径。
- change / plan 归档仅作为“做过什么、测过什么”的旁证，不提升为长期真相。

### 接口草案
- 动作集合:
  - `set`
  - `delete`
  - `run`
  - `cancel_run`
  - `status`
  - `detail`
  - `list_runs`
  - `list`
  - `get`
- 关键能力字段:
  - `max_active_runs`
  - `trigger.dedup_window_ms`
  - `flow.run_archive.backend`
- 关键节点种类:
  - `call`
  - `compose`
  - `transform`
  - `set_var`
  - `branch`
  - `foreach`
  - `subflow`

### 错误与安全
- 权限必须区分 `flow.set` / `flow.delete` / `flow.run` / `flow.read`。
- 非法图、非法 trigger、非法 node kind、非法 field 组合必须在保存前或运行前显式拒绝。
- 审查过程中若发现仅内部 helper 存在而无用户入口，不算 surface 完备。

### 性能与测试策略
- 先用 docs 与源码静态核对确定 coverage matrix，再抽查关键测试与装配测试。
- 对 runtime 面重点看：
  - graph / trigger / runtime_fix / orchestrator / capability provider / delete tests
- 对 Win 面重点看：
  - frontend vitest
  - Go service tests
  - 页面或窗口是否真的暴露动作入口

### 可扩展性设计点
- 若本轮判定为 `partial`，后续修复可拆成独立工作流：
  - contract alignment
  - runtime gap repair
  - Win surface repair
  - integration / evidence repair
- 本轮计划保持只读，避免把审查与修复混在同一 workflow。

## Related Requirements / Specs / Lessons
- Related requirements:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\requirements\flow-editor-visual-form.md`
- Related specs:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\specs\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\docs\specs\protocol_map.md`
- Related lessons:
  - `none`

## 可执行任务清单（Checklist）
- [x] `FLOW-AUDIT-1` 契约面审查
- [x] `FLOW-AUDIT-2` runtime / Server 装配与测试面审查
- [x] `FLOW-AUDIT-3` Win surface 审查
- [x] `FLOW-AUDIT-4` 历史证据、残余风险与结论整合

## Task Details

### `FLOW-AUDIT-1` - Contract Coverage
- Owner:
  - delegated explorer or main agent
- Worktree:
  - `D:\project\MyFlowHub3\worktrees\chore-flow-completeness-audit`
- Plan Path:
  - `D:\project\MyFlowHub3\worktrees\chore-flow-completeness-audit\todo.md`
- Goal:
  - 核对 requirements / specs / protocol_map / Proto payload 是否一致
- Files / Modules:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\requirements\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\specs\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Proto\protocol\flow\types.go`
  - `D:\project\MyFlowHub3\docs\specs\protocol_map.md`
- Acceptance:
  - 产出契约一致性结论和缺口列表
- Tests:
  - 只读核对文档与类型定义
- Rollback:
  - `none`
- Write Set:
  - `none`

### `FLOW-AUDIT-2` - Runtime / Server Coverage
- Owner:
  - delegated explorer or main agent
- Worktree:
  - `D:\project\MyFlowHub3\worktrees\chore-flow-completeness-audit`
- Plan Path:
  - `D:\project\MyFlowHub3\worktrees\chore-flow-completeness-audit\todo.md`
- Goal:
  - 核对 SubProto runtime、Server 装配和关键测试是否支撑稳定契约
- Files / Modules:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-SubProto\flow`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\modules`
  - `D:\project\MyFlowHub3\docs\change\*.md` 中与 flow 相关归档
- Acceptance:
  - 明确主链路覆盖、缺口、装配和测试盲区
- Tests:
  - 静态审查源码与现有测试
- Rollback:
  - `none`
- Write Set:
  - `none`

### `FLOW-AUDIT-3` - Win Surface Coverage
- Owner:
  - delegated explorer or main agent
- Worktree:
  - `D:\project\MyFlowHub3\worktrees\chore-flow-completeness-audit`
- Plan Path:
  - `D:\project\MyFlowHub3\worktrees\chore-flow-completeness-audit\todo.md`
- Goal:
  - 核对 Win 的 authoring 与 run-consumption surface 是否真的对用户可见
- Files / Modules:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\internal\services\flow`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\frontend\src\stores\flow*.ts`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\frontend\src\pages\Flow.vue`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\frontend\src\windows\FlowEditorWindow.vue`
  - 相关 tests
- Acceptance:
  - 明确 Win 已覆盖面、缺失动作和仅内部存在但未暴露的 surface
- Tests:
  - 静态审查源码与现有测试
- Rollback:
  - `none`
- Write Set:
  - `none`

### `FLOW-AUDIT-4` - Evidence Synthesis
- Owner:
  - main agent
- Worktree:
  - `D:\project\MyFlowHub3\worktrees\chore-flow-completeness-audit`
- Plan Path:
  - `D:\project\MyFlowHub3\worktrees\chore-flow-completeness-audit\todo.md`
- Goal:
  - 汇总 contract、runtime、Win、history evidence，形成最终结论
- Files / Modules:
  - 本计划引用的 requirements、specs、source、tests、change docs
- Acceptance:
  - 形成可归档的审查结论、优先级建议和残余风险
- Tests:
  - 证据交叉复核
- Rollback:
  - `none`
- Write Set:
  - `none`

## 并行性评估（进入 3.2 后执行）
- `FLOW-AUDIT-1`、`FLOW-AUDIT-2`、`FLOW-AUDIT-3` 可独立并行，且均为只读任务。
- 计划在 `3.2` 使用 explorer 子 agent 执行：
  - 契约面
  - runtime / Server 面
  - Win surface 面
- `FLOW-AUDIT-4` 保留由主 agent 统一整合与复核。

## 风险与备注
- workspace 根目录存在未提交的 flow 审查与修复草稿；它们仅作线索，不作为本轮稳定真相。
- 若静态证据不足以判定“complete”，默认降级为 `partial`，并把不足点显式列出。
- 若发现某个缺口实质是 in-progress 未提交改动，需要把它标成“当前主线未闭环”，不能提前记为完成。

## Stage 3.2 / 3.3 结果
- `FLOW-AUDIT-1`
  - 已完成。顶层 action / permission / payload contract 基本一致，但 graph / node canonical contract 仍主要依赖 Server / Win specs，而非 Proto 强类型。
- `FLOW-AUDIT-2`
  - 已完成。SubProto runtime 覆盖面很广，Server 默认装配存在 targeted tests，但 `flow` 模块在当前 workspace 模式下仍有失败测试，且 Server 侧缺少行为级 end-to-end 装配验证。
- `FLOW-AUDIT-3`
  - 已完成。Win authoring 与 run-consumption surface 已闭环，`cancel_run` / `list_runs` 经过 service -> store -> editor toolbar 打到真实入口，并有前端 / Go 测试覆盖。
- `FLOW-AUDIT-4`
  - 已完成。主结论为 `partial`，不是 `complete`。

## Stage 3.3 - Review
- 需求覆盖：通过
- 架构合理性：通过
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：不通过
  - 原因：`go test github.com/yttydcs/myflowhub-subproto/flow/... -count=1 -p 1` 在当前 workspace 模式下失败，暴露 `TestLoadFlowsFromDiskKeepsLegacyKindsForCompatibility` 仍未收口。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过

## Stage 4 - Archive
- 已使用 `$m-docs` 完成 impact 检查：
  - Requirements impact: `none`
  - Specs impact: `none`
  - Lessons impact: `none`
- 已新增归档：
  - `D:\project\MyFlowHub3\worktrees\chore-flow-completeness-audit\docs\change\2026-04-04_flow-completeness-review.md`
- 已更新索引：
  - `D:\project\MyFlowHub3\worktrees\chore-flow-completeness-audit\docs\change\README.md`

## 阻塞
- 否
- Stage 4 已完成
- 等待用户确认是否结束 workflow

## Iteration 2 - 移除旧 flow 数据兼容

### 当前阶段
- Stage 1 已完成
- Stage 2 已完成
- Stage 3.1 已完成
- Stage 3.2 已完成
- Stage 3.3 已完成
- Stage 4 已完成

### 用户确认的新范围
- 用户明确说明：项目尚未上线，不需要保留旧 flow 数据兼容。
- 本轮目标改为：
  - 删除 `MyFlowHub-SubProto/flow` 对旧 `local/exec` 节点的运行期兼容
  - 删除旧格式磁盘 flow 的加载兼容
  - 同步删除对应测试与 repo-local 归档口径
- 本轮不处理：
  - `max_active_runs` 的 legacy 默认语义
  - `run_archive_enabled` 的 legacy 配置兼容
  - Win `call-only` 读兼容

### `$m-docs` 路由与 impact 检查
- 文档分类：
  - 控制面：`plan`
  - 完成归档：目标 repo 的 `change`
- Requirements impact:
  - `none`
- Specs impact:
  - `none`
- Notes:
  - 当前稳定 `requirements/specs` 未把 `local/exec` 运行兼容作为长期真相。
  - 需要同步更新 `repo/MyFlowHub-SubProto/docs/change` 中会误导后续排查的历史兼容口径。

### 实现 worktree
- Repo:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-SubProto`
- Branch:
  - `fix/flow-drop-legacy-compat`
- Worktree path:
  - `D:\project\MyFlowHub3\worktrees\fix-flow-drop-legacy-compat`
- Worktree plan path:
  - `D:\project\MyFlowHub3\worktrees\fix-flow-drop-legacy-compat\plan.md`

### 任务拆分
- `FLOW-FIX-1`
  - 删除 `flow/handler.go` 中 `local/exec` 节点 decode 兼容
- `FLOW-FIX-2`
  - 删除或改写旧格式兼容测试，保留 call-only 契约与非法旧格式被拒绝的断言
- `FLOW-FIX-3`
  - 更新 `repo/MyFlowHub-SubProto/docs/change` 索引与本轮归档，明确项目不再支持旧 flow 数据

### 验收
- `flow.set` 继续只接受当前正式节点类型
- `loadFlowsFromDisk()` 不再保留旧 `local/exec` 格式
- runtime 执行路径不再支持 `local/exec`
- 定向 Go 测试通过，且不存在对旧格式仍“应成功”的断言

### Stage 3.2 结果
- `FLOW-FIX-1`
  - 已完成。`flow/handler.go` 删除 `legacyLocalSpec` / `legacyExecSpec`，`decodeNodeCallSpec(...)` 只接受 `kind=call`。
- `FLOW-FIX-2`
  - 已完成。旧兼容测试已改写为严格拒绝口径：
    - `TestLoadFlowsFromDiskSkipsLegacyKinds`
    - `TestFlowLegacyLocalNodeRejectedAtRuntime`
- `FLOW-FIX-3`
  - 已完成。目标 repo 已新增归档 `D:\project\MyFlowHub3\worktrees\fix-flow-drop-legacy-compat\docs\change\2026-04-05_flow-drop-legacy-compat.md`，并更新 `docs/change/README.md`。
- 设计补充：
  - 仅删除运行期 decode 分支还不够；`loadFlowsFromDisk()` 也必须重跑 `validateGraph(...)`，否则旧 JSON 定义仍会在重启后回流进内存。
- 实际验证：
  - `D:\project\MyFlowHub3`
    - `GOWORK=D:\project\MyFlowHub3\.tmp\verify-flow-drop-legacy\go.work go test github.com/yttydcs/myflowhub-subproto/flow/... -count=1 -p 1`
    - 结果：通过
  - `D:\project\MyFlowHub3`
    - `GOWORK=D:\project\MyFlowHub3\.tmp\verify-flow-drop-legacy\go.work go test github.com/yttydcs/myflowhub-subproto/flow/... -run 'TestLoadFlowsFromDiskSkipsLegacyKinds|TestFlowLegacyLocalNodeRejectedAtRuntime|TestValidateGraphRejectsLegacyKind' -count=1 -p 1 -v`
    - 结果：通过

### Stage 3.3 - Review
- 需求覆盖：通过
- 架构合理性：通过
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：通过
  - 说明：本轮计划内 full package 验证与定向 legacy 回归均已通过。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 说明：本轮未使用子Agent。

### Stage 4 - Archive
- 已使用 `$m-docs` 完成 impact 检查：
  - Requirements impact: `none`
  - Specs impact: `none`
  - Lessons impact: `none`
- 归档落点：
  - 控制面继续使用当前 `todo.md` 记录本轮 stage 结果
  - 完成归档使用目标 repo 的 `change`，避免在 workspace 控制面重复写同一份实现真相
- 已新增归档：
  - `D:\project\MyFlowHub3\worktrees\fix-flow-drop-legacy-compat\docs\change\2026-04-05_flow-drop-legacy-compat.md`
- 已更新索引：
  - `D:\project\MyFlowHub3\worktrees\fix-flow-drop-legacy-compat\docs\change\README.md`
- Lessons decision：
  - 暂不单独新增 `docs/lessons`
  - 原因：本轮规律已在 repo-local `change` 中记录完整快速检查线索，且问题范围集中在 `flow` 模块单点兼容收口，不需要提升为跨模块长期排障知识

## 阻塞
- 否
- Iteration 2 的 Stage 4 已完成
- 等待用户确认是否结束 workflow
