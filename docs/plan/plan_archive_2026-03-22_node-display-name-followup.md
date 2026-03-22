# Plan Archive - 2026-03-22 Node Display Name Follow-up

## Workflow 信息

- Workflow：`node-display-name-followup`
- 控制仓：`MyFlowHub3`
- 控制分支：`feat/node-display-name-followup`
- Base：`master`
- 控制 worktree：`D:\project\MyFlowHub3\worktrees\MyFlowHub3-feat-node-display-name-followup`
- 关联 worktree：
  - `MyFlowHub-Win` -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-node-display-name-followup`
  - `MyFlowHub-SubProto` -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-feat-node-display-name-followup`
  - `MyFlowHub-Server` -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-feat-node-display-name-followup`

## 目标

- 在 Win `Devices` 现有 `Edit` 配置弹窗中让 `node.display_name` 可首次编辑。
- 让 Win 自身节点 `node_info` / Config 标题与远端节点显示名回退规则一致。
- 保持 `list_nodes` 低成本直连枚举，通过 auth bootstrap 和 `config_set_resp` 刷新让父节点稳定看到 child 名称。

## 文档治理与影响

- 使用 `$docs-governor` 结论：
  - 需求澄清落在 [management-node-display-name.md](../requirements/management-node-display-name.md)
  - 技术契约澄清落在 [management-config-layering.md](../specs/management-config-layering.md)
  - Server auth 长期说明落在 `MyFlowHub-Server/docs/specs/auth.md`
  - 完成归档落在 `docs/change/2026-03-22_node-display-name-followup.md`
- Requirements impact：`clarify`
- Specs impact：`clarify`

## Checklist

- [x] `CTRL1` requirement/spec/主计划澄清
- [x] `WIN1` Win Config synthetic `node.display_name`
- [x] `WIN2` Win self `node_info` 与 auth `display_name`
- [x] `SUB1` Auth direct-child `display_name` bootstrap
- [x] `SUB2` Management `config_set_resp` rename refresh
- [x] `SRV1` `hubruntime` 父链 register 携带 `display_name`
- [x] `INT1` 跨仓验证
- [x] `REV1` 强制 Code Review
- [x] `ARC1` 归档与索引更新

## 关键设计点

- `list_nodes` 继续只做 O(children) 的连接枚举，不改成按 child 远程查 `node_info`。
- auth 扩展使用本地兼容 JSON struct，而不是在本轮继续拉高 Proto 依赖升级面。
- direct-child rename refresh 只在 `hdr.SourceID == conn.meta(nodeID)` 时生效，避免后代名称污染中间连接。
- `assist_register` 前绑定阶段缺少稳定的“直连 child 本人”判定，因此 authority 侧名称 bootstrap 以 `login` 和后续 rename refresh 为主。

## 验证摘要

- Win：
  - `GOWORK=off go test ./... -count=1` 通过
  - `npm run build` 失败，阻塞于仓内既有 `wailsjs` 生成绑定缺失
- SubProto：
  - `auth`: `GOWORK=off go test ./... -count=1` 通过
  - `management`: 通过临时 `modfile + replace` 指向本地 `exec` module 后 `go test ./... -count=1` 通过
  - 直接 `GOWORK=off go test ./...` 仍受既有 published `exec` 版本缺少 `exec/capability` 包阻塞
- Server：
  - `GOWORK=off go test ./hubruntime -count=1` 通过
  - `GOWORK=off go test ./... -count=1` 失败，阻塞于仓内既有 `protocol/exec/types.go` 与 Proto 类型不匹配

## Code Review 结论

- 需求覆盖：通过
- 架构合理性：通过
- 性能风险：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖：通过
- 子 Agent 治理与审计：通过

## 回滚点

- Win：回退 `frontend/src/stores/management.ts`、`frontend/src/pages/Devices.vue`、`internal/services/auth/service.go`、`internal/services/management/service.go`、`app.go`
- SubProto：回退 `auth/types.go`、`auth/actions_register.go`、`auth/actions_login.go`、`auth/session.go`、`management/management.go`
- Server：回退 `hubruntime/runtime.go` 与 `docs/specs/auth.md`
