# Todo - 带深度的拓扑查询与 Desktop 按展开加载

> 这是执行验收完成时的计划/清单快照，保留当时阶段与授权描述。随后用户调用 `$m-archive`；本地合入、清理及最终状态以[变更归档](../change/2026-09-06_depth-scoped-topology-discovery.md)为准。旧worktree路径仅作历史定位；本地证据已保留到工作区artifacts，文中链接已重新定位。


## Current Status

- Stage：$m-execute；七个已批准 Task ID 的实施与验证完成，未归档或合入。
- Base：master @ ab19d3913fc4be6789c4eb3013840cc3f61f709f。
- Branch：feat/depth-scoped-topology。
- Worktree：D:/project/MyFlowHub3/worktrees/depth-scoped-topology。
- Plan：[plan.md](plan_archive_2026-09-06_depth-scoped-topology.md)。
- Intake：[讨论与新基线](../intake/2026-09-06_depth-scoped-topology-discovery.md)。
- Blocked: no。
- 尚未提交/合并本轮内容；执行证据见下方，各任务按实际完成状态更新。

## Approval Gate

- [x] 用户显式调用 $m-plan，且要求以清理后的新基础规划。
- [x] 已读取参考任务并核对当前 Git 与退役文档。
- [x] 专用分支/工作树存在，已 fast-forward 至 ab19d39。
- [x] 已使用 $m-docs 确认 canonical docs 路由与稳定文档影响。
- [x] 主检出及其他工作树用户修改保留。
- [x] 用户批准 DOC01、PROTO01、TREE01、MGMT01、SDK01、DESK01、QA01。
- [x] 用户选择 $m-execute 执行。

## Approved Execution

- [x] DOC01 — 固化拓扑发现 requirement/spec/ADR 与 Desktop/Hub 范围、搜索语义。
- [x] PROTO01 — 数值 depth、children/subtree 请求与有界响应 schema、生成定义。
- [x] TREE01 — 原子拓扑快照与 membership 版本读取；不改变路由。
- [x] MGMT01 — 现有 topology Resource 的浅/深查询能力、兼容 snapshot/Observe、有界缓存。
- [x] SDK01 — attached Go SDK helper 与 Resource/schema 生成契约；不增加 owning 或重复 binding facade。
- [x] DESK01 — 默认一层、展开加载、部分树缓存、按使用加载 catalog、错误与重试、View 恢复。
- [x] QA01 — 深度/授权/缓存/竞态/键盘与当前 Windows 产品门禁。

## Will Not Execute Now

- [ ] LIVE02 — 实时拓扑推送、全面事件驱动刷新；另立一致性和恢复契约。
- [ ] SEARCH02 — 服务端全范围搜索、分页、超大树传输；本轮仅已加载搜索和有界查询。
- [ ] AUTHZ02 — 通用 payload/member 权限与 effective discovery；本轮复用 capability policy。
- [ ] HOST02 — 所有 NodeHost 自动挂载 provider、Hub 宿主迁移；现有 management provider 之外另立范围。
- [ ] ANDROID-REDESIGN — [Android 重新设计](../requirements/mobile-embedded-redesign.md)，已退役，本轮不恢复。
- [ ] EMBEDDED-REDESIGN — [受限实现重新设计](../requirements/mobile-embedded-redesign.md)，已退役，本轮不恢复。
- [ ] FLOW-REDESIGN — [自动化模型重做](../requirements/flow-redesign.md)，已退役，本轮不恢复。
- [ ] ARC01 — 实现通过后显式 $m-archive：提交、归档、合入、工作树清理。
- [ ] PUB01 — 推送、签名、发布、部署；独立授权。

## Dependency Order

DOC01 → PROTO01/TREE01 → MGMT01 → SDK01 → DESK01 → QA01。
PROTO01 与 TREE01 使用独立子 Agent 写集；主 Agent 负责 management、整合与最终验收，生成文件保持单一写入者。
每个 Task ID 只属于一个执行范围；本文件不继承上一轮已完成的批准。

## Acceptance Reminders

- [x] depth 默认 1；wire 显式 0 保留；负数/非整数/缺失/null 拒绝。
- [x] children capability 不能接受 depth 0/2；只有获对应授权才可递归。
- [x] 不把 response root 误认为网络根，不自动向上查询。
- [x] 缓存未变无需重新组树；announce/withdraw/reparent/name/restart 能正确失效。
- [x] 未加载、空、Forbidden、unsupported、不可达分别表达。
- [x] 展开与 catalog 请求去重/限并发；旧 Profile/连接响应不提交。
- [x] 保存 View 的资源独立加载，不要求树先展开；布局与业务 draft 保留。
- [x] 搜索明确覆盖已加载节点；不暗中遍历。
- [x] legacy snapshot 的当前行为保留；不恢复已删除 SDK owning 方法与平台构建目标。
- [x] 生成契约、Go/race、Desktop、Metrics、Clipboard 当前验证完成并记录实际证据。

## Evidence

- DOC01 新 requirement/spec/ADR 与分类索引已更新；相对链接及 diff 空白检查通过。
- PROTO01 协议严格解码、线性图校验、生成约束测试通过；TREE01 快照、零分配未变路径及 race 通过。
- MGMT01 focused + race 通过；4000 节点测试树 shallow 2 节点/260 字节，legacy 4000 节点/254910 字节；缓存读取基准不代表网络端到端延迟。
- SDK01 attached helper、owner/depth/schema 错误、权限和生命周期测试通过；go generate ./sdk/bindings 后 bindings/internal/protocoltest 通过。
- QA01 已完成全量 go test ./... -count=1、go vet ./...、go build ./...；Memory/TCP 六层拓扑与权限集成及 race 通过。汇总、真实日志和产物入口见 [QA 报告](../../../../artifacts/2026-09-06_depth-scoped-topology/evidence/REPORT.md)。
- 复核补齐旧订阅有序发布与 Overflow 终止，重入改变树的双 observer、已有订阅超限→恢复测试及七个相关 Go 包的 race 通过；最终全量 Go/test/vet/build 再次通过。
- Desktop 22 个 Vitest 文件、224 项测试和最终 Windows Wails build 通过；Profile/View 异步、迟到保存、排队失效清理与名称保持回归通过。真实 Wails + 六层 TCP 验证默认一层、鼠标/键盘、Forbidden、Unsupported、完整子树、已加载搜索和树外 owner 的 View 恢复；最终页面复测紧凑起点入口与名称保持通过。
- Metrics test/build、Clipboard analyze/widget/Web/Windows build 均通过，最终 core 修改后针对性产品重建通过。12 个生成文件二次生成 SHA256 一致，最终 Wails build 后哈希仍一致。
- 9 项无语义构建换行改动已定点恢复；实际 Desktop bundle 保留。隔离测试进程与页面已关闭，fixture 正常结束。没有待完成的批准门禁；未提交、归档、合入、推送或清理工作树。
- 并行路径：Planck(PROTO01)、Faraday(TREE01及MGMT只读复核)、Nash(SDK01及隔离GUI fixture)、Beauvoir(DESK01)、Pauli(QA01网络集成)、Laplace(QA01 Metrics/Clipboard)。主 Agent 保留生成、整合、最终验收职责。
