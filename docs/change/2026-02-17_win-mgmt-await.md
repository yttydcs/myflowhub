# 2026-02-17 - Win：Management 改为 send+await（SDK v1 Awaiter）

## 背景 / 目标
- 现状：Win 侧 `management` 子协议为 fire-and-forget，调用方无法同步获知业务成功/失败，只能依赖 `session.frame` 异步更新 UI。
- 目标：将常用 Management 动作升级为“发送并等待响应”，让调用方在超时内得到明确结果，同时保持前端 `session.frame` 事件链路不变。

## 具体变更内容
### 修改
- `internal/services/management/service.go`
  - 将以下动作改为 `SendCommandAndAwait` 并匹配对应 `*_resp`：
    - `node_echo` → `node_echo_resp`
    - `list_nodes` → `list_nodes_resp`
    - `list_subtree` → `list_subtree_resp`
    - `config_get` → `config_get_resp`
    - `config_set` → `config_set_resp`
    - `config_list` → `config_list_resp`
  - `*Simple` 方法统一增加默认超时：`8s`（`context.WithTimeout`）。
  - resp 业务判定规则：
    - `code == 1` 视为成功，返回 `nil`
    - `code != 1` 视为失败，返回 `error`（优先 `msg`，否则携带 `code`）

## Plan.md 任务映射
- MWA1：Management：全量动作改为 send+await
- MWA2：Simple：默认 timeout=8s
- MWA3：回归测试（Windows）
- MWA4：Code Review + 归档变更

## 关键设计决策与权衡
- Await 匹配维度：`MsgID + SubProto(Management=1) + Action(*_resp)`。
- 保持前端行为不变：
  - PR13 已通过 `await.Client.SetOnFrame` 保证“被 Awaiter 匹配的帧”仍会回调 `SessionService.handleFrame` 并发布 `session.frame`。
  - 因此本 PR 仅增加“调用方同步等待”能力，不改动前端 store 的解析与更新逻辑。
- 失败语义统一由 payload `code/msg` 表达：
  - 不依赖 header major 表达业务失败（保持 wire/框架语义稳定）。

## 测试与验证方式 / 结果
### 回归（Windows）
- 命令：
  - `$env:GOTMPDIR='d:\\project\\MyFlowHub3\\.tmp\\gotmp'`
  - `New-Item -ItemType Directory -Force -Path $env:GOTMPDIR | Out-Null`
  - `go test ./... -count=1 -p 1`
- 结果：通过（无测试文件，编译链路通过）。

### 冒烟（手动）
- Win 连接到 Server 后执行：
  - NodeEcho / ListNodes / ListSubtree / ConfigGet / ConfigSet / ConfigList
- 预期：
  - 调用 Promise 在 `8s` 内 resolve/reject（可区分成功/失败/超时）
  - UI 仍能收到 `session.frame` 并按既有 store 逻辑更新

## 潜在影响与回滚方案
- 影响：
  - management 调用由“立即返回”变为“等待响应/超时后返回”，更利于脚本化与错误呈现。
  - 若服务端响应 action 不匹配或未继承 MsgID，可能导致 await 超时（调用方可见）。
- 回滚：
  - 回退本 PR 提交（恢复为 fire-and-forget 的 `SendCommand`）。

