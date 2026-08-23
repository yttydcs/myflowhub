# 2026-04-12_cross-repo-release-chain-publish

## 变更背景 / 目标

- `MyFlowHub3` workspace 下除 `MyFlowHub-EmbeddedSDK` 之外，仍有多仓未提交改动、未公开 patch tag，以及下游 `go.mod` 基线滞后的问题。
- 本次目标是在不混入计划外功能开发的前提下，完成一次受控的多仓发布链收口：
  - 先审查 dirty changes，只吸收合理变更
  - 严格按依赖方向推进 release
  - 仅在上游 tag 远端可见后才修改下游依赖
  - 用真实 `GOWORK=off` 验证证明各消费方能解析和测试通过

## 具体变更内容

### Workspace / Strategy
- 明确本轮 release policy：
  - 只为“相对上一个正式 tag 有有效发布内容，或本轮新增了 release-alignment 提交”的仓库 / module 发新 patch tag
  - 不为 `Core` / `Proto` 强行补 no-op patch tag
- 明确排除：
  - `MyFlowHub-EmbeddedSDK`
  - `Win` 的 `myflowhub-mcp.exe`
  - `MetricsNode` 的 `wailsjs` / `windows/go.mod` CRLF-only noise
  - `SubProto/flow` 当前 post-`flow/v0.1.5` 的本地 replace / proto 降级基线

### 发布结果矩阵
- `MyFlowHub-Core`
  - push branch: `origin/chore/release-chain-20260412-core`
  - reused release baseline: `v0.4.10`
  - no new tag
- `MyFlowHub-Proto`
  - push branch: `origin/chore/release-chain-20260412-proto`
  - reused release baseline: `v0.1.7`
  - no new tag
- `MyFlowHub-SubProto`
  - commit: `1917fdd`
  - push branch: `origin/chore/release-chain-20260412-subproto`
  - push module tag: `auth/v0.1.6`
  - skipped module: `flow`
- `MyFlowHub-SDK`
  - commits:
    - `49305eb` - 忽略 `.ace-tool/`
    - `27e7476` - 对齐 `core v0.4.10` / `proto v0.1.7`
  - push branch: `origin/chore/release-chain-20260412-sdk`
  - push tag: `v0.1.14`
- `MyFlowHub-Server`
  - commit: `2b0d0a3`
  - change: `auth v0.1.5 -> v0.1.6`
  - push branch: `origin/chore/release-chain-20260412-server`
  - push tag: `v0.0.16`
- `MyFlowHub-Win`
  - base functional commits retained:
    - `dc1f02f`
    - `a631e26`
  - release alignment commit: `77a21fd`
  - change: `core v0.4.9 -> v0.4.10`, `sdk v0.1.13 -> v0.1.14`
  - push branch: `origin/chore/release-chain-20260412-win`
  - push tag: `v0.0.16`
- `MyFlowHub-MetricsNode`
  - commits:
    - `03583a6` - 忽略 `.ace-tool/`
    - `a5e14fb` - 对齐 `core v0.4.10` / `proto v0.1.7` / `sdk v0.1.14`
  - push branch: `origin/chore/release-chain-20260412-metrics`
  - push tag: `v0.1.3`
- `MyFlowHub-Android`
  - commit: `1f966e7`
  - change: `server v0.0.13 -> v0.0.16`, `core v0.4.9 -> v0.4.10`, `proto v0.1.5 -> v0.1.7`, `sdk v0.1.13 -> v0.1.14`, `auth v0.1.5 -> v0.1.6`, `flow v0.1.2 -> v0.1.5`
  - 保留 `replace ../../MyFlowHub-Server` / `../../MyFlowHub-SDK` / `../../MyFlowHub-Proto`
  - push branch: `origin/chore/release-chain-20260412-android`
  - push tag: `v0.1.30`

## Requirements impact
`none`

## Specs impact
`none`

## Lessons impact
`none`

## Related requirements
- `none`

## Related specs
- `none`

## Related lessons
- [cross-repo-semver-release.md](../lessons/cross-repo-semver-release.md)

## 对应 plan.md 任务映射
- `CTRL-1` - 固化版本矩阵与 release policy
- `CORE-1` - Core branch push / conditional tag
- `PROTO-1` - Proto branch push / conditional tag
- `SUB-1` - SubProto auth dirty fix 与 module release 判断
- `SDK-1` - SDK `.gitignore` 收口与 Core / Proto 依赖对齐
- `SRV-1` - Server 对齐 `auth v0.1.6`
- `WIN-1` - Win 保留已有功能提交并对齐 `core v0.4.10` / `sdk v0.1.14`
- `MET-1` - MetricsNode `.gitignore` 收口与 `core/proto/sdk` 依赖对齐
- `AND-1` - Android `hubmobile` 对齐 `server/core/proto/sdk/auth/flow` 基线
- `VAL-1` - branch/tag remote verify 与 `GOWORK=off` 验证
- `REVIEW-1` - Stage 3.3 review
- `DOC-1` - Stage 4 归档

## 经验 / 教训摘要
- 依赖方向仍然必须严格保持 `Core / Proto -> SubProto module -> SDK / Server -> 应用层`，否则下游很容易在远端尚未可见时写入不可解析版本。
- 对像 `SubProto` 这样的多 module 仓库，必须区分“仓库有变化”和“某个 module 有可发布变化”；本轮 `flow` 被刻意跳过，就是为了避免把本地 replace / baseline 噪音误发成正式 tag。
- `Android` 这类保留 sibling `replace` 的仓库，即使本地 `GOWORK=off` 已通过，也不能把它等同于远端 release workflow 已完全安全；CI checkout 口径必须单独审视。

