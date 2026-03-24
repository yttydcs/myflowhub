# 变更归档：TopicBus 独立窗口操作区高度承接修复

## 变更背景 / 目标
- 背景：
  - 虽然右侧侧栏整体已经具备滚动能力，但 `Window Actions` 卡片自身的父子高度关系仍不完整。
  - 在某些窗口高度下，`Window Actions` 内的按钮和底部说明看起来像是没有把父容器撑开，导致内容显示不全。
- 目标：
  - 让 `Window Actions` 卡片本身成为清晰的纵向布局容器。
  - 让卡片内部内容拥有明确的高度承接层，在受限高度下也能完整访问。

## 对应 plan 任务映射
- TBAH-1：修复 TopicBus 独立窗口操作区高度承接
- TBAH-2：验证、Review 与归档

## 具体变更内容

### 修改
- `frontend/src/windows/TopicBusWindow.vue`
  - 右侧 `aside` 改为显式的 `flex h-full min-h-0` 容器。
  - 侧栏内部滚动列改为 `flex-1 min-h-0`，确保子卡片参与高度分配。
  - `Window Actions` 卡片改为 `flex flex-col` 容器。
  - `Window Actions` 内部新增 `flex-1 min-h-0` 的主体层，把按钮组和说明块纳入同一个可承接高度的区域。

## 关键设计决策与权衡
- 不仅修外层侧栏，还修卡片自身高度关系
  - 原因：用户反馈的问题已经从“整列无法滚动”进一步收敛到“操作卡片内容似乎没有撑开父容器”。
  - 结论：只保留外层滚动不够，必须把 `Window Actions` 自身改成明确的 `flex` 卡片。

- 保留最小修改面
  - 原因：问题局限在右侧操作卡片的结构层级，不需要碰 TopicBus 的业务逻辑或左侧主工作区。
  - 收益：回归风险低，后续更容易继续演进。

## 性能 / 可扩展性说明
- 性能：
  - 仅调整布局结构和 Tailwind 类，不增加脚本层开销。
- 可扩展性：
  - 以后如果 `Window Actions` 再增加更多操作按钮或提示区，可以继续沿用当前的卡片主体层结构，不容易再次出现父子高度错位。

## 测试与验证方式 / 结果
- `npm ci`
  - 结果：通过
  - 说明：新 worktree 初始缺少 `frontend/node_modules`

- `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 说明：新 worktree 初始缺少 `frontend/wailsjs` 绑定生成物

- `cd frontend && npm run build`
  - 结果：通过
  - 说明：仍存在既有 Vite chunk size warning，不是本次修复新增问题

## 潜在影响与回滚方案

### 潜在影响
- `Window Actions` 卡片在高度受限时会更明确地表现为内部主体承接高度，布局会比之前稳定。

### 回滚方案
- 回退以下文件即可恢复本次改动前的行为：
  - `frontend/src/windows/TopicBusWindow.vue`

## 子Agent执行轨迹
- 本 workflow 未使用子Agent。
- 原因：
  - 任务仅涉及单文件布局修复，主Agent 直接实现和验证更合适。
