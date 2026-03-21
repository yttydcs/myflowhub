# 2026-03-05 - Win：File Console 按钮可理解性修复 + 拖拽放置导入（本地节点）

## 变更背景 / 目标
- 背景：
  - 用户反馈 `File Console` 的 `Up` / `Download` 点击“似乎没有反应”；
  - 现状不支持拖拽文件直接放入当前目录。
- 目标：
  1) 明确按钮行为与触发条件，减少“无反馈”误解；
  2) 支持将系统文件拖拽到 File Console 当前目录导入（V1：本地节点）；
  3) 保持安全默认：禁止目录穿越，默认不覆盖同名文件。

## 具体变更内容（新增 / 修改 / 删除）

### 新增
- `internal/services/file/import.go`
  - 新增 `ImportLocalFiles(targetDir, sourcePaths, overwrite)` Wails 暴露接口。
  - 新增导入结果结构：
    - `FileImportResult`（`dir/imported/skipped`）
    - `FileImportItem`
    - `FileImportFailure`
  - 导入逻辑要点：
    - 目标目录走 `fileSanitizeDir`，目标文件路径走 `fileResolvePaths`，确保只能落在 `BaseDir` 下；
    - 只允许 regular file，目录与非常规文件跳过并返回原因；
    - 默认不覆盖同名文件（`target already exists`）；
    - 复制采用临时文件 + rename，避免半写入文件污染目标路径；
    - 跳过项与成功项分别返回，支持部分成功。
- `internal/services/file/import_test.go`
  - 新增导入逻辑测试覆盖：
    - 正常导入 + 目录跳过；
    - 无覆盖模式下同名跳过；
    - 覆盖模式替换文件；
    - 非法目标目录拦截。

### 修改
- `main.go`
  - 启用 Wails 拖拽能力：
    - `DragAndDrop.EnableFileDrop = true`。
- `frontend/src/stores/file.ts`
  - 新增 `FileImportResult` 相关类型；
  - 新增 `importLocalFiles()` 调用后端 `ImportLocalFiles`。
- `frontend/src/pages/File.vue`
  - 增加 `Up` / `Download` 的动态 `title` 提示，解释禁用原因；
  - 接入 `OnFileDrop/OnFileDropOff`；
  - 目录列表区声明 `--wails-drop-target: drop`，支持目标区域拖放；
  - 增加拖拽高亮样式（`wails-drop-target-active`）；
  - 拖拽导入后 toast 反馈成功/跳过数量，并刷新目录列表；
  - 远端节点拖拽时给出明确提示（仅本地节点可导入）。

### 删除
- 无。

## 对应计划任务映射（todo.md）
- `FC-DND-1` 按钮行为与禁用提示：已完成。
- `FC-DND-2` 启用并接入 Wails 文件拖拽：已完成。
- `FC-DND-3` 后端导入能力（安全默认 + 错误处理）：已完成。
- `FC-DND-4` 单测与回归验证：已完成（Go 测试）。
- `FC-DND-5` Code Review + 归档：已完成（本文 + 评审结论）。

## 关键设计决策与权衡（性能 / 扩展性）
- 拖拽实现选择：
  - 采用 Wails 原生 file-drop（绝对路径）而非前端二进制上传；
  - 原因：避免大文件经 JS 内存中转，减少拷贝和序列化开销。
- 安全默认：
  - 默认 `overwrite=false` 防误覆盖；
  - 路径统一经现有 sanitize/resolve 逻辑，禁止越界写入。
- 可扩展点：
  - 当前接口已预留 `overwrite` 参数；
  - 返回结构为逐文件结果，后续可扩展为进度事件与批处理策略。

## 测试与验证方式 / 结果

### 自动测试
```powershell
$env:GOWORK='off'
go test ./internal/services/file -count=1
go test ./... -count=1
```
- 结果：通过。

### 手动验证建议（Windows 运行态）
1) 启动应用，进入 `File Console`，切换本地节点；
2) 将 1~N 个文件拖拽到目录列表区域；
3) 预期：
   - 拖拽区域高亮；
   - 导入成功 toast；
   - 列表刷新后可见文件；
4) 再拖同名文件：
   - 预期提示 skipped（默认不覆盖）；
5) 切到远端节点后拖拽：
   - 预期提示仅本地节点支持导入。

## 潜在影响与回滚方案

### 潜在影响
- 大文件导入耗时期间暂无进度条（V1 仅 toast + 完成后刷新）；
- 前端构建环境若未安装 node 依赖，无法在本地直接执行 `vite build` 验证 UI 打包。

### 回滚方案
- 回滚以下文件即可完整撤销本次能力：
  - `main.go`
  - `frontend/src/pages/File.vue`
  - `frontend/src/stores/file.ts`
  - `internal/services/file/import.go`
  - `internal/services/file/import_test.go`

