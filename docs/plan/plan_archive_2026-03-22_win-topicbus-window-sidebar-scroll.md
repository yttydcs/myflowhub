# Plan - MyFlowHub-Win：TopicBus 独立窗口右侧快照列滚动修复

## Workflow 信息
- 仓库：MyFlowHub-Win
- 分支：fix/topicbus-window-snapshot-scroll
- Base：main
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-window-snapshot-scroll\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-window-snapshot-scroll\MyFlowHub-Win\todo.md
- 当前阶段：4 归档变更
- 状态：编码、验证、Code Review 与归档已完成，待用户确认是否结束本次 workflow
- 规范：
  - D:\project\MyFlowHub3\guide.md
  - D:\project\MyFlowHub3\AGENTS.md（会话内用户提供）

---

## 初始化

### 已完成
- 已读取 `guide.md`
- 已确认参与仓库：`MyFlowHub-Win`
- 已确认基线分支：`main`
- 已创建独占分支：`fix/topicbus-window-snapshot-scroll`
- 已创建独占 worktree：
  - `D:\project\MyFlowHub3\worktrees\topicbus-window-snapshot-scroll\MyFlowHub-Win`

### 问题清单
- 阻塞：否

---

## 1) 需求分析

### 目标
- 修复 TopicBus 独立窗口右侧 `Window Snapshot` / `Window Actions` 侧栏在窗口高度不足时无法滚动、底部内容显示不全的问题。

### 范围
#### 必须
- 调整 `frontend/src/windows/TopicBusWindow.vue` 的右侧侧栏布局与滚动容器。
- 保证在聚合窗口 `All` 模式下，右侧内容在低高度窗口内仍可完整访问。
- 保证不破坏左侧收发主工作区的上下分栏与拖拽逻辑。

#### 可选
- 若布局调整天然覆盖单频道独立窗口，则一并受益，但不额外扩展功能范围。

#### 不做
- 不修改 TopicBus 独立窗口的业务逻辑、事件过滤、发送逻辑或右侧信息内容结构。
- 不重构主页面 `TopicBus.vue`。

### 使用场景
- 用户打开 `All` 聚合独立窗口后，在较小高度窗口中查看右侧快照和控制按钮。
- 用户选中事件后，需要能滚动到右侧详情卡片底部查看完整信息和操作按钮。

### 功能需求
- 右侧侧栏必须在内容超出可视高度时出现纵向滚动。
- `Window Snapshot` 和 `Window Actions` 两块内容都必须可访问，不能被底部裁切。
- 左侧收发工作区继续保持当前的主区域优先布局。

### 非功能需求
- 简洁：仅修复滚动容器，不引入额外视觉复杂度。
- 可维护性：优先使用现有 Tailwind 布局约定，避免新增自定义样式。
- 稳定性：不影响 TopicBus 窗口的 existing interaction，包括接收列表滚动、发送区滚动和上下拖拽。

### 输入输出
- 输入：
  - 用户调整后的窗口高度
  - 右侧侧栏现有的快照信息、事件详情和控制按钮
- 输出：
  - 可滚动的右侧侧栏
  - 完整可访问的底部内容

### 边界异常
- 窗口高度很小且选中事件 payload 很长时，右侧仍要能滚动到底部。
- 未选中事件时，右侧空态说明和控制按钮仍要可见。
- 桌面宽屏使用右侧固定侧栏，小屏回落为垂直堆叠时不能引入新的裁切。

### 验收标准
1. TopicBus 聚合独立窗口右侧侧栏在内容超高时可滚动。
2. `Window Actions` 区域和底部说明不再被裁切。
3. 长 payload 不会导致整个右侧列失去滚动能力。
4. 前端构建通过。

### 风险
- 如果只给某个内部卡片加滚动，而外层容器继续 `overflow-hidden`，底部仍可能被裁掉。
- 如果错误地把整个窗口主容器改成可滚动，可能破坏左侧主工作区的固定布局与拖拽体验。

### 问题清单
- 阻塞：否

---

## 2) 架构设计（分析）

### 总体方案
- 在 `frontend/src/windows/TopicBusWindow.vue` 中，将右侧 `aside` 从“外层裁切”改为“外层可滚动 / 内层可收缩”的布局。
- 参考现有 `FlowEditorWindow.vue` 中“固定头部 + 内容区 `flex-1 overflow-y-auto`”的模式，尽量让滚动发生在右侧侧栏自身，而不是整个窗口。

### 模块职责
- `frontend/src/windows/TopicBusWindow.vue`
  - 负责独立窗口骨架、左右布局、右侧侧栏滚动和卡片分区。

### 数据 / 调用流
- 本次不涉及数据流变更。
- 仅修改模板层容器的 `flex / min-h-0 / overflow-y-auto` 布局关系。

### 接口草案
- 无新增接口。

### 错误与安全
- 本次为纯前端布局修复，不新增输入，也不改动 TopicBus 请求发送逻辑。

