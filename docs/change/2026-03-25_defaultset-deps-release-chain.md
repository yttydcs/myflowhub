# 2026-03-25 Defaultset 依赖链正式发布与下游对齐

## 变更背景 / 目标
- `MyFlowHub-Server/modules/defaultset` 已切到显式 `WithDeps` / `runtimedeps` 装配，但真实 semver 发布链没有同步收口。
- 初始暴露症状是：
  - `undefined: filehandler.NewHandlerWithDeps`
  - `undefined: management.NewHandlerWithDeps`
- 在 `GOWORK=off` 下继续追查后，确认这不是两个孤立符号缺失，而是整条上游依赖链缺口：
  - `myflowhub-proto v0.1.2` 缺失 `flow delete` 契约
  - `myflowhub-subproto/broker v0.1.0` 缺失 `SharedExecCapQueryBroker`
  - `myflowhub-subproto/exec v0.1.1` 缺失 `capability` / `runtimedeps`
  - `file/flow/topicbus/varstore/management` 的已发布 patch 版本也未覆盖当前 `WithDeps` 入口
- 本次目标：
  - 补齐 `Proto -> SubProto -> Server` 的 patch release chain；
  - 确认 `Server` 在 `GOWORK=off` 下可以通过真实依赖解析完成构建和测试；
  - 归档跨仓排查路径和可复用 lesson。

## 具体变更内容
### 上游发布
- `MyFlowHub-Proto`
  - 发布 `v0.1.3`
  - 承载 `flow delete` 协议契约：
    - `ActionDelete`
    - `ActionDeleteResp`
    - `PermFlowDelete`
    - `DeleteReq`
    - `DeleteResp`
- `MyFlowHub-SubProto`
  - 发布 `broker/v0.1.1`
  - 发布 `exec/v0.1.2`
  - 发布 `file/v0.1.4`
  - 发布 `flow/v0.1.2`
  - 发布 `topicbus/v0.1.2`
  - 发布 `varstore/v0.1.4`
  - 发布 `management/v0.1.4`
  - 对齐的依赖收口：
    - `exec` 对齐 `broker v0.1.1`
    - `flow` 对齐 `proto v0.1.3` 与 `exec v0.1.2`
    - `file/topicbus/varstore/management` 对齐 `exec v0.1.2`

### 下游对齐
- `MyFlowHub-Server`
  - `myflowhub-proto v0.1.2 -> v0.1.3`
  - `myflowhub-subproto/exec v0.1.0 -> v0.1.2`
  - `myflowhub-subproto/file v0.1.2 -> v0.1.4`
  - `myflowhub-subproto/flow v0.1.0 -> v0.1.2`
  - `myflowhub-subproto/management v0.1.2 -> v0.1.4`
  - `myflowhub-subproto/topicbus v0.1.0 -> v0.1.2`
  - `myflowhub-subproto/varstore v0.1.2 -> v0.1.4`
  - 间接依赖 `myflowhub-subproto/broker v0.1.0 -> v0.1.1`

### 验证环境修复
- 过程中发现本机 `go1.25.0` 下载工具链缓存损坏，导致误报：
  - `package context is not in std`
- 已通过定点重拉损坏的 `golang.org/toolchain@v0.0.1-go1.25.0.windows-amd64` 恢复验证环境。
- 该问题不属于仓库代码缺陷，但影响了正式收口，需要在归档中记录。

## Requirements impact
`none`

## Specs impact
`none`

## Lessons impact
`updated`

## Related requirements
- `none`

## Related specs
- `none`

## Related lessons
- [cross-repo-semver-release.md](../lessons/cross-repo-semver-release.md)

## 对应 plan.md 任务映射
- `PROTOREL1` - 发布 `MyFlowHub-Proto v0.1.3`
- `SUBREL1` - 确认 `defaultset` 依赖链发布边界
- `SUBREL2` - 完成 `MyFlowHub-SubProto` repo-local 联调验证
- `SUBREL3` - 发布 `broker/exec/file/flow/topicbus/varstore/management` patch tags
- `SRVREL1` - 升级 `MyFlowHub-Server` 依赖链
- `VAL1` - 在真实依赖解析模式下完成构建与测试验证
- `DOC1` - 完成 repo/workspace archive 与索引更新

