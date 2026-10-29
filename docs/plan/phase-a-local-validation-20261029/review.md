# A 本机多实例验证结果

2026-10-29；基线 `6cd785a7b179fa899a9fd70c7d48188161d1fb58`。用户批准F02/F03在本机多实例推进；F01不补测、F04双机/干净环境暂缓。原AC定义与历史结果见[归档审查](../../plan/phase-a-20261029/review.md)，不将未验证项改判Passed。

## R2 当前结论（2026-10-29）

本机 F02/F03 的本轮补验已完成；此前严格时限采样和原生重复次数缺口由下列新证据补齐。A-AC-v1、产品定义、F01 不补测与 F04 暂缓均保持原状，完整 A 仍为 pending。没有归档、合并或推送。后文保留前轮结果，不覆盖 b1fcd1a Metrics GUI 的失败。

| Task / AC | 当前结果 | 证据与边界 |
| --- | --- | --- |
| F02 / AC07 | Passed（本机无回执补验） | 5 轮原生明确提交，终态截图上界4754/4777/4740/4760/4791ms，均小于5秒。故障发生在写前读取阶段，原生端确认未发送写入并显示拒绝；不能把这些样本称为已执行但丢失写回执。草稿保留、恢复不补发；1801次独立CoreAudio采样保持初值。迟到写确认、前置读取截止及回读共用截止另由31项前端测试和Go写入测试覆盖。 |
| F02 / AC08/15 | Passed（原生5轮） | 5次撤写后明确提交均被拒绝（音量2次、静音3次）；随后撤read/subscribe，音量与静音订阅失效，界面上界394/501/340/352/402ms。每轮独立改变实际音量至20%，UI仍保留已标失效的25快照，不显示新保护值；重新授权后手动刷新，旧草稿不自动发送。 |
| F02 / AC09/15 | Passed（补足严格采样） | Desktop静默中断新增2轮，失效上界4823/4462ms，2000订阅恢复上界2198/2238ms。结合前轮其他18次有效严格计时；保留前轮2次Inconclusive，不改写历史样本。网络/恢复实现未改，复用其余结果。 |
| F03 / AC16/17 | Passed（明确基线与边界） | 真实6cd785a导出源码，归一化核对284个Go/module输入；000—111八组合及6项进程独立切换通过。原生Hub/Desktop/Metrics独立替换6项，界面恢复2443—3549ms；其他PID/创建时间不变，身份、Metrics配置及保存View哈希不变。基线与当前Hub都保持policy revision37，原生旧Desktop的实际写入仍拒绝。 |
| F03 / AC03/19 | Passed（预览交付校验） | 六个Windows x64程序、独立产品包及完整源码包，最终指纹和校验见out/phase-a/evidence/oct29-r2-final-package.json；仍标pending，不冻结正式N。 |
| 范围保留 | 不变 | F01历史正常2秒失败不重测、不洗绿；F04真实双机/干净系统暂缓。正常退出/重启历史矩阵与退出草稿抽查仍按原证据边界复用，不声称本轮重跑。 |

生产修复只涉及 `apps/desktop/variable_write.go` 与前端 `variable/controller.ts`：原生写前读取/发送共用4秒预算、前端整次等待4.5秒，为5秒可见终态留出桥接与渲染余量；回读不重新计时，无写重试。正常2秒门槛不变。对应Go/前端测试和Wails生成物同步；31项前端测试、Go Prepared写入测试、go vet、实际GUI操作与范围内diff检查通过。测试脚本还改为明确并校验BaselineCommit，误标基线会在启动测试进程前拒绝。

**Requirements：Passed（本轮已批准本机F02/F03补验范围）**；完整A仍pending。新基线完整SHA为6cd785a7b179fa899a9fd70c7d48188161d1fb58，archive SHA256为9A81728DC0906D1A61D6720ABADE2CD08684B2E0C0902D40B2AF709A1F7C060C。新Desktop SHA256为434C0F4A103FD4D07A6BB58457046D04E1FDE71DC193F66ADBF9D8D57338E62C，与基线674939...确有差异；Hub与Metrics两快照程序相同，只计独立替换恢复，不冒充不同二进制兼容。旧b1fcd1a GUI回滚仍Failed且不在支持窗口。

