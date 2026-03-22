# Plan - MyFlowHub-Win：VarPool 节点变量弹窗 + Watch 订阅状态本地持久化

## Workflow 信息
- 范围：单仓库（`MyFlowHub-Win`）
- 分支：`feat/varpool-vars-dialog`
- Worktree：`d:\project\MyFlowHub3\worktrees\varpool-vars-dialog\MyFlowHub-Win`
- Base：`main`
- 规范：
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文，前缀可英文）
  - `d:\project\MyFlowHub3` 根目录 `AGENTS.md`（阶段纪律、worktree 禁令等）

---

## 0) 当前状态（复用能力）
- VarPool 页面：`frontend/src/pages/VarPool.vue`（支持 Get/Set/List/Revoke/Subscribe/Unsubscribe）
- 前端 store：`frontend/src/stores/varpool.ts`
  - 已有 watch 列表（keys）与 value cache（data）
  - 已有内存态 `desiredSubs`（用于避免 subscribe_resp 竞态），但 **不会持久化**，重启后丢失
- Go 本地存储：`internal/storage` → `settings.json`（按 profile 前缀隔离）
  - watch 列表已持久化：`app_varpool.go` 使用 key `varpool.names`
- Go VarPool service：`internal/services/varpool/*`
  - bindings：`VarPoolService.ListSimple/GetSimple/SubscribeSimple/UnsubscribeSimple/...`
  - 业务事件：`varpool.changed` / `varpool.deleted`
- 已知限制（本 workflow 接受）：
  - VarStore 的 `list` 当前仅能列出 **public** 变量名（无服务端分页能力）。

---

## 1) 需求分析（已确认）

### 目标
1) 在 Win `VarPool` 页面新增弹窗：可按 owner NodeID 列出该节点的变量名列表，便捷 Add Watch。
2) watch 的订阅偏好（每个 `name#owner` 的 subscribe=true/false）持久化到本地，并在 **每次 session 重连/重新登录** 后自动恢复订阅；不依赖打开 `/varpool` 页面。

### 范围（必须 / 可选 / 不做）
- 必须：
  - 入口 C：
    - `#/varpool` 页面提供入口打开弹窗；
    - `#/devices` 节点列表每个节点提供入口打开弹窗（owner 预填为该节点）。
  - 弹窗：
    - 以 `owner NodeID` 为查询对象；
    - `list` 请求的 `targetID = owner NodeID`（直查该节点）；
    - 默认仅展示变量 `name`；支持搜索与客户端分页；
    - 点击 `Add Watch`：仅添加 watch（不自动订阅）。
  - 订阅偏好：
    - 按 profile 本地持久化 subscribe=true/false；
    - 重连/重新登录自动恢复订阅；
    - Subscribe/Unsubscribe 的手动操作会同步更新持久化。
- 可选（本轮先做轻量实现）：
  - 点击变量名后再 `Get` 拉取 value/type/visibility（避免全量 N+1）。
- 不做：
  - 扩展协议/wire 或服务端分页；
  - 列出 private 变量名（当前服务端 list 逻辑无法提供）。

### 验收标准（MVP）
1) 在 `#/devices` 选任意节点，点击 “Vars” 能打开弹窗并成功加载变量名列表；点击 `Add Watch` 后，该变量出现在 `#/varpool` 的 watched 列表中（重启后仍在）。
2) 在 `#/varpool` 对某个 watched 变量点击 Subscribe 后：重启应用 → 重新连接并登录后自动恢复为已订阅（无需手点）。
3) 断线重连/重新登录：会再次自动恢复订阅，不依赖打开 `#/varpool`。
4) 未连接/未登录：弹窗加载与订阅恢复不会崩溃，提示清晰。

### 风险
- list 仅 public：弹窗看不到 private 变量名（但可手工 Add Watch）。
- 自动恢复订阅数量大时可能产生瞬时请求峰值：需要做并发限制与失败汇总。

