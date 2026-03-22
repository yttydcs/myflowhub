# Plan - MyFlowHub-Win：TopicBus 页面双段式重构 + 独立频道窗口

## Workflow 信息
- 仓库：MyFlowHub-Win
- 分支：feat/topicbus-window-console
- Base：main
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-window-console\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-window-console\MyFlowHub-Win\todo.md
- 当前阶段：4 归档变更
- 状态：已完成，等待用户确认是否结束本次 workflow
- 规范：
  - D:\project\MyFlowHub3\guide.md
  - D:\project\MyFlowHub3\AGENTS.md（会话内用户提供）

---

## 0) 项目目标与当前状态

### 目标
- 将 Win 端 TopicBus 页面从当前单页大面板重构为类似 VarPool 的双段式体验：
  - `Overview`：主要信息与设置
  - `Channels`：`All` + 各频道列表
- 支持从频道列表打开独立窗口。
- 独立窗口采用“上收下发”的聊天式布局：
  - 上半区：接收新事件
  - 下半区：发送表单
  - 中间可拖拽调整高度
- UI 要简洁、用户友好、低认知负担。

### 已确认需求
- 主页面采用 tab，而不是拆成多个普通路由。
- `All` 仅为前端聚合查看所有已知频道，不表示协议层 wildcard 订阅。
- 频道列表来源采用混合模式：本地保存 topics + 远端 `list_subs`。
- 每个频道打开一个独立窗口，窗口内同时提供接收与发送。
- 独立窗口只显示“打开窗口之后的新事件”。
- 从具体频道打开窗口时，发送 topic 默认锁定到该频道。
- 从 `All` 打开窗口时，发送 topic 允许手工输入。
- 接受 TopicBus 不回显自身 publish 的真实协议行为。

### 当前代码事实
- 当前主页面：`frontend/src/pages/TopicBus.vue`
  - 单页同时承载设置、订阅、发布、事件流、详情。
- 当前 store：`frontend/src/stores/topicbus.ts`
  - 已有本地偏好：topics、`maxEvents`
  - 已有 `topicbus.event` 监听与 200ms 批量 flush
  - 尚无远端 `list_subs` 同步状态与频道视图模型
- 当前窗口模式可复用示例：
  - `frontend/src/pages/Logs.vue`
  - `frontend/src/windows/LogWindow.vue`
  - `frontend/src/pages/ShowcaseCenter.vue`
  - `frontend/src/windows/ShowcaseWindow.vue`
- 当前依赖中无现成 resizable panel 组件；本 workflow 采用轻量自实现分隔条。

### 本 workflow 不做
- 不修改 TopicBus 协议与服务端语义
- 不实现 wildcard / 订阅全部
- 不做历史回放
- 不做主仓库 `repo` 内实现性改动

---

## 1) 需求分析结论（归档）

### 范围
#### 必须
- `#/topicbus` 改为双 tab 页面。
- 第一页展示状态、身份、Target、订阅管理、`maxEvents`、摘要。
- 第二页展示 `All` + 各频道列表。
- 频道列表体现本地保存与远端订阅两种状态。
- 新增 `#/topicbus-window` 独立窗口。
- 独立窗口显示打开后收到的新事件。
- 独立窗口上下分区，可拖拽调整高度。

#### 可选
- 频道行提供精简状态徽标。
- 主页面展示远端订阅概况摘要。

#### 不做
- TopicBus 历史查询
- 跨 profile 窗口同步
- 新增后端存储项

### 验收标准
1. `#/topicbus` 显示双 tab，且结构明显简化。
2. `Channels` 页能显示 `All` 与混合来源的频道列表。
3. 用户可从 `All` 或具体频道打开独立窗口。
4. 独立窗口中：
   - 上半区显示新到事件
   - 下半区可发送
   - 分隔条可调整上下高度
5. 具体频道窗口 topic 被正确锁定。
6. `All` 窗口允许手输 topic。
7. 断线 / 未登录 / popup blocked 时反馈明确。

