# 2026-03-25 MyFlowHub-Win MCP AI 客户端

## 当前阶段
- Stage 1 已完成
- Stage 2 已完成
- Stage 3.1 已完成
- Stage 3.2 已完成
- Stage 3.3 已完成
- Stage 4 已完成

## 仓库与执行上下文
- Control Repo: `D:\project\MyFlowHub3`
- Control Base branch: `master`
- Control Worktree branch: `feat/mcp-ai-client`
- Control Worktree path: `D:\project\MyFlowHub3\worktrees\feat-mcp-ai-client`
- Active control doc: `D:\project\MyFlowHub3\worktrees\feat-mcp-ai-client\todo.md`
- Participating Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Participating Base branch: `main`
- Participating Worktree branch: `feat/mcp-ai-client`
- Participating Worktree path: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client`
- Participating Plan Doc: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\todo.md`

## 使用 $m-docs 的文档路由结论
- 文档分类: `requirements` + `specs` + `plan`
- Requirements impact: add
- Specs impact: add
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\docs\requirements\mcp-client.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\docs\specs\mcp-client.md`
  - `D:\project\MyFlowHub3\docs\specs\management-config-layering.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`
- Related lessons: none
- Canonical destination:
  - Win 长期需求: `MyFlowHub-Win/docs/requirements/mcp-client.md`
  - Win 长期技术约束: `MyFlowHub-Win/docs/specs/mcp-client.md`
  - workflow 控制文档: 当前 control worktree 根 `todo.md` + Win worktree 根 `todo.md`
  - 完成后的结果归档: `MyFlowHub-Win/docs/change/`，并在 workflow 结束时按 workspace 规则合并到全局 `docs/change/`

## Stage 1 - 需求分析

### 目标
- 为 AI host 提供一个可通过 MCP `stdio` 调用的无界面 MyFlowHub 客户端。
- 该客户端独立连接 Hub，作为独立节点身份暴露变量读写与节点查询能力。
- 首版必须避免与现有 `MyFlowHub-Win` GUI 客户端的本地配置互相污染。

### 范围

#### Must
- 在 `MyFlowHub-Win` 仓内新增无界面可执行入口，不能依赖 Wails UI 启动。
- 通过 MCP `stdio` 暴露首版工具：
  - 会话：`connect` / `disconnect` / `status`
  - 认证：`register` / `login`
  - 节点查询：`list_nodes` / `node_info`
  - 变量：`list` / `get` / `set` / `revoke`
- 以独立本地配置目录保存 MCP 客户端自己的 settings 与 node keys。
- 首版默认作为独立节点出现，拥有独立 `node_id` / `display_name`。
- 写操作必须受显式开关保护，避免默认允许 AI 直接改变量。
- `stdout` 只能输出 MCP JSON-RPC，日志与调试信息只能写 `stderr`。

#### Optional
- 在首版中保留为后续扩展 `topicbus`、`config_get`、`subscribe/unsubscribe` 的接口空间。
- 支持通过启动参数设置默认 `endpoint` / `device_id` / `display_name` / `timeout`。

#### Out of Scope
- 不复用现有 GUI 进程或 GUI 已建立的 session。
- 首版不开放 `config_set`。
- 首版不驱动现有 Win 界面，也不增加新的 GUI 页面。
- 首版不实现“连接外部第三方 MCP Server”的通用 client。

### 使用场景
- AI host 启动 `myflowhub-mcp` 后，先连接 Hub，再执行 auth register/login。
- AI 查询当前节点列表，确定 Hub 或目标节点。
- AI 读取某个变量值、列出变量名、创建变量、修改变量、撤销变量。
- 用户同时运行 GUI Win 客户端与 MCP 客户端，两者各自独立在线。

### 功能需求
1. MCP 客户端必须可被标准 MCP host 以 `stdio` 模式拉起。
2. 客户端必须能维持一个长连接 session，而不是每次 tool 调用重新拨号。
3. 首版必须显式支持 `auth register/login`，因为未认证连接默认只能访问 `auth` 子协议。
4. 成功的 auth 响应必须更新进程内默认身份状态，供后续 tool 调用回退使用。
5. Tool 参数必须支持显式传入 `source_id` / `target_id`；未传时可按“最近 auth 状态 -> 启动默认值”回退。
6. `varstore set/revoke` 等写操作在 `allow_write=false` 时必须被本地拒绝。
7. 本地配置、settings、node keys 不能默认写入 GUI 客户端正在使用的目录。
8. Tool 错误必须明确区分：未连接、未认证、参数非法、权限不足、目标未找到、超时。

### 非功能需求
- 最小改动：优先复用 Win 已有 `session` / `auth` / `management` / `varpool` 服务。
- 可读性：新入口与 Wails 主程序边界清晰，避免把 MCP 逻辑散落到现有 GUI 入口。
- 可维护性：本地配置路径、tool 名称、参数解析和状态回退必须集中实现。
- 安全性：默认只读；写操作、敏感状态与日志输出行为可审计。
- 兼容性：不改变现有 Win GUI 行为，不引入 GUI 配置兼容负担。

### 输入 / 输出
- 输入:
  - 启动参数：`endpoint`、`config_dir`、`device_id`、`display_name`、`timeout`、`allow_write`
  - MCP tool 参数：会话、认证、管理和变量操作所需 JSON 参数
- 输出:
  - MCP `tools/list`
  - MCP `tools/call` 结构化结果
  - `stderr` 日志
  - 独立配置目录中的 settings / node keys

### 边界异常
- 已连接时重复 `connect`。
- 未连接时调用非 `session/auth` 工具。
- 未认证时直接调用 `management/varstore`。
- 首次 `register` 返回 `pending` / `rejected`。
- `login` 使用了错误的 `device_id`、`node_id` 或本地 key。
- `target_id` 缺失且当前没有可用的默认 `hub_id`。
- `allow_write=false` 时调用 `set/revoke`。
- 独立配置目录不存在或不可写。

### 验收标准
1. `go build ./cmd/myflowhub-mcp` 成功。
2. MCP host 能发现并调用首版工具。
3. 在真实 Hub 上可完成 `connect -> register/login -> list_nodes/node_info -> varstore list/get/set/revoke` 的基本链路。
4. MCP 客户端会在 Hub 中表现为独立节点，而不是复用 GUI Win 的节点身份。
5. MCP 客户端的 settings 与 keys 不会写入 GUI Win 默认配置目录。
6. `allow_write=false` 时，写工具被本地拒绝且错误可读。

### 风险
- Win 现有服务默认围绕 GUI store 设计，需要补一层无界面启动与独立 base dir 支持。
- 认证成功后的默认身份回退若处理不清晰，会造成后续 tool 参数语义混乱。
- MCP `stdout/stderr` 分流若处理不严，会破坏宿主与协议层通信。

## Stage 2 - 架构设计

### 总体方案
- 在 `MyFlowHub-Win` 仓新增 `cmd/myflowhub-mcp`，实现一个无界面、`stdio` 传输的 MCP 进程。
- 复用 Win 现有 `internal/services/session`、`auth`、`management`、`varpool`，补充无界面 bootstrap 与独立配置目录支持。
- 在进程内维护一个轻量 session/auth 状态，记录最近成功的 `device_id`、`node_id`、`hub_id`、`role`，作为后续 tool 的默认回退身份。

### 备选方案对比
- 方案 A: 落在 `MyFlowHub-Win` 仓并复用现有服务
  - 优点: 改动最小，现有 `auth/management/varpool` 逻辑可直接复用。
  - 缺点: 首版能力边界留在 Win 仓，未来若要下沉到 SDK 需再次抽象。
- 方案 B: 先在 `MyFlowHub-SDK` 做 typed clients，再实现 MCP 入口
  - 优点: 边界更纯。
  - 缺点: 本轮范围扩大，需要先补 typed clients 与状态管理。
- 方案 C: 在现有 GUI Win 进程中嵌入 MCP bridge
  - 不采用原因: 会话和配置强耦合，无法隔离节点身份和本地状态。

### 模块职责
- `cmd/myflowhub-mcp`
  - 解析启动参数
  - 初始化无界面 runtime
  - 启动 MCP `stdio` 循环
- `internal/...` 新增 headless runtime 组装层
  - 统一创建 logs、session、auth、management、varpool、store
  - 负责 `stderr` 日志与进程内状态
- `internal/storage`
  - 提供显式 base dir 的 store 构造能力
  - 保证 MCP 配置与 GUI 配置隔离
- MCP tool 层
  - 参数校验
  - source/target/default 回退
  - 响应与错误映射

### 数据 / 调用流
1. MCP host 通过 `stdin/stdout` 与 `myflowhub-mcp` 通信。
2. `myflowhub-mcp` 在启动后初始化 headless runtime。
3. `session_connect` 建立到 Hub 的长连接。
4. `auth_register/login` 走现有 auth 服务，成功后写入进程内 auth snapshot。
5. `management_*` / `varstore_*` 工具通过现有 Win 服务发起请求，并按 snapshot / 显式参数决定 `source_id` / `target_id`。
6. 所有业务日志写 `stderr`，MCP 响应只写 `stdout`。

### 接口草案
- 启动参数:
  - `--endpoint`
  - `--config-dir`
  - `--device-id`
  - `--display-name`
  - `--default-target`
  - `--timeout`
  - `--allow-write`
- 首版工具:
  - `myflowhub_session_status`
  - `myflowhub_session_connect`
  - `myflowhub_session_disconnect`
  - `myflowhub_auth_register`
  - `myflowhub_auth_login`
  - `myflowhub_management_list_nodes`
  - `myflowhub_management_node_info`
  - `myflowhub_varstore_list`
  - `myflowhub_varstore_get`
  - `myflowhub_varstore_set`
  - `myflowhub_varstore_revoke`

### 错误与安全
- `stdout` 保留给 MCP，不写普通日志。
- 写工具默认受 `allow_write` 保护。
- 未连接、未认证、缺省 `target_id`、参数非法等错误在本地优先失败。
- `config_set` 首版不暴露，避免与用户手动配置互相踩踏。

### 性能与测试策略
- 进程内复用长连接，避免每次 tool 重新拨号。
- 单测优先覆盖:
  - 配置目录解析与隔离
  - tool 参数校验
  - source/target 回退与 auth snapshot 更新
  - 写工具 gate
- 构建验证:
  - `go test ./... -count=1`
  - `go build ./cmd/myflowhub-mcp`

### 可扩展性设计点
- 后续可继续补 `topicbus`、`config_get`、`subscribe/unsubscribe`。
- 若后续把 typed client 下沉到 SDK，本轮 MCP tool 层应尽量保持薄封装。
- 配置键使用独立 `mcp.*` namespace，避免与 GUI `home.*` / `app.*` 语义耦合。

## Stage 3.1 - 计划

### 项目目标与当前状态
- 目标: 为 `MyFlowHub-Win` 增加一个可被 AI host 拉起的无界面 MCP 客户端首版。
- 当前状态:
  - Control worktree 与 Win worktree 已创建。
  - 已确认现有 Win 服务可直接复用 `session/auth/management/varpool`。
  - 已确认未认证连接默认只能访问 `auth` 子协议，因此首版必须纳入 `auth register/login`。
  - 已确认 Win store 默认使用 GUI 配置目录，若直接复用会与 GUI 有冲突风险，因此需要显式独立 base dir。
  - Win 仓当前缺少该能力的 requirements/specs，已在本阶段补入。

### 可执行 Checklist
- [x] DOCS-1 新增 Win MCP 客户端 requirements/specs 与索引
- [x] WIN-MCP-1 增加 headless runtime 与独立 store base dir
- [x] WIN-MCP-2 增加 MCP `stdio` 运行时与工具注册
- [x] WIN-MCP-3 接入 auth/session 状态与 management/varstore 工具
- [x] WIN-MCP-4 补齐测试与构建验证
- [x] ARCHIVE-1 归档 change / review 结果

### Task IDs
- `DOCS-1`
- `WIN-MCP-1`
- `WIN-MCP-2`
- `WIN-MCP-3`
- `WIN-MCP-4`
- `ARCHIVE-1`

### 任务明细

##### DOCS-1 - Win 稳定文档落盘
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\todo.md`
- Goal: 为 Win MCP 客户端建立 requirements/specs 真相来源与索引入口
- Files / Modules:
  - `docs/requirements/mcp-client.md`
  - `docs/specs/mcp-client.md`
  - `docs/requirements/README.md`
  - `docs/specs/README.md`
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\docs\requirements\mcp-client.md`
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\docs\specs\mcp-client.md`
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\docs\requirements\README.md`
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\docs\specs\README.md`
- Acceptance:
  - 稳定需求与技术边界已落文档
  - 索引可导航
