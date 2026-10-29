# Agent Gateway

## Status

产品定义已确认，实施延期到其他部分充分完善之后；不能仅以整体系统接口与权限模型趋稳代替这一条件。前置范围和验收门槛仍待收束，见[统一产品定义](product-definition.md)及[讨论 P03](../intake/2026-09-06_product-definition-grill.md)。本文件定义长期产品边界，不表示当前已有可用服务或稳定外部 API。

## Background

Codex 等外部 Agent 需要以标准 Web 接口发现和操作 MyFlowHub 网络，但不应嵌入 Desktop，也不应因接入方式不同获得绕过节点树的特权。旧 Desktop MCP/stdio 入口会让 UI 产品同时承担 Agent 接口兼容责任，现已从 canonical Desktop 中移除。

Agent Gateway 应作为独立、可无界面部署的产品加入网络，并在一个服务实例内安全承载多个外部 Agent 客户端。

## Goal

- 以独立 Web 服务向 Codex 等 Agent 暴露经过授权的 MyFlowHub 发现、读取、订阅、发布、调用和 session 能力。
- 让一个 Gateway Node 服务多个客户端，并为每个客户端提供独立身份标记、访问范围、配额、撤销和审计。
- 保持 authoritative Node tree、Resource ownership 和 canonical 权限裁决为唯一网络事实来源。
- 坚持“Desktop、MetricsNode、Agent Gateway、Hub 等产品名表示默认能力组合，不表示网络角色或特权级别”的长期原则。

## Product Model

Agent Gateway 同时具有两个不同安全边界：

1. **Gateway Node 边界**：服务使用自己的 Node identity，通过 Node Admission Permit 加入 authoritative Node tree，并受当前父子拓扑和网络策略约束。
2. **External Client 边界**：Codex/Agent 使用 Gateway 签发或登记的 Client Access Token 访问 Web API；每个令牌代表一个受限外部客户端，不直接成为 Node Admission Permit。

允许一次操作必须同时满足：

```text
Gateway Node 网络权限
  ∩ Client Access Token scope
  ∩ API endpoint / schema / limit validation
```

首期网络请求以 Gateway NodeID 为主体。`client_id` 和 `token_id` 由 Gateway 在本地认证、限制并审计，不伪装成树内其他 Node。未来只有在树上策略确实需要直接区分外部 Agent 时，才扩展委托身份。

## Scope

### Included

- 独立、headless 的 Agent Gateway Web 服务。
- 单个 Gateway Node 服务多个 Codex/Agent 客户端。
- Client Access Token 的签发/登记、验证、过期、撤销、scope、配额和审计边界。
- 对 Node/Resource catalog、descriptor、snapshot、subscription、generic operation 和 resource session 的受控 API 映射。
- 可选的 MCP adapter；它是 Web/API 边界的适配方式，不改变网络权限模型。
- Gateway 自身的 Node 生命周期、配置、健康状态和安全关闭。

### Out of Scope for the First Version

- 让 Gateway 成为全局超级用户、特殊根角色或权限旁路。
- 把 Gateway 合并进 Desktop，或要求客户端安装完整 Desktop。
- 每个外部客户端自动成为 authoritative Node tree 中的独立 Node。
- 网络级 delegated subject / delegated token。
- 在产品定义阶段冻结具体 HTTP path、MCP transport、token 编码或部署拓扑。
- 直接访问内部 store、router 或 owner 实现以绕过 canonical SDK 和资源契约。

## Scenarios

- 同一个 Gateway 为多个 Codex 会话签发不同 Client Access Token；一个只能读取指标，另一个可以调用指定维护操作。
- 某个客户端令牌被撤销后，其他客户端和 Gateway Node 的网络连接不受影响。
- Gateway Node 在树上被撤权或 reparent 后，即使客户端令牌仍声明原 scope，对应操作也立即不能继续执行。
- Client Access Token 允许订阅某个有界 Resource 范围，但订阅数、速率或 payload 超过限制时被明确拒绝。
- 运维人员根据 Gateway NodeID、`client_id` 和 `token_id` 追踪一次 Agent 操作，而日志中不出现令牌明文。
- 需要强隔离的部署为不同信任域运行多个 Gateway Node，而不是把一个超大权限 Gateway 作为必选中心。

## Functional Requirements

