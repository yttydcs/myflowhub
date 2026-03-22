# Plan - MyFlowHub3 全局索引

> 本文件只保留：当前状态、进行中事项、文档入口。
> 已完成 workflow 的正文、Checklist、Review、Merge 记录不再堆叠在这里，统一沉淀到 `docs/plan/` 与 `docs/change/`。
> 压缩前的完整根级快照：`docs/plan/plan_archive_2026-03-21_global-plan-pre-slim.md`。

## Current Status
- 协议字典以 `MyFlowHub-Proto`（`github.com/yttydcs/myflowhub-proto/protocol/*`）为准；`MyFlowHub-Server/protocol/*` 仅保留兼容壳语义。
- 客户端基础能力以 `MyFlowHub-SDK` 为统一入口；`MyFlowHub-Win`、`MyFlowHub-Android`、`MyFlowHub-MetricsNode` 作为上层应用尽量复用 SDK / Core / Proto。
- `repo/` 是控制面与集成面；`worktrees/` 是实现面；`docs/` 负责全局归档与接手材料。
- 根级 `plan.md` 不再承载已完成 workflow 的详细正文，避免历史内容反复堆积。
- `2026-03-22`：workspace 根文档与 `MyFlowHub-Server` 文档已完成 taxonomy 治理、索引重建与计划归档收敛；详见 `docs/change/2026-03-22_workspace-docs-governance.md`、`docs/change/2026-03-22_server-docs-governance.md`、`docs/plan/plan_archive_2026-03-22_workspace-docs-governance.md` 与 `docs/plan/plan_archive_2026-03-22_server-docs-governance.md`。
- `2026-03-22`：`Flow` 数据流 DAG 跨仓 workflow 已完成，覆盖 `MyFlowHub-SubProto` 运行时结果绑定与 `compose`、`MyFlowHub-Win` 表单化 DAG 编辑器，以及 `MyFlowHub-Server` 长期 requirements/specs；详见 `docs/change/2026-03-22_flow-data-dag-runtime.md`、`docs/change/2026-03-22_win-flow-data-dag-editor.md`、`docs/change/2026-03-22_server-flow-data-dag-docs.md` 与 `docs/plan/plan_archive_2026-03-22_flow-data-dag-runtime.md`、`docs/plan/plan_archive_2026-03-22_win-flow-data-dag-editor.md`、`docs/plan/plan_archive_2026-03-22_server-flow-data-dag-docs.md`。
- `2026-03-22`：`MyFlowHub-Win` 已完成主页面首屏头部收敛，移除顶部小标题并统一为单一大标题加一行提示语；详见 `docs/change/2026-03-22_win-page-hero-simplify.md` 与 `docs/plan/plan_archive_2026-03-22_win-page-hero-simplify.md`。
- `2026-03-22`：`MyFlowHub-Win` 已完成页面与独立窗口卡片头收敛，引入共享 `CardHeader` 去除重复顶部小标题，仅保留主标题与原有提示语；详见 `docs/change/2026-03-22_win-card-header-simplify.md` 与 `docs/plan/plan_archive_2026-03-22_win-card-header-simplify.md`。
- `2026-03-22`：`MyFlowHub-Win` 已统一按钮 hover 为手型并禁止按钮文字选中复制；详见 `docs/change/2026-03-22_win-button-pointer.md` 与 `docs/plan/plan_archive_2026-03-22_win-button-pointer.md`。
- `2026-03-22`：`MyFlowHub-Win` 已完成 `Flow` 本地项目列表精简，列表不再显示 `Flow ID` / `Project ID`，并将名称与更新时间收敛为同一行灰色文本；详见 `docs/change/2026-03-22_flow-list-row-simplify.md` 与 `docs/plan/plan_archive_2026-03-22_win-flow-list-row-simplify.md`。
- `2026-03-22`：`MyFlowHub-Win` 已完成 `Flow` 本地项目列表的名称/时间布局修正，名称恢复主文本样式，更新时间保持灰色，并优化了与右侧按钮的同排行为；详见 `docs/change/2026-03-22_flow-list-inline-meta.md` 与 `docs/plan/plan_archive_2026-03-22_win-flow-list-inline-meta.md`。
- `2026-03-22`：`MyFlowHub-Win` 已完成 `Settings` 置底、`Other` 分组、全局语言设置与现有前端页面 i18n 基础接入；详见 `docs/change/2026-03-22_win-settings-i18n.md` 与 `docs/plan/plan_archive_2026-03-22_win-settings-i18n.md`。
- `2026-03-22`：`MyFlowHub-Win` 已完成 TopicBus 独立窗口头部精简，并将 `Event Cache Settings` 迁移到 `Settings` 页面；详见 `docs/change/2026-03-22_topicbus-settings-pane.md` 与 `docs/plan/plan_archive_2026-03-22_win-topicbus-settings-pane.md`。
- `2026-03-22`：`MyFlowHub-Win` 已完成 TopicBus 默认目标迁移到 `Settings`、`Overview` 单列化、`Channels` 集中订阅管理与独立窗口目标收敛；详见 `docs/change/2026-03-22_topicbus-target-settings.md` 与 `docs/plan/plan_archive_2026-03-22_win-topicbus-target-settings.md`。
- `2026-03-22`：`MyFlowHub-Win` 已修复 TopicBus 独立窗口右侧 `Window Snapshot / Window Actions` 侧栏在低高度窗口下无法滚动的问题；详见 `docs/change/2026-03-22_topicbus-window-sidebar-scroll.md` 与 `docs/plan/plan_archive_2026-03-22_win-topicbus-window-sidebar-scroll.md`。
- `2026-03-22`：`MyFlowHub-Win` 已修复 TopicBus 独立窗口 `Window Actions` 卡片的父子高度承接问题，按钮区与说明区在受限高度下可完整访问；详见 `docs/change/2026-03-22_topicbus-window-actions-height.md` 与 `docs/plan/plan_archive_2026-03-22_win-topicbus-window-actions-height.md`。
- `2026-03-22`：`MyFlowHub-Win` 已完成“变量池 / 主题总线”中文术语收敛、漏翻 key 补齐与设置页卡片独立保存；详见 `docs/change/2026-03-22_win-translation-settings.md` 与 `docs/plan/plan_archive_2026-03-22_win-translation-settings.md`。
- `2026-03-22`：`management-node-display-name` 跨仓 workflow 已完成，覆盖 `MyFlowHub-Proto` 的 `display_name` schema、`MyFlowHub-SubProto` 的昵称回传与持久化 hook、`MyFlowHub-Server` 的 `hubruntime` 分层持久化，以及 `MyFlowHub-Win` 的 Devices 昵称显示；详见 `docs/change/2026-03-22_management-node-display-name.md`、`docs/change/2026-03-22_management-display-name-and-persistence-hook.md`、`docs/change/2026-03-22_hubruntime-layered-config-persistence.md`、`docs/change/2026-03-22_devices-node-display-name.md` 与对应 `docs/plan/plan_archive_2026-03-22_*management-node-display-name.md`。
- `2026-03-22`：`MyFlowHub-SubProto/flow` 已修复 `set/delete` 状态一致性、远端失败显式响应和 run 保留策略；详见 `docs/change/2026-03-22_flow-state-route-retention.md`。
- `2026-03-22`：`MyFlowHub-Win` 左侧边栏已进一步收敛，菜单按钮迁移到顶部连接状态左侧，收起态 tooltip 与滚动宽度对齐问题已补齐；详见 `docs/change/2026-03-22_win-sidebar-toggle-polish.md`。
- `2026-03-21`：`flow` 子协议已补齐 `run/status/list/get` 契约，且 `flow_id` 已收紧为 UUID 校验；详见 `docs/change/2026-03-21_server-flow-contract-align.md` 与 `docs/change/2026-03-21_flow-id-guard-and-contract-align.md`。
- `2026-03-21`：`MyFlowHub-Win` 左侧边栏已移除顶部 `Session` 卡片，并支持收起为窄栏图标导航；详见 `docs/change/2026-03-21_win-sidebar-toggle.md`。
- `2026-03-21`：`MyFlowHub-Win` 已新增独立 `Settings` 页面，覆盖默认地址、默认设备 ID、自动连接、自动登录、界面偏好与 About 信息展示；详见 `docs/change/2026-03-21_win-settings-page.md`。

