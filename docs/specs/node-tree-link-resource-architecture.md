# 节点树、链路与资源架构规范

## Status

- 状态：Accepted architecture baseline。
- 对应决策：[统一权威节点树与可插拔链路](../decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md)。
- 本文定义后续大范围重构必须保持的稳定边界，不定义具体 wire 编码或迁移任务。

## Scope

本文约束以下长期架构关系：

- 节点树与运行期 authority；
- 可插拔 Transport、连接和 LinkSession；
- 节点资源及其归属；
- Subscription、Variable、Stream 与 Command 的一级关系；
- 请求链、裁决点和控制链；
- 断连、重连与 reparent 的安全含义。

## Core Model

```text
Transport Driver
      ↓
Pipe / Connection
      ↓
Frame Codec + LinkSession
      ↓ authentication + role activation
Authenticated Parent-Child Edge
      ↓
Authoritative Node Tree
      ↓
Resources + Subscriptions + Commands
```

### Node

Node 是树中唯一的拓扑和 authority 主体。每个 Node 至少具有：

- 稳定节点身份；
- 零个或一个当前有效父节点；
- 零个或多个直接子节点；
- 本地拥有的资源集合；
- 当前直接链路、后代路由和订阅聚合状态。

根节点没有父节点。一个普通节点不得在同一 authority 域内同时接受多个父节点控制。

### Resource

资源由一个 Node 唯一拥有，稳定标识至少包含 `owner_node_id + local_resource_id`。资源不参与父子拓扑，不能独立选择父节点，也不能绕过所属 Node 获得另一套 authority。

基础资源形态为：

- `Variable`：具有当前值、版本或修订，可提供初始快照并持续发布变化；
- `Stream`：发布有序或带序列信息的事件，不要求存在当前值；
- `Command`：接受调用参数并产生确认、结果或错误，用于主动控制和无法由订阅表达的行为。

资源别名、标签和发现索引只是元数据或索引，不能改变资源归属和控制权。

### Subscription

Subscription 是订阅者与 Variable/Stream 之间的一级运行时关系。其稳定语义至少包含：

- 订阅者身份；
- 目标资源；
- 授权范围；
- 租约或有效期；
- 订阅状态与恢复标识；
- 创建时的拓扑或策略世代；
- 可选的过滤、采样和背压策略。

Variable 订阅默认可包含初始快照及后续变化；Stream 订阅默认只保证约定起点之后的事件。具体重放与可靠性等级由后续协议规范定义。

## Authority and Routing Invariants

1. 已认证的直接父子 LinkSession 是节点树的边，也是运行期权限边和直接路由边。
2. 父节点对其子树具有控制权；子节点只信任当前父节点下发的显式控制阶段消息。
3. `parent-origin` 只证明消息入口方向，不能单独证明消息已经完成裁决。
4. 请求阶段消息在目标不属于本地可控下游时继续向父节点上送。
5. 请求自然到达能够覆盖请求者与目标的 LCA 或其他明确可裁决节点。
6. 裁决点验证请求者身份、签名或令牌、权限与目标范围；通过后生成显式控制阶段消息。
7. 控制阶段消息向下逐级投递，子节点不重复执行同一权限裁决，但仍需校验消息结构、目标、会话、世代和业务输入。
8. 来自子连接的 `SourceID` 必须能够由该直连子节点或其后代路由关系证明；节点不得伪造同级或其他子树身份。
9. 路由不得因为未知目标把来自父节点的消息再次送回同一父节点。

## Subscription and Command Paths

### Establishing a Subscription

1. 订阅者创建包含目标资源和订阅参数的请求阶段消息。
2. 若目标资源位于订阅者直接可控的下游，可按父控子规则向下处理；否则请求沿父链上送。
3. LCA 或可裁决节点完成权限检查，并将已授权订阅作为控制阶段消息向资源 owner 下发。
4. 沿途节点可以维护按资源与下一跳聚合的兴趣状态，但聚合不得改变原始订阅者的授权范围和租约。
5. 资源事件按照建立的订阅状态投递；中间节点负责路由、聚合、背压和清理，不重新定义资源权限。

### Invoking a Command

1. Command 调用在未裁决时属于请求阶段。
2. 直接父控子方向可以进入控制阶段；跨子树调用沿父链上送到裁决点。
3. 裁决通过后，调用以显式控制阶段消息下发到 Command owner。
4. owner 仍必须执行输入校验、幂等、限流、超时和业务错误处理；父节点信任不等于跳过业务安全检查。

## Pluggable Link Contract

### Transport Driver

Transport Driver 负责具体承载差异，包括：

- endpoint 解析；
- listen、accept、dial 和 close；
- 平台权限、发现、配对或设备选择；
- MTU、分片、短写、报文边界和底层错误适配；
- 将承载收敛为统一 Pipe/Connection 所需的可靠性和顺序语义。

TCP、Bluetooth RFCOMM、QUIC、串口、USB、WebSocket 等可以分别实现 Driver。同一节点可以监听多种承载，同一棵节点树的不同边也可以使用不同承载。

### Pipe / Connection

Pipe/Connection 是 Core 的最小读写承载边界。当前已有 `IPipe/IConnection` 可作为实现基线。契约必须明确：

- 读、写和关闭的行为；
- 短写及完整写入责任；
- 取消、超时和断连错误；
- 并发写入的串行化或帧不交错保证；
- 资源释放和重复关闭行为。

