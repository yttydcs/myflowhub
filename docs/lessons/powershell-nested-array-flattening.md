# PowerShell 嵌套数组自动展开导致安全预检误报

## Summary

PowerShell 会枚举函数和管道输出中的集合。把一个数组放进另一层数组或通过函数返回时，调用方收到的形状可能被扁平化；依赖嵌套数组位置和 `Count` 的安全预检因此可能产生假失败或错误计数。跨仓清理门禁应使用具名对象、hashtable 或明确 schema，而不是隐式嵌套数组。

## Lookup Hints

- 症状：目标数量、允许状态数量或候选计数与逐项输出不一致；数据本身正确但比较仍失败。
- 关键词：`PowerShell nested array`、`pipeline enumeration`、`array flattening`、`Count drift`、`Compare-Object`。
- 触发条件：函数返回数组；数组被追加到另一数组；管道收集结果；单元素集合退化为 scalar。
- 快速检查：
  - 输出 `$value.GetType().FullName` 和 `@($value).Count`。
  - 对每个元素输出类型、关键属性和索引。
  - 把嵌套数组替换为 `[pscustomobject]` 或按稳定 key 建立的 hashtable 后重跑只读检查。

## Symptoms

- 预检声称目标缺失、候选数量漂移或状态集合不同，但逐项 Git/文件哈希检查正常。
- 期望“每个目标拥有一个 Allowed 数组”，实际收到的是所有 Allowed 元素被摊平成同一层。
- 单元素结果和多元素结果的数据类型不一致，导致 `.Count` 或位置索引判断不稳定。

## Impact

在破坏性 Git/worktree 清理中，这种误报通常会阻止安全操作；如果脚本反向把错误形状解释为通过，也可能扩大删除范围。因此集合形状必须被视为安全边界的一部分。

## Trigger Conditions

- PowerShell 函数直接输出数组或多条 pipeline object。
- 使用 `+=` 聚合数组且预期保留嵌套层级。
- 在属性中混用 scalar、单元素数组和多元素数组。
- 使用位置索引表示 target、allowed residue、recovery refs 等不同语义。

## Root Cause

PowerShell pipeline 会枚举大多数集合；函数返回值本质上也是写入 success output stream 的对象序列。调用方收集输出时，原本的嵌套数组边界不一定保留。单元素集合还可能在后续表达式中表现为 scalar，使位置和数量判断进一步不稳定。

## Investigation Trail

1. 首次 `PRE-1` 脚本报告目标/残留集合不匹配。
2. 逐项核对 branch、tip、status、SHA-256 和 recovery ref 后未发现真实漂移。
3. 检查运行时对象类型和数组层级，确认函数输出被 PowerShell 枚举。
4. 将目标、允许残留和恢复引用改为具名对象/映射结构。
5. 重新运行完整只读门禁，12 个目标、13 条残留状态、18 个恢复引用和 281 个分支候选全部通过。

## Resolution

- 用 `[pscustomobject]` 表达 target，并给 `Allowed`、`Recovery` 等字段明确命名。
- 跨目标查找使用 owner/path/branch 等稳定 key，不依赖数组位置。
- 在破坏性命令前重新校验 runtime type、元素数量和精确字段。
- 首次异常只作为 Failed/Blocked 处理，不绕过门禁；修复数据结构后从头重跑只读预检。

## Prevention / Guardrails

- 安全脚本优先使用具名对象或 hashtable schema。
- 对单元素和多元素输入都运行测试，显式使用 `@(...)` 归一化消费侧集合。
- 任何 count drift 都应先检查运行时类型与枚举行为，再判断 Git 状态是否真实漂移。
- 破坏性命令不得依赖未经验证的嵌套数组位置。
- 修复预检脚本后必须完整重跑，不复用首次失败运行的部分结果。

## Related Docs

- [2026-08-23_worktree-branch-balanced-cleanup.md](../change/2026-08-23_worktree-branch-balanced-cleanup.md)
- [plan_archive_2026-08-23_worktree-branch-balanced-cleanup.md](../plan/plan_archive_2026-08-23_worktree-branch-balanced-cleanup.md)
- Related intake: none
- Related features: none
- Related requirements: none
- Related specs: none
- Related decisions: none
