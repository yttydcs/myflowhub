# 2026-03-03 - Win：VarPool RefreshAll 跳过 not found 汇总提示 + Subscribe 不覆盖已有值 + 订阅按钮状态保持

## 变更背景 / 目标

VarPool（`#/varpool`）存在三个影响体验的问题：

1) 点击 `Refresh All` 时，若 watch 列表包含不存在的变量，会弹错 `not found (code=4)` 并中断刷新（历史遗留）。
2) 点击 `Subscribe` 后，部分变量原本已有值会变成 `-`（空值被覆盖）。
3) 已订阅变量离开再进入 `#/varpool` 后，按钮会回退成 `Subscribe`，但实际上订阅仍有效（仍能接收数据变化）。

目标是在**不修改协议/服务端**的前提下，修复上述 3 个问题，并保持行为可预测：
- `not found (code=4)` 跳过继续，结束时仅提示一次汇总。
- `Subscribe` 不强制额外 `Get`，但也不覆盖已有值。
- watch list reload / 再次进入页面时，订阅状态显示不回退。

## 具体变更内容

### 前端（UI：Refresh All 汇总提示）
- `frontend/src/pages/VarPool.vue`
  - `refreshAll()` 改为逐项 `try/catch`：
    - 识别 `code=4`（not found）并跳过继续；
    - 统计 `not found` 与其它 `failed` 数量；
    - 刷新结束后仅 toast 一次：无问题则 success；有问题则 warn（包含计数汇总）。

### 前端（store：Subscribe 不覆盖 value + watch reload 保留缓存）
- `frontend/src/stores/varpool.ts`
  - `handleVarSubscribeResp()`：
    - 不再写入 `value`（订阅响应仅确认订阅成功），避免把已有值覆盖为空；
    - 仍会更新 `subKnown/subscribed` 与元信息（`owner/visibility/type`）。
  - `loadWatchList()`：
    - 不再清空 `state.data`；
    - 改为按新 watchList **prune**：删除不再 watch 的 `state.data[id]`，保留仍 watch 的缓存值与订阅标记，避免 UI 状态回退。

## plan.md 任务映射

- V1：修复 Refresh All：跳过 not found 并汇总提示
- V2：修复 Subscribe：subscribe_resp 不覆盖已有值
- V3：修复订阅按钮状态：loadWatchList 不清空 data，避免状态回退
- V4：回归与构建（go test / 前端 build）

## 关键设计决策与权衡

- **`code=4` 识别方式**：当前错误来自 Wails binding reject 的 error message，前端用字符串匹配 `"(code=4)"` / `code=4` 进行分类；优点是无需改后端与协议，缺点是对错误文案有耦合（后续如改错误格式需同步调整）。
- **subscribe_resp 不等同于取值**：订阅响应不保证携带 `value`（或可能为空），因此不写入 value；值更新交由 `varpool.changed` 推送或用户手动刷新/获取。
- **watchList reload 采用 prune**：避免清空导致 UI 订阅状态/缓存值丢失；同时删除不再 watch 的条目，降低跨 profile / 历史脏数据风险。

## 测试与验证方式 / 结果

### Go 单测

```powershell
$env:GOWORK='off'
go test ./... -count=1 -p 1
```

结果：通过。

### 前端构建

```powershell
cd frontend
npm ci
npm run build
```

结果：通过。

### 手工冒烟（建议）

1) watch 列表中添加若干不存在的变量 → 点 `Refresh All`：
   - 不再弹 `Failed to refresh VarPool data. not found (code=4)`；
   - 结束仅出现一次汇总 toast（示例：`not found: 3/10`）。
2) 对已有值的 watched key 点击 `Subscribe`：
   - 值不应变成 `-`；
   - 仍能接收后续 `varpool.changed` 更新。
3) 已订阅项离开再进入 `#/varpool`（或点 `Reload Saved`）：
   - 按钮仍应显示 `Unsubscribe`（不回退成 `Subscribe`）。

## 潜在影响与回滚方案

- 影响范围：
  - `Refresh All` 由“遇错即失败”改为“尽力刷新 + 汇总提示”，可能降低单次错误的显著性（但保留 warn 汇总与 console.warn）。
  - watchList reload 不再清空缓存：可能短暂保留已 watch key 的历史值（直到下一次变更/刷新覆盖）。
- 回滚：
  - 回滚本次提交即可恢复旧行为（逐项弹错/订阅响应覆盖/重载清空缓存）。

