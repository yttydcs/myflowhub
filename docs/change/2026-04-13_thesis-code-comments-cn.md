# 2026-04-13 thesis-code-comments-cn

## 变更背景 / 目标
- 用户在上一轮“高覆盖补注释”基础上，继续要求两件事：
  - 注释尽量改成中文
  - 除自动生成代码外，尽量再补一些帮助理解的说明
- 本轮目标是在不改运行时行为的前提下，把 `repo/` 下 9 个仓库残留的英文模板解释性注释改成中文，并补少量更贴近仓库职责的中文说明。
- 继续遵守“以未提交为基线”，不吞掉用户在 `MetricsNode / SDK / SubProto / Win` 中已有的脏改动。

## 具体变更内容

### 跨仓执行方式
- 使用 `$m-autoflow` 沿用上一轮的 9 个 repo worktree：
  - `MyFlowHub-Android`
  - `MyFlowHub-Core`
  - `MyFlowHub-EmbeddedSDK`
  - `MyFlowHub-MetricsNode`
  - `MyFlowHub-Proto`
  - `MyFlowHub-SDK`
  - `MyFlowHub-Server`
  - `MyFlowHub-SubProto`
  - `MyFlowHub-Win`
- 保留并继续携带 4 个带未提交基线的仓库状态：
  - `MyFlowHub-MetricsNode`: `.gitignore`、`windows/frontend/wailsjs/**`、`windows/go.mod`
  - `MyFlowHub-SDK`: `.gitignore`
  - `MyFlowHub-SubProto`: `auth/actions_login.go`、`auth/display_name_test.go`、`docs/change/README.md` 与两份未跟踪 `docs/change/*.md`
  - `MyFlowHub-Win`: `go.mod`、`myflowhub-mcp.exe`
- 排除范围继续保持不变：
  - 自动生成代码
  - `docs/**`
  - `frontend/wailsjs/**`
  - `windows/frontend/wailsjs/**`
  - `frontend/src/generated/**`
  - `generated/**`
  - `dist/**`
  - `build/**`
  - `bin/**`
  - `obj/**`
  - `*.exe`

### 注释中文化与回补
- `MyFlowHub-Android`
  - 把 Android 宿主、`hubmobile` 桥接、工具入口和脚本头部说明改成中文
- `MyFlowHub-Core`
  - 把 bootstrap / config / listener / process / reader / server 等基础设施层说明改成中文
- `MyFlowHub-EmbeddedSDK`
  - 把 C SDK、MicroPython SDK、示例和工具链入口说明改成中文
- `MyFlowHub-MetricsNode`
  - 把 Windows/Android 宿主、核心 runtime、`nodemobile` 和脚本说明改成中文
- `MyFlowHub-Proto`
  - 把协议类型字典、generator 入口和内部渲染逻辑说明改成中文
- `MyFlowHub-SDK`
  - 把 `await/session/transport` 的统一语义注释改成中文
- `MyFlowHub-Server`
  - 把 `cmd`、`hubruntime`、`modules/defaultset` 和测试装配层说明改成中文
- `MyFlowHub-SubProto`
  - 把 `auth/broker/exec/file/flow/forward/management/stream/topicbus/varstore` 等模块说明改成中文
- `MyFlowHub-Win`
  - 把 Wails backend、frontend 页面/状态层、MCP 入口和 PowerShell 脚本说明改成中文
  - 对 Win 仓中较模板化的旧注释做了二次重写，避免只剩文件名复述

### 变更规模
- `MyFlowHub-Android`: `57 files changed, 57 insertions(+), 57 deletions(-)`
- `MyFlowHub-Core`: `67 files changed, 67 insertions(+), 67 deletions(-)`
- `MyFlowHub-EmbeddedSDK`: `83 files changed, 83 insertions(+), 83 deletions(-)`
- `MyFlowHub-MetricsNode`: `43 files changed, 43 insertions(+), 42 deletions(-)`
- `MyFlowHub-Proto`: `18 files changed, 18 insertions(+), 18 deletions(-)`
- `MyFlowHub-SDK`: `9 files changed, 9 insertions(+), 8 deletions(-)`
- `MyFlowHub-Server`: `59 files changed, 59 insertions(+), 59 deletions(-)`
- `MyFlowHub-SubProto`: `121 files changed, 155 insertions(+), 120 deletions(-)`
- `MyFlowHub-Win`: `200 files changed, 200 insertions(+), 200 deletions(-)`

## Requirements impact
- none

## Specs impact
- none

## Lessons impact
- none
- 原因：
  - 本轮主要是注释文本中文化和批量校验收口，没有新增稳定的架构约束、运行时陷阱或可复用排障路线需要沉淀到 `docs/lessons`

## Related requirements
- none

## Related specs
- `D:\project\MyFlowHub3\repos.md`
- `D:\project\MyFlowHub3\docs\specs\README.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\README.md`

