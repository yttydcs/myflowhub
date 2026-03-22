# 2026-03-21 Win Flow Method Capability Query Node

## 变更背景 / 目标
- 背景：
  - Win Flow 编辑器的方法能力对话框只能查询当前 executor，无法临时指定查询节点。
  - popup/editor 窗口在未收到实时 session 事件时，能力查询会直接报 `Login required to send Flow requests.`，即使 Home 页面已经持久化了身份。
- 目标：
  - 在方法能力对话框中增加临时 `query node id` 输入。
  - 让该 `query node id` 仅参与能力查询，不写回当前 call 节点配置。
  - 为 Flow 编辑器窗口补充 `HomeState` 身份兜底，降低误报登录缺失的概率。

## 具体变更内容（新增 / 修改 / 删除）

### 修改
- `frontend/src/windows/FlowEditorWindow.vue`
  - 引入 `LoadHomeState()`，在 popup/editor 挂载时加载持久化 `nodeId/hubId` 作为 fallback identity。
  - 新增对话框本地状态：
    - `queryNodeIdDraft`
    - `lastCapabilityQueryNode`
    - `fallbackIdentity`
  - 方法对话框新增 `Query Node ID` 输入框与状态标签。
  - 打开对话框时默认把查询 node 初始化为：
    - 当前节点显式远端 `target`
    - 否则当前 executor / hub
  - 刷新能力时调用 `flowStore.queryExecCapabilities(undefined, queryNodeIdDraft)`。
  - 交互提示明确说明：查询 node 仅用于 lookup，不会写回 call 节点。

- `frontend/src/stores/flow.ts`
  - 新增 `resolveCapabilityQueryNode(...)`，用于解析并校验临时查询目标。
  - `queryExecCapabilities(...)` 扩展为可接收 `queryNodeId?: string | number`。
  - override 仅用于 `ExecCapQuerySimple` 的目标节点，不写入 `state.targetId`。
  - 能力查询成功提示包含实际查询的 node id，便于用户确认。

### 新增
- `docs/change/2026-03-21_win-flow-method-capability-query-node.md`

### 删除
- 无

## 对应 `plan.md` 任务映射
- `FLOW-CAP-QUERY-1`
  - `frontend/src/windows/FlowEditorWindow.vue`
- `FLOW-CAP-QUERY-2`
  - `frontend/src/stores/flow.ts`
- `FLOW-CAP-QUERY-3`
  - `plan.md`
  - `docs/change/2026-03-21_win-flow-method-capability-query-node.md`
  - 验证记录
  - Code Review 结论

## 关键设计决策与权衡（尤其性能 / 扩展性）
- 决策：把“查询目标”与“运行时 target”彻底分离。
  - 原因：查询 node 只是编辑辅助信息，不能污染实际执行图，否则会把一次性的 lookup 行为误持久化。
  - 权衡：增加一个对话框内临时状态，但保持 graph/spec 语义稳定。

- 决策：popup/editor 使用 `HomeState` 做身份兜底，而不是修改全局 `sessionStore` 初始化逻辑。
  - 原因：本次问题只出现在独立窗口；局部 fallback 更小、更可控。
  - 权衡：避免扩大改动面；后续若多个窗口都有同类问题，再抽公共能力。

- 决策：查询 node 输入做严格正整数校验。
  - 原因：避免 `parseInt("12abc")` 之类的宽松解析把错误输入静默吞掉。
  - 性能：仅增加一次常量级字符串校验，不引入额外 I/O。

## 测试与验证方式 / 结果
- `git diff --check`
  - 结果：通过
  - 说明：仅有 Windows 行尾提示，不影响 diff 完整性

- `cd frontend && npm run build`
  - 结果：失败
  - 原因：当前工作区不存在 `frontend/node_modules`，`vite` 不可执行
  - 报错：`'vite' is not recognized as an internal or external command`

- 浏览器级联调
  - 结果：未执行
  - 原因：前端依赖未安装，无法拉起可用的 Vite 页面进行 Chrome DevTools 冒烟

## Code Review（3.3）
- 需求覆盖：通过
  - 方法对话框已支持临时 `query node id`
  - 查询 node 不会写回当前 call 节点
  - popup/editor 增加了持久化身份兜底

- 架构合理性：通过
  - 查询 override 收敛在 `flow.ts` 的查询入口，未扩散到 graph/model 层
  - UI fallback 仅作用于 Flow 编辑窗口，边界清晰

- 性能风险：通过
  - 仅增加本地字符串解析和一次 `HomeState` 读取
  - 未新增自动轮询或重复查询

- 可读性与一致性：通过
  - 命名与现有窗口模式一致，`ShowcaseWindow` 的 fallback 思路得到复用
  - 对话框文案明确区分了查询用途与回填用途

- 可扩展性与配置化：通过
  - 后续若需要“按查询 node 返回 schema/参数模板”，可继续挂在对话框临时态，不需要调整图数据结构

- 稳定性与安全：通过
  - 非法 query node 输入会显式失败
  - 未放宽登录/身份校验，只是补上本地持久化 fallback

- 测试覆盖情况：部分通过
  - 已完成 diff 级检查与静态代码复核
  - 受缺少前端依赖影响，未完成构建和浏览器冒烟

- 子Agent治理与审计：通过
  - 本次未使用子Agent
  - 原因：`FlowEditorWindow.vue` 与 `flow.ts` 写集紧耦合，由主Agent连续实现和复核更安全

## 潜在影响与回滚方案

### 潜在影响
- popup/editor 会在挂载时额外读取一次 `HomeState`。
- 能力查询成功 toast 文案会包含查询 node id。
- 若 `HomeState` 本身没有保存身份，查询仍会按原逻辑失败，这是预期行为。

### 回滚方案
- 回退以下文件即可恢复本轮变更：
  - `frontend/src/windows/FlowEditorWindow.vue`
  - `frontend/src/stores/flow.ts`
  - `plan.md`
  - `docs/change/2026-03-21_win-flow-method-capability-query-node.md`

## 子Agent执行轨迹（Task ID → Agent → Worktree → 文件 → 验收结果）
- 本次未使用子Agent
  - Task ID：`FLOW-CAP-QUERY-1` / `FLOW-CAP-QUERY-2` / `FLOW-CAP-QUERY-3`
  - Agent：主Agent
  - Worktree：`D:/project/MyFlowHub3/worktrees/refactor-win-method-cap-query-node/MyFlowHub-Win`
  - 文件：
    - `frontend/src/windows/FlowEditorWindow.vue`
    - `frontend/src/stores/flow.ts`
    - `plan.md`
    - `docs/change/2026-03-21_win-flow-method-capability-query-node.md`
  - 验收结果：已完成实现、Review 与归档；待用户确认是否结束 workflow