**Standards：Passed（当前改动复核）**。写集/模块边界符合原计划；未改协议、授权与持久格式；迟到结果隔离、单次写入、明确错误继续成立。检查了已提交基线至整个工作树的自有改动、生成物新增/删除、测试适配器与脚本清理；原有go.mod/SVG换行差异及主线无关改动保留。串行原生操作避免共享音频/焦点竞争，无子Agent。性能与安全复核限本次路径：2000实时订阅下计时，不重跑F01；私钥/Permit仅留TEMP，不进入包。原生测试结束后恢复初始音量0.25196868和静音false，夹具产品/中转进程剩余0。

证据主索引：[R2汇总](evidence/oct29-r2-summary.json)、[兼容矩阵](evidence/oct29-r2-compatibility.json)、[前端回归](evidence/oct29-r2-frontend-tests.log)、[输入不符拒绝](evidence/oct29-r2-baseline-mismatch.log)。最终候选/计划文件指纹与打包前后等价核对在oct29-r2-final-package.json；原生所测包为phase-a-worktree-d23a7e974226，后续只增加测试、脚本和文档，不改生产程序。

R2夹具排除项保留：首个external20调用因PowerShell参数解析失败，未发生音量改变，随后显式重试成功；首次替换因启动瞬间Desktop路径为空，在停止任何产品前被校验拦截，核实真实进程路径后重试；首版矩阵报告硬编码旧commit，保留为mislabeled并重新核对/执行；go.sum因CRLF导致原始blob不一致，采用仓库文本归一化后匹配。均不计入产品通过次数。修复循环有新有效证据进展，连续无进展次数0。

## 前轮结果（保留历史）

| Task / AC | 结果 | 证据及边界 |
| --- | --- | --- |
| F02 / AC09/15 | Passed（协议层） | 两条真实loopback TCP路径的关闭/静默中断各5轮，共20轮；oct29-f02-protocol-faults.log。不是原生UI计时或满负载故障证明。 |
| F02 / AC07/08/11 | Passed（协议层） | 撤write保留read/sub、撤read/sub阻止后生成保护值新增5轮；CAS/实例替换各20轮；oct29-f02-revoke-five.log与protocol-faults.log。 |
| F02 / AC06/07/10 | Passed（针对性回归） | Desktop/Metrics GUI/SDK写入、退出、配置恢复与续租边界测试；oct29-f02-boundaries.log。 |
| F03 / AC16 | Passed（进程/绑定层） | 最新源码构建程序与真实b1fcd1a的000—111八组合；现有状态读取、订阅、配置控制、管理CLI与撤权回滚检查；compatibility-bridge-matrix.json。 |
| F03 / AC17 | Passed（进程/绑定层） | 独立替换Desktop绑定测试进程、Metrics控制台、Hub，升级/回滚共6项；其余进程PID/创建时间不变、身份/配置hash不变，最新撤权持续拒绝写且read/sub可用。恢复66—310ms，只计该层观察，不是原生UI时限。 |
| F03 / AC03/19 | 构建Passed，包校验另见证据 | 当前Hub/admin、Desktop、Metrics GUI/控制台、示例已构建，包含10月29日UI简化；包manifest保持pending，非正式N。 |
| F02/F03 原生UI | 工具已恢复；结果见下文 | 用户重启 Codex 后，node_repl + @oai/sky 可正常操作实际 Wails 窗口。本轮完成隔离夹具与原生补验，已发现 Metrics GUI 回滚失败；不能再把剩余工作归因于工具初始化。 |

测试修改仅涉及 `scripts/test-phase-a-compat.ps1` 与 `tests/fixtures/phase-a-desktop/main.go`：增加显式probe/stop模式保持同一消费节点进程，并在单产品替换前后独立检查存活进程与状态/权限。初次脚本误将metrics.json放在state根上一级，R1停止、旧日志保留；修正为state/metrics.json后R2通过，不计作产品缺陷。

当前生产代码未修改；重建的生成物仅已有换行差异，独立源码比较及包manifest用于绑定结果。既有原生候选结果保留为历史证据，不充当本轮新包原生验收。

## 复核与恢复入口

Requirements：**未通过**，原生 Metrics GUI 回滚失败，部分严格时限采样仍不充分；F01明确不补测、F04暂缓，本轮Passed只覆盖具体层次。Standards：**Passed（本轮测试工具改动的轻量审查）**，Go编译/实际进程矩阵、go vet与范围内git diff --check通过；检查测试工具的输入边界、独立进程/状态保护、finally清理、不通过观察不伪装成功。无新产品权限、协议或持久化变更。

