# 2026-02-17 - Win：Auth Register/Login 改为 send+await（SDK v1 Awaiter）（PR13-WIN-Awaiter）

## 变更背景 / 目标
Win 端的 Home/Auth 当前为“发送即返回”（fire-and-forget），前端通过监听 `session.frame` 事件解析 `*_login_resp/*_register_resp` 来更新 UI。

这会带来两个问题：
1) UI 发起 `RegisterSimple/LoginSimple` 时，Promise 只能表示“发送成功/失败”，无法同步得知服务端业务成功/失败（只能等后续 frame 更新）。
2) 接入 SDK v1 Awaiter 后，**匹配成功的响应帧会被拦截 deliver**，若不做额外处理，`session.frame` 事件将丢失，前端无法更新登录状态。

本次变更目标（仅覆盖 Auth）：
1) `Register/Login` 变为“发送并等待响应”（按 `MsgID + SubProto + Action` 匹配）。
2) 保持前端不改：`session.frame` 事件链路不缺失，仍能更新 UI。
3) `RegisterSimple/LoginSimple` 默认 timeout=`8s`；`code!=1` 视为 error（前端 catch 可见）。

## 具体变更内容
### 修改：Session 底层接入 Awaiter（保持帧事件不缺失）
- `internal/session/session.go`
  - 底层由 `myflowhub-sdk/await.Client` 承载 connect/send/await
  - 通过 SDK 的 `onFrame` hook 把所有接收帧继续回调给 `SessionService.handleFrame`

### 新增：SessionService send+await 能力（自动 MsgID）
- `internal/services/session/service.go`
  - 新增 `SendCommandAndAwait`
  - **不再使用 `uint32(time.Now().UnixNano())` 作为 MsgID**（避免 32-bit 截断导致的高概率冲突）；MsgID 由 SDK 自动生成并写回 header

### 修改：Auth 使用 send+await（默认 8s 超时）
- `internal/services/auth/service.go`
  - `Register/Login`：改为 send+await 并解析 `auth.RespData`
    - `code==1` 成功返回 nil
    - `code!=1` 返回 error（优先使用 `msg`）
  - `RegisterSimple/LoginSimple`：包一层 `context.WithTimeout(..., 8s)`

### 文档
- `plan.md`：本 workflow 的需求/架构/计划与验收
- `repo/MyFlowHub-Win/docs/plan/plan_archive_2026-02-15_win-use-sdk-v0.md`：归档上一轮计划文档（便于审计回放）

## plan.md 任务映射
- WA1 - Session：切换到 SDK await.Client 并保持 session.frame 不缺失 ✅
- WA2 - SessionService：提供 SendCommandAndAwait ✅
- WA3 - Auth：Register/Login 使用 send+await ✅
- WA4 - 回归测试 ✅（`go test ./... -count=1 -p 1`）
- WA5 - Code Review + 归档 ✅

## 关键设计决策与权衡
- **不改前端 store**：通过 SDK onFrame hook 维持既有 `session.frame` 事件语义，减少联动改动面。
- **默认 timeout 仅用于 Simple 方法**：保留 `Register/Login(ctx)` 的可控性，避免强行覆盖调用方超时策略。
- **失败也保持协议 wire 不变**：仍由 `data.code/msg` 表达错误（Header Major 不作为业务失败语义）。

## Code Review（结论：通过）
- 需求覆盖：通过（Auth send+await；timeout=8s；code!=1 返回 error；frame 事件不缺失）
- 架构合理性：通过（Awaiter 复用 SDK；Win 仍为应用层；等待语义不在 Win 分叉）
- 性能风险：通过（Auth 频率低；onFrame 与既有 session onFrame 等价；无额外重复 JSON 解码在热路径）
- 可读性与一致性：通过（API 命名清晰；错误路径明确；日志与行为一致）
- 可扩展性与配置化：通过（后续可将其它子协议小步迁移到 `SendCommandAndAwait`）
- 稳定性与安全：通过（ctx timeout/cancel 清晰；Close 会唤醒等待者避免泄漏）
- 测试覆盖情况：通过（Go 全量回归通过；端到端冒烟见下）

## 测试与验证方式 / 结果
### 自动回归（Windows）
```powershell
$env:GOTMPDIR='d:\\project\\MyFlowHub3\\.tmp\\gotmp'
New-Item -ItemType Directory -Force -Path $env:GOTMPDIR | Out-Null
go test ./... -count=1 -p 1
```
结果：通过。

### 冒烟验证（手动）
1) 启动 `hub_server`
2) Win 点击 Connect
3) Home 页面点击 Register 或 Login
   - 期望：调用在 8s 内返回（失败返回 error）
   - 期望：前端仍能收到 `session.frame` 并更新 `Logged In/Last message`

## 潜在影响与回滚方案
### 潜在影响
- `RegisterSimple/LoginSimple` 现在会等待响应（最多 8s），并在失败时返回 error（前端会进入 catch）。

### 回滚方案
- 回滚 Auth await：`git revert 1b61562`
- 回滚 Session 接入 await：`git revert 5d16f4a` + `git revert 0f5c386`
- 回滚本 workflow 文档：`git revert 290fca8`

