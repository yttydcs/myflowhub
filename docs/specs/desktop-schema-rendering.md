# Desktop Schema Rendering

## Status

Current。定义 Desktop Resource Widget 的数据 schema、renderer 选择、用户展示偏好、生成契约与安全 fallback。
布局拓扑继续由 [Desktop Resource Workspace v3](desktop-resource-workspace-v3.md) 定义。

## Ownership

- Resource provider 拥有数据语义：类型、约束、格式、capability 和运行时校验。
- Desktop 拥有 renderer 实现、兼容性过滤、默认排名、响应式密度和允许保存的展示设置。
- 用户可在兼容 renderer 中选择；选择只改变 View，不调用 Resource operation。
- authority 和 Resource owner 始终拥有权限、payload、revision、session 和路径的最终裁决。

provider 的 presentation metadata 是非强制默认 hint，不得携带 HTML、JavaScript、任意 CSS、远程组件代码
或削弱 schema 的指令。

## Schema Source

标准 Variable 使用平台 [统一变量契约](unified-variable-resource.md)：read 与订阅初始快照直接携带有界 definition，后续更新按 instance_id 复用定义。定义的语义 ID 可选，不能用内置列表覆盖远端有效定义。变量只支持当前 type_version=2 和标准 envelope，不保留旧 renderer、旧 payload 解码或按 Metrics 路径选择组件。

其他资源的 command/event/collection schema 继续使用 `protocol` canonical definition 与 `go generate ./sdk/bindings` 生成的内置描述；未知定义仍显示明确原因及 raw 数据。其他资源远程 schema 分发是独立延期范围。远程 executable renderer 不属于扩展点。

## Bounded Data Vocabulary

支持：

- `null`、`boolean`、`integer`、`number`、`string`、`object`、homogeneous `array`；
- object properties、required、stable order；
- enum；
- numeric minimum、maximum、multiple/step；
- string min/max length、pattern 与 allowlisted format；
- array min/max items；
- title、description、unit、precision、readOnly、writeOnly/sensitive 等 annotation。

标准 Variable definition 的语义 ID 可选；其他注册定义有稳定 schema ID。属性/枚举排序确定，且通过最大深度、字段、枚举、字符串和数组限制。
recursive ref、未知 keyword、无法忠实表达的 union/conditional 或超限 definition 返回 `unsupported/invalid`
并使用 raw fallback，不做宽松猜测。

## Resolver And Registry

`SchemaResolver` 返回 `resolved | missing | unsupported | invalid`，同时保留 schema ID、provider ID 和可操作
reason。provider 顺序固定且结果确定；phase 1 只有 generated built-in provider。

每个 renderer definition 至少包含：

- versioned ID；
- value/operation/event/file/structured mode；
- schema/capability compatibility predicate；
- deterministic rank；
- minimum pane size 与 compact/normal/expanded variants；
- allowlisted settings validator。

选择顺序：

1. 已保存且兼容的明确 renderer；
2. `automatic` 的最高排名兼容 renderer；
3. structured/raw fallback。

已保存 renderer 不兼容时必须显示原因并临时 fallback，不得覆盖已保存 preference。不自动迁移旧 renderer alias。

## Renderer Matrix

| Shape / mode | Read | Writable / action |
| --- | --- | --- |
| boolean | text/status | staged switch/checkbox |
| enum | label/badge | select/segmented choice |
| bounded number | number/stat/progress/gauge/trend | number/stepper/slider |
| unbounded number | number/stat/trend | number/stepper |
| string | text/code | single-line/multiline/code textarea |
| date/time/duration | localized value | compatible date/time/duration input |
| object | definition list/grouped display | generated grouped form |
| homogeneous array | list/table | bounded repeatable rows |
| unsupported | JSON tree/raw | Advanced JSON when operation capability exists |

一个 renderer 不能忠实执行 provider constraint 时不得出现在兼容列表。slider 必须同时提供 keyboard 或数字
输入路径。切换 renderer 不修改 value 或丢弃 draft。

## Resource Controllers

Variable 的 Inspector 手动提交与 View 自动提交、加载态及草稿所有权按 [统一变量规格](unified-variable-resource.md) 执行；两种界面共享会话，但提交意图属于产生该编辑的组件。

