# 2026-03-24 Auth authority 严格判定与 fail-closed

## 变更背景 / 目标

上一轮受控准入实现里，`authority` 选择仍保留了历史的 fail-open 回退语义：

- 显式 `authority.node_id` 不可达时，仍可能回退到 parent 或本地 authority
- 未显式配置 authority 但 parent 不可达时，仍可能把本地当作 authority

这会导致网络分区或 authority 失联场景下，节点错误地在本地继续处理 `register` 或依赖上游 authority 的 `login` 路径。

本次目标是把 authority 判定收紧为你确认的规则：

1. 显式配置 `authority.node_id` 时，只能使用该节点。
2. 未配置 `authority.node_id` 时，若已配置 parent，则直接父节点是唯一 authority。
3. 仅在既未配置 `authority.node_id` 也未配置 parent 时，本地才是 authority。
4. authority / parent 不可达时，auth 必须显式失败，不能回退本地处理。

Requirements impact: `updated`
Specs impact: `updated`
Related requirements: [auth-controlled-admission.md](../requirements/auth-controlled-admission.md)
Related specs: [auth.md](../../repo/MyFlowHub-Server/docs/specs/auth.md)

## 具体变更内容

### SubProto/auth
- `MyFlowHub-SubProto/auth/routing.go`
  - 将 authority 判定从“连接存在则上送，否则本地兜底”改为显式三态：
    - `local`
    - `remote`
    - `unavailable`
  - 新增 parent 配置识别逻辑，区分“未配置 parent”和“已配置但当前不可达”。
- `MyFlowHub-SubProto/auth/actions_register.go`
  - 普通 `register` 在 authority 不可达时返回 `code=4500,msg="authority unavailable"`。
  - 不再把 authority 不可达误判为本地 authority。
- `MyFlowHub-SubProto/auth/actions_login.go`
  - 对需要上游 authority 的路径（assist login / assist_query_credential）在 authority 不可达时显式失败。
  - 已有本地 binding 且可本地验签的登录路径保持不变。
- `MyFlowHub-SubProto/auth/perm_helpers.go`
  - 只有 authority 处于 `remote` 状态时才会上送权限刷新请求，不再依赖旧的 nil/非 nil 隐式语义。

### Tests
- `MyFlowHub-SubProto/auth/authority_test.go`
  - 新增回归测试覆盖：
    - 显式 authority 不可达时，register 不回退
    - parent 已配置但不可达时，register 不回退
    - 无 authority / 无 parent 配置时，本地可作为 authority
    - parent 已配置但不可达时，依赖上游 authority 的 login 显式失败
- `MyFlowHub-SubProto/auth/test_mocks_test.go`
  - 测试 server mock 支持注入 config，便于覆盖 authority / parent 配置分支

### Stable Docs
- `MyFlowHub-Server/docs/specs/auth.md`
  - 更新 authority 选择规则与 authority unavailable 的错误语义

## 对应 plan.md 任务映射

- `AUTH5` - authority strict selection and fail-closed
- `DOC1` - 更新 server auth spec
- `TEST1` - 运行 subproto/server auth 相关回归
- `REV1` - 3.3 Code Review
- `ARC1` - 本文归档与索引更新

## 关键设计决策与权衡

1. 采用显式三态 `local / remote / unavailable`
   - 优点：彻底消除“`nil` 同时表示本地 authority 和 authority 不可达”的歧义
   - 代价：需要同步修改 register/login/permission refresh 的分支判断

2. authority 不可达时返回显式错误，而不是 `rejected`
   - 优点：保留“这是临时不可达，不是业务拒绝”的语义
   - 代价：调用方需要把 `4500 authority unavailable` 视为重试类错误

3. 已有本地 binding 的登录路径保持不变
   - 优点：不额外破坏已经持有本地 credential 的已注册节点登录
   - 代价：authority 严格化主要约束“新准入”和“需上游 authority 的缺 credential 路径”，不是把全部登录都强制改成远端依赖

## 测试与验证方式 / 结果

- `go test github.com/yttydcs/myflowhub-subproto/auth/... -count=1 -p 1`
  - 结果：通过
- `go test github.com/yttydcs/myflowhub-server/tests -run "TestLoginHandler" -count=1 -p 1`
  - 结果：通过

## 潜在影响与回滚方案

### 潜在影响

- 配置了 `authority.node_id` 或 `parent.addr` 但上游连接不可达的节点，不再静默回退成本地 authority，而是显式失败。
- 这会让以前“网络分区时还能继续本地注册”的部署行为变为失败；这是本次安全收紧的预期结果。

### 回滚方案

- 回滚 `MyFlowHub-SubProto/auth/routing.go`、`actions_register.go`、`actions_login.go`、`perm_helpers.go` 与对应测试
- 回滚 `MyFlowHub-Server/docs/specs/auth.md`

## 子Agent执行轨迹

- 无。全部由主 agent 在当前 worktree 本地完成。
