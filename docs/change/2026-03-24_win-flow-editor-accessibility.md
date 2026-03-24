# 2026-03-24 Win Flow Editor Accessibility

## Background

- `FLOW-ENH-4` 负责把 Win Flow 编辑器从“功能可用”推进到“基础无障碍和键盘路径稳定可用”。
- 在 `FLOW-ENH-1/2/3` 完成壳层拆分、草稿可靠性和 visual form UX 后，editor 还缺少稳定的标签关联、dialog 焦点管理和 inspector 键盘退出约束。

## Goal

- 为 editor 弹层建立统一的焦点进入、Tab 环路和焦点恢复能力。
- 为关键表单和 inspector 建立最小但稳定的无障碍语义。

## Changes

- 新增稳定 docs：
  - `docs/requirements/flow-editor-accessibility.md`
  - `docs/specs/flow-editor-accessibility.md`
- `Overlay.vue`
  - 新增可选 `trapFocus`、`initialFocusSelector`、`restoreFocus`。
  - 打开时记录先前焦点并移入当前 overlay。
  - Tab / Shift+Tab 在当前 overlay 内循环。
  - 关闭或卸载时恢复到先前焦点。
- editor dialogs
  - `FlowAddNodeDialog.vue`
  - `FlowMethodPickerDialog.vue`
  - `FlowFieldBindingDialog.vue`
  - 以上均补齐 dialog 语义、`aria-labelledby` / `aria-describedby`、关键表单标签和初始焦点声明。
- `FlowNodeInspector.vue`
  - 补齐区域语义。
  - 为动态 visual form 字段和 compose binding 输入生成稳定的 `id` / `aria-*` 关联。
  - 补充 validation 区域的状态语义。
- `FlowEditorWindow.vue`
  - 在未进入输入态且无弹层时，Escape 可关闭 inspector。
- 自动化测试
  - 新增 `Overlay.test.ts`，覆盖焦点进入、Tab 环路和关闭后焦点恢复。

## Related Plan

- `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\plan.md`
- Task mapping:
  - `FLOW-ENH-4-A` Overlay 焦点管理
  - `FLOW-ENH-4-B` Editor dialogs 语义与标签
  - `FLOW-ENH-4-C` Inspector 无障碍与键盘退出
  - `FLOW-ENH-4-D` 最小验证覆盖

## Related Requirements

- `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\flow-editor-accessibility.md`
- `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\flow-editor-visual-form.md`

## Related Specs

- `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\flow-editor-accessibility.md`
- `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\flow-editor-visual-form.md`

## Lessons Impact

- `none`

## Related Lessons

- `none`

## Searchable Lessons Summary

- Symptoms:
  - 打开 editor 弹层后焦点停留在背景区域
  - Tab 会跳出当前 dialog
  - inspector 表单字段难以被辅助技术识别
  - 无法用 Escape 快速退出 inspector
- Trigger Conditions:
  - 使用键盘或屏幕阅读器操作 `Add Node`、`Select Capability`、`Bind Field`
  - ordinary mode 渲染动态字段和 compose binding 列表
- Keywords:
  - `trapFocus`
  - `initialFocusSelector`
  - `restoreFocus`
  - `aria-labelledby`
  - `Escape close inspector`
- Quick Checks:
  - 打开 dialog 后看焦点是否落在首个预期输入
  - 用 Tab / Shift+Tab 检查焦点是否仍在 dialog 内
  - 关闭 dialog 后看焦点是否回到触发按钮
  - inspector 打开时按 Escape 是否关闭

## Requirements Impact

- `updated`

## Specs Impact

- `updated`

## Design Decisions and Tradeoffs

- 焦点管理集中到 `Overlay.vue`，且以可选 prop 暴露：
  - 好处是 editor dialogs 共用一套实现，不必各自维护焦点逻辑。
  - 代价是需要给具体 dialog 显式声明启用，不能自动覆盖所有旧页面。
- 动态字段标签采用 `aria-labelledby/aria-describedby`，而不是强行把每个字段都改造成单独 `label` 包裹：
  - 好处是更适合 visual form 动态列表和 compose binding 列表。
  - 代价是模板中会多一层 id helper。
- 这轮只补 editor 和 overlay 的基础语义，不扩张到 FlowCanvas 完整键盘模型：
  - 好处是范围可控。
  - 代价是画布级键盘可编辑性仍待后续独立工作。

## Validation

- `$env:GOWORK='off'; wails generate module`：通过
  - 仍打印 `Not found: time.Time`，但命令最终成功。
- `cd frontend && npm test`：通过
  - `flow_visual_form.test.ts` 3 个用例通过。
  - `Overlay.test.ts` 3 个用例通过。
- `cd frontend && npm run build`：通过
  - 仍保留单 chunk 约 `969.91 kB` 的既有告警。

## Potential Impact and Rollback

- 潜在影响：
  - overlay 焦点管理会影响显式启用 `trapFocus` 的 editor dialogs。
  - inspector 和 dialog 的 DOM 属性、焦点和键盘路径发生变化，但不影响 Flow 保存协议。
- Rollback:
  - 回退 `frontend/src/components/ui/overlay/Overlay.vue`
  - 回退 `frontend/src/components/CardHeader.vue`
  - 回退 `frontend/src/components/flow/editor/FlowAddNodeDialog.vue`
  - 回退 `frontend/src/components/flow/editor/FlowMethodPickerDialog.vue`
  - 回退 `frontend/src/components/flow/editor/FlowFieldBindingDialog.vue`
  - 回退 `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - 回退 `frontend/src/windows/FlowEditorWindow.vue`
  - 回退 `frontend/src/i18n/messages/automation.ts`
  - 如需一并撤销测试，再回退 `frontend/src/components/ui/overlay/Overlay.test.ts` 及新增测试依赖

## SubAgent Trace

- `none`