活动计划见[plan.md](plan.md)，机器可读汇总为 `out/phase-a/evidence/oct29-local-summary.json` 与 `oct29-native-summary.json`；日志、程序/包SHA和恢复指针保存在本工作树out/phase-a/evidence。测试夹具位于本机TEMP，仅使用本次创建的身份与端口，私钥不进入源码包。原生补验曾通过独立 CoreAudio 夹具将音量改至20%，已精确恢复原始 level=0.25196868、静音false；所有本轮夹具产品/中转进程已停止。未推送、未合并、未自动归档。

## 重启后的原生补验

所测生产程序来自 `phase-a-worktree-6b89358614cd`；源码基线仍为6cd785a7。Desktop二进制SHA为674939017C8CB4EB7B8EC7BD5BE5F67D89F1B6E106B63AFD70ECC1AA2C6E99F9。只更新文档并重新打包时复用这些程序哈希，不把不同源码包指纹当成不同生产实现。实际原生截图均由Computer Use获取，没有使用网页模拟界面代替。

| 原生检查 | 结果与证据边界 |
| --- | --- |
| 2000变量 | 界面订阅1998个示例变量及音量/静音2项，截图 `oct29-native-2000-live.jpg`。不重新执行F01耐久/100写。 |
| 满负载网络故障 | 两条loopback TCP路径×关闭/静默中断×5个有效轮次，共20轮，均显示失效/禁写并恢复2000实时项。18轮在5秒内截图确认失效；Desktop静默中断补测6/7首次截图为5679/5592ms，时限判Inconclusive，不能证明实际超过或满足5秒。20轮恢复截图上界最高3039ms，满足10秒。 |
| 撤权 | 5轮撤read/subscribe，volume界面失效上界287—566ms；首轮同时观察静音失效及撤权后新生成的20%未被显示。撤write仍可读取；实际提交被拒绝，重新授权/刷新后旧草稿要求重置。保护值生成和真实拒写的原生抽查不冒充各5次；协议层5轮另有日志。 |
| 无回执与不重放 | 有效轮次3/4/5/7/8均由原生界面明确提交，显示等待后转为结果不确定；恢复后保留草稿、没有自动重试。3826次独立CoreAudio采样均保持原始音量/静音。结果截图在9379/5466/5522/5578/5537ms取得，严格5秒UI上限仍为Inconclusive，不能按提示文本认定计时通过。 |
| 正常退出草稿 | 原生窗口关闭弹出未保存内容提示，取消退出保留草稿；单次抽查截图 `oct29-native-unsaved-exit-dialog-valid.jpg`。本轮未重做历史重启矩阵。 |
| 独立升级/回滚 | 实际Wails Desktop、Metrics GUI及Hub，每次只替换一个产品，其余PID/创建时间保持；身份、Metrics配置、保存View哈希不变。Desktop回退/升级、Hub回退/升级、Metrics升级5项均在2709—3148ms观察恢复。Hub旧/新版均保留最新policy revision17，旧Desktop实际写入仍被拒绝。 |
| Metrics GUI回滚 | **Failed**：真实b1fcd1a没有connection_settings.go，启动显示默认Node20/已停止，未恢复保存的Node4连接。保留状态文件本身不等于恢复了状态。恢复当前GUI后二者无需重建身份或清空状态即可恢复。部署说明已明确禁止把该GUI作为A受支持的回滚目标。 |

原始的夹具错误与无效样本均保留：policy get误用id而非key被服务端拒绝；Desktop断连首轮采样过早；静默中断第4轮中转控制文件替换被占用、下一轮仍属同次中断；无回执前两次未有效提交，第6次在故障注入前发生文件占用错误。中转控制写入已增加有上限的文件占用重试；这不是产品写入重试。遮挡中的Metrics截图出现Desktop内容，只有激活Metrics后的截图用于GUI结论。机器可读汇总明确列出排除项。

下一步仍限F02/F03：补严格界面时限证据，补足原生保护值/拒写抽查次数，并处理Metrics GUI支持的回滚基线。不得修改真实b1fcd1a源码后仍冒称原版本，不得用绑定适配器/控制台通过代替GUI通过。原A合同不变；修订受支持版本窗口需要独立说明与重新验证。当前预览包仍为pending，不能发布为完整A验收版本。


归档说明：2026-10-29 用户授权合并。本文件保留验证时的结论与历史阶段描述；最新收尾见 [本次变更记录](../../change/2026-10-29_phase-a-local-validation.md)。原 out/phase-a 证据已保存在项目 archives/phase-a-local-validation-20261029/workflow-evidence-private.zip。
