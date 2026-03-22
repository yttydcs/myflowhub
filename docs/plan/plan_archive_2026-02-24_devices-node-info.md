# Plan - Devices：节点信息弹窗（management/node_info）

## Workflow 信息
- 范围：跨仓库（Proto + SubProto + Server + Win）
- 分支（各仓一致）：`feat/node-info`（已合并到 `main`）
- Worktrees（执行时）：
  - Proto：`d:\project\MyFlowHub3\worktrees\node-info\MyFlowHub-Proto`
  - SubProto：`d:\project\MyFlowHub3\worktrees\node-info\MyFlowHub-SubProto`
  - Server：`d:\project\MyFlowHub3\worktrees\node-info\MyFlowHub-Server`
  - Win：`d:\project\MyFlowHub3\worktrees\node-info\MyFlowHub-Win`
- Base：`main`
- Workspace（本地联调用）：`d:\project\MyFlowHub3\worktrees\node-info\go.work`
- 规范：`d:\project\MyFlowHub3\guide.md`（commit 信息中文，前缀可英文）

## 约束（边界）
- UI：使用页面内 Modal（不做多 OS 窗口）。
- 数据来源：
  - 远端节点：必须由 **目标节点本地采集并回包**（不允许 Hub/Win 侧推断/缓存伪造）。
  - Win 自身节点（sourceID==targetID）：由 Win 本地采集并返回（不走网络，避免 timeout）。
- 协议：仅新增 management action（wire **新增**，不改既有 action/schema/header 语义）。
- 展示：不强约束字段集合；以 `items[key]=value` 的 KV 形式返回并展示。

## Checklist（执行记录）

### T1. Workspace 准备（Completed）
- 目标：创建跨仓 worktrees，并提供本地联调 go.work。
- 验收：
  - 4 个 worktree 均存在且分支为 `feat/node-info`
  - `worktrees/node-info/go.work` 可被 go 命令识别

### T2. Proto：新增 management `node_info`（Completed）
- 变更：新增 `ActionNodeInfo/Resp`、`NodeInfoReq/Resp{items: map[string]string}`
- 验收：
  - `go test ./...`（Proto）通过
  - tag：`v0.1.1`

### T3. SubProto(management)：实现 `node_info` handler（Completed）
- 变更：目标节点本地采集并回包 KV（平台/版本/commit 等）
- 验收：
  - `go test ./...`（`myflowhub-subproto/management`）通过
  - tag：`management/v0.1.1`

### T4. Server：接入新能力 + 集成测试 + 发布（Completed）
- 变更：
  - 升级依赖：`proto v0.1.1`、`subproto/management v0.1.1`
  - 新增集成测试：`integration_management_node_info_test.go`
- 验收：
  - `go test ./...`（Server）通过
  - tag：`v0.0.2`（用于触发 GitHub Release）

### T5. Win：Devices 点击节点弹出 Modal + 查询 node_info（Completed）
- 变更：
  - `ManagementService.NodeInfo/NodeInfoSimple`（自身节点短路）
  - `Devices` 节点行点击弹窗，展示 KV，并支持 Reload/Close
- 验收：
  - `go test ./...`（Win）通过
  - 冒烟：点击任意节点弹窗可用；点击自身节点不超时

### T6. 合并与发布顺序（Completed）
- 已完成：
  - 合并回 `main` 并 push：Proto/SubProto/Server/Win
  - tags 已推送：`v0.1.1`、`management/v0.1.1`、`v0.0.2`

---

## 风险与注意事项
- 旧节点未实现 `node_info`：仍可能出现 timeout；Win 已用 Toast + 弹窗错误态提示，但需要逐步让节点补齐该动作。
- `debug.ReadBuildInfo()` 的 `Main.Version` 在某些构建方式下可能为 `(devel)`：因此同时返回 `commit/vcs_time/vcs_modified` 作为可定位信息。