## Active Items
- `management-node-display-name` 已完成收敛，相关 plan/change 已迁入全局 `docs/plan/` 与 `docs/change/`。
- 新 workflow 仍遵守：
  - 根级 `plan.md` 只保留摘要、状态和入口链接
  - 详细执行计划放到对应 worktree 根目录的 `plan.md` / `todo.md`
  - workflow 完成后，将详细内容归档到 `docs/plan/` 与 `docs/change/`

## Documentation Map
- `docs/README.md`
  - `docs/` 目录总入口；串联 requirements、specs、plan、change、lessons 等文档。
- `target.md`
  - 当前根目录中不存在；仅作为历史归档中的旧引用保留理解。
- `repos.md`
  - 记录仓库职责、依赖边界、推进顺序、接手说明。
- `plan.md`
  - 记录全局当前状态、进行中事项入口、文档分工。
- `repo/*`
  - 各个实际源码仓库。
  - 仓库主线不再长期保留已完成 workflow 的 `plan.md` / `todo.md`；这类文档应优先存在于 worktree 并在完成后归档到 `docs/`。
- `docs/change/`
  - 记录已完成变更的背景、结果、验证、影响与回滚。
  - 入口索引：`docs/change/README.md`
- `docs/plan/`
  - 记录历史 workflow 的完整计划正文、Checklist、Review 证据，以及根级旧版 plan 快照。
  - 入口索引：`docs/plan/README.md`
