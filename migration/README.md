# Legacy Source Retirement

`migration/` 是 10 个已退役 MyFlowHub 旧仓库的可审计迁移记录，不是构建输入或旧实现 fallback。

## Files

- `sources.yaml`：来源仓库、branch、固定 commit、canonical target 与退役状态；
- `source-audit.json`：固定 commit/tree、tracked file count，以及删除前唯一未提交生成物的逐文件 SHA-256；
- `inventory.json`：每项能力和构建入口的 migrate/replace/drop 结论与验证状态；
- `audit-sources.ps1`：删除前使用的来源一致性检查器，仅在临时恢复全部旧 checkout 时有意义。

## Recovery

已提交内容可从 `sources.yaml` 指定的远端仓库 checkout 固定 commit 恢复。旧 `MyFlowHub-MetricsNode/windows/frontend/wailsjs` 的 29 个未提交文件是可再生成绑定；审计只保留其路径、状态和 hash，不保留文件正文，因此不能从本仓恢复原始字节。

如需历史调查，应在 workspace 外的临时目录 checkout 精确提交，禁止把旧 module、SubProto 或生成绑定重新接入 canonical build。当前实现依据始终从 [../docs/README.md](../docs/README.md) 进入。
