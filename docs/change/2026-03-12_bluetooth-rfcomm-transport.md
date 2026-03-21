# 2026-03-12 - RFCOMM（Bluetooth Classic）Transport（重大变更）

## 背景 / 目标
- 背景：MyFlowHub 目标是构建“虚拟网络”，可在不同承载（TCP / Bluetooth Classic RFCOMM 等）上复用同一套 Header 编解码、路由与子协议。
- 本次变更目标：
  - 新增 Bluetooth Classic RFCOMM（SPP 风格字节流）承载；
  - 保持 wire 协议不变（仍使用 HeaderTcpCodec 对字节流 encode/decode）；
  - 让 RFCOMM 达到与 TCP 对齐的 listen/dial 能力，并支持按 Header 路由（payload 仅透传）。

## 重大变更说明
- **重大变更**：本次引入新 transport + 新 endpoint scheme（`bt+rfcomm://...`），并在多仓（Core/Server/SDK/Android）形成联动能力。
- 平台差异：Windows/Linux/Android 的 RFCOMM 能力依赖 OS 蓝牙栈，运行期可能受权限、配对状态、BlueZ 可用性影响。

## 端点规范（跨仓一致）
- RFCOMM：
  - `bt+rfcomm://<bdaddr>?uuid=<uuid>&channel=<1-30>&adapter=hci0&secure=true&name=<reserved>`
  - 约定：
    - `uuid` 缺省为 MyFlowHub 默认 UUID；
    - `channel=0` 表示 UUID-first（由平台实现进行解析/连接）；
    - `name` 为未来扩展点：v1 不实现扫描/解析，传入则返回明确错误。
- TCP：
  - `tcp://host:port` 或兼容旧格式 `host:port`

## 具体变更内容（按仓）
### MyFlowHub-Core
- 新增 `listener/rfcomm_listener`：
  - Endpoint 解析/校验（含保留字段 `name`）。
  - `RFCOMMListener`（实现 `core.IListener`）与 `DialEndpoint`（用于 parent link / SDK）。
  - Windows：AF_BTH/RFCOMM（支持 UUID-first + channel override；listen 侧注册服务记录）。
  - Linux：
    - listen：BlueZ D-Bus Profile（`ProfileManager1.RegisterProfile` + `Profile1.NewConnection` 获取 FD）
    - dial：`channel>0` 走内核 RFCOMM socket；`channel=0` 走 BlueZ `Device1.ConnectProfile`（UUID-first）。
  - Android：Provider 注入点（Go interface，gomobile-friendly），由 Java/Kotlin 实现。

### MyFlowHub-Server
- runtime 装配 RFCOMM listener（可与 TCP 并存、可分别开关）。
- ParentEndpoint 支持 `bt+rfcomm://...` 并复用 Core dial。
- CLI flags 增补 RFCOMM 参数（uuid/channel/adapter/insecure）。

### MyFlowHub-SDK
- Session 支持 endpoint connect：`tcp://...` 与 `bt+rfcomm://...`。
- RFCOMM dial 复用 Core 的 `rfcomm_listener.DialEndpoint`（保持帧语义不变）。

### MyFlowHub-Android（含 hubmobile）
- Kotlin 实现 RFCOMM Provider（listen + dial），封装 `BluetoothServerSocket/BluetoothSocket` 为字节流 Pipe。
- `hubmobile.SetRFCOMMProvider(...)` 注入入口；启动侧 best-effort 安装（旧 AAR 不报错）。

## 关键设计决策与权衡
- **Pipe 抽象**：上层管理器持有 `core.IPipe(io.ReadWriteCloser)`，避免暴露 `net.Conn`，便于接入非 TCP 承载。
- **路由开销**：仅要求解析 Header 以进行路由；payload 在无需解包时原样转发（避免对所有帧做 payload 解析）。
- **Linux 实现选择**：
  - UUID-first（推荐）：使用 BlueZ D-Bus Profile/ConnectProfile，让蓝牙栈负责 SDP/通道解析；
  - channel-first（兜底）：直接 RFCOMM socket 连接，减少对 BlueZ 对象存在性的依赖，但需要用户手工提供 channel。
- **扩展点**：预留 `name` 参数用于未来“按设备名扫描/解析到 MAC”，v1 明确拒绝以避免隐式行为。

## 测试与验证
- 自动化（主机平台）：
  - `repo/MyFlowHub-Core`: `go test ./... -count=1`
  - `repo/MyFlowHub-Server`: `go test ./... -count=1`
  - `repo/MyFlowHub-SDK`: `go test ./... -count=1`
  - `repo/MyFlowHub-Android/hubmobile`: `go test ./... -count=1`
- 交叉编译（编译通过即可）：
  - Core 关键测试包已验证 Linux/Android 可编译（`go test -c`）。
- 手工冒烟（依赖真实设备/权限/配对环境）：
  - 任一端 listen，另一端 dial，通过同一 UUID 建立 RFCOMM 连接；
  - 建链后完成至少一条 MyFlowHub 帧的收发（Cmd/Resp）。

## Code Review（结论）
- 需求覆盖：通过（listen/dial、endpoint、Windows/Linux/Android 入口齐全；Server/SDK/Android 端装配完成）
- 架构合理性：通过（Pipe 抽象隔离底层承载；endpoint scheme 分发集中在装配层）
- 性能风险：通过（仅解析 Header；payload 透传；新增的 D-Bus 仅发生在建链阶段）
- 可读性与一致性：通过（端点/Options/错误路径明确；平台实现隔离到 build tags）
- 可扩展性与配置化：通过（adapter/channel/secure 可配；`name` 预留扩展点）
- 稳定性与安全：通过（secure 默认；未注入 Provider/不支持平台给出明确错误）
- 测试覆盖：通过（Go 单测通过；跨平台编译可验证；真机冒烟需环境支持）

## 潜在影响
- Linux 运行期依赖：需要 BlueZ + system D-Bus；UUID-first dial 可能要求目标设备已在 BlueZ 中可见/已配对。
- Android 运行期依赖：需满足 Android 12+ 蓝牙权限要求；未注入 Provider 时会返回明确错误（避免静默失败）。

## 回滚方案
- 回滚到不含 RFCOMM 变更的提交（按仓分别 revert）。
- 运行期可通过关闭 `RFCOMMEnable`/相关 flags 退回 TCP-only（无需改 wire 协议）。

## 与计划映射
- 对应 `worktrees/feat-bluetooth-rfcomm-transport/plan.md`：
  - WF-BT1 ~ WF-BT6
