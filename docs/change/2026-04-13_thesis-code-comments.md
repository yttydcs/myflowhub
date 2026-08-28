# 2026-04-13 thesis-code-comments

## 变更背景 / 目标
- 用户为毕设准备，需要在 `repo/` 下的全部仓库尽量补充帮助理解代码的解释性注释。
- 用户进一步明确了两个边界：
  - 尽量覆盖所有源码
  - 以未提交为基线
- 本轮目标是在不改变行为的前提下，跨 `Android / Core / EmbeddedSDK / MetricsNode / Proto / SDK / Server / SubProto / Win` 九个仓库补齐高覆盖说明性注释，并保留用户已有未提交基线。

## 具体变更内容

### 跨仓执行方式
- 使用 `$m-autoflow` 在 `D:\project\MyFlowHub3\worktrees\` 下为 control repo 和九个参与仓库创建独立 `chore/thesis-code-comments` worktree。
- 将 4 个带未提交基线的仓库同步到执行 worktree，避免注释任务覆盖用户原有改动：
  - `MyFlowHub-MetricsNode`: `.gitignore`
  - `MyFlowHub-SDK`: `.gitignore`
  - `MyFlowHub-SubProto`: `auth/actions_login.go`, `auth/display_name_test.go`, `docs/change/README.md`, `docs/change/2026-04-03_flow-transform-node-runtime.md`, `docs/change/2026-04-05_flow-drop-legacy-compat.md`
  - `MyFlowHub-Win`: `myflowhub-mcp.exe`
- 明确排除 generated / binary / build outputs，不把 `docs/**`、`node_modules/**`、`dist/**`、`build/**`、`bin/**`、`obj/**`、`frontend/wailsjs/**`、`frontend/src/generated/**`、`*.exe` 等非维护中源码计入覆盖。

### 各仓注释补充
- `MyFlowHub-Android`
  - 为 Android App、Foreground Service、`hubmobile` 桥接、工具入口和脚本补充职责注释。
- `MyFlowHub-Core`
  - 为 bootstrap、config、connmgr、header、listener、process、reader、server 等基础设施层补充职责说明。
- `MyFlowHub-EmbeddedSDK`
  - 为 C SDK、MicroPython SDK、examples、fixture 工具与测试入口补充分层说明。
- `MyFlowHub-MetricsNode`
  - 为 Windows / Android 宿主、前端、采集上报相关入口与脚本补充职责说明。
- `MyFlowHub-Proto`
  - 为 protocol 类型字典、generator 入口和 flow contract / protocol map 相关工具补充职责说明。
- `MyFlowHub-SDK`
  - 为 `await`、`session`、`transport` 与顶层 client 统一语义补充职责说明。
- `MyFlowHub-Server`
  - 为 `cmd` 入口、`hubruntime`、`modules` 和相关测试补充装配层注释。
- `MyFlowHub-SubProto`
  - 为 `auth/broker/exec/file/flow/forward/management/topicbus/varstore` 等模块补充说明，同时保留用户已存在的未提交基线。
- `MyFlowHub-Win`
  - 为 Wails `app_*.go`、`internal/services/*`、`internal/mcp*`、frontend `pages/stores/windows/components`、scripts 等层补充职责说明。
  - 额外回补 Win 仓早期批量注释的低质量模板文案。
  - 修正 `.ps1` 顶部误插入的非法 `// Context:`，改为 PowerShell 可解析的 `# Context:`。

### 变更规模
- `MyFlowHub-Android`: `57 files changed, 92 insertions(+), 18 deletions(-)`
- `MyFlowHub-Core`: `67 files changed, 147 insertions(+), 18 deletions(-)`
- `MyFlowHub-EmbeddedSDK`: `83 files changed, 165 insertions(+)`
- `MyFlowHub-MetricsNode`: `43 files changed, 77 insertions(+), 9 deletions(-)`
- `MyFlowHub-Proto`: `18 files changed, 36 insertions(+), 6 deletions(-)`
- `MyFlowHub-SDK`: `9 files changed, 17 insertions(+)`
- `MyFlowHub-Server`: `59 files changed, 118 insertions(+), 2 deletions(-)`
- `MyFlowHub-SubProto`: `121 files changed, 273 insertions(+), 2 deletions(-)`
- `MyFlowHub-Win`: `200 files changed, 362 insertions(+), 12 deletions(-)`

## Requirements impact
- none

## Specs impact
- none

## Lessons impact
- none
- 原因：
  - 本轮暴露的问题主要是 workflow 内部批量注释脚本的实现细节，不属于需要沉淀成长期检索入口的稳定 repo / runtime 经验。

## Related requirements
- none

## Related specs
- `D:\project\MyFlowHub3\repos.md`
- `D:\project\MyFlowHub3\docs\specs\README.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\README.md`

## Related lessons
- `D:\project\MyFlowHub3\docs\lessons\frontend-worktree-wailsjs-missing.md`
- `D:\project\MyFlowHub3\docs\lessons\wails-bindings-cross-project.md`

## 对应 `plan.md` 任务映射
- `CTRL-1`
  - 固化跨仓范围、排除规则、基线同步和归档路径。
- `AND-1`
  - 完成 `MyFlowHub-Android` 注释补充。
- `CORE-1`
  - 完成 `MyFlowHub-Core` 注释补充。
- `EMB-1`
  - 完成 `MyFlowHub-EmbeddedSDK` 注释补充。
- `MET-1`
  - 完成 `MyFlowHub-MetricsNode` 注释补充。
- `PROTO-1`
  - 完成 `MyFlowHub-Proto` 注释补充。
- `SDK-1`
  - 完成 `MyFlowHub-SDK` 注释补充。
- `SERVER-1`
  - 完成 `MyFlowHub-Server` 注释补充。
- `SUB-1`
  - 完成 `MyFlowHub-SubProto` 注释补充，并保留用户未提交基线。
- `WIN-1`
  - 完成 `MyFlowHub-Win` 注释补充与脚本注释样式修正。
- `ARCHIVE-1`
  - 完成 review 与归档。

## 经验 / 教训摘要
- comment-only 的高覆盖任务仍需要语言感知的注释落点和 comment style；否则很容易在 `.ps1` 这类脚本里产生真正的语法回归。
- 跨仓大面积注释任务不适合只看 `git diff --stat`；需要配合样本 diff 和 `git diff --check` 识别模板化文案、额外空行和无意义格式漂移。
- 用户明确要求“以未提交为基线”时，应先同步脏基线，再开始注释补充，否则容易把用户自己的功能改动误判成任务范围内内容。

## 可复用排查线索
- 症状：
  - `.ps1` 顶部出现 `// Context:` 导致脚本可能失效
  - 顶层注释内容只重复文件名或目录名，无法帮助理解职责
  - 批量注释后 `git diff --check` 报空行或尾部格式问题
- 触发条件：
  - 对多语言仓库使用单一模板注释生成规则
  - 未在 comment-only 任务中加入语法级 / diff-level 验证
  - 忽略用户原有未提交基线
- 关键词：
  - `Context:`
  - `git diff --check`
  - `PARSE_OK`
  - `frontend/wailsjs`
  - `myflowhub-mcp.ps1`
  - `以未提交为基线`
- 快速检查：
  1. 在各 repo worktree 执行 `git diff --check -- . ':(exclude)todo.md'`
  2. 对脚本仓使用对应语言 parser 做快速语法检查
  3. 复核是否误改 generated / binary / build outputs
  4. 复核是否保留了用户已有未提交文件

## 关键设计决策与权衡
- 决策：采用“跨仓广覆盖 + 顶层职责注释”为主，而不是把少量文件写成深度行内注释。
  - 原因：用户明确要求尽量覆盖所有源码，优先目标是建立快速理解入口，而不是重写局部热点文档。
- 决策：严格排除 generated / binary / build outputs。
  - 原因：这些文件不适合作为理解源码的稳定入口，也会显著放大误改风险。
- 决策：在 Win 仓额外做第二轮文案回补，而不是接受初始模板化结果。
  - 原因：Win 仓前端、Wails backend 和 MCP 脚本三层并存，若注释过于模板化，实际帮助有限；脚本 comment style 错误还会引入语法回归。
- 决策：保留用户未提交基线，不对脏仓做清理。
  - 原因：本轮任务是注释补充，不应吞掉用户未提交功能改动。

## 测试与验证方式 / 结果
- `gofmt`
  - 对第一轮批量改动涉及的 Go 文件已统一执行。
  - 结果：通过。
- `git diff --check -- . ':(exclude)todo.md'`
  - 已在 9 个 repo worktree 全部执行。
  - 结果：通过。
- `PowerShell AST parse`
  - `D:\project\MyFlowHub3\worktrees\chore-thesis-code-comments-win\scripts\start-myflowhub-mcp.ps1`
  - `D:\project\MyFlowHub3\worktrees\chore-thesis-code-comments-win\scripts\install-codex-myflowhub-mcp.ps1`
  - `D:\project\MyFlowHub3\worktrees\chore-thesis-code-comments-win\scripts\test-myflowhub-mcp-smoke.ps1`
  - 结果：全部 `PARSE_OK`。
- 样本 diff 复核
  - 重点抽查 `Win` 的 `Home.vue`、`stream.ts`、`FlowCanvas.vue`、`internal/services/file/events.go`、`internal/services/localhub/service.go` 和三份 MCP 脚本。
  - 结果：模板化或非法注释样式问题已修正。
- 说明：
  - 本轮为 comment-only 任务，未执行全仓 build / runtime smoke；验证重点放在语法安全、格式安全和基线保留。

## 潜在影响与回滚方案
- 潜在影响：
  - 大面积 comment-only 变更会提高后续 merge 冲突概率。
  - Win 仓存在较多前端和脚本文字说明，若后续模块职责调整，注释需要同步维护。
- 回滚方案：
  1. 在各 repo worktree 直接回退本轮注释 diff
  2. 保留用户同步进来的未提交基线文件，不把它们一起回退
  3. 若仅需局部回滚，可按 Task ID 对应 repo worktree 独立撤回

## 子Agent执行轨迹
- none
- 本轮未派发子Agent；所有变更均由主 Agent 在各自 worktree 内完成并复核。
