# Workflow Todo - 2026-04-14 Thesis Function Comments CN

## 当前阶段
- Stage 1 已完成
- Stage 2 已完成
- Stage 3.1 已完成
- Stage 3.2 已完成
- Stage 3.3 已完成
- Stage 4 已完成（root 归档同步与 worktree cleanup 已执行）

## 仓库与执行上下文
- Control Repo: `D:\project\MyFlowHub3`
- Control Base branch: `master`
- Control Worktree branch: `chore/thesis-function-comments-cn`
- Control Worktree path: `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-control`
- Active control doc: `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-control\todo.md`
- Participating repos:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Android`
    - Base: `main`
    - Worktree: `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-android`
    - Plan: `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-android\todo.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Core`
    - Base: `master`
    - Worktree: `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-core`
    - Plan: `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-core\todo.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-EmbeddedSDK`
    - Base: `main`
    - Worktree: `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-embeddedsdk`
    - Plan: `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-embeddedsdk\todo.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-MetricsNode`
    - Base: `main`
    - Worktree: `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-metricsnode`
    - Plan: `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-metricsnode\todo.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Proto`
    - Base: `main`
    - Worktree: `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-proto`
    - Plan: `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-proto\todo.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-SDK`
    - Base: `main`
    - Worktree: `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-sdk`
    - Plan: `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-sdk\todo.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server`
    - Base: `main`
    - Worktree: `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-server`
    - Plan: `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-server\todo.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-SubProto`
    - Base: `main`
    - Worktree: `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-subproto`
    - Plan: `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-subproto\todo.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
    - Base: `main`
    - Worktree: `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-win`
    - Plan: `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-win\todo.md`

## 使用 $m-docs 的文档路由结论
- 文档分类: `plan`
- Requirements impact: none
- Specs impact: none
- Related requirements: none
- Related specs:
  - `D:\project\MyFlowHub3\repos.md`
  - `D:\project\MyFlowHub3\docs\specs\README.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\README.md`
- Related lessons:
  - `D:\project\MyFlowHub3\docs\lessons\frontend-worktree-wailsjs-missing.md`
  - `D:\project\MyFlowHub3\docs\lessons\wails-bindings-cross-project.md`
- Canonical destination:
  - workflow 控制文档：当前 control worktree 根 `todo.md`
  - repo 执行边界：各 repo worktree 根 `todo.md`
  - 完成后的结果归档：`D:\project\MyFlowHub3\docs\change\2026-04-14_thesis-function-comments-cn.md`

## Stage 1 - 需求分析

### 目标
- 在上一轮“文件级职责注释 + 中文化”基础上，继续为 `repo/` 下 9 个仓库的非自动生成源码补函数级中文注释。
- 目标不是翻译函数名，而是让阅读者更快知道函数的职责、输入约束、状态变化、边界处理或 why。
- 继续以当前主路径的未提交状态为基线，不吞掉用户已有脏改动。

### 范围

#### Must
- 覆盖 `Android / Core / EmbeddedSDK / MetricsNode / Proto / SDK / Server / SubProto / Win` 九个仓库。
- 以当前未提交状态为基线，把该基线同步到执行 worktree 后再改。
- 对 eligible source 中的具名函数/方法尽量补简短中文注释，优先补 public API、handler、service、bridge、store、runtime、协议转换、持久化、脚本入口、测试入口。
- 注释语言统一使用中文，风格遵守各语言原生 comment style。
- 保持运行时行为不变，只做 comment-only 改动。

#### Optional
- 对类型方法、复杂测试辅助函数、脚本函数和构造流程补比“一句话职责”更强的信息，如状态前提、回退逻辑、跨仓依赖方向。
- 对一眼难懂的关键闭包前补少量 why 注释，但不追求匿名函数逐个解释。

#### Out Of Scope
- 不修改自动生成代码、二进制、构建产物、缓存和历史归档文档。
- 不把注释任务扩展成行为修复、重构、依赖升级、release 或 spec 改造。
- 不强行给一行 getter/setter、纯字面转发器、表驱动测试样本和显然匿名回调加噪声注释。

