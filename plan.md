# Plan - MyFlowHub3 全局计划与归档

> 说明：
> - 本文件最初用于记录 `MyFlowHub-Win` 的 Fyne → Wails 迁移计划（下方 “Task Checklist”，现已全部 Completed）。
> - 随着 PR1/PR2/PR19 等推进，本文件也用于记录 MyFlowHub3 多仓“彻底重构”的 workflow archives（见文末 “Workflow Archives”）。
> - 最新架构/仓库职责以 `target.md` / `repos.md` 为准；本文件中与之冲突的旧描述会标注为 Legacy 或在归档中注明。

## Current Status (Global)
- 协议字典：以 `MyFlowHub-Proto`（`github.com/yttydcs/myflowhub-proto/protocol/*`）为准；`MyFlowHub-Server/protocol/*` 仅为兼容壳（alias 到 Proto）。
- 客户端能力：`MyFlowHub-SDK` 统一 session/transport + awaiter；`MyFlowHub-Win` 为上层应用，尽量只“调用 SDK”，不重复实现协议机制。
- 变体产品化：PR2-4（minimal/full）按当前决策暂缓（Deferred）。

## Legacy Project Goal (Win: Fyne → Wails)
Refactor MyFlowHub-Win from Fyne to Wails (Vue 3 + TS + Vite + Tailwind + shadcn-vue), keep all existing features, redesign the UI with a tool-console layout, improve usability, and support Windows and Linux single-file builds. Prefer reusing MyFlowHub-Core and MyFlowHub-Server packages to avoid duplicate logic. (Legacy constraint: initially avoided modifying Core/Server except for exporting protocol types; later PRs extracted protocols to `MyFlowHub-Proto` and refactored Core/Server, so this constraint is no longer applicable.)

## Legacy Current Status
- Existing app: Go + Fyne in `MyFlowHub-Win` with full feature set.
- Dependencies: `github.com/yttydcs/myflowhub-core` (via local replace).
- MyFlowHub-Server exists at `../MyFlowHub-Server` and is allowed to be imported as a Go module (via replace).
- Wails scaffold + frontend 已落地（Vue 3 + TS + Vite + Tailwind + shadcn-vue），Fyne 迁移正在收敛中（目标：彻底移除 Fyne）。
- Preferences and profiles stored via Fyne preferences with keys in `MyFlowHub-Win/internal/ui/helpers.go`.
- (Historical) Protocol types 曾位于 `internal/handler/*`（不可被其它模块 import）；现已抽离到 `MyFlowHub-Proto`，并由 `MyFlowHub-Server/protocol/*` 以兼容壳形式保留旧 import path。

## Reuse Strategy (Core/Proto/SDK)
- Prefer MyFlowHub-Core for protocol headers, config keys, event bus, and shared domain types.
- Protocol definitions: prefer `github.com/yttydcs/myflowhub-proto/protocol/*` as the canonical source; `github.com/yttydcs/myflowhub-server/protocol/*` exists as a compatibility shim only.
- Wrap server/core logic in Wails-facing services rather than re-implementing serialization or validation.
- If a server package is tightly coupled to server runtime, create thin adapters in MyFlowHub-Win and document the limitation.

## Task Checklist

### T1 - Inventory and protocol mapping
- Goal: Document feature-to-protocol mapping and data flows to keep parity, and identify reusable core/server packages.
- Status: Completed
- Scope / files:
  - Read existing feature modules in `MyFlowHub-Win/internal/ui/*.go` and `MyFlowHub-Win/internal/session/session.go`.
  - Capture preference keys in `MyFlowHub-Win/internal/ui/helpers.go`.
  - Inspect `MyFlowHub-Core` and `MyFlowHub-Server` for reusable protocol, model, and validation packages.
- Acceptance:
  - Written mapping of each feature to SubProto/action payload and key fields.
  - List of preference keys and profile behavior.
  - List of core/server packages to import and a per-feature reuse map.
- Tests:
  - Manual verification by cross-checking current UI behavior and protocol code paths.
- Rollback:
  - No code changes; only notes.

### T1a - Extract public protocol packages in MyFlowHub-Server
- Goal: Create `protocol/*` public packages that expose request/response models and action constants.
- Status: Completed
- Scope / files:
  - New `MyFlowHub-Server/protocol/common` (shared message envelope if needed).
  - New `MyFlowHub-Server/protocol/auth`, `protocol/varstore`, `protocol/topicbus`, `protocol/file`, `protocol/flow`, `protocol/management`.
  - Update server handlers to use the new protocol packages (no behavior change).
- Acceptance:
  - MyFlowHub-Server builds with `protocol/*` exported types/consts.
  - No duplicate model definitions in Win for the same protocol.
- Tests:
  - Existing server tests compile and pass (or at least build).
- Rollback:
  - Revert protocol packages and handler imports.

### T2 - Wails scaffold and base structure
- Goal: Create Wails app skeleton within `MyFlowHub-Win` and define project layout.
- Status: Completed
- Scope / files:
  - New Wails entry (likely `MyFlowHub-Win/main.go`, `MyFlowHub-Win/app.go`, `MyFlowHub-Win/wails.json`).
  - New frontend at `MyFlowHub-Win/frontend/` (Vue 3 + TS + Vite + Tailwind + shadcn-vue).
  - Keep existing Go module `MyFlowHub-Win/go.mod`, add `replace` for MyFlowHub-Server as needed.
- Acceptance:
  - `wails dev` boots with a basic window.
  - Frontend builds with shadcn-vue installed and theme tokens set.
- Tests:
  - `wails dev` (manual) and a smoke page render.
- Rollback:
  - Remove Wails files and frontend folder.

### T3 - Backend services and event model
- Goal: Move existing Fyne-bound logic into Wails service layer with clear APIs.
- Status: Completed
- Scope / files:
  - New packages: `MyFlowHub-Win/internal/services/*` (session, auth, varpool, topicbus, file, flow, management, debug, logs).
  - Prefer protocol usage and shared logic from MyFlowHub-Core, `myflowhub-proto/protocol/*`, and MyFlowHub-SDK where available.
- Acceptance:
  - Public Wails bindings for all operations and data queries.
  - Event stream for logs and live updates.
  - No duplicate serialization/validation when equivalent core/server helpers exist.
- Tests:
  - Unit tests for header/payload build helpers and parsing.
- Rollback:
  - Revert new service packages and bindings.

### T4 - Storage and migration (profiles, preferences, keys)
- Goal: Implement profile-aware storage in user config dir, and migrate Fyne prefs.
- Status: Completed
- Scope / files:
  - New `MyFlowHub-Win/internal/storage/*` for read/write and migration.
  - Detect and import Fyne prefs for app id `myflowhub.debugclient`.
  - Support keys listed in `MyFlowHub-Win/internal/ui/helpers.go`.
- Acceptance:
  - Profiles list and last profile are preserved.
  - Node keys migrate from `config/node_keys*.json` into profile storage.
  - Old data can still be read if present.
- Tests:
  - Manual: start app with existing prefs, verify values loaded.
  - Unit: migration path selection on Windows and Linux.
- Rollback:
  - Disable migration logic and keep new storage only.

### T5 - Frontend shell (tool-console layout)
- Goal: Build main layout and navigation (left nav, top status bar, main content).
- Status: Completed
- Scope / files:
  - `MyFlowHub-Win/frontend/src/layout/*`
  - `MyFlowHub-Win/frontend/src/router/*`
  - `MyFlowHub-Win/frontend/src/components/*`
- Acceptance:
  - All modules appear in nav.
  - Top bar shows connection state and profile selector.
- Tests:
  - Manual navigation across pages.
- Rollback:
  - Revert frontend layout files.

### T5-1 - Layout UX adjustments
- Goal: Remove non-Home "Current Module" card, separate sidebar/content scroll, refresh header connection badge, and align profile dropdown styling.
- Status: Completed
- Scope / files:
  - `frontend/src/layout/AppShell.vue`
  - `frontend/src/style.css` (if new tokens/classes are required)
- Acceptance:
  - "Current Module" card only appears on Home page.
  - Sidebar and main content scroll independently with header fixed.
  - Header connection indicator shows icon + "Connected to {addr}" in rounded gray pill; disconnected shows "Disconnected / Last error".
  - Profile selector uses a custom dropdown menu consistent with the UI theme.
- Tests:
  - Manual: resize window, verify independent scroll regions and dropdown styling.
- Rollback:
  - Revert AppShell layout changes.

### T5-2 - Home spacing fix
- Goal: Restore spacing between the Home module card and the Home content sections, and remove the "Current Module" title plus subtitle text from that card.
- Status: Completed
- Scope / files:
  - `frontend/src/layout/AppShell.vue`
- Acceptance:
  - Home page shows clear spacing between the "Current Module" card and the Home content sections.
  - The Home module card no longer shows the "Current Module" title or the subtitle text.
- Tests:
  - Manual: visually confirm spacing on Home without affecting other routes.
- Rollback:
  - Revert the spacing class changes.

### T6 - Home/Auth module
- Goal: Port connect/login/register and status UI.
- Status: Completed
- Scope / files:
  - Backend: `internal/services/session`, `internal/services/auth`
  - Frontend: `frontend/src/pages/Home.vue`, shared form components.
- Acceptance:
  - Connect/disconnect, auto-connect/auto-login, device id, node/hub/role display.
  - Login/register payload parity with existing logic.
- Tests:
  - Manual connect/login against server.
- Rollback:
  - Revert module page and backend methods.

### T7 - VarPool module
- Goal: Port list/get/set/revoke/subscribe workflows and cached view.
- Status: Completed
- Scope / files:
  - Backend: `internal/services/varpool`
  - Frontend: `frontend/src/pages/VarPool.vue`
- Acceptance:
  - Same actions and data fields as existing Fyne UI.
  - Profile-aware storage of watch list and subscriptions.
- Tests:
  - Manual: list, get, set, revoke, subscribe/unsubscribe.
- Rollback:
  - Revert module page and service methods.

### T7-1 - VarPool Edit dialog parity fix
- Goal: Restore Fyne-style edit dialog for both "my variables" and "watched variables", and auto-refresh after save.
- Status: Completed
- Scope / files:
  - Frontend: `frontend/src/pages/VarPool.vue`
  - Frontend: `frontend/src/stores/varpool.ts` (refresh behavior)
- Acceptance:
  - Clicking Edit opens a modal that shows variable name (read-only) and allows editing value + visibility.
  - Works for both "my variables" and "watched variables".
  - Save triggers set + refresh (get) to update cached view.
- Tests:
  - Manual: open Edit, change value/visibility, save, verify UI updates after refresh.
- Rollback:
  - Revert dialog UI and restore inline edit behavior.

### T7-2 - VarPool add dialogs
- Goal: Replace inline "add" sections with modal dialogs in VarPool list sections.
- Status: Completed
- Scope / files:
  - `frontend/src/pages/VarPool.vue`
- Acceptance:
  - VarPool page shows lists by default without inline add forms.
  - "My Variables" section has an "Add" action that opens a dialog (name/value/visibility/type, owner auto from NodeID).
  - "Watched Variables" section has an "Add Watch" action that opens a dialog (name + owner).
  - Save triggers set/get (my variables) or add-watch/get (watched variables) to refresh UI.
- Tests:
  - Manual: add variable/watch via dialogs and verify list updates.
- Rollback:
  - Revert dialog changes and restore inline forms.

### T8 - TopicBus module
- Goal: Port subscribe/publish/filters and event stream UI.
- Status: Completed
- Scope / files:
  - Backend: `internal/services/topicbus`
  - Frontend: `frontend/src/pages/TopicBus.vue`
- Acceptance:
  - Batch subscribe, event list with detail, filters, max events.
- Tests:
  - Manual: publish and verify event display.
- Rollback:
  - Revert module page and service methods.

### T9 - File transfer and file browser
- Goal: Port file browser, preview, upload/download, and transfer tasks (file browser is the primary UI with a button to open the task list window).
- Status: Completed
- Scope / files:
  - Backend: `internal/services/file` plus new transfer manager/state (events, sessions, tasks).
  - Storage/App: profile-aware file prefs and file-browser node list.
  - Frontend: `frontend/src/pages/File.vue`, `frontend/src/stores/file.ts`, `frontend/src/windows/FileTasks.vue`.
- Acceptance:
  - File browser is the main page; task list opens via a button as a separate window.
  - Remote list and text preview work (local + remote), download/upload support pull/offer with resume and optional sha256.
  - Incoming offers show a confirm dialog; auto-accept toggle exists (off by default).
  - File settings have a visual entry point (base_dir, max size, chunk, concurrency, ttl, want_sha256, auto_accept).
- Tests:
  - Manual: list/preview, pull/offer (incl. resume), reject/accept offer, task window updates.
- Rollback:
  - Revert module page, window, store, and file service changes.

### T9-1 - File browser main UI
- Goal: Implement the Wails file browser page with node tree, directory list, preview entry, and task window launcher.
- Status: Completed
- Scope / files:
  - Frontend: `frontend/src/pages/File.vue`, `frontend/src/stores/file.ts`.
  - Routing: `frontend/src/router/index.ts`.
- Acceptance:
  - Node tree shows local + saved remote nodes; directory list refreshes; preview opens on file double-click.
  - Task window launcher opens the task list as a separate window.
- Tests:
  - Manual: navigate directories, open preview, open/close task window.
- Rollback:
  - Revert the File page, store, and route changes.

### T9-2 - File settings and auto-accept toggle
- Goal: Add a visual settings entry and persist file-related preferences per profile.
- Status: Completed
- Scope / files:
  - Backend/App storage: new file preferences APIs and keys.
  - Frontend: settings panel or dialog in `frontend/src/pages/File.vue`.
- Acceptance:
  - Settings UI updates base_dir, max size, chunk bytes, concurrency, ttl, want_sha256, and auto_accept.
  - Auto-accept only changes behavior when enabled; default is manual confirm.
- Tests:
  - Manual: toggle auto-accept and verify incoming offers honor the setting.
- Rollback:
  - Revert settings UI and storage changes.

### T9-3 - Transfer manager and task events
- Goal: Port file transfer sessions, task tracking, and events from the Fyne implementation into the Wails backend.
- Status: Completed
- Scope / files:
  - Backend: new file transfer manager/state, event emission, and Wails bindings.
  - Frontend: `frontend/src/windows/FileTasks.vue` consuming task events.
- Acceptance:
  - Tasks show status/progress, retry/cancel, and open-folder actions.
  - Sessions clean up on disconnect; .part cleanup follows TTL.
- Tests:
  - Manual: verify progress, retry/Cancel, and cleanup after disconnect.
- Rollback:
  - Revert transfer manager and task window changes.

### T10 - Flow module
- Goal: Port flow list/get/set/run/status and node/edge editor.
- Status: Completed
- Scope / files:
  - Backend: `internal/services/flow`
  - Frontend: `frontend/src/pages/Flow.vue`
- Acceptance:
  - Flow CRUD and status UI parity.
- Tests:
  - Manual: create flow, run, check status.
- Rollback:
  - Revert module page and service methods.

### T11 - Management module
- Goal: Port node list/subtree and config list/get/set.
- Status: Completed
- Scope / files:
  - Backend: `internal/services/management`
  - Frontend: `frontend/src/pages/Management.vue`
- Acceptance:
  - Node list and config editor working.
- Tests:
  - Manual: list nodes and read config values.
- Rollback:
  - Revert module page and service methods.

### T12 - Debug and Presets
- Goal: Port custom header/payload sender and preset actions.
- Status: Completed
- Scope / files:
  - Backend: `internal/services/debug`, `internal/services/presets`
  - Frontend: `frontend/src/pages/Debug.vue`, `frontend/src/pages/Presets.vue`
- Acceptance:
  - Header form and payload sender parity.
  - Preset actions match existing behaviors.
- Tests:
  - Manual: send sample preset and verify server response.
- Rollback:
  - Revert module page and service methods.

### T13 - Logs and separate log window
- Goal: Centralized logging with log window and controls (pause, hex, truncate).
- Status: Completed
- Scope / files:
  - Backend: `internal/services/logs`
  - Frontend: `frontend/src/pages/Logs.vue`, `frontend/src/windows/LogWindow.vue`
- Acceptance:
  - Log stream in main UI and separate log window.
  - Pause and formatting controls are functional.
- Tests:
  - Manual: flood logs and verify UI stays responsive.
- Rollback:
  - Revert log pages and event wiring.

### T13-1 - Log item list + per-item expand/hex/format
- Goal: Replace the single log text block with per-item cards, each with expand/collapse, per-item hex toggle, and JSON format toggle (session-only).
- Status: Completed
- Scope / files:
  - Backend: `internal/services/logs` (ensure payload length is exposed per item if needed).
  - Frontend: `frontend/src/pages/Logs.vue`, `frontend/src/windows/LogWindow.vue`.
  - Frontend: new log item component (if needed) in `frontend/src/components/logs/`.
  - Frontend: `frontend/src/lib/logs.ts` (format helpers) and `frontend/src/stores/logs.ts` (local UI state if needed).
- Acceptance:
  - Logs and LogWindow render each log as an item with time/level/message/payload length and a one-line preview (truncated with ellipsis).
  - Each log item can expand to show full content and payload detail (text + hex).
  - Each log item can toggle hex independently; toggling does not affect other items.
  - Each log item provides a JSON format toggle with failure message on invalid JSON.
  - Global Pause remains functional; per-item states are session-only.
- Tests:
  - Manual: stream logs, expand/collapse items, toggle hex and JSON format per item, verify independent behavior.
- Rollback:
  - Revert log item components and restore flat log view.

## Dependencies
- T1 precedes all.
- T1a depends on T1.
- T2 precedes all code changes.
- T3 and T4 precede frontend modules.
- T5 precedes all module pages.
- T6-T13 can proceed in parallel after T3-T5.

## Risks and Notes
- Migration paths for Fyne prefs are OS-specific; implement guarded fallback.
- (Legacy) Keep MyFlowHub-Core unchanged. MyFlowHub-Server changes limited to public `protocol/*` exports only (no behavior change).
- High-volume events (logs/topicbus) must be throttled to protect UI.

## Workflow Archives

### 2026-03-11 - SubProto：修复 Flow/Exec 响应继承 MsgID/TraceID（解除 Win Flow Refresh 超时）
- Repo：`repo/MyFlowHub-SubProto`（已合并到 `main`：SubProto @ `11901c8`）
- Branch: `fix/resp-msgid`
- 背景：Win Flow `Refresh` 报 `flow list: request timed out`；根因是服务端 `*_resp` 回包头 `msg_id=0` 导致 SDK await（按 `MsgID+SubProto+Action`）无法匹配。
- 变更：
  - Flow：`set_resp/run_resp/status_resp/list_resp/get_resp` 回包继承请求 `MsgID/TraceID`。
  - Exec：`call_resp` 回包继承请求 `MsgID/TraceID`。
  - 单测：新增断言覆盖回包头字段继承。
- 验收：
  - `cd repo/MyFlowHub-SubProto/flow && go test ./... -count=1 -p 1`
  - `cd repo/MyFlowHub-SubProto/exec && go test ./... -count=1 -p 1`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-11_subproto-resp-msgid.md`
- 归档（全局）：`docs/change/2026-03-11_resp-msgid.md`

### 2026-03-08 - Auth：修复多 hop 路由索引缺失（避免 sourceMismatch 丢帧）
- Repo：
  - `repo/MyFlowHub-SubProto`（已合并到 `main`：SubProto @ `de9a210`，已发布 tag：`auth/v0.1.2`）
  - `repo/MyFlowHub-Server`（已合并到 `main`：Server @ `7c27ea7`）
- Branch: `fix/auth-route-index-heal`
- 背景：多 hop 场景 Root Hub 出现 `drop frame due to source mismatch (hdr_source=11 meta_node=9)`，导致 VarStore `list/get` 返回 `not found (code=4)`。
- 关键修复（较大改动）：
  - Auth：`register` 缺省 `pubkey` 不再填本机公钥，避免 trusted/binding 公钥“毒化”。
  - Auth：`up_login` sender trusted 验签失败时允许用 `sender_pub` 受限自愈（约束 `sender_id == hdr.SourceID == conn.meta(nodeID)`），自愈后建立 `nodeIndex[descendant] -> childConn`。
  - Auth：`auth.disable_persist=true` 不读写 `config/trusted_nodes.json`（避免落盘副作用）。
  - Server：升级依赖 `myflowhub-subproto/auth` 至 `v0.1.2`，并同步更新 `docs/2-auth.md`。
- 验收：
  - SubProto/auth：`cd auth; GOWORK=off go test ./... -count=1 -p 1`
  - Server：`GOWORK=off go test ./... -count=1 -p 1`
- 原 plan（归档）：
  - `docs/plan_archive/plan_archive_2026-03-08_subproto-auth-route-index-heal.md`
  - `docs/plan_archive/plan_archive_2026-03-08_server-auth-route-index-heal.md`
- 归档（全局）：
  - `docs/change/2026-03-08_auth-route-index-heal.md`
  - `docs/change/2026-03-08_auth-route-index-heal-server.md`

### 2026-03-07 - MetricsNode：修复 CI 构建失败（Windows embed dist + gomobile tidy）
- Repo: `repo/MyFlowHub-MetricsNode`（已合并到 `main`：MetricsNode @ `92f4186`）
- Branch: `fix/metricsnode-ci`
- 目标：修复 GitHub Actions `ci` 在 clean checkout 场景下的构建失败，恢复 `debug-latest` 发布链路。
- 变更：
  - CI（Windows）：在 `wails build` 前创建 `windows/frontend/dist/.keep`，避免 `//go:embed all:frontend/dist` 因空目录报错。
  - Android：提交 `nodemobile` 子模块 `go mod tidy` 结果，对齐间接依赖（`myflowhub-sdk v0.1.2`），确保 `gomobile bind` 不再提示只读 tidy。
  - 脚本：`scripts/build-windows.ps1` 同步补齐 `dist/.keep` 预创建逻辑，避免本地 clean checkout 构建失败。
- 验收：
  - Windows：`wails build -platform windows/amd64 -nopackage` 可产出 `windows.exe`
  - Android：`bash scripts/build_aar.sh` 可产出 `myflowhub.aar`；`./gradlew :app:assembleDebug` 可产出 debug APK
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-07_metricsnode-ci-fix.md`
- 归档（全局）：`docs/change/2026-03-07_metricsnode-ci-fix.md`

### 2026-03-04 - Win：Showcase `canvas_percent` 自由布局 + Slider 值右置
- Repo: `repo/MyFlowHub-Win`（已合并到 `main`：Win @ `d0913ee`）
- Branch: `feat/showcase-canvas-layout`
- 目标：
  - 新增 Showcase 布局模式 `canvas_percent`：Widget 位置/大小按百分比保存（`xPct/yPct/wPct/hPct`），支持非等比缩小适配且 `scale<=1`。
  - Designer 支持拖拽移动/缩放（pointerup 保存）与右键置顶/置底。
  - Viewer Window 支持 `canvas_percent` 渲染；保留 `columns` 兼容。
  - Slider 值（Badge）在 Designer/Viewer 统一显示在滑块右侧，极窄时可退化换行。
- 验收：
  - `GOWORK=off go test ./... -count=1`
  - `GOWORK=off wails build -debug -skipembedcreate -nopackage`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-04_win-showcase-canvas-layout.md`
- 归档（全局）：`docs/change/2026-03-04_showcase-canvas-layout.md`

### 2026-03-04 - Win：File Console 首次进入报错修复（BaseDir 自动创建 + 相对路径稳定）
- Repo: `repo/MyFlowHub-Win`（已合并到 `main`：Win @ `d51ff7a`）
- Branch: `fix/win-file-console-base-dir`
- 目标：修复本地节点（nodeId=2）进入 `File Console` 立即报 `open file ... cannot find`；并将相对 `BaseDir=./file` 稳定解析到软件目录下，避免随 CWD 漂移导致文件散落。
- 变更：
  - 后端：相对 `BaseDir` 以 `os.Executable()` 的目录为基准解析（`exeDir/baseDir`）。
  - 后端：root list（`dir==""`）自动 `MkdirAll(BaseDir)`，首次进入不再透传系统错误。
  - 测试：补充 `resolveRuntimeBaseDir` 单测覆盖 CWD 变化不影响落点。
- 验收：
  - Go：`GOWORK=off go test ./... -count=1`
  - 手工：首次进入 `File Console` 不报错，软件目录下生成 `file/`。
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-04_win-file-console-base-dir.md`
- 归档（全局）：`docs/change/2026-03-04_win-file-console-base-dir.md`

### 2026-03-04 - SubProto(file)：Hub File Console root list 修复（BaseDir exeDir 解析 + 自动创建）
- Repo: `repo/MyFlowHub-SubProto`（已合并到 `main`：SubProto @ `74b98dc`）
- Branch: `fix/hub-file-console-base-dir`
- 目标：修复 Hub（node1）File Console root list 在默认 `file.base_dir=./file` 且目录不存在时返回 `not found`；并将相对 `file.base_dir` 稳定解析到 `hub_server.exe` 同目录下，避免随 CWD 漂移导致目录散落。
- 变更：
  - 配置：相对 `file.base_dir` 以 `os.Executable()` 的目录为基准解析（`exeDir/baseDir`）。
  - 行为：root list（`dir==""`）自动 `MkdirAll(BaseDir)`；创建/读取失败返回 `500`（并记录日志），不再伪装成 `404 not found`。
  - 测试：补充 BaseDir 解析与 root list 自动创建的单测覆盖。
  - 发布：推送 tag `file/v0.1.1`。
- 验收：`cd file; GOWORK=off go test ./... -count=1 -p 1`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-04_subproto-file-base-dir-exedir-root-list.md`
- 归档（全局）：`docs/change/2026-03-04_file-base-dir-exedir-root-list.md`

### 2026-03-04 - Server：升级 subproto/file 至 v0.1.1（修复 Hub File Console not found）
- Repo: `repo/MyFlowHub-Server`（已合并到 `main`：Server @ `81d9681`）
- Branch: `fix/hub-file-console-base-dir`
- 目标：升级 `myflowhub-subproto/file` 至 `v0.1.1`，使 Hub（node1）获得 root list 自动创建与 BaseDir exeDir 解析的修复。
- 变更：`go.mod/go.sum`：`github.com/yttydcs/myflowhub-subproto/file v0.1.0` → `v0.1.1`
- 验收：`GOWORK=off go test ./... -count=1 -p 1`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-04_server-bump-subproto-file-v0.1.1.md`
- 归档（全局）：`docs/change/2026-03-04_bump-subproto-file-v0.1.1.md`

### 2026-03-03 - SubProto：Management Nodes children-only（过滤 parent link）
- Repo: `repo/MyFlowHub-SubProto`（已合并到 `main`：SubProto @ `d512b41`）
- Branch: `fix/management-children-only`
- 目标：将 `management list_nodes/list_subtree` 语义调整为 children-only（仅返回下游 children，不包含上游 parent），避免 Devices/Nodes 树回指/环与 Duplicate 节点。
- 变更：
  - `management/action_nodes.go`：跳过 `role=parent` 的连接；direct nodes 不再输出 parent node。
  - `management/action_nodes_test.go`：增加最小单测覆盖（parent 不应出现在 nodes 列表）。
- 验收：`cd management; GOWORK=off go test ./... -count=1 -p 1`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-03_subproto-management-children-only.md`
- 归档（全局）：`docs/change/2026-03-03_management-nodes-children-only.md`

### 2026-03-03 - Proto：补充 Management Nodes children-only 语义说明
- Repo: `repo/MyFlowHub-Proto`（已合并到 `main`：Proto @ `e915e5f`）
- Branch: `docs/management-children-only`
- 目标：在协议文档中明确 `list_nodes/list_subtree` 的 children-only 语义（不改 wire / 不改生成区块），消除实现与 UI 使用歧义。
- 变更：
  - `docs/protocol_map.md`：在 Manual Notes 补充 Nodes 语义（children-only / list_subtree=direct+self / has_children 为 hint）。
- 验收：`GOWORK=off go test ./... -count=1 -p 1`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-03_proto-management-children-only.md`
- 归档（全局）：`docs/change/2026-03-03_management-nodes-children-only-spec.md`

### 2026-03-03 - Server：升级 subproto/management 至 v0.1.2（children-only）
- Repo: `repo/MyFlowHub-Server`（已合并到 `main`：Server @ `0f6f30d`）
- Branch: `chore/bump-management-v0.1.2`
- 目标：发布 `management/v0.1.2` 并升级 Server 依赖，确保 nodes 列表遵循 children-only 语义。
- 变更：
  - SubProto：推送 tag `management/v0.1.2`（SubProto @ `d512b41`）
  - Server：`go.mod/go.sum` 升级 `management v0.1.1` → `v0.1.2`
- 验收：`GOWORK=off go test ./... -count=1 -p 1`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-03_server-bump-management-v0.1.2.md`
- 归档（全局）：`docs/change/2026-03-03_server-bump-management-v0.1.2.md`

### 2026-03-03 - Android：升级 hubmobile management 至 v0.1.2（触发 debug-latest）
- Repo: `repo/MyFlowHub-Android`（已合并到 `main`：Android @ `6708fa2`）
- Branch: `chore/android-bump-management-v0.1.2`
- 目标：让 Android 构建确定性使用 `management v0.1.2`（children-only），并通过 `main` push 触发 Actions 更新 `debug-latest`。
- 变更：
  - `hubmobile/go.mod/go.sum`：`management v0.1.1` → `v0.1.2`
- 验收：
  - `cd hubmobile; GOWORK=off go test ./... -count=1 -p 1`
  - `.\gradlew.bat :app:assembleDebug`（具备本地 SDK 时）
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-03_android-bump-management-v0.1.2.md`
- 归档（全局）：`docs/change/2026-03-03_android-bump-management-v0.1.2.md`

### 2026-03-03 - Android：发布 Release v0.1.12
- Repo: `repo/MyFlowHub-Android`（tag：`v0.1.12`；commit：Android @ `6708fa2`）
- Branch: `chore/android-release-v0.1.12`
- 目标：推送 `v0.1.12` tag 触发 `release.yml`，发布签名 Release APK + AAR。
- 变更：
  - 发布 tag：`v0.1.12`
- 验收：
  - `git ls-remote --tags origin v0.1.12`
  - GitHub Actions `release` workflow 成功并上传 `app-release.apk`/`myflowhub.aar`/`build-info.txt`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-03_android-release-v0.1.12.md`
