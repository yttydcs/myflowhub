# 2026-03-28 Cross Repo 依赖版本收口

## 变更背景 / 目标
- 近期已经把多个仓库的最新提交推到远端并补了新 tag，但消费方仓库仍存在旧 patch、proto pseudo-version 和过时的子模块依赖。
- 这会导致：
  - 单仓 `GOWORK=off` 构建仍停在旧依赖基线
  - Android / MetricsNode / Win 等上层仓库与最新发布版本脱节
  - 后续继续打 tag 时，发布内容和实际依赖链不一致
- 本次目标：
  - 把当前已发布的上游 tag 收口到所有消费方仓库
  - 保持改动范围严格停留在 `go.mod/go.sum`
  - 用真实依赖解析模式完成回归验证

## 具体变更内容
### MyFlowHub-SDK
- `go.mod/go.sum`
  - `github.com/yttydcs/myflowhub-core`: `v0.4.7 -> v0.4.9`
  - `github.com/yttydcs/myflowhub-proto`: `v0.1.1 -> v0.1.5`

### MyFlowHub-Server
- `go.mod/go.sum`
  - `github.com/yttydcs/myflowhub-core`: `v0.4.8 -> v0.4.9`
  - `github.com/yttydcs/myflowhub-proto`: `v0.1.3 -> v0.1.5`
  - `github.com/yttydcs/myflowhub-subproto/forward`: `v0.1.0 -> v0.1.1`
- 其余 `auth/exec/file/flow/management/topicbus/varstore/broker` 维持当前最新 patch，不引入计划外模块变更

### MyFlowHub-Win
- `go.mod/go.sum`
  - `github.com/yttydcs/myflowhub-core`: `v0.4.7 -> v0.4.9`
  - `github.com/yttydcs/myflowhub-proto`: `v0.1.2-0.20260318063708-7eef50dcc471 -> v0.1.5`
  - `github.com/yttydcs/myflowhub-sdk`: `v0.1.10 -> v0.1.12`
- 收口后不再依赖 proto pseudo-version

### MyFlowHub-MetricsNode
- 根模块 `go.mod/go.sum`
  - `github.com/yttydcs/myflowhub-core`: `v0.4.0 -> v0.4.9`
  - `github.com/yttydcs/myflowhub-proto`: `v0.1.1 -> v0.1.5`
  - `github.com/yttydcs/myflowhub-sdk`: `v0.1.4 -> v0.1.12`
- `windows/go.mod/go.sum`
  - 对齐同一组 `core/proto/sdk` 版本
  - 按当前上游解析结果同步 `golang.org/x/crypto`、`golang.org/x/net`、`golang.org/x/text` 和 `quic-go` 等间接依赖
- `nodemobile/go.mod/go.sum`
  - 对齐同一组 `core/proto/sdk` 版本
  - 同步当前依赖图对应的间接依赖版本

### MyFlowHub-Android/hubmobile
- `hubmobile/go.mod/go.sum`
  - `github.com/yttydcs/myflowhub-server`: `v0.0.7 -> v0.0.13`
  - `github.com/yttydcs/myflowhub-core`: `v0.4.0 -> v0.4.9`
  - `github.com/yttydcs/myflowhub-proto`: `v0.1.1 -> v0.1.5`
  - `github.com/yttydcs/myflowhub-sdk`: `v0.1.4 -> v0.1.12`
  - `github.com/yttydcs/myflowhub-subproto/auth`: `v0.1.2 -> v0.1.4`
  - `github.com/yttydcs/myflowhub-subproto/broker`: `v0.1.0 -> v0.1.1`
  - `github.com/yttydcs/myflowhub-subproto/exec`: `v0.1.0 -> v0.1.2`
  - `github.com/yttydcs/myflowhub-subproto/file`: `v0.1.2 -> v0.1.4`
  - `github.com/yttydcs/myflowhub-subproto/flow`: `v0.1.0 -> v0.1.2`
  - `github.com/yttydcs/myflowhub-subproto/forward`: `v0.1.0 -> v0.1.1`
  - `github.com/yttydcs/myflowhub-subproto/management`: `v0.1.2 -> v0.1.4`
  - `github.com/yttydcs/myflowhub-subproto/topicbus`: `v0.1.0 -> v0.1.2`
  - `github.com/yttydcs/myflowhub-subproto/varstore`: `v0.1.2 -> v0.1.4`
- 保留既有：
  - `replace github.com/yttydcs/myflowhub-server => ../../MyFlowHub-Server`
  - `replace github.com/yttydcs/myflowhub-sdk => ../../MyFlowHub-SDK`

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
- [wails-embed-dist-placeholder.md](../lessons/wails-embed-dist-placeholder.md)

## 对应 plan.md 任务映射
- `SDKDEP1` - 升级 SDK 上游依赖
- `SRVDEP1` - 升级 Server 上游依赖
- `WINDEP1` - 升级 Win 上游依赖
- `METDEP1` - 升级 MetricsNode 多模块上游依赖
- `ANDDEP1` - 升级 Android hubmobile 上游依赖
- `VAL1` - 执行跨仓 `GOWORK=off` 验证
- `REVIEW1` - 执行阶段 3.3 Code Review
- `DOC1` - 归档 workspace 变更

