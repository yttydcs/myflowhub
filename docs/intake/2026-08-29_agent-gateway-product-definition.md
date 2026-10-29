# Agent Gateway 产品定义

## Source

- 来源：2026-08-29 Codex 讨论。
- 背景：Desktop 内置 MCP/stdio 入口已移除，对外 Agent 接口将在整体架构稳定后作为独立产品重新设计。

## Original Intent

用户对 Agent Gateway 的定位是：

- Agent Gateway 是 MyFlowHub 网络中的普通 Node，不是特权产品或网络外的超级控制面。
- 它以 Web 服务形式，把已授权的网络操作通过 API 暴露给 Codex 等 Agent。
- 一个 Agent Gateway 可以同时服务多个 Codex/Agent 客户端。
- Gateway 可以为不同客户端签发独立准入令牌，并通过令牌赋予不同、可撤销的权限。
- Desktop、MetricsNode、Agent Gateway、Hub 等产品名只表示默认能力组合；实际权限来自身份、拓扑关系和策略配置，而不是产品名称。

## Confirmed Product Direction

```text
Codex A ── Client Access Token A ─┐
Codex B ── Client Access Token B ─┼── Agent Gateway Web Service
Agent C ── Client Access Token C ─┘           │
                                               │ Gateway Node identity
                                               ▼
                                  authoritative Node tree
```

首期采用一个普通 Gateway Node 服务多个外部客户端的模型：

为避免与节点入网 permit 混淆，讨论中面向 Codex/Agent 的“准入 token”在正式产品术语中定名为 **Client Access Token**。

1. Gateway 使用自己的 Node identity 和 Node Admission Permit 加入 authoritative Node tree。
2. Codex/Agent 使用各自的 Client Access Token 访问 Gateway；该令牌不等于节点入网 permit。
3. Gateway 在本地完成客户端认证、scope、配额和输入验证，再以自身 Node 身份发起网络操作。
4. 一次操作只有在 Gateway Node 的网络权限、Client Access Token scope 和 API 边界校验均通过时才允许执行。
5. 网络首期只识别 Gateway NodeID；Gateway 额外记录 `client_id` / `token_id` 审计信息。
6. 如果未来需要网络策略直接区分外部 Agent，可增加委托身份/委托令牌；如果需要更强故障域隔离，也可部署多个 Gateway Node。

## Required Boundaries

- Gateway 不依赖 Desktop 安装，也不要求运行 UI。
- Gateway 不通过内部 store/router 绕过 canonical SDK、资源 owner 或树上权限裁决。
- Client Access Token 不能授予超过 Gateway Node 自身权限的能力。
- 不同客户端的令牌可以独立签发、过期、撤销、限流和审计。
- 令牌管理需要独立管理权限；普通客户端令牌不能为自己扩权。
- 完整令牌只在创建结果中返回一次；后续请求、日志和审计必须脱敏，且不得记录不必要的敏感 payload。
- 产品名称、部署包或 UI 不自动产生网络权限；拓扑角色之间现有的父控子权力仍按 authoritative Node tree 规则生效。

## Deferred Questions

- 首期外部协议仅提供 HTTP JSON，还是同时提供 MCP adapter；具体 endpoint、错误模型和版本协商尚未确定。
- Client Access Token 采用 opaque token、签名 token 或二者组合，及其轮换、吊销和持久化格式尚未确定。
- Token 管理面的 bootstrap、TLS、反向代理和远程部署策略尚未确定。
- 委托身份是否进入网络协议、何时需要一个外部 Agent 对应一个独立 Gateway Node，留待真实隔离需求出现后决定。

## Routed Documents

- 长期产品需求：[Agent Gateway](../requirements/agent-gateway.md)
- 架构决策：[Agent Gateway 采用普通节点、多客户端令牌的独立 Web 服务](../decisions/2026-08-29_agent-gateway-ordinary-node-web-service.md)
- 当前 Desktop 边界：[Desktop](../features/desktop.md)
