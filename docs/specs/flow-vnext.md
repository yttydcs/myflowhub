# Flow vNext

## Current Status — Retired

当前 Flow 已移除，重新设计列为[待办](../requirements/flow-redesign.md)。Hub 不再注册 Flow 资源，
协议与生成契约不再导出 Flow schema，Go SDK 和 Android 不再提供 Flow 专属入口。
已有 `flow.json` 原样保留但不再读取、执行或更新；通用资源调用、订阅、Collection 和授权继续保留。
本次不引入替代执行器或内部语言。参见[移除决策](../decisions/2026-09-06_retire-flow-pending-redesign.md)。

## Historical Reference

以下保留退役前的描述，供重新设计时回顾。下文的“当前”“已实现”及 API/测试说明均属于历史状态，
不代表当前版本能力，也不要求继续维护旧 Flow。

## Status

Current contract。runtime catalog、Go SDK、bindings manifest 与 generated contract 已收敛为两个 Collections；
当前行为见 [Flow feature dossier](../features/flow.md)，通用 Collection/action 边界见
[Resource Collections and Actions](resource-collections-and-actions.md)。

## Resources

- `flow/definitions` Collection：管理按 flow ID 定位的版本化定义；按 descriptor 提供
  `list/get/create/update/archive/run` capabilities。
- `flow/runs` Collection：管理有界活动与近期运行实例；按 descriptor 提供
  `list/get/subscribe/cancel` capabilities。

`create/update/archive/run` 是 definitions 的 Resource capabilities，`cancel` 是 runs 的 Resource
capabilities，不再默认注册为 `flow/create`、`flow/update`、`flow/run`、`flow/cancel`、`flow/archive`
独立 Command Resources。调用 `flow/definitions.run` 会在 `flow/runs` 中产生一个 member；definition/run
只有需要独立全局权限、引用、订阅、生命周期或 owner 时才提升为 catalog Resource。

| Resource | Capabilities | Contract |
| --- | --- | --- |
| `flow/definitions` | `list/get/create/update/archive/run` | list/get 使用 Collection seam；其余使用 Flow domain schemas |
| `flow/runs` | `list/get/subscribe/cancel` | subscribe 产生 Flow events；cancel 幂等并保留 initiator/domain checks |

授权按 exact Flow ResourceID + CapabilityID 分离；旧 Resource grant 不自动映射或扩大。catalog 已不再包含
旧五个 Flow Command Resources 或 `flow/events` Stream；saved Widget 引用保持 missing，不静默重定向。

## Definition

schema `mfh.flow.definition.v1` 包含稳定 ID、revision、nodes、edges、inputs 和 limits。解析时拒绝未知 node kind、重复 ID、悬空 edge、违反该 kind 约束的 cycle、超限结构和未知 major。定义引用资源完整 ID，不保存 Transport 或物理连接信息。

## Run

run request schema `mfh.flow.run.v1` 包含 run ID、flow ID/revision、dedupe key、deadline 与 inputs；
`mfh.flow.run-summary.v1` 包含 run/flow ID、flow revision、initiator、dedupe key、state、开始/完成时间与 error，
不包含 outputs。节点 outputs 只在执行过程中向后继节点传递，不通过 run summary 暴露。每个 flow 和 owner 均有
并发上限。取消幂等；terminal 状态不可逆。retry 只针对可重试执行错误，并受次数、deadline 和 backoff 上限约束。

每次资源调用保留原始 initiator 与 authority context；Flow host 不能成为隐式超级用户。定义和运行状态用原子、版本化 store 保存，恢复时把不确定的 in-flight 调用标为 interrupted，而不是假定成功。

## Related

- [Flow feature dossier](../features/flow.md)
- [Resource Platform v2](resource-platform-v2.md)
- [Resource Collections and Actions](resource-collections-and-actions.md)
