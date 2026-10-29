# Desktop Node 与 Metrics Node 产品独立性

## Source

- 来源：2026-08-30 Codex 讨论及用户提供的界面截图。
- 截图中的确认表述：Desktop Node 与 Metrics Node 完全独立；两个 Node、两个进程、两个安装包、边界清晰。
- 上下文：此前曾讨论是否把 MetricsNode 合并到 Desktop，用户进一步明确不采用该方向。

## Request Summary

Desktop 与 Metrics 不只是两个可选功能模块，而是两个相互独立的 Node 产品：

- 分别拥有 Node identity、准入、权限和父链关系；
- 分别运行在独立进程中；
- 分别持有配置、凭证、运行状态和日志；
- 分别提供可独立安装、升级、停止和卸载的产品包；
- 任一产品都不能把另一产品作为必装依赖；
- Desktop 访问 Metrics 时，只能像访问其他 Node 一样通过 authoritative Node tree、Resource capability 和正常权限裁决。

## Clarified Boundary

```text
Desktop package                         Metrics package
      │                                       │
Desktop process                         Metrics process
      │                                       │
Desktop NodeID                          Metrics NodeID
      └──────── authoritative Node tree ──────┘
                 normal policy only
```

“完全独立”约束的是产品和运行时边界，不禁止在 canonical monorepo 中共享协议、SDK、UI primitive、构建工具或其他无状态基础代码。共享源码不能演变为共享 Node identity、共享可变状态、隐式权限或强制共同安装。

Desktop 可以发现、显示和操作获授权的 Metrics Resource，但它不是 Metrics 的宿主进程、资源 owner 或默认 authority。Metrics 也不能依赖 Desktop 才能完成采集、执行、入网或后台运行。

## Current Evidence

- Desktop 与 Metrics 已有独立 Wails 产品目录和独立无 UI 命令入口。
- 根构建脚本分别产出 `mfh-desktop.exe` 与 `mfh-metrics.exe`。
- 当前尚未把“两个独立安装包、独立升级/卸载”固化为已验证发布契约，因此不能仅凭两个二进制宣称该目标完全实现。

## Confirmed Requirements

- Node、进程、状态、凭证、生命周期和安装包全部独立。
- 同机安装不产生共享身份、默认父子关系或额外权限。
- 两个产品可以单独安装和运行，也可以同时安装和运行。
- 一个产品崩溃、重启、升级或卸载不得破坏另一个产品的身份和持久状态。
- Desktop 与 Metrics 的交互必须通过 canonical 网络资源边界，不新增仅因同机部署而存在的权限旁路。
- 产品独立不等于代码复制；共享 Core/SDK 和构建基础设施仍然允许。

## Routed Documents

- 长期需求：[Desktop 与 Metrics 产品独立性](../requirements/desktop-metrics-product-independence.md)
- 架构决策：[Desktop 与 Metrics 保持独立 Node、进程和安装包](../decisions/2026-08-30_desktop-metrics-independent-products.md)
- 当前产品事实：[Desktop](../features/desktop.md)、[MetricsNode](../features/metrics-node.md)

