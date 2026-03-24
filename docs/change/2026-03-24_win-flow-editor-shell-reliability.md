# 变更归档：Win Flow Editor Shell Reliability

## 变更背景 / 目标

- 经过上一轮评估后，本轮明确先做两件事：
  - `FLOW-ENH-1`：收敛 editor shell 结构，停止继续把主要交互堆进 `FlowEditorWindow.vue`
  - `FLOW-ENH-2`：补齐草稿可靠性，降低误关窗口、刷新或崩溃时的丢稿风险
- 本轮不扩 capability / visual form 能力面，目标是先把窗口壳层和草稿边界打稳。

## 具体变更内容

### `FLOW-ENH-1` - 结构收敛

- `frontend/src/windows/FlowEditorWindow.vue`
  - 收敛为窗口壳层，保留：
    - 生命周期
    - project load/save
    - 顶层快捷键
    - dirty-state / recovery 协调
    - 顶层事件装配
- 新增 `frontend/src/components/flow/editor/*`
  - `FlowEditorToolbar.vue`
  - `FlowNodeInspector.vue`
  - `FlowMethodPickerDialog.vue`
  - `FlowFieldBindingDialog.vue`
  - `FlowAddNodeDialog.vue`
- 子组件只负责 UI 和事件上抛，不承担 project save 或 recovery 逻辑。

### `FLOW-ENH-2` - 草稿可靠性

- `frontend/src/stores/flow.ts`
  - 新增 `FlowGraphEditorState`
  - 新增 `exportGraphEditorState()`
  - 新增 `loadGraphEditorState(...)`
  - 新增 `graphEditorSignature()`
  - signature 只基于 graph 内容，不依赖 history / selection 变更作为保存基线
- `frontend/src/windows/FlowEditorWindow.vue`
  - 增加保存状态展示：
    - `Saved / Unsaved changes`
    - `Last saved {time}`
  - 增加 `beforeunload` 保护
  - 增加按 `projectId` 隔离的本地 recovery 草稿
  - 恢复逻辑基于原始 editor state，而不是仅基于可导出的有效 graph
  - 仅当 recovery 记录的 `baseSignature` 与当前已保存 project 基线一致时才提示恢复
  - 手动保存成功后更新 saved baseline 并清理 recovery 记录

### 文档治理

- 新增稳定 requirements：
  - `docs/requirements/flow-editor-draft-reliability.md`
- 新增稳定 specs：
  - `docs/specs/flow-editor-draft-reliability.md`
- 更新索引：
  - `docs/requirements/README.md`
  - `docs/specs/README.md`

## Requirements / Specs / Lessons Impact

- Requirements impact: `updated`
- Specs impact: `updated`
- Lessons impact: `none`
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\flow-editor-draft-reliability.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\flow-editor-visual-form.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\flow-editor-draft-reliability.md`
  - `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\flow-editor-visual-form.md`
- Related lessons:
  - `none`

## 对应 plan 任务映射

- `FLOW-ENH-1`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/components/flow/editor/*`
- `FLOW-ENH-2`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/i18n/messages/automation.ts`

## 经验 / 教训摘要

- 对 Win Flow 编辑器来说，“结构收敛”和“草稿可靠性”必须先于 capability 扩展；否则每次加功能都会继续放大单体窗口和丢稿风险。
- 本地恢复不能复用 `exportGraphDraft()` 的保存导出，因为中间编辑态可能本来就暂时不合法。
- dirty-state 的签名必须基于 graph 内容，而不是 history 或选择状态，否则保存状态会抖动。

## 可复用排查线索

- 症状：
  - 刷新或误关 editor 后，未保存 graph 丢失
  - `FlowEditorWindow.vue` 再次膨胀为大体量热点文件
  - 恢复草稿提示反复出现或恢复到错误 project
- 触发条件：
  - recovery 记录没有绑定 `projectId`
  - recovery 记录没有记录 saved baseline
  - save success 后没有清理 recovery
- 关键词：
  - `FlowGraphEditorState`
  - `graphEditorSignature`
  - `beforeunload`
  - `myflowhub.flow-editor.recovery`
  - `FlowEditorToolbar`
- 快速检查：
  - 看 `flow.ts` 是否仍暴露 editor state 导入导出 API
  - 看 `FlowEditorWindow.vue` 是否只保留窗口壳层职责
  - 看 recovery 是否校验 `baseSignature`

## 关键设计决策与权衡

- 选择“本地 recovery + 手动保存”，而不是立即引入 autosave：
  - 好处：不改变当前项目保存模型，也不引入额外后端 I/O
  - 代价：用户仍需显式保存才能持久化到 project store
- 把 recovery snapshot 建立在原始 editor state 上：
  - 好处：即使 graph 处于中间不合法状态也可恢复
  - 代价：恢复后仍可能处于 dirty / invalid 状态，需要用户继续修正
- 通过子组件拆壳，而不直接重写 editor：
  - 好处：保持现有交互和 store 语义不变，回滚面清晰
  - 代价：`flow.ts` 仍然偏大，后续还要继续做测试和体验增强

## 测试与验证方式 / 结果

- `$env:GOWORK='off'; wails generate module`
  - 结果：通过
- `cd frontend && npm run build`
  - 结果：通过
- 结果备注：
  - `dist/assets/index-*.js` 仍约 `955 kB`，chunk 过大告警保持不变
  - 本回合未执行 `wails dev` 手工 UI 冒烟

## 潜在影响与回滚方案

### 潜在影响

- editor window 现在会对未保存 graph 改动触发浏览器关闭提示。
- 本地会为每个 `projectId` 写入一份 recovery 记录。
- `FlowEditorWindow.vue` 的模板结构已迁移到多个子组件，后续改 inspector / dialog 需在对应子组件内处理。

### 回滚方案

- 回退以下文件即可完整撤销本轮行为：
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/components/flow/editor/*`
  - `frontend/src/i18n/messages/automation.ts`
  - `docs/requirements/flow-editor-draft-reliability.md`
  - `docs/specs/flow-editor-draft-reliability.md`
  - 对应 `README.md` 索引与本 change 文档

## 子Agent执行轨迹

- 本轮未使用子Agent。
- 原因：
  - `FLOW-ENH-1` 与 `FLOW-ENH-2` 共用 `FlowEditorWindow.vue` / `flow.ts` 的高耦合写集
  - 主Agent 需要统一处理壳层拆分、状态基线与 recovery 语义
