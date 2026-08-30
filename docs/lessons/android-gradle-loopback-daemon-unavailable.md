# Android Gradle daemon loopback 不可用

## Summary

在 Windows 宿主环境中，Gradle single-use daemon 可能在任何 project task 执行前因为 JVM loopback IPC 建立失败而退出。此时结果应记为环境 `Unavailable`，不能把缺失的 unit/lint/assemble 证据记为通过，也不能在 Go/gomobile 已通过时直接归因于产品代码。

## Lookup Hints

- 错误文本：`java.io.IOException: Unable to establish loopback connection`
- 常见 cause：`java.net.SocketException: Invalid argument: connect`
- 关键词：`Gradle daemon`、`single-use daemon`、`loopback`、`testDebugUnitTest`、`lintDebug`、`assembleDebug`
- 快速检查：确认错误发生在 task graph/编译前；比较两个独立 Android project 是否同签名失败；单独验证 `gomobile bind`、AAR ABI 与 Go tests。

## Symptoms

- Gradle 在 daemon 启动或客户端连接阶段退出，尚未进入 Kotlin compile、unit test、lint 或 assemble task。
- generic Android 与 Metrics Android 使用同一 JDK/Gradle 时出现相同异常。
- Go tests、gomobile AAR 生成、双 ABI 和 Java 导出面仍可成功。

## Impact

- 无法提供 Gradle unit/lint/APK assemble 证据。
- 不能据此判定 Kotlin/Android app 门禁通过，也不应把它自动归类为实现失败。
- 真机 install、前后台和权限生命周期仍需独立 device smoke。

## Trigger Conditions

- Windows 宿主上的 Gradle single-use daemon 需要本地 loopback socket IPC。
- JVM/宿主网络栈、endpoint policy 或运行环境使本地 socket `connect` 返回 `Invalid argument`。
- 故障发生于 Gradle task 执行之前，与具体 Android source change 无直接因果证据。

## Root Cause

本轮可确认的边界是宿主 JVM daemon IPC 无法建立 loopback connection；由于两个独立 Android project 在相同启动阶段得到同一异常，而 canonical gomobile toolchain 正常，证据不支持把根因定位到某个 Kotlin/Go 产品模块。更底层的 Windows/JDK 网络策略根因需在具备可观测宿主网络配置的环境另行诊断。

## Investigation Trail

1. 分别对 generic Android 和 Metrics Android 执行 `testDebugUnitTest`、`lintDebug`、`assembleDebug`。
2. 多次 single-use daemon 均在 task 前以同一 loopback exception 退出。
3. 验证 Go focused/full tests、race、vet 通过。
4. 分别生成 generic 与 Metrics gomobile AAR，核对 `classes.jar`、`arm64-v8a`、`x86_64` 和 `javap` ABI。
5. 因宿主未安装 `adb`，device smoke 另记为 Unavailable，不用 host-side tests 代替。

## Resolution

- 在当前环境将 Gradle 与 device checks 明确记录为 `Unavailable`，保留原始错误签名。
- 使用 canonical gomobile AAR 和 Go 门禁作为已取得的独立证据，但不把它们提升为 Gradle/APK/device 通过。
- 在可用 Android/JDK 环境重新运行两套 project 的 unit、lint、assemble，并在有设备时补 install/TCP/RFCOMM lifecycle smoke。

## Prevention / Guardrails

- Android gate 必须分开记录 Go、gomobile artifact、Gradle app 和 device 四层证据。
- Gradle 报错时先判断是否已经进入 project task；task 前 daemon IPC 失败优先按环境问题排查。
- 同时验证第二个最小/独立 Android project，避免把公共宿主故障误归因于当前改动。
- 无设备、无 `adb`、无 Gradle task evidence 时使用 `Unavailable`，禁止用模拟或较低层测试冒充通过。

## Related Docs

- [NodeHost runtime 与产品边界收敛 change](../change/2026-08-31_nodehost-runtime-and-product-boundaries.md)
- [Android feature](../features/android.md)
- [NodeHost Runtime spec](../specs/node-host-runtime.md)
- [Android runtime 与 mobile bindings](android-runtime-and-mobile-bindings.md)
- [Windows clean checkout、EOL 与 generated drift](windows-clean-checkout-eol-and-generated-drift.md)
