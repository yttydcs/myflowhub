# Management Node Display Name

## Goal

为设备管理场景提供“节点级全局可编辑名称”，避免 UI 直接暴露 `node_id` 造成理解成本。

## Scope

### Must

- `Devices` 树、节点详情等面向用户的 Win 设备管理界面优先显示节点显示名。
- 节点显示名是节点级全局名称，不依赖连接会话或局部上下文。
- 无显示名时，UI 必须回退到 `node_id`。
- 编辑入口放在 Win `Devices` 的现有 `Edit` 配置弹窗内。
- Win `Devices` 的现有 `Edit` 配置弹窗必须能在“首次命名”场景下直接编辑 `node.display_name`，不能要求该 key 先出现在 `config_list` 中。
- 对支持持久化的节点，显示名修改后重启仍保留。
- 对直连父节点，节点显示名修改成功后，当前会话中的后续管理查询应能看到更新后的名称，而不要求先断线重连。

### Optional

- 后续可扩展到更多客户端宿主或更多管理视图。
- 后续可补充“配置来源 / 被覆盖状态”提示。

### Out Of Scope

- 本轮不要求所有节点类型都立即支持持久化。
- 本轮不新增独立的昵称管理页面。

## User Scenarios

- 用户在 Win `Devices` 树中快速识别节点，而不是只看到 `Node 123`。
- 用户在节点详情中确认当前节点的显示名与基础信息。
- 用户通过统一的 `config_set`/`Edit` 配置入口修改显示名，而不是引入单独专用 action。
- 用户第一次给节点命名时，可以直接在现有 `Edit` 配置弹窗中看到并编辑 `node.display_name`。

## Acceptance

- 当节点存在显示名时，Win `Devices` 树和节点详情优先显示该名称。
- 当节点不存在显示名时，Win UI 仍正确显示 `node_id`，不出现空标题。
- 用户无需预先创建配置键，即可在现有 `Edit` 配置弹窗中首次设置 `node.display_name`。
- `list_nodes` 与 `node_info` 都能提供显示名信息，满足不同调用场景。
- Win 自身节点的本地 `node_info` 与 Config 标题对 `display_name` 的解析语义保持一致。
- 对直连父节点，`config_set(node.display_name)` 成功后，当前会话中的后续 `list_nodes` 查询可看到更新值，无需断线重连。
- 对 `hubruntime` 节点，显示名通过远程配置修改后，重启后仍然有效。

## Notes

- 本需求是跨仓稳定需求，影响 `MyFlowHub-Proto`、`MyFlowHub-SubProto`、`MyFlowHub-Server` 与 `MyFlowHub-Win`。
