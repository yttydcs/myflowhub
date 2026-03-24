# 变更归档：TopicBus 独立窗口右侧侧栏滚动修复

## 变更背景 / 目标
- 背景：
  - TopicBus 独立窗口右侧 `Window Snapshot` / `Window Actions` 侧栏在窗口高度不足时无法滚动。
  - 在聚合窗口中，底部操作按钮和说明文案会被直接裁掉，用户无法完整访问。
- 目标：
  - 让右侧侧栏在内容超出可视高度时具备独立纵向滚动能力。
  - 保持左侧接收 / 发送主工作区和上下拖拽逻辑不回退。

## 对应 plan 任务映射
- TBS-1：修复 TopicBus 独立窗口右侧侧栏滚动
- TBS-2：验证、Review 与归档

## 具体变更内容

### 修改
- `frontend/src/windows/TopicBusWindow.vue`
  - 将右侧 `aside` 从直接堆叠卡片改为“外层容器 + 内层滚动列”结构。
  - 新增 `h-full min-h-0 flex-col overflow-y-auto` 的侧栏内部容器。
  - `Window Snapshot` 卡片改为按内容自然高度展示。
  - `Window Actions` 卡片保留在高度充足时填充剩余空间的能力，但在窗口较矮时会随整个侧栏一起滚动，而不再被底部裁切。

## 关键设计决策与权衡
- 选择“右侧整列滚动”，而不是只让某个内部卡片滚动
  - 原因：用户的问题是整列底部内容不可达，不只是单个 payload 区超高。
  - 权衡：保持 `Window Snapshot` 与 `Window Actions` 的阅读顺序不变，避免在两个卡片里引入多层滚动。

- 不改左侧主工作区
  - 原因：左侧接收 / 发送区已经有稳定的 `overflow` 与上下拖拽关系，本次问题只存在于右侧侧栏。
  - 收益：修复面最小，降低布局回归风险。

## 性能 / 可扩展性说明
- 性能：
  - 仅修改模板层布局类，不增加脚本逻辑、事件监听或重复计算。
- 可扩展性：
  - 侧栏形成稳定滚动容器后，后续即使增加更多快照项、详情字段或快捷操作，也不容易再次出现底部裁切。

## 测试与验证方式 / 结果
- `npm ci`
  - 结果：通过
  - 说明：新 worktree 初始缺少 `frontend/node_modules`

- `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 说明：新 worktree 初始缺少 `frontend/wailsjs` 绑定生成物

- `cd frontend && npm run build`
  - 结果：通过
  - 说明：仍存在既有 Vite chunk size warning，不是本次滚动修复新增问题

## 潜在影响与回滚方案

### 潜在影响
- 右侧侧栏现在会显示独立滚动条；在极低窗口高度下，用户需要滚动右侧列来访问底部控制区，这是预期行为。

### 回滚方案
- 回退以下文件即可恢复本次改动前的行为：
  - `frontend/src/windows/TopicBusWindow.vue`

## 子Agent执行轨迹
- 本 workflow 未使用子Agent。
- 原因：
  - 任务写集集中在单个窗口组件，主Agent 直接修改和验证更安全。
