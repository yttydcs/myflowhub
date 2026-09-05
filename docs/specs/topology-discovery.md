# 拓扑发现协议与缓存

状态：Current，2026-09-06 实现及验收通过。参见[需求](../requirements/topology-discovery.md)与[能力边界决策](../decisions/2026-09-06_depth-scoped-topology-capabilities.md)。

## 资源与 wire 合同

资源为 `(N, system/topology)`，owner N 就是查询根。请求不另带任意 root 参数。原 read/subscribe 和 `mfh.management.topology.v1` 保留。

| Capability | 输入 schema | depth |
| --- | --- | --- |
| children | mfh.management.topology-children-request.v1 | 必须为 1 |
| subtree | mfh.management.topology-query-request.v1 | 0..4096 整数 |

两个请求均为 `{version:1,depth:number}`，depth 必填且不可 null；未知字段、负数、分数、文本和超界值拒绝。0 为不限深度；1 为直接孩子；正数 n 为最多 n 层。可选参数的前端入口默认 1，Go helper 显式接收 depth。

输出 schema 为 `mfh.management.topology-query.v1`：version=1、root_node_id、depth、instance_id、revision、nodes。instance_id 为 provider 生命周期内稳定的随机 128-bit 标识；revision 为 1..2^53-1。Node ID 使用正整数字符串。节点字段延续 node_id、parent_id、display_name、role、generation，另有必填 has_children。

响应恰有 owner 根，根省略 parent_id；其他节点父项均在响应内，无环、无重复、无不可达项。边界 has_children=true 表示快照中存在下一层，不表示访问权限。depth 内的直接孩子集合完整；未到深度边界的 has_children 必须与返回关系一致。校验用有界线性图遍历。

任何查询最多 4096 节点且受 DefaultMaxPayload 约束。超限返回 Overflow；全量超限不能阻塞合法浅层结果。revision 仅在同 owner、同 instance 下比较。

## 权限

复用 Subject + Resource + Capability 的 exact/scoped policy。children 不隐含 subtree/read/subscribe；subtree 不隐含 children。children handler 再次验证 depth=1，无法通过伪造 payload 扩大授权。

旧 read/subscribe 仍暴露完整旧快照，已有宽授权不自动变窄；superadmin/all 覆盖新能力。不得自动迁移或写入用户授权。缓存查询仍经过正常鉴权，不缓存许可结论；保留现有父控子信任。目录与业务操作独立授权。未挂载 provider 的节点明确返回 unsupported/not found。

## 服务端状态

runtime/tree 原子读取 local、parent、直接边、后代关系、Epoch 和 membershipEpoch；未变化可在锁内判断后直接返回。该层不依赖 management 或 JSON。

management 在现有周期刷新中根据 Epoch、membershipEpoch 和输出相关本地配置生成不可变索引，预计算 depth 1 和 0。其他深度最多缓存 8 项、合计 8 MiB，不为每个后代复制整棵子树。加入、退出、reparent、本地名称变化更新 revision；provider 重启更换 instance。请求读取已发布快照，不等待后代网络查询。

原 Variable 被可观察 Resource wrapper 包装；旧读/订阅保持可用，新查询归属于同一资源。序列化、网络和 watcher 回调不应持有全局树锁。

发布按顺序排空，重入刷新只排入最新待发布值；观察者版本不得倒退。旧全量快照超限时，已有订阅通过现有 wire Error 收到 Overflow 并终止；恢复后可以重新订阅。该终止由内部 Observation.Failure 传递，不改变 wire schema。

## SDK 与 Desktop

Go `ManagementClient.QueryTopology(ctx, depth)`：1 选择 children，其余选择 subtree，使用 attached Client 的通用 operation。Desktop `topology(owner, depth=1)` 使用已有 OperateJSON；不增加第二套生命周期或重复 Wails facade。

发现控制器按 Profile、连接 generation、浏览根隔离；每 owner 记录 instance/revision、请求序号、孩子集合及 unloaded/loading/loaded/error、stale、错误时间。相同在途请求去重，发现和 catalog 合计最多 4 个请求。折叠保留缓存，软有效期 30 秒；重新展开过期项先显示旧结果再刷新。

合并只替换响应完全覆盖的直接孩子集合。边界以外缓存保留；查询根缺省 parent 不抹除外层父关系。祖先快照缺省后代名称时保留已知自报名称，owner 自己的根响应仍可更新或清除名称。删除或迁移分支使其请求失效；跨连接/起点的迟到结果丢弃。

展开偏好只恢复已知路径。鼠标和键盘 Right/Left/* 共用加载逻辑。搜索标为“搜索已加载节点”；明确起点上的“加载完整子树”使用 depth 0，无自动回退。

catalog 按选中节点和当前 View 引用加载，可在 owner 尚未出现在树中时独立访问。只有成功空目录才显示无资源；未加载不是 detached。节点缓存上限 4096，catalog owner 上限 128，LRU 只淘汰未使用项，选中和当前 View owner 固定。超限明确显示。缓存不写入 View/preferences，现有持久化版本不变。

View 加载结果绑定 Profile 及其 generation，新 Profile 的 View 到达前不使用旧引用加载目录或渲染业务组件。保存结果仅在仍处于同一 View 且没有后续编辑时替换草稿；同一 View 的后续编辑只推进已确认 revision，保留未保存内容。
