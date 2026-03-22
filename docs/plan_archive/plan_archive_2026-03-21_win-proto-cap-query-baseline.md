# Win 构建链路阻塞修复

## Workflow 信息
- 主仓库：MyFlowHub-Win
- 主分支：fix/win-vite-build
- Base：main
- 主 Worktree：D:\project\MyFlowHub3\worktrees\fix-win-vite-build\MyFlowHub-Win
- 当前阶段：4. 归档变更（已完成）

## 跨仓信息
- 参与仓库 1：
  - 仓库：MyFlowHub-Win
  - 分支：fix/win-vite-build
  - Base：main
  - Worktree：D:\project\MyFlowHub3\worktrees\fix-win-vite-build\MyFlowHub-Win
  - 责任边界：消费对齐后的 Proto 协议，修复 Win 构建链路并完成验证。
- 参与仓库 2：
  - 仓库：MyFlowHub-Proto
  - 分支：fix/proto-exec-cap-query
  - Base：main
  - Worktree：D:\project\MyFlowHub3\worktrees\fix-proto-exec-cap-query\MyFlowHub-Proto
  - 责任边界：补齐/确认 `protocol/exec` 中 `cap_query` 相关协议定义，形成 Win 可依赖的基线。
- 依赖关系：
  - `MyFlowHub-Proto` 先完成协议基线对齐。
  - `MyFlowHub-Win` 再对齐依赖并验证 `wails build`。

## 当前状态
- 已创建独占分支与 worktree。
- 已完成阶段 1 需求分析。
- 已完成阶段 2 架构设计分析。
- 因依赖版本策略未确认，阻塞进入 3.1 / 3.2。

## 1. 需求分析

### 目标
- 修复 `MyFlowHub-Win` 当前构建链路阻塞，恢复 `wails build` 可执行性。

### 范围
- 必须：
  - 查明用户看到的 `vite` 报错是否为真实根因。
  - 修复当前构建阻塞点，使 Wails 的 bindings 生成与后续前端构建具备通过条件。
  - 保持现有 Flow 能力查询功能不被静默删除。
- 可选：
  - 顺带验证 `npm run build` 与 `wails build -debug -skipembedcreate -nopackage`。
- 不做：
  - 不扩展新的 Flow/Exec 功能。
  - 不重构无关前端页面。
  - 不回退用户在主仓中的已有本地修改。

### 使用场景
- 开发者在 `MyFlowHub-Win` 执行 `wails build`。
- Wails 先生成 Go bindings，再安装/构建 frontend。
- 需要保证当前 Flow 编辑器的能力查询接口仍可被前端调用。

### 功能需求
- Win 端 `FlowService.ExecCapQuery*` 必须能与当前依赖的协议定义匹配。
- `go.mod` 的协议依赖策略必须与代码实现一致。
- 构建输出应不再被当前已定位的编译错误阻塞。

### 非功能需求
- 变更最小化。
- 依赖策略可审计、可交接。
- 尽量保持构建可复现性，避免引入隐式环境耦合。

### 输入输出
- 输入：
  - `wails build` 构建命令。
  - 当前 `go.mod` 依赖版本。
  - 当前 Win/Proto 本地仓状态。
- 输出：
  - 可执行的依赖对齐方案。
  - 修复后的构建验证结果。

### 边界异常
- 若 `MyFlowHub-Proto` 尚无正式发布版本包含 `CapQuery*`，则无法直接靠现有 semver 版本修复。
- 若选择本地 `replace`，构建将依赖当前工作区存在 `MyFlowHub-Proto` 仓库。
- 若选择未发布 commit/pseudo-version，需要明确采用哪一个 commit 作为基线。

### 验收标准
- `wails build -debug -skipembedcreate -nopackage` 不再因 `protocolexec.CapQuery*` 缺失而失败。
- `npm run build` 不再出现由 `wailsjs` 未生成导致的假性阻塞链路。
- 变更方案在文档中可说明“为何这样依赖”。

