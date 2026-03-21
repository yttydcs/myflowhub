# 2026-03-06 - Win：Send Offer 远端目录可调 + 目标节点树状选择

## 变更背景 / 目标
- 背景：
  - 现有 `Send Offer` 只能把 `dir` 同时用于“本地源文件定位”和“远端落盘目录”，无法独立控制 `req.dir`；
  - Offer 目标节点只能手输 `Target Node ID`，缺少可视化选择与层级浏览。
- 目标：
  1. 支持发送 Offer 时显式设置远端落盘目录；
  2. Offer 目标节点改为树状选择，交互风格对齐 `Devices`；
  3. 保持旧调用兼容，不破坏已有 `StartOffer` 路径。

## 具体变更内容（新增 / 修改 / 删除）

### 修改
- `internal/services/file/transfer.go`
  - 新增 `StartOfferToDir(sourceID, hubID, consumer, sourceDir, name, remoteDir, wantHash)`；
  - 旧 `StartOffer(...)` 改为兼容包装，默认 `remoteDir = sourceDir`；
  - Offer 任务元数据拆分：
    - `dir` 保存远端目录（用于协议 `WriteReq.Dir`）；
    - `localDir/localPath` 保存本地源目录与绝对路径；
  - `RetryTask` 的 Offer 重试改为走 `StartOfferToDir`，优先使用 `localDir` 作为源目录。

- `frontend/src/stores/file.ts`
  - `startOffer` 签名改为 `(consumer, sourceDir, name, remoteDir, wantHash)`；
  - 优先调用新 binding：`StartOfferToDir`；
  - 兼容旧后端：若无 `StartOfferToDir` 且 `sourceDir !== remoteDir`，明确报错，避免静默错行为。

- `frontend/src/pages/File.vue`
  - Offer 表单增加 `remoteDir` 字段，默认回填当前目录；
  - Offer 提交时传入 `sourceDir=currentDir` 与用户编辑后的 `remoteDir`；
  - `Target Node ID` 文本输入替换为树状节点选择组件；
  - 增加目标节点校验（必须选中且不能是本地节点）。

### 新增
- `frontend/src/components/file/OfferNodeTreePicker.vue`
  - 提供树状节点选择 UI；
  - 支持根节点加载、节点展开/折叠、节点选择、失败重试；
  - 使用 `ManagementService.ListNodesSimple` 按需懒加载子节点，避免一次性全量加载。

### 删除
- 无。

## 对应计划任务映射（todo.md）
- `OFFER-DIR-1`：已完成。
- `OFFER-DIR-2`：已完成。
- `OFFER-DIR-3`：已完成。
- `OFFER-DIR-4`：已完成。
- `OFFER-DIR-5`：已完成。
- `OFFER-DIR-6`：已完成。

## 关键设计决策与权衡（性能 / 扩展性）
- 兼容性优先：
  - 通过保留 `StartOffer` 并新增 `StartOfferToDir`，避免破坏旧调用链路。
- 语义分离：
  - 源目录与远端目录分离，`req.dir` 与实际本地取文件路径不再耦合；
  - 重试逻辑读取任务元数据保持一致语义，避免 remote/source 混淆。
- 性能：
  - 树选择器采用懒加载子节点，仅在展开时请求，降低首屏与网络开销；
  - 未引入额外轮询。
- 可扩展性：
  - 树选择器组件独立，后续可复用到其他“选目标节点”场景；
  - 后端 `StartOfferToDir` 为后续更细粒度落盘策略预留扩展点。

## 测试与验证方式 / 结果

### 后端测试
```powershell
$env:GOWORK='off'; go test ./...
```
- 结果：通过。

### 前端构建
```powershell
cd frontend
npm install
npm run build
```
- 结果：失败（环境现状问题），报错为 `Home.vue` 引用 `../../wailsjs/go/session/SessionService` 无法解析；与本次改动文件无直接关联。

### 前端 TypeScript 检查
```powershell
cd frontend
npx tsc --noEmit
```
- 结果：失败（环境现状问题），存在项目级 `*.vue` / `wailsjs` 声明缺失。

### 手工验证建议
1. 选择本地节点并选中文件，打开 `Send Offer`。
2. 在树中选择目标远端节点，修改 `Remote Dir` 后发送。
3. 在接收端检查 Offer 提示目录与落盘目录是否为设置值。
4. 触发失败后在 Tasks 中执行 Retry，确认仍使用原 sourceDir + remoteDir 语义。

## Code Review 结论（阶段 3.3）
- 需求覆盖：通过。
- 架构合理性：通过（后端语义分离 + 前端组件化，兼容旧接口）。
- 性能风险：通过（树懒加载，无新增高频 I/O）。
- 可读性与一致性：通过（命名清晰，职责边界明确）。
- 可扩展性与配置化：通过（独立 picker 组件，可复用）。
- 稳定性与安全：通过（路径与 ID 校验保留/增强）。
- 测试覆盖情况：部分通过（Go 测试通过；前端构建受环境缺失依赖阻塞）。

## 潜在影响与回滚方案

### 潜在影响
- 若运行环境仍缺少 `wailsjs` 绑定，前端完整构建无法通过；
- 树选择器依赖 `ManagementService.ListNodesSimple`，该 binding 不可用时会展示错误并允许重试。

### 回滚方案
- 回滚以下文件可完整撤销本次变更：
  - `internal/services/file/transfer.go`
  - `frontend/src/stores/file.ts`
  - `frontend/src/pages/File.vue`
  - `frontend/src/components/file/OfferNodeTreePicker.vue`
  - `todo.md`
  - `docs/change/2026-03-06_win-file-offer-dir-node-picker.md`
