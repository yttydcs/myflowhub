# Plan - Win Project Meta UUID

## Workflow Information
- Repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
- Branch: `fix/win-project-meta-uuid`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-project-meta-uuid`
- Current Stage: `4 archived / awaiting workflow end confirmation`

## Stage Records

### Initialization
- guide.md:
  - 已读取 workspace 根 `D:\project\MyFlowHub3\guide.md`
  - 已读取 `$m-autoflow` 的 `references/initialization.md`、`references/stages.md`、`references/m-docs-integration.md`、`references/templates.md`
  - 已读取 `$m-docs` 的 `SKILL.md` 与 `references/requirement-impact.md`、`references/indexing-rules.md`、`references/lessons-rules.md`
- base/worktree confirmation:
  - implementation repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - dedicated branch: `fix/win-project-meta-uuid`
  - dedicated worktree: `D:\project\MyFlowHub3\worktrees\fix-win-project-meta-uuid`
  - implementation will stay inside the worktree only
  - active modules:
    - `frontend/src/stores/flowProjects.ts`
    - `frontend/src/stores/flowProjects.test.ts` (new)

### Stage 1 - Requirements Analysis
#### Goal
- 修复 Win 端本地 Flow 项目元数据中的默认 `flowId` 生成逻辑，使自动创建的新项目默认使用 UUID，并让本地校验与部署契约保持一致，避免“本地可创建、部署时才因非 UUID 被拒绝”的延迟失败。

#### Scope
- 必须:
  - `createProject(...)` 在未显式传入 `flowId` 时自动生成 UUID 格式的 `flowId`
  - 本地 `flowId` 校验同时覆盖“非空 + 本地唯一 + UUID 格式”
  - `updateProjectMeta(...)` 与 `saveProjectPayload(...)` 复用同一校验，避免手工编辑或 payload 回写继续写入非 UUID
  - `deployProject(...)` 在发送远端 `flow.set` 前对当前项目 `flowId` 做本地契约校验，并返回明确错误
  - 新增/更新前端 store 级测试，覆盖自动生成与非法 ID 拒绝路径
- 可选:
  - 将 UUID 校验错误文案收敛为统一提示
- 不做:
  - 不修改 `MyFlowHub-Server` / `MyFlowHub-SubProto` 的协议实现
  - 不自动迁移已有本地项目中的历史非 UUID `flowId`
  - 不改动项目列表、部署弹窗或编辑器的整体交互结构

#### Use Cases
- 用户在项目中心点 `Create` 创建新项目，不需要手填 `flowId`，且后续部署不会因默认 ID 格式错误而失败
- 用户在 `Meta` 中手动编辑 `flowId` 时，如果输入非 UUID，会在本地立即收到明确报错，而不是等到部署远端时失败
- 旧项目若仍保存了历史非 UUID `flowId`，用户在部署时能看到明确的本地错误，并可通过元数据编辑修正

#### Functional Requirements
1. 自动生成的 `flowId` 必须符合 Server `flow` 协议要求的 UUID 格式
2. `flowId` 仍必须保持本地项目内唯一
3. 手工传入或回写的 `flowId` 若不是 UUID，必须显式报错，不能静默接受
4. 本地持久化仍需兼容读取历史项目记录，不能因旧数据中存在非 UUID `flowId` 而在加载时直接丢弃项目
5. 部署前必须在本地暴露“`flowId` 不是 UUID”这一错误，避免无意义的远端请求

#### Non-functional Requirements
- 改动面最小，限制在 Win 前端 store 与对应测试
- 不引入额外网络请求或新的持久化 schema
- 校验逻辑集中复用，避免不同入口出现不一致规则
- 错误提示明确，便于用户自行修复历史项目元数据

#### Inputs / Outputs
- 输入:
  - `createProject({ projectId?, flowId?, name? })`
  - `updateProjectMeta({ projectId, flowId, name? })`
  - `saveProjectPayload(projectId, payload.flow_id)`
  - `deployProject({ projectId, nodeId, trigger, overwrite })`
- 输出:
  - 自动生成的 UUID `flowId`
  - 非 UUID 输入时报错
  - 保持现有本地唯一性与部署编排行为

#### Edge Cases
- 运行环境缺少 `crypto.randomUUID`，仍需生成合法 UUID
- 历史项目保存了旧格式 `fl_xxx`，加载应成功，但部署与更新时需要明确提示
- 用户手工输入大写 UUID，应按 UUID 合法格式接受
- 项目间仍需处理重复 `flowId`

#### Acceptance Criteria
1. 新建项目默认 `flowId` 为 UUID，而不是 `fl_` 前缀随机串
2. `updateProjectMeta(...)` 对非 UUID `flowId` 抛出明确错误
3. `saveProjectPayload(...)` 对非 UUID `flow_id` 抛出明确错误
4. `deployProject(...)` 对历史非 UUID `flowId` 在本地直接失败，且不会继续发送 `flow.set`
5. 相关前端测试通过

#### Risks
- 若直接拒绝历史非 UUID `flowId` 的加载，会破坏已有本地项目；因此本次只能在写入/部署路径拦截
- 若 UUID 校验与服务端实际接受格式不一致，可能产生误拒；需要保持为常见 canonical UUID 格式判断并允许大小写
- 若改动遗漏某个写入入口，仍可能留下“本地可写入、部署时失败”的路径

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 在 `frontend/src/stores/flowProjects.ts` 中引入统一的 `flowId` helper：
  - `isUUIDLike(...)` 判断字符串是否为 canonical UUID 格式
  - `makeFlowID(...)` 改为生成 UUID，而不是 `fl_` 随机串
  - `ensureUniqueFlowID(...)` 升级为“非空 + UUID + 本地唯一”的集中校验
- `createProject(...)`、`updateProjectMeta(...)`、`saveProjectPayload(...)`、`deployProject(...)` 全部复用该 helper
- 新增 `frontend/src/stores/flowProjects.test.ts`，用 mocked Wails bindings 验证持久化与校验路径

#### Alternatives Considered
- 方案 A（采用）：同时修复默认生成和本地写入/部署校验
  - 优点：
    - 与现有 Server 契约完全对齐
    - 能覆盖自动生成、手工编辑、payload 回写和历史项目部署四条路径
    - 改动集中在一个 store helper，测试边界清晰
  - 代价：
    - 历史非 UUID 项目会在部署前收到更早的本地错误
- 方案 B：只把 `makeFlowID(...)` 改成 UUID，不新增本地校验
  - 优点：
    - 改动更小
  - 代价：
    - 手工编辑和 payload 回写仍能写入非法 `flowId`
    - 历史项目部署仍只会在远端失败，问题没有真正收口
- 方案 C：加载时自动迁移历史非 UUID `flowId`
  - 优点：
    - 用户无需手工修复旧项目
  - 代价：
    - `flowId` 是部署身份，自动迁移会改变覆盖/删除目标，风险过高

#### Module Responsibilities
- `frontend/src/stores/flowProjects.ts`
  - 统一 `flowId` 生成、格式校验、唯一性校验和部署前校验
  - 保持现有本地项目 CRUD 与部署编排职责不变
- `frontend/src/stores/flowProjects.test.ts`
  - 验证 UUID 自动生成
  - 验证非法 `flowId` 在元数据编辑、payload 回写、部署前被本地拒绝

#### Data / Call Flow
1. 用户创建项目
2. `createProject(...)` 调用 `makeFlowID(...)` 生成 UUID
3. 若用户编辑元数据或 payload 回写 `flowId`
4. `ensureUniqueFlowID(...)` 先做非空，再做 UUID 校验，再做本地唯一性校验
5. 用户部署项目时
6. `deployProject(...)` 先验证当前 `project.flowId`，再继续 `list/set` 远端调用

#### Interface Drafts
- 新增内部 helper:
  - `const uuidPattern = ...`
  - `const isUUIDLike = (value: string) => boolean`
  - `const makeUUID = () => string`
- 复用点:
  - `ensureUniqueFlowID(projects, flowId, excludeProjectId?)`
  - `deployProject(...)` 增加 `ensureFlowIDDeployable(project.flowId)` 形式的预检查或直接复用 `ensureUniqueFlowID`

#### Error Handling and Safety
- 空 `flowId` 继续报 `Flow ID is required.`
- 非 UUID 改为显式报 `Flow ID must be a UUID.`
- 本地重复继续报 `Flow ID already exists in local projects.`
- 对历史项目只在写入/部署路径拦截，不在加载时破坏数据

#### Performance and Testing Strategy
- UUID 生成与校验为纯本地常数级逻辑，不增加 I/O
- 测试:
  - `frontend/src/stores/flowProjects.test.ts`
  - `npm test -- flowProjects`
- 如时间允许，再运行更小范围的关联测试，避免把基线外问题混入本次修复判断

#### Extensibility Design Points
- 未来若其它模块也需要 UUID 生成，可再抽公共 helper；本次先保持改动最小，不提前共用
- 若后续需要引导用户修复历史非 UUID 项目，可在现有本地校验基础上扩充 UI 提示，而不必重写部署链路

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Goal:
  - 修复 Win 本地 Flow 项目的 `flowId` 默认生成与本地校验，使其与 Server `flow` 契约对齐
- Current State:
  - `frontend/src/stores/flowProjects.ts` 的 `makeFlowID(...)` 当前生成 `fl_` 前缀随机串
  - `ensureUniqueFlowID(...)` 只校验非空和本地唯一，不校验 UUID 格式
  - `deployProject(...)` 会直接把当前 `project.flowId` 发给远端 `flow.set`

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口
- Requirements impact: none
- Specs impact: none
- Stable docs destination:
  - Server `flow_id` 契约继续以 `repo/MyFlowHub-Server/docs/specs/flow.md` 为稳定真相
  - 本次 Win 侧修复结果进入 `docs/change`
  - 暂无证据表明需要新增 lesson；stage 4 再复核
- Related requirements:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\requirements\flow_data_dag.md`
- Related specs:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- Related lessons:
  - none
