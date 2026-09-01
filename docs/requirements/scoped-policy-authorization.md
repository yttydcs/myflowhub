# Scoped Policy Authorization

## Goal

MyFlowHub 的权限必须属于网络中的明确 Subject，而不是属于 Desktop、Metrics、Agent Gateway 或安装包。系统需要在保留精确授权的同时，提供可复用、可持久化、可解释的权限组，并让新加入作用域的 Node/Resource 自动受既有规则约束。

## Stable Requirements

- 所有产品都是普通 Node；产品名、进程形态、UI、是否拥有 Listener 都不构成权限。
- `Policy Definition` 是一组有界的 Resource/Capability 规则，不是网络角色或 Node 类型。
- `Policy Binding` 把一个 Definition 分配给一个精确 Subject Node ID，并指定 Resource owner 的 `owner`、`subtree` 或 `authority-domain` 作用域。
- Resource selector 只支持 exact、path-segment-safe prefix 和 all；Capability selector 只支持有序 exact set 和 all。
- 精确 `(Subject, Capability, Resource Owner, Resource Name)` Grant 保持兼容，与 Binding 授权取并集；未匹配时默认拒绝。
- 作用域必须针对 Authority 当前权威树的单个原子快照计算，不得复制第二棵权限树。
- Definition、Binding 或精确 Grant 的任何有效变化必须提高 policy generation，使旧 generation 绑定的 subscription/session 明确失效。
- 内建 `superadmin` Definition 是不可变的 all/all 规则，但升级、Enrollment、首次 Desktop 或固定 Node ID 都不得自动获得 Binding。
- 在线策略写操作除了普通 Resource capability 裁决，还必须验证真实路由 Subject 具有本 Authority 的 `authority-domain` superadmin Binding。给策略 Resource 一个精确 create/invoke Grant 不能绕过此检查。
- 首个 superadmin Binding 只能在 Hub 停止时通过显式离线操作创建；操作必须输出 Binding ID 和 generation，并可按 Binding ID 回滚。

## Security And Capacity

- 缺失拓扑 resolver、未知/脱离作用域、损坏状态、未知 Definition、revision 冲突与持久化失败都 fail closed。
- Definition、rule、Binding、Grant、分页和 payload 均有硬上限；phase one 不支持 regex、脚本表达式、显式 deny、继承或 Subject group。
- 审计只记录 actor、Resource、目标记录 ID 与结果，不记录凭据、Permit、密钥或 operation payload。
- Authority 的 policy state 是事实源；Desktop UI、catalog visibility 与 descriptor `permission` 字符串仅用于展示和兼容，不能给出授权结论。

## Acceptance

- v1 policy state 升级后保留所有精确 Grants，增加一个未绑定的不可变 superadmin Definition，不产生新权限。
- 一个显式绑定 `superadmin + Subject 41 + authority-domain:1` 的 Desktop 能访问该域内当前及后来加入的 Resources，重启后 Binding 仍存在。
- Prefix `metrics` 匹配 `metrics/cpu_percent`，但不匹配 `metrics-evil/cpu_percent`。
- 撤销 Binding、更新 Definition 或到期清理提高 generation；owner 脱离 subtree 后立即不再匹配。
- 只有精确策略写权限、没有 Authority-domain superadmin Binding 的调用者不能修改任何 Definition、Binding 或 Grant。
- Agent Gateway 可作为普通 Node 给多个 Codex/agent 发行它自己的 API token；这些 token 的网络权限最终仍映射为明确 Subject/Binding，不使 Gateway 成为特权产品。

## Canonical Contract

技术模型、wire schemas、Resources、迁移与 bootstrap 见 [Scoped Policy Authorization Spec](../specs/scoped-policy-authorization.md)。首次身份注册与权限分配保持分离，见 [Auth Controlled Admission](auth-controlled-admission.md)。

## Related Change And Lesson

- [作用域策略定义与 Authority 权限绑定](../change/2026-09-01_scoped-policy-authority.md)
- [策略状态迁移与真实授权证明](../lessons/scoped-policy-state-migration-and-live-proof.md)
