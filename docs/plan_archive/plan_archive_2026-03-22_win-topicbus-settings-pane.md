# Plan - MyFlowHub-Win：TopicBus 窗口头部精简与设置迁移

## Workflow 信息
- 仓库：MyFlowHub-Win
- 分支：refactor/topicbus-settings-pane
- Base：main
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-settings-pane\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-settings-pane\MyFlowHub-Win\todo.md
- 当前阶段：4 归档变更
- 状态：已完成实现与 Review，等待用户确认是否结束本次 workflow
- 规范：
  - D:\project\MyFlowHub3\guide.md
  - D:\project\MyFlowHub3\AGENTS.md（会话内用户提供）

---

## 1) 需求分析

### 目标
- 进一步精简 TopicBus 独立窗口头部，只保留名称和状态。
- 将 TopicBus 的 `Event Cache Settings` 从 `Overview` 页面迁移到统一的 `Settings` 页面。

### 范围
#### 必须
- 调整 `frontend/src/windows/TopicBusWindow.vue`
  - 移除头部中的 `TopicBus Window` 眉题。
  - 移除头部说明文案：
    - `Watch every known topic from the moment this window opens.`
    - `Focus on one topic with a dedicated receive and send workspace.`
  - 头部仅保留窗口名称和状态区域。
- 调整 `frontend/src/pages/TopicBus.vue`
  - 删除 `Overview` 中的 `Event Cache Settings` 区块。
- 调整 `frontend/src/pages/Settings.vue`
  - 新增 TopicBus 设置区块，承载原 `Event Cache Settings` 的内容和交互。

#### 可选
- 根据 Settings 页现有视觉结构，对 TopicBus 设置区块的文案做轻量收敛。

#### 不做
- 不改 TopicBus 协议、频道页、独立窗口主区收发逻辑。
- 不改 `topicbus.ts` 的数据结构和持久化接口。
- 不改全局 App Settings 的存储模型。

### 使用场景
- 用户打开 TopicBus 独立窗口时，希望头部更克制，不要重复解释性文案。
- 用户希望缓存上限和清空缓存这种偏全局/偏偏好设置项统一进入 `Settings` 页面管理。

### 功能需求
- 独立窗口头部继续显示：
  - 当前窗口名称
  - 连接状态
  - 如有必要的模式状态 badge
- Settings 页面新增 TopicBus 设置卡片，至少包含：
  - `Max Events`
  - `Apply Limit`
  - `Clear Cached`
  - 对“窗口仅显示打开后收到的事件”和“自身 publish 不回显”的说明
- TopicBus 页面 `Overview` 删除缓存设置后，剩余设置和摘要保持完整可用。

### 非功能需求
- 简洁：减少独立窗口头部的无效视觉噪音。
- 一致性：把偏设置性质的能力收敛进 `Settings` 页面。
- 可维护性：复用现有 `topicbus` store，不重复实现缓存设置逻辑。
- 稳定性：不破坏现有独立窗口、TopicBus 页面和 Settings 页面行为。

### 输入输出
- 输入：
  - TopicBus 窗口 route query
  - TopicBus prefs: `maxEvents`
- 输出：
  - 精简后的 TopicBus 窗口头部
  - `Settings` 页面中的 TopicBus 设置区块
  - `Overview` 中移除的缓存设置区块

### 边界异常
- 切换 profile 后，Settings 页面中的 TopicBus 设置也要重新加载对应 profile 的 prefs。
- `Max Events` 仍需保持正整数校验。
- 清空缓存后，不能影响 TopicBus 订阅列表或窗口模式。

### 验收标准
1. TopicBus 独立窗口头部不再出现 `TopicBus Window` 和那段说明文案。
2. 窗口头部仍能清楚显示名称与状态。
3. TopicBus `Overview` 不再出现 `Event Cache Settings`。
4. `Settings` 页面出现 TopicBus 设置区块，且 `Max Events / Apply Limit / Clear Cached` 可用。
5. 构建与测试通过，未引入 TopicBus 行为回退。

### 风险
- 若 Settings 页面没有在进入时主动加载 TopicBus prefs，显示值会过期。
- 若直接复用原 TopicBus 页面逻辑而不调整 profile 切换时机，可能导致 Settings 页中的 TopicBus 设置不同步。

### 问题清单
- 阻塞：否

---

## 2) 架构设计（分析）

### 总体方案
- `TopicBusWindow.vue`
  - 只做头部文案和结构收敛，不触碰主收发区和右侧侧栏逻辑。
- `TopicBus.vue`
  - 删除 `Event Cache Settings` 区块及其本页私有状态/方法引用。
- `Settings.vue`
  - 引入 `useTopicBusStore()`。
  - 新增独立的 TopicBus 设置卡片。
  - 在页面加载与 profile 切换时同步加载 TopicBus prefs。
  - 在本页实现 TopicBus 专用的 `maxEvents` draft、保存和清空缓存动作。

### 模块职责
- `frontend/src/windows/TopicBusWindow.vue`
  - 显示更克制的窗口头部。
- `frontend/src/pages/TopicBus.vue`
  - 保留 TopicBus 主页面的身份、频道和摘要，不再管理缓存设置 UI。
- `frontend/src/pages/Settings.vue`
  - 承载 TopicBus 的偏好设置入口。

### 数据 / 调用流
- Settings 页面挂载或 profile 切换时：
  - 继续加载 `appSettings`
  - 额外调用 `topicbus.loadPrefs()`
  - 同步 `topicbus.state.maxEvents` 到本页输入 draft
- 用户在 Settings 页面调整 TopicBus 缓存时：
  - 输入 `maxEvents`
  - 调用 `topicbus.setMaxEvents()`
  - 调用 `topicbus.clearEvents()`（如用户点清空）

