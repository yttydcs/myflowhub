# 2026-03-06 VarStore：逐跳 `MajorCmd` 回程联动检查（Win）

## 变更规模
- 级别：较大（联动项：SDK await 需升级，否则 VarPool await 会超时）

## 背景 / 目标
VarStore 的 `*_resp/assist_*_resp` 已按规范改为 `Header.Major=MajorCmd`（逐跳可见）。Win 侧 VarPool 通过 `myflowhub-sdk/await` 实现 `SendCommandAndAwait`，旧 SDK 仅接受 `MajorOKResp/MajorErrResp`，会导致：

- VarPool 的 set/get/list/revoke/subscribe await 超时（响应帧被当作 unmatched 丢给 onUnmatched）

目标：确认 Win 侧调用链对 `MajorCmd` 响应不存在额外过滤/假设，并给出联动升级路径。

## 检查结论
- Win 侧 `internal/session.Session` 直接使用 `myflowhub-sdk/await.Client.SendAndAwait`。
- Win 侧 `VarPoolService.sendAndAwait` 只依赖 `respAction`（payload.action）与 `MsgID/SubProto` 的 await 匹配，不依赖响应 header.major。
- 因此：**Win 侧无需额外代码适配**，关键在于 SDK await 必须升级到包含 VarStore `MajorCmd` 白名单支持的版本。

## 后续联动（待执行）
- 依赖升级：
  - 将 `go.mod` 中 `github.com/yttydcs/myflowhub-sdk` 升级到包含本次 await 修复的版本（发布后再落版本号）。
- 冒烟验证（建议）：
  - 连接到 Hub，验证 VarPool：`list/get/set/revoke/subscribe/unsubscribe` 在多 hop 场景下均能收到响应且不超时。
  - 观察 UI：list 空集合视为成功展示空列表；set 失败不应更新本地值；notify/var_changed 不重复提示。

## Code Review
- 结论：通过（本仓未引入协议行为改造代码；联动关键在 SDK 版本升级）

## 回滚方案
- 若升级后出现问题：回退 Win 的 SDK 版本依赖即可（Win 本仓未引入协议行为改造）。