---

## 2) 架构设计（分析）

### 总体方案（含选型理由）
- UI 侧新增可复用弹窗组件（基于现有 `Overlay`），在 `VarPool.vue` 与 `Devices.vue` 复用。
- 数据侧复用 VarStore `ListSimple` 获取 names（无服务端分页 → 客户端分页/搜索）。
- 持久化新增独立 key 存储“订阅偏好”，保持现有 watch list key 不变，降低迁移风险。
- 全局自动恢复由 `AppShell.vue` 驱动：监听 session/profile 变化，加载配置并触发恢复逻辑。

### 模块职责
- Go（`App`）
  - 新增：`VarPoolSubPrefs` / `SaveVarPoolSubPrefs`（profile-scoped）。
- 前端 store（`frontend/src/stores/varpool.ts`）
  - 加载/保存订阅偏好；Subscribe/Unsubscribe 时同步持久化；移除 watch 时清理偏好。
  - 自动恢复订阅：并发限制、失败不刷屏。
  - 提供 `listOwnerNames(ownerId)` 给弹窗调用（`targetID=ownerId`）。
- 前端 UI
  - 新增弹窗组件（Node vars dialog）：加载列表、搜索、分页、Add Watch。
  - `Devices.vue` / `VarPool.vue` 接入弹窗。
- `AppShell.vue`
  - 在应用层启动 varpool store：不依赖打开页面；在 session ready 时触发恢复。

### 接口草案
- Wails (Go App)：
  - `VarPoolSubPrefs() -> []{name, owner, subscribed}`
  - `SaveVarPoolSubPrefs(prefs) -> normalized prefs`
- 弹窗 list：
  - `VarPoolService.ListSimple(sourceID, targetID=ownerID, {owner: ownerID})`

### 错误与安全
- 输入校验：ownerID > 0；name 非空；未连接/未登录禁止请求。
- 安全默认：Add Watch 不自动订阅；仅对 `subscribed=true` 的偏好自动恢复。

### 性能与测试策略
- list：单请求拿 names，客户端分页/搜索；可选点击再 Get，不做全量 N+1。
- 自动恢复订阅：限制并发（例如 4），失败汇总一次 toast。
- 测试：Go 单测覆盖解析/规范化；手工冒烟覆盖 Devices/VarPool 弹窗与重连恢复。

### 可扩展性设计点
- 订阅偏好结构为 `{name, owner, subscribed}`，后续可扩展字段（例如 `lastError/lastRestoredAt`）。
- 弹窗可扩展为批量 Add Watch、Add&Subscribe、value 预览等。

---

## 3.1) 计划拆分（Checklist）

> 进入 3.2 前必须：本 plan.md 获得确认（阻塞：是）。

### 依赖与顺序
- 顺序：V1（Go 存储 API）→ V2（前端 store 持久化 + 自动恢复）→ V3（弹窗组件）→ V4（Devices/VarPool/AppShell 接入）→ V5（验证回归）→ 3.3（Code Review）→ 4（归档）

### V1 - Go：订阅偏好持久化 API（App）
- 目标：在 `settings.json`（profile-scoped）新增保存/读取 VarPool 订阅偏好。
- 涉及文件（预期）：
  - `app_varpool.go`
  - `app_varpool_test.go`（新增）
- 验收条件：
  - 前端可调用 `VarPoolSubPrefs/SaveVarPoolSubPrefs`；重启后仍能读取；
  - 规范化规则正确：去重、trim name、丢弃非法项（空 name / owner=0 视需求处理）。
- 测试点：
  - `GOWORK=off go test ./... -count=1 -p 1`
- 回滚点：
  - 删除新增 key 与方法，不影响既有 `varpool.names` watch 列表。

