# 以独立能力表达浅层与递归拓扑查询

状态：Accepted，2026-09-06 已实现并验证。

现有 `system/topology` 是完整 Variable 快照；现有策略按 Subject、Resource、Capability 授权，不检查 payload。Desktop 需要默认一层导航及显式递归读取。

## 决策

同一资源增加 children 和 subtree；owner 就是查询根。数值 depth 统一表示范围，children 强制 depth=1，subtree 接受 0..4096。原 read/subscribe 保持兼容。具体合同见[规格](../specs/topology-discovery.md)。

| 备选 | 结果 |
| --- | --- |
| 完整快照后前端裁剪 | 已暴露深层内容，传输仍全量，不满足浅层授权 |
| 单 query 能力并扩展 payload policy | 需要新的策略、委托和审计语义，超出本轮 |
| 新建查询资源或另一棵权威树 | 引入重复身份与状态，增加维护成本 |
| 同一资源分 children/subtree | 使用现有权限模型，保持资源身份与协议兼容，采用 |

## 后果

旧宽 read/subscribe 权限仍可读取全量，不自动改写授权。深度不隐含权限，不自动向上查找网络根。父节点复用已维护关系构建有界快照，Desktop 缓存部分树。普通节点未必挂载 provider，明确表达不支持。全网实时推送、分页、通用 payload 权限与全 Host 自动管理资源留待独立设计。
