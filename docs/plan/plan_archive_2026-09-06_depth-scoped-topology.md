# Plan - 带深度的拓扑查询与 Desktop 按展开加载

> 这是执行验收完成时的计划/清单快照，保留当时阶段与授权描述。随后用户调用 `$m-archive`；本地合入、清理及最终状态以[变更归档](../change/2026-09-06_depth-scoped-topology-discovery.md)为准。旧worktree路径仅作历史定位；本地证据已保留到工作区artifacts，文中链接已重新定位。


## Workflow Information

- 日期：2026-09-06。
- Project Root：D:/project/MyFlowHub3。
- Canonical Repo / Code Repos：D:/project/MyFlowHub3/repo/MyFlowHub，单一 canonical monorepo。
- Branch：feat/depth-scoped-topology。
- Base：master @ ab19d3913fc4be6789c4eb3013840cc3f61f709f。
- Active Worktree：D:/project/MyFlowHub3/worktrees/depth-scoped-topology。
- Active Docs Root：D:/project/MyFlowHub3/worktrees/depth-scoped-topology/docs；合入后对应 canonical repo/docs。
- Current Stage：$m-execute，七项实施及批准的验证已完成；提交、归档、合并、发布不在授权范围。
- Approval Status：用户已批准 DOC01、PROTO01、TREE01、MGMT01、SDK01、DESK01、QA01，并显式调用 $m-execute；Blocked: no。
- Source：本任务的连续 $m-discuss，以及用户显式调用 $m-plan 后要求根据参考任务「全面审视项目现状」的清理结果重新规划。
- Referenced Task：local / 01a06ff5-b616-7f73-a66b-4141c07cb509；已用 read_thread 读取，结合当前 Git 与稳定文档确认实际状态。
- Original discussion record：[本轮 intake](../intake/2026-09-06_depth-scoped-topology-discovery.md)。

## Initialization And New Baseline

1. 已读取 guide.md 的工程规则：中文提交信息、canonical docs、项目级 sibling worktrees；验证使用 GOWORK=off，PowerShell 使用无 profile 方式。环境配置与连接秘密不复制到计划。
2. 专用工作树最初基于 476c6ea。用户补充清理背景后，在工作树无修改、无本轮提交时 fast-forward 到 ab19d39；这是规划分支更新，不是在主检出实施功能。
3. 最新基础已移除 Flow、Android/C/ESP32/MicroPython、移动专用 bindings、旧 owning SDK/bindings 生命周期；Clipboard 已迁到 NodeHost。保留 Windows Desktop/Metrics/Clipboard、资源操作、订阅、EnrollmentBootstrap、Legacy Join、通用网络和链路。
4. 旧 root plan/todo 属于已经归档的 scoped-policy workflow。只在本专用工作树替换控制文件；历史证据保留在 docs/plan/plan_archive_2026-09-01_scoped-policy-authority.md。Flow/Android/Embedded 重新设计待办通过本计划和 todo 链接继续保留。
5. 主检出原有 8 个 tracked 修改及未跟踪产品文档、设计稿不复制、不暂存、不覆盖。尤其 Desktop requirement 中未提交的 AUTHZ02 表述，执行和合并时必须按本轮已确认语义与原稿做语义合并。
6. remove-legacy-flow、retire-mobile-embedded-bindings 是其他任务工作树，不使用或清理。最终归档前重新检查 master 演进和冲突；不从旧工作树恢复已退役实现。
7. 规划阶段只创建 root plan/todo、intake 及其索引。稳定需求/spec/ADR/feature 在 DOC01 实施、验收通过后更新为 Current。

## Discussion Summary

### Confirmed Goals

- 核心模型只有节点、网络、资源；设备和插件不成为网络对象。
- 查询起点为指定 Node；Desktop 默认以直接父节点为起点。
- depth 统一为非负整数：0 为不限后代深度，1 为直接孩子，n 为向下最多 n 层；响应包含起点节点。可省略参数的 UI/API 入口默认 1，Go typed helper 要求调用方显式传整数。
- 整树 = 已确认网络根作为起点 + depth 0；深度不扩大权限或自动发现祖先。
- Desktop 初始读取一层；首次展开已知节点再请求其直接孩子；不猜测 Node ID、不逐个探测未知节点。
- 父节点维护自身子树；查询读取维护好的状态/快照，不在每次请求时逐节点采集。
- 树是导航，目录读取与资源操作独立鉴权；已保存 Resource reference 不以成功枚举拓扑为访问前提。
- 本轮要准确表达未加载、已加载为空、失败、撤权、过期和恢复，保留资源工作区与 View 的已有行为。

### Planning Baseline Facts