### V2 - 前端：varpool store 持久化 + 自动恢复（不依赖页面）
- 目标：
  - 加载/保存订阅偏好；Subscribe/Unsubscribe 同步更新偏好；
  - session 重连/重新登录后自动恢复订阅（并发限制）。
- 涉及文件（预期）：
  - `frontend/src/stores/varpool.ts`
- 验收条件：
  - Subscribe 后重启并重连/登录，会自动恢复订阅；
  - Unsubscribe 会持久化为 false，重连不会再次自动订阅；
  - remove watch 会清理对应偏好；
  - 恢复过程失败不刷屏（最多 1 次汇总 toast），并记录到 console/logs（按现有风格）。
- 测试点：
  - 手工：订阅 3 个变量 → 断线重连 → 自动恢复且无重复订阅风暴。
- 回滚点：
  - 移除偏好读写与自动恢复逻辑，不影响现有手动订阅。

### V3 - 前端：节点变量列表弹窗组件（client paging + search）
- 目标：实现可复用弹窗：按 owner NodeID 拉取 names，支持搜索/分页与 Add Watch。
- 涉及文件（预期）：
  - `frontend/src/components/varpool/NodeVarsDialog.vue`（新增）
- 验收条件：
  - `targetID=ownerID` 的 list 能加载 names；
  - 搜索与分页只在前端处理（不影响请求）；
  - Add Watch 对已存在 key 去重（按钮禁用或提示）。
- 测试点：
  - 手工：owner 变量名列表较多时翻页/搜索正常。
- 回滚点：
  - 删除新增组件与入口按钮。

### V4 - 接入：Devices / VarPool / AppShell
- 目标：
  - `Devices.vue`：每个节点提供 “Vars” 按钮打开弹窗（owner 预填）。
  - `VarPool.vue`：提供入口打开弹窗（owner 可编辑）。
  - `AppShell.vue`：应用启动后初始化 varpool store，并在 session/profile 变化时触发加载与恢复（不依赖页面）。
- 涉及文件（预期）：
  - `frontend/src/pages/Devices.vue`
  - `frontend/src/pages/VarPool.vue`
  - `frontend/src/layout/AppShell.vue`
- 验收条件：
  - 两处入口均可用；
  - 自动恢复在未打开 `#/varpool` 的情况下仍会执行（可通过 Logs/subscribe 状态观察）。
- 测试点：
  - 手工：只停留在 Devices 页，断线重连后仍自动恢复订阅。
- 回滚点：
  - 移除入口与 AppShell 初始化，不影响其他模块。

### V5 - bindings 同步 + 回归验收
- 目标：生成 Wails bindings、完成 MVP 验收条目，并确认无明显 UI 回归。
- 验收：
  - `GOWORK=off wails generate module` 成功；
  - `GOWORK=off go test ./... -count=1 -p 1` 通过；
  - `cd frontend; npm run build` 通过；
  - 手工验收 1~4 全部通过。
- 回滚点：
  - 回滚本分支全部提交。

---

## 3.3) Code Review（完成编码后执行）
- 需求覆盖：弹窗列表、Devices/VarPool 入口、订阅偏好持久化、重连恢复不依赖页面
- 架构合理性：职责边界清晰（App 存储 / store 恢复 / UI 仅展示）；无循环依赖
- 性能风险：list 无 N+1；恢复订阅并发受控；避免重复计算与重复 I/O
- 可读性与一致性：命名/结构与现有 pages/stores 风格一致
- 稳定性与安全：输入校验、未连接提示、失败处理不刷屏
- 测试覆盖：Go 单测 + 前端 build + 手工冒烟

---

## 4) 归档变更（完成 Review 后执行）
- 在 worktree 根目录创建 `docs/change/` 并新增文档：`docs/change/2026-03-03_varpool-dialog-subprefs.md`
- 文档必须包含：背景/目标、变更内容、plan 任务映射、关键决策与权衡（target=owner、client paging、并发恢复策略）、测试结果、影响与回滚方案
