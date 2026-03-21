# 2026-03-21 repos 接手指南压缩

## 变更背景 / 目标

根级 `repos.md` 之前同时承载仓库职责说明、重构历史和已完成路线图，信息量过大，和 `docs/change`、`docs/plan_archive` 中的归档产生重复。

本次调整目标：

- 将 `repos.md` 压缩为“当前有效边界 + 接手建议”；
- 已完成的详细里程碑继续以 `docs/change/README.md` 与 `docs/plan_archive/README.md` 为主入口；
- 补齐根级接手文档中遗漏的仓库职责，特别是 `MyFlowHub-Android` 与 `MyFlowHub-MetricsNode`。

## 具体变更内容

### 修改

- 重写 `repos.md`
  - 删除已完成 PR/里程碑的大段正文；
  - 保留工作区含义、依赖方向、仓库职责、当前有效路线、常用注意点；
  - 新增 `MyFlowHub-Android` 与 `MyFlowHub-MetricsNode` 的职责说明。
- 更新 `docs/change/README.md`
  - 增加本次文档压缩的归档入口。
- 更新 `docs/README.md`
  - 将 `repos.md` 的定位描述从“重构路线图”调整为“边界与接手说明”。

## 任务映射

- 根级文档整理主线：
  - `repos.md` 只保留当前有效指导信息；
  - 已完成历史继续由 `docs/change` 与 `docs/plan_archive` 承接。

## 关键设计决策与权衡

1. `repos.md` 不再做历史流水账。
   - 原因：这类内容已经在归档目录中存在，继续堆在根级文档只会提高维护成本。
2. `repos.md` 保留“边界判断”而非“详细过程”。
   - 接手者真正需要的是“改动应该落在哪个仓、不能越过哪些边界、下一步是否仍有效”。
3. Android 与 MetricsNode 单独补齐职责。
   - 原因：它们已经成为稳定仓库角色，继续缺席根级指南会导致接手者误判代码归属。

## 测试与验证方式 / 结果

- 人工核对：
  - `repos.md` 仍保留当前有效的仓库职责、依赖边界、Deferred 项与常用命令；
  - 已完成里程碑不再在根级正文展开。
- 索引核对：
  - `docs/change/README.md` 已加入本文件入口；
  - `docs/README.md` 对 `repos.md` 的描述已与新定位一致。

## 潜在影响与回滚方案

### 潜在影响

- 接手者不能再从根级 `repos.md` 直接看到完整历史路线，需要通过 `docs/change/README.md` 和 `docs/plan_archive/README.md` 进入。
- 这是有意调整，目的是让根级文档维持短入口角色。

### 回滚

- 若后续确认根级 `repos.md` 仍需恢复更长的历史正文，可基于 git 历史或本次变更前版本回滚该文件；
- 但更推荐继续保持根级精简，仅在 `docs/` 中扩充归档。
