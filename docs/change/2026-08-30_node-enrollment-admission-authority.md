# 2026-08-30 Node Enrollment 与集中式 Admission Authority

## 变更背景 / 目标

原有首次连接要求客户端预先填写 Node ID、父 Node ID 和父节点公钥，并把一次性 Join Permit 与普通登录混在同一流程中。该模型无法表达“尚未获批就没有正式 Node ID”，也无法在多级节点树中提供统一、可撤销、可审计的 Permit 与 Pending 管理。

本次工作将节点树的链路控制与准入安全状态分开：直接父节点继续控制物理连接和本地信任，单一逻辑 Admission Authority 统一管理 Permit、Pending、Enrollment、撤销和 Node ID 分配。管理操作可以从任意受权节点沿树路由到 Authority，不引入父节点间的多写合并。

## 具体变更内容

- 新增独立 `MFHE` pre-auth 帧族、版本化 Enrollment/Admission schema、严格长度与签名校验；普通 `MFH4` Envelope 的非零 Source/Target 约束保持不变。
- 分离“设备密钥身份”和“已获批 Node 身份”。设备首次准备密钥时没有 Node ID；只有 Authority Grant 原子持久化后才进入普通 Join。
- 实现单写 Authority 状态机：Permit 签发/消费/撤销、Pending 提交/批准/拒绝、Enrollment、随机正 63 位 Node ID、唯一索引、tombstone、epoch、审计和恢复。
- 实现父节点 challenge/proof、transcript binding、Permit 直入、无 Permit Pending、幂等重试、父节点 Grant 缓存和同管道/重连后的普通 Join。
- 增加本地与远程 Authority broker。管理节点 A 可以为目标父节点 B 操作，但由 Authority 唯一校验、提交和签名。
- 增加 `system/admission/*` 资源、分离的 read/issue/revoke/approve/reject/submit 权限、分页与稳定 cursor/epoch，以及不含 Permit 正文和公钥的审计摘要。
- 扩展 Hub 和 `mfh-admin`，覆盖 Authority 配置、headless identity/enroll/status、Permit 管理、审批和撤销；Authority 不可用时新准入 fail closed，已有 Grant 的 Join 不新增实时 Authority 依赖。
- 扩展 Go bindings 和 Desktop protected credential storage；Windows 使用单独 DPAPI 文件保存设备密钥与 Enrollment binding，Profile/settings 不保存一次性 Permit。
- Desktop 生产界面实现 endpoint-first Authority Profile、Permit/Pending/TOFU 路径和集中式 Admission Console；Legacy Profile 保留兼容字段。
- 另保留 [登录与首次连接交互原型](../../design-demos/login-profile-flow.html) 作为后续信息架构参考。该原型把“使用现有 Profile”和“首次连接”分成两个入口，但尚未映射回生产 React，不属于本次已上线行为。

## Docs root

`D:\project\MyFlowHub3\worktrees\node-enrollment-admission\docs`（合并后为 canonical monorepo 的 `docs/`）。不存在单独的私有 docs repository；本次不设置 remote、不 push、不发布。

## Intake impact

updated

## Feature impact

updated

## Requirements impact

updated

## Specs impact

updated

## Decision impact

updated

## Lessons impact

updated

## Related intake

- [Node Enrollment 与集中式准入 Authority](../intake/2026-08-30_node-enrollment-central-authority.md)
- [Desktop 首次准入引导](../intake/2026-08-29_desktop-first-admission-onboarding.md)

## Related features

- [Hub](../features/hub.md)
- [Desktop](../features/desktop.md)
- [Android](../features/android.md)
- [Embedded leaf SDK](../features/embedded.md)

## Related requirements

- [受控准入](../requirements/auth-controlled-admission.md)

## Related specs

- [Node Enrollment 与 Admission Authority](../specs/node-enrollment-and-admission-authority.md)
- [节点树、链路与资源架构](../specs/node-tree-link-resource-architecture.md)
- [Operational lifecycle](../specs/operational-lifecycle.md)

