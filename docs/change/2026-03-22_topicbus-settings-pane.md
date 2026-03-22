# 变更归档：TopicBus 窗口头部精简与设置迁移

## 变更背景 / 目标
- 背景：
  - TopicBus 独立窗口头部仍保留多余的眉题和说明文案，视觉上偏冗余。
  - `Event Cache Settings` 原本位于 TopicBus `Overview`，更像偏好设置，不适合继续留在功能页。
- 目标：
  - 让独立窗口头部只保留名称和状态。
  - 把 `Event Cache Settings` 迁移到统一的 `Settings` 页面。

## 具体变更内容

### 修改
- `frontend/src/windows/TopicBusWindow.vue`
  - 移除头部中的 `TopicBus Window` 眉题。
  - 移除头部说明文案。
  - 移除头部中的窗口模式 badge。
  - 头部仅保留窗口名称和连接状态。

- `frontend/src/pages/TopicBus.vue`
  - 删除 `Overview` 中的 `Event Cache Settings` 区块。
  - 删除本页不再需要的 `maxEvents` 输入与应用逻辑。

- `frontend/src/pages/Settings.vue`
  - 引入 `useTopicBusStore()`。
  - 在 Settings 页面右侧新增 TopicBus 设置卡片。
  - 新增 `topicbusMaxEventsInput` 与 `topicbusBusy`。
  - 在页面加载和 profile 切换时同步加载 TopicBus prefs。
  - 提供：
    - `Max Events`
    - `Apply Limit`
    - `Clear Cached`

### 新增
- `docs/change/2026-03-22_topicbus-settings-pane.md`
  - 记录本次设置迁移与头部收敛结果。

### 删除
- 删除了 TopicBus `Overview` 中的缓存设置区块，但 TopicBus 缓存能力本身未删除，只是迁移了位置。

## 对应计划任务映射
- TBSP-1：精简 TopicBus 独立窗口头部
- TBSP-2：迁移 Event Cache Settings 到 Settings 页面
- TBSP-3：集成验证与归档

## 关键设计决策与权衡
- 窗口头部只保留名称和状态
  - 原因：用户已熟悉 TopicBus 独立窗口语义，重复眉题和说明文案价值低。

- 缓存设置迁移到 `Settings` 页面
  - 原因：`Max Events` 和清空缓存更偏偏好设置，放到统一设置页更符合页面职责。
  - 权衡：Settings 页中的 TopicBus 设置是即时生效动作，不纳入全局 App Settings 的“统一保存”流程。

- 继续复用 `topicbus` store
  - 原因：`TopicBusPrefs`、`setMaxEvents()`、`clearEvents()` 已经稳定，不应重复实现。

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

## 潜在影响与回滚方案

### 潜在影响
- 用户需要从 `Settings` 页面调整 TopicBus 缓存上限，不再从 TopicBus `Overview` 修改。
- 独立窗口头部信息更克制，但窗口模式说明从头部移除。

### 回滚方案
- 回退以下文件即可恢复到本次改动前：
  - `frontend/src/windows/TopicBusWindow.vue`
  - `frontend/src/pages/TopicBus.vue`
  - `frontend/src/pages/Settings.vue`

## 子Agent执行轨迹
- 无子Agent
- 原因：本轮改动范围小，且 TopicBus 页面与 Settings 页面调整强耦合，由主Agent直接完成
