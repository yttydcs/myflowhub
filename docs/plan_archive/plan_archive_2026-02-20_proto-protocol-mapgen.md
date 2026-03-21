# Plan - Proto：协议映射文档生成器（路线 A：Go 为真源，生成衍生物）

## Workflow 信息
- Repo：`MyFlowHub-Proto`
- 分支：`chore/protocol-mapgen`
- Worktree：`d:\project\MyFlowHub3\worktrees\pr5-protocol-mapgen\MyFlowHub-Proto`
- Base：`origin/main`
- 参考：
  - `d:\project\MyFlowHub3\target.md`
  - `d:\project\MyFlowHub3\repos.md`
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文）

## 约束（边界）
- 路线 A（已确认）：
  - `MyFlowHub-Proto/protocol/*/types.go` 仍作为 **single source-of-truth**；
  - 本次只新增“生成器 + 强制校验 + 文档”，不生成/不改 Go 协议类型本身。
- wire 不改：SubProto 值、Action 名称、JSON struct 字段与 tag 不允许改动。
- 生成器只用标准库实现（不引入第三方依赖），只做源码解析（`go/parser/go/ast`），不执行协议包代码。
- 文档采用“半自动”（已确认）：仅覆盖 `<!-- BEGIN GENERATED -->` 与 `<!-- END GENERATED -->` 中间区块，保留手写 Notes。
- 强制校验范围（已确认）：
  - 仅在 `MyFlowHub-Proto` 仓库内强制（`go test ./...` 必须保证 `docs/protocol_map.md` 最新）。
  - `d:\project\MyFlowHub3\docs/protocol_map.md` 仅为同步副本，不做门禁。
- 文档同步策略（已确认，C）：
  - 以 `repo/MyFlowHub-Proto/docs/protocol_map.md` 为准；
  - workflow 结束合并后，同步覆盖到 `d:\project\MyFlowHub3\docs/protocol_map.md`（整文件复制，保证一致）。

## 目标
1) 在 `MyFlowHub-Proto` 增加协议映射文档生成器：
   - CLI：`cmd/protocolmapgen`
   - 内部库：`internal/protocolmap`（解析 + 渲染 + 文档区块替换）
2) 在 `MyFlowHub-Proto` 新增并维护 `docs/protocol_map.md`（包含手写说明 + 生成区块）。
3) 增加强制校验：`go test ./...` 失败时提示如何更新文档（减少漂移）。
4) 更新 `README.md`：写清楚真源、生成方式、强制校验范围、同步到 MyFlowHub3 的方式（用户要求）。

## 非目标
- 不引入 CUE / YAML spec，不做“spec -> Go types/constants”生成（路线 B 未来再做）。
- 不生成 TS types / JSON Schema（后续如需要另起 workflow）。
- 不修改任何 `protocol/*/types.go` 的协议内容。

---

## 3.1) 计划拆分（Checklist）

### PMGEN0 - 归档旧 plan
- 目标：保留历史 plan，避免覆盖。
- 已执行（可审计）：`git mv plan.md docs/plan_archive/plan_archive_2026-02-20_proto-protocol-mapgen-prev.md`
- 验收条件：归档文件存在且可阅读。
- 回滚点：撤销该 `git mv`。

### PMGEN1 - 新增 canonical 文档 `docs/protocol_map.md`（含生成区块）
- 目标：
  - 新增 `docs/protocol_map.md`；
  - 保留手写说明区（例如 auth 签名说明、file frame kind 解释）；
  - 预留生成区块标记：`BEGIN/END GENERATED`。
- 涉及文件：
  - `docs/protocol_map.md`
- 验收条件：
  - 文档可独立阅读；
  - 生成区块存在且为空或已填充（后续任务会填充）。
- 回滚点：revert 提交。

### PMGEN2 - 实现协议解析（SubProto/Action/Types/常量）
- 目标：从 `protocol/*` 的 Go 源码中提取：
  - SubProto 编号（`SubProto*`）
  - Action 常量（`Action*`）
  - Exported 类型（`type X struct {...}` 等）
  - 其它常量（如 `Perm*`、`Kind*`、`Op*`、`Visibility*`）
- 涉及模块/文件（预期）：
  - `internal/protocolmap/model.go`
  - `internal/protocolmap/parse.go`
- 验收条件：
  - 解析覆盖全部子协议目录：management/auth/varstore/topicbus/file/flow/exec；
  - 输出稳定（遍历顺序确定：目录名排序 + 文件名排序 + decl 顺序）。
