# 2026-03-01 - Win：VarPool 订阅后实时更新（接收 MajorCmd 通知帧）

## 变更背景 / 目标

在 `MyFlowHub-Win` 的 VarPool 页面中，用户对变量点击 **Subscribe** 后，变量值在 Hub 侧发生变化时应当自动刷新。

现状：订阅状态可成功建立，但 UI 端只有手动点击 `Refresh`（主动 `get`）时才会更新。

根因：Hub（VarStore 子协议）对订阅者推送变更通知（例如 `action=var_changed/var_deleted/notify_set`）使用的是 `HeaderTcp.Major=MajorCmd`；而 Win 侧 `VarPoolService` 仅监听 `MajorMsg`，导致通知帧被过滤掉。

## 具体变更内容

### 修改
- `internal/services/varpool/events.go`
  - VarPool 的帧监听从“仅接受 `MajorMsg`”调整为“接受 `MajorCmd` 与 `MajorMsg`”，并继续保持 `SubProto=VarStore` + action 白名单过滤，仅处理：
    - 变更：`notify_set` / `up_set` / `var_changed`
    - 删除：`notify_revoke` / `up_revoke` / `var_deleted`

### 新增
- `internal/services/varpool/events_test.go`
  - 新增单测覆盖：`MajorCmd` 的 `var_changed` / `var_deleted` 帧能触发 `varpool.changed` / `varpool.deleted` 事件。

## plan.md 任务映射

- V1：修复 VarPool 事件过滤（支持 `MajorCmd` 通知帧）
- V2：补齐单元测试（关键链路）
- V3：构建与冒烟验证（Go test + Wails build）

## 关键设计决策与权衡

- **只修 Win 客户端，不改 Server/Hub 行为**：避免影响协议 wire 与跨仓行为；以最小改动修复消费端过滤逻辑。
- **兼容 `MajorMsg`**：保留对可能存在的旧实现/其它实现的兼容性。
- **继续使用 action 白名单**：即使放宽 Major 过滤，也只解析 VarStore 的通知类 action，避免误处理其它 Cmd 帧带来的额外开销与风险。

## 测试与验证方式 / 结果

### Go 单测

在 `MyFlowHub-Win` 仓库根目录执行：

```powershell
$env:GOWORK='off'
go test ./... -count=1 -p 1
```

结果：通过。

### Windows 构建链路

```powershell
$env:GOWORK='off'
wails build -nopackage
```

结果：通过。

### 手工冒烟（建议）

1) 启动 Hub + MetricsNode（持续更新 `sys_volume_percent`）。
2) Win 打开 VarPool，添加 Watch 并点击 Subscribe。
3) 在系统中调整音量，观察 VarPool 对应变量值无需点击 Refresh 即自动变化。

## 潜在影响与回滚方案

- 影响范围：仅 Win 客户端 VarPool 订阅通知消费逻辑；不改协议、不改 Server。
- 回滚：回滚本次变更提交即可恢复旧行为。

