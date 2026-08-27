# 仓库与模块边界规范

## Status

- 状态：Accepted architecture baseline。
- 对应决策：[使用单一 Canonical Monorepo](../decisions/2026-08-27_single-canonical-monorepo.md)。
- 本文定义目标源码布局与依赖约束，不表示仓库迁移已经完成。

## Scope

本文约束：

- canonical monorepo 的源码分类；
- Go module 与 package 边界；
- 运行时、传输、应用和 SDK 的依赖方向；
- 旧仓迁移与历史文档保留原则；
- 构建和发布单元与源码仓的关系。

## Target Source Layout

```text
MyFlowHub/
├── docs/                  长期需求、规范、决策与历史入口
├── protocol/              wire 类型、schema、代码生成与跨语言夹具
├── runtime/
│   ├── link/              帧、LinkSession、连接生命周期
│   ├── tree/              节点树、路由与 reparent
│   ├── auth/              身份、准入、请求与控制裁决
│   ├── resource/          Variable、Stream、Command
│   └── subscription/      订阅、租约、聚合与背压
├── transport/             TCP、QUIC、RFCOMM、串口等适配器
├── feature/               File、Flow 等建立在核心原语上的能力
├── host/                  Hub 和其他运行时组合入口
├── sdk/                   第一方客户端 SDK
├── apps/                  Desktop、Android 与节点应用
├── embedded/              C 与 MicroPython SDK
└── tests/                 contract、integration 与 transport matrix
```

目录名可在实施计划中按语言和工具链调整，但职责关系不得反转。

## Go Module Policy

1. 默认只有一个根 `go.mod`。
2. 内部可替换性通过 interface、package 和测试实现，不通过独立 module 实现。
3. 不为 Variable、Stream、Command、Subscription、Transport 或 Feature 分别建立 module。
4. 嵌套 module 只允许用于已验证的工具链约束，例如某些 gomobile 或独立代码生成工具；每个例外必须记录所有者、原因和退出条件。
5. 本地开发不得依赖长期 `replace => ../MyFlowHub-*` 维持正常构建。
6. 禁止使用 Git submodule 或嵌套 `.git` 目录作为内部组件管理方式。

## Dependency Direction

### protocol

- 不依赖 runtime、host、sdk 或 apps。
- 只包含稳定 wire/schema、生成器输入输出和共享测试夹具。
- 不包含连接、路由、权限或业务运行时逻辑。

### runtime

- 可以依赖 protocol。
- 不依赖具体应用、UI 或平台宿主。
- Node Tree、LinkSession、Auth、Resource、Subscription 和 Command 的核心语义在此形成闭环。

### transport

- 实现 runtime/link 定义的最低承载接口。
- 可以依赖平台 API，但不得向 runtime/resource、Subscription、Command 或应用层泄漏具体 Transport 类型。
- runtime 核心不得通过类型判断硬编码 TCP、RFCOMM、QUIC 等实现。

### feature

- 依赖 runtime 提供的 Resource、Subscription 和 Command 原语。
- File、Flow 等不是新的核心路由或权限体系，不得重新建立 SubProto 风格的垂直协议栈。

### host

- 是组合根，可以选择 runtime、transport 和 feature 实现并启动 Hub 或节点宿主。
- 不拥有另一套协议、权限或订阅逻辑。

### sdk

- `sdk/go` 和普通 binding 是客户端层，不依赖 `host/`。
- 唯一例外是 `sdk/bindings/android/host.go`：Android 产品在同一 gomobile artifact 中暴露可选 in-process Hub，因此该文件是平台组合根，只允许导入 `host/hub`。architecture test 固定这一精确 file/import pair，例外不得扩散。

- 依赖 protocol 和面向客户端的公共契约。
- 不依赖 host 或应用实现。
- 第一方应用不得复制发送、等待、超时、重连或订阅生命周期逻辑。

### apps

- 根据角色依赖 sdk 或 host。
- UI 和平台层只负责交互、权限申请、平台 Provider 和产品行为，不重新实现 runtime。

### embedded

- 复用 protocol schema、夹具和行为测试，但可以使用适合 C/MicroPython 的独立实现。
- Embedded 实现必须通过相同 contract tests 验证，而不是维护一份仅靠文档同步的协议副本。

