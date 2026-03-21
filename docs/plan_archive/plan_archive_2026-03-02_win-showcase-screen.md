# Plan - MyFlowHub-Win：展示界面（Showcase Screen）

## Workflow 信息
- 范围：单仓库（`MyFlowHub-Win`）
- 分支：`feat/showcase-screen`
- Worktree：`d:\project\MyFlowHub3\worktrees\showcase-screen\MyFlowHub-Win`
- Base：`main`
- 规范：
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文，前缀可英文）
  - `d:\project\MyFlowHub3` 根目录 `AGENTS.md`（阶段纪律、worktree 禁令等）

---

## 0) 当前状态（可复用能力）
- Win 已具备：
  - TopicBus：`TopicBusService.PublishSimple/Subscribe*` + Wails 事件 `topicbus.event`
  - VarStore(VarPool)：`VarPoolService.Set/Get/List/Revoke/Subscribe/Unsubscribe/Send` + Wails 事件 `varpool.changed` / `varpool.deleted`
  - Profile 存储：`internal/storage` → `settings.json`，按 profile 前缀隔离
- 本 workflow 不改 wire、不新增子协议，仅做 Win 本地 UI + 本地配置持久化。

### 分支当前实现状态（待验收/待补齐边界）
> 说明：当前 worktree 内已存在初版实现草稿（未完成按本 plan 的验收与回归）。后续 3.2 将以本计划为准对齐实现并补齐测试与边界。
- Go：`app_showcase.go`、`app_showcase_test.go`
- 前端：`frontend/src/stores/showcase.ts`、`frontend/src/pages/Showcase.vue`
- 导航/路由：`frontend/src/layout/AppShell.vue`、`frontend/src/router/index.ts`

---

## 1) 需求分析（已确认）

### 目标
在 Win 端新增一个“展示界面（Showcase）”，用户可自行组装多个界面（Screen），并在界面中添加多个展示组件（Widget），用于：
1) 通过按钮发送 TopicBus publish 事件（topic/name/payload）。
2) 展示/控制 VarStore 变量（按 `ownerNodeID + varName` 引用，可自定义显示标题）。

### 术语
- Screen：展示界面实例（可创建多个）。
- Widget：展示组件（Screen 内的最小可配置单元）。

### 范围
必须：
- Screen：创建/重命名/删除/切换；配置按 profile 持久化。
- Widget：每个 widget 可单独配置 `targetId`（发送到哪个节点）。
  - `targetId` 必填；UI 创建/编辑时默认预填：Self NodeID；若 Self NodeID 未知（0）则兜底预填 `1`。
- TopicBus Button：
  - 点击发送 `publish(topic,name,payloadText)`。
  - payload 采用 Auto（合法 JSON 按 JSON，否则当字符串；由 TopicBus service 统一处理）。
- Var Widgets：
  - 引用：`ownerNodeID + varName`；允许自定义标题 `title`。
  - `mode`：
    - 默认 `auto`：根据变量响应里的 `type` 推断展示形态；
    - 允许用户手动选择 `display/slider/switch` 覆盖推断。
  - Slider：
    - 拖动实时发送：高频 `SendSimple(action=set, SetReq)`，不 await；
    - 限频：`throttleMs` 可配置；
      - `throttleMs=0`：不节流（每次 input 都发送）；
      - UI 必须明确提示 `throttleMs=0` 的拥塞风险；
    - 松手/变更确认：补发一次 `SetSimple(await)` 确认最终值。
  - Switch：
    - `onValue/offValue` 可配置且必填；
    - 显示态：当前值 `!= onValue` 视为 OFF；
    - 写入：严格按 `onValue/offValue` 写入。
  - Display：只读展示（兜底）。
  - `type` 必填（写入 VarStore `SetReq.Type`），默认预填建议：
    - `mode=slider` → `float64`
    - `mode=switch` → `bool`
    - `mode=display/auto` → `string`
- 生命周期：
  - 进入 Screen：对涉及变量执行 `Get + Subscribe`（按订阅键去重）。
  - 离开 Screen：自动 `Unsubscribe`（使用创建订阅时同一 `targetId`）。
- 未连接/未登录：禁用发送/写入，并给出清晰提示，不崩溃。

可选（本轮不做）：
- 事件流展示（本轮仅做事件按钮发送）。
- 拖拽布局/网格自定义（先做简洁布局，后续再加）。
- payload 模板/变量引用（本轮 payload 只是文本/JSON）。
- 导入/导出配置。

### 默认参数（可在 UI 中改）
- Slider：`min=0`、`max=100`、`step=1`、`throttleMs=50`（允许 `0`）。

### 验收标准（MVP）
1) 能创建 ≥2 个 Screen，并在重启后仍存在（同 profile）。
2) 每个 Screen 至少可添加并成功使用：
   - 1 个 TopicBus Button（发送成功可在 TopicBus 页面/Logs 中观察）。
   - 1 个 Var Slider + 1 个 Var Switch（对端或刷新后可观察变量变化）。
