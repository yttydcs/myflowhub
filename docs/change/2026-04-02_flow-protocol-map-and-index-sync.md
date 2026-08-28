# 2026-04-02_flow-protocol-map-and-index-sync

## 变更背景 / 目标

- `RC-P0-1` 到 `RC-P0-3` 已在各参与 worktree 内补齐 `flow` 的 `cancel_run`、`list_runs`、`flow.run` / `flow.read`。
- 在 workflow 未结束前，workspace 根级 `docs/` 仍需要作为接手和检索入口，明确：
  - `protocol_map` 的 canonical source 在哪里
  - root `protocol_map` 只是同步副本
  - 当前阶段不把 worktree-local change 归档提前搬回全局目录

## 具体变更内容

### 修改

- `docs/README.md`
  - 补充 `protocol_map` 是同步副本、不可在 workspace 根直接生成的入口说明。
- `docs/specs/README.md`
  - 明确 `protocol_map.md` 的 canonical / sync-copy 关系和正确更新顺序。
- `docs/change/README.md`
  - 收录本次 workspace-level convergence 归档。

### 验证结论

- `docs/specs/protocol_map.md`
  - 已与 `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\protocol_map.md` 比对
  - 本轮无需内容修改；保持 whole-file sync copy

## Requirements impact

- `none`

## Specs impact

- `none`

## Lessons impact

- `none`

## Related requirements

- 无

## Related specs

- `D:\project\MyFlowHub3\docs\specs\protocol_map.md`
- `D:\project\MyFlowHub3\docs\specs\README.md`
- `D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\protocol_map.md`

## Related lessons

- `D:\project\MyFlowHub3\docs\change\2026-02-20_protocol-mapgen.md`

## 对应 plan.md 任务映射

- `RC-P0-4`
  - root `protocol_map` sync-copy verification
  - root docs entry/index clarification
  - workspace-level convergence archive

## 经验 / 教训摘要

- `protocol_map.md` 在 workspace 根不是第二真源，只是 handoff 友好的同步副本。
- 如果入口 README 不明确说明 canonical source，后续接手者很容易在错误目录本地生成或手改副本。
- workflow 未结束时，worktree-local `docs/change/*` 不应提前批量迁回全局 `docs/change/`。

## 可复用排查线索

- 症状：
  - root `docs/specs/protocol_map.md` 内容看起来过时
  - 读者不清楚该在 Proto 还是 workspace 根更新协议映射
  - 根级 `docs/` 入口无法说明当前 protocol-map sync 规则
- 触发条件：
  - 更新了 `MyFlowHub-Proto` 但漏掉 root sync copy
  - 入口索引只给链接，不解释 canonical / sync-copy 边界
- 关键词 / 错误文本：
  - `protocol_map`
  - `protocolmapgen`
  - `sync copy`
  - `single source-of-truth`
- 快速检查：
  1. 运行 `git diff --no-index D:\\project\\MyFlowHub3\\worktrees\\proto-run-control-phase1\\docs\\protocol_map.md D:\\project\\MyFlowHub3\\docs\\specs\\protocol_map.md`
  2. 看 `docs/specs/README.md` 是否明确写出 canonical / sync-copy 关系
  3. 看 `docs/README.md` 是否说明 root `protocol_map` 不能在本地直接生成

## 关键设计决策与权衡

- 保持 `protocol_map` whole-file sync copy，而不是给 workspace 根维护一份定制版
  - 好处：减少双份手写说明和内容漂移
  - 代价：root 文档的更新说明需要放在索引 README，而不是直接嵌到 `protocol_map.md` 顶部
- 本轮只做控制面 convergence，不提前执行 workflow 结束时的 archive 搬运
  - 好处：阶段边界清晰
  - 代价：当前详细 repo-local 归档仍分散在各 worktree 下

## 测试与验证方式 / 结果

- `D:\project\MyFlowHub3`
  - `git diff --no-index D:\project\MyFlowHub3\worktrees\proto-run-control-phase1\docs\protocol_map.md D:\project\MyFlowHub3\docs\specs\protocol_map.md`
  - `git diff --check`
- 结果：
  - `protocol_map` 比对无差异
  - root docs 改动无格式错误

## 潜在影响

- 仅影响 workspace 根 docs 的可发现性和维护规则说明，不改变任何运行时行为。

## 回滚方案

1. 回退 `docs/README.md`
2. 回退 `docs/specs/README.md`
3. 回退 `docs/change/README.md`
4. 删除 `docs/change/2026-04-02_flow-protocol-map-and-index-sync.md`

## 子Agent执行轨迹

- 本轮未使用子Agent