- apps/desktop/frontend/src/App.tsx 的 refreshPlatform 读取父节点完整 topology，再 Promise.allSettled 请求全部 owner 的 catalog，最后才 setTopology。
- Explorer.tsx 的 setExpanded 只更新 UI preference；store.ts 假定传入范围已经完整。空 catalog 和失败 catalog 均可能表现为“此节点没有资源”。
- runtime/tree.State 已维护父连接、直接孩子和所有已通告后代关系；变动增加 membershipEpoch。Tree.Epoch 是连接/拓扑协议世代，不能单独用作后代内容缓存版本。
- feature/management/controller.go 把关系序列化为 system/topology Variable；host/hub 默认约每秒刷新。Variable read 返回已有字节快照。
- system/topology 目前没有 depth 输入；查询能力在挂载 management 的 Node 上提供。并非所有普通 NodeHost 都自动挂载完整 management。
- runtime/auth 按 Subject + ResourceID + CapabilityID 授权，不解析 depth；新基础仍保留此模型。
- 当前 SDK/bindings 仅为 attached facade；其 Close 不得终止 NodeHost。架构测试已禁止恢复 owning API。

## Requirements Analysis

### User-visible Scenarios

1. 登录后先显示直接父节点和它的一层孩子。一个 catalog 超时不阻塞树；未选中的节点不自动批量加载目录。
2. 点击或键盘展开 N，使用 N 的 topology 查询直接孩子；折叠后再展开复用未过期缓存，同一查询只保持一个在途请求。
3. 在明确选择的起点执行“加载完整子树”，调用 depth 0。服务端拒绝后显示 Forbidden，不降级、更换起点或扩大查询范围。
4. 选择 N 读取 N/system/catalog；View 打开时按其 Resource references 读取所需 owner catalogs，独立于导航展开状态。
5. 节点退出、reparent、重连或切换 Profile 后，旧结果不得复活已移除关系、覆盖新 Profile、清空 View 布局或未保存业务 draft。
6. 搜索入口明确显示“搜索已加载节点”；过滤仅作用于已加载索引，不因搜索暗中遍历整树。所有已加载当前 scope 的祖先路径可搜索/展开。
7. 遇到未提供深度查询的 Node，显示“不支持此查询”及可重试/升级提示；不会把 unsupported 当叶子，也不自动用完整快照替代浅层查询。

### Functional And Non-functional Requirements

- 起点由 Resource owner 确定；不增加第二个任意 root_node_id 请求字段，避免“向 P 的资源授权却查询 N”造成新的 payload-scoped 权限语义。
- 返回根是查询边界，不等于网络根。N 查询响应根节点省略 parent_id 时，Desktop 合并时保留已知外层父关系。
- 每个响应是所请求深度内完整结果；达到协议数量/字节上限明确 Overflow，不静默截断后标为完整。
- depth 0 只取消深度限制，不取消节点数、payload、CPU、deadline 或权限边界。
- 新请求 wire 中 depth 必须显式存在，0 不能因 omitempty 丢失，也不能把缺失或 null 当全量；UI 可选参数入口在调用边界应用默认 1，Go typed helper 保留显式值。
- 请求/响应使用 generated schema、严格 JSON 和正整数 Node ID 字符串；负数、文本、浮点数、布尔值、不安全数字与未知字段明确拒绝。
- 无新依赖，无第二套 runtime/路由/权限状态；所有远端访问使用现有 NodeHost attached Client 的 generic resource operation。
- 前端保留独立 Node/Resource 分区、键盘 tree 行为、收起/分隔条、Inspector/View 区分、拖拽与持久化版本。
- 缓存是当前 Profile/连接的内存数据；不写入 View、credential 或 UI preferences。已有展开 preference 可作为有界恢复意图，不可证明缓存已加载或仍有权限。

## Architecture Design

### Current / Proposed / Alternatives

| 方案 | 传输与计算 | 权限 | 维护成本 | 结论 |
| --- | --- | --- | --- | --- |
| 当前无参数完整快照 | 每次传完整 owner 子树；客户端全量 catalog | read/subscribe 都暴露完整快照 | 已有路径 | 保留当前受支持的快照契约；Desktop 导航改用新查询 |
| 先全量拉取、前端裁剪 depth | 仍传完整树，客户端假装按需 | 浅层调用先获得深层数据 | 改动少但违反需求 | 拒绝 |
| 单 query capability + 通用 payload 深度策略 | 可以限制传输 | 需扩展 policy evaluator、委托与审计语义 | 将本轮变成权限引擎重构 | 延期，不采用 |
| 同一 Resource 新增浅层/递归 capability，SDK 统一 depth 方法 | 服务端缓存并按深度返回 | 现有 exact/scoped policy 可区分授权 | 局部 protocol/management/SDK/Desktop 变更 | 本轮推荐 |
| 新建 Command/query endpoint 或另一棵导航权威树 | 可实现查询 | 引入新的资源身份或权威关系 | 与现有对象行为归属冲突 | 拒绝 |