- Related prior archives:
  - `D:\project\MyFlowHub3\worktrees\fix-win-project-meta-uuid\docs\change\2026-03-21_win-flow-project-center.md`
  - `D:\project\MyFlowHub3\docs\change\2026-03-21_flow-id-guard-and-contract-align.md`

#### Related Requirements / Specs / Lessons
- Requirement:
  - `repo/MyFlowHub-Server/docs/requirements/flow_data_dag.md`
- Spec:
  - `repo/MyFlowHub-Server/docs/specs/flow.md`
- Lessons:
  - none

#### Executable Task List
- [x] `UUID-1` 收敛 `flowProjects` 内的 UUID 生成与校验 helper
- [x] `UUID-2` 在 create/update/payload/deploy 路径接入统一校验
- [x] `TEST-1` 新增 store 级回归测试并运行定向测试
- [x] `REVIEW-1` 完成 3.3 checklist 复核
- [x] `ARCHIVE-1` 写入 `docs/change/2026-03-27_win-project-meta-uuid.md`

#### Task Details
##### `UUID-1` - Flow ID UUID Helper
- Owner:
  - main agent
- Worktree:
  - `D:\project\MyFlowHub3\worktrees\fix-win-project-meta-uuid`
- Plan Path:
  - `D:\project\MyFlowHub3\worktrees\fix-win-project-meta-uuid\plan.md`
