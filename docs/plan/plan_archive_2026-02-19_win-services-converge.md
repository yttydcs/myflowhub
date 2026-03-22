# Plan - Win：Auth/Flow/TopicBus/VarPool services 收敛（返回值 + 业务事件）

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`refactor/win-services-converge`
- Worktree：`d:\project\MyFlowHub3\worktrees\win-services-converge\MyFlowHub-Win`
- Base：`main`
- 参考：
  - `d:\project\MyFlowHub3\target.md`
  - `d:\project\MyFlowHub3\repos.md`（重点：L171 “Win services 收敛”）
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文）

## 约束（边界）
- 本 workflow 一次性收敛 4 个模块：`auth` / `flow` / `topicbus` / `varpool`（允许一个 PR 覆盖多个模块以保持一致性）。
- wire 不改：SubProto / Action / JSON schema / Header 语义不变。
- `session.frame` 事件保留用于 Debug/原始帧观测；但这 4 个模块的业务 UI **不再解析** `session.frame`（不做 payload decode/JSON.parse/switch(action)）。
- `topicbus` 与 `varpool` 的通知/事件流：改为 **Go 侧解析帧 → 发布业务事件 → 前端订阅业务事件**（你已确认）。
- Linux 构建验收暂忽略；以 Windows 为准。
- 验收建议使用 `GOWORK=off`（确保单仓 clone 可用）。

## 当前状态（事实，可审计）
- 前端仍存在 “解析 session.frame” 的模块：
  - `frontend/src/stores/session.ts`：解析 auth login/register *_resp 更新本地 auth snapshot。
  - `frontend/src/stores/flow.ts`：解析 flow `*_resp` 更新 flows/nodes/status。
  - `frontend/src/stores/topicbus.ts`：解析 topicbus `publish` 帧作为事件流。
  - `frontend/src/stores/varpool.ts`：解析 varstore `*_resp/assist_*/notify_*` + `var_changed/var_deleted` 更新 UI。
- Go 侧 service 已大部分升级为 send+await，但多为“只返回 error”，导致 UI 必须靠 frame 反推业务结果：
  - `internal/services/auth/service.go`
  - `internal/services/flow/service.go`
  - `internal/services/topicbus/service.go`（publish 仍是 send-only）
  - `internal/services/varpool/service.go`
- App 事件桥接现状：
  - `app.go` 仅桥接 `session.*` 与 `file.*`、`presets.*` 等事件，尚无 `topicbus.*`、`varpool.*` 的业务事件桥接。

---

## 1) 需求分析

### 目标
1) 彻底移除上述 4 个模块在前端对 `session.frame` 的业务解析依赖，实现“UI 不扩散协议细节”的目标态。
2) Go services 收敛为“业务编排层”：
   - req/resp 类动作：Wails API 直接返回对应 Proto 响应结构体（resp + error）。
   - notify/事件流：Go 侧解析帧后发布“业务事件”，前端只处理业务事件，不处理 wire。
3) 统一 UI 友好错误与日志：
   - UI 能得到清晰错误（超时/未连接/业务 code!=1 等）；
   - logs 中能定位 action 与失败原因（必要时包含关键字段，但避免泄露敏感值）。

### 范围（必须 / 可选 / 不做）
#### 必须（本 PR）
- Go：调整 Wails bindings 返回值（resp + error）并补齐必要业务事件：
  - `internal/services/auth`：`Register*` / `Login*` 返回 `auth.RespData`。
  - `internal/services/flow`：`List/Get/Set/Run/Status*` 返回对应 `flow.*Resp`。
  - `internal/services/topicbus`：
    - `Subscribe/Unsubscribe/ListSubs*` 返回对应 resp；
    - `publish` 事件流：Go 侧解析并发布业务事件（例如 `topicbus.event`）。
  - `internal/services/varpool`：`List/Get/Set/Revoke/Subscribe/Unsubscribe*` 返回 `varstore.VarResp`；
    - Go 侧解析并发布业务事件（`var_changed/var_deleted/notify_set/notify_revoke` → 业务事件）。
- 前端：4 个模块的 store 仅使用“返回值/业务事件”更新 UI：
  - `frontend/src/stores/session.ts`：移除 auth payload 解析（保留连接态/错误/最后帧时间展示）。
  - `frontend/src/stores/flow.ts`：移除 frame 解析，改用返回值更新 flows/status。
  - `frontend/src/stores/topicbus.ts`：移除 frame 解析，改订阅业务事件并维持现有节流/展示。
  - `frontend/src/stores/varpool.ts`：移除 frame 解析，改用返回值 + 业务事件更新 keys/data/sub 状态。
- `app.go`：桥接新增业务事件（`topicbus.*`、`varpool.*`）到 Wails 前端。

