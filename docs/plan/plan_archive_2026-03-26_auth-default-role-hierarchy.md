# Plan Archive - Auth 默认角色分层与 Bootstrap 默认角色调整

## Goal
- 将 auth 的默认角色模型收敛为 `superadmin / admin / node` 三层。
- 让开箱即用配置直接可用：`superadmin` 拥有全部权限，`admin` 拥有当前运维管理能力集合，`node` 保持现有普通工作节点能力。
- 将 cold start first-register bootstrap 的默认授予角色从 `admin` 调整为 `superadmin`。

## Current State
- `MyFlowHub-Core/config` 当前默认：
  - `auth.default_role=node`
  - `auth.default_perms=""`
  - `auth.role_perms=""`
  - `auth.bootstrap.first_register.role=admin`
- `MyFlowHub-Server/hubruntime` 当前运行时默认 `auth.role_perms` 只有 `node:file.read,file.write,flow.set,exec.call,exec.cap.query,exec.cap.sync`。
- `MyFlowHub-SubProto/auth` 当前 bootstrap handler 的内部默认角色常量为 `admin`。
- 现状会导致：
  - 开箱配置下 `bootstrap role=admin` 不是稳定可用默认值
  - 角色层级不清晰
  - `admin` 与 `node` 的职责边界需要收敛成稳定默认策略

