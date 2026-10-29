# 2026-10-29 A 本机 F02/F03 补验收尾

## Result

用户在完成本机补验后明确要求“合并回去”。本次合入 F02/F03 的实现、测试工具和文档；A-AC-v1 与此前确认的产品结论不变。F01 按用户决定不补测，F04 双机/干净环境暂缓，历史失败仍保留，完整 A 和正式 N 均未验收冻结。

Desktop 原生写前读取/发送预算由5秒改为4秒，前端整次等待含回读由5秒改为4.5秒，为5秒可见反馈留出余量；仍只提交一次，保留失败草稿，不自动重试或补发。兼容脚本校验真实基线SHA与Go/module输入，并支持独立保持其他产品进程。GUI回滚基线限定为真实 `6cd785a7b179fa899a9fd70c7d48188161d1fb58`，保留旧b1fcd1a Metrics GUI恢复失败。

Docs root：同仓 `docs/`。完整[计划](../plan/phase-a-local-validation-20261029/plan.md)、[待办](../plan/phase-a-local-validation-20261029/todo.md)、[权威验收与Requirements/Standards结论](../plan/phase-a-local-validation-20261029/review.md)已移出临时工作树；主线原有根控制文档和设计稿不纳入本次提交。

## Evidence

- F02 / AC07–11/15：无回执前置读取5轮终态4.740—4.791秒；撤权实际拒写/新保护值5轮，界面切换340—501ms；补测2轮满负载断网，识别4.462/4.823秒，恢复2.198/2.238秒。未发送写入的拒绝与执行后丢失回执严格区分，迟到确认和回读截止另有单元测试。
- F03 / AC03/16/17/19/21：8个后端组合与6项进程独立替换通过；6项原生替换恢复2.443—3.549秒，身份、配置、View及最新撤权保留。Hub/Metrics两个快照中的程序相同，只证明独立替换恢复；不同二进制版本验证限Desktop。
- 31项前端测试、Go写入测试、go vet、源码/ZIP/Windows x64程序校验通过；主机音频精确恢复，测试进程已清理。
- 候选以6cd785a为基线，已测预览包源码指纹 `8ffefe1f9b01c378cb8f975e8af1a3b0e0b9ed9f7e1e0f3d05c4adbc677caba4`；[候选指纹与复核](../plan/phase-a-local-validation-20261029/evidence/oct29-r2-final-package.json)记录生产程序等价性、计划/需求哈希及未验证边界。本次归档仅补充文档，不重建或重标该预览包。
- Requirements与Standards在已批准的本机F02/F03范围Passed；不以此替代完整A。F01历史正常2秒失败及F04未测风险沿用用户已确认处置。

完整证据、恢复副本及原始失败记录见[持久证据入口](../plan/phase-a-local-validation-20261029/evidence/README.md)。私有夹具和主线修改恢复ZIP仅保留本机，不提交Git；可提交摘要与代表截图单独归档。

## Documentation

- Intake impact: none；原始讨论继续引用[产品讨论](../intake/2026-09-06_product-definition-grill.md)，本次合并授权记录于本页。
- Feature impact: updated；[Desktop](../features/desktop.md)澄清等待预算和持久验收入口。
- Requirements impact: none；[A标准](../requirements/phase-a-acceptance.md)与[产品定义](../requirements/product-definition.md)未改。
- Specs impact: updated；[部署](../specs/phase-a-deployment.md)与[构建](../specs/build-and-ci.md)记录时间预算及真实回滚基线。
- Decision impact: none；没有改变协议、授权、持久格式或架构选择。
- Lessons impact: updated；[Windows原生验收](../lessons/windows-native-acceptance-evidence.md#界面截止时间与真实回滚基线)记录渲染余量、截图上界、基线核对与相同二进制证据边界。plan/change/lessons索引已更新，docs根拓扑不变。
- 串行操作共享GUI/音频，无子Agent委派。

## Rollback and closeout

本次只授权本地master合入，不推送、不发布。主线原有25个未提交文件逐字节保存并在合并后核对；保留其他工作树。收尾结果与恢复清单位于项目 `archives/phase-a-local-validation-20261029/`。合并前核对候选与已验证内容等价，合并后核对实际主线内容，再清理本轮工作树。

回滚应对本次自有提交定向revert，先检查后续依赖，不重置用户工作区。产品回滚只替换受支持目标程序，保留最新身份/配置及撤权；禁止用旧策略备份复活已撤销权限。预览包和失败证据继续保留。

收尾已完成：实现及归档提交 `55319c5dade6f6f17c97311035b36783e3592529` 快进合入本地master，31个提交文件在主检出核对一致。主线25个原有未提交文件逐字节未变、dirty状态未变。`phase-a-local-validation`工作树及已合并的`feat/phase-a-local-validation`分支已移除，其他工作树保留。未推送或发布；完整A状态仍pending。核对明细见本机归档目录`closeout-result.json`。
