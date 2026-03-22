# Plan - MyFlowHub-Win Flow 编辑器抽屉与方法选择器收敛

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`refactor/win-method-selector-dialog`
- Base：`main`
- Worktree：`D:\project\MyFlowHub3\worktrees\refactor-win-method-selector-dialog\MyFlowHub-Win`
- 当前阶段：`4 归档变更`

## 项目目标与当前状态

### 目标
- 修复 Flow 编辑器右侧节点详情抽屉的半透明问题，改为不透明展示。
- 为 `call` 节点引入独立“方法选择”对话框，替代直接在抽屉中手填 method 的主路径。
- 保留 `target` 在数据层中的运行时语义，但从主编辑表单中隐藏，由方法选择对话框统一维护。
- 选择方法后自动回填：
  - `method`
  - `target`
  - `args` 本次不自动生成模板；保留原值，若原值为空则保持 `{}`

### 当前事实
- `frontend/src/windows/FlowEditorWindow.vue`
  - 右侧抽屉使用 `Overlay`，当前遮罩和面板都带透明度。
  - `call` 节点仍直接暴露 `Target Node / Method / Args` 手工编辑表单。
- `frontend/src/stores/flow.ts`
  - 已有 `queryExecCapabilities(...)` 与 `applyCallCapability(...)`。
  - 当前能力应用会回填 `target=providerNode` 与 `method`。
  - `buildSpec(...)` 中 `target > 0` 会真实写入 graph spec，`target <= 0` 则不写入。
- 协议文档显示：
  - `call.target_node` 仍有运行时语义。
  - `cap_query_resp.routes[]` 当前稳定字段为 `provider_node / via_node / method / version`。
  - 当前未确认存在可直接用于前端自动回填 `args` 模板的稳定字段。

### 已确认决策
- 抽屉改为不透明。
- `call` 节点的方法选择改为单独对话框。
- 本次仅稳定回填 `target + method`。
- `args` 保留原值；若为空则保持 `{}`。
- UI 中不再暴露 `target` 手工输入框，但底层保留 `target` 字段以兼容未来远程调用。

## 需求分析输出

### 目标
- 收敛 Flow 编辑器中 `call` 节点的配置入口，减少用户直接操作 `target/method` 的心智负担。

### 范围
- 必须：
  - 节点详情抽屉不透明
  - 方法选择独立对话框
  - 对话框可查询能力并回填 `method/target`
  - 主表单隐藏 `target`
- 可选：
  - 对话框内显示 provider/via/version 辅助信息
- 不做：
  - 不改后端协议
  - 不新增 args schema 驱动表单
  - 不修改主 `/flow` 项目中心

### 使用场景
- 用户点击画布节点后打开右侧抽屉。
- 用户点击抽屉中的 `Select Method` 打开对话框。
- 用户在对话框中刷新能力、搜索并选择方法。
- 系统自动回填 `method/target`；用户继续编辑 `args/retry/timeout` 并保存 graph。

### 功能需求
- 抽屉遮罩与面板均不透明。
- 方法选择器对话框支持：
  - 打开/关闭
  - 能力刷新
  - 文本筛选
  - 显示 provider/via/version
  - 应用到当前选中节点
- 抽屉中保留 `method` 只读展示或摘要展示，避免主路径重新退回手工输入。

### 非功能需求
- 不引入额外协议依赖。
- 失败路径必须 toast 可见。
- 不允许因为对话框交互破坏当前历史记录、保存和画布选择逻辑。

### 输入输出
- 输入：
  - 当前节点 `method / target / args`
  - 会话身份 `selfNodeId / hubId`
  - `ExecCapQuerySimple` 返回的能力列表
- 输出：
  - 更新后的节点 `method / target / args`
  - 用户可见的提示信息

### 边界异常
- 未登录或缺失 hubId，无法查询能力
- 能力列表为空
- 当前节点未选中
- 旧数据存在 `target > 0`
- `args` 非法 JSON 时保存失败

### 验收标准
- 抽屉视觉上不透明。
- 抽屉中不再提供 `target` 手工输入。
- `Select Method` 对话框可打开并查询能力。
- 选择能力后，节点 `method` 被正确回填。
- 选择能力后，节点 `target` 被正确回填：
  - provider 为当前 executor/self 时可置 `0`
  - 其它 provider 保留对应 nodeId
- `args` 不被错误清空。

### 风险
- 当前协议未提供稳定 args schema；本次只能做 `target/method` 级别自动回填。
- 若 provider 等于 executor/self，如何规范化 `target` 需在前端统一处理，避免保存出多余目标。

### 阻塞
- 否

## 架构设计输出

### 总体方案
- 延续现有 `FlowEditorWindow + flow store` 架构，不新增页面级 store。
- 由 `flow.ts` 负责：
  - 能力查询
  - 选择结果规范化
  - `target` 的“self/local vs remote”判定
