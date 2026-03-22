# Plan - MyFlowHub-Win：Showcase 界面进一步精简

## 项目目标与当前状态

### 背景
- 仓库：`MyFlowHub-Win`
- 目标：在上一轮 `Showcase Center + 独立编辑窗口` 基础上继续做减法，进一步压缩编辑页与列表页的视觉复杂度。

### 当前状态
- `frontend/src/pages/Showcase.vue`
  - 已是独立编辑窗口页。
  - 当前仍保留左侧 `Screen` 与 `Widget Outline` 两个信息区块。
  - 预览区与布局控制区在右侧，整体仍是双栏结构。
- `frontend/src/pages/ShowcaseCenter.vue`
  - 当前 screen 列表是两段式信息布局。
  - 每个条目展示：
    - 名称
    - layout badge
    - widget 数量
    - `Updated xxx`
    - `screen_id`
    - 操作按钮

### 本次用户确认的调整方向
- 编辑界面：
  - 不要 `Outline`
  - 不要 `Screen`
  - 所有元素尽可能简化
  - 保持“编辑界面”本身，不回退为只读页
- 列表界面：
  - 优先一行展示
  - 不显示 `screen id`
  - 时间只显示时间值，不带 `Updated` 文案

---

## Workflow 信息
- Repo：`d:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch：`refactor/showcase-ui-simplify`
- Base：`main`
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-showcase-ui-simplify`
- Plan Path：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-showcase-ui-simplify\plan.md`
- 当前阶段：`3.1`
- 规范：
  - `d:\project\MyFlowHub3\guide.md`
  - `d:\project\MyFlowHub3\AGENTS.md`

### 主仓状态说明
- `repo/MyFlowHub-Win` 当前存在用户未提交改动：
  - `go.mod`
- 本 workflow 不触碰该文件。
- 本 workflow 的实现性改动仅发生在当前 worktree。

---

## 1. 需求分析

### 目标
- 继续降低 Showcase 的认知负担。
- 让编辑器只保留“编辑必需项”，去掉辅助性信息面板。
- 让列表页更接近“紧凑管理表”，减少噪音字段。

### 范围
- 必须：
  - 编辑页移除 `Screen` 区块
  - 编辑页移除 `Widget Outline` 区块
  - 编辑页改为更直接的单列编辑界面
  - 保留现有编辑能力：
    - Save / Revert
    - Open Viewer
    - Add Event / Add Var
    - layout 调整
    - widget 直接运行交互
  - 列表页压缩成单行优先布局
  - 列表页移除 `screen_id`
  - 列表页时间字段仅显示时间值，不加 `Updated`
- 可选：
  - 在编辑页头部合并少量必要元信息
  - 列表页在窄屏下允许自然换行，但桌面宽度优先单行
- 不做：
  - 后端 schema / store 行为调整
  - Viewer 功能调整
  - 新增筛选、搜索、排序、分页
  - 重新设计创建 / 重命名弹窗流程

### 使用场景
- 用户从 `Showcase Center` 快速浏览 screen，并直接执行编辑、查看、复制、删除。
- 用户在独立编辑窗口中专注修改布局和 widget，而不是被信息面板分散注意力。
- 用户在桌面宽度下能快速横向扫描列表信息。

### 功能需求
- 编辑页
  - 头部仍需可识别当前 screen
  - 必要操作按钮保留
  - layout 控制区保留
  - 预览区保留
  - 不再展示单独的 `Screen` 卡片和 `Widget Outline` 卡片
- 列表页
  - 每条 screen 优先一行展示：
    - 名称
    - layout
    - widget 数
    - 时间
    - 操作按钮
  - 不显示 `screen_id`
  - 时间文案不再以 `Updated` 前缀提示

### 非功能需求
- 性能：
  - 仅调整视图结构，不引入新的数据层开销
  - 不新增不必要的 watch、load、save、订阅逻辑
- 可读性：
  - 保持模板层级更浅
  - 减少重复容器与冗余提示文案
- 可扩展性：
  - 不破坏现有 store API
  - 仍能为后续再次加回其他摘要信息保留位置
- 一致性：
  - 维持现有视觉语言，不做风格跳变

### 输入输出
- 输入：
  - 现有 `showcase` store 提供的 screen 摘要与当前 screen 草稿
- 输出：
  - 更紧凑的 `ShowcaseCenter` 列表界面
  - 更纯粹的 `Showcase` 编辑界面

### 边界异常
- screen 不存在时，编辑页仍要明确显示 `Screen not found`
- 列表为空时，现有空态不应被破坏
- 窄屏下列表允许换行，但桌面优先单行
- 未保存草稿提示逻辑必须保留

### 验收标准
- 编辑页不再出现 `Screen` 区块
- 编辑页不再出现 `Widget Outline` 区块
- 编辑页仍可完成当前已有的核心编辑动作
- 列表页不再显示 `screen_id`
- 列表页时间字段不再显示 `Updated`
- 桌面宽度下单条列表优先单行呈现

### 风险
- 过度简化可能让 screen 名称编辑入口变得不明显，需要在头部或布局卡片里保留清晰入口
- 去掉 widget outline 后，用户缺少快速点击某个 widget 的旁路入口，因此不能移除预览区中的右键/编辑入口

### 结论
- 阻塞：否
- 可进入下一阶段：`2. 架构设计`

---

## 2. 架构设计（分析）

### 总体方案
- 编辑页：
  - 从“双栏信息 + 预览”改为“头部工具条 + 单主体编辑区”
  - 将 screen 名称输入、少量状态信息、layout 控制压缩进主编辑区顶部
  - 预览区继续作为主要交互面
- 列表页：
  - 从“名称区 + 次信息区 + id 行”改为“单行摘要 + 操作区”
  - 只保留用户当前明确需要的信息

### 选型理由 / 备选对比
- 方案 A：完全移除左栏，改成单列
  - 优点：最符合“尽可能简化”
  - 缺点：需要把 screen 名称和元信息重新安置
- 方案 B：保留左栏但弱化样式
  - 优点：改动小
  - 缺点：本质上仍保留用户明确不想要的 `Screen` / `Outline`
- 结论：
  - 采用方案 A

### 模块职责
- `frontend/src/pages/Showcase.vue`
  - 负责编辑页结构精简
  - 不改动 store 行为
- `frontend/src/pages/ShowcaseCenter.vue`
  - 负责列表项结构压缩
  - 不改动操作行为
- `frontend/src/stores/showcase.ts`
  - 本轮只作为现有数据来源，不做接口改动

### 数据 / 调用流
- 编辑页：
  - 继续使用现有 `screenNameDraft`、`layoutForm`、`dirty`、`saveDraft`、`revertDraft`
  - 只改变控件布局，不改变保存流程
- 列表页：
  - 继续使用 `listScreenSummaries()`
  - 只改变展示结构，不改变打开窗口和 CRUD 动作

### 接口草案
- 无新增前后端接口
- 无新增 store API

### 错误与安全
- 继续保留：
  - `screenMissing` 明确提示
  - 编辑页保存前 `screen name` 校验
  - `beforeunload` 的 dirty 提示

### 性能与测试策略
- 性能：
  - 本轮应为纯视图层收敛，不增加额外数据处理
- 测试：
  - Vue SFC 语法检查
  - 手工检查：
    - 编辑页无 `Screen` / `Widget Outline`
    - 列表页无 `screen_id`
    - 列表页时间文案变化正确
    - 编辑页保存、回退、打开 viewer、添加 widget 仍可触发

### 可扩展性设计点
- 编辑页压缩后，若后续要恢复更多摘要信息，可优先放入头部次级信息行，而不是重新引入左栏
- 列表页保持“信息摘要区 + 操作区”的二段结构，后续增加筛选或标签时仍可扩展

### 结论
- 阻塞：否
- 可进入下一阶段：`3.1 计划拆分`

---

## 3.1 计划拆分（Checklist）

### 并行性评估
- 当前任务主要集中在两个页面：
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/pages/ShowcaseCenter.vue`
- 理论上可拆分，但本轮改动规模较小，且需要统一把握“简化程度”和视觉一致性。
- 本次不使用子Agent：
  - 原因：任务高度耦合于同一套 UI 取舍，拆分价值低；且当前会话未获得显式的子Agent授权。

