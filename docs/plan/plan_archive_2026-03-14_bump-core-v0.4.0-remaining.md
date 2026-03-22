# Plan - 剩余下游对齐到 Core v0.4.0

## Workflow 信息
- Workflow 根目录：`d:\project\MyFlowHub3\worktrees\chore-bump-core-v0.4.0-remaining`
- 分支：
  - `repo/MyFlowHub-SubProto`：`chore/bump-core-v0.4.0-remaining`
  - `repo/MyFlowHub-MetricsNode`：`chore/bump-core-v0.4.0-remaining`
  - `repo/MyFlowHub-Win`：`chore/bump-core-v0.4.0-remaining`
- 基线分支：
  - `MyFlowHub-SubProto`：`main`
  - `MyFlowHub-MetricsNode`：`main`
  - `MyFlowHub-Win`：`main`

## 1. 需求分析

### 目标
- 将本轮剩余下游仓库统一对齐到 `github.com/yttydcs/myflowhub-core v0.4.0`。
- 在不引入计划外功能开发的前提下，完成依赖升级、验证、Code Review、归档、merge、push 与发布。

### 范围
#### 必须
- `MyFlowHub-SubProto`
  - 仅处理直接依赖 `myflowhub-core` 的 8 个 module：
    - `auth`
    - `exec`
    - `file`
    - `flow`
    - `forward`
    - `management`
    - `topicbus`
    - `varstore`
  - 版本策略：
    - `auth/file/management/varstore`：`v0.1.2 -> v0.1.3`
    - `exec/flow/forward/topicbus`：`v0.1.0 -> v0.1.1`
  - `broker` 不纳入本轮

- `MyFlowHub-MetricsNode`
  - 主模块与子模块统一对齐到：
    - `myflowhub-core v0.4.0`
    - `myflowhub-sdk v0.1.4`
  - 发布策略：
    - 不新建 semver tag
    - 通过 merge/push 到 `main` 触发现有 `debug-latest` 流程

- `MyFlowHub-Win`
  - 对齐到：
    - `myflowhub-core v0.4.0`
    - `myflowhub-sdk v0.1.4`
  - 发布目标：`v0.0.2`

#### 可选（条件触发）
- 若单纯升级版本后出现编译 / 测试失败，可在对应仓库做最小必要兼容修复。
- 若 `go mod tidy` 引发必要 `go.sum` 更新，允许一并提交。

#### 不做
- 不修改协议 wire 语义、子协议业务行为、MetricsNode/Win 功能逻辑。
- 不处理 `MetricsNode` 主仓现有脏工作区中的无关前端生成文件。
- 不纳入 `Core/Server/SDK/Android`（这些已在前一轮完成）。

### 使用场景
- `SubProto` 各 module 可以被上游仓库通过新 tag 正常拉取。
- `MetricsNode` 和 `Win` 可以依赖 `Core v0.4.0` 与 `SDK v0.1.4` 的发布版本完成构建与测试。
- `MetricsNode` 仍沿用现有 `debug-latest` 发布模式。

### 功能需求
- 更新三个仓库的 `go.mod` / `go.sum`。
- 为 `SubProto` 的 8 个目标 module 发布新 tag。
- 为 `Win` 发布 `v0.0.2`。
- `MetricsNode` 不打 semver tag，仅 push 到 `main`。

### 非功能需求
- 变更最小化，仅做依赖升级和必要兼容修复。
- 优先保证可维护性、版本一致性、真实可拉取依赖可用。
- 验证优先使用 `GOWORK=off`。

### 输入输出
- 输入：
  - `MyFlowHub-Core` 已发布 `v0.4.0`
  - `MyFlowHub-SDK` 已发布 `v0.1.4`
  - 目标仓库当前主分支状态与现有 tag 习惯
- 输出：
  - `SubProto / MetricsNode / Win` 完成版本对齐、测试、归档与发布

### 边界异常
- 远端网络异常导致 push / tag 失败
- `SubProto` 中某个 module 升级后暴露兼容问题
- `MetricsNode` / `Win` 升级 `Core v0.4.0` 后暴露旧接口假设

### 验收标准
- `SubProto` 目标 8 个 module 均升级到 `Core v0.4.0` 并发布对应 tag
- `MetricsNode` 升级到 `Core v0.4.0`、`SDK v0.1.4`，测试通过，push 到 `main`
- `Win` 升级到 `Core v0.4.0`、`SDK v0.1.4`，测试通过，发布 `v0.0.2`
- 各仓 `docs/change` 与 workflow 级归档文档齐备

