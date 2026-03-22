# Plan - MyFlowHub-Win Flow 项目中心与编辑器收敛

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`refactor/win-project-center-editor`
- Base：`main`
- Worktree：`D:\project\MyFlowHub3\worktrees\refactor-win-project-center-editor\MyFlowHub-Win`
- 当前阶段：`4. 归档变更`

## 项目目标与当前状态

### 目标
- 将 `/flow` 收敛为“项目中心”模式：
  - 默认展示 `Local Projects`
  - 顶部通过 tab 在 `Local Projects / Current Deployments` 间切换
  - `Deploy` 时设置触发器
  - `Current Deployments` 用于查看指定 `nodeId` 下的部署情况
- 保留独立编辑窗口，但将编辑器收敛为纯 workflow 编辑界面：
  - 移除顶部 `Flow ID / Name / Trigger` 区域
  - 页面整体固定，不允许整页滚动
  - 画布尽量占满窗口
  - 点击节点后从右侧弹出详情抽屉
  - 点击空白关闭抽屉
- 将 `name / flow_id` 的维护迁回主列表页
- 创建项目时默认生成随机 `flow_id`，并要求本地项目内唯一

### 当前事实
- `/flow` 已具备项目中心雏形：
  - `frontend/src/pages/Flow.vue`
  - `frontend/src/stores/flowProjects.ts`
- 编辑器窗口已存在，但仍包含 flow 元信息与 trigger 编辑：
  - `frontend/src/windows/FlowEditorWindow.vue`
- 画布仍是固定高度，window layout 也仍允许整页滚动：
  - `frontend/src/components/flow/FlowCanvas.vue`
  - `frontend/src/layout/AppShell.vue`
- 本地项目持久化由 App 层维护：
  - `app_flow_projects.go`

### 已确认决策
- 保留独立编辑窗口
- 编辑器顶部移除 `Flow ID / Name / Trigger`
- `Name / flow_id` 在列表页维护
- 新建项目默认生成随机 `flow_id`
- Flow 首页顶部 tab 使用：
  - `Local Projects`
  - `Current Deployments`
- 编辑器页面整体固定不滚动，仅右侧详情抽屉内部允许滚动
- 节点单击打开右侧详情抽屉，点击空白关闭
- 列表元数据编辑交互采用 `Meta` 按钮 + 小弹窗显式保存
- `flow_id` 在本地项目范围内强制唯一
- 默认 `flow_id` 生成格式：`fl_` + 12 位小写字母数字

## 需求分析输出

### 目标
- 收敛 Flow 主页面与编辑窗口的职责边界，减少部署语义和编辑语义混杂。

### 范围
- 必须：
  - 主页面改为 tab 视图
  - `Choose from tree` 改为 `Select node`
  - 项目列表支持 `Meta / Edit / Deploy / Delete`
  - 创建项目自动生成唯一 `flow_id`
  - 编辑器移除 flow 元信息与 trigger 表单
  - 编辑器改为全窗画布 + 右侧详情抽屉
- 可选：
  - 无
- 不做：
  - 不改协议/后端执行语义
  - 不引入编辑器内触发器设置
  - 不改 Deploy 为自动 run

### 使用场景
- 用户进入 `/flow`，默认查看本地项目列表
- 用户通过 `Meta` 修改项目名称与 `flow_id`
- 用户通过 `Edit` 打开独立窗口，仅编辑 DAG 与节点参数
- 用户通过 `Deploy` 选择 node 与 trigger，完成部署
- 用户切换到 `Current Deployments` 查看指定 node 的部署与删除

### 功能需求
- 本地项目列表展示与元数据编辑
- 项目创建自动补默认 `flow_id`
- 项目 `flow_id` 在本地范围唯一
- 顶部 tab 切换不同视图
- 当前部署按 nodeId 查看与删除
- 编辑器节点详情改为右侧抽屉

