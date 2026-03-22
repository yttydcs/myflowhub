# 2026-03-21 Win 设置页面

## 变更背景 / 目标

- Win 端此前没有独立的应用设置页，默认地址、默认设备 ID、自动连接、自动登录等配置分散在 `Home` 页中，页面职责混杂。
- 本次目标是新增独立 `Settings` 页面，在侧边栏提供一级入口，并补齐界面偏好与 About 信息展示能力。

## 具体变更内容（新增 / 修改 / 删除）

### 新增

- `app_settings.go`
  - 新增 `App.SettingsState()` / `App.SaveSettingsState()` / `App.ResetSettingsState()` / `App.AboutState()`
  - 使用单独的 `app.settings` profile-scoped 配置模型保存：
    - 默认地址
    - 默认设备 ID
    - 自动连接
    - 自动登录
    - 默认启动页
    - 密度模式
    - 减少动效
  - About 信息通过 Go runtime + `debug.ReadBuildInfo()` + profile state 汇总生成。
- `app_settings_test.go`
  - 补充设置默认值、旧 `HomeState` 回退、保存归一化、重置默认、About 字段、`HomeState` 不改写设置的测试。
- `frontend/src/stores/appSettings.ts`
  - 新增前端设置 store，负责：
    - 读取 / 保存 / 重置设置
    - 读取 About
    - 将默认启动页 / 密度 / 减少动效镜像到 `localStorage`
    - 启动时和加载后应用全局 UI 偏好
- `frontend/src/pages/Settings.vue`
  - 新增设置页表单与 About 展示区。

### 修改

- `frontend/src/router/index.ts`
  - 新增 `/settings` 路由。
  - 根路径 `/` 改为根据本地启动偏好镜像决定跳转页。
- `frontend/src/layout/AppShell.vue`
  - 侧边栏和移动端导航新增 `Settings` 一级菜单。
  - 在 profile 切换时全局加载应用设置。
  - Shell 的 header / sidebar / main spacing 改为通过 CSS 变量响应密度设置。
- `frontend/src/main.ts`
  - 在 Vue 挂载前应用启动镜像中的 UI 偏好，降低首次渲染闪烁。
- `frontend/src/style.css`
  - 新增 shell / panel 密度变量。
  - 新增 `data-ui-density` 与 `data-ui-motion` 的全局样式响应。
- `frontend/src/pages/Home.vue`
  - `Home` 保留连接、登录、清除 auth 等运行时操作。
  - 默认地址与自动连接 / 自动登录行为改为从 `Settings` 读取。
  - 移除在 `Home` 中直接修改持久化设置的交互，避免设置职责继续分散。

### 未做

- 未引入主题切换、多语言、字号自定义。
- 未引入独立设置窗口。

## 对应 plan.md 任务映射

- `T1` Go：新增设置与 About API，并补齐持久化默认值
  - `app_settings.go`
  - `app_settings_test.go`
  - `app_home.go`
- `T2` Frontend Infra：新增设置 store，接入启动页与全局 UI 偏好
  - `frontend/src/stores/appSettings.ts`
  - `frontend/src/router/index.ts`
  - `frontend/src/main.ts`
  - `frontend/src/style.css`
- `T3` Frontend Page：新增 Settings 页面并调整 Home / 导航接入
  - `frontend/src/pages/Settings.vue`
  - `frontend/src/layout/AppShell.vue`
  - `frontend/src/pages/Home.vue`
  - `frontend/src/router/index.ts`
- `T4` Verification：测试、Review 与归档
  - 本文档
  - `plan.md` Review 记录

## 关键设计决策与权衡（尤其性能 / 扩展性）

- 设置真源使用独立的 `app.settings` 单 key JSON，而不是继续把字段散落在多个独立 key 或继续扩展 `HomeState`。
  - 原因：减少保存时的多次 I/O，后续增加设置项时不必继续污染 `Home` 域模型。
- `HomeState` 与 `Settings` 不做双向自动同步。
  - 原因：用户已确认“设置点击保存后生效”；如果登录成功后自动改写默认设备 ID，会破坏设置页的显式保存语义。
  - 权衡：`Home` 仍保留 auth 快照语义；默认值由设置页提供，运行态由 `Home` 自己维护。
- 默认启动页、密度、减少动效额外镜像到 `localStorage`。
  - 原因：路由初始化和首次渲染需要同步读取，Wails Go 绑定是异步的；本地镜像仅用于启动优化，不作为真源。
- 紧凑模式优先作用于 shell spacing 和关键面板 padding，而不是一次性重写所有页面 class。
  - 原因：本次变更最小化、风险可控；后续如果要全局更强的密度控制，再逐步把常用容器抽象成统一 panel 组件。

## 测试与验证方式 / 结果

### 已执行

- Go：
  - `GOWORK=off go test ./... -count=1`
  - 结果：通过
- Wails 绑定生成：
  - `GOWORK=off wails generate module`
  - 结果：通过
- Frontend：
  - `npm install`
  - `npm run build`
  - 结果：通过

### 冒烟 / UI 验证

- 尝试使用 `chrome-devtools` 做页面级冒烟。
- 结果：未完成。
  - 原因：本机已有占用中的 `chrome-devtools-mcp` browser profile，当前会话无法附着新的浏览器上下文。
  - 影响：本次已完成构建级验证，但缺少浏览器内 DOM 冒烟记录。

## 潜在影响与回滚方案

### 潜在影响

- `Home` 页不再直接保存自动连接 / 自动登录等持久化设置，用户需要到 `Settings` 页显式保存默认值。
- 前端产物仍有一个较大的 chunk（Vite build 提示 >500 kB），本次未处理代码分包。

### 回滚方案

- 代码回滚：
  - 回退 `app_settings.go` / `app_settings_test.go`
  - 回退 `frontend/src/stores/appSettings.ts`
  - 回退 `frontend/src/pages/Settings.vue`
  - 回退路由、AppShell、Home、style 相关接线
- 数据回滚：
  - `settings.json` 中新增的 `app.settings` key 可忽略；旧版本不会依赖它。
- 行为回滚：
  - 回退后 `Home` 恢复为原有的默认配置管理入口。

## 子Agent 执行轨迹

- 本次未使用子Agent。
- 原因：
  - 当前运行时规则要求只有在用户显式授权委派 / 子Agent 时才可调用子Agent。
  - 因此本次所有 Task ID 均由主Agent 在当前 worktree 中完成。
- 审计结果：
  - `T1 -> 主Agent -> D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-page -> app_settings.go / app_settings_test.go / app_home.go -> 通过`
  - `T2 -> 主Agent -> D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-page -> frontend/src/stores/appSettings.ts / router / main / style -> 通过`
  - `T3 -> 主Agent -> D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-page -> frontend/src/pages/Settings.vue / AppShell.vue / Home.vue -> 通过`
  - `T4 -> 主Agent -> D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-settings-page -> plan.md / docs/change -> 通过`
