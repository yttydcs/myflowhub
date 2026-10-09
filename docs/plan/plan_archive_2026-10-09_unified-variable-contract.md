> 历史执行快照；下列阶段/工作树路径保留当时状态。最终状态见文末归档补充。

# Plan — 平台统一 Variable 与通用消费

## Workflow Information

- Workflow：2026-09-06 unified-variable-contract。
- Current Stage：$m-execute completed；批准范围实现与验收完成，未归档、提交、合入或推送。
- Plan Revision：R2；用户明确“当前不需要任何兼容，产品还没发布”，取消 R1 全部变量兼容安排。
- R2 复核：用户再次调用 $m-plan，本轮补齐本地发布入口与目录提交的验收边界；方案、Task ID、执行范围不变，不将重复规划调用视作实施批准。
- Project Root：D:/project/MyFlowHub3。
- Canonical Repo / Code Repos：D:/project/MyFlowHub3/repo/MyFlowHub，唯一 Go module。
- Active Worktree：D:/project/MyFlowHub3/worktrees/unified-variable-contract。
- Branch：feat/unified-variable-contract。
- Base：master / d77dc54b2fd6efe749547120683dcb913eea4714。
- Docs Root：本工作树 docs/，即 canonical repository docs 的隔离版本；无独立文档仓库或发布目标。
- Plan Path：D:/project/MyFlowHub3/worktrees/unified-variable-contract/plan.md。
- 工作路径：本轮计划、文档和未来实现均在本工作树；GOWORK=off。
- Approval：用户于 2026-09-06 调用 $m-execute，批准 DOC01、VAR01、PROV01、MET01、DESK01、LEGACY02、QA01；EXT02 延期。

## Initialization and state protection

已读取 guide.md、docs 入口、相关需求/规格/决策和 lessons，显式使用 $m-docs 校验文档路由。guide 要求中文 commit、工作区 sibling worktree、canonical 生成契约，浏览器 smoke 不能代替单测与生产构建。

从 d77dc54 创建独立分支/工作树。主检出仍有未提交的 Desktop 节点树刷新改动、生成 dist、产品/Agent Gateway 文档和其他草稿；没有 stash、复制、提交或覆盖。主线 guide.md 的本地附加规则只读取，凭据内容不复制进计划。

执行前重新检查主线/工作树：若主线推进，在工作树集成相应已提交变更并使受影响证据失效；主线未提交刷新功能仍归原改动，不静默包含为本轮实现。最终合入必须保留它，真实联调核对所用二进制基线。刷新功能是否独立提交不阻塞变量实现。

既有 topology workflow 已归档，见 [历史计划](../plan/plan_archive_2026-09-06_depth-scoped-topology.md)。主检出根计划保持原样，本根计划仅管理新工作树。规划阶段仅修改文档；获批后在隔离工作树实施和验证，不重启既有手动联调进程，不提交、不合入、不推送。

## Discussion brief

来源：[本轮 intake](../intake/2026-09-06_unified-variable-contract.md)。$m-discuss 已在当前对话完成，无单独旧 brief 文件，本 intake 持久化已确认约束；无需重新探索或外部研究。

### Confirmed / rejected / deferred