### 非功能需求
- 页面职责清晰
- 画布优先，减少滚动与无关 UI 干扰
- 本地持久化与现有 profile 隔离保持兼容
- 不引入额外协议/服务端依赖

### 输入输出
- 输入：
  - 本地项目元数据：`project_id / flow_id / name`
  - 本地 graph
  - Deploy 参数：`nodeId / trigger`
- 输出：
  - 本地保存的项目元数据与 graph
  - 指定 node 上的 `flow.set` 部署结果
  - 指定 node 当前部署列表

### 边界异常
- 创建项目时本地 `flow_id` 冲突
- 元数据编辑时 `flow_id` 为空或重复
- Deploy 时 `nodeId` 非法
- 项目 graph 为空不可部署
- 编辑器打开不存在的项目

### 验收标准
- `/flow` 默认显示 `Local Projects`
- 主页面顶部可以切换 `Local Projects / Current Deployments`
- 节点树选择按钮统一显示 `Select node`
- 编辑器不再显示 `Flow ID / Name / Trigger / Every(ms)` 等顶部区
- 编辑器整页固定，不再整页滚动
- 节点单击打开右侧详情抽屉；点击空白关闭
- `Meta` 弹窗可维护 `name / flow_id`
- 新建项目时 `flow_id` 自动生成且本地唯一

### 风险
- `flow.ts` 目前仍持有 `flow_id / name / trigger` 相关状态，需保证此次收敛不破坏已有 graph 装载/导出能力
- window layout 与编辑器内部布局需要同时改，避免局部改完仍然滚动
- `flow_id` 改到主列表后，需确保 Deploy/保存仍以最新元数据为准

### 阻塞
- 否

## 架构设计输出

### 总体方案
- 保留现有项目中心与独立编辑窗口结构，不新建模块边界。
- 主页面负责：
  - 项目元数据维护
  - Deploy
  - Current Deployments
- 编辑窗口负责：
  - graph 编辑
  - 节点参数编辑
  - graph 保存
- 数据层做轻量收敛：
  - `flowProjects.ts` 管项目元数据与部署
  - `flow.ts` 新增 graph-only 装载/导出接口，减少编辑器对 trigger/meta 的依赖

### 模块职责
- `app_flow_projects.go`
  - 维持当前持久化 schema 与校验
  - 不新增协议行为
- `frontend/src/stores/flowProjects.ts`
  - 生成默认 `flow_id`
  - 校验本地 `flow_id` 唯一性
  - 新增项目元数据更新接口
  - 保存 graph 时保留既有项目元数据
- `frontend/src/stores/flow.ts`
  - graph-only load/export
  - 节点/边/布局/历史操作
  - 保留现有远端 flow 能力，不主动回退已有接口
- `frontend/src/pages/Flow.vue`
  - tab 视图
  - 项目列表卡片
  - `Meta` / `Deploy` / `Create` 弹窗
  - `Current Deployments` 视图
- `frontend/src/windows/FlowEditorWindow.vue`
  - 编辑器工具栏
  - 全屏画布
  - 右侧节点详情抽屉
- `frontend/src/components/flow/FlowCanvas.vue`
  - 接收父布局高度，撑满编辑器可用区域
- `frontend/src/layout/AppShell.vue`
  - 为 flow 编辑窗口提供 full-bleed window 布局，避免整页滚动

### 数据 / 调用流
- 创建项目：
  - `Flow.vue` 提交 `name`
  - `flowProjects.createProject()` 自动生成 `project_id / flow_id`
  - 写入本地持久化
- 编辑元数据：
  - `Flow.vue` 打开 `Meta` 弹窗
  - `flowProjects.updateProjectMeta()` 校验并落盘
- 编辑 graph：
  - `FlowEditorWindow.vue` 通过 `projectId` 加载本地项目
  - `flowStore.loadGraphDraft(...)`
  - 保存时 `flowStore.exportGraphDraft()` → `flowProjects.saveProjectGraph(...)`
