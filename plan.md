# Plan - MyFlowHub 重构可复现性与文档治理收口

## Workflow Information

- Repo: `D:/project/MyFlowHub3/repo/MyFlowHub`
- Branch: `refactor/reproducible-closeout`
- Base: `master@078f4b75ee8316b0baaed67c4be4d6f7fbf301d9`
- Project Root: `D:/project/MyFlowHub3`
- Docs Root: `D:/project/MyFlowHub3/repo/MyFlowHub/docs`（与 canonical repo 同库，本地无 remote）
- Code Repos: 仅 `D:/project/MyFlowHub3/repo/MyFlowHub`
- Active Worktree: `D:/project/MyFlowHub3/worktrees/reproducible-closeout`
- Current Stage: `3.2 implementation complete; RC04 enters 3.3`
- Approval: 用户于 2026-08-28 明确要求自行设置 Goal 并把整个重构流程跑完；该授权适用于下列 `Will Execute` Task IDs，不包含计划外、远端或破坏性动作。

## Stage Records

### Initialization

- 已读取 repository `guide.md`。
- canonical repo、docs root、base 和唯一参与仓库均已确认。
- 主 checkout 仅作为 control plane；实现只在 sibling `worktrees/reproducible-closeout` 进行。
- 主 checkout 的论文、附件、私有本机资料与含凭据的本地 guide 变化均排除在写集外。

### Discuss - Discovery And Requirements Shaping

#### Original Request / Source

- [重构可复现性与文档治理收口 intake](docs/intake/2026-08-28_reproducible-refactor-closeout.md)
- 用户要求继续整体重构并无人值守跑完完整 workflow。

#### Goal

- 把“脏工作树已验证”提升为“canonical HEAD 在全新 checkout 可复现”。
- 将旧仓提取出的 MyFlowHub 受治理文档纳入正确分类和索引。
- 保留不属于本轮重构的用户资产和外部动作边界。

#### Scope

- 必须：构建输入、生成文件 EOL 约束、架构测试修正、并发测试夹具修正、文档迁移与索引、完整本地验证、归档、合并、worktree 清理。
- 可选：无；发现额外正确性缺陷必须先回到本计划扩写 Task ID。
- 不做：论文/附件处理、新 Transport、legacy bridge、push/release/sign/publish、外部硬件或商店认证。

#### Assumptions

- 主 checkout 未提交的 `docs/` 内容是旧仓退役前已提取的历史材料；只有进入明确分类、被索引且通过链接门禁的 Markdown 才可导入。
- 生产运行时语义已经完成，不在本轮重新设计。

#### Options Considered

- 最小修复：改动小，但继续遗留文档治理缺口。
- 受控导入：同时解决可复现性和文档历史，写集可审查。
- 整体快照：会混入论文、二进制和本机敏感信息。

#### Rejected Options

- 最小修复不足以完成此前明确要求的 docs 迁移与治理。
- 整体快照违反最小写集、隐私和可回滚原则。

#### Recommended Direction

- 受控导入；先冻结输入清单，再分别处理代码/构建与 docs，最后在第二个全新验证 checkout 中证明 HEAD 自包含。

#### Research Summary

- 未使用外部研究；本任务只依赖仓库稳定文档、Git 状态和本地验证事实。

#### Issue List

- Blocking issues: none。

## Plan - Requirements And Architecture

### Requirements Analysis

#### Use Cases

1. 开发者 clone/checkout 当前 HEAD 后，无需主 checkout 的未提交文件即可运行架构测试与统一构建入口。
2. 文档读者从 `docs/README.md` 和分类索引可追溯已退役旧仓的计划、变更和可复用经验。
3. 后续 workflow 能明确区分 canonical repo、sibling worktree 和工作区本机资产。

#### Functional Requirements

- `build/toolchain.json` 必须成为已跟踪构建输入，且 `.gitignore` 只放行该文件。
- `sdk/bindings/generated/contracts.json` 必须固定 LF，避免 Windows/CI 字节级漂移。
- repository boundary test 必须忽略当前 checkout 自身 `.git`，仍拒绝嵌套 `.git`。
- blocking pipe 测试夹具的 write/close 幂等必须使用独立同步状态。
- Desktop package fingerprint 必须与 `package.json` 当前内容一致。
- 导入的历史文档必须按 intake/plan/change/lessons 分类，最近索引必须覆盖新增 leaf，并通过相对链接检查。
- 新增或更新的 root/稳定 docs 必须描述当前 `repo/MyFlowHub` + sibling `worktrees/` 布局。

#### Non-functional Requirements

- 不改变 Node/Resource/Subscription/Command/Transport 的生产协议与运行时语义。
- 新的 canonical 配置不得硬编码本机绝对工具路径；历史归档可保留非秘密路径证据。Wi-Fi 凭据、论文二进制和无关用户变更不得进入 commit。
- 所有验证命令从 worktree 根以 `GOWORK=off` 语义运行；失败必须显式修复或记录外部阻塞。
- commit 使用中文说明；不配置 remote、不 push、不发布。

#### Inputs / Outputs

