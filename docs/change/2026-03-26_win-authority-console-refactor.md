# 2026-03-26_win-authority-console-refactor

## 变更背景 / 目标
- 将 Win 端 Authority 管理从单一 `Permissions` 页面重构为三类独立入口，分别覆盖权限编排、注册审批、permit 签发。
- 提升权限编辑界面的易读性和操作效率，同时保持后端协议和登录注册主流程不变。

## 具体变更内容
- 稳定文档：
  - 新增 `docs/requirements/authority-admin-console.md`
  - 新增 `docs/specs/authority-admin-console.md`
  - 更新 `docs/requirements/README.md` 与 `docs/specs/README.md`
- 后端：
  - 在 `internal/services/auth/authority.go` 增加 Win 本地 approval / permit typed request/response，避免因 `GOWORK=off` 下依赖固定到旧版 `myflowhub-proto` 而无法生成 bindings。
  - 扩展 `internal/services/permission/service.go`，增加注册审批与 permit 编排接口，并保留原有 policy 能力。
  - 新增 / 更新 `internal/services/auth/authority_test.go`、`internal/services/permission/service_test.go`。
- 前端：
  - 删除旧单页 `frontend/src/pages/Permissions.vue` 和 `frontend/src/stores/permissions.ts`。
  - 新增三页：
    - `frontend/src/pages/AccessPolicy.vue`
    - `frontend/src/pages/RegistrationApprovals.vue`
    - `frontend/src/pages/PermitIssuance.vue`
  - 新增 authority 相关 store：
    - `frontend/src/stores/authority.ts`
    - `frontend/src/stores/accessPolicy.ts`
    - `frontend/src/stores/registrationApprovals.ts`
    - `frontend/src/stores/permitIssuance.ts`
  - 更新 `frontend/src/layout/AppShell.vue`、`frontend/src/router/index.ts`、`frontend/src/i18n/messages/shell.ts`、`frontend/src/i18n/messages/operations.ts`。
  - 保留旧路由兼容：
    - `/permissions` -> `/access-policy`
    - `/approvals` -> `/registration-approvals`
    - `/permits` -> `/permit-issuance`

## Requirements impact
- updated

## Specs impact
- updated

## Lessons impact
- none

## Related requirements
- `repo/MyFlowHub-Win/docs/requirements/authority-admin-console.md`

## Related specs
- `repo/MyFlowHub-Win/docs/specs/authority-admin-console.md`
- `repo/MyFlowHub-Server/docs/specs/auth.md`

## Related lessons
- none

## 对应 plan.md 任务映射
- `DOCS-1`
  - Authority Console 的 requirements/specs 与索引已建立。
- `IMPL-1`
  - `PermissionService` 扩展 approval / permit orchestration，`AuthService` 增加对应 typed helper。
- `IMPL-2`
  - 左侧 Authority 组、三条独立路由、双语文案和旧入口重定向已完成。
- `IMPL-3`
  - `Access Policy` 可用性重构完成；`Registration Approvals` 与 `Permit Issuance` 页面完成。
- `IMPL-4`
  - 测试、构建和 bindings 生成全部通过。

## 经验 / 教训摘要
- 在 `go.work` 不包含 worktree module 的前提下，Wails bindings 生成要和 Go 测试一样显式使用 `GOWORK=off`。
- approval / permit 能力先在 Win 端落本地 typed payload，可以避免为了 GUI 重构同步扩大协议升级范围。
- permit token 只保留最近一次成功签发结果的内存态，默认更安全，也更符合当前无列表协议的边界。

## 可复用排查线索
- 症状：
  - `wails generate module` 提示当前目录所在 module 不在 `go.work` modules 列表中。
- 触发条件：
  - 在 worktree 中直接执行 `wails generate module`，且父级 `go.work` 未包含当前 worktree module。
- 关键词：
  - `wails generate module`
  - `go.work`
  - `GOWORK=off`
- 快速检查：
  - 先执行 `$env:GOWORK='off'; go test ./... -count=1`
  - 再执行 `$env:GOWORK='off'; wails generate module`
  - 若仍失败，再检查 `go.mod` 中 `myflowhub-proto` 版本是否满足当前 bindings 所需结构

## 关键设计决策与权衡
- 采用“单个 authority orchestration service + 三个前端页面/store”而不是新增三套 Wails service。
  - 好处：后端改动面小，authority 解析逻辑不重复。
  - 代价：`PermissionService` 责任扩大，需要靠明确命名控制边界。
- 采用 Win 本地 auth payload，而不是本轮直接升级 `myflowhub-proto`。
  - 好处：不扩大依赖升级范围，worktree 下更稳。
  - 代价：未来若 proto 升级，需要再决定是否收敛回共享类型。
- permit 历史不持久化。
  - 好处：避免敏感 token 落本地配置。
  - 代价：页面只能展示当前会话最近一次成功签发结果。

## 测试与验证方式 / 结果
- 后端：
  - `$env:GOWORK='off'; go test ./... -count=1`
  - 结果：通过
- 前端测试：
  - `npm test`
  - 结果：`7` 个测试文件、`23` 个测试全部通过
- 前端构建：
  - `npm run build`
  - 结果：通过
  - 备注：Vite 继续提示主 bundle chunk size warning，但不影响本轮构建通过
- Wails bindings：
  - `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 备注：输出中仍有重复 `Not found: time.Time` 警告，但命令退出码为 `0`

## 潜在影响
- Authority 管理入口从单页改为三页，用户的旧书签或习惯路径可能变化，但旧路由已做重定向兼容。
- `PermissionService` 能力范围扩大，后续若继续增加 authority 管理动作，应继续按页面和方法语义拆清楚，避免形成新的“超级服务”。
- 当前前端主包体积依旧偏大，后续若继续扩前端功能，可能需要单独 workflow 处理 chunk 拆分。

## 回滚方案
- 恢复旧 `Permissions.vue` / `permissions.ts`，移除新页面、路由和导航分组。
- 回退 `internal/services/auth/authority.go` 及 `PermissionService` 新增审批 / permit 接口。
- 删除本轮新增 requirements/specs/change 文档并恢复索引。

## 子Agent执行轨迹
- none
