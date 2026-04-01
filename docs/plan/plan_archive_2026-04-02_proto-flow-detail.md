# Plan - proto-local-vars-clean

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Proto`
- Branch: `feat/local-vars-proto-clean`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\proto-local-vars-clean`
- Current Stage: `4 Change Archive`

## Stage Records

### Initialization
- `guide.md`: 已阅读，确认 `repo/` 只做控制面，所有实现改动必须进入 `worktrees/`
- base/worktree confirmation:
  - base=`main`
  - 本仓为参与仓库之一
  - 本轮使用新的干净 worktree，旧 `proto-local-vars-observability` 仅作参考，不直接提交

### Stage 1 - Requirements Analysis
#### Goal
- 为 `flow.detail` 补齐 Proto 协议字典，使 `SubProto` 与 Win/Server 可共享正式 JSON 契约

#### Scope
- 必须：
  - 补齐 `detail` / `detail_resp` action 常量
  - 补齐 `DetailReq` / `DetailResp` payload 类型
  - 更新 `docs/protocol_map.md` 生成结果
- 可选：
  - 无
- 不做：
  - 不在本仓引入运行时逻辑
  - 不在本仓维护 `set_var` / `flow_var` 的业务执行语义

#### Use Cases
- `SubProto flow` 编译和运行 `detail` handler
- Win 服务层用共享 Proto 类型直接发送 `detail`

#### Functional Requirements
- `protocol/flow/types.go` 必须公开 `detail` / `detail_resp`
- `protocol/flow/types.go` 必须提供节点结果详情查询所需请求/响应结构
- 生成后的协议映射文档必须可检索到新增 action 和 payload

#### Non-functional Requirements
- 保持 wire 命名稳定
- 不引入非协议字典职责

#### Inputs / Outputs
- 输入：
  - 既有稳定 spec：`D:\project\MyFlowHub3\worktrees\server-local-vars-clean\docs\specs\flow.md`
- 输出：
  - `protocol/flow/types.go`
  - `docs/protocol_map.md`

#### Edge Cases
- 生成文档不得手工篡改 generated 区域外的无关内容
- payload 结构必须与 Win 当前本地 detail 类型保持 JSON 兼容

#### Acceptance Criteria
1. `protocol/flow/types.go` 暴露 `ActionDetail` / `ActionDetailResp`
2. `protocol/flow/types.go` 暴露 `DetailReq` / `DetailResp`
3. `go test ./...` 在本仓通过

#### Risks
- 若 Proto 契约与 Server spec 不一致，会导致 SubProto/Win 分别定义本地类型并继续漂移
- 若生成文档未同步，后续检索入口会失真

### Stage 2 - Architecture Design
#### Overall Solution
- 以 Server 稳定 spec 为唯一契约来源，在 Proto 中最小补齐 `detail` action 和 payload

#### Alternatives Considered
- 备选：继续让 Win 保持本地 detail 类型
- 不采用原因：会使 Proto/SubProto/Win 长期分叉，违背协议字典仓职责

#### Module Responsibilities
- `protocol/flow/types.go`
  - 承载 flow 子协议 action 与 payload 字典
- `docs/protocol_map.md`
  - 提供可检索的协议映射副本

#### Data / Call Flow
- Server spec 定义 detail 契约
- Proto 落地 action/payload
- SubProto 通过 Proto alias 使用 detail 类型
- Win 服务层回退到共享 Proto 类型

#### Interface Drafts
- `ActionDetail = "detail"`
- `ActionDetailResp = "detail_resp"`
- `DetailReq`
  - `req_id/origin_node/executor_node/flow_id/run_id/node_id/path`
- `DetailResp`
  - `req_id/code/msg/executor_node/flow_id/run_id/path/node/result`

#### Error Handling and Safety
- 不新增运行时错误路径
- 仅保证字段名和 JSON tag 稳定

#### Performance and Testing Strategy
- 以 `go test ./...` 和 `git diff --check` 验证

#### Extensibility Design Points
- 后续若 `detail` 扩展 vars 或整次 run dump，可继续在 Proto 增字段，无需回退 action 命名

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- 目标：把旧 dirty worktree 中已经验证过的 detail 协议变更迁移到干净分支
- 当前状态：
  - `main` 尚无 `detail` action/types
  - 旧 `proto-local-vars-observability` 已有未提交实现和验证结果

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- 稳定真相：
  - requirements/specs 归 `MyFlowHub-Server/docs`
- 本仓根部 `todo.md` 为 workflow 控制面例外
- `docs/protocol_map.md` 属于生成文档更新，不替代稳定 spec

#### Related Requirements / Specs / Lessons
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\server-local-vars-clean\docs\requirements\flow_data_dag.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\server-local-vars-clean\docs\specs\flow.md`
- Related lessons:
  - 无

#### Executable Task List
- `PROTO-DTL-1`：迁移 detail 协议字典
- `PROTO-VAL-1`：回归生成文档与测试

#### Task Details
##### PROTO-DTL-1 - detail protocol contract
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\proto-local-vars-clean`
- Plan Path: `D:\project\MyFlowHub3\worktrees\proto-local-vars-clean\todo.md`
- Goal: 迁移旧 worktree 的 `detail` action 和 payload 到 clean branch
- Files / Modules:
  - `protocol/flow/types.go`
  - `docs/protocol_map.md`
- Write Set:
  - `protocol/flow/types.go`
  - `docs/protocol_map.md`
- Acceptance:
  - `ActionDetail` / `ActionDetailResp` 存在
  - `DetailReq` / `DetailResp` 与 Server spec 对齐
- Test Points:
  - `go test ./...`
  - `git diff --check`
- Rollback:
  - 回退 Proto detail action/types 与协议映射更新

##### PROTO-VAL-1 - downstream compatibility smoke
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\proto-local-vars-clean`
- Plan Path: `D:\project\MyFlowHub3\worktrees\proto-local-vars-clean\todo.md`
- Goal: 为 SubProto clean worktree 提供可引用的 detail 契约
- Files / Modules:
  - 本任务不额外改文件，以联调验证为主
- Write Set:
  - 无计划内新增写集
- Acceptance:
  - SubProto clean worktree 可通过引用本仓临时 workspace 成功编译目标测试
- Test Points:
  - `go test github.com/yttydcs/myflowhub-subproto/flow/... -run 'TestFlowDetail|TestExecuteFlow|TestValidateGraph|TestFlowHandlersRejectInvalidFlowID|TestFlowRemoteForwardFailureReturnsResp'`
- Rollback:
  - 不需要单独回滚；依赖 `PROTO-DTL-1`

#### Dependencies
- 本仓先于 `subproto-local-vars-clean` 执行
- 稳定 requirements/specs 由 `server-local-vars-clean` 同步更新

#### Risks and Notes
- 当前旧实现主要存在于未提交 diff，不在 branch 历史内，迁移时必须逐文件核对
- 不得把旧 worktree 的无关历史一起带入 clean branch

#### Parallelism Assessment
- 不使用子Agent
- 原因：
  - 写集很小，且需要和 SubProto clean worktree 紧密联调

#### Issue List
- none

阻塞：否
进入 3.2

## Current State
- `PROTO-DTL-1` 已完成
- `PROTO-VAL-1` 已完成
- 已归档到 `docs/change/2026-04-02_proto-flow-detail.md`
