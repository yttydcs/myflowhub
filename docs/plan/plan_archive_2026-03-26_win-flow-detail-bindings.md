# Plan - win-flow-detail-bindings

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `fix/wails-detail-bindings`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-wails-detail-bindings`
- Current Stage: `4 - Change Archive completed / waiting workflow end confirmation`

## Stage Records

### Initialization
- guide.md:
  - 已读。确认所有 worktree 必须创建在 `D:\project\MyFlowHub3\worktrees\`，提交信息使用中文。
- base/worktree confirmation:
  - 参与仓库：`D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - 参与模块：Win app module `github.com/yttydcs/myflowhub-win`
  - Base branch：`main`
  - 执行分支 / worktree：`fix/wails-detail-bindings` / `D:\project\MyFlowHub3\worktrees\fix-wails-detail-bindings`
  - 主仓路径仅作为 control-plane 使用，不在主仓直接实现。
  - worktree 校验：
    - `wails generate module` 在 worktree 下若不设 `GOWORK=off`，会先命中父级 `go.work` 的模块外错误。
    - 本 workflow 的编译/绑定验证统一使用 `GOWORK=off`。

### Stage 1 - Requirements Analysis
#### Goal
- 修复 Win 仓库在 `GOWORK=off` 下执行 `wails generate module` / `go test ./internal/services/flow` 时的编译失败，使 Wails bindings 能重新生成。

#### Scope
- 必须：
  - 消除 `internal/services/flow/service.go` 中对不存在的 `flow.DetailReq` / `flow.DetailResp` / `flow.ActionDetail*` 的编译依赖。
  - 保持现有 Win `DetailSimple` 入口和前端 JSON 字段契约不变。
  - 为关键 detail 入口补齐最小单元测试。
  - 记录 requirements/specs 影响，并在完成后归档到 `docs/change/`。
- 可选：
  - 如果稳定 spec 明确写死了共享 proto 类型名，则做最小澄清，说明 Win 侧可使用本地 typed payload 以解耦 proto 漂移。
- 不做：
  - 不扩展或实现 Server `flow.detail` 运行时能力。
  - 不升级 `myflowhub-proto` / `myflowhub-sdk` / `myflowhub-core` 版本。
  - 不改动前端 detail 面板行为。

#### Use Cases
- 开发者在 Win worktree 中执行 `$env:GOWORK='off'; wails generate module` 时，命令不再因 `DetailReq/Resp` 缺失而失败。
- 开发者在 Win worktree 中执行 `$env:GOWORK='off'; go test ./internal/services/flow` 时，包可编译并通过新增测试。

#### Functional Requirements
- `FlowService.Detail` / `DetailSimple` 必须继续暴露给 Wails。
- detail 请求必须继续支持：
  - `req_id`
  - `origin_node`
  - `executor_node`
  - `flow_id`
  - `run_id`
  - `node_id`
  - `path`
- detail 响应必须继续支持：
  - `code`
  - `msg`
  - `executor_node`
  - `flow_id`
  - `run_id`
  - `path`
  - `node`
  - `result`
- 输入校验必须显式拒绝空 `req_id` / `flow_id` / `node_id`。

#### Non-functional Requirements
- 变更面保持最小，尽量限制在 `internal/services/flow` 和必要文档。
- 不引入额外 I/O、额外依赖或新的跨模块耦合。
- 继续兼容 `GOWORK=off` 的 Wails bindings 生成路径。

#### Inputs / Outputs
- 输入：
  - Wails / Go 调用 `FlowService.Detail*`
  - 验证命令：`$env:GOWORK='off'; go test ./internal/services/flow`
  - 验证命令：`$env:GOWORK='off'; wails generate module`
- 输出：
  - Go 包成功编译并通过最小测试
  - Wails bindings 生成成功

#### Edge Cases
- 在 `GOWORK` 未关闭时，worktree 仍会先命中父级 `go.work` 的模块外错误。
- 当前仓库可用的 shared proto / server code 中不存在 `flow.detail` 定义。
- detail 运行时能力可能仍依赖未合入的 server/proto 契约，本次仅修复 Win 端编译与 bindings 生成。

#### Acceptance Criteria
- `$env:GOWORK='off'; go test ./internal/services/flow` 通过。
- `$env:GOWORK='off'; wails generate module` 通过。
- `DetailSimple` 的前端调用字段不需要跟随本次修复改动。

#### Risks
- 若后续 shared proto 正式补充 `flow.detail`，Win 本地类型需要再决定是否收敛回共享类型。
- 若当前 server 实际不支持 `detail`，运行时仍可能返回协议级失败；该风险不在本轮修复范围。

#### Issue List
- 无阻塞项。`GOWORK=off` 路径与缺失符号问题均已复现并确认。

### Stage 2 - Architecture Design
#### Overall Solution
- 采用 Win 本地 typed detail request/response 与 action 常量，保持 JSON wire 字段和 `FlowService.Detail*` 方法语义不变，避免对当前缺失 `flow.detail` 定义的 shared proto 产生编译依赖。

