# 变更说明（Workflow）：Devices 节点信息弹窗（management/node_info）

## 变更背景 / 目标
Win 的 `Devices` 树形视图需要在点击节点时展示“设备基本信息”（平台、版本号等）。
你要求该信息必须来自**节点本身**：
- 远端节点：由目标节点本地采集并回包；
- Win 自身节点（sourceID==targetID）：由 Win 本地采集并返回（避免发包后 timeout）。

## 具体变更内容（跨仓汇总）
- `myflowhub-proto`（tag：`v0.1.1`）
  - 新增 wire：`management/node_info`（`ActionNodeInfo/Resp`、`NodeInfoReq/Resp{items: map[string]string}`）
- `myflowhub-subproto`（management module tag：`management/v0.1.1`）
  - 新增 handler：`node_info`，采集并返回 KV（`platform/go_version/module/version/commit/...`）
  - management forward 失败时可回 `node_info_resp(code!=1,msg=...)`，减少纯 timeout
- `myflowhub-server`（tag：`v0.0.2`）
  - 依赖升级到 `proto v0.1.1` 与 `subproto/management v0.1.1`
  - 新增集成测试：`integration_management_node_info_test.go`
- `myflowhub-win`
  - 新增 `ManagementService.NodeInfo/NodeInfoSimple`（自身节点短路）
  - `Devices` 节点行点击弹出 Modal 展示 KV，并提供 Reload/Close

## 关键设计决策与权衡（性能 / 扩展性）
- 返回 `items: map[string]string`：
  - 优点：字段可扩展，UI 不绑定固定 schema；未来增加 uptime/能力列表等无需破坏兼容。
  - 代价：弱类型；但本需求以“通用信息展示”为主，收益更大。
- 只做“点击时按需查询”：
  - 避免对树上每个节点预拉取，减少无效 I/O 与等待。

## 测试与验证方式 / 结果
- 已通过（本地）：
  - `myflowhub-proto`: `go test ./...`
  - `myflowhub-subproto/management`: `go test ./...`
  - `myflowhub-server`: `go test ./...`（包含集成测试）
  - `myflowhub-win`: `go test ./...`
- 建议冒烟：
  1) 启动 `hub_server`，Win 连接并登录；
  2) 打开 `Devices`，点击任意节点；
  3) 确认弹窗展示 `platform/go_version/...` 等字段；
  4) 点击 Win 自身节点不应出现 timeout。

## 潜在影响与回滚方案
- 影响：旧节点不支持 `node_info` 时仍可能超时；UI 已做提示，但需要逐步让节点实现该动作以完善体验。
- 回滚：
  - 已合并主线：revert 对应提交；若 tag/release 已发布，优先走补丁版本（避免删除远端 tag/release）。