- 输入：`master@078f4b7`、主 checkout 中明确选择的代码/构建 diff、82 个受治理 Markdown leaf、12 个 docs tracked diff。
- 输出：自包含的 feature branch commits、可追溯 plan/change/lesson、全新 checkout 验证证据、本地合并后的 `master`。

#### Edge Cases

- 主 checkout 自身 `.git` 是 gitfile 而非目录时，boundary guard 仍必须正确；嵌套真实 `.git` 继续失败。
- Windows `core.autocrlf` 不得让 generated contract freshness 误报。
- 文档索引可能引用已存在、待导入或缺失文件；缺失链接必须阻塞归档。
- 主 checkout 在 workflow 期间继续保持脏状态；合并不得覆盖或暂存这些用户变化。

#### Acceptance Criteria

- 计划内文件以外没有进入分支；论文目录零变更。
- focused tests、`go test ./... -count=1`、`go vet ./...`、前端测试/typecheck/build、默认三进程 dev smoke 均通过。
- 新 branch commit 在第二个全新 checkout 中通过核心/架构/迁移/集成与生成一致性门禁。
- 所有新 docs leaf 可从分类索引到达，相对 Markdown 链接无缺失。
- archive 完成后本地 fast-forward/merge 到 `master`，workflow worktree 被移除；主 checkout 原有无关状态不丢失。

#### Risks

- 历史 docs 数量较多，错误复制会制造断链或重复真相；通过固定来源清单、分类索引和链接检查控制。
- 在脏 `master` 上合并可能受同路径未提交变更阻塞；若发生，停止合并并保留已验证 branch，不覆盖用户内容。
- 真实 Android/RFCOMM/ESP32 硬件与签名环境不在本地自包含验收范围。

### Architecture Design

#### Overall Solution

采用三层收口：

1. Repository reproducibility：跟踪工具链输入、EOL 规则、准确 fingerprint 和防回归测试。
2. Docs governance：以当前 canonical docs root 为唯一入口，历史只进入 plan/change/lessons，稳定事实只在 features/requirements/specs/decisions。
3. Clean-checkout gate：在当前 worktree 轻量验证后，再从 branch commit 创建临时验证 worktree，确保结果不依赖主 checkout dirt。

#### Module Responsibilities

- `.gitignore` / `.gitattributes` / `build/`：构建与生成输入边界。
- `internal/archtest`：仓库结构、文档和 canonical 约束守卫。
- `runtime/link/*_test.go`：并发关闭测试夹具的确定性。
- `docs/`：稳定事实、历史归档与排障入口。
- `migration/`：本轮选入/排除来源记录。

#### Data / Call Flow

`master dirty evidence → bounded source inventory → active worktree edits/import → focused gates → branch commit → clean validation worktree → full gates → archive commit → control-plane merge/cleanup`。

#### Interface Drafts

- 不新增生产 API。
- `build/toolchain.json` 保持现有 JSON keys；`scripts/mfh.ps1` 与 archtest 继续读取相同路径。
- 文档链接使用 repository-relative Markdown 路径。

#### Error Handling And Safety

- 缺少选定源文件、hash 变化、链接缺失或测试失败均显式失败，不静默跳过。
- 不使用 destructive reset/checkout 清理主 checkout。
- 合并若与用户未提交变更冲突，保留 branch/worktree 并报告阻塞。

#### Performance And Testing Strategy

- 代码变化仅涉及测试和元数据，运行时性能风险为零。
- 文档验证使用一次文件枚举和一次链接遍历，避免重复扫描。
- 分阶段运行 focused → full Go/vet → frontend → real dev smoke → clean checkout gates。

#### Extensibility Design Points

- `.gitignore` 精确放行单个 canonical toolchain manifest，未来新增构建输入必须显式评审。
- `.gitattributes` 只约束字节级生成契约，避免全仓 EOL 改写。
- docs taxonomy 和分类索引继续支持新的产品/历史而不恢复多仓 source-of-truth。

## Stage 3.1 - Planning

### Related Stable Docs

- Intake: [本轮收口](docs/intake/2026-08-28_reproducible-refactor-closeout.md)、[vNext 全量迁移](docs/intake/2026-08-27_vnext-full-migration.md)
- Features: `docs/features/*.md`（行为不变）
- Requirements: [统一节点运行时](docs/requirements/unified-node-runtime.md)
- Specs: [Build and CI](docs/specs/build-and-ci.md)、[Repository boundaries](docs/specs/repository-and-module-boundaries.md)
- Decisions: [单一 canonical monorepo](docs/decisions/2026-08-27_single-canonical-monorepo.md)、[门禁式全量切换](docs/decisions/2026-08-27_gated-full-migration-cutover.md)
- Lessons: [Wails bindings drift](docs/lessons/wails-binding-proto-drift.md)、[Wails cross-project bindings](docs/lessons/wails-bindings-cross-project.md)

### Stable Docs Impact

