# 2026-03-25 Pluggable State Backend

## 变更背景 / 目标

- 现状里 `flow` 的定义持久化绑定本地 `./flows/*.json`，`varstore` 的业务记录只存在内存 `records map`。
- 目标是把这两类业务状态改造成可插拔后端，同时保持 `config_get/config_set` 继续走独立配置层，不与数据库可用性耦合。
- 你确认的本轮口径是：
  - 无数据库时保持默认行为：
    - `flow` 继续本地 JSON
    - `varstore` 继续纯内存
  - 配置 `pg` 时：
    - `flow` 直接把完整 flow 定义存入 PG，而不是存文件路径
    - `varstore` 只持久化 owner 侧权威业务记录
  - capability registry 等共享对象改为显式 `RuntimeDeps` 注入，不再依赖 `cfg` 指针共享作用域

## Related Plan

- `D:\project\MyFlowHub3\docs\plan\plan_archive_2026-03-25_subproto-pluggable-state-backend.md`
- `D:\project\MyFlowHub3\docs\plan\plan_archive_2026-03-25_server-pluggable-state-backend.md`

## Related Requirements

- 无新增或修改的长期 requirement。

## Related Specs

- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`
- `D:\project\MyFlowHub3\docs\specs\management-config-layering.md`

## Related Lessons

- 无。

## Requirements Impact

- `none`

## Specs Impact

- `updated`

## Lessons Impact

- `none`

## 具体变更内容

### `MyFlowHub-SubProto`

- `exec/runtimedeps/deps.go`
  - 新增显式共享 runtime deps 入口，统一承载 `CapRegistry`、`PermConfig` 等共享对象。
- `exec/file/flow/topicbus/varstore/management`
  - 新增 `WithDeps` / `WithOptions` 构造入口。
  - 默认构造函数保持兼容，避免上游一次性大面积破坏。
- `flow`
  - 新增 `Persistence` 接口与默认 `JSON` backend。
  - `Init()/set/delete` 改为通过 persistence 读写定义。
  - 默认无 PG 时仍由 `flow.base_dir` 驱动本地 JSON 行为。
- `varstore`
  - 新增 `Persistence` 接口与默认 `Memory` backend。
  - `Init()` 启动时从 persistence 预热 records cache。
  - owner 本地 `set/revoke` 改为先持久化，再更新 cache / trigger / notify / `up_*` / success resp。
  - `up_set/notify_set/get_resp/subscribe_resp` 等逐跳链路继续只刷新内存 cache，不写持久层。
- 测试
  - 新增显式 deps 构造测试。
  - 新增 `flow` injected persistence 初始化 / 失败语义测试。
  - 新增 `varstore` preload 与 persist-failure 顺序测试。

### `MyFlowHub-Server`

- `modules/defaultset`
  - 默认装配阶段统一创建共享 `runtimedeps.Deps`。
  - `management/exec/file/topicbus/flow/varstore` 改为显式接收共享 deps。
- backend 选择
  - `flow.backend=json|pg`
  - `varstore.backend=memory|pg`
  - `state.pg.dsn`
  - `state.pg.flow_table`
  - `state.pg.varstore_table`
- PG backend
  - `flow` 直接把完整 flow 定义存为 `jsonb`。
  - `varstore` 只存 `(owner, name, value, value_type, visibility)`。
  - schema 采用 `CREATE TABLE IF NOT EXISTS`。
  - 当前版本按操作连接 PG，未额外引入长生命周期连接池。
- 默认 backend 未改
  - `flow.backend` 未配置时仍走 SubProto JSON backend。
  - `varstore.backend` 未配置时仍走 SubProto memory backend。
- 错误路径
  - backend 非法值、配置 `pg` 但缺失 DSN、PG 连接失败均返回明确错误，不做静默降级。

## Plan Task Mapping

- `SUB1` 显式化 runtime deps 与 capability registry 共享
- `SUB2` 为 `flow` 抽取 persistence 接口并保留 JSON backend
- `SUB3` 为 `varstore` 抽取 persistence 接口并固定 owner 写序
- `SUB4` 补齐 repo-local tests and compatibility layer
- `SRV1` 在 `Server` 引入 backend 选择、PG wiring 与 adapter
- `SRV2` 更新 `modules/defaultset` 构造路径与测试
- `DOC1` 更新长期 specs 与 change archive

## 经验 / 教训摘要

- 默认 backend 和业务语义应留在 `SubProto`，`Server` 只做装配和外部 backend 注入，这样 PG 接入不会反向侵蚀协议层。
- `varstore` 的权威写序必须由 owner 节点统一控制；只要持久化失败，就不能继续发送成功响应和广播副作用。
- 把共享对象显式收敛到 `RuntimeDeps` 后，构造路径和测试都更容易推导，避免 `cfg` 被继续当成隐式 service locator。

## 可复用排查线索

- 症状
  - `flow.backend=pg` 或 `varstore.backend=pg` 后启动直接失败
  - `varstore set/revoke` 返回错误且没有成功事件 / notify
  - capability registry 在不同模块间不共享，出现重复注册或查询不到能力
- 触发条件
  - `state.pg.dsn` 缺失或不可达
  - backend 名称配置非法
  - 仍通过旧的 `cfg` 指针共享路径构造 handler
- 关键词
  - `flow.backend`
  - `varstore.backend`
  - `state.pg.dsn`
  - `runtimedeps`
  - `SharedRegistry`
- 快速检查
  - 检查 `modules/defaultset/state_backends.go` 的 backend 选择错误是否命中
  - 检查 `modules/defaultset/runtime_deps.go` 是否把同一个 `CapRegistry` 注入给各子协议
  - 检查 `varstore` owner 写路径是否先命中 persistence，再命中 cache / `up_*` / resp

## 关键设计决策与权衡

- 显式共享对象集中到 `exec/runtimedeps`
  - 好处：消除 `cfg` 指针身份耦合，新增共享对象时也有稳定入口。
  - 代价：构造函数签名和测试初始化需要同步补齐。
- 默认 backend 保留在 `SubProto`
  - 好处：`Server` 无 PG 时继续复用既有 JSON/memory 行为，不需要复制业务默认实现。
  - 代价：`Server` 装配层需要同时支持“注入外部 backend”和“让 repo 默认实现生效”两条路径。
- PG backend 首版按操作连接
  - 好处：接入面最小，生命周期简单。
  - 代价：高频写场景会多一次连接开销，后续如压测发现瓶颈再评估连接池。
- `varstore` 只持久化 owner 权威记录
  - 好处：避免把逐跳 cache、notify、`up_*` 误写成真相源。
  - 代价：非 owner 节点重启后仍只保留运行期 cache 视图。

## 测试与验证方式 / 结果

- `MyFlowHub-SubProto`
  - 在 `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-refactor-pluggable-state` 临时生成 `go.work` 绑定本地 `Core/Proto` 后执行：
  - `go test ./... -count=1 -p 1`
  - 结果：`broker`、`exec`、`file`、`flow`、`management`、`topicbus`、`varstore` 均通过。
- `MyFlowHub-Server`
  - 在 `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-refactor-pluggable-state` 临时生成 `go.work` 绑定本地 `Core/Proto` 与本次 `SubProto` worktree 后执行：
  - `go test ./... -count=1 -p 1`
  - 结果：全量通过。

## 潜在影响与回滚方案

### 潜在影响

- `flow.backend=pg` 或 `varstore.backend=pg` 时，启动 / 首次预热依赖 PG 可达。
- PG table 名当前只接受简单标识符，不支持 schema-qualified 名称。
- `flow` 的 run 状态、scheduler、runtime context 以及 `varstore` 的订阅、pending、writing、逐跳缓存仍然是内存态，重启不会恢复。

### 回滚

1. 回退 `MyFlowHub-SubProto` 中的 `exec/runtimedeps`、`flow/varstore` persistence 抽象与对应测试。
2. 回退 `MyFlowHub-Server` 中 `modules/defaultset` 的显式 deps / backend 选择 / PG adapter 改动。
3. 回退 `repo/MyFlowHub-Server/docs/specs/flow.md`、`repo/MyFlowHub-Server/docs/specs/varstore.md` 与相关 repo-level `docs/change` 索引。
4. 重新执行 `SubProto` 与 `Server` 对应 `go test` 验证默认 JSON/memory 路径已恢复。

## Agent Trace

- 本轮未使用子 Agent。
- 执行轨迹：
  - `SUB1` -> MainAgent -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-refactor-pluggable-state` -> `exec/runtimedeps`, `exec/file/flow/topicbus/varstore/management` -> 通过
  - `SUB2` -> MainAgent -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-refactor-pluggable-state` -> `flow/*` -> 通过
  - `SUB3` -> MainAgent -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-refactor-pluggable-state` -> `varstore/*` -> 通过
  - `SUB4` -> MainAgent -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-refactor-pluggable-state` -> tests / compatibility -> 通过
  - `SRV1` -> MainAgent -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-refactor-pluggable-state` -> `modules/defaultset/*`, `go.mod`, `go.sum` -> 通过
  - `SRV2` -> MainAgent -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-refactor-pluggable-state` -> runtime wiring / tests -> 通过
  - `DOC1` -> MainAgent -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-refactor-pluggable-state` -> repo `docs/specs/*`, repo/root archive docs -> 通过