### 风险
- 本地保存与远端订阅状态不一致时，文案与徽标必须清楚，避免误导。
- 若窗口事件流直接复用主页面缓存，会破坏“只看新事件”的需求；必须使用窗口本地缓冲。
- 分隔条若实现粗糙，可能带来窗口布局抖动或文本选中问题。

### 问题清单
- 阻塞：否

---

## 2) 架构设计结论（归档）

### 总体方案
- 路由层：
  - 保留 `#/topicbus`
  - 新增 `#/topicbus-window`
- 页面层：
  - `TopicBus.vue` 负责 `overview/channels` 两个 tab
  - `TopicBusWindow.vue` 负责频道工作窗口
- store 层：
  - 继续由 `topicbus.ts` 提供偏好、身份、subscribe/unsubscribe/publish
  - 增加远端 `list_subs` 拉取与频道状态汇总能力
- 事件层：
  - 主页面继续使用全局 TopicBus store
  - 独立窗口只订阅 `topicbus.event`，维护窗口私有事件列表

### 模块职责
- `frontend/src/stores/topicbus.ts`
  - TopicBus prefs / identity / subscribe / unsubscribe / publish
  - 远端订阅列表拉取
  - 频道视图模型生成
- `frontend/src/pages/TopicBus.vue`
  - 主页面 tab、设置区、频道页、打开窗口入口
- `frontend/src/windows/TopicBusWindow.vue`
  - 本地事件缓冲、详情查看、发送表单、可调高度分区
- `frontend/src/router/index.ts`
  - 注册新窗口路由

### 数据 / 调用流
1. 主页面进入后：
   - `loadPrefs()`
   - `setIdentity()`
   - session ready 时按需 `resubscribe()`
   - 调用 `ListSubsSimple()` 获取远端状态
2. 频道列表：
   - 本地 `topics`
   - 远端 `remoteTopics`
   - 合并成统一频道视图模型
3. 独立窗口：
   - 从 query 读取 `scope=all` 或 `topic=<name>`
   - mount 后开始监听 `topicbus.event`
   - 仅收集 mount 之后的事件
   - 发送动作仍调用 store 的 `publish()`

### 错误与安全
- `All` 仅为 UI 聚合，不映射协议 wildcard。
- 具体频道窗口 topic 锁定，避免误发。
- `All` 窗口发送前强校验 topic 非空。
- 未连接 / 未登录时发送动作必须阻断并提示。
- 窗口内提示 TopicBus 不回显自身消息。

### 性能与测试策略
- 主页面频道列表不跟随每条事件刷新。
- 独立窗口采用与 store 一致的批量 flush 节奏，控制重渲染频率。
- 独立窗口事件数按 `maxEvents` 截断。
- 分隔条只维护一个比例值，避免复杂布局依赖。

### 可扩展点
- 频道视图模型可扩展 unread、lastSeen、pin 状态。
- 窗口 query 可扩展 readonly、compact、title。
- 分隔比例后续可本地持久化。

### 问题清单
- 阻塞：否

---

## 3) 可执行任务清单（Checklist）

### TBW-1：扩展 TopicBus store 的远端订阅与频道视图模型
- 状态：已完成
- Owner：主Agent
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-window-console\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-window-console\MyFlowHub-Win\todo.md
- 目标：
  - 在 store 中增加远端订阅拉取能力
  - 提供“本地 + 远端”混合后的频道列表视图模型
- 涉及模块 / 文件：
  - `frontend/src/stores/topicbus.ts`
- Write set：
  - 允许修改：`frontend/src/stores/topicbus.ts`
  - 禁止修改：其余文件
- 关键上下文引用：
  - `frontend/src/pages/TopicBus.vue`
  - `internal/services/topicbus/service.go`
  - `repo/MyFlowHub-Server/docs/4-topicbus.md`
- 设计要点：
  - 保持现有 prefs / publish / 事件 flush 逻辑
  - 新增 `remoteTopics`
  - 新增 `refreshRemoteTopics()`
  - 新增频道状态视图模型，至少区分 `localSaved` / `remoteSubscribed`