3) 切换 Screen/离开页面后，不应持续产生订阅（避免重复订阅/泄漏）。
4) 未连接/未登录时操作 widget：不会崩溃，提示清晰。
5) `throttleMs=0` 时拖动 slider：不会卡 UI；松手/变更后最终值可通过 `SetSimple(await)` 确认写入。

### 风险
- VarStore `type` 字段可能不规范，auto 推断不准：必须允许用户手动覆盖 mode；并允许用户配置 type。
- Slider 高频发送可能造成拥塞：必须支持限频，且 `throttleMs=0` 需要 UI 风险提示。
- 每 widget `targetId` 可能导致订阅/取消订阅不一致：必须以 `(targetId,ownerId,varName)` 为订阅键，且取消订阅使用同一 targetId。

---

## 2) 架构设计（分析）

### 总体方案（选型与理由）
仅在 `MyFlowHub-Win` 内落地（Win 本地 UI + 本地配置持久化），不新增子协议，不修改 Server/SubProto/Proto。
- 理由：MVP 最小改动、迭代快、风险可控。
- 备选（本轮不选）：新增子协议由服务端下发/管理展示界面配置（需要多端共享/权限审计时再开新 workflow）。

### 模块职责
- Go（Wails backend）
  - `App.ShowcaseConfig()`：读取并返回规范化配置（含默认 Screen）。
  - `App.SaveShowcaseConfig(cfg)`：校验+规范化后写入 profile settings，再返回保存后的配置。
- 前端 store：`frontend/src/stores/showcase.ts`
  - load/save 配置、Screen/Widget CRUD。
  - 生命周期：enter/leave 时批量 `Get/Subscribe/Unsubscribe`（去重，避免泄漏）。
  - Slider：按 `throttleMs` 发送 `SendSimple(set)`；松手/change 用 `SetSimple(await)` 确认最终值。
  - Switch：按 on/offValue 计算显示态与写入值。
  - 监听 `varpool.changed` / `varpool.deleted` 更新变量快照。
- 前端页面：`frontend/src/pages/Showcase.vue`
  - Screen 列表 + 当前 Screen widgets 渲染 + 创建/编辑弹窗 + 基本交互（发送/控制/刷新）。
- 路由/导航
  - `frontend/src/router/index.ts` 新增 `/showcase`
  - `frontend/src/layout/AppShell.vue` 新增入口

### 数据/调用流（关键链路）
- 进入页面：
  1) load config
  2) 计算需要订阅的 `(targetId,ownerId,varName)` 集合（去重）
  3) 对每个变量：`SubscribeSimple`（后台静默失败不 spam）+ `GetSimple` 拉取初值
- Wails 事件：
  - `varpool.changed`：更新快照
  - `varpool.deleted`：删除快照
- Slider：
  - input：本地更新显示值；按 `throttleMs` 频率 `SendSimple(set)`（`throttleMs=0` 则每次 input 都发送）
  - change/松手：`SetSimple(await)` 确认最终值（失败 toast）
- Switch：
  - change：写入 `onValue/offValue`，用 `SetSimple(await)`
- 离开页面/切换 Screen：
  - 清理 slider 定时器/草稿
  - 批量 `UnsubscribeSimple`（按 activeSubs 记录的订阅键）

### 错误与安全
- UI 输入强校验：
  - `targetId > 0`（必填）、`ownerId > 0`、`varName` 非空
  - TopicBus `topic/name` 非空
  - Slider：`min/max/step` 合法；`throttleMs >= 0`（允许 0）
  - Switch：`onValue/offValue` 非空
  - Var：`type` 非空（必填）
- 未连接/未登录：禁用发送/写入，提示清晰。
- 不执行脚本、不引入表达式语言；payload 仅文本，Auto JSON 包装由 TopicBus service 统一处理。

### 性能与测试策略
- 订阅去重键：`(targetId,ownerId,varName)`，避免重复 Subscribe。
- Slider 高频写入：
  - input 写入走 `SendSimple` 且不 await，避免 UI 卡顿；
  - 最终值走 `SetSimple(await)` 确保一致性；
  - `throttleMs=0` 必须有 UI 风险提示。
- 测试策略：
  - Go 单测：配置解析/规范化/默认值/非法 widget 丢弃；覆盖 `throttleMs=0` 不被改写。
  - 手工冒烟：按“验收标准（MVP）”逐条执行。

### 可扩展性设计点
- `config.version` + widget `kind` 可扩展（后续加布局、更多 widget 类型不破坏存量）。
- `mode=auto` + `type`/远端 type 推断可演进（未来可加“强制使用配置 type / 强制使用远端 type”策略位）。

---

## 3.1) 计划拆分（Checklist）

> 进入 3.2 前必须：本 plan.md 获得确认（阻塞：是）。

### 依赖与执行顺序
- 顺序：V1（Go 配置）→ V2（store）→ V3（页面/路由）→ V4（bindings 同步）→ V5（验收回归）→ 3.3（Code Review）→ 4（归档变更）
- 说明：由于当前分支已有草稿实现，执行时以每个任务的“验收条件/测试点”为准对齐与补齐。

### V1 - Go：配置模型 + 持久化 API（App）
- 目标：按 profile 读取/保存 Showcase 配置（JSON），并进行校验/规范化（含默认 Screen）。
- 涉及文件：
  - `app_showcase.go`
  - `app_showcase_test.go`
