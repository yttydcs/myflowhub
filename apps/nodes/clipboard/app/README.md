# MyFlowHub ClipboardNode Flutter shell

同一 Flutter source 构建三种界面：

- Windows：有界 NDJSON 调用同目录 `mfh-clipboard.exe -bridge`，Go 后端使用真实 Win32 clipboard adapter。
- Android：MethodChannel 调用 canonical gomobile AAR；Kotlin 只执行平台 ClipboardManager 读写并回报结果。
- Web：只提供明确标注的配置/UI 预览，因为浏览器不能托管当前原生 Node runtime。

Flutter/Dart 层不解析 MyFlowHub wire envelope，也不持有 Go runtime 对象。AAR 和 Windows 后端由 monorepo 构建入口生成；不应从旧 ClipboardNode 仓复制生成物。