## Workflow Info
- Control repo: `MyFlowHub3`
- Branch: `feat/auth-default-role-hierarchy`
- Base: `master`
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-auth-default-role-hierarchy\control`
- Stage: `4`

## Docs Routing
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- 结论：
  - Requirements impact: `none`
  - Specs impact: `updated`
  - Related requirements: `docs/requirements/auth-controlled-admission.md`
  - Related specs: `repo/MyFlowHub-Server/docs/specs/auth.md`
  - Related lessons: `none`
- Canonical destination:
  - 稳定默认行为与配置语义更新到 `repo/MyFlowHub-Server/docs/specs/auth.md`
  - workflow 执行结果归档到 `docs/change/`
  - 本 worktree 的计划正文后续归档到 `docs/plan/`

## Stage 1 - Requirements Analysis
- 目标：
  - 提供稳定、开箱即用的默认角色层级，并让 bootstrap 冷启动默认落到最高权限角色。
- 范围：
  - 必须：
    - 新增稳定默认角色层级：`superadmin`、`admin`、`node`
    - `superadmin` 默认拥有全部权限
    - `admin` 默认拥有现状中的运维管理能力
    - `node` 默认保持现有普通工作节点权限集合
    - bootstrap 默认角色改为 `superadmin`
    - Core / SubProto / Server 运行时默认值保持一致
    - 更新相关 spec / change / plan 归档
  - 可选：
    - 补充针对默认值的锁定测试
  - 不做：
    - 不引入角色继承机制
    - 不扩展新的权限节点
    - 不修改用户显式配置覆盖逻辑
- 使用场景：
  - 本地 authority 冷启动且启用审批时，首个注册者默认直接成为 `superadmin`
  - 默认开箱部署下，`admin` 可进行审批、permit、变量管理、flow 管理
  - 默认普通节点仍可进行 file/flow/exec 常规工作，不自动拥有审批或全局管理权限
- 功能需求：
  - `auth.role_perms` 需要具备稳定默认值，而不是空映射
  - `auth.bootstrap.first_register.role` 默认改为 `superadmin`
  - 角色默认值必须在当前权限配置中真实存在
- 非功能需求：
  - 保持显式配置优先，不破坏已有 override
  - 默认值需跨 Core / Server / SubProto 对齐
  - 仅做最小必要改动
- 输入输出：
  - 输入：未显式配置或部分配置的 auth 角色相关配置
  - 输出：一致的默认角色/权限语义与 bootstrap 默认角色
- 边界异常：
  - 仍然不支持角色继承，`admin` 权限必须平铺列出
  - 用户显式把 `auth.role_perms` 配为空时，保留当前“由显式配置负责”的语义
- 验收标准：
  - Core/Server 默认值一致
  - SubProto bootstrap 默认角色与 Core 默认值一致
  - 对应测试能够锁定默认值
  - spec 与归档文档反映最新默认行为
- 风险：
  - 默认 `admin` 权限集合过大可能抬高默认暴露面
  - 默认 `node` 权限集合如果偏离现状，会造成行为回归

## Stage 2 - Architecture Design
- 总体方案：
  - 直接在默认值层收敛角色体系，不引入角色继承或额外解析逻辑。
  - 这样改动面最小，回滚成本最低，也不会影响现有配置覆盖链路。
- 模块职责：
  - `MyFlowHub-Core/config`
    - 稳定基础默认值定义
  - `MyFlowHub-SubProto/auth`
    - bootstrap handler 默认角色常量与测试
  - `MyFlowHub-Server/hubruntime`
    - 运行时默认 options 对齐基础默认值
  - `MyFlowHub-Server/docs/specs/auth.md`
    - 稳定说明默认层级、bootstrap 默认角色与推荐配置
- 数据 / 调用流：
  - 启动时：
    - Server `DefaultOptions()` 生成默认 config
    - layered config 合并 persistent / explicit / runtime
    - Core `config.NewMap()` 补齐缺省键
    - Auth handler 读取 bootstrap role 与 role_perms
  - bootstrap 命中时：
    - 直接以默认 `superadmin` 或显式覆盖角色准入
- 接口草案：
  - 不新增新接口
  - 仅修改默认配置值和文档口径
- 错误与安全：
  - 保持现有 `unknown role` fail-fast
  - `superadmin=*` 只作为 bootstrap 默认角色与可选默认层级，不放宽普通 `node`
- 性能与测试策略：
  - 无热路径算法变化
  - 增补默认值锁定测试
  - 跑 Core config、SubProto auth bootstrap、Server hubruntime 测试
- 可扩展性设计点：
  - 角色层级仍由 `auth.role_perms` 纯配置化表达
  - 未来若增加权限点，只需决定是否显式加入 `admin`

## Participating Repos
- Control
  - Worktree: `D:\project\MyFlowHub3\worktrees\feat-auth-default-role-hierarchy\control`
  - Plan: `D:\project\MyFlowHub3\worktrees\feat-auth-default-role-hierarchy\control\todo.md`
  - Write set: `docs/change/*`, `docs/change/README.md`, `docs/plan/*`, `plan.md`（仅在 workflow end 时）
- MyFlowHub-Core
  - Worktree: `D:\project\MyFlowHub3\worktrees\feat-auth-default-role-hierarchy\MyFlowHub-Core`
  - Plan: `D:\project\MyFlowHub3\worktrees\feat-auth-default-role-hierarchy\MyFlowHub-Core\todo.md`
  - Write set: `config/config.go`, related tests
- MyFlowHub-SubProto
  - Worktree: `D:\project\MyFlowHub3\worktrees\feat-auth-default-role-hierarchy\MyFlowHub-SubProto`
  - Plan: `D:\project\MyFlowHub3\worktrees\feat-auth-default-role-hierarchy\MyFlowHub-SubProto\todo.md`
  - Write set: `auth/bootstrap_first_register.go`, `auth/bootstrap_first_register_test.go`, related tests if needed
- MyFlowHub-Server
  - Worktree: `D:\project\MyFlowHub3\worktrees\feat-auth-default-role-hierarchy\MyFlowHub-Server`
  - Plan: `D:\project\MyFlowHub3\worktrees\feat-auth-default-role-hierarchy\MyFlowHub-Server\todo.md`
  - Write set: `hubruntime/options.go`, `hubruntime/*_test.go`, `docs/specs/auth.md`

## Task Checklist
- [x] `CORE-1` 调整 Core 默认 `auth.role_perms`
- [x] `CORE-2` 调整 Core 默认 bootstrap role 为 `superadmin`
- [x] `CORE-3` 增补或更新 Core 默认值测试
- [x] `AUTH-1` 调整 SubProto bootstrap 默认角色常量
- [x] `AUTH-2` 更新 bootstrap 相关测试期望与 role 配置
- [x] `SRV-1` 调整 hubruntime 默认 `auth.role_perms`
- [x] `SRV-2` 增补或更新 hubruntime 默认值测试
- [x] `SRV-DOC-1` 更新 auth spec，明确默认三层角色与 bootstrap 默认角色
- [x] `TEST-1` 跑 Core 配置测试
- [x] `TEST-2` 跑 SubProto auth bootstrap 测试
- [x] `TEST-3` 跑 Server hubruntime 测试
- [x] `REV-1` 完成 3.3 review
- [x] `ARC-1` 写 docs/change 归档并更新索引

## Repo Tasks
- `CORE-1`
  - Goal: 在基础配置层提供稳定默认 `role_perms`
  - Files: `config/config.go`
  - Acceptance: 默认 `superadmin/admin/node` 三层映射可直接使用
  - Tests: `go test ./config -count=1`
  - Rollback: 回滚默认值字符串
- `AUTH-1`
  - Goal: bootstrap 默认角色与 Core 默认值保持一致
  - Files: `auth/bootstrap_first_register.go`, `auth/bootstrap_first_register_test.go`
  - Acceptance: bootstrap 成功测试默认角色变为 `superadmin`
  - Tests: `go test ./auth/... -run FirstRegisterBootstrap -count=1 -p 1`
  - Rollback: 回滚默认常量与测试期望
- `SRV-1`
  - Goal: Server runtime 默认角色权限与基础默认值保持一致
  - Files: `hubruntime/options.go`, related tests
  - Acceptance: `DefaultOptions()` 输出三层默认角色映射
  - Tests: `go test ./hubruntime/... -count=1`
  - Rollback: 回滚默认 options 值与测试
- `SRV-DOC-1`
  - Goal: 记录稳定默认角色体系与 bootstrap 默认角色
  - Files: `docs/specs/auth.md`
  - Acceptance: 文档与实现默认值一致
  - Tests: docs consistency review
  - Rollback: 回滚 spec 文案

## Parallelism Assessment
- 本轮不派发子 agent。
- 原因：
  - 改动面小，但跨仓默认值与文档强耦合
  - 主线关键路径依赖统一口径，不值得拆并行

## Notes
- 子 agent：无
- `3.3` Review：
  - 需求覆盖：通过
  - 架构合理性：通过
  - 性能风险：通过
  - 可读性与一致性：通过
  - 可扩展性与配置化：通过
  - 稳定性与安全：通过
  - 测试覆盖情况：通过
  - 子Agent治理与审计：通过
- 验证结果：
  - `MyFlowHub-Core`: `go test ./config -count=1`
  - `MyFlowHub-SubProto`: `go test ./auth/... -run FirstRegisterBootstrap -count=1 -p 1`
  - `MyFlowHub-SubProto`: `go test ./auth/... -count=1 -p 1`
  - `MyFlowHub-Server`: `go test ./hubruntime/... -count=1`
  - `MyFlowHub-Server`: `go test ./tests -run TestLoginHandler -count=1 -p 1`
- 阻塞：否
- Stage：`4`
