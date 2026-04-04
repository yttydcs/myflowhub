# 2026-04-04_core-bootstrap-dialer-release-align

## 变更背景 / 目标
- 在上一轮 `flow` 发布链收口之后，`MyFlowHub-Server` 仍然在 `GOWORK=off` 下报：
  - `hubruntime/runtime.go:466:3: unknown field Dial in struct literal of type bootstrap.SelfRegisterOptions`
- 用户已明确选择“升级依赖包版本”而不是回退 `Server` 已落地的 bootstrap endpoint dialer 改造。
- 本次目标是在不扩 scope 到 Proto / SubProto / Win 的前提下，完成：
  - `MyFlowHub-Core` 最小 patch release 对齐
  - `MyFlowHub-Server` 对 `myflowhub-core v0.4.10` 的真实消费验证
  - 相关 semver 排障经验沉淀

## 具体变更内容
### MyFlowHub-Core
- 分支：
  - `release/bootstrap-dialer-v0.4.10`
- 版本基线：
  - `v0.4.9`
- 本轮确认的最小必要 patch：
  - `e542301 feat: bootstrap 支持通用 endpoint dialer`
  - `d4cf011 feat: flow 运行控制一期权限默认值补齐`
- 对齐结果：
  - 本地 `v0.4.10` tag 已重新指向 release branch HEAD `d4cf011`
  - 该 tag 现在同时包含：
    - `bootstrap.SelfRegisterOptions.Dial`
    - `config.DefaultAuthRolePerms` 中的 `flow.run` / `flow.read`
- 远端发布状态：
  - 已完成
  - branch：
    - `release/bootstrap-dialer-v0.4.10`
  - tag：
    - `v0.4.10`
  - `git ls-remote origin refs/heads/release/bootstrap-dialer-v0.4.10 refs/tags/v0.4.10`
    - 结果：
      - `d4cf011cf8c1a1aaf1a054adf9574fc8bc662bab refs/heads/release/bootstrap-dialer-v0.4.10`
      - `2649f82cadbf71b941ef18a1df7116fe191c5678 refs/tags/v0.4.10`
  - 判断：Core release branch 与 tag 已经对外可见

### MyFlowHub-Server
- 提交：
  - `0b3d422 chore: bump myflowhub-core to v0.4.10`
- 修改：
  - `go.mod`
    - `github.com/yttydcs/myflowhub-core`：`v0.4.9 -> v0.4.10`
  - `go.sum`
    - 更新到当前 `v0.4.10` 的正确校验和
- 说明：
  - 本轮没有修改 `hubruntime` / `cmd/hub_server` 代码
  - 只做正式 semver 消费对齐

### Control / Lessons
- 更新：
  - `docs/lessons/cross-repo-semver-release.md`
    - 补充“未公开 tag 在本地被重指向后，旧 module cache / go.sum 造成假性漂移”的案例
  - `docs/lessons/README.md`
    - 补充对应检索关键词
  - `docs/change/README.md`
    - 增补本次 archive 索引

## Requirements impact
`none`

## Specs impact
`none`

## Lessons impact
`updated`

## Related requirements
- 无

