# 2026-03-03 - Win：VarPool 节点变量弹窗 + Watch 订阅偏好本地持久化

## 变更背景 / 目标

1) 在 `#/varpool` 与 `#/devices` 增加一个弹窗：输入/选择 **owner NodeID** 后，列出该节点下可见（public）的变量名列表，便于一键 **Add Watch**。
2) 将每个 watched key（`name#owner`）的订阅偏好（`subscribe=true/false`）**按 profile** 持久化到本地，并在 **session 重连/重新登录** 后自动恢复订阅，且不依赖用户打开 `#/varpool` 页面。

## 具体变更内容

### Go（App 本地存储）
- `app_varpool.go`
  - 新增本地存储 key：`varpool.sub_prefs`
  - 新增 API：
    - `VarPoolSubPrefs() ([]VarPoolSubPref, error)`
    - `SaveVarPoolSubPrefs(prefs []VarPoolSubPref) ([]VarPoolSubPref, error)`
  - 新增规范化逻辑：`name trim`、过滤非法项（空 name / owner=0）、按 `{name, owner}` 去重（后写覆盖前写）
- `app_varpool_test.go`
  - 新增单测覆盖：解析失败返回空、规范化 trim/filter/dedupe 行为

### 前端（store：持久化 + 自动恢复）
- `frontend/src/stores/varpool.ts`
  - 新增：`loadSubPrefs()` / `saveSubPrefs()`，并在 `Subscribe/Unsubscribe/AddWatch/RemoveWatch` 时 best-effort 持久化
  - 新增：`restoreDesiredSubscriptions({ concurrency })`，用于 session ready 后自动恢复订阅（默认并发 4，失败汇总提示）
  - 新增：`listOwnerNames(ownerId)`：`ListSimple(sourceID, targetID=ownerId, { owner: ownerId })`（弹窗 list 的核心能力）
  - 行为调整：`desiredSubscribe()` 默认值改为 `false`，仅当用户显式订阅/本地偏好为 true 时才认可 subscribe_resp

### 前端（UI：节点变量弹窗 + 两处入口）
- `frontend/src/components/varpool/NodeVarsDialog.vue`（新增）
  - Overlay 弹窗：owner NodeID 输入、Load、搜索、客户端分页（50/页）
  - 列表项仅提供 **Add Watch**（不自动 Subscribe）
- `frontend/src/pages/Devices.vue`
  - 每个节点行新增 `Vars` 按钮打开弹窗（owner 预填为该节点 NodeID）
- `frontend/src/pages/VarPool.vue`
  - `Watched Variables` 区域新增 `Node Vars` 按钮打开弹窗（owner 可编辑）
- `frontend/src/layout/AppShell.vue`
  - App 启动与 profile 切换时加载 `watchList + subPrefs`
  - 监听 session（connected + loggedIn + nodeId/hubId）触发自动恢复订阅（不依赖打开 `#/varpool`）

## plan.md 任务映射

- V1：Go - 订阅偏好持久化 API（`varpool.sub_prefs` + 单测）
- V2：前端 - varpool store 持久化 + 自动恢复订阅
- V3：前端 - NodeVarsDialog 弹窗（search + client paging + Add Watch）
- V4：接入 - Devices / VarPool / AppShell
- V5：bindings 同步 + 回归验收（go test / wails generate / 前端 build）

## 关键设计决策与权衡

- **list 仅 public + 无服务端分页**：受限于现有 list 能力；采用前端搜索与分页，避免协议/服务端改动。
- **list 使用 target=owner**：按需求直查 owner 节点（`targetID = owner NodeID`）。
- **Add Watch 不自动 Subscribe**：保持操作可控，订阅由用户在 VarPool 页面手动触发。
- **订阅偏好独立存储**：使用独立 key `varpool.sub_prefs`，避免改动既有 watch list 格式，降低迁移风险。
- **自动恢复并发受控**：恢复订阅并发限制为 4；失败仅汇总 toast，避免刷屏。

## 测试与验证方式 / 结果

### Go 单测

```powershell
$env:GOWORK='off'
go test ./... -count=1 -p 1
```

结果：通过。

### bindings 生成

```powershell
$env:GOWORK='off'
wails generate module
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

1) `#/devices`：选任意节点点击 `Vars` → Load 成功看到变量名列表 → 点 `Add Watch`。
2) `#/varpool`：watched 列表出现新变量 → 手动点击 `Subscribe`。
3) 重启应用 / 断线重连 / 重新登录：无需打开 `#/varpool`，订阅会自动恢复（可观察订阅标识/数据变化）。

## 潜在影响与回滚方案

- 影响范围：
  - 新增本地持久化数据：`varpool.sub_prefs`（profile-scoped）
  - session 恢复时会触发一定数量的订阅请求（已并发限制）
- 回滚：
  - 回滚本次变更即可恢复旧行为（不再持久化/自动恢复订阅）
  - 如需清理本地偏好，可删除对应 profile 的 `varpool.sub_prefs` 配置项（保留 watch list 不受影响）

