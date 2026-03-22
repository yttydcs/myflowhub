# Plan - MyFlowHub-Win：Showcase Center + 独立编辑窗口重构

## 项目目标与当前状态

### 背景
- 仓库：`MyFlowHub-Win`
- 目标：将当前 Showcase 从“单页混合管理/编辑/运行”重构为：
  - `Showcase Center`：列表与管理中心
  - `Showcase Editor Window`：独立编辑窗口
  - `Showcase Viewer Window`：独立展示/运行窗口（保留现有能力）

### 当前状态
- 当前 `frontend/src/pages/Showcase.vue` 同时承担：
  - Screen 列表管理
  - Screen layout 编辑
  - Widget 创建/编辑/删除
  - Widget 拖拽排序
  - Canvas 自由布局编辑
  - 运行态交互（TopicBus / VarPool）
- 当前 `frontend/src/windows/ShowcaseWindow.vue` 已存在，但定位为 Viewer，而非 Editor。
- 当前 `frontend/src/stores/showcase.ts` 同时承载：
  - 配置读取/保存
  - 订阅生命周期
  - TopicBus 发送
  - VarPool 控制
- 当前 `ShowcaseScreen` schema 无 `updatedAt` 字段，无法稳定支撑列表页“最近更新时间”摘要。

### 本次确认的需求结论
- 独立编辑界面采用“独立窗口”，对齐 Flow 的交互方式。
- 保留现有 `showcase-window` Viewer。
- 列表页展示字段：
  - 名称
  - 布局类型
  - widget 数量
  - 最近更新时间
  - 操作按钮
- 编辑器内允许直接操作 widget，不是纯配置页。
- 第一版保留两种布局：
  - `columns`
  - `canvas_percent`
- 第一版支持：
  - Blank 新建
  - Duplicate Existing（复制 screen）
- 保存策略：
  - 手动保存
  - dirty 提示
  - 关闭前确认
- 多窗口同时编辑同一 screen：
  - V1 接受“最后保存者生效”
  - 不做锁，不做冲突合并

---