#### 可选（本 PR，若不扩大范围则不做）
- 对 varpool 的 “list 后并发 get” 增加轻量并发上限（避免极端 key 数造成瞬时并发过高）。

#### 不做（本 PR）
- 不改 Server/Core/SDK/Proto（除非出现阻塞性 bug）。
- 不改 wire（协议字段、action、SubProto 等完全不动）。
- 不做 “minimal/full 变体产品化”（已 Deferred）。

### 使用场景
- Home：
  - Connect / Disconnect
  - Register / Login（成功后写回 nodeId/hubId/role）
- Flow：
  - List → Get → Edit → Set → Run/Status
- TopicBus：
  - Subscribe/Unsubscribe/Resubscribe
  - 接收 publish 事件并展示（高频下 UI 仍稳定）
- VarPool：
  - listMine 获取自有 keys
  - get/set/revoke
  - subscribe/unsubscribe + 实时 var_changed/var_deleted/notify_* 更新

### 功能需求
- UI 不解析 wire：不再出现 `decodePayloadText/JSON.parse/switch(action)` 这类“帧解码分支”。
- 关键数据流保持一致：
  - Auth 登录/注册成功后：Home 能保存并反显 nodeId/hubId/role。
  - Flow 操作：列表/详情/状态能正确刷新。
  - TopicBus publish：事件内容与当前展示一致（topic/name/ts + payload 展示）。
  - VarPool：watch list、value 更新、删除/撤销逻辑保持一致。

### 非功能需求
- 性能：
  - 减少前端对每帧的 decode/parse 分支；高频事件（topicbus publish）不引发卡顿。
- 可维护性：
  - 业务解析集中在 Go service（或 SDK/transport decode），前端只处理业务 state。
- 可回滚：
  - 单 PR 内按模块拆成多个提交，便于逐步回退。

### 输入输出
- Wails bindings：
  - 输入：`sourceID/targetID`、以及各协议请求参数（req_id/flow_id/topic/name/key/value 等）。
  - 输出：
    - 成功：返回对应 `*Resp`（`code==1`）
    - 失败：返回 `error`（msg 优先；必要时归一化超时/未连接）
- 业务事件（Wails events）：
  - `topicbus.event`：publish 事件（按 UI 展示需要的字段）。
  - `varpool.changed` / `varpool.deleted`：变量变更/删除（避免暴露 action 字符串给 UI）。

### 边界异常
- 未连接：返回明确错误；UI 提示“未连接/请先连接”。
- 超时：默认 `8s`（沿用既有约定），UI 显示“请求超时(8s)”或等价文案。
- 断线中途收到通知：Go 侧解析失败不崩溃；必要时丢弃并写 warn 日志。
- 并发请求/切换目标节点：避免旧结果覆盖新状态（按模块做必要竞态保护）。

### 验收标准
- 代码级：
  - `frontend/src/stores/session.ts|flow.ts|topicbus.ts|varpool.ts` 不再解析 `session.frame` payload（不再出现 `decodePayloadText/JSON.parse/switch(action)` 的帧解析路径）。
- 命令级：
  - `GOWORK=off go test ./... -count=1 -p 1`
  - `GOWORK=off wails generate module`
  - `cd frontend && npm ci && npm run build`
- 冒烟（手工）：
  - Win 启动 → Connect → Register/Login → Flow/TopicBus/VarPool 页面关键操作可用，TopicBus/VarPool 实时更新可见。

### 风险
- Wails API 签名变化（从 `error` 变为 `resp, error`）会影响前端调用方式；需要同步修改 UI。
- notify/事件流的“业务事件”设计需要稳定命名与结构，避免后续再破坏性变更。
- 高并发事件（topicbus publish）可能导致 event 量过大；需要保持 UI 侧节流（现有实现可复用）。

## 问题清单（阻塞：否）
- 你已确认：一次性收敛 4 模块，并且 `topicbus/varpool` 的通知类消息改为 Go 侧解析后发业务事件，前端不再解析 `session.frame`。

---

## 2) 架构设计（分析）

### 总体方案（采用）
- **Req/Resp（Auth/Flow/VarPool/TopicBus 的订阅类）**：
  - Go service：`SendCommandAndAwait` → `json.Unmarshal(resp.Message.Data)` → `code/msg` 判定 → 返回 `Resp, error`。
  - 前端 store：调用 binding 后用返回值更新 state（必要时做“过期结果丢弃”）。
- **Notify / 事件流（TopicBus publish、VarPool changed/deleted）**：
  - Go service：订阅 `session.frame`（仅过滤目标 SubProto），使用 `myflowhub-sdk/transport.DecodeMessage` 解析 `{action,data}`，再按业务含义发布为 **业务事件**：
    - `topicbus.event`
    - `varpool.changed`
    - `varpool.deleted`
  - App：将上述业务事件桥接到 Wails runtime（`runtime.EventsEmit`）。
  - 前端 store：只订阅业务事件，不订阅/不解析 `session.frame`。

