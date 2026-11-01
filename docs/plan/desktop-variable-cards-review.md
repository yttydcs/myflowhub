# Desktop 多变量卡片：权威审查与验收

## 候选与定义身份

- 评审/实施仓库：`D:/project/MyFlowHub3/worktrees/desktop-variable-cards`，分支 `feat/desktop-variable-cards`；控制面为 `D:/project/MyFlowHub3/repo/MyFlowHub` 的 `master`。
- 固定比较 base：`cf1436115c10a6d3c6ecca3334c1a0e009aa7247`（两边初始 HEAD 与 merge-base 一致）。
- 产品候选 HEAD：`4b3a216bdd708b710401e9c3464c8ad6c3bfc815`。其 25 个产品路径已经提交；归档文档另行提交，不改变产品内容。
- [产品 manifest](desktop-variable-cards-evidence/product-manifest.json) UTF-8/LF 内容 SHA-256：`acfcc5ef30c22e974313d0d285162fb7271b9b46daf0383c3e5f9755823df43d`，包含新增/修改/删除、路径、模式与 Git blob；没有 symlink/mode 变更。初次验证后的源码未变，收尾重跑测试/构建并确认 dist assets 相同。
- [计划](desktop-variable-cards.md)定义 SHA-256：`1a48f81cc9a2ea90dfda434a117bbbe3f007b47dc5b9637864bc364affe66218`。取 UTF-8/LF、从文件开始到“交付状态”之前，覆盖确认交互、Task/AC/写入范围及边界；只排除进度/报告元数据，不把本报告纳入自身哈希。计算规则保存在 [identity](desktop-variable-cards-evidence/identity.json)。
- 约束来源：[用户请求及合并授权](../intake/2026-10-31_desktop-variable-cards.md)、[工作区需求](../requirements/desktop-resource-workspace.md) 20—23、[v3 规格](../specs/desktop-resource-workspace-v3.md)；未把附件中的指令当成用户请求。

## 唯一 AC / Task / 证据映射

| Task / AC | 实际检查面 | 结果与证据 |
| --- | --- | --- |
| T01 / AC01 | Go round-trip/legacy/stable-single/非法绑定/重复/上限；TS View 校验 | passed；`views_variable_cards_test.go`、`workspace-variables.test.ts`、[收尾 Go](desktop-variable-cards-evidence/closeout-go.txt)、[前端](desktop-variable-cards-evidence/closeout-tests.txt) |
| T02 / AC02 | 事务性 transfer、来源移除前拒绝重复、空容器/单行转换、ID 稳定 | passed；`workspace-variables.test.ts`；[模拟浏览器](desktop-variable-cards-evidence/initial-browser.txt)覆盖合并/插入/指针与键盘排序/移动/拆出 |
| T03 / AC03 | 标题/行别名覆盖与清空、逐行状态；稳定 portal/同一无效输入 DOM；结构 blur 零写入；菜单等价入口 | passed；`VariableCards.test.tsx`、`workspace-variables.test.ts` 与浏览器源码中的 DOM 身份/write 断言；[编辑截图](desktop-variable-cards-evidence/editing.png) |
| T04 / AC04 | 查看锁定结构而值控件可用；新建/重开/保存/取消；外部有效/无效/取消 drop；保存失败和在途新编辑/Profile 保护 | passed；`App.test.tsx`、`App.discovery.test.tsx`、Workspace 测试及浏览器；[查看](desktop-variable-cards-evidence/viewing.png)/[窄窗口](desktop-variable-cards-evidence/narrow.png)/[深色](desktop-variable-cards-evidence/overview-dark.png) |
| T05 / AC05 | 全量前端、Go、TS/Vite、Wails Windows 包；文档链与安全边界 | passed（表中列明的测试范围）；见下方命令、[证据索引](desktop-variable-cards-evidence/README.md)。归档与集成收尾状态在[change](../change/2026-11-01_desktop-variable-cards.md)，不把未执行的设备/GUI 检查标成通过 |

## Requirements review — passed

2026-11-01 按原始确认约束逐项核对实际候选，而非仅依赖测试绿灯：

- 卡片类型与布局正交，标题/行别名不改真实资源；对应 `types.ts`、`VariableCards.tsx`、`WidgetSettings.tsx` 与 `updateVariable`。
- 两级拖动的 source/target 分离；catalog 添加与内部移动区分；单卡合并、重复拒绝、保留空列表/显式转换由 `transferVariable`、`asVariableList/asSingleVariableCard` 保证。
- `App.dragEnd` 重新计算实际落点，不用旧 hover 兜底；仅成功变更进入编辑。模式不持久化；异步 save 以 Profile/edit generation 防止覆盖更新草稿；取消到编辑 baseline/最近确认保存结构。
- `VariableCardsProvider` 按引用 ID 持有 portal/editor，`HostSlot` 只移动 DOM 宿主；结构交互 context 拦截结构 blur 的提交意图。无值/草稿写入 View，结构操作不调用 write；查看模式显式值操作仍经原 controller。
- 无发现本次原始确认范围内的阻塞遗漏或未授权功能扩张。实现侧 64/512/50 有界保护记录为技术边界，不冒充用户新增业务要求。