- 验收条件：
  - 已连接且已登录时可拉取远端订阅列表
  - 无订阅时安全返回空列表
  - 混合频道列表可驱动 UI 渲染
- 测试点：
  - 手工：保存本地 topic 但不订阅，状态正确
  - 手工：远端存在订阅时，状态正确
- 回滚点：
  - 回退 `frontend/src/stores/topicbus.ts`
- 依赖：
  - 无
- 风险 / 注意事项：
  - 不要让频道列表跟随事件流高频重算

### TBW-2：重构 TopicBus 主页面为双 tab 与频道页
- 状态：已完成
- Owner：主Agent
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-window-console\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-window-console\MyFlowHub-Win\todo.md
- 目标：
  - 将 `TopicBus.vue` 改为 `overview/channels` 双 tab 结构
  - 频道页提供 `All` 与各频道的窗口打开入口
- 涉及模块 / 文件：
  - `frontend/src/pages/TopicBus.vue`
- Write set：
  - 允许修改：`frontend/src/pages/TopicBus.vue`
  - 禁止修改：其余文件
- 关键上下文引用：
  - `frontend/src/pages/VarPool.vue`
  - `frontend/src/pages/Logs.vue`
  - `frontend/src/pages/ShowcaseCenter.vue`
  - `frontend/src/stores/topicbus.ts`
- 设计要点：
  - 视觉风格对齐 VarPool
  - 保留主页面必要设置，去掉大块事件流噪音
  - 提供 `Open Window` 操作
- 验收条件：
  - 主页面明显分为两个 tab
  - 主路径操作简洁
  - `Channels` 页可打开窗口
- 测试点：
  - 手工：tab 切换正常
  - 手工：频道页打开 `All` / 具体频道窗口正常
- 回滚点：
  - 回退 `frontend/src/pages/TopicBus.vue`
- 依赖：
  - 依赖 TBW-1 提供频道视图模型
- 风险 / 注意事项：
  - 不要在主页面继续保留原始大事件流块，避免目标漂移

### TBW-3：新增 TopicBus 独立窗口与路由
- 状态：已完成
- Owner：主Agent
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-window-console\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-window-console\MyFlowHub-Win\todo.md
- 目标：
  - 新增 `TopicBusWindow.vue`
  - 新增窗口路由
  - 实现“上收下发 + 分隔条可调高度”
- 涉及模块 / 文件：
  - `frontend/src/windows/TopicBusWindow.vue`
  - `frontend/src/router/index.ts`
- Write set：
  - 允许修改：
    - `frontend/src/windows/TopicBusWindow.vue`
    - `frontend/src/router/index.ts`
  - 禁止修改：其余文件
- 关键上下文引用：
  - `frontend/src/windows/LogWindow.vue`
  - `frontend/src/windows/ShowcaseWindow.vue`
  - `frontend/src/pages/Logs.vue`
  - `frontend/src/stores/topicbus.ts`
- 设计要点：
  - `scope=all` 与 `topic=<name>` 两种窗口模式
  - mount 后才开始收集事件
  - 上半区事件流、下半区发送区
  - 分隔条拖拽时处理 pointer 生命周期与边界值
  - 具体频道锁定 topic；`All` 允许输入
- 验收条件：
  - 路由可独立打开
  - 事件流只包含打开后的新事件
  - 发送区符合频道模式约束
  - 可拖拽调整上下区域高度
- 测试点：
  - 手工：`All` 窗口与具体频道窗口都能工作
  - 手工：拖拽后布局稳定
  - 手工：popup blocked 时主页面提示明确
- 回滚点：
  - 回退新增窗口页与路由注册
- 依赖：
  - 可与 TBW-2 并行，但最终需主Agent集成
- 风险 / 注意事项：
  - 不要污染全局 store 事件列表

### TBW-4：集成、回归与计划内修正
- 状态：已完成
- Owner：主Agent
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-window-console\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-window-console\MyFlowHub-Win\todo.md
- 目标：
  - 集成 TBW-1 ~ TBW-3
  - 执行构建与必要回归
  - 修复集成问题
