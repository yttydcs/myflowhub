# File transfer vNext

## Resources

- `file/transfers` Variable：当前节点可见的有界 transfer summaries。
- `file/progress` Stream：带 transfer ID、received bytes、total bytes 和 terminal state 的事件。
- `file/offer`、`file/chunk`、`file/complete`、`file/cancel` Commands。

## Limits and validation

所有 payload 使用 version 1 schema。transfer ID 为 128-bit 随机值；路径为相对 UTF-8 路径；size、chunk size、active sessions、临时总字节和 metadata 项数均由配置上限约束。chunk 输入包含 offset、data 与 SHA-256；整个文件 complete 时再次验证 size 与 SHA-256。

同 transfer ID、offset、length、digest 和 content 的重复 chunk 返回相同成功结果。相同 offset 的不同内容、超出声明 size、gap complete、已取消/完成会话或过期会话明确失败。Command output 只返回状态，不回传大块文件数据。

## Storage

路径在固定根目录内解析并拒绝 escape。接收写入 owner-only 临时文件；complete 后 fsync（平台支持时）并原子 rename。重启扫描并清理过期临时会话，不把不完整文件暴露为完成状态。

## Related

- [File transfer feature dossier](../features/file-transfer.md)
- [Command vNext](command-vnext.md)