#### Alternatives Considered
- 方案 A：升级 `myflowhub-proto` 到包含 `flow.detail` 的版本。
  - 不采用：当前本地 proto 仓库与依赖均不存在该定义，无法在本 workflow 内安全落地。
- 方案 B：直接删除 `Detail` / `DetailSimple`。
  - 不采用：会破坏已合入的前端入口与稳定 requirements。
- 方案 C：Win 本地新增 typed payload，并在 spec 中最小澄清实现边界。
  - 采用：可最小化恢复编译与 bindings，同时不扩大依赖升级范围。

#### Module Responsibilities
- `internal/services/flow`
  - 持有 Win 侧 detail typed payload、action 常量、输入校验、await 解包与错误包装。
- `docs/specs/flow-editor-run-detail.md`
  - 若需要，澄清 Win `FlowService` detail binding 可使用本地 typed payload，只要 JSON 契约保持一致。
- `docs/change/*`
  - 记录本次编译/绑定修复与验证结果。

#### Data / Call Flow
- 前端 / Wails -> `FlowService.DetailSimple(...)`
- Go service：
  - 校验 `req_id` / `flow_id` / `node_id`
  - `transport.EncodeMessage("detail", req)`
  - `session.SendCommandAndAwait(..., "detail_resp")`
  - `json.Unmarshal(..., DetailResp)`
  - `extractCodeMsg(...)` 根据 `code/msg` 返回 resp 或 error

#### Interface Drafts
- `type DetailReq struct { req_id, origin_node, executor_node, flow_id, run_id, node_id, path }`
- `type DetailResp struct { req_id, code, msg, executor_node, flow_id, run_id, path, node, result }`
- `const actionDetail = "detail"`
- `const actionDetailResp = "detail_resp"`

#### Error Handling and Safety
- 保持已有错误包装链路：
  - 输入缺失直接返回显式 error
  - await 失败走 `toUIError`
  - `code != 1` 继续按 `msg/code` 归一化
- 不吞掉 session 未初始化等错误。

#### Performance and Testing Strategy
- 无新增 I/O 或额外请求。
- 新增最小单元测试覆盖：
  - detail 输入校验
  - 校验通过后在空 session 下返回初始化错误
  - `extractCodeMsg` 可读取本地 `DetailResp`
- 验证命令：
  - `$env:GOWORK='off'; go test ./internal/services/flow`
  - `$env:GOWORK='off'; wails generate module`

#### Extensibility Design Points
- 本地 typed payload 隔离 proto 漂移；未来 shared proto 补齐后，可单点收敛回共享类型。
- detail action 常量独立后，未来若 server 契约扩展字段，只需在本地 struct 增量兼容。

#### Issue List
- 无阻塞项；运行时 server 支持度作为已知风险记录，不阻塞本轮编译修复。

### Stage 3.1 - Planning
#### Project Goal and Current State
- 目标：恢复 Win worktree 中 `GOWORK=off` 的 flow detail 编译与 Wails bindings 生成。
- 当前状态：
  - worktree 已创建并进入执行路径。
  - `GOWORK=off` 下已复现真实编译错误：`flow.DetailReq` / `flow.DetailResp` / `flow.ActionDetail*` 未定义。
  - shared proto / server 当前代码树中均未找到 `flow.detail` 定义。

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验后确认：
  - stable truth：
    - `docs/requirements/flow-editor-run-detail.md`
    - `docs/specs/flow-editor-run-detail.md`
  - workflow control：
    - 本 worktree 根 `plan.md`
  - completed results：
    - `docs/change/2026-03-26_win-flow-detail-bindings.md`
  - reusable troubleshooting：
    - 先在 stage 4 判断是否需要新 `docs/lessons`；当前先记录为待判定
