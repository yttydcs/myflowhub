# Windows 原生验收的进程归属与计时

## Summary

原生界面功能成立不等于严格时间门槛通过。并行 Wails 窗口可能共享 WebView2 进程，污染每产品 private committed 统计；原生输入派发和截图工具的延迟也必须保留。

## Lookup Hints

关键词：WebView2、同名exe、private committed、截图上界、input_completed_ms、100次写入。先查每个窗口 browser/renderer 父PID与用户数据目录，再查提交前、派发完成、实际读回和截图的时间点。

## Symptoms / Impact / Trigger Conditions

并行测试两个同名Desktop程序，某窗口新增renderer被另一窗口计入内存；UI输入完成迟于业务响应预算，截图晚于状态变化。会导致错误内存归属或无法判定严格时限，不能事后扣除估算后改判Passed。

## Root Cause / Investigation Trail

Wails默认WebView2数据目录与可执行文件名相关；本次设置WEBVIEW2_USER_DATA_FOLDER被加载器清除。关闭隔离窗口后对应renderer消失，主窗口renderer仍在，证明统计混入其他窗口。输入调用曾耗时2365ms，而独立提供者从收到请求到回读拒绝只有32ms，两条时间线不能混同。

## Resolution

使用不同可执行文件名的字节相同副本，核对浏览器进程和数据目录确实独立后重跑内存检查。保存旧Failed结果，不以事后减去共享进程代替重测。写入分别记录提交前、输入派发完成、实际副作用/独立读回和界面确认；严格验收仍以批准的起止口径判定。

## Prevention / Guardrails

每轮保存候选SHA、进程PID与创建时间、原始样本及有效健康窗口；次数只计窗口内操作，未达100不得补入窗口外动作。辅助观察器自然到期与产品错误分开解释，原始stderr/summary不覆盖。工作树清理前将ignored证据与私有夹具复制到本机持久目录，私钥/Permit不提交Git。

## Related Docs

[Desktop](../features/desktop.md) · [A验收](../requirements/phase-a-acceptance.md) · [部署](../specs/phase-a-deployment.md) · [阶段结果](../change/2026-10-29_phase-a-foundation.md)