## Current-to-Target Mapping

| Current repository | Target responsibility |
| --- | --- |
| MyFlowHub-Proto | `protocol/` |
| MyFlowHub-Core | `runtime/link`、`runtime/tree`、`transport/` |
| MyFlowHub-SDK | `sdk/` |
| MyFlowHub-Server | `host/` 与必要的 runtime 组合逻辑 |
| MyFlowHub-SubProto/auth | `runtime/auth` |
| MyFlowHub-SubProto/varstore、stream、topicbus | `runtime/resource` 与 `runtime/subscription` |
| MyFlowHub-SubProto/exec | `runtime/resource` 中的 Command 与调用机制 |
| MyFlowHub-SubProto/broker | `runtime/subscription` 内部实现 |
| MyFlowHub-SubProto/forward | `runtime/tree` 路由实现 |
| MyFlowHub-SubProto/file、flow | `feature/`，建立在核心原语之上 |
| MyFlowHub-Win、Android | `apps/desktop`、`apps/android` |
| MetricsNode、ClipboardNode | `apps/nodes` |
| MyFlowHub-EmbeddedSDK | `embedded/` |

上述映射表达职责去向，不要求机械复制旧目录。旧实现只有在符合新契约时才复用。

## Compatibility Policy

- 已确认当前没有项目外部用户依赖旧 module path 或旧 wire 协议。
- 新架构可以进行破坏性 API、module path、wire envelope 和持久化格式调整。
- 不建立长期兼容承诺。
- 如为了分批迁移第一方应用而引入兼容桥，兼容桥必须：
  - 位于明确的 legacy/adapter 边界；
  - 不进入新核心依赖方向；
  - 有覆盖范围、删除条件和最迟删除阶段；
  - 不被新功能继续依赖。

## Migration and History Preservation

1. 新 monorepo 是新的 canonical 实现来源；旧仓在切换前继续描述旧系统事实。
2. 旧仓不得在迁移中被直接删除或重写历史。
3. 新仓需要维护迁移清单，至少记录旧仓、旧 module、来源 commit/tag、新目录、迁移状态和验证证据。
4. 稳定架构文档和 ADR 优先迁入；历史 plan/change 保留为只读演进证据，不自动升级为当前规范。
5. 不要求拼接全部 Git 历史。旧仓远端和 tag 本身承担原始历史保存，新仓通过文档建立可追溯链接。
6. 旧仓设置只读、归档远端或停止发布必须在新系统完成切换并验证后单独执行。

## Build and Release Boundaries

- 单一 Git 仓库不等于单一构建产物。
- Hub、Desktop、Android、节点应用、Go SDK 和 EmbeddedSDK 可以使用独立入口、CI job 和版本号。
- CI 应按依赖和路径选择任务，但合并门禁至少覆盖协议 contract、runtime 核心测试和受影响的集成闭环。
- 内部 package 不单独发布标签；只有真正面向使用者的产物才形成发布单元。

## Non-goals

- 不保留现有 10 仓、22 module 的长期发布结构。
- 不保证旧 import path、SubProto action 或 wire frame 的兼容性。
- 不在本规范中决定具体迁移提交、Task ID、CI 平台配置或远端归档时间。
- 不因为采用 monorepo 而取消 runtime、transport、feature、host 和 app 的依赖边界。

## Related Intake

- [节点树、订阅与指令重构诉求](../intake/2026-08-27_node-tree-subscription-command-redesign.md)

## Related Requirements

- [统一节点运行时需求](../requirements/unified-node-runtime.md)

## Related Decisions

- [使用单一 Canonical Monorepo](../decisions/2026-08-27_single-canonical-monorepo.md)
- [统一权威节点树与可插拔链路](../decisions/2026-08-27_authoritative-node-tree-and-pluggable-links.md)

## Related Specs

- [节点树、链路与资源架构规范](node-tree-link-resource-architecture.md)

## Related Current Workspace Docs

- [当前仓库职责](../../repos.md)
- [当前根 Go module](../../go.mod)

## Related Changes

- [Canonical Monorepo 与统一节点运行时第一阶段](../change/2026-08-27_canonical-monorepo-unified-node-runtime.md)