- 由 `FlowEditorWindow.vue` 负责：
  - 抽屉样式收敛
  - 方法选择对话框 UI
  - 对话框内筛选与应用交互

### 选型理由 / 备选对比
- 采用“在现有编辑器窗口内新增 Overlay 对话框”：
  - 优点：最小改动、状态局部、复用既有 `Overlay` 和 toast 机制
  - 备选：拆出独立组件
  - 未选原因：本次交互范围小，单文件维护成本更低，避免过早抽象
- 采用“能力查询后前端筛选”：
  - 优点：延续当前 `queryExecCapabilities` 语义，不增加新接口
  - 代价：列表较大时需要前端过滤；当前规模可接受

### 模块职责
- `frontend/src/stores/flow.ts`
  - 统一管理 graph/node state
  - 提供能力查询与能力应用
  - 规范化 `providerNode === executorNode` 时的 target 写法
- `frontend/src/windows/FlowEditorWindow.vue`
  - 节点详情抽屉
  - 方法选择对话框
  - 查询/筛选/应用交互
- `frontend/src/components/ui/overlay/Overlay.vue`
  - 继续承担弹层承载；本次不改组件实现，只调整调用样式

### 数据 / 调用流
- 打开方法选择器：
  - 用户点击 `Select Method`
  - 对话框打开，读取当前节点 method 作为搜索初值
- 刷新能力：
  - 调用 `flowStore.queryExecCapabilities(searchText?)`
  - store 更新 `execCapabilities`
- 选择能力：
  - `FlowEditorWindow.vue` 调用 `flowStore.applyCallCapability(...)`
  - store 根据 route 与当前 executor 规范化 `target`
  - 回填 `method/target`
  - `args` 保留原值
- 保存 graph：
  - 仍使用 `flowStore.exportGraphDraft()`
  - `buildSpec(...)` 根据 `target` 是否大于 0 决定是否写入 spec.target

### 接口草案
- 继续复用：
  - `flowStore.queryExecCapabilities(methodFilter?)`
  - `flowStore.applyCallCapability(key)`
- 本次新增/调整：
  - `flowStore.normalizeCallTarget(providerNode)` 或等价内部逻辑
  - `FlowEditorWindow` 本地状态：
    - `methodDialogOpen`
    - `methodSearch`
    - `pendingCapabilityKey`

### 错误与安全
- 查询前校验登录态与 hubId。
- 无能力或未选中能力时不给应用。
- 所有失败路径保留 toast。
- 不放宽现有 `args` JSON 校验。

### 性能与测试策略
- 能力查询显式触发，不在每次节点切换时自动请求，避免重复 I/O。
- 对话框内过滤使用本地 `computed`。
- 测试策略：
  - `vue-tsc` 定向检查变更文件
  - 如环境允许，执行前端 build
  - 手工冒烟：
    - 抽屉不透明
    - 方法对话框打开/关闭
    - 刷新能力
    - 选择方法后回填
    - 保存 graph

### 可扩展性设计点
- 后续如果 `cap_query_resp` 提供 schema/template，可在方法对话框中无缝补入 args 模板预览与应用。
- `target` 虽从 UI 隐藏，但数据层保留，未来仍可支持远程 provider 调用。

### 阻塞
- 否

## 可执行任务清单（Checklist）

- [x] `FLOW-METHOD-1` 修复节点详情抽屉视觉不透明
  - Owner：主Agent
  - Worktree：`D:\project\MyFlowHub3\worktrees\refactor-win-method-selector-dialog\MyFlowHub-Win`
  - Plan：`D:\project\MyFlowHub3\worktrees\refactor-win-method-selector-dialog\MyFlowHub-Win\plan.md`
  - 目标：
    - 抽屉遮罩与面板改为不透明视觉
  - 涉及文件：
    - `frontend/src/windows/FlowEditorWindow.vue`
  - Write set：
    - `frontend/src/windows/FlowEditorWindow.vue`
  - 验收条件：
    - 抽屉打开后背景被实色遮罩覆盖
    - 抽屉面板本身不再透底
  - 测试点：
    - 打开节点详情
    - 点击空白关闭
  - 回滚点：
    - 回退 `FlowEditorWindow.vue` 样式修改