### Approved Query Contract

Resource 继续为 (N, system/topology)，owner N 即本次查询起点。以下名称与语义已经批准并实现，当前稳定合同见 docs/specs/topology-discovery.md。

- 保留现有 read / subscribe、mfh.management.topology.v1、SDK ManagementClient.Topology(ctx) 和 Wails TopologyJSON(owner) 行为。
- 新增 children capability：仅接受 depth 1。
- 新增 subtree capability：接受 depth 0..protocol.MaxItems，覆盖有限深度和完整子树。SDK depth 1 选择 children，其余选择 subtree。
- children 专属输入 schema：mfh.management.topology-children-request.v1（version 1，必填 depth 且恒为 1）。
- subtree 输入 schema：mfh.management.topology-query-request.v1（version 1，必填 depth，整数范围 0..4096）。
- 统一输出 schema：mfh.management.topology-query.v1。
- 请求没有独立 target/root 字段；owner 就是 root。不能通过 generic operation 在 children capability 上塞 depth 0/2。
- 响应包含 version、root_node_id、depth、instance_id、revision、nodes。
- instance_id 为 provider 生命周期内稳定的随机 128-bit 标识字符串；revision 为 1..2^53-1 的单调安全整数。跨 owner / instance 不比较 revision 大小。
- nodes 延续 NodeID 字符串、parent_id、display_name、role、generation，并增加 has_children 布尔提示；该提示只表示快照中的后代存在性，不是访问授权。
- 根节点必为 resource owner，只有该根可无响应内 parent；其余节点 parent 必须存在于结果、关系无环，所有距离不超过 depth（0 除外）。响应不包含查询根以上的祖先。
- 内层节点的孩子列表在响应深度内完整；边界节点 has_children=true 表示尚有未包含的下一层，不能被当空叶子。
- instance_id/revision 描述 provider 的快照，不是全网同步世代。新 DTO 不复用 Tree.Epoch 来断言内容不变。
- protocol.Validate 对新 schema 的验证须为有界线性关系检查；不为此重写所有历史 schema validator。

### Permission Semantics

- children / subtree 是现有 topology Resource 的 capability，不是设备、插件、第二棵树或额外权限模型。
- Subject 必须获得 owner 范围内相应 capability；目录 read、资源操作权限不随树查询赠送。
- 只授 children 不授 subtree/read/subscribe，可允许逐层导航并拒绝一次递归查询。
- 授 subtree 不隐含授 children；推荐的浏览角色应按需要显式列出二者，现有 all/superadmin 按已有规则覆盖新能力。
- 旧 read/subscribe 仍返回完整快照，属于既有广权限；不能保留这两项却声称该 Subject 只能看一层。不自动扩大或回写现有 Grants/Bindings。
- 两层以上暂统一由 subtree 授权，不引入按任意最大 depth 配置的 policy。授权范围是 owner/capability，depth 决定返回量。
- Root/children(depth 1) 不等于 Root/subtree(depth 0)。完整全网查询须针对可信已知网络根并获对应权限；Desktop 不自动向上爬树定位根。
- 对每一层均获 children 权限的 Subject，可以逐层构造更大的树视图；本轮不声称一次性查询限制能够阻止合法信息汇总。
- handler 始终校验 capability 与 payload 一致；缓存命中仍走每次请求的 canonical auth 路径，不缓存“允许”结果。
- 继续遵守现有父控子与 Subject 传递规则，不在 owner 用空的本地 policy 重做一次不等价的上级裁决。

### Runtime Snapshot And Provider

- 在 runtime/tree 新增窄的原子只读快照接口：在同一 read lock 下取得 local、parent、直接/后代父关系、membershipEpoch、现有 epoch。不得逐个无锁读取后拼装不同世代数据。
- 快照接口支持未变化时只返回版本标记，避免每个 refresh tick 复制全部关系；深度索引/JSON 位于 management，runtime/tree 不依赖 protocol JSON DTO 或产品代码。
- management 从已通告关系构建稳定排序的 nodes/children/depth 索引；保留现有刷新周期，不增加拓扑事件总线。版本未变化且影响输出的本地 settings 未变化时复用缓存。
- 高频 depth 1 和 depth 0 在刷新阶段预生成；其他正深度从同一不可变快照筛选，按 (instance_id, revision, depth) 做最多 8 项、有总字节上限的缓存。不为每个后代同时存一份完整子树，避免重复存储平方增长。
- 完整树过大只让相应结果为 Overflow；仍可返回符合限制的浅层结果。错误不能被伪装成空树或旧的完整结果；legacy read/subscribe 的正常大小内容保持兼容，超限状态需显式可诊断。
- 查询不向后代发请求、不重新排序整棵树；缓存命中主要保留验证、复制、传输成本。其他深度首次查询允许 O(本次输出规模) 筛选/编码，不能承诺任意大小响应零计算。
- 原 system/topology Variable 保持 read/subscribe 和 Observable 生命周期。通过 management 内组合 wrapper 增加查询 capability，委托原 Variable 的 read/Observe；无需增加 runtime type switch 或新的 Command Resource。
- controller Refresh 是唯一发布者；查询持有不可变快照。不得在全局 tree 写锁/刷新锁内发送网络、通知 watcher 或等待调用方。
- descendant announce/withdraw/reparent、父连接改变、本地显示名改变都必须导致受影响快照失效；新 instance 允许 revision 重启，不与旧 instance 混用。
- 本轮 provider 装配仍由现有 management.Register 提供。对没有该 Resource/capability 的中继，Desktop 表达 unsupported；叶子可依据已授权响应的 has_children=false 不再查询下一层。通用 NodeHost 自动挂载拓扑 provider 是独立后续设计。

