# Plan - Win：Management services 收敛（返回值 + UI 友好错误/日志）

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`refactor/win-mgmt-service`
- Worktree：`d:\project\MyFlowHub3\worktrees\win-mgmt-service\MyFlowHub-Win`
- Base：`main`
- 参考：
  - `d:\project\MyFlowHub3\target.md`
  - `d:\project\MyFlowHub3\repos.md`（重点：L171 “Win services 收敛”）
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文）

## 约束（边界）
- 本 workflow 仅处理 **Management 模块**（先做一个子协议作为样板）。
- 不改 wire（SubProto/Action/JSON schema 不变）。
- `session.frame` 事件 **保留**（用于 Debug/原始帧观察），但 Management UI 不再依赖它做业务更新。
- Linux 构建验收暂时忽略；以 Windows 为准。
- 验收建议使用 `GOWORK=off`（确保单仓 clone 可用）。

## 当前状态（事实，可审计）
- `internal/services/management` 已实现 send+await，并能解包 `code/msg`，但对前端仅暴露 `error`（不返回业务数据）。
- `frontend/src/stores/management.ts` 监听 `session.frame`，自行解析 payload → `switch(action)` 更新 nodes/config：
  - 造成 UI 与 wire 强耦合、重复解码逻辑、错误体验与可维护性较差。

---

## 1) 需求分析（待确认）

### 目标
1) 让 Win 的 Management service 成为“业务编排层”：Wails API 直接返回 `myflowhub-proto/protocol/management` 的响应结构体（resp），前端用返回值更新 UI。
2) Management UI 不再依赖 `session.frame` 做业务更新（但 Debug 仍可观察帧）。
3) 提升 UI 友好错误/日志：
   - UI 侧拿到可读、可定位的错误信息；
   - Go 侧记录更完整的失败原因（含 action、timeout、code/msg 等）。

### 范围（必须 / 可选 / 不做）
#### 必须（本 PR）
- Go（`internal/services/management/service.go`）
  - 将以下方法从“仅返回 error”调整为“返回 resp + error”：
    - `NodeEcho*`、`ListNodes*`、`ListSubtree*`、`ConfigList*`、`ConfigGet*`、`ConfigSet*`
  - 统一输入校验、超时、resp code 判定：`code != 1` → 返回 error（msg 优先）。
  - 在失败时追加 UI 友好错误（必要时归一化超时/未连接等）并写入 logs。
- 前端（`frontend/src/stores/management.ts`）
  - 移除对 `session.frame` 的解析依赖（不再 decode payload / JSON.parse / switch(action)）。
  - 调用 Go 返回值并更新 `state.nodes / state.configEntries / state.message`。
  - 保持现有交互体验：选择节点 → 加载 keys → 并发拉取各 key 的 value。

#### 可选（本 PR，如不扩大改动则不做）
- 将 Management store 的“并发拉取 ConfigGet”加一个轻量并发上限（避免极端 key 数导致瞬时并发过高）。

#### 不做（本 PR）
- 不迁移其它子协议（auth/varpool/topicbus/file/flow 另起 workflow 按模块推进）。
- 不移除 `session.frame`，不改变 Debug/Logs 页面的帧展示。
- 不改 SDK/Proto/Server/Core（除非出现阻塞性缺陷）。

### 使用场景
- 已连接并登录后：在 Management 页面点击 List Direct/Subtree；选择节点后查看/编辑 config。
- 断线/超时/服务器返回失败：UI 给出可读提示；Logs 中能定位 action 与失败原因。

### 输入输出（Wails API）
- 输入：`sourceID/targetID`（node id），以及 `message/key/value`。
- 输出：
  - 成功：返回对应 `*Resp`（`code==1`）。
  - 失败：返回 `error`（优先展示 `msg`；必要时转换为更友好的超时/未连接等提示）。

### 边界异常
- 未登录 / identity 缺失：前端在调用前阻止并提示（保持现状）。
- 未连接 / session 未初始化：Go 返回明确错误并写日志。
- 超时：Go 返回 UI 友好错误（例如 “请求超时(8s)”），并记录内部原因。
- 响应解包失败 / schema 不匹配：Go 返回错误（协议问题），并写日志。

### 验收标准
- `frontend/src/stores/management.ts` 不再监听 `session.frame`（Management 业务不靠 frame 更新）。
- Management 页面功能可用：
  - List Direct/Subtree 能展示 nodes；
  - 选择 node 能展示 config keys，并逐步填充 values；
  - 编辑并保存 config 后，UI 状态与提示正确。
- 命令级验证：
  - `GOWORK=off go test ./... -count=1 -p 1`
  - `frontend` 能 `npm run build`
- 冒烟（手工）：Win 启动 → Connect → 登录 → 打开 Management 页面完成以上操作。

### 风险
- Wails 返回值序列化形态变化：需要确认前端按 JSON 字段读取（使用 `node_id/has_children` 等）。
- Config keys 数量过大时并发 `ConfigGet` 可能造成短时压力（可选并发上限缓解）。

## 问题清单（阻塞：否）
- 你已确认：先做 Management（A）、保留 `session.frame`、方式选 A（返回值）。

---

## 2) 架构设计（分析）

### 总体方案（采用：方式 A - 返回值）
- 对于 Management 这种典型 req/resp 协议：采用“Wails 方法返回 resp + error”。
- `session.frame` 继续作为 Debug 通道存在，但 UI 业务数据流以“返回值”为主。

### 备选对比
- 备选 B：仍由前端监听 `session.frame` 并解析（不采用）
  - 问题：UI 与 wire 强耦合、重复解析、难以统一错误与可观测性。
