# Plan - Win Frontend Empty NodeModules Guard

## Workflow Information
- Repo: `MyFlowHub-Win`
- Branch: `fix/win-vite-build-cli`
- Base: `main`
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-vite-build-cli`
- Current Stage: `4`

## Stage Records

### Initialization
- `guide.md`:
  - workspace `D:\project\MyFlowHub3` guide and repo boundary rules were read before implementation
  - `$m-autoflow` initialization, stage, sub-agent, and template references were read
  - `$m-docs` routing rules were read before updating `plan.md`
- base/worktree confirmation:
  - control repo: `D:\project\MyFlowHub3\repo\MyFlowHub-Win`
  - control branch: `main`
  - control repo has unrelated user-owned dirtiness and stays control-plane only:
    - `go.mod`
    - `myflowhub-mcp.exe`
  - active execution worktree: `D:\project\MyFlowHub3\worktrees\fix-win-vite-build-cli`
  - active execution branch: `fix/win-vite-build-cli`
  - this workflow only modifies `MyFlowHub-Win`

### Stage 1 - Requirements Analysis
#### Goal
- 修复 Win 前端构建链路里“`frontend/node_modules` 目录存在但为空时，Wails 仍打印 `Installing frontend dependencies: Done.`，随后在 `npm run build` 阶段因缺少 `vite/bin/vite.js` 崩溃”的问题。

#### Scope
- Must:
  - 让 `wails build` / `wails dev` 经过仓内脚本时，能够识别空或残缺的前端构建依赖并自愈
  - 保持现有 Vite 构建行为和 `frontend/dist/placeholder.txt` 回写逻辑
  - 给出可重复验证路径，覆盖“空 `node_modules` 目录复现失败”这一关键场景
- Optional:
  - 在不扩大改动面的前提下，让 `preview` / `test` 也复用同一前端工具链守卫
- Not in scope:
  - 不修改 Wails 上游实现
  - 不做依赖版本升级或大规模前端工具链重构
  - 不把当前内部构建问题升级成新的产品 requirement / spec

#### Use Cases
- 用户或脚本清空了 `frontend/node_modules` 内容，但保留了目录本身
- fresh worktree 或主仓在一次失败/中断安装后留下空目录
- `wails build` / `wails dev` 进入前端编译阶段时，需要确保 `vite` 和关键构建依赖真实可用

#### Functional Requirements
- `npm run build`、`npm run dev`、`npm run preview` 进入 Vite 前，必须先检查关键构建依赖是否存在
- 若关键依赖缺失，仓内脚本必须主动执行本地 `npm install`
- 若自愈后依赖仍缺失，必须显式报错并指出缺少的文件，而不是继续进入 Vite
- `build` 路径仍需在成功后补回 `frontend/dist/placeholder.txt`
- 在依赖完整时，不应做额外 reinstall

#### Non-functional Requirements
- 最小安全改动，不引入与当前问题无关的行为变化
- Windows / PowerShell / Wails 路径下可用，不依赖 bash-only 语义
- 避免每次构建都全量 reinstall，只有在缺失时才触发
- 错误信息必须足够明确，便于下次快速定位

#### Inputs / Outputs
- Inputs:
  - `frontend/package.json`
  - `wails.json`
  - `README.md`
  - `docs/change/2026-03-21_win-frontend-build-chain.md`
  - `docs/lessons/frontend-build-babel-parser-missing.md`
  - `docs/lessons/wails-embed-dist-placeholder.md`
- Outputs:
  - 仓内前端工具链守卫脚本
  - 更新后的 `frontend/package.json`
  - 本 worktree 的 `plan.md`
  - Stage 4 归档到 `docs/change/`
  - 如确认有复用价值，新增或更新 `docs/lessons`

#### Edge Cases
- `frontend/node_modules` 目录存在，但内容为空
- `node_modules` 非空，但关键文件如 `vite/bin/vite.js`、`@vitejs/plugin-vue/package.json`、`@babel/parser/package.json` 缺失
- `npm` 不可用或 reinstall 失败
- `build` 成功但 `dist/placeholder.txt` 未补回，导致后续 `go:embed` 风险回归

#### Acceptance Criteria
- 在 worktree 内手工构造“空 `frontend/node_modules` 目录 + 保留其目录本身”后，`wails build -debug -skipembedcreate -nopackage` 不再复现 `Cannot find module '...vite/bin/vite.js'`
- 正常依赖已存在时，`wails build -debug -skipembedcreate -nopackage` 仍然通过
- `build` 结束后 `frontend/dist/placeholder.txt` 仍存在
- 变更后的错误处理能在依赖缺失且无法自愈时给出明确失败信息

#### Risks
- 从 `npm run build` 内再触发 `npm install` 属于递归式工具调用，需要避免脚本无限循环和含糊失败
- 关键依赖检查过窄会漏掉其他同类问题，过宽会无谓触发 reinstall
- `frontend/package.json` 是 Wails / 手工 npm / README bootstrap 的共用入口，改动后要做完整回归

#### Issue List
- none

### Stage 2 - Architecture Design
#### Overall Solution
- 在仓内新增一个 Node 守卫脚本，接管 `frontend/package.json` 的 `dev` / `build` / `preview` 入口。
- 守卫脚本在真正调用 Vite 前先检查关键构建依赖文件是否存在；若缺失，则在 `frontend/` 目录下执行一次 `npm install`，然后重新校验。
- `build` 成功后由同一脚本负责补回 `dist/placeholder.txt`，避免继续保留内联 `node -e` 逻辑。
- 保持 `wails.json` 的 `frontend:build = npm run build` 不变，因为根因发生在编译入口前的依赖完整性，而不是 `wails.json` 命令字符串本身。

#### Alternatives Considered
- 只做本地清理指导，不改仓：
  - 不采纳；可以临时恢复，但无法防止同类问题再次出现
- 把 `wails.json` 的 `frontend:install` 改成 `npm ci`：
  - 不采纳；当前复现表明只要保留空 `node_modules` 目录，Wails 仍可能在编译前落到错误状态，且 `npm ci` 会扩大每次构建的 I/O
- 把 `vite` 等工具迁到 `dependencies`：
  - 不采纳；不能解决“空目录存在”的判断问题，还会扩大运行时依赖面
- 直接修改 Wails 上游或等待上游修复：
  - 不采纳；超出当前仓边界

#### Module Responsibilities
- `frontend/scripts/run-vite.mjs`:
  - 检查关键依赖
  - 触发缺失时的 `npm install`
  - 调起 Vite
  - 在 `build` 成功后补回 placeholder
- `frontend/package.json`:
  - 把 `dev` / `build` / `preview` 指向仓内守卫脚本
- docs:
  - 在 Stage 4 记录复现条件、根因和回滚方式
  - 如果确认高复用，沉淀到 `docs/lessons`

#### Data / Call Flow
1. `wails build` 或用户执行 `npm run build`
2. npm 进入仓内守卫脚本，而不是直接跑 `node_modules/vite/bin/vite.js`
3. 守卫脚本检查关键构建依赖文件
4. 若缺失，则执行 `npm install`
5. 校验通过后调用本地 Vite CLI
6. `build` 成功后写回 `frontend/dist/placeholder.txt`

#### Interface Drafts
- `frontend/package.json`
  - `dev`: `node ./scripts/run-vite.mjs dev`
  - `build`: `node ./scripts/run-vite.mjs build`
  - `preview`: `node ./scripts/run-vite.mjs preview`
- `frontend/scripts/run-vite.mjs`
  - 参数：`dev | build | preview`
  - 内部关键检查项：
    - `node_modules/vite/bin/vite.js`
    - `node_modules/@vitejs/plugin-vue/package.json`
    - `node_modules/@babel/parser/package.json`

#### Error Handling and Safety
- 若 `npm install` 执行失败，直接透传 exit code 并停止
- 若 reinstall 后关键文件仍缺失，显式列出缺失文件并退出
- 不吞掉 Vite 原始错误
- 只在缺失时触发 install，避免正常路径上的多余 I/O

#### Performance and Testing Strategy
- 关键依赖存在时只做少量文件存在性检查
- 验证路径：
  - 正常路径：`wails build -debug -skipembedcreate -nopackage`
  - 复现路径：手工构造空 `frontend/node_modules` 目录后再跑同一命令
  - 必要时补充直接 `npm run build` 验证 placeholder 行为
- 若实现拆出纯函数，优先加轻量 Node 侧测试；若不拆，则至少保留可重复的命令级回归验证

#### Extensibility Design Points
- 守卫脚本后续可按同一模式接管 `vitest` 或其他构建工具，不必再在 `package.json` 中写多段内联命令
- 关键依赖清单集中在一个脚本里，后续升级 Vite / Vue 编译链时可统一维护

#### Issue List
- none

### Stage 3.1 - Planning
#### Project Goal and Current State
- Current state:
  - `fix/win-vite-build-cli` worktree 中已完成问题复现和对照验证
  - 直接删除 `frontend/node_modules` 后，`wails build` 会重新安装依赖并通过
  - 保留空 `frontend/node_modules` 目录时，可稳定复现用户日志中的 `Cannot find module '...vite/bin/vite.js'`
  - 主仓 `D:\project\MyFlowHub3\repo\MyFlowHub-Win\frontend\node_modules` 当前计数为 `0`，与该复现条件一致
- Project goal:
  - 让仓内构建入口在遇到空或残缺 `node_modules` 时自动修复，而不是继续误报“Installing frontend dependencies: Done.”

#### Docs Governance Routing Decision
- 使用 `$m-docs` 校验计划文档路由、requirements/specs 影响和 lessons 查询入口
- docs tree 已存在且结构清晰，无需 bootstrap
- canonical destinations:
  - 稳定产品 truth: 无新增 requirement / spec
  - workflow control: worktree-root `plan.md`
  - workflow result: `docs/change/YYYY-MM-DD_topic.md`
  - reusable troubleshooting: `docs/lessons/<topic>.md`（若 Stage 4 判定需要）
- Requirements impact: `none`
- Specs impact: `none`
- Related requirements paths:
  - none
- Related specs paths:
  - none
- Related lessons paths:
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\lessons\frontend-build-babel-parser-missing.md`
  - `D:\project\MyFlowHub3\repo\MyFlowHub-Win\docs\lessons\wails-embed-dist-placeholder.md`

