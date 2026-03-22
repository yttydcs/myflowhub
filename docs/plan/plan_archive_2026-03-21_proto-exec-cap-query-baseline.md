# Proto exec cap_query 基线确认

## Workflow 信息
- 仓库：MyFlowHub-Proto
- 分支：fix/proto-exec-cap-query
- Base：main
- Worktree：D:\project\MyFlowHub3\worktrees\fix-proto-exec-cap-query\MyFlowHub-Proto
- 当前阶段：4. 归档变更（已完成）

## 当前状态
- 已创建独占分支与 worktree。
- 当前 `protocol/exec/types.go` 已包含 `CapQueryReq/Resp` 与 `ActionCapQuery*`。
- Win 将对齐到 commit `7eef50d` 对应的 pseudo-version：`v0.1.2-0.20260318063708-7eef50dcc471`。

## 项目目标
- 作为跨仓 workflow 的协议基线仓，确认并归档 Win 所依赖的 exec `cap_query` 协议基线。

## 任务清单

- [x] `PROTO-BASELINE`
  - Owner：主Agent
  - Worktree：D:\project\MyFlowHub3\worktrees\fix-proto-exec-cap-query\MyFlowHub-Proto
  - Plan：D:\project\MyFlowHub3\worktrees\fix-proto-exec-cap-query\MyFlowHub-Proto\plan.md
  - 目标：记录 `7eef50d` 引入的 exec capability query 协议基线，供 Win 对齐与审计。
  - 涉及模块/文件：`protocol/exec/types.go`、`docs/change/**`
  - Write set：`docs/change/**`
  - 验收条件：
    - 文档中明确 `ActionCapQuery`、`ActionCapQueryResp`、`CapQueryReq`、`CapQueryResp` 的存在。
  - 测试点：
    - `go list -m -json github.com/yttydcs/myflowhub-proto@7eef50d`
  - 回滚点：
    - 删除本次文档变更。

## 完成情况
- 已完成：exec `cap_query` 基线确认与归档。
