# Resource Collections and Actions

> 归档说明：这是 MAIN02 中主检出原有的实施前 accepted-design 草案。当前已实现合同以
> [Resource Collections and Actions spec](../specs/resource-collections-and-actions.md) 为准；保留本快照是为了不丢失
> `watch/delete`、effective capability 与 shared-lifecycle named mounts 等后续设计意图。

## Status

Accepted design，尚未实现。本文扩展 [Resource Platform v2](../specs/resource-platform-v2.md) 的资源表达，
固定 Collection、Resource Capability、独立 Command Resource 与 Desktop 交互的目标边界；当前 wire、
catalog 和 policy 仍以已实现的 v2 contract 为准，具体 schema 与迁移由后续计划定义。

## Core concepts

- Resource 是网络中的独立寻址、发现、授权和审计边界，以 `(owner NodeID, local name)` 标识。
- Capability 是调用方可以对 Resource 执行的操作。对已有 Resource 的普通操作默认不是新的 Resource。
- Collection 是一种 Resource contract：Resource 自身保持一个全局身份，同时管理一组可枚举、可定位的
  内部 member。
- Collection member 默认只有 collection-scoped identity，不自动进入 Node catalog，也不自动成为全局
  Resource。
- Resource name 中的 `/` 只允许 Desktop 派生 presentation namespace；纯 namespace 不是 Resource、
  Collection、permission scope 或 routing hop。

Collection 不等同于磁盘目录。它可以由本地目录、对象存储、数据库记录、Flow definition、运行实例、
设备集合或动态生成内容提供。provider 的物理路径、表名、bucket 和其他部署细节不得成为网络 ResourceID。

## Action ownership

对现有对象的行为声明为该 Resource 的 Capability。例如取消运行是 `flow/runs` 的 `cancel` capability，
不再默认建模为 `flow/cancel` Command Resource。

只有当一个操作自身形成稳定、独立的寻址、schema、权限、生命周期和审计边界，并且没有自然目标 Resource
时，才建模为独立 Command Resource。多个紧密相关的系统操作可以由一个 Service Resource 提供多个
capabilities，例如 `system/admission` 的 `issue` 与 `revoke`。

任意操作系统命令执行不得因为属于 Command Resource 就获得隐式安全性。不得默认暴露接受任意命令字符串的
远程 shell；预定义任务或受控执行服务必须限制 executable、arguments、working directory、environment、
identity、deadline、output size 和 cancellation，并继续经过 Resource policy 与 owner validation。

## Collection contract

Collection descriptor 必须显式声明其支持的 capabilities 和对应 schema，不因类型名自动获得操作。常见
capabilities 包括 `list`、`get`、`create`、`update`、`delete`、`watch` 和领域操作；实际集合只声明其真实
支持的子集。

每次 member operation 必须携带由 provider schema 约束的 member locator。locator 可以是 path、stable ID、
object key 或组合键，但必须有界、可验证且只在当前 Collection 内解释。对文件系统 provider，owner 必须在
解析 traversal、symlink/junction 和大小写语义后验证目标仍位于配置 root 内。

授权分为两层：

1. 网络 authority 裁决 subject 是否可以对 Collection Resource 执行该 capability；
2. Collection owner 根据同一 authority context 和 provider policy 裁决该 member locator 是否在允许范围内。

UI namespace、presentation hint 和物理存储位置都不得扩大权限。Collection member 只有在需要独立全局授权、
长期跨系统引用、独立订阅、独立生命周期或独立 owner 时，才提升为普通 Resource 并进入 catalog。

## Filesystem example

三个需要分别授权的本机目录应注册为三个 Collection Resources，而不是为每个文件注册 Resource：

```yaml
resources:
  - name: storage/a
    provider: filesystem
    root: D:\data\a
  - name: storage/b
    provider: filesystem
    root: E:\shared\b
  - name: storage/c
    provider: filesystem
    root: C:\workspace\c
```

网络只发现 `storage/a`、`storage/b`、`storage/c`；物理 root 保留在 owner 本地配置。若三个目录总是共享同一
权限和生命周期，provider 可以把它们作为一个 Collection Resource 的命名 mounts；若需要分别授权，默认
保持三个 Resources。

## Flow example

Flow 默认提供两个 Collection Resources：

- `flow/definitions` 管理可执行定义，支持按声明提供 `list/get/create/update/delete/watch/run`；
- `flow/runs` 管理执行实例，支持按声明提供 `list/get/subscribe/cancel/archive/delete`。

调用 `flow/definitions.run(definitionID, input)` 产生 `flow/runs` 中的 member。Definition 与 Run 默认不逐项
进入全局 catalog；只有满足独立 Resource 提升条件时才注册为普通 Resource。

## Desktop interaction

- Resource Explorer 只在真实 Resource 上提供上下文菜单；纯 presentation namespace 只负责展开和折叠。
- 上下文菜单根据 descriptor 与当前有效权限提供快捷操作，并提供“添加到当前 View”等可访问的键盘等价路径。
- Resource 加入 View 后，由兼容 renderer 决定主要内容和常用操作按钮。按钮只是 capability 的入口，不能
  授权、隐藏 owner validation 或把 presentation hint 变成行为事实。
- Renderer 按 type、schema、capability、content type 和 pane 条件匹配，不按 Resource name 猜测。例如
  filesystem Collection 的文本 member 可以使用文本/代码 renderer，图片可以使用图片 renderer，未知格式
  回退到 metadata、下载或 raw inspector。
- 同一 capability 无论从上下文菜单、Inspector 还是 View Widget 触发，都必须经过同一 SDK 调用、policy、
  schema validation、deadline、错误与审计路径。

## Deferred protocol details

后续实施计划必须单独确定 member locator schema、catalog 对 Collection 的描述字段、member selector policy、
filtered discovery/effective capabilities、分页与 continuation、watch event、session/data lane 复用以及旧 Command
Resources 的兼容或迁移策略。本文不把这些未实现细节宣称为当前 API。

## Related docs

- [Resource Platform v2](../specs/resource-platform-v2.md)
- [Flow vNext](../specs/flow-vnext.md)
- [Desktop Resource Workspace v3](../specs/desktop-resource-workspace-v3.md)
- [Desktop Schema Rendering](../specs/desktop-schema-rendering.md)
- [可扩展资源平台 requirement](../requirements/extensible-resource-platform.md)
- [Desktop 资源工作区 requirement](../requirements/desktop-resource-workspace.md)
