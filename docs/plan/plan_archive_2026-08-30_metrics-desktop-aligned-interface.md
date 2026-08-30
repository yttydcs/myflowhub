# Plan Archive - Metrics 对齐 Desktop 视觉语言的独立界面

## Workflow Information

- Canonical repo: `D:\project\MyFlowHub3\repo\MyFlowHub`
- Branch: `feat/metrics-desktop-ui`
- Base: `master` @ `54b46e6bc0c1bbea5b265a875e7c8132ae28e004`
- Project root: `D:\project\MyFlowHub3`
- Worktree: `D:\project\MyFlowHub3\worktrees\metrics-desktop-ui`
- Docs root: `D:\project\MyFlowHub3\worktrees\metrics-desktop-ui\docs`
- Participating modules: `apps/nodes/metrics/windows/frontend`、`apps/nodes/metrics/windows`、canonical `docs/`
- Current stage: `$m-archive` complete；archive、local merge 与 worktree cleanup 已完成
- Publication: local-only；不新增 remote，不 push、release 或 publish

## Planning Inputs

- 用户已确认 Metrics 高保真设计稿，要求进入实施规划。
- 讨论记录：`D:\project\MyFlowHub3\repo\MyFlowHub\design-demos\metrics-desktop-aligned\discussion-brief.md`
- 高保真原型：`D:\project\MyFlowHub3\repo\MyFlowHub\design-demos\metrics-desktop-aligned\metrics-desktop-aligned.html`
- 本 worktree intake：`docs/intake/2026-08-30_metrics-desktop-aligned-interface.md`
- 当前能力事实源：`docs/features/metrics-node.md`、Metrics Wails contracts/backend 和现有 frontend。
- 主检出中的产品独立性草案只作为已接受约束；执行阶段不得覆盖、暂存或改写主检出中的用户未提交内容。

## Locked Scope Decisions

1. Metrics 是独立产品，只复用 Desktop 的视觉语言；不导入 Desktop 私有组件、状态、IPC 或运行时。
2. 保留现有 Wails API：`Identity`、`Start`、`Stop`、`Status`、`Configuration`、`Definitions`、`UpdateConfiguration`；本阶段不修改后端 contract。
3. 默认页为“资源状态”；页面刷新不记忆最后导航页。主题使用 Metrics 自有、带版本的 local-storage key 持久化。
4. 宽屏显示右侧上下文面板；窄屏隐藏的内容只能是补充信息，运行状态、错误与操作必须留在主内容区。
5. 只渲染真实 `Status.samples`。原型中的示例数值不是生产数据，不得进入生产 fallback。
6. 音量、亮度即时调节不在本阶段实现，因为当前后端没有对应控制 operation；`writable` 仍仅表示既有采集配置。
7. 主题 token 和 canonical 品牌资产可以共享设计规范，但 Metrics 必须拥有自己的静态资源副本和样式边界。

## Current-State Findings

- 当前 frontend 是 `main.ts` 中的单文件命令式 DOM，导航、主题、选中状态和可释放轮询都没有独立边界。
- 当前样式为深色径向渐变卡片布局，与 Desktop 的浅色工作台视觉语法不一致，且页面最小宽度会限制紧凑窗口。
- `Status.samples` 已提供 metric、value、unit、`fresh|stale|unavailable`、采样/观测时间和 error，足以实现真实资源表格与上下文信息。
- 配置、连接、身份和启停 API 已存在；没有必要改动 Go backend 或重新定义 Wails bindings。
- 前端只有纯函数测试，缺少导航、主题、轮询释放、请求互斥和错误反馈的 DOM 级回归测试。
- `dist/` 是受版本管理的生产产物，完成实现后需要由正式 build 更新并审查。

## Target Architecture

### Module Boundaries

