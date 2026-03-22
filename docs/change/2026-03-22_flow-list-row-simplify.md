# 2026-03-22 Flow 列表行简化

## 变更背景 / 目标

- 背景：`Flow` 页本地项目列表同时显示项目名称、`Flow ID`、`Project ID` 和更新时间，信息密度偏高，且当名称为空时会直接回退展示 `flowId`。
- 目标：让本地项目列表只保留用户更关心的名称和最后更新时间，不再在列表中展示技术 ID，并让名称为空时使用非 ID 的占位文案。

## 文档治理与影响检查

- Requirements impact: `none`
- Specs impact: `none`
- Related requirements: `none`
- Related specs: `none`
- Lessons impact: `none`
- Index updates:
  - 已更新 `docs/change/README.md`
  - 未更新 `docs/README.md`，原因是 docs 顶层拓扑未变化
  - 未更新 `docs/plan/README.md`，原因是当前 workflow 计划仍以 worktree 根 `plan.md` 为准

## 具体变更内容（新增 / 修改 / 删除）

### 修改

- `frontend/src/pages/Flow.vue`
  - 删除本地项目列表中的 `Flow ID` 行。
  - 删除本地项目列表中的 `Project ID` 行。
  - 将项目名称与最后更新时间合并为同一行。
  - 将该行样式统一为灰色 `text-muted-foreground`。
  - 将名称为空时的回退值从 `flowId` 改为 `Untitled Project`，避免继续在列表中暴露流程 ID。
- `plan.md`
  - 更新当前 workflow 的任务状态、验证结果与 Code Review 结论。

### 未修改

- 未修改 `flowProjects` store、后端接口、数据结构或更新时间生成逻辑。
- 未修改部署列表、项目元数据弹窗和编辑器窗口。
- 未修改 `requirements`、`specs`、`lessons` 分类文档。

## 对应 plan.md 任务映射

- `T1` 调整 Flow 本地项目列表展示：完成。
- `T2` 执行验证与 Code Review：完成。
- `T3` 归档变更并更新索引：完成。

## 关键设计决策与权衡

- 只改页面模板，不改 store。
  - 原因：本次是纯展示层收敛，不应把 UI 呈现策略下沉到状态层。
  - 收益：影响面最小，回滚简单，不引入新的状态耦合。
- 空名称兜底改为 `Untitled Project`，而不是继续用 `flowId`。
  - 原因：用户明确要求列表不再显示流程 ID。
  - 收益：即使缺少名称，列表也不会再次暴露技术标识。
- 颜色继续复用 `text-muted-foreground`。
  - 原因：保持现有设计 token，一致性优于硬编码灰色值。
- 性能说明：
  - 本次变更只减少模板节点，不新增请求、不新增循环、不新增计算、不新增 I/O。
  - 列表渲染成本只会下降，不存在 N+1 或重复计算风险。

## 测试与验证方式 / 结果

- 文本核对：
  - 检查 `frontend/src/pages/Flow.vue` 的本地项目列表模板。
  - 结果：本地项目列表片段已不再渲染 `Flow ID` 和 `Project ID`。
- Vue SFC 解析：
  - 命令：使用 `@vue/compiler-sfc` 解析 `frontend/src/pages/Flow.vue`
  - 结果：通过。
- 依赖准备：
  - 命令：`npm ci`
  - 结果：通过。
- 前端构建：
  - 命令：`npm run build`
  - 结果：失败。
  - 失败原因：`src/pages/Home.vue` 仍无法解析 `../../wailsjs/go/main/App`，属于当前仓既有 `frontend/wailsjs` 生成物缺失问题，与本次列表界面调整无关。

## Code Review 结论

- 需求覆盖：通过。目标列表已隐藏 `Flow ID` / `Project ID`，并把名称和更新时间合并为单行灰色文本。
- 架构合理性：通过。改动限定在页面模板层，没有扩大到 store 或接口层。
- 性能风险：通过。纯渲染层删减，无额外请求、计算或 I/O。
- 可读性与一致性：通过。列表头部信息更聚焦，继续复用现有国际化和设计 token。
- 可扩展性与配置化：通过。后续如需调整列表密度，仍可只在页面层扩展。
- 稳定性与安全：通过。未触碰业务调用、权限、存储或输入处理。
- 测试覆盖情况：通过。目标文件语法校验和静态核对完成；完整构建受既有环境问题阻塞，已记录残余风险。
- 子Agent治理与审计：通过。本轮未使用子Agent。

## 潜在影响与回滚方案

- 潜在影响：
  - 本地项目列表会更紧凑，名称缺失时显示 `Untitled Project`。
  - 依赖技术 ID 做人工辨识的用户将无法在列表直接看到 ID，但元数据弹窗仍保留这些字段。
- 回滚方案：
  - 回退 `frontend/src/pages/Flow.vue`
  - 回退 `plan.md`
  - 删除 `docs/change/2026-03-22_flow-list-row-simplify.md`
  - 回退 `docs/change/README.md`

## 子Agent执行轨迹

- `T1` → `主Agent` → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-flow-list-row-simplify` → `frontend/src/pages/Flow.vue` → 验收通过
- `T2` → `主Agent` → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-flow-list-row-simplify` → `plan.md` → 验收通过
- `T3` → `主Agent` → `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-flow-list-row-simplify` → `docs/change/2026-03-22_flow-list-row-simplify.md`, `docs/change/README.md` → 验收通过
