# Plan - 文档一致性（MyFlowHub3）

> 位置：`d:\project\MyFlowHub3\worktrees\docs-consistency\plan.md`  
> 本 plan 仅处理“文档一致性”（不改 wire、不改业务逻辑），目标是让接手者以文档为入口即可理解当前架构与推进状态。

---

## Workflow 信息

- 范围：
  - ✅ 更新 MyFlowHub3 控制面文档（`target.md` / `repos.md` / `plan.md` / `docs/specs/protocol_map.md`）
  - ✅ 更新关键仓库 README（`MyFlowHub-Win` / `MyFlowHub-SDK`）
  - ❌ 不修改任何协议 wire（action 名称、SubProto、JSON schema、HeaderTcp 语义）
  - ❌ 不做“minimal/full 变体产品化”（用户要求暂不处理）
- 验收方式：
  - 文档内容自洽、无冲突、无过期路径/过期术语
  - `rg` 搜索关键过期关键词无命中（见各任务验收）

---

## 1) 需求分析

### 目标
1) 统一“仓库职责/依赖方向/协议归属”的表述：Proto 是协议字典，Server 的 `protocol/*` 仅为兼容壳（推荐新代码直引 Proto）。
2) 统一“PR2+ 重构路线图/编号约定”：消除 `repos.md` 与 `plan.md`/归档文档的编号冲突，并明确未来的编号规则。
3) 统一“开发/验收命令与路径”：避免 README 指向已不存在的 worktree 名称或历史路径。

### 不做
- 不改代码、不改接口、不改功能；仅调整文档表达与结构。

### 验收标准（全局）
- `d:\project\MyFlowHub3\repos.md`、`target.md`、`plan.md`、`docs/specs/protocol_map.md` 之间无互相矛盾的描述。
- 文档内出现的路径在当前工作区语义上成立（`repo/*` 为控制面；worktree 仅用于实现与分支隔离）。
- 不再出现“把协议定义放在 `myflowhub-server/protocol/*` 作为主入口”的误导性表述（需要改为：Proto 为主，Server/protocol 为兼容壳）。

---

## 2) 架构设计（分析）

### 总体方案（采用）
- 建立“文档分层 + 单一事实来源”的一致性策略：
  - `target.md`：高层目标/已确认决策/里程碑（偏“为什么/做成什么样”）
  - `repos.md`：仓库职责/依赖边界/推进路线图（偏“怎么分工/怎么推进”）
  - `plan.md`：全局 workflow 归档与审计记录（偏“做过什么/证据在哪里”）
  - `docs/specs/protocol_map.md`：协议映射速查表（偏“具体 action 与数据结构在哪里”）
- 对“编号冲突”的处理原则：
  - 已归档的编号不追溯修改（保持可审计）。
  - `repos.md` 的未来路线图以“归档中已使用的编号”为准；对尚未开展的事项，如需改号则在文档中明确“已更名/已延后”。

### 备选（不采用）
- 将所有历史文档全部重写为同一套编号：风险高、审计成本高、容易引入新错误。

---

## 3.1) 计划拆分（Checklist）

> 说明：每个任务都必须包含“目标/涉及文件/验收/回滚点”。本 workflow 不涉及自动化测试；验收以文档一致性检查为主。

### DC1 - 文档现状审计与差异清单
- 目标：列出当前不一致点与拟修改项，避免“拍脑袋大改”。
- 涉及文件（只读扫描）：
  - `d:\project\MyFlowHub3\target.md`
  - `d:\project\MyFlowHub3\repos.md`
  - `d:\project\MyFlowHub3\plan.md`
  - `d:\project\MyFlowHub3\docs\protocol_map.md`
  - `repo/MyFlowHub-Win/README.md`
  - `repo/MyFlowHub-SDK/README.md`
- 验收：
  - 在 plan 中形成“差异点列表”（含修改策略：修正/补充/保留但加注释）。
- 回滚点：
  - 无代码改动，无需回滚。

#### DC1 现状差异（已审计，2026-02-18）

> 结论：`repos.md` / `target.md` 的“Proto 为主、Server/protocol 为兼容壳”口径已基本一致；主要不一致集中在 **全局 `plan.md`** 与 **`docs/specs/protocol_map.md`**，以及两个仓库 README 的过期路径/叙述。

