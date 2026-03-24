# 2026-03-23_monkey-bsd-reference-image-preview

## 变更背景 / 目标
- BSD 定制页面里点击已有上传图片时，实际走的是 `ReferenceImageCard` 本地预览弹窗，标题为 `图片预览`，只支持纵向滚动查看，不支持历史生成中已有的缩放查看能力。
- 之前的修正误改到了通用上传器链路，未命中真实入口；本次需要修正真实入口，并回退之前误加到通用上传器上的改动。

## 具体变更内容
- `ui/src/components/layout/workbench/view/customers/scenes/components/ReferenceImageCard.tsx`
  - 将已有图片点击后的本地 `Dialog + 图片预览` 替换为 `HistoryImageDetailDialog`。
  - 点击已有图片时构造最小图片 payload，直接复用历史生成的缩放查看器。
  - 保留 `base` 槽位的 `onOpenMask` 优先级，不影响遮罩流。
  - 保留 hover 预览、删除、编辑、上传、个人资产选择链路。
- `ui/src/components/ui/vines-uploader/files.tsx`
  - 回退 commit `f02d81018e135cb30cca939cb9f7bd9c5baffbe1` 在通用上传器上引入的 BSD 特殊预览逻辑。
  - 恢复通用图片预览职责，不再在这里打开历史查看器。
- `ui/src/components/layout/workbench/view/customers/scenes/components/HistoryImageDetailDialog.tsx`
  - 回退 commit `f02d81018e135cb30cca939cb9f7bd9c5baffbe1` 为 uploader 复用引入的 `hideDetailsPanel` / `showDetailsToggle` 兼容层。
  - 恢复为历史详情查看器原有接口和行为。

## Requirements impact
- none

## Specs impact
- none

## Related requirements
- none found

## Related specs
- none found

## 对应 plan.md 任务映射
- T1
  - `ReferenceImageCard.tsx` 替换真实 BSD 自定义页面入口的预览实现。
- T2
  - `files.tsx`、`HistoryImageDetailDialog.tsx` 回退误改的通用 uploader / shared viewer 逻辑。
- T3
  - 复用现有 `HistoryImageDetailDialog`，确保 `ReferenceImageCard` 传入的最小图片数据可用。
- T4
  - 完成定点格式化、`git diff --check`、定点 `eslint` 验证。

## 关键设计决策与权衡
- 直接改 `ReferenceImageCard`，不继续扩大通用上传器职责。
  - 原因：用户实际点击到的不是 `VinesUploader` 预览链路，继续在通用上传器上修只会维持错位。
- 复用现有 `HistoryImageDetailDialog`，但不保留上一次为 uploader 增加的轻量兼容层。
  - 原因：用户明确要求回退之前的误改 commit；本次改动优先保证真实入口生效和误改回退。
- 保持最小改动面。
  - 只提交 3 个代码文件，不带入工作流文档和无关 `.gitignore` 改动。

## 测试与验证方式 / 结果
- `git diff --check`
  - 通过
- `ui` 目录定点 `prettier --write`，仅处理修改文件
  - 通过
- `ui` 目录定点 `eslint`
  - 通过
  - 文件：
    - `src/components/layout/workbench/view/customers/scenes/components/ReferenceImageCard.tsx`
    - `src/components/layout/workbench/view/customers/scenes/components/HistoryImageDetailDialog.tsx`
    - `src/components/ui/vines-uploader/files.tsx`
- 未执行
  - 浏览器人工点选验证
  - 全量构建 / `yarn build`

## 潜在影响
- `ReferenceImageCard` 被多个 BSD 自定义场景复用，本次预览体验会统一切换到历史查看器。
- 由于已按用户要求回退 `f02d81018...`，通用上传器的 BSD 特化预览能力不再存在。
- 当前未做浏览器手工回归，仍建议在 BSD 页面上实际点选已有图片验证一次。

## 回滚方案
- 代码回滚：
  - 回退 commit `17bf559af fix: use history viewer for bsd reference image preview`
- 手工回滚目标：
  - `ReferenceImageCard.tsx` 恢复本地 `图片预览` Dialog
  - `files.tsx` 和 `HistoryImageDetailDialog.tsx` 重新应用 `f02d81018...` 的改动

## 子Agent执行轨迹
- 未使用子 Agent
