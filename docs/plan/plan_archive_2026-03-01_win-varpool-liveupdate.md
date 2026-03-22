# Plan - MyFlowHub-Win：VarPool 订阅后实时更新（接收 MajorCmd 通知帧）

## Workflow 信息
- 范围：单仓库（`MyFlowHub-Win`）
- 分支：`fix/win-varpool-liveupdate`
- Worktree：`d:\project\MyFlowHub3\worktrees\win-varpool-liveupdate\MyFlowHub-Win`
- Base：`main`
- 规范：
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文，前缀可英文）

---

## 1) 需求分析（已确认）

### 目标
在 `MyFlowHub-Win` 的 VarPool 页面中：
- 用户点击 **Subscribe** 后，当变量值在 Hub 侧发生变化时，页面值应 **自动刷新**（无需手动点 Refresh）。

### 现状与根因（已定位）
- Hub（VarStore 子协议）向订阅者推送变更时，使用的是 `HeaderTcp.Major=MajorCmd` 的通知帧（例如 `action=var_changed` / `notify_set`）。
- `MyFlowHub-Win/internal/services/varpool/events.go` 当前只处理 `MajorMsg`，导致这些通知帧被过滤掉；前端自然只能靠手动 Refresh（主动 Get）才能看到新值。

### 范围
- 必须：
  - 修复 VarPool 的订阅通知链路：支持处理 `MajorCmd`（并保持兼容 `MajorMsg`）。
- 可选：
  - 暂无（本 workflow 只聚焦 VarPool）。
- 不做：
  - 不修改 Hub/Server 的推送行为（不改协议 wire、不改 Major 语义）。
  - 不引入额外轮询刷新策略。

### 验收标准
1) 启动 Hub + MetricsNode（或任何持续 set 变量的节点）。
2) 在 `MyFlowHub-Win` 的 VarPool 页面 Watch 某变量并点击 Subscribe。
3) 外部改变该变量（例如 MetricsNode 音量变化触发 `sys_volume_percent` 更新）。
4) `MyFlowHub-Win` 页面中该变量的值在 1-2 秒内自动变化（无需点 Refresh）。

### 风险
- 放宽 Major 过滤会让 VarPool 看到更多帧：通过 **SubProto + Action 白名单** 继续严格过滤，避免误处理非通知消息。

---

## 2) 架构设计（分析）

### 总体方案（选型）
在 Win 侧修复（最小变更、低风险）：
- VarPoolService 的事件监听不再只认 `MajorMsg`，而是接受 `MajorCmd`/`MajorMsg`，再通过 action 白名单挑出通知类 action：
  - `notify_set` / `up_set` / `var_changed`
  - `notify_revoke` / `up_revoke` / `var_deleted`

### 备选方案（不选）
- 改 Server：把通知帧改为 `MajorMsg` 再下发（影响面更大，属于协议行为调整，本 workflow 不做）。

### 关键数据流（修复后）
`session.frame`（MajorCmd, SubProto=VarStore, action=var_changed） →
`internal/services/varpool/events.go` 解析 payload →
发布 `varpool.changed`（Wails EventsEmit） →
`frontend/src/stores/varpool.ts` 更新 `state.data` →
`frontend/src/pages/VarPool.vue` 自动展示新值。

---

## 3.1) 计划拆分（Checklist）

### V1 - 修复 VarPool 事件过滤
- 目标：订阅后能接收到 `MajorCmd` 的变更通知并刷新 UI。
- 涉及文件（预期）：
  - `internal/services/varpool/events.go`
- 验收：
  - 订阅变量后，变更能触发前端值更新（见“验收标准”）。
- 回滚：
  - 回滚本 workflow 提交即可恢复旧行为。

### V2 - 单元测试（关键链路）
- 目标：覆盖“MajorCmd 的 var_changed/var_deleted 能触发 varpool.changed/varpool.deleted”。
- 涉及文件（预期）：
  - `internal/services/varpool/events_test.go`
- 验收：
  - `go test ./...` 通过。

### V3 - 构建与冒烟
- 目标：确保修复不破坏编译链路，并完成端到端手工验证。
- 验收：
  - `GOWORK=off go test ./... -count=1 -p 1`
  - `GOWORK=off wails build -nopackage`
  - 手工冒烟通过（见“验收标准”）

---

## 3.3) Code Review（完成编码后执行）
- 需求覆盖：订阅后无需 Refresh 自动更新
- 架构合理性：只改 Win 客户端过滤逻辑，不动协议 wire
- 性能风险：过滤放宽但仍以 action 白名单处理，避免无意义解析
- 可读性与一致性：条件判断清晰，测试用例覆盖关键路径
- 稳定性与安全：异常 payload 安全忽略；无额外权限变化
- 测试覆盖：新增单测 + 手工冒烟

---

## 4) 归档变更（完成 Review 后执行）
- 在仓库内新增：
  - `docs/change/2026-03-01_win-varpool-liveupdate.md`
- 内容需包含：背景/目标、变更内容、任务映射、关键决策、测试结果、回滚方案。