### 备选对比
- 备选 A：仍由前端解析 `session.frame`（不采用）
  - 问题：协议细节在 UI 扩散、重复解析、难统一错误与可观测。
- 备选 B：Go 侧直接把 `action` 原样透传到前端（不采用）
  - 问题：前端仍需要 switch(action)，等同于把 wire 细节换个通道扩散。

### 模块职责
- `internal/services/session`
  - 连接 + send/await；发布原始 `session.frame/state/error`（Debug/观测通道）。
- `internal/services/auth` / `flow` / `varpool` / `topicbus`
  - 业务编排（输入校验、await、解包、错误归一化、日志）
  - notify/事件流：解包并发布业务事件（topicbus/varpool）
- `app.go`
  - 统一桥接 bus 事件到 Wails 前端（新增业务事件名）
- `frontend/src/stores/*`
  - UI state 管理（不做 wire 解码）

### 数据 / 调用流
#### Req/Resp
1) UI 调用 binding（例如 `FlowService.ListSimple`）
2) Go service 发送并 await
3) Go service 解包 resp（Proto struct），成功返回 resp；失败返回 error（UI 友好）
4) UI 用 resp 更新 state

#### Notify / 事件流
1) TCP 收到帧 → `SessionService.handleFrame` 发布 `session.frame`（原始事件）
2) `TopicBusService/VarPoolService` 订阅并解析属于本 SubProto 的帧
3) service 发布业务事件（`topicbus.event` / `varpool.changed` / `varpool.deleted`）
4) `app.go` 将业务事件桥接到 Wails 前端
5) UI store 订阅业务事件并更新 state

### 接口草案（Go / Wails）
- Auth：
  - `RegisterSimple(sourceID, targetID uint32, deviceID string) (auth.RespData, error)`
  - `LoginSimple(sourceID, targetID uint32, deviceID string, nodeID uint32) (auth.RespData, error)`
- Flow：
  - `ListSimple(...) (flow.ListResp, error)`
  - `GetSimple(...) (flow.GetResp, error)`
  - `SetSimple(...) (flow.SetResp, error)`
  - `RunSimple(...) (flow.RunResp, error)`
  - `StatusSimple(...) (flow.StatusResp, error)`
- TopicBus：
  - `SubscribeSimple(...) (topicbus.Resp, error)`
  - `UnsubscribeSimple(...) (topicbus.Resp, error)`
  - `ListSubsSimple(...) (topicbus.ListResp, error)`
  - `PublishSimple(...) error`（保持 send-only，不等待 publish_resp）
- VarPool：
  - `ListSimple(...) (varstore.VarResp, error)`
  - `GetSimple(...) (varstore.VarResp, error)`
  - `SetSimple(...) (varstore.VarResp, error)`
  - `RevokeSimple(...) (varstore.VarResp, error)`
  - `SubscribeSimple(...) (varstore.VarResp, error)`
  - `UnsubscribeSimple(...) (varstore.VarResp, error)`

### 业务事件草案
- `topicbus.event`：
  - `{ topic, name, ts, dataRaw }`
- `varpool.changed`：
  - `{ name, owner, value, visibility, type }`
- `varpool.deleted`：
  - `{ name, owner }`

### 错误与安全
- 输入校验：空 device_id/topic/name/key 等直接返回 error。
- `code!=1`：返回 `msg` 优先的 error；同时写 warn 日志（含 action + 关键字段）。
- topicbus/varpool 事件 payload 可能包含敏感信息：
  - logs 侧避免主动打印 value/payload；
  - UI 展示保持既有行为（这是 DebugClient 的功能属性），但不在 logs 重复扩散。

### 性能与测试策略
- 性能：
  - 去除前端对每帧的 decode/parse/switch；TopicBus 事件仍保持 UI 侧节流（现有 `pendingEvents + flushTimer` 机制）。
- 测试：
  - Go：`GOWORK=off go test ./... -count=1 -p 1`
  - Wails types：`GOWORK=off wails generate module`
  - Frontend：`npm ci && npm run build`
  - 手工冒烟：见 WINALL7

### 可扩展性设计点
- 本 PR 形成模板后，其它子协议迁移时遵循：
  - req/resp → 返回值式 API；
  - notify/stream → Go 侧业务事件；
  - `session.frame` 仅保留为 Debug/观测通道。

---

## 3.1) 计划拆分（Checklist）

### WINALL0 - 计划归档
- 目标：保留上一轮 Management workflow 的 `plan.md`，避免覆盖丢失。
- 涉及文件：
  - `plan_archive_2026-02-19_win-mgmt-service.md`
- 验收条件：归档文件可独立阅读复现上一轮结论。
- 回滚点：删除该归档文件。

