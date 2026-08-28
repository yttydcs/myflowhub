# Resource Platform v2

## Status

这是已实现的 Resource、Topic、通用操作与目录 canonical contract。它取代
`resource-model-vnext.md`、`resource-catalog.md`、`command-vnext.md` 中依赖固定三类型的部分。

## Identity and ownership

所有设备都是 Node。资源只存在于某个 Node 下，以 `(owner NodeID, local name)` 唯一寻址。
资源名可以分段用于展示，但不会形成第二棵权限树；authority、路由和资源归属仍服从唯一节点树。

## Descriptor

每个资源公开稳定描述符：

- `type` 与独立演进的 `type_version`；
- 一个或多个 capability，每个 capability 声明 action、input/output/event schema 和尺寸限制；
- 资源级 limits 与 presentation hint；
- 可选标签和说明，仅用于发现与默认呈现。

类型标识和 capability 标识是受限、可扩展的字符串。Core 不包含产品类型枚举，也不根据类型名
决定权限。未知类型仍可发现；未知 capability、schema 或不支持的 major version 必须明确拒绝。

## Generic operation

调用方发送 resource、capability、schema 和 bounded payload。authority 按
`subject + resource + capability/action` 裁决，owner registry 再执行类型拥有的验证与处理。
结果使用关联 ID 返回，错误码稳定区分 unsupported、malformed、forbidden、expired、gone、overflow
和 internal。handler panic、deadline、重复请求及过大输入均在 owner 边界隔离。

## Built-in types

- Variable：`read`/可选条件 `write`/`subscribe`；写入携带 expected revision，快照先于更新，revision 单调递增。
- Stream：`subscribe`，可选 `publish`；事件 sequence 单调，丢失必须以 gap 表达。
- Topic：`publish`/`subscribe`；多个发布者和订阅者，按 publisher 分别排序，默认无持久化和回放。
- Command：`invoke`；输入与输出 schema 独立声明，支持有界输入输出、deadline、dedupe 与 panic 隔离。
- File：`open` session 和 progress observable；数据块不占普通控制消息队列。

新增类型只需注册 descriptor validator/handler，可选 Observable/Session 接口，以及可选 Desktop
renderer；不得要求修改 Core type switch。

## Catalog

每个 Node 的 `system/catalog` 是一个 Variable，内容按资源名稳定排序并携带非零 revision。
注册、注销或 descriptor 变化必须原子更新目录。目录自身也必须被目录描述。

## Topic contract

Topic 的 owner 负责 broker、policy check、速率限制和 fan-out。发布事件包含 publisher、
publisher-local sequence、timestamp、schema 与 payload。subscriber 慢消费不会阻塞 registry 锁；
有界队列溢出时产生 gap。默认订阅只接收建立关系后的事件，不承诺全局总序或 replay。

## Security invariants

- 父节点对直接子节点保持原有强控制权，子节点信任父节点的 authority 裁决。
- owner 仍必须验证 schema、尺寸和类型约束，authority 许可不是业务输入验证。
- permission key 不从 presentation hint、物理 transport 或 UI 控件推导。
- reparent、policy generation 变化或资源注销会撤销相关 subscription/session。
