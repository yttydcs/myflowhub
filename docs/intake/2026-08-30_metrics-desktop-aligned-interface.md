# Metrics 对齐 Desktop 视觉语言的独立界面

## Intake Metadata

- Date: 2026-08-30
- Source: 用户在 Metrics 现状审阅后确认高保真设计稿，并进入 `$m-plan`
- Status: accepted for planning
- Product boundary: Metrics 仍是独立产品；只复用 MyFlowHub Desktop 的视觉语言，不复用 Desktop 的运行时、状态或私有组件

## Original Request

观察目前较简陋的 `metrics` 界面，模仿 Desktop 调整界面；先完成设计稿，再为已确认方向制定实施计划。

## Accepted Design Direction

- 使用 44px 顶栏、272px 左侧导航和宽屏 310px 右侧上下文面板，视觉基调与 Desktop 一致。
- 默认进入“资源状态”，低频能力分为“采集策略”和“连接与身份”两个页面。
- 默认浅色，提供独立深色主题；颜色、间距、边框和排版采用 Metrics 自己维护的局部 token。
- 资源状态只展示后端真实 `Status.samples`，并保留 `fresh`、`stale`、`unavailable`、错误和空状态的明确表达。
- 当前选中资源驱动右侧上下文面板；窄窗口隐藏补充面板，但所有核心状态与操作仍在主内容区可用。
- 策略、连接、身份、启动和停止继续调用现有 Wails API，不改变协议和后端语义。
- 品牌图标从 canonical 品牌资产复制到 Metrics 自有 `public/brand`，不从 Desktop 产品目录运行时引用。

## Explicit Non-goals

- 不新增趋势图、历史记录、告警中心、日志中心或模拟监控数据。
- 不实现音量、亮度等本地即时控制；当前后端只公开配置写入，没有对应控制 contract。
- 不把 Desktop 与 Metrics 合并为同一应用，不共享私有状态、IPC 或产品内组件。
- 不在本阶段改造 Android Metrics 界面。
- 不新增 remote，不 push、release 或 publish。

## Acceptance Signals

- 资源、策略、连接三页可通过键盘与指针访问，主题可切换并持久化。
- 连接、停止、保存配置、轮询和错误反馈全部来自现有 API，忙碌状态不会触发重复请求。
- 1440×900 与 1024×768 下无页面级横向溢出；窄窗口没有因隐藏上下文面板而丢失核心能力。
- 生产构建、前端测试、Metrics Go 测试和 Wails 构建通过，真实 GUI 无控制台错误。

## Evidence

- Design discussion: `D:\project\MyFlowHub3\repo\MyFlowHub\design-demos\metrics-desktop-aligned\discussion-brief.md`
- Approved prototype: `D:\project\MyFlowHub3\repo\MyFlowHub\design-demos\metrics-desktop-aligned\metrics-desktop-aligned.html`
- Existing capability record: [Metrics Node](../features/metrics-node.md)
- Execution control plane: [root plan](../../plan.md)
