# 2026-08-31 Desktop Profile 入口生产化

## 变更背景与目标

Desktop 的登录页此前把保存的 Profile、首次连接、手工 Node ID 与信任字段混在同一条表单流程中。用户已确认生产界面应先区分“使用现有 Profile”和“首次连接”，并继续遵守 Admission Authority 下发 Node ID、Permit/Pending 准入和父节点控制语义。

本轮把已确认的高保真方案接入真实 Wails/React 产品，同时补齐受保护 Enrollment 状态的只读投影、非破坏性返回选择器和主线 NodeHost 生命周期兼容。

## 结果

- 有保存记录时默认显示“使用现有 Profile”，无记录时直接进入“首次连接”；移除 01/02/03 编号和常规副标题。
- Authority Profile 按受保护 credential 的 `missing/device/pending/enrolled/error` 真相显示动作，Legacy 保持兼容。
- Profile ID 由客户端随机生成并稳定保存；普通流程不输入 Profile ID、Node ID、父节点 ID 或公钥 pin。
- Permit 路径显式准备受保护设备身份并只展示设备公钥；Permit 只进入本次连接调用。
- Pending 重试复用 request 与已固定观察；Enrolled 直接连接；损坏或身份冲突明确阻止连接。
- “返回 Profile 选择”会持久化清空 active Profile，关闭活动 runtime/subscription/session，但保留 Profile、credential、Views 和 preference。
- 合入最新 `master` 后，Legacy Profile 使用 Parent-only NodeHost；Authority Enrollment 继续通过受限 owning binding 完成 bootstrap/reconnect，未增加 Listener、MCP 或网络权限。

## 文档影响

- Docs root：仓库内 `docs/`。
- Intake：新增并索引 `docs/intake/2026-08-31_desktop-profile-entry-production.md`。
- Feature：更新 `docs/features/desktop.md` 为当前双入口、状态与 NodeHost 行为真相。
- Requirements：更新 `docs/requirements/desktop-resource-workspace.md` 的场景、功能要求和验收条件。
- Specs：新增并索引 `docs/specs/desktop-profile-entry.md`。
- Decision：无需新增 ADR；`2026-08-30_centralized-admission-authority.md` 仍是 Node ID、Permit 和 Authority 所有权依据。
- Lessons：新增 `credential-status-inspection-must-be-read-only.md`，记录只读状态路径不能调用 create-on-missing 的通用约束。

## Task 映射

| Task | 完成内容 | 主要证据 |
| --- | --- | --- |
| `DOC01` | 澄清并索引 Profile 入口稳定契约 | intake、requirements、spec、feature |
| `STATE01` | 非创建式 credential 检查与安全状态 DTO | `runtime/auth`、Desktop Go tests |
| `SESSION01` | 非破坏性 deactivate 与前端权威清理 | Wails API、App tests、GUI smoke |
| `UI01` | Existing Profile / First Connection 生产 UI | 87 个 Vitest、浅/深色截图 |
| `QA01` | 全量、生成、生产包和真实 GUI 验证 | 下述测试结果与 verification 证据 |

## 关键设计与取舍

- Profile ID 是本机存储键，不是网络 Node ID；前者客户端生成，后者只能由 Authority Grant 下发。
- 父节点公钥不是常规输入。没有预置锚时使用一次显式 TOFU，后续 Pending/Enrolled 重试复用已观察信任。
- `settings.json` 不复制 credential 生命周期；列表按需读取受保护状态，避免双真相。
- 读取状态与准备身份严格分离。不存在返回 `missing`，损坏返回 `error`，两者都不得触发静默重建。
- 保留 active Profile 的启动/auto-connect 兼容；只有显式返回选择器后才跨重启停留在选择页。

## 验证结果

- `$env:GOWORK='off'; go test ./... -count=1`：通过。
- `$env:GOWORK='off'; go vet ./...`：通过。
- `scripts/mfh.ps1 -Action generate -Target generated`：通过，随后无 generated drift。
- `npm test`：14 个文件、87/87 测试通过。
- `npm run build`：TypeScript 与 Vite production build 通过。
- `wails build -clean -trimpath -platform windows/amd64 -o mfh-desktop.exe`：通过。
- 合并主线后再次运行上述全量门禁，Desktop Profile 与 NodeHost 生命周期兼容测试通过。
- `$m-test` 已在隔离 DPAPI 状态下验证 missing、Device、Pending、Enrolled、Legacy、deactivate、键盘、浅/深色和 1024×768 packaged GUI。

## 验证证据

- [首次连接](verification/2026-08-31_desktop-profile-entry-first.png)
- [深色 1024×768 Profile 选择](verification/2026-08-31_desktop-profile-entry-chooser-dark.png)

## 可检索排障线索

- 打开 Profile 选择器后意外创建目录：检查状态路径是否调用 load-or-create 或 `MkdirAll`。
- Pending 每次产生新 request：检查重试是否复用受保护 credential 和空 Permit。
- Enrolled 仍显示未注册：检查 Profile projection 是否错误依赖 `settings.node_id`。
- 返回选择器后数据消失：区分 deactivate、disconnect 与 delete 三个边界。
- 主线同步后 `a.client` 编译失败：Desktop runtime 已收敛为 `a.active *profileRuntime`。

## 影响与回滚

- settings version、wire protocol、Authority、Permit、撤销和 headless 流程未改变，无数据迁移。
- 回滚 UI 时可恢复旧 LoginScreen/ProfileEditor，同时保留只读状态 API；回滚 session API 时应先恢复旧 active-runtime 调用方。
- 新增状态读取和 deactivation 均为本机边界；回滚不会删除 Profile、credential 或 Views。
- 本轮只做本地提交与合并，不 push、发布或部署。

## 执行追踪

- 实现与验证由 primary agent 完成。
- 未派发 sub-agent：用户和适用指令未要求委派，且 Go/Wails/React/生成物写集高度重叠。

## 相关文档

- [请求入口](../intake/2026-08-31_desktop-profile-entry-production.md)
- [Desktop 当前行为](../features/desktop.md)
- [Desktop 资源工作区需求](../requirements/desktop-resource-workspace.md)
- [Desktop Profile 入口规格](../specs/desktop-profile-entry.md)
- [集中式 Admission Authority](../decisions/2026-08-30_centralized-admission-authority.md)
- [Credential 状态检查必须是只读路径](../lessons/credential-status-inspection-must-be-read-only.md)
- [完整计划快照](../plan/plan_archive_2026-08-31_desktop-profile-entry-production.md)