- 涉及模块 / 文件：
  - 仅允许修改前述任务文件中为修正集成问题所必须的内容
- Write set：
  - 允许修改：
    - `frontend/src/stores/topicbus.ts`
    - `frontend/src/pages/TopicBus.vue`
    - `frontend/src/windows/TopicBusWindow.vue`
    - `frontend/src/router/index.ts`
  - 禁止修改：计划外文件
- 验收条件：
  - 相关页面与窗口可正常运行
  - 无明显交互回退
- 测试点：
  - `cd frontend && npm run build`
  - 如构建链允许：`wails build -debug -skipembedcreate -nopackage`
  - 手工冒烟：主页面、频道页、窗口、发送、接收、分隔条
- 回滚点：
  - 回退本 workflow 全部改动
- 依赖：
  - 依赖 TBW-1 ~ TBW-3
- 风险 / 注意事项：
  - 如果遇到仓库既有构建问题，需明确区分“本次引入”与“基线已有”

### TBW-5：Code Review（强制）
- 状态：已完成（通过）
- Owner：主Agent
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-window-console\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-window-console\MyFlowHub-Win\todo.md
- 目标：
  - 按 AGENTS 规则输出逐项审查结论
- 涉及模块 / 文件：
  - 全部本 workflow 改动文件
- 验收条件：
  - 明确给出通过 / 不通过
  - 若不通过，返回 3.2 修正
- 测试点：
  - 复核需求覆盖、架构、性能、可读性、扩展性、稳定性、安全、测试覆盖
- 回滚点：
  - 不适用（Review 阶段不写业务代码）
- 依赖：
  - 依赖 TBW-4

### TBW-6：归档变更（强制）
- 状态：已完成
- Owner：主Agent
- Worktree：D:\project\MyFlowHub3\worktrees\topicbus-window-console\MyFlowHub-Win
- Plan 路径：D:\project\MyFlowHub3\worktrees\topicbus-window-console\MyFlowHub-Win\todo.md
- 目标：
  - 在当前 worktree 下补齐 `docs/change/` 归档文档
- 涉及模块 / 文件：
  - `docs/change/YYYY-MM-DD_topicbus-window-console.md`
- 验收条件：
  - 文档包含背景、目标、变更内容、任务映射、设计权衡、测试结果、影响与回滚
- 测试点：
  - 文档与实际改动一致
- 回滚点：
  - 删除本次变更文档
- 依赖：
  - 依赖 TBW-5 通过

---

## 4) 并行性评估（供 3.2 使用）

### 结论
- 可安全拆分为两个不重叠写集的实现任务，再由主Agent做集成：
  - 一条线：`topicbus.ts`
  - 一条线：`TopicBus.vue` + `TopicBusWindow.vue` + `router/index.ts`
- 但是否派发子Agent，必须在进入 3.2 时再次执行一次并行性评估并记录理由。

### 预设拆分候选
- 候选 A：
  - Task ID：TBW-1
  - Write set：`frontend/src/stores/topicbus.ts`
- 候选 B：
  - Task ID：TBW-2 + TBW-3
  - Write set：
    - `frontend/src/pages/TopicBus.vue`
    - `frontend/src/windows/TopicBusWindow.vue`
    - `frontend/src/router/index.ts`

### 主Agent职责（不得外包）
- 并行性最终判断
- 文件所有权确认
- 结果集成
- 冲突处理
- 最终验收
- Review 与归档

---

## 5) 执行记录
- 2026-03-21：完成初始化（独占分支 + worktree + 文档骨架）
- 2026-03-21：完成需求分析并消除阻塞
- 2026-03-21：完成架构设计分析
- 2026-03-21：完成 3.1 计划文档，可进入 3.2
- 2026-03-21：完成 TBW-1 ~ TBW-4，实现主页面双 tab、频道状态模型、独立窗口与路由
- 2026-03-21：完成 3.3 Code Review，结论：通过
- 2026-03-21：完成 4 归档变更，新增 `docs/change/2026-03-21_topicbus-window-console.md`