- 归档（全局）：`docs/change/2026-03-03_android-release-v0.1.12.md`

### 2026-03-03 - Android：TopicBus（对齐 Win）
- Repo: `repo/MyFlowHub-Android`（已合并到 `main`：Android @ `1afc9c8`；tag：`v0.1.11`）
- Branch: `feat/android-topicbus`
- 目标：Android 侧新增 TopicBus 整页，行为对齐 Win（订阅/退订/发布/事件流与详情；`maxEvents=500` 持久化；订阅列表本地持久化；事件流 200ms flush）。
- 变更：
  - Go（hubmobile）：TopicBus subscribe/unsubscribe（send+await）+ publish（fire-and-forget）；捕获 publish 广播帧 → ring buffer；Kotlin 轮询 `TopicBusEventsPull`。
  - Android：侧边栏新增 `TopicBus`；新增 TopicBus 页面与组件；TopicBus prefs（subs/maxEvents）持久化；Go 反射 Bridge 接入。
- 验收：
  - Go：`GOWORK=off go test ./... -count=1 -p 1`（`hubmobile/`）
  - Android：`./gradlew :app:assembleDebug`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-03_android-topicbus.md`
- 归档（全局）：`docs/change/2026-03-02_android-topicbus.md`

### 2026-03-03 - Win：Showcase 独立窗口 + Columns 布局
- Repo: `repo/MyFlowHub-Win`（已合并到 `main`：Win @ `6441c83`）
- Branch: `feat/win-showcase-layout`
- 目标：在现有 Showcase 基础上支持独立 Viewer 窗口、Columns 响应式布局（`maxColumns/minColumnWidth` + `colSpan`）、拖拽排序（DnD）与多窗口实时同步。
- 变更：
  - Go：ShowcaseConfig schema 增加 screen/widget layout；保存成功后广播 `showcase.config_changed`。
  - 前端：store 支持 layout normalize + fixed `screenId` viewer + 监听 config_changed 自动 reload；Designer 增加 Columns 配置 + `colSpan` + DnD；新增 `#/showcase-window?screenId=...` Viewer 页。
- 验收：
  - Go：`GOWORK=off go test ./... -count=1 -p 1`
  - Win 前端：`cd frontend; npm ci; GOWORK=off wails generate module; npm run build`
  - 手工：Designer 保存后，已打开 Viewer 窗口实时同步（排序/布局/增删 widgets）；`screenId` 不存在显示 “Screen not found” 并停止。
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-03_win-showcase-layout.md`
- 归档（全局）：`docs/change/2026-03-03_showcase-layout.md`

### 2026-03-03 - Win：VarPool 节点变量弹窗 + Watch 订阅偏好本地持久化
- Repo: `repo/MyFlowHub-Win`（已合并到 `main`：Win @ `5fd2b75`）
- Branch: `feat/varpool-vars-dialog`
- 目标：在 `#/varpool` 与 `#/devices` 增加节点变量弹窗（按 owner NodeID 列 public names，便捷 Add Watch）；并将 watched key 的订阅偏好（subscribe=true/false）按 profile 持久化，重连/重新登录自动恢复订阅（不依赖打开 VarPool 页）。
- 变更：
  - Go：新增 `varpool.sub_prefs` 持久化（`VarPoolSubPrefs/SaveVarPoolSubPrefs`）+ 规范化去重 + 单测覆盖。
  - 前端：varpool store 读写订阅偏好并在 session ready 时并发恢复订阅；新增 `NodeVarsDialog`，在 Devices/VarPool 两处接入入口。
- 验收：
  - Go：`GOWORK=off go test ./... -count=1 -p 1`
  - Win 前端：`cd frontend; npm ci; GOWORK=off wails generate module; npm run build`
  - 手工：Devices 打开 Vars→Add Watch；VarPool Subscribe 后重启/重连自动恢复订阅。
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-03_win-varpool-dialog-subprefs.md`
- 归档（全局）：`docs/change/2026-03-03_varpool-dialog-subprefs.md`

### 2026-03-03 - Win：Showcase 面板简化（仅 title + value）+ 设计页右键菜单
- Repo: `repo/MyFlowHub-Win`（已合并到 `main`：Win @ `6a5ca41`）
- Branch: `feat/showcase-panel-simplify`
- 目标：简化 Designer/Viewer widget 卡片为 title + value；Designer 将 `Edit/Remove` 迁移到右键菜单并保留拖拽手柄。
- 变更：
  - 前端：Designer + Viewer 移除元信息（Target/Span、Var(owner/name)、payload 预览、slider 参数文案等）；`topic_button` 仅保留 Send；slider 保留数值 Badge。
  - 前端：Designer 新增页面内轻量 ContextMenu（右键 Edit/Remove），支持 `Esc`/点击空白/scroll/resize 关闭，菜单位置钳制不出屏。
- 验收：
  - Go：`go test ./... -count=1`
  - Win：`GOWORK=off wails build -debug -skipembedcreate -nopackage`
  - 手工：右键菜单可用；拖拽排序不受影响；Viewer 与 Designer 展示一致。
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-03_win-showcase-panel-simplify.md`
- 归档（全局）：`docs/change/2026-03-03_showcase-panel-simplify.md`

### 2026-03-03 - Win：Showcase Widget 卡片 Key-Value 单行布局
- Repo: `repo/MyFlowHub-Win`（已合并到 `main`：Win @ `88b43f5`）
- Branch: `feat/showcase-kv-layout`
- 目标：Designer + Viewer 的 widget 卡片严格 Key-Value（key=title，value=展示/控制组件），尽量单行不换行；`display` 单行截断 + tooltip；`switch` 仅显示开关本体；`slider` 极窄时允许两行退化以保证可用。
- 变更：
  - 前端：Designer/Viewer 卡片 DOM 重排为两列 grid（左 title，右 value），并统一 value 渲染策略。
  - 前端：`display` 由多行 `<pre>` 改为单行截断文本，完整值通过 `title` tooltip 查看。
  - 前端：`switch` value 区仅显示 checkbox；`slider` value 区使用 `flex-wrap + min-width`，极窄时自然换行退化。
- 验收：
  - Go：`GOWORK=off go test ./... -count=1`
  - Win：`GOWORK=off wails build -debug -skipembedcreate -nopackage`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-03_win-showcase-kv-layout.md`
- 归档（全局）：`docs/change/2026-03-03_showcase-kv-layout.md`

### 2026-03-03 - Win：VarPool 刷新/订阅体验修复（RefreshAll 跳过 not found 汇总提示 + Subscribe 不覆盖已有值 + 订阅按钮状态保持）
- Repo: `repo/MyFlowHub-Win`（已合并到 `main`：Win @ `2e1ce2d`）
- Branch: `fix/varpool-refresh-subscribe-ui`
- 目标：修复 `#/varpool` 中 `Refresh All` 的 `not found (code=4)` 刷屏报错；修复 Subscribe 后值变 `-`；修复离开再进入页面订阅按钮状态回退。
- 变更：
  - 前端：`Refresh All` 逐项容错，跳过 `code=4` 并在结束时 toast 汇总一次（not found/failed 计数）。
  - 前端：订阅响应仅更新订阅状态与元信息，不写入 value；watchList reload 改为 prune 保留缓存/订阅标记，避免 UI 状态回退。
- 验收：
  - Go：`GOWORK=off go test ./... -count=1 -p 1`
  - Win 前端：`cd frontend; npm ci; GOWORK=off wails generate module; npm run build`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-03_win-varpool-refresh-subscribe-ui.md`
- 归档（全局）：`docs/change/2026-03-03_varpool-refresh-subscribe-ui.md`

### 2026-03-03 - MetricsNode：两页 UI + 指标设置页（Windows + Android）
- Repo: `repo/MyFlowHub-MetricsNode`（已合并到 `main`：MetricsNode @ `a8c99fe`）
- Branch: `feat/metricsnode-settings-ui`
- 目标：将 MetricsNode 改为“连接页 + 设置页”两页 UI；设置页声明式配置指标（enabled / writable / var_name），并通过 devices config（runtime config）持久化与远程编辑。
- 变更：
  - core：新增 canonical key `metrics.settings_json`（兼容 `metrics.bindings_json` 双向互转），并支持“全关”（bindings 为空）。
  - core：按 `writable=false` 拦截 VarStore 下行写入；不执行控制动作并纠偏回真实值。
  - Windows：collectors 在 `enabled=false` 时真正停止采集/轮询（阻塞等待 config change）；Wails UI 拆为 Connect/Settings 两页，列表实时显示与配置保存。
  - Android：Compose UI 拆为 Connect/Settings；`NodeService` 按 settings 启停采集/执行；新增 flashlight（读写，不可读上报 `-1`）；StopAll 一键停止上报并退出前台服务。
  - build：修复 `scripts/build_aar.ps1` 对 gomobile 失败的检测。
- 验收：
  - Go：`GOWORK=off go test ./... -count=1 -p 1`
  - Windows：`cd windows/frontend; npm ci; npm run build` 后 `cd ..; GOWORK=off go test ./... -count=1 -p 1`
  - Android：设置 `ANDROID_HOME/ANDROID_SDK_ROOT` 后 `cd android; ./gradlew :app:assembleDebug`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-03_metricsnode-settings-ui.md`
- 归档（全局）：`docs/change/2026-03-03_metricsnode-settings-ui.md`
- 归档（全局）：`docs/change/2026-03-03_metricsnode-p0-metrics-flashlight.md`

### 2026-03-03 - MetricsNode：Windows UI 打磨（Connect/Settings）
- Repo: `repo/MyFlowHub-MetricsNode`（已合并到 `main`：MetricsNode @ `2bf9a1f`）
- Branch: `fix/metricsnode-ui-polish`
- 目标：Settings 页标题区对齐；Enabled/Writable 置于最右并改为开关样式；Connect 页恢复分块卡片并移除 metrics dump（调试信息）。
- 变更：
  - Windows：Settings 标题区 `Metrics Settings` 与 `Reload/Ready` 对齐；表格列顺序改为 `Metric | Var Name | Value | Enabled | Writable`；Enabled/Writable 改为 CSS 开关样式。
  - Windows：Connect 页拆为 `Bootstrap / Auth / Reporting` 三卡片，降低信息密度；移除 metrics dump/调试信息。
- 验收：
  - Win 前端：`cd windows/frontend; npm ci; npm run build`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-03_metricsnode-ui-polish.md`
- 归档（全局）：`docs/change/2026-03-03_metricsnode-ui-polish.md`

### 2026-03-03 - MetricsNode：CI 自动构建（Windows EXE + Android APK）+ debug-latest 直链
- Repo: `repo/MyFlowHub-MetricsNode`（已合并到 `main`：MetricsNode @ `a12b7b8`）
- Branch: `chore/metricsnode-ci-build`
- 目标：新增 GitHub Actions 自动构建 Windows（Wails）EXE 与 Android Debug APK，并在 main push 后发布/更新 `debug-latest` release 提供直链下载。
- 变更：
  - 新增 `.github/workflows/ci.yml`：Windows/Android 构建 + `debug-latest` 发布 job。
  - Release：assets 包含 `windows.exe` / `app-debug.apk` / `myflowhub.aar`（`--clobber` 覆盖）。
- 验收：
  - push 到 `main` 后查看 Actions：构建成功且 artifacts 存在；`debug-latest` release 可下载。
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-03_metricsnode-ci-build.md`
- 归档（全局）：`docs/change/2026-03-03_metricsnode-ci-build.md`

### 2026-03-02 - Win：展示界面（Showcase Screen）
- Repo: `repo/MyFlowHub-Win`（已合并到 `main`：Win @ `7ccbcc1`）
- Branch: `feat/showcase-screen`
- 目标：新增“展示界面”页面，允许用户创建多个 Screen，并通过 Widget 发送 TopicBus 事件/控制 VarStore 变量。
- 变更：
  - Go：新增 `ShowcaseConfig/SaveShowcaseConfig`（profile-scoped 存储）+ 配置规范化 + 单测覆盖（含 `throttleMs=0` 保留）。
  - 前端：新增 `/showcase` 页面与 store；支持 TopicBus Button、Var Display/Slider/Switch；进入/离开页面自动 Subscribe/Unsubscribe 去重。
  - 交互：slider 支持 `throttleMs=0`（不节流，每次 input 都发送）并在 UI 明确提示拥塞风险；松手/change 使用 `SetSimple(await)` 确认最终值。
- 验收：
  - Go：`GOWORK=off go test ./... -count=1 -p 1`
  - 前端：`cd frontend; npm ci; GOWORK=off wails generate module; npm run build`
  - 手工：创建 ≥2 个 Screen；添加 event/slider/switch；切换 Screen 不重复订阅；`throttleMs=0` 拖动不冻结且松手后最终值落地。
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-02_win-showcase-screen.md`
- 归档（全局）：`docs/change/2026-03-02_showcase-screen.md`

### 2026-03-02 - Android：Debug Action 直接下载 APK（debug-latest）
- Repo: `repo/MyFlowHub-Android`（已合并到 `main`：Android @ `c26c52e`；tag：`debug-latest`）
- Branch: `fix/android-ci-debug-release-assets`
- 目标：在 GitHub Actions 页面提供 Debug APK 直链下载（避免 artifact 下载 zip 才能拿到 apk）。
- 变更：
  - `ci.yml`：仅 `main` push 发布/更新 `debug-latest` pre-release，并上传/覆盖：
    - `app-debug.apk`
    - `myflowhub.aar`
  - `ci.yml`：Actions Summary 输出 Release 与两条直链（APK/AAR）；忽略所有 tag push 防止循环触发。
  - `ci.yml`：发布阶段基于 `artifacts/` 实际解压结构定位产物，避免硬编码路径导致找不到 APK；并使用 `curl` 更新 tag（避免 `gh api -f force=true` 参数歧义）。
- 验收：
  - push 到 `main` 后，打开该次 Actions run 的 Summary，点击 `Direct download` 直接下载 `.apk`。
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-02_android-debug-direct-apk.md`
- 归档（全局）：`docs/change/2026-03-02_android-debug-direct-apk.md`
 - 归档（全局）：`docs/change/2026-03-02_android-debug-release-assets.md`

### 2026-03-02 - MetricsNode：亮度变量下行控制（写入 → 执行）
- Repo: `repo/MyFlowHub-MetricsNode`（已合并到 `main`：MetricsNode @ `fc486d9`）
- Branch: `feat/metricsnode-brightness-control`
- 背景：亮度已可上报到 VarStore（默认 `sys_brightness_percent`），但外部写入该变量时 MetricsNode 未执行本机亮度调整，需要补齐“变量既是状态也是命令”。
- 变更：
  - core：VarStore `notify_set` 新增 `brightness_percent` 路由；解析整数并 clamp `0~100` 后入队控制动作；新增单测覆盖 clamp/非法值/owner 不匹配。
  - Windows：新增主显示器亮度执行器；control worker 消费 `brightness_percent` 并设置亮度。
  - Android：控制动作增加 `brightness_percent`，在 `Settings.System.canWrite==true` 时写入 `SCREEN_BRIGHTNESS`；Manifest 增加 `WRITE_SETTINGS`。
- 验收：
  - Go：`GOWORK=off go test ./... -count=1 -p 1`
  - Windows：`cd windows/frontend; npm ci; npm run build` 后 `cd ..; GOWORK=off go test ./... -count=1 -p 1`
  - Android：设置 `ANDROID_HOME/ANDROID_SDK_ROOT` 后 `cd android; ./gradlew :app:assembleDebug`
  - 手工：对 owner=<MetricsNodeNodeID> set `sys_brightness_percent=30/200` 观察亮度变化与 clamp。
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-02_metricsnode-brightness-control.md`
- 归档（全局）：`docs/change/2026-03-02_metricsnode-brightness-control.md`

### 2026-03-02 - Android：侧边栏/抽屉图标 + Drawer 圆角矩形化
- Repo: `repo/MyFlowHub-Android`（已合并到 `main`：Android @ `16b9f30`）
- Branch: `feat/android-nav-icons`
- 变更：
  - 导航：`NavigationRail` / `Drawer` 的 Tab 增加 Outlined 图标。
  - 视觉：抽屉面板与条目改为低圆角（圆角矩形风格）。
  - 依赖：补齐 `material-icons-extended`（确保 Outlined 图标可用）。
- 验收：
  - Android：`./gradlew :app:assembleDebug`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-02_android-nav-icons.md`
- 归档（全局）：`docs/change/2026-03-02_android-nav-icons.md`

### 2026-03-02 - MetricsNode：新增屏幕亮度 + Device ID 默认生成
- Repo: `repo/MyFlowHub-MetricsNode`（已合并到 `main`：MetricsNode @ `179ca6b`）
- Branch: `feat/metricsnode-brightness-deviceid`
- 背景：新增 `brightness_percent` 指标并默认绑定到 `sys_brightness_percent`；同时为 MetricsNode 自动生成并持久化 Device ID，避免忘填导致 Register/Login 失败。
- 变更：
  - core：新增 metric `brightness_percent`；默认 bindings 增加 `sys_brightness_percent`；仅当检测到“旧默认 3 项 bindings”时自动迁移到新默认（不覆盖用户自定义）。
  - Windows：主显示器亮度采集（2s 轮询；失败上报 `-1`）；bootstrap `auth.device_id` 为空时自动生成 `win-<hex>` 并落盘。
  - Android：亮度采集（1s 轮询，变化才上报；失败 `-1`）；prefs `device_id` 为空时生成 `android-<uuid>` 并落盘；Service 也兜底生成；GoBridge 对 `UpdateBrightnessPercent` 采用可选反射兼容旧 AAR。
  - 构建：补齐并纳入 `android/gradle/wrapper/gradle-wrapper*.jar`，并在 `.gitignore` 反忽略，避免 `./gradlew` 失效。
- 验收：
  - Go：`GOWORK=off go test ./... -count=1 -p 1`
  - Windows：`cd windows/frontend; npm ci; npm run build` 后 `cd ..; GOWORK=off go test ./... -count=1 -p 1`
  - Android：设置 `ANDROID_HOME/ANDROID_SDK_ROOT` 后 `cd android; ./gradlew :app:assembleDebug`
  - 手工：调整系统亮度，观察 VarStore `sys_brightness_percent` 更新；首次启动无需手填 Device ID 也可注册/登录。
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-02_metricsnode-brightness-deviceid.md`
- 归档（全局）：`docs/change/2026-03-02_metricsnode-brightness-deviceid.md`

### 2026-03-02 - MetricsNode：Windows 内屏亮度 WMI 读写兜底
- Repo: `repo/MyFlowHub-MetricsNode`（已合并到 `main`：MetricsNode @ `9df17da`）
- Branch: `fix/metricsnode-win-brightness-wmi`
- 背景：部分 Windows 笔记本内屏在 DXVA2/DDC/CI 路径下读取/写入亮度失败（常见 I2C 错误），导致 `brightness_percent=-1` 且下行写入难以生效。
- 变更：
  - 亮度读取：DXVA2 失败时使用 WMI（`ROOT\\WMI` / `WmiMonitorBrightness`）兜底，并在 WMI 可用时对 DXVA2 失败启用短期 backoff，减少无效重试与日志噪音。
  - 亮度写入：DXVA2 失败时使用 WMI（`WmiMonitorBrightnessMethods.WmiSetBrightness`）兜底（best-effort）。
  - 稳定性：修复 COM/VARIANT 生命周期与空值防护，避免双重释放与空指针风险。
- 验收：
  - Go：`GOWORK=off go test ./... -count=1 -p 1`
  - Windows：`cd windows/frontend; npm ci; npm run build` 后 `cd ..; GOWORK=off go test ./... -count=1 -p 1`
  - 手工：在笔记本内屏机上调节系统亮度，观察 VarStore `sys_brightness_percent` 跟随变化；对 owner=<MetricsNodeNodeID> 写入 `sys_brightness_percent=30` 观察亮度变化。
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-02_metricsnode-win-brightness-wmi.md`
- 归档（全局）：`docs/change/2026-03-02_metricsnode-win-brightness-wmi.md`

### 2026-03-02 - MetricsNode：修复 Windows WMI 亮度设置类型不匹配
- Repo: `repo/MyFlowHub-MetricsNode`（已合并到 `main`：MetricsNode @ `ebb30ab`）
- Branch: `fix/metricsnode-win-brightness-wmi-set`
- 背景：部分设备在 DXVA2 设置亮度失败（I2C/DDC）后，WMI fallback 调用 `WmiSetBrightness` 会返回 `类型不匹配`，导致亮度下行控制失败。
- 变更：
  - WMI `WmiSetBrightness` 默认改用 `int32`（`VT_I4`）传参，提升 provider 兼容性；并保留原 `uint32/uint8` 调用作为 fallback。
- 验收：
  - Go：`GOWORK=off go test ./... -count=1 -p 1`
  - Windows：`cd windows/frontend; npm ci; npm run build` 后 `cd ..; GOWORK=off go test ./... -count=1 -p 1`
  - 手工：对 owner=<MetricsNodeNodeID> 写入 `sys_brightness_percent=32`，亮度变化且日志不再出现 `wmi: ... 类型不匹配`。
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-02_metricsnode-win-brightness-wmi-set.md`
- 归档（全局）：`docs/change/2026-03-02_metricsnode-win-brightness-wmi-set.md`

### 2026-03-01 - Workspace：run-dev.ps1 默认启动 MetricsNode（Server + Win + MetricsNode）
- Scope: `scripts/run-dev.ps1`
- 背景：本地联调除 Server + Win 外，还需要同时启动 MetricsNode 上报指标到 VarStore，便于 Win 端订阅验证。
- 变更：
  - 默认新增启动 MetricsNode（`repo/MyFlowHub-MetricsNode/windows`：`wails dev`）
  - 新增 `-SkipMetricsNode` 开关以保持旧行为可用（仅启动 Server + Win）
- 验收：
  - `pwsh -File scripts/run-dev.ps1 -SkipServer -SkipWin -SkipMetricsNode`
  - `Get-Help .\scripts\run-dev.ps1 -Detailed`
- 归档（全局）：`docs/change/2026-03-01_dev-run-dev-metricsnode.md`

### 2026-03-01 - Workspace：go.work 加入 MetricsNode modules（修复 wails dev 报错）
- Scope: `go.work`
- 背景：MetricsNode Windows（`repo/MyFlowHub-MetricsNode/windows`）运行 `wails dev` 会执行 `go mod tidy`；若 workspace `go.work` 未包含该 module，会报 go.work module 错误并退出。
- 变更：
  - `go.work` 的 `use (...)` 增加：
    - `./repo/MyFlowHub-MetricsNode`
    - `./repo/MyFlowHub-MetricsNode/windows`
- 验收：
  - `cd repo/MyFlowHub-MetricsNode/windows; go list -m`
  - `cd repo/MyFlowHub-MetricsNode/windows; wails dev`
- 归档（全局）：`docs/change/2026-03-01_go-work-metricsnode.md`

### 2026-03-01 - MetricsNode：VarStore 下行控制（音量）+ 只读纠偏（电量）
- Repo: `repo/MyFlowHub-MetricsNode`（已合并到 `main`：MetricsNode @ `f265416`）
- Branch: `feat/metricsnode-var-control`
- 背景：VarStore `notify_set` 下行未被 MetricsNode 处理，导致“变量既是状态也是命令”的反向控制不可用；电量变量被误写后也缺少纠偏机制。
- 变更：
  - MetricsNode：接收 VarStore `notify_set`（兼容 MajorCmd/MajorMsg），按 bindings 反查 metric 并路由：
    - `sys_volume_percent` / `sys_volume_muted`：执行系统音量/静音（Windows + Android）。
    - `sys_battery_percent`：不执行本机行为，回写当前真实值（只读纠偏）。
  - Windows：COM endpoint volume 执行器 + 异步控制 worker（不阻塞收包循环，endpoint 复用）。
  - Android：actions 队列 + gomobile `DequeueActions()` + `NodeService` 轮询执行；新增权限 `MODIFY_AUDIO_SETTINGS`。
- 验收：
  - Go：`GOWORK=off go test ./... -count=1 -p 1`
  - 手工：对 owner=<MetricsNodeNodeID> set `sys_volume_percent/sys_volume_muted` 观察系统音量变化；set `sys_battery_percent` 观察纠偏回写。
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-01_metricsnode-var-control.md`
- 归档（全局）：`docs/change/2026-03-01_metricsnode-var-control.md`

### 2026-03-01 - MetricsNode：忽略 Windows 本地 config 目录（避免误提交密钥/快照）
- Repo: `repo/MyFlowHub-MetricsNode`（已合并到 `main`：MetricsNode @ `2528759`）
- Branch: `chore/metricsnode-ignore-windows-config`
- 背景：本地运行 `repo/MyFlowHub-MetricsNode/windows`（`wails dev/build`）会生成 `windows/config/*`（含 node_keys、auth_snapshot 等），既会导致 repo 长期 dirty，也存在误提交风险。
- 变更：
  - `windows/.gitignore` 增加 `config/` 忽略规则。