- Goal:
  - 将默认 `flowId` 生成从 `fl_` 随机串改为 UUID，并集中格式校验
- Files / Modules:
  - `frontend/src/stores/flowProjects.ts`
- Write Set:
  - `frontend/src/stores/flowProjects.ts`
- Acceptance:
  - 新建项目默认 `flowId` 为 UUID
  - UUID 校验逻辑集中定义
- Test Points:
  - store 级自动生成测试
- Rollback:
  - 回退 `flowProjects.ts` 中的 helper 变更

##### `UUID-2` - Validation Wiring
- Owner:
  - main agent
- Worktree:
  - `D:\project\MyFlowHub3\worktrees\fix-win-project-meta-uuid`
- Plan Path:
  - `D:\project\MyFlowHub3\worktrees\fix-win-project-meta-uuid\plan.md`
- Goal:
  - 让 `createProject / updateProjectMeta / saveProjectPayload / deployProject` 全部使用统一 UUID 校验
- Files / Modules:
  - `frontend/src/stores/flowProjects.ts`
- Write Set:
  - `frontend/src/stores/flowProjects.ts`
- Acceptance:
  - 非 UUID `flowId` 无法被本地写入或部署
  - 历史项目仍可加载
- Test Points:
  - 元数据更新、payload 回写、部署前失败路径
- Rollback:
  - 回退这些入口对 helper 的接线改动

