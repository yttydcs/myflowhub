# Collection browser 的跨运行时契约边界

## 适用场景

当 Go provider、JSON/Wails binding、JavaScript UI 与真实 packaged Desktop 共同实现 Collection 分页、member detail
和内容 renderer，或共同消费平台 Variable 时，组件测试通过并不代表跨运行时契约正确。尤其要同时区分 UI 原子状态、
各资源自己的数字合同、opaque cursor、wire envelope 与解码后的领域值或 member content。

## 复现症状与根因

### 返回或关闭后操作状态再次出现

- 症状：jsdom 测试通过，但真实 Wails 中 Return/Close 看起来无效，之后切换 View 或 selection 又恢复旧 draft。
- 根因：selection 与 focused action 分散在多个 state 更新中，旧 closure 可以重新提交半套状态；同时 packaged
  进程可能仍嵌入上一次 build 的 `dist`。
- 防错：把 selection/action 作为一个原子状态更新；真实 GUI 验证前重新 build、重新 package、关闭本次测试启动的
  旧进程并确认 asset hash 变化，不把已有手动环境当作可重置的测试 fixture。

### 下一页报 revision 不是安全正整数

- 症状：第一页正常，翻页后 JavaScript 报 `Collection page revision 必须是安全范围内的正整数`，或 cursor 被判 stale。
- 根因：Go `uint64` hash 被直接放进 JSON number，超过 `2^53-1` 后 JavaScript 舍入；若 cursor 仍绑定完整 hash，
  页面 revision 与 cursor 就不再代表同一快照。
- 防错：wire revision 限制在 `1..2^53-1` 且耗尽时明确失败；cursor 继续使用 opaque、完整 fingerprint，并同时绑定
  parent/revision/fingerprint。这是 Collection page 的数字合同，不能套到采用规范十进制字符串的 Variable revision。
  不能让客户端解析 cursor，也不能用缩短数字代替快照指纹。

### JSON member 显示 wrapper 字段或大量占位符

- 症状：读取 JSON 文件后 renderer 显示 `Version/Encoding/Data` 等 envelope 字段，真正的 `fixture/items` 不见了。
- 根因：filesystem read response schema 被直接套到已解码的 inner JSON 上，混淆了 transport envelope 与内容 schema。
- 防错：先严格校验 envelope 和 encoding，再依据 `content_type`/member schema 选择 inner renderer；unknown binary、
  HTML 与 SVG 必须走不可执行 fallback，不能因为扩展名或 Resource name 绕过 content gate。

### Variable 首次读取正常，后续 data 丢失类型或串用旧定义

- 症状：read 能显示，订阅 data 省略 definition 后失败；同名资源重新注册后沿用旧类型；false、0、空字符串或合法 null 被当作 unavailable。
- 根因：把 typed Variable envelope 当成任意领域 JSON，或把 definition 缓存成按 schema ID/资源名索引的全局表；只验证 data 的 JSON 结构而未复验缓存定义。
- 防错：先验证标准 envelope；read/snapshot 必须携带 definition。每条订阅缓存其 resource、instance、revision 和不可变定义，后续 data 按同一实例的定义复验。gap/expired 后丢弃缓存，收到新 snapshot 前拒绝 data；实例或同实例定义非法变化不能污染已确认缓存。ready/stale 的 value 按实际字段存在性检查，unavailable 则必须没有 value。
- 证据：[SDK 变量合同测试](../../sdk/go/variable_contract_test.go) 的 `TestVariableEventDecoderCacheAndBoundary`、`TestVariableUnavailableAndStaleRemainDistinct`；`TestVariableDataChecksFullEnvelopeLimitWithCachedDefinition` 还验证加回缓存 definition 后超限时，decoder revision 不推进。领域 helper 在标准 envelope 校验后解码 value，通用 Operate 保留 wire。

### 大 NodeID 使 Catalog 校验失败，或 CAS 修订在 JS 中变化

