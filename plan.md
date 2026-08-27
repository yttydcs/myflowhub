# Plan - MyFlowHub vNext 全量迁移与最终切换

## Workflow Information

- Repo: `D:/project/MyFlowHub3`
- Branch: `refactor/vnext-full-migration`
- Base: `master@87cd7b655b3669abfadf8c0612265d4e5029cdde`
- Project Root: `D:/project/MyFlowHub3`
- Docs Root: `D:/project/MyFlowHub3/worktrees/vnext-full-migration/docs`
- Code Repos:
  - canonical write target: `D:/project/MyFlowHub3/worktrees/vnext-full-migration`
  - legacy sources: local `D:/project/MyFlowHub3/repo/MyFlowHub-*` checkouts retired and removed; recovery metadata in `migration/`
- Active Worktree: `D:/project/MyFlowHub3/worktrees/vnext-full-migration`
- Current Stage: `4.0 archive complete; local merge and worktree cleanup authorized by $m-archive`
- Owner: main agent
- Compatibility: clean break；不保留旧 module path、SubProto API 或 legacy wire compatibility bridge

## Stage Records

### Execute - Approved

- 用户通过显式调用 `$m-execute` 批准执行 FM00–FM16。
- DX01–DX05 继续排除；不推送、不发布、不归档、不删除旧仓或 main 工作树内容。
- 当前主机策略不允许主动委派，执行由 main agent 按门禁顺序串行完成。

### Post-execution - DX04a legacy repository retirement

- 用户在 FM00–FM16 完成后另行显式授权移除 `D:/project/MyFlowHub3/repo/MyFlowHub-*`；该授权不包含 main checkout dirt、canonical worktree、remote archive 或发布操作。
- 旧仓文档完成 777 文件审计：193 个 exact duplicate，584 个独有文件经分类后仅提炼仍适用于 vNext 的长期 lessons；旧 SubProto/VarStore/多仓计划不作为当前文档复制。
- 10 个本地旧仓已移除，恢复 remote/commit/tree、MetricsNode 29 个旧生成绑定 hash、变更证据和回滚方式见 `migration/` 与 `docs/change/2026-08-27_legacy-repository-retirement.md`。
- 删除后 `go test ./internal/archtest ./internal/migrationtest`、Markdown 相对链接检查和 `git diff --check` 通过。

### Execute - FM00-FM02

- FM00 complete：固定 10 个来源 commit/tree/file count，记录所有旧能力和构建入口目标处置，并对 MetricsNode 29 个 dirty 路径保存逐文件 SHA-256；read-only audit 通过。
- FM01 complete：建立 Hub、Desktop、Android、MetricsNode、ClipboardNode、File、Flow dossier，以及 lifecycle、catalog、notification、file、flow 稳定 specs。
- FM02 complete：新增 provisioning/catalog/management/notification/file/flow v1 payload、统一尺寸/版本/输入校验和 6 组跨语言 golden fixtures。
- Lightweight validation：`GOWORK=off go test ./protocol ./internal/protocoltest ./internal/archtest ./internal/migrationtest` 通过；`migration/audit-sources.ps1` 验证 10 个旧仓未漂移。

### Execute - FM03-FM05

- FM03 complete：版本化原子 keystore、稳定 identity、持久 trust/policy generation、父签发的单次 permit 与 revoke 已接入 Node join；损坏 identity 明确失败且不覆盖。
- FM04 complete：ParentSupervisor 提供错误分类、退避、状态/世代与同父自动重连；100 次重连和 reparent 禁止旧父恢复通过；durable Variable 订阅与 policy generation 失效已验证。
- FM05 complete：QUIC 使用 TLS 1.3 与单 stream preface 接入统一 Driver；RFCOMM 迁移固定 Core commit 的 Windows/Linux/Android native provider 并增加 MTU/short-write wrapper。
- Lightweight validation：TCP/QUIC/RFCOMM contract 通过；RFCOMM Windows test 与 Linux/Android cross-build 通过；`go test -race ./runtime/... ./sdk/go/... ./transport/...` 通过（恢复竞态修正后 focused race 重复 5 次）。

### Execute - FM06

- FM06 complete：生产 Hub 使用 durable identity/trust/policy/admission/settings，支持统一 Driver 的多 Listener 和失败全回滚；`mfh-hub` 默认不再生成临时身份或启用 allow-all。
- Management 以 `system/catalog`、topology/health/config Variables、audit Stream、permit/revoke/config Commands 表达；真实多层拓扑保存 immediate parent，远端 allow/deny 在策略裁决边界审计且不记录 payload、key 或 permit 正文。
- `mfh-admin` 通过持久客户端身份和父公钥/permit 接入 Hub，使用普通 Subscription/Command 读取或管理资源，不建立本地旁路管理协议。
- Lightweight validation：restart identity、multi-listener、partial rollback、config corruption、remote policy deny/allow audit、permit/config/revoke tests 和 `go test -race ./feature/management ./host/... ./runtime/node ./runtime/resource ./runtime/tree` 通过；`go build ./cmd/...` 通过。

### Execute - FM07

- FM07 complete：Notification 使用 `notifications/events` Stream + publish Command；File 使用 transfers Variable、progress Stream 与 offer/chunk/complete/cancel Commands；Flow 使用 definitions/runs Variables、events Stream 与 create/update/run/cancel/archive Commands。
- File 已验证 ordered chunk、重复块幂等、chunk/final SHA-256、zero-length、gap/ownership、路径边界、临时文件原子 rename、取消、自动过期和重启 stale partial 清理。
- Flow definition/run 使用版本化原子 store；DAG 拒绝 cycle，运行有 dedupe、全局/单 flow 并发限制、取消、bounded output、temporary-only bounded retry/backoff、archive 和 restart→interrupted 语义。
- 为避免 Flow host 成为隐式超级用户，MFH3 envelope 增加由 authority root 裁决的独立 Principal；Source 继续负责链路与回包，child 禁止自行委托，root delegated invoke/subscribe 对原始 initiator 做策略校验。
- Lightweight validation：`GOWORK=off go test ./...` 和 `go test -race ./feature/... ./host/hub ./runtime/auth ./runtime/command ./runtime/node ./protocol` 通过；delegated deny/allow 测试曾发现并修复本地 Source 绕过与 retryable 回包丢失。

### Execute - FM08

- FM08 complete：Go SDK 提供稳定 catalog/snapshot/subscribe/invoke、Management/Notification/File/Flow convenience、typed schema/remote errors 和可观察 ParentSupervisor connection state；durable subscription 以 SDK Event 暴露并增加首次远端订阅 Ready 门禁，取消与重连不泄露 wire phase。
- 跨语言 canonical contract 由 protocol 常量生成并有 stale diff test；gomobile 使用 callback subscription，Wails 使用无 interface 参数的 desktop polling adapter。纯 `gobind` Java/JNI 生成与 Wails JS/TS 生成通过，且修复了直接绑定 callback interface 会产生悬空 `models` import 的问题。
- 持久 identity/trust/policy/admission 状态下沉到 `runtime/auth.OpenState`，Hub 在其上叠加 host settings，bindings 不反向依赖 host；单 module 和 import boundary architecture gate 保持不变。
- Lightweight validation：`GOWORK=off go test ./...`、`go test -race ./sdk/go ./sdk/bindings/... ./runtime/auth ./host/config`、binding TCP/permit/default-deny/File/Flow tests 和 10 个 read-only source audit 通过。完整 Android AAR 构建仍由 FM12/FM14 处理，本机现有 NDK 指向已不支持的 API 16；FM08 所需 Java/JNI generation smoke 已通过。