## Workflow 信息
- Repo：`d:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch：`refactor/showcase-center-editor`
- Base：`main`
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-showcase-center-editor`
- Plan Path：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-showcase-center-editor\plan.md`
- 当前阶段：`4`
- 规范：
  - `d:\project\MyFlowHub3\guide.md`
  - `d:\project\MyFlowHub3\AGENTS.md`

### 主仓状态说明
- `repo/MyFlowHub-Win` 当前存在未提交改动（非本 workflow 所有权）。
- 本 workflow 的全部实现性改动必须仅发生在当前 worktree 中。

---

## 需求分析摘要（阶段 1 结论）

### 目标
- 降低 Showcase 的页面复杂度与认知负担。
- 让主页面回归“列表/管理中心”，不再承担重编辑职责。
- 通过独立编辑窗口承载复杂布局与 widget 编辑。
- 保留 Viewer 作为独立展示/运行面板。

### 范围
- 必须：
  - `Showcase Center` 列表页
  - `Showcase Editor Window`
  - 保留并兼容 `Showcase Viewer Window`
  - `updatedAt` 持久化
  - 手动保存与 dirty 提示
  - Blank / Duplicate 创建入口
  - `columns` / `canvas_percent` 均可继续编辑与运行
- 可选：
  - 将创建入口组织为“Blank / Duplicate Existing”
- 不做：
  - 服务端协议调整
  - 并发编辑冲突合并
  - 通用模板系统

### 验收原则
- 列表页不再承载重型编辑逻辑。
- `Edit` 打开独立编辑窗口。
- `View` 打开独立 Viewer。
- 编辑器可直接操作 widget。
- 两种布局能力不回退。

---

## 架构设计摘要（阶段 2 结论）

### 总体方案
- `Showcase Center`
  - 只负责 screen 列表与管理动作
- `Showcase Editor Window`
  - 单 screen 独立编辑
  - 三栏结构：
    - 左侧：结构/图层列表
    - 中间：实时预览
    - 右侧：属性面板
- `Showcase Viewer Window`
  - 固定 screenId 渲染
  - 继续接收 `showcase.config_changed` 自动更新

### 关键设计点
- `ShowcaseScreen` 增加 `updatedAt`
  - 支撑列表摘要、排序与后续扩展
- 保存策略由“编辑即保存”转为“本地草稿 + 手动保存”
  - 避免编辑过程持续广播 `showcase.config_changed`
  - 降低 Viewer 被频繁刷新导致的干扰
- 运行态交互保留在编辑器内
  - 但订阅范围仅限当前活动 screen，避免多余 I/O
- Viewer 继续作为独立运行面板
  - 保持职责单一

### 关键风险
- `frontend/src/stores/showcase.ts` 当前职责较重，若不补充清晰的 draft/save 边界，重构后仍会高耦合。
- `canvas_percent` 编辑器要求较高，需要避免把复杂度重新堆回主列表页。
- 多窗口编辑同一 screen 采用“最后保存者生效”，需在 UI 层明确提示该行为。

---

## 可执行任务清单（Checklist）

### SC-01 - 配置模型与 Store 基础重构
- Status：已完成
- Owner：主Agent
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-showcase-center-editor`
- Plan：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-showcase-center-editor\plan.md`
- Write set：
  - `app_showcase.go`
  - `app_showcase_test.go`
  - `frontend/src/stores/showcase.ts`
- 关键上下文引用：
  - `app_showcase.go`
  - `frontend/src/stores/showcase.ts`
  - `frontend/src/windows/ShowcaseWindow.vue`
- 目标：
  - 为 `ShowcaseScreen` 增加 `updatedAt`
  - 保持旧配置兼容
  - 在 store 中补齐 screen 复制、screen 元信息更新、editor draft/save 所需接口
  - 明确“加载配置”和“提交保存”的边界，支撑手动保存策略
- 涉及模块 / 文件：
  - Go Showcase 配置 schema 与 normalize
  - 前端 Showcase store
- 验收条件：
  - 旧 profile 配置可正常加载
  - screen 缺失 `updatedAt` 时自动补齐
  - store 能稳定支持：
    - 读取 screen 列表摘要
    - duplicate screen
    - 更新 screen 名称 / 更新时间
    - 从 editor draft 显式保存
  - 运行订阅与发送行为不回退
- 测试点：
  - `go test ./... -count=1 -p 1`
  - 针对 normalize / `updatedAt` / duplicate 的单测或最小回归
- 回滚点：
  - 回滚 `app_showcase.go`
  - 回滚 `app_showcase_test.go`
  - 回滚 `frontend/src/stores/showcase.ts`
- 依赖关系：
  - 无
- 风险与注意事项：
  - 不得破坏现有 `showcase.config_changed` 行为
  - 不得引入对旧配置的破坏性 schema 不兼容

---

### SC-02 - Showcase Center 列表页重构
- Status：已完成
- Owner：主Agent
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-showcase-center-editor`
- Plan：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-showcase-center-editor\plan.md`
- Write set：
  - `frontend/src/pages/ShowcaseCenter.vue`
  - `frontend/src/router/index.ts`
- 关键上下文引用：
  - `frontend/src/pages/ShowcaseCenter.vue`
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/stores/showcase.ts`
- 目标：
  - 将 `Showcase.vue` 重构为 `Showcase Center`
  - 页面只保留 screen 列表与管理入口
  - 提供：
    - New Blank
    - Duplicate Existing
    - Rename
    - Delete
    - Edit
    - View
- 涉及模块 / 文件：
  - `frontend/src/pages/ShowcaseCenter.vue`
  - `frontend/src/router/index.ts`
- 验收条件：
  - 页面不再承载 widget 级重编辑逻辑
  - 列表页展示：
    - 名称
    - layout 类型
    - widget 数量
    - updatedAt
    - 操作按钮
  - 交互不再依赖 `prompt/confirm` 作为主要编辑入口
  - 可从列表页打开 editor / viewer
