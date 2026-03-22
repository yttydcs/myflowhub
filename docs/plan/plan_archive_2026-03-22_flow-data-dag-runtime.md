# Data DAG Bindings Plan

## 项目目标与当前状态

### 目标

- 将 `flow` 执行器升级为“数据流 DAG”：
  - 节点执行后可产生结构化结果
  - 后续节点可显式引用祖先节点结果
  - 首批新增 `compose` 节点
- 为 Win 编辑器提供默认表单化输入绑定模型，并保留高级 JSON 模式
- 将长期需求与技术规范同步到 `MyFlowHub-Server/docs`

### 当前状态

- `MyFlowHub-SubProto/flow` 目前只记录节点 `status/code/msg`，不保留节点结果
- `MyFlowHub-Win` 目前的节点草稿模型仅包含 `method/target/args`
- `MyFlowHub-Server/docs/specs/flow.md` 目前未覆盖数据流节点结果与输入绑定模型

## Workflow 信息

- 当前仓库：`MyFlowHub-SubProto`
- 当前分支：`feat/data-dag-bindings`
- Base 分支：`main`
- 当前 worktree：`D:\project\MyFlowHub3\worktrees\subproto-data-dag-bindings`
- 当前阶段：`4`
- 主执行 worktree：是

### 跨仓 worktree

- `MyFlowHub-Win`
  - 分支：`feat/data-dag-editor`
  - Worktree：`D:\project\MyFlowHub3\worktrees\win-data-dag-editor`
  - Plan：`D:\project\MyFlowHub3\worktrees\win-data-dag-editor\plan.md`
  - 责任边界：编辑器节点模型、输入绑定 UI、序列化导入导出
- `MyFlowHub-Server`
  - 分支：`chore/data-dag-specs`
  - Worktree：`D:\project\MyFlowHub3\worktrees\server-data-dag-specs`
  - Plan：`D:\project\MyFlowHub3\worktrees\server-data-dag-specs\plan.md`
  - 责任边界：长期 requirements/specs 文档与索引

## Docs Governor 结论

- 使用 `$docs-governor` 进行文档路由与影响检查
- Requirements impact：`add`
- Specs impact：`update`
- Related requirements：
  - `D:\project\MyFlowHub3\worktrees\server-data-dag-specs\docs\requirements\flow_data_dag.md`
- Related specs：
  - `D:\project\MyFlowHub3\worktrees\server-data-dag-specs\docs\specs\flow.md`
- 计划归档目标：
  - `D:\project\MyFlowHub3\worktrees\subproto-data-dag-bindings\docs\change\2026-03-22_flow-data-dag-runtime.md`

## 可执行任务清单（Checklist）

- [x] `DAG-DOC-1` 补齐长期 requirement 文档与 requirements 索引
- [x] `DAG-DOC-2` 更新 Flow 长期 spec
- [x] `DAG-RT-1` 建立 run context 与节点结果存储
- [x] `DAG-RT-2` 实现输入绑定解析与 call 节点输入物化
- [x] `DAG-RT-3` 实现 compose 节点与图校验增强
- [x] `DAG-RT-4` 增补回归测试与错误路径测试
- [x] `DAG-WIN-1` 扩展编辑器 store 数据模型与 spec 导入导出
- [x] `DAG-WIN-2` 实现表单化输入绑定与 compose 节点编辑 UI
- [x] `DAG-WIN-3` 补齐前端验证与高级 JSON 模式联动
- [x] `DAG-INT-1` 统一集成、回归验证、Code Review 与归档

## Task 详情

### `DAG-DOC-1`

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\server-data-dag-specs`
- Plan：`D:\project\MyFlowHub3\worktrees\server-data-dag-specs\plan.md`
- 目标：
  - 新增 Flow 数据流 DAG requirement 文档
  - 更新 `docs/requirements/README.md`
- 涉及模块 / 文件：
  - `docs/requirements/flow_data_dag.md`
  - `docs/requirements/README.md`
- Write set：
  - `docs/requirements/flow_data_dag.md`
  - `docs/requirements/README.md`
- 验收条件：
  - requirements 文档覆盖目标、范围、场景、验收标准
  - requirements 索引可导航到新文档
- 测试点：
  - 文档一致性人工审阅
- 回滚点：
  - 回退上述两个文件
- 依赖关系：
  - 无
- 风险与注意事项：
  - requirements 与 specs 不得重复维护相同技术细节
- 关键上下文引用：
  - `docs/specs/flow.md`
  - 本计划

### `DAG-DOC-2`

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\server-data-dag-specs`
- Plan：`D:\project\MyFlowHub3\worktrees\server-data-dag-specs\plan.md`
- 目标：
  - 更新 `docs/specs/flow.md`
  - 明确 `call/compose` 节点、输入绑定、运行上下文与结果可见性
- 涉及模块 / 文件：
  - `docs/specs/flow.md`
