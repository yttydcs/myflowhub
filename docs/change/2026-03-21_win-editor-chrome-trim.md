# 2026-03-21 Win Flow 编辑器文案精简

## 变更背景 / 目标
- 用户希望 Flow 独立编辑界面更简洁，去掉“解释当前怎么操作”的说明性文字，让页面进入后更接近纯编辑视图。
- 本次目标是只收敛编辑窗口中的静态提示文案，不改变编辑器功能、部署逻辑和节点详情抽屉行为。

## 具体变更内容（新增 / 修改 / 删除）
### 修改
- `frontend/src/windows/FlowEditorWindow.vue`
  - 删除 header 下方 `project_id ...` 文案。
  - 删除 header 下方 `Pure workflow editing only...` 文案。
  - 删除画布上方两段说明文案：
    - `Drag nodes to reposition...`
    - `Click blank canvas to close the drawer.`
  - 删除不再使用的 `loadedProjectId` 状态与赋值，保持脚本区干净。

### 新增
- `docs/change/2026-03-21_win-editor-chrome-trim.md`
  - 记录本次界面文案精简的范围、验证与回滚方式。

### 删除
- 无独立文件删除。

## 对应 plan.md 任务映射
- `EDITOR-CHROME-1`
  - 更新本次 workflow 的需求分析、架构设计与任务拆分。
- `EDITOR-CHROME-2`
  - 修改 `FlowEditorWindow.vue`，删除目标说明文案与无用状态。
- `EDITOR-CHROME-3`
  - 生成 Wails bindings，执行前端构建验证，完成 Review 与归档。

## 关键设计决策与权衡
- 只收敛模板层，不改布局骨架和交互。
  - 原因：用户要的是“去掉描述性文字”，不是重做编辑器结构；最小改动更稳。
- 保留 `Flow Project Editor` 标题和工具栏按钮。
  - 原因：这部分仍承担页面识别和编辑动作入口，不属于冗余说明。
- 删除 `loadedProjectId` 状态。
  - 原因：模板移除 `project_id` 后，该状态已无消费点，继续保留会形成死代码。

## 测试与验证方式 / 结果
- Wails bindings 生成：
  - 执行：`$env:GOWORK='off'; wails generate module`
  - 结果：通过
- 前端依赖安装：
  - 执行：`cd frontend && npm install`
  - 结果：通过
- 前端构建：
  - 执行：`cd frontend && npm run build`
  - 结果：通过

## 潜在影响与回滚方案
### 潜在影响
- 编辑器首屏说明进一步减少，新用户需要自行探索拖拽和空白关闭抽屉行为。
- 由于这次不改工具栏和标题，界面仍保留基本识别信息，不会变成“无上下文”的空壳。

### 回滚方案
- 回退以下文件即可恢复：
  - `frontend/src/windows/FlowEditorWindow.vue`

## 子Agent执行轨迹（Task ID → Agent → Worktree → 文件 → 验收结果）
- 本次未使用子Agent
  - 原因：改动集中在单一窗口组件，主Agent本地串行处理更直接。
