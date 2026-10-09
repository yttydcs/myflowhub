# 2026-09-06 平台 Variable 契约与随快照传递的值定义

## Status

Implemented R2，2026-09-06。按用户“当前不需要任何兼容，产品还没发布”完成一次性替换；2026-10-09 完成归档，后续通用控件交互不改变平台契约和权限边界。执行证据和本地合入状态见 [变更记录](../change/2026-10-09_unified-variable-contract.md)，未对外发布。

## Context

改造前平台统一 Variable 的存储与操作，但值采用提供者私有 schema；Desktop 只认识生成的内置定义。Metrics 的字符串采样包装暴露了标准变量与通用消费之间的缺口。

## Options

| 方案 | 收益 | 代价与结论 |
| --- | --- | --- |
| 当前：统一操作、产品 payload、Desktop 内置 schema | 已有实现简单 | 新节点的变量仍可能不可理解 |
| 给 Desktop 增加 Metrics schema/专用 renderer | 快速改善这一个产品 | 保留产品耦合，不满足目标，不采用 |
| 只增加动态 schema，保留产品变量包装 | 可生成任意对象表单 | 仍不知道哪个字段是标准变量值，不足以解决问题 |
| 标准变量值 + Catalog 内嵌完整定义 | 发现时就有类型 | 目录膨胀、无权限读取时暴露更多数据语义，本轮不采用 |
| 标准变量值 + 独立 describe/schema 服务 | 可复用独立缓存 | 新 capability、权限、版本与失效面更大，本轮不采用 |
| 标准变量值 + read/订阅快照携带定义 | 同一资源权限、无需预装产品定义、后续事件不重复定义 | 新变量版本和提供者迁移，推荐 |

## Accepted decision

采用 mfh.variable 的新 type_version=2。平台定义版本化值 envelope；read 和订阅初始快照携带有界 definition，后续状态更新复用同一实例定义。值直接保持 JSON 类型，元数据与值分开。定义为本次注册实例的不可变数据，替换实例必须令旧订阅和旧写入失效。

Core 的路由、权限和通用操作不按 Metrics 或其他产品分支。Variable 包负责标准类型校验、状态与序列化；提供者声明实例语义并执行业务逻辑；SDK 处理平台契约；Desktop 选择通用 renderer。

所有当前第一方 Variable 都迁移；Catalog 也是标准变量，其值仍为 ResourceCatalogV2。Catalog descriptor 不内嵌 definition，避免 schema 自描述造成目录递归和体积膨胀。

## R2 change compared with the previous proposal

| 项目 | 上版提案 | 本次 R2 |
| --- | --- | --- |
| wire / runtime | 新旧 codec 并存 | 直接替换，仅实现当前变量契约 |
| SDK | 保留旧入口/签名与旧节点读取 | 统一接口与调用方，不保留 deprecated 桥接 |
| Metrics 控制 | 标准 write 与 /set 并存 | 只保留标准 write，删除被替代的 /set |
| Desktop | 保留 legacy 变量 renderer | 只实现标准变量路径，不写旧格式/视图迁移 |
| 实施门禁 | 为中间批次保持旧模式绿色 | 集成分支分批替换，最后统一完整门禁 |
| LEGACY02 | 后续删除 | 本轮清除残留与禁止双轨的必做门禁 |

## Cutover and consequences

- 所有第一方声明、消费方、测试 fixture、CLI 与生成绑定一起更新；旧私有 payload、旧 Variable 写入格式不转换、不解码，明确拒绝。
- 版本字段仍标识当前协议，但不实现版本协商或兼容分支。接口可原位调整，不因已有名称/参数签名保留两个 API。
- 公共 SDK 的领域 helper 只消费新标准值；它们是正常领域便利 API，不是旧协议适配器。直接同步修改其调用方。
- Metrics 可控制指标使用同资源 write，移除 /set、旧控制协议类型和相关别名/调用。新权限 fixture 直接按标准 write 授权，旧授权不自动映射。
- 产品内部显示 DTO 可由标准状态投影，但不得保留旧 SampleV1 解码或复制一份变量状态来维持旧界面接口。
- 不新增用户 identity、policy、profiles、配置/历史的兼容或迁移器，不自动扩大权限或清空文件。无关持久化结构无需为此次契约调整而修改。
- “仅内置定义”的旧阶段限制仅继续适用于本轮未改造的非 Variable 能力；Desktop 的 Variable 路径完全按标准 definition 消费。
- 回退依靠 Git 中整组变更和匹配产品构建，不在新运行时保留旧模式作为回退机制。

## Supersession

部分澄清 [Provider/Renderer ownership](2026-08-30_provider-schema-desktop-renderer-ownership.md)：提供者拥有实例语义，不等于可以另造标准变量协议。保留 renderer 归属 Desktop、远程定义不可执行的原则。

## Related

- [需求](../requirements/unified-variable-resource.md)
- [当前协议](../specs/unified-variable-resource.md)
- [执行计划](../../plan.md)