- Write set：
  - `docs/specs/flow.md`
- 验收条件：
  - 新模型可指导 `SubProto/flow` 与 `MyFlowHub-Win` 实现
  - 旧的动作集与路由约定未被误写
- 测试点：
  - 对照需求文档、现有 proto actions 和执行器实现人工审阅
- 回滚点：
  - 回退 `docs/specs/flow.md`
- 依赖关系：
  - `DAG-DOC-1`
- 风险与注意事项：
  - 不得手改 generated 文档
- 关键上下文引用：
  - `docs/requirements/flow_data_dag.md`
  - 本计划

### `DAG-RT-1`

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\subproto-data-dag-bindings`
- Plan：`D:\project\MyFlowHub3\worktrees\subproto-data-dag-bindings\plan.md`
- 目标：
  - 建立单次 run 的运行上下文
  - 为节点状态增加结果存储
  - 规范化 trigger 运行时上下文
- 涉及模块 / 文件：
  - `flow/handler.go`
  - `flow/types.go`
  - 视实现需要新增 helper 文件
- Write set：
  - `flow/handler.go`
  - `flow/types.go`
  - `flow/*.go` 中新增 helper 文件
- 验收条件：
  - `call` / `compose` 执行后都能写入 `RunContext`
  - 现有 `status` 摘要接口不被破坏
- 测试点：
  - 本地节点返回 JSON 结果后可在运行上下文中命中
  - interval/event/var_changed 触发时可生成统一 trigger 上下文
- 回滚点：
  - 回退运行上下文新增结构与状态记录逻辑
- 依赖关系：
  - `DAG-DOC-2`
- 风险与注意事项：
  - 避免在 `status` 路径引入大结果回传
- 关键上下文引用：
  - `D:\project\MyFlowHub3\worktrees\server-data-dag-specs\docs\specs\flow.md`

### `DAG-RT-2`

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\subproto-data-dag-bindings`
- Plan：`D:\project\MyFlowHub3\worktrees\subproto-data-dag-bindings\plan.md`
- 目标：
  - 实现结构化输入绑定解析
  - 支持 `call` 节点基于 `args_template + inputs` 物化最终调用参数
- 涉及模块 / 文件：
  - `flow/handler.go`
  - 新增输入绑定解析 helper
- Write set：
  - `flow/handler.go`
  - `flow/*.go` 中新增 binding helper 文件
- 验收条件：
  - 支持 `node_result` / `trigger` / `flow_meta` / `run_meta`
  - 必填绑定缺失时节点失败并返回明确错误
- 测试点：
  - 上游结果字段映射到下游参数
  - 非法 JSON Pointer 被拒绝
  - 缺失必填绑定失败
- 回滚点：
  - 回退输入绑定解析与 call 物化逻辑
- 依赖关系：
  - `DAG-RT-1`
- 风险与注意事项：
  - 避免多次整包 JSON 深拷贝
- 关键上下文引用：
  - `D:\project\MyFlowHub3\worktrees\server-data-dag-specs\docs\specs\flow.md`

### `DAG-RT-3`

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\subproto-data-dag-bindings`
- Plan：`D:\project\MyFlowHub3\worktrees\subproto-data-dag-bindings\plan.md`
- 目标：
  - 新增 `compose` 节点执行能力
  - 在 `set` 阶段补齐输入绑定与祖先依赖校验
- 涉及模块 / 文件：
  - `flow/handler.go`
  - `flow/graph_test.go`
  - 视实现需要新增 helper 文件
- Write set：
  - `flow/handler.go`
  - `flow/graph_test.go`
  - `flow/*.go` 中新增 compose / validation helper 文件
- 验收条件：
  - `compose` 节点可输出 JSON 结果
  - 非祖先引用在保存前被拒绝
- 测试点：
  - `compose` 多来源拼装
  - 未来节点引用失败
  - 未知 node_id 引用失败
- 回滚点：
  - 回退 compose 和校验增强逻辑
- 依赖关系：
  - `DAG-RT-2`
- 风险与注意事项：
  - 祖先判定必须基于 DAG 真实可达性，而不是节点声明顺序
- 关键上下文引用：
  - `D:\project\MyFlowHub3\worktrees\server-data-dag-specs\docs\specs\flow.md`

### `DAG-RT-4`

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\subproto-data-dag-bindings`
- Plan：`D:\project\MyFlowHub3\worktrees\subproto-data-dag-bindings\plan.md`
- 目标：
  - 增补回归测试、错误路径测试和兼容测试
- 涉及模块 / 文件：
  - `flow/*_test.go`
- Write set：
  - `flow/*_test.go`
- 验收条件：
  - 关键数据流与错误路径有自动化测试覆盖
- 测试点：
  - 节点结果传递
  - compose
  - 缺失必填输入
  - 兼容旧 `args` 直写 call 节点
- 回滚点：
  - 回退新增测试文件
- 依赖关系：
  - `DAG-RT-1`
  - `DAG-RT-2`
  - `DAG-RT-3`
- 风险与注意事项：
  - 测试不要依赖主工作区的非稳定文件路径
- 关键上下文引用：
  - 本计划

### `DAG-WIN-1`

- Owner：待分配
- Worktree：`D:\project\MyFlowHub3\worktrees\win-data-dag-editor`
- Plan：`D:\project\MyFlowHub3\worktrees\win-data-dag-editor\plan.md`
- 目标：
  - 扩展 store 节点草稿模型
  - 支持 `call/compose` 新 spec 的导入导出
- 涉及模块 / 文件：
  - `frontend/src/stores/flow.ts`
- Write set：
  - `frontend/src/stores/flow.ts`
- 验收条件：
  - 新 spec 可无损导出
  - 旧 `args` 节点可加载并迁移到新草稿结构
- 测试点：
  - load/export round-trip
  - 旧 payload 兼容加载
- 回滚点：
  - 回退 `frontend/src/stores/flow.ts`
- 依赖关系：
  - `DAG-DOC-2`
- 风险与注意事项：
  - 不能破坏现有节点布局 `_ui`
- 关键上下文引用：
  - `D:\project\MyFlowHub3\worktrees\server-data-dag-specs\docs\specs\flow.md`

### `DAG-WIN-2`

- Owner：待分配
- Worktree：`D:\project\MyFlowHub3\worktrees\win-data-dag-editor`
- Plan：`D:\project\MyFlowHub3\worktrees\win-data-dag-editor\plan.md`
- 目标：
  - 在编辑器中提供表单化输入绑定 UI
  - 提供 `compose` 节点配置 UI
- 涉及模块 / 文件：
  - `frontend/src/windows/FlowEditorWindow.vue`
  - 视实现需要新增子组件
- Write set：
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/components/flow/*` 中新增或修改的输入绑定组件
- 验收条件：
  - 默认可视化配置 `inputs`
  - `compose` 节点可被创建和编辑
- 测试点：
  - 新增 call node 后可配置绑定
  - 新增 compose node 后可配置模板
- 回滚点：
  - 回退编辑器 UI 改动
- 依赖关系：
  - `DAG-WIN-1`
- 风险与注意事项：
  - 保留高级 JSON 模式，不强制锁死在表单模式
- 关键上下文引用：
  - `frontend/src/stores/flow.ts`
  - `D:\project\MyFlowHub3\worktrees\server-data-dag-specs\docs\specs\flow.md`

### `DAG-WIN-3`

- Owner：待分配
- Worktree：`D:\project\MyFlowHub3\worktrees\win-data-dag-editor`
- Plan：`D:\project\MyFlowHub3\worktrees\win-data-dag-editor\plan.md`
- 目标：
  - 补齐前端验证、错误提示和高级 JSON 模式联动
- 涉及模块 / 文件：
  - `frontend/src/stores/flow.ts`
  - `frontend/src/windows/FlowEditorWindow.vue`
- Write set：
  - `frontend/src/stores/flow.ts`
  - `frontend/src/windows/FlowEditorWindow.vue`
- 验收条件：
  - 非祖先引用、非法路径等在前端有明确提示
  - 高级 JSON 编辑与表单模式不会互相覆盖损坏
- 测试点：
  - 校验提示
  - 模式切换
- 回滚点：
  - 回退前端验证与模式联动逻辑
- 依赖关系：
  - `DAG-WIN-1`
  - `DAG-WIN-2`
- 风险与注意事项：
  - 避免把协议细节再次扩散成大量手写字符串规则
- 关键上下文引用：
  - 本计划

### `DAG-INT-1`

- Owner：主Agent
- Worktree：`D:\project\MyFlowHub3\worktrees\subproto-data-dag-bindings`
- Plan：`D:\project\MyFlowHub3\worktrees\subproto-data-dag-bindings\plan.md`
- 目标：
  - 集成跨仓改动
  - 执行回归验证
  - 完成 Code Review 与变更归档
- 涉及模块 / 文件：
  - `flow/*`
  - `frontend/src/stores/flow.ts`
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `docs/requirements/flow_data_dag.md`
  - `docs/specs/flow.md`
  - 各仓 `docs/change/*`
- Write set：
  - 限于上述任务已声明文件
- 验收条件：
  - 跨仓实现、Review、归档闭环
  - 无计划外改动
- 测试点：
  - `go test ./...`
  - 前端构建 / 冒烟
  - 文档一致性复核
- 回滚点：
  - 按 repo 分别回退对应分支上的本次提交
- 依赖关系：
  - 所有前置任务
- 风险与注意事项：
  - 集成前必须复核 write set 和任务映射
- 关键上下文引用：
  - 本计划
  - 两个关联 worktree 的 plan
