# MyFlowHub canonical 仓库边界

`repo/MyFlowHub` 是一套产品、一个 Git 仓库和一个 Go module：`github.com/yttydcs/myflowhub`。协议、运行时、Transport、宿主、SDK 与第一方应用在同一版本中演进和验证，不再通过多个内部仓库、`go.work` 或 sibling `replace` 拼装。

## 工作区布局

```text
workspace/
├── repo/
│   └── MyFlowHub/      canonical Git checkout
├── worktrees/          sibling Git worktrees
└── local assets        本机 SDK、附件等非仓库资料
```

仓库内不得创建第二个 checkout、嵌套 `.git` 或 `worktrees/`。本机工具链和未纳入版本控制的资料留在工作区根，不进入 canonical checkout。

## 接手入口

- `README.md`：产品定位、目录和本地启动入口；
- `docs/README.md`：当前有效架构、规范、决策与功能 dossier 的总索引；
- `plan.md` / `todo.md`：当前 workflow 的控制面和执行状态；
- `migration/inventory.json`：旧实现逐项迁移、替代或删除的审计记录；
- `docs/specs/repository-and-module-boundaries.md`：可由自动化校验的详细边界。

## 依赖方向

```text
protocol
  ↓
runtime/{link,tree,auth,resource,subscription,command,node}
  ↓
transport · feature · host · sdk
  ↓
cmd · apps
```

- `protocol/` 只定义 transport-neutral envelope、codec 和版本化 payload schema。
- `runtime/link/` 消费抽象字节流；具体 TCP、QUIC、RFCOMM 实现在 `transport/`，不得把链路类型泄漏到资源或权限语义。
- `runtime/tree/` 的父子边同时是路由边和 authority 边；节点拥有资源，资源不是树节点。
- `runtime/resource/` 只提供 `Variable`、`Stream`、`Command`；订阅是 Variable/Stream 上的一等关系，Command 是有界补充手段。
- `feature/` 提供 File、Notification、Management 等可组合能力，不创建第二套协议或 dispatcher。
- `host/`、`sdk/` 负责稳定组合面；`cmd/`、`apps/` 是最终产品入口，不反向成为底层依赖。

## 产品位置

- Hub：`cmd/mfh-hub`、`host/hub`、`feature/management`；
- 管理 CLI：`cmd/mfh-admin`；
- Desktop：`apps/desktop`、`cmd/mfh-desktop`；
- Metrics：`apps/nodes/metrics/windows`；
- Clipboard：`apps/nodes/clipboard`；

Android 与 C/ESP32/MicroPython 旧实现已移除，见[重新设计待办](docs/requirements/mobile-embedded-redesign.md)。

所有产品必须复用 canonical runtime/SDK，不在 UI、平台 binding 或节点应用中复制 wire、路由、权限和订阅实现。

## 旧仓库状态

旧的 `repo/MyFlowHub-*` 多仓 checkout 已于 2026-08-27 在完成实现迁移和文档提炼后移除。当前唯一主 checkout 是 `repo/MyFlowHub`。精确远端、commit/tree、文件数量、未提交生成物 hash 与 disposition 保存在 `migration/`；旧仓不再是构建输入、运行时 fallback、发布单元或开发启动目标。追溯时应按审计记录在 sibling `worktrees/` 或临时目录检出对应提交，不得恢复旧 module path、SubProto API、wire bridge 或多模块 release。

## 常用命令

```powershell
$env:GOWORK='off'
./scripts/mfh.ps1 -Action test -Target core
./scripts/mfh.ps1 -Action check -Target all -AllowUnavailable
./scripts/run-dev.ps1 -WaitHub
```

构建矩阵、工具链和产物位置见 `docs/specs/build-and-ci.md`。开发 Hub 使用默认拒绝策略；首次接入的离线 permit 与 policy 引导见 `docs/features/hub.md`。
