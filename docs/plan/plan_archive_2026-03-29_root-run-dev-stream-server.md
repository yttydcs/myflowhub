# Plan - run-dev Stream Server 启动路径修复

## Workflow Information
- Repo: `D:\project\MyFlowHub3`
- Branch: `fix/run-dev-stream-server`
- Base: `master @ db93ba6`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server`
- Current Stage: `4`

## Stage Records

### Initialization
- guide.md: `D:\project\MyFlowHub3\guide.md` 存在，已确认「commit 信息使用中文，worktree 必须创建在 D:\project\MyFlowHub3\worktrees 中」。
- base/worktree confirmation:
  - 控制面主工作区存在未提交改动：`plan.md`、`go.work.sum`、`docs/lessons/authority-local-admin-actions.md`
  - 本次执行面使用独立 worktree：`D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server`
  - 关联只读依赖 worktree：`D:\project\MyFlowHub3\worktrees\server-stream-subproto-design`

### Stage 1 - Requirements Analysis
#### Goal
- 修复 workspace 根 `scripts/run-dev.ps1` 启动后 `Stream` 页面仍对 Hub `target=1` 发起 `list_sources / list_consumers / announce` 并超时的问题。
- 让默认本地启动路径能够真正拉起带 `stream` handler 的 Hub，并避免被根 `go.work` 污染回旧依赖。

#### Scope
- 必须
  - 修复 `scripts/run-dev.ps1` 的 Server 路径解析逻辑。
  - 对所选 Server 的 `stream` 支持能力做最小校验或显式提示。
  - 对非主线 Server worktree 启动默认启用 `GOWORK=off`。
  - 更新 workflow 文档与可复用排查线索。
- 可选
  - 增加显式参数，允许手动指定 Server 项目目录。
  - 输出更明确的启动摘要，便于确认当前实际使用的 Server 路径与 `GOWORK` 模式。
- 不做
  - 不在本轮把 `stream` 合并到 `repo/MyFlowHub-Server` 主线。
  - 不改 `MyFlowHub-Win` UI 或 `stream` 业务协议。
  - 不改 `Proto / SubProto` 主线发布链。

#### Use Cases
- 用户在 workspace 根执行 `.\\scripts\\run-dev.ps1` 后，Win `Stream` 页面新增本地 source 不再因为 Hub 缺少 `stream` handler 而超时。
- 用户需要继续使用现有主线 Server 做普通冒烟时，脚本能明确提示当前 Server 是否支持 `stream`。
- 用户需要手动切换其他 Server 目录时，可以显式指定路径，并在脚本输出里看到实际生效路径。

#### Functional Requirements
- 脚本必须能解析并选择实际用于启动的 Server 项目目录。
- 默认路径若不支持 `stream`，脚本必须优先尝试已知的 `stream` worktree 候选。
- 若最终选中的 Server 仍不支持 `stream`，脚本必须输出明确警告，不允许继续“静默成功”地诱发 UI 超时误判。
- 当选中的 Server 目录不在根 `go.work` 的 module 列表中时，脚本必须默认让该 Server 进程使用 `GOWORK=off`。
- 脚本输出必须展示实际 Server 路径、是否具备 `stream` 支持，以及 Server 的 `GOWORK` 模式。

#### Non-functional Requirements
- 保持改动面最小，限定在 workspace 根脚本与文档。
- 不依赖硬编码的绝对用户目录；候选路径必须相对 workspace root 解析。
- 失败必须显式，不能吞错。

#### Inputs / Outputs
- 输入
  - `run-dev.ps1` 参数
  - workspace root 下的 `repo/MyFlowHub-Server`
  - workspace root 下可能存在的 `worktrees/server-stream-subproto-design`
- 输出
  - 新的 Server 选择与启动行为
  - 启动摘要与警告
  - `docs/change` 归档
  - 必要时的 `docs/lessons` 排查文档

#### Edge Cases
- `stream` worktree 不存在。
- 用户显式指定的 Server 目录不存在或缺少 `cmd/hub_server`。
- 选中 worktree Server 但未关闭 `GOWORK` 时，会被根 `go.work` 污染导致编译/运行回退。
- 未来主线 Server 已补齐 `stream` 时，脚本不应仍强依赖旧 worktree 名称。

#### Acceptance Criteria
- 默认执行 `.\\scripts\\run-dev.ps1 -SkipWin -SkipMetricsNode` 时，输出能显示实际选中的 Server 路径与 `stream` 支持状态。
- 若存在 `worktrees/server-stream-subproto-design` 且具备 `stream` handler，默认启动应优先使用该路径。
- 从 `server-stream-subproto-design` 工作树执行的验证命令在 `GOWORK=off` 下通过，且在默认 workspace 模式下能复现污染失败，用作脚本行为依据。
- 脚本帮助或输出足以指导用户确认“当前是否吃到 stream 更新”。

#### Risks
- 依赖当前 worktree 名称约定，未来目录变动时需要同步维护候选列表。
- 自动切换到 worktree Server 可能改变部分用户对“默认总是主线 repo”的预期，需要在输出中说明。

#### Issue List
- 无

### Stage 2 - Architecture Design
#### Overall Solution
- 在 root `scripts/run-dev.ps1` 内新增「Server 路径解析 + `stream` 支持探测」层：
  - 优先级：显式传参 > 主线 Server（若已支持 `stream`）> 已知 `stream` worktree 候选 > 主线 Server + 明确警告。
  - 通过文件存在性与默认集合代码特征做轻量探测，避免引入复杂解析。
- Server 启动时独立计算 `GOWORK` 策略：
  - `-GoWorkOff` 仍保持全局最高优先级。
  - 若 Server 目录不是根 `go.work` 中的主线 `repo/MyFlowHub-Server`，则仅对 Server 默认强制 `GOWORK=off`。

#### Alternatives Considered
- 方案 A：只增加 `-ServerProjectDir`，让用户自己切路径。
  - 不选：默认行为仍旧落到错误主线，不能解决当前“已经按脚本启动但仍 timeout”的问题。
- 方案 B：直接把 Server 主线补齐 `stream`。
  - 不选：会扩展到 `Proto / SubProto / Server` 发布链，不符合本轮最小修复面。
- 方案 C：默认自动优先 `stream` worktree，同时保留显式路径覆盖。
  - 采用：既解决当前问题，又保留显式控制权。

#### Module Responsibilities
- `scripts/run-dev.ps1`
  - 解析 Server 候选路径
  - 探测 `stream` 支持
  - 决定 Server 的 `GOWORK` 模式
  - 输出最终启动摘要和警告
- `docs/change/*`
  - 记录启动路径修复、验证依据和回滚点
- `docs/lessons/*`
  - 沉淀“worktree Server + root go.work 污染”的排查线索

#### Data / Call Flow
1. 解析 workspace root。
2. 生成主线 Server 路径、worktree Server 候选路径和可选显式参数路径。
3. 对候选路径做目录存在性、`cmd/hub_server` 与 `stream` 支持探测。
4. 选定最终 Server 目录，并计算 Server 专属 `GOWORK` 模式。
5. 输出实际路径 / 支持状态 / `GOWORK` 模式，再启动 `go run ./cmd/hub_server`。

#### Interface Drafts
- `run-dev.ps1`
  - 新增可选参数：`ServerProjectDir`
  - 新增 helper：
    - `Resolve-ServerProjectDir`
    - `Test-ServerSupportsStream`
    - `Get-ServerGoWorkMode`

#### Error Handling and Safety
- 显式路径不存在时直接报错。
- 若回退到不支持 `stream` 的主线 Server，脚本在启动前输出高可见 warning，说明 `Stream` 页面将出现哪些超时症状。
- 不捕获并吞掉 `go run` 启动错误。

#### Performance and Testing Strategy
- 性能
  - 仅做本地文件检查，不执行重型扫描。
- 测试
  - 脚本静态验证：`pwsh -File scripts/run-dev.ps1 -SkipServer -SkipWin -SkipMetricsNode`
  - 帮助输出：`Get-Help .\\scripts\\run-dev.ps1 -Detailed`
  - 定向验证：`go env GOWORK` 与 `go test` 在 `server-stream-subproto-design` 下分别验证 workspace 污染 / `GOWORK=off` 成功

#### Extensibility Design Points
- 候选路径列表集中管理，后续主线合并 `stream` 后可自然回退到主线 Server。
- `stream` 支持探测函数保持独立，后续若需要检测其它模块能力可复用。

#### Issue List
- 无

### Stage 3.1 - Planning
#### Project Goal and Current State
- 当前 `MyFlowHub-Win` 已补齐本地 owner 控制面，但 workspace 根 `run-dev.ps1` 仍固定启动 `repo/MyFlowHub-Server`。
- `repo/MyFlowHub-Server` 主线缺少 `stream` handler，因此 Win 对 `target=1` 的 `stream list_sources / list_consumers / announce` 控制面请求必然超时。
- `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design` 已具备 `stream` 依赖链与 `newStreamHandler(...)`，且 `GOWORK=off` 下定向 `go test` 通过。
- 该 worktree 若不显式 `GOWORK=off`，会被根 `go.work` 污染并直接构建失败。

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口。
- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `add`
- 归档目标：
  - workflow control: `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server\todo.md`
  - change archive: `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server\docs\change\2026-03-29_root-run-dev-stream-server.md`
  - lesson: `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server\docs\lessons\run-dev-stream-server-selection.md`

#### Related Requirements / Specs / Lessons
- Related requirements
  - `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design\docs\requirements\stream.md`
- Related specs
  - `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design\docs\specs\stream.md`
- Related lessons
  - `D:\project\MyFlowHub3\worktrees\server-stream-subproto-design\docs\lessons\stream-control-plane-validation.md`
  - `D:\project\MyFlowHub3\docs\lessons\wails-binding-proto-drift.md`

#### Executable Task List
- [x] `RDS-1` 解析 Server 候选路径并默认优先 `stream` worktree
- [x] `RDS-2` 为非主线 Server 启动默认启用 `GOWORK=off`，并输出路径 / 支持状态 / 模式摘要
- [x] `RDS-3` 完成脚本帮助与定向验证
- [x] `RDS-4` 归档 `docs/change` 与 `docs/lessons`

#### Task Details
##### `RDS-1` - Server 路径解析与 `stream` 能力探测
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server\todo.md`
- Goal: 让默认启动选择到具备 `stream` handler 的 Server 项目目录，并保留显式路径覆盖。
- Files / Modules:
  - `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server\scripts\run-dev.ps1`
- Write Set:
  - `scripts/run-dev.ps1`
- Acceptance:
  - 默认可识别 `worktrees/server-stream-subproto-design`
  - 输出实际 Server 路径
  - 回退到不支持 `stream` 时有显式 warning
- Test Points:
  - `pwsh -File scripts/run-dev.ps1 -SkipWin -SkipMetricsNode`
  - `Get-Help .\\scripts\\run-dev.ps1 -Detailed`
- Rollback:
  - 回退 `scripts/run-dev.ps1` 本次改动

##### `RDS-2` - Server 专属 `GOWORK` 模式保护
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server\todo.md`
- Goal: 避免 `stream` Server worktree 被根 `go.work` 污染。
- Files / Modules:
  - `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server\scripts\run-dev.ps1`
- Write Set:
  - `scripts/run-dev.ps1`
- Acceptance:
  - 选中 worktree Server 时，脚本显示 Server `GOWORK=off`
  - 主线 Server 保持既有语义，除非用户显式 `-GoWorkOff`
- Test Points:
  - `go env GOWORK`（server-stream worktree）
  - `go test ./tests -run TestStreamRootHubConnectDisconnect -count=1` with / without `GOWORK=off`
- Rollback:
  - 回退 Server `GOWORK` 计算逻辑

##### `RDS-3` - 验证与说明输出
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server\todo.md`
- Goal: 保证脚本帮助和启动摘要足够让用户确认是否“吃到更新”。
- Files / Modules:
  - `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server\scripts\run-dev.ps1`
- Write Set:
  - `scripts/run-dev.ps1`
- Acceptance:
  - 帮助示例包含新参数或新默认行为说明
  - 启动时可见当前 Server 路径、`stream` 支持、Server `GOWORK`
- Test Points:
  - `Get-Help .\\scripts\\run-dev.ps1 -Detailed`
  - `pwsh -File scripts/run-dev.ps1 -SkipServer -SkipWin -SkipMetricsNode`
- Rollback:
  - 回退新增说明文本

##### `RDS-4` - 文档归档与复盘线索
- Owner: 主Agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server\todo.md`
- Goal: 把“脚本路径选错 + `go.work` 污染”的排查线索沉淀到 root docs。
- Files / Modules:
  - `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server\docs\change\2026-03-29_root-run-dev-stream-server.md`
  - `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server\docs\change\README.md`
  - `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server\docs\lessons\run-dev-stream-server-selection.md`
  - `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server\docs\lessons\README.md`
- Write Set:
  - `docs/change/*`
  - `docs/lessons/*`
- Acceptance:
  - change / lesson 均可独立说明症状、触发条件、关键词、快速检查
  - README 索引更新
- Test Points:
  - 人工校对路径与引用
- Rollback:
  - 删除新增归档并回退 README 更新

#### Dependencies
- 只读依赖：`D:\project\MyFlowHub3\worktrees\server-stream-subproto-design`
- 验证命令依赖本机 `go`、`pwsh`

#### Risks and Notes
- 当前 root `go.work` 不包含 `server-stream-subproto-design`，因此 Server worktree 启动必须脱离 workspace。
- 若未来主线 `repo/MyFlowHub-Server` 已合入 `stream`，脚本应自动优先主线，不应永久依赖 worktree 名称。

#### Parallelism Assessment
- 本轮不派发子Agent。
- 原因：改动面仅一处脚本，验证与文档都直接依赖同一份上下文，串行更稳妥。

#### Issue List
- 无

### Stage 3.3 - Code Review
- 需求覆盖：通过。默认启动路径、显式路径覆盖、失配 warning、Server `GOWORK` 防护均已落地。
- 架构合理性：通过。改动集中在 root 启动脚本，不扩展到 `Server / Proto / SubProto` 主线。
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过。仅新增本地文件存在性检查和少量文本探测，无额外运行时热点。
- 可读性与一致性：通过。helper 命名与既有 PowerShell 结构一致，启动输出保持现有风格。
- 可扩展性与配置化：通过。保留 `-ServerProjectDir` 覆盖；主线未来合入 `stream` 后会自动恢复优先主线。
- 稳定性与安全：通过。错误路径、缺少 `go.mod/cmd\hub_server`、无 `stream` 支持等情况均转为显式反馈。
- 测试覆盖情况：通过。完成帮助验证、脚本行为验证、`GOWORK` 正反验证。
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过。本轮未使用子Agent。

### Stage 4 - Change Archive
- Change:
  - `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server\docs\change\2026-03-29_root-run-dev-stream-server.md`
- Lessons:
  - `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server\docs\lessons\run-dev-stream-server-selection.md`
- Index updates:
  - `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server\docs\change\README.md`
  - `D:\project\MyFlowHub3\worktrees\fix-run-dev-stream-server\docs\lessons\README.md`
- Requirements impact: `none`
- Specs impact: `none`
- Lessons impact: `add`
- Validation summary:
  - `go env GOWORK`（server-stream worktree）命中根 `go.work`
  - `$env:GOWORK='off'; go test ./tests -run TestStreamRootHubConnectDisconnect -count=1` 通过
  - `go test ./tests -run TestStreamRootHubConnectDisconnect -count=1` 在 workspace 模式下失败
  - `Get-Help .\scripts\run-dev.ps1 -Detailed` 通过
  - `pwsh -File scripts/run-dev.ps1 -ServerAddr ':9011' -SkipWin -SkipMetricsNode` 输出确认自动切到 `stream` worktree
  - `pwsh -File scripts/run-dev.ps1 -ServerProjectDir 'repo\MyFlowHub-Server' -ServerAddr ':9012' -SkipWin -SkipMetricsNode` 输出明确 warning

阻塞：否
进入 3.2
