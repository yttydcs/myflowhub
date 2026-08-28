# 2026-08-23 MyFlowHub 本地 worktree 与分支平衡清理

## 变更背景 / 目标

MyFlowHub3 本地存在大量已经合入主线的历史分支，以及一组只残留旧控制文档、且 commit 已由主线、tag 或远程跟踪引用保护的 worktree。本 workflow 采用用户批准的“平衡清理”边界：只清理精确批准的低风险本地对象，保留独有成果、外部项目、远程分支与 tags。

## 具体变更内容

- `PRE-1`：Passed；完成 12 个目标、13 条允许残留状态、18 个恢复引用、281 个合并分支候选和保留集合复核。
- `ARC-1`：已完成删除前归档、恢复账本、索引与 reusable lesson。
- `WT-1`：Passed；12/12 个批准 worktree 已由 owning repo 的 Git API 移除。
- `BR-1`：Passed；12/12 个关联本地分支已删除，其中 4 个使用 `-d`，8 个在 tag/remote recovery 门禁通过后使用受控 `-D`。
- `BR-2`：Changed；冻结的 281 个候选中 278 个由 `git branch -d` 删除，3 个安全拒绝项保留，未升级为 `-D`。
- `VAL-1`：Passed；目标、分支、保留集合、主工作区、外部 worktree、目录和 prune dry-run 均已验证。
- `DOC-1`：Passed；活动计划 worktree 已移除，本地计划分支使用 `-d` 删除。

## Docs root

- `D:\project\MyFlowHub3\docs`
- 本文档及索引修改为 local-only。
- 不执行 stage、commit、merge、push、remote publication 或 backup target 选择。
- canonical Docs Root 已有用户未提交修改；本 workflow 只添加精确的新文件与索引条目，不重置既有内容。

## Intake impact

- Intake impact: none

## Feature impact

- Feature impact: none

## Requirements impact

- Requirements impact: none

## Specs impact

- Specs impact: none

## Decision impact

- Decision impact: none

## Lessons impact

- Lessons impact: updated
- 新增 `docs/lessons/powershell-nested-array-flattening.md`，记录 PowerShell 嵌套数组/管道自动展开导致安全预检误报的复用规则。

## Related intake

- none

## Related features

- none

## Related requirements

- none

## Related specs

- none

## Related decisions

- none

## Related lessons

- [powershell-nested-array-flattening.md](../lessons/powershell-nested-array-flattening.md)

## Related plan

- [plan_archive_2026-08-23_worktree-branch-balanced-cleanup.md](../plan/plan_archive_2026-08-23_worktree-branch-balanced-cleanup.md)

## 对应 `plan.md` 任务映射

- `PRE-1`：只读安全预检，已 Passed。
- `ARC-1`：删除前归档、索引、lesson 与无关脏状态保护。
- `WT-1`：精确移除 12 个批准 worktree。
- `BR-1`：删除 12 个关联本地分支。
- `BR-2`：删除 281 个已合并未挂载分支。
- `VAL-1`：跨仓状态与保留集合验证。
- `DOC-1`：结果回填和活动计划 worktree/branch 收尾。
- `REV-1`、`EXT-1`、`REMOTE-1`：未执行，分别延期或超出范围。

## 经验 / 教训摘要

- Git 清理必须以 owner repo、绝对路径、branch、tip SHA、状态和恢复引用的组合门禁为准，不能仅依赖目录名。
- PowerShell 函数/管道会枚举并展开集合；安全脚本需要显式对象/映射结构，不能把“嵌套数组形状”当作稳定契约。
- 已合并分支使用 `git branch -d`，失败即保留；只有已由精确 tag/remote ref 保护的 12 个关联分支允许在复核后使用 `-D`。
- canonical Docs Root 的既有脏状态不应被为了归档而整体暂存或提交。

## 可复用排查线索

- 症状：候选数量、目标数量或状态数组在数据正确时仍出现离谱偏差。
- 触发条件：PowerShell 函数返回数组，随后进入另一层数组、管道或属性赋值。
- 关键词：`PowerShell`、`nested array`、`pipeline enumeration`、`array flattening`、`Count drift`。
- 快速检查：打印元素运行时类型和 `Count`；改用 `[pscustomobject]` / hashtable 映射后重跑只读门禁。

## 关键设计决策与权衡

1. 先归档再删除：在任何 worktree removal 前保留批准范围、tip SHA 与恢复账本。
2. 精确 worktree API：只使用 owning repo 的 `git worktree remove --force <exact-path>`，不递归删除目录。
3. 两类分支分开处理：
   - 12 个关联分支按恢复引用决定 `-d` 或受控 `-D`；
   - 281 个合并分支只允许 `-d`。
4. 状态漂移 fail closed：任何 owner、branch、tip、状态、哈希或恢复引用变化都会停止。
5. 文档保持 local-only：当前 Docs Root 已脏，不将用户修改混入 workflow commit。

## 测试与验证方式 / 结果

### 删除前门禁

- 12/12 个目标 worktree owner/path/branch/tip 匹配。
- 13/13 条允许残留文件状态与 SHA-256 匹配。
- 18/18 个恢复引用存在且仍包含对应 target tip。
- 281/281 个已合并未挂载候选匹配；本地分支总数 340。
- 15 个保留 MyFlowHub worktree、11 个主工作区、20 个外部注册 worktree已冻结签名。
- 非目标顶层目录快照：48 个。
- 清理结果：12 个目标 worktree和12个关联分支已删除；281 个合并候选中删除 278 个、保留 3 个 `-d` 拒绝项。
- 验证结果：Passed；目标清理后、活动计划收尾前本地分支总数 50；15 个保留 MyFlowHub worktree、11 个主工作区、20 个外部注册 worktree、48 个非目标顶层目录均保持，11 仓 prune dry-run 无计划外条目。

### `git branch -d` 安全拒绝项

| Repo | Branch | Tip SHA | Result |
| --- | --- | --- | --- |
| Android | `chore/android-deps-sync` | `c9c55dfb9b66e38d48105e921bf1297a49d30d6a` | Preserved；upstream 未包含 tip |
| MetricsNode | `chore/metricsnode-deps-sync` | `0f5857e1684a769f0d51f94eaf97d7131816d3f9` | Preserved；upstream 未包含 tip |
| Win | `chore/win-deps-sync` | `b93c5cdbcafcd37ddb01e4a931b84053a829f787` | Preserved；upstream 未包含 tip |

### 归档与控制面收尾

- 活动计划 worktree：`D:\project\MyFlowHub3\worktrees\chore-worktree-branch-cleanup`，Removed。
- 活动计划分支：`chore/worktree-branch-cleanup`，使用 `git branch -d` 删除。
- 计划分支 tip：`c238aa8aec2c97cd8623439e2ca923a0fc1b2a62`，删除前与控制仓 `master` 完全一致，无独有提交。
- Final local branch count: 49。
- 相对初始化时的 340 个本地分支，共减少 291 个：
  - 12 个批准关联分支；
  - 278 个 `BR-2` 合并候选；
  - 1 个活动计划分支。
- Merge status: not applicable；workflow 没有产品代码提交，归档直接写入 canonical Docs Root。
- Publication status: local-only；未 stage、commit、merge、push 或选择 backup target。

### 每仓候选计数

| Repo | Base | Frozen candidates | Expected |
| --- | --- | ---: | ---: |
| Control | `master` | 5 | 5 |
| Android | `main` | 25 | 25 |
| ClipboardNode | `master` | 2 | 2 |
| Core | `master` | 14 | 14 |
| EmbeddedSDK | `main` | 5 | 5 |
| MetricsNode | `main` | 25 | 25 |
| Proto | `main` | 12 | 12 |
| SDK | `main` | 13 | 13 |
| Server | `main` | 55 | 55 |
| SubProto | `main` | 31 | 31 |
| Win | `main` | 94 | 94 |

