# VarPool Tab 化单列重构计划

## 项目目标与当前状态
- 项目目标：将 `VarPool` 页面改造成与 `Flow` 接近的顶部 tab 切换结构，收敛为单列内容流，降低响应式布局压力，同时保持现有变量操作能力和 store / 协议语义不变。
- 当前状态：阶段 1（需求分析）完成，阶段 2（架构设计）完成，阶段 3.1（计划拆分）完成，阶段 3.2（代码编写）完成，阶段 3.3（Code Review）完成，阶段 4（归档变更）完成。
- 当前仓库：MyFlowHub-Win
- 当前分支：refactor/varpool-tabs-layout
- Base 分支：main
- Base 提交：312923d
- Worktree：D:/project/MyFlowHub3/worktrees/MyFlowHub-Win-varpool-tabs-layout
- 计划文档：D:/project/MyFlowHub3/worktrees/MyFlowHub-Win-varpool-tabs-layout/plan.md

## 阶段 1：需求分析

### 目标
- 为 `VarPool` 页面增加顶部 tab：`Control`、`Mine`、`Watch`
- 用单列页面结构替代当前主双列布局
- 保留现有变量列表、订阅、编辑、撤销、删除、刷新、新增 watch / mine 的操作能力

### 范围
- 必须：
  - 新增 `Control / Mine / Watch` 顶部 tab
  - 默认展示 `Control`
  - `Control` 保留当前顶部控制能力
  - `Control` 包含 `Active Subscriptions`
  - `Control` 增加同级摘要信息：`Connected`、`NodeID`、`HubID`、`Cached Keys`、`Mine Count`、`Watch Count`、`Subscribed Count`、`Last Frame`
  - `Mine` 和 `Watch` 分别承载现有两类变量列表
  - 变量展示改为单列、一行一个
  - `Watch` 继续保留 `Edit / Revoke`
- 可选：
  - 仅允许对摘要信息做版式优化，不扩展新业务能力
- 不做：
  - 不调整后端协议
  - 不改动 `varpool` store 对外接口
  - 不新增搜索、过滤、排序、分页
  - 不改动路由、导航位置或其它页面

### 使用场景
- 用户在 `Control` 中完成连接态确认、目标节点设置、全量刷新、watch list 保存和订阅概览查看
- 用户在 `Mine` 中集中管理自有变量
- 用户在 `Watch` 中集中管理他人变量和订阅关系
- 用户在中等宽度窗口下仍能稳定浏览页面，不受双列布局挤压影响

### 功能需求
- 页面顶部采用与 `Flow` 一致的 pill 式 tab 交互风格
- tab 切换仅切换可见内容区，不改变既有网络调用语义
- 对话框 `Add Mine`、`Add Watch`、`Edit`、`Node Vars` 保持可用
- `Mine` 空态、`Watch` 空态、订阅空态保持明确提示

### 非功能需求
- 性能：减少页面级大块响应式 grid，避免不必要重排；不新增额外请求
- 可读性：让 `Control / Mine / Watch` 成为显式视觉分区
- 可扩展性：后续增加新 tab 或新的摘要项时无需改协议层
- 可维护性：尽量收敛在页面层，变更最小化，可回滚

### 输入输出
- 输入：会话连接状态、`nodeId`、`hubId`、`varpool.state`、用户 tab 切换和操作按钮点击
- 输出：重构后的 `VarPool` 页面结构与交互呈现

### 边界异常
- 未连接或未登录时继续沿用现有错误提示逻辑
- 变量为空、订阅为空时需要稳定空态
- 长变量值不能因布局重排而明显降低可读性

### 验收标准
- 页面顶部存在 `Control / Mine / Watch` 三个 tab
- 默认 tab 为 `Control`
- 页面级双列布局移除，主内容采用单列结构
- `Control` 中可完成现有顶部操作，并展示订阅列表和摘要信息
- `Mine` 和 `Watch` 保持原有操作能力，行为不退化
- 不改动协议层和 store 接口语义

### 风险
- 页面重排时遗漏原有摘要信息或按钮入口
- `Watch` 保留强操作能力，布局必须保持清晰，避免误触

### 结论
- 阻塞：否

## 阶段 2：架构设计（分析）

### 总体方案
- 仅改前端页面层，主写集收敛在 `frontend/src/pages/VarPool.vue`
- 复用 `Flow.vue` 的 tab 样式和交互方式，不引入新的 tab 组件
- 保持 `useVarPoolStore()`、现有 action handlers、对话框和 Wails 调用链不变
- 通过新增页面内 tab 状态和少量 computed view-model，重组视图而不是重写业务逻辑

### 选型理由 / 备选对比
- 方案 A：只改 `VarPool.vue`
  - 优点：写集最小，风险最低，便于回滚，适合本次纯布局语义重构
  - 缺点：页面文件长度增加
- 方案 B：拆出变量卡片子组件
  - 优点：复用更强
  - 缺点：新增文件、事件透传和 props 设计，超过本轮最小变更目标
- 结论：采用方案 A，在单文件内通过 computed 和局部结构优化保持可读性

