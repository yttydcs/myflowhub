# 2026-03-24 Auth 准入发布链与 `GOWORK=off` 验证

## 变更背景 / 目标
- `auth` 受控准入、一次性角色 permit 和 authority fail-closed 功能在 workflow-local `go.work` 联调下已通过，但这还不足以证明下游仓库可单独拉取和构建。
- 实际收口时发现：
  - `MyFlowHub-SubProto/auth` 在 `GOWORK=off` 下仍依赖尚未发布的 `MyFlowHub-Proto` / `MyFlowHub-Core` 新 API。
  - `MyFlowHub-Server` 在 `GOWORK=off` 下仍依赖尚未对齐的 `Proto/Core/auth` 版本。
- 本次目标是补齐发布链，确认不依赖本地 `go.work` 时，`auth` module 和 `Server` 仍可通过依赖解析、编译和关键测试。

## 具体变更内容
### 发布顺序收口
- `MyFlowHub-Proto`
  - 发布 tag：`v0.1.2`
  - 承载 auth admission 新字段、状态和 action schema。
- `MyFlowHub-Core`
  - 发布 tag：`v0.4.8`
  - 承载 auth admission config、permission 常量和 `SelfRegisterOptions.JoinPermit`。
- `MyFlowHub-SubProto/auth`
  - 对齐依赖到 `myflowhub-proto v0.1.2` 与 `myflowhub-core v0.4.8`
  - 发布 module tag：`auth/v0.1.4`
- `MyFlowHub-Server`
  - 对齐依赖到：
    - `github.com/yttydcs/myflowhub-proto v0.1.2`
    - `github.com/yttydcs/myflowhub-core v0.4.8`
    - `github.com/yttydcs/myflowhub-subproto/auth v0.1.4`

### 发布操作策略
- `Proto/Core` 主线控制面本地历史超前较多，为避免把无关提交一起推到远端，本次使用基于远端基线的 release worktree 做 cherry-pick 后打 tag。
- `auth` module 尝试复用 release worktree 时，遇到与本地未上远端的显示名基线相关冲突；最终改为直接在已验证通过的 feature worktree commit 上打 `auth/v0.1.4`，避免引入额外发布噪音。
- 对已 push 的 semver tag 继续遵守“不改写历史”的策略；若后续发现问题，只追加更高 patch 修复。

### 单仓验证补齐
- 验证已发布上游 tag 可被外部拉取：
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-proto@v0.1.2`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-core@v0.4.8`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-subproto/auth@v0.1.4`
- 验证下游不依赖本地 `go.work` 仍能通过关键测试：
  - `MyFlowHub-SubProto/auth`: `GOWORK=off go test ./... -count=1 -p 1`
  - `MyFlowHub-Server`: `GOWORK=off go test ./hubruntime/... -count=1 -p 1`
  - `MyFlowHub-Server`: `GOWORK=off go test ./tests -run TestLoginHandler -count=1 -p 1`

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
- `REL1` - 发布 `MyFlowHub-Proto v0.1.2` 与 `MyFlowHub-Core v0.4.8`
- `AUTH6` - 对齐并发布 `auth/v0.1.4`
- `SRV3` - Server semver 依赖对齐
- `TEST2` - `GOWORK=off` 回归验证
- `REV1` - 3.3 Code Review
- `ARC1` - 本次归档

## 经验 / 教训摘要
- workflow-local `go.work` 联调通过，不等于跨仓发布工作流已经完成。
- 只要本轮涉及上游 API 或公共常量变更，结束前必须至少对一个真实下游执行 `GOWORK=off` 验证。
- 本次发布顺序必须是 `Proto/Core -> auth -> Server`；跳过上游 tag 发布会让下游 `go.mod` 对齐变成假成功。

## 可复用排查线索
- 症状：
  - 本地 `go.work` 联调通过，但 `GOWORK=off go test` 编译失败。
  - 下游仓库升级代码后，外部环境仍提示缺少新字段、常量或方法。
- 触发条件：
  - 本轮改动跨越多个仓库，且下游引用了尚未发布的新 API。
  - `go.mod` 仍指向旧 tag，或远端尚无对应 semver tag。
- 关键词：
  - `GOWORK=off`
  - `undefined: config.KeyParentJoinPermit`
  - `undefined: bootstrap.SelfRegisterOptions.JoinPermit`
  - `admission types`
  - `go list -m`
- 快速检查：
  - 先对目标 tag 执行 `go list -m`，确认远端可拉取。
  - 再检查下游 `go.mod` 是否已经对齐到本轮目标版本。
  - 最后在至少一个真实下游执行 `GOWORK=off go test`。

## 关键设计决策与权衡
- 以 semver tag 作为发布与审计基线，而不是把 `go.work` 或本地 `replace` 当作最终交付证据。
- 为避免把控制面主线里无关 ahead 历史一起推到远端，优先使用 release worktree 或已验证通过的最小 commit 作为打 tag 基线。
- 对 `auth` module 冲突场景，选择“在已验证 commit 上直接发 module tag”而不是硬清理冲突后的临时基线，优先保证发布链真实可复现。

## 测试与验证方式 / 结果
- `GOWORK=off go list -m github.com/yttydcs/myflowhub-proto@v0.1.2`
  - 结果：通过
- `GOWORK=off go list -m github.com/yttydcs/myflowhub-core@v0.4.8`
  - 结果：通过
- `GOWORK=off go list -m github.com/yttydcs/myflowhub-subproto/auth@v0.1.4`
  - 结果：通过
- `MyFlowHub-SubProto/auth`: `GOWORK=off go test ./... -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-Server`: `GOWORK=off go test ./hubruntime/... -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-Server`: `GOWORK=off go test ./tests -run TestLoginHandler -count=1 -p 1`
  - 结果：通过

## 潜在影响
- 从本次开始，受控准入链路在发布口径上不再依赖 workspace 本地 `go.work`；下游必须升级到新 tag 才能拿到 admission / permit / authority strict selection 的稳定 API。
- 若有外部仓库仍钉在旧版 `Proto/Core/auth`，会继续表现为“代码看起来对了，但构建缺符号”。

## 回滚方案
- 已 push 的 semver tag 不回滚、不改写；若发现问题，改发更高 patch 版本修正。
- 对尚未 push 的本地主线 merge，可通过常规 revert 或后续 fix commit 处理，但不影响已发布 tag 的可拉取性。

## 子Agent执行轨迹
- 无。全部由主 agent 在当前 workflow 中完成。