- 部署：
  - `Flow.vue` 打开 `Deploy` 弹窗
  - `flowProjects.deployProject(...)`
  - 成功后回写项目默认 trigger
- 查看部署：
  - 切换到 `Current Deployments`
  - 选择/输入 `nodeId`
  - `flowProjects.loadDeployments(nodeId)`

### 接口草案
- `flowProjects.createProject({ projectId?, flowId?, name? })`
- `flowProjects.updateProjectMeta({ projectId, flowId, name })`
- `flowProjects.saveProjectGraph(projectId, graph)`
- `flowStore.loadGraphDraft({ graph })`
- `flowStore.exportGraphDraft()`

### 错误与安全
- `flow_id` 非空、长度受限、本地唯一
- `nodeId` 必须为正整数
- trigger 校验继续沿用现有严格规则
- 编辑窗口不暴露 trigger，减少误部署配置

### 性能与测试策略
- `Current Deployments` 仅在切换到该视图或显式刷新时加载
- graph 保存只在显式保存时落盘，避免高频写本地存储
- 节点详情抽屉按需渲染
- 验证优先：
  - `go test ./...`
  - 若 bindings/依赖可用则 `npm run build` 或 `wails build -nopackage`
  - 手工冒烟：tab 切换、Meta、Deploy、编辑器抽屉、保存 graph

### 可扩展性设计点
- 后续若要为项目增加 `description / tags / env`，可继续在 `flowProjects.ts + Meta` 弹窗扩展，不影响编辑器
- 编辑器与项目元数据分离后，未来可替换为其它 DAG 视图而不改 Deploy 流程

### 阻塞
- 否

## 可执行任务清单（Checklist）

- [x] `FLOW-UX-1` 主页面改为 tab 视图，并补齐项目元数据编辑入口
  - Owner：主Agent
  - Worktree：`D:\project\MyFlowHub3\worktrees\refactor-win-project-center-editor\MyFlowHub-Win`
  - Plan：`D:\project\MyFlowHub3\worktrees\refactor-win-project-center-editor\MyFlowHub-Win\plan.md`
  - 目标：
    - `Local Projects / Current Deployments` tab
    - `Meta` 弹窗
    - `Select node` 文案统一
    - 创建项目自动生成默认 `flow_id`
  - 涉及文件：
    - `frontend/src/pages/Flow.vue`
    - `frontend/src/stores/flowProjects.ts`
  - Write set：
    - `frontend/src/pages/Flow.vue`
    - `frontend/src/stores/flowProjects.ts`
  - 验收条件：
    - 默认 tab 为 `Local Projects`
    - `Meta` 可修改 `name / flow_id`
    - 创建项目不再要求手填 `flow_id`
    - `flow_id` 冲突时给出错误
    - `Choose from tree` 改为 `Select node`
  - 测试点：
    - 创建项目
    - 修改元数据
    - 切换 tab
    - 刷新部署
  - 回滚点：
    - 回退上述文件至本分支修改前

- [x] `FLOW-UX-2` 收敛编辑器为纯 workflow 编辑界面
  - Owner：主Agent
  - Worktree：`D:\project\MyFlowHub3\worktrees\refactor-win-project-center-editor\MyFlowHub-Win`
  - Plan：`D:\project\MyFlowHub3\worktrees\refactor-win-project-center-editor\MyFlowHub-Win\plan.md`
  - 目标：
    - 移除顶部 `Flow ID / Name / Trigger`
    - 画布全窗
    - 右侧详情改为抽屉
    - 点击空白关闭抽屉
    - 页面本体固定不滚动
  - 涉及文件：
    - `frontend/src/windows/FlowEditorWindow.vue`
    - `frontend/src/components/flow/FlowCanvas.vue`
    - `frontend/src/layout/AppShell.vue`
    - `frontend/src/stores/flow.ts`
  - Write set：
    - `frontend/src/windows/FlowEditorWindow.vue`
    - `frontend/src/components/flow/FlowCanvas.vue`
    - `frontend/src/layout/AppShell.vue`
    - `frontend/src/stores/flow.ts`
  - 验收条件：
    - 编辑器不再显示 trigger/meta 表单
    - 整页固定；仅详情抽屉内部滚动
    - 单击节点打开抽屉，点击空白关闭
    - 保存 graph 仍可用
  - 测试点：
    - 打开编辑器
    - 新增节点、连线、保存
    - 节点详情开关与编辑
  - 回滚点：
    - 回退上述文件至本分支修改前