### Execute - FM09

- FM09 complete：MetricsNode core 以 typed Variables 暴露 CPU、内存、电池、网络、音量、亮度和手电筒状态；可控指标使用独立 Commands，collector 错误保留最近真实值并显式进入 stale/unavailable，不再使用伪数值。
- Windows product 迁入 canonical Wails app，真实 collector/actuator 覆盖 CPU、内存、电池、网络、Core Audio 与显示亮度；Wails 边界只暴露版本化 JSON。Android product 使用 canonical gomobile Client、Compose 前台服务和容量 64 的 action queue，Kotlin 执行平台动作后回报实际状态。
- 旧 lightweight notify 行为已由父节点 `notifications/events` durable Stream 取代；频道与采样配置同 revision 原子持久化，Windows 使用隐藏 PowerShell presenter，Android 使用系统 notification channel。旧 SDK、VarStore、TopicBus 和嵌套 Go modules 均未带入。
- Build validation：Wails binding generation + production build、固定 `x/mobile@v0.0.0-20250520180527-a1d90793fc63` 的 4-ABI Android AAR、Gradle `testDebugUnitTest assembleDebug lintDebug` 均通过；Gradle wrapper 已从 canonical project 生成。
- Runtime validation：真实 Windows collector read-only smoke、fake actuator/config/restart/notification tests、TCP persistent Hub→独立 `mfh-metrics.exe` 进程 e2e、`GOWORK=off go test ./...` 和 `go test -race ./apps/nodes/metrics/...` 通过。10 个固定迁移源审计通过，旧 MetricsNode 工作树既有 dirty binding paths 未被读取或修改。

### Execute - FM10

- FM10 complete：ClipboardNode 使用 `clipboard/status`、`clipboard/config` Variables，`clipboard/events` Stream，以及 send/apply/config-update/history-clear Commands；接收端按显式 peer NodeID 建立 durable subscription，不再使用 TopicBus、VarStore、SubProto 或旧 SDK。
- 隐私与可靠性：正文只进入受保护事件、平台写入队列和可配置的本地有界历史；status/config/log/error 只保存 ID、大小与 hash 前缀。256 KiB 协议上限、64 条 pending/mobile action queue、history 条数/字节/TTL、event/hash 去重和 apply→watch loop suppression 均有测试。
- 产品迁移：Windows 使用真实 Win32 Unicode clipboard adapter 和 512 KiB 有界 NDJSON Flutter bridge；Android 使用 canonical gomobile JSON/primitive binding、ClipboardManager observer/write completion；Flutter 同源构建 Windows/Android/Web，Web 明确为 UI preview 而不伪装原生 runtime。
- Build validation：Flutter 3.47.1 `analyze`/widget test、Web/Windows release、arm64/x86_64 split Android debug APK、Android `lintDebug`、双 ABI gomobile AAR 和 Windows release 目录 bridge identity smoke 通过。
- Runtime validation：双 Node 经同一父树完成 sibling Stream 同步，真实 persistent TCP Hub→独立 `mfh-clipboard.exe` Command/Stream e2e 通过；`GOWORK=off go test ./...`、`go vet ./...`、Clipboard race 和 10 个固定迁移源审计通过。迁移清单 7 个 Clipboard capability/build entry 全部进入 verified。

### Execute - FM11

- FM11 complete：Desktop 重新实现为单一 canonical SDK 会话；身份、父信任、managed TCP 重连、catalog、Variable snapshot、durable Stream subscription 和 Command invoke 通过一套 Wails binding 暴露，不再复制 TopicBus、VarPool、Management 等旧 service wrapper。
- 产品表面：Wails 前端保留节点树、动态资源浏览、MetricsNode、ClipboardNode、File、Flow、权限/准入、设置和日志入口。拓扑与权限结果来自 Hub management resources；文件上传使用有界分块和摘要；剪贴板正文不进入操作日志。
- 本地存储：`settings.json` 固定 version 1、原子 replace、profile 路径校验和尺寸门禁；未知/损坏版本明确要求显式 reset。reset 只重置 UI/连接设置，不隐式删除持久 identity/trust state。
- 自动化表面：`apps/desktop/mcp` 使用 JSON-RPC/MCP 的 identity/status/catalog/snapshot/Command 工具替换旧工具服务，Command 仍需独立 `--allow-write` 本地门禁；stdio 保持 stdout 纯 JSON。
- Validation：Go service/MCP tests 与 vet、Wails binding generation、Vitest、TypeScript/Vite production build、Windows amd64 Wails production build、CLI identity、MCP initialize 和隐藏 GUI 真实进程 smoke 均通过；源码扫描无旧 module、TopicBus、VarPool 或 SubProto import。迁移清单 6 个 Desktop capability/build entry 全部 verified。

### Execute - FM12

- FM12 complete：`sdk/bindings/android` 直接包装 canonical identity、managed parent connection、catalog、Variable snapshot、durable Stream callback、Command 和 typed File upload；同时提供可承载 management/notification/File/Flow 的 persistent local Hub Host。Client 与 Host 均支持 TCP/RFCOMM，使用同一 authority tree 和 supervisor 语义。
- RFCOMM：gomobile 直接导出 `RFCOMMProvider/Listener/Pipe`，Kotlin 使用 Bluetooth Classic socket 实现生成接口；没有反射。Android 12+ 的 CONNECT/SCAN 权限、Bluetooth unavailable/disabled、dial/listen/accept 失败均显式报错；平台 permission 不替代 MyFlowHub permit/policy。
- 产品 UI：Compose 前台服务提供 client/host、连接、资源目录、snapshot/subscription/Command、File content URI upload、Flow 和状态页面；版本 1 SharedPreferences 对未知版本/字段显式进入 reset 页面，reset 不删除 Go identity/trust/policy state。
- Build validation：canonical gomobile AAR 包含 arm64-v8a/x86_64 `libgojni.so`，Gradle JVM settings tests、`assembleDebug`、`lintDebug` 和双 ABI APK 检查通过；源码构建强制要求 AAR，禁止缺 binding stub。
- Runtime validation：Go binding/Host tests、race、Android arm64 cross-compile，以及真实 in-process persistent TCP Hub→Android Client permit/join/connected smoke 通过。当前主机没有可用 Android 设备，因此不伪报 Bluetooth socket 或设备安装 smoke；对应硬件验证留在 FM15 available-device evidence。迁移清单 4 个 Android capability/build entry 全部 verified。

### Execute - FM13

