# Win 设置页分组与 i18n Workflow Plan

## Workflow 信息

- 仓库：`MyFlowHub-Win`
- 分支：`feat/win-settings-i18n`
- Base：`main`
- Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n`
- 当前阶段：`4 归档变更`
- 状态：`T1 ~ T6 已完成，Code Review 通过，待用户确认是否结束本次 workflow`

## 当前状态

- 已完成独占分支与独占 worktree 创建。
- 已完成的实现结果：
  - `T1`：`app_settings.go` / `app_settings_test.go` 已新增全局语言设置模型、默认值、白名单校验与读写 API
  - `T2`：`frontend/src/i18n/**`、`frontend/src/stores/language.ts`、`frontend/src/main.ts` 已接入轻量 i18n 内核与启动语言镜像
  - `T3`：`frontend/src/layout/AppShell.vue` 已新增 `Other` 分组，`Settings` 已下沉到侧边栏底部
  - `T4`：`frontend/src/pages/Settings.vue` 已新增语言设置项，并与全局语言设置 API 联动
  - `T5`：已完成全站前端页面 / 组件 / window / store 的首批 i18n 迁移，覆盖 `Session / Operations / Signals / Flow / File / Showcase / Settings`
  - `T6`：已完成 `GOWORK=off go test ./... -count=1`、`frontend/npm run build`、Code Review 与归档
- 当前残余风险：
  - 尚未执行桌面运行态的人工 UI 冒烟，仅完成构建、测试与静态核对
  - `vite` 构建仍有既有 chunk-size warning，不影响本次功能正确性

## 1. 需求分析

### 目标

- 调整 Win 设置页与导航结构：
  - 将 `Settings` 导航入口移动到侧边栏最底部
  - 新增 `Other` 分组
  - 在设置页中新增语言选项
- 为 Win 前端建立可持续扩展的 i18n 基础能力

### 范围

- 必须做（已确认）
  - `Settings` 导航入口调整到最底部
  - 新增 `Other` 分组
  - 在设置页新增语言选项
  - 进行 i18n 相关整体改造
- 可选
  - 待澄清
- 不做
  - 待澄清

### 使用场景

- 用户希望在设置页中切换界面语言
- 用户希望设置页导航位置更稳定，固定在导航底部
- 后续新增页面时可以继续沿用统一的多语言文案机制

### 功能需求

- 侧边栏桌面端与移动端导航中，`Settings` 显示在最底部
- 设置页新增 `Other` 分组，并包含语言设置项
- 保存后语言设置按约定生效
- 前端文案改造为可通过 i18n 机制切换

### 非功能需求

- i18n 方案要尽量低耦合，避免后续每个页面重复造轮子
- 初次启动与语言切换的体验应可控，避免闪烁和大面积回归
- 保持当前风格，不做额外视觉体系重构

### 输入输出

- 输入：
  - 用户选择的语言
  - 用户现有设置项
- 输出：
  - 保存后的设置快照
  - 已应用的界面语言

### 边界异常

- 待澄清语言枚举、缺失翻译回退策略、语言切换生效时机

### 验收标准

- 待澄清 i18n 覆盖范围与首批支持语言后补充

### 风险

- 当前全站文案大量硬编码，若“整体支持 i18n”范围过大，会显著扩大改造面
- 若语言设置纳入 profile-scoped 设置，需要确认是否符合用户预期

### 阶段结论

- 阻塞：`否`
- 已确认：
  - 首批语言：`简体中文 + English`
  - 覆盖范围：`MyFlowHub-Win` 整个现有前端页面
  - 语言设置作用域：`全局应用级`

## 问题清单

 - 无

## 2. 架构设计

### 总体方案（采用）

- 采用“轻量自建 i18n 内核 + 全局语言设置 API + 全站文案迁移”的方案。
- 保留现有 profile-scoped `AppSettingsState` 用于连接默认值和 UI 偏好。
- 新增独立 `GlobalPreferencesState`，仅管理全局应用级语言，不随 profile 切换。
- 前端新增 `i18n` 模块与 `language` store：
  - `i18n/messages.ts` 维护 `zh-CN` / `en` 字典
  - `i18n/index.ts` 提供 `t()`、参数插值、回退逻辑、locale state
  - `stores/language.ts` 负责加载 / 保存全局语言设置，并在启动前应用
- `Settings` 页新增 `Other` 分组，语言项放在其中。
- `AppShell` 导航新增 `Other` 分组，并把 `Settings` 移动到最后一组、最后一个入口。

### 选型理由 / 备选对比

- 方案 A：引入 `vue-i18n`
  - 优点：生态成熟，复数/格式化能力完善
  - 缺点：当前仓库没有该依赖；本次以静态 UI 文案为主，引入额外依赖和接线成本偏高
- 方案 B：自建轻量 i18n 层
  - 优点：改造面可控、无新增运行时依赖、适合当前 hash-router + store 结构
  - 缺点：高级能力需要后续自行扩展
- 采用方案 B。

### 模块职责

- `app_settings.go`
  - 保持现有 profile-scoped 设置能力
  - 新增全局语言读取 / 保存 API
  - 对语言枚举做白名单校验与默认值回退
- `internal/storage/store.go`
  - 继续通过 `SetRaw/GetRaw` 持久化全局应用级语言 key
- `frontend/src/i18n/messages.ts`
  - 定义全部前端页面所需的中英文消息表
- `frontend/src/i18n/index.ts`
  - 定义 locale 类型、默认语言、字典查找、参数插值、缺失回退
  - 暴露响应式 `locale`、`setLocale`、`t`
- `frontend/src/stores/language.ts`
  - 调用 Go API 读写全局语言设置
  - 将语言镜像到 `localStorage`，用于启动前预应用
- `frontend/src/main.ts`
  - 在挂载前应用启动语言
  - 将 i18n 运行时注入 Vue
- `frontend/src/router/index.ts`
  - 将路由 `meta.title/subtitle` 改为 key，运行时翻译
- `frontend/src/layout/AppShell.vue`
  - 导航组改造为可翻译 key
  - `Settings` 放到底部 `Other` 分组
  - 页面标题/副标题、profile 菜单等接入 i18n
- `frontend/src/pages/**/*.vue` / `frontend/src/components/**/*.vue` / `frontend/src/stores/**/*.ts`
  - 将现有硬编码用户可见文案迁移为 `t(...)`
- `frontend/src/pages/Settings.vue`
  - 重组为 `Startup Defaults / Interface / About / Other`
  - `Other` 中新增语言选项

### 数据 / 调用流

- App 启动：
  - `main.ts` 先从 `localStorage` 读取语言镜像并设置 `locale`
  - 再应用现有 UI 偏好镜像，最后挂载 Vue
- 语言加载：
  - `language` store 调用 Go 全局 API 获取真实语言
  - 若与启动镜像不同，以后端值覆盖并同步镜像
- 语言切换：
  - `Settings` 页面修改语言 -> 点击保存
  - 前端先保存 profile-scoped 设置，再保存全局语言
  - 保存成功后更新 `locale`、写入 `localStorage`、整个界面即时切换
- 导航显示：
  - `AppShell` 通过 key -> `t()` 生成分组名、标签、描述
- 页面文案：
  - 各页面 / 组件 / toast 统一通过 `t()` 获取文案

### 接口草案

- Go
  - `GlobalPreferencesState() (GlobalPreferencesState, error)`
  - `SaveGlobalPreferencesState(state GlobalPreferencesState) (GlobalPreferencesState, error)`
- 前端
  - `useLanguageStore().load()`
  - `useLanguageStore().save(locale)`
  - `t(key, params?)`
  - `applyStartupLocale()`

### 错误与安全

- Go 侧对全局语言仅允许 `zh-CN` / `en`
- 存储未初始化返回显式错误
- 前端缺失 key 时回退到英文，再回退到 key 文本，避免空白 UI
- 语言保存失败时不切换运行时 locale，避免界面状态与持久化状态不一致

### 性能与测试策略

- 性能：
  - 词典静态内存对象，不在渲染期做异步加载
  - 启动镜像避免首屏先英文再中文的闪烁
  - `t()` 保持 O(depth) 的轻量 key 查找
- 测试：
  - Go：新增全局语言默认值、保存、非法值回退测试
  - Frontend：执行 `npm run build`
  - 集成：执行 `go test ./...`
  - UI：如环境允许，使用 `chrome-devtools` 冒烟验证语言切换和导航顺序

### 可扩展性设计点

- 词典按 key 分层命名，后续新增语言仅补消息表
- 语言设置与 profile 设置分离，避免跨 profile 干扰
- 路由 meta 使用 key 而不是最终文案，避免再改 router 结构时重复翻译接线

### 阶段结论

- 阻塞：`否`

## 3.1 计划拆分

### 项目目标与当前状态

- 目标：完成 Win 全站 i18n 基础设施，支持 `简体中文 + English`，并调整 Settings / 导航结构。
- 当前状态：需求、架构、计划、编码、Review 与归档已完成，待用户确认是否结束本次 workflow。

### 可执行任务清单（Checklist）

- [x] `T1` Go：新增全局语言设置模型与 API
- [x] `T2` Frontend Infra：新增 i18n 内核与语言 store，并接入启动流程
- [x] `T3` Shell / Router：导航重排、Settings 置底、路由元信息改为可翻译 key
- [x] `T4` Settings：新增 `Other` 分组与语言选项，保存时联动全局语言
- [x] `T5` Full Frontend Migration：迁移全站现有前端页面 / 组件 / store 用户可见文案
- [x] `T6` Verification：测试、构建、Review、归档

### 任务明细

- `T1`
  - 标题：Go 全局语言设置 API
  - Owner：`主Agent`
  - Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n`
  - Plan：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n\plan.md`
  - 目标：
    - 新增全局语言设置结构体与读写 API
    - 使用非 profile-scoped key 持久化语言
    - 完成默认值、校验、测试
  - 涉及模块 / 文件：
    - `app_settings.go`
    - `app_settings_test.go`
  - 验收条件：
    - API 可返回 `zh-CN/en`
    - 非法值会被回退
    - 不影响现有 profile-scoped 设置
  - 测试点：
    - 默认值
    - 保存合法值
    - 非法值回退
  - 回滚点：
    - 回退全局语言 API 与测试

- `T2`
  - 标题：前端 i18n 内核与语言 store
  - Owner：`主Agent`
  - Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n`
  - Plan：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n\plan.md`
  - 目标：
    - 新增消息表、翻译函数、响应式 locale
    - 新增语言 store，接入启动镜像与全局 API
  - 涉及模块 / 文件：
    - `frontend/src/i18n/**`
    - `frontend/src/stores/language.ts`
    - `frontend/src/main.ts`
  - 验收条件：
    - 启动时可预应用语言
    - 保存后可即时切换文案
  - 测试点：
    - 启动镜像
    - 缺失 key 回退
  - 回滚点：
    - 回退 i18n / language store / main 接线

- `T3`
  - 标题：Shell 与 Router i18n 化及导航结构调整
  - Owner：`主Agent`
  - Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n`
  - Plan：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n\plan.md`
  - 目标：
    - `Settings` 移到导航最底部
    - 新增 `Other` 分组
    - 路由 meta 改为 translation key
  - 涉及模块 / 文件：
    - `frontend/src/layout/AppShell.vue`
    - `frontend/src/router/index.ts`
  - 验收条件：
    - 桌面 / 移动导航顺序正确
    - 标题副标题可随语言切换
  - 测试点：
    - 导航排序
    - 设置项高亮
  - 回滚点：
    - 回退 shell / router 改动

- `T4`
  - 标题：Settings 页面结构调整与语言设置
  - Owner：`主Agent`
  - Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n`
  - Plan：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n\plan.md`
  - 目标：
    - 新增 `Other` 分组
    - 增加语言下拉项
    - 保存时同时处理 profile 设置与全局语言设置
  - 涉及模块 / 文件：
    - `frontend/src/pages/Settings.vue`
    - `frontend/src/stores/appSettings.ts`
    - `frontend/src/stores/language.ts`
  - 验收条件：
    - 语言项可编辑、保存、恢复默认
    - 其它设置语义不回归
  - 测试点：
    - 保存语言后即时切换
    - 切 profile 时语言保持不变
  - 回滚点：
    - 回退 settings 页面与语言接线

- `T5`
  - 标题：全站前端文案迁移
  - Owner：`主Agent（集成） + 子Agent（并行子任务）`
  - Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n`
  - Plan：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n\plan.md`
  - 目标：
    - 迁移整个现有前端页面的用户可见文案到 i18n key
  - 涉及模块 / 文件：
    - `frontend/src/pages/**`
    - `frontend/src/components/**`
    - `frontend/src/stores/**`
    - `frontend/src/layout/**`
  - 验收条件：
    - 英文与中文两套文案完整可切换
    - 无明显裸露 key / 空白文案
  - 测试点：
    - 关键页面切换语言
    - Toast / 对话 / 按钮文案切换
  - 回滚点：
    - 回退消息表和文案迁移
  - 风险与注意事项：
    - 改动面大，需分批核对，避免遗漏字符串
  - 子任务拆分：
    - `T5-A`
      - 标题：会话与运维页面迁移
      - Owner：`子Agent 019d113f-60e0-76a1-acec-0f6416b52664`
      - Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n`
      - Plan：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n\plan.md`
      - Write set：
        - `frontend/src/pages/Home.vue`
        - `frontend/src/pages/Devices.vue`
        - `frontend/src/pages/LocalHub.vue`
        - `frontend/src/pages/Debug.vue`
        - `frontend/src/pages/Logs.vue`
        - `frontend/src/pages/Permissions.vue`
        - `frontend/src/pages/Presets.vue`
        - `frontend/src/i18n/messages/session.ts`
        - `frontend/src/i18n/messages/operations.ts`
      - 当前结果：`已完成，子Agent已交付 Home / LocalHub / Permissions / Presets 与消息表收口，主Agent补齐剩余项并复核通过`
      - 验收条件：
        - 上述页面不出现裸露英文 key
        - `operations.ts` 补齐对应 `zh-CN` 文案
      - 测试点：
        - 页面标题、按钮、toast、错误提示语言可切换
      - 回滚点：
        - 回退上述页面及消息表
    - `T5-B`
      - 标题：信号台页面迁移
      - Owner：`子Agent 019d113f-75ad-7612-9eca-08ef89c78d91`
      - Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n`
      - Plan：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n\plan.md`
      - Write set：
        - `frontend/src/pages/TopicBus.vue`
        - `frontend/src/pages/VarPool.vue`
        - `frontend/src/i18n/messages/signals.ts`
      - 当前结果：`已完成，子Agent已交付 TopicBus / VarPool / signals.ts，主Agent复核通过`
      - 验收条件：
        - 页面主流程文案支持中英切换
      - 测试点：
        - tab、按钮、toast、空态、错误提示可切换
      - 回滚点：
        - 回退上述页面及消息表
- `T5-C`
  - 标题：剩余自动化、文件、展示台、窗口与 stores 迁移集成
  - Owner：`主Agent`
  - Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n`
  - Plan：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n\plan.md`
  - 当前结果：`已完成，主Agent完成 File / ModuleStub / windows / stores 收口并统一集成`
  - Write set：
        - `frontend/src/pages/File.vue`
        - `frontend/src/pages/ModuleStub.vue`
        - `frontend/src/components/file/OfferNodeTreePicker.vue`
        - `frontend/src/components/logs/LogItem.vue`
        - `frontend/src/components/ToastHost.vue`
        - `frontend/src/components/varpool/NodeVarsDialog.vue`
        - `frontend/src/windows/FileTasks.vue`
        - `frontend/src/windows/LogWindow.vue`
        - `frontend/src/windows/TopicBusWindow.vue`
        - `frontend/src/stores/file.ts`
        - `frontend/src/stores/logs.ts`
        - `frontend/src/stores/management.ts`
        - `frontend/src/stores/presets.ts`
        - `frontend/src/stores/profile.ts`
        - `frontend/src/stores/topicbus.ts`
        - `frontend/src/stores/varpool.ts`
        - `frontend/src/i18n/messages/file.ts`
        - `frontend/src/i18n/messages/stores.ts`
      - 验收条件：
        - 剩余页面 / 组件 / window / store 用户可见文案可切换
        - 各消息表不再为空
      - 测试点：
        - 窗口标题、对话框、提示信息、占位符、toast 切换正常
      - 回滚点：
        - 回退上述迁移文件
    - `T5-C1`
      - 标题：展示台页面与窗口迁移
      - Owner：`子Agent 019d1154-e13b-7321-b4f5-04b3bf422904 + 主Agent集成`
      - Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n`
      - Plan：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n\plan.md`
      - 当前结果：`已完成，子Agent完成 Showcase 页面 / store / 消息表迁移，主Agent复核并集成`
      - Write set：
        - `frontend/src/pages/Showcase.vue`
        - `frontend/src/pages/ShowcaseCenter.vue`
        - `frontend/src/windows/ShowcaseWindow.vue`
        - `frontend/src/windows/ShowcaseEditorWindow.vue`
        - `frontend/src/stores/showcase.ts`
        - `frontend/src/i18n/messages/showcase.ts`
      - 验收条件：
        - 展示台中心页、编辑器、展示窗口文案支持中英切换
      - 测试点：
        - screen 创建 / 复制 / 删除 / 打开窗口相关按钮与 toast
      - 回滚点：
        - 回退展示台相关页面、窗口、store、消息表
    - `T5-C2`
      - 标题：流程页面与编辑器迁移
      - Owner：`主Agent（集成收口）`
      - Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n`
      - Plan：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n\plan.md`
      - 当前结果：`已完成，主Agent补齐 Flow / FlowEditor / FlowNode / flow stores / automation.ts 的剩余 i18n 与消息等级链路`
      - Write set：
        - `frontend/src/pages/Flow.vue`
        - `frontend/src/components/flow/FlowCanvas.vue`
        - `frontend/src/components/flow/FlowNode.vue`
        - `frontend/src/windows/FlowEditorWindow.vue`
        - `frontend/src/stores/flow.ts`
        - `frontend/src/stores/flowProjects.ts`
        - `frontend/src/i18n/messages/automation.ts`
      - 验收条件：
        - 流程编辑主页面与独立窗口文案支持中英切换
      - 测试点：
        - 项目列表、节点面板、编辑器按钮、错误提示、toast 切换
      - 回滚点：
        - 回退流程页面、组件、window、store、消息表
    - `T5-C3`
      - 标题：预设页与 operations 消息表收口
      - Owner：`子Agent 019d113f-60e0-76a1-acec-0f6416b52664 + 主Agent集成`
      - Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n`
      - Plan：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n\plan.md`
      - 当前结果：`已完成，子Agent完成 placeholder 与消息表收口，主Agent统一复核`
      - Write set：
        - `frontend/src/pages/Presets.vue`
        - `frontend/src/stores/devices.ts`
        - `frontend/src/i18n/messages/operations.ts`
      - 验收条件：
        - Presets 页面主要表单、按钮、toast、错误提示支持中英切换
        - operations.ts 补齐 Devices / Permissions / Presets 相关 zh-CN 文案
      - 测试点：
        - 压测、认证、VarPool、TopicBus、Flow、Management、File 预设区块文案可切换
      - 回滚点：
        - 回退 Presets / devices store / operations 消息表

- `T6`
  - 标题：验证、Review 与归档
  - Owner：`主Agent`
  - Worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n`
  - Plan：`D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-i18n\plan.md`
  - 目标：
    - 执行测试 / 构建 / 冒烟
    - 完成 Code Review
    - 生成 `docs/change`
  - 涉及模块 / 文件：
    - 测试输出
    - `docs/change/YYYY-MM-DD_*.md`
  - 验收条件：
    - 测试通过
    - Review 通过
    - 归档完整
  - 执行结果：
    - `GOWORK=off go test ./... -count=1`：通过（2026-03-22）
    - `frontend/npm run build`：通过（2026-03-22，存在既有 chunk-size warning）
    - `3.3 Code Review`：通过
    - `4 归档变更`：已写入 `docs/change/2026-03-22_win-settings-i18n.md`

### 依赖关系

- `T2` 依赖 `T1`
- `T3` 依赖 `T2`
- `T4` 依赖 `T1`、`T2`
- `T5` 依赖 `T2`
- `T6` 依赖 `T1` ~ `T5`

### 并行性评估

- 已获得用户明确授权使用子Agent。
- 并行拆分结论：
  - `T5-A` 与 `T5-B` 写集不重叠，可并行执行
  - `T5-C` 与 `T5-A` 在 `Presets.vue`、`operations.ts` 上存在潜在冲突，因此由主Agent统一收口
  - `T6` 中的 Code Review 可在编码完成后进一步拆分为子Agent辅助审查，但最终结论仍由主Agent负责

### 阶段结论

- 计划已确认并执行完成
- 阻塞：`否`
- 已完成 `3.1`、`3.2`、`3.3`、`4`
- 待用户确认：`是否结束本次 workflow？`
