# Win VarPool Node Vars `list` 超时修复（走 Hub 缓存）

## 变更背景 / 目标
- 背景：在拓扑 `2 -> 1 -> 9 -> 11(metrics)` 下，Win 端 Devices 的 Node Vars 读取 Node 11 变量名时出现 `varpool list: request timed out`。
- 目标：将 Node Vars 的 `list` 请求对齐 VarStore 规范，改为发往直接父/Hub 缓存路径，而非直发 owner 节点。

## 具体变更内容
- 修改：
  - `frontend/src/stores/varpool.ts`
    - 新增 `resolveHubTargetId()`：强制读取 `state.defaultTargetId`（登录下发的 hubId），并做正数校验。
    - `listOwnerNames(ownerId)` 的 `ListSimple` 调用从 `target=owner` 改为 `target=hubId`。
    - 保持 payload `owner` 不变（仍由 `data.owner` 指向目标节点）。
- 新增：
  - 本变更归档文档。

## Plan 任务映射
- `NODEVARS-1`：完成。`listOwnerNames` 改为走 Hub target。
- `NODEVARS-2`：完成。执行前端构建验证并记录结果。
- `NODEVARS-3`：完成。完成 Code Review 检查项与归档。

## 关键设计决策与权衡
- 选择：Node Vars 的 `list` 固定使用 hubId，不复用可编辑的 `Target ID`。
  - 原因：规范语义是“查询发父/Hub”；复用可编辑 target 容易被误设为 owner，导致 leaf 无回包再次超时。
  - 性能：Hub 命中缓存可直接回包，减少无效转发与超时等待。
  - 可扩展性：后续若引入 owner->Hub 映射或多级路由，仍可在 Hub 侧演进，不需要改 UI 端 owner 查询入口语义。

## 测试与验证方式 / 结果
- 执行：`npm ci`（frontend）  
  - 结果：通过。
- 执行：`npm run build`（frontend）  
  - 结果：失败（仓库现状问题，非本次改动引入）。报错：`Could not resolve "../../wailsjs/go/main/App"`。
- 执行：`npx tsc --noEmit`（frontend）  
  - 结果：失败（仓库现状问题，缺少 `wailsjs` 与 `.vue` 类型解析上下文）。

## Code Review 结论（3.3）
- 需求覆盖：通过。Node Vars `list` 已改为 Hub 缓存路径。
- 架构合理性：通过。与 `docs/3-varstore.md` 的 `get/list` 查询语义一致。
- 性能风险：通过。避免 list 直发 leaf 导致的无回包超时。
- 可读性与一致性：通过。新增 helper 命名清晰，改动集中。
- 可扩展性与配置化：通过。保留 owner 语义在 payload，不影响后端路由扩展。
- 稳定性与安全：通过。新增 hubId 校验，缺失时返回明确错误。
- 测试覆盖情况：部分通过。受仓库现状（wailsjs 生成物/类型环境）限制，未取得完整绿色构建。

## 潜在影响与回滚方案
- 潜在影响：
  - 未登录或未正确注入 hubId 时，Node Vars `list` 将提示 hub target 不可用（早失败，避免无意义超时）。
- 回滚方案：
  - 回滚 `frontend/src/stores/varpool.ts` 中本次提交（恢复 `target=owner` 的旧逻辑）。
