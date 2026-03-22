# 2026-03-22_win-settings-i18n

## 变更背景 / 目标
- 背景：
  - Win 端原有设置能力集中在连接默认值与界面偏好，缺少语言设置入口，也没有独立的全局语言持久化模型。
  - 侧边栏缺少 `Other` 一级分组，`Settings` 不在导航最底部。
  - 现有前端页面存在大量硬编码用户可见文案，无法支持整站语言切换。
- 目标：
  - 调整侧边栏结构，在底部新增 `Other` 分组并将 `Settings` 下沉到最底部。
  - 扩展设置页，纳入默认地址、默认设备 ID、自动连接、自动登录、界面偏好、语言与 About 信息。
  - 新增全局应用级语言设置，并完成整个现有 Win 前端页面的首批 i18n 改造。

## 具体变更内容（新增 / 修改 / 删除）

### 新增
- `app_settings.go`
  - 新增 `GlobalPreferencesState`
  - 新增 `GlobalPreferencesState()` / `SaveGlobalPreferencesState(...)` / `ResetGlobalPreferencesState()`
  - 为全局语言设置增加默认值、白名单校验、持久化与回退逻辑
- `app_settings_test.go`
  - 新增全局语言默认值、合法保存、非法值回退、重置等测试
- `frontend/src/i18n/**`
  - 新增轻量 i18n 运行时
  - 新增 `zh-CN` 消息表分模块组织：`common / shell / settings / session / operations / signals / automation / file / showcase / stores`
- `frontend/src/stores/language.ts`
  - 新增全局语言 store
  - 新增启动镜像 `localStorage` 读写
- `docs/change/2026-03-22_win-settings-i18n.md`

### 修改
- `frontend/src/main.ts`
  - 启动前先应用语言镜像，避免首屏语言闪烁
- `frontend/src/layout/AppShell.vue`
  - 新增 `Other` 分组
  - `Settings` 移动到侧边栏最底部
  - 导航、标题、副标题、连接状态、弹窗提示接入 `t(...)`
- `frontend/src/pages/Settings.vue`
  - 页面重组为 `Startup Defaults / Interface / About / Other`
  - 新增语言选项
  - 保存时同时持久化 profile-scoped 设置与全局语言设置，并即时应用
- `frontend/src/pages/**`
  - `Home / Devices / LocalHub / Debug / Logs / Permissions / Presets / File / Flow / Showcase / ShowcaseCenter / TopicBus / VarPool / Settings / ModuleStub`
  - 页面级按钮、标题、说明、placeholder、toast、confirm 等用户可见文案统一接入 i18n
- `frontend/src/components/**`
  - `ToastHost / OfferNodeTreePicker / FlowNode / LogItem / NodeVarsDialog`
  - 组件内的用户可见文案、状态标签与提示接入 i18n
- `frontend/src/windows/**`
  - `FileTasks / FlowEditorWindow / LogWindow / ShowcaseWindow / TopicBusWindow`
  - 独立窗口标题、按钮、提示文案接入 i18n
- `frontend/src/stores/**`
  - `devices / file / flow / flowProjects / logs / management / presets / profile / showcase / toast / topicbus / varpool`
  - store 层用户可见错误、状态、toast 文案改为 `t(...)`
  - `flow.ts` 引入 `messageLevel` 与状态 key 映射，移除基于英文子串判断消息级别的耦合逻辑
- `frontend/wailsjs/**`
  - 通过 `wails generate module` 刷新生成绑定，保证前端构建可用

### 删除
- 无功能删除

## 对应 `plan.md` 任务映射
- `T1`
  - `app_settings.go`
  - `app_settings_test.go`
- `T2`
  - `frontend/src/i18n/**`
  - `frontend/src/stores/language.ts`
  - `frontend/src/main.ts`
- `T3`
  - `frontend/src/layout/AppShell.vue`
- `T4`
  - `frontend/src/pages/Settings.vue`
  - `frontend/src/stores/language.ts`
- `T5`
  - `frontend/src/pages/**`
  - `frontend/src/components/**`
  - `frontend/src/windows/**`
  - `frontend/src/stores/**`
  - `frontend/src/i18n/messages/**`
- `T6`
  - `GOWORK=off go test ./... -count=1`
  - `GOWORK=off wails generate module`
  - `frontend/npm run build`
  - `3.3 Code Review`
  - `docs/change/2026-03-22_win-settings-i18n.md`

## 关键设计决策与权衡（尤其性能 / 扩展性）
- 决策：采用轻量自建 i18n，而不是引入 `vue-i18n`
  - 原因：本次以静态 UI 文案和运行态切换为主，仓库当前没有该依赖，自建方案改造面更可控。
  - 权衡：高级格式化能力后续若有需要可继续扩展；当前先保证低耦合与落地效率。

- 决策：语言设置采用全局应用级模型，与 profile-scoped `AppSettingsState` 分离
  - 原因：用户已确认语言作用域为整个应用，而不是随 profile 切换。
  - 权衡：连接默认值和 UI 偏好继续随 profile 存储；语言单独存储，避免切 profile 时语言抖动。

- 决策：前端增加启动镜像 `localStorage`
  - 原因：应用启动时先用镜像语言设置 `document.documentElement.lang` 与运行时 locale，可避免“首屏先英文再中文”的闪烁。
  - 权衡：后端存储仍是最终真源；镜像只用于启动体验优化。