- 已确认：网络只认识节点、网络和资源；节点不等于物理设备，网络没有插件。
- 已确认：Variable 为平台统一资源与格式；Metrics 提供采集结果，Desktop 通用理解、展示和操作，无需预装节点专用 schema。
- 已确认：统一格式允许不同值类型；保留 owner、权限和统一 runtime/SDK，不成为全局任意可写命名空间。
- 已确认：切换前比较当前方案和拟议方案；规划阶段不写实现，用户随后调用 $m-execute 批准 R2。
- 已确认：产品未发布，当前不需要任何兼容；只实现新契约，相关旧 codec/API/renderer/Metrics /set 在本轮清除。
- 不采用：补 Metrics schema 作为终点、按 metrics/* 猜主字段或 /set 绑定、复制 MetricsVariable、远程 executable renderer。
- 延期：其他资源 schema 服务、异步执行编排、Flow/mobile/embedded 重设计；变量旧实现删除不再延期。

### Approved R2

采用 mfh.variable type_version=2；随 read/订阅快照传递标准 definition；统一 value/status/revision/instance；同资源 write 经 owner 执行；全部第一方与调用方一次性替换，仅保留当前格式。版本标识不代表有兼容分支；旧 /set 与被替代的接口/编码同步删除。字段细节和实施范围随 R2 获批。

### R2 compared with R1 / evidence invalidation

| 项目 | R1 提案 | 用户明确无需兼容后的 R2 |
| --- | --- | --- |
| 协议/runtime | 新旧格式并存 | 原位替换，删除旧 codec |
| SDK/提供者 | 保留旧入口并解包新旧值 | API 与全部调用方统一更新，无 deprecated 桥接 |
| Metrics | write 与 /set 并存 | 只留标准 write，删除被替代的 /set |
| Desktop | legacy 变量 renderer/视图兼容 | 只有标准变量路径，无旧格式转换或别名 |
| 集成 | 用兼容扩展保持各批次旧模式绿色 | 同一集成分支分批替换，最后统一完整门禁 |
| 旧实现清理 | 后续删除 | 本轮必做清除残留与禁止双轨门禁（LEGACY02） |

变更来源是用户本轮明确纠正；R1 的兼容范围和相关检查已失效。AC ID 保持，AC-07/12/13 及相关 Task/验收口径修订；R1 规划检查不充当 R2 实施证据，当前状态按下表及执行记录核定。

- [方案比较与决策](../decisions/2026-09-06_platform-variable-contract.md)
- [当前接口规格](../specs/unified-variable-resource.md)

## Requirements Analysis

### Goal / scope / use cases

验收目标：任意未登记到 Desktop 内置 schema 的节点，只通过平台标准 Variable 接口，就能显示布尔、数值/百分比、枚举、对象/数组并按权限编辑。不能仅以 Metrics 截图作为完成证明。

1. 提供者按标准定义实例的类型、范围、单位和状态；消费者通过原资源权限取 definition/value。
2. Desktop 读取、持续订阅、手动刷新和草稿/CAS 编辑；只获 read 权限也能取得版本。
3. Metrics 真类型采样与可控音量/静音/亮度的实际执行；保留 unavailable/stale 和配置业务语义；删除 /set 及旧变量协议类型，更新本机 facade/UI/CLI 调用方。
4. Catalog、管理、文件摘要、Clipboard 和 Metrics config 的所有 Variable 迁移；原领域值和持久化行为保持。
5. 验证旧格式被拒绝、拒绝授权、并发写、资源重建、断连和 Profile 切换；不支持混合版本运行。

### Functional / non-functional requirements

长期意图见 [unified-variable-resource.md](../requirements/unified-variable-resource.md)。平台定义格式和统一校验；provider 声明实例语义与更严格的业务校验；Desktop 不按产品分支。原 owner/name、exact capability、节点树与链路模型保持。

输入和输出在系统边界严格验证；definition、值、缓存、队列和错误都有界。数值/修订精度不丢失；定义只在快照发送，后续事件复用；不遍历整树取 schema；View 不持久化值、定义、draft 或权限。

不新增旧格式解码、转发别名、双版本实现、旧视图/配置迁移器。不因“无需兼容”自动清空真实数据，也不扩大清理到与变量改造无关的历史协议。

输入：标准实例声明、采集/业务更新、read/subscribe/write、授权、期望实例/revision、用户草稿。输出：标准值 envelope、有序事件、真实写入结果或明确错误、通用 UI 状态。

### Acceptance traceability

技术数字和接口选择由 R2 设计提出，并随本计划获批；不冒充用户逐一指定数值。

| AC ID | Source / exact requirement | Task IDs | Verification / independent expectation | Status |
| --- | --- | --- | --- | --- |
| AC-01 | 用户：网络只有节点、网络和资源；节点不是设备；无网络插件 | DOC01、VAR01、QA01 | 无新增 device/plugin 网络对象，无平台对产品反向依赖 | passed |
| AC-02 | 用户：变量格式平台统一，Metrics 只提供数据 | VAR01、PROV01、MET01、QA01 | 枚举全第一方 Catalog，Variable 均标准版本/读事件 schema；无私有变量运行时 | passed |
| AC-03 | 用户：Desktop 通用，无需预装节点 schema | VAR01、DESK01、QA01 | 随机资源路径/语义 ID 的陌生节点 bool/number/enum/object/array 可呈现，未加入内置 schema | passed |
| AC-04 | 真类型与既有通用组件需求 | VAR01、MET01、DESK01 | false、0、空字符串、合法 null、unavailable 区分；0..100 %；101/string false 被 owner 拒绝 | passed |
| AC-05 | 当前值、单调修订、snapshot-first；拟议 read 携带版本 | VAR01、PROV01、DESK01 | read 有定义/版本，无需 subscribe 授权；本地 Set/Apply 与远端写均发布标准事件；并发 snapshot 先于 data，revision 不倒退 | passed |
| AC-06 | 既有 CAS；拟议实例保护与 JSON 精度 | VAR01、DESK01、QA01 | 同版本两写仅一次成功；同名重建拒绝旧 instance；revision 大于 2^53 仍精确 | passed |
| AC-07 | owner/权限原则，不自动授权；用户无需兼容 | VAR01、MET01、LEGACY02、QA01 | read-only 无 write；未授权无副作用；/set 从 Catalog 移除且调用失败；旧 grant 不转译为 write；不改用户 policy | passed |
| AC-08 | 通用输入 + Metrics 实际执行语义 | VAR01、MET01、DESK01、QA01 | actuator 有效写一次，越界/冲突零次；展示确认值，未知结果不伪报成功；真实 Windows smoke | passed |
| AC-09 | Metrics 当前 stale/unavailable | VAR01、MET01、DESK01 | 失败保留最近真实值为 stale，无历史为 unavailable，无零/false 伪值，恢复 ready | passed |
| AC-10 | Desktop 草稿与异步状态保护 | DESK01、QA01 | 时间更新保留草稿；真值变化提示冲突；迟到请求不覆盖新会话；unmount 无订阅残留 | passed |
| AC-11 | 输入有界与不可执行；拟议 limits | VAR01、PROV01、DESK01、QA01 | depth=12/13、properties=128/129、enum=64/65、definition 64 KiB、完整 envelope 上限；所有本地/远端入口非法值均不改变状态/revision/目录；Go/TS 向量一致 | passed |
| AC-12 | 用户：“当前不需要任何兼容，产品还没发布” | DOC01、VAR01、PROV01、MET01、DESK01、LEGACY02、QA01 | 仅新格式/API；旧 payload/写请求拒绝；无 legacy codec/renderer、deprecated 桥接、旧格式/视图迁移器；更新后的领域与产品功能通过 | passed |
| AC-13 | canonical SDK/生成契约；同批替换 | VAR01、MET01、DESK01、LEGACY02、QA01 | freshness、无 TS shadow wire；全 provider/调用方/fixture 已更新，旧变量 schema 常量及生成引用清除 | passed |
| AC-14 | 用户：方案比较后才改代码 | DOC01、QA01 | 规划仅 docs+plan/todo；比较方案后，用户调用 $m-execute 才进入批准 Task 的实施 | passed |
| AC-15 | 现有拓扑与用户改动保护 | PROV01、DESK01、QA01 | children/subtree depth=0/1/n、权限/缓存回归，主检出未提交刷新和草稿不覆盖 | passed |
| AC-16 | 通用消费产品闭环 | DOC01、DESK01、QA01 | 需求/规格/决策/features 与最终生产包一致，缺测不记通过，raw 可见不代替通用展示 | passed |

## Architecture Analysis

### 规划时的当前方案与获批 R2 比较（保留决策依据）

| 边界 | 当前 | 本轮拟议 |
| --- | --- | --- |
| Variable | 公共 byte store + 产品 schema | 共用实现 + 平台类型定义/状态 envelope |
| 类型发现 | Catalog 只给 schema ID，Desktop 内置映射 | read/订阅快照携带 definition |
| Metrics | SampleV1.value string + /set | 真类型 value + 标准 write，旧 /set 与被替代协议删除 |
| Desktop | 读一次或临时订阅取 revision | 持续消费、显式草稿与 CAS |
| 其他变量 | 领域对象直接作 payload | 原对象作标准 value，公共 SDK 解包 |
| 契约支持 | 当前旧变量格式 | 全产品只实现当前新格式，不保留兼容层 |

### Solution / interfaces / flow

唯一当前接口规格为 [规格](../specs/unified-variable-resource.md)：固定平台 schema、read/snapshot 完整 definition、后续 data 不重复定义；JSON 原生 value；ready/stale/unavailable；字符串 uint64 revision；注册实例身份；write 仅允许写值。

Provider -> shared Variable -> Registry -> existing authority/Operation -> SDK -> Desktop controller -> generic renderer。

Apply -> standard write(instance, expected revision, value) -> authority -> common validation/CAS -> optional owner handler -> confirmed state -> response + subscriber update。

无需新增 describe、独立 schema 服务、全树查询或 Catalog 内嵌完整定义。Registry 自身 Catalog 也采用标准变量，但业务 value 仍 ResourceCatalogV2；直接按标准 schema 解码，避免先取 Catalog 才知道怎样读 Catalog。

### Module responsibilities and migration inventory

- protocol/schema_variable.go、data_schema.go：平台 envelope/definition/写请求与验证；不导入产品。
- runtime/resource/variable.go、registry_variable.go、resource.go：共用状态、实例、CAS、编码、声明和 Catalog。
- feature/management/controller.go、admission_authority.go、topology.go：标准变量；特别迁移 topology 自定义 read/Observe/publish，但 children/subtree 保留。
- feature/file/file.go：transfer summaries。
- apps/nodes/clipboard/controller.go：status/config。
- apps/nodes/metrics/controller.go、schema.go、Windows facade/frontend、cmd/mfh-metrics：config/采样/执行与调用方同步更新；显示 DTO 仅由标准状态派生，不保留旧 SampleV1 转换桥。
- sdk/go/client.go、features.go、sdk/bindings/**：统一标准 API 与当前值解包，直接更新调用方和生成契约。
- apps/desktop/app.go、frontend api/types/rendering/components：通用会话、值定义解析、展示编辑。

自定义 Resource 只要声明 mfh.variable，同样接受标准 read/subscribe 验收；不能只 grep NewVariable。

### Errors / concurrency / performance / extension points

- 无效定义/值/版本/超限边界拒绝；未授权拒绝；过期 CAS 冲突；实例变化重新加载；采集失败为标准状态；未知版本 unsupported。
- 当前本地 Set/Apply 和远端 CompareAndSet 一并进入标准校验/发布；Catalog 的资源可见性、编码成功与版本提交作为同一个验收边界。依据现有 observable-side-effects lesson，既检查最终存储，也检查订阅者所见事件。
- mutation gate 串行化条件检查和 owner 执行，状态锁短持有，watcher 锁外调用；产品 handler 不重入同一 Variable.Publish。采集、标准 write 和配置切换需顺序测试；/set 不再存在。
- 外部 actuator 无事务回滚保证；不确认则报告未知，不自动重试，下一次采集校正；不能同步确认的操作继续 Command，异步模型延期。
- 每次状态更新推进 revision；用户草稿仅当新 value 等于 base 时推进 CAS 基准，真实值变化保留旧基准并提示冲突。
- definition 在注册预验证/预编码，缓存绑定 Profile/资源/instance，最后消费者释放时清理。所有大小包括 wrapper 开销，不能提高全局限额掩盖迁移问题。
- definition 递归结构不能直接塞入现有反射生成器无限展开；使用 canonical 语法/类型生成与跨语言 fixtures。
- 标准 write 直接按 exact capability 授权；/set 删除，旧授权不转译，不自动修改用户 policy 或把 capability 当有效授权。
- 删除旧 codec/API 与冗余 schema/renderer；测试只保留最小旧格式拒绝样本，不保留一份完整旧实现充当测试依赖。
- 其他类型/非同步执行为扩展点，本轮不建新服务或执行框架。

## Docs Governance Routing and Impact

已显式使用 $m-docs。无私有 docs 位置歧义，无需修复文档树；用户无需兼容的明确纠正优先于 R1 计划和既往阶段安排。规划时同步原话、长期需求、Proposed R2 规格/决策与索引；本次执行完成后，规格为 Current R2、决策为 Implemented R2，features 与 lessons 已按实际证据更新。下表保留原批准的文档路由，最终状态见执行记录。

| Category | Impact / destination |
| --- | --- |
| intake | add，已写 docs/intake/2026-09-06_unified-variable-contract.md |
| requirements | add/clarify，已写 unified-variable-resource；DOC01 补 extensible-resource-platform、desktop-resource-workspace 入口/约束 |
| specs | add Proposed unified-variable-resource；实施时对齐 resource-platform-v2、desktop-schema-rendering、protocol_map |
| decisions | add Proposed R2 2026-09-06_platform-variable-contract；记录取消 R1 兼容，批准后 Accepted、验证后 Implemented |
| features | clarify，等待实现后更新 desktop、metrics-node、clipboard-node、file-transfer、hub 受影响段落 |
| lessons | 验证后更新 observable-side-effects-and-generated-contracts、collection-browser-cross-runtime-contract-boundaries；不新建空洞复盘 |
| indexes | intake/requirements/specs/decisions 已更新；后续新增叶子更新最近索引；docs/README 分类不变 |
| change / archive | 本轮不生成完成归档，执行后交 $m-archive |

相关入口：docs/requirements/unified-node-runtime.md、extensible-resource-platform.md、desktop-resource-workspace.md；docs/specs/resource-platform-v2.md、desktop-schema-rendering.md、topology-discovery.md；docs/decisions/2026-08-30_provider-schema-desktop-renderer-ownership.md。

已查 lessons 入口及相关记录：observable-side-effects-and-generated-contracts、collection-browser-cross-runtime-contract-boundaries。执行还需注意 partial-discovery-and-async-ownership、wails-binding-proto-drift、frontend-worktree-wailsjs-missing 的异步/生成约束。

## Execution Scope After Approval

下表是获批的唯一任务清单。Will execute 为 R2 原定执行范围，已获用户 $m-execute 批准；实际完成状态见 [todo.md](todo_archive_2026-10-09_unified-variable-contract.md)。所有本轮任务恰好属于一个范围。

| Task ID | Title | Scope | Files / Modules | Acceptance / Tests | Risk / Notes |
| --- | --- | --- | --- | --- | --- |
| DOC01 | 固化产品边界与一次性替换契约 | Will execute | docs requirements/specs/decisions/features/indexes | AC-01、12、14、16；无需兼容边界明确 | 无阻塞；R2 草案供审阅 |
| VAR01 | 平台到 SDK 的标准变量闭环 | Will execute | protocol、runtime/resource、sdk/go、sdk/bindings | AC-01..07、09、11..13；新格式 read/write/subscribe | 依赖 DOC01；原位替换，Catalog bootstrap |
| PROV01 | 管理、文件与 Clipboard 迁移 | Will execute | feature/management、file、apps/nodes/clipboard、领域 SDK | AC-02、05、11..13、15；目录与领域回归 | 依赖 VAR01；topology read/Observe/Apply |
| MET01 | Metrics 真类型与实际写入 | Will execute | Metrics、Windows UI/adapter、CLI/tests | AC-02、04、07..09、12、13；采样/actuator | 依赖 VAR01；删除 /set，同步全部调用方 |
| DESK01 | 通用展示、实时状态与编辑 | Will execute | apps/desktop、frontend controller/rendering、bindings | AC-03..11、13、15、16；陌生节点与草稿 | 依赖 VAR01；可用通用 fixture 独立验证 |
| LEGACY02 | 清除旧契约残留并禁止双轨 | Will execute | 旧变量 schema/API 引用、fixtures、生成/静态检查 | AC-07、12、13；无旧协议/别名/转换层残留 | 依赖 PROV01、MET01、DESK01；原延期任务改为必做 |
| QA01 | 全提供者与生产包集成 | Will execute | integration/产品测试、生成物、docs 事实核对 | AC-01..16；Go/race/生成/UI/Windows TCP | 依赖 LEGACY02；只验收同一新契约 |
| EXT02 | 其他资源 schema 与异步执行模型 | Will not execute now | 非 Variable/独立服务/执行状态机 | 不在本轮验收 | 超出本轮需要，独立讨论 |

旧 LIVE02、SEARCH02、AUTHZ02、HOST02、Flow/mobile/embedded 重设计、发布待办归先前工作流，本轮不重定义也不授权；原始记录保留在历史计划/长期需求。

### DOC01 — 文档合同

- Owner：主 Agent；Worktree/Plan Path 同顶部。
- Deliverable：冻结已批准接口，澄清旧“仅内置 schema”阶段边界与提供者/平台职责；最终与 QA01 核对 Current 状态。
- Blocked by：无。
- Files / Write Set：上述 Docs Governance 所列受影响文档、索引、plan/todo；不改 guide、无关文档和历史归档。
- Acceptance：AC-01、12、14、16。
- Verification / Tests：双向 requirement -> AC -> Task -> evidence，本地 Markdown 链接、版本/状态对照；不需运行时测试。
- Rollback：仅回退本工作树本轮文档，保留 intake 历史确认；不更改主检出草稿。

### VAR01 — 平台到 SDK 的完整行为切片

- Owner：批准后分配的平台 Agent；Worktree/Plan Path 同顶部。
- Deliverable：无 Metrics 依赖的 fixture 节点经 memory/TCP 完成标准声明、read/write/subscribe，SDK 能取定义/状态，Catalog 自举成功。
- Blocked by：DOC01。
- Files / Write Set：protocol/schema_variable.go、data_schema.go 与必要 contract；runtime/resource/**；sdk/go/client.go、标准 variable helper；sdk/bindings/**、生成器；标准变量 integration fixture；现有共享包内对应调用方/tests。
- 原位替换声明与操作接口，包含 definition/value 验证、二进制标准值、instance/revision、状态、owner handler、只读/CAS、原子订阅；删除对应旧 codec，Catalog 自身在此更新。
- 覆盖现有本地 Set、CompareAndSet、Apply 与 Registry 注册/注销的 Catalog 发布路径；完整 envelope 编码/校验失败不能留下已变更的值/revision/资源可见性，成功事件全部采用标准格式。
- 给下游冻结统一 API 和 fixtures，所有调用方同步修改，不保留旧函数签名/deprecated 桥。领域 helper 仅解包当前标准值；Operate 返回真实 wire。
- Acceptance：AC-01..07、09、11..13。
- Verification：独立期望值；本地 Set/CompareAndSet/Apply 与远端 write 的成功事件/失败不变性；并发订阅；冲突不调 actuator；大 revision/实例复用；超限无 Catalog 半发布；新 API 和旧请求拒绝。
- Tests：protocol/runtime/resource/sdk focused test、memory/TCP fixture、共享包 race、go generate ./sdk/bindings freshness。
- Rollback：在本集成分支回退相关 Git 变更；下游依赖后需整组回退和匹配构建，不以保留旧运行时模式回退。中间未更新产品可暂不编译，禁止为维持旧测试引入兼容。

### PROV01 — 第一方其他提供者迁移

- Owner：批准后分配的迁移 Agent；Worktree/Plan Path 同顶部。
- Deliverable：管理、文件摘要和 Clipboard 全部变量被标准消费者理解，原业务读取/更新与持久化保持。
- Blocked by：VAR01。
- Files / Write Set：feature/management/controller.go、admission_authority.go、topology.go/tests；feature/file/file.go/tests；apps/nodes/clipboard/controller.go/tests/本地标准 definition；sdk/go/features.go 仅领域迁移。
- topology 自定义 read 直接返回旧 payload，须与 base Observe/publish、Apply(current.revision+1, value) 一起迁移；不能留下 raw 本地注入入口。children/subtree 的 schema/depth/权限保持。
- 领域 value 内自己的 version/revision 与外层变量版本分层，不删除业务字段；不能只改构造函数。
- Acceptance：AC-02、05、11..13、15。
- Verification：逐 provider Catalog/read/subscribe；目录自身可读；管理 query depth=0/1/n；file summaries、Clipboard status/config/桥接一致。
- Tests：相关 feature/clipboard Go、SDK 领域、既有 topology integration、必要 bridge 构建；UI 未变不新增无关 UI 测试。
- Rollback：依赖接口已改变时整组 Git/构建回退，不更改真实配置/历史，不提供双轨开关。中间切片通过不等于平台最终交付。

### MET01 — Metrics 采集与控制切片

- Owner：批准后分配的 Metrics Agent；Worktree/Plan Path 同顶部。
- Deliverable：10 项采样和 config 使用标准 Variable；音量/静音/亮度通过标准 write 返回真实结果，/set 从目录和实现中删除。
- Blocked by：VAR01。
- Files / Write Set：apps/nodes/metrics/controller.go、schema.go、tests/process、platform/windows、windows facade/frontend/bindings；cmd/mfh-metrics 和其他旧采样/控制调用方。
- 采集器/OS adapter 可使用平台接口实际需要的内部类型，但不受旧 API 签名约束。删除被取代的 SampleV1/ControlV1/ControlResultV1 协议与解码；本机状态从标准值派生，同步改 facade/UI/CLI。
- 不重做配置/通知业务，不新建 metrics.json 迁移器。write、collector 共用标准状态与 per-metric 顺序；配置禁用、平台不支持、未知结果明确失败。
- Acceptance：AC-02、04、07..09、12、13。
- Verification：fake 采样/执行的值与次数；越界/冲突/未授权零次；成功发布确认值；stale/unavailable 恢复；/set 无注册且旧调用拒绝；当前格式配置与本机 UI 一致。
- Tests：go test ./apps/nodes/metrics/...、focused race、process/TCP；同步后的 Metrics Vitest、Wails 生成/生产构建及 CLI 编译。
- Rollback：整组匹配 Git/构建回退；不删除用户配置/身份，不保留旧网络注册开关。物理效果不由 Git 回退，真实 smoke 记录并恢复原状态。

### DESK01 — 通用消费者切片

- Owner：批准后分配的 Desktop Agent；Worktree/Plan Path 同顶部。
- Deliverable：Inspector/View 共用 controller，以标准 definition 显示、订阅、编辑随机路径陌生节点，无 Metrics schema 依赖。
- Blocked by：VAR01；不等待 MET01，通用 fixture 可独立验证。
- Files / Write Set：apps/desktop/app.go/tests、Wails bindings；frontend api/types、rendering/schema/registry、拟新增 rendering/variable.ts/controller、components/Renderer/tests；App 仅接线。
- 复用现有控件和 Reset/Apply，不新增 Metrics renderer、/set 猜测或产品 UI 重设计。
- 会话绑定 Profile/owner/name/instance，持续有界订阅，草稿/CAS/刷新/写响应合并与旧结果隔离。
- Acceptance：AC-03..11、13、15、16。
- Verification：API fake+真实 fixture；bool/number/enum/object/array/bytes；只读/可写；read allowed/subscribe denied；失败保草稿；取消/重连/大 revision；旧变量格式明确 unsupported，无 legacy renderer/别名/视图迁移。
- Tests：Vitest controller/renderer/API、TypeScript/Vite、Go facade/binding、Wails build；截图不代替实际 API 断言。
- Rollback：回退同批匹配构建；不开发旧 View 转换，真实本地文件不自动清理；不得覆盖主检出刷新改动。

### LEGACY02 — 清除残留与禁止双轨

- Owner：主 Agent 顺序集成；Worktree/Plan Path 同顶部。
- Deliverable：在 VAR01/PROV01/MET01/DESK01 已直接替换的基础上，清除漏下的旧变量协议、API、别名、renderer、生成引用与旧成功路径 fixtures，并增加无双轨门禁。Task ID 沿用 R1，范围由延期改为本轮必做。
- Blocked by：PROV01、MET01、DESK01。
- Files / Write Set：上述受影响 protocol/runtime/sdk/apps/cmd 的旧变量残留；相关 tests/integration、生成契约/检查；非历史当前文档中的旧合同表述。对共享文件只在各切片结束后修改，不与实现并行。
- 不在此任务重新实现已被前序任务负责的功能，也不删除无关历史协议、归档或身份/策略迁移代码。
- Acceptance：AC-07、12、13。
- Verification：生产目录无 /set；旧 Variable schema/API/codec 不再有运行/生成入口；不存在 deprecated 转发、旧格式猜测或 legacy renderer；最低限度旧 payload/写请求拒绝样本即可，不能携带完整旧实现。
- Tests：静态/生成覆盖检查、新契约拒绝测试与受清理模块单测/编译；随后 QA01 执行整体门禁。
- Rollback：整组 Git/匹配构建回退；不重新加入兼容开关作为功能回退。

### QA01 — 集成并逐项证明

- Owner：主 Agent 集成，批准后可分配独立验证 Agent；Worktree/Plan Path 同顶部。
- Deliverable：AC 全有证据，全第一方迁移，真实 Windows Desktop+Hub+Metrics+陌生 fixture 联调与文档一致。
- Blocked by：LEGACY02（已包含三个实现切片完成）。
- Files / Write Set：tests/integration、必要黑盒测试、生成物、本轮 plan/todo 与 docs Current 核对；out/ 运行证据；不借 QA 做无关重构。
- Acceptance：AC-01..16。
- Verification：陌生 provider、跨子树 read/write/subscribe、撤权/重连/并发 actuator/资源重建；全生产 Variable 声明及自定义包装审计；只支持新格式的负向验证与更新后产品功能回归。
- 原联调进程不重启/占用；测试端口、身份、state 在本工作树 out 隔离生成。生产包从本源码构建并记录 hash，防止旧 dist 假通过。
- 用户后续要求：自动联调使用后台浏览器，不抢占其鼠标、键盘和桌面焦点。保留已完成的生产包 GUI 与物理音量/静音证据；补充验证使用同一生产 dist、真实 Wails 开发桥和 Go 后端。浏览器桥不冒充原生安装包，原生特有功能不借浏览器结果声明通过。
- Windows 控制先记录音量/静音或亮度原状态，做有界改动并恢复；不触发无关邮件/剪贴板对外发送。
- Tests：下节矩阵；缺环境明确 blocked/unavailable，不能以 AllowUnavailable 视作通过。
- Rollback：整批匹配版本回退；仅核实本轮 PID 后停 fixture；不改真实 identity/policy/Profile/state，无部署/发布授权。

## Approved validation plan（规划阶段未运行，实际结果见执行记录）

所有命令从 Active Worktree 执行，GOWORK=off。先每个切片的必要测试，再一次总体门禁；无新变更/失败不反复扩大测试。

| Gate | Command / observation | Expected result |
| --- | --- | --- |
| 平台 | go test ./protocol ./runtime/resource/... ./sdk/... | 类型/值/CAS、新格式接口与旧请求拒绝通过 |
| 提供者 | go test ./feature/management/... ./feature/file/... ./apps/nodes/clipboard/... ./apps/nodes/metrics/... | 标准变量和原领域行为通过 |
| 网络 | go test ./tests/integration/...；Metrics process tests | 未知 provider、跨子树授权与 topology 回归 |
| 并发 | go test -race ./runtime/resource/... ./feature/management/... ./apps/nodes/metrics/... ./sdk/go/... | 无 race/deadlock，发布和执行顺序正确 |
| 生成 | scripts/mfh.ps1 -Action generate -Target generated；freshness | 只有预期契约，无手工 shadow/旧工作树产物 |
| Desktop | frontend 的 npm ci、npm test -- --run、npm run build；canonical build | Vitest、TS/Vite、Wails production 通过 |
| Metrics | canonical metrics test/build，受影响 UI tests | Windows 包/本机投影/UI/CLI/当前配置一致 |
| 总体 | go test ./...；go vet ./...；scripts/mfh.ps1 -Action check -Target all | 共享契约影响覆盖当前产品，不恢复退役平台 |
| 真实观察 | 隔离 Hub+陌生 fixture+Metrics+Desktop 生产包 | 通用控件、实际写入、失败/恢复/无泄漏达到 AC |
| 文档 | 链接、AC/Task 双向映射、状态、git diff 范围 | 无缺证据 Current 声明，无用户改动覆盖 |

执行时从 scripts/mfh.ps1 的 list/具体定义确认真实 target 名，不猜命令。Clipboard Flutter 门禁如环境缺失明确记录并解决，不能宣称共享协议全产品验证完成。

已核对脚本：当前 target 为 all/core/hub/desktop/metrics/clipboard/generated。check all 包含架构/协议、全 Go vet 及各产品检查；全模块 Go test 和并发集成另行验证，避免误把部分包测试视为全模块测试。

## Dependencies / parallelism

初始化和规划由主 Agent 完成，未派发实现子 Agent。

顺序：DOC01 -> VAR01 -> {PROV01、MET01、DESK01} -> LEGACY02 -> QA01。VAR01 先以新格式 fixture 验证公共切片，提供者/界面更新可并行；LEGACY02 清除残留，QA01 才承诺整个平台完成。不采用 expand/compatibility 阶段。

本次用户明确无需兼容，优先于技能的一般迁移分批建议。平台接口替换后，未完成的下游暂时编译/测试失败可以作为显式集成未完成状态；不得隐藏失败或交付半成品，也不得为中间全绿写兼容桥。所有产品编译、生成与整体测试在最终新契约上必须通过。

- protocol、sdk/go/client.go、sdk/bindings/生成器归 VAR01；下游不并行改共享契约。
- PROV01 的 sdk/go/features.go 仅领域适配，公共 API 问题交主 Agent 顺序整合。
- MET01 定义留产品包；DESK01 不依赖其类型；bindings/dist 由主 Agent 顺序生成。
- LEGACY02 只在三个切片完成后顺序清理跨模块残留，QA01 不得在该门禁之前宣称完整验收。
- 文档由主 Agent/DOC01 与 QA01 收口，避免实现 Agent 覆盖彼此的计划状态。
- 批准后子 Agent 上下文包必须含：工作树绝对路径、本 plan/spec、Task/AC、明确写集、Base/保护状态、冻结接口、错误与验证矩阵。
- 未授权执行前不派实现；本阶段不提交/合入/推送。后续执行阶段的 Git 行为按用户授权与相应技能处理。

## Risks and decision gate

以下为规划阶段历史检查，不充当实现证据。R1 规划检查不作为 R2 通过证据。R2 已重新通过 git diff --check、本地链接、8 个唯一 Task ID（7 执行/1 延期）、16 个 AC 和无环依赖检查；仅文档写集，主检出 HEAD 与既有修改清单保持。未运行实现测试。输出采用 Markdown 比较/任务表，单一拟议范围无需交互附件。

以下显著选择已随 R2 获批，执行保持这些边界：

1. 按用户要求不支持旧二进制/旧格式，接口和调用方同批更新，明确拒绝旧请求；协议版本仍保留标识作用，没有版本协商或兼容层。
2. “全平台统一”直接要求迁移全部第一方 Variable，影响超过单个 Metrics UI。
3. 持续订阅、草稿/CAS、公共 owner handler 属于完整交付，不能只改 SampleV1.Value 类型。
4. 标准 write 替代 /set；旧命令及其协议删除，旧 grant 不转译，不自动改用户 policy。测试直接配置当前权限。
5. 主检出刷新 UI 不在 d77dc54 基线；集成和生产包比较必须保留并说明其来源。

- Approved execution：DOC01、VAR01、PROV01、MET01、DESK01、LEGACY02、QA01。
- Will not execute now：EXT02。
- Blocked：no；R2 已获 $m-execute 批准。
- 实施子 Agent 按本计划的分离写集工作；最终证据由主 Agent 汇总。
- 本阶段不生成 docs/change，不提交、合入、推送或清理工作树。

## Execution log

- 2026-09-06：R2 已获执行批准；DOC01、VAR01 in progress。采用 protocol 与 runtime 独立写集，接口冻结后并行提供者、Metrics、Desktop；最终由主 Agent 集成和验收。无运行时兼容分支；保持当前手动联调进程和主检出修改。

- 2026-09-06：VAR01 接口冻结后按分离写集完成 PROV01、MET01、DESK01，主 Agent 顺序完成 LEGACY02/QA01；DOC01 同步 Current R2 规格、Implemented R2 决策、features、索引及既有 lessons。7 个批准 Task 均 completed，EXT02 仍延期。
- 集成审查修复了注销时实例失效边界、typed owner handler 执行前取消检查、watcher panic 导致事件排空中断；定向回归及 race 复核通过。修复默认 renderer ID 误报不支持，最终 Desktop 438 项测试与生产构建通过。
- Go 全模块、vet、相关 race、canonical check/build all、生成新鲜度通过。生产包 GUI 已验证陌生节点 bool/slider/object；后续按用户要求使用后台浏览器，验证 enum/array 提交回读、订阅超过 4 分钟、视图重载恢复与真实 Metrics CPU 更新。未操作用户前台输入。
- Windows 音量/静音标准 write 实测并恢复；物理亮度未测，计划中的“音量/静音或亮度”由前者满足。没有把开发浏览器桥声明为原生安装包验收。
- 后台测试使用同一生产 dist、真实 Wails dev bridge 与 Go/TCP 后端。测试标签已关闭，核实 PID/可执行路径后停止本轮 Desktop/Metrics；fixture 收到 STOP 正常 PASS。保留工作树、证据及隔离配置。

### Final evidence and review

- [完整执行验收报告](../../../../artifacts/m-execute/unified-variable-contract/REPORT.md)：AC-01..16 与 Task/文件/验证逐项对应。
- [实际候选身份](../../../../artifacts/m-execute/unified-variable-contract/candidate-identity.json)：固定 base、HEAD、dirty/untracked 内容、删除/模式、计划定义投影及相关生成物哈希。证据在源码工作树之外，避免自引用。
- Requirements：passed；Standards：passed。无未解决发现或豁免；延期 EXT02 和未测物理亮度不计为通过项。
- 主检出未做写入；最终 HEAD 与指定 base 相同。其预存节点树刷新/文档草稿属于其他改动，本工作树生产包未包含它们，未来合入必须保留。
- 本阶段没有提交、合入、推送、归档或清理工作树。回退必须整批使用匹配的源码与构建，不以恢复旧 Variable 兼容层回退。

## 2026-10-09 归档补充：用户手动验收范围

计划标识 R2 + UI-20261009。原 AC-01..16 与任务批准保持；用户后续直接请求扩展 DESK01/QA01，见 [追加 intake](../intake/2026-10-09_desktop-variable-manual-acceptance.md)。Inspector 显式 Apply 保持，View 自动应用按最后确认的控件提交时机执行，取代原计划中对所有呈现场景一概手动提交的限制。

| 补充验收 | 任务 | 最终行为 / 证据 |
| --- | --- | --- |
| UI-01 | DESK01, QA01 | shadcn 通用控件、统一标题/圆点、View 简洁布局；完整前端与 UI 报告 |
| UI-02 | DESK01, QA01 | View 本地提交意图、Enter/blur/滑块提交、控件加载态、owner 确认；VariablePanel/control tests 与真实 control-commit |
| UI-03 | DESK01, QA01 | 只读复杂值全宽有界显示、权限拒绝保留快照；catalog-preview 和组件测试 |
| UI-04 | DESK01, QA01 | 分栏比例与最小尺寸共同满足，布局结构变化丢弃旧预览；28 项布局测试及 10 组真实浏览器布局 |

完成状态与最新 488 项前端测试、生产构建、候选指纹、Requirements / Standards 审查见 [归档](../change/2026-10-09_unified-variable-contract.md)。EXT02 仍延期；无需物理亮度设备的替代验收已通过实际音量写入/恢复，不声称验证了物理亮度。