- 验收条件：
  - Wails 前端可调用 `ShowcaseConfig/SaveShowcaseConfig` 获得/保存配置；重启后仍存在（同 profile）。
  - 规范化规则正确：
    - 无 screens 时补默认 Screen；
    - 丢弃非法 widget；
    - slider 默认值补齐；
    - `throttleMs=0` 允许存在且不会被“纠正回默认 50”。
- 测试点：
  - `go test ./... -count=1 -p 1`
- 回滚点：
  - 移除 `app_showcase*.go` 与 `showcase.config` 存储 key，不影响既有模块。

### V2 - 前端：Showcase store（状态/生命周期/发送）
- 目标：实现 Screen/Widget CRUD、load/save 配置、enter/leave 自动订阅、slider 限频发送与松手确认。
- 涉及文件：
  - `frontend/src/stores/showcase.ts`
- 验收条件：
  - `enter()` 会对当前 Screen 中涉及的变量做 `Subscribe + Get`（去重）。
  - `leave()` 会 `Unsubscribe` 并清理 slider 定时器/草稿，不产生订阅泄漏。
  - Slider：
    - `throttleMs>0`：按节流频率 `SendSimple(set)`；
    - `throttleMs=0`：每次 input 都 `SendSimple(set)`，且不 await、不阻塞 UI；
    - change/松手：必须 `SetSimple(await)` 确认最终值。
  - Switch：显示态/写入严格遵循 `onValue/offValue` 规则。
- 测试点：
  - 手工：切换 Screen/离开页面后不持续订阅（看 Logs 里的 subscribe/unsubscribe 调用）。
  - 手工：`throttleMs=0` 拖动 slider 时 UI 不冻结，且松手后最终值能稳定落地。
- 回滚点：
  - 删除 `frontend/src/stores/showcase.ts` 并移除相关路由入口即可回退。

### V3 - 前端：Showcase 页面 + 路由 + 导航入口
- 目标：新增 `Showcase.vue` 页面与导航入口（AppShell + router），提供 Screen/Widget 的最小可用交互。
- 涉及文件：
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/router/index.ts`
  - `frontend/src/layout/AppShell.vue`
- 验收条件：
  - 导航可进入 Showcase 页面；可创建 ≥2 个 Screen；每个 Screen 可添加/编辑/删除 widget。
  - 表单强校验与默认预填：
    - `targetId` 必填且预填 Self NodeID；Self=0 时预填 1；
    - Var `ownerId` 预填 Self NodeID；
    - Var `type` 必填且按 mode 默认预填建议值。
  - `throttleMs=0`：在表单或组件处显示明确风险提示文案。
  - 未连接/未登录：发送/写入按钮禁用且提示明确。
- 测试点：
  - `npm run build`（前端构建无报错）
- 回滚点：
  - 移除路由 `/showcase`、导航项、页面文件即可回退。

### V4 - Wails bindings 同步（如需要）
- 目标：确保 `frontend/wailsjs/**` 与 Go bindings 同步（避免类型/调用缺失）。
- 验收条件：
  - `wails dev` / `wails build` 不报 binding 缺失。
- 测试点：
  - 运行一次 `wails generate`（如项目需要）后再 `npm run build` / `go test`。
- 回滚点：
  - 回滚本次生成文件变更。

### V5 - 验收与回归（手工 + 最小自动化）
- 目标：完成 MVP 验收标准；补齐必要错误处理与提示文案。
- 验收条件：
  - 通过“验收标准（MVP）”全部条目。
- 测试点：
  - `go test ./... -count=1 -p 1`
  - `npm run build`
- 回滚点：
  - 回滚此分支所有提交即可。

---

## 3.3) Code Review（完成编码后执行）
- 需求覆盖：多 Screen、多 widget、topic publish、var slider/switch、按 profile 存储、targetId/type 必填、`throttleMs=0` 语义与提示
- 架构合理性：只改 Win；模块边界清晰；不侵入既有 VarPool/TopicBus 页面
- 性能风险：去重订阅；slider 高频 `SendSimple` 不 await；`throttleMs=0` 风险提示；避免重复计算/重复订阅
- 可读性与一致性：命名/结构与现有 stores/pages 风格一致
- 可扩展性与配置化：widget kind/version；参数可配置（min/max/step/throttle/on/off/type/targetId）
- 稳定性与安全：输入校验、错误处理、默认安全（不执行脚本）
- 测试覆盖：Go 单测 + 手工冒烟覆盖关键路径与边界

---

## 4) 归档变更（完成 Review 后执行）
- 在 worktree 根目录创建 `docs/change/` 并新增文档：`docs/change/2026-03-02_showcase-screen.md`
- 内容必须包含：
  - 变更背景 / 目标
  - 具体变更内容（新增 / 修改 / 删除）
  - 对应 `plan.md` 任务映射（V1-V5）
  - 关键设计决策与权衡（尤其 `throttleMs=0` 与性能风险、订阅去重策略）
  - 测试与验证方式 / 结果
  - 潜在影响与回滚方案
