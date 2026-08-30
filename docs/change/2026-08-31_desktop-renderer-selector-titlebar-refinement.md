# Desktop Renderer Selector 标题栏收敛

## 变更背景 / 目标

Schema-driven Resource Widget 已支持在多个兼容 renderer 之间切换，但原选择器仍占用 Widget 内容区，并显示“显示方式”和 renderer 描述等重复文字。后续视觉验收进一步确认：切换入口应直接并入标题栏，只显示当前 renderer 名称，不用分割线制造额外层级，同时不能损失键盘、读屏、持久化与 presentation-only 语义。

本补充归档属于既有 Desktop Schema-driven Resource Widgets workflow 的 `VIEW01` / `QA01` 收敛，不包含品牌图标、协议、权限、Profile、资源数据模型或平台发布改动。

## 具体变更内容

- 从 `ResourceRenderer` 内容区移除 renderer toolbar、可见的“显示方式”文字和描述文案。
- 抽出 `ResourceRendererSelector` 并放入 `Workspace` 的 Widget 标题栏；只有存在两个及以上兼容 renderer 时才显示。
- 保留不可见的 `aria-label` / `sr-only`“显示方式”，确保选择器仍可被键盘和辅助技术识别。
- 保留 incompatible/fallback 原因在内容区显式呈现，不把真正的错误说明误删为装饰文字。
- 移除选择器左侧分割线及聚焦态竖线，聚焦时只使用轻量背景反馈。
- 更新 Workspace/Renderer 组件测试并重新生成生产 `dist`。

## Docs root

- `D:\project\MyFlowHub3\repo\MyFlowHub\docs`
- 本地 canonical monorepo；本次归档与提交均为 local-only，没有 remote、push、release 或 publication。

## 文档治理影响

- Intake impact: none
- Feature impact: none；既有 [Desktop feature](../features/desktop.md) 已规定用户在 Widget 标题区切换兼容 renderer，本次只是实现与视觉收敛。
- Requirements impact: none
- Specs impact: none
- Decision impact: none
- Lessons impact: none；修改路径直接且无昂贵、非显然或高复发排障过程。
- Related intake: [Desktop schema-driven Resource widgets](../intake/2026-08-30_desktop-schema-driven-resource-widgets.md)
- Related features: [Desktop](../features/desktop.md)
- Related requirements: [Desktop resource workspace](../requirements/desktop-resource-workspace.md)
- Related specs: [Desktop schema rendering](../specs/desktop-schema-rendering.md)
- Related decisions: [Provider schema and Desktop renderer ownership](../decisions/2026-08-30_provider-schema-desktop-renderer-ownership.md)
- Related lessons: none added；现有 [Frontend and PowerShell preflight](../lessons/frontend-and-powershell-preflight.md) 继续覆盖前端验证环境注意事项。

## 对应 plan.md 任务映射

| Task | 本次补充结果 |
| --- | --- |
| VIEW01 | renderer 选择器进入 Widget 标题栏；展示更紧凑，无内容区重复文字和标题栏分割线；选择仍只更新 View。 |
| QA01 | 增加 DOM 位置、说明文字移除和 renderer 切换断言；通过完整前端测试、生产构建和实际 UI 操作。 |
| ARC01 | 补充 change 记录、计划快照、验证截图与本地主线提交；无专用 worktree 需要再次合并或清理。 |

## 经验 / 教训摘要

- 面板 chrome 中的 renderer 选择属于 View 展示设置，应靠近 Widget 标题与移除动作，而不是占用资源内容区。
- “去掉无关文字”不能删除真正的 fallback/error 说明；装饰性描述与运行时可操作错误需要分开处理。
- 视觉简化仍需保留无障碍名称，并实际切换 renderer，不能只验证初始截图。

## 可复用排查线索

- 症状：Widget 内容顶部出现“显示方式”工具条、renderer 描述或额外横向空间；标题栏选择器左侧出现竖线。
- 触发条件：renderer 兼容选择多于一个，或选择器获得焦点。
- 关键词：`renderer-toolbar`、`widget-renderer-selector`、`显示方式`、`border-left`、`focus-within`。
- 快速检查：确认 `.widget-header .widget-renderer-selector` 存在、`.widget-body .widget-renderer-selector` 不存在；读取选择器容器计算样式时 `border-left-width` 应为 `0px`、`box-shadow` 应为 `none`；从“健康状态”切换到 `Raw JSON` 后内容应同步变化且不额外调用资源写操作。

## 关键设计决策与权衡

1. 标题栏只显示当前 renderer label，`显示方式`保留为无障碍名称，不再成为可见占位文字。
2. 选择器仍使用原生 `select`，避免为这一小型行为增加依赖或自定义键盘状态机。
3. fallback 原因继续位于内容区，因为它是资源可用性/兼容性信息，而不是冗余副标题。
4. 聚焦态使用既有 token 的背景反馈，不使用竖线或新的装饰边框。

## 测试与验证方式 / 结果

- `npm test -- --run src/components/Workspace.test.tsx`: 1 文件 / 7 测试通过。
- `npm test`: 13 文件 / 81 测试通过。
- `npm run build`: TypeScript 检查与 Vite production build 通过。
- `wails build -trimpath -platform windows/amd64 -o mfh-desktop-archive.exe`: Windows production build 通过。
- 实际 UI 操作：选择器位于 `.widget-header`，从“健康状态”切换到 `Raw JSON` 后 Raw 内容出现，再切回成功。
- 计算样式：选择器容器 `borderLeft = 0px`、`boxShadow = none`。
- 视觉证据：[标题栏 renderer 选择器无分割线](verification/2026-08-31_desktop-renderer-selector-titlebar.png)。
- `git diff --check`: 归档与产品改动通过；仅报告现有 LF/CRLF 工作树提示，无 whitespace error。

## 潜在影响

- 标题栏可用于 Widget 标题的水平空间略少；选择器已有最大宽度和 compact 密度约束，长 label 使用省略显示。
- renderer 选择、View dirty/persistence、资源读取及 fallback 行为均未改变。

## 回滚方案

- 回滚 `Workspace.tsx` / `Renderer.tsx` 的选择器位置、相关 CSS 和组件测试，并重新执行 `npm run build` 生成 `dist`。
- 回滚不涉及 View schema、资源值、Profile、CredentialStore、协议或运行时数据迁移。

## 子Agent执行轨迹

- 无。本次为单仓、小范围且写集重叠的 UI 收敛，由主 agent 直接完成；未请求 `$m-go` 或委派执行。
