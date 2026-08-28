# Desktop Resource Workspace v2

## Status

这是已实现的 Desktop Profile、资源浏览器、renderer 和本地 View canonical contract。

## Product boundary

Desktop 是通用 Node/Resource 工作台。它使用 SDK 和 generated Wails boundary，不解析 wire、
不自行路由，也不把按钮可见性当作权限裁决。

## Profile and login

Profile 隔离一个 Hub endpoint、节点 identity/trust、非秘密偏好和 Views。每个应用实例最多一个
active Profile。启动时若 credential store 中存在有效凭据则自动连接；否则显示登录页。

secret 只能通过 `CredentialStore` 保存，首选系统安全存储；不可用时必须明确报告并允许仅会话使用，
不得退化为 settings JSON 明文。Profile switch 先关闭 subscription/session/connection，再原子激活新配置；
失败保持可恢复的未连接状态。

## Shell

左侧窄工作栏使用 tabs 切换 Resource Explorer 与 View Manager，右侧是 Workspace。
Explorer 展示 authoritative Node tree；每个 Node 下展示它拥有的 Resources。选择 Node 或 Resource
打开临时预览。拖放、键盘添加或命令面板可把资源加入当前 View。

## Renderer registry

renderer 按 type、capability 和 schema 选择，presentation hint 只决定默认偏好。内置 renderer 覆盖：

- Variable：当前值、更新时间和有权限时的编辑器；
- Stream/Topic：有界事件列表、pause/resume、gap 与 publisher；
- Command：schema-driven form、调用状态和明确错误；
- File：会话选择、进度、校验和完成状态；
- unknown：显示完整 descriptor 与 generic capability inspector。

renderer 不认识资源时必须使用 unknown fallback，不能隐藏资源或崩溃。

## Views

View 属于 Profile，首版只在本机保存。文档包含 version、id、name、布局和 widget 配置。
布局是响应式可调整网格，不是无限画布；每个 widget 保存 resource ID、renderer ID、grid position
和非秘密设置。加载时缺失资源显示 detached 状态，损坏或未知版本不会被静默覆盖。

## Interaction and accessibility

拖放必须有等价键盘入口；树、tabs、dialog、resize handle、commands 和表单具有可见 focus、标签与
语义状态。loading、empty、offline、forbidden、expired、gap、unknown、detached 和 corrupt 都有明确呈现。

视觉方向是安静的技术工作台：温暖浅色背景、墨色层级、低饱和青绿 accent、紧凑但不拥挤的排版，
依靠间距、细边框和状态密度建立层次，而不是渐变堆叠或装饰性卡片墙。
