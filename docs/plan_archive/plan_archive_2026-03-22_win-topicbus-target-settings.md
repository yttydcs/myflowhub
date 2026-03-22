# Plan - MyFlowHub-Win：TopicBus 目标设置迁移与 Overview 单列化

## Workflow 信息
- 仓库：MyFlowHub-Win
- 分支：refactor/topicbus-target-settings
- Base：main
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-target-settings\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-target-settings\MyFlowHub-Win\todo.md
- 当前阶段：4 归档变更
- 状态：编码、验证、Code Review 与归档已完成，待用户确认是否结束本次 workflow
- 规范：
  - D:\project\MyFlowHub3\guide.md
  - D:\project\MyFlowHub3\AGENTS.md（会话内用户提供）

---

## 1) 需求分析

### 目标
- 将 TopicBus 的 `Target Node ID` 从 TopicBus 页面迁移到 `Settings`，做成真正可持久化的默认目标设置。
- 将 TopicBus `Overview` 从左右双栏收敛为单列布局。
- 让 TopicBus 主页面职责更清晰：
  - `Settings` 负责 TopicBus 偏好设置
  - `Overview` 负责状态摘要
  - `Channels` 负责主题列表、订阅管理和开窗

### 范围
#### 必须
- 调整 `app_topicbus.go`
  - 扩展 `TopicBusPrefs`，新增默认目标持久化字段。
- 调整 `frontend/src/stores/topicbus.ts`
  - 支持加载 / 保存默认 TopicBus 目标。
  - 保持无目标时自动回退到当前 `hubId` 的逻辑。
- 调整 `frontend/src/pages/Settings.vue`
  - 在已有 TopicBus 设置卡片中新增默认目标输入与保存动作。
- 调整 `frontend/src/pages/TopicBus.vue`
  - 删除 `Overview` 中的 `Target Node ID` 输入。
  - 删除 `Overview` 的左右双栏布局，改为单列状态页。
  - 将订阅管理入口收拢到 `Channels` tab。
- 保持 `frontend/src/windows/TopicBusWindow.vue`
  - 独立窗口仍可沿用当前目标逻辑。
  - 已打开窗口如带 route query `targetId`，优先使用 query 覆盖。

#### 可选
- 轻量收敛 TopicBus 文案，使 `Overview / Channels / Settings` 三处职责更直白。

#### 不做
- 不改 TopicBus 协议和后端服务调用方式。
- 不改 TopicBus 独立窗口的主收发分栏和右侧信息布局。
- 不新增节点选择器或管理树联动。

### 使用场景
- 用户平时默认对当前 Hub 操作，不希望在 TopicBus 页面每次都看到目标节点输入。
- 用户需要在 Settings 里统一维护 TopicBus 默认目标和缓存策略。
- 用户进入 TopicBus 页面时，希望先看到清晰状态，再到 Channels 中集中管理频道和开窗。

### 功能需求
- Settings 页面中的 TopicBus 卡片至少包含：
  - `Default Target Node ID`
  - `Max Events`
  - `Apply` 类保存动作
  - `Clear Cached`
- TopicBus `Overview`：
  - 改为单列。
  - 仅保留状态摘要 / 说明性内容，不再承担目标设置和主题编辑。
- TopicBus `Channels`：
  - 承载主题输入、保存/订阅、移除/退订、同步远端、重新订阅、开窗。
- 默认目标为空时：
  - 自动回退到当前 `hubId`。
- 独立窗口：
  - 沿用设置中的默认目标。
  - 若由 URL 显式携带 `targetId`，则该窗口实例优先使用它。

### 非功能需求
- 简洁：Overview 页面避免双栏分散注意力。
- 一致性：默认目标与缓存上限同属 TopicBus 偏好，应统一进入 Settings。
- 可维护性：复用现有 `TopicBusPrefs`、`topicbus` store 和现有独立窗口 query 覆盖逻辑。
- 可扩展性：为后续 TopicBus 更多设置项预留 Settings 卡片空间。
- 稳定性：不破坏当前订阅、退订、远端同步和独立窗口收发行为。

### 输入输出
- 输入：
  - Settings 页面中的默认目标输入
  - 频道页中的主题批量输入
  - 独立窗口 route query `targetId`
- 输出：
  - 持久化的 TopicBus 默认目标
  - 单列化的 Overview
  - 更聚焦的 Channels 管理区

