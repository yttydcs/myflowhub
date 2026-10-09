# Observable Side Effects And Generated Contracts

## Summary

替代 provider、直接存储写入或异步 trigger 即使改变了最终值，如果没有产生相同的 revision/sequence、subscription delivery、gap/error 和 audit，也不是等价实现。协议和平台 binding 同样必须来自一个可生成的 contract 真相。

## Lookup Hints

- 症状：值已变化但订阅者没收到、Flow trigger 执行却缺事件、生成后 contract diff、某平台手写类型与 Go schema 不一致；Catalog 失败却改变资源可见性、已移除实例仍能发布、一个 watcher panic 后所有更新停止。
- 关键词：observable side effect、revision、sequence、StreamGap、runtime context、generated contract、foreign binding、single source of truth、full envelope、catalog preflight、PrepareRetire、draining、cached definition。
- 快速检查：从 canonical mutation 入口到 subscriber/audit 的完整路径；重新生成是否零 diff；异步任务是否携带主体/deadline/cancel；完整编码是否早于提交、实例退役是否在 Catalog 事务内、排空器是否能越过故障 observer。

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

### 完整 Variable envelope 必须先于状态提交

业务 value 的 JSON 能编码且大小合规，不代表新状态能被读取：definition、instance、revision、状态和时间也占用资源限额。只在发送时包装，会出现值和 revision 已提交、完整 read 却超限的状态。Catalog 自身也是 Variable，同样可能在注册或注销时遇到这个问题。

在 [Variable.prepare](../../runtime/resource/variable.go) 中，注册时验证并编码不可变 definition；更新时用已验证定义检查值，同时准备省略 definition 的 data event 和携带 definition 的完整 snapshot。完整输出满足限额后才提交状态和通知队列。不能提高全局上限，或只测 inner value 大小来掩盖 wrapper 开销。

[Registry](../../runtime/resource/resource.go) 的资源表变更、下一版 Catalog 编码和 Variable 提交必须处于同一锁定边界；预检或提交失败时恢复资源表，保持 Catalog revision、订阅事件和实例状态不变。回归同时观察 `Resolve`、snapshot 与 watcher 次数，见 [variable_contract_test.go](../../runtime/resource/variable_contract_test.go) 的 `TestVariableEveryMutationRejectsBeforeStateAndObservation`、`TestVariableFullEnvelopeLimitPrecedesEveryMutation`、`TestRegistryCatalogFailureAndReentrantPublication`。

### 实例退役标记不能晚于 Catalog 事务

移除资源后再等 owner handler 返回、最后才标记 retired，会留下旧对象重新注册或在途写入继续提交的窗口。应在 Catalog 事务释放锁前同步执行 `PrepareRetire`，禁止旧实例再提交或注册；等待在途 writer 和投递终止通知的完成动作放到锁外，避免阻塞注册表及回调重入。

Catalog 预检失败不能顺便退役实例。成功移除后的旧 writer 即使返回，也必须被提交检查拒绝，旧 revision 不增加；这只保护平台确认状态，不表示外部 actuator 副作用可以回滚。`TestVariableRemovalInvalidatesBeforeBlockedWriterFinishes` 用阻塞 handler 验证可见性、旧实例注册拒绝及最终写失败，`TestRegistryCatalogFailureAndReentrantPublication` 验证失败注销保留实例，两者均位于上述回归文件。

### Variable 排空器必须隔离单个 observer 的 panic

单一排空器能解决重入乱序，但直接调用外部 watcher 仍可能因 panic 留下 `draining=true`，导致后续通知只入队、不再投递。恢复边界应围绕每个 observer：移除并取消故障 watcher，记录资源身份对应的错误，继续排空健康订阅；不要记录任意 panic 内容或资源值。

`TestVariableObserverPanicDetachesOnlyFaultyObserver` 验证健康 watcher 仍依次收到 revision 2、3、4，且队列和 draining 状态恢复。owner handler 的 panic 则属于执行结果未确认，需释放 mutation gate、保留最近确认状态，不能伪报写成功；对应 `TestVariableHandlerPanicAndUnknownOutcomeKeepLastConfirmedState`。两类恢复职责不同，不能用一层吞掉异常的 recover 代替。

### 递归定义保留类型引用，生成物与行为分别验收

Variable 的 definition 是递归语法，value 是 JSON 原生值。把 envelope 直接放入按字段展开的反射清单，会递归展开或把动态值错误描述为 string。canonical TS 生成应保留 `DataSchemaDefinition` 引用和 `JSONValue`，让 revision 保持 string；不能靠手写 UI 类型或旧 base64 变量 decoder 补齐。

[typescript_test.go](../../sdk/bindings/contract/typescript_test.go) 检查递归类型只声明一次、字段和生成文件一致；[共享校验向量](../../protocol/testdata/variable_validation_vectors.json) 则验证 Go/TS 对定义、值、修订及严格 wire 的相同判断。生成 freshness 与运行时校验是两份证据，任何一份都不能替代另一份。envelope 与缓存边界的排障见[跨运行时契约边界](collection-browser-cross-runtime-contract-boundaries.md)。

## Related Docs

- [Unified Variable Resource](../specs/unified-variable-resource.md)
- [Resource model](../specs/resource-model-vnext.md)
- [Subscription vNext](../specs/subscription-vnext.md)
- [Flow vNext](../specs/flow-vnext.md)
- [Protocol mapping](../specs/protocol_map.md)
- [vNext full migration](../change/2026-08-27_vnext-full-migration.md)
