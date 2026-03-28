# Plan Archive - Auth 活动 permit 列表与 Win 准入许可页收敛

## Goal
- 为 auth 体系补齐真实的活动 permit 列表能力。
- 让 Win 的“准入许可”页从 latest-only 快照切换为 `Refresh + New Permit + 活动列表 + 行内 Revoke`。
- 保持权限模型最小改动，不新增独立 `auth.permit.list` 权限。

## Current State
- `MyFlowHub-Win` 的 permit issuance 页此前仍围绕 `lastIssued / lastRevoke` 组织，无法反映 authority 当前全部活动 permit。
- `MyFlowHub-Proto` / `MyFlowHub-Server` / `MyFlowHub-SubProto` 尚未提供 `list_register_permits` 稳定契约和运行时实现。
- `MyFlowHub-Win` 现有 requirements / specs 仍写着“不支持 permit list/query”，和目标交互不一致。

## Workflow Information
- Control plane repo: `MyFlowHub3`
- Control plane branch: `master`
- Root archive path: `D:\project\MyFlowHub3`
- Stage: `4`
- Participating repos:
  - `MyFlowHub-Proto`
  - `MyFlowHub-Server`
  - `MyFlowHub-SubProto`
  - `MyFlowHub-Win`
- Feature worktrees:
  - `D:\project\MyFlowHub3\worktrees\feat-proto-permit-list`
  - `D:\project\MyFlowHub3\worktrees\feat-server-permit-list`
  - `D:\project\MyFlowHub3\worktrees\feat-subproto-permit-list`
  - `D:\project\MyFlowHub3\worktrees\feat-win-permit-list`

## Docs Routing
- 使用 `$m-docs` 校验本轮 root 归档和索引落点。
- 结论：
  - Requirements impact: `updated`
  - Specs impact: `updated`
  - Related requirements:
    - `repo/MyFlowHub-Win/docs/requirements/authority-admin-console.md`
  - Related specs:
    - `repo/MyFlowHub-Proto/docs/protocol_map.md`
    - `repo/MyFlowHub-Server/docs/specs/auth.md`
    - `repo/MyFlowHub-Server/docs/specs/protocol_map.md`
    - `repo/MyFlowHub-Win/docs/specs/authority-admin-console.md`
  - Related lessons: `none`
- Canonical destination:
  - 稳定 wire 与 protocol map：`MyFlowHub-Proto`
  - 稳定 auth spec / protocol 壳：`MyFlowHub-Server`
  - 运行时行为：`MyFlowHub-SubProto/auth`
  - Win 用户交互与本地 requirements / specs：`MyFlowHub-Win`
  - 全局 workflow 归档：`docs/change/2026-03-28_auth-permit-list.md`
  - 全局 plan 归档：`docs/plan/plan_archive_2026-03-28_auth-permit-list.md`

## Stage 1 - Requirements Analysis
- 必须：
  - 定义 `list_register_permits / list_register_permits_resp` 稳定协议
  - permit list 只返回当前活动 permit
  - Win 页面顶部提供 `Refresh` 与 `New Permit`
  - permit 行内支持 `Revoke`
  - 页面初次加载和手动刷新必须有可见 loading 提示
  - issue / revoke 成功后回到真实列表，而不是只更新前端 latest 状态
- 不做：
  - 不补已消费 / 已撤销 / 已过期 permit 历史查询
  - 不支持编辑既有 permit
  - 不新增独立 `auth.permit.list` 权限
- 验收标准：
  - 四仓对齐 permit list 契约
  - Win 准入许可页只围绕真实活动列表组织交互
  - 旧 binding 缺失 `perms` 时不会造成 list 权限误判

## Stage 2 - Architecture Design
- 总体方案：
  - `MyFlowHub-Proto` 定义 canonical wire action / payload
  - `MyFlowHub-Server` 同步兼容壳与稳定 specs
  - `MyFlowHub-SubProto/auth` 基于 `registerPermits` 活动态提供 list，实现过期清理、过滤、排序、分页和权限判断
  - `MyFlowHub-Win` 新增 typed 服务封装，把页面改成真实 permit 列表
- 数据 / 调用流：
  1. Win 页面 ready 后解析 `authorityId`
  2. `permitIssuance` store 调用 `ListRegisterPermits`
  3. auth runtime 清理过期 permit 后返回当前活动列表
  4. issue / revoke 成功后页面统一重新加载列表
