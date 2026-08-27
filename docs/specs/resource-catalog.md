# Resource catalog

## Built-in resource

每个 Node 必须注册 `system/catalog` Variable。它是本节点资源 descriptor 的版本化快照，不是独立 discovery 协议。catalog 本身也出现在快照中，但更新 catalog 值不得递归改变 descriptor 集。

## Schema

payload 使用 `application/json` 与 schema `mfh.catalog.v1`：

```json
{"version":1,"revision":3,"resources":[{"name":"system/catalog","kind":"variable","content_type":"application/json","schema":"mfh.catalog.v1","permission":"resource.catalog.read","max_value_bytes":1048576}]}
```

- `version` 必须为 1；未知 major 明确拒绝。
- `revision` 非零且只在 descriptor 集发生有效变化时单调增加。
- `resources` 按 name 排序；name 在单个 Node 内唯一。
- 每个 descriptor 受协议长度和 payload 上限约束；未知 JSON 字段可忽略，缺少必填字段失败。

## Update rules

注册、删除或替换 descriptor 后，在 registry 锁外编码并更新 catalog Variable。值更新必须是确定性的；资源当前值、Stream event、Command handler 或密钥不进入 catalog。

## Authorization

读取 catalog 仍需 `resource.catalog.read`。客户端不能因看见 descriptor 而假定自己有权限，必须以实际订阅或调用裁决为准。

管理资源 `system/policy/grant` 与 `system/policy/revoke` 是 Command，payload schema 为 `mfh.management.policy-rule.v1`。规则精确绑定 subject NodeID、`subscribe|invoke` action、resource owner NodeID 与 resource name；catalog 不承载 grant，也不会产生隐式通配权限。

## Related

- [Resource Model vNext](resource-model-vnext.md)
- [Subscription vNext](subscription-vnext.md)
