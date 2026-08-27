# Management Config Layering

## Scope

定义 vNext 节点显示名与可管理配置的稳定资源契约。旧 `config_get/config_set` action、management SubProto 和特定 Hub 文件路径不再是协议真相。

## Resource Contract

- `system/config` 是 `mfh.management.config.v1` Variable，包含 `version`、单调递增的 `revision`、可选 `display_name` 与有界字符串 `values`。
- `system/config/update` 是 `mfh.management.config-update.v1` Command；调用者必须提交 `expected_revision`，冲突显式失败。
- 成功更新必须原子持久化，再发布新的 Variable revision；持久化失败不能只修改内存快照。
- 读取需要 `management.config.read`，更新需要 `management.config.write`，资源可见性不能替代授权。

## Display Name

- `display_name` 是节点级、非身份字段；为空时 UI 回退到稳定 NodeID。
- `system/topology` 和 `system/config` 对同一节点必须返回一致名称。
- 修改显示名不轮换 key、NodeID、trust 或 admission permit。
- 当前连接中的 topology/config 订阅必须观察到更新，无需断线重连。

## Configuration Layers

- 身份、trust、admission、policy 和 listener 等启动安全状态由 host 持久状态管理，不进入普通 `values`。
- `system/config/update` 只修改可在线管理的有界配置，不隐式覆盖仅进程级的 CLI/部署参数。
- 产品专有设置使用各自的 `*/config` Variable 与 `*/config/update` Command，不向 `system/config` 堆叠任意 JSON。
- 配置损坏、revision 冲突或不支持的键显式失败，不回退到旧文件、环境变量或旁路 handler。

## Verification

- schema round-trip 与大小/字段校验；
- stale revision 被拒绝；
- 持久化后重启仍返回相同值和更高 revision；
- topology 与 config 的显示名一致；
- 授权失败不会改变状态或发出成功更新。

## Related Docs

- [resource-catalog.md](resource-catalog.md)
- [node-tree-link-resource-architecture.md](node-tree-link-resource-architecture.md)
- [hub.md](../features/hub.md)
