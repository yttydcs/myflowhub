# Android

## 定位

Android 当前保留显式 Node ID、父节点公钥和既有 Join Permit 的兼容路径，不发送新的 `MFHE` Enrollment。无 Node ID 的首次注册由 Desktop/Go 路径先行落地，Android 迁移需单独批准。

Android 产品有两个明确模式：

- **Client**：作为普通子节点加入父链，浏览和操作任意授权节点的资源；
- **Host**：在应用进程中承载 persistent Hub，为移动或蓝牙邻近节点提供管理、通知、File 和 Flow resources，并可继续作为子节点连接上级。

两种模式使用同一 `runtime/node`、authority tree、ParentSupervisor 与资源模型。Android 不是一套独立协议实现。

## 源码边界

- `sdk/bindings/android`：gomobile-friendly Client、Host、Listener 与 RFCOMM provider contract；
- `apps/android`：Compose UI、前台服务、Bluetooth Classic adapter 和 versioned settings；
- `transport/rfcomm`：跨平台 Driver、endpoint parser、MTU chunking 与 Android provider adapter。

```text
Compose / foreground service
          │
          ▼
generated gomobile Client / Host
          │
          ├── TCP Driver
          └── RFCOMM Driver ── generated provider interface ── BluetoothSocket
          │
          ▼
authority tree + catalog + Variable / Stream / Command
```

## Client contract

Client 暴露 persistent identity、parent trust、TCP/RFCOMM start、managed reconnect state、catalog、snapshot、callback subscription、Command、File upload 和 close。订阅由 Go durable subscription 负责重新建立，Kotlin 不重放 wire frame。

File 页面先通过 Android content resolver 把用户选定文档复制到随机 cache 文件，再由 SDK 执行 64 KiB 分块、逐块/最终 SHA-256、complete 和失败 cancel。临时文件在成功或失败后删除。

Flow 页面读取 `flow/definitions`、`flow/runs`，并调用 create/update/run/cancel/archive Commands。MetricsNode、ClipboardNode 和未来节点应用通过相同 catalog/resource 页面接入，不硬编码旧 TopicBus/VarStore 协议。

## Local Hub Host

Host 使用 `host/hub.StartPersistent`，因此本地 identity、trust、policy、admission、settings、File 和 Flow state 都保存在应用私有目录。可同时开启 TCP 与 RFCOMM listener；至少一个 listener 必须有效。父公钥与一次性 permit 必须在启动/连接边界设置，运行后不能静默更换 trust。

Host status 包含实际 listener endpoints 与 parent connection state。Android 前台服务每两秒刷新状态，通知只显示 runtime 生命周期，不记录资源 payload。

## RFCOMM 与权限

gomobile AAR 直接生成 `RFCOMMProvider`、`RFCOMMListener`、`RFCOMMPipe` Java 接口，Kotlin 直接实现；禁止用反射容忍 binding 漂移。

- Android 12+ 启动前检查 `BLUETOOTH_CONNECT`，UI 请求 CONNECT/SCAN；
- 无 Bluetooth Classic、蓝牙关闭、权限拒绝、无效地址、dial/listen/accept 错误均明确失败；
- Go Driver 继续执行 endpoint、UUID、channel 和 MTU 校验；
- 系统蓝牙权限只授权访问硬件，不授予 MyFlowHub resource 权限；join permit 和父节点 policy 仍是 authority 边界。

## 本地设置

连接设置使用 SharedPreferences 单键 JSON version 1。只接受已知字段、positive NodeID、`client|host` 模式和 `tcp|rfcomm` transport，并限制 endpoint/key/permit 大小。

未知版本、未知字段或损坏 JSON 不会回退默认值，而是进入显式恢复页。用户必须输入 `RESET ANDROID V1`；reset 只替换 UI/connection settings，Go identity、trust、policy、admission 状态不删除。

## Build

AAR 是生成产物且被仓库 `*.aar` 规则忽略：

```powershell
$env:GOWORK='off'
gomobile bind '-target=android/arm64,android/amd64' -androidapi 26 `
  -javapkg com.myflowhub.mobile `
  -o apps/android/app/libs/myflowhub.aar `
  ./sdk/bindings/android

cd apps/android
./gradlew testDebugUnitTest assembleDebug lintDebug
```

Gradle 缺少 AAR 时立即失败，不生成“可编译但运行时不可用”的 stub APK。debug APK 必须包含 `arm64-v8a/libgojni.so` 与 `x86_64/libgojni.so`。

## 验证边界

- Go：Client/Host lifecycle、persistent identity、真实 TCP Hub permit/join、race、Android cross-compile；
- JVM/Gradle：settings version/reset tests、Compose compile、assemble、lint；
- artifact：双 ABI AAR/APK 原生库检查；
- device：只有实际连接设备时才记录 install/TCP/RFCOMM socket smoke。没有设备时明确标为 unavailable，不用 host-side provider test 冒充设备证据。

## 明确移除

- nested `hubmobile` module 和 sibling `replace`；
- TopicBus、VarStore 与旧 Management service wrappers；
- 反射定位 gomobile class/method；
- AAR 不存在时的 stub fallback；
- Android permission 等同于资源权限的错误假设。
