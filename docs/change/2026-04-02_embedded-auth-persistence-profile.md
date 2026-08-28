# 2026-04-02 Embedded Auth Persistence Profile

## 变更背景 / 目标

上一轮已经把 built-in local auth 从“只有 `register/login`”扩到了 optional local authority profile skeleton，但仍缺三块关键能力：

- 真实可用的 persistence backend
- `pending / approved / permit` 的 expiry/ttl
- local authority admin/query 的可选 authz skeleton

本轮目标是在不把 heavy auth 默认压进轻量 SDK 基线的前提下，把这三块补成显式 opt-in profile。

## 具体变更内容

- 更新 `micropython/myflowhub/async_edge_local.py`
  - 新增 `LocalAuthFileStoreBackend(path)`：
    - 通过 JSON 文件保存/恢复 local auth state
    - load 缺失文件时返回空 state
    - save 时不再持久化 runtime-only `child_id / child_devices`
  - 扩展 store adapter optional hook：
    - `now_ms()`
    - `authorize_admin_action(action, actor, actor_record, context, state)`
  - local auth state load/restore 规范化：
    - `node_index` 从 durable `devices` 重建
    - `child_id / child_devices` 恢复时清空
    - `next_node_id` 与已用/已预留 `node_id` 对齐
  - local policy 增量：
    - `policy.require_admin_authz`
    - `policy.pending_ttl_sec`
    - `policy.approved_ttl_sec`
    - `policy.permit_ttl_sec`
    - `policy.role_perms`
    - `policy.action_perms`
  - `pending / approved / permit` 增加 TTL/expiry pruning
  - local admin/query 增加可选 authz skeleton：
    - 默认关闭
    - 开启后 deny 返回 `code=4403,msg="forbidden"`
  - local auth response 补充可选 `perms`
- 更新 `micropython/myflowhub/__init__.py`
  - 导出 `LocalAuthFileStoreBackend`
- 更新 tests
  - `micropython/tests/test_async_edge.py`
    - 新增 file backend restore
    - 新增 pending expiry
    - 新增 approved expiry
    - 新增 permit ttl
    - 新增 authz allow / deny
  - `micropython/tests/test_edge_runtime.py`
    - 新增 sync file backend restore
    - 新增 sync permit ttl
    - 新增 sync authz deny
- 更新文档
  - `docs/specs/micropython-edge-runtime.md`
  - `docs/specs/micropython-edge-server-alignment.md`
  - `micropython/README.md`
  - `docs/lessons/micropython-edge-local-auth-persistence-restore.md`

## Requirements impact

- `none`

## Specs impact

- `updated`

## Lessons impact

- `updated`

## Related requirements

- `docs/requirements/embedded-sdk-edge-hub.md`

## Related specs

- `docs/specs/micropython-edge-runtime.md`
- `docs/specs/micropython-edge-server-alignment.md`
- `repo/MyFlowHub-Server/docs/specs/auth.md`

## Related lessons

- `docs/lessons/micropython-edge-local-auth-nodeid-allocation.md`
- `docs/lessons/micropython-edge-local-auth-persistence-restore.md`

## 对应 plan.md 任务映射

- `T1`
- `T2`
- `T3`
- `T4`

## 经验 / 教训摘要

- local auth persistence 不能直接把 runtime child 绑定字段原样落盘；`child_id / child_devices` 必须被视为 runtime-only。
- JSON 持久化会把整数 key 字典变成字符串 key；像 `node_index` 这样的索引必须在 restore 时重建，而不是盲信反序列化结果。
- local authority admin authz 适合先做显式 opt-in skeleton，而不是默认启用完整权限模型。

## 可复用排查线索

- 症状：
  - 重启后 local auth state 看似恢复，但 child targeted route 异常
  - file backend 恢复后 `node_index` 查不到已有 `node_id`
  - admin/query deny 行为与 `authority unavailable` 混淆
- 触发条件：
  - 给 `enable_builtin_local_auth(..., state=...)` 接 JSON/file backend
  - 打开 `policy.require_admin_authz`
  - 给 pending/approved/permit 配 ttl
- 关键词：
  - `LocalAuthFileStoreBackend`
  - `node_index`
  - `child_id`
  - `pending_ttl_sec`
  - `require_admin_authz`
  - `4403 forbidden`
- 快速检查：
  - 看持久化 JSON 是否还包含 live child 绑定
  - 看 restore 后是否重建 `node_index`
  - 看 deny 是否返回 `4403` 而不是 `4500`

## 关键设计决策与权衡

- 采用显式 `LocalAuthFileStoreBackend(path)`
  - 优点：给调用方一个开箱即用的 persistence backend，同时保持 store adapter 边界
  - 代价：仍然只覆盖 local auth profile，不等于 full hub trust persistence
- 采用 `policy.require_admin_authz` 默认关闭
  - 优点：不打破现有 local demo / baseline
  - 代价：部署方要显式开启才会得到本地 admin authz
- 采用 `role_perms + action_perms` skeleton
  - 优点：足够覆盖 local admin/query 的最小 authz
  - 代价：仍不是 server 级完整权限模型

## 测试与验证方式 / 结果

- targeted:
  - `cmd /c "call C:\ProgramData\anaconda3\Scripts\activate.bat micropython && python -m unittest micropython.tests.test_async_edge micropython.tests.test_edge_runtime"`
  - 结果：通过（`51` tests）
- full:
  - `cmd /c "call C:\ProgramData\anaconda3\Scripts\activate.bat micropython && python -m unittest discover -s micropython/tests"`
  - 结果：通过（`91` tests）
- static:
  - `git diff --check`
  - 结果：通过（仅 CRLF warning，无 diff error）

## 潜在影响与回滚方案

- 潜在影响：
  - permit/pending/approved 若显式设置了过去时间，现会被视为过期
  - 开启 `policy.require_admin_authz=true` 后，未授权 actor 会收到 `4403`
  - file backend restore 不再保留旧 runtime child 绑定
- 回滚方案：
  - 回退 `LocalAuthFileStoreBackend` 与 policy/ttl/authz 增量
  - 回退新增 tests / specs / README / lesson
  - 恢复为上一轮的 in-memory local authority skeleton

## 子Agent执行轨迹

- 本轮未派发子 Agent