### 风险
- 依赖未发布协议会破坏 semver 可复现性。
- 临时本地 `replace` 会把工作区结构耦合进构建链路。

### 问题清单
- 阻塞：否
- 已确认：用户要求“先对齐 Proto”，本 workflow 采用 Proto 基线对齐方案，而不是临时本地 `replace`。

## 2. 架构设计（分析）

### 总体方案
- 事实结论：
  - 用户看到的 `vite` 报错不是稳定根因。
  - 顺序执行 `npm install` 后，`vite` 可正常启动。
  - 实际阻塞点是 Go bindings 生成阶段：`internal/services/flow/service.go` 引用了 `protocolexec.CapQueryReq/Resp` 与 `ActionCapQuery*`，但当前 `go.mod` 中的 `github.com/yttydcs/myflowhub-proto v0.1.1` 不包含这些定义。
- 方案候选：
  - 方案 A：在 `go.mod` 中增加本地 `replace github.com/yttydcs/myflowhub-proto => ../MyFlowHub-Proto`
    - 优点：最快恢复当前 workspace 构建；与本地代码事实一致。
    - 缺点：破坏“单仓 clone 可复现构建”；依赖本地目录结构。
  - 方案 B：升级到包含 `CapQuery*` 的正式发布版本或指定 pseudo-version
    - 优点：保持依赖可复现、可审计。
    - 缺点：当前本地 `MyFlowHub-Proto` 尚未看到新的 tag，需要先明确发布版本或 commit 基线。
- 最终采用：
  - 方案 B。
  - 基线 commit：`7eef50dcc471db88d00cb15d9a5b5f3acc0fe1ad`
  - 对应可解析版本：`v0.1.2-0.20260318063708-7eef50dcc471`

### 模块职责
- `internal/services/flow/service.go`
  - Win 对 Flow/Exec 能力查询的 Go 侧入口。
- `go.mod`
  - 决定 Win 依赖哪一版 Proto 协议定义。
- `frontend/src/stores/flow.ts`
  - 前端调用 `ExecCapQuerySimple`，不应被静默删掉。

### 数据 / 调用流
- `frontend/src/stores/flow.ts` -> `window.go.flow.FlowService.ExecCapQuerySimple`
- `internal/services/flow/service.go` -> `protocolexec.CapQueryReq/Resp`
- `go.mod` 解析 `github.com/yttydcs/myflowhub-proto`
- `wails build` 在生成 bindings 时先编译 Go，因此先暴露协议类型缺失

### 接口草案
- 保持现有 `ExecCapQuerySimple(sourceID, targetID, req)` 接口不变。
- 仅调整其底层协议依赖来源。

### 错误与安全
- 不通过删除能力查询代码来规避编译错误，避免功能回退。
- 构建错误必须通过依赖对齐解决，而不是以跳过 bindings/跳过 frontend 方式掩盖。

### 性能与测试策略
- 性能影响极低，核心是依赖对齐，无额外运行时请求。
- 验证命令：
  - `npm run build`
  - `wails build -debug -skipembedcreate -nopackage`

### 可扩展性设计点
- 若后续发布新 proto tag，优先收敛回 semver 依赖。
- 若本次采用 `replace`，后续需有独立 workflow 收敛为正式版本。

## 进入下一阶段条件
- 已满足，可进入 3.2。

## 3.1 计划拆分（Checklist）

### 项目目标与当前状态
- 目标：恢复 `MyFlowHub-Win` 的 Wails 构建链路，保持 Flow 能力查询功能不回退。
- 当前状态：
  - Win 当前依赖 `myflowhub-proto v0.1.1`，缺失 `exec cap_query` 协议类型。
  - Proto 已存在可用 commit `7eef50d`，其 pseudo-version 为 `v0.1.2-0.20260318063708-7eef50dcc471`。

### 可执行任务清单