### 边界异常
- 默认目标为空或非法时，仍需回退到 `hubId` 或给出校验错误。
- profile 切换后，Settings 与 TopicBus 页面都要重新加载对应 profile 的默认目标与 topics。
- 旧配置没有目标字段时，必须兼容，不应导致 TopicBus 页面报错。

### 验收标准
1. `Settings` 页面可查看和保存 TopicBus 默认目标。
2. `TopicBus` 页面不再显示 `Target Node ID` 输入。
3. `Overview` 改为单列，不再是左右两栏。
4. 订阅管理入口集中在 `Channels` tab。
5. 打开的 TopicBus 独立窗口仍能按默认目标或 query 目标正常发送 / 接收。
6. 前端构建与 Go 测试通过。

### 风险
- 若仅迁移 UI，不同步扩展 `TopicBusPrefs`，会导致“看起来是设置，实际不保存”的误导。
- 若 `targetId` 的默认回退逻辑处理不当，可能影响独立窗口或订阅请求的目标解析。

### 问题清单
- 阻塞：否

---

## 2) 架构设计（分析）

### 总体方案
- 后端偏好层：
  - 在 `app_topicbus.go` 的 `TopicBusPrefs` 中新增默认目标字段，并通过 `store.GetInt/SetInt` 以 profile 维度持久化。
- 前端状态层：
  - `frontend/src/stores/topicbus.ts` 继续以 `state.targetId` 作为当前默认目标。
  - `loadPrefs()` 负责把持久化目标加载到 `state.targetId`。
  - `savePrefs()` 一并保存 `topics/maxEvents/targetId`。
  - `setIdentity()` 仅在未设置默认目标时回退到 `hubId`。
- 页面层：
  - `Settings.vue` 维护 TopicBus 默认目标 draft。
  - `TopicBus.vue` 删除目标输入，Overview 只做摘要，Channels 承接主题与订阅管理。
  - `TopicBusWindow.vue` 保持 query `targetId` 优先级高于默认设置。

### 模块职责
- `app_topicbus.go`
  - 负责 TopicBus prefs 的读取、归一化和持久化。
- `frontend/src/stores/topicbus.ts`
  - 负责默认目标解析、TopicBus prefs 同步、订阅和发布调用。
- `frontend/src/pages/Settings.vue`
  - 承载 TopicBus 默认目标和缓存上限设置。
- `frontend/src/pages/TopicBus.vue`
  - 承载 TopicBus 状态总览和频道订阅管理。
- `frontend/src/windows/TopicBusWindow.vue`
  - 继续作为收发工作区；默认目标来自 prefs，必要时被 route query 覆盖。

### 数据 / 调用流
- Settings 页面加载：
  - `topicbus.loadPrefs()`
  - 同步 `targetId/maxEvents` 到本页 draft
- 用户保存 TopicBus 设置：
  - 校验默认目标
  - 更新 `topicbus.state.targetId`
  - 调用 `SaveTopicBusPrefs`
- TopicBus 页面加载：
  - `topicbus.loadPrefs()`
  - `topicbus.setIdentity(selfNodeId, hubId)`
  - 若 prefs 中无目标，则使用 `hubId`
- 独立窗口加载：
  - `topicbus.loadPrefs()`
  - `setIdentity(...)`
  - 若 URL 含 `targetId`，覆盖当前窗口实例的 `state.targetId`

### 接口草案
- `TopicBusPrefs`
  - 新增 `TargetID int` 或等价字段，表示默认目标节点。
- `topicbus` store
  - 复用 `loadPrefs()` / `savePrefs()` / `resolveTargetId()`
  - 新增或内聚默认目标保存辅助逻辑

### 错误与安全
- 默认目标输入允许空值，表示使用当前 Hub。
- 非空时必须为正整数。
- `resolveTargetId()` 继续作为统一目标解析入口，避免多处分散解析。

### 性能与测试策略
- 性能：
  - 仅增加一次 prefs 字段读写，不引入额外高频计算。
  - Overview 单列化只改模板结构，无额外数据负担。
- 测试：
  - `cd frontend && npm run build`
  - `GOWORK=off go test ./... -count=1 -p 1`
  - 如可行，用 `chrome-devtools` 做一次 TopicBus 页面与 Settings 页的基础冒烟

