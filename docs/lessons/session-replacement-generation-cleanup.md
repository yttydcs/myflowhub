# 会话替换与异步清理必须绑定 Generation

## Summary

同一 peer 的新旧 LinkSession 可能短暂并存。新会话激活后，旧会话仍会执行关闭路径；如果清理只按 NodeID 或“当前 map entry”判断，就可能撤销新树边、遗留旧订阅，或让未登记的后台协程在 Node 关闭后继续写入已关闭通道。

正确模型是：topology state 按 peer + epoch 保护，link-local state 按具体 LinkSession 清理，所有 Node-owned 协程在 `closed` 状态锁下登记后才能启动。

## Lookup Hints

- 症状：
  - 同父重连稳定失败或偶发 `join topology epoch mismatch`。
  - 新连接建立后路由/树边又被旧连接关闭路径删除。
  - 被替换链路上的订阅直到租约过期才消失。
  - `Node.Close()` 偶发 `panic: send on closed channel`。
- 关键词：
  - `TopologyEpoch`
  - `DetachParentEpoch`
  - `WithdrawChildEpoch`
  - `CleanupLink`
  - `WaitGroup.Add`
  - `send on closed channel`
  - `join topology epoch mismatch`
- 快速检查：
  - 搜索所有 parent/child detach/withdraw，确认同时校验 session epoch。
  - 检查 superseded session 是否仍执行 `CleanupLink(current.linkID)`。
  - 检查所有后台协程是否在 `closed` 锁下先 `Add`，再 `go`。
  - 对同父重连、关闭和订阅清理执行 `-race` 与高重复计数。

## Symptoms

- 第二次连接同一 parent 时，本地 epoch 已被旧会话断开推进，新握手仍按“再次自增”提交，导致 ack epoch 与本地 epoch 不一致。
- parent 已把 child edge 从 epoch 1 更新到 epoch 2，但 epoch 1 的 run loop 随后无条件 `WithdrawChild(child)`，把 epoch 2 的边一起删除。
- session map 被新会话覆盖后，旧 run loop 因“不是 current”直接返回，跳过旧 link ID 上的订阅清理。
- 远程订阅协程未进入 Node 的 WaitGroup；Node 关闭 diagnostics 后，该协程仍调用 `emit`。

## Impact

- 重连不可用或出现偶发路由丢失。
- 订阅与 watcher 泄漏到租约截止时间。
- 旧父/旧链路状态可能污染新拓扑。
- 关闭路径产生进程级 panic。

## Trigger Conditions

- 同一 NodeID 快速重连或 Transport 迁移。
- 新旧会话的握手、心跳、关闭和 route cleanup 并发。
- 资源 owner 上存在绑定旧 LinkSession 的订阅。
- 外部 API 在 Node 即将关闭时启动远程订阅或连接。

## Root Cause

- 把 NodeID 当成唯一会话 generation，没有让 cleanup 携带 epoch。
- 把“是否仍是 sessions map 中的 current entry”错误地用作所有清理的前置条件。
- topology cleanup 和 link-local cleanup 没有分开：前者只能由当前 epoch 执行，后者必须由每个具体 session 执行。
- `WaitGroup.Add` 与 `closed` 检查没有原子化，违反了 Node close 的生命周期边界。

## Investigation Trail

1. 普通单次测试全部通过。
2. 增加同父重连测试后稳定复现 `join topology epoch mismatch`。
3. 跟踪发现 parent 更新 child epoch 后会关闭旧 session；child 旧 session 先 detach 并推进 epoch，新握手随后再次自增。
4. 将 parent 激活改为提交签名 epoch 后，又发现旧 session 的无条件 detach/withdraw 可以删除新边。
5. 把订阅计数加入重连回归后，确认 superseded session 跳过了 link cleanup。
6. 100 轮 Node 测试进一步触发 `send on closed channel`，定位到未跟踪的远程订阅协程。

## Resolution

- child 在验证 join ack 后提交 claim 中的精确签名 epoch，不再基于可能已变化的本地状态二次自增。
- 新增 epoch-aware parent detach 与 child withdraw；旧 generation 的关闭路径无法删除新边。
- 每个 runSession 无条件清理自己的 link ID，再单独判断是否仍可修改 topology state。
- Node 统一通过 closed-state lock 登记后台任务，`Close()` 设置 closed 后不再允许新的 `WaitGroup.Add`。
- 同父重连测试验证旧链路订阅被清理，新树边保持有效。

## Prevention / Guardrails

- 所有可替换对象都应有显式 generation/epoch；不要只用业务 ID 判断 cleanup 所有权。
- 将 topology、link-local、request-local 三类状态分别定义清理责任。
- stale session cleanup 必须是幂等且 generation-scoped。
- 启动 owner-managed goroutine 时，必须在 owner 的关闭锁下先登记生命周期。
- 重连测试至少覆盖“旧 session 在新 edge 提交之前关闭”和“提交之后关闭”两种顺序。
- 对连接、订阅与关闭路径持续运行 `go test -race` 和重复压力测试。

## Related Docs

- Intake:
  - [节点树、订阅与指令重构诉求](../intake/2026-08-27_node-tree-subscription-command-redesign.md)
- Requirements:
  - [统一节点运行时需求](../requirements/unified-node-runtime.md)
- Specs:
  - [节点树、链路与资源架构规范](../specs/node-tree-link-resource-architecture.md)
  - [Wire Protocol vNext](../specs/wire-protocol-vnext.md)
  - [Subscription vNext](../specs/subscription-vnext.md)
- Decisions:
  - [统一权威节点树与可插拔链路](../decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md)
- Changes:
  - [Canonical Monorepo 与统一节点运行时第一阶段](../change/2026-08-27_canonical-monorepo-unified-node-runtime.md)
