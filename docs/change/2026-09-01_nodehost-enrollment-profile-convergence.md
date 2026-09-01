# 2026-09-01 NodeHost、Enrollment 与 Profile 生命周期收敛

## 变更背景 / 目标

Desktop 的 Legacy Profile 已经使用 Parent-only NodeHost，但 authority Profile 在获得 Grant 后仍由 Enrollment binding 持有普通 Node runtime，造成两套连接、重试、关闭和身份事实源语义。本轮把 Enrollment 收窄为尚无 Node ID 时的 bootstrap；Grant 持久化后关闭 bootstrap，并让通用 NodeHost 从同一受保护 credential 接管普通运行时，不改变 MFHE/MFH4、Permit/Pending、集中式 Authority 或已有 Profile 格式。

## 具体变更内容

- `runtime/auth` 新增只读、克隆、防冲突的 `NodeCredentialSource`，只接受完整且已验证的 Enrollment Grant；missing/device/pending、损坏、签名失败和身份冲突均 fail closed。
- `host/nodehost` 新增 credential-backed 构造模式，在启动 Listener/Parent 前解析 Node identity 与直接父锚点；不生成或复制 Legacy identity 文件。
- `sdk/bindings` 新增只暴露状态、Enroll 与 Close 的窄 `EnrollmentBootstrap`；旧 owning API 继续作为未迁移调用方的兼容入口。
- Desktop authority Profile 在 Grant 保存后先关闭 bootstrap，再创建 Parent-only NodeHost 与 attached Client；已 enrolled Profile 重启直接进入 Host，重复 Connect 复用同一 ParentSupervisor。
- 回归门禁覆盖 credential 缺失/损坏/冲突、单一 owner、无重复 identity、Permit/Pending handoff、失败回滚、重启和重复 Connect。
- 稳定 feature、requirements、specs、ADR 和 intake 已同步；旧 Desktop post-Grant owning 例外正式关闭。

## Docs root

- `D:\project\MyFlowHub3\worktrees\nodehost-enrollment-convergence\docs`，属于 canonical MyFlowHub monorepo。
- 本轮只执行本地归档、提交、主线合并和 worktree cleanup；未授权 remote、push、release、publication 或 deployment。

## Intake impact

- updated — 新增并索引 NodeHost、Enrollment 与 Profile 生命周期收敛的原始请求记录。

## Feature impact

- updated — Desktop 当前行为明确 authority 与 Legacy Profile 在 Grant 后都由 Parent-only NodeHost 运行。

## Requirements impact

- updated — 受控准入、Desktop 工作区与统一节点运行时需求明确 Grant 单一事实源、bootstrap/Host 交接和幂等 Connect。

## Specs impact

- updated — Desktop Profile、Enrollment/Authority、NodeHost 与 operational lifecycle 技术合同同步完成。

## Decision impact

- updated — 新增 Enrollment bootstrap → NodeHost handoff ADR，并关闭通用 NodeHost ADR 中记录的临时 Desktop 例外。

## Lessons impact

- updated — 扩充 Desktop binding 重连诊断，并把 GUI 控制运行时初始化失败、PowerShell profile 输出污染结构化输入加入 Windows 前端验证排查入口。

## Related intake

- [NodeHost、Enrollment 与 Profile 生命周期收敛](../intake/2026-09-01_nodehost-enrollment-profile-convergence.md)

## Related features

- [Desktop](../features/desktop.md)

## Related requirements

- [受控准入](../requirements/auth-controlled-admission.md)
- [Desktop 资源工作区](../requirements/desktop-resource-workspace.md)
- [统一节点运行时](../requirements/unified-node-runtime.md)

## Related specs

- [Desktop Profile 入口](../specs/desktop-profile-entry.md)
- [Node Enrollment 与 Admission Authority](../specs/node-enrollment-and-admission-authority.md)
- [NodeHost Runtime](../specs/node-host-runtime.md)
- [Operational lifecycle](../specs/operational-lifecycle.md)

## Related decisions

- [Enrollment bootstrap → NodeHost handoff](../decisions/2026-09-01_enrollment-bootstrap-nodehost-handoff.md)
- [通用 NodeHost、非 owning SDK Client 与薄平台适配](../decisions/2026-08-30_generic-node-host-and-non-owning-sdk-client.md)

## Related lessons

- [Desktop Binding 重连与准入诊断](../lessons/desktop-binding-reconnect-and-admission-diagnostics.md)
- [Frontend And PowerShell Preflight](../lessons/frontend-and-powershell-preflight.md)

## Related plan

- [实施与验证计划](../plan/plan_archive_2026-09-01_nodehost-enrollment-profile-convergence.md)

## 对应 plan.md 任务映射

| Task ID | 结果 |
| --- | --- |
| DOC01 | intake、feature、requirements、specs、ADR、lessons 与索引收敛完成。 |
| AUTH01 | fail-closed、只读、克隆的 enrolled Node credential source 完成。 |
| HOST01 | NodeHost credential-backed 模式、身份/父锚校验与无重复 identity 完成。 |
| BOOT01 | 窄 Enrollment bootstrap、取消和关闭所有权完成。 |
| DESK01 | Desktop bootstrap → Parent-only NodeHost handoff、幂等连接与状态投影完成。 |
| REG01 | ownership、迁移、失败回滚、重启和架构回归门禁完成。 |
| QA01 | full/vet/race/generated/frontend/Wails、真实 TCP 与 packaged GUI 门禁完成。 |
| ARC01 | change/plan/lesson 归档、local integration 与 cleanup 由本阶段处理。 |