## 潜在影响

- 目标 worktree 中的 13 个精确控制文档/ignore 残留会随 worktree 删除，无法从文件系统直接恢复。
- 本地 branch refs 删除后需要依据下方 tip SHA、base、tag 或 remote tracking ref 重建。
- 不影响远程 refs、tags、外部项目、15 个保留 MyFlowHub worktree或任何产品文件。

## 回滚方案

- 目标 worktree：`git worktree add <original-path> <original-branch>`。
- 关联分支：`git branch <original-name> <recorded-tip-sha>`。
- 281 个合并分支：按候选恢复账本执行同样的 `git branch` 命令；所有 tip 都是对应 base 的祖先。
- 文档：仅移除本文、计划归档、lesson 及三个新增索引条目，不重置既有 Docs Root 修改。

## 子Agent执行轨迹

- none；本次归档与破坏性步骤全部由主 Agent 串行执行。

## 删除前目标恢复账本

| Owner | Worktree path | Local branch | Tip SHA | Recovery refs |
| --- | --- | --- | --- | --- |
| Android | `D:\project\MyFlowHub3\worktrees\release-all-20260412-android` | `chore/release-chain-20260412-android` | `2f0f11b5707190821893bed59489a90c0f6537c1` | `origin/chore/release-chain-20260412-android`, `v0.1.31` |
| Core | `D:\project\MyFlowHub3\worktrees\release-all-20260412-core` | `chore/release-chain-20260412-core` | `536b3d56e83984bd9fa64e8964c6b7937be69222` | `master` |
| MetricsNode | `D:\project\MyFlowHub3\worktrees\release-all-20260412-metrics` | `chore/release-chain-20260412-metrics` | `a5e14fbe4e4fa7573f586a5277293ede24406766` | `origin/chore/release-chain-20260412-metrics`, `v0.1.3` |
| Proto | `D:\project\MyFlowHub3\worktrees\proto-server-release-align` | `chore/server-release-align` | `fe2f79004e10dc845b0dbea84667766fefe12c06` | `v0.1.6` |
| Proto | `D:\project\MyFlowHub3\worktrees\proto-stream-subproto` | `feat/proto-stream-subproto` | `73ec78c0e09b54602aeb1cc3c8073505202cd035` | `main` |
| Proto | `D:\project\MyFlowHub3\worktrees\release-all-20260412-proto` | `chore/release-chain-20260412-proto` | `c8f0352e8a512612caf827fe7f46324eef52f985` | `main` |
| SDK | `D:\project\MyFlowHub3\worktrees\release-all-20260412-sdk` | `chore/release-chain-20260412-sdk` | `27e7476d5ab36517f40cb1c1aad535eb52181533` | `origin/chore/release-chain-20260412-sdk`, `v0.1.14` |
| Server | `D:\project\MyFlowHub3\worktrees\release-all-20260412-server` | `chore/release-chain-20260412-server` | `2b0d0a311a66d107dad424c67dfdb78e1f3985d4` | `main` |
| SubProto | `D:\project\MyFlowHub3\worktrees\release-all-20260412-subproto` | `chore/release-chain-20260412-subproto` | `1917fdd10a542744c48ac2815620ab86eb4ca1cb` | `origin/chore/release-chain-20260412-subproto`, `auth/v0.1.6` |
| SubProto | `D:\project\MyFlowHub3\worktrees\subproto-server-release-align` | `chore/server-release-align` | `3f61a6e146a31daf78cf516d5603abf5fdeb493f` | `flow/v0.1.4` |
| SubProto | `D:\project\MyFlowHub3\worktrees\subproto-stream-subproto` | `feat/subproto-stream-subproto` | `77ff6c8434120b7d471d122696dcca63fe056b88` | `origin/feat/subproto-stream-subproto`, `stream/v0.1.0` |
| Win | `D:\project\MyFlowHub3\worktrees\release-all-20260412-win` | `chore/release-chain-20260412-win` | `77a21fd75dcd919c7d3a20da686bcf8930cc0a5e` | `origin/chore/release-chain-20260412-win`, `v0.0.16` |

## 主工作区状态签名

| Repo | Path | Status entries | Status SHA-256 |
| --- | --- | ---: | --- |
| Control | `D:\project\MyFlowHub3` | 3015 | `01a53b26900365d61ed386403ab56f84ba22a7fa323f0386b314896fdfe5ac25` |
| Android | `D:\project\MyFlowHub3\repo\MyFlowHub-Android` | 0 | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| ClipboardNode | `D:\project\MyFlowHub3\repo\MyFlowHub-ClipboardNode` | 0 | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| Core | `D:\project\MyFlowHub3\repo\MyFlowHub-Core` | 0 | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| EmbeddedSDK | `D:\project\MyFlowHub3\repo\MyFlowHub-EmbeddedSDK` | 0 | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| MetricsNode | `D:\project\MyFlowHub3\repo\MyFlowHub-MetricsNode` | 29 | `dfe45817a3cf2d447769d29bea06f6a5654621e14f2e408ec1c2b906353d6bf4` |
| Proto | `D:\project\MyFlowHub3\repo\MyFlowHub-Proto` | 0 | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| SDK | `D:\project\MyFlowHub3\repo\MyFlowHub-SDK` | 0 | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| Server | `D:\project\MyFlowHub3\repo\MyFlowHub-Server` | 0 | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| SubProto | `D:\project\MyFlowHub3\repo\MyFlowHub-SubProto` | 0 | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| Win | `D:\project\MyFlowHub3\repo\MyFlowHub-Win` | 0 | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |

## 保留 MyFlowHub worktree 状态签名

