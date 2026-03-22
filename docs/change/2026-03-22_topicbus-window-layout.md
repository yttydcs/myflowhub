# 变更归档：TopicBus 独立窗口布局精简

## 变更背景 / 目标
- 背景：
  - 现有 TopicBus 独立窗口仍偏卡片堆叠，虽然功能完整，但和 Flow 编辑窗口相比，工作区感不够强，信息和控制区域分层也不够清楚。
  - `Overview` 页面中 `Quick Window / Jump Into Live Traffic` 区块偏跳转导向，和主页面“设置优先”的定位不够一致。
- 目标：
  - 让 TopicBus 独立窗口更接近 Flow 编辑窗口的骨架，整体更简洁。
  - 保留 TopicBus 原有的核心收发体验：上收下发、上下可调高度。
  - 移除 `Overview` 中的快速窗口区块。

## 具体变更内容

### 修改
- `frontend/src/windows/TopicBusWindow.vue`
  - 将窗口从“顶部信息卡 + 主收发卡”重组为：
    - 顶部轻量 header
    - 左侧主工作区
    - 右侧侧栏
  - 左侧主工作区继续保留：
    - Receive 区
    - 可拖拽分隔条
    - Send 区
  - 右侧侧栏拆分为：
    - `Other Info`
    - `Control Buttons`
  - 将已选事件的完整 payload 查看从列表展开改为侧栏查看，主列表更紧凑。
  - 保留聚合窗口 topic 输入、指定频道锁定 topic、发送校验、事件 flush、拖拽高度调整等原有行为。

- `frontend/src/pages/TopicBus.vue`
  - 删除 `Overview` 中的 `Quick Window / Jump Into Live Traffic` 区块。
  - 保留设置区、缓存区、状态摘要区。

### 新增
- `docs/change/2026-03-22_topicbus-window-layout.md`
  - 记录本次布局收敛结果与验证情况。

### 删除
- 删除了 `Overview` 中的快速窗口卡片内容，但未删除主页面打开窗口能力本身。

## 对应计划任务映射
- TBWL-1：重构 TopicBus 独立窗口骨架
- TBWL-2：精简 TopicBus Overview
- TBWL-3：集成验证与归档

## 关键设计决策与权衡
- 独立窗口采用 “header + main + aside” 骨架
  - 原因：Flow 编辑窗口已经证明这种结构更适合长时间停留的工作台视图。
  - 权衡：不直接照搬 Flow 的右侧抽屉，而是保留固定侧栏，避免 TopicBus 界面过于复杂。

- 收发主区继续保留上下分区
  - 原因：这是 TopicBus 的核心交互，不应为了对齐 Flow 风格而改变。
  - 性能点：不改动原有的批量 flush 和 `maxEvents` 截断。

- 已选事件详情移到侧栏
  - 原因：让接收列表更紧凑，减少列表内大块展开内容。
  - 扩展点：后续可以继续在侧栏增加事件元信息，而不需要再改主列表布局。

- `Overview` 移除快速窗口卡片
  - 原因：主页面更应聚焦 TopicBus 设置和状态摘要；打开窗口的入口留在 `Channels` 页即可。

## Code Review 结论
- 需求覆盖：通过
- 架构合理性：通过
- 性能风险：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：通过
- 子Agent治理与审计：通过

## 测试与验证方式 / 结果
- `npm ci`
  - 结果：通过
  - 说明：新 worktree 初始缺少 `frontend/node_modules`

- `GOWORK=off wails generate module`
  - 结果：通过
  - 说明：新 worktree 初始缺少 `frontend/wailsjs` 绑定生成物

- `cd frontend && npm run build`
  - 结果：通过
  - 说明：仍存在既有 Vite chunk 体积警告，不是本次改动新增问题

- `GOWORK=off go test ./... -count=1 -p 1`
  - 结果：通过

- Chrome DevTools 静态 UI 冒烟
  - 结果：未完成
  - 原因：`chrome-devtools-mcp` 的既有 browser profile 已被占用，无法在当前会话内创建或接管页面

## 潜在影响与回滚方案

### 潜在影响
- 已选事件的完整 payload 查看位置从列表内展开改为右侧侧栏，老用户需要适应。
- `Overview` 页面更克制，快速打开窗口的路径收敛到 `Channels` 页。

### 回滚方案
- 回退以下文件即可恢复到本次改动前：
  - `frontend/src/windows/TopicBusWindow.vue`
  - `frontend/src/pages/TopicBus.vue`

## 子Agent执行轨迹
- 无子Agent
- 原因：当前任务规模小，且两个布局改动强耦合，由主Agent直接完成更稳妥