| Area | Planned responsibility |
| --- | --- |
| `src/main.ts` | 只负责定位根节点、创建 production API adapter、启动应用并在卸载时调用 `dispose()`。 |
| `src/api.ts` | 定义可注入的 `MetricsAPI` 接口并适配现有 Wails generated bindings；不改变 request/response schema。 |
| `src/app.ts` | 拥有页面、主题、状态、配置、definition、选中项、busy/error 和轮询生命周期；串联用户动作与渲染。 |
| `src/view.ts` | 输出静态结构和语义化视图，绑定事件入口；所有后端/用户字符串通过 DOM text APIs 写入，避免把外部值拼进 HTML。 |
| `src/model.ts` | 保留并扩展无副作用的格式化、sample selection reconciliation、状态/时间显示和配置输入转换。 |
| `src/style.css` | Metrics 自有 Desktop-aligned token、三栏布局、两套主题、交互状态和响应式规则。 |
| `public/brand/` | 从 canonical `design-demos/brand` 复制经过确认的 MyFlowHub symbol；不运行时依赖 Desktop 目录。 |

### State and Data Flow

- 启动时并行读取 `Status` 与 `Definitions`，配置直接采用 `Status.configuration`；空闲 backend 会按 contract 拒绝独立 `Configuration` 调用，因此不把预期的 idle error 误报成初始化故障。任何真实失败都进入可见错误区，不使用假数据掩盖失败。
- 资源页按后端 sample 顺序稳定展示；刷新后优先保留仍存在的选中 metric，否则选择首项，没有 sample 时选择为空。
- `fresh`、`stale` 和 `unavailable` 使用文字、图标/形状与颜色共同表达；error、采样时间和观测时间保持可读。
- 轮询保持现有约 1 秒节奏，但使用单一可释放调度器和 in-flight guard，避免慢请求重叠；停止、卸载或 generation 变化后，旧响应不能覆盖新状态。
- 策略保存使用当前 revision 和既有 backend validation；成功后以返回配置替换本地状态，冲突或非法输入显示可操作错误。
- 身份和连接输入只在当前页面内存中存在；permit 等敏感字段不进入 local storage、日志或错误回显。
- 主题仅持久化 `light|dark`；损坏或未知值显式回到 light。导航和连接表单不持久化。

### Interface Composition

- 顶栏：Metrics 标识、连接摘要、主题切换与必要的当前运行反馈。
- 左栏：资源状态、采集策略、连接与身份三个互斥导航项；使用 button/ARIA current，不伪装链接。
- 资源状态：真实 sample 行、状态概览、空态/错误态；行可用指针、Tab 与 Enter/Space 选择。
- 上下文面板：仅展示当前 sample 的补充元数据；窄窗口隐藏时不承载唯一操作或唯一错误信息。
- 采集策略：enabled、writable、interval 和 notification channels；字段 label、帮助文本、校验和保存 busy 状态完整。
- 连接与身份：state directory、node ID、endpoint、parent node ID、permit，以及获取身份、连接、停止动作；公开密钥可展示和复制，秘密不回显。

### Responsive and Accessibility Rules

- 目标视口：1440×900 完整三栏；1024×768 收起补充 inspector，主内容无页面级横向溢出。
- 保留清晰的 `:focus-visible`、键盘导航、禁用/忙碌状态和 `aria-live` 错误/成功反馈。
- 支持 `prefers-reduced-motion`；主题切换不依赖动画才能理解。
- 不只依赖颜色表达连接、sample 或保存状态；所有核心按钮有可读名称和合理命中区域。

## Planned File Impact

| Path | Change |
| --- | --- |
| `apps/nodes/metrics/windows/frontend/src/main.ts` | 收敛为 bootstrap/dispose。 |
| `apps/nodes/metrics/windows/frontend/src/api.ts` | 增加 typed injectable adapter，保持现有 Wails contract。 |
| `apps/nodes/metrics/windows/frontend/src/app.ts` | 新增 UI state、动作、请求互斥与 polling lifecycle。 |
| `apps/nodes/metrics/windows/frontend/src/view.ts` | 新增语义化三页 shell、资源/inspector/表单渲染。 |
| `apps/nodes/metrics/windows/frontend/src/model.ts` | 扩展纯展示和输入转换函数。 |
| `apps/nodes/metrics/windows/frontend/src/style.css` | 改为独立 Desktop-aligned light/dark、三栏和 responsive 样式。 |
| `apps/nodes/metrics/windows/frontend/src/*.test.ts` | 增加 model 与 DOM 交互回归测试。 |
| `apps/nodes/metrics/windows/frontend/package.json`、`package-lock.json` | 仅在 DOM 测试需要时增加 `jsdom` dev dependency。 |
| `apps/nodes/metrics/windows/frontend/public/brand/myflowhub-symbol-v5.svg` | 新增 Metrics 自有 canonical symbol 副本。 |
| `apps/nodes/metrics/windows/frontend/dist/` | 由 production build 生成并审查。 |
| `docs/features/metrics-node.md` | 实现完成后补充 Windows UI 当前事实、状态与约束。 |

