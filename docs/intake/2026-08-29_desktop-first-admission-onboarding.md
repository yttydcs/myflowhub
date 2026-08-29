# Desktop 首次准入引导

## 来源

用户在启动生产版 MyFlowHub Desktop 后询问首次登录表单应填写什么。现场检查确认本机没有已保存 Profile、Hub 状态或监听中的 Hub；进一步对照实现发现，父 Hub 签发一次性 Permit 必须绑定 Desktop 公钥，但当前登录页只能直接尝试登录，不能先生成并展示受保护的 Desktop 身份，因此全新环境无法仅靠界面闭环首次准入。

## 目标

- 在登录页允许先验证并保存 Profile 配置、生成或复用该 Profile 的持久 Ed25519 身份；
- 展示可复制的本机 raw-base64 Ed25519 公钥，供父 Hub 签发绑定 Node ID 与身份的一次性 Permit；
- 准备身份时不把 Profile 标记为已登录或 active，不启动网络连接；
- Permit 仍只用于一次登录调用，不写入 settings、日志或 UI preference；
- 使用真实本地 Hub 完成“准备身份 → 签发 Permit → 启动 Hub → 登录”的生产 Wails 验证。

## 非目标

- 不改变 MFH4 身份、准入 Permit、角色或 default-deny policy 协议；
- 不在 Desktop 内加入离线 Hub 管理、Permit 签发或权限放宽能力；
- 不持久化 Permit，不显示或导出 Desktop 私钥；
- 不处理项目品牌图标；
- 不发布或推送。

## 推荐方向

新增窄化的 `PrepareProfileJSON` Desktop host API：严格校验 Profile，在 Profile credential store 中生成或加载身份，返回 Profile 与公开身份，并把 Profile 保存为可选但非 active 记录。React 登录页以明确的两步引导调用该 API、展示/复制公钥，再使用既有 `LoginJSON` 发送一次性 Permit。成功连接仍是激活 Profile 的唯一首次登录路径。
