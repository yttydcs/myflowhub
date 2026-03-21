# 2026-02-19 Win：Auth/Flow/TopicBus/VarPool services 收敛（返回值 + 业务事件）

## 变更背景 / 目标
Win 前端此前通过监听 `session.frame` 并解析 `payload`（JSON decode + `action` switch）来驱动 Auth/Flow/TopicBus/VarPool 的业务状态更新，导致：
- UI 需要理解 wire 细节（action 名称、数据结构、异常语义）；
- 前端解析逻辑重复且分散（性能与维护成本高）；
- Win “上层应用”边界不清晰（协议机制向 UI 层扩散）。

本次目标：
1) **彻底移除**上述 4 个模块在前端对 `session.frame` 的业务解析依赖；
2) **req/resp** 类动作：Wails API 直接返回对应 Proto 响应结构体（`resp, error`）；
3) **notify/事件流**：Go 侧解析帧后发布“业务事件”，前端仅订阅业务事件，不处理 wire；
4) `session.frame` 事件仍保留用于 Debug/观测，但业务 UI 不再解析其 payload。

约束：wire 不改（SubProto / Action / JSON schema / Header 语义不变）。

## 具体变更内容（新增 / 修改 / 删除）

### 修改：Auth（返回值化）
- `internal/services/auth/service.go`
  - `Register* / Login*` 从仅返回 `error` 改为返回 `auth.RespData, error`；
  - 统一 await 超时/未连接等错误的 UI 友好化（`toUIError`）。
- `frontend/src/stores/session.ts`
  - 保留 `session.frame` 的时间戳更新（Debug/观测用），**移除** Auth payload 的业务解析。
- `frontend/src/pages/Home.vue`
  - `RegisterSimple / LoginSimple` 使用返回值更新 `sessionStore.auth` 与 Home 持久化状态，不再依赖 `session.frame`。

### 修改：Flow（返回值化）
- `internal/services/flow/service.go`
  - `List/Get/Set/Run/Status*` 改为返回对应的 `flow.*Resp, error`；
  - 统一 await 超时/未连接等错误的 UI 友好化（`toUIError`）。
- `frontend/src/stores/flow.ts`
  - 调用 Wails API 后直接处理返回值（`handle*Resp(resp)`），**移除** `session.frame` 订阅与 payload 解析。

### 修改：TopicBus（publish 事件下沉为业务事件）
- `internal/services/topicbus/service.go`
  - `Subscribe/Unsubscribe/ListSubs*` 返回 `topicbus.Resp` / `topicbus.ListResp`；
  - await 错误做 UI 友好化（`toUIError`）。
- `internal/services/topicbus/events.go`（新增）
  - 订阅 `session.frame`（仅 `MajorMsg + SubProtoTopicBus`），解析 `publish` 并发布业务事件 `topicbus.event`。
- `app.go`
  - bridge 新增 `topicbus.event` 到 Wails 前端；
  - Shutdown 增加 `topicbus.Close()`，清理订阅。
- `frontend/src/stores/topicbus.ts`
  - 订阅 `topicbus.event` 并维持原有节流/展示逻辑，**移除** `session.frame` 解析。

### 修改：VarPool（返回值化 + 通知下沉为业务事件）
- `internal/services/varpool/service.go`
  - `List/Get/Set/Revoke/Subscribe/Unsubscribe*` 返回 `varstore.VarResp, error`；
  - await 错误做 UI 友好化（`toUIError`）。
- `internal/services/varpool/events.go`（新增）
  - 订阅 `session.frame`（仅 `MajorMsg + SubProtoVarStore`），解析并发布：
    - `notify_set / up_set / var_changed` → `varpool.changed`
    - `notify_revoke / up_revoke / var_deleted` → `varpool.deleted`
- `app.go`
  - bridge 新增 `varpool.changed`、`varpool.deleted` 到 Wails 前端；
  - Shutdown 增加 `varpool.Close()`，清理订阅。
- `frontend/src/stores/varpool.ts`
  - `List/Get/Set/Revoke/Subscribe` 改为使用返回值更新；
  - 订阅 `varpool.changed / varpool.deleted` 更新 UI；
  - **移除** `session.frame` payload 解析。

## 对应 plan.md 任务映射
- WINALL1：Auth 返回值化 + 前端去 frame 解析（`AuthService`、`session.ts`、`Home.vue`）
- WINALL2：Flow 返回值化 + 前端去 frame 解析（`FlowService`、`flow.ts`）
- WINALL3：TopicBus publish 事件下沉为 Go 业务事件（`topicbus/events.go`、`topicbus.ts`、`app.go`）
- WINALL4：VarPool 返回值化 + 通知下沉为业务事件（`varpool/events.go`、`varpool.ts`、`app.go`）
- WINALL5：App bridge 新增业务事件 + 生命周期清理（`app.go`）
- WINALL6：回归验证（见下）

## 关键设计决策与权衡（性能 / 扩展性）
1) **业务事件下沉到 Go**：UI 不再解析 `action` 与 payload，降低协议扩散与维护成本。
2) **仅处理 `MajorMsg` 的通知帧**：TopicBus/VarPool 事件解析只对 `MajorMsg` 做 decode，避免对大量 OKResp/ErrResp 做 JSON 解包（性能关键点）。
3) **事件数据结构**：
   - TopicBus：发布 `topicbus.PublishReq`（字段清晰，前端用通用 stringify 展示）；
   - VarPool：发布 `varstore.VarResp`（复用既有字段集，前端无需理解 wire）。
4) `toUIError` 局部实现（每个 service 独立）：减少跨包耦合，但会有一定重复；后续如需要可收敛为公共 helper（需另起任务确认）。

## 测试与验证方式 / 结果
命令级验证（Windows）：
- `GOWORK=off go test ./... -count=1 -p 1`：通过
- `GOWORK=off wails generate module`：通过（`frontend/wailsjs/` 为 gitignore 产物）
- `cd frontend && npm ci && npm run build`：通过

手工冒烟（建议步骤，未在此文档中自动执行）：
1) 启动 `hub_server`（确保 auth/flow/topicbus/varstore 可用）
2) 启动 Win：`wails dev`
3) Home：Connect → Register/Login 成功（nodeId/hubId/role 持久化）
4) Flow：list/get/set/run/status 跑通
5) TopicBus：subscribe 后 publish，事件列表持续刷新
6) VarPool：listMine/get/set/revoke 正常；触发 notify/changed/deleted 时 UI 实时更新

## 潜在影响与回滚方案
潜在影响：
- 若服务端将 TopicBus/VarStore 的通知帧以非 `MajorMsg` 发送（非常规），Win 侧不会产生业务事件，UI 将看不到事件流/通知更新。

回滚方案：
- 直接 revert 本次分支相关提交（保持 wire 不变，回滚风险低）。
- 若仅需快速恢复事件流：可临时回退到前端 `session.frame` 解析（不建议长期保留）。

