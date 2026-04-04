# Cross Repo Semver Release

## Summary
- 跨仓 API 变更如果只在 workspace 本地 `go.work` 联调通过，仍可能在真实下游仓库的 `GOWORK=off` 模式下失败。
- 这类问题通常不是业务逻辑 bug，而是“上游 tag 未发布”“下游 `go.mod` 未对齐”，或者“下游 CI 仍 checkout 到旧的 sibling repo 基线”的发布链问题。
- 新 tag 刚 push 后若直接经由公共 proxy 校验，还可能被 proxy lag 伪装成 `unknown revision`；需要先排除缓存传播延迟，再判断 tag 是否真的有问题。
- 对未公开 tag 的本地 semver 模拟，如果同一版本号在首次抓取后又被重新指向新 commit，还会让本机 module cache 和 `go.sum` 继续锁在旧 zip 上，表象可能是 `checksum mismatch`，也可能是“版本号对了但默认值还是旧的”。
- 对 `Server/defaultset` 这类跨多个 subproto module 的装配入口，最先报出的 1 到 2 个缺符号通常只是入口症状，根因往往藏在更上游的同链路 module 或协议版本。

## Lookup Hints
- 症状：
  - `go.work` 下测试全部通过
  - `GOWORK=off go test` 编译失败
  - 本地 worktree 测试通过，但远端 CI/tag release 仍打出旧行为
- 关键词：
  - `GOWORK=off`
  - `go list -m`
  - `GOPROXY=direct`
  - `GOPRIVATE=github.com/yttydcs/*`
  - `GONOSUMDB=github.com/yttydcs/*`
  - `unknown revision`
  - `undefined: protocol.ActionAuthorityPolicySync`
  - `undefined: protocol.ActionListRegisterPermits`
  - `undefined: coreconfig.DefaultAuthBootstrapFirstRegisterRole`
  - `undefined: filehandler.NewHandlerWithDeps`
  - `undefined: management.NewHandlerWithDeps`
  - `undefined: broker.SharedExecCapQueryBroker`
  - `undefined: protocol.ActionDelete`
  - `no required module provides package github.com/yttydcs/myflowhub-subproto/exec/runtimedeps`
  - `undefined: config.KeyParentJoinPermit`
  - `undefined: bootstrap.SelfRegisterOptions.JoinPermit`
  - `undefined: bootstrap.SelfRegisterOptions.Dial`
  - `DefaultAuthRolePerms`
  - `flow.run`
  - `flow.read`
  - `checksum mismatch`
  - `go.sum`
  - `flow/v0.1.3`
  - `flow/v0.1.4`
  - `Checkout Server (for hubmobile replace)`
  - `actions/checkout`
  - `default branch`
- 快速检查：
  - 对刚 push 的 tag 首轮验证优先改用：
    - `GOPROXY=direct`
    - `GOPRIVATE=github.com/yttydcs/*`
    - `GONOSUMDB=github.com/yttydcs/*`
  - 对目标版本执行 `go list -m`
  - 检查下游 `go.mod` 是否已升级到本轮发布版本
  - 检查下游 CI 是否用 `replace` 拉 sibling repo，以及它实际 checkout 的 ref 是什么
  - 若同一未公开版本号在本机被重新打到新 commit，先清理该 module 的本地 cache 与旧 `go.sum`，再判断是否真有代码漂移

## Symptoms
- 代码在本地多仓联调环境可编译、可测试。
- 一旦关闭 `go.work` 或切到单仓环境，立即出现找不到新类型、常量或字段的错误。
- 对 submodule 来说，还可能表现为：当前源码已经引用了上游新符号，但 module 自己的 `go.mod` 最低版本仍停在旧 tag。
- 常见表现是：修完首个缺符号后，继续暴露 sibling module 的 shared package 缺失，或继续暴露更上游 `Proto` 契约缺失。
- 或者本地 workflow worktree 测试通过，但远端 release workflow 仍因为 checkout 到旧 sibling repo 基线而产出旧行为。

## Impact
- 阻塞 semver 发布和下游依赖升级。
- 容易让 workflow 在“本地看起来全绿”时被过早判定完成。
- 如果不显式记录，后续团队成员会重复踩同一个坑。

## Trigger Conditions
- 本轮改动跨越 `Proto`、`Core`、`SubProto`、`Server` 等多个仓库。
- 下游仓库依赖了上游刚新增的 API，但上游对应 tag 还没有发布。
- 子 module 已经引用上游新 API，但 module `go.mod` 的最低依赖版本没有同步抬高。
- 仅使用 workspace 本地 `go.work` 做联调验证，没有补 `GOWORK=off` 检查。
- 下游仓库通过 `replace` 依赖 sibling repo，而 CI/release workflow checkout 的是该 sibling repo 的远端默认分支或旧基线。
- 本地用 file rewrite 或其他方式模拟 semver 发布时，同一未公开版本号的 tag 在首次抓取后又被重新指向新 commit。

