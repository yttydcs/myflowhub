# 2026-03-29 Win Stream Page i18n

## 变更背景 / 目标

- 当前 `Stream` 页把 source / consumer 查询、两个新增表单、delivery 控制和 runtime viewer 同时摊在主页面，信息密度过高。
- 页面虽然已经使用 `t(...)`，但 `Stream` 相关 key、导航描述和路由副标题没有完整落入 `zh-CN` 词典，中文环境下仍会大量回退到英文。
- 本次目标是在不改动 `stream.ts` 与后端协议的前提下，重排 `Stream` UX，使页面以摘要、列表和侧栏操作为主，并把新增入口收进弹窗，同时补齐中文文案覆盖。

## 具体变更内容

### 修改
- `frontend/src/pages/Stream.vue`
  - 重构为 `PageHero + summary cards + source/consumer 紧凑目录 + 右侧 action/runtime/viewer` 布局
  - source / consumer 新增改为 `Overlay` 弹窗，主页面不再长期常驻大表单
  - 在身份同步完成后静默拉取 sources / consumers / deliveries，让页面首次进入就有目录摘要
  - 增加测试选择器，方便验证“默认无常驻表单，点击后再打开新增弹窗”
- `frontend/src/i18n/messages/operations.ts`
  - 补齐 `Stream` 页面主体文案、toast 文案和 `stream.ts` 相关错误文案的 `zh-CN` 映射
- `frontend/src/i18n/messages/shell.ts`
  - 补齐 `Stream` 导航标题、导航描述和路由副标题的 `zh-CN` 映射
- `frontend/src/pages/Stream.test.ts`
  - 新增页面测试，覆盖首屏自动加载和弹窗式新增 source
- `frontend/src/i18n/messages/shell.test.ts`
  - 增补 `Stream` 导航/路由词条断言
- `docs/plan/plan_archive_2026-03-29_win-stream-page-i18n.md`
  - 归档本轮 workflow 的完整计划、review 与验证记录
- `docs/lessons/frontend-worktree-wailsjs-missing.md`
  - 记录 worktree 缺少 `frontend/wailsjs` 时的复用排查线索
- `docs/lessons/README.md`
  - 更新 lesson 索引
- `docs/change/README.md`
  - 挂载本次 change 入口

### 不变
- 不修改 `frontend/src/stores/stream.ts` 对外接口
- 不修改 Go `StreamService`、runtime 事件协议或持久化契约
- 不新增新的 i18n 框架

## Requirements impact

- `none`

## Specs impact

- `none`

## Lessons impact

- `updated`

## Related requirements

- `repo/MyFlowHub-Win/docs/requirements/stream.md`

## Related specs

- `repo/MyFlowHub-Win/docs/specs/stream.md`

## Related lessons

- `docs/lessons/frontend-worktree-wailsjs-missing.md`

## 对应 plan.md 任务映射

- `STREAM-UX-1`
  - 重构 `Stream` 主页面布局与摘要区
- `STREAM-UX-2`
  - 将 source / consumer 新增录入改为弹窗式交互
- `STREAM-I18N-1`
  - 补齐 `Stream` 页面、导航和路由中文词条
- `STREAM-VAL-1`
  - 新增页面测试并执行前端验证
- `STREAM-REVIEW-1`
  - 完成 3.3 checklist
- `STREAM-ARCHIVE-1`
  - 更新 `docs/change` 与 `docs/lessons` 归档

## 经验 / 教训摘要

- `Stream` 这类控制台页如果把“目录查询、编辑录入、运行态查看”同时平铺，会迅速失去可读性；主页面应优先保留摘要、选择和高频动作，把录入收进弹窗。
- 轻量 i18n 架构下，页面、导航、路由、副标题和 toast 都可能各自落在不同消息表，补词条时必须一并核对。
- 在 worktree 中跑前端验证时，`frontend/wailsjs` 这类被忽略的生成目录不会自动出现，若不先补齐会把环境问题误判成代码回归。

