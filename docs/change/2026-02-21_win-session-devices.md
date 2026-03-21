# 2026-02-21 - Win：Session 下新增 Devices（设备查询）页面

## 变更背景 / 目标
Win 客户端在联调阶段需要一个更直观、低成本的方式来查询当前会话下的“节点/设备列表”，用于验证连接拓扑与管理面行为。

本次变更在左侧导航的 **Session** 分组下新增 **Devices** 页面，复用现有 management 协议动作进行查询，不修改 wire/协议。

> 后续演进：同日已将 Devices 升级为“树形懒加载展示”（Mode 下拉 + 节点展开查询），见 `docs/change/2026-02-21_win-devices-tree.md`。本文档记录的是 Devices 初版（平铺列表）。

## 具体变更内容

### 新增
- `MyFlowHub-Win/frontend/src/pages/Devices.vue`
  - 新增 Devices 页面：支持 `List Direct`（`list_nodes`）与 `List Subtree`（`list_subtree`）两种查询。
  - 明确提示：Subtree **非递归**（当前实现仅“直连节点 + 自身”）。
- `MyFlowHub-Win/frontend/src/stores/devices.ts`
  - 新增独立 store：负责输入校验、调用 Wails binding、列表整理与错误信息。

### 修改
- `MyFlowHub-Win/frontend/src/router/index.ts`
  - 新增路由：`/devices`
- `MyFlowHub-Win/frontend/src/layout/AppShell.vue`
  - 左侧导航分组标题 `Console` 更名为 `Session`
  - 在 Session 分组下新增导航入口 `Devices`

## 与计划任务映射（plan.md）
- DEV1：完成（路由与导航入口）
- DEV2：完成（Devices store）
- DEV3：完成（Devices 页面）
- DEV4：完成（验证：go test + wails build；手工冒烟步骤见下）
- DEV5：完成（Code Review 结论：通过）
- DEV6：完成（本归档文档）

## 关键设计决策与权衡
1) **不改 wire/协议，不新增后端接口**
   - 复用 Wails binding：`ManagementService.ListNodesSimple` / `ListSubtreeSimple`
   - 优点：改动小、落地快、风险低；满足联调工具定位
2) **Devices 独立 store，不复用 Management store**
   - 避免 Management 页面的状态（target/config/selected 等）与 Devices 查询互相污染
   - 为后续扩展（如递归子树、过滤、分页）保留边界
3) **Subtree 的语义对齐现状实现**
   - 当前 `list_subtree` 并非递归全量子树，UI 文案必须避免误导

## 测试与验证方式 / 结果

### 命令验证（Windows）
- `GOWORK=off go test ./...`：通过（仓库无单测文件）
- `GOWORK=off wails build -nopackage`：通过，成功生成 `build/bin/myflowhub-win.exe`

### 推送状态（GitHub）
已合并并 push 至 GitHub：`repo/MyFlowHub-Win` 的 `main`（包含该变更的提交：`b497c40`）。

### 手工冒烟（建议步骤）
1) 启动 server + win（可用现有启动脚本或手动启动）
2) Home：Connect → Register/Login（拿到 nodeId/hubId）
3) Session → Devices：
   - `List Direct`：应返回直连节点列表
   - `List Subtree`：应返回“直连节点 + 自身”（非递归）

## 潜在影响与回滚方案
- 影响：左侧导航首个分组标题从 `Console` 调整为 `Session`；新增 `/devices` 路由与页面。
- 回滚：
  - 直接 revert 本分支的 2 个提交即可恢复：
    - `feat(win): 新增 Devices 查询页面与 store`
    - `feat(win): Session 增加 Devices 入口与路由`