| Owner | Path | Branch | Head | Status entries | Status SHA-256 |
| --- | --- | --- | --- | ---: | --- |
| Control | `D:/project/MyFlowHub3/worktrees/esp32-embedded-sdk-plan` | `feat/esp32-embedded-sdk-plan` | `4568f20675da0a889b97473978d3ea7188b126e4` | 5 | `c107bed8d601a5646cc4a6587869a5028fa112bbcce53f98c6fb5abc80c08e1c` |
| Control | `D:/project/MyFlowHub3/worktrees/feat-embedded-demo-flows` | `feat/embedded-demo-flows` | `c238aa8aec2c97cd8623439e2ca923a0fc1b2a62` | 9 | `d1353913284e7fc97bf744a79e3a4dca24242de0321d8df8428e9a25a6cd5829` |
| Control | `D:/project/MyFlowHub3/worktrees/feat-local-vars-control` | `feat/local-vars` | `ffbaefacd251c851d2997c186ca6e01fe8cc53a2` | 1 | `33985b968bd2f40cba948cf1692c8351f39239c62b2426705093fce2df852a32` |
| Control | `D:/project/MyFlowHub3/worktrees/fix-win-deploy-graph-check` | `fix/win-deploy-graph-check` | `26339118492fff4e93b6d9361e23348295bf53e9` | 3 | `85928852333dfd5b2545a6f75ed8cad88a638b68b2abe9c0f4aff1d14166b83a` |
| Control | `D:/project/MyFlowHub3/worktrees/release-all-20260412-control` | `chore/release-chain-20260412-control` | `93d27fc3bd46adac6aecb5abea6588b5e53064a1` | 0 | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| Android | `D:/project/MyFlowHub3/worktrees/MyFlowHub-Android-refactor-sdk-auth-typed` | `refactor/sdk-auth-typed` | `a7d3be194621a5eaae81514c096568b9505a79aa` | 10 | `e086a6546c20cc2ce4f47c5b338baa4f3c85b5187e67c6b17204c7bff2570d51` |
| EmbeddedSDK | `D:/project/MyFlowHub3/worktrees/fix-embedded-dht11-interval` | `fix/embedded-dht11-interval` | `9605b74d01004599904237f9d14854b0b5bf9394` | 36 | `7f1e070b69525977c82e1053cf80a3615b11954d8a34889ad45de5d019910307` |
| MetricsNode | `D:/project/MyFlowHub3/worktrees/MyFlowHub-MetricsNode-fix-wails-bindings-build` | `fix/metricsnode-wails-bindings-build` | `0f5857e1684a769f0d51f94eaf97d7131816d3f9` | 9 | `7908b3477f0f0b1f18ba57ffc94d37a255d3e36c2d55de7010c9d52ad017a0d8` |
| Proto | `D:/project/MyFlowHub3/worktrees/proto-local-vars-observability` | `feat/local-vars-proto` | `cb8ea4a1af333d9a0a62bf2a6a2709f93872946d` | 4 | `50de0f9de4471203bbe066e3c217aaf46d856b845dfbf2d0d39e82e0e29c71b3` |
| Server | `D:/project/MyFlowHub3/worktrees/chore-protocol-capability-review` | `chore/protocol-capability-review` | `9fc6ab6eef1cd3abc45ccff0bb48581490555670` | 3 | `6069d9e9471fb486dc6fa79410c374399c05ee274fbd6593649c477149654af9` |
| Server | `D:/project/MyFlowHub3/worktrees/server-local-vars-docs` | `feat/local-vars-server` | `47b48bef04466ebe00b4880cde1112c7e5d1993c` | 5 | `667f6cdb6e2c34574bc49c3d65ed3c872af5d8f7a14f0295c9441d40b85aa63b` |
| SubProto | `D:/project/MyFlowHub3/worktrees/subproto-local-vars-runtime` | `feat/local-vars-runtime` | `486fb2a5fbf287bead5ef13ee6f7366ca48515e5` | 13 | `6ddabe38b1d25c77409305705d149d0a6d9acb8b3773226ef0db4ba08c363352` |
| Win | `D:/project/MyFlowHub3/worktrees/fix-win-deploy-graph-check-win` | `fix/win-deploy-graph-check` | `9774216fd80797c47190f3968c8e8af4672b13df` | 2 | `de0a911543a0f1f8b131b261816ea08f5e88f4ca150e207c39e8d3c40117db68` |
| Win | `D:/project/MyFlowHub3/worktrees/fix-win-stream-content-type-sync` | `fix/win-stream-content-type-sync` | `aa03009ea211435d1cbfc6d990fca31e44480749` | 1 | `09e0f91e3bd5599655497add4a8d488dfe81710def440a5eb7fe3c2ec842b962` |
| Win | `D:/project/MyFlowHub3/worktrees/MyFlowHub-Win-refactor-sdk-auth-typed` | `refactor/sdk-auth-typed` | `cfac0c89db5dac90eb3e367d013be3ab4cdc796a` | 13 | `25224747f3f30f509c87027e34a516f24bd80e05838f97da5cdb91d9958ecd85` |

## 外部注册 worktree 快照

| Path | Branch | Head | Common dir |
| --- | --- | --- | --- |
| `D:\project\MyFlowHub3\worktrees\feat-view-team-transfer-assets\monkeys-ui-admin` | `feat/view-team-transfer-assets` | `4e79a4a54871b830b4218a84ee9e4c95ce904001` | `D:/project/monkeys/repo/monkeys-ui-admin/.git` |
| `D:\project\MyFlowHub3\worktrees\feat-data-bucket-roles\monkeys-server` | `feat/data-bucket-roles` | `a812fb778f1e3fd4973efd1dac453f42e6245a0c` | `D:/project/monkeys/repo/monkeys-server/.git` |
| `D:\project\MyFlowHub3\worktrees\feat-data-bucket-roles\monkeys-ui-admin` | `feat/data-bucket-roles` | `563dfee32fd0743c4d8237394e086f64af2fd05e` | `D:/project/monkeys/repo/monkeys-ui-admin/.git` |
| `D:\project\MyFlowHub3\worktrees\feat-view-team-transfer-assets\monkeys-data-server` | `feat/view-team-transfer-assets` | `132a006539bdee2a5150aacafe831f86e5992443` | `D:/project/monkeys/repo/monkeys-data-server/.git` |
| `D:\project\MyFlowHub3\worktrees\aiwo-go-platform-foundation` | `feat/go-platform-foundation` | `cb27d54234621da18269cb37839e8ebe4dd7bbb5` | `D:/project/ai-workflow-orchestrator/.git` |
| `D:\project\MyFlowHub3\worktrees\refactor-data-browser-remove-paired-special-ui\monkeys-studio` | `refactor/data-browser-remove-paired-special-ui` | `284954b1d1addb4ebfbee9efe0803e671dfd2f8e` | `D:/project/monkeys/repo/monkeys-studio/.git` |
| `D:\project\MyFlowHub3\worktrees\feat-gemini-region-preserve` | `feat/gemini-region-preserve` | `c3bf48d02be144d422a5fb8d5f27f8f5141eb37a` | `D:/project/monkeys/repo/monkey-tools-third-party-api/.git` |
| `D:\project\MyFlowHub3\worktrees\feat-view-team-transfer-assets\monkey` | `feat/view-team-transfer-assets` | `59d0d1b2fe05d7e28bd76acf6acdc03c560bc04e` | `D:/project/monkeys/repo/monkey/.git` |
| `D:\project\MyFlowHub3\worktrees\fix-data-container-items-v2-repair-migration\monkey` | `fix/data-container-items-v2-repair-migration` | `a808862dad3f8bfaea02dd7aeab86224f776373e` | `D:/project/monkeys/repo/monkey/.git` |
| `D:\project\MyFlowHub3\worktrees\feat-data-browser-config-profiles\monkeys-studio` | `feat/data-browser-config-profiles` | `96ed6e62d6c86bf96c37bea98b8d6636ae793d37` | `D:/project/monkeys/repo/monkeys-studio/.git` |
| `D:\project\MyFlowHub3\worktrees\refactor-data-browser-favorites-api\monkey` | `refactor/data-browser-favorites-api` | `583908e5eca1ae80e487ff00dfed2d62c4fbae45` | `D:/project/monkeys/repo/monkey/.git` |
| `D:\project\MyFlowHub3\worktrees\fix-techpack-history-visibility\monkeys-studio` | `fix/techpack-history-visibility` | `522caee6376ba72880a514dbad43fe7633710766` | `D:/project/monkeys/repo/monkeys-studio/.git` |
| `D:\project\MyFlowHub3\worktrees\refactor-bsd-generic-ui\monkeys-studio` | `refactor/bsd-generic-ui` | `89d300b646ad4a271c8b09ac560013e5cb497784` | `D:/project/monkeys/repo/monkeys-studio/.git` |
| `D:\project\MyFlowHub3\worktrees\feat-data-browser-config-profiles\monkeys-server` | `feat/data-browser-config-profiles` | `8fa6c3f0821aaac68a61b9b5061db830c27f93c3` | `D:/project/monkeys/repo/monkeys-server/.git` |
| `D:\project\MyFlowHub3\worktrees\feat-bucket-display-components\monkeys-ui-admin` | `feat/bucket-display-components` | `43fb51947aa205b0bda355c99cde206967b791fb` | `D:/project/monkeys/repo/monkeys-ui-admin/.git` |
| `D:\project\MyFlowHub3\worktrees\feat-bucket-display-components\monkeys-server` | `feat/bucket-display-components` | `e791ff937545aab69b53773671472d293f09e302` | `D:/project/monkeys/repo/monkeys-server/.git` |
| `D:\project\MyFlowHub3\worktrees\feat-bucket-display-components\monkeys-data-server` | `feat/bucket-display-components` | `fe0a0def82b210d476921d300f44f86fffa69b99` | `D:/project/monkeys/repo/monkeys-data-server/.git` |
| `D:\project\MyFlowHub3\worktrees\feat-lf-asset-default-views\monkey` | `feat/lf-asset-default-views` | `7eb52684068331f8ab70019440eb1e6204e28781` | `D:/project/monkeys/repo/monkey/.git` |
| `D:\project\MyFlowHub3\worktrees\refactor-bsd-generic-ui\monkeys-server` | `refactor/workflow-builtin-custom-options` | `7c536510b4f1beaff7d11a9f013f6107421ae8fa` | `D:/project/monkeys/repo/monkeys-server/.git` |
| `D:\project\MyFlowHub3\worktrees\feat-bucket-display-components\monkeys-studio` | `feat/bucket-display-components` | `294b36661ee765181510f7bf361afb209dd93bec` | `D:/project/monkeys/repo/monkeys-studio/.git` |

