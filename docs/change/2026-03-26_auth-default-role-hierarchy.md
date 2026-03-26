# 2026-03-26 Auth 默认角色分层与 Bootstrap 默认角色调整

## 变更背景 / 目标
- 现有 auth 默认值存在两个问题：
  - `MyFlowHub-Core/config` 默认 `auth.role_perms=""`，导致开箱配置下 `admin` 不是稳定可用角色。
  - `bootstrap first-register` 默认角色是 `admin`，但在未显式配置 `auth.role_perms` 的场景下可能直接触发 `unknown role` 初始化失败。
- 本次目标是把默认角色体系收敛成开箱即用的三层模型：
  - `superadmin`：拥有全部权限
  - `admin`：拥有当前运维管理能力集合
  - `node`：保持现有普通工作节点能力
- 同时将 cold start bootstrap 的默认授予角色从 `admin` 调整为 `superadmin`。

## 具体变更内容
### MyFlowHub-Core
- 更新 [config.go](../../repo/MyFlowHub-Core/config/config.go)
  - 新增导出常量：
    - `DefaultAuthRolePerms`
    - `DefaultAuthBootstrapFirstRegisterRole`
  - `NewMap()` 默认值改为：
    - `auth.role_perms=superadmin:*;admin:file.read,file.write,flow.set,flow.delete,exec.call,exec.cap.query,exec.cap.sync,var.private_set,var.revoke,var.subscribe,auth.revoke,auth.pending.list,auth.register.approve,auth.register.reject,auth.permit.issue,auth.permit.revoke;node:file.read,file.write,flow.set,exec.call,exec.cap.query,exec.cap.sync`
    - `auth.bootstrap.first_register.role=superadmin`
- 新增 [config_test.go](../../repo/MyFlowHub-Core/config/config_test.go)
  - 锁定默认角色层级和 bootstrap 默认角色

### MyFlowHub-SubProto/auth
- 更新 [bootstrap_first_register.go](../../repo/MyFlowHub-SubProto/auth/bootstrap_first_register.go)
  - bootstrap 内部默认角色不再硬编码 `admin`，改为引用 Core 默认常量
- 更新 [bootstrap_first_register_test.go](../../repo/MyFlowHub-SubProto/auth/bootstrap_first_register_test.go)
  - 默认 bootstrap 成功角色期望改为 `superadmin`
  - 测试 helper 不再手工覆盖 `auth.role_perms` / bootstrap role，直接吃默认值

### MyFlowHub-Server
- 更新 [options.go](../../repo/MyFlowHub-Server/hubruntime/options.go)
  - `DefaultOptions().AuthRolePerms` 改为复用 Core 默认角色层级
- 新增 [options_test.go](../../repo/MyFlowHub-Server/hubruntime/options_test.go)
  - 锁定 runtime 默认角色层级
- 更新 [auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md)
  - 明确默认三层角色：
    - `superadmin:*`
    - `admin:file.read,file.write,flow.set,flow.delete,exec.call,exec.cap.query,exec.cap.sync,var.private_set,var.revoke,var.subscribe,auth.revoke,auth.pending.list,auth.register.approve,auth.register.reject,auth.permit.issue,auth.permit.revoke`
    - `node:file.read,file.write,flow.set,exec.call,exec.cap.query,exec.cap.sync`
  - 明确 `auth.bootstrap.first_register.role` 默认值改为 `superadmin`

## Requirements impact
- none

## Specs impact
- updated

## Lessons impact
- none

## Related requirements
- [auth-controlled-admission.md](../requirements/auth-controlled-admission.md)

## Related specs
- [auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md)

## Related lessons
- none

## 对应 plan.md 任务映射
- `CORE-1`
- `CORE-2`
- `CORE-3`
- `AUTH-1`
- `AUTH-2`
- `SRV-1`
- `SRV-2`
- `SRV-DOC-1`
- `TEST-1`
- `TEST-2`
- `TEST-3`
- `REV-1`
- `ARC-1`

