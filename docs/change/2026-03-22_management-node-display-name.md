# 2026-03-22 Management Node Display Name

## 变更背景 / 目标

为 management `list_nodes` / `list_subtree` 引入节点显示名字段，使消费端可以优先展示节点昵称，同时保持旧客户端兼容。

## 具体变更内容

- `protocol/management/types.go`
  - 在 `NodeInfo` 中新增可选字段 `display_name`
  - `ListNodesResp` / `ListSubtreeResp` 复用该结构体，自动具备新字段
- `node_info` 保持原有 `items` 承载方式，不改 `NodeInfoResp` wire 结构

## plan.md 任务映射

- `PROTO1 - Add Display Name To Management NodeInfo`

## 关键设计决策与权衡

- 采用可选字段 `display_name`，避免破坏旧消费者的解码行为
- 不同时改 `NodeInfoResp` schema，保持 `node_info.items["display_name"]` 与既有扩展方式一致

## 需求 / 规范影响检查

- 控制面 requirement 已记录在 `D:\project\MyFlowHub3\docs\requirements\management-node-display-name.md`
- 控制面 spec 已记录在 `D:\project\MyFlowHub3\docs\specs\management-config-layering.md`
- 本仓未维护独立 `requirements/specs` 索引，本次无需新增 repo-local 真相文档
- 本仓 `docs/change/` 无独立索引 README，本次无需更新分类索引
- 无 lessons 沉淀新增

## 测试与验证方式 / 结果

在本 worktree 下执行：

```powershell
$env:GOWORK='off'
go test ./... -count=1
```

结果：通过。

## 潜在影响与回滚方案

### 潜在影响

- 只有下游实现实际填充 `display_name` 时，消费端才会看到昵称；否则应继续回退 `node_id`

### 回滚

- 回退 `protocol/management/types.go` 中 `NodeInfo.display_name` 字段

## 子 Agent 执行轨迹

- `PROTO1` -> `Lovelace (019d15a7-a200-73e2-8c6a-92f194e44ece)` -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-Proto-feat-management-node-display-name`
  - 文件：`protocol/management/types.go`
  - 验收：schema 向后兼容，模块测试通过
