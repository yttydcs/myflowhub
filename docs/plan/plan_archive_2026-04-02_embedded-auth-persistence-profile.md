# Plan - Embedded Auth Persistence Profile

## Workflow Information
- Repo: `MyFlowHub-EmbeddedSDK`
- Branch: `feat/embedded-auth-persistence-profile`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-embedded-auth-persistence-profile`
- Current Stage: `4 (Archived, waiting for workflow end confirmation)`

## Stage Records

### Initialization
- 使用 `$m-autoflow` 执行本轮实现工作。
- workspace `guide.md` 已读取并确认：
  - worktree 必须在 `D:\project\MyFlowHub3\worktrees\`
  - Python 验证必须走 `cmd /c "call C:\ProgramData\anaconda3\Scripts\activate.bat micropython && ..."`
  - commit 信息使用中文
- control-plane repo:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-EmbeddedSDK`
- active implementation worktree:
  - `D:\project\MyFlowHub3\worktrees\feat-embedded-auth-persistence-profile`
- active branch:
  - `feat/embedded-auth-persistence-profile`
- participating modules:
  - `micropython/myflowhub/async_edge_local.py`
  - `micropython/myflowhub/__init__.py` if new public helper is exported
  - `micropython/README.md`
  - `micropython/tests/test_async_edge.py`
  - `micropython/tests/test_edge_runtime.py`
  - `docs/specs/micropython-edge-runtime.md`
  - `docs/specs/micropython-edge-server-alignment.md`
  - `docs/change/`
  - `docs/lessons/` if this round出现可复用排查经验

### Stage 1 - Requirements Analysis

#### 目标

在已交付的 in-memory local authority skeleton 之上，补齐下一层最小但真实可用的 `auth persistence/profile` 能力：

- 给 built-in local auth 提供一个可直接落盘恢复的 persistence backend
- 为 `pending / approved / permit` 增加 TTL/expiry 处理
- 给 local authority admin/query 增加可选的本地 authz skeleton
- 保持这一切仍然是显式 opt-in profile，不把 heavy auth 默认压进轻量 SDK 基线

#### 范围

- 必须：
  - 在 store adapter 后补一个真实的持久化 backend，允许调用方直接传给 `enable_builtin_local_auth(next_node_id, state=...)`
  - 支持保存和恢复：
    - `devices`
    - `node_index`
    - `child_devices`
    - `next_node_id`
    - `pending_registers`
    - `pending_by_device`
    - `approved_registers`
    - `register_permits`
    - local authority policy/config state
  - `pending / approved / permit` 支持 expiry/ttl
  - local authority admin/query 支持可选的 authz skeleton
  - host-side async/sync regression 覆盖：
    - persistence restore
    - ttl expiry
    - admin authz allow / deny
  - specs / README / archive 对齐
- 可选：
  - backend 暴露最小 hook 扩展点，便于 future 自定义 time/authz
  - local auth response 补充可选 `perms`
- 不做：
  - full ES256 local auth
  - server 级 persisted trust/whitelist 完整对齐
  - semi-central authority lease
  - ancestor-chain / non-parent authority routing 扩展
  - `varstore/topicbus` full hub parity

#### 使用场景

1. 板子作为本地 authority，使用文件型 backend 保存 `pending / approved / permit / binding`，重启后恢复状态。
2. `require_approval=true` 时，child `register` 进入 `pending`，超过 TTL 后自动失效。
3. `approve_register()` 产生的预留身份有 TTL；child 若长期不 retry `register`，预留会过期而不是永久占位。
4. `issue_register_permit()` 若未显式给 `expires_at`，可按 profile 默认 TTL 生成；过期 permit 不再可消费。
5. local authority admin/query 若开启 authz skeleton，只允许具备本地 policy 权限的已登录 actor 执行。

#### 功能需求

- store adapter 继续支持：
  - `dict`
  - `get_state()/save_state(state)` backend
- 本轮新增的内建 backend 必须：
  - 使用稳定 JSON 结构落盘
  - 加载不存在文件时返回空 state
  - 保存或加载失败时显式抛错，不静默吞掉
- local auth state 需要增加 policy/config 结构，用于：
  - `pending_ttl_sec`
  - `approved_ttl_sec`
  - `permit_ttl_sec`
  - admin authz 开关
  - action -> permission / role skeleton
- `register` 流程必须先清理已过期的：
  - `pending`
  - `approved`
  - `permit`
- admin/query 流程必须在执行前：
  - 校验 actor 已登录
  - 若启用 authz skeleton，按本地 policy 判定 allow/deny
  - deny 时返回显式 `4403`
