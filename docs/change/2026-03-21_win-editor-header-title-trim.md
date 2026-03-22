# 2026-03-21 Win Flow 编辑器顶部标题移除

## 变更背景 / 目标
- 用户要求继续精简 Win Flow 独立编辑窗口，移除顶部 `Flow Project Editor` 标题说明，只保留项目名和工具栏。
- 本次目标是以最小改动完成 header 收敛，不影响编辑器已有交互、节点抽屉和方法选择逻辑。

## 具体变更内容（新增 / 修改 / 删除）
### 修改
- `frontend/src/windows/FlowEditorWindow.vue`
  - 删除 header 左侧的 `Flow Project Editor` 静态 `<p>` 标签。
  - 同步移除项目标题 `<h1>` 的 `mt-1`，避免副标题删除后标题垂直位置下沉。

### 新增
- `docs/change/2026-03-21_win-editor-header-title-trim.md`
  - 记录本次变更背景、任务映射、验证结果和回滚方式。

### 删除
- 无独立文件删除。

## 对应 plan.md 任务映射
- `FLOW-HEADER-1`
  - 完成当前 workflow 的需求分析、架构设计和任务拆分文档更新。
- `FLOW-HEADER-2`
  - 修改 `FlowEditorWindow.vue`，移除顶部标题说明并校正项目名对齐。
- `FLOW-HEADER-3`
  - 执行 Wails 模块生成、前端依赖安装、构建验证，并完成 Review 与归档。

## 关键设计决策与权衡
- 直接删除静态副标题，而不是保留空 DOM 或条件渲染。
  - 原因：该文案已被明确判定为不需要，保留空结构只会增加模板噪音。
  - 收益：实现最简单，副作用面最小，可维护性最好。
- 同步移除 `<h1>` 的 `mt-1`。
  - 原因：原间距只服务于副标题与主标题之间的纵向层级。
  - 收益：副标题消失后，项目名可自然回到 header 顶部基线，不产生额外留白。

## 测试与验证方式 / 结果
- Wails 模块生成：
  - 命令：`$env:GOWORK='off'; wails generate module`
  - 结果：通过。
- 前端依赖安装：
  - 命令：`cd frontend && npm install`
  - 结果：通过。
- 前端生产构建：
  - 命令：`cd frontend && npm run build`
  - 结果：通过。
- 静态检查：
  - 命令：`rg -n "Flow Project Editor" frontend/src/windows/FlowEditorWindow.vue`
  - 结果：无匹配，目标文案已移除。

## 潜在影响与回滚方案
### 潜在影响
- header 左侧只保留项目名后，视觉层级更简洁；如果未来又需要恢复“编辑器说明”定位，需要重新设计 header 文案层级。
- 构建阶段仍存在 Vite chunk size warning，但属于现有前端体积问题，不是本次变更引入。

### 回滚方案
- 回滚 `frontend/src/windows/FlowEditorWindow.vue`，恢复顶部 `<p>` 标签和 `<h1>` 的 `mt-1` 类名。
- 删除本次归档文档。

## 子Agent执行轨迹
- 本次未使用子Agent。
- 原因：改动集中在单个 Vue 文件和文档，写集单一，不具备安全并行拆分价值。
