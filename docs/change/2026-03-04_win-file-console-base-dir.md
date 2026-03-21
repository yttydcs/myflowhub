# 2026-03-04 - Win：File Console 首次进入报错修复（BaseDir 自动创建 + 相对路径稳定）

## 背景 / 目标
- 现象：Win UI 进入 `File Console` 时立即提示 `open file: The system cannot find the file specified.`。
- 根因：本地节点 list 走 `os.ReadDir(BaseDir)`，默认 `BaseDir=./file` 在首次使用时目录不存在，系统错误透传到 UI。
- 目标：
  1) 默认配置下首次进入 `File Console` 不再报错；
  2) `BaseDir` 使用相对路径时，尽量落在软件同一目录下，避免随 CWD 漂移导致文件散落。

## 具体变更内容
### 修改（后端）
- `internal/services/file/config.go`
  - `fileConfig().BaseDir` 由“原样使用 prefs.BaseDir”改为运行时解析：
    - 绝对路径：直接使用
    - 相对路径：以 `os.Executable()` 的目录为基准拼接（`exeDir/baseDir`），避免依赖 CWD
  - `exeDir` 使用 `sync.Once` 缓存，避免频繁获取可执行文件路径
- `internal/services/file/local.go`
  - 本地 list 在 `dir==""`（root list）时先 `MkdirAll(BaseDir)`：
    - 目录不存在：自动创建后返回空列表（成功）
    - 目录不可创建：返回可读错误（包含根因）

### 新增（测试）
- `internal/services/file/config_test.go`
  - 覆盖：相对 `BaseDir` 在切换 CWD 后仍解析到同一落点（以 `exeDir` 为基准）

## Plan.md 任务映射
- FC1 - 后端：BaseDir 相对路径稳定解析 ✅
- FC2 - 后端：root list 时 BaseDir 不存在自动创建 ✅
- FC3 - 测试：补充 BaseDir 解析单测 ✅
- FC4 - 冒烟（手动）🟨（需人工执行）
- FC5 - Code Review + 归档变更 ✅（本文）

## 关键设计决策与权衡
- 为什么用 `exeDir` 而不是 CWD：
  - CWD 在 dev/build/快捷方式启动等场景可能变化，导致 `./file` 位置不稳定；
  - `exeDir` 更符合“目录与软件同一目录”的诉求。
- 为什么只在 root list 自动创建：
  - 避免“浏览即创建任意子目录”的副作用；
  - 首次进入体验收敛，同时不改变对子目录不存在的既有语义。

## 测试与验证方式 / 结果
### 自动测试
```powershell
$env:GOWORK='off'
go test ./... -count=1
```
结果：通过。

### 冒烟（手动，建议在 Windows 联调环境执行）
1) 启动 Win App 并 Connect（本地节点 nodeId=2）
2) 打开 `File Console`
3) 预期：不再出现 `open file ... cannot find`；列表为空可接受
4) 检查：软件目录下出现 `file/`（默认 BaseDir）目录

## 潜在影响与回滚方案
### 潜在影响
- 若软件安装目录不可写（例如 `Program Files`），自动创建 `file/` 可能失败并返回明确错误；可通过 File Settings 将 `BaseDir` 改为可写绝对路径规避。

### 回滚方案
- revert 本次分支相关提交（恢复为原先 BaseDir 相对 CWD 的行为 + 不自动创建目录）。

