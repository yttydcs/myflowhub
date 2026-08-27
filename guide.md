1. commit 提交信息请使用中文，前缀标记（例如 feat、fix）可以使用英文。
2. 界面验证可以使用可用的浏览器自动化工具，但不能以浏览器 smoke 替代前端单测和生产构建。
3. 当前有效架构与协议文档统一由 `docs/README.md` 索引；历史多仓文档只用于追溯，不是实现依据。
4. 所有 worktree 必须创建在 `D:\project\MyFlowHub3\worktrees` 中，不得在 `repo` 或其子目录内创建。
5. canonical 实现始终设置 `GOWORK=off`；旧 `repo\MyFlowHub-*` 已移除，仅通过 `migration/` 中的远端和提交审计追溯，不得作为构建、运行或 fallback 输入恢复。
