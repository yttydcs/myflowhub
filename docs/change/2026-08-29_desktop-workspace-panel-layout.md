# Desktop 工作区满区与可调分栏布局

## 变更背景 / 目标

旧 Workspace 把新加入的资源按固定 `4 × 4` 网格单元放在左上角，单个资源只占很小区域；多个资源虽然
具有 `x/y/w/h`，但缺少明确的平铺规则、直接拖拽换位和相邻面板比例控制。本次把工作区改为有界平铺：
首个资源占满可用区，第二个默认加入右侧，面板可拖拽或通过按钮左右换位，双栏比例可调并随 View 保存。

## 文档治理影响

- Docs root: `docs/`
- Intake impact: none；沿用 [可扩展资源平台与 Desktop 工作区重构](../intake/2026-08-28_extensible-resources-and-desktop-workspace-redesign.md) 的 View/Workspace 需求，不新增原始需求副本。
- Feature impact: updated [Desktop](../features/desktop.md)。
- Requirements impact: updated [Desktop 资源工作区](../requirements/desktop-resource-workspace.md)。
- Specs impact: updated [Desktop Resource Workspace v2](../specs/desktop-resource-workspace-v2.md)。
- Decision impact: none；继续使用既有 View version 1 与 12 列 `x/y/w/h`，没有新增架构或持久化模型。
- Lessons impact: none；没有形成新的可复用故障模式。
- Index impact: updated `docs/change/README.md`；既有 feature/requirement/spec 叶子未改变索引拓扑。

## 具体变更

- 首个 Widget 规范化为 `x=0, y=0, w=12, h=24`，占满工作区可用宽高。
- 第二个 Widget 默认插到右侧并形成 `6:6` 双栏；拖入目标左/右半侧时按对应方向插入。
- Widget 标题栏新增拖动手柄，拖到另一面板左/右侧即可换位；左右按钮提供稳定的键盘等价入口。
- 双栏中间增加 vertical separator，支持 pointer capture、Left/Right、Home/End、Enter 和双击复位；比例限制为 `2:10` 到 `10:2`。
- 删除至一个 Widget 后自动恢复满区；三个以上 Widget 使用每行最多四个、总计 24 行单位的有界分块规则。
- 旧单 Widget 和非完整双栏 View 在加载时规范化并标记为未保存，用户保存后写回同一个 View v1 schema；三个以上的旧 View 只在发生布局操作后进入新规则。
- 保留缺失资源 detached 状态、renderer 行为、Inspector、Profile 隔离和原子 View store；没有引入新依赖。

## 测试与验证

- `npm test`：7 个测试文件、31 个测试通过；新增 Store 与 Workspace 回归覆盖满区、左右插入、重排、删除和键盘调宽。
- `npm run build`：TypeScript 与 Vite production build 通过，tracked `dist/**` 已刷新。
- `GOWORK=off go test ./apps/desktop/... -count=1`：Desktop 与 Desktop MCP package 通过。
- `GOWORK=off wails build -clean -platform windows/amd64`：通过并生成 Windows 生产可执行文件。
- 真实 Wails GUI：连接本地 default-deny Hub 后，`system/catalog` 单面板满区；加入 `system/health` 后为完整双栏；分隔条从 `6:6` 拖到约 `8:4` 正常；左右按钮和标题拖拽均可换位并保持物理比例。
- `git diff --check`：通过；Windows checkout 仅有 LF/CRLF 提示。

## 兼容性、迁移与回滚

- View 文档仍是 version 1，Go validation、Resource 引用和 renderer 设置没有变化；无需后端或 Profile 数据迁移。
- 双栏比例以整数列保存，因此步进为 `1/12`；这是沿用现有存储契约的有意边界，避免引入浮点比例与第二套布局状态。
- 超过四个 Widget 时会按行分块；自由二维画布、跨面板嵌套与任意像素比例不属于本次范围。
- 回滚可恢复本变更涉及的 Workspace、Store、App 拖拽、CSS、测试、tracked dist 和稳定文档；不会删除用户 Profile、View 或资源数据。
- 本次没有设计、替换、定稿或归档品牌图标，也没有 push、release 或 remote 操作。

## 子 Agent 执行轨迹

- 无。当前执行策略未授权主动委派，布局模型、实现、测试、文档与 GUI 验证均由主 agent 顺序完成。
