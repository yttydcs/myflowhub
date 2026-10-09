# Unified Variable Resource

## Status and scope

Current R2，2026-09-06。用户明确产品未发布、无需任何兼容；已一次性替换被新变量契约取代的实现，不保留双轨。最终实现、匹配生产构建及本地合入状态见 [归档](../change/2026-10-09_unified-variable-contract.md)；不表示对外发布。
执行任务与验收状态见 [plan.md](../../plan.md)，产品意图见 [需求](../requirements/unified-variable-resource.md)。

仅升级 Variable 资源契约；继续使用 ResourceDescriptorV2、通用 Operation、既有订阅和树权限。不新增设备、插件或 schema 服务对象。

## 1. Responsibility

- protocol：固定平台值 envelope、写请求、值定义语法和有界校验。
- runtime/resource：统一存储、实例身份、revision、值校验、CAS、订阅快照和发布；不得导入产品包。
- 提供者：按标准定义实例类型/约束，采集或执行业务逻辑；可加强业务校验。
- SDK/bindings：严格解码标准变量，统一调整接口与所有调用方；领域 helper 只处理当前契约，不由 Desktop 重写 wire。
- Desktop：依据随快照传递的 definition 和 descriptor capability 选择通用组件。

## 2. Version and operation mapping

| 项目 | 当前契约 |
| --- | --- |
| 资源类型 | mfh.variable，type_version=2 |
| read 输出 | mfh.variable-value.v1；完整 envelope，必须有 definition |
| subscribe event | mfh.variable-value.v1；snapshot 必须有 definition，data 必须属于同一实例并可省略 definition |
| write 输入 | mfh.variable-write.v3；真实 JSON value + expected_instance_id + 十进制字符串 expected_revision |
| write 输出 | mfh.variable-value.v1；成功后完整、已确认状态 |
| 寻址和权限 | owner + name 不变；read/write/subscribe 继续是独立 exact capability |

schema ID 标识平台契约。产品可以给 definition 附带语义 ID，但客户端无需根据该 ID 查表才能理解变量，也不得按 ID 覆盖网络携带的有效定义。

上述版本号用于识别当前格式，并不表示同时实现此前版本；旧 descriptor、旧私有变量 payload 和旧写请求必须拒绝，不能走格式猜测、协商或兼容解码。

## 3. Standard value envelope draft

布尔变量的 read 示例：

    {
      "version": 1,
      "instance_id": "4c2d8b7e398043e59b2943e96512c641",
      "revision": "17",
      "definition": {"type": "boolean", "title": "充电中"},
      "value": false,
      "status": "ready",
      "source_time_unix_ms": 1788660000000,
      "observed_at_unix_ms": 1788660000001
    }

后续 data event 使用相同 envelope 字段，省略 definition；事件外层继续表达 snapshot/data/gap/expired 等订阅状态。
definition 只在本次注册实例内不变。更换定义必须显式替换资源实例，原订阅终止后重新发现/订阅，不允许同一个 instance_id 下悄悄换类型。

- instance_id：每次资源注册生成新的 128-bit 随机身份，32 位小写十六进制；不新增网络对象，不作为权限主体。
- revision：该实例单调递增的 uint64，以规范十进制字符串传递；范围 1..18446744073709551615，无前导零。耗尽时明确失败。
- value：JSON 原生值；Go 使用保真解码，不能先经 float64 抹平整数。false、0、空字符串不能被 omitempty 丢弃。
- status：ready / stale / unavailable。ready 必须有符合 definition 的 value；stale 必须有最近真实值；unavailable 不含 value。
- null：只有定义允许 null 时才是合法值；与 unavailable 的字段缺失严格区分。
- source_time_unix_ms：可选，表示 owner 提供的值来源时间；不要求所有变量都来自采集器。
- observed_at_unix_ms：平台记录此次状态更新的时间，整数毫秒。时间不代替 revision，也不假设跨节点时钟同步。
- error：stale/unavailable 可携带有界的 code/message；message 最多 1024 UTF-8 字节，不传堆栈或敏感实现信息。
- 只改状态或时间也增加状态 revision；不额外引入第二套值版本。Desktop 草稿规则见第 8 节。

远端 write 不接受 definition/status/time/revision 赋值，只接受：

    {
      "version": 3,
      "expected_instance_id": "4c2d8b7e398043e59b2943e96512c641",
      "expected_revision": "17",
      "value": true
    }

实例不同或版本过期均不能写入；前者重新加载，后者返回明确冲突。成功响应包括实际值和新 revision，不能把请求值当作外部执行已生效的证明。