- [x] `FLOW-METHOD-2` 收敛 call 节点方法选择为独立对话框
  - Owner：主Agent
  - Worktree：`D:\project\MyFlowHub3\worktrees\refactor-win-method-selector-dialog\MyFlowHub-Win`
  - Plan：`D:\project\MyFlowHub3\worktrees\refactor-win-method-selector-dialog\MyFlowHub-Win\plan.md`
  - 目标：
    - 主表单隐藏 `target`
    - 新增 `Select Method` 对话框
    - 查询并选择能力后自动回填 `method/target`
    - `args` 保留原值
  - 涉及文件：
    - `frontend/src/windows/FlowEditorWindow.vue`
    - `frontend/src/stores/flow.ts`
  - Write set：
    - `frontend/src/windows/FlowEditorWindow.vue`
    - `frontend/src/stores/flow.ts`
  - 验收条件：
    - 对话框可打开/关闭
    - 可刷新并筛选能力
    - 选中后 method 回填成功
    - self/executor provider 时 target 规范化为本地调用写法
  - 测试点：
    - 打开方法对话框
    - 空结果提示
    - 应用能力
    - 保存 graph
  - 回滚点：
    - 回退上述两文件

- [x] `FLOW-METHOD-3` 验证、Review 与归档
  - Owner：主Agent
  - Worktree：`D:\project\MyFlowHub3\worktrees\refactor-win-method-selector-dialog\MyFlowHub-Win`
  - Plan：`D:\project\MyFlowHub3\worktrees\refactor-win-method-selector-dialog\MyFlowHub-Win\plan.md`
  - 目标：
    - 完成验证记录
    - 执行 Code Review
    - 生成 `docs/change`
  - 涉及文件：
    - `plan.md`
    - `docs/change/*`
  - Write set：
    - `plan.md`
    - `docs/change/*`
  - 验收条件：
    - 有明确验证结果
    - 有 review 结论
    - 有 change 文档
  - 测试点：
    - `vue-tsc` 定向检查
    - 若可行则 `npm run build`
  - 回滚点：
    - 不适用

## 并行性评估
- 评估结果：本轮不派发子Agent。
- 原因：
  - 当前运行策略要求只有在用户显式要求委派/并行 agent 时才可使用子Agent。
  - 本次 `FlowEditorWindow.vue` 与 `flow.ts` 写集直接耦合，拆分后集成成本高于收益。

## 实施结果
- 已完成：
  - `FlowEditorWindow.vue`
    - 右侧节点详情抽屉改为不透明面板
    - 抽屉主表单移除 `target` 手工输入
    - 新增独立 `Select Capability` 方法选择对话框
    - 对话框支持能力刷新、本地筛选、选中与应用
  - `flow.ts`
    - 新增 `normalizeCallTarget(...)`
    - `applyCallCapability(...)` 改为：
      - provider 等于当前 executor 时写回 `target=0`
      - 其它 provider 保留远端 nodeId
      - `args` 为空时补 `{}`，否则保留原值

## 验证记录
- `frontend/ npm install`
  - 结果：通过
- `frontend/ npx vue-tsc --noEmit --pretty false`
  - 结果：失败
  - 原因：
    - 仓库基线缺失 `wailsjs` 生成物
    - 三方类型依赖缺失（`d3-*`、`radix-vue`、Bluetooth DOM types）
    - 其它页面既有 TS 错误（如 `Presets.vue`、`Showcase.vue`）
  - 结论：
    - 错误列表未落到本次修改的 `FlowEditorWindow.vue / flow.ts`
- `frontend/ npm run build`
  - 结果：失败
  - 原因：仓库基线缺失 `../../wailsjs/go/session/SessionService`
- `git diff --check`
  - 结果：通过
- Chrome DevTools + Vite dev server
  - 结果：失败
  - 原因：页面启动后被 Vite import overlay 阻塞，缺失 `../../wailsjs/runtime/runtime`（`src/stores/presets.ts`），无法进入 Flow 页面继续交互冒烟

## Code Review 结论
- 需求覆盖：通过
  - 抽屉已改为不透明
  - `call` 节点已有独立方法选择对话框
  - `target` 已从主表单隐藏，改由能力选择统一维护
- 架构合理性：通过
  - 变更集中在编辑器窗口与 flow store，未扩散到项目中心或协议层
- 性能风险：通过
  - 能力查询仍为显式触发，无新增自动轮询或高频请求
- 可读性与一致性：通过
  - 对话框状态与节点详情状态分离明确，命名与现有编辑器风格一致
- 可扩展性与配置化：通过
  - 保留 `target` 数据字段，后续可继续支持远程 provider 调用
  - 若未来协议提供 schema/template，可直接扩展方法对话框
- 稳定性与安全：通过
  - 查询/应用失败继续统一 toast 提示
  - 保存时 `args` JSON 校验未被放宽
- 测试覆盖情况：部分通过
  - 已完成依赖安装、类型检查尝试、构建尝试、Vite+DevTools 启动验证
  - 受基线缺失 `wailsjs` 绑定阻塞，无法完成完整 UI 冒烟
- 子Agent治理与审计：通过
  - 本次未使用子Agent；原因已在“并行性评估”中记录

## 当前结论
- 本 workflow 已完成 `1 → 4` 阶段产物，等待用户确认是否结束本次 workflow。
