# 2026-03-21_flow-editor-method-selector-dialog

## 变更背景 / 目标
- 背景：
  - Flow 编辑器右侧节点详情抽屉存在透明感，影响阅读与聚焦。
  - `call` 节点虽然已统一成 call-only 模型，但编辑器里缺少独立的方法选择器，用户只能直接手填 `method/target`，配置成本高且容易错配。
- 目标：
  - 将节点详情抽屉收敛为不透明展示。
  - 为 `call` 节点补充独立的方法选择对话框。
  - 隐藏主表单里的 `target` 手工编辑入口，但保留 `target` 在数据层中的运行时语义。

## 具体变更内容（新增 / 修改 / 删除）

### 修改
- `frontend/src/windows/FlowEditorWindow.vue`
  - 节点详情抽屉遮罩改深、面板改为不透明 `bg-card`
  - 移除主表单中的 `Target Node / Method` 手工编辑区
  - 新增 `Select Capability` 对话框：
    - 支持能力刷新
    - 支持本地过滤
    - 展示 provider / via / version
    - 选择后应用到当前 `call` 节点
  - 对话框打开期间屏蔽底层全局删除/撤销快捷键，避免误操作

- `frontend/src/stores/flow.ts`
  - 新增 `normalizeCallTarget(...)`
  - `applyCallCapability(...)` 调整为：
    - provider 等于当前 executor 时，写回 `target=0`
    - 远端 provider 保留对应 nodeId
    - 若 `args` 为空则补 `{}`，否则保持原值
  - 导出 `normalizeCallTarget(...)`，便于后续扩展复用

### 新增
- `docs/change/2026-03-21_flow-editor-method-selector-dialog.md`

### 删除
- 删除编辑器抽屉中对 `target` 的手工输入入口

## 对应 `plan.md` 任务映射
- `FLOW-METHOD-1`
  - `frontend/src/windows/FlowEditorWindow.vue`
- `FLOW-METHOD-2`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/stores/flow.ts`
- `FLOW-METHOD-3`
  - 验证记录
  - Code Review
  - 本变更文档

## 关键设计决策与权衡（尤其性能 / 扩展性）
- 决策：`target` 从 UI 主表单隐藏，但不从数据模型删除
  - 原因：当前协议中 `target` 仍有运行时语义，不能当成纯编辑辅助字段删除
  - 权衡：用户界面更简洁；未来仍能支持远程 provider 调用

- 决策：能力选择使用独立对话框，而不是继续暴露 method 手工输入
  - 原因：主路径应以“查能力并应用”为主，避免 `method` 与 `target` 脱节
  - 权衡：少量高级手工兜底能力被弱化，但整体易用性更高

- 决策：本次只自动回填 `method + target`，不生成 `args` 模板
  - 原因：当前已确认的 `cap_query_resp.routes[]` 不包含可稳定驱动前端自动回填的 args schema/template
  - 权衡：先保证方法选择链路正确；后续若协议提供 schema，可继续扩展

- 决策：provider 等于 executor 时将 `target` 规范化为 `0`
  - 原因：与现有 “本地调用不写 target” 的 graph 语义对齐，减少冗余 target 写入
  - 权衡：保留旧图兼容；仅在“重新选择能力”时执行规范化

## 测试与验证方式 / 结果
- `frontend/ npm install`
  - 结果：通过

- `frontend/ npx vue-tsc --noEmit --pretty false`
  - 结果：失败
  - 原因：
    - 基线缺失 `wailsjs` 生成物
    - 三方类型缺失：`d3-*`、`radix-vue`、Bluetooth DOM types
    - 其它页面既有错误：`Presets.vue`、`Showcase.vue` 等
  - 结论：错误列表未落到本次修改的 `FlowEditorWindow.vue / flow.ts`

- `frontend/ npm run build`
  - 结果：失败
  - 原因：仓库基线缺失 `../../wailsjs/go/session/SessionService`

- `git diff --check`
  - 结果：通过

- Chrome DevTools + Vite dev server
  - 结果：失败
  - 原因：页面启动后立即被 Vite import overlay 阻塞，缺失 `../../wailsjs/runtime/runtime`（`src/stores/presets.ts`），无法继续进入 Flow 编辑器进行浏览器级冒烟

## 潜在影响与回滚方案

### 潜在影响
- 用户不再能在抽屉主表单里直接手改 `target/method`，主路径变为对话框选择。
- 旧图中若存在“`target == executor` 但仍显式写入”的节点，只有在重新选择能力时才会被规范化为本地调用。
- 由于本次没有 args schema，方法应用后不会自动生成参数模板，仍需手工维护 `args`。

### 回滚方案
- 回退以下文件即可恢复：
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/stores/flow.ts`

## 子Agent执行轨迹（Task ID → Agent → Worktree → 文件 → 验收结果）
- 本次未使用子Agent
  - 原因：当前运行策略未授权显式委派；且 `FlowEditorWindow.vue / flow.ts` 写集与状态语义紧耦合，由主Agent本地连续实现更安全
