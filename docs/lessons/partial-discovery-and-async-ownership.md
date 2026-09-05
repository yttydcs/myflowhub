# 部分发现缓存与异步操作归属

## Summary

把完整树改为按需发现时，最容易遗漏的是“这个结果仍属于当前状态吗”。响应完整性、缓存有效性、权限和异步归属是不同问题，必须各自验证；只为请求增加depth不能解决这些边界。

## Lookup Hints

- 症状：展开后丢失父关系或名称、退出分支复活、没有请求却一直loading、切换Profile后访问旧资源、保存完成覆盖新草稿。
- 关键词：partial topology、scope root、has_children、instance/revision、inFlight、queued request、Profile generation、View owner、late save。
- 触发条件：多owner不同深度响应交错，连接切换，队列未启动就失效，View或编辑发生在异步操作完成前。
- 快速检查：从调度、执行到提交/finally逐步检查请求归属；同时检查View和catalog effect，不能只检查topology handler。

## Symptoms / Impact

浅层响应被当全量替换会删除尚未重新读取的分支；查询根省略外层parent不代表reparent。祖先缺省名称可能覆盖后代自己的已知名称。旧View引用可能在新Profile中触发目录或业务访问。迟到保存会使用户输入丢失，即使网络请求本身成功。

## Trigger Conditions / Root Cause

- 用一个完整数组表达部分知识，未区分未加载、已加载为空、失败和过期。
- 把provider revision当全网世代，或只检查请求ID而不检查Profile/连接/分支的归属。
- 在任务函数内部清理inFlight；任务在调度器入口被跳过时，finally根本没有运行。
- 把旧View先留在组件中，等新View返回再覆盖；新连接的catalog effect先观察到了旧引用。
- 保存回调无条件setView和clearDirty，没有比较当前View及后续编辑世代。

## Investigation Trail

1. 用deferred promise控制返回顺序，记录实际owner、深度、Profile和在途数量。
2. 分别触发owner浅层/深层、ancestor/descendant响应，检查哪些直接孩子集合确实被完整覆盖。
3. 在4个并发槽占满时排队，再切换scope或用更深响应覆盖排队请求；检查队列清理。
4. 让Profile B的status先到、views后到，确认没有用A的View引用访问B网络。
5. 保存V1后新建/打开V2并编辑，或继续编辑V1，再交付旧保存响应。

## Resolution

- 仅替换响应明确完整覆盖的集合；边界以外缓存、已知外层parent和后代自报名称按契约保留。
- 以Profile、连接generation、查询边界和请求序号约束结果提交；分支删除或迁移同时使相应在途结果失效。
- finally覆盖调度器整个任务生命周期，包括未进入执行体的取消/跳过路径。
- View文档绑定Profile及generation；未加载好当前View前不消费旧owner列表或渲染旧业务组件。
- 保存结果仅在同View且编辑generation未变时替换草稿；同View已继续编辑时只推进已确认revision，保留未保存内容。

## Prevention / Guardrails

自动化测试验证异步先后顺序、实际API调用及草稿值；真实页面验证导航、错误、键盘和空间可用性。两类证据互补。缓存不持久化、不代表授权；树不可见不阻止已知Resource reference按正常权限路径访问。

## Related Docs

- Intake：[按深度发现讨论](../intake/2026-09-06_depth-scoped-topology-discovery.md)
- Feature：[Desktop](../features/desktop.md)
- Requirement：[拓扑发现](../requirements/topology-discovery.md)
- Spec：[查询与缓存合同](../specs/topology-discovery.md)
- Decision：[浅层/递归能力](../decisions/2026-09-06_depth-scoped-topology-capabilities.md)
- Change：[本轮执行证据](../change/2026-09-06_depth-scoped-topology-discovery.md)