#### Executable Task List
- [x] `WIN-BUILD-GUARD-1` 新增仓内 Vite 守卫脚本并接管 `frontend/package.json` 的构建入口
- [x] `WIN-BUILD-GUARD-2` 用正常路径和“空 `node_modules` 目录”路径回归验证修复
- [x] `WIN-BUILD-GUARD-3` 完成 Stage 4 归档，并在确认高复用时新增/更新 lessons

#### Task Details
##### `WIN-BUILD-GUARD-1` - Add Repo-owned Vite Guard
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-vite-build-cli`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-vite-build-cli\plan.md`
- Goal:
  - 把 `dev` / `build` / `preview` 从直连 `node_modules/vite/bin/vite.js` 改成可自愈的仓内守卫入口
- Files / Modules:
  - `frontend/package.json`
  - `frontend/scripts/run-vite.mjs`
- Write Set:
  - 仅限上述文件；如实现需要新增轻量测试文件，必须回到本计划范围内
- Acceptance:
  - 空 `node_modules` 目录不再导致 `vite.js` 缺失崩溃
  - 依赖完整时不触发 reinstall
  - `build` 仍能补回 placeholder
- Test Points:
  - `wails build -debug -skipembedcreate -nopackage`
  - `npm run build`
  - 手工构造空 `frontend/node_modules` 目录后重跑 `wails build`
