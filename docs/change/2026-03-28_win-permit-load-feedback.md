# 2026-03-28 Win 准入许可页加载反馈与头部动作收敛

## 变更背景 / 目标
- `MyFlowHub-Win` 的 `Permit Issuance` 页在打开时会自动加载活动 permit 列表；一旦请求失败，页面会直接弹出 `Failed to load permits.` toast，首屏体验偏突兀。
- 同时，`Refresh / New Permit` 仍放在上方介绍卡片中，而“活动许可”卡片右上角保留 `共 X 条 / 实时` 标签，整体显得臃肿。
- 本轮目标是：
  - 修复 Win 侧 permit list 响应码解析缺口
  - 将 permit list 失败反馈改为页面内提示
  - 将按钮移动到“活动许可”卡片右上角，并移除多余标签

## 具体变更内容
### MyFlowHub-Win
- 更新 [internal/services/auth/service.go](../../repo/MyFlowHub-Win/internal/services/auth/service.go)
  - 在 `extractAuthCodeMsg()` 中补齐 `*ListRegisterPermitsResp`
  - 确保 `list_register_permits` 成功回包不会被误判为 `code=0`
- 更新 [internal/services/auth/service_test.go](../../repo/MyFlowHub-Win/internal/services/auth/service_test.go)
  - 新增 permit list response code 抽取回归
- 更新 [frontend/src/pages/PermitIssuance.vue](../../repo/MyFlowHub-Win/frontend/src/pages/PermitIssuance.vue)
  - 新增 permit page 本地 `loadState.error`
  - permit list 自动加载或手动刷新失败时，改为在 permit card 内显示错误提示
  - 不再对 permit list 加载失败直接触发全局 toast
  - 将 `Refresh / New Permit` 移动到“活动许可”卡片右上角
  - 移除 permit 总数和“实时”标签
- 更新 [frontend/src/pages/PermitIssuance.test.ts](../../repo/MyFlowHub-Win/frontend/src/pages/PermitIssuance.test.ts)
  - 补按钮位置断言
  - 补首屏自动加载失败时的页面内错误提示断言
- 更新 [frontend/src/i18n/messages/operations.ts](../../repo/MyFlowHub-Win/frontend/src/i18n/messages/operations.ts)
  - 新增 `Failed to load permits.` 中文文案
- repo 级归档：
  - [2026-03-28_win-permit-load-feedback.md](../../repo/MyFlowHub-Win/docs/change/2026-03-28_win-permit-load-feedback.md)

## Requirements impact
- `none`

## Specs impact
- `none`

## Lessons impact
- `none`

## Related requirements
- [authority-admin-console.md](../../repo/MyFlowHub-Win/docs/requirements/authority-admin-console.md)

## Related specs
- [authority-admin-console.md](../../repo/MyFlowHub-Win/docs/specs/authority-admin-console.md)
- [auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md)

## Related lessons
- `none`

## 对应 plan.md 任务映射
- `PERMIT-FIX-1`
- `PERMIT-FIX-2`
- `PERMIT-FIX-3`
- `REVIEW-1`
- `ARCHIVE-1`
- `MERGE-1`

## 经验 / 教训摘要
- auth typed action 新增响应结构后，统一 `code/msg` 抽取必须同步扩展；否则成功回包也会被 UI 误判。
- 对首屏自动加载型页面，更合适的失败反馈通常是页面内错误态，而不是直接弹全局 toast。
- permit 列表的主动作属于列表本身，把按钮放回“活动许可”卡片头部会比挂在介绍卡里更顺手。

## 可复用排查线索
- 症状：
  - 打开准入许可页立刻看到 `Failed to load permits.`
  - permit list 成功回包后仍被 UI 判失败
  - “活动许可”卡片头部仍显示 `共 X 条 / 实时`
- 触发条件：
  - `extractAuthCodeMsg()` 未覆盖 `ListRegisterPermitsResp`
  - permit 页首屏自动加载失败仍走 toast 路径
- 关键词：
  - `ListRegisterPermitsResp`
  - `extractAuthCodeMsg`
  - `data-permit-load-error`
  - `data-permit-card-actions`
- 快速检查：
  - 看 [service.go](../../repo/MyFlowHub-Win/internal/services/auth/service.go) 是否已处理 `*ListRegisterPermitsResp`
  - 看 [PermitIssuance.vue](../../repo/MyFlowHub-Win/frontend/src/pages/PermitIssuance.vue) 是否使用页面内错误提示
  - 看 [PermitIssuance.vue](../../repo/MyFlowHub-Win/frontend/src/pages/PermitIssuance.vue) 中按钮是否位于 permit card header actions

## 关键设计决策与权衡
- 不扩大到 authority 解析或协议侧，只修 Win 本地可确定的解析和交互问题
  - 优点：改动面小，回滚清晰
  - 代价：若现场真实根因是 authority 根本没回 `list_register_permits_resp`，页面仍会显示错误提示而不会伪造成功
- permit list 失败改为页面内错误提示，而不是继续 toast
  - 优点：首屏更稳定，用户仍可继续点 `Refresh` 或 `New Permit`
  - 代价：错误提示不再自动进入全局提示层

## 测试与验证方式 / 结果
- `MyFlowHub-Win`
  - `$env:GOWORK='off'; go test ./internal/services/auth/... -count=1`
  - 结果：通过
- `MyFlowHub-Win/frontend`
  - `npm test -- PermitIssuance`
  - 结果：通过
- `MyFlowHub-Win`
  - `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 备注：仍会打印 `Not found: time.Time`，但退出码为 0
- `MyFlowHub-Win/frontend`
  - `npm run build`
  - 结果：通过

## Merge 结果
- [repo/MyFlowHub-Win](../../repo/MyFlowHub-Win)
  - `main` 已 fast-forward 到 `0ddcdec`
  - commit: `fix: 收敛准入许可页加载反馈`

## 潜在影响与回滚方案
- 潜在影响：
  - permit 页加载失败后优先显示卡片内错误提示，不再自动弹 toast
  - permit 页动作入口改到列表卡片头部
- 回滚方案：
  - 回退 `MyFlowHub-Win` 中的 `service.go`、`service_test.go`、`PermitIssuance.vue`、`PermitIssuance.test.ts`、`operations.ts`
  - 回退 [2026-03-28_win-permit-load-feedback.md](../../repo/MyFlowHub-Win/docs/change/2026-03-28_win-permit-load-feedback.md)

## 子Agent执行轨迹
- 无子 agent
