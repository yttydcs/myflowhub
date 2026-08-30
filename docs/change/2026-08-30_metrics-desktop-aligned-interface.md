# 2026-08-30 Metrics 对齐 Desktop 视觉语言的独立界面

## 变更背景 / 目标

Metrics Windows Wails 前端原为单文件命令式 DOM 与深色渐变卡片界面，导航、主题、轮询、错误反馈和选中状态缺少清晰边界。本次在保持 Metrics 独立产品、既有 Wails API 和真实数据语义不变的前提下，按用户确认的高保真稿对齐 Desktop 的工作台视觉语言。

## 具体变更内容

- 将启动入口收敛为可释放 bootstrap，新增可注入 `MetricsAPI` adapter、独立应用状态编排和语义化 view 层。
- 新增“资源状态”“采集策略”“连接与身份”三页，以及 Metrics 自有 light/dark 主题、品牌资产副本和响应式三栏布局。
- 资源页只渲染 backend `Status.samples`，稳定保持选择，并明确展示 empty、fresh、stale、unavailable、error、时间与单位。
- `battery_charging=true` 映射为“充电中”等属于本地化展示；电量、CPU、内存等当前值仍直接来自 sample，不存在生产演示值或 fallback fixture。
- 采集策略继续使用 revision 与 backend validation；身份、连接、停止、轮询、busy/error 均接入既有 Wails API。
- permit 只驻留当前表单，连接成功或应用释放时清空；主题使用 Metrics 自有版本化 local-storage key，导航和敏感输入不持久化。
- 增加 model、API adapter 与 DOM app 测试；更新 tracked production `dist/` 和 Metrics feature 当前事实。

## Docs root

- `D:\project\MyFlowHub3\worktrees\metrics-desktop-ui\docs`，归档后随 canonical `repo\MyFlowHub\docs` 本地合并。
- 本次不新增 remote，不 push、release、publish 或选择备份目标。

## Intake impact

- updated — 新增并索引 `docs/intake/2026-08-30_metrics-desktop-aligned-interface.md`，保留原始请求、确认方向、非目标和验收信号。

## Feature impact

- updated — `docs/features/metrics-node.md` 增加 Windows UI 当前布局、真实 sample、状态、配置、身份、安全和轮询事实。

## Requirements impact

- none — 未新增长期业务能力、权限边界或非 UI 验收口径。

## Specs impact

- none — Wails methods、request/response schema、backend configuration 和 sample contract 均未改变。

## Decision impact

- none — Metrics 与 Desktop 的独立产品边界属于已接受约束；本次仅实现其可逆 presentation 层，不新增架构决策。

## Lessons impact

- none — npm/Wails/PowerShell 与真实 packaged-runtime 分层验证已由既有 `frontend-and-powershell-preflight.md` 覆盖；视觉 fixture 与生产数据边界已写入 feature、change 和测试，不需要新建独立 lesson。

## Related intake

- [Metrics 对齐 Desktop 视觉语言的独立界面](../intake/2026-08-30_metrics-desktop-aligned-interface.md)

## Related features

- [MetricsNode](../features/metrics-node.md)

## Related requirements

- none

## Related specs

- [Build and CI](../specs/build-and-ci.md)

## Related decisions

- `docs/decisions/2026-08-30_desktop-metrics-independent-products.md` 位于主检出用户未提交文档中，本工作流只保留其约束，不暂存或归档该文件。

## Related lessons

- [Frontend 与 PowerShell 预检](../lessons/frontend-and-powershell-preflight.md)
- [新 worktree 缺 Wails bindings](../lessons/frontend-worktree-wailsjs-missing.md)
- [Windows clean checkout、EOL 与 generated drift](../lessons/windows-clean-checkout-eol-and-generated-drift.md)

## 对应 plan.md 任务映射

| Task | 结果 |
| --- | --- |
| MUI-01 | Injectable API、纯展示 model、app bootstrap/dispose 完成。 |
| MUI-02 | Desktop-aligned shell、导航、主题与独立 brand asset 完成。 |
| MUI-03 | 真实 samples、selection、inspector、空态和状态表达完成。 |
| MUI-04 | 策略、身份、启停、轮询、busy/error 接线完成。 |
| MUI-05 | model/API/DOM、timer、输入、安全与无障碍回归测试完成。 |
| MUI-06 | docs、tracked dist、frontend/Go/Wails build 与 production browser visual 通过；真实 WebView2 截图缺口作为已知残余限制接受。 |
| ARC-01 | change/plan 归档、索引、local merge 与 worktree cleanup 由本阶段完成。 |

## 经验 / 教训摘要

