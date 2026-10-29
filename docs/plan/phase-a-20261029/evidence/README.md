# A 阶段保留证据

这些是原候选的历史证据，验收范围和最终处置以 [review](../review.md) 为唯一入口。`current-package-verification.json` 中 current 表示当时包，不表示包含 2026-10-29 UI 修改的最终源码。R2 原始 Failed 结果未改写。

完整本机证据保存于 `D:/project/MyFlowHub3/archives/phase-a-20261029`：

- `workflow-local-evidence.zip`：原 out/、测试辅助程序及本机 build 产物；同路径查找历史引用。
- `private-native-main-evidence.zip`：主原生夹具原始记录，仅留本机。
- `historical-preview-package/`：原5个ZIP和包清单的本机保留副本，不含10月29日UI修改。
- `recovery-manifest.json`：恢复副本SHA-256清单。
- `private-native-evidence.zip`：本机原始时间线/截图/采样及私有夹具恢复记录，仅留本机，含私有状态，不发布。
- `main-preexisting.zip`、JSON 和 patch：合并前主检出45个已有改动文件的恢复副本，不属于交付内容。
- `local-evidence-manifest.json`：原工作树 evidence 文件的大小与SHA-256；ZIP完整性在归档前校验。

本目录摘要不包含私钥、Permit、令牌。后续复测不覆盖旧失败记录。