## 可复用排查线索
- Symptoms:
  - 下游准备 bump 到某个上游版本时，`go list -m` / `go test` 报 `unknown revision` 或解析到旧版本
  - 单仓多 module 仓库里，只有部分 module 适合发布，另一些只出现 replace / CRLF / generated noise
  - Android / Win / MetricsNode 这类应用层仓库在本地能过，但 CI 仍可能拿到旧 sibling repo 基线
- Trigger Conditions:
  - 上游 tag 尚未完成远端可见性验证，就开始修改下游 `go.mod/go.sum`
  - 把 CRLF-only、generated 目录、临时二进制或本地工作目录变更混进 release commit
  - 误把 workspace `replace` / `go.work` 联调成功当成公开 release 已闭环
- Keywords:
  - `git ls-remote origin refs/tags/...`
  - `GOWORK=off`
  - `unknown revision`
  - `replace ../../MyFlowHub-Server`
  - `wailsjs`
  - `myflowhub-mcp.exe`
- Quick Checks:
  - 先查远端 tag 是否可见，再更新下游依赖
  - 用 `git diff --ignore-cr-at-eol` 或等价方式确认 CRLF-only 噪音不入提交
  - 对关键下游执行 `GOWORK=off go test ./... -count=1 -p 1`
  - 对保留 sibling `replace` 的仓库额外检查 CI workflow checkout 的 repo 与 ref

## 关键设计决策与权衡

1. 不为 `Core` / `Proto` 补 no-op patch tag
- 好处：避免制造没有有效发布内容的新版本号。
- 代价：下游版本收口必须直接复用现有 `v0.4.10` / `v0.1.7`。

2. `SubProto` 只发布 `auth/v0.1.6`
- 好处：只把本轮审过且有明确功能价值的 module 对外公开。
- 代价：`flow` 当前 post-`flow/v0.1.5` 的本地基线问题被显式留待后续单独处理。

3. Android 仍保留 sibling `replace`
- 好处：保持现有本地/CI 目录拓扑，不额外扩大 Android workflow 变更面。
- 代价：GitHub Actions 仍 checkout `main` 的 sibling repos，release 结果需要结合实际 checkout 口径解读。

## 测试与验证方式 / 结果

- `MyFlowHub-SubProto/auth`
  - `GOWORK=off go test ./... -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-SDK`
  - `GOWORK=off go test ./... -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-Server`
  - `GOWORK=off go test ./... -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-Win`
  - `GOWORK=off go test ./... -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-MetricsNode`
  - root `GOWORK=off go test ./... -count=1 -p 1`
  - `windows` `GOWORK=off go test ./... -count=1 -p 1`
    - 测试前临时创建 `windows/frontend/dist/placeholder.txt`
  - `nodemobile` `GOWORK=off go test ./... -count=1 -p 1`
  - 结果：全部通过
- `MyFlowHub-Android/hubmobile`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-server github.com/yttydcs/myflowhub-core github.com/yttydcs/myflowhub-proto github.com/yttydcs/myflowhub-sdk github.com/yttydcs/myflowhub-subproto/auth github.com/yttydcs/myflowhub-subproto/flow`
  - `GOWORK=off go test ./... -count=1 -p 1`
  - 结果：通过
- Remote refs
  - 对所有发布分支 / tag 执行 `git ls-remote origin refs/heads/... refs/tags/... refs/tags/...^{}` 复验
  - 结果：全部可见

## Code Review 结论

- 需求覆盖：通过
  - dirty changes 已做合理性筛选，并严格按依赖方向发版
- 架构合理性：通过
  - `Core/Proto -> SubProto -> SDK/Server -> Win/Metrics/Android` 的 gate 未被跳过
- 稳定性与安全：通过
  - 未重写已有 tag
  - 未把二进制、generated noise、CRLF-only 变更纳入提交
- 测试覆盖情况：通过
  - 关键消费方均补了真实 `GOWORK=off` 验证

## 潜在影响

- `MyFlowHub-Win` / `MyFlowHub-MetricsNode` / `MyFlowHub-Android` 将解析到更高的 `SDK/Core/Proto/Server/auth` 发布基线。
- Android 的 GitHub Actions 仍 checkout sibling repo 的 `main`；本轮由于上游与 Android 相关的差异主要是依赖基线收口，本地验证证明当前 graph 仍会解析到目标版本，但远端 release 仍建议关注实际 Actions 结果。

## 回滚方案

- 已 push 的 `auth/v0.1.6` / `v0.1.14` / `v0.0.16` / `v0.0.16` / `v0.1.3` / `v0.1.30` 不改写；若发现问题，只追加更高 patch 修复。
- 各仓代码层回滚：
  - `SDK` / `Server` / `Win` / `MetricsNode` / `Android` 可在各自 release 分支上 revert 依赖升级提交并重新验证。
- `SubProto/flow` 后续若要修正 release baseline，应单开 workflow，不复用本轮 tag。

## 子Agent执行轨迹
- 无子 agent；本轮发布链按强顺序 gate 由主代理串行完成。

## Notes

- GitHub `443` 在本 workflow 中两次出现 transient `Connection was reset` / `Could not connect to server`；重试后恢复，最终远端发布完成。
- 控制面 `todo.md` 与本归档只保留在 `MyFlowHub3` 控制工作树，不混入各产品仓代码提交。
