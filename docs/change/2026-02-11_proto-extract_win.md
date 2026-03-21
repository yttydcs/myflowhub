# 2026-02-11 协议依赖切换到 Proto（Win）

## 背景 / 目标
- Win 作为更上层应用：不“重新实现一个 server”，尽量通过统一协议/SDK 调用 server 能力。
- 本 PR 先做地基：Win 移除对 `MyFlowHub-Server` 仓库的 Go 依赖，只依赖 `MyFlowHub-Core` + `MyFlowHub-Proto`，降低耦合与同步成本。
- wire 保持不变（策略 A）。

## 具体变更内容（新增 / 修改 / 删除）
### 修改
- `go.mod`
  - 移除 `github.com/yttydcs/myflowhub-server`（require/replace）
  - 新增 `github.com/yttydcs/myflowhub-proto v0.0.0`
  - `replace github.com/yttydcs/myflowhub-proto => ../MyFlowHub-Proto`
- `internal/services/**`
  - 将 `github.com/yttydcs/myflowhub-server/protocol/*` 的 import 全量切换为 `github.com/yttydcs/myflowhub-proto/protocol/*`

### 新增
- 无

### 删除
- 无

## plan.md 任务映射
- W1：切换协议 import 到 Proto ✅
- W2：go.mod 解耦 Server 依赖 ✅
- W3：全量回归 ✅

## 关键设计决策与权衡（性能 / 扩展性）
- 本 PR 不改协议 wire，仅改 import/依赖来源，确保迁移风险最小化。
- 后续 PR2+ 可在 Win 引入更高层 SDK（Encode/Decode/Hook/队列），进一步减少对“协议细节”的直接触达。

## 测试与验证方式 / 结果
- `go test ./... -count=1 -p 1`：通过。
- 说明：当前环境并行编译可能触发 OOM；如遇到临时目录权限问题可设置 `GOTMPDIR` 指向项目内目录后重试。

## 潜在影响与回滚方案
### 潜在影响
- 本地多仓联调依赖 `replace => ../MyFlowHub-Proto`；若需要仓库独立构建/CI，需要在 Proto 发布可拉取版本后移除 `replace`。

### 回滚方案
- 回滚本 PR 提交即可恢复对 `myflowhub-server/protocol/*` 的依赖方式。

