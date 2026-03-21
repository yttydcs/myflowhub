# 2026-02-17 - Win：Flow 改为 send+await（SDK v1 Awaiter）

## 背景 / 目标
- 现状：Win 侧 Flow 的 `set/run/status/list/get` 控制动作为 fire-and-forget；调用方无法同步获知业务成功/失败，只能依赖 `session.frame`（或缺少直接反馈）。
- 目标：将 Flow 控制动作升级为“发送并等待响应”，让调用方在超时内得到明确结果，同时保持前端 `session.frame` 事件链路不变（仍由 frame 驱动 UI 状态更新）。

## 具体变更内容
### 修改
- `internal/services/flow/service.go`
  - 将以下控制动作改为 `SendCommandAndAwait` 并匹配对应 `*_resp`：
    - `set` → `set_resp`
    - `run` → `run_resp`
    - `status` → `status_resp`
    - `list` → `list_resp`
    - `get` → `get_resp`
  - `*Simple` 方法统一增加默认超时：`8s`（`context.WithTimeout`）。
  - resp 业务判定规则：
    - `code == 1` 视为成功，返回 `nil`
    - `code != 1` 视为失败，返回 `error`（优先 `msg`，否则携带 `code`）
  - `Send(action, data)` 仍保持 send-only（本 PR 不为“任意 action”引入通用 await 规则）。

## Plan.md 任务映射
- FWA1：Flow：控制动作改为 send+await
- FWA2：Simple：默认 timeout=8s
- FWA3：回归测试（Windows）
- FWA4：Code Review + 归档变更

## 关键设计决策与权衡
- Await 匹配维度：`MsgID + SubProto(Flow=6) + Action(*_resp)`。
- 保持前端行为不变：
  - SDK v1 `SetOnFrame` 保障 matched frame 仍会发布 `session.frame`，因此前端 `frontend/src/stores/flow.ts` 无需修改。
- `list/get/status` 的返回数据在 Win 后端仅用于判定 `code` 成功/失败；是否在 UI 层消费由既有 `session.frame` store 逻辑负责（本 PR 不改前端）。

## 测试与验证方式 / 结果
### 回归（Windows）
- 命令：
  - `$env:GOTMPDIR='d:\\project\\MyFlowHub3\\.tmp\\gotmp'`
  - `New-Item -ItemType Directory -Force -Path $env:GOTMPDIR | Out-Null`
  - `go test ./... -count=1 -p 1`
- 结果：通过（无测试文件，编译链路通过）。

### 冒烟（手动）
- Win 连接到 Server 后执行：
  - Flow：List / Get / Set / Run / Status（timeout 内返回）
  - UI 仍能基于 `session.frame` 正常更新 flow 列表、加载内容、run/status 状态
- 预期：
  - 控制类调用 Promise 在 `8s` 内 resolve/reject（可区分成功/失败/超时）
  - 前端 store 的 frame 消费链路不丢帧

## 潜在影响与回滚方案
- 影响：
  - Flow 控制类调用由“立即返回”变为“等待响应/超时后返回”，更利于脚本化与错误呈现。
  - 若服务端响应 action 不匹配或未继承 MsgID，可能导致 await 超时（调用方可见）。
- 回滚：
  - 回退本 PR 提交（恢复为 fire-and-forget 的 `SendCommand`）。

