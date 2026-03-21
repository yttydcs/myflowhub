# 变更归档：文档一致性整理（MyFlowHub3）

日期：2026-02-18  
Workflow：`worktrees/docs-consistency`  
性质：仅文档/README 一致性修正（不改 wire、不改业务实现）

---

## 1) 变更背景 / 目标

随着 PR1/PR2/PR19 推进，MyFlowHub 的关键事实已发生变化并固化为“目标态”：
- 协议字典已抽离为独立仓库 **MyFlowHub-Proto**（`github.com/yttydcs/myflowhub-proto/protocol/*` 为 canonical source）
- `github.com/yttydcs/myflowhub-server/protocol/*` 变为 **兼容壳**（alias 到 Proto，仅用于保留旧 import path）
- 客户端侧基础能力（session/transport/await）已下沉到 **MyFlowHub-SDK**，Win 应尽量只“调用 SDK”，不重复实现协议机制

但部分文档仍残留历史表述（例如把 `myflowhub-server/protocol/*` 当作复用主入口、README 指向已不存在的 worktree 路径、对 SDK v1 的“计划/已实现”自相矛盾等），影响接手与审计。

本次变更目标：
1) 统一 MyFlowHub3 控制面文档的口径，使其与已落地事实一致且互不冲突。  
2) 修正 Win/SDK 两仓 README 的过期路径与矛盾叙述，保证按 README 可直接执行基本开发与冒烟。  
3) 将当前决策明确化：**暂不推进 “minimal/full 变体产品化”（PR2-4）**，并标注为 Deferred，避免误导。

---

## 2) 具体变更内容（新增/修改/删除）

### 2.1 MyFlowHub3 控制面文档（本地工作区，非 git repo）

修改文件：
- `d:\project\MyFlowHub3\plan.md`
  - 文件定位纠偏：从“Win 单项目计划”调整为“全局计划与归档”，并将原 Win 迁移计划标注为 Legacy（保持可审计）。
  - 统一协议口径：明确 Proto 为 canonical；Server/protocol 为兼容壳。
  - 标注 PR2-4（minimal/full）为 Deferred。
- `d:\project\MyFlowHub3\repos.md`
  - 将 PR2-4 明确标注为 Deferred（按当前决策暂缓）。
  - 补充编号口径说明：`plan.md` 归档中可能有更细粒度编号（例如 PR2-9a/PR2-10a），路线图按更粗粒度聚合展示。
- `d:\project\MyFlowHub3\target.md`
  - 在“PR2 建议切入点”处补充“已完成”的事实说明，并提示以 `repos.md`/`plan.md`/`docs/change` 为准。
  - 将 “minimal/full 变体产品化” 标注为 Deferred。
- `d:\project\MyFlowHub3\docs\protocol_map.md`
  - 将协议模型来源从 `myflowhub-server/protocol/*` 改为 `myflowhub-proto/protocol/*`，并明确 server/protocol 为兼容壳。
  - Exec 的 payload 类型改为引用 `myflowhub-proto/protocol/exec`（与现状一致）。

### 2.2 仓库 README（通过独占 worktree + 分支提交，待合并）

MyFlowHub-Win（分支：`chore/docs-consistency`，commit：`f8a44e0`）：
- 修正 README 中的历史 worktree 路径 `myflowhub-win_remove-fyne` → `MyFlowHub-Win`
- 将最小冒烟步骤收敛为：Connect 后在 Presets 中执行 Node Echo（更贴合全局 `management node_echo` 冒烟口径）

MyFlowHub-SDK（分支：`chore/docs-consistency`，commit：`f8cf23f`）：
- 消除 README 内“v1 是计划”与“v1 已实现”的矛盾：统一为“截至 v0.1.0 已包含 v0(session/transport) + v1(await) 能力”
- 统一联调/验收说明：联调用 `go.work`（不提交），单仓验收使用 `GOWORK=off go test`

---

## 3) 对应 plan.md 任务映射（worktrees/docs-consistency/plan.md）

- DC1：文档现状审计与差异清单（已完成并写入 plan）
- DC2：更新 MyFlowHub3 控制面文档（已完成：plan.md / repos.md / target.md / docs/protocol_map.md）
- DC3：更新关键仓库 README（已完成：Win/SDK 各自分支提交，待合并）
- DC4：Code Review（已通过）
- DC5：归档变更（本文件）

---

## 4) 关键设计决策与权衡

1) **不追溯重写历史归档**  
   - 选择：在 `plan.md` 顶部增加全局说明，并将旧内容标注为 Legacy。  
   - 原因：避免破坏可审计性/历史上下文；降低改错风险。

2) **Proto 为协议字典单一事实来源**  
   - 选择：在所有“协议映射/类型来源”文档中，明确 `myflowhub-proto/protocol/*` 为 canonical。  
   - 原因：避免 Server/Win/SDK 各自维护一份类型导致漂移；server/protocol 仅作为兼容壳保留旧 import path。

3) **Deferred 明确化（PR2-4）**  
   - 选择：把 “minimal/full 变体产品化” 在 `repos.md/target.md/plan.md` 中显式标注 Deferred。  
   - 原因：减少接手者在当前不做的方向投入时间；恢复时再以独立 workflow 推进。

---

## 5) 测试与验证方式 / 结果

本 workflow 不涉及代码逻辑改动，验收以“文档一致性与关键词扫描”为主：
- Win README：`myflowhub-win_remove-fyne` 已消失（路径已修正）
- SDK README：不再出现 “v1 计划/已实现” 矛盾段落
- `docs/protocol_map.md`：已明确以 `myflowhub-proto/protocol/*` 为主入口，并注明 server/protocol 为兼容壳
- `plan.md`：标题与定位已与现状一致（全局计划与归档）

---

## 6) 潜在影响与回滚方案

潜在影响：
- 仅影响文档阅读与接手方式；不改变任何协议 wire、运行时行为与构建产物。

回滚方案：
- MyFlowHub3 控制面文档：均为纯文档修改，可按本归档第 2 节反向编辑回退。
- MyFlowHub-Win / MyFlowHub-SDK：如不接受 README 改动，可在各自仓库对 `chore/docs-consistency` 分支提交执行 `git revert` 或直接丢弃该分支（未合并前成本最低）。

