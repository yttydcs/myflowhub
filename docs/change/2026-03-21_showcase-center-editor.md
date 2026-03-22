# Showcase Center + 独立编辑窗口重构

## 变更背景 / 目标
- 将原本混合了列表、配置编辑、运行交互的 `Showcase` 单页拆分为更清晰的三块：
  - `Showcase Center`：只负责 screen 列表与管理
  - `Showcase Editor Window`：独立编辑窗口，承载草稿编辑与运行态交互
  - `Showcase Viewer Window`：独立展示/运行窗口，保留自动同步能力
- 补齐 `updatedAt`，支撑列表页“最近更新时间”摘要。
- 将保存策略改为“本地草稿 + 手动保存 + dirty 提示”，降低编辑过程中的频繁落盘与 Viewer 干扰。

## 具体变更内容

### 新增
- `frontend/src/pages/ShowcaseCenter.vue`
  - 新的 Showcase 主入口，展示 screen 列表、布局类型、widget 数量、更新时间与操作按钮。
  - 支持 Blank 创建、Duplicate、Rename、Delete、打开 Editor、打开 Viewer。
- `frontend/src/windows/ShowcaseEditorWindow.vue`
  - 独立编辑窗口包装页，承载 `frontend/src/pages/Showcase.vue`。
- `plan.md`
  - 当前 workflow 计划文档与阶段记录。

### 修改
- `app_showcase.go`
  - `ShowcaseScreen` 增加 `updatedAt`。
  - normalize 阶段补齐 / 纠正 `updatedAt`，兼容空值和旧配置。
- `app_showcase_test.go`
  - 增加 `updatedAt` 默认值、保留合法值、替换非法值的回归测试。
- `frontend/src/stores/showcase.ts`
  - 增加 `ShowcaseScreenSummary`。
  - 增加 screen 摘要、复制、保存草稿、按 screen 进入订阅、配置 reload 开关等能力。
  - 在 `showcase.config_changed` 监听中支持 defer reload，避免编辑窗口的本地草稿被外部保存覆盖。
  - 持续保留 Viewer 的自动同步逻辑。
- `frontend/src/router/index.ts`
  - `/showcase` 切换为 `ShowcaseCenter`。
  - 新增 `/showcase-editor-window`。
  - `showcase-window` 标题语义调整为 `Showcase Viewer`。
- `frontend/src/pages/Showcase.vue`
  - 从“中心页 + 编辑器”混合页重构为独立编辑器。
  - 引入本地草稿、手动保存、Revert、dirty 提示、关闭前确认。
  - 保留 `columns` / `canvas_percent` 两种布局。
  - 允许在编辑器中直接操作 widget，并通过订阅刷新保持运行态交互。
  - 增加 widget 轮廓列表、布局控制区、右键菜单、canvas 拖拽 / 缩放。
  - 保存前增加 screen name 校验，避免空名称造成无效 screen。
- `frontend/src/windows/ShowcaseWindow.vue`
  - 语义统一为 `Showcase Viewer`，继续保持纯展示 / 运行定位。

### 删除
- 无物理删除文件。
- 逻辑上移除了 `frontend/src/pages/Showcase.vue` 中原有的 screen 列表管理职责。

## 对应 plan.md 任务映射
- `SC-01`
  - `app_showcase.go`
  - `app_showcase_test.go`
  - `frontend/src/stores/showcase.ts`
- `SC-02`
  - `frontend/src/pages/ShowcaseCenter.vue`
  - `frontend/src/router/index.ts`
- `SC-03`
  - `frontend/src/pages/Showcase.vue`
  - `frontend/src/windows/ShowcaseEditorWindow.vue`
  - `frontend/src/router/index.ts`
- `SC-04`
  - `frontend/src/windows/ShowcaseWindow.vue`
  - `frontend/src/stores/showcase.ts`
- `SC-05`
  - 构建 / 测试命令与验证记录

## 关键设计决策与权衡
- 保存策略采用“本地草稿 + 手动保存”而不是“编辑即保存”。
  - 原因：避免每次布局拖拽、widget 调整都触发持久化和 `showcase.config_changed` 广播。
  - 性能收益：减少不必要 I/O、降低 leave/enter 订阅抖动、避免 Viewer 高频刷新。
- 编辑窗口在挂载期间禁用 `showcase.config_changed` 自动 reload。
  - 原因：保护本地草稿，不让外部窗口保存直接覆盖当前编辑状态。
  - 权衡：V1 接受“最后保存者生效”，不做锁与冲突合并。
- 运行态交互保留在编辑器中，但通过 `enterScreen(screen)` 仅订阅当前 draft screen 相关变量。
  - 原因：满足“编辑时可直接操作 widget”的体验要求。
  - 性能收益：避免为无关 screen 建立额外订阅。
- `columns` 与 `canvas_percent` 两种布局都保留。
  - 原因：满足既有使用习惯，避免一次性砍掉现有能力。

## 测试与验证方式 / 结果
- Vue SFC 语法检查：
  - 命令：使用 `@vue/compiler-sfc` 解析以下文件
    - `frontend/src/pages/Showcase.vue`
    - `frontend/src/pages/ShowcaseCenter.vue`
    - `frontend/src/windows/ShowcaseWindow.vue`
    - `frontend/src/windows/ShowcaseEditorWindow.vue`
  - 结果：全部解析通过。
- 前端完整构建：
  - 命令：`frontend/npm run build`
  - 结果：失败。
  - 阻塞原因：仓库现有缺失 binding，`frontend/src/pages/Home.vue` 无法解析 `../../wailsjs/go/auth/AuthService`。
  - 结论：失败点不在本次 Showcase 改动范围内。
- Go 定向测试：
  - 命令：`GOWORK=off go test . -run TestNormalizeShowcase -count=1`
  - 结果：失败。
  - 阻塞原因：仓库现有 `internal/services/flow/service.go` 依赖的 `protocolexec.CapQueryReq/Resp`、`ActionCapQuery` 等符号缺失，导致根包编译失败。
  - 结论：失败点不在本次 Showcase 改动范围内。
- Wails bindings 生成：
  - 命令：`GOWORK=off wails generate module`
  - 结果：失败。
  - 阻塞原因：与上述 `internal/services/flow/service.go` 的既有编译错误相同。
- 手工代码复核：
  - 确认中心页、编辑页、Viewer 的职责边界已经拆开。
  - 确认编辑页已移除旧的 screen 管理入口与即时保存路径。
  - 确认 Viewer 仍沿用 store 的自动同步机制。

## 潜在影响与回滚方案
- 潜在影响
  - 多窗口同时编辑同一 screen 时仍是“最后保存者生效”。
  - 编辑窗口关闭前若存在未保存草稿，会触发浏览器离开确认。
  - 完整构建链路当前仍受仓库既有 Flow / Wails 绑定问题阻塞。
- 回滚方案
  - 回滚本分支对以下文件的改动即可恢复旧行为：
    - `app_showcase.go`
    - `app_showcase_test.go`
    - `frontend/src/stores/showcase.ts`
    - `frontend/src/router/index.ts`
    - `frontend/src/pages/Showcase.vue`
    - `frontend/src/pages/ShowcaseCenter.vue`
    - `frontend/src/windows/ShowcaseEditorWindow.vue`
    - `frontend/src/windows/ShowcaseWindow.vue`

## 子Agent执行轨迹
- 本次 workflow 未使用子Agent。
- 原因：当前会话存在更高优先级工具约束，只有用户显式要求时才允许派发子Agent，因此由主Agent在当前 worktree 串行完成实现、验证和归档。