### 使用场景
- 毕设写作前，需要快速回答“这个函数在链路里负责什么”“为什么这里要这样判断”。
- 顺着 `Proto -> Core/SDK -> SubProto/Server -> Win/Android/MetricsNode` 追能力落地时，减少反复跳读。
- 阅读脚本或前端/移动端 glue code 时，降低“函数名看懂了，但上下文没懂”的成本。

### 功能需求
1. 每个参与仓库都要做一轮函数级注释审查，而不是只补文件头。
2. 优先给具名函数/方法补职责说明；复杂函数再补关键副作用、边界或 why。
3. 对测试文件只给测试入口、辅助构造器、重要断言准备逻辑补注释，不给每个 case data 加注释。
4. 对 Go/Kotlin/TS/Vue/Python/C/PowerShell 分别采用安全注释语法，避免 parser 问题。
5. 对已有上一轮中文文件头注释的文件，本轮在函数层继续增量补，不重复改同一句职责文案。
6. 对带用户脏基线的仓库，保持原有未提交改动原样存在，只在允许写集内补注释。

### 非功能需求
- 变更面尽量小，只改注释，不改代码顺序和行为。
- 注释必须可辩护，不复述函数名，不写废话。
- 多仓并行时 write set 必须隔离，防止冲突。
- 验证以语法/格式/diff-level 检查为主，不为了注释任务做高成本全量构建。

### 输入 / 输出
- Inputs:
  - `D:\project\MyFlowHub3\repo\*` 当前未提交源码基线
  - `D:\project\MyFlowHub3\guide.md`
  - `D:\project\MyFlowHub3\repos.md`
  - `D:\project\MyFlowHub3\docs\change\2026-04-13_thesis-code-comments.md`
  - `D:\project\MyFlowHub3\docs\change\2026-04-13_thesis-code-comments-cn.md`
- Outputs:
  - 9 个 repo worktree 中的函数级中文注释增量
  - control worktree / repo worktree 的 `todo.md`
  - review 记录与最终 `docs/change` 归档

### 边界异常
- `Win` 与 `MetricsNode` 含 `wailsjs` 生成绑定，必须排除，但又要保留该脏基线本身。
- `Win` 含未跟踪 `myflowhub-mcp.exe`；`SubProto` 含未跟踪 `docs/change/*.md`；都要保留但不纳入写集。
- 某些函数非常短，若硬加注释只会制造噪声，需要按“是否提升理解效率”决定是否跳过。
- 某些测试文件函数数量大，但价值集中在 helper 和入口，不能机械全加。

### 验收标准
1. 9 个仓库都完成一轮函数级中文注释审查。
2. eligible source 中大多数具名函数/方法都有简短职责说明或更高信息量注释。
3. generated/binary/build/docs 归档未被误改。
4. comment-only 变更不引入格式、语法或明显行为变更。
5. 每个 repo 都能给出清晰 write set、验证方式和回滚边界。

### 风险
- 跨仓大面积 comment-only diff 会提高后续 merge 冲突概率。
- 机械地给每个函数都写一句话，容易把仓库变成噪声源。
- 子 agent 并行时如果边界不严，容易误碰 generated 或用户基线文件。

## Stage 2 - 架构设计

### 总体方案
- 采用“沿用上一轮文件级覆盖 + 本轮函数级补充 + 按仓并行”方案。
- 选型理由：
  - 上一轮已经建立文件级职责说明，本轮直接在同一基线上补函数级注释，收益最高。
  - 按 repo 拆分 write set，最适合子 agent 并行，冲突最小。
  - 以注释密度而非全仓 build 为目标，验证重点放在语法、格式和 diff 纯度。
