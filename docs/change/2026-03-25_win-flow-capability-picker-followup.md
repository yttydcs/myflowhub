# 2026-03-25 Win Flow Capability Picker Follow-up

## 变更背景 / 目标

- Flow 编辑器的 capability picker 和 ordinary mode unavailable 诊断已经上线，但当前交互仍有三个明显 UX 缺口：
  - `Query Node ID` 与 `Filter` 输入一高一低；
  - 已经选过方法后再次打开方法对话框，筛选框会自动回填当前 method；
  - 方法缺少 visual form schema 时，inspector 仍直接展示“当前方法没有可用的 visual form schema”提示。
- 本轮目标是在不修改 Flow / Exec 协议和 store 契约的前提下，只收敛这三个前端体验问题，并补最小自动化回归。

## 具体变更内容

- `frontend/src/components/flow/editor/FlowMethodPickerDialog.vue`
  - 将顶部输入区从 `items-end` 调整为 `items-start`，并让刷新按钮单独贴底对齐，修正 `Query Node ID` 与 `Filter` 输入错位。
- `frontend/src/windows/FlowEditorWindow.vue`
  - `openMethodDialog()` 不再把 `selectedNode.method` 写入 `methodSearch`。
  - `closeMethodDialog()` 统一清空本地筛选词，避免 dialog 临时态继续被节点 method 污染。
  - 选中节点切换、apply 完成、节点详情关闭或节点类型切换时，沿用统一关闭逻辑，保持筛选状态生命周期清晰。
- `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - 新增组件层过滤，只隐藏 `missing_schema` 这类不友好的 unavailable reason。
  - 其他 compatibility reason 和 `Open Advanced JSON` 入口保持不变。
- 测试
  - 新增 `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - 新增 `frontend/src/windows/FlowEditorWindow.test.ts`
  - 新增 `frontend/src/test/wails_main_app.stub.ts`
  - 更新 `frontend/vitest.config.ts`，仅为测试环境补 `../../wailsjs/go/main/App` stub alias，避免缺失的 Wails 生成物阻塞组件级回归。

## Requirements impact

- `none`

## Specs impact

- `none`

## Lessons impact

- `none`

## Related requirements

- `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux\docs\requirements\flow-editor-visual-form.md`
- `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux\docs\requirements\flow-editor-accessibility.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux\docs\specs\flow-editor-visual-form.md`
- `D:\project\MyFlowHub3\worktrees\win-capability-picker-ux\docs\specs\flow-editor-accessibility.md`

## Related lessons

- `none`

## 对应 plan.md 任务映射

- `CAPUX-1`
  - `frontend/src/components/flow/editor/FlowMethodPickerDialog.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
- `CAPUX-2`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
- `CAPUX-3`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - `frontend/src/windows/FlowEditorWindow.test.ts`
  - `frontend/src/test/wails_main_app.stub.ts`
  - `frontend/vitest.config.ts`

## 经验 / 教训摘要

- picker 的筛选词属于 dialog 临时态，不应从节点 method 反向推导；否则每次 reopen 都会覆盖用户自己的搜索意图。
- visual form compatibility reason 可以继续保留在 helper / store 中做结构化判定，但不等于每条 reason 都必须原样暴露给最终用户。
- 针对依赖 Wails 生成物的窗口组件，组件测试需要单独准备测试期 stub，避免把验证链绑定到外部生成物是否齐全。

## 可复用排查线索

- Symptoms:
  - capability picker 顶部两个输入没有对齐
  - 重新选方法时筛选框自动出现当前 method
  - inspector 出现 “The current method does not provide a supported visual form schema.”
- Trigger Conditions:
  - `call` 节点已有 method，用户重新打开 `Select Capability`
  - query node 输入区带 help text，而相邻 filter 输入区没有
  - 当前方法没有可解析 visual form schema
- Keywords:
  - `methodSearch`
  - `missing_schema`
  - `FlowMethodPickerDialog`
  - `FlowNodeInspector`
- Quick Checks:
  - 打开方法对话框，确认 filter 初始值是否为空
  - 检查顶部输入容器是否仍使用 `items-end`
  - 检查 unavailable reason 列表是否仍渲染 `missing_schema`

## 关键设计决策与权衡

- 决策：本轮只在组件层隐藏 `missing_schema`，不改 store 的 compatibility reason 结构。
  - 原因：用户抱怨的是直接暴露给 UI 的提示，不是底层判定本身；保留 reason 结构有利于后续调试和更细粒度 UX 调整。
- 决策：method picker 每次关闭时清空筛选词，而不是继续从节点 method 派生默认值。
  - 原因：这让筛选行为回到“用户主动输入”模型，避免 reopen 时被动回填。
- 决策：为 Vitest 单独加 `wailsjs` stub alias，而不是修改生产构建入口。
  - 原因：只解决测试期缺失生成物的问题，不改变运行时代码的真实依赖路径。

## 测试与验证方式 / 结果

- `cd frontend && npm ci`
  - 结果：通过
- `cd frontend && npm test`
  - 结果：通过
  - 说明：`6` 个测试文件、`15` 个用例全部通过；新增覆盖 method picker filter 初始化和 inspector `missing_schema` 展示收敛。
- `cd frontend && npm run build`
  - 结果：失败
  - 失败点：仓库现有 `wailsjs` 生成物缺失，报错为 `Could not resolve "../../wailsjs/go/session/SessionService" from "src/pages/Home.vue"`
  - 结论：失败点不在本轮改动文件，本轮新增测试和组件修改未引入新的构建错误定位。
- chrome-devtools 冒烟
  - 结果：未执行
  - 原因：当前 worktree 缺少完整 Wails 生成物，页面运行基线不完整，先以组件测试作为回归主证据。

## 3.3 Code Review 结论

- 需求覆盖：通过
  - 三个用户报告的问题都已对应到代码与测试：输入对齐、methodSearch 自动回填移除、`missing_schema` 提示收敛。
- 架构合理性：通过
  - 变更限制在窗口层与组件层，没有扩散到 store 协议或后端接口。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 仅调整局部布局与渲染条件，不增加网络请求和额外数据扫描。
- 可读性与一致性：通过
  - `closeMethodDialog()` 统一管理 dialog 筛选状态生命周期，inspector 的展示过滤保持单点收敛。
- 可扩展性与配置化：通过
  - compatibility reason 结构仍留在 store/helper，后续 UI 若要恢复或细分展示不需要回退底层判定。
- 稳定性与安全：通过
  - 现有 query node 校验、capability apply 和 `Advanced JSON` 出口都保持不变，没有引入 silent failure。
- 测试覆盖情况：通过
  - 新增两个组件级回归测试；全量 `npm test` 通过。
- 子Agent治理与审计：通过
  - 本轮未使用子Agent；原因是当前写集集中在同一窗口/组件语义，且用户未显式要求派发。

## 潜在影响

- 方法选择对话框重新打开时，筛选框不再保留“当前 method”这个自动默认值；用户需要主动输入搜索词。
- inspector 对 `missing_schema` 的说明减少后，当前场景会更偏向直接引导用户走 `Advanced JSON`。

## 回滚方案

- 回退以下文件即可撤销本轮改动：
  - `frontend/src/components/flow/editor/FlowMethodPickerDialog.vue`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/components/flow/editor/FlowNodeInspector.vue`
  - `frontend/src/components/flow/editor/FlowNodeInspector.test.ts`
  - `frontend/src/windows/FlowEditorWindow.test.ts`
  - `frontend/src/test/wails_main_app.stub.ts`
  - `frontend/vitest.config.ts`

## 子Agent执行轨迹

- `none`
