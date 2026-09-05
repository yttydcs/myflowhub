# Android 与嵌入式重新设计

## 状态

Deferred。2026-09-06 已移除现有 Android 与嵌入式源码、移动绑定和平台构建门禁；尚未选择替代实现，不承诺重启时间。

## 长期目标

电脑、平板和受限硬件上的能力最终仍可通过同一网络协作。网络只认识节点、网络和资源：一个物理设备可以运行多个节点，节点可以提供邮箱、感知或其他能力；设备、插件不是网络模型中的新对象。

平台实现承担 OS、存储、权限和功耗限制。它们不决定资源权限或创造独立的路由、寻址模型。具体语言、轻量 runtime、移动平台组合方式、支持的链路和首发能力需重新讨论。

## 待办

- ANDROID-REDESIGN：重新定义 Android 节点的运行边界、前后台行为、受保护身份、平台能力适配与可验证的最小闭环。
- EMBEDDED-REDESIGN：明确实际芯片、内存/存储/功耗预算、身份保存与入网方式、链路承载和最小资源能力，再选择实现。
- 恢复任何平台前，补齐与共享协议的 contract 测试、可靠构建与针对目标平台的验证；不能把旧历史测试记录作为新实现通过的证据。

## 当前范围与兼容性

现有源码不再提供 Android APK/AAR、C SDK、MicroPython SDK 或 ESP32 固件。删除不清除用户本机 SDK、已安装应用或运行状态文件；旧实现可按 Git 历史追溯，但不是受支持的构建输入。

Windows Desktop、Metrics、Clipboard 继续独立运行。资源调用与订阅能力保留；SDK/bindings 不再兼容自行构建或关闭节点的旧 API，调用方应由 NodeHost 管理生命周期。

## 完成本轮清理的验收

- 当前构建、依赖、CI 和生成契约不再要求已退役平台。
- Clipboard 使用单一 NodeHost，保留持久身份、配置、历史和同步行为。
- SDK/bindings 只操作现有节点；关闭 bindings 只清理本 facade 的订阅。
- 当前 features/specs 与源码一致，历史与重新设计待办可追溯。

## 关联

- [原始请求](../intake/2026-09-06_android-embedded-retirement-and-runtime-convergence.md)
- [决策](../decisions/2026-09-06_retire-mobile-embedded-and-owning-bindings.md)
- [统一运行时需求](unified-node-runtime.md)
- [Flow 重新设计](flow-redesign.md)
