# 旧 MyFlowHub 仓库退役与文档提炼

## Outcome

2026-08-27，在 canonical monorepo 完成全量迁移、inventory 无未决软件项且用户显式授权后，移除了 `D:\project\MyFlowHub3\repo` 下全部 10 个 `MyFlowHub-*` 本地仓库。`repo` 父目录保留为空目录；canonical worktree、控制仓和其他 worktree 未被修改或删除。

移除的仓库：Android、ClipboardNode、Core、EmbeddedSDK、MetricsNode、Proto、SDK、Server、SubProto、Win。固定 commit、tree、tracked file count、目标目录与逐项 migrate/replace/drop 结论分别保存在 `migration/source-audit.json`、`migration/sources.yaml` 和 `migration/inventory.json`。

## Documentation Audit

- 扫描旧仓 777 个 Markdown 文件；
- 193 个与 canonical docs 内容完全重复；
- 584 个内容独有，但主体是已废弃的 SubProto/VarStore/TopicBus action 表、多仓 release 计划和逐次 change log；
- 未将这些历史文件整目录复制为当前文档，以免重新引入第二套架构真相；
- 将仍适用于 vNext 的经验综合迁入 5 个稳定 lesson，并重写协议映射和 5 个仍引用旧多仓环境的 lesson。

新增的长期经验：

- [android-runtime-and-mobile-bindings.md](../lessons/android-runtime-and-mobile-bindings.md)
- [embedded-toolchain-and-board-preflight.md](../lessons/embedded-toolchain-and-board-preflight.md)
- [authority-routing-and-subscription-state.md](../lessons/authority-routing-and-subscription-state.md)
- [frontend-and-powershell-preflight.md](../lessons/frontend-and-powershell-preflight.md)
- [observable-side-effects-and-generated-contracts.md](../lessons/observable-side-effects-and-generated-contracts.md)

[protocol_map.md](../specs/protocol_map.md) 已从旧 SubProto 同步副本改为 vNext envelope operation、三资源模型和内置资源映射。历史 `docs/plan`、`docs/change` 仍可保留旧路径和动作名，因为它们记录的是当时事实，不作为当前实现入口。

## Destructive Operation Evidence

- 删除前验证每个目标解析为 `D:\project\MyFlowHub3\repo` 的直接子目录，且每仓只有自身 main worktree；
- 9 个仓库使用 Windows 回收站移除，可以从当前用户回收站恢复；
- `MyFlowHub-Win` 因超长路径无法进入回收站，随后仅剩被锁定的 `build/bin/myflowhub-mcp.exe`；停止 10 个从该精确旧路径运行的进程后永久删除，因此不能从回收站恢复；
- `MyFlowHub-Win` 删除前为干净提交 `606b747e8ecdeed368cf518f3629ac0b75c8ff60`，可从远端按 commit 恢复；
- 其余 9 仓除 MetricsNode 外均干净。MetricsNode 的 29 个未提交文件全部是旧 `windows/frontend/wailsjs` 生成绑定，路径、状态和 SHA-256 已审计，但正文未保留。

## Recovery

已提交内容按 [migration/sources.yaml](../../migration/sources.yaml) 中的 remote、branch 和 commit 临时 checkout。MetricsNode 未提交生成绑定只能按旧生成输入重建或与 hash 对照，不能从 canonical 仓恢复原始字节。恢复仅用于历史调查，不得重新接入 canonical build。

## Validation

- `go test ./internal/archtest ./internal/migrationtest`
- 删除后确认 `D:\project\MyFlowHub3\repo` 下不存在 `MyFlowHub-*` 目录；
- 当前 stable docs 入口不依赖已删除的本地仓库；
- Markdown 相对链接和迁移清单在删除后重新校验。

## Rollback

代码无需回滚；按 `migration/sources.yaml` 从远端 checkout 固定 commit 即可恢复任一干净旧仓。前 9 个目录也可在回收站尚未清空时恢复。不要恢复旧多 module 构建关系；需要回退 canonical 实现时应在当前 Git 仓按提交进行。
