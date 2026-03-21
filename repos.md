# MyFlowHub3 仓库职责与接手者指南

> 本文档只保留当前有效的仓库边界、依赖规则和接手建议。  
> 已完成 workflow 的详细里程碑、验收记录、旧路线图正文，统一看 `docs/change/README.md` 与 `docs/plan_archive/README.md`。

## 1. 先看哪里

- `plan.md`
  - 根级控制面索引，只看当前状态、入口和文档分工。
- `docs/README.md`
  - `docs/` 总入口。
- `docs/change/README.md`
  - 已完成变更的结果归档。
- `docs/plan_archive/README.md`
  - 历史 workflow 的完整计划、Checklist、Review 线索。
- `repo/MyFlowHub-Server/docs`
  - 当前子协议规范文档所在地。

## 2. 这个目录是什么

`d:\project\MyFlowHub3` 是多仓协作工作区，不是单一代码仓。

- `repo/`
  - 控制面和集成面。
  - 只做仓库管理、合并、发布、集成验证、worktree 管理。
- `worktrees/`
  - 实现面。
  - 所有实现性改动都应在这里的独占 worktree 中完成。
- `docs/`
  - 全局归档和接手材料。
- `guide.md`
  - 当前工作区规范。
- `target.md`
  - 当前根目录不存在；历史文档引用它时按历史上下文理解，不要自动补回。

## 3. 依赖方向与硬边界

### 基本分层

```text
Proto
  -> Core
    -> SDK
      -> Win
      -> MetricsNode（客户端侧复用）
    -> SubProto
      -> Server

Android
  -> 复用 Core / SDK / Server hubruntime 的 Go 能力
  -> 负责 Android 宿主、前台服务、gomobile 集成、平台 Provider
```

### 必守规则

- `MyFlowHub-Proto`
  - 只放协议字典、Action、payload、常量；不放运行时逻辑。
- `MyFlowHub-Core`
  - 只放底层框架能力：header、transport、router、process、连接抽象；不放具体业务协议实现。
- `MyFlowHub-SDK`
  - 只放客户端统一能力：session、envelope、await、错误语义；不依赖 Server/Win。
- `MyFlowHub-SubProto`
  - 只放子协议实现 module；module 之间禁止互相耦合，公共能力抽到显式 shared module。
- `MyFlowHub-Server`
  - 只做运行时装配、入口、默认模块集和集成测试；不要把新的子协议实现塞回 Server。
- `MyFlowHub-Win`、`MyFlowHub-Android`、`MyFlowHub-MetricsNode`
  - 都是上层应用或节点宿主；不要在 UI 层重复实现协议栈或复制 Server 内部机制。

## 4. 各仓库职责

- `MyFlowHub-Proto`
  - 协议字典仓，定义 `SubProto`、`Action`、JSON 结构和稳定 wire 语义。
- `MyFlowHub-Core`
  - 底层通信与调度框架仓，承载 HeaderTcp、连接、路由、process/dispatcher 等共用基础设施。
- `MyFlowHub-SDK`
  - 客户端 SDK 仓，统一发送封装、等待响应、超时/取消、trace/hop 等客户端语义。
- `MyFlowHub-SubProto`
  - 子协议实现仓，按 module 交付 `auth`、`management`、`varstore`、`topicbus`、`file`、`exec`、`flow` 等实现，以及共享模块如 `broker`。
- `MyFlowHub-Server`
  - Hub/Login 运行时装配仓，负责 `cmd/*` 入口、`modules/*` 组合、`hubruntime`、默认启用集合和服务端集成测试。
- `MyFlowHub-Win`
  - Windows Wails/Vue 应用仓，负责桌面 UI、面向 UI 的 services、调试与运维入口；通过 SDK/Proto 访问服务端能力。
- `MyFlowHub-Android`
  - Android 应用宿主仓，负责 Compose UI、Foreground Service、gomobile 绑定壳、嵌入式 Hub 宿主，以及 Android 专属平台能力接入。
  - Android 作为 Hub 时，嵌入运行时复用 `MyFlowHub-Server/hubruntime`，不要在 Kotlin 里重写 Go 侧协议逻辑。
  - Android 专属 transport/provider 接入点也放这里，例如 RFCOMM Provider。
- `MyFlowHub-MetricsNode`
  - 独立节点应用仓，负责 Windows/Android 指标采集、VarStore 上报、Devices Config 远程配置、`nodemobile` 绑定与各自 UI。
  - 这是“节点应用”，不是 Win 的子模块，也不是 Server 的内置功能。

## 5. 当前有效路线

- 根基拆分已经完成：
  - Proto/Core/SDK/SubProto/Server/Win 的职责边界已基本成型。
  - 已完成的详细过程不要再写回根级文档，直接查 `docs/change/README.md`。
- 新工作优先遵守现有边界：
  - 协议结构改动先落 `Proto`，再同步服务端文档和上下游消费方。
  - 通用 transport / runtime / await 能力优先放 `Core` 或 `SDK`，不要先做仓内私有实现。
  - 子协议业务放 `SubProto`，Server 只做装配。
  - 平台宿主、平台权限、平台 Provider 放对应应用仓，例如 Android。
- 当前明确仍处于延后状态的方向：
  - `PR2-4` 编译期裁切与 `minimal/full` 变体产品化仍为 deferred。
  - 没有明确产品需求前，不要重启这一条。
- 依赖管理建议：
  - 日常依赖优先用 tag 版本。
  - `replace => ../MyFlowHub-*` 只作为本地联调或跨仓 workflow 手段，不应长期保留在主线。

## 6. 接手时常用命令与注意点

### Go 测试

```powershell
$env:GOWORK='off'
$env:GOTMPDIR='d:\\project\\MyFlowHub3\\.tmp\\gotmp'
New-Item -ItemType Directory -Force -Path $env:GOTMPDIR | Out-Null
go test ./... -count=1 -p 1
```

### 常见踩坑

- 不要在 `repo/` 里直接做实现改动，那里是控制面。
- 不要在 Win 或 Android UI 里复制一套协议发送/等待语义，优先复用 SDK / Go runtime。
- 不要让子协议 module 互相 import；共享逻辑抽 shared module。
- 不要把历史完成项继续堆在根级 `plan.md` 或 `repos.md`；完成后进 `docs/change` / `docs/plan_archive`。
