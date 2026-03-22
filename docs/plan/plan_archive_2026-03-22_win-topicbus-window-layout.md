# Plan - MyFlowHub-Win：TopicBus 独立窗口布局精简

## Workflow 信息
- 仓库：MyFlowHub-Win
- 分支：refactor/topicbus-window-layout
- Base：main
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-window-layout\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-window-layout\MyFlowHub-Win\todo.md
- 当前阶段：4 归档变更
- 状态：已完成实现与 Review，等待用户确认是否结束本次 workflow
- 规范：
  - D:\project\MyFlowHub3\guide.md
  - D:\project\MyFlowHub3\AGENTS.md（会话内用户提供）

---

## 1) 需求分析

### 目标
- 将 TopicBus 独立窗口进一步简化，参考 Flow 编辑窗口的布局骨架。
- 保留 TopicBus 现有“上收下发 + 可调高度”的核心交互。
- 移除 `Overview` 页中偏跳转导向的 `Quick Window / Jump Into Live Traffic` 区块。

### 范围
#### 必须
- 调整 `frontend/src/windows/TopicBusWindow.vue` 的整体结构：
  - 采用接近 `FlowEditorWindow` 的“顶部轻量标题 + 主工作区 + 侧栏信息/控制区”布局。
  - 主工作区仍保留 Receive / Send 上下分区与拖拽分隔条。
  - 侧栏拆分为“其他信息”和“控制按钮”两组，界面更简洁。
- 调整 `frontend/src/pages/TopicBus.vue`：
  - 从 `Overview` 中移除 `Quick Window` / `Jump Into Live Traffic` 区块。

#### 可选
- 在 TopicBus 窗口中压缩顶部说明文案与徽标数量。
- 将当前已选事件详情迁移到更符合新布局的位置。

#### 不做
- 不改 TopicBus 协议行为。
- 不改频道列表页结构与打开窗口方式。
- 不改 `topicbus.ts` 里的订阅、事件流、发送逻辑。

### 使用场景
- 用户从 `Channels` 打开独立 TopicBus 窗口后，希望界面更像一个专用工作台，而不是大块卡片堆叠。
- 用户在独立窗口里主要做三件事：
  - 观察新事件
  - 编辑并发送事件
  - 查看当前窗口状态和执行常用控制动作

### 功能需求
- 独立窗口继续区分：
  - `All` 聚合窗口
  - 指定 topic 窗口
- 指定 topic 窗口继续锁定 topic 输入。
- `All` 窗口继续允许手工输入 topic。
- 右侧侧栏必须能呈现：
  - 当前窗口模式 / 连接 / self / target / cache 等其他信息
  - 清空、滚动到最新、清空发送区等控制按钮
- `Overview` 移除快速打开窗口卡片后，剩余内容仍然完整可用。

### 非功能需求
- 简洁：减少重复说明和顶层大卡片层级。
- 可读：布局层次清楚，信息密度集中在主工作区和右侧侧栏。
- 可扩展：右侧侧栏后续可继续添加状态项或控制动作，不破坏主收发区。
- 稳定：不破坏现有事件监听、发送校验、拖拽高度调整。

### 输入输出
- 输入：
  - route query: `scope=all` / `topic=<name>` / `targetId`
  - TopicBus 运行时事件：`topicbus.event`
- 输出：
  - 调整后的独立窗口 UI
  - 精简后的 `Overview` 页面 UI

### 边界异常
- 未连接 / 未登录时，发送仍需阻断。
- 聚合窗口发送前，topic 仍需非空。
- 无事件时，接收区仍需清晰空态。
- 侧栏在窄屏下要退化为堆叠布局，不能压坏主区。

### 验收标准
1. 独立窗口结构明显接近 Flow 编辑窗口的骨架，而不是原来的上下两大卡片堆叠。
2. 窗口仍保留 Receive / Send 的上下分区和可拖拽高度。
3. 右侧存在“其他信息”和“控制按钮”分区，整体更简洁。
4. `Overview` 页不再出现 `Quick Window` / `Jump Into Live Traffic`。
5. 发送、接收、topic 锁定、聚合窗口 topic 输入等原有行为不回退。

### 风险
- 如果把过多按钮从主区移到侧栏，可能影响发送路径可发现性。
- 如果布局过度贴近 Flow 编辑窗口，可能牺牲 TopicBus 收发区的直观性。

### 问题清单
- 阻塞：否

---

## 2) 架构设计（分析）

### 总体方案
- `TopicBusWindow.vue`
  - 复用 Flow 编辑窗口的高层骨架：
    - 顶部轻量 header
    - 左侧主工作区
    - 右侧 aside
  - 不动脚本中的 TopicBus 行为逻辑，主要重组 template 和局部样式结构。
- `TopicBus.vue`
  - 删除 `Overview` 中的快速窗口区块，保留设置、摘要和缓存设置卡片。

### 模块职责
- `frontend/src/windows/TopicBusWindow.vue`
  - 负责独立窗口的主收发工作区、状态信息区、控制按钮区。
- `frontend/src/pages/TopicBus.vue`
  - 负责主页面 overview 的设置和摘要，不再承担快速跳转卡片。

### 数据 / 调用流
- 不改变现有调用流：
  - `TopicBusWindow.vue` 继续监听 `topicbus.event`
  - 继续使用 `topicbus.publish()`
  - 继续使用本地 `localEvents` + `pendingEvents` flush
- 只改变这些数据在模板中的组织与呈现位置。

### 接口草案
- 无新增接口。
- 无新增 store 字段。

### 错误与安全
- 保持现有 `ensureReady()`、聚合窗口 topic 校验、具体频道锁定 topic 的策略。
- 控制按钮重排后，危险动作仍仅限“清空本地视图 / 清空发送区”，不新增风险操作。

