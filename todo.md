# Todo - Desktop 多面板嵌套停靠

## Executed

- [x] DOC01 — View v3 稳定文档、v2 supersession 与 n 元停靠 ADR
- [x] VIEW01 — Go View v3 持久化、v1/v2 迁移、递归校验与回滚快照
- [x] LAYOUT01 — TypeScript 分割树领域模型、停靠/移动/删除/调整与测试
- [x] RENDER01 — 递归满区面板、嵌套 separator、RAF 平滑预览与键盘可访问性
- [x] DOCK01 — 无专用按钮的面板/分隔线/工作区外缘拖拽和真实结果预览
- [x] QA01 — 自动化测试、TypeScript/Vite、Go、生成漂移与生产 `dist`

## Completed Closeout

- [x] TEST01 — packaged Wails、v2/v3 迁移、6–8 面板、窄窗口滚动、主题、Forbidden、保存/重启与平滑度验收通过
- [x] ARC01 — `$m-docs` 影响复核、change/plan 归档、中文提交、安全本地合并与清理完成

## Will Not Execute

- [ ] STACK01 — 中心标签栈；首期中心为无效区
- [ ] FLOAT01 — 浮动/跨窗口停靠与布局预设
- [ ] LIB01 — 完整第三方 Docking 框架
- [ ] BRAND01 — 图标/品牌资产，由独立任务负责
- [ ] PUB01 — push/release/publication，未授权且无 remote

## Acceptance Checklist

- [x] 拖拽领域操作生成 `A | C | B`
- [x] 拖拽领域操作生成 `A | (B / C)`
- [x] 拖拽领域操作生成 `(A | B | C) / D`
- [x] 任意嵌套相邻面板可 RAF 预览、键盘调整且只在释放时提交
- [x] 无“左右 / 上下 / 交换”按钮，中心不生成隐式标签
- [x] v1/v2 保留全部 widgets 迁移到 v3；损坏输入不覆盖原文件
- [x] 保存、重开和重启恢复拓扑、顺序与权重
- [x] themes、窄窗口、detached/forbidden 与 Profile/View 行为无回归

## Gate

- Blocked: no
- Approved Task IDs: `DOC01, VIEW01, LAYOUT01, RENDER01, DOCK01, QA01`
- Active phase: `$m-archive` closeout
- Terminal reason: tests passed; archive authorized and completed locally
- Do not dispatch implementation subagents
