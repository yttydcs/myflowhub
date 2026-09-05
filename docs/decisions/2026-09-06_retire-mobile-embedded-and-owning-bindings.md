# 移除移动与嵌入式旧实现，收敛 SDK 生命周期

## 状态与依据

Accepted，2026-09-06。用户已确认清理范围，见[原始请求](../intake/2026-09-06_android-embedded-retirement-and-runtime-convergence.md)。本决策取代早期 NodeHost ADR 中 Android 单文件组合例外和 owning SDK 暂存条款。

## 方案对比

| 范围 | 此前方案 | 本轮采用方案 | 代价 |
| --- | --- | --- | --- |
| Android | 通用 Client/Host、Metrics、Clipboard 三套应用与移动绑定 | 删除实现与构建门禁，重新设计待办 | 当前不再构建 APK/AAR |
| 嵌入式 | C、ESP32、MicroPython 并行跟随协议 | 删除实现与专属夹具、测试、CI | 恢复需重新定义约束和验证 |
| Clipboard | Go SDK 拥有 Node 和父连接 | 产品持有 NodeHost，SDK 只操作同一 Node | 更新启动/关闭和连接观察接口 |
| SDK/bindings | owning 与 attached 两种路径并存 | 只保留 attached 资源操作；Enrollment bootstrap 独立 | 移除旧 API 的源码兼容性 |

选择该方案是为了减少协议和模型演进时需同步维护的实现。NodeHost 已被 Desktop、Metrics 使用，Clipboard 迁移后不再需要旧 owning 路径。

## 边界与后果

- NodeHost 管理普通节点的身份、Node、ParentSupervisor、Listener 和退出。
- Go SDK 删除 NewClient、Close、Connect、ConnectManaged 及 owning Connection；连接以只读 ConnectionStatus 暴露。
- 通用 bindings 删除 NewClient/NewEnrollmentClient、TrustParent、StartTCP/StartRFCOMM/StartEnrolledTCP；保留 attached Client 与独立 EnrollmentBootstrap。
- bindings.Close 只取消本地 facade 的订阅；bootstrap.Close 只结束入网握手，不停止普通 Node。
- Windows Clipboard 资源在 Host.Start 前注册，关闭和失败回滚由产品按 ownership 顺序完成；身份与业务持久文件布局保留。
- Android 专属 RFCOMM provider 删除，Windows/Linux RFCOMM、QUIC 和通用 Driver 契约保留。Hub 自身迁移到 NodeHost 仍为独立工作。
- Legacy Join 仍用于现有 Windows/CLI 身份路径，不因平台退役被删除。
- 不恢复 Flow，不决定内部语言，不把设备或插件加入网络模型。

## 追溯与后续

历史 feature 说明标记为退役，旧 plan/change 与 migration 记录保留。Android、嵌入式重启条件见[重新设计需求](../requirements/mobile-embedded-redesign.md)。后续实现应重新评估旧方案，不能机械恢复目录后宣布完成。

当前契约：[NodeHost](../specs/node-host-runtime.md)、[构建与 CI](../specs/build-and-ci.md)、[模块边界](../specs/repository-and-module-boundaries.md)。