## 经验 / 教训摘要
- 最先报错的两个 `undefined` 只是入口症状，不能据此缩小修复范围；必须沿依赖方向继续追到完整发布链。
- `go.work` 或本地 sibling worktree 只能证明联调通过，不能证明真实 semver 消费成立。
- 同仓 shared package 新增后，所有依赖该 package 的 sibling modules 都要重新核对 `go.mod` 约束。
- `GOWORK=off` 验证前，先确认本机 Go 工具链缓存没有被污染，否则会把环境故障误判成代码回归。

## 可复用排查线索
- 症状
  - `undefined: filehandler.NewHandlerWithDeps`
  - `undefined: management.NewHandlerWithDeps`
  - `undefined: broker.SharedExecCapQueryBroker`
  - `undefined: protocol.ActionDelete`
  - `no required module provides package github.com/yttydcs/myflowhub-subproto/exec/runtimedeps`
  - `no required module provides package github.com/yttydcs/myflowhub-subproto/exec/capability`
- 触发条件
  - `Server` 或其他下游单仓构建 `defaultset`
  - 本地已引入 `WithDeps` / `runtimedeps` / `delete` 契约，但上游 semver tag 未同步
  - 用 `go.work` 联调通过后直接跳过真实依赖解析验证
- 关键词
  - `defaultset`
  - `WithDeps`
  - `runtimedeps`
  - `SharedExecCapQueryBroker`
  - `ActionDelete`
  - `GOWORK=off`
  - `go list -m`
- 快速检查
  - 先看 `go list -m github.com/yttydcs/myflowhub-proto`
  - 再看 `go list -m github.com/yttydcs/myflowhub-subproto/broker`
  - 再看 `go list -m github.com/yttydcs/myflowhub-subproto/exec`
  - 核对 `file/flow/topicbus/varstore/management` 是否仍锁在旧 `exec`
  - 若出现 `package <stdlib> is not in std`，先检查本机 `go env GOROOT` 与下载工具链缓存

## 关键设计决策与权衡
- 采用“沿依赖方向补齐发布链”的正式修复，而不是回退 `Server/defaultset` 代码：
  - 能保留显式 runtime deps 设计，不引入架构倒退。
- 把 `Proto`、`broker`、`exec` 和所有 `defaultset` module 一次性纳入同一 release chain：
  - 避免只修第一个错误后，后续继续在下游反复爆出新的缺包或缺符号。
- 验证口径以真实依赖解析为准：
  - 上游内部联调使用 repo-local `go.work`
  - 正式有效性由下游 `GOWORK=off` 构建和测试给证据

## 测试与验证方式 / 结果
- `MyFlowHub-SubProto`
  - repo-local `go.work` 绑定本地 `Core/Proto` 与相关 modules
  - `broker/exec/file/flow/topicbus/varstore/management` 分模块 `go test ./... -count=1 -p 1`
  - 结果：通过
- `MyFlowHub-Server`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-proto`
    - 结果：`v0.1.3`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-subproto/broker`
    - 结果：`v0.1.1`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-subproto/exec`
    - 结果：`v0.1.2`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-subproto/file`
    - 结果：`v0.1.4`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-subproto/flow`
    - 结果：`v0.1.2`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-subproto/topicbus`
    - 结果：`v0.1.2`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-subproto/varstore`
    - 结果：`v0.1.4`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-subproto/management`
    - 结果：`v0.1.4`
  - `GOWORK=off go build ./...`
    - 结果：通过
  - `GOWORK=off go test ./... -count=1 -p 1`
    - 结果：通过

## 潜在影响与回滚方案
- 潜在影响
  - 若外部下游仍锁定旧 patch 版本，会继续看到相同缺符号错误。
  - 若本地 tag 未推送远端，则“正式修复”对外仍不成立。
- 回滚方案
  - 对尚未公开的本地分支或 tag，可删除本地 tag 并回退版本文件。
  - 对已公开的 semver tag，不重写历史；若发现问题，改发更高 patch 版本覆盖。

## 子Agent执行轨迹
- 无。全部由主 agent 完成。
