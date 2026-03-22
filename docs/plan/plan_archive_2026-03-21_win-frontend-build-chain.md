# Win 前端构建链路恢复

## Workflow 信息
- 仓库：MyFlowHub-Win
- 分支：fix/win-vite-npm-exec
- Base：main
- Worktree：D:\project\MyFlowHub3\worktrees\fix-win-vite-npm-exec\MyFlowHub-Win
- 当前阶段：4 归档变更

## 1. 需求分析

### 目标
- 修复 `repo/MyFlowHub-Win` 在当前机器上执行 `wails dev/build` 时前端阶段失败的问题，恢复 `.\run-dev.ps1` 所依赖的 Win 启动链路。

### 范围
- 必须：
  - 保证 Win 前端脚本在当前 npm 环境下不依赖 `node_modules/.bin/vite.cmd`。
  - 保证 Wails 自动 `npm install` 后得到的前端依赖集合可正常构建。
  - 保证 `wails dev/build` 走到前端阶段可以正常执行。
- 可选：
  - 验证 `run-dev.ps1` 所依赖的 Win 启动链路是否恢复。
- 不做：
  - 不修改 `scripts/run-dev.ps1` 的窗口管理逻辑。
  - 不修改 MetricsNode 仓库。
  - 不修改业务页面与协议逻辑。

### 使用场景
- 用户通过 `.\scripts\run-dev.ps1` 启动 `repo/MyFlowHub-Win`。
- Wails 进入 `frontend:build` / `frontend:dev:watcher` 时调用 `npm run build` / `npm run dev`。

### 功能需求
- `frontend/package.json` 的 `dev/build/preview` 脚本必须在当前 npm 11.8.0 环境下可执行。
- `frontend/package-lock.json` 必须锁定到可正常构建的 `lucide-vue-next` 版本。

### 非功能需求
- 改动最小化。
- 保持跨环境兼容，不依赖本机特殊 npm `.bin` 行为。

### 输入输出
- 输入：`wails dev`、`wails build`、`npm run build`
- 输出：前端正常构建，不再报 `vite` 不存在。

### 边界异常
- 当前环境中 `frontend/node_modules` 内存在 `vite` 包，但缺失 `.bin/vite.cmd`，说明不能假设 npm 一定会生成 `.bin` 启动器。
- 当前锁定的 `lucide-vue-next@0.575.0` 安装产物缺失 `dist/esm/icons/mic-vocal.js`，导致构建时解析包总导出文件失败。

### 验收标准
- `cd frontend && npm run build` 通过。
- `wails build -debug -skipembedcreate -nopackage` 通过。

### 风险
- 需要选择一个包含完整产物的 `lucide-vue-next` 版本，并同步更新锁文件，避免 Wails 的自动安装流程再次拉取坏包。

### 问题清单
- 阻塞：否

## 2. 架构设计（分析）

### 总体方案
- 根因：
  - 当前机器上 `frontend/node_modules/vite` 已安装，但 `frontend/node_modules/.bin/vite.cmd` 缺失。
  - `package.json` 脚本直接写 `vite`，会依赖 `.bin` 注入，因此在当前环境稳定失败。
  - `frontend/package-lock.json` 当前锁定的 `lucide-vue-next@0.575.0` 包产物不完整，缺失 `dist/esm/icons/mic-vocal.js`，即使绕过 `vite` 入口问题，构建仍会失败。
- 方案：
  - 将脚本改为直接调用 `node ./node_modules/vite/bin/vite.js`，绕过对 `.bin` 启动器的依赖。
  - 将 `lucide-vue-next` 固定到确认包含完整构建产物的版本，并刷新锁文件，保证 `npm install` 与 Wails 自动安装结果一致。

### 模块职责
- `frontend/package.json`
  - 负责 Win 前端 dev/build/preview 入口与前端依赖版本声明。
- `frontend/package-lock.json`
  - 负责锁定 Wails 自动 `npm install` 的实际依赖产物。