Backend Go、schema、generated Wails bindings、Desktop frontend 和 Android Metrics 不在预计写集内；只有验证发现 contract 已漂移时才暂停并重新规划，而不是顺手修改。

## Exact Task Table

### Will Execute After Approval

| ID | Deliverable | Dependencies | Completion evidence |
| --- | --- | --- | --- |
| MUI-01 | 建立 `MetricsAPI` 注入边界、纯展示模型和可释放 app bootstrap；保持 Wails schema 不变。 | none | TypeScript typecheck；model/API adapter tests。 |
| MUI-02 | 实现 Metrics 自有的 Desktop-aligned 顶栏、左导航、light/dark token、主题持久化和 canonical brand asset。 | MUI-01 | DOM tests 覆盖导航/主题；1440 与 1024 shell 截图。 |
| MUI-03 | 实现真实资源状态列表、稳定 selection、宽屏 inspector、空/stale/unavailable/error 状态及紧凑窗口行为。 | MUI-01, MUI-02 | model + DOM tests；无假 sample；窄屏核心能力不丢失。 |
| MUI-04 | 实现采集策略、连接与身份页面，并把保存、identity、start、stop、status polling、busy/error 全部接到既有 API。 | MUI-01, MUI-02 | fake API DOM tests 覆盖成功、失败、重复点击、revision 与 dispose。 |
| MUI-05 | 补齐无障碍、响应式和状态回归测试，审查敏感输入不持久化/不回显，处理发现的范围内缺陷。 | MUI-03, MUI-04 | `npm test` 全绿；键盘、ARIA、错误和 timer cases 有断言。 |
| MUI-06 | 更新 feature docs 与 tracked `dist/`，执行 frontend、Go、Wails production build 和真实 GUI 视觉/交互验收。 | MUI-05 | 下列 validation matrix 全绿；变更集仅含计划写集。 |

### Will Not Execute Now

| ID | Deferred work | Reason / future gate |
| --- | --- | --- |
| MUI-07 | 音量、亮度等即时本地控制。 | 当前没有 backend control contract；需要独立权限、安全和 API 决策。 |
| MUI-08 | 历史趋势、告警、日志或监控图表。 | 没有数据源与持久化 contract，且用户要求先保持现有能力。 |
| MUI-09 | Android Metrics 视觉对齐。 | 本次批准对象是 Windows Wails 界面；Android 需独立设计和验收。 |
| MUI-10 | Desktop/Metrics 组件共享、身份合并或私有 IPC。 | 违反已确认的独立产品边界。 |
| ARC-01 | change/plan/evidence 归档、lesson 判断、本地 merge 与 worktree cleanup。 | 由实现/可选重测试通过后的 `$m-archive` 执行，不属于下一次 `$m-execute`。 |
| PUB-01 | push、remote、release 或 publication。 | 用户未授权，当前仓库流程为 local-only。 |

## Execution Status

| ID | Status | Evidence |
| --- | --- | --- |
| MUI-01 | Changed / Passed | Injectable Wails adapter、纯 model、app bootstrap/dispose；typecheck 与 adapter/model tests 通过。 |
| MUI-02 | Changed / Passed | 独立 Desktop-aligned shell、light/dark theme、品牌资产与 1440/1024 production dist 截图。 |
| MUI-03 | Changed / Passed | 真实 sample、selection、inspector、空/stale/unavailable/error 与 responsive DOM tests 通过。 |
| MUI-04 | Changed / Passed | 策略、身份、启停、polling、busy/error 与 permit lifecycle fake-API tests 通过。 |
| MUI-05 | Changed / Passed | 12/12 frontend tests；敏感字段、timer dispose、键盘原生 button semantics 和输入校验已覆盖。 |
| MUI-06 | Passed with accepted residual | docs、tracked dist、`npm ci/test/build`、Metrics Go tests、Wails production build 和 production browser visual checks 通过；真实 WebView2 截图因 Windows/browser control kernel 初始化失败未取得。用户查看界面后显式调用 `$m-archive`，该证据缺口作为已知残余限制收口。 |