## Related specs
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\core.md`

## Related lessons
- [cross-repo-semver-release.md](../lessons/cross-repo-semver-release.md)

## 对应 plan.md 任务映射
- `BOOTREL-1` - Core clean release
- `BOOTREL-2` - Server dependency align
- `BOOTREL-3` - Review / archive / lesson routing

## 经验 / 教训摘要
- `go.work` 联调通过只能证明本地代码兼容，不能证明正式 semver 发布链已经闭环。
- 未公开的 tag 即使还没 push，只要本地 downstream 曾经抓取过该版本，后续再把 tag 重指向新 commit，也会让 `go` 继续命中旧 zip 和旧 `go.sum` 哈希。
- 这类缓存污染的首个表象不一定是 `checksum mismatch`，也可能先表现为“版本号已经对了，但默认值还是旧的”。
- 当 GitHub 连通性异常时，可以先用本地 git rewrite + `GOWORK=off` 完成真实依赖模拟，但它不能替代最终远端发布。

## 可复用排查线索
- 症状：
  - `go list -m github.com/yttydcs/myflowhub-core` 已经显示 `v0.4.10`
  - `Dial` 缺字段错误消失了，但 `hubruntime/options_test.go` 仍然看到旧的 `DefaultAuthRolePerms`
  - 重新拉取同一版本后报 `checksum mismatch`
  - `git ls-remote origin ...` 直接报 `Recv failure: Connection was reset`
- 触发条件：
  - 本地用 file rewrite 模拟 semver 发布
  - 同一未公开版本号的 tag 在第一次抓取后被重新指向了新 commit
  - `go.sum` 已经记录了旧 zip 的哈希
- 关键词：
  - `bootstrap.SelfRegisterOptions.Dial`
  - `DefaultAuthRolePerms`
  - `flow.run`
  - `flow.read`
  - `checksum mismatch`
  - `go.sum`
  - `Recv failure: Connection was reset`
- 快速检查：
  - 先看上游源码里目标 tag 指向的 commit 是否真的包含目标字段 / 默认值
  - 若同版本号曾被本地抓取过，删除该 module 的：
    - `pkg/mod/<module>@<version>`
    - `pkg/mod/cache/download/.../@v/<version>.*`
    - `pkg/mod/cache/vcs/<repo-cache>`
  - 同步清理 `go.sum` 中该版本的旧记录后再重下
  - 最后用 `GOWORK=off + GOPROXY=direct + GOPRIVATE + GONOSUMDB` 重跑真实 downstream 验证

## 关键设计决策与权衡
- 保持 `v0.4.10`，不直接顺延到 `v0.4.11`
  - 原因：该 tag 尚未公开，当前问题只发生在本地缓存层；校正 tag 指向并刷新本地缓存即可保持最小 release 面
  - 代价：本地验证步骤必须额外处理 stale cache / checksum
- `Server` 继续采用“依赖升级，不改运行时代码”的最小策略
  - 好处：回归面最小，和用户选择的方案一致
  - 代价：需要额外做一次去掉本地 rewrite 的真实远端复验
- 不扩 scope 到 SSH / 代理替代发布链
  - 原因：在 HTTPS 已恢复并成功 push 后，继续折腾替代链路没有必要

## 测试与验证方式 / 结果
- Core release branch 既有验证：
  - `GOWORK=off go test ./bootstrap -count=1`
    - 结果：通过
  - `GOWORK=off go test ./... -count=1`
    - 结果：通过

### MyFlowHub-Server 本地 semver 模拟验证环境
- 环境：
  - `GOWORK=off`
  - `GOPROXY=direct`
  - `GOPRIVATE=github.com/yttydcs/*`
  - `GONOSUMDB=github.com/yttydcs/*`
  - `GIT_CONFIG_NOSYSTEM=1`
  - `GIT_CONFIG_GLOBAL=D:\project\MyFlowHub3\.tmp\gitconfig-core-local.txt`

### 首轮异常验证
- `go mod download github.com/yttydcs/myflowhub-core@v0.4.10`
  - 结果：失败
  - 错误：`checksum mismatch`
- 结论：
  - `go.sum` 记录的是 tag 旧指向对应的哈希
  - 需要清理 module cache 与旧 checksum 后重下，不能把它误判为新的 Server / Core API 漂移

### 刷新 cache 后的最终验证
- `go list -m github.com/yttydcs/myflowhub-core`
  - 结果：`v0.4.10`
- `go test ./hubruntime -count=1`
  - 结果：通过
- `go test ./cmd/hub_server -count=1`
  - 结果：通过
- `go test ./modules/... -count=1 -run TestDefaultHub_ContainsFlow`
  - 结果：通过
- `go test ./tests -count=1 -run '^(TestRootHubPing|TestIntegrationVarStoreSetGetAcrossHub)$'`
  - 结果：通过

### 远端发布与真实 GitHub 复验
- `git ls-remote origin refs/heads/release/bootstrap-dialer-v0.4.10 refs/tags/v0.4.10`
  - 结果：通过
- 在未设置 `GIT_CONFIG_GLOBAL` rewrite 的前提下执行：
  - `go mod download github.com/yttydcs/myflowhub-core@v0.4.10`
  - `go list -m github.com/yttydcs/myflowhub-core`
  - `go test ./hubruntime -count=1`
  - `go test ./cmd/hub_server -count=1`
  - `go test ./modules/... -count=1 -run TestDefaultHub_ContainsFlow`
  - `go test ./tests -count=1 -run '^(TestRootHubPing|TestIntegrationVarStoreSetGetAcrossHub)$'`
  - 结果：全部通过
  - 判断：`MyFlowHub-Server` 已经在真实 GitHub 远端依赖下完成消费验证

## Code Review（3.3）结论
- 需求覆盖：通过
- 架构合理性：通过
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
- 可读性与一致性：通过
- 可扩展性与配置化：通过
- 稳定性与安全：通过
- 测试覆盖情况：通过
- 子Agent治理与审计：通过
  - `Dirac` 只读审计 `BOOTREL-2B`
  - 结论：除 `SelfRegisterOptions.Dial` 与 `DefaultAuthRolePerms` 之外，未发现额外直接 Core semver 漂移
  - 文件变更：无

## 潜在影响
- 正向影响：
  - `MyFlowHub-Server` 已经具备消费 `myflowhub-core v0.4.10` 的最小提交和通过验证的 `go.sum`
  - `Dial` 与默认 `flow.run` / `flow.read` 依赖链已在真实 GitHub 远端依赖下闭合
- 注意事项：
  - 只要本机还保留旧的 `v0.4.10` module cache，就可能再次复现 checksum 或旧默认值症状

## 回滚方案
- `MyFlowHub-Server`
  - 可直接回退提交 `0b3d422`
- `MyFlowHub-Core`
  - `v0.4.10` 已公开；后续若发现问题，不改写历史 tag，只追加更高 patch 修复
- 控制面文档
  - 在 workflow 未结束前，可单独回退本 worktree 的归档与 lesson 更新

## 子Agent执行轨迹
- `Dirac`
  - 阶段：`3.2`
  - Task ID：`BOOTREL-2B`
  - 类型：只读静态审计
  - 结论：确认 `Server` 直接依赖的 Core 新面只有：
    - `bootstrap.SelfRegisterOptions.Dial`
    - `coreconfig.DefaultAuthRolePerms`
  - 风险提醒：
    - 显式空 `auth.role_perms` 不会自动回填默认值
    - 新 transport 仍需额外关注 injected dialer 的运行期连接契约