## 4. Definition and validation

以 protocol/DataSchemaDefinition 的有界语法为基础，抽出适用于标准变量的验证入口；它不是一份以产品 ID 为必需键的内置清单。

- 标准类型：null、boolean、integer、number、string、object、array。
- 数值约束：minimum、maximum、multiple_of、precision、unit。
- 文本/枚举：长度、字符串 enum、allowlisted format；时间/大整数/有界二进制通过平台规定的 string format 表达。
- 对象与数组：properties、required、items、additional_properties；nullable 明确表达可空字段，不用采集失败冒充 null。
- 标准二进制值：平台 base64 string format 与 content type 标注；不强制把原始 bytes 当 UTF-8。这是当前数据类型支持，不是旧 byte payload 的自动转换路径。
- integer 的 JSON number 仅允许 JS 安全整数范围；需要完整 int64/uint64 的值采用平台规定的十进制字符串格式，禁止静默截断。revision 始终使用上节字符串。
- 复用现有最大深度 12、总属性数 128、每个 enum 最大 64 项；增加 definition 序列化后最大 64 KiB，上限与整个 envelope/resource payload 上限同时生效。
- 对象键、required、数值有限性、步长、字符串和数组边界在 owner 统一验证；外部数据在 SDK/Desktop 解码边界再次验证。Go/TS 共用测试向量，避免两端规则漂移。
- 不扩展任意表达式或远程代码。pattern/format 只允许可跨 Go/TS 一致验证的已有受限子集，不能把远端正则当任意 JavaScript 执行；不支持的定义明确报 unsupported。
- definition 是数据而非权限。read_only/write_only/sensitive annotation 不授予访问权；owner 不得依赖 UI 隐藏来保护数据。

定义在注册时验证并预编码；每次更新只验证值/状态和 envelope 大小。初值、定义或最终编码超限时注册失败，不发布半个 Catalog。
最大值接近旧 payload 上限的提供者可能因新增 envelope 开销超限，必须在迁移和边界测试中显式处理，不增加全局上限掩盖问题。

生成约束：DataSchemaDefinition 自身递归，不能把新 envelope 直接塞进现有反射生成器无限展开。平台值解析器与有限语法类型由 canonical 源生成/校验；动态 definition 不要求加入 Desktop 内置 schema 列表。

## 5. Storage and externally backed write

仅有一个公共 Variable 实现、一套当前格式和一条校验/存储/revision/watcher 管线。直接替换旧 byte-payload 契约，不保留旧 codec 分支或 MetricsVariable 运行时。

- 原位调整 Registry.Variable(VariableSpec)/NewVariable 的标准声明合同，包含 definition、initial state、capability limits 与可选 owner write handler，所有调用方同步更新。不新增平行的旧/新声明入口，不保留 deprecated 转发方法。
- owner 本地 Publish/Set 与远端 write 都经过公共约束、深拷贝、revision 和发布路径。标准接口返回结构化状态，不迫使产品手写 envelope JSON。
- 当前源码的本地入口包括 Set、CompareAndSet 和 Apply(revision, value)；其中拓扑包装使用 Apply，Catalog 使用 Set。所有保留入口都必须同步调整为标准值接口，执行相同 definition/value/status/最终编码大小校验；不能只更新远端 Operate(write) 或保留任意 byte payload 旁路。Apply 仅供可信本地发布，检查非零且严格递增的 revision，不变更实例身份，也不成为远端绕过 CAS 的入口。
- 每条发布路径必须先校验和准备完整输出，再提交值/revision；失败保持原状态且不发送成功事件。Catalog 注册/注销需要在资源表与新 Catalog 一致提交前完成业务值和标准 envelope 的编码/大小预检，不能在表中已可见后才发现新格式超限。最终通知继续走有序、锁外的公共发布机制。
- mutation gate 串行化条件检查、owner handler 与最终状态发布；快照锁仅用于短时读取/提交，不在 watcher 回调或外部 I/O 时持有。
- 先检查上下文、输入、instance/revision，再调用 handler。过期请求的 actuator 调用次数必须为零。
- handler 接收标准值并返回已确认实际值/状态，不得重入同一 Variable 的 Set；Metrics collector 和标准 write 使用明确的 per-metric 执行协调，避免锁顺序反转。/set 被删除。
- Windows 当前音量、静音、亮度 adapter 返回 Applied=true，可承接同步 write。声明标准 write 的 handler 必须能返回确认结果。
- deadline/故障后外部副作用可能已发生：明确报告结果未知，保留最近已确认值并触发下一次采集校正，不自动重试、不宣称物理回滚。不能同步确认的能力暂继续独立 Command，不伪装成已完成 write。
- 校验拒绝或正常 CAS 冲突不改变值，不向订阅者发布伪成功。结果发布失败也不得返回成功。