- 备选 C：Go 侧解析 frame 后发业务事件（`management.nodesUpdated` 等）（暂不采用）
  - 适合 notify/订阅类协议；但对 Management 这类请求式交互，返回值更直观、侵入更小。

### 模块职责
- `internal/services/session`
  - 维护连接；提供 `SendCommandAndAwait`；发布 `session.frame/state/error` 事件。
- `internal/services/management`
  - 业务编排：输入校验、请求编码、await、resp 解包、错误归一化与日志。
- `frontend/src/stores/management.ts`
  - 纯 UI 状态管理：调用 binding，基于 resp 更新 state；不处理 wire 解码。

### 数据 / 调用流（关键链路）
1) UI：`ListNodesSimple(sourceID,targetID)`
2) Go：`EncodeMessage(action,data)` → `SessionService.SendCommandAndAwait(..., expectAction=*_resp)`
3) Go：`json.Unmarshal(resp.Message.Data)` → 判定 `code/msg` → 返回 resp 或 error
4) UI：拿到 resp 更新列表/配置状态

### 接口草案（Go / Wails）
- `NodeEchoSimple(sourceID, targetID uint32, message string) (management.NodeEchoResp, error)`
- `ListNodesSimple(sourceID, targetID uint32) (management.ListNodesResp, error)`
- `ListSubtreeSimple(sourceID, targetID uint32) (management.ListSubtreeResp, error)`
- `ConfigListSimple(sourceID, targetID uint32) (management.ConfigListResp, error)`
- `ConfigGetSimple(sourceID, targetID uint32, key string) (management.ConfigResp, error)`
- `ConfigSetSimple(sourceID, targetID uint32, key, value string) (management.ConfigResp, error)`

### 错误与安全
- 输入校验：空 key/message 立即返回 error。
- `code!=1`：返回 msg（如为空则回落到 `code` 描述）；并写入 warn 日志。
- 超时/取消：归一化为更友好的 UI 文案；内部原因写入日志。
- 注意：config value 可能包含敏感信息；logs 侧避免主动打印完整 value（仅记录 action、key、结果）。

### 性能与测试策略
- 性能：减少前端对每帧的解析与 switch；ConfigGet 仍为多请求（与现状一致）。
- 测试：以“能 build + go test + 手工冒烟”为主；如后续把 store 抽象出来，可加 TS 单测（本 PR 不做）。

### 可扩展性设计点
- 本 PR 产出的“返回值式 service + UI 友好错误”作为模板，可逐协议迁移：
  - req/resp 类（auth/varpool/flow 等）优先走返回值。
  - notify/订阅类（topicbus notify 等）后续再用 Go 侧事件化收敛。

---

## 3.1) 计划拆分（Checklist）

### WINMGMT0 - 计划归档（如需要）
- 目标：保留上一轮 `plan.md` 内容（避免被覆盖后丢失）。
- 涉及文件：
  - `plan_archive_2026-02-18_win-semver-deps.md`（如需纳入 git）
- 验收条件：归档文件可独立阅读复现上一轮结论。
- 回滚点：删除该归档文件。

### WINMGMT1 - Go：ManagementService 返回 resp + error（业务编排层）
- 目标：前端不再需要解析 `session.frame` 即可获得业务数据。
- 涉及文件：
  - `internal/services/management/service.go`
- 验收条件：
  - 以上 6 组方法签名调整为返回 resp；
  - `code!=1` 返回 error（msg 优先）；
  - 失败写入 logs（含 action、原因）；
  - 不改 wire。
- 测试点：
  - `GOWORK=off go test ./... -count=1 -p 1`
- 回滚点：revert 该提交。

### WINMGMT2 - 前端：Management store 迁移到返回值（移除 frame 解析）
- 目标：Management UI 状态仅依赖调用返回值，不依赖 frame。
- 涉及文件：
  - `frontend/src/stores/management.ts`
  - （如需要）`frontend/src/pages/Management.vue`
- 验收条件：
  - store 不再 `EventsOn("session.frame")`；
  - list/select/config 操作都能正确更新 UI；
  - 选择节点快速切换时不会错写到当前节点（加必要的竞态保护）。
- 测试点：
  - `npm install`（如未安装）
  - `npm run build`
- 回滚点：revert 该提交。

### WINMGMT3 - 冒烟验证步骤（手工）
- 目标：可交接的验证流程，覆盖关键路径。
- 步骤：
  1) 启动 `hub_server`（按你的既有方式）并确保 Management handler 可用。
  2) 启动 Win：`wails dev`（或运行已构建 exe）。
  3) Connect 到 server → 登录成功（Session 页面应显示 nodeId/hubId）。
  4) 打开 Management：
     - 点击 List Direct/Subtree 能看到 nodes；
     - 选择 node：看到 keys，values 逐步填充；
     - Edit 保存：提示成功且 value 更新。
- 验收条件：上述流程通过。

### WINMGMT4 - Code Review（阶段 3.3）
- 目标：按 checklist 输出结论（通过/不通过）。

### WINMGMT5 - 归档变更（阶段 4）
- 目标：记录本次“Management UI 解耦 session.frame”的背景、设计与验证结果。
- 涉及文件：
  - `docs/change/2026-02-18_win-mgmt-service.md`
- 验收条件：文档包含：任务映射、关键决策、验证命令、回滚方案。

### WINMGMT6 - 合并与 push（你确认结束 workflow 后执行）
- 目标：合并到 `main` 并 push。
- 步骤（在 `repo/MyFlowHub-Win` 执行）：
  1) `git merge --ff-only origin/refactor/win-mgmt-service`
  2) `git push origin main`
- 回滚点：revert 合并提交（或 revert 分支提交）。
