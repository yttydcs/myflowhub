# MyFlowHub ClipboardNode Flutter shell

同一 Flutter source 构建 Windows 界面和 Web 预览：

- Windows：有界 NDJSON 调用同目录 `mfh-clipboard.exe -bridge`，Go 后端使用真实 Win32 clipboard adapter。
- Web：只提供明确标注的配置/UI 预览，因为浏览器不能托管当前原生 Node runtime。

Flutter/Dart 层不解析 MyFlowHub wire envelope，也不持有 Go runtime 对象。Windows 后端由 monorepo 构建入口生成；不应从旧 ClipboardNode 仓复制生成物。
