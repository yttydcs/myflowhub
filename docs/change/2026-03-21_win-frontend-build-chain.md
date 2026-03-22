# 2026-03-21 Win 前端构建链路恢复

## 变更背景 / 目标
- 用户通过 `.\run-dev.ps1` 启动 Win 客户端时，Wails 在前端阶段失败，错误为 `'vite' is not recognized as an internal or external command`。
- 进一步验证发现，绕过 `vite` 入口后仍会在 `lucide-vue-next@0.575.0` 上失败，因为安装产物缺失 `dist/esm/icons/mic-vocal.js`。
- 本次目标是恢复 Win 仓库在当前环境下的可重复前端构建链路，使 `wails dev/build` 与 `run-dev.ps1` 的依赖安装阶段可正常通过。

## 具体变更内容（新增 / 修改 / 删除）
### 修改
- `frontend/package.json`
  - `dev/build/preview` 脚本改为直接调用 `node ./node_modules/vite/bin/vite.js`，不再依赖 `node_modules/.bin/vite.cmd`。
  - 将 `lucide-vue-next` 从 `^0.575.0` 固定为 `0.577.0`。
- `frontend/package-lock.json`
  - 同步锁定 `lucide-vue-next@0.577.0` 的 `resolved` 与 `integrity`，确保 Wails 自动 `npm install` 时拿到完整包。

### 新增
- `docs/change/2026-03-21_win-frontend-build-chain.md`
  - 记录本次构建链路修复的背景、设计决策、验证结果与回滚方案。

### 删除
- 无。

## 对应 plan.md 任务映射
- `WIN-BUILD-1`
  - `frontend/package.json` 中改写 Vite 启动脚本。
- `WIN-BUILD-2`
  - `frontend/package.json`、`frontend/package-lock.json` 固定 `lucide-vue-next@0.577.0`。
- `WIN-BUILD-3`
  - 执行 `npm run build` 与 `wails build -debug -skipembedcreate -nopackage` 验证。
- `WIN-BUILD-4`
  - 完成 Code Review 与本归档文档。

## 关键设计决策与权衡
- 直接调用 `node ./node_modules/vite/bin/vite.js`，而不是继续依赖裸 `vite` 或 `node_modules/.bin/vite.cmd`。
  - 原因：当前环境中曾稳定复现 `vite` 包已安装但 `.bin/vite.cmd` 缺失，直接调用包入口更稳，且已在本机验证可构建。
- 将 `lucide-vue-next` 固定为精确版本 `0.577.0`，而不是继续使用 `^0.x` 范围。
  - 原因：`0.575.0` 在当前镜像源上的 tarball 产物不完整；精确版本可降低 `npm install` 漂移到坏包的风险。
- 不修改 `scripts/run-dev.ps1`。
  - 原因：根因在 Win 仓库前端依赖与脚本入口，不在控制脚本本身。

## 测试与验证方式 / 结果
- 前端依赖刷新：
  - 执行：`cd frontend && npm install`
  - 结果：通过
- 前端构建：
  - 执行：`cd frontend && npm run build`
  - 结果：通过
- Wails 构建：
  - 执行：`$env:GOWORK='off'; wails build -debug -skipembedcreate -nopackage`
  - 结果：通过，生成 `build/bin/myflowhub-win.exe`

## 潜在影响与回滚方案
### 潜在影响
- `lucide-vue-next` 被固定为精确版本，后续若需要新图标或上游修复，需要显式升级版本与锁文件。
- Vite 构建仍有既有的大 chunk 告警，但不影响本次启动问题修复。

### 回滚方案
- 回退以下文件即可恢复到修复前状态：
  - `frontend/package.json`
  - `frontend/package-lock.json`

## 子Agent执行轨迹（Task ID → Agent → Worktree → 文件 → 验收结果）
- 本次未使用子Agent
  - 原因：写集集中、验证路径单一，由主Agent本地连续处理更安全。