## Standards review — passed

这是独立于需求轴的主代理轻量审查，不是声称有第二个外部审查代理。

- 检查固定 base 到实际工作树、staged/unstaged/untracked 的完整归属；新建源码/测试/浏览器 harness 已检查。只替换两个 dist assets 及入口；无依赖/协议/go.mod/凭据变更，无额外二进制/符号链接纳入提交。
- 不可变纯领域操作独立于渲染；重组先校验后移除来源，失败不丢数据。引用和布局 ID 的不变量在前端与 Go 持久化边界复核，保留原原子存储保护。
- 复用原变量 controller、schema、授权与订阅路径，不复制 wire/runtime 或绕过 owner 裁决；portal 宿主清理/ResizeObserver 生命周期有对应释放路径。
- 沿用 React/dnd-kit/CSS，无新增库；新增结构操作有菜单/键盘路径和 live feedback。视觉顺序不被误报为完整读屏验收。
- 遵守 `guide.md` 中文提交、canonical 路径及 `GOWORK=off` 约定；未操作主线 Flow 文档。`git diff --check` 通过。
- 无发现需阻止合并的本轮硬性规范违反。既有大 bundle 警告保留为非阻塞性能风险，不扩展到无关重构。

## 验证命令与新鲜度

- 2026-11-01 收尾重跑：前端 `npm test -- --maxWorkers=4`，32 文件 / 574 测试通过；`npm run build`，TypeScript/Vite 通过，仍输出 `index-B-SFZkbr.js` / `index-D5Kth4yX.css`。
- 收尾 Go：`GOWORK=off go test . -count=1` 通过。
- 复用第一版当前内容的 Wails：`wails build -s -m -nosyncgomod -platform windows/amd64 -o MyFlowHub-Desktop-variable-cards.exe` 成功；[原日志](desktop-variable-cards-evidence/initial-wails.txt)。首次未跳过前端的构建遇 Vite 占用 lightningcss 的 npm ci EPERM；后续成功构建已确认，不把失败尝试隐藏或算成通过。
- 复用同一源码/脚本与相同 dist 的 Headless Edge 内存模拟 API 验证；[日志](desktop-variable-cards-evidence/initial-browser.txt)与跟踪的 `test/verify-variable-cards.cjs` 对应。覆盖结构操作零写入、同一无效 input、查看模式一次显式模拟 write、无页面错误。不是重新运行了一次，也不是设备验证。
- 合并时仍须固定 base 未前进、没有产品冲突，并逐条比较集成 tree 的 manifest Git blob/删除。产品内容/依赖若变化，对应结论失效，不能只凭分支测试复用。

## 未验证 / 未豁免事项

- 没有真实用户 Profile/设备读写，没有真实 Wails GUI 操作或完整读屏遍历；用户要求合并不等于把这些项目标成通过或授予真实设备写入权限。本轮确认功能的自动化/模拟范围已通过，上述作为后续真实环境验收边界保留。
- Vite 单 chunk 超 500 kB 是既有非阻塞 warning。
- 旧二进制不能前向读取新字段；需升级前独立备份 `views.json`，回滚恢复备份。本次没有迁移或触碰真实 Profile。

## 集成绑定

2026-11-01 控制面 `master` 以 `2944af806e78730198453e5b71bd475844a90653` 完成本地 no-ff 合并（归档候选 `bbe7855`）。主线原 HEAD 未前进，产品合并没有冲突；合并整个 tracked tree 与候选完全相同，25 个 manifest 路径的 blob/删除逐一相同，实际 apps/go.mod/go.sum 无未提交变化，故原两轴结论继续有效。

在集成后的 canonical 主线另行重跑：前端 32 文件/574 测试、TS/Vite、`GOWORK=off go test . -count=1`、`GOWORK=off wails build -s -m -nosyncgomod -platform windows/amd64 -o MyFlowHub-Desktop-variable-cards.exe` 全部通过；[主线测试](desktop-variable-cards-evidence/integrated-tests.txt)、[构建](desktop-variable-cards-evidence/integrated-build.txt)、[Go](desktop-variable-cards-evidence/integrated-go.txt)、[Wails](desktop-variable-cards-evidence/integrated-wails.txt)。未重复浏览器或真实设备测试。

主线原有 255 个修改/未跟踪文件全部保留：253 个逐字节相同；两个重叠索引去掉本轮各一条链接后 UTF-8/LF 内容相同。仅窄范围保护这两个索引；恢复时 intake 索引发生一处追加行冲突，保留 Flow 与 Desktop 双方条目并核对原文，未改产品内容。恢复后取消索引暂存，本次临时 stash 已移除，原有三个 stash 保留。安全检查与证据保留完成后，已删除 `feat/desktop-variable-cards` 及其 worktree，其他 worktree 不动。

[集成结果元数据](desktop-variable-cards-evidence/integration-result.json)与[归档](../change/2026-11-01_desktop-variable-cards.md)记录最终状态；后续仅追加本次收尾证据的文档提交，不改变产品/计划定义身份。
