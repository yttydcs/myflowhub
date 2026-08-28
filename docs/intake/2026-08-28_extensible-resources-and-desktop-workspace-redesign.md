# 2026-08-28 可扩展资源平台与 Desktop 工作区重构

## Source

- 来源：用户在当前 Codex 会话中的直接请求。
- 日期：2026-08-28。
- 工作区：`D:/project/MyFlowHub3`。
- Canonical repository：`repo/MyFlowHub`。
- Discussion branch：`refactor/resource-platform-desktop-workspace`。
- Discussion worktree：`worktrees/resource-platform-desktop-workspace`。

## Request Text / Source-preserving Summary

用户希望进行比现有 vNext 更大范围的 clean-break 重构：

- 一切设备都是 Node；
- Node 下的一切能力和数据都是 Resource；
- Resource 具有不同类型，不同类型支持不同操作；
- Variable、Stream、Topic 都应是一等 Resource；
- Desktop 不再停留在简陋固定页面，而要成为极简、舒适、可组合的资源工作台；
- Desktop 需要登录界面、登录持久化和多个 Profile；
- 主界面左侧在资源树浏览器与视图管理器之间通过 Tab 切换；
- 资源树同时展示 Node 以及 Node 拥有的 Resource；
- 右侧是工作区；点击 Node 或 Resource 显示预览；Resource 可拖入工作区组合，并保存为 View；
- UI 倾向使用 shadcn/ui 等成熟组件体系。

## Problem / Opportunity

当前核心把 Resource kind 固定为 Variable、Stream、Command，产品能力通过这些原语组合；Desktop 前端是 Vanilla TypeScript + 手写 CSS，以固定产品页为主。这个模型已经建立统一树、权限和订阅闭环，但存在两个新的产品级限制：

1. 新能力只能继续被压缩进三种固定 kind，或在 feature 层反复组合，难以直接表达 Topic、File、Media 等具有独特操作语义的资源。
2. Desktop 只能按预置页面消费资源，尚不是可发现、可预览、可组合、可持久化视图的通用工作台。

本轮机会是把 Node/Resource 关系提升为唯一产品抽象，并让 Desktop 成为该抽象的主要可视化入口。

## Confirmed Goals

- 继续保持唯一 authoritative Node tree；Resource 必须由一个 Node 拥有，Resource 不建立第二棵权限树或路由树。
- Resource type 可扩展；类型声明决定可用操作、payload/schema、QoS 和默认 UI renderer。
- 至少覆盖 Variable、Stream、Topic、Command；File、Media 等能力能够作为扩展类型进入，而不恢复 SubProto。
- Desktop 使用 React + TypeScript + Vite，采用 shadcn/ui/Tailwind 形成统一设计系统。
- 登录成功后的身份和连接状态可安全持久化；下次启动可自动进入或自动连接。
- 支持多个相互隔离的 Profile。
- 左侧提供 Node/Resource tree 与 View manager 两个 Tab。
- 右侧工作区支持 Node/Resource preview、拖放组合、调整布局和保存 View。
- 整体视觉保持克制、低噪声、舒适，避免传统运维控制台的高密度边框和重复卡片。

## Recommended Architecture Direction

### Extensible resource contract

不要把所有新类型继续硬编码到 Core 的大型 `switch`。推荐 Resource descriptor 至少包含：

- stable resource reference：`owner_node_id + local_resource_id`；
- versioned type identifier；
- supported capabilities/operations；
- input、output、event 与 configuration schema；
- permission metadata；
- size、rate、retention、reliability 等边界；
- 非权威的 UI presentation hint。

Core 只拥有寻址、描述、权限裁决、路由、订阅关系、操作调用和生命周期；类型包实现具体行为。未知类型仍可被 catalog 发现，并由通用 JSON/binary inspector 展示。

建议的第一批操作族：

| Operation | Typical resource types | Meaning |
| --- | --- | --- |
| `read` / `write` | Variable | 当前值与有条件更新 |
| `subscribe` | Variable / Stream / Topic | 建立有权限和租约的观察关系 |
| `publish` | Stream / Topic | 发布事件；具体 publisher 约束由类型定义 |
| `invoke` | Command | 有界请求/结果 |
| `open` / session operations | File / Media | 协商受控数据会话，不把大数据硬塞进控制消息 |

### Topic semantic recommendation

Topic 应是由某个 Node 拥有和管理的 brokered Resource，而不是全局匿名字符串：

- 多发布者、多订阅者；
- `publish` 与 `subscribe` 分离授权；
- 默认不保留当前值；
- 默认只保证单 publisher 顺序，不虚构全局总序；
- retention/replay/QoS 必须显式声明，不能由 `Topic` 名称隐含；
- 所有路由和撤权仍经过 owner 所在的 authoritative tree。

### Desktop renderer and workspace model

Desktop 应维护 Resource Renderer Registry：

- `preview`：单击后的临时预览；
- `widget`：拖入工作区后的持久组件；
- `editor/actions`：该类型支持的写入、发布、调用或会话操作；
- `fallback`：未知类型的 descriptor、JSON、binary metadata 和原始操作入口。

推荐 View 使用响应式网格而不是无限自由画布。View 保存 Resource reference、renderer、位置、尺寸和局部展示设置，不复制资源正文。

