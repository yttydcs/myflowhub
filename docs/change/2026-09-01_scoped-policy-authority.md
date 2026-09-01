# 2026-09-01 作用域策略定义与 Authority 权限绑定

## 变更背景 / 目标

当前 vNext 授权只持久化精确 `(Subject, Capability, Resource Owner, Resource Name)` Grant。它能保持默认拒绝，却要求 Desktop 每遇到新 Node、Resource 或 Capability 都重新展开授权。本轮在不制造 Desktop、Metrics 或 Agent Gateway 特权产品的前提下，引入可复用 Policy Definition、精确 Subject Binding 与动态 owner scope，并为 Desktop Node 41 显式建立可撤销、可持久的 Authority-domain superadmin Binding。

## 具体变更内容

- `protocol` 增加有界 Policy Definition/Binding/Grant/Evaluation schemas、selector、owner scope 和 Collection capabilities，并纳入 canonical generated contracts。
- `runtime/auth` 将 policy state 升级到 v2：保留 exact Grants，插入默认未绑定且不可变的 `superadmin` Definition，增加持久 Definition/Binding、expiry、generation 与可解释 evaluator。
- `runtime/tree` 提供单锁、单快照的 `ScopeContains` 与 membership epoch；Hub 在开放管理 Resource/Listener 前绑定本地 Authority resolver。
- `feature/management` 暴露 definitions、bindings、grants 三个 Policy Collections，并对所有在线 mutation 二次验证真实路由 Subject 的 Authority-domain superadmin Binding；legacy grant/revoke aliases 保持兼容。
- Go SDK 在同一个 attached Client/Node 路径上增加 typed `PolicyClient`；Desktop 增加 Policy Console 与 generated schema；停止 Hub 的 CLI 支持 show/bind/revoke-binding 和兼容 exact grant/revoke。
- 真实本机状态把 Node 41 显式绑定到 `superadmin + authority-domain:1`。原有 70 条 exact Grants 保留，便于独立回滚与后续 `CLEAN02` 决策。

## Docs root

- `D:\project\MyFlowHub3\worktrees\scoped-policy-authority\docs`，属于 canonical MyFlowHub monorepo。
- 本轮只执行本地归档、提交、主线合并与 worktree cleanup；未授权 remote、push、release、publication 或 deployment。

## Intake impact

- updated — 新增并索引作用域策略角色与持久 Binding 的原始讨论、历史模型对比和目标约束。

## Feature impact

- updated — Hub 当前行为增加 Policy Collections、显式 bootstrap 与动态作用域裁决；Desktop 当前行为增加 Authority-backed Policy Console，且仍不根据产品名判断权限。

## Requirements impact

- updated — 新增 scoped policy authorization 的长期需求，并保持 Enrollment/登录不自动授予业务权限。

## Specs impact

- updated — 新增 PolicyState v2、selectors、evaluation、topology scope、management Resources、SDK/UI/CLI 与迁移合同；protocol map 同步。

## Decision impact

- updated — 新增“使用作用域 Policy Binding，而不是产品特权”的 ADR。

## Lessons impact

- updated — 新增策略状态迁移、停机 bootstrap 与真实授权证明的可复用排查入口。

## Related intake

- [作用域策略角色与持久绑定](../intake/2026-09-01_scoped-policy-roles-and-bindings.md)

## Related features

- [Hub](../features/hub.md)
- [Desktop](../features/desktop.md)

## Related requirements

- [Scoped Policy Authorization](../requirements/scoped-policy-authorization.md)
- [Auth Controlled Admission](../requirements/auth-controlled-admission.md)

## Related specs

- [Scoped Policy Authorization Spec](../specs/scoped-policy-authorization.md)
- [Protocol map](../specs/protocol_map.md)
- [节点树、链路与资源架构](../specs/node-tree-link-resource-architecture.md)

## Related decisions

- [使用作用域 Policy Binding，而不是产品特权](../decisions/2026-09-01_scoped-policy-bindings-over-product-privilege.md)
- [通用 NodeHost 与非 owning SDK Client](../decisions/2026-08-30_generic-node-host-and-non-owning-sdk-client.md)

## Related lessons

- [策略状态迁移与真实授权证明](../lessons/scoped-policy-state-migration-and-live-proof.md)
- [Authority 管理必须沿节点树执行](../lessons/authority-local-admin-actions.md)

## Related plan

- [作用域策略实施计划](../plan/plan_archive_2026-09-01_scoped-policy-authority.md)

## 对应 plan.md 任务映射

| Task ID | 结果 |
| --- | --- |
| DOC01 | intake、features、requirements、specs、ADR、change、lesson 与索引完成。 |
| PROTO01 | Policy schemas、selectors、scope、evaluation 与 canonical schema 注册完成。 |
| AUTH01 | PolicyState v2、无损迁移、Definitions/Bindings、expiry 和 evaluator 完成。 |
| TREE01 | 权威树原子 scope resolver、membership epoch 与 Hub attachment 完成。 |
| MGMT01 | Policy Collections、二次 superadmin guard、兼容 aliases 与安全审计完成。 |
| SDK01 | attached Client typed facade 和 generated binding/schema contracts 完成。 |
| DESK01 | Authority-backed Policy Console、危险操作警告和 Forbidden 状态完成。 |
| BOOT01 | 停止 Hub 的 show/bind/revoke-binding 与 exact grant compatibility 完成。 |
| QA01 | full/vet/race/generated/frontend/Wails/isolated integration 门禁通过。 |
| LIVE01 | Node 41 Binding 持久化、Hub/Metrics/Desktop 双重启、read/subscribe/effective deny/allow 证明通过。 |

