# 2026-09-01 作用域策略角色与持久绑定

## Source

- 来源：用户在当前 Codex 会话中追问 Desktop 读取 Metrics Resource 时的 `Forbidden`、是否存在超级管理员、身份和权限为何需要重复配置，以及重构前角色模型能否在当前架构下恢复。
- 日期：2026-09-01。
- Canonical repository：`repo/MyFlowHub`。
- 讨论分支：`refactor/scoped-policy-authority`。
- 讨论基线：`70add9b`。

## Original Request / Source-preserving Summary

用户记得重构前可以为节点授予 `superadmin` 等角色，希望在当前统一 NodeHost、普通节点平权、Resource 由 Node 所有、Authority 沿树裁决的视角下重新规划权限模型：Desktop 身份与登录状态应持久，作为管理客户端时可以稳定访问当前网络及未来新加入节点的资源，而不应每新增一个 Node 或 Resource 就重新枚举精确授权。

同时，Desktop、Metrics 和 Agent Gateway 都必须保持普通节点地位；产品名、安装包或是否带 UI 不得隐式带来网络特权。

## Problem / Opportunity

当前 `PolicyState` 只持久化精确的 `(subject NodeID, capability, resource owner NodeID, resource name)` 授权。它安全、明确并支持 policy generation 失效，但存在以下可用性问题：

1. 管理节点要访问现有多个节点时，需要按 catalog 展开大量精确 grant。
2. 新 Node、新 Resource 或新 Capability 加入后，旧 grant 不会自动覆盖。
3. 当前 `AllowAll` 是进程内 Policy 实现，主要用于测试或显式运行时装配，不是可持久、可审计、按主体授予的超级管理员角色。
4. 旧 `superadmin:*`、`admin`、`node` 模型已经随 vNext clean break 退役；直接恢复旧权限字符串会丢失当前 Resource owner、路由 authority、topology epoch 与 policy generation 语义。

重构前确实存在 `superadmin:*`、平铺的 `admin` 权限集合和 `node` 权限集合，并在 cold-start bootstrap 中默认向首个注册者授予 `superadmin`。旧 UI 也提供默认角色、角色编辑、节点覆盖和内建预设。该历史证明角色化管理有现实价值，但旧实现不能原样成为当前协议的授权事实源。

## Goals

- 恢复“一次授予、持续适用于范围内新节点和新资源”的管理体验。
- 保留当前默认拒绝、Resource owner、逐级路由裁决、policy generation 和审计边界。
- 让角色成为策略复用机制，让绑定成为 Authority 持久事实；不创建特权产品或特权节点类型。
- 身份、准入和授权继续分离：Enrollment Grant 负责稳定 Node 身份与树边，Policy Binding 负责资源权限。
- 保留精确 grant，支持最小权限和兼容迁移。
- 让 Desktop、CLI、SDK 和未来 Agent Gateway 使用同一管理资源与同一策略语义。

## Non-goals

- 不让 Desktop、Metrics、Agent Gateway 或某种安装包自动获得管理员权限。
- 不把 admission profile、Permit 或成功登录等同于业务 Resource 授权。
- 不恢复旧多仓权限 token、旧 SubProto handler 或旧 `auth.role_perms` 配置格式。
- 不引入第二棵权限树、独立资源树或绕过父子 authority 的中心化旁路。
- 第一阶段不引入任意正则表达式、复杂显式 deny、角色继承或通用 ABAC 表达式语言。
- 本讨论阶段不修改 runtime、协议、Desktop 或持久状态。

## Current Invariants

- 请求主体由稳定 NodeID 表示；产品名称不是授权主体。
- Resource 以 `(owner NodeID, local resource name)` 唯一定位，Capability 是最终操作边界。
- 跨节点请求在权威路径上裁决，已裁决请求以 control 向 Resource owner 下行；中继本身不因此获得权限。
- Policy 变化必须提升 generation，使依赖旧 generation 的 subscription、session 和 route 状态失效。
- Topology 变化必须以当前 topology epoch 重新判断子树成员，不能缓存永久的“属于某子树”结论。
- 未匹配任何规则时继续 default deny。

