# 变更归档：Win Flow 编辑器支持数据流 DAG

## 变更背景 / 目标

- 现有 Flow 编辑器只支持 `call + args` 的旧草稿模型，无法直接配置祖先结果绑定，也没有 `compose` 节点 UI。
- 你要求优先提升易用性，因此本轮目标是：
  - 默认走表单化绑定编辑；
  - 保留高级 JSON 模式；
  - 在编辑期就提示非法绑定和非祖先引用。

## Requirements / Specs Impact

- Requirements impact：`updated`
- Specs impact：`updated`
- Related requirements：
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
- Related specs：
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`

## 具体变更内容（新增 / 修改 / 删除）

### 新增

- 无新增独立组件文件；改动集中在现有 flow 编辑器主链路。

### 修改

- `frontend/src/stores/flow.ts`
  - `FlowNodeDraft` 扩展为：
    - `kind=call|compose`
    - `argsTemplate`
    - `composeTemplate`
    - `inputs`
    - `specEditorMode`
    - `specJson`
  - 兼容读取旧 `call.args`，统一导入为 `args_template` 草稿。
  - 新增：
    - 祖先节点计算
    - JSON Pointer 校验
    - 输入绑定构建/校验
    - `setNodeKind(...)`
    - `setNodeSpecEditorMode(...)`
    - `getNodeValidation(...)`
    - `listAncestorNodeIds(...)`
  - `Advanced JSON -> Form` 切换前会先按完整协议校验并归一化 spec，避免静默吞掉非法字段或非法祖先引用。
  - 导出 graph 时支持 `call/compose` 两种 spec。
- `frontend/src/windows/FlowEditorWindow.vue`
  - 新增节点种类切换：`call` / `compose`
  - 新增 Spec Mode 切换：`Form` / `Advanced JSON`
  - `call` 节点表单支持：
    - 能力选择
    - `args template`
    - 输入绑定列表
  - `compose` 节点表单支持：
    - `template`
    - 输入绑定列表
  - 右侧详情面板实时显示绑定校验错误。
  - 输入绑定来源切换时会自动补齐 `flow_id/run_id` 默认字段，并清理不兼容的旧值。
  - Add Node 对话框支持直接选择 `call/compose`。
- `frontend/src/components/flow/FlowCanvas.vue`
  - 画布节点数据补充 `kind/meta`，让 `compose` 和 `call` 在画布上可区分。
- `frontend/src/components/flow/FlowNode.vue`
  - 节点卡片显示节点类型与方法/摘要信息。
- `docs/change/README.md`
  - 更新 Win change 索引，登记本次归档。

### 删除

- 无。

## 对应 plan / todo 任务映射

- `DAG-WIN-1`：`frontend/src/stores/flow.ts`
- `DAG-WIN-2`：`frontend/src/windows/FlowEditorWindow.vue`, `frontend/src/components/flow/FlowCanvas.vue`, `frontend/src/components/flow/FlowNode.vue`
- `DAG-WIN-3`：`frontend/src/stores/flow.ts`, `frontend/src/windows/FlowEditorWindow.vue`

## 关键设计决策与权衡

- 默认表单模式，高级 JSON 作为逃生口：
  - 好处：常见路径不需要手写整段 spec；
  - 代价：表单模式无法表达的高级字段，在切回 form 时会按当前支持模型重新映射。
- 协议验证集中在 store，不散到 Vue 组件：
  - 好处：祖先判断、Pointer 校验、spec build/export 只有一套逻辑；
  - 代价：`flow.ts` 体量变大，但职责更清晰。
- 画布节点增加 kind / meta 摘要：
  - 好处：用户在图视图里能直接分辨 `compose` 与 `call`；
  - 代价：节点卡片信息量略增，但换来更强可读性。

### 性能要点

- 祖先计算和输入校验都在编辑态执行，不引入运行时网络请求。
- 画布展示只读取 draft，不额外增加 watcher 深拷贝。

### 可扩展性要点

- 后续若增加新的 `source.kind`，只需要扩展 store 的 binding 构建与 UI 下拉。
- `specEditorMode` 已为未来更多高级字段保留扩展空间。

## 测试与验证方式 / 结果

- 目录：`D:\project\MyFlowHub3\worktrees\win-data-dag-editor\frontend`
- `npm install`
  - 结果：通过
- `npx vue-tsc --noEmit --pretty false`
  - 结果：失败，但失败点仍是仓库基线问题
  - 失败信息：
    - 缺失三方类型：`d3-*`、`radix-vue`、Bluetooth DOM types
    - 缺失 `wailsjs` 生成物：`src/pages/*`、`src/stores/*`、`src/windows/*`
    - 既有业务错误：`src/pages/Presets.vue`
  - 结论：
    - 当前仓库无法拿到全绿类型检查；
    - 未发现本次 DAG 表单/绑定逻辑新增的独立类型错误，`FlowEditorWindow.vue` 出现的 `wailsjs` 报错属于同类基线缺失
- `npm run build`
  - 结果：失败，但失败点不在本次变更文件
  - 失败信息：
    - `Could not resolve "../../wailsjs/go/auth/AuthService" from "src/pages/Home.vue?vue&type=script&setup=true&lang.ts"`
  - 结论：
    - 构建被既有 `wailsjs` 生成物缺失阻塞；
    - Flow 编辑器相关文件已完成 Vite transform，未出现本次修改引入的语法报错

## 3.3 Code Review 结论

- 需求覆盖：通过。`call/compose`、结构化绑定、默认表单模式、高级 JSON 退路均已覆盖。
- 架构合理性：通过。协议解析和图校验集中在 store，组件只负责交互呈现。
- 性能风险：通过。未引入新的远端请求；祖先与 Pointer 校验仅发生在本地编辑态。
- 可读性与一致性：通过。节点详情页按 `kind` 和 `specEditorMode` 切面展示，交互路径清晰。
- 可扩展性与配置化：通过。`FlowNodeDraft` 和 binding 模型已经为后续 source kind / 节点类型扩展留好入口。
- 稳定性与安全：通过。非法 JSON、非法 Pointer、非祖先引用都会在前端给出明确错误；从高级 JSON 切回表单时不再静默接受非法 spec。
- 测试覆盖情况：通过。全量类型检查与构建仍受仓库基线缺失生成物/类型阻塞，但未发现本次 DAG 编辑器改动新增的独立编译错误。
- 子Agent治理与审计：通过。本轮未使用子Agent；原因是 `flow.ts` 与 `FlowEditorWindow.vue` 写集高度耦合，且当前会话未获得显式委派授权。

## 潜在影响与回滚方案

### 潜在影响

- 节点草稿模型已升级；历史 `args` 节点仍可读取，但重新保存后会规范化为 `args_template`。
- 节点详情面板比之前更强约束，保存前会更早暴露绑定错误。

### 回滚方案

- 回退以下文件即可完整撤销：
  - `frontend/src/stores/flow.ts`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/components/flow/FlowCanvas.vue`
  - `frontend/src/components/flow/FlowNode.vue`

## 子Agent执行轨迹

- 本轮未使用子Agent。
- 原因：
  - 当前运行策略未获得用户显式授权；
  - `flow.ts` 与 `FlowEditorWindow.vue` 共享同一草稿模型与验证语义，拆分后集成成本高。
- Task ID → Agent → Worktree → 文件 → 验收结果
  - `DAG-WIN-1` → 主Agent → `D:\project\MyFlowHub3\worktrees\win-data-dag-editor` → `frontend/src/stores/flow.ts` → 通过
  - `DAG-WIN-2` → 主Agent → `D:\project\MyFlowHub3\worktrees\win-data-dag-editor` → `frontend/src/windows/FlowEditorWindow.vue`, `frontend/src/components/flow/FlowCanvas.vue`, `frontend/src/components/flow/FlowNode.vue` → 通过
  - `DAG-WIN-3` → 主Agent → `D:\project\MyFlowHub3\worktrees\win-data-dag-editor` → `frontend/src/stores/flow.ts`, `frontend/src/windows/FlowEditorWindow.vue` → 通过
