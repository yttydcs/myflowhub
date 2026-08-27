# Android Runtime And Mobile Bindings

## Summary

Android 的持久 identity/config 与 live session 必须分离；平台权限、Foreground Service、content URI 和 gomobile 产物都需要独立的边界验证。工程能编译不代表真实 AAR、JNI、后台重启或 RFCOMM 行为可用。

## Lookup Hints

- 症状：重启后显示已登录但请求超时、`content://` 上传失败、Bluetooth permission denied、APK 启动时报 JNI/class 缺失。
- 关键词：`START_STICKY`、null intent、`BLUETOOTH_CONNECT`、`content://`、gomobile AAR、`libgojni.so`、stub binding。
- 快速检查：新进程 session 是否从 disconnected 开始；AAR 是否含目标 class/ABI；URI 是否已 staging；Android 12+ 权限是否在运行时授予。

## Symptoms

- sticky service 被系统拉起后沿用旧 `logged_in` 标志，却没有有效 socket/subscription。
- Go sender 把 `content://` 当文件路径打开。
- 系统蓝牙权限通过，但 MyFlowHub join/resource 请求仍被拒绝，或反之。
- Gradle/Compose 构建通过，运行时才发现 AAR 是 stub 或缺少目标 ABI。

## Impact

形成半连接状态、后台重启循环、无法上传文件、错误权限提示或只在设备运行时暴露的崩溃。

## Trigger Conditions

- 把 durable UI snapshot 当成 live session proof。
- 未经 ContentResolver staging 就跨 Kotlin/Go 传递 URI。
- 混淆 Android OS permission 与 MyFlowHub resource authorization。
- 只验证 Gradle 编译，不检查 AAR/JNI 内容和实际 binding 方法。

## Root Cause

移动平台生命周期、文件能力和系统权限属于宿主边界；网络 session 和资源权限属于 canonical runtime 边界。把两者压成一个状态或用 stub 掩盖生成物缺失，会让构建和运行事实分叉。

## Investigation Trail

1. 对照持久 identity、用户期望运行状态和当前 transport/session 状态。
2. 检查启动失败后 transport 是否关闭，重连后是否重新 join 并恢复 connection-scoped 状态。
3. 对文件路径确认 scheme；`content://` 先复制到应用私有随机 staging 文件。
4. 解包 AAR，核对 Java class、JNI 库、ABI 和 minSdk；再做真实设备 smoke。

## Resolution

- sticky restart 使用版本化运行快照，但每个进程从 disconnected 开始并重新认证。
- identity 变化使 auth snapshot 失效；display name 变化不轮换 key。
- URI staging 成功后再调用 Go，完成/取消后清理临时文件。
- Android 12+ 显式申请 CONNECT/SCAN；OS 权限与 permit/policy 错误分别呈现。
- canonical 构建强制真实 gomobile AAR，不提供 stub fallback。

## Prevention / Guardrails

- Foreground Service 只声明实际工作所需类型，状态通知与业务通知使用独立 channel。
- 固定 Go/gomobile 工具链，验证 AAR class/JNI/ABI 和 APK 内原生库。
- 启动、重连、取消、半连接和 null-intent 路径都进入测试；真机不可用时明确标记 Unavailable。

## Related Docs

- [Android feature](../features/android.md)
- [ClipboardNode feature](../features/clipboard-node.md)
- [MetricsNode feature](../features/metrics-node.md)
- [Build and CI](../specs/build-and-ci.md)
- [vNext full migration](../change/2026-08-27_vnext-full-migration.md)