- FM13 complete：Embedded 被限定为 leaf/client profile；C99 实现 MFH3 完整 envelope、固定 identity、signed Join/ack、Subscription、Variable/Stream 接收和 Command，默认 8 KiB payload、无动态分配或隐藏队列。MicroPython 使用同一 contract，提供原子 identity、依赖注入 transport/crypto、重连与订阅恢复。
- 跨语言门禁：`tests/fixtures/embedded/envelope-command-v1.hex` 由 Go、C、MicroPython 对完整 frame 做 byte-for-byte 校验；C 使用 `-Wall -Wextra -Wpedantic -Werror` 编译并通过 CTest，MicroPython 6 个行为测试和 Go protocol tests 通过。
- ESP32-S3：ESP-IDF 6 source 直接消费 C component，提供 Wi-Fi/TCP exact I/O、NVS version/revision/identity persistence、显式 reset、parent-key pinning、signed Join、health Subscription 和 notification Command。Ed25519 固定 Espressif `libsodium 1.0.22~1`；没有把 PSA 规范支持误报为当前 Mbed TLS Edwards 实现支持。
- Available evidence：当前 Windows 主机没有 `idf.py`、ESP32 板卡和 MinGW ASan/UBSan runtime，因此 ESP-IDF build/flash/device smoke 与 sanitizer 标记为 Unavailable；普通 host C warning-as-error、MicroPython、Go fixture parity 均通过。inventory 中 C/MicroPython 为 verified，ESP32 为 implemented，不伪报 hardware pass。

### Execute - FM14

- FM14 complete：`scripts/mfh.ps1` 成为唯一根级 test/build/generate/check/manifest 入口，始终强制 `GOWORK=off`，可独立选择 Core、Hub、Desktop、Android、Metrics、Clipboard、Embedded 和 generated gates；工具链版本固定于 `build/toolchain.json`。
- 可复现本机覆盖：`MFH_JAVA_HOME`、`MFH_FLUTTER_HOME` 和 `MFH_SHORT_TEMP` 都要求显式绝对路径并只影响当前进程；Android SDK 只从环境或标准用户路径发现，不写入机器相关 `local.properties`。
- CI：无 path filter 的 9 个 job 覆盖 core、generated、Desktop Windows、Android、Metrics Windows/Android、Clipboard、Embedded host 和 ESP-IDF 6；architecture tests 固定 job/product 集合、工具链和 required AAR 输入。
- Validation：Go 全测试/vet、generated drift、Desktop Wails、Android Gradle/gomobile、Metrics Wails/Gradle、Clipboard Flutter Web/Windows/双 ABI Android、C/MicroPython 均经根入口通过。64 项 artifact manifest 包含哈希；当前主机唯一 unavailable 为 ESP-IDF/板卡，CI 已提供 IDF 6 source-build gate。

### Execute - FM15

- FM15 complete：新增同一 persistent Hub / authority tree 下的 SDK + Metrics + Clipboard 联合门禁，先验证 default-deny，再授权并连续传递 256 次 Metrics Variable 更新、64 次 Clipboard Command/Stream，最后撤权并确认订阅者、裁决根和两个资源 owner 的状态全部清零。
- 门禁发现并修复跨子树撤权漏洞：原先策略世代只清理裁决节点本地资源订阅，已转发到后代 owner 的订阅会继续存在。现在裁决点维护有界、无 payload 的转发订阅记录；世代变化同时向订阅者发送 Expired、向 owner 下发父控 Unsubscribe，强制控制权与 authority tree 一致。
- Security/integration：memory/TCP 跨子树 allow/deny/reparent，持久 Hub Notification/File/Flow，Metrics/Clipboard 独立真实进程，100 次 supervisor restart/reconnect、queue/size/concurrency/retry/restart limits 全部通过。
- Validation：`GOWORK=off go test ./...`、完整 runtime/transport/feature/host/SDK/products/integration race、10 秒 protocol fuzz（2,247,055 executions）、产品进程和 focused soak 均通过。三个 AAR 均含 arm64-v8a/x86_64 `libgojni.so`，64 项 artifact manifest 哈希复验通过；Android/RFCOMM/ESP32 真实设备仍明确 Unavailable，不伪报通过。

### Execute - FM16

- FM16 complete：默认 `scripts/run-dev.ps1` 已完全切换到 canonical `cmd/mfh-hub`、`apps/desktop`、`apps/nodes/metrics/windows`，强制 `GOWORK=off`、状态/日志隔离、后台窗口默认隐藏，并提供无副作用 dry-run 与端口等待；不再探测或启动旧仓。
- 可用性闭环：Hub 保持 default-deny，新增持久 identity 查询、一次性 permit 签发、离线精确 policy grant/revoke CLI，以及在线 `system/policy/grant` / `system/policy/revoke` Commands；Desktop/SDK/generated bindings 同步。离线状态修改明确要求 Hub 停止，避免双写。
- 清理与边界：删除 canonical 根中依赖旧 VarStore/Flow payload 的 ESP32 demo 和指向旧多仓 release 的 Android secrets helper；旧仓本身保持冻结只读。`README.md`、`repos.md`、`guide.md`、Hub/Desktop/spec 文档已切换到单仓事实来源。
- 门禁：迁移清单无 pending，仅 ESP32 source/IDF entry 因本机无 `idf.py`/板卡保留 implemented；生产输入 architecture scan 禁止旧 module、旧 repo 启动路径和 `cmd/hub_server`。121 个 module graph 条目、289 个 package dependency 均无旧 module，10 个固定旧仓 read-only audit 无漂移。
- Validation：canonical run-dev dry-run、真实 Hub 随机 TCP 启动、离线 identity/permit/policy persistence、全仓 `go test ./...`、`go vet ./...`、management/node/integration/Hub CLI race、generated drift、Desktop Vitest/Vite/Wails production build、JDK 17 下 Android AAR/APK 重建均通过。PowerShell 5.1 artifact manifest 兼容问题已修复，64 项产物哈希全部复验通过。

### Initialization

- `guide.md` 已读取：所有 worktree 必须位于 `D:/project/MyFlowHub3/worktrees`，当前 worktree 符合约束。
- `$m-discuss` 已创建专用 worktree；因用户把范围从第一产品波次扩大为全量迁移，worktree 和 branch 已安全重命名。
- main checkout 只作为 control plane；其大量既有未提交内容不属于本 workflow 写集。
- 旧仓均为只读迁移输入，不创建参与写入的旧仓 worktree。
- MetricsNode 当前工作树包含 29 个既有未提交路径；迁移只允许读取清单固定 commit，不允许复制工作树状态。
- canonical base 是已归档的统一节点运行时第一阶段提交 `87cd7b6`。

### Discuss - Discovery And Requirements Shaping

#### Goal

在一个迁移 workflow 中完成全部第一方 canonical source migration，以多个可验证门禁推进，最终一次切换默认本地入口；不长期维护两套架构。

#### Scope

- 生产级节点身份、准入、权限、重连和资源目录。
- 当前真实存在的 TCP、QUIC、RFCOMM Transport。
- Hub、SDK、Management、Notification/Topic、File、Flow。
- MetricsNode、ClipboardNode、Desktop、Android。
- Embedded C、MicroPython 和 ESP32 构建入口。
- monorepo 构建、测试、生成、CI 定义和本地 canonical cutover。

#### Assumptions

- 当前没有项目外部用户依赖旧接口。
- 旧仓 commit/tag 和文档保留历史，不需要拼接 Git history。
- 所有旧行为都必须有显式处置，但允许删除无意义 API。
- 远端、发布、签名、应用商店和旧仓远端归档需要独立授权。

#### Open Questions

