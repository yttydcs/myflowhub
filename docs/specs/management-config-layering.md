# Management Config Layering

## Scope

定义与管理子协议相关的两个跨仓稳定技术约束：

1. 节点显示名的管理协议回传约束
2. `hubruntime` 的持久化默认配置层与运行时覆盖语义

## Node Display Name Contract

- 配置键：`node.display_name`
- `list_nodes`
  - `nodes[]` 增加可选字段 `display_name`
- `node_info`
  - `items["display_name"]` 表示节点显示名
- Win `Devices` Config UI
  - 可在现有 `Edit` 弹窗中合成 `node.display_name` 的可编辑入口
  - 写路径仍使用标准 `config_get` / `config_set`
- UI 回退规则
  - 当 `display_name` 为空或缺失时，消费方必须回退到 `node_id`
- 直连子节点名称缓存
  - `list_nodes` 的 child `display_name` 以直连连接 metadata 为准
  - 实现应在“子节点身份建立”与“`config_set(node.display_name)` 成功返回”后尽量刷新该 metadata
  - 若 metadata 缺失，消费方继续回退到 `node_id`，不得改为 `list_nodes` 现查每个 child 的 `node_info`
- 本地节点一致性
  - 本地短路的 `node_info`、Config 标题和远程 `node_info` 都应基于同一键 `node.display_name`

## Persistent Config Layering

### Target

仅对实现了持久化能力的节点生效；本轮必须覆盖 `hubruntime`。

### File Layer

- `hubruntime` 的持久化默认层文件路径：`config/runtime_config.json`
- 路径相对 `WorkDir` 解析；若未设置 `WorkDir`，则相对当前进程工作目录

### Precedence

从低到高：

1. 持久化默认层 `config/runtime_config.json`
2. 环境变量
3. 命令行 flags / `Options`

### Read / Write Semantics

- `config_get` 返回当前生效值（effective value），不是单独的持久化层原值
- 支持持久化的节点：
  - `config_set` 写入持久化默认层
  - 写入成功后刷新运行时生效配置
- 不支持持久化的节点：
  - `config_set` 保持现有“仅运行期生效”语义

## Management Handler Capability

- `MyFlowHub-SubProto/management` 不直接承担文件存储职责
- `management config_set` 需要识别目标节点是否暴露“持久化写入”能力
- 若存在该能力，则优先调用持久化写入；否则回退到现有运行时 `Set`

## Compatibility

- 协议保持向后兼容：
  - 旧客户端可忽略新增的 `display_name`
  - 未实现持久化能力的节点继续按旧行为工作

## Verification Focus

- `display_name` 在 `list_nodes` / `node_info` 中回传一致
- `hubruntime` 启动后按 `persistent < env < flags` 计算 effective config
- `config_set(node.display_name)` 对 `hubruntime` 重启后仍保留
