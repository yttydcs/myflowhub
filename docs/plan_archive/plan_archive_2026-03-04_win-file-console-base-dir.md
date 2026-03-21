# Plan - MyFlowHub-Win：File Console（node=2）首次进入报错修复（BaseDir 自动创建 + 路径稳定）

## Workflow 信息
- 范围：单仓库（`MyFlowHub-Win`）
- 分支：`fix/win-file-console-base-dir`
- Worktree：`d:\project\MyFlowHub3\worktrees\fix-win-file-console-base-dir\MyFlowHub-Win`
- Base：`main`
- 规范：
  - `d:\project\MyFlowHub3\guide.md`（commit 信息中文，前缀可英文）
  - `d:\project\MyFlowHub3` 根目录 `AGENTS.md`（阶段纪律、worktree 禁令等）

---

## 0) 当前状态

### 现象
- Win UI 进入 `File Console` 后立即报错：`open file: The system cannot find the file specified.`
- 触发条件：本地节点（你当前 Win 的 nodeId=2），且 File Settings 的 `Base Dir` 为默认值 `./file`。

### 根因（已定位）
- 本地 `list` 走 `internal/services/file/local.go:localList()`，对 `BaseDir + dir` 执行 `os.ReadDir(...)`。
- 默认 `BaseDir=./file` 目录不存在时，`os.ReadDir` 返回系统错误（Windows 文案即 “cannot find the file specified”），并被上层透传到 UI。

---

## 1) 需求分析（已确认）

### 目标
1) `File Console` 首次进入（root list）不再抛系统级 “open file ... cannot find” 错误。
2) 当 `Base Dir` 使用相对路径（默认 `./file`）时，目录应尽量位于**软件同一目录下**，避免因运行目录变化导致文件散落。

### 范围（必须 / 可选 / 不做）
- 必须：
  - 本地 list：当 `BaseDir` 不存在且列 root 目录时，自动创建 `BaseDir` 并返回空列表（成功）。
  - `BaseDir` 相对路径解析：不再依赖进程当前工作目录（CWD），尽量稳定地解析到“软件目录”。
- 可选：
  - 当软件目录不可写时，给出明确可操作的错误信息（提示更改 BaseDir）。
- 不做：
  - 不新增“浏览整机文件系统”的能力；仍以 `BaseDir` 为沙箱根目录。

### 输入 / 输出
- 输入：`nodeId`、`dir`（相对 BaseDir）、`name`、File Settings 的 `BaseDir`
- 输出：目录条目（dirs/files）或文本预览；失败时返回可读错误信息（UI loading 可收敛）

### 边界异常
- `BaseDir` 不存在 / 无权限创建
- `dir/name` 非法（`..`、绝对路径、包含分隔符等）
- 文件不存在 / 非 UTF-8 文本 / 预览截断

### 验收标准
- 默认设置（`BaseDir=./file`）下，进入 `File Console`：不再报错；列表显示为空即可。
- 创建出的目录落点符合“软件同一目录”的预期（相对路径以软件目录为基准）。

### 风险
- 若软件安装在 `Program Files` 等不可写目录，自动创建可能失败；需提供明确错误信息或可配置替代路径。

---

## 2) 架构设计（分析结论）

### 方案 A（采用）：相对 BaseDir 以“可执行文件目录”为基准解析 + root list 兜底创建
- **BaseDir 解析规则**
  - `BaseDir` 为绝对路径：按绝对路径使用。
  - `BaseDir` 为相对路径（例如 `./file`）：解析为 `exeDir/BaseDir`，避免随 CWD 漂移。
- **自动创建规则**
  - 仅当列 root（`dir==""`）且目标目录不存在时：`MkdirAll(BaseDir)` 后返回空列表（成功）。
  - 列非 root 子目录不存在：保持现有语义（返回 not found）。

### 备选对比（不采用）
- 方案 B：仍以 CWD 为基准，仅补 `MkdirAll(./file)`
  - 优点：改动最小
  - 缺点：CWD 不稳定，文件可能散落到不同目录（与“软件同一目录”诉求冲突）