- 测试点：
  - 手工：
    - 新建 blank
    - duplicate
    - rename
    - delete
    - 打开 editor
    - 打开 viewer
- 回滚点：
  - 回滚 `frontend/src/pages/Showcase.vue`
- 依赖关系：
  - 依赖 `SC-01`
- 风险与注意事项：
  - 列表页不得偷偷保留完整编辑器逻辑
  - 不得建立所有 screen 的运行订阅，避免多余 I/O

---

### SC-03 - Showcase Editor Window 新增与路由接入
- Status：已完成
- Owner：主Agent
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-showcase-center-editor`
- Plan：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-showcase-center-editor\plan.md`
- Write set：
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/windows/ShowcaseEditorWindow.vue`
  - `frontend/src/router/index.ts`
- 关键上下文引用：
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/windows/ShowcaseWindow.vue`
  - `frontend/src/stores/showcase.ts`
  - `frontend/src/lib/showcaseLayout.ts`
- 目标：
  - 新增独立编辑窗口路由 `/showcase-editor-window`
  - 以 `screenId` 为上下文编辑单个 screen
  - 实现三栏结构：
    - 左侧结构/图层列表
    - 中间实时预览
    - 右侧属性面板
  - 支持两种 layout：
    - `columns`
    - `canvas_percent`
  - 支持直接操作 widget
  - 支持 dirty 状态、手动保存、关闭前确认
- 涉及模块 / 文件：
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/windows/ShowcaseEditorWindow.vue`
  - `frontend/src/router/index.ts`
- 验收条件：
  - `#/showcase-editor-window?screenId=...` 可独立访问
  - `screenId` 不存在时有明确提示
  - 编辑器中可修改 screen 与 widget 配置
  - 编辑器中可直接操作 widget
  - 修改未保存时存在 dirty 提示
  - 点击保存后才会落盘并广播配置变更
- 测试点：
  - 手工：
    - columns 编辑
    - canvas 编辑
    - widget 新增 / 编辑 / 删除 / 排序
    - 直接操作 topic_button / slider / switch
    - dirty 提示与保存
- 回滚点：
  - 删除 `frontend/src/windows/ShowcaseEditorWindow.vue`
  - 移除 `/showcase-editor-window` 路由
- 依赖关系：
  - 依赖 `SC-01`
- 风险与注意事项：
  - 不得把旧 `Showcase.vue` 的整页逻辑原样复制到新窗口后继续膨胀
  - canvas 编辑继续保持 pointerup/显式保存的低 I/O 原则

---

### SC-04 - Viewer 对齐与联动回归
- Status：已完成
- Owner：主Agent
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-showcase-center-editor`
- Plan：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-showcase-center-editor\plan.md`
- Write set：
  - `frontend/src/windows/ShowcaseWindow.vue`
- 关键上下文引用：
  - `frontend/src/windows/ShowcaseWindow.vue`
  - `frontend/src/stores/showcase.ts`
  - `frontend/src/windows/ShowcaseEditorWindow.vue`
- 目标：
  - 校准 Viewer 与新的保存策略、schema、事件同步逻辑
  - 确保 editor 保存后 Viewer 自动更新
  - 保持 Viewer 的“纯展示/运行”定位不回退
- 涉及模块 / 文件：
  - `frontend/src/windows/ShowcaseWindow.vue`
- 验收条件：
  - Editor 保存后，已打开 Viewer 自动同步
  - screen 不存在时仍明确报错，不 fallback
  - `columns` / `canvas_percent` 均可正常展示与运行
- 测试点：
  - 手工：
    - 同 screen 打开 editor + viewer
    - 保存后 viewer 自动更新
    - screen missing 提示
- 回滚点：
  - 回滚 `frontend/src/windows/ShowcaseWindow.vue`
- 依赖关系：
  - 依赖 `SC-01`
  - 推荐在 `SC-03` 基本完成后执行
- 风险与注意事项：
  - 不得将 editor 专属状态污染到 viewer

---