### WINALL1 - Auth：返回值化 + UI 改为使用返回值
- 目标：Home 的 register/login 不再依赖 frame 解析更新 auth state。
- 涉及文件（预期）：
  - `internal/services/auth/service.go`
  - `frontend/src/pages/Home.vue`
  - `frontend/src/stores/session.ts`
- 验收条件：
  - `RegisterSimple/LoginSimple` 返回 resp；
  - `session.ts` 不再解析 auth payload；
  - Home 登录/注册成功后能写回 nodeId/hubId/role 并持久化。
- 测试点：
  - `GOWORK=off wails generate module`
  - 手工冒烟：Connect → Register/Login → Home 反显正确。
- 回滚点：revert 提交。

### WINALL2 - Flow：返回值化 + store 去 frame 解析
- 目标：Flow 页面 list/get/set/run/status 全部用返回值更新 UI。
- 涉及文件（预期）：
  - `internal/services/flow/service.go`
  - `frontend/src/stores/flow.ts`
- 验收条件：
  - `frontend/src/stores/flow.ts` 不再 `EventsOn("session.frame")`；
  - 关键操作 UI 状态正确（列表/详情/状态）。
- 测试点：
  - `GOWORK=off go test ./...`
  - 手工冒烟：Flow list/get/set/run/status。
- 回滚点：revert 提交。

### WINALL3 - TopicBus：publish 事件下沉到 Go 业务事件 + store 去 frame 解析
- 目标：TopicBus 事件流不再由前端解析 wire，而由 Go 解析并发布业务事件。
- 涉及文件（预期）：
  - `internal/services/topicbus/*`（新增 events、订阅 session.frame、发布 `topicbus.event`）
  - `app.go`（桥接 `topicbus.event`）
  - `frontend/src/stores/topicbus.ts`（改为订阅业务事件）
- 验收条件：
  - store 不再解析 payload；
  - publish 事件展示与现状一致，且高频下不崩溃（保持节流）。
- 测试点：
  - 手工冒烟：subscribe → publish → 事件列表刷新。
- 回滚点：revert 提交。

### WINALL4 - VarPool：返回值化 + 通知下沉为业务事件 + store 去 frame 解析
- 目标：VarPool 的 req/resp 使用返回值更新；通知类用业务事件更新；前端不解析 wire。
- 涉及文件（预期）：
  - `internal/services/varpool/*`（返回值化；订阅并发布 `varpool.changed/deleted`）
  - `app.go`（桥接 `varpool.*`）
  - `frontend/src/stores/varpool.ts`
  - `frontend/src/pages/VarPool.vue`（如需适配）
- 验收条件：
  - store 不再 `EventsOn("session.frame")`；
  - watch list / get/set/revoke / subscribe/unsubscribe 语义保持一致；
  - `var_changed/var_deleted/notify_*` 仍能实时更新 UI。
- 测试点：
  - 手工冒烟：listMine/get/set/revoke；模拟/触发变更通知。
- 回滚点：revert 提交。

### WINALL5 - App：桥接业务事件 + 生命周期清理
- 目标：新增业务事件能被前端订阅；服务侧订阅不泄漏（如需 Close）。
- 涉及文件（预期）：
  - `app.go`
  - `internal/services/topicbus/*`
  - `internal/services/varpool/*`
- 验收条件：启动/关闭无异常；事件能正常到前端。

### WINALL6 - 回归验证（命令级）
- Go：`GOWORK=off go test ./... -count=1 -p 1`
- Wails：`GOWORK=off wails generate module`
- Frontend：`cd frontend && npm ci && npm run build`

### WINALL7 - 冒烟验证（手工）
1) 启动 `hub_server`（确保 auth/flow/topicbus/varstore handler 可用）
2) 启动 Win：`wails dev`
3) Home：
   - Connect → Register/Login 成功（nodeId/hubId/role 持久化）
4) Flow：list/get/set/run/status 跑通
5) TopicBus：
   - subscribe 后 publish，事件列表能滚动更新
6) VarPool：
   - listMine 后能拉取 values；set/revoke 正常；
   - 触发 var_changed/var_deleted/notify_* 时 UI 实时更新

### WINALL8 - Code Review（阶段 3.3）
- 按 checklist 输出结论（通过/不通过），不通过则回到 3.2 修正。

### WINALL9 - 归档变更（阶段 4）
- 文件：`docs/change/2026-02-19_win-services-converge.md`
- 内容：背景/目标、变更映射、设计权衡、验证结果、回滚方案。

### WINALL10 - 合并与 push（你确认结束 workflow 后执行）
- 在 `repo/MyFlowHub-Win` 执行：
  1) `git merge --ff-only origin/refactor/win-services-converge`
  2) `git push origin main`
- 回滚点：revert 合并提交（或 revert 分支提交）。