- 备选方案：
  - 方案 A：仅补导出函数
    - 放弃原因：用户明确希望“每个函数上都适当加一点”，只补导出函数不够。
  - 方案 B：逐文件人工串行补全
    - 放弃原因：9 仓体量过大，子 agent 并行更合理。
  - 方案 C：用模板批量生成注释
    - 放弃原因：容易生成低质量废话，违背“帮助理解代码”的目标。

### 模块职责
- Control worktree:
  - 固化 Stage 1/2/3.1、并行拆分、review 与 archive。
- Repo worktrees:
  - 只负责本仓函数级注释补充与本地验证。
- Main repo paths:
  - 仅保留 control-plane，不做实现编辑。

### 数据 / 调用流
1. 读取 workspace 约束与上一轮注释归档，确认本轮增量目标是“函数级中文注释补充”。
2. 在 control worktree 产出 Stage 1/2/3.1 和子 agent 拆分。
3. 在各 repo worktree 固化本仓 `todo.md`，锁定 write set 与排除边界。
4. 用户确认计划后进入 3.2，按 repo 分派子 agent 并行补注释。
5. 主 agent 汇总结果，统一做 diff/语法/格式校验和 review。
6. 在 control worktree 完成 archive，并等待是否结束 workflow。

### 接口草案
- 无运行时接口变更。
- 注释插入模式：
  - Go：函数前 `//`，导出与非导出一视同仁，但按信息量决定长短。
  - Kotlin：函数/方法前 `//` 或必要时 `/** */`，避免 KDoc 噪声泛滥。
  - Vue/TS：以 `script` 区域为主，在方法、store action、watcher helper 前补 `//`。
  - Python：优先函数前行注释，少量稳定 API 可用简短 docstring。
  - C：函数定义前 `/* ... */` 或连续 `//`，保持现有风格一致。
  - PowerShell：仅用 `#`，函数前补一句职责说明。

### 错误与安全
- 只允许 comment-only 变更，禁止借机改逻辑。
- 明确排除 generated、binary、build outputs 和历史 docs。
- 子 agent 只拿各自 repo worktree 的有界写集，禁止跨仓改动。

### 性能与测试策略
- 不做无意义全量 build。
- Go 仓优先 `gofmt` + `git diff --check`。
- `.ps1` 走 AST parse。
- Vue/TS/Kotlin/Python/C 以 diff 复核、基础语法级检查和 comment style 检查为主；只在必要处补最小验证。
- 额外扫描残留模板注释和未预期非注释改动。

### 可扩展性设计点
- 任务按 repo 独立，后续若需要继续补“类型级 / 字段级 / 测试 helper”注释，可直接复用这套拆分。
- `todo.md` 中保留 repo-specific write set 与验证策略，便于后续继续迭代。

## Stage 3.1 - 可执行计划

### Checklist
- [x] `CTRL-FUNC-1` 建立 control/repo worktree，并同步当前未提交基线
- [x] `CTRL-FUNC-2` 固化 Stage 1 / Stage 2 / Stage 3.1 文档
- [x] `CORE-FUNC-1` 为 `Proto / Core / SDK` 补函数级中文注释
- [x] `SERVER-FUNC-1` 为 `SubProto / Server` 补函数级中文注释
- [x] `APP-FUNC-1` 为 `Android / EmbeddedSDK / MetricsNode` 补函数级中文注释
- [x] `WIN-FUNC-1` 为 `Win` backend/frontend/scripts/MCP 补函数级中文注释
- [x] `REVIEW-FUNC-1` 汇总验证、review 与冲突复核
- [x] `ARCHIVE-FUNC-1` 归档到 `docs/change`

### 任务明细
- `CTRL-FUNC-1`
  - Goal: 创建 control 与 9 个 repo worktree，并把当前未提交状态同步过去
  - Files: worktree 管理，无业务源码 write set
  - Acceptance: worktree 全部存在，repo worktree 状态与主路径基线对齐
  - Tests: `git worktree list`、各 repo `git status --short`
  - Rollback: `git worktree remove` + 删除分支
