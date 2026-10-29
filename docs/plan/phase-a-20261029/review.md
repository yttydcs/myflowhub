# A 阶段验收与证据映射

## 2026-10-29 阶段性收尾结论（优先于下面历史状态）

用户在本会话明确表示“好的 $m-archive 接受风险”，接受此前列出的残余风险，并授权阶段性归档、本地合并与工作树清理。此处关闭的是本次实施工作流，A 产品正式验收未完成；A-AC-v1、P01—P33 及所有数值门槛保持。失败记录不改判 Passed。

- Requirements：**waived（仅归档门槛）**。AC01—AC21 整体仍 Pending；AC05/AC13 的 R2 计时与次数检查保留 Failed。A01/A02/A06 的已完成子范围保留；A03/A04/A05/A07/A08 的实施已交付，未完成验收转入[后续清单](follow-up.md)。AC20 的“全必验通过才完成”继续生效，当前不是正式 N 或合格发行。
- Standards：**passed（已实现范围的轻量复核）**。复用本记录中有版本标识的源码审查、Go/前端/生成物门禁；收尾复核有界队列、路由失败清理、单次 CAS 写入、Metrics 退出顺序与无 Permit 的配置恢复，未发现新增硬性规范违例。最终变化仅为下述展示简化及文档归档，未改权限、协议、恢复与写入合同。此结论不代替 Requirements，不宣称尚缺的原生或双机证据已通过。
- 2026-10-29 最后 UI 变更：删除资源总览副标题和搜索命中变量数，移除对应多余样式；筛选、订阅、实时订阅计数和无障碍行数保持。`npm test -- src/components/ResourceOverview.test.tsx` 3/3 通过，`npm run build` 通过（已有大 chunk 提示）。Desktop dist 已重建；尚未重建 Windows EXE/包。
- 历史 Windows 包 source_sha256=`e441947a6f1eb4a1075fe81938feb8c5b992d36c1e6ed60f5778fe76d13ea85d`，六个程序哈希见保留的证据。它对应 10 月 25 日程序与后续证据文档，**不包含 10 月 29 日 UI 简化**。不得把旧包或旧原生观察绑定为收尾源码的完整原生验收。
- 收尾格式检查：原有tracked差异检查通过；全部新文件暂存后的 `git diff --cached --check` 报告生成的Vite许可证空白行、Wails models.ts空白，以及3份原始导入文档尾部空行。为保留生成器产物与已确认输入未手工改写；这是格式提示，不作为行为测试通过。排除这5个来源明确文件后的同项检查通过。
- 固定比较基线与归档前 HEAD：`b1fcd1ac0319ebb349c8289cc6f61ed2d7302dc8`。本地提交将把实际候选固定到 Git；产品源文件与归档计划的内容标识见 [candidate-identity.json](candidate-identity.json)，不会把报告自身纳入递归哈希。此前包/集群/中间候选各自身份保留，不混合放行。
- 证据保存：[evidence](evidence/README.md) 保留可复核摘要；原 ignored out、辅助程序与原始本机过程记录已复制至 `D:/project/MyFlowHub3/archives/phase-a-20261029`，ZIP CRC 检查通过。私有夹具身份与配置只留本机、不提交 Git。下文历史 `out/phase-a/...` 可从该目录 `workflow-local-evidence.zip` 的同路径读取；C 盘原始路径另有本机私有 ZIP 副本。


日期：2026-10-24；需求版本 A-AC-v1；候选：feat/phase-a-foundation 未提交工作树（base b1fcd1a，非冻结 N）。状态：本机实现与验证收尾，正式产品验收仍 Pending。下面 AC 表保留整条产品合同，局部通过证据单列，不能推导整条已通过。

本文件是唯一 task/AC/候选证据映射。精确产品合同见[A 定稿](../../requirements/phase-a-acceptance.md)，其来源为[P20—P33](../../intake/2026-09-06_product-definition-grill.md)。每行包含原条款的全部前提/禁止行为；摘要不覆盖或弱化源文。变更记录需引用AC及候选commit/hash。

| AC ID | Exact requirement / source | Tasks | Verification / evidence | Status |
| --- | --- | --- | --- | --- |
| AC01 | P20/P21/P23/P25：两真实Win11 x64、LAN TCP；A=Hub+Desktop，B=Metrics，干净环境，独立节点 | A01/A07/A08 | 硬件/构建/架构/网络/依赖及节点身份记录、拓扑、从包部署步骤 | Pending |
| AC02 | P22：首次部署/准入/配权可CLI；日常发现/查看/控制全走UI | A03/A07/A08 | UI操作记录及命令边界，不能用SDK操作替代日常UI证明 | Pending |
| AC03 | P24/P25：独立包+完整部署说明，无开发工具链；运行依赖可检查/安装，版本/hash/配置示例；安装器/自动更新后置 | A01/A07/A08 | 干净Win11 x64运行、SHA-256/manifest/源码commit、Hub必要admin、独立包和依赖检查 | Pending |
| AC04 | P26/P27：真实B主音量20%/40%及初值，误差≤1个百分点；mute开/关及恢复，布尔精确；ack/回读/UI一致 | A04/A08 | Windows独立实际读取、owner确认、UI及restore原始值；无需播音，不拿模拟值代替 | Pending |
| AC05 | P27：采样1秒；有效提交至真实确认+成功UI+独立回读≤2秒；B本地改变至A订阅UI≤3秒；逐次串行、无竞争写 | A04/A05/A08 | 关联单调时钟事件与原始读回，参考负载下逐次报告，不用平均/百分位放行 | Pending |
| AC06 | P28：明确拒绝/未执行反馈原因并留草稿；可能已发且5秒无可靠终态=>结果不确定并结束loading；成功但回读失败单列 | A04/A08 | 拒绝/发送前失败/发送后丢ack/成功后读失败/超时晚响应注入，5秒不替代2秒也不承诺回滚 | Pending |
| AC07 | P28/P30/P31：无自动写重试、无离线排队补发或重启重放；再写先检查instance/revision/perms并由用户明确提交；同值不证明原请求成功 | A04/A05/A08 | 网络包与owner副作用计数、草稿代次/会话/实例晚结果隔离及显式再提交 | Pending |
| AC08 | P29：有效权威策略已提交+管理成功为撤权点；新write拒绝而仍授权read/sub保留；撤read/sub后新访问拒绝且不送点后保护值，订阅/UI≤3秒失效 | A05/A08 | 独立管理主体、有效权限核对、点前在途与点后请求/值分类、UI stale/禁用/布局草稿、重授权不自动提交 | Pending |
| AC09 | P30：两条路径各关闭/静默黑洞；中断生效起≤5秒识别+禁写；双向网络实际可达起≤10秒自动快照/订阅，不用重启/刷新 | A05/A08 | 注入器与独立TCP探测时间、受影响资源不误判整条连接、gap不装无损、现态恢复且无旧会话污染 | Pending |
| AC10 | P31：Hub/Desktop/Metrics分别正常/强制终止后手工启动，其余继续；进程启动起≤15秒含初始化，P30可达后10秒仍适用；同用户/状态/有效active Profile | A05/A07/A08 | 六种重启场景，身份/准入/有效授权/已保存配置/View保留、无重新准入；当前采集不回写旧设备值；正常退出未保存提示可取消；主动退出意图保留 | Pending |
| AC11 | P33第1组：固定同一修订两个逻辑消费者不同值，恰一成功/另冲突且业务仅执行一次；并发和替换各20轮零误写 | A04/A08 | 暂停无关采样的独立测试变量、SDK第二消费者；他方变值保留冲突草稿；纯时间/质量刷新不假冲突；同owner/name新实例隔离旧请求/草稿/响应 | Pending |
| AC12 | P32/P33第2组：单Desktop真实2000活动标准Variable订阅，各每秒更新一次≈2000更新/秒；全体可浏览查看按权限操作；含真实音量/静音，可标注负载提供者 | A02/A03/A06/A08 | 1998六类型示例+2真实；业务订阅/发布/缓存计数与不可见资源更新；只虚拟DOM不缩水订阅；记录owner/类型/字节/硬件而非每owner最大payload承诺 | Pending |
| AC13 | P33第2组：预热10分钟后健康持续2小时，至少100分散合法串行写，真实音频独立验证/恢复；原P27–P31时限不放宽 | A02/A03/A06/A08 | 连续时间线、参考负载下逐次结果；故障注入不混健康统计；合法写用最新修订且不隐式重试 | Pending |
| AC14 | P33第2组：零意外崩溃/死锁/身份或配置损坏/越权/跨实例误写，健康正常操作无未解释失败；每产品专属进程私有提交内存首尾10分钟中位数增量≤max(25%基线,64MiB) | A03/A05/A08 | PID/进程归属/原始private committed采样、10分钟预热除外，共享进程不重计；CPU/句柄诊断曲线无新增硬阈值 | Pending |
| AC15 | P33第2组：与健康2小时分开，每种P29/P30/P31已定义场景各5次全部达标；恢复后连接/订阅/待请求回预期基数，无持续累积 | A05/A08 | 分场景逐轮日志、5/3/10/15秒判定，预期基数含真实2000活动业务资源和明确管理余量 | Pending |
| AC16 | P33第3组：真实相邻N/N−1，同wire/本地状态主版本且当前Variable；Hub/Desktop/Metrics共8组合基本连通/准入/read/sub/一次授权控制；完整负载故障测N/N/N | A01/A07/A08 | 指定N−1=b1fcd1ac0319ebb349c8289cc6f61ed2d7302dc8，N候选冻结hash；000至111矩阵；不是旧退役协议或同包改版本号；不能安全兼容要阻断报告 | Pending |
| AC17 | P33第3组：Desktop-only/Metrics-only/共存；独立身份/状态/进程；手动升级/回滚保留N新保存状态与最新有效授权，不恢复过期备份复活撤权 | A03/A07/A08 | 独立停止/替包不伤另一产品、P31启动计时；N−1读取N保存配置/View/身份/策略；默认移除包留状态再部署恢复，显式purge只目标；本地删身份不代表网络撤准入 | Pending |
| AC18 | P33第4组：独立示例bool/number/enum/text/object/array六类型，至少bool/number可写，陌生命名空间与定义；只新增提供者，不改核心/消费者或安装专用schema/renderer | A06/A08 | 源码/构建依赖/运行资料+Win11 x64示例包；真实通用发现/read/sub/write/类型范围/CAS/陈旧不可用/错误演示 | Pending |
| AC19 | P33第4组：独立Hub含必要管理工具/Desktop/Metrics/示例，包版本/源码或完整指纹/SHA256/依赖/状态目录；完整部署、准入授权、保护备份、升级/回滚/移除资料 | A07/A08 | 从公开无密钥配置重现；敏感状态原保护方式保管，不公开身份私钥/Permit/token；开发示例工具链与用户包运行依赖分开 | Pending |
| AC20 | P33第4组：真实双设备报告/环境/包hash/逐项结果/原始计时内存及实际设备读回，截图不足；Passed/Failed/Unavailable真实，全必验通过才完成 | A01/A08 | 原始NDJSON/CSV/设备证据绑定候选；旧证据仅同输入且有复用依据；缺真实设备项不通过、正确性/安全/数据/时间问题不作非阻断豁免 | Pending |
| AC21 | P33第4组：受影响Go/前端/generated/build门禁及Clipboard/File回归；隔离配置、初音量/静音、临时服务规则恢复，脱敏证据保留 | A02/A03/A04/A05/A06/A07/A08 | 按改动风险脚本输出、生成差异检查、范围化回归记录、finally清理清单；不扩为Clipboard/File完整发行、不发布 | Pending |