## Options Considered

### Option A：继续精确 grant，在新节点加入时自动展开 catalog

- 做法：Enrollment 或 catalog 更新后，为管理节点批量生成新资源的精确 grant。
- 优点：对现有 `PolicyState` 改动最小，授权结果仍完全显式。
- 缺点：动态 Resource 和 Capability 会持续放大 policy 文件；节点加入、catalog 更新与授权生成之间存在竞态；不能自然表达“这个主体管理整棵子树”。
- 结论：可作为短期运维脚本，不作为长期模型。

### Option B：只增加通配/前缀规则，不增加命名角色

- 做法：允许一个主体的规则匹配 `owner subtree`、Resource name 前缀和 Capability 集合。
- 优点：足以解决新节点和新 Resource 的重复配置，runtime 模型相对简单。
- 缺点：多个主体会复制同一组规则；更新、审计、UI 和预览缺少稳定的策略单元；难以表达“给这个节点授予与另一个节点相同的管理员集合”。
- 结论：匹配器可作为底层能力，但不单独作为产品模型。

### Option C：命名策略角色 + 作用域绑定 + 精确 grant

- 做法：Authority 持久保存可复用的 Policy Definition；通过 Policy Binding 把角色授给精确 Subject，并指定 Resource owner 作用域。现有精确 grant 继续作为最小权限与兼容层。
- 优点：恢复旧模型的易用性，同时保留 vNext 的 owner/capability/authority 语义；可审计、可撤销、可预览，并能动态覆盖未来加入作用域的 Node。
- 缺点：需要版本化状态迁移、拓扑作用域解析、新管理资源和 UI；必须定义授权委派上限以避免管理员自行扩大权限。
- 结论：推荐。

### Option D：把 Desktop 或固定 NodeID 特判为 `AllowAll`

- 做法：runtime 识别 Desktop 产品、某个安装包或 Node 41，跳过普通策略判断。
- 优点：实现和本机联调最直接。
- 缺点：制造特权产品/特权节点；身份迁移、重装和多 Desktop 场景脆弱；无法合理撤销和审计；破坏网络内节点平权原则。
- 结论：拒绝。

## Recommended Model

### 1. Policy Definition：角色是规则组合

Policy Definition 是 Authority 管理的命名授权集合，例如：

- `superadmin`：所有 Resource name、所有 Capability；只有配合绑定作用域后才产生实际权限。
- `network-admin`：网络日常管理与业务 Resource 的管理集合；是否包含策略委派需明确配置。
- `observer`：catalog、topology、health 和指定业务数据的只读/订阅集合。

角色名只用于管理、展示和审计，不进入 ResourceID，也不是节点类型。第一阶段采用无继承的平铺规则，避免旧模型中默认权限集合跨模块漂移。

单条规则至少描述：

- Resource name selector：精确名称、受约束的路径前缀或全部。
- Capability selector：精确集合或全部。
- 可委派范围：默认不可委派；允许管理策略的角色必须有显式 delegation ceiling。

不允许把 `*` 伪装成合法 ResourceID 或 CapabilityID；selector 使用独立、可验证的数据结构。

### 2. Policy Binding：把角色授给主体并限定 owner 作用域

Policy Binding 至少包含：

- 精确 Subject NodeID。
- Policy Definition ID 与版本。
- Resource owner scope。
- 签发 Authority、创建者、创建时间和可选过期时间。
- 状态与用于审计的稳定 Binding ID。

Owner scope 第一阶段只提供受约束的三类：

- `owner:<NodeID>`：只覆盖一个 Resource owner。
- `subtree:<NodeID>`：按当前权威拓扑动态覆盖该节点及其后代，包括未来加入的节点。
- `authority-domain:<AuthorityNodeID>`：覆盖该 Authority 当前管理域；仅用于明确的域级管理角色。

不以 Subject 当前位于哪棵子树隐式决定其权限。Binding 始终绑定精确 Subject；Resource owner 是否落入 scope 由裁决点使用当前 topology snapshot 判断。reparent 后必须按新 topology epoch 重算。