## 可复用排查线索

- 症状
  - 中文环境下 `Stream` 页、导航或 toast 仍出现英文原文
  - `Stream` 首屏仍然出现大块常驻录入表单
  - worktree 中执行 `npm test` 或 `npm run build` 报 `Failed to resolve import "../../wailsjs/runtime/runtime"`
- 触发条件
  - 页面只改模板，没有同步补 `shell.ts` / `operations.ts`
  - 新 worktree 未包含被 `.gitignore` 忽略的 `frontend/wailsjs`
  - 主页面同时承载新增录入和 runtime viewer，交互密度没有拆开
- 关键词
  - `Stream`
  - `Sources and deliveries`
  - `Query typed sources and consumers, connect deliveries, and inspect runtime traffic.`
  - `Failed to resolve import "../../wailsjs/runtime/runtime"`
  - `frontend/wailsjs`
- 快速检查
  - 搜索 `frontend/src/pages/Stream.vue` 中的 `t(...)` key 是否都在消息表落地
  - 检查 `frontend/src/i18n/messages/shell.ts` 是否包含 `Stream` 的导航/路由词条
  - 检查 worktree 下 `frontend/wailsjs` 是否存在；若不存在，先生成或复制再验证

## 关键设计决策与权衡

- 决策：只重构 `Stream.vue` 的 UX shell，不改 `stream.ts`
  - 原因：本轮目标是降低页面复杂度和补齐 i18n，不是改变协议或 store 职责边界
- 决策：新增 source / consumer 使用弹窗，不做整页编辑模式
  - 原因：用户明确希望减少页面常驻表单；当前 store 也没有单独编辑 API，弹窗新增是最小安全改动
- 决策：导航/路由继续复用现有 key 文案，由消息表承接中文翻译
  - 原因：最小改动即可满足 i18n 目标，不需要额外调整 router / shell 结构
- 决策：将 `frontend/wailsjs` worktree 缺失问题沉淀为 lesson
  - 原因：该问题与本轮业务改动无关，但在 worktree-first workflow 中容易重复出现

## 测试与验证方式 / 结果

- 安装依赖
  - 执行：`npm ci`
  - 结果：通过
- 全量前端测试
  - 执行：`npm test`
  - 结果：通过，`17` 个测试文件 / `69` 个用例全部通过
- 前端构建
  - 执行：`npm run build`
  - 结果：通过
  - 备注：保留既有 Vite chunk size warning，本轮无新增阻塞
- worktree 生成态依赖补齐
  - 执行：将 `repo/MyFlowHub-Win/frontend/wailsjs` 复制到 worktree 的 `frontend/wailsjs`
  - 结果：完成验证所需前置条件，且该目录仍被 `.gitignore` 忽略，不进入提交集

## 潜在影响

- `Stream` 页默认会在身份可用后自动拉取一次目录与 delivery 快照
- `frontend/wailsjs` 仍是生成态依赖；若后续在新的 worktree 重复前端验证，需要再次确认该目录是否存在
- 中文文案改动会直接影响 `Stream` 模块的导航、页面和 toast 呈现

## 回滚方案

- 回退 `frontend/src/pages/Stream.vue`
- 回退 `frontend/src/i18n/messages/operations.ts`
- 回退 `frontend/src/i18n/messages/shell.ts`
- 回退 `frontend/src/pages/Stream.test.ts`
- 回退 `frontend/src/i18n/messages/shell.test.ts`
- 回退 `docs/plan/plan_archive_2026-03-29_win-stream-page-i18n.md`
- 回退 `docs/change/README.md`
- 回退 `docs/lessons/frontend-worktree-wailsjs-missing.md`
- 回退 `docs/lessons/README.md`
- 删除本归档文件

## 子Agent执行轨迹

- 未使用子Agent
