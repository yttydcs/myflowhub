# 2026-04-14 thesis-function-comments-cn

## 变更背景 / 目标
- 用户在上一轮“文件级职责注释 + 中文化”基础上，继续要求把 `repo/` 下 9 个仓库的非自动生成源码补到“函数级中文注释”。
- 目标不是翻译函数名，而是让阅读者更快理解函数职责、输入约束、状态变化、边界处理和关键 why。
- 全程继续遵守“以未提交为基线”，不吞掉主路径当前已有脏改动。

## 具体变更内容

### 跨仓执行方式
- 使用 `$m-autoflow` 在 control worktree 下继续执行，主路径 `repo/` 只保留基线，不直接落实现。
- 使用 `$m-docs` 校验本轮归档路由：
  - 稳定 truth：`requirements/specs`
  - workflow 结果：`change`
  - 可复用排障经验：`lessons`
- 继续沿用本轮已建立的 9 个 repo worktree：
  - `MyFlowHub-Android`
  - `MyFlowHub-Core`
  - `MyFlowHub-EmbeddedSDK`
  - `MyFlowHub-MetricsNode`
  - `MyFlowHub-Proto`
  - `MyFlowHub-SDK`
  - `MyFlowHub-Server`
  - `MyFlowHub-SubProto`
  - `MyFlowHub-Win`

### 注释补充范围
- `MyFlowHub-Android`
  - 补了 Android 宿主、`hubmobile` bridge、management / varstore 等函数级中文注释。
- `MyFlowHub-Core`
  - 补了 dispatcher / pre-route / queue strategy / reader / server / subproto kit / write util 等核心管线函数注释。
- `MyFlowHub-EmbeddedSDK`
  - 补了 C runtime / transport、MicroPython client/runtime/async runtime 的函数级中文注释。
- `MyFlowHub-MetricsNode`
  - 补了 runtime control queue / management / `nodemobile` 桥接函数注释。
- `MyFlowHub-Proto`
  - 保留上一阶段已完成的 generator / internal render 相关函数注释增量。
- `MyFlowHub-SDK`
  - 保留上一阶段已完成的 `await/session/transport` 函数注释增量。
- `MyFlowHub-Server`
  - 补了 `modules/defaultset/*_enabled.go` 等默认装配链路函数注释。
- `MyFlowHub-SubProto`
  - 补了 `file/stream/topicbus` 等剩余 handler / provider / session 生命周期函数注释。
- `MyFlowHub-Win`
  - 补了 Home / AccessPolicy / AppShell / authority / stream / varpool / topicbus / showcaseChart / flow store / FlowEditorWindow 等页面、store、独立窗口的函数级中文注释。

### 排除与边界
- 保持以下排除不变：
  - `frontend/wailsjs/**`
  - `windows/frontend/wailsjs/**`
  - `docs/change/**`
  - `generated/**`
  - `build/**`
  - `dist/**`
  - `*.exe`
- 对简单 getter / 纯转发器维持克制，避免把仓库变成注释噪声源。
- 所有改动限定为注释和最小必要格式化，无行为变更。

## Related Plan
- `D:\project\MyFlowHub3\worktrees\chore-thesis-function-comments-cn-control\todo.md`

## Related Requirements
- none

## Related Specs
- `D:\project\MyFlowHub3\repos.md`
- `D:\project\MyFlowHub3\docs\specs\README.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\README.md`

## Lessons Impact
- none
- 原因：
  - 本轮主要是注释补充与验证收口，没有新增稳定的架构约束、运行时陷阱或可复用排障路线需要沉淀到 `docs/lessons`

## Related Lessons
- `D:\project\MyFlowHub3\docs\lessons\frontend-worktree-wailsjs-missing.md`
- `D:\project\MyFlowHub3\docs\lessons\wails-bindings-cross-project.md`

## Searchable Lessons Summary
- 症状：
  - 仓库里仍有“函数名能看懂，但职责/边界难以快速判断”的源码段落
  - Win Flow 编辑器、SubProto handler、Core dispatcher 这类长链路文件阅读成本高
- 触发条件：
  - 毕设写作前需要快速回答函数职责
  - 多仓联读时缺少函数级 why 注释
- 关键词：
  - `函数级中文注释`
  - `comment-only`
  - `git diff --check`
  - `PowerShell AST parse`
  - `以未提交为基线`
- 快速检查：
  1. 对 repo worktree 跑 `git diff --check -- . ':(exclude)todo.md'`
  2. 对改过的 `.ps1` 跑 AST parse
  3. 抽查代表文件，确认新增内容是函数级中文注释，不是行为改动

## Requirements Impact
- none

## Specs Impact
- none

## 对应 plan 任务映射
- `CTRL-FUNC-1`
  - 建立 control/repo worktree，并同步当前未提交基线
- `CTRL-FUNC-2`
  - 固化 Stage 1 / Stage 2 / Stage 3.1 文档
