# Plan - MyFlowHub-Win：TopicBus 独立窗口操作区高度修复

## Workflow 信息
- 仓库：MyFlowHub-Win
- 分支：fix/topicbus-window-actions-height
- Base：main
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-window-actions-height\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-window-actions-height\MyFlowHub-Win\todo.md
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
- 已创建独占分支：`fix/topicbus-window-actions-height`
- 已创建独占 worktree：
  - `D:\project\MyFlowHub3\worktrees\topicbus-window-actions-height\MyFlowHub-Win`

### 问题清单
- 阻塞：否

---

## 1) 需求分析

### 目标
- 修复 TopicBus 独立窗口右侧 `Window Actions` 卡片在当前布局下内容显示不全的问题。
- 保证卡片内部按钮和说明块在不同窗口高度下都能完整访问。

### 范围
#### 必须
- 调整 `frontend/src/windows/TopicBusWindow.vue` 中右侧 `Window Actions` 卡片的高度与内部布局。
- 保证 `Window Actions` 的子元素能正确参与父容器高度分配，必要时在卡片内部提供滚动。
- 不破坏同一侧栏中的 `Window Snapshot` 卡片和整体右侧列滚动行为。

#### 可选
- 若本次调整能顺带提升右侧侧栏整体高度分配一致性，可一并收口，但不额外扩展功能。

#### 不做
- 不修改 TopicBus 事件接收、发送、过滤或窗口路由逻辑。
- 不重构 TopicBus 主页面。

### 使用场景
- 用户在聚合窗口中查看右侧操作区，需要访问 `Scroll to Latest`、`Clear Receive List`、`Reset Draft` 以及底部说明。
- 用户缩小窗口高度后，仍需完整访问 `Window Actions` 内全部内容。

### 功能需求
- `Window Actions` 卡片必须完整显示其内部内容，或在卡片内部提供可预期的滚动。
- 卡片子元素不能因为父级高度分配错误而被裁切。
- 右侧 `Window Snapshot` 与 `Window Actions` 的上下关系保持清晰，不引入新的布局跳动。

### 非功能需求
- 可维护性：优先沿用现有 `flex / min-h-0 / overflow-y-auto` 模式，不新增自定义 CSS。
- 稳定性：只在单一窗口组件内修复，不扩大修改面。
- 一致性：保持与 Flow 编辑器/现有侧栏卡片的高度处理方式一致。

### 输入输出
- 输入：
  - 右侧 `Window Actions` 卡片中的现有按钮与说明内容
  - 用户调整后的窗口高度
- 输出：
  - 完整可访问的 `Window Actions` 卡片
  - 正确撑开或滚动的父子容器关系

### 边界异常
- 在超低窗口高度下，仍要能通过滚动访问到底部说明。
- 在较高窗口高度下，卡片不应出现异常的空白坍塌或布局拉伸。
- 右侧存在长快照内容时，`Window Actions` 不能因为兄弟卡片增长而变成不可见。

### 验收标准
1. `Window Actions` 卡片中的三个按钮和底部说明均可访问。
2. 卡片内容不会再因为父容器高度分配不正确而被裁切。
3. 前端构建通过。

### 风险
- 如果只调整外层侧栏滚动，而不处理 `Window Actions` 卡片自身的 `flex` 关系，卡片内部仍可能显示不全。
- 如果把整个卡片强制固定高度但不提供内部滚动，会在低高度窗口下再次出现裁切。

### 问题清单
- 阻塞：否

---

## 2) 架构设计（分析）

### 总体方案
- 在 `frontend/src/windows/TopicBusWindow.vue` 中，把 `Window Actions` 卡片改为真正的纵向 `flex` 容器，并将其内容区放入 `flex-1 min-h-0` 的主体层。
- 让右侧整列继续保持整体滚动，同时给 `Window Actions` 卡片内部一个明确的高度承接关系，避免“父卡片拉伸但子内容没有参与高度计算”。

### 模块职责
- `frontend/src/windows/TopicBusWindow.vue`
  - 负责独立窗口整体骨架、右侧侧栏卡片与内部滚动关系。

### 数据 / 调用流
- 本次不涉及数据流调整。
- 仅修改模板中的结构层级和 `flex / overflow` 类。

### 接口草案
- 无新增接口。

### 错误与安全
- 本次为纯前端布局修复，不新增输入校验或后端调用。

### 性能与测试策略
- 性能：
  - 仅调整布局类，不新增脚本逻辑、事件监听或计算负担。
