# MetricsNode

## Purpose

MetricsNode 是独立 Node 产品，采集设备指标，并在平台允许时执行亮度等有界控制。它不依赖 Desktop 的进程、安装包、身份或 UI；共享 NodeHost 只表示复用运行时组合，不合并 Resource ownership。

## Observable behavior

- CPU、内存、电池、显示等只读指标表现为带 schema、单位和采样元数据的 Variables。
- 可控指标表现为状态 Variable 加独立 Command；Command 输入经过范围与平台能力校验。
- collector 失败不会伪造数值，状态 Variable 暴露 unavailable/stale 和最近错误摘要。
- 采样节奏、启用项与父节点配置持久化；敏感身份材料不进入普通配置或日志。
- 通知频道是同一版本化配置的一部分；节点消费父节点的 `notifications/events` Stream，并由 Windows/Android 宿主呈现系统通知。

## Windows UI

- Windows Wails 产品默认进入“资源状态”，以表格展示真实 `Status.samples`；没有 sample 时显示 idle/等待空态，不生成演示值或 fallback 指标。
- 资源行同时表达 `fresh`、`stale`、`unavailable`、单位、采样间隔和错误摘要；宽屏检查器补充 schema、采样时间、观察时间与写入边界，但不承载唯一操作。
- “采集策略”继续编辑版本化 configuration，包括 enabled、writable、interval 与 notification channels；保存沿用 revision 冲突和 backend validation。
- “连接与身份”继续使用独立 Metrics state directory、Node identity、父节点 trust 和 provisioning permit；permit 仅驻留当前表单，连接成功或应用释放时清空。
- 界面使用 Metrics 自有的 Desktop-aligned token、品牌静态资源和 light/dark 偏好；只共享视觉规范，不依赖 Desktop 私有组件、状态、IPC 或生命周期。
- 状态轮询保持约 1 秒节奏，单次请求不重叠；停止或释放 UI 后取消调度，旧请求不能覆盖新的启停结果。

## Permissions

读指标和改设备状态使用不同权限点。父控子仍需通过显式 control；本地执行端始终执行范围、安全和设备存在性校验。

## Non-goals

- 不保留 VarStore/TopicBus runtime。
- 不复制 Desktop 的通用管理服务或导入 MetricsNode 工作树中的未提交生成绑定。

## Architecture disposition

- 指标状态：每个指标一个 `Variable`，失败时显式进入 `stale` 或 `unavailable`，不再用 `-1` 等伪值表示错误。
- 设备控制：可写指标由独立 `Command` 接收请求；Windows 在受控 platform adapter 内执行，Android 通过容量为 64 的 action queue 交给 Kotlin 执行并回报实际状态。
- 配置：`metrics/config` Variable 加 `metrics/config/update` Command，revision 冲突、平台能力、采样间隔、可写性与通知频道都经过严格校验并原子持久化。
- 通知：旧 TopicBus 订阅被 durable Stream subscription 取代；频道切换会重建订阅，断线恢复由 SDK 管理，宿主只处理已经通过 schema 校验的事件。
- 链路：产品 runtime 只依赖可替换 `link.Driver`；当前命令行和 UI 使用 TCP，Android 与未来链路可在不改变资源模型的情况下替换 driver。
- Host：MetricsNode 拥有自己的 Parent-only NodeHost，在网络启动前注册指标、配置、控制与通知资源；采样更新走 Variable/Registry，远端访问继续走 attached SDK Client 所绑定的同一 Node。
- 身份与权限：持久 Ed25519 身份、父节点 trust 与一次性 provisioning permit 由共享 runtime 管理；父控子仍由权威树执行，平台 adapter 继续执行本地范围和权限检查。

## Canonical source

- 共享 core：`apps/nodes/metrics`。
- Windows collector/actuator：`apps/nodes/metrics/platform/windows`。
- Windows Wails 产品：`apps/nodes/metrics/windows`。
- Android gomobile binding 与 Compose 产品：`apps/nodes/metrics/android`。
- 无 UI Windows 入口：`cmd/mfh-metrics`。

旧 MetricsNode 仓已在完成审计后移除；其固定提交 `c5e2f217afc8063d3e2bb1c6b21fb8fc878925c0`、tree 与未提交生成绑定 hash 保存在 `migration/source-audit.json`。旧 SDK、VarStore、TopicBus 和嵌套 Go modules 均不进入 canonical source。

## Acceptance

collector/actuator 可用 fake 测试；Windows collector 有平台 smoke；Windows UI 与 Android binding 均来自 canonical 契约并可重建。Wails production build、gomobile AAR、Gradle unit/assemble/lint 和真实 TCP 进程 e2e 均是 FM09 门禁。

运行时 ownership、四种拓扑组合与非 owning SDK Client 的稳定约束见 [NodeHost Runtime](../specs/node-host-runtime.md)。
