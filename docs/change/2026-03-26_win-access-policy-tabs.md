# 2026-03-26_win-access-policy-tabs

## 变更背景 / 目标
- 将中文“权限编排”统一改名为“访问策略”，避免 Authority 页面在中文环境下继续使用含义偏弱的旧命名。
- 把 `Access Policy` 从单页长表单改为 tab 化访问策略控制台：首页保留当前策略编辑与运行时校验，新增“角色管理”页专门维护角色权限。
- 提升权限编辑易用性：默认权限和角色权限改为目录勾选，不再要求用户手动输入 CSV。

## 具体变更内容
- 稳定文档：
  - 更新 `docs/requirements/authority-admin-console.md`
  - 更新 `docs/specs/authority-admin-console.md`
- 前端页面：
  - 重写 `frontend/src/pages/AccessPolicy.vue`
  - 引入 `PageHero` tab 结构：`Current Policy` / `Role Management`
  - “当前策略”页保留 authority 自动解析、默认角色、默认权限、节点覆盖、保存面板、运行时快照和节点权限查询
  - “角色管理”页抽离角色权限编辑，支持新增/删除角色、内置角色预设和未知权限保留展示
- 前端模型：
  - 新增 `frontend/src/stores/accessPolicyCatalog.ts`
  - 维护内置权限目录、内置角色预设、known/unknown 权限拆分与回写
  - 对 `*` 通配权限做独占归一化
- 文案与入口：
  - 更新 `frontend/src/i18n/messages/operations.ts`
  - 更新 `frontend/src/i18n/messages/shell.ts`
  - 更新 `frontend/src/layout/AppShell.vue`
  - 更新 `frontend/src/router/index.ts`
- 测试：
  - 新增 `frontend/src/pages/AccessPolicy.test.ts`
  - 新增 `frontend/src/stores/accessPolicyCatalog.test.ts`

## Requirements impact
- updated

## Specs impact
- updated

## Lessons impact
- none

## Related requirements
- `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-tabs\docs\requirements\authority-admin-console.md`

## Related specs
- `D:\project\MyFlowHub3\worktrees\refactor-win-access-policy-tabs\docs\specs\authority-admin-console.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\file.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\flow.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\exec.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\varstore.md`

## Related lessons
- none

## 对应 plan.md 任务映射
- `DOCS-AP-1`
  - authority admin console 的 requirements/specs 已同步到“访问策略 + tab + 目录化权限编辑”语义
- `IMPL-AP-1`
  - 中文命名改为“访问策略”，`Access Policy` 页面改为 tab 结构，导航和 route 副标题同步更新
- `IMPL-AP-2`
  - 新增权限目录 helper，默认权限与角色权限改为目录勾选，未知权限保存时保留
- `TEST-AP-1`
  - 新增页面/目录 helper 测试，完成前端测试、Wails module 生成和前端构建验证

## 经验 / 教训摘要
- Access Policy 的真实问题不是字段不全，而是把“当前策略”和“角色编辑”堆在一个长页面里，用户缺少清晰的任务切面。
- 权限 catalog 化时必须保留 unknown perms，否则 UI 一旦落后于服务端权限点就会把旧配置误删。
- `*` 通配权限在编辑态里应视为独占选择，否则“全量权限 + 若干普通权限”的组合只会制造歧义。

## 可复用排查线索
- 症状：
  - `npm run build` 报 `Could not resolve "../../wailsjs/go/main/App"`
- 触发条件：
  - 新 worktree 中还没有生成 Wails 前端 bindings
- 关键词：
  - `wailsjs/go/main/App`
  - `wails generate module`
  - `GOWORK=off`
- 快速检查：
  - 先确认 `frontend/wailsjs/go/main/App.js` 是否存在
  - 若不存在，执行 `$env:GOWORK='off'; wails generate module`
  - 再重跑 `npm run build`

## 关键设计决策与权衡
- 采用“前端静态权限目录 + unknown perms 保留”而不是新增后端权限元数据接口
  - 好处：不扩大协议和 Wails 改动面
  - 代价：后续新增权限点时需要同步更新 catalog
- 采用“当前策略 / 角色管理”双 tab，而不是继续在单页内堆折叠块
  - 好处：按任务切面分离，默认准入与角色权限不再互相抢视线
  - 代价：页面内部状态组织更复杂
- 允许内置角色预设，但不强制所有角色只能来自预设
  - 好处：保留现有灵活性，同时给 `superadmin/admin/node` 更快的起步体验
  - 代价：仍需处理自定义角色和目录外权限

## 测试与验证方式 / 结果
- 前端测试：
  - `npm test`
  - 结果：通过，`9` 个测试文件、`28` 个测试全部通过
- Wails module 生成：
  - `$env:GOWORK='off'; wails generate module`
  - 结果：通过
  - 备注：输出仍有 `Not found: time.Time` 警告，但退出码为 `0`
- 前端构建：
  - `npm run build`
  - 结果：通过
  - 备注：仍有 Vite chunk size warning，但不影响构建成功

## 潜在影响
- 中文用户看到的 Authority 入口和页面主标题会从“权限编排”变成“访问策略”。
- `Access Policy` 页面交互发生明显变化，旧的“单页滚动编辑”习惯会切换到 tab 化工作流。
- 当服务端未来新增 catalog 外权限时，页面会把它们标记为额外权限并保留，而不是直接可勾选编辑。

## 回滚方案
- 回退 `frontend/src/pages/AccessPolicy.vue` 到旧的单页版本。
- 删除 `frontend/src/stores/accessPolicyCatalog.ts` 及对应测试。
- 回退 `frontend/src/i18n/messages/operations.ts`、`frontend/src/i18n/messages/shell.ts`、`frontend/src/layout/AppShell.vue`、`frontend/src/router/index.ts`。
- 回退 `docs/requirements/authority-admin-console.md` 和 `docs/specs/authority-admin-console.md` 的本轮更新。

## 子Agent执行轨迹
- none
