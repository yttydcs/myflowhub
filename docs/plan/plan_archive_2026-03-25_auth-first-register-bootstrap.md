# Plan - Auth 首个注册 Bootstrap

## Goal
- 为审批模式增加一次性、受控的首个注册 bootstrap：命中显式 `device_id` 的首个注册者可在单个 `epoch` 槽位内直接获得指定角色，消费后恢复正常审批流。

## Related Requirements
- [auth-controlled-admission.md](../requirements/auth-controlled-admission.md)

## Related Specs
- [auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md)

## Related Lessons
- none

## Requirements Impact
- `updated`

## Specs Impact
- `updated`

## Lessons Impact
- `none`

## Workflow Information
- Control plane repo: `MyFlowHub3`
- Control plane branch: `feat/auth-bootstrap-first-register`
- Control plane worktree: `d:\project\MyFlowHub3\worktrees\feat-auth-bootstrap-first-register\control`
- Base: `master`
- Participating repos:
  - `MyFlowHub-Core`
  - `MyFlowHub-SubProto`
  - `MyFlowHub-Server`

## Confirmed Scope
- 必须：
  - bootstrap 只允许在 `local authority` 生效。
  - bootstrap 必须显式绑定 `device_id`，可选再绑定 `pubkey`。
  - bootstrap 必须按 `epoch` 一次性消费，并把消费状态持久化。
  - 非命中设备继续走既有 `pending -> approve -> retry register`。
  - 非法配置或与当前拓扑冲突时必须 fail-fast。
- 不做：
  - 不把“普通首个 register”自动提升为管理员。
  - 不新增独立 bootstrap action / protocol。
  - 本轮不新增 env / flag，继续通过 `runtime_config.json` 承载配置。

## Architecture Summary
- `register` 链路顺序收敛为：
  - existing rebind
  - permit
  - approved retry
  - first-register bootstrap
  - pending approval
  - open register fallback
- `MyFlowHub-Core` 负责 bootstrap 配置键与默认值：
  - `auth.bootstrap.first_register.enabled`
  - `auth.bootstrap.first_register.role`
  - `auth.bootstrap.first_register.device_id`
  - `auth.bootstrap.first_register.pubkey`
  - `auth.bootstrap.first_register.epoch`
- `MyFlowHub-SubProto/auth` 负责 bootstrap 配置加载、合法性校验、消费判定与状态持久化。
- bootstrap 消费状态写入 `trusted_nodes.json.meta.first_register_bootstrap`；重启恢复时同步抬高 `maxNode`，避免重用已预留的 `node_id`。
- `MyFlowHub-Server` 仅更新稳定 spec，不扩张 runtime CLI / env 入口，保持最小改动面。

## Task Checklist
- [x] `DOC-REQ-1` 更新 requirement，定义 bootstrap exception 的边界
- [x] `CORE-1` 新增 bootstrap config key 常量与默认值
- [x] `AUTH-1` 新增 bootstrap config/state 结构、加载与持久化
- [x] `AUTH-2` 在 register 链路接入 bootstrap 判定与消费
- [x] `AUTH-3` 对非法 bootstrap 配置做 fail-fast 校验
- [x] `AUTH-4` 补充单测：成功、消费后失效、pubkey mismatch、epoch 重开、非法配置
- [x] `SRV-DOC-1` 更新 auth spec，写清 bootstrap 配置和 register 顺序
- [x] `SRV-OPT-1` 评估并决定是否补 `hubruntime/cmd` 的 env/flag 暴露
- [x] `TEST-0` 对齐 workflow-local `go.work` 到当前 worktree 路径
- [x] `TEST-1` 跑 Core / SubProto / Server 相关测试
- [x] `REV-1` 完成 3.3 code review checklist
- [x] `ARC-1` 写 `docs/change` 归档并更新索引

## Code Review
- 需求覆盖：通过
  - bootstrap exception 只覆盖冷启动审批缺口，没有放宽普通审批主语义。
