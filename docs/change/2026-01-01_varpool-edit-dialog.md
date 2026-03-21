# VarPool Edit Dialog Parity Fix

## 背景/目标
VarPool 的 Edit 按钮在新版 UI 中仅做表单填充，没有弹窗，且用户期望与旧版 Fyne 一致。目标是恢复弹窗编辑体验，并在保存后自动刷新变量内容。

## 具体变更内容
- 新增 VarPool 编辑弹窗（只读名称/Owner/类型，编辑 value 与 visibility）。
- “我的变量”和“监视变量”两处 Edit 按钮均打开弹窗。
- 保存后发送 set 并触发一次 get 自动刷新缓存值。

## 计划任务映射
- T7-1 VarPool Edit dialog parity fix

## 关键设计决策与权衡
- 采用页面内轻量弹窗实现，避免引入新的 UI 依赖，降低改动范围。
- 自动刷新以 get 方式确认最终值，略增加一次请求，但确保与服务端状态一致。

## 测试与验证
- 手动：点击任意变量 Edit，弹窗展示 name/owner/type，编辑 value/visibility 保存后 UI 更新。
- 手动：未连接或未登录时保存应提示错误且不提交。

## 潜在影响与回滚方案
- 影响：增加一次 get 请求用于刷新缓存值。
- 回滚：移除弹窗逻辑并恢复原先的表单填充操作。
