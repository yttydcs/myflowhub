# 2026-02-17 - Win：TopicBus 改为 send+await（SDK v1 Awaiter）

## 背景 / 目标
- 现状：Win 侧 TopicBus 的 subscribe/unsubscribe/list_subs 等控制动作为 fire-and-forget，调用方无法同步获知业务成功/失败，只能依赖 `session.frame`（或缺少直接反馈）。
- 目标：将 TopicBus 控制动作升级为“发送并等待响应”，让调用方在超时内得到明确结果，同时保持前端 `session.frame` 事件链路不变（publish 事件继续由 frame 驱动）。

## 具体变更内容
### 修改
- `internal/services/topicbus/service.go`
  - 将以下控制动作改为 `SendCommandAndAwait` 并匹配对应 `*_resp`：
    - `subscribe` → `subscribe_resp`
    - `subscribe_batch` → `subscribe_batch_resp`
    - `unsubscribe` → `unsubscribe_resp`
    - `unsubscribe_batch` → `unsubscribe_batch_resp`
    - `list_subs` → `list_subs_resp`
  - `*Simple` 方法统一增加默认超时：`8s`（`context.WithTimeout`）。
  - resp 业务判定规则：
    - `code == 1` 视为成功，返回 `nil`
    - `code != 1` 视为失败，返回 `error`（优先 `msg`，否则携带 `code`）
  - `publish` 保持 send-only（协议无 `publish_resp`），仍通过 `session.frame` 驱动前端事件更新。

## Plan.md 任务映射
- TBA1：TopicBus：控制动作改为 send+await
- TBA2：Simple：默认 timeout=8s
- TBA3：回归测试（Windows）
- TBA4：Code Review + 归档变更

## 关键设计决策与权衡
- Await 匹配维度：`MsgID + SubProto(TopicBus=4) + Action(*_resp)`。
- 保持前端行为不变：
  - SDK v1 `SetOnFrame` 保障 matched frame 仍会发布 `session.frame`，因此前端 `frontend/src/stores/topicbus.ts`（主要处理 `publish`）无需修改。
- `list_subs` 的返回数据（topics 列表）在 Win 后端仅用于判定 `code` 成功/失败；是否在 UI 层消费由后续需求决定（本 PR 不改前端）。

## 测试与验证方式 / 结果
### 回归（Windows）
- 命令：
  - `$env:GOTMPDIR='d:\\project\\MyFlowHub3\\.tmp\\gotmp'`
  - `New-Item -ItemType Directory -Force -Path $env:GOTMPDIR | Out-Null`
  - `go test ./... -count=1 -p 1`
- 结果：通过（无测试文件，编译链路通过）。

### 冒烟（手动）
- Win 连接到 Server 后执行：
  - TopicBus：Subscribe / Unsubscribe（含 batch）/ ListSubs（timeout 内返回）
  - TopicBus：Publish 后前端事件列表仍能收到新事件（frame 驱动）
- 预期：
  - 控制类调用 Promise 在 `8s` 内 resolve/reject（可区分成功/失败/超时）
  - publish 事件仍按既有逻辑正常显示

## 潜在影响与回滚方案
- 影响：
  - 控制类 TopicBus 调用由“立即返回”变为“等待响应/超时后返回”，更利于脚本化与错误呈现。
  - 若服务端响应 action 不匹配或未继承 MsgID，可能导致 await 超时（调用方可见）。
- 回滚：
  - 回退本 PR 提交（恢复为 fire-and-forget 的 `SendCommand`）。

