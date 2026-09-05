# 2026-09-06 按深度查询拓扑与 Desktop 按需发现

## 变更背景 / 目标

网络模型只包含节点、网络、资源，节点不等同于设备。Desktop 原先读取父节点完整拓扑并批量获取全部 catalog；本轮让导航按已知起点与展开深度加载，复用父节点维护的状态，同时保留目录和业务权限的独立性。

基线 `ab19d3913fc4be6789c4eb3013840cc3f61f709f`，实现提交 `a3757f8de753dcbb67a615ed3c7ae58b3a4ac2f8`。用户批准 DOC01、PROTO01、TREE01、MGMT01、SDK01、DESK01、QA01，全部验收通过；随后显式调用 `$m-archive`，授权本地归档、提交、合入和本轮工作树清理。

## 具体变更内容与任务映射

| Task | 实现与结果 |
| --- | --- |
| DOC01 | 固化节点/网络/资源边界、发现需求、协议、独立能力决策及 Desktop/Hub 当前行为 |
| PROTO01 | children/subtree 输入、统一 query 输出、数值 depth、显式0、严格解码与有界线性图校验 |
| TREE01 | 单锁原子拓扑快照和 membership/epoch 版本；未变不复制，路由算法不变 |
| MGMT01 | 同一个 topology Resource 增加独立浅/深能力；不可变索引、depth0/1预生成和有界其他深度缓存；旧观察有序发布，超限通过现有 wire Error 终止 |
| SDK01 | attached Go `QueryTopology(ctx, depth)`，保留 legacy helper；生成 Resource/schema 合同，未新增 owning 或重复 Wails facade |
| DESK01 | 默认直接父节点一层，展开加载、已加载搜索、独立 catalog、4请求并发、30秒软有效期、4096导航项和128 owner LRU；Profile/连接/View及保存草稿竞态保护 |
| QA01 | Memory/TCP六层拓扑、权限与超限恢复、相关race、生成一致性、真实Wails开发桥联调、当前Windows产品门禁 |
| ARC01 | 完整计划/checklist归档、变更与经验文档、索引、证据保留、本地合入和清理 |

完整拆解及延期任务见[计划快照](../plan/plan_archive_2026-09-06_depth-scoped-topology.md)和[验收清单](../plan/todo_archive_2026-09-06_depth-scoped-topology.md)。

## Docs root 与稳定文档影响

Docs root：canonical monorepo `D:/project/MyFlowHub3/repo/MyFlowHub/docs`。实现期间在 sibling worktree 的 `docs/` 中修改；无独立文档仓库、远端配置或发布操作。

| 影响 | 状态 | Related docs |
| --- | --- | --- |
| Intake impact | updated | [讨论与批准记录](../intake/2026-09-06_depth-scoped-topology-discovery.md) |
| Feature impact | updated | [Desktop](../features/desktop.md)、[Hub](../features/hub.md) |
| Requirements impact | updated | [拓扑发现](../requirements/topology-discovery.md)、[Desktop工作区](../requirements/desktop-resource-workspace.md) |
| Specs impact | updated | [查询与缓存合同](../specs/topology-discovery.md)、[Desktop v2](../specs/desktop-resource-workspace-v2.md)、[v3](../specs/desktop-resource-workspace-v3.md)、[protocol map](../specs/protocol_map.md)、[资源平台](../specs/resource-platform-v2.md) |
| Decision impact | updated | [浅层与递归能力分离](../decisions/2026-09-06_depth-scoped-topology-capabilities.md) |
| Lessons impact | updated | [部分发现缓存与异步归属](../lessons/partial-discovery-and-async-ownership.md)、[可观察副作用](../lessons/observable-side-effects-and-generated-contracts.md) |

新增文档已进入对应分类索引。顶层目录及阅读路径不变，未改写无关历史文档或生成区。

## 关键设计决策与权衡

- `(N, system/topology)` 的 owner N 就是查询起点。depth1为直接孩子，depth0为该起点不限后代深度；查询根不等于网络根，不自动查祖先。
- children只接受depth1；subtree接受0..4096。复用现有Subject/Resource/Capability策略；树、目录、业务操作各自鉴权，缓存命中仍重新鉴权。
- 旧read/subscribe仍暴露完整快照，已有宽权限不自动变窄。未迁移真实Grants/Bindings，没有用前端裁剪冒充浅层权限。
- 服务端读已维护关系，周期刷新派生不可变快照；请求不逐节点采集。不承诺全网实时强一致或任意大小查询。
- 请求显式失败时保留可恢复导航状态；未加载不等于空，不支持不触发全量降级。View的owner可在导航树外。
- 保留已清理的Flow/mobile/embedded/owning SDK基线；实时推送、分页、effective discovery、全NodeHost provider装配另立任务。

## 测试与验证方式 / 结果

以下均为本轮执行结果，Go命令使用 `GOWORK=off`，PowerShell使用无profile模式。

