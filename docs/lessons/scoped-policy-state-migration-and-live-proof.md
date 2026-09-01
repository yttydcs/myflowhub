# 策略状态迁移与真实授权证明

## Summary

从逐 Resource exact Grants 升级到 Definition/Binding 时，安全边界不只是“新 schema 能加载”。必须同时保证旧授权无损、默认不产生新权限、动态 scope 使用当前权威树、首个管理员只能停机显式建立，并通过此前未精确授权的新 Resource 证明 effective source 确实来自 Binding。

## Lookup Hints

- `forbidden: subject ... cannot read`
- `source=binding`、`source=exact-grant`、`source=none`
- `authority-domain`、`topology_epoch`、`policy_generation`
- `policy.json version 1`、`PolicyState v2`
- `superadmin`、`revoke-binding`、新 Node 仍无权限

快速检查：先确认身份/Enrollment 已稳定，再在 Hub 停止时执行 policy show，记录 exact Grant 数量、Binding ID 和 generation；重启后 evaluate 一项旧 exact Grants 未覆盖的新 Resource，并用另一个 Subject 复测 default deny。

## Symptoms

- Desktop 已登录且能看到 Node，却读取 Metrics 报 Forbidden；每新增 Node/Resource 都要手工追加 grant。
- 创建了 superadmin Binding，但 effective preview 仍显示 `none` 或只显示 `exact-grant`。
- v1 state 升级后 exact Grant 数量变化、默认出现管理员，或第二次启动损坏。
- Binding 在文件中存在，但 reparent/withdraw 后仍错误生效，或新加入 scope 的 Node 不生效。
- 给策略管理 Resource 精确 create/invoke 权限后仍不能修改策略。

## Impact

- 可能造成隐式提权、撤权不生效、未来 Resource 漏授权、每次启动重复配置，或通过 compatibility alias 绕过策略管理边界。

## Trigger Conditions

- 旧 policy state 首次从 v1 加载到 v2。
- 从 exact Grants 切换为 owner/subtree/authority-domain Binding。
- Hub 在线时尝试离线 bootstrap，或迁移代码向带默认项的容器解码。
- scope evaluator 使用缓存 projection，而不是 Authority 当前树的原子快照。
- live smoke 只访问已有 exact Grant 覆盖的 Resource。

## Root Cause

- 把产品、身份或成功 Enrollment 误当成授权来源。
- 把 Definition 的规则和 Binding 的 owner scope 混为一体，或维护第二棵权限树。
- migration 未遵循 decode-zero → validate → normalize → persist → swap。
- 验证只看“请求成功”，没有检查 effective source、generation/topology epoch 和独立 denied Subject。
- 在线 mutation 只检查目标 Resource capability，没有二次限制授予者的 Authority-domain superadmin 身份。

## Investigation Trail

1. 确认 Subject Node ID、Authority ID、Resource owner/name/capability，不使用产品名推断权限。
2. 停止 Hub并确认 listener 已释放；执行 policy show，记录 state version、generation、exact Grants、Definitions、Bindings。
3. 检查 Binding 的精确 Subject、Definition ID、scope kind/anchor、expiry 与当前 Authority topology。
4. evaluate 目标 Resource，读取 `source`、Binding/Definition ID、rule index、policy generation、topology epoch。
5. 用 prefix 边界和 owner detach/reparent 复测 dynamic scope；用另一 Subject 复测 default deny。
6. 若 migration 异常，检查是否从零值 state 解码、是否在完整校验前写回、persist 失败是否仍 swap 了内存状态。

## Resolution

- PolicyState v2 保留 exact Grants，新增默认未绑定的 immutable superadmin、Definitions 和 Bindings；所有 mutation persist-before-swap 并提升 generation。
- Scope 解析由 Authority Node tree 在同一 read lock 下完成，并返回 membership epoch；未知/脱离/缺 resolver 都 fail closed。
- 首个高权限 Binding 只在 Hub 停止时显式创建并记录 ID/generation；在线 mutation 再验证真实 Subject 的 Authority-domain superadmin Binding。
- live proof 使用 Node 41 对新增 Policy Collection 的 `list` evaluation 证明 `source=binding`，并分别验证 Metrics read/subscribe 与 unauthorized Subject deny。

## Prevention / Guardrails

- 永远分开记录 identity、admission 和 authorization 事实；产品名、安装包和首个登录不是 policy selector。
- migration 测试必须断言旧 exact Grant 集合完全相等、新 Definition 未绑定、无自动 allow、二次加载稳定。
- scope 测试必须覆盖 prefix segment、future child、detach/reparent、missing resolver 和 membership epoch。
- 高权限 live mutation 必须先停服务、记录 rollback ID、保留原授权作为回滚保护，并在重启后再次 show/evaluate。
- 策略管理 handler 必须使用真实 `OperationRequest.Subject`；不使用会丢失主体的 convenience command wrapper。

## Related Docs

- [作用域策略变更归档](../change/2026-09-01_scoped-policy-authority.md)
- [作用域策略需求](../requirements/scoped-policy-authorization.md)
- [作用域策略技术合同](../specs/scoped-policy-authorization.md)
- [作用域 Binding ADR](../decisions/2026-09-01_scoped-policy-bindings-over-product-privilege.md)
- [Hub](../features/hub.md)
- [Authority 管理必须沿节点树执行](authority-local-admin-actions.md)
