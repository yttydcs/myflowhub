# 带深度的拓扑查询与 Desktop 按展开加载

## 状态与来源

- 日期：2026-09-06。
- 状态：产品行为已确认，$m-plan 技术方案待 Task ID 批准；尚未实施。
- 来源：当前任务「全面审视项目现状 (2)」的连续讨论和用户显式 $m-plan。
- 参考任务：local / 01a06ff5-b616-7f73-a66b-4141c07cb509，「全面审视项目现状」；已读取其 Flow 退役和模型讨论。
- 用户最新要求：“这个暂时移除了部分代码，减少演进的负担，然后请在新的基础上规划”。
- 实际代码基础：master @ ab19d3913fc4be6789c4eb3013840cc3f61f709f；独立规划工作树已同步。

## 原始诉求保真摘要

用户希望建立通用平台，使不同实现的节点能通过网络和资源协作。剪贴板、通知、自动化只是平台应用。用户反复明确网络只认识节点、网络和资源；设备、进程、插件都不是新的网络对象。

本轮聚焦模型与 Desktop，暂不讨论 Flow。用户确认节点树是导航，访问需要权限；从明确起点枚举已知节点关系，而非猜测 ID 或探测未知节点。

拓扑查询最初讨论整树与指定子树，随后用户提出增加深度参数，Desktop 点击展开再获取孩子，并关心父节点能否提前维护整棵子树、快速返回。最终明确 depth 必须使用纯数字：0 返回完整子树，1 返回直接孩子，n 返回向下最多 n 层；默认 1。

## 已确认行为

1. 查询起点默认直接父节点，也可明确指定其他节点。
2. depth 0 不改变查询起点、不扩大权限，不等于自动查全网。
3. 网络根 + depth 0 才是对应网络整树查询，仍需指定授权。
4. 父节点维护自己和后代关系；查询复用状态/快照，避免请求时逐节点采集。
5. Desktop 维护已加载部分树，展开时按需请求、折叠可复用缓存。
6. 树查询、catalog 读取、资源操作分别鉴权；已保存 View 可按 Resource reference 独立访问。
7. 未加载、真实空结果和失败不能混同；支持局部重试、失效、Profile/连接切换隔离。
8. 用户要求方案变化先给当前/准备方案对比，不直接改代码。

## 新基线约束

- Flow 已移除，重新设计保持待办。
- Android、C/ESP32/MicroPython 与移动专属 bindings/构建已退役，不再是本轮同步演进对象。
- Clipboard 已迁到 NodeHost。SDK/bindings 只操作已有节点，生命周期属于 NodeHost，EnrollmentBootstrap 独立。
- Windows Desktop、Metrics、Clipboard、通用网络与资源机制保留；Legacy Join 仍在当前支持范围。
- 不恢复 owning SDK API，不机械恢复旧平台目录或历史门禁。

## 规划选择与待审批范围

具体接口/capability、缓存与 UI 状态设计见 [计划快照](../plan/plan_archive_2026-09-06_depth-scoped-topology.md)。拟在同一个 system/topology Resource 上增加可分别授权的 children/subtree capability，对外 SDK 保持统一数值 depth 方法；不重构通用权限 evaluator。

按需加载使本地搜索只覆盖已加载节点。计划明确显示该范围，完整服务器搜索另立后续；不把这一限制隐藏在“全树搜索”文案后。

这一技术方案与 Task IDs 仍需批准。当前未修改运行时代码、业务代码或 tests。

上述为规划阶段记录。随后用户明确批准 DOC01、PROTO01、TREE01、MGMT01、SDK01、DESK01、QA01 并调用 $m-execute；七项实施与验收已完成，稳定文档已同步，最新状态以根级 todo 为准。未执行提交、合入或发布。

## 文档关系

- [执行验收快照](../plan/todo_archive_2026-09-06_depth-scoped-topology.md)
- [Desktop 需求](../requirements/desktop-resource-workspace.md)
- [当前 Desktop](../features/desktop.md)
- [Resource/capability 契约](../specs/resource-platform-v2.md)
- [权限契约](../specs/scoped-policy-authorization.md)
- [平台退役原始请求](2026-09-06_android-embedded-retirement-and-runtime-convergence.md)
- [平台与 SDK 退役决策](../decisions/2026-09-06_retire-mobile-embedded-and-owning-bindings.md)
- [Android/Embedded 待办](../requirements/mobile-embedded-redesign.md)
- [Flow 待办](../requirements/flow-redesign.md)

稳定 requirement/spec/ADR 的落地与索引更新在本计划 DOC01 中追踪，不把本 intake 当作已实现 feature 文档。

## 归档请求

用户随后显式调用 `$m-archive`，授权本地归档、提交、合入和本轮工作树清理。记录见[变更归档](../change/2026-09-06_depth-scoped-topology-discovery.md)，原执行批准及当时未提交状态保留为历史。