## Root Cause
- `go.work` 会把本地 worktree 直接接到编译链上，掩盖远端 semver 依赖尚未发布或下游版本尚未升级的问题。
- 对单仓多 module 结构，`go.work` 也会掩盖“module 源码实际依赖更高版本上游，但 `go.mod` 仍声明旧最低版本”的问题。
- 同样地，`replace ../../SomeRepo` 在本地 workflow worktree 可以指向最新 sibling worktree，但远端 CI 如果只是 checkout 远端默认分支，就不会自动拿到你本地尚未公开的代码。
- 公共 module proxy 对刚 push 的 tag 存在刷新延迟；如果不先绕过 proxy，就容易把缓存传播延迟误判为真正的 release-chain 失败。
- 对未公开 tag 的本地重指向不会自动刷新本机 `pkg/mod` 与 `go.sum`；第一次抓到的 zip 和哈希会继续污染后续验证。
- 一旦切回真实依赖解析或真实 CI checkout 模式，隐藏问题就会立刻暴露。

## Investigation Trail
1. 在 workflow-local `go.work` 下确认功能与测试都通过。
2. 切换到 `GOWORK=off` 后，在真实下游复现缺符号或依赖不一致问题。
3. 反查缺失符号对应的上游来源，确认新 API 仍只存在于本地 worktree。
4. 若报错发生在上游 submodule 自身，检查该 module `go.mod` 是否仍停在旧最低版本。
5. 若下游使用 sibling repo `replace`，继续检查 CI/release workflow 中 `actions/checkout` 拉取的是哪个 repo、哪个 ref。
6. 先发布上游 tag，或先把 CI 依赖的 sibling repo 基线公开，再执行下游 release。
7. 如果本地曾经抓取过一个后来被重指向的未公开 tag，删除该 module 的下载缓存、VCS 缓存和 `go.sum` 旧记录，再重下验证。

## Resolution
- 发布：
  - `myflowhub-proto v0.1.2`
  - `myflowhub-core v0.4.8`
  - `myflowhub-subproto/auth v0.1.4`
  - `myflowhub-sdk v0.1.11`
  - `myflowhub-win v0.0.11`
- 将 `MyFlowHub-Server` 对齐到上述 auth admission 上游版本。
- 对关键下游补充 `GOWORK=off` 构建和测试验证，作为 workflow 收口条件。
- 对仍依赖 sibling repo `replace` 的应用仓，在确认 CI checkout 到正确上游基线前，不要直接发 release tag。

### Flow v0.1.4 release-chain example
- 发布：
  - `myflowhub-proto v0.1.6`
  - `myflowhub-subproto/flow v0.1.4`
- 现象：
  - `flow/v0.1.3` 首次公开后，`Server` 侧立即校验仍失败
  - 经 `GOPROXY=direct` 复验后，确认问题不只是 proxy lag，而是 `flow/v0.1.3` 缺少 `myflowhub-proto v0.1.6` 的 `go.sum`
- 修复：
  - 不改写 `flow/v0.1.3`
  - 直接补发 `flow/v0.1.4`
  - 在 `Server` 侧用 `GOWORK=off` + direct fetch 环境执行 `go list -m` 和定向模块测试，确认真实依赖链解析到 `Proto v0.1.6` / `flow v0.1.4`

### Auth v0.1.5 minimum-version drift example
- 症状：
  - `MyFlowHub-SubProto/auth` 在 workspace 联调正常，但 `GOWORK=off go test ./...` 报：
    - `undefined: protocol.ActionAuthorityPolicySync`
    - `undefined: protocol.ActionListRegisterPermits`
    - `undefined: coreconfig.DefaultAuthBootstrapFirstRegisterRole`
- 根因：
  - `auth` 源码已经依赖 `Proto v0.1.5` 与 `Core v0.4.9` 的新符号
  - 但 `auth/go.mod` 仍停在 `Proto v0.1.2` 与 `Core v0.4.8`
- 修复：
  - 先把 `auth/go.mod/go.sum` 提升到真实最低依赖版本
  - 再发布 `auth/v0.1.5`
  - 最后让 `MyFlowHub-Server` 升级到 `auth v0.1.5` 并执行 `GOWORK=off go test ./...`

