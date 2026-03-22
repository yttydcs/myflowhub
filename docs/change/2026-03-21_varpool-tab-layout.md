# VarPool Tab 化单列布局重构

## 变更背景 / 目标
- 现有 `VarPool` 页面采用顶部双列控制区加下方混合列表的布局，在中等宽度窗口下响应式压力较大，信息层级也不够清晰。
- 本次变更目标是让 `VarPool` 页面在交互结构上向 `Flow` 靠拢，引入顶部 tab，拆分为 `Control`、`Mine`、`Watch` 三个语义明确的视图，并尽量收敛为单列内容流。

## 具体变更内容

### 新增
- 在 `frontend/src/pages/VarPool.vue` 新增顶部 tab 状态：
  - `Control`
  - `Mine`
  - `Watch`
- 新增页面级摘要视图模型：
  - `summaryItems`
  - `mineEntries`
  - `watchEntries`
  - `subscribedEntries`

### 修改
- 将页面主结构从双列主布局改为单列分段布局
- `Control` tab 承载：
  - `Target Node ID`
  - `Refresh All`
  - `Save Watch List`
  - 连接状态
  - `Connected`、`NodeID`、`HubID`、`Cached Keys`、`Mine Count`、`Watch Count`、`Subscribed Count`、`Last Frame`
  - `Active Subscriptions`
- `Mine` tab 承载原 `My Variables` 列表，保持 `Add Variable / Refresh / Edit / Revoke / Remove`
- `Watch` tab 承载原 `Watched Variables` 列表，保持 `Node Vars / Add Watch / Reload Saved / Refresh / Edit / Revoke / Remove / Subscribe`
- 变量列表统一调整为一行一个的单列卡片流

### 删除
- 删除 `VarPool` 页面原有的页面级双列主结构

## 对应 plan.md 任务映射
- `VWP-001`：VarPool 顶部 tab 与单列布局重构

## 关键设计决策与权衡
- 仅修改 `frontend/src/pages/VarPool.vue`
  - 原因：本次属于页面语义与版式重构，优先控制写集，避免同时触碰 store、对话框组件和后端接口
- 复用 `Flow` 的 pill tab 交互风格
  - 原因：保持 Win 前端内部一致性，降低用户切换模块时的认知成本
- 引入 `mineEntries / watchEntries / subscribedEntries`
  - 原因：将模板中的多次 `valueForKey()` 访问收敛为视图模型，提升模板可读性，并减少重复条件判断
- 不新增搜索、过滤、分页
  - 原因：本轮目标是结构重排和响应式减压，避免需求扩散

## 测试与验证方式 / 结果
- `npm ci`
  - 结果：通过
- `npm run build`
  - 结果：失败
  - 原因：仓库当前缺失 `frontend/wailsjs` 生成物，`Home.vue` 无法解析 `../../wailsjs/go/session/SessionService`
- `GOWORK=off wails build -platform windows/amd64`
  - 结果：失败
  - 原因：仓库当前 `internal/services/flow/service.go` 存在 `protocolexec.CapQuery*` 相关未定义符号
- UI 冒烟验证
  - 结果：未执行
  - 原因：受上述既有仓库构建问题阻塞，无法稳定启动可验证界面

## 潜在影响与回滚方案
- 潜在影响：
  - `VarPool` 页面信息组织方式变化，用户需要从混合列表适应为 tab 视图
  - `Control` tab 集中承载状态信息，若未来继续增加摘要字段，需要关注内容堆叠
- 回滚方案：
  - 回滚 `frontend/src/pages/VarPool.vue` 到本次变更前版本即可恢复旧布局

## 子Agent执行轨迹
- VWP-001 → 主Agent（本轮未使用子Agent） → D:/project/MyFlowHub3/worktrees/MyFlowHub-Win-varpool-tabs-layout → `frontend/src/pages/VarPool.vue`、`plan.md`、`docs/change/2026-03-21_varpool-tab-layout.md` → 页面重构完成，完整构建验证受仓库既有问题阻塞
