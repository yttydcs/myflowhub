# MicroPython Edge Local Auth Persistence Restore

## Summary

当 built-in local auth 开始支持 JSON/file 持久化时，不能把 runtime 级 child 会话状态原样恢复。
`child_id / child_devices` 这类字段只对当前进程内的 live session 有意义；`node_index` 这类带整数 key 的索引在 JSON 里还会被字符串化。
正确做法是只持久化 durable auth data，并在 load 时重建索引、清空 runtime-only 绑定。

## Lookup Hints

- Symptoms:
  - 板子重启后，本地 auth 看起来恢复了，但 targeted frame 仍打向不存在的 child
  - file backend 恢复后，`node_index` 查不到已存在的 `node_id`
  - 本地 pending/permit 恢复正常，但 child route / session 归属异常
- Trigger Conditions:
  - 给 `enable_builtin_local_auth(..., state=...)` 接 JSON/file backend
  - 用 `json` 持久化包含整数 key 的索引
  - 把 live child session 相关字段一起保存到磁盘
- Keywords:
  - `LocalAuthFileStoreBackend`
  - `child_id`
  - `child_devices`
  - `node_index`
  - `restore`
  - `json`
- Quick checks:
  - 看 save 时是否把 `child_id / child_devices` 归零
  - 看 load 时是否从 durable `devices` 重建 `node_index`
  - 看恢复后 admin/query 正常但 child route 异常时，是否仍在信任旧 runtime state

## Symptoms

- 重启后 local auth `register/login` 似乎可用，但 child targeted forwarding 丢失或命中错误对象
- 恢复后的 local auth state 里明明有设备记录，却查不到对应 `node_id`
- admin/query 能列出 pending/permit，但 runtime child 绑定状态不一致

## Impact

- child route 绑定被脏状态污染
- 持久化恢复看似成功，实际 runtime 行为不可靠
- 后续 var/topic/targeted response 都可能受影响

## Trigger Conditions

- 新增 file backend 或自定义 `save_state()/get_state()` backend
- 直接把完整内存态 dict 原样 JSON 化
- 假设 `node_index`、`child_devices` 也属于 durable truth

## Root Cause

- `child_id / child_devices` 依赖当前进程内 live child session，跨重启没有意义
- JSON 会把 dict 的整数 key 变成字符串，导致 `node_index` 这类索引恢复后失真
- 如果 load 时继续信任这些字段，就会把 runtime 暂态误当成 durable state

## Investigation Trail

1. 检查 file backend 保存的 JSON，确认是否还带着旧 `child_id / child_devices`
2. 检查恢复后的 `node_index` key 类型，确认是否已被字符串化
3. 对比 durable `devices` 与 load 后的 `node_index` 是否一致
4. 重启后先跑 admin/query，再跑 child targeted/route 相关验证

## Resolution

- save 时把 `child_id` 归零，并把 `child_devices` 清空
- load 时从 durable `devices` 重建 `node_index`
- runtime child 绑定在新会话建立前一律视为空

## Prevention / Guardrails

- 不要把 runtime-only child state 直接当成持久化真相
- 任何 JSON 持久化的整型索引，load 时都要显式 normalize
- 增加“重启后 restore + local admin/query + child route”组合回归

## Related Docs

- Requirements:
  - `docs/requirements/embedded-sdk-edge-hub.md`
- Specs:
  - `docs/specs/micropython-edge-runtime.md`
  - `docs/specs/micropython-edge-server-alignment.md`
- Change:
  - `docs/change/2026-04-02_embedded-auth-persistence-profile.md`