### Defaultset release-chain example
- 发布：
  - `myflowhub-proto v0.1.3`
  - `myflowhub-subproto/broker v0.1.1`
  - `myflowhub-subproto/exec v0.1.2`
  - `myflowhub-subproto/file v0.1.4`
  - `myflowhub-subproto/flow v0.1.2`
  - `myflowhub-subproto/topicbus v0.1.2`
  - `myflowhub-subproto/varstore v0.1.4`
  - `myflowhub-subproto/management v0.1.4`
- 将 `MyFlowHub-Server` 对齐到上述版本，并在 `GOWORK=off` 下执行 `go build ./...` 和 `go test ./... -count=1 -p 1`。
- 如果 first error 是 `NewHandlerWithDeps`，不要止步于 `file/management` 两个 module；继续检查 `broker`、`exec` 与 `Proto` 契约版本。

### Core v0.4.10 unpublished-retag cache example
- 症状：
  - `MyFlowHub-Server` 已把 `myflowhub-core` 升级到 `v0.4.10`
  - `go list -m github.com/yttydcs/myflowhub-core` 也显示 `v0.4.10`
  - 但 `hubruntime/options_test.go` 仍看到旧的 `DefaultAuthRolePerms`，缺少 `flow.run` / `flow.read`
  - 清理 module cache 后再次下载同版本，会报 `checksum mismatch`
- 根因：
  - 本地第一次抓取 `v0.4.10` 时，tag 还指向只包含 `Dial` 的旧 commit
  - 后续虽然把未公开的 `v0.4.10` 重新指到包含 `flow.run` / `flow.read` 的 commit，本机 `pkg/mod` 和 `go.sum` 仍保留第一次下载的 zip / hash
- 修复：
  - 先确认源码与 tag 现在确实指到正确 commit
  - 删除该 module 的：
    - `pkg/mod/github.com/yttydcs/myflowhub-core@v0.4.10`
    - `pkg/mod/cache/download/github.com/yttydcs/myflowhub-core/@v/v0.4.10.*`
    - `pkg/mod/cache/vcs/<myflowhub-core-cache>`
  - 删除 `go.sum` 中该版本的旧 hash
  - 再用 `GOWORK=off + GOPROXY=direct + GOPRIVATE + GONOSUMDB` 重下并重跑 downstream 测试
- 结论：
  - 这种症状是本地 semver 模拟缓存污染，不应误判成新的 Core / Server 代码漂移

## Prevention / Guardrails
- 只要本轮改动触达上游公共 API，结束前必须至少执行一次真实下游的 `GOWORK=off go test`。
- 对单仓多 module，还必须检查每个目标 module 自身的 `go.mod` 最低版本是否与当前源码一致。
- 发布顺序按依赖方向执行：`Proto/Core -> SubProto module -> SDK/Server -> 应用层`。
- 同仓 shared package 新增后，继续沿依赖方向检查所有 sibling modules 的最小版本，而不是只修首个报错 module。
- 如果下游 CI 依赖 `replace` 到 sibling repo，必须检查该 sibling repo 的远端默认分支或 checkout ref 是否已经包含本轮代码。
- 刚 push tag 的首轮验收优先绕开 proxy，再决定是缓存传播问题还是 tag / module 本身的问题。
- 已 push 的 semver tag 不重写；若发现缺 `go.sum`、module 自测不完整等问题，直接发更高 patch 版本。
- 即使 tag 尚未公开，只要某个版本号已经在本机被抓取过，再重新指向时也要同步刷新 module cache 与 `go.sum`，否则本地验证会继续吃旧内容。
- 不要把 `go.work` 通过误判成“可发布”证据；它只能证明本地联调成功。

## Related Docs
- Requirements:
  - [auth-controlled-admission.md](../requirements/auth-controlled-admission.md)
- Specs:
  - [auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md)
- Changes:
  - [2026-03-24_auth-admission-downstream-followup.md](../change/2026-03-24_auth-admission-downstream-followup.md)
  - [2026-03-23_auth-controlled-admission.md](../change/2026-03-23_auth-controlled-admission.md)
  - [2026-03-24_auth-authority-strict-selection.md](../change/2026-03-24_auth-authority-strict-selection.md)
  - [2026-03-24_auth-admission-release-chain.md](../change/2026-03-24_auth-admission-release-chain.md)
  - [2026-03-25_defaultset-deps-release-chain.md](../change/2026-03-25_defaultset-deps-release-chain.md)
  - [2026-04-04_flow-release-chain-align.md](../change/2026-04-04_flow-release-chain-align.md)
  - [2026-04-04_core-bootstrap-dialer-release-align.md](../change/2026-04-04_core-bootstrap-dialer-release-align.md)
