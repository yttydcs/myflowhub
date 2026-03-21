# Link / Router 内核收敛（重大变更）

## 变更背景 / 目标

本次 workflow 的目标是在**不改变既有 wire 协议语义**（HeaderTcp v2 / Major / SubProto / Action / JSON schema）的前提下，
将 MyFlowHub 从“厚 `IConnection` 模型”向更清晰的分层内核收敛：

- `Pipe`：底层字节流承载
- `Link`：最小链路对象（Pipe + 元数据 + 生命周期）
- `Frame IO`：统一帧读写
- `Header Router`：仅按 Header 做快速路由决策
- `SubProto Handler`：仅在本地处理时才继续做 payload 业务解包

> 该变更为**重大变更**：它引入了新的 Link/Frame/Router 抽象，并把 Core 的收发路径改造成“头部路由优先、payload 按需解包”的内核。

## 具体变更内容

### 新增

1. `MyFlowHub-Core`
   - 新增 `link.go`
     - 引入 `ILink`
     - 引入 `LinkHooks`
     - 引入 `ILinkManager`
     - 引入 `LinkFromConnection` / `ConnectionFromLink` 兼容适配
   - 新增 `frame.go`
     - 引入 `Frame`
     - 引入 `IFrameReader` / `IFrameWriter`
   - 新增 `reader/stream_frame_reader.go`
     - 提供传输无关的流式帧读取器
   - 新增 `process/frame_writer.go`
     - 提供传输无关的帧写入器
     - 对 `HeaderTcpCodec` 保留 `net.Buffers` 快路径
   - 新增 `process/header_router.go`
     - 引入显式 `HeaderRouter`
     - 将路由结果拆分为 `drop / hop_dispatch / local_dispatch / fast_forward / broadcast_children`
   - 新增测试 `process/header_router_test.go`
   - 新增测试 `process/prerouting_test.go`
     - 将 Server docs 中关键语义固化为 Core 自动化回归：
       - 非 auth 的 `SourceID=0` 直接丢弃
       - `SubProto=2` 放行进入 hop/local 处理
       - `MajorCmd` 始终逐跳可见
       - `TargetID=0` 仅向子节点广播，不回父链
       - 远端目标优先命中本地下级节点，命不中再回父链

### 修改

1. `MyFlowHub-Core/connmgr/manager.go`
   - 现有 `Manager` 继续以 `IConnection` 为内部存储，作为过渡兼容层
   - 同时实现 `ILinkManager`
   - 增加 Link 视角的增删查与 node/device 索引操作
   - 保留 direct-node 冲突处理逻辑

2. `MyFlowHub-Core/reader/tcp_reader.go`
   - 改为依赖 `IFrameReader`，从连接读取切换到统一帧读取模型

3. `MyFlowHub-Core/process/senddispatcher.go`
   - 改为通过 `Frame` + `WriteFrame(...)` 输出
   - 不再在调度器内部持有“仅 TCP 的编码写法”

4. `MyFlowHub-Core/process/prerouting.go`
   - 预路由阶段改为显式调用 `HeaderRouter.Decide(...)`
   - 保持 Server docs 约定的语义不变：
     - `MajorCmd`：逐跳进入 handler
     - `MajorMsg/MajorOKResp/MajorErrResp`：`target!=local` 时优先 Core 快速转发
     - `TargetID=0`：仅广播给 children，不表示“向父链发送”

5. `MyFlowHub-Core/connmgr/manager_test.go`
   - 增补 Link 兼容行为与 hook 相关测试

### 未做 / 明确延期

- `LKR-4`（最小 `DecoderRegistry`）本轮**主动延期**。
- 原因：当前内核已经实现“转发路径不解 payload、仅本地 handler 自行解包”的目标；若此时继续把所有 payload decode 再统一抽象，容易扩大变更面并侵入现有子协议实现。
- 结论：保留为下一轮、在各子协议接入边界稳定后再推进。

## plan.md 任务映射

- `LKR-1 - Core：引入 Link 抽象与兼容适配层`
  - 已完成
  - 对应文件：`link.go`、`connmgr/manager.go`、`connmgr/manager_test.go`

- `LKR-2 - Core：Frame IO 与 HeaderRouter 分层`
  - 已完成
  - 对应文件：`frame.go`、`reader/stream_frame_reader.go`、`reader/tcp_reader.go`、`process/frame_writer.go`、`process/header_router.go`、`process/prerouting.go`、`process/senddispatcher.go`、`process/header_router_test.go`、`process/prerouting_test.go`

- `LKR-3 - Server：切换到 Link/Router 内核语义`
  - 本轮以“**保持 Server 源码最小变更** + **用 Server docs 驱动 Core 回归测试** + **本地 worktree 联调验证**”方式完成
  - 原因：Core 兼容层已保证 `Server` 现有 runtime / modules / subproto 装配点无需立即重写，先把语义收口在 Core，能以更小风险完成过渡

- `LKR-4 - 最小 DecoderRegistry`
  - 延期，未纳入本轮代码落地