- `docs/specs/protocol_map.md`
  - 记录协议映射速查表。

## Historical Entry Points
- `Flow 数据流 DAG 运行时` workflow 计划归档：`docs/plan/plan_archive_2026-03-22_flow-data-dag-runtime.md`
- `Win Flow 数据流 DAG 编辑器` workflow 计划归档：`docs/plan/plan_archive_2026-03-22_win-flow-data-dag-editor.md`
- `Server Flow 数据流 DAG 文档` workflow 计划归档：`docs/plan/plan_archive_2026-03-22_server-flow-data-dag-docs.md`
- `Flow 数据流 DAG 运行时` 变更归档：`docs/change/2026-03-22_flow-data-dag-runtime.md`
- `Win Flow 数据流 DAG 编辑器` 变更归档：`docs/change/2026-03-22_win-flow-data-dag-editor.md`
- `Server Flow 数据流 DAG 文档` 变更归档：`docs/change/2026-03-22_server-flow-data-dag-docs.md`
- 文档总入口：`docs/README.md`
- 根级旧版全量历史：`docs/plan/plan_archive_2026-03-21_global-plan-pre-slim.md`
- `Flow 子协议契约补齐` 变更归档：`docs/change/2026-03-21_server-flow-contract-align.md`
- `Flow ID 防护与契约对齐` 变更归档：`docs/change/2026-03-21_flow-id-guard-and-contract-align.md`
- `Proto exec cap_query 基线确认` workflow 计划归档：`docs/plan/plan_archive_2026-03-21_proto-exec-cap-query-baseline.md`
- `Proto exec cap_query 基线确认` 变更归档：`docs/change/2026-03-21_proto-exec-cap-query-baseline.md`
- `Win 对齐 Proto exec cap_query 基线恢复构建` workflow 计划归档：`docs/plan/plan_archive_2026-03-21_win-proto-cap-query-baseline.md`
- `Win 对齐 Proto exec cap_query 基线恢复构建` 变更归档：`docs/change/2026-03-21_win-proto-cap-query-baseline.md`
- `Win 前端构建链路恢复` workflow 计划归档：`docs/plan/plan_archive_2026-03-21_win-frontend-build-chain.md`
- `Win 前端构建链路恢复` 变更归档：`docs/change/2026-03-21_win-frontend-build-chain.md`
- `Win Flow 编辑器文案精简` workflow 计划归档：`docs/plan/plan_archive_2026-03-21_win-editor-chrome-trim.md`
- `Win Flow 编辑器文案精简` 变更归档：`docs/change/2026-03-21_win-editor-chrome-trim.md`
- `Win Flow 编辑器顶部标题移除` workflow 计划归档：`docs/plan/plan_archive_2026-03-21_win-editor-header-title-trim.md`
- `Win Flow 编辑器顶部标题移除` 变更归档：`docs/change/2026-03-21_win-editor-header-title-trim.md`
- `Codex MCP runtime cleanup` workflow 计划归档：`docs/plan/plan_archive_2026-03-21_codex-mcp-runtime-cleanup.md`
- `Codex MCP runtime cleanup` 变更归档：`docs/change/2026-03-21_codex-mcp-runtime-cleanup.md`
- `Win Flow 项目中心与编辑器收敛` workflow 计划归档：`docs/plan/plan_archive_2026-03-21_win-flow-project-center-editor.md`
- `Win Flow 项目中心与编辑器收敛` 变更归档：`docs/change/2026-03-21_flow-project-center-editor.md`
- `Win VarPool 页签化单列重构` workflow 计划归档：`docs/plan/plan_archive_2026-03-21_win-varpool-tab-layout.md`
- `Win VarPool 页签化单列重构` 变更归档：`docs/change/2026-03-21_varpool-tab-layout.md`
- `Win Flow 编辑器方法选择与抽屉样式收敛` workflow 计划归档：`docs/plan/plan_archive_2026-03-21_win-flow-editor-method-selector-dialog.md`
- `Win Flow 编辑器方法选择与抽屉样式收敛` 变更归档：`docs/change/2026-03-21_flow-editor-method-selector-dialog.md`
- `Win Flow 方法能力按指定节点查询` workflow 计划归档：`docs/plan/plan_archive_2026-03-21_win-flow-method-capability-query-node.md`
- `Win Flow 方法能力按指定节点查询` 变更归档：`docs/change/2026-03-21_win-flow-method-capability-query-node.md`
- `Win Showcase Center 与独立编辑窗口重构` workflow 计划归档：`docs/plan/plan_archive_2026-03-21_win-showcase-center-editor.md`
- `Win Showcase Center 与独立编辑窗口重构` 变更归档：`docs/change/2026-03-21_showcase-center-editor.md`
- `Win Showcase 界面进一步精简` workflow 计划归档：`docs/plan/plan_archive_2026-03-21_win-showcase-ui-simplify.md`
- `Win Showcase 界面进一步精简` 变更归档：`docs/change/2026-03-21_showcase-ui-simplify.md`
- `Win Showcase 编辑窗口进一步极简化` workflow 计划归档：`docs/plan/plan_archive_2026-03-21_win-showcase-editor-minimal.md`
- `Win Showcase 编辑窗口进一步极简化` 变更归档：`docs/change/2026-03-21_win-showcase-editor-minimal.md`
- `Win Showcase 窗口标题占位精简` workflow 计划归档：`docs/plan/plan_archive_2026-03-21_win-showcase-window-title-trim.md`
- `Win Showcase 窗口标题占位精简` 变更归档：`docs/change/2026-03-21_showcase-window-title-trim.md`
- `Win Showcase 头部对齐 Flow` workflow 计划归档：`docs/plan/plan_archive_2026-03-21_win-showcase-header-align.md`
- `Win Showcase 头部对齐 Flow` 变更归档：`docs/change/2026-03-21_showcase-header-align.md`
- `Win TopicBus 双段式页面与独立频道窗口` workflow 计划归档：`docs/plan/plan_archive_2026-03-21_win-topicbus-window-console.md`
- `Win TopicBus 双段式页面与独立频道窗口` 变更归档：`docs/change/2026-03-21_topicbus-window-console.md`
- `Win TopicBus 独立窗口布局精简` workflow 计划归档：`docs/plan/plan_archive_2026-03-22_win-topicbus-window-layout.md`
- `Flow 状态一致性、转发失败响应与 run 保留修复` 变更归档：`docs/change/2026-03-22_flow-state-route-retention.md`
- `Win Showcase 顶部按钮图标化` workflow 计划归档：`docs/plan/plan_archive_2026-03-22_win-showcase-header-icon-actions.md`
- `Win Showcase 顶部按钮图标化` 变更归档：`docs/change/2026-03-22_showcase-header-icon-actions.md`
- `Win TopicBus 独立窗口布局精简` 变更归档：`docs/change/2026-03-22_topicbus-window-layout.md`
- `Win TopicBus 头部精简与设置迁移` workflow 计划归档：`docs/plan/plan_archive_2026-03-22_win-topicbus-settings-pane.md`
- `Win TopicBus 头部精简与设置迁移` 变更归档：`docs/change/2026-03-22_topicbus-settings-pane.md`
- `Win TopicBus 默认目标设置与频道总览收敛` workflow 计划归档：`docs/plan/plan_archive_2026-03-22_win-topicbus-target-settings.md`
- `Win TopicBus 默认目标设置与频道总览收敛` 变更归档：`docs/change/2026-03-22_topicbus-target-settings.md`
- `Win TopicBus 窗口右侧侧栏滚动修复` workflow 计划归档：`docs/plan/plan_archive_2026-03-22_win-topicbus-window-sidebar-scroll.md`
- `Win TopicBus 窗口右侧侧栏滚动修复` 变更归档：`docs/change/2026-03-22_topicbus-window-sidebar-scroll.md`
- `Win TopicBus 窗口操作区高度承接修复` workflow 计划归档：`docs/plan/plan_archive_2026-03-22_win-topicbus-window-actions-height.md`
- `Win TopicBus 窗口操作区高度承接修复` 变更归档：`docs/change/2026-03-22_topicbus-window-actions-height.md`
- `Proto management 节点显示名` workflow 计划归档：`docs/plan/plan_archive_2026-03-22_proto-management-node-display-name.md`
- `Proto management 节点显示名` 变更归档：`docs/change/2026-03-22_management-node-display-name.md`
- `SubProto management 显示名与持久化 hook` workflow 计划归档：`docs/plan/plan_archive_2026-03-22_subproto-management-node-display-name.md`
- `SubProto management 显示名与持久化 hook` 变更归档：`docs/change/2026-03-22_management-display-name-and-persistence-hook.md`
- `Server hubruntime 分层持久化` workflow 计划归档：`docs/plan/plan_archive_2026-03-22_server-management-node-display-name.md`
- `Server hubruntime 分层持久化` 变更归档：`docs/change/2026-03-22_hubruntime-layered-config-persistence.md`
- `Win Devices 节点显示名` workflow 计划归档：`docs/plan/plan_archive_2026-03-22_win-management-node-display-name.md`
- `Win Devices 节点显示名` 变更归档：`docs/change/2026-03-22_devices-node-display-name.md`
- `Win 左侧边栏二次收敛` workflow 计划归档：`docs/plan/plan_archive_2026-03-22_win-sidebar-toggle-polish.md`
- `Win 左侧边栏二次收敛` 变更归档：`docs/change/2026-03-22_win-sidebar-toggle-polish.md`
- `Win 设置页分组与 i18n` workflow 计划归档：`docs/plan/plan_archive_2026-03-22_win-settings-i18n.md`
- `Win 设置页分组与 i18n` 变更归档：`docs/change/2026-03-22_win-settings-i18n.md`
- `Win 中文术语与设置卡片独立保存` workflow 计划归档：`docs/plan/plan_archive_2026-03-22_win-translation-settings.md`
- `Win 中文术语与设置卡片独立保存` 变更归档：`docs/change/2026-03-22_win-translation-settings.md`
- `Win 按钮 hover 手型与禁选中文本` workflow 计划归档：`docs/plan/plan_archive_2026-03-22_win-button-pointer.md`
- `Win 按钮 hover 手型与禁选中文本` 变更归档：`docs/change/2026-03-22_win-button-pointer.md`
- `Win Flow 列表简化` workflow 计划归档：`docs/plan/plan_archive_2026-03-22_win-flow-list-row-simplify.md`
- `Win Flow 列表简化` 变更归档：`docs/change/2026-03-22_flow-list-row-simplify.md`
- `Win Flow 列表名称与时间布局` workflow 计划归档：`docs/plan/plan_archive_2026-03-22_win-flow-list-inline-meta.md`
- `Win Flow 列表名称与时间布局` 变更归档：`docs/change/2026-03-22_flow-list-inline-meta.md`
- `Win 卡片头收敛` workflow 计划归档：`docs/plan/plan_archive_2026-03-22_win-card-header-simplify.md`
- `Win 卡片头收敛` 变更归档：`docs/change/2026-03-22_win-card-header-simplify.md`
- `Win 页面头部简化` workflow 计划归档：`docs/plan/plan_archive_2026-03-22_win-page-hero-simplify.md`
- `Win 页面头部简化` 变更归档：`docs/change/2026-03-22_win-page-hero-simplify.md`
- `Win 左侧边栏收起改造` workflow 计划归档：`docs/plan/plan_archive_2026-03-21_win-sidebar-toggle.md`
- `Win 左侧边栏收起改造` 变更归档：`docs/change/2026-03-21_win-sidebar-toggle.md`
- `Win 设置页面` workflow 计划归档：`docs/plan/plan_archive_2026-03-21_win-settings-page.md`
- `Win 设置页面` 变更归档：`docs/change/2026-03-21_win-settings-page.md`
- `Win Flow 画布拖拽连线修复` workflow 计划归档：`docs/plan/plan_archive_2026-03-21_win-canvas-drag-state.md`
- `Win Flow 画布拖拽连线修复` 变更归档：`docs/change/2026-03-21_win-canvas-drag-state.md`
- `Win Flow 画布连线与头尾辨识修复` workflow 计划归档：`docs/plan/plan_archive_2026-03-21_win-canvas-connectors.md`
- `Win Flow 画布连线与头尾辨识修复` 变更归档：`docs/change/2026-03-21_win-canvas-connectors.md`
- 变更归档总索引：`docs/change/README.md`
- 计划归档总索引：`docs/plan/README.md`
- 历史 workflow 计划正文：`docs/plan/`
- 历史变更结果与验证记录：`docs/change/`

## Maintenance Rules
- 已完成内容不再直接写回根级 `plan.md` 正文。
- 根级 `plan.md` 优先做索引，不重复拷贝 `docs/change/` 或 `docs/plan/` 的大段内容。
- `repo/*` 主线目录不应堆积历史 workflow 的 `plan.md` / `todo.md`；若仍需仓内长期说明，应改写到 README 或正式 docs。
- 若后续发现某段历史内容仍只存在于临时计划中，应先归档到 `docs/`，再在根级 `plan.md` 保留简要引用。

