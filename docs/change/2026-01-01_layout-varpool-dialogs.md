# Layout UX & VarPool Add Dialogs

## 背景/目标
优化主界面布局与交互体验：移除非 Home 的“Current Module”卡片、分离侧边栏与内容滚动、调整顶部连接状态样式、统一 Profile 下拉风格，并将 VarPool 的新增操作移至弹窗。

## 具体变更内容
- AppShell：
  - 非 Home 页面隐藏“Current Module”卡片。
  - 侧边栏与主内容区域滚动分离，Header 固定不随内容滚动。
  - 顶部连接状态改为灰色圆角矩形，图标红/绿，文案显示 “Connected to {addr}” 或 “Disconnected / Last error”。
  - Profile 选择器替换为自定义弹层菜单。
- VarPool：
  - 移除页面内新增表单。
  - “我的变量”添加弹窗新增入口。
  - “Watched Variables”添加弹窗新增监视入口。
  - 新增保存后自动 get 刷新列表。

## 计划任务映射
- T5-1 Layout UX adjustments
- T7-2 VarPool add dialogs

## 关键设计决策与权衡
- 通过布局级滚动容器分离滚动区域，避免全页滚动导致的导航与内容混用。
- 自定义 Profile 菜单，保持视觉一致性并减少默认 select 样式割裂感。
- VarPool 新增通过弹窗完成，减少主界面表单占用空间；保存后追加一次 get，保证一致性。

## 测试与验证
- 手动：检查 Header 固定、侧边栏滚动与内容滚动分离。
- 手动：连接/断开时顶部状态文案与样式正确。
- 手动：Profile 菜单风格与交互正常。
- 手动：VarPool 新增弹窗提交后列表更新。

## 潜在影响与回滚方案
- 影响：VarPool 新增后触发一次额外 get。
- 回滚：恢复原布局与内嵌新增表单即可。