- [x] `PROTO-BASELINE`
  - Owner：主Agent
  - Worktree：D:\project\MyFlowHub3\worktrees\fix-proto-exec-cap-query\MyFlowHub-Proto
  - Plan：D:\project\MyFlowHub3\worktrees\fix-proto-exec-cap-query\MyFlowHub-Proto\plan.md
  - 目标：确认并记录 Win 依赖的 Proto 基线为 commit `7eef50d` / pseudo-version `v0.1.2-0.20260318063708-7eef50dcc471`。
  - 涉及模块/文件：`protocol/exec/types.go`、`docs/change/*`
  - Write set：`docs/change/**`
  - 验收条件：
    - 明确记录该基线已包含 `CapQueryReq/Resp` 与 `ActionCapQuery*`。
  - 测试点：
    - `go list -m -json github.com/yttydcs/myflowhub-proto@7eef50d`
  - 回滚点：
    - 删除本次 Proto 文档变更即可。
  - 风险与注意事项：
    - 不修改协议 wire，只做基线确认。

- [x] `WIN-MOD-ALIGN`
  - Owner：主Agent
  - Worktree：D:\project\MyFlowHub3\worktrees\fix-win-vite-build\MyFlowHub-Win
  - Plan：D:\project\MyFlowHub3\worktrees\fix-win-vite-build\MyFlowHub-Win\plan.md
  - 目标：将 Win 的 `go.mod/go.sum` 对齐到 Proto pseudo-version。
  - 涉及模块/文件：`go.mod`、`go.sum`
  - Write set：`go.mod`、`go.sum`
  - 依赖：`PROTO-BASELINE`
  - 验收条件：
    - `go.mod` 中 `github.com/yttydcs/myflowhub-proto` 升级为 `v0.1.2-0.20260318063708-7eef50dcc471`。
  - 测试点：
    - `go test ./... -count=1 -p 1`
  - 回滚点：
    - 回退 `go.mod/go.sum`。

- [x] `WIN-BUILD-VERIFY`
  - Owner：主Agent
  - Worktree：D:\project\MyFlowHub3\worktrees\fix-win-vite-build\MyFlowHub-Win
  - Plan：D:\project\MyFlowHub3\worktrees\fix-win-vite-build\MyFlowHub-Win\plan.md
  - 目标：验证 bindings 生成、前端构建与 Wails 构建链路。
  - 涉及模块/文件：无代码写集（验证任务）
  - Write set：无
  - 依赖：`WIN-MOD-ALIGN`
  - 验收条件：
    - `wails build -debug -skipembedcreate -nopackage` 通过。
  - 测试点：
    - `npm run build`
    - `wails build -debug -skipembedcreate -nopackage`
  - 回滚点：
    - 无；若失败则回到 `WIN-MOD-ALIGN` 排查。

- [x] `WIN-ARCHIVE`
  - Owner：主Agent
  - Worktree：D:\project\MyFlowHub3\worktrees\fix-win-vite-build\MyFlowHub-Win
  - Plan：D:\project\MyFlowHub3\worktrees\fix-win-vite-build\MyFlowHub-Win\plan.md
  - 目标：归档本次 Win 构建链路修复结果。
  - 涉及模块/文件：`docs/change/*`
  - Write set：`docs/change/**`
  - 依赖：`WIN-BUILD-VERIFY`
  - 验收条件：
    - 归档文档说明依赖基线、验证结果与回滚方案。
  - 测试点：
    - 文档可脱离对话独立理解。
  - 回滚点：
    - 删除对应文档。

### 并行性评估
- 结论：本轮 3.2 不使用子Agent。
- 原因：
  - 关键路径是单一依赖对齐与整体验证，强耦合且结果立即影响下一步。
  - `MyFlowHub-Proto` 本轮主要是基线确认与文档记录，不存在值得并行拆分的独立实现写集。

## 完成情况
- 已完成：Proto 基线确认、Win 依赖升级、Go/前端/Wails 构建验证、归档文档。