1) `d:\project\MyFlowHub3\plan.md`
   - 标题仍为 “MyFlowHub-Win Wails Refactor”，但文件尾部已承担“全局 workflow 归档”职责，定位不一致。
   - 仍存在将 `MyFlowHub-Server/protocol/*` 作为复用主入口的表述（与 PR1/Proto 抽离后的事实冲突）。
   - 存在历史约束语句（例如“保持 Core 不变、Server 仅允许 protocol/* 导出”）与当前已完成的 PR2+ 改动事实冲突，需要显式标注“历史约束已失效/仅适用于当时”。
2) `d:\project\MyFlowHub3\docs\protocol_map.md`
   - 明确写了“复用 public `MyFlowHub-Server/protocol/*` models”，与当前推荐（直接依赖 `myflowhub-proto`）冲突；需要改为“Proto 为主，Server/protocol 为兼容壳”。
3) `repo/MyFlowHub-Win/README.md`
   - 仍引用历史 worktree 名称 `myflowhub-win_remove-fyne`（路径已过期）。
   - 冒烟步骤偏“Register/Login”，建议与全局口径一致：最小冒烟以 `management node_echo` 为主（Register/Login 作为可选）。
4) `repo/MyFlowHub-SDK/README.md`
   - 同时出现 “v1 是计划” 与 “v1 已实现” 两段矛盾表述；需要统一为“v0.1.0 已包含 session/transport + awaiter 能力”，并明确未来计划从 v2 开始。
5) PR 编号口径
   - `plan.md` 的归档中存在 PR2-10a/PR2-9a 等编号；`repos.md` 的路线图使用 PR2-1a/1b/2a/2b/2c/3/4/5。两者并不直接冲突（一个是归档编号、一个是路线图编号），但需要在文档中明确“编号体系来源不同，如何对照”以避免误读。

### DC2 - 更新 MyFlowHub3 控制面文档（统一口径）
- 目标：修正过期表述（例如“协议模型来自 myflowhub-server/protocol/*”）并统一编号/路线图口径。
- 涉及文件：
  - `d:\project\MyFlowHub3\repos.md`
  - `d:\project\MyFlowHub3\target.md`
  - `d:\project\MyFlowHub3\plan.md`
  - `d:\project\MyFlowHub3\docs\protocol_map.md`
- 验收：
  - `rg -n \"MyFlowHub-Server/protocol\\*\" plan.md docs/specs/protocol_map.md` 不再出现“作为主入口”的误导语（允许在“兼容壳说明/历史归档”里出现）。
  - `repos.md` 的 PR2+ 路线图编号不再与 `plan.md`/归档里已使用编号冲突。
  - `plan.md` 标题与内容定位匹配（不再表现为“仅 Win Wails 迁移计划”）。
- 回滚点：
  - 将四个文档恢复到修改前版本（手动回退；本目录非 git 管理）。

### DC3 - 更新关键仓库 README（Win/SDK）
- 目标：修正过期路径与过期叙述，确保新同事按 README 即可跑通基本开发与验收。
- 涉及仓库与文件：
  - `MyFlowHub-Win`：`README.md`
  - `MyFlowHub-SDK`：`README.md`
- 执行方式：
  - 通过 git worktree + 分支完成（不在 `repo/*` 直接改）。
- 验收：
  - Win README 不再引用历史 worktree 名称（例如 `myflowhub-win_remove-fyne`）。
  - SDK README 不再出现“v1 是计划”与“v1 已实现”互相矛盾的段落。
- 回滚点：
  - 两仓各自回滚 merge commit（或 revert commit）。

### DC4 - Code Review（文档审查）
- 目标：按“需求覆盖/一致性/可交接/风险”逐条审查（见阶段 3.3 输出模板）。
- 验收：
  - Review 结论为“通过”，否则回到 DC2/DC3 修正。

### DC5 - 归档变更
- 目标：生成全局归档文档（`docs/change/YYYY-MM-DD_docs-consistency.md`）说明本次“为何修改/改了什么/如何验证/如何回滚”。
- 验收：
  - 归档文档存在且内容可审计（包含映射到 DC1~DC3）。

---

## 问题清单（阻塞：否）

- 约束已明确：本 workflow 仅做文档一致性调整，不涉及 wire/实现逻辑。
- “minimal/full 变体产品化”按用户要求暂不处理：仅在文档中标注为 Deferred。


