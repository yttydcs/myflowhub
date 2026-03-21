# 2026-02-22 Win：本机 Local Hub（下载/启动 hub_server）

## 背景 / 目标

此前 `MyFlowHub-Win` 更偏向“客户端/控制台”，无法在本机直接提供 Hub 能力；当需要在本机做管理面/设备树/协议联调时，往往还要手工启动独立的 `hub_server` 并维护其生命周期。

本次变更目标：

- Win 侧新增 `Session → Local Hub` 页面，提供 **下载 / 安装 / 启动 / 停止 / 重启** `hub_server` 的入口；
- `hub_server` 以 **sidecar 子进程**方式运行，Win 本身不重新实现 server/路由能力；
- 端口冲突时 **自动选择可用端口**；
- Win 退出后 Hub **继续运行**（不自动 stop）；
- 监听地址允许配置；对非回环监听在 UI 上做安全提示；
- 下载来源采用 `MyFlowHub-Server` 的 GitHub Releases（资产 + `checksums.txt` 校验）。

## 具体变更内容

### 1) MyFlowHub-Server：Release 构建流水线（SRV1）

新增 GitHub Actions：当 push `v*` tag 时自动构建并发布 Release assets：

- `hub_server_windows_amd64.zip`（含 `hub_server.exe`）
- `hub_server_linux_amd64.zip`（含 `hub_server`）
- `checksums.txt`（`sha256sum hub_server_*.zip`）

文件：

- `MyFlowHub-Server/.github/workflows/release-hub-server.yml`

备注：

- 当前仓库若还没有 Release，需要在合并到主分支后 **推送一个 `v*` tag** 才能在 GitHub 上生成可供 Win 下载的 assets。

### 2) MyFlowHub-Win：LocalHubService（WIN1）

新增后端服务 `internal/services/localhub`，负责：

- 查询 GitHub latest release（`/repos/{owner}/{repo}/releases/latest`）
- 下载 `checksums.txt` 并解析 sha256
- 以流式方式下载 zip（避免一次性读入内存），边下载边计算 sha256，完成后与期望值比对
- 解压安装二进制到用户配置目录下的 `localhub/bin/`
- 安装覆盖：为兼容 Windows 的文件覆盖语义，写入/解压时会先清理旧文件；并在 hub 运行中禁止安装（避免 exe 锁导致失败）
- 启动 `hub_server` 子进程（stdout/stderr 重定向到 `localhub/logs/`）
- 停止/重启：非 Windows 尝试 `SIGTERM` + 超时 fallback `Kill`；Windows 直接 `Kill`
- 端口冲突：若指定端口不可用则自动挑选可用端口，并在运行状态中暴露实际 `addr`

主要文件：

- `MyFlowHub-Win/internal/services/localhub/*.go`
- `MyFlowHub-Win/app.go`（Bindings 增加 `LocalHubService`）

安装目录约定：

- 以 `internal/storage.Store.BaseDir()` 为根（当前为用户配置目录下的 legacy fyne 路径）
- 追加子目录：`localhub/`（其中包含 `bin/`、`logs/`、`downloads/`）

配置持久化（当前为全局 key，不按 profile 分隔）：

- `localhub.host`（默认 `127.0.0.1`）
- `localhub.port`（默认 `9000`；`0` 表示自动选择）
- `localhub.installed.tag` / `localhub.installed.at`（安装版本与时间，仅用于展示）

### 3) MyFlowHub-Win：前端页面（WIN2）

新增 `Session → Local Hub` 页面：

- 展示 latest release 状态与错误信息
- 一键安装 latest
- host/port 配置（port=0 表示自动选择）
- Start/Stop/Restart 按钮
- 对非回环 host 显示安全提示
- 展示运行态：`addr`、`pid`、`logPath`

文件：

- `MyFlowHub-Win/frontend/src/pages/LocalHub.vue`
- `MyFlowHub-Win/frontend/src/router/index.ts`
- `MyFlowHub-Win/frontend/src/layout/AppShell.vue`

## 任务映射（plan.md）

- SRV1：Server Release 流水线（已提交）
- WIN1：LocalHubService（已提交）
- WIN2：Local Hub 页面/路由/nav（已提交）

## 关键设计决策与权衡

1) **sidecar 而非内嵌 server**
- 优点：Win 保持上层应用定位；协议机制/路由/模块注册仍由 Server 统一维护；避免 Win 再实现一套“伪 server”。
- 代价：需要管理二进制下载、安装目录、进程生命周期；但这些都封装在 LocalHubService 内，边界清晰。

2) **GitHub Releases + checksums.txt**
- 优点：实现简单、可审计；避免 Win 从源码编译；checksums 可检测下载损坏。
- 注意：sha256 校验主要防损坏，不等同于供应链签名；后续如需更高安全性可引入签名/验证链。

3) **端口冲突自动选择**
- 优点：减少“启动失败”的阻塞；适合开发场景。
- 代价：实际端口可能与用户预期不同，因此 UI 必须展示最终 `addr`（已实现）。

4) **Win 退出后 Hub 继续运行**
- 优点：符合“把 Hub 当作本机服务”使用方式；避免误杀用户手工启动的 Hub。
- 代价：需要用户在需要时显式 Stop；未来若要“优雅守护/自愈/系统服务”可另起 workflow。

## 测试与验证方式 / 结果

已完成（本地）：

- `MyFlowHub-Server`：`GOWORK=off go test ./...`、`GOWORK=off go build ./cmd/hub_server` 通过
- `MyFlowHub-Win`：`GOWORK=off go test ./...` 通过
- `MyFlowHub-Win`：`GOWORK=off wails build -nopackage` 通过

需要人工冒烟（依赖 GitHub Release assets 可用）：

1. 在 `MyFlowHub-Server` 合并并推送一个 `v*` tag，等待 GitHub Actions 生成 Release assets
2. 启动 `MyFlowHub-Win` → `Session → Local Hub`
3. 点击 `Install Latest`，确认下载/校验/解压完成
4. 点击 `Start`，确认页面展示 `addr/pid/log`
5. 回到 `Home`，连接 `addr`，执行 register/login

## 潜在影响与回滚方案

影响：

- 新增 LocalHubService 与前端页面/路由/导航项；不影响既有协议/业务逻辑路径
- 会在用户配置目录下新增 `localhub/` 目录（包含下载缓存与日志）

回滚：

- 回滚 `MyFlowHub-Win` 相关提交即可移除 UI 与 LocalHubService
- 回滚 `MyFlowHub-Server` workflow 提交即可停止 release 自动发布（不会影响已有 tag/release）
