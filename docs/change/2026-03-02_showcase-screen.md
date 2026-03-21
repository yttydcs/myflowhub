# 2026-03-02：Win Showcase（展示界面）MVP

## 背景 / 目标
为了便于控制与展示信号（TopicBus 事件与 VarStore 变量），在 Win 端新增一个可由用户自行组装的“展示界面（Showcase Screen）”，允许创建多个 Screen，并在每个 Screen 内配置多个 Widget。

本次实现定位为 Win 本地能力（不新增子协议、不改 Server/Proto），配置按 profile 存储，快速迭代 MVP。

## 具体变更内容

### 新增（Go / Wails backend）
- 新增 Showcase 配置持久化：
  - `app_showcase.go`：`App.ShowcaseConfig()` / `App.SaveShowcaseConfig(cfg)`，存储 key：`showcase.config`（profile-scoped）。
  - 配置模型：Screen/Widget（topic_button / var）与 slider/switch 参数。
  - 规范化策略：
    - 无 Screen 时补默认 Screen；
    - 丢弃非法 widget（缺必要字段）；
    - `widget.targetId` 为空时规范化为 `1`；
    - Var `type` 为空时按 mode 补默认（slider=float64、switch=bool、其他=string）；
    - Slider `throttleMs` 允许为 `0`（不节流），仅当 `<0` 时回退默认值。
- 单测：
  - `app_showcase_test.go`：覆盖默认 Screen、非法 widget 丢弃、`targetId/type` 默认值、`throttleMs=0` 保留。

### 新增（前端 / Vue）
- 新增 Showcase 页面与 store：
  - `frontend/src/stores/showcase.ts`：load/save 配置、Screen/Widget CRUD、enter/leave 自动订阅、TopicBus publish、VarStore slider/switch 控制。
  - `frontend/src/pages/Showcase.vue`：Screen 列表、Widget 卡片、创建/编辑弹窗（事件按钮 / 变量组件）。
- 路由/导航入口：
  - `frontend/src/router/index.ts`
  - `frontend/src/layout/AppShell.vue`

### 关键交互与规则
- TopicBus Button：点击调用 `TopicBusService.PublishSimple(...)`；payload 走 Auto（合法 JSON 视为 JSON，否则当字符串）。
- Var Widget：
  - 引用：`ownerId + varName`；允许自定义 `title`；
  - `mode=auto`：按变量响应里的 `type` 推断展示形态（bool→switch；number→slider；否则 display），允许手动覆盖；
  - `type` 为必填（写入 `SetReq.Type`）；默认按 mode 预填建议值；
  - Slider：拖动高频 `SendSimple(set)`（不 await UI），松手/change 用 `SetSimple(await)` 确认最终值；
  - `throttleMs=0`：不节流（每次 input 都发送），UI 提示拥塞风险；
  - Switch：显示态使用 `value == onValue` 判断；写入严格使用 on/offValue。

## 对应 plan.md 任务映射
- V1：Go 配置模型 + 持久化 API → `app_showcase.go`、`app_showcase_test.go`
- V2：前端 Showcase store → `frontend/src/stores/showcase.ts`
- V3：Showcase 页面 + 路由 + 导航 → `frontend/src/pages/Showcase.vue`、`frontend/src/router/index.ts`、`frontend/src/layout/AppShell.vue`
- V4：Wails bindings 同步（仅生成本地文件，gitignore） → `wails generate module`
- V5：验收与回归 → `go test`、`npm run build`、手工冒烟

## 关键设计决策与权衡
- 不引入新子协议：MVP 仅做 Win 本地配置与 UI，降低改动面与联调成本。
- `throttleMs=0` 允许：满足“低延迟/高频率”场景，但强制 UI 提示风险；并通过“松手 Set/await”保证最终一致性。
- 订阅去重键：`(targetId,ownerId,varName)`，避免重复 Subscribe；离开页面统一 Unsubscribe 防泄漏。

## 测试与验证方式 / 结果
- Go：
  - `GOWORK=off go test ./... -count=1 -p 1`（通过）
- 前端：
  - `npm ci`
  - `GOWORK=off wails generate module`（生成 `frontend/wailsjs/**`，该目录被 `.gitignore` 忽略）
  - `npm run build`（通过）
- 手工冒烟（建议执行）：
  - 创建 ≥2 个 Screen，分别添加 TopicBus Button / Var Slider / Var Switch；
  - 切换 Screen、离开页面确认不重复订阅；
  - `throttleMs=0` 拖动 slider，确认 UI 不冻结且松手后最终值落地。

## 潜在影响与回滚方案
- 影响：
  - 新增 Showcase 页面入口与配置存储 key：`showcase.config`。
  - `frontend/wailsjs/**` 为本地生成依赖（gitignore），CI/构建环境需要执行 `wails generate module`。
- 回滚：
  - 删除 Showcase 页面/路由/导航、移除 `app_showcase*.go`，并忽略/清理 `showcase.config` 存储 key 即可回退，不影响既有 TopicBus/VarPool 模块。