### 模块职责
- `frontend/src/pages/VarPool.vue`
  - 管理 `activeTab`
  - 组织 `Control / Mine / Watch` 三个内容区
  - 组织摘要视图模型和变量展示
  - 复用现有按钮、对话框和操作 handler
- `frontend/src/stores/varpool.ts`
  - 保持现有数据与动作接口，不做语义调整
- `frontend/src/components/varpool/NodeVarsDialog.vue`
  - 保持现状，仅作为 watch 辅助入口

### 数据 / 调用流
- 页面初始化与现有一致：`loadHomeDefaults()` -> `reloadWatchList(true)` -> 满足条件时 `refreshAll()`
- tab 切换只影响展示层，不触发新的请求
- `Mine` 使用现有 `groupedKeys.mine`
- `Watch` 使用现有 `groupedKeys.others`
- `Control` 使用订阅条目计算结果与新的摘要计算结果

### 接口草案
- 不新增对外接口
- 页面内部新增：
  - `activeTab: "control" | "mine" | "watch"`
  - `setActiveTab(tab)`
  - `summaryItems` 或等价计算视图模型
  - 如有必要，为 mine / watch 卡片引入带快照的计算结果，减少模板内重复读取

### 错误与安全
- 保持现有 `busy`、`ensureReady()`、`toast.errorOf()`、输入校验逻辑
- 不改变任何危险操作的权限边界或默认值
- tab 切换不增加额外副作用，避免误触发写操作

### 性能与测试策略
- 性能：
  - 移除页面级双列主布局，降低中等宽度下的重排压力
  - 通过计算视图模型减少模板内重复读取和条件分支
- 测试：
  - `npm run build`
  - 本地 UI 冒烟：检查 tab 切换、按钮存在性、空态与单列布局
  - 若开发服务器可启动，使用 `chrome-devtools` 观察页面结构

### 可扩展性设计点
- tab 状态采用字面量联合类型，便于后续增加新 tab
- 摘要信息集中为一组视图模型，后续新增字段不需要改动 store
- `Mine` / `Watch` 内容区按语义拆分，后续可独立加入筛选或批量操作

### 结论
- 阻塞：否

## 阶段 3.1：可执行任务清单（Checklist）
- [x] VWP-001：在 `VarPool.vue` 中完成 tab 化单列重构；构建验证受仓库既有问题阻塞，已记录

## 任务详情

### Task ID：VWP-001
- 标题：VarPool 顶部 tab 与单列布局重构
- Owner：主Agent
- Worktree：D:/project/MyFlowHub3/worktrees/MyFlowHub-Win-varpool-tabs-layout
- Plan 路径：D:/project/MyFlowHub3/worktrees/MyFlowHub-Win-varpool-tabs-layout/plan.md
- 当前阶段：3.3 完成
- 任务目标：
  - 在不改协议层和 store 接口的前提下，重构 `VarPool` 页面为 `Control / Mine / Watch` 三 tab 单列布局
- 涉及模块 / 文件：
  - `frontend/src/pages/VarPool.vue`
- Write set：
  - `D:/project/MyFlowHub3/worktrees/MyFlowHub-Win-varpool-tabs-layout/frontend/src/pages/VarPool.vue`
- 禁止修改的文件 / 目录：
  - `frontend/src/stores/varpool.ts`
  - `frontend/src/components/varpool/NodeVarsDialog.vue`
  - 任意 Go 后端文件
- 关键上下文引用：
  - `frontend/src/pages/VarPool.vue`
  - `frontend/src/pages/Flow.vue`
- 验收条件：
  - 新增三个顶部 tab，默认 `Control`
  - 页面级双列布局移除
  - `Control` 包含控制区、摘要信息和 `Active Subscriptions`
  - `Mine` / `Watch` 改为独立 tab，变量一行一个，原有操作保持可用
  - 无协议层 / store 接口改动
- 测试点：
  - `npm run build`
  - UI 冒烟检查：tab 切换、空态、按钮存在、单列布局
- 验证结果：
  - `npm ci`：通过
  - `npm run build`：失败，阻塞于仓库现有 `frontend/wailsjs` 生成物缺失，报错点为 `Home.vue` 中 `../../wailsjs/go/session/SessionService`
  - `GOWORK=off wails build -platform windows/amd64`：失败，阻塞于仓库现有 `internal/services/flow/service.go` 对 `protocolexec.CapQuery*` 符号的编译错误
  - 结论：本次页面改动已完成，完整构建验证受既有仓库问题阻塞
- 回滚点：
  - 单文件回滚到变更前的 `frontend/src/pages/VarPool.vue`
- 依赖关系：
  - 无跨任务依赖；依赖阶段 1 / 2 结论已确认
- 风险与注意事项：
  - 避免在模板中遗失现有按钮入口
  - 保持 watch 区的强操作按钮清晰分组
  - 避免新增不必要的网络请求

## 并行性评估
- 当前任务的核心写集完全收敛在 `frontend/src/pages/VarPool.vue`
- 本会话未获得用户明确授权进行子Agent委派
- 结论：本轮由主Agent串行完成实现、验证和后续审查，避免单文件写集冲突

## 当前结论
- 阻塞：否
- 当前 workflow 已完成，待用户确认是否结束本次 workflow
