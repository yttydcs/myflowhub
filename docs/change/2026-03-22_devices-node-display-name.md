# 2026-03-22 Devices Node Display Name

## 变更背景 / 目标

Win `Devices` 页面此前直接以 `Node {nodeId}` 作为主标题，用户难以区分节点。目标是：

- 设备树、节点详情、现有 Config/Edit 弹窗优先显示节点昵称
- 继续复用现有 `Edit` 配置弹窗，不新增专用 UI
- 无昵称时保持 `nodeId` 回退

## 具体变更内容

- `frontend/src/stores/devices.ts`
  - 解析 `list_nodes` / `list_subtree` 的 `display_name` / `displayName`
  - 在 `DeviceTreeNode` 上保存 `displayName`
  - 增加 `getDisplayName(nodeId)` 供页面回退查找
- `frontend/src/pages/Devices.vue`
  - 设备树主标题优先显示 `display_name`
  - `node_info` 弹窗标题优先显示 `display_name`
  - 现有 Config/Edit 弹窗标题优先显示 `display_name`
  - 当存在昵称时，次级文案仍保留 `Node {nodeId}` 便于识别

## plan.md 任务映射

- `WIN1 - Show Display Name In Devices And Reuse Existing Edit Entry`
- `INT1 - Cross Repo Integration And Verification`（Win 验证部分）

## 关键设计决策与权衡

- 不新增新的“昵称编辑”专用弹窗，直接复用现有 `Edit` 配置入口，减少 UI 分叉
- 有昵称时保留 `Node {nodeId}` 作为次级信息，兼顾可读性与可追踪性
- 回退顺序固定为 `display_name -> node_id`

## 需求 / 规范影响检查

- 控制面 requirement 已记录在 `D:\project\MyFlowHub3\docs\requirements\management-node-display-name.md`
- 控制面 spec 已记录在 `D:\project\MyFlowHub3\docs\specs\management-config-layering.md`
- 本仓 repo-local `requirements/specs` 无新增长期真相；变更已由控制面 requirement/spec 承载
- lessons 无新增
- 需要更新 `docs/change/README.md` 索引

## 测试与验证方式 / 结果

Go 侧验证：

```powershell
$env:GOWORK='off'
go test ./... -count=1
```

结果：通过。

前端构建验证：

```powershell
npm run build
```

结果：失败，阻塞于仓内既有依赖缺失：

- `frontend/src/pages/Home.vue` 依赖 `../../wailsjs/go/session/SessionService`
- 当前 worktree 缺少该生成产物
- 本次 `Devices` 改动未修改该文件或其依赖链路

## 潜在影响与回滚方案

### 潜在影响

- 设备树要稳定展示所有 child 昵称，仍依赖后端 `list_*` 实际返回 `display_name`
- 当前 `npm run build` 阻塞点与本次改动无关，但会影响前端整仓发布验证

### 回滚

- 回退 `frontend/src/stores/devices.ts` 与 `frontend/src/pages/Devices.vue`

## 子 Agent 执行轨迹

- `WIN1` -> `Hegel (019d15a7-ca9e-7e01-bcc6-41ea01b65262)` -> `D:\project\MyFlowHub3\worktrees\MyFlowHub-Win-feat-management-node-display-name`
  - 文件：`frontend/src/stores/devices.ts`、`frontend/src/pages/Devices.vue`
  - 验收：Win Go 测试通过；前端构建失败点已确认为仓内既有 `wailsjs` 生成物缺失