- 保持 server-aligned 语义：
  - `approve_register` 只预留身份，不直接完成 binding
  - permit 一次性消费
  - `list_register_permits` 只返回当前有效 permit

#### 非功能需求

- 保持 opt-in，不改变默认 edge/lightweight baseline
- 不把 persistence I/O 或 authz 逻辑塞进 router 主流程
- 过期清理使用常数级索引 + 局部扫描，不做重复 JSON 解码
- 兼容 host `CPython` 与板侧 `MicroPython`
- 失败路径显式，不吞错

#### 输入输出

- 输入：
  - `docs/requirements/embedded-sdk-edge-hub.md`
  - `docs/specs/micropython-edge-runtime.md`
  - `docs/specs/micropython-edge-server-alignment.md`
  - `repo/MyFlowHub-Server/docs/specs/auth.md`
  - `docs/lessons/micropython-edge-local-auth-nodeid-allocation.md`
  - `micropython/myflowhub/async_edge_local.py`
  - `micropython/tests/test_async_edge.py`
  - `micropython/tests/test_edge_runtime.py`
- 输出：
  - local auth persistence/profile 增量实现
  - regression tests
  - updated specs / README / change archive

#### 边界异常

- child 断连后恢复 state 时，旧 `child_id` 不能被当成可复用 live binding
- 过期记录被清理后，`pending_by_device` / `approved_registers` / `register_permits` 必须保持一致
- admin authz deny 不能伪装成 `authority unavailable`
- 文件 backend 缺路径、文件内容非法 JSON、state 非 dict 都要显式失败
- 不能把新 policy 变成强制默认，否则会打破既有 local demo

#### 验收标准

- 使用内建持久化 backend 时，状态可以保存并在新 runtime 实例中恢复
- pending/approved/permit 的 TTL 过期后不再被列出或消费
- 开启 authz skeleton 后：
  - 有权限 actor 可以执行 admin/query
  - 无权限 actor 收到 `4403`
- async/sync targeted tests 和 full discover 通过

#### 风险

- 若 backend 设计成 runtime 强耦合，会再次破坏 store adapter 边界
- 若过期清理不彻底，会留下 `pending_by_device` / `node_index` 脏状态
- 若 authz 默认开启，会让现有 demo/local tests 行为回归
- 若 role/perms 做得过重，会把本轮切片推成 mini full-auth

#### 问题清单

- 当前无阻塞

### Stage 2 - Architecture Design

#### 总体方案（含选型理由 / 备选对比）

- 方案 A（采用）：
  - 继续把 local authority profile 状态机留在 `async_edge_local.py`
  - 在 store adapter 后新增一个 JSON 文件 backend
  - policy 仍作为 local auth state 的一部分持久化
  - authz 采用“默认关闭 + 显式启用”的 skeleton
- 采用理由：
  - 最小改动面，不动 `AsyncEdgeRuntime` 的 authority/routing contract
  - persistence 与 authz 都挂在 local auth profile 层，不污染 edge core
  - state/policy 同步保存，恢复简单

- 方案 B：
  - 单独新增完整 auth profile 模块或 mini authority engine
  - 不采用：范围过大，本轮目标只是把 skeleton 变成真实 profile 增量

- 方案 C：
  - 只补 TTL/policy，不提供内建 backend
  - 不采用：仍然缺“开箱即用的真实 persistence backend”

#### 模块职责

- `micropython/myflowhub/async_edge_local.py`
  - local auth state normalization
  - store adapter optional hooks
  - JSON file backend
  - ttl/expiry pruning
  - local admin authz skeleton
- `micropython/myflowhub/__init__.py`
  - 若新增 public helper，则导出
- `micropython/tests/test_async_edge.py`
  - async persistence/ttl/authz regression
- `micropython/tests/test_edge_runtime.py`
  - sync parity regression
- `docs/specs/*.md`
  - local auth optional profile 新 contract
- `micropython/README.md`
  - 调用方式和 profile 边界说明

#### 数据 / 调用流

1. 调用方创建 local auth state 或文件 backend。
2. `enable_builtin_local_auth(next_node_id, state=...)` 挂载 built-in auth。
3. service 初始化时：
  - 读取 persisted state
  - 规范化 policy/defaults
  - 清理 runtime-only child binding 残留
4. admin/query/register 执行前：
  - prune expired pending/approved/permit
  - admin action 先做 authz
5. 状态修改后：
  - `save_state(state)` 落盘
