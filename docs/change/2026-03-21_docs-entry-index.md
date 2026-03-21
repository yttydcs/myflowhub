# 2026-03-21 文档入口收敛

## 变更背景 / 目标

在压缩根级 `plan.md`、为 `docs/change` 与 `docs/plan_archive` 建索引之后，工作区文档入口仍然偏分散：

- 新接手者仍需要先知道去哪里找总入口；
- `plan.md`、`repos.md`、`docs/` 目录之间的入口关系尚未完全显式化；
- 历史文档仍会引用 `target.md`，但当前工作区里该文件已不存在，容易误导。

本次目标是把文档入口再收敛一层，形成“先看哪里、按什么问题进入”的明确路径。

## 具体变更内容

### 新增
- `docs/README.md`
  - 作为 `docs/` 目录总入口；
  - 串联 `docs/change/README.md`、`docs/plan_archive/README.md`、`docs/protocol_map.md`、根级 `plan.md`、`repos.md`、`guide.md`；
  - 明确说明 `target.md` 当前缺失，仅保留历史引用语义。

### 修改
- `plan.md`
  - 在 Documentation Map 中加入 `docs/README.md`；
  - 将 `target.md` 改为“当前缺失、仅历史引用”；
  - 在 Historical Entry Points 中加入文档总入口。
- `repos.md`
  - 在工作区结构说明中加入 `docs/README.md`、`docs/plan_archive/`、`docs/protocol_map.md`；
  - 将 `target.md` 改为“当前缺失、按历史上下文理解”。
- `docs/change/README.md`
  - 将“完整执行过程”入口从目录路径改为显式索引 `../plan_archive/README.md`；
  - 增补本次变更文档入口。

## 设计决策与权衡

- 采用“总入口页 + 专题索引页”的两层结构，而不是继续把所有说明堆回根级 `plan.md`。
  - 优点：入口清晰，职责更稳定，后续增长时不容易再次失控。
  - 代价：多一个入口文件，但层级仍然简单。
- 对 `target.md` 不做占位重建。
  - 原因：当前没有足够上下文证明应恢复哪个版本；直接声明“当前缺失”比伪造入口更安全。

## 测试与验证方式 / 结果

- 验证 `docs/README.md` 已创建并可作为总入口阅读。
- 验证 `plan.md` 与 `repos.md` 已显式指向新的文档入口。
- 验证 `docs/change/README.md` 仍保持有效索引，并改为指向 `docs/plan_archive/README.md`。
- 结果：通过。

## 潜在影响

- 历史文档中仍会出现 `target.md` 的旧引用；本次不改历史正文，只在新入口中明确说明其状态。

## 回滚方案

- 回滚 `docs/README.md`、`plan.md`、`repos.md`、`docs/change/README.md` 的本次改动即可恢复到入口收敛前状态。
