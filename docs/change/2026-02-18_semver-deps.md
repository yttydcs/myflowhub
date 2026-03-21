# 2026-02-18 Semver 依赖化（PR19）

## 变更背景 / 目标
在多仓协作开发期，为了方便联调，SDK/Server/Win 常使用本地 `replace ../MyFlowHub-*`。但这种方式会导致：
- 单仓 clone 后无法构建（依赖本地目录结构）；
- 依赖不可审计、不可复现（外部无法拉取同样的版本基线）。

本次 PR19 目标：
1) 发布可拉取的 semver 依赖基线：
   - `myflowhub-core@v0.2.0`
   - `myflowhub-proto@v0.1.0`
   - `myflowhub-sdk@v0.1.0`
2) SDK/Server/Win 移除本地 `replace`，保证 **`GOWORK=off go test ./...`** 可直接通过。
3) 保留本地多仓联调体验：使用不提交的 `d:\project\MyFlowHub3\go.work` 替代 replace。

## 具体变更内容（按仓库）
### MyFlowHub-Core
- 新增发布归档文档：`docs/change/2026-02-18_core-v0.2.0.md`
- 发布 tag：`v0.2.0`（作为 HeaderTcp v2 / Core 路由规则统一后的上游基线）

### MyFlowHub-Proto
- 新增发布归档文档：`docs/change/2026-02-18_proto-v0.1.0.md`
- 发布 tag：`v0.1.0`（首次可拉取版本）

### MyFlowHub-SDK
- `go.mod/go.sum`：移除本地 `replace`，依赖固定为 `core v0.2.0 + proto v0.1.0`
- 新增发布归档文档：`docs/change/2026-02-18_sdk-v0.1.0.md`
- 发布 tag：`v0.1.0`

### MyFlowHub-Server
- `go.mod/go.sum`：移除本地 `replace`，依赖固定为 `core v0.2.0 + proto v0.1.0`
- 新增归档文档：`docs/change/2026-02-18_server-semver-deps.md`

### MyFlowHub-Win
- `go.mod/go.sum`：移除本地 `replace`，依赖固定为 `core v0.2.0 + proto v0.1.0 + sdk v0.1.0`
- 新增归档文档：`docs/change/2026-02-18_win-semver-deps.md`

### Workspace（不提交）
- 新增：`d:\project\MyFlowHub3\go.work`（仅本地联调用）

## plan.md 任务映射
- SEM1：Core：准备发布 v0.2.0 ✅
- SEM2：Proto：准备发布 v0.1.0 ✅
- SEM3：SDK：移除 replace + 发布 v0.1.0 ✅
- SEM4：Server：去 replace（依赖 tag）✅
- SEM5：Win：去 replace（依赖 tag）✅
- SEM6：Workspace：创建本地 go.work（不提交）✅

## 关键设计决策与权衡
- 采用 **semver tag** 作为“可拉取依赖基线”，用于发布/审计/复现。
- 采用 **go.work（不提交）** 作为“本地联调层”，避免仓库内长期保留 replace。
- 验收强制使用 **`GOWORK=off`**，防止 go.work 在 CI/他人环境中掩盖依赖缺失问题。
- tag 一旦 push **不回滚/不改写**；发现问题通过更高 patch 版本修复。

## 测试与验证（结果：通过）
以下命令均在对应仓库执行（必须 `GOWORK=off`）：
- `go test ./... -count=1 -p 1`

## 潜在影响
- 本地开发习惯需要迁移：从 `replace` 改为 go.work（或直接使用已发布的 tag）。
- 若误打 tag，将影响所有下游；需通过追加更高 patch 版本修复（不可强行重写历史）。

## 回滚方案
- 对下游（SDK/Server/Win）：可通过 `git revert` 恢复 `replace`（仅建议临时救火）。
- 对已发布 tag：不删除/不改写；通过发布更高版本修复问题。
