# Plan - Win：本机 Hub（下载/启动 hub_server）能力

> Workflow 目录：`d:\project\MyFlowHub3\worktrees\localhub\`
>
> 本 workflow 涉及 2 个仓库（同名分支，便于对齐）：
> - `MyFlowHub-Server`：`d:\project\MyFlowHub3\worktrees\localhub\MyFlowHub-Server`（branch：`feat/localhub`）
> - `MyFlowHub-Win`：`d:\project\MyFlowHub3\worktrees\localhub\MyFlowHub-Win`（branch：`feat/localhub`）
>
> guide.md 约束：
> - commit 信息使用中文（前缀如 `feat`/`fix` 可英文）

---

## 1. 项目目标与当前状态

### 1.1 目标
在 Windows 上让 `MyFlowHub-Win` 具备“本机 Hub”能力：
- 在 UI 中新增 `Session → Local Hub` Tab；
- 支持从 `MyFlowHub-Server` 的 GitHub Releases **下载** `hub_server` 二进制；
- 支持一键 **启动 / 停止 / 重启** 本机 `hub_server`；
- 端口冲突时自动选择可用端口；
- Win 退出后本机 Hub **继续运行**（不自动停止）；
- 监听地址允许配置（包含回环与局域网暴露），并对非回环给出安全提示。

### 1.2 当前状态
- `MyFlowHub-Win` 目前主要是“客户端/控制台”，不实现 management 等子协议的入站响应；因此对 `targetID=自身` 的 `management list_nodes` 会超时。
- `MyFlowHub-Server` 仓库当前没有 GitHub Releases（`/releases` 为空，`/releases/latest` 返回 404），需要补齐 release 构建与发布流水线。

---

## 2. 范围定义

### 必须（本 workflow）
- Server：支持 tag 触发的 GitHub Release，产出 `hub_server`（Windows/Linux amd64）压缩包与校验文件。
- Win：Local Hub 页签（下载/安装/启动/停止/重启/状态展示），以及必要的后端 service + 前端页面/路由/nav。

### 可选（本 workflow 若不增加复杂度）
- Win：启动后可选择自动填充/提示“建议连接到本机 Hub 地址”（不强制自动连接，避免影响现有手工连接流程）。

### 不做（本 workflow）
- 不将 `hub_server` “内嵌同进程运行”（避免 Win 重新实现 server/路由能力）。
- 暂不改 `hub_server` 的优雅停止控制面（不新增 admin HTTP / wire 以外的控制协议）。
- 不实现“本机 Hub 的完整守护/自愈”（例如系统服务注册、崩溃自动重启等）。

---

## 3. 任务清单（Checklist）

### DEV0 - 基线确认
**目标**
- 两个 worktree 均存在且分支正确
- 工作区干净

**验收**
- `MyFlowHub-Server` / `MyFlowHub-Win`：`git status` 无未提交改动
- 分支：均为 `feat/localhub`

**回滚点**
- 不适用

---

### SRV1 - Server：GitHub Release 流水线（hub_server）
**目标**
- 在 `MyFlowHub-Server` 新增 GitHub Actions：tag 推送后自动构建并创建 Release。
- 产物：
  - `hub_server_windows_amd64.zip`（含 `hub_server.exe`）
  - `hub_server_linux_amd64.zip`（含 `hub_server`）
  - `checksums.txt`（sha256 校验）

**涉及文件**
- `MyFlowHub-Server/.github/workflows/release-hub-server.yml`（新增）
- （可选）`MyFlowHub-Server/cmd/hub_server/main.go`（仅用于输出版本信息；如不需要可不改）
- （可选）`MyFlowHub-Server/README.md`（补充 release 产物说明）

**验收**
- 本地：`GOWORK=off go test ./...` 通过
- 本地：`GOWORK=off go build ./cmd/hub_server` 通过（Windows 环境下 build Windows 版本）
- GitHub：推送 tag 后能生成 Release 并附带上述 assets（由 CI 产出；本地不强制验证）

**测试点**
- Actions workflow 语法正确；构建矩阵为 windows/linux amd64

**回滚点**
- revert 本提交

---

### WIN1 - Win：LocalHubService（下载/安装/进程管理/状态）
**目标**
- 新增 Wails 后端 service：管理本机 `hub_server` 二进制与进程生命周期。
- 关键能力：
  - 查询 latest release 元数据；无 release 时给出可理解提示
  - 下载 assets + 校验 `checksums.txt`
  - 解压安装到用户配置目录下的 `localhub/`（不污染 repo）
  - 端口冲突自动选择（Win 侧预选可用端口；必要时重试）
  - Stop/Restart：先尝试发送终止信号（仅非 Windows）再 fallback kill；Windows 直接 kill
  - Win 退出不自动 stop（保留后台运行）

**涉及文件（预计）**
- `MyFlowHub-Win/internal/services/localhub/*`（新增）
- `MyFlowHub-Win/app.go`（绑定 service）
- `MyFlowHub-Win/internal/storage/store.go`（如需新增配置 key，尽量复用现有 Get/Set）

**验收**
- `GOWORK=off go test ./...` 通过
- 手工：能下载、安装、启动、停止 hub_server；并能从状态里拿到 listen addr / pid / 日志路径

**回滚点**
- revert 本提交

---

### WIN2 - Win：前端 Tab（Session → Local Hub）
**目标**
- 增加页面与路由，并在导航 Session 分组下新增入口。
- 页面包含：
  - Release 状态：latest / installed
  - 下载按钮与进度/错误提示
  - 监听地址配置（host + port；port=0 表示自动）
  - Start/Stop/Restart 按钮与运行状态
  - 非回环监听安全提示

**涉及文件（预计）**
- `MyFlowHub-Win/frontend/src/pages/LocalHub.vue`（新增）
- `MyFlowHub-Win/frontend/src/router/index.ts`（新增 route）
- `MyFlowHub-Win/frontend/src/layout/AppShell.vue`（新增 nav item）
- `MyFlowHub-Win/frontend/src/stores/localhub.ts`（可选：若需要 store 承载状态）

**验收**
- `GOWORK=off wails build -nopackage` 通过
- 手工：
  - 打开 `Session → Local Hub` 能看到 latest 状态
  - 下载后能启动本机 Hub
  - 端口冲突时能自动换端口并显示实际 addr

**回滚点**
- revert 本提交

---

### DEV3 - Code Review（阶段 3.3）
逐项输出结论（通过/不通过）：
- 需求覆盖
- 架构合理性（Win 仍为上层应用；Hub 能力通过子进程提供）
- 性能风险（下载/校验/解压的 I/O；避免一次性读入大文件）
- 可读性与一致性
- 可扩展性（后续可加优雅停止/自动重启/多版本）
- 稳定性与安全（监听地址暴露提示；校验下载资产）
- 测试覆盖情况（build + 冒烟）

---

### DEV4 - 归档变更（阶段 4）
- 在 workflow 根目录创建：`docs/change/`
- 新增：`docs/change/2026-02-22_win-localhub.md`
  - 背景/目标、具体变更、任务映射、关键设计权衡、验证方式/结果、回滚方案

---

## 4. 依赖关系与注意事项
- `WIN1/WIN2` 依赖 `SRV1` 产出 Releases 才能完整联调；但 Win 侧可先实现“无 release 时的可理解提示”。
- GitHub API 若未认证存在 rate limit；先按匿名访问实现，必要时后续补 token 支持。
- 本机 Hub 以子进程方式运行；Win 退出不负责 stop，需避免在 `Shutdown()` 中自动 kill。