### 风险
- `SubProto` 为多 module 单仓，版本发布粒度更细，tag 数量较多，容易漏发
- `MetricsNode` 当前主仓有未提交改动，必须坚持只在独立 worktree 中操作
- `Win` 依赖 Wails，若 Go 版本或依赖升级引发生成物差异，需要谨慎控制变更范围

### 结论
- 阻塞：否

## 2. 架构设计（分析）

### 总体方案（含选型理由 / 备选对比）
- 采用“先协议模块、再终端应用”的三段式收口方案：
  1. `SubProto`
  2. `MetricsNode`
  3. `Win`
- 选型理由：
  - `SubProto` 是协议模块层，先完成有利于后续应用层验证
  - `MetricsNode` 和 `Win` 都依赖 `Core/SDK`，但相互独立，可在 `SubProto` 收口后分别处理
- 备选方案：
  - 三仓同时升级
  - 不采用，因为 `SubProto` 的多 module tag 发布本身就需要单独核对，适合先独立收口

### 模块职责
- `MyFlowHub-SubProto`
  - 对齐 `Core v0.4.0`
  - 发布 8 个 module 的新 tag
- `MyFlowHub-MetricsNode`
  - 对齐 `Core v0.4.0` 与 `SDK v0.1.4`
  - 作为应用仓，继续沿用 `debug-latest`
- `MyFlowHub-Win`
  - 对齐 `Core v0.4.0` 与 `SDK v0.1.4`
  - 作为应用仓发布 `v0.0.2`

### 数据 / 调用流
- 版本流：
  - `Core v0.4.0`
  - `SDK v0.1.4`
  - `SubProto` 各 module 新 tag
  - `MetricsNode` push `main`
  - `Win v0.0.2`
- 验证流：
  - 先 `go mod tidy`
  - 再 `GOWORK=off go test ./...`
  - 通过后 merge / push / tag

### 接口草案
- 无新增业务接口
- 仅升级依赖：
  - `github.com/yttydcs/myflowhub-core v0.4.0`
  - `github.com/yttydcs/myflowhub-sdk v0.1.4`

### 错误与安全
- 若远端 tag 不可拉取，禁止继续应用层对齐
- 若某仓测试失败，仅允许最小必要兼容修复
- `MetricsNode` 主仓脏工作区不参与本轮，避免误提交生成物

### 性能与测试策略
- 本轮不主动修改运行时行为，性能风险来自依赖升级后的接口兼容
- 测试策略：
  - `SubProto`：对 8 个目标 module 逐个 `GOWORK=off go test ./... -count=1`
  - `MetricsNode`：主模块与必要子模块测试
  - `Win`：`GOWORK=off go test ./... -count=1`

### 可扩展性设计点
- 继续沿用“Core 先发版、协议层收口、应用层收口”的模式
- `SubProto` 维持多 module 分标签发布，不引入单仓统一版本号
- `MetricsNode` 保持 debug-latest，避免仓促引入不成熟 semver 体系

### 结论
- 阻塞：否

## 3.1 计划拆分（Checklist）

### REM-1 - SubProto 8 个模块升级到 Core v0.4.0
- 目标：将 `auth/exec/file/flow/forward/management/topicbus/varstore` 的 `Core` 依赖统一升级到 `v0.4.0`
- 涉及模块 / 文件：
  - `repo/MyFlowHub-SubProto/*/go.mod`
  - `repo/MyFlowHub-SubProto/*/go.sum`
  - `repo/MyFlowHub-SubProto/docs/change/2026-03-14_bump-core-v0.4.0-subproto.md`
- 验收条件：
  - 8 个 module 全部升级完成
  - 对应模块测试通过
  - 新 tag 全部创建并 push
- 测试点：
  - 每个目标 module `GOWORK=off go test ./... -count=1`
- 回滚点：
  - 回退对应 module 升级提交，并停止该 module 发布

