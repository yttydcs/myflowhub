# Hub

## Purpose

Hub 是权威节点树的根或中继宿主，负责持久身份、子节点准入、路由、策略裁决和内建管理资源。它不是全局业务模块容器。

## Observable behavior

- 首次启动原子创建身份与版本化配置，后续启动保留 NodeID 与信任状态。
- 可同时监听多个已配置 Transport；链路差异不泄漏到资源和权限层。
- 通过 catalog、topology、health、config Variables 暴露状态，通过 permit、revoke、config、policy grant/revoke Commands 执行管理动作。
- 断连、重连、reparent 和 revoke 会使旧树边、旧策略世代与相关订阅明确失效。

## Permissions

- 只有当前直接父节点可以下发父控子 control；普通下级请求必须在权威路径上裁决。
- 准入许可必须绑定父身份、子身份或一次性挑战、角色、有效期和使用状态。
- 管理 Commands 使用独立权限点并写入不含敏感材料的审计事件。
- `system/policy/grant` 与 `system/policy/revoke` 只修改显式的 subject/action/resource 三元组；资源被看见不代表获得权限。

## 首次引导

Hub 不预置超级用户，也不会为了方便而关闭默认拒绝。首次接入按以下顺序完成：

1. 保持 Hub 停止，使用 `mfh-hub -identity` 读取持久 NodeID 与 raw-base64 Ed25519 公钥；
2. 为子节点身份使用 `-issue-node-id`、`-issue-public-key`、`-issue-role` 签发短期、一次性 permit；
3. 使用 `-policy grant` 为该主体授予最小的 `subscribe` / `invoke` 资源集合；
4. 启动 Hub，之后优先通过已鉴权的 `system/admission/issue`、`system/admission/revoke`、`system/policy/grant`、`system/policy/revoke` Commands 管理。

离线命令直接原子更新同一持久状态，必须在 Hub 停止时运行；它不是绕过在线权限的并发旁路。示例见仓库 `README.md` 与 `scripts/run-dev.ps1 -DryRun` 输出。

## Non-goals

- 不提供旧 SubProto handler、兼容 wire bridge 或隐式 defaultset。
- 不根据物理 Transport 类型决定权限。

## Acceptance

重启身份稳定；多 listener 可用；allow/deny/revoke 可重复验证；损坏配置显式失败；优雅关闭不遗留会话或 pending 请求。