不得在 Node、Subscription、Variable、Stream 或 Command 中根据具体 Transport 类型分支。

### LinkSession

LinkSession 是节点树实际依赖的统一会话语义。实现名称可以沿用现有工程约定，但必须覆盖：

- 统一帧编解码；
- 本地和对端会话标识；
- 对端身份与认证状态；
- parent/child 角色；
- 请求阶段与控制阶段区分；
- 发送队列、背压和关闭状态；
- 心跳、存活检测和断连通知；
- 可选的 MTU、带宽、延迟和成本等能力信息。

Transport 连接建立后，只有完成认证和角色激活的 LinkSession 才能注册为节点树边。

## Lifecycle

### Connect and Join

```text
Transport Connected
    → Framing Ready
    → Peer Authenticated
    → Parent/Child Role Confirmed
    → LinkSession Active
    → Tree Edge Registered
```

任何失败都必须显式关闭连接并撤销尚未完成的临时状态；不得把未认证连接加入路由或权限索引。

### Disconnect

- 直接链路断开后必须清理对应直连边和经该边学习的后代路由。
- 与该边绑定的订阅聚合、租约和 pending 请求必须进入明确的中断、过期或恢复状态。
- 子树失去上游时仍可维持内部自治，但不能继续接受旧父节点之外的上游控制。

### Reconnect and Transport Migration

- 对端父身份不变时，可以建立新 LinkSession 并执行受控恢复；是否恢复订阅由租约、会话连续性和世代共同决定。
- 不得仅凭相同 endpoint 或 NodeID 恢复安全状态，必须重新证明对端身份。
- 新会话按已签名且已确认的 topology epoch 原子替换旧边；旧 LinkSession 的关闭清理必须同时匹配对端身份和旧 epoch，不得撤销已经替换的新边。
- 每个 LinkSession 绑定的订阅状态独立清理，即使该会话已被同一 peer 的新会话替换，也不能把旧订阅遗留到租约自然到期。

### Reparent

- 父身份变化属于 reparent，而不是普通重连。
- reparent 必须增加拓扑世代，拒绝旧父会话的后续控制，并重建上游路由声明。
- 跨旧父边建立的订阅和未完成授权默认失效；任何迁移都必须重新裁决或具有明确的连续性证明。

## Security / Safety

- 父节点密钥和实现处于高信任级别，其失陷影响整个子树。
- 父控子不允许父节点伪造子节点身份向上发起请求。
- 控制阶段标记必须是会话内部受保护的语义，不能允许未经裁决的请求者自行声明。
- 资源和 Command owner 始终负责边界输入校验、限流、资源配额与错误返回。
- 无法提供双向通信、身份认证和断连检测的设备不能作为完整 Node；应由可信网关将其表示为本地资源。

## Performance Constraints

- Transport 特性不得迫使上层复制业务实现。
- 低 MTU 链路的分片与重组必须有明确内存上限。
- Subscription 必须具有背压、采样、聚合或丢弃策略，不能允许慢速链路形成无界队列。
- 控制消息应优先于大体积 Stream 数据，避免关键控制被队头阻塞；当前有界队列与 gap 语义见 [Subscription vNext](subscription-vnext.md)。

## Non-goals

- 当前不支持同一 authority 域内多个同时有效的父节点。
- 当前不建立独立资源树或独立权限树。
- 当前不要求所有单向、广播或无认证设备成为 Node。
- 本规范不决定新 wire envelope、资源 schema、订阅 QoS 等级或旧 SubProto 的迁移时间表。

## Compatibility and Documentation Preservation

- 原有 Server 权限规范继续作为树形自治语义依据；本规范不删除其历史内容。
- 历史 plan/change 用于说明演进和已验证实现，不再单独承担当前架构真相。
- 后续如果新实现取代现有 `IPipe/IConnection` 或 SubProto，必须更新本规范、相关 ADR 和 supersession 链，而不是删除旧文档以掩盖架构演进。
- 任何改变“唯一父节点、父控子、资源归属节点、传输不泄漏到业务层”四项约束的工作，都必须先新增取代本决策的 ADR。

## Related Requirements

- [原始重构诉求](../intake/2026-08-27_node-tree-subscription-command-redesign.md)
- [统一节点运行时需求](../requirements/unified-node-runtime.md)

## Related Decisions

- [统一权威节点树与可插拔链路](../decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md)
- [使用单一 Canonical Monorepo](../decisions/2026-08-27_single-canonical-monorepo.md)

## Related Existing Specs

- [Wire Protocol vNext](wire-protocol-vnext.md)
- [Resource Model vNext](resource-model-vnext.md)
- [Subscription vNext](subscription-vnext.md)
- [Command vNext](command-vnext.md)
- 旧 Server 权限/路由规范及 EmbeddedSDK C/MicroPython Transport Contract 按精确 commit 保留，来源见 [迁移清单](../../migration/sources.yaml)。

## Related Changes

- [Canonical Monorepo 与统一节点运行时第一阶段](../change/2026-08-27_canonical-monorepo-unified-node-runtime.md)
- [Core Pipe 抽象与多承载基础](../change/2026-03-12_transport-pipe-core.md)
- [Bluetooth RFCOMM Transport](../change/2026-03-12_bluetooth-rfcomm-transport.md)
- [Link Router Kernel 重构](../change/2026-03-13_link-router-kernel-major-refactor.md)
- 旧 Core QUIC Transport 实现按精确 commit 保留，来源见 [迁移清单](../../migration/sources.yaml)。
