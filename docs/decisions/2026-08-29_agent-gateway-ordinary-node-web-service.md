# 2026-08-29 Agent Gateway 采用普通节点、多客户端令牌的独立 Web 服务

## Status

Accepted as product architecture baseline；implementation deferred。待整体系统接口与权限模型趋于稳定后，再以独立 plan/spec 进入实现。

## Context

Codex 等外部 Agent 需要操作 MyFlowHub 网络。早期 Desktop 曾包含 MCP/stdio 自动化入口，但这会让 Desktop 从交互工作台扩张为长期兼容的 Agent 执行入口，也会迫使只需要自动化接入的部署安装完整 Desktop。

同时，MyFlowHub 的核心模型是一棵 authoritative Node tree：产品名称不构成权限，真实 authority 来自 Node identity、父子拓扑与策略。Agent Gateway 即使向网络外提供 API，也不应成为网络内的特权产品。

外部接入还需要服务多个 Codex/Agent，并为不同客户端配置不同权限。如果所有客户端共享 Gateway 的一个长期凭证，就无法独立撤销、限流或追责；如果一开始强制每个外部客户端成为树内 Node，又会把 Web 客户端生命周期和网络拓扑过早耦合。

## Options Considered

### 1. 将 Agent/MCP 接口继续内置在 Desktop

- 优点：本地 UI、身份和连接可以直接复用。
- 缺点：Desktop 同时承担控制台、执行面和外部 API 兼容责任；headless 使用者被迫安装 UI 产品；发布生命周期无法独立。
- 结论：拒绝。现有 Desktop MCP/stdio 入口已移除。

### 2. 独立 Gateway，但所有外部客户端共享一个访问令牌

- 优点：实现和配置最简单。
- 缺点：不能按客户端授权、撤销、限流和审计；共享泄漏的爆炸半径过大。
- 结论：拒绝。

### 3. 一个普通 Gateway Node 服务多个客户端，使用独立 Client Access Token

- 优点：Gateway 的网络权限继续受节点树约束；外部客户端可以独立授权、撤销、限流和审计；服务可 headless 部署并独立演进。
- 缺点：首期网络只看到 Gateway NodeID，客户端身份和 scope 需要 Gateway 可靠执行；Gateway 本身仍是多个客户端的共同故障域。
- 结论：采用。

### 4. 每个外部客户端对应独立 Node，或立即引入网络级委托身份

- 优点：树上策略和审计可以直接区分每个外部主体，隔离更强。
- 缺点：显著扩大协议、准入、路由、生命周期和凭证委托范围；当前没有足够实现需求支撑。
- 结论：延期。强隔离场景可先部署多个 Gateway Node，网络级委托身份由后续 ADR 决定。

## Decision

1. Agent Gateway 是独立、headless 的 Web 服务，也是 authoritative Node tree 中的普通 Node；它没有产品级超级权限或权限旁路。
2. “普通 Node”表示 Gateway 与 Desktop、MetricsNode 等产品一样由身份、拓扑和策略获得权限，不表示树上父子角色权限完全对称；现有父控子和 LCA 裁决保持不变。
3. Desktop、MetricsNode、Agent Gateway、Hub 等产品名只代表默认能力组合与交付形态，不代表固定网络角色、资源 owner 范围或特权级别。
4. 一个 Gateway Node 可以服务多个 Codex/Agent 客户端，并为每个客户端使用独立 Client Access Token。
5. Gateway 的 Node Admission Permit 与外部 Client Access Token 是两类凭证：前者只用于 Gateway Node 入网，后者只用于访问 Gateway Web API。
6. 允许操作取决于 Gateway Node 当前网络权限、Client Access Token scope 与 API schema/limit 校验的共同结果；任一层拒绝即 fail closed。
7. 首期 Gateway 以自身 NodeID 发起网络操作，网络内不伪造或委托外部 Agent 身份；Gateway 在本地以 `client_id` / `token_id` 完成隔离与审计。
8. Client Access Token 必须独立支持 audience、有效期、capability/resource scope、write 边界、配额、撤销和审计；Token 管理能力与普通访问能力分离。
9. Gateway 必须使用 canonical SDK 与 Resource/Subscription/Session 契约，不直接访问内部 store/router 绕过资源 owner 或树上裁决。
10. HTTP、MCP adapter、token 编码、管理 endpoint、TLS 和部署细节由实施前独立 spec 定义，本决策不提前冻结协议表面。

## Consequences

### Positive

- Desktop 保持通用交互工作台职责，不再承担外部 Agent API 的兼容周期。
- 只需要自动化接入的部署可独立安装 Gateway，无需安装 Desktop。
- 多个 Codex/Agent 可以共享一个服务进程，同时保持令牌、scope、配额、撤销与审计隔离。
- Gateway 无法因为产品名称获得超出节点树策略的权限；网络安全模型保持单一事实来源。
- 未来可按风险选择水平扩展多个 Gateway Node，或在有充分需求时引入委托身份。

### Costs and Risks

- Gateway Node 的网络权限是所有下游客户端能力的上界，其密钥失陷会影响它可访问的全部范围。
- 首期树上审计只看到 Gateway NodeID；必须依赖不可绕过的本地客户端审计完成细粒度追踪。
- 长连接订阅和 session 在令牌撤销、策略变化、断连与 reparent 时需要即时或有界失效。
- 多租户 Web 边界需要严格的输入验证、配额、秘密存储和管理面隔离。
- 外部 API 尚未冻结，产品不能在 spec 和验证完成前宣称兼容稳定。

## Confidence

High。独立 Web 服务、多客户端令牌和普通 Node 身份同时满足部署解耦、最小权限与现有 authoritative Node tree 约束；剩余不确定性主要在外部协议和凭证实现细节，适合留给后续 spec。

## Supersedes / Superseded By

- 取代“Desktop 内置 MCP/stdio 是长期 Agent 接口”的产品方向；历史 plan/change 中的实现记录仅保留为演进证据。
- 不取代 [统一权威节点树与可插拔链路](2026-08-27_authoritative-node-tree-and-pluggable-links.md)，而是在其约束下定义一种普通节点产品。
- 如果未来让外部 Agent 以 delegated subject 直接参与树上权限裁决，必须由新 ADR 明确扩展或取代本决策第 7 项。

## Related Requirements

- [Agent Gateway](../requirements/agent-gateway.md)
- [统一节点运行时](../requirements/unified-node-runtime.md)
- [Auth Controlled Admission](../requirements/auth-controlled-admission.md)

## Related Specs

- [节点树、链路与资源架构规范](../specs/node-tree-link-resource-architecture.md)
- [Resource platform v2](../specs/resource-platform-v2.md)
- [Resource sessions v2](../specs/resource-sessions-v2.md)

## Related Features

- [Desktop](../features/desktop.md)

## Related Intake

- [Agent Gateway 产品定义](../intake/2026-08-29_agent-gateway-product-definition.md)