外部写协调的实现必须验证采集、配置切换和标准 write 并发；不能以“加锁”一词替代顺序与失败行为。

## 6. Provider migration

| 提供者 | 保留的业务事实 | 本轮变化 |
| --- | --- | --- |
| runtime system/catalog | ResourceCatalogV2 的资源列表、排序与原子 revision | 外层采用标准变量；SDK 不依赖先读 Catalog 才能解码 Catalog |
| feature/management | health/config/admission/topology 的对象值 | 统一声明与 read/subscribe envelope；topology 自定义 read/Observe 路径也必须一致 |
| feature/file | transfer summaries 与实际传输流程 | 仅摘要变量迁移；File session 和大数据面保持 |
| Clipboard | status/config、历史、桥接和现有更新操作 | 状态/配置成为标准对象变量；不重做剪贴板同步 |
| Metrics config | 持久化 ConfigV1 和现有 update 命令 | 标准对象变量，提供者给出平台语法的对象定义 |
| Metrics samples | 原指标路径、owner、采样失败语义与平台适配 | 输出真实数值/布尔/文本和标准状态，网络不再以 SampleV1 包装变量 |
| Metrics controls | 有界实际控制、业务校验与 exact 权限原则 | 使用同变量 write，删除被取代的 /set、旧控制协议类型与调用方 |

Metrics 本机 Wails 状态界面、CLI 和测试直接改用平台状态或其显示投影，同步更新 facade/生成绑定；不保留旧 SampleV1 解码、转换桥或另一份采样变量状态。允许产品显示 DTO，但它必须由标准状态派生，不能是兼容旧接口的理由。
定义、标签和业务对象 schema 由产品本地声明；protocol 不导入 apps/nodes/metrics 或 Clipboard。

配置 update 等有持久化/业务校验语义的能力按其实际职责处理；不能仅为兼容旧入口保留已被标准 write 完全替代的 wrapper，也不把无关 Command 全部删除。

## 7. SDK and subscription

- `Snapshot` 返回标准 `VariableValueV1`；`WriteVariable` 接收期望 instance/revision 与原生 JSON 值并返回确认状态；`SubscribeVariable` 消费当前标准订阅。`VariableEventDecoder` 为后续 data 缓存实例 definition，领域调用使用 `DecodeVariableEvent`；没有旧签名重载/转发层。
- Catalog/领域客户端只解码当前固定平台 envelope，验证后读取 value 并执行目标业务 Validate；可保留有独立价值的领域 helper，但无旧 payload 检测、fallback 或多版本分支。
- 通用 Operate 返回真实 wire envelope，不暗中解包；标准变量 API 与领域 helper 的职责写入文档。
- `EncodeVariablePayload` / `DecodeVariablePayload` 只在标准变量的 Go 领域值边界工作：definition 明确声明的 node-id/int64/uint64 十进制字符串无损对应 Go 整数；先验证当前格式再恢复目标类型，数值形式的旧 owner 字段不作为兼容输入接受。DataSchemaByID 为精确 Go NodeID 类型生成字符串定义，不按字段名猜测，也不改其他 operation、身份或策略保存的 JSON 格式。
- Catalog bootstrap 直接读取 system/catalog 并判断标准 schema/版本，不再先请求 Catalog 或全树。
- Subscribe 的初始状态和 watcher 注册必须原子，先 snapshot 后 data；重连重新取得 definition，撤权/注销终止或失效，保留现有有界队列和背压。
- SDK/binding 必须保持 instance_id 和 revision 字符串。原订阅外层 uint64 revision 在 Go 边界可校验一致性；Desktop 新控制器仅以标准 envelope 的精确 revision 为依据，不使用可能被 JS 舍入的外层数字。
- binding/TS/Wails 由 canonical Go 定义生成，不手写影子契约。

## 8. Desktop behavior