##### `TEST-1` - Store Regression Tests
- Owner:
  - main agent
- Worktree:
  - `D:\project\MyFlowHub3\worktrees\fix-win-project-meta-uuid`
- Plan Path:
  - `D:\project\MyFlowHub3\worktrees\fix-win-project-meta-uuid\plan.md`
- Goal:
  - 用 mocked bindings 验证本次行为，不依赖后端联调
- Files / Modules:
  - `frontend/src/stores/flowProjects.test.ts`
- Write Set:
  - `frontend/src/stores/flowProjects.test.ts`
- Acceptance:
  - 测试覆盖自动生成与关键错误路径
- Test Points:
  - `npm test -- flowProjects`
- Rollback:
  - 删除新增测试文件

#### Dependencies
- `repo/MyFlowHub-Server/docs/specs/flow.md` 提供稳定 `flow_id` UUID 契约
- `docs/change/2026-03-21_win-flow-project-center.md` 说明当前随机 `fl_` 默认值的引入背景

#### Risks and Notes
- 旧本地项目仍可能保留非 UUID `flowId`；本次不会自动迁移
- 若定向测试暴露仓库基线问题，需要明确区分是否由本次改动引入

#### Parallelism Assessment
- 不使用子Agent
- 原因:
  - 代码改动集中在同一个 store 与其测试，写集高度重叠
  - 用户未显式要求 sub-agent delegation
  - 当前主路径不值得拆分并行

#### Issue List
- none

### Stage 3.2 - Implementation
#### Execution Record
- `UUID-1`
  - `flowProjects.ts` 删除旧的 `fl_` 随机 token 生成路径
  - 新增 `uuidPattern`、`isUUIDLike(...)`、`makeUUID(...)`、`ensureFlowIDFormat(...)`
  - `makeFlowID(...)` 改为生成 UUID
- `UUID-2`
  - `createProject(...)` 继续自动生成 `flowId`，但现在生成结果已与 UUID 契约对齐
  - `updateProjectMeta(...)`、`saveProjectPayload(...)` 接入统一 UUID 校验
  - `deployProject(...)` 在发起远端请求前先本地校验 `flowId`
  - overwrite 比较改为基于规范化后的 `flowId`
- `TEST-1`
  - 新增 `frontend/src/stores/flowProjects.test.ts`
  - 覆盖自动生成、非法元数据、非法 payload、历史坏数据部署前阻断

#### Files Changed
- `frontend/src/stores/flowProjects.ts`
- `frontend/src/stores/flowProjects.test.ts`

### Stage 3.3 - Review
#### Review Checklist
- 需求覆盖：通过
  - 自动生成、手工元数据更新、payload 回写、部署前校验都已覆盖
- 架构合理性：通过
  - 改动集中在 `flowProjects` store，未扩散到协议层和页面结构
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 新增逻辑均为本地常数级字符串校验，无新增 I/O
- 可读性与一致性：通过
  - UUID 规则收敛到单一 helper，避免各入口各自判断
- 可扩展性与配置化：通过
  - helper 可继续复用于未来 `flowId` 写入入口；本次未引入新硬编码环境依赖
- 稳定性与安全：通过
  - 非法 `flowId` 现在在本地提前失败，减少远端无效请求
- 测试覆盖情况：通过
  - 新增 4 条 store 级回归测试覆盖本次变更主路径
  - `npm run build` 仍受仓库基线缺失 `wailsjs` 生成物影响，但失败点不在本次改动文件
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 未使用子Agent

### Stage 4 - Change Archive
#### Archive Outputs
- Change archive:
  - `docs/change/2026-03-27_win-project-meta-uuid.md`
- Index updates:
  - `docs/change/README.md`
- Requirements impact:
  - none
- Specs impact:
  - none
- Lessons impact:
  - none
  - 原因：本次问题定位路径短、根因单点明确，归档到 `change` 已足够检索

#### Validation Results
- `frontend`
  - `npm test -- flowProjects`
  - 通过
- `frontend`
  - `npm run build`
  - 失败，原因是仓库基线缺失 `../../wailsjs/go/main/App`，报错文件为 `src/windows/TopicBusWindow.vue`
- worktree
  - `git diff --check`
  - 通过（仅有 Git 行尾转换 warning）

阻塞：否
已完成 4，等待用户确认是否结束 workflow