- macOS/iOS 签名与商店发布环境不属于本地源码迁移授权。
- 真实 Android、RFCOMM、ESP32 设备证据取决于设备可用性，不可用时必须记为 `Unavailable`。
- canonical 远端和版本策略在本地迁移验收后决定。

#### Options Considered

- 分产品长期双轨迁移：拒绝；需要长期 compatibility bridge。
- 单个巨大提交：拒绝；无法分段验证和回滚。
- 单迁移分支、多个门禁、最终一次切换：采用。

#### Recommended Direction

所有迁移在一个长期分支中按依赖顺序实施。每个门禁保持 canonical 仓库可构建、可测试；旧入口在 FM16 之前不切换。

#### Research Summary

- 未使用外部研究。
- 依据来自当前 canonical 代码、10 个旧仓固定 commit、迁移清单、第一阶段归档、稳定 requirements/specs/decisions 和 lessons。

#### Worktree / Branch / Docs Root Status

- Branch: ready。
- Worktree: ready。
- Docs root: ready，和 canonical code 位于同一专用 worktree。
- Legacy sources: 已固定 commit，read-only。

#### Issue List

- 无阻塞问题。

## Plan - Requirements And Architecture

### Discussion Summary

用户选择全量 clean-break 迁移。这里的“一次性”定义为一个总 workflow、一个迁移分支和最终一次本地入口切换，而不是一个巨大提交。源码迁移完成不自动授权远端发布、签名或旧仓删除。

### Accepted / Rejected Requirements

Accepted:

- 全部第一方产品和仍有意义的业务能力进入 canonical monorepo。
- 每项旧能力必须标记为 `migrated`、`replaced` 或 `dropped-with-reason`。
- Variable、Stream、Command、Subscription 是应用能力的默认表达。
- 权限、协议父子链路和路由继续使用同一权威 Node 树。
- Transport 通过统一 Driver/Pipe/LinkSession 接入，上层不得按具体 Transport 分支。
- 默认一个根 Go module；Flutter/Gradle/C 工具链作为构建产物边界，不重新制造 Go module 边界。
- 中间门禁必须保持仓库可测试，最终再切换默认入口。

Rejected:

- API-for-API 复制旧 SubProto。
- 长期 legacy wire/module compatibility bridge。
- 为了迁移速度直接复制旧工作树、生成目录、构建产物或嵌套 `.git`。
- 把所有数据都强行做成 Stream；有当前值的状态优先 Variable，主动行为优先 Command。
- 在没有当前实现或产品需求时把 serial/USB/WebSocket 伪装成“现有能力迁移”。
- 未取得证据时把跨平台、签名或硬件结果标为通过。

### Requirements Analysis

#### Goal

让 canonical monorepo 独立承载 MyFlowHub 的协议、运行时、Hub、全部第一方应用、节点应用和 Embedded 实现，并在本地默认开发入口中完全退出旧多仓/SubProto 运行链。

#### Scope

包含：

- durable identity、direct-child admission、trust/policy persistence、revocation；
- resource catalog、topology/health/status resources；
- reconnect、lease renewal、subscription restoration、session generation cleanup；
- TCP/QUIC/RFCOMM；
- Management、Notification、File、Flow；
- Hub、Go SDK、platform bindings；
- MetricsNode、ClipboardNode、Desktop、Android、Embedded；
- build scripts、generated bindings、CI definitions、integration gates；
- local canonical cutover 和 legacy dependency elimination。

不包含：

- remote push、tag/release、签名、商店发布；
- 旧仓远端只读/归档操作；
- 删除本地旧仓、主工作区脏文件或论文/附件目录；
- 没有既有来源或需求的全新 Transport。

#### Use Cases

1. Hub 重启后保持稳定身份、信任与权限策略，并同时监听多个 Transport。
2. 新节点使用父节点签发的一次性准入材料加入，重连时继续使用相同身份和 authority parent。
3. Desktop/Android 订阅节点资源目录、Variable 和 Stream，并调用 Command。
4. MetricsNode 发布当前指标，接收经过裁决的音量/亮度等控制 Command。
5. ClipboardNode 以 Stream 表达事件，以 Command 表达发送/应用/清理等主动行为，并保持剪贴板隐私边界。
6. File 使用 Command 建立传输、分块和确认，使用 Stream 报告进度/结果；Flow 使用 Variable 保存定义/状态、Command 控制运行、Stream 输出运行事件。
7. Embedded C/MicroPython 节点通过相同 MFH3 envelope、认证、订阅和 Command contract 接入。
8. 节点断线、重连、Transport 切换或 reparent 后，旧 route、subscription、pending command 和 control 不再生效。
9. 默认开发与构建入口只引用 canonical source，不读取旧仓 module 或 `go.work`。

#### Functional Requirements

- IdentityStore 必须原子持久化 Ed25519 identity，并验证 NodeID/key 一致性。
- 直接父节点负责 child admission、role、policy 与 revoke；准入材料必须一次性或有明确有效期。
- Trust/policy generation 改变必须撤销旧 session、route、subscription 和 pending operation。
- 每个 Node 暴露版本化 resource catalog Variable；descriptor 变化通过订阅可观察。
- ResourceID、schema、content type、权限动作和 owner 必须稳定且可校验。
- SDK 必须提供连接状态、重连、订阅续租/恢复、Command、资源目录订阅和结构化错误。
- QUIC/RFCOMM 必须通过与 TCP 相同的 Driver/Pipe/LinkSession contract tests。
- 旧 VarStore/Stream/TopicBus/Exec/Management/File/Flow 行为不得以 SubProto dispatch 形式进入新核心。
- 所有应用必须从固定 source commit 选择性迁移，生成绑定必须在 canonical 内重新生成。
- migration manifest 必须记录每个旧仓、能力和构建入口的最终处置与验证证据。

#### Non-functional Requirements

- 安全：密钥文件最小权限、敏感字段不进入日志、未认证连接不得注册树边。
- 正确性：所有 cleanup 按 peer + topology/session generation 约束。
- 有界性：frame、queue、file chunk、flow output、subscription、pending request、history 均有上限。
- 可维护性：依赖方向保持 `protocol → runtime/transport → feature/sdk/host → apps/embedded adapters`，禁止反向依赖。
- 可移植性：平台相关代码位于 adapter/provider 边界。
- 可测试性：核心使用 fake/memory，真实 TCP/QUIC/RFCOMM 和进程级 smoke 分层验证。
- 可追溯性：source commit、dirty exclusions、处置矩阵和证据完整。

#### Inputs / Outputs

Inputs:

- 旧仓固定 commit 的源码和稳定文档；
- MFH3 frames、identity/trust/admission 配置；
- Variable updates、Stream events、Command calls；
- 文件、Flow、剪贴板、指标和平台输入；
- Flutter/Wails/Android/Embedded build configurations。

Outputs:

- 单一 canonical source tree；
- 可重复启动的 Hub/Node 和完整应用入口；
- 跨 Transport 的权威节点树；
- 资源目录、订阅、指令和 feature contracts；
- 可复现构建/测试/生成流程；
- 完整迁移处置与验证矩阵。

#### Edge Cases