```text
Desktop
├── Login / Profile chooser
└── Main shell
    ├── Left panel
    │   ├── Resource Explorer tab
    │   │   └── Node tree -> owned Resources
    │   └── Views tab
    │       └── saved / recent Views
    └── Workspace
        ├── transient Node/Resource preview
        └── saved View -> resizable Resource widgets
```

### Profile recommendation

Profile 不是简单的主题偏好，也不等同于远端用户名。推荐一个 Profile 隔离保存：

- Hub endpoint 与 Transport 设置；
- 本地 Node identity 和受信任父节点；
- 首次 admission 结果与可恢复会话状态；
- 视图集合、最近打开项和 UI 偏好；
- secret reference，而不是明文密钥或长期 permit。

首版每个应用实例只激活一个 Profile；切换 Profile 会明确关闭旧连接、释放订阅，再打开新身份。敏感材料进入操作系统 credential store；普通布局和 View 文档使用版本化原子本地存储。

## UI Technology Research

- shadcn/ui 官方支持在现有 Vite + React + TypeScript 项目中初始化，并提供 Sidebar、Tabs、Resizable、Scroll Area、Context Menu、Command、Card 等适合桌面工作台的组件。
- Wails 官方支持 React/TypeScript/Vite 前端，并通过生成绑定连接 Go host。
- shadcn/ui 没有直接提供完整资源树或工作区布局引擎；推荐用可访问的 shadcn primitives 自建 Tree row，并使用 dnd-kit 处理拖放。
- 大型展开树可按需要引入 TanStack Virtual；不要在节点规模尚小时过早虚拟化。

Primary sources：

- <https://ui.shadcn.com/docs/installation/vite>
- <https://ui.shadcn.com/docs/components>
- <https://ui.shadcn.com/docs/components/base/sidebar>
- <https://wails.io/docs/introduction/>
- <https://dndkit.com/react/quickstart/>
- <https://tanstack.com/virtual/latest/docs/introduction>

## Options Considered

| Area | Option | Tradeoff | Recommendation |
| --- | --- | --- | --- |
| Resource types | Core hard-coded enum | 简单，但每种新类型都扩大 Core 和 wire switch | Rejected |
| Resource types | Extensible type + capability descriptor | 边界清晰，未知类型可发现，需要新的注册与验证机制 | Recommended |
| Resource types | 所有行为都退化为 Command | wire 最小，但失去订阅、发现和类型化 UI | Rejected |
| Workspace | 固定产品页面 | 实现成本低，但继续限制扩展 | Rejected |
| Workspace | 无限自由画布 | 最自由，但交互、对齐、键盘操作和持久化复杂 | Deferred |
| Workspace | 响应式可调整网格 | 满足组合视图，仍保持克制和可维护 | Recommended |
| Frontend | 继续 Vanilla TS | 依赖少，但组件、状态和拖放维护成本高 | Rejected |
| Frontend | React + shadcn/ui | 与 Vite/Wails兼容，组件源码可控，适合工作台 | Recommended |

## Constraints and Risks

- “一切都是 Resource”不能演变为“所有类型实现都进入 Core”，否则会重新制造 SubProto 式垂直耦合。
- Topic 与 Stream 必须有可测试的差异，否则 Topic 只是重复命名。
- File/Media 的大数据面不能阻塞树上的认证、心跳、权限和控制消息。
- 自动登录不能通过明文保存密码、私钥或一次性 permit 实现。
- 拖放不是唯一操作路径；资源必须支持单击、键盘添加和上下文菜单，保证可访问性。
- View 只能保存资源引用和展示状态；资源失联、权限撤销、类型升级时必须显示明确占位和恢复路径。
- 当前 accepted resource spec、Desktop feature 和 authoritative-tree ADR 会受到影响；在用户确认前不覆盖它们。

## Open Questions / Recommended Defaults

1. Topic 语义：建议确认“Node-owned、多发布者/多订阅者、publish/subscribe 分权、默认无持久重放”。
2. Profile 语义：建议确认为“一个 Hub/身份/信任域/视图集合的本地隔离容器”，首版一次只激活一个。
3. View 存储：建议首版 local-first、按 Profile 原子持久化；后续把 View 文档暴露为 Desktop Node 拥有的 Resource，再支持同步。
4. Workspace：建议首版使用可调整响应式网格，不做无限画布。
5. 第一阶段 Resource types：建议 Variable、Stream、Topic、Command 为基础；File 和 Media 采用扩展类型及独立数据会话，不把大数据面直接塞进通用 Stream。

## Stable Docs Impact

- Intake impact: add（本文）。
- Feature impact: 待确认后重写 `docs/features/desktop.md`。
- Requirements impact: 待确认后新增资源平台与 Desktop 工作区长期需求。
- Specs impact: 待确认后 supersede 当前固定三类型 resource model，并补充 Topic、renderer/view contract。
- Decision impact: 待确认后新增取代或扩展现有 resource-model 决策的 ADR；保留唯一 Node tree 与 Resource ownership 不变。
- Lessons impact: none。

## Handoff Criteria for `$m-plan`

只有在 Open Questions 的推荐默认值被确认或逐项调整后，才能进入 `$m-plan`。规划阶段必须覆盖 Core contract、迁移策略、Desktop React 重建、Profile/credential storage、renderer registry、View persistence、兼容/删除边界和端到端门禁；本 intake 本身不授权实现。

