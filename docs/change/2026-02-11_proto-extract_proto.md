# 2026-02-11 协议定义抽离到 MyFlowHub-Proto（PR1）

## 背景 / 目标
- 将协议定义（SubProto / Action / JSON struct）从 `MyFlowHub-Server` 抽离为独立仓库 `MyFlowHub-Proto`，便于多端复用与未来“按子协议裁切/组装”。
- **wire 保持不变**：action 名称、消息 JSON 结构、SubProto 值不变（本轮策略 A）。
- `MyFlowHub-Proto` 作为“纯协议字典”：仅依赖标准库，不依赖 Core/Server/Win，避免循环依赖。

## 具体变更内容（新增 / 修改 / 删除）
### 新增
- `go.mod`
  - module：`github.com/yttydcs/myflowhub-proto`
  - `go 1.23.0` / `toolchain go1.24.5`（与现有仓库对齐）
- 协议包（均为类型/常量定义，wire 不变）：
  - `protocol/auth`
  - `protocol/varstore`
  - `protocol/topicbus`
  - `protocol/file`
  - `protocol/flow`
  - `protocol/management`
  - `protocol/exec`（新增子协议定义：`SubProtoExec=7`，`call/call_resp`，权限 `exec.call`）
- `plan.md`

### 修改
- 无（新仓库初始化）

### 删除
- 无

## plan.md 任务映射
- P1：初始化 go.mod ✅
- P2：迁移现有协议包（wire 不变）✅
- P3：新增 exec 协议包 ✅
- P4：全量回归 ✅

## 关键设计决策与权衡（性能 / 扩展性）
- 策略 A：短期不做 `assist_* / up_* / notify_*` 的 wire 收敛，仅做代码层抽象与仓库拆分，降低迁移风险。
- 每个子协议一个 package：为后续“子协议独立库/可裁切”提供天然边界（PR2+ 再做编译期裁切与依赖注册收敛）。

## 测试与验证方式 / 结果
- `go test ./... -count=1`：通过。

## 潜在影响与回滚方案
### 潜在影响
- 下游仓库（Server/Win）需要切换依赖到 `myflowhub-proto`；本 PR 已在同批次同步完成（见各仓库对应归档）。
- 目前下游使用 `require v0.0.0 + replace => ../MyFlowHub-Proto`（便于本地多仓联调）；若需要仓库独立构建/CI，需要在 Proto 发布可拉取的版本（tag/伪版本）后移除 `replace`。

### 回滚方案
- 回滚下游对 `myflowhub-proto` 的引用（恢复旧协议定义来源），并删除/回退本仓库提交即可。

## 例外说明（审计）
- 为创建 `git worktree`，仓库初始化阶段在主 worktree（`repo/MyFlowHub-Proto`）进行了 initial commit；本 PR 的实现性变更在独占 worktree（`worktrees/proto-extract/MyFlowHub-Proto`）完成。

