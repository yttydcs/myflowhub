# 变更归档：MyFlowHub-Win 移除 Fyne + Wails Windows 构建验收

日期：2026-02-09

## 变更背景 / 目标
- 目标 1：彻底移除 Fyne 旧 UI（代码路径与依赖），统一以 Wails（Go 后端 + Vue 前端）作为唯一桌面端实现，降低维护成本。
- 目标 2：补齐 Windows 构建与冒烟链路，使开发阶段的交付可复现、可审计。

## 具体变更内容（新增 / 修改 / 删除）
### 新增
- Wails 应用入口与绑定：
  - `main.go`、`wails.json`、`app*.go`
- 前端（Vue3 + Vite + Tailwind）：
  - `frontend/**`（不包含 `node_modules/` 与构建产物；保留 `package-lock.json`）
- Wails 后端服务与存储：
  - `internal/services/**`
  - `internal/storage/**`（包含旧配置迁移能力，但不依赖 Fyne 库）
- 文档：
  - `README.md`（build/run/smoke）

### 修改
- `go.mod` / `go.sum`：移除 Fyne 相关依赖并 `go mod tidy` 收敛依赖集合。
- `.gitignore`：忽略 `frontend/node_modules`、`frontend/dist`、`build/` 等构建产物；保留 `frontend/dist/.keep` 用于满足 `go:embed` 编译期路径匹配。
- `.gitignore`：忽略 `frontend/package.json.md5`（Wails 前端依赖安装缓存文件，构建/安装时会更新，不应纳入版本控制）。
- `frontend/package.json`：构建脚本在 `vite build` 后补写 `dist/.keep`，避免前端构建清理目录导致 `.keep` 被删除而出现脏工作区。

### 删除
- 旧 Fyne UI 与入口：
  - `internal/ui/**`
  - `internal/app/**`
  - `cmd/main.go`

## 对应 plan.md 任务映射
- W1：迁移主 worktree WIP 到本 worktree（并排除构建产物）
- W2：移除 Fyne 代码路径（删除旧 UI 与入口）
- W3：移除 Fyne 依赖并收敛 Go 模块
- W4：Windows 构建验证与冒烟步骤文档

## 关键设计决策与权衡（性能 / 扩展性）
- 采用分层：前端仅做 UI/交互；协议与逻辑尽量复用 `myflowhub-core` + `myflowhub-server/protocol/*`，避免重复序列化/校验。
- `frontend/dist/.keep`：
  - 目的：避免 `go test` 因 `//go:embed all:frontend/dist` 在未构建前端时无法匹配路径而失败。
  - 权衡：构建依然依赖真实 `frontend/dist`（由 `wails build` 自动生成），`.keep` 仅用于编译期占位；同时通过前端构建脚本保证 `.keep` 在构建后仍存在，避免影响 git 状态。

## 测试与验证方式 / 结果
- `go test ./...`：通过。
- Windows 构建：`wails build -platform windows/amd64`：通过，产物 `build/bin/myflowhub-win.exe`。
- 冒烟验证步骤：见 `README.md`（启动 server → 启动 app → Connect → Register/Login → 观察 Connected 与 Logs）。

## 潜在影响与回滚方案
### 潜在影响
- 若后续仍需保留 Fyne 版本，将需要维护两套 UI；本次变更已删除旧 UI 代码，默认不再支持回退到 Fyne 运行时。

### 回滚方案
- 回滚对应提交即可恢复旧 UI 文件与依赖；但建议仅在确认 Wails 版本不可用时执行。