- 测试：
  - `npm ci`
  - `$env:GOWORK='off'; wails generate module`
  - `cd frontend && npm run build`

### 可扩展性设计点
- 为 `Window Actions` 建立清晰的卡片内部分区后，后续如果再增加更多按钮或说明内容，也能继续通过内部滚动承接，而不再依赖外层偶然的高度分配。

### 问题清单
- 阻塞：否

---

## 3.1) 可执行任务清单（Checklist）

### TBAH-1：修复 TopicBus 独立窗口操作区高度承接
- 状态：已完成
- Owner：主Agent
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-window-actions-height\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-window-actions-height\MyFlowHub-Win\todo.md
- 目标：
  - 让 `Window Actions` 卡片正确承接父容器高度并完整显示内部内容
- 涉及模块 / 文件：
  - `frontend/src/windows/TopicBusWindow.vue`
- 验收条件：
  - 按钮和底部说明不再被裁切
  - 低高度时滚动行为可预期
- 测试点：
  - 前端构建
- 回滚点：
  - 回退 `frontend/src/windows/TopicBusWindow.vue`
- Write set：
  - `frontend/src/windows/TopicBusWindow.vue`

### TBAH-2：验证、Review 与归档
- 状态：已完成
- Owner：主Agent
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-window-actions-height\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-window-actions-height\MyFlowHub-Win\todo.md
- 目标：
  - 完成构建验证、Code Review 和归档文档
- 涉及模块 / 文件：
  - `docs/change/YYYY-MM-DD_topicbus-window-actions-height.md`
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
- 任务集中在单文件布局修复，写集过小且高度耦合，不适合拆分。

### 问题清单
- 阻塞：否

---

## 3.2) 实施记录

### TBAH-1：修复 TopicBus 独立窗口操作区高度承接
- 完成内容：
  - 将 `frontend/src/windows/TopicBusWindow.vue` 的右侧 `aside` 改为显式的 `flex h-full min-h-0` 容器。
  - 将侧栏内部主容器改为 `flex-1 min-h-0`，确保卡片本身参与高度分配。
  - 将 `Window Actions` 卡片改为纵向 `flex` 卡片，并为其内部主体增加 `flex-1 min-h-0` 承接层。
  - 让按钮组和底部说明位于同一个可承接高度的内容区，避免内容被父级裁切。
- 关键设计点：
  - 保持右侧整列滚动能力。
  - 同时修复 `Window Actions` 卡片自身的父子高度关系，避免只依赖外层滚动偶然兜底。
- 验收结果：
  - 代码实现完成。

### TBAH-2：验证、Review 与归档
- 完成内容：
  - 补齐前端依赖和 Wails 绑定。
  - 完成前端构建验证、Code Review 与归档文档编写。
- 验收结果：
  - `npm ci`：通过
  - `$env:GOWORK='off'; wails generate module`：通过
  - `cd frontend && npm run build`：通过

---

## 3.3) Code Review

- 需求覆盖：通过。`Window Actions` 按钮区和说明区已进入明确的高度承接层，不再依赖模糊的父容器拉伸。
- 架构合理性：通过。修改集中在 `TopicBusWindow.vue` 的模板结构和布局类，没有引入新的状态或逻辑耦合。
- 性能风险：通过。仅调整布局类和少量结构层级，不新增事件监听、计算或渲染热点。
- 可读性与一致性：通过。使用项目已有的 `flex / min-h-0 / overflow-y-auto` 组合，和现有侧栏、抽屉模式一致。
- 可扩展性与配置化：通过。后续即使 `Window Actions` 再增加更多按钮或提示块，也能沿用当前卡片内部主体层承接高度。
- 稳定性与安全：通过。本次未改 TopicBus 业务逻辑、运行时请求或事件监听。
- 测试覆盖情况：通过。前端构建通过；运行态浏览器独立验证未做，因为纯 Vite 页面缺少 Wails runtime，不能等价复现窗口逻辑。
- 子Agent治理与审计：通过。本 workflow 未使用子Agent，原因是任务集中在单文件布局修复且写集不可再安全拆分。

### Review 结论
- 结论：通过
- 阻塞：否

---

## 4) 归档变更

- 状态：已完成
- 归档文档：
  - `D:\project\MyFlowHub3\worktrees\topicbus-window-actions-height\MyFlowHub-Win\docs\change\2026-03-22_topicbus-window-actions-height.md`
- 问题清单：
  - 阻塞：否
