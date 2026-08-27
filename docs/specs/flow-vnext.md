# Flow vNext

## Resources

- `flow/definitions` Variable：按 flow ID 排序的版本化定义摘要。
- `flow/runs` Variable：有界活动与近期运行摘要。
- `flow/events` Stream：运行状态变化、节点结果摘要和 terminal event。
- `flow/create`、`flow/update`、`flow/run`、`flow/cancel`、`flow/archive` Commands。

## Definition

schema `mfh.flow.definition.v1` 包含稳定 ID、revision、nodes、edges、inputs 和 limits。解析时拒绝未知 node kind、重复 ID、悬空 edge、违反该 kind 约束的 cycle、超限结构和未知 major。定义引用资源完整 ID，不保存 Transport 或物理连接信息。

## Run

run schema `mfh.flow.run.v1` 包含 run ID、flow revision、initiator、dedupe key、deadline、state 和 bounded outputs。每个 flow 和 owner 均有并发上限。取消幂等；terminal 状态不可逆。retry 只针对 node kind 声明的 temporary failure，并受次数、deadline 和 backoff 上限约束。

每次资源调用保留原始 initiator 与 authority context；Flow host 不能成为隐式超级用户。定义和运行状态用原子、版本化 store 保存，恢复时把不确定的 in-flight 调用标为 interrupted，而不是假定成功。

## Related

- [Flow feature dossier](../features/flow.md)
- [Resource Model vNext](resource-model-vnext.md)
- [Command vNext](command-vnext.md)