## 经验 / 教训摘要
- 用户先补 tag，不代表消费方已经自动完成版本收口；下游仍需逐仓升级 `go.mod/go.sum`。
- Android 即使保留 `replace`，也应该同步提升 `require` 的 semver 基线，否则本地和远端发布语义会继续漂移。
- `MetricsNode/windows` 这类带 `go:embed all:frontend/dist` 的模块，直接 `go test` 之前要先满足嵌入目录存在的前提；这不是依赖升级回归，而是既有构建前置条件。

## 可复用排查线索
- 症状
  - `go list -m` 仍解析到旧版本或 pseudo-version
  - Android `go mod tidy` 在 worktree 下找不到 `../../MyFlowHub-Server` / `../../MyFlowHub-SDK`
  - `pattern all:frontend/dist: no matching files found`
- 触发条件
  - 已发布新 tag，但下游仓库未同步 bump 依赖
  - Android `replace` 在 worktree 目录结构下解析不到目标 sibling path
  - Wails/Go module 直接运行测试时缺少 `frontend/dist`
- 关键词
  - `GOWORK=off`
  - `go list -m`
  - `replace ../../MyFlowHub-Server`
  - `replace ../../MyFlowHub-SDK`
  - `frontend/dist`
- 快速检查
  - 先对目标仓库执行 `GOWORK=off go list -m github.com/yttydcs/myflowhub-*`
  - 再执行 `GOWORK=off go test ./... -count=1 -p 1`
  - 若 Android `replace` 因 worktree 路径失效，可临时提供本地 junction 后再验证
  - 若 `windows` 模块报 embed 缺目录，先提供临时 `frontend/dist/placeholder.txt` 再测试

## 关键设计决策与权衡
- 采用“只升级依赖，不改业务代码”的最小方案：
  - 保证本轮只是版本收口，不引入计划外行为变化。
- Android 继续保留 `replace` 策略：
  - 这是当前仓库的既有开发/CI 设计，不在本轮顺手改成纯 semver，避免把问题扩大成发布链重构。
- `MetricsNode/windows` 的验证使用临时占位文件满足 embed 前置条件：
  - 只作为验证手段，不提交生成物或占位目录，避免把工作区临时文件带入仓库历史。

## 测试与验证方式 / 结果
- `MyFlowHub-SDK`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-core`
    - 结果：`v0.4.9`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-proto`
    - 结果：`v0.1.5`
  - `GOWORK=off go test ./... -count=1 -p 1`
    - 结果：通过
- `MyFlowHub-Server`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-core`
    - 结果：`v0.4.9`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-proto`
    - 结果：`v0.1.5`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-subproto/forward`
    - 结果：`v0.1.1`
  - `GOWORK=off go test ./... -count=1 -p 1`
    - 结果：通过
- `MyFlowHub-Win`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-core`
    - 结果：`v0.4.9`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-proto`
    - 结果：`v0.1.5`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-sdk`
    - 结果：`v0.1.12`
  - `GOWORK=off go test ./... -count=1 -p 1`
    - 结果：通过
- `MyFlowHub-MetricsNode`
  - 根模块 `GOWORK=off go list -m github.com/yttydcs/myflowhub-core`
    - 结果：`v0.4.9`
  - 根模块 `GOWORK=off go list -m github.com/yttydcs/myflowhub-proto`
    - 结果：`v0.1.5`
  - 根模块 `GOWORK=off go list -m github.com/yttydcs/myflowhub-sdk`
    - 结果：`v0.1.12`
  - 根模块 `GOWORK=off go test ./... -count=1 -p 1`
    - 结果：通过
  - `nodemobile` `GOWORK=off go test ./... -count=1 -p 1`
    - 结果：通过
  - `windows` `GOWORK=off go test ./... -count=1 -p 1`
    - 初次结果：失败，原因是 `frontend/dist` 缺失
    - 追加处理：临时创建 `frontend/dist/placeholder.txt`
    - 最终结果：通过
- `MyFlowHub-Android/hubmobile`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-server`
    - 结果：`v0.0.13 => ../../MyFlowHub-Server`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-core`
    - 结果：`v0.4.9`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-proto`
    - 结果：`v0.1.5`
  - `GOWORK=off go list -m github.com/yttydcs/myflowhub-sdk`
    - 结果：`v0.1.12 => ../../MyFlowHub-SDK`
  - `GOWORK=off go test ./... -count=1 -p 1`
    - 结果：通过

## 潜在影响
- `MetricsNode` 和 `Android` 由于上游升级，额外带入了当前依赖图的间接模块版本变化；虽然测试已通过，但后续若再 bump 上游，仍应重复做真实解析验证。
- Android 仍采用 `replace`，所以实际本地/CI 编译会消费 sibling Server/SDK 路径；本轮只是把 semver 基线和当前实现对齐。

## 回滚方案
- 对各消费方仓库：
  - 直接回退对应 worktree 分支上的依赖升级提交
  - 或把 `go.mod` 版本恢复到升级前并重新执行 `go mod tidy`
- 对已存在的远端 tag：
  - 不删除、不改写
  - 若发现问题，通过更高 patch 版本修复

## 子Agent执行轨迹
- 无。全部由主 agent 完成。
