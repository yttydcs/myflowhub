# 2026-02-20 - Proto：协议映射文档生成器（路线 A：Go 为真源，生成衍生物）

## 变更背景 / 目标

`MyFlowHub-Proto` 作为协议字典仓库，`protocol/*/types.go` 是 wire 相关常量与 payload struct 的 single source-of-truth。

在日常演进中，协议映射/说明文档容易出现“代码已改但文档未同步”的漂移，导致：
- 接手者难以快速确认 SubProto/Action 的真实定义；
- PR Review 难以一次性看到“协议变更 + 文档变更”的全量影响；
- 下游（Server/SDK/Win）在排查对齐问题时成本上升。

本次变更采用路线 A：
- **仍以 Go 源码为 single source-of-truth**（不引入 CUE/YAML spec，不生成 Go types/constants）；
- 自动生成“协议映射文档”的生成区块，并在 Proto 仓库内强制校验，保证文档与代码一致。

## 总体方案

### single source-of-truth
- `protocol/*/types.go`

### canonical 文档（半自动）
- `docs/protocol_map.md`
  - 生成器仅覆盖 `<!-- BEGIN GENERATED -->` 与 `<!-- END GENERATED -->` 中间内容；
  - 生成区块外可保留手写说明（例如 `Notes（Manual）`）。

### 强制校验范围
- 仅在 `MyFlowHub-Proto` 仓库内强制：`go test ./...` 会校验 `docs/protocol_map.md` 是否最新。
- meta workspace（`d:\project\MyFlowHub3`）的 `docs/protocol_map.md` 仅为同步副本，不做门禁。

## 具体变更内容（新增 / 修改 / 删除）

### 新增
- 生成器 CLI：`cmd/protocolmapgen`
  - `-write`：写入更新 `docs/protocol_map.md`
  - `-check`：校验文档是否最新（不写入）
  - 参数化：`-protocol-root`、`-out`
- 生成器内部库：`internal/protocolmap`
  - 基于 `go/parser/go/ast` 解析 `protocol/*` 的 `SubProto* / Action* / types / 其它常量`
  - Markdown 渲染与“生成区块替换”逻辑
- 强制校验测试：`internal/protocolmap/doc_test.go`

### 修改
- `README.md`：补充 single source-of-truth、生成/校验命令、强制校验范围、同步到 MyFlowHub3 的方式说明

## 测试与验证方式 / 结果

验收（建议统一）：
```powershell
$env:GOTMPDIR='d:\\project\\MyFlowHub3\\.tmp\\gotmp'
New-Item -ItemType Directory -Force -Path $env:GOTMPDIR | Out-Null
$env:GOWORK='off'
go test ./... -count=1 -p 1
go run ./cmd/protocolmapgen -check -out docs/protocol_map.md
```

结果：通过。

## 潜在影响
- 修改 `protocol/*/types.go` 后，若未运行生成器更新 `docs/protocol_map.md`，`go test ./...` 会失败并提示更新命令。
- 生成器默认假定 `protocol/` 下的每个子目录都是一个 subproto 包且包含 `SubProto*` 常量；若未来引入非 subproto 的共享包，需要调整目录/约束或更新生成器规则。

## 回滚方案
- revert 本次提交即可移除门禁与生成器（回到“文档手工维护”的状态，存在漂移风险）。

## 计划任务映射
- PMGEN0：归档旧 plan ✅
- PMGEN1：新增 `docs/protocol_map.md`（含生成区块）✅
- PMGEN2~PMGEN6：解析/渲染/区块替换/CLI/强制校验 ✅
- PMGEN7：README 记录规则 ✅
- PMGEN8：回归验证 ✅