### 3. Evaluation：精确授权与角色绑定取并集，未命中即拒绝

第一阶段授权顺序：

1. 规范化并验证 Subject、ResourceID 和 Capability。
2. 检查现有精确 grant。
3. 查询 Subject 的有效 bindings。
4. 使用当前拓扑判断 Resource owner 是否命中 binding scope。
5. 在绑定角色中匹配 Resource name 与 Capability selector。
6. 任一有效 allow 命中则允许，否则返回现有 Forbidden。

第一阶段不加入显式 deny，避免 exact grant、角色规则和继承之间产生难以解释的优先级。需要例外时使用更窄角色或撤销 binding。后续若确认 deny 是硬需求，再以独立决策定义优先级。

### 4. Recommended Desktop Binding

Desktop 不应自动成为 `superadmin`。在当前本地网络中，Authority 可显式、持久地建立：

```text
subject: Desktop NodeID (当前为 41)
policy: superadmin
scope: authority-domain:1
```

这让 Node 41 对当前 Authority 域内已有和未来 Node 的匹配 Resource 自动生效，但仍具备以下性质：

- 它是普通 Subject 的一条可撤销 Binding。
- 更换 Desktop identity 不会继承权限，必须明确迁移 Binding。
- 第二个 Desktop、Agent Gateway 或 CLI 节点可以获得不同角色与作用域。
- Metrics 不需要知道 Desktop 的产品身份，只看到 Authority 已裁决的 control。

针对用户已经确认的当前目标，Node 41 使用 `superadmin + authority-domain:1`；它是一次显式的 Authority 域级授权，不是 Desktop 产品默认值。后续可为日常运维者提供范围更窄的 `network-admin`。`superadmin` 不随首个注册或成功登录自动授予；首次 Authority 初始化可通过离线、安全的本机命令创建第一个持久 Binding，随后所有在线变更走受权管理 Resource。

### 5. Persistence And Migration

- 将当前 policy state 升级为新版本，原有精确 grants 无损保留。
- 新增 definitions 和 bindings；加载时校验重复 ID、未知角色、非法 selector、零 generation 和越界 scope，损坏状态显式失败。
- definition、binding 或精确 grant 的任何有效变化都原子持久化并提升 policy generation。
- 旧 `policy.json` 自动迁移只补结构，不自动给任何现有 Subject 添加管理员 Binding。
- 为当前 Node 41 授权属于显式运维动作，应由后续执行计划单独完成并留下审计记录。

### 6. Management Resources And UI

建议把当前精确 `system/policy/grant` / `revoke` 保留为兼容入口，并新增 schema-driven 管理能力：

- definitions：list/read/create/update/delete。
- bindings：list/read/create/revoke。
- effective policy preview：输入 Subject 与可选 Resource，解释命中的 exact grant、Binding、Definition 和 scope。

Desktop 的 Access Policy 视图可以恢复为“角色 / 主体绑定 / 精确授权”三层，而不是恢复旧配置字符串编辑器。所有 UI 动作都调用普通管理 Resource；Desktop 本地代码不持有授权旁路。

### 7. Delegation Safety

拥有调用策略管理 Command 的权限，不应默认允许授予比自身更大的 scope 或更强的角色。下一阶段必须冻结以下规则之一：

- 保守基线：只有带不可伪造 `superadmin` Binding 的主体能管理 definitions/bindings；或
- 有界委派：Binding 明确记录 delegation ceiling，创建者只能授予自己可委派权限的子集与子作用域。

推荐先实现保守基线，再把有界委派作为独立增强；不能仅依赖“能 invoke `system/policy/bind`”就允许任意 payload。

## Assumptions

- 当前单一逻辑 Authority 与统一权威节点树决策继续有效。
- NodeID 是授权主体的稳定标识；Display name、Profile 名称和产品类型只用于展示。
- Resource name 保持路径风格，可使用段边界明确的 prefix selector；不需要任意 regex。
- 当前 policy generation 和 topology epoch 机制继续作为缓存、session 与 subscription 失效基础。
- 近期目标是单 Authority 域；跨 Authority federation 不在本轮范围。

