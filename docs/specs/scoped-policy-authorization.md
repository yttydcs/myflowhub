# Scoped Policy Authorization Spec

## Domain Model

`PolicyState v2` 持久化 `generation`、精确 Grants、Definitions 与 Bindings。Definition 包含稳定 ID、label、optimistic revision、immutable 标记和至多 128 条规则；Binding 包含 16-byte hex ID、精确 Subject、Definition ID、owner scope、创建 actor/time 与可选 expiry。

作用域不嵌入 Definition：同一个 `metrics-reader` Definition 可以分别绑定到单个 owner、一个 relay subtree 或整个 Authority domain。Binding 引用 Definition 的当前 revision；更新 Definition 会统一影响引用它的 Bindings，并提高 policy generation。

## Evaluation

```text
normalize(Subject, Capability, ResourceID)
  → exact grant map
  → Subject binding index
  → current tree ScopeContains(anchor, resource owner)
  → bounded Definition rule match
  → allow explanation / default deny
```

精确 Grant 为 O(1)。Binding 只扫描当前 Subject 的有界 Binding/rule 集合。Prefix 使用 `name == prefix || strings.HasPrefix(name, prefix + "/")`，所以不会越过 Resource path segment。Binding-dependent evaluation 没有拓扑 resolver 时返回拒绝；精确 Grant 不依赖 resolver。

允许解释包含 source（`exact-grant` 或 `binding`）、policy generation；Binding 命中时还包含 topology membership epoch、Binding ID、Definition ID/revision、rule index 与 scope。解释不返回规则外的敏感内容。

## State And Migration

- policy state 使用自己的 version 2，不修改 identity/trust/admission 的 state version。
- 新状态从 generation 1 开始，只有不可变 `superadmin` Definition，没有 Binding。
- v1 加载先验证和规范化全部 exact grants，再提高 generation、插入未绑定 superadmin，并原子写回 v2。
- mutation 顺序是 clone → validate → persist → swap/index → generation notify；持久化失败不发布内存状态。
- Binding expiry 由 PolicyState 定时清理并作为一次 policy mutation 提高 generation。
- generation 使用 JavaScript-safe Collection revision 上限；耗尽后显式失败。

## Topology Scope Resolver

`runtime/tree.State.ScopeContains` 在一把 read lock 内验证 anchor/candidate 是否仍可达并遍历权威 parent 关系，返回同一快照的 membership epoch。attach、announce、withdraw、reparent 与 detach 都改变 membership epoch。

- `owner`: anchor 必须等于 Resource owner 且当前可达。
- `subtree`: owner 必须是 anchor 本身或当前后代。
- `authority-domain`: anchor 必须是本地 Policy Authority，owner 必须在其当前树中。

Hub 在注册管理 Resources 和打开 Listener 前只绑定一次 resolver。未知或已经脱离的 Node 是 non-match；不得通过 Hub 的上游默认路由推断它仍属于本 Authority。

## Resources

| Resource | Type | Capabilities |
| --- | --- | --- |
| `system/policy/definitions` | `mfh.collection` | `create/delete/get/list/update` |
| `system/policy/bindings` | `mfh.collection` | `create/evaluate/get/list/revoke` |
| `system/policy/grants` | `mfh.collection` | `create/get/list/revoke` |
| `system/policy/grant` | `mfh.command` compatibility alias | `invoke` |
| `system/policy/revoke` | `mfh.command` compatibility alias | `invoke` |

Collection list 使用 policy generation 作为 revision 和不透明 generation-bound cursor；get 返回具体 Definition/Binding/Grant schema。所有 mutation handler 使用 `OperationRequest.Subject`，并二次调用 `HasAuthoritySuperadmin(subject, local Authority)`。list/get/evaluate 只接受普通 capability authorization。

## Schemas

- `mfh.policy.definition.v1` / `definition-put.v1` / `definition-delete.v1`
- `mfh.policy.binding.v1` / `binding-create.v1` / `binding-revoke.v1`
- `mfh.policy.grant.v1`
- `mfh.policy.evaluate-request.v1` / `evaluation.v1`

机器真相位于 `protocol/schema_policy.go`、`protocol/data_schema.go` 与 generated binding/schema artifacts。

## SDK, Desktop And Bootstrap

Go `PolicyClient` 是同一个 attached `Client` 的 typed facade；所有方法仍走 Node 的 canonical optimized `Operate` path。generic bindings 的 `OperateJSON` 保持跨平台基线。

Desktop Policy Console 从 Authority Collections 读取 Definitions、Bindings、Grants 和 effective preview；创建 all/superadmin 与撤销都有明确警告。UI 不根据 Profile 或产品名推断权限，Authority `Forbidden` 是最终结果。

停止 Hub 后：

```powershell
mfh-hub -id 1 -state <dir> -policy show
mfh-hub -id 1 -state <dir> -policy bind -subject 41 -definition superadmin -policy-scope authority-domain
mfh-hub -id 1 -state <dir> -policy revoke-binding -binding-id <id>
```

离线 bind 默认 anchor 为本 Hub，并输出稳定 Binding ID 与 resulting generation。子节点拓扑不持久化，因此离线 bootstrap 不接受非本 Hub anchor。旧 exact `-policy grant/revoke` 保持兼容。

## Deferred

Explicit deny、Definition 继承、Subject groups、非 superadmin 有界 delegation、跨 Authority federation 和 Android 专用管理 UI 不属于 phase one。