| 门禁 | 结果 |
| --- | --- |
| 全量 `go test ./... -count=1`、`go vet ./...`、`go build ./...` | Passed |
| tree及resource/subscription/node/command/management/SDK/integration相关race | Passed；覆盖快照隔离、旧订阅Overflow终止/恢复、重入版本顺序与清理 |
| Memory/TCP六层集成 | Passed；depth0/1/2、children拒绝递归与旧广权限、伪造参数、缓存后撤权、catalog独立和既有grant不迁移 |
| Desktop `npm test`、`npm run build` | Passed；22文件224测试，TypeScript与Vite完成 |
| Desktop Windows Wails production build | Passed；最终EXE SHA256 `A328EB54D1E33DB101BE577D6749787C5055DB7A4FB2F7F92C9675B0ADC6A984` |
| Metrics canonical test/build | Passed；3个Go包、12项前端测试、Wails EXE；最终订阅改动后针对性重建通过 |
| Clipboard canonical test/build | Passed；3个Go包、Flutter analyze/widget/Web/Windows及Go bridge；最终bridge重建通过 |
| 生成一致性 | Passed；12文件二次生成及最终Wails后的SHA256一致，API架构门禁通过；实现提交后再运行canonical `check/generated`，与已提交基线零差异，bindings/protocoltest通过 |
| GUI smoke | Passed；实际Wails开发HTTP/WebSocket桥、最终源码和生产前端资产、隔离TCP节点；鼠标/键盘、拒绝/不支持、完整子树、已加载搜索、树外owner View恢复、紧凑入口及名称保持 |

GUI不是对生产WebView2窗口的原生自动化；生产EXE另经canonical构建验证。慢响应、跨Profile和迟到保存以自动化回归为证据。Desktop前端原始输出保存在Agent工具记录，没有独立原始日志；Wails原生进度部分绕过脚本重定向，不能将短日志误解为完整输出。

同一4000节点链式树，legacy为4000节点/254910字节，depth1为2节点/260字节。缓存微基准不等于网络延迟，未设置脆弱的耗时断言。

最终暂存检查发现新Vite bundle中的5处模板字符串空白行含空格；这是原样生成的CSS字符串，不手工裁剪。源码/文档的diff检查通过，bundle与已验证生产构建SHA256一致。Vite主包大小和Clipboard Web字体提示未阻断构建。

## 证据与产物保留

本地证据和产物移出待清理工作树，存放在工作区 `artifacts/2026-09-06_depth-scoped-topology/`，127项文件逐项SHA256核对；[保留清单](../../../../artifacts/2026-09-06_depth-scoped-topology/retained-manifest.json)。只保留日志、报告、manifest及已验收产品产物，排除测试credential、依赖缓存和临时状态。此位置不是远端备份或发布目的地。

- [证据入口](../../../../artifacts/2026-09-06_depth-scoped-topology/README.md)
- [最终全量Go](../../../../artifacts/2026-09-06_depth-scoped-topology/evidence/go-test-all-final.log)
- [七包race](../../../../artifacts/2026-09-06_depth-scoped-topology/evidence/go-race-final.log)
- [Windows产品报告](../../../../artifacts/2026-09-06_depth-scoped-topology/evidence/windows-20260906-022257/REPORT.md)
- [真实页面记录](../../../../artifacts/2026-09-06_depth-scoped-topology/evidence/gui-smoke.md)
- [提交后生成门禁](../../../../artifacts/2026-09-06_depth-scoped-topology/generated-check-after-commit.log)

历史报告中的原worktree绝对路径是执行时信息，保留位置以retained-manifest为准；没有为保持旧路径而留下junction。

## 经验 / 教训摘要与排查线索

- 节点名称消失、被移除分支复活：检查响应覆盖的完整孩子集合、边界关系保留、owner/instance/revision和分支请求失效。
- 切换Profile后访问旧View owner、保存后新草稿消失：检查View加载归属、当前View ID和编辑generation，不能只守护topology请求。
- 持续loading但没有网络请求：检查调度前被跳过的任务是否仍经过finally清除inFlight。
- 订阅版本倒退或全量超限仍显示旧值：检查回调重入与发布顺序、Failure从Observable到delivery再到现有wire Error的完整终止路径。

可复用的根因与快速检查已进入上述lessons；稳定行为仍以spec/feature为准。

## 潜在影响与回滚方案

新Desktop浅层导航需要相应owner的children权限；显式深层查询需要subtree。无该provider的节点显示不支持；旧客户端及旧read/subscribe继续可用。既有all/superadmin按当前规则覆盖新增能力。

无持久数据、View或credential迁移，无真实授权变更。需要回滚时优先revert实现提交 `a3757f8`，按Desktop→SDK→management→tree/protocol检查依赖并重新构建；保留用户草稿及平台退役基线。归档和证据可继续留存，不能通过reset覆盖用户未提交内容。

## 子Agent执行轨迹

执行与QA阶段：Planck负责PROTO01（01a072c5-4b6e-7291-9d73-059343d11f6c）；Faraday负责TREE01、management复核及订阅失败直接后果（01a072c5-4c39-70a1-a4b6-41714663b42b）；Nash负责SDK01及隔离GUI fixture（01a072c9-78b3-75f0-9a8c-049f9207d0b7）；Beauvoir负责DESK01（01a072c5-bbad-7513-bb10-796477e3dea0）；Pauli负责网络集成与前端只读复核（01a072ca-819c-70c3-b8cc-812b7f754482）；Laplace负责Metrics/Clipboard门禁（01a072cd-c55d-7163-b18f-0425585762d0）。主Agent负责DOC01、MGMT01、生成、整合、最终验收。

归档阶段由主Agent完成，未委派子Agent。

## 本地收口状态

归档文档完成；本地master合入与工作树清理待执行。主检出有8项tracked改动及独立未跟踪草稿，本轮仅6份文档重叠，先准备三方合并并校验，再保留为未暂存改动。其他工作树不属于本轮清理范围。未推送、发布、部署或配置远端。
