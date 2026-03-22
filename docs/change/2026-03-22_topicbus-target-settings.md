# 变更归档：TopicBus 默认目标迁移与 Overview 单列化

## 变更背景 / 目标
- 背景：
  - TopicBus 主页面同时承担了状态查看、目标设置和主题管理，信息密度偏高。
  - `Target Node ID` 放在功能页中，容易让用户误以为它只是临时输入，而不是可持久化的默认偏好。
  - `Overview` 左右双栏对当前场景过重，且和用户希望的简洁、友好风格不一致。
- 目标：
  - 将 TopicBus 默认目标迁移到 `Settings`，并做成真正持久化的偏好设置。
  - 将 `Overview` 收敛为单列状态摘要。
  - 将主题列表、订阅操作和独立窗口入口集中到 `Channels`。
  - 保持独立窗口的“上收下发 + 右侧信息/控制”布局，同时减少冗余配置入口。

## 对应 plan 任务映射
- TTSP-1：扩展 TopicBus prefs，持久化默认目标
  - 文件：
    - `app_topicbus.go`
    - `app_topicbus_test.go`
    - `frontend/src/stores/topicbus.ts`
- TTSP-2：收敛 Settings 中的 TopicBus 设置区
  - 文件：
    - `frontend/src/pages/Settings.vue`
- TTSP-3：重构 TopicBus 页面结构
  - 文件：
    - `frontend/src/pages/TopicBus.vue`
    - `frontend/src/i18n/messages/signals.ts`
- TTSP-4：兼容独立窗口与集成验证
  - 文件：
    - `frontend/src/windows/TopicBusWindow.vue`

## 具体变更内容

### 新增
- `app_topicbus_test.go`
  - 新增 TopicBus prefs 相关单元测试，覆盖默认值、持久化结果和非法输入归一化。

### 修改
- `app_topicbus.go`
  - 新增 `topicbus.target_id` 持久化键。
  - `TopicBusPrefs` 增加 `TargetID` 字段。
  - `TopicBusPrefs()` / `SaveTopicBusPrefs()` 支持默认目标的读取、保存和非法值归零。

- `frontend/src/stores/topicbus.ts`
  - 新增配置目标的归一化逻辑。
  - `loadPrefs()` / `savePrefs()` 支持 `targetId`。
  - `setIdentity()` 不再把 `hubId` 回写成已配置目标，只保留运行时 fallback。
  - `savePrefs()` 保存后重新读取归一化结果，避免前端状态与后端持久化不一致。

- `frontend/src/pages/Settings.vue`
  - 在 TopicBus 设置卡片中新增 `Default Target Node ID`。
  - 将 `Default Target Node ID` 与 `Max Events` 合并到同一个保存动作中。
  - 保存失败时回退到持久化值，并给出错误提示。

- `frontend/src/pages/TopicBus.vue`
  - `Overview` 改为单列状态页，不再使用左右两栏。
  - 移除 `Overview` 中的目标输入和旧的快捷入口。
  - `Channels` 统一承载：
    - 已保存主题输入
    - 保存并订阅
    - 移除并退订
    - 刷新远端订阅
    - 重新订阅
    - All / 单频道独立窗口入口
  - 已知频道区新增 `All` 聚合行，主要用于接收所有已知频道的数据。

- `frontend/src/windows/TopicBusWindow.vue`
  - 保留“接收 / 发送 / 侧栏信息 / 控制按钮”布局。
  - 窗口头部只保留名称与连接状态。
  - 发送区移除 `Target Node ID` 输入，避免和 Settings 的默认目标配置重复。
  - route query `targetId` 新增正整数校验，非法值忽略；合法值仍优先覆盖默认目标。

- `frontend/src/i18n/messages/signals.ts`
  - 新增并调整 TopicBus 设置、频道管理、说明文案相关翻译。

## 关键设计决策与权衡

### 1. `Target Node ID` 做成真正的持久化偏好
- 决策：
  - 不仅移动 UI，还在后端 `TopicBusPrefs` 中增加 `TargetID` 持久化。
- 原因：
  - 只改界面位置会让“默认目标”看起来像设置，实际却不会保存，用户心智会错位。

### 2. 空目标继续表示“使用当前 Hub”
- 决策：
  - `state.targetId` 只表示“用户明确配置的默认目标”。
  - 运行时真正发送请求时，由 `resolveTargetId()` 在空值时回退到 `hubId`。
- 原因：
  - 这样既保留了原本的默认行为，又避免把运行时 fallback 误保存成显式配置。

### 3. 主页面和独立窗口都不再鼓励频繁改目标
- 决策：
  - 主页面把目标配置放入 `Settings`。
  - 独立窗口只显示解析结果，不再在发送区重复暴露目标输入。
  - 需要特殊目标时，通过开窗链接的 `targetId` 覆盖。
- 原因：
  - 符合“设置归设置，操作归操作”的分层，也更贴近用户想要的简洁界面。

### 4. 独立窗口仍保留 query 覆盖，但加输入校验
- 决策：
  - route query `targetId` 仍可覆盖默认目标。
  - 非法 query 直接忽略，不污染窗口状态。
- 原因：
  - 满足灵活开窗需求，同时避免坏链接导致运行时错误。

## 性能 / 可扩展性说明
- 性能：
  - 只增加一次轻量的 prefs 字段读写，没有引入新的高频请求或额外轮询。
  - 频道列表和事件统计仍基于已有内存态处理，没有新增重复计算热点。
- 可扩展性：
  - TopicBus 偏好已统一进入 `Settings`，后续可以继续增加更多窗口级默认行为。
  - `resolveTargetId()` 继续作为统一目标解析入口，便于未来扩展其他目标策略。

## 测试与验证方式 / 结果
- Go 单元测试：
  - 命令：`$env:GOWORK='off'; go test ./... -count=1 -p 1`
  - 结果：通过

- 前端依赖准备：
  - 命令：`npm ci`
  - 结果：完成，补齐 worktree 缺失的前端依赖

- Wails 前端绑定生成：
  - 命令：`$env:GOWORK='off'; wails generate module`
  - 结果：完成，生成 `frontend/wailsjs/go/main/App.*`

- 前端生产构建：
  - 命令：`npm run build`
  - 结果：通过
  - 备注：构建有现存 chunk size warning，但不属于本次 TopicBus 变更引入的问题

## 潜在影响与回滚方案
- 潜在影响：
  - 旧 profile 首次读取 `targetId` 时会得到默认值 `0`，行为为“继续使用当前 Hub”，兼容旧配置。
  - 独立窗口不再允许在发送区临时手改目标；如果需要特殊目标，需从主页面传 `targetId` 开窗或修改 Settings。
- 回滚方案：
  - 回退以下文件即可恢复原行为：
    - `app_topicbus.go`
    - `app_topicbus_test.go`
    - `frontend/src/stores/topicbus.ts`
    - `frontend/src/pages/Settings.vue`
    - `frontend/src/pages/TopicBus.vue`
    - `frontend/src/windows/TopicBusWindow.vue`
    - `frontend/src/i18n/messages/signals.ts`

## 子Agent 执行轨迹
- 本 workflow 未使用子Agent。
- 原因：
  - 用户未显式授权使用子Agent。
  - 本次写集集中在 TopicBus 同一组前后端文件，主Agent 直接收敛更安全。