### REM-2 - MetricsNode 对齐 Core v0.4.0 / SDK v0.1.4
- 目标：升级 `MetricsNode` 主模块及其相关子模块依赖
- 涉及模块 / 文件：
  - `repo/MyFlowHub-MetricsNode/go.mod`
  - `repo/MyFlowHub-MetricsNode/go.sum`
  - `repo/MyFlowHub-MetricsNode/nodemobile/go.mod`
  - `repo/MyFlowHub-MetricsNode/nodemobile/go.sum`
  - `repo/MyFlowHub-MetricsNode/windows/go.mod`
  - `repo/MyFlowHub-MetricsNode/windows/go.sum`
  - `repo/MyFlowHub-MetricsNode/docs/change/2026-03-14_bump-core-v0.4.0-metricsnode.md`
- 验收条件：
  - `core/sdk` 版本全部对齐
  - 测试通过
  - merge/push 到 `main`
- 测试点：
  - 主模块 `GOWORK=off go test ./... -count=1`
  - 必要子模块测试通过
- 回滚点：
  - 回退本任务相关提交

### REM-3 - Win 对齐 Core v0.4.0 / SDK v0.1.4
- 目标：升级 `Win` 依赖到新的 `Core/SDK`
- 涉及模块 / 文件：
  - `repo/MyFlowHub-Win/go.mod`
  - `repo/MyFlowHub-Win/go.sum`
  - `repo/MyFlowHub-Win/docs/change/2026-03-14_bump-core-v0.4.0-win.md`
- 验收条件：
  - `core/sdk` 版本升级完成
  - 测试通过
  - merge/push 到 `main` 并发布 `v0.0.2`
- 测试点：
  - `GOWORK=off go test ./... -count=1`
- 回滚点：
  - 回退本任务相关提交

### REM-4 - 条件兼容修复
- 目标：若版本升级后暴露接口兼容问题，仅做最小必要修复
- 涉及模块 / 文件：
  - 以测试失败定位结果为准
- 验收条件：
  - 修复范围仅限兼容性
  - 不扩展为新功能开发
- 测试点：
  - 原失败项通过
- 回滚点：
  - 独立回退兼容修复提交

### REM-5 - 回归验证、Code Review、归档与发布
- 目标：完成三仓验证、评审、归档、merge/push/tag
- 涉及模块 / 文件：
  - `worktrees/chore-bump-core-v0.4.0-remaining/docs/change/2026-03-14_bump-core-v0.4.0-remaining.md`
- 验收条件：
  - 各仓测试通过
  - Code Review 输出完整
  - 文档归档齐全
  - 发布动作完成
- 测试点：
  - 版本、tag、分支结果一致
- 回滚点：
  - 若未 push/tag，可回退对应 merge

## 依赖关系
- `REM-1` 是应用层收口的前置
- `REM-2` 与 `REM-3` 在 `REM-1` 完成后可独立进行
- `REM-4` 仅在升级暴露问题时启用
- `REM-5` 依赖全部实现任务完成

## 风险与注意事项
- `SubProto` 需要逐 module 打 tag，必须逐项核对，不能遗漏
- `MetricsNode` 主仓现有脏工作区必须继续绕开，不可直接修改主仓
- `Win` 若因 Wails 相关工具链产生无关生成物，必须避免将其混入提交

## 执行进展
- `REM-1`：已完成
  - 仓库：`repo/MyFlowHub-SubProto`
  - 提交：`0e69be2`
  - 结果：8 个目标 module 已升级到 `myflowhub-core v0.4.0`，逐 module `GOWORK=off go test` 通过，仓库级归档已补齐。
- `REM-2`：已完成
  - 仓库：`repo/MyFlowHub-MetricsNode`
  - 提交：`cb9e835`
  - 结果：主模块、`nodemobile`、`windows` 已对齐到 `core v0.4.0` / `sdk v0.1.4`，验证通过，仓库级归档已补齐。
- `REM-3`：已完成
  - 仓库：`repo/MyFlowHub-Win`
  - 提交：`4212cfe`
  - 结果：已对齐到 `core v0.4.0` / `sdk v0.1.4`，`GOWORK=off go test` 通过，仓库级归档已补齐。
- `REM-4`：已完成
  - 结果：未发现需要提交的额外兼容修复；依赖升级后现有代码可通过验证。
- `REM-5`：已完成（待用户确认是否结束 workflow 后执行 merge / push / tag / 清理）
  - 结果：三仓回归验证通过，Code Review 已完成，workflow 级归档已补齐。

## 当前状态
- 阶段：`4 归档变更`
- 阻塞：否
- 待办：
  - 用户确认：`是否结束本次 workflow？`
  - 若确认结束：执行 merge / push / tag / 下游版本发布 / worktree 清理