### 性能与测试策略
- 性能：
  - 仅调整 CSS utility class，不引入额外 watcher、事件监听或计算。
- 测试：
  - `cd frontend && npm run build`
  - 如需要，补 `wails generate module`
  - 可选：使用 Chrome DevTools 观察低高度窗口布局和滚动条行为

### 可扩展性设计点
- 让右侧侧栏形成稳定的滚动容器后，后续即使再增加更多快照项或操作按钮，也不会再次出现底部裁切。

### 问题清单
- 阻塞：否

---

## 3.1) 可执行任务清单（Checklist）

### TBS-1：修复 TopicBus 独立窗口右侧侧栏滚动
- 状态：已完成
- Owner：主Agent
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-window-snapshot-scroll\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-window-snapshot-scroll\MyFlowHub-Win\todo.md
- 目标：
  - 让右侧 `Window Snapshot` / `Window Actions` 侧栏在内容超高时能完整滚动
- 涉及模块 / 文件：
  - `frontend/src/windows/TopicBusWindow.vue`
- 验收条件：
  - 右侧侧栏出现纵向滚动
  - 底部按钮和说明可访问
  - 左侧收发区布局不回退
- 测试点：
  - 前端构建
  - 低高度窗口的静态布局检查
- 回滚点：
  - 回退 `frontend/src/windows/TopicBusWindow.vue`
- Write set：
  - `frontend/src/windows/TopicBusWindow.vue`

### TBS-2：验证、Review 与归档
- 状态：已完成
- Owner：主Agent
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-window-snapshot-scroll\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-window-snapshot-scroll\MyFlowHub-Win\todo.md
- 目标：
  - 完成构建验证、Code Review 与归档
- 涉及模块 / 文件：
  - `docs/change/YYYY-MM-DD_topicbus-window-sidebar-scroll.md`
- 验收条件：
  - 构建通过
  - Review 通过
  - 归档完整
- 测试点：
  - `cd frontend && npm run build`
- 回滚点：
  - 回退本 workflow 全部改动
- Write set：
  - `docs/change/`

---

## 3.2 前并行性评估

### 结论
- 本次不使用子Agent。

### 原因
- 当前任务集中在单文件布局修复，写集不可再安全拆分。
- 用户未显式要求子Agent并行执行。

### 问题清单
- 阻塞：否

---

## 3.2) 实施记录

### TBS-1：修复 TopicBus 独立窗口右侧侧栏滚动
- 完成内容：
  - 将 `frontend/src/windows/TopicBusWindow.vue` 的右侧 `aside` 改为独立滚动容器。
  - 为侧栏内部增加 `h-full min-h-0 flex-col overflow-y-auto` 包裹层。
  - 让 `Window Snapshot` 卡片固定按内容高度展示，`Window Actions` 在高度充足时继续填充剩余空间，在高度不足时随侧栏整体滚动。
- 关键设计点：
  - 滚动发生在右侧侧栏自身，而不是整个窗口根容器。
  - 保持左侧收发主区的 `overflow-hidden` 和拖拽分栏逻辑不变，避免把布局问题扩散到主工作区。
- 验收结果：
  - 代码实现完成。

### TBS-2：验证、Review 与归档
- 完成内容：
  - 补齐新 worktree 的前端依赖与 Wails 绑定生成。
  - 完成前端构建验证、Code Review 与归档文档编写。
- 验收结果：
  - `npm ci`：通过
  - `$env:GOWORK='off'; wails generate module`：通过
  - `cd frontend && npm run build`：通过

---

## 3.3) Code Review

- 需求覆盖：通过。右侧侧栏已具备独立纵向滚动能力，底部操作区不再依赖固定视口高度。
- 架构合理性：通过。修复集中在 `TopicBusWindow.vue` 布局层，没有引入新的状态或额外交互耦合。
- 性能风险：通过。仅调整 Tailwind 布局类，不增加 watcher、定时器或额外 DOM 计算。
- 可读性与一致性：通过。沿用现有 `flex + min-h-0 + overflow-y-auto` 模式，和项目内侧栏/抽屉实现一致。
- 可扩展性与配置化：通过。右侧列形成稳定滚动容器后，后续新增快照项或控制按钮不容易再次触发裁切问题。
- 稳定性与安全：通过。本次未改业务逻辑、TopicBus 请求、事件监听或输入处理。
- 测试覆盖情况：通过。前端构建已通过；当前未补浏览器运行态冒烟，但静态模板与构建链路无报错。
- 子Agent治理与审计：通过。本 workflow 未使用子Agent；任务为单文件紧耦合布局修复，不适合拆分。

### Review 结论
- 结论：通过
- 阻塞：否

---

## 4) 归档变更

- 状态：已完成
- 归档文档：
  - `D:\project\MyFlowHub3\worktrees\topicbus-window-snapshot-scroll\MyFlowHub-Win\docs\change\2026-03-22_topicbus-window-sidebar-scroll.md`
- 问题清单：
  - 阻塞：否
