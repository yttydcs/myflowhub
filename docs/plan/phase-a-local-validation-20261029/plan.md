# A 本机多实例验证计划

状态：Approved / validation-needed；2026-10-29 用户确认“好的请继续”。承接已批准 A01—A08 的剩余 F02/F03；F01不补测，F04双机/干净环境暂缓。

- 活动工作树：D:/project/MyFlowHub3/worktrees/phase-a-local-validation，分支feat/phase-a-local-validation；基线6cd785a7b179fa899a9fd70c7d48188161d1fb58。
- 原方案/写集：[批准计划](../../plan/phase-a-20261029/plan.md)；当前范围：[后续清单](../../plan/phase-a-20261029/follow-up.md)；合同：[A-AC-v1](../../requirements/phase-a-acceptance.md)。不更改架构和数值要求。
- F02 / A04/A05/A08 / AC07–11/15：当前候选本机多实例撤权、新保护值、两条路径关闭/静默中断、退出草稿/不补发，各场景5轮。必要修复限原计划apps/desktop、apps/nodes/metrics、runtime、sdk及直接相关测试。
- F03 / A07/A08 / AC03/16/17/19/21：重建最新程序与独立包；目标产品独立升级/回滚，其他产品持续运行，保存状态及最新有效撤权保留。scripts/package-phase-a.ps1、scripts/test-phase-a-compat.ps1及原生/集成测试在原写集内。
- 用户确认本机独立身份/状态/端口，多Desktop核对WebView2归属；同物理音频串行并恢复初值。CLI负责夹具部署、准入和权限注入；日常UI验证使用原生操作。
- 不重跑F01两小时/100写，不冒充真实双机，不推送发布、不自动归档合并。程序指纹可固定为验证候选，但F01/F04未通过时不称作完整A验收通过。
- 串行执行：GUI输入与音频共享本机资源，不委派。Docs root为本工作树docs；仅测试阶段证据更新，不新增完成归档。缺陷改变行为时同步相关features/specs。
- 回滚：停止路径核验过的本次夹具进程，恢复音频初值；保留失败证据与最新权限状态，修复可按文件相对基线回退，不触碰用户主线改动。

当前结果见[review.md](review.md)。

2026-10-29 R2：已完成本机 F02/F03 补验。新回滚基线固定为真实 6cd785a7b179fa899a9fd70c7d48188161d1fb58；候选仅对 Desktop 写入等待预算作生产修复（原生4秒、前端4.5秒，为5秒可见上限留余量）。测试脚本新增基线输入核对。原AC数值、F01不补测、F04暂缓及不自动归档边界均不变；结果与证据边界见review的R2结论。

2026-10-29 重启后更新：原生自动化已恢复，20轮满负载故障及5轮无回执功能检查完成；严格采样和原生撤权抽查次数仍有缺口。独立GUI替换发现b1fcd1a Metrics GUI不恢复连接，保留为Failed并收紧部署说明，不改变验收标准。继续项与证据见review/todo；没有新的归档或合并授权。


归档说明：2026-10-29 用户授权合并。本文件保留验证时的结论与历史阶段描述；最新收尾见 [本次变更记录](../../change/2026-10-29_phase-a-local-validation.md)。原 out/phase-a 证据已保存在项目 archives/phase-a-local-validation-20261029/workflow-evidence-private.zip。