## 经验 / 教训摘要

- 身份、Enrollment 与 Resource authorization 是三个独立事实；固定 Node ID、产品名、首个登录或成功注册都不能隐式获得 superadmin。
- 作用域成员关系必须直接读取 Authority 当前权威树的一个原子快照，不能从异步管理 projection 重建第二棵权限树。
- v1 → v2 迁移必须先从零值容器解码并完整验证，再 persist/swap；不能向已经带默认项的内存对象解码，否则会制造重复或伪造状态。
- 真实高权限 smoke 不能只验证已有 exact Grant 覆盖的 Metrics Resource；必须让 effective preview 命中一项以前没有精确授权的新 Resource，并另测未授权 Subject。

## 可复用排查线索

- 症状：Desktop 重启后再次逐项 `permit/grant`，新 Node 仍 `forbidden`。关键词：`source=none`、`authority-domain`、`policy generation`。快速检查：区分 Enrollment 与 Policy Binding，并检查 Subject/Definition/scope 是否持久存在。
- 症状：Binding 存在但新 Resource 不生效。触发：owner 不在当前 scope、resolver 未绑定或 prefix 越段。快速检查：读取 effective evaluation 的 Binding ID、topology epoch、rule index 和 scope；确认 `metrics` 不会匹配 `metrics-evil`。
- 症状：CLI bootstrap 后在线 Hub 覆盖状态或迁移失败。触发：Hub 未停止或旧 state 解码路径复用了预初始化容器。快速检查：确认端口/进程已停止，记录迁移前 exact Grant 数量、Binding ID 和 resulting generation。
- 症状：策略写 Resource 已精确授权但 mutation 仍 Forbidden。触发：调用者没有本 Authority-domain superadmin Binding。快速检查：这是预期的二次越权保护，不应通过 UI 或 compatibility alias 绕过。

## 关键设计决策与权衡

- phase one 使用 allow-only Definition + Binding 与 exact Grant 并集，避免 deny、继承和 group precedence 尚未冻结时引入不可解释结果。
- 内建 `superadmin` 使用 all/all 以自然覆盖未来 Capability，但默认永远不绑定；UI/CLI 显式提示其未来扩张语义。
- 在线策略写采用普通 Resource authorization 加 Authority-domain superadmin 二次 guard；换取保守、安全的 bootstrap，非 superadmin 有界委派延后到 `DELEG02`。
- Node 41 的旧 exact Grants 暂不清理；这保留了撤销新 Binding 的独立回滚路径，代价是短期存在两种授权来源。

## 测试与验证方式 / 结果

- `GOWORK=off go test ./... -count=1`：通过。
- `go vet ./...`：通过。
- `go test -race ./protocol ./runtime/auth ./runtime/tree ./feature/management ./host/hub ./sdk/go -count=1`：通过。
- `go test ./sdk/bindings` generated freshness：通过。
- Desktop frontend：18 files / 136 tests；Vite production build：通过。
- Desktop Windows Wails production build：通过。
- isolated product matrix：当前 Metrics Node、未来 Metrics Node、Binding revoke 与默认拒绝：通过。
- live smoke：Hub 停机写入 Binding `b3df4b135fdf4f3589f9c66aece5ce6a`，generation 77；Hub/Metrics/Desktop 双重启，既有 Profile 无 Permit 自动连接，Metrics read/subscribe 成功，新 Policy Collection 的 effective source 为该 Binding，Subject 999999 保持拒绝；日志无 secret/Permit 字段和 forbidden/panic/fatal/error。
- `gofmt -l`、`git diff --check` 和临时 live test 残留检查：通过。

## 潜在影响

- `superadmin` all/all 会覆盖作用域内未来 Capability；必须把 Binding 当作高风险、可审计、可撤销状态。
- 当前显式 deny、Definition 继承、Subject groups、delegation ceiling、跨 Authority federation 与 Android 专用管理 UI 均未实现。
- 现有 legacy Desktop Profile 没有显式 `authority_node_id` 时不会显示专用 Settings Policy Console，但同一 Node 的 canonical API/Resource 授权仍正常；Profile 迁移不应通过猜测 parent=Authority 完成。
- live Binding 位于用户本机 Hub state，不属于 Git；分支回滚不会自动撤销它。

## 回滚方案

- 代码按 Desktop → SDK/management → Hub/tree resolver → auth state → protocol 的逆依赖顺序回退。
- 停止 Hub 后执行 `mfh-hub -id 1 -state <hub-state> -policy revoke-binding -binding-id b3df4b135fdf4f3589f9c66aece5ce6a`，再重启；70 条 exact Grants 保持不变。
- PolicyState v2 代码回退前必须评估 v2 文件兼容性；不要用旧二进制并发打开已迁移状态。

## 子Agent执行轨迹

- 未派发子 Agent。用户未授权 delegation；实现、测试、LIVE01、文档治理和归档均由主 Agent 在批准 Task IDs 内完成。

## Closeout status

- Implementation commit：`408afb5 feat: 引入作用域策略定义与权限绑定`。
- Archive commit：将在本归档文件与索引完成后创建。
- Master integration / cleanup：待 archive commit 后从 canonical control checkout 执行。
- Publication：local-only；仓库无 remote，未授权 push、release、publication 或 deployment。
