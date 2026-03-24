# 2026-03-24 Auth admission 下游依赖跟进

## 变更背景 / 目标
- 上一轮已经完成 `Proto v0.1.2`、`Core v0.4.8`、`auth/v0.1.4` 的发布，并让 `MyFlowHub-Server` 在 `GOWORK=off` 下通过验证。
- 但消费层仍未全部跟进：
  - `MyFlowHub-SDK` 仍停在 `core v0.4.7 + proto v0.1.1`
  - `MyFlowHub-Win` 仍停在 `core v0.4.7 + proto pseudo-version + sdk v0.1.10`
  - `MyFlowHub-Android/hubmobile` 仍停在 `core v0.4.0 + proto v0.1.1 + auth v0.1.2`
- 本次目标是继续把 auth admission 相关的下游依赖链往前推，并明确哪些仓库可以安全发布，哪些还存在 release blocker。

## 具体变更内容
### SDK
- `MyFlowHub-SDK/go.mod`
  - `github.com/yttydcs/myflowhub-core`：`v0.4.7` -> `v0.4.8`
  - `github.com/yttydcs/myflowhub-proto`：`v0.1.1` -> `v0.1.2`
- 执行：
  - `go mod tidy`
  - `GOWORK=off go test ./... -count=1 -p 1`
- 发布 tag：
  - `v0.1.11`

### Win
- `MyFlowHub-Win/go.mod`
  - `github.com/yttydcs/myflowhub-core`：`v0.4.7` -> `v0.4.8`
  - `github.com/yttydcs/myflowhub-proto`：`v0.1.2-0.20260318063708-7eef50dcc471` -> `v0.1.2`
  - `github.com/yttydcs/myflowhub-sdk`：`v0.1.10` -> `v0.1.11`
- 执行：
  - `go mod tidy`
  - `GOWORK=off go test ./... -count=1 -p 1`
- 发布 tag：
  - `v0.0.11`

### Android
- `MyFlowHub-Android/hubmobile/go.mod`
  - `github.com/yttydcs/myflowhub-core`：`v0.4.0` -> `v0.4.8`
  - `github.com/yttydcs/myflowhub-proto`：`v0.1.1` -> `v0.1.2`
  - `github.com/yttydcs/myflowhub-sdk`：`v0.1.4` -> `v0.1.11`
  - `github.com/yttydcs/myflowhub-subproto/auth`：`v0.1.2` -> `v0.1.4`（indirect）
- 保留：
  - `replace github.com/yttydcs/myflowhub-server => ../../MyFlowHub-Server`
  - `replace github.com/yttydcs/myflowhub-sdk => ../../MyFlowHub-SDK`
- 执行：
  - `GOWORK=off go mod tidy`
  - `GOWORK=off go test ./... -count=1 -p 1`

### Android release blocker
- `MyFlowHub-Android/.github/workflows/ci.yml` 与 `release.yml`
  - 仍通过 `actions/checkout` 拉取 `yttydcs/myflowhub-server` 作为 `repo/MyFlowHub-Server`
  - 没有固定到本轮 auth admission server 基线的 `ref`
- 由于 `MyFlowHub-Server` 的 auth admission 代码当前还未以远端 branch/release 形式公开，这意味着：
  - 本地 workflow worktree 的 Android 测试可以通过
  - 但直接发 Android tag 时，远端 CI 仍可能 checkout 到旧 Server 默认分支，产出不含本轮 server auth 逻辑的包
- 结论：
  - Android 依赖升级已完成并本地验证通过
  - 本轮不发布 Android 新 tag

## Requirements impact
`none`

## Specs impact
`none`

## Lessons impact
`updated`

## Related requirements
- [auth-controlled-admission.md](../requirements/auth-controlled-admission.md)

## Related specs
- [auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md)

## Related lessons
- [cross-repo-semver-release.md](../lessons/cross-repo-semver-release.md)

## 对应 plan.md 任务映射
- `SCAN1` - 依赖面扫描
- `SDK1` - SDK 依赖对齐
- `WIN1` - Win 依赖对齐
- `AND1` - Android/hubmobile 依赖对齐
- `TEST1` - `GOWORK=off` 验证
- `REL1` - SDK/Win 发布，Android release blocker 记录
- `REV1` - 3.3 Code Review
- `ARC1` - 本文归档

## 经验 / 教训摘要
- 真正可发布的下游链要继续沿依赖方向推进，不能停在 `Server`。
- `Win` 这类纯消费方可以在上游 tag 就绪后直接跟进并发布。
- `Android` 这类通过 sibling repo `replace` 构建的应用仓，即使本地 worktree 绿了，也不能假设远端 tag release 一定安全。

## 可复用排查线索
- 症状：
  - 本地 `GOWORK=off go test` 通过，但 Android 远端 release 仍可能基于旧 Server 构建。
- 触发条件：
  - 应用仓通过 `replace ../../SomeRepo` 引入 sibling repo
  - CI/release workflow 用 `actions/checkout` 拉取 sibling repo 默认分支
  - 本轮所需上游代码尚未公开到该默认分支或对应 release ref
- 关键词：
  - `Checkout Server (for hubmobile replace)`
  - `actions/checkout`
  - `default branch`
  - `replace ../../MyFlowHub-Server`
- 快速检查：
  - 打开应用仓 `.github/workflows/*.yml`
  - 确认 sibling repo checkout 是否指定 `ref`
  - 确认该 `ref` 是否已包含本轮所需的上游代码

## 关键设计决策与权衡
- SDK 先发布 `v0.1.11`，让 Win 可以在真实 `GOWORK=off` 下依赖新版本，而不是临时加 `replace`。
- Win 将 Proto 从 pseudo-version 切回稳定 tag `v0.1.2`，避免继续把一次性的联调基线当作正式发行依赖。
- Android 只做依赖升级和本地验证，不强行发 tag；因为当前 release workflow 对 Server 的获取方式还不满足“拿到本轮 auth admission server 基线”的条件。

## 测试与验证方式 / 结果
- `MyFlowHub-SDK`
  - `GOWORK=off go test ./... -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-Win`
  - `GOWORK=off go test ./... -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-Android/hubmobile`
  - `GOWORK=off go test ./... -count=1 -p 1`
  - 结果：通过
- 发布验证：
  - `go list -m github.com/yttydcs/myflowhub-sdk@v0.1.11`
  - `go list -m github.com/yttydcs/myflowhub-win@v0.0.11`
  - 结果：通过

## 潜在影响
- SDK 与 Win 现在已经跟上 auth admission 上游依赖基线。
- Android 代码级依赖也已跟上，但远端 release 仍受 Server checkout 基线约束。
- 如果现在直接发 Android tag，可能产出版本号更新、但内嵌 Server 仍是旧基线的 release 包。

## 回滚方案
- SDK / Win：
  - 若仅限本地提交，可直接 revert 依赖升级提交。
  - 已 push 的 tag 不改写，若发现问题发更高 patch 修复。
- Android：
  - revert `hubmobile/go.mod` 与 `hubmobile/go.sum` 即可恢复旧依赖口径。
  - 本轮未发布 Android tag，无远端 tag 回滚问题。

## 子Agent执行轨迹
- 无。全部由主 agent 在当前 workflow 中完成。
