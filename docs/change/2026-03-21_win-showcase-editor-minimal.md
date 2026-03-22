# 2026-03-21 - Showcase 编辑窗口进一步极简化

## 变更背景 / 目标
- 背景：上一轮已完成 Showcase 列表页与独立编辑窗口拆分，但编辑窗口仍保留 `Screen Name` 与常驻布局表单，信息密度偏高。
- 目标：
  - 移除编辑窗口中的 `Screen Name` 编辑；
  - 将布局编辑改为顶部按钮触发的弹窗；
  - 让 `showcaseEditorWindow` 像 Flow 编辑器一样使用 full-bleed 窗口布局，避免页面级滚动；
  - 保留现有 widget 直接交互、布局模式切换、保存 / 回退能力。

## 具体变更内容
### 修改
- `frontend/src/pages/Showcase.vue`
  - 移除 `screenNameDraft` 与主界面中的 `Screen Name` 输入。
  - 新增 `layoutDialogOpen`、`openLayoutDialog`、`closeLayoutDialog`、`applyLayoutDialog`。
  - 保留原 `layoutForm` 与 `saveScreenLayout` 校验逻辑，但改为返回成功 / 失败结果，供弹窗控制关闭行为。
  - 重构页面结构为 `header + flex-1 editor surface`。
  - 顶部工具栏新增 `Layout` 按钮，并在主编辑区只保留编辑说明、布局摘要与预览面板。
  - columns 预览区与 canvas 预览区都改为吃满剩余高度，移除原 `Screen Name` 卡片和常驻布局表单。
- `frontend/src/layout/AppShell.vue`
  - 将 full-bleed window 判断从 route name 硬编码改为读取 `route.meta.windowMode`。
- `frontend/src/router/index.ts`
  - 为 `flowEditorWindow` 和 `showcaseEditorWindow` 增加 `windowMode: "full-bleed"`。

### 删除
- `frontend/src/pages/Showcase.vue` 主界面的 `Screen Name` 输入区。
- `frontend/src/pages/Showcase.vue` 主界面的常驻 `Layout Mode / Max Columns / Min Column Width / Apply Layout` 表单区。

## 对应 plan.md 任务映射
- `T1` 明确本轮需求边界与验收标准：完成。
- `T2` 完成架构设计与交互方案：完成。
- `T3` 实现编辑器极简化改造：完成。
- `T4` 进行 Code Review 与归档：完成。

## 关键设计决策与权衡
- 采用 `route.meta.windowMode` 而不是继续在 `AppShell` 中硬编码 route name。
  - 原因：窗口壳层策略属于路由层能力，不应耦合具体业务页名称。
  - 收益：后续其它独立编辑窗口也可复用 full-bleed 模式。
- 布局编辑下沉为弹窗，而不是保留在主界面。
  - 原因：布局设置是低频动作，常驻会持续占用视觉和空间预算。
  - 收益：主界面聚焦在 widget 直接编辑。
- 为满足“不要有滚动”，columns 预览区使用固定编辑高度 + `overflow-hidden`。
  - 权衡：极端多 widget 场景下内容会更拥挤甚至被裁切，但可以换取单屏编辑的一致体验。

## 测试与验证方式 / 结果
- 前端依赖安装：
  - 命令：`npm ci`
  - 结果：通过。
- Vue SFC 语法解析：
  - 命令：使用 `@vue/compiler-sfc` 解析 `frontend/src/pages/Showcase.vue` 与 `frontend/src/layout/AppShell.vue`
  - 结果：通过。
- 全量前端构建：
  - 命令：`npm run build`
  - 结果：失败。
  - 原因：现有 Wails 生成绑定缺失，失败点为 `src/pages/Home.vue` 依赖 `../../wailsjs/go/session/SessionService` 无法解析；不是本次改动引入。
- 关键静态检查：
  - `Showcase.vue` 中已无 `Screen Name` / `screenNameDraft` 主界面逻辑。
  - `showcaseEditorWindow` 已标记 `windowMode: "full-bleed"`。

## 潜在影响与回滚方案
### 潜在影响
- columns 模式在 widget 数量过多时，为了避免滚动会更拥挤。
- full-bleed window 逻辑现在依赖 route meta；若未来新增编辑器窗口忘记标注 `windowMode`，会退回普通 window 布局。

### 回滚方案
- 回滚 `frontend/src/pages/Showcase.vue`，恢复主界面常驻布局表单与名称输入。
- 回滚 `frontend/src/layout/AppShell.vue` 与 `frontend/src/router/index.ts`，恢复 route name 硬编码的 full-bleed 逻辑。

## 子Agent执行轨迹
| Task ID | Agent | Worktree | 文件 | 验收结果 |
| --- | --- | --- | --- | --- |
| T1 | 主Agent | `D:\\project\\MyFlowHub3\\worktrees\\MyFlowHub-Win-showcase-editor-minimal` | `plan.md` | 通过 |
| T2 | 主Agent | `D:\\project\\MyFlowHub3\\worktrees\\MyFlowHub-Win-showcase-editor-minimal` | `plan.md` | 通过 |
| T3 | 主Agent | `D:\\project\\MyFlowHub3\\worktrees\\MyFlowHub-Win-showcase-editor-minimal` | `frontend/src/pages/Showcase.vue`、`frontend/src/layout/AppShell.vue`、`frontend/src/router/index.ts` | 通过 |
| T4 | 主Agent | `D:\\project\\MyFlowHub3\\worktrees\\MyFlowHub-Win-showcase-editor-minimal` | `plan.md`、`docs/change/2026-03-21_win-showcase-editor-minimal.md` | 通过 |

## 结论
- 本轮目标已完成，可进入 workflow 结束确认。
