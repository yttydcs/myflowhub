# 高频变量链路与定义缓存

## Summary

2000个活动订阅并不证明2000@1Hz能够及时到达界面。A阶段本机验证先发现帧分段写导致发送队列溢出，随后发现原生界面采样明显落后。合并帧写入已通过本机参考负载；类型化definition缓存降低了解码成本，但原生全链路时效仍须复测，不能从微基准推导通过。

## Lookup Hints

搜索：queue overflow、superseded、observed_at、expected revision/current revision、VariableEventDecoder、DataSchemaDefinition.UnmarshalJSON、2000订阅、实时但数值过期。先比较采样时间与观察端时间，检查队列和provider/consumer计数，再查看CPU与分配profile；先核实两端时钟。

## Symptoms

- Windows参考流量下短时间内发送队列溢出。
- 总览显示2000/2000订阅，但采样时间落后数十秒，显式写入发生CAS冲突。
- CPU及分配profile显示同一schema在每次数据事件中重复解析。

## Impact

更新积压影响实时性与条件写入。仅扩大队列或放宽CAS会掩盖吞吐不足并损害状态一致性；活动句柄计数不能代替值的新鲜度。

## Trigger Conditions

Windows TCP、2000个1Hz变量、原生Wails IPC与前端同时接收；定义只在快照中携带、后续事件从会话缓存恢复。单进程SDK负载缺少原生桥接和界面成本。

## Root Cause

帧编码原先将头部及载荷拆成多次底层写入。SDK还把已验证schema保留为JSON，每个无definition事件重新解码。前端目录排序在每次比较时创建带numeric选项的locale比较器，增加大目录排序开销。后续profile仍显示事件解码分配高；检查发现无definition事件先通用解析value，再按缓存定义重复解析。上述成本分别已有代码与测试/profile证据；原生积压的全部原因尚未收敛。

## Investigation Trail

先保留失败日志和原始采样时间；确认同机时钟一致，再比较SDK参考负载与原生GUI。使用不开放调试端口的临时测试入口采集Go CPU/alloc profile；修改后运行相同解码微基准，并增加返回schema被调用者修改的回归测试。

## Resolution

帧按既有字节布局一次写出并保留短写补齐；批量poll从64调到256事件，仍受原字节预算限制；目录排序复用Intl.Collator。SDK缓存类型化definition并对所有可变字段深拷贝，保留独立绑定校验、事件校验、完整载荷硬界及失效规则。代表性对象事件微基准从443336ns/36321B/623次分配降至161330ns/11385B/214次分配；该数字只描述本次微基准。

后续候选将无definition数据事件合并为一次基于会话定义的校验，仍使用VariableValueV1严格解码，并分别检查原始wire（含空白）和补全定义后的大小。新增坏包测试曾发现json.Unmarshal只把对象本体交给自定义解码器，故不能只在UnmarshalJSON中检查整体字节上限；已补显式原始长度检查。该轮微基准75744ns/8607B/171次分配，端到端效果另验。

轮询合批放在Go侧，避免前端通过浏览器定时器推动消费；已有12项前端相关测试及后端合批/取消测试。浏览器定时器16ms实验虽有一次实际控制成功，持续采样仍滞后，已保留失败截图并移除该方案，不能据单次写成功放行参考负载。

进一步以同源原生被动poll统计定位桥接成本：旧字符串接口约1827条/秒，扩大512条批次仍约1923条/秒。保留256条/字节预算，新增结构化批量返回，去掉二次字符串包装和前端重复解析后，60秒消费122576条，采样年龄中位441ms、最大1098ms。旧接口保留，新增编码等价和非法批次回归。活动计数与吞吐、新鲜度必须分别核对。

2000项故障测试还揭示续租间隙：旧句柄正常退休后，如果下一次建立前等待断连或建立失败，没有gap会让旧值无限保持live。把失效放入连接等待循环和瞬态建立失败路径，在下一份新值前只发一次gap；确定性测试先重现两条旧路径失败，再验证修复。只检查音频两项会漏掉未显示的281项错误状态，故障验收必须核对全体活动订阅。

## Prevention / Guardrails

同时检查provider发布数、实际订阅数、消费数、采样年龄和显式写入结果；不可丢弃协议事件来伪造吞吐。缓存返回对象不得泄露可变引用，非法事件不得推进缓存修订。性能检查与重构前后使用同一用例和边界；记录机器负载，保留超时失败后再隔离复测。Linux容器、SDK、浏览器模拟及原生UI证据分别标记。

## Related Intake / Features / Requirements / Specs / Decisions / Changes

- [Desktop现状](../features/desktop.md)
- [A验收标准](../requirements/phase-a-acceptance.md)
- [容量与恢复合同](../specs/phase-a-foundation.md)
- [本次实施和证据](../../review.md)

本次未改变产品需求或架构决策。2026-10-29 用户接受风险并完成阶段性收尾，A正式验收仍未完成，见[阶段归档](../change/2026-10-29_phase-a-foundation.md)。