- 验收：
  - 运行一次 `cd windows; wails dev` 生成 `config/*` 后，`cd ..; git status --porcelain` 不应出现 `windows/config/`。
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-01_metricsnode-windows-ignore-config.md`
- 归档（全局）：`docs/change/2026-03-01_metricsnode-windows-ignore-config.md`

### 2026-03-01 - MetricsNode：升级间接依赖 + 同步 package.json.md5
- Repo: `repo/MyFlowHub-MetricsNode`（已合并到 `main`：MetricsNode @ `6d48032`）
- Branch: `chore/metricsnode-deps-upgrade`
- 背景：本地执行 `wails dev / go mod tidy` 产生 `golang.org/x/*` 间接依赖升级与 `package.json.md5` 不一致；合并以降低安全/兼容风险并避免反复 dirty。
- 变更：
  - Go：升级根模块 `golang.org/x/sys` 以及 Windows 子模块 `golang.org/x/{crypto,sys,text}`（均为 indirect）。
  - Windows：同步 `windows/frontend/package.json.md5` 与 `package.json` 的 MD5。
- 验收：
  - Go：`GOWORK=off go test ./... -count=1 -p 1`
  - Windows：`cd windows/frontend; npm ci; npm run build` 后，`cd windows; GOWORK=off go test ./... -count=1 -p 1`
  - MD5：`windows/frontend/package.json.md5` 与 `windows/frontend/package.json` 一致（大小写忽略）。
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-01_metricsnode-deps-upgrade.md`
- 归档（全局）：`docs/change/2026-03-01_metricsnode-deps-upgrade.md`

### 2026-03-01 - Android：新增 VarStore（订阅/自动更新/值直显）（v0.1.9）
- Repo: `repo/MyFlowHub-Android`（已合并到 `main`：Android @ `5cb88e5`；tag：`v0.1.9`）
- Branch: `feat/android-varstore`
- 变更：
  - 新增 VarStore Tab（对齐 Win VarPool）：My Variables / Watched Variables + Add/Edit 弹窗；列表直显 `value`。
  - Subscribe/Unsubscribe + 自动更新：Go 侧事件 ring-buffer + `VarStoreEventsPull`；Android 侧轮询消费并刷新 UI。
  - 交互：操作可取消（新操作打断旧操作）+ 成功/失败均有 snackbar 强反馈。
  - 输入限制：变量名提前校验 `^[A-Za-z0-9_]+$`（不合法直接提示，不发请求）。
- 验收：
  - Android：`./gradlew :app:assembleDebug`
  - Go：`cd hubmobile; GOWORK=off go test ./...`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-01_android-varstore.md`
- 归档（全局）：`docs/change/2026-03-01_android-varstore.md`

### 2026-03-01 - Win：VarPool 订阅后实时更新（接收 MajorCmd 通知帧）
- Repo: `repo/MyFlowHub-Win`（已合并到 `main`：Win @ `4a36f8f`）
- Branch: `fix/win-varpool-liveupdate`
- 背景：VarStore 订阅变更通知使用 `MajorCmd` 下发，Win 侧 VarPool 仅监听 `MajorMsg`，导致订阅后 UI 不能自动刷新。
- 变更：
  - Win：VarPool 接收 `MajorCmd`/`MajorMsg` 的 VarStore 通知帧（仍按 action 白名单过滤），订阅后可实时更新；
  - 测试：新增关键链路单测覆盖 `var_changed/var_deleted`（MajorCmd）。
- 验收：
  - Win：`GOWORK=off go test ./... -count=1 -p 1`
  - Win：`GOWORK=off wails build -nopackage`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-01_win-varpool-liveupdate.md`
- 归档（全局）：`docs/change/2026-03-01_win-varpool-liveupdate.md`

### 2026-03-01 - Win：修复 Release tag 格式校验（vX.Y.Z）
- Repo: `repo/MyFlowHub-Win`（已合并到 `main`：Win @ `9b05e1a`；tag：`v0.0.1`）
- Branch: `fix/win-release-tag-fix`
- 背景：首次 tag 发布 `v0.0.1` 时，Release workflow 在 `Validate tag format` 失败（PowerShell 正则双反斜杠导致无法匹配合法 tag）。
- 变更：
  - 修复 `release.yml` 的正则为 `^v\d+\.\d+\.\d+$`，并加注释避免再次误用。
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-03-01_win-release-tag-validate.md`
- 归档（全局）：`docs/change/2026-03-01_win-release-tag-validate.md`

### 2026-02-28 - Win：GitHub Actions 自动构建 + Tag Release（Windows amd64）
- Repo: `repo/MyFlowHub-Win`（已合并到 `main`：Win @ `0bfff4e`）
- Branch: `chore/win-actions-release`
- 变更：
  - CI：`pull_request -> main` + `push -> main` 触发 Windows `amd64` 构建，并上传 artifact（保留 30 天）。
  - Release：push tag `v1.2.3` 触发；严格校验 tag 格式与 “tag commit 属于 origin/main 历史”；生成 sha256；创建 GitHub Release 并上传 exe+sha256。
  - 版本固定：Go `1.24.5` / Node `22` / Wails CLI `v2.11.0`。
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-28_win-actions-release.md`
- 归档（全局）：`docs/change/2026-02-28_win-actions-release.md`

### 2026-02-28 - MetricsNode：Windows/Android 指标上报（电量/音量 → VarStore）
- Repo: `repo/MyFlowHub-MetricsNode`（已合并到 `main`：MetricsNode @ `d1242de`）
- Branch: `feat/metricsnode-mvp`
- 变更：
  - 形态：Windows（Wails UI）+ Android（Foreground Service）两个独立 App，运行后作为节点持续上报本机指标。
  - 指标：电量（无电池写 `-1`）、音量（Windows 默认输出设备 master；Android STREAM_MUSIC）→ VarStore（差量 `set`，默认 `visibility=public`）。
  - 配置：通过 Devices Config（Management `config_list/get/set`）远程下发 `metrics.bindings_json` 等运行时配置并热更新。
- 验收：
  - Go：`GOWORK=off go test ./... -count=1 -p 1`
  - Windows：`GOWORK=off wails build -nopackage`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-28_metricsnode-mvp.md`
- 归档（全局）：`docs/change/2026-02-28_metricsnode-mvp.md`

### 2026-02-28 - Android：分区块配色交换 + Devices Key/Value 编辑器（v0.1.8）
- Repo: `repo/MyFlowHub-Android`（已合并到 `main`：Android @ `0133ca1`；tag：`v0.1.8`）
- Branch: `fix/android-devices-editor`
- 变更：
  - UI：内容区背景使用 `surfaceColorAtElevation(1.dp)`；分区块 Card 显式使用 `surface`，实现“背景略灰/块更白”的层次交换。
  - Devices：树行展开按钮方形；错误态出现 `Retry` 时 `Edit` 仍保持最右对齐。
  - Details：`node_info` 失败/为空时给出明确提示；UI node 额外显示本地兜底信息（deviceId/hubId/role/targetAddr/nodeId）。
  - Edit：弹窗打开即 Key/Value 列表；value 懒加载 + 并发限制（Semaphore=4）；行内 Save；移除 `List/Get/Set`。
- 验收：
  - Android：`./gradlew :app:assembleDebug`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-28_android-ui-devices-kv-editor.md`
- 归档（全局）：`docs/change/2026-02-28_android-ui-devices-kv-editor.md`

### 2026-02-28 - Android：移除系统标题栏 + Devices 弹窗详情/编辑（v0.1.7）
- Repo: `repo/MyFlowHub-Android`（已合并到 `main`：Android @ `6002154`；tag：`v0.1.7`）
- Branch: `fix/android-devices-dialog`
- 变更：
  - ActionBar：切换到 `Theme.DeviceDefault.NoActionBar`，移除系统黑底白字标题栏，避免与 Compose TopBar 重复。
  - Devices：顶部改为 `Root node id + Load` 单行；移除 Direct/Subtree；节点行对齐 Win（展开 / 标题弹详情 / Edit 弹编辑，支持 Retry）；Duplicate 禁止展开避免环。
  - Login：登录后显示 `Node <id>`（与 `Hub <id>` 同行 chip）。
- 验收：
  - Android：`./gradlew :app:assembleDebug`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-28_android-devices-dialog.md`
- 归档（全局）：`docs/change/2026-02-28_android-devices-dialog.md`

### 2026-02-28 - Android：拆分 UI/HUB 身份（-ui/-hub）并自动迁移（v0.1.6）
- Repo: `repo/MyFlowHub-Android`（已合并到 `main`：Android @ `94f9f9c`；tag：`v0.1.6`）
- Branch: `fix/android-identity-split`
- 变更：
  - Prefs：拆分 `hub_self_id` / `ui_device_id`，旧 `self_id` 自动迁移为 `base-hub/base-ui`；迁移时清空登录状态。
  - AppRoot：移除 Hub/UI 身份互相覆盖；启动迁移后 snackbar 提示。
  - UI：Hub Start 自动规范化 `*-hub`；Login/Register/Login 自动规范化 `*-ui`；Register/Login 严格校验 `node_id/hub_id` 防止“假成功”。
- 验收：
  - Android：`./gradlew :app:assembleDebug`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-28_android-identity-split.md`
- 归档（全局）：`docs/change/2026-02-28_android-identity-split.md`

### 2026-02-27 - Android：外框浅色层次白 + 顶部品牌栏 + 全页面可滚动
- Repo: `repo/MyFlowHub-Android`（已合并到 `main`：Android @ `2fb0ed3`）
- Branch: `fix/android-ui-chrome`
- 变更：
  - 顶部：固定品牌栏 `MyFlowHub`；窄屏菜单按钮打开 Drawer；移除各 Tab 的顶部标题（Login/Hub/Devices...）。
  - 外框：TopBar / NavigationRail / DrawerSheet 使用 `surfaceColorAtElevation` 的浅色层次白；内容区使用 `surface`。
  - 滚动：Login/Hub/Protocols 整页支持滚动；全局内容增加 `imePadding()` 避免键盘遮挡；Logs/Protocols 移除重复页内标题。
- 验收：
  - Android：`./gradlew :app:assembleDebug`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-27_android-ui-chrome-scroll.md`
- 归档（全局）：`docs/change/2026-02-27_android-ui-chrome-scroll.md`

### 2026-02-27 - Android：抽屉导航 + Hub/Login Snackbar 强反馈（5s 状态刷新）
- Repo: `repo/MyFlowHub-Android`（已合并到 `main`：Android @ `477706f`）
- Branch: `feat/android-ui-feedback-drawer`
- 变更：
  - 导航：底部 Tab → 左侧抽屉（`ModalNavigationDrawer` + `TopAppBar` 菜单按钮）。
  - 反馈：`Hub`/`Login` 关键操作统一使用 `Snackbar` 提示成功/失败。
  - Hub：点击一次 `Start/Stop` 后，5s 内轮询刷新状态（`running=true/false`），并提示成功/失败/超时；状态读取放到 `Dispatchers.IO` 避免卡 UI。
  - Login：`Connect/Disconnect/Refresh/EnsureKeys/Register/Login/ClearAuth` 统一异步执行（`Dispatchers.IO`）+ Snackbar 强反馈 + 基础输入校验。
- 验收：
  - Hub：只点一次 `Start`，5s 内状态可自动刷新到 `running=true`（或提示失败/超时）。
  - Login/Hub：点击任一关键按钮都弹出 Snackbar，且 UI 触摸反馈不“发闷”。
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-27_android-ui-drawer-snackbar.md`
- 归档（全局）：`docs/change/2026-02-27_android-ui-drawer-snackbar.md`

### 2026-02-27 - Android：Snackbar 及时替换 + Hub/Login/Devices UI 精细化（v0.1.5）
- Repo: `repo/MyFlowHub-Android`（已合并到 `main`：Android @ `d78a567`）
- Branch: `feat/android-ui-polish-snackbar`
- 变更：
  - Snackbar：新提示到来 `dismiss()` 旧提示并立即展示（Replace 而非 Queue），解决 Register/Login “结果提示延迟”问题。
  - Login：更清晰分区（连接 / 身份与认证）；增加进度条 + 当前操作文案 + Cancel（UI 层打断，旧结果不回写）。
  - Hub：Card 分区 + 状态 Chip；Start/Stop 5s 轮询等待与超时提示；加载态更明确。
  - Devices：宽屏双栏（Tree / Details）+ Config 操作提示更清晰（仅宽屏双栏，窄屏保持单栏）。
  - 导航：宽屏使用 `NavigationRail`（左侧栏），窄屏保留 Drawer。
- 验收：
  - Android：`./gradlew :app:assembleDebug`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-27_android-ui-polish-snackbar.md`
- 归档（全局）：`docs/change/2026-02-27_android-ui-polish-snackbar.md`

### 2026-02-27 - Android：修复 Android15 前台服务类型崩溃 + gomobile 反射兼容（v0.1.3）
- Repo: `repo/MyFlowHub-Android`（已合并到 `main`：Android @ `9fcf46c`；tag：`v0.1.3`）
- Branch: `fix/android-fgs-type-gomobile-reflect`
- 背景 / 根因：
  - Android 15（targetSdk=34）点击 `Hub -> Start` 崩溃：`MissingForegroundServiceTypeException`。
  - `Login` 页面 `Go AAR unavailable`：gomobile Java 方法名大小写差异导致反射 `NoSuchMethod`（如 `EnsureInit/ensureInit`）。
- 变更：
  - `HubService` 声明/传递 `foregroundServiceType=dataSync`（使用 `ServiceCompat.startForeground`）。
  - 新增 `GoReflect`，反射调用兼容首字母大小写差异。
- 验收：
  - Android 15：点击 `Hub -> Start` 不再崩溃；通知常驻。
  - `Login`：不再出现 `Go AAR unavailable`（AAR 正常打包时）。
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-27_android-fgs-type-gomobile-reflect.md`
- 归档（全局）：`docs/change/2026-02-27_android-fgs-type-gomobile-reflect.md`

### 2026-02-27 - Android：手机作为 Hub（v1：登录/Devices/日志/子协议入口）
- Repo: `repo/MyFlowHub-Android`（已合并到 `main`：Android @ `1b4ff8e`）
- Branch: `feat/android-login-devices`
- 目标：
  - Android：增加 `Login/Devices/Logs/Protocols` 页面，支持手动 `ip:port` 连接；协议入口选型 B（每协议独立入口，v1 复用通用控制台）。
  - `hubmobile`：补齐 ES256 登录/签名（对齐 Win）、`Send&Await`、management（Devices）封装、10k ring buffer 日志与错误信息透出。
- 验收：
  - `hubmobile`：`GOWORK=off go test ./... -count=1`
  - Android：`./gradlew :app:assembleDebug`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-27_android-hub-ui-v1.md`
- 归档（全局）：`docs/change/2026-02-27_android-hub-ui-v1.md`

### 2026-02-26 - Android：修复 CI/Release 构建链路（gomobile/Gradle Wrapper）
- Repo: `repo/MyFlowHub-Android`（已合并到 `main`：Android @ `e59fd0a`）
- Branch: `fix/android-ci-gomobile-androidapi`
- 目标：
  - 修复 GitHub Actions 中 `gomobile bind` 的默认 `androidapi=16` 与 NDK r26（21..34）不兼容问题，并对齐 `minSdk=26`。
  - 修复 `javapkg` 默认值包含 Java 关键字导致的生成代码编译失败。
  - 修复 Ubuntu runner 上 Gradle Wrapper 启动与 Kotlin DSL 脚本编译问题，使 CI/Release 链路稳定。
- 验收：
  - Actions（push/PR）：`ci` 可稳定产出 `app-debug.apk` + `myflowhub.aar`（artifact）。
- 原 plan（repo）：`repo/MyFlowHub-Android/todo.md`
- 归档（全局）：`docs/change/2026-02-26_fix-android-ci-gomobile-androidapi.md`

### 2026-02-25 - Android：Hub + UI（M0 可行性验证）
- Repos:
  - `repo/MyFlowHub-Server`（已合并到 `main`：Server @ `63fb161`）
  - `repo/MyFlowHub-Android`（已合并到 `main`：Android @ `7eb0437`）
- Branch: `feat/android-hub-m0`
- 目标：
  - Server：提供可嵌入 `hubruntime`（Start/Stop/Status），并实现 parent bootstrap（解决父链 `sourceMismatch` 丢帧问题）。
  - Android：Compose 最小 UI + Foreground Service 常驻；提供最小配置/状态/启停；gomobile AAR 集成入口与构建脚本。
- 验收：
  - Server：`GOWORK=off go test ./... -count=1 -p 1`
  - Android：按 `repo/MyFlowHub-Android/docs/m0_smoke.md` 做 AAR+APK 构建与真机 LAN 冒烟。
- 原 plan（归档）：
  - `docs/plan_archive/plan_archive_2026-02-25_android-hub-m0_server.md`
  - `docs/plan_archive/plan_archive_2026-02-25_android-hub-m0_android.md`
- 归档（全局）：
  - `docs/change/2026-02-25_android-hubruntime-m0.md`
  - `docs/change/2026-02-25_android-hub-m0.md`

### 2026-02-25 - Android：M0 冒烟补齐（LAN 直连 + Parent 链路）
- Repo: `repo/MyFlowHub-Android`（已合并到 `main`：Android @ `87fef58`）
- Branch: `fix/android-hub-m0-smoke`
- 目标：
  - 修正 `docs/m0_smoke.md` 的历史 worktree 路径引用，补齐“LAN 直连 + 上联 Parent 并转发到手机”的冒烟链路。
  - 增加 PC 侧最小冒烟工具 `tools/hubsmoke`（不依赖 Win UI）用于 `register/list_nodes/node_echo`。
- 验收：
  - `tools/hubsmoke/`：`GOWORK=off go test ./... -count=1`
  - Android：按 `repo/MyFlowHub-Android/docs/m0_smoke.md` 做 AAR+APK 构建与真机双链路冒烟。
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-25_android-hub-m0-smoke_android.md`
- 归档（全局）：`docs/change/2026-02-25_android-hub-m0-smoke.md`
- 说明：本 workflow 不涉及 `repo/MyFlowHub-Server` 功能改动，仅在文档中作为 Parent Hub 的最小启动方式。

### 2026-02-25 - Android：GitHub Actions 自动构建/发布 APK（CI + Release）
- Repo: `repo/MyFlowHub-Android`（已合并到 `main`：Android @ `4482687`）
- Branch: `feat/android-apk-release`
- 目标：
  - CI：push/PR 自动构建（含 `gomobile` 生成 `myflowhub.aar`），产出 `app-debug.apk` 并作为 Actions Artifact 提供下载。
  - Release：推送 tag `vMAJOR.MINOR.PATCH` 后自动构建 **已签名** `app-release.apk` 并发布到 GitHub Releases（同时上传 `myflowhub.aar` 与 `build-info.txt`）。
- 验收：
  - 配置 GitHub Secrets（keystore + 密码/alias），按 `docs/release.md` 推送 tag（例如 `v0.1.0`）验证 Release 资产齐全。
- 归档（全局）：`docs/change/2026-02-25_android-apk-release-ci.md`
- 说明：
  - `hubmobile/go.mod` 仍使用 `replace ../../MyFlowHub-Server`；Actions 通过额外 checkout `myflowhub-server` 到 `repo/MyFlowHub-Server` 对齐 meta-workspace 目录结构。

### 2026-02-24 - Win：自身节点 Config（settings.json）接入 management config_*（避免 Edit 超时）
- Repo: `repo/MyFlowHub-Win`
- Branch: `fix/win-self-config`（已合并到 `main`：Win @ `ba1aae0`）
- 目标：
  - 修复 `Devices` 对自身节点点击 `Edit`（Config）超时问题（self-node 不再走 `sendAndAwait`）。
  - self-node 的 Config 真源对齐 Win 本地 `settings.json`（可读取/可编辑/可落盘）。
  - `config_list` 一次性列出 `settings.json` 的 raw keys（含 `profile.` 前缀与全局 key）。
- 验收：
  - `GOWORK=off go test ./... -count=1`
  - `GOWORK=off wails build -nopackage`
  - `frontend/`：`npm run build`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-24_win-self-config.md`
- 归档（全局）：`docs/change/2026-02-24_win-self-config.md`

### 2026-02-24 - Win：Devices 列表交互增强 + 页面头部移除 + 导航图标化
- Repo: `repo/MyFlowHub-Win`
- Branch: `refactor/win-ui-polish`（已合并到 `main`：Win @ `409f078`）
- 目标：
  - `Devices` 节点列表增加更清晰的 hover 反馈。
  - 修复 `Devices` 树 root 标记错误：仅真实 root 行显示 `(root)`（基于 `depth===0`）。
  - 全局移除页面顶部“模块说明 header 区块”，并迁移原 header 控件到页面内相关卡片 header 区（功能不丢失）。
  - 左侧导航从两字母缩写改为图标（新增依赖：`lucide-vue-next`）。
- 验收：
  - `GOWORK=off wails build -nopackage`
  - `frontend/`：`npm run build`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-24_win-ui-polish.md`
- 归档（全局）：`docs/change/2026-02-24_win-ui-polish.md`

### 2026-02-24 - Win：Devices 集成 Config 编辑 + 移除 Management + 导航重排
- Repo: `repo/MyFlowHub-Win`
- Branch: `refactor/win-devices-config`（已合并到 `main`：Win @ `d43889a`）
- 目标：
  - `Devices` 节点行右侧新增 Config 编辑入口（Overlay）。
  - 移除 `Management` 页面、导航与路由（不做 redirect）。
  - `Devices` 导航移动到 `Operations` 分组，并位于 `File Console` 上方。
- 验收：
  - `GOWORK=off wails build -nopackage`
  - `frontend/`：`npm run build`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-24_win-devices-config.md`
- 归档（全局）：`docs/change/2026-02-24_devices-config-converge.md`

### 2026-02-24 - Win：统一修复弹窗遮罩覆盖与 Esc 关闭
- Repo: `repo/MyFlowHub-Win`
- Branch: `fix/win-overlay-mask`（已合并到 `main`：Win @ `d8351af`）
- 目标：
  - 修复 Modal 背景遮罩无法覆盖顶部 Header 的问题（根因：祖先 `transform` 影响 `position: fixed` 参照系）。
  - 新增通用 `<Overlay>`（默认 Teleport 到 `body`）并迁移各页面遮罩，统一关闭策略与层级策略。
  - 支持 Esc 关闭（Modal + Profile 菜单），多层 Overlay 并存时仅关闭最上层。
- 验收：
  - `GOWORK=off wails build -nopackage`
  - `frontend/`：`npm run build`
  - 冒烟：Devices/File/Flow/Management/VarPool 任一弹窗打开时 Header 必须被遮罩；Profile 菜单外部点击 + Esc 可关闭（透明 click-catcher 不出现黑遮罩）
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-24_win-overlay-mask.md`
- 归档（全局）：`docs/change/2026-02-24_win-overlay-mask.md`

### 2026-02-24 - Devices：节点信息弹窗（management/node_info）
- Repo:
  - `repo/MyFlowHub-Proto`
  - `repo/MyFlowHub-SubProto`（module：`subproto/management`）
  - `repo/MyFlowHub-Server`
  - `repo/MyFlowHub-Win`
- Branch: `feat/node-info`（已合并到 `main`：Proto @ `d852a93`；SubProto @ `ecd7c33`；Server @ `a1b933e`；Win @ `f583edd`）
- Tags:
  - Proto：`v0.1.1`
  - SubProto/management：`management/v0.1.1`
  - Server：`v0.0.2`
- 目标：
  - Win 的 `Devices`：点击节点弹出详情弹窗，展示平台/版本等基础信息（KV）。
  - 信息必须来自节点本身：目标节点本地采集并通过 management `node_info` 回包；Win 自身节点短路本地采集（避免 timeout）。
- 验收：
  - `GOWORK=off go test ./...`（Proto/SubProto(management)/Server/Win）
  - 冒烟：Win `Session → Devices` 点击任意节点弹窗可用；点击自身节点不超时
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-24_devices-node-info.md`
- 归档（全局）：`docs/change/2026-02-24_devices-node-info.md`

### 2026-02-23 - Win：全局 Toast + 导航图标统一 + Local Hub 参数 + Devices 自身节点短路
- Repo: `repo/MyFlowHub-Win`
- Branch: `feat/win-ui-toast`（已合并到 `main`：Win @ `503ce3c`）
- 目标：
  - 全仓操作反馈统一为顶部 Toast（success/info/warn/error），替换页面内零散 message 红字。
  - 左侧导航 `short` 图标块统一为固定尺寸方形圆角矩形（避免被压缩导致形变）。
  - Local Hub 支持常用 `hub_server` flags 配置 + 高级 extra args（每行一个完整参数）。
  - Devices：当 management 目标节点为 Win 自身（`targetID == sourceID`）短路返回空列表，避免超时。
- 验收：
  - `GOWORK=off go test ./...`
  - `GOWORK=off wails build -nopackage`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-23_win-ui-toast.md`
- 归档（全局）：`docs/change/2026-02-23_win-ui-toast-localhub.md`

### 2026-02-22 - Win：本机 Local Hub（下载/启动 hub_server）
- Repo:
  - `repo/MyFlowHub-Server`
  - `repo/MyFlowHub-Win`
- Branch: `feat/localhub`（已合并到 `main`：Server @ `2e611b7`；Win @ `68b299c`）
- Tag: `myflowhub-server v0.0.1`（用于触发 GitHub Release assets）
- 注意：如遇到 `git push` 连接重置/无法连接 GitHub，可稍后重试（Win 当前本地 `main` 可能领先 `origin/main`）。
- 目标：
  - Win 新增 `Session → Local Hub`：从 GitHub Releases 下载/校验/安装 `hub_server`，并支持启动/停止/重启；端口冲突自动选择可用端口；展示 `addr/pid/log`。
  - Server 新增 Release 流水线：push `v*` tag 自动构建并发布 windows/linux amd64 压缩包与 `checksums.txt`。
- 验收：
  - Server：`GOWORK=off go test ./...`、`GOWORK=off go build ./cmd/hub_server`
  - Win：`GOWORK=off go test ./...`、`GOWORK=off wails build -nopackage`
  - 冒烟（依赖 Release assets）：Local Hub Install Latest → Start → Home Connect 到实际 `addr` → Register/Login
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-22_win-localhub.md`
- 归档（全局）：`docs/change/2026-02-22_win-localhub.md`

### 2026-02-21 - Workspace：一键启动 Server + Win（本地验证脚本）
- 目标：一条命令分别打开两个窗口启动 `hub_server`（`go run ./cmd/hub_server`）与 Win（`wails dev`），便于快速冒烟验证。
- 文件：
  - 启动脚本：`scripts/run-dev.ps1`
  - 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-21_dev-run-server-win.md`
  - 归档（全局）：`docs/change/2026-02-21_dev-run-server-win.md`
- 验收：
  - `pwsh -File scripts/run-dev.ps1 -SkipServer -SkipWin`
  - `Get-Help .\\scripts\\run-dev.ps1 -Detailed`

### 2026-02-21 - Win：Session 下新增 Devices（设备查询）
- Repo: `repo/MyFlowHub-Win`
- Branch: `feat/win-session-devices` (merged to `main` @ `b497c40`)
- 目标：
  - 在左侧导航 Session 分组下新增 Devices 入口
  - 提供 `management.list_nodes` / `management.list_subtree` 的快速查询（Subtree 非递归，仅“直连节点 + 自身”）
- 验收：
  - `GOWORK=off go test ./...`
  - `GOWORK=off wails build -nopackage`
  - 冒烟：Home Connect + Register/Login → Session/Devices → List Direct / List Subtree
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-21_win-session-devices.md`
- 归档（全局）：`docs/change/2026-02-21_win-session-devices.md`

### 2026-02-21 - Win：Devices 树形展示（懒加载）
- Repo: `repo/MyFlowHub-Win`
- Branch: `feat/win-devices-tree` (merged to `main` @ `3109091`)
- 目标：
  - `Session → Devices` 改为树形展示，按需展开懒加载（逐级 `list_nodes`）
  - Mode 下拉切换（Direct/Subtree），切换自动重载；Root 输入 Enter 重载
  - 重复节点允许展示但禁止继续展开（防环）
  - Subtree 明确非递归（当前为“直连 + 自身”）
- 验收：
  - `GOWORK=off go test ./...`
  - `GOWORK=off wails build -nopackage`
  - 冒烟：Home Connect + Register/Login → Session/Devices → 自动加载 Root → 展开/切换 mode/Enter 重载
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-21_win-devices-tree.md`
- 归档（全局）：`docs/change/2026-02-21_win-devices-tree.md`

### 2026-02-21 - Win：Devices 树展开修复（子节点可见/可展开）
- Repo: `repo/MyFlowHub-Win`
- Branch: `fix/win-devices-tree-expand` (merged to `main` @ `97b5f30`)
- 目标：
  - 修复“Root 行显示 Children 数量但列表不渲染子节点/无法展开”的问题
  - 不改 wire/协议与后端，仅修复前端 store 响应式更新
- 验收：
  - `GOWORK=off go test ./...`
  - `GOWORK=off wails build -nopackage`
  - 冒烟：Home Connect + Register/Login → Session/Devices → Root 自动加载应立即出现 child 行 → 点击 `+` 展开下一层
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-21_win-devices-tree-expand.md`
- 归档（全局）：`docs/change/2026-02-21_win-devices-tree-expand.md`

### 2026-02-21 - Win：Flow DAG 编辑器体验增强（Auto Layout / UndoRedo / 快捷键 / 状态 Overlay）（PR2）
- Repo: `repo/MyFlowHub-Win`
- Branch: `feat/win-dag-editor-ux` (merged to `main` @ `14df58d`)
- 目标：
  - Auto Layout：TB（从上到下）
  - Undo/Redo：覆盖结构编辑 + 节点移动 + auto layout + Node Detail 参数编辑（blur/change 合并策略）
  - 快捷键：Delete（优先删边）、Ctrl+Z / Ctrl+Y（或 Ctrl+Shift+Z）、Ctrl+S 保存
  - 画布辅助：MiniMap + Controls + Background
  - 节点状态 Overlay：点击 Status 刷新后渲染到画布节点
- 验收：
  - `GOWORK=off go test ./... -count=1 -p 1`
  - `GOWORK=off wails build -nopackage`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-21_win-flow-dag-editor-pr2.md`
- 归档（全局）：`docs/change/2026-02-21_win-flow-dag-editor-pr2.md`

### 2026-02-21 - Win：Flow DAG 画布编辑器（拖拽/连线/布局持久化）（PR1）
- Repo: `repo/MyFlowHub-Win`
- Branch: `feat/win-dag-editor` (merged to `main` @ `cd06760`)
- 目标：
  - Flow 编辑改为 DAG 画布：拖拽节点、拖拽连线创建边
  - 布局持久化：`spec._ui={x,y}`（仅 payload 扩展；不改 HeaderTcp）
  - `node.id` 作为图 key：在 UI 中只读；新增边即时阻止成环
- 验收：
  - `GOWORK=off go test ./... -count=1 -p 1`
  - `GOWORK=off wails build -nopackage`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-21_win-flow-dag-editor-pr1.md`
- 归档（全局）：`docs/change/2026-02-21_win-flow-dag-editor-pr1.md`

### 2026-02-20 - Proto：协议映射文档生成器（路线 A：Go 为真源，生成衍生物）
- Repo: `repo/MyFlowHub-Proto`
- Branch: `chore/protocol-mapgen` (merged to `main` @ `74864d9`)
- 目标：
  - 保持 `protocol/*/types.go` 为 single source-of-truth（wire 不改）
  - 新增 canonical 文档 `repo/MyFlowHub-Proto/docs/protocol_map.md`（半自动：仅替换生成区块）
  - 新增生成器：`go run ./cmd/protocolmapgen -write|-check -out docs/protocol_map.md`
  - Proto 仓内 `go test ./...` 强制校验文档与协议定义一致（门禁）
- 验收：
  - `GOWORK=off go test ./... -count=1 -p 1`
  - `GOWORK=off go run ./cmd/protocolmapgen -check -out docs/protocol_map.md`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-20_proto-protocol-mapgen.md`
- 归档（全局）：`docs/change/2026-02-20_protocol-mapgen.md`

### 2026-02-20 - SubProto：拆剩余子协议为独立 Go module（含 broker）
- Repo: `repo/MyFlowHub-SubProto`
- Branch: `refactor/subproto-modules-all` (merged to `main` @ `fdd0ae2`；tags：`auth/broker/exec/file/flow/forward/varstore` 均 `v0.1.0`)
- 目标：
  - A2（单仓多 module）：将 `auth/varstore/file/forward/exec/flow` 一次性拆为独立 module（wire 不改）
  - 新增共享 module：`broker`（承载 `exec/flow` “同进程 reqID -> resp 投递”能力，解耦 Server 私有包）
