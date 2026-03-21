# Plan - Proto：补充 Management Nodes（children-only）协议语义说明

## Workflow 信息
- Repo：`MyFlowHub-Proto`
- 分支：`docs/management-children-only`
- Worktree：`d:\project\MyFlowHub3\worktrees\docs-proto-management-children-only`
- Base：`origin/main`
- 参考：`d:\project\MyFlowHub3\guide.md`（commit 信息中文）

## 背景 / 问题陈述
- `protocol/management/types.go` 仅定义了动作名与 JSON payload schema，并未对 `list_nodes/list_subtree` 的“direct/subtree/parent/child”语义做明确说明。
- 当前实现侧（`myflowhub-subproto/management`）存在“parent 连接也会被枚举进 list_nodes”的行为，导致 UI 侧将其误当 children 出现回指/duplicate。
- 本 workflow 目标是 **在协议文档中明确语义**：Nodes 查询仅用于 children 枚举；不把 upstream(parent) 作为 children 返回。

## 目标
1) 在 `docs/protocol_map.md` 的手写 Notes 区域补充 Management Nodes 语义说明：
   - `list_nodes`：children-only
   - `list_subtree`：direct(children) + self（不递归）
   - `has_children`：best-effort hint（客户端不得以此作为强约束）
2) 不修改生成区块（`BEGIN/END GENERATED`），保持生成器可重复运行。

## 约束（边界）
- wire 不改：不修改 `protocol/*/types.go` 的常量与 struct schema。
- 文档只允许修改 `docs/protocol_map.md` 的手写区域（`<!-- END GENERATED -->` 之后）。

## 验收标准
- `docs/protocol_map.md` 明确写出上述语义，且不触碰生成区块。
- `go test ./...`（建议 `GOWORK=off`）通过，确保文档门禁/生成器不受影响。

---

## 3.1) 计划拆分（Checklist）

### PMG0 - 归档旧 plan.md
- 目标：避免旧 workflow plan 覆盖本次任务。
- 已执行：`plan.md` → `docs/plan_archive/plan_archive_2026-03-03_proto-management-children-only-prev.md`
- 验收条件：归档文件存在且可阅读。
- 回滚点：撤销该移动提交。

### PMG1 - 补充协议文档语义说明（Manual Notes）
- 目标：在协议文档中明确 `list_nodes/list_subtree` 的 children-only 语义。
- 涉及文件：
  - `docs/protocol_map.md`
- 修改范围：
  - 仅允许修改 `<!-- END GENERATED -->` 之后的 `## Notes（Manual）` 区域（或其子段落）。
- 验收条件：
  - 文档包含：
    - `list_nodes`：仅返回 downstream children；不返回 upstream parent；
    - `list_subtree`：返回 `list_nodes` 的结果 + self（不递归）；
    - `has_children`：best-effort hint，客户端不得据此禁用展开。
- 回滚点：revert 文档提交。

### PMG2 - 本地校验
- 目标：避免破坏生成器门禁与仓库测试。
- 验收命令（建议）：
  - `$env:GOWORK='off'; go test ./... -count=1 -p 1`

### PMG3 - Code Review（阶段 3.3）
- 目标：检查文档是否清晰、准确、与实现兼容，且没有编辑生成区块。

### PMG4 - 归档变更（阶段 4）
- 目标：补齐可审计变更说明。
- 涉及文件：
  - `docs/change/2026-03-03_management-nodes-children-only-spec.md`（新增）
- 必含内容：
  - 变更背景/目标、具体变更内容、任务映射、测试与验证方式/结果、潜在影响与回滚方案。