- `CORE-FUNC-1`
  - `Proto / Core / SDK` 函数级中文注释补充
- `SERVER-FUNC-1`
  - `SubProto / Server` 函数级中文注释补充
- `APP-FUNC-1`
  - `Android / EmbeddedSDK / MetricsNode` 函数级中文注释补充
- `WIN-FUNC-1`
  - `Win` backend/frontend/scripts/MCP 函数级中文注释补充
- `REVIEW-FUNC-1`
  - 汇总验证、review 与冲突复核
- `ARCHIVE-FUNC-1`
  - 归档到 `docs/change`

## 经验 / 教训摘要
- 函数级注释最有价值的部分是职责、边界和 why，不是把函数名翻译成中文。
- 在 dirty-baseline workflow 里，验证应优先依赖 task-scoped diff 复核和 worktree 内语法/格式检查，而不是把主路径当前状态当作唯一对照源。
- 大文件最适合按“状态快照 / schema 对齐 / runtime 编排 / handler 生命周期”这类语义块补注释，不应机械给每个短 helper 都写一行。

## 关键设计决策与权衡
- 决策：继续按 repo worktree 边界并行，而不是回到串行逐仓
  - 原因：9 仓写集天然隔离，子 agent 并行能明显缩短收口时间
- 决策：对 Win 最后的两个大文件拆成主线程 + 子 agent 分担
  - 原因：`flow.ts` 与 `FlowEditorWindow.vue` 写集不重叠，拆分后更容易保持 comment density 一致
- 决策：保持 `Lessons impact: none`
  - 原因：本轮没有形成新的可复用排障路径，现有 lessons 已能覆盖 worktree / wailsjs 边界

## 测试与验证方式 / 结果
- `git diff --check -- . ':(exclude)todo.md'`
  - 对 9 个 repo worktree 执行
  - 结果：通过
- `gofmt -w`
  - 对本轮 task-scoped 触达的 Go 文件执行
  - 结果：通过
- PowerShell AST parse
  - 对 8 个改动过的 `.ps1` 执行
  - 结果：通过
- 目标文件级 diff 复核
  - 各子任务都对自己负责的目标文件做了 diff-level 复核
  - 主线程补做了 Android / Core / SubProto / Win 代表文件抽查，以及 `flow.ts` / `FlowEditorWindow.vue` 的 comment-only 复核
  - 结果：通过
- 未执行项
  - 未做全量 build / runtime smoke；本轮目标是 comment-only 注释补充，不做高成本构建验证

## 潜在影响与回滚方案
- 潜在影响：
  - 这是高覆盖 comment-only diff，后续若同一批文件再做功能改造，merge 冲突概率会升高
  - 注释如果后续不跟代码一起维护，仍可能重新老化成模板文案
- 回滚方案：
  1. 按 repo worktree 粒度回退本轮注释 diff
  2. 对带脏基线的仓库，只撤销本轮新增注释，不回退用户原有未提交内容
  3. 若只需局部回退，按 Task ID 对应文件集单独撤销

## 子Agent执行轨迹
- `CORE-FUNC-1`
  - 子 agent：`019d87ee-8a21-78b0-b2ef-2b7e376630a1`
  - Write set：`MyFlowHub-Core` 剩余 dispatcher / reader / server / subproto kit / write util 热点文件
  - 结果：完成；已执行 `gofmt -w` 与 `git diff --check`
- `SERVER-FUNC-1`
  - 子 agent：`019d87ee-a1de-7432-88fb-ecbe9902148f`
  - Write set：`MyFlowHub-SubProto` 的 `file/stream/topicbus` 与 `MyFlowHub-Server` 的 `modules/defaultset/*_enabled.go`
  - 结果：完成；已执行 `gofmt -w` 与双 worktree `git diff --check`
- `APP-FUNC-1`
  - 子 agent：`019d87ee-b935-74c0-831a-6d7bb8775f9f`
  - Write set：Android `management/varstore`、EmbeddedSDK `runtime/transport/client`、MetricsNode `control_queue/management/nodemobile`
  - 结果：完成；已执行 task-scoped `gofmt -w` 与 `git diff --check`
- `WIN-FUNC-1`
  - 子 agent：`019d87ee-d325-7ea0-bf3f-2ed8202bf39c`
  - Write set：Win 前端 `Home / AccessPolicy / AppShell / authority / stream / varpool / topicbus / showcaseChart`
  - 结果：完成安全点；剩余 `flow.ts` / `FlowEditorWindow.vue`
- `WIN-FUNC-1` 收口
  - 主线程：补完 `frontend/src/stores/flow.ts`
  - 子 agent：`019d87fd-47a5-7252-9e1d-68d510cfb737`
  - Write set：`frontend/src/windows/FlowEditorWindow.vue`
  - 结果：完成；Win 尾文件全部收口