- 架构合理性：通过
  - 继续复用 `register` 统一入口，不引入平行协议；bootstrap 仅作为受控例外插入既有判定链路。
- 性能风险：通过
  - 新增逻辑只涉及配置读取、本地状态判定和持久化，不增加热路径级别的额外扫描。
- 可读性与一致性：通过
  - 配置键、spec、handler 顺序和测试命名保持同一语义。
- 可扩展性与配置化：通过
  - `role`、`device_id`、`pubkey`、`epoch` 保持可配置；未来若需要显式重开，仅提升 `epoch`。
- 稳定性与安全：通过
  - `unknown role`、`disable_persist`、存在 `parent` / `authority.node_id`、非法 `pubkey`、非法 `epoch` 均显式失败。
- 测试覆盖情况：通过
  - 成功路径、消费后失效、`pubkey` mismatch、epoch 重开和持久化恢复都已覆盖。
- 子Agent治理与审计：通过
  - 无子 agent。

## Validation
- `GOWORK=d:\project\MyFlowHub3\worktrees\feat-auth-bootstrap-first-register\control\go.work go test github.com/yttydcs/myflowhub-core/config -count=1`
- `GOWORK=d:\project\MyFlowHub3\worktrees\feat-auth-bootstrap-first-register\control\go.work go test github.com/yttydcs/myflowhub-subproto/auth/... -count=1 -p 1`
- `GOWORK=d:\project\MyFlowHub3\worktrees\feat-auth-bootstrap-first-register\control\go.work go test github.com/yttydcs/myflowhub-server/hubruntime/... -count=1 -p 1`
- `GOWORK=d:\project\MyFlowHub3\worktrees\feat-auth-bootstrap-first-register\control\go.work go test github.com/yttydcs/myflowhub-server/tests -run TestLoginHandler -count=1 -p 1`
- 结果：
  - 全部通过

## Merge Results
- `MyFlowHub-Core/master`
  - 已合入 `feat/auth-bootstrap-first-register`
  - 关联提交：`83b4f00 feat: 增加 auth 首个注册 bootstrap 配置键`
- `MyFlowHub-SubProto/main`
  - 已合入 `feat/auth-bootstrap-first-register`
  - 关联提交：`673ef59 feat: 增加 auth 首个注册 bootstrap`
- `MyFlowHub-Server/main`
  - 已合入 `feat/auth-bootstrap-first-register`
  - 关联提交：`2a643d9 docs: 更新 auth 首个注册 bootstrap 规范`
- `MyFlowHub3/master`
  - 已合入归档提交：`c7bec36 docs: 归档 auth 首个注册 bootstrap`
  - workflow end 阶段补齐根级 `plan.md` 与 `docs/plan/` 入口索引

## Worktree Cleanup
- 已在 workflow end 阶段移除并 prune 以下 worktree：
  - `d:\project\MyFlowHub3\worktrees\feat-auth-bootstrap-first-register\control`
  - `d:\project\MyFlowHub3\worktrees\feat-auth-bootstrap-first-register\MyFlowHub-Core`
  - `d:\project\MyFlowHub3\worktrees\feat-auth-bootstrap-first-register\MyFlowHub-SubProto`
  - `d:\project\MyFlowHub3\worktrees\feat-auth-bootstrap-first-register\MyFlowHub-Server`

## Rollback
- 代码层：
  - 回滚 `MyFlowHub-Core/config/config.go` 的 bootstrap 配置键和默认值。
  - 回滚 `MyFlowHub-SubProto/auth` 的 bootstrap state、register 判定和测试。
  - 回滚 `MyFlowHub-Server/docs/specs/auth.md` 的 bootstrap 规范更新。
- 文档层：
  - 回滚 `docs/requirements/auth-controlled-admission.md`、`docs/change/2026-03-25_auth-first-register-bootstrap.md` 以及本计划归档。
