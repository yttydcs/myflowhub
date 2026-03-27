# Win Access Policy Role Dialog Refine

## 变更背景 / 目标

- 访问策略页上一轮已经完成角色弹窗化，但角色编辑弹窗仍存在 3 个明显问题：
  - 弹窗滚动区边距不足，输入框 focus ring 会被裁切
  - 角色编辑仍是左右分栏，阅读路径偏重
  - 角色权限仍可通过行内下拉直接改写，`Add Permission` 和 `Apply Built-in Preset` 的反馈不够明确
- 本轮目标是把角色列表和角色弹窗进一步收敛为更轻量、更明确的单列交互，并把权限新增改成显式选择流程。

## 具体变更内容

- `frontend/src/pages/AccessPolicy.vue`
  - 角色列表改成更接近 `Flow` 页面列表的一行式摘要布局，只保留角色名、基线标识、摘要信息和编辑/移除动作。
  - 角色编辑弹窗改成单列布局，主体宽度收敛为 `max-w-2xl`。
  - 角色弹窗滚动区增加内层 padding，避免 focus ring 被 overflow 裁切。
  - 角色权限列表改成只读行 + 删除按钮，不再通过行内 `<select>` 直接改写。
  - `Add Permission` 改成打开独立权限选择 overlay；用户从列表中选择权限后再显式追加。
  - `Apply Built-in Preset` 现在会覆盖当前 known perms、关闭权限选择弹窗，并给出成功反馈。
- `frontend/src/i18n/messages/operations.ts`
  - 补充权限选择弹窗、预设成功反馈和角色权限新说明文案。
- `frontend/src/pages/AccessPolicy.test.ts`
  - 更新角色列表摘要、角色弹窗滚动区、预设动作和权限 picker 流程测试。
- `docs/requirements/authority-admin-console.md`
- `docs/specs/authority-admin-console.md`
  - 补充角色弹窗单列化、选择式新增权限和只读权限行的长期约束。

## Requirements impact

- `updated`

## Specs impact

- `updated`

## Lessons impact

- `none`

## Related requirements

- `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-role-dialog-refine\docs\requirements\authority-admin-console.md`

## Related specs

- `D:\project\MyFlowHub3\worktrees\fix-win-access-policy-role-dialog-refine\docs\specs\authority-admin-console.md`
- `D:\project\MyFlowHub3\repo\MyFlowHub-Server\docs\specs\auth.md`

## Related lessons

- `D:\project\MyFlowHub3\docs\lessons\README.md`

## 对应 plan.md 任务映射

- `DOC-APR-1`
  - 更新 requirements/specs，明确角色弹窗的单列和选择式新增权限约束
- `IMPL-APR-1`
  - 收敛角色列表为单行摘要
- `IMPL-APR-2`
  - 修复角色弹窗 focus ring 裁切并改单列布局
- `IMPL-APR-3`
  - 改造角色权限为“选择后新增 + 行内删除”
- `TEST-APR-1`
  - 补齐角色弹窗新结构和权限 picker 测试

## 经验 / 教训摘要

- 只把弹窗设成可滚动并不能解决 focus ring 裁切，滚动容器内部仍需要预留额外 padding。
- 当用户强调“添加”语义时，行内 `<select>` 编辑虽然功能等价，但会显著削弱操作的可感知性；显式 picker 更符合预期。
- 列表类 UI 的紧凑感更多取决于信息密度和单行节奏，而不是简单减少文案。

## 可复用排查线索

- 症状：
  - 角色弹窗内输入框获得焦点时外圈被切掉
  - `Add Permission` 点击后用户感知不到新增
  - 权限列表看起来仍像大表单
- 触发条件：
  - 弹窗内容区使用 `overflow-y-auto` 但没有内层缓冲边距
  - 已选权限仍通过行内 `<select>` 直接修改
- 关键词：
  - `data-role-editor-scroll`
  - `rolePermissionPickerDialog`
  - `Built-in preset applied.`
  - `data-role-permission-picker-dialog`
- 快速检查：
  - 查看角色弹窗中是否还存在权限行内 `<select>`
  - 查看 `Add Permission` 是否打开独立 picker overlay
  - 查看滚动容器内部是否有额外 `px/py` padding

## 关键设计决策与权衡

- 采用嵌套 `Overlay` 做权限选择，而不是内联 `select` 或 popover：
  - 优点是选择路径更显式，和用户要求一致
  - 代价是多一层局部弹窗状态，需要确认 overlay stack 仍稳定
- 角色列表改成单行摘要，不再预览权限 chips：
  - 优点是扫读更快，更贴近 `Flow` 页面列表
  - 代价是列表页不再直接暴露每个权限 token，需要进弹窗查看详情
- 预设动作保留在角色弹窗内而不是移到列表页：
  - 这样能避免列表页操作过多，也能让预设作用于当前编辑上下文

## 测试与验证方式 / 结果

- `npm test -- AccessPolicy`
  - 通过
- `GOWORK=off wails generate module`
  - 通过
- `npm run build`
  - 通过
- `chrome-devtools` browser smoke check
  - Vite dev host 可访问，但 `/#/access-policy` 呈现空白根视图，未形成可信视觉验证结果

## 潜在影响与回滚方案

- 潜在影响：
  - 角色权限新增路径从行内修改切换为 picker，用户需要适应新的操作方式
  - 角色列表去掉权限 chips 后，列表页更简洁，但即时信息量会下降
- 回滚方案：
  - 回退 `frontend/src/pages/AccessPolicy.vue`、`frontend/src/pages/AccessPolicy.test.ts`、`frontend/src/i18n/messages/operations.ts`
  - 将 requirements/specs 恢复到上一版角色弹窗描述
  - 删除本次 change 归档并回退 `docs/change/README.md`

## 子Agent执行轨迹

- 未使用子Agent
