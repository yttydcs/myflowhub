# Todo - 控制仓：清理残留 worktree / workflow 目录

## Workflow 信息
- 控制仓分支：`chore/cleanup-residual-worktrees`
- 当前 worktree：`d:\project\MyFlowHub3`
- 性质：控制面 / worktree 管理，不涉及业务实现代码改动
- 约束：
  - 仅清理已确认可丢弃的残留 worktree / workflow 目录
  - 不触碰当前仍作为长期工作区使用的：
    - `worktrees/MyFlowHub-Core`
    - `worktrees/MyFlowHub-Proto`
    - `worktrees/MyFlowHub-Server`

## 1. 需求分析

### 目标
- 清理当前已经失效、且不再需要保留的残留 git worktree 和旧 workflow 目录，降低控制仓噪音与误判风险。

### 范围
#### 必须
- 清理以下 git worktree：
  - `worktrees/fix-subproto-varstore-action-regression`
  - `worktrees/fix-android-release-gomobile-pin/MyFlowHub-Android`
  - `worktrees/release-auth-route-index-heal/repo/MyFlowHub-Server`
  - `worktrees/release-auth-route-index-heal/repo/MyFlowHub-Android`
  - `worktrees/release-auth-route-index-heal/repo/MyFlowHub-MetricsNode`
- 清理以下旧 workflow 目录（不再作为 git worktree 使用）：
  - `worktrees/chore-rfcomm-release-deps`
  - `worktrees/feat-showcase-var-quickpick`
  - `worktrees/refactor-transport-pipe`
  - `worktrees/release-auth-route-index-heal`

#### 可选
- 若清理后出现空父目录，可一并删除空目录。

#### 不做
- 不删除当前仍在使用的长期工作区：
  - `worktrees/MyFlowHub-Core`
  - `worktrees/MyFlowHub-Proto`
  - `worktrees/MyFlowHub-Server`
- 不删除任何远端分支 / tag。
- 不回收仓库 `repo/*` 的主工作区。

### 使用场景
- 控制仓后续继续开展 workflow 时，不会被旧 worktree / 旧 workflow 目录干扰。
- `git worktree list` 输出更准确，避免误把历史残留当成活跃任务。

### 功能需求
- 删除已确认可丢弃的残留 git worktree。
- 删除已完成归档、仅剩目录壳的旧 workflow 目录。
- 保证删除后 `git worktree list` 与文件系统状态一致。

### 非功能需求
- 安全性优先：仅清理已确认可丢弃、且不再作为有效工作区使用的残留项。
- 可审计：在变更文档中记录删除对象、判断依据、验证结果与回滚方式。
- 变更最小化：不扩大到分支整理、历史重写、远端清理。

### 输入输出
- 输入：
  - 当前 `worktrees/` 目录
  - 各仓库 `git worktree list`
  - 各残留路径的存在性与用途判断
  - 用户确认：未提交改动可全部丢弃
- 输出：
  - 控制仓只保留当前仍有效的 worktree / workflow 目录
  - 对应清理归档文档

### 边界异常
- 某 worktree 存在未提交改动或独有提交，不能直接删除
- 某 workflow 根目录仍包含未归档文档，需要先确认是否保留
- shell 工具策略拦截批量删除，需要拆分为单步执行

### 验收标准
- 目标 git worktree 全部成功移除并 `prune`
- 目标旧 workflow 目录全部从文件系统移除
- `worktrees/MyFlowHub-Core`、`worktrees/MyFlowHub-Proto`、`worktrees/MyFlowHub-Server` 仍保留
- 形成归档文档并完成 Code Review

### 风险
- 删除动作不可逆，只能依赖 git 历史或重新创建 worktree 恢复
- 若误删仍有用途的 workflow 目录，后续审计信息可能不完整

### 结论
- 阻塞：否

## 2. 架构设计（分析）

### 总体方案（含选型理由 / 备选对比）
- 采用“两阶段清理”：
  1. 先删除 git 注册中的残留 worktree
  2. 再删除仅剩控制面文件的旧 workflow 目录
- 选型理由：
  - 先清理 git worktree，可避免目录仍被 git 占用导致后续删目录失败
  - 再清理目录壳，更容易验证“git 状态”和“文件系统状态”都已收敛

### 模块职责
- 控制仓根：
  - 负责记录本次 workflow 的 `todo.md`、归档文档、全局 `plan.md` 收敛
- 各仓 `repo/*`：
  - 仅负责执行对应 `git worktree remove/prune`
- 文件系统：
  - 删除已无 git 绑定的旧 workflow 根目录

### 数据 / 调用流
1. 读取 `worktrees/` 与各仓 `git worktree list`
2. 核对候选项是否无独有提交或已明确允许丢弃
3. 执行 `git worktree remove --force`
4. 执行 `git worktree prune`
5. 删除空壳 workflow 目录
6. 重新核对 `git worktree list` 与 `Test-Path`

### 接口草案
- 无新增业务接口
- 仅使用现有命令：
  - `git worktree list`
  - `git worktree remove --force`
  - `git worktree prune`
  - 文件系统目录删除

### 错误与安全
- 对存在未提交改动的 worktree，仅在用户明确允许丢弃后才使用 `--force`
- 对长期工作区显式列入保留名单，避免误删

### 性能与测试策略
- 清理任务以元数据操作为主，无性能瓶颈
- 验证策略：
  - 清理前后分别执行 `git worktree list`
  - 使用 `Test-Path` / `Get-ChildItem` 核对目录是否已删除

### 可扩展性设计点
- 后续如需批量清理其它残留 worktree，可复用同一判断流程：
  - 先判定是否仍被 git 注册
  - 再判定是否需要保留
  - 最后执行 remove / prune / 目录删除

### 结论
- 阻塞：否

## 3.1 计划拆分（Checklist）

### WTCL-1 - 复核候选清单与安全边界
- 目标：最终确认待删除对象与保留对象
- 结果：已完成；用户已明确允许丢弃未提交改动。

### WTCL-2 - 删除残留 git worktree
- 结果：已完成。

### WTCL-3 - 删除旧 workflow 目录壳
- 结果：已完成。

### WTCL-4 - 回归验证与状态盘点
- 结果：已完成；仅保留 `MyFlowHub-Core` / `MyFlowHub-Proto` / `MyFlowHub-Server`。

### WTCL-5 - Code Review + 归档变更
- 结果：已完成。

## 最终状态
- 阶段：`4 归档变更`
- 阻塞：否
- 结果：本 workflow 已完成，待用户确认是否结束 workflow。
