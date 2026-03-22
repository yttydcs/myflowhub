# Showcase 顶部按钮图标化

## 变更背景 / 目标
- 背景：Showcase 编辑窗口头部已经对齐到单行结构，但 `Save`、`Revert`、`Layout`、`Open Viewer` 仍然使用文字按钮，与 Flow 的图标工具栏风格不一致。
- 目标：将这些按钮改为图标按钮，并通过 tooltip 展示原动作文案，进一步统一头部工具栏风格。

## 具体变更内容
### 修改
- `frontend/src/pages/Showcase.vue`
  - 为 `Save`、`Revert` 分别引入 `Save`、`Undo2` icon。
  - 将 `Save`、`Revert`、`Layout`、`Open Viewer` 改为 `size="icon"` 的图标按钮。
  - 为上述按钮补充 tooltip，并保留对应 `sr-only` 文案，确保可访问性。

### 未修改
- 未修改 `Refresh Vars`、`Add Event`、`Add Var` 的既有图标按钮实现。
- 未修改保存逻辑、回退逻辑、layout 弹窗逻辑、viewer 打开逻辑和任何 store 行为。
- 未对 viewer 窗口新增按钮。

## 对应 plan.md 任务映射
- `T1` 明确顶部按钮图标化边界：完成。
- `T2` 完成实现方案设计：完成。
- `T3` 实现 Showcase 头部按钮图标化：完成。
- `T4` 进行 Code Review 与归档：完成。

## 关键设计决策与权衡
- 采用 Flow 风格的 icon + tooltip 模式，而不是保留文字按钮。
  - 原因：用户明确要求顶部按钮图标化，文字改由 tooltip 展示。
  - 收益：头部更紧凑，视觉语言与 Flow 更一致。
- 保留 `sr-only` 文本，而不是只依赖 tooltip。
  - 原因：tooltip 不等于无障碍标签。
  - 收益：屏幕阅读器仍能正确识别按钮语义。
- viewer 不新增按钮。
  - 原因：当前 viewer 没有对应控制动作，用户也明确说“没有的部分可以忽略”。
  - 收益：范围保持最小，不引入无必要交互。

## 测试与验证方式 / 结果
- 静态搜索：
  - 命令：`rg -n '@click="saveDraft">Save|@click="revertDraft">Revert|Layout\\s*$|Open Viewer\\s*$' frontend/src/pages/Showcase.vue`
  - 结果：无命中，说明头部目标按钮不再以文字正文形式渲染。
- Vue SFC 解析：
  - 命令：使用 `@vue/compiler-sfc` 解析 `src/pages/Showcase.vue`
  - 结果：通过。
- 依赖准备：
  - 命令：`npm ci`
  - 结果：通过。
- 前端构建：
  - 命令：`npm run build`
  - 结果：失败。
  - 失败原因：现有 `src/pages/Home.vue` 无法解析 `../../wailsjs/go/session/SessionService`，属于当前 Wails 生成绑定缺失问题，与本次按钮图标化改动无关。

## 潜在影响与回滚方案
- 潜在影响：
  - 顶部工具栏会更紧凑，按钮含义更依赖 hover tooltip 和 `sr-only` 文本。
  - 键鼠交互与禁用逻辑不变。
- 回滚方案：
  - 回退 `frontend/src/pages/Showcase.vue`

## 子Agent执行轨迹
- 本轮未使用子Agent。