- Requirements impact: `none`
- Specs impact: `clarify`
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\fix-wails-detail-bindings\docs\requirements\flow-editor-run-detail.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\fix-wails-detail-bindings\docs\specs\flow-editor-run-detail.md`
- Related lessons:
  - none

#### Executable Task List
- [x] `DOC-1` 澄清 flow run detail spec 的 Win typed payload 边界（如实现需要）
- [x] `FIX-1` 在 `internal/services/flow` 中补齐本地 detail req/resp 与 action 常量，并接回 `Detail*`
- [x] `TEST-1` 增加最小单元测试并完成 `GOWORK=off` 验证

#### Task Details
##### DOC-1 - Clarify Win Detail Spec
- Owner: main
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-wails-detail-bindings`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-wails-detail-bindings\plan.md`
- Goal: 若代码实现改为 Win 本地 typed payload，则同步澄清 spec，避免稳定文档继续暗示不存在的 shared proto 类型。
- Files / Modules:
  - `docs/specs/flow-editor-run-detail.md`
- Write Set:
  - `docs/specs/flow-editor-run-detail.md`
- Acceptance:
  - spec 保留同一 JSON 字段契约，但不再强依赖当前仓库不存在的 shared proto detail 类型。
- Test Points:
  - 人工复核 spec 与代码接口一致。
- Rollback:
  - 回退 spec 澄清段落。

##### FIX-1 - Restore Win Flow Detail Bindings
- Owner: main
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-wails-detail-bindings`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-wails-detail-bindings\plan.md`
- Goal: 通过本地 typed payload 恢复 `FlowService.Detail` / `DetailSimple` 编译与 bindings 生成。
- Files / Modules:
  - `internal/services/flow/service.go`
  - `internal/services/flow/detail_types.go`
- Write Set:
  - `internal/services/flow/service.go`
  - `internal/services/flow/detail_types.go`
- Acceptance:
  - `Detail*` 方法恢复可编译。
  - `extractCodeMsg(...)` 可正确处理本地 `DetailResp`。
  - 现有前端请求/响应 JSON 字段不变。
- Test Points:
  - `$env:GOWORK='off'; go test ./internal/services/flow`
  - `$env:GOWORK='off'; wails generate module`
- Rollback:
  - 回退本地 detail 类型与 service 引用。

##### TEST-1 - Add Flow Detail Guard Tests
- Owner: main
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-wails-detail-bindings`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-wails-detail-bindings\plan.md`
- Goal: 为 detail 新增最小回归测试，确保后续不会再次因类型/分支遗漏而静默回归。
- Files / Modules:
  - `internal/services/flow/service_test.go`
- Write Set:
  - `internal/services/flow/service_test.go`
- Acceptance:
  - 覆盖 detail 参数校验、空 session 错误，以及 `extractCodeMsg` 的 `DetailResp` 分支。
- Test Points:
  - `$env:GOWORK='off'; go test ./internal/services/flow`
- Rollback:
  - 回退测试文件。

#### Dependencies
- `wails generate module` 需要在 worktree 中显式 `GOWORK=off`。
- 当前 fix 不依赖 server / proto 升级。

#### Risks and Notes
- stable requirement 仍假设 server 已有 `flow.detail`；当前代码树无法确认。该风险在归档中继续记录。
- 若 `wails generate module` 成功后出现新的非 detail 编译问题，需要回到 `3.1` 更新计划，而不是顺手扩 scope。

#### Parallelism Assessment
- 结论：本轮不派发子 Agent。
- 原因：
  - `FIX-1`、`TEST-1`、`DOC-1` 写集高度相关，围绕同一 detail 契约，存在显著耦合。
  - 任务规模小，主路径阻塞在单模块编译修复，拆分不会提升吞吐，反而增加集成成本。
  - 无独立且非关键路径的验证任务需要并行。

#### Issue List
- 无阻塞项。

### Stage 3.2 - Implementation Result
- `FIX-1`
  - 新增 `internal/services/flow/detail_types.go`
  - 在 Win 侧补齐本地 `DetailReq` / `DetailResp` 与 `detail` / `detail_resp` action 常量
  - `internal/services/flow/service.go` 改为使用本地 detail 类型，并让 `extractCodeMsg(...)` 支持本地 `DetailResp`
- `TEST-1`
  - 新增 `internal/services/flow/service_test.go`
  - 覆盖 detail 输入校验、空 session 错误、`extractCodeMsg` detail 分支
- `DOC-1`
  - `docs/specs/flow-editor-run-detail.md` 澄清 Win 可使用本地 typed payload，只要 JSON 契约保持一致
- 验证结果：
  - `$env:GOWORK='off'; go test ./internal/services/flow`：通过
  - `$env:GOWORK='off'; wails generate module`：通过
  - `$env:GOWORK='off'; go test ./...`：通过
  - `git diff --check`：通过

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - 编译失败点仅来自缺失的 detail DTO / action；本轮已恢复 Wails binding 与 Go 编译路径，未改变前端 JSON 契约。
- 架构合理性：通过
  - 采用 Win 本地 typed payload，和现有 authority 本地 typed payload 模式一致，不扩大 shared proto 升级范围。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 仅新增本地 struct / 常量与最小测试，无新增运行时 I/O 或额外请求。
- 可读性与一致性：通过
  - detail 类型单独放入 `detail_types.go`，边界明确；命名与已有 service 风格一致。
- 可扩展性与配置化：通过
  - 后续若 shared proto 补齐 `flow.detail`，可单点收敛；当前不引入硬编码环境值。
- 稳定性与安全：通过
  - 保留原有输入校验和错误包装，不吞错误。
- 测试覆盖情况：通过
  - 关键 detail guard 新增单元测试；`GOWORK=off` 全量 `go test ./...` 与 `wails generate module` 均已通过。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未派发子 Agent。

### Stage 4 - Archive Result
- 使用 `$m-docs` 完成归档路由检查。
- 归档输出：
  - `docs/change/2026-03-26_win-flow-detail-bindings.md`
  - `docs/lessons/wails-binding-proto-drift.md`
- 索引更新：
  - `docs/change/README.md`
  - `docs/lessons/README.md`
- Requirements impact：`none`
- Specs impact：`updated`
- Lessons impact：`updated`

阻塞：否
等待用户确认是否结束 workflow