### SDK And Binding Boundaries On ab19d39

- Go SDK 在 ManagementClient 增加 QueryTopology(ctx, depth) typed helper，使用已存在的 attached Client.OperatePayload。
- DesktopApi.topology(ownerNodeID, depth=1) 在前端映射 capability，使用现有 App.OperateJSON 和 operation envelope 解码，不新增另一个连接/生命周期 facade。
- 根目标与 depth、响应 schema/owner/深度/版本在 Go、协议 handler、前端各自边界校验。必要的输入验证不得依赖 UI 控件。
- 不恢复 NewClient、StartTCP、StartEnrolledTCP、OpenEnrollment 等已删除 owning API。EnrollmentBootstrap 与 NodeHost 生命周期不参与改造。
- 不给 portable/Desktop bindings 增加重复的 topology facade 方法；contract manifest 只更新 Resource capabilities/schemas，methods/desktop_methods 必须继续与实际 attached APIs 一致。
- 使用现有生成工具更新 protocol builtin schema、contracts.json、Desktop builtin-schemas.json。Wails module 仅由 canonical 工具校验，预计无需新增 Go App export。
- 老客户端仍可读取当前受支持的旧 snapshot；新 Desktop 在旧 provider 返回 NotFound/Unsupported 时明确失败，不自动把浅层请求变成全量读取。用户仍可通过通用资源面板显式使用已授权 legacy read。

### Desktop Cache And Interaction

