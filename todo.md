# Todo - Desktop Explorer 可折叠分区与 Resource path tree

## Will Execute After Approval

- [x] DOC01 — 稳定文档与控制面收敛
- [x] TREE01 — Resource path-tree trie/index、搜索祖先与扁平可见行
- [x] UI01 — Resource WAI-ARIA tree、hybrid item、预览/拖拽/添加与 section disclosure
- [x] PREF01 — collapse/resource expansion per-Profile preference 与 split layout
- [x] QA01 — frontend/Go/build/Wails/真实 GUI 验证与 production dist

## Will Not Execute In The Next Phase

- [ ] LAZY01 — 服务端分页、lazy loading 与真正 DOM windowing；现有 API 不支持，独立阶段
- [ ] BRAND01 — 品牌图标与平台资产；独立任务拥有
- [ ] ARC01 — change/plan archive 已生成；本地 merge、恢复验证与 cleanup 进行中
- [ ] PUB01 — push/release/publication；未授权且无 remote

## Acceptance Checklist

- [x] Node/Resource heading 可通过 pointer、Enter、Space 收起/展开并暴露正确 ARIA 状态
- [x] 最多一个 pane 收起；另一 pane 满高；separator 隐藏且重新展开恢复比例
- [x] Resource path 任意深度、纯 namespace、leaf 和 Resource/parent hybrid 正确
- [x] Resource tree roving focus、方向键、Home/End、Enter/Space 与选择状态正确
- [x] 搜索保留祖先并临时展开，清空后恢复持久 expanded paths
- [x] Inspector preview、type icon、拖拽和添加工作区无回归
- [x] collapse/expanded paths 按 Profile 保存，旧/损坏 preference 行为明确
- [x] 独立滚动、固定 Profile footer、浅/深色和窄窗口通过
- [x] 10,000 Resource synthetic test 保持线性有界
- [x] `npm test`、`npm run build`、`GOWORK=off go test ./...`、Wails production build 通过
- [x] packaged Desktop 连接真实 Hub 后的手动/自动 GUI smoke 通过

## Gate

- Technical blockers: none
- Blocked: no
- Approved Task IDs: `DOC01, TREE01, UI01, PREF01, QA01`
- Planned Task IDs: `DOC01, TREE01, UI01, PREF01, QA01`
- Active phase: `$m-archive` closeout in progress
- Implementation started: yes
- Do not dispatch implementation subagents
