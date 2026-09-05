# 可扩展资源平台

## Background

当前统一节点运行时已经把设备、权限、路由和资源收敛到一棵 authoritative Node tree，但 Resource kind 固定为 Variable、Stream、Command。新的产品方向要求 Node 下的一切可发现能力都作为 Resource 表达，并允许 Topic、File、Media 等具有不同操作语义的类型进入，而不恢复独立 SubProto、权限树或路由树。

## Goal

建立可扩展、可发现、可授权、可订阅和可操作的 Resource type system。Core 保持稳定的寻址、权限、路由、订阅、操作和生命周期边界；具体类型在这些边界内扩展。

## Scope

- 所有设备和运行实体继续作为 Node 加入唯一 authoritative tree。
- 每个 Resource 由恰好一个 Node 拥有，并以 owner NodeID 与本地路径寻址。
- Resource descriptor 声明版本化 type、capabilities、schemas、permissions、limits 和 presentation hints。
- 第一阶段基础类型：Variable、Stream、Topic、Command。
- File 使用 session-oriented 扩展类型迁移现有分块传输。
- Collection 作为 Resource contract 表达文件目录、对象集合、Policy definitions/bindings 等动态成员集合；
  member 默认不逐项进入全局 catalog。
- Media 能被 descriptor/session contract 表达；生产级实时音视频数据面单独交付。
- 未知类型必须保持可发现，并能够通过通用 inspector 查看描述和执行被授权的通用操作。

## Scenarios

- 客户端发现一个 Variable，读取当前值、条件写入并订阅后续变化。
- 客户端发现一个 owner-originated Stream，订阅有序事件并显式处理 gap。
- 多个获授权 Node 向一个 Node-owned Topic 发布，多个订阅者接收事件。
- 客户端发现一个 Command，按照 descriptor schema 调用并接收结果。
- 客户端发现一个 Collection，按 provider 约束的 member locator 枚举和操作内部成员，而不需要把每个成员
  注册为全局 Resource。
- Policy 客户端通过 definitions/bindings Collections 管理策略定义与绑定，操作权限仍由 Authority 裁决。
- 文件发送方打开 File session，在独立有界数据 lane 上传内容，同时订阅进度。
- Desktop 遇到未来新增的 Resource type 时仍能展示 descriptor，而不需要升级 Core 才能看见资源。

## Functional Requirements

1. Resource descriptor 必须包含稳定 ResourceID、type ID、type version 和显式 capability 列表。
2. 每个 capability 必须声明输入/输出或事件 schema、大小/速率边界以及允许的交互模式；网络授权继续使用
   exact `subject + ResourceID + CapabilityID`，descriptor `Permission` 兼容字段不得形成第二个授权 key。
3. 客户端不能通过伪造 capability 或 permission 名称绕过 owner、route 或 policy 裁决。
4. Variable 必须保留当前值、revision、snapshot-first subscription 和条件写入语义。
5. Stream 必须保留 owner-originated sequence、bounded delivery 和 explicit gap 语义。
6. Topic 必须由明确 Node 拥有，支持多发布者/多订阅者，分别授权 publish/subscribe，默认无 retention/replay，只保证单 publisher 顺序。
7. Command 必须保留 deadline、dedupe、bounded input/output、panic isolation 和 correlated result。
8. Subscription 必须针对 descriptor 声明为 observable 的 capability 建立，并继续受 lease、link、topology epoch 和 policy generation 控制。
9. File/Media 等大数据类型必须通过 session control 与有界 data lane 承载，不得把无界内容伪装为普通 Variable 或 Stream event。
10. catalog 必须版本化、确定性排序，并能表达未知类型；catalog presentation hint 不构成权限或行为事实。
11. 删除或替换 type/capability 时，既有 subscription、pending operation 和 session 必须显式完成、取消或过期。
12. 新模型是 clean break，不保留旧固定 kind wire、SubProto 或 TopicBus compatibility bridge。
13. 对已有 Resource 的行为必须优先声明为该 Resource 的 capability；只有操作自身形成独立寻址、schema、
    权限、生命周期和审计边界时，才注册独立 Command Resource。
14. Collection 必须保持单一 Resource identity，并用有界、可验证的 member locator 管理内部成员；
    presentation namespace 和 provider 物理存储位置不得成为第二棵资源、路由或权限树。
15. Collection member operation 必须同时通过 Collection Resource capability 授权和 owner 的 member scope
    校验；member 只有在需要独立全局治理时才提升为 catalog Resource。
16. 第一阶段 Collection 通用能力只固定 `list/get`；member descriptor 不得携带 owner/ResourceID，必须使用
    bounded opaque key、deterministic order、sorted unique capability subset 和 bounded attributes。
17. 通用 member selector policy、filtered/effective capability discovery 不得由客户端或 provider 猜测，
    作为独立 `AUTHZ02` 设计。

## Non-functional Requirements

- 权限、心跳和 session control 不得被大体积数据永久队头阻塞。
- 所有队列、payload、session、retention、publisher 和 subscriber 数量必须有明确上限。
- 未知类型、未知 capability 和未知 major version 必须显式失败或降级为只读 descriptor，不得猜测行为。
- Memory、TCP、QUIC、RFCOMM 等 Transport 不得泄漏到 Resource type 实现。
- 类型注册不能制造 Core 到产品包的反向依赖或循环依赖。
- Go SDK、bindings、Desktop、Android 和 Embedded contract 必须由同一 descriptor/operation schema 生成或验证。

## Edge Cases

- 资源在 View 打开期间消失、换 owner、升级 major version 或被撤权。
- Topic publisher 重连后 sequence 重置、重复 event ID 或超过速率限制。
- 慢订阅者、跨低带宽链路和 topology reparent。
- session 打开后授权、父链或 policy generation 变化。
- File session 中断、重复 chunk、offset gap、checksum 错误和重启清理。
- 未安装 renderer 的自定义类型。

## Acceptance Criteria

- Variable、Stream、Topic、Command 在 memory 与 TCP 跨子树测试中通过发现、权限、操作和撤权门禁。
- Topic 至少覆盖两个 publisher、两个 subscriber、拒绝未授权 publish/subscribe、断线恢复与默认无 replay。
- 未知 Resource type 可以出现在 catalog 和 Desktop generic inspector 中，不能被错误调用。
- filesystem fixture 可以把多个本地 root 注册为可分别授权的 Collection Resources，拒绝越界 member locator；
  Policy Collections 验证领域成员操作，不要求每个普通操作成为独立 Resource。
- File 迁移后保持 offer/cancel/checksum/atomic completion，并证明大传输期间控制消息仍可及时处理。
- 固定三类型的 Core switch、旧 wire operation 和兼容桥从 canonical build 中移除。
- `go test ./...`、`go vet ./...`、生成契约和全产品门禁通过。

## Related Features

- [Desktop](../features/desktop.md)
- [File transfer](../features/file-transfer.md)

## Related Specs

- [Resource Platform v2](../specs/resource-platform-v2.md)
- [Resource Collections and Actions](../specs/resource-collections-and-actions.md)
- [节点树、链路与资源架构规范](../specs/node-tree-link-resource-architecture.md)

## Related Decisions

- [可扩展 Resource type system 与 Desktop workspace](../decisions/2026-08-28_extensible-resource-type-system-and-desktop-workspace.md)
- [Collection Resource 与 Capability Action](../decisions/2026-08-31_collection-resource-and-capability-actions.md)

## Related Intake

- [可扩展资源平台与 Desktop 工作区重构](../intake/2026-08-28_extensible-resources-and-desktop-workspace-redesign.md)