- identity 文件损坏、NodeID 与 key 不匹配、重复准入 token、撤销后重连；
- 同 NodeID 并发连接、旧 generation 延迟 cleanup、reparent 与 subscription renewal 交叉；
- QUIC stream reset、RFCOMM short write/chunking/abort、Transport 切换；
- resource descriptor 删除/重命名、schema mismatch、revision/sequence regression；
- File chunk 重复/乱序/缺失/checksum mismatch、取消与断连；
- Flow 重复触发、并发上限、取消、恢复、归档和输出背压；
- Clipboard oversize/重复/回环/隐私泄漏；
- Metrics actuator 越界、平台不支持和读取/执行竞态；
- Android/Flutter/Wails generated binding 漂移；
- Embedded 低 MTU、内存不足、重启后 identity/config 恢复。

#### Acceptance Criteria

- migration disposition matrix 无未分类第一方能力或入口。
- canonical source 无旧 `myflowhub-*` module import、SubProto dispatch、nested `.git` 或无理由 nested Go module。
- Hub、SDK、Desktop、Android、MetricsNode、ClipboardNode、Embedded source 均位于目标目录并使用 vNext contracts。
- TCP/QUIC/RFCOMM contract tests 和可用环境 smoke 通过。
- 跨产品树、权限、订阅、Command、File、Flow、断连/reparent 集成通过。
- Go race/fuzz、Flutter/Wails/Gradle、C/MicroPython/ESP 构建在可用环境通过；不可用证据明确标记而非伪造。
- 默认本地开发入口无需旧仓和 `go.work` 即可运行 canonical Hub 与产品。
- 旧仓保持未修改；main checkout 既有脏文件保持不被覆盖。

#### Risks

- 分支大且存续时间长；任何共享 contract 漂移会造成多产品返工。
- 产品现有行为文档分散，可能出现静默遗漏；处置矩阵是强制门禁。
- 跨平台工具链和硬件并非全部本机可用。
- File/Flow 容易诱导新增第二套消息框架；必须严格建立在资源/订阅/Command 上。
- 生成绑定和 build artifacts 容易把旧工作树脏状态带入 canonical。

### Architecture Design

#### Overall Solution

目标布局：

```text
protocol/                     MFH3 envelope、IDs、schemas、built-in resource contracts
runtime/                      auth、link、tree、resource、subscription、command、node
transport/                    memory、tcp、quic、rfcomm
feature/                      management、notification、file、flow
host/hub/                     durable Hub composition
sdk/go/                       operational client and lifecycle
sdk/bindings/                 gomobile / generated language boundaries
apps/desktop/                 MyFlowHub desktop
apps/android/                 MyFlowHub Android
apps/nodes/metrics/           MetricsNode
apps/nodes/clipboard/         ClipboardNode
embedded/c/                   C SDK/runtime
embedded/micropython/         MicroPython runtime
embedded/esp32/               ESP-IDF integration and demos
cmd/                          canonical executable entry points
tests/                        contract、integration、e2e、fixtures
migration/                    source and disposition manifests
docs/                         governed stable and workflow docs
```

#### Migration Disposition Model

每项旧能力记录：

```text
source_repo + source_path + source_commit
legacy_behavior
target_path
disposition: migrated | replaced | dropped-with-reason
tests/evidence
dirty-source exclusion
```

`migrated` 表示保留行为并适配新契约；`replaced` 表示保留目标但用资源模型重写；`dropped-with-reason` 表示明确删除无价值 API。任何空状态阻塞 FM16。

#### Operational Identity And Admission

- host/app 层提供文件或平台安全存储实现，runtime 只依赖接口。
- child 首次加入使用父节点签发的短期一次性 permit；permit 绑定 child NodeID/public key、parent NodeID、role、expiry 和 nonce。
- parent 成功准入后持久化 child trust/policy，child 持久化 parent key/config。
- revoke 提升 trust/policy generation 并关闭匹配 session；旧 permit、旧 epoch 和旧 control 均无效。
- root Hub 由本地 bootstrap 配置建立，不引入第二棵 authority 树。

#### Resource Catalog

- 每个 Node 保留一个 built-in catalog Variable，内容是本节点资源 descriptor 的版本化快照。
- catalog revision 在注册、删除或 schema/metadata 变化时单调增加。
- 客户端通过普通 Variable Subscription 获得快照与变化，不新增旁路 discovery 协议。
- alias/tag 仅用于查找，最终授权和路由始终使用 owner NodeID + ResourceID。

#### Legacy Capability Mapping

| Legacy capability | vNext replacement |
| --- | --- |
| Auth | `runtime/auth` + host Identity/Trust/Admission stores |
| Forward | authoritative tree routing |
| Broker | Subscription interest aggregation |
| VarStore | Node-owned Variable |
| Stream | Node-owned Stream + Subscription |
| Exec | Command |
| Management | built-in Variables/Commands under `feature/management` |
| TopicBus/notification | owner or Hub-owned Stream plus publish/config Commands |
| File | transfer Commands + bounded chunk calls + progress/result Stream |
| Flow | definition/status Variables + lifecycle Commands + run-event Stream |

File upload 使用有序 chunk Command（sequence、size、checksum、idempotency）；下载和进度使用 transfer-owned Stream。它是 feature contract，不新增第二套 route/handler 框架。

#### Product Assembly

- Hub 组合 runtime、stores、multi-listener、management、notification、file、flow。
- Desktop/Android 使用 SDK 订阅 catalog/Variables/Streams 并调用 Commands；UI 不直接理解 wire phase。
- MetricsNode 只保留采集器、执行器和本地产品配置；旧 VarStore/TopicBus runtime 删除式重写。
- ClipboardNode 保留剪贴板 adapter、dedupe、隐私和桥接/UI；同步语义改为 Stream/Command。
- Embedded 实现同一 envelope 和资源语义的受限 profile，不复制旧 SubProto action 表。

#### Data / Call Flow

```text
Platform / UI
    ↓ app service
SDK / Host composition
    ↓ catalog + subscribe + invoke
Node Runtime
    ↓ authenticated authority tree
LinkSession
    ↓ Driver / Pipe
TCP | QUIC | RFCOMM
```

#### Interface Drafts

```go
type IdentityStore interface {
    LoadOrCreate(ctx context.Context, expected protocol.NodeID) (auth.Identity, error)
}

type AdmissionStore interface {
    Consume(ctx context.Context, permit []byte, claim auth.JoinClaim) (auth.Admission, error)
    Revoke(ctx context.Context, child protocol.NodeID) error
}

type ParentProfile struct {
    NodeID    protocol.NodeID
    Endpoint  link.Endpoint
    Driver    string
    PublicKey []byte
}

type ConnectionSupervisor interface {
    Run(context.Context) error
    State() ConnectionState
    Changes() <-chan ConnectionState
}

type CatalogEntry struct {
    Resource protocol.ResourceID
    Kind     resource.Kind
    Schema   string
    MediaType string
    Metadata map[string]string
}
```

具体命名可在实现时遵循现有 package 风格调整，但 store/host、runtime 和 Transport 的依赖方向不可反转。

#### Error Handling And Safety

- 所有外部配置、permit、descriptor、payload、chunk 和 generated binding 输入在边界验证。
- key/config 使用 atomic replace，损坏时明确失败，不自动生成新身份覆盖旧身份。
- Transport 错误分类为 temporary/permanent/closed，重连只处理明确 temporary 类别。
- 所有队列满、gap、checksum、timeout、unsupported platform 和 permission deny 返回结构化错误。
- 不记录密钥、permit、剪贴板正文或文件内容；诊断只记录 ID、size、hash prefix 和错误码。
- platform actuator 默认通过 fake 测试；真实副作用 smoke 必须显式开启。