### SC-05 - 集成验证与构建回归
- Status：已完成（完整构建受仓库既有问题阻塞，已归档）
- Owner：主Agent
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-showcase-center-editor`
- Plan：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-showcase-center-editor\plan.md`
- Write set：
  - 无业务文件固定写集；允许生成必要的本地 bindings 产物
- 关键上下文引用：
  - `wails.json`
  - `frontend/package.json`（若存在）
- 目标：
  - 执行构建、测试、关键路径冒烟
  - 记录失败项及是否为环境问题
- 涉及模块 / 文件：
  - 构建命令与测试命令
- 验收条件：
  - 至少完成：
    - `go test`
    - `npm run build`
  - 如需 bindings：
    - `wails generate module`
  - 输出关键手工验证结果
- 测试点：
  - `GOWORK=off go test ./... -count=1 -p 1`
  - `npm run build`
  - 必要时 `GOWORK=off wails generate module`
- 回滚点：
  - 回滚本分支提交
- 依赖关系：
  - 依赖 `SC-01` ~ `SC-04`
- 风险与注意事项：
  - 若构建失败，必须判断是本次改动还是环境/既有问题

---

## 3.2 正式并行性评估
- 当前任务可拆分为至少两个相对独立验收块：
  - SC-01：后端 schema + store 基础
  - SC-02 / SC-03 / SC-04：前端界面与窗口重构
- 潜在并行方案：
  - SC-02（Center）与 SC-04（Viewer）可按文件写集独立推进
  - SC-03（Editor）可与 SC-05（验证）分阶段并行
- 本次未使用子Agent：
  - 原因：当前会话存在更高优先级工具约束，只有用户显式要求时才允许派发子Agent；因此由主Agent串行执行并统一集成。
- 结论：
  - 本阶段继续由主Agent在当前 worktree 串行完成，实现与验证记录统一收敛到本 `plan.md`

---

## 风险与注意事项
- 不允许在 `repo/MyFlowHub-Win` 中直接做实现性修改。
- `updatedAt` 必须保证旧配置兼容，不得要求用户清空 profile。
- 列表页必须避免为了展示摘要而建立全量运行订阅。
- 编辑器允许直接操作 widget，但保存策略仍是手动保存；运行交互与配置保存要分清楚。
- Viewer 保持纯展示/运行定位，不接入 editor 专属 dirty 状态。
- 多窗口编辑冲突 V1 只提示，不做锁与合并。

---

## 3.3 Code Review 结论
- 需求覆盖：通过
  - Center / Editor / Viewer 三段式拆分已经落地，保留了 duplicate、两种布局、直接操作 widget、手动保存与 dirty 提示。
- 架构合理性：通过
  - 列表管理、编辑草稿、Viewer 自动同步职责已拆开；store 新增的 draft/save/reload 边界明确。
- 性能风险：通过
  - 编辑过程不再即时保存，降低持久化 I/O 与订阅抖动；当前未见新增的 N+1、重复 reload 或显著多余计算。
- 可读性与一致性：通过
  - 路由命名、窗口语义、中心页字段与编辑页操作都已对齐；局部文案已收敛到 draft / viewer 语义。
- 可扩展性与配置化：通过
  - `updatedAt`、screen summary、editor draft 保存接口为后续模板化、排序、筛选、冲突提示留下了稳定扩展点。
- 稳定性与安全：通过
  - 编辑页增加了关闭前确认和保存前 screen name 校验；screen missing 路径明确，不会静默 fallback 到错误 screen。
- 测试覆盖情况：通过（附带外部阻塞说明）
  - 已补充 `updatedAt` 相关 Go 单测代码。
  - Vue SFC 语法检查通过。
  - 完整 Go / Wails / 前端构建仍被仓库既有问题阻塞，详见 `docs/change/2026-03-21_showcase-center-editor.md`。
- 子Agent治理与审计：通过
  - 本次未使用子Agent；原因、阶段、责任边界已在本 `plan.md` 与变更归档中记录。

---

## 当前结论
- 阶段 `4`：归档完成
- 阻塞：否
- 下一步：向用户汇报结果，并确认是否结束本次 workflow