- 测试点：
  - `const` block 与单行 `const` 均可解析；
  - 字符串常量取值为字面量时可提取；
  - 缺失 SubProto 时应返回明确错误。
- 回滚点：revert 提交。

### PMGEN3 - 实现 Markdown 渲染（生成区块内容）
- 目标：
  - 生成稳定的 Markdown（按 SubProto 数值升序）：
    - 总览：`SubProto` 编号 -> 名称
    - 每个子协议：actions 列表、payload types 列表、常量补充（如 ops/kinds/perms）
- 涉及模块/文件（预期）：
  - `internal/protocolmap/render.go`
- 验收条件：
  - 连续运行两次生成结果完全一致（idempotent）。
- 回滚点：revert 提交。

### PMGEN4 - 实现“半自动文档”替换（仅覆盖生成区块）
- 目标：
  - 读取 `docs/protocol_map.md`；
  - 精确替换 `BEGIN/END GENERATED` 中间内容；
  - 保留手写说明区（不被覆盖）。
- 涉及模块/文件（预期）：
  - `internal/protocolmap/doc.go`
- 验收条件：
  - 手写区不变；
  - 生成区块可更新且边界正确。
- 回滚点：revert 提交。

### PMGEN5 - 提供生成器 CLI（write/check）
- 目标：
  - `-write`：写入文档；
  - `-check`：仅校验，不一致时返回非 0；
  - 参数化 `-protocol-root` 与 `-out`（不绑死 workspace 路径）。
- 涉及模块/文件（预期）：
  - `cmd/protocolmapgen/main.go`
- 验收条件：
  - `go run ./cmd/protocolmapgen -write -out docs/protocol_map.md` 可生成；
  - `go run ./cmd/protocolmapgen -check -out docs/protocol_map.md` 在文档最新时返回 0。
- 回滚点：revert 提交。

### PMGEN6 - 强制校验（go test 门禁，Proto 仓内）
- 目标：
  - 增加测试：若 `docs/protocol_map.md` 未更新，则 `go test ./...` 失败；
  - 失败信息提示开发者运行生成器更新文档。
- 涉及文件（预期）：
  - `internal/protocolmap/doc_test.go`
- 验收条件：
  - 人为改动文档生成区块后，`go test` 必须失败并给出明确提示。
- 回滚点：revert 提交。

### PMGEN7 - README 记录“真源/生成/强制/同步”规则（用户要求）
- 目标：在 `README.md` 中写清楚：
  - single source-of-truth 是 `protocol/*/types.go`
  - `docs/protocol_map.md` 为生成+手写混合文档（区块替换）
  - 强制校验仅在 Proto 仓库内
  - 如何同步到 `d:\project\MyFlowHub3\docs/protocol_map.md`
- 涉及文件：
  - `README.md`
- 验收条件：README 信息完整且可执行（给出命令）。
- 回滚点：revert 提交。

### PMGEN8 - 回归验证（命令级）
- 命令（建议统一）：
```powershell
$env:GOTMPDIR='d:\\project\\MyFlowHub3\\.tmp\\gotmp'
New-Item -ItemType Directory -Force -Path $env:GOTMPDIR | Out-Null
$env:GOWORK='off'
go test ./... -count=1 -p 1
go run ./cmd/protocolmapgen -check -out docs/protocol_map.md
```
- 验收条件：命令通过。
- 回滚点：revert 提交。

### PMGEN9 - Code Review（阶段 3.3）
- 按 3.3 清单逐项审查并输出结论（通过/不通过）；不通过则回到对应任务修正。

### PMGEN10 - 归档变更（阶段 4）
- 新增变更文档：`docs/change/2026-02-20_protocol-mapgen.md`
- 必须包含：
  - 背景/目标
  - 生成器命令（write/check）
  - 强制校验策略（仅 Proto）
  - 同步到 MyFlowHub3 的方式
  - 测试与验证方式/结果
  - 回滚方案

### PMGEN11 - 合并与同步（你确认结束 workflow 后执行）
- 在 `repo/MyFlowHub-Proto`：
  - 合并本分支到 `main` 并 push
- 在 `d:\project\MyFlowHub3`：
  - 同步复制：`repo/MyFlowHub-Proto/docs/protocol_map.md` → `docs/protocol_map.md`
  - 将本 workflow 的 `plan.md` 与 `docs/change/*` 按既定规则归档到全局 `docs/` 与 `plan.md` 的 Workflow Archives
  - 清理本 worktree（`git worktree remove + prune`）

