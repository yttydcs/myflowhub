# 2026-03-25 Proto：发布 flow delete 协议契约

## 变更背景 / 目标
- 背景：
  - `MyFlowHub-SubProto/flow` 当前实现已经依赖 `flow delete` 协议契约。
  - 已发布的 `myflowhub-proto v0.1.2` 仍缺失 `ActionDelete`、`DeleteReq`、`DeleteResp` 等符号。
  - 下游 `MyFlowHub-Server` 在 `GOWORK=off` + 真实 semver 解析下会因此编译失败。
- 目标：
  - 将当前主线已存在的 `flow delete` 契约正式发布为 `myflowhub-proto v0.1.3`；
  - 为 `SubProto/flow` 和 `Server/defaultset` 的发布链补齐上游基础。

## 具体变更内容
- 发布范围：
  - `protocol/flow/types.go`
- 对外契约：
  - `ActionDelete`
  - `ActionDeleteResp`
  - `PermFlowDelete`
  - `DeleteReq`
  - `DeleteResp`
- 本次不新增协议设计，只对当前主线已存在符号做 patch release。

## Requirements impact
- `none`

## Specs impact
- `none`

## Lessons impact
- `updated`

## Related requirements
- `none`

## Related specs
- `none`

## Related lessons
- `D:\project\MyFlowHub3\docs\lessons\cross-repo-semver-release.md`

## 对应 plan.md 任务映射
- `PROTOREL1`：确认 `flow delete` 契约发布边界
- `PROTOREL2`：发布 `myflowhub-proto v0.1.3`

## 经验 / 教训摘要
- 即使协议代码已在上游主线存在，只要 patch tag 没跟上，下游仍会在真实依赖解析时得到旧契约。
- `SubProto` 新功能若引用新增协议常量 / schema，必须先检查 `Proto` 当前已发布版本是否真的包含这些符号。

## 可复用排查线索
- 症状
  - `undefined: protocol.ActionDelete`
  - `undefined: protocol.ActionDeleteResp`
  - `undefined: protocol.PermFlowDelete`
  - `undefined: protocol.DeleteReq`
  - `undefined: protocol.DeleteResp`
- 触发条件
  - `flow` module 或 `Server` 在 `GOWORK=off` 模式下解析到旧版 `myflowhub-proto`
- 关键词
  - `flow delete`
  - `ActionDelete`
  - `DeleteReq`
  - `DeleteResp`
  - `GOWORK=off`
- 快速检查
  - 检查 `go list -m github.com/yttydcs/myflowhub-proto`
  - 检查当前 proto tag 是否晚于 `flow delete` 契约引入提交

## 关键设计决策与权衡
- 采用 patch release，而不是回退 `flow` module 对 `delete` 契约的使用。
- 只发布已有契约，不在本次 workflow 里引入新的协议设计调整。

## 测试与验证方式 / 结果
- 符号检查：
  - 当前 worktree `protocol/flow/types.go` 已包含上述 `delete` 契约
- 下游验证：
  - 该 tag 供 `MyFlowHub-SubProto/flow` 与 `MyFlowHub-Server` 的后续 `GOWORK=off` 验证使用

## 潜在影响
- 下游若继续锁定 `myflowhub-proto v0.1.2`，仍会复现相同未定义符号错误。

## 回滚方案
- 若 tag 尚未推送：
  - 删除本地 tag
  - 回退本次 archive
- 若 tag 已推送：
  - 不重写已发布 tag，改发更高 patch 版本覆盖

## 子Agent执行轨迹
- 本轮未使用子 Agent。