### 接口草案
- 无新增接口。
- 复用：
  - `topicbus.loadPrefs()`
  - `topicbus.setMaxEvents()`
  - `topicbus.clearEvents()`

### 错误与安全
- `Max Events` 输入继续校验正整数。
- Settings 页面中的 TopicBus 操作失败时，toast 报错。
- 清空缓存仅影响本地事件缓存，不触碰 TopicBus 订阅状态。

### 性能与测试策略
- 不新增高频计算。
- 仅在 Settings 页面加载和 profile 切换时多一次 TopicBus prefs 读取。
- 验证：
  - `cd frontend && npm run build`
  - `GOWORK=off go test ./... -count=1 -p 1`

### 可扩展性设计点
- Settings 页面中的 TopicBus 卡片后续可继续承载更多 TopicBus 偏好项。
- TopicBus 页面本身继续专注功能操作，避免设置和实时工作流耦合。

### 问题清单
- 阻塞：否

---

## 3) 可执行任务清单（Checklist）

### TBSP-1：精简 TopicBus 独立窗口头部
- 状态：已完成
- Owner：主Agent
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-settings-pane\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-settings-pane\MyFlowHub-Win\todo.md
- 目标：
  - 让头部只显示名称和状态
- 涉及模块 / 文件：
  - `frontend/src/windows/TopicBusWindow.vue`
- 验收条件：
  - 眉题和说明文案移除
  - 名称与状态仍清楚
- 测试点：
  - 前端构建
- 回滚点：
  - 回退 `frontend/src/windows/TopicBusWindow.vue`
- Write set：
  - `frontend/src/windows/TopicBusWindow.vue`

### TBSP-2：迁移 Event Cache Settings 到 Settings 页面
- 状态：已完成
- Owner：主Agent
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-settings-pane\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-settings-pane\MyFlowHub-Win\todo.md
- 目标：
  - 从 `TopicBus.vue` 删除缓存设置区块
  - 在 `Settings.vue` 新增 TopicBus 设置卡片
- 涉及模块 / 文件：
  - `frontend/src/pages/TopicBus.vue`
  - `frontend/src/pages/Settings.vue`
- 验收条件：
  - `Overview` 不再出现缓存设置
  - `Settings` 页面出现 TopicBus 设置卡片
  - `Max Events / Apply Limit / Clear Cached` 可用
- 测试点：
  - 前端构建
  - Settings 页面 profile 切换后值能刷新
- 回滚点：
  - 回退 `frontend/src/pages/TopicBus.vue`
  - 回退 `frontend/src/pages/Settings.vue`
- Write set：
  - `frontend/src/pages/TopicBus.vue`
  - `frontend/src/pages/Settings.vue`

### TBSP-3：集成验证与归档
- 状态：已完成
- Owner：主Agent
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-settings-pane\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-settings-pane\MyFlowHub-Win\todo.md
- 目标：
  - 完成构建、测试、Code Review、归档
- 涉及模块 / 文件：
  - `frontend/src/windows/TopicBusWindow.vue`
  - `frontend/src/pages/TopicBus.vue`
  - `frontend/src/pages/Settings.vue`
  - `docs/change/YYYY-MM-DD_topicbus-settings-pane.md`
- 验收条件：
  - 构建与测试通过
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
- 改动面小，但 `TopicBus.vue` 和 `Settings.vue` 都围绕同一组缓存设置逻辑迁移，耦合度高。
- 当前平台规则下，未获用户显式授权也不派发子Agent。

### 问题清单
- 阻塞：否

---

## 5) Code Review（3.3）

### 结论
- 通过

### 逐项审查
- 需求覆盖：通过
  - 独立窗口头部已只保留名称和连接状态。
  - `Overview` 中的 `Event Cache Settings` 已移除。
  - `Settings` 页面已新增 TopicBus 设置卡片。
- 架构合理性：通过
  - TopicBus 偏好设置迁移到 `Settings.vue`，更符合职责划分。
  - 复用 `topicbus` store，未复制设置逻辑。
- 性能风险：通过
  - 未新增高频计算或重复 I/O。
  - Settings 页只在加载和 profile 切换时多做一次 `topicbus.loadPrefs()`。
- 可读性与一致性：通过
  - 独立窗口头部明显更克制。
  - Settings 页卡片结构与现有页面风格一致。
- 可扩展性与配置化：通过
  - TopicBus 设置区块后续可继续承载更多 TopicBus 偏好项。
- 稳定性与安全：通过
  - `Max Events` 正整数校验保持不变。
  - 清空缓存仍仅影响本地事件缓存。
- 测试覆盖情况：通过
  - `npm ci` 通过
  - `GOWORK=off wails generate module` 通过
  - `cd frontend && npm run build` 通过
  - `GOWORK=off go test ./... -count=1 -p 1` 通过
- 子Agent治理与审计：通过
  - 本 workflow 未使用子Agent
  - 原因：任务规模小且文件写集强耦合

### 残余风险
- 本轮未做真实 TopicBus 会话联动的手工 UI 冒烟；建议在实际连接环境再看一轮 Settings 页改动后的使用路径。

---

## 6) 执行记录
- 2026-03-22：完成初始化（独占分支 + worktree + 文档骨架）
- 2026-03-22：完成需求分析与架构设计，无阻塞
- 2026-03-22：完成 3.1 计划文档
- 2026-03-22：完成 TBSP-1，精简 TopicBus 独立窗口头部
- 2026-03-22：完成 TBSP-2，将 Event Cache Settings 迁移到 Settings 页面
- 2026-03-22：完成 TBSP-3，执行 `npm ci`、`GOWORK=off wails generate module`、`npm run build`、`GOWORK=off go test ./... -count=1 -p 1`
- 2026-03-22：完成 3.3 Code Review，结论：通过