#### Performance And Testing Strategy

- contract tests 对所有 Driver 复用同一套读写、短写、close、deadline 和 backpressure 用例。
- runtime 关键并发生命周期执行 `-race`、高重复和故障注入。
- protocol/schema/chunk parser 执行 fuzz。
- File/Flow/Clipboard/metrics 分别验证 payload、队列、历史和并发上限。
- 产品层先 unit/contract，再 process smoke，最后跨产品 e2e。
- generated bindings 必须可从 canonical source 重建，并由 diff guard 防止漂移。

#### Extensibility Design Points

- 新 Transport 只实现 Driver/Pipe 和可选 capability metadata。
- 新业务默认通过注册 Variable/Stream/Command 组合，不创建新顶层协议分发器。
- 第二个真实需求出现前不抽象通用插件系统。
- Embedded profile 可以限制 schema、payload 和 feature 集，但不能改变 authority、resource identity 或 phase 语义。

#### Issue List

- 无阻塞问题；跨平台/硬件/远端证据在单独任务中明确边界。

## Stage 3.1 - Planning

### Project Goal And Current State

当前 master 已完成核心第一阶段。此次计划从 `87cd7b6` 开始，迁移全部剩余第一方源码和产品，最终使本地 canonical 默认入口不再依赖旧仓。

### Docs Governance Routing Decision

使用 `$m-docs` 校验：

- Docs root: `D:/project/MyFlowHub3/worktrees/vnext-full-migration/docs`
- 原始范围扩展已更新到 intake。
- 全量迁移/切换策略已记录为 Accepted decision。
- `docs/features` 当前缺失，FM01 先建立产品 feature dossiers。
- requirement 已覆盖完整迁移目标，无需在计划阶段重复改写。
- operational lifecycle、resource catalog、feature contracts 将在 FM01 先形成稳定 spec，再写业务代码。
- change/lessons 仅在执行和归档产生真实结果后更新。

### Related Docs

- Intake:
  - [vNext 全量迁移与最终切换](docs/intake/2026-08-27_vnext-full-migration.md)
  - [节点树、订阅与指令重构诉求](docs/intake/2026-08-27_node-tree-subscription-command-redesign.md)
- Features:
  - 当前无 canonical feature index；FM01 建立。
- Requirements:
  - [统一节点运行时](docs/requirements/unified-node-runtime.md)
- Specs:
  - [节点树、链路与资源架构](docs/specs/node-tree-link-resource-architecture.md)
  - [仓库与模块边界](docs/specs/repository-and-module-boundaries.md)
  - [Wire Protocol vNext](docs/specs/wire-protocol-vnext.md)
  - [Resource Model vNext](docs/specs/resource-model-vnext.md)
  - [Subscription vNext](docs/specs/subscription-vnext.md)
  - [Command vNext](docs/specs/command-vnext.md)
- Decisions:
  - [统一权威节点树与可插拔链路](docs/decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md)
  - [使用单一 Canonical Monorepo](docs/decisions/2026-08-27_single-canonical-monorepo.md)
  - [单迁移分支、分门禁实施与最终一次切换](docs/decisions/2026-08-27_gated-full-migration-cutover.md)
- Lessons:
  - [会话替换与清理必须绑定 generation](docs/lessons/session-replacement-generation-cleanup.md)
  - [跨仓 SemVer 发布链](docs/lessons/cross-repo-semver-release.md)

### Stable Docs Impact

- Intake impact: `clarify`，已从第一产品波次扩展为全量迁移。
- Feature impact: `add`，FM01 建立 Hub、Desktop、Android、MetricsNode、ClipboardNode、File/Flow 当前行为档案。
- Requirements impact: `clarify`，FM01 只在处置矩阵发现缺口时补充，不复制计划内容。
- Specs impact: `add/clarify`，FM01 增加 operational lifecycle、resource catalog 和 feature contracts。
- Decision impact: `add`，已新增 gated full migration/cutover ADR。
- Lessons known at planning time: session generation cleanup、跨仓依赖链；执行期间出现可复用问题再新增。

### Execution Scope After Approval

#### Will Execute

- FM00-FM16：全部 canonical source migration、验证和本地切换任务。

#### Will Not Execute Now

- DX01：serial/USB/WebSocket 等无既有来源的新 Transport；不是迁移任务，等待真实需求。
- DX02：legacy wire/SubProto bridge；设计上拒绝，除非未来出现无法迁移的第一方硬阻塞并重新审批。
- DX03：remote push、tag/release、签名、商店发布、旧仓远端归档；需要独立授权和凭据。
- DX04：删除/移动本地旧仓、清理 main checkout 既有脏文件；属于破坏性外部收尾，单独授权。
- DX05：当前环境不可提供的 macOS/iOS 签名与外部硬件发布认证；保留 CI/构建定义，但不能伪报真实证据。

### Executable Task List

