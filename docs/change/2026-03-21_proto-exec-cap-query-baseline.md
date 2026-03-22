# 2026-03-21 Proto：exec cap_query 基线确认

## 变更背景 / 目标
- 背景：`MyFlowHub-Win` 已接入 Flow 方法能力查询，Win 构建依赖 `protocol/exec` 中的 `CapQueryReq/Resp` 与 `ActionCapQuery*`。
- 问题：已发布 tag `v0.1.1` 不包含这组协议定义，而 Proto 当前仓库已存在对应实现。
- 目标：确认一个可被 Win 审计引用的 Proto 基线 commit，并记录其包含的协议定义。

## 具体变更内容（新增 / 修改 / 删除）
### 新增
- 本文档
  - 记录本次基线确认结果。

### 修改
- 无代码修改；协议定义已存在于当前仓库。

### 删除
- 无。

## 对应 plan.md 任务映射
- `PROTO-BASELINE`
  - 完成 exec `cap_query` 协议基线确认与归档。

## 关键设计决策与权衡
- 决策：以 commit `7eef50dcc471db88d00cb15d9a5b5f3acc0fe1ad` 作为 Win 对齐的 Proto 基线。
  - 原因：该 commit 已包含 `ActionCapQuery`、`ActionCapQueryResp`、`CapQueryReq`、`CapQueryResp`，且可由 Go module 解析为 pseudo-version `v0.1.2-0.20260318063708-7eef50dcc471`。
  - 权衡：当前仍未形成正式 tag，因此先作为过渡基线；后续建议发布正式版本以收敛依赖策略。

## 测试与验证方式 / 结果
- 检查文件：`repo/MyFlowHub-Proto/protocol/exec/types.go`
  - 结果：已包含 `ActionCapQuery`、`ActionCapQueryResp`、`CapQueryReq`、`CapQueryResp`。
- 执行：`go list -m -json github.com/yttydcs/myflowhub-proto@7eef50d`
  - 结果：解析为 `v0.1.2-0.20260318063708-7eef50dcc471`。

## 3.3 Code Review 结论
- 需求覆盖：通过
  - 已确认 Win 所需协议定义在指定 Proto 基线中存在。
- 架构合理性：通过
  - 未修改协议 wire，仅为跨仓依赖选择提供明确基线。
- 性能风险：通过
  - 文档归档，无运行时影响。
- 可读性与一致性：通过
  - 基线、commit、版本号与协议项一一对应。
- 可扩展性与配置化：通过
  - 后续正式发版后可直接从 pseudo-version 平滑切换到 tag。
- 稳定性与安全：通过
  - 无代码路径变化。
- 测试覆盖情况：通过
  - 已完成协议文件核对与 module 解析验证。
- 子Agent治理与审计：通过
  - 未使用子Agent。

## 潜在影响与回滚方案
- 潜在影响：
  - 若后续 Proto 正式 tag 内容与该基线不同，Win 仍需再次对齐。
- 回滚方案：
  1. 删除本次文档归档。
  2. 不影响任何协议代码行为。

## 子Agent执行轨迹
- 本轮未使用子Agent。
