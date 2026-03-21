# 2026-02-18 Win / Management：从“解析 session.frame”收敛为“业务编排 + 返回值 API”

## 变更背景 / 目标

依据 `d:\project\MyFlowHub3\repos.md`（约 L171）的目标态要求：Win services 应继续收敛为“业务编排 + UI 友好错误/日志”，并尽量让协议细节下沉到 SDK/Proto。

此前 `management` 模块存在的问题：

- 前端 `frontend/src/stores/management.ts` 依赖 `session.frame`，自行解码 payload 并 `switch(action)` 更新 UI 状态。
- 这种做法导致：
  - UI 与 wire 强耦合（action 名称/JSON 结构在 UI 侧扩散）；
  - 重复解码/解析逻辑分散在多个 store，长期可维护性差；
  - 业务错误体验不统一（更多依赖“帧流”而非明确的 req/resp 返回值）。

本次目标（先以 management 作为样板）：

- Wails API 直接返回 `myflowhub-proto/protocol/management` 的响应结构体（resp），UI 通过返回值更新状态；
- `session.frame` 事件保留用于 Debug/观测，但 management UI 不再依赖它做业务更新；
- 常见错误（超时、未连接等）对 UI 更友好，同时在 logs 中保留可定位信息。

## 具体变更内容

### 新增 / 修改

1) **Go：ManagementService 返回值化**

- 文件：`internal/services/management/service.go`
- 将以下方法从“仅返回 error”调整为“返回 resp + error”：
  - `NodeEcho*` → `management.NodeEchoResp`
  - `ListNodes*` → `management.ListNodesResp`
  - `ListSubtree*` → `management.ListSubtreeResp`
  - `ConfigList*` → `management.ConfigListResp`
  - `ConfigGet*` / `ConfigSet*` → `management.ConfigResp`
- 统一规则：
  - 输入校验失败：直接返回 `error`
  - await/解包失败：返回 `error`，同时写入 logs（含 action 与原因）
  - `code != 1`：返回 `error`（`msg` 优先）并写 warn 日志

2) **前端：Management store 移除 frame 解析，改用返回值更新 UI**

- 文件：`frontend/src/stores/management.ts`
- 移除对 `session.frame` 的监听与 payload 解码（不再 `EventsOn("session.frame")`、不再 `JSON.parse(payload)`、不再 `switch(action)`）。
- `listNodes/listSubtree/selectNode/refreshConfig/setConfig` 调用对应 Wails binding，基于 resp 更新：
  - nodes 列表
  - config keys 与 values
  - UI message
- 增加轻量竞态保护：异步返回时若 `selectedNodeId` 已变化，则丢弃过期结果。

### 删除

- 无（`session.frame` 仍保留，仅 management UI 不再依赖它做业务更新）。

## 对应 plan.md 任务映射

- WINMGMT1：Go：ManagementService 返回 resp + error（业务编排层）
- WINMGMT2：前端：Management store 迁移到返回值（移除 frame 解析）
- WINMGMT3：冒烟验证步骤（手工）
- WINMGMT4：Code Review

## 关键设计决策与权衡

### 为什么选择“返回值（方式 A）”而不是“事件化/继续 frame 解析”

- Management 属于典型 req/resp 协议：返回值是最直观的 UI 数据流，改动面最小，能快速落地“UI 与 wire 解耦”。
- `session.frame` 继续保留为 Debug/观测通道，避免影响现有调试能力。
- 后续对 notify/订阅类协议（更偏事件流）再考虑 Go 侧事件化收敛，更贴合协议特性。

### 性能关键点

- 前端不再对每个帧做 decode + JSON.parse + switch，减少无谓解析与分支。
- ConfigGet 仍是多次请求（与现状一致）；若 keys 极大，后续可按需增加并发上限/批量接口（本次不扩大范围）。

### 安全与日志

- Go 侧在 await/解包/code!=1 时写入 logs，便于定位问题（action + 失败原因）。
- 仍避免在 service 层主动打印完整 config value（底层 TX/RX payload log 属于既有行为，本次不改）。

## 测试与验证方式 / 结果

### Go（Windows）

- `GOWORK=off go test ./... -count=1 -p 1`（通过）

### 前端（本地生成 wailsjs 后）

> `frontend/wailsjs/` 在 `.gitignore` 中，需由 wails 生成（或在 `wails dev/build` 时自动生成）。

- `GOWORK=off wails generate module`
- `cd frontend && npm ci && npm run build`（通过）

### 冒烟（手工，连接 server）

1) 启动 `hub_server`（确保 management handler 可用）
2) 启动 Win：`wails dev`
3) Connect → 登录成功（Session 页面应显示 nodeId/hubId）
4) 打开 Management：
   - List Direct/Subtree 能看到 nodes
   - 选择 node：keys 出现，values 逐步填充
   - Edit 保存：提示成功且 value 更新

## 潜在影响与回滚方案

### 潜在影响

- Management 的 Wails binding 现在返回结构体：旧前端若仍假设“无返回值”也不会报错（Promise resolve 带对象），但已同步更新本仓前端逻辑。
- 若服务端响应 schema 与 Proto 定义不一致，会在 Go 解包阶段直接报错（更早暴露协议问题）。

### 回滚

- `git revert` 本次提交（或回退合并提交），即可恢复为“仅返回 error + 前端解析 frame”的旧模式。