- Intake impact: add
- Feature impact: clarify index only; feature behavior unchanged
- Requirements impact: clarify index only
- Specs impact: clarify `build-and-ci.md` and `repository-and-module-boundaries.md`
- Decision impact: none
- Lessons impact: import historical reusable lessons and add clean-checkout reproducibility lesson if validation confirms the pattern

### Execution Scope After Approval

#### Will Execute

- RC01 — 冻结并审计选入/排除来源。
- RC02 — 修复 canonical HEAD 的构建与测试可复现性。
- RC03 — 导入并治理旧仓提取文档和 checkout 路由文档。
- RC04 — 运行工作树与全新 checkout 的完整本地门禁。

#### Will Not Execute Now

- NX01 — 论文、附件和主 checkout 其他无关变化；属于用户资产，保持原样。
- NX02 — 新 serial/USB/WebSocket Transport；属于新功能，需要独立 workflow。
- NX03 — legacy compatibility bridge；与已接受 clean break 决策冲突。
- NX04 — push/release/sign/publish/remote archive；需要单独外部授权。
- NX05 — 真实硬件、签名平台、商店认证；依赖外部设备/凭据，不能伪报。

### Task Details

#### RC01 - Freeze And Audit Inputs

- Owner: main agent
- Worktree: `D:/project/MyFlowHub3/worktrees/reproducible-closeout`
- Plan Path: `D:/project/MyFlowHub3/worktrees/reproducible-closeout/plan.md`
- Goal: 记录主 checkout 选入和排除集合，避免从脏状态静默复制。
- Files / Modules: `migration/reproducible-closeout-inputs.md`
- Write Set: 仅该审计文档。
- Acceptance: 记录 base、源 checkout、选入类别、排除类别、敏感信息边界和验证方式。
- Test Points: 路径存在、Git status 与计划一致、无论文/凭据路径进入 selected set。
- Rollback: 删除该审计文档。

#### RC02 - Make The Repository Self-contained

- Owner: main agent
- Goal: 把已验证但未提交的必要构建输入和测试修正变成可复现 HEAD。
- Files / Modules: `.gitignore`, `.gitattributes`, `build/toolchain.json`, `apps/desktop/frontend/package.json.md5`, `sdk/bindings/generated/contracts.json`, `internal/archtest/architecture_test.go`, `runtime/link/session_test.go`
- Write Set: 上述精确文件。
- Acceptance: toolchain manifest 被跟踪；generated contract 固定 LF；MD5 匹配；root `.git` 不误报且 nested git 仍拒绝；并发 close 测试确定。
- Test Points: focused Go tests、MD5 校验、`git ls-files --eol`、generate freshness、`git diff --check`。
- Rollback: 回退 RC02 commit；不影响生产 runtime API。

#### RC03 - Govern And Import Refactor Documentation

- Owner: main agent
- Goal: 将旧仓提取的 MyFlowHub Markdown 历史和当前 checkout 布局纳入 canonical docs root。
- Files / Modules: `README.md`, `repos.md`, `docs/{README,intake,features,requirements,specs,plan,change,lessons}/**`
- Write Set: 仅 Markdown 与分类索引；明确排除 `论文/**`、附件、二进制和含本机凭据的 `guide.md` dirty version。
- Acceptance: 82 个候选 leaf 逐项存在于明确分类；新增 leaf 被索引；stable docs 不被历史 archive 取代；根入口说明本地 publication boundary。
- Test Points: Markdown relative-link checker、category index coverage、archtest docs guards、敏感字符串扫描。
- Rollback: 回退 RC03 commit；现有 canonical stable docs 保留在前一 commit。

#### RC04 - Validate Current And Fresh Checkouts

- Owner: main agent
- Goal: 证明 commit 不依赖主 checkout dirt。
- Files / Modules: no product write set；只允许测试发现后回到 owning RC02/RC03。
- Acceptance: focused、full Go/vet、frontend tests/typecheck/build、real run-dev smoke、fresh validation worktree gates 全部通过。
- Test Points: `go test ./... -count=1`, `go vet ./...`, Desktop/Metrics Vitest + TypeScript + Vite, `scripts/run-dev.ps1`, clean worktree generated/arch/migration/integration checks。
- Rollback: 无行为文件；失败回到 owning Task ID 修复。

### Dependencies

- RC01 → RC02/RC03 → RC04。
- RC02 与 RC03 逻辑可分，但都依赖同一脏 source snapshot；主 agent 串行导入后统一验证。

### Parallelism Assessment

- 不派发子Agent：宿主策略要求只有用户显式要求 sub-agent/delegation 时才能创建；本次用户授权的是完整 workflow，而非 Agent 委派。
- 此外 RC02/RC03 都需要统一审计同一主 checkout 状态，串行执行更能避免 source snapshot 漂移。

### Risks And Notes

- 计划只确认上述 RC01-RC04；任何生产行为修改都必须回到 3.1 更新计划。
- 本轮 archive/merge/cleanup 由 `$m-archive` 阶段完成，不扩大 RC01-RC04 实现写集。

### Gate

- Blocked: no
- RC01-RC03 implemented and lightweight checks passed；RC04 enters `$m-test`。
- Do not dispatch implementation sub-agents。