- Test Points:
  - 文档自检与交叉链接检查
- Rollback:
  - 删除新增文档并恢复索引

##### WIN-MCP-1 - Headless Runtime 与独立配置目录
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\todo.md`
- Goal: 在不影响 GUI 主程序的前提下构建可复用的无界面运行时，并支持显式 `config_dir`
- Files / Modules:
  - `internal/storage/*`
  - 新增 headless runtime 组装模块
  - `cmd/myflowhub-mcp/*`
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\internal\storage\*.go`
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\internal\mcp*.go`
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\cmd\myflowhub-mcp\*.go`
- Acceptance:
  - headless runtime 可独立初始化 logs/session/auth/management/varpool/store
  - 可显式指定独立 base dir，不落到 GUI 默认目录
  - 日志不写入 MCP `stdout`
- Test Points:
  - base dir 解析单测
  - runtime 初始化单测
- Rollback:
  - 回退新增 runtime / storage 构造入口

##### WIN-MCP-2 - MCP `stdio` 运行时与工具注册
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\todo.md`
- Goal: 实现最小可用 MCP server，支持 `tools/list` / `tools/call`
- Files / Modules:
  - `cmd/myflowhub-mcp/*`
  - 新增 MCP protocol / tool registry 模块
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\cmd\myflowhub-mcp\*.go`
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\internal\mcp\*.go`
- Acceptance:
  - host 可正常初始化
  - 可枚举首版工具
  - `stdout` 仅输出 MCP 消息
- Test Points:
  - `tools/list` / `tools/call` 单测
  - 错误 JSON-RPC 路径单测
- Rollback:
  - 删除新增 MCP 入口与 registry

##### WIN-MCP-3 - Auth / Management / VarStore 工具接入
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\todo.md`
- Goal: 把现有 Win 服务封装为 MCP 工具，并维护默认 auth snapshot
- Files / Modules:
  - `internal/services/auth/*`
  - `internal/services/management/*`
  - `internal/services/varpool/*`
  - 新增 tool adapter / state 模块
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\internal\mcp\*.go`
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\internal\services\auth\*.go`
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\internal\services\management\*.go`
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\internal\services\varpool\*.go`
- Acceptance:
  - 支持 `auth register/login`
  - 成功 auth 后更新默认 `device_id/node_id/hub_id/role`
  - `management` / `varstore` 支持显式参数与默认回退
  - `set/revoke` 在 `allow_write=false` 时被本地拒绝
- Test Points:
  - auth snapshot 更新单测
  - source/target 回退单测
  - write gate 单测
- Rollback:
  - 回退新增 tool adapter 与状态模块

##### WIN-MCP-4 - 测试与构建验证
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client`
- Plan Path: `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\todo.md`
- Goal: 对首版 MCP 客户端做最小可交付验证
- Files / Modules:
  - 新增 / 修改单测
  - `README.md` 或命令使用说明
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\**\*_test.go`
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\README.md`
- Acceptance:
  - 目标测试通过
  - `go build ./cmd/myflowhub-mcp` 成功
- Test Points:
  - `go test ./... -count=1`
  - `go build ./cmd/myflowhub-mcp`
- Rollback:
  - 回退新增测试与说明变更

##### ARCHIVE-1 - Review 与归档
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\feat-mcp-ai-client`
- Plan Path: `D:\project\MyFlowHub3\worktrees\feat-mcp-ai-client\todo.md`
- Goal: 完成 review、change 归档与必要 lessons 判断
- Files / Modules:
  - `docs/change/*`
  - 需要时的 `docs/lessons/*`
- Write Set:
  - `D:\project\MyFlowHub3\worktrees\feat-mcp-ai-client\docs\change\*.md`
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\docs\change\*.md`
- Acceptance:
  - review 记录完整
  - change 文档可追溯
  - lessons 是否需要已明确记录
- Test Points:
  - 文档自检
- Rollback:
  - 删除新增归档并恢复索引

### 依赖
- `MyFlowHub-Win` 现有 `internal/services/session/auth/management/varpool`
- `MyFlowHub-SDK` 的 `await` / `session`
- 真实或本地可连通的 Hub 环境

### 风险与备注
- `auth register/login` 成功后的默认状态若不集中管理，会导致后续工具参数语义漂移。
- 若首版把过多 UI store 语义直接搬进 MCP runtime，会把 GUI 偏好污染到无界面场景。
- 如需新增服务端协议或 SDK typed client，应回到 `3.1` 先扩 plan，不直接在 `3.2` 扩面。

### 并行评估
- 本轮不派发子 Agent。
- 原因:
  - 写集高度重叠，核心文件集中在 `internal/storage`、新 MCP runtime 与现有服务适配层。
  - 关键风险点在配置隔离、auth 状态和协议输出分流，适合主Agent连续完成。

阻塞：否
进入 3.2

## Stage 3.2 - 实施结果
- 已在 `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client` 完成 `cmd/myflowhub-mcp`、`internal/mcpapp`、`internal/mcp` 与 `internal/storage` 扩展。
- 已完成首版工具：
  - `session status/connect/disconnect`
  - `auth register/login`
  - `management list_nodes/node_info`
  - `varstore list/get/set/revoke`
- 已完成独立 `mcp.*` 配置与 `stderr` 日志分流。

## Stage 3.3 - Review
- review 结果：全部通过
- 关键结论：
  - 要求覆盖、分层边界、写 gate、安全分流、测试覆盖均满足计划要求
  - 未使用子 Agent，治理记录与理由已保留在计划文档

## Stage 4 - Archive
- 使用 `$m-docs` 完成 impact 检查：
  - Requirements impact: updated
  - Specs impact: updated
  - Lessons impact: none
- 归档产物：
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\docs\change\2026-03-25_win-mcp-ai-client.md`

## 验证
- `$env:GOWORK='off'; go test ./... -count=1`
  - 结果：通过
- `$env:GOWORK='off'; go build -o (Join-Path $env:TEMP 'myflowhub-mcp.exe') ./cmd/myflowhub-mcp`
  - 结果：通过
- 进程级 smoke：
  - `initialize` + `tools/list` 直接返回正确 MCP 响应

阻塞：否
Stage 4 已完成
等待用户确认是否结束 workflow

## Round 2 - MCP 启动脚本

### Stage 1 / 2 结论
- 新需求：在 `scripts/` 下增加一个可单独启动 MCP CLI 的脚本。
- 设计：新增 `scripts/start-myflowhub-mcp.ps1`，固定从 repo root 调用 `go run ./cmd/myflowhub-mcp`，并透传参数。

### Stage 3.1
- Requirements impact: none
- Specs impact: none
- Related requirements:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\docs\requirements\mcp-client.md`
- Related specs:
  - `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\docs\specs\mcp-client.md`
- Related lessons:
  - none
- Task IDs:
  - `SCRIPT-1`
  - `SCRIPT-2`
  - `SCRIPT-3`

阻塞：否
进入 3.2

## Round 2 - 实施结果
- `SCRIPT-1`
  - 已新增 `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\scripts\start-myflowhub-mcp.ps1`
- `SCRIPT-2`
  - 已更新 `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\README.md`
- `SCRIPT-3`
  - 已完成脚本级 smoke 与 `docs/change` 归档

## Round 2 - 验证
- `powershell -ExecutionPolicy Bypass -File scripts/start-myflowhub-mcp.ps1 --version`
  - 结果：通过
- 通过脚本直连 `initialize` + `tools/list`
  - 结果：返回正确 MCP 响应

## Round 2 - 归档
- `D:\project\MyFlowHub3\worktrees\win-mcp-ai-client\docs\change\2026-03-25_win-mcp-start-script.md`

阻塞：否
可执行 workflow 结束收口
