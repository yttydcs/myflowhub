# Notification vNext

Notification 用 `notifications/events` Stream 和 `notifications/publish` Command 取代 TopicBus handler。可选 `notifications/config` Variable 仅保存有界、非敏感的本地配置。

事件 schema `mfh.notification.event.v1` 包含 version、event_id、channel、source、created_at、content_type、body 和 attributes。event ID 必须非零；channel 和 attributes 有数量/长度上限；body 受 resource descriptor 上限约束。

publish 输入 schema `mfh.notification.publish.v1` 不接受伪造 source 或 sequence。owner 在验证权限、channel、content type 和 body 后生成 event ID 与时间并发布。重复 request ID 服从 Command dedupe 语义。

订阅者通过 Stream sequence 识别 gap；Notification 不承诺永久历史。需要可靠状态的场景使用 Variable 或专门持久 feature，不扩大 Stream 语义。

权限至少区分 `notification.read`、`notification.publish` 和 `notification.configure`。

相关产品行为见 [Notification 使用边界](../features/desktop.md) 与 [ClipboardNode](../features/clipboard-node.md)。
