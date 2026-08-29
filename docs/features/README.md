# Features

存放 MyFlowHub3 当前面向用户的完整功能行为，是 UI、工作流、权限和状态的长期事实入口。

## How To Use
- 想了解某项功能现在如何端到端工作时从这里进入。
- API、协议和架构约束继续查看 `specs/`，不要在 feature 文档中重复维护。

## What Belongs Here
- 入口、布局、导航和页面状态
- 权限、校验、CRUD 与异常流程
- 跨仓职责和可验收场景

## Products

- [Hub](hub.md)：权威根节点、准入、树管理和系统资源。
- [Desktop](desktop.md)：桌面宿主与本地资源工作台。
- [Android](android.md)：Android 节点、前台服务和 RFCOMM 接入。
- [MetricsNode](metrics-node.md)：指标变量、通知与平台控制。
- [ClipboardNode](clipboard-node.md)：跨平台剪贴板同步与历史。
- [Embedded leaf SDK](embedded.md)：C、ESP32 和 MicroPython 叶子节点能力。

## Composed Capabilities

- [File transfer](file-transfer.md)：基于资源与指令的文件传输。
- [Flow](flow.md)：流程定义、运行、状态和取消。

## Rules
- 使用不带日期的稳定文件名。
- 一个用户功能只保留一份完整当前事实，并链接参与仓库。
- 新增叶子文档后同步更新本索引；技术契约只链接到 `specs/`。
- 历史旧仓和 `change/` 只作为来源证据，本目录描述迁移后的当前行为。