## Related lessons
- `D:\project\MyFlowHub3\docs\lessons\frontend-worktree-wailsjs-missing.md`
- `D:\project\MyFlowHub3\docs\lessons\wails-bindings-cross-project.md`

## 对应 plan.md 任务映射
- `CTRL-1`
  - 固化跨仓边界、排除范围和归档目标
- `AND-1`
  - Android 注释中文化
- `CORE-1`
  - Core 注释中文化
- `EMB-1`
  - EmbeddedSDK 注释中文化
- `MET-1`
  - MetricsNode 注释中文化
- `PROTO-1`
  - Proto 注释中文化
- `SDK-1`
  - SDK 注释中文化
- `SERVER-1`
  - Server 注释中文化
- `SUB-1`
  - SubProto 注释中文化并保留脏基线
- `WIN-1`
  - Win backend/frontend/MCP/脚本注释中文化
- `REVIEW-1`
  - 完成 review checklist 与验证记录
- `ARCHIVE-1`
  - 完成 `docs/change` 归档与索引更新

## 经验 / 教训摘要
- “把英文注释翻成中文”如果只做机械替换，价值很低；仍然需要按仓库角色重写成职责导向的中文说明。
- 多语言仓库批量改注释时，必须按语言复核 comment style，尤其是 `.ps1` 这类对注释前缀敏感的脚本。
- 用户明确要求“以未提交为基线”时，先同步脏基线再编辑，比事后甄别哪些文件属于用户改动更可靠。

## 可复用排查线索
- 症状：
  - 仓库里还有 `// Context:`、`# Context:` 之类模板注释
  - Win 仓前端或脚本注释虽然变中文了，但仍然只是在重复文件名
  - 注释任务误碰 `wailsjs`、`*.exe` 或已有未提交文件
- 触发条件：
  - 直接用统一模板批量替换，不看语言和模块角色
  - 忽略“以未提交为基线”的脏状态
  - 没有在收尾阶段做语法级或 diff-level 校验
- 关键词：
  - `Context:`
  - `COMMENT_CONTEXT_CLEAR`
  - `PARSE_OK`
  - `git diff --check`
  - `以未提交为基线`
- 快速检查：
  1. 对 9 个 repo worktree 运行 `git diff --check -- . ':(exclude)todo.md'`
  2. 用注释前缀模式扫描残留 `Context:` 模板注释
  3. 对改动过的 `.ps1` 执行 PowerShell AST parse
  4. 复核是否误改 generated / binary / build outputs 和用户脏基线

## 关键设计决策与权衡
- 决策：继续沿用上一轮的高覆盖范围，不重做一次新的“从零补注释”
  - 原因：上一轮已建立覆盖面，本轮更适合做语言统一和质量回补
- 决策：对 `.sh` 不把“未验证”写成“已验证”
  - 原因：本机缺少 `bash/sh` 可执行程序；本轮只把它们记为“手工 diff 确认只改注释”，不伪造 parser 结果
- 决策：继续严格排除 generated / binary / build outputs
  - 原因：这些文件不属于帮助毕设理解源码的稳定入口，误改风险高
- 决策：保留 4 个脏仓的未提交基线
  - 原因：本轮任务是注释理解增强，不应该覆盖用户已有功能改动或本地产物

## 测试与验证方式 / 结果
- `gofmt -w`
  - 对 9 个 repo worktree 改动过的 Go 文件执行
  - 结果：通过
- `git diff --check -- . ':(exclude)todo.md'`
  - 对 9 个 repo worktree 执行
  - 结果：全部 `DIFF_CHECK_OK`
- PowerShell AST parse
  - 对 8 个改动过的 `.ps1` 执行
  - 结果：全部 `PARSE_OK`
- 模板注释残留扫描
  - 对 9 个 repo worktree 以注释前缀模式扫描 `Context:`
  - 结果：全部 `COMMENT_CONTEXT_CLEAR`
- Shell 脚本复核
  - 本机缺少 `bash/sh`，未执行 `bash -n`
  - 已检查 Android 与 MetricsNode 的 `scripts/build_aar.sh` diff
  - 结果：两者都只修改了首行注释文本，未改命令结构
- 样本抽查
  - 已抽查 Android / Core / EmbeddedSDK / Server / SubProto / Win 的代表文件
  - 结果：注释均为中文，且保留各语言原生 comment style

## 潜在影响
- 这是高覆盖 comment-only 变更，后续若这些文件同时有功能改造，merge 冲突概率会升高
- Win 和 SubProto 改动面最大，后续如果模块职责调整，注释需要同步维护，避免再变成模板文案

## 回滚方案
1. 按 repo worktree 粒度回退本轮注释 diff
2. 对带脏基线的仓库，只回退本轮新增注释改动，不回退同步进来的用户文件
3. 若只需局部回退，按 Task ID 对应仓库的 diff 独立撤销

## 子Agent执行轨迹
- none
- 本轮未派发子Agent；所有执行、复核和归档均由主 Agent 串行完成
