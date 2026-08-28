# 2026-04-09 Win Stream Control Target Picker

## 变更背景 / 目标

- 当前 `Stream` 的 `Control` 页虽然已经具备查询、连接和运行态观察能力，但主页面仍常驻 `Source Catalog` 和 `Consumer Catalog` 两块大面板，页面密度偏高。
- 本轮目标是在不改 `stream` 协议、不改 `StreamService`/store 契约的前提下，移除这两块常驻 catalog，并让 `Control Target` 通过弹出式选择框完成 source / consumer 配对与连接。

## 具体变更内容

### 修改：Control 主页面收敛为目标与摘要视图

- `frontend/src/pages/Stream.vue`
  - 移除 `Control` 主页面常驻的 `Source Catalog` 与 `Consumer Catalog`
  - 保留 `Control Target` 与 `Runtime Deliveries`
  - 新增当前已选 source / consumer 摘要区，主页面只展示已选 pair，而不再常驻渲染两个目录列表

### 修改：新增 control 配对弹窗

- `frontend/src/pages/Stream.vue`
  - 新增 `data-stream-open-control-picker` 入口和 `data-stream-control-dialog` 配对弹窗
  - 在弹窗内承载 source / consumer 查询、选择、`connect`、`subscribe`
  - 继续复用已有：
    - `stream.listSources(...)`
    - `stream.listConsumers(...)`
    - `stream.connect(...)`
    - `stream.subscribe(...)`
    - `stream.selectSource(...)`
    - `stream.selectConsumer(...)`
  - `connect` / `subscribe` 成功后关闭弹窗，失败仍沿用现有 toast 错误反馈

### 修改：文案与页面测试

- `frontend/src/i18n/messages/stores.ts`
  - 新增 control picker、已选 pair 摘要和交互提示文案
- `frontend/src/pages/Stream.test.ts`
  - 新增控制页不再显示 catalog 的断言
  - 新增 control picker 连接路径测试

## Requirements impact

- `none`

## Specs impact

- `none`

## Lessons impact

- `none`

## Related requirements

- `docs/requirements/stream.md`

## Related specs

- `docs/specs/stream.md`

## Related lessons

- `docs/lessons/frontend-build-empty-node-modules.md`
- `docs/lessons/wails-binding-proto-drift.md`

## 对应 plan.md 任务映射

- `SCTP-1`
  - `Stream.vue` control 主页面重构与配对弹窗
- `SCTP-2`
  - `stores.ts` control 文案补充
- `SCTP-3`
  - `Stream.test.ts` control picker 回归测试与前端验证
- `SCTP-4`
  - 3.3 review 与本次归档

## 经验 / 教训摘要

- `Stream` 这类产品化页面里，连接所需的 catalog 查询可以收纳进弹窗，不必长期占住主页面；主页面只保留 target、当前摘要和 runtime 状态更符合 Win 当前交互风格。
- 这次改动不需要动 store 或协议调用，只要让页面继续复用现有选择态与 `withToast(...)` 即可，改动面比新增 control store 状态更小。
- fresh worktree 的前端验证依然受环境准备影响，先补 `node_modules`，再补 `wails generate module`，能把环境缺失和功能回归拆开。

## 可复用排查线索

- 症状
  - `Control` 页仍然出现 `Source Catalog` / `Consumer Catalog`
  - 点开 control picker 后没有 source / consumer 行
  - `npm exec vitest run src/pages/Stream.test.ts` 启动时报找不到 `vitest` 或 `@vitejs/plugin-vue`
  - `npm run build` 报 `Could not resolve "../../wailsjs/runtime/runtime"`
- 触发条件
  - 页面仍使用旧的 control 双目录布局
  - worktree 缺少 `frontend/node_modules`
  - worktree 未执行 `wails generate module`
- 关键词
  - `data-stream-open-control-picker`
  - `data-stream-control-dialog`
  - `Select Pair`
  - `frontend-build-empty-node-modules`
  - `wailsjs/runtime/runtime`
- 快速检查
  - 看 `frontend/src/pages/Stream.vue` 是否只保留 `Control Target` / `Runtime Deliveries`
  - 看 `frontend/src/pages/Stream.vue` 是否存在 `data-stream-control-dialog`
  - 看 `frontend/node_modules` 是否存在；缺失先执行 `npm ci`
  - 看 worktree 根是否执行过 `$env:GOWORK='off'; wails generate module`

## 关键设计决策与权衡

- 决策：使用一个共享 control picker 弹窗，而不是把 source 和 consumer 目录继续常驻在主页面
  - 原因：满足“主页面简洁、动作走弹窗”的用户要求，同时改动面最小
- 决策：继续复用 store 的 `selectedSourceId` / `selectedConsumerId`
  - 原因：避免引入新的 control-specific 数据模型或重复协议调用封装
- 决策：主页面只保留已选 pair 摘要，不保留直接 `connect` 按钮
  - 原因：连接动作应通过弹出式选择框完成，避免又退回“主页面操作依赖 catalog”的旧模式

## 测试与验证方式 / 结果

- `MyFlowHub-Win/frontend`
  - `npm ci`
  - 结果：通过
- `MyFlowHub-Win/frontend`
  - `npm exec vitest run src/pages/Stream.test.ts`
  - 结果：通过（`1` file, `5` tests）
- `MyFlowHub-Win`
  - `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 备注：仍有既有 `Not found: time.Time` 提示，但退出码为 `0`
- `MyFlowHub-Win/frontend`
  - `npm run build`
  - 结果：通过
  - 备注：保留既有大 chunk warning，非本轮新增问题

## 潜在影响与回滚方案

- 潜在影响
  - `Control` 页现在通过弹窗完成 source / consumer 配对
  - 主页面不再常驻显示远端 source / consumer 目录
  - 已选 pair 会在主页面以摘要形式展示
- 回滚方案
  - 回退以下文件即可恢复到旧的 control 双目录布局：
    - `frontend/src/pages/Stream.vue`
    - `frontend/src/pages/Stream.test.ts`
    - `frontend/src/i18n/messages/stores.ts`
    - `plan.md`
    - `docs/change/2026-04-09_win-stream-control-target-picker.md`

## 子Agent执行轨迹

- 未使用子Agent
