# Todo - MetricsNode CI 构建失败修复（Windows embed dist + gomobile tidy）

- Worktree：`d:\project\MyFlowHub3\worktrees\fix-metricsnode-ci`
- Branch：`fix/metricsnode-ci`
- Date：2026-03-07
- 状态：已完成（已合并到 `repo/MyFlowHub-MetricsNode`：MetricsNode @ `92f4186`）

## 项目目标与当前状态

### 目标

1. 修复 GitHub Actions `ci` 构建失败，使其稳定产出：
   - Windows：`windows.exe`
   - Android：`app-debug.apk` + `myflowhub.aar`
2. 恢复 `push(main)` → `publish-debug-latest` 发布链路（避免被 `needs` 阻塞）。

### 当前状态（问题复现）

- Windows：`wails build` 失败，错误为 `pattern all:frontend/dist: no matching files found`
  - 根因：`windows/main.go` 使用 `//go:embed all:frontend/dist`，clean checkout 下 `windows/frontend/dist` 不存在且为空目录，Go embed pattern 必须至少命中 1 个文件。
- Android：`scripts/build_aar.sh`（`gomobile bind`）失败，错误为 `go: updates to go.mod needed; to update it: go mod tidy`
  - 根因：`nodemobile` 子模块依赖图未对齐（间接依赖仍指向旧版本），gomobile 在只读依赖模式下拒绝自动改写。

## 可执行任务清单（Checklist）

- [x] CI1 需求与验收基线确认
- [x] CI2 修复 Windows：确保 `frontend/dist` 非空（go:embed）
- [x] CI3 修复 Android：提交 `nodemobile` 模块 tidy
- [x] CI4 回归验证（本地）
- [x] CI5 Code Review + 归档 docs/change

## 任务详情（摘要）

### CI2 修复 Windows：`frontend/dist` 占位

- 涉及文件：
  - `repo/MyFlowHub-MetricsNode/.github/workflows/ci.yml`
  - `repo/MyFlowHub-MetricsNode/scripts/build-windows.ps1`
- 实施：
  - CI：`wails build` 前创建 `windows/frontend/dist/.keep`，保证 embed pattern 命中。
  - 脚本：本地构建脚本同样在执行 Wails 前确保 `dist/.keep` 存在，避免 clean checkout 本地也失败。
- 关键权衡：
  - 不提交 `dist/` 产物；仅在构建阶段生成占位文件，保持仓库整洁且可回滚。

### CI3 修复 Android：`nodemobile` tidy

- 涉及文件：
  - `repo/MyFlowHub-MetricsNode/nodemobile/go.mod`
  - `repo/MyFlowHub-MetricsNode/nodemobile/go.sum`
- 实施：
  - 执行 `GOWORK=off go mod tidy`
  - 对齐关键间接依赖：`github.com/yttydcs/myflowhub-sdk` → `v0.1.2`（与仓库主模块一致）

## 验证记录

### Windows

- 命令：
  - `cd repo/MyFlowHub-MetricsNode/windows; $env:GOWORK='off'; wails build -platform windows/amd64 -nopackage`
- 预期：
  - 生成 `windows/build/bin/windows.exe`

### Android（gomobile + APK）

- AAR：
  - `bash scripts/build_aar.sh` 成功生成 `android/app/libs/myflowhub.aar`
- APK：
  - `cd android; ./gradlew :app:assembleDebug`
- 备注（本地 Windows 验证）：
  - `android/local.properties` 的 `sdk.dir/ndk.dir` 使用 `D:/...` 形式路径更稳健（避免 Gradle/AGP 在某些场景下对 `D:\...` 报路径语法错误）。

## 回滚方案

- 回滚单次修复提交即可恢复到旧行为：
  - 删除 `.github/workflows/ci.yml` 中 `Prepare frontend dist (go:embed)` step
  - 回退 `nodemobile/go.mod` 与 `nodemobile/go.sum`
  - 回退 `scripts/build-windows.ps1` 的 `dist/.keep` 预创建逻辑

