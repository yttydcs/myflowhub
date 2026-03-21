# 2026-02-18 - Win：File(list/read_text) 改为 send+await（SDK v1 Awaiter）

## 背景 / 目标
- 现状：Win 侧 File 的 `list/read_text` 控制动作为 fire-and-forget；调用方无法同步获知业务成功/失败/超时，只能依赖 `file.list/file.text` 事件更新 UI。
- 问题：在断线/无响应/中间链路丢帧等场景下，UI 可能出现 “loading 不结束” 的体验问题（因为事件回包缺失）。
- 目标：将 `list/read_text` 升级为“发送并等待响应”，让调用方在超时内得到明确结果，同时保持既有事件链路不变（仍由 `session.frame` 解析驱动 `file.list/file.text`）。

## 具体变更内容
### 修改（后端）
- `internal/services/file/service.go`
  - `List/ReadText` 改为 `SendCommandAndAwait(..., expectAction=read_resp)`：
    - 请求仍为 `MajorCmd + SubProtoFile`，payload 仍为 `KindCtrl + JSON(action=read,data=ReadReq)`（wire 不变）
    - 等待响应 action：`read_resp`
  - `ListSimple/ReadTextSimple` 增加默认超时：`8s`（`context.WithTimeout`）
  - 业务判定规则：
    - `code == 1`：成功，返回 `nil`
    - `code != 1`：失败，返回 `error`（优先使用 `msg`，否则携带 `code`）
- `internal/services/file/transfer.go`
  - 本地 `list/read_text` 仍会发布 `file.list/file.text` 事件；当本地操作失败（code!=1）时，同步返回 error（与 send+await 语义保持一致）。

### 修改（前端兜底）
- `frontend/src/stores/file.ts`
  - `requestList/openPreview` 增加错误兜底：
    - 失败/超时/断线时确保 `listing/previewLoading` 能收敛
    - 展示错误信息，避免未处理的 Promise reject

## Plan.md 任务映射
- WFA1 - 后端：List/ReadText 改为 send+await ✅
- WFA2 - 前端：兜底处理 await reject ✅
- WFA3 - 回归测试（Windows）✅
- WFA4 - 冒烟（手动）🟨（需人工执行）
- WFA5 - Code Review + 归档变更 ✅（本文）

## 关键设计决策与权衡
- Await 匹配维度：`MsgID + SubProto(File=5) + Action(read_resp)`（统一 Awaiter 框架规则）。
- 保持事件链路不变：
  - matched frame 仍会发布 `session.frame`（依赖 SDK `SetOnFrame` 语义）
  - FileService 继续从 `session.frame` 消费并发布 `file.list/file.text/file.tasks/file.offer`
- 本 PR 仅覆盖控制类 `list/read_text`：
  - 不 await 化 `pull/offer`（传输链路保持事件驱动与任务状态机，避免范围扩大）。

## 测试与验证方式 / 结果
### 回归（Windows）
```powershell
$env:GOTMPDIR='d:\\project\\MyFlowHub3\\.tmp\\gotmp'
New-Item -ItemType Directory -Force -Path $env:GOTMPDIR | Out-Null
go test ./... -count=1 -p 1
```
结果：通过（无测试文件，编译链路通过）。

### 冒烟（手动，建议在联调环境执行）
1. Win 启动并 Connect 到 server
2. 打开 File 页面，刷新目录（list）
3. 选择文本文件预览（read_text）

验收要点：
- 两类操作均在 `8s` 内成功/失败/超时返回（Promise resolve/reject）
- UI 仍通过事件更新目录与预览；异常时 loading 能收敛并展示错误信息

## 潜在影响与回滚方案
### 潜在影响
- `list/read_text` 调用由“立即返回”变为“等待响应/超时后返回”，更利于脚本化与错误呈现。
- 若服务端 `read_resp` 未继承 `MsgID` 或 SDK 未正确解包 File CTRL，将表现为 await 超时（调用方可见，便于暴露问题）。

### 回滚方案
- revert 本 PR 提交（恢复为 fire-and-forget 的 `SendCommand`）。
