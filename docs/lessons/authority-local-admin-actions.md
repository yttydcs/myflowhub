# Authority Administration Must Follow The Node Tree

## Summary

vNext 中父子链路同时是路由边和 authority 边。管理操作可以跨多跳路由，但授权主体必须由 `Principal` 保留，最终由目标节点按当前树关系、policy generation 和 topology epoch 校验；不能把“请求来自父链路”简化成“任意中继拥有管理员权限”。

## Failure Pattern

- UI 或 CLI 把超时当成权限拒绝，没有展示结构化错误；
- 中继把自己的节点身份写进转发请求，导致权限被意外提升或拒绝；
- 节点 reparent 后继续接受旧 policy generation 或旧 topology epoch 的管理请求；
- 离线引导和在线管理共用旁路接口，形成第二套权限真相。

## Guardrails

- admission permit 与初始 policy 可由本机离线 CLI 引导；进入在线状态后统一通过 `system/*` Command 管理。
- 转发必须保留原始 `Principal`、deadline、message/correlation ID 和拓扑代次。
- 权限拒绝、过期、拓扑失效必须返回可观察错误，不能退化为等待超时。
- reparent、revoke 或 policy generation 更新必须使旧授权和相关订阅失效。

## Related Docs

- [node-tree-link-resource-architecture.md](../specs/node-tree-link-resource-architecture.md)
- [operational-lifecycle.md](../specs/operational-lifecycle.md)
- [hub.md](../features/hub.md)
