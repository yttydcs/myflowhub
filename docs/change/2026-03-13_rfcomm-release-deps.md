# 2026-03-13 - RFCOMM 发布后下游依赖对齐与发版

## 背景 / 目标
- Core 已发布 `v0.3.1`，包含 RFCOMM transport。
- Server / SDK / Android 主分支虽然已接入 RFCOMM，但依赖版本尚未全部对齐到可拉取 tag。
- 本 workflow 目标是完成下游依赖升级、验证、合并、push 与 tag。

## 具体结果
- `MyFlowHub-Server`
  - `myflowhub-core` 升级到 `v0.3.1`
  - 已合并到 `main`
  - 已发布 tag：`v0.0.6`
- `MyFlowHub-SDK`
  - `myflowhub-core` 升级到 `v0.3.1`
  - 已合并到 `main`
  - 已发布 tag：`v0.1.3`
- `MyFlowHub-Android`
  - `hubmobile` 对齐 `core v0.3.1`、`sdk v0.1.3`、`server v0.0.6`
  - 保留 `replace github.com/yttydcs/myflowhub-server => ../../MyFlowHub-Server`
  - 已合并到 `main`
  - 已发布 tag：`v0.1.22`

## Code Review
- 需求覆盖：通过（下游三仓均完成版本对齐与发版）
- 架构合理性：通过（只做版本收口，不改 RFCOMM 逻辑）
- 性能风险：通过（无运行时逻辑变更）
- 可读性与一致性：通过（归档文档齐全，版本策略一致）
- 可扩展性与配置化：通过（Android 仍保留既有 replace/CI 策略）
- 稳定性与安全：通过（全部以 `GOWORK=off` 验证可拉取版本）
- 测试覆盖：通过（三仓测试通过）

## 测试与验证
- `repo/MyFlowHub-Server`：`GOWORK=off go test ./... -count=1`
- `repo/MyFlowHub-SDK`：`GOWORK=off go test ./... -count=1`
- `repo/MyFlowHub-Android/hubmobile`：`GOWORK=off go test ./... -count=1`

## 风险与回滚
- 若发现发布版本有问题，不回写已发布 tag，改发更高 patch 版本修正。
- Android 仍依赖本地 `replace` 进行 meta-workspace / CI 对齐；本次未改变该策略。