- 验收：`GOWORK=off go test ./... -count=1 -p 1`（在各 module 目录分别执行）
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-20_subproto-split-remaining-modules.md`
- 归档（全局）：`docs/change/2026-02-20_subproto-split-remaining-modules.md`

### 2026-02-20 - Server：剩余子协议改为依赖 `myflowhub-subproto/*` modules（并删除内置实现）
- Repo: `repo/MyFlowHub-Server`
- Branch: `refactor/subproto-modules-all` (merged to `main` @ `ca374b4`)
- 目标：
  - Server 改为依赖 `myflowhub-subproto/{auth,varstore,file,forward,exec,flow}`（`v0.1.0`）
  - 删除 `subproto/auth|varstore|file|forward|exec|flow` 与 `internal/broker`（wire 不改，行为保持）
- 验收：
  - `GOWORK=off go test ./... -count=1 -p 1`
  - `GOWORK=off go test ./tests -run TestRootHubPing -count=1`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-20_server-use-subproto-remaining-modules.md`
- 归档（全局）：`docs/change/2026-02-20_server-use-subproto-remaining-modules.md`

### 2026-02-19 - Win：Auth/Flow/TopicBus/VarPool services 收敛（返回值 + 业务事件）
- Repo: `repo/MyFlowHub-Win`
- Branch: `refactor/win-services-converge` (merged to `main` @ `3af94f5`)
- 目标：
  - Auth/Flow：Wails API 返回 `resp, error`，前端用返回值更新业务状态（不再解析 `session.frame` payload）。
  - TopicBus/VarPool：通知/事件流改为 Go 侧解析 `session.frame` → 发布业务事件（`topicbus.event`、`varpool.changed/deleted`）→ 前端订阅业务事件。
  - `session.frame` 保留用于 Debug/观测，不作为业务数据源。
- 验收：
  - `GOWORK=off go test ./... -count=1 -p 1`
  - `GOWORK=off wails generate module`
  - `cd frontend && npm ci && npm run build`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-19_win-services-converge.md`
- 归档（全局）：`docs/change/2026-02-19_win-services-converge.md`

### 2026-02-19 - Core：上移 subproto/kit 到 MyFlowHub-Core（发布 v0.2.1）
- Repo: `repo/MyFlowHub-Core`
- Branch: `refactor/subproto-kit-core` (merged to `master` @ `a2d7d29`, tag `v0.2.1`)
- 目标：
  - 将 `MyFlowHub-Server/subproto/kit` 上移为 `github.com/yttydcs/myflowhub-core/subproto/kit`
  - 形成“子协议独立 module”可复用的 action 模板与响应发送工具（wire 不改）
- 验收：`GOWORK=off go test ./... -count=1 -p 1`
- 归档（全局）：`docs/change/2026-02-19_core-subproto-kit.md`

### 2026-02-19 - Server：切换到 MyFlowHub-Core/subproto/kit（依赖 core@v0.2.1）
- Repo: `repo/MyFlowHub-Server`
- Branch: `refactor/subproto-management-module`（堆叠包含 `refactor/subproto-kit-core`；merged to `main` @ `e7ac863`）
- 目标：
  - Server 删除 `subproto/kit`，全仓改依赖 `github.com/yttydcs/myflowhub-core/subproto/kit`
  - wire 不变（SubProto/Action/JSON/HeaderTcp 语义不变）
- 验收：`GOWORK=off go test ./... -count=1 -p 1`
- 归档（全局）：`docs/change/2026-02-19_server-use-core-kit.md`

### 2026-02-19 - SubProto：拆 management 为独立 Go module（management/v0.1.0）
- Repo: `repo/MyFlowHub-SubProto`
- Branch: `refactor/subproto-management-module`（初始化主分支：`main` @ `22af44a`）
- 目标：
  - 新仓 `myflowhub-subproto`（A2：单仓多 module）
  - `management/` 独立 `go.mod` + tag：`management/v0.1.0`
  - 依赖方向：仅依赖 `myflowhub-core` 与 `myflowhub-proto`（不依赖 Server/Win）
- 验收：`GOWORK=off go test ./... -count=1 -p 1`
- 归档（全局）：`docs/change/2026-02-19_subproto-management-v0.1.0.md`

### 2026-02-19 - Server：拆 management 为独立 module 依赖（myflowhub-subproto/management）
- Repo: `repo/MyFlowHub-Server`
- Branch: `refactor/subproto-management-module` (merged to `main` @ `e7ac863`)
- 目标：
  - Server 删除 `subproto/management` 实现目录
  - 依赖 `github.com/yttydcs/myflowhub-subproto/management v0.1.0`（wire 不改）
- 验收：`GOWORK=off go test ./... -count=1 -p 1`
- 归档（全局）：`docs/change/2026-02-19_server-use-subproto-management.md`

### 2026-02-19 - SubProto：拆 topicbus 为独立 Go module（topicbus/v0.1.0）
- Repo: `repo/MyFlowHub-SubProto`
- Branch: `refactor/subproto-topicbus-module`（merged to `main` @ `6e8338a`，tag `topicbus/v0.1.0`）
- 目标：
  - 在 `myflowhub-subproto` 新增 module：`github.com/yttydcs/myflowhub-subproto/topicbus`
  - 迁入原 Server 的 `subproto/topicbus` 实现（wire 不改）
  - 依赖方向：仅依赖 `myflowhub-core` + `myflowhub-proto`（禁止依赖 Server/Win）
- 验收：`GOWORK=off go test ./... -count=1 -p 1`（在 `topicbus/` 下执行）
- 归档（全局）：`docs/change/2026-02-19_subproto-topicbus-v0.1.0.md`

### 2026-02-19 - Server：拆 topicbus 为独立 module 依赖（myflowhub-subproto/topicbus）
- Repo: `repo/MyFlowHub-Server`
- Branch: `refactor/subproto-topicbus-module` (merged to `main` @ `5e54882`)
- 目标：
  - Server 删除 `subproto/topicbus` 实现目录
  - 依赖 `github.com/yttydcs/myflowhub-subproto/topicbus v0.1.0`（wire 不改）
- 验收：`GOWORK=off go test ./... -count=1 -p 1`
- 归档（全局）：`docs/change/2026-02-19_server-use-subproto-topicbus.md`

### 2026-02-18 - Win：Management 收敛为返回值 API（去 frame 解析）
- Repo: `repo/MyFlowHub-Win`
- Branch: `refactor/win-mgmt-service` (merged to `main` @ `627e301`)
- 目标：
  - Management Wails API 直接返回 `myflowhub-proto/protocol/management` 的 `*Resp`（resp + error）
  - `frontend/src/stores/management.ts` 不再解析 `session.frame` 作为业务数据源（frame 保留用于 Debug/观测）
  - Go 侧增加 UI 友好错误与可定位日志（action + 原因）
- 验收：
  - `GOWORK=off go test ./... -count=1 -p 1`
  - `GOWORK=off wails generate module`
  - `cd frontend && npm ci && npm run build`
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-18_win-mgmt-service.md`
- 归档（全局）：`docs/change/2026-02-18_win-mgmt-service.md`

### 2026-02-18 - 文档一致性整理（Docs consistency）
- 目标：统一 MyFlowHub3 控制面文档口径（Proto 为 canonical、Server/protocol 为兼容壳），修正过期路径/矛盾叙述，并将 PR2-4（minimal/full）明确标注为 Deferred。
- 文档变更：
  - `plan.md`：定位调整为“全局计划与归档”（Legacy 内容保留但不再误导）。
  - `repos.md` / `target.md`：PR2-4 标注 Deferred + 编号口径说明。
  - `docs/protocol_map.md`：协议模型来源改为 `myflowhub-proto/protocol/*`。
- 原 plan（归档）：`docs/plan_archive/plan_archive_2026-02-18_docs-consistency.md`
- Repos（已合并并 push）：
  - `repo/MyFlowHub-Win`：`chore/docs-consistency` → `main` @ `f8a44e0`
  - `repo/MyFlowHub-SDK`：`chore/docs-consistency` → `main` @ `f8cf23f`
- 归档（全局）：`docs/change/2026-02-18_docs-consistency.md`

### 2026-02-18 - Semver 依赖化：发布 core/proto/sdk tag + SDK/Server/Win 去 replace（PR19）
- Repos / Branch：
  - `repo/MyFlowHub-Core`：`chore/core-semver` (merged to `master` @ `a838345`, tag `v0.2.0`)
  - `repo/MyFlowHub-Proto`：`chore/proto-semver` (merged to `main` @ `71a1b5c`, tag `v0.1.0`)
  - `repo/MyFlowHub-SDK`：`chore/sdk-semver` (merged to `main` @ `bf6d741`, tag `v0.1.0`)
  - `repo/MyFlowHub-Server`：`chore/server-semver-deps` (merged to `main` @ `cd623df`)
  - `repo/MyFlowHub-Win`：`chore/win-semver-deps` (merged to `main` @ `dd213a9`)
- 目标：
  - 移除本地 `replace ../MyFlowHub-*`，确保单仓 clone 可 `GOWORK=off go test ./...`
  - 本地联调改用 `d:\\project\\MyFlowHub3\\go.work`（不提交）
- 归档（全局）：
  - `docs/change/2026-02-18_semver-deps.md`
  - `docs/change/2026-02-18_core-v0.2.0.md`
  - `docs/change/2026-02-18_proto-v0.1.0.md`
  - `docs/change/2026-02-18_sdk-v0.1.0.md`
  - `docs/change/2026-02-18_server-semver-deps.md`
  - `docs/change/2026-02-18_win-semver-deps.md`

### 2026-02-18 - Server：assist/up/notify action 注册模板化（wire 不改）（PR2-3）
- Repo: `repo/MyFlowHub-Server`
- Branch: `refactor/server-action-template` (merged to `main` @ `d514337`)
- 目标：
  - 引入统一 action 模板（`kit.NewAction`/`FuncAction`）与 action 分类（`ActionKind`，仅用于工程组织/可观测，不参与路由语义）
  - `auth` + `varstore` 的 `assist_* / up_* / notify_*` 注册方式收敛为同一写法（wire 不改，语义不变）
- 验收：
  - `GOWORK=off go test ./... -count=1 -p 1`
  - 冒烟：`go test ./tests -run TestRootHubPing -count=1`
- 归档（全局）：`docs/change/2026-02-18_assist-up-notify-action-template.md`

### 2026-02-18 - Server：action 注册模板化补齐（exec/flow/topicbus/management → kit.NewAction）
- Repo: `repo/MyFlowHub-Server`
- Branch: `refactor/server-action-kit` (merged to `main` @ `e1e36e1`)
- 目标：
  - 将 `exec/flow/topicbus/management` 的 action 注册从 BaseAction wrapper/结构体写法迁移到 `kit.NewAction(...)`
  - wire 不改，行为与语义不变
- 验收：
  - `GOWORK=off go test ./... -count=1 -p 1`
  - 冒烟：`go test ./tests -run TestRootHubPing -count=1`
- 归档（全局）：`docs/change/2026-02-18_server-action-kit-coverage.md`

### 2026-02-18 - Server：File CTRL 响应继承 MsgID/TraceID（PR18-SERVER-FileCtrl-Ids）
- Repo: `repo/MyFlowHub-Server`
- Branch: `fix/file-ctrl-inherit-ids` (merged to `main` @ `46039f8`)
- 目标：
  - `file.read_resp/write_resp` 响应头继承请求 `MsgID/TraceID`，支持 SDK v1 Awaiter 按 `MsgID+SubProto+Action` 匹配。
  - wire 不变；`Major` 保持 `MajorOKResp`。
  - 单测增强断言 `MsgID/TraceID` 继承。
- 归档（全局）：`docs/change/2026-02-18_file-ctrl-inherit-ids.md`

### 2026-02-18 - SDK Awaiter：支持 File CTRL KindCtrl 前缀解包（PR18-SDK-Await-FileCtrl）
- Repo: `repo/MyFlowHub-SDK`
- Branch: `feat/await-file-ctrl` (merged to `main` @ `c8d8a31`)
- 目标：
  - 当 `SubProto=file` 且 `payload[0]==KindCtrl` 时，Awaiter 解包 `payload[1:]` 获取 `action` 用于匹配。
  - 保持 Awaiter 语义不变：matched frame 仍会 deliver，且 `SetOnFrame` 仍能观察 matched frame。
  - 新增单测覆盖 KindCtrl 前缀下的 send+await 成功匹配。
- 归档（全局）：`docs/change/2026-02-18_sdk-await-file-ctrl.md`

### 2026-02-18 - Win：File(list/read_text) 改为 send+await（SDK v1 Awaiter）（PR18-WIN-File-Await）
- Repo: `repo/MyFlowHub-Win`
- Branch: `refactor/win-file-ctrl-await` (merged to `main` @ `ac03193`)
- 目标：
  - `FileService.List/ReadText` 改为 send+await（等待 `read_resp`），`*Simple` 默认超时 `8s`，`code!=1` 返回 error。
  - 保持事件链路不变：仍由 `session.frame` 驱动 `file.list/file.text`；前端增加 loading 兜底。
- 归档（全局）：`docs/change/2026-02-18_win-file-await.md`

### 2026-02-16 - Auth 直返 *_resp 统一为 MajorOKResp（PR10-Auth-DirectOKResp）
- Repo: `repo/MyFlowHub-Server`
- Branch: `fix/auth-okresp-direct` (merged to `main` @ `69be0cd`)
- 目标：
  - 修复 Auth 直返客户端 `register_resp/login_resp/revoke_resp` 继承 `MajorCmd` 导致 SDK v1 Awaiter 超时的问题。
  - 不改 wire；仅在复用请求头时强制 `Major=MajorOKResp`，其余字段（含 `MsgID/TraceID`）保持继承。
  - 内部逐跳链路（`assist_* / up_*`）保持 `MajorCmd` 不变；失败响应仍用 `MajorOKResp`，错误由 `data.code` 表达。
- 归档（全局）：`docs/change/2026-02-16_auth-direct-resp-major-ok.md`

### 2026-02-16 - Exec/Flow 响应帧统一为 MajorOKResp（PR11-Resp-MajorOKResp）
- Repo: `repo/MyFlowHub-Server`
- Branch: `fix/resp-major-okresp` (merged to `main` @ `863da57`)
- 目标：
  - 将 `exec.call_resp` 与 `flow.*_resp`（`set/run/status/list/get`）的 HeaderTcp `Major` 统一为 `MajorOKResp`。
  - 与 Core 路由统一规则对齐：请求帧仍为 `MajorCmd` 逐跳进入 handler；响应帧走 `TargetID` 的 Core 快速转发，中间节点无需子协议 handler 转发。
  - big-bang：不提供兼容开关；失败响应仍使用 `MajorOKResp`，错误通过 payload 的 `code/msg` 表达。
- 归档（全局）：`docs/change/2026-02-16_exec-flow-resp-major-ok.md`

### 2026-02-17 - File 响应帧统一为 MajorOKResp（PR12-File-Resp-MajorOKResp）
- Repo: `repo/MyFlowHub-Server`
- Branch: `fix/file-resp-major-okresp` (merged to `main` @ `2e2dec2`)
- 目标：
  - 将 `file.read_resp/write_resp` 的 HeaderTcp `Major` 统一为 `MajorOKResp`，使其按 `TargetID` 走 Core 快速转发。
  - 请求帧（CTRL：`read/write`）保持 `MajorCmd` 逐跳进入 handler；DATA/ACK 仍为 `MajorMsg`。
  - big-bang：不提供兼容开关；失败响应仍使用 `MajorOKResp`，错误通过 payload 的 `code/msg` 表达。
- 归档（全局）：`docs/change/2026-02-17_file-resp-major-ok.md`

### 2026-02-17 - SDK Awaiter：增加 onFrame hook（匹配帧也可观察）（PR13-SDK-Await-Hooks）
- Repo: `repo/MyFlowHub-SDK`
- Branch: `feat/sdk-await-hooks` (merged to `main` @ `0d030a2`)
- 目标：
  - 为 SDK v1 `await.Client` 增加 `SetOnFrame`（全帧 tap）：匹配成功帧也可被上层观察/记录。
  - 保持语义不变：匹配成功帧仍会被拦截 deliver，不会走 onUnmatched。
  - 为 Win 等调用方“send+await 与 session.frame 事件并存”提供基础能力。
- 归档（全局）：`docs/change/2026-02-17_sdk-await-onframe.md`

### 2026-02-17 - Win：Auth Register/Login 改为 send+await（SDK v1 Awaiter）（PR13-WIN-Awaiter）
- Repo: `repo/MyFlowHub-Win`
- Branch: `refactor/win-auth-await` (merged to `main` @ `321a0be`)
- 目标：
  - Session 底层接入 SDK v1 Awaiter，并通过 `onFrame` 保持 `session.frame` 事件不缺失；新增 `SendCommandAndAwait`。
  - Auth 的 `Register/Login` 改为 send+await：默认 `8s` 超时（Simple 方法）；`code!=1` 直接返回 error。
- 归档（全局）：`docs/change/2026-02-17_win-auth-await.md`

### 2026-02-17 - Win：Management 改为 send+await（SDK v1 Awaiter）（PR14-WIN-Mgmt-Awaiter）
- Repo: `repo/MyFlowHub-Win`
- Branch: `refactor/win-mgmt-await` (merged to `main` @ `86f8406`)
- 目标：
  - 将 Management 的常用动作升级为 send+await：`node_echo/list_nodes/list_subtree/config_get/config_set/config_list`。
  - `*Simple` 方法增加默认 `8s` 超时；`code!=1` 视为错误并返回 error。
  - 保持前端行为不变：仍能收到 `session.frame` 并按既有 store 逻辑更新 UI。
- 归档（全局）：`docs/change/2026-02-17_win-mgmt-await.md`

### 2026-02-17 - Win：VarPool（VarStore）改为 send+await（SDK v1 Awaiter）（PR15-WIN-VarPool-Awaiter）
- Repo: `repo/MyFlowHub-Win`
- Branch: `refactor/win-varpool-await` (merged to `main` @ `233888a`)
- 目标：
  - 将 VarPool（VarStore 子协议）的常用动作升级为 send+await：`set/get/list/revoke/subscribe/unsubscribe`。
  - `*Simple` 方法增加默认 `8s` 超时；`code!=1` 视为错误并返回 error。
  - 保持前端行为不变：仍能收到 `session.frame` 并按既有 store 逻辑更新 UI。
  - `unsubscribe` 回包 action 以现有协议实现为准（复用 `subscribe_resp`）。
- 归档（全局）：`docs/change/2026-02-17_win-varpool-await.md`

### 2026-02-17 - Win：TopicBus 改为 send+await（SDK v1 Awaiter）（PR16-WIN-TopicBus-Awaiter）
- Repo: `repo/MyFlowHub-Win`
- Branch: `refactor/win-topicbus-await` (merged to `main` @ `c5ecb13`)
- 目标：
  - 将 TopicBus 控制动作升级为 send+await：`subscribe/subscribe_batch/unsubscribe/unsubscribe_batch/list_subs`。
  - `*Simple` 方法增加默认 `8s` 超时；`code!=1` 视为错误并返回 error。
  - `publish` 保持 send-only（无 `publish_resp`），前端事件仍由 `session.frame` 驱动。
- 归档（全局）：`docs/change/2026-02-17_win-topicbus-await.md`

### 2026-02-17 - Win：Flow 改为 send+await（SDK v1 Awaiter）（PR17-WIN-Flow-Awaiter）
- Repo: `repo/MyFlowHub-Win`
- Branch: `refactor/win-orch-await` (merged to `main` @ `4f8b12c`)
- 目标：
  - 将 Flow 控制动作升级为 send+await：`set/run/status/list/get`。
  - `*Simple` 方法增加默认 `8s` 超时；`code!=1` 视为错误并返回 error。
  - 保持前端行为不变：仍能收到 `session.frame` 并按既有 store 逻辑更新 UI。
- 归档（全局）：`docs/change/2026-02-17_win-flow-await.md`

### 2026-02-16 - SDK v1 Awaiter（按 MsgID+SubProto+Action 等待响应）（PR9-SDK-Awaiter）
- Repo: `repo/MyFlowHub-SDK`
- Branch: `feat/sdk-v1-await` (merged to `main` @ `19eadd5`)
- 目标：
  - 在 SDK 内新增 `await` 包（Broker+Client），提供通用的请求-响应等待语义（timeout/cancel/重复 key 防护/Close 清理）。
  - 匹配粒度统一为：`MsgID + SubProto + Action`；`SendAndAwait` 在 `MsgID==0` 时自动生成非 0 MsgID 并写回 header。
  - 未匹配帧不吞：仍转交给 onUnmatched 回调，避免影响 notify/broadcast 等无等待语义的帧。
- 归档（全局）：`docs/change/2026-02-16_sdk-v1-await.md`

### 2026-02-16 - pending 回落响应继承 MsgID/TraceID（Auth/VarStore）（PR8-Pending-Ids）
- Repo: `repo/MyFlowHub-Server`
- Branch: `fix/pending-msgid-traceid` (merged to `main` @ `47e09c4`)
- 目标：
  - Auth/VarStore 的 pending 回落响应继承原请求的 `MsgID/TraceID`（为后续 SDK v1 按 MsgID Awaiter 铺路）。
  - wire 不变；仅补齐回落链路的 header 透传，并新增单测覆盖关键路径。
- 归档（全局）：`docs/change/2026-02-16_pending-msgid-traceid.md`

### 2026-02-16 - Auth assist 响应收敛（assist_*_resp 回落为 *_resp）（PR7-Auth-AssistResp）
- Repo: `repo/MyFlowHub-Server`
- Branch: `fix/server-auth-assist-resp` (merged to `main` @ `4ca34bd`)
- 目标：
  - 补齐 Auth 对 `assist_register_resp` / `assist_login_resp` 的接收处理，并在中间节点消费后回落为 `register_resp` / `login_resp` 下发（assist 语义不泄漏到客户端）。
  - 新增单测覆盖“assist 响应回落”的最小链路。
- 归档（全局）：`docs/change/2026-02-16_auth-assist-resp.md`

### 2026-02-16 - defaultset build tags（裁切默认子协议集合）（PR6-BuildTags）
- Repo: `repo/MyFlowHub-Server`
- Branch: `refactor/server-defaultset-buildtags` (merged to `main` @ `78e8a25`)
- 目标：
  - 为 `modules/defaultset` 引入 build tags，实现“编译期裁切默认子协议集合”（默认无 tags 行为不变）。
  - 支持 tags：`noauth/novarstore/notopicbus/noexec/noflow/nofile`（指定即禁用对应模块）。
- 归档（全局）：`docs/change/2026-02-16_defaultset-buildtags.md`

### 2026-02-16 - modules/defaultset（默认装配集合解耦）（PR2-10a）
- Repo: `repo/MyFlowHub-Server`
- Branch: `refactor/server-modules-defaultset` (merged to `main` @ `0e5bc88`)
- 目标：
  - 新增 `modules/defaultset` 承载 hub_server 默认启用集合构造策略（为后续裁切/组装预留落点）。
  - `modules.DefaultHub` 委托 `modules/defaultset`，避免 `modules` 直接依赖所有 `subproto/*`。
- 归档（全局）：`docs/change/2026-02-16_modules-defaultset.md`

### 2026-02-16 - Auth 子协议迁移到 subproto/auth（PR2-9a）
- Repo: `repo/MyFlowHub-Server`
- Branch: `refactor/server-subproto-auth` (merged to `main` @ `70c441f`)
- 目标：
  - 将 Auth handler 从 `internal/handler/auth` 迁移到 `subproto/auth`，为后续“可裁切组装 / 拆库复用”铺路。
  - `subproto/auth` 直接依赖 `MyFlowHub-Proto` 协议包（减少对 Server 兼容壳耦合；wire 不变）。
- 归档（全局）：`docs/change/2026-02-16_auth-subproto.md`

### 2026-02-16 - File 子协议迁移到 subproto/file（PR2-8a）
- Repo: `repo/MyFlowHub-Server`
- Branch: `refactor/server-subproto-file` (merged to `main` @ `4897454`)
- 目标：
  - 将 File handler 从 `internal/handler/file` 迁移到 `subproto/file`，为后续“可裁切组装 / 拆库复用”铺路。
  - `subproto/file` 直接依赖 `MyFlowHub-Proto` 协议包（减少对 Server 兼容壳耦合；wire 不变）。
- 归档（全局）：`docs/change/2026-02-16_file-subproto.md`

### 2026-02-16 - Flow 子协议迁移到 subproto/flow（PR2-7a）
- Repo: `repo/MyFlowHub-Server`
- Branch: `refactor/server-subproto-orchestration` (merged to `main` @ `a2fd43a`)
- 目标：
  - 将 Flow handler 从 `internal/handler/flow` 迁移到 `subproto/flow`，为后续“可裁切组装 / 拆库复用”铺路。
  - `subproto/flow` 直接依赖 `MyFlowHub-Proto` 协议包（减少对 Server 兼容壳耦合；wire 不变）。
- 归档（全局）：`docs/change/2026-02-16_flow-subproto.md`

### 2026-02-15 - hub_server 模块装配层（modules）地基（PR2-1a）
- Repo: `repo/MyFlowHub-Server`
- Branch: `refactor/server-modules` (merged to `main` @ `e2c3616`)
- 目标：
  - 将 `cmd/hub_server` 的子协议 handler 清单与装配逻辑抽离为 `modules` 统一入口（为后续“可裁切/可组装、拆库解耦”铺路）。
  - 保持 wire/SubProto/权限语义/handler 业务逻辑不变，并兼容 flow 的 `BindServer`（通过启动后 hook 机制）。
- 归档（全局）：`docs/change/2026-02-15_server-modules.md`

### 2026-02-15 - 子协议去 internal + subproto 基础（PR2-2a）
- Repo: `repo/MyFlowHub-Server`
- Branch: `refactor/server-subproto-public` (merged to `main` @ `3e78ecd`)
- 目标：
  - 引入 `subproto/kit`（共享响应发送/头部克隆等通用能力）。
  - management 子协议迁移到 `subproto/management`，并让 `modules` 使用新路径（小步迁移策略）。
  - 彻底移除 LoginServer（删除 `cmd/login_server` 与 `internal/login_server` 及其文档入口）。
- 归档（全局）：`docs/change/2026-02-15_subproto-foundation.md`

### 2026-02-15 - DefaultForwardHandler 去 internal（subproto/forward）（PR2-3a）
- Repo: `repo/MyFlowHub-Server`
- Branch: `refactor/server-default-forward` (merged to `main` @ `07ede9f`)
- 目标：
  - 将 Dispatcher 默认 fallback `DefaultForwardHandler` 迁移到 `subproto/forward`（为后续裁切/拆库预留落点）。
  - `modules` 与测试切换为新包路径，并删除 `internal/handler` 顶层残留（仅保留 `internal/handler/<sub>`）。
- 归档（全局）：`docs/change/2026-02-15_default-forward-subproto.md`

### 2026-02-15 - VarStore 子协议迁移到 subproto/varstore（PR2-4a）
- Repo: `repo/MyFlowHub-Server`
- Branch: `refactor/server-subproto-varstore` (merged to `main` @ `1eb787b`)
- 目标：
  - 将 VarStore handler 从 `internal/handler/varstore` 迁移到 `subproto/varstore`，为后续裁切/拆库预留落点。
  - `modules` 与测试切换为新包路径；`subproto/varstore` 直接依赖 `MyFlowHub-Proto` 协议包（wire 不变）。
- 归档（全局）：`docs/change/2026-02-15_varstore-subproto.md`

### 2026-02-15 - TopicBus 子协议迁移到 subproto/topicbus（PR2-5a）
- Repo: `repo/MyFlowHub-Server`
- Branch: `refactor/server-subproto-topicbus` (merged to `main` @ `9f2bafa`)
- 目标：
  - 将 TopicBus handler 从 `internal/handler/topicbus` 迁移到 `subproto/topicbus`，为后续裁切/拆库预留落点。
  - `modules`、测试与文档切换为新包路径；`subproto/topicbus` 直接依赖 `MyFlowHub-Proto` 协议包（wire 不变）。
- 归档（全局）：`docs/change/2026-02-15_topicbus-subproto.md`

### 2026-02-15 - Exec 子协议迁移到 subproto/exec（PR2-6a）
- Repo: `repo/MyFlowHub-Server`
- Branch: `refactor/server-subproto-exec` (merged to `main` @ `8a66c05`)
- 目标：
  - 将 Exec handler 从 `internal/handler/exec` 迁移到 `subproto/exec`，为后续裁切/拆库预留落点。
  - `modules` 切换为新包路径；`subproto/exec` 直接依赖 `MyFlowHub-Proto` 协议包（wire 不变）。
- 归档（全局）：`docs/change/2026-02-15_exec-subproto.md`

### 2026-02-15 - MyFlowHub-SDK v0（Session/Transport 基础）（PR2-SDK-1）
- Repo: `repo/MyFlowHub-SDK`
- Branch: `feat/sdk-v0-session` (merged to `main` @ `7531d66`)
- 目标：
  - 新建独立仓库 `MyFlowHub-SDK`（Go module：`github.com/yttydcs/myflowhub-sdk`），作为客户端侧统一 SDK。
  - v0 落地最小可复用能力：`session`（connect/send/readloop）+ `transport`（`action+data` envelope 编解码）。
  - 统一 `trace_id/hop_limit` 默认补齐规则，为 Win/CLI 等客户端后续迁移做准备。
- 归档（全局）：`docs/change/2026-02-15_sdk-v0-session.md`

### 2026-02-15 - Win 接入 MyFlowHub-SDK v0（Session/Transport）（PR2-WIN-1）
- Repo: `repo/MyFlowHub-Win`
- Branch: `refactor/win-sdk-v0` (merged to `main` @ `1686f7f`)
- 目标：
  - Win 侧引入 `github.com/yttydcs/myflowhub-sdk`，收敛客户端底层能力到统一 SDK。
  - 保持 Win 上层 services/bindings API 不变：仅将 `internal/session` 与 `internal/services/transport` 改为薄封装委托 SDK。
- 归档（全局）：`docs/change/2026-02-15_win-use-sdk-v0.md`

### 2026-02-11 - 移除 T14 Packaging and build validation
- 变更：从任务列表移除 `T14 - Packaging and build validation`（原状态：Partially Completed / Windows done; Linux deferred）。
- 原因：当前阶段不纳入 Linux 构建验收；Windows 构建已完成且无需继续占用全局待办。
- 影响：如后续需要补齐 Linux 构建说明与验收，将另起任务与归档记录。

### 2026-02-11 - 协议仓库拆分（Proto）+ Win 上移解耦（PR1）
- Repos / Branch：`refactor/proto-extract`（已合并到主分支）
  - `repo/MyFlowHub-Proto`（main @ `592a43d`）
  - `repo/MyFlowHub-Server`（main @ `d8165d8`）
  - `repo/MyFlowHub-Win`（main @ `ad31199`）
  - `repo/MyFlowHub-Core`（master @ `02fc034`，Core 本轮无实现改动）
- 目标：
  - 将协议定义抽离为独立仓库 `MyFlowHub-Proto`（纯协议字典，wire 不变）。
  - Server 保留 `protocol/*` 兼容壳指向 Proto；并解耦 `flow -> exec`（新增 `internal/broker`）。
  - Win 移除对 Server 的 Go 依赖，仅依赖 Core + Proto（作为应用层）。
- 归档（全局）：
  - `docs/change/2026-02-11_proto-extract_proto.md`
  - `docs/change/2026-02-11_proto-extract_server.md`
  - `docs/change/2026-02-11_proto-extract_win.md`
  - `docs/change/2026-02-11_proto-extract_core.md`
- 注意：
  - `myflowhub-proto` 已创建并推送 `main` 分支；建议在 GitHub 将默认分支切换为 `main`（便于后续协作与发布）。

### 2026-02-09 - MyFlowHub-Server 公共协议包收敛（protocol/*）
- Repo: `repo/MyFlowHub-Server`
- Branch: `feat/public-protocol` (merged to `main` @ `96d35a5`)
- Workflow plan: `repo/MyFlowHub-Server/plan.md`

#### 项目目标
将当前开发中的 `protocol/*` 公共协议包（对外可复用的请求/响应模型、常量、必要校验）纳入版本控制，并将 `internal/handler/*` 的类型定义与引用收敛到 `protocol/*`（保持行为不变），形成可审计、可回放的提交链，供 `MyFlowHub-Win` 通过 `replace` 复用。

#### 非目标 / 约束
- 不新增或改变运行时行为（仅类型/常量/校验辅助导出 + handler 引用迁移）。
- 不在主 worktree（`repo/MyFlowHub-Server`）直接做实现性改动；所有改动在独立 worktree 完成。

#### 任务清单（Checklist）
- S1：迁移主 worktree 的 WIP（含未跟踪 `protocol/**`）→ `go test ./...`
- S2：审核并最小化变更范围（确保无行为修改）→ `go test ./...`
- S3：提交并可审计化（清晰提交信息 + 工作区干净）
- S4：Code Review + 归档（`docs/change/YYYY-MM-DD_public-protocol.md`）

#### 风险与注意事项
- 行尾 LF/CRLF 可能导致噪音 diff；避免无意义格式化。

### 2026-02-09 - MyFlowHub-Win 移除 Fyne + Windows 打包验收（Wails）
- Repo: `repo/MyFlowHub-Win`
- Branch: `refactor/remove-fyne` (merged to `main` @ `d853666`)
- Workflow plan: `repo/MyFlowHub-Win/plan.md`

#### 项目目标
1) 在 `MyFlowHub-Win` 中彻底移除 Fyne（代码路径与依赖），确保仅保留 Wails（Vue3/Vite/Tailwind 前端 + Go 后端服务）作为唯一 UI 实现。  
2) 完成 Windows 打包与冒烟验证文档：能启动并连到 `MyFlowHub-Server`，跑通最小功能链路。

#### 非目标 / 约束
- 不在主 worktree（`repo/MyFlowHub-Win`）直接做实现性改动；所有改动在独立 worktree 完成。
- Linux 构建验收：本轮暂忽略（仅要求 Windows）。

#### 任务清单（Checklist）
- W1：迁移主 worktree WIP（排除 `node_modules/`、`frontend/dist`）→ `go test ./...`
- W2：删除旧 Fyne UI/入口（`internal/ui/**`、`internal/app/**`、`cmd/main.go`）→ `rg fyne.io` 无命中
- W3：移除 Fyne 依赖并 `go mod tidy` → `go test ./...`
- W4：`wails build -platform windows/amd64` + README 冒烟步骤
- W5：Code Review + 归档（`docs/change/YYYY-MM-DD_remove-fyne.md`）

### 2026-02-10 - HeaderTcp v2（32B）big-bang + Core 路由规则统一（Core/Server/Win）
- Repos:
  - `repo/MyFlowHub-Core` (merged to `master` @ `ab36c84`)
  - `repo/MyFlowHub-Server` (merged to `main` @ `01a1df7`)
  - `repo/MyFlowHub-Win` (merged to `main` @ `e2da481`)
- Branch: `refactor/hdrtcp-v2`
- Workflow plans:
  - `repo/MyFlowHub-Core/plan.md`
  - `repo/MyFlowHub-Server/plan.md`
  - `repo/MyFlowHub-Win/plan.md`

#### 关键决策
- HeaderTcp v2 固定头：32B（+8B），`magic=0x4D48`（"MH"）、`ver=2`、`hdr_len=32`；big-bang 后 v1 不再兼容。
- `hop_limit`：默认 16；仅在“发生转发”时递减；耗尽丢弃并告警（防环）。
- `trace_id`：发送侧自动补齐随机 `uint32`；响应继承；转发不改。
- 路由框架规则：`MajorCmd` 必须进 handler（逐跳可见）；`MajorMsg/OK/Err` 走 Core 快速转发。

#### 验收 / 测试
- 三仓均 `go test ./...` 通过。
- 冒烟：启动 Server + 客户端 register/login + management node_echo 端到端收发通过（HeaderTcp v2）。

#### 归档
- `docs/change/2026-02-10_hdrtcp-v2_core.md`
- `docs/change/2026-02-10_hdrtcp-v2_server.md`
- `docs/change/2026-02-10_hdrtcp-v2_win.md`

### 2026-03-05 - MetricsNode Connect 权限修复 + Settings 紧凑化（Android）
- Repo: `repo/MyFlowHub-MetricsNode`
- Branch: `fix/metricsnode-connect-permission-ui` (merged to `main` @ `e9df753`)
- 目标：
  - 修复 Connect 报错 `dial tcp 127.0.0.1:9000: socket: operation not permitted`。
  - 收敛 Settings 页面组件尺寸，提升信息密度。
- 关键变更：
  - Android Manifest 新增 `android.permission.INTERNET`。
  - Settings 页面改为更紧凑布局（间距收敛、输入框高度收敛、`Switch` -> `Checkbox`）。
- Workflow plan（归档）：`docs/plan_archive/plan_archive_2026-03-05_metricsnode-connect-settings-compact.md`
- 归档（全局）：`docs/change/2026-03-04_metricsnode-connect-settings-compact.md`

### 2026-03-05 - Android：VarStore 对齐 Win（Node Vars 查询 + 快捷 Add Watch + 订阅偏好恢复）
- Repo: `repo/MyFlowHub-Android`
- Branch: `feat/android-varstore-nodevars-watch` (merged to `main` @ `c94a790`)
- 目标：
  - 补齐 Android 与 Win 在 VarStore/VarPool 的关键差异。
  - 新增“查询某节点变量名列表 + 一键加入 Watch”能力。
  - 新增订阅偏好持久化与连接后自动恢复订阅。
- 关键变更：
  - `Prefs.kt` 新增 `varstore_sub_prefs` 读写（`name+owner+subscribed`）。
  - `VarStoreScreen.kt` 新增 `Node Vars` 对话框（Owner 查询、搜索、Add Watch）。
  - `VarStoreScreen.kt` 新增订阅目标态持久化与自动恢复（并发限流 4）。
- Workflow plan（归档）：`worktrees/feat-android-varstore-nodevars-watch/MyFlowHub-Android/todo.md`
- 归档（全局）：`docs/change/2026-03-05_android-varstore-nodevars-watch.md`


---
## [2026-03-05] Workflow归档 - MyFlowHub-Win File Console DnD

# Plan - MyFlowHub-Win：File Console 按钮可理解性 + 拖拽放置导入

## Workflow 信息
- 范围：单仓库（`MyFlowHub-Win`）
- 分支：`fix/file-console-dnd-upload`
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-file-console-dnd`
- Base：`main`
- 状态：已完成（待用户确认是否结束 workflow）

## 1) 目标与当前状态
- 目标：
  - 解释并修正 File Console 中 `Up` / `Download` “点击似乎无反应”的可理解性问题。
  - 支持拖拽文件到 File Console 当前目录，完成“直接放置到对应位置”的导入能力（V1：本地节点）。
- 当前状态：
  - `Up` 按钮仅在非根目录可用；根目录时禁用（无明显提示）。
  - `Download` 仅对“远端节点 + 选中文件”可用；其行为是“打开下载任务参数弹窗”，不是立即下载。
  - 当前未启用 Wails 文件拖拽（`EnableFileDrop` 未开启），无拖拽导入链路。

## 2) 任务清单（Checklist）

- [x] `FC-DND-1` 明确按钮行为与禁用原因提示（前端）
  - 目标：为 `Up` / `Download` 增加可理解提示，降低“无反应”感知。
  - 涉及文件：
    - `frontend/src/pages/File.vue`
  - 验收条件：
    - `Up` 在根目录显示“已在根目录”语义提示；
    - `Download` 在禁用时可看出触发条件（需远端文件被选中）。
  - 测试点：
    - 本地节点、远端节点、目录/文件选中状态切换时按钮提示变化正确。
  - 回滚点：
    - 回滚 `File.vue` 中按钮提示相关改动。

- [x] `FC-DND-2` 启用并接入 Wails 文件拖拽（前端 + 启动配置）
  - 目标：允许将外部文件拖入 File Console 列表区域。
  - 涉及文件：
    - `main.go`
    - `frontend/src/pages/File.vue`
  - 验收条件：
    - 拖拽到目录列表区域可触发导入逻辑；
    - 仅本地节点允许导入，远端节点拖拽给出明确提示。
  - 测试点：
    - 拖拽区域高亮状态可见；
    - 回调收到文件路径并进入导入流程。
  - 回滚点：
    - 回滚 `main.go` 的 DragAndDrop 配置；
    - 回滚 `File.vue` 的 OnFileDrop 监听逻辑。

- [x] `FC-DND-3` 后端导入能力（安全默认 + 错误处理）
  - 目标：新增 `ImportLocalFiles`，将拖拽源文件复制到 `BaseDir/currentDir`。
  - 涉及文件：
    - `internal/services/file/local.go`
    - `frontend/src/stores/file.ts`
  - 验收条件：
    - 仅接受常规文件；目录/非法路径被跳过并返回原因；
    - 目标路径受 `fileSanitizeDir/fileResolvePaths` 约束，禁止越界；
    - 默认不覆盖同名文件（安全默认），可返回跳过列表；
    - 成功后刷新目录列表。
  - 测试点：
    - 正常导入、重复文件跳过、目标目录非法、目录拖入跳过。
  - 回滚点：
    - 回滚新增导入 API 与前端调用。

- [x] `FC-DND-4` 单测与回归验证
  - 目标：补充导入逻辑关键路径测试，执行回归。
  - 涉及文件：
    - `internal/services/file/import_test.go`（如需新增）
  - 验收条件：
    - 新增测试通过；
    - 现有 `go test ./...` 不回归。
  - 测试点：
    - 覆盖导入成功/覆盖策略/非法输入。
  - 回滚点：
    - 回滚新增测试文件。

- [x] `FC-DND-5` Code Review（3.3）与归档（4）
  - 目标：按要求完成逐项审查并归档到 `docs/change`。
  - 涉及文件：
    - `docs/change/2026-03-05_win-file-console-dnd-upload.md`
  - 验收条件：
    - 评审结论完整（通过/不通过）；
    - 归档文档包含任务映射、权衡、验证与回滚方案。
  - 回滚点：
    - 文档层无需代码回滚。

## 3) 依赖与风险
- 依赖：
  - Wails v2 拖拽能力（`options.DragAndDrop.EnableFileDrop` + 前端 `OnFileDrop`）。
- 风险：
  - 平台差异导致拖拽事件触发行为不同；需在 Windows 运行态冒烟。
  - 大文件复制耗时导致 UI 感知延迟；V1 先保证正确性与可观测 toast，后续可扩展进度事件。

## 4) 注意事项
- 本次拖拽导入范围限定为“本地节点当前目录”；远端节点不直接写入，避免越权与协议复杂度上升。
- 默认不覆盖同名文件，避免误覆盖；若后续需要覆盖策略，回到本计划新增任务确认后再做。



---
## [2026-03-05] Workflow归档 - MyFlowHub-SubProto file mkdir

# Plan - MyFlowHub-SubProto：file 子协议新增 mkdir 操作

## Workflow 信息
- 仓库：`MyFlowHub-SubProto`
- 分支：`feat/file-mkdir-op`
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-SubProto-file-mkdir`
- Base：`main`
- 当前状态：已完成（待你确认是否结束 workflow）

## 项目目标与当前状态
- 目标：
  - 在 `SubProto=5 file` 中新增 `action=write, op=mkdir`，支持创建目录。
  - 保持现有 `offer/list/pull/read_text` 行为不变。
- 现状：
  - `handleWriteRequest` 仅支持 `op=offer`；
  - 无显式创建目录控制操作。

## 可执行任务清单（Checklist）

- [x] `MKDIR-1` 扩展 file 操作常量与写请求分发
  - 目标：识别 `op=mkdir` 并纳入 write 路由。
  - 涉及文件：
    - `file/types.go`
    - `file/handler.go`
  - 验收条件：
    - `handleWriteRequest` 能区分 `offer/mkdir`；
    - `mkdir` 走 `file.write` 权限链路。
  - 测试点：
    - 非法 op 仍返回 `invalid op`；
    - 既有 `offer` 行为无回归。
  - 回滚点：
    - 回滚上述文件对 `op=mkdir` 的分发逻辑。

- [x] `MKDIR-2` 新增本地 mkdir 处理逻辑（安全默认）
  - 目标：实现 `handleMkdirLocal`，仅允许在 `BaseDir` 下创建目录。
  - 涉及文件：
    - `file/handler.go`
    - `file/path.go`（如需复用/补充路径辅助函数）
  - 验收条件：
    - `dir/name` 均做校验；
    - 同名文件冲突返回明确错误；
    - 成功返回 `code=1`。
  - 测试点：
    - 成功创建；
    - 已存在目录幂等处理；
    - 非法输入拦截。
  - 回滚点：
    - 回滚 `handleMkdirLocal` 及调用路径。

- [x] `MKDIR-3` 补充测试覆盖
  - 目标：新增 mkdir 关键路径测试，确保回归可控。
  - 涉及文件：
    - `file/handler_mkdir_test.go`（新增）
  - 验收条件：
    - go test 通过；
    - 关键边界（合法/非法/冲突）被覆盖。
  - 回滚点：
    - 删除新增测试文件。

- [x] `MKDIR-4` 文档与变更归档
  - 目标：归档本次协议变更（请求/响应语义、错误码、兼容性）。
  - 涉及文件：
    - `docs/change/2026-03-05_file-mkdir-op.md`（新增）
  - 验收条件：
    - 文档包含目标、变更细节、测试结果、回滚方案。
  - 回滚点：
    - 仅文档回滚，无运行时影响。

## 依赖关系
- `MKDIR-1` -> `MKDIR-2` -> `MKDIR-3` -> `MKDIR-4`

## 风险与注意事项
- `WriteReq` 原先为 `offer` 设计，`mkdir` 复用时需明确“非相关字段忽略”。
- 目录创建应保持路径安全，禁止绝对路径/`..` 越界。
- 必须保证 wire 兼容：不破坏既有字段与 action 命名。



---
## [2026-03-05] Workflow归档 - MyFlowHub-Server file v0.1.2

# Plan - MyFlowHub-Server：升级 subproto/file 到 v0.1.2（mkdir）

## Workflow 信息
- 仓库：`MyFlowHub-Server`
- 分支：`chore/server-bump-file-v0.1.2`
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Server-file-v012`
- Base：`main`
- 当前状态：已完成（待你确认是否结束 workflow）

## 项目目标与当前状态
- 目标：
  - 将 Server 依赖 `github.com/yttydcs/myflowhub-subproto/file` 升级到 `v0.1.2`；
  - 同步 Server 协议文档 `docs/5-file.md`，补充 `op=mkdir` 说明；
  - 执行回归验证，确保构建与测试不回退。
- 现状：
  - `go.mod` 依赖仍为 `v0.1.1`；
  - `docs/5-file.md` 仅覆盖 `pull/list/read_text/offer`。

## 可执行任务清单（Checklist）

- [x] `SRV-FILE-1` 升级依赖版本
  - 目标：`go.mod/go.sum` 切换到 `myflowhub-subproto/file v0.1.2`。
  - 涉及文件：
    - `go.mod`
    - `go.sum`
  - 验收条件：
    - `go list -m github.com/yttydcs/myflowhub-subproto/file` 显示 `v0.1.2`。
  - 测试点：
    - `go mod tidy` / `go test` 可通过（在当前 workspace 条件下）。
  - 回滚点：
    - 版本回退到 `v0.1.1`。

- [x] `SRV-FILE-2` 同步协议文档
  - 目标：在 `docs/5-file.md` 新增 `op=mkdir` 语义、请求/响应与权限说明。
  - 涉及文件：
    - `docs/5-file.md`
  - 验收条件：
    - 文档与 `v0.1.2` 行为一致；
    - 不修改既有 action/SubProto 编号语义。
  - 回滚点：
    - 回滚文档改动。

- [x] `SRV-FILE-3` 回归验证 + 归档
  - 目标：执行测试并生成变更归档。
  - 涉及文件：
    - `docs/change/2026-03-05_bump-subproto-file-v0.1.2.md`
  - 验收条件：
    - 测试结果明确记录；
    - 归档文档包含任务映射、影响与回滚。
  - 回滚点：
    - 文档层可独立回滚。

## 依赖与风险
- 依赖：SubProto 仓库 `file/v0.1.2` tag 已可解析。
- 风险：网络/代理不可达会导致 `go get` 失败；若出现，可改为本地 workspace 验证并在文档标注。


---
## [2026-03-05] Workflow归档 - MyFlowHub-Android 设备树异常可观测性修复

# TODO - Android：设备树跨级请求异常（10 -> 1）定位与修复

## Workflow 信息
- Repo：`MyFlowHub-Android`
- 分支：`fix/android-devices-listnodes-error`
- Worktree：`d:\project\MyFlowHub3\repo\MyFlowHub-Android\worktrees\fix-android-devices-listnodes-error\MyFlowHub-Android`
- 参考规范：`d:\project\MyFlowHub3\guide.md`

## 项目目标与当前状态
- 目标：
  - 修复 Android 端设备树请求失败时仅显示 `java.lang.reflect.InvocationTargetException` 的可观测性问题。
  - 给出 10 -> 1 跨级请求异常的可验证定位路径，避免“看不见根因”。
- 当前状态：
  - `DevicesScreen` 的错误展示直接使用 `Throwable.message`，反射调用场景会丢失真实 cause。
  - `GoClientBridge` 直接 `Method.invoke`，未统一解包 `InvocationTargetException`。

## 可执行任务清单（Checklist）

- [x] DEVTREE-1：梳理异常链路与输出规范
  - 目标：明确 `listNodes` 从 UI -> 反射 -> Go 的异常传播路径与期望错误文案。
  - 涉及模块/文件：
    - `app/src/main/java/com/myflowhub/android/GoReflect.kt`
    - `app/src/main/java/com/myflowhub/android/GoClientBridge.kt`
    - `app/src/main/java/com/myflowhub/android/ui/DevicesScreen.kt`
  - 验收条件：
    - 明确“优先展示 root cause；必要时补充 lastError”策略。
  - 测试点：
    - 人工审查异常分支逻辑覆盖（反射异常、空 message、普通异常）。
  - 回滚点：
    - 回退上述文件的异常处理改动。

- [x] DEVTREE-2：实现反射调用统一解包（核心修复）
  - 目标：在 Android 端统一处理 `InvocationTargetException`，保留真实错误上下文。
  - 涉及模块/文件：
    - `app/src/main/java/com/myflowhub/android/GoReflect.kt`
    - `app/src/main/java/com/myflowhub/android/GoClientBridge.kt`
  - 验收条件：
    - Go 方法抛错时，UI 可拿到具体 cause 信息（如 `not found` / `timeout` / `not connected`），不再只看到 `InvocationTargetException`。
    - 不改变正常成功路径性能（仅异常路径增加处理）。
  - 测试点：
    - 人工路径：构造失败请求，确认文案包含根因。
  - 回滚点：
    - 回退 `GoReflect` 的 invoke 辅助函数与 `GoClientBridge` 调用点。

- [x] DEVTREE-3：设备树 UI 错误展示增强
  - 目标：统一 `DevicesScreen` 错误文案格式，追加可诊断信息（必要时拼接 `lastError`）。
  - 涉及模块/文件：
    - `app/src/main/java/com/myflowhub/android/ui/DevicesScreen.kt`
  - 验收条件：
    - 设备树加载失败时可直接看到可操作错误（而非反射包装类名）。
  - 测试点：
    - 人工路径：Root Load 失败、子节点展开失败、NodeInfo/Config 失败时文案一致。
  - 回滚点：
    - 回退 `DevicesScreen.kt` 文案处理改动。

- [x] DEVTREE-4：验证、Code Review、归档
  - 目标：完成质量闭环与审计文档。
  - 涉及模块/文件：
    - `docs/change/2026-03-05_android-devices-listnodes-error.md`
  - 验收条件：
    - 关键编译/静态检查通过。
    - 完成 3.3 Code Review（需求覆盖、架构、性能、可读性、扩展性、稳定性、安全、测试）。
    - 完成 docs/change 归档。
  - 测试点：
    - `./gradlew.bat :app:assembleDebug`
  - 回滚点：
    - 回退本分支提交，或按任务粒度回滚。

## 依赖关系
- DEVTREE-1 -> DEVTREE-2 -> DEVTREE-3 -> DEVTREE-4

## 风险与注意事项
- 仅修复“错误可观测性”不会自动修复网络拓扑/认证问题；但可显著降低排障成本。
- 不能在主 repo 工作区改实现；仅在当前 worktree 变更。
- 需避免在成功路径引入额外 I/O 或重计算。

## 当前执行状态
- 已完成：DEVTREE-1、DEVTREE-2、DEVTREE-3、DEVTREE-4
- 进行中：无
- 待完成：无


---
## [2026-03-05] Workflow归档 - MyFlowHub-Win File Console mkdir

# Plan - MyFlowHub-Win：File Console 新建文件夹（mkdir）

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`feat/file-console-mkdir`
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-mkdir`
- Base：`main`
- 当前状态：已完成（待你确认是否结束 workflow）

## 项目目标与当前状态
- 目标：
  - 在 File Console 增加“新建文件夹”入口；
  - 前端可输入目录名并调用后端创建；
  - 本地/远端节点均支持（按 file 子协议 `write(op=mkdir)`）。
- 当前状态：
  - `New Folder` 已支持，目录创建成功后自动刷新列表；
  - 本地节点走本地 mkdir，远端节点走 `write(op=mkdir)` 并 await 响应。

## 可执行任务清单（Checklist）

- [x] `WIN-MKDIR-1` 后端 FileService 新增 mkdir API
  - 目标：新增 `CreateDirSimple`（必要校验 + await 响应）。
  - 涉及文件：
    - `internal/services/file/service.go`
    - `internal/services/file/local.go`（如需本地 helper）
  - 验收条件：
    - 非法输入返回明确错误；
    - 目标节点返回成功后接口返回 nil；
    - 失败时透传可读错误。
  - 测试点：
    - 本地创建、非法名称、目标缺失。
  - 回滚点：
    - 回滚新增 API 与调用路径。

- [x] `WIN-MKDIR-2` 前端 store 接入 mkdir 调用
  - 目标：`useFileStore` 新增 `createDir` 方法封装 Wails binding。
  - 涉及文件：
    - `frontend/src/stores/file.ts`
  - 验收条件：
    - 页面可通过 store 调用创建目录。
  - 回滚点：
    - 回滚 store 新增方法。

- [x] `WIN-MKDIR-3` File Console UI 增加“New Folder”
  - 目标：新增按钮 + 弹窗输入目录名 + 成功后刷新列表。
  - 涉及文件：
    - `frontend/src/pages/File.vue`
  - 验收条件：
    - 可创建目录并在列表看到；
    - 输入为空/非法有提示；
    - 不影响既有 Up/Download/Offer/拖拽逻辑。
  - 测试点：
    - 根目录/子目录创建；
    - 已存在目录行为（应成功或合理提示）。
  - 回滚点：
    - 回滚页面新增弹窗与按钮。

- [x] `WIN-MKDIR-4` 回归验证与归档
  - 目标：执行 Go 测试并补充变更归档文档。
  - 涉及文件：
    - `docs/change/2026-03-05_win-file-console-mkdir.md`
  - 验收条件：
    - `go test ./...` 通过；
    - 文档包含任务映射、验证与回滚方案。

## 风险与注意事项
- `CreateDir` 需与现有 `fileSanitizeDir/fileSanitizeName` 一致，避免目录穿越。
- 若远端节点尚未升级到 `subproto/file v0.1.2`，`op=mkdir` 可能返回 `invalid op`，需给前端友好错误提示。


---
## [2026-03-05] Workflow归档 - MyFlowHub-SubProto up_login SenderPub 路由修复

# TODO - SubProto：修复跨级 VarStore 目标节点路由丢失（2 -> 10 set 返回 code=4）

## Workflow 信息
- Repo：`MyFlowHub-SubProto`
- 分支：`fix/subproto-uplogin-sender-pub`
- Worktree：`d:\project\MyFlowHub3\repo\MyFlowHub-SubProto\worktrees\fix-subproto-uplogin-sender-pub\MyFlowHub-SubProto`
- 触发问题：拓扑 `1 -> (2,9) -> 10`，由 2(Win) 对 owner=10 执行 VarStore set 报 `code=4 not found`。

## 项目目标与当前状态
- 目标：
  - 修复 node10 向上登录传播（up_login）链路中 sender 公钥携带错误，避免父节点无法校验 sender 签名进而丢失路由索引。
  - 让 `2 -> 1 -> 9 -> 10` 的 VarStore set 能正确命中 owner=10 路由。
- 当前状态（已定位）：
  - `auth/sendUpLogin` 当前把 `SenderPub` 错误设置为“登录设备(10)公钥”，而不是“sender 节点(9)公钥”；
  - 当父节点缺少 sender 已信任公钥时，会依赖 `SenderPub` 回填校验，导致校验失败并不建立 `node10` 路由。

## 可执行任务清单（Checklist）

- [x] UPLOGIN-1：修复 sendUpLogin 的 SenderPub 取值
  - 目标：`SenderPub` 必须携带 sender 节点公钥（`h.nodePubB64`），而非被登录设备公钥。
  - 涉及文件：
    - `auth/actions_up_login.go`
  - 验收条件：
    - 构造的 `up_login` 数据中，`PubKey` 仍是登录节点公钥；`SenderPub` 为 sender 节点公钥。
  - 测试点：
    - 单测断言 `SenderPub != PubKey`（在不同公钥输入下）。
  - 回滚点：
    - 回滚 `actions_up_login.go` 本任务变更。

- [x] UPLOGIN-2：补充单测覆盖关键构包逻辑
  - 目标：防止未来回归把 `SenderPub` 再次误绑到目标节点公钥。
  - 涉及文件：
    - `auth/actions_up_login_test.go`（新增）
  - 验收条件：
    - `go test ./auth -count=1 -p 1` 通过；
    - 用例覆盖 `SenderPub` 与 `PubKey` 字段语义。
  - 回滚点：
    - 删除新增测试文件。

- [x] UPLOGIN-3：验证、Code Review、归档
  - 目标：完成质量闭环与审计文档。
  - 涉及文件：
    - `docs/change/2026-03-05_auth-up-login-sender-pub-fix.md`
  - 验收条件：
    - 测试命令通过并记录；
    - Review 清单逐项给出通过/不通过结论；
    - docs/change 归档完整（背景、任务映射、权衡、验证、回滚）。
  - 测试点：
    - `go test ./auth -count=1 -p 1`
  - 回滚点：
    - 按提交回滚或按任务粒度回滚。

## 依赖关系
- `UPLOGIN-1 -> UPLOGIN-2 -> UPLOGIN-3`

## 风险与注意事项
- 本修复影响 Auth 上行校验与路由传播，属于链路关键路径；必须保持 wire 字段不变，仅修正字段值来源。
- 不改 VarStore/Win 协议调用参数，避免引入跨仓耦合变更。

## 当前执行状态
- 已完成：UPLOGIN-1、UPLOGIN-2、UPLOGIN-3
- 进行中：无
- 待完成：无

---
## [2026-03-05] Workflow归档 - MyFlowHub-Server 依赖升级 auth v0.1.1

# TODO - Server：升级 subproto/auth 到 v0.1.1（修复跨级路由传播）

## Workflow 信息
- Repo：`MyFlowHub-Server`
- 分支：`chore/server-bump-auth-v0.1.1`
- Worktree：`d:\project\MyFlowHub3\repo\MyFlowHub-Server\worktrees\chore-server-bump-auth-v0.1.1\MyFlowHub-Server`
- 上游版本：`github.com/yttydcs/myflowhub-subproto/auth v0.1.1`

## 项目目标与当前状态
- 目标：
  - 将 Server 的 `myflowhub-subproto/auth` 依赖从 `v0.1.0` 升级到 `v0.1.1`，纳入 `up_login SenderPub` 修复。
- 当前状态：
  - `go.mod` 仍为 `auth v0.1.0`。

## 可执行任务清单（Checklist）

- [x] SRVAUTH-1：升级依赖版本
  - 目标：`go.mod/go.sum` 对齐 `auth v0.1.1`。
  - 涉及文件：
    - `go.mod`
    - `go.sum`
  - 验收条件：
    - `go list -m github.com/yttydcs/myflowhub-subproto/auth` 输出 `v0.1.1`。
  - 测试点：
    - `GOWORK=off go list -m github.com/yttydcs/myflowhub-subproto/auth`
  - 回滚点：
    - 回退 `go.mod/go.sum` 到 `v0.1.0`。

- [x] SRVAUTH-2：最小回归验证
  - 目标：保证升级后 Server 模块可构建/测试。
  - 涉及文件：
    - 无新增功能文件。
  - 验收条件：
    - `GOWORK=off go test ./... -count=1 -p 1` 通过。
  - 回滚点：
    - 回退依赖升级提交。

- [x] SRVAUTH-3：Code Review + 归档
  - 目标：完成审查闭环与变更归档。
  - 涉及文件：
    - `docs/change/2026-03-05_server-bump-subproto-auth-v0.1.1.md`
  - 验收条件：
    - 文档包含任务映射、验证结果、影响与回滚。
  - 回滚点：
    - 回滚文档提交。

## 依赖关系
- `SRVAUTH-1 -> SRVAUTH-2 -> SRVAUTH-3`

## 风险与注意事项
- 若远端 tag 解析异常，会阻塞升级并需回到上游发布流程核对。
- 仅升级依赖，不混入功能改动。

## 当前执行状态
- 已完成：SRVAUTH-1、SRVAUTH-2、SRVAUTH-3
- 进行中：无
- 待完成：无

---
## [2026-03-05] Workflow归档 - MyFlowHub-Android hubmobile 依赖升级 auth v0.1.1

# TODO - Android(hubmobile)：升级 subproto/auth 到 v0.1.1（下游依赖对齐）

## Workflow 信息
- Repo：`MyFlowHub-Android`
- 分支：`chore/android-bump-auth-v0.1.1`
- Worktree：`d:\project\MyFlowHub3\repo\MyFlowHub-Android\worktrees\chore-android-bump-auth-v0.1.1\MyFlowHub-Android`
- 升级目标：`hubmobile/go.mod` 中 `github.com/yttydcs/myflowhub-subproto/auth`（indirect）

## 项目目标与当前状态
- 目标：
  - 对齐 Android hubmobile 的 indirect 依赖到 `auth v0.1.1`，避免下游模块版本滞后。
- 当前状态：
  - `hubmobile/go.mod` 中 `auth` 仍为 `v0.1.0 // indirect`。

## 可执行任务清单（Checklist）

- [x] ANDAUTH-1：升级 hubmobile 依赖版本
  - 目标：`hubmobile/go.mod/go.sum` 对齐 `auth v0.1.1`（并按依赖求解结果同步 `file v0.1.2`）。
  - 涉及文件：
    - `hubmobile/go.mod`
    - `hubmobile/go.sum`
  - 验收条件：
    - `cd hubmobile && GOWORK=off go list -m github.com/yttydcs/myflowhub-subproto/auth` 输出 `v0.1.1`。
  - 回滚点：
    - 回退 `hubmobile/go.mod/go.sum`。

- [x] ANDAUTH-2：最小验证
  - 目标：确认 hubmobile 模块在新依赖下可通过测试。
  - 验收条件：
    - `cd hubmobile && GOWORK=off go test ./... -count=1 -p 1` 通过。
  - 回滚点：
    - 回退依赖升级提交。

- [x] ANDAUTH-3：Code Review + 归档
  - 目标：完成审查闭环与变更归档。
  - 涉及文件：
    - `docs/change/2026-03-05_android-hubmobile-bump-subproto-auth-v0.1.1.md`
  - 验收条件：
    - 文档包含任务映射、验证结果、影响与回滚。
  - 回滚点：
    - 回滚文档提交。

## 依赖关系
- `ANDAUTH-1 -> ANDAUTH-2 -> ANDAUTH-3`

## 风险与注意事项
- `hubmobile` 使用 `replace` 指向本地 Server，依赖解析时需显式 `GOWORK=off` 保持可审计结果。
- 仅升级依赖，不改 Android UI/业务逻辑。

## 当前执行状态
- 已完成：ANDAUTH-1、ANDAUTH-2、ANDAUTH-3
- 进行中：无
- 待完成：无

---
## [2026-03-06] Workflow归档 - MyFlowHub-SubProto VarStore跨层转发修复

# TODO - SubProto：VarStore 跨层（1->9->10）set/get 路由修复

## Workflow 信息
- Repo：`MyFlowHub-SubProto`
- 分支：`fix/varstore-crosshop-routing`
- Worktree：`d:\project\MyFlowHub3\worktrees\fix-subproto-varstore-crosshop-routing`
- 触发问题：拓扑 `1 -> (2,9) -> 10`，Win(2) 对 owner=10 执行 var set 返回 `not found (code=4)`；同层（1直连10）可用。

## 项目目标与当前状态
- 目标：
  - 修复 VarStore 在多跳拓扑下的目标转发能力，确保跨层 owner 可达。
  - 保持同层行为与现有 wire 协议兼容。
- 当前状态（已定位）：
  - `varstore` 当前未像 management 一样处理 `hdr.TargetID != local` 的命令转发；
  - 跨层 notify/command 在中间节点可能被本地消费，导致链路中断与值不一致。

## 可执行任务清单（Checklist）

- [x] VSROUTE-1：补齐 VarStore 的按 target 转发入口
  - 目标：当 `MajorCmd` 且 `TargetID != local` 时，按路由索引转发到下一跳（或父节点），本地不消费。
  - 涉及文件：
    - `varstore/varstore.go`
  - 验收条件：
    - 多跳场景下，中间节点不再错误本地处理目标非本机的 varstore 命令。
  - 测试点：
    - 新增单测覆盖 target-forward 成功/未命中分支。
  - 回滚点：
    - 回滚 `varstore/varstore.go` 中新增的 forward 入口。

- [x] VSROUTE-2：补充回归测试覆盖 1->9->10 关键路径
  - 目标：锁定跨层 set/get/revoke 至少一条关键路径不回归。
  - 涉及文件：
    - `varstore/*_test.go`（新增或修改）
  - 验收条件：
    - `go test ./... -count=1 -p 1` 通过。
  - 回滚点：
    - 删除新增测试并回滚相关断言。

- [x] VSROUTE-3：Code Review、归档与发布准备
  - 目标：完成审计闭环，并给出下游依赖升级建议。
  - 涉及文件：
    - `docs/change/2026-03-05_varstore-crosshop-routing.md`
  - 验收条件：
    - Review 清单逐项通过；
    - docs/change 完整记录背景、任务映射、验证、回滚。
  - 回滚点：
    - 回滚文档与版本发布动作。

## 依赖关系
- `VSROUTE-1 -> VSROUTE-2 -> VSROUTE-3`

## 风险与注意事项
- 转发逻辑位于子协议关键路径，必须避免回环与 hop-limit 违规。
- 不改协议字段，仅补齐行为；确保与既有 Win/Android 客户端兼容。

---
## [2026-03-06] Workflow归档 - MyFlowHub-Server 依赖升级 varstore v0.1.1

# TODO - Server：升级 subproto/varstore 到 v0.1.1（跨层转发修复）

## Workflow 信息
- Repo：`MyFlowHub-Server`
- 分支：`chore/server-bump-varstore-v0.1.1`
- Worktree：`d:\project\MyFlowHub3\worktrees\chore-server-bump-varstore-v0.1.1`
- 上游版本：`github.com/yttydcs/myflowhub-subproto/varstore v0.1.1`

## 项目目标与当前状态
- 目标：
  - 将 Server 的 `myflowhub-subproto/varstore` 依赖从 `v0.1.0` 升级到 `v0.1.1`，纳入跨层 target 转发与 owner 路由自愈修复。
- 当前状态：
  - `go.mod` 仍为 `varstore v0.1.0`。

## 可执行任务清单（Checklist）

- [x] SRVVAR-1：升级依赖版本
  - 目标：`go.mod/go.sum` 对齐 `varstore v0.1.1`。
  - 涉及文件：
    - `go.mod`
    - `go.sum`
  - 验收条件：
    - `go list -m github.com/yttydcs/myflowhub-subproto/varstore` 输出 `v0.1.1`。
  - 回滚点：
    - 回退 `go.mod/go.sum` 到 `v0.1.0`。

- [x] SRVVAR-2：最小回归验证
  - 目标：保证升级后 Server 模块可构建/测试。
  - 验收条件：
    - `GOWORK=off go test ./... -count=1 -p 1` 通过。
  - 回滚点：
    - 回退依赖升级提交。

- [x] SRVVAR-3：Code Review + 归档
  - 目标：完成审查闭环与变更归档。
  - 涉及文件：
    - `docs/change/2026-03-05_server-bump-subproto-varstore-v0.1.1.md`
  - 验收条件：
    - 文档包含任务映射、验证结果、影响与回滚。

## 依赖关系
- `SRVVAR-1 -> SRVVAR-2 -> SRVVAR-3`

## 风险与注意事项
- 仅升级依赖，不混入功能改动。

---
## [2026-03-06] Workflow归档 - MyFlowHub-Android hubmobile 依赖升级 varstore v0.1.1

# TODO - Android(hubmobile)：升级 subproto/varstore 到 v0.1.1（下游依赖对齐）

## Workflow 信息
- Repo：`MyFlowHub-Android`
- 分支：`chore/android-bump-varstore-v0.1.1`
- Worktree：`d:\project\MyFlowHub3\worktrees\chore-android-bump-varstore-v0.1.1`
- 升级目标：`hubmobile/go.mod` 中 `github.com/yttydcs/myflowhub-subproto/varstore`（indirect）

## 项目目标与当前状态
- 目标：
  - 对齐 Android hubmobile 的 indirect 依赖到 `varstore v0.1.1`，纳入跨层转发修复。
- 当前状态：
  - `hubmobile/go.mod` 中 `varstore` 仍为 `v0.1.0 // indirect`。

## 可执行任务清单（Checklist）

- [x] ANDVAR-1：升级 hubmobile 依赖版本
  - 目标：`hubmobile/go.mod/go.sum` 对齐 `varstore v0.1.1`。
  - 涉及文件：
    - `hubmobile/go.mod`
    - `hubmobile/go.sum`
  - 验收条件：
    - `cd hubmobile && GOWORK=off go list -m github.com/yttydcs/myflowhub-subproto/varstore` 输出 `v0.1.1`。

- [x] ANDVAR-2：最小验证
  - 目标：确认 hubmobile 模块在新依赖下可通过测试。
  - 验收条件：
    - `cd hubmobile && GOWORK=off go test ./... -count=1 -p 1` 通过。

- [x] ANDVAR-3：Code Review + 归档
  - 目标：完成审查闭环与变更归档。
  - 涉及文件：
    - `docs/change/2026-03-05_android-hubmobile-bump-subproto-varstore-v0.1.1.md`
  - 验收条件：
    - 文档包含任务映射、验证结果、影响与回滚。

## 依赖关系
- `ANDVAR-1 -> ANDVAR-2 -> ANDVAR-3`

## 风险与注意事项
- `hubmobile` 使用 `replace` 指向本地 Server，依赖解析时需显式 `GOWORK=off`。
- 仅升级依赖，不改 Android UI/业务逻辑。

---
## [2026-03-06] Workflow归档 - MyFlowHub-Win Send Offer远端目录可调与树状节点选择

# TODO - Win File Console：Offer 支持 remoteDir(req.dir) + 目标节点树状选择

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`feat/file-console-offer-dir-node-picker`
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-offer-dir-node-picker`
- 归档文档：`docs/change/2026-03-06_win-file-offer-dir-node-picker.md`

## 可执行任务清单（Checklist）
- [x] OFFER-DIR-1：后端新增 `StartOfferToDir`（sourceDir 与 remoteDir 分离）
- [x] OFFER-DIR-2：`RetryTask` 适配分离目录语义
- [x] OFFER-DIR-3：前端 store 支持 remoteDir 参数并兼容旧 binding
- [x] OFFER-DIR-4：Offer 弹窗新增 `Remote Dir` 输入并默认当前目录
- [x] OFFER-DIR-5：目标节点改为树状选择组件（可展开/重试/选中）
- [x] OFFER-DIR-6：回归验证、Code Review、归档

## 验证结果
- `GOWORK=off go test ./...`：通过。
- `frontend npm run build`：受项目环境缺失 `wailsjs` 生成文件影响失败（非本次改动引入）。

## 当前执行状态
- 已完成：OFFER-DIR-1 ~ OFFER-DIR-6
- 进行中：无
- 待完成：无

---
## [2026-03-06] Workflow归档 - MyFlowHub-Win Offer目标节点输入框与树选择弹窗

# TODO - Win File Console：Target Node 输入框 + Select按钮 + 树弹窗回填

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`feat/file-offer-target-picker-dialog`
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-offer-target-picker-dialog`
- 归档文档：`docs/change/2026-03-06_win-file-offer-target-picker-dialog.md`

## 可执行任务清单（Checklist）
- [x] OFFER-PICKER-UI-1：Offer 目标节点改为输入框 + 右侧 Select 按钮
- [x] OFFER-PICKER-UI-2：点击 Select 打开树形弹窗并在点击节点后回填 NodeID
- [x] OFFER-PICKER-UI-3：回归验证、Code Review、归档

## 验证结果
- `GOWORK=off go test ./...`：通过。
- `frontend npm run build`：受项目环境缺失 `wailsjs` 生成文件影响失败（非本次改动引入）。

## 当前执行状态
- 已完成：OFFER-PICKER-UI-1 ~ OFFER-PICKER-UI-3
- 进行中：无
- 待完成：无

---
## [2026-03-06] Workflow归档 - MyFlowHub-Win 文件节点选择确认回填优化

# TODO - Win File Console：Nodes顶部图标化 + Select节点选择 + Confirm回填

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`feat/file-node-picker-confirm`
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-node-picker-confirm`
- 归档文档：`docs/change/2026-03-06_win-file-node-picker-confirm.md`

## 可执行任务清单（Checklist）
- [x] NODE-PICKER-1：左侧 Nodes 顶部 Add 图标化，并新增 Select 按钮
- [x] NODE-PICKER-2：Offer 目标树选择改为“点击选中 + Confirm回填”
- [x] NODE-PICKER-3：移除说明文案与 selected target，Close 改无边框图标
- [x] NODE-PICKER-4：左侧 Select 选择器接入并 Confirm 后切换节点
- [x] NODE-PICKER-5：回归验证、Code Review、归档

## 验证结果
- `GOWORK=off go test ./...`：通过。
- `frontend npm run build`：受项目环境缺失 `wailsjs` 生成文件影响失败（非本次改动引入）。

## 当前执行状态
- 已完成：NODE-PICKER-1 ~ NODE-PICKER-5
- 进行中：无
- 待完成：无

---
## [2026-03-06] Workflow归档 - MyFlowHub-Win Add Remote Node 选择器回填

# TODO - Win File Console：Add Remote Node 输入框旁 Select 选择回填

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`feat/file-add-node-select-picker`
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-add-node-select-picker`
- 归档文档：`docs/change/2026-03-06_win-file-add-node-select-picker.md`

## 可执行任务清单（Checklist）
- [x] ADD-NODE-SELECT-1：Add Remote Node 输入区增加 Select 按钮
- [x] ADD-NODE-SELECT-2：树选择弹窗 + Confirm 回填 Node ID
- [x] ADD-NODE-SELECT-3：回归验证、Code Review、归档

## 验证结果
- `GOWORK=off go test ./...`：通过。
- `frontend npm run build`：受项目环境缺失 `wailsjs` 生成文件影响失败（非本次改动引入）。

## 当前执行状态
- 已完成：ADD-NODE-SELECT-1 ~ ADD-NODE-SELECT-3
- 进行中：无
- 待完成：无

---
## [2026-03-06] Workflow归档 - MyFlowHub-Win 移除Nodes顶部Select入口

# TODO - Win File Console：删除左侧Nodes顶部Select按钮与选择弹窗

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`feat/file-remove-nodes-select`
- Worktree：`d:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-remove-nodes-select`
- 归档文档：`docs/change/2026-03-06_win-file-remove-nodes-select.md`

## 可执行任务清单（Checklist）
- [x] REMOVE-NODES-SELECT-1：删除 Nodes 顶部 Select 按钮
- [x] REMOVE-NODES-SELECT-2：删除 browserNodePicker 相关状态、函数、弹窗
- [x] REMOVE-NODES-SELECT-3：回归验证、Code Review、归档

## 验证结果
- `GOWORK=off go test ./...`：通过。
- `frontend npm run build`：受项目环境缺失 `wailsjs` 生成文件影响失败（非本次改动引入）。

## 当前执行状态
- 已完成：REMOVE-NODES-SELECT-1 ~ REMOVE-NODES-SELECT-3
- 进行中：无
- 待完成：无

---
## [2026-03-06] Workflow归档 - MyFlowHub-Win Node Vars list 超时修复
- 归档文档：`docs/change/2026-03-06_win-varpool-nodevars-list-timeout.md`

# TODO - Win(VarPool)：Node Vars 查询超时修复（list 不应直连 target=owner）

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`fix/nodevars-list-timeout`
- Worktree：`d:\project\MyFlowHub3\worktrees\fix-win-nodevars-list-timeout`
- Base：`main`
- 背景：拓扑 `2 -> 1 -> 9 -> 11(metrics)` 中，节点 2 在 Devices 页点击 Node 11 的 `Vars` 时提示：
  - `Failed to load node variables.`
  - `varpool list: request timed out`

## 项目目标与当前状态

### 目标
1) Node Vars 查询不再超时：在上述拓扑下，Node 11 的变量名列表可在超时前返回（成功或空列表），不弹出 timeout 错误。
2) 行为对齐 `repo/MyFlowHub-Server/docs/3-varstore.md` 的规范：`list` 请求应发送到“当前连接的 Hub/目标节点”（默认 hubId），而不是直发到 owner 节点。

### 当前状态（事实，可审计）
- 前端 `listOwnerNames(ownerId)` 当前调用 `ListSimple(sourceID, targetID=ownerId, { owner: ownerId })`，会构造 `MajorCmd + TargetID=ownerId` 的帧。
- Server 侧 `varstore`（v0.1.1）在 `TargetID!=local` 时会按 header target 前置转发 cmd，导致 list 被转发到叶子节点 owner。
- `metrics node`（Node 11）是 leaf client，只会发送 `varstore set` 到 hub 发布指标，并不会处理 `list/get` 请求，因此不会回包，最终触发超时。

## 可执行任务清单（Checklist）

- [x] `NODEVARS-1`：修正 Node Vars list 的 target 选择（对齐规范）
  - 目标：`listOwnerNames(ownerId)` 改为 `targetID=state.defaultTargetId(hubId)`（直连“直接父/Hub”的缓存），payload 仍为 `{ owner }`。
    - 说明：避免复用可编辑的 `Target ID` 输入，防止用户误设为 owner 再次触发 leaf 超时；需要调试时仍可在 VarPool 页手动 list/get。
  - 涉及文件：
    - `frontend/src/stores/varpool.ts`
  - 验收条件：
    - Node Vars 对话框对任意 owner 执行 Load，不再出现 `request timed out`；
    - owner 无缓存/无变量时返回空列表（不报错）。
  - 测试点：
    - 复现拓扑下：Node 11 Vars 可正常加载（有值或空列表均可接受，关键是“不超时”）；
    - 将 VarPool 页 `Target ID` 改为其它节点（如 9/1），Node Vars 查询仍可用且不超时。
  - 回滚点：
    - revert 本任务提交。

- [x] `NODEVARS-2`：最小回归验证（构建/类型检查）
  - 目标：确保 TS 改动不破坏构建与运行。
  - 验收条件：
    - `cd frontend && npm run build` 通过（若项目已有更合适的 lint/test 命令，可替换）。
  - 回滚点：
    - revert 本任务提交。

- [x] `NODEVARS-3`：Code Review + 归档（强制）
  - 目标：完成审查闭环与变更归档。
  - 涉及文件：
    - `docs/change/2026-03-06_win-varpool-nodevars-list-timeout.md`
  - 验收条件：
    - Review 通过；
    - 归档文档包含：问题原因、方案选择、任务映射、验证方式与回滚方案。

## 依赖关系
- `NODEVARS-1 -> NODEVARS-2 -> NODEVARS-3`

## 风险与注意事项
- Node Vars 的 list 在规范语义上依赖祖先链缓存（`up_set`），若 hub 重启而 metrics node 未重新 publish，可能出现“空列表但不超时”；这属于数据刷新问题，不属于本次“超时”回归范围。

---
## [2026-03-06] Workflow归档 - MyFlowHub-Android Node Vars list 走Hub缓存
- 归档文档：`docs/change/2026-03-06_android-nodevars-list-hub-target.md`

# TODO - Android VarStore：Node Vars `list` 走 Hub 缓存

## Workflow 信息
- Repo：`MyFlowHub-Android`
- 分支：`fix/android-nodevars-list-hub-target`
- Worktree：`d:\project\MyFlowHub3\worktrees\fix-android-nodevars-list-hub-target`
- Base：`main`

## 项目目标与当前状态

### 目标
1) Android Node Vars 查询与规范一致：`list` 请求发送到“直接父/Hub”，payload 通过 `owner` 指定目标节点。
2) 避免 `target=owner` 触发 leaf 节点不回 `list_resp` 的超时。

### 当前状态（事实）
- 当前 `listOwnerNames(ownerId)` 调用：
  - `g.varStoreList(sourceId, targetId=ownerId, owner=ownerId)`。
- 该行为会在 `varstore v0.1.1` 的 header target 转发逻辑下把 `list` 下发到 owner 节点；若 owner 为 metrics leaf（不处理 list/get），会超时。

## 可执行任务清单（Checklist）

- [x] `ANDROID-NODEVARS-1`：修正 Node Vars list target
  - 目标：将 `listOwnerNames(ownerId)` 的 `target` 改为 `parseDefaultTargetId()`（Hub），保持 `owner=ownerId` 不变。
  - 涉及文件：
    - `app/src/main/java/com/myflowhub/android/ui/VarStoreScreen.kt`
  - 验收条件：
    - Node Vars 加载不再因直发 owner 导致 `request timed out`；
    - owner 无缓存/无变量时返回空列表，不抛异常。
  - 测试点：
    - 业务链路验证：Node Vars 查询 owner=metrics 节点时应返回成功或空列表；
    - 静态验证：Kotlin 编译通过。
  - 回滚点：
    - revert 本任务提交。

- [x] `ANDROID-NODEVARS-2`：最小回归验证
  - 目标：确保改动不破坏 Android 构建。
  - 验收条件：
    - 执行 `./gradlew :app:compileDebugKotlin`（Windows 使用 `gradlew.bat`）通过；若受环境限制失败，记录失败原因。
  - 回滚点：
    - revert 本任务提交。

- [x] `ANDROID-NODEVARS-3`：Code Review + 归档
  - 目标：完成审查闭环与归档。
  - 涉及文件：
    - `docs/change/2026-03-06_android-nodevars-list-hub-target.md`
  - 验收条件：
    - Review 结论完整（覆盖需求/架构/性能/可扩展性/稳定性/测试）；
    - 归档文档包含任务映射、验证结果和回滚方案。

## 依赖关系
- `ANDROID-NODEVARS-1 -> ANDROID-NODEVARS-2 -> ANDROID-NODEVARS-3`

## 风险与注意事项
- Node Vars 的查询结果依赖链路缓存；Hub 重启后若 owner 尚未重新 publish，可能出现“空列表但不超时”，属于缓存刷新窗口，不属于本次修复范围。

---
## [2026-03-06] Workflow归档 - MyFlowHub-Server VarStore 文档错误码勘误
- 归档文档：`docs/change/2026-03-06_server-varstore-doc-errorcode-align.md`

# TODO - Server 文档勘误：VarStore `not found` 错误码示例对齐

## Workflow 信息
- Repo：`MyFlowHub-Server`
- 分支：`chore/server-varstore-docs-errorcode`
- Worktree：`d:\project\MyFlowHub3\worktrees\chore-server-varstore-docs-errorcode`
- Base：`main`

## 项目目标与当前状态

### 目标
1) 将 VarStore 规范文档中的错误码示例与“错误码约定”对齐，避免客户端误读（`4` vs `404`）。
2) 不改变协议实现，仅修正文档描述。

### 当前状态（事实）
- 文档错误码约定定义：`4` = 未找到。
- 同一文档示例处出现：`{"code":404,"msg":"not found"}`。
- 当前实现侧在 `list/get` 等场景使用 `code=4`（非 `404`）。

## 可执行任务清单（Checklist）

- [x] `DOCERRATA-1`：修正规范文档错误码示例
  - 目标：将 `docs/3-varstore.md` 的 not found 示例码从 `404` 调整为 `4`。
  - 涉及文件：
    - `docs/3-varstore.md`
  - 验收条件：
    - 文档示例与错误码约定一致。
  - 回滚点：
    - revert 本任务提交。

- [x] `DOCERRATA-2`：文档一致性复查
  - 目标：确认同文档无其它 `404` 残留语义冲突。
  - 验收条件：
    - `docs/3-varstore.md` 内相关示例与约定一致。
  - 回滚点：
    - revert 本任务提交。

- [x] `DOCERRATA-3`：Code Review + 归档
  - 目标：完成审查闭环与变更归档。
  - 涉及文件：
    - `docs/change/2026-03-06_server-varstore-doc-errorcode-align.md`
  - 验收条件：
    - 审查结论完整；
    - 归档内容包含任务映射、影响评估、回滚方案。

## 依赖关系
- `DOCERRATA-1 -> DOCERRATA-2 -> DOCERRATA-3`

## 风险与注意事项
- 本 workflow 仅修文档，不改协议 wire 与实现行为；避免与功能修复混杂。

---
## [2026-03-06] Workflow归档 - MyFlowHub-Win Showcase 顶部按钮图标化与 Tooltip
- 归档文档：`docs/change/2026-03-06_win-showcase-topbar-icon-tooltip.md`

# Plan - MyFlowHub-Win：Showcase 顶部按钮图标化与 Tooltip

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`feat/showcase-toolbar-icons`
- Worktree：`d:\project\MyFlowHub3\worktrees\feat-showcase-toolbar-icons\MyFlowHub-Win`
- Base：`main`
- 当前阶段：`4 归档变更`（已完成，待用户确认是否结束 workflow）

## 项目目标与当前状态
- 目标：
  - 将 `Showcase` 页面顶部操作区按钮由文字按钮改为图标按钮；
  - 为每个按钮增加 tooltip（显示原按钮文案）。
- 当前状态：
  - 顶部操作区包含 6 个文字按钮：`Refresh Vars / New Screen / Rename Screen / Delete Screen / Add Event / Add Var`。

## 可执行任务清单（Checklist）

- [x] `SHC-TOPBAR-1` 图标化顶部按钮并保留原行为
  - 目标：
    - 将 6 个按钮切换为 `size="icon"` 图标按钮；
    - 保持点击事件、禁用状态与视觉分组行为不变。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
  - 验收条件：
    - 6 个按钮均为图标按钮；
    - 原有事件处理函数（如 `refreshVars` 等）不变且可触发。
  - 测试点：
    - 进入 `#/showcase` 页面，逐个点击按钮验证行为。
  - 回滚点：
    - 回滚 `Showcase.vue` 顶部按钮模板区块。

- [x] `SHC-TOPBAR-2` 增加 tooltip 与可访问性文本
  - 目标：
    - 每个图标按钮新增 tooltip，文案与原按钮文字一致；
    - 增加 `sr-only` 文本，保障可访问性。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
  - 验收条件：
    - 鼠标悬浮可显示对应文案；
    - 页面无可访问性文本丢失问题。
  - 测试点：
    - 手动悬浮验证：`Refresh Vars / New Screen / Rename Screen / Delete Screen / Add Event / Add Var`。
  - 回滚点：
    - 回滚按钮 `title` 与 `sr-only` 相关改动。

- [x] `SHC-TOPBAR-3` 回归验证与归档准备
  - 目标：
    - 完成最小回归验证；
    - 为阶段 3.3 与 4 产出输入。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
  - 验收条件：
    - 代码可通过前端构建（若环境允许）；
    - 无多余计划外改动。
  - 测试点：
    - `npm --prefix frontend run build`（若脚本存在）；
    - `git diff -- frontend/src/pages/Showcase.vue` 人工审查。
  - 执行结果：
    - `git diff` 人工审查完成；
    - `npm --prefix frontend run build` 失败：环境缺少 `vite` 可执行文件（依赖未安装）。
  - 回滚点：
    - 回滚本次分支全部修改。

## 依赖关系
- `SHC-TOPBAR-2` 依赖 `SHC-TOPBAR-1`。
- `SHC-TOPBAR-3` 依赖 `SHC-TOPBAR-1`、`SHC-TOPBAR-2`。

## 风险与注意事项
- 图标选择需避免语义混淆，尽量使用已有页面常用图标风格。
- 仅修改 `Showcase` 顶部操作区，不影响 widget 区域与其他页面。
- tooltip 采用原生 `title`，无需引入新组件，降低依赖和回归风险。


---
## [2026-03-06] Workflow归档 - MyFlowHub-Win Showcase 工具栏Tooltip组件化与变量弹窗优化
- 归档文档：`docs/change/2026-03-06_win-showcase-var-dialog-polish.md`

# Plan - MyFlowHub-Win：Showcase 工具栏 Tooltip 组件化与变量弹窗优化

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`feat/showcase-var-dialog-polish`
- Worktree：`d:\project\MyFlowHub3\worktrees\feat-showcase-var-dialog-polish\MyFlowHub-Win`
- Base：`main`
- 当前阶段：`4 归档变更`（已完成，待用户确认是否结束 workflow）

## 项目目标与当前状态
- 目标：
  1. 顶部按钮改为“组件 tooltip”，并在正下方显示；
  2. `Add Event/Add Var` 与 screen 级操作分组并增加分割线；
  3. 新增变量默认 target=hub nodeId，且不暴露 target 输入；
  4. `Throttle (ms)` 详细说明放到标题 tooltip；
  5. `On Value/Off Value` 仅在 `mode=switch` 显示，并通过分割线与通用设置区分。
- 当前状态：
  - 顶部按钮使用 `title` 原生 tooltip；
  - 变量弹窗中 target 可编辑；
  - Throttle 说明在常驻文本；
  - On/Off 常驻显示。

## 可执行任务清单（Checklist）

- [x] `SHC-UX-1` 新增 Tooltip 组件并应用到顶部按钮
  - 目标：
    - 引入可复用 Tooltip 组件，tooltip 固定在触发元素下方；
    - 顶部按钮从 `title` 切到组件 tooltip；
    - screen 操作组与 widget 新增组之间加分割线。
  - 涉及模块/文件：
    - `frontend/src/components/ui/tooltip/*`
    - `frontend/src/pages/Showcase.vue`
  - 验收条件：
    - 顶部 6 个按钮悬浮显示 tooltip（底部）；
    - 中间分割线可见；
    - 按钮点击行为与禁用态不回归。
  - 测试点：
    - 手工悬浮与点击 `Refresh/New/Rename/Delete/Add Event/Add Var`。
  - 回滚点：
    - 回滚 Tooltip 组件与按钮区改动。

- [x] `SHC-UX-2` 变量 target 固定为 hub 且隐藏输入
  - 目标：
    - 变量新增时 target 固定 `hubId`；
    - 变量表单不再展示 `Target ID` 输入。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
  - 验收条件：
    - 新增变量不需要用户输入 target；
    - hubId 不可用时阻断保存并提示；
    - 主题事件（topic_button）Target 输入保持可编辑。
  - 测试点：
    - 新增 var 成功；hubId 为空时保存报错。
  - 回滚点：
    - 回滚 `submitWidgetDialog` 目标ID处理与表单字段显示逻辑。

- [x] `SHC-UX-3` 模式专属设置分区优化
  - 目标：
    - `Throttle (ms)` 标题支持 tooltip 详细说明；
    - `On/Off` 仅 `switch` 模式显示，并增加分割线与标题。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
  - 验收条件：
    - Throttle 说明不再常驻文本显示；
    - 非 switch 模式不显示 On/Off；
    - switch 模式显示独立分区（含分割线）。
  - 测试点：
    - 切换 mode 验证字段动态显示。
  - 回滚点：
    - 回滚变量配置区模板改动。

- [x] `SHC-UX-4` 验证、Code Review、归档
  - 目标：
    - 完成差异审查、可执行构建验证（环境允许范围）；
    - 完成阶段 3.3 审查与阶段 4 归档文档。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
    - `frontend/src/components/ui/tooltip/*`
    - `docs/change/2026-03-06_win-showcase-var-dialog-polish.md`
  - 验收条件：
    - 变更与任务映射完整；
    - 归档文档包含验证结果与回滚方案。
  - 测试点：
    - `npm --prefix frontend run build`（若依赖齐全）。
  - 执行结果：
    - `npm --prefix frontend install`：通过；
    - `npm --prefix frontend run build`：失败，报错 `Could not resolve ../../wailsjs/go/auth/AuthService from src/pages/Home.vue`（项目现有环境/生成物问题，非本次变更引入）。
  - 回滚点：
    - 回滚本分支改动。

## 依赖关系
- `SHC-UX-2` 依赖 `SHC-UX-1`（tooltip 组件引入后再统一改模板更稳妥）。
- `SHC-UX-3` 可与 `SHC-UX-2` 并行修改同文件，但提交前统一联调。
- `SHC-UX-4` 依赖全部前置任务。

## 风险与注意事项
- Tooltip 触发器使用 `as-child` 时必须保证单一可交互根节点，避免事件丢失。
- 变量 target 隐藏后需保证编辑旧数据不被意外覆盖。


---
## [2026-03-06] Workflow归档 - MyFlowHub-Win 修复Showcase页面空白（Tooltip Provider注入）
- 归档文档：`docs/change/2026-03-06_win-showcase-page-blank-fix.md`

# Plan - MyFlowHub-Win：修复 Showcase 页面空白（Tooltip Provider 注入）

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`fix/showcase-page-blank`
- Worktree：`d:\project\MyFlowHub3\worktrees\fix-showcase-page-blank\MyFlowHub-Win`
- Base：`main`
- 当前阶段：`4 归档变更`（已完成，待用户确认是否结束 workflow）

## 项目目标与当前状态
- 目标：
  - 修复 `Showcase` 页面进入后主内容区空白的问题，恢复页面渲染与交互可用性。
- 当前状态（已定位）：
  - 新增 `Tooltip` 封装缺少 `TooltipProvider`；
  - `radix-vue` 在 `TooltipRoot` 初始化时强依赖 Provider 注入，缺失即抛错并中断页面渲染。

## 可执行任务清单（Checklist）

- [x] `SHC-BLANK-1` 修复 Tooltip 注入链路
  - 目标：在 Tooltip 组件内补齐 `TooltipProvider`，保证 `TooltipRoot` 上下文完整。
  - 涉及模块/文件：
    - `frontend/src/components/ui/tooltip/Tooltip.vue`
  - 验收条件：
    - Tooltip 使用点不改即可正常渲染；
    - 不再出现 Provider 注入缺失导致的渲染中断。
  - 测试点：
    - 静态审查：`Tooltip.vue` 引入并包裹 `TooltipProvider`。
  - 回滚点：
    - 回滚该文件改动。

- [x] `SHC-BLANK-2` 最小回归验证
  - 目标：确认修复未引入额外编译错误。
  - 验收条件：
    - 执行前端构建命令并记录结果；
    - 若受项目环境影响失败，明确失败与本次改动的关系。
  - 测试点：
    - `npm --prefix frontend run build`
  - 执行结果：
    - 构建命令执行到 Vite 打包阶段；
    - 失败于项目现有 `wailsjs` 生成物缺失（`Home.vue` 引用 `../../wailsjs/go/session/SessionService` 无法解析），非本次 tooltip 修复引入。
  - 回滚点：
    - 回滚本次修复提交。

- [x] `SHC-BLANK-3` Code Review 与归档
  - 目标：完成 3.3 审查与 4 阶段归档文档。
  - 涉及模块/文件：
    - `docs/change/2026-03-06_win-showcase-page-blank-fix.md`
  - 验收条件：
    - 审查结论覆盖需求/架构/性能/可读性/稳定性/测试。
  - 回滚点：
    - 回滚本 workflow 改动。

## 依赖关系
- `SHC-BLANK-2` 依赖 `SHC-BLANK-1`。
- `SHC-BLANK-3` 依赖 `SHC-BLANK-1`、`SHC-BLANK-2`。

## 风险与注意事项
- TooltipProvider 若放置层级错误（不包裹 TooltipRoot）仍会报注入错误。
- 本修复不触碰 Showcase 业务数据逻辑，避免引入功能性副作用。

---
## [2026-03-06] Workflow归档 - MyFlowHub-Win Showcase Throttle 文案调整
- 归档文档：docs/change/2026-03-06_win-showcase-throttle-label.md

# Plan - MyFlowHub-Win：Showcase Throttle 文案调整

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`fix/showcase-throttle-label`
- Worktree：`d:\project\MyFlowHub3\worktrees\fix-showcase-throttle-label\MyFlowHub-Win`
- Base：`main`
- 当前阶段：`4 归档变更`（已完成，待用户确认是否结束 workflow）

## 项目目标与当前状态
- 目标：
  - 将变量配置标题 `Throttle (ms)` 调整为 `Throttle`；
  - 在 tooltip 内明确单位为毫秒（ms）。
- 当前状态：
  - 界面标签显示 `Throttle (ms)`；
  - tooltip 只说明 0 的语义，未显式说明单位。

## 可执行任务清单（Checklist）

- [x] `SHC-THROTTLE-1` 调整 Throttle 标签与 tooltip
  - 目标：
    - 标题由 `Throttle (ms)` 改为 `Throttle`；
    - tooltip 文案新增“单位毫秒（ms）”说明。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
  - 验收条件：
    - 界面显示 `Throttle`；
    - 悬浮 tooltip 包含单位说明。
  - 测试点：
    - 打开 Add/Edit Var 弹窗，检查 Throttle 标签与 tooltip。
  - 回滚点：
    - 回滚该文件相关文案修改。

- [x] `SHC-THROTTLE-2` 一致性与回归验证
  - 目标：
    - 同步校验错误文案字段名一致；
    - 执行最小构建验证并记录结果。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
  - 验收条件：
    - 校验字段名与 UI 标签一致；
    - 构建命令执行并有结果记录。
  - 测试点：
    - `npm --prefix frontend run build`。
  - 执行结果：
    - `npm --prefix frontend install`：通过；
    - `npm --prefix frontend run build`：失败，报错 `Could not resolve ../../wailsjs/go/session/SessionService from src/pages/Home.vue`（项目现有环境/生成物问题，非本次文案改动引入）。
  - 回滚点：
    - 回滚本次提交。

- [x] `SHC-THROTTLE-3` Code Review 与归档
  - 目标：
    - 完成 3.3 审查输出；
    - 产出 docs/change 归档文档。
  - 涉及模块/文件：
    - `docs/change/2026-03-06_win-showcase-throttle-label.md`
  - 验收条件：
    - 审查结论完整；
    - 归档文档包含任务映射、验证结果、回滚方案。

## 依赖关系
- `SHC-THROTTLE-2` 依赖 `SHC-THROTTLE-1`。
- `SHC-THROTTLE-3` 依赖 `SHC-THROTTLE-1`、`SHC-THROTTLE-2`。

## 风险与注意事项
- 本次仅文案调整，不应改变节流逻辑与默认值。

---

# Archive - 2026-03-06 VarStore 逐跳回程对齐（较大改动）

## 关联提交（已合并至各仓 `main`）
- `MyFlowHub-SubProto`：`e4cbd20`（`refactor/varstore-hop-align`）
- `MyFlowHub-SDK`：`77dbd7e`（`refactor/varstore-hop-align`）
- `MyFlowHub-Win`：`d37251b`（`refactor/varstore-hop-align`）
- `MyFlowHub-Server`：`3771701`（`chore/varstore-hop-align-docs`）

## 全局归档（docs/change）
- `docs/change/2026-03-06_varstore-hop-align-subproto.md`
- `docs/change/2026-03-06_varstore-cmd-await-sdk.md`
- `docs/change/2026-03-06_varstore-hop-align-win.md`
- `docs/change/2026-03-06_varstore-hop-align-server.md`

## 迁移计划（来自各仓 worktree 的 `plan.md`）

# Plan - MyFlowHub-SubProto：VarStore 逐跳可见与并发匹配对齐

## Workflow 信息
- Repo：`MyFlowHub-SubProto`
- 分支：`refactor/varstore-hop-align`
- Worktree：`d:\project\MyFlowHub3\worktrees\varstore-hop-align\subproto`
- Base：`main`
- 关联仓库：`MyFlowHub-SDK`、`MyFlowHub-Win`、`MyFlowHub-Server`

## 项目目标与当前状态
- 目标：按已确认规范完成 VarStore 行为对齐，覆盖逐跳回程可见、并发写一一对应、notify 下行“转发+本地处理”、SourceID 端到端保留。
- 当前状态：已完成阶段 1/2 文档（`varstore_requirements.md`、`varstore_architecture.md`）；本仓已完成 VS-1~VS-6 核心改造与关键单测（VS-7 部分），待联动 SDK/Win 与归档。

## 依赖关系
- 先做本仓核心改造（本计划 VS-1~VS-7），再联动 SDK/Win await 兼容验证。
- Server 文档同步可并行，但最终验收需以本仓行为为准。

## 风险与注意事项
- `MajorCmd` 回程会提高 handler 处理频次，需谨慎处理 pending 生命周期，避免泄漏。
- 并发写入映射表若缺少超时/异常清理，容易造成状态堆积。
- 不得破坏 wire（action/data schema/SubProto 不变）。

## 可执行任务清单（Checklist）

### VS-1 统一响应逐跳可见（`MajorCmd`）
- 目标：`*_resp` 与 `assist_*_resp` 改为逐跳可见，不再依赖 `MajorOKResp` 快速转发。
- 涉及模块/文件：`varstore/varstore.go`
- 验收条件：多 hop 场景中间节点能进入 `handle*Resp`；pending 可逐跳出队。
- 测试点：构造两跳链路请求，验证中间节点收到并处理 resp。
- 回滚点：恢复 `buildRespHeader` 的旧 major 策略。

### VS-2 SourceID 端到端保留
- 目标：`assist_* / up_* / notify_* / *_resp` 转发不改写原始 actor SourceID。
- 涉及模块/文件：`varstore/varstore.go`（`forward`/resp 构建相关）
- 验收条件：跨 hop 抓包或日志可见 SourceID 恒等于原请求发起方。
- 测试点：转发链路断言 SourceID 不变。
- 回滚点：恢复旧 forward 源 ID 赋值逻辑。

### VS-3 写操作并发一一对应（策略 C）
- 目标：`set/revoke` 使用“上行 msg_id + 映射表”，回程按请求粒度匹配并恢复下游 msg_id。
- 涉及模块/文件：`varstore/types.go`、`varstore/varstore.go`
- 验收条件：同 `(owner,name)` 并发两次写入时，不错配、不吞响应。
- 测试点：并发 set/revoke 的映射命中、超时清理、断连清理。
- 回滚点：恢复旧 pendingKey 方案（仅 owner/name/kind）。

### VS-4 查询去重扇出与沿途缓存
- 目标：`get/list/subscribe` 保持去重上送，一次回程可扇出全部等待者；成功响应按规则沿途更新缓存。
- 涉及模块/文件：`varstore/varstore.go`
- 验收条件：同 key 并发查询仅上送一次 assist，所有等待者都收到正确响应。
- 测试点：去重命中、扇出数量、缓存写入内容完整性。
- 回滚点：恢复旧 pending 处理逻辑。

### VS-5 notify 下行“转发 + 本地处理”
- 目标：`notify_set/notify_revoke` 在 `TargetID!=local` 场景仍执行本地缓存/订阅推送，再继续转发。
- 涉及模块/文件：`varstore/varstore.go`
- 验收条件：中间节点收到 notify 时，本地缓存变更与下游推送均生效。
- 测试点：notify 多 hop 链路中间节点状态变化。
- 回滚点：恢复旧“转发即 return”路径。

### VS-6 规则收敛：`set_resp/list/subscriber/private`
- 目标：落实已确认规则：
  - `set_resp code=1` 必带 `value`，失败不更新缓存；
  - owner 自查 list 包含 private，空集合 `code=1 names=[]`；
  - `set.value` 禁止空串/纯空白；
  - `subscriber` 只允许 0 或等于 SourceID；
  - private 允许权限例外。
- 涉及模块/文件：`varstore/varstore.go`
- 验收条件：规则对应分支全部可触发且返回码正确。
- 测试点：每条规则至少 1 个正例 + 1 个反例。
- 回滚点：按规则维度逐项回退。

### VS-7 测试与归档
- 目标：补齐关键单测并形成本仓变更归档。
- 涉及模块/文件：`varstore/*_test.go`、`docs/change/2026-03-06_varstore-hop-align-subproto.md`
- 验收条件：`go test ./... -count=1 -p 1` 通过；归档文档完整映射 VS-1~VS-6。
- 测试点：优先跑 `varstore` module，再跑仓库全量。
- 回滚点：测试与文档改动可独立回退，不影响功能提交。

# Plan - MyFlowHub-SDK：await 兼容 VarStore `MajorCmd` 响应

## Workflow 信息
- Repo：`MyFlowHub-SDK`
- 分支：`refactor/varstore-hop-align`
- Worktree：`d:\project\MyFlowHub3\worktrees\varstore-hop-align\sdk`
- Base：`main`
- 关联仓库：`MyFlowHub-SubProto`、`MyFlowHub-Win`、`MyFlowHub-Server`

## 项目目标与当前状态
- 目标：在不破坏现有 await 行为的前提下，支持 VarStore 响应走 `MajorCmd`。
- 当前状态：已完成 SDK-1/SDK-2：await 对 VarStore 子协议新增 `MajorCmd` 白名单兼容路径，并补齐单测验证；`go test ./...` 通过。

## 依赖关系
- 依赖 SubProto 新行为定义（resp 逐跳可见）。
- Win 侧服务调用依赖 SDK await 行为，需联调验证。

## 风险与注意事项
- 不能全局放宽到所有 `MajorCmd`，避免误匹配普通命令帧。
- 匹配条件需保留 `subproto + action + msg_id/trace_id` 的约束。

## 可执行任务清单（Checklist）

### SDK-1 await 匹配策略扩展
- 目标：为 VarStore 响应引入 `MajorCmd` 兼容路径（白名单 action 或按 await 选项控制）。
- 涉及模块/文件：`await/client.go`（及相关 matcher/option 文件）
- 验收条件：VarStore `*_resp` 为 `MajorCmd` 时可正常命中 await。
- 测试点：VarStore 响应 `MajorCmd` 命中；非目标 Cmd 帧不误命中。
- 回滚点：恢复旧 major 匹配策略。

### SDK-2 回归兼容与测试
- 目标：保证现有 `MajorOKResp/MajorErrResp` 流程不回归。
- 涉及模块/文件：`await/*_test.go`
- 验收条件：旧用例继续通过；新增 VarStore Cmd 响应用例通过。
- 测试点：超时、取消、多并发 await 匹配。
- 回滚点：回退新增 matcher 分支。

### SDK-3 归档变更
- 目标：沉淀本仓改动、风险与回滚策略。
- 涉及模块/文件：`docs/change/2026-03-06_varstore-cmd-await-sdk.md`
- 验收条件：文档映射 SDK-1~SDK-2，给出联调命令。
- 测试点：文档中的命令可复现验证。
- 回滚点：文档回退不影响功能代码。

# Plan - MyFlowHub-Win：VarPool 联动适配 VarStore 回应语义

## Workflow 信息
- Repo：`MyFlowHub-Win`
- 分支：`refactor/varstore-hop-align`
- Worktree：`d:\project\MyFlowHub3\worktrees\varstore-hop-align\win`
- Base：`main`
- 关联仓库：`MyFlowHub-SubProto`、`MyFlowHub-SDK`、`MyFlowHub-Server`

## 项目目标与当前状态
- 目标：确保 Win 侧 VarPool 在 VarStore 新回程策略（逐跳 Cmd 响应）下行为一致、UI 不回归。
- 当前状态：SDK await 已完成 VarStore `MajorCmd` 响应兼容；Win 侧待在升级 SDK/SubProto 版本后做冒烟验证，并按需做最小适配（WIN-1/WIN-2）。

## 依赖关系
- 依赖 SDK await 对 VarStore `MajorCmd` 响应的兼容。
- 依赖 SubProto 新规则（`set_resp` value、list 空集合语义、subscriber 规则）。

## 风险与注意事项
- 避免 UI 侧重复处理 owner 通知与请求响应导致状态闪动。
- 避免对无关子协议 await 行为产生副作用。

## 可执行任务清单（Checklist）

### WIN-1 VarPool 服务层兼容检查与最小改造
- 目标：确认并修复 VarPool 服务对响应 major/action 的假设，保证请求-响应闭环稳定。
- 涉及模块/文件：`internal/services/varpool/service.go`、必要的 await 调用封装文件。
- 验收条件：get/set/list/revoke/subscribe 在新回程语义下可正常返回。
- 测试点：VarPool 服务层单测或最小集成测试。
- 回滚点：恢复服务层匹配策略改动。

### WIN-2 前端状态语义校准
- 目标：前端 store/UI 与新规则一致：list 空集合成功展示、set 失败不误刷新缓存、notify 不重复提示。
- 涉及模块/文件：`frontend/src/stores/varpool.ts`、必要页面文件。
- 验收条件：UI 行为与服务返回码一致，不出现空列表误报错误。
- 测试点：手动冒烟 + （如存在）前端测试用例。
- 回滚点：回退 store 侧处理分支。

### WIN-3 归档变更
- 目标：沉淀 Win 联动改造内容与验证结果。
- 涉及模块/文件：`docs/change/2026-03-06_varstore-hop-align-win.md`
- 验收条件：文档覆盖 WIN-1~WIN-2，含风险与回滚步骤。
- 测试点：文档命令/操作可复现。
- 回滚点：文档改动可独立回退。

# Plan - MyFlowHub-Server：VarStore 规范文档与依赖联动

## Workflow 信息
- Repo：`MyFlowHub-Server`
- 分支：`chore/varstore-hop-align-docs`
- Worktree：`d:\project\MyFlowHub3\worktrees\varstore-hop-align\server`
- Base：`main`
- 关联仓库：`MyFlowHub-SubProto`、`MyFlowHub-SDK`、`MyFlowHub-Win`

## 项目目标与当前状态
- 目标：把 `docs/3-varstore.md` 与已确认语义对齐，并准备 Server 侧对新 VarStore 版本的联动说明。
- 当前状态：已完成 SRV-1：`docs/3-varstore.md` 已按最终决策更新；SRV-2（依赖版本联动）待 SubProto 发布可解析版本后再评估执行。

## 依赖关系
- 文档语义依赖已确认的阶段 1/2 决策。
- 如需要更新 `go.mod` 的 VarStore 版本，依赖 SubProto 发布可解析版本。

## 风险与注意事项
- 文档必须明确 `MajorCmd` 与 `TargetID=0` 的关系，避免误导实现。
- 若版本未发布，避免提交不可解析的依赖版本号。

## 可执行任务清单（Checklist）

### SRV-1 更新 VarStore 规范文档
- 目标：按确认结论修订 `docs/3-varstore.md`：
  - `*_resp/assist_*_resp` 逐跳可见；
  - requester/owner 回程语义；
  - notify 下行“转发+本地处理”；
  - SourceID 端到端保留；
  - set_resp value、list 空集合、set.value、private 例外、subscriber 规则。
- 涉及模块/文件：`docs/3-varstore.md`
- 验收条件：与 `varstore_requirements.md`/`varstore_architecture.md` 无冲突。
- 测试点：人工审阅 + 与实现交叉核对。
- 回滚点：回退文档提交。

### SRV-2 评估并执行依赖联动（条件任务）
- 目标：在 SubProto 发布新版本后，评估是否升级 Server 的 `myflowhub-subproto/varstore` 版本。
- 涉及模块/文件：`go.mod`、`go.sum`（如执行）
- 验收条件：仅在版本可解析时提交依赖变更。
- 测试点：`go list -m` + `go test ./... -count=1 -p 1`。
- 回滚点：回退依赖版本提交。

### SRV-3 归档变更
- 目标：记录文档与依赖联动（若有）的最终结果。
- 涉及模块/文件：`docs/change/2026-03-06_varstore-hop-align-server.md`
- 验收条件：文档映射 SRV-1~SRV-2，说明未执行条件任务的原因（如适用）。
- 测试点：归档内容可供他人复核。
- 回滚点：文档可独立回退。

---
## [2026-03-09] Workflow归档 - MyFlowHub-Win Showcase 变量快捷选择（订阅/Mine）
- 归档文档：`docs/change/2026-03-09_win-showcase-var-quickpick.md`

# Plan - MyFlowHub-Win：Showcase 变量快捷选择（订阅 / Mine）

## Workflow 信息
- 仓库：`MyFlowHub-Win`
- 分支：`feat/showcase-var-quickpick`
- Worktree：`d:\project\MyFlowHub3\worktrees\feat-showcase-var-quickpick\MyFlowHub-Win`
- Base：`main`
- 当前阶段：`4 归档变更`（已完成，待用户确认是否结束 workflow）

## 项目目标与当前状态
- 目标：
  - 在变量弹窗 `Variable Name` 右侧增加选择按钮；
  - 点击后弹出变量候选，来源为“当前订阅 + mine”；
  - 选中后自动回填 `Owner NodeID` 与 `Variable Name`。
- 当前状态：
  - 变量名仅可手输，无快捷选择入口。

## 可执行任务清单（Checklist）

- [x] `SHC-QUICKPICK-1` 增加变量快捷选择弹窗与按钮
  - 目标：
    - 在 `Variable Name` 右侧增加选择按钮；
    - 新增弹窗展示候选变量并支持点击回填。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
  - 验收条件：
    - 按钮可见可点；
    - 弹窗可打开/关闭；
    - 点击候选后回填成功。
  - 测试点：
    - 打开 Add/Edit Var，选择候选变量并观察字段回填。
  - 回滚点：
    - 回滚 `Showcase.vue` 相关 UI 代码。

- [x] `SHC-QUICKPICK-2` 复用 VarPool 数据并分组展示
  - 目标：
    - 候选数据来自 `VarPool` 当前缓存；
    - 以 `Subscribed` 与 `Mine` 分组展示，避免口径不一致。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
  - 验收条件：
    - 仅展示有 owner 的有效变量；
    - 订阅与 mine 分组正确。
  - 测试点：
    - 有订阅/有mine/无数据三种场景显示正常。
  - 回滚点：
    - 回滚数据计算逻辑。

- [x] `SHC-QUICKPICK-3` 回归验证、Code Review、归档
  - 目标：
    - 完成最小构建验证并记录结果；
    - 完成阶段 3.3 与阶段 4 文档归档。
  - 涉及模块/文件：
    - `frontend/src/pages/Showcase.vue`
    - `docs/change/2026-03-09_win-showcase-var-quickpick.md`
  - 验收条件：
    - Review 结论完整；
    - 归档文档包含任务映射与回滚方案。
  - 测试点：
    - `npm --prefix frontend run build`（环境允许）。
  - 执行结果：
    - `npm --prefix frontend install`：通过；
    - `npm --prefix frontend run build`：失败，报错 `Could not resolve ../../wailsjs/go/session/SessionService from src/pages/Home.vue`（项目现有环境/生成物问题，非本次改动引入）。
  - 回滚点：
    - 回滚本 workflow 改动。

## 依赖关系
- `SHC-QUICKPICK-2` 依赖 `SHC-QUICKPICK-1`。
- `SHC-QUICKPICK-3` 依赖前置任务完成。

## 风险与注意事项
- 候选来自缓存数据，若缓存为空需提供明确空状态提示。
- 该功能应仅影响 Var Widget 表单，不影响 Topic Widget。

## 2026-03-12 - 重大变更：Pipe 抽象与多承载扩展点（为 RFCOMM 铺路）
- [x] 目标：管理器不再直接持有底层连接（TCP 等），统一持有字节流 Pipe；按 Header 路由，payload 仅按需解析。
- [x] 变更概览（Breaking Change）：
  - Core：`core.IConnection` 移除 `RawConn()`，新增 `Pipe()`；Reader/SendDispatcher/ParentDialer 解耦 TCP；新增 `MultiListener`；ConnMgr 直连 nodeID 冲突仅保留一条。
  - Server：多 listener 装配与开关（重启生效）；新增 `ParentEndpoint`（当前仅支持 `tcp://`）；RFCOMM 仅预留配置位。
  - SubProto：仅适配测试 stub/mock，不改子协议语义。
- [x] 归档：
  - `docs/change/2026-03-12_transport-pipe-core.md`
  - `docs/change/2026-03-12_transport-pipe-server.md`
  - `docs/change/2026-03-12_transport-pipe-subproto.md`
  - 计划归档：`docs/plan_archive/plan_archive_2026-03-12_transport-pipe-*.md`

---
## [2026-03-12] Workflow归档 - Server/SubProto 升级 Core v0.3.0（修复 GOWORK=off）
- 归档文档：
  - `docs/change/2026-03-12_bump-core-v0.3.0-server.md`
  - `docs/change/2026-03-12_bump-core-v0.3.0-subproto.md`
- 计划归档：
  - `docs/plan_archive/plan_archive_2026-03-12_bump-core-v0.3.0-server-prev.md`
  - `docs/plan_archive/plan_archive_2026-03-12_bump-core-v0.3.0-subproto-prev.md`
- 结论：两仓 `GOWORK=off go test ./...` 均通过；已合并并 push 到各自 `main`。

# Plan - Server：升级依赖到 Core v0.3.0（对齐 Pipe 抽象重大变更）

## Workflow 信息
- Repo：`MyFlowHub-Server`
- 分支：`chore/bump-core-v0.3.0`
- Worktree：`d:\project\MyFlowHub3\worktrees\chore-bump-core-v0.3.0\MyFlowHub-Server`
- Base：`main`
- 关联仓库：
  - `MyFlowHub-Core`：已发布 `v0.3.0`（重大变更：`IConnection.RawConn()` → `IConnection.Pipe()`，并新增 `listener/multi_listener`）

## 背景 / 问题陈述（事实，可审计）
- Server `main` 已合入 Pipe 抽象相关改动，并开始依赖 Core 新增包：`github.com/yttydcs/myflowhub-core/listener/multi_listener`。
- 但 `go.mod` 仍依赖 `github.com/yttydcs/myflowhub-core v0.2.1`，导致在 `GOWORK=off`（CI/用户默认）下无法编译与运行测试。

## 目标
1) 将 `github.com/yttydcs/myflowhub-core` 依赖升级到 `v0.3.0`。
2) 执行 `go mod tidy` 并确保 `GOWORK=off go test ./...` 通过。

## 非目标
- 不改任何业务逻辑/协议语义（仅做依赖升级与必要的 go.mod/go.sum 更新）。
- 不发布新 tag（如需发布由后续 workflow 决策）。

## 验收标准
- `cd d:\project\MyFlowHub3\worktrees\chore-bump-core-v0.3.0\MyFlowHub-Server`
  - `GOWORK=off go test ./... -count=1 -p 1` 通过。
- 合并到 `main` 并 push。

## 3.1) 计划拆分（Checklist）

### SRVDEP0 - 归档旧 plan（已执行）
- 已执行：`git mv plan.md docs/plan_archive/plan_archive_2026-03-12_bump-core-v0.3.0-server-prev.md`

### SRVDEP1 - 升级 Core 依赖到 v0.3.0
- 目标：`go.mod` 中 `github.com/yttydcs/myflowhub-core` 从 `v0.2.1` 升级到 `v0.3.0`。
- 涉及文件：`go.mod`、`go.sum`
- 验收条件：`GOWORK=off go test ./...` 编译通过。
- 回滚点：revert 本任务提交。

### SRVDEP2 - 回归测试（GOWORK=off）
- 目标：确保 CI/用户默认模式可运行。
- 测试点：
  - `GOWORK=off go test ./... -count=1 -p 1`

### SRVDEP3 - Code Review + 归档变更
- 输出：`docs/change/2026-03-12_bump-core-v0.3.0-server.md`

### SRVDEP4 - 合并 / push（需 workflow 结束后执行）
- 在 `repo/MyFlowHub-Server` 合并到 `main` 并 push。

# Plan - SubProto：升级依赖到 Core v0.3.0（对齐 Pipe 抽象重大变更）

## Workflow 信息
- Repo：`MyFlowHub-SubProto`
- 分支：`chore/bump-core-v0.3.0`
- Worktree：`d:\project\MyFlowHub3\worktrees\chore-bump-core-v0.3.0\MyFlowHub-SubProto`
- Base：`main`
- 关联仓库：
  - `MyFlowHub-Core`：已发布 `v0.3.0`（重大变更：`IConnection.RawConn()` → `IConnection.Pipe()`）

## 背景 / 问题陈述（事实，可审计）
- 本仓 `main` 的部分单测已适配 `core.IConnection.Pipe()`（不再实现 `RawConn()`），但各 module `go.mod` 仍固定依赖 `myflowhub-core v0.2.1`。
- 在 `GOWORK=off`（CI/用户默认）下运行 `go test` 会拉取 `v0.2.1`，从而导致测试编译失败。

## 目标
1) 将受影响的子模块（至少 `auth` / `management` / `varstore`）的 `myflowhub-core` 升级到 `v0.3.0`。
2) 执行 `go mod tidy` 并确保 `GOWORK=off` 下单测通过。

## 非目标
- 不改子协议 wire/语义/路由规则（仅做依赖升级与 go.mod/go.sum 更新）。
- 不强制发布新 tag（如需发布由后续 workflow 决策）。

## 验收标准
- `GOWORK=off` 下至少通过：
  - `cd auth; go test ./... -count=1 -p 1`
  - `cd management; go test ./... -count=1 -p 1`
  - `cd varstore; go test ./... -count=1 -p 1`
- 合并到 `main` 并 push。

## 3.1) 计划拆分（Checklist）

### SUBDEP0 - 归档旧 plan（已执行）
- 已执行：`git mv plan.md docs/plan_archive/plan_archive_2026-03-12_bump-core-v0.3.0-subproto-prev.md`

### SUBDEP1 - 升级依赖（auth/management/varstore）
- 目标：各模块 `go.mod` 中 `github.com/yttydcs/myflowhub-core` 从 `v0.2.1` 升级到 `v0.3.0`，并 tidy。
- 涉及文件：
  - `auth/go.mod`、`auth/go.sum`
  - `management/go.mod`、`management/go.sum`
  - `varstore/go.mod`、`varstore/go.sum`
- 回滚点：revert 本任务提交。

### SUBDEP2 - 回归测试（GOWORK=off）
- 测试点：
  - `cd auth; GOWORK=off go test ./... -count=1 -p 1`
  - `cd management; GOWORK=off go test ./... -count=1 -p 1`
  - `cd varstore; GOWORK=off go test ./... -count=1 -p 1`

### SUBDEP3 - Code Review + 归档变更
- 输出：`docs/change/2026-03-12_bump-core-v0.3.0-subproto.md`

### SUBDEP4 - 合并 / push（需 workflow 结束后执行）
- 在 `repo/MyFlowHub-SubProto` 合并到 `main` 并 push。

---
## [2026-03-12] Workflow归档 - SubProto 补齐模块与 Android hubmobile 升级 Core v0.3.0
- 归档文档：
  - `docs/change/2026-03-12_bump-core-v0.3.0-subproto-more.md`
  - `docs/change/2026-03-12_bump-core-v0.3.0-android-hubmobile.md`
- 计划归档：
  - `docs/plan_archive/plan_archive_2026-03-12_bump-core-v0.3.0-subproto-more-prev.md`
  - `docs/plan_archive/plan_archive_2026-03-12_bump-core-v0.3.0-android-hubmobile-prev.md`
- 结论：两仓 `GOWORK=off go test` 均通过；已合并并 push 到各自 `main`。

# Plan - SubProto：补齐剩余模块对 Core v0.3.0 的依赖（对齐 Pipe 抽象重大变更）

## Workflow 信息
- Repo：`MyFlowHub-SubProto`
- 分支：`chore/bump-core-v0.3.0-subproto-more`
- Worktree：`d:\project\MyFlowHub3\worktrees\chore-bump-core-v0.3.0-subproto-more\MyFlowHub-SubProto`
- Base：`main`
- 关联仓库：
  - `MyFlowHub-Core`：已发布 `v0.3.0`（重大变更：`IConnection.RawConn()` → `IConnection.Pipe()`）

## 背景 / 问题陈述（事实，可审计）
- 本仓已开始在部分模块（如 `auth` / `management` / `varstore`）依赖 Core `v0.3.0`。
- 但仍有模块的 `go.mod` 固定 `myflowhub-core v0.2.1`（`exec` / `forward` / `file` / `flow` / `topicbus`）。
- 在 `GOWORK=off`（CI/用户默认）下，这些模块会拉取旧 Core 版本，带来：
  - 编译口径不一致（同一仓库内不同模块锁不同 Core 版本）；
  - 当代码/测试已按 Pipe 抽象演进时，可能出现 `go test` 编译失败。

## 目标
1) 将剩余模块的 `myflowhub-core` 依赖升级到 `v0.3.0`：
   - `exec` / `forward` / `file` / `flow` / `topicbus`
2) 执行 `go mod tidy`，并确保 `GOWORK=off` 下相关模块 `go test` 通过。

## 非目标
- 不改子协议 wire/语义/路由规则（仅为对齐 Core 版本做必要编译适配）。
- 不发布新 tag（如需发布由后续 workflow 决策）。

## 验收标准
- `GOWORK=off` 下通过：
  - `cd exec; go test ./... -count=1 -p 1`
  - `cd forward; go test ./... -count=1 -p 1`
  - `cd file; go test ./... -count=1 -p 1`
  - `cd flow; go test ./... -count=1 -p 1`
  - `cd topicbus; go test ./... -count=1 -p 1`
- 仓库内不再存在 `myflowhub-core v0.2.1` 的 `go.mod` 引用（用 `rg` 可验证）。
- 合并到 `main` 并 push。

## 3.1) 计划拆分（Checklist）

### SUBMORE0 - 归档旧 plan（已执行）
- 已执行：`git mv plan.md docs/plan_archive/plan_archive_2026-03-12_bump-core-v0.3.0-subproto-more-prev.md`

### SUBMORE1 - 升级依赖到 Core v0.3.0（exec/forward/file/flow/topicbus）
- 目标：各模块 `go.mod` 中 `github.com/yttydcs/myflowhub-core` 从 `v0.2.1` 升级到 `v0.3.0`，并 tidy。
- 说明：若升级后出现编译失败（例如接口从 `RawConn()` 迁移到 `Pipe()` 的实现差异），允许在不改变子协议语义的前提下做最小必要适配。
- 涉及文件：
  - `exec/go.mod`、`exec/go.sum`
  - `forward/go.mod`、`forward/go.sum`
  - `file/go.mod`、`file/go.sum`
  - `flow/go.mod`、`flow/go.sum`
  - `topicbus/go.mod`、`topicbus/go.sum`
- 验收条件：`GOWORK=off go test` 至少编译通过（见 SUBMORE2）。
- 回滚点：revert 本任务提交。

### SUBMORE2 - 回归测试（GOWORK=off）
- 测试点：
  - `cd exec; GOWORK=off go test ./... -count=1 -p 1`
  - `cd forward; GOWORK=off go test ./... -count=1 -p 1`
  - `cd file; GOWORK=off go test ./... -count=1 -p 1`
  - `cd flow; GOWORK=off go test ./... -count=1 -p 1`
  - `cd topicbus; GOWORK=off go test ./... -count=1 -p 1`

### SUBMORE3 - Code Review（强制）
- 逐项审查：需求覆盖/架构/性能/可读性/扩展性/稳定性与安全/测试覆盖。

### SUBMORE4 - 归档变更（强制）
- 输出：`docs/change/2026-03-12_bump-core-v0.3.0-subproto-more.md`

### SUBMORE5 - 合并 / push（需 workflow 结束后执行）
- 在 `repo/MyFlowHub-SubProto` 合并到 `main` 并 push。

# Plan - Android：hubmobile 升级依赖到 Core v0.3.0（对齐 Pipe 抽象重大变更）

## Workflow 信息
- Repo：`MyFlowHub-Android`
- 分支：`chore/bump-core-v0.3.0-android-hubmobile`
- Worktree：`d:\project\MyFlowHub3\worktrees\chore-bump-core-v0.3.0-android-hubmobile\MyFlowHub-Android`
- Base：`main`
- 关联仓库：
  - `MyFlowHub-Core`：已发布 `v0.3.0`（重大变更：`IConnection.RawConn()` → `IConnection.Pipe()`）

## 背景 / 问题陈述（事实，可审计）
- Android 仓库内仅 `hubmobile` 模块依赖 `myflowhub-core`，当前仍固定 `v0.2.1`。
- Core 已发布 `v0.3.0`（Pipe 抽象重大变更），且其它仓库已逐步完成依赖升级。
- 若 `hubmobile` 长期锁定旧 Core，将导致跨仓协作时版本不一致，并在 `GOWORK=off` 下潜在出现编译口径差异。

## 目标
1) 将 `hubmobile/go.mod` 中 `github.com/yttydcs/myflowhub-core` 升级到 `v0.3.0`。
2) 执行 `go mod tidy` 并确保 `GOWORK=off go test ./...` 通过（至少主机平台可编译）。

## 非目标
- 不改 Android UI/业务逻辑/协议语义。
- 不调整 CI/release workflow（仅做依赖升级；如需额外验证链路另开 workflow）。

## 验收标准
- `cd hubmobile; GOWORK=off go test ./... -count=1 -p 1` 通过。
- `hubmobile/go.mod` 不再引用 `myflowhub-core v0.2.1`。
- 合并到 `main` 并 push。

## 3.1) 计划拆分（Checklist）

### ANDDEP0 - 归档旧 plan（已执行）
- 已执行：`git mv plan.md docs/plan_archive/plan_archive_2026-03-12_bump-core-v0.3.0-android-hubmobile-prev.md`

### ANDDEP1 - 升级 hubmobile 的 Core 依赖到 v0.3.0
- 目标：`hubmobile/go.mod` 中 `github.com/yttydcs/myflowhub-core` 从 `v0.2.1` 升级到 `v0.3.0`，并 tidy。
- 说明：若升级后出现编译失败（例如 `IConnection` 接口迁移到 `Pipe()`），允许做最小必要适配，但不改变对外行为与协议语义。
- 涉及文件：`hubmobile/go.mod`、`hubmobile/go.sum`
- 回滚点：revert 本任务提交。

### ANDDEP2 - 回归测试（GOWORK=off）
- 测试点：
  - `cd hubmobile; GOWORK=off go test ./... -count=1 -p 1`

### ANDDEP3 - Code Review（强制）
- 逐项审查：需求覆盖/架构/性能/可读性/扩展性/稳定性与安全/测试覆盖。

### ANDDEP4 - 归档变更（强制）
- 输出：`docs/change/2026-03-12_bump-core-v0.3.0-android-hubmobile.md`

### ANDDEP5 - 合并 / push（需 workflow 结束后执行）
- 在 `repo/MyFlowHub-Android` 合并到 `main` 并 push。

---
## [2026-03-12] Workflow归档 - RFCOMM（Bluetooth Classic）Transport 落地
- 归档文档：
  - `docs/change/2026-03-12_bluetooth-rfcomm-transport.md`
- 分仓结果：
  - `repo/MyFlowHub-Core`：已合并到 `master`（Core @ `33c95b0`）
  - `repo/MyFlowHub-Server`：已合并到 `main`（Server @ `5050441`）
  - `repo/MyFlowHub-SDK`：已合并到 `main`（SDK @ `6c9bf71`）
  - `repo/MyFlowHub-Android`：已合并到 `main`（Android @ `119fe7e`）
- 主要结论：
  - Core 新增 RFCOMM transport，统一到 Pipe 字节流抽象，支持 `listen` / `dial`。
  - Server 支持 TCP + RFCOMM 同时监听，并支持 `-parent-endpoint bt+rfcomm://...`。
  - SDK 支持 `bt+rfcomm://...` endpoint 拨号。
  - Android 提供 RFCOMM Provider 注入入口与 Kotlin 实现。
- 平台策略：
  - Windows：AF_BTH / RFCOMM。
  - Linux：BlueZ D-Bus（UUID-first）+ RFCOMM socket（channel-first 兜底）。
  - Android：Provider 注入（secure 默认开启）。
- 验证：
  - `repo/MyFlowHub-Core`：`go test ./... -count=1`
  - `repo/MyFlowHub-Server`：`go test ./... -count=1`
  - `repo/MyFlowHub-SDK`：`go test ./... -count=1`
  - `repo/MyFlowHub-Android/hubmobile`：`go test ./... -count=1`
- 计划来源：`worktrees/feat-bluetooth-rfcomm-transport/plan.md`
---
# Plan - RFCOMM 发布后下游依赖对齐与发版

## Workflow 信息
- Worktree：`d:\project\MyFlowHub3\worktrees\chore-rfcomm-release-deps`
- 分支：`chore/rfcomm-release-deps`
- 关联仓：
  - `repo/MyFlowHub-Server`
  - `repo/MyFlowHub-SDK`
  - `repo/MyFlowHub-Android`
- 上游已发布：`github.com/yttydcs/myflowhub-core v0.3.1`

## 背景 / 目标
- RFCOMM transport 已合并到各仓主分支，但下游仓当前依赖仍未全部对齐到可拉取的发布版本。
- 目标：
  1) Server 升级到 `myflowhub-core v0.3.1` 并发布新 tag；
  2) SDK 升级到 `myflowhub-core v0.3.1` 并发布新 tag；
  3) Android `hubmobile` 对齐 `core/server/sdk` 的新版本并发布新 tag；
  4) 全过程保持 `GOWORK=off` 可编译 / 可测试。

## 版本策略（按现有仓库习惯）
- Core：已发布 `v0.3.1`
- Server：发布 `v0.0.6`
- SDK：发布 `v0.1.3`
- Android：发布 `v0.1.22`

## 范围
### 必须
- `MyFlowHub-Server/go.mod`：`myflowhub-core v0.3.0 -> v0.3.1`
- `MyFlowHub-SDK/go.mod`：`myflowhub-core v0.2.0 -> v0.3.1`
- `MyFlowHub-Android/hubmobile/go.mod`：
  - `myflowhub-core -> v0.3.1`
  - `myflowhub-sdk -> v0.1.3`
  - `myflowhub-server -> v0.0.6`
  - 保留 `replace github.com/yttydcs/myflowhub-server => ../../MyFlowHub-Server`
- 三仓测试、提交、合并、push、tag

### 不做
- 不改 RFCOMM 功能逻辑
- 不新增协议能力
- 不改 Android UI / Server 运行时行为

## Checklist

### REL1 - Server 依赖升级到 Core v0.3.1
- 目标：升级 Server 对 Core 的版本引用，并验证 `GOWORK=off go test ./...`
- 涉及文件：
  - `repo/MyFlowHub-Server/go.mod`
  - `repo/MyFlowHub-Server/go.sum`
  - `repo/MyFlowHub-Server/docs/change/2026-03-13_bump-core-v0.3.1-server.md`
- 发布：`v0.0.6`

### REL2 - SDK 依赖升级到 Core v0.3.1
- 目标：升级 SDK 对 Core 的版本引用，并验证 `GOWORK=off go test ./...`
- 涉及文件：
  - `repo/MyFlowHub-SDK/go.mod`
  - `repo/MyFlowHub-SDK/go.sum`
  - `repo/MyFlowHub-SDK/docs/change/2026-03-13_bump-core-v0.3.1-sdk.md`
- 发布：`v0.1.3`

### REL3 - Android 对齐 Server / SDK / Core 新版本
- 目标：升级 `hubmobile` 依赖版本并验证 `GOWORK=off go test ./...`
- 涉及文件：
  - `repo/MyFlowHub-Android/hubmobile/go.mod`
  - `repo/MyFlowHub-Android/hubmobile/go.sum`
  - `repo/MyFlowHub-Android/docs/change/2026-03-13_bump-rfcomm-deps-android-hubmobile.md`
- 发布：`v0.1.22`

### REL4 - Code Review + 归档
- 输出：
  - `worktrees/chore-rfcomm-release-deps/docs/change/2026-03-13_rfcomm-release-deps.md`
- 内容：覆盖需求 / 架构 / 版本策略 / 测试 / 风险 / 回滚

### REL5 - 合并 / Push / Tag
- 顺序：
  1) Server merge/push/tag `v0.0.6`
  2) SDK merge/push/tag `v0.1.3`
  3) Android merge/push/tag `v0.1.22`

## 验证命令
```powershell
cd d:\project\MyFlowHub3\worktrees\chore-rfcomm-release-deps\repo\MyFlowHub-Server
$env:GOWORK='off'; go test ./... -count=1

cd ..\MyFlowHub-SDK
$env:GOWORK='off'; go test ./... -count=1

cd ..\MyFlowHub-Android\hubmobile
$env:GOWORK='off'; go test ./... -count=1
```

## 收敛结果
- 状态：已完成并结束 workflow
- 归档文档：docs/change/2026-03-13_rfcomm-release-deps.md`r
- 来源：worktrees/chore-rfcomm-release-deps/plan.md`r
---
## [2026-03-13] Workflow归档 - Link / Router 内核收敛（重大变更）
- 归档文档：
  - `docs/change/2026-03-13_link-router-kernel-major-refactor.md`
- 分仓结果：
  - `repo/MyFlowHub-Core`：已合并到 `master`（merge commit: `50058be`；feature commit: `9c58d75`）
  - `repo/MyFlowHub-Server`：`refactor/link-router-kernel` 与 `main` 无差异，已确认对齐（HEAD: `077865e`）
- 主要结果：
  - Core 新增 `Link / Frame / HeaderRouter` 分层骨架
  - `PreRouting` 改为显式 Header-only 路由决策，保留 `MajorCmd` 逐跳可见与 `TargetID=0` children-only 语义
  - 转发路径维持“不解业务 payload，仅按 header 路由”的策略
  - TCP 发送路径保留 `net.Buffers` 快路径
  - 新增 Core 回归测试，直接固化 Server docs 中的关键路由语义
- 验证：
  - `worktrees/refactor-link-router-kernel/repo/MyFlowHub-Core`：`$env:GOWORK='off'; go test ./... -count=1`
  - `worktrees/refactor-link-router-kernel/repo/MyFlowHub-Server`：基于 workflow `go.work` 联调后 `go test ./... -count=1`
- 计划来源：`worktrees/refactor-link-router-kernel/plan.md`（已收敛并归档到本节与上述 change 文档）
---
## [2026-03-13] Workflow归档 - 下游版本对齐到 Core v0.4.0
- 归档文档：
  - `docs/change/2026-03-13_bump-core-v0.4.0-downstream.md`
- 分仓结果：
  - `repo/MyFlowHub-Server`：已合并到 `main`（HEAD: `86627ce`），已发布 tag `v0.0.7`
  - `repo/MyFlowHub-SDK`：已合并到 `main`（HEAD: `94a1022`），已发布 tag `v0.1.4`
  - `repo/MyFlowHub-Android`：已合并到 `main`（HEAD: `5b9b516`），已发布 tag `v0.1.23`
- 主要结果：
  - `Server` 对齐到 `myflowhub-core v0.4.0`
  - `SDK` 对齐到 `myflowhub-core v0.4.0`
  - `Android/hubmobile` 对齐到 `core v0.4.0`、`sdk v0.1.4`、`server v0.0.7`
  - Android 保留本地 `Server` replace，并新增本地 `SDK` replace，以维持既有 meta-workspace / CI 联调模式
- 验证：
  - `repo/MyFlowHub-Server`：`$env:GOWORK='off'; go test ./... -count=1`
  - `repo/MyFlowHub-SDK`：`$env:GOWORK='off'; go test ./... -count=1`
  - `repo/MyFlowHub-Android/hubmobile`：`$env:GOWORK='off'; go test ./... -count=1`
- 计划来源：`worktrees/chore-bump-core-v0.4.0-downstream/plan.md`

---
## [2026-03-14] Workflow归档 - 剩余下游对齐到 Core v0.4.0
- 归档文档：
  - `docs/change/2026-03-14_bump-core-v0.4.0-remaining.md`
- 计划归档：
  - `docs/plan_archive/plan_archive_2026-03-14_bump-core-v0.4.0-remaining.md`
- 分仓结果：
  - `repo/MyFlowHub-SubProto`：已合并到 `main`（HEAD: `0e69be2`），已发布 tags：
    - `auth/v0.1.3`
    - `file/v0.1.3`
    - `management/v0.1.3`
    - `varstore/v0.1.3`
    - `exec/v0.1.1`
    - `flow/v0.1.1`
    - `forward/v0.1.1`
    - `topicbus/v0.1.1`
  - `repo/MyFlowHub-MetricsNode`：已合并到 `main`（HEAD: `cb9e835`），沿用既有 `debug-latest` 发布流
  - `repo/MyFlowHub-Win`：已合并到 `main`（HEAD: `4212cfe`），已发布 tag `v0.0.2`
- 主要结果：
  - `SubProto` 的 8 个目标 module 已统一对齐到 `myflowhub-core v0.4.0`
  - `MetricsNode` 已统一对齐到 `myflowhub-core v0.4.0` / `myflowhub-sdk v0.1.4`
  - `Win` 已统一对齐到 `myflowhub-core v0.4.0` / `myflowhub-sdk v0.1.4`
  - `SubProto` 与 `Win` 的最低 Go 版本随上游依赖约束提升到 `1.25.0`
- 验证：
  - `repo/MyFlowHub-SubProto`：8 个目标 module 在 `GOWORK=off` 下逐个 `go test ./... -count=1` 通过
  - `repo/MyFlowHub-MetricsNode`：根模块、`nodemobile`、`windows` 在 `GOWORK=off` 下测试通过
  - `repo/MyFlowHub-Win`：`GOWORK=off go test ./... -count=1` 通过
- 计划来源：`worktrees/chore-bump-core-v0.4.0-remaining/plan.md`

---
## [2026-03-14] Workflow归档 - 清理残留 worktree / workflow 目录
- 归档文档：
  - `docs/change/2026-03-14_cleanup-residual-worktrees.md`
- 计划归档：
  - `docs/plan_archive/plan_archive_2026-03-14_cleanup-residual-worktrees.md`
- 主要结果：
  - 已删除残留 git worktree：
    - `fix-subproto-varstore-action-regression`
    - `fix-android-release-gomobile-pin/MyFlowHub-Android`
    - `release-auth-route-index-heal/repo/MyFlowHub-Server`
    - `release-auth-route-index-heal/repo/MyFlowHub-Android`
    - `release-auth-route-index-heal/repo/MyFlowHub-MetricsNode`
  - 已删除旧 workflow 目录壳：
    - `chore-rfcomm-release-deps`
    - `feat-showcase-var-quickpick`
    - `refactor-transport-pipe`
    - `release-auth-route-index-heal`
    - `fix-android-release-gomobile-pin`
  - 当前仅保留长期工作区：
    - `worktrees/MyFlowHub-Core`
    - `worktrees/MyFlowHub-Proto`
    - `worktrees/MyFlowHub-Server`
- 验证：
  - `Get-ChildItem worktrees` 仅剩上述三个保留项
  - 各仓 `git worktree list` 均仅剩主工作区
- 计划来源：`todo.md`

---
## [2026-03-15] Workflow归档 - RFCOMM 短写修复与 Win 发布对齐
- 归档文档：
  - `docs/change/2026-03-15_rfcomm-write-contract-fix.md`
  - `docs/change/2026-03-15_sdk-rfcomm-write-contract-fix.md`
  - `docs/change/2026-03-15_win-rfcomm-write-bump.md`
- 分仓结果：
  - `repo/MyFlowHub-Core`：已合并到 `master`（HEAD: `73938c9`），已发布 tag `v0.4.3`
  - `repo/MyFlowHub-SDK`：已合并到 `main`（HEAD: `6549eb4`），已发布 tag `v0.1.5`
  - `repo/MyFlowHub-Win`：已合并到 `main`（HEAD: `48b57a1`），已发布 tag `v0.0.5`
- 主要结果：
  - Core 新增统一 `write-all` 写出契约，修复字节流短写导致的半帧阻塞
  - `HeaderTcp` 帧写出与 RFCOMM connection 直接发送路径统一改为“写满后返回”
  - SDK `Session.Send` 对齐到相同语义，客户端不再依赖单次 `Write`
  - Win 依赖升级到 `myflowhub-core v0.4.3`、`myflowhub-sdk v0.1.5`
  - Win release 已重新发布，供真实 RFCOMM 场景验证
- 验证：
  - `repo/MyFlowHub-Core`：`GOWORK=off go test ./... -count=1`
  - `repo/MyFlowHub-SDK`：`GOWORK=off go test ./... -count=1`
  - `repo/MyFlowHub-Win`：`GOWORK=off go test ./... -count=1`
- 计划来源：
  - `worktrees/fix-rfcomm-write-contract/repo/MyFlowHub-Core/plan.md`
  - `worktrees/fix-sdk-rfcomm-write-contract/repo/MyFlowHub-SDK/plan.md`
  - `worktrees/chore-win-rfcomm-write-bump/repo/MyFlowHub-Win/plan.md`

---
## [2026-03-15] Workflow归档 - Windows RFCOMM 连接中止修复
- 归档文档：
  - `docs/change/2026-03-15_windows-rfcomm-abort-fix.md`
- 分仓结果：
  - `repo/MyFlowHub-Core`：已合并到 `master`（HEAD 含 merge `fix/rfcomm-win-abort`），已发布 tag `v0.4.4`
- 主要结果：
  - 修复 Windows RFCOMM `Read` 的 EOF 语义（0 字节返回显式 `io.EOF`）
  - 修复 Windows RFCOMM `Write` 在 `WSAEMSGSIZE` 场景下的降级分块发送
  - 补充 Windows 回归测试：EOF 语义与消息尺寸边界
- 验证：
  - `repo/MyFlowHub-Core`：`GOWORK=off go test ./listener/rfcomm_listener -count=1`
  - `repo/MyFlowHub-Core`：`GOWORK=off go test ./... -count=1`
- 计划来源：
  - `worktrees/fix-rfcomm-win-abort/repo/MyFlowHub-Core/plan.md`

---
## [2026-03-15] Workflow归档 - 下游依赖对齐 Core v0.4.4
- 归档文档：
  - `docs/change/2026-03-15_bump-core-v0.4.4-sdk.md`
  - `docs/change/2026-03-15_bump-core-v0.4.4-server.md`
  - `docs/change/2026-03-15_bump-core-v0.4.4-win.md`
- 分仓结果：
  - `repo/MyFlowHub-SDK`：已合并到 `main`，已发布 tag `v0.1.6`
  - `repo/MyFlowHub-Server`：已合并到 `main`，已发布 tag `v0.0.8`
  - `repo/MyFlowHub-Win`：已合并到 `main`，已发布 tag `v0.0.6`
- 主要结果：
  - SDK 依赖升级到 `myflowhub-core v0.4.4`
  - Server 依赖升级到 `myflowhub-core v0.4.4`
  - Win 依赖升级到 `myflowhub-core v0.4.4`、`myflowhub-sdk v0.1.6`
- 验证：
  - `repo/MyFlowHub-SDK`：`GOWORK=off go test ./... -count=1`
  - `repo/MyFlowHub-Server`：`GOWORK=off go test ./... -count=1`
  - `repo/MyFlowHub-Win`：`GOWORK=off go test ./... -count=1`
- 计划来源：
  - `worktrees/chore-bump-core-v0.4.4-sdk/repo/MyFlowHub-SDK/todo.md`
  - `worktrees/chore-bump-core-v0.4.4-server/repo/MyFlowHub-Server/todo.md`
  - `worktrees/chore-bump-core-v0.4.4-win/repo/MyFlowHub-Win/todo.md`

---
## [2026-03-15] Workflow归档 - Windows RFCOMM 流式读写修复
- 归档文档：
  - `docs/change/2026-03-15_windows-rfcomm-stream-fix.md`
- 分仓结果：
  - `repo/MyFlowHub-Core`：已合并到 `master`（HEAD: `6107826`），已发布 tag `v0.4.5`
- 主要结果：
  - Win RFCOMM Pipe 增加内部读缓存，修复“连接成功但 register/login 无返回”的流式读取兼容问题
  - 写路径改为受控分块发送并保留 `WSAEMSGSIZE` 降级，降低蓝牙栈消息边界差异影响
  - 补充回归测试：小块读取缓存复用与写入分块上限
- 验证：
  - `repo/MyFlowHub-Core`：`GOWORK=off go test ./listener/rfcomm_listener -count=1`
  - `repo/MyFlowHub-Core`：`GOWORK=off go test ./... -count=1`
- 计划来源：
  - `worktrees/fix-rfcomm-win-stream/repo/MyFlowHub-Core/plan.md`

---
## [2026-03-15] Workflow归档 - 下游依赖对齐 Core v0.4.5
- 归档文档：
  - `docs/change/2026-03-15_bump-core-v0.4.5-sdk.md`
  - `docs/change/2026-03-15_bump-core-v0.4.5-server.md`
  - `docs/change/2026-03-15_bump-core-v0.4.5-win.md`
- 分仓结果：
  - `repo/MyFlowHub-SDK`：已合并到 `main`，已发布 tag `v0.1.7`
  - `repo/MyFlowHub-Server`：已合并到 `main`，已发布 tag `v0.0.9`
  - `repo/MyFlowHub-Win`：已合并到 `main`，已发布 tag `v0.0.7`
- 主要结果：
  - SDK 依赖升级到 `myflowhub-core v0.4.5`
  - Server 依赖升级到 `myflowhub-core v0.4.5`
  - Win 依赖升级到 `myflowhub-core v0.4.5`、`myflowhub-sdk v0.1.7`
- 验证：
  - `repo/MyFlowHub-SDK`：`GOWORK=off go test ./... -count=1`
  - `repo/MyFlowHub-Server`：`GOWORK=off go test ./... -count=1`
  - `repo/MyFlowHub-Win`：`GOWORK=off go test ./... -count=1`
- 计划来源：
  - `worktrees/chore-bump-core-v0.4.5-sdk/repo/MyFlowHub-SDK/todo.md`
  - `worktrees/chore-bump-core-v0.4.5-server/repo/MyFlowHub-Server/todo.md`
  - `worktrees/chore-bump-core-v0.4.5-win/repo/MyFlowHub-Win/todo.md`

---
## [2026-03-18] Workflow归档 - Flow 事件模式扩展 + Exec 能力注册中心加固 + Win 能力选择器
- 归档文档：
  - `docs/change/2026-03-18_exec-capability-registry-hardening.md`
  - `docs/change/2026-03-18_flow-event-received-trigger-mode.md`
  - `docs/change/2026-03-18_win-flow-exec-capability-picker.md`
- 分仓结果：
  - `repo/MyFlowHub-SubProto`：`main` 已提交（`28e791b`）
  - `repo/MyFlowHub-Proto`：`main` 已提交（`7eef50d`）
  - `repo/MyFlowHub-Server`：`main` 已提交（`099cab8`）
  - `repo/MyFlowHub-Win`：`main` 已提交（`df592e6`）
- 主要结果：
  - `flow.event` 新增 `event_mode=publish|received|any`，支持 `topicbus.received` 触发。
  - `exec` 能力注册中心补齐 `cap_sync_resp` 失败自愈、连接关闭清理、`cap.sync/query` 权限校验。
  - Win Flow 编辑器新增 exec 能力选择器（查询后回填 `target+method`），保持 exec 定点调用语义不变。
- 验证：
  - `repo/MyFlowHub-SubProto/flow`：`go test ./...`
  - `repo/MyFlowHub-SubProto/topicbus`：`go test ./...`
  - `repo/MyFlowHub-SubProto/exec`：`go test ./...`
  - `repo/MyFlowHub-Proto`：`go test ./...`
  - `repo/MyFlowHub-Win`：`go test ./...`
  - `repo/MyFlowHub-Win/frontend`：`npm run build`
- 实施说明：
  - 本轮为历史上下文延续，直接在现有主工作区推进并提交；无新增独立 worktree 需要清理。
