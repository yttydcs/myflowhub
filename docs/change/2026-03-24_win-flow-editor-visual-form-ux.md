# 2026-03-24 Win Flow Editor Visual Form UX

## Background

- `FLOW-ENH-3` 的目标是在不改动 Flow / Exec 协议和 visual form 适用范围的前提下，继续提升 Win Flow 编辑器的 capability 选择、字段 binding 和 ordinary mode 诊断体验。
- 前两轮已经完成 editor 壳层收敛与草稿可靠性，本轮可以安全地把工作收束在体验层和纯逻辑 helper。

## Goal

- 让 capability picker 更容易扫描和筛选。
- 让字段 binding 编辑减少对裸 JSON Pointer 和协议细节的暴露。
- 让 ordinary mode unavailable 的原因可结构化展示和复用。

## Changes

- capability picker：
  - 扩展本地筛选维度，支持方法、节点、版本、permission、tag 和已有 label 的匹配。
  - 列表行补充 `input_schema`、默认超时、permission / tag 预览，帮助快速判断 capability。
- visual form helper：
  - `flow_visual_form.ts` 新增结构化 compatibility reason 类型。
  - binding summary 改为更偏用户可读的摘要，并通过 i18n 输出。
  - compatibility reason 新增统一描述 helper，避免组件层直接依赖英文裸字符串。
- inspector / binding dialog：
  - field binding dialog 新增目标字段说明、来源预览、路径 helper 和 `node_result` apply gating。
  - inspector 中的字段 pointer 改为“写入到某路径”的辅助说明。
  - ordinary mode unavailable 改为分类 badge + 原因说明 + 修复提示的结构化展示。
- 最小测试：
  - 引入 `vitest`，新增 `flow_visual_form.test.ts`，覆盖 binding 摘要和 compatibility reason 核心分支。

## Related Plan

- `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\plan.md`
- Task mapping:
  - `FLOW-ENH-3-A` capability picker 信息密度与检索增强
  - `FLOW-ENH-3-B` binding / compatibility helper 收敛
  - `FLOW-ENH-3-C` inspector / field binding dialog 体验增强
  - `FLOW-ENH-3-D` 最小纯逻辑测试

## Related Requirements

- `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\requirements\flow-editor-visual-form.md`

## Related Specs

- `D:\project\MyFlowHub3\worktrees\feat-win-dag-editor-enhancement-review\docs\specs\flow-editor-visual-form.md`

## Lessons Impact

- `none`

## Related Lessons

- `none`

## Searchable Lessons Summary

- Symptoms:
  - capability picker 难以从现有 metadata 中快速定位目标方法
  - field binding dialog 暴露裸 JSON Pointer，缺少来源预览
  - ordinary mode unavailable 只显示原始 reason 字符串，诊断性差
- Trigger Conditions:
  - `call` 节点 method 较多、capability route 带 permissions / tags / input_schema
  - visual form compatibility 因 binding 越界、extra literal field 或非法 args template 失败
- Keywords:
  - `Visual form unavailable`
  - `binding target`
  - `Args template contains a field`
  - `Search method / node / permission / tag`
- Quick Checks:
  - 看 capability picker 是否显示 schema / timeout / metadata badge
  - 看 binding dialog 是否出现目标字段说明和来源预览
  - 看 ordinary mode unavailable 是否按 Method / Schema / Bindings / Template 分类展示

## Requirements Impact

- `none`

## Specs Impact

- `none`

## Design Decisions and Tradeoffs

- compatibility reason 结构化放在 `flow_visual_form.ts`，而不是写死在组件内：
  - 好处是 UI、测试、后续无障碍都能复用同一判定出口。
  - 代价是 helper 类型略有增加，但没有改变持久化模型。
- capability picker 只消费现有 `ExecCapabilityRoute`：
  - 好处是无需触碰后端协议。
  - 代价是当前只展示轻量 metadata 预览，不引入更重的分组或远程二次查询。
- 测试只补最小 `vitest` 纯逻辑覆盖：
  - 好处是能锁住本轮新增 helper 行为，同时不把工作范围扩张到完整 editor 回归。
  - 代价是 UI 交互级回归仍留待 `FLOW-ENH-5`。

## Validation

- `$env:GOWORK='off'; wails generate module`：通过
  - 命令仍打印 `Not found: time.Time`，但最终成功完成。
- `cd frontend && npm test`：通过
  - `flow_visual_form.test.ts` 共 3 个用例通过。
- `cd frontend && npm run build`：通过
  - 保留单 chunk 约 `964 kB` 的现有告警，本轮未处理 chunk 拆分。

## Potential Impact and Rollback

- 潜在影响：
  - capability picker、binding dialog 和 inspector 的文案与展示结构发生变化，主要影响编辑体验，不影响 Flow 保存协议。
  - 新增 `vitest` 仅影响开发依赖和测试脚本。
- Rollback:
  - 回退 `frontend/src/windows/FlowEditorWindow.vue`
  - 回退 `frontend/src/components/flow/editor/FlowMethodPickerDialog.vue`
  - 回退 `frontend/src/components/flow/editor/FlowFieldBindingDialog.vue`
  - 回退 `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - 回退 `frontend/src/stores/flow.ts`
  - 回退 `frontend/src/stores/flow_visual_form.ts`
  - 回退 `frontend/src/i18n/messages/automation.ts`
  - 如需一并撤销测试，再回退 `frontend/package.json`、`frontend/package-lock.json`、`frontend/vitest.config.ts`、`frontend/src/stores/flow_visual_form.test.ts`

## SubAgent Trace

- `none`