- 设计验收 fixture 只能稳定复现界面状态，不能成为生产数据、fallback 或“真实设备已连接”的证据。
- “充电中”“在线”等文字是对 backend boolean 的本地化 presentation mapping；当前值的真实性仍由 `Status.samples` 决定。
- Wails idle 状态下独立 `Configuration` 可能按 contract 报错，初始化应优先使用 `Status.configuration`，但不能吞掉其他真实错误。
- 轮询必须同时具备 in-flight guard、generation 边界和 `dispose()`，否则慢响应可能覆盖新的启停状态。

## 可复用排查线索

- 症状：界面似乎固定显示“充电中”或某个百分比。触发条件：正在查看设计/视觉测试截图。关键词：`visual fixture`、`Status.samples`、`sampleReading`。快速检查：确认生产 adapter 调用 Wails `Status()`，测试 fixture 未出现在 production source/dist fallback 中。
- 症状：布尔值显示为中文而不是 `true/false`。关键词：`booleanReading`、`battery_charging`。快速检查：这是 metric-aware 展示映射；原始值仍保留在 backend sample contract。
- 症状：慢网络下状态倒退或请求重叠。关键词：`poll generation`、`in-flight guard`、`dispose`。快速检查：旧 generation 的 response 不得提交到当前 state。
- 症状：permit 在重连后仍出现在表单。关键词：`permit`、`clear sensitive input`。快速检查：连接成功和 app dispose 两条路径都必须清空内存字段。

## 关键设计决策与权衡

- 复用 Desktop 的视觉语言而非组件或运行时，以少量本地 token/asset 重复换取产品生命周期和 IPC 完全独立。
- 继续使用既有 Wails contract，不为原型中的控制或趋势能力伪造后端接口；因此音量/亮度即时控制和历史图表保持延期。
- 将 app state、view、model 与 adapter 分层，增加少量文件数量，换取可注入测试、轮询释放和敏感输入边界可验证。
- 宽屏 inspector 只承载补充元数据，1024px 紧凑布局可以隐藏它而不丢失核心状态与操作。

## 测试与验证方式 / 结果

- `npm ci`：通过；锁文件可复现安装 91 个 package。
- `npm test`：3 个测试文件、12 个测试通过。
- `npm run build`：TypeScript `--noEmit` 与 Vite production build 通过；生成 `index-DO-g1wDL.js` 与 `index-zJmkIj8S.css`。
- `GOWORK=off go test ./apps/nodes/metrics/... -count=1`：Metrics core、Android mobile、Windows platform 和 Wails package 全部通过。
- Wails CLI v2.11.0 `wails build -clean`：production Windows executable 构建通过；SHA-256 为 `70555454D0AF37BA22F91C52224490620EBE1BB30E6243E00AEBA2D3A0BC15D7`。
- Production-dist browser visual：1440×900 light/dark 与 1024×768 compact 已检查，页面级横向溢出和 runtime error 为 0。
- 真实 WebView2 GUI 截图因 Windows/browser control kernel 初始化失败未取得；用户查看当前界面后显式调用 `$m-archive`，该证据缺口作为已知残余限制接受，不改写为通过的真实设备证据。
- `git diff --check`：通过。

## 潜在影响

- 前端结构与 tracked dist 变化较大，但 backend、generated Wails bindings、Desktop、Android 和协议均未修改。
- 新增 `jsdom` 仅为 dev dependency，会扩大 lockfile，但不进入生产 bundle。
- 未连接或 collector 不可用时会诚实展示 idle/unavailable/error，不再用示例数值填充，因此空态比设计稿更常见。

## 回滚方案

- 回退本次 frontend source、测试、package lock、brand asset 与 regenerated `dist/`，恢复 `54b46e6` 时的 Metrics UI。
- 同步回退 intake、feature、plan/change 归档和分类索引；backend 配置、身份、permit、设备状态和协议数据无需迁移。

## 子Agent执行轨迹

- 无。用户未请求 delegated/parallel Agent work，且 app state、view、styles、tests 与 generated dist 写集高度重叠；实现、验证和归档由主 Agent 顺序完成。

## Closeout status

- 本地 `master` 已 fast-forward 到归档候选 `cde3ec9`；产品提交为 `2b995e0`，首轮归档提交为 `15dcb7d`，并已同步此前 `master` 的准入与 schema 工作。
- 主检出 dirty 状态通过临时 stash `4820bc4` 保护并恢复：所有非重叠 tracked 内容与 stash snapshot 一致，28 个 untracked 文件经 repository filter 校验一致；重叠的 intake 索引同时保留 Metrics 与用户原有记录。
- 临时 stash 已在恢复验证后删除；此前已有的两个 stash 未修改。
- 专用 `metrics-desktop-ui` worktree 已移除，本地 `feat/metrics-desktop-ui` 分支已删除。
- 用户未提交的 Desktop 界面改动、产品独立性/Agent Gateway 文档、设计稿、生成 dist 与 `guide.md` 在恢复时均保留且未暂存；其后续工作区状态继续由对应用户工作流管理。
- 仓库无 remote；本次仅完成本地合并，未 push、release 或 publication。
