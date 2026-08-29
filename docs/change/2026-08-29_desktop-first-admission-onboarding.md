# Desktop 首次准入引导

## 变更背景 / 目标

新的 Desktop 登录页虽然能够提交一次性 Permit，但首次准入的 Permit 必须绑定 Desktop 自身公钥，而旧界面只有在 Profile 已激活后才能读取该身份，形成了“必须先准入才能拿到准入所需公钥”的闭环。本次变更让全新 Profile 可以先在既有受保护凭据存储中准备身份、只展示公钥，再由父 Hub 签发 Permit；成功连接仍是激活 Profile 的唯一入口。

项目品牌图标由独立任务处理。本次没有设计、替换、派生或归档任何图标资产。

## 文档治理影响

- Docs root: `docs/`
- Intake impact: updated
- Feature impact: updated
- Requirements impact: updated
- Specs impact: updated
- Decision impact: none
- Lessons impact: updated
- Related intake: [Desktop 首次准入引导](../intake/2026-08-29_desktop-first-admission-onboarding.md)
- Related features: [Desktop](../features/desktop.md)
- Related requirements: [Desktop 资源工作区](../requirements/desktop-resource-workspace.md)
- Related specs: [Desktop Resource Workspace v2](../specs/desktop-resource-workspace-v2.md)
- Related decisions: 既有 CredentialStore、default-deny 与一次性 Permit 决策保持不变；不新增 ADR。
- Related lessons: [Desktop binding 重连与准入诊断](../lessons/desktop-binding-reconnect-and-admission-diagnostics.md)

## 具体变更内容

- Desktop host 新增 `PrepareProfileJSON`：严格复用 Profile 校验，在当前 per-Profile CredentialStore 中生成或读取稳定身份，仅返回 Node ID 与公钥。
- 设置存储新增显式的 inactive Profile upsert；准备身份不会修改 `active_profile_id`、安装 active client 或建立网络连接。
- React 登录页拆成“准备本机身份”和“提交一次性 Permit”两步，支持复制公钥、复制结果反馈和编辑字段后清除陈旧公钥。
- 已准入的保存 Profile 仍可在 Permit 留空时按原路径登录；登录失败时 Permit 继续留在表单中供修正重试。
- 本地默认端点提示与当前 Hub 默认监听统一为 `127.0.0.1:7331`。
- Wails bindings 与 tracked production `dist` 同步再生成；没有新增依赖或协议格式。

## 任务映射

| Task | 结果 |
| --- | --- |
| ADM01 | 完成 inactive Profile 身份准备、公开响应、稳定性与 active Profile 拒绝测试。 |
| UI01 | 完成两阶段登录、复制反馈、陈旧身份失效和前端测试。 |
| DOC01 | 完成 intake、feature、requirement、spec 与 lesson 更新。 |
| VAL01 | 完成自动化、Windows production build、真实 default-deny Hub 准入与安全状态检查。 |
| ARC01 | 完成本 change、测试截图、plan 归档、索引与本地 closeout。 |

## 关键设计决策与权衡

1. 准备身份不是认证：API 不激活 Profile、不连接网络，也不接受或保存 Permit，避免把准备动作误当成登录成功。
2. 身份只由 CredentialStore 创建和持有：没有浏览器临时密钥、私钥导出或第二套生产身份模型。
3. 先成功打开凭据并读取公开身份，再原子保存 inactive Profile；失败不会留下半完成的 active 设置。
4. Hub 管理保持在信任边界之外：Desktop 只展示公钥和接收一次性 Permit，不内嵌签发或策略管理权限。
5. 保留现有 `LoginJSON` 为唯一首次激活路径，避免持久化 schema 或迁移行为变化。

## 经验 / 教训摘要

准入 UI 不能只暴露 Permit 输入：只要 Permit 绑定客户端密钥，客户端就必须能在未准入、未激活、未连接时从生产 CredentialStore 获得稳定公钥。这个准备状态必须与 active/authenticated/connected 三种状态明确分开。

真实 default-deny 验收还应区分 Variable 的 `read` 与 `subscribe` 权限。只授予 subscribe 不能满足 Desktop 首次 topology/catalog 快照读取；测试 Hub 的离线策略修改必须在 Hub 停止时串行执行，避免状态文件锁或双写。

## 可复用排查线索

- 症状：首次登录不知道公钥，但 Permit 又要求公钥。触发条件：全新 Profile。关键词：`PrepareProfileJSON`、inactive Profile、CredentialStore。
- 症状：准备身份后直接进入空工作区。快速检查：准备后 `active_profile_id` 应仍为空，`IdentityJSON` 不应有 active client。
- 症状：连接成功但 topology 返回 forbidden。快速检查：主体对 `system/topology`、`system/catalog` 是否具有 `read`；实时订阅另需 `subscribe`。
- 症状：离线策略命令提示状态文件被占用。快速检查：Hub 是否已停止，以及是否并行运行多个写命令。
- 症状：本地连接被拒绝。快速检查：当前 Hub 默认端点是 `127.0.0.1:7331`，不是旧演示占位值 `9540`。

## 测试与验证

- `npm test`: 4 个文件、20 个测试通过。
- `npm run build`: TypeScript 与 Vite production build 通过。
- `GOWORK=off go test ./apps/desktop/...`: 通过。
- `GOWORK=off go test ./... -count=1`: 全仓 package 与 integration tests 通过。
- `gofmt -d`（变更 Go 文件）与 `git diff --check`: 通过。
- `wails build -clean -platform windows/amd64`: Windows/amd64 production executable 构建通过。
- 实机链路：准备 Node 2 稳定公钥 → 父 Hub 离线签发 one-use Permit → 启动 `127.0.0.1:7331` → 登录 → 加载 2 Nodes / 25 Resources → 预览 `system/health` 运行状态。
- 安全检查：`%APPDATA%/MyFlowHub/desktop-resource-workspace/settings.json` 只在登录成功后将 `local-dev` 设为 active，且不包含 Permit 或 private key。
- 窗口证据：[已连接的真实资源树与 system/health 预览](verification/2026-08-29_desktop-first-admission-connected.png)。

## 潜在影响、迁移边界与回滚

- 现有 Profile、DPAPI 身份、View 和 UI preference 无需迁移；已准入 Profile 的登录行为保持兼容。
- 新增 host 方法是窄化扩展，未改变 wire protocol、Permit schema、Hub policy 或 CredentialStore 数据格式。
- 回滚可恢复 Desktop host/config、登录组件、bindings 与 dist；已经准备但未准入的 Profile 可在登录页删除，删除仍走既有凭据清理路径。
- 验收 Hub 状态位于仓库外的隔离目录 `D:\project\MyFlowHub3\.tmp\desktop-onboarding-demo\hub`，不属于生产数据。本地 Hub 保持运行供用户查看；未推送、发布或创建 remote。

## 子 Agent 执行轨迹

- 无。Go API、生成 binding、React 状态与安全验收共享同一契约面，本 workflow 由主 agent 顺序完成。