- `LKR-5 - 回归验证与兼容性检查`
  - 已完成
  - 见“测试与验证方式 / 结果”章节

## 关键设计决策与权衡

### 1) 先引入 Link 视角，但不立即删除 IConnection

- 选择：保留 `IConnection` 作为兼容层，同时让 `Manager` 实现 `ILinkManager`
- 原因：
  - 可避免一次性波及 Server / SubProto / SDK / Android
  - 允许逐模块迁移，而不是“全仓同步爆改”
- 权衡：
  - 短期内存在 `Connection -> Link` 双视角
  - 但兼容层足够薄，风险远小于一步到位替换全部调用点

### 2) 热路径只解析 Header，不统一解 payload

- 选择：在 `PreRouting` 阶段只看 Header 决策，payload 原样保留
- 原因：
  - 满足“中转节点无需解析业务 payload”的目标
  - 降低 CPU 与分配开销
  - 更符合未来 TCP / RFCOMM / 其他字节流 transport 的共用内核
- 权衡：
  - 业务 payload decode 责任仍在各子协议 handler
  - 但边界清晰，避免过早抽象出侵入式统一 decoder

### 3) 保留 TCP 写出快路径

- 选择：`process/frame_writer.go` 对 `HeaderTcpCodec` 保留 `net.Buffers` 快路径
- 原因：
  - 避免新抽象退化现有发送性能
  - 保证新内核不会因为“更通用”而牺牲已有 TCP 路径效率

### 4) 用 Server docs 驱动 Core 路由回归，而不是重写 Server 模块

- 选择：优先把 `Server/docs/*.md` 的关键语义固化为 Core 回归测试
- 原因：
  - 子协议语义本来就由 Core 路由 + handler 边界共同保证
  - 先把语义保护网补齐，比立即改动 Server runtime 更稳妥
- 权衡：
  - `Server` 本轮源码改动为 0
  - 但通过同 workflow 本地联调，已验证 `Server` 能直接运行在新 Core 上

## 性能关键点

- 转发路径仅解析 Header，不解析 payload 业务结构
- `HeaderRouter` 显式化后，避免在 `PreRouting` 中散落多处分支判断
- `SendDispatcher` 统一改为 `Frame` 输出，仍保留 TCP `net.Buffers` 快路径
- `TargetID=0` children-only 与 `target!=local` 快速转发保持在 Core 热路径完成，避免不必要进入子协议 handler

## 测试与验证方式 / 结果

### 自动化测试

1. `MyFlowHub-Core`
   - 命令：`$env:GOWORK='off'; go test ./... -count=1`
   - 结果：通过

2. `MyFlowHub-Server`
   - 方式：使用本 workflow 现有 `worktrees/refactor-link-router-kernel/go.work` 联调到本地 `MyFlowHub-Core`
   - 命令：`go test ./... -count=1`
   - 目录：`worktrees/refactor-link-router-kernel/repo/MyFlowHub-Server`
   - 结果：通过

### 重点验证点

- `SourceID=0` 非 auth 丢弃
- auth (`SubProto=2`) 例外放行
- `MajorCmd` 逐跳可见，不被 Core 自动快转
- `TargetID=0` 仅 children broadcast
- `MajorMsg/MajorOKResp/MajorErrResp` 在 `target!=local` 时优先 Core 快速转发
- 发送路径在新抽象下未丢失 TCP 快路径

## Code Review 结论

- 需求覆盖：**通过**
  - 已实现 Link / Frame / Router 核心抽象，并保持 payload 按需解包
- 架构合理性：**通过**
  - 分层边界比原先清晰，兼容层控制在 Core 内部
- 性能风险：**通过**
  - 转发路径不解析 payload；TCP 发送快路径保留
- 可读性与一致性：**通过**
  - 新概念命名统一，路由决策显式化
- 可扩展性与配置化：**通过**
  - 为 RFCOMM 与未来其他字节流 transport 预留统一入口
- 稳定性与安全：**通过**
  - 关键路由语义与认证前置约束均有显式测试覆盖
- 测试覆盖情况：**通过**
  - Core 全量测试通过，Server 基于本地新 Core 联调测试通过

## 潜在影响

- 这是一次内核抽象升级，后续仓库若直接依赖 `IConnection` 的内部假设，需要逐步转向 `Link/Pipe/Frame` 思维模型
- 当前仍保留 `IConnection` 兼容层，短期内不会强制下游一次性重写
- 若后续继续推进 `DecoderRegistry`，需要再次审视各子协议的 payload decode 边界

## 回滚方案

若需要回滚本次 workflow，可按任务粒度回退：

1. 回退 `LKR-2` 相关提交，恢复旧的 prerouting / senddispatcher / tcp_reader 流程
2. 回退 `LKR-1` 相关提交，移除 `Link` / `Frame` 抽象与兼容层
3. 删除本次新增测试，恢复旧的测试基线

建议按以下优先级回滚：

- 先回退 `process/prerouting.go` / `process/senddispatcher.go` / `reader/tcp_reader.go`
- 再回退 `link.go` / `frame.go` / `connmgr/manager.go`
- 最后回退测试文件