- 决策：缺失翻译按 `当前语言 -> 英文 -> key` 回退
  - 原因：避免界面出现空白，同时保留排查能力。
  - 权衡：极端情况下仍可能出现英文或 key，但不会阻断主流程。

- 决策：Flow 编辑器 toast 级别改为显式 `messageLevel`
  - 原因：原逻辑依赖英文文案子串判断成功/错误，在 i18n 后不可靠。
  - 权衡：store 多维护一个显式字段，但解除语言文本与业务判断的耦合。

## 测试与验证方式 / 结果
- `GOWORK=off go test ./... -count=1`
  - 结果：通过
- `GOWORK=off wails generate module`
  - 结果：通过
  - 说明：用于补齐 `frontend/wailsjs/**` 生成绑定，消除构建缺失依赖
- `frontend/npm run build`
  - 结果：通过
  - 备注：保留既有 chunk-size warning，不影响本次功能正确性
- i18n key 覆盖静态核对
  - 结果：工作流内未发现真实缺失 key
- 启发式裸露英文扫描
  - 结果：剩余命中主要为技术标识、品牌字样或 `t(...)` 中的源 key

## 3.3 Code Review 结论
- 需求覆盖：通过
  - 设置页、Other 分组、Settings 置底、语言全局生效、保存后应用、About 展示与整站 i18n 均已覆盖
- 架构合理性：通过
  - Go 全局语言状态、前端 i18n 运行时与语言 store 分层清晰，profile 设置与全局设置边界明确
- 性能风险：通过
  - 词典为静态内存对象，`t()` 为同步查找；未引入额外网络往返或运行态重复 I/O
- 可读性与一致性：通过
  - 文案入口统一为 `t(...)`，消息表按域拆分；Flow 侧显式 `messageLevel` 提升了可读性
- 可扩展性与配置化：通过
  - 新增语言只需扩展消息表与白名单；语言与 profile 解耦便于未来扩展更多全局偏好
- 稳定性与安全：通过
  - Go 侧做语言白名单与默认值回退；前端保存失败时不会切换运行时 locale，避免状态不一致
- 测试覆盖情况：通过
  - Go 测试与前端构建通过；已完成生成绑定与静态核对
- 子Agent治理与审计：通过
  - 3.2 阶段按 Task ID 和 write set 拆分，主Agent统一集成与复核；辅助审查超时结果未采纳，不影响最终审查责任归属
- 结论：
  - Review 通过，无阻塞性缺陷
  - 残余风险：未做桌面运行态人工 UI 冒烟；构建仍有既有 chunk-size warning

## 潜在影响与回滚方案

### 潜在影响
- 整个现有前端页面的用户可见文案已统一经过 i18n 运行时，后续新增文案若未补 key，将按英文或 key 回退显示。
- 语言设置改为全局应用级后，切换 profile 不会再切换语言；这是本次需求确认后的预期行为。
- 前端构建依赖 `frontend/wailsjs/**` 生成绑定；若后续接口签名继续变化，需要重新执行生成步骤。

### 回滚方案
- 以任务维度回滚：
  - 回退 `T1`：`app_settings.go`、`app_settings_test.go`
  - 回退 `T2`：`frontend/src/i18n/**`、`frontend/src/stores/language.ts`、`frontend/src/main.ts`
  - 回退 `T3` / `T4`：`frontend/src/layout/AppShell.vue`、`frontend/src/pages/Settings.vue`
  - 回退 `T5`：页面 / 组件 / window / store / 消息表的 i18n 迁移文件
  - 如需回到改造前构建状态，可同时回退本次刷新后的 `frontend/wailsjs/**`

## 子Agent执行轨迹（Task ID → Agent → Worktree → 文件 → 验收结果）
- `T5-A` → `Ampere (019d113f-60e0-76a1-acec-0f6416b52664)` → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n` → `frontend/src/pages/Home.vue`、`frontend/src/pages/LocalHub.vue`、`frontend/src/pages/Permissions.vue`、`frontend/src/pages/Presets.vue`、`frontend/src/i18n/messages/session.ts`、`frontend/src/i18n/messages/operations.ts` → 已采纳，主Agent补齐剩余项并复核通过
- `T5-B` → `Maxwell (019d113f-75ad-7612-9eca-08ef89c78d91)` → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n` → `frontend/src/pages/TopicBus.vue`、`frontend/src/pages/VarPool.vue`、`frontend/src/i18n/messages/signals.ts` → 已采纳，主Agent复核通过
- `T5-C1` → `Heisenberg (019d1154-e13b-7321-b4f5-04b3bf422904)` → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n` → `frontend/src/pages/Showcase.vue`、`frontend/src/stores/showcase.ts`、`frontend/src/i18n/messages/showcase.ts` → 已采纳，主Agent复核通过
- `T5-C2` → `Ramanujan (019d1154-f560-7ff3-a241-1ac16426d9cf)` → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n` → `frontend/src/pages/Flow.vue`、`frontend/src/i18n/messages/automation.ts` → 中间检查结果提示未完成；最终未直接采纳，主Agent后续完成 `FlowEditorWindow.vue`、`FlowNode.vue`、`flow.ts`、`flowProjects.ts` 收口并重新验证
- `T6-Review-Assist` → `Goodall (019d1194-87c3-74e2-b7bf-e781f1fbf0ff)` → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n` → 只读审查任务 → 超时未返回结果，最终 Review 结论由主Agent独立完成
