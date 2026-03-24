# 变更归档：Win Flow Editor Bootstrap Baseline

## 变更背景 / 目标

- fresh worktree 下 `frontend/wailsjs/**` 不会随 git 带出，之前容易把前端构建失败误判成“代码坏了”。
- `FLOW-ENH-0` 的目标是把这条前端 bootstrap / verification 基线写回仓库 README，而不是继续依赖历史 change 或聊天上下文。

## 具体变更内容

- 更新 `README.md`
  - 新增 `Fresh Worktree Bootstrap (PowerShell)` 段落
  - 明确以下命令顺序：
    - `wails version`
    - `$env:GOWORK='off'; wails generate module`
    - `cd frontend`
    - `npm ci`
    - `npm run build`
  - 明确说明：
    - `frontend/wailsjs/**` 是 gitignore 生成物
    - `wails dev/build` 可能自动生成绑定，但 fresh worktree 推荐显式执行 `wails generate module`
    - backend binding 变化或 `frontend/wailsjs/**` 缺失时应重新执行生成步骤

## Requirements / Specs / Lessons Impact

- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `none`
- Related requirements:
  - `none`
- Related specs:
  - `none`
- Related lessons:
  - `none`

## 对应 plan 任务映射

- `FLOW-ENH-0`
  - `README.md`

## 经验 / 教训摘要

- 这类 bootstrap 信息如果只写在 `docs/change`，下一次 fresh worktree 仍然会重复踩坑；必须提升到仓库入口 README。
- 对当前 Win 仓库来说，前端最容易出问题的不是 `npm run build` 本身，而是缺少 `frontend/wailsjs/**`。

## 可复用排查线索

- 症状：
  - fresh worktree 下前端直接报缺少 `../../wailsjs/...`
  - `npm run build` 失败，但代码本身未必有语法问题
- 触发条件：
  - 新 worktree 未先生成 bindings
  - `frontend/wailsjs/**` 被清理或尚未生成
- 关键词：
  - `frontend/wailsjs`
  - `wails generate module`
  - `GOWORK=off`
- 快速检查：
  - 先看 README 的 bootstrap 段落
  - 先跑 `$env:GOWORK='off'; wails generate module`
  - 再跑 `cd frontend && npm ci && npm run build`

## 关键设计决策与权衡

- 选择只更新 `README.md`，不新增脚本：
  - 好处：最小变更面，避免为了固定命令顺序引入额外维护对象
  - 代价：仍要求开发者按文档执行，而不是一键脚本
- 明确使用 PowerShell 语法：
  - 好处：与当前 Win 仓库主场景一致，`GOWORK` 写法不含糊
  - 代价：若后续需要跨 shell 文档，可再补充 bash 等价写法

## 测试与验证方式 / 结果

- `wails version`
  - 结果：通过，输出 `v2.11.0`
- `$env:GOWORK='off'; wails generate module`
  - 结果：通过
- `cd frontend && npm ci`
  - 结果：通过
- `cd frontend && npm run build`
  - 结果：通过
- 残余告警：
  - `dist/assets/index-*.js` 仍约 `955 kB`，存在 chunk 过大告警，但不影响本轮 bootstrap 基线固化

## 潜在影响与回滚方案

### 潜在影响

- README 现在把 fresh worktree 的启动前置条件写得更明确，开发流程更可重复。

### 回滚方案

- 回退 `README.md`
- 回退当前 change 文档和 `docs/change/README.md` 索引项

## 子Agent执行轨迹

- 本轮未使用子Agent。
- 原因：
  - `FLOW-ENH-0` 为单文件文档收口任务，无并行价值。