- 方案 C：默认落到 `%APPDATA%/...`
  - 优点：符合常规桌面软件实践
  - 缺点：与当前诉求不一致（“尽量在软件目录”）

### 模块职责
- Frontend：
  - `frontend/src/pages/File.vue`：页面交互（选节点、刷新、预览、传输）
  - `frontend/src/stores/file.ts`：封装 Wails binding 调用 + 事件监听（`file.list/file.text/...`）
- Backend：
  - `internal/services/file/service.go`：List/ReadText 的 send + await 入口
  - `internal/services/file/transfer.go`：本地/远端读写分发 + 任务状态机 + 事件发射
  - `internal/services/file/local.go`：本地文件系统实现（list/read_text）
  - `internal/services/file/config.go`：FilePrefs 读取与归一

### 调用流（本次不改 wire）
- UI 进入 File Console → `requestList(nodeId, dir)` → `FileService.ListSimple(sourceId, hubId, targetId, dir, recursive)`
- 本地节点：`handleLocalRead(list)` → `localList(dir)` → emit `file.list`

### 错误与安全
- 仍以 `BaseDir` 为根目录沙箱；`dir/name` 继续走 sanitize（防 `..`、绝对路径等）。
- 自动创建只发生在 `BaseDir` 本身，避免“列目录即隐式创建任意子目录”。

### 性能与测试策略
- 性能：root list 仅在首次缺失时触发一次 `MkdirAll`；后续无额外 I/O。
- 测试：
  - 单测：覆盖“相对 BaseDir 解析与 CWD 无关”。
  - 手动：覆盖 File Console 首次进入与目录创建落点。

### 可扩展性设计点
- 若后续需要“软件目录不可写时自动 fallback 到 AppData”，在 BaseDir 解析处增加策略分支即可（本次不做）。

---

## 3) 任务清单（Checklist）

### FC1 - 后端：BaseDir 相对路径稳定解析（以 exeDir 为基准）
**目标**
- `BaseDir=./file` 时不再依赖 CWD；落点稳定为 `exeDir/file`。

**涉及模块 / 文件**
- `internal/services/file/config.go`

**验收条件**
- 临时切换进程 CWD 后，最终 BaseDir 仍以 exeDir 为基准。

**测试点**
- 单测覆盖（见 FC3）。

**回滚点**
- revert 本任务提交。

### FC2 - 后端：root list 时 BaseDir 不存在自动创建
**目标**
- root list（`dir==""`）时若 BaseDir 不存在，自动创建并返回空列表（成功）。

**涉及模块 / 文件**
- `internal/services/file/local.go`

**验收条件**
- 默认设置下首次进入 `File Console` 不再出现系统错误提示。

**测试点**
- 手动冒烟（见 FC4）。

**回滚点**
- revert 本任务提交。

### FC3 - 测试：补充 BaseDir 解析单测
**目标**
- 覆盖相对 BaseDir 解析与 CWD 无关的关键路径。

**涉及模块 / 文件**
- `internal/services/file/*_test.go`（新增）

**验收条件**
- `go test ./... -count=1` 通过。

**回滚点**
- revert 测试提交（不影响功能）。

### FC4 - 冒烟（手动）
**步骤**
1) 启动 Win App 并 Connect（nodeId=2）
2) 打开 `File Console`
3) 确认不再出现 “open file ... cannot find”
4) 确认 `file/` 目录已在软件目录下创建

**注意**
- 若软件目录不可写：记录错误信息，并在 File Settings 中把 BaseDir 改到可写目录后重试。

**回滚点**
- 无（仅验证）。

### FC5 - Code Review（强制）+ 归档变更（强制）
**目标**
- 完成强制 Code Review 清单；输出 `docs/change/YYYY-MM-DD_*.md`。

**验收条件**
- Review 通过；变更文档包含任务映射、关键权衡、验证结果与回滚方案。