transport lifecycle 与 presentation 分离：

- Variable controller 管理 snapshot/subscription、revision、draft、Reset/Apply、refresh 和 conflict；
- Operation controller 管理 input draft、显式 Execute、busy/error 和 typed output；
- Event controller 管理 subscription、bounded buffer、pause/filter/clear/rate/autoscroll/gap/expired；
- File controller 管理 native picker、destination、in-flight state 和 session error。当前 `UploadFile` binding 是
  单次阻塞调用，不暴露 transfer handle，因此 phase 1 不伪造逐块进度或单次上传取消；已有
  `file/progress` Stream 与 `file/transfers` Variable 仍可作为独立 Widget 展示 owner 报告的进度。绑定提供
  transfer handle/cancel 后，再把它们合并进同一上传 controller。

presentation component 只读写 controller state，不直接创建 Wails subscription/session。unmount、Resource
变化、Profile 切换和连接退出必须清理 effect。

## First-party Structured Adapters

adapter 使用 schema ID 或 explicit renderer ID 注册，不使用 Resource path：

- catalog：searchable resource table/details；
- topology/health/config：hierarchy/table、status、grouped values；
- audit/notification/flow events：timeline/log；
- file transfers/progress：table/progress/status；
- flow definitions/runs：table/details；
- management/flow operations：generated form；不支持的 opaque/conditional field 保留 Advanced JSON。

## View Persistence

View document 保持 version 3。`ViewWidget.renderer` 保存 versioned renderer ID；`settings` 只保存 bounded、
versioned、allowlisted、非秘密 presentation metadata。禁止保存 value、draft、payload、event body、文件内容、
permission、permit、private key 或 credential。

phase 1 renderer 的 settings allowlist 为空；`undefined` 与空 object 有效，其他 settings 被领域 validator
拒绝。切换 renderer 会清除旧 renderer settings，避免把不兼容或可能含数据的状态带入新 renderer。

renderer 切换标记 View dirty。pane resize 只改变 runtime density，不写 View；用户明确修改 renderer setting
才持久化。

## Accessibility And Responsive Behavior

- 只有多个兼容 renderer 时显示 presentation selector，并提供 programmatic label。
- field 有 label、description、required、error association；状态不能只依赖颜色。
- read-only 与 writable 控件在语义和视觉上区分。
- compact 保留主要值/状态与一个安全动作；normal 显示主要 metadata/control；expanded 显示 history/details/raw。
- 使用 pane container size；ResizeObserver 更新通过 animation frame 合并，不产生 API traffic 或 View 保存。
- 低于 renderer minimum 时 Workspace 滚动或使用兼容 fallback，不隐藏关键动作。

## Failure Behavior

- schema missing/unsupported/invalid、renderer incompatible、malformed settings、Forbidden、revision conflict、
  gap/expired、disconnect 和 session failure 分别显示。
- 客户端 validation 只改善反馈；server/owner error 必须原样保留可操作上下文。
- 写入失败保留 draft；成功后重新读取 authoritative value/revision。
- unknown schema 绝不获得推断出来的 mutating control。

## Verification

- protocol definition validation、schema-ID coverage、generation freshness 和 fixture parity；
- resolver/registry/settings 纯单元测试；
- Variable pointer/keyboard、draft/conflict、read-only 和 renderer-switch-without-operation；
- operation form/output/fallback；event bounded buffer/gap/cleanup；File picker cancel/in-flight/error；
- View save/reopen/legacy alias/incompatible fallback；
- light/dark、nested compact/normal/expanded panes、Forbidden/offline；
- Vitest、TypeScript/Vite、`GOWORK=off go test ./...`、generated check、Wails production build 和真实 GUI smoke。

## Related Docs

- [Desktop feature](../features/desktop.md)
- [Desktop requirements](../requirements/desktop-resource-workspace.md)
- [Resource platform v2](resource-platform-v2.md)
- [Provider schema/Desktop renderer decision](../decisions/2026-08-30_provider-schema-desktop-renderer-ownership.md)
- [Discussion intake](../intake/2026-08-30_desktop-schema-driven-resource-widgets.md)
