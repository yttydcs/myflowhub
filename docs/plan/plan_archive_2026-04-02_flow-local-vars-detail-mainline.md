# Plan - subproto-local-vars-clean

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-SubProto`
- Branch: `feat/local-vars-runtime-clean`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-local-vars-clean`
- Current Stage: `4 Change Archive`

## Stage Records

### Initialization
- `guide.md`: 已阅读，确认 `SubProto` 只放子协议实现，Server 只做装配
- base/worktree confirmation:
  - base=`main`
  - 本仓为主执行 worktree
  - 旧 `subproto-local-vars-runtime` 仅作参考，不直接提交

### Stage 1 - Requirements Analysis
#### Goal
- 把 `set_var`、`flow_var`、`detail` 从旧 dirty worktree 收口到 `main` 基线上的 clean branch

#### Scope
- 必须：
  - `set` 接受 `set_var`
  - 运行时支持 `RunContext.vars`
  - 绑定解析支持 `flow_var`
  - 新增 `detail/detail_resp` handler
  - 补齐关键测试
- 可选：
  - 若迁移中发现明显遗漏，可一并补最小兼容测试
- 不做：
  - 不扩展新的控制流节点
  - 不处理无关子协议
  - 不顺手修复本轮外既有 delete 权限基线问题

#### Use Cases
- `set_var` 写入 run-local JSON 变量供后续节点读取
- `flow_var` 读取唯一祖先写入者的变量值或子路径
- Win/调用方按 `flow_id/run_id/node_id/path` 查询节点结果详情

#### Functional Requirements
- `validateGraph` 必须允许 `set_var`
- `flow_var` 必须在 `set` 阶段校验唯一祖先写入者
- `executeNode` 必须支持 `set_var` 并同步写入节点结果和 runtime vars
- `detail` 必须与 `status` 分离
- `detail` 必须支持最近 run、指定 run、根结果和 JSON Pointer 子路径

#### Non-functional Requirements
- 保持改动最小，不扩散到无关模块
- 错误必须明确，不做静默降级
- `status` 继续保持轻量摘要

#### Inputs / Outputs
- 输入：
  - Proto clean worktree 的 detail 协议契约
  - Server clean worktree 的 flow requirements/specs
- 输出：
  - `flow/**` 中 local vars 与 detail 相关实现和测试

#### Edge Cases
- `flow_var` 无祖先写入者
- `flow_var` 有并行歧义写入者
- `detail` 缺少 `node_id`
- `detail.path` 非法
- 查询的 run/node/path 不存在

#### Acceptance Criteria
1. `flow set` 可接受 `set_var`
2. `flow_var` 对唯一祖先写入者通过，对缺失/歧义写入明确失败
3. `detail` 目标测试通过
4. 关键回归测试通过，且不引入 plan 外改动

#### Risks
- 旧 dirty worktree 改动量较大，迁移时容易遗漏测试或辅助类型
- `detail` 依赖 Proto 契约，若顺序错误会导致编译失败

### Stage 2 - Architecture Design
#### Overall Solution
- 沿用旧 dirty worktree 已验证的实现路径，在 `flow` 内部最小落地：
  - `runtime_bindings.go` 承载 vars / binding / 静态校验
  - `handler.go` 承载 `set_var` 执行与 `detail` handler
  - `types.go` / `actions.go` 只补 action alias

#### Alternatives Considered
- 备选：重新设计 `get_var` 节点
- 不采用原因：当前稳定 spec 明确通过 `source.kind=flow_var` 读取，不需要新增读取节点
- 备选：把 detail 并入 `status`
- 不采用原因：会把轻量轮询和重结果读取耦合

#### Module Responsibilities
- `flow/runtime_bindings.go`
  - `flow_var` 解析
  - `set_var` spec 解码、模板物化、祖先写入者静态校验
- `flow/handler.go`
  - `set_var` 执行
  - `detail` handler
  - graph 校验入口
- `flow/types.go` / `flow/actions.go`
  - 对接 Proto action/types
- `flow/*_test.go`
  - 锁定 local vars/detail 关键路径

#### Data / Call Flow
- `set`：
  - build graph index
  - 收集祖先 `set_var` writer
  - 校验 `flow_var`
- `run`：
  - 节点执行写入 `RunContext.nodes`
  - `set_var` 同步更新 `RunContext.vars`
- `detail`：
  - 命中 run
  - 命中 node
  - 可选按 path 读取结果

#### Interface Drafts
- `node.kind = "set_var"`
- `binding.source.kind = "flow_var"`
- `detail` 请求：
  - `req_id/origin_node/executor_node/flow_id/run_id/node_id/path`
- `detail_resp` 响应：
  - `req_id/code/msg/executor_node/flow_id/run_id/path/node/result`

#### Error Handling and Safety
- `flow_var` 缺失或歧义必须返回明确错误
- `detail` 非法 path 返回 `400`
- 节点结果 path 不存在返回 `404`
- `status` 不默认暴露完整结果和 vars

