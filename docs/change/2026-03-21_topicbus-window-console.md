# 变更归档：TopicBus 页面双段式重构 + 独立频道窗口

## 变更背景 / 目标
- 背景：
  - 现有 `TopicBus` 页面将设置、频道、发布、事件流和详情全部堆在同一页，主路径信息密度过高。
  - `VarPool` 已经演进为“总览/控制 + 分段内容”的结构，TopicBus 也需要类似的可用性提升。
- 目标：
  - 将 `TopicBus` 主页面重构为更简洁的双 tab 结构。
  - 提供 `All` + 各频道列表。
  - 支持从频道页打开独立窗口，窗口内同时进行接收与发送。
  - 独立窗口仅显示打开后的新事件，并支持上下分区拖拽调整高度。

## 具体变更内容

### 修改
- `frontend/src/stores/topicbus.ts`
  - 新增远端订阅状态：`remoteTopics`、`remoteSyncedAt`
  - 新增 `refreshRemoteTopics()`
  - 新增 `channelItems()`，合并本地保存 topics 与远端 `list_subs` 结果
  - 新增事件工具：
    - `normalizeTopicBusEvent()`
    - `formatTopicBusTimestamp()`
  - `subscribe` / `unsubscribe` / `resubscribe` 成功后同步更新远端订阅视图

- `frontend/src/pages/TopicBus.vue`
  - 将页面改为 `Overview` / `Channels` 双 tab
  - `Overview`：
    - 保留 Target、topic 列表管理、Resubscribe、`maxEvents`
    - 新增远端订阅同步入口与状态摘要
    - 去掉原本占位很大的事件流 / 详情块
  - `Channels`：
    - 展示 `All` 与各频道列表
    - 列表项展示 `Saved` / `Active` 状态
    - 提供 `Open Window` 和快速 `Subscribe / Unsubscribe`

- `frontend/src/router/index.ts`
  - 新增窗口路由：`/topicbus-window`
  - 使用 `layout: "window"` + `windowMode: "full-bleed"`

### 新增
- `frontend/src/windows/TopicBusWindow.vue`
  - 独立频道窗口
  - 上半区：接收事件流
  - 下半区：发送表单
  - 中间拖拽条：调整上下区域高度
  - `All` 模式允许输入 topic
  - 具体频道模式锁定 topic
  - 仅收集窗口打开后收到的新事件

### 删除
- 无文件删除
- 但主页面删除了旧的“大事件流 + 详情”页面结构

## 对应计划任务映射
- TBW-1：扩展 TopicBus store 的远端订阅与频道视图模型
- TBW-2：重构 TopicBus 主页面为双 tab 与频道页
- TBW-3：新增 TopicBus 独立窗口与路由
- TBW-4：集成与回归
- TBW-5：Code Review
- TBW-6：归档变更（本文档）

## 关键设计决策与权衡
- 主页面不再承担实时消费主视图
  - 原因：主页面应聚焦设置与频道管理，把高频事件流放到独立窗口，降低认知负担和重渲染噪音。

- `All` 仅作为 UI 聚合视图，不做协议 wildcard
  - 原因：当前 TopicBus 协议只支持精确 topic 订阅，不能在 UI 层伪造“订阅全部”的语义。

- 独立窗口使用本地事件缓冲，而不是复用全局缓存列表
  - 原因：必须严格满足“只显示打开后的新事件”。
  - 性能点：窗口内同样使用 200ms 批量 flush，避免高频 publish 时频繁重排。

- 频道列表采用“本地 + 远端”混合视图模型
  - 原因：仅看本地 topics 无法知道当前连接实际是否生效；仅看远端 `list_subs` 又无法体现用户保存意图。
  - 扩展点：后续可继续扩展 unread、pin、lastSeen 等字段，而不必改动页面结构。

- 分隔条采用轻量自实现，而不引入新依赖
  - 原因：当前前端依赖中没有现成 splitter 组件，本次只需要一个稳定的上下拖拽场景，自实现更小、更可控。

## Code Review 结论
- 需求覆盖：通过
- 架构合理性：通过
- 性能风险：通过
  - 未发现明显 N+1、重复计算热点或不必要 I/O
  - 频道列表不随每条事件重算布局
  - 独立窗口事件流做了本地截断与批量 flush
- 可读性与一致性：通过
  - 主页面结构对齐 VarPool
  - 窗口路由与 Logs / Showcase 模式保持一致
- 可扩展性与配置化：通过
  - 频道视图模型与窗口 query 可继续扩展
- 稳定性与安全：通过
  - `All` 发送强制 topic 输入
  - 具体频道发送锁定 topic
  - 明确提示“不回显自身消息”
- 测试覆盖情况：通过
  - 以构建链路与最小回归为主
- 子Agent治理与审计：通过
  - 本 workflow 未使用子Agent
  - 原因：当前运行规则要求必须有用户显式授权后才能派发子Agent

## 测试与验证方式 / 结果
- `npm ci`
  - 结果：通过
  - 说明：worktree 初始缺少 `frontend/node_modules`

- `wails generate module`
  - 结果：通过
  - 说明：为当前 worktree 生成 `frontend/wailsjs`，否则前端构建无法解析既有绑定引用

- `cd frontend && npm run build`
  - 结果：通过
  - 说明：产物生成成功；仍存在 Vite 的大 chunk 警告，但不是本次改动新增问题

- `GOWORK=off go test ./... -count=1 -p 1`
  - 结果：通过

- `GOWORK=off wails build -debug -skipembedcreate -nopackage`
  - 结果：通过
  - 说明：`build/bin/myflowhub-win.exe` 已生成

- UI 手工冒烟
  - 结果：未执行
  - 原因：本轮未启动配套 server/session 环境，未进行真实交互验证

## 潜在影响与回滚方案

### 潜在影响
- TopicBus 主页面的操作路径发生了明显变化，老用户需要适应新的 tab 结构。
- 独立窗口引入后，推荐使用方式从“主页面直接看事件流”转变为“频道页打开专用窗口”。

### 回滚方案
- 回退以下文件即可恢复到原始实现：
  - `frontend/src/stores/topicbus.ts`
  - `frontend/src/pages/TopicBus.vue`
  - `frontend/src/router/index.ts`
  - `frontend/src/windows/TopicBusWindow.vue`

## 子Agent执行轨迹
- 无子Agent
- 原因：本轮未获得用户显式授权委派；由主Agent单独完成全部 Task ID 并统一集成与验收
