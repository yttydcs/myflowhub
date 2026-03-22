# 2026-03-22 Node Display Name Follow-up

## 变更背景 / 目标

此前 `management-node-display-name` 主线已经补齐了节点显示名的基础 schema、Win 标题展示和 SubProto 的名称返回能力，但仍存在三个会让用户困惑的缺口：

- Win `Devices` 的现有 `Edit` 配置弹窗在首次命名时看不到 `node.display_name`
- Win 自身节点的本地 `node_info` 没有和配置标题复用同一份名称
- 直连父节点缺少稳定的名称 bootstrap / rename refresh，`list_nodes` 容易回退成裸 `node_id`

本轮 follow-up 的目标是补齐这些缺口，同时继续保持 `list_nodes` 低成本直连枚举模型。

## 具体变更内容

- `MyFlowHub-Win`
  - 在 `Devices` 的 Config 列表中 synthetic 暴露 `node.display_name`
  - Config 标题优先读取当前已加载的 `node.display_name`
  - self `node_info` 返回本地 `display_name`
  - auth register/login JSON 可选携带 `display_name`
  - 补充 auth / management 相关单测
- `MyFlowHub-SubProto`
  - auth 本地兼容 wire struct 接收 / 回显可选 `display_name`
  - direct-child login bootstrap 成功后缓存连接 metadata 中的 `display_name`
  - direct-child `config_set_resp(node.display_name)` 成功回程时刷新连接 metadata
  - 补充 auth / management 单测
- `MyFlowHub-Server`
  - `hubruntime` 父链 register payload 读取当前 `node.display_name` 并按非空发送
  - 更新 `docs/specs/auth.md` 对可选 `display_name` 字段的说明
- 控制仓文档
  - 澄清 requirement/spec 与当前 follow-up 的归档边界

## 对应计划任务映射

- `CTRL1`
- `WIN1`
- `WIN2`
- `SUB1`
- `SUB2`
- `SRV1`
- `INT1`
- `REV1`
- `ARC1`

## 关键设计决策与权衡

- 不把 `list_nodes` 变成逐 child `node_info` fan-out，继续复用连接 metadata 作为直连 child 名称缓存。
- auth 扩展使用本地兼容 JSON struct，而不是在本轮继续拉高 Proto 依赖升级面。
- `config_set_resp` refresh 只在 direct-child 源判定成立时更新 metadata，避免把后代 rename 误写到中间连接。
- authority 侧的 `assist_register` 阶段在首次绑定前缺少稳定的“child 本人”判定，因此父节点名称 bootstrap 依赖 `login` 和后续 rename refresh 补全。

## Requirements / Specs 影响检查

- Requirements impact：`updated`
- Specs impact：`updated`
- Related requirements：
  - [management-node-display-name.md](/D:/project/MyFlowHub3/worktrees/MyFlowHub3-feat-node-display-name-followup/docs/requirements/management-node-display-name.md)
- Related specs：
  - [management-config-layering.md](/D:/project/MyFlowHub3/worktrees/MyFlowHub3-feat-node-display-name-followup/docs/specs/management-config-layering.md)
  - [auth.md](/D:/project/MyFlowHub3/worktrees/MyFlowHub-Server-feat-node-display-name-followup/docs/specs/auth.md)
- Lessons：`none`

## 测试与验证方式 / 结果

- Win：
  - `GOWORK=off go test ./... -count=1` 通过
  - `npm run build` 失败
  - 阻塞点：`frontend/src/pages/Home.vue` 依赖的 `../../wailsjs/go/session/SessionService` 缺失，属于仓内既有生成物问题
- SubProto：
  - `auth`: `GOWORK=off go test ./... -count=1` 通过
  - `management`: 通过临时 `modfile + replace` 指向本地 `exec` module 后 `go test ./... -count=1` 通过
  - 直接 `GOWORK=off go test ./...` 仍失败，原因是已发布 `exec` 依赖缺少 `exec/capability` 包
- Server：
  - `GOWORK=off go test ./hubruntime -count=1` 通过
  - `GOWORK=off go test ./... -count=1` 失败
  - 阻塞点：`protocol/exec/types.go` 依赖的 Proto 类型在当前仓内即已不匹配，与本次 `hubruntime` 改动无关

## 潜在影响与回滚方案

### 潜在影响

- Win 前端 build 仍然无法作为完整发布验收，需要先补 `wailsjs` 生成物。
- SubProto `management` module 的独立 `GOWORK=off` 测试仍受既有 published `exec` 版本阻塞。
- authority 侧在 `assist_register` 阶段不会盲目缓存 `display_name`，首次命名显示依赖后续 login 或 rename refresh。

### 回滚方案

- 回退各仓对应 change 文档中列出的实现文件。
- 若仅需撤销名称缓存刷新，可单独回退 `MyFlowHub-SubProto/management/management.go`。

## 子 Agent 执行轨迹

- `WIN1` / `WIN2` -> `Erdos (019d1618-0ff3-7312-96d1-bda0ab0c8c39)` -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-node-display-name-followup`
  - 文件：`frontend/src/stores/management.ts`、`frontend/src/pages/Devices.vue`、`internal/services/auth/service.go`、`internal/services/management/service.go`、`app.go`、Win 测试文件
  - 验收：Win Go 测试通过；前端 build 失败点已确认为既有环境问题
- `SUB1` / `SUB2` -> `Arendt (019d1618-11cf-7923-8e62-babfc05af8bb)` 初始执行超时，主Agent接管收口并补齐清空旧 metadata 与 assist 保护 -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-feat-node-display-name-followup`
  - 文件：`auth/types.go`、`auth/actions_register.go`、`auth/actions_login.go`、`auth/session.go`、`management/management.go` 与相关测试
  - 验收：`auth` 与 `management` 关键验证完成；`management` 的 `GOWORK=off` 阻塞已确认为依赖解析问题
- `SRV1` -> `Copernicus (019d1618-11ec-7632-82b9-8bd406d8b22a)` -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-Server-feat-node-display-name-followup`
  - 文件：`hubruntime/runtime.go`、`hubruntime/runtime_test.go`
  - 验收：`go test ./hubruntime -count=1` 通过；整仓失败点已确认为仓内既有 `protocol/exec` 构建问题
