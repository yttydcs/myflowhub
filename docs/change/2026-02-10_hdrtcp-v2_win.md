# 2026-02-10 HeaderTcp v2（32B）适配（Win）

## 背景 / 目标
本仓库配合 Core/Server 完成 **HeaderTcp v2（32B）big-bang** 升级，确保 Win 端可与同批次 Server 互通，并提供可执行的冒烟验证步骤。

## 具体变更内容
### 修改
- `internal/session/session.go`
  - `Session.Send()` 在发送前补齐：
    - `hop_limit==0 => DefaultHopLimit(16)`
    - `trace_id==0 => 自动生成随机 uint32`
  - 目的：与 Core `Server.Send()` 行为对齐，避免直连发送出现 `trace_id=0` 或 `hop_limit=0` 的不一致。
- `plan.md`
  - 补齐 W3：启动 HubServer + 启动 Win + Connect/Register/Login + VarPool List/Get 的可执行冒烟步骤。

### 新增 / 删除
- 无

## plan.md 任务映射
- W1：transport codec 适配 HeaderTcp v2 ✅（复用 Core `HeaderTcpCodec`）
- W2：与 Core 路由规则对齐 ✅（本次以保持协议 Major 现状为主，逐跳转发由 Server 侧兜底；后续可再做 Major 规范化）
- W3：冒烟验证步骤 ✅（已在 `plan.md` 补齐）

## 关键设计决策与权衡
- big-bang：切换后 v1 不再兼容，Win 必须与同批次 Core/Server 一起升级。
- 最小改动优先：本次仅补齐发送侧默认字段与验证步骤，不引入 UI/协议层大改动。

## 测试与验证
- `go test ./...` 通过
- 本地端到端验证（同批次 Core/Server）：
  - register/login + management node_echo 可跑通（验证 HeaderTcp v2 收发）。

## 潜在影响
- 旧 Server/Core 无法与新 Win 互通（wire 不匹配）。

## 回滚方案
- `git revert` 回退本 PR（需与 Core/Server 同步回退）。