- 关键边界：
  - 所谓后端就是 auth 协议 + auth runtime，不读取本地持久化文件拼装列表
  - list 权限沿用 `auth.permit.issue` 或 `auth.permit.revoke`
  - 旧 whitelist 绑定若缺失 `perms`，按当前角色配置回填

## Participating Repos
- `MyFlowHub-Proto`
  - Worktree: `D:\project\MyFlowHub3\worktrees\feat-proto-permit-list`
  - Branch: `feat/proto-permit-list`
  - Result commit: `7a4b09a`
- `MyFlowHub-Server`
  - Worktree: `D:\project\MyFlowHub3\worktrees\feat-server-permit-list`
  - Branch: `feat/server-permit-list`
  - Result commit: `ef5de15`
- `MyFlowHub-SubProto`
  - Worktree: `D:\project\MyFlowHub3\worktrees\feat-subproto-permit-list`
  - Branch: `feat/subproto-permit-list`
  - Result commit: `ddd1a7b`
- `MyFlowHub-Win`
  - Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-permit-list`
  - Branch: `feat/win-permit-list`
  - Result commit: `8d385db`

## Task Checklist
- [x] `PROTO-PERMIT-1` 定义 permit list canonical wire
- [x] `SERVER-PERMIT-1` 更新 Server 兼容壳和 stable specs
- [x] `SUBPROTO-PERMIT-1` 实现 auth runtime permit list 与测试
- [x] `WIN-PERMIT-1` 补 Win typed 服务封装
- [x] `WIN-PERMIT-2` 重构准入许可页为活动 permit 列表
- [x] `VALIDATE-PERMIT-1` 完成跨 repo 验证
- [x] `REVIEW-PERMIT-1` 完成 3.3 review
- [x] `ARCHIVE-PERMIT-1` 完成各 repo 归档
- [x] `MERGE-PERMIT-1` 合并回各 repo 主线
- [x] `CLEANUP-PERMIT-1` 清理结束的 worktree / feature branch

## Validation Results
- `MyFlowHub-Proto`
  - `GOWORK=off go test ./... -count=1 -p 1`
  - `GOWORK=off go run ./cmd/protocolmapgen -check -out docs/protocol_map.md`
  - 结果：通过
- `MyFlowHub-Server`
  - 临时 `go.work` + `go test ./protocol/... -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-SubProto/auth`
  - 临时 `go.work` + `go test ./... -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-Win`
  - 临时 `go.work` + `go test ./internal/services/... -count=1 -p 1`
  - `npm test -- PermitIssuance authority_admin`
  - `$env:GOWORK='off'; wails generate module`
  - `npm run build`
  - 结果：通过

## Merge Results
- `repo/MyFlowHub-Proto/main`
  - fast-forward 到 `7a4b09a` (`feat: 补齐 auth permit 列表协议`)
- `repo/MyFlowHub-Server/main`
  - fast-forward 到 `ef5de15` (`docs: 同步 auth permit 列表契约`)
- `repo/MyFlowHub-SubProto/main`
  - fast-forward 到 `ddd1a7b` (`feat: 支持列出活动注册许可`)
- `repo/MyFlowHub-Win/main`
  - fast-forward 到 `8d385db` (`feat: 将准入许可改为活动列表`)
- 注意：
  - `MyFlowHub-Server`、`MyFlowHub-Proto`、`MyFlowHub-SubProto` 主线仍有用户侧无关脏文件，本轮未触碰

## Worktree Cleanup
- 已移除并 prune：
  - `D:\project\MyFlowHub3\worktrees\feat-proto-permit-list`
  - `D:\project\MyFlowHub3\worktrees\feat-server-permit-list`
  - `D:\project\MyFlowHub3\worktrees\feat-subproto-permit-list`
  - `D:\project\MyFlowHub3\worktrees\feat-win-permit-list`
- 已删除本地 feature branch：
  - `feat/proto-permit-list`
  - `feat/server-permit-list`
  - `feat/subproto-permit-list`
  - `feat/win-permit-list`

## Rollback
- 代码层：
  - 按 repo 分别 revert Proto wire、Server 壳 / specs、SubProto runtime、Win 服务层 / UI 变更
- 文档层：
  - 回退本次新增的 root change / plan archive，以及各 repo 的 `docs/change/*`
- 清理层：
  - 已删除 worktree 只能通过重新创建 worktree 恢复，不做 silent rollback

## Notes
- 本轮未使用子 agent。
- 当前 workflow 已完成 root 归档、主线合并和 worktree cleanup。
