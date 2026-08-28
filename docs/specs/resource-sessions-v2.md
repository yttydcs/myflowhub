# Resource Sessions v2

## Status

这是已实现的长时、双向或大数据 Resource 操作 canonical contract，取代
`file-transfer-vnext.md` 中把 chunk 作为普通 Command 的数据面设计。

## Lifecycle

Session Resource 先通过控制 lane 接收 `open`，验证 subject、resource、capability、schema、
limits 和策略后返回短期 grant。grant 绑定 subject、resource、link、topology epoch、policy
generation 和 expiry，不得跨 Node、Profile 或连接重用。

状态为 `opening -> active -> closing -> closed`，也可因 expire、revoke、link loss、checksum
failure 或 protocol error 进入 closed。close 必须幂等，任何终止路径都释放队列、token 和临时文件。

## Lanes and bounds

LinkSession 提供小而优先的 control lane，以及按 session 隔离的 bounded data lane。调度采用公平优先：
控制消息能越过普通数据积压，但持续控制流不能永久饿死已获 grant 的数据会话。

每个 grant 明确最大 chunk、最大总字节、队列容量、并发数和 deadline。超过限制时返回稳定错误，
不得通过扩容或静默丢弃继续运行。

## File Resource

File 的 `file/upload` Resource 通过 `open` capability 建立会话；chunk 携带 offset、长度和 checksum，在 data lane 传输。
重复同 offset/内容幂等，冲突内容、gap、越界或 checksum 错误明确失败。完整 size 和 digest 验证后，
临时文件以 atomic rename 提交；失败、取消、过期和重启清理未完成状态。progress 是 observable event。

路径始终解析在配置根内，拒绝绝对路径、dot segment、平台前缀、非法 UTF-8 和 symlink escape。

## Extension boundary

Media 可以复用 session open/control/lease 和 data-lane backpressure，但生产级 codec、capture、
playback、WebRTC/QUIC media QoS 不属于本规格的当前交付。
