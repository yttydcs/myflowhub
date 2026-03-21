# 2026-02-15 - Win 接入 MyFlowHub-SDK v0（Session/Transport）（PR2-WIN-1）

## 变更背景 / 目标
为推进“彻底重构（Core/Server/子协议解耦、Win 上移为应用层）”，需要把 Win 端重复实现的客户端底层能力收敛到统一 SDK，避免各端各自维护：
- TCP Session（connect/send/readLoop）
- HeaderTcp v2 编解码链路上的默认字段补齐（trace_id/hop_limit）
- `{action,data}` JSON envelope 编码

本次变更目标（保持对上层 services/bindings 行为稳定，最小 diff）：
1) Win 引入 `github.com/yttydcs/myflowhub-sdk`（v0：`session` + `transport`）。
2) 保留 Win 内部包路径不变，仅将实现改为薄封装委托 SDK：
   - `internal/session` → `myflowhub-sdk/session`
   - `internal/services/transport` → `myflowhub-sdk/transport`

## 具体变更内容
### 修改
- `go.mod`
  - 新增依赖：`github.com/yttydcs/myflowhub-sdk`
  - 开发期联调：新增 `replace github.com/yttydcs/myflowhub-sdk => ../MyFlowHub-SDK`
- `internal/services/transport/codec.go`
  - `EncodeMessage` 改为直接委托 SDK 的 `transport.EncodeMessage`（输入校验与 wire 保持一致）
- `internal/session/session.go`
  - `Session` 改为包装 SDK `session.Session`（Connect/Close/Send 由 SDK 统一实现）
  - 保留 `Login` 兼容入口（legacy 调试用途）
  - 新增 `ErrSessionNotInitialized`：当 Session wrapper 未初始化时返回更明确的错误

## 与旧实现的可见差异（重要）
- 旧 Win 实现：调用 `Close()` 时通常会触发一次 readLoop 的解码错误，从而回调 `onError`（UI/日志会出现一次“session error”）。
- 现在使用 SDK：`Close()` 触发的解码错误会被视为“正常退出”，**不会**回调 `onError`。
  - 目的：减少正常关闭时的误报噪音。
  - 风险：如果上层逻辑依赖“关闭也会产生 error event”的副作用，需要调整；当前 Win 的 `SessionService.Close()` 已显式发布 `connected=false` 的 state event，因此不依赖该副作用。

## plan.md 任务映射
- WSDK1 - 引入 myflowhub-sdk 依赖 ✅
- WSDK2 - transport 薄封装委托 SDK ✅
- WSDK3 - session 薄封装委托 SDK ✅
- WSDK4 - 全量回归 ✅（`go test ./... -count=1 -p 1` 通过）
- WSDK5 - Code Review + 归档 ✅

## 关键设计决策与权衡
- **最小改动优先**：保留 `internal/session` 与 `internal/services/transport` 包路径，避免触及上层 `internal/services/*` 与 Wails bindings，符合“小步多 PR”。
- **统一默认字段规则**：由 SDK 统一补齐 `trace_id/hop_limit`，避免客户端侧多处复制实现导致行为漂移。
- **开发期 replace**：继续通过 `replace ../MyFlowHub-SDK` 本地联调；后续待 SDK/Proto 打 tag 再移除 replace 以支持独立 clone 构建。

## Code Review（结论：通过）
- 需求覆盖：通过（Win 已接入 SDK v0；内部实现已委托 SDK）
- 架构合理性：通过（Win 上移为应用层；底层能力收敛到 SDK，依赖方向符合 `Core/Proto/SDK/Win`）
- 性能风险：通过（读循环仍为 `bufio.Reader`；envelope 编码复用 SDK；无新增多余 I/O）
- 可读性与一致性：通过（薄封装清晰；错误更可读；diff 面可控）
- 可扩展性与配置化：通过（为后续删除兼容壳、引入 SDK v1 Awaiter 预留空间）
- 稳定性与安全：通过（输入校验保持；未初始化/未连接错误明确；Close 不误报）
- 测试覆盖情况：通过（Go 全量回归通过；后续可在 Win 冒烟流程中验证端到端）

## 测试与验证方式 / 结果
- `GOTMPDIR=d:\\project\\MyFlowHub3\\.tmp\\gotmp`
- `go test ./... -count=1 -p 1`（通过）

## 潜在影响与回滚方案
### 潜在影响
- `Close()` 不再触发 `onError` 回调（减少噪音，但属于行为差异，见上文）。
- 目前仍为本地 `replace` 联调；如在独立环境构建需要先移除 replace 并使用已发布版本。

### 回滚方案
- 可直接 revert 本 PR 对应提交：
  - revert `refactor: Win 复用 SDK v0 的 Session/Transport`
  - revert `refactor: session 未初始化错误更清晰`
  - revert `go.mod` 中的 `myflowhub-sdk` 依赖与 replace

