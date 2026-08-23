# 2026-08-24 剩余 worktree 成果救援与清理

## 变更背景 / 目标

在上一轮平衡清理后，对 6 个剩余 MyFlowHub worktree 逐一核对提交时间、相对主线差异、完成记录和当前产品价值。目标是把仍有价值且尚未进入主分支的成果精确合入，并删除已被主线取代、仅剩生成物或过期控制文档的 worktree 与本地分支。

## 具体变更内容

- `CTRL-DEMO`：保留 ESP32-S3 双板演示流程包、参数模板、生成脚本、示例 JSON 和变更记录。
- `CTRL-REL`：从旧发布控制分支保留跨仓发布链结果记录，不合入过期 `todo.md`。
- `AND-AUTH`：保留 Android 对 SDK typed 认证 API 的迁移、密钥测试、依赖更新和文档。
- `EMB-DHT`：只保留 DHT11 默认采样间隔 `5000 ms -> 2000 ms` 及变更记录；丢弃自动下载的 `managed_components`。
- `WIN-AUTH`：保留 Win 对 SDK typed 认证 API 的迁移、会话访问器、测试、依赖更新和文档。
- `CLEAN-ESP`：删除已经被 EmbeddedSDK 主线实现取代的 ESP32 SDK 规划 worktree，不保留重复需求/规格草稿。
- `CLOSEOUT`：主分支验证通过后移除上述 6 个 worktree，并删除对应本地分支。

## Docs root

- `D:\project\MyFlowHub3\docs`
- 归档与代码提交均保持 local-only；不推送远端、不创建标签、不选择发布或备份目标。
- 控制仓文档索引已有无关未提交修改，本轮只精确暂存本记录及对应索引行。

## Intake impact

- Intake impact: none

## Feature impact

- Feature impact: none

## Requirements impact

- Requirements impact: none

## Specs impact

- Specs impact: none

## Decision impact

- Decision impact: none

## Lessons impact

- Lessons impact: updated
- Android 与 Win 仓分别归档 `docs/lessons/sdk-typed-pseudo-version.md`，记录 SDK 新 typed API 尚无正式 tag 时使用精确伪版本的约束。

## Related intake

- none

## Related features

- none

## Related requirements

- EmbeddedSDK canonical MVP requirement: `repo/MyFlowHub-EmbeddedSDK/docs/requirements/embedded-sdk-mvp.md`

## Related specs

- none

## Related decisions

- none

## Related lessons

- `repo/MyFlowHub-Android/docs/lessons/sdk-typed-pseudo-version.md`
- `repo/MyFlowHub-Win/docs/lessons/sdk-typed-pseudo-version.md`

## 对应 `plan.md` 任务映射

- `CTRL-DEMO`：演示流程包与结果文档救援。
- `CTRL-REL`：跨仓发布链结果文档救援。
- `AND-AUTH`：Android typed auth 迁移。
- `EMB-DHT`：EmbeddedSDK DHT11 默认采样间隔调整。
- `WIN-AUTH`：Win typed auth 迁移。
- `CLEAN-ESP`：过期 ESP32 规划清理。
- `CLOSEOUT`：跨仓主分支集成、验证、worktree 与本地分支清理。

## 经验 / 教训摘要

- worktree 的价值不能只看提交时间或落后数量；必须同时比较未提交树、主线是否已有等价实现、完成记录和当前接口契约。
- 救援旧 worktree 时应按最小写集提交，避免把 `plan.md`、`todo.md`、生成依赖目录或已经失效的控制文档带入主线。
- 当主工作区已有无关脏状态时，可使用精确路径和索引补丁只暂存本轮内容，不整体暂存共享索引文件。

## 可复用排查线索

- 症状：旧 worktree 显示“已完成”，但主分支没有对应提交。
- 触发条件：成果只存在于 worktree 未提交文件，或控制分支只记录发布/计划结果。
- 关键词：`git worktree list --porcelain`、`git status --short`、`git rev-list --left-right --count`、`git diff <base>`、`git branch --contains`。
- 快速检查：先比较 branch head，再单独审计 dirty tree；确认主线等价实现后才删除。

## 关键设计决策与权衡

1. Android、Win、DHT 先在原 worktree 形成最小提交，再由各自干净主工作区快进，便于验证和安全删除分支。
2. 控制仓主工作区存在大量无关修改；Demo 分支提交不触碰共享索引，发布记录和三条索引使用精确暂存。
3. SDK typed API 当前提交尚无正式 tag；下游固定到精确伪版本，不擅自创建或推送 tag。
4. 未注册的实体目录不属于本轮 Git worktree 收口范围，保持不动。

## 测试与验证方式 / 结果

- Demo：PowerShell 生成脚本语法解析通过；4 个 JSON 文件均可解析。
- Android：目标 Go 文件 `gofmt` 检查通过；主分支执行 `go test -mod=readonly ./... -count=1 -p 1` 通过。
- EmbeddedSDK：主分支快进完成，目标提交差异检查通过，Kconfig 默认值为 `2000`；不把自动下载组件纳入版本控制。
- Win：主分支执行 `go test -mod=readonly ./... -count=1` 全量通过。
- 所有目标提交均执行 `git diff --check`，未发现空白错误。

## 潜在影响

- 6 个 worktree 中未被纳入最小提交的过期计划、待办和生成物会随强制移除而消失。
- Android/Win 使用 SDK 精确伪版本，后续正式发布 SDK tag 时可再切换到正式语义版本。
- 所有合并和提交仅存在于本地，远端不会自动获得这些结果。

## 回滚方案

- 已集成提交可在对应主分支使用新的反向提交回滚；不重写主分支历史。
- 清理前记录各分支 tip；必要时可执行 `git branch <branch> <tip>` 并重新 `git worktree add`。
- 控制仓索引只回滚本轮三条条目和对应新增文档，不重置其他用户修改。

## 子Agent执行轨迹

- none；跨仓索引存在写集交叉，本轮由主 Agent 串行执行。

## 清理前恢复账本

| Owner | Branch | Tip / recovery basis | Resolution |
| --- | --- | --- | --- |
| Control | `feat/embedded-demo-flows` | `d839aee` | 合入控制仓后安全删除 |
| Control | `chore/release-chain-20260412-control` | `93d27fc` | 救援发布记录后受控强制删除 |
| Control | `feat/esp32-embedded-sdk-plan` | `4568f20` | 等价实现已在 EmbeddedSDK 主线，安全删除 |
| Android | `refactor/sdk-auth-typed` | `93d2032` | 快进主分支后安全删除 |
| EmbeddedSDK | `fix/embedded-dht11-interval` | `343d5b6` | 快进主分支后安全删除 |
| Win | `refactor/sdk-auth-typed` | `606b747` | 快进主分支后安全删除 |