- `CTRL-FUNC-2`
  - Goal: 在 control worktree 与 repo worktree 固化计划、边界和拆分
  - Files:
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-control\todo.md`
    - 各 repo worktree 根 `todo.md`
  - Acceptance: plan 可脱离当前聊天独立阅读
  - Tests: 人工复核文档完整性
  - Rollback: 回退对应 `todo.md`
- `CORE-FUNC-1`
  - Goal: `Proto / Core / SDK` 函数级中文注释补充
  - Files:
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-proto\**`
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-core\**`
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-sdk\**`
  - Acceptance: eligible source 的具名函数/方法完成一轮注释补充
  - Tests: `gofmt`, `git diff --check`
  - Rollback: 按 repo 撤销 comment-only diff
- `SERVER-FUNC-1`
  - Goal: `SubProto / Server` 函数级中文注释补充
  - Files:
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-subproto\auth|broker|exec|file|flow|forward|management|stream|topicbus|varstore\**`
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-server\cmd|hubruntime|modules|tests\**`
  - Acceptance: handler/runtime/provider/装配链路函数完成一轮注释补充
  - Tests: `gofmt`, `git diff --check`
  - Rollback: 按 repo 撤销 comment-only diff
- `APP-FUNC-1`
  - Goal: `Android / EmbeddedSDK / MetricsNode` 函数级中文注释补充
  - Files:
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-android\**`
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-embeddedsdk\**`
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-metricsnode\android|core|nodemobile|scripts|windows\**`
  - Acceptance: 宿主入口、bridge、runtime、SDK helper 函数完成一轮注释补充
  - Tests: `gofmt`, `git diff --check`, 必要的脚本语法检查
  - Rollback: 按 repo 撤销 comment-only diff
- `WIN-FUNC-1`
  - Goal: `Win` backend/frontend/scripts/MCP 函数级中文注释补充
  - Files:
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-win\app*.go`
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-win\internal\**`
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-win\frontend\src\**`
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-win\scripts\*.ps1`
  - Acceptance: 页面/store/service/Go backend 中的具名函数完成一轮注释补充
  - Tests: `gofmt`, `git diff --check`, `.ps1` AST parse
  - Rollback: 按 repo 撤销 comment-only diff
- `REVIEW-FUNC-1`
  - Goal: 汇总各子任务结果，做 review checklist 与验证
  - Files: 无新增功能文件，必要时只修 comment-only 问题
  - Acceptance: review checklist 全部通过
  - Tests: 统一 diff/格式/语法检查
  - Rollback: 按 Task ID 回退
- `ARCHIVE-FUNC-1`
  - Goal: 输出 `docs/change/2026-04-14_thesis-function-comments-cn.md` 并更新索引
  - Files:
    - `D:\project\MyFlowHub3\docs\change\2026-04-14_thesis-function-comments-cn.md`
    - 必要的 `docs/change/README.md`
    - 必要的 `docs/plan/README.md`
    - `D:\project\MyFlowHub3\plan.md`
  - Acceptance: 归档可回溯，子 agent 轨迹可审计
  - Tests: 文档交叉链接人工复核
  - Rollback: 回退对应 docs 变更

### 并行拆分与子 agent 预案
- Owner: 主 agent
  - Worktree: `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-control`
  - Write set: control `todo.md`、集成验证、review、archive
  - Key refs:
    - `D:\project\MyFlowHub3\guide.md`
    - `D:\project\MyFlowHub3\repos.md`
    - `D:\project\MyFlowHub3\docs\change\2026-04-13_thesis-code-comments-cn.md`
- Worker A
  - Task ID: `CORE-FUNC-1`
  - Worktree roots:
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-proto`
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-core`
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-sdk`
  - Write set: 仅上述 3 个 repo worktree，排除 generated/build/docs
- Worker B
  - Task ID: `SERVER-FUNC-1`
  - Worktree roots:
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-subproto`
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-server`
  - Write set: 仅上述 2 个 repo worktree，保留 SubProto 脏基线文件
