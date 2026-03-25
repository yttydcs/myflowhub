# 2026-03-25 Auth 首个注册 Bootstrap

## 变更背景 / 目标
- 现有 auth 受控准入已经支持 `pending -> approve -> retry register` 与一次性 `permit`，但在 `local authority` 冷启动且开启审批时，缺少首个审批者 bootstrap。
- 本次目标是在不放宽普通审批语义的前提下，增加一个受控的一次性例外：
  - 仅对显式配置的 `device_id` 生效
  - 可选绑定 `pubkey`
  - 通过 `epoch` 一次性消费
  - 消费后恢复正常审批流

## 具体变更内容
### Control / Docs
- 更新 [auth-controlled-admission.md](../requirements/auth-controlled-admission.md)
  - 将 first-register bootstrap 定义为审批模型下的受控冷启动例外，而不是普通 `register` 放宽。

### MyFlowHub-Core
- `config/config.go`
  - 新增 bootstrap 配置键：
    - `auth.bootstrap.first_register.enabled`
    - `auth.bootstrap.first_register.role`
    - `auth.bootstrap.first_register.device_id`
    - `auth.bootstrap.first_register.pubkey`
    - `auth.bootstrap.first_register.epoch`
  - 为上述键补齐默认值。

### MyFlowHub-SubProto/auth
- 新增 `bootstrap_first_register.go`
  - 加载 bootstrap 配置
  - 校验非法配置：
    - `auth.disable_persist=true`
    - 存在 `parent` / `authority.node_id`
    - role 未定义
    - epoch 非正整数
    - 缺失 `device_id`
    - bootstrap `pubkey` 非法
  - 在 `register` 链路里接入 bootstrap 判定
- `actions_register.go`
  - 新顺序为：
    - existing rebind
    - permit
    - approved retry
    - first-register bootstrap
    - pending approval
    - open register fallback
- `node_keys.go` / `session.go`
  - 将 bootstrap 消费状态持久化到 `trusted_nodes.json.meta.first_register_bootstrap`
  - 重启时恢复 `consumed_epoch`
  - bootstrap 已消费但 binding 尚未存在时，也会抬高 `maxNode`，避免重用 `node_id`
- 测试
  - 新增 bootstrap 成功、消费后失效、pubkey mismatch、epoch 重开、非法配置单测
  - 扩展 admission persist/reload 测试，覆盖 bootstrap state 落盘

### MyFlowHub-Server
- 更新 [auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md)
  - 写清 bootstrap state 持久化位置、register 顺序、配置键与限制
- 评估 `hubruntime/cmd` 配置暴露后，决定本轮不新增 env/flag
  - 现有 `runtime_config.json` 已可承载新键
  - 本轮优先最小安全改动面，不扩展启动参数矩阵

## Requirements impact
- updated

## Specs impact
- updated

## Lessons impact
- none

## Related requirements
- [auth-controlled-admission.md](../requirements/auth-controlled-admission.md)

## Related specs
- [auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md)

## Related lessons
- none

## 对应 plan.md 任务映射
- `DOC-REQ-1`
- `CORE-1`
- `AUTH-1`
- `AUTH-2`
- `AUTH-3`
- `AUTH-4`
- `SRV-DOC-1`
- `SRV-OPT-1`
- `TEST-0`
- `TEST-1`
- `REV-1`
- `ARC-1`

## 经验 / 教训摘要
- 冷启动问题不应通过“普通首个 register 自动提权”解决，而应建模为显式 bootstrap exception。
- bootstrap 一次性消费状态如果不参与 `maxNode` 恢复，重启后可能把已预留的 `node_id` 重发出去。
- 默认角色是 `admin` 不等于默认可用；role 必须在当前 `auth.role_perms` 配置里真实存在，否则应 fail-fast。

## 可复用排查线索
- 症状：
  - 开启审批后 root/authority 没有首个审批者，所有新节点都只会 `pending`
  - 想配置 bootstrap 但 handler 初始化失败
- 触发条件：
  - `auth.register.require_approval=true`
  - `local authority` 冷启动
  - 开启了 bootstrap 但 role / persist / authority 配置不合法
- 关键词：
  - `auth bootstrap first register`
  - `requires local authority`
  - `requires persist enabled`
  - `unknown role`
  - `pubkey mismatch`
- 快速检查：
  - 看 `config/runtime_config.json` 中 bootstrap 配置是否完整
  - 看 `config/trusted_nodes.json.meta.first_register_bootstrap`
  - 看 `auth.role_perms` 是否定义了 bootstrap role
  - 看是否同时配置了 `parent.addr` 或 `authority.node_id`

## 关键设计决策与权衡
- 采用“bootstrap exception”而不是“自动批准第一个普通注册者”
  - 优点：不破坏审批模型的主语义
  - 代价：部署侧需要显式配置目标设备
- 采用 `epoch` 而不是单纯 `enabled` 作为重开开关
  - 优点：避免误重启/重载后重复开放
  - 代价：运维侧需要手工提升 epoch
- 本轮不加新的 env/flag
  - 优点：保持最小改动面，避免 runtime option / CLI / docs 三处同步膨胀
  - 代价：当前通过 `runtime_config.json` 配置，而不是命令行参数

## 测试与验证方式 / 结果
- `GOWORK=d:\project\MyFlowHub3\worktrees\feat-auth-bootstrap-first-register\control\go.work go test github.com/yttydcs/myflowhub-core/config -count=1`
  - 通过
- `GOWORK=d:\project\MyFlowHub3\worktrees\feat-auth-bootstrap-first-register\control\go.work go test github.com/yttydcs/myflowhub-subproto/auth/... -count=1 -p 1`
  - 通过
- `GOWORK=d:\project\MyFlowHub3\worktrees\feat-auth-bootstrap-first-register\control\go.work go test github.com/yttydcs/myflowhub-server/hubruntime/... -count=1 -p 1`
  - 通过
- `GOWORK=d:\project\MyFlowHub3\worktrees\feat-auth-bootstrap-first-register\control\go.work go test github.com/yttydcs/myflowhub-server/tests -run TestLoginHandler -count=1 -p 1`
  - 通过

## 潜在影响与回滚方案
- 潜在影响：
  - bootstrap 默认 role 为 `admin`，但若部署侧没有定义 `admin` 的权限集，auth 会显式初始化失败
  - 若只绑 `device_id` 不绑 `pubkey`，仍存在隔离网络以外的抢注风险
- 回滚方案：
  - 回滚 `MyFlowHub-Core/config/config.go`
  - 回滚 `MyFlowHub-SubProto/auth` bootstrap 相关文件与测试
  - 回滚 requirement/spec 文档更新
  - 删除 `trusted_nodes.json.meta.first_register_bootstrap` 使用

## 子Agent执行轨迹
- 无子 agent
