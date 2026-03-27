# Win Access Policy Dialog Editors

## Background

- 访问策略页虽然已经改名为“访问策略”，但默认准入和角色权限仍然以整页展开表单呈现，页面信息密度过高。
- 用户希望把角色改成紧凑列表，右侧提供编辑动作，并在弹窗里完成详细权限维护。
- 默认准入也需要采用同样的“摘要 + 弹窗 + 权限列表增删”模式。

## Goal

- 收敛访问策略页主视图，只在页面上保留摘要和列表。
- 把默认准入与角色详情编辑统一迁移到 Overlay 弹窗。
- 用权限目录驱动的列表式 add / remove 交互替代整块权限矩阵。

## Changes

- `frontend/src/pages/AccessPolicy.vue`
  - 当前策略页改成轻量摘要布局。
  - 默认准入改成摘要卡 + `Edit Default Access` 弹窗。
  - 角色管理改成紧凑列表，每个角色只保留摘要、编辑、移除。
  - 节点覆盖改成紧凑列表，每个覆盖只保留摘要、编辑、移除。
  - 新增角色弹窗与默认准入弹窗，权限编辑统一为“列表 + 添加 / 移除”。
  - 新增节点覆盖弹窗，创建和编辑都不再直接占用页面空间。
  - 右侧保存、运行时和节点查询收敛为单一操作面板，运行时详情默认折叠。
  - 删除仍被默认准入或节点覆盖引用的角色时显式阻止。
  - 编辑角色名时同步更新当前页面中的默认角色与节点覆盖引用。
- `frontend/src/stores/accessPolicyCatalog.ts`
  - 新增扁平权限 option 导出和按权限查询元数据 helper，供弹窗下拉和描述展示复用。
- `frontend/src/i18n/messages/operations.ts`
  - 补充访问策略轻量布局、弹窗、权限列表和引用提示相关文案。
- `frontend/src/pages/AccessPolicy.test.ts`
  - 更新为默认准入弹窗、节点覆盖弹窗、操作面板和角色编辑弹窗的关键路径测试。
- `frontend/src/stores/accessPolicyCatalog.test.ts`
  - 补充 flattened option / metadata helper 测试。
- `docs/requirements/authority-admin-console.md`
- `docs/specs/authority-admin-console.md`
  - 将“摘要列表 + 弹窗编辑 + 权限列表增删”的交互边界写入稳定文档。

## Validation

- `npm test -- AccessPolicy accessPolicyCatalog authority_admin`
  - 通过
- `wails generate module`
  - 执行，用于 fresh worktree 生成 `frontend/wailsjs/**`
- `npm run build`
  - 通过

## Rollback

- 回退 `AccessPolicy.vue`、`accessPolicyCatalog.ts`、对应测试与 i18n 文案。
- 删除本次 change 归档，并把 requirements/specs 恢复到上一版交互说明。