## Constraints / Risks

- 动态 subtree scope 需要裁决器读取一致的拓扑快照；PolicyState 不能私自维护第二份树关系。
- `all capability` 会自动覆盖未来新增 Capability，便利但扩大兼容与安全风险；必须只出现在明确的高权限角色，并在 UI 中突出显示。
- 删除或修改 Definition 会影响多个 Subject，必须以 generation 统一失效，并在提交前展示 impact preview。
- 如果角色规则继续散落在 Hub、Desktop、SDK 多处，旧模型的默认权限漂移会重现；Definition 的事实源必须只有 Authority 持久状态和协议 schema。
- policy 管理 payload 的越权校验必须在 Authority 服务端完成，UI 隐藏按钮不是安全边界。
- 离线 bootstrap 命令必须要求 Hub 停止并原子写入，不能与在线 Authority 双写。

## Open Questions

- Blocking：无。当前 Desktop Node 41 按用户已确认目标采用 `superadmin + authority-domain:1`，但该 Binding 必须由 Authority 显式创建且可撤销，不得成为产品内建特权。
- Deferred：是否需要显式 deny、角色继承、主体组、跨 Authority federation、有界委派和 Binding 过期自动续期。

## Documentation Impact

如果进入 `$m-plan` 并实施，至少需要评估并更新：

- `features/hub.md`：从“只修改精确三元组、无超级用户”调整为“默认无自动超级用户，支持显式持久角色绑定”。
- `requirements/auth-controlled-admission.md`：继续保持 admission profile 不自动授予业务权限，并补充初始化 Binding 的边界。
- `specs/node-tree-link-resource-architecture.md`：增加 owner scope 与当前 topology snapshot 的裁决规则。
- 新增或更新 Policy spec：Definition、Binding、selector、generation、迁移和委派校验。
- 如确认推荐选项，新增 decision 记录“作用域角色绑定而非产品特权”。

## Repository / Branch / Worktree Status

- Canonical repository：`D:/project/MyFlowHub3/repo/MyFlowHub`。
- Base branch：`master`，讨论起点 `70add9b`。
- Dedicated worktree：`D:/project/MyFlowHub3/worktrees/scoped-policy-authority`。
- Discussion branch：`refactor/scoped-policy-authority`。
- Main checkout 的既有未提交改动保持不动；本讨论文档只写入 dedicated worktree。
- Repository 无 remote；本讨论不包含 push、release 或 publication。

## `$m-plan` Handoff Criteria

进入 `$m-plan` 前需要：

1. 以 Desktop Node 41 的 `superadmin + authority-domain:1` 持久 Binding 作为当前目标，并保留后续较窄 `network-admin` 角色的扩展空间。
2. 将 Option C 视为目标模型，Option A 仅保留为临时运维方案，Option D 明确禁止。
3. 计划覆盖协议 schema、持久状态迁移、拓扑作用域 evaluator、generation 失效、管理 Resources/SDK、Desktop 调用方、现有精确 grant 兼容和 Hub + Desktop + Metrics 联调。
4. 计划包含 default-deny、未来新 Node/Resource 自动覆盖、reparent、revoke、越权 payload、损坏状态、迁移和重启持久化测试。
5. 任何为 Node 41 创建真实高权限 Binding 的动作单列为可审计、可回滚的执行项，不在讨论阶段直接修改运行状态。

## Related Stable / Historical Docs

- [Hub](../features/hub.md)
- [受控准入](../requirements/auth-controlled-admission.md)
- [节点树、链路与资源架构](../specs/node-tree-link-resource-architecture.md)
- [统一权威节点树与可插拔链路](../decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md)
- [Authority 管理必须沿节点树执行](../lessons/authority-local-admin-actions.md)
- [历史：Auth 默认角色分层](../change/2026-03-26_auth-default-role-hierarchy.md)
- [历史：Access Policy 角色编辑器](../change/2026-03-27_win-access-policy-role-dialog-refine.md)

## Related Changes

- 尚未实施；本记录仅作为讨论结果和下一阶段规划输入。

## Routed Plan

- [Active plan](../../plan.md)