### SS-01 - 简化 Showcase Center 列表
- Owner：主Agent
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-showcase-ui-simplify`
- Plan：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-showcase-ui-simplify\plan.md`
- Write set：
  - `frontend/src/pages/ShowcaseCenter.vue`
- 关键上下文引用：
  - `frontend/src/pages/ShowcaseCenter.vue`
  - `frontend/src/stores/showcase.ts`
- 目标：
  - 将 screen 列表压缩为单行优先布局
  - 移除 `screen_id`
  - 时间仅显示值，不显示 `Updated`
- 验收条件：
  - 桌面宽度下单条 screen 优先单行
  - 不显示 `screen_id`
  - 时间文案无 `Updated`
  - 操作按钮保持可用
- 测试点：
  - 手工检查列表展示与按钮行为
- 回滚点：
  - 回滚 `frontend/src/pages/ShowcaseCenter.vue`

### SS-02 - 简化 Showcase 编辑页
- Owner：主Agent
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-showcase-ui-simplify`
- Plan：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-showcase-ui-simplify\plan.md`
- Write set：
  - `frontend/src/pages/Showcase.vue`
- 关键上下文引用：
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/stores/showcase.ts`
- 目标：
  - 移除 `Screen` 区块
  - 移除 `Widget Outline` 区块
  - 保留必要工具栏、screen name 编辑、layout 控制与预览
- 验收条件：
  - 页面不再出现上述两个区块
  - 保存 / 回退 / 打开 Viewer / Add Event / Add Var 仍可操作
  - 两种 layout 预览仍可渲染
- 测试点：
  - 手工检查编辑页结构与关键按钮
  - SFC 语法检查
- 回滚点：
  - 回滚 `frontend/src/pages/Showcase.vue`

### SS-03 - 回归验证与归档
- Owner：主Agent
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-showcase-ui-simplify`
- Plan：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-showcase-ui-simplify\plan.md`
- Write set：
  - `plan.md`
  - `docs/change/*`
- 关键上下文引用：
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/pages/ShowcaseCenter.vue`
- 目标：
  - 做静态校验
  - 形成 review 结论
  - 归档变更
- 验收条件：
  - 至少完成 Vue SFC 语法检查
  - review 结论明确
  - `docs/change` 文档完整
- 测试点：
  - Vue SFC parse
- 回滚点：
  - 回滚本轮 UI 改动

---

## 3.3 Code Review 结论
- 需求覆盖：通过
  - 编辑页已移除 `Screen` 与 `Widget Outline`
  - 列表页已移除 `screen_id`
  - 列表页时间字段不再显示 `Updated`
- 架构合理性：通过
  - 仅调整 `Showcase.vue` 与 `ShowcaseCenter.vue` 的模板结构，不引入新的 store 或路由耦合
- 性能风险：通过
  - 本轮为纯 UI 减法，没有新增额外 I/O、watch 或订阅刷新
- 可读性与一致性：通过
  - 编辑页结构更浅，列表页条目更紧凑，视觉噪音减少
- 可扩展性与配置化：通过
  - store API 未变，后续仍可在头部或列表摘要区增减信息
- 稳定性与安全：通过
  - 现有 `screenMissing`、dirty 提示、保存前校验逻辑未回退
- 测试覆盖情况：通过（附带环境阻塞说明）
  - Vue SFC 解析通过
  - `npm run build` 仍因 worktree 缺少 `wailsjs/go/main/App` 绑定而失败，不是本轮模板改动导致
- 子Agent治理与审计：通过
  - 本轮未使用子Agent；原因已在并行性评估中记录

---

## 当前结论
- 阶段 `4`：归档中
- 阻塞：否
- 下一步：写入 `docs/change` 归档文档
