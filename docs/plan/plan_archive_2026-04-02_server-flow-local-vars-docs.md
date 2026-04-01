# Plan - server-local-vars-clean

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Server`
- Branch: `feat/local-vars-docs-clean`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\server-local-vars-clean`
- Current Stage: `4 Change Archive`

## Stage Records

### Initialization
- `guide.md`: 已阅读，确认子协议长期规范统一放在 `repo\MyFlowHub-Server\docs`
- base/worktree confirmation:
  - base=`main`
  - 本仓负责稳定 requirements/specs 收口
  - 旧 `server-local-vars-docs` 仅作参考，不直接提交

### Stage 1 - Requirements Analysis
#### Goal
- 将 `set_var`、`flow_var`、`detail` 补回主线稳定 requirements

#### Scope
- 必须：
  - 更新 `docs/requirements/flow_data_dag.md`
  - 明确 `set_var` / `flow_var` 与 `varstore` 的边界
  - 明确 `detail` 与 `status` 的职责分离
- 可选：
  - 无
- 不做：
  - 不在本仓实现运行时代码
  - 不扩展 branch/foreach 等新节点

#### Use Cases
- 开发者以长期 requirements 为依据实现 run-local vars 和结果详情查询

#### Functional Requirements
- requirement 必须覆盖 run-local vars 生命周期和读取规则
- requirement 必须覆盖 detail 的查询维度和错误边界

#### Non-functional Requirements
- requirements 只保留长期真相
- 不把 change 归档当稳定真相

#### Inputs / Outputs
- 输入：
  - 旧 `server-local-vars-docs` 未提交文档改动
  - 主线现有 `docs/requirements/flow_data_dag.md`
- 输出：
  - 主线 clean branch 上更新后的 `docs/requirements/flow_data_dag.md`

#### Edge Cases
- run-local vars 不得与 `varstore` 混淆
- detail 第一版边界必须写清，避免调用方误以为能查整次 run 全量 vars

#### Acceptance Criteria
1. requirements 明确包含 `set_var`
2. requirements 明确包含 `flow_var`
3. requirements 明确包含 `detail` 独立查询能力

#### Risks
- 若 requirements 不更新，Proto/SubProto/Win 后续会继续各自推断，重新分叉

### Stage 2 - Architecture Design
#### Goal
- 将 `set_var`、`flow_var`、`detail` 补回主线稳定 specs

#### Overall Solution
- 以旧 dirty docs 为基础，把 flow 协议正式节点类型、binding source、detail action 契约同步到主线 `flow.md`

#### Alternatives Considered
- 备选：只写 change，不改 specs
- 不采用原因：这会把 archive 误当成长期真相，违背 `$m-docs` 路由规则

#### Module Responsibilities
- `docs/specs/flow.md`
  - 正式协议契约
- `docs/requirements/flow_data_dag.md`
  - 长期需求边界和验收标准
- `docs/change/README.md`
  - 仅在 stage 4 追加归档入口，不替代 requirements/specs

#### Data / Call Flow
- requirements 定义业务边界
- specs 定义 action、graph、binding、runtime contract
- Proto/SubProto/Win 以这里为长期真相

#### Interface Drafts
- action:
  - `detail`
  - `detail_resp`
- node kinds:
  - `call`
  - `compose`
  - `set_var`
- binding source kinds:
  - `node_result`
  - `trigger`
  - `flow_meta`
  - `run_meta`
  - `flow_var`

#### Error Handling and Safety
- 规范必须写清 `400/404` 边界
- 明确 `status` 不默认返回完整结果和 vars

#### Performance and Testing Strategy
- 文档以人工自洽校验为主
- 通过下游 Proto/SubProto clean worktree 联调验证规范落地

#### Extensibility Design Points
- 为后续 vars 调试视图、run dump、run archive 留出扩展空间

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- 目标：把旧 `server-local-vars-docs` 的稳定文档改动迁移到 clean branch
- 当前状态：
  - 主线 docs 仍停留在 `call/compose + status`
  - 旧 dirty docs 已经把 local vars/detail 写入 requirements/specs，但未提交

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Canonical destination:
  - 稳定行为边界 -> `docs/requirements/flow_data_dag.md`
  - 稳定技术契约 -> `docs/specs/flow.md`
  - workflow 结果 -> stage 4 `docs/change`
- 本仓根部 `todo.md` 为 workflow 控制面例外

#### Related Requirements / Specs / Lessons
- Requirements impact: `add`
- Specs impact: `add`
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\server-local-vars-clean\docs\requirements\flow_data_dag.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\server-local-vars-clean\docs\specs\flow.md`
- Related lessons:
  - 无

#### Executable Task List
- `SERV-DOC-1`：迁移 flow_data_dag requirements
- `SERV-DOC-2`：迁移 flow specs

#### Task Details
##### SERV-DOC-1 - flow data dag requirements
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\server-local-vars-clean`
- Plan Path: `D:\project\MyFlowHub3\worktrees\server-local-vars-clean\todo.md`
- Goal: 将 local vars 和 detail 的业务边界补回主线 requirements
- Files / Modules:
  - `docs/requirements/flow_data_dag.md`
- Write Set:
  - `docs/requirements/flow_data_dag.md`
- Acceptance:
  - `set_var + flow_var` 与 `varstore` 边界清晰
  - `detail` 查询边界清晰
- Test Points:
  - 人工自洽校验
- Rollback:
  - 回退 requirements 改动

##### SERV-DOC-2 - flow protocol specs
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\server-local-vars-clean`
- Plan Path: `D:\project\MyFlowHub3\worktrees\server-local-vars-clean\todo.md`
- Goal: 将 `detail` action、`set_var` node、`flow_var` binding source 补回主线 specs
- Files / Modules:
  - `docs/specs/flow.md`
- Write Set:
  - `docs/specs/flow.md`
- Acceptance:
  - flow 正式节点类型包含 `set_var`
  - binding source 包含 `flow_var`
  - action 包含 `detail/detail_resp`
- Test Points:
  - 人工自洽校验
  - 对照 Proto/SubProto clean worktree 联调
- Rollback:
  - 回退 specs 改动

#### Dependencies
- 可与 Proto/SubProto clean worktree 并行推进
- 最终以本仓 docs 作为长期真相收口

#### Risks and Notes
- 旧 dirty docs 含 change README 改动，本轮只迁 requirements/specs，不把 archive 杂项提前带入
- lessons 当前没有已知必须新增项，stage 4 再复核

#### Parallelism Assessment
- 不使用子Agent
- 原因：
  - 文档写集集中，且需和代码迁移同步核对

#### Issue List
- none

阻塞：否
进入 3.2

## Current State
- `SERV-DOC-1` 已完成
- `SERV-DOC-2` 已完成
- 已归档到 `docs/change/2026-04-02_server-flow-local-vars-docs.md`