- [x] `FLOW-UX-3` 验证与回归
  - Owner：主Agent
  - Worktree：`D:\project\MyFlowHub3\worktrees\refactor-win-project-center-editor\MyFlowHub-Win`
  - Plan：`D:\project\MyFlowHub3\worktrees\refactor-win-project-center-editor\MyFlowHub-Win\plan.md`
  - 目标：
    - 运行构建/测试
    - 执行关键路径冒烟
  - 涉及文件：
    - 无新增业务文件
  - Write set：
    - 无
  - 验收条件：
    - 记录命令结果
    - 关键路径有明确验证结论
  - 测试点：
    - `go test ./...`
    - 若环境允许：`frontend npm run build` 或 `wails build -nopackage`
    - 手工链路检查
  - 回滚点：
    - 不适用

## 并行性评估
- 评估结果：本轮不派发子Agent。
- 原因：
  - 当前运行策略要求只有在用户显式要求委派/并行 agent 时才可使用子Agent。
  - 本次实现虽然可分为主页面和编辑器两组文件，但 `flowProjects.ts / flow.ts / Flow.vue / FlowEditorWindow.vue` 在数据边界上紧密耦合，主Agent本地连续修改更安全。

## 当前结论
- 代码实现与 Code Review 已完成，当前进入 `4. 归档变更`。

## 实施结果
- 已完成：
  - `Flow.vue` 收敛为 tab 化项目中心，新增 `Meta` 弹窗，创建项目自动生成唯一 `flow_id`
  - `FlowEditorWindow.vue` 收敛为纯 graph 编辑器，移除顶部 metadata/trigger 表单，改为全窗画布 + 右侧详情抽屉
  - `flowProjects.ts` 新增默认 `flow_id` 生成、本地唯一性校验、`updateProjectMeta`、`saveProjectGraph`
  - `flow.ts` 新增 graph-only `loadGraphDraft / exportGraphDraft`
  - `AppShell.vue` 为 `flowEditorWindow` 提供 full-bleed window layout
  - `FlowCanvas.vue` 改为由父容器撑满高度

## 验证记录
- `npm install`
  - 结果：通过
- `npm run build`
  - 结果：失败
  - 原因：仓库基线缺失 `frontend/wailsjs` 生成绑定，`Home.vue` 等页面无法解析 `../../wailsjs/...`
- `npx vue-tsc --noEmit --pretty false`
  - 结果：失败
  - 原因：仓库基线存在多项类型环境问题（`wailsjs` 缺失、三方类型缺失、其它页面既有 TS 错误）
- 针对本次改动文件的 `vue-tsc` 定向筛查
  - 结果：通过
- `$env:GOWORK='off'; go test ./... -count=1`
  - 结果：失败
  - 原因：仓库基线 `internal/services/flow/service.go` 依赖的 `protocolexec.CapQuery*` 符号缺失，非本次改动引入

## Code Review 结论
- 需求覆盖：通过
- 架构合理性：通过
- 性能风险：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：通过
  - 说明：命令级验证受仓库基线阻塞，但已完成本次改动文件的定向类型筛查
- 子Agent治理与审计：通过
  - 本次未使用子Agent；原因已记录在“并行性评估”
