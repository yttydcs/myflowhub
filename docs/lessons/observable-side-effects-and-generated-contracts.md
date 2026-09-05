# Observable Side Effects And Generated Contracts

## Summary

替代 provider、直接存储写入或异步 trigger 即使改变了最终值，如果没有产生相同的 revision/sequence、subscription delivery、gap/error 和 audit，也不是等价实现。协议和平台 binding 同样必须来自一个可生成的 contract 真相。

## Lookup Hints

- 症状：值已变化但订阅者没收到、Flow trigger 执行却缺事件、生成后 contract diff、某平台手写类型与 Go schema 不一致。
- 关键词：observable side effect、revision、sequence、StreamGap、runtime context、generated contract、foreign binding、single source of truth。
- 快速检查：从 canonical mutation 入口到 subscriber/audit 的完整路径；重新生成是否零 diff；异步任务是否携带主体/deadline/cancel。

## Symptoms

- 测试只断言最终存储值，线上订阅 UI 保持旧状态。
- alternative provider 绕过 Variable/Stream 更新与审计。
- async trigger 丢失 server/runtime context，执行部分成功但通知失败。
- Wails/AAR/Flutter 接口靠手写副本维持，生成物与 schema 漂移。

## Impact

造成不可观察状态、静默数据陈旧、审计缺口、跨平台不兼容和无法从干净 checkout 重建的发布产物。

## Trigger Conditions

- 直接调用 store 或 platform adapter，绕过 resource controller。
- 只把内部接口可替换当作行为等价证明。
- 异步任务依赖同步调用栈的隐式 context。
- 多处维护 action/payload/type 表，或复制旧 generated output。

## Root Cause

系统契约不仅包含最终数据，还包含订阅和错误的可观察时序。生成 contract 也不是文档副本，而是 protocol schema 到平台 API 的受测派生物；复制会建立第二个真相源。

## Investigation Trail

1. 列出 canonical mutation 应产生的 revision/sequence、delivery、gap/error 和 audit。
2. 对比替代路径是否经过同一 resource/controller 边界。
3. 检查 async context 中的 principal、deadline、cancel 和 runtime dependencies。
4. 从干净 worktree 重新生成并检查 diff、round-trip 和平台导出。

## Resolution

- 所有 mutation 统一进入 Variable/Stream/Command controller，替代实现只替换窄 adapter。
- 异步任务显式携带所需 context，不使用隐藏 global/server pointer。
- wire/payload 只在 `protocol/` 定义，SDK/Wails/mobile contract 从 canonical 输入生成。
- CI 对重新生成后的 Git diff、关键 schema round-trip 和跨语言 fixture 失败关闭。

## Prevention / Guardrails

### 快照发布重入与失败终止

当wrapper更新Variable时，不能在controller全局锁中调用watcher；仅移到锁外仍不足以保证顺序：第一个watcher重入Refresh可能先向其他watcher发布新版本，然后外层继续发布旧版本。使用单一发布排空器，重入只更新待发布快照；双watcher回归应改变树并断言各自版本不倒退。

当旧全量快照超过协议上限时，已有订阅必须明确终止，不能继续展示最后一个正常值。检查链路：`Observation.Failure → subscription.EventFailure → node.sendFailure`。终止应清空待发值、阻止晚到观察、取消watcher/interest/entry并释放客户端pending；慢消费者与即时失败的租约清理都应经过race验证。当前合同见[拓扑发现](../specs/topology-discovery.md)，本轮证据见[变更归档](../change/2026-09-06_depth-scoped-topology-discovery.md)。

- 测试同时断言最终状态和 subscriber/audit 观察结果。
- 新 provider 必须通过相同 contract suite，不能只做接口编译测试。
- 不允许手写 action 表、本地 proto shadow type、foreign `wailsjs` 或 AAR stub fallback。

## Related Docs

- [Resource model](../specs/resource-model-vnext.md)
- [Subscription vNext](../specs/subscription-vnext.md)
- [Flow vNext](../specs/flow-vnext.md)
- [Protocol mapping](../specs/protocol_map.md)
- [vNext full migration](../change/2026-08-27_vnext-full-migration.md)
