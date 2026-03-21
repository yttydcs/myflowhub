# 2026-03-21_win-flow-project-center

## 变更背景 / 目标
将 Win 端 Flow 页面改造为“本地项目中心”模式，并满足以下确认需求：
- 本地长期持久化（按 profile 隔离）
- 独立 `project_id`
- 首页展示项目列表（每项 Edit / Deploy）
- Deploy 仅执行 `set`（不 run），支持触发器编辑并回写项目默认触发器
- 顶部 `Current Deployments` 支持按 nodeId 查看与删除部署
- 支持手输 nodeId + 设备树选择
- 编辑器使用独立窗口，支持并行编辑多个项目
- 同 flow_id 覆盖部署需二次确认

## 具体变更内容（新增 / 修改 / 删除）
- 新增 `app_flow_projects.go`：
  - App 层新增 `FlowProjectsState` / `SaveFlowProjectsState`。
  - 本地持久化 key：`flow.projects.state.v1`。
  - 使用 profile 作用域存取，包含 normalize/validate。
- 新增 `app_flow_projects_test.go`：
  - 覆盖 parse/normalize/validate 关键路径。
- 修改 `internal/services/flow/service.go`：
  - 新增 `Delete/DeleteSimple`，支持 `delete/delete_resp`。
  - 增加 `DeleteReq/DeleteResp` 本地类型与统一错误码提取分支。
- 新增 `frontend/src/stores/flowProjects.ts`：
  - 本地项目 CRUD + 快照保存。
  - 部署编排（list/get/set/delete）。
  - 覆盖检查（overwriteRequired）与触发器回写策略（方案 B）。
  - 并发编辑保护：保存前先拉取最新快照再合并，减少多窗口互相覆盖。
  - `Current Deployments` 查询（并发 get 限流）与删除。
- 修改 `frontend/src/stores/flow.ts`：
  - 新增 `loadFromPayload` / `exportPayload`。
  - 提炼 payload 装载逻辑，支持本地项目编辑器复用。
- 修改 `frontend/src/pages/Flow.vue`：
  - 重构为项目中心页面。
  - 顶部 `Current Deployments`（nodeId 输入 + 设备树选择 + 删除）。
  - 本地项目列表（Edit / Deploy / Delete）。
  - 新建项目弹窗、部署弹窗（触发器配置 + 覆盖确认）。
- 新增 `frontend/src/windows/FlowEditorWindow.vue`：
  - 独立编辑窗口（纯 workflow 编辑 + 本地保存）。
  - 支持多窗口并行编辑不同项目。
- 修改 `frontend/src/router/index.ts`：
  - 新增 `/flow-editor-window`（`layout: window`）。
  - `/flow` 子标题更新为项目中心语义。

## 对应 plan.md 任务映射
- `WIN-PRJ-1`：完成（App 持久化 API + profile 隔离）。
- `WIN-PRJ-2`：完成（FlowService delete 接口）。
- `WIN-PRJ-3`：完成（flowProjects store）。
- `WIN-PRJ-4`：完成（项目中心页 + 编辑窗口路由）。
- `WIN-PRJ-5`：完成（部署弹窗、触发器回写、覆盖确认、仅 set）。
- `WIN-PRJ-6`：完成（Current Deployments + nodeId 输入/设备树 + 删除）。
- `WIN-PRJ-7`：完成（执行验证并记录阻塞）。

## 关键设计决策与权衡（性能 / 扩展性）
- 项目状态集中持久化到单 key JSON，减少多 key I/O。
- Deployments 采用 list + 并发 get（有限并发）补齐 trigger，控制 N+1 风险。
- 编辑窗口保存采用“读取最新快照后合并”策略，减少多窗口并发覆盖冲突。
- delete 使用 Win 本地类型兜底，避免短期强耦合到 proto 版本升级节奏。

## 测试与验证方式 / 结果
- Win Go 测试：
  - 执行：`GOWORK=off go test ./... -count=1`
  - 结果：失败（仓库基线依赖问题，`protocolexec.CapQueryReq/Resp` 在当前依赖解析中缺失，非本次改动引入）。
- Frontend 构建：
  - 执行：`npm install`，`npm run build`
  - 结果：失败（仓库既有阻塞：缺失 `frontend/wailsjs` 绑定文件，错误发生在 `src/pages/Home.vue` 引用 `../../wailsjs/...`）。
- 功能链路代码检查：
  - 本地项目持久化、编辑窗口加载/保存、部署覆盖确认、当前部署删除逻辑均已落地。

## 潜在影响与回滚方案
- 潜在影响：
  - 多窗口并发编辑同一项目仍可能发生业务层最后写入覆盖（已通过保存前快照合并降低范围）。
  - 若后端未部署 delete 能力，前端删除部署会收到协议错误。
- 回滚方案：
  - 回退新增 `flowProjects` store、`FlowEditorWindow` 路由与页面改造。
  - 回退 App 持久化 API 与 FlowService delete 增量。 