## 经验 / 教训摘要

- Enrollment 是 Grant 前的 pre-auth bootstrap，不是 Grant 后的普通 Node runtime owner。
- Grant credential 同时约束 Node identity、直接父锚点和 Authority provenance；Profile 缓存只能校验或展示，不能覆盖事实源。
- 注册成功不等于获得 Resource 权限。真实 GUI 中两个新节点都被默认拒绝读取 `system/topology`，证明准入与资源授权保持分离。
- 测试工具在第一次产品交互前失败应归类为 harness/environment；shell profile 的额外 stdout 不能直接作为结构化 GUI 输入。

## 可复用排查线索

- 症状：已 enrolled Profile 重连出现 `already started` 或再次要求 Permit。关键词：`EnrollmentBootstrap`、`NodeCredentialSource`、`ParentSupervisor`。快速检查：Grant 后是否先关闭 bootstrap，并从 credential 创建 Host。
- 症状：authority Profile 生成 `identity.dpapi`。关键词：`enrollment.dpapi`、duplicate identity。快速检查：credential-backed Host state 中只能存在 Enrollment credential，不能复制 Legacy identity。
- 症状：GUI 控制在首次点击前报 `failed to write kernel assets`。触发：控制运行时初始化失败。快速检查：产品进程是否已正常启动、控制 runtime 是否在交互前失败；不要把它记为产品回归。
- 症状：GUI 返回 `permit_json must be valid bounded JSON`，但 Permit 文件本身有效。触发：PowerShell profile 输出混入文件内容。快速检查：使用无 profile shell读取并先解析 JSON，再填入界面。

## 关键设计决策与权衡

- 采用通用 NodeHost 的 credential source 配置，而不是新增 `EnrollmentHost`、`LeafHost` 或 Desktop 专用运行时。
- bootstrap facade 刻意不暴露普通 Resource operation，换取 Grant 前后所有权边界可由类型和 architecture test 证明。
- 旧 owning binding API 暂留以保持未迁移 Android/Embedded/下游源码兼容；全面删除需要独立 downstream 审计。
- settings v2 的兼容缓存仍保留，但任何非空字段必须与 credential 完全匹配；settings v3 clean break 延后。

## 测试与验证方式 / 结果

- `go test ./... -count=1`、`go vet ./...`、focused `-race`：通过。
- canonical generated freshness、Desktop frontend 134 tests、Vite production build、Windows/amd64 Wails production build：通过。
- 真实 Hub TCP Permit、Pending、Grant、restart 与重复 Connect：通过。
- packaged Desktop GUI：Permit 注册、断开/重连、进程重启自动连接、Pending 等待、Authority 准入页批准、批准后检查并连接、再次重启自动连接：通过。
- 安全事实：两个 Profile 均只有受保护 Enrollment credential、没有 `identity.dpapi`；两个新 Node 均被默认拒绝读取 `system/topology`。
- 最终 focused Go 回归、`gofmt -d`、`git diff --check`、敏感路径检查和测试进程/端口清理：通过。
- 真实 GUI 证据针对 Desktop ↔ Hub/NodeHost 准入链路；不额外宣称 Metrics 数据采集业务链路的 GUI 覆盖。

## 潜在影响

- 旧 runtime-owning bindings 与新 bootstrap/attached Client 在迁移期并存；新增产品必须走 NodeHost canonical path。
- Android/Embedded Enrollment、旧 API 全量删除、settings v3 和多 Authority/reparent 不在本轮范围。
- GUI 防止活动连接重复提交；精确并发重复 Connect 由 Go 回归测试覆盖。

## 回滚方案

- 可按 Desktop handoff → binding bootstrap facade → NodeHost credential mode → auth credential source 的逆依赖顺序回退。
- 回退不需要删除或重发用户 Grant、Node ID、Profile 或 Permit；本轮没有 wire、Profile schema 或 Authority state migration。
- 若主线集成冲突，保留功能提交和 worktree，先恢复主检出未提交内容，不覆盖用户修改。

## 子Agent执行轨迹

- 未派发子 Agent。实现、GUI 联调、审查、文档治理与归档均由主 Agent 在批准的 Task ID 范围内完成。

## Closeout status

- Implementation commit：`b94fe34 refactor: 收敛 Enrollment 与 NodeHost 生命周期`。
- Archive commit：`a0c2fe7 docs: 归档 NodeHost Enrollment 生命周期收敛`。
- Master integration：`master` 从 `a9eb270` fast-forward 到 `a0c2fe7`；本记录与最终控制状态由后续 closeout commit 固化。
- Preservation：主检出修改先保存为临时 stash `58e8bb8`，在 detached preview worktree 上重放成功；4 个非重叠 tracked 文件与 24 个 untracked 文件逐 blob 一致，4 个重叠文档的增删计数一致且无冲突。master 重放后再次得到相同结果，临时 stash 已删除；两个既有 stash 未触碰。
- Remaining local state：用户的 8 个 tracked 修改和 24 个 untracked 文档/设计文件仍以未暂存状态保留在主检出，没有进入本 workflow 提交。
- Cleanup：preview worktree、`worktrees/nodehost-enrollment-convergence` 和已合并的 `refactor/nodehost-enrollment-convergence` 分支均已删除。
- Publication：local-only；未授权 push、release 或 publish。
