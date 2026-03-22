# Plan - MyFlowHub-Win：VarPool RefreshAll 跳过 not found + Subscribe 不覆盖已有值 + 订阅按钮状态保持

## Workflow 信息
- 范围：单仓库（`MyFlowHub-Win`）
- 分支：`fix/varpool-refresh-subscribe-ui`
- Worktree：`d:\project\MyFlowHub3\worktrees\varpool-refresh-subscribe-ui\MyFlowHub-Win`
- Base：`main`
- 规范：
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文，前缀可英文）
  - `d:\project\MyFlowHub3` 根目录 `AGENTS.md`（阶段纪律、worktree 禁令等）

---

## 1) 需求分析（已确认）

### 背景
当前 VarPool 页面存在三个用户可见问题：
1) 点击 `Refresh All` 经常弹错：`Failed to refresh VarPool data. not found (code=4)`（历史遗留问题）。
2) 点击 `Subscribe` 后，原本已有值会被覆盖成 `-`（空值）。
3) 已订阅变量再次进入 `#/varpool` 后 UI 会变回 `Subscribe`（但实际上仍在订阅并能收到数据变化）。

### 目标
在不改协议/服务端的前提下，修复上述 3 个问题，提升 UI 一致性与容错体验。

### 范围（必须 / 不做）
- 必须：
  - `Refresh All`：遇到 `not found (code=4)` 时 **跳过并继续**；不逐项弹错误；结束时 toast **汇总一次**（not found/failed 计数）。
  - `Subscribe`：当 `subscribe_resp` 未携带 `value` 字段时，**不覆盖已有值**（不强制额外 `Get`）。
  - 订阅按钮状态：再次进入 VarPool / 点击 `Reload Saved` 后，已订阅项仍显示为 `Unsubscribe`（不再回退成 `Subscribe`）。
- 不做：
  - 不修改 MyFlowHub-Proto / Hub / Server 行为与 `code=4` 语义。
  - 不强制在 Subscribe 后自动 `Get`。

### 验收标准（MVP）
1) watch 若干不存在的 key → 点 `Refresh All`：
   - 不再弹 `Failed to refresh VarPool data. not found (code=4)`
   - 仅在结束时 toast 一次汇总（示例：`not found: 3/10`）。
2) 某 watched key 已显示非空值 → 点 `Subscribe`：
   - 值不变（不变成 `-`）。
3) 对已订阅项：
   - 离开再回到 `#/varpool`（或点 `Reload Saved`）后按钮仍为 `Unsubscribe`；
   - 且仍能接收数据变化（订阅真实有效）。

### 风险
- 将 `code=4` 静默跳过可能掩盖错误：通过“结束时一次汇总 toast”保留可见性。

---

## 2) 架构设计（分析）

### 总体方案
- 仅在前端进行修复（store + page），不改协议与 Go VarPoolService。
- 三个修复点：
  1) `Refresh All` 改为逐项容错：捕获错误并分类 `code=4`（not found）与其它错误（failed），结束后汇总提示一次。
  2) `subscribe_resp` 不写入 `value`：订阅响应仅作为“订阅成功确认”，只更新订阅状态与元信息，避免覆盖已有值（不强制额外 `Get`）。
  3) `loadWatchList()` 不再清空 `state.data`，改为“按新 watchList 修剪不存在条目”，保留已存在 key 的订阅标记/值，避免 UI 回退。

### 关键实现策略
- `code=4` 识别：前端错误消息字符串匹配 `"(code=4)"`（当前 Go 层会返回形如 `not found (code=4)`）。
- `subscribe_resp` 与 value：`handleVarSubscribeResp` 不写入 value（订阅响应不等同于取值响应）；如后端推送 `varpool.changed`，再由 `handleVarChanged` 更新值。
- watchList reload：保留 `state.data` 中仍被 watch 的 key；移除不再 watch 的 key 对应缓存，避免跨 profile/历史脏数据。

---

## 3.1) 计划拆分（Checklist）

### V1 - 修复 Refresh All：跳过 not found 并汇总提示
- 目标：`Refresh All` 不再因单个 `code=4` 打断；结束时 toast 一次汇总。
- 涉及文件（预期）：
  - `frontend/src/pages/VarPool.vue`
- 验收条件：
  - 验收标准 #1 通过。
- 测试点：
  - watchList 中混合存在/不存在变量，确保刷新后存在变量仍更新。
- 回滚点：
  - 回滚页面内刷新逻辑改动（不影响 store/协议）。

### V2 - 修复 Subscribe：subscribe_resp 不覆盖已有值
- 目标：订阅成功后不把已有值覆盖成空；订阅响应仅更新订阅状态/元信息，不写入 value（不强制额外 Get）。
- 涉及文件（预期）：
  - `frontend/src/stores/varpool.ts`
- 验收条件：
  - 验收标准 #2 通过。
- 测试点：
  - Subscribe 对已有值变量：value 保持；
  - Subscribe 后仍能收到 `varpool.changed` 更新。
- 回滚点：
  - 回滚 `handleVarSubscribeResp/parseResp` 改动。

### V3 - 修复订阅按钮状态：loadWatchList 不清空 data + UI 不回退
- 目标：再次进入 VarPool / Reload Saved 后仍显示 `Unsubscribe`（对已订阅项）。
- 涉及文件（预期）：
  - `frontend/src/stores/varpool.ts`（loadWatchList 改为 prune）
  - `frontend/src/pages/VarPool.vue`（如需，按钮判断逻辑微调）
- 验收条件：
  - 验收标准 #3 通过。
- 测试点：
  - Reload Saved 后订阅标记仍在；
  - profile 切换后不会显示前一个 profile 的旧缓存（应被 prune）。
- 回滚点：
  - 恢复 `loadWatchList` 原行为（清空 data），并回退 UI 判断改动。

### V4 - 回归与构建
- 目标：确保构建链路通过且无明显 UI 回归。
- 验收：
  - `cd worktrees/varpool-refresh-subscribe-ui/MyFlowHub-Win; $env:GOWORK='off'; go test ./... -count=1 -p 1`
  - `cd worktrees/varpool-refresh-subscribe-ui/MyFlowHub-Win/frontend; npm ci; npm run build`
  - 手工验收 #1~#3 全部通过。
- 回滚点：
  - 回滚本分支全部提交。

---

## 3.3) Code Review（完成编码后执行）
- 需求覆盖：三项问题均修复且符合“跳过继续/汇总一次/不强制 Get/状态不回退”
- 架构合理性：修复集中在 store/page；不改协议语义
- 性能风险：Refresh All 仍为顺序执行；无额外 N+1；无请求风暴
- 可读性与一致性：错误分类与汇总逻辑清晰；订阅响应不覆盖 value
- 稳定性与安全：未连接/未登录提示保持；异常不会崩溃
- 测试覆盖：前端 build + go test + 手工冒烟覆盖关键路径

---

## 4) 归档变更（完成 Review 后执行）
- 在当前 worktree 根目录下创建 `docs/change/` 并新增文档：
  - `docs/change/2026-03-03_varpool-refresh-subscribe-ui.md`
- 内容需包含：背景/目标、变更内容、plan 任务映射、关键决策（code=4 跳过策略、subscribe_resp 不写入 value、prune 策略）、测试结果、影响与回滚方案。
