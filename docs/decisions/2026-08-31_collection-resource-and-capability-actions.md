# 2026-08-31 使用 Collection Resource 管理成员，以 Capability 表达普通操作

## Status

Accepted for implementation。

## Context

当前 Catalog 同时出现 Variable/Stream/File 等业务对象和 `flow/create`、`flow/cancel`、
`system/admission/issue` 等 API 端点。Desktop 因而把业务资源、展示 namespace 和命令端点混在同一棵树中；
文件目录、Flow definitions/runs 等动态对象集合又缺少统一、可分页且不膨胀 catalog 的表达。

## Options considered

1. 每个 member 和每个操作都注册为 Resource。
   - 授权直观，但动态集合会放大 catalog、订阅和 UI，操作端点继续污染资源树。
2. 把所有能力放进一个全局 Command/Service Resource。
   - 入口数量少，但丢失业务 owner、schema、能力边界和可组合 renderer。
3. Resource 保持全局治理单位，普通操作成为其 Capability；Collection Resource 管理 scoped members。
   - 保留统一 Resource/policy/wire，同时避免把内部成员和普通操作提升为全局身份。
4. 继续沿用现状，只在 Desktop 视觉上把命令折叠起来。
   - 不解决协议、权限、SDK 和 provider 的根本混合。

## Decision

- 采用选项 3。
- Collection 是 Resource contract，不是第二棵树；member 默认只有 collection-scoped identity。
- 对已有 Resource 的操作默认是 Capability。独立 Command Resource 只用于没有自然目标且自身需要独立
  schema、权限、生命周期和审计的执行边界。
- 第一阶段增加通用 `mfh.collection`、标准 member list/page contract 和可复用多 Capability handler Resource。
- Flow 首先迁移为 `flow/definitions` 与 `flow/runs` Collections，旧的普通操作 Resources clean-break 移除；
  其他产品命令分组另行迁移。
- filesystem provider 以每个配置 root 注册一个 Resource；第一阶段默认只读，写、删和大文件下载留给独立
  安全/session 设计。第一阶段精确能力为 `list/get/read`，不提供 virtual multi-root mount。
- Desktop 从 descriptor 生成操作入口；Context Menu、Inspector 与 Widget 共享 operation controller。
  Renderer 只使用 type/schema/capability/content type，不根据 Resource name 猜内容。

## Consequences

- 现有 generic operation envelope 与 exact `subject + resource + capability` policy 可以继续使用。
- 需要新增 Collection schema、multi-capability Resource helper、typed SDK/bindings 和生成契约。
- Flow 的 Resource IDs、capability grants、SDK 调用和 Desktop fixtures 会发生 clean-break；旧 grant 不自动放大为
  新权限，管理员必须显式重授新 Resource/capability。
- UI 只能把 descriptor capability 当作“支持的操作”。在通用 effective-capability discovery 落地前，
  authority 仍可能返回 Forbidden；客户端必须明确显示，不能把按钮可见性当作权限事实。
- `AUTHZ02` member selector policy 与 filtered/effective discovery、`CMD02` 其他旧命令迁移、`FS02` 文件写删/
  virtual multi-root/大文件 data lane 保持显式后续边界。

## Confidence

High。当前 wire、Registry 和 SDK 已以 Resource + Capability 为核心，主要新增面是通用 handler、集合 member
contract、第一方迁移和 Desktop 交互，而不是恢复 SubProto 或第二套权限系统。

## Supersedes / superseded by

- 细化 [可扩展 Resource type system 与 Desktop workspace](2026-08-28_extensible-resource-type-system-and-desktop-workspace.md)
  中“不要把所有行为退化为 Command”的边界。
- 不取代唯一 Node tree、Resource ownership、NodeHost 或 provider-owned schema / Desktop-owned renderer 决策。

## Related docs

- [Discussion intake](../intake/2026-08-31_resource-collections-and-actions.md)
- [Resource Collections and Actions](../specs/resource-collections-and-actions.md)
- [Resource Platform v2](../specs/resource-platform-v2.md)
- [Flow vNext](../specs/flow-vnext.md)
- [Desktop Schema Rendering](../specs/desktop-schema-rendering.md)
