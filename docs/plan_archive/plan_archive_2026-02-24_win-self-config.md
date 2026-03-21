# Plan - Win：自身节点 Config（settings.json）接入 Management config_*（避免 Edit 超时）

## Workflow 信息
- 范围：单仓库（`MyFlowHub-Win`）
- 分支：`fix/win-self-config`
- Worktree：`d:\project\MyFlowHub3\worktrees\win-self-config\MyFlowHub-Win`
- Base：`main`（当前 worktree 基于：`d43889a`）
- 规范：
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文，前缀可英文）

## 背景（问题清单与用户需求）
### 问题
在 Win 的 `Devices` 页面点击“自身节点”（`targetID == sourceID`）的 `Edit`（Config）后会超时。

根因（已定位，可审计）：
- `internal/services/management/service.go` 中：
  - `NodeInfo/ListNodes/ListSubtree` 已对 self-node 做了短路返回（不走 send+await）。
  - 但 `ConfigList/ConfigGet/ConfigSet` 没有 self-node 短路，仍走 `sendAndAwait`，导致请求被发往自身但没有对应响应，最终触发 `defaultManagementTimeout` 超时。

### 需求（已确认）
1) 自身节点 Config **完全支持编辑**（C）。
2) Config 数据源使用 Win 本地 `settings.json`（`internal/storage`）作为真源（A）。
3) 行为尽量与 Server 的 `management config_*` 接近（语义/错误码/Keys/Get/Set 模型）。

## 目标
1) `Devices → Edit` 点击自身节点时不再超时，Config keys 能正常列出，values 可读取/可保存。
2) 非自身节点（远端节点）的 Config 行为保持不变（仍走 send+await）。
3) 本地 Config 的 `Keys/Get/Set` 语义与 server 侧尽量一致：
   - `config_list`：返回可编辑/可读取的 key 列表；
   - `config_get`：不存在时返回 “not found (code=404)” 的等价错误语义；
   - `config_set`：写入成功后回显 key/value。

## 范围与约束
- 必须：
  - 仅修复自身节点 `config_list/get/set` 的超时与缺失响应问题。
  - 不改协议 wire（SubProto/Action/JSON schema 不变）。
  - 不影响远端节点 config_* 的现有链路。
- 不做：
  - 不新增 UI 搜索/过滤（如 keys 过多，后续另起需求）。
  - 不调整 `defaultManagementTimeout`（本次问题通过 self-node 短路解决）。

## 总体方案（简述）
在 Win 的 `ManagementService` 中为 `targetID == sourceID` 增加本地实现：
- 读取/枚举：从 `internal/storage.Store` 的 `settings.json` 内存镜像中获取（加锁后复制/遍历）。
- 写入：通过 `Store` 统一写盘（复用其“临时文件写入 + rename”策略），并更新内存状态。
- 对外仍返回 `protocol/management` 中的 `ConfigListResp/ConfigResp`，确保前端与远端逻辑一致。

## 3.1) 计划拆分（Checklist）

### S1 - Workspace 准备
- 目标：创建独占 worktree 与专业分支，确保不在 `repo/` 里做实现改动。
- 验收：
  - `git status -sb`：在 `fix/win-self-config`，工作区干净。
- 回滚点：`git worktree remove` + `git worktree prune` + 删除分支（若未推送）。

### S2 - Storage：暴露 settings.json 的可枚举 key 与 raw value（只读快照）
- 目标：
  - 为 `ManagementService` 提供枚举 keys 与读取 raw value 的能力（受锁保护，避免竞态）。
- 涉及文件（预期）：
  - `internal/storage/store.go`（或新增 `internal/storage/config.go`）
- 验收：
  - 可获取稳定排序的 key 列表；
  - 可按 key 读取到 raw value，并能格式化为 string（bool/number/object/array 也能返回可读字符串）。
- 测试点：
  - 并发读不 panic（加锁）。
  - key 不存在返回明确 not found。
- 回滚点：revert 对 storage 的新增接口。

### S3 - ManagementService：self-node ConfigList/Get/Set 本地短路实现
- 目标：
  - `targetID == sourceID && sourceID != 0` 时：
    - `ConfigList` 从本地 store 列举 keys，返回 `Code=1, Msg=ok`；
    - `ConfigGet` 读取本地 store，返回 `Code=1`；不存在时对齐 server 行为（404 not found）；
    - `ConfigSet` 写入本地 store，返回 `Code=1`。
  - 非 self-node：维持原 send+await 行为不变。
- 涉及文件（预期）：
  - `internal/services/management/service.go`
  - `app.go`（注入 `store` 到 `ManagementService`）
- 验收：
  - self-node 点击 Edit 不再超时；
  - 远端节点 Edit 行为不变。
- 回滚点：revert management/service.go 与 app.go 改动。

### S4 - 验证（构建 + 最小冒烟）
- 命令级验收：
  - `cd frontend && npm run build`
  - `$env:GOWORK='off'; wails build -nopackage`
- 人工冒烟（关键路径）：
  1) 登录后进入 `Devices`，对自身节点点 `Edit`：
     - keys 能显示，且 value 能逐条填充；
  2) 修改任意 key 的 value → `Save`：
     - toast 成功；
     - 再次 `Refresh` 后 value 保持更新；
     - 重启 App 后值仍保留（验证落盘）。
- 回滚点：逐提交 revert。

### S5 - Code Review（阶段 3.3）
- 按 AGENTS 3.3 清单逐项审查（需求覆盖/架构/性能/一致性/安全/测试）。

### S6 - 归档变更（阶段 4）
- 新增文档：`docs/change/2026-02-24_win-self-config.md`
- 必须包含：
  - 背景/目标、变更清单、与 plan 任务映射、关键设计权衡（尤其 keys 范围与类型格式化）、验证方式/结果、回滚方案。

### S7 - 结束确认后合并与清理
- 在 `repo/MyFlowHub-Win`：
  - `git merge --ff-only origin/fix/win-self-config`（或本地分支）并 push（按你需要）
- 将 worktree 中的 `docs/change` 迁移到 `d:\project\MyFlowHub3\docs/change`
- 将 worktree 的 `plan.md` 归档到 `d:\project\MyFlowHub3\docs/plan_archive`
- 清理：`git worktree remove` + `git worktree prune`，并删除 `worktrees/win-self-config` 空目录

