# Desktop 多变量卡片：保留证据

本目录是 [唯一权威 review](../desktop-variable-cards-review.md) 的附件，不另维护验收结论。

## 身份

- [product-manifest.json](product-manifest.json)：产品候选 commit、固定 base、25 个变化路径的 Git blob/mode，删除项为 null。覆盖源码、测试、浏览器 harness 与跟踪的 frontend/dist；未变化依赖由候选 Git tree/base 标识。
- [identity.json](identity.json)：manifest UTF-8/LF 内容 SHA-256 及计划定义 SHA-256；计划只排除进度段，定义范围包含确认约束、Task/write sets、AC 和边界。

## 命令日志

原始日志来自工作区 `artifacts/`，移除终端 ANSI 颜色控制序列、行尾空格及末尾空行后转为 `.txt`；不改命令结果。

| 证据 | 范围 |
| --- | --- |
| [初次前端测试](initial-tests.txt) | 32 文件、574 测试 |
| [初次 TS/Vite 构建](initial-build.txt) | dist 资源名及大 bundle 警告 |
| [初次 Go 测试](initial-go.txt) | desktop package |
| [初次 Wails 构建](initial-wails.txt) | Windows exe，使用已构建 dist |
| [初次模拟浏览器](initial-browser.txt) | Headless Edge 完整交互脚本、零结构写入与查看模式显式写入 |
| [收尾重跑前端测试](closeout-tests.txt) | 相同候选，32 文件、574 测试 |
| [收尾重跑 TS/Vite](closeout-build.txt) | 与初次相同 dist assets |
| [收尾重跑 Go](closeout-go.txt) | `GOWORK=off go test . -count=1` |
| [集成主线前端测试](integrated-tests.txt) | 32 文件、574 测试 |
| [集成主线 TS/Vite](integrated-build.txt) | 产品 dist 与候选一致，仍有既有 bundle 警告 |
| [集成主线 Go](integrated-go.txt) | `GOWORK=off go test . -count=1` |
| [集成主线 Wails](integrated-wails.txt) | `GOWORK=off`、已验证 dist、Windows exe |
| [集成结果元数据](integration-result.json) | merge/产品等价、原有修改保护、清理与未 push 状态；不包含无关文件正文 |

## 浏览器截图

- [编辑模式](editing.png)
- [查看模式](viewing.png)
- [窄窗口](narrow.png)
- [深色概览](overview-dark.png)

这些是内存模拟 API 的合成变量，不包含真实 Profile、设备值或凭据。截图不证明真实 Wails GUI、完整读屏顺序或真实设备写入。可执行交互脚本在 `apps/desktop/frontend/test/verify-variable-cards.cjs`，原始附件独立于待删除 worktree。
