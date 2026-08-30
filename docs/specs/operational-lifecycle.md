# Operational lifecycle

## States

长期运行组件统一使用 `new → starting → running → stopping → stopped` 生命周期。连接型组件在 running 内额外暴露 `disconnected / connecting / connected / degraded`。状态转换是单向的；显式重新创建实例才能从 stopped 再启动。

NodeHost 使用两阶段边界：`New` 验证配置、加载持久 state 并创建本地 Node/attached Client，但不建立树边；`Start` 才启动 Listeners 与 Parent supervision。产品可在二者之间注册固定资源。

Host 是 Node、ParentSupervisor、Listeners 和后台任务的 owner。`Host.Client()` 返回非 owning operation facade；Client-local cleanup 不得关闭或重配 Host。

## Start

- `Start` 前必须已经验证配置、加载身份和持久状态，然后才创建 listener 或外部 goroutine。
- 部分 listener 启动失败时，已启动资源按逆序关闭并返回聚合后的可行动错误。
- 准备完成前不得对外宣称 healthy 或接纳树边。
- Parent-only、Parent + Listeners、Listeners-only 与无外部链路使用同一状态机；角色只是配置组合。
- 修改持久 identity、admission 或 policy 的离线引导命令只能在组件停止时执行；运行中管理必须走已鉴权 Command，禁止两个进程并发写同一状态目录。
- 新设备在 Authority Grant 原子保存前没有 Node ID；Enrollment 的重试与断线恢复遵循 [node-enrollment-and-admission-authority.md](node-enrollment-and-admission-authority.md)，不得在 supervisor 中本地补发身份。

## Run and supervision

- 每个后台任务有明确 owner、取消来源和退出汇报通道。
- temporary error 可进入有上限和 jitter 的退避；认证、revoke、配置损坏和不支持能力属于 permanent error。
- supervisor 暴露连接状态、最近错误摘要、重试次数和下次重试时间，但不得包含密钥、permit 或业务正文。

## Stop

- `Stop`/`Close` 幂等：停止接纳新工作，取消 Parent supervision，逆序关闭 listener，等待有界 drain，再关闭 Node/store。
- deadline 到达后返回未完成组件列表；不得吞掉 goroutine、pending call 或临时文件泄漏。
- 崩溃恢复只读取已完成原子写入的版本化状态。

## Reconnect and reparent

同一父身份和同一 policy generation 的重连可以恢复声明为 durable 的订阅意图，但创建新的 link generation。父身份或 authority generation 变化即 reparent：旧树边、旧控制权、旧 permit、旧 pending 和旧订阅全部失效，不自动复制。

## Related

- [Subscription vNext](subscription-vnext.md)
- [Command vNext](command-vnext.md)
- [Hub feature dossier](../features/hub.md)
- [NodeHost Runtime](node-host-runtime.md)