6. 重启后：
  - 同一 backend 重新提供 state
  - runtime 恢复剩余有效 binding/pending/approved/permit/policy

#### 接口草案

- 维持：
  - `enable_builtin_local_auth(next_node_id, state=None)`
- 新增内建 backend：
  - `LocalAuthFileStoreBackend(path)`
    - `get_state()`
    - `save_state(state)`
- store backend 可选 hook：
  - `now_ms()` if provided
  - `authorize_admin_action(action, actor, actor_record, context, state)` if provided
- local policy state 草案：
  - `policy.require_admin_authz`
  - `policy.pending_ttl_sec`
  - `policy.approved_ttl_sec`
  - `policy.permit_ttl_sec`
  - `policy.role_perms`
  - `policy.action_perms`

#### 错误与安全

- 文件读写或 JSON decode 失败显式抛错
- admin authz deny 返回 `4403`
- 仅在显式 `require_admin_authz=true` 时启用权限判定
- 默认 role/perms skeleton 仅覆盖 auth admin 最小动作，不宣称 full server permission parity

#### 性能与测试策略

- 运行时仍以内存 dict 为主；持久化只在状态变更后写入
- prune 只扫描有限的 `pending / approved / permit` 集合
- async/sync regression 覆盖：
  - backend save/restore
  - pending expiry
  - approved expiry
  - permit expiry
  - authz allow / deny
- 最终验证：
  - targeted unittest
  - full discover
  - `git diff --check`

#### 可扩展性设计点

- backend 仍通过 store adapter 接入，future 可换成 flash/NVS
- policy 和 optional hook 允许 future 接更强的 authority authz / bootstrap
- authz 仍局限在 local authority profile，不改变 parent/authority forwarding contract

#### 问题清单

- 当前无阻塞

### Stage 3.1 - Planning

#### Docs Governance Routing Decision

- 使用 `$m-docs` 校验文档路由、requirements/specs 影响和 lessons 查询入口。
- Requirements impact: `none`
- Specs impact: `clarify`
- Related requirements:
  - `docs/requirements/embedded-sdk-edge-hub.md`
- Related specs:
  - `docs/specs/micropython-edge-runtime.md`
  - `docs/specs/micropython-edge-server-alignment.md`
  - `repo/MyFlowHub-Server/docs/specs/auth.md`
- Related lessons:
  - `docs/lessons/micropython-edge-local-auth-nodeid-allocation.md`
- canonical destination:
  - stable technical contract -> `docs/specs/`
  - workflow control -> current worktree root `plan.md`
  - completed result -> `docs/change/`
  - reusable troubleshooting -> `docs/lessons/` only if this round出现非显然调试路径

#### Executable Checklist

- [x] `T0` 初始化并完成 Stage 1 / 2
- [x] `T1` 在 local auth store adapter 后补真实 persistence backend 和 policy normalization
- [x] `T2` 补 TTL/expiry 流程与 local admin authz skeleton
- [x] `T3` 增加 async/sync regression 并完成验证
- [x] `T4` 更新 specs / README / change archive / lessons 判定

#### Task Details

##### T1 - Local Auth Persistence Backend
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-embedded-auth-persistence-profile`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-embedded-auth-persistence-profile\plan.md`
- Goal:
  - 提供一个可直接使用的本地 auth 持久化 backend，并完成 state/policy 规范化
- Files / Modules:
  - `micropython/myflowhub/async_edge_local.py`
  - `micropython/myflowhub/__init__.py` if exported
  - `micropython/README.md` if usage changes
- Acceptance:
  - 可通过 backend save/restore 恢复 auth local profile 状态
  - 仍保持 `dict` 与 custom backend 兼容
- Tests:
  - async persistence restore
  - sync persistence restore or surface parity
- Rollback:
  - 去掉新增 backend 和导出，仅保留旧 store adapter 壳子

##### T2 - TTL and Admin Authz Skeleton
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-embedded-auth-persistence-profile`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-embedded-auth-persistence-profile\plan.md`
- Goal:
  - 为 pending/approved/permit 加 TTL/expiry，并为 local admin/query 增加可选 authz skeleton
- Files / Modules:
  - `micropython/myflowhub/async_edge_local.py`
- Acceptance:
  - 过期记录不会继续列出或消费
  - authz 启用时 allow/deny 正确
- Tests:
  - pending expiry
  - approved expiry
  - permit expiry
  - board-side admin authz deny
  - child-side admin authz deny if relevant