- Rollback:
  - 回退 `frontend/package.json`
  - 删除并回退 `frontend/scripts/run-vite.mjs`

##### `WIN-BUILD-GUARD-2` - Validate Normal and Corrupted Install Paths
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-vite-build-cli`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-vite-build-cli\plan.md`
- Goal:
  - 证明修复覆盖了用户实际报错路径，同时不破坏正常构建
- Files / Modules:
  - implementation write set only
- Write Set:
  - 不新增业务文件；只允许为了验证临时构造或清理 `frontend/node_modules`
- Acceptance:
  - 正常构建路径通过
  - 空目录复现路径通过
  - 若失败，日志能明确指出哪一步未满足
- Test Points:
  - `wails build -debug -skipembedcreate -nopackage`
  - `npm run build`
- Rollback:
  - 删除验证期间生成的临时依赖目录，保留代码改动待 review

##### `WIN-BUILD-GUARD-3` - Archive the Failure Pattern
- Owner: main agent
- Worktree: `D:\project\MyFlowHub3\worktrees\fix-win-vite-build-cli`
- Plan Path: `D:\project\MyFlowHub3\worktrees\fix-win-vite-build-cli\plan.md`
- Goal:
  - 把这次“空 `node_modules` 目录导致 Wails 假通过安装阶段”的排查经验沉淀为 change/lesson
- Files / Modules:
  - `docs/change/YYYY-MM-DD_topic.md`
  - `docs/lessons/*.md`（如需要）
  - `docs/lessons/README.md`（如需要）
- Write Set:
  - 仅限 Stage 4 文档