- 症状：固定小节点 fixture 正常，真实分配的大 NodeID 使 Catalog 的 `resources[].id.owner` 被 JS-safe 校验拒绝；超过 `2^53` 的 revision 经 Number 转换后无法精确 CAS。
- 根因：Go 的 `uint64`/NodeID 直接成为 JSON number；字段名和字符串外观不能证明它采用哪一种 wire 合同。
- 防错：Variable revision 始终使用 `1..18446744073709551615` 的规范十进制字符串，禁止前导零、符号和 Number 中转。`DataSchemaByID` 的变量视图仅把确切的 Go `NodeID` 映射成 string/`node-id`；变量领域 codec 按定义显式声明的 `node-id/int64/uint64` 保真转换，拒绝旧数字形式，已有 string 字段保持 string。其他普通整数仍受 JS-safe 约束，不缩减 NodeID 分配范围，也不修改全域 NodeID JSON、普通 command 或 persistence codec。
- 证据：[领域 codec 回归](../../protocol/data_schema_variable_payload_test.go) 的 `TestVariableDomainCodecFullWidthRoundTrip`、`TestVariableNodeIDViewAndNonVariableWireUnchanged`、`TestVariableDomainCodecCatalogAndExistingProviderStrings`，以及 SDK 的 `TestVariableCatalogPreservesLargeNodeIDs`。当前内置定义中，此次 NodeID 映射改变的字段只有 Catalog 的 `resources[].id.owner`。

### 统一 Variable JSON 时误伤 Stream 二进制事件

- 症状：变量能显示，但 Stream 的非 UTF-8 字节被替换、JSON-looking bytes 被拆成对象，或合法空事件被丢弃。
- 根因：把 Variable 的 `json.RawMessage` 编码分支用于所有订阅，误以为每个 `value` 都是标准 Variable envelope。
- 防错：先按资源/事件契约区分传输层。Variable 携带 typed JSON envelope；非 Variable Stream 继续传递原始 bytes，并在 JSON binding 中使用对应的 base64 字节表示。即使 bytes 看起来像 JSON，也不能据此转成 Variable 或套用其 definition。
- 证据：[binding 回归](../../sdk/bindings/variable_contract_test.go) 的 `TestBindingStreamValuesKeepByteTransport` 分别验证非 UTF-8 binary、JSON bytes、普通文本和空 bytes 的往返及 sequence 保持。

### 真实进程验证只能管理本次测试拥有的状态

- 触发：新契约不兼容旧试验配置，或需要重新打包的 Desktop/Metrics 验证时，已有用户 Profile、身份、policy 和手动进程仍在使用。
- 防错：使用本次测试新建的空目录、独立节点状态和显式 exact grants；非空目录应拒绝并另选 fresh run，不能清空旧 Profile 来换取测试通过。路径校验后只清理本次分配的临时状态，停止本次启动的进程；不修改既有身份/策略，也不把“无需兼容”解释为可删除真实数据。
- 证据：[TCP GUI fixture](../../apps/desktop/topology_gui_test.go) 检查目录位于工作树 `out/qa-topology/gui` 内且初始为空，清理仅指向新建 node-state；[Variable GUI fixture](../../apps/desktop/unified_variable_gui_test.go) 只在该测试目录准备 Metrics 状态与 permit/grants。这些隔离机制不等于已完成所有真实平台 smoke，最终运行证据应单独记录。

## 最小检查顺序

1. 确认测试使用的是当前源码生成的 `dist` 与当前 packaged executable，而不是陈旧嵌入物。
2. 比较 Go response、Wails JSON 与浏览器实际值：Collection page 检查安全整数，Variable 检查规范 revision 字符串与 instance；不要共用错误的数值假设。
3. 分别记录 envelope schema、member descriptor schema 与 decoded content schema；不要让一个 schema 承担三层职责。
4. 在本次测试拥有的目录与进程中，用真实 Hub、permit/policy 和 provider 做分页、Forbidden、Return/Close 与二次重启 smoke。
5. 检查 View persistence 中不存在 member key、content、draft、result、credential、permit 或 permission decision。

## 关键词

`focusedAction`、`generation token`、`Number.isSafeInteger`、`2^53-1`、`opaque cursor`、`full fingerprint`、
`content_type`、`filesystem read envelope`、`VariableValueV1`、`cached definition`、`node-id`、`base64 Stream`、
`test-owned state`、`stale bundle`、`Wails go:embed`。

## 当前技术真相

- [Resource Collections and Actions](../specs/resource-collections-and-actions.md)
- [Unified Variable Resource](../specs/unified-variable-resource.md)
- [Observable side effects](observable-side-effects-and-generated-contracts.md)
- [Desktop](../features/desktop.md)
- [Flow](../features/flow.md)
- [Resource Platform v2](../specs/resource-platform-v2.md)

## 变更证据

- [Resource Collections、Capability Actions 与 Desktop 交互](../change/2026-08-31_resource-collections-and-capability-actions.md)
