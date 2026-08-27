# Authority, Routing And Subscription State

## Summary

持久 identity/desired interest 与 live session/route/attached interest 必须分离。转发必须保留原始主体和 generation；cache、transport session 或 next hop 都不能成为 authority 证明。

## Lookup Hints

- 症状：重连后重复订阅、reparent 后旧父继续控制、撤权后 owner 仍推送、response 回不到请求者。
- 关键词：`Principal`、`TopologyEpoch`、`PolicyGeneration`、desired interest、attached interest、pending requester、same-child routeback。
- 快速检查：比较 session generation、resource owner、actor、next hop、forwarded subscription 和 pending return record。

## Symptoms

- session reset 后旧 attached interest 仍被当成已恢复。
- 中继使用本地发送快捷路径，改写 `Principal`、message ID 或 sequence。
- 远端 cache 中存在资源就被误认为本地 owner/authority。
- 同一 child ID 的替换连接被旧 cleanup 撤销，或 routed response 错发给资源 target。

## Impact

造成权限提升、撤权失效、订阅泄漏、事件重复/丢失、错误返程和 reparent 后拓扑污染。

## Trigger Conditions

- durable snapshot 保存了 live session 标志。
- desired 与 attached 状态使用同一布尔值或 map。
- relay 复用 local-origin API，未携带原始 envelope 字段。
- cleanup 只按 NodeID，不校验 epoch/generation。

## Root Cause

路由边、authority 边和连接代次相关但不等价。节点树决定合法父控子和 next hop；具体 session 只承载当前代次。跨子树请求还需要有界 pending/forwarding 状态恢复返程和撤权控制。

## Investigation Trail

1. 区分持久 identity/config/desired interest 与 session/next hop/attached state。
2. 比较转发前后的 `Principal`、deadline、message/correlation ID、revision/sequence 和 topology epoch。
3. 检查 reparent、policy generation 更新和 link replacement 是否清理旧 route/pending/subscription。
4. 验证 routed response 通过 pending requester 返回，而不是假定 target 就是 caller。

## Resolution

- session reset 清除 attached 状态；新链路 ready 后按当前 topology/policy 重算并回放 desired interest。
- relay 使用专用 forwarding path，完整保留原始主体和时序字段。
- cache 只表示已观察数据，不授予 resource ownership 或 authority。
- cleanup 同时匹配 peer 与 generation/epoch；转发记录保持有界且不保存 payload。

## Prevention / Guardrails

- reparent、revoke、policy generation 和 same-peer replacement 必须有 focused/race/soak 测试。
- 所有 forwarded subscription 都能在授权点观察并撤销 owner 侧状态。
- 路由、权限和 session 诊断分别暴露，不用单个“connected”状态覆盖全部语义。

## Related Docs

- [Node tree architecture](../specs/node-tree-link-resource-architecture.md)
- [Subscription vNext](../specs/subscription-vnext.md)
- [Operational lifecycle](../specs/operational-lifecycle.md)
- [Session generation cleanup](session-replacement-generation-cleanup.md)
- [vNext full migration](../change/2026-08-27_vnext-full-migration.md)
