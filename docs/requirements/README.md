# Requirements

存放 MyFlowHub3 meta workspace 的长期需求、范围和验收口径。

## How To Use
- 先看根级 [../README.md](../README.md) 了解阅读顺序。
- 只有当某个需求是长期稳定真相时，才在这里新增叶子文档。

## What Belongs Here
- workspace 级需求边界
- 长期接手约束
- 可被反复引用的验收标准

## Current Status
- [unified-node-runtime.md](unified-node-runtime.md)
  - 统一权威节点树、可插拔链路、资源/订阅/指令内核和 monorepo 收敛的长期需求。
- [management-node-display-name.md](management-node-display-name.md)
  - 设备管理中的节点显示名需求、范围与验收口径。
- [auth-controlled-admission.md](auth-controlled-admission.md)
  - auth 普通注册审批、一次性角色 permit 与父链受控准入需求。

## Rules
- 使用稳定文件名，不使用日期前缀。
- 需求变更先落这里，再写对应的 `plan/change`。