### 性能与测试策略
- 不增加新的事件级计算。
- 仍保留现有批量 flush 和 `maxEvents` 截断。
- 主要验证：
  - 前端构建通过
  - Go 测试通过
  - 若可行，窗口 UI 用 devtools 做基本静态检查

### 可扩展性设计点
- 右侧 aside 采用两块卡片，未来可继续扩展状态和动作。
- 主收发区继续独立，后续若要增加事件详情抽屉或过滤器，改动范围局部。

### 问题清单
- 阻塞：否

---

## 3) 可执行任务清单（Checklist）

### TBWL-1：重构 TopicBus 独立窗口骨架
- 状态：已完成
- Owner：主Agent
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-window-layout\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-window-layout\MyFlowHub-Win\todo.md
- 目标：
  - 将 TopicBus 独立窗口改为更接近 Flow 编辑窗口的骨架
  - 主区保留收发上下分区
  - 侧栏承载“其他信息 / 控制按钮”
- 涉及模块 / 文件：
  - `frontend/src/windows/TopicBusWindow.vue`
- 验收条件：
  - 顶部、主区、右侧侧栏结构清晰
  - 发送接收逻辑不回退
  - 拖拽分隔条继续工作
- 测试点：
  - 前端构建
  - 侧栏和主区在桌面宽度下排版正确
- 回滚点：
  - 回退 `frontend/src/windows/TopicBusWindow.vue`
- Write set：
  - `frontend/src/windows/TopicBusWindow.vue`

### TBWL-2：精简 TopicBus Overview
- 状态：已完成
- Owner：主Agent
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-window-layout\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-window-layout\MyFlowHub-Win\todo.md
- 目标：
  - 删除 `Overview` 中的 `Quick Window / Jump Into Live Traffic`
- 涉及模块 / 文件：
  - `frontend/src/pages/TopicBus.vue`
- 验收条件：
  - `Overview` 页面不再出现该卡片
  - 页面结构仍平衡
- 测试点：
  - 前端构建
  - Overview 卡片布局无破坏
- 回滚点：
  - 回退 `frontend/src/pages/TopicBus.vue`
- Write set：
  - `frontend/src/pages/TopicBus.vue`

### TBWL-3：集成验证与归档
- 状态：已完成
- Owner：主Agent
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-window-layout\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-window-layout\MyFlowHub-Win\todo.md
- 目标：
  - 执行构建、测试、Code Review 和变更归档
- 涉及模块 / 文件：
  - `frontend/src/windows/TopicBusWindow.vue`
  - `frontend/src/pages/TopicBus.vue`
  - `docs/change/YYYY-MM-DD_topicbus-window-layout.md`
- 验收条件：
  - 构建 / 测试通过
  - Review 通过
  - 归档文档完整
- 测试点：
  - `cd frontend && npm run build`
  - `GOWORK=off go test ./... -count=1 -p 1`
- 回滚点：
  - 回退本 workflow 的全部改动

---

## 4) 并行性评估（供 3.2 使用）

### 结论
- 本次不使用子Agent。

### 原因
- 改动面很小，但两个目标文件在视觉和布局层面高度耦合。
- 当前任务不值得再拆分出独立验收的并行写集。

### 问题清单
- 阻塞：否

---

## 5) Code Review（3.3）

### 结论
- 通过

### 逐项审查
- 需求覆盖：通过
  - 独立窗口已改为更接近 Flow 编辑窗口的“header + 主工作区 + 侧栏”骨架。
  - `Overview` 已移除 `Quick Window / Jump Into Live Traffic`。
- 架构合理性：通过
  - 只调整 `TopicBusWindow.vue` 和 `TopicBus.vue` 的布局组织。
  - TopicBus 的监听、发送、拖拽、频道模式逻辑未改。
- 性能风险：通过
  - 未新增事件级循环或高频计算。
  - 仍沿用原有的事件批量 flush 与 `maxEvents` 截断。
- 可读性与一致性：通过
  - 独立窗口的高层骨架对齐 `FlowEditorWindow`。
  - 侧栏职责清楚，主区聚焦收发。
- 可扩展性与配置化：通过
  - 右侧 `Other Info / Control Buttons` 可继续扩展，不会侵入主区。
- 稳定性与安全：通过
  - 发送前校验、聚合窗口 topic 输入、具体频道锁定 topic 保持原状。
- 测试覆盖情况：通过
  - `npm run build` 通过
  - `GOWORK=off go test ./... -count=1 -p 1` 通过
  - DevTools 静态 UI 冒烟尝试执行，但受已有 Chrome MCP profile 占用影响，未完成页面快照核验
- 子Agent治理与审计：通过
  - 本 workflow 未使用子Agent
  - 原因：任务规模小且布局上下文耦合高

### 残余风险
- 未做真实 TopicBus 会话联动的手工窗口冒烟，仍建议在带 session 的环境下再看一轮收发体验。

---

## 6) 执行记录
- 2026-03-22：完成初始化（独占分支 + worktree + 文档骨架）
- 2026-03-22：完成需求分析与架构设计，无阻塞
- 2026-03-22：完成 3.1 计划文档
- 2026-03-22：完成 TBWL-1，重构 TopicBus 独立窗口骨架
- 2026-03-22：完成 TBWL-2，移除 Overview 快速窗口区块
- 2026-03-22：完成 TBWL-3，执行 `npm ci`、`GOWORK=off wails generate module`、`npm run build`、`GOWORK=off go test ./... -count=1 -p 1`
- 2026-03-22：完成 3.3 Code Review，结论：通过
