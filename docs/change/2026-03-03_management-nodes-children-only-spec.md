# 2026-03-03 - Management：补充 Nodes(children-only) 协议语义说明

## 变更背景 / 目标

`protocol/management/types.go` 作为 wire 的 single source-of-truth，仅定义了动作名与 JSON payload schema，但未对 `list_nodes/list_subtree` 的“direct/subtree/parent/child”语义做明确约束。

在上游实现与 UI 侧使用中，`list_nodes` 被用于 Devices/Nodes 树的“children 枚举”，因此需要在协议说明中明确：
- `list_nodes` 的结果应被解释为 downstream children；
- 不应把 upstream parent link 作为 children 返回（否则会造成树回指/环）。

本次目标：在协议文档中补齐上述语义说明（不改 wire、不改生成区块）。

## 具体变更内容（新增 / 修改 / 删除）

### 修改
- `docs/protocol_map.md`
  - 在 `## Notes（Manual）` 增加 Management（Nodes）语义说明：
    - `list_nodes`：children-only（下游直连子节点）
    - `list_subtree`：`list_nodes` + self（不递归）
    - `nodes[].has_children`：best-effort hint，客户端不得据此做强约束

## 对应 plan.md 任务映射
- PMG1 - 补充协议文档语义说明（Manual Notes）✅
- PMG2 - 本地校验 ✅（见下）

## 关键设计决策与权衡
1) **仅在 Manual Notes 描述语义**
   - 避免编辑生成区块，保持生成器可重复运行与门禁稳定。

2) **不改 payload schema**
   - 现有 `NodeInfo` 结构缺少 parent/role 等字段，本次仅先明确“children-only”语义以消除歧义；
   - 若未来需要表达拓扑方向或链接信息，应通过新增动作或扩展结构另起设计。

## 测试与验证方式 / 结果
```powershell
$env:GOWORK='off'
go test ./... -count=1 -p 1
```

## 潜在影响与回滚方案

### 潜在影响
- 文档新增约束可能与旧实现不一致；需要上游实现同步调整（本次仅更新文档，不修改 wire）。

### 回滚方案
- revert 本次文档提交（仅影响 `docs/protocol_map.md` 的 Manual Notes）。