## Validation Matrix

1. Frontend dependency reproducibility: 在 `apps/nodes/metrics/windows/frontend` 执行 `npm ci`。
2. Frontend automated checks: `npm test`、`npm run build`；测试中使用 fake `MetricsAPI` 和 fake timers，不依赖真实 parent。
3. Go regression: 从 repo root 以 `GOWORK=off` 执行 `go test ./apps/nodes/metrics/...`。
4. Wails packaging: 在 `apps/nodes/metrics/windows` 执行 `wails build -clean`，确认生产可执行文件使用新 dist。
5. Real GUI smoke: 启动真实 Metrics Wails 应用，检查三页导航、theme、idle/empty/error、identity、连接表单、停止和保存策略；可用凭据缺失时不伪造 live sample 成功证据。
6. Visual acceptance: 1440×900 light resources、1440×900 dark policy/connection、1024×768 compact；检查页面级横向溢出、焦点、文本裁切和 console/page errors。
7. Change audit: `git diff --check`、`git status --short`、tracked dist diff；确认没有触碰主检出用户改动、Desktop/Android/backend contract 或秘密数据。

## Documentation Governance (`$m-docs`)

- Intake impact: 已新增并索引 `docs/intake/2026-08-30_metrics-desktop-aligned-interface.md`。
- Feature impact: MUI-06 更新 `docs/features/metrics-node.md`，只记录实现后可验证的 Windows UI 当前事实。
- Requirements impact: 无；不新增长期业务能力或 contract。
- Specs impact: 无；Wails/backend schema 预计不变。
- Decisions impact: 无；独立产品边界是已接受约束，本计划不重复创建 ADR。
- Lessons impact: 先复用 `docs/lessons/frontend-and-powershell-preflight.md`；只有实施出现可迁移新结论时才在 `$m-archive` 判断。
- Plan/change archive: 延后至 ARC-01；当前根 `plan.md` / `todo.md` 是唯一活动控制面。

## Risks and Stop Conditions

- 如果执行发现原型所需信息不在现有 `Status` / `Configuration` / `Definitions` 中，停止并请求重新规划，不构造模拟字段。
- 如果 Wails generated bindings 与 backend 实际接口不一致，先给出差异证据；未经批准不扩展 contract。
- 如果主检出或 worktree 出现与本任务重叠的用户修改，停止覆盖并请求协调。
- 如果真实 parent/permit 不可用，自动测试和 idle/error GUI 仍可完成；live connected evidence 标记为环境受限，不能被假成功替代。
- `package-lock.json` 只允许由明确的测试依赖变化产生；不做无关升级。

## Archive - Documentation And Closeout

- `$m-docs` impact review：新增并索引 intake；更新 Metrics feature；requirements、specs、decisions 无变化。
- Change record：`docs/change/2026-08-30_metrics-desktop-aligned-interface.md`。
- Plan snapshot：`docs/plan/plan_archive_2026-08-30_metrics-desktop-aligned-interface.md`。
- Lessons：不新增；本次工具链与 Wails 验证边界已由 `frontend-and-powershell-preflight.md` 覆盖。
- Production data boundary：资源值只来自 `Status.samples`；`充电中` 等是 boolean 展示映射，视觉验收 fixture 不进入生产 fallback。
- Closeout result：本地 `master` 已 fast-forward 到 `cde3ec9`；主检出 dirty 状态验证恢复，专用 worktree 与本地 feature branch 已移除。
- Publication：local-only；不 push、release 或 publish。

## Approval Gate

- Blocked: no。
- Approval: 用户已批准 `$m-execute`，范围为 MUI-01～MUI-06。
- Rollback point: `master` @ `54b46e6bc0c1bbea5b265a875e7c8132ae28e004`；所有实现只位于 `feat/metrics-desktop-ui` worktree。
- `$m-archive` 已由用户显式调用；归档、local merge 与 cleanup 已获授权，publication 仍未授权。
