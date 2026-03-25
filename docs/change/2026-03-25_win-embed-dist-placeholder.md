# 2026-03-25 Win：修复 Wails `frontend/dist` embed 占位失效

## 变更背景 / 目标
- 背景：
  - `MyFlowHub-Win/main.go` 使用 `//go:embed all:frontend/dist` 嵌入前端静态资源目录。
  - Wails CLI 在生成 bindings 前会执行 `go mod tidy` / 包加载；如果 `frontend/dist` 没有可嵌入文件，会直接失败。
  - 当前主线只依赖隐藏文件 `frontend/dist/.keep` 作为占位，而该文件一旦缺失，就会出现：
    - `pattern all:frontend/dist: cannot embed directory frontend/dist: contains no embeddable files`
- 目标：
  - 让 `frontend/dist` 在未构建前端时仍始终具备稳定的可嵌入占位文件。
  - 让 `go mod tidy`、`GOWORK=off go test ./...` 和 `GOWORK=off wails build -nopackage` 全部恢复通过。

## 具体变更内容
- `.gitignore`
  - 将 `frontend/dist` 的白名单从 `frontend/dist/.keep` 改为 `frontend/dist/placeholder.txt`。
- `frontend/dist/placeholder.txt`
  - 新增普通文本占位文件，作为 `go:embed` 的稳定匹配目标。
- `frontend/package.json`
  - `build` 脚本在 `vite build` 后改为自动重建 `dist/placeholder.txt`，避免 `dist` 被清空后再次触发 embed 错误。
- 删除
  - 移除旧的 `frontend/dist/.keep` 占位策略。

## Requirements impact
`none`

## Specs impact
`none`

## Lessons impact
`updated`

## Related requirements
- `none`

## Related specs
- `none`

## Related lessons
- `docs/lessons/wails-embed-dist-placeholder.md`

## 对应 plan.md 任务映射
- `WINEMBED1` - 收敛 `frontend/dist` 占位文件策略
- `WINEMBED2` - 更新前端 build 脚本和忽略规则
- `WINEMBED3` - 完成 `go mod tidy` / `go test` / `wails build` 验证
- `DOC1` - 归档本次修复并更新索引

## 经验 / 教训摘要
- 对 Wails 项目而言，`frontend/dist` 不只是前端产物目录，还是 Go 侧 `go:embed` 的编译前置条件。
- 只依赖隐藏 `.keep` 文件容易在手工清理、脚本切换或目录同步时被误删，恢复成本高。
- 占位文件策略必须同时在三处保持一致：
  - `.gitignore`
  - `frontend/dist` 版本库初始状态
  - `frontend/package.json` build 后补回逻辑

## 可复用排查线索
- 症状
  - `pattern all:frontend/dist: cannot embed directory frontend/dist: contains no embeddable files`
  - Wails CLI 在 `Generating bindings` 阶段失败
  - 日志前面显示 `Executing: go mod tidy`
- 触发条件
  - `frontend/dist` 为空目录
  - 旧的 `.keep` 占位文件被删掉
  - 尚未执行前端构建，但先执行了 `go mod tidy`、`go test` 或 `wails build`
- 关键词
  - `go:embed all:frontend/dist`
  - `frontend/dist`
  - `go mod tidy`
  - `Generating bindings`
  - `contains no embeddable files`
- 快速检查
  - 查看 `frontend/dist` 是否至少有一个普通文件
  - 查看 `.gitignore` 是否仍对白名单指向有效占位文件
  - 查看 `frontend/package.json` 的 `build` 脚本是否会在构建后补回占位文件

## 关键设计决策与权衡
- 保持 `main.go` 的 `go:embed` 路径不变：
  - 避免扩大到 Wails 资源加载逻辑。
- 用普通文本文件替代隐藏 `.keep`：
  - 目标是降低被误删或被工具忽略的概率。
- 继续让前端 build 后重建占位文件：
  - `vite build` 会清理 `dist`，不补回占位文件的话问题还会复发。

## 测试与验证方式 / 结果
- `go mod tidy`
  - 结果：通过
- `GOWORK=off go test ./... -count=1`
  - 结果：通过
- `GOWORK=off wails build -nopackage`
  - 结果：通过
  - 说明：Wails 绑定阶段不再出现 `frontend/dist` embed 失败
- 验证产物
  - `build/bin/myflowhub-win.exe` 已成功生成

## 潜在影响与回滚方案
- 潜在影响
  - 仓库中 `frontend/dist` 的占位文件名从 `.keep` 改为 `placeholder.txt`。
  - 若外部脚本仍硬编码 `.keep`，需要一并调整。
- 回滚方案
  - 回退 `.gitignore`、`frontend/package.json` 和 `frontend/dist` 占位文件改动。
  - 回退后若仍需构建，需手工保证 `frontend/dist` 至少存在一个可嵌入文件。

## 子Agent执行轨迹
- 本轮未使用子 Agent。
