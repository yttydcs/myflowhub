# Win：升级 Core / SDK 以携带 RFCOMM 写出修复

## 变更背景 / 目标
- 背景：
  - Win 客户端的 RFCOMM 连接已经可以建立，但真实注册链路会卡在发包后无回包；
  - 根因修复已分别在 `MyFlowHub-Core v0.4.3` 与 `MyFlowHub-SDK v0.1.5` 落地；
  - Win 需要升级依赖并重新发布，才能让 release 包获得完整修复。
- 目标：
  - 将 Win 依赖对齐到新 Core / SDK 版本；
  - 通过最小必要测试验证；
  - 准备新的 Win release。

## 具体变更内容
- 修改：
  - `go.mod`
    - `github.com/yttydcs/myflowhub-core` 升级到 `v0.4.3`
    - `github.com/yttydcs/myflowhub-sdk` 升级到 `v0.1.5`
  - `go.sum`
    - 更新依赖校验

## 对应 plan.md 任务映射
- `WIN-RFCOMM-1`：完成 Core / SDK 依赖升级
- `WIN-RFCOMM-2`：完成最小验证
- `WIN-RFCOMM-3`：已执行代码审查
- `WIN-RFCOMM-4`：本文档

## 关键设计决策与权衡
- 本仓只做依赖升级，不额外改 UI / auth / session 业务逻辑；
- 这样能把修复集中在 Core / SDK 两层，Win 只负责消费稳定版本，维护成本最低；
- 发布风险也更可控，回滚只需回退依赖版本。

## 测试与验证方式 / 结果
- 已执行：
  - `GOWORK=off go test ./... -count=1`
- 结果：
  - 通过

## 潜在影响与回滚方案
- 潜在影响：
  - Win 的 TCP 行为不应变化；
  - RFCOMM 注册 / 登录链路将随底层依赖修复一起稳定。
- 回滚方案：
  - 回退以下文件：
    - `go.mod`
    - `go.sum`