1. Agent Gateway 必须以普通 Node 身份加入 authoritative Node tree，具有独立 NodeID、本地状态和明确父节点关系。
2. Gateway 必须通过 canonical SDK、Resource descriptor/capability、subscription 和 session 边界操作网络，不得建立权限旁路。
3. 一个 Gateway 实例必须能够同时服务多个外部客户端，并为每个客户端使用独立 Client Access Token。
4. Node Admission Permit 与 Client Access Token 必须使用不同类型、配置入口和生命周期，不能相互替代或混用。
5. Client Access Token 至少必须关联 `client_id`、唯一 `token_id`、Gateway audience、有效期、capability scope 和 resource scope。
6. capability scope 至少应能分别限制 discover/read、subscribe、publish、invoke、open session 和显式 write 类操作。
7. resource scope 必须绑定明确 owner/resource path 或有界、可验证的 selector；不得通过客户端自报目标扩大范围。
8. 每个令牌必须支持有界速率、并发、订阅数、session 数和 payload 大小；超限必须返回明确错误。
9. Gateway 只能在自身当前网络权限范围内签发和执行客户端能力；客户端操作在任一权限层失败时必须 fail closed。
10. Gateway Node 权限、拓扑世代或策略变化后，既有客户端请求、订阅和 session 必须重新验证、失效或按 canonical 生命周期明确结束。
11. 令牌必须能独立过期和撤销；撤销一个令牌不得要求重建 Gateway Node identity 或影响无关令牌。
12. Token 管理操作必须受独立管理权限保护；普通 Client Access Token 不得签发新令牌、扩大自身 scope 或解除自身配额。
13. 完整令牌只允许在创建结果中向调用方返回一次；opaque token 只持久化不可逆摘要，签名 token 只持久化必要撤销状态和受保护的签名材料；后续请求、日志、错误与审计必须对令牌脱敏。
14. 每次受保护操作至少审计 Gateway NodeID、`client_id`、`token_id`、目标资源、capability、结果、时间和可操作错误，不记录不必要的敏感 payload。
15. 外部 API 必须由 Resource descriptor、schema 和 limits 驱动，未知 type/capability 不得通过猜测或静默 fallback 执行。
16. Gateway 必须可以在不安装、不启动 Desktop 的环境中独立部署和升级。

## Non-functional Requirements

- 安全：默认最小权限、短期凭证优先、敏感配置受保护、认证和授权失败显式可查。
- 隔离：单个客户端的慢请求、订阅洪泛、session 泄漏或错误不得无限占用整个 Gateway。
- 可审计：客户端操作可以从外部访问入口追溯到 Gateway Node 的网络行为。
- 可维护：Web/API adapter 不反向污染 Core 的 Node、Resource 或 Transport 抽象；Desktop 也不承担其兼容责任。
- 可演进：外部 API 和 token 格式必须版本化；未确定的 endpoint 与 transport 不在本阶段固化。
- 部署安全：远程监听必须支持受保护传输和明确的监听范围；具体 TLS/代理方案由后续 spec 定义。

## Edge Cases

- Client Access Token 有效，但 Gateway Node 已断连、被撤权或更换父节点。
- 令牌撤销与长连接 subscription/session 并发发生。
- 两个客户端拥有重叠 resource scope，但限额与 write 权限不同。
- Gateway 自身有较宽权限，而令牌请求超出已分配 scope。
- 令牌 audience、过期时间、签名/摘要或 resource selector 无效。
- Token 管理员试图签发超过 Gateway 可分配上限的 scope。
- 未知 Resource type、schema major version 不兼容或 owner 在请求期间消失。
- Gateway 重启后令牌状态、撤销记录、审计游标和有界 session 清理。

## Acceptance Criteria

- 一个 Gateway Node 可使用至少两个独立 Client Access Token 服务两个模拟 Agent，并证明 scope、配额和撤销彼此隔离。
- 相同操作分别覆盖 Gateway 网络权限拒绝、Client Access Token scope 拒绝、schema/limit 拒绝和完整允许路径。
- Node Admission Permit 无法作为 Client Access Token 使用，Client Access Token 也无法用于节点注册。
- Gateway 被撤权、断连或 reparent 后，已有请求、subscription 和 session 按 canonical 生命周期 fail closed。
- 无管理权限的客户端无法创建、扩大或恢复令牌；Gateway 也无法签发超过自身允许范围的令牌。
- 审计能够关联 Gateway NodeID、`client_id` 和 `token_id`，且自动化检查确认日志、错误和审计记录不包含令牌明文。
- Gateway 可以在没有 Desktop 的环境中构建、配置、启动、健康检查和安全关闭。
- 精确的 HTTP/MCP、token 和部署契约在实现前由独立 spec 与安全测试门禁确认。

## Related Features

- [Desktop](../features/desktop.md)（当前已移除内置 MCP/stdio；Agent Gateway 尚未实现为 feature）

## Related Requirements

- [统一节点运行时](unified-node-runtime.md)
- [可扩展资源平台](extensible-resource-platform.md)
- [Auth Controlled Admission](auth-controlled-admission.md)

## Related Specs

- [节点树、链路与资源架构规范](../specs/node-tree-link-resource-architecture.md)
- [Resource platform v2](../specs/resource-platform-v2.md)
- [Resource sessions v2](../specs/resource-sessions-v2.md)
- [Operational lifecycle](../specs/operational-lifecycle.md)

## Related Decisions

- [Agent Gateway 采用普通节点、多客户端令牌的独立 Web 服务](../decisions/2026-08-29_agent-gateway-ordinary-node-web-service.md)
- [统一权威节点树与可插拔链路](../decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md)

## Related Intake

- [Agent Gateway 产品定义](../intake/2026-08-29_agent-gateway-product-definition.md)
