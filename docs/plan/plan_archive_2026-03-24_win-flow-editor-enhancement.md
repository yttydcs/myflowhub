# Plan - MyFlowHub-Win Flow Editor Enhancement Assessment

## Workflow Information

- Repo: `MyFlowHub-Win`
- Branch: `feat/win-dag-editor-enhancement-review`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review`
- Current Stage: `4 归档变更（FLOW-ENH-5 已归档，等待是否结束 workflow）`

## Stage Records

### Initialization

- `guide.md`: 已阅读，遵守 commit 中文说明、worktree 必须位于 `D:\project\MyFlowHub3\worktrees\`、优先使用 chrome-devtools 做界面验证的约束。
- base/worktree confirmation:
  - 控制面仓库：`D:\project\MyFlowHub3`，仅用于 workflow 编排、worktree 管理与最终归档。
  - 参与仓库：
    - `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
    - 参考协议文档：`D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`、`D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`
  - 活跃执行 worktree：`D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review`
  - 当前未进入实现阶段，不在主 repo 路径做业务改动。
- validation baseline:
  - `wails version`：`v2.11.0`
  - `$env:GOWORK='off'; wails generate module`：通过
  - `frontend/ npm ci`：通过
  - `frontend/ npm run build`：通过

### Stage 1 - Requirements Analysis

#### Goal

- 评估 `repo\MyFlowHub-Win` 当前 Flow 编辑界面是否具备继续增强的基础。
- 输出可执行的后续增强方向，而不是直接进入新功能实现。
- 明确当前实现的稳定骨架、主要风险和进入下一轮开发前的前置条件。

#### Scope

##### Must

