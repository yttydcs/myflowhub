# Android、嵌入式退役与运行时收敛

## 原始请求与上下文

用户希望建立通用协作平台；剪贴板、通知和自动化只是其上的能力。网络模型只有节点、网络和资源，节点不等同于设备，插件可作为某个节点的实现细节。

在移除早期 Flow 后，用户明确提出：“嵌入式部分的可以考虑直接删除，后续可以重新设计，然后安卓也可以全部移除掉”，并询问旧 bindings 运行时入口的含义。讨论解释了旧入口自行管理身份、Node 和父连接，与 NodeHost + attached SDK 并存的问题，以及 Clipboard 尚依赖旧入口。

本轮确认：“采纳你的建议，可以继续移除，同时注意更新文档”，并指定使用 m-docs。

## 已确认范围

- 删除 Android 通用应用、Metrics/Clipboard Android 应用及移动 bindings。
- 删除 C、ESP32、MicroPython 实现和专属测试/构建/CI；保留 Git 与迁移历史。
- Windows Clipboard 改用 NodeHost 后，删除 SDK/bindings 的旧 owning 生命周期入口。
- 保留资源操作、订阅、独立 Enrollment bootstrap、通用网络核心和现有 Windows 产品。
- Android 与嵌入式重新设计列为待办；不在本轮决定新语言、Flow 模型或替代移动架构。

## 稳定文档影响

产品现状进入 features，重新设计目标进入 requirement，API ownership 和构建范围更新 specs；历史 plan/change 保持原有证据。

- [重新设计待办](../requirements/mobile-embedded-redesign.md)
- [已采纳决策与方案对比](../decisions/2026-09-06_retire-mobile-embedded-and-owning-bindings.md)
- [NodeHost Runtime](../specs/node-host-runtime.md)