- Worker C
  - Task ID: `APP-FUNC-1`
  - Worktree roots:
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-android`
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-embeddedsdk`
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-metricsnode`
  - Write set: 仅上述 3 个 repo worktree，保留 MetricsNode 脏基线与 `windows/frontend/wailsjs/**` 排除
- Worker D
  - Task ID: `WIN-FUNC-1`
  - Worktree root:
    - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-win`
  - Write set: 仅 Win worktree，排除 `frontend/wailsjs/**`、`*.exe`

### 依赖、风险与说明
- 依赖:
  - 必须以当前未提交基线继续工作
  - 必须在用户确认计划后才进入 3.2
- 风险:
  - 过度加注释会制造噪声
  - 子 agent 若不遵守 write set 会互相污染
  - Win / MetricsNode 的 generated 目录最容易误碰
- 说明:
  - 本轮优先追求“函数级理解增益”，不是把每个函数名再说一遍

阻塞：否
进入 3.2

## Stage 3.3 - Code Review

- 需求覆盖：通过
  - 9 个参与仓库都完成了一轮函数级中文注释补充。
  - Win 的剩余热点 `frontend/src/stores/flow.ts` 与 `frontend/src/windows/FlowEditorWindow.vue` 已补完。
- 架构合理性：通过
  - 变更保持在 comment-only 范围，没有扩展到行为修复、依赖升级或结构重构。
  - 注释重点落在 handler / runtime / bridge / store / page orchestration / schema 与协议转换链路。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 本轮没有引入新的运行时逻辑、I/O 或并发路径。
- 可读性与一致性：通过
  - 注释统一改为中文，并按各语言原生 comment style 落地。
  - 对一行 getter / 纯转发器维持克制，没有机械性全量噪声注释。
- 可扩展性与配置化：通过
  - 没有增加硬编码配置，也没有引入新的跨模块依赖。
- 稳定性与安全：通过
  - 主路径 repo 未被直接实现编辑。
  - `frontend/wailsjs/**`、`windows/frontend/wailsjs/**`、`docs/change/**`、`*.exe` 等排除边界保持不变。
- 测试覆盖情况：通过
  - 9 个 repo worktree 的 `git diff --check -- . ':(exclude)todo.md'` 全部通过。
  - 8 个 `.ps1` 的 PowerShell AST parse 全部通过。
  - Go 目标文件已按 task-scoped 范围执行 `gofmt -w`。
  - 各子任务都做了目标文件级 diff 复核；主线程补做了 Android / Core / SubProto / Win 的样本抽查，以及 `flow.ts` / `FlowEditorWindow.vue` 的 comment-only 复核。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 在 Stage 3.2 依 Task ID 派发了 `CORE-FUNC-1`、`SERVER-FUNC-1`、`APP-FUNC-1`、`WIN-FUNC-1` 四组 bounded write set。
  - `WIN-FUNC-1` 后续按文件粒度拆分，由主线程接手 `flow.ts`，新增子 agent 收口 `FlowEditorWindow.vue`，写集无重叠。
  - 所有子 agent 结果都已回收，并记录到本次归档的执行轨迹中。

结论：通过，进入 Stage 4。

## Stage 4 - Change Archive

- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Archive path:
  - `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-control\docs\change\2026-04-14_thesis-function-comments-cn.md`
- Requirements impact: none
- Specs impact: none
- Lessons impact: none
  - 原因：本轮没有产出新的稳定架构约束或可复用排障链路，既有 lessons 已足够覆盖 worktree / wailsjs 边界。
- Related requirements: none
- Related specs:
  - `D:\project\MyFlowHub3\repos.md`
  - `D:\project\MyFlowHub3\docs\specs\README.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\README.md`
- Related lessons:
  - `D:\project\MyFlowHub3\docs\lessons\frontend-worktree-wailsjs-missing.md`
  - `D:\project\MyFlowHub3\docs\lessons\wails-bindings-cross-project.md`
- 索引更新：
  - 已更新 `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-control\docs\change\README.md`

Stage 4 完成，等待用户确认是否结束 workflow。