- 阅读现有 requirements / specs / change 文档，确认稳定真相与近期演进轨迹。
- 审查 Flow 编辑器关键实现：
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/stores/flow_visual_form.ts`
  - `frontend/src/stores/flow_schema_resolver.ts`
  - `frontend/src/components/flow/*`
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/stores/flowProjects.ts`
- 判断当前是否具备继续增强的架构基础。
- 形成 handoff-ready 的 `plan.md`，为下一轮实现提供任务拆分、验收、测试与回滚入口。

##### Optional

- 结合现有前端界面规范，指出明显的可访问性 / 交互改进点。
- 验证本地前端构建基线，确认阻力来自环境还是编辑器实现本身。

##### Out of Scope

- 本轮不直接修改 Flow 编辑器行为。
- 本轮不改动 Flow / Exec 协议。
- 本轮不新增 docs/change 归档，因为还未进入实现与归档阶段。

#### Use Cases

- 你想判断当前 Flow 编辑器是否值得继续加功能，而不是推倒重写。
- 你想知道下一轮优先应该补“结构”、补“体验”、补“可靠性”还是补“验证体系”。
- 你想在不依赖当前聊天上下文的前提下，把下一步工作交给后续 workflow 继续执行。

#### Functional Requirements

1. 必须明确当前编辑器的稳定骨架和已有扩展点。
2. 必须识别继续增强的主要阻力，不得只给泛泛建议。
3. 必须给出可执行的后续任务拆分，并映射到明确文件范围。
4. 必须说明下一轮实现前是否需要补 requirements / specs。
5. 必须给出当前验证链路是否可用的结论。

#### Non-functional Requirements

- 结论必须以当前 repo 中文档和代码为主，不凭印象猜测。
- 计划必须可交接，不依赖“再回忆一下本次聊天”的隐含上下文。
- 不把 `docs/change` 当成长期真相；稳定行为仍以 `requirements/specs` 为准。

#### Inputs / Outputs

- Inputs:
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
  - `docs/change/2026-03-21_flow-editor-method-selector-dialog.md`
  - `docs/change/2026-03-22_win-flow-data-dag-editor.md`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/stores/flow_visual_form.ts`
  - `frontend/src/stores/flow_schema_resolver.ts`
  - `frontend/src/stores/flow_method_schemas.ts`
  - `frontend/src/components/flow/FlowCanvas.vue`
  - `frontend/src/components/flow/FlowNode.vue`
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/stores/flowProjects.ts`
- Outputs:
  - 当前 Flow 编辑器的评估结论
  - 下一轮增强的任务级计划
  - 阶段阻塞条件与待确认事项

#### Edge Cases

- 新 worktree 初始缺少 `frontend/wailsjs/**`，会导致前端构建误判为“代码坏了”。
- 当前稳定 requirements/specs 只覆盖 `call` 节点 visual form，不覆盖后续可能新增的 autosave、无障碍或编辑器壳层约束。
- 方法 schema 依赖 capability 查询结果，某些体验增强涉及“显式查询”与“本地缓存”边界。

#### Acceptance Criteria

1. 能明确回答“可以继续增强”，或给出具体阻塞理由。
2. 能清晰指出下一轮最值得做的 3-5 个方向。
3. `plan.md` 包含任务 ID、文件范围、验收、测试点和回滚点。
4. 当前验证基线是否可恢复有明确结论。

#### Risks

- 如果继续把新功能直接堆到单体文件中，增强速度会越来越快地被维护成本吞掉。
- 如果先做功能不做验证与结构收敛，后续回归风险会放大。
- 如果不先确认本轮优先级，就很容易一次把“结构重构、可靠性、UX、无障碍、测试栈”混在一起，导致计划失焦。

#### Issue List

- 无

### Stage 2 - Architecture Design

#### Overall Solution

- 结论：当前 Flow 编辑器具备继续增强的基础，不需要推倒重写。
- 继续增强应基于现有三层骨架，而不是另起一套实现：
  - 图与协议语义集中在 `frontend/src/stores/flow.ts`
  - visual form 兼容性与字段映射集中在 `frontend/src/stores/flow_visual_form.ts`
  - 方法 schema 解析集中在 `frontend/src/stores/flow_schema_resolver.ts` / `flow_method_schemas.ts`
- 下一轮最合理的路线不是“直接再加一批新控件”，而是：
  1. 先收敛 editor 壳层与组件边界
  2. 再补草稿可靠性 / 退出保护
  3. 再扩展 visual form 体验
  4. 同步补无障碍和测试

#### Alternatives Considered

- 继续在 `FlowEditorWindow.vue` / `flow.ts` 上直接叠加新功能：
  - 放弃作为主路线。
  - 原因：当前已形成单体热点文件，再叠加会显著放大回归与审查成本。

- 直接重写整套 Flow 编辑器：
  - 放弃。
  - 原因：当前 visual form、schema resolver、拓扑校验已经形成可复用骨架，重写性价比低。

- 只做样式和文案层面的轻量 polish：
  - 不建议作为下一轮唯一目标。
  - 原因：能改善观感，但不能解决结构、可靠性和验证短板。

#### Module Responsibilities

- `frontend/src/windows/FlowEditorWindow.vue`
  - 当前承担：窗口生命周期、快捷键、三类弹层、节点详情、能力选择、字段 binding 编辑、模板渲染。
  - 下一轮应收敛为：页面壳层 + 状态装配 + 顶层事件协调。

- `frontend/src/stores/flow.ts`
  - 当前承担：draft 模型、历史、拓扑、祖先计算、binding 校验、spec 构建、能力查询、schema defaults。
  - 下一轮应继续保留“协议/图语义中心”角色，但可按职责拆 helper 模块。

- `frontend/src/stores/flow_visual_form.ts`
  - 已经是较好的扩展点。
  - 适合作为字段级 literal/binding 行为和兼容性规则的长期承载层。

- `frontend/src/stores/flow_schema_resolver.ts`
  - 已经是较好的扩展点。
  - 适合继续承载 capability `input_schema` 子集解析和本地 override 合并。

- `frontend/src/components/flow/FlowCanvas.vue`
  - 继续负责画布渲染与交互转发，不建议把 inspector 逻辑再拉回画布层。

- `frontend/src/pages/Flow.vue` / `frontend/src/stores/flowProjects.ts`
  - 继续负责项目入口、编辑器窗口打开、项目图保存与部署链路。
  - 若新增 autosave / dirty-state，需与这里的项目存储模型一起设计。

#### Data / Call Flow

1. `Flow.vue` 从本地 project 列表打开 editor window。
2. `FlowEditorWindow.vue` 加载 project graph，并注入 `flowStore`。
3. `flowStore` 规范化节点 / 边 / spec draft，并维护历史。
4. `call` 节点通过 capability route + method schema resolver 获取 visual form。
5. inspector 对 literal / binding 的编辑，经 `flow_visual_form.ts` 和 `flow.ts` 写回 `args_template + inputs`。
6. 保存时导出 graph draft，写回 `flowProjects` 持久化。
7. 未来若补 dirty-state / autosave，应围绕步骤 5-6 增强，而不是绕开现有 graph draft。

#### Interface Drafts

- 推荐下一轮组件边界草案：
  - `frontend/src/components/flow/editor/FlowEditorToolbar.vue`
  - `frontend/src/components/flow/editor/FlowInspectorShell.vue`
  - `frontend/src/components/flow/editor/CallMethodPickerDialog.vue`
  - `frontend/src/components/flow/editor/FieldBindingDialog.vue`
  - `frontend/src/components/flow/editor/CallVisualFormSection.vue`
  - `frontend/src/components/flow/editor/ComposeBindingsSection.vue`

- 推荐新增状态能力草案：
  - `isDirty`
  - `lastSavedAt`
  - `beforeUnloadGuard`
  - `draftRecoverySnapshot`

#### Error Handling and Safety

- 当前已有的安全边界应保留：
  - 非法 JSON Pointer 立即拒绝
  - 非祖先引用立即拒绝
  - form/json 切换前做完整 spec 校验
- 下一轮应新增的安全边界：
  - 未保存草稿关闭窗口前提示
  - fresh worktree 的 `wails generate module` 前置校验
  - dialog / form 焦点和键盘行为的可访问性约束

#### Performance and Testing Strategy

- 当前 `frontend/ npm run build` 在补齐 `wailsjs` 后可以通过，但产物存在单 chunk 过大告警：
  - `dist/assets/index-*.js` 约 `945.44 kB`
- 下一轮应优先避免：
  - 再把大段 UI 分支继续塞进 `FlowEditorWindow.vue`
  - 在 capability picker 中无边界地扩大一次性渲染列表
- 测试策略建议：
  - 保留 `wails generate module` + `npm ci` + `npm run build` 作为基线验证
  - 为 `flow_visual_form.ts`、`flow_schema_resolver.ts`、`flow.ts` 的纯逻辑部分补前端自动化测试
  - 如要做 UI 交互回归，再补 editor 级组件测试或浏览器冒烟

#### Extensibility Design Points

- 正向结论：
  - `flow_visual_form.ts` 和 `flow_schema_resolver.ts` 已经把“继续增强 visual form”最危险的一部分抽出来了。
  - 这意味着后续增强不必从 `FlowEditorWindow.vue` 里硬编码每个方法的页面。

- 需要优先处理的扩展性问题：
  - `FlowEditorWindow.vue` 当前承担过多职责
  - `flow.ts` 当前聚合了太多领域逻辑与窗口协作逻辑
  - 还缺少稳定的 editor-level 回归测试

#### Issue List

- 无

### Stage 3.1 - Planning

#### Project Goal and Current State

- 当前可继续利用的基础：
  - `call` / `compose` DAG 编辑已成形
  - visual form requirements/specs 已存在
  - `flow_visual_form.ts` / `flow_schema_resolver.ts` 已形成独立扩展点
  - `FlowCanvas.vue` / `FlowNode.vue` 相对清晰，画布层不是主要阻塞
  - 当前 worktree 中通过 `wails generate module` 后，前端构建链路可恢复

- 当前主要约束：
  - `frontend/src/windows/FlowEditorWindow.vue` 约 `1655` 行
  - `frontend/src/stores/flow.ts` 约 `1934` 行
  - 编辑器缺少 dirty-state / beforeunload / draft recovery
  - inspector / dialog 中大量表单标签仍偏“视觉标签”，未完全形成无障碍友好的可编程关联
  - capability 列表与 bundle 规模已经出现扩展性信号
  - 前端缺少 Flow 编辑器专项自动化测试

#### Docs Governance Routing Decision

- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- docs tree 结构已存在且可用，本轮目标文档为 worktree 根 `plan.md`。
- Canonical destination:
  - 当前稳定行为真相 -> `docs/requirements/flow-editor-visual-form.md` + `docs/requirements/flow-editor-draft-reliability.md`
  - 当前技术契约 -> `docs/specs/flow-editor-visual-form.md` + `docs/specs/flow-editor-draft-reliability.md`
  - 本轮评估与后续执行控制面 -> worktree 根 `plan.md`
  - 完成实现后的结果 -> `docs/change/*`
- Requirements impact: `add`
- Specs impact: `add`
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\flow-editor-draft-reliability.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\flow-editor-visual-form.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\flow-editor-draft-reliability.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\flow-editor-visual-form.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`
- Related lessons:
  - `none`
- routing note:
  - 本轮已将 draft reliability 作为独立稳定能力补入 requirements/specs。
  - visual form 与 draft reliability 继续分治维护，避免把 editor shell 约束挤回 visual form 文档。

#### Executable Task List

- [x] `FLOW-ENH-0` 固化前端验证基线与 fresh worktree 启动步骤
- [x] `FLOW-ENH-1` 拆分 editor 壳层与 inspector/dialog 组件边界
- [x] `FLOW-ENH-2` 补齐草稿可靠性：dirty-state / 退出保护 / 恢复策略
- [x] `FLOW-ENH-3` 增强 capability + visual form 体验
- [x] `FLOW-ENH-4` 补齐无障碍与键盘交互约束
- [x] `FLOW-ENH-5` 增加 flow editor 自动化回归测试

#### Task Details

##### `FLOW-ENH-0` - 固化前端验证基线与 fresh worktree 启动步骤

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\plan.md`
- Goal:
  - 把 `wails generate module`、依赖安装和前端构建恢复为可重复的前置步骤，避免后续每轮都先被环境噪音阻塞。
- Files / Modules:
  - `README.md`
  - 如有必要：新增或调整本地开发脚本
- Write Set:
  - `README.md`
  - 可选的开发脚本文件
- Acceptance:
  - fresh worktree 下能按文档恢复 `frontend/wailsjs/**`
  - 前端构建命令路径清晰，不再靠 change log 翻历史记录
- Test Points:
  - `wails version`
  - `$env:GOWORK='off'; wails generate module`
  - `cd frontend && npm ci && npm run build`
- Rollback:
  - 回退新增文档和脚本说明，不影响运行时代码

##### `FLOW-ENH-1` - 拆分 editor 壳层与 inspector/dialog 组件边界

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\plan.md`
- Goal:
  - 将 `FlowEditorWindow.vue` 从“单体页面”收敛为壳层装配文件，把 inspector、capability picker、field binding dialog 等拆成独立组件。
- Files / Modules:
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/components/flow/editor/*`
  - 如有必要：`frontend/src/stores/flow.ts`
- Write Set:
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/components/flow/editor/*`
  - 必要时的 `frontend/src/stores/flow.ts`
- Acceptance:
  - 主要交互分区有独立组件边界
  - `FlowEditorWindow.vue` 只保留窗口生命周期、顶层协调和组合逻辑
  - 行为与当前编辑器保持一致
- Test Points:
  - `cd frontend && npm run build`
  - `wails dev` 手工冒烟：打开 editor、选节点、开 method dialog、开 field binding dialog
- Rollback:
  - 回退 `FlowEditorWindow.vue` 与新增 editor 组件

##### `FLOW-ENH-2` - 补齐草稿可靠性：dirty-state / 退出保护 / 恢复策略

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\plan.md`
- Goal:
  - 降低用户在 editor window 中丢失未保存 graph 改动的风险。
- Files / Modules:
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/stores/flowProjects.ts`
  - 视方案而定：`frontend/src/pages/Flow.vue`
- Write Set:
  - 上述 flow editor / project store 相关文件
- Acceptance:
  - 未保存改动时关闭窗口、刷新或切换关键路径会有明确保护
  - 若引入 autosave / recovery，行为边界清晰且可关闭或可解释
- Test Points:
  - 手工验证：修改 graph 后直接关闭或刷新窗口
  - `cd frontend && npm run build`
- Rollback:
  - 回退 dirty-state / autosave / recovery 相关改动，恢复手动保存模型

##### `FLOW-ENH-3` - 增强 capability + visual form 体验

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\plan.md`
- Goal:
  - 在保留现有 schema-driven 架构的前提下，提高 method 选择、字段 binding 和 unsupported reason 的可理解性。
- Files / Modules:
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/stores/flow_visual_form.ts`
  - `frontend/src/stores/flow_schema_resolver.ts`
  - `frontend/src/stores/flow_method_schemas.ts`
  - 如已拆分：`frontend/src/components/flow/editor/*`
- Write Set:
  - 仅限 flow editor 与 visual form 相关前端文件
- Acceptance:
  - capability picker 信息密度与可筛选性提升
  - binding/source 选择更少依赖手写 pointer 和记忆协议细节
  - visual form unavailable 的原因更可诊断
- Test Points:
  - `cd frontend && npm run build`
  - `wails dev` 手工冒烟：method 选择、field binding、form/json 切换
- Rollback:
  - 回退 visual form 体验层改动，不触碰底层 flow 协议

##### `FLOW-ENH-4` - 补齐无障碍与键盘交互约束

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\plan.md`
- Goal:
  - 让 inspector 和弹层表单达到更稳的可访问性和键盘可用性基线。
- Files / Modules:
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/pages/Flow.vue`
  - 视需要：`frontend/src/components/ui/*`
- Write Set:
  - flow / editor 相关页面与必要的基础 UI 组件
- Acceptance:
  - 关键表单控件有可编程可识别的标签
  - icon-only 操作具备明确 accessible name
  - 弹层打开、关闭和切换时焦点行为稳定
- Test Points:
  - 键盘 Tab / Shift+Tab 路径检查
  - `cd frontend && npm run build`
- Rollback:
  - 回退 accessibility 加固改动，恢复原 UI 结构

##### `FLOW-ENH-5` - 增加 flow editor 自动化回归测试

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\plan.md`
- Goal:
  - 为 schema resolver、visual form compatibility、binding build 和 editor 关键交互建立稳定回归口径。
- Files / Modules:
  - `frontend/package.json`
  - 测试配置文件
  - `frontend/src/stores/*` 或 `frontend/src/components/flow/editor/*` 的测试文件
- Write Set:
  - 前端测试相关配置与测试文件
- Acceptance:
  - 至少覆盖：
    - visual form compatibility
    - schema resolver 子集解析
    - ancestor / binding 基础校验
    - dirty-state 或关键 editor 交互中的一个主路径
- Test Points:
  - 待确认测试栈后执行相应测试命令
  - `cd frontend && npm run build`
- Rollback:
  - 回退新增测试栈与测试文件

### Stage 3.3 - Review

#### Implementation Summary

- `FLOW-ENH-1` 已完成：
  - `frontend/src/windows/FlowEditorWindow.vue` 收敛为壳层装配文件。
  - 新增 `frontend/src/components/flow/editor/*`，将 toolbar、inspector、method dialog、field binding dialog、add node dialog 拆出单独边界。
- `FLOW-ENH-2` 已完成：
  - `frontend/src/stores/flow.ts` 补充 graph editor state 导入导出与内容签名能力。
  - editor 壳层新增 dirty-state、`beforeunload` 保护和 `localStorage` 草稿恢复。
  - 保存成功后更新基线并清理恢复草稿，恢复逻辑仅基于 editor state，不依赖导出有效 spec。

#### Validation

- `$env:GOWORK='off'; wails generate module`：通过
- `cd frontend && npm run build`：通过
- 残余告警：
  - `dist/assets/index-*.js` 仍约 `955 kB`，chunk 过大告警保持不变，留待后续独立处理

#### Review Checklist

- 需求覆盖：通过
  - `FLOW-ENH-1` 已把 toolbar / inspector / dialog 边界从壳层中拆出。
  - `FLOW-ENH-2` 已提供 dirty-state、`beforeunload` 和 project-scoped 本地恢复。
- 架构合理性：通过
  - `FlowEditorWindow.vue` 只保留窗口壳层职责。
  - UI 子组件不直接承担项目保存或 recovery 语义。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - dirty signature 只基于 graph 内容。
  - recovery 写入为本地去抖写入，未增加后端 I/O。
- 可读性与一致性：通过
  - 顶层模板显著收敛，交互分区对应独立组件文件。
- 可扩展性与配置化：通过
  - 后续 capability / visual form 迭代可继续在 `frontend/src/components/flow/editor/*` 与 `flow_visual_form.ts` 扩展。
- 稳定性与安全：通过
  - 恢复逻辑校验 `projectId + baseSignature`，避免把旧草稿恢复到错误基线。
- 测试覆盖情况：通过（受当前批次范围约束）
  - `$env:GOWORK='off'; wails generate module` 与 `cd frontend && npm run build` 通过。
  - 自动化测试栈仍待 `FLOW-ENH-5` 独立建设；本轮未执行 `wails dev` 手工冒烟。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未派发子Agent，原因是 `FlowEditorWindow.vue` / `flow.ts` 写集高度重叠。

#### Dependencies

- 本轮已确认执行顺序：
  - `FLOW-ENH-1`
  - `FLOW-ENH-2`
- 后续推荐顺序：
  - `FLOW-ENH-0`
  - `FLOW-ENH-3` / `FLOW-ENH-4`
  - `FLOW-ENH-5`

#### Risks and Notes

- 若不先做 `FLOW-ENH-1`，后续功能增强仍会继续堆积到 `FlowEditorWindow.vue` 和 `flow.ts`。
- 若先做 `FLOW-ENH-3` 不做 `FLOW-ENH-2`，用户仍会在复杂编辑链路里承担较高的丢稿风险。
- `FLOW-ENH-5` 需要确认是否接受新增前端测试栈；当前仓库未见 flow editor 专项前端测试基线。
- 当前 worktree 的前端 build 已可通过，但这依赖先执行 `wails generate module`。

#### Parallelism Assessment

- 当前阶段不派发子Agent。
- 原因：
  - `FLOW-ENH-1` 与 `FLOW-ENH-2` 在 `FlowEditorWindow.vue` / `flow.ts` 上存在高耦合写集冲突。
  - 当前批次的关键路径是同一套 editor shell / recovery 语义，拆开并行会放大集成风险。

#### Issue List

- 已确认本轮先执行：
  - `FLOW-ENH-1`
  - `FLOW-ENH-2`
- `FLOW-ENH-5` 的测试栈选择留待后续独立确认，不影响当前实现批次。

阻塞：否
进入 4

### Stage 4 - Change Archive

- Archive Path:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\change\2026-03-24_win-flow-editor-shell-reliability.md`
- Requirements impact:
  - `updated`
- Specs impact:
  - `updated`
- Lessons impact:
  - `none`
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\flow-editor-draft-reliability.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\flow-editor-visual-form.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\flow-editor-draft-reliability.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\flow-editor-visual-form.md`
- Related lessons:
  - `none`
- Index updates:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\README.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\README.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\change\README.md`

### Stage 3.3 - Review (`FLOW-ENH-0`)

- 需求覆盖：通过
  - README 已补齐 fresh worktree 的前端 bootstrap 和验证路径。
- 架构合理性：通过
  - 仅更新 `README.md`，不引入额外脚本或运行时代码变更。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 无运行时影响；只增加开发期说明。
- 可读性与一致性：通过
  - 命令顺序与当前实际验证链路一致：`wails version` → `wails generate module` → `npm ci` → `npm run build`。
- 可扩展性与配置化：通过
  - 明确了 `frontend/wailsjs/**` 为 gitignore 生成物，后续新 worktree 可直接复用同一流程。
- 稳定性与安全：通过
  - 显式要求 `$env:GOWORK='off'; wails generate module`，减少 fresh worktree 误判和环境噪音。
- 测试覆盖情况：通过
  - `wails version`、`$env:GOWORK='off'; wails generate module`、`cd frontend && npm ci && npm run build` 均已实际执行通过。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未派发子Agent；任务为单文件文档收口，无并行收益。

### Stage 4 - Change Archive (`FLOW-ENH-0`)

- Archive Path:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\change\2026-03-24_win-flow-editor-bootstrap-baseline.md`
- Requirements impact:
  - `none`
- Specs impact:
  - `none`
- Lessons impact:
  - `none`
- Related requirements:
  - `none`
- Related specs:
  - `none`
- Related lessons:
  - `none`
- Index updates:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\change\README.md`

## Execution Results Summary

- `FLOW-ENH-0`：已完成。`README.md` 已补齐 fresh worktree 的前端 bootstrap 和验证步骤，实际命令已复跑通过。
- `FLOW-ENH-1`：已完成。`FlowEditorWindow.vue` 收敛为壳层文件，toolbar / inspector / dialogs 已拆出到 `frontend/src/components/flow/editor/*`。
- `FLOW-ENH-2`：已完成。editor 已具备 dirty-state、`beforeunload` 和 project-scoped 本地 recovery。
- requirements/specs：已补齐 draft reliability 的稳定文档与索引。
- 变更归档：已写入 `docs/change/2026-03-24_win-flow-editor-shell-reliability.md`。
- 变更归档：已写入 `docs/change/2026-03-24_win-flow-editor-bootstrap-baseline.md`。

已完成 Stage 4，等待用户确认是否结束 workflow

## Iteration Restart - `FLOW-ENH-3`

### Stage 1 - Requirements Analysis (`FLOW-ENH-3`)

#### 目标

- 在不修改 Flow 协议、visual form 适用范围和持久化模型的前提下，提升 `call` 节点编辑体验的可理解性和可诊断性。
- 把本轮范围限定在 capability 选择、字段 binding 编辑和 ordinary mode 不可用诊断，不扩展新节点能力。

#### 范围

##### 必须

- 提升 capability picker 的可筛选性和信息密度，但仍复用现有 capability query 返回结构。
- 降低字段 binding 编辑对“手记协议细节”的依赖，至少补充更清晰的目标/来源说明、预览和表单边界。
- 让 visual form unavailable 的原因更结构化、更可诊断，而不是仅堆原始 reason 字符串。
- 继续保持 `args_template + inputs`、schema resolver、compatibility 判定的现有契约不变。

##### 可选

- 为本轮新增的纯逻辑 helper 补最小自动化测试。
- 对 capability metadata 做轻量分组或 badge 展示。

##### 不做

- 不新增 Flow / Exec 协议字段。
- 不新增 `compose` 普通模式。
- 不做新的 autosave / recovery 行为。
- 不在本轮引入完整 editor 交互测试栈；更大范围回归测试仍留给 `FLOW-ENH-5`。

#### 使用场景

- 用户在能力列表中快速定位某个方法，并判断它是否带 schema、权限或标签信息。
- 用户给普通模式字段配置 binding 时，希望先理解“会从哪里读、会写到哪里”，而不是只看到裸 JSON Pointer。
- 用户遇到 ordinary mode 无法编辑的节点时，希望知道是 schema 缺失、binding 超范围，还是 template 超出覆盖范围。

#### 功能需求

1. capability picker 必须支持基于方法、节点、版本以及现有 metadata 的更宽松筛选。
2. capability picker 必须显示足够的 route metadata，帮助用户在不打开高级 JSON 的前提下判断目标 capability。
3. field binding dialog 必须给出当前目标字段和来源配置的更清晰摘要，并对明显无效状态提供表单级约束。
4. ordinary mode unavailable 的诊断必须来自现有 compatibility 规则，不得新造第二套判定逻辑。
5. 任何体验增强都不得改变保存后的 `args_template + inputs` 结果。

#### 非功能需求

- 变更面保持在 flow editor 前端和纯逻辑 helper 范围，避免把体验增强扩展成协议重构。
- 搜索和诊断增强不得引入无意义的 capability 重查或全图重复计算。
- 新增文案和诊断必须可维护，避免继续堆难以本地化的裸字符串。

#### 输入输出

- Inputs:
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-visual-form.md`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/components/flow/editor/FlowMethodPickerDialog.vue`
  - `frontend/src/components/flow/editor/FlowFieldBindingDialog.vue`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/stores/flow_visual_form.ts`
  - `frontend/src/stores/flow_schema_resolver.ts`
  - `frontend/src/i18n/messages/automation.ts`
- Outputs:
  - 更可扫描的 capability picker
  - 更可解释的字段 binding 编辑体验
  - 更结构化的 visual form 兼容性诊断

#### 边界异常

- capability route 可能不带 `input_schema`、`permissions` 或 `tags`。
- field binding 对话框在当前节点没有可绑定祖先时，仍必须允许 `trigger` / meta 来源。
- compatibility 失败原因可能同时来自 template、binding 和 schema 覆盖范围，UI 不应误导成单一问题。

#### 验收标准

1. 用户可以通过增强后的 picker 更快区分当前节点、本地/远端、schema 可用性及权限/标签元数据。
2. 字段 binding dialog 在不改协议的前提下，减少对裸 pointer 和内部术语的暴露，并提供明确预览。
3. unsupported ordinary mode 的原因展示更可读，且与实际 compatibility 判定一致。
4. `cd frontend && npm run build` 继续通过。

#### 风险

- 若把诊断增强直接写死在组件里，后续兼容性规则变化时会再次出现文案与判定脱节。
- 若 capability picker 只改样式不补元数据和检索，体验收益会很有限。
- 若本轮顺手扩大到协议或新表单能力，会打破当前“体验增强不改契约”的范围控制。

#### Issue List

- 无

阻塞：否
进入 2

### Stage 2 - Architecture Design (`FLOW-ENH-3`)

#### 总体方案

- 继续沿用现有 `flow.ts -> flow_visual_form.ts -> editor components` 的职责边界。
- 体验增强优先落在三个层面：
  - `FlowEditorWindow.vue`：增强 capability 过滤与 dialog 装配数据。
  - `flow_visual_form.ts`：把 binding 摘要和 compatibility reason 收敛为可复用的结构化 helper。
  - `editor/*` 组件：展示 richer metadata、preview 和诊断说明。
- 不调整 `flow_schema_resolver.ts` 的 schema 子集约束，避免把 UI 增强误扩成 schema 契约变更。

#### 模块职责

- `frontend/src/windows/FlowEditorWindow.vue`
  - 负责 capability 列表排序 / 搜索策略，以及 field binding dialog 所需状态装配。
- `frontend/src/stores/flow_visual_form.ts`
  - 负责把底层 binding 和 compatibility 失败原因转成可展示、可测试的结构化描述。
- `frontend/src/components/flow/editor/FlowMethodPickerDialog.vue`
  - 负责 richer capability row 展示，不写业务规则。
- `frontend/src/components/flow/editor/FlowFieldBindingDialog.vue`
  - 负责目标字段摘要、来源预览、表单级 helper text 与 apply gating。
- `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - 负责渲染 structured compatibility diagnostics 和更友好的 binding summary。

#### 数据 / 调用流

1. `FlowEditorWindow.vue` 基于现有 `execCapabilities` 生成排序后的候选列表。
2. 搜索逻辑同时匹配方法、provider/via/version、permissions、tags 和已有 label。
3. `flow_visual_form.ts` 继续从 `args_template + inputs + schema` 构建 visual form model，但 compatibility reason 改为结构化描述。
4. inspector 和 field binding dialog 基于统一 helper 呈现 binding summary / diagnostics，不直接重复拼文案。

#### 接口草案

- `flow_visual_form.ts`
  - 新增结构化 compatibility reason 类型与描述 helper。
  - 保留 `describeFieldBinding(...)` 出口，但输出更偏用户可读摘要。
- `FlowMethodPickerDialog.vue`
  - 复用现有 `ExecCapabilityRoute[]`，不新增后端接口字段。
- `FlowFieldBindingDialog.vue`
  - 复用现有 `fieldBindingDraft`，不新增持久化状态。

#### 错误与安全

- field binding apply 仍走 store 校验，不在组件里吞掉非法状态。
- compatibility 诊断必须基于 store/helper 输出，避免 UI 自行猜测真实失败原因。
- 搜索仅使用已加载 metadata，不触发隐式网络或后端请求。

#### 性能与测试策略

- capability 搜索仍基于本地已加载列表，不新增 I/O。
- compatibility helper 仅处理当前节点数据，不扩大图级计算范围。
- 若本轮补自动化测试，只覆盖 `flow_visual_form.ts` 的纯逻辑回归，不扩张到完整 editor 测试栈。

#### 可扩展性设计点

- 结构化 compatibility reason 可被后续无障碍和测试直接复用，避免继续依赖英文裸字符串。
- richer capability metadata 展示不依赖特定方法，后续 capability 类型扩展可直接复用。
- field binding 预览和 helper 文案集中在 dialog/helper 层，后续若新增 path picker 也可平滑替换。

#### Issue List

- 无

阻塞：否
进入 3.1

### Stage 3.1 - Planning (`FLOW-ENH-3`)

#### Project Goal and Current State

- 当前 stable truth 仍由 `docs/requirements/flow-editor-visual-form.md` 和 `docs/specs/flow-editor-visual-form.md` 提供。
- `FLOW-ENH-1/2` 已提供清晰组件边界和草稿可靠性，本轮只在该基础上补 capability / visual form 体验。

#### Docs Governance Routing Decision

- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- docs tree 不需要 bootstrap；本轮控制文档仍为 worktree 根 `plan.md`。
- Canonical destination:
  - 稳定需求 / 契约：维持在现有 `docs/requirements/flow-editor-visual-form.md`、`docs/specs/flow-editor-visual-form.md`
  - 本轮实现结果：`docs/change/YYYY-MM-DD_topic.md`
  - lessons：当前未发现必须独立沉淀的排障知识
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\flow-editor-visual-form.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\flow-editor-visual-form.md`
- Related lessons:
  - `none`

#### Executable Checklist

- [x] `FLOW-ENH-3-A` 扩展 capability picker 的本地筛选维度与 metadata 展示
- [x] `FLOW-ENH-3-B` 收敛 binding summary / preview / structured compatibility reason helper
- [x] `FLOW-ENH-3-C` 在 inspector 和 field binding dialog 中落地更可解释的体验
- [x] `FLOW-ENH-3-D` 补最小纯逻辑测试

#### Task Details

##### `FLOW-ENH-3-A` - capability picker 信息密度与检索增强

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\plan.md`
- Files / Modules:
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/components/flow/editor/FlowMethodPickerDialog.vue`
  - `frontend/src/i18n/messages/automation.ts`
- Write Set:
  - 仅限上述 capability picker 相关文件
- Acceptance:
  - 过滤支持方法 / provider / via / version / permission / tag 等现有 metadata
  - 列表项可直接看出 self/remote、schema 可用性和主要 metadata
- Tests:
  - `cd frontend && npm run build`
- Rollback:
  - 回退 capability picker 相关 UI 与搜索逻辑

##### `FLOW-ENH-3-B` - binding / compatibility helper 收敛

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\plan.md`
- Files / Modules:
  - `frontend/src/stores/flow_visual_form.ts`
  - `frontend/src/stores/flow.ts`
  - 可选：测试文件
- Write Set:
  - 仅限 visual form helper 与对应测试文件
- Acceptance:
  - binding summary 更偏用户可读
  - compatibility reasons 可被 UI 结构化消费，而不是只依赖原始英文字符串
- Tests:
  - `cd frontend && npm run build`
  - 若补测试，则执行新增测试命令
- Rollback:
  - 回退 helper 与测试文件，恢复现有描述方式

##### `FLOW-ENH-3-C` - inspector / field binding dialog 体验增强

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\plan.md`
- Files / Modules:
  - `frontend/src/components/flow/editor/FlowFieldBindingDialog.vue`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/i18n/messages/automation.ts`
- Write Set:
  - 仅限 inspector / dialog 和对应文案
- Acceptance:
  - field binding dialog 给出目标字段、来源预览和表单边界
  - inspector 对 unavailable ordinary mode 的原因展示更可解释
- Tests:
  - `cd frontend && npm run build`
- Rollback:
  - 回退 inspector / dialog 体验层改动

##### `FLOW-ENH-3-D` - 最小纯逻辑测试（可选）

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\plan.md`
- Files / Modules:
  - 仅限 flow visual helper 测试栈所需最小文件
- Write Set:
  - 测试配置与 `flow_visual_form` 相关测试文件
- Acceptance:
  - 至少覆盖本轮新增 helper 的核心分支
- Tests:
  - 执行新增测试命令
- Rollback:
  - 回退本轮新增测试配置和测试文件

#### Dependencies

- 依赖已完成的 `FLOW-ENH-1` / `FLOW-ENH-2` 组件边界与 editor state 能力。
- 不依赖 requirements/specs 更新即可进入实现。

#### Risks and Notes

- 仓库目前没有现成前端测试栈；若补最小测试，只应覆盖纯逻辑，避免把本轮演变成 `FLOW-ENH-5`。
- `flow.ts` 与 `flow_visual_form.ts` 已有未提交改动，本轮必须在其现有基础上增量修改，不回退前序实现。

#### Parallelism Assessment

- 本轮不派发子Agent。
- 原因：
  - capability picker、binding helper、inspector/dialog 文案与类型互相耦合，拆分并行的集成收益不高。
  - 当前关键路径集中在同一批 editor 文件，写集冲突概率高。

#### Issue List

- 无

阻塞：否
进入 3.2

## Iteration Restart - `FLOW-ENH-5`

### Stage 1 - Requirements Analysis (`FLOW-ENH-5`)

#### 目标

- 为 Win Flow 编辑器补齐可重复执行的自动化回归测试基线，优先覆盖已经写入 requirements/specs 的关键行为。
- 本轮不再扩新功能，而是把 schema resolver、store 级 binding / draft-state 行为收敛为稳定测试口径。

#### 范围

##### 必须

- 为 `flow_schema_resolver.ts` 增加 capability schema 子集解析回归测试。
- 为 `flow.ts` 增加 ancestor、binding、spec mode、graph editor state / dirty-signature 相关回归测试。
- 沿用现有 `vitest` 测试栈，不引入第二套前端测试框架。
- 保持测试只验证既有 requirements/specs，不在测试实现里偷偷改行为契约。

##### 可选

- 若成本足够低，可补一个 editor 级交互主路径；否则维持现有 `Overlay.test.ts` 作为 UI 层基线。

##### 不做

- 不新增 Flow 编辑器运行时功能。
- 不修改 `docs/requirements/*` 或 `docs/specs/*` 的稳定行为描述。
- 不把本轮扩大成完整浏览器 E2E 测试建设。

#### 使用场景

- 开发者修改 visual form schema 解析逻辑后，需要快速判断 capability `input_schema` 子集是否仍能正确落到字段模型。
- 开发者修改 `flow.ts` 中的 binding、ancestor 或 draft-state 逻辑后，需要快速发现导出 graph、JSON/form 切换或 dirty 判定回归。
- 后续继续扩 capability / visual form 体验时，需要一条低成本自动化验证链，而不是只靠手工冒烟。

#### 功能需求

1. 自动化测试必须覆盖 capability schema 解析优先级与失败回退。
2. 自动化测试必须覆盖 ancestor 约束下的字段 binding 写入与清理。
3. 自动化测试必须覆盖 graph editor state 的导入导出或 dirty-signature 关键边界中的至少一个主路径。
4. 自动化测试必须覆盖 JSON/form spec mode 切换的至少一个稳定回写路径。
5. 测试命令必须可以通过 `cd frontend && npm test` 执行。

#### 非功能需求

- 变更应尽量局限在测试文件，不扩大运行时代码写面。
- 测试应避免依赖 Wails 运行时；若可在纯前端环境验证，则不引入额外 mock 基础设施。
- 测试断言应尽量基于公开 store API 和稳定输出，避免过度耦合内部实现细节。

#### 输入输出

- Inputs:
  - `docs/requirements/flow-editor-visual-form.md`
  - `docs/requirements/flow-editor-draft-reliability.md`
  - `docs/specs/flow-editor-visual-form.md`
  - `docs/specs/flow-editor-draft-reliability.md`
  - `frontend/src/stores/flow_schema_resolver.ts`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/stores/flow_visual_form.test.ts`
  - `frontend/src/components/ui/overlay/Overlay.test.ts`
  - `frontend/vitest.config.ts`
- Outputs:
  - `flow_schema_resolver` 的自动化测试文件
  - `flow.ts` 的自动化测试文件
  - 可重复执行的前端测试验证结果

#### 边界异常

- capability `input_schema` 可能是 JSON 字符串，也可能是对象。
- 本地 override schema 必须继续优先于 capability schema。
- `flow.ts` 是共享 singleton store；测试之间必须显式重置 graph state，避免互相污染。
- graph editor dirty-signature 不应因选择态变化而变化。

#### 验收标准

1. `cd frontend && npm test` 通过，且新增用例覆盖 resolver 与 store 两类主路径。
2. resolver 测试能识别支持子集、嵌套对象展开、enum/select 映射和不支持特性拒绝。
3. store 测试能识别祖先 binding 限制、binding 清理、spec mode 切换和 graph editor state / signature 的稳定边界。
4. `cd frontend && npm run build` 继续通过。

#### 风险

- 若测试直接耦合私有实现细节，后续重构会造成高噪音失败。
- 若不显式重置共享 store，测试结果会随机受前序用例污染。
- 若把本轮扩大到组件级大范围交互测试，会拖慢当前回归基线落地。

#### Issue List

- 无

阻塞：否
进入 2

### Stage 2 - Architecture Design (`FLOW-ENH-5`)

#### 总体方案

- 采用现有 `vitest` 栈继续扩展，不新增 Jest、Playwright 或 Cypress。
- 测试分两层：
  - `flow_schema_resolver.test.ts`：纯函数级覆盖 schema 解析、fallback 与 clone 边界。
  - `flow.test.ts`：通过 `useFlowStore()` 的公开 API 覆盖 graph ancestor、binding、JSON/form mode、editor state / signature。
- 放弃把本轮主目标放在 editor 级 UI 交互测试：
  - 原因：`Overlay.test.ts` 已覆盖当前最关键的 dialog 焦点路径。
  - 当前更缺的是协议与草稿状态层的稳定回归口径。

#### 模块职责

- `frontend/src/stores/flow_schema_resolver.test.ts`
  - 负责校验 capability schema 子集解析、local override precedence 和 unsupported feature rejection。
- `frontend/src/stores/flow.test.ts`
  - 负责校验 `useFlowStore()` 暴露的 graph 编辑 API，而不是直接调用私有 helper。
- `frontend/vitest.config.ts`
  - 保持当前 node 环境基线；如个别测试需要 DOM，则继续使用文件级环境声明，而不是全局切换。
- 现有 `flow_visual_form.test.ts` / `Overlay.test.ts`
  - 继续作为 visual compatibility 与 overlay 焦点管理的既有基线，不在本轮重写。

#### 数据 / 调用流

1. `vitest` 在 `frontend/` 下启动。
2. resolver 测试直接调用 `resolveMethodVisualSchema(...)`，比较输出 schema 结构。
3. store 测试通过 `useFlowStore()` 获取共享 store。
4. 每个 store 用例先重置 graph editor state，再执行公开 API：
   - `loadGraphEditorState(...)`
   - `setFieldBinding(...)`
   - `clearFieldBinding(...)`
   - `setFieldLiteralValue(...)`
   - `setNodeSpecEditorMode(...)`
   - `exportGraphDraft(...)`
   - `exportGraphEditorState(...)`
   - `graphEditorSignature(...)`
5. 测试以导出的 graph / editor state / signature 作为断言出口。

#### 接口草案

- 新增测试文件：
  - `frontend/src/stores/flow_schema_resolver.test.ts`
  - `frontend/src/stores/flow.test.ts`
- 推荐测试夹具：
  - call node builder
  - store reset helper
  - shared locale setup (`setLocale("en")`)

#### 错误与安全

- 非祖先 `node_result` binding 仍应通过 store API 立即抛错。
- 非法 JSON Pointer、非法 advanced spec 或缺失 method 的情况，测试只验证现有报错 / 校验出口，不修改运行时代码。
- 测试不得依赖真实 `window.go.flow.FlowService`，避免把 Wails 绑定变成前端单测前置条件。

#### 性能与测试策略

- 优先选择纯函数和 store 级测试，执行速度快、维护成本低。
- 使用显式 reset 避免测试间共享状态导致重复构建或随机失败。
- 验证链保持：
  - `cd frontend && npm test`
  - `cd frontend && npm run build`
  - `$env:GOWORK='off'; wails generate module`

#### 可扩展性设计点

- resolver 测试覆盖 capability schema 子集后，后续若扩 `input_schema` 支持范围，可先改测试再扩实现。
- store 测试覆盖公开 API 后，内部 helper 拆分或重构时仍能保持回归口径稳定。
- 继续把 editor UI 级更重的测试留给后续真正需要时再单独规划，避免当前测试体系过早复杂化。

#### Issue List

- 无

阻塞：否
进入 3.1

### Stage 3.1 - Planning (`FLOW-ENH-5`)

#### Project Goal and Current State

- 当前已具备最小前端测试栈：
  - `frontend/package.json` 已提供 `npm test -> vitest run`
  - `frontend/vitest.config.ts` 已可执行 node 环境测试
  - 已有 `flow_visual_form.test.ts` 与 `Overlay.test.ts` 作为前序基线
- 当前缺口：
  - `flow_schema_resolver.ts` 仍无自动化覆盖
  - `flow.ts` 的 ancestor / binding / spec mode / graph editor state 仍无自动化覆盖
  - dirty-state 对应的 signature 约束还没有测试保护

#### Docs Governance Routing Decision

- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- docs tree 结构已存在且可用，本轮目标文档为 worktree 根 `plan.md`，完成后归档到 `docs/change/*`。
- Canonical destination:
  - 稳定行为真相 -> `docs/requirements/flow-editor-visual-form.md` + `docs/requirements/flow-editor-draft-reliability.md`
  - 技术契约 -> `docs/specs/flow-editor-visual-form.md` + `docs/specs/flow-editor-draft-reliability.md`
  - 本轮执行控制面 -> worktree 根 `plan.md`
  - 本轮完成结果 -> `docs/change/*`
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\flow-editor-draft-reliability.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\flow-editor-visual-form.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\flow-editor-draft-reliability.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\flow-editor-visual-form.md`
- Related lessons:
  - `none`

#### Executable Checklist

- [x] `FLOW-ENH-5-A` 为 `flow_schema_resolver.ts` 增加 capability schema 解析测试
- [x] `FLOW-ENH-5-B` 为 `flow.ts` 增加 ancestor / binding / spec mode / graph state 测试
- [x] `FLOW-ENH-5-C` 执行前端测试、构建与 Wails 基线验证

#### Task Details

##### `FLOW-ENH-5-A` - Schema Resolver 回归测试

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\plan.md`
- Files / Modules:
  - `frontend/src/stores/flow_schema_resolver.test.ts`
  - 如有必要：`frontend/vitest.config.ts`
- Write Set:
  - resolver 测试文件与必要的最小测试配置
- Acceptance:
  - 覆盖 local override 优先级
  - 覆盖 capability schema 的对象字段展开与 enum/select 映射
  - 覆盖不支持 schema 特性时返回 `null`
- Tests:
  - `cd frontend && npm test`
- Rollback:
  - 回退 resolver 测试文件与相关最小配置

##### `FLOW-ENH-5-B` - Flow Store 回归测试

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\plan.md`
- Files / Modules:
  - `frontend/src/stores/flow.test.ts`
- Write Set:
  - 仅限 `flow.ts` 对应测试文件
- Acceptance:
  - 覆盖 ancestor 列表与非祖先 binding 拒绝
  - 覆盖 binding 写入 / 清理与导出 graph
  - 覆盖 JSON/form spec mode 切换至少一个主路径
  - 覆盖 graph editor state / signature 对 dirty-state 边界的保护
- Tests:
  - `cd frontend && npm test`
- Rollback:
  - 回退 store 测试文件

##### `FLOW-ENH-5-C` - 验证与归档准备

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\plan.md`
- Files / Modules:
  - `plan.md`
  - `docs/change/*`
  - `docs/change/README.md`
- Write Set:
  - 仅限 workflow 控制面和归档文档
- Acceptance:
  - `npm test`、`npm run build`、`wails generate module` 结果被记录
  - review 与 archive 明确记录 requirements/specs impact 为 `none`
- Tests:
  - `$env:GOWORK='off'; wails generate module`
  - `cd frontend && npm test`
  - `cd frontend && npm run build`
- Rollback:
  - 回退本轮新增测试与归档文档

#### Dependencies

- 依赖已完成的 `FLOW-ENH-3` visual form compatibility 测试基线和 `FLOW-ENH-4` overlay 测试基线。
- 不依赖新的 requirements/specs 更新，可以直接进入实现。

#### Risks and Notes

- `flow.ts` 为共享 singleton store，本轮测试必须显式 reset state，不能依赖进程级初始值。
- 本轮优先覆盖 store / resolver 公开边界，不把测试扩大到重 UI 交互路径。
- 现有 build 仍会提示大 chunk 告警，本轮不处理拆包。

#### Parallelism Assessment

- 本轮不派发子Agent。
- 原因：
  - `flow_schema_resolver.test.ts` 与 `flow.test.ts` 写集虽然分离，但当前实现与验证量都较小，主Agent 直接完成更快。
  - 共享 store 的测试夹具和验证链路需要集中复核，拆开并行收益有限。

#### Issue List

- 无

阻塞：否
进入 3.2

### Stage 3.3 - Review (`FLOW-ENH-5`)

- 需求覆盖：通过
  - `flow_schema_resolver.test.ts` 已覆盖 local override 优先级、capability schema 子集解析以及 unsupported feature rejection。
  - `flow.test.ts` 已覆盖 ancestor 列表、非祖先 binding 拒绝、binding 写入 / 清理、graph draft 导出、JSON/form mode 切换和 graph editor signature 稳定性。
  - 既有 `flow_visual_form.test.ts` 与 `Overlay.test.ts` 继续覆盖 visual compatibility 与 overlay 焦点主路径。
- 架构合理性：通过
  - 本轮沿用现有 `vitest` 栈，没有引入第二套前端测试框架。
  - store 回归测试只通过 `useFlowStore()` 公开 API 断言，没有把测试耦合到私有 helper。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 新增测试均为纯前端单测，不增加运行时 I/O。
  - 共享 store 通过每用例显式 reset，避免用例间累积状态导致额外重算或随机失败。
- 可读性与一致性：通过
  - 测试夹具收敛在 `createCallNode` / `loadGraph` helper，中英文断言口径与现有测试保持一致。
- 可扩展性与配置化：通过
  - resolver 测试锁定了 capability schema 子集边界，后续扩展可先改测试再改实现。
  - store 测试围绕公开导出接口，允许内部重构而不丢失回归口径。
- 稳定性与安全：通过
  - 测试不依赖真实 Wails 绑定，不会因宿主环境差异导致前端回归基线失效。
  - 非祖先 binding、spec mode 切换和 signature 边界均有自动化保护。
- 测试覆盖情况：通过
  - `cd frontend && npm test` 通过，共 4 个测试文件、13 个用例全部通过。
  - `cd frontend && npm run build` 通过，保留单 chunk 约 `969.91 kB` 的既有告警。
  - `$env:GOWORK='off'; wails generate module` 通过，命令仍打印 `Not found: time.Time` 但最终成功。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未派发子Agent；测试写集和验证链较小，由主Agent 集中实现与复核。

#### Dependencies

- 本轮复用了既有 `flow_visual_form.test.ts` 和 `Overlay.test.ts` 的基线。
- 未引入新的 requirements/specs 依赖或新的测试框架依赖。

#### Risks and Notes

- `flow.ts` 仍是共享 singleton store，后续继续扩测试时仍需遵守显式 reset 约束。
- 当前 build 的大 chunk 告警和 `wails generate module` 的 `Not found: time.Time` 仍为既有残余，不是本轮新增问题。

#### Issue List

- 无

阻塞：否
进入 4

### Stage 4 - Change Archive (`FLOW-ENH-5`)

- Archive Path:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\change\2026-03-24_win-flow-editor-regression-tests.md`
- Requirements impact:
  - `none`
- Specs impact:
  - `none`
- Lessons impact:
  - `none`
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\flow-editor-draft-reliability.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\flow-editor-visual-form.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\flow-editor-draft-reliability.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\flow-editor-visual-form.md`
- Related lessons:
  - `none`
- Index updates:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\change\README.md`

### Stage 3.3 - Review (`FLOW-ENH-4`)

- 需求覆盖：通过
  - editor overlay 已支持可选 focus trap、初始焦点与关闭后焦点恢复。
  - `Add Node`、`Select Capability`、`Bind Field` 已补 dialog 语义和关键表单标签。
  - inspector 已补区域语义、动态字段 `aria-*` 关联和 Escape 关闭路径。
- 架构合理性：通过
  - 焦点管理集中在 `Overlay.vue`，dialog 组件只声明语义和初始焦点。
  - inspector 关闭逻辑继续挂在 `FlowEditorWindow.vue` 的 editor 级键盘入口，没有把窗口级监听分散到子组件。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 焦点管理只在 overlay 打开、关闭和 Tab 事件时运行，没有新增轮询或后端调用。
  - 标签增强仅增加 DOM 属性和少量 id 生成逻辑。
- 可读性与一致性：通过
  - dialog 语义和标签关联保持统一模式：`role="dialog"` + `aria-labelledby` + 初始焦点选择器。
  - inspector 动态字段 id / aria 由统一 helper 生成，避免散落硬编码。
- 可扩展性与配置化：通过
  - `Overlay` 的焦点管理以可选 prop 形式暴露，可逐步复用到其他页面而不是全局强制开启。
  - 新增 accessibility requirements/specs 为后续 editor 扩展提供稳定边界。
- 稳定性与安全：通过
  - 焦点恢复会在原元素仍存在时才执行，否则静默跳过。
  - Tab 环路只作用于当前打开且显式启用 `trapFocus` 的 overlay。
- 测试覆盖情况：通过
  - `cd frontend && npm test` 通过，新增 `Overlay.test.ts` 覆盖焦点进入、Tab 环路和关闭后焦点恢复。
  - `cd frontend && npm run build` 通过。
  - `$env:GOWORK='off'; wails generate module` 通过。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未派发子Agent；overlay + dialog + inspector 焦点路径属于同一高耦合关键链路。

#### Dependencies

- `FLOW-ENH-4` 复用了 `FLOW-ENH-1` 的 editor 组件边界和 `FLOW-ENH-3` 的 dialog / inspector 结构。
- 本轮新增的 accessibility docs 已作为稳定真相补齐。

#### Risks and Notes

- 当前覆盖仍聚焦 editor overlay 与 inspector，Flow 画布本体的完整键盘模型仍不在本轮范围。
- 前端 bundle 单 chunk 告警仍存在，本轮未处理拆分。
- `wails generate module` 仍会打印 `Not found: time.Time`，但命令最终通过。

#### Issue List

- 无

阻塞：否
进入 4

### Stage 4 - Change Archive (`FLOW-ENH-4`)

- Archive Path:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\change\2026-03-24_win-flow-editor-accessibility.md`
- Requirements impact:
  - `updated`
- Specs impact:
  - `updated`
- Lessons impact:
  - `none`
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\flow-editor-accessibility.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\flow-editor-visual-form.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\flow-editor-accessibility.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\flow-editor-visual-form.md`
- Related lessons:
  - `none`
- Index updates:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\README.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\README.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\change\README.md`

### Stage 3.3 - Review (`FLOW-ENH-3`)

- 需求覆盖：通过
  - capability picker 已支持基于方法 / 节点 / 版本 / permission / tag 的本地筛选，并补充 schema / timeout / metadata 展示。
  - field binding dialog 已补目标字段说明、来源预览、路径 helper 和 apply gating。
  - ordinary mode unavailable 已改为结构化原因展示，不再直接堆原始英文字符串。
- 架构合理性：通过
  - compatibility reason 和 binding 摘要收敛在 `flow_visual_form.ts`，组件层只负责展示。
  - capability picker 仍只消费现有 `ExecCapabilityRoute`，未引入新后端契约。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - capability 搜索只在已加载列表上做本地匹配，没有新增网络请求。
  - compatibility helper 仍只围绕当前节点数据运行，没有扩大图级扫描范围。
- 可读性与一致性：通过
  - binding summary、compatibility reason、dialog helper text 由统一 helper / 文案出口提供，减少散落字符串拼接。
- 可扩展性与配置化：通过
  - 结构化 reason 可被后续无障碍、测试和更多诊断 UI 复用。
  - 最小 `vitest` 仅覆盖纯逻辑，为后续 `FLOW-ENH-5` 保留继续扩展空间。
- 稳定性与安全：通过
  - field binding 仍依赖 store 做最终校验；组件只增加表单级防呆，不吞掉错误。
  - 本轮未更改 `args_template + inputs` 写回契约。
- 测试覆盖情况：通过
  - `cd frontend && npm test` 通过，覆盖 binding 摘要和 compatibility reason 核心分支。
  - `cd frontend && npm run build` 通过。
  - `$env:GOWORK='off'; wails generate module` 通过。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未派发子Agent；editor 体验层与 helper 写集高度耦合，由主Agent 集中完成和复核。

#### Dependencies

- `FLOW-ENH-3` 复用了已完成的 `FLOW-ENH-1/2` 组件边界和 editor state 能力。
- 最小测试栈只覆盖纯逻辑，不替代后续 `FLOW-ENH-5` 的 editor 回归测试。

#### Risks and Notes

- 前端产物仍存在单 chunk 约 `964 kB` 的告警，本轮未处理 chunk 拆分。
- `wails generate module` 过程中仍会打印 `Not found: time.Time`，但命令最终通过，与本轮改动无直接冲突。

#### Issue List

- 无

阻塞：否
进入 4

### Stage 4 - Change Archive (`FLOW-ENH-3`)

- Archive Path:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\change\2026-03-24_win-flow-editor-visual-form-ux.md`
- Requirements impact:
  - `none`
- Specs impact:
  - `none`
- Lessons impact:
  - `none`
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\flow-editor-visual-form.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\flow-editor-visual-form.md`
- Related lessons:
  - `none`
- Index updates:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\change\README.md`

## Iteration Restart - `FLOW-ENH-4`

### Stage 1 - Requirements Analysis (`FLOW-ENH-4`)

#### 目标

- 为 Win Flow 编辑器建立稳定的无障碍与键盘交互基线，而不是继续把可访问性问题留在零散 UI 细节里。
- 本轮优先解决 editor 弹层焦点、关键表单标签、icon-only 名称和 inspector 基础键盘退出。

#### 范围

##### 必须

- editor 弹层打开后把焦点带入弹层，关闭后恢复到之前焦点。
- editor 弹层在打开期间保持稳定的 Tab / Shift+Tab 焦点循环。
- editor 关键表单控件补齐可编程可识别标签。
- inspector 至少补齐明确区域语义和 Escape 关闭路径。

##### 可选

- 为选择态按钮补充状态语义。
- 为字段说明补充 `aria-describedby` 关联。

##### 不做

- 不为 Flow 画布本体建立完整键盘编辑模型。
- 不做全站 accessibility 审计。
- 不引入新的状态管理或持久化协议字段。

#### 使用场景

- 用户只用键盘打开 “Add Node”“Select Capability”“Bind Field” 并完成输入。
- 用户用屏幕阅读器读取 ordinary mode 字段时，能知道输入框对应哪个字段和说明。
- 用户在 inspector 打开时，希望用 Escape 快速关闭当前节点详情。

#### 功能需求

1. editor 关键输入必须具备 `label/for`、`aria-labelledby` 或 `aria-label` 中的一种稳定标签机制。
2. icon-only 按钮必须具备明确 accessible name。
3. 弹层打开时必须把焦点移入当前弹层，关闭时恢复先前焦点。
4. 打开中的弹层必须拦截 Tab / Shift+Tab，避免焦点移出当前弹层。
5. 在未打开 editor 弹层且当前焦点不在可编辑控件中时，Escape 应允许关闭 inspector。

#### 非功能需求

- 焦点管理应优先复用现有 `Overlay`，避免再造一套 editor 专用弹层基础设施。
- 这轮改动不得影响 graph 保存、method 选择、binding 写回或 visual form 兼容性判定。
- 焦点恢复失败时必须安全降级，不抛出破坏性异常。

#### 输入输出

- Inputs:
  - `docs/requirements/flow-editor-accessibility.md`
  - `docs/specs/flow-editor-accessibility.md`
  - `frontend/src/components/ui/overlay/Overlay.vue`
  - `frontend/src/components/flow/editor/FlowAddNodeDialog.vue`
  - `frontend/src/components/flow/editor/FlowMethodPickerDialog.vue`
  - `frontend/src/components/flow/editor/FlowFieldBindingDialog.vue`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
- Outputs:
  - editor 弹层稳定的焦点进入 / 焦点恢复 / Tab 环路
  - 关键表单控件与 inspector 的基础无障碍语义

#### 边界异常

- 弹层内可能没有显式输入框，只存在按钮或只读内容。
- 弹层可能因 Escape、backdrop、确认按钮或业务状态切换而关闭。
- inspector 中存在动态 visual form 字段和 binding 列表，标签与描述关系仍需可识别。

#### 验收标准

1. `Add Node`、`Select Capability`、`Bind Field` 打开后焦点落点稳定，关闭后恢复焦点。
2. 打开的 editor 弹层 Tab / Shift+Tab 路径稳定，不会把焦点带出当前弹层。
3. inspector 和 editor 关键表单具备明确标签或等效 accessible name。
4. inspector 支持 Escape 关闭。

#### 风险

- 若只在个别组件手补焦点逻辑，后续新增弹层会再次漏掉相同问题。
- 若把焦点管理做成全局强制行为，可能影响当前 repo 中其他 Overlay 使用场景。
- 若标签增强散落在模板中没有约定，会继续增加维护成本。

#### Issue List

- 无

阻塞：否
进入 2

### Stage 2 - Architecture Design (`FLOW-ENH-4`)

#### 总体方案

- 这轮以 `Overlay` 为焦点管理中心，editor 各弹层只声明是否启用 focus trap 和初始焦点目标。
- 表单标签增强分两类处理：
  - 简单输入：`label[for] + id`
  - 动态 visual form 字段：`aria-labelledby` + `aria-describedby`
- inspector 的键盘退出由 `FlowEditorWindow.vue` 现有全局快捷键入口接管，避免在子组件里再挂一套窗口级监听。

#### 模块职责

- `frontend/src/components/ui/overlay/Overlay.vue`
  - 负责可选 focus trap、初始焦点和关闭后的焦点恢复。
- `frontend/src/components/flow/editor/FlowAddNodeDialog.vue`
  - 负责新增节点表单标签、dialog 语义和初始焦点声明。
- `frontend/src/components/flow/editor/FlowMethodPickerDialog.vue`
  - 负责 method picker 的 dialog 语义、可编程标签和初始焦点声明。
- `frontend/src/components/flow/editor/FlowFieldBindingDialog.vue`
  - 负责 binding dialog 的标签、描述和 dialog 语义。
- `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - 负责 inspector 区域语义、表单标签和动态字段的 `aria-*` 关联。
- `frontend/src/windows/FlowEditorWindow.vue`
  - 负责 inspector 的 Escape 关闭路径和 editor 级快捷键边界。

#### 数据 / 调用流

1. 用户触发 editor 弹层打开。
2. dialog 通过 `Overlay` 启用 focus trap，并把焦点移动到声明的初始目标或首个可聚焦元素。
3. 用户在 dialog 内使用 Tab / Shift+Tab 时，焦点循环保持在当前 overlay 内。
4. dialog 关闭时，`Overlay` 恢复到打开前焦点。
5. inspector 打开期间，若没有 editor 弹层且焦点不在可编辑输入中，Escape 由 `FlowEditorWindow.vue` 关闭 inspector。

#### 接口草案

- `Overlay` 新增可选 props：
  - `trapFocus?: boolean`
  - `initialFocusSelector?: string`
  - `restoreFocus?: boolean`
- editor dialog 面板补充：
  - `role="dialog"`
  - `aria-modal="true"`
  - `aria-labelledby`
  - 必要时的 `aria-describedby`

#### 错误与安全

- 焦点恢复必须先确认原元素仍在文档中且可聚焦，否则静默跳过。
- 若 overlay 内找不到可聚焦元素，可回退焦点到 overlay 容器。
- 不改变 overlay 栈已有的 Escape 规则，只在 Tab 循环上增强。

#### 性能与测试策略

- focus trap 只在弹层打开时响应键盘事件，不引入轮询。
- 表单标签增强仅增加 DOM 属性，不增加 store 或 I/O 开销。
- 自动化测试优先覆盖 overlay 焦点管理的关键路径；其余仍以 `npm run build` 和键盘冒烟为主。

#### 可扩展性设计点

- 让 `Overlay` 的焦点管理以可选 prop 形式存在，避免全局强制影响其他页面。
- 把动态字段的 `aria-labelledby/aria-describedby` 生成留在 inspector helper 函数中，方便后续 visual form 扩展继续复用。

#### Issue List

- 无

阻塞：否
进入 3.1

### Stage 3.1 - Planning (`FLOW-ENH-4`)

#### Project Goal and Current State

- `FLOW-ENH-1/2/3` 已分别完成壳层拆分、草稿可靠性和 visual form UX，但 editor accessibility 仍缺少稳定 requirements/specs 和统一实现。
- 当前 `Overlay` 只有 backdrop / Escape 关闭，没有焦点进入、恢复和 Tab 环路能力。

#### Docs Governance Routing Decision

- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- 本轮新增稳定文档：
  - `docs/requirements/flow-editor-accessibility.md`
  - `docs/specs/flow-editor-accessibility.md`
- Canonical destination:
  - 稳定无障碍要求与约束 -> 新增 accessibility requirements/specs
  - 本轮实施结果 -> `docs/change/YYYY-MM-DD_topic.md`
  - lessons：当前暂未发现必须独立沉淀的排障知识
- Requirements impact: `add`
- Specs impact: `add`
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\flow-editor-accessibility.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\flow-editor-visual-form.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\flow-editor-accessibility.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\flow-editor-visual-form.md`
- Related lessons:
  - `none`

#### Executable Checklist

- [x] `FLOW-ENH-4-A` 为 editor overlay 增加可选焦点管理能力
- [x] `FLOW-ENH-4-B` 为 editor dialogs 补齐 dialog 语义、标签关联和初始焦点
- [x] `FLOW-ENH-4-C` 为 inspector 补齐区域语义、动态字段标签和 Escape 关闭路径
- [x] `FLOW-ENH-4-D` 补最小自动化测试或等效验证覆盖

#### Task Details

##### `FLOW-ENH-4-A` - Overlay 焦点管理

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\plan.md`
- Files / Modules:
  - `frontend/src/components/ui/overlay/Overlay.vue`
  - 视测试方案而定：测试文件
- Write Set:
  - overlay 组件与其测试文件
- Acceptance:
  - 支持可选 focus trap、初始焦点和关闭后焦点恢复
- Tests:
  - `cd frontend && npm run build`
  - 若补组件测试，则执行新增测试命令
- Rollback:
  - 回退 overlay accessibility props 和对应测试

##### `FLOW-ENH-4-B` - Editor dialogs 语义与标签

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\plan.md`
- Files / Modules:
  - `frontend/src/components/flow/editor/FlowAddNodeDialog.vue`
  - `frontend/src/components/flow/editor/FlowMethodPickerDialog.vue`
  - `frontend/src/components/flow/editor/FlowFieldBindingDialog.vue`
  - `frontend/src/i18n/messages/automation.ts`
- Write Set:
  - 仅限上述 editor dialog 相关文件
- Acceptance:
  - 关键表单控件具备标签
  - dialog 具备语义和初始焦点声明
- Tests:
  - `cd frontend && npm run build`
- Rollback:
  - 回退 dialog 语义和标签增强

##### `FLOW-ENH-4-C` - Inspector 无障碍与键盘退出

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\plan.md`
- Files / Modules:
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/i18n/messages/automation.ts`
- Write Set:
  - inspector / editor 壳层与对应文案
- Acceptance:
  - inspector 具备区域标题语义
  - 动态 visual form 字段有可识别标签关联
  - Escape 可关闭 inspector
- Tests:
  - `cd frontend && npm run build`
- Rollback:
  - 回退 inspector 与 Escape 交互增强

##### `FLOW-ENH-4-D` - 最小验证覆盖

- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\plan.md`
- Files / Modules:
  - 测试配置或最小测试文件
- Write Set:
  - 仅限本轮所需最小测试文件
- Acceptance:
  - 能覆盖至少一个焦点管理关键路径或给出明确无法自动化覆盖的原因
- Tests:
  - 执行本轮新增测试命令
- Rollback:
  - 回退本轮新增测试文件

#### Dependencies

- 依赖已完成的 editor 组件边界；不需要修改 Flow 协议或 visual form 数据模型。
- 本轮新增 docs 已作为稳定真相补齐，可以进入实现。

#### Risks and Notes

- 如果 overlay 焦点管理做成默认全局开启，可能影响 `frontend/src/pages/Flow.vue` 等未适配页面。
- 当前测试栈仍偏轻量；若组件焦点测试成本过高，需要在 review 中明确记录覆盖缺口。

#### Parallelism Assessment

- 本轮不派发子Agent。
- 原因：
  - overlay、dialogs、inspector 和 editor 壳层存在同一条焦点 / 键盘交互关键路径。
  - 写集交叉明显，并行会放大集成和回归成本。

#### Issue List

- 无

阻塞：否
进入 3.2
