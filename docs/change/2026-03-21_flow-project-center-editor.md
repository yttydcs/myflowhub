# 2026-03-21_flow-project-center-editor

## 变更背景 / 目标
- 收敛 Win 端 Flow 项目中心与编辑器职责：
  - `/flow` 默认以本地项目列表为主
  - 顶部通过 tab 在 `Local Projects / Current Deployments` 间切换
  - `Deploy` 时设置触发器
  - 编辑窗口仅负责 workflow 图编辑，不再展示 `Flow ID / Name / Trigger`
  - 编辑窗口整页固定，不允许页面滚动；节点详情改为右侧抽屉
- 额外确认要求：
  - 保留独立编辑窗口
  - `name / flow_id` 在项目列表侧维护
  - 新建项目时自动生成唯一 `flow_id`
  - `Select node` 替换原 `Choose from tree`

## 具体变更内容（新增 / 修改 / 删除）

### 修改
- `frontend/src/pages/Flow.vue`
  - 新增顶部 tab：`Local Projects / Current Deployments`
  - 本地项目区新增 `Meta` 按钮与元数据弹窗
  - 新建项目改为仅输入 `name`，自动生成 `project_id / flow_id`
  - 部署相关文案统一为 `Select node`
  - `Current Deployments` 改为按 tab 懒加载，并在重连后强制刷新

- `frontend/src/windows/FlowEditorWindow.vue`
  - 移除顶部 `Flow ID / Name / Trigger` 表单
  - 收敛为纯 graph 编辑器
  - 页面布局改为全窗固定
  - 节点详情改为右侧抽屉，点击空白关闭
  - 保存逻辑改为只保存 graph

- `frontend/src/stores/flowProjects.ts`
  - 新增默认 `flow_id` 生成逻辑：`fl_` + 12 位小写字母数字
  - 新增本地 `flow_id` 唯一性校验
  - 新增：
    - `updateProjectMeta(...)`
    - `saveProjectGraph(...)`
  - `createProject(...)` 改为支持自动生成 `flow_id`

- `frontend/src/stores/flow.ts`
  - 新增 graph-only 接口：
    - `loadGraphDraft(...)`
    - `exportGraphDraft()`
  - 保留原有 `loadFromPayload / exportPayload`，避免影响现有远端 flow 调用链

- `frontend/src/components/flow/FlowCanvas.vue`
  - 画布容器由固定高度改为跟随父容器撑满

- `frontend/src/layout/AppShell.vue`
  - 为 `flowEditorWindow` 提供 full-bleed window layout
  - 移除该窗口页面级滚动与默认 padding

### 删除
- 删除编辑器顶部 metadata/trigger 表单入口

## 对应 `plan.md` 任务映射
- `FLOW-UX-1`
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/stores/flowProjects.ts`
- `FLOW-UX-2`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/components/flow/FlowCanvas.vue`
  - `frontend/src/layout/AppShell.vue`
- `FLOW-UX-3`
  - 命令验证与定向类型筛查

## 关键设计决策与权衡（尤其性能 / 扩展性）
- 主页面与编辑窗口职责拆分：
  - 项目元数据与部署保留在项目中心
  - graph 编辑独立到窗口
  - 好处：后续扩展项目属性时不需要再污染编辑器
- `Current Deployments` 采用 tab 懒加载：
  - 避免用户每次进入 `/flow` 都触发远端 `list + get`
  - 重连后强制刷新，避免展示旧状态
- `flow.ts` 新增 graph-only 接口而不是删除原接口：
  - 兼容已有远端 flow 调试/保存调用
  - 降低本次 UI 收敛对既有能力的破坏面
- `flow_id` 本地唯一：
  - 在部署前就尽量阻止“本地不同项目覆盖同一远端 flow”的误操作

## 测试与验证方式 / 结果
- `npm install`
  - 结果：通过
- `npm run build`
  - 结果：失败
  - 原因：仓库基线缺失 `frontend/wailsjs` 生成绑定，`Home.vue` 等页面无法解析 `../../wailsjs/...`
- `npx vue-tsc --noEmit --pretty false`
  - 结果：失败
  - 原因：仓库基线存在三方类型缺失、`wailsjs` 缺失、其它页面既有 TS 错误
- 定向 `vue-tsc` 筛查本次改动文件
  - 结果：通过
  - 范围：
    - `src/pages/Flow.vue`
    - `src/windows/FlowEditorWindow.vue`
    - `src/stores/flow.ts`
    - `src/stores/flowProjects.ts`
    - `src/components/flow/FlowCanvas.vue`
- `$env:GOWORK='off'; go test ./... -count=1`
  - 结果：失败
  - 原因：仓库基线 `internal/services/flow/service.go` 依赖的 `protocolexec.CapQuery*` 符号缺失，非本次改动引入

## 潜在影响与回滚方案
- 潜在影响：
  - 已打开的编辑窗口不会实时感知列表页的 `name / flow_id` 变更；保存 graph 时会保留最新快照，但窗口标题可能需要重新打开后才完全同步
  - 本地 `flow_id` 唯一性当前在前端 store 层强制，旧持久化数据若已存在重复项，不会自动清洗
- 回滚方案：
  - 回退以下文件即可恢复到本次改动前：
    - `frontend/src/pages/Flow.vue`
    - `frontend/src/windows/FlowEditorWindow.vue`
    - `frontend/src/stores/flow.ts`
    - `frontend/src/stores/flowProjects.ts`
    - `frontend/src/components/flow/FlowCanvas.vue`
    - `frontend/src/layout/AppShell.vue`

## 子Agent执行轨迹（Task ID → Agent → Worktree → 文件 → 验收结果）
- 本次未使用子Agent
  - 原因：当前运行策略未授权显式委派；且本次 `flowProjects / flow / Flow.vue / FlowEditorWindow.vue` 写集与上下文边界紧密耦合，由主Agent本地连续实现更安全
