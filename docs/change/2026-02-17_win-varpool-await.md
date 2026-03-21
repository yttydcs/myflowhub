# 2026-02-17 - Win：VarPool（VarStore）改为 send+await（SDK v1 Awaiter）

## 背景 / 目标
- 现状：Win 侧 VarPool（VarStore 子协议）的 `set/get/list/revoke/subscribe/unsubscribe` 为 fire-and-forget，调用方无法同步获知业务成功/失败，只能依赖 `session.frame` 异步更新 UI。
- 目标：将上述动作升级为“发送并等待响应”，让调用方在超时内得到明确结果，同时保持前端 `session.frame` 事件链路不变。

## 具体变更内容
### 修改
- `internal/services/varpool/service.go`
  - 将以下动作改为 `SendCommandAndAwait` 并匹配对应 resp action：
    - `set` → `set_resp`
    - `get` → `get_resp`
    - `list` → `list_resp`
    - `revoke` → `revoke_resp`
    - `subscribe` → `subscribe_resp`
    - `unsubscribe` → `subscribe_resp`（现有协议实现：unsubscribe 复用 subscribe_resp 回包）
  - `*Simple` 方法统一增加默认超时：`8s`（`context.WithTimeout`）。
  - resp 业务判定规则（`varstore.VarResp`）：
    - `code == 1` 视为成功，返回 `nil`
    - `code != 1` 视为失败，返回 `error`（优先 `msg`，否则携带 `code`）
  - 实现细节：`sendAndAwait` 内部预先 `TrimSpace` action/name，减少重复处理（不影响 wire 行为）。

## Plan.md 任务映射
- VPA1：VarPool：动作改为 send+await
- VPA2：Simple：默认 timeout=8s
- VPA3：回归测试（Windows）
- VPA4：Code Review + 归档变更

## 关键设计决策与权衡
- Await 匹配维度：`MsgID + SubProto(VarStore=3) + Action(*_resp)`。
- 保持前端行为不变：
  - SDK v1 `SetOnFrame` 保障 matched frame 仍会发布 `session.frame`，因此前端 `frontend/src/stores/varpool.ts` 的解析与 UI 更新逻辑无需修改。
- `unsubscribe` 的响应 action 选择：
  - 以现有 Server 逻辑为准：`unsubscribe`（非 assisted）回包 action 为 `subscribe_resp`，因此 await 也匹配 `subscribe_resp`。

## 测试与验证方式 / 结果
### 回归（Windows）
- 命令：
  - `$env:GOTMPDIR='d:\\project\\MyFlowHub3\\.tmp\\gotmp'`
  - `New-Item -ItemType Directory -Force -Path $env:GOTMPDIR | Out-Null`
  - `go test ./... -count=1 -p 1`
- 结果：通过（无测试文件，编译链路通过）。

### 冒烟（手动）
- Win 连接到 Server 后执行：
  - VarPool：ListMine / Get / Set / Revoke / Subscribe / Unsubscribe
- 预期：
  - 调用 Promise 在 `8s` 内 resolve/reject（可区分成功/失败/超时）
  - UI 仍能收到 `session.frame` 并按既有 store 逻辑更新 keys/value/subscribed 状态

## 潜在影响与回滚方案
- 影响：
  - VarPool 调用由“立即返回”变为“等待响应/超时后返回”，更利于脚本化与错误呈现。
  - 若服务端响应 action 不匹配或未继承 MsgID，可能导致 await 超时（调用方可见）。
- 回滚：
  - 回退本 PR 提交（恢复为 fire-and-forget 的 `SendCommand`）。