## 281 个已合并分支恢复账本

| Repo | Base | Branch | Tip SHA |
| --- | --- | --- | --- |
| Control | `master` | `chore/deploy-claude-code-hub` | `4a55cd733db7cf45d6e2de37f5e5143007cf7106` |
| Control | `master` | `chore/flow-completeness-review` | `fd52ffb84607ae4aa2bfb6eb5facc5e3dc53ddbf` |
| Control | `master` | `chore/root-docs-governance` | `9cbb9d9aa00545aa06b84c60f77051484910c4cc` |
| Control | `master` | `chore/subproto-capability-review` | `7d4434a37fdc70cf7db4cb6c105fc737186f6231` |
| Control | `master` | `feat/node-display-name-followup` | `21290df371f39b69224b27e9b578ded240a59b71` |
| Android | `main` | `chore/android-bump-auth-v0.1.1` | `e19d9abef3aa02784ef4881c8e3c75c1bd50e4b7` |
| Android | `main` | `chore/android-bump-management-v0.1.2` | `6708fa296c8b7058eb0d9555faf91726d0e2b9a6` |
| Android | `main` | `chore/android-bump-varstore-v0.1.1` | `32c14fcc0aa3c0c8eda3db221512d80520002b7f` |
| Android | `main` | `chore/android-debug-release-apk` | `669235916c149ae1aacdc0ed2fa4e2b4894705f9` |
| Android | `main` | `chore/android-deps-sync` | `c9c55dfb9b66e38d48105e921bf1297a49d30d6a` |
| Android | `main` | `chore/bump-core-v0.3.0-android-hubmobile` | `9aa4d9a236f489f6f19b09d9069dcce464de9d48` |
| Android | `main` | `chore/bump-core-v0.4.0-downstream` | `f9f4777c819fa640e16665f28b10d95d1ee4ac33` |
| Android | `main` | `chore/bump-sdk-varstore-v0.1.2` | `16256ddf9e1662d6d9e965bc9e40fc15591ab4f9` |
| Android | `main` | `chore/release-auth-route-index-heal` | `16256ddf9e1662d6d9e965bc9e40fc15591ab4f9` |
| Android | `main` | `chore/rfcomm-release-deps` | `94780d023c18f65c90de84b3dacdf347a150c990` |
| Android | `main` | `feat/android-apk-release` | `448268799bce2c8f8f50c1b62342ad727a6fbd98` |
| Android | `main` | `feat/android-file-module` | `77c6d77e71f35d9244da76012321084aaaa371fb` |
| Android | `main` | `feat/android-hub-m0` | `7eb0437d836ab04a181c1a91b882bf89695feebf` |
| Android | `main` | `feat/android-login-devices` | `1b4ff8e9858def375bf902e56a6fa9505b6398ad` |
| Android | `main` | `feat/android-nav-icons` | `b13a37559c19b30dc0ec78a52f3467b33446c686` |
| Android | `main` | `feat/android-topicbus` | `727b2f7c502a9665da70bb5e285b18ef323ae42e` |
| Android | `main` | `feat/android-ui-feedback-drawer` | `b5355c30dddda82343fece9773dc62c9903164b3` |
| Android | `main` | `feat/android-varstore` | `cd2d6ba370bab4caf87ebc4b8c7b6087fb07581b` |
| Android | `main` | `feat/bluetooth-rfcomm-transport` | `6e501c5bb35ebc1612bfb992f9e3d013791aabd4` |
| Android | `main` | `fix/android-ci-debug-release-assets` | `c26c52ea9a92231aa964c232defc43aff04c9309` |
| Android | `main` | `fix/android-ci-gomobile-androidapi` | `e59fd0a08b8d9e8fb52fc59db47e3fb3ec6e5310` |
| Android | `main` | `fix/android-devices-listnodes-error` | `015254d9bcef120cf63d0eeb0d53bb5499dbb040` |
| Android | `main` | `fix/android-fgs-type-gomobile-reflect` | `9fcf46c6f5ea498cc098caae1592bae6e1372bc5` |
| Android | `main` | `fix/android-nodevars-list-hub-target` | `47c2c9daa8d0a8bfa9020915dc68f25142829ec3` |
| Android | `main` | `fix/android-release-gomobile-pin` | `28c106921a261ca44d240d01b5c2f04fb284f6d1` |
| ClipboardNode | `master` | `chore/debug-latest-all-platforms` | `6a8c582551287a65283e337fe173eee9c1d6749f` |
| ClipboardNode | `master` | `feat/full-platform-clipboard-sync` | `18cfea51f22dcb1303571b23510693a3d0bccb1d` |
| Core | `master` | `chore/bootstrap-dialer-release-align` | `536b3d56e83984bd9fa64e8964c6b7937be69222` |
| Core | `master` | `chore/core-semver` | `a838345e9f05790ab0edc1d9e5f3b18b2524a0e0` |
| Core | `master` | `feat/auth-admission-control` | `b83897cdfe98cd99768795515e98762629a831a7` |
| Core | `master` | `feat/bluetooth-rfcomm-transport` | `1318ce0c0e6d9f3fd334b4d10a324353bd2d4129` |
| Core | `master` | `feat/quic-transport-core` | `d992975ec6ad39b6894195d140da84f60790bcf9` |
| Core | `master` | `feat/run-control-phase1` | `c205ea25b638c778aee436ffae481e6ce066d30e` |
| Core | `master` | `fix/rfcomm-win-abort` | `7d321bbae4d5d2a2e0fe2dc8cc977bef812ec19c` |
| Core | `master` | `fix/rfcomm-win-stream-read` | `e964e0b32d29516ced37a3d3864a5b017944bcf3` |
| Core | `master` | `fix/rfcomm-write-contract` | `73938c9f3f64e91bb89f968576d5a33fa0494735` |
| Core | `master` | `refactor/hdrtcp-v2` | `ab36c843af0bdbf3c8fa2d3c850737d517749db7` |
| Core | `master` | `refactor/link-router-kernel` | `9c58d75aacb134e28b22c334a8bc2fd386af2978` |
| Core | `master` | `refactor/proto-extract` | `02fc034ad5ec672c071cc1178a4d15629eef5c86` |
| Core | `master` | `refactor/subproto-kit-core` | `a2d7d298c0002e302bc00255cd048f3cd5cb51a4` |
| Core | `master` | `refactor/transport-pipe` | `a22ba405fd0c03b0efe9059780406859dc71f7fa` |
| EmbeddedSDK | `main` | `feat/embedded-board-demo-models` | `e6adde6da8a6d50739ace0aabda43d60060661e0` |
| EmbeddedSDK | `main` | `feat/embedded-local-subscribe-broker` | `04f15a580497ae26f6ac59c2f489b2960d0288e9` |
| EmbeddedSDK | `main` | `feat/embedded-sdk-c1` | `990a5e996c5f78168860205f56b60a6722d219b8` |
| EmbeddedSDK | `main` | `feat/esp32-embedded-sdk` | `99529d2786be3fe632d7cb0f18194030e719b61f` |
| EmbeddedSDK | `main` | `feat/micropython-full-hub-parity` | `0d6da12f4a3a5d799985b73df0fc17ba58f39e27` |
| MetricsNode | `main` | `chore/bump-core-v0.4.0-remaining` | `cb9e8359721b81b5e861732be83d603af1a3f85a` |
| MetricsNode | `main` | `chore/bump-sdk-v0.1.2` | `450558b5c6394d2bde19827d95d5eacd0060f26f` |
| MetricsNode | `main` | `chore/metricsnode-ci-build` | `a12b7b8dcad898e4fba74dcb3e0565efddc65555` |
| MetricsNode | `main` | `chore/metricsnode-deps-sync` | `0f5857e1684a769f0d51f94eaf97d7131816d3f9` |
| MetricsNode | `main` | `chore/metricsnode-deps-upgrade` | `6d48032e7d40d67b6c3e504f2a8b04783997e6b1` |
| MetricsNode | `main` | `chore/metricsnode-ignore-windows-config` | `25287591e197ece46a251792c272d60c564993ce` |
| MetricsNode | `main` | `chore/release-auth-route-index-heal` | `92f4186786d97881ec4b26c96f12288a0e38c0e3` |
| MetricsNode | `main` | `chore/windows-build-readme` | `48594cb1845749d5d4efec8664cbe61c9762783d` |
| MetricsNode | `main` | `feat/metricsnode-brightness-control` | `fc486d9b6a091448aa09899ee5bf759cf463e922` |
| MetricsNode | `main` | `feat/metricsnode-brightness-deviceid` | `179ca6b83dab3990b07514c3e8260cc4276e7ff6` |
| MetricsNode | `main` | `feat/metricsnode-mvp` | `d1242de393483409dbbfc0ee609a3554ac9a274a` |
| MetricsNode | `main` | `feat/metricsnode-settings-ui` | `a8c99fe39ced3a7e983f6110ca207759fadd456b` |
| MetricsNode | `main` | `feat/metricsnode-var-control` | `f265416fb5a9a1f0d401b56173e36642c1b9e022` |
| MetricsNode | `main` | `fix/android-fgs-notification-behavior` | `72ef23518d0a80cb81a72d44df524ae89a4d5cf9` |
| MetricsNode | `main` | `fix/android-notify-heads-up-channel` | `5cedf562ec643c758e573432fc7591f9c582afbd` |
| MetricsNode | `main` | `fix/android-settings-switch-order` | `7fd055f4d745a256caf28493e1fdb7df5885db91` |
| MetricsNode | `main` | `fix/metricsnode-android-ui` | `8e915158c9e2b876f3367909ce20815ebb0a0073` |
| MetricsNode | `main` | `fix/metricsnode-ci` | `92f4186786d97881ec4b26c96f12288a0e38c0e3` |
| MetricsNode | `main` | `fix/metricsnode-ci-android` | `19831abd90e36e3eb90a7e047ce1e5b27e3c6c39` |
| MetricsNode | `main` | `fix/metricsnode-ui-polish` | `2bf9a1f0c75c971aba89a888b9baee57cc11d953` |
| MetricsNode | `main` | `fix/metricsnode-win-brightness-wmi` | `9df17da6245a2e0646d6a0ae714cd5b14f13dc51` |
| MetricsNode | `main` | `fix/metricsnode-win-brightness-wmi-set` | `ebb30abc08a883024d0788b7a85a929c94cac998` |
| MetricsNode | `main` | `refactor/android-settings-label-cleanup` | `f85ec55a4460de2b492fd190c9964c2eb8862530` |
| MetricsNode | `main` | `refactor/android-settings-modern-ui` | `370c499059fdbc838350fdd120c3949cb878ba10` |
| MetricsNode | `main` | `refactor/android-settings-remove-control-and-item-frame` | `29a5ecf69268452e0f74fb574831c6c0b94231ac` |
| Proto | `main` | `chore/proto-semver` | `71a1b5c21b441e3f62744fdeca5d62d8ac4ff3aa` |
| Proto | `main` | `chore/protocol-mapgen` | `74864d91489a0715ecbf0cd94ba6c91feda3208c` |
| Proto | `main` | `docs/management-children-only` | `e915e5fd42785dc3f0b6e7fc460cc8dd8ee95ccb` |
| Proto | `main` | `feat/auth-admission-control` | `58455ad7ef5b8e58f7c602c456e17e318395570c` |
| Proto | `main` | `feat/flow-orchestrators` | `d76433b111f70289d4661821951d12637f1e499a` |
| Proto | `main` | `feat/local-vars-proto-clean` | `4732ad50f7f740fd0941403ccf609107c67d1060` |
| Proto | `main` | `feat/management-node-display-name` | `7993dd95a3179095a0fe9b299fcfdff92af873f4` |
| Proto | `main` | `feat/node-info` | `d852a931c5c25e1f12483ab3c7243b70bffb92ba` |
| Proto | `main` | `feat/proto-deploy-delete` | `52f687082b3cecd69da6f40128cdf2a901b4baf2` |
| Proto | `main` | `feat/run-control-phase1` | `75ed89799211097b48aa6120e06455edb862098f` |
| Proto | `main` | `fix/defaultset-deps-release` | `d559f949c3501f6d366781af5e163039ab223296` |
| Proto | `main` | `fix/proto-exec-cap-query` | `9f4a4ce60d236e60e7dfadf700e1f6a0161c440f` |
| SDK | `main` | `chore/bump-core-v0.4.0-downstream` | `c3a50c5e2fc9cfc5a20b3adccc063a3258385b30` |
| SDK | `main` | `chore/bump-core-v0.4.4` | `1398e9abb5efa259d8873a9492609e346453f6d8` |
| SDK | `main` | `chore/bump-core-v0.4.5-sdk` | `f45e5df49cbcd87066d8ba33f9bdd062c6882f71` |
| SDK | `main` | `chore/rfcomm-release-deps` | `e885eab158d991ce94c4bf8c26a5a34cb8f46822` |
| SDK | `main` | `chore/sdk-deps-sync` | `5fb7df9250a5dc2402770bcc8fe5b206d30fcb08` |
| SDK | `main` | `chore/sdk-semver` | `bf6d741e24264fb839205679a2c6103ce514f736` |
| SDK | `main` | `feat/await-file-ctrl` | `c8d8a31830cb6f0b41d04144225ca8dc4c0b29fc` |
| SDK | `main` | `feat/bluetooth-rfcomm-transport` | `69b4fc82673004c10c98e0d41bb6dd15709e3a6f` |
| SDK | `main` | `feat/quic-transport-sdk` | `958154e20b9b95a83cfc26f89c02e1aeb3a9a6de` |
| SDK | `main` | `feat/sdk-await-hooks` | `0d030a2bd9464c75874ccaef025a79f7a57975bd` |
| SDK | `main` | `feat/sdk-v0-session` | `7531d6614893aa7b37af43966f884f1344147826` |
| SDK | `main` | `fix/sdk-rfcomm-write-contract` | `6549eb4e6b38b7b575a107a3f7b5c7fb1b12ad9a` |
| SDK | `main` | `refactor/varstore-hop-align` | `77dbd7e7731d4cf5d8666fabb22245dc78540d0a` |
| Server | `main` | `chore/auth-admission-downstream` | `acbcfecede3860ab53e4ead4933d6201dc78bc92` |
| Server | `main` | `chore/bump-core-v0.3.0` | `2808aa5ac4e5624ea152c234f1ebe8df95f16ac5` |
| Server | `main` | `chore/bump-core-v0.4.0-downstream` | `cdb65affaf048a533cfaeda2c4af21daffbb11a8` |
| Server | `main` | `chore/bump-core-v0.4.4` | `bfecac2bddc60effb300b704046d3c210da29a05` |
| Server | `main` | `chore/bump-core-v0.4.5-server` | `73fb61aeb97f82c656ad533f590e53031a6a174e` |
| Server | `main` | `chore/bump-management-v0.1.2` | `0f6f30d0808a1e862f648f68832835c8951de2db` |
| Server | `main` | `chore/bump-varstore-v0.1.2` | `d3d4095c7f0faa10c2f386d51cdacea5cba75d01` |
| Server | `main` | `chore/data-dag-specs` | `fae225ad7ec0cba0d06bb9a56a4b7850f5ff0fdc` |
| Server | `main` | `chore/merge-server-auth-v0.1.6` | `c6d90936bfef268e4a5689d6bed0c6e6c1b7e3c3` |
| Server | `main` | `chore/protocol-delete-docs` | `a2ce21dfe5f1151d06aa0a5d13de43e32eb23e82` |
| Server | `main` | `chore/rfcomm-release-deps` | `15d06fad9d6f8132fcb4ecff78c31ca966dcaeab` |
| Server | `main` | `chore/server-bump-auth-v0.1.1` | `ffafec98b0a1880d99b968852e37d298c04db09a` |
| Server | `main` | `chore/server-bump-file-v0.1.2` | `9f6252de3f770f0a2a9db768c266dc4c404bb269` |
| Server | `main` | `chore/server-bump-varstore-v0.1.1` | `3d223affe77a69bcf82c46fbf8879a933403092f` |
| Server | `main` | `chore/server-deps-sync` | `93659c9cabc0ed7828cd0355e73e1a5f70a92dc0` |
| Server | `main` | `chore/server-docs-governance` | `3cdf2367ed2949473f4945a3f13096856c9b21f5` |
| Server | `main` | `chore/server-semver-deps` | `cd623dfa848eeaae4dd24194ed7f701aceb7522d` |
| Server | `main` | `chore/server-varstore-docs-errorcode` | `d4d23e138243b31e46cbef54f19b6085722a36e1` |
| Server | `main` | `chore/varstore-hop-align-docs` | `3771701f5cc9bfacfe9209c3f07790a09879626a` |
| Server | `main` | `docs/subproto-contract-spec` | `7adf95e924e6061001c6b4dcac3a58c2a1c61bed` |
| Server | `main` | `feat/android-hub-m0` | `63fb161511f51371aa666383a45004ef5a187282` |
| Server | `main` | `feat/auth-admission-control` | `a5de85e40769705d5b4ee034befb3597549be558` |
| Server | `main` | `feat/bluetooth-rfcomm-transport` | `6be43f9167068aaabb5dde8d9aba8b078a8ec68f` |
| Server | `main` | `feat/flow-orchestrators` | `9fc6ab6eef1cd3abc45ccff0bb48581490555670` |
| Server | `main` | `feat/local-vars-docs-clean` | `3bce37715122520e330df3edf6a376e1a3b3de6c` |
| Server | `main` | `feat/localhub` | `2e611b7930a7f0b54874268d8856066d0622bae6` |
| Server | `main` | `feat/management-node-display-name` | `69f9ab7fb3464c50a27b7684dde334dc0adeffc5` |
| Server | `main` | `feat/node-display-name-followup` | `ee6e76dfa8681b37ffffa5e441ddc2a1388ab257` |
| Server | `main` | `feat/node-info` | `a1b933eee484622da05b4cb08e125141db635d68` |
| Server | `main` | `feat/public-protocol` | `96d35a5be1afa5ffa12bd525a8a29c30c9202922` |
| Server | `main` | `feat/quic-dev-cert-auto` | `3ed6aca6bfade2aaf1bb28cc92770394c7c80d48` |
| Server | `main` | `feat/quic-transport-server` | `8e8c65551ad69efea308c23eae10955776f7e10b` |
| Server | `main` | `feat/run-archive-backend` | `d3a00342c58fd346b6e86381aa9217ead03371fc` |
| Server | `main` | `feat/run-control-phase1` | `5d927a0ea381a1cffaaf4f808609e11554d891ed` |
| Server | `main` | `fix/auth-route-index-heal` | `7c27ea7b36f23bb69f0d2c84ff7fce58b6fbba57` |
| Server | `main` | `fix/defaultset-deps-release` | `5ca8cc2706290c5aec175aa5efec154965cd0108` |
| Server | `main` | `fix/file-ctrl-inherit-ids` | `46039f870989f4d35abfcfad9a92acd7fcbf6ec5` |
| Server | `main` | `fix/file-resp-major-okresp` | `2e2dec238caff2e528813318ef263f7bde23d3a8` |
| Server | `main` | `fix/hub-file-console-base-dir` | `81d9681a8782f2f3d8ffb1f1cb388c0c2a326732` |
| Server | `main` | `fix/resp-inherit-ids` | `2e2dec238caff2e528813318ef263f7bde23d3a8` |
| Server | `main` | `fix/resp-major-okresp` | `863da5789cb878518da02a279c55ac7900afb477` |
| Server | `main` | `refactor/hdrtcp-v2` | `01a1df7788337b258252e726b5938901b7ef7c13` |
| Server | `main` | `refactor/link-router-kernel` | `077865e00efaac946ce4b4d1df9f563bb8e82129` |
| Server | `main` | `refactor/pluggable-state-backend` | `af40a72a5f6ef3cb47e85c0f34244565c4fd8612` |
| Server | `main` | `refactor/proto-extract` | `d8165d8c2a4448b5a0a5fd60f8d3a1a0acd6d903` |
| Server | `main` | `refactor/server-action-kit` | `e1e36e1c68cf950f00767809e71748a9d19a6860` |
| Server | `main` | `refactor/server-action-template` | `d5143372842de2950616e9230b824b9467550850` |
| Server | `main` | `refactor/server-default-forward` | `07ede9f6aa619045fff6e3a5a055d5ed26e53870` |
| Server | `main` | `refactor/server-modules` | `e2c361684d616dbfb5af2270fb4cc1fd5cbe5696` |
| Server | `main` | `refactor/server-subproto-public` | `3e78ecd0e7429c8bc479d4306e566eb537af5f8e` |
| Server | `main` | `refactor/subproto-kit-core` | `85d622b496b07c2054de20bcf6ac6e54b95b2aed` |
| Server | `main` | `refactor/subproto-management-module` | `e7ac863a1578c9f8377190fd91dff323eeade8b2` |
| Server | `main` | `refactor/subproto-modules-all` | `ca374b4e5fb0e22e390e445b74c643ab29d9056e` |
| Server | `main` | `refactor/subproto-topicbus-module` | `5e54882252cf55ea13cb8991f2ea2870c90d24c2` |
| Server | `main` | `refactor/transport-pipe` | `e5d5f66a5b8d2e374c153b186e1ca8c86a632eec` |
| SubProto | `main` | `chore/bump-core-v0.3.0` | `48c48d6dbd28afa66f021245bbca78e99106c667` |
| SubProto | `main` | `chore/bump-core-v0.3.0-subproto-more` | `ebd88cd289d1e735b7fc84e680f601fd5274149a` |
| SubProto | `main` | `chore/bump-core-v0.4.0-remaining` | `0e69be22e8d91ae5d4577809b749e3ec29e4bc52` |
| SubProto | `main` | `feat/auth-admission-control` | `3b07e92f9d30988a0ca689805ffb475af9377e37` |
| SubProto | `main` | `feat/capability-registry-ext` | `cd4f9c04a286b10b0866b0fcdd76e4f0d3a99381` |
| SubProto | `main` | `feat/data-dag-bindings` | `59b8a11d0eaf3bce89cc60031a99cceee145d08c` |
| SubProto | `main` | `feat/file-mkdir-op` | `2b8cf76667b1c858e97c599cbef8c50a86eefdbc` |
| SubProto | `main` | `feat/flow-orchestrators` | `e98be7e8a7d290d76d2973094e1d32fb9d5ff221` |
| SubProto | `main` | `feat/local-vars-runtime-clean` | `f5e68b0f14a90c142b4963dd871f6c2f43333a8c` |
| SubProto | `main` | `feat/management-node-display-name` | `94d7e823aaace4feceed7cd8709544fa494dfe18` |
| SubProto | `main` | `feat/node-display-name-followup` | `5aa00e278317747aefe03e63aeb5e2d75f58f076` |
| SubProto | `main` | `feat/node-info` | `ecd7c33c40e8e220d0a6780253b435d0aa17247c` |
| SubProto | `main` | `feat/run-archive-backend` | `f61fac356b9cbeeb01ba28846d033dcfcc383597` |
| SubProto | `main` | `feat/run-control-phase1` | `671236b6964aadc033d85b5cc575afbafd861a76` |
| SubProto | `main` | `feat/subproto-deploy-delete` | `f4e0fbb0eee1856eaa858866a5cdfb58485d891a` |
| SubProto | `main` | `fix/auth-route-index-heal` | `de9a210878b6f326e5d8dd9107bb1a05f80f0d23` |
| SubProto | `main` | `fix/defaultset-deps-release` | `e2230a3dc45d9d5c238e2839670083ef1cb8e9da` |
| SubProto | `main` | `fix/delete-baseline-mainline` | `9786e4d703daa3dcb7080e41d1b1f7533c50b7eb` |
| SubProto | `main` | `fix/flow-state-route-retention` | `b3d380307eb3a543d11b615ad1b23bfd9d2c423d` |
| SubProto | `main` | `fix/hub-file-console-base-dir` | `74b98dc2fd5252c92c571dfa494deadb81c73ed6` |
| SubProto | `main` | `fix/management-children-only` | `d512b41a50aa8ab6527ecafc85e5759d44b732fe` |
| SubProto | `main` | `fix/resp-msgid` | `11901c8618e05a74552a1df45e03eef4bccaa925` |
| SubProto | `main` | `fix/subproto-id-guard` | `fab63d957ec8d2179ccee8bce7a648ea731a80fa` |
| SubProto | `main` | `fix/subproto-uplogin-sender-pub` | `b59d3a4fd62bf69504da23c7dc6744a57ff21730` |
| SubProto | `main` | `fix/varstore-action-regression` | `40d89258a663a8c975e0684be0e105d09bb0dc65` |
| SubProto | `main` | `fix/varstore-crosshop-routing` | `5b9050f846efeb24260643f8962aa10684d0aba0` |
| SubProto | `main` | `refactor/pluggable-state-backend` | `c5135496a02f7e89e4bd90cbdbbb0ac54f1b383c` |
| SubProto | `main` | `refactor/subproto-modules-all` | `fdd0ae2338ca2c2ace592d8f69684f2b0f48282c` |
| SubProto | `main` | `refactor/subproto-topicbus-module` | `6e8338a831a47b83d18793830c766456d9f54cd9` |
| SubProto | `main` | `refactor/transport-pipe` | `4cf2a3856dbf5fb351ac98974976918648870237` |
| SubProto | `main` | `refactor/varstore-hop-align` | `e4cbd20405da009dcca4843520170107b82dfa6e` |
| Win | `main` | `chore/bump-core-v0.4.0-remaining` | `4212cfea9238d8846f12c9c07d2d1a77883b2681` |
| Win | `main` | `chore/bump-core-v0.4.4` | `d5c4728de206289604fd72e4ed40fe6b0875e9d6` |
| Win | `main` | `chore/bump-core-v0.4.5-win` | `a183e3319f18a1092e6c95386aada1891c7aef6a` |
| Win | `main` | `chore/bump-sdk-v0.1.2` | `f54b3622b4bdf0f525412a527a179852a4c8e8f8` |
| Win | `main` | `chore/win-deps-sync` | `b93c5cdbcafcd37ddb01e4a931b84053a829f787` |
| Win | `main` | `chore/win-rfcomm-write-bump` | `48b57a1bb293cc363c0fb53cd1a46263486fdccf` |
| Win | `main` | `chore/win-semver-deps` | `dd213a9de04094c70b36885c90f8673c4b3b9e30` |
| Win | `main` | `feat/data-dag-editor` | `eef25ae1ce4fd945a09e5f91939de565eefefc72` |
| Win | `main` | `feat/file-add-node-select-picker` | `c5363d18d1950f2635c65ba00fc4fb02b0bda9db` |
| Win | `main` | `feat/file-console-mkdir` | `21efdd769760d8b57af4a00fcfa607bdb8f69168` |
| Win | `main` | `feat/file-console-offer-dir-node-picker` | `a126c4ec19b7835e4d0e34b81b05843395ae24a3` |
| Win | `main` | `feat/file-console-ui-upgrade` | `616f35faecd7f73d05b3e86743c0815072364fbe` |
| Win | `main` | `feat/file-node-picker-confirm` | `87dda7ec70502cfb1389f928c048159ce0dbc365` |
| Win | `main` | `feat/file-offer-target-picker-dialog` | `ee74a2a08d4067dbc9e5daa461cd9848c86adbcf` |
| Win | `main` | `feat/file-remove-nodes-select` | `bb22c1a728f63b2727feb12d7dd992d85d234fae` |
| Win | `main` | `feat/flow-list-row-simplify` | `1c6baf91f5fd037bcfa75cfd6562638348fb2da5` |
| Win | `main` | `feat/localhub` | `68b299c0dd3d335b73553adb4ffe60bca28b2c83` |
| Win | `main` | `feat/management-node-display-name` | `e390b7390d549ec1bef7c13681771786596e791c` |
| Win | `main` | `feat/mcp-flow-create` | `a631e26d53a53a9b54117c5d0a69ed3515d9cfdf` |
| Win | `main` | `feat/node-display-name-followup` | `af24f897b6bd64f2fb98705859fc114324507667` |
| Win | `main` | `feat/node-info` | `f583edda21670e6bc36a3c63bff5ef404f895b53` |
| Win | `main` | `feat/quic-transport-win` | `54d4c808646a05ce1e0e377fdf17226df6492778` |
| Win | `main` | `feat/showcase-canvas-layout` | `d0913ee52ebfd1b9dd02ad520ffe2a3c2dd4e91b` |
| Win | `main` | `feat/showcase-screen` | `b2a3f1bc7f22eca5fef1d59e71d01a48e1d712f8` |
| Win | `main` | `feat/showcase-toolbar-icons` | `ad4578b359311d78a226fd41449ecd23170ec364` |
| Win | `main` | `feat/showcase-var-dialog-polish` | `b2d424546ebec4cedc75d70855eac7a62dedd7a1` |
| Win | `main` | `feat/showcase-var-quickpick` | `57a485c8d9ff473fc314bc8a5b716d2fb14a73a7` |
| Win | `main` | `feat/topicbus-window-console` | `66f3f297c7f1eb82814f3ea3fac77413d35e4f70` |
| Win | `main` | `feat/varpool-vars-dialog` | `4efb98880d51e664bd2ae681fed0b36800ffa7e3` |
| Win | `main` | `feat/win-authority-permissions-v1` | `c8b4c421580270a30be39223d862a9111378d59f` |
| Win | `main` | `feat/win-call-visual-form` | `466e490575975167b7a51447d23877aee54dc09b` |
| Win | `main` | `feat/win-capability-local-picker` | `8875f1fb282a93b92d27b604fed2c25ebf3c7bc0` |
| Win | `main` | `feat/win-dag-editor` | `cd067607206b8666750c54bfc925b334e86a5166` |
| Win | `main` | `feat/win-dag-editor-ux` | `14df58d75d8c31a5a5fe2a85c8e157b81bc3997c` |
| Win | `main` | `feat/win-dag-nodeid-icons` | `b0b69686f5141176d0a6aa784b5d339c464331bd` |
| Win | `main` | `feat/win-flow-capability-picker` | `c396bddb236db54cc65bee4523d6c7efff2587c8` |
| Win | `main` | `feat/win-mcp-full-chain-smoke` | `b1acdb22bbaaa6c600b20d0c17fe26831918eaf8` |
| Win | `main` | `feat/win-orchestrator-editor` | `1b8634ffa8d9bca1f0d6bdc806016b44b4971421` |
| Win | `main` | `feat/win-project-center` | `b0360ede8f57c9a8b418afcbecfac0db48534936` |
| Win | `main` | `feat/win-session-devices` | `b497c40fe5d6152abe44ddc63f9e2a0d14e197c8` |
| Win | `main` | `feat/win-settings-i18n` | `5f7d90e5eb27ff94b654c65d152fa7495375f7ad` |
| Win | `main` | `feat/win-settings-page` | `699af33249b6c87f42325ecc6d4d7da82b429dda` |
| Win | `main` | `feat/win-showcase-layout` | `c025006ecb2572a728b70cf792d3ac079210d505` |
| Win | `main` | `feat/win-stream-console` | `7b209a467b586262757992abdefd3c1ca6fb93d5` |
| Win | `main` | `feat/win-stream-control-target-picker` | `a631e26d53a53a9b54117c5d0a69ed3515d9cfdf` |
| Win | `main` | `feat/win-stream-product-tabs` | `d47a84c76575be7042fdbcb9021955c99c45b342` |
| Win | `main` | `fix/file-console-dnd-upload` | `0d65ea788f6878e86de1653acc3df7bf2fb0ffaf` |
| Win | `main` | `fix/flow-varstore-owner-constant` | `86e33a20cd0153f4f15c525e91a421bca61cbe7c` |
| Win | `main` | `fix/nodevars-list-timeout` | `d8c74933fa68bc05c9ea77703b516bd4e73d2e52` |
| Win | `main` | `fix/showcase-page-blank` | `ab57baca36961283bb5968a0600ffa747357d12f` |
| Win | `main` | `fix/showcase-throttle-label` | `7d1590c7efa89e6b5e56a640db9bd7a3481568bc` |
| Win | `main` | `fix/topicbus-window-actions-height` | `ed3317eb0eca6e01d54bedeccebe3e69dc462c3d` |
| Win | `main` | `fix/topicbus-window-snapshot-scroll` | `14d7e636c990f86859811f00d138f9baee0c47cc` |
| Win | `main` | `fix/varpool-refresh-subscribe-ui` | `1fedb374fae88b6e8e62de1c0f0d5f933904bde3` |
| Win | `main` | `fix/win-access-policy-role-dialog-refine` | `6571954fa132bcc2659b06c16c8a601b723fb524` |
| Win | `main` | `fix/win-button-pointer` | `7f03a4975a7d53f9030d59fbca7a06fded5b8e96` |
| Win | `main` | `fix/win-canvas-drag-state` | `9b2488b6bd28f89ae51f047f78a4904e43706e85` |
| Win | `main` | `fix/win-canvas-events` | `847c42137dfc4cd8eb3ce59ef772602e432d0fd4` |
| Win | `main` | `fix/win-devices-tree-expand` | `97b5f30624e288431cb274151a9f8be7eaa3ede2` |
| Win | `main` | `fix/win-embed-dist-placeholder` | `68e1d2a662731e65de21bf79e97d87e45ee8f015` |
| Win | `main` | `fix/win-file-console-base-dir` | `e5f9a038bade337751676b81b1c1216be86412ba` |
| Win | `main` | `fix/win-flow-list-inline-meta` | `5c6cfd29e0ef0c1121179437db8bac3e1546423f` |
| Win | `main` | `fix/win-i18n-coverage` | `856f0f591e5c4ed7811a269d0f86a1bc2b9eaaea` |
| Win | `main` | `fix/win-overlay-mask` | `d8351af5d40d4b1c4714ac5122d3514ac7fb5ddb` |
| Win | `main` | `fix/win-self-config` | `ba1aae096c8de935a7a04f450a0d091f9b1d779a` |
| Win | `main` | `fix/win-single-hero-title` | `5ed178ec4a0ec58d72621a8bce7b8398131cad6a` |
| Win | `main` | `fix/win-translation-settings` | `c39695f591139e5dd45b8b92a6b587919bac4042` |
| Win | `main` | `fix/win-varpool-live-events` | `856f0f591e5c4ed7811a269d0f86a1bc2b9eaaea` |
| Win | `main` | `fix/win-vite-build` | `088f08666890c055ece81506eb65d98f9f5e89c2` |
| Win | `main` | `fix/win-wails-bindings-build` | `aa1b482829f149a6f112a86f9d1b608ca69dcac5` |
| Win | `main` | `refactor/hdrtcp-v2` | `e2da4810576fdbf86e9f8543f053869f7603b31a` |
| Win | `main` | `refactor/proto-extract` | `ad311998f8eb0f183e96a88227a124cad67b7b23` |
| Win | `main` | `refactor/remove-fyne` | `d853666e6fd8a617c17e4eb1b50a4c7025558554` |
| Win | `main` | `refactor/showcase-center-editor` | `eea00a40b14dbf4cb3e2386b6d45b3b131726fd6` |
| Win | `main` | `refactor/showcase-ui-simplify` | `0674af14dd4fcb83d466b84e4183940bf54d2887` |
| Win | `main` | `refactor/topicbus-settings-pane` | `8998bcef364c93a43035f3ee0a91d35c5d7f1018` |
| Win | `main` | `refactor/topicbus-target-settings` | `6923f04ecf464ceb4fe9b8e73ce581d4bfbd9cf3` |
| Win | `main` | `refactor/topicbus-window-layout` | `63f4dce8ceea2d678d1cbede32cea31549139711` |
| Win | `main` | `refactor/varpool-tabs-layout` | `2d850680a760fd8d5a75d78c6730c1d5aa01ec5d` |
| Win | `main` | `refactor/varstore-hop-align` | `d37251bf54f603ccebb301982642c98727b68764` |
| Win | `main` | `refactor/win-access-policy-dialog-editor` | `1ccd03f2b576960a0d9ac61aaabad2238c11c390` |
| Win | `main` | `refactor/win-access-policy-tabs` | `c13092cc390a675fb067801726b0a34e49fa77e5` |
| Win | `main` | `refactor/win-auth-await` | `321a0be3cb510df6a19663bfd5867b372e353e28` |
| Win | `main` | `refactor/win-file-ctrl-await` | `ac031932730aa918e586fcbbfe77fbae0afb12ee` |
| Win | `main` | `refactor/win-method-selector-dialog` | `074f1a8666c5700392b52b1d3594bd82cc1a2e30` |
| Win | `main` | `refactor/win-mgmt-await` | `86f8406feb2f7845416a6b586e40fcb1dd484cd0` |
| Win | `main` | `refactor/win-mgmt-service` | `627e301ef9bde963687560869018de5da585b750` |
| Win | `main` | `refactor/win-orch-await` | `4f8b12c88c14bdcad85775da2f2e6cbc1d310964` |
| Win | `main` | `refactor/win-project-center-editor` | `ccdb9902e322c3521fe020bbd7d88a3f7a8b0648` |
| Win | `main` | `refactor/win-sdk-v0` | `1686f7f262e39da521d595817aa4385cb69df8b5` |
| Win | `main` | `refactor/win-services-converge` | `3af94f5506444a68d4c296f47a74cc11bd3e92fe` |
| Win | `main` | `refactor/win-topicbus-await` | `c5ecb1355e7a6e556a86732cfb7f3af558df2c53` |
| Win | `main` | `refactor/win-ui-polish` | `409f07888088a1b9efe98f6fd08fae8899dad923` |
| Win | `main` | `refactor/win-varpool-await` | `233888a411fc2cff34baefba352f509443a10f7d` |