- Acceptance:
  - 归档中包含症状、触发条件、快速检查和回滚
  - 若 lessons 新增，索引同步更新
- Test Points:
  - 文档与最终实现、验证结果一致
- Rollback:
  - 删除本轮新增的 change/lesson 文档

#### Dependencies
- `npm` 可执行
- `wails v2.11.0`
- Go / Node 当前本机环境保持可用

#### Risks and Notes
- 该问题与 `frontend/wailsjs/**` 缺失、`@babel/parser` 缺失、`dist/placeholder.txt` 丢失属于不同层级，需要在归档里明确区分
- 当前计划的推荐方案是“仓内守卫脚本”，而不是“仅修主仓本地环境”
- 如果实现阶段发现 `npm install` 从 npm script 内调用存在不可接受副作用，需要回到 3.1 重新确认方案

#### Parallelism Assessment
- 当前不派发子 Agent
- 原因：
  - 写集高度耦合，核心改动集中在 `frontend/package.json` 和单个守卫脚本
  - 验证路径依赖同一套构建环境，拆分后会引入重复复现和冲突
  - 进入 3.2 后重新评估，结论仍是单 Agent 串行实现更安全

#### Issue List
- none

### Stage 3.2 - Implementation
#### File-level Change Summary
- `frontend/package.json`
  - `dev` / `build` / `preview` 改为走仓内 `run-vite.mjs` 守卫入口
- `frontend/scripts/run-vite.mjs`
  - 新增前端构建守卫脚本
  - 在真正调用 Vite 前检查关键依赖
  - 缺失时执行 `npm install`
  - `build` 成功后补回 `dist/placeholder.txt`
- `frontend/scripts/run-vite.test.mjs`
  - 新增轻量脚本级测试
  - 覆盖缺失检测、自愈安装、失败显式报错和 placeholder 回写

#### Design Notes
- 保持 `wails.json` 不变，把自愈逻辑收口到仓内 npm script 入口
- 关键依赖检查项限定在当前已知会直接挡住构建的构建链文件
- 不把 `npm ci` 变成每次构建的默认路径，只在缺失时才做 install

#### Validation
- `npm install`
  - 通过
- `npm run test -- scripts/run-vite.test.mjs`
  - 通过，`5` 个测试全部通过
- `npm run build`
  - 通过
- 手工构造空 `frontend/node_modules` 目录后执行 `$env:GOWORK='off'; wails build -debug -skipembedcreate -nopackage`
  - 通过
  - 原始 `Cannot find module '...vite/bin/vite.js'` 未再复现

### Stage 3.3 - Code Review
- 需求覆盖：通过
  - 已覆盖空或残缺 `node_modules` 的自愈、显式失败、placeholder 回写和关键复现路径验证
- 架构合理性：通过
  - 修复落在仓内前端脚本入口，没有扩展到 Wails 上游或无关模块
- 性能风险（N+1 / 重复计算 / 多余 I/O / 锁竞争）：通过
  - 正常路径只做少量文件存在性检查，只有缺失时才执行 `npm install`
- 可读性与一致性：通过
  - `package.json` 脚本入口统一，构建链守卫集中在单文件维护
- 可扩展性与配置化：通过
  - 关键检查项集中在脚本常量，后续可扩展到其它前端工具入口
- 稳定性与安全：通过
  - `npm install` 失败和自愈后仍缺文件都会显式报错，不吞原始 Vite 错误
- 测试覆盖情况：通过
  - 新增脚本级 Vitest，并完成正常路径与空目录路径命令级验证
- 子Agent治理与审计（任务映射、上下文完整性、文件所有权、结果复核、冲突处理、记录完整性）：通过
  - 本轮未使用子 Agent

#### Review Notes
- `go.mod` 与 `frontend/dist/placeholder.txt` 当前在工作区显示为修改，但 `git diff --ignore-cr-at-eol --exit-code` 对两者均返回通过，判定为本机 CRLF 规范化噪音，不计入本轮语义改动

### Stage 4 - Archive Record
- repo archive
  - `docs/change/2026-04-04_win-frontend-empty-node-modules-guard.md`
- lesson updates
  - `docs/lessons/frontend-build-empty-node-modules.md`
  - `docs/lessons/README.md`
- index updates
  - `docs/change/README.md`
  - `docs/plan/README.md`
  - `docs/lessons/README.md`

阻塞：否
Stage 4 完成；已合并到 `repo/MyFlowHub-Win/main`，进入工作区归档与 cleanup
