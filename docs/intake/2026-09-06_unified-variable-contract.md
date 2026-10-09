# 平台统一变量与通用消费

## 来源与状态

- 来源：2026-09-06 当前 Codex 对话，Metrics 与 Desktop 手动联调后的连续讨论。
- 用户已确认产品原则，完成 $m-plan 后调用 $m-execute，批准 R2 的 7 项执行任务；EXT02 延期。执行结果见活动计划。
- 技术方案与执行范围以 [活动计划](../../plan.md) 为准。

## 用户原话

> 变量应该是全局通用的，不是metrics特有的，是metrics采集到的数据，以网络中的一种资源“变量”，提供给了网络

> 这样变量的格式是统一的，而不是node单独实现一个变量，这个价值取向和产品设计里面没写吗

> mfh.metrics.sample.v1 这个又是什么呢，为什么展示面和节点耦合了？不应该是desktop提供一个通用的变量展示吗

本轮 $m-plan 修订时用户进一步明确：

> 当前不需要任何兼容，产品还没发布

此前用户要求：切换方案前给出当前方案与拟议方案对比，不直接修改代码；网络模型只认识节点、网络和资源，节点不是设备，网络没有插件概念。

## 讨论结论

Variable 是平台定义的资源类型；Metrics 是变量实例的提供者，负责采集和更新。Desktop 按统一契约消费，不需要认识 Metrics、资源路径或产品私有 schema ID。值定义的语法由平台规定，实例类型、范围、单位、领域校验由提供者按该语法声明。

统一格式允许布尔、数值、文本及结构化值；不等于去掉 owner、共享一个全局可写命名空间，或让所有变量都可写。未知的新节点按标准声明变量后，应立即可被现有通用消费者理解。

产品尚未发布，本轮采用一次性替换。上一版计划中保留旧 Variable 格式/接口、legacy renderer、Metrics /set 和混合版本读取的安排由此取消；这些是助手此前提出的兼容假设，不是用户要求。平台、提供者、SDK、Desktop 和相关调用方一起更新，不添加双轨代码、过渡别名或自动迁移器。协议仍可有版本标识，但只实现当前一种契约。

不需要兼容不等于授权清空用户本地数据或修改其他工作树；规划时仅修订文档，后续实施依据单独的 $m-execute 批准进行。

用户在执行联调时进一步要求：

> 我的意思是能不能通过浏览器来操作？

随后回复“好的，请继续”。后续自动联调使用后台浏览器，不占用用户鼠标、键盘或桌面焦点；已完成的原生生产包观察继续保留，浏览器验证不替代原生特有功能验收。

## 规划时的实现证据与缺口（历史快照）

- 公共 runtime/resource.Variable 已实现 revision、read/write/subscribe；Metrics 调用公共 Registry.Variable，并未复制整套运行时。
- apps/nodes/metrics/schema.go 的 SampleV1 将 value 定义为 string，并把采样状态和时间一起包装为 Metrics 专用 payload。
- Desktop 的 SchemaResolver 只有编译生成的内置定义，未知 ID 回退 raw；这来自已批准的旧阶段边界，不是仅漏写了一个 renderer。
- Desktop VariableRenderer 的只读路径读一次快照，可写路径临时订阅以取得 revision 后取消；尚未形成持续更新与草稿冲突处理的完整通用控制器。
- 现有其他提供者包括 system/catalog、管理状态/配置/拓扑/准入状态、文件传输摘要、Clipboard 状态/配置和 Metrics 配置，必须纳入迁移。

## 不采用的修补

- 不以补齐 mfh.metrics.sample.v1 内置定义作为最终方案。
- 不根据 metrics/* 路径猜类型、主字段或 /set 绑定。
- 不把设备或插件加入网络模型，不恢复 Flow 或退役平台代码。

## 后续文档

- [长期需求](../requirements/unified-variable-resource.md)
- [当前技术契约](../specs/unified-variable-resource.md)
- [方案选择与一次性切换影响](../decisions/2026-09-06_platform-variable-contract.md)