#### Performance and Testing Strategy
- 祖先写入者解析仅发生在 `set` 校验
- 运行时 vars 仅做 map 读写
- 重点验证：
  - `go test github.com/yttydcs/myflowhub-subproto/flow/... -run 'TestFlowDetail|TestExecuteFlow|TestValidateGraph|TestFlowHandlersRejectInvalidFlowID|TestFlowRemoteForwardFailureReturnsResp' -count=1 -p 1`
  - 视情况补 `go test github.com/yttydcs/myflowhub-subproto/flow/... -count=1 -p 1`

#### Extensibility Design Points
- `RunContext.vars` 为后续 vars 调试视图保留扩展位
- `detail` 后续可增整次 run dump 或 vars 视图

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- 目标：把旧 `subproto-local-vars-runtime` 的未提交 local vars/detail 实现迁移到 clean branch
- 当前状态：
  - `main` 仍只接受 `call/compose`
  - 旧 dirty worktree 已有实现和测试，但不在正式提交历史中

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- 稳定真相：
  - requirements/specs 归 `MyFlowHub-Server/docs`
- 本仓根部 `todo.md` 为 workflow 控制面例外
- `docs/change` 在 stage 4 再归档，不提前替代 requirements/specs

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
- `SUB-LV-1`：迁移 local vars runtime 和 graph 校验
- `SUB-RD-1`：迁移 detail handler 和协议接线
- `SUB-VAL-1`：执行目标测试与回归评估

#### Task Details
##### SUB-LV-1 - local vars runtime and validation
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-local-vars-clean`
- Plan Path: `D:\project\MyFlowHub3\worktrees\subproto-local-vars-clean\todo.md`
- Goal: 迁移 `RunContext.vars`、`set_var`、`flow_var` 及其静态校验
- Files / Modules:
  - `flow/handler.go`
  - `flow/runtime_bindings.go`
  - `flow/data_dag_test.go`
  - `flow/graph_test.go`
- Write Set:
  - `flow/handler.go`
  - `flow/runtime_bindings.go`
  - `flow/data_dag_test.go`
  - `flow/graph_test.go`
- Acceptance:
  - `set_var` 可写入 run-local vars
  - `flow_var` 可唯一解析祖先写入者
  - 缺失/歧义写入明确失败
- Test Points:
  - `TestExecuteFlow_*FlowVar*`
  - `TestValidateGraphAllowsFlowVarWithUniqueAncestorWriter`
  - `TestValidateGraphRejectsFlowVarWithoutAncestorWriter`
  - `TestValidateGraphRejectsAmbiguousFlowVarWriter`
- Rollback:
  - 回退 local vars 相关实现与测试

##### SUB-RD-1 - run detail handler
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-local-vars-clean`
- Plan Path: `D:\project\MyFlowHub3\worktrees\subproto-local-vars-clean\todo.md`
- Goal: 迁移 `detail/detail_resp` handler、types alias 与目标测试
- Files / Modules:
  - `flow/actions.go`
  - `flow/types.go`
  - `flow/handler.go`
  - `flow/detail_test.go`
  - `flow/flow_id_test.go`
  - `flow/runtime_fix_test.go`
- Write Set:
  - `flow/actions.go`
  - `flow/types.go`
  - `flow/handler.go`
  - `flow/detail_test.go`
  - `flow/flow_id_test.go`
  - `flow/runtime_fix_test.go`
- Acceptance:
  - `detail` 可查询最近 run 或指定 run 的节点结果
  - `detail` 与 `status` 解耦
  - 非法输入明确返回 `400/404`
- Test Points:
  - `TestFlowDetailReturnsLatestRunNodeResult`
  - `TestFlowDetailReturnsResultPath`
  - `TestFlowDetailNotFound`
  - `TestFlowDetailRejectsInvalidPath`
- Rollback:
  - 回退 detail handler/types/tests

##### SUB-VAL-1 - regression verification
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\subproto-local-vars-clean`
- Plan Path: `D:\project\MyFlowHub3\worktrees\subproto-local-vars-clean\todo.md`
- Goal: 在 clean branch 上复跑旧 worktree 已通过的目标验证
- Files / Modules:
  - 无新增计划内写集，以测试和结果记录为主
- Write Set:
  - 无计划内新增写集
- Acceptance:
  - 目标测试通过
  - 若全量测试仍失败，能明确区分既有基线失败和新增回归
- Test Points:
  - 目标测试命令
  - 视情况补全量 `go test`
- Rollback:
  - 不需要单独回滚；依赖 `SUB-LV-1` / `SUB-RD-1`

#### Dependencies
- 依赖 `proto-local-vars-clean` 先提供 detail action/types
- 与 `server-local-vars-clean` 并行更新稳定 docs，但 code 以其 requirements/specs 为准

#### Risks and Notes
- 全量 `go test` 可能继续暴露本轮外 delete 权限基线失败，需明确隔离
- 迁移时不得把旧 worktree 中无关变更和 CRLF 杂讯一起带入

#### Parallelism Assessment
- 不使用子Agent
- 原因：
  - `handler.go` 与 `runtime_bindings.go` 写集高度耦合
  - 当前任务关键路径依赖主代理统一迁移和回归验证

#### Issue List
- none

阻塞：否
进入 3.2

## Current State
- `SUB-LV-1` 已完成
- `SUB-RD-1` 已完成
- `SUB-VAL-1` 已完成
- 已归档到 `docs/change/2026-04-02_flow-local-vars-detail-mainline.md`