## 经验 / 教训摘要
- 默认角色层级一旦跨 Core / SubProto / Server 三处出现重复字面量，就会很容易漂移；将稳定默认值抽成 Core 导出常量更稳。
- `bootstrap` 默认角色不能只看“语义上应该是谁”，还要看“开箱配置下这个角色是否真实存在”，否则 fail-fast 会把冷启动体验打断。
- 当前权限系统没有角色继承，因此 `admin` 的默认权限必须完整平铺，不能假设它自动继承 `node`。

## 可复用排查线索
- 症状：
  - 开启审批和 bootstrap 后，auth handler 启动时报 `unknown role`
  - 默认部署下首个 bootstrap 节点没有拿到预期权限
  - Core / Server 默认 `role_perms` 行为不一致
- 触发条件：
  - `auth.bootstrap.first_register.role` 使用默认值
  - 部署侧没有显式覆盖 `auth.role_perms`
  - Core / Server / SubProto 使用了不同来源的默认值
- 关键词：
  - `unknown role`
  - `bootstrap first register`
  - `superadmin`
  - `auth.role_perms`
- 快速检查：
  - 看 `config/runtime_config.json` 里是否显式覆盖了 `auth.role_perms`
  - 看 `auth.bootstrap.first_register.role` 当前 effective value
  - 看 `DefaultOptions().AuthRolePerms` 和 `config.NewMap(nil)` 是否一致

## 关键设计决策与权衡
- 采用三层默认角色而不是继续沿用 `admin/node`
  - 优点：把“根权限”和“日常管理权限”拆开，冷启动语义更清晰
  - 代价：默认配置文案更长，`auth.role_perms` 字符串更复杂
- 采用 Core 导出常量而不是在 Server / SubProto 各自重复维护默认值
  - 优点：减少跨仓默认值漂移
  - 代价：Server / SubProto 对 Core 默认常量形成了更显式的依赖
- 不引入角色继承
  - 优点：保持当前权限解析模型不变，风险最小
  - 代价：`admin` 权限必须显式列全

## 测试与验证方式 / 结果
- `GOWORK=D:\project\MyFlowHub3\worktrees\feat-auth-default-role-hierarchy\control\go.work go test ./config -count=1`
  - 仓库：`MyFlowHub-Core`
  - 结果：通过
- `GOWORK=D:\project\MyFlowHub3\worktrees\feat-auth-default-role-hierarchy\control\go.work go test ./auth/... -run FirstRegisterBootstrap -count=1 -p 1`
  - 仓库：`MyFlowHub-SubProto`
  - 结果：通过
- `GOWORK=D:\project\MyFlowHub3\worktrees\feat-auth-default-role-hierarchy\control\go.work go test ./auth/... -count=1 -p 1`
  - 仓库：`MyFlowHub-SubProto`
  - 结果：通过
- `GOWORK=D:\project\MyFlowHub3\worktrees\feat-auth-default-role-hierarchy\control\go.work go test ./hubruntime/... -count=1`
  - 仓库：`MyFlowHub-Server`
  - 结果：通过
- `GOWORK=D:\project\MyFlowHub3\worktrees\feat-auth-default-role-hierarchy\control\go.work go test ./tests -run TestLoginHandler -count=1 -p 1`
  - 仓库：`MyFlowHub-Server`
  - 结果：通过

## 潜在影响与回滚方案
- 潜在影响：
  - 任何依赖 `config.NewMap(nil)` 默认权限为空的代码或测试，现在会拿到新的 `node/admin/superadmin` 默认映射
  - `bootstrap` 默认角色从 `admin` 变成 `superadmin` 后，未显式覆盖的冷启动首个注册者权限会上升
- 回滚方案：
  - 回滚 `MyFlowHub-Core/config` 默认值常量与测试
  - 回滚 `MyFlowHub-SubProto/auth` bootstrap 默认角色常量和测试
  - 回滚 `MyFlowHub-Server/hubruntime/options.go` 与 `docs/specs/auth.md`

## 子Agent执行轨迹
- 无子 agent
