# 2026-02-24 Win：自身节点 Config（settings.json）接入 management config_*（避免 Edit 超时）

## 变更背景 / 目标
在 Win 的 `Devices` 页面点击“自身节点”（`targetID == sourceID`）的 `Edit`（Config）后会超时。

根因（可审计）：
- `internal/services/management/service.go` 已对 self-node 实现了：
  - `node_info`（本地采集并返回）
  - `list_nodes` / `list_subtree`（对自身直接返回空列表，避免超时）
- 但 `config_list` / `config_get` / `config_set` 仍走 `sendAndAwait`，请求被发往自身后没有对应响应，最终触发 `defaultManagementTimeout` 超时。

本次目标：
1) self-node `config_list/get/set` 不再超时，且支持完整查看/编辑/保存；
2) 数据源使用 Win 本地 `settings.json`（`internal/storage`）；
3) 语义尽量对齐 Server 的 `management config_*`（Keys/Get/Set + 错误码风格），同时不影响远端节点的现有行为。

## 具体变更内容

### 修改
- `app.go`
  - 将 `store` 注入 `ManagementService`，使其具备 self-node 的本地 config 能力。
- `internal/services/management/service.go`
  - 为 self-node（`sourceID != 0 && sourceID == targetID`）增加本地短路：
    - `ConfigList`：返回 `settings.json` 中的 raw keys（排序后）。
    - `ConfigGet`：从 `settings.json` 读取 raw value 并格式化为 string 返回。
    - `ConfigSet`：将 value 以 string 写入 `settings.json` 并回显。
  - 新增 `formatConfigValue()`：将 `any` 统一格式化为 string（nil→`"null"`；number/bool→`fmt`；object/array→`json.Marshal`）。
- `internal/storage/store.go`
  - 新增 `Keys()` / `GetRaw()` / `SetRaw()`：
    - `Keys()`：稳定排序输出 keys（用于 UI 渲染与一致性）。
    - `GetRaw()`：读取 raw value（只读）。
    - `SetRaw()`：按 raw key 写入 settings（绕过 profile 前缀逻辑，匹配 management config_* 的“key 即真源”语义）。

### 不变（保持原行为）
- 非 self-node：`config_list/get/set` 仍走原有 `sendAndAwait` 链路（与远端节点交互逻辑不变）。

## 对应 plan.md 任务映射
- S2：Storage 暴露 settings.json keys/raw value（只读快照）
- S3：ManagementService self-node config_* 本地短路实现
- S4：构建与最小冒烟验证

## 关键设计决策与权衡
1) **保持“远端 config_*”不变，仅对 self-node 短路**
   - 降低变更面，避免引入协议/后端侧改动。
2) **self-node 的 config 真源采用 `settings.json`**
   - 与 Win 现有配置存储一致；同时满足“可落盘、可重启验证”。
3) **raw keys 全量暴露（含 profile 前缀与全局 key）**
   - 便于一致性与可审计；后续若需按 profile 过滤/搜索，需要另起需求（避免本次引入额外 UI/逻辑复杂度）。
4) **写入值按 string 保存（对齐 Server MapConfig 的 string value 语义）**
   - 现有 `Store.GetInt/GetBool` 仍能从 string 解析，兼容多数现有读取逻辑；
   - 若未来需要“保留原类型/自动 JSON 解析写入”，需另起任务并明确验收标准。

## 测试与验证方式 / 结果
### 命令级验证（通过）
- `GOWORK=off go test ./... -count=1`
- `GOWORK=off wails build -nopackage`
- `cd frontend && npm run build`

### 人工冒烟建议（关键路径）
1) 登录后进入 `Devices`，对自身节点点击 `Edit`：
   - keys 可展示，且 value 可逐条填充；
2) 修改任意 key 的 value → `Save`：
   - toast 成功；
   - `Refresh` 后仍为新值；
   - 重启 App 后值仍保留（验证落盘）。

## 潜在影响与回滚方案
### 潜在影响
- self-node 的 config keys 会包含更多“内部/历史”键（因为来自 `settings.json` 的 raw keys），可能导致列表较长。

### 回滚方案
- 回滚按文件 revert：
  1) 回退 `internal/services/management/service.go` 的 self-node 短路；
  2) 回退 `internal/storage/store.go` 新增的 raw 接口；
  3) 回退 `app.go` 的依赖注入签名改动。

