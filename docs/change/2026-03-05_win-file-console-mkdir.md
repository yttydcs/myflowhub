# 变更归档：MyFlowHub-Win File Console 新建文件夹（mkdir）

## 变更背景 / 目标
- 背景：File Console 已支持目录浏览、下载、offer、拖拽导入，但缺少“创建文件夹”能力，导致本地整理和远端目录预创建流程不完整。
- 目标：
  - 在 File Console 增加 `New Folder` 入口；
  - 前端可输入目录名并调用后端；
  - 后端支持本地与远端统一走 `file.write(op=mkdir)` 语义，并返回明确成功/失败。

## 具体变更内容
### 新增
- 后端新增目录创建 API（await 写响应）：
  - `FileService.CreateDir(...)`
  - `FileService.CreateDirSimple(...)`
  - `FileService.writeAndAwait(...)`（复用 write resp 同步确认能力）
- 后端本地写入分发：
  - `handleLocalWrite(req protocol.WriteReq)`（当前支持 `op=mkdir`）
  - `localMkdir(...)` / `mkdirInBase(...)`
- 前端能力：
  - `useFileStore().createDir(targetNodeId, dir, name)`
  - File Console 顶部工具栏新增 `New Folder` 按钮
  - 新增 `New Folder` 弹窗（输入目录名、校验、调用、提示、刷新）
- 测试：
  - 新增 `internal/services/file/local_mkdir_test.go`

### 修改
- `internal/services/file/local.go`：
  - 增加 `protocol/file` 引入；
  - 增加本地 mkdir 路径与 `handleLocalWrite`。
- `internal/services/file/service.go`：
  - 增加 `opMkdir` 常量；
  - 增加 `CreateDir/CreateDirSimple/writeAndAwait`。
- `frontend/src/stores/file.ts`：
  - 增加 `createDir` 封装并导出。
- `frontend/src/pages/File.vue`：
  - 增加 `New Folder` 按钮、弹窗与提交流程。

### 删除
- 无。

## 对应 plan/todo 任务映射
- `WIN-MKDIR-1` -> `service.go` / `local.go` 后端 mkdir API + 本地写处理
- `WIN-MKDIR-2` -> `frontend/src/stores/file.ts` 增加 `createDir`
- `WIN-MKDIR-3` -> `frontend/src/pages/File.vue` 增加按钮与弹窗
- `WIN-MKDIR-4` -> 新增单测 + `go test` 回归 + 本文档归档

## 关键设计决策与权衡
- 协议对齐：mkdir 使用既有 `action=write`，`op=mkdir`，避免新增 action，降低协议扩张成本。
- 可拦截性：mkdir 走 MajorCmd 控制面链路，父节点可在路由链路中观察并执行权限策略。
- 稳定性：前端 `createDir` 与后端 `writeAndAwait` 都做输入校验和错误透传，避免“点击无反馈”。
- 性能：
  - mkdir 仅一次控制消息与一次响应，无大数据传输；
  - 本地路径处理采用 sanitize + resolve，避免额外目录扫描与重复 I/O。
- 扩展点：
  - `handleLocalWrite` 保留 op 分发结构，可后续扩展 `rename/remove` 等写操作；
  - `writeAndAwait` 可复用于需要同步确认的其他 write op。

## 测试与验证方式 / 结果
- Go 回归：
  - 命令：`GOWORK=off go test ./... -count=1`
  - 结果：通过（含 `internal/services/file`）
- 新增单测：
  - `TestMkdirInBase_CreateSuccess`
  - `TestMkdirInBase_IdempotentWhenDirExists`
  - `TestMkdirInBase_FileConflict`
  - `TestMkdirInBase_InvalidInput`
- 前端构建：
  - 命令：`npm run build`
  - 结果：当前环境失败，原因：`vite` 不存在（`'vite' is not recognized...`），未能在本机完成前端构建验证。

## Code Review（3.3）结论
- 需求覆盖：通过
  - 已覆盖“创建文件夹”后端 API、前端按钮入口、交互提示与刷新。
- 架构合理性：通过
  - 复用现有 write 控制流，不引入旁路协议。
- 性能风险：通过
  - 未引入 N+1、重复计算或多余 I/O。
- 可读性与一致性：通过
  - 命名与现有 `FileService`/store 风格一致，错误信息可读。
- 可扩展性与配置化：通过
  - 本地写分发与 await 写流程可复用扩展。
- 稳定性与安全：通过
  - 目录/名称 sanitize、越界防护、冲突处理与错误透传完整。
- 测试覆盖：通过（前端构建验证受环境依赖限制）
  - Go 单测与全量 Go 测试通过；
  - 前端仅完成静态代码审查，构建需补齐 vite 依赖后复测。

## 潜在影响与回滚方案
- 潜在影响：
  - 若远端节点未升级到支持 `op=mkdir` 的子协议版本，可能返回 `invalid op`。
- 回滚方案：
  1. 回滚 `service.go` 中 `CreateDir/CreateDirSimple/writeAndAwait`。
  2. 回滚 `local.go` 的 `mkdirInBase/localMkdir/handleLocalWrite`。
  3. 回滚 `frontend/src/stores/file.ts` 的 `createDir`。
  4. 回滚 `frontend/src/pages/File.vue` 的 `New Folder` UI。
  5. 删除新增测试 `local_mkdir_test.go`（如需完全恢复旧行为）。
