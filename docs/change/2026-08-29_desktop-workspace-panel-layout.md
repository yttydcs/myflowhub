# Desktop 工作区满区与可调分栏布局

## 变更背景 / 目标

旧 Workspace 把新加入的资源按固定 `4 × 4` 网格单元放在左上角，单个资源只占很小区域；多个资源虽然
具有 `x/y/w/h`，但缺少明确的平铺规则、直接拖拽换位和相邻面板比例控制。本次把工作区改为有界平铺：
首个资源占满可用区，第二个默认加入右侧，面板可切换左右/上下结构并交换位置，比例可连续调整并随 View
精确保存。

## 文档治理影响

- Docs root: `docs/`
- Intake impact: none；沿用 [可扩展资源平台与 Desktop 工作区重构](../intake/2026-08-28_extensible-resources-and-desktop-workspace-redesign.md) 的 View/Workspace 需求，不新增原始需求副本。
- Feature impact: updated [Desktop](../features/desktop.md)。
- Requirements impact: updated [Desktop 资源工作区](../requirements/desktop-resource-workspace.md)。
- Specs impact: updated [Desktop Resource Workspace v2](../specs/desktop-resource-workspace-v2.md)。
- Decision impact: none；View document 从 version 1 升级为 version 2 并提供确定性 v1 迁移，但没有改变
  Profile 隔离、Resource reference 或 renderer 架构。
- Lessons impact: none；没有形成新的可复用故障模式。
- Index impact: updated `docs/change/README.md`；既有 feature/requirement/spec 叶子未改变索引拓扑。

## 具体变更

- 首个 Widget 规范化为 `x=0, y=0, w=12, h=24`，占满工作区可用宽高。
- 第二个 Widget 默认插到右侧并形成 `6:6` 双栏；拖入目标左/右半侧时按对应方向插入。
- 工作区工具栏提供明确的“左右 / 上下 / 交换”；Widget 标题栏拖动和方向按钮按当前结构重排。
- 左右布局使用 vertical separator，上下布局使用 horizontal separator；比例限制为 `20%`–`80%`，支持
  对应方向键、Home/End、Enter 和双击复位。
- 指针移动只更新 React 本地预览，松手后才向 View 提交一次精确 `split_ratio`，不再按 `1/12` 跳变；
  12 列 `x/w` 只同步近似值用于旧 View 兼容。
- 删除至一个 Widget 后自动恢复满区；三个以上 Widget 使用每行最多四个、总计 24 行单位的有界分块规则。
- View document version 2 增加可选 layout；已知 version 1 文档确认不含 layout 后在内存中迁移，首次保存
  原子写回 v2。没有 layout 的旧双栏从首个 Widget 的 `w/12` 推导 horizontal 比例；旧单 Widget 和非完整
  双栏 View 仍按原规则规范化，三个以上的旧 View 只在发生布局操作后进入新规则。
- 保留缺失资源 detached 状态、renderer 行为、Inspector、Profile 隔离和原子 View store；没有引入新依赖。

## 测试与验证

- `npm test`：7 个测试文件、35 个测试通过；新增 Store 与 Workspace 回归覆盖精确比例、上下切换、交换、
  键盘细调，以及指针移动只预览、释放才提交。
- `npm run build`：TypeScript 与 Vite production build 通过，tracked `dist/**` 已刷新。
- `GOWORK=off go test ./apps/desktop/... ./sdk/bindings/desktop -count=1`：Desktop 与 Desktop binding 通过；
  `go vet ./apps/desktop/... ./cmd/mfh-desktop` 通过。
- `GOWORK=off wails build -clean -platform windows/amd64`：通过并生成 Windows 生产可执行文件。
- 真实 Wails GUI：连接本地 default-deny Hub 后，`file/progress` 单面板满区；加入 `file/transfers` 后为完整
  双栏；“上下”立即切换为纵向平铺，“交换”立即对调两个资源；horizontal separator 从 50% 连续拖到
  约 62%，没有 1/12 跳格。Inspector 打开时发现并修复布局按钮文字被压缩换行的问题。
- `git diff --check`：通过；Windows checkout 仅有 LF/CRLF 提示。

## 兼容性、迁移与回滚

- View 文档仍是 version 1，Go validation、Resource 引用和 renderer 设置没有变化；无需后端或 Profile 数据迁移。
- View document version 2 的 `layout.direction` 与 `layout.split_ratio` 是明确的格式升级；v1 可向前迁移，
  但写回 v2 后旧 Desktop 会按其严格版本门禁拒绝读取。若需回滚旧 Desktop，应同时恢复 View store 的自动
  backup，或先用当前版本移除 v2 数据。比例必须是 `0.2`–`0.8` 的有限数，非法值在 Go 保存边界被拒绝。
- 超过四个 Widget 时会按行分块；自由二维画布、跨面板嵌套与任意像素比例不属于本次范围。
- 回滚可恢复本变更涉及的 Workspace、Store、App 拖拽、CSS、测试、tracked dist 和稳定文档；不会删除用户 Profile、View 或资源数据。
- 本次没有设计、替换、定稿或归档品牌图标，也没有 push、release 或 remote 操作。

## 子 Agent 执行轨迹

- 无。当前执行策略未授权主动委派，布局模型、实现、测试、文档与 GUI 验证均由主 agent 顺序完成。