- 新建专用 discovery state/controller，把节点加载和 catalog 加载从 App.tsx 的单个 refreshPlatform 全量循环拆出；React 展示只消费单一派生索引。
- key 至少含 Profile ID + connection generation + 浏览起点。每个 owner 分别保存 topology response instance/revision、请求序号与 children 状态。
- children 状态：unloaded / loading / loaded / error，另记录 stale、错误码、已成功 children IDs 和更新时间；loaded+空数组才表示确认无孩子。错误不能清空成“没有孩子”。
- 初始 query(parent,1) 成功就呈现树；首次展开 N 请求 query(N,1)。同 key 去重，最多 4 个并发 discovery/catalog 请求，显式刷新可替代旧请求。
- 折叠保留缓存。再次展开在 30 秒软有效期内复用，超期显示旧结果为 stale 并刷新；这只是 UI 缓存期限，不是授权租约或网络实时性保证。已展开树由显式刷新更新，本轮不增加后台全树轮询。
- 已保存展开 preference 按已知路径逐层恢复，仅恢复实际展开分支，使用同一去重/并发预算；未加载到的 ID 不直接扫描。
- depth n 响应合并时，仅替换已完整覆盖节点的直接孩子集合。边界节点不删除尚未覆盖的缓存孩子；被权威父列表移除的分支立即从导航脱离并使在途请求失效。
- root parent_id 缺失只表示响应边界，不抹去外部父关系。来自不同 owner 的 revision 不比较；连接/request generation 决定异步结果能否提交。
- 发生 Profile 切换、断连/重连、浏览起点变化时清除或失效旧 discovery；既有 View 布局与业务 draft 仍由现有保存/退出流程管理。
- has_children 控制可展开提示；加载未完成显示 busy，错误行可重试。鼠标与 Right/Left/* 键走同一加载逻辑，避免键盘绕过请求调度；不误报未知节点为 leaf。
- Node 搜索标明已加载范围；只有显式“加载完整子树”动作请求 depth 0，失败保持原范围并显示错误。浏览起点可明确指定，默认始终为直接父节点，不推测网络根。
- catalog 以 owner 独立缓存，选中节点和当前 View 的 Resource references 按需触发。目录状态区分未加载、加载中、空、Forbidden、不可达、unsupported、其他失败。
- Resource 菜单的 capability 支持与实际权限仍独立；不实现 AUTHZ02 的 effective discovery。
- 已保存 View references 即使不在当前树中，也能直接经合法资源寻址加载；catalog 未加载不能先渲染为永久 detached。明确目录成功却找不到引用后才进入对应缺失状态。
- 缓存有硬边界：每个活动连接最多 4096 个导航节点、128 个 catalog owner；仅对未使用的 catalog 做 LRU 回收，当前选择/活动 View owner 固定保留。达到边界明确提示，不能删除 View 或静默截断树。
- 缓存拓扑与资源值不落盘；无新增 View major version。单独的 UI request generation 不能写进网络协议或权威树。

## Docs Governance Routing

使用 $m-docs；目录已完整，无需 bootstrap。新结构与当前 spec 不一致是用户明确授权的演进，不是阻塞：更新相关条款并保留历史证据。

| 分类 | 影响 | 目标与执行时机 |
| --- | --- | --- |
| Intake | add | 本阶段新增 docs/intake/2026-09-06_depth-scoped-topology-discovery.md 及分类索引 |
| Requirements | clarify / add | DOC01 更新 desktop-resource-workspace.md 的发现/搜索/状态要求；新增 requirements/topology-discovery.md 记录范围/深度/权限/新基础 |
| Specs | add / clarify | DOC01 新增 specs/topology-discovery.md；更新 desktop-resource-workspace-v3.md 的 lazy-loading 延期条款，v2 标明对应发现部分由新 spec 细化；更新 protocol_map.md；仅窄修 resource-platform-v2.md 中 Node=设备的错误等同表述 |
| Features | clarify | DOC01 标记计划中变化，QA01 通过后更新 desktop.md / hub.md 当前行为；不宣称已退役平台支持新能力 |
| Decisions | add | DOC01 新增 2026-09-06_depth-scoped-topology-capabilities.md，记录同资源 capability、数值 depth、原子快照、部分树缓存及代价 |
| Lessons | none now | 阅读已有 generation/JSON-safe revision/generated-binding lessons；若实施发现可复用问题，再在 ARC01 判断 |
| Plan / change | root control only | 本阶段写 plan.md / todo.md；docs/plan 与 docs/change 归档属于 ARC01，不在此提前创建 |

相关入口：
- [Desktop requirement](../requirements/desktop-resource-workspace.md)、[Resource 平台需求](../requirements/extensible-resource-platform.md)。
- [当前 Desktop](../features/desktop.md)、[Hub](../features/hub.md)。
- [Resource v2](../specs/resource-platform-v2.md)、[Collection/capability](../specs/resource-collections-and-actions.md)、[权限](../specs/scoped-policy-authorization.md)。
- [树与链路](../specs/node-tree-link-resource-architecture.md)、[NodeHost](../specs/node-host-runtime.md)、[构建门禁](../specs/build-and-ci.md)。
- [已退役平台与 SDK 决策](../decisions/2026-09-06_retire-mobile-embedded-and-owning-bindings.md)、[退役需求](../requirements/mobile-embedded-redesign.md)、[Flow 待办](../requirements/flow-redesign.md)。
- [JSON / UI 状态经验](../lessons/collection-browser-cross-runtime-contract-boundaries.md)、[Generation 清理](../lessons/session-replacement-generation-cleanup.md)、[绑定生成](../lessons/frontend-worktree-wailsjs-missing.md)。

## Execution Scope After Approval

### Will Execute

| Task ID | Title | Files / Modules | Acceptance / Tests | Risk / Notes |
| --- | --- | --- | --- | --- |
| DOC01 | 固化发现契约与产品范围 | docs/requirements、specs、decisions、features 及索引 | 新旧范围、数值 depth、新基础、授权和搜索语义可追溯 | 不覆盖主检出未提交产品稿；不提前宣称实现 |
| PROTO01 | 深度查询 schema 与能力描述 | protocol/schema_management.go、新 topology schema、data_schema.go、相关 tests | 0/1/n、缺失/负数/非整数、边界关系和 limits；生成 schema 一致 | children 恒深度 1；revision 安全数 |
| TREE01 | 原子拓扑快照与版本读取 | runtime/tree/tree.go 及 tests | 并发 attach/announce/withdraw/reparent 保持单世代；未变不复制 | 不改变路由/权限树，不把 link epoch 当内容版本 |
| MGMT01 | 同资源查询 capability 与有界缓存 | feature/management/controller.go、新 topology.go/tests | read/subscribe 保留；浅/深权限分离；缓存失效/超限/并发 | wrapper 保留 Observable；不重建 legacy owning API |
| SDK01 | attached SDK 与生成契约接入 | sdk/go/features.go/tests、sdk/bindings/contract、generated、Desktop generated schema | typed 查询、显式 0 保留、错误保真、生成/API 表面一致 | 不新增 portable/Desktop 生命周期或重复 facade |
| DESK01 | 按展开加载及目录状态 | frontend api/types、discovery controller/store、App、Explorer、相关 tests | 初始/展开/重试/搜索范围/Profile 隔离/View 按引用恢复 | 部分树不能冒充全树，旧请求不能覆盖新状态 |
| QA01 | 协议到 Windows 产品验收 | tests/integration、受影响模块测试、构建证据 | 深度、授权、缓存、UI/键盘、现有 Windows 产品回归 | 无真实业务节点/授权数据变更，不恢复移动门禁 |

### Will Not Execute Now

| Task ID | Title | Reason |
| --- | --- | --- |
| LIVE02 | 拓扑变化推送与全面事件驱动刷新 | 当前定时快照 + 版本复用足够；实时一致性/订阅恢复另立契约 |
| SEARCH02 | 未加载范围的服务端搜索、分页与超大树流式传输 | 本轮只有有界 depth 查询；本地搜索范围明确 |
| AUTHZ02 | 通用 member/深度条件策略与 effective-capability discovery | 复用现有 capability 授权，不重构权限 evaluator |
| HOST02 | 所有普通 NodeHost 自动挂载拓扑 provider、Hub 宿主迁移 | 本轮只增强现有 management provider，避免新生命周期装配范围 |
| ANDROID-REDESIGN | Android 重启与新移动实现 | 已退役，见 mobile-embedded-redesign.md，不是当前验证/兼容负担 |
| EMBEDDED-REDESIGN | C/ESP32/MicroPython 或其他受限实现重启 | 已退役，后续先定义约束再选实现 |
| FLOW-REDESIGN | 自动化模型、语言与执行器重做 | 已退役，本轮不讨论或恢复 |
| ARC01 | 提交、文档归档、本地合入、工作树清理 | 实现/验证后显式进入 $m-archive；现在不执行 |
| PUB01 | push/release/sign/publish/deploy | 独立授权范围，不包含在计划批准内 |

旧 scoped-policy workflow 的 DENY02/DELEG02/FED02/CLEAN02 等条目留在其历史计划，本轮不复制成待执行任务；AUTHZ02 与上述平台重启项目因直接约束本轮而明确列出。每个本轮 Task ID 只属于一个执行范围。

## Task Details

所有任务最终 Owner 为当前主 Agent；工作路径为本专用 worktree。批准后划分 PROTO01、TREE01、SDK01、DESK01 与 QA01 集成/Windows 回归写集；按固定合同并行实施，集成验收仍遵循依赖顺序。主 Agent 负责 DOC01、MGMT01、生成文件和最终验收。

### DOC01
- Goal：把已确认产品语义写成稳定文档，标记 Proposed/Planned 与 Current 的区别。
- Write set：上述 docs routing 行中的 requirement/spec/ADR/feature、对应分类索引，必要时本 intake 的状态。
- Tests：相对链接、范围/任务对应、退役状态与代码基线一致；不修无关历史链接。
- Rollback：只回退本轮文档差异；历史 archive 和用户草稿保留。
- Dependency：无。

### PROTO01
- Goal：声明请求/输出、capability constants 和有界验证，不改 envelope 或 ResourceID。
- Write set：protocol/schema_topology.go/tests（新增），schema_management.go、data_schema.go/tests 中直接引用。
- Tests：children 的 depth=0/2 被拒绝；subtree 的 0/1/2/4096；缺省/null/负数/分数/字符串/过大值；重复 ID、环、外部 parent、错 root、深度越界、has_children 边界、payload limits、JS-safe revision/instance。
- Rollback：与消费者按依赖逆序回退；不迁移用户持久数据。
- Dependency：DOC01。

### TREE01
- Goal：原子取得运行时已维护关系及版本，供 provider 派生；优化不变时的读取成本。
- Write set：runtime/tree/tree.go、tree_test.go，可新增 topology_snapshot.go/tests。
- Tests：孩子与后代进入/退出、同父重连、reparent、快照不可变、并发读取、版本变化与未变路径；运行 race。
- Rollback：移除新读取接口，回到旧 Relations 读取；树状态与 routing 行为不能改变。
- Dependency：DOC01；与 PROTO01 使用独立写集并行执行。

### MGMT01
- Goal：在已有 topology Resource 增加 children/subtree，维护一致快照和有界缓存。
- Write set：feature/management/controller.go、topology.go/tests、controller_test.go；host/hub 仅在刷新错误暴露或 provider 装配测试直接需要时修改。
- 必要直接后果：runtime/command/dispatcher.go/tests 将现有 protocol.ErrInvalidPayload/ErrPayloadTooLarge 映射为 Malformed/Overflow；runtime/node/operations.go 的订阅错误映射补充同一 Overflow。否则正确的查询失败会被 wire 错报为 Internal/Conflict，不新增资源类型或权限策略。
- 订阅复核直接后果：runtime/resource.Observation 增加可选 Failure，runtime/subscription 将其作为终止事件，经既有 wire Error 返回。用于让已建立的旧全量订阅在超限时明确失败；不增加 wire schema、资源类型或权限含义。management 使用单一发布排空器，回调重入只更新待发布快照，避免观察者收到倒序版本。
- Tests：legacy snapshot/subscribe；正常 child 和 subtree 响应；直接 generic operation 伪造 depth；仅浅授权的深读拒绝；root/relay 各范围；缓存命中无需重建；本地名称/关系变化失效；重启 instance；满缓存/超大结果；并发 Refresh/query/Observe。
- Existing test adjustment：当前 helper 强制把 topology cast 为 *resource.Variable，应改为经公开 Resource/SDK 契约验证，不能通过保留特殊后门绕过 wrapper。
- Rollback：卸下新增 capability wrapper 并保留原 Variable；新 grants 可成为无效能力引用但不会扩大权限，不自动删除用户 grants。
- Dependency：PROTO01 + TREE01。

### SDK01
- Goal：新增 typed helper；Desktop 继续经现有 generic operation；生成契约只含当前 attached APIs。
- Write set：sdk/go/features.go/tests；sdk/bindings/contract/contract.go/tests 的 topology Resource 描述；生成 contracts.json/builtin-schemas.json，必要的 schema fixture。
- Tests：真实 attached Node/Host fixture；depth 选择能力、0 保留、默认 1、schema/owner/错误；api_contract_test 防止已删除 owning 方法重现；facade 关闭不停止 Host。
- Rollback：删除新 helper 与新增 descriptor/schema 记录；原 snapshot helper 与当前 attached lifecycle 不变。
- Dependency：PROTO01 + MGMT01。

### DESK01
- Goal：把完整树假设替换为有加载状态的部分树；目录按使用加载。
- Write set：apps/desktop/frontend/src/api.ts、types.ts、App.tsx、store.ts、components/Explorer.tsx；新增 discovery 模型/controller/hooks 与集中测试；必要的 scoped 状态样式；View/Renderer 只改 descriptor loading 的直接后果。canonical build 更新已跟踪 frontend/dist/index.html 和对应 bundle；仅换行变化的品牌资产已恢复。
- Tests：默认父节点一层；深层展开只发一条已知 Node 查询；重复/双击/*键并发去重；错误与空态；软过期/手动刷新；scope root 合并；Profile/reconnect/重试竞态；节点退出/reparent；已加载范围搜索；当前 View 独立 catalog 加载、布局/draft 保留；unsupported 不全量 fallback。
- Rollback：回到旧全量工作区控制路径；UIPreferences/View 格式保持可读，不删除用户数据。
- Dependency：SDK01；接通 provider 合约后再改 UI，避免生产中 mock fallback。

### QA01
- Goal：以当前支持的产品范围证明端到端行为，不使用旧 Android/Embedded/Flow 门禁。
- Write set：tests/integration/topology_discovery_test.go（新增或同等位置）、受影响模块的必要回归；只产生隔离证据，不改 CI 来掩盖失败。
- Integration：Memory 与 TCP root→relay→多层节点；直父子树默认；root 整树；仅 children grant 拒绝 subtree/read/subscribe；catalog grant 独立；旧 subject/grant 不隐式迁移。
- UI：至少六层树，正常/拒绝/慢响应节点；仅展开路径请求；折叠复用；切换 Profile 后延迟响应丢弃；已保存 View 可恢复；mouse/keyboard 等价。
- Validation commands：执行时统一 GOWORK=off。先 focused Go tests/race 与 Desktop Vitest；随后 go test ./... -count=1、go vet ./...、go build ./...；生成工具与 Windows Desktop Wails build；Metrics/Clipboard 的当前 Windows 回归使用 scripts/mfh.ps1 支持的 targets。
- Generation：go generate ./sdk/bindings，scripts/mfh.ps1 -Action generate -Target generated；api manifest tests 与二次生成哈希一致。check/generated 与 HEAD 的预期新增差异须在候选已记录基线后核验，不能把有意生成变更误当随机漂移。
- Windows products：Desktop/Metrics 前端 test/build 与 Wails；Clipboard Go/Flutter analyze/widget，必要的 Windows build 依照当前 CI 范围。无需 APK/AAR/ESP-IDF/C/MicroPython 构建，且不得恢复被删除 targets。
- GUI smoke：只使用隔离临时状态与本机测试拓扑，不连接/改写真实 Node 41、Hub 或 Metrics 权限/数据；记录当前构建资产和二进制。工具/环境缺失如实标为 Unavailable，不能用 mock 截图宣称真实联调通过。
- Performance evidence：构造同一树，比较新 depth=1 与 legacy 全量的返回节点数/字节数，记录缓存命中/未命中基准；不使用不稳定毫秒阈值当单测断言。
- Rollback：仅清理本轮隔离测试资源与证据；不删除用户状态；报告不可用/失败门禁。
- Dependency：所有实现任务完成。

## Dependencies, Risks And Rollback

顺序：DOC01 → PROTO01/TREE01 → MGMT01 → SDK01 → DESK01 → QA01。执行阶段依据 m-execute 的并行评估规则，将 PROTO01 与 TREE01 分为互不重叠的子 Agent 写集；主 Agent 保留整合、生成文件和最终验收责任。其余依赖保持不变。

主要风险：
- 仅给前端增加 depth 无法形成权限边界；双 capability + handler 参数校验是本方案的关键约束。
- 旧广 read/subscribe 权限仍可读全量，必须在角色说明中明确，不能自动迁移真实授权。
- 每个 Node 不是天然提供 management；unsupported 是明确兼容边界，不能冒充空孩子或自动扩大查询。
- 多 owner 返回部分树，revision 不可全局比较；必须用连接/请求 generation 和查询边界合并。
- 缓存可能在 tick/网络传播期间滞后；本轮承诺明确刷新与有界复用，不承诺强一致或全网实时。
- NodeHost/SDK 新生命周期是既定基线，生成操作不能复活旧 owning API 或已退役平台。
- 有限范围协议查询不等于任意规模分页能力；超过限制明确失败。
- main 的未提交文档与独立工作树仍可能继续变化，合入时需语义合并，不覆盖。

整体回滚点为 ab19d3913fc4be6789c4eb3013840cc3f61f709f。按 DESK→SDK→MGMT→TREE/PROTO→DOC 的反向依赖回退本轮差异；不得回退至平台退役之前。无持久 schema/credential/View 迁移，无 live policy mutation。

## Plan Verification And Handoff

- Planning checks：基线/worktree 路径、实际 API/模块、退役范围、Task ID 唯一归属、文档链接及 diff 范围。
- 本轮规划核验：plan/todo/intake 相对链接均存在，16 个 Task ID 在 plan/todo 一致且执行范围不重复；写集仅四个规划文件。产品验证未运行。
- 规划阶段未运行产品 tests/build；下方执行记录均来自本轮已批准的工作，不使用历史报告替代。
- 技术阻塞：无。需要批准的架构选择已完整记录在本计划，而非在执行时静默切换。
- 执行审批：已取得，批准范围为 DOC01、PROTO01、TREE01、MGMT01、SDK01、DESK01、QA01；含计划中所列验证。
- 若不接受双 capability 或“不支持时不自动全量降级”，应在计划阶段修改契约，再批准；不得执行后补问。

## Execution Result — 2026-09-06

DOC01、PROTO01、TREE01、MGMT01、SDK01、DESK01、QA01 均已实现、复核并通过批准范围内验证。稳定 requirement/spec/ADR 和 Desktop/Hub 文档已同步。原始证据与产物索引见 [QA 报告](../../../../artifacts/2026-09-06_depth-scoped-topology/evidence/REPORT.md)，逐项状态见 [todo](todo_archive_2026-09-06_depth-scoped-topology.md)。

- Go：全量 test / vet / build 通过；tree 及七个受影响包的 race、Memory/TCP 六层拓扑和权限隔离通过。
- Desktop：22 个 Vitest 文件、224 项测试通过，TypeScript/Vite 与最终 Windows Wails build 通过。真实 Wails 开发桥 + 隔离 TCP 网络验证默认一层、鼠标/键盘、拒绝/不支持、完整子树、已加载搜索和 View 独立恢复；最终构建资产再次验证紧凑入口与名称保持。
- Metrics / Clipboard：当前 canonical test/build 全部通过；订阅终止补丁后已重建 Metrics Wails 与 Clipboard Go bridge。Flutter analyze/widget/Web/Windows 通过。
- 生成：12 个文件二次生成哈希一致，最终 Wails build 后仍一致；依赖、owning SDK 和已退役平台未回流。
- 服务端复核补齐有序重入发布与既有订阅 Overflow 终止；前端复核补齐排队失效清理、Profile/View 引用隔离和迟到保存保护。所有修复都属于批准验收的直接后果。
- 仅恢复 9 个经 Git 确认无语义差异的构建换行改动。隔离 GUI 测试进程与页面已关闭，fixture 收到 STOP 正常退出。

没有未完成的批准验证或技术阻塞。未执行提交、归档、合并、推送、部署或工作树清理。回滚遵循上方反向依赖，仅撤销本轮差异，无持久数据或真实权限迁移。