#### FM00 - Freeze Sources And Build Migration Inventory

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/vnext-full-migration`
- Plan Path: `plan.md`
- Goal: 固定所有 source commit、dirty exclusions、产品/能力/构建入口处置矩阵和目标路径。
- Files / Modules: `migration/**`, `internal/archtest/**`, `docs/intake/**`
- Write Set: migration manifests、guard tests、planning docs；不写旧仓。
- Acceptance: 10 个旧仓 commit 可验证；每个旧 capability/build entry 有唯一处置状态；MetricsNode dirty hash 清单完整；旧仓状态前后不变。
- Test Points: manifest schema test、source commit/hash audit、legacy status snapshot comparison、docs link guard。
- Rollback: 删除新增 manifest/guard；旧仓因只读无需数据回滚。

#### FM01 - Establish Stable Product And Migration Contracts

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 在写业务代码前建立产品 feature dossiers 和 operational/resource/feature specs。
- Files / Modules: `docs/features/**`, `docs/requirements/**`, `docs/specs/**`, `docs/decisions/**`, category/root indexes
- Write Set: governed docs only。
- Acceptance: Hub、Desktop、Android、MetricsNode、ClipboardNode、File/Flow 当前行为、权限、非目标和验收可查；operational lifecycle、catalog、notification/file/flow contracts 明确；不把历史 change 当当前真相。
- Test Points: docs relative-link guard、index coverage、generated/protected region guard、requirement-impact review。
- Rollback: 回退本任务稳定文档；不会影响运行时。

#### FM02 - Extend Protocol Schemas And Built-in Resource Catalog

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 定义 provisioning、catalog、management、notification、file、flow 的版本化 schema 与跨语言 fixtures。
- Files / Modules: `protocol/**`, `tests/fixtures/**`, `internal/protocoltest/**`
- Write Set: protocol and fixtures。
- Acceptance: schema/operation 不依赖 SubProto；catalog 使用 Variable；所有 payload 有 size/version/validation；Go round-trip 和 golden fixtures 稳定。
- Test Points: `go test ./protocol ./internal/protocoltest`, fuzz codec/schema/chunk parsers, fixture reproducibility。
- Rollback: 回退新增 schema/fixture；第一阶段 protocol 保持可用。

#### FM03 - Durable Identity, Admission, Trust And Policy

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 将临时身份/信任升级为可重复启动、直接父控子的持久运行模型。
- Files / Modules: `runtime/auth/**`, `runtime/node/**`, `host/config/**`, `internal/keystore/**`
- Write Set: auth/config/runtime lifecycle。
- Acceptance: identity 原子 load/create；permit 单次/过期/绑定校验；trust/policy 持久化与 revoke；损坏配置明确失败；敏感信息不泄漏。
- Test Points: focused tests、restart tests、permit replay/expiry/fake-parent tests、`-race`、Windows file-permission behavior。
- Rollback: 保留版本化 config backup，回退 store/composition；不覆盖用户旧 key。

#### FM04 - Connection Supervision, Reconnect And Subscription Recovery

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 提供单父连接监督、退避重连、状态观测、租约续期和产品订阅恢复。
- Files / Modules: `runtime/link/**`, `runtime/node/**`, `runtime/subscription/**`, `sdk/go/**`
- Write Set: session/client lifecycle。
- Acceptance: temporary/permanent errors 分类；同父重连不删除新 edge；reparent 不恢复旧权限/订阅；关闭无 goroutine/pending 泄漏。
- Test Points: reconnect/reparent count=100、fake clock/backoff、link replacement、`go test -race ./runtime/... ./sdk/go/...`。
- Rollback: 回退 supervisor；保留显式一次连接 API。

#### FM05 - Migrate QUIC And RFCOMM Drivers

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 把旧 Core 的 QUIC/RFCOMM 经验迁入统一 Driver/Pipe/LinkSession contract。
- Files / Modules: `transport/tcp/**`, `transport/quic/**`, `transport/rfcomm/**`, `runtime/link/**`, platform providers
- Write Set: transport packages only，除必要 link capability metadata。
- Acceptance: TCP/QUIC/RFCOMM 共用 contract suite；short write、chunking、abort、deadline、close、MTU 行为明确；上层无 Transport type switch。
- Test Points: contract tests、QUIC loopback、Windows/Linux/Android provider compile、可用设备 RFCOMM smoke、fault injection。
- Rollback: 每个 Driver 可独立回退；TCP 保持基线回退路径。

#### FM06 - Build Production Hub And Management Resources

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 用 durable stores、多 Listener 和 built-in resources 取代旧 Server runtime/defaultset。
- Files / Modules: `host/hub/**`, `feature/management/**`, `cmd/mfh-hub/**`, `cmd/mfh-admin/**`
- Write Set: host/management/cmd。
- Acceptance: Hub 重启身份稳定；多 Transport listener；topology/catalog/health/config Variables；admit/revoke/config Commands；策略允许/拒绝可审计。
- Test Points: host unit、restart/process smoke、multi-listener、policy deny/revoke、config corruption、graceful shutdown。
- Rollback: 旧 minimal Hub 保留到新 composition 验收；config 格式版本化。

#### FM07 - Replace Remaining SubProto Features

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 用资源组合实现 notification、File、Flow，彻底替代 TopicBus/File/Flow/Management handler 模型。
- Files / Modules: `feature/notification/**`, `feature/file/**`, `feature/flow/**`, related protocol schemas
- Write Set: feature packages；发现 core gap 必须回到 FM02-FM04 修复并重跑下游。
- Acceptance: notification 使用 Stream/Command；File chunk 有序/幂等/checksum/cancel/progress；Flow definition/status/run/cancel/archive 有界；无通用 action dispatcher。
- Test Points: feature unit/contract、large/zero file、chunk duplicate/gap、flow dedup/concurrency/cancel/retry/output overflow、permission tests。
- Rollback: feature 可按 package 回退，不修改 runtime authority 规则。

#### FM08 - Complete Go SDK And Platform Binding Contracts

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 为所有产品提供稳定 catalog/subscribe/invoke/file/flow/lifecycle API 和跨语言绑定边界。
- Files / Modules: `sdk/go/**`, `sdk/bindings/**`, `tests/fixtures/**`
- Write Set: sdk/bindings/fixtures。
- Acceptance: SDK 不暴露 wire phase；subscription cancel/recover、typed schema errors、connection state 可用；bindings 从 canonical source 重建。
- Test Points: SDK contract、fake server、reconnect、fixture parity、gomobile/Wails binding generation smoke。
- Rollback: 保留低层 Node API；回退 convenience layers/bindings。

#### FM09 - Migrate MetricsNode Entire Product Source

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 迁移 MetricsNode core、Windows UI、Android binding/build，删除旧 SDK/VarStore/TopicBus runtime。
- Files / Modules: `apps/nodes/metrics/**`, `cmd/mfh-metrics/**`
- Write Set: metrics app only；generated bindings 重新生成，不读取旧 dirty paths。
- Acceptance: read-only metrics 为 Variables；controllable metrics 为 state Variable + Command；config/collector/actuator/UI 行为有处置；Windows/Android source 使用新 SDK。
- Test Points: Go unit、fake actuator、Windows collector smoke、Wails generation/build、Android binding/Gradle build、process e2e。
- Rollback: 回退 canonical app directory；旧 MetricsNode 仓保持原状。

#### FM10 - Migrate ClipboardNode Entire Product Source

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 迁移 core、bridge、platform adapters、Flutter shell 和 mobile bindings，并用 Stream/Command 重写同步链。
- Files / Modules: `apps/nodes/clipboard/**`, `cmd/mfh-clipboard/**`
- Write Set: clipboard app only。
- Acceptance: body/privacy/history/dedupe/loop suppression/oversize 行为有处置；状态用 Variable、事件用 Stream、发送/应用/清理用 Command；不记录正文到日志/status/config。
- Test Points: Go unit/bridge contract、two-node smoke、Flutter analyze/test、Windows/Android/Web build、available platform adapter tests。
- Rollback: 回退 canonical app；旧 ClipboardNode 发布入口不在本任务切换。

#### FM11 - Migrate Desktop Application

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 迁移 MyFlowHub-Win UI/host/storage，把旧 service wrappers 替换为 catalog/subscription/Command/feature APIs。
- Files / Modules: `apps/desktop/**`, `cmd/mfh-desktop/**`
- Write Set: desktop app；共享 contract 变更回到 owning task。
- Acceptance: auth/admission、nodes/resources、permissions、metrics/clipboard、file/flow 页面使用新 SDK；无旧 module import；storage/config 有版本化迁移或明确 reset。
- Test Points: Go service tests、frontend unit/build、Wails binding generation、Windows app smoke、error/empty/reconnect states。
- Rollback: 回退 canonical desktop app/default entry；旧 Win 仓未修改。

#### FM12 - Migrate Android Application

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 迁移 Android UI、hubmobile、RFCOMM provider 和产品能力到 canonical bindings。
- Files / Modules: `apps/android/**`, `sdk/bindings/android/**`
- Write Set: Android app/binding/provider。
- Acceptance: join/reconnect/catalog/subscription/Command、file/flow、node apps integration 使用 vNext；没有 sibling repo `replace` 或旧 module；RFCOMM platform errors 显式。
- Test Points: Go binding tests、Gradle unit/lint/build、Android compile matrix、可用设备 TCP/RFCOMM smoke。
- Rollback: 回退 canonical Android tree；旧 Android repo 保持回退构建。

#### FM13 - Migrate Embedded C, MicroPython And ESP32

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 为受限设备实现 vNext envelope、identity/admission、Variable/Stream/Command 和所需 Transport profile。
- Files / Modules: `embedded/c/**`, `embedded/micropython/**`, `embedded/esp32/**`, `tests/fixtures/**`
- Write Set: embedded and fixtures；核心 schema 变更回到 FM02。
- Acceptance: C/MicroPython 与 Go golden fixtures 一致；有界内存/MTU；重启恢复 identity/config；ESP32 demo 可连接 Hub 并完成订阅/Command。
- Test Points: host C tests/sanitizers、MicroPython tests、fixture parity、ESP-IDF v6.0 build、可用板卡 flash/smoke。
- Rollback: 回退 embedded directories；旧 EmbeddedSDK 仓未修改。

#### FM14 - Unify Builds, Generation, Packaging And CI Definitions

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 提供 monorepo 根级可复现开发、测试、生成和各产物构建入口。
- Files / Modules: `scripts/**`, `.github/workflows/**`, root build metadata, app build configs, generated-file guards
- Write Set: build/tooling/CI definitions。
- Acceptance: root commands 可选择性构建 Hub/Desktop/Android/Metrics/Clipboard/Embedded；generated bindings 可重建且无漂移；路径门禁不遗漏共享 core 变更。
- Test Points: script unit/smoke、workflow/config validation、clean-generation diff、artifact path checks、`GOWORK=off` root builds。
- Rollback: 各 build entry 可独立回退；不发布产物。

#### FM15 - Run Cross-product Security, Integration And Performance Gates

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 证明完整产品矩阵在同一权威树和不同 Transport 下可协作，并收集可审计证据。
- Files / Modules: `tests/integration/**`, `tests/e2e/**`, test harnesses, non-versioned evidence outputs
- Write Set: tests/harness only；失败修复回到 owning task。
- Acceptance: Hub + Desktop/Android + Metrics/Clipboard + Embedded 场景；allow/deny/revoke/reparent；Variable/Stream/Command/File/Flow；disconnect/restart/soak；证据矩阵无伪 pass。
- Test Points: full Go tests/race/fuzz、process e2e、transport contract、Flutter/Wails/Gradle/C/MicroPython/ESP available checks、resource/queue limits、leak/soak。
- Rollback: 测试本身可回退；任何行为修复按 owning task rollback。

#### FM16 - Eliminate Legacy Dependencies And Switch Local Canonical Entry

- Owner: main agent
- Worktree: active worktree
- Plan Path: `plan.md`
- Goal: 完成本地 canonical cutover，并证明默认开发入口不再读取旧仓。
- Files / Modules: `migration/**`, `internal/archtest/**`, `scripts/run-dev*`, root `README.md`, `repos.md`, docs indexes/specs/requirements
- Write Set: guards、manifests、default local entry、stable docs；不删除旧仓。
- Acceptance: 无旧 imports/SubProto/nested git/go.work dependency；处置矩阵全部终态；默认 local run 使用新 Hub/products；旧仓 status 与 FM00 snapshot 一致。
- Test Points: clean canonical checkout full validation、legacy string/import scan、module/dependency graph guard、default run-dev smoke、migration manifest completeness。
- Rollback: 回退默认入口/manifest/guards；旧产品仍在旧仓可独立运行。

### Tasks Will Not Execute Now

#### DX01 - Add New Serial/USB/WebSocket Transports

- Scope: Will not execute now。
- Reason: 当前迁移来源和已确认产品没有对应实现；属于新功能而非全量迁移。
- Future Gate: 出现真实设备/产品需求后独立 `$m-discuss`/`$m-plan`。

#### DX02 - Build A Legacy Compatibility Bridge

- Scope: Will not execute now。
- Reason: 无外部用户，且 bridge 会重新制造双轨和 SubProto 依赖；明确拒绝。
- Future Gate: 只有第一方硬阻塞且无法直接迁移时重新审批，并必须有删除期限。

#### DX03 - Push, Release, Sign, Publish Or Archive Remotes

- Scope: Will not execute now。
- Reason: 当前未授权远端、发布、签名、商店或旧仓远端状态变化。
- Future Gate: FM16 和 `$m-test`/`$m-archive` 完成后单独授权。

#### DX04 - Delete Or Move Legacy Repositories / Main Dirt

- Scope: Will not execute now。
- Reason: 破坏性且不属于 canonical source migration；旧仓是回退和历史来源。
- Future Gate: 远端备份/归档和本地路径确认后单独执行。

#### DX05 - Certify Unavailable Signed Platforms Or Hardware

- Scope: Will not execute now。
- Reason: macOS/iOS 签名、商店和部分真实设备证据依赖当前环境之外的主机、设备或凭据。
- Future Gate: 对应环境可用且用户授权后执行；FM14 仍负责构建/CI 定义。

### Dependencies

```text
FM00 → FM01 → FM02
                ├→ FM03 → FM04 → FM06
                ├→ FM05 ─────────┤
                └→ FM07 → FM08 ──┤
                                  ├→ FM09
                                  ├→ FM10
                                  ├→ FM11
                                  ├→ FM12
                                  └→ FM13
FM09..FM13 → FM14 → FM15 → FM16
```

- FM02 冻结共享 schema 后才进入持久运行和 feature 实现。
- FM03-FM05 完成后 Hub/SDK 才能作为产品基础。
- FM07/F08 完成后应用迁移；应用发现共享缺口必须回到 owning task，而不是建立 app-local fork。
- FM09-FM13 全部完成后统一生成/构建和 e2e。
- FM16 只能在 FM15 证据门禁完成后执行。

### Risks And Notes

- 不允许从 main checkout 或旧仓脏工作树复制未记录内容。
- 迁移期间不改变旧仓 branch、tag、remote、release 或 working tree。
- 各应用可保留独立产物版本，但共享协议和内部 package 不单独发 module tag。
- 若某任务引入跨层依赖或需要兼容层，停止该任务并回到计划复核。
- 每个门禁完成后必须重新运行其上游 contract tests，避免大分支累积不可定位回归。
- `$m-execute` 不等于 remote/publish/cleanup 授权。

### Rollback Strategy

- 使用任务边界提交或等价可审查 patch 作为 rollback 单位。
- FM16 之前默认产品入口不切换，旧系统始终保留运行回退。
- config/key 格式均带版本并使用 atomic replace；升级前保留可恢复副本。
- 任何 schema 变更必须同步 fixture 和所有已迁移消费者；失败时整体回退该 schema 门禁。
- 不通过覆盖或删除旧仓/main dirt 回滚。

### Parallelism Assessment

- 当前不派发实现子Agent：用户未显式请求委派，宿主策略不允许主动并行 Agent。
- 即使后续授权委派，FM00-FM08 的共享 contract 任务必须串行冻结。
- FM09-FM13 在共享 contract 冻结后具备文件级并行潜力，但任何共享 API 修改仍由主任务统一收敛。

### Issue List

- Blocking issues: none。
- Approval recorded: 用户已通过显式 `$m-execute` 批准 FM00-FM16；全部执行门禁完成，未提交、合并、推送、发布、归档或删除旧仓。