## 原文定位

- [范围/主场景与部署](../../requirements/phase-a-acceptance.md#已确认范围与主场景)、[操作入口](../../requirements/phase-a-acceptance.md#已确认标准操作入口p22)、[独立包](../../requirements/phase-a-acceptance.md#已确认标准独立可运行包与完整部署说明p24)。
- [正常数值与时限](../../requirements/phase-a-acceptance.md#已确认标准取值容差与正常响应时间p27)、[失败语义](../../requirements/phase-a-acceptance.md#已确认标准写入失败等待上限与结果不确定p28)、[撤权](../../requirements/phase-a-acceptance.md#已确认标准撤权与重新授权p29)。
- [网络](../../requirements/phase-a-acceptance.md#已确认标准断网检测与自动重连p30)、[重启](../../requirements/phase-a-acceptance.md#已确认标准独立进程重启与已保存状态恢复p31)、[P33全部补充](../../requirements/phase-a-acceptance.md#补充验收标准定稿p33)。

## 证据登记和停止条件

每次登记：AC ID、Task、candidate commit/包SHA256、设备/日期、参数、实际结果及原始证据相对路径、Passed/Failed/Unavailable、复用依据。证据目录在实施时创建并在此链接；尚无文件不得提前伪造链接。规格/单元测试可证明局部行为，不能替代整条真实链路时限和硬件控制。

发生正确性/权限/状态损坏或任何已确认时限失败即记录失败并回到对应任务；修复后针对受影响证据重测，候选变化不默默复用旧报告。正常稳定性、20轮竞态、5轮故障与8版本矩阵各自保留分母与逐次结果。

## 原规划检查（2026-10-19，仅历史文档证据）

- 已通过（仅文档）：两检出共 759 个本地链接/锚点；10 个 Task 唯一归属（A01—A08 批准后执行、D01/D02 本轮不执行）；21 个 AC 全部映射且保持 Pending，无孤立实施任务。依赖无环，先合同/容量，再消费者/控制/恢复/交付与最终证据。
- 已通过（仅文件保护）：22 份 governed 文档输入含 SHA-256 快照；主检出 1339 个范围外文件 hash 保持；P01—P19 原文及 Flow 已确认定义逐字保留。两检出 git diff --check 通过，HEAD 与 Git 暂存区未改；新增专用工作树/分支。核查结果保存在项目 `.tmp/phase-a-planning-20261019/verification.json`。
- 规划时尚未运行产品测试；当前实施结果见下节。
- 此后用户已批准 A01—A08 实施。当前尚未提交、合并、发布或清理工作树。

## 本机实施与局部证据（2026-10-24）

用户要求先在本机实现并运行多个用例，机器后续处理。A-AC-v1 和 P01—P33 保持。工作树中的源文件、生成物和二进制均为本地候选；包 manifest 用完整 source_sha256 标识未提交内容，不能把 base_commit 当作正式 N。以下忽略目录中的证据为本会话原始输出，未公开私钥、Permit 或业务状态；私有过程日志留在本次 C 盘临时目录。

| 范围 / AC | 本机结果 | 证据和限制 |
| --- | --- | --- |
| A01 / AC16 | Passed（基线） | out/phase-a/evidence/a01-baseline.json、baseline-source-verification.json；真实 b1fcd1a 的245份Go/模块文件逐项匹配Git原文，基线独立构建；非伪版本号 |
| A02 / AC12 | Passed（本机） | 全量Go、容量测试及TCP测试：分页授权/代际/边界、注册原子性、有界帧和字节队列、2000实际订阅；保留旧小目录读取及View v3/64控件 |
| A03 / AC02/12 | Passed（局部） | Desktop前端30文件505用例；overview-2000-filter-write.png、overview-2000-dark-narrow.png为真实App挂模拟API的浏览器预览：2000活动句柄、仅27—32行DOM、筛选/滚动不退订、Inspector编辑；不是原生IPC或真实设备 |
| A04 / AC06/07/11 | Passed（局部） | write-outcomes-final.log及前端全量；覆盖5秒晚ack、不确定不重试、可靠成功后回读失败/显式刷新、草稿保留；phase_a_failures_test.go：CAS和实例替换各20轮，恰一次副作用 |
| A05 / AC08/09/15 | Passed（本机） | 两条真实TCP路径 × 关闭/静默黑洞 × 各5轮，共20次；静默检测已测约3—4.5秒，恢复快照在预算内；撤write保留read/sub，撤read/sub停止新保护值。完整原生UI及正式负载时限另测 |
| A05 / 策略清理 | Passed（本机） | go-race-final.log；权限提交回调只通知单个有界合并worker，清理不持策略提交锁；重入读取策略的provider Abort回归通过；旧link清理不误伤替代连接 |
| A06 / AC12/13/18 | Passed（60秒局部负载） | local-reference-60s.log及go-full-final.log：1998六类型+2模拟控制、2000实际Desktop SDK订阅，各1Hz，60轮/119880示例采样、100分散写；续租后consumer待请求2000，关闭后订阅/待请求0。单测试进程、模拟控制，无Wails/真实音频/私有提交测量；不代替2小时 |
| A07 / AC16/17 | Passed（后端子集） | compatibility-bridge-matrix.json：000—111共8组合，真实Hub/Metrics控制台进程+各版本Desktop Go绑定适配器，read/sub/配置控制；真实管理CLI创建/revoke权限；旧Hub从N最新状态读取已撤权限，写被拒绝而read/sub保留。未验证原生UI/View/音频，不宣称AC16/17整体Passed |
| AC21 / Go及回归 | Passed（本机） | go-full-final.log、go-vet-final.log、go-race.log、go-race-final.log、phase-a-final-tcp-race.log、check-core.log；全量含Clipboard及File受影响包/集成。检查范围不扩为其完整发行 |
| AC21 / 前端与构建 | Passed（本机） | Desktop 505、Metrics 14；Go核心及两个Wails windows/amd64生产构建；统一生成物再生成一致性和包检查记录另见verification-summary.json |
| AC02/10 / 原生窗口完整交互 | Unavailable（本会话） | native-ui-availability.json；Wails加载器清除WebView2调试环境，未开放CDP；computer-use所需node_repl工具不可用。未执行原生UI产品断言，未修改生产安全设置绕过，也未以启动成功/浏览器预览代替 |

## 尚待完成的正式验收

- 两台真实 Win11 x64、独立音频读回/恢复、2秒/3秒真实链路、原生界面日常操作及未保存正常退出确认。
- 全部进程正常/强制终止、独立启动及同状态恢复的完整5轮矩阵和15秒界面时限；保存View/Profile/主动退出意图/移除后再部署。
- 10分钟预热后2小时健康2000@1Hz、至少100次分散串行合法写入（可用测试变量），真实音量/静音独立覆盖并恢复初值；产品及专属子进程private committed首尾中位数、完整负载下各类权限/网络/重启故障各5轮及资源基数。
- 冻结正式N完整commit和对应包SHA；干净无工具链环境从包部署；原生八版本组合及N新状态的完整生命周期验证。

要求轴：未收敛，21项整体验收仍Pending；工程规范轴：已记录的本机门禁通过，尚缺原生/物理/正式耐久证据。未创建docs/change完成记录，未合并/推送/发布。设备后续可用时从此清单继续，无需重新讨论已接受标准。

## A08 集群补充验证（2026-10-24）

以下首轮记录保留历史时点。后续继续迭代已启动有限长测，不再适用末尾“无后台长测”的旧状态：Job reference-endurance-20261024，UID 237e160f-bf3f-472a-bd19-3b1b280cfe11；600秒预热后7200秒健康，100次分散串行写入，activeDeadlineSeconds=8100/backoffLimit=0。源SHA256=3284369ab28b267a09ace263c3583def8ef0d77dd780a0893d728471ac5c2b91，Linux测试二进制SHA256=0767c01913dff6707e735b878472aa82e47e4e85788af7a76c6a636bf0724c10；image/endurance-source-files及部署/监测记录均位于out/phase-a/evidence。cgroup每5秒采样，仅作Linux诊断，不能代替Windows private committed。后续握手/音频/界面修复单独验证并记录对应指纹，不将运行中旧二进制冒充最新候选。

原生窗口操作已可用，替代下文Unavailable环境事实但不自动放行原验收：native-overview-2000-audio.jpg记录Wails原生窗口2000/2000订阅、真实40%/未静音；native-old-instance-draft-blocked.jpg记录Metrics重启后旧草稿阻断；native-unsaved-prompt.jpg及操作记录证明取消退出保留草稿。两次原生音量提交均明确CAS拒绝，未记正常控制通过，发现写前版本过期后进入A04修复。新写前读取21项针对性测试通过；协议分段写与join就绪竞态修复分别已有Go/race证据，最新全量Go/vet通过，前端全量及新版原生控制复测仍在进行。工程规范轴和要求轴的最终复核待本次证据收束，21项整体验收继续Pending。

用户提供专用k3s-test，已通过ESXi固定证书和原SSH主机公钥完成接入。现场为单节点8 vCPU/24 GiB、Ready；保留既有开发服务。仅创建独立myflowhub-phase-a namespace及有界测试资源，未改变现有路由、防火墙、默认kubeconfig或系统组件。私钥始终留在本机。

| 检查 | 状态 | 候选与证据 | 限制 / 下一步 |
| --- | --- | --- | --- |
| 集群接入与隔离 | Passed | out/phase-a/evidence/k3s-cluster-readonly.json、k3s-smoke-start.json；请求1CPU/1GiB，上限2CPU/2GiB，非root、无token/host网络/hostPath，deny-all网络隔离 | 原Metrics API不可用，未修改；改用host cgroup v2诊断 |
| Linux参考负载 | Passed（60秒子项） | k3s-reference-60s.log、k3s-validation-summary.json；2000活动SDK订阅1Hz，60轮/119880示例采样，100写，读—写—回读最慢482.659ms，关闭后订阅/待请求0；Pod退出0、无重启，Job Complete | 单容器多节点真实TCP、模拟控制；不等于独立产品进程/原生界面/真实音频 |
| 内存观察 | Passed（仅采集） | k3s-smoke-running.json；约429MiB当前/496MiB峰值，OOM0 | 单个运行中采样；不作2小时增长或Windows私有提交判定 |
| 原生UI、正式耐久与双机设备 | Pending | 保留上述AC01—AC21及本机已有UI预览证据 | 未因新增集群替代原验收边界，不允许归档为A完成 |

被测Linux测试二进制SHA-256为63a00dab5d72fdfd7223f8e44fd1993f896dced768c1dd778ad4e1ed60f92af5，来自既有预览源指纹4df09ba35f6401dd3a6715b2e7f34ae1857c69f5290c034c9cd12fb3ce2883ae；本节及plan/todo仅追加进展，不改变已测代码或已接受合同。此前包和报告指纹保留其测量时点，不将新文档内容冒充原归档源码。

首轮集群子项工程检查通过；整体要求轴仍未收敛。首轮Job已结束；之后启动的有限长测状态见下节。namespace、配额/隔离策略和测试镜像保留供后续测试，已完成Job的TTL为24小时。完整记录：out/phase-a/K3S-VALIDATION.md。未提交、合并、发布或创建完成归档。

## 最新继续轮：缓存优化与验证边界（2026-10-24）

本节覆盖前文同名检查的历史状态；历史失败与旧候选报告保留。映射A03/A04/A07/A08，接受标准、21项完整AC及D01/D02边界不变。

- 原生batch256仍出现60—100秒采样滞后和CAS拒绝，单纯扩大事件批次没有证明解决问题。原生Go profile显示VariableEventDecoder累计CPU热点及重复schema解码分配；SDK改为类型化definition缓存，返回深拷贝，绑定层独立校验保留。
- 相同对象事件微基准从443336ns/op、36321B/op、623allocs/op降至161330ns/op、11385B/op、214allocs/op。证据为decoder-benchmark-before.log/decoder-benchmark-after.log；只证明该解码用例，不能外推原生时限。
- 缓存被调用者修改的测试同时覆盖快照/数据返回对象、嵌套数值指针/枚举/属性名，以及非法值不得绕过校验或推进修订。protocol、resource、SDK和Desktop绑定race通过（decoder-cache-race.log）。Go全量及vet再次通过（go-full-decoder-cache.log、go-vet-decoder-cache.log）。
- Desktop前端首次复测509/510通过，ResourceOverview的2000资源用例达到既有5秒测试超时；当时存在并行Go/原生进程负载。保留frontend-decoder-cache-final.log，隔离负载后降低测试worker复核，不放宽750ms排序门槛或5秒用例超时。
- 12:27启动缓存修复版原生Desktop并选中Metrics节点；UIA点击超时，随后出现helper active request；重置后窗口截图与目标不一致，立即停止输入。因此本轮没有新的真实音频提交或原生写入成功证据，不能覆盖前面的CAS失败。仅本次拥有的本机服务/GUI/音频观察器已停止；独立恢复测试确认原音量40.0000095%与未静音精确恢复（native-audio-restored-v4-fixed.log）。首次恢复命令参数解析失败记录也保留，不作为产品缺陷。
- 集群12:26 HKT状态查询在管理TLS建立阶段超时（k3s-phase-a-endurance-status.json）；用户正在检查连接。最后可确认的长测进度是健康2342秒、33写、2000订阅、superseded=0，不能把推算结束时间或后来失联当作通过。任务UID、原候选指纹保持，未停止或重新启动集群Job；后续代码变化不能由旧Job代验。

当前要求轴未收敛：原生全链路、正式双机/耐久/生命周期及冻结N仍Pending。规范轴已通过的代码验证保持对应范围，UI自动化与集群结果采集受阻；新的包与构建验证见后续本节结果。不能进入归档、发布或把旧包标成最新。

本轮后续结果：

| 检查 | 结果 | 证据与范围 |
| --- | --- | --- |
| 新缓存候选2000@1Hz参考负载 | Passed（本机60秒） | reference-decoder-cache-60s.log：2秒预热、健康60.57秒、100写、最慢124.1183ms、关闭后活动订阅/待请求0；模拟控制/SDK，无原生UI或正式内存证据 |
| 8组合及最新撤权状态回滚 | Passed（后端子集） | compatibility-decoder-cache.log、compatibility-bridge-matrix.json；000—111均通过，回滚后控制被拒绝且读/订阅保留；旧报告已单独备份 |
| 产品构建 | Passed | build-current-candidate.json及构建命令记录；core、Desktop、Metrics的windows/amd64生产构建，未开启Devtools |
| 生成文件一致性 | Passed | generated-current-candidate.json；统一generate命令重生成15文件，与本工作树预期内容逐字节一致，不以旧HEAD无差异代替新合同检查 |
| 前端超时复核 | Passed（串行全量） | 两轮并行均509/510通过，失败日志保留；全部构建/负载停止后，单独ResourceOverview在2.878秒内通过。随后1-worker全量30文件、510/510通过，总览3.308秒（frontend-serial-final.log）；未改代码、断言或5秒时限 |
| 已确认结论保护 | Passed | accepted-requirements-unchanged.json；A验收标准与产品定义均与先前候选SHA256一致；未修改主检出、HEAD或暂存区 |

12:38 HKT集群重查可建立ESXi管理连接，但中转Windows到本机10.3.3.2的回连仍在SSH认证前超时（最新k3s-phase-a-endurance-status.json）。没有复制私钥或修改路由/防火墙。继续等待网络恢复后取回原Job结果。

### 当前候选轻量复核

| 维度 | 结论 | 边界 |
| --- | --- | --- |
| 需求覆盖、任务归属 | Pending | 原标准文件hash保持，改动归属A02/A03/A04/A07/A08；完整21项AC未放行，不添加产品范围 |
| 架构、可读性及一致性 | Passed（代码范围） | definition深拷贝移至协议类型所属层供resource/SDK复用；不新增依赖、跨层反向调用或wire版本；原生退出映射采用Wails的实际返回值 |
| 性能、重复计算及配置边界 | 局部Passed / 原生Pending | 同一定义不重复JSON解码，目录排序复用collator；保留4096/256/字节预算。微基准、60秒TCP通过，未从缓存优化推导端到端时限 |
| 稳定性、安全与权限 | Passed（已测代码范围） | 返回schema与缓存不共享可变字段，非法值不推进修订；保留绑定层输入校验、精确CAS、5秒终态及不重试；race与授权/回滚测试通过 |
| 可用性、用户路径与整体联调 | Blocked | 原生自动化目标不可靠后停止输入；真实控制、退出确认、全负载时效仍无新版通过证据。旧截图仅证明其原时点 |
| 测试覆盖与证据版本 | Passed（已测范围） | 新候选Go全量/vet/race、前端510项串行全量、参考负载和兼容矩阵通过；前端并行失败及隔离复测分别保留，不用旧包证明新代码 |
| 单代理治理 | Passed | 本轮未委派，所有改动在既有工作树；主检出用户修改未触碰，HEAD/暂存区未改。源文件清单由预览包manifest完整列出 |

回滚边界：保留旧预览包、失败日志及基线；后端已验证N−1读取最新撤权状态。不得用旧身份/授权备份恢复被撤权限，不以回滚声称解决尚未验证的原生时效。当前不能进入m-archive；无需修改或重新确认A-AC-v1。

交付记录入口：out/phase-a/evidence/current-package-verification.json保存本轮预览包的绝对路径、完整source_sha256、源文件数及ZIP/EXE哈希检查结果；manifest的acceptance_status必须保持pending。包位于C盘隔离目录，避免继续挤占D盘；验证结果文件产生前不得推定打包通过。包与源码均不代表正式冻结N，不发布、不推送、不归档。

### 集群连接恢复后的结果

原长测已于2026-10-24 12:55 HKT结束，随后通过固定主机公钥的SSH取回完整证据；前面的“失联/结果待取回”保留为历史事件。Job UID仍为237e160f-bf3f-472a-bd19-3b1b280cfe11，无重建、无重启、无故障注入。

| 检查 | 结果 | 证据与边界 |
| --- | --- | --- |
| 旧候选Linux健康长测 | Passed | k3s-endurance-2h.log：600秒预热+7200.214秒健康、2000订阅/1Hz、100写、最慢199.727ms、关闭后活动订阅和待请求0，superseded=0 |
| Linux内存诊断 | Passed（非Windows口径） | 1562条采样，首尾健康10分钟各120条，memory.current中位数573188096→571555840 bytes，下降1632256 bytes；无采集错误/容器重启；k3s-endurance-independent-check.json独立复算一致 |
| 证据版本 | 明确限定 | 长测源SHA仍为3284369ab28b267a09ace263c3583def8ef0d77dd780a0893d728471ac5c2b91，测试二进制0767c01913dff6707e735b878472aa82e47e4e85788af7a76c6a636bf0724c10；不含后来缓存/界面修复，不代验新候选2小时 |
| 新缓存候选集群补测 | Passed（SDK子集） | candidate-checks-157803057b15，UID6885512c-f863-4f43-8724-bc434ff3b370；源SHA157803057b1533f60e945aa22daafb068e04aa5aa25bbb90d6d18b4dbba11043、Linux二进制f04c60fbc597fcb26af330951987f7dbf2d745e2cc7f8ab7fd9f9d8f42b84554；3个顺序阶段均退出0、无重启，13:05 HKT结束 |
| 新候选60秒参考负载 | Passed | k3s-candidate-reference.log：2000@1Hz、100写，最慢202.537ms，关闭后活动订阅和待请求0；与旧候选2小时分开，不宣称新候选耐久已过 |
| 新候选故障、竞态与撤权 | Passed（小型TCP夹具） | k3s-candidate-validation.json与三个原始日志：2路径×关闭/黑洞×5轮=20次，最慢识别4.49775秒/恢复0.10286秒；CAS和实例替换各20轮，撤权5轮。故障阶段使用独立小夹具，不等同2000负载/原生UI/跨物理机故障验收 |

原始JSON完整保存在k3s-phase-a-endurance-status.json；提取日志、采样和汇总另存，不覆盖原记录。新候选部署已返回真实Job UID，管理通道随后读取收尾记录失败；后续独立查询已确认同UID运行，未重复创建。测试仍局限于专用namespace，1CPU/1GiB请求、2CPU/2GiB上限、无特权/宿主网络/凭据转存。

本轮终态：本机和集群可运行的已述检查完成，两个有限集群Job及长测观察器均已退出；保留namespace/镜像与TTL24小时的完成Job供复核。原生自动化仍受阻，最终正式验收仍Blocked，不进入归档。后续需要恢复原生操作能力验证采样时效、真实音频写入和退出确认，再按用户后续提供的两台Windows设备完成正式耐久/生命周期。新交付包相对157803057b15仅更新本次结果文档；current-delivery-source-equivalence.json必须证明所有非Markdown文件相同，否则不能复用该候选测试证据。

### 原生自动化恢复与满负载补测（2026-10-24 后续轮次，进行中）

用户要求优先解决原生UI自动化并继续全部可在本机完成的验收，双机后置。沿用A03/A04/A05/A08，无新产品范围、无子代理、无发布。上面的自动化Blocked是历史时点；本轮已恢复可重复操作，不再作为当前阻塞理由。

| 检查 | 状态 | 证据与边界 |
| --- | --- | --- |
| 原生自动化目标恢复 | Passed（操作路径） | 重新初始化工具、从list_apps选择唯一返回窗口、单独激活、截图坐标逐步操作；每次动作后重新截图。UIA索引输入的超时原因未证实，不宣称修复工具内部缺陷 |
| 新候选低负载真实控制 | Passed（功能）/时限部分Pending | native-ui-actions.json、native-ui-resume-results.json；20%/40%及静音开/关均由原生控件发起，独立CoreAudio首次确认相对输入调用前时刻为0.905—1.079秒；截图调用含工具开销，不把2.75—3.34秒直接当产品延迟，也不据此放行完整2秒UI标准 |
| 2000真实订阅 | Passed（建立）/控制Failed | native-ui-overview-current-2000.jpg显示2000/2000，1998示例与音量/静音2项；native-ui-volume20-2000.jpg记录写入被拒绝：expected revision 667/current 668。无实际20%变更，草稿保留；不自动重试，不修改CAS条件 |
| 原生未保存退出 | Passed（单轮功能） | native-ui-unsaved-close.jpg、native-ui-unsaved-cancelled.jpg；选择“否”保留草稿，随后选择“是”正常退出。不能将Escape键未关闭Yes/No对话框记作取消通过 |
| 保存边界恢复 | Passed（单轮功能）/矩阵Pending | native-ui-restored-saved-view.jpg；同源性能采样构建恢复已保存2控件View和自动连接，实际值40%、静音关闭；未提交20%草稿未补发。尚非正式生产版5轮/15秒矩阵 |
| 性能调查 | 已记录 | native-current-cpu.pprof、native-current-allocs.pprof及对应top.txt；生产同源测试入口仅采样Go，无调试端口或界面旁路。频繁poll反复编码全部句柄，16ms合批修复归属A03，原生效果待复测 |

本轮音频原值在新的独立观察器目录保存，不复用早先基线。修复构建前已通过恢复模式精确恢复level=0.4000001、muted=false；后续操作必须继续保留该恢复边界。完整AC仍Pending，最终结果须追加至此处，不能用本段进行中记录宣称收束。

#### 后续修复循环与尚未通过的检查

- 浏览器16ms合批：12项针对性测试及生产构建通过；实际2000负载20%写成功，输入调用前到截图完成上界1.916秒，但其后采样年龄约23秒并增长，保留native-ui-volume20-coalesced.jpg/native-ui-coalesced-browser-stale.jpg，不作整组通过。
- 合批迁至Go后端，前端不依赖浏览器timer：后端合批/取消及前端无timer连续消费测试通过，生产构建通过；2000/2000下采样年龄11.786秒→17.451秒（约47.6秒观察间隔），仍Failed。记录于native-ui-iterations.json及native-ui-server-coalesced-stale.jpg。排除了“只调整合批等待位置即可消除积压”的假设，仍需解码路径验证。
- SDK单次缓存定义校验：初始坏包测试发现尾随空白超预算未被自定义UnmarshalJSON计入，已补原始载荷长度检查，失败不隐去。修正后SDK/Desktop完整单元测试、相关SDK/Desktop race通过；原绑定/协议测试通过；前端12项不受该后端变更影响。微基准75744ns/op、8607B/op、171allocs/op。新生产构建及原生复测进行中，不能复用前一EXE的UI结论。

当前签名：A03/A04/AC04/AC12/AC13，2000@1Hz原生订阅采样积压；UI自动化前提已恢复，完整时效尚未改善。尚未开始健康两小时或完整五轮原生故障矩阵，避免用已知不健康基线跑长测。所有音频变更均先按本轮独立基线恢复，native-audio-restored-after-browser-coalescing.log记录精确恢复成功。

#### 原生消费瓶颈定位与结构化桥修复（进行中）

- 单次SDK校验生产版D9012135…复测仍Failed：2000/2000下截图观测年龄5.509秒→20.284秒→40.137秒，记录native-ui-iterations.json；优化微基准不等于端到端改善。
- 同源原生测试入口新增被动poll统计，只记录批次条数/大小、开始结束时间和末条时间戳，不代替UI输入。main的窗口创建提取为runDesktop，生产与测试共享同一入口。native-singleparse-poll-trace.jsonl及pprof保留：59.969秒消费109568条（约1827条/秒），后端取批中位8ms，批次间隔中位127ms，末条年龄5.903→12.820秒。新的证据将主要瓶颈缩小到桥接/界面消费阶段，不能归咎于自动化失效。
- 512条/256KiB诊断轮：Go及前端针对性测试、构建通过；59.899秒消费115200条（约1923条/秒），取批中位13ms、间隔235ms，仍积压；native-batch512-poll-summary.json、原始trace与截图保留。仅提高条数没有解决问题，正式代码恢复256条预算。
- 下一轮A03修复：新增结构化PollSubscriptionsBatch，旧JSON/单订阅接口保留，网络严格校验和字节界限不变；消除原生桥重复字符串包装及前端重复JSON解析。新增结构化/旧接口编码等价、非法批次回归，生成Wails绑定。当前验证进行中，不作通过宣称。

#### 结构化桥结果与原生控制复测（2026-10-25，继续进行中）

结构化桥诊断轮在2000真实订阅下60秒消费122576条，采样年龄中位441ms、最大1098ms；原始trace和native-structured-poll-summary.json保留。生产候选2955008C…再次确认2000订阅和575ms采样年龄。全Go测试/vet、前端518项、三产品构建及8种后端N/N−1组合通过；这些不是完整原生版本组合验收。

该生产候选20%写成功，但40%仍发生expected406/current407的CAS拒绝；据此将预读比较和唯一一次条件写入放在同一原生调用，消除两次WebView往返之间的版本老化窗口。仍严格检查实例、定义、基值、修订及5秒总预算，不自动重试；成功后前端独立回读保留。正负零沿用旧路径以避免JSON序列化丢失符号。新增读写阶段/冲突/超时/晚到回复等回归，299项相关前端测试、生产构建、Desktop完整Go测试及相关race通过。一次Go构建因同时运行Wails清理临时syso失败；串行重跑通过，原失败日志保留。

2026-10-24用户Escape中断后停止UI工具。2026-10-25用户再次要求继续，恢复隔离测试服务，未修改验收要求。当前Desktop SHA256为4661987BF9EEC4FCDB01EACDFEF9E8AFB71EA7D87B2CCA115BA585DD0FCCD98E；其他进程版本见native-oct25-processes.json。原生UI建立1998示例+2真实音频变量，筛选音频后仍为2000/2000。

| 操作 | 独立CoreAudio首次确认 | UI确认截图耗时上界 | 结果 |
| --- | --- | --- | --- |
| 音量20% | 763.405ms | 1260.748ms | Passed（本机当前候选单轮） |
| 音量40% | 200.056ms | 417.418ms | Passed（同上） |
| 静音开启 | 196.815ms | 293.011ms | Passed（同上） |
| 静音关闭 | 191.454ms | 268.423ms | Passed（同上） |
| 系统侧改为20%→订阅UI | 实际CoreAudio操作确认 | 601.989ms | Passed（本机单轮，3秒门槛） |

证据：native-oct25-actions.json、native-oct25-audio-results.json及对应截图；独立观察器路径记录于进程映射。初次PowerShell汇总脚本未匹配样本，后改用独立Python按时间戳和值复核原始JSONL通过，并非产品写入失败。测试后精确恢复本轮baseline level=0.4000001、muted=false，native-oct25-audio-restored.log通过。尚未执行当前候选健康两小时及全部五轮原生故障/生命周期；完整AC仍Pending，不能将这组单轮通过当作完整验收。

#### 原生故障矩阵与续租失效缺口（2026-10-25）

- 4661987B…候选：Metrics及Hub强制退出/重启各5轮通过本机原生状态检查，native-oct25-process-faults.jsonl记录逐次起点和截图上界。Metrics识别439—701ms、恢复1126—8805ms；Hub识别494—757ms、恢复2735—7347ms。每轮恢复2000/2000。第一轮Metrics保留未提交静音草稿，恢复后旧实例草稿明确失效，实际静音保持关闭，无自动补发。此处不是正常退出、双机或全部生命周期矩阵。
- 首轮在线撤权：write撤销后新UI写被服务端拒绝（469.286ms），read/subscribe仍运行；继续收回read/subscribe后491ms内失效，UI新读取被拒绝，草稿保留。恢复策略不会自动提交；正常退出提示、取消、再确认退出均有截图。重新启动恢复已保存两控件View和40%/关闭，截图确认上界12052ms。其余撤权重复轮次待补。
- 总览展示缺陷：写入拒绝时live仍为true，但行文案仅因blocked/error显示“不可用”。修复为实时数据与操作异常分开表达；新组件用例验证拒写后仍接收新值、随后撤订阅才失效，连同2000虚拟化/独立订阅共3项通过。生产D751930E…构建成功。
- 本机TCP转发器仅监听loopback、转发不记录opaque payload。两条路径可分别明确关闭或暂停双向转发（静默链路不可用）；不改变系统防火墙。首次恢复因控制文件替换窗口导致测试器FileNotFound退出，保留stderr与events，不把该次恢复归责产品。随后使用原子替换并处理初始控制文件缺席，重启相同端口，两条normal控制确认通过。
- D751930E…：Metrics静默中断5轮成功，全部识别截图上界4275—4498ms，恢复2926—3454ms，恢复2000/2000。证据native-oct25-relay-actions.jsonl、native-oct25-captures.json及截图。首次Desktop静默中断失败：约4.16秒时仍281个变量live，17.60秒后仍未失效；连接恢复约4.02秒重新2000/2000。不能用音频单项及时失效替全部2000项放行。
- 根因修复A05：续租取消旧句柄后，等待父连接或重建订阅失败的路径没有发gap，旧快照持续显示实时。新增durable_renewal_gap_test.go，先证明两路径在旧代码失败，再在等待连接/重建失败时统一、仅一次失效。SDK/绑定完整测试及durable race通过，生产原生复测进行中；原失败及测试夹具初始配置错误均保留日志。当前未启动健康两小时，不把已知失败候选用于长测。

#### 续租修复版原生重复矩阵（2026-10-25，后续状态）

Desktop SHA-256 `20026DDE5D27371AB80150295114CFEC2C3FABC6E6C519C8FC74FF65F96B5BD9`。满负载仍为1998示例与2真实音频订阅。下列时长均从测试器确认故障/恢复生效或管理操作成功起算，到原生截图完成的保守上界；不是应用内部精确耗时。

| 本机原生检查 | 次数 | 识别/失效上界 | 恢复上界 | 结果 |
| --- | --- | --- | --- | --- |
| Desktop—Hub 静默中断 | 5个有效计时轮次 | 4470—4755ms | 3626—4211ms | 2000项全部失效并自动恢复 |
| Desktop—Hub 明确断连 | 5 | 1130—1357ms | 3618—3657ms | 同上 |
| Metrics—Hub 静默中断 | 5 | 4517—4808ms | 3707—4040ms | 仅音频2项失效，其余1998继续更新 |
| Metrics—Hub 明确断连 | 5 | 1166—1243ms | 3606—3676ms | 同上 |
| write撤销→拒写，read/subscribe撤销→失效，重新授权→手动刷新 | 5 | 418—671ms；拒写273—921ms | 用户手动刷新恢复2000项 | 草稿保留、没有自动提交，新读取均Forbidden |

原始证据在`out/phase-a/evidence/native-oct25-relay-actions.jsonl`、`native-oct25-policy-events.jsonl`、`native-oct25-actions.json`、`native-oct25-captures.json`及同前缀截图；逐轮可重算结果为`native-oct25-network-revocation-results.json`。Desktop静默第二轮截图为5269ms，虽已全部失效，无法证明5秒，保留并标记计时证据不足，补第六轮而不删除失败观察。控制文件原子替换另遇Windows短暂共享冲突，测试器增加有界2秒重试，未执行的normal请求不作为恢复起点。

撤权第一轮另外在收回权限后从系统端修改真实音量到20%，UI继续保留已标失效的40%；授权恢复并手动刷新后才获得20%，随后精确恢复原始音量。该新值保护附加检查只执行一轮，不能记作五轮全覆盖；新订阅拒绝由集成回归覆盖，原生五轮覆盖新读取拒绝及已有订阅失效。音频观察器独立记录实际静音值。

SDK/绑定全量、durable race、网络四场景各5轮、撤权和CAS/实例替换各20轮集成回归通过（`native-oct25-post-renewal-integration.log`）。续租281项持续live的原生问题在当前复测未再出现。TCP转发器暂停字节用于模拟静默链路不可用，不能替代双机物理丢包证据；正式双机、完整生命周期/失败矩阵、原生健康两小时及private committed内存仍Pending。没有发布、冻结N或执行D02。

#### 正常退出与Windows原生长测准备（2026-10-25）

Hub与Metrics控制台交付程序各正常退出/重启5轮。测试器先核对映射PID/可执行路径及目标独占控制台，只有控制台成员恰好为目标和信号助手时才发送CTRL_C_EVENT；全部退出码0，未用强杀充当正常退出。同一状态、身份和权限启动后，原生2000项恢复上界Metrics3136—4292ms、Hub3558—3737ms，满足本机15秒恢复检查。证据`native-oct25-normal-restarts.jsonl`、`native-oct25-normal-restart-results.json`及对应截图；不替代Metrics GUI关闭行为或Desktop生命周期矩阵。

当前源码全Go测试、Desktop前端30文件527项通过，日志`native-oct25-all-go-final.log`、`native-oct25-frontend-final.log`。本轮npm未将maxWorkers参数转交vitest，日志已明确警告；此次实际为默认并行并成功，不描述为串行验证。

原生长测在Unix毫秒1792904812612开始预热600秒，随后计划健康7200秒。状态/辅助进程路径见`native-oct25-endurance.json`；独立证据存C盘隔离TEMP，避免D盘空间不足。System.Diagnostics.Process.PrivateMemorySize64按产品根进程与验证创建时间的专属子进程归属，Desktop包含WebView2，Hub/Metrics包含独占conhost；保存每个PID、创建时间、CPU/句柄和提交量。独立只读客户端每100ms读取示例布尔变量，原生工作区控件发起写入；预热写入不计100次健康窗口写入。当前为进行中，不能提前判定两小时或内存通过。

#### Metrics 图形版保存与退出补齐（2026-10-25，继续进行中）

A05核对发现原Metrics图形版只保存主题，无法以已保存连接自动重开；控制台的正常重启证据不能替代此行为。已补独立GUI连接配置、active/inactive意图、恢复错误展示及未保存连接/策略退出确认。Permit不落入GUI配置，缺失身份不重建，持久化失败停止新节点；原产品状态格式不变。针对性后端、race及Metrics前端17项通过，生产构建通过，当前GUI SHA256为`1702734433DD1BDD60729B3F678CD7901739F6E39B382A759C0BBF390D153DC3`。最后的干净策略刷新处理已进入此构建，后续全量复核继续。

另建普通身份隔离夹具，不操作健康长测的进程、策略或网络。首次夹具因.NET长TEMP路径与Go短TEMP路径不同被安全边界拒绝，保留bootstrap.log，改用实际TEMP并重建测试器后成功。初始公开参数设inactive，两个音频View由已保存v3布局准备；节点准入走部署助手，实际连接、关闭、草稿输入、取消和停止均走原生UI。首次窗口重开自动连接截图上界6190ms；未保存策略退出出现原生Yes/No、默认No，取消后草稿保留；恢复草稿初值后明确停止，关闭重开仍为停止。证据指针`native-oct25-gui-fixture.json`，私有原始截图/时间在夹具`ui-captures.jsonl`和`metrics-gui-lifecycle.jsonl`。这是独立小规模图形版烟测，尚不是满负载5轮生命周期或8组合全部通过。

健康长测另保留第2次正常写入的计时证据不足：独立读回435ms、完成截图上界2073ms，不能证明UI在2秒内完成，不计有效100次。原始操作未重试；后续独立操作继续逐次计时。不得把此观察改记为通过或删去。

#### Metrics 正常退出订阅缺陷与兼容性补测（2026-10-25，进行中）

隔离原生夹具发现：Metrics GUI明确停止后，以同一身份启动控制台，已打开Desktop的音频订阅持续not_found。根因是Runtime.Close在Host仍连通时注销资源，发布了永久删除事件；SDK按契约终止订阅。现调整为先关闭Host、再注销本地指标，保留NotFound终止语义。新增16个durable订阅、5轮正常重启的真实运行时回归；旧代码第一轮失败，修复后10次重复通过（50次重启），race重复3次通过。日志native-oct25-metrics-close-before-fix.log、native-oct25-metrics-close-after-fix.log、native-oct25-metrics-close-race.log及native-oct25-metrics-after-close-fix.log保留。主健康长测仍运行原Metrics二进制，不能把其健康证据冒充修复后退出路径验收。

混合版本原生烟测使用隔离普通身份、真实Hub/Metrics控制台二进制和原生Desktop，沿用已保存两音频控件及状态。每次独立CoreAudio回读，记录程序hash与实际控制结果。001（旧Hub/旧Desktop/新Metrics）出现旧修订拒写，随后一次新操作成功，后续仍有冲突；全部结果写入native-compat-failures.jsonl及native-compat-writes.jsonl，不据一次成功声称该组合完整通过。其他组合继续验证，不冻结最终N。

健康运行第33次截图确认上界2314ms、独立回读201ms，同样标记计时证据不足，原始记录保留。汇总脚本明确列出所有inconclusive操作并保持Pending；补足100次有效观察不能抹去这些不确定项。

Metrics退出修复的实际控制台程序（6F1935B2…）完成隔离原生5轮正常退出，每轮CTRL_C退出码0，Desktop音频订阅自动恢复上界分别3016、2483、2917、2590、2761ms，未再永久not_found。仍是两音频订阅的小规模回归，不替代主2000订阅矩阵。

001异常进一步确定为混合心跳回归：全旧版000能收到外部音量20%变化；旧SDK读取新Metrics的订阅先收到revision2—7，随后连续超时，和原生停更一致。b1fcd1a产品没有周期心跳，而新Metrics三秒空闲检测误断旧Hub连接；旧Hub没有新版断链转发清理，旧消费者仍显示live。新增产品显式IdleAfterPeerHeartbeat选项，仅在同一认证peer/current epoch首次心跳后启用空闲检测，启用后持续生效；非产品显式IdleTimeout保持原义。link及nodehost测试/race通过。修复Metrics39256CAE…搭配旧Hub和旧SDK，独立12条订阅事件连续更新至revision14，没有此前5秒后停更。原生外部值和控制继续复测，正式全新版故障时限需在新构建核对，不把兼容路径当作新版时限证据。

#### 心跳兼容最终回归与原生重复检查（2026-10-25，仍在实施）

仅等待周期首心跳的初版在连接完成后立即黑洞场景未开启检测，网络集成失败记录保留于native-oct25-all-go-heartbeat.log；该日志另有测试命令继承原生夹具环境变量导致的错误，不能混称产品失败。最终实现子连接认证后立即发送首心跳、对端仅首次回应，之后沿用周期心跳。网络四场景各5轮通过（native-oct25-heartbeat-fast-start.log），清除夹具环境后的全Go测试通过（native-oct25-all-go-heartbeat-clean.log），link/node/nodehost race通过（native-oct25-heartbeat-final-race.log）。

同一源码生产构建为Hub9640C3B3…、Desktop8B7575EE…、Metrics控制台E0F18B8E…、Metrics GUI35299945…。真实b1fcd1a基线000沿用其独立证据，其余001—111以此统一构建完成原生Desktop外部音量20%订阅、静音开启/关闭及独立CoreAudio回读，均恢复初值40.000009536743164%/false。私有夹具native-compat-cases.jsonl/native-compat-writes.jsonl中trial=unified-build及unified-XXX截图记录本轮；早先001失败继续保留。此处仅两音频变量兼容烟测，不是正式N冻结或满负载八组合。

隔离夹具进一步启动1998示例，原生确认2000/2000。最新Metrics控制台正常退出5轮（退出码均0）恢复上界3497—4394ms，强制终止5轮3386—3975ms；Hub正常退出5轮（退出码均0）3913—4680ms，全部自动恢复2000项及可见音频值。逐轮数据见native-oct25-unified-lifecycle-results.json和夹具unified-lifecycle.jsonl。主健康两小时仍运行先前二进制，版本边界不能合并。

Hub强制终止第一轮发现新缺陷：计数3853ms恢复2000，但目录缓存重置后总览行全部消失，超过15秒仍未出现，记Failed，不能按计数放行。修复归属A03/A05：总览保留其自身订阅集合的资源描述，当前目录优先更新描述，明确停止才释放，值/失效继续由原控制器表达；列表引用只在订阅集合变化时重建，避免每次数据事件重新筛选2000行。新增回归先在旧代码失败（native-oct25-overview-retention-before.log），修复后总览/2000后台订阅4项通过（native-oct25-overview-retention-after.log）；完整前端及新原生构建复测继续进行。总览选择仍遵守已批准计划的会话状态边界，Desktop重启仅要求已保存View/Profile恢复，不擅自新增持久字段。

#### Hub 重连丢订阅及长测统计复核（2026-10-25）

总览保留修复版 Desktop `BF1CF363…` 的 Hub 强杀第一轮在3930ms恢复2000项并保留行；第二轮只恢复1999/2000，超过15秒仍有1项异常，记录为失败（`retention-hub-force-2-*`）。新增真实TCP/产品容量的2000订阅重启回归，在旧转发实现第9轮复现1991项恢复：Hub保留2000份转发兴趣，Owner只有1991份，客户端2000份pending。根因是路由已发布但下一跳会话未就绪时，转发失败既未清理刚登记的兴趣，也未向远端返回错误。

`routeEnvelope` 现在在未入队的转发失败时清理兴趣，并向远端请求/控制帧返回stale_epoch；durable订阅可按既有机制恢复，写入仍不自动重试。确定性回归把路由就绪/会话未就绪状态固定，检查及时错误和双方清理，20次通过；2000项×10轮重启测试连续5次通过（50轮），另3次race通过（30轮）。全量`go test ./...`通过。证据`native-oct25-hub-restart-2000-diagnostic.log`、`native-oct25-hub-restart-2000-forward-fix.log`、`native-oct25-forward-fix-all-go.log`、`native-oct25-forward-fix-race.log`。生产原生复验进行中，不能把后端回归当作原生通过。

前一候选健康窗口13:16:52—15:16:52结束，最终原生界面2000/2000，独立读取71982次、0错误、1个provider实例、最大快照年龄1002ms。88次有效串行写入；另2次截图上界超过2秒保留为无法判定，未凑满100次。Hub私有提交内存首末中位增长229376字节、Metrics659456字节，均在预算内。Desktop原始进程树统计超预算，原始summary保持Failed，不删改。

该Desktop内存结果存在已证实的归属污染：并行原生窗口同名可执行文件共用Wails默认WebView2数据目录及浏览器进程；后启动窗口renderer PID49292（末10分钟中位1031.8MiB）被计入主窗口。关闭隔离窗口后49292消失，主窗口renderer40688继续存在。原有Desktop进程中位483.6→484.6MiB，主renderer871.2→906.8MiB，只作诊断，不能用扣除共享进程的事后估算改判通过。正式内存/100写检查仍Pending，需隔离后重测。随后测试副本改为独立可执行文件名，二进制内容哈希不变；已核实两个浏览器父PID分别为22836/5052、数据目录分别为mfh-desktop.exe/mfh-desktop-isolated.exe。设置WEBVIEW2_USER_DATA_FOLDER无效的尝试亦保留，当前Wails原生loader会清除此环境变量。证据`native-oct25-browser-attribution-isolated.json`、长测私有目录的`summary.json`及原始逐进程样本。

#### 当前转发修复版原生复验（2026-10-25，进行中）

当前统一构建：Hub `916AA1E3…`、Desktop `93FB21C1…`、Metrics控制台 `D816C605…`、示例 `C5823811…`。隔离夹具1998示例加2真实音频订阅，Hub正常退出5轮恢复3625—3904ms、强杀5轮3650—4367ms；Metrics控制台正常退出5轮4005—4834ms、强杀5轮3950—4510ms。正常退出全部退出码0；每轮原生截图核对失效、2000项恢复及总览行保留。逐轮证据`native-oct25-forwardfix-lifecycle-results.json`及夹具`unified-lifecycle.jsonl`，trial为forwardfix。节点树为按需快照，手动刷新可恢复树；不把实时变量恢复解释为自动拓扑推送。

Desktop继续按会话订阅与持久View的原边界验证：退出前2000项，重启后恢复已保存音量/静音View。正常第1轮上界6762ms，第3轮5565ms；第2轮窗口尚未出现时取图失败，后续截图上界71095ms，状态恢复但15秒计时无法判定，保留并补测，不能改记通过。原始证据`desktop-lifecycle.jsonl`及`ui-forwardfix-desktop-*.jpg`。

第二次健康长测使用上述统一构建及独立浏览器进程，预热后窗口为16:14:07.844—18:14:07.844（UTC+8）；指针`native-oct25-endurance-r2.json`。第3次正常写入保守计时UI2864ms、独立读回2366ms，超过2秒，保持未通过。独立观察者期间每100ms持续采样、读取耗时1—8ms；该次未记录原生输入派发完成点，尚无法区分输入工具延迟与应用处理延迟。第4次起另记input_completed_ms作诊断，仍使用提交前保守上界，不扣除工具延迟来放宽门槛。长测结果待结束汇总，额外成功操作不抹去第3次失败观察。

#### 当前候选图形版生命周期与第二次长测结果（2026-10-25）

当前 Desktop `93FB21C1…` 与 Metrics GUI `5CB5A217…` 已补齐本机图形版生命周期。正常退出由原生窗口关闭触发并核对进程退出；强制退出核对目标路径后终止。恢复计时遵循 P31，从进程启动起包含初始化，到原生截图确认完成，全部有效轮次小于15秒。

| 产品及操作 | 有效轮次 | 恢复截图上界（ms） | 检查边界 |
| --- | --- | --- | --- |
| Desktop 正常退出 | 5（1、3、4、5、6） | 6762、5565、5972、4697、6113 | 退出前2000项；重开恢复已保存的两个音频控件和实时值 |
| Desktop 强制退出 | 5 | 6035、5692、3658、4598、2914 | 同上；会话级总览选择不要求持久化 |
| Metrics GUI 正常退出 | 5 | 8303、6606、6199、7913、7878 | 保存的active连接自动启动；Desktop自动恢复2000项及音频行 |
| Metrics GUI 强制退出 | 5 | 7332、7313、7356、7397、6577 | 同上，无手动重新连接或订阅 |

汇总`native-oct25-forwardfix-gui-lifecycle-results.json`共21条：20条Passed、1条Inconclusive。Desktop正常第2轮71095ms仍保留为无法判定。Metrics正常第2轮首次关闭因过期截图ID被工具拒绝，未发生关闭；刷新窗口截图后重新执行，原工具错误保留，不计作产品退出失败或成功。上述结果不替代未保存/不补发的全部失败矩阵。

R2健康窗口已结束：72000次独立读取、0错误、1个实例，最大快照年龄987ms。首末10分钟各119个内存样本，Hub私有提交中位45719552→46153728字节（+434176），Desktop 1737527296→1712087040（−25440256），Metrics 28422144→29577216（+1155072），三者均满足既定增长预算。Desktop统计包含本窗口专属WebView2进程，隔离窗口使用不同可执行名称，归属证据保留。

本轮只有82次健康窗口内原生写入，79次满足双重2秒检查，未达到100次；003、028、079的UI/独立读回分别为2864/2366、2222/726、2153/1748ms，保留未通过。028与079输入派发完成上界分别595ms、1587ms，只用于诊断，不从验收计时中扣除。后段执行中断导致窗口内未补足次数，18:36后的2000项截图仅为晚到状态证据，不能充当窗口末时刻或新增健康写入。原始summary.json保持Failed。

只读观察器在18:19:05达到配置的135分钟期限，stderr为`context deadline exceeded`；末次成功读取位于启动后134分59.901秒，晚于健康窗口结束约5分钟。此项已核对为辅助程序预期终止，不是产品读取失败；原始stderr及summary中的通用错误提示仍保留，解释另存`end-of-run-diagnosis.json`。两小时内存与连续读取有本机通过证据，但100次原生写入及其异常计时仍需重测，完整A验收继续Pending。

#### 包内程序、八组合与独立移除复验（2026-10-25）

生成当前预览包并逐项验证5个ZIP、6个Windows x64可执行文件、1394个源路径记录（其中3个删除项，源码ZIP含1391项）。包及源码指纹见`current-package-verification.json`；acceptance_status始终为pending，不冻结正式N。实测包内二进制与前述统一构建哈希相同。本轮仅将review/todo证据同步进源码归档，再次核对二进制哈希与已测包相同。

解压后将子进程PATH限制为Windows/System32与Windows，确认go/node/npm/wails不可发现；Hub、admin、Metrics控制台、示例分别首次生成身份并再次读取，身份一致。包内新版Hub、Metrics控制台和Desktop沿用隔离身份与已保存View，在相同PATH下完成000—111八种真实二进制组合。每组核对外部实际音量20%订阅、原生静音开启/关闭及独立CoreAudio回读，共16次实际控制，独立确认206—639ms。最终精确恢复原始音量40.000009536743164%和静音false。证据`native-oct25-package-clean-path.json`、`native-oct25-package-native-compat-results.json`，原始记录trial为package-forwardfix。

000首个立即截图早于控件更新，后续确认上界9137ms；010首个外部改值截图早于下一次采样，后续截图确认20%。原始观察保留，八组合结论仅为功能兼容，不声称混合版本逐次满足P27时限。各组均复用已准入状态并成功接入，不宣称本轮重新执行首次准入；仅两个音频订阅，不替代全新版2000项容量。包内Metrics GUI同样在系统目录PATH下自动恢复连接，1秒音频采样显示新鲜40%/false，Desktop同步恢复；GUI首次窗口枚举过早失败，重新获取实际窗口后确认，不记为产品启动失败。

分别停止并移除Desktop、Metrics GUI、Hub的测试解压程序目录，再从已校验ZIP重新解压；状态目录位于程序目录之外，移除前后分别核对8、8、9个文件哈希完全一致。每次其他两产品的PID和创建时间不变，恢复后原生Desktop显示原View和当前40%/false，Metrics GUI自动连接。证据`native-oct25-package-redeploy.jsonl`及`package-*-redeploy*`截图。只删除本次创建且验证绝对路径的测试程序目录/副本，未删除身份、用户目录或工作树。

上述证据补足本机系统PATH隔离与默认移除后恢复检查，不能称作已验证无开发工具安装的干净Windows虚拟机；WebView2已存在。独立单产品升级/回滚时其他产品持续运行、最新撤权保留的完整原生矩阵，以及正式N冻结仍待补。

#### 2000订阅下原生写入终态故障检查（2026-10-25）

当前包内Desktop/Hub/Metrics不变，独立测试提供者保持1998个1Hz示例变量，仅对一个标准布尔变量包装显式故障，再加2个真实音频变量，原生确认2000/2000。测试夹具位于ignored out目录，不进入产品程序。其操作日志记录请求、实际提交及确认返回时刻，不能以UI显示代替实际执行次数。

| 注入场景 | 原生观察 | 提供者证据 |
| --- | --- | --- |
| 写入已执行，确认延后6秒 | 显示结果不确定、草稿保留、禁用直接提交；迟到确认不改判成功；手动刷新后仍需明确处理旧草稿 | 1次请求、1次实际写入true；确认延迟6002ms；没有重发 |
| 成功确认后独立回读失败 | 明确显示“写入已确认，回读失败”，保留待核实状态；手动刷新成功后恢复 | 1次请求、1次实际写入false，随后读请求返回显式故障 |
| 明确冲突拒绝 | 795ms截图上界显示拒绝原因，草稿true保留，实际值false，未等待满5秒 | 1次请求、0次业务写入 |

故障恢复正常并等待后，总计仍仅3次请求、2次实际写入，未自动提交旧草稿。清理本次草稿、停止故障辅助程序并恢复原1998变量示例，音频未被这些故障操作修改。证据`native-oct25-fault-ui-results.json`、夹具`fault-provider-events.jsonl`/`fault-ui-actions.jsonl`及`fault-*`截图。

缺确认场景的原生截图在提交前起点后6122ms取得，只能证明终态正确；既有精确虚拟时钟5秒回归保留，但此次原生观察不能独立证明5秒上限。回读失败场景输入工具调用本身耗时2365ms，总截图上界3106ms，独立提供者从收到请求到回读拒绝仅32ms；这证实本机输入派发工具存在显著延迟，不据此扣除工具时间或改判历史R2超限。严格原生计时与100次健康窗口写入仍Pending，不以补录窗口外写入凑次数。