### 可扩展性设计点
- TopicBus 设置卡片后续可继续放：
  - 默认目标
  - 缓存策略
  - 其他窗口级默认行为
- `resolveTargetId()` 作为唯一目标解析入口，便于未来加入“空值=Hub”以外的策略。

### 问题清单
- 阻塞：否

---

## 3.1) 可执行任务清单（Checklist）

### TTSP-1：扩展 TopicBus prefs，持久化默认目标
- 状态：已完成
- Owner：主Agent
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-target-settings\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-target-settings\MyFlowHub-Win\todo.md
- 目标：
  - 在 TopicBus prefs 中新增默认目标字段并接入前端 store
- 涉及模块 / 文件：
  - `app_topicbus.go`
  - `frontend/src/stores/topicbus.ts`
- 验收条件：
  - `topicbus.loadPrefs()` 能读出默认目标
  - `topicbus.savePrefs()` 能保存默认目标
  - 无目标时仍回退 `hubId`
- 测试点：
  - Go 测试
  - Settings 页面刷新后默认目标不丢失
- 回滚点：
  - 回退上述两个文件
- Write set：
  - `app_topicbus.go`
  - `frontend/src/stores/topicbus.ts`

### TTSP-2：收敛 Settings 中的 TopicBus 设置区
- 状态：已完成
- Owner：主Agent
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-target-settings\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-target-settings\MyFlowHub-Win\todo.md
- 目标：
  - 在 Settings 中新增默认目标输入，并与缓存设置一起管理
- 涉及模块 / 文件：
  - `frontend/src/pages/Settings.vue`
- 验收条件：
  - 用户可设置默认目标
  - 保存和错误提示明确
- 测试点：
  - 前端构建
  - profile 切换后显示同步
- 回滚点：
  - 回退 `frontend/src/pages/Settings.vue`
- Write set：
  - `frontend/src/pages/Settings.vue`

### TTSP-3：重构 TopicBus 页面结构
- 状态：已完成
- Owner：主Agent
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-target-settings\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-target-settings\MyFlowHub-Win\todo.md
- 目标：
  - 移除 TopicBus 主页面中的目标输入
  - 将 Overview 改为单列
  - 将订阅管理入口集中到 Channels tab
- 涉及模块 / 文件：
  - `frontend/src/pages/TopicBus.vue`
- 验收条件：
  - Overview 单列化完成
  - Channels 可完成保存/订阅/退订/同步/开窗
- 测试点：
  - 前端构建
  - TopicBus 页面手工冒烟
- 回滚点：
  - 回退 `frontend/src/pages/TopicBus.vue`
- Write set：
  - `frontend/src/pages/TopicBus.vue`

### TTSP-4：兼容独立窗口与集成验证
- 状态：已完成
- Owner：主Agent
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-target-settings\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-target-settings\MyFlowHub-Win\todo.md
- 目标：
  - 确认独立窗口继续使用默认目标 / query 覆盖
  - 完成构建、测试、Review、归档
- 涉及模块 / 文件：
  - `frontend/src/windows/TopicBusWindow.vue`
  - `docs/change/YYYY-MM-DD_topicbus-target-settings.md`
- 验收条件：
  - TopicBus 窗口不回退
  - build/test 通过
  - 归档完整
- 测试点：
  - `cd frontend && npm run build`
  - `GOWORK=off go test ./... -count=1 -p 1`
- 回滚点：
  - 回退本 workflow 全部改动
- Write set：
  - `frontend/src/windows/TopicBusWindow.vue`
  - `docs/change/`

---

## 3.2) 实施记录

### TTSP-1：扩展 TopicBus prefs，持久化默认目标
- 完成内容：
  - `app_topicbus.go` 新增 `topicbus.target_id` 持久化键和 `TopicBusPrefs.TargetID`。
  - `frontend/src/stores/topicbus.ts` 支持 `targetId` 的加载、保存、归一化和 `hubId` 回退。
  - `app_topicbus_test.go` 新增默认值、持久化、非法值归一化测试。
- 关键设计点：
  - 空目标表示“使用当前 Hub”，不把 `hubId` 反写成已配置值。
  - `resolveTargetId()` 继续作为统一目标解析入口，减少页面层分散处理。
- 验收结果：
  - Go 测试通过。