## Related decisions

- [集中式 Admission Authority](../decisions/2026-08-30_centralized-admission-authority.md)
- [统一权威节点树与可插拔链路](../decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md)

## Related lessons

- [Windows 上 Go 交叉编译测试必须使用 compile-only](../lessons/go-cross-compile-tests-on-windows.md)
- [Desktop binding 重连与准入诊断](../lessons/desktop-binding-reconnect-and-admission-diagnostics.md)
- [Authority 管理必须沿节点树路由](../lessons/authority-local-admin-actions.md)

## 对应 plan.md 任务映射

| Task ID | 结果 | 主要落点 |
| --- | --- | --- |
| `DOC-ENROLL-1` | 完成 | intake/features/requirements/specs/decisions 与索引 |
| `PROTO-ENROLL-1` | 完成 | `protocol/` Enrollment frame 与 Admission schema |
| `AUTH-STATE-1` | 完成 | `runtime/auth/` device identity、Authority state 与持久化 |
| `LINK-ENROLL-1` | 完成 | `runtime/enrollment/` 与 `runtime/link/prefix.go` |
| `NODE-AUTHORITY-1` | 完成 | `runtime/node/`、`host/hub/` 的 broker、trust 与 revoke |
| `MGMT-ADMISSION-1` | 完成 | `feature/management/` 与 SDK management contracts |
| `HOST-CLI-1` | 完成 | `cmd/mfh-hub`、`cmd/mfh-admin` 与 README |
| `BINDING-COMPAT-1` | 完成 | `sdk/bindings/`、Desktop facade 与 generated contract |
| `DESKTOP-ENROLL-1` | 完成 | Desktop Go/React、credential storage、Admission Console 与 dist |
| `TEST-ENROLL-1` | 完成 | full/race/vet/frontend/build/package/live UI/security evidence |

明确延期：`FEDERATED-AUTH-1`、`OFFLINE-TICKET-1`、`NATIVE-CLIENT-1`、`REPARENT-REKEY-1`。双入口登录原型的生产 React 映射也未在本次归档前执行。

## 经验 / 教训摘要

- 分布式节点树不等于准入状态也必须多写。Permit 单次消费、撤销和 Node ID 唯一性需要一个强一致事实源；把操作路由到 Authority 比合并父节点本地状态更简单且可审计。
- “父节点拥有链路控制权”与“父节点可以任意签发全局身份”是两个不同权限。父节点可以拒绝和断开子连接，但正式身份仍由 Authority Grant 决定。
- 未注册设备的稳定身份应是设备密钥指纹，而不是客户端先占一个 Node ID。Pending 创建不分配 ID，恶意申请无法耗尽 ID 空间。
- Windows 主机上的 `GOOS=linux go test` 会尝试执行 Linux 测试二进制；交叉平台门禁必须使用 compile-only。
- 登录界面应先区分已有 Profile 与首次连接；公钥和 Node ID 的协议细节不应在常规路径中变成用户输入负担。

## 可复用排查线索

- 症状：Windows 交叉验证出现 `not a valid Win32 application`。触发：设置 `GOOS=linux` 后直接运行 `go test`。快速检查：改用 `go test -c -o NUL` 或只运行 `go build`。
- 症状：失败登录后修正地址或 Permit 仍提示生命周期/父信任错误。关键词：`parent trust cannot change`、`context deadline exceeded`。快速检查：每次重试是否创建新的 binding client，并仅在成功后清空 Permit。
- 症状：管理节点 A 无法为父节点 B 操作或发生本地状态分叉。关键词：remote Authority、`system/admission/*`。快速检查：请求是否路由到配置的 Authority，Principal 是否保留，A/B 是否误建本地 Authority 状态。
- 症状：重试分配第二个 Node ID。快速检查：Permit/request 是否进入同一原子 Grant 事务，幂等 key、device fingerprint 和 target parent 是否一致。

## 关键设计决策与权衡