- Rollback:
  - 回退 policy/authz/ttl 增量，恢复到无限期 in-memory skeleton

##### T3 - Regression and Validation
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-embedded-auth-persistence-profile`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-embedded-auth-persistence-profile\plan.md`
- Goal:
  - 锁住 persistence/profile 增量行为，不回归 async/sync facade
- Files / Modules:
  - `micropython/tests/test_async_edge.py`
  - `micropython/tests/test_edge_runtime.py`
- Acceptance:
  - targeted 和 full regressions 通过
- Tests:
  - `cmd /c "call C:\ProgramData\anaconda3\Scripts\activate.bat micropython && python -m unittest micropython.tests.test_async_edge micropython.tests.test_edge_runtime"`
  - `cmd /c "call C:\ProgramData\anaconda3\Scripts\activate.bat micropython && python -m unittest discover -s micropython/tests"`
  - `git diff --check`
- Rollback:
  - 移除新增测试，恢复到上一轮 local authority profile coverage

##### T4 - Specs / README / Archive
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-embedded-auth-persistence-profile`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-embedded-auth-persistence-profile\plan.md`
- Goal:
  - 把 persistence/profile 新 contract 对齐到 specs，并完成本轮 archive
- Files / Modules:
  - `docs/specs/micropython-edge-runtime.md`
  - `docs/specs/micropython-edge-server-alignment.md`
  - `micropython/README.md`
  - `docs/change/2026-04-02_embedded-auth-persistence-profile.md`
  - `docs/lessons/*` if needed
- Acceptance:
  - specs 与代码一致
  - archive 满足 `$m-autoflow` / `$m-docs` 要求
- Tests:
  - doc self-review
  - link sanity by file existence
- Rollback:
  - 回退 spec/README/archive 到本轮之前状态

#### Dependencies

- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
- 当前 main 已合并的 local authority profile skeleton
- conda env `micropython`

#### Risks and Notes

- 本轮不派发子 agent；主要写热点集中在 `async_edge_local.py`
- 若实现中发现 authz 需要稳定 requirements 变更，而非 specs clarify，必须回到 3.1 重写计划
- backend helper 若成为 public surface，必须同步维护 README 和导出

#### Parallelism Assessment

- 本轮不派发子 Agent
- 原因：
  - 持久化、TTL、authz 都落在同一状态机，写集高度重叠
  - 关键风险在于状态一致性和回归验证，串行更稳

#### Issue List

- 当前无阻塞

阻塞：否
进入 3.2

### Stage 3.3 - Code Review

- 需求覆盖：通过
  - 已交付 local auth file backend、restore normalization、ttl/expiry、可选 admin authz skeleton
- 架构合理性：通过
  - persistence 继续留在 store adapter 后面
  - runtime child 暂态未被混入 durable truth
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 主要查找仍为 dict 索引
  - prune 只扫描 `pending / approved / permit`
- 可读性与一致性：通过
  - policy key、response code、README/spec 命名保持一致
- 可扩展性与配置化：通过
  - custom backend 仍可挂 `now_ms()` / `authorize_admin_action(...)`
  - future 可继续接 flash/NVS/trust/bootstrap
- 稳定性与安全：通过
  - 过期记录显式清理
  - authz deny 显式 `4403`
  - file load/save 失败不吞错
- 测试覆盖情况：通过
  - targeted `51` tests 通过
  - full `discover` `91` tests 通过
  - `git diff --check` 通过
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未派发子 Agent

### Stage 4 - Change Archive

- Change:
  - `docs/change/2026-04-02_embedded-auth-persistence-profile.md`
- Requirements impact:
  - `none`
- Specs impact:
  - `updated`
- Lessons impact:
  - `updated`
- Related requirements:
  - `docs/requirements/embedded-sdk-edge-hub.md`
- Related specs:
  - `docs/specs/micropython-edge-runtime.md`
  - `docs/specs/micropython-edge-server-alignment.md`
  - `repo/MyFlowHub-Server/docs/specs/auth.md`
- Related lessons:
  - `docs/lessons/micropython-edge-local-auth-nodeid-allocation.md`
  - `docs/lessons/micropython-edge-local-auth-persistence-restore.md`
- Validation:
  - `cmd /c "call C:\ProgramData\anaconda3\Scripts\activate.bat micropython && python -m unittest micropython.tests.test_async_edge micropython.tests.test_edge_runtime"`
  - `cmd /c "call C:\ProgramData\anaconda3\Scripts\activate.bat micropython && python -m unittest discover -s micropython/tests"`
  - `git diff --check`