### TTSP-2：收敛 Settings 中的 TopicBus 设置区
- 完成内容：
  - `frontend/src/pages/Settings.vue` 新增 `Default Target Node ID` 与 `Max Events` 统一保存入口。
  - 保存前对目标节点和缓存上限做正整数校验；失败时回滚到持久化值。
- 关键设计点：
  - 将 TopicBus 偏好收口到 `Settings`，避免功能页承担配置职责。
  - `Clear Cached` 保留在同一卡片，便于用户在设置区统一处理缓存策略。
- 验收结果：
  - 前端构建通过。

### TTSP-3：重构 TopicBus 页面结构
- 完成内容：
  - `frontend/src/pages/TopicBus.vue` 的 `Overview` 改为单列状态摘要。
  - `Channels` 集中承载主题输入、保存/订阅、移除/退订、远端同步、重订阅和开窗。
  - 移除 `Overview` 上的目标输入和多余快捷入口。
- 关键设计点：
  - 保留必要状态信息，但把可操作内容集中到 `Channels`，降低页面跳转心智负担。
  - “All” 入口固定展示，便于聚合接收所有已知频道。
- 验收结果：
  - 前端构建通过。

### TTSP-4：兼容独立窗口与集成验证
- 完成内容：
  - `frontend/src/windows/TopicBusWindow.vue` 继续支持 route query `targetId` 覆盖默认目标。
  - 独立窗口发送区移除 `Target Node ID` 输入，目标来源收口为 Settings 默认值或开窗链接覆盖。
  - 对窗口 query `targetId` 新增正整数校验，非法值直接忽略。
- 关键设计点：
  - 独立窗口保留“上收下发 + 右侧信息/控制”结构，但减少冗余配置入口。
  - 新窗口只接收打开后的新事件，仍保持原有监听语义。
- 验收结果：
  - `$env:GOWORK='off'; go test ./... -count=1 -p 1` 通过。
  - `npm ci`、`$env:GOWORK='off'; wails generate module`、`npm run build` 通过。

---

## 3.3) Code Review

- 需求覆盖：通过。默认目标已迁移到 `Settings` 持久化；`Overview` 已单列化；`Channels` 已承接主题与订阅管理；独立窗口继续支持默认目标与 query 覆盖。
- 架构合理性：通过。后端 prefs、前端 store、Settings、TopicBus、独立窗口职责边界清晰，没有把目标解析逻辑散落到多个页面。
- 性能风险：通过。本次只增加一次轻量 prefs 字段读写；频道聚合和事件统计仍基于已有内存数据，无新增高频 I/O、重复订阅或明显重复计算风险。
- 可读性与一致性：通过。命名与现有 TopicBus 语义一致；页面职责与文案更直白；目标解析和输入校验集中在 store/页面入口。
- 可扩展性与配置化：通过。TopicBus 偏好统一收口到 `Settings`，便于后续继续追加窗口级默认行为；`resolveTargetId()` 仍是唯一扩展点。
- 稳定性与安全：通过。Settings 保存与窗口 query 都做了正整数校验；空目标保持回退到当前 `hubId`；独立窗口不会因为坏 query 污染默认目标。
- 测试覆盖情况：通过。新增 Go 单元测试覆盖 prefs 默认值、持久化和非法值归一化；前端完成依赖安装、Wails 生成和生产构建验证。
- 子Agent治理与审计：通过。本 workflow 未使用子Agent，原因是用户未显式授权，且写集集中在同一组 TopicBus 文件，主Agent直接收敛更安全。

### Review 结论
- 结论：通过
- 阻塞：否

---

## 4) 归档变更

- 状态：已完成
- 归档文档：
  - `D:\project\MyFlowHub3\worktrees\topicbus-target-settings\MyFlowHub-Win\docs\change\2026-03-22_topicbus-target-settings.md`
- 归档说明：
  - 已记录任务映射、设计决策、验证结果、潜在影响与回滚方案。
- 问题清单：
  - 阻塞：否

---

## 附录) 并行性评估（供 3.2 使用）

### 结论
- 本次不使用子Agent。

### 原因
- 当前平台规则下，未获用户显式授权不派发子Agent。
- 本次改动横跨 prefs、store、Settings、TopicBus 页面，写集高度相关，主Agent直接收敛更安全。

### 问题清单
- 阻塞：否