- 采用单一逻辑 Authority，而不是父节点本地多写或 CRDT。代价是新准入依赖 Authority 可达；收益是 Permit、撤销和 Node ID 只有一个事实源。
- Node ID 使用 Authority 随机正 63 位分配并检查 active/revoked/tombstone。相比顺序 ID，能降低未导入 legacy ID 的碰撞风险，但仍保留持久唯一校验。
- 已注册 Join 依赖父节点缓存的已验证 Grant，不在每次重连时访问 Authority。这样控制面短暂不可用不会扩大成数据面停机。
- 审批路径允许显式 TOFU，headless 部署可预置父节点或 Authority 指纹；不把手工公钥输入作为所有用户的默认负担。
- 保留 Legacy Join 与旧 Profile，Android/Embedded 本阶段不发送 MFHE，避免一次性破坏全部客户端。

## 测试与验证方式 / 结果

- `GOWORK=off go test -count=1 ./...`：Passed。
- `GOWORK=off go test -race -count=1 ./runtime/auth ./runtime/enrollment ./runtime/node ./feature/management ./host/hub ./sdk/bindings`：Passed。
- `GOWORK=off go vet ./...`：Passed。
- Authority、Enrollment、Management、Protocol 安全状态测试使用 shuffle/repeat 多轮运行：Passed。
- `go generate ./sdk/bindings`：幂等，contract SHA-256 为 `0D5931116EC8A0CED71A9DF30482242EA1F3D0F05C4A24D50BEE5E2E256FC1A5`。
- Desktop：Enrollment 候选分支 50 个 Vitest 测试通过；同步最新 Explorer/BrandMark 主线后共 60 个 Vitest 测试、TypeScript 与 Vite production build：Passed。
- Linux `amd64` Desktop credential path compile-only、Windows `amd64` Wails clean production package 与可执行文件启动 smoke：Passed。
- 实际 packaged UI：完成无 Node ID 设备准备、显式 TOFU、Permit Enrollment、Authority 分配 ID、DPAPI 恢复、重启重连，以及 Admission Console list/issue：Passed。
- Permit ID/签名未出现在 settings、Profile、日志、credential 外文件、管理列表或 audit event：Passed。
- 双入口登录 HTML 原型：3 个 Playwright 测试通过，覆盖 registered/pending Profile、approval/Permit 切换、无常规 Node ID 输入和 1024×768 布局；仅作设计证据。
- `git diff --check`：Passed，仅有仓库既有 Windows EOL 提示。
- 最新 `master` 的 Explorer resource tree、Coupled Seam BrandMark 与本分支完成冲突合并；合并后重新运行全仓 Go、targeted race、`go vet`、frontend tests/build、binding generation 和 Wails clean package，全部 Passed。

## 潜在影响

- 单一 Authority 是新准入与准入管理的可用性/容量集中点；已有 Join 不受其实时可用性影响。
- 当前 Authority 使用有界原子 snapshot，100,000 记录上限是容量保护而非高吞吐 SLO。
- 原生/嵌入式客户端仍走 Legacy Join；在后续迁移完成前不能删除兼容入口。
- 新双入口登录原型尚未进入生产 React；当前生产界面仍是已复验的 endpoint-first 三阶段版本。

## 回滚方案

- 关闭 `MFHE` listener dispatch 和 `system/admission/*` 管理入口，保留既有 `MFH4` Join。
- 保留版本化 Device/Enrollment/Profile 数据；不删除已生成密钥、Grant、Node ID 或 tombstone，避免回滚后身份重用。
- Desktop Admission Console 可从 capability/navigation 中隐藏；Legacy Profile 与 binding entry point 保持可用。
- 若 Authority 状态损坏或密钥不匹配，停止服务并从受验证备份恢复，不静默重建或本地重新分配 ID。

## 子Agent执行轨迹

未使用子 Agent。讨论、规划、实施、验证、UI 操作、归档与集成均由当前主 Agent 在专用 worktree 中完成。
