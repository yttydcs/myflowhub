# Desktop Codex 风格界面生产实现

## Summary

MyFlowHub Desktop 已从旧的卡片式资源工作区重构为以矿物蓝和石墨灰为主的平面分栏界面。实现保留现有 Wails 2、React、DesktopAPI、Profile/CredentialStore、Topology/Catalog、Renderer 与 View contract，原型只作为 UI/interaction 规格，没有把演示数据带入生产模型。

本 workflow 明确不包含项目品牌图标。现有品牌 slot 保持原状，图标设计、替换、平台派生与品牌 ADR 由另一任务处理。

## User-visible Changes

- 登录页改为紧凑的 Profile 登录表单，继续支持持久化 Profile、一次性 Permit、自动连接和已有 Profile 删除。
- 默认浅色；完整深色主题、树展开与 focused Node 以 versioned non-secret preference 按 Profile 保存。
- 顶栏只保留品牌、刷新和主题操作，不再重复显示连接状态或 Profile。
- 左侧支持 Resource/View 切换、资源搜索、任意深度 Node/Resource tree 和左下 Profile/connection Settings 入口。
- Tree 支持集中式展开、深层搜索、focused subtree、breadcrumb/back，以及 Arrow/Home/End/Enter/Space/asterisk 的 WAI-ARIA roving focus interaction。
- Node/Resource preview 移到右侧 Inspector；Node 可聚焦，Resource 可添加到 Workspace。
- Settings 作为主区完整 Tab，包含 Connection、Profile 与 Appearance。
- Workspace 只承载 View toolbar 和 widgets，保留按钮/拖放添加、布局调整、dirty guard、保存与删除确认。

## Implementation

- `store.ts` 新增 iterative authority index、parent cycle 截断、search index、flat visible rows、ARIA hierarchy metadata 与 breadcrumb derivation。
- `Explorer.tsx` 不再为每个递归分支保存本地 state；使用 memoized Map/Set、`useDeferredValue`、单一 roving tab stop 与可见行渲染。
- `preferences.ts` 只持久化 theme、expanded Node IDs 与 focused Node ID；拒绝损坏或过大的存储文档并显式回退。
- `ProfileEditor.tsx` 复用 Login/Settings 的完整 Profile 字段。
- `Settings.tsx`、`Inspector.tsx` 分别承载完整设置与 transient preview。
- `App.tsx` 统一 content mode、Profile/connection lifecycle、View dirty guard 和刷新后的 selection validity。
- `style.css` 用浅/深 token 重写连续分栏、紧凑 tree、settings、inspector、workspace 和 login；large rows/widgets 使用 `content-visibility`。
- 没有新增 npm 依赖，没有更改 Go API 或持久化 schema。

## Migration Boundary

- Profile settings、credentials 与 View documents 不迁移，旧数据继续由现有 Go stores 读取。
- UI preferences 是新建的 Profile-scoped WebView local storage 文档；无记录时采用浅色和两层默认展开。
- Preference 只包含非秘密 display state，不包含 Permit、公钥、Resource payload 或凭据。
- topology/catalog 当前仍为完整响应；首期只优化前端索引和离屏绘制。服务端 pagination/lazy loading 作为 TREE02 延后。

## Verification

- `npm test`: 4 test files，19 tests passed。
- Tree fixtures: 3,000 层无递归溢出；2,000 Nodes + 10,000 Resources build/filter 保持既有 `<750ms` contract。
- `npm run build`: TypeScript 与 Vite production build passed。
- `GOWORK=off go test ./apps/desktop/...`: passed。
- `GOWORK=off wails build -clean -platform windows/amd64`: Wails 2.11 Windows/amd64 production package passed。
- `git diff --check`: passed。
- 1440×900 页面实操验证：light workspace、deep tree、Node/Resource inspector、focused breadcrumb/back、full settings、dark theme、login。
- 生产 Wails executable 可启动并暴露唯一 `MyFlowHub` window；未创建测试 Profile，也未修改真实连接数据。

## Test-driven Follow-ups

`$m-continue` 根据验证结果修复了：

- empty Workspace 在 12 列 grid 中未真正居中；
- 深层 breadcrumb 末端不可见；
- 一个 Resource 命中会带出同 Node 的无关 Resources；
- 平台刷新后已消失的 selection 仍可能留在 Inspector。

## Rollback

- Runtime rollback 可整体恢复旧 `frontend/src` shell、components、store、tests、style 与 tracked `dist`；Go data 不需要回滚或迁移。
- UI preference key 为 `mfh.desktop.ui.v1:<profile-id>`，旧客户端忽略该 key；清除它只会恢复浅色和默认树状态。
- 不涉及 remote、release、publish 或项目图标资产。
