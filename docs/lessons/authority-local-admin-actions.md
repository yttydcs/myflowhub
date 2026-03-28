# Authority Local Admin Actions

## Summary
- 当前 auth backend 的审批与 permit 管理动作并不是完整的通用 remote authority 管理链路。
- 在 `sourceId != authorityId` 的场景下，Win 侧如果仍把这些动作当成普通远程请求发送，常见结果就是页面等待到 `request timed out`。

## Symptoms
- `Permit Issuance` 页打开后显示：
  - `加载准入许可失败。`
  - `auth list_register_permits: request timed out`
- 当前会话节点不是 authority 节点
- 页面看起来像“普通加载失败”，但同一 authority 本机上又能正常工作

## Trigger Conditions
- `sourceId != authorityId`
- 当前 backend 版本仍要求审批 / permit 管理从 authority 节点本机发起
- Win 页面对这个限制没有显式 guard，还继续自动加载或触发 permit 管理动作

## Quick Checks
1. 对比当前 session `nodeId` 和解析出的 `authorityId`。
2. 查看 [auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md)：
   - 若仍写着审批 / permit 管理建议从 authority 节点操作，这不是单纯前端渲染问题。
3. 查看 Win permit 页是否已经进入 authority-local 受限态：
   - 关键字：`data-permit-remote-authority`
   - 关键字：`requires authority-local session`

## Resolution
- 若只是收敛当前 UX：
  - 在 Win orchestration 层对 permit / admin 动作做 authority-local 快速失败
  - 页面进入显式限制提示，停止继续等待 timeout
- 若目标是支持真实 remote authority 管理：
  - 需要另起跨仓 workflow
  - 明确 `MyFlowHub-Win`、`MyFlowHub-SubProto`、`MyFlowHub-Server` 的 remote authority 管理链路，而不是只改页面

## Prevention
- 当 backend 仍有 authority-local 前提时，前端不得把 timeout 当成常规失败 UX。
- authority 管理页面在 `authorityId != sourceId` 时，应优先显示能力边界提示。
- 这种限制一旦确认，应同步写入 requirements/specs，而不是只留在 change log。

## Related Docs
- [2026-03-28_win-permit-remote-authority-guard.md](../change/2026-03-28_win-permit-remote-authority-guard.md)
- [auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md)