### 错误与安全
- 不做全局 npm 配置修改。
- 不依赖删除 `node_modules` 作为长期修复手段。
- 不修改 `scripts/run-dev.ps1`，避免把依赖问题误修成脚本问题。

### 性能与测试策略
- 性能影响可忽略，仅脚本入口变化。
- 验证：
  - `npm run build`
  - `wails build -debug -skipembedcreate -nopackage`

### 可扩展性设计点
- 后续若切换包管理器，只需在脚本层统一调整，不影响业务代码。
- 依赖包若再次出现上游坏包，可继续通过锁文件精确固定，避免影响业务层。

## 3.1 计划拆分（Checklist）

- [x] `WIN-BUILD-1`
  - Owner：主Agent
  - 目标：将 Win 前端 `dev/build/preview` 脚本改为直接调用本地 Vite 入口。
  - 涉及文件：`frontend/package.json`
  - 验收条件：不再直接调用裸 `vite`。
  - 测试点：查看脚本字符串。
  - 回滚点：回退 `frontend/package.json`。

- [x] `WIN-BUILD-2`
  - Owner：主Agent
  - 目标：锁定完整的 `lucide-vue-next` 版本，修复自动安装后的前端依赖完整性。
  - 涉及文件：`frontend/package.json`、`frontend/package-lock.json`
  - 依赖：`WIN-BUILD-1`
  - 验收条件：锁文件不再解析到缺失 `mic-vocal.js` 的坏包。
  - 测试点：检查依赖版本与构建日志。
  - 回滚点：回退 `frontend/package.json`、`frontend/package-lock.json`。

- [x] `WIN-BUILD-3`
  - Owner：主Agent
  - 目标：验证前端与 Wails 构建链路恢复。
  - 涉及文件：无
  - 依赖：`WIN-BUILD-2`
  - 验收条件：
    - `npm run build` 通过。
    - `wails build -debug -skipembedcreate -nopackage` 通过。
  - 测试点：构建日志。
  - 回滚点：若失败，回退脚本改动并重新定位。

- [x] `WIN-BUILD-4`
  - Owner：主Agent
  - 目标：完成 Code Review 与归档文档。
  - 涉及文件：`plan.md`、`docs/change/*`
  - 依赖：`WIN-BUILD-3`
  - 验收条件：文档可独立交接。
  - 回滚点：删除新增文档。

## 3.3 Code Review

### 需求覆盖
- 通过：已同时修复 `vite` 启动入口依赖 `.bin` 与 `lucide-vue-next@0.575.0` 坏包导致的构建失败，覆盖 `run-dev.ps1` 所依赖的 Wails 前端安装/构建链路。

### 架构合理性
- 通过：改动仅限前端构建入口与依赖锁文件，未侵入业务页面、Go 运行逻辑或启动脚本控制流。

### 性能风险
- 通过：运行时无额外 I/O 或计算路径；仅构建入口与依赖版本调整。
- 注意：Vite 仍提示单 chunk 大于 500 kB，这是仓库既有体积告警，不是本次回归。

### 可读性与一致性
- 通过：脚本入口明确指向本地 Vite 可执行文件，依赖版本固定可审计。

### 可扩展性与配置化
- 通过：后续如需升级 Vite 或图标库，可继续在 `frontend/package.json`/`frontend/package-lock.json` 单点维护。

### 稳定性与安全
- 通过：避免依赖本机 `.bin` 生成行为，也避免 `npm install` 在镜像源上漂移到已知坏包版本。

### 测试覆盖情况
- 通过：
  - `cd frontend && npm install`
  - `cd frontend && npm run build`
  - `$env:GOWORK='off'; wails build -debug -skipembedcreate -nopackage`

### 子Agent治理与审计
- 通过：未使用子Agent。
- 原因：写集集中在 `frontend/package*.json` 与文档，验证依赖单条关键路径，本地串行更安全。

## 并行性评估
- 不使用子Agent。
- 原因：单文件脚本修复，写集高度集中，验证依赖单条关键路径。