- read 获取 definition 与 state 后选择 renderer；无需识别 Metrics schema ID，Inspector 与 View 使用同一个 Variable controller。
- 有 subscribe 能力且授权成功时持续消费 snapshot/updates；SDK durable subscription 负责续租和恢复，Desktop 在 gap/expired 后等待新快照。仅 read 被允许时仍可读取并手动刷新，清楚显示未订阅原因，不能把订阅失败标为实时。
- 快照、写响应、订阅更新按 instance_id + revision 合并，丢弃旧会话结果；注册实例改变后重新取定义，不跨 owner/Profile 复用缓存。
- definition 缓存只属于正在使用的资源会话，有上限，释放最后一个消费者后清理；不持久化网络定义、值或草稿进 View。
- 只读数值/布尔展示，支持 write 的变量提供通用编辑；capability 不代表当前主体已获授权，forbidden 显示在当前控件，不执行旁路。
- 用户修改后保存 draft base。新事件若 value 与 base 相同且仅时间/状态变化，保留草稿并将 CAS 基准推进到最新 revision；实际 value 改变则保持旧基准、提示冲突。不可用、撤权或实例更换时禁止提交旧草稿。
- Inspector 编辑草稿后显式 Apply，Reset 还原最新确认值；View 开关/枚举点击提交，滑块结束交互提交，数字/单行文本 Enter 或 blur 提交，多行/高级 JSON blur 提交。输入法确认不提交，Enter+blur 去重，相同值不写入。
- View 只提交本编辑器创建的草稿 generation，释放组件、切换 Profile/实例或后续其他编辑即令旧意图失效；不代替 Inspector 提交。写入期间控件禁用并显示 aria-busy/加载态，开关/枚举保留确认值直至 write 响应；订阅先到不提前显示成功或在途冲突。失败保留草稿，不自动重试，提供显式恢复入口。
- 所有面板按 definition.title（缺省资源末段）显示标题，订阅圆点紧随标题；View 隐藏常驻详情与操作页脚，权限失败、失效、冲突等异常仍可见。只读嵌套结构全宽分组，数组对象按项折叠，展示有数量/深度边界。
- 未知新节点按标准声明即可展示；旧变量/未知版本明确 unsupported。Variable 路径不保留 legacy renderer，不新增旧 renderer ID 别名、旧视图格式解码或自动迁移；按当前契约选择兼容控件不属于历史版本兼容。
- 包含 false/0、stale/unavailable、超过 JS 安全范围 revision、刷新与订阅交错、组件卸载、Profile 切换、同名资源重建的测试。

## 9. One-time replacement

同一集成分支内先替换平台/SDK，再并行更新提供者与 Desktop，清除旧实现残留，最后做全量门禁并交付匹配构建。中间阶段允许未更新调用方暂时不通过编译/旧测试，只验证已完成切片；必须明确未集成状态，不能为维持中间绿色引入兼容代码。

- 不支持旧 Variable type_version、旧 payload 或写格式；删除对应运行时代码、schema 常量、注册/解码和生成物；保留最小拒绝测试证明没有隐式接受旧格式，不把旧完整实现留作测试依赖。
- 不支持新旧节点混用；旧请求明确拒绝，不协商、不转译、不回退。产品尚未发布，不提供外部 API 兼容承诺。
- 删除 Metrics /set；测试策略和调用方直接使用标准 write。旧 capability grant 不自动映射，用户持久化策略不自动修改。
- 新测试使用隔离目录/当前格式配置。无需新增数据、视图或配置迁移器；也不自动删除真实文件。无关身份、策略和历史的现有合同保持本轮范围外。
- 回退是整组 Git 变更/匹配构建回退，不在运行时切回旧模式；涉及不再可读的本地试验状态时显式报告，由用户决定处置。
- 旧变量 API/codec 和 /set 的移除属于本轮必做；非 Variable schema 分发与异步执行模型继续延期。

## 10. Validation

平台合同测试、未知提供者 memory/TCP 闭环、全 provider catalog/read/subscribe 审计、并发写与 actuator 计数、Go race、SDK/生成契约、Desktop 控制器/组件、Windows Desktop/Metrics 生产包联调。
本地 Set/CompareAndSet/Apply 与远端 write 分别验证非法类型/状态/超限时状态和 revision 不变、订阅者未收到伪更新；成功均观察到同一标准契约事件。Catalog 注册/注销失败验证目录、资源可见性和 revision 一致，拓扑 Apply 验证版本回退拒绝及订阅发布。
既有 children/subtree 的 depth=0/1/n、权限和缓存回归保留；本方案不修改拓扑导航策略。
具体命令、任务与观察边界见 [计划](../../plan.md)。

实现补充：Registry 在目录事务内先通过 `PrepareRetire` 标记旧实例不可再注册，锁外等待在途写入并发送终止通知；状态提交后 watcher 逐个隔离，异常回调会被移除并记录错误，不中断其余观察者的有序发布。此边界同样适用于 topology 的 Variable 包装。